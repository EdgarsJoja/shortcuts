package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	osexec "os/exec"
	"strings"

	"github.com/google/shlex"
	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/shim"
)

func newEditCmd(s *Streams) *cobra.Command {
	var command, argv, description, rename string
	var force, editConfig bool

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a shortcut",
		Args: func(cmd *cobra.Command, args []string) error {
			if editConfig {
				return cobra.NoArgs(cmd, args)
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sv, err := newService(s)
			if err != nil {
				return err
			}
			if editConfig {
				return editConfigFile(sv, s)
			}

			name := args[0]
			cfg, err := sv.load()
			if err != nil {
				return err
			}
			sc, ok := cfg.Shortcuts[name]
			if !ok {
				return fmt.Errorf("no shortcut named %q", name)
			}

			hasCommand := cmd.Flags().Changed("command")
			hasArgv := cmd.Flags().Changed("argv")
			hasDesc := cmd.Flags().Changed("description")
			if !hasCommand && !hasArgv && !hasDesc && rename == "" {
				return Usagef("no changes specified (use --command, --argv, --description, --rename, or --config)")
			}
			if hasCommand && hasArgv {
				return Usagef("--command and --argv are mutually exclusive")
			}

			if hasCommand {
				sc.Command = config.Command{Shell: command}
			}
			if hasArgv {
				parts, err := shlex.Split(argv)
				if err != nil {
					return fmt.Errorf("parse --argv: %w", err)
				}
				if parts == nil {
					parts = []string{}
				}
				sc.Command = config.Command{Argv: parts}
			}
			if hasDesc {
				sc.Description = description
			}

			newName := name
			if rename != "" {
				if err := config.ValidateName(rename); err != nil {
					return err
				}
				newName = rename
			}
			if err := config.ValidateShortcut(newName, sc); err != nil {
				return err
			}
			if newName != name {
				if _, exists := cfg.Shortcuts[newName]; exists && !force {
					return fmt.Errorf("shortcut %q already exists; use --force to overwrite", newName)
				}
			}

			mgr, err := sv.manager(force)
			if err != nil {
				return err
			}
			if newName != name && !force {
				if p, ok := shim.LookPathOutside(newName, sv.shimDir, s.env("PATH")); ok {
					return fmt.Errorf("refusing to shadow existing command %q; use --force to override", p)
				}
			}

			if err := sv.store.Update(func(c *config.Config) error {
				if newName != name {
					delete(c.Shortcuts, name)
				}
				c.Shortcuts[newName] = sc
				return nil
			}); err != nil {
				return err
			}

			if newName != name {
				err = mgr.Rename(name, newName)
			} else {
				err = mgr.Create(name)
			}
			if err != nil {
				return fmt.Errorf("shortcut updated, but shim sync failed: %w (run 'shortcuts doctor --fix')", err)
			}

			if newName != name {
				fmt.Fprintf(s.Out, "renamed %s to %s\n", name, newName)
			} else {
				fmt.Fprintf(s.Out, "updated %s\n", name)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&command, "command", "c", "", "replace with a shell command")
	cmd.Flags().StringVar(&argv, "argv", "", "replace with a direct-mode argv string")
	cmd.Flags().StringVarP(&description, "description", "d", "", "description")
	cmd.Flags().StringVar(&rename, "rename", "", "rename the shortcut and its shim")
	cmd.Flags().BoolVar(&editConfig, "config", false, "edit the raw config file in $VISUAL/$EDITOR")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite another shortcut or shadow an existing command")
	return cmd
}

func editConfigFile(sv *service, s *Streams) error {
	if _, err := os.Stat(sv.cfgPath); errors.Is(err, fs.ErrNotExist) {
		if err := sv.store.Save(config.Config{Version: 1, Shortcuts: map[string]config.Shortcut{}}); err != nil {
			return err
		}
	}

	editor := strings.TrimSpace(s.env("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(s.env("EDITOR"))
	}
	if editor == "" {
		return errors.New("no editor configured; set $VISUAL or $EDITOR")
	}
	parts, err := shlex.Split(editor)
	if err != nil || len(parts) == 0 {
		return fmt.Errorf("invalid editor command %q", editor)
	}

	execCmd := osexec.Command(parts[0], append(parts[1:], sv.cfgPath)...)
	execCmd.Stdin = s.In
	execCmd.Stdout = s.Out
	execCmd.Stderr = s.Err
	if err := execCmd.Run(); err != nil {
		return err
	}

	if _, err := sv.load(); err != nil {
		return fmt.Errorf("config is invalid after editing: %w", err)
	}
	fmt.Fprintln(s.Out, "config updated")
	return nil
}
