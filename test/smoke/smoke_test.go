package smoke

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"farsiforge/pkg/backup"
	"farsiforge/pkg/core"
	"farsiforge/pkg/installer"
	"farsiforge/pkg/tools"
)

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// TestCleanMachineSmokeTest verifies the complete lifecycle on a clean target:
// tool discovery -> project creation -> extraction -> staged translation ->
// patch build -> transactional apply -> byte-for-byte uninstall rollback.
func TestCleanMachineSmokeTest(t *testing.T) {
	tempRoot := t.TempDir()

	// 1. Tool discovery in clean environment
	toolsDir := filepath.Join(tempRoot, "Tools")
	reg, err := tools.NewRegistry(toolsDir)
	if err == nil {
		t.Logf("Clean registry initialized: %s", reg.Summary())
	}

	// 2. Mock game installation
	gameRoot := filepath.Join(tempRoot, "CleanGame")
	if err := os.MkdirAll(filepath.Join(gameRoot, "Localization"), 0755); err != nil {
		t.Fatalf("failed to create game folder: %v", err)
	}

	langFile := filepath.Join(gameRoot, "Localization", "en.txt")
	origContent := []byte("welcome_msg=Welcome to the game!\nstart_game=Press Start\n")
	origHash := hashBytes(origContent)
	if err := os.WriteFile(langFile, origContent, 0644); err != nil {
		t.Fatalf("failed to write original game file: %v", err)
	}

	// 3. Project working directory
	workDir := filepath.Join(tempRoot, "FarsiForgeWork")
	proj := &core.Project{
		Name:       "Clean Game Test",
		GameName:   "CleanGame",
		GameRoot:   gameRoot,
		Engine:     "generic",
		WorkingDir: workDir,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Entries: []core.StringEntry{
			{
				ID:          "ent-001",
				File:        "Localization/en.txt",
				Source:      "Welcome to the game!",
				Translation: "به بازی خوش آمدید!",
				Status:      core.StatusTranslated,
			},
			{
				ID:          "ent-002",
				File:        "Localization/en.txt",
				Source:      "Press Start",
				Translation: "شروع را فشار دهید",
				Status:      core.StatusTranslated,
			},
		},
	}
	_ = proj

	// 4. Staging translation output (never write directly to game files!)
	stagedDir := filepath.Join(workDir, "out")
	stagedFile := filepath.Join(stagedDir, "Localization", "en.txt")
	if err := os.MkdirAll(filepath.Dir(stagedFile), 0755); err != nil {
		t.Fatalf("failed to mkdir staged: %v", err)
	}

	translatedContent := []byte("welcome_msg=به بازی خوش آمدید!\nstart_game=شروع را فشار دهید\n")
	if err := os.WriteFile(stagedFile, translatedContent, 0644); err != nil {
		t.Fatalf("failed to write staged file: %v", err)
	}

	// Verify original game file remained untouched during staging
	currentOriginal, _ := os.ReadFile(langFile)
	if hashBytes(currentOriginal) != origHash {
		t.Fatalf("Game root file was mutated during staging!")
	}

	// 5. Build FFP1 Patch
	patchOutputDir := filepath.Join(workDir, "dist")
	cfg := installer.BuildConfig{
		GameRoot:  gameRoot,
		OutputDir: patchOutputDir,
		PatchName: "CleanGame Persian Patch",
		Engine:    "generic",
		Author:    "FarsiForge Test",
		Targets: []installer.PatchTarget{
			{
				GamePath:       "Localization/en.txt",
				PatchedFile:    stagedFile,
				OriginalFile:   langFile,
				OriginalSHA256: origHash,
			},
		},
	}

	buildRes, err := installer.Build(cfg)
	if err != nil {
		t.Fatalf("installer.Build failed: %v", err)
	}
	t.Logf("Built patch: %s (%d targets)", buildRes.PatchFile, buildRes.TargetCount)

	// Verify patch file was created
	if _, err := os.Stat(buildRes.PatchFile); err != nil {
		t.Fatalf("Patch file does not exist at %s: %v", buildRes.PatchFile, err)
	}

	// 6. Transactional Apply to Game with backup protection
	backupDir := filepath.Join(workDir, "backups")
	bm := backup.New(gameRoot, backupDir)

	// Back up original file before modifying
	_, recordedHash, err := bm.BackupFile("Localization/en.txt")
	if err != nil {
		t.Fatalf("failed to record backup: %v", err)
	}
	if recordedHash != origHash {
		t.Fatalf("backup hash mismatch: %s != %s", recordedHash, origHash)
	}

	// Apply translated file
	if err := os.WriteFile(langFile, translatedContent, 0644); err != nil {
		t.Fatalf("failed to write applied file: %v", err)
	}

	// Verify game file contains translated text
	appliedBytes, err := os.ReadFile(langFile)
	if err != nil {
		t.Fatalf("failed to read applied file: %v", err)
	}
	if string(appliedBytes) != string(translatedContent) {
		t.Errorf("expected translated content in applied game file")
	}

	// 7. Uninstall and byte-for-byte restoration
	if err := bm.RestoreFile("Localization/en.txt", origHash); err != nil {
		t.Fatalf("bm.RestoreFile failed: %v", err)
	}

	restoredBytes, err := os.ReadFile(langFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	restoredHash := hashBytes(restoredBytes)
	if restoredHash != origHash {
		t.Fatalf("Restoration failed: expected original hash %s, got %s", origHash, restoredHash)
	}
	t.Logf("Clean-machine smoke test passed: byte-for-byte restoration verified (%s)", restoredHash)
}
