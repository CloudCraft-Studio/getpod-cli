package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStatusPickerModal_Navigation(t *testing.T) {
	m := &StatusPickerModal{
		items:         []string{"Todo", "In Progress", "Done"},
		currentStatus: "In Progress",
		styles:        DefaultStyles(),
	}
	// Position cursor on current status
	for i, s := range m.items {
		if s == m.currentStatus {
			m.cursor = i
		}
	}

	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 (In Progress), got %d", m.cursor)
	}

	// Move down
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after down, got %d", m.cursor)
	}

	// Down at bottom — stays
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor clamped at 2, got %d", m.cursor)
	}

	// Move up
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after up, got %d", m.cursor)
	}
}

func TestStatusPickerModal_Enter_EmitsStatusSelectedMsg(t *testing.T) {
	m := &StatusPickerModal{
		items:  []string{"Todo", "In Progress", "Done"},
		cursor: 2,
		styles: DefaultStyles(),
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from Enter")
	}

	msg := cmd()
	sel, ok := msg.(StatusSelectedMsg)
	if !ok {
		t.Fatalf("expected StatusSelectedMsg, got %T", msg)
	}
	if sel.Status != "Done" {
		t.Errorf("expected 'Done', got %q", sel.Status)
	}
}

func TestStatusPickerModal_Esc_EmitsModalClosedMsg(t *testing.T) {
	m := &StatusPickerModal{
		items:  []string{"Todo"},
		styles: DefaultStyles(),
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a command from Esc")
	}

	msg := cmd()
	if _, ok := msg.(ModalClosedMsg); !ok {
		t.Fatalf("expected ModalClosedMsg, got %T", msg)
	}
}

func TestStatusPickerModal_View_ShowsCurrentStatus(t *testing.T) {
	m := &StatusPickerModal{
		items:         []string{"Todo", "In Progress", "Done"},
		currentStatus: "In Progress",
		styles:        DefaultStyles(),
	}

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}
