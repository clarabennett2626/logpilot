package source

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func stdinCollectLines(t *testing.T, s *StdinSource, timeout time.Duration) []LogEntry {
	t.Helper()
	var entries []LogEntry
	deadline := time.After(timeout)
	for {
		select {
		case entry, ok := <-s.Lines():
			if !ok {
				return entries
			}
			entries = append(entries, entry)
		case <-deadline:
			t.Fatal("timed out waiting for lines")
			return nil
		}
	}
}

func TestStdinSource_BasicRead(t *testing.T) {
	input := "line one\nline two\nline three\n"
	src := NewStdinSource(WithReader(strings.NewReader(input)))

	go src.Start(context.Background())
	entries := stdinCollectLines(t, src, 2*time.Second)

	want := []string{"line one", "line two", "line three"}
	if len(entries) != len(want) {
		t.Fatalf("got %d lines, want %d", len(entries), len(want))
	}
	for i, e := range entries {
		if e.Line != want[i] {
			t.Errorf("line %d: got %q, want %q", i, e.Line, want[i])
		}
		if e.Source != "stdin" {
			t.Errorf("line %d: source = %q, want \"stdin\"", i, e.Source)
		}
	}
}

func TestStdinSource_EmptyInput(t *testing.T) {
	src := NewStdinSource(WithReader(strings.NewReader("")))

	go src.Start(context.Background())
	entries := stdinCollectLines(t, src, 2*time.Second)

	if len(entries) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(entries))
	}
}

func TestStdinSource_ContextCancellation(t *testing.T) {
	pr, pw := io.Pipe()

	src := NewStdinSource(WithReader(pr))
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		src.Start(ctx)
		close(done)
	}()

	pw.Write([]byte("hello\n"))
	<-src.Lines()
	cancel()
	pw.Close() // unblock scanner

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestStdinSource_Stop(t *testing.T) {
	pr, pw := io.Pipe()

	src := NewStdinSource(WithReader(pr))

	go src.Start(context.Background())

	pw.Write([]byte("line\n"))
	<-src.Lines()

	go pw.Close()
	src.Stop()

	_, ok := <-src.Lines()
	if ok {
		t.Fatal("expected lines channel to be closed")
	}
}

func TestStdinSource_DropOldest(t *testing.T) {
	input := "a\nb\nc\nd\n"
	src := NewStdinSource(
		WithReader(strings.NewReader(input)),
		WithBufferSize(2),
		WithBackpressure(DropOldest),
	)

	go src.Start(context.Background())
	entries := stdinCollectLines(t, src, 2*time.Second)

	if len(entries) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(entries))
	}
	if entries[len(entries)-1].Line != "d" {
		t.Errorf("last line should be 'd', got %q", entries[len(entries)-1].Line)
	}
}

func TestStdinSource_LongLines(t *testing.T) {
	long := strings.Repeat("x", 500_000)
	src := NewStdinSource(WithReader(strings.NewReader(long + "\n")))

	go src.Start(context.Background())
	entries := stdinCollectLines(t, src, 2*time.Second)

	if len(entries) != 1 || len(entries[0].Line) != 500_000 {
		t.Fatalf("expected 1 line of 500000 chars, got %d lines", len(entries))
	}
}

func TestStdinSource_Errors(t *testing.T) {
	src := NewStdinSource(WithReader(strings.NewReader("")))

	go src.Start(context.Background())
	stdinCollectLines(t, src, 2*time.Second)

	select {
	case err := <-src.Errors():
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	default:
	}
}

func TestStdinSource_ImplementsSource(t *testing.T) {
	var _ Source = (*StdinSource)(nil)
}

func TestIsPipe(t *testing.T) {
	_ = IsPipe()
}
