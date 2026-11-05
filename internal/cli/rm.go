package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
)

func newRmCmd(s *Streams) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a shortcut and its shim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			sv, err := newService(s)
			if err != nil {
				return err
			}
			cfg, err := sv.load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Shortcuts[name]; !ok {
				return fmt.Errorf("no shortcut named %q", name)
			}

			if !force {
				if !s.InTTY {
					return Usagef("refusing to remove %q in non-interactive mode without --force", name)
				}
				fmt.Fprintf(s.Out, "Remove shortcut %q? [y/N] ", name)
				line, _ := bufio.NewReader(s.In).ReadString('\n')
				if !strings.EqualFold(strings.TrimSpace(line), "y") {
					fmt.Fprintln(s.Out, "cancelled")
					return nil
				}
			}

			mgr, err := sv.manager(force)
			if err != nil {
				return err
			}
			if err := sv.store.Update(func(c *config.Config) error {
				delete(c.Shortcuts, name)
				return nil
			}); err != nil {
				return err
			}
			if err := mgr.Remove(name); err != nil {
				return fmt.Errorf("shortcut %q removed, but shim cleanup failed: %w (run 'shortcuts doctor --fix')", name, err)
			}
			fmt.Fprintf(s.Out, "removed %s\n", name)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "remove without confirmation and allow removing foreign files")
	return cmd
}
