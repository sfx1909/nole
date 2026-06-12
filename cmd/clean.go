package cmd

import (
	"github.com/sfx1909/nole/internal/cleaner"
	"github.com/spf13/cobra"
)

var cleanApplyFlag bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Garbage-collect old generations and optimise the Nix store",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleaner.Run(cleanApplyFlag)
	},
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanApplyFlag, "apply", "a", false, "Run garbage collection and store optimisation")
	rootCmd.AddCommand(cleanCmd)
}
