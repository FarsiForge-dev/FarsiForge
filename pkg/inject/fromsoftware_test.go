package inject

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
	"farsiforge/pkg/extract"
)

func TestFromSoftwareInjectorLooseFMG(t *testing.T) {
	tempDir := t.TempDir()
	gameRoot := filepath.Join(tempDir, "game")
	if err := os.MkdirAll(gameRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create initial FMG file on disk
	initialEntries := []extract.FMGEntry{
		{ID: 100, Text: "Sword of Night"},
		{ID: 101, Text: "Armor of Solitude"},
	}
	fmgBytes, err := extract.EncodeFMG(initialEntries, 2, true, false)
	if err != nil {
		t.Fatal(err)
	}

	fmgRel := "item.fmg"
	fmgPath := filepath.Join(gameRoot, fmgRel)
	if err := os.WriteFile(fmgPath, fmgBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	proj := core.NewProject("test-fs", tempDir, "fromsoftware")
	proj.AddEntry(core.StringEntry{
		ID:          "entry-1",
		File:        fmgRel,
		Path:        "100",
		Source:      "Sword of Night",
		Translation: "شمشیر شب",
		Status:      core.StatusTranslated,
	})

	injector := &FromSoftwareInjector{}
	res, err := injector.Inject(context.Background(), &core.GameInfo{GameRoot: gameRoot, Engine: "fromsoftware"}, proj, nil, core.PersianOptions{})
	if err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected injection errors: %v", res.Errors)
	}
	if len(res.ModifiedFiles) != 1 || res.ModifiedFiles[0] != fmgRel {
		t.Fatalf("modified files = %v, want [%s]", res.ModifiedFiles, fmgRel)
	}
	if res.StringCount != 1 {
		t.Fatalf("string count = %d, want 1", res.StringCount)
	}

	// Verify modified FMG on disk
	modifiedBytes, err := os.ReadFile(fmgPath)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := extract.ParseFMG(modifiedBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Text != "شمشیر شب" {
		t.Errorf("row 0 text = %q, want 'شمشیر شب'", rows[0].Text)
	}
}

func TestFromSoftwareInjectorMsgbndDCX(t *testing.T) {
	tempDir := t.TempDir()
	gameRoot := filepath.Join(tempDir, "game")
	msgDir := filepath.Join(gameRoot, "msg", "engus")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create initial FMG inside a BND4 wrapped in DCX
	initialItemFMG, err := extract.EncodeFMG([]extract.FMGEntry{
		{ID: 200, Text: "Moonveil Katana"},
	}, 2, true, false)
	if err != nil {
		t.Fatal(err)
	}

	bndFiles := []extract.BND4File{
		{ID: 1, Name: "item.fmg", Data: initialItemFMG},
	}
	bndBytes, err := extract.WriteBND4(bndFiles, extract.DefaultBND4Options())
	if err != nil {
		t.Fatal(err)
	}
	dcxBytes, err := extract.CompressDCXDFLT(bndBytes)
	if err != nil {
		t.Fatal(err)
	}

	containerRel := "msg/engus/item.msgbnd.dcx"
	containerPath := filepath.Join(gameRoot, filepath.FromSlash(containerRel))
	if err := os.WriteFile(containerPath, dcxBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	proj := core.NewProject("test-fs-dcx", tempDir, "fromsoftware")
	proj.AddEntry(core.StringEntry{
		ID:          "entry-katana",
		File:        containerRel + "/item.fmg",
		Path:        "200",
		Source:      "Moonveil Katana",
		Translation: "کاتانای مون‌ویل",
		Status:      core.StatusTranslated,
	})

	injector := &FromSoftwareInjector{}
	res, err := injector.Inject(context.Background(), &core.GameInfo{GameRoot: gameRoot, Engine: "fromsoftware"}, proj, nil, core.PersianOptions{})
	if err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected injection errors: %v", res.Errors)
	}
	if len(res.ModifiedFiles) != 1 || res.ModifiedFiles[0] != containerRel {
		t.Fatalf("modified files = %v, want [%s]", res.ModifiedFiles, containerRel)
	}

	// Verify on disk by unpacking DCX -> BND4 -> FMG
	reloadedDCX, err := os.ReadFile(containerPath)
	if err != nil {
		t.Fatal(err)
	}
	unpackedBND, err := extract.DecompressDCX(reloadedDCX)
	if err != nil {
		t.Fatal(err)
	}
	unpackedFiles, err := extract.ParseBND4(unpackedBND)
	if err != nil {
		t.Fatal(err)
	}
	if len(unpackedFiles) != 1 {
		t.Fatalf("expected 1 file in BND, got %d", len(unpackedFiles))
	}
	fmgRows, err := extract.ParseFMG(unpackedFiles[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(fmgRows) != 1 || fmgRows[0].Text != "کاتانای مون‌ویل" {
		t.Fatalf("unexpected injected text: %+v", fmgRows)
	}
}
