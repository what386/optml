package cmd

import (
	"fmt"

	"optml/internal/integration"
	"optml/internal/packaging"
	"optml/internal/storage"

	"github.com/spf13/cobra"
)

var addOptRoot string
var addMetadataPath string

func init() {
	addCmd.Flags().StringVar(&addOptRoot, "opt-root", "/opt", "installation root directory")
	addCmd.Flags().StringVar(&addMetadataPath, "metadata-path", "/opt/optml/metadata.json", "metadata JSON file path")
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <name> <source>",
	Short: "Add a program to /opt",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		source := args[1]

		store := storage.NewMetadataStore(addMetadataPath)
		manager := packaging.NewManagerWithRoot(addOptRoot, store)

		entry, err := manager.Add(name, source)
		if err != nil {
			return err
		}
		shellMgr := integration.NewShellManager("")
		if err := shellMgr.AddEntry(entry); err != nil {
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
