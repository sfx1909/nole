package cmd

import (
	"github.com/sfx1909/nole/internal/maintainer"
	"github.com/spf13/cobra"
)

var maintainCmd = &cobra.Command{
	Use:   "maintain",
	Short: "Update flake inputs and rebuild if needed",
	RunE: func(cmd *cobra.Command, args []string) error {
		return maintainer.Run()
	},
}

func init() {
	rootCmd.AddCommand(maintainCmd)
}
