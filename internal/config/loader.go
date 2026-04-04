package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir  = ".getpod"
	DefaultConfigFile = "config.yaml"
)

// DefaultConfigPath retorna la ruta por defecto: ~/.getpod/config.yaml
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(DefaultConfigDir, DefaultConfigFile)
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
}

// Load lee la configuración desde configPath usando Viper.
// Los valores de config pueden referenciar env vars con la sintaxis ${VAR_NAME}.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Permitir sobreescritura de config via env var GETPOD_CONFIG
	v.AutomaticEnv()
	v.SetEnvPrefix("GETPOD")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error leyendo config desde %s: %w", configPath, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parseando config: %w", err)
	}

	// Resolver referencias a env vars (${VAR_NAME}) en valores de config
	resolveEnvVars(&cfg)

	return &cfg, nil
}

// InitConfig crea la configuración por defecto en configPath.
// Retorna error si el archivo ya existe.
func InitConfig(configPath string) error {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	// Crear directorio si no existe
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio %s: %w", dir, err)
	}

	// No sobreescribir si ya existe
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config ya existe en %s — usa 'getpod config show' para verla", configPath)
	}

	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error serializando config: %w", err)
	}

	header := []byte("# GetPod CLI Configuration\n# Edita este archivo para configurar tus workspaces y plugins.\n# Las credenciales pueden usar referencias a env vars: ${MI_API_TOKEN}\n\n")
	content := append(header, data...)

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		return fmt.Errorf("error escribiendo config en %s: %w", configPath, err)
	}

	return nil
}

// envVarPattern coincide con ${VAR_NAME}
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// resolveEnvVars recorre los plugin_config y resuelve referencias ${VAR}.
func resolveEnvVars(cfg *Config) {
	for wsName, ws := range cfg.Workspaces {
		for pluginName, pluginCfg := range ws.PluginCfg {
			for k, v := range pluginCfg {
				resolved := envVarPattern.ReplaceAllStringFunc(v, func(match string) string {
					varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
					if val := os.Getenv(varName); val != "" {
						return val
					}
					return match // mantener el placeholder si la var no está definida
				})
				cfg.Workspaces[wsName].PluginCfg[pluginName][k] = resolved
			}
		}
	}

	// También resolver URL del server
	cfg.Server.URL = envVarPattern.ReplaceAllStringFunc(cfg.Server.URL, func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match
	})
}
