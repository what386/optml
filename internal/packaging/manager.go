package packaging

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"optml/internal/storage"
)

const DefaultOptRoot = "/opt"

type Manager struct {
	optRoot string
	store   *storage.MetadataStore
}

func NewManager(store *storage.MetadataStore) *Manager {
	return NewManagerWithRoot(DefaultOptRoot, store)
}

func NewManagerWithRoot(optRoot string, store *storage.MetadataStore) *Manager {
	if strings.TrimSpace(optRoot) == "" {
		optRoot = DefaultOptRoot
	}
	if store == nil {
		store = storage.NewMetadataStore("")
	}
	return &Manager{optRoot: filepath.Clean(optRoot), store: store}
}

func (m *Manager) Add(key, srcPath string) (storage.OptEntry, error) {
	if strings.TrimSpace(key) == "" {
		return storage.OptEntry{}, fmt.Errorf("invalid input: key is required")
	}
	if strings.TrimSpace(srcPath) == "" {
		return storage.OptEntry{}, fmt.Errorf("invalid input: srcPath is required")
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return storage.OptEntry{}, fmt.Errorf("stat source: %w", err)
	}

	destRoot, err := safeJoin(m.optRoot, key)
	if err != nil {
		return storage.OptEntry{}, fmt.Errorf("invalid key %q: %w", key, err)
	}
	if _, err := os.Stat(destRoot); err == nil {
		return storage.OptEntry{}, fmt.Errorf("already installed: destination exists %s", destRoot)
	} else if !os.IsNotExist(err) {
		return storage.OptEntry{}, fmt.Errorf("check destination: %w", err)
	}

	switch {
	case srcInfo.IsDir():
		if err := copyDir(srcPath, destRoot); err != nil {
			return storage.OptEntry{}, err
		}
	default:
		if _, derr := DetectArchiveType(srcPath); derr == nil {
			if err := ExtractArchive(srcPath, destRoot); err != nil {
				return storage.OptEntry{}, err
			}
		} else {
			if err := os.MkdirAll(destRoot, 0o755); err != nil {
				return storage.OptEntry{}, fmt.Errorf("mkdir destination: %w", err)
			}
			targetFile := filepath.Join(destRoot, filepath.Base(srcPath))
			if err := copyFile(srcPath, targetFile); err != nil {
				return storage.OptEntry{}, err
			}
		}
	}

	now := time.Now().UTC()
	checksum, err := WriteInstallChecksum(destRoot)
	if err != nil {
		return storage.OptEntry{}, fmt.Errorf("write install checksum: %w", err)
	}
	binPaths, err := FindExecutables(destRoot)
	if err != nil {
		return storage.OptEntry{}, fmt.Errorf("discover executables: %w", err)
	}
	sort.Strings(binPaths)
	pathDirs := []string(nil)
	if len(binPaths) == 0 {
		pathDirs, err = SelectFallbackPathDirs(destRoot)
		if err != nil {
			return storage.OptEntry{}, fmt.Errorf("discover fallback path dirs: %w", err)
		}
	}

	entry := storage.OptEntry{
		Name:        key,
		RootDir:     destRoot,
		BinPaths:    binPaths,
		PathDirs:    pathDirs,
		Managed:     true,
		InstalledAt: now,
		UpdatedAt:   now,
		Checksum:    checksum,
	}

	if err := handoffOwnershipIfSudo(destRoot); err != nil {
		return storage.OptEntry{}, fmt.Errorf("handoff ownership: %w", err)
	}

	if err := m.store.Upsert(key, entry); err != nil {
		return storage.OptEntry{}, fmt.Errorf("persist metadata: %w", err)
	}
	return entry, nil
}

func (m *Manager) Remove(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("invalid input: key is required")
	}
	destRoot, err := safeJoin(m.optRoot, key)
	if err != nil {
		return fmt.Errorf("invalid key %q: %w", key, err)
	}
	if err := os.RemoveAll(destRoot); err != nil {
		return fmt.Errorf("remove install root: %w", err)
	}
	if err := m.store.Delete(key); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("remove metadata: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("open destination file: %w", err)
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}

func copyDir(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk source dir: %w", err)
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("resolve relative path: %w", err)
		}
		targetPath := filepath.Join(dstDir, rel)
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("read dir entry info: %w", err)
		}
		if d.IsDir() {
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("mkdir target dir: %w", err)
			}
			return nil
		}
		return copyFile(path, targetPath)
	})
}
