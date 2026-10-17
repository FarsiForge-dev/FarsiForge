package exchange

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
)

// makeProject creates a test project with two entries.
func makeProject(t *testing.T) *core.Project {
	t.Helper()
	proj := core.NewProject("test", t.TempDir(), "unity")
	proj.AddEntry(core.StringEntry{
		ID:     "entry-1",
		Source: "Hello",
		File:   "ui.json",
		Status: core.StatusUntranslated,
	})
	proj.AddEntry(core.StringEntry{
		ID:     "entry-2",
		Source: "World",
		File:   "ui.json",
		Status: core.StatusUntranslated,
	})
	return proj
}

// writeCSV writes a CSV file with BOM and returns its path.
func writeCSV(t *testing.T, rows [][]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "import.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Write UTF-8 BOM
	_, _ = f.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(f)
	_ = w.Write([]string{"ID", "Source", "Translation", "Status", "Context", "File", "Path", "Notes"})
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
	return path
}

// TestImportCSV_Valid verifies a clean import applies all rows.
func TestImportCSV_Valid(t *testing.T) {
	proj := makeProject(t)
	path := writeCSV(t, [][]string{
		{"entry-1", "Hello", "سلام", "translated"},
		{"entry-2", "World", "جهان", "approved"},
	})
	res, err := ImportCSV(proj, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Accepted != 2 {
		t.Errorf("accepted = %d, want 2", res.Accepted)
	}
	if res.Rejected != 0 {
		t.Errorf("rejected = %d, want 0", res.Rejected)
	}
	// Project state must be updated.
	if proj.Entries[0].Translation != "سلام" {
		t.Errorf("entry-1 translation = %q, want سلام", proj.Entries[0].Translation)
	}
	if proj.Entries[1].Status != core.StatusApproved {
		t.Errorf("entry-2 status = %q, want approved", proj.Entries[1].Status)
	}
}

// TestImportCSV_InvalidStatus verifies that an invalid status value causes a
// full import rollback — zero changes applied.
func TestImportCSV_InvalidStatus(t *testing.T) {
	proj := makeProject(t)
	path := writeCSV(t, [][]string{
		{"entry-1", "Hello", "سلام", "INVALID_STATUS"},
		{"entry-2", "World", "جهان", "approved"},
	})
	res, err := ImportCSV(proj, path)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if res.Rejected != 1 {
		t.Errorf("rejected = %d, want 1", res.Rejected)
	}
	// ATOMIC: no rows should have been applied.
	if proj.Entries[0].Translation != "" {
		t.Errorf("entry-1 was mutated despite invalid row: translation = %q", proj.Entries[0].Translation)
	}
	if proj.Entries[1].Translation != "" {
		t.Errorf("entry-2 was mutated despite invalid row: translation = %q", proj.Entries[1].Translation)
	}
}

// TestImportCSV_DuplicateID verifies that duplicate IDs reject the whole import.
func TestImportCSV_DuplicateID(t *testing.T) {
	proj := makeProject(t)
	path := writeCSV(t, [][]string{
		{"entry-1", "Hello", "سلام", "translated"},
		{"entry-1", "Hello", "سلام ۲", "translated"}, // duplicate
	})
	res, err := ImportCSV(proj, path)
	if err == nil {
		t.Fatal("expected error for duplicate ID, got nil")
	}
	if res.Rejected < 1 {
		t.Errorf("rejected = %d, want >= 1", res.Rejected)
	}
	// No changes applied.
	if proj.Entries[0].Translation != "" {
		t.Errorf("entry-1 was mutated despite duplicate: translation = %q", proj.Entries[0].Translation)
	}
}

// TestImportCSV_UnknownID verifies that an unknown entry ID causes rejection.
func TestImportCSV_UnknownID(t *testing.T) {
	proj := makeProject(t)
	path := writeCSV(t, [][]string{
		{"entry-1", "Hello", "سلام", "translated"},
		{"NONEXISTENT", "X", "Y", "translated"},
	})
	res, err := ImportCSV(proj, path)
	if err == nil {
		t.Fatal("expected error for unknown ID, got nil")
	}
	if res.Rejected < 1 {
		t.Errorf("rejected = %d, want >= 1", res.Rejected)
	}
	// Rollback: entry-1 must not be updated.
	if proj.Entries[0].Translation != "" {
		t.Errorf("entry-1 was mutated despite rollback")
	}
}

// TestImportCSV_PartialRollback confirms that even a single bad row rolls back all rows.
func TestImportCSV_PartialRollback(t *testing.T) {
	proj := makeProject(t)
	path := writeCSV(t, [][]string{
		{"entry-1", "Hello", "سلام", "translated"},     // valid
		{"entry-2", "World", "جهان", "bad_status"},     // invalid — should block entire import
	})
	_, err := ImportCSV(proj, path)
	if err == nil {
		t.Fatal("expected error for bad status, got nil")
	}
	// Both entries must remain unchanged.
	if proj.Entries[0].Translation != "" || proj.Entries[1].Translation != "" {
		t.Errorf("project was partially mutated (rollback failed)")
	}
}

// TestImportCSV_PreservesWhitespace confirms translation whitespace is preserved.
func TestImportCSV_PreservesWhitespace(t *testing.T) {
	proj := makeProject(t)
	// Leading space in translation is intentional (e.g., UI formatting).
	path := writeCSV(t, [][]string{
		{"entry-1", "Hello", " سلام ", "translated"},
	})
	res, err := ImportCSV(proj, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Accepted != 1 {
		t.Fatalf("accepted = %d, want 1", res.Accepted)
	}
	// Whitespace around translation must be preserved (not silently trimmed).
	if proj.Entries[0].Translation != " سلام " {
		t.Errorf("translation whitespace was trimmed: got %q, want %q", proj.Entries[0].Translation, " سلام ")
	}
}
