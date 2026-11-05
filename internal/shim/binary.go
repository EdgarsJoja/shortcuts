package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveBinary returns the absolute path of the running shortcuts binary.
// It refuses ephemeral `go run`/`go test` binaries so development invocations
// cannot create permanent shims to a temporary executable.
func ResolveBinary() (string, error) {
	if p := os.Getenv("SHORTCUTS_BINARY"); p != "" {
		if !filepath.IsAbs(p) {
			return "", fmt.Errorf("SHORTCUTS_BINARY must be an absolute path, got %q", p)
		}
		return filepath.Clean(p), nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if !filepath.IsAbs(exe) {
		return "", fmt.Errorf("cannot determine an absolute binary path (%q); set SHORTCUTS_BINARY", exe)
	}
	if strings.Contains(exe, string(filepath.Separator)+"go-build") || strings.HasSuffix(exe, ".test") {
		return "", fmt.Errorf("refusing to manage shims for a temporary binary (%s); install shortcuts or set SHORTCUTS_BINARY", exe)
	}
	return filepath.Clean(exe), nil
}
