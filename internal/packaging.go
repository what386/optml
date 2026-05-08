package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DiscoverOptEntries scans optRoot and builds metadata entries for existing installs.
func DiscoverOptEntries(optRoot string) (MetadataState, error) {
	items, err := os.ReadDir(optRoot)
	if err != nil {
		return MetadataState{}, fmt.Errorf("read opt root: %w", err)
	}

	now := time.Now().UTC()
	state := MetadataState{
		Entries: make(map[string]OptEntry),
	}

	for _, item := range items {
		name := item.Name()
		fullPath := filepath.Join(optRoot, name)

		info, err := item.Info()
		if err != nil {
			return MetadataState{}, fmt.Errorf("read entry info %q: %w", fullPath, err)
		}

		if info.IsDir() {
			binPaths, err := FindExecutables(fullPath)
			if err != nil {
				return MetadataState{}, fmt.Errorf("find executables for %q: %w", fullPath, err)
			}

			sort.Strings(binPaths)
			state.Entries[name] = OptEntry{
				Name:        name,
				RootDir:     fullPath,
				BinPaths:    binPaths,
				Managed:     false,
				InstalledAt: info.ModTime().UTC(),
				UpdatedAt:   now,
			}
			continue
		}

		if info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			state.Entries[name] = OptEntry{
				Name:        name,
				RootDir:     fullPath,
				BinPaths:    []string{fullPath},
				Managed:     false,
				InstalledAt: info.ModTime().UTC(),
				UpdatedAt:   now,
			}
		}
	}

	return state, nil
}

// RefreshMetadata discovers entries under optRoot and saves them to metadataPath.
func RefreshMetadata(optRoot, metadataPath string) (MetadataState, error) {
	state, err := DiscoverOptEntries(optRoot)
	if err != nil {
		return MetadataState{}, err
	}

	store := NewMetadataStore(metadataPath)
	if err := store.Save(state); err != nil {
		return MetadataState{}, err
	}

	return state, nil
}

// WriteInstallChecksum computes a checksum for the installed content under installRoot
// and writes it to installRoot/content.sha256.
func WriteInstallChecksum(installRoot string) (string, error) {
	contentPath, err := resolveInstallContentPath(installRoot)
	if err != nil {
		return "", err
	}

	sum, err := ChecksumPath(contentPath)
	if err != nil {
		return "", fmt.Errorf("compute install checksum: %w", err)
	}

	checksumFile := filepath.Join(installRoot, "content.sha256")
	payload := []byte(sum + "\n")
	if err := os.WriteFile(checksumFile, payload, 0o644); err != nil {
		return "", fmt.Errorf("write checksum file: %w", err)
	}

	return sum, nil
}

// VerifyInstallChecksum validates installRoot/content.sha256 against current content.
func VerifyInstallChecksum(installRoot string) error {
	checksumFile := filepath.Join(installRoot, "content.sha256")
	data, err := os.ReadFile(checksumFile)
	if err != nil {
		return fmt.Errorf("read checksum file: %w", err)
	}

	expected := strings.TrimSpace(string(data))
	contentPath, err := resolveInstallContentPath(installRoot)
	if err != nil {
		return err
	}

	return VerifyChecksum(contentPath, expected)
}

func resolveInstallContentPath(installRoot string) (string, error) {
	items, err := os.ReadDir(installRoot)
	if err != nil {
		return "", fmt.Errorf("read install root: %w", err)
	}

	contentCandidates := make([]os.DirEntry, 0, len(items))
	for _, item := range items {
		if item.Name() == "content.sha256" {
			continue
		}
		contentCandidates = append(contentCandidates, item)
	}

	if len(contentCandidates) == 0 {
		return installRoot, nil
	}
	if len(contentCandidates) == 1 {
		return filepath.Join(installRoot, contentCandidates[0].Name()), nil
	}

	// Multiple top-level items: checksum the full install root.
	return installRoot, nil
}
