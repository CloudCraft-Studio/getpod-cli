# GPOD-117: Issue List + Detail View + Work Context — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an issue list view with filterable issues per client, a detail view with Work Context (repos + workspace + env selectors), and three picker modals, all backed by an SQLite knowledge store.

**Architecture:** Sub-models Bubbletea — each view and modal is a `tea.Model` in the same `tui` package. `App` orchestrates routing and modal overlays. SQLite stores issues with their work context; the knowledge store fields (notes, started_at, finished_at) are preserved on fetch upserts.

**Tech Stack:** Go 1.25, Bubbletea v1.3, Lipgloss v1.1, modernc.org/sqlite, encoding/json, database/sql

---

## File Map

| Action | File | Responsibility |
|---|---|---|
| Modify | `internal/store/migrations.go` | Add `issues` table + index to schema |
| Create | `internal/store/issue.go` | `IssueRecord` type + CRUD methods |
| Create | `internal/store/issue_test.go` | Table-driven tests for issue CRUD |
| Create | `internal/plugin/planning.go` | `PlanningPlugin`, `RepoPlugin`, `Issue`, `Repo` types |
| Create | `internal/tui/msgs.go` | All message types for inter-model communication |
| Create | `internal/tui/issue_list.go` | `IssueListModel` — filterable issue list |
| Create | `internal/tui/issue_detail.go` | `IssueDetailModel` — detail + work context + actions |
| Create | `internal/tui/modal.go` | `Modal` interface |
| Create | `internal/tui/repo_picker.go` | `RepoPickerModal` — multi-select with async fetch |
| Create | `internal/tui/workspace_picker.go` | `WorkspacePickerModal` — single select from config |
| Create | `internal/tui/env_picker.go` | `EnvPickerModal` — single select with prod warning |
| Modify | `internal/tui/app.go` | Wire sub-models, modal overlay, Esc routing |
| Modify | `cmd/getpod/main.go` | Open store, pass to `NewApp` |

---

## Task 1: SQLite issues table + IssueRecord CRUD

**Files:**
- Modify: `internal/store/migrations.go`
- Create: `internal/store/issue.go`
- Create: `internal/store/issue_test.go`

- [ ] **Step 1.1: Write failing tests**

Create `internal/store/issue_test.go`:

```go
package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

func TestUpsertIssue_InsertsNew(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	ir := store.IssueRecord{
		ID:        "linear:LULO-1",
		Client:    "lulo",
		Key:       "LULO-1",
		Title:     "Fix EKS ingress",
		Status:    "Todo",
		Priority:  "High",
		Labels:    []string{"backend"},
		RawData:   json.RawMessage(`{"id":"LULO-1"}`),
		FetchedAt: now,
	}

	if err := s.UpsertIssue(ctx, ir); err != nil {
		t.Fatalf("UpsertIssue: %v", err)
	}

	got, err := s.GetIssue(ctx, "linear:LULO-1")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue, got nil")
	}
	if got.Key != "LULO-1" {
		t.Errorf("Key: want %q, got %q", "LULO-1", got.Key)
	}
	if got.Priority != "High" {
		t.Errorf("Priority: want %q, got %q", "High", got.Priority)
	}
	if len(got.Labels) != 1 || got.Labels[0] != "backend" {
		t.Errorf("Labels: want [backend], got %v", got.Labels)
	}
}

func TestUpsertIssue_PreservesWorkContextOnUpdate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Insert with work context
	ir := store.IssueRecord{
		ID: "linear:LULO-2", Client: "lulo", Key: "LULO-2",
		Title: "Old title", Status: "Todo", FetchedAt: now,
	}
	if err := s.UpsertIssue(ctx, ir); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := s.UpdateWorkContext(ctx, "linear:LULO-2", []string{"repo-a"}, "lulo-x", "qa"); err != nil {
		t.Fatalf("UpdateWorkContext: %v", err)
	}

	// Upsert again with new title (simulates a fetch refresh)
	ir.Title = "New title"
	ir.FetchedAt = now.Add(time.Minute)
	if err := s.UpsertIssue(ctx, ir); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err := s.GetIssue(ctx, "linear:LULO-2")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if got.Title != "New title" {
		t.Errorf("Title not updated: got %q", got.Title)
	}
	// work context must be preserved
	if got.Workspace != "lulo-x" {
		t.Errorf("Workspace clobbered: got %q", got.Workspace)
	}
	if len(got.Repos) != 1 || got.Repos[0] != "repo-a" {
		t.Errorf("Repos clobbered: got %v", got.Repos)
	}
}

func TestListIssuesByClient_FiltersCorrectly(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for _, ir := range []store.IssueRecord{
		{ID: "linear:A-1", Client: "client-a", Key: "A-1", Title: "A1", Status: "Todo", FetchedAt: now},
		{ID: "linear:A-2", Client: "client-a", Key: "A-2", Title: "A2", Status: "Todo", FetchedAt: now},
		{ID: "linear:B-1", Client: "client-b", Key: "B-1", Title: "B1", Status: "Todo", FetchedAt: now},
	} {
		if err := s.UpsertIssue(ctx, ir); err != nil {
			t.Fatalf("UpsertIssue: %v", err)
		}
	}

	got, err := s.ListIssuesByClient(ctx, "client-a")
	if err != nil {
		t.Fatalf("ListIssuesByClient: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
}

func TestGetIssue_NotFound(t *testing.T) {
	s := newTestStore(t)
	got, err := s.GetIssue(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateWorkContext_PersistsValues(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := s.UpsertIssue(ctx, store.IssueRecord{
		ID: "linear:C-1", Client: "c", Key: "C-1", Title: "C", Status: "Todo", FetchedAt: now,
	}); err != nil {
		t.Fatalf("UpsertIssue: %v", err)
	}

	if err := s.UpdateWorkContext(ctx, "linear:C-1", []string{"repo-x", "repo-y"}, "ws-1", "prod"); err != nil {
		t.Fatalf("UpdateWorkContext: %v", err)
	}

	got, err := s.GetIssue(ctx, "linear:C-1")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if got.Workspace != "ws-1" {
		t.Errorf("Workspace: want %q, got %q", "ws-1", got.Workspace)
	}
	if got.Environment != "prod" {
		t.Errorf("Environment: want %q, got %q", "prod", got.Environment)
	}
	if len(got.Repos) != 2 || got.Repos[0] != "repo-x" {
		t.Errorf("Repos: got %v", got.Repos)
	}
}

func TestHasIssuesForClient(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	ok, err := s.HasIssuesForClient(ctx, "nobody")
	if err != nil {
		t.Fatalf("HasIssuesForClient: %v", err)
	}
	if ok {
		t.Error("expected false for empty store")
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := s.UpsertIssue(ctx, store.IssueRecord{
		ID: "linear:D-1", Client: "someone", Key: "D-1", Title: "D", Status: "Todo", FetchedAt: now,
	}); err != nil {
		t.Fatalf("UpsertIssue: %v", err)
	}

	ok, err = s.HasIssuesForClient(ctx, "someone")
	if err != nil {
		t.Fatalf("HasIssuesForClient: %v", err)
	}
	if !ok {
		t.Error("expected true after insert")
	}
}
```

- [ ] **Step 1.2: Run tests to verify they fail**

```bash
cd /Users/lx-duv0-x/Documents/repos/CloudCraft\ Studio/getpod-cli
go test ./internal/store/... -run "TestUpsertIssue|TestListIssues|TestGetIssue|TestUpdateWork|TestHasIssues" -v 2>&1 | head -30
```

Expected: compilation error — `store.IssueRecord` and methods not defined.

- [ ] **Step 1.3: Add issues table to migrations.go**

In `internal/store/migrations.go`, append to the `schema` constant (before the closing backtick):

```go
const schema = `
CREATE TABLE IF NOT EXISTS events (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	plugin     TEXT NOT NULL,
	event      TEXT NOT NULL,
	timestamp  DATETIME NOT NULL,
	meta       TEXT NOT NULL,
	synced     BOOLEAN DEFAULT FALSE,
	synced_at  DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_unsynced ON events(synced) WHERE synced = FALSE;
CREATE INDEX IF NOT EXISTS idx_events_plugin   ON events(plugin, timestamp);

CREATE TABLE IF NOT EXISTS collection_cursors (
	plugin            TEXT PRIMARY KEY,
	last_collected_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS issues (
	id          TEXT PRIMARY KEY,
	client      TEXT NOT NULL,
	key         TEXT NOT NULL,
	title       TEXT NOT NULL,
	status      TEXT NOT NULL,
	priority    TEXT,
	description TEXT,
	labels      TEXT,
	raw_data    TEXT,
	fetched_at  DATETIME NOT NULL,
	repos       TEXT,
	workspace   TEXT,
	environment TEXT,
	notes       TEXT,
	started_at  DATETIME,
	finished_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_issues_client ON issues(client);
`
```

- [ ] **Step 1.4: Create internal/store/issue.go**

```go
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// IssueRecord is a cached issue with its local work context and knowledge store.
type IssueRecord struct {
	ID          string
	Client      string
	Key         string
	Title       string
	Status      string
	Priority    string          // empty when plugin does not provide priority
	Description string
	Labels      []string
	RawData     json.RawMessage // full plugin payload
	FetchedAt   time.Time
	// Work context — persists per issue, never overwritten by fetch
	Repos       []string
	Workspace   string
	Environment string
	// Knowledge store — for future use (notes, timing)
	Notes      string
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// UpsertIssue inserts or updates the fetched fields of an issue
// (title, status, priority, description, labels, raw_data, fetched_at).
// On conflict it leaves repos, workspace, environment, notes, started_at,
// and finished_at untouched — preserving the knowledge store.
func (s *Store) UpsertIssue(ctx context.Context, ir IssueRecord) error {
	labelsJSON, err := json.Marshal(ir.Labels)
	if err != nil {
		return fmt.Errorf("encoding labels: %w", err)
	}
	rawData := string(ir.RawData)
	if rawData == "" {
		rawData = "null"
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO issues (id, client, key, title, status, priority, description, labels, raw_data, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title       = excluded.title,
			status      = excluded.status,
			priority    = excluded.priority,
			description = excluded.description,
			labels      = excluded.labels,
			raw_data    = excluded.raw_data,
			fetched_at  = excluded.fetched_at`,
		ir.ID, ir.Client, ir.Key, ir.Title, ir.Status,
		nullableString(ir.Priority), ir.Description,
		string(labelsJSON), rawData, encodeTime(ir.FetchedAt),
	)
	if err != nil {
		return fmt.Errorf("upserting issue %q: %w", ir.ID, err)
	}
	return nil
}

// ListIssuesByClient returns all cached issues for a client, ordered by key.
func (s *Store) ListIssuesByClient(ctx context.Context, client string) ([]IssueRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client, key, title, status, priority, description, labels, raw_data,
		       fetched_at, repos, workspace, environment, notes, started_at, finished_at
		FROM issues WHERE client = ? ORDER BY key ASC`, client)
	if err != nil {
		return nil, fmt.Errorf("listing issues for client %q: %w", client, err)
	}
	defer rows.Close()

	var result []IssueRecord
	for rows.Next() {
		ir, err := scanIssueRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ir)
	}
	return result, rows.Err()
}

// GetIssue returns a single issue by ID. Returns nil, nil when not found.
func (s *Store) GetIssue(ctx context.Context, id string) (*IssueRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client, key, title, status, priority, description, labels, raw_data,
		       fetched_at, repos, workspace, environment, notes, started_at, finished_at
		FROM issues WHERE id = ?`, id)
	ir, err := scanIssueRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ir, err
}

// UpdateWorkContext persists work context for an issue.
// Replaces existing repos/workspace/environment — call after each picker selection.
func (s *Store) UpdateWorkContext(ctx context.Context, id string, repos []string, workspace, environment string) error {
	reposJSON, err := json.Marshal(repos)
	if err != nil {
		return fmt.Errorf("encoding repos: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE issues SET repos = ?, workspace = ?, environment = ? WHERE id = ?`,
		string(reposJSON), nullableString(workspace), nullableString(environment), id,
	)
	if err != nil {
		return fmt.Errorf("updating work context for %q: %w", id, err)
	}
	return nil
}

// HasIssuesForClient reports whether any cached issues exist for the given client.
func (s *Store) HasIssuesForClient(ctx context.Context, client string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM issues WHERE client = ?`, client).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("counting issues for client %q: %w", client, err)
	}
	return count > 0, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanIssueRow(s scanner) (*IssueRecord, error) {
	var (
		ir             IssueRecord
		priorityNull   sql.NullString
		labelsStr      string
		rawDataStr     string
		fetchedAtStr   string
		reposNull      sql.NullString
		workspaceNull  sql.NullString
		envNull        sql.NullString
		notesNull      sql.NullString
		startedAtNull  *string
		finishedAtNull *string
	)

	if err := s.Scan(
		&ir.ID, &ir.Client, &ir.Key, &ir.Title, &ir.Status,
		&priorityNull, &ir.Description, &labelsStr, &rawDataStr, &fetchedAtStr,
		&reposNull, &workspaceNull, &envNull, &notesNull, &startedAtNull, &finishedAtNull,
	); err != nil {
		return nil, fmt.Errorf("scanning issue: %w", err)
	}

	ir.Priority = priorityNull.String
	ir.RawData = json.RawMessage(rawDataStr)
	ir.Workspace = workspaceNull.String
	ir.Environment = envNull.String
	ir.Notes = notesNull.String

	if err := json.Unmarshal([]byte(labelsStr), &ir.Labels); err != nil {
		ir.Labels = nil // tolerate missing/null labels
	}
	if reposNull.Valid && reposNull.String != "" && reposNull.String != "null" {
		if err := json.Unmarshal([]byte(reposNull.String), &ir.Repos); err != nil {
			ir.Repos = nil
		}
	}

	var err error
	if ir.FetchedAt, err = decodeTime(fetchedAtStr); err != nil {
		return nil, fmt.Errorf("parsing fetched_at: %w", err)
	}
	if startedAtNull != nil {
		t, err := decodeTime(*startedAtNull)
		if err != nil {
			return nil, fmt.Errorf("parsing started_at: %w", err)
		}
		ir.StartedAt = &t
	}
	if finishedAtNull != nil {
		t, err := decodeTime(*finishedAtNull)
		if err != nil {
			return nil, fmt.Errorf("parsing finished_at: %w", err)
		}
		ir.FinishedAt = &t
	}

	return &ir, nil
}

func nullableString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
```

- [ ] **Step 1.5: Run tests to verify they pass**

```bash
go test ./internal/store/... -v 2>&1 | tail -20
```

Expected: all store tests PASS including the new ones.

- [ ] **Step 1.6: Commit**

```bash
git add internal/store/migrations.go internal/store/issue.go internal/store/issue_test.go
git commit -m "feat(GPOD-117): add issues table and IssueRecord CRUD to store"
```

---

## Task 2: PlanningPlugin and RepoPlugin interfaces

**Files:**
- Create: `internal/plugin/planning.go`

- [ ] **Step 2.1: Create internal/plugin/planning.go**

```go
package plugin

import (
	"context"
	"encoding/json"
	"time"
)

// PlanningPlugin is implemented by plugins that manage issues (Jira, Linear, etc.).
// The TUI does a type assertion to detect this capability — only implement what applies.
type PlanningPlugin interface {
	// ListIssues returns open (non-closed) issues for the active context.
	ListIssues(ctx context.Context) ([]Issue, error)

	// GetIssue returns a single issue by its display key (e.g. "LULO-1234").
	GetIssue(ctx context.Context, key string) (*Issue, error)

	// AddComment posts a comment on the given issue.
	AddComment(ctx context.Context, key, body string) error

	// ChangeStatus transitions an issue to the named status.
	ChangeStatus(ctx context.Context, key, status string) error

	// ListStatuses returns all available status names for the active context.
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
```

- [ ] **Step 2.2: Verify compilation**

```bash
go build ./internal/plugin/...
```

Expected: no errors.

- [ ] **Step 2.3: Commit**

```bash
git add internal/plugin/planning.go
git commit -m "feat(GPOD-117): add PlanningPlugin, RepoPlugin interfaces and Issue, Repo types"
```

---

## Task 3: TUI message types

**Files:**
- Create: `internal/tui/msgs.go`

- [ ] **Step 3.1: Create internal/tui/msgs.go**

```go
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
```

- [ ] **Step 3.2: Verify compilation**

```bash
go build ./internal/tui/...
```

Expected: no errors.

- [ ] **Step 3.3: Commit**

```bash
git add internal/tui/msgs.go
git commit -m "feat(GPOD-117): add TUI inter-model message types"
```

---

## Task 4: IssueListModel

**Files:**
- Create: `internal/tui/issue_list.go`
- Create: `internal/tui/issue_list_test.go`

- [ ] **Step 4.1: Write failing tests**

Create `internal/tui/issue_list_test.go`:

```go
package tui

import (
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

func makeTestIssues() []store.IssueRecord {
	return []store.IssueRecord{
		{Key: "LULO-1", Title: "Fix EKS ingress controller", Status: "In Progress"},
		{Key: "LULO-2", Title: "Update Terraform modules", Status: "Todo"},
		{Key: "LULO-3", Title: "Add CloudWatch alarms", Status: "Backlog"},
	}
}

func TestIssueListModel_ApplyFilter_EmptyReturnsAll(t *testing.T) {
	items := makeTestIssues()
	m := &IssueListModel{items: items}
	m.filter = ""
	m.applyFilter()

	if len(m.filtered) != len(items) {
		t.Errorf("expected %d, got %d", len(items), len(m.filtered))
	}
}

func TestIssueListModel_ApplyFilter_MatchesKey(t *testing.T) {
	m := &IssueListModel{items: makeTestIssues()}
	m.filter = "LULO-2"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match, got %d", len(m.filtered))
	}
	if m.filtered[0].Key != "LULO-2" {
		t.Errorf("wrong match: %q", m.filtered[0].Key)
	}
}

func TestIssueListModel_ApplyFilter_MatchesTitleCaseInsensitive(t *testing.T) {
	m := &IssueListModel{items: makeTestIssues()}
	m.filter = "eks"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match, got %d", len(m.filtered))
	}
	if m.filtered[0].Key != "LULO-1" {
		t.Errorf("wrong match: %q", m.filtered[0].Key)
	}
}

func TestIssueListModel_ApplyFilter_NoMatchReturnsEmpty(t *testing.T) {
	m := &IssueListModel{items: makeTestIssues()}
	m.filter = "zzznomatch"
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.filtered))
	}
}

func TestIssueListModel_ApplyFilter_ResetsCursorToZero(t *testing.T) {
	m := &IssueListModel{items: makeTestIssues(), cursor: 2}
	m.filter = "eks"
	m.applyFilter()

	if m.cursor != 0 {
		t.Errorf("cursor not reset: got %d", m.cursor)
	}
}

func TestIssueListModel_ApplyFilter_MatchesStatus(t *testing.T) {
	m := &IssueListModel{items: makeTestIssues()}
	m.filter = "backlog"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match on status, got %d", len(m.filtered))
	}
	if m.filtered[0].Key != "LULO-3" {
		t.Errorf("wrong match: %q", m.filtered[0].Key)
	}
}
```

- [ ] **Step 4.2: Run tests to verify they fail**

```bash
go test ./internal/tui/... -run "TestIssueListModel" -v 2>&1 | head -20
```

Expected: compilation error — `IssueListModel` and `applyFilter` not defined.

- [ ] **Step 4.3: Create internal/tui/issue_list.go**

```go
package tui

import (
	"context"
	"fmt"
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
		issues, err := m.db.ListIssuesByClient(ctx, m.client)
		if err != nil {
			return IssuesFetchedMsg{Client: m.client, Err: err}
		}
		if len(issues) > 0 {
			return IssuesFetchedMsg{Client: m.client, Issues: issues}
		}
		// No cache: fetch from plugin
		return m.fetchFromPlugin(ctx)
	}
}

func (m *IssueListModel) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return m.fetchFromPlugin(context.Background())
	}
}

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
			if upsertErr := m.db.UpsertIssue(ctx, ir); upsertErr == nil {
				// Re-read to pick up preserved work context
				if full, readErr := m.db.GetIssue(ctx, ir.ID); readErr == nil && full != nil {
					records = append(records, *full)
					continue
				}
				records = append(records, ir)
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
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
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
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
```

- [ ] **Step 4.4: Run tests to verify they pass**

```bash
go test ./internal/tui/... -run "TestIssueListModel" -v
```

Expected: all 6 tests PASS.

- [ ] **Step 4.5: Commit**

```bash
git add internal/tui/issue_list.go internal/tui/issue_list_test.go
git commit -m "feat(GPOD-117): add IssueListModel with filter, navigation and async fetch"
```

---

## Task 5: IssueDetailModel

**Files:**
- Create: `internal/tui/issue_detail.go`
- Create: `internal/tui/issue_detail_test.go`

- [ ] **Step 5.1: Write failing tests**

Create `internal/tui/issue_detail_test.go`:

```go
package tui

import (
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

func newTestDetailModel(ir store.IssueRecord) *IssueDetailModel {
	cfg := &config.Config{
		Clients: map[string]config.ClientConfig{
			"lulo": {
				Workspaces: map[string]config.WorkspaceConfig{
					"lulo-x": {
						Contexts: map[string]config.ContextConfig{
							"qa":   {"aws_account": "111111111111", "aws_region": "us-east-1"},
							"prod": {"aws_account": "333333333333", "aws_region": "us-east-1"},
						},
					},
				},
			},
		},
	}
	return &IssueDetailModel{issue: ir, cfg: cfg, styles: DefaultStyles()}
}

func TestIssueDetailModel_IsReady_FalseWhenEmpty(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{})
	if m.IsReady() {
		t.Error("expected not ready with empty work context")
	}
}

func TestIssueDetailModel_IsReady_FalseWhenMissingEnv(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Repos:     []string{"repo-a"},
		Workspace: "lulo-x",
		// Environment missing
	})
	if m.IsReady() {
		t.Error("expected not ready without environment")
	}
}

func TestIssueDetailModel_IsReady_TrueWhenAllSet(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Repos:       []string{"repo-a"},
		Workspace:   "lulo-x",
		Environment: "qa",
	})
	if !m.IsReady() {
		t.Error("expected ready with repos + workspace + env")
	}
}

func TestIssueDetailModel_MissingContext_AllMissing(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{})
	got := m.missingContext()
	for _, want := range []string{"repos", "workspace", "environment"} {
		if !containsStr(got, want) {
			t.Errorf("expected %q in missing context, got %q", want, got)
		}
	}
}

func TestIssueDetailModel_MissingContext_OnlyMissingEnv(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Repos: []string{"r"}, Workspace: "ws",
	})
	got := m.missingContext()
	if !containsStr(got, "environment") {
		t.Errorf("expected 'environment' in %q", got)
	}
	if containsStr(got, "repos") || containsStr(got, "workspace") {
		t.Errorf("unexpected items in %q", got)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		len(s) > len(sub) && (s[:len(sub)] == sub || s[len(s)-len(sub):] == sub ||
			findSubstr(s, sub)))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5.2: Run tests to verify they fail**

```bash
go test ./internal/tui/... -run "TestIssueDetailModel" -v 2>&1 | head -20
```

Expected: compilation error — `IssueDetailModel` not defined.

- [ ] **Step 5.3: Create internal/tui/issue_detail.go**

```go
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

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.descOffset > 0 {
				m.descOffset--
			}
		case "down", "j":
			m.descOffset++
		case "w":
			return m, func() tea.Msg { return OpenRepoPickerMsg{} }
		case "x":
			return m, func() tea.Msg { return OpenWorkspacePickerMsg{} }
		case "e":
			if m.issue.Workspace != "" {
				return m, func() tea.Msg { return OpenEnvPickerMsg{} }
			}
		}
	}
	return m, nil
}

func (m *IssueDetailModel) persistWorkContextCmd() tea.Cmd {
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
	return strings.Join(sections, "\n")
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
	if m.descOffset >= len(lines) {
		m.descOffset = len(lines) - 1
	}
	const maxVisible = 6
	end := m.descOffset + maxVisible
	if end > len(lines) {
		end = len(lines)
	}
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
	return ctx["aws_account"], ctx["aws_region"]
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
```

- [ ] **Step 5.4: Run tests to verify they pass**

```bash
go test ./internal/tui/... -run "TestIssueDetailModel" -v
```

Expected: all 5 tests PASS.

- [ ] **Step 5.5: Commit**

```bash
git add internal/tui/issue_detail.go internal/tui/issue_detail_test.go
git commit -m "feat(GPOD-117): add IssueDetailModel with work context and action stubs"
```

---

## Task 6: Modal interface + RepoPickerModal

**Files:**
- Create: `internal/tui/modal.go`
- Create: `internal/tui/repo_picker.go`
- Create: `internal/tui/repo_picker_test.go`

- [ ] **Step 6.1: Write failing tests**

Create `internal/tui/repo_picker_test.go`:

```go
package tui

import (
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
)

func makeTestRepos() []plugin.Repo {
	return []plugin.Repo{
		{Name: "repo-backend", Source: "github", Language: "Go"},
		{Name: "repo-frontend", Source: "github", Language: "TypeScript"},
		{Name: "infra-terraform", Source: "github", Language: "HCL"},
	}
}

func TestRepoPickerModal_ToggleSelection(t *testing.T) {
	m := &RepoPickerModal{
		items:    makeTestRepos(),
		selected: make(map[string]bool),
		styles:   DefaultStyles(),
	}
	m.applyFilter()

	// Toggle on
	m.selected[m.filtered[0].Name] = !m.selected[m.filtered[0].Name]
	if !m.selected["repo-backend"] {
		t.Error("expected repo-backend selected after toggle")
	}

	// Toggle off
	m.selected[m.filtered[0].Name] = !m.selected[m.filtered[0].Name]
	if m.selected["repo-backend"] {
		t.Error("expected repo-backend deselected after second toggle")
	}
}

func TestRepoPickerModal_ApplyFilter(t *testing.T) {
	m := &RepoPickerModal{
		items:    makeTestRepos(),
		selected: make(map[string]bool),
		styles:   DefaultStyles(),
	}
	m.filter = "terra"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match, got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "infra-terraform" {
		t.Errorf("wrong match: %q", m.filtered[0].Name)
	}
}

func TestRepoPickerModal_ApplyFilter_EmptyReturnsAll(t *testing.T) {
	m := &RepoPickerModal{
		items:    makeTestRepos(),
		selected: make(map[string]bool),
		styles:   DefaultStyles(),
	}
	m.filter = ""
	m.applyFilter()

	if len(m.filtered) != len(m.items) {
		t.Errorf("expected %d, got %d", len(m.items), len(m.filtered))
	}
}

func TestRepoPickerModal_PreselectedMarked(t *testing.T) {
	m := NewRepoPickerModal(nil, []string{"repo-backend"}, DefaultStyles())
	if !m.selected["repo-backend"] {
		t.Error("expected preselected repo to be marked")
	}
}
```

- [ ] **Step 6.2: Run tests to verify they fail**

```bash
go test ./internal/tui/... -run "TestRepoPickerModal" -v 2>&1 | head -20
```

Expected: compilation error — `RepoPickerModal` not defined.

- [ ] **Step 6.3: Create internal/tui/modal.go**

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// Modal is the interface for all overlay components.
// App uses hasModal bool alongside this to avoid the nil-interface trap.
type Modal interface {
	tea.Model
	Title() string
}
```

- [ ] **Step 6.4: Create internal/tui/repo_picker.go**

```go
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
	return func() tea.Msg {
		ctx := context.Background()
		for _, name := range m.reg.ActivePlugins() {
			p, ok := m.reg.Get(name)
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
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
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
		m.filtered = m.items
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

	var lines []string
	if m.filterOn {
		lines = append(lines, m.styles.HelpKey.Render("/")+" "+m.styles.Paragraph.Render(m.filter+"_"))
	}

	for i, r := range m.filtered {
		checkbox := m.styles.Muted.Render("[ ]")
		if m.selected[r.Name] {
			checkbox = lipgloss.NewStyle().Foreground(Success400).Render("[x]")
		}
		age := repoAge(r.UpdatedAt)
		row := fmt.Sprintf("%s  %-30s  %-10s  %-8s  %s", checkbox, r.Name, r.Source, r.Language, age)
		if i == m.cursor {
			row = lipgloss.NewStyle().Foreground(Primary400).Render("> " + row)
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
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
```

- [ ] **Step 6.5: Run tests to verify they pass**

```bash
go test ./internal/tui/... -run "TestRepoPickerModal" -v
```

Expected: all 4 tests PASS.

- [ ] **Step 6.6: Commit**

```bash
git add internal/tui/modal.go internal/tui/repo_picker.go internal/tui/repo_picker_test.go
git commit -m "feat(GPOD-117): add Modal interface and RepoPickerModal with multi-select"
```

---

## Task 7: WorkspacePickerModal

**Files:**
- Create: `internal/tui/workspace_picker.go`
- Create: `internal/tui/workspace_picker_test.go`

- [ ] **Step 7.1: Write failing tests**

Create `internal/tui/workspace_picker_test.go`:

```go
package tui

import (
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

func testConfigWithWorkspaces() *config.Config {
	return &config.Config{
		Clients: map[string]config.ClientConfig{
			"lulo": {
				Workspaces: map[string]config.WorkspaceConfig{
					"lulo-x": {
						DisplayName: "Lulo X",
						Contexts:    map[string]config.ContextConfig{"qa": {}, "stg": {}},
					},
					"lulo-business": {
						DisplayName: "Lulo Business",
						Contexts:    map[string]config.ContextConfig{"qa": {}},
					},
				},
			},
		},
	}
}

func TestWorkspacePickerModal_BuildsItemsFromConfig(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	if len(m.items) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(m.items))
	}
}

func TestWorkspacePickerModal_ItemsSortedByName(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	if m.items[0].name != "lulo-business" {
		t.Errorf("expected lulo-business first (alphabetical), got %q", m.items[0].name)
	}
}

func TestWorkspacePickerModal_EnvCountCorrect(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	var luloX workspaceItem
	for _, it := range m.items {
		if it.name == "lulo-x" {
			luloX = it
		}
	}
	if luloX.envCount != 2 {
		t.Errorf("lulo-x should have 2 envs, got %d", luloX.envCount)
	}
}
```

- [ ] **Step 7.2: Run tests to verify they fail**

```bash
go test ./internal/tui/... -run "TestWorkspacePickerModal" -v 2>&1 | head -20
```

Expected: compilation error.

- [ ] **Step 7.3: Create internal/tui/workspace_picker.go**

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// WorkspacePickerModal lets the user pick a single workspace from the client config.
type WorkspacePickerModal struct {
	items  []workspaceItem
	cursor int
	styles Styles
}

type workspaceItem struct {
	name        string
	displayName string
	envCount    int
}

// NewWorkspacePickerModal builds the list from cfg for the given client.
func NewWorkspacePickerModal(cfg *config.Config, client string, styles Styles) *WorkspacePickerModal {
	cl := cfg.Clients[client]
	var items []workspaceItem
	for name, ws := range cl.Workspaces {
		dn := ws.DisplayName
		if dn == "" {
			dn = name
		}
		items = append(items, workspaceItem{
			name:        name,
			displayName: dn,
			envCount:    len(ws.Contexts),
		})
	}
	// sort alphabetically by name
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].name > items[j].name {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return &WorkspacePickerModal{items: items, styles: styles}
}

func (m *WorkspacePickerModal) Title() string { return "Select Workspace" }
func (m *WorkspacePickerModal) Init() tea.Cmd { return nil }

func (m *WorkspacePickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.items) {
				name := m.items[m.cursor].name
				return m, func() tea.Msg { return WorkspaceSelectedMsg{Workspace: name} }
			}
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m *WorkspacePickerModal) View() string {
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No workspaces configured for this client.")
	}
	var lines []string
	for i, ws := range m.items {
		envLabel := fmt.Sprintf("%d env", ws.envCount)
		if ws.envCount != 1 {
			envLabel += "s"
		}
		row := fmt.Sprintf("%-22s  %s", ws.displayName, m.styles.Muted.Render(envLabel))
		if i == m.cursor {
			row = lipgloss.NewStyle().Foreground(Primary400).Render("> " + row)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("↑↓ navigate · enter select · esc cancel"))
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 7.4: Run tests to verify they pass**

```bash
go test ./internal/tui/... -run "TestWorkspacePickerModal" -v
```

Expected: all 3 tests PASS.

- [ ] **Step 7.5: Commit**

```bash
git add internal/tui/workspace_picker.go internal/tui/workspace_picker_test.go
git commit -m "feat(GPOD-117): add WorkspacePickerModal"
```

---

## Task 8: EnvPickerModal

**Files:**
- Create: `internal/tui/env_picker.go`
- Create: `internal/tui/env_picker_test.go`

- [ ] **Step 8.1: Write failing tests**

Create `internal/tui/env_picker_test.go`:

```go
package tui

import (
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

func testConfigWithEnvs() *config.Config {
	return &config.Config{
		Clients: map[string]config.ClientConfig{
			"lulo": {
				Workspaces: map[string]config.WorkspaceConfig{
					"lulo-x": {
						Contexts: map[string]config.ContextConfig{
							"qa":   {"aws_account": "111111111111", "aws_region": "us-east-1"},
							"stg":  {"aws_account": "222222222222", "aws_region": "us-east-1"},
							"prod": {"aws_account": "333333333333", "aws_region": "us-east-1"},
						},
					},
				},
			},
		},
	}
}

func TestEnvPickerModal_BuildsItemsFromConfig(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	if len(m.items) != 3 {
		t.Fatalf("expected 3 environments, got %d", len(m.items))
	}
}

func TestEnvPickerModal_ProdFlaggedAsProd(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	var prodItem envItem
	for _, it := range m.items {
		if it.name == "prod" {
			prodItem = it
		}
	}
	if !prodItem.isProd {
		t.Error("expected prod environment to have isProd=true")
	}
}

func TestEnvPickerModal_QaNotFlaggedAsProd(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	for _, it := range m.items {
		if it.name == "qa" && it.isProd {
			t.Error("qa should not be flagged as prod")
		}
	}
}

func TestEnvPickerModal_AWSAccountPopulated(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	for _, it := range m.items {
		if it.name == "qa" {
			if it.awsAccount != "111111111111" {
				t.Errorf("expected AWS account 111111111111, got %q", it.awsAccount)
			}
		}
	}
}
```

- [ ] **Step 8.2: Run tests to verify they fail**

```bash
go test ./internal/tui/... -run "TestEnvPickerModal" -v 2>&1 | head -20
```

Expected: compilation error.

- [ ] **Step 8.3: Create internal/tui/env_picker.go**

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// EnvPickerModal lets the user pick a single environment from the selected workspace.
// Environments containing "prod" in their name display a ⚠ warning.
type EnvPickerModal struct {
	items  []envItem
	cursor int
	styles Styles
}

type envItem struct {
	name       string
	awsAccount string
	awsRegion  string
	isProd     bool
}

// NewEnvPickerModal builds the list from the workspace's contexts in cfg.
func NewEnvPickerModal(cfg *config.Config, client, workspace string, styles Styles) *EnvPickerModal {
	cl := cfg.Clients[client]
	ws := cl.Workspaces[workspace]
	var items []envItem
	for name, ctx := range ws.Contexts {
		items = append(items, envItem{
			name:       name,
			awsAccount: ctx["aws_account"],
			awsRegion:  ctx["aws_region"],
			isProd:     strings.Contains(strings.ToLower(name), "prod"),
		})
	}
	// sort alphabetically so order is stable
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].name > items[j].name {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return &EnvPickerModal{items: items, styles: styles}
}

func (m *EnvPickerModal) Title() string { return "Select Environment" }
func (m *EnvPickerModal) Init() tea.Cmd { return nil }

func (m *EnvPickerModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.items) {
				name := m.items[m.cursor].name
				return m, func() tea.Msg { return EnvSelectedMsg{Env: name} }
			}
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m *EnvPickerModal) View() string {
	if len(m.items) == 0 {
		return m.styles.Muted.Render("No environments configured for this workspace.")
	}
	var lines []string
	for i, env := range m.items {
		name := env.name
		if env.isProd {
			name = lipgloss.NewStyle().Foreground(Warning400).Render("⚠ " + name)
		}
		row := fmt.Sprintf("%-14s  AWS %-14s  %s", name, env.awsAccount, env.awsRegion)
		if i == m.cursor {
			row = lipgloss.NewStyle().Foreground(Primary400).Render("> " + row)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("↑↓ navigate · enter select · esc cancel"))
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 8.4: Run tests to verify they pass**

```bash
go test ./internal/tui/... -run "TestEnvPickerModal" -v
```

Expected: all 4 tests PASS.

- [ ] **Step 8.5: Commit**

```bash
git add internal/tui/env_picker.go internal/tui/env_picker_test.go
git commit -m "feat(GPOD-117): add EnvPickerModal with prod warning"
```

---

## Task 9: Wire App + update main.go

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `cmd/getpod/main.go`

- [ ] **Step 9.1: Replace internal/tui/app.go**

Replace the entire file with:

```go
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
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
		if a.issueList != nil {
			a.issueList.Update(msg) //nolint — propagate size, ignore returned model
		}
		if a.issueDetail != nil {
			a.issueDetail.Update(msg) //nolint
		}
		return a, nil

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

	// ── Navigation messages ────────────────────────────────────────────────

	case IssueSelectedMsg:
		a.issueDetail = a.createDetailModel(msg.Issue)
		a.screen = screenIssueDetail
		return a, nil

	case NavigateBackMsg:
		a.screen = screenIssueList
		a.issueDetail = nil
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
			a.styles.HelpKey.Render("[w]") + " " + a.styles.HelpDesc.Render("Repos"),
			a.styles.HelpKey.Render("[x]") + " " + a.styles.HelpDesc.Render("Workspace"),
			a.styles.HelpKey.Render("[e]") + " " + a.styles.HelpDesc.Render("Env"),
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
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
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
```

- [ ] **Step 9.2: Update cmd/getpod/main.go to open the store and pass it to NewApp**

Replace only the `runTUI` function and add the store import:

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
	"github.com/CloudCraft-Studio/getpod-cli/internal/tui"
)

var (
	cfgFile string
	cfg     *config.Config
	reg     *plugin.Registry
)

var rootCmd = &cobra.Command{
	Use:   "getpod",
	Short: "Developer workflow CLI",
	Long:  "GetPod CLI — unified developer workbench",
	RunE:  runTUI,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		skipConfig := map[string]bool{
			"version":     true,
			"config init": true,
		}
		key := cmd.CommandPath()
		if len(key) > 7 {
			key = key[7:]
		}
		if skipConfig[key] {
			return nil
		}

		path := cfgFile
		if path == "" {
			if envPath := os.Getenv("GETPOD_CONFIG"); envPath != "" {
				path = envPath
			}
		}

		var err error
		cfg, err = config.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar la config: %v\n", err)
			return nil
		}

		s, err := state.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar el estado actual: %v\n", err)
			return nil
		}

		active, err := s.Resolve(cfg)
		if err != nil {
			if key == "client list" || key == "client use" ||
				key == "workspace list" || key == "workspace use" ||
				key == "context list" || key == "context use" ||
				key == "config show" {
				return nil
			}
			fmt.Fprintf(os.Stderr, "⚠ Contexto incompleto: %v\n", err)
			return nil
		}

		if err := reg.SetupAll(*active); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Error en inicialización de plugins: %v\n", err)
		}

		for _, pCmd := range reg.AllCommands() {
			cmd.Root().AddCommand(pCmd)
		}

		return nil
	},
}

func runTUI(cmd *cobra.Command, args []string) error {
	if cfg == nil {
		path := cfgFile
		if path == "" {
			if envPath := os.Getenv("GETPOD_CONFIG"); envPath != "" {
				path = envPath
			}
		}
		var err error
		cfg, err = config.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar la config: %v\n", err)
			cfg = config.DefaultConfig()
		}
	}

	db, err := store.NewStore(store.DefaultDBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ No se pudo abrir la base de datos: %v\n", err)
		db = nil // App handles nil db gracefully
	}
	if db != nil {
		defer db.Close()
	}

	app := tui.NewApp(cfg, reg, db)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"ruta a la config (default: ~/.getpod/config.yaml)",
	)
	reg = plugin.NewRegistry()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 9.3: Build and run all tests**

```bash
go build ./...
```

Expected: no compilation errors.

```bash
go test ./... -v 2>&1 | tail -40
```

Expected: all tests PASS, no failures.

- [ ] **Step 9.4: Smoke test the TUI**

```bash
go run ./cmd/getpod/
```

Expected: TUI opens, shows issue list (loading state), [r] triggers fetch (no plugin → empty), client buttons respond, Esc toggles client focus, q quits.

- [ ] **Step 9.5: Commit**

```bash
git add internal/tui/app.go cmd/getpod/main.go
git commit -m "feat(GPOD-117): wire App with sub-models, modal overlay, and store integration"
```

---

## Self-Review

**Spec coverage check:**

| Requirement | Task |
|---|---|
| Lista de issues del client activo con scroll y filtro | Task 4 — IssueListModel |
| Auto-fetch al abrir sin cache; refresh manual [r] | Task 4 — loadCachedCmd + refreshCmd |
| Vista detalle: metadata, description scroll, work context, actions | Task 5 — IssueDetailModel |
| Repo picker modal: multi-select, filtro, fetch dinámico | Task 6 — RepoPickerModal |
| Workspace picker modal: lista desde config | Task 7 — WorkspacePickerModal |
| Environment picker modal: AWS info, warning prod | Task 8 — EnvPickerModal |
| Ready indicator | Task 5 — IsReady() + renderWorkContext |
| Work context persiste inmediatamente | Task 5 — persistWorkContextCmd |
| Actions habilitadas/deshabilitadas | Task 5 — renderActions |
| Environment cambia sin resetear repos ni workspace | Task 5 — EnvSelectedMsg handler |
| Auto-selección con un solo workspace/env | Task 9 — createDetailModel |
| [Esc] con modal cierra modal; sin modal en detalle vuelve a lista | Task 9 — App.Update [Esc] routing |
| Migración SQLite tabla issues + índice | Task 1 — migrations.go |
| PlanningPlugin y RepoPlugin interfaces | Task 2 — planning.go |
| IssuesFetchedMsg con campo Client | Task 3 — msgs.go |

All 16 acceptance criteria covered.
