package store_test

import (
	"testing"

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
