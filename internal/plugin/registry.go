package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// ActiveContext contiene el contexto resuelto que está activo en la sesión actual.
// Es el resultado de resolver client + workspace + context desde el state.
type ActiveContext struct {
	ClientName    string
	Client        config.ClientConfig
	WorkspaceName string
	Workspace     config.WorkspaceConfig
	ContextName   string
	Context       config.ContextConfig
}

// Registry gestiona el ciclo de vida de todos los plugins registrados.
// Es thread-safe y compiled-in: los plugins se registran en main.go directamente.
// No usa dynamic loading, gRPC ni hashicorp/go-plugin — un solo binario.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	active  []string // nombres de plugins activos para el contexto actual
}

// NewRegistry crea un Registry vacío listo para recibir plugins.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register añade un plugin al registry por su Name().
// Si ya existe uno con el mismo nombre, lo sobreescribe.
func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[p.Name()] = p
}

// Get retorna un plugin por nombre.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

// SetupAll inicializa los plugins declarados en el cliente activo.
// Para cada plugin listado en active.Client.Plugins:
//  1. Busca el plugin en el registry (si no está registrado, lo omite con warning)
//  2. Hace merge de credenciales del cliente con la config del context activo
//  3. Llama Setup() con el mapa merged
//
// Los plugins del context que no estén en client.Plugins son ignorados.
func (r *Registry) SetupAll(active ActiveContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.active = make([]string, 0, len(active.Client.Plugins))
	var errs []error

	for pluginName, clientCfg := range active.Client.Plugins {
		p, ok := r.plugins[pluginName]
		if !ok {
			slog.Warn("plugin declarado en config pero no registrado en la CLI",
				"plugin", pluginName,
				"client", active.ClientName,
			)
			continue
		}

		// Merge: credenciales del cliente + config específica del context
		merged := make(map[string]string, len(clientCfg))
		for k, v := range clientCfg {
			merged[k] = v
		}
		if ctxCfg, ok := active.Context[pluginName]; ok {
			for k, v := range ctxCfg {
				merged[k] = v // el context puede sobreescribir valores del cliente
			}
		}

		if err := p.Setup(merged); err != nil {
			errs = append(errs, fmt.Errorf("setup del plugin %q falló: %w", pluginName, err))
			continue
		}

		r.active = append(r.active, pluginName)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errores durante setup: %v", errs)
	}
	return nil
}

// CollectAll recolecta métricas de todos los plugins activos.
// Si un plugin falla, se loguea el error y se continúa con los demás (skip on error).
func (r *Registry) CollectAll(ctx context.Context, since time.Time) ([]Metric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []Metric
	for _, name := range r.active {
		p, ok := r.plugins[name]
		if !ok {
			continue
		}
		metrics, err := p.CollectMetrics(ctx, since)
		if err != nil {
			slog.Error("error recolectando métricas", "plugin", name, "error", err)
			continue // skip on error
		}
		all = append(all, metrics...)
	}
	return all, nil
}

// DeriveAllSkills deriva habilidades de todos los plugins activos.
// Si un plugin falla, se loguea el error y se continúa (skip on error).
func (r *Registry) DeriveAllSkills(ctx context.Context) ([]Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []Skill
	for _, name := range r.active {
		p, ok := r.plugins[name]
		if !ok {
			continue
		}
		skills, err := p.DeriveSkills(ctx)
		if err != nil {
			slog.Error("error derivando skills", "plugin", name, "error", err)
			continue // skip on error
		}
		all = append(all, skills...)
	}
	return all, nil
}

// AllCommands retorna todos los subcomandos Cobra de los plugins activos.
// Se llaman en main.go para registrarlos en el root command.
func (r *Registry) AllCommands() []*cobra.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var cmds []*cobra.Command
	for _, name := range r.active {
		p, ok := r.plugins[name]
		if !ok {
			continue
		}
		cmds = append(cmds, p.Commands()...)
	}
	return cmds
}

// ActivePlugins retorna los nombres de los plugins activos en el contexto actual.
func (r *Registry) ActivePlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, len(r.active))
	copy(result, r.active)
	return result
}
