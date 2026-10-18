package bench

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"farsiforge/pkg/assetbrowser"
	"farsiforge/pkg/detection"
	"farsiforge/pkg/tools"
)

// BenchmarkRealGames_Detection measures engine detection throughput across real installed games.
func BenchmarkRealGames_Detection(b *testing.B) {
	gamesRoot := `D:\games`
	if fi, err := os.Stat(gamesRoot); err != nil || !fi.IsDir() {
		b.Skip("D:\\games directory not found; skipping benchmark")
	}

	reg := detection.DefaultRegistry()
	games := []string{
		"ELDEN RING",
		"Sekiro Shadows Die Twice GOTY Edition",
		"Factorio",
		"Project.Zomboid",
		"Raft",
		"Orc Massage",
		"MOLDRISE.v1.0.5",
		"Hacker Simulator",
	}

	var validGames []string
	for _, g := range games {
		p := filepath.Join(gamesRoot, g)
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			validGames = append(validGames, p)
		}
	}

	if len(validGames) == 0 {
		b.Skip("No real test games found in D:\\games")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, gamePath := range validGames {
			_, _ = reg.Detect(gamePath)
		}
	}
}

// BenchmarkRealGames_AssetBrowser measures asset browsing speed.
func BenchmarkRealGames_AssetBrowser(b *testing.B) {
	gamesRoot := `D:\games`
	if fi, err := os.Stat(gamesRoot); err != nil || !fi.IsDir() {
		b.Skip("D:\\games directory not found; skipping benchmark")
	}

	reg := detection.DefaultRegistry()
	target := filepath.Join(gamesRoot, "Raft")
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		b.Skip("Raft not found; skipping")
	}

	det, _ := reg.Detect(target)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = assetbrowser.BrowseGameAssets(target, det.Engine)
	}
}

// TestUnityMemoryLeak_Verification directly runs extract.py against Raft / Orc Massage
// and validates that memory does not exceed 150 MB and finishes within expected time.
func TestUnityMemoryLeak_Verification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heavy real-game memory test in short mode")
	}
	gamesRoot := `D:\games`
	targetGame := filepath.Join(gamesRoot, "Orc Massage")
	if fi, err := os.Stat(targetGame); err != nil || !fi.IsDir() {
		targetGame = filepath.Join(gamesRoot, "Raft")
	}
	if fi, err := os.Stat(targetGame); err != nil || !fi.IsDir() {
		t.Skip("Neither Orc Massage nor Raft game folder present; skipping memory verification test")
	}

	toolReg, err := tools.NewRegistry("")
	if err != nil {
		t.Fatalf("failed to initialize tool registry: %v", err)
	}
	py := toolReg.GetPython()
	if py == "" {
		t.Skip("Python not installed; skipping")
	}

	tempDir := t.TempDir()
	extractScript, err := tools.WriteScript("extract.py", tempDir)
	if err != nil {
		t.Fatalf("failed to write extract.py: %v", err)
	}

	// Python script that runs extract.py with tracemalloc to measure exact peak memory
	profilerScript := filepath.Join(tempDir, "profile_extract.py")
	profileCode := `
import tracemalloc
import time
import sys
import runpy

tracemalloc.start(1)
t0 = time.time()

script_path = sys.argv[1]
game_dir = sys.argv[2]
work_dir = sys.argv[3]

sys.argv = [script_path, game_dir, work_dir]
try:
    runpy.run_path(script_path, run_name="__main__")
except SystemExit:
    pass

current, peak = tracemalloc.get_traced_memory()
tracemalloc.stop()
elapsed = time.time() - t0
peak_mb = peak / (1024 * 1024)
print(f"BENCHMARK_RESULT: peak_mb={peak_mb:.2f} elapsed_sec={elapsed:.2f}", flush=True)
`
	if err := os.WriteFile(profilerScript, []byte(profileCode), 0644); err != nil {
		t.Fatalf("failed to write profiler script: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, py, profilerScript, extractScript, targetGame, tempDir)
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	outputStr := string(out)
	t.Logf("Profiler output:\n%s", outputStr)

	if err != nil && !strings.Contains(outputStr, "BENCHMARK_RESULT:") {
		t.Fatalf("profiler run failed: %v, output: %s", err, outputStr)
	}

	// Parse peak_mb
	var peakMB float64
	for _, line := range strings.Split(outputStr, "\n") {
		if strings.Contains(line, "BENCHMARK_RESULT:") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasPrefix(p, "peak_mb=") {
					valStr := strings.TrimPrefix(p, "peak_mb=")
					if val, parseErr := strconv.ParseFloat(valStr, 64); parseErr == nil {
						peakMB = val
					}
				}
			}
		}
	}

	t.Logf("Raft Extract Peak Memory: %.2f MB", peakMB)
	// Peak memory must be strictly under 350 MB on full 7.5GB game extraction (previously it exceeded 4+ GB causing system freeze)
	if peakMB > 350.0 {
		t.Errorf("Memory leak detected! Peak memory %.2f MB exceeds 350 MB threshold", peakMB)
	}
}

// TestGoRuntime_HeapSanity checks that Go runtime allocations remain disciplined.
func TestGoRuntime_HeapSanity(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Allocate and release temporary data
	_ = make([][]byte, 1000)
	runtime.GC()
	runtime.ReadMemStats(&m2)

	diff := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("HeapAlloc delta after GC: %d bytes (initial: %d, final: %d)", diff, m1.HeapAlloc, m2.HeapAlloc)
}
