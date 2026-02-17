package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)

// Model is the main TUI model for LogPilot.
type Model struct {
	width  int
	height int
	ready  bool
}

// NewModel creates a new LogPilot TUI model.
func NewModel() Model {
	return Model{}
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
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}
	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	title := titleStyle.Render("LogPilot")
	status := statusStyle.Render(fmt.Sprintf("Terminal: %dx%d | Press 'q' to quit", m.width, m.height))

	return fmt.Sprintf("%s\n\n  No log sources connected.\n  Usage: logpilot <file.log>\n\n%s", title, status)
}
