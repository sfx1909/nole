package cmd

import (
	"github.com/sfx1909/nole/internal/analyser"
	"github.com/spf13/cobra"
)

var applyFlag bool

var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "Analyse config and suggest optimisations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return analyser.Run(applyFlag)
	},
}

func init() {
	analyseCmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Generate optimisation modules")
	rootCmd.AddCommand(analyseCmd)
}
