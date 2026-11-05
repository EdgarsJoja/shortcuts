package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/tui"
)

func newTUICmd(s *Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive manager",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(s)
		},
	}
}

func runTUI(s *Streams) error {
	if !s.InTTY || !s.OutTTY {
		return errors.New("the TUI requires an interactive terminal")
	}
	sv, err := newService(s)
	if err != nil {
		return err
	}
	mgr, err := sv.manager(false)
	if err != nil {
		return err
	}
	return tui.Run(sv.store, mgr, s.In, s.Out)
}
