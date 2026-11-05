package cli

import (
	"fmt"

	"github.com/google/shlex"
	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/shim"
	"github.com/ejoja/shortcuts/internal/template"
)

func newAddCmd(s *Streams) *cobra.Command {
	var command, argv, description string
	var force bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a shortcut and its PATH shim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.ValidateName(name); err != nil {
				return err
			}

			hasCommand := cmd.Flags().Changed("command")
			hasArgv := cmd.Flags().Changed("argv")
			if hasCommand == hasArgv {
				return Usagef("exactly one of --command (shell) or --argv (direct) is required")
			}

			sc := config.Shortcut{Description: description}
			if hasCommand {
				sc.Command = config.Command{Shell: command}
			} else {
				parts, err := shlex.Split(argv)
				if err != nil {
					return fmt.Errorf("parse --argv: %w", err)
				}
				if parts == nil {
					parts = []string{}
				}
				sc.Command = config.Command{Argv: parts}
			}
			if err := config.ValidateShortcut(name, sc); err != nil {
				return err
			}

			sv, err := newService(s)
			if err != nil {
				return err
			}
			mgr, err := sv.manager(force)
			if err != nil {
				return err
			}
			cfg, err := sv.load()
			if err != nil {
				return err
			}
			if _, exists := cfg.Shortcuts[name]; exists && !force {
				return fmt.Errorf("shortcut %q already exists; use 'shortcuts edit %s' or --force", name, name)
			}
			if !force {
				if p, ok := shim.LookPathOutside(name, sv.shimDir, s.env("PATH")); ok {
					return fmt.Errorf("refusing to shadow existing command %q; use --force to override", p)
				}
			}

			if err := sv.store.Update(func(c *config.Config) error {
				c.Shortcuts[name] = sc
				return nil
			}); err != nil {
				return err
			}

			if err := mgr.Create(name); err != nil {
				return fmt.Errorf("shortcut %q saved, but shim creation failed: %w (run 'shortcuts doctor --fix')", name, err)
			}
			fmt.Fprintf(s.Out, "added %s (%s)\n", name, sc.Command.Mode())

			if !shim.PathContains(sv.shimDir, s.env("PATH")) {
				fmt.Fprintf(s.Err, "\nwarning: %s is not in PATH, so %q cannot be invoked from your shell yet.\n", sv.shimDir, name)
				fmt.Fprintf(s.Err, "Add this line to your shell rc, then open a new shell:\n\n  export PATH=%s:\"$PATH\"\n\n", template.ShellQuote(sv.shimDir))
				fmt.Fprintf(s.Err, "Or run: shortcuts init --write --shell zsh\n")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&command, "command", "c", "", "shell command (shell mode)")
	cmd.Flags().StringVar(&argv, "argv", "", "argv string, split shell-style (direct mode)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "description")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing shortcut or shadow an existing command")
	return cmd
}
