package internal

import (
	"os"
	"strings"
)

func IsRunningAsRoot() bool {
	return os.Geteuid() == 0
}

// CommandNeedsRoot returns whether the current argv (without program name)
// should require root privileges.
func CommandNeedsRoot(args []string) bool {
	if len(args) == 0 {
		return false
	}

	cmd := firstNonFlag(args)
	switch cmd {
	case "add", "remove", "refresh":
		return true
	case "doctor":
		return hasFixFlag(args)
	default:
		return false
	}
}

func firstNonFlag(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}

func hasFixFlag(args []string) bool {
	for _, a := range args {
		if a == "--fix" {
			return true
		}
	}
	return false
}
