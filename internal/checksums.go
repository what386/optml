package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChecksumFile returns the SHA-256 checksum (hex) for a single file.
func ChecksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file for checksum: %w", err)
	}
	if info.IsDir() {
		return "", WrapInvalidInput("expected file, got directory: %s", path)
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file for checksum: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ChecksumDirectory returns a deterministic SHA-256 checksum (hex) for a directory tree.
// It hashes relative paths and file content digests in sorted order.
func ChecksumDirectory(rootDir string) (string, error) {
	rootInfo, err := os.Stat(rootDir)
	if err != nil {
		return "", fmt.Errorf("stat directory for checksum: %w", err)
	}
	if !rootInfo.IsDir() {
		return "", WrapInvalidInput("expected directory, got file: %s", rootDir)
	}

	type fileDigest struct {
		relPath string
		digest  string
	}

	files := make([]fileDigest, 0)
	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk directory for checksum: %w", walkErr)
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat entry for checksum: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("build relative path for checksum: %w", err)
		}

		fileSum, err := ChecksumFile(path)
		if err != nil {
			return err
		}

		files = append(files, fileDigest{
			relPath: filepath.ToSlash(rel),
			digest:  fileSum,
		})

		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})

	h := sha256.New()
	for _, f := range files {
		line := f.relPath + ":" + f.digest + "\n"
		if _, err := io.WriteString(h, line); err != nil {
			return "", fmt.Errorf("hash directory manifest: %w", err)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ChecksumPath computes a SHA-256 for either a file or directory.
func ChecksumPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat checksum path: %w", err)
	}
	if info.IsDir() {
		return ChecksumDirectory(path)
	}
	return ChecksumFile(path)
}

// VerifyChecksum compares an expected checksum with the actual checksum of a path.
func VerifyChecksum(path, expected string) error {
	if strings.TrimSpace(expected) == "" {
		return WrapInvalidInput("expected checksum is empty")
	}

	actual, err := ChecksumPath(path)
	if err != nil {
		return err
	}

	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("%w: expected=%s actual=%s path=%s", ErrIntegrityCheckFail, expected, actual, path)
	}

	return nil
}
