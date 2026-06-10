package cmd

import (
	"github.com/sfx1909/nole/internal/status"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show a quick dashboard of system and flake state",
	RunE: func(cmd *cobra.Command, args []string) error {
		return status.Run()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
