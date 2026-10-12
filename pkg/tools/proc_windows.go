//go:build windows

package tools

import (
	"os/exec"
	"syscall"
)

// createNoWindow is the Windows process creation flag that prevents a console
// window from being shown for the new process.
const createNoWindow = 0x08000000

// isWindows is a compile-time constant used by tests to pick the right
// command for cross-platform sanity checks.
const isWindows = true

// applySilentSysProcAttr configures the command so that no visible window is
// shown when it runs. On Windows it sets both HideWindow and the
// CREATE_NO_WINDOW creation flag for maximum reliability across tool launchers.
func applySilentSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
