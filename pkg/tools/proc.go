package tools

import (
	"context"
	"os/exec"
	"time"
)

// RunSilent runs a command with no visible window, capturing combined stdout
// and stderr output. On Windows it sets BOTH SysProcAttr{HideWindow: true}
// AND CreationFlags: CREATE_NO_WINDOW (0x08000000) so that no console window
// flashes when launching external tools. When dir is non-empty it is used as
// the command's working directory.
//
// All external tool execution in FarsiForge should route through this
// function (or its timeout/legacy wrappers) to guarantee silent execution.
func RunSilent(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	applySilentSysProcAttr(cmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunSilentTimeout is RunSilent with a timeout. It creates an internal
// context (with the given timeout) that is cancelled when the call returns.
func RunSilentTimeout(timeout time.Duration, dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return RunSilent(ctx, dir, name, args...)
}

// Run executes a command silently and returns its combined output.
// It is a legacy thin wrapper kept for backward compatibility; new code
// should prefer RunSilent / RunSilentTimeout.
func Run(name string, args ...string) (string, error) {
	return RunSilent(context.Background(), "", name, args...)
}

// RunDir executes a command silently with a specific working directory.
// It is a legacy thin wrapper kept for backward compatibility. Unlike the
// previous implementation it is now fully silent (no window flash).
func RunDir(dir, name string, args ...string) (string, error) {
	return RunSilent(context.Background(), dir, name, args...)
}
