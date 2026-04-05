package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
)

var (
	ErrNoStateLoaded     = errors.New("no hay estado cargado")
	ErrNoActiveClient    = errors.New("no hay un cliente activo")
	ErrNoActiveWorkspace = errors.New("no hay un workspace activo")
	ErrNoActiveContext   = errors.New("no hay un contexto activo")
	ErrClientNotFound    = errors.New("el cliente no existe en config")
	ErrWorkspaceNotFound = errors.New("el workspace no existe")
	ErrContextNotFound   = errors.New("el contexto no existe")
)

type ResolvedPluginConfig struct {
	Client    string
	Workspace string
	Context   string
	Plugins   map[string]map[string]string
}

type SessionManager struct {
	cfg  *config.Config
	st   *state.State
	reg  *plugin.Registry
	opts []SessionOption
}

type SessionOption func(*SessionManager)

func WithRegistry(reg *plugin.Registry) SessionOption {
	return func(sm *SessionManager) {
		sm.reg = reg
	}
}

func NewSessionManager(cfg *config.Config, opts ...SessionOption) *SessionManager {
	sm := &SessionManager{cfg: cfg}
	for _, opt := range opts {
		opt(sm)
	}
	return sm
}

func (sm *SessionManager) LoadState() error {
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	sm.st = st
	return nil
}

func (sm *SessionManager) UseClient(name string) error {
	if sm.st == nil {
		sm.st = &state.State{}
	}
	if _, ok := sm.cfg.Clients[name]; !ok {
		return fmt.Errorf("%w: %q", ErrClientNotFound, name)
	}
	return sm.st.UseClient(name)
}

func (sm *SessionManager) UseWorkspace(name string) error {
	if sm.st == nil || sm.st.ActiveClient == "" {
		return ErrNoActiveClient
	}
	client, ok := sm.cfg.Clients[sm.st.ActiveClient]
	if !ok {
		return fmt.Errorf("%w: %q", ErrClientNotFound, sm.st.ActiveClient)
	}
	if _, ok := client.Workspaces[name]; !ok {
		return fmt.Errorf("%w: %q", ErrWorkspaceNotFound, name)
	}
	return sm.st.UseWorkspace(name)
}

func (sm *SessionManager) UseContext(name string) error {
	if sm.st == nil || sm.st.ActiveClient == "" || sm.st.ActiveWorkspace == "" {
		return fmt.Errorf("%w or %w", ErrNoActiveClient, ErrNoActiveWorkspace)
	}
	client, _ := sm.cfg.Clients[sm.st.ActiveClient]
	ws, ok := client.Workspaces[sm.st.ActiveWorkspace]
	if !ok {
		return fmt.Errorf("%w: %q", ErrWorkspaceNotFound, sm.st.ActiveWorkspace)
	}
	if _, ok := ws.Contexts[name]; !ok {
		return fmt.Errorf("%w: %q", ErrContextNotFound, name)
	}

	if err := sm.st.UseContext(name); err != nil {
		return err
	}
	return sm.setupPlugins()
}

func (sm *SessionManager) setupPlugins() error {
	if sm.reg == nil {
		return nil
	}
	activeCtx, err := sm.st.Resolve(sm.cfg)
	if err != nil {
		return err
	}
	return sm.reg.SetupAll(*activeCtx)
}

func (sm *SessionManager) ResolvedConfig() (*ResolvedPluginConfig, error) {
	if sm.st == nil {
		return nil, ErrNoStateLoaded
	}
	if sm.st.ActiveClient == "" {
		return nil, ErrNoActiveClient
	}
	client, ok := sm.cfg.Clients[sm.st.ActiveClient]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrClientNotFound, sm.st.ActiveClient)
	}
	if sm.st.ActiveWorkspace == "" {
		return nil, ErrNoActiveWorkspace
	}
	ws, ok := client.Workspaces[sm.st.ActiveWorkspace]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrWorkspaceNotFound, sm.st.ActiveWorkspace)
	}
	if sm.st.ActiveContext == "" {
		return nil, ErrNoActiveContext
	}
	ctx, ok := ws.Contexts[sm.st.ActiveContext]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrContextNotFound, sm.st.ActiveContext)
	}

	merged := sm.mergePlugins(client, ctx)

	return &ResolvedPluginConfig{
		Client:    sm.st.ActiveClient,
		Workspace: sm.st.ActiveWorkspace,
		Context:   sm.st.ActiveContext,
		Plugins:   merged,
	}, nil
}

func (sm *SessionManager) mergePlugins(client config.ClientConfig, ctx config.ContextConfig) map[string]map[string]string {
	merged := make(map[string]map[string]string)
	for pluginName, clientCreds := range client.Plugins {
		pluginConfig := make(map[string]string, len(clientCreds)+len(ctx[pluginName]))
		for k, v := range clientCreds {
			pluginConfig[k] = v
		}
		if ctxCreds, ok := ctx[pluginName]; ok {
			for k, v := range ctxCreds {
				pluginConfig[k] = v
			}
		}
		merged[pluginName] = pluginConfig
	}
	return merged
}

func (sm *SessionManager) ActivePlugins() []plugin.Plugin {
	if sm.reg == nil {
		return nil
	}
	names := sm.reg.ActivePlugins()
	plugins := make([]plugin.Plugin, 0, len(names))
	for _, name := range names {
		if p, ok := sm.reg.Get(name); ok {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

func (sm *SessionManager) State() *state.State {
	return sm.st
}

func (sm *SessionManager) LastSwitched() time.Time {
	if sm.st == nil {
		return time.Time{}
	}
	return sm.st.LastSwitched
}
