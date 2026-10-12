package extract_test

import (
	"encoding/binary"
	"strings"
	"testing"

	"farsiforge/pkg/extract"
)

// ── GXT2 (RAGE localization tables) ──────────────────────────────────────

func TestParseGXT2Valid(t *testing.T) {
	t.Parallel()
	// GXT2 format:
	// - Header (8 bytes): "GXT2" + entry count
	// - Entries (n * 8 bytes): each = hash(4) + rel_offset(4)
	// - Text section header (8 bytes): "GXT2" + text_section_size
	// - Text data (null-terminated strings)
	//
	// For 1 entry:
	//   header: 0-7
	//   entry:  8-15
	//   text section header: 16-23
	//   textStart = 24
	//   text at 24: "Hello\0"
	data := make([]byte, 24+6)                 // header(8) + entry(8) + text_section_header(8) + text(6)
	copy(data, "GXT2")                         // magic at 0-3
	binary.LittleEndian.PutUint32(data[4:], 1) // entry count = 1
	// Entry at 8-15: hash=0x12345678, rel_offset=0
	binary.LittleEndian.PutUint32(data[8:], 0x12345678)
	binary.LittleEndian.PutUint32(data[12:], 0)
	// Text section header at 16-23: "GXT2" + size=6
	copy(data[16:], "GXT2")
	binary.LittleEndian.PutUint32(data[20:], 6)
	// Text at textStart=24: "Hello\0"
	copy(data[24:], "Hello\x00")

	got, err := extract.ParseGXT2(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "Hello" || got[0].Hash != 0x12345678 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestParseGXT2MultipleEntries(t *testing.T) {
	t.Parallel()
	// 2 entries:
	//   header: 0-7
	//   entries: 8-23 (2 * 8 = 16 bytes)
	//   text section header: 24-31
	//   textStart = 32
	//   entry 0 rel=0 -> pos=32: "Hello\0" (6 bytes)
	//   entry 1 rel=6 -> pos=38: "World\0" (6 bytes)
	//   total text size = 12
	data := make([]byte, 8+16+8+12)
	copy(data, "GXT2")
	binary.LittleEndian.PutUint32(data[4:], 2) // 2 entries
	// Entry 0 at 8-15: hash=0x11111111, rel_offset=0
	binary.LittleEndian.PutUint32(data[8:], 0x11111111)
	binary.LittleEndian.PutUint32(data[12:], 0)
	// Entry 1 at 16-23: hash=0x22222222, rel_offset=6
	binary.LittleEndian.PutUint32(data[16:], 0x22222222)
	binary.LittleEndian.PutUint32(data[20:], 6)
	// Text section header at 24-31: "GXT2" + size=12
	copy(data[24:], "GXT2")
	binary.LittleEndian.PutUint32(data[28:], 12)
	// Text at textStart=32
	copy(data[32:], "Hello\x00") // pos 32 (rel 0)
	copy(data[38:], "World\x00") // pos 38 (rel 6)

	got, err := extract.ParseGXT2(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %#v", len(got), got)
	}
	if got[0].Hash != 0x11111111 || got[0].Text != "Hello" {
		t.Fatalf("entry 0: %#v", got[0])
	}
	if got[1].Hash != 0x22222222 || got[1].Text != "World" {
		t.Fatalf("entry 1: %#v", got[1])
	}
}

// ── JOAAT hash ───────────────────────────────────────────────────────────

func TestJOAATHashKnownVectors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want uint32
	}{
		{"", 0x00000000},
		{"a", 0xca2e9442},
		{"hello", 0xc8fd181b},
		{"Hello World!", 0x9bd919d3},
		{"مرحبا", 0xb79bd3a1},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := extract.JOAATHash(tc.in); got != tc.want {
				t.Errorf("JOAATHash(%q) = 0x%08x, want 0x%08x", tc.in, got, tc.want)
			}
		})
	}
}

// ── ERPathHash (FromSoftware path hashing) ───────────────────────────────

func TestERPathHash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want uint64
	}{
		{"/msg/engus/item.msgbnd.dcx", 0x0},
		{"/msg/engus/menu.msgbnd.dcx", 0x0},
	}
	// Just verify deterministic output
	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			h1 := extract.ERPathHash(tc.path)
			h2 := extract.ERPathHash(tc.path)
			if h1 != h2 {
				t.Errorf("ERPathHash not deterministic: %d vs %d", h1, h2)
			}
			if h1 == 0 {
				t.Errorf("ERPathHash should not be zero for %q", tc.path)
			}
		})
	}
}

// ── DCX decompression ────────────────────────────────────────────────────

func TestDecompressDCXValidZlib(t *testing.T) {
	t.Parallel()
	// Build a valid DCX file with embedded zlib stream
	// DCX header needs to be at least 16 bytes, contain "DCX" in first 64 bytes,
	// and have a valid zlib stream (0x78 0x9c = default compression)
	// zlib compressed "hello": 0x78 0x9c + compressed data + adler32
	dcxHeader := []byte{
		'D', 'C', 'X', 0, // DCX signature
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0, // 16 bytes total so far
	}
	zlibStream := []byte{0x78, 0x9c, 0xca, 0x48, 0xcd, 0xc9, 0xc9, 0x07, 0x04, 0x00, 0x00, 0xff, 0xff, 0x06, 0x2c, 0x02, 0x15}
	data := append(dcxHeader, zlibStream...)
	out, err := extract.DecompressDCX(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello" {
		t.Fatalf("got %q, want %q", string(out), "hello")
	}
}

func TestDecompressDCXInvalidSignature(t *testing.T) {
	t.Parallel()
	// Data >= 16 bytes but no "DCX" signature in first 64 bytes
	data := make([]byte, 20)
	copy(data, "NOT A VALID HEADER")
	_, err := extract.DecompressDCX(data)
	if err == nil {
		t.Fatal("expected error for invalid DCX signature")
	}
	if !strings.Contains(err.Error(), "invalid DCX signature") {
		t.Errorf("error should mention DCX signature: %v", err)
	}
}

func TestDecompressDCXTruncated(t *testing.T) {
	t.Parallel()
	// Data < 16 bytes
	data := []byte("DCX")
	_, err := extract.DecompressDCX(data)
	if err == nil {
		t.Fatal("expected error for truncated DCX")
	}
	if !strings.Contains(err.Error(), "DCX data is truncated") {
		t.Errorf("error should mention truncated: %v", err)
	}
}

// ── FMG parsing ──────────────────────────────────────────────────────────

func TestParseFMGValidV2Wide(t *testing.T) {
	t.Parallel()
	// From existing test in family_extractors_test.go
	b := make([]byte, 70)
	b[0] = 0
	b[1] = 0
	b[2] = 2 // version 2 (wide)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)))
	b[8] = 1
	binary.LittleEndian.PutUint32(b[12:], 1) // groupCount
	binary.LittleEndian.PutUint32(b[16:], 1) // stringCount
	b[20] = 0xff
	binary.LittleEndian.PutUint64(b[24:], 56)
	binary.LittleEndian.PutUint32(b[40:], 0)
	binary.LittleEndian.PutUint32(b[44:], 42)
	binary.LittleEndian.PutUint32(b[48:], 42)
	binary.LittleEndian.PutUint64(b[56:], 64)
	copy(b[64:], []byte{'H', 0, 'i', 0, 0, 0})

	got, err := extract.ParseFMG(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 42 || got[0].Text != "Hi" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestParseFMGInvalidSignature(t *testing.T) {
	t.Parallel()
	data := make([]byte, 24)
	data[0] = 1 // wrong signature
	_, err := extract.ParseFMG(data)
	if err == nil {
		t.Fatal("expected error for invalid FMG signature")
	}
	if !strings.Contains(err.Error(), "invalid FMG signature") {
		t.Errorf("error should mention signature: %v", err)
	}
}

func TestParseFMGTruncated(t *testing.T) {
	t.Parallel()
	data := make([]byte, 10)
	_, err := extract.ParseFMG(data)
	if err == nil {
		t.Fatal("expected error for truncated FMG")
	}
	if !strings.Contains(err.Error(), "FMG header is truncated") {
		t.Errorf("error should mention truncated header: %v", err)
	}
}

// ── BND4 parsing ─────────────────────────────────────────────────────────

func TestParseBND4InvalidMagic(t *testing.T) {
	t.Parallel()
	data := make([]byte, 64)
	copy(data, "BND5") // wrong magic
	_, err := extract.ParseBND4(data)
	if err == nil {
		t.Fatal("expected error for invalid BND4 magic")
	}
	if !strings.Contains(err.Error(), "invalid BND4 header") {
		t.Errorf("error should mention BND4 header: %v", err)
	}
}

func TestParseBND4Truncated(t *testing.T) {
	t.Parallel()
	data := make([]byte, 10)
	_, err := extract.ParseBND4(data)
	if err == nil {
		t.Fatal("expected error for truncated BND4")
	}
}

// ── ERBHD5 / BHD5 decryption ─────────────────────────────────────────────

func TestDecryptERData0BHDWrongMagic(t *testing.T) {
	t.Parallel()
	data := make([]byte, 256) // must be multiple of 256 for RSA blocks
	copy(data, "BHD4")        // wrong magic after decryption
	_, err := extract.DecryptERData0BHD(data)
	if err == nil {
		t.Fatal("expected error for wrong BHD5 magic")
	}
}

func TestParseERBHD5InvalidMagic(t *testing.T) {
	t.Parallel()
	data := make([]byte, 28)
	copy(data, "BHD4") // wrong magic
	_, err := extract.ParseERBHD5(data)
	if err == nil {
		t.Fatal("expected error for invalid BHD5 magic")
	}
	if !strings.Contains(err.Error(), "invalid BHD5 magic") {
		t.Errorf("error should mention BHD5 magic: %v", err)
	}
}

// ── Registry and extractor dispatch ──────────────────────────────────────

func TestDefaultRegistryHasExpectedEngines(t *testing.T) {
	t.Parallel()
	reg := extract.DefaultRegistry()
	expected := []string{
		"unity",
		"unreal",
		"godot",
		"goldsrc",
		"source2",
		"factorio",
		"zomboid",
		"sage",
		"rage",
		"fromsoftware",
		"generic",
	}
	for _, eng := range expected {
		tc := eng
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			ext, err := reg.GetExtractor(tc)
			if err != nil {
				t.Fatalf("GetExtractor(%q): %v", tc, err)
			}
			if ext.SupportedEngine() != tc {
				t.Fatalf("extractor SupportedEngine() = %q, want %q", ext.SupportedEngine(), tc)
			}
		})
	}
}

func TestDefaultRegistryUnknownEngineFallsBackToGeneric(t *testing.T) {
	t.Parallel()
	reg := extract.DefaultRegistry()
	ext, err := reg.GetExtractor("unknown-engine-xyz")
	if err != nil {
		t.Fatalf("should fallback to generic: %v", err)
	}
	if ext.SupportedEngine() != "generic" {
		t.Fatalf("fallback extractor = %q, want generic", ext.SupportedEngine())
	}
}

func TestDefaultRegistryEmptyEngineFallsBackToGeneric(t *testing.T) {
	t.Parallel()
	reg := extract.DefaultRegistry()
	ext, err := reg.GetExtractor("")
	if err != nil {
		t.Fatalf("empty engine should fall back to generic: %v", err)
	}
	if ext.SupportedEngine() != "generic" {
		t.Fatalf("empty engine fallback = %q, want generic", ext.SupportedEngine())
	}
}

// ── Extractor capabilities ───────────────────────────────────────────────

func TestExtractorCapabilities(t *testing.T) {
	t.Parallel()
	reg := extract.DefaultRegistry()
	tests := []struct {
		engine    string
		needsTool bool
		toolName  string
	}{
		{"unity", true, "python"},
		{"unreal", true, "fftools"},
		{"godot", true, "fftools"},
		{"goldsrc", false, ""},
		{"source2", false, ""},
		{"factorio", false, ""},
		{"zomboid", false, ""},
		{"sage", false, ""},
		{"rage", false, ""},
		{"fromsoftware", false, ""},
		{"generic", false, ""},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.engine, func(t *testing.T) {
			t.Parallel()
			ext, err := reg.GetExtractor(tc.engine)
			if err != nil {
				t.Fatal(err)
			}
			caps := ext.Capabilities()
			if caps.NeedsExternalTool != tc.needsTool {
				t.Errorf("NeedsExternalTool = %v, want %v", caps.NeedsExternalTool, tc.needsTool)
			}
			if tc.needsTool && caps.ToolName != tc.toolName {
				t.Errorf("ToolName = %q, want %q", caps.ToolName, tc.toolName)
			}
		})
	}
}

// ── Error path: missing file / unsupported format / corrupt header ───────

func TestParseFMGFileSizeMismatch(t *testing.T) {
	t.Parallel()
	b := make([]byte, 24)
	b[0] = 0
	b[1] = 0
	b[2] = 1
	binary.LittleEndian.PutUint32(b[4:], 1000) // fileSize much larger than actual data
	b[8] = 1
	binary.LittleEndian.PutUint32(b[12:], 1)
	binary.LittleEndian.PutUint32(b[16:], 1)
	_, err := extract.ParseFMG(b)
	if err == nil {
		t.Fatal("expected error for file size mismatch")
	}
	if !strings.Contains(err.Error(), "invalid FMG file size") {
		t.Errorf("error should mention file size: %v", err)
	}
}

func TestParseGXT2EntryCountExceedsBounds(t *testing.T) {
	t.Parallel()
	data := make([]byte, 16)
	copy(data, "GXT2")
	binary.LittleEndian.PutUint32(data[4:], 1000000) // huge entry count
	_, err := extract.ParseGXT2(data)
	if err == nil {
		t.Fatal("expected error for entry count exceeding bounds")
	}
	if !strings.Contains(err.Error(), "exceeds file bounds") {
		t.Errorf("error should mention bounds: %v", err)
	}
}

func TestParseBND4InvalidCount(t *testing.T) {
	t.Parallel()
	data := make([]byte, 64)
	copy(data, "BND4")
	binary.LittleEndian.PutUint32(data[12:16], 1000000) // huge file count
	_, err := extract.ParseBND4(data)
	if err == nil {
		t.Fatal("expected error for invalid BND4 count")
	}
	if !strings.Contains(err.Error(), "invalid BND4 file-header table") {
		t.Errorf("error should mention invalid table: %v", err)
	}
}

// ── Deterministic output tests ───────────────────────────────────────────

func TestParseGXT2Deterministic(t *testing.T) {
	t.Parallel()
	// Build valid GXT2 data
	data := make([]byte, 24+15)
	copy(data, "GXT2")
	binary.LittleEndian.PutUint32(data[4:], 1)
	binary.LittleEndian.PutUint32(data[8:], 0xDEADBEEF)
	binary.LittleEndian.PutUint32(data[12:], 0)
	copy(data[16:], "GXT2")
	binary.LittleEndian.PutUint32(data[20:], 15)
	copy(data[24:], "Deterministic\x00")

	for i := 0; i < 10; i++ {
		got, err := extract.ParseGXT2(data)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Hash != 0xDEADBEEF || got[0].Text != "Deterministic" {
			t.Fatalf("run %d: non-deterministic output: %#v", i, got)
		}
	}
}

func TestParseFMGDeterministic(t *testing.T) {
	t.Parallel()
	b := make([]byte, 70)
	b[0] = 0
	b[1] = 0
	b[2] = 2
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)))
	b[8] = 1
	binary.LittleEndian.PutUint32(b[12:], 1)
	binary.LittleEndian.PutUint32(b[16:], 1)
	b[20] = 0xff
	binary.LittleEndian.PutUint64(b[24:], 56)
	binary.LittleEndian.PutUint32(b[40:], 0)
	binary.LittleEndian.PutUint32(b[44:], 42)
	binary.LittleEndian.PutUint32(b[48:], 42)
	binary.LittleEndian.PutUint64(b[56:], 64)
	copy(b[64:], []byte{'H', 0, 'i', 0, 0, 0})

	for i := 0; i < 10; i++ {
		got, err := extract.ParseFMG(b)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != 42 || got[0].Text != "Hi" {
			t.Fatalf("run %d: non-deterministic output: %#v", i, got)
		}
	}
}

func TestJOAATHashDeterministic(t *testing.T) {
	t.Parallel()
	input := "The quick brown fox jumps over the lazy dog."
	for i := 0; i < 100; i++ {
		if got := extract.JOAATHash(input); got != extract.JOAATHash(input) {
			t.Fatalf("JOAATHash not deterministic at iteration %d", i)
		}
	}
}

func TestERPathHashDeterministic(t *testing.T) {
	t.Parallel()
	input := "/msg/engus/item.msgbnd.dcx"
	for i := 0; i < 100; i++ {
		if got := extract.ERPathHash(input); got != extract.ERPathHash(input) {
			t.Fatalf("ERPathHash not deterministic at iteration %d", i)
		}
	}
}
