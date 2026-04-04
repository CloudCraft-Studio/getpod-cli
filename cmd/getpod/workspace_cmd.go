package main

import (
	"fmt"

	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Gestiona los workspaces (proyectos/equipos)",
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista los workspaces del cliente activo",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no hay configuración cargada")
		}

		s, _ := state.Load()
		if s.ActiveClient == "" {
			return fmt.Errorf("no hay un cliente activo. Usa 'getpod client use <nombre>' primero")
		}

		client, ok := cfg.Clients[s.ActiveClient]
		if !ok {
			return fmt.Errorf("el cliente activo %q no existe en config.yaml", s.ActiveClient)
		}

		fmt.Printf("Workspaces para el cliente: %s (%s)\n", client.DisplayName, s.ActiveClient)
		for name, ws := range client.Workspaces {
			prefix := "  "
			if s.ActiveWorkspace == name {
				prefix = "* "
			}
			displayName := ws.DisplayName
			if displayName == "" {
				displayName = name
			}
			fmt.Printf("%s%-15s (%s)\n", prefix, name, displayName)
		}
		return nil
	},
}

var workspaceUseCmd = &cobra.Command{
	Use:   "use [nombre]",
	Short: "Selecciona el workspace activo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s, _ := state.Load()
		if s.ActiveClient == "" {
			return fmt.Errorf("no hay un cliente activo. Usa 'getpod client use <nombre>' primero")
		}

		client, ok := cfg.Clients[s.ActiveClient]
		if !ok {
			return fmt.Errorf("el cliente activo %q no existe en config.yaml", s.ActiveClient)
		}

		if _, ok := client.Workspaces[name]; !ok {
			return fmt.Errorf("el workspace %q no existe para el cliente %q", name, s.ActiveClient)
		}

		s.ActiveWorkspace = name
		s.ActiveContext = "" // Limpiar contexto

		if err := s.Save(); err != nil {
			return err
		}

		fmt.Printf("✓ Workspace activo: %s\n", name)
		fmt.Printf("Nota: Usa 'getpod context use <nombre>' para seleccionar un contexto (dev, staging, prod) dentro de este workspace.\n")
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceUseCmd)
	rootCmd.AddCommand(workspaceCmd)
}
