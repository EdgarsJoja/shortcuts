// Package doctor reports mismatches between the config, shim directory, and
// PATH.
package doctor

import (
	"fmt"
	"sort"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/shim"
)

// Issue is one diagnostic.
type Issue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Shortcut string `json:"shortcut,omitempty"`
	Message  string `json:"message"`
	Fixable  bool   `json:"fixable"`
}

// Check inspects config and shim state. It does not modify anything.
func Check(cfg config.Config, mgr *shim.Manager, pathEnv string) ([]Issue, error) {
	links, err := mgr.ManagedLinks()
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0)
	if !shim.PathContains(mgr.Dir, pathEnv) {
		issues = append(issues, Issue{
			Severity: "warning",
			Message:  fmt.Sprintf("shim directory %q is not in PATH", mgr.Dir),
		})
	}

	for _, name := range cfg.Names() {
		target, ok := links[name]
		if !ok {
			issues = append(issues, Issue{
				Severity: "error",
				Shortcut: name,
				Message:  "shim is missing",
				Fixable:  true,
			})
		} else if !shim.SamePath(target, mgr.Binary) {
			issues = append(issues, Issue{
				Severity: "warning",
				Shortcut: name,
				Message:  fmt.Sprintf("shim points to %s instead of %s", target, mgr.Binary),
				Fixable:  true,
			})
		}

		if msg := shim.CheckPrecedence(name, mgr.Dir, pathEnv); msg != "" {
			issues = append(issues, Issue{Severity: "warning", Shortcut: name, Message: msg})
		}
		if p, ok := shim.LookPathOutside(name, mgr.Dir, pathEnv); ok {
			issues = append(issues, Issue{
				Severity: "warning",
				Shortcut: name,
				Message:  fmt.Sprintf("also collides with existing command %q", p),
			})
		}
	}

	var stale []string
	for name := range links {
		if _, ok := cfg.Shortcuts[name]; !ok {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	for _, name := range stale {
		issues = append(issues, Issue{
			Severity: "warning",
			Shortcut: name,
			Message:  "stale shim is not present in the config",
			Fixable:  true,
		})
	}

	return issues, nil
}
