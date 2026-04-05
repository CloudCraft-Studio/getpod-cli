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
