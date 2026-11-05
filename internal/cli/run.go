package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/runner"
	"github.com/ejoja/shortcuts/internal/template"
)

type planJSON struct {
	Name    string   `json:"name"`
	Mode    string   `json:"mode"`
	Command string   `json:"command,omitempty"`
	Argv    []string `json:"argv,omitempty"`
	Cwd     string   `json:"cwd,omitempty"`
}

func newRunCmd(s *Streams) *cobra.Command {
	var dryRun, asJSON bool
	cmd := &cobra.Command{
		Use:   "run <name> [args...]",
		Short: "Run a shortcut",
		Long:  "Run a shortcut by name. Flags after the shortcut name are passed to it verbatim.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sv, err := newService(s)
			if err != nil {
				return err
			}
			cfg, err := sv.load()
			if err != nil {
				return err
			}
			name := args[0]
			sc, ok := cfg.Shortcuts[name]
			if !ok {
				return fmt.Errorf("no shortcut named %q", name)
			}

			cwd, err := s.getwd()
			if err != nil {
				return err
			}
			depth, _ := strconv.Atoi(s.env("SHORTCUTS_DEPTH"))
			plan, err := runner.BuildPlan(name, sc, args[1:], runner.BuildOptions{
				Env:   s.environ(),
				Cwd:   cwd,
				Depth: depth,
			})
			if err != nil {
				return err
			}

			if dryRun {
				if asJSON {
					out := planJSON{Name: plan.Name, Mode: string(plan.Mode), Cwd: plan.Dir}
					if plan.Mode == "direct" {
						out.Argv = plan.Argv
					} else {
						out.Command = plan.Command
					}
					return writeJSON(s, out)
				}
				fmt.Fprintf(s.Out, "name: %s\nmode: %s\n", plan.Name, plan.Mode)
				if plan.Mode == "direct" {
					fmt.Fprintf(s.Out, "argv: %s\n", template.QuoteArgv(plan.Argv))
				} else {
					fmt.Fprintf(s.Out, "command: %s\n", plan.Command)
				}
				if plan.Dir != "" {
					fmt.Fprintf(s.Out, "cwd: %s\n", plan.Dir)
				}
				return nil
			}

			code, err := runner.Run(context.Background(), plan, s.In, s.Out, s.Err)
			if err != nil {
				return err
			}
			if code != 0 {
				return &ExitError{Code: code}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the expanded command without running it")
	cmd.Flags().BoolVar(&asJSON, "json", false, "with --dry-run, emit JSON")
	cmd.Flags().SetInterspersed(false)
	return cmd
}
