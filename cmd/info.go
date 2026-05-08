package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"optml/internal/storage"

	"github.com/spf13/cobra"
)

var infoMetadataPath string

func init() {
	infoCmd.Flags().StringVar(&infoMetadataPath, "metadata-path", "/opt/optml/metadata.json", "metadata JSON file path")
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info <item>",
	Short: "Show metadata for a program",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		store := storage.NewMetadataStore(infoMetadataPath)
		entry, err := store.Get(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("program %q not found", name)
			}
			return err
		}

		out, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return fmt.Errorf("encode metadata: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	},
}
