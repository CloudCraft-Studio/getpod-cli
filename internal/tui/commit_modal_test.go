package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommitModal_DefaultScope(t *testing.T) {
	m := NewCommitModal("LULO-1234", DefaultStyles())
	got := string(m.scope)
	if got != "lulo-1234" {
		t.Errorf("expected 'lulo-1234', got %q", got)
	}
}

func TestCommitModal_TabCyclesFields(t *testing.T) {
	m := NewCommitModal("X-1", DefaultStyles())
	if m.field != 0 {
		t.Errorf("expected field 0, got %d", m.field)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 1 {
		t.Errorf("expected field 1 after tab, got %d", m.field)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 2 {
		t.Errorf("expected field 2 after tab, got %d", m.field)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 0 {
		t.Errorf("expected field 0 after wrap, got %d", m.field)
	}
}

func TestCommitModal_TypeSelector(t *testing.T) {
	m := NewCommitModal("X-1", DefaultStyles())
	if m.typeIdx != 0 {
		t.Errorf("expected typeIdx 0, got %d", m.typeIdx)
	}

	// Move down to "fix"
	m.handleTypeKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.typeIdx != 1 {
		t.Errorf("expected typeIdx 1, got %d", m.typeIdx)
	}
	if commitTypes[m.typeIdx] != "fix" {
		t.Errorf("expected 'fix', got %q", commitTypes[m.typeIdx])
	}
}

func TestCommitModal_CtrlS_SubmitsWithMessage(t *testing.T) {
	m := NewCommitModal("X-1", DefaultStyles())
	m.field = 2
	m.message = []rune("add feature")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("expected a command from Ctrl+S")
	}

	msg := cmd()
	sub, ok := msg.(CommitSubmitMsg)
	if !ok {
		t.Fatalf("expected CommitSubmitMsg, got %T", msg)
	}
	if sub.Type != "feat" {
		t.Errorf("expected type 'feat', got %q", sub.Type)
	}
	if sub.Scope != "x-1" {
		t.Errorf("expected scope 'x-1', got %q", sub.Scope)
	}
	if sub.Message != "add feature" {
		t.Errorf("expected message 'add feature', got %q", sub.Message)
	}
}

func TestCommitModal_CtrlS_IgnoresEmptyMessage(t *testing.T) {
	m := NewCommitModal("X-1", DefaultStyles())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Error("expected nil command for empty message")
	}
}

func TestCommitModal_Esc_EmitsModalClosed(t *testing.T) {
	m := NewCommitModal("X-1", DefaultStyles())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a command from Esc")
	}
	msg := cmd()
	if _, ok := msg.(ModalClosedMsg); !ok {
		t.Fatalf("expected ModalClosedMsg, got %T", msg)
	}
}
