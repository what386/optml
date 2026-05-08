package cmd

import (
	"fmt"

	"optml/internal/storage"

	"github.com/spf13/cobra"
)

var listMetadataPath string

func init() {
	listCmd.Flags().StringVar(&listMetadataPath, "metadata-path", "/opt/optml/metadata.json", "metadata JSON file path")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed programs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store := storage.NewMetadataStore(listMetadataPath)
		entries, err := store.List()
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no installed programs")
			return nil
		}

		for _, entry := range entries {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", entry.Name, entry.RootDir)
		}
		return nil
	},
}
