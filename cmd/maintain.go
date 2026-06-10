package cmd

import (
	"github.com/sfx1909/nole/internal/maintainer"
	"github.com/spf13/cobra"
)

var maintainCleanFlag bool

var maintainCmd = &cobra.Command{
	Use:   "maintain",
	Short: "Update flake inputs and rebuild if needed",
	RunE: func(cmd *cobra.Command, args []string) error {
		return maintainer.Run(maintainCleanFlag)
	},
}

func init() {
	maintainCmd.Flags().BoolVarP(&maintainCleanFlag, "clean", "c", false, "Also garbage-collect old generations and optimise the store")
	rootCmd.AddCommand(maintainCmd)
}
