package cmd

import (
	"github.com/sfx1909/nole/internal/config"
	"github.com/sfx1909/nole/internal/maintainer"
	"github.com/spf13/cobra"
)

var maintainCleanFlag bool

var maintainCmd = &cobra.Command{
	Use:   "maintain",
	Short: "Update flake inputs and rebuild if needed",
	RunE: func(cmd *cobra.Command, args []string) error {
		clean := maintainCleanFlag
		if !cmd.Flags().Changed("clean") {
			if cfg, err := config.Load(); err == nil {
				clean = cfg.Maintain.Clean
			}
		}
		return maintainer.Run(clean)
	},
}

func init() {
	maintainCmd.Flags().BoolVarP(&maintainCleanFlag, "clean", "c", false, "Also garbage-collect old generations and optimise the store (default: maintain.clean in config)")
	rootCmd.AddCommand(maintainCmd)
}
