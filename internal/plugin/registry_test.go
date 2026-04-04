package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// makeActiveContext construye un ActiveContext de prueba.
func makeActiveContext(plugins map[string]map[string]string, ctxCfg config.ContextConfig) ActiveContext {
	return ActiveContext{
		ClientName: "test-client",
		Client: config.ClientConfig{
			DisplayName: "Test Client",
			Plugins:     plugins,
		},
		WorkspaceName: "test-ws",
		Workspace: config.WorkspaceConfig{
			DisplayName: "Test Workspace",
		},
		ContextName: "dev",
		Context:     ctxCfg,
	}
}

// TestRegister verifica que un plugin se registra y se recupera por nombre.
func TestRegister(t *testing.T) {
	r := NewRegistry()
	p := newMockPlugin("jira")
	r.Register(p)

	got, ok := r.Get("jira")
	if !ok {
		t.Fatal("esperaba encontrar el plugin 'jira' pero no estaba registrado")
	}
	if got.Name() != "jira" {
		t.Errorf("nombre incorrecto: got %q, want %q", got.Name(), "jira")
	}
}

// TestRegister_NotFound verifica que Get retorna false si el plugin no existe.
func TestRegister_NotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("esperaba que Get retornara false para un plugin no registrado")
	}
}

// TestSetupAll_MergesClientAndContext verifica que Setup recibe el merge
// de credenciales del cliente y la config del context activo.
func TestSetupAll_MergesClientAndContext(t *testing.T) {
	r := NewRegistry()
	jira := newMockPlugin("jira")
	r.Register(jira)

	active := makeActiveContext(
		map[string]map[string]string{
			"jira": {"token": "secret-token", "url": "https://acme.atlassian.net"},
		},
		config.ContextConfig{
			"jira": {"project_key": "BACK", "board_id": "42"},
		},
	)

	if err := r.SetupAll(active); err != nil {
		t.Fatalf("SetupAll falló: %v", err)
	}

	// Verificar merge: debe tener tanto las credenciales como el project_key
	wantKeys := []string{"token", "url", "project_key", "board_id"}
	for _, k := range wantKeys {
		if _, ok := jira.setupCfg[k]; !ok {
			t.Errorf("falta la clave %q en la config merged del plugin", k)
		}
	}
	if jira.setupCfg["token"] != "secret-token" {
		t.Errorf("token incorrecto: got %q", jira.setupCfg["token"])
	}
	if jira.setupCfg["project_key"] != "BACK" {
		t.Errorf("project_key incorrecto: got %q", jira.setupCfg["project_key"])
	}
}

// TestSetupAll_SkipsUnregisteredPlugin verifica que SetupAll no falla si
// un plugin está en la config pero no está registrado en la CLI.
func TestSetupAll_SkipsUnregisteredPlugin(t *testing.T) {
	r := NewRegistry()
	// No registramos ningún plugin

	active := makeActiveContext(
		map[string]map[string]string{
			"bitbucket": {"token": "bb-token"}, // declarado pero no registrado
		},
		config.ContextConfig{},
	)

	// No debe retornar error
	if err := r.SetupAll(active); err != nil {
		t.Fatalf("SetupAll no debería fallar con plugins no registrados: %v", err)
	}
}

// TestCollectAll_SkipsOnError verifica que si un plugin falla en CollectMetrics,
// los demás continúan y sus métricas se acumulan correctamente.
func TestCollectAll_SkipsOnError(t *testing.T) {
	r := NewRegistry()

	// Plugin que falla
	failing := mockWithMetricError("jira", errMockMetric)
	// Plugin que funciona
	ok := newMockPlugin("linear")
	ok.metrics = []Metric{{Plugin: "linear", Event: "issue.created"}}

	r.Register(failing)
	r.Register(ok)

	active := makeActiveContext(
		map[string]map[string]string{
			"jira":   {"token": "t1"},
			"linear": {"api_key": "k1"},
		},
		config.ContextConfig{},
	)
	if err := r.SetupAll(active); err != nil {
		t.Fatalf("SetupAll falló: %v", err)
	}

	metrics, err := r.CollectAll(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("CollectAll retornó error inesperado: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("esperaba 1 métrica (de linear), got %d", len(metrics))
	}
	if metrics[0].Plugin != "linear" {
		t.Errorf("métrica incorrecta: got plugin=%q", metrics[0].Plugin)
	}
}

// TestDeriveAllSkills_SkipsOnError verifica que si un plugin falla en DeriveSkills,
// los demás continúan correctamente.
func TestDeriveAllSkills_SkipsOnError(t *testing.T) {
	r := NewRegistry()

	failing := mockWithSkillError("jira", errMockSkill)
	working := newMockPlugin("linear")
	working.skills = []Skill{{Name: "Go", Confidence: 0.9, Source: "linear"}}

	r.Register(failing)
	r.Register(working)

	active := makeActiveContext(
		map[string]map[string]string{
			"jira":   {"token": "t1"},
			"linear": {"api_key": "k1"},
		},
		config.ContextConfig{},
	)
	if err := r.SetupAll(active); err != nil {
		t.Fatalf("SetupAll falló: %v", err)
	}

	skills, err := r.DeriveAllSkills(context.Background())
	if err != nil {
		t.Fatalf("DeriveAllSkills retornó error inesperado: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("esperaba 1 skill (de linear), got %d", len(skills))
	}
}

// TestAllCommands verifica que AllCommands expone los cobra.Commands de los plugins activos.
func TestAllCommands(t *testing.T) {
	r := NewRegistry()
	p1 := newMockPlugin("jira")
	p2 := newMockPlugin("linear")
	r.Register(p1)
	r.Register(p2)

	active := makeActiveContext(
		map[string]map[string]string{
			"jira":   {"token": "t1"},
			"linear": {"api_key": "k1"},
		},
		config.ContextConfig{},
	)
	if err := r.SetupAll(active); err != nil {
		t.Fatalf("SetupAll falló: %v", err)
	}

	cmds := r.AllCommands()
	if len(cmds) != 2 {
		t.Errorf("esperaba 2 comandos (uno por plugin), got %d", len(cmds))
	}
}
