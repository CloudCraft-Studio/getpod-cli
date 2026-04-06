package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPRModal_PreFilledValues(t *testing.T) {
	m := NewPRModal("LULO-123", "Fix login bug", "qa", "lulo-x", DefaultStyles())

	title := string(m.title)
	if !strings.Contains(title, "LULO-123") {
		t.Errorf("expected title to contain issue key, got %q", title)
	}

	body := string(m.body)
	if !strings.Contains(body, "lulo-x") {
		t.Errorf("expected body to contain workspace, got %q", body)
	}
	if !strings.Contains(body, "qa") {
		t.Errorf("expected body to contain environment, got %q", body)
	}
}

func TestPRModal_BaseBranchFromEnv_QA(t *testing.T) {
	m := NewPRModal("X-1", "title", "qa", "ws", DefaultStyles())
	// qa → develop
	if baseBranchOptions[m.branchIdx] != "develop" {
		t.Errorf("expected 'develop' for qa, got %q", baseBranchOptions[m.branchIdx])
	}
}

func TestPRModal_BaseBranchFromEnv_Prod(t *testing.T) {
	m := NewPRModal("X-1", "title", "prod", "ws", DefaultStyles())
	// prod → main
	if baseBranchOptions[m.branchIdx] != "main" {
		t.Errorf("expected 'main' for prod, got %q", baseBranchOptions[m.branchIdx])
	}
}

func TestPRModal_TabCyclesFields(t *testing.T) {
	m := NewPRModal("X-1", "title", "qa", "ws", DefaultStyles())

	if m.field != 0 {
		t.Errorf("expected field 0, got %d", m.field)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 1 {
		t.Errorf("expected field 1, got %d", m.field)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 2 {
		t.Errorf("expected field 2, got %d", m.field)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 0 {
		t.Errorf("expected wrap to 0, got %d", m.field)
	}
}

func TestPRModal_CtrlS_Submits(t *testing.T) {
	m := NewPRModal("X-1", "title", "qa", "ws", DefaultStyles())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("expected a command from Ctrl+S")
	}

	msg := cmd()
	sub, ok := msg.(PRSubmitMsg)
	if !ok {
		t.Fatalf("expected PRSubmitMsg, got %T", msg)
	}
	if !strings.Contains(sub.Title, "X-1") {
		t.Errorf("expected title to contain 'X-1', got %q", sub.Title)
	}
	if sub.BaseBranch != "develop" {
		t.Errorf("expected base 'develop', got %q", sub.BaseBranch)
	}
}

func TestPRModal_Esc_EmitsModalClosed(t *testing.T) {
	m := NewPRModal("X-1", "title", "qa", "ws", DefaultStyles())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cmd from Esc")
	}
	msg := cmd()
	if _, ok := msg.(ModalClosedMsg); !ok {
		t.Fatalf("expected ModalClosedMsg, got %T", msg)
	}
}

func TestPRModal_BranchSelector(t *testing.T) {
	m := NewPRModal("X-1", "title", "qa", "ws", DefaultStyles())
	m.field = 2 // base branch field

	// qa defaults to "develop" (index 0)
	if m.branchIdx != 0 {
		t.Fatalf("expected branchIdx 0, got %d", m.branchIdx)
	}

	m.handleBranchKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.branchIdx != 1 {
		t.Errorf("expected branchIdx 1. got %d", m.branchIdx)
	}
}
