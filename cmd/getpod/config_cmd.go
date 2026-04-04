package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
)

// configCmd es el grupo de comandos `getpod config`
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestiona la configuración de GetPod CLI",
}

// configInitCmd crea la config por defecto
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Crea ~/.getpod/config.yaml con valores por defecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			if envPath := os.Getenv("GETPOD_CONFIG"); envPath != "" {
				path = envPath
			}
		}
		if path == "" {
			path = config.DefaultConfigPath()
		}

		if err := config.InitConfig(path); err != nil {
			return err
		}

		fmt.Printf("✓ Config creada en: %s\n", path)
		fmt.Println("  Edítala para configurar tus workspaces y plugins.")
		return nil
	},
}

// configShowCmd muestra la config activa
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Muestra la configuración activa",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no hay config cargada — ejecuta 'getpod config init' primero")
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("error serializando config: %w", err)
		}

		path := cfgFile
		if path == "" {
			path = config.DefaultConfigPath()
		}

		fmt.Printf("# Config activa: %s\n", path)
		fmt.Println("---")
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
