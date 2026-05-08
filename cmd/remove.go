package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <item>",
	Short: "Remove a program from /opt",
	Args:  cobra.ExactArgs(1),
	RunE:  runStub,
}
