package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"optml/internal"
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

func (m *ShellManager) AddEntry(entry internal.OptEntry) error {
	state, err := m.loadStateFromPathsFile()
	if err != nil {
		return err
	}
	state[entry.Name] = entry
	return m.writeState(state)
}

func (m *ShellManager) RemoveEntry(entry internal.OptEntry) error {
	state, err := m.loadStateFromPathsFile()
	if err != nil {
		return err
	}
	delete(state, entry.Name)
	return m.writeState(state)
}

func (m *ShellManager) RebuildFromState(state internal.MetadataState) error {
	return m.writeState(state.Entries)
}

func (m *ShellManager) loadStateFromPathsFile() (map[string]internal.OptEntry, error) {
	content, err := os.ReadFile(m.pathsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]internal.OptEntry{}, nil
		}
		return nil, fmt.Errorf("read paths file: %w", err)
	}

	state := make(map[string]internal.OptEntry)
	for _, line := range strings.Split(string(content), "\n") {
		pathValue, ok := parseExportPath(strings.TrimSpace(line))
		if !ok {
			continue
		}
		if pathValue == defaultSymlinkPath {
			continue
		}

		clean := filepath.Clean(pathValue)
		name := filepath.Base(clean)
		root := clean
		if name == "bin" {
			root = filepath.Dir(clean)
			name = filepath.Base(root)
		}
		if name == "" || name == "." || name == string(filepath.Separator) {
			continue
		}

		state[name] = internal.OptEntry{
			Name:    name,
			RootDir: root,
		}
	}

	return state, nil
}

func (m *ShellManager) writeState(entries map[string]internal.OptEntry) error {
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
		path := bestPathEntry(entry)
		if path == "" {
			continue
		}
		escaped := strings.ReplaceAll(path, `"`, `\\"`)
		escaped = strings.ReplaceAll(escaped, `$`, `\\$`)
		lines = append(lines, fmt.Sprintf(`export PATH="%s:$PATH"`, escaped))
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

func bestPathEntry(entry internal.OptEntry) string {
	if strings.TrimSpace(entry.RootDir) == "" {
		return ""
	}
	binDir := filepath.Join(entry.RootDir, "bin")
	if info, err := os.Stat(binDir); err == nil && info.IsDir() {
		return binDir
	}
	return entry.RootDir
}

func parseExportPath(line string) (string, bool) {
	prefix := `export PATH="`
	suffix := `:$PATH"`
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
	inner = strings.ReplaceAll(inner, `\\$`, `$`)
	inner = strings.ReplaceAll(inner, `\\"`, `"`)
	return inner, true
}
