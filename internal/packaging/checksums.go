package packaging

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
		return "", fmt.Errorf("expected file, got directory: %s", path)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file for checksum: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ChecksumDirectory(rootDir string) (string, error) {
	rootInfo, err := os.Stat(rootDir)
	if err != nil {
		return "", fmt.Errorf("stat directory for checksum: %w", err)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("expected directory, got file: %s", rootDir)
	}
	type fileDigest struct{ relPath, digest string }
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
		sum, err := ChecksumFile(path)
		if err != nil {
			return err
		}
		files = append(files, fileDigest{relPath: filepath.ToSlash(rel), digest: sum})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })
	h := sha256.New()
	for _, f := range files {
		line := f.relPath + ":" + f.digest + "\n"
		if _, err := io.WriteString(h, line); err != nil {
			return "", fmt.Errorf("hash directory manifest: %w", err)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

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

func VerifyChecksum(path, expected string) error {
	if strings.TrimSpace(expected) == "" {
		return fmt.Errorf("expected checksum is empty")
	}
	actual, err := ChecksumPath(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("integrity check failed: expected=%s actual=%s path=%s", expected, actual, path)
	}
	return nil
}

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
	if err := os.WriteFile(checksumFile, []byte(sum+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("write checksum file: %w", err)
	}
	return sum, nil
}

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
	candidates := make([]os.DirEntry, 0, len(items))
	for _, item := range items {
		if item.Name() == "content.sha256" {
			continue
		}
		candidates = append(candidates, item)
	}
	if len(candidates) == 0 {
		return installRoot, nil
	}
	if len(candidates) == 1 {
		return filepath.Join(installRoot, candidates[0].Name()), nil
	}
	return installRoot, nil
}
