package tui

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/git"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
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

type appScreen int

const (
	screenIssueList appScreen = iota
	screenIssueDetail
)

// App is the root Bubbletea model. It owns routing between views, modal overlays,
// and propagates window size to sub-models.
type App struct {
	cfg    *config.Config
	reg    *plugin.Registry
	db     *store.Store
	st     *state.State
	styles Styles

	// UI state
	width     int
	height    int
	clientIdx int
	view      View
	focus     FocusArea

	// Issue navigation
	screen      appScreen
	issueList   *IssueListModel
	issueDetail *IssueDetailModel

	// Modal overlay — use hasModal alongside activeModal to avoid nil-interface issues
	activeModal Modal
	hasModal    bool
}

// NewApp constructs the App. db may be nil (graceful degradation: no caching).
func NewApp(cfg *config.Config, reg *plugin.Registry, db *store.Store) *App {
	return &App{
		cfg:    cfg,
		reg:    reg,
		db:     db,
		styles: DefaultStyles(),
		view:   ViewIssues,
		focus:  FocusContent,
	}
}

func (a *App) Init() tea.Cmd {
	st, err := state.Load()
	if err != nil {
		st = &state.State{}
	}
	a.st = st

	if a.st.ActiveClient != "" {
		clientNames := a.getSortedClientNames()
		for i, name := range clientNames {
			if name == a.st.ActiveClient {
				a.clientIdx = i
				break
			}
		}
	}

	a.initIssueList()

	return tea.Batch(
		tea.EnterAltScreen,
		a.issueList.Init(),
	)
}

// initIssueList creates a fresh IssueListModel for the current active client.
func (a *App) initIssueList() {
	name, _ := a.getActiveClient()
	a.issueList = NewIssueListModel(a.db, a.reg, name, a.styles)
	a.screen = screenIssueList
	a.issueDetail = nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		var cmds []tea.Cmd
		if a.issueList != nil {
			m, cmd := a.issueList.Update(msg)
			a.issueList = m.(*IssueListModel)
			cmds = append(cmds, cmd)
		}
		if a.issueDetail != nil {
			m, cmd := a.issueDetail.Update(msg)
			a.issueDetail = m.(*IssueDetailModel)
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	// ── Modal message routing ──────────────────────────────────────────────

	case ReposFetchedMsg:
		if a.hasModal {
			newModal, cmd := a.activeModal.Update(msg)
			a.activeModal = newModal.(Modal)
			return a, cmd
		}

	case ReposSelectedMsg, WorkspaceSelectedMsg, EnvSelectedMsg:
		a.hasModal = false
		a.activeModal = nil
		if a.issueDetail != nil {
			newModel, cmd := a.issueDetail.Update(msg)
			a.issueDetail = newModel.(*IssueDetailModel)
			return a, cmd
		}

	case ModalClosedMsg:
		a.hasModal = false
		a.activeModal = nil
		return a, nil

	case OpenRepoPickerMsg:
		var preselected []string
		if a.issueDetail != nil {
			preselected = a.issueDetail.issue.Repos
		}
		modal := NewRepoPickerModal(a.reg, preselected, a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	case OpenWorkspacePickerMsg:
		clientName, _ := a.getActiveClient()
		modal := NewWorkspacePickerModal(a.cfg, clientName, a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	case OpenEnvPickerMsg:
		ws := ""
		if a.issueDetail != nil {
			ws = a.issueDetail.issue.Workspace
		}
		if ws == "" {
			return a, nil // env picker requires a workspace — silently ignore
		}
		clientName, _ := a.getActiveClient()
		modal := NewEnvPickerModal(a.cfg, clientName, ws, a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	// ── Action modal openers (GPOD-118) ────────────────────────────────────

	case OpenBranchConfirmMsg:
		if a.issueDetail == nil {
			return a, nil
		}
		return a, a.createBranchCmd()

	case OpenCommitModalMsg:
		if a.issueDetail == nil {
			return a, nil
		}
		modal := NewCommitModal(a.issueDetail.issue.Key, a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	case OpenCommentModalMsg:
		modal := NewCommentModal(a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	case OpenStatusPickerMsg:
		if a.issueDetail == nil {
			return a, nil
		}
		modal := NewStatusPickerModal(a.reg, a.issueDetail.issue.Status, a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	case OpenPRModalMsg:
		if a.issueDetail == nil {
			return a, nil
		}
		issue := a.issueDetail.issue
		modal := NewPRModal(issue.Key, issue.Title, issue.Environment, issue.Workspace, a.styles)
		a.activeModal = modal
		a.hasModal = true
		return a, modal.Init()

	case OpenPlanAIMsg:
		if a.issueDetail == nil {
			return a, nil
		}
		return a, a.launchPlanAICmd()

	// ── Action submit messages (from modals → execute action) ──────────────

	case CommitSubmitMsg:
		a.hasModal = false
		a.activeModal = nil
		return a, a.commitPushCmd(msg.Type, msg.Scope, msg.Message)

	case CommentSubmitMsg:
		a.hasModal = false
		a.activeModal = nil
		return a, a.addCommentCmd(msg.Body)

	case StatusSelectedMsg:
		a.hasModal = false
		a.activeModal = nil
		return a, a.changeStatusCmd(msg.Status)

	case PRSubmitMsg:
		a.hasModal = false
		a.activeModal = nil
		return a, a.createPRCmd(msg.Title, msg.Body, msg.BaseBranch)

	// ── Action result messages (forward to detail for feedback) ────────────

	case StatusesFetchedMsg:
		if a.hasModal {
			newModal, cmd := a.activeModal.Update(msg)
			a.activeModal = newModal.(Modal)
			return a, cmd
		}

	case BranchCreatedMsg, CommitPushedMsg, PRCreatedMsg, CommentAddedMsg, StatusChangedMsg, PlanStartedMsg:
		if a.issueDetail != nil {
			newModel, cmd := a.issueDetail.Update(msg)
			a.issueDetail = newModel.(*IssueDetailModel)
			return a, cmd
		}

	// ── Navigation messages ────────────────────────────────────────────────

	case IssueSelectedMsg:
		a.issueDetail = a.createDetailModel(msg.Issue)
		a.screen = screenIssueDetail
		return a, nil

	// ── Issue fetch result ─────────────────────────────────────────────────

	case IssuesFetchedMsg:
		if a.issueList != nil {
			newModel, cmd := a.issueList.Update(msg)
			a.issueList = newModel.(*IssueListModel)
			return a, cmd
		}

	// ── Keyboard ──────────────────────────────────────────────────────────

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// [Esc] routing: modal → detail back → client focus
		if msg.String() == "esc" {
			if a.hasModal {
				a.hasModal = false
				a.activeModal = nil
				return a, nil
			}
			if a.screen == screenIssueDetail {
				a.screen = screenIssueList
				a.issueDetail = nil
				return a, nil
			}
			if a.focus == FocusContent {
				a.focus = FocusClients
				return a, nil
			}
		}

		// Delegate to modal when open
		if a.hasModal {
			newModal, cmd := a.activeModal.Update(msg)
			a.activeModal = newModal.(Modal)
			return a, cmd
		}

		// Client-focus navigation
		if a.focus == FocusClients {
			switch msg.String() {
			case "tab":
				return a, a.nextClient()
			case "shift+tab":
				return a, a.prevClient()
			case "enter":
				a.focus = FocusContent
			}
			return a, nil
		}

		// Content-focus: view tab switching
		switch msg.String() {
		case "1":
			a.view = ViewIssues
			return a, nil
		case "2":
			a.view = ViewPRs
			return a, nil
		case "3":
			a.view = ViewStatus
			return a, nil
		}

		// Delegate to active sub-model
		if a.focus == FocusContent && a.view == ViewIssues {
			if a.screen == screenIssueList && a.issueList != nil {
				newModel, cmd := a.issueList.Update(msg)
				a.issueList = newModel.(*IssueListModel)
				return a, cmd
			}
			if a.screen == screenIssueDetail && a.issueDetail != nil {
				newModel, cmd := a.issueDetail.Update(msg)
				a.issueDetail = newModel.(*IssueDetailModel)
				return a, cmd
			}
		}
	}

	return a, nil
}

// createDetailModel builds an IssueDetailModel, auto-selecting workspace/env
// when the client has only one option.
func (a *App) createDetailModel(issue store.IssueRecord) *IssueDetailModel {
	_, client := a.getActiveClient()

	if issue.Workspace == "" && len(client.Workspaces) == 1 {
		for wsName, ws := range client.Workspaces {
			issue.Workspace = wsName
			if len(ws.Contexts) == 1 {
				for ctxName := range ws.Contexts {
					issue.Environment = ctxName
				}
			}
		}
		if a.db != nil {
			_ = a.db.UpdateWorkContext(
				context.Background(),
				issue.ID, issue.Repos, issue.Workspace, issue.Environment,
			)
		}
	}

	return NewIssueDetailModel(a.db, issue, a.cfg, a.styles)
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	header := a.renderHeader()
	clientButtons := a.renderClientButtons()
	nav := a.renderNav()
	content := a.renderContentArea()

	// Modal overlay
	if a.hasModal {
		content = a.renderModalOverlay()
	}

	contentPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Surface700).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, nav, "", content))

	footerPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Surface700).
		Padding(0, 2).
		Render(a.renderFooter())

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header, "",
		clientButtons, "",
		contentPanel, "",
		footerPanel,
	)

	return lipgloss.NewStyle().
		Border(a.styles.BorderRounded).
		BorderForeground(Surface700).
		Padding(1, 2).
		Render(body)
}

func (a *App) renderContentArea() string {
	switch a.view {
	case ViewIssues:
		if a.screen == screenIssueDetail && a.issueDetail != nil {
			return a.issueDetail.View()
		}
		if a.issueList != nil {
			return a.issueList.View()
		}
		return a.styles.Placeholder.Render("Loading...")
	case ViewPRs:
		return a.styles.Placeholder.Render("Pull requests view (coming soon)")
	case ViewStatus:
		return a.styles.Placeholder.Render("Status view (coming soon)")
	default:
		return "Unknown view"
	}
}

func (a *App) renderModalOverlay() string {
	title := a.styles.Title.Render(a.activeModal.Title())
	body := a.activeModal.View()
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		strings.Repeat("─", 40),
		body,
	)
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
	return brand + strings.Repeat(" ", spacerWidth) + toolInfo
}

func (a *App) renderClientButtons() string {
	var buttons []string
	for idx, name := range a.getSortedClientNames() {
		client := a.cfg.Clients[name]
		dn := client.DisplayName
		if dn == "" {
			dn = name
		}
		var button string
		if idx == a.clientIdx {
			button = a.styles.ClientButtonActive.Render(" " + dn + " ")
		} else {
			button = a.styles.ClientButton.Render(" " + dn + " ")
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
	var rendered []string
	for _, tab := range tabs {
		style := a.styles.NavTab
		if tab.view == a.view {
			style = a.styles.NavTabActive
		}
		rendered = append(rendered, style.Render(tab.label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
}

func (a *App) renderFooter() string {
	var hints []string

	if a.hasModal {
		hints = []string{
			a.styles.HelpKey.Render("↑↓") + " " + a.styles.HelpDesc.Render("Navigate"),
			a.styles.HelpKey.Render("⏎") + " " + a.styles.HelpDesc.Render("Select"),
			a.styles.HelpKey.Render("ESC") + " " + a.styles.HelpDesc.Render("Cancel"),
		}
	} else if a.focus == FocusClients {
		hints = []string{
			a.styles.HelpKey.Render("tab") + " " + a.styles.HelpDesc.Render("Switch"),
			a.styles.HelpKey.Render("⏎") + " " + a.styles.HelpDesc.Render("Select"),
			a.styles.HelpKey.Render("q") + " " + a.styles.HelpDesc.Render("Quit"),
		}
	} else if a.view == ViewIssues && a.screen == screenIssueDetail {
		hints = []string{
			a.styles.HelpKey.Render("[b]") + " " + a.styles.HelpDesc.Render("Branch"),
			a.styles.HelpKey.Render("[c]") + " " + a.styles.HelpDesc.Render("Commit"),
			a.styles.HelpKey.Render("[r]") + " " + a.styles.HelpDesc.Render("PR"),
			a.styles.HelpKey.Render("[m]") + " " + a.styles.HelpDesc.Render("Comment"),
			a.styles.HelpKey.Render("[s]") + " " + a.styles.HelpDesc.Render("Status"),
			a.styles.HelpKey.Render("ESC") + " " + a.styles.HelpDesc.Render("Back"),
		}
	} else {
		hints = []string{
			a.styles.HelpKey.Render("↑↓") + " " + a.styles.HelpDesc.Render("Navigate"),
			a.styles.HelpKey.Render("⏎") + " " + a.styles.HelpDesc.Render("Open"),
			a.styles.HelpKey.Render("/") + " " + a.styles.HelpDesc.Render("Filter"),
			a.styles.HelpKey.Render("[r]") + " " + a.styles.HelpDesc.Render("Refresh"),
			a.styles.HelpKey.Render("ESC") + " " + a.styles.HelpDesc.Render("Clients"),
		}
	}

	return lipgloss.NewStyle().
		Foreground(Content400).
		Padding(0, 2).
		Render(strings.Join(hints, "  •  "))
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (a *App) getSortedClientNames() []string {
	var names []string
	for name := range a.cfg.Clients {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (a *App) getActiveClient() (string, config.ClientConfig) {
	names := a.getSortedClientNames()
	if a.clientIdx < len(names) {
		name := names[a.clientIdx]
		return name, a.cfg.Clients[name]
	}
	return "", config.ClientConfig{}
}

func (a *App) updateActiveClient() {
	name, _ := a.getActiveClient()
	if a.st == nil {
		a.st = &state.State{}
	}
	_ = a.st.UseClient(name)
}

func (a *App) nextClient() tea.Cmd {
	if len(a.cfg.Clients) == 0 {
		return nil
	}
	a.clientIdx = (a.clientIdx + 1) % len(a.cfg.Clients)
	a.updateActiveClient()
	a.initIssueList()
	return a.issueList.Init()
}

func (a *App) prevClient() tea.Cmd {
	n := len(a.cfg.Clients)
	if n == 0 {
		return nil
	}
	a.clientIdx = (a.clientIdx + n - 1) % n
	a.updateActiveClient()
	a.initIssueList()
	return a.issueList.Init()
}

func (a *App) getPlanningTool() string {
	_, client := a.getActiveClient()
	for pluginName := range client.Plugins {
		return pluginName
	}
	return "None"
}

func (a *App) getIssueCount() int {
	if a.issueList != nil {
		return len(a.issueList.items)
	}
	return 0
}

// ── Action commands (GPOD-118) ─────────────────────────────────────────────

// createBranchCmd creates a feature branch in all selected repos.
func (a *App) createBranchCmd() tea.Cmd {
	issue := a.issueDetail.issue
	repos := issue.Repos
	branchName := git.BranchNameForIssue(issue.Key)

	return func() tea.Msg {
		ctx := context.Background()
		runner := git.NewRunner("")
		for _, repo := range repos {
			if err := runner.CreateBranch(ctx, repo, branchName); err != nil {
				return BranchCreatedMsg{Err: fmt.Errorf("%s: %w", repo, err)}
			}
		}
		return BranchCreatedMsg{Branch: branchName}
	}
}

// commitPushCmd commits and pushes in all selected repos.
func (a *App) commitPushCmd(commitType, scope, message string) tea.Cmd {
	if a.issueDetail == nil {
		return nil
	}
	repos := a.issueDetail.issue.Repos
	fullMsg := git.CommitMessage(commitType, scope, message)

	return func() tea.Msg {
		ctx := context.Background()
		runner := git.NewRunner("")
		for _, repo := range repos {
			if err := runner.Commit(ctx, repo, fullMsg); err != nil {
				return CommitPushedMsg{Err: fmt.Errorf("commit %s: %w", repo, err)}
			}
			if err := runner.Push(ctx, repo); err != nil {
				return CommitPushedMsg{Err: fmt.Errorf("push %s: %w", repo, err)}
			}
		}
		return CommitPushedMsg{Message: fullMsg}
	}
}

// addCommentCmd posts a comment on the issue via the planning plugin.
func (a *App) addCommentCmd(body string) tea.Cmd {
	if a.issueDetail == nil || a.reg == nil {
		return nil
	}
	issue := a.issueDetail.issue
	reg := a.reg

	// Auto-include context in the comment
	var contextLine string
	if issue.Workspace != "" || issue.Environment != "" {
		parts := []string{}
		if issue.Workspace != "" {
			parts = append(parts, "Workspace: "+issue.Workspace)
		}
		if issue.Environment != "" {
			parts = append(parts, "Env: "+issue.Environment)
		}
		if len(issue.Repos) > 0 {
			parts = append(parts, "Repos: "+strings.Join(issue.Repos, ", "))
		}
		contextLine = "\n\n---\n_" + strings.Join(parts, " · ") + "_"
	}
	fullBody := body + contextLine

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
			err := pp.AddComment(ctx, issue.Key, fullBody)
			return CommentAddedMsg{Err: err}
		}
		return CommentAddedMsg{Err: fmt.Errorf("no planning plugin configured")}
	}
}

// changeStatusCmd changes the issue status via the planning plugin.
func (a *App) changeStatusCmd(status string) tea.Cmd {
	if a.issueDetail == nil || a.reg == nil {
		return nil
	}
	issueKey := a.issueDetail.issue.Key
	reg := a.reg

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
			err := pp.ChangeStatus(ctx, issueKey, status)
			if err != nil {
				return StatusChangedMsg{Err: err}
			}
			return StatusChangedMsg{NewStatus: status}
		}
		return StatusChangedMsg{Err: fmt.Errorf("no planning plugin configured")}
	}
}

// createPRCmd creates a PR via the PRPlugin for each selected repo.
func (a *App) createPRCmd(title, body, baseBranch string) tea.Cmd {
	if a.issueDetail == nil || a.reg == nil {
		return nil
	}
	issue := a.issueDetail.issue
	repos := issue.Repos
	reg := a.reg

	return func() tea.Msg {
		ctx := context.Background()

		// Find the first PRPlugin
		for _, name := range reg.ActivePlugins() {
			p, ok := reg.Get(name)
			if !ok {
				continue
			}
			prp, ok := p.(plugin.PRPlugin)
			if !ok {
				continue
			}

			// Get current branch from the first repo
			runner := git.NewRunner("")
			headBranch, err := runner.CurrentBranch(ctx, repos[0])
			if err != nil {
				headBranch = git.BranchNameForIssue(issue.Key)
			}

			var lastURL string
			for _, repo := range repos {
				pr, err := prp.CreatePR(ctx, plugin.PRRequest{
					Repo:       repo,
					Title:      title,
					Body:       body,
					HeadBranch: headBranch,
					BaseBranch: baseBranch,
				})
				if err != nil {
					return PRCreatedMsg{Err: fmt.Errorf("%s: %w", repo, err)}
				}
				lastURL = pr.URL
			}
			return PRCreatedMsg{URL: lastURL}
		}
		return PRCreatedMsg{Err: fmt.Errorf("no PR plugin configured")}
	}
}

// launchPlanAICmd launches the AI planner as an external process.
// Uses the first available AI tool: opencode, claude, or ollama.
func (a *App) launchPlanAICmd() tea.Cmd {
	if a.issueDetail == nil {
		return nil
	}
	issue := a.issueDetail.issue

	// Build the prompt with full work context
	prompt := fmt.Sprintf(
		"Plan implementation for issue %s: %s\n\nDescription:\n%s\n\nWorkspace: %s\nEnvironment: %s\nRepos: %s",
		issue.Key, issue.Title, issue.Description,
		issue.Workspace, issue.Environment,
		strings.Join(issue.Repos, ", "),
	)

	return func() tea.Msg {
		// Try AI tools in order of preference
		aiTools := []string{"opencode", "claude", "ollama"}
		for _, tool := range aiTools {
			if _, err := exec.LookPath(tool); err == nil {
				cmd := exec.Command(tool, "--prompt", prompt)
				if err := cmd.Start(); err != nil {
					return PlanStartedMsg{Err: fmt.Errorf("failed to start %s: %w", tool, err)}
				}
				return PlanStartedMsg{}
			}
		}
		return PlanStartedMsg{Err: fmt.Errorf("no AI tool found (install opencode, claude, or ollama)")}
	}
}
