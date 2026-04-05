package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
)

const (
	AppTitle = "getpod v0.1.0"
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
	width     int
	height    int
	clientIdx int
	view      View

	// Footer hints
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
		// Continue with empty state
		st = &state.State{}
	}
	a.st = st

	// Determine active client
	if a.st.ActiveClient != "" {
		// Find index of active client
		idx := 0
		for name := range a.cfg.Clients {
			if name == a.st.ActiveClient {
				a.clientIdx = idx
				break
			}
			idx++
		}
	}

	// Update footer hints
	a.updateFooterHints()

	return tea.Batch(
		tea.EnterAltScreen,
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
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

	header := a.renderHeader()
	nav := a.renderNav()
	content := a.renderContent()
	footer := a.renderFooter()

	// Calculate heights
	headerHeight := lipgloss.Height(header)
	navHeight := lipgloss.Height(nav)
	footerHeight := lipgloss.Height(footer)

	availableHeight := a.height - headerHeight - navHeight - footerHeight - 2

	if availableHeight < 0 {
		availableHeight = 0
	}

	content = a.styles.ContentArea.
		Height(availableHeight).
		Render(content)

	return a.styles.AppContainer.
		Width(a.width).
		Height(a.height).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			nav,
			content,
			footer,
		))
}

func (a *App) renderHeader() string {
	clientName, _ := a.getActiveClient()

	// Top bar: App title + planning tool + issue count
	topBar := a.styles.TopBar.Width(a.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			a.styles.Title.Render(AppTitle),
			"          ",
			a.styles.Subtitle.Render(fmt.Sprintf("%s · %d issues", a.getPlanningTool(), a.getIssueCount())),
		),
	)

	// Client tabs
	clientTabs := a.styles.ClientTabActive.Render("• " + clientName)

	return lipgloss.JoinVertical(lipgloss.Left, topBar, clientTabs)
}

func (a *App) renderNav() string {
	tabs := []string{
		"[1] Issues",
		"[2] PRs",
		"[3] Status",
	}

	var renderedTabs []string
	for i, tab := range tabs {
		style := a.styles.NavTab
		if View(i) == a.view {
			style = a.styles.NavTabActive
		}
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	return lipgloss.NewStyle().
		Width(a.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...))
}

func (a *App) renderContent() string {
	switch a.view {
	case ViewIssues:
		return "Issues view (placeholder) - Press 'q' to quit\n\n• No plugin configured\n• Press 1, 2, 3 to switch views\n• Tab to switch clients"
	case ViewPRs:
		return "PRs view (coming soon)"
	case ViewStatus:
		return "Status view (coming soon)"
	default:
		return "Unknown view"
	}
}

func (a *App) renderFooter() string {
	if len(a.footerHints) == 0 {
		return a.styles.Footer.Width(a.width).Render("")
	}

	var hints []string
	for i, hint := range a.footerHints {
		if i > 0 {
			hints = append(hints, "·")
		}
		hints = append(hints, hint)
	}

	return a.styles.Footer.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Left, hints...),
	)
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
		a.footerHints = []string{"↑↓ Nav", "⏎ Open", "/ Filter", "tab Client"}
	case ViewPRs, ViewStatus:
		a.footerHints = []string{"↑↓ Nav", "⏎ Open", "esc Back", "tab Client"}
	}
}
