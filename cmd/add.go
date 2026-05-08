package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <item>",
	Short: "Add a program to /opt",
	Args:  cobra.ExactArgs(1),
	RunE:  runStub,
}
