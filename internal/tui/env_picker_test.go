package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

func testConfigWithEnvs() *config.Config {
	return &config.Config{
		Clients: map[string]config.ClientConfig{
			"lulo": {
				Workspaces: map[string]config.WorkspaceConfig{
					"lulo-x": {
						Contexts: map[string]config.ContextConfig{
							"qa":   {"aws": {"aws_account": "111111111111", "aws_region": "us-east-1"}},
							"stg":  {"aws": {"aws_account": "222222222222", "aws_region": "us-east-1"}},
							"prod": {"aws": {"aws_account": "333333333333", "aws_region": "us-east-1"}},
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

func TestEnvPickerModal_EnterEmitsEnvSelectedMsg(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	// items sorted: prod(0), qa(1), stg(2) — alphabetical
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(enter)
	if cmd == nil {
		t.Fatal("expected cmd from enter, got nil")
	}
	msg := cmd()
	ev, ok := msg.(EnvSelectedMsg)
	if !ok {
		t.Fatalf("expected EnvSelectedMsg, got %T", msg)
	}
	if ev.Env != "prod" {
		t.Errorf("expected prod (first sorted), got %q", ev.Env)
	}
}

func TestEnvPickerModal_EscEmitsModalClosedMsg(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	esc := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := m.Update(esc)
	if cmd == nil {
		t.Fatal("expected cmd from esc, got nil")
	}
	if _, ok := cmd().(ModalClosedMsg); !ok {
		t.Error("expected ModalClosedMsg")
	}
}

func TestEnvPickerModal_NavigationAndSelection(t *testing.T) {
	m := NewEnvPickerModal(testConfigWithEnvs(), "lulo", "lulo-x", DefaultStyles())
	// sorted: prod(0), qa(1), stg(2)

	// down moves cursor to qa
	down := tea.KeyMsg{Type: tea.KeyDown}
	m.Update(down)
	if m.cursor != 1 {
		t.Fatalf("expected cursor=1 after down, got %d", m.cursor)
	}

	// enter at cursor=1 emits qa
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(enter)
	ev := cmd().(EnvSelectedMsg)
	if ev.Env != "qa" {
		t.Errorf("expected qa after navigating down, got %q", ev.Env)
	}

	// up from cursor=1 returns to 0
	m.cursor = 1
	up := tea.KeyMsg{Type: tea.KeyUp}
	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("expected cursor=0 after up, got %d", m.cursor)
	}

	// down beyond last item clamps at 2
	m.cursor = 2
	m.Update(down)
	if m.cursor != 2 {
		t.Errorf("expected cursor clamped at 2, got %d", m.cursor)
	}
}
