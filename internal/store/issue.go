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
	rawDataNull := sql.NullString{
		String: string(ir.RawData),
		Valid:  len(ir.RawData) > 0,
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
		string(labelsJSON), rawDataNull, encodeTime(ir.FetchedAt),
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
func (s *Store) UpdateWorkContext(ctx context.Context, id string, repos []string, workspace, environment string) error {
	reposJSON, err := json.Marshal(repos)
	if err != nil {
		return fmt.Errorf("encoding repos: %w", err)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE issues SET repos = ?, workspace = ?, environment = ? WHERE id = ?`,
		string(reposJSON), nullableString(workspace), nullableString(environment), id,
	)
	if err != nil {
		return fmt.Errorf("updating work context for %q: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("issue %q not found", id)
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
		rawDataNull    sql.NullString
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
		&priorityNull, &ir.Description, &labelsStr, &rawDataNull, &fetchedAtStr,
		&reposNull, &workspaceNull, &envNull, &notesNull, &startedAtNull, &finishedAtNull,
	); err != nil {
		return nil, fmt.Errorf("scanning issue: %w", err)
	}

	ir.Priority = priorityNull.String
	ir.RawData = json.RawMessage(rawDataNull.String)
	ir.Workspace = workspaceNull.String
	ir.Environment = envNull.String
	ir.Notes = notesNull.String

	if err := json.Unmarshal([]byte(labelsStr), &ir.Labels); err != nil {
		ir.Labels = nil
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
