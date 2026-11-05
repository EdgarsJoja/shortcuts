// Package shim manages the PATH directory of shortcut symlinks.
package shim

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const binaryName = "shortcuts"

// Manager owns a shim directory whose entries symlink to the shortcuts binary.
type Manager struct {
	Dir    string
	Binary string
	Force  bool
}

// Report summarizes a Sync operation.
type Report struct {
	Synced  []string
	Removed []string
	Errors  []error
}

// Create makes name point at the current binary. Managed links are repointed;
// foreign files and symlinks are refused unless Force is set.
func (m *Manager) Create(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if m.Dir == "" {
		return errors.New("shim directory is not configured")
	}
	if m.Binary == "" {
		return errors.New("shortcut binary path is unknown; set SHORTCUTS_BINARY")
	}
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		return err
	}

	target := filepath.Join(m.Dir, name)
	info, err := os.Lstat(target)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return m.replace(target)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		cur, err := os.Readlink(target)
		if err != nil {
			return err
		}
		cur = absoluteLink(cur, m.Dir)
		if SamePath(cur, m.Binary) {
			return nil
		}
		if !isManaged(cur) && !m.Force {
			return fmt.Errorf("refusing to replace %s: it points to %s (use --force)", target, cur)
		}
		return m.replace(target)
	}

	if !m.Force {
		return fmt.Errorf("refusing to replace existing file %s (use --force)", target)
	}
	return m.replace(target)
}

// Remove deletes a managed shim. Foreign links and files require Force.
func (m *Manager) Remove(name string) error {
	if m.Dir == "" {
		return errors.New("shim directory is not configured")
	}
	target := filepath.Join(m.Dir, name)
	info, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		cur, err := os.Readlink(target)
		if err != nil {
			return err
		}
		if isManaged(absoluteLink(cur, m.Dir)) || m.Force {
			return os.Remove(target)
		}
		return fmt.Errorf("refusing to remove %s: it points to %s (use --force)", target, cur)
	}
	if m.Force {
		return os.Remove(target)
	}
	return fmt.Errorf("refusing to remove non-symlink %s (use --force)", target)
}

// Rename creates the new link before removing the old one, so a collision
// leaves the original shortcut intact.
func (m *Manager) Rename(oldName, newName string) error {
	if oldName == newName {
		return m.Create(newName)
	}
	if err := m.Create(newName); err != nil {
		return err
	}
	return m.Remove(oldName)
}

// Sync converges the directory on desired: it creates missing links, repoints
// relocated managed links, and prunes managed links no longer configured.
func (m *Manager) Sync(desired []string) (Report, error) {
	if m.Dir == "" {
		return Report{}, errors.New("shim directory is not configured")
	}
	var rep Report
	want := make(map[string]bool, len(desired))
	sorted := append([]string(nil), desired...)
	sort.Strings(sorted)

	for _, name := range sorted {
		want[name] = true
		if err := m.Create(name); err != nil {
			rep.Errors = append(rep.Errors, fmt.Errorf("%s: %w", name, err))
			continue
		}
		rep.Synced = append(rep.Synced, name)
	}

	links, err := m.ManagedLinks()
	if err != nil {
		rep.Errors = append(rep.Errors, err)
	}

	var stale []string
	for name := range links {
		if !want[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	for _, name := range stale {
		if err := os.Remove(filepath.Join(m.Dir, name)); err != nil {
			rep.Errors = append(rep.Errors, err)
		} else {
			rep.Removed = append(rep.Removed, name)
		}
	}

	if len(rep.Errors) > 0 {
		return rep, errors.Join(rep.Errors...)
	}
	return rep, nil
}

// ManagedLinks returns name -> target for symlinks in Dir that point at a
// binary named "shortcuts". Other files are never considered ours.
func (m *Manager) ManagedLinks() (map[string]string, error) {
	out := map[string]string{}
	entries, err := os.ReadDir(m.Dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.Type()&os.ModeSymlink == 0 {
			continue
		}
		path := filepath.Join(m.Dir, e.Name())
		target, err := os.Readlink(path)
		if err != nil {
			continue
		}
		target = absoluteLink(target, m.Dir)
		if isManaged(target) {
			out[e.Name()] = target
		}
	}
	return out, nil
}

func (m *Manager) replace(target string) error {
	tmp, err := os.CreateTemp(m.Dir, ".shim-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	tmp.Close()
	os.Remove(tmpName)

	if err := os.Symlink(m.Binary, tmpName); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func isManaged(target string) bool {
	return filepath.Base(target) == binaryName
}

func absoluteLink(link, dir string) string {
	if filepath.IsAbs(link) {
		return filepath.Clean(link)
	}
	return filepath.Clean(filepath.Join(dir, link))
}

func validateName(name string) error {
	// Kept behind a small wrapper so config and shim rules cannot drift.
	return validateShortcutName(name)
}
