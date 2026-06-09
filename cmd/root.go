package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nole",
	Short: "NixOS configuration manager and optimiser",
	Long:  "Nole is a smart NixOS rebuild wrapper that summarises warnings, deprecations, and suggests optimisations based on your config.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
