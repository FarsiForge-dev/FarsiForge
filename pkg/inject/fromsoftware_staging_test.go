package inject

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
	"farsiforge/pkg/extract"
)

// buildMinimalFMG returns a minimal FMG v2 little-endian file with one entry.
func buildMinimalFMG(t *testing.T, id int32, text string) []byte {
	t.Helper()
	entries := []extract.FMGEntry{{ID: id, Text: text}}
	data, err := extract.EncodeFMG(entries, 2, true, false)
	if err != nil {
		t.Fatalf("EncodeFMG: %v", err)
	}
	return data
}

// TestFromSoftwareInjector_StagingContract verifies that:
// 1. The original installed game file is byte-for-byte identical after injection.
// 2. The staged output appears under WorkingDir/out/.
// 3. ModifiedFileHashes maps the game-relative path to the original file's SHA-256.
func TestFromSoftwareInjector_StagingContract(t *testing.T) {
	// --- set up fake game root ---
	gameRoot := t.TempDir()
	relPath := filepath.Join("msg", "engus", "item.fmg")
	gameSrc := filepath.Join(gameRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(gameSrc), 0o755); err != nil {
		t.Fatal(err)
	}
	originalFMG := buildMinimalFMG(t, 100, "Original Text")
	if err := os.WriteFile(gameSrc, originalFMG, 0o644); err != nil {
		t.Fatal(err)
	}

	// --- set up project ---
	workDir := t.TempDir()
	proj := &core.Project{
		GameRoot:   gameRoot,
		WorkingDir: workDir,
		Entries: []core.StringEntry{
			{
				ID:          "fromsoftware:100",
				File:        filepath.ToSlash(relPath),
				Path:        "100",
				Source:      "Original Text",
				Translation: "ترجمه تستی",
				Status:      core.StatusTranslated,
			},
		},
	}

	info := &core.GameInfo{
		Engine:   "fromsoftware",
		GameRoot: gameRoot,
	}

	inj := &FromSoftwareInjector{}
	res, err := inj.Inject(t.Context(), info, proj, nil, core.DefaultPersianOptions())
	if err != nil {
		t.Fatalf("Inject error: %v", err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("Inject returned errors: %v", res.Errors)
	}

	// 1. Original game file must be untouched.
	afterBytes, err := os.ReadFile(gameSrc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(originalFMG, afterBytes) {
		t.Errorf("game file was mutated: original %d bytes, now %d bytes", len(originalFMG), len(afterBytes))
	}

	// 2. Staged file must exist under WorkingDir/out/.
	stagedFile := filepath.Join(workDir, "out", relPath)
	if _, err := os.Stat(stagedFile); err != nil {
		t.Errorf("staged file not found at %s: %v", stagedFile, err)
	}

	// 3. ModifiedFileHashes must contain the original hash.
	slashRel := filepath.ToSlash(relPath)
	wantHash := sha256File(originalFMG)
	gotHash, ok := proj.ModifiedFileHashes[slashRel]
	if !ok {
		t.Errorf("ModifiedFileHashes missing entry for %q", slashRel)
	} else if gotHash != wantHash {
		t.Errorf("hash mismatch: want %s got %s", wantHash, gotHash)
	}

	// 4. ModifiedFiles must list the game-relative path.
	found := false
	for _, f := range res.ModifiedFiles {
		if f == slashRel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ModifiedFiles does not contain %q, got: %v", slashRel, res.ModifiedFiles)
	}
}

// TestFromSoftwareInjector_StagedPathTraversal verifies that a path trying to
// escape the WorkingDir/out boundary is rejected.
func TestFromSoftwareInjector_StagedPathTraversal(t *testing.T) {
	workDir := t.TempDir()
	_, err := stagedOutputPath(workDir, "../../etc/passwd")
	if err == nil {
		t.Error("expected error for traversal path, got nil")
	}
}

// TestFromSoftwareInjector_EmptyWorkingDir ensures a missing WorkingDir is an
// error rather than a silent write somewhere unexpected.
func TestFromSoftwareInjector_EmptyWorkingDir(t *testing.T) {
	_, err := stagedOutputPath("", "msg/item.fmg")
	if err == nil {
		t.Error("expected error for empty WorkingDir, got nil")
	}
}

// TestAtomicWriteFile_CorruptionGuard writes via atomicWriteFile and confirms
// the destination has the correct content.
func TestAtomicWriteFile_CorruptionGuard(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	want := []byte("hello atomic world")
	if err := atomicWriteFile(dest, want, 0o644); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("content mismatch: want %q got %q", want, got)
	}
}

