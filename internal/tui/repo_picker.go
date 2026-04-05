package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
)

// RepoPickerModal lets the user multi-select repositories.
// It fetches repos from the RepoPlugin at init time.
type RepoPickerModal struct {
	reg      *plugin.Registry
	items    []plugin.Repo
	selected map[string]bool
	cursor   int
	filter   string
	filterOn bool
	filtered []plugin.Repo
	loading  bool
	err      error
	styles   Styles
}

// NewRepoPickerModal creates the modal. preselected names will appear checked.
// reg may be nil in tests.
func NewRepoPickerModal(reg *plugin.Registry, preselected []string, styles Styles) *RepoPickerModal {
	sel := make(map[string]bool, len(preselected))
	for _, r := range preselected {
		sel[r] = true
	}
	return &RepoPickerModal{reg: reg, selected: sel, styles: styles, loading: reg != nil}
}

func (m *RepoPickerModal) Title() string { return "Select Repositories" }

func (m *RepoPickerModal) Init() tea.Cmd {
	if m.reg == nil {
		return nil
	}
	return m.fetchReposCmd()
}

func (m *RepoPickerModal) fetchReposCmd() tea.Cmd {
	reg := m.reg
	return func() tea.Msg {
		ctx := context.Background()
		for _, name := range reg.ActivePlugins() {
			p, ok := reg.Get(name)
			if !ok {
				continue
			}
			rp, ok := p.(plugin.RepoPlugin)
			if !ok {
				continue
			}
			repos, err := rp.ListRepos(ctx)
			if err != nil {
				return ReposFetchedMsg{Err: err}
			}
			return ReposFetchedMsg{Repos: repos}
		}
		return ReposFetchedMsg{} // no repo plugin configured
	}
}

func (m *RepoPickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ReposFetchedMsg:
		m.loading = false
		m.err = msg.Err
		m.items = msg.Repos
		m.applyFilter()

	case tea.KeyMsg:
		if m.filterOn {
			return m.handleFilterKey(msg)
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case " ":
			if m.cursor < len(m.filtered) {
				name := m.filtered[m.cursor].Name
				m.selected[name] = !m.selected[name]
			}
		case "/":
			m.filterOn = true
		case "enter":
			var repos []string
			for _, r := range m.items {
				if m.selected[r.Name] {
					repos = append(repos, r.Name)
				}
			}
			return m, func() tea.Msg { return ReposSelectedMsg{Repos: repos} }
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m *RepoPickerModal) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterOn = false
		m.filter = ""
		m.applyFilter()
	case "backspace":
		runes := []rune(m.filter)
		if len(runes) > 0 {
			m.filter = string(runes[:len(runes)-1])
			m.applyFilter()
		}
	case "enter":
		m.filterOn = false
	default:
		if len(msg.Runes) == 1 {
			m.filter += string(msg.Runes)
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *RepoPickerModal) applyFilter() {
	m.cursor = 0
	if m.filter == "" {
		m.filtered = m.items[:len(m.items):len(m.items)]
		return
	}
	query := strings.ToLower(m.filter)
	m.filtered = nil
	for _, r := range m.items {
		if strings.Contains(strings.ToLower(r.Name), query) {
			m.filtered = append(m.filtered, r)
		}
	}
}

func (m *RepoPickerModal) View() string {
	if m.loading {
		return m.styles.Placeholder.Render("Loading repositories...")
	}
	if m.err != nil {
		return m.styles.Placeholder.Render(fmt.Sprintf("Error: %v", m.err))
	}
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No repositories found. Configure a repo plugin.")
	}

	checkboxSelected := lipgloss.NewStyle().Foreground(Success400)
	cursorStyle := lipgloss.NewStyle().Foreground(Primary400)

	var lines []string
	if m.filterOn {
		lines = append(lines, m.styles.HelpKey.Render("/")+" "+m.styles.Paragraph.Render(m.filter+"_"))
	}

	for i, r := range m.filtered {
		checkbox := m.styles.Muted.Render("[ ]")
		if m.selected[r.Name] {
			checkbox = checkboxSelected.Render("[x]")
		}
		age := repoAge(r.UpdatedAt)
		row := fmt.Sprintf("%s  %-30s  %-10s  %-8s  %s", checkbox, r.Name, r.Source, r.Language, age)
		if i == m.cursor {
			row = cursorStyle.Render("> " + row)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("space toggle · enter confirm · esc cancel"))
	return strings.Join(lines, "\n")
}

func repoAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours())/24)
	}
}
