package parser

import (
	"regexp"
	"strings"
)

// PlainParser parses plain text log lines with regex-based timestamp extraction.
type PlainParser struct{}

var plainTimestampPatterns = []*regexp.Regexp{
	// ISO 8601 variants
	regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+`),
	// Syslog: Jan  2 15:04:05
	regexp.MustCompile(`^([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+`),
	// Apache/Nginx: 02/Jan/2006:15:04:05 -0700
	regexp.MustCompile(`\[(\d{2}/[A-Z][a-z]{2}/\d{4}:\d{2}:\d{2}:\d{2}\s+[+-]\d{4})\]`),
	// Slash date: 2006/01/02 15:04:05
	regexp.MustCompile(`^(\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2})\s+`),
}

var levelPattern = regexp.MustCompile(`(?i)\b(TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL|PANIC)\b`)

// Parse parses a plain text log line.
func (p *PlainParser) Parse(line string) LogEntry {
	entry := LogEntry{
		Raw:    line,
		Format: FormatPlain,
		Fields: make(map[string]string),
	}

	remaining := line

	// Try to extract timestamp
	for _, pat := range plainTimestampPatterns {
		if m := pat.FindStringSubmatch(line); m != nil {
			entry.Timestamp = parseTimestamp(m[1])
			if !entry.Timestamp.IsZero() {
				// Remove timestamp from remaining
				remaining = strings.TrimSpace(strings.Replace(line, m[0], "", 1))
				break
			}
		}
	}

	// Try to extract level
	if m := levelPattern.FindString(remaining); m != "" {
		entry.Level = strings.ToUpper(m)
		if entry.Level == "WARNING" {
			entry.Level = "WARN"
		}
	}

	entry.Message = remaining
	return entry
}
