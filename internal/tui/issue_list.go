package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

// IssueListModel renders a scrollable, filterable list of issues for the active client.
type IssueListModel struct {
	db       *store.Store
	reg      *plugin.Registry
	client   string
	items    []store.IssueRecord // full cached list
	filtered []store.IssueRecord // filtered view (same as items when filter is empty)
	cursor   int
	filter   string
	filterOn bool
	loading  bool
	err      error
	width    int
	height   int
	styles   Styles
}

// NewIssueListModel constructs an IssueListModel for the given client.
func NewIssueListModel(db *store.Store, reg *plugin.Registry, client string, styles Styles) *IssueListModel {
	return &IssueListModel{
		db:     db,
		reg:    reg,
		client: client,
		styles: styles,
	}
}

// Init kicks off a cache-or-fetch load. Sets loading=true immediately.
func (m *IssueListModel) Init() tea.Cmd {
	m.loading = true
	return m.loadCachedCmd()
}

func (m *IssueListModel) loadCachedCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if m.db != nil {
			issues, err := m.db.ListIssuesByClient(ctx, m.client)
			if err != nil {
				return IssuesFetchedMsg{Client: m.client, Err: err}
			}
			if len(issues) > 0 {
				return IssuesFetchedMsg{Client: m.client, Issues: issues}
			}
		}
		// No cache (or no db): fetch from plugin
		return m.fetchFromPlugin(ctx)
	}
}

func (m *IssueListModel) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return m.fetchFromPlugin(context.Background())
	}
}

// fetchFromPlugin uses the first PlanningPlugin found in the registry.
// Each client is expected to have exactly one planning plugin (Jira or Linear).
func (m *IssueListModel) fetchFromPlugin(ctx context.Context) tea.Msg {
	for _, name := range m.reg.ActivePlugins() {
		p, ok := m.reg.Get(name)
		if !ok {
			continue
		}
		pp, ok := p.(plugin.PlanningPlugin)
		if !ok {
			continue
		}
		issues, err := pp.ListIssues(ctx)
		if err != nil {
			return IssuesFetchedMsg{Client: m.client, Err: err}
		}
		now := time.Now().UTC()
		var records []store.IssueRecord
		for _, iss := range issues {
			ir := store.IssueRecord{
				ID:          iss.ID,
				Client:      m.client,
				Key:         iss.Key,
				Title:       iss.Title,
				Status:      iss.Status,
				Priority:    iss.Priority,
				Description: iss.Description,
				Labels:      iss.Labels,
				RawData:     iss.RawData,
				FetchedAt:   now,
			}
			if m.db == nil {
				records = append(records, ir)
			} else if upsertErr := m.db.UpsertIssue(ctx, ir); upsertErr != nil {
				slog.Warn("failed to upsert issue", "id", ir.ID, "error", upsertErr)
				records = append(records, ir) // use in-memory version if DB fails
			} else {
				// Re-read to pick up preserved work context
				if full, readErr := m.db.GetIssue(ctx, ir.ID); readErr == nil && full != nil {
					records = append(records, *full)
				} else {
					records = append(records, ir)
				}
			}
		}
		return IssuesFetchedMsg{Client: m.client, Issues: records}
	}
	return IssuesFetchedMsg{Client: m.client} // no planning plugin found
}

// Update handles messages. WindowSizeMsg is propagated by App before delegating keys.
func (m *IssueListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case IssuesFetchedMsg:
		if msg.Client == m.client {
			m.loading = false
			m.err = msg.Err
			m.items = msg.Issues
			m.applyFilter()
		}

	case tea.KeyMsg:
		if m.filterOn {
			return m.handleFilterKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *IssueListModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "/":
		m.filterOn = true
	case "r":
		m.loading = true
		return m, m.refreshCmd()
	case "enter":
		if m.cursor < len(m.filtered) {
			issue := m.filtered[m.cursor]
			return m, func() tea.Msg { return IssueSelectedMsg{Issue: issue} }
		}
	}
	return m, nil
}

func (m *IssueListModel) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// applyFilter rebuilds m.filtered from m.items using m.filter (case-insensitive).
// Always resets cursor to 0.
func (m *IssueListModel) applyFilter() {
	m.cursor = 0
	if m.filter == "" {
		m.filtered = m.items
		return
	}
	query := strings.ToLower(m.filter)
	m.filtered = nil
	for _, ir := range m.items {
		if strings.Contains(strings.ToLower(ir.Key), query) ||
			strings.Contains(strings.ToLower(ir.Title), query) ||
			strings.Contains(strings.ToLower(ir.Status), query) {
			m.filtered = append(m.filtered, ir)
		}
	}
}

// View renders the issue list.
func (m *IssueListModel) View() string {
	if m.loading {
		return m.styles.Placeholder.Render("Loading issues...")
	}
	if m.err != nil {
		return m.styles.Placeholder.Render(fmt.Sprintf("Error: %v", m.err))
	}
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No issues. Configure a planning plugin or press [r] to fetch.")
	}

	var lines []string

	if m.filterOn {
		lines = append(lines, m.styles.HelpKey.Render("/")+" "+m.styles.Paragraph.Render(m.filter+"_"))
	} else if m.filter != "" {
		lines = append(lines, m.styles.Muted.Render("filter: "+m.filter))
	}

	for i, ir := range m.filtered {
		lines = append(lines, m.renderRow(ir, i == m.cursor))
	}

	if len(m.filtered) == 0 && m.filter != "" {
		lines = append(lines, m.styles.Muted.Render("No matches for \""+m.filter+"\"  [Esc to clear]"))
	}

	return strings.Join(lines, "\n")
}

func (m *IssueListModel) renderRow(ir store.IssueRecord, selected bool) string {
	dot := issueStatusDot(ir.Status)
	key := fmt.Sprintf("%-12s", ir.Key)
	title := truncateStr(ir.Title, 38)

	envPart := fmt.Sprintf("%-6s", "")
	if ir.Environment != "" {
		envPart = lipgloss.NewStyle().Foreground(Primary400).Render(fmt.Sprintf("%-6s", "●"+ir.Environment))
	}

	priority := issuePriorityLabel(ir.Priority, m.styles)
	status := m.styles.Muted.Render(fmt.Sprintf("%-14s", ir.Status))

	row := fmt.Sprintf("%s %s  %-38s  %s  %-8s  %s", dot, key, title, envPart, priority, status)

	if selected {
		return lipgloss.NewStyle().Foreground(Primary400).Bold(true).Render("> " + row)
	}
	return "  " + row
}

func issueStatusDot(status string) string {
	switch strings.ToLower(status) {
	case "in progress", "in_progress":
		return lipgloss.NewStyle().Foreground(Primary400).Render("●")
	case "in review", "in_review":
		return lipgloss.NewStyle().Foreground(Warning400).Render("◐")
	case "done", "closed", "completed":
		return lipgloss.NewStyle().Foreground(Success400).Render("✓")
	default:
		return lipgloss.NewStyle().Foreground(Content500).Render("○")
	}
}

func issuePriorityLabel(priority string, s Styles) string {
	switch strings.ToLower(priority) {
	case "urgent", "critical":
		return s.BadgeDanger.Render("Urgent")
	case "high":
		return s.BadgeWarning.Render(" High ")
	case "medium", "med":
		return s.Muted.Render("Med  ")
	default:
		return s.Muted.Render("Low  ")
	}
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
