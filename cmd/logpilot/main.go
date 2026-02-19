package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clarabennett2626/logpilot/internal/parser"
	"github.com/clarabennett2626/logpilot/internal/source"
	"github.com/clarabennett2626/logpilot/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("logpilot %s (%s) built %s\n", version, commit, date)
		os.Exit(0)
	}

	// If stdin is a pipe, run in streaming mode (no TUI).
	if source.IsPipe() {
		if err := runPipeMode(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runPipeMode reads from stdin, parses each line, and renders output to stdout.
func runPipeMode() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	src := source.NewStdinSource()
	autoParser := parser.NewAutoParser()
	renderer := tui.NewRenderer(tui.DefaultConfig())

	// Start reading stdin in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- src.Start(ctx)
	}()

	// Consume lines and render them.
	for entry := range src.Lines() {
		parsed := autoParser.Parse(entry.Line)
		fmt.Println(renderer.RenderEntry(parsed))
	}

	// Check for read errors.
	if err := <-errCh; err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}
