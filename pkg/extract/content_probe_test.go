package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
)

func TestHasUnityContent(t *testing.T) {
	t.Parallel()

	t.Run("empty data folder has no content", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "Game_Data"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "Game.exe"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if hasUnityContent(root) {
			t.Error("an empty <Game>_Data folder must not count as extractable content")
		}
	})

	t.Run("serialized assets count as content", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "resources.assets"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if !hasUnityContent(root) {
			t.Error("resources.assets must count as extractable content")
		}
	})

	t.Run("files inside a data folder count as content", func(t *testing.T) {
		root := t.TempDir()
		managed := filepath.Join(root, "Game_Data", "Managed")
		if err := os.MkdirAll(managed, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(managed, "Assembly-CSharp.dll"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if !hasUnityContent(root) {
			t.Error("files under <Game>_Data must count as extractable content")
		}
	})

	t.Run("missing directory has no content", func(t *testing.T) {
		if hasUnityContent(filepath.Join(t.TempDir(), "does-not-exist")) {
			t.Error("a missing directory must not report content")
		}
		if hasUnityContent("") {
			t.Error("an empty root must not report content")
		}
	})
}

func TestHasExtractableContentUsesDataPathOrRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dataDir := filepath.Join(root, "Game_Data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "level0"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	info := &core.GameInfo{GameRoot: root, DataPath: dataDir}
	if !(&UnityExtractor{}).HasExtractableContent(info) {
		t.Error("Unity extractor should find content through DataPath")
	}
	if (&UnityExtractor{}).HasExtractableContent(nil) {
		t.Error("nil game info must not report content")
	}
}

func TestUnrealAndGodotContentProbes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Game", "pakchunk0.pak"), []byte("x"), 0o755); err != nil {
		if err = os.MkdirAll(filepath.Join(root, "Game"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, "Game", "pakchunk0.pak"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	info := &core.GameInfo{GameRoot: root}
	if !(&UnrealExtractor{}).HasExtractableContent(info) {
		t.Error("Unreal extractor should detect .pak content")
	}
	if (&GodotExtractor{}).HasExtractableContent(info) {
		t.Error("Godot extractor must not treat .pak files as Godot content")
	}
}

// fakeToolRegistry is a minimal core.ToolRegistry for pipeline tests.
type fakeToolRegistry struct {
	available map[string]bool
}

func (f fakeToolRegistry) IsAvailable(name string) bool { return f.available[name] }
func (f fakeToolRegistry) GetPath(string) string        { return "" }
func (f fakeToolRegistry) GetPython() string            { return "" }
func (f fakeToolRegistry) Summary() string              { return "" }

func TestRunSkipsExtractionWhenNoContentAndToolMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Game_Data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "UnityPlayer.dll"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := core.NewProject("probe", root, "unity")
	info := &core.GameInfo{Engine: "unity", GameRoot: root}

	// No python/UnityPy available: a game without assets must still succeed
	// (with zero entries) instead of failing on the missing tool.
	if err := Run(context.Background(), info, proj, fakeToolRegistry{}); err != nil {
		t.Fatalf("Run() with no extractable content = %v, want nil", err)
	}
}

func TestRunFailsWhenContentExistsButToolMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "resources.assets"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := core.NewProject("probe", root, "unity")
	info := &core.GameInfo{Engine: "unity", GameRoot: root}

	err := Run(context.Background(), info, proj, fakeToolRegistry{})
	if err == nil {
		t.Fatal("Run() with real assets and a missing tool = nil, want an error")
	}
}
