package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"optml/internal/storage"
)

const (
	defaultShellPathsFile = "/opt/optml/paths.sh"
	defaultSymlinkPath    = "/opt/optml/bin"
)

const (
	bourneHookLine = `[ -f /opt/optml/paths.sh ] && source /opt/optml/paths.sh`
	fishHookLine   = `test -f /opt/optml/paths.sh; and source /opt/optml/paths.sh`
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

func (m *ShellManager) InitHooks() error {
	home, err := resolveInvokerHomeDir()
	if err != nil {
		return err
	}

	profiles, err := detectShellProfiles(home)
	if err != nil {
		return err
	}

	for _, p := range profiles {
		if err := ensureProfileHook(p.path, p.line); err != nil {
			return err
		}
	}
	return nil
}

func (m *ShellManager) CleanHooks() error {
	home, err := resolveInvokerHomeDir()
	if err != nil {
		return err
	}

	profiles, err := detectShellProfiles(home)
	if err != nil {
		return err
	}

	for _, p := range profiles {
		if err := removeProfileHook(p.path); err != nil {
			return err
		}
	}
	return nil
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

type shellProfile struct {
	path string
	line string
}

func resolveInvokerHomeDir() (string, error) {
	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil && strings.TrimSpace(u.HomeDir) != "" {
			return u.HomeDir, nil
		}
	}
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home != "" {
		return home, nil
	}
	return "", fmt.Errorf("unable to resolve home directory")
}

func detectShellProfiles(home string) ([]shellProfile, error) {
	const shellsFile = "/etc/shells"
	f, err := os.Open(shellsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", shellsFile, err)
	}
	defer f.Close()

	seen := make(map[string]shellProfile)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		shellName := strings.ToLower(filepath.Base(line))
		switch shellName {
		case "bash", "sh":
			p := filepath.Join(home, ".bashrc")
			seen[p] = shellProfile{path: p, line: bourneHookLine}
		case "zsh":
			p := filepath.Join(home, ".zshrc")
			seen[p] = shellProfile{path: p, line: bourneHookLine}
		case "fish":
			p := filepath.Join(home, ".config", "fish", "config.fish")
			seen[p] = shellProfile{path: p, line: fishHookLine}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", shellsFile, err)
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]shellProfile, 0, len(paths))
	for _, p := range paths {
		out = append(out, seen[p])
	}
	return out, nil
}

func ensureProfileHook(profilePath, hookLine string) error {
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return fmt.Errorf("mkdir profile dir: %w", err)
	}

	content := ""
	if b, err := os.ReadFile(profilePath); err == nil {
		content = string(b)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read profile %s: %w", profilePath, err)
	}

	if strings.Contains(content, hookLine) {
		return nil
	}

	appendText := hookLine + "\n"
	if strings.TrimSpace(content) != "" && !strings.HasSuffix(content, "\n") {
		appendText = "\n" + appendText
	}
	if strings.TrimSpace(content) != "" {
		appendText = "\n" + appendText
	}

	f, err := os.OpenFile(profilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open profile %s: %w", profilePath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(appendText); err != nil {
		return fmt.Errorf("write profile hook %s: %w", profilePath, err)
	}
	return nil
}

func removeProfileHook(profilePath string) error {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read profile %s: %w", profilePath, err)
	}

	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == bourneHookLine || trimmed == fishHookLine {
			continue
		}
		out = append(out, line)
	}
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write profile %s: %w", profilePath, err)
	}
	return nil
}
