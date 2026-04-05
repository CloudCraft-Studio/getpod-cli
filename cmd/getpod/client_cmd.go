package main

import (
	"fmt"

	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/spf13/cobra"
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

func init() {
	clientCmd.AddCommand(clientListCmd)
	clientCmd.AddCommand(clientUseCmd)
	rootCmd.AddCommand(clientCmd)
}
