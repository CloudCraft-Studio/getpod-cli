package tui

import (
	"strings"
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

