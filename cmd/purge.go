package cmd

import (
	"github.com/sfx1909/nole/internal/purger"
	"github.com/spf13/cobra"
)

var purgeApplyFlag bool

var purgeCmd = &cobra.Command{
	Use:   "purge [path]",
	Short: "Find and remove dev build artifacts (node_modules, target, dist, ...)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ""
		if len(args) == 1 {
			path = args[0]
		}
		return purger.Run(path, purgeApplyFlag)
	},
}

func init() {
	purgeCmd.Flags().BoolVarP(&purgeApplyFlag, "apply", "a", false, "Delete found build artifacts")
	rootCmd.AddCommand(purgeCmd)
}
