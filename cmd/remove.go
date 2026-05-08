package cmd

import (
	"fmt"

	"optml/internal"

	"github.com/spf13/cobra"
)

var removeOptRoot string
var removeMetadataPath string

func init() {
	removeCmd.Flags().StringVar(&removeOptRoot, "opt-root", "/opt", "installation root directory")
	removeCmd.Flags().StringVar(&removeMetadataPath, "metadata-path", "/opt/metadata.json", "metadata JSON file path")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <item>",
	Short: "Remove a program from /opt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		store := internal.NewMetadataStore(removeMetadataPath)
		manager := internal.NewOptManagerWithRoot(removeOptRoot, store)

		if err := manager.Remove(name); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "removed %q\n", name)
		return nil
	},
}
