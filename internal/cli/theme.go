package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/theme"
)

type themeListJSON struct {
	Current string   `json:"current"`
	Themes  []string `json:"themes"`
}

func newThemeCmd(s *Streams) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "theme [name]",
		Short: "List or select a TUI theme",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sv, err := newService(s)
			if err != nil {
				return err
			}
			cfg, err := sv.load()
			if err != nil {
				return err
			}
			current := canonicalTheme(cfg.Theme)

			if len(args) == 0 {
				if asJSON {
					return writeJSON(s, themeListJSON{Current: current, Themes: theme.Names()})
				}
				fmt.Fprintf(s.Out, "current: %s\n\n", current)
				for _, name := range theme.Names() {
					marker := "  "
					if name == current {
						marker = "* "
					}
					fmt.Fprintf(s.Out, "%s%s\n", marker, name)
				}
				return nil
			}

			p, ok := theme.Get(args[0])
			if !ok {
				return fmt.Errorf("unknown theme %q; available: %s", args[0], strings.Join(theme.Names(), ", "))
			}
			if err := sv.store.Update(func(c *config.Config) error {
				c.Theme = p.Name
				return nil
			}); err != nil {
				return err
			}
			fmt.Fprintf(s.Out, "theme set to %s\n", p.Name)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	return cmd
}

func canonicalTheme(name string) string {
	if p, ok := theme.Get(name); ok {
		return p.Name
	}
	return theme.Default().Name
}
