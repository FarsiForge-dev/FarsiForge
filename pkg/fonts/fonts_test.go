package fonts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSFNT_RealVazirmatnFont(t *testing.T) {
	vazirPath := filepath.Join("..", "..", "build", "bin", "Tools", "vazirmatn", "Vazirmatn-Regular.ttf")
	if !fileExists(vazirPath) {
		t.Skip("Vazirmatn font not available at build path; skipping real font test")
	}

	pf, err := ParseSFNT(vazirPath)
	if err != nil {
		t.Fatalf("failed to parse Vazirmatn font: %v", err)
	}

	if pf.Family != "Vazirmatn" && pf.FullName != "Vazirmatn Regular" {
		t.Logf("Detected font family: %s, fullName: %s", pf.Family, pf.FullName)
	}

	cov := EvaluatePersianCoverage(pf)
	if !cov.SupportsPersian {
		t.Fatalf("Vazirmatn font must support all Persian letters (missing: %v)", cov.MissingRunes)
	}
	if !cov.SupportsPersianDigits {
		t.Fatalf("Vazirmatn font must support Persian digits (missing: %v)", cov.MissingRunes)
	}
	if cov.CoveragePercent < 95.0 {
		t.Errorf("expected >95%% coverage, got %.2f%%", cov.CoveragePercent)
	}
}

func TestEvaluatePersianCoverage_Synthetic(t *testing.T) {
	// 1. Empty font
	emptyCov := EvaluatePersianCoverage(nil)
	if emptyCov.SupportsPersian || emptyCov.CoveragePercent != 0.0 {
		t.Fatalf("expected 0%% coverage for nil font")
	}

	// 2. Font with only basic ASCII
	asciiPf := &ParsedFont{
		SupportedChars: map[rune]bool{
			'A': true, 'B': true, 'C': true,
		},
	}
	asciiCov := EvaluatePersianCoverage(asciiPf)
	if asciiCov.SupportsPersian || asciiCov.SupportsArabic || asciiCov.SupportsPersianDigits {
		t.Fatalf("ASCII font should not report Persian or Arabic support")
	}
	if len(asciiCov.MissingRunes) != len(PersianTestRunes) {
		t.Errorf("expected all test runes missing, got %d missing", len(asciiCov.MissingRunes))
	}

	// 3. Complete Persian font
	completePf := &ParsedFont{
		SupportedChars: make(map[rune]bool),
		HasGSUB:        true,
	}
	for _, r := range PersianTestRunes {
		completePf.SupportedChars[r] = true
	}
	compCov := EvaluatePersianCoverage(completePf)
	if !compCov.SupportsPersian || !compCov.SupportsArabic || !compCov.SupportsPersianDigits {
		t.Fatalf("complete font must report all capabilities true")
	}
	if !compCov.HasOpenTypeShaping {
		t.Fatalf("expected HasOpenTypeShaping to be true when HasGSUB is set")
	}
	if compCov.CoveragePercent != 100.0 {
		t.Errorf("expected 100%% coverage, got %.2f%%", compCov.CoveragePercent)
	}
}

func TestScanGameFonts_InventoryClassification(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Direct font file
	fontDir := filepath.Join(tempDir, "Game", "Content", "Fonts")
	_ = os.MkdirAll(fontDir, 0o755)
	_ = os.WriteFile(filepath.Join(fontDir, "DefaultFont.ttf"), []byte("dummy ttf bytes"), 0o644)

	// 2. BMFont atlas
	uiDir := filepath.Join(tempDir, "Game", "UI")
	_ = os.MkdirAll(uiDir, 0o755)
	_ = os.WriteFile(filepath.Join(uiDir, "HudFont.fnt"), []byte("info face=\"Arial\" size=32"), 0o644)

	// 3. Unreal Font Resource
	_ = os.WriteFile(filepath.Join(fontDir, "SubtitleFont.uasset"), []byte("UE4 serialized font object"), 0o644)

	report, err := ScanGameFonts(tempDir, "unreal")
	if err != nil {
		t.Fatalf("ScanGameFonts failed: %v", err)
	}

	if report.TotalFontsFound != 3 {
		t.Fatalf("expected 3 fonts found, got %d", report.TotalFontsFound)
	}
	if report.DirectFontFiles != 1 {
		t.Errorf("expected 1 direct font file, got %d", report.DirectFontFiles)
	}
	if report.BitmapAtlases != 1 {
		t.Errorf("expected 1 bitmap atlas, got %d", report.BitmapAtlases)
	}
	if report.EngineResources != 1 {
		t.Errorf("expected 1 engine resource, got %d", report.EngineResources)
	}

	// Verify replacement permissions
	for _, item := range report.Items {
		switch item.AssetType {
		case FontAssetDirectFile:
			if !item.CanReplace {
				t.Errorf("direct font file should be replaceable: %s", item.Path)
			}
		case FontAssetBitmapAtlas, FontAssetEngineResource:
			if item.CanReplace {
				t.Errorf("complex font asset should not be marked replaceable without specialized compiler: %s", item.Path)
			}
			if item.UnsupportedReason == "" {
				t.Errorf("expected unsupported reason for %s", item.Path)
			}
		}
	}
}

func TestStageFontReplacement_TransactionalSafety(t *testing.T) {
	tempGame := t.TempDir()
	tempWork := t.TempDir()

	// Create original game font
	origFontPath := filepath.Join(tempGame, "Fonts", "GameFont.ttf")
	_ = os.MkdirAll(filepath.Dir(origFontPath), 0o755)
	_ = os.WriteFile(origFontPath, []byte("original game font content"), 0o644)

	// Find real custom font or skip
	vazirPath := filepath.Join("..", "..", "build", "bin", "Tools", "vazirmatn", "Vazirmatn-Regular.ttf")
	if !fileExists(vazirPath) {
		t.Skip("Vazirmatn font not available; skipping replacement staging test")
	}

	relPath := "Fonts/GameFont.ttf"
	res, err := StageFontReplacement(tempGame, tempWork, relPath, vazirPath)
	if err != nil {
		t.Fatalf("StageFontReplacement failed: %v", err)
	}

	if res.TargetRelPath != relPath {
		t.Errorf("expected relPath %s, got %s", relPath, res.TargetRelPath)
	}
	if res.SourceSHA256 == "" || res.TargetSHA256 == "" {
		t.Fatalf("SHA256 hashes must be populated")
	}
	if !fileExists(res.StagedPath) {
		t.Fatalf("staged font file does not exist at: %s", res.StagedPath)
	}

	// Verify original game installation file was NEVER modified
	origContent, _ := os.ReadFile(origFontPath)
	if string(origContent) != "original game font content" {
		t.Fatalf("CRITICAL SAFETY VIOLATION: game font was mutated in place!")
	}

	// Verify traversal attempt is blocked
	_, err = StageFontReplacement(tempGame, tempWork, "../../Windows/Fonts/Arial.ttf", vazirPath)
	if err == nil {
		t.Fatal("expected traversal attempt to be rejected with error")
	}
}
