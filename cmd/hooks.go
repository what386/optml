package cmd

import (
	"fmt"

	"optml/internal/integration"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(hooksCmd)
	hooksCmd.AddCommand(hooksInitCmd)
	hooksCmd.AddCommand(hooksCleanCmd)
}

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage shell integration hooks",
}

var hooksInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Add shell integration hooks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := integration.NewShellManager("")
		if err := mgr.InitHooks(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "shell hooks initialized")
		return nil
	},
}

var hooksCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove shell integration hooks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := integration.NewShellManager("")
		if err := mgr.CleanHooks(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "shell hooks cleaned")
		return nil
	},
}
