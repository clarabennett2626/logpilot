package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func collectLines(t *testing.T, src *FileSource, timeout time.Duration, n int) []LogEntry {
	t.Helper()
	var entries []LogEntry
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for len(entries) < n {
		select {
		case e, ok := <-src.Lines():
			if !ok {
				return entries
			}
			entries = append(entries, e)
		case <-timer.C:
			t.Fatalf("timeout waiting for lines: got %d, want %d", len(entries), n)
		}
	}
	return entries
}

func TestFileSource_ReadFromStart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	entries := collectLines(t, src, 2*time.Second, 3)
	if entries[0].Line != "line1" || entries[1].Line != "line2" || entries[2].Line != "line3" {
		t.Errorf("unexpected lines: %v", entries)
	}

	cancel()
	src.Stop()
}

func TestFileSource_TailLastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("a\nb\nc\nd\ne\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}, TailLines: 2})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	entries := collectLines(t, src, 2*time.Second, 2)
	if entries[0].Line != "d" || entries[1].Line != "e" {
		t.Errorf("expected last 2 lines, got: %v", entries)
	}

	cancel()
	src.Stop()
}

func TestFileSource_LiveTailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("initial\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Read the initial line.
	collectLines(t, src, 2*time.Second, 1)

	// Append new lines.
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("tailed1\ntailed2\n")
	f.Close()

	entries := collectLines(t, src, 3*time.Second, 2)
	if entries[0].Line != "tailed1" || entries[1].Line != "tailed2" {
		t.Errorf("unexpected tailed lines: %v", entries)
	}

	cancel()
	src.Stop()
}

func TestFileSource_Truncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("old1\nold2\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	collectLines(t, src, 2*time.Second, 2)

	// Truncate and write new content.
	os.WriteFile(path, []byte("new1\n"), 0644)

	entries := collectLines(t, src, 3*time.Second, 1)
	if entries[0].Line != "new1" {
		t.Errorf("expected new1 after truncation, got: %s", entries[0].Line)
	}

	cancel()
	src.Stop()
}

func TestFileSource_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("before\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	collectLines(t, src, 2*time.Second, 1)

	// Simulate rotation: rename old, create new.
	os.Rename(path, path+".1")
	time.Sleep(200 * time.Millisecond)
	os.WriteFile(path, []byte("after\n"), 0644)

	entries := collectLines(t, src, 5*time.Second, 1)
	if entries[0].Line != "after" {
		t.Errorf("expected 'after' after rotation, got: %s", entries[0].Line)
	}

	cancel()
	src.Stop()
}

func TestFileSource_GlobPattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app1.log"), []byte("from-app1\n"), 0644)
	os.WriteFile(filepath.Join(dir, "app2.log"), []byte("from-app2\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore\n"), 0644)

	pattern := filepath.Join(dir, "*.log")
	src := NewFileSource(FileConfig{Patterns: []string{pattern}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	entries := collectLines(t, src, 2*time.Second, 2)
	lines := map[string]bool{}
	for _, e := range entries {
		lines[e.Line] = true
	}
	if !lines["from-app1"] || !lines["from-app2"] {
		t.Errorf("expected both app logs, got: %v", entries)
	}

	cancel()
	src.Stop()
}

func TestFileSource_MultiplePatterns(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.log")
	p2 := filepath.Join(dir, "b.txt")
	os.WriteFile(p1, []byte("aaa\n"), 0644)
	os.WriteFile(p2, []byte("bbb\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{p1, p2}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	entries := collectLines(t, src, 2*time.Second, 2)
	lines := map[string]bool{}
	for _, e := range entries {
		lines[e.Line] = true
	}
	if !lines["aaa"] || !lines["bbb"] {
		t.Errorf("expected both files, got: %v", entries)
	}

	cancel()
	src.Stop()
}

func TestFileSource_MissingFile(t *testing.T) {
	src := NewFileSource(FileConfig{Patterns: []string{"/nonexistent/file.log"}})
	ctx := context.Background()
	err := src.Start(ctx)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileSource_SourceMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.log")
	os.WriteFile(path, []byte("hello\n"), 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	entries := collectLines(t, src, 2*time.Second, 1)
	abs, _ := filepath.Abs(path)
	if entries[0].Source != abs {
		t.Errorf("expected source %s, got %s", abs, entries[0].Source)
	}

	cancel()
	src.Stop()
}

func TestFileSource_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.log")
	os.WriteFile(path, []byte{}, 0644)

	src := NewFileSource(FileConfig{Patterns: []string{path}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Write something after start.
	time.Sleep(200 * time.Millisecond)
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("appeared\n")
	f.Close()

	entries := collectLines(t, src, 3*time.Second, 1)
	if entries[0].Line != "appeared" {
		t.Errorf("expected 'appeared', got: %s", entries[0].Line)
	}

	cancel()
	src.Stop()
}
