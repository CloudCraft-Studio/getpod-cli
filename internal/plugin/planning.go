package plugin

import (
	"context"
	"encoding/json"
	"time"
)

// PlanningPlugin is implemented by plugins that manage issues (Jira, Linear, etc.).
// The TUI does a type assertion to detect this capability.
type PlanningPlugin interface {
	ListIssues(ctx context.Context) ([]Issue, error)
	GetIssue(ctx context.Context, key string) (*Issue, error)
	AddComment(ctx context.Context, key, body string) error
	ChangeStatus(ctx context.Context, key, status string) error
	ListStatuses(ctx context.Context) ([]string, error)
}

// RepoPlugin is implemented by plugins that manage repositories (GitHub, Bitbucket).
type RepoPlugin interface {
	ListRepos(ctx context.Context) ([]Repo, error)
}

// Issue is a normalized ticket from a planning plugin.
type Issue struct {
	ID          string          // globally unique: "pluginname:KEY" e.g. "linear:LULO-1234"
	Key         string          // display key, e.g. "LULO-1234"
	Title       string
	Status      string
	Priority    string          // empty when the plugin does not provide priority
	Description string
	Labels      []string
	RawData     json.RawMessage // full plugin payload — preserved for future extraction
}

// Repo is a code repository returned by a RepoPlugin.
type Repo struct {
	Name      string
	Source    string    // "github" | "bitbucket"
	Language  string
	CloneURL  string
	UpdatedAt time.Time
}
