package cmd

import (
	"errors"
	"fmt"

	"optml/internal/integration"
	"optml/internal/packaging"
	"optml/internal/storage"

	"github.com/spf13/cobra"
)

var removeOptRoot string
var removeMetadataPath string

func init() {
	removeCmd.Flags().StringVar(&removeOptRoot, "opt-root", "/opt", "installation root directory")
	removeCmd.Flags().StringVar(&removeMetadataPath, "metadata-path", "/opt/optml/metadata.json", "metadata JSON file path")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <item>",
	Short: "Remove a program from /opt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		store := storage.NewMetadataStore(removeMetadataPath)
		manager := packaging.NewManagerWithRoot(removeOptRoot, store)
		entry, err := store.Get(name)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if err == nil {
			symlinkMgr := integration.NewSymlinkManager("")
			if err := symlinkMgr.RemoveEntry(entry); err != nil {
				return err
			}
		}

		if err := manager.Remove(name); err != nil {
			return err
		}
		shellMgr := integration.NewShellManager("")
		state, err := store.Load()
		if err != nil {
			return err
		}
		if err := shellMgr.RebuildFromState(state); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "removed %q\n", name)
		return nil
	},
}
