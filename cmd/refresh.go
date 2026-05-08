package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(refreshCmd)
}

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh package metadata",
	Args:  cobra.NoArgs,
	RunE:  runStub,
}
