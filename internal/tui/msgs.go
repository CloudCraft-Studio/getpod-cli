package tui

import (
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

// IssueSelectedMsg is emitted by IssueListModel when the user presses Enter on an issue.
type IssueSelectedMsg struct{ Issue store.IssueRecord }

// ReposSelectedMsg is emitted by RepoPickerModal when the user confirms a selection.
type ReposSelectedMsg struct{ Repos []string }

// WorkspaceSelectedMsg is emitted by WorkspacePickerModal.
type WorkspaceSelectedMsg struct{ Workspace string }

// EnvSelectedMsg is emitted by EnvPickerModal.
type EnvSelectedMsg struct{ Env string }

// ModalClosedMsg is emitted by any modal when dismissed without a selection (Esc).
type ModalClosedMsg struct{}

// IssuesFetchedMsg carries the result of a ListIssues call.
// Client is included so App can route to the correct IssueListModel.
type IssuesFetchedMsg struct {
	Client string
	Issues []store.IssueRecord
	Err    error
}

// ReposFetchedMsg carries the result of a ListRepos call.
type ReposFetchedMsg struct {
	Repos []plugin.Repo
	Err   error
}

// OpenRepoPickerMsg signals App to create and display a RepoPickerModal.
type OpenRepoPickerMsg struct{}

// OpenWorkspacePickerMsg signals App to create and display a WorkspacePickerModal.
type OpenWorkspacePickerMsg struct{}

// OpenEnvPickerMsg signals App to create and display an EnvPickerModal.
type OpenEnvPickerMsg struct{}

// ── Action messages (GPOD-118) ────────────────────────────────────────────

// ActionResultMsg carries the result of any workflow action (branch, commit, PR, etc.).
type ActionResultMsg struct {
	Action  string // "branch", "commit", "pr", "comment", "status", "plan"
	Success bool
	Message string // human-readable result or error
}

// BranchCreatedMsg is sent after branch creation completes across all repos.
type BranchCreatedMsg struct {
	Branch string
	Err    error
}

// CommitPushedMsg is sent after commit + push completes.
type CommitPushedMsg struct {
	Message string
	Err     error
}

// PRCreatedMsg is sent after PR creation completes.
type PRCreatedMsg struct {
	URL string
	Err error
}

// CommentAddedMsg is sent after a comment is posted on an issue.
type CommentAddedMsg struct{ Err error }

// StatusChangedMsg is sent after the issue status is changed.
type StatusChangedMsg struct {
	NewStatus string
	Err       error
}

// StatusesFetchedMsg carries the list of available statuses from the planning plugin.
type StatusesFetchedMsg struct {
	Statuses []string
	Err      error
}

// PlanStartedMsg is sent when the AI planning tool has been launched.
type PlanStartedMsg struct{ Err error }

// ── Open modal messages ───────────────────────────────────────────────────

// OpenCommitModalMsg signals App to open the commit message modal.
type OpenCommitModalMsg struct{}

// OpenCommentModalMsg signals App to open the comment modal.
type OpenCommentModalMsg struct{}

// OpenStatusPickerMsg signals App to open the status picker modal.
type OpenStatusPickerMsg struct{}

// OpenPRModalMsg signals App to open the PR creation modal.
type OpenPRModalMsg struct{}

// OpenBranchConfirmMsg signals App to start branch creation.
type OpenBranchConfirmMsg struct{}

// OpenPlanAIMsg signals App to launch the AI planner.
type OpenPlanAIMsg struct{}

// ── Submit messages from modals ───────────────────────────────────────────

// CommitSubmitMsg is emitted by CommitModal when the user confirms.
type CommitSubmitMsg struct {
	Type    string // feat, fix, chore, etc.
	Scope   string
	Message string
}

// CommentSubmitMsg is emitted by CommentModal when the user confirms.
type CommentSubmitMsg struct {
	Body string
}

// StatusSelectedMsg is emitted by StatusPickerModal when the user selects a status.
type StatusSelectedMsg struct {
	Status string
}

// PRSubmitMsg is emitted by PRModal when the user confirms.
type PRSubmitMsg struct {
	Title      string
	Body       string
	BaseBranch string
}
