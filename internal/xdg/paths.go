// Package xdg resolves the configuration and data locations used by shortcuts.
//
// Linux and macOS both follow the XDG base directory variables when set.
// Environment overrides exist primarily for tests:
//
//	SHORTCUTS_CONFIG     full path to config.toml
//	SHORTCUTS_DATA_HOME  data directory (default: $XDG_DATA_HOME/shortcuts)
//	SHORTCUTS_BIN_DIR    shim directory (default: <data dir>/bin)
package xdg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigPath returns the absolute path of the configuration file.
func ConfigPath() (string, error) {
	if v := os.Getenv("SHORTCUTS_CONFIG"); v != "" {
		return cleanAbs("SHORTCUTS_CONFIG", v)
	}
	if d := absEnv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "shortcuts", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "shortcuts", "config.toml"), nil
}

// DataDir returns the absolute data directory (not the shim directory).
func DataDir() (string, error) {
	if v := os.Getenv("SHORTCUTS_DATA_HOME"); v != "" {
		return cleanAbs("SHORTCUTS_DATA_HOME", v)
	}
	if d := absEnv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "shortcuts"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "shortcuts"), nil
}

// ShimDir returns the absolute directory that holds shortcut symlinks.
func ShimDir() (string, error) {
	if v := os.Getenv("SHORTCUTS_BIN_DIR"); v != "" {
		return cleanAbs("SHORTCUTS_BIN_DIR", v)
	}
	d, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "bin"), nil
}

func cleanAbs(name, v string) (string, error) {
	if !filepath.IsAbs(v) {
		return "", fmt.Errorf("%s must be an absolute path, got %q", name, v)
	}
	return filepath.Clean(v), nil
}

// absEnv returns the value of an XDG variable only when it is absolute, as
// required by the XDG base directory specification.
func absEnv(name string) string {
	v := os.Getenv(name)
	if v == "" || !filepath.IsAbs(v) {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(v))
}
