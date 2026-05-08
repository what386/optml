package cmd

import (
	"fmt"

	"optml/internal/integration"
	"optml/internal/packaging"
	"optml/internal/storage"

	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(refreshCmd) }

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh package metadata",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := packaging.RefreshMetadata(packaging.DefaultOptRoot, storage.DefaultMetadataPath)
		if err != nil {
			return err
		}
		shellMgr := integration.NewShellManager("")
		if err := shellMgr.RebuildFromState(state); err != nil {
			return err
		}
		symlinkMgr := integration.NewSymlinkManager("")
		if err := symlinkMgr.RebuildFromState(state); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "refreshed %d entries from %s into %s\n", len(state.Entries), packaging.DefaultOptRoot, storage.DefaultMetadataPath)
		return nil
	},
}
