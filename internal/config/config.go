// Package config defines the on-disk shortcut configuration and its safe
// persistence.
package config

import (
	"sort"
	"strings"
)

// Mode is the execution mode of a shortcut.
type Mode string

const (
	// ModeShell runs the command through a POSIX shell.
	ModeShell Mode = "shell"
	// ModeDirect executes argv without a shell.
	ModeDirect Mode = "direct"
)

// Command represents a shell command or a direct argv vector. A TOML string
// decodes as shell mode; a TOML array decodes as direct mode.
type Command struct {
	Shell string
	Argv  []string
}

// Mode reports the execution mode implied by the representation.
func (c Command) Mode() Mode {
	if c.Argv != nil {
		return ModeDirect
	}
	return ModeShell
}

// Shortcut is one user-defined command.
type Shortcut struct {
	Command     Command `toml:"command"`
	Description string  `toml:"description,omitempty"`
}

// Config is the root configuration document.
type Config struct {
	Version   int                 `toml:"version"`
	Theme     string              `toml:"theme,omitempty"`
	Shortcuts map[string]Shortcut `toml:"shortcuts"`
}

func empty() Config {
	return Config{Version: 1, Shortcuts: map[string]Shortcut{}}
}

// Clone returns a deep copy of c.
func (c Config) Clone() Config {
	out := Config{Version: c.Version, Theme: c.Theme, Shortcuts: make(map[string]Shortcut, len(c.Shortcuts))}
	for k, v := range c.Shortcuts {
		if v.Command.Argv != nil {
			argv := make([]string, len(v.Command.Argv))
			copy(argv, v.Command.Argv)
			v.Command.Argv = argv
		}
		out.Shortcuts[k] = v
	}
	return out
}

// Names returns shortcut names in case-insensitive sorted order.
func (c Config) Names() []string {
	names := make([]string, 0, len(c.Shortcuts))
	for name := range c.Shortcuts {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	return names
}
