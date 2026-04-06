package plugin

import (
	"context"
	"encoding/json"
	"time"
)

// CommentContext holds GetPod work context to be included in issue comments
type CommentContext struct {
	Workspace   string   // ej: "core-services"
	Environment string   // ej: "qa"
	Branch      string   // ej: "feature/lulo-1234"
	Repos       []string // ej: []string{"backend-core", "infra-terraform"}
}

// PlanningPlugin is implemented by plugins that manage issues (Jira, Linear, etc.).
// The TUI does a type assertion to detect this capability.
type PlanningPlugin interface {
	ListIssues(ctx context.Context) ([]Issue, error)
	GetIssue(ctx context.Context, key string) (*Issue, error)
	AddComment(ctx context.Context, key, body string, gpCtx *CommentContext) error
	ChangeStatus(ctx context.Context, key, status string) error
	ListStatuses(ctx context.Context) ([]string, error)
}

// RepoPlugin is implemented by plugins that manage repositories (GitHub, Bitbucket).
type RepoPlugin interface {
	ListRepos(ctx context.Context) ([]Repo, error)
}

// PRPlugin is implemented by plugins that can create pull requests.
// A plugin may implement both RepoPlugin and PRPlugin.
type PRPlugin interface {
	CreatePR(ctx context.Context, req PRRequest) (*PR, error)
}

// PRRequest contains the parameters for creating a pull request.
type PRRequest struct {
	Repo       string // repository name
	Title      string
	Body       string
	HeadBranch string // source branch
	BaseBranch string // target branch (e.g. "develop", "main")
}

// PR is the result of creating a pull request.
type PR struct {
	URL    string
	Number int
}

// Issue is a normalized ticket from a planning plugin.
type Issue struct {
	ID          string // globally unique: "pluginname:KEY" e.g. "linear:LULO-1234"
	Key         string // display key, e.g. "LULO-1234"
	Title       string
	Status      string
	Priority    string // empty when the plugin does not provide priority
	Description string
	Labels      []string
	RawData     json.RawMessage // full plugin payload — preserved for future extraction
}

// Repo is a code repository returned by a RepoPlugin.
type Repo struct {
	Name      string
	Source    string // "github" | "bitbucket"
	Language  string
	CloneURL  string
	UpdatedAt time.Time
}
