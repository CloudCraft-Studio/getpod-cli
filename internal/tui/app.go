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

type App struct {
	cfg    *config.Config
	reg    *plugin.Registry
	st     *state.State
	styles Styles

	// UI state
	width       int
	height      int
	clientIdx   int
	view        View
	footerHints []string
}

func NewApp(cfg *config.Config, reg *plugin.Registry) *App {
	return &App{
		cfg:    cfg,
		reg:    reg,
		styles: DefaultStyles(),
		view:   ViewIssues,
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
		for name := range a.cfg.Clients {
			if name == a.st.ActiveClient {
				a.clientIdx = idx
				break
			}
			idx++
		}
	}

	a.updateFooterHints()

	return tea.EnterAltScreen
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return a, tea.Quit
		case "tab":
			a.nextClient()
			return a, nil
		case "shift+tab":
			a.prevClient()
			return a, nil
		case "1":
			a.view = ViewIssues
			a.updateFooterHints()
			return a, nil
		case "2":
			a.view = ViewPRs
			a.updateFooterHints()
			return a, nil
		case "3":
			a.view = ViewStatus
			a.updateFooterHints()
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

	// Build sections
	header := a.renderHeader()
	clientTabs := a.renderClientTabsHeader()
	nav := a.renderNav()
	content := a.renderContent()
	footer := a.renderFooter()

	// Content panel (nav + content + footer) wrapped in active tab border
	panelContent := lipgloss.JoinVertical(lipgloss.Left, nav, "", content, "", footer)

	// Wrap content in active tab border
	activeTabPanel := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "",
			TopRight:    "",
			BottomLeft:  "╰",
			BottomRight: "╯",
		}).
		BorderForeground(Primary400).
		Padding(1, 2).
		Render(panelContent)

	// Stack everything
	body := lipgloss.JoinVertical(lipgloss.Left, header, clientTabs, activeTabPanel)

	// Outer container
	container := lipgloss.NewStyle().
		Border(a.styles.BorderRounded).
		BorderForeground(Surface700).
		Padding(1, 2).
		Render(body)

	return container
}

func (a *App) renderHeader() string {
	// Line 1: Brand and metadata
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

	// Calculate spacing
	spacerWidth := a.width - lipgloss.Width(brand) - lipgloss.Width(toolInfo) - 8
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	spacer := strings.Repeat(" ", spacerWidth)

	line1 := brand + spacer + toolInfo

	return line1
}

func (a *App) renderClientTabsHeader() string {
	var tabs []string

	// Get sorted client names for consistent order
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

	// Render tabs
	for idx, name := range clientNames {
		client := a.cfg.Clients[name]
		displayName := client.DisplayName
		if displayName == "" {
			displayName = name
		}

		var tab string
		if idx == a.clientIdx {
			// Active tab - only top border, will connect to panel below
			tab = lipgloss.NewStyle().
				Foreground(Primary400).
				Bold(true).
				Border(lipgloss.Border{
					Top:         "─",
					Bottom:      "",
					Left:        "│",
					Right:       "│",
					TopLeft:     "╭",
					TopRight:    "╮",
					BottomLeft:  "",
					BottomRight: "",
				}).
				BorderForeground(Primary400).
				Render(" " + displayName + " ")
		} else {
			// Inactive tab - complete small box
			tab = lipgloss.NewStyle().
				Foreground(Content400).
				Border(lipgloss.Border{
					Top:         "─",
					Bottom:      "─",
					Left:        "│",
					Right:       "│",
					TopLeft:     "╭",
					TopRight:    "╮",
					BottomLeft:  "╰",
					BottomRight: "╯",
				}).
				BorderForeground(Surface700).
				Render(" " + displayName + " ")
		}

		tabs = append(tabs, tab)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
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
	if len(a.footerHints) == 0 {
		return ""
	}

	var hints []string
	for _, hint := range a.footerHints {
		parts := strings.SplitN(hint, " ", 2)
		if len(parts) == 2 {
			hints = append(hints, a.styles.HelpKey.Render(parts[0])+" "+a.styles.HelpDesc.Render(parts[1]))
		} else {
			hints = append(hints, a.styles.HelpDesc.Render(hint))
		}
	}

	return strings.Join(hints, "  •  ")
}

// Helper methods
func (a *App) getActiveClient() (string, config.ClientConfig) {
	// Get sorted client names
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

func (a *App) updateFooterHints() {
	switch a.view {
	case ViewIssues:
		a.footerHints = []string{"↑↓ Navigate", "⏎ Open", "/ Filter", "tab Switch", "q Quit"}
	case ViewPRs, ViewStatus:
		a.footerHints = []string{"↑↓ Navigate", "⏎ Open", "esc Back", "tab Switch", "q Quit"}
	}
}
