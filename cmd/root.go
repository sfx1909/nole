package cmd

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

//go:embed title.txt
var titleArt string

// version is set at build time via -ldflags "-X github.com/sfx1909/nole/cmd.version=...".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "nole",
	Short:   "NixOS configuration manager and optimiser",
	Long:    "Nole is a smart NixOS rebuild wrapper that summarises warnings, deprecations, and suggests optimisations based on your config.",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(color.CyanString(titleArt))
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
