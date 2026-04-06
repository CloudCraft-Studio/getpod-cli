package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
)

// StatusPickerModal lets the user select a new status for the current issue.
// Fetches available statuses from the PlanningPlugin at init time.
type StatusPickerModal struct {
	reg           *plugin.Registry
	items         []string
	cursor        int
	loading       bool
	err           error
	currentStatus string
	styles        Styles
}

// NewStatusPickerModal creates the modal. currentStatus is highlighted in the list.
func NewStatusPickerModal(reg *plugin.Registry, currentStatus string, styles Styles) *StatusPickerModal {
	return &StatusPickerModal{
		reg:           reg,
		currentStatus: currentStatus,
		styles:        styles,
		loading:       reg != nil,
	}
}

func (m *StatusPickerModal) Title() string { return "Change Status" }

func (m *StatusPickerModal) Init() tea.Cmd {
	if m.reg == nil {
		return nil
	}
	return m.fetchStatusesCmd()
}

func (m *StatusPickerModal) fetchStatusesCmd() tea.Cmd {
	reg := m.reg
	return func() tea.Msg {
		ctx := context.Background()
		for _, name := range reg.ActivePlugins() {
			p, ok := reg.Get(name)
			if !ok {
				continue
			}
			pp, ok := p.(plugin.PlanningPlugin)
			if !ok {
				continue
			}
			statuses, err := pp.ListStatuses(ctx)
			if err != nil {
				return StatusesFetchedMsg{Err: err}
			}
			return StatusesFetchedMsg{Statuses: statuses}
		}
		return StatusesFetchedMsg{Err: fmt.Errorf("no planning plugin configured")}
	}
}

func (m *StatusPickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusesFetchedMsg:
		m.loading = false
		m.err = msg.Err
		m.items = msg.Statuses
		// Position cursor on the current status
		for i, s := range m.items {
			if strings.EqualFold(s, m.currentStatus) {
				m.cursor = i
				break
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.items) {
				status := m.items[m.cursor]
				return m, func() tea.Msg { return StatusSelectedMsg{Status: status} }
			}
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m *StatusPickerModal) View() string {
	if m.loading {
		return m.styles.Placeholder.Render("Loading statuses...")
	}
	if m.err != nil {
		return m.styles.Placeholder.Render(fmt.Sprintf("Error: %v", m.err))
	}
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No statuses available.")
	}

	cursorStyle := lipgloss.NewStyle().Foreground(Primary400)
	currentStyle := lipgloss.NewStyle().Foreground(Success400)

	var lines []string
	for i, status := range m.items {
		indicator := "  "
		style := m.styles.Paragraph

		if strings.EqualFold(status, m.currentStatus) {
			indicator = currentStyle.Render("● ")
		}

		row := indicator + style.Render(status)
		if i == m.cursor {
			row = cursorStyle.Render("> ") + indicator + style.Render(status)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("enter select · esc cancel"))
	return strings.Join(lines, "\n")
}
