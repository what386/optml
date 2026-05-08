package packaging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var executableDirAllowlist = []string{
	"bin",
	"sbin",
	"usr/bin",
	"usr/sbin",
	"libexec",
}

func FindExecutables(rootDir string) ([]string, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("stat root dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("rootDir must be a directory: %s", rootDir)
	}

	normalizedRoot := filepath.Clean(rootDir)
	allowlistHits := []string{}

	for _, relDir := range executableDirAllowlist {
		absDir := filepath.Join(normalizedRoot, filepath.FromSlash(relDir))
		dirInfo, err := os.Stat(absDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat allowlist directory %s: %w", absDir, err)
		}
		if !dirInfo.IsDir() {
			continue
		}

		err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return fmt.Errorf("walk allowlist directory: %w", walkErr)
			}
			if d.IsDir() {
				return nil
			}
			entryInfo, err := d.Info()
			if err != nil {
				return fmt.Errorf("stat entry: %w", err)
			}
			if !entryInfo.Mode().IsRegular() || entryInfo.Mode().Perm()&0o111 == 0 {
				return nil
			}
			if isDeniedExecutable(path) {
				return nil
			}
			allowlistHits = append(allowlistHits, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(allowlistHits) > 0 {
		return uniqueSorted(allowlistHits), nil
	}

	// Fallback: root-level files only, no recursion.
	items, err := os.ReadDir(normalizedRoot)
	if err != nil {
		return nil, fmt.Errorf("read root dir: %w", err)
	}

	rootHits := []string{}
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		path := filepath.Join(normalizedRoot, item.Name())
		entryInfo, err := item.Info()
		if err != nil {
			return nil, fmt.Errorf("stat root entry: %w", err)
		}
		if !entryInfo.Mode().IsRegular() || entryInfo.Mode().Perm()&0o111 == 0 {
			continue
		}
		if isDeniedExecutable(path) {
			continue
		}
		rootHits = append(rootHits, path)
	}

	return uniqueSorted(rootHits), nil
}

func uniqueSorted(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	sort.Strings(paths)
	out := make([]string, 0, len(paths))
	var prev string
	for i, p := range paths {
		if i == 0 || p != prev {
			out = append(out, p)
		}
		prev = p
	}
	return out
}

func isDeniedExecutable(path string) bool {
	name := strings.ToLower(filepath.Base(path))

	if strings.HasSuffix(name, ".so") || strings.Contains(name, ".so.") {
		return true
	}
	if strings.HasSuffix(name, ".a") {
		return true
	}
	if strings.HasSuffix(name, ".la") {
		return true
	}
	if strings.HasSuffix(name, ".o") {
		return true
	}
	if strings.HasSuffix(name, ".dylib") {
		return true
	}

	return false
}
