package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestRunSilentTimeoutEcho verifies that RunSilentTimeout runs a command,
// captures its combined output, and returns no error on success.
func TestRunSilentTimeoutEcho(t *testing.T) {
	var out string
	var err error

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

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

func TestBoundedBuffer(t *testing.T) {
	buf := newBoundedBuffer(10)
	n, err := buf.Write([]byte("12345"))
	if err != nil || n != 5 {
		t.Fatalf("write failed: n=%d, err=%v", n, err)
	}
	if buf.IsTruncated() {
		t.Errorf("buffer should not be truncated yet")
	}

	n, err = buf.Write([]byte("67890EXTRA"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if !buf.IsTruncated() {
		t.Errorf("buffer should be marked truncated")
	}
	if buf.String() != "1234567890" {
		t.Errorf("expected '1234567890', got %q", buf.String())
	}
}

func TestRunSilentBounded_StructureAndExitCode(t *testing.T) {
	ctx := context.Background()
	var res ExecResult
	var err error

	if isWindows {
		res, err = RunSilentBounded(ctx, 1024, "", "cmd", "/c", "echo stdout message")
	} else {
		res, err = RunSilentBounded(ctx, 1024, "", "sh", "-c", "echo stdout message")
	}

	if err != nil {
		t.Fatalf("RunSilentBounded failed: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "stdout message") {
		t.Errorf("stdout does not contain message: %q", res.Stdout)
	}
	if res.TimedOut {
		t.Errorf("unexpected timeout")
	}
	if res.Duration <= 0 {
		t.Errorf("expected positive duration")
	}

	// Test non-zero exit code
	if isWindows {
		res, err = RunSilentBounded(ctx, 1024, "", "cmd", "/c", "exit 42")
	} else {
		res, err = RunSilentBounded(ctx, 1024, "", "sh", "-c", "exit 42")
	}
	if err == nil {
		t.Errorf("expected non-zero exit to return error")
	}
	if res.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", res.ExitCode)
	}
}

func TestRunSilentBounded_TimeoutHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var res ExecResult
	if isWindows {
		res, _ = RunSilentBounded(ctx, 1024, "", "powershell", "-Command", "Start-Sleep -Seconds 2")
	} else {
		res, _ = RunSilentBounded(ctx, 1024, "", "sleep", "2")
	}

	if !res.TimedOut {
		t.Errorf("expected TimedOut to be true")
	}
}
