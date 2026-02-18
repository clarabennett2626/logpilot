package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileConfig holds configuration for a file source.
type FileConfig struct {
	// Patterns is a list of file paths or glob patterns.
	Patterns []string
	// TailLines is the number of lines to read from the end on startup.
	// If 0, read from the beginning. If negative, read from the beginning.
	TailLines int
}

// FileSource reads log lines from one or more files with live tailing
// and log rotation support.
type FileSource struct {
	config  FileConfig
	lines   chan LogEntry
	errs    chan error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	stopped chan struct{}
}

// NewFileSource creates a new file source from the given config.
func NewFileSource(cfg FileConfig) *FileSource {
	return &FileSource{
		config:  cfg,
		lines:   make(chan LogEntry, 256),
		errs:    make(chan error, 32),
		stopped: make(chan struct{}),
	}
}

func (fs *FileSource) Lines() <-chan LogEntry { return fs.lines }
func (fs *FileSource) Errors() <-chan error   { return fs.errs }

// Start resolves glob patterns and begins tailing all matched files.
func (fs *FileSource) Start(ctx context.Context) error {
	ctx, fs.cancel = context.WithCancel(ctx)

	paths, err := fs.resolvePatterns()
	if err != nil {
		return fmt.Errorf("resolving file patterns: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no files matched patterns: %v", fs.config.Patterns)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}

	// Watch directories containing the files (for rotation detection).
	dirs := map[string]struct{}{}
	for _, p := range paths {
		d := filepath.Dir(p)
		dirs[d] = struct{}{}
	}
	for d := range dirs {
		if err := watcher.Add(d); err != nil {
			fs.sendError(fmt.Errorf("watching directory %s: %w", d, err))
		}
	}

	// Start a tailer goroutine per file.
	for _, p := range paths {
		fs.wg.Add(1)
		go fs.tailFile(ctx, watcher, p)
	}

	// Wait for all tailers then clean up.
	go func() {
		fs.wg.Wait()
		watcher.Close()
		close(fs.lines)
		close(fs.errs)
		close(fs.stopped)
	}()

	return nil
}

// Stop cancels tailing and waits for goroutines to finish.
func (fs *FileSource) Stop() error {
	if fs.cancel != nil {
		fs.cancel()
	}
	<-fs.stopped
	return nil
}

// resolvePatterns expands glob patterns into unique absolute file paths.
func (fs *FileSource) resolvePatterns() ([]string, error) {
	seen := map[string]struct{}{}
	var result []string

	for _, pattern := range fs.config.Patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Treat as literal path.
			abs, err := filepath.Abs(pattern)
			if err != nil {
				return nil, err
			}
			if _, err := os.Stat(abs); err != nil {
				return nil, fmt.Errorf("file not found: %s", abs)
			}
			if _, ok := seen[abs]; !ok {
				seen[abs] = struct{}{}
				result = append(result, abs)
			}
			continue
		}
		for _, m := range matches {
			abs, err := filepath.Abs(m)
			if err != nil {
				return nil, err
			}
			info, err := os.Stat(abs)
			if err != nil || info.IsDir() {
				continue
			}
			if _, ok := seen[abs]; !ok {
				seen[abs] = struct{}{}
				result = append(result, abs)
			}
		}
	}
	return result, nil
}

// tailFile reads initial lines then tails a single file, handling rotation.
func (fs *FileSource) tailFile(ctx context.Context, watcher *fsnotify.Watcher, path string) {
	defer fs.wg.Done()

	f, err := os.Open(path)
	if err != nil {
		fs.sendError(fmt.Errorf("opening %s: %w", path, err))
		return
	}
	defer f.Close()

	// Read initial lines.
	if fs.config.TailLines > 0 {
		if err := fs.seekToLastN(f, fs.config.TailLines); err != nil {
			fs.sendError(fmt.Errorf("seeking in %s: %w", path, err))
		}
	}

	offset, err := fs.readLines(f, path)
	if err != nil {
		fs.sendError(fmt.Errorf("initial read of %s: %w", path, err))
		return
	}

	// Record inode for rotation detection.
	lastStat, _ := f.Stat()
	lastSize := offset

	// Poll ticker as fallback for missed events.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			abs, _ := filepath.Abs(event.Name)
			if abs != path {
				continue
			}

			if event.Has(fsnotify.Write) {
				offset, lastSize, err = fs.handleWrite(f, path, offset, lastSize)
				if err != nil {
					fs.sendError(err)
				}
			}

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
				// File was rotated — reopen.
				newF, newOffset, reopened := fs.tryReopen(path, lastStat)
				if reopened {
					f.Close()
					f = newF
					offset = newOffset
					lastStat, _ = f.Stat()
					lastSize = newOffset
				}
			}

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}

		case <-ticker.C:
			// Check for truncation or new data.
			stat, err := os.Stat(path)
			if err != nil {
				// File gone — try to reopen (rotation).
				newF, newOffset, reopened := fs.tryReopen(path, lastStat)
				if reopened {
					f.Close()
					f = newF
					offset = newOffset
					lastStat, _ = f.Stat()
					lastSize = newOffset
				}
				continue
			}

			if stat.Size() < lastSize {
				// Truncated — reread from start.
				f.Close()
				f2, err := os.Open(path)
				if err != nil {
					fs.sendError(fmt.Errorf("reopening truncated %s: %w", path, err))
					continue
				}
				f = f2
				offset = 0
				lastStat, _ = f.Stat()
			}

			newOff, err := fs.readLines(f, path)
			if err != nil {
				fs.sendError(err)
				continue
			}
			if newOff > 0 {
				offset = newOff
			}
			lastSize = offset
		}
	}
}

// handleWrite reads new data after a write event, handling truncation.
func (fs *FileSource) handleWrite(f *os.File, path string, offset, lastSize int64) (int64, int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return offset, lastSize, fmt.Errorf("stat %s: %w", path, err)
	}
	if stat.Size() < lastSize {
		// Truncated.
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return 0, 0, fmt.Errorf("seek after truncation %s: %w", path, err)
		}
		offset = 0
	}
	newOff, err := fs.readLines(f, path)
	if err != nil {
		return offset, lastSize, err
	}
	if newOff > 0 {
		offset = newOff
	}
	return offset, stat.Size(), nil
}

// tryReopen attempts to reopen a file after rotation. Returns the new file,
// offset after initial read, and whether reopening succeeded.
func (fs *FileSource) tryReopen(path string, lastStat os.FileInfo) (*os.File, int64, bool) {
	// Wait briefly for the new file to appear.
	for i := 0; i < 5; i++ {
		f, err := os.Open(path)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		// Check if it's actually a new file (different inode or smaller).
		newStat, _ := f.Stat()
		if lastStat != nil && os.SameFile(lastStat, newStat) {
			f.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}
		off, _ := fs.readLines(f, path)
		return f, off, true
	}
	return nil, 0, false
}

// readLines reads available lines from the current position, sends them,
// and returns the new offset.
func (fs *FileSource) readLines(f *os.File, path string) (int64, error) {
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fs.lines <- LogEntry{
			Line:   scanner.Text(),
			Source: path,
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}
	off, _ := f.Seek(0, io.SeekCurrent)
	return off, nil
}

// seekToLastN positions the file to read approximately the last n lines.
// It works by scanning backwards from the end.
func (fs *FileSource) seekToLastN(f *os.File, n int) error {
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()
	if size == 0 {
		return nil
	}

	// Read chunks from the end to find newlines.
	const chunkSize = 8192
	newlines := 0
	offset := size

	for offset > 0 && newlines <= n {
		readSize := int64(chunkSize)
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize

		buf := make([]byte, readSize)
		if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
			return err
		}
		for i := len(buf) - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				newlines++
				if newlines > n {
					offset += int64(i) + 1
					break
				}
			}
		}
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	return nil
}

func (fs *FileSource) sendError(err error) {
	select {
	case fs.errs <- err:
	default:
	}
}
