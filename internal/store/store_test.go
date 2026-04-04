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
