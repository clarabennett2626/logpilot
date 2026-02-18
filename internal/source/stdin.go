package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	// DefaultBufferSize is the default capacity for the lines channel.
	DefaultBufferSize = 1000

	// DropOldest discards the oldest unread line when the buffer is full.
	DropOldest BackpressureStrategy = iota
	// Block waits until a reader consumes a line before accepting more.
	Block
)

// BackpressureStrategy controls behaviour when the lines channel is full.
type BackpressureStrategy int

// StdinOption configures a StdinSource.
type StdinOption func(*StdinSource)

// WithBufferSize sets the capacity of the lines channel.
func WithBufferSize(n int) StdinOption {
	return func(s *StdinSource) { s.bufSize = n }
}

// WithBackpressure sets the backpressure strategy.
func WithBackpressure(bp BackpressureStrategy) StdinOption {
	return func(s *StdinSource) { s.backpressure = bp }
}

// WithReader overrides the default stdin reader (useful for testing).
func WithReader(r io.Reader) StdinOption {
	return func(s *StdinSource) { s.reader = r }
}

// StdinSource reads log lines from standard input. It is designed to work
// with piped input such as:
//
//	kubectl logs -f pod | logpilot
//	cat app.log | logpilot
//	docker logs -f container | logpilot
type StdinSource struct {
	reader       io.Reader
	lines        chan LogEntry
	errs         chan error
	bufSize      int
	backpressure BackpressureStrategy
	cancel       context.CancelFunc
	once         sync.Once
	done         chan struct{}
}

// NewStdinSource creates a new StdinSource with the given options.
func NewStdinSource(opts ...StdinOption) *StdinSource {
	s := &StdinSource{
		reader:       os.Stdin,
		bufSize:      DefaultBufferSize,
		backpressure: Block,
		done:         make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	s.lines = make(chan LogEntry, s.bufSize)
	s.errs = make(chan error, 1)
	return s
}

// IsPipe reports whether stdin appears to be a pipe (not a terminal).
func IsPipe() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// Lines returns the channel of log entries.
func (s *StdinSource) Lines() <-chan LogEntry { return s.lines }

// Errors returns the channel of errors.
func (s *StdinSource) Errors() <-chan error { return s.errs }

// Start reads lines from stdin until ctx is cancelled or EOF is reached.
func (s *StdinSource) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)
	defer close(s.lines)
	defer close(s.errs)
	defer close(s.done)

	scanner := bufio.NewScanner(s.reader)
	// Support very long log lines (up to 1 MB).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		entry := LogEntry{
			Line:   scanner.Text(),
			Source: "stdin",
		}
		if !s.emit(ctx, entry) {
			return ctx.Err()
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case s.errs <- fmt.Errorf("stdin read error: %w", err):
		default:
		}
		return err
	}
	return nil
}

// emit sends an entry to the lines channel, respecting backpressure strategy.
func (s *StdinSource) emit(ctx context.Context, entry LogEntry) bool {
	switch s.backpressure {
	case DropOldest:
		select {
		case s.lines <- entry:
		default:
			// Channel full — drop oldest.
			select {
			case <-s.lines:
			default:
			}
			select {
			case s.lines <- entry:
			case <-ctx.Done():
				return false
			}
		}
	default: // Block
		select {
		case s.lines <- entry:
		case <-ctx.Done():
			return false
		}
	}
	return true
}

// Stop cancels reading and waits for the reader goroutine to finish.
func (s *StdinSource) Stop() error {
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
	<-s.done
	return nil
}
