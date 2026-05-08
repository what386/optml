package cmd

import (
	"fmt"

	"optml/internal"
	"optml/internal/packaging"
	"optml/internal/storage"

	"github.com/spf13/cobra"
)

var doctorFix bool

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "attempt safe repairs before re-running diagnostics")
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := internal.RunIntegrityCheck(internal.IntegrityConfig{
			OptRoot:      packaging.DefaultOptRoot,
			MetadataPath: storage.DefaultMetadataPath,
			PathsFile:    "/opt/optml/paths.sh",
			SymlinkDir:   "/opt/optml/bin",
			Fix:          doctorFix,
		})
		if err != nil {
			return err
		}

		if doctorFix {
			fmt.Fprintln(cmd.OutOrStdout(), "Applied --fix: refreshed metadata and rebuilt integration artifacts.")
		}
		for _, line := range result.Lines {
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Summary: ok=%d warn=%d fail=%d\n", result.OK, result.Warn, result.Fail)
		if result.Fail > 0 {
			return fmt.Errorf("doctor found %d failing checks", result.Fail)
		}
		return nil
	},
}
