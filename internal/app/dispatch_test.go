package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchName(t *testing.T) {
	empty := func(string) string { return "" }

	tests := []struct {
		base  string
		env   func(string) string
		isCLI bool
		name  string
	}{
		{"shortcuts", empty, true, ""},
		{"shortcuts.exe", empty, true, ""},
		{"", empty, true, ""},
		{"deploy", empty, false, "deploy"},
		{"deploy", func(k string) string {
			if k == "SHORTCUTS_CLI" {
				return "1"
			}
			return ""
		}, true, ""},
	}
	for _, tt := range tests {
		name, isCLI := DispatchName(tt.base, Deps{Getenv: tt.env})
		if isCLI != tt.isCLI || name != tt.name {
			t.Errorf("DispatchName(%q) = (%q, %v), want (%q, %v)", tt.base, name, isCLI, tt.name, tt.isCLI)
		}
	}
}

func testDeps(t *testing.T) (Deps, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	if err := os.MkdirAll(filepath.Join(root, "home"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Shims only need a plausible target in this test; dispatch is exercised
	// in-process via Main, not through the symlink.
	t.Setenv("SHORTCUTS_BINARY", "/bin/sh")

	var out, errOut bytes.Buffer
	deps := Deps{
		Stdin:   strings.NewReader(""),
		Stdout:  &out,
		Stderr:  &errOut,
		InTTY:   false,
		OutTTY:  false,
		Getenv:  os.Getenv,
		Environ: os.Environ,
		Getwd:   os.Getwd,
	}
	return deps, &out, &errOut
}

func TestAddAndDispatch(t *testing.T) {
	deps, out, errOut := testDeps(t)

	if code := Main([]string{"shortcuts", "add", "greet", "-c", `printf 'hello %s\n' {{1}}`}, deps); code != 0 {
		t.Fatalf("add failed: code=%d stderr=%s", code, errOut.String())
	}

	shimPath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "shortcuts", "bin", "greet")
	info, err := os.Lstat(shimPath)
	if err != nil {
		t.Fatalf("shim missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("shim is not a symlink")
	}

	// Run through dispatch, as if invoked by the shim name.
	out.Reset()
	errOut.Reset()
	if code := Main([]string{"greet", "world"}, deps); code != 0 {
		t.Fatalf("dispatch failed: code=%d stderr=%s", code, errOut.String())
	}
	if got := out.String(); got != "hello world\n" {
		t.Fatalf("stdout = %q", got)
	}

	// Arguments must not be interpreted by the shell.
	out.Reset()
	errOut.Reset()
	if code := Main([]string{"greet", "; echo pwned"}, deps); code != 0 {
		t.Fatalf("dispatch failed: code=%d stderr=%s", code, errOut.String())
	}
	if got := out.String(); got != "hello ; echo pwned\n" {
		t.Fatalf("injection result = %q", got)
	}
}

func TestDispatchExitCodeAndUnknown(t *testing.T) {
	deps, _, errOut := testDeps(t)

	if code := Main([]string{"shortcuts", "add", "fail", "-c", "exit 5"}, deps); code != 0 {
		t.Fatalf("add failed: %s", errOut.String())
	}
	if code := Main([]string{"fail"}, deps); code != 5 {
		t.Fatalf("exit code = %d, want 5", code)
	}
	if code := Main([]string{"does-not-exist"}, deps); code != 127 {
		t.Fatalf("unknown shim code = %d, want 127", code)
	}
}

func TestDryRun(t *testing.T) {
	deps, out, errOut := testDeps(t)
	if code := Main([]string{"shortcuts", "add", "deploy", "-c", "echo {{args}}"}, deps); code != 0 {
		t.Fatalf("add failed: %s", errOut.String())
	}
	out.Reset()
	if code := Main([]string{"shortcuts", "run", "--dry-run", "deploy", "a b"}, deps); code != 0 {
		t.Fatalf("dry-run failed: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "echo 'a b'") {
		t.Fatalf("dry-run output = %q", out.String())
	}
}
