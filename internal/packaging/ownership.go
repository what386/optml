package packaging

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func handoffOwnershipIfSudo(rootPath string) error {
	if os.Geteuid() != 0 {
		return nil
	}

	sudoUID := os.Getenv("SUDO_UID")
	sudoGID := os.Getenv("SUDO_GID")
	if sudoUID == "" || sudoGID == "" {
		return nil
	}

	uid, err := strconv.Atoi(sudoUID)
	if err != nil {
		return fmt.Errorf("invalid SUDO_UID %q: %w", sudoUID, err)
	}
	gid, err := strconv.Atoi(sudoGID)
	if err != nil {
		return fmt.Errorf("invalid SUDO_GID %q: %w", sudoGID, err)
	}

	if err := os.Lchown(rootPath, uid, gid); err != nil {
		return fmt.Errorf("chown root path %s: %w", rootPath, err)
	}

	err = filepath.WalkDir(rootPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk for chown: %w", walkErr)
		}
		if err := os.Lchown(path, uid, gid); err != nil {
			return fmt.Errorf("chown %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
