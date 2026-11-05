package cli

import (
	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/buildinfo"
)

// NewRoot builds the command tree.
func NewRoot(s *Streams) *cobra.Command {
	root := &cobra.Command{
		Use:           "shortcuts",
		Short:         "Manage command shortcuts and their PATH shims",
		Long:          "shortcuts creates small commands that expand to longer ones.\nRun 'shortcuts' with no arguments to open the TUI.",
		Version:       buildinfo.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.InTTY && s.OutTTY {
				return runTUI(s)
			}
			return cmd.Help()
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return Usagef("%v", err)
	})

	root.AddCommand(
		newTUICmd(s),
		newListCmd(s),
		newShowCmd(s),
		newRunCmd(s),
		newAddCmd(s),
		newEditCmd(s),
		newRmCmd(s),
		newInitCmd(s),
		newDoctorCmd(s),
		newThemeCmd(s),
		newVersionCmd(s),
	)
	return root
}
