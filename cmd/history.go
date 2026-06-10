package cmd

import (
	"github.com/sfx1909/nole/internal/oplog"
	"github.com/spf13/cobra"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent nole operations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return oplog.PrintRecent(historyLimit)
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "Number of entries to show")
	rootCmd.AddCommand(historyCmd)
}
