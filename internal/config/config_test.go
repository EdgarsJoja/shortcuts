package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := Config{
		Version: 1,
		Theme:   "dracula",
		Shortcuts: map[string]Shortcut{
			"deploy": {
				Command:     Command{Shell: "kubectl apply -f {{args}}"},
				Description: "Apply a manifest",
			},
			"g.st": {
				Command:     Command{Argv: []string{"git", "status", "--short"}},
				Description: "Short status",
			},
		},
	}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("config mode = %o, want 600", perm)
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version != 1 || len(cfg.Shortcuts) != 0 {
		t.Fatalf("unexpected empty config: %#v", cfg)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("version = 1\nnope = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown-key error, got %v", err)
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"deploy", "g.st", "a_b-c", "X1"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v", name, err)
		}
	}
	invalid := []string{"", "-x", "has space", ".", "..", "run", "shortcuts", strings.Repeat("a", 65)}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) succeeded, want error", name)
		}
	}
}

func TestValidateShortcut(t *testing.T) {
	bad := []Shortcut{
		{Command: Command{Shell: ""}},
		{Command: Command{Shell: `echo "{{1}}"`}},
		{Command: Command{Argv: []string{}}},
		{Command: Command{Argv: []string{"x{{args}}y"}}},
		{Command: Command{Shell: "echo", Description: strings.Repeat("x", 501)}},
	}
	for i, sc := range bad {
		if err := ValidateShortcut("test", sc); err == nil {
			t.Errorf("case %d: ValidateShortcut succeeded, want error", i)
		}
	}
}

func TestValidateCaseInsensitiveDuplicates(t *testing.T) {
	cfg := &Config{Version: 1, Shortcuts: map[string]Shortcut{
		"Deploy": {Command: Command{Shell: "true"}},
		"deploy": {Command: Command{Shell: "true"}},
	}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected duplicate-name error")
	}
}

func TestStoreUpdateConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	store := Open(path)

	var wg sync.WaitGroup
	names := []string{"one", "two", "three", "four", "five"}
	for _, name := range names {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := store.Update(func(c *Config) error {
				c.Shortcuts[name] = Shortcut{Command: Command{Shell: "true"}}
				return nil
			})
			if err != nil {
				t.Errorf("Update %s: %v", name, err)
			}
		}()
	}
	wg.Wait()

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if _, ok := cfg.Shortcuts[name]; !ok {
			t.Errorf("shortcut %q lost during concurrent updates", name)
		}
	}
}
