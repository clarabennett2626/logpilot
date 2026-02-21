package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clarabennett2626/logpilot/internal/parser"
	"github.com/clarabennett2626/logpilot/internal/source"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Background(lipgloss.Color("#333333")).
			Bold(true).
			Padding(0, 1)

	cursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3C3C5C"))

	detailBorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))

	detailKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#117")).
			Bold(true)

	detailValStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#252"))
)

// LogMsg carries a new parsed and rendered log line into the TUI.
type LogMsg struct {
	Rendered string
	Entry    parser.LogEntry
}

// LogBatchMsg carries multiple rendered log lines at once.
type LogBatchMsg struct {
	Lines   []string
	Entries []parser.LogEntry
}

// ErrMsg carries a source error into the TUI.
type ErrMsg struct {
	Err error
}

// Model is the main TUI model for LogPilot.
type Model struct {
	width  int
	height int
	ready  bool

	// Log buffer — stores rendered strings for display.
	lines   []string
	entries []parser.LogEntry // parallel to lines; stores parsed entries

	// Virtual scrolling state.
	offset     int  // index of the first visible line
	autoScroll bool // stick to bottom when new lines arrive

	// Cursor and detail pane.
	cursor    int  // index of the highlighted line
	showDetail bool // whether the detail pane is visible

	// Source info for status bar.
	sourceName string

	// Filter status for status bar.
	filterText string
}

// NewModel creates a new LogPilot TUI model with no sources.
func NewModel() Model {
	return Model{
		autoScroll: true,
	}
}

// NewModelWithSource creates a TUI model wired to a log source.
func NewModelWithSource(src source.Source, sourceName string) Model {
	return Model{
		autoScroll: true,
		sourceName: sourceName,
	}
}

// viewHeight returns the number of lines available for log display
// (total height minus title bar and status bar).
func (m Model) viewHeight() int {
	// 1 line title + 1 blank + 1 status bar = 3 overhead lines
	h := m.height - 3
	if h < 1 {
		return 1
	}
	return h
}

// detailPaneHeight returns the height of the detail pane when visible.
func (m Model) detailPaneHeight() int {
	h := m.viewHeight() / 3
	if h < 5 {
		h = 5
	}
	if h > 15 {
		h = 15
	}
	return h
}

// logPaneHeight returns the log viewport height when detail pane is visible.
func (m Model) logPaneHeight() int {
	if !m.showDetail {
		return m.viewHeight()
	}
	// detail pane takes detailPaneHeight + 1 (border line)
	h := m.viewHeight() - m.detailPaneHeight() - 1
	if h < 3 {
		return 3
	}
	return h
}

// maxOffset returns the maximum valid scroll offset.
func (m Model) maxOffset() int {
	max := len(m.lines) - m.viewHeight()
	if max < 0 {
		return 0
	}
	return max
}

// clampCursor ensures cursor is within valid bounds.
func (m *Model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if max := len(m.lines) - 1; m.cursor > max {
		if max < 0 {
			m.cursor = 0
		} else {
			m.cursor = max
		}
	}
}

// scrollToCursor adjusts offset so the cursor is visible.
func (m *Model) scrollToCursor() {
	vh := m.viewHeight()
	if m.showDetail {
		vh = m.logPaneHeight()
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vh {
		m.offset = m.cursor - vh + 1
	}
	m.clampOffset()
}

// clampOffset ensures offset is within valid bounds.
func (m *Model) clampOffset() {
	if m.offset < 0 {
		m.offset = 0
	}
	if max := m.maxOffset(); m.offset > max {
		m.offset = max
	}
}

// isAtBottom returns true if the viewport is scrolled to the bottom.
func (m Model) isAtBottom() bool {
	return m.offset >= m.maxOffset()
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if len(m.lines) > 0 {
				m.showDetail = !m.showDetail
			}
		case "esc":
			if m.showDetail {
				m.showDetail = false
			}
		case "j", "down":
			m.autoScroll = false
			m.cursor++
			m.clampCursor()
			m.scrollToCursor()
			if m.isAtBottom() {
				m.autoScroll = true
			}
		case "k", "up":
			m.autoScroll = false
			m.cursor--
			m.clampCursor()
			m.scrollToCursor()
		case "g", "home":
			m.autoScroll = false
			m.cursor = 0
			m.offset = 0
		case "G", "end":
			m.cursor = len(m.lines) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.offset = m.maxOffset()
			m.autoScroll = true
		case "pgdown", "f", "ctrl+f":
			m.autoScroll = false
			m.cursor += m.viewHeight()
			m.clampCursor()
			m.offset += m.viewHeight()
			m.clampOffset()
			if m.isAtBottom() {
				m.autoScroll = true
			}
		case "pgup", "b", "ctrl+b":
			m.autoScroll = false
			m.cursor -= m.viewHeight()
			m.clampCursor()
			m.offset -= m.viewHeight()
			m.clampOffset()
		case "d", "ctrl+d":
			m.autoScroll = false
			m.cursor += m.viewHeight() / 2
			m.clampCursor()
			m.offset += m.viewHeight() / 2
			m.clampOffset()
			if m.isAtBottom() {
				m.autoScroll = true
			}
		case "u", "ctrl+u":
			m.autoScroll = false
			m.cursor -= m.viewHeight() / 2
			m.clampCursor()
			m.offset -= m.viewHeight() / 2
			m.clampOffset()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		if m.autoScroll {
			m.offset = m.maxOffset()
		}
		m.clampOffset()

	case LogMsg:
		m.lines = append(m.lines, msg.Rendered)
		m.entries = append(m.entries, msg.Entry)
		if m.autoScroll {
			m.offset = m.maxOffset()
			m.cursor = len(m.lines) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
		}

	case LogBatchMsg:
		m.lines = append(m.lines, msg.Lines...)
		m.entries = append(m.entries, msg.Entries...)
		if m.autoScroll {
			m.offset = m.maxOffset()
			m.cursor = len(m.lines) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
		}

	case ErrMsg:
		// Show error as a log line.
		m.lines = append(m.lines, fmt.Sprintf("ERROR: %v", msg.Err))
		if m.autoScroll {
			m.offset = m.maxOffset()
		}
	}
	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar.
	title := titleStyle.Render("LogPilot")
	b.WriteString(title)
	b.WriteByte('\n')

	// Log viewport — virtual scrolling: only render visible slice.
	vh := m.logPaneHeight()
	if len(m.lines) == 0 {
		// Empty state.
		for i := 0; i < vh; i++ {
			if i == vh/2-1 {
				b.WriteString("  No log entries yet.")
			} else if i == vh/2 {
				b.WriteString("  Waiting for input...")
			}
			b.WriteByte('\n')
		}
	} else {
		end := m.offset + vh
		if end > len(m.lines) {
			end = len(m.lines)
		}
		start := m.offset
		if start < 0 {
			start = 0
		}
		// Render visible lines with cursor highlight.
		rendered := 0
		for i := start; i < end; i++ {
			line := m.lines[i]
			if i == m.cursor {
				line = cursorStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteByte('\n')
			rendered++
		}
		// Pad remaining lines.
		for i := rendered; i < vh; i++ {
			b.WriteByte('\n')
		}
	}

	// Detail pane.
	if m.showDetail && len(m.entries) > 0 && m.cursor < len(m.entries) {
		b.WriteString(m.renderDetailPane())
	}

	// Status bar.
	total := len(m.lines)
	scrollInfo := "bottom"
	if total > 0 && !m.isAtBottom() {
		pct := 0
		if m.maxOffset() > 0 {
			pct = m.offset * 100 / m.maxOffset()
		}
		scrollInfo = fmt.Sprintf("%d%%", pct)
	}

	src := m.sourceName
	if src == "" {
		src = "stdin"
	}

	left := statusKeyStyle.Render("Lines:") + statusBarStyle.Render(fmt.Sprintf(" %d ", total))
	right := statusKeyStyle.Render("Pos:") + statusBarStyle.Render(fmt.Sprintf(" %s ", scrollInfo))
	srcInfo := statusKeyStyle.Render("Src:") + statusBarStyle.Render(fmt.Sprintf(" %s ", src))

	// Filter status.
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = statusKeyStyle.Render("Filter:") + statusBarStyle.Render(fmt.Sprintf(" %s ", m.filterText))
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - lipgloss.Width(srcInfo) - lipgloss.Width(filterInfo)
	if gap < 0 {
		gap = 0
	}
	statusLine := left + srcInfo + filterInfo + strings.Repeat(" ", gap) + right
	// Fill background.
	statusLine = statusBarStyle.Render(statusLine)
	b.WriteString(statusLine)

	return b.String()
}

// renderDetailPane renders the detail pane for the selected log entry.
func (m Model) renderDetailPane() string {
	var b strings.Builder

	// Separator line.
	sep := detailBorderStyle.Render(strings.Repeat("─", m.width))
	b.WriteString(sep)
	b.WriteByte('\n')

	entry := m.entries[m.cursor]
	dh := m.detailPaneHeight()
	rendered := 0

	// Header.
	header := detailBorderStyle.Render("▼ Detail")
	b.WriteString(header)
	b.WriteByte('\n')
	rendered++

	// Timestamp.
	if !entry.Timestamp.IsZero() && rendered < dh {
		b.WriteString(detailKeyStyle.Render("  timestamp") + " " + detailValStyle.Render(entry.Timestamp.Format("2006-01-02 15:04:05.000")))
		b.WriteByte('\n')
		rendered++
	}

	// Level.
	if entry.Level != "" && rendered < dh {
		b.WriteString(detailKeyStyle.Render("  level    ") + " " + detailValStyle.Render(entry.Level))
		b.WriteByte('\n')
		rendered++
	}

	// Message.
	if entry.Message != "" && rendered < dh {
		b.WriteString(detailKeyStyle.Render("  message  ") + " " + detailValStyle.Render(entry.Message))
		b.WriteByte('\n')
		rendered++
	}

	// Format.
	if rendered < dh {
		b.WriteString(detailKeyStyle.Render("  format   ") + " " + detailValStyle.Render(entry.Format.String()))
		b.WriteByte('\n')
		rendered++
	}

	// Fields.
	if len(entry.Fields) > 0 {
		keys := make([]string, 0, len(entry.Fields))
		for k := range entry.Fields {
			keys = append(keys, k)
		}
		sortDetailKeys(keys)
		for _, k := range keys {
			if rendered >= dh {
				break
			}
			label := fmt.Sprintf("  %-10s", k)
			b.WriteString(detailKeyStyle.Render(label) + " " + detailValStyle.Render(entry.Fields[k]))
			b.WriteByte('\n')
			rendered++
		}
	}

	// Pad remaining.
	for rendered < dh {
		b.WriteByte('\n')
		rendered++
	}

	return b.String()
}

// sortDetailKeys sorts keys alphabetically (simple insertion sort).
func sortDetailKeys(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// WaitForLines returns a tea.Cmd that reads from a source and sends LogMsg
// messages to the TUI. Call this to wire a source into the model.
func WaitForLines(src source.Source, p *parser.AutoParser, r *Renderer) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-src.Lines()
		if !ok {
			return nil
		}
		entry := p.Parse(line.Line)
		rendered := r.RenderEntry(entry)
		return LogMsg{Rendered: rendered, Entry: entry}
	}
}

// ListenForLines returns a tea.Cmd that continuously reads from a source
// and sends lines to the program. Use with tea.Program.Send from a goroutine.
func ListenForLines(src source.Source, p *parser.AutoParser, r *Renderer, prog *tea.Program) {
	go func() {
		for line := range src.Lines() {
			entry := p.Parse(line.Line)
			rendered := r.RenderEntry(entry)
			prog.Send(LogMsg{Rendered: rendered, Entry: entry})
		}
	}()
	go func() {
		for err := range src.Errors() {
			prog.Send(ErrMsg{Err: err})
		}
	}()
}
