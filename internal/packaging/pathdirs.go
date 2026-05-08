package packaging

import (
	"os"
	"path/filepath"
)

var fallbackPathDirOrder = []string{"bin", "sbin"}

func SelectFallbackPathDirs(rootDir string) ([]string, error) {
	for _, rel := range fallbackPathDirOrder {
		candidate := filepath.Join(rootDir, rel)
		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			return []string{candidate}, nil
		}
	}
	return nil, nil
}
