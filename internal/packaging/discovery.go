package packaging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"optml/internal/storage"
)

func DiscoverOptEntries(optRoot string) (storage.MetadataState, error) {
	items, err := os.ReadDir(optRoot)
	if err != nil {
		return storage.MetadataState{}, fmt.Errorf("read opt root: %w", err)
	}

	now := time.Now().UTC()
	state := storage.MetadataState{Entries: make(map[string]storage.OptEntry)}
	for _, item := range items {
		name := item.Name()
		fullPath := filepath.Join(optRoot, name)
		info, err := item.Info()
		if err != nil {
			return storage.MetadataState{}, fmt.Errorf("read entry info %q: %w", fullPath, err)
		}
		if info.IsDir() {
			binPaths, err := FindExecutables(fullPath)
			if err != nil {
				return storage.MetadataState{}, fmt.Errorf("find executables for %q: %w", fullPath, err)
			}
			sort.Strings(binPaths)
			state.Entries[name] = storage.OptEntry{
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
			state.Entries[name] = storage.OptEntry{
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

func RefreshMetadata(optRoot, metadataPath string) (storage.MetadataState, error) {
	state, err := DiscoverOptEntries(optRoot)
	if err != nil {
		return storage.MetadataState{}, err
	}
	store := storage.NewMetadataStore(metadataPath)
	if err := store.Save(state); err != nil {
		return storage.MetadataState{}, err
	}
	return state, nil
}
