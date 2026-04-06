package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin/jira"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/CloudCraft-Studio/getpod-cli/internal/store"
	"github.com/CloudCraft-Studio/getpod-cli/internal/tui"
)

var (
	cfgFile string
	cfg     *config.Config
	reg     *plugin.Registry
)

// rootCmd es el comando base de la CLI.
var rootCmd = &cobra.Command{
	Use:   "getpod",
	Short: "Developer workflow CLI",
	Long:  "GetPod CLI — unified developer workbench",
	RunE:  runTUI,
	// PersistentPreRunE carga la config y el contexto activo antes de cualquier subcomando
	// (excepto los que no lo necesitan como version o config init).
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Comandos que no requieren configuración completa
		skipConfig := map[string]bool{
			"version":     true,
			"config init": true,
		}
		// Evaluar también si es un subcomando de config
		key := cmd.CommandPath()
		if len(key) > 7 {
			key = key[7:]
		}

		if skipConfig[key] {
			return nil
		}

		// 1. Cargar Configuración
		path := cfgFile
		if path == "" {
			if envPath := os.Getenv("GETPOD_CONFIG"); envPath != "" {
				path = envPath
			}
		}

		var err error
		cfg, err = config.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar la config: %v\n", err)
			return nil // Permitimos continuar para comandos básicos
		}

		// 2. Cargar Estado
		s, err := state.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar el estado actual: %v\n", err)
			return nil
		}

		// 3. Resolver Contexto Activo
		active, err := s.Resolve(cfg)
		if err != nil {
			// Comandos de client/workspace/context pueden funcionar sin contexto completo
			if key == "client list" || key == "client use" ||
				key == "workspace list" || key == "workspace use" ||
				key == "context list" || key == "context use" ||
				key == "config show" {
				return nil
			}

			// Partial context: set up plugins with client config only (no context merge).
			// The TUI allows selecting workspace/env interactively, so plugins
			// only need their base credentials (base_url, email, api_token) to fetch issues.
			if s.ActiveClient != "" {
				if client, ok := cfg.Clients[s.ActiveClient]; ok {
					partial := plugin.ActiveContext{
						ClientName: s.ActiveClient,
						Client:     client,
					}
					if setupErr := reg.SetupAll(partial); setupErr != nil {
						fmt.Fprintf(os.Stderr, "⚠ Error en inicialización de plugins: %v\n", setupErr)
					}
				}
			}
			return nil
		}

		// 4. Inicializar Registry (SetupAll)
		if err := reg.SetupAll(*active); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Error en inicialización de plugins: %v\n", err)
		}

		// 5. Inyectar subcomandos de plugins activos
		for _, pCmd := range reg.AllCommands() {
			cmd.Root().AddCommand(pCmd)
		}

		return nil
	},
}

func runTUI(cmd *cobra.Command, args []string) error {
	if cfg == nil {
		path := cfgFile
		if path == "" {
			if envPath := os.Getenv("GETPOD_CONFIG"); envPath != "" {
				path = envPath
			}
		}
		var err error
		cfg, err = config.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar la config: %v\n", err)
			cfg = config.DefaultConfig()
		}
	}

	db, err := store.NewStore(store.DefaultDBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ No se pudo abrir la base de datos: %v\n", err)
		db = nil // App handles nil db gracefully
	}
	if db != nil {
		defer db.Close()
	}

	app := tui.NewApp(cfg, reg, db)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"ruta a la config (default: ~/.getpod/config.yaml)",
	)

	// Inicializar el Registry (aquí se registrarán los plugins compilados en el futuro)
	reg = plugin.NewRegistry()
	reg.Register(&jira.Plugin{})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
