// Demo tool that showcases LogPilot's parsing and rendering capabilities.
// Used for generating README GIF demos.
package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/clarabennett2626/logpilot/internal/parser"
	"github.com/clarabennett2626/logpilot/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: demo <logfile>\n")
		os.Exit(1)
	}

	// Detect format
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	format := parser.DetectFormat(lines)
	fmt.Printf("📋 Detected format: %s (%d lines)\n\n", format, len(lines))

	// Parse and render each line
	p := parser.NewAutoParser()
	renderer := tui.NewRenderer(tui.RenderConfig{
		TimestampFormat: tui.TimestampRelative,
		Theme:           tui.ThemeDark,
		TerminalWidth:   120,
		WrapMode:        tui.WrapTruncate,
		ShowAllFields:   true,
		ANSIMode:        tui.ANSIStrip,
	})

	for _, line := range lines {
		entry := p.Parse(line)
		rendered := renderer.RenderEntry(entry)
		fmt.Println(rendered)
		time.Sleep(80 * time.Millisecond) // Simulate streaming
	}
}
