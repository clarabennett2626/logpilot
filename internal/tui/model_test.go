package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func setupModel(width, height int, lines int) Model {
	m := NewModel()
	// Simulate window size.
	m.width = width
	m.height = height
	m.ready = true
	// Add lines.
	for i := 0; i < lines; i++ {
		m.lines = append(m.lines, fmt.Sprintf("line %d", i))
	}
	if m.autoScroll {
		m.offset = m.maxOffset()
	}
	return m
}

func TestNewModel(t *testing.T) {
	m := NewModel()
	if !m.autoScroll {
		t.Error("expected autoScroll to be true by default")
	}
	if len(m.lines) != 0 {
		t.Error("expected empty lines buffer")
	}
}

func TestViewHeight(t *testing.T) {
	m := setupModel(80, 24, 0)
	// height=24, overhead=3 → viewHeight=21
	if vh := m.viewHeight(); vh != 21 {
		t.Errorf("viewHeight() = %d, want 21", vh)
	}
}

func TestViewHeightMinimum(t *testing.T) {
	m := setupModel(80, 2, 0)
	if vh := m.viewHeight(); vh < 1 {
		t.Errorf("viewHeight() = %d, want >= 1", vh)
	}
}

func TestMaxOffset(t *testing.T) {
	m := setupModel(80, 24, 100)
	// viewHeight=21, 100 lines → maxOffset=79
	if max := m.maxOffset(); max != 79 {
		t.Errorf("maxOffset() = %d, want 79", max)
	}
}

func TestMaxOffsetFewLines(t *testing.T) {
	m := setupModel(80, 24, 5)
	if max := m.maxOffset(); max != 0 {
		t.Errorf("maxOffset() = %d, want 0 (fewer lines than viewport)", max)
	}
}

func TestScrollDown(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 0
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)

	if m.offset != 1 {
		t.Errorf("offset = %d, want 1 after scroll down", m.offset)
	}
}

func TestScrollUp(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 10
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(Model)

	if m.offset != 9 {
		t.Errorf("offset = %d, want 9 after scroll up", m.offset)
	}
}

func TestScrollUpClampsAtZero(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 0
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(Model)

	if m.offset != 0 {
		t.Errorf("offset = %d, want 0 (clamped)", m.offset)
	}
}

func TestScrollToTop(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 50
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = updated.(Model)

	if m.offset != 0 {
		t.Errorf("offset = %d, want 0 after 'g'", m.offset)
	}
	if m.autoScroll {
		t.Error("autoScroll should be false after scrolling to top")
	}
}

func TestScrollToBottom(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 0
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)

	if m.offset != m.maxOffset() {
		t.Errorf("offset = %d, want %d after 'G'", m.offset, m.maxOffset())
	}
	if !m.autoScroll {
		t.Error("autoScroll should be true after scrolling to bottom")
	}
}

func TestPageDown(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 0
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(Model)

	if m.offset != 21 {
		t.Errorf("offset = %d, want 21 after page down", m.offset)
	}
}

func TestPageUp(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.offset = 50
	m.autoScroll = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(Model)

	if m.offset != 29 {
		t.Errorf("offset = %d, want 29 after page up", m.offset)
	}
}

func TestAutoScrollOnNewLine(t *testing.T) {
	m := setupModel(80, 24, 10)
	// autoScroll is true, so new lines should keep offset at bottom.
	updated, _ := m.Update(LogMsg{Rendered: "new line"})
	m = updated.(Model)

	if m.offset != m.maxOffset() {
		t.Errorf("offset = %d, want %d (auto-scroll to bottom)", m.offset, m.maxOffset())
	}
	if len(m.lines) != 11 {
		t.Errorf("lines count = %d, want 11", len(m.lines))
	}
}

func TestNoAutoScrollWhenScrolledUp(t *testing.T) {
	m := setupModel(80, 24, 50)
	m.autoScroll = false
	m.offset = 5

	updated, _ := m.Update(LogMsg{Rendered: "new line"})
	m = updated.(Model)

	if m.offset != 5 {
		t.Errorf("offset = %d, want 5 (should not auto-scroll)", m.offset)
	}
}

func TestLogBatchMsg(t *testing.T) {
	m := setupModel(80, 24, 0)
	batch := LogBatchMsg{Lines: []string{"a", "b", "c"}}

	updated, _ := m.Update(batch)
	m = updated.(Model)

	if len(m.lines) != 3 {
		t.Errorf("lines count = %d, want 3", len(m.lines))
	}
}

func TestWindowResize(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.autoScroll = true

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	if m.width != 120 || m.height != 40 {
		t.Errorf("dimensions = %dx%d, want 120x40", m.width, m.height)
	}
	if m.offset != m.maxOffset() {
		t.Errorf("offset = %d, want %d after resize with autoScroll", m.offset, m.maxOffset())
	}
}

func TestWindowResizeNoAutoScroll(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.autoScroll = false
	m.offset = 10

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	// Offset should be clamped but not forced to bottom.
	if m.offset != 10 {
		t.Errorf("offset = %d, want 10 after resize without autoScroll", m.offset)
	}
}

func TestViewNotReady(t *testing.T) {
	m := NewModel()
	if v := m.View(); v != "Loading..." {
		t.Errorf("View() = %q, want 'Loading...'", v)
	}
}

func TestViewEmptyState(t *testing.T) {
	m := setupModel(80, 24, 0)
	v := m.View()
	if v == "" {
		t.Error("View() should not be empty when ready")
	}
}

func TestViewWithLines(t *testing.T) {
	m := setupModel(80, 24, 5)
	v := m.View()
	if v == "" {
		t.Error("View() should not be empty with lines")
	}
	// Should contain the lines.
	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("line %d", i)
		if !contains(v, expected) {
			t.Errorf("View() should contain %q", expected)
		}
	}
}

func TestAutoScrollReenableAtBottom(t *testing.T) {
	m := setupModel(80, 24, 100)
	m.autoScroll = false
	m.offset = m.maxOffset() - 1

	// Scroll down to bottom.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)

	if !m.autoScroll {
		t.Error("autoScroll should re-enable when scrolled to bottom")
	}
}

func TestErrMsg(t *testing.T) {
	m := setupModel(80, 24, 0)
	updated, _ := m.Update(ErrMsg{Err: fmt.Errorf("test error")})
	m = updated.(Model)

	if len(m.lines) != 1 {
		t.Fatalf("lines count = %d, want 1", len(m.lines))
	}
	if !contains(m.lines[0], "test error") {
		t.Errorf("error line = %q, should contain 'test error'", m.lines[0])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
