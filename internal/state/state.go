package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"gopkg.in/yaml.v3"
)

const (
	DefaultStateDir  = ".getpod"
	DefaultStateFile = "state.yaml"
)

// State persiste el contexto activo del desarrollador entre sesiones.
type State struct {
	ActiveClient    string    `yaml:"active_client"`
	ActiveWorkspace string    `yaml:"active_workspace"`
	ActiveContext   string    `yaml:"active_context"`
	LastSwitched    time.Time `yaml:"last_switched"`
}

// DefaultStatePath retorna la ruta: ~/.getpod/state.yaml
func DefaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(DefaultStateDir, DefaultStateFile)
	}
	return filepath.Join(home, DefaultStateDir, DefaultStateFile)
}

// Load lee el estado desde el disco. Si no existe, retorna un estado vacío.
func Load() (*State, error) {
	path := DefaultStatePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &State{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo state: %w", err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("error parseando state: %w", err)
	}

	return &s, nil
}

// Save persiste el estado en el disco.
func (s *State) Save() error {
	path := DefaultStatePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// UseClient selecciona un cliente y limpia workspace/context.
// Atomically updates state and persists to disk.
func (s *State) UseClient(name string) error {
	s.ActiveClient = name
	s.ActiveWorkspace = ""
	s.ActiveContext = ""
	s.LastSwitched = time.Now()
	return s.Save()
}

// UseWorkspace selecciona un workspace y limpia context.
// Atomically updates state and persists to disk.
func (s *State) UseWorkspace(name string) error {
	s.ActiveWorkspace = name
	s.ActiveContext = ""
	s.LastSwitched = time.Now()
	return s.Save()
}

// UseContext selecciona un context.
// Atomically updates state and persists to disk.
func (s *State) UseContext(name string) error {
	s.ActiveContext = name
	s.LastSwitched = time.Now()
	return s.Save()
}

// Resolve valida que la jerarquía guardada en el state exista en la config actual.
// Si todo es válido, construye un plugin.ActiveContext listo para el Registry.
func (s *State) Resolve(cfg *config.Config) (*plugin.ActiveContext, error) {
	if s.ActiveClient == "" {
		return nil, fmt.Errorf("no hay un cliente activo. Usa 'getpod client use <nombre>'")
	}

	client, ok := cfg.Clients[s.ActiveClient]
	if !ok {
		return nil, fmt.Errorf("el cliente activo %q no existe en la configuración", s.ActiveClient)
	}

	if s.ActiveWorkspace == "" {
		return nil, fmt.Errorf("no hay un workspace activo para el cliente %q. Usa 'getpod workspace use <nombre>'", s.ActiveClient)
	}

	ws, ok := client.Workspaces[s.ActiveWorkspace]
	if !ok {
		return nil, fmt.Errorf("el workspace %q no existe para el cliente %q", s.ActiveWorkspace, s.ActiveClient)
	}

	if s.ActiveContext == "" {
		return nil, fmt.Errorf("no hay un contexto activo para el workspace %q. Usa 'getpod context use <nombre>'", s.ActiveWorkspace)
	}

	ctx, ok := ws.Contexts[s.ActiveContext]
	if !ok {
		return nil, fmt.Errorf("el contexto %q no existe para el workspace %q", s.ActiveContext, s.ActiveWorkspace)
	}

	return &plugin.ActiveContext{
		ClientName:    s.ActiveClient,
		Client:        client,
		WorkspaceName: s.ActiveWorkspace,
		Workspace:     ws,
		ContextName:   s.ActiveContext,
		Context:       ctx,
	}, nil
}
