package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "optml",
	Short: "The optimal /opt manager",
}

func Execute() error {
	return rootCmd.Execute()
}

func runStub(cmd *cobra.Command, args []string) error {
	fmt.Printf("command=%s args=%v\n", cmd.CommandPath(), args)
	return errors.New("not implemented")
}
