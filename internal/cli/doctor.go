package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/doctor"
)

func newDoctorCmd(s *Streams) *cobra.Command {
	var asJSON, fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check config, shims, and PATH",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sv, err := newService(s)
			if err != nil {
				return err
			}
			cfg, err := sv.load()
			if err != nil {
				return err
			}
			mgr, err := sv.manager(fix)
			if err != nil {
				return err
			}

			issues, err := doctor.Check(cfg, mgr, s.env("PATH"))
			if err != nil {
				return err
			}

			if fix {
				if _, err := mgr.Sync(cfg.Names()); err != nil {
					return err
				}
				issues, err = doctor.Check(cfg, mgr, s.env("PATH"))
				if err != nil {
					return err
				}
			}

			if asJSON {
				if issues == nil {
					issues = []doctor.Issue{}
				}
				return writeJSON(s, issues)
			}
			if len(issues) == 0 {
				fmt.Fprintln(s.Out, "ok")
				return nil
			}
			for _, is := range issues {
				fixable := ""
				if is.Fixable {
					fixable = " (fixable)"
				}
				if is.Shortcut != "" {
					fmt.Fprintf(s.Out, "%-7s %-16s %s%s\n", is.Severity, is.Shortcut, is.Message, fixable)
				} else {
					fmt.Fprintf(s.Out, "%-7s %-16s %s%s\n", is.Severity, "-", is.Message, fixable)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	cmd.Flags().BoolVar(&fix, "fix", false, "recreate and prune managed shims")
	return cmd
}
