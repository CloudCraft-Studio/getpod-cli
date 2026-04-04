package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd es el comando base de la CLI.
var rootCmd = &cobra.Command{
	Use:   "getpod",
	Short: "Developer workflow CLI",
	Long:  "GetPod CLI — unified developer workbench",
	// PersistentPreRunE carga la config antes de cualquier subcomando
	// (excepto los que no la necesitan como version o config init).
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Comandos que no requieren config existente
		skipConfig := map[string]bool{
			"version":     true,
			"config init": true,
		}
		key := cmd.CommandPath()
		// Eliminar prefijo "getpod " para comparar
		if len(key) > 7 {
			key = key[7:]
		}
		if skipConfig[key] {
			return nil
		}

		// Resolver path desde flag, env var o default
		path := cfgFile
		if path == "" {
			if envPath := os.Getenv("GETPOD_CONFIG"); envPath != "" {
				path = envPath
			}
		}

		var err error
		cfg, err = config.Load(path)
		if err != nil {
			// No es fatal si config no existe para ciertos comandos
			// pero sí para otros (plugins, sync, etc.)
			fmt.Fprintf(os.Stderr, "⚠ No se pudo cargar la config: %v\n", err)
			fmt.Fprintf(os.Stderr, "  Tip: ejecuta 'getpod config init' para crear una config base.\n")
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"ruta a la config (default: ~/.getpod/config.yaml)",
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
