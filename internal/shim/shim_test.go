package shim

import (
	"os"
	"path/filepath"
	"testing"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "shortcuts")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Manager{Dir: filepath.Join(t.TempDir(), "bin"), Binary: binary}
}

func TestCreateRemove(t *testing.T) {
	m := testManager(t)
	if err := m.Create("deploy"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	target := filepath.Join(m.Dir, "deploy")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", target)
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != m.Binary {
		t.Fatalf("link target = %q, want %q", got, m.Binary)
	}

	// Idempotent.
	if err := m.Create("deploy"); err != nil {
		t.Fatalf("second Create: %v", err)
	}

	if err := m.Remove("deploy"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("link still exists: %v", err)
	}
}

func TestCreateRefusesForeignFile(t *testing.T) {
	m := testManager(t)
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(m.Dir, "deploy")
	if err := os.WriteFile(target, []byte("user data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := m.Create("deploy"); err == nil {
		t.Fatal("Create overwrote a foreign file without Force")
	}
	m.Force = true
	if err := m.Create("deploy"); err != nil {
		t.Fatalf("Create with Force: %v", err)
	}
}

func TestRenameKeepsOldOnCollision(t *testing.T) {
	m := testManager(t)
	if err := m.Create("one"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.Dir, "two"), []byte("user data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := m.Rename("one", "two"); err == nil {
		t.Fatal("Rename succeeded over a foreign file")
	}
	if _, err := os.Lstat(filepath.Join(m.Dir, "one")); err != nil {
		t.Fatalf("old link was removed on failed rename: %v", err)
	}
}

func TestSyncCreatesAndPrunes(t *testing.T) {
	m := testManager(t)
	if _, err := m.Sync([]string{"one", "two"}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if _, err := m.Sync([]string{"one"}); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if _, err := os.Lstat(filepath.Join(m.Dir, "one")); err != nil {
		t.Fatalf("one missing: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(m.Dir, "two")); !os.IsNotExist(err) {
		t.Fatalf("two was not pruned: %v", err)
	}
}

func TestLookPathOutside(t *testing.T) {
	shimDir := filepath.Join(t.TempDir(), "shims")
	otherDir := t.TempDir()

	if err := os.MkdirAll(shimDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "deploy"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	pathEnv := shimDir + string(os.PathListSeparator) + otherDir
	got, ok := LookPathOutside("deploy", shimDir, pathEnv)
	if !ok {
		t.Fatal("LookPathOutside did not find deploy")
	}
	if got != filepath.Join(otherDir, "deploy") {
		t.Fatalf("found %q", got)
	}
}

func TestResolveBinaryRejectsRelativeOverride(t *testing.T) {
	t.Setenv("SHORTCUTS_BINARY", "relative/path")
	if _, err := ResolveBinary(); err == nil {
		t.Fatal("expected error for relative SHORTCUTS_BINARY")
	}
}
