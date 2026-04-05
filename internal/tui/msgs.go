package tui

import (
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

// IssueSelectedMsg is emitted by IssueListModel when the user presses Enter on an issue.
type IssueSelectedMsg struct{ Issue store.IssueRecord }

// NavigateBackMsg is emitted by IssueDetailModel to return to the issue list.
type NavigateBackMsg struct{}

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
