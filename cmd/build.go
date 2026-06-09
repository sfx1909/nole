package cmd

import (
	"github.com/sfx1909/nole/internal/builder"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Rebuild NixOS and show a clean summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		return builder.Run()
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
