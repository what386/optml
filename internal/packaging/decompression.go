package packaging

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ArchiveType string

const (
	ArchiveZip   ArchiveType = "zip"
	ArchiveTar   ArchiveType = "tar"
	ArchiveTarGz ArchiveType = "tar.gz"
	ArchiveTGZ   ArchiveType = "tgz"
)

var ErrUnsupportedArchive = errors.New("unsupported archive type")

func DetectArchiveType(path string) (ArchiveType, error) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return ArchiveZip, nil
	case strings.HasSuffix(lower, ".tar.gz"):
		return ArchiveTarGz, nil
	case strings.HasSuffix(lower, ".tgz"):
		return ArchiveTGZ, nil
	case strings.HasSuffix(lower, ".tar"):
		return ArchiveTar, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedArchive, path)
	}
}

func ExtractArchive(srcPath, destDir string) error {
	archiveType, err := DetectArchiveType(srcPath)
	if err != nil {
		return err
	}
	switch archiveType {
	case ArchiveZip:
		return extractZip(srcPath, destDir)
	case ArchiveTar:
		return extractTar(srcPath, destDir)
	case ArchiveTarGz, ArchiveTGZ:
		return extractTarGz(srcPath, destDir)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedArchive, srcPath)
	}
}

func extractZip(srcPath, destDir string) error {
	reader, err := zip.OpenReader(srcPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	for _, f := range reader.File {
		targetPath, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
				return fmt.Errorf("mkdir zip dir: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("mkdir zip parent: %w", err)
		}
		in, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip file: %w", err)
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			in.Close()
			return fmt.Errorf("create zip output: %w", err)
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			in.Close()
			return fmt.Errorf("copy zip file: %w", err)
		}
		if err := out.Close(); err != nil {
			in.Close()
			return fmt.Errorf("close zip output: %w", err)
		}
		if err := in.Close(); err != nil {
			return fmt.Errorf("close zip input: %w", err)
		}
	}
	return nil
}

func extractTar(srcPath, destDir string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open tar: %w", err)
	}
	defer f.Close()
	return extractTarStream(tar.NewReader(f), destDir)
}

func extractTarGz(srcPath, destDir string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open tar.gz: %w", err)
	}
	defer f.Close()
	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gzReader.Close()
	return extractTarStream(tar.NewReader(gzReader), destDir)
}

func extractTarStream(reader *tar.Reader, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	for {
		header, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read tar header: %w", err)
		}
		targetPath, err := safeJoin(destDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("mkdir tar dir: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("mkdir tar parent: %w", err)
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create tar output: %w", err)
			}
			if _, err := io.Copy(out, reader); err != nil {
				out.Close()
				return fmt.Errorf("copy tar file: %w", err)
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("close tar output: %w", err)
			}
		default:
			continue
		}
	}
}

func safeJoin(baseDir, entryName string) (string, error) {
	cleanBase := filepath.Clean(baseDir)
	cleanEntry := filepath.Clean(entryName)
	target := filepath.Join(cleanBase, cleanEntry)
	rel, err := filepath.Rel(cleanBase, target)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry escapes destination: %s", entryName)
	}
	return target, nil
}
