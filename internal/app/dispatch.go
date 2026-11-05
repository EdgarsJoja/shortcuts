// Package app owns process entry and argv[0]-based dispatch.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/term"

	"github.com/ejoja/shortcuts/internal/cli"
	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/runner"
	"github.com/ejoja/shortcuts/internal/xdg"
)

// Deps allows tests to replace process-level dependencies.
type Deps struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	InTTY   bool
	OutTTY  bool
	Getenv  func(string) string
	Environ func() []string
	Getwd   func() (string, error)
}

// DefaultDeps wires the real process.
func DefaultDeps() Deps {
	return Deps{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		InTTY:   term.IsTerminal(int(os.Stdin.Fd())),
		OutTTY:  term.IsTerminal(int(os.Stdout.Fd())),
		Getenv:  os.Getenv,
		Environ: os.Environ,
		Getwd:   os.Getwd,
	}
}

func (d Deps) streams() *cli.Streams {
	return &cli.Streams{
		In:      d.Stdin,
		Out:     d.Stdout,
		Err:     d.Stderr,
		InTTY:   d.InTTY,
		OutTTY:  d.OutTTY,
		Getenv:  d.Getenv,
		Environ: d.Environ,
		Getwd:   d.Getwd,
	}
}

// Main runs either the CLI or a shortcut and returns the process exit code.
func Main(argv []string, deps Deps) int {
	if len(argv) == 0 {
		argv = []string{"shortcuts"}
	}

	name, isCLI := DispatchName(filepath.Base(argv[0]), deps)
	if !isCLI {
		return runShim(name, argv[1:], deps)
	}

	root := cli.NewRoot(deps.streams())
	root.SetArgs(argv[1:])
	if err := root.Execute(); err != nil {
		var ee *cli.ExitError
		if errors.As(err, &ee) {
			if ee.Msg != "" {
				fmt.Fprintln(deps.Stderr, "shortcuts:", ee.Msg)
			}
			return ee.Code
		}
		fmt.Fprintln(deps.Stderr, "shortcuts:", err)
		return 1
	}
	return 0
}

// DispatchName decides whether argv[0] names the CLI itself or a shortcut.
func DispatchName(base string, deps Deps) (name string, isCLI bool) {
	if base == "" || base == "shortcuts" || base == "shortcuts.exe" {
		return "", true
	}
	if deps.Getenv != nil && deps.Getenv("SHORTCUTS_CLI") != "" {
		return "", true
	}
	return base, false
}

func runShim(name string, args []string, deps Deps) int {
	cfgPath, err := xdg.ConfigPath()
	if err != nil {
		fmt.Fprintln(deps.Stderr, "shortcuts:", err)
		return 1
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintln(deps.Stderr, "shortcuts:", err)
		return 1
	}
	sc, ok := cfg.Shortcuts[name]
	if !ok {
		fmt.Fprintf(deps.Stderr, "shortcuts: no shortcut named %q\nRun 'shortcuts list' or 'shortcuts' to manage shortcuts.\n", name)
		return 127
	}

	cwd, err := deps.Getwd()
	if err != nil {
		fmt.Fprintln(deps.Stderr, "shortcuts:", err)
		return 1
	}
	var depth int
	if deps.Getenv != nil {
		depth, _ = strconv.Atoi(deps.Getenv("SHORTCUTS_DEPTH"))
	}
	env := os.Environ()
	if deps.Environ != nil {
		env = deps.Environ()
	}

	plan, err := runner.BuildPlan(name, sc, args, runner.BuildOptions{
		Env:   env,
		Cwd:   cwd,
		Depth: depth,
	})
	if err != nil {
		fmt.Fprintln(deps.Stderr, "shortcuts:", err)
		return 1
	}
	code, err := runner.Run(context.Background(), plan, deps.Stdin, deps.Stdout, deps.Stderr)
	if err != nil {
		fmt.Fprintln(deps.Stderr, "shortcuts:", err)
		return 1
	}
	return code
}
