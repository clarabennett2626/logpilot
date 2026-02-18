// Package tui provides terminal UI components for LogPilot.
package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/clarabennett2626/logpilot/internal/parser"
)

// TimestampFormat controls how timestamps are displayed.
type TimestampFormat int

const (
	// TimestampRelative shows "2m ago", "3h ago", etc.
	TimestampRelative TimestampFormat = iota
	// TimestampISO shows ISO 8601 format.
	TimestampISO
	// TimestampLocal shows local time format.
	TimestampLocal
)

// Theme represents terminal color theme.
type Theme int

const (
	ThemeDark Theme = iota
	ThemeLight
)

// ANSIMode controls how ANSI escape codes in source logs are handled.
type ANSIMode int

const (
	ANSIStrip ANSIMode = iota
	ANSIPassthrough
)

// WrapMode controls how long lines are handled.
type WrapMode int

const (
	WrapTruncate WrapMode = iota
	WrapWrap
)

// RenderConfig holds rendering configuration.
type RenderConfig struct {
	TimestampFormat TimestampFormat
	Theme          Theme
	ANSIMode       ANSIMode
	WrapMode       WrapMode
	TerminalWidth  int
	FieldOrder     []string // ordered field names to display; empty = alphabetical
	ShowAllFields  bool     // when false, extra fields are collapsed
	Now            func() time.Time // for testing; defaults to time.Now
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() RenderConfig {
	return RenderConfig{
		TimestampFormat: TimestampLocal,
		Theme:          ThemeDark,
		ANSIMode:       ANSIStrip,
		WrapMode:       WrapTruncate,
		TerminalWidth:  120,
		ShowAllFields:  false,
		Now:            time.Now,
	}
}

// Renderer renders parsed log entries as styled terminal output.
type Renderer struct {
	config RenderConfig
	styles themeStyles
}

type themeStyles struct {
	debug     lipgloss.Style
	info      lipgloss.Style
	warn      lipgloss.Style
	errLevel  lipgloss.Style
	fatal     lipgloss.Style
	timestamp lipgloss.Style
	message   lipgloss.Style
	fieldKey  lipgloss.Style
	fieldVal  lipgloss.Style
	separator lipgloss.Style
}

func darkStyles() themeStyles {
	return themeStyles{
		debug:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),            // gray
		info:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")),             // blue
		warn:      lipgloss.NewStyle().Foreground(lipgloss.Color("220")),            // yellow
		errLevel:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")),            // red
		fatal:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),  // red bold
		timestamp: lipgloss.NewStyle().Foreground(lipgloss.Color("243")),            // dim gray
		message:   lipgloss.NewStyle().Foreground(lipgloss.Color("255")),            // white
		fieldKey:  lipgloss.NewStyle().Foreground(lipgloss.Color("117")),            // light blue
		fieldVal:  lipgloss.NewStyle().Foreground(lipgloss.Color("252")),            // light gray
		separator: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),            // dark gray
	}
}

func lightStyles() themeStyles {
	return themeStyles{
		debug:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		info:      lipgloss.NewStyle().Foreground(lipgloss.Color("27")),
		warn:      lipgloss.NewStyle().Foreground(lipgloss.Color("172")),
		errLevel:  lipgloss.NewStyle().Foreground(lipgloss.Color("160")),
		fatal:     lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Bold(true),
		timestamp: lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		message:   lipgloss.NewStyle().Foreground(lipgloss.Color("0")),
		fieldKey:  lipgloss.NewStyle().Foreground(lipgloss.Color("25")),
		fieldVal:  lipgloss.NewStyle().Foreground(lipgloss.Color("237")),
		separator: lipgloss.NewStyle().Foreground(lipgloss.Color("249")),
	}
}

// NewRenderer creates a new Renderer with the given config.
func NewRenderer(config RenderConfig) *Renderer {
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.TerminalWidth <= 0 {
		config.TerminalWidth = 120
	}
	var styles themeStyles
	if config.Theme == ThemeLight {
		styles = lightStyles()
	} else {
		styles = darkStyles()
	}
	return &Renderer{config: config, styles: styles}
}

// ansiRegex matches ANSI escape sequences.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape codes from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// RenderEntry renders a single LogEntry as a styled string.
func (r *Renderer) RenderEntry(entry parser.LogEntry) string {
	var parts []string

	// Level badge
	levelStr := r.renderLevel(entry.Level)
	if levelStr != "" {
		parts = append(parts, levelStr)
	}

	// Timestamp
	if !entry.Timestamp.IsZero() {
		ts := r.renderTimestamp(entry.Timestamp)
		parts = append(parts, ts)
	}

	// Message
	msg := entry.Message
	if msg == "" {
		msg = entry.Raw
	}
	if r.config.ANSIMode == ANSIStrip {
		msg = StripANSI(msg)
	}
	if msg != "" {
		parts = append(parts, r.styles.message.Render(msg))
	}

	// Fields
	if r.config.ShowAllFields && len(entry.Fields) > 0 {
		fieldStr := r.renderFields(entry.Fields)
		if fieldStr != "" {
			parts = append(parts, fieldStr)
		}
	}

	line := strings.Join(parts, r.styles.separator.Render(" │ "))

	// Truncate or wrap
	line = r.applyWrap(line)

	return line
}

// RenderEntryPlain renders without styling (for piping/testing visible text).
func (r *Renderer) RenderEntryPlain(entry parser.LogEntry) string {
	var parts []string

	if entry.Level != "" {
		parts = append(parts, strings.ToUpper(normalizeLevel(entry.Level)))
	}
	if !entry.Timestamp.IsZero() {
		parts = append(parts, r.formatTimestamp(entry.Timestamp))
	}
	msg := entry.Message
	if msg == "" {
		msg = entry.Raw
	}
	if r.config.ANSIMode == ANSIStrip {
		msg = StripANSI(msg)
	}
	if msg != "" {
		parts = append(parts, msg)
	}
	if r.config.ShowAllFields && len(entry.Fields) > 0 {
		parts = append(parts, r.renderFieldsPlain(entry.Fields))
	}
	return strings.Join(parts, " │ ")
}

// CollapsedFieldCount returns how many extra fields would be hidden.
func CollapsedFieldCount(entry parser.LogEntry) int {
	return len(entry.Fields)
}

func (r *Renderer) renderLevel(level string) string {
	norm := normalizeLevel(level)
	label := fmt.Sprintf("%-5s", strings.ToUpper(norm))
	switch norm {
	case "debug":
		return r.styles.debug.Render(label)
	case "info":
		return r.styles.info.Render(label)
	case "warn", "warning":
		return r.styles.warn.Render(label)
	case "error":
		return r.styles.errLevel.Render(label)
	case "fatal", "panic", "critical":
		return r.styles.fatal.Render(label)
	default:
		if level == "" {
			return ""
		}
		return r.styles.message.Render(label)
	}
}

func normalizeLevel(level string) string {
	l := strings.ToLower(strings.TrimSpace(level))
	switch l {
	case "warning":
		return "warn"
	case "critical", "panic":
		return "fatal"
	default:
		return l
	}
}

func (r *Renderer) renderTimestamp(t time.Time) string {
	return r.styles.timestamp.Render(r.formatTimestamp(t))
}

func (r *Renderer) formatTimestamp(t time.Time) string {
	switch r.config.TimestampFormat {
	case TimestampRelative:
		return relativeTime(t, r.config.Now())
	case TimestampISO:
		return t.Format(time.RFC3339)
	case TimestampLocal:
		return t.Format("15:04:05")
	default:
		return t.Format(time.RFC3339)
	}
}

func relativeTime(t time.Time, now time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = -d
		return formatDuration(d) + " from now"
	}
	if d < time.Second {
		return "just now"
	}
	return formatDuration(d) + " ago"
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		s := int(d.Seconds())
		return fmt.Sprintf("%ds", s)
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh", h)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

func (r *Renderer) renderFields(fields map[string]string) string {
	ordered := r.orderedFieldKeys(fields)
	var parts []string
	for _, k := range ordered {
		v := fields[k]
		part := r.styles.fieldKey.Render(k) + r.styles.separator.Render("=") + r.styles.fieldVal.Render(v)
		parts = append(parts, part)
	}
	return strings.Join(parts, " ")
}

func (r *Renderer) renderFieldsPlain(fields map[string]string) string {
	ordered := r.orderedFieldKeys(fields)
	var parts []string
	for _, k := range ordered {
		parts = append(parts, k+"="+fields[k])
	}
	return strings.Join(parts, " ")
}

func (r *Renderer) orderedFieldKeys(fields map[string]string) []string {
	if len(r.config.FieldOrder) > 0 {
		var result []string
		seen := make(map[string]bool)
		for _, k := range r.config.FieldOrder {
			if _, ok := fields[k]; ok {
				result = append(result, k)
				seen[k] = true
			}
		}
		// append remaining keys alphabetically
		remaining := make([]string, 0, len(fields))
		for k := range fields {
			if !seen[k] {
				remaining = append(remaining, k)
			}
		}
		sortStrings(remaining)
		result = append(result, remaining...)
		return result
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

// simple insertion sort to avoid importing sort package
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func (r *Renderer) applyWrap(line string) string {
	if r.config.WrapMode == WrapTruncate && r.config.TerminalWidth > 0 {
		// Strip ANSI to measure visible length, but truncate the raw string
		visible := StripANSI(line)
		if len(visible) > r.config.TerminalWidth {
			// Truncate by visible chars. Rough approach: walk raw string.
			return truncateToWidth(line, r.config.TerminalWidth-1) + "…"
		}
	}
	// WrapWrap: lipgloss handles wrapping naturally, just return as-is
	return line
}

// truncateToWidth truncates a string with ANSI codes to fit a visible width.
func truncateToWidth(s string, width int) string {
	visible := 0
	inEscape := false
	var result []byte
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '\x1b' {
			inEscape = true
			result = append(result, b)
			continue
		}
		if inEscape {
			result = append(result, b)
			if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
				inEscape = false
			}
			continue
		}
		if visible >= width {
			break
		}
		result = append(result, b)
		visible++
	}
	return string(result)
}
