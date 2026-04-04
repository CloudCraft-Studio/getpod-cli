package config

// Config es la estructura raíz de ~/.getpod/config.yaml
type Config struct {
	Server     ServerConfig               `yaml:"server"     mapstructure:"server"`
	Sync       SyncConfig                 `yaml:"sync"       mapstructure:"sync"`
	Workspaces map[string]WorkspaceConfig `yaml:"workspaces" mapstructure:"workspaces"`
}

// ServerConfig almacena la URL base del servidor GetPod.
type ServerConfig struct {
	URL string `yaml:"url" mapstructure:"url"`
}

// SyncConfig controla el comportamiento de sincronización automática.
type SyncConfig struct {
	Interval  string `yaml:"interval"    mapstructure:"interval"`    // e.g. "15m"
	BatchSize int    `yaml:"batch_size"  mapstructure:"batch_size"`  // e.g. 100
}

// WorkspaceConfig define un workspace y sus plugins activos.
type WorkspaceConfig struct {
	DisplayName string                       `yaml:"display_name"   mapstructure:"display_name"`
	Plugins     []string                     `yaml:"plugins"        mapstructure:"plugins"`
	PluginCfg   map[string]map[string]string `yaml:"plugin_config"  mapstructure:"plugin_config"`
}

// DefaultConfig retorna una configuración de ejemplo con valores razonables.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			URL: "https://api.getpod.dev",
		},
		Sync: SyncConfig{
			Interval:  "15m",
			BatchSize: 100,
		},
		Workspaces: map[string]WorkspaceConfig{
			"default": {
				DisplayName: "Default Workspace",
				Plugins:     []string{},
				PluginCfg:   map[string]map[string]string{},
			},
		},
	}
}
