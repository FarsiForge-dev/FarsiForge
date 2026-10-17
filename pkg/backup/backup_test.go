package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
)

func TestBackupAndRestoreRejectPathsOutsideGameRoot(t *testing.T) {
	gameRoot := t.TempDir()
	manager := New(gameRoot, filepath.Join(t.TempDir(), "backup"))

	if _, _, err := manager.BackupFile("../outside.bin"); err == nil {
		t.Fatal("BackupFile accepted a path outside the game root")
	}
	if err := manager.RestoreFile(`..\outside.bin`, ""); err == nil {
		t.Fatal("RestoreFile accepted a path outside the game root")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(gameRoot), "outside.bin")); !os.IsNotExist(err) {
		t.Fatalf("unexpected outside file: %v", err)
	}
}

func TestRollbackAll_NilAndEmptyManifest(t *testing.T) {
	gameRoot := t.TempDir()
	manager := New(gameRoot, filepath.Join(t.TempDir(), "backup"))

	// Nil manifest must not panic
	if err := manager.RollbackAll(nil); err != nil {
		t.Fatalf("expected nil error for nil manifest, got: %v", err)
	}

	// Empty manifest must succeed cleanly
	if err := manager.RollbackAll(&core.PatchManifest{}); err != nil {
		t.Fatalf("expected nil error for empty manifest, got: %v", err)
	}

	// Nil manager must return error safely
	var nilManager *BackupManager
	if err := nilManager.RollbackAll(&core.PatchManifest{}); err == nil {
		t.Fatal("expected error when calling RollbackAll on nil BackupManager")
	}
}

func TestBackupManager_UnsafeAndDevicePaths(t *testing.T) {
	gameRoot := t.TempDir()
	manager := New(gameRoot, filepath.Join(t.TempDir(), "backup"))

	unsafePaths := []string{
		"CON", "con.txt", "PRN", "aux.bin", "NUL",
		"COM1", "com2.txt", "LPT1",
		"Data/file.bin:stream", "trailing.", "trailing ",
		"/abs/path", `C:\abs\path`,
		"dir/../../escaped",
	}

	for _, p := range unsafePaths {
		if _, _, err := manager.BackupFile(p); err == nil {
			t.Errorf("BackupFile(%q) accepted unsafe or reserved path", p)
		}
		if err := manager.RestoreFile(p, ""); err == nil {
			t.Errorf("RestoreFile(%q) accepted unsafe or reserved path", p)
		}
	}
}

func TestBackupAndRestore_RoundTripAndHashMismatch(t *testing.T) {
	gameRoot := t.TempDir()
	backupDir := filepath.Join(t.TempDir(), "backup")
	manager := New(gameRoot, backupDir)

	relFile := "data/config.ini"
	srcPath := filepath.Join(gameRoot, "data", "config.ini")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	origContent := []byte("setting=123\n")
	if err := os.WriteFile(srcPath, origContent, 0o644); err != nil {
		t.Fatal(err)
	}
	origHashBytes := sha256.Sum256(origContent)
	origHash := hex.EncodeToString(origHashBytes[:])

	// 1. Create backup
	bPath, bHash, err := manager.BackupFile(relFile)
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}
	if bHash != origHash {
		t.Fatalf("got hash %s, want %s", bHash, origHash)
	}
	if !fileExists(bPath) {
		t.Fatalf("backup file %s does not exist", bPath)
	}

	// 2. Modify original file in game
	modContent := []byte("setting=MODIFIED\n")
	if err := os.WriteFile(srcPath, modContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Attempt Restore with incorrect hash -> must fail and preserve modified file
	if err := manager.RestoreFile(relFile, "deadbeef00000000deadbeef00000000deadbeef00000000deadbeef00000000"); err == nil {
		t.Fatal("expected RestoreFile to fail on hash mismatch")
	}
	currentContent, _ := os.ReadFile(srcPath)
	if string(currentContent) != string(modContent) {
		t.Fatal("game file was corrupted after hash mismatch restore failure")
	}

	// 4. Restore with correct hash
	if err := manager.RestoreFile(relFile, origHash); err != nil {
		t.Fatalf("RestoreFile failed: %v", err)
	}
	restoredContent, _ := os.ReadFile(srcPath)
	if string(restoredContent) != string(origContent) {
		t.Fatalf("restored content = %q, want %q", string(restoredContent), string(origContent))
	}

	// 5. RollbackAll with manifest
	manifest := &core.PatchManifest{
		Files: []core.PatchFile{
			{RelativePath: relFile, OriginalHash: origHash},
		},
	}
	if err := manager.RollbackAll(manifest); err != nil {
		t.Fatalf("RollbackAll failed: %v", err)
	}
}
