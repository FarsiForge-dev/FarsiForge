package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	binaryName = "farsiforge-cli-test"
)

// TestMain builds the CLI binary once and runs all tests.
func TestMain(m *testing.M) {
	// Build the binary
	binPath := filepath.Join(os.TempDir(), binaryName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/farsiforge-cli")
	cmd.Dir = getProjectRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build CLI: %v\n%s", err, out)
		os.Exit(1)
	}
	defer os.Remove(binPath)

	// Store binary path for tests
	os.Setenv("FARSIFORGE_CLI_BIN", binPath)

	os.Exit(m.Run())
}

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

func cli(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	bin := os.Getenv("FARSIFORGE_CLI_BIN")
	if bin == "" {
		t.Fatal("FARSIFORGE_CLI_BIN not set")
	}
	return exec.Command(bin, args...)
}

func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	bin := os.Getenv("FARSIFORGE_CLI_BIN")
	if bin == "" {
		t.Fatal("FARSIFORGE_CLI_BIN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	return string(out), "", err
}

// ── Engine detection with synthetic fixtures ──────────────────────────────

func TestCLIDetectUnity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create minimal Unity game structure
	os.MkdirAll(filepath.Join(dir, "Game_Data"), 0o755)
	os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("dummy"), 0o644)
	os.WriteFile(filepath.Join(dir, "UnityPlayer.dll"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Engine     : unity") {
		t.Errorf("expected 'unity' engine, got:\n%s", out)
	}
}

func TestCLIDetectUnreal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create minimal Unreal game structure
	os.MkdirAll(filepath.Join(dir, "Game", "Content", "Paks"), 0o755)
	os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("dummy"), 0o644)
	os.WriteFile(filepath.Join(dir, "Game", "Content", "Paks", "game.pak"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Engine     : unreal") {
		t.Errorf("expected 'unreal' engine, got:\n%s", out)
	}
}

func TestCLIDetectGodot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create minimal Godot game structure
	os.MkdirAll(filepath.Join(dir, "game"), 0o755)
	os.WriteFile(filepath.Join(dir, "game.exe"), []byte("dummy"), 0o644)
	// Godot 4 PCK v4 magic
	pckData := make([]byte, 8)
	copy(pckData, "GDPK")
	// Version 4
	pckData[4] = 4
	pckData[5] = 0
	pckData[6] = 0
	pckData[7] = 0
	os.WriteFile(filepath.Join(dir, "game.pck"), pckData, 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Engine     : godot") {
		t.Errorf("expected 'godot' engine, got:\n%s", out)
	}
}

func TestCLIDetectSAGE(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create minimal SAGE game structure
	os.MkdirAll(filepath.Join(dir, "Data"), 0o755)
	os.WriteFile(filepath.Join(dir, "game.exe"), []byte("dummy"), 0o644)
	// SAGE needs: .skudef files (score 4) OR generals.exe (score 4) OR .big files + other markers
	// Add .skudef file for score 4
	os.WriteFile(filepath.Join(dir, "game.skudef"), []byte("dummy"), 0o644)
	// Also add .big file in Data/
	os.WriteFile(filepath.Join(dir, "Data", "english.big"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Engine     : sage") {
		t.Errorf("expected 'sage' engine, got:\n%s", out)
	}
}

func TestCLIDetectRAGE(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "update"), 0o755)
	os.MkdirAll(filepath.Join(dir, "x64"), 0o755)
	os.WriteFile(filepath.Join(dir, "GTA5.exe"), []byte("dummy"), 0o644)
	// RAGE needs: .rpf files (score 4) + GTA5.exe (score 3) + update/ + x64/ (score 2) = min 5
	// Add .rpf file
	os.WriteFile(filepath.Join(dir, "x64", "levels.gta5.rpf"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Engine     : rage") {
		t.Errorf("expected 'rage' engine, got:\n%s", out)
	}
}

func TestCLIDetectFromSoftware(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Game"), 0o755)
	os.WriteFile(filepath.Join(dir, "Game", "eldenring.exe"), []byte("dummy"), 0o644)
	// FromSoftware needs: .bdt (score 3) + .bhd (score 2) + regulation.bin (score 2) + eldenring.exe (score 2) = min 4
	// Add .bdt and .bhd files at root
	os.WriteFile(filepath.Join(dir, "Game", "Data0.bdt"), []byte("dummy"), 0o644)
	os.WriteFile(filepath.Join(dir, "Game", "Data0.bhd"), []byte("dummy"), 0o644)
	// Add regulation.bin
	os.WriteFile(filepath.Join(dir, "Game", "regulation.bin"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Engine     : fromsoftware") {
		t.Errorf("expected 'fromsoftware' engine, got:\n%s", out)
	}
}

func TestCLIDetectGeneric(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "game"), 0o755)
	os.WriteFile(filepath.Join(dir, "game.exe"), []byte("dummy"), 0o644)
	// Generic text files
	os.WriteFile(filepath.Join(dir, "game", "readme.txt"), []byte("Welcome to the game"), 0o644)
	os.WriteFile(filepath.Join(dir, "game", "config.json"), []byte(`{"name": "Test Game"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "game", "data.csv"), []byte("key,value\nhello,world\n"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	// Generic detector returns "custom" as engine name
	if !strings.Contains(out, "Engine     : custom") {
		t.Errorf("expected 'custom' engine, got:\n%s", out)
	}
}

// ── Error paths ───────────────────────────────────────────────────────────

func TestCLIMissingDir(t *testing.T) {
	t.Parallel()
	bin := os.Getenv("FARSIFORGE_CLI_BIN")
	if bin == "" {
		t.Fatal("FARSIFORGE_CLI_BIN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-action", "detect")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing -dir")
	}
	outStr := string(out)
	if !strings.Contains(outStr, "--dir is required") && !strings.Contains(outStr, "dir is required") {
		t.Errorf("expected 'dir is required' error, got:\n%s", outStr)
	}
}

func TestCLINonexistentDir(t *testing.T) {
	t.Parallel()
	out, _, err := runCLI(t, "-dir", "/nonexistent/path/12345", "-action", "detect")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
	outStr := string(out)
	if !strings.Contains(outStr, "Detection error") && !strings.Contains(outStr, "error") {
		t.Errorf("expected error output, got:\n%s", outStr)
	}
}

func TestCLIUnsupportedFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "game"), 0o755)
	os.WriteFile(filepath.Join(dir, "game.exe"), []byte("dummy"), 0o644)
	// Only unknown file types - should fall back to generic or fail
	os.WriteFile(filepath.Join(dir, "game", "data.xyz"), []byte("unknown"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	// Should still detect something (generic)
	if !strings.Contains(out, "Engine") {
		t.Errorf("expected engine detection output, got:\n%s", out)
	}
}

// ── Deterministic output ──────────────────────────────────────────────────

func TestCLIDeterministicDetect(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Game_Data"), 0o755)
	os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("dummy"), 0o644)
	os.WriteFile(filepath.Join(dir, "UnityPlayer.dll"), []byte("dummy"), 0o644)

	var outputs []string
	for i := 0; i < 5; i++ {
		out, _, err := runCLI(t, "-dir", dir, "-action", "detect")
		if err != nil {
			t.Fatalf("run %d: CLI failed: %v\n%s", i, err, out)
		}
		// Strip timestamps for comparison
		stripped := stripTimestamps(out)
		outputs = append(outputs, stripped)
	}
	// All outputs should be identical (after stripping timestamps)
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("run %d output differs from run 0:\n---\n%s\n---\nvs\n---\n%s\n---", i, outputs[0], outputs[i])
		}
	}
}

// stripTimestamps removes timestamp prefixes and log lines from output
func stripTimestamps(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		// Skip JSON log lines (contain [detection], [extract], etc.)
		if strings.Contains(line, "] [") && (strings.Contains(line, "detection") || strings.Contains(line, "extract") || strings.Contains(line, "inject")) {
			continue
		}
		// Remove timestamp like [2026-10-11 21:46:54] [INFO ]
		if idx := strings.Index(line, "] ["); idx != -1 {
			if idx2 := strings.Index(line[idx+2:], "] "); idx2 != -1 {
				line = line[idx+idx2+4:]
			}
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// ── Extract action (without game assets, should complete with 0 entries) ───

func TestCLIExtractEmptyUnity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Game_Data"), 0o755)
	os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("dummy"), 0o644)
	os.WriteFile(filepath.Join(dir, "UnityPlayer.dll"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "extract")
	if err != nil {
		if strings.Contains(string(out), "UnityPy Python package is not installed") {
			t.Skip("UnityPy is not installed on test runner; skipping TestCLIExtractEmptyUnity")
		}
		t.Fatalf("CLI failed: %v\nOutput:\n%s", err, out)
	}
	outStr := string(out)
	// Should complete successfully with 0 entries (no actual game assets)
	if !strings.Contains(outStr, "Extraction successful") {
		t.Errorf("expected successful extraction, got:\n%s", outStr)
	}
}

// ── Search action (works on existing project) ─────────────────────────────

func TestCLISearchNoProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Game_Data"), 0o755)
	os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("dummy"), 0o644)

	out, _, err := runCLI(t, "-dir", dir, "-action", "search", "-query", "test")
	if err == nil {
		t.Fatal("expected error for missing project file")
	}
	outStr := string(out)
	if !strings.Contains(outStr, "project file") && !strings.Contains(outStr, "run.*extract") {
		t.Errorf("expected 'no project file' error, got:\n%s", outStr)
	}
}

// ── Help/Usage ────────────────────────────────────────────────────────────

func TestCLIHelp(t *testing.T) {
	t.Parallel()
	bin := os.Getenv("FARSIFORGE_CLI_BIN")
	if bin == "" {
		t.Fatal("FARSIFORGE_CLI_BIN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-help")
	out, err := cmd.CombinedOutput()
	// Help typically exits with 0 in Go flag.Parse
	outStr := string(out)
	if !strings.Contains(outStr, "-dir") || !strings.Contains(outStr, "-action") {
		t.Errorf("help should show flags, got:\n%s", outStr)
	}
	_ = err // ignore exit code, Go flag.Parse returns 0 for -help
}
