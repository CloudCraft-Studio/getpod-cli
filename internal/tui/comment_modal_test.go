package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommentModal_Typing(t *testing.T) {
	m := NewCommentModal(DefaultStyles())

	// Type "hello"
	for _, r := range "hello" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	got := string(m.body)
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestCommentModal_Backspace(t *testing.T) {
	m := NewCommentModal(DefaultStyles())
	m.body = []rune("hello")
	m.cursor = 5

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	got := string(m.body)
	if got != "hell" {
		t.Errorf("expected 'hell', got %q", got)
	}
}

func TestCommentModal_CtrlS_SubmitsNonEmpty(t *testing.T) {
	m := NewCommentModal(DefaultStyles())
	m.body = []rune("my comment")
	m.cursor = len(m.body)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("expected a command from Ctrl+S")
	}

	msg := cmd()
	sub, ok := msg.(CommentSubmitMsg)
	if !ok {
		t.Fatalf("expected CommentSubmitMsg, got %T", msg)
	}
	if sub.Body != "my comment" {
		t.Errorf("expected 'my comment', got %q", sub.Body)
	}
}

func TestCommentModal_CtrlS_IgnoresEmpty(t *testing.T) {
	m := NewCommentModal(DefaultStyles())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Error("expected nil command for empty comment")
	}
}

func TestCommentModal_Esc_EmitsModalClosed(t *testing.T) {
	m := NewCommentModal(DefaultStyles())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a command from Esc")
	}

	msg := cmd()
	if _, ok := msg.(ModalClosedMsg); !ok {
		t.Fatalf("expected ModalClosedMsg, got %T", msg)
	}
}

func TestCommentModal_Enter_AddsNewline(t *testing.T) {
	m := NewCommentModal(DefaultStyles())
	m.body = []rune("line1")
	m.cursor = 5

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := string(m.body)
	if got != "line1\n" {
		t.Errorf("expected 'line1\\n', got %q", got)
	}
}
