package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

// IssueDetailModel renders the detail view: header, metadata, description,
// work context selectors, and action list.
type IssueDetailModel struct {
	db         *store.Store
	issue      store.IssueRecord
	descOffset int
	cfg        *config.Config
	styles     Styles
	width      int
	height     int

	// Action feedback — shown temporarily after an action completes.
	actionMsg     string
	actionSuccess bool
}

// NewIssueDetailModel constructs the detail view for the given issue.
func NewIssueDetailModel(db *store.Store, issue store.IssueRecord, cfg *config.Config, styles Styles) *IssueDetailModel {
	return &IssueDetailModel{db: db, issue: issue, cfg: cfg, styles: styles}
}

func (m *IssueDetailModel) Init() tea.Cmd { return nil }

func (m *IssueDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ReposSelectedMsg:
		m.issue.Repos = msg.Repos
		return m, m.persistWorkContextCmd()

	case WorkspaceSelectedMsg:
		m.issue.Workspace = msg.Workspace
		m.issue.Environment = "" // reset env when workspace changes
		return m, m.persistWorkContextCmd()

	case EnvSelectedMsg:
		m.issue.Environment = msg.Env
		return m, m.persistWorkContextCmd()

	// ── Action result messages ─────────────────────────────────────────
	case BranchCreatedMsg:
		if msg.Err != nil {
			m.actionMsg = fmt.Sprintf("Branch failed: %v", msg.Err)
			m.actionSuccess = false
		} else {
			m.actionMsg = fmt.Sprintf("Branch created: %s", msg.Branch)
			m.actionSuccess = true
		}
	case CommitPushedMsg:
		if msg.Err != nil {
			m.actionMsg = fmt.Sprintf("Commit failed: %v", msg.Err)
			m.actionSuccess = false
		} else {
			m.actionMsg = fmt.Sprintf("Committed + pushed: %s", msg.Message)
			m.actionSuccess = true
		}
	case PRCreatedMsg:
		if msg.Err != nil {
			m.actionMsg = fmt.Sprintf("PR failed: %v", msg.Err)
			m.actionSuccess = false
		} else {
			m.actionMsg = fmt.Sprintf("PR created: %s", msg.URL)
			m.actionSuccess = true
		}
	case CommentAddedMsg:
		if msg.Err != nil {
			m.actionMsg = fmt.Sprintf("Comment failed: %v", msg.Err)
			m.actionSuccess = false
		} else {
			m.actionMsg = "Comment added"
			m.actionSuccess = true
		}
	case StatusChangedMsg:
		if msg.Err != nil {
			m.actionMsg = fmt.Sprintf("Status change failed: %v", msg.Err)
			m.actionSuccess = false
		} else {
			m.issue.Status = msg.NewStatus
			m.actionMsg = fmt.Sprintf("Status changed to: %s", msg.NewStatus)
			m.actionSuccess = true
		}
	case PlanStartedMsg:
		if msg.Err != nil {
			m.actionMsg = fmt.Sprintf("Plan failed: %v", msg.Err)
			m.actionSuccess = false
		} else {
			m.actionMsg = "AI planner launched"
			m.actionSuccess = true
		}

	case tea.KeyMsg:
		// Clear action message on any keypress
		m.actionMsg = ""

		switch msg.String() {
		case "up", "k":
			if m.descOffset > 0 {
				m.descOffset--
			}
		case "down", "j":
			descLines := len(strings.Split(m.issue.Description, "\n"))
			if m.descOffset < descLines-1 {
				m.descOffset++
			}

		// ── Work context keybindings ──────────────────────────────────
		case "w":
			return m, func() tea.Msg { return OpenRepoPickerMsg{} }
		case "x":
			return m, func() tea.Msg { return OpenWorkspacePickerMsg{} }
		case "e":
			if m.issue.Workspace != "" {
				return m, func() tea.Msg { return OpenEnvPickerMsg{} }
			}

		// ── Action keybindings ────────────────────────────────────────
		case "b":
			if len(m.issue.Repos) > 0 {
				return m, func() tea.Msg { return OpenBranchConfirmMsg{} }
			}
		case "c":
			return m, func() tea.Msg { return OpenCommitModalMsg{} }
		case "r":
			if len(m.issue.Repos) > 0 {
				return m, func() tea.Msg { return OpenPRModalMsg{} }
			}
		case "m":
			return m, func() tea.Msg { return OpenCommentModalMsg{} }
		case "s":
			return m, func() tea.Msg { return OpenStatusPickerMsg{} }
		case "p":
			if m.IsReady() {
				return m, func() tea.Msg { return OpenPlanAIMsg{} }
			}
		}
	}
	return m, nil
}

func (m *IssueDetailModel) persistWorkContextCmd() tea.Cmd {
	if m.db == nil {
		return nil
	}
	issue := m.issue
	db := m.db
	return func() tea.Msg {
		_ = db.UpdateWorkContext(
			context.Background(),
			issue.ID, issue.Repos, issue.Workspace, issue.Environment,
		)
		return nil
	}
}

// IsReady reports whether repos, workspace, and environment are all set.
func (m *IssueDetailModel) IsReady() bool {
	return len(m.issue.Repos) > 0 && m.issue.Workspace != "" && m.issue.Environment != ""
}

// missingContext returns a comma-separated list of missing work context fields.
func (m *IssueDetailModel) missingContext() string {
	var missing []string
	if len(m.issue.Repos) == 0 {
		missing = append(missing, "repos")
	}
	if m.issue.Workspace == "" {
		missing = append(missing, "workspace")
	}
	if m.issue.Environment == "" {
		missing = append(missing, "environment")
	}
	return strings.Join(missing, ", ")
}

// View renders all five sections of the detail view.
func (m *IssueDetailModel) View() string {
	sections := []string{
		m.renderHeader(),
		"",
		m.renderMetadata(),
		"",
		m.styles.Subtitle.Render("Description"),
		m.renderDescription(),
		"",
		m.styles.Subtitle.Render("Work Context"),
		m.renderWorkContext(),
		"",
		m.styles.Subtitle.Render("Actions"),
		m.renderActions(),
	}
	if m.actionMsg != "" {
		sections = append(sections, "", m.renderActionFeedback())
	}
	return strings.Join(sections, "\n")
}

func (m *IssueDetailModel) renderActionFeedback() string {
	if m.actionSuccess {
		return lipgloss.NewStyle().Foreground(Success400).Render("✓ " + m.actionMsg)
	}
	return lipgloss.NewStyle().Foreground(Danger400).Render("✗ " + m.actionMsg)
}

func (m *IssueDetailModel) renderHeader() string {
	keyEnv := m.issue.Key
	if m.issue.Environment != "" {
		keyEnv += " · " + lipgloss.NewStyle().Foreground(Primary400).Render("●"+m.issue.Environment)
	}
	return m.styles.Title.Render(keyEnv) + "\n" + m.styles.Paragraph.Render(m.issue.Title)
}

func (m *IssueDetailModel) renderMetadata() string {
	parts := []string{"Status: " + m.issue.Status}
	if m.issue.Priority != "" {
		parts = append(parts, "Priority: "+m.issue.Priority)
	}
	if len(m.issue.Labels) > 0 {
		parts = append(parts, "Labels: "+strings.Join(m.issue.Labels, ", "))
	}
	return m.styles.Muted.Render(strings.Join(parts, "  ·  "))
}

func (m *IssueDetailModel) renderDescription() string {
	if m.issue.Description == "" {
		return m.styles.Muted.Render("(no description)")
	}
	lines := strings.Split(m.issue.Description, "\n")
	const maxVisible = 6
	end := min(m.descOffset+maxVisible, len(lines))
	text := strings.Join(lines[m.descOffset:end], "\n")
	if len(lines) > maxVisible {
		text += "\n" + m.styles.Muted.Render(fmt.Sprintf("(%d/%d lines, ↑↓ to scroll)", m.descOffset+1, len(lines)))
	}
	return m.styles.Paragraph.Render(text)
}

func (m *IssueDetailModel) renderWorkContext() string {
	var lines []string

	// Repos
	repoDisplay := m.styles.Muted.Render("(none)")
	if len(m.issue.Repos) > 0 {
		d := m.issue.Repos[0]
		if len(m.issue.Repos) > 1 {
			d += fmt.Sprintf(" (+%d more)", len(m.issue.Repos)-1)
		}
		repoDisplay = m.styles.Paragraph.Render(d)
	}
	lines = append(lines, m.styles.HelpKey.Render("[w]")+" Repositories  "+repoDisplay)

	// Workspace
	wsDisplay := m.styles.Muted.Render("(none)")
	if m.issue.Workspace != "" {
		wsDisplay = m.styles.Paragraph.Render(m.issue.Workspace)
	}
	lines = append(lines, m.styles.HelpKey.Render("[x]")+" Workspace     "+wsDisplay)

	// Environment
	envDisplay := m.styles.Muted.Render("(none)")
	if m.issue.Environment != "" {
		envText := m.issue.Environment
		awsAccount, awsRegion := m.envAWS(m.issue.Workspace, m.issue.Environment)
		if awsAccount != "" {
			envText += " · AWS " + awsAccount
		}
		if awsRegion != "" {
			envText += " · " + awsRegion
		}
		envDisplay = m.styles.Paragraph.Render(envText)
	}
	lines = append(lines, m.styles.HelpKey.Render("[e]")+" Environment   "+envDisplay)

	// Ready indicator
	lines = append(lines, "")
	if m.IsReady() {
		awsAccount, _ := m.envAWS(m.issue.Workspace, m.issue.Environment)
		ready := fmt.Sprintf("✓ Ready: %d repos · %s · %s", len(m.issue.Repos), m.issue.Workspace, m.issue.Environment)
		if awsAccount != "" {
			ready += " · AWS " + awsAccount
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(Success400).Render(ready))
	} else {
		lines = append(lines, m.styles.Muted.Render("○ Not ready: missing "+m.missingContext()))
	}

	return strings.Join(lines, "\n")
}

// envAWS looks up aws_account and aws_region for the given workspace/env from config.
// ContextConfig is map[string]map[string]string (plugin → key/value), so we search
// all plugin sub-maps for the aws_account and aws_region keys.
func (m *IssueDetailModel) envAWS(workspace, env string) (account, region string) {
	cl, ok := m.cfg.Clients[m.issue.Client]
	if !ok {
		return
	}
	ws, ok := cl.Workspaces[workspace]
	if !ok {
		return
	}
	ctx, ok := ws.Contexts[env]
	if !ok {
		return
	}
	for _, pluginVals := range ctx {
		if v, found := pluginVals["aws_account"]; found && account == "" {
			account = v
		}
		if v, found := pluginVals["aws_region"]; found && region == "" {
			region = v
		}
	}
	return
}

func (m *IssueDetailModel) renderActions() string {
	type action struct {
		key     string
		label   string
		enabled bool
		reason  string
	}

	hasRepos := len(m.issue.Repos) > 0
	ready := m.IsReady()

	actions := []action{
		{"p", "Plan with AI", ready, "requires repos + workspace + env"},
		{"b", "Create branch", hasRepos, "requires repos"},
		{"c", "Commit + Push", true, ""},
		{"r", "Create PR", hasRepos, "requires repos"},
		{"m", "Comment", true, ""},
		{"s", "Change status", true, ""},
	}

	var lines []string
	for _, a := range actions {
		key := m.styles.HelpKey.Render("[" + a.key + "]")
		var label string
		if a.enabled {
			label = m.styles.Paragraph.Render(a.label)
		} else {
			label = m.styles.Muted.Render(a.label) + "  " + m.styles.Muted.Render("("+a.reason+")")
		}
		lines = append(lines, key+"  "+label)
	}
	return strings.Join(lines, "\n")
}
