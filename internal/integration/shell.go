package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"optml/internal/storage"
)

const (
	defaultShellPathsFile = "/opt/optml/paths.sh"
	defaultSymlinkPath    = "/opt/optml/bin"
)

type ShellManager struct {
	pathsFile string
}

func NewShellManager(pathsFile string) *ShellManager {
	if strings.TrimSpace(pathsFile) == "" {
		pathsFile = defaultShellPathsFile
	}
	return &ShellManager{pathsFile: pathsFile}
}

func (m *ShellManager) AddEntry(entry storage.OptEntry) error {
	return m.writeState(map[string]storage.OptEntry{entry.Name: entry})
}

func (m *ShellManager) RemoveEntry(entry storage.OptEntry) error {
	return m.writeState(map[string]storage.OptEntry{})
}

func (m *ShellManager) RebuildFromState(state storage.MetadataState) error {
	return m.writeState(state.Entries)
}

func (m *ShellManager) writeState(entries map[string]storage.OptEntry) error {
	if err := os.MkdirAll(filepath.Dir(m.pathsFile), 0o755); err != nil {
		return fmt.Errorf("mkdir paths dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := []string{
		"#!/bin/bash",
		"# optml managed PATH additions",
		fmt.Sprintf(`export PATH="%s:$PATH"`, defaultSymlinkPath),
		"",
	}

	for _, name := range names {
		entry := entries[name]
		for _, path := range entry.PathDirs {
			if strings.TrimSpace(path) == "" {
				continue
			}
			escaped := strings.ReplaceAll(path, `"`, `\\"`)
			escaped = strings.ReplaceAll(escaped, `$`, `\\$`)
			lines = append(lines, fmt.Sprintf(`export PATH="%s:$PATH"`, escaped))
		}
	}

	content := strings.Join(lines, "\n") + "\n"
	tmpPath := m.pathsFile + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write temp paths file: %w", err)
	}
	if err := os.Rename(tmpPath, m.pathsFile); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace paths file: %w", err)
	}

	return nil
}
