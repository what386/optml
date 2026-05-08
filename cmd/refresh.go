package cmd

import (
	"fmt"

	"optml/internal"
	"optml/internal/integration"

	"github.com/spf13/cobra"
)

var refreshOptRoot string
var refreshMetadataPath string

func init() {
	refreshCmd.Flags().StringVar(&refreshOptRoot, "opt-root", "/opt", "installation root directory")
	refreshCmd.Flags().StringVar(&refreshMetadataPath, "metadata-path", "/opt/metadata.json", "metadata JSON file path")
	rootCmd.AddCommand(refreshCmd)
}

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh package metadata",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := internal.RefreshMetadata(refreshOptRoot, refreshMetadataPath)
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

		fmt.Fprintf(cmd.OutOrStdout(), "refreshed %d entries from %s into %s\n", len(state.Entries), refreshOptRoot, refreshMetadataPath)
		return nil
	},
}
