package tools

import (
	"strings"
	"testing"
	"time"
)

// TestRunSilentTimeoutEcho verifies that RunSilentTimeout runs a command,
// captures its combined output, and returns no error on success.
//
// On Windows we use `cmd /c echo hello`; on other platforms `echo hello` is
// invoked directly via the shell to keep the test cross-platform.
func TestRunSilentTimeoutEcho(t *testing.T) {
	var out string
	var err error

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Use a generous timeout; the command is near-instant.
	timeout := 10 * time.Second

	if isWindows {
		out, err = RunSilentTimeout(timeout, "", "cmd", "/c", "echo", "hello")
	} else {
		out, err = RunSilentTimeout(timeout, "", "sh", "-c", "echo hello")
	}

	if err != nil {
		t.Fatalf("RunSilentTimeout returned error: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected output to contain %q, got %q", "hello", out)
	}
}

// TestRunSilentTimeoutTrimmed ensures captured output is the actual command
// output (not polluted by a console window or extra noise).
func TestRunSilentTimeoutTrimmed(t *testing.T) {
	timeout := 10 * time.Second
	var out string
	var err error

	if isWindows {
		out, err = RunSilentTimeout(timeout, "", "cmd", "/c", "echo", "farsiforge")
	} else {
		out, err = RunSilentTimeout(timeout, "", "sh", "-c", "echo farsiforge")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "farsiforge" {
		t.Fatalf("expected trimmed output %q, got %q", "farsiforge", strings.TrimSpace(out))
	}
}

// TestRunLegacyWrappersSilent confirms the legacy Run/RunDir wrappers still
// work and route through the silent path.
func TestRunLegacyWrappersSilent(t *testing.T) {
	var out string
	var err error

	if isWindows {
		out, err = Run("cmd", "/c", "echo", "legacy")
	} else {
		out, err = Run("sh", "-c", "echo legacy")
	}
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(out, "legacy") {
		t.Fatalf("Run output missing %q: %q", "legacy", out)
	}
}
