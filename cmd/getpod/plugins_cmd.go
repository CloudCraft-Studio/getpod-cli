package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// pluginsCmd es el grupo `getpod plugins`
var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Gestiona los plugins de GetPod CLI",
}

// pluginsListCmd lista los plugins habilitados por workspace
var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista los plugins habilitados en cada workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no hay config cargada — ejecuta 'getpod config init' primero")
		}

		if len(cfg.Workspaces) == 0 {
			fmt.Println("No hay workspaces configurados.")
			return nil
		}

		for wsName, ws := range cfg.Workspaces {
			displayName := ws.DisplayName
			if displayName == "" {
				displayName = wsName
			}
			fmt.Printf("Workspace: %s (%s)\n", displayName, wsName)
			if len(ws.Plugins) == 0 {
				fmt.Println("  (sin plugins habilitados)")
			} else {
				for _, p := range ws.Plugins {
					fmt.Printf("  • %s\n", p)
				}
			}
			fmt.Println()
		}

		return nil
	},
}

// pluginsValidateCmd verifica que las credenciales de los plugins estén definidas
var pluginsValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Verifica las credenciales de los plugins configurados",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no hay config cargada — ejecuta 'getpod config init' primero")
		}

		allOk := true

		for wsName, ws := range cfg.Workspaces {
			displayName := ws.DisplayName
			if displayName == "" {
				displayName = wsName
			}
			fmt.Printf("Workspace: %s\n", displayName)

			for _, plugin := range ws.Plugins {
				pluginCfg, hasCfg := ws.PluginCfg[plugin]
				if !hasCfg {
					fmt.Printf("  ⚠ %s: sin configuración de credenciales\n", plugin)
					allOk = false
					continue
				}

				// Verificar que los valores no sigan siendo placeholders ${...}
				missingVars := []string{}
				for k, v := range pluginCfg {
					if len(v) > 2 && v[0] == '$' && v[1] == '{' {
						missingVars = append(missingVars, fmt.Sprintf("%s=%s", k, v))
					}
				}

				if len(missingVars) > 0 {
					fmt.Printf("  ✗ %s: env vars no resueltas:\n", plugin)
					for _, mv := range missingVars {
						fmt.Printf("      - %s\n", mv)
					}
					allOk = false
				} else {
					fmt.Printf("  ✓ %s: credenciales OK\n", plugin)
				}
			}
			fmt.Println()
		}

		if !allOk {
			fmt.Fprintln(os.Stderr, "✗ Algunas credenciales no están configuradas correctamente.")
			os.Exit(1)
		}

		fmt.Println("✓ Todas las credenciales están configuradas correctamente.")
		return nil
	},
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsValidateCmd)
	rootCmd.AddCommand(pluginsCmd)
}
