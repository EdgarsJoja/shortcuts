// Package cli implements the shortcuts command-line interface.
package cli

import (
	"fmt"
	"io"
	"os"
)

// Streams bundles the process dependencies commands need. Tests inject
// buffers and a temporary environment instead of touching process globals.
type Streams struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	InTTY   bool
	OutTTY  bool
	Getenv  func(string) string
	Environ func() []string
	Getwd   func() (string, error)
}

func (s *Streams) env(name string) string {
	if s.Getenv == nil {
		return os.Getenv(name)
	}
	return s.Getenv(name)
}

func (s *Streams) environ() []string {
	if s.Environ == nil {
		return os.Environ()
	}
	return s.Environ()
}

func (s *Streams) getwd() (string, error) {
	if s.Getwd == nil {
		return os.Getwd()
	}
	return s.Getwd()
}

// ExitError carries a specific process exit code. An empty Msg means the exit
// code came from a child process and should not produce an extra error line.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

// Usagef returns a usage error (exit code 2).
func Usagef(format string, args ...any) error {
	return &ExitError{Code: 2, Msg: fmt.Sprintf(format, args...)}
}
