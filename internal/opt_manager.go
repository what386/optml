package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultOptRoot = "/opt"

// OptManager manages filesystem installs under /opt and keeps metadata in sync.
type OptManager struct {
	optRoot string
	store   *MetadataStore
}

func NewOptManager(store *MetadataStore) *OptManager {
	return NewOptManagerWithRoot(defaultOptRoot, store)
}

func NewOptManagerWithRoot(optRoot string, store *MetadataStore) *OptManager {
	if strings.TrimSpace(optRoot) == "" {
		optRoot = defaultOptRoot
	}
	if store == nil {
		store = NewMetadataStore("")
	}
	return &OptManager{
		optRoot: filepath.Clean(optRoot),
		store:   store,
	}
}

// Add installs srcPath into /opt/<key> and writes metadata.
// Files and directories are copied; archives are extracted.
func (m *OptManager) Add(key, srcPath string) (OptEntry, error) {
	if strings.TrimSpace(key) == "" {
		return OptEntry{}, WrapInvalidInput("key is required")
	}
	if strings.TrimSpace(srcPath) == "" {
		return OptEntry{}, WrapInvalidInput("srcPath is required")
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return OptEntry{}, fmt.Errorf("stat source: %w", err)
	}

	destRoot, err := safeJoin(m.optRoot, key)
	if err != nil {
		return OptEntry{}, fmt.Errorf("%w: invalid key %q", ErrPathTraversal, key)
	}

	if _, err := os.Stat(destRoot); err == nil {
		return OptEntry{}, WrapAlreadyInstalled("destination already exists: %s", destRoot)
	} else if !os.IsNotExist(err) {
		return OptEntry{}, fmt.Errorf("check destination: %w", err)
	}

	switch {
	case srcInfo.IsDir():
		if err := copyDir(srcPath, destRoot); err != nil {
			return OptEntry{}, err
		}
	default:
		if _, derr := DetectArchiveType(srcPath); derr == nil {
			if err := ExtractArchive(srcPath, destRoot); err != nil {
				return OptEntry{}, err
			}
		} else {
			if err := os.MkdirAll(destRoot, 0o755); err != nil {
				return OptEntry{}, fmt.Errorf("mkdir destination: %w", err)
			}
			targetFile := filepath.Join(destRoot, filepath.Base(srcPath))
			if err := copyFile(srcPath, targetFile); err != nil {
				return OptEntry{}, err
			}
		}
	}

	now := time.Now().UTC()
	checksum, err := WriteInstallChecksum(destRoot)
	if err != nil {
		return OptEntry{}, fmt.Errorf("write install checksum: %w", err)
	}

	binPaths, err := FindExecutables(destRoot)
	if err != nil {
		return OptEntry{}, fmt.Errorf("discover executables: %w", err)
	}

	entry := OptEntry{
		Name:        key,
		RootDir:     destRoot,
		BinPaths:    binPaths,
		Managed:     true,
		InstalledAt: now,
		UpdatedAt:   now,
		Checksum:    checksum,
	}

	if err := m.store.Upsert(key, entry); err != nil {
		return OptEntry{}, fmt.Errorf("persist metadata: %w", err)
	}

	return entry, nil
}

// Remove deletes /opt/<key> and removes the corresponding metadata entry.
func (m *OptManager) Remove(key string) error {
	if strings.TrimSpace(key) == "" {
		return WrapInvalidInput("key is required")
	}

	destRoot, err := safeJoin(m.optRoot, key)
	if err != nil {
		return fmt.Errorf("%w: invalid key %q", ErrPathTraversal, key)
	}

	if err := os.RemoveAll(destRoot); err != nil {
		return fmt.Errorf("remove install root: %w", err)
	}

	if err := m.store.Delete(key); err != nil && err != ErrNotFound {
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

// FindExecutables recursively discovers executable files under rootDir.
func FindExecutables(rootDir string) ([]string, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("stat root dir: %w", err)
	}
	if !info.IsDir() {
		return nil, WrapInvalidInput("rootDir must be a directory: %s", rootDir)
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
