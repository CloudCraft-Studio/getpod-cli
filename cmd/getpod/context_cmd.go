package main

import (
	"fmt"

	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Gestiona los contextos/environments (dev, staging, prod)",
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista los contextos del workspace activo",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no hay configuración cargada")
		}

		s, _ := state.Load()
		if s.ActiveClient == "" || s.ActiveWorkspace == "" {
			return fmt.Errorf("no hay un cliente o workspace activos. Usa 'getpod client use' y 'getpod workspace use'")
		}

		client, _ := cfg.Clients[s.ActiveClient]
		ws, ok := client.Workspaces[s.ActiveWorkspace]
		if !ok {
			return fmt.Errorf("el workspace activo %q no existe para el cliente %q", s.ActiveClient, s.ActiveWorkspace)
		}

		fmt.Printf("Contextos para el workspace: %s (%s)\n", ws.DisplayName, s.ActiveWorkspace)
		for name := range ws.Contexts {
			prefix := "  "
			if s.ActiveContext == name {
				prefix = "* "
			}
			fmt.Printf("%s%s\n", prefix, name)
		}
		return nil
	},
}

var contextUseCmd = &cobra.Command{
	Use:   "use [nombre]",
	Short: "Selecciona el contexto activo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s, err := state.Load()
		if err != nil {
			return fmt.Errorf("error cargando estado: %w", err)
		}
		if s.ActiveClient == "" || s.ActiveWorkspace == "" {
			return fmt.Errorf("no hay un cliente o workspace activos. Usa 'getpod client use' y 'getpod workspace use'")
		}

		client, _ := cfg.Clients[s.ActiveClient]
		ws, ok := client.Workspaces[s.ActiveWorkspace]
		if !ok {
			return fmt.Errorf("el workspace activo %q no existe para el cliente %q", s.ActiveClient, s.ActiveWorkspace)
		}

		if _, ok := ws.Contexts[name]; !ok {
			return fmt.Errorf("el contexto %q no existe para el workspace %q", name, s.ActiveWorkspace)
		}

		if err := s.UseContext(name); err != nil {
			return err
		}

		fmt.Printf("✓ Contexto activo: %s\n", name)
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextUseCmd)
	rootCmd.AddCommand(contextCmd)
}
