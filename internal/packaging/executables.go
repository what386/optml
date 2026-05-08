package packaging

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindExecutables(rootDir string) ([]string, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("stat root dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("rootDir must be a directory: %s", rootDir)
	}
	binPaths := []string{}
	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk path: %w", walkErr)
		}
		if d.IsDir() {
			return nil
		}
		entryInfo, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat entry: %w", err)
		}
		if entryInfo.Mode().IsRegular() && entryInfo.Mode().Perm()&0o111 != 0 {
			binPaths = append(binPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return binPaths, nil
}
