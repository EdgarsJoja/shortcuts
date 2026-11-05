package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/template"
)

type shortcutJSON struct {
	Name        string   `json:"name"`
	Mode        string   `json:"mode"`
	Description string   `json:"description,omitempty"`
	Command     string   `json:"command,omitempty"`
	Argv        []string `json:"argv,omitempty"`
}

func describeShortcut(name string, sc config.Shortcut) shortcutJSON {
	out := shortcutJSON{
		Name:        name,
		Mode:        string(sc.Command.Mode()),
		Description: sc.Description,
	}
	if sc.Command.Argv != nil {
		out.Argv = sc.Command.Argv
	} else {
		out.Command = sc.Command.Shell
	}
	return out
}

func commandDisplay(sc config.Shortcut) string {
	if sc.Command.Argv != nil {
		return template.QuoteArgv(sc.Command.Argv)
	}
	return sc.Command.Shell
}

func newListCmd(s *Streams) *cobra.Command {
	var long, asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured shortcuts",
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

			if asJSON {
				list := make([]shortcutJSON, 0, len(cfg.Shortcuts))
				for _, name := range cfg.Names() {
					list = append(list, describeShortcut(name, cfg.Shortcuts[name]))
				}
				return writeJSON(s, list)
			}

			tw := tabwriter.NewWriter(s.Out, 0, 4, 2, ' ', 0)
			if long {
				fmt.Fprintln(tw, "NAME\tMODE\tCOMMAND\tDESCRIPTION")
			} else {
				fmt.Fprintln(tw, "NAME\tMODE\tDESCRIPTION")
			}
			for _, name := range cfg.Names() {
				sc := cfg.Shortcuts[name]
				if long {
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", name, sc.Command.Mode(), commandDisplay(sc), sc.Description)
				} else {
					fmt.Fprintf(tw, "%s\t%s\t%s\n", name, sc.Command.Mode(), sc.Description)
				}
			}
			return tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&long, "long", false, "include the expanded command")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	return cmd
}

func writeJSON(s *Streams, v any) error {
	enc := json.NewEncoder(s.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
