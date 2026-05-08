package cmd

import (
	"fmt"

	"optml/internal/integration"
	"optml/internal/packaging"
	"optml/internal/storage"

	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(addCmd) }

var addCmd = &cobra.Command{
	Use:   "add <name> <source>",
	Short: "Add a program to /opt",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		source := args[1]

		store := storage.NewMetadataStore("")
		manager := packaging.NewManager(store)

		entry, err := manager.Add(name, source)
		if err != nil {
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
		symlinkMgr := integration.NewSymlinkManager("")
		if err := symlinkMgr.AddEntry(entry); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "added %q to %s\n", entry.Name, entry.RootDir)
		return nil
	},
}
