package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

func testConfigWithWorkspaces() *config.Config {
	return &config.Config{
		Clients: map[string]config.ClientConfig{
			"lulo": {
				Workspaces: map[string]config.WorkspaceConfig{
					"lulo-x": {
						DisplayName: "Lulo X",
						Contexts:    map[string]config.ContextConfig{"qa": {}, "stg": {}},
					},
					"lulo-business": {
						DisplayName: "Lulo Business",
						Contexts:    map[string]config.ContextConfig{"qa": {}},
					},
				},
			},
		},
	}
}

func TestWorkspacePickerModal_BuildsItemsFromConfig(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	if len(m.items) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(m.items))
	}
}

func TestWorkspacePickerModal_ItemsSortedByName(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	if m.items[0].name != "lulo-business" {
		t.Errorf("expected lulo-business first (alphabetical), got %q", m.items[0].name)
	}
}

func TestWorkspacePickerModal_EnvCountCorrect(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	var luloX workspaceItem
	for _, it := range m.items {
		if it.name == "lulo-x" {
			luloX = it
		}
	}
	if luloX.envCount != 2 {
		t.Errorf("lulo-x should have 2 envs, got %d", luloX.envCount)
	}
}

func TestWorkspacePickerModal_DisplayNameFallback(t *testing.T) {
	cfg := &config.Config{
		Clients: map[string]config.ClientConfig{
			"c": {Workspaces: map[string]config.WorkspaceConfig{
				"ws-key": {}, // no DisplayName
			}},
		},
	}
	m := NewWorkspacePickerModal(cfg, "c", DefaultStyles())
	if m.items[0].displayName != "ws-key" {
		t.Errorf("expected displayName to fall back to key, got %q", m.items[0].displayName)
	}
}

func TestWorkspacePickerModal_EnterEmitsWorkspaceSelectedMsg(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	// items sorted: lulo-business (0), lulo-x (1)
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(enter)
	if cmd == nil {
		t.Fatal("expected cmd from enter, got nil")
	}
	msg := cmd()
	ws, ok := msg.(WorkspaceSelectedMsg)
	if !ok {
		t.Fatalf("expected WorkspaceSelectedMsg, got %T", msg)
	}
	if ws.Workspace != "lulo-business" {
		t.Errorf("expected lulo-business (first sorted), got %q", ws.Workspace)
	}
}

func TestWorkspacePickerModal_EscEmitsModalClosedMsg(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	esc := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := m.Update(esc)
	if cmd == nil {
		t.Fatal("expected cmd from esc, got nil")
	}
	msg := cmd()
	if _, ok := msg.(ModalClosedMsg); !ok {
		t.Fatalf("expected ModalClosedMsg, got %T", msg)
	}
}

func TestWorkspacePickerModal_NavigationClamped(t *testing.T) {
	m := NewWorkspacePickerModal(testConfigWithWorkspaces(), "lulo", DefaultStyles())
	// cursor starts at 0; up should not go below 0
	up := tea.KeyMsg{Type: tea.KeyUp}
	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 when pressing up at top, got %d", m.cursor)
	}
	// down twice should stop at len(items)-1 = 1
	down := tea.KeyMsg{Type: tea.KeyDown}
	m.Update(down)
	m.Update(down)
	m.Update(down)
	if m.cursor != 1 {
		t.Errorf("cursor should clamp at 1 (last item), got %d", m.cursor)
	}
}

func TestWorkspacePickerModal_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Clients: map[string]config.ClientConfig{"c": {}},
	}
	m := NewWorkspacePickerModal(cfg, "c", DefaultStyles())
	if len(m.items) != 0 {
		t.Errorf("expected 0 items for empty workspaces, got %d", len(m.items))
	}
}
