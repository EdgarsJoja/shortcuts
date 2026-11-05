package runner

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/template"
)

// MaxDepth is the maximum shortcut nesting depth. It prevents a shortcut that
// invokes itself through PATH from forking without bound.
const MaxDepth = 16

// Plan is a fully expanded command ready to execute.
type Plan struct {
	Name    string
	Mode    config.Mode
	Shell   string
	Command string
	Argv    []string
	Dir     string
	Env     []string
}

// BuildOptions carries invocation context into BuildPlan.
type BuildOptions struct {
	Env   []string
	Cwd   string
	Depth int
	Shell string
}

// BuildPlan expands templating and resolves the execution mode.
func BuildPlan(name string, sc config.Shortcut, args []string, opts BuildOptions) (Plan, error) {
	if opts.Depth >= MaxDepth {
		return Plan{}, fmt.Errorf("shortcut recursion limit (%d) exceeded while running %q", MaxDepth, name)
	}

	shell := opts.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	if err := checkShell(shell); err != nil {
		return Plan{}, err
	}

	p := Plan{
		Name: name,
		Dir:  opts.Cwd,
		Env:  BuildEnv(opts.Env, name, opts.Depth+1),
	}

	switch sc.Command.Mode() {
	case config.ModeShell:
		parts, err := template.Parse(sc.Command.Shell, true)
		if err != nil {
			return Plan{}, fmt.Errorf("shortcut %q: %w", name, err)
		}
		cmd := template.ExpandShell(parts, args)
		if !template.HasPlaceholder(parts) {
			cmd = template.AppendShell(cmd, args)
		}
		p.Mode = config.ModeShell
		p.Shell = shell
		p.Command = cmd

	case config.ModeDirect:
		hasPlaceholder := false
		out := make([]string, 0, len(sc.Command.Argv)+len(args))
		for _, el := range sc.Command.Argv {
			parts, err := template.Parse(el, false)
			if err != nil {
				return Plan{}, fmt.Errorf("shortcut %q: %w", name, err)
			}
			if template.HasPlaceholder(parts) {
				hasPlaceholder = true
			}
			out = append(out, template.ExpandDirectElement(parts, args)...)
		}
		if !hasPlaceholder {
			out = append(out, args...)
		}
		if len(out) == 0 {
			return Plan{}, fmt.Errorf("shortcut %q expands to an empty command", name)
		}
		p.Mode = config.ModeDirect
		p.Argv = out

	default:
		return Plan{}, fmt.Errorf("shortcut %q has an unknown execution mode", name)
	}

	return p, nil
}

func checkShell(shell string) error {
	base := filepath.Base(shell)
	switch base {
	case "sh", "bash", "dash", "ksh", "ash", "zsh":
		return nil
	case "fish", "csh", "tcsh":
		return fmt.Errorf("shell %q has incompatible quoting semantics; use a POSIX shell such as /bin/sh or /bin/bash", shell)
	default:
		return fmt.Errorf("unsupported shell %q; use sh, bash, dash, ksh, ash, or zsh", shell)
	}
}

// BuildEnv returns env with previous shortcut bookkeeping removed and fresh
// SHORTCUTS_NAME / SHORTCUTS_DEPTH entries appended.
func BuildEnv(base []string, name string, depth int) []string {
	out := make([]string, 0, len(base)+2)
	for _, kv := range base {
		if strings.HasPrefix(kv, "SHORTCUTS_NAME=") || strings.HasPrefix(kv, "SHORTCUTS_DEPTH=") {
			continue
		}
		out = append(out, kv)
	}
	out = append(out,
		"SHORTCUTS_NAME="+name,
		"SHORTCUTS_DEPTH="+strconv.Itoa(depth),
	)
	return out
}
