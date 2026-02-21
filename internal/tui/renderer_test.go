package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/clarabennettdev/logpilot/internal/parser"
)

var fixedNow = time.Date(2026, 2, 17, 20, 0, 0, 0, time.UTC)

func fixedTime() time.Time { return fixedNow }

func plainRenderer(opts ...func(*RenderConfig)) *Renderer {
	cfg := DefaultConfig()
	cfg.Now = fixedTime
	cfg.TerminalWidth = 200 // wide enough to avoid truncation
	for _, o := range opts {
		o(&cfg)
	}
	return NewRenderer(cfg)
}

func TestRenderLevel_Colors(t *testing.T) {
	r := plainRenderer()
	tests := []struct {
		level    string
		contains string
	}{
		{"debug", "DEBUG"},
		{"INFO", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"fatal", "FATAL"},
		{"PANIC", "FATAL"},
		{"critical", "FATAL"},
	}
	for _, tt := range tests {
		entry := parser.LogEntry{Level: tt.level, Message: "test"}
		out := r.RenderEntryPlain(entry)
		if !strings.Contains(out, tt.contains) {
			t.Errorf("level=%q: expected %q in output, got %q", tt.level, tt.contains, out)
		}
	}
}

func TestRenderTimestamp_Relative(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.TimestampFormat = TimestampRelative })
	tests := []struct {
		offset   time.Duration
		contains string
	}{
		{0, "just now"},
		{30 * time.Second, "30s ago"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
		{48 * time.Hour, "2d ago"},
	}
	for _, tt := range tests {
		ts := fixedNow.Add(-tt.offset)
		entry := parser.LogEntry{Timestamp: ts, Message: "test"}
		out := r.RenderEntryPlain(entry)
		if !strings.Contains(out, tt.contains) {
			t.Errorf("offset=%v: expected %q in %q", tt.offset, tt.contains, out)
		}
	}
}

func TestRenderTimestamp_ISO(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.TimestampFormat = TimestampISO })
	ts := time.Date(2026, 2, 17, 15, 30, 45, 0, time.UTC)
	entry := parser.LogEntry{Timestamp: ts, Message: "hello"}
	out := r.RenderEntryPlain(entry)
	if !strings.Contains(out, "2026-02-17T15:30:45Z") {
		t.Errorf("expected ISO timestamp in %q", out)
	}
}

func TestRenderTimestamp_Local(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.TimestampFormat = TimestampLocal })
	ts := time.Date(2026, 2, 17, 15, 30, 45, 0, time.UTC)
	entry := parser.LogEntry{Timestamp: ts, Message: "hello"}
	out := r.RenderEntryPlain(entry)
	if !strings.Contains(out, "15:30:45") {
		t.Errorf("expected local time in %q", out)
	}
}

func TestRenderFields_ShowAll(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.ShowAllFields = true })
	entry := parser.LogEntry{
		Level:   "info",
		Message: "request handled",
		Fields:  map[string]string{"method": "GET", "path": "/api", "status": "200"},
	}
	out := r.RenderEntryPlain(entry)
	if !strings.Contains(out, "method=GET") || !strings.Contains(out, "path=/api") || !strings.Contains(out, "status=200") {
		t.Errorf("expected all fields in %q", out)
	}
}

func TestRenderFields_Collapsed(t *testing.T) {
	r := plainRenderer() // ShowAllFields = false
	entry := parser.LogEntry{
		Level:   "info",
		Message: "request handled",
		Fields:  map[string]string{"method": "GET", "path": "/api"},
	}
	out := r.RenderEntryPlain(entry)
	if strings.Contains(out, "method=GET") {
		t.Errorf("fields should be collapsed but got %q", out)
	}
}

func TestCollapsedFieldCount(t *testing.T) {
	entry := parser.LogEntry{
		Fields: map[string]string{"a": "1", "b": "2", "c": "3"},
	}
	if got := CollapsedFieldCount(entry); got != 3 {
		t.Errorf("expected 3 collapsed fields, got %d", got)
	}
}

func TestFieldOrder(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) {
		c.ShowAllFields = true
		c.FieldOrder = []string{"status", "method"}
	})
	entry := parser.LogEntry{
		Message: "req",
		Fields:  map[string]string{"method": "GET", "path": "/api", "status": "200"},
	}
	out := r.RenderEntryPlain(entry)
	statusIdx := strings.Index(out, "status=200")
	methodIdx := strings.Index(out, "method=GET")
	pathIdx := strings.Index(out, "path=/api")
	if statusIdx == -1 || methodIdx == -1 || pathIdx == -1 {
		t.Fatalf("missing fields in %q", out)
	}
	if statusIdx > methodIdx {
		t.Error("status should come before method per FieldOrder")
	}
	if methodIdx > pathIdx {
		t.Error("method should come before path (remaining sorted)")
	}
}

func TestStripANSI(t *testing.T) {
	input := "\x1b[31mERROR\x1b[0m something failed"
	got := StripANSI(input)
	if got != "ERROR something failed" {
		t.Errorf("StripANSI=%q, want %q", got, "ERROR something failed")
	}
}

func TestANSIPassthrough(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.ANSIMode = ANSIPassthrough })
	entry := parser.LogEntry{Message: "\x1b[31mred text\x1b[0m"}
	out := r.RenderEntryPlain(entry)
	if !strings.Contains(out, "\x1b[31m") {
		t.Error("ANSI codes should pass through")
	}
}

func TestANSIStrip(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.ANSIMode = ANSIStrip })
	entry := parser.LogEntry{Message: "\x1b[31mred text\x1b[0m"}
	out := r.RenderEntryPlain(entry)
	if strings.Contains(out, "\x1b[") {
		t.Error("ANSI codes should be stripped")
	}
	if !strings.Contains(out, "red text") {
		t.Error("text content should remain")
	}
}

func TestTruncation(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) {
		c.TerminalWidth = 30
		c.WrapMode = WrapTruncate
	})
	entry := parser.LogEntry{Message: "This is a very long message that should be truncated at the terminal width boundary"}
	out := r.RenderEntry(entry)
	plain := StripANSI(out)
	// ellipsis "…" is 3 bytes in UTF-8 but 1 visible char
	visibleLen := len([]rune(plain))
	if visibleLen > 31 { // 30 visible + ellipsis
		t.Errorf("expected truncated output <=31 runes, got %d: %q", visibleLen, plain)
	}
	if !strings.HasSuffix(plain, "…") {
		t.Error("truncated output should end with ellipsis")
	}
}

func TestWrapMode(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) {
		c.TerminalWidth = 30
		c.WrapMode = WrapWrap
	})
	entry := parser.LogEntry{Message: "This is a long message that should not be truncated in wrap mode"}
	out := r.RenderEntry(entry)
	plain := StripANSI(out)
	if strings.HasSuffix(plain, "…") {
		t.Error("wrap mode should not truncate")
	}
}

func TestDarkTheme(t *testing.T) {
	r := NewRenderer(RenderConfig{Theme: ThemeDark, TerminalWidth: 200, Now: fixedTime})
	entry := parser.LogEntry{Level: "error", Message: "fail"}
	out := r.RenderEntry(entry)
	if !strings.Contains(out, "ERROR") {
		t.Errorf("expected ERROR in output: %q", out)
	}
}

func TestLightTheme(t *testing.T) {
	r := NewRenderer(RenderConfig{Theme: ThemeLight, TerminalWidth: 200, Now: fixedTime})
	entry := parser.LogEntry{Level: "info", Message: "ok"}
	out := r.RenderEntry(entry)
	if !strings.Contains(out, "INFO") {
		t.Errorf("expected INFO in output: %q", out)
	}
}

func TestRenderEntry_EmptyEntry(t *testing.T) {
	r := plainRenderer()
	entry := parser.LogEntry{}
	out := r.RenderEntryPlain(entry)
	if out != "" {
		t.Errorf("empty entry should produce empty output, got %q", out)
	}
}

func TestRenderEntry_RawFallback(t *testing.T) {
	r := plainRenderer()
	entry := parser.LogEntry{Raw: "raw log line here"}
	out := r.RenderEntryPlain(entry)
	if !strings.Contains(out, "raw log line here") {
		t.Errorf("should fall back to Raw when Message empty: %q", out)
	}
}

func TestRenderEntry_FullIntegration(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) {
		c.TimestampFormat = TimestampISO
		c.ShowAllFields = true
	})
	entry := parser.LogEntry{
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     "error",
		Message:   "connection refused",
		Fields:    map[string]string{"host": "db.local", "port": "5432"},
		Format:    parser.FormatJSON,
	}
	out := r.RenderEntryPlain(entry)
	for _, want := range []string{"ERROR", "2026-01-15T10:30:00Z", "connection refused", "host=db.local", "port=5432"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output %q", want, out)
		}
	}
}

func TestRelativeTime_Future(t *testing.T) {
	r := plainRenderer(func(c *RenderConfig) { c.TimestampFormat = TimestampRelative })
	future := fixedNow.Add(5 * time.Minute)
	entry := parser.LogEntry{Timestamp: future, Message: "future"}
	out := r.RenderEntryPlain(entry)
	if !strings.Contains(out, "from now") {
		t.Errorf("expected 'from now' for future timestamp: %q", out)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.TerminalWidth != 120 {
		t.Errorf("default width=%d, want 120", cfg.TerminalWidth)
	}
	if cfg.Theme != ThemeDark {
		t.Error("default theme should be dark")
	}
	if cfg.ANSIMode != ANSIStrip {
		t.Error("default ANSI mode should be strip")
	}
	if cfg.ShowAllFields {
		t.Error("default should collapse fields")
	}
}
