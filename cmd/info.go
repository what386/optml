package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info <item>",
	Short: "Show metadata for a program",
	Args:  cobra.ExactArgs(1),
	RunE:  runStub,
}
