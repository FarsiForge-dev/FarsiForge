package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkDirWithDiagnostics(t *testing.T) {
	root := t.TempDir()

	// Structure:
	// root/
	//   file1.txt
	//   file1.bin
	//   level1/
	//     file2.txt
	//     level2/
	//       file3.txt
	if err := os.WriteFile(filepath.Join(root, "file1.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file1.bin"), []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	lvl1 := filepath.Join(root, "level1")
	if err := os.MkdirAll(lvl1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lvl1, "file2.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	lvl2 := filepath.Join(lvl1, "level2")
	if err := os.MkdirAll(lvl2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lvl2, "file3.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Walk with depth 1 for .txt files
	matches, diag := WalkDirWithDiagnostics(root, 1, func(p string) bool {
		return filepath.Ext(p) == ".txt"
	})

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches at depth 1, got %d: %v", len(matches), matches)
	}
	if diag.TotalScannedFiles < 3 {
		t.Fatalf("expected at least 3 scanned files, got %d", diag.TotalScannedFiles)
	}

	// 2. WalkDir wrapper compatibility
	simpleMatches := WalkDir(root, 1, func(p string) bool {
		return filepath.Ext(p) == ".txt"
	})
	if len(simpleMatches) != len(matches) {
		t.Fatalf("WalkDir returned %d, want %d", len(simpleMatches), len(matches))
	}
}

func TestScannerHelpers(t *testing.T) {
	root := t.TempDir()

	fPath := filepath.Join(root, "game.exe")
	if err := os.WriteFile(fPath, []byte("exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	dPath := filepath.Join(root, "Data")
	if err := os.MkdirAll(dPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if !FileExists(fPath) {
		t.Fatal("FileExists returned false for existing file")
	}
	if FileExists(dPath) {
		t.Fatal("FileExists returned true for a directory")
	}
	if !DirExists(dPath) {
		t.Fatal("DirExists returned false for existing dir")
	}
	if DirExists(fPath) {
		t.Fatal("DirExists returned true for a file")
	}

	if !HasFileWithSuffix(root, ".exe") {
		t.Fatal("HasFileWithSuffix failed to find .exe")
	}
	if HasFileWithSuffix(root, ".dll") {
		t.Fatal("HasFileWithSuffix found non-existent .dll")
	}

	if !HasFileWithPrefix(root, "game") {
		t.Fatal("HasFileWithPrefix failed to find game prefix")
	}

	foundSub := FindSubdir(root, "data")
	if foundSub == "" {
		t.Fatal("FindSubdir failed to find Data directory case-insensitively")
	}

	exe, name := FindGameExe(root, []string{".exe"})
	if exe == "" || name != "game" {
		t.Fatalf("FindGameExe returned (%q, %q), want game.exe and game", exe, name)
	}
}
