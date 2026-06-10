package cmd

import (
	"github.com/sfx1909/nole/internal/analyser"
	"github.com/sfx1909/nole/internal/config"
	"github.com/spf13/cobra"
)

var (
	applyFlag  bool
	formatFlag string
)

var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "Analyse config and suggest optimisations",
	RunE: func(cmd *cobra.Command, args []string) error {
		formatStr := formatFlag
		if formatStr == "" {
			if cfg, err := config.Load(); err == nil {
				formatStr = cfg.Format
			}
		}

		format, err := analyser.ParseFormat(formatStr)
		if err != nil {
			return err
		}

		return analyser.Run(applyFlag, format)
	},
}

func init() {
	analyseCmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Generate optimisation modules")
	analyseCmd.Flags().StringVarP(&formatFlag, "format", "f", "", "Output format: module (default), flake-part, or flake")
	rootCmd.AddCommand(analyseCmd)
}
