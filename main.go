package main

import (
	"fmt"
	"os"

	"optml/cmd"
	"optml/internal"
)

func main() {
	args := os.Args[1:]
	if internal.CommandNeedsRoot(args) && !internal.IsRunningAsRoot() {
		fmt.Fprintln(os.Stderr, "this command requires root privileges; run with sudo")
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
