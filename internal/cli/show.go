package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
)

type showJSON struct {
	shortcutJSON
	ShimPath   string `json:"shim_path"`
	ShimTarget string `json:"shim_target,omitempty"`
	ShimOK     bool   `json:"shim_ok"`
}

func newShowCmd(s *Streams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show one shortcut and its shim status",
		Args:  cobra.ExactArgs(1),
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

			if !asJSON {
				b, err := config.Marshal(config.Config{
					Version:   1,
					Shortcuts: map[string]config.Shortcut{name: sc},
				})
				if err != nil {
					return err
				}
				_, err = s.Out.Write(b)
				return err
			}

			shimPath := filepath.Join(sv.shimDir, name)
			out := showJSON{shortcutJSON: describeShortcut(name, sc), ShimPath: shimPath}
			if target, err := os.Readlink(shimPath); err == nil {
				if !filepath.IsAbs(target) {
					target = filepath.Join(sv.shimDir, target)
				}
				out.ShimTarget = filepath.Clean(target)
				out.ShimOK = filepath.Base(out.ShimTarget) == "shortcuts"
			}
			return writeJSON(s, out)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	return cmd
}
