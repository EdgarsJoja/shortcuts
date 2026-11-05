package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// wireConfigIn decodes TOML into the concrete union representation.
type wireConfigIn struct {
	Version   int                       `toml:"version"`
	Theme     string                    `toml:"theme"`
	Shortcuts map[string]wireShortcutIn `toml:"shortcuts"`
}

type wireShortcutIn struct {
	Command     any    `toml:"command"`
	Description string `toml:"description"`
}

// wireConfigOut encodes the union as an untyped TOML value.
type wireConfigOut struct {
	Version   int                        `toml:"version"`
	Theme     string                     `toml:"theme,omitempty"`
	Shortcuts map[string]wireShortcutOut `toml:"shortcuts"`
}

type wireShortcutOut struct {
	Command     any    `toml:"command"`
	Description string `toml:"description,omitempty"`
}

// Load reads and validates the config at path. A missing file yields an empty
// version-1 config so first-run commands work without an explicit init.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return empty(), nil
		}
		return Config{}, err
	}

	var w wireConfigIn
	md, err := toml.Decode(string(b), &w)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if und := md.Undecoded(); len(und) > 0 {
		keys := make([]string, 0, len(und))
		for _, k := range und {
			// The encoder may emit an empty [shortcuts] table, and some
			// decoders report a shortcut's own table as undecoded even though
			// its fields were consumed. Neither is a typo.
			parts := []string(k)
			if len(parts) == 1 && parts[0] == "shortcuts" {
				continue
			}
			if len(parts) == 2 && parts[0] == "shortcuts" {
				if _, ok := w.Shortcuts[parts[1]]; ok {
					continue
				}
			}
			keys = append(keys, k.String())
		}
		if len(keys) > 0 {
			return Config{}, fmt.Errorf("parse %s: unknown keys: %v", path, keys)
		}
	}

	cfg := Config{Version: w.Version, Theme: w.Theme, Shortcuts: make(map[string]Shortcut, len(w.Shortcuts))}
	for name, sc := range w.Shortcuts {
		cmd, err := commandFromValue(sc.Command)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: shortcut %q: %w", path, name, err)
		}
		cfg.Shortcuts[name] = Shortcut{Command: cmd, Description: sc.Description}
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Version != 1 {
		return Config{}, fmt.Errorf("parse %s: unsupported config version %d", path, cfg.Version)
	}
	if err := Validate(&cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

// Marshal encodes cfg as canonical TOML.
func Marshal(cfg Config) ([]byte, error) {
	w := wireConfigOut{
		Version:   cfg.Version,
		Theme:     cfg.Theme,
		Shortcuts: make(map[string]wireShortcutOut, len(cfg.Shortcuts)),
	}
	if w.Version == 0 {
		w.Version = 1
	}
	for name, sc := range cfg.Shortcuts {
		var command any
		if sc.Command.Argv != nil {
			argv := make([]string, len(sc.Command.Argv))
			copy(argv, sc.Command.Argv)
			command = argv
		} else {
			command = sc.Command.Shell
		}
		w.Shortcuts[name] = wireShortcutOut{Command: command, Description: sc.Description}
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(w); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Save validates cfg and atomically replaces the file at path.
func Save(path string, cfg Config) error {
	if err := Validate(&cfg); err != nil {
		return err
	}
	b, err := Marshal(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// If the config is a symlink, update the target rather than replacing the
	// link with a regular file.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	f, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}
