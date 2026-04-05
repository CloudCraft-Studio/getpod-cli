package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// EnvPickerModal lets the user pick a single environment from the selected workspace.
// Environments containing "prod" in their name display a ⚠ warning.
type EnvPickerModal struct {
	items  []envItem
	cursor int
	styles Styles
}

type envItem struct {
	name       string
	awsAccount string
	awsRegion  string
	isProd     bool
}

// NewEnvPickerModal builds the list from the workspace's contexts in cfg.
func NewEnvPickerModal(cfg *config.Config, client, workspace string, styles Styles) *EnvPickerModal {
	cl := cfg.Clients[client]
	ws := cl.Workspaces[workspace]
	var items []envItem
	for name, ctx := range ws.Contexts {
		var awsAccount, awsRegion string
		// ContextConfig = map[string]map[string]string (plugin → key/value)
		for _, pluginVals := range ctx {
			if v, ok := pluginVals["aws_account"]; ok && awsAccount == "" {
				awsAccount = v
			}
			if v, ok := pluginVals["aws_region"]; ok && awsRegion == "" {
				awsRegion = v
			}
		}
		items = append(items, envItem{
			name:       name,
			awsAccount: awsAccount,
			awsRegion:  awsRegion,
			isProd:     strings.Contains(strings.ToLower(name), "prod"),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	return &EnvPickerModal{items: items, styles: styles}
}

func (m *EnvPickerModal) Title() string { return "Select Environment" }
func (m *EnvPickerModal) Init() tea.Cmd { return nil }

func (m *EnvPickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, func() tea.Msg { return EnvSelectedMsg{Env: name} }
			}
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m *EnvPickerModal) View() string {
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No environments configured for this workspace.")
	}
	prodStyle := lipgloss.NewStyle().Foreground(Warning400)
	cursorStyle := lipgloss.NewStyle().Foreground(Primary400)
	var lines []string
	for i, env := range m.items {
		// Build the indicator prefix separately so fmt.Sprintf only pads plain text.
		// Applying ANSI codes before %-Ns causes under-padding because escape bytes
		// count toward the width but are invisible in the terminal.
		indicator := "  "
		if env.isProd {
			indicator = prodStyle.Render("⚠") + " "
		}
		row := indicator + fmt.Sprintf("%-12s  AWS %-14s  %s", env.name, env.awsAccount, env.awsRegion)
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
