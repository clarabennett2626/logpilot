// Package source provides log source readers (file, stdin, k8s, docker, ssh).
package source

import "context"

// LogEntry represents a single log line with metadata.
type LogEntry struct {
	// Line is the raw log line text.
	Line string
	// Source identifies which file/source produced this entry.
	Source string
}

// Source defines the interface for all log sources.
type Source interface {
	// Lines returns a channel that emits log entries.
	Lines() <-chan LogEntry
	// Errors returns a channel that emits errors encountered during reading.
	Errors() <-chan error
	// Start begins reading/tailing. It blocks until ctx is cancelled.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the source.
	Stop() error
}
