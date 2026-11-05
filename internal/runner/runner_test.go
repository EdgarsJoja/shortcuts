package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ejoja/shortcuts/internal/config"
)

func TestBuildPlanShell(t *testing.T) {
	sc := config.Shortcut{Command: config.Command{Shell: "echo {{args}}"}}
	p, err := BuildPlan("greet", sc, []string{"a b", "c"}, BuildOptions{
		Env: []string{"PATH=/bin"},
		Cwd: "/tmp",
	})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if p.Mode != config.ModeShell || p.Shell != "/bin/sh" {
		t.Fatalf("unexpected plan: %#v", p)
	}
	if want := "echo 'a b' 'c'"; p.Command != want {
		t.Fatalf("command = %q, want %q", p.Command, want)
	}
	if p.Dir != "/tmp" {
		t.Fatalf("dir = %q", p.Dir)
	}
}

func TestBuildPlanAppendsArgsWithoutPlaceholder(t *testing.T) {
	sc := config.Shortcut{Command: config.Command{Shell: "echo hi"}}
	p, err := BuildPlan("x", sc, []string{"a b"}, BuildOptions{})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if want := "echo hi 'a b'"; p.Command != want {
		t.Fatalf("command = %q, want %q", p.Command, want)
	}
}

func TestBuildPlanDirect(t *testing.T) {
	sc := config.Shortcut{Command: config.Command{Argv: []string{"git", "checkout", "{{args}}"}}}
	p, err := BuildPlan("co", sc, []string{"main", "feature/x"}, BuildOptions{})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if p.Mode != config.ModeDirect {
		t.Fatalf("mode = %q", p.Mode)
	}
	if len(p.Argv) != 4 || p.Argv[2] != "main" || p.Argv[3] != "feature/x" {
		t.Fatalf("argv = %#v", p.Argv)
	}
}

func TestBuildPlanDepthGuard(t *testing.T) {
	sc := config.Shortcut{Command: config.Command{Shell: "true"}}
	if _, err := BuildPlan("x", sc, nil, BuildOptions{Depth: MaxDepth}); err == nil {
		t.Fatal("expected recursion depth error")
	}
}

func TestBuildEnvReplacesDepth(t *testing.T) {
	env := BuildEnv([]string{"A=1", "SHORTCUTS_DEPTH=3", "SHORTCUTS_NAME=old"}, "new", 4)
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "SHORTCUTS_DEPTH=3") || strings.Contains(joined, "SHORTCUTS_NAME=old") {
		t.Fatalf("stale entries kept: %q", env)
	}
	if !strings.Contains(joined, "SHORTCUTS_NAME=new") || !strings.Contains(joined, "SHORTCUTS_DEPTH=4") {
		t.Fatalf("new entries missing: %q", env)
	}
}

func TestRunExitCode(t *testing.T) {
	p := Plan{
		Mode:    config.ModeShell,
		Shell:   "/bin/sh",
		Command: "exit 7",
		Env:     []string{"PATH=/bin:/usr/bin"},
	}
	var out, errOut bytes.Buffer
	code, err := Run(context.Background(), p, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != 7 {
		t.Fatalf("code = %d, want 7", code)
	}
}

func TestRunSignalExitCode(t *testing.T) {
	p := Plan{
		Mode:    config.ModeShell,
		Shell:   "/bin/sh",
		Command: "kill -TERM $$",
		Env:     []string{"PATH=/bin:/usr/bin"},
	}
	var out, errOut bytes.Buffer
	code, err := Run(context.Background(), p, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != 128+15 {
		t.Fatalf("code = %d, want %d", code, 128+15)
	}
}

func TestRunStdinPassthrough(t *testing.T) {
	p := Plan{
		Mode:    config.ModeShell,
		Shell:   "/bin/sh",
		Command: "cat",
		Env:     []string{"PATH=/bin:/usr/bin"},
	}
	var out, errOut bytes.Buffer
	code, err := Run(context.Background(), p, strings.NewReader("hello\n"), &out, &errOut)
	if err != nil || code != 0 {
		t.Fatalf("Run: code=%d err=%v", code, err)
	}
	if out.String() != "hello\n" {
		t.Fatalf("stdout = %q", out.String())
	}
}
