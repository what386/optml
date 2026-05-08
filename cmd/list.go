package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed programs",
	Args:  cobra.NoArgs,
	RunE:  runStub,
}
