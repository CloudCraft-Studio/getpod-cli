package main

import (
	"fmt"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	clientAddDisplayName string
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Gestiona los clientes/empresas",
}

var clientListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista los clientes configurados",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no hay configuración cargada")
		}

		s, err := state.Load()
		if err != nil {
			return fmt.Errorf("error cargando estado: %w", err)
		}

		fmt.Println("\nCLIENTES")
		fmt.Printf("%-20s %-25s %s\n", "NOMBRE", "DISPLAY NAME", "PLUGINS")
		fmt.Println("─────────────────────────────────────────────────────")
		for name, client := range cfg.Clients {
			prefix := ""
			if s.ActiveClient == name {
				prefix = "* "
			}
			displayName := client.DisplayName
			if displayName == "" {
				displayName = name
			}

			var plugins []string
			for p := range client.Plugins {
				plugins = append(plugins, p)
			}
			pluginStr := ""
			if len(plugins) > 0 {
				pluginStr = plugins[0]
				for i := 1; i < len(plugins); i++ {
					pluginStr += ", " + plugins[i]
				}
			}
			fmt.Printf("%s%-20s %-25s %s\n", prefix, name, displayName, pluginStr)
		}
		return nil
	},
}

var clientUseCmd = &cobra.Command{
	Use:   "use [nombre]",
	Short: "Selecciona el cliente activo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if _, ok := cfg.Clients[name]; !ok {
			return fmt.Errorf("el cliente %q no existe en config.yaml", name)
		}

		s, err := state.Load()
		if err != nil {
			return fmt.Errorf("error cargando estado: %w", err)
		}

		if err := s.UseClient(name); err != nil {
			return err
		}

		fmt.Printf("✓ Cliente activo: %s\n", name)
		fmt.Printf("Nota: Usa 'getpod workspace use <nombre>' para seleccionar un workspace dentro de este cliente.\n")
		return nil
	},
}

var clientAddCmd = &cobra.Command{
	Use:   "add [nombre]",
	Short: "Agrega un nuevo cliente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if cfg == nil {
			return fmt.Errorf("no hay configuración cargada")
		}

		// Initialize Clients map if nil
		if cfg.Clients == nil {
			cfg.Clients = make(map[string]config.ClientConfig)
		}

		// Check if client already exists
		if _, exists := cfg.Clients[name]; exists {
			return fmt.Errorf("el cliente %q ya existe", name)
		}

		// Create new client
		displayName := clientAddDisplayName
		if displayName == "" {
			displayName = name
		}

		newClient := config.ClientConfig{
			DisplayName: displayName,
			Plugins:     make(map[string]map[string]string),
			Workspaces:  make(map[string]config.WorkspaceConfig),
		}

		cfg.Clients[name] = newClient

		// Save config
		configPath := cfgFile
		if configPath == "" {
			configPath = config.DefaultConfigPath()
		}

		if err := config.Save(cfg, configPath); err != nil {
			return fmt.Errorf("error guardando config: %w", err)
		}

		fmt.Printf("✓ Cliente %q agregado exitosamente\n", name)
		fmt.Printf("  Display Name: %s\n", displayName)
		fmt.Printf("\nPróximos pasos:\n")
		fmt.Printf("  1. Edita ~/.getpod/config.yaml para agregar plugins\n")
		fmt.Printf("  2. Crea workspaces con 'getpod workspace add'\n")
		fmt.Printf("  3. Activa el cliente con 'getpod client use %s'\n", name)

		return nil
	},
}

func init() {
	clientAddCmd.Flags().StringVar(&clientAddDisplayName, "display-name", "", "Nombre visible del cliente")

	clientCmd.AddCommand(clientListCmd)
	clientCmd.AddCommand(clientUseCmd)
	clientCmd.AddCommand(clientAddCmd)
	rootCmd.AddCommand(clientCmd)
}
