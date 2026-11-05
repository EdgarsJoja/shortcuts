package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/buildinfo"
)

func newVersionCmd(s *Streams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				return writeJSON(s, map[string]string{
					"version": buildinfo.Version,
					"commit":  buildinfo.Commit,
				})
			}
			fmt.Fprintln(s.Out, buildinfo.Version)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	return cmd
}
