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

func TestUpdateWorkContext_ErrorOnNotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.UpdateWorkContext(context.Background(), "nonexistent", nil, "", "")
	if err == nil {
		t.Error("expected error when updating non-existent issue, got nil")
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
