package tui

import (
	"testing"

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
