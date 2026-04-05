package tui

import (
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
)

func makeTestRepos() []plugin.Repo {
	return []plugin.Repo{
		{Name: "repo-backend", Source: "github", Language: "Go"},
		{Name: "repo-frontend", Source: "github", Language: "TypeScript"},
		{Name: "infra-terraform", Source: "github", Language: "HCL"},
	}
}

func TestRepoPickerModal_ToggleSelection(t *testing.T) {
	m := &RepoPickerModal{
		items:    makeTestRepos(),
		selected: make(map[string]bool),
		styles:   DefaultStyles(),
	}
	m.applyFilter()

	// Toggle on
	m.selected[m.filtered[0].Name] = !m.selected[m.filtered[0].Name]
	if !m.selected["repo-backend"] {
		t.Error("expected repo-backend selected after toggle")
	}

	// Toggle off
	m.selected[m.filtered[0].Name] = !m.selected[m.filtered[0].Name]
	if m.selected["repo-backend"] {
		t.Error("expected repo-backend deselected after second toggle")
	}
}

func TestRepoPickerModal_ApplyFilter(t *testing.T) {
	m := &RepoPickerModal{
		items:    makeTestRepos(),
		selected: make(map[string]bool),
		styles:   DefaultStyles(),
	}
	m.filter = "terra"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match, got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "infra-terraform" {
		t.Errorf("wrong match: %q", m.filtered[0].Name)
	}
}

func TestRepoPickerModal_ApplyFilter_EmptyReturnsAll(t *testing.T) {
	m := &RepoPickerModal{
		items:    makeTestRepos(),
		selected: make(map[string]bool),
		styles:   DefaultStyles(),
	}
	m.filter = ""
	m.applyFilter()

	if len(m.filtered) != len(m.items) {
		t.Errorf("expected %d, got %d", len(m.items), len(m.filtered))
	}
}

func TestRepoPickerModal_PreselectedMarked(t *testing.T) {
	m := NewRepoPickerModal(nil, []string{"repo-backend"}, DefaultStyles())
	if !m.selected["repo-backend"] {
		t.Error("expected preselected repo to be marked")
	}
}
