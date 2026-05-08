package cmd

import (
	"fmt"

	"optml/internal"

	"github.com/spf13/cobra"
)

var doctorOptRoot string
var doctorMetadataPath string
var doctorPathsFile string
var doctorSymlinkDir string
var doctorFix bool

func init() {
	doctorCmd.Flags().StringVar(&doctorOptRoot, "opt-root", "/opt", "installation root directory")
	doctorCmd.Flags().StringVar(&doctorMetadataPath, "metadata-path", "/opt/optml/metadata.json", "metadata JSON file path")
	doctorCmd.Flags().StringVar(&doctorPathsFile, "paths-file", "/opt/optml/paths.sh", "shell integration file path")
	doctorCmd.Flags().StringVar(&doctorSymlinkDir, "symlink-dir", "/opt/optml/bin", "symlink integration directory")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "attempt safe repairs before re-running diagnostics")
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := internal.RunIntegrityCheck(internal.IntegrityConfig{
			OptRoot:      doctorOptRoot,
			MetadataPath: doctorMetadataPath,
			PathsFile:    doctorPathsFile,
			SymlinkDir:   doctorSymlinkDir,
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
