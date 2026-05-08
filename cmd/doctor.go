package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics",
	Args:  cobra.NoArgs,
	RunE:  runStub,
}
