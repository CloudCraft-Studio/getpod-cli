package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pluginsCmd es el grupo `getpod plugins`
var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Gestiona los plugins de GetPod CLI",
}

// pluginsListCmd lista los plugins habilitados por cliente/workspace/context activo
var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista los plugins habilitados en el contexto activo",
	RunE: func(cmd *cobra.Command, args []string) error {
		if reg == nil {
			return fmt.Errorf("no hay plugins activos cargados")
		}

		active := reg.ActivePlugins()
		if len(active) == 0 {
			fmt.Println("No hay plugins habilitados para el contexto actual. Verifica el config.yaml del cliente.")
			return nil
		}

		fmt.Println("Plugins activos para el contexto actual:")
		for _, name := range active {
			p, _ := reg.Get(name)
			fmt.Printf("  • %-10s (v%s)\n", name, p.Version())
		}

		return nil
	},
}

// pluginsValidateCmd verifica que las credenciales de los plugins estén definidas
var pluginsValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Verifica las credenciales de los plugins en el contexto activo",
	RunE: func(cmd *cobra.Command, args []string) error {
		if reg == nil {
			return fmt.Errorf("no hay plugins cargados para validar")
		}

		active := reg.ActivePlugins()
		allOk := true

		fmt.Println("Validando configuración de plugins...")
		for _, name := range active {
			p, _ := reg.Get(name)
			if err := p.Validate(); err != nil {
				fmt.Printf("  ✗ %-10s: Error: %v\n", name, err)
				allOk = false
			} else {
				fmt.Printf("  ✓ %-10s: Configuración OK (v%s)\n", name, p.Version())
			}
		}

		if !allOk {
			return fmt.Errorf("algunas credenciales no están configuradas correctamente")
		}

		fmt.Println("✓ Todos los plugins configurados correctamente.")
		return nil
	},
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsValidateCmd)
	rootCmd.AddCommand(pluginsCmd)
}
