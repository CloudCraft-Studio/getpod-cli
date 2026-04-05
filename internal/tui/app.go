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

	return tea.Batch(
		tea.EnterAltScreen,
	)
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

	// Build the UI with a bordered container
	header := a.renderHeader()
	nav := a.renderNav()
	content := a.renderContent()
	footer := a.renderFooter()

	// Calculate available height for content
	headerHeight := lipgloss.Height(header)
	navHeight := lipgloss.Height(nav)
	footerHeight := lipgloss.Height(footer)

	// Account for border (2 lines), padding, and spacing
	availableHeight := a.height - headerHeight - navHeight - footerHeight - 6

	if availableHeight < 3 {
		availableHeight = 3
	}

	content = a.styles.ContentArea.
		Width(a.width - 4).
		Height(availableHeight).
		Render(content)

	// Combine all components
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		nav,
		content,
		footer,
	)

	// Add main container with border
	container := lipgloss.NewStyle().
		BorderStyle(a.styles.BorderRounded).
		BorderForeground(Surface700).
		Background(Surface950).
		Width(a.width).
		Height(a.height).
		Render(body)

	return container
}

func (a *App) renderHeader() string {
	clientName, _ := a.getActiveClient()

	// Brand title
	brand := a.styles.BrandText.Render(AppTitle + " " + AppVersion)

	// Planning tool and issue count
	planningTool := a.getPlanningTool()
	issueCount := a.getIssueCount()

	var toolBadge string
	if planningTool != "None" {
		toolBadge = a.styles.Badge.
			Background(Primary500).
			Foreground(Surface950).
			Render(planningTool)
	} else {
		toolBadge = a.styles.Badge.Render("No plugin")
	}

	issueBadge := a.styles.Muted.Render(fmt.Sprintf("%d issues", issueCount))

	rightInfo := lipgloss.JoinHorizontal(
		lipgloss.Left,
		toolBadge,
		"  ",
		issueBadge,
	)

	// Top bar with brand and info
	topBarWidth := a.width - 4 // Account for padding
	spacer := strings.Repeat(" ", max(0, topBarWidth-lipgloss.Width(brand)-lipgloss.Width(rightInfo)-2))

	topBar := a.styles.TopBar.
		Width(topBarWidth).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, brand, spacer, rightInfo))

	// Client tabs
	clientTab := a.styles.ClientTabActive.Render("● " + clientName)

	return lipgloss.JoinVertical(lipgloss.Left, topBar, clientTab)
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

	return lipgloss.NewStyle().
		Width(a.width-4).
		Padding(1, 0).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...))
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

	// Check if plugin is configured
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
		return a.styles.Footer.Width(a.width - 4).Render("")
	}

	var hints []string
	for _, hint := range a.footerHints {
		// Split on space to style key and description
		parts := strings.SplitN(hint, " ", 2)
		if len(parts) == 2 {
			hints = append(hints, a.styles.HelpKey.Render(parts[0])+" "+a.styles.HelpDesc.Render(parts[1]))
		} else {
			hints = append(hints, a.styles.HelpDesc.Render(hint))
		}
	}

	hintText := strings.Join(hints, "  •  ")

	return a.styles.Footer.
		Width(a.width - 4).
		Render(hintText)
}

// Helper methods
func (a *App) getActiveClient() (string, config.ClientConfig) {
	var name string
	var client config.ClientConfig

	idx := 0
	for n, c := range a.cfg.Clients {
		if idx == a.clientIdx {
			name = n
			client = c
			break
		}
		idx++
	}

	return name, client
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
		a.footerHints = []string{"↑↓ Navigate", "⏎ Open", "/ Filter", "tab Switch Client", "q Quit"}
	case ViewPRs, ViewStatus:
		a.footerHints = []string{"↑↓ Navigate", "⏎ Open", "esc Back", "tab Switch Client", "q Quit"}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
