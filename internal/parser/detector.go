// Package parser provides log format detection and parsing.
package parser

import (
	"strings"
	"time"
)

// Format represents a log format type.
type Format int

const (
	FormatUnknown Format = iota
	FormatJSON
	FormatLogfmt
	FormatPlain
)

func (f Format) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatLogfmt:
		return "logfmt"
	case FormatPlain:
		return "plain"
	default:
		return "unknown"
	}
}

// LogEntry is the unified parsed log entry.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Fields    map[string]string
	Raw       string
	Format    Format
}

// Parser can parse a single log line into a LogEntry.
type Parser interface {
	Parse(line string) LogEntry
}

// DetectFormat analyzes the first N lines and returns the most likely format.
func DetectFormat(lines []string) Format {
	if len(lines) == 0 {
		return FormatUnknown
	}

	jsonCount, logfmtCount, plainCount := 0, 0, 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch detectLine(line) {
		case FormatJSON:
			jsonCount++
		case FormatLogfmt:
			logfmtCount++
		default:
			plainCount++
		}
	}

	if jsonCount >= logfmtCount && jsonCount >= plainCount && jsonCount > 0 {
		return FormatJSON
	}
	if logfmtCount >= jsonCount && logfmtCount >= plainCount && logfmtCount > 0 {
		return FormatLogfmt
	}
	if plainCount > 0 {
		return FormatPlain
	}
	return FormatUnknown
}

// detectLine determines the format of a single line.
func detectLine(line string) Format {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 {
		return FormatUnknown
	}
	if trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		return FormatJSON
	}
	if isLogfmt(trimmed) {
		return FormatLogfmt
	}
	return FormatPlain
}

// isLogfmt checks if a line looks like key=value pairs.
func isLogfmt(line string) bool {
	// Must have at least 2 key=value pairs to be considered logfmt
	count := 0
	i := 0
	for i < len(line) {
		// skip whitespace
		for i < len(line) && line[i] == ' ' {
			i++
		}
		// find '='
		start := i
		for i < len(line) && line[i] != '=' && line[i] != ' ' {
			i++
		}
		if i >= len(line) || line[i] != '=' || i == start {
			break
		}
		i++ // skip '='
		if i < len(line) && line[i] == '"' {
			// quoted value
			i++
			for i < len(line) && line[i] != '"' {
				if line[i] == '\\' {
					i++
				}
				i++
			}
			if i < len(line) {
				i++ // skip closing quote
			}
		} else {
			for i < len(line) && line[i] != ' ' {
				i++
			}
		}
		count++
	}
	return count >= 2
}

// NewParser returns the appropriate parser for the given format.
func NewParser(f Format) Parser {
	switch f {
	case FormatJSON:
		return &JSONParser{}
	case FormatLogfmt:
		return &LogfmtParser{}
	default:
		return &PlainParser{}
	}
}

// AutoParser detects the format per-line for mixed format streams.
type AutoParser struct {
	jsonParser   JSONParser
	logfmtParser LogfmtParser
	plainParser  PlainParser
}

// NewAutoParser creates a parser that handles mixed formats.
func NewAutoParser() *AutoParser {
	return &AutoParser{}
}

// Parse detects and parses a single line.
func (a *AutoParser) Parse(line string) LogEntry {
	switch detectLine(line) {
	case FormatJSON:
		return a.jsonParser.Parse(line)
	case FormatLogfmt:
		return a.logfmtParser.Parse(line)
	default:
		return a.plainParser.Parse(line)
	}
}
