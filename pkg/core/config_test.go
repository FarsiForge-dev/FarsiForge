package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Basic sanity checks — values should be non-empty
	if cfg.ToolsDir == "" {
		t.Error("ToolsDir should not be empty")
	}
	if cfg.OutputDir == "" {
		t.Error("OutputDir should not be empty")
	}
	if cfg.TempDir == "" {
		t.Error("TempDir should not be empty")
	}
	if cfg.MaxFileSizeMB <= 0 {
		t.Error("MaxFileSizeMB should be positive")
	}
	if cfg.MaxScanDepth <= 0 {
		t.Error("MaxScanDepth should be positive")
	}
	if cfg.PythonExe == "" {
		t.Error("PythonExe should not be empty")
	}
	if cfg.ProjectRoot == "" {
		t.Error("ProjectRoot should not be empty")
	}
	if len(cfg.GameSearchPaths) == 0 {
		t.Error("GameSearchPaths should not be empty")
	}
}

func TestDefaultConfig_PythonExeEnvOverride(t *testing.T) {
	t.Setenv(EnvPythonExe, "/custom/python3.11")
	cfg := DefaultConfig()
	if cfg.PythonExe != "/custom/python3.11" {
		t.Errorf("PythonExe = %q, want %q", cfg.PythonExe, "/custom/python3.11")
	}
}

func TestDefaultConfig_GameSearchPathEnvOverride(t *testing.T) {
	// Build the override with the platform list separator so the test does not
	// depend on Windows-style paths (a hardcoded `C:\...;D:\...` value only
	// worked on Windows). Entries are kept verbatim as explicit overrides.
	first := filepath.Join(string(filepath.Separator), "MyGames")
	second := filepath.Join(string(filepath.Separator), "MoreGames")
	t.Setenv(EnvGameSearchPath, strings.Join([]string{first, second}, string(filepath.ListSeparator)))
	cfg := DefaultConfig()

	for _, want := range []string{first, second} {
		found := false
		for _, p := range cfg.GameSearchPaths {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GameSearchPaths %v should contain env-provided path %q", cfg.GameSearchPaths, want)
		}
	}
}

func TestDefaultConfig_GameSearchPathEnvDeduplicates(t *testing.T) {
	shared := filepath.Join(string(filepath.Separator), "MyGames")
	t.Setenv(EnvGameSearchPath, strings.Join([]string{shared, shared}, string(filepath.ListSeparator)))
	cfg := DefaultConfig()

	count := 0
	for _, p := range cfg.GameSearchPaths {
		if p == shared {
			count++
		}
	}
	if count != 1 {
		t.Errorf("GameSearchPaths contains %q %d times, want exactly once", shared, count)
	}
}

func TestDefaultConfig_ProjectRootEnvOverride(t *testing.T) {
	// Create a temp dir with a fake go.mod
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(modPath, []byte("module farsiforge\n\ngo 1.26.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvProjectRoot, dir)
	cfg := DefaultConfig()
	if cfg.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", cfg.ProjectRoot, dir)
	}
}

func TestFindProjectRootUsesExactModuleName(t *testing.T) {
	t.Run("finds nested FarsiForge checkout", func(t *testing.T) {
		root := t.TempDir()
		nested := filepath.Join(root, "work")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module farsiforge\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := findProjectRootFrom(nested); got != root {
			t.Fatalf("findProjectRootFrom() = %q, want %q", got, root)
		}
	})

	t.Run("does not match similarly named modules", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/Amirwopi/farsiforge-tools\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := findProjectRootFrom(root); got != "" {
			t.Fatalf("findProjectRootFrom() = %q, want no match", got)
		}
	})
}

func TestDefaultGameSearchPathsUsesConfiguredSteamLibraries(t *testing.T) {
	temp := t.TempDir()
	lib := filepath.Join(temp, "library")
	common := filepath.Join(lib, "steamapps", "common")
	if err := os.MkdirAll(common, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"STEAM_PATH", "ProgramFiles", "ProgramFiles(x86)"} {
		t.Setenv(key, "")
	}
	t.Setenv("STEAM_LIBRARY_PATHS", lib+string(os.PathListSeparator)+lib)
	got := defaultGameSearchPaths(filepath.Join(temp, "home"))
	count := 0
	for _, candidate := range got {
		if candidate == common {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("defaultGameSearchPaths() = %#v, expected %q exactly once", got, common)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.DefaultAuthor = "TestAuthor"

	if err := SaveConfig(cfg, path); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.DefaultAuthor != "TestAuthor" {
		t.Errorf("DefaultAuthor = %q, want %q", loaded.DefaultAuthor, "TestAuthor")
	}
}

func TestLoadConfig_NonexistentFile(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadConfig should not error on missing file: %v", err)
	}
	if cfg.MaxScanDepth <= 0 {
		t.Error("missing config should return defaults")
	}
}
