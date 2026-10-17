package tools

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"
)

// DefaultMaxOutputBytes is the maximum bytes captured from stdout/stderr (2 MB)
// to prevent memory exhaustion from runaway subprocess output.
const DefaultMaxOutputBytes = 2 * 1024 * 1024

// ExecResult contains structured outcome of a subprocess run.
type ExecResult struct {
	ExitCode  int           `json:"exit_code"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	Combined  string        `json:"combined"`
	Duration  time.Duration `json:"duration"`
	TimedOut  bool          `json:"timed_out"`
	Truncated bool          `json:"truncated"`
}

// boundedBuffer captures up to limit bytes, safely dropping anything beyond
// and recording truncation.
type boundedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	if limit <= 0 {
		limit = DefaultMaxOutputBytes
	}
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	curLen := b.buf.Len()
	if curLen >= b.limit {
		b.truncated = true
		return len(p), nil
	}

	remaining := b.limit - curLen
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}

	return b.buf.Write(p)
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *boundedBuffer) IsTruncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

// RunSilentBounded executes a command silently with bounded output collection,
// strict timeout/cancellation handling, and structured outcome reporting.
func RunSilentBounded(ctx context.Context, maxBytes int, dir, name string, args ...string) (ExecResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	applySilentSysProcAttr(cmd)

	stdoutBuf := newBoundedBuffer(maxBytes)
	stderrBuf := newBoundedBuffer(maxBytes)
	combinedBuf := newBoundedBuffer(maxBytes)

	cmd.Stdout = io.MultiWriter(stdoutBuf, combinedBuf)
	cmd.Stderr = io.MultiWriter(stderrBuf, combinedBuf)

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	res := ExecResult{
		ExitCode:  exitCode,
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
		Combined:  combinedBuf.String(),
		Duration:  duration,
		TimedOut:  timedOut,
		Truncated: stdoutBuf.IsTruncated() || stderrBuf.IsTruncated() || combinedBuf.IsTruncated(),
	}

	return res, err
}

// RunSilent runs a command with no visible window, capturing combined stdout
// and stderr output up to DefaultMaxOutputBytes.
func RunSilent(ctx context.Context, dir, name string, args ...string) (string, error) {
	res, err := RunSilentBounded(ctx, DefaultMaxOutputBytes, dir, name, args...)
	return res.Combined, err
}

// RunSilentTimeout is RunSilent with an explicit timeout.
func RunSilentTimeout(timeout time.Duration, dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return RunSilent(ctx, dir, name, args...)
}

// Run executes a command silently and returns its combined output.
func Run(name string, args ...string) (string, error) {
	return RunSilent(context.Background(), "", name, args...)
}

// RunDir executes a command silently with a specific working directory.
func RunDir(dir, name string, args ...string) (string, error) {
	return RunSilent(context.Background(), dir, name, args...)
}
