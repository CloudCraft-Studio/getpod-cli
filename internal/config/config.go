package config

// Config es la estructura raíz de ~/.getpod/config.yaml
type Config struct {
	Server  ServerConfig            `yaml:"server"  mapstructure:"server"`
	Sync    SyncConfig              `yaml:"sync"    mapstructure:"sync"`
	Clients map[string]ClientConfig `yaml:"clients" mapstructure:"clients"`
}

// ServerConfig almacena la URL base del servidor GetPod.
type ServerConfig struct {
	URL string `yaml:"url" mapstructure:"url"`
}

// SyncConfig controla el comportamiento de sincronización automática.
type SyncConfig struct {
	Interval  string `yaml:"interval"   mapstructure:"interval"`   // e.g. "15m"
	BatchSize int    `yaml:"batch_size" mapstructure:"batch_size"` // e.g. 100
}

// ClientConfig representa una empresa o cliente.
// Los plugins se declaran aquí con sus credenciales (a nivel global del cliente).
// Cada workspace del cliente hereda estos plugins.
type ClientConfig struct {
	DisplayName string                       `yaml:"display_name" mapstructure:"display_name"`
	Plugins     map[string]map[string]string `yaml:"plugins"      mapstructure:"plugins"`
	Workspaces  map[string]WorkspaceConfig   `yaml:"workspaces"   mapstructure:"workspaces"`
}

// WorkspaceConfig agrupa trabajo dentro de un cliente (proyecto, equipo, área).
type WorkspaceConfig struct {
	DisplayName string                    `yaml:"display_name" mapstructure:"display_name"`
	Contexts    map[string]ContextConfig  `yaml:"contexts"     mapstructure:"contexts"`
}

// ContextConfig representa un environment específico dentro de un workspace
// (dev, staging, prod, etc.). Solo puede referenciar plugins activos en el
// ClientConfig padre. Los valores aquí son específicos del environment
// (project_key, repo, branch, board_id...) y se mezclan con las credenciales
// del cliente en tiempo de ejecución.
type ContextConfig map[string]map[string]string // plugin → { clave: valor }

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
		Clients: map[string]ClientConfig{
			"mi-cliente": {
				DisplayName: "Mi Cliente",
				Plugins: map[string]map[string]string{
					"jira": {
						"url":   "https://mycompany.atlassian.net",
						"token": "${JIRA_TOKEN}",
					},
				},
				Workspaces: map[string]WorkspaceConfig{
					"backend": {
						DisplayName: "Backend Team",
						Contexts: map[string]ContextConfig{
							"dev": {
								"jira": {
									"project_key": "BACK",
									"board_id":    "1",
								},
							},
						},
					},
				},
			},
		},
	}
}
