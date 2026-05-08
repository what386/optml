package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"optml/internal"
)

const defaultSymlinkDir = "/opt/optml/bin"

type SymlinkManager struct {
	symlinkDir string
}

func NewSymlinkManager(symlinkDir string) *SymlinkManager {
	if strings.TrimSpace(symlinkDir) == "" {
		symlinkDir = defaultSymlinkDir
	}
	return &SymlinkManager{symlinkDir: symlinkDir}
}

func (m *SymlinkManager) AddEntry(entry internal.OptEntry) error {
	if err := os.MkdirAll(m.symlinkDir, 0o755); err != nil {
		return fmt.Errorf("mkdir symlink dir: %w", err)
	}

	desired, err := m.buildDesiredLinks(internal.MetadataState{Entries: map[string]internal.OptEntry{entry.Name: entry}})
	if err != nil {
		return err
	}

	for name, target := range desired {
		if err := m.ensureLink(name, target); err != nil {
			return err
		}
	}
	return nil
}

func (m *SymlinkManager) RemoveEntry(entry internal.OptEntry) error {
	items, err := os.ReadDir(m.symlinkDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read symlink dir: %w", err)
	}

	targetSet := make(map[string]struct{}, len(entry.BinPaths))
	for _, target := range entry.BinPaths {
		targetSet[target] = struct{}{}
	}

	for _, item := range items {
		linkPath := filepath.Join(m.symlinkDir, item.Name())
		info, err := os.Lstat(linkPath)
		if err != nil {
			return fmt.Errorf("lstat symlink: %w", err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		resolved, err := os.Readlink(linkPath)
		if err != nil {
			return fmt.Errorf("read symlink: %w", err)
		}

		if _, ok := targetSet[resolved]; ok {
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("remove symlink: %w", err)
			}
		}
	}

	return nil
}

func (m *SymlinkManager) RebuildFromState(state internal.MetadataState) error {
	if err := os.MkdirAll(m.symlinkDir, 0o755); err != nil {
		return fmt.Errorf("mkdir symlink dir: %w", err)
	}

	desired, err := m.buildDesiredLinks(state)
	if err != nil {
		return err
	}

	items, err := os.ReadDir(m.symlinkDir)
	if err != nil {
		return fmt.Errorf("read symlink dir: %w", err)
	}

	for _, item := range items {
		name := item.Name()
		linkPath := filepath.Join(m.symlinkDir, name)
		info, err := os.Lstat(linkPath)
		if err != nil {
			return fmt.Errorf("lstat symlink: %w", err)
		}

		target, wanted := desired[name]
		if info.Mode()&os.ModeSymlink == 0 {
			if wanted {
				return fmt.Errorf("refusing to overwrite non-symlink path: %s", linkPath)
			}
			continue
		}

		currentTarget, err := os.Readlink(linkPath)
		if err != nil {
			return fmt.Errorf("read symlink: %w", err)
		}

		if !wanted || currentTarget != target {
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("remove stale symlink: %w", err)
			}
		}
	}

	for name, target := range desired {
		if err := m.ensureLink(name, target); err != nil {
			return err
		}
	}

	return nil
}

func (m *SymlinkManager) ensureLink(name, target string) error {
	linkPath := filepath.Join(m.symlinkDir, name)
	info, err := os.Lstat(linkPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to overwrite non-symlink path: %s", linkPath)
		}

		currentTarget, err := os.Readlink(linkPath)
		if err != nil {
			return fmt.Errorf("read symlink: %w", err)
		}
		if currentTarget == target {
			return nil
		}
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("remove conflicting symlink: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("lstat symlink: %w", err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}
	return nil
}

func (m *SymlinkManager) buildDesiredLinks(state internal.MetadataState) (map[string]string, error) {
	names := make([]string, 0, len(state.Entries))
	for name := range state.Entries {
		names = append(names, name)
	}
	sort.Strings(names)

	desired := make(map[string]string)
	used := make(map[string]string)

	for _, pkg := range names {
		entry := state.Entries[pkg]
		binPaths := append([]string(nil), entry.BinPaths...)
		sort.Strings(binPaths)

		for _, target := range binPaths {
			base := filepath.Base(target)
			if base == "." || base == string(filepath.Separator) || base == "" {
				continue
			}

			candidate := base
			if existing, ok := used[candidate]; ok && existing != target {
				candidate = fmt.Sprintf("%s-%s", pkg, base)
				if existing2, taken := used[candidate]; taken && existing2 != target {
					for i := 2; ; i++ {
						next := fmt.Sprintf("%s-%s-%d", pkg, base, i)
						if existing3, taken3 := used[next]; !taken3 || existing3 == target {
							candidate = next
							break
						}
					}
				}
			}

			used[candidate] = target
			desired[candidate] = target
		}
	}

	return desired, nil
}
