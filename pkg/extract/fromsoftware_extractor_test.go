package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
)

func TestFromSoftwareExtractor_LooseAndBinders(t *testing.T) {
	tmp := t.TempDir()

	// 1. Create a loose FMG file: item.fmg
	fmgEntries := []FMGEntry{
		{ID: 10, Text: "Moonlight Greatsword"},
		{ID: 20, Text: "Dark Moon Ring"},
	}
	fmgData, err := EncodeFMG(fmgEntries, 2, true, false)
	if err != nil {
		t.Fatalf("encode loose FMG: %v", err)
	}
	looseFMGPath := filepath.Join(tmp, "item.fmg")
	if err := os.WriteFile(looseFMGPath, fmgData, 0o644); err != nil {
		t.Fatalf("write loose FMG: %v", err)
	}

	// 2. Create a loose msgbnd binder: msg/engus/menu.msgbnd
	menuEntries := []FMGEntry{
		{ID: 1, Text: "Start Game"},
		{ID: 2, Text: "Options"},
	}
	menuFMG, err := EncodeFMG(menuEntries, 2, true, false)
	if err != nil {
		t.Fatalf("encode menu FMG: %v", err)
	}
	bndFiles := []BND4File{
		{ID: 100, Name: "menu.fmg", Data: menuFMG},
	}
	bndData, err := WriteBND4(bndFiles, DefaultBND4Options())
	if err != nil {
		t.Fatalf("write BND4: %v", err)
	}
	msgDir := filepath.Join(tmp, "msg", "engus")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatalf("mkdir msg: %v", err)
	}
	binderPath := filepath.Join(msgDir, "menu.msgbnd")
	if err := os.WriteFile(binderPath, bndData, 0o644); err != nil {
		t.Fatalf("write binder: %v", err)
	}

	extractor := &FromSoftwareExtractor{}
	info := &core.GameInfo{GameRoot: tmp, Engine: "fromsoftware"}
	proj := core.NewProject("TestFS", tmp, "fromsoftware")

	if err := extractor.Extract(context.Background(), info, proj, nil); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	// Should have extracted 4 entries (2 from loose FMG + 2 from loose binder)
	if len(proj.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(proj.Entries))
	}

	// Verify entries have empty Container since they are loose files
	for _, e := range proj.Entries {
		if e.Container != "" {
			t.Errorf("loose entry %q has unexpected container %q", e.File, e.Container)
		}
	}
}
