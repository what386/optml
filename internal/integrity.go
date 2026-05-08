package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"optml/internal/integration"
	"optml/internal/packaging"
	"optml/internal/storage"
)

type IntegrityConfig struct {
	OptRoot      string
	MetadataPath string
	PathsFile    string
	SymlinkDir   string
	Fix          bool
}

type IntegrityResult struct {
	OK    int
	Warn  int
	Fail  int
	Lines []string
}

func (r *IntegrityResult) addOK(format string, args ...any) {
	r.OK++
	r.Lines = append(r.Lines, fmt.Sprintf("OK: "+format, args...))
}

func (r *IntegrityResult) addWarn(format string, args ...any) {
	r.Warn++
	r.Lines = append(r.Lines, fmt.Sprintf("WARN: "+format, args...))
}

func (r *IntegrityResult) addFail(format string, args ...any) {
	r.Fail++
	r.Lines = append(r.Lines, fmt.Sprintf("FAIL: "+format, args...))
}

func RunIntegrityCheck(cfg IntegrityConfig) (IntegrityResult, error) {
	if cfg.Fix {
		state, err := packaging.RefreshMetadata(cfg.OptRoot, cfg.MetadataPath)
		if err != nil {
			return IntegrityResult{}, fmt.Errorf("fix: refresh metadata: %w", err)
		}
		shellMgr := integration.NewShellManager(cfg.PathsFile)
		if err := shellMgr.RebuildFromState(state); err != nil {
			return IntegrityResult{}, fmt.Errorf("fix: rebuild shell integration: %w", err)
		}
		symlinkMgr := integration.NewSymlinkManager(cfg.SymlinkDir)
		if err := symlinkMgr.RebuildFromState(state); err != nil {
			return IntegrityResult{}, fmt.Errorf("fix: rebuild symlink integration: %w", err)
		}
	}

	result := IntegrityResult{}
	store := storage.NewMetadataStore(cfg.MetadataPath)
	state, err := store.Load()
	if err != nil {
		result.addFail("metadata file unreadable (%s): %v", cfg.MetadataPath, err)
		return result, nil
	}
	result.addOK("metadata file readable (%s)", cfg.MetadataPath)

	if _, err := os.Stat(cfg.OptRoot); err != nil {
		result.addFail("opt root unavailable (%s): %v", cfg.OptRoot, err)
	} else {
		result.addOK("opt root available (%s)", cfg.OptRoot)
	}
	if _, err := os.Stat(cfg.PathsFile); err != nil {
		result.addFail("shell integration file unavailable (%s): %v", cfg.PathsFile, err)
	} else {
		result.addOK("shell integration file available (%s)", cfg.PathsFile)
	}
	if _, err := os.Stat(cfg.SymlinkDir); err != nil {
		result.addFail("symlink directory unavailable (%s): %v", cfg.SymlinkDir, err)
	} else {
		result.addOK("symlink directory available (%s)", cfg.SymlinkDir)
	}

	allBinPaths := make(map[string]struct{})
	names := make([]string, 0, len(state.Entries))
	for name := range state.Entries {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := state.Entries[name]

		info, err := os.Stat(entry.RootDir)
		if err != nil {
			result.addFail("entry %q root missing (%s): %v", name, entry.RootDir, err)
		} else if !info.IsDir() {
			result.addFail("entry %q root is not a directory (%s)", name, entry.RootDir)
		} else {
			result.addOK("entry %q root exists", name)
		}

		for _, bin := range entry.BinPaths {
			allBinPaths[bin] = struct{}{}
			binInfo, err := os.Stat(bin)
			if err != nil {
				result.addFail("entry %q bin missing (%s): %v", name, bin, err)
				continue
			}
			if !binInfo.Mode().IsRegular() {
				result.addFail("entry %q bin is not a regular file (%s)", name, bin)
				continue
			}
			if binInfo.Mode().Perm()&0o111 == 0 {
				result.addFail("entry %q bin not executable (%s)", name, bin)
				continue
			}
			result.addOK("entry %q bin healthy (%s)", name, bin)
		}

		if entry.Checksum == "" {
			result.addWarn("entry %q has no checksum", name)
		} else if err := packaging.VerifyInstallChecksum(entry.RootDir); err != nil {
			result.addFail("entry %q checksum invalid: %v", name, err)
		} else {
			result.addOK("entry %q checksum valid", name)
		}
	}

	items, err := os.ReadDir(cfg.SymlinkDir)
	if err != nil {
		result.addFail("unable to read symlink dir (%s): %v", cfg.SymlinkDir, err)
		return result, nil
	}

	for _, item := range items {
		linkPath := filepath.Join(cfg.SymlinkDir, item.Name())
		info, err := os.Lstat(linkPath)
		if err != nil {
			result.addFail("symlink entry unreadable (%s): %v", linkPath, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			result.addWarn("non-symlink file present in symlink dir (%s)", linkPath)
			continue
		}
		target, err := os.Readlink(linkPath)
		if err != nil {
			result.addFail("unable to read symlink target (%s): %v", linkPath, err)
			continue
		}
		if _, ok := allBinPaths[target]; !ok {
			result.addFail("symlink target not present in metadata bin paths (%s -> %s)", linkPath, target)
		} else {
			result.addOK("symlink healthy (%s -> %s)", linkPath, target)
		}
	}

	return result, nil
}
