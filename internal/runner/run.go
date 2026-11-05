package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"syscall"

	"github.com/ejoja/shortcuts/internal/config"
)

// Run executes plan with stdio passthrough and returns the process exit code.
// A non-nil error means the command could not be started; a child that exits
// non-zero is reported through the returned code.
func Run(ctx context.Context, p Plan, in io.Reader, out, errOut io.Writer) (int, error) {
	var cmd *exec.Cmd
	switch p.Mode {
	case config.ModeDirect:
		if len(p.Argv) == 0 {
			return 1, fmt.Errorf("empty argv")
		}
		cmd = exec.CommandContext(ctx, p.Argv[0], p.Argv[1:]...)
	default:
		cmd = exec.CommandContext(ctx, p.Shell, "-c", p.Command)
	}

	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Env = p.Env
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}

	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if st, ok := ee.Sys().(syscall.WaitStatus); ok {
				if st.Signaled() {
					return 128 + int(st.Signal()), nil
				}
				if code := st.ExitStatus(); code >= 0 {
					return code, nil
				}
			}
			if code := ee.ExitCode(); code >= 0 {
				return code, nil
			}
		}
		return 1, err
	}
	return 0, nil
}
