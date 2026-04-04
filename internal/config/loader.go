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
	v.AutomaticEnv()
	v.SetEnvPrefix("GETPOD")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error leyendo config desde %s: %w", configPath, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parseando config: %w", err)
	}

	resolveEnvVars(&cfg)
	return &cfg, nil
}

// InitConfig crea la configuración por defecto en configPath.
func InitConfig(configPath string) error {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio %s: %w", dir, err)
	}

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config ya existe en %s — usa 'getpod config show' para verla", configPath)
	}

	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error serializando config: %w", err)
	}

	header := []byte(
		"# GetPod CLI — Configuración\n" +
			"# Jerarquía: clients → workspaces → contexts\n" +
			"# Las credenciales pueden referenciar env vars: ${MI_API_TOKEN}\n\n",
	)
	content := append(header, data...)

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		return fmt.Errorf("error escribiendo config en %s: %w", configPath, err)
	}

	return nil
}

// envVarPattern coincide con ${VAR_NAME}
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func resolveStr(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match
	})
}

// resolveEnvVars recorre la config y resuelve referencias ${VAR} en todos los valores.
func resolveEnvVars(cfg *Config) {
	cfg.Server.URL = resolveStr(cfg.Server.URL)

	for clientName, client := range cfg.Clients {
		// Resolver credenciales de plugins del cliente
		for pluginName, pluginCfg := range client.Plugins {
			for k, v := range pluginCfg {
				cfg.Clients[clientName].Plugins[pluginName][k] = resolveStr(v)
			}
		}
		// Resolver valores en los contexts de cada workspace
		for wsName, ws := range client.Workspaces {
			for ctxName, ctx := range ws.Contexts {
				for pluginName, pluginCfg := range ctx {
					for k, v := range pluginCfg {
						cfg.Clients[clientName].Workspaces[wsName].Contexts[ctxName][pluginName][k] = resolveStr(v)
					}
				}
			}
		}
	}
}
