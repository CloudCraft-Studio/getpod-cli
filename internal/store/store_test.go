package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNewStore_CreatesTablesIdempotent(t *testing.T) {
	s1, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("first NewStore: %v", err)
	}
	s1.Close()

	s2, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("second NewStore: %v", err)
	}
	s2.Close()
}

func TestDefaultDBPath_ContainsGetpod(t *testing.T) {
	p := store.DefaultDBPath()
	if p == "" {
		t.Fatal("DefaultDBPath returned empty string")
	}
	if len(p) < 10 {
		t.Fatalf("path too short: %q", p)
	}
}

func TestSaveEvents_NoError(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	events := []plugin.Metric{
		{Plugin: "jira", Event: "issue_closed", Timestamp: now, Meta: map[string]string{"key": "PROJ-1"}},
		{Plugin: "jira", Event: "pr_merged", Timestamp: now.Add(time.Minute), Meta: map[string]string{"pr": "42"}},
	}

	if err := s.SaveEvents(ctx, events); err != nil {
		t.Fatalf("SaveEvents: %v", err)
	}
}

func TestSaveEvents_EmptySliceIsNoop(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveEvents(context.Background(), []plugin.Metric{}); err != nil {
		t.Fatalf("SaveEvents with empty slice: %v", err)
	}
}

func TestGetUnsynced_ReturnsDataCorrectly(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	events := []plugin.Metric{
		{Plugin: "jira", Event: "issue_closed", Timestamp: now, Meta: map[string]string{"key": "PROJ-1"}},
	}
	if err := s.SaveEvents(ctx, events); err != nil {
		t.Fatalf("SaveEvents: %v", err)
	}

	got, err := s.GetUnsynced(ctx, 10)
	if err != nil {
		t.Fatalf("GetUnsynced: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Plugin != "jira" {
		t.Errorf("wrong plugin: %q", got[0].Plugin)
	}
	if got[0].Event != "issue_closed" {
		t.Errorf("wrong event: %q", got[0].Event)
	}
	if !got[0].Timestamp.Equal(now) {
		t.Errorf("timestamp mismatch: got %v, want %v", got[0].Timestamp, now)
	}
	if got[0].Meta["key"] != "PROJ-1" {
		t.Errorf("meta not preserved: %v", got[0].Meta)
	}
	if got[0].Synced {
		t.Error("expected synced=false")
	}
}

func TestGetUnsynced_RespectsLimit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	events := []plugin.Metric{
		{Plugin: "linear", Event: "e1", Timestamp: now, Meta: map[string]string{}},
		{Plugin: "linear", Event: "e2", Timestamp: now.Add(time.Second), Meta: map[string]string{}},
		{Plugin: "linear", Event: "e3", Timestamp: now.Add(2 * time.Second), Meta: map[string]string{}},
	}
	if err := s.SaveEvents(ctx, events); err != nil {
		t.Fatalf("SaveEvents: %v", err)
	}

	got, err := s.GetUnsynced(ctx, 2)
	if err != nil {
		t.Fatalf("GetUnsynced: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Event != "e1" || got[1].Event != "e2" {
		t.Errorf("wrong order: %q %q", got[0].Event, got[1].Event)
	}
}

func TestMarkSynced_UpdatesBatch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	events := []plugin.Metric{
		{Plugin: "github", Event: "pr_opened", Timestamp: now, Meta: map[string]string{}},
		{Plugin: "github", Event: "pr_merged", Timestamp: now.Add(time.Minute), Meta: map[string]string{}},
	}
	if err := s.SaveEvents(ctx, events); err != nil {
		t.Fatalf("SaveEvents: %v", err)
	}

	unsynced, err := s.GetUnsynced(ctx, 10)
	if err != nil {
		t.Fatalf("GetUnsynced: %v", err)
	}
	if len(unsynced) != 2 {
		t.Fatalf("expected 2 unsynced, got %d", len(unsynced))
	}

	ids := []int64{unsynced[0].ID, unsynced[1].ID}
	if err := s.MarkSynced(ctx, ids, now.Add(time.Hour)); err != nil {
		t.Fatalf("MarkSynced: %v", err)
	}

	remaining, err := s.GetUnsynced(ctx, 10)
	if err != nil {
		t.Fatalf("GetUnsynced after MarkSynced: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected 0 unsynced after mark, got %d", len(remaining))
	}
}

func TestMarkSynced_EmptySliceIsNoop(t *testing.T) {
	s := newTestStore(t)
	if err := s.MarkSynced(context.Background(), []int64{}, time.Now()); err != nil {
		t.Fatalf("MarkSynced with empty ids: %v", err)
	}
}
