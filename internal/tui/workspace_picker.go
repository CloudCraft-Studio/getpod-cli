package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// WorkspacePickerModal lets the user pick a single workspace from the client config.
type WorkspacePickerModal struct {
	items  []workspaceItem
	cursor int
	styles Styles
}

type workspaceItem struct {
	name        string
	displayName string
	envCount    int
}

// NewWorkspacePickerModal builds the list from cfg for the given client.
func NewWorkspacePickerModal(cfg *config.Config, client string, styles Styles) *WorkspacePickerModal {
	cl := cfg.Clients[client]
	var items []workspaceItem
	for name, ws := range cl.Workspaces {
		dn := ws.DisplayName
		if dn == "" {
			dn = name
		}
		items = append(items, workspaceItem{
			name:        name,
			displayName: dn,
			envCount:    len(ws.Contexts),
		})
	}
	// sort alphabetically by name
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].name > items[j].name {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return &WorkspacePickerModal{items: items, styles: styles}
}

func (m *WorkspacePickerModal) Title() string { return "Select Workspace" }
func (m *WorkspacePickerModal) Init() tea.Cmd { return nil }

func (m *WorkspacePickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
				name := m.items[m.cursor].name
				return m, func() tea.Msg { return WorkspaceSelectedMsg{Workspace: name} }
			}
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m *WorkspacePickerModal) View() string {
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No workspaces configured for this client.")
	}
	cursorStyle := lipgloss.NewStyle().Foreground(Primary400)
	var lines []string
	for i, ws := range m.items {
		envLabel := fmt.Sprintf("%d env", ws.envCount)
		if ws.envCount != 1 {
			envLabel += "s"
		}
		row := fmt.Sprintf("%-22s  %s", ws.displayName, m.styles.Muted.Render(envLabel))
		if i == m.cursor {
			row = cursorStyle.Render("> " + row)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("↑↓ navigate · enter select · esc cancel"))
	return strings.Join(lines, "\n")
}
