package assetbrowser_test

import (
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/assetbrowser"
)

func TestBrowseGameAssets_Unreal(t *testing.T) {
	tmp := t.TempDir()

	// Setup synthetic Unreal game structure
	pakDir := filepath.Join(tmp, "Content", "Paks")
	if err := os.MkdirAll(pakDir, 0755); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(tmp, "Content", "Localization", "Game")
	if err := os.MkdirAll(locDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files
	_ = os.WriteFile(filepath.Join(pakDir, "Game-Windows.pak"), []byte("FAKE_PAK_DATA"), 0644)
	_ = os.WriteFile(filepath.Join(pakDir, "Game-Windows.utoc"), []byte("FAKE_UTOC_DATA"), 0644)
	_ = os.WriteFile(filepath.Join(locDir, "Game.locres"), make([]byte, 128), 0644)
	_ = os.WriteFile(filepath.Join(locDir, "Widget.uexp"), []byte("UE_BLUEPRINT_UEXP_DATA"), 0644)
	_ = os.WriteFile(filepath.Join(locDir, "Widget.uasset"), []byte("UE_UASSET_HEADER"), 0644)

	view, err := assetbrowser.BrowseGameAssets(tmp, "unreal")
	if err != nil {
		t.Fatalf("BrowseGameAssets failed: %v", err)
	}

	if view.TotalFiles != 5 {
		t.Errorf("expected 5 total files, got %d", view.TotalFiles)
	}
	if view.TotalContainers != 2 {
		t.Errorf("expected 2 containers (.pak, .utoc), got %d", view.TotalContainers)
	}

	// Verify root tree structure
	if len(view.RootNodes) != 1 || view.RootNodes[0].Name != "Content" {
		t.Fatalf("expected Content folder at root, got %+v", view.RootNodes)
	}

	contentNode := view.RootNodes[0]
	if len(contentNode.Children) != 2 {
		t.Fatalf("expected 2 subfolders in Content, got %d", len(contentNode.Children))
	}
}

func TestBrowseGameAssets_FromSoftware(t *testing.T) {
	tmp := t.TempDir()

	// Data0 container
	_ = os.WriteFile(filepath.Join(tmp, "Data0.bhd"), []byte("BHD5_HEADER"), 0644)
	_ = os.WriteFile(filepath.Join(tmp, "Data0.bdt"), []byte("BDT_PAYLOAD"), 0644)

	msgDir := filepath.Join(tmp, "msg", "engus")
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(msgDir, "item.fmg"), []byte("FAKE_FMG"), 0644)

	view, err := assetbrowser.BrowseGameAssets(tmp, "fromsoftware")
	if err != nil {
		t.Fatalf("BrowseGameAssets failed: %v", err)
	}

	if view.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", view.TotalFiles)
	}

	// Verify Data0 container node has virtual catalog children
	var data0Node *assetbrowser.AssetNode
	for _, n := range view.RootNodes {
		if n.Name == "Data0.bhd" {
			data0Node = n
			break
		}
	}
	if data0Node == nil {
		t.Fatal("Data0.bhd node not found at root")
	}
	if data0Node.Type != assetbrowser.NodeContainer {
		t.Errorf("expected NodeContainer for Data0.bhd, got %v", data0Node.Type)
	}
	if len(data0Node.Children) == 0 {
		t.Error("expected Data0.bhd to have member children")
	}
	for _, child := range data0Node.Children {
		if !child.ReadOnly || child.CanInject {
			t.Errorf("expected Data0 member %s to be read-only", child.Name)
		}
	}
}

func TestBrowseGameAssets_GodotAndUnity(t *testing.T) {
	tmp := t.TempDir()

	// Godot
	_ = os.WriteFile(filepath.Join(tmp, "game.pck"), []byte("GDPCK"), 0644)
	_ = os.WriteFile(filepath.Join(tmp, "dialogue.tres"), []byte(`[resource]\ntext = "hello"`), 0644)

	view, err := assetbrowser.BrowseGameAssets(tmp, "godot")
	if err != nil {
		t.Fatalf("BrowseGameAssets failed: %v", err)
	}
	if view.TotalContainers != 1 {
		t.Errorf("expected 1 PCK container, got %d", view.TotalContainers)
	}

	// Unity
	unityTmp := t.TempDir()
	dataDir := filepath.Join(unityTmp, "Game_Data")
	_ = os.MkdirAll(dataDir, 0755)
	_ = os.WriteFile(filepath.Join(dataDir, "sharedassets0.assets"), []byte("UNITY_ASSETS"), 0644)
	_ = os.WriteFile(filepath.Join(dataDir, "subtitles.txt"), []byte("Line 1\nLine 2"), 0644)

	uView, err := assetbrowser.BrowseGameAssets(unityTmp, "unity")
	if err != nil {
		t.Fatalf("BrowseGameAssets failed: %v", err)
	}
	if uView.TotalContainers != 1 {
		t.Errorf("expected 1 assets container, got %d", uView.TotalContainers)
	}
}

func TestBrowseGameAssets_InvalidRoot(t *testing.T) {
	_, err := assetbrowser.BrowseGameAssets(filepath.Join(t.TempDir(), "nonexistent"), "generic")
	if err == nil {
		t.Error("expected error on nonexistent root, got nil")
	}
}
