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

		s, _ := state.Load() // si falla, el activo es vacío

		for name, client := range cfg.Clients {
			prefix := "  "
			if s.ActiveClient == name {
				prefix = "* "
			}
			displayName := client.DisplayName
			if displayName == "" {
				displayName = name
			}
			fmt.Printf("%s%-15s (%s)\n", prefix, name, displayName)
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

		s, _ := state.Load()
		s.ActiveClient = name
		// Limpiar workspace/context si ya no son válidos para el nuevo cliente
		s.ActiveWorkspace = ""
		s.ActiveContext = ""

		if err := s.Save(); err != nil {
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
