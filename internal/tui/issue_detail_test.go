package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
							"qa": {
								"aws": {"aws_account": "111111111111", "aws_region": "us-east-1"},
							},
							"prod": {
								"aws": {"aws_account": "333333333333", "aws_region": "us-east-1"},
							},
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
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in missing context, got %q", want, got)
		}
	}
}

func TestIssueDetailModel_MissingContext_OnlyMissingEnv(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Repos: []string{"r"}, Workspace: "ws",
	})
	got := m.missingContext()
	if !strings.Contains(got, "environment") {
		t.Errorf("expected 'environment' in %q", got)
	}
	if strings.Contains(got, "repos") || strings.Contains(got, "workspace") {
		t.Errorf("unexpected items in %q", got)
	}
}

// ── Action keybinding tests (GPOD-118) ─────────────────────────────────────

func TestIssueDetailModel_KeyB_EmitsBranchMsg_WhenReposSet(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Key:   "LULO-123",
		Repos: []string{"repo-a"},
	})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if cmd == nil {
		t.Fatal("expected a command from [b] with repos set")
	}
	msg := cmd()
	if _, ok := msg.(OpenBranchConfirmMsg); !ok {
		t.Fatalf("expected OpenBranchConfirmMsg, got %T", msg)
	}
}

func TestIssueDetailModel_KeyB_NoOp_WhenNoRepos(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if cmd != nil {
		t.Error("expected no command from [b] without repos")
	}
}

func TestIssueDetailModel_KeyC_EmitsCommitMsg(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if cmd == nil {
		t.Fatal("expected a command from [c]")
	}
	msg := cmd()
	if _, ok := msg.(OpenCommitModalMsg); !ok {
		t.Fatalf("expected OpenCommitModalMsg, got %T", msg)
	}
}

func TestIssueDetailModel_KeyR_EmitsPRMsg_WhenReposSet(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Key:   "LULO-123",
		Repos: []string{"repo-a"},
	})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected a command from [r] with repos set")
	}
	msg := cmd()
	if _, ok := msg.(OpenPRModalMsg); !ok {
		t.Fatalf("expected OpenPRModalMsg, got %T", msg)
	}
}

func TestIssueDetailModel_KeyM_EmitsCommentMsg(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("expected a command from [m]")
	}
	msg := cmd()
	if _, ok := msg.(OpenCommentModalMsg); !ok {
		t.Fatalf("expected OpenCommentModalMsg, got %T", msg)
	}
}

func TestIssueDetailModel_KeyS_EmitsStatusMsg(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected a command from [s]")
	}
	msg := cmd()
	if _, ok := msg.(OpenStatusPickerMsg); !ok {
		t.Fatalf("expected OpenStatusPickerMsg, got %T", msg)
	}
}

func TestIssueDetailModel_KeyP_EmitsPlanMsg_WhenReady(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{
		Key:         "LULO-123",
		Repos:       []string{"repo-a"},
		Workspace:   "lulo-x",
		Environment: "qa",
	})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd == nil {
		t.Fatal("expected a command from [p] when ready")
	}
	msg := cmd()
	if _, ok := msg.(OpenPlanAIMsg); !ok {
		t.Fatalf("expected OpenPlanAIMsg, got %T", msg)
	}
}

func TestIssueDetailModel_KeyP_NoOp_WhenNotReady(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd != nil {
		t.Error("expected no command from [p] when not ready")
	}
}

func TestIssueDetailModel_ActionFeedback_Success(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	m.Update(BranchCreatedMsg{Branch: "feature/lulo-123"})

	if !m.actionSuccess {
		t.Error("expected actionSuccess = true")
	}
	if !strings.Contains(m.actionMsg, "feature/lulo-123") {
		t.Errorf("expected branch name in actionMsg, got %q", m.actionMsg)
	}
}

func TestIssueDetailModel_ActionFeedback_Error(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	m.Update(BranchCreatedMsg{Err: fmt.Errorf("repo not found")})

	if m.actionSuccess {
		t.Error("expected actionSuccess = false")
	}
	if !strings.Contains(m.actionMsg, "repo not found") {
		t.Errorf("expected error message in actionMsg, got %q", m.actionMsg)
	}
}

func TestIssueDetailModel_StatusChanged_UpdatesIssueStatus(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123", Status: "Todo"})
	m.Update(StatusChangedMsg{NewStatus: "In Progress"})

	if m.issue.Status != "In Progress" {
		t.Errorf("expected status 'In Progress', got %q", m.issue.Status)
	}
}

func TestIssueDetailModel_ActionFeedback_ClearsOnKeypress(t *testing.T) {
	m := newTestDetailModel(store.IssueRecord{Key: "LULO-123"})
	m.actionMsg = "some feedback"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.actionMsg != "" {
		t.Errorf("expected actionMsg to be cleared, got %q", m.actionMsg)
	}
}
