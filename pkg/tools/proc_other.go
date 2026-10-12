//go:build !windows

package tools

import "os/exec"

// applySilentSysProcAttr is a no-op on non-Windows platforms where there is no
// concept of a hidden console window. This keeps the package cross-compilable.
func applySilentSysProcAttr(cmd *exec.Cmd) {
	_ = cmd
}

// isWindows is a compile-time constant used by tests to pick the right
// command for cross-platform sanity checks.
const isWindows = false
