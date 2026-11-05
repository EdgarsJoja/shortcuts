package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/template"
)

const (
	markerStart = "# >>> shortcuts >>>"
	markerEnd   = "# <<< shortcuts <<<"
)

func newInitCmd(s *Streams) *cobra.Command {
	var printOnly, write, yes, repair bool
	var rcFile, shellName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create config directories and set up PATH",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sv, err := newService(s)
			if err != nil {
				return err
			}
			if _, err := os.Stat(sv.cfgPath); errors.Is(err, fs.ErrNotExist) {
				if err := sv.store.Save(config.Config{Version: 1, Shortcuts: map[string]config.Shortcut{}}); err != nil {
					return err
				}
			}
			if err := os.MkdirAll(sv.shimDir, 0o700); err != nil {
				return err
			}

			if shellName == "" {
				shellName = filepath.Base(s.env("SHELL"))
			}
			if shellName == "" {
				shellName = "bash"
			}
			if rcFile == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				switch shellName {
				case "zsh":
					rcFile = filepath.Join(home, ".zshrc")
				case "bash":
					rcFile = filepath.Join(home, ".bashrc")
				case "fish":
					return errors.New("fish is not supported in v1; add the printed PATH line to your fish config manually")
				default:
					rcFile = filepath.Join(home, ".profile")
				}
			}

			line := fmt.Sprintf("export PATH=%s:\"$PATH\"", template.ShellQuote(sv.shimDir))

			fmt.Fprintf(s.Out, "config:  %s\nshims:   %s\n\n", sv.cfgPath, sv.shimDir)
			fmt.Fprintf(s.Out, "Add this line to %s:\n\n  %s\n\n", rcFile, line)
			fmt.Fprintf(s.Out, "Then open a new shell or run: source %s\n", rcFile)
			fmt.Fprintf(s.Out, "Verify with: shortcuts doctor\n")

			if write || yes {
				if err := writeRCPath(rcFile, line); err != nil {
					return err
				}
				fmt.Fprintf(s.Out, "\nupdated %s\n", rcFile)
			} else if !printOnly {
				fmt.Fprint(s.Out, "\nNo files were modified. Use --write to add the line automatically.\n")
			}

			if repair {
				mgr, err := sv.manager(false)
				if err != nil {
					return err
				}
				cfg, err := sv.load()
				if err != nil {
					return err
				}
				rep, err := mgr.Sync(cfg.Names())
				if err != nil {
					return err
				}
				fmt.Fprintf(s.Out, "synced %d shims\n", len(rep.Synced))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&printOnly, "print", false, "only print instructions (default)")
	cmd.Flags().BoolVar(&write, "write", false, "append the PATH line to the shell rc file")
	cmd.Flags().BoolVar(&yes, "yes", false, "append without prompting (implies --write)")
	cmd.Flags().BoolVar(&repair, "repair", false, "recreate shortcut shims")
	cmd.Flags().StringVar(&rcFile, "file", "", "rc file to modify")
	cmd.Flags().StringVar(&shellName, "shell", "", "shell name: bash or zsh")
	return cmd
}

// writeRCPath appends a managed PATH block if the exact line is not already
// present. The previous file is backed up before modification.
func writeRCPath(path, line string) error {
	b, err := os.ReadFile(path)
	exists := true
	if errors.Is(err, fs.ErrNotExist) {
		exists = false
	} else if err != nil {
		return err
	}
	if strings.Contains(string(b), line) {
		return nil
	}

	mode := fs.FileMode(0o644)
	if exists {
		if info, err := os.Stat(path); err == nil {
			mode = info.Mode().Perm()
		}
		if err := os.WriteFile(path+".bak", b, mode); err != nil {
			return err
		}
	}

	block := "\n" + markerStart + "\n" + line + "\n" + markerEnd + "\n"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(block)
	return err
}
