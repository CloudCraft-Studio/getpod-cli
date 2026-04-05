package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
)

const (
	AppTitle   = "GETPOD"
	AppVersion = "v0.1.0"
)

type View int

const (
	ViewIssues View = iota
	ViewPRs
	ViewStatus
)

type FocusArea int

const (
	FocusClients FocusArea = iota
	FocusContent
)

type App struct {
	cfg    *config.Config
	reg    *plugin.Registry
	st     *state.State
	styles Styles

	// UI state
	width     int
	height    int
	clientIdx int
	view      View
	focus     FocusArea
}

func NewApp(cfg *config.Config, reg *plugin.Registry) *App {
	return &App{
		cfg:    cfg,
		reg:    reg,
		styles: DefaultStyles(),
		view:   ViewIssues,
		focus:  FocusContent, // Start with content focused
	}
}

func (a *App) Init() tea.Cmd {
	// Load state
	st, err := state.Load()
	if err != nil {
		st = &state.State{}
	}
	a.st = st

	// Determine active client
	if a.st.ActiveClient != "" {
		idx := 0
		clientNames := a.getSortedClientNames()
		for i, name := range clientNames {
			if name == a.st.ActiveClient {
				idx = i
				break
			}
		}
		a.clientIdx = idx
	}

	return tea.EnterAltScreen
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "esc":
			// ESC switches between clients and content
			if a.focus == FocusContent {
				a.focus = FocusClients
			}
			return a, nil
		case "tab":
			if a.focus == FocusClients {
				a.nextClient()
			}
			return a, nil
		case "shift+tab":
			if a.focus == FocusClients {
				a.prevClient()
			}
			return a, nil
		case "enter":
			if a.focus == FocusClients {
				// Enter on client switches to content
				a.focus = FocusContent
			}
			return a, nil
		case "1":
			if a.focus == FocusContent {
				a.view = ViewIssues
			}
			return a, nil
		case "2":
			if a.focus == FocusContent {
				a.view = ViewPRs
			}
			return a, nil
		case "3":
			if a.focus == FocusContent {
				a.view = ViewStatus
			}
			return a, nil
		}
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	}

	return a, nil
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	header := a.renderHeader()
	clientButtons := a.renderClientButtons()
	nav := a.renderNav()
	content := a.renderContent()

	// Content area with border
	contentPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Surface700).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, nav, "", content))

	// Footer with border
	footerPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Surface700).
		Padding(0, 2).
		Render(a.renderFooter())

	// Stack sections
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		clientButtons,
		"",
		contentPanel,
		"",
		footerPanel,
	)

	// Main container
	container := lipgloss.NewStyle().
		Border(a.styles.BorderRounded).
		BorderForeground(Surface700).
		Padding(1, 2).
		Render(body)

	return container
}

func (a *App) renderHeader() string {
	brand := a.styles.BrandText.Render(AppTitle + " " + AppVersion)

	planningTool := a.getPlanningTool()
	issueCount := a.getIssueCount()

	var toolInfo string
	if planningTool != "None" {
		toolBadge := a.styles.Badge.Render(" " + planningTool + " ")
		issueInfo := a.styles.Muted.Render(fmt.Sprintf("%d issues", issueCount))
		toolInfo = toolBadge + " " + issueInfo
	} else {
		toolInfo = a.styles.Muted.Render("No plugin configured")
	}

	spacerWidth := a.width - lipgloss.Width(brand) - lipgloss.Width(toolInfo) - 8
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	spacer := strings.Repeat(" ", spacerWidth)

	return brand + spacer + toolInfo
}

func (a *App) renderClientButtons() string {
	var buttons []string

	clientNames := a.getSortedClientNames()

	for idx, name := range clientNames {
		client := a.cfg.Clients[name]
		displayName := client.DisplayName
		if displayName == "" {
			displayName = name
		}

		var button string
		if idx == a.clientIdx {
			// Active button - cyan border
			button = a.styles.ClientButtonActive.Render(" " + displayName + " ")
		} else {
			// Inactive button - gray border
			button = a.styles.ClientButton.Render(" " + displayName + " ")
		}

		buttons = append(buttons, button)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, buttons...)
}

func (a *App) renderNav() string {
	tabs := []struct {
		label string
		view  View
	}{
		{"[1] Issues", ViewIssues},
		{"[2] PRs", ViewPRs},
		{"[3] Status", ViewStatus},
	}

	var renderedTabs []string
	for _, tab := range tabs {
		style := a.styles.NavTab
		if tab.view == a.view {
			style = a.styles.NavTabActive
		}
		renderedTabs = append(renderedTabs, style.Render(tab.label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...)
}

func (a *App) renderContent() string {
	switch a.view {
	case ViewIssues:
		return a.renderIssuesView()
	case ViewPRs:
		return a.styles.Placeholder.Render("Pull requests view (coming soon)")
	case ViewStatus:
		return a.styles.Placeholder.Render("Status view (coming soon)")
	default:
		return "Unknown view"
	}
}

func (a *App) renderIssuesView() string {
	var lines []string

	lines = append(lines, a.styles.Title.Render("📋 Issues"))
	lines = append(lines, "")

	planningTool := a.getPlanningTool()
	if planningTool == "None" {
		lines = append(lines, a.styles.Muted.Render("No planning tool configured"))
		lines = append(lines, "")
		lines = append(lines, a.styles.Paragraph.Render("Configure a plugin (Jira, Linear, etc.) to see your issues here."))
		lines = append(lines, "")
		lines = append(lines, a.styles.Muted.Render("Example: Edit ~/.getpod/config.yaml"))
	} else {
		lines = append(lines, a.styles.Placeholder.Render(fmt.Sprintf("Loading issues from %s...", planningTool)))
		lines = append(lines, "")
		lines = append(lines, a.styles.Muted.Render("Plugin integration coming soon"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (a *App) renderFooter() string {
	var hints []string

	if a.focus == FocusClients {
		// Hints for client selection
		hints = []string{
			a.styles.HelpKey.Render("tab") + " " + a.styles.HelpDesc.Render("Switch"),
			a.styles.HelpKey.Render("⏎") + " " + a.styles.HelpDesc.Render("Select"),
			a.styles.HelpKey.Render("q") + " " + a.styles.HelpDesc.Render("Quit"),
		}
	} else {
		// Hints for content (varies by view)
		switch a.view {
		case ViewIssues:
			hints = []string{
				a.styles.HelpKey.Render("↑↓") + " " + a.styles.HelpDesc.Render("Navigate"),
				a.styles.HelpKey.Render("⏎") + " " + a.styles.HelpDesc.Render("Open"),
				a.styles.HelpKey.Render("/") + " " + a.styles.HelpDesc.Render("Filter"),
				a.styles.HelpKey.Render("ESC") + " " + a.styles.HelpDesc.Render("Salir (Ir a clientes)"),
			}
		case ViewPRs, ViewStatus:
			hints = []string{
				a.styles.HelpKey.Render("↑↓") + " " + a.styles.HelpDesc.Render("Navigate"),
				a.styles.HelpKey.Render("⏎") + " " + a.styles.HelpDesc.Render("Open"),
				a.styles.HelpKey.Render("ESC") + " " + a.styles.HelpDesc.Render("Back"),
			}
		}
	}

	hintText := strings.Join(hints, "  •  ")

	return lipgloss.NewStyle().
		Foreground(Content400).
		Padding(0, 2).
		Render(hintText)
}

// Helper methods
func (a *App) getSortedClientNames() []string {
	var clientNames []string
	for name := range a.cfg.Clients {
		clientNames = append(clientNames, name)
	}

	// Sort alphabetically
	for i := 0; i < len(clientNames); i++ {
		for j := i + 1; j < len(clientNames); j++ {
			if clientNames[i] > clientNames[j] {
				clientNames[i], clientNames[j] = clientNames[j], clientNames[i]
			}
		}
	}

	return clientNames
}

func (a *App) getActiveClient() (string, config.ClientConfig) {
	clientNames := a.getSortedClientNames()

	if a.clientIdx < len(clientNames) {
		name := clientNames[a.clientIdx]
		return name, a.cfg.Clients[name]
	}

	return "", config.ClientConfig{}
}

func (a *App) updateActiveClient() {
	name, _ := a.getActiveClient()
	if a.st == nil {
		a.st = &state.State{}
	}
	a.st.UseClient(name)
}

func (a *App) nextClient() {
	clientCount := len(a.cfg.Clients)
	if clientCount == 0 {
		return
	}
	a.clientIdx = (a.clientIdx + 1) % clientCount
	a.updateActiveClient()
}

func (a *App) prevClient() {
	clientCount := len(a.cfg.Clients)
	if clientCount == 0 {
		return
	}
	a.clientIdx = (a.clientIdx + clientCount - 1) % clientCount
	a.updateActiveClient()
}

func (a *App) getPlanningTool() string {
	_, client := a.getActiveClient()
	for pluginName := range client.Plugins {
		return pluginName
	}
	return "None"
}

func (a *App) getIssueCount() int {
	return 0 // Placeholder
}
