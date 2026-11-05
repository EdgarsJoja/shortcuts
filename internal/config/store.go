package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/gofrs/flock"
)

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

// Store serializes read-modify-write cycles against one config file.
type Store struct {
	path string
	mu   sync.Mutex
}

// Open returns a store for path.
func Open(path string) *Store {
	return &Store{path: path}
}

// Path returns the config file path.
func (s *Store) Path() string { return s.path }

// Load reads the current config.
func (s *Store) Load() (Config, error) {
	return Load(s.path)
}

// Save validates and atomically writes cfg.
func (s *Store) Save(cfg Config) error {
	return Save(s.path, cfg)
}

// Update runs fn against the latest config under an advisory file lock, then
// validates and saves. Callers must not retain the pointer after fn returns.
func (s *Store) Update(fn func(*Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ensureDir(filepath.Dir(s.path)); err != nil {
		return err
	}
	lock := flock.New(s.path + ".lock")
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	cfg, err := Load(s.path)
	if err != nil {
		return err
	}
	if err := fn(&cfg); err != nil {
		return err
	}
	return Save(s.path, cfg)
}
