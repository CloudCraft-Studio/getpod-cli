package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version se inyecta en build time con -ldflags "-X main.Version=x.y.z"
var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Muestra la versión de GetPod CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("getpod v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
