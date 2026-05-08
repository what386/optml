package tests

import (
	"os"
	"path/filepath"
	"testing"

	"optml/internal/packaging"
)

func TestFindExecutablesFiltersDeniedSuffixes(t *testing.T) {
	root := t.TempDir()

	mustMkdirAll(t, filepath.Join(root, "bin"), 0o755)
	mustWriteExec(t, filepath.Join(root, "bin", "app"))
	mustWriteExec(t, filepath.Join(root, "bin", "libxul.so"))
	mustWriteExec(t, filepath.Join(root, "bin", "libnss3.so.1"))
	mustWriteExec(t, filepath.Join(root, "bin", "libhelper.a"))
	mustWriteExec(t, filepath.Join(root, "bin", "plugin.la"))
	mustWriteExec(t, filepath.Join(root, "bin", "obj.o"))
	mustWriteExec(t, filepath.Join(root, "bin", "native.dylib"))

	got, err := packaging.FindExecutables(root)
	if err != nil {
		t.Fatalf("FindExecutables returned error: %v", err)
	}

	if len(got) != 1 || got[0] != filepath.Join(root, "bin", "app") {
		t.Fatalf("unexpected executables: %#v", got)
	}
}

func TestFindExecutablesPrioritizesBinAndSbin(t *testing.T) {
	root := t.TempDir()

	mustWriteExec(t, filepath.Join(root, "rootcmd"))
	mustMkdirAll(t, filepath.Join(root, "bin"), 0o755)
	mustWriteExec(t, filepath.Join(root, "bin", "tool"))
	mustMkdirAll(t, filepath.Join(root, "sbin"), 0o755)
	mustWriteExec(t, filepath.Join(root, "sbin", "sys"))

	got, err := packaging.FindExecutables(root)
	if err != nil {
		t.Fatalf("FindExecutables returned error: %v", err)
	}

	expected := []string{
		filepath.Join(root, "bin", "tool"),
		filepath.Join(root, "sbin", "sys"),
	}
	if len(got) != len(expected) {
		t.Fatalf("unexpected executable count: got=%d want=%d (%#v)", len(got), len(expected), got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("unexpected executable at index %d: got=%s want=%s", i, got[i], expected[i])
		}
	}
}

func TestFindExecutablesFallsBackToRootNonRecursive(t *testing.T) {
	root := t.TempDir()

	mustWriteExec(t, filepath.Join(root, "rootcmd"))
	mustMkdirAll(t, filepath.Join(root, "nested"), 0o755)
	mustWriteExec(t, filepath.Join(root, "nested", "nestedcmd"))

	got, err := packaging.FindExecutables(root)
	if err != nil {
		t.Fatalf("FindExecutables returned error: %v", err)
	}

	if len(got) != 1 || got[0] != filepath.Join(root, "rootcmd") {
		t.Fatalf("fallback should only include root-level executables: %#v", got)
	}
}

func mustMkdirAll(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

func mustWriteExec(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
