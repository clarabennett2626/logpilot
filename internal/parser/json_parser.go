package parser

import (
	"encoding/json"
	"strings"
	"time"
)

var (
	timestampKeys = []string{"timestamp", "time", "ts", "@timestamp", "created_at"}
	levelKeys     = []string{"level", "severity", "log_level", "lvl"}
	messageKeys   = []string{"message", "msg", "log", "text"}
)

// JSONParser parses JSON log lines.
type JSONParser struct{}

// Parse parses a JSON log line.
func (p *JSONParser) Parse(line string) LogEntry {
	entry := LogEntry{
		Raw:    line,
		Format: FormatJSON,
		Fields: make(map[string]string),
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &raw); err != nil {
		entry.Message = line
		return entry
	}

	// Extract known fields
	entry.Timestamp = extractTimestamp(raw, timestampKeys)
	entry.Level = extractString(raw, levelKeys)
	entry.Message = extractString(raw, messageKeys)

	// Remaining fields
	for k, v := range raw {
		kl := strings.ToLower(k)
		if isKnownKey(kl, timestampKeys) || isKnownKey(kl, levelKeys) || isKnownKey(kl, messageKeys) {
			continue
		}
		switch val := v.(type) {
		case string:
			entry.Fields[k] = val
		default:
			b, _ := json.Marshal(val)
			entry.Fields[k] = string(b)
		}
	}

	// Normalize level
	entry.Level = strings.ToUpper(entry.Level)

	return entry
}

func extractString(m map[string]interface{}, keys []string) string {
	for k, v := range m {
		kl := strings.ToLower(k)
		for _, key := range keys {
			if kl == key {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func extractTimestamp(m map[string]interface{}, keys []string) time.Time {
	for k, v := range m {
		kl := strings.ToLower(k)
		for _, key := range keys {
			if kl == key {
				return parseTimestamp(v)
			}
		}
	}
	return time.Time{}
}

func isKnownKey(k string, keys []string) bool {
	for _, key := range keys {
		if k == key {
			return true
		}
	}
	return false
}

var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.000",
	"2006-01-02T15:04:05.000Z",
	"02/Jan/2006:15:04:05 -0700",
	"Jan  2 15:04:05",
	"Jan 2 15:04:05",
	"2006/01/02 15:04:05",
}

func parseTimestamp(v interface{}) time.Time {
	switch val := v.(type) {
	case string:
		for _, layout := range timeFormats {
			if t, err := time.Parse(layout, val); err == nil {
				return t
			}
		}
	case float64:
		// Unix timestamp (seconds or milliseconds)
		if val > 1e12 {
			return time.UnixMilli(int64(val))
		}
		return time.Unix(int64(val), 0)
	}
	return time.Time{}
}
