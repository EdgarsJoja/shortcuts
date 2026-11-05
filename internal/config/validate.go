package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ejoja/shortcuts/internal/template"
)

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

var reservedNames = map[string]struct{}{
	"shortcuts": {},
	"list":      {},
	"show":      {},
	"run":       {},
	"add":       {},
	"edit":      {},
	"rm":        {},
	"remove":    {},
	"init":      {},
	"doctor":    {},
	"theme":     {},
	"version":   {},
	"help":      {},
	"tui":       {},
	"completion": {},
	"__complete": {},
	"__dispatch": {},
}

// IsReserved reports whether name collides with a CLI command.
func IsReserved(name string) bool {
	_, ok := reservedNames[strings.ToLower(name)]
	return ok
}

// ValidateName checks a shortcut/shim name.
func ValidateName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid shortcut name %q: use letters, digits, '.', '_' or '-', starting with a letter or digit (max 64 chars)", name)
	}
	if IsReserved(name) {
		return fmt.Errorf("shortcut name %q is reserved by the CLI", name)
	}
	return nil
}

// ValidateShortcut validates one entry. It does not check cross-entry
// constraints such as case-insensitive duplicates.
func ValidateShortcut(name string, sc Shortcut) error {
	var errs []error

	switch sc.Command.Mode() {
	case ModeShell:
		if strings.TrimSpace(sc.Command.Shell) == "" {
			errs = append(errs, errors.New("command is empty"))
		} else if _, err := template.Parse(sc.Command.Shell, true); err != nil {
			errs = append(errs, fmt.Errorf("command: %w", err))
		}
	case ModeDirect:
		if len(sc.Command.Argv) == 0 {
			errs = append(errs, errors.New("direct-mode command argv is empty"))
		}
		for i, el := range sc.Command.Argv {
			if _, err := template.Parse(el, false); err != nil {
				errs = append(errs, fmt.Errorf("command element %d: %w", i+1, err))
			}
		}
	}

	if n := len([]rune(sc.Description)); n > 500 {
		errs = append(errs, fmt.Errorf("description is %d characters; maximum is 500", n))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shortcut %q: %w", name, errors.Join(errs...))
	}
	return nil
}

// Validate checks all entries, aggregating every problem it finds.
func Validate(cfg *Config) error {
	var errs []error
	seen := make(map[string]string, len(cfg.Shortcuts))

	for _, name := range cfg.Names() {
		if err := ValidateName(name); err != nil {
			errs = append(errs, err)
			continue
		}
		lower := strings.ToLower(name)
		if prev, ok := seen[lower]; ok {
			errs = append(errs, fmt.Errorf("shortcut names %q and %q differ only by case", prev, name))
		} else {
			seen[lower] = name
		}
		if err := ValidateShortcut(name, cfg.Shortcuts[name]); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
