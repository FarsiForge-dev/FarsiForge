package extract

import (
	"bytes"
	"testing"
)

func TestFMGRoundtripV2(t *testing.T) {
	entries := []FMGEntry{
		{ID: 100, Text: "Sword of Light"},
		{ID: 101, Text: "شمشیر نور"},
		{ID: 102, Text: "A blade imbued with ancient radiance."},
		{ID: 200, Text: "خاکستر جنگ"},
		{ID: 205, Text: "Ash of War: Lion's Claw"},
	}

	encoded, err := EncodeFMG(entries, 2, true, false)
	if err != nil {
		t.Fatalf("EncodeFMG failed: %v", err)
	}

	decoded, err := ParseFMG(encoded)
	if err != nil {
		t.Fatalf("ParseFMG failed: %v", err)
	}

	if len(decoded) != len(entries) {
		t.Fatalf("decoded %d entries, want %d", len(decoded), len(entries))
	}

	for i, exp := range entries {
		if decoded[i].ID != exp.ID || decoded[i].Text != exp.Text {
			t.Errorf("entry %d mismatch: got (%d, %q), want (%d, %q)",
				i, decoded[i].ID, decoded[i].Text, exp.ID, exp.Text)
		}
	}
}

func TestFMGRoundtripV1(t *testing.T) {
	entries := []FMGEntry{
		{ID: 10, Text: "Broadsword"},
		{ID: 11, Text: "Longsword"},
		{ID: 12, Text: "Dagger"},
	}

	encoded, err := EncodeFMG(entries, 1, false, false)
	if err != nil {
		t.Fatalf("EncodeFMG v1 failed: %v", err)
	}

	decoded, err := ParseFMG(encoded)
	if err != nil {
		t.Fatalf("ParseFMG v1 failed: %v", err)
	}

	if len(decoded) != len(entries) {
		t.Fatalf("decoded %d entries, want %d", len(decoded), len(entries))
	}

	for i, exp := range entries {
		if decoded[i].ID != exp.ID || decoded[i].Text != exp.Text {
			t.Errorf("v1 entry %d mismatch: got (%d, %q), want (%d, %q)",
				i, decoded[i].ID, decoded[i].Text, exp.ID, exp.Text)
		}
	}
}

func TestBND4Roundtrip(t *testing.T) {
	files := []BND4File{
		{ID: 1, Name: "item.fmg", Data: []byte("Sample FMG data 1")},
		{ID: 2, Name: "menu.fmg", Data: []byte("Sample FMG data 2 with more bytes")},
		{ID: 3, Name: "sub/dialogue.fmg", Data: []byte("Third file payload")},
	}

	opts := DefaultBND4Options()
	bndBytes, err := WriteBND4(files, opts)
	if err != nil {
		t.Fatalf("WriteBND4 failed: %v", err)
	}

	parsed, err := ParseBND4(bndBytes)
	if err != nil {
		t.Fatalf("ParseBND4 failed: %v", err)
	}

	if len(parsed) != len(files) {
		t.Fatalf("parsed %d files, want %d", len(parsed), len(files))
	}

	for i, exp := range files {
		if parsed[i].ID != exp.ID {
			t.Errorf("file %d ID mismatch: got %d, want %d", i, parsed[i].ID, exp.ID)
		}
		if parsed[i].Name != exp.Name {
			t.Errorf("file %d Name mismatch: got %q, want %q", i, parsed[i].Name, exp.Name)
		}
		if !bytes.Equal(parsed[i].Data, exp.Data) {
			t.Errorf("file %d Data mismatch: got %q, want %q", i, parsed[i].Data, exp.Data)
		}
	}
}

func TestBND3Roundtrip(t *testing.T) {
	files := []BND4File{
		{ID: 10, Name: "Item_Title.fmg", Data: []byte("BND3 payload 1")},
		{ID: 20, Name: "Menu_Text.fmg", Data: []byte("BND3 payload 2")},
	}

	opts := DefaultBND3Options()
	bndBytes, err := WriteBND3(files, opts)
	if err != nil {
		t.Fatalf("WriteBND3 failed: %v", err)
	}

	parsed, err := ParseBND3(bndBytes)
	if err != nil {
		t.Fatalf("ParseBND3 failed: %v", err)
	}

	if len(parsed) != len(files) {
		t.Fatalf("parsed %d files, want %d", len(parsed), len(files))
	}

	for i, exp := range files {
		if parsed[i].ID != exp.ID {
			t.Errorf("BND3 file %d ID mismatch: got %d, want %d", i, parsed[i].ID, exp.ID)
		}
		if parsed[i].Name != exp.Name {
			t.Errorf("BND3 file %d Name mismatch: got %q, want %q", i, parsed[i].Name, exp.Name)
		}
		if !bytes.Equal(parsed[i].Data, exp.Data) {
			t.Errorf("BND3 file %d Data mismatch: got %q, want %q", i, parsed[i].Data, exp.Data)
		}
	}
}

func TestDCXRoundtrip(t *testing.T) {
	original := []byte("The Elden Ring was broken by someone or something...")
	dcx, err := CompressDCXDFLT(original)
	if err != nil {
		t.Fatalf("CompressDCXDFLT failed: %v", err)
	}

	decompressed, err := DecompressDCX(dcx)
	if err != nil {
		t.Fatalf("DecompressDCX failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Fatalf("decompressed data mismatch: got %q, want %q", decompressed, original)
	}
}

func TestFromSoftwareFullStackRoundtrip(t *testing.T) {
	// Full composite pipeline:
	// FMG Entries -> EncodeFMG -> BND4 -> CompressDCXDFLT -> DecompressDCX -> ParseBND4 -> ParseFMG
	itemEntries := []FMGEntry{
		{ID: 1000, Text: "Ranni's Dark Moon"},
		{ID: 1001, Text: "ماه تاریک رانی"},
	}
	menuEntries := []FMGEntry{
		{ID: 1, Text: "Continue"},
		{ID: 2, Text: "Load Game"},
		{ID: 3, Text: "ادامه بازی"},
	}

	itemFMG, err := EncodeFMG(itemEntries, 2, true, false)
	if err != nil {
		t.Fatalf("encode item FMG: %v", err)
	}
	menuFMG, err := EncodeFMG(menuEntries, 2, true, false)
	if err != nil {
		t.Fatalf("encode menu FMG: %v", err)
	}

	bndFiles := []BND4File{
		{ID: 1, Name: "item.fmg", Data: itemFMG},
		{ID: 2, Name: "menu.fmg", Data: menuFMG},
	}

	bndBytes, err := WriteBND4(bndFiles, DefaultBND4Options())
	if err != nil {
		t.Fatalf("write BND4: %v", err)
	}

	dcxBytes, err := CompressDCXDFLT(bndBytes)
	if err != nil {
		t.Fatalf("compress DCX: %v", err)
	}

	// Decompress DCX
	unpackedBnd, err := DecompressDCX(dcxBytes)
	if err != nil {
		t.Fatalf("decompress DCX: %v", err)
	}

	// Parse BND4
	unpackedFiles, err := ParseBND4(unpackedBnd)
	if err != nil {
		t.Fatalf("parse BND4: %v", err)
	}

	if len(unpackedFiles) != 2 {
		t.Fatalf("expected 2 BND files, got %d", len(unpackedFiles))
	}

	// Parse unpacked item FMG
	decodedItems, err := ParseFMG(unpackedFiles[0].Data)
	if err != nil {
		t.Fatalf("parse item FMG: %v", err)
	}
	if len(decodedItems) != 2 || decodedItems[1].Text != "ماه تاریک رانی" {
		t.Fatalf("unexpected item entries: %+v", decodedItems)
	}

	// Parse unpacked menu FMG
	decodedMenu, err := ParseFMG(unpackedFiles[1].Data)
	if err != nil {
		t.Fatalf("parse menu FMG: %v", err)
	}
	if len(decodedMenu) != 3 || decodedMenu[2].Text != "ادامه بازی" {
		t.Fatalf("unexpected menu entries: %+v", decodedMenu)
	}
}
