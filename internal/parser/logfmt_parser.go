package parser

import (
	"strings"
)

// LogfmtParser parses logfmt (key=value) log lines.
type LogfmtParser struct{}

// Parse parses a logfmt line.
func (p *LogfmtParser) Parse(line string) LogEntry {
	entry := LogEntry{
		Raw:    line,
		Format: FormatLogfmt,
		Fields: make(map[string]string),
	}

	pairs := parseLogfmtPairs(strings.TrimSpace(line))

	for k, v := range pairs {
		kl := strings.ToLower(k)
		switch {
		case isKnownKey(kl, timestampKeys):
			entry.Timestamp = parseTimestamp(v)
		case isKnownKey(kl, levelKeys):
			entry.Level = strings.ToUpper(v)
		case isKnownKey(kl, messageKeys):
			entry.Message = v
		default:
			entry.Fields[k] = v
		}
	}

	return entry
}

func parseLogfmtPairs(line string) map[string]string {
	pairs := make(map[string]string)
	i := 0
	for i < len(line) {
		// skip whitespace
		for i < len(line) && line[i] == ' ' {
			i++
		}
		// read key
		start := i
		for i < len(line) && line[i] != '=' && line[i] != ' ' {
			i++
		}
		if i >= len(line) || line[i] != '=' {
			break
		}
		key := line[start:i]
		i++ // skip '='

		var value string
		if i < len(line) && line[i] == '"' {
			// quoted value
			i++
			vstart := i
			for i < len(line) && line[i] != '"' {
				if line[i] == '\\' {
					i++
				}
				i++
			}
			value = line[vstart:i]
			if i < len(line) {
				i++ // skip closing quote
			}
		} else {
			vstart := i
			for i < len(line) && line[i] != ' ' {
				i++
			}
			value = line[vstart:i]
		}
		pairs[key] = value
	}
	return pairs
}
