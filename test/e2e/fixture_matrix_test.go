package e2e

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
	"farsiforge/pkg/extract"
	"farsiforge/pkg/inject"
)

// TextCategory classifies game text for localization workflow.
type TextCategory string

const (
	CategoryDialogue   TextCategory = "dialogue"
	CategorySubtitle   TextCategory = "subtitles"
	CategoryUI         TextCategory = "ui"
	CategoryJournal    TextCategory = "journal"
	CategoryItem       TextCategory = "items"
	CategoryTutorial   TextCategory = "tutorials"
	CategoryQuest      TextCategory = "quests"
)

type FixtureSample struct {
	Category    TextCategory
	AssetFormat string
	Source      string
	Context     string
}

func TestFixtureMatrix(t *testing.T) {
	samples := []FixtureSample{
		// 1. Dialogue & Subtitles
		{Category: CategoryDialogue, AssetFormat: "FromSoftware FMG", Source: "May the flames guide thee, Ashen One.", Context: "Talk"},
		{Category: CategorySubtitle, AssetFormat: "RAGE GXT2", Source: "[Radio] All units be advised, suspect heading north.", Context: "Subtitle"},
		{Category: CategoryDialogue, AssetFormat: "Unreal locres", Source: "Greetings, traveler! What brings you to our tavern?", Context: "Dialogue"},
		
		// 2. UI Elements
		{Category: CategoryUI, AssetFormat: "Godot translation", Source: "Start Game", Context: "Menu"},
		{Category: CategoryUI, AssetFormat: "Unity TextAsset", Source: "Settings", Context: "UI"},
		{Category: CategoryUI, AssetFormat: "FromSoftware FMG", Source: "Inventory", Context: "Menu"},
		{Category: CategoryUI, AssetFormat: "Generic CSV", Source: "Quit to Desktop", Context: "System"},

		// 3. Journals & Notes
		{Category: CategoryJournal, AssetFormat: "Unity MonoBehaviour", Source: "Day 42: The frost has set in. Supplies are dwindling rapidly.", Context: "Note"},
		{Category: CategoryJournal, AssetFormat: "Unreal locres", Source: "Expedition Log: Ancient runes discovered deep within the catacombs.", Context: "Journal"},

		// 4. Items & Descriptions
		{Category: CategoryItem, AssetFormat: "FromSoftware FMG", Source: "Moonveil Katana", Context: "ItemName"},
		{Category: CategoryItem, AssetFormat: "FromSoftware FMG", Source: "A masterpiece blade forged from glintstone, unsheathing beams of lunar light.", Context: "ItemDesc"},
		{Category: CategoryItem, AssetFormat: "RAGE GXT2", Source: "Special Carbine Rifle", Context: "Weapon"},
		{Category: CategoryItem, AssetFormat: "Godot translation", Source: "Healing Potion", Context: "Item"},

		// 5. Tutorials & Quest Text
		{Category: CategoryTutorial, AssetFormat: "Generic JSON", Source: "Press [Shift] to sprint. Manage stamina carefully.", Context: "Tutorial"},
		{Category: CategoryQuest, AssetFormat: "Unreal locres", Source: "Quest: Slay the Dragon of the Lake and return with its scale.", Context: "Quest"},
		{Category: CategoryQuest, AssetFormat: "FromSoftware FMG", Source: "Deliver the letter to Lady Ranni at the Three Sisters.", Context: "Event"},
	}

	// Validate format encoding and decoding for representative format fixtures
	
	// A. FromSoftware FMG Matrix Fixture (Dialogue, Item, Menu)
	fmgEntries := []extract.FMGEntry{
		{ID: 1001, Text: "May the flames guide thee, Ashen One."},
		{ID: 2001, Text: "Inventory"},
		{ID: 3001, Text: "Moonveil Katana"},
		{ID: 3002, Text: "A masterpiece blade forged from glintstone."},
		{ID: 4001, Text: "Deliver the letter to Lady Ranni."},
	}
	fmgData, err := extract.EncodeFMG(fmgEntries, 2, true, false)
	if err != nil {
		t.Fatalf("matrix FMG encode failed: %v", err)
	}
	parsedFMG, err := extract.ParseFMG(fmgData)
	if err != nil {
		t.Fatalf("matrix FMG parse failed: %v", err)
	}
	if len(parsedFMG) != len(fmgEntries) {
		t.Fatalf("matrix FMG count mismatch: got %d, want %d", len(parsedFMG), len(fmgEntries))
	}

	// B. RAGE GXT2 Matrix Fixture (Subtitles, Weapons)
	var gxt2Buf bytes.Buffer
	// Magic 1: "2TXG"
	gxt2Buf.Write([]byte{0x32, 0x54, 0x58, 0x47})
	// Count = 2
	binary.Write(&gxt2Buf, binary.LittleEndian, uint32(2))
	// Entry 0: hash 0x1111, offset 0 (relative to text section)
	binary.Write(&gxt2Buf, binary.LittleEndian, uint32(0x1111))
	binary.Write(&gxt2Buf, binary.LittleEndian, uint32(0))
	// Entry 1: hash 0x2222, offset 30
	binary.Write(&gxt2Buf, binary.LittleEndian, uint32(0x2222))
	binary.Write(&gxt2Buf, binary.LittleEndian, uint32(30))
	// Magic 2: "2TXG"
	gxt2Buf.Write([]byte{0x32, 0x54, 0x58, 0x47})
	// Text length
	binary.Write(&gxt2Buf, binary.LittleEndian, uint32(60))
	// String payloads
	gxt2Buf.WriteString("[Radio] All units be advised.\x00")
	gxt2Buf.WriteString("Special Carbine Rifle\x00")
	parsedGXT2, err := extract.ParseGXT2(gxt2Buf.Bytes())
	if err != nil {
		t.Fatalf("matrix GXT2 parse failed: %v", err)
	}
	if len(parsedGXT2) != 2 {
		t.Fatalf("matrix GXT2 count mismatch: got %d, want 2", len(parsedGXT2))
	}

	// Aggregate counts by category and by asset format
	countsByCategory := make(map[TextCategory]int)
	countsByFormat := make(map[string]int)
	for _, s := range samples {
		countsByCategory[s.Category]++
		countsByFormat[s.AssetFormat]++
	}

	t.Logf("=== Fixture Matrix Validation Report ===")
	t.Logf("Total Sample Fixtures: %d", len(samples))
	t.Logf("Categories Breakdown:")
	for cat, count := range countsByCategory {
		t.Logf("  - %-12s: %d", cat, count)
	}
	t.Logf("Asset Formats Breakdown:")
	for fmtName, count := range countsByFormat {
		t.Logf("  - %-22s: %d", fmtName, count)
	}

	// Verify coverage
	requiredCategories := []TextCategory{
		CategoryDialogue,
		CategorySubtitle,
		CategoryUI,
		CategoryJournal,
		CategoryItem,
		CategoryTutorial,
		CategoryQuest,
	}
	for _, req := range requiredCategories {
		if countsByCategory[req] == 0 {
			t.Errorf("missing fixture coverage for category: %s", req)
		}
	}

	// Verify project integration
	proj := core.NewProject("Fixture Matrix Project", t.TempDir(), "composite")
	for i, s := range samples {
		proj.AddEntry(core.StringEntry{
			ID:      string(s.Category) + "_" + string(rune(i)),
			File:    filepath.ToSlash(string(s.AssetFormat) + "/strings"),
			Path:    string(s.Category),
			Source:  s.Source,
			Context: s.Context,
		})
	}
	if len(proj.Entries) != len(samples) {
		t.Fatalf("project entry count = %d, want %d", len(proj.Entries), len(samples))
	}
}

// TestEngineSupportMatrixDrift verifies that every engine detected by the system
// has an exact, tested capability boundary in pkg/inject, catching registry drift.
func TestEngineSupportMatrixDrift(t *testing.T) {
	injectReg := inject.DefaultRegistry()

	// Supported for injection/write-back
	injectableEngines := []string{"unity", "unreal", "godot", "generic", "custom", "fromsoftware"}
	for _, engine := range injectableEngines {
		inj, err := injectReg.GetInjector(engine)
		if err != nil {
			t.Errorf("expected injector for %q, got error: %v", engine, err)
		}
		if inj == nil {
			t.Errorf("injector for %q is nil", engine)
		}
	}

	// Extract-only engines: must fail with descriptive extract-only message
	extractOnlyEngines := []string{"sage", "goldsrc", "source2", "factorio", "zomboid", "rage"}
	for _, engine := range extractOnlyEngines {
		_, err := injectReg.GetInjector(engine)
		if err == nil {
			t.Errorf("expected extract-only error for %q, got nil", engine)
		} else if !bytes.Contains([]byte(err.Error()), []byte("extract-only")) {
			t.Errorf("expected 'extract-only' in error for %q, got: %v", engine, err)
		}
	}

	// Detection-only engines: must fail with descriptive detection-only message
	detectOnlyEngines := []string{"ue3", "source"}
	for _, engine := range detectOnlyEngines {
		_, err := injectReg.GetInjector(engine)
		if err == nil {
			t.Errorf("expected detection-only error for %q, got nil", engine)
		} else if !bytes.Contains([]byte(err.Error()), []byte("detection-only")) {
			t.Errorf("expected 'detection-only' in error for %q, got: %v", engine, err)
		}
	}
}
