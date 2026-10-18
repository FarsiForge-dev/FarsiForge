package fonts

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"farsiforge/pkg/tools"
)

// ScanGameFonts inspects the game installation directory and identifies all direct font files,
// engine font resources, and bitmap glyph atlases.
func ScanGameFonts(gameRoot, engine string) (*FontInventoryReport, error) {
	absRoot, err := filepath.Abs(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve game root: %w", err)
	}

	report := &FontInventoryReport{
		GameRoot: absRoot,
		Engine:   engine,
		Items:    []FontInventoryItem{},
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // Skip inaccessible directories without failing entire scan
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == ".git" || name == ".ffbuild" || name == "backup" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		lowerRel := strings.ToLower(rel)
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size := info.Size()

		// 1. Direct loose font files (.ttf, .otf, .ttc, .woff)
		if strings.HasSuffix(lowerRel, ".ttf") || strings.HasSuffix(lowerRel, ".otf") || strings.HasSuffix(lowerRel, ".ttc") {
			item := FontInventoryItem{
				Path:           rel,
				Engine:         engine,
				AssetType:      FontAssetDirectFile,
				DetectedFormat: strings.ToUpper(strings.TrimPrefix(filepath.Ext(lowerRel), ".")),
				Confidence:     0.95,
				CanExport:      true,
				CanReplace:     true,
				FileSize:       size,
			}

			// Parse SFNT details and glyph coverage
			if pf, err := ParseSFNT(path); err == nil {
				item.Family = pf.Family
				item.Style = pf.Subfamily
				cov := EvaluatePersianCoverage(pf)
				item.GlyphCoverage = &cov
				if cov.SupportsPersian {
					report.PersianReadyCount++
				}
			}

			report.DirectFontFiles++
			report.Items = append(report.Items, item)
			return nil
		}

		// 2. Bitmap / BMFont glyph atlases (.fnt)
		if strings.HasSuffix(lowerRel, ".fnt") {
			item := FontInventoryItem{
				Path:              rel,
				Engine:            engine,
				AssetType:         FontAssetBitmapAtlas,
				DetectedFormat:    "BMFont (AngelCode)",
				Confidence:        0.90,
				CanExport:         true,
				CanReplace:        false,
				UnsupportedReason: "Bitmap glyph atlas requires texture atlas repacking and metric re-encoding",
				FileSize:          size,
			}
			report.BitmapAtlases++
			report.Items = append(report.Items, item)
			return nil
		}

		// 3. Engine-specific compiled font resources
		switch strings.ToLower(engine) {
		case "unreal", "ue4", "ue5":
			if strings.HasSuffix(lowerRel, ".uasset") && (strings.Contains(lowerRel, "/font") || strings.Contains(lowerRel, "font")) {
				item := FontInventoryItem{
					Path:              rel,
					Engine:            engine,
					AssetType:         FontAssetEngineResource,
					DetectedFormat:    "Unreal Font/FontFace (.uasset)",
					Confidence:        0.85,
					CanExport:         false,
					CanReplace:        false,
					UnsupportedReason: "Compiled Unreal font asset requires serialized font cooker or loose override",
					FileSize:          size,
				}
				report.EngineResources++
				report.Items = append(report.Items, item)
			}
		case "unity":
			if strings.HasSuffix(lowerRel, ".fontsettings") || (strings.Contains(lowerRel, "font") && strings.HasSuffix(lowerRel, ".assets")) {
				item := FontInventoryItem{
					Path:              rel,
					Engine:            engine,
					AssetType:         FontAssetEngineResource,
					DetectedFormat:    "Unity Font / TextMeshPro",
					Confidence:        0.80,
					CanExport:         false,
					CanReplace:        false,
					UnsupportedReason: "Unity font asset requires asset bundle repacking with UnityPy",
					FileSize:          size,
				}
				report.EngineResources++
				report.Items = append(report.Items, item)
			}
		case "godot":
			if strings.HasSuffix(lowerRel, ".tres") && strings.Contains(lowerRel, "font") {
				item := FontInventoryItem{
					Path:              rel,
					Engine:            engine,
					AssetType:         FontAssetEngineResource,
					DetectedFormat:    "Godot FontFile Resource (.tres)",
					Confidence:        0.80,
					CanExport:         true,
					CanReplace:        false,
					UnsupportedReason: "Godot resource font requires PCK repacking",
					FileSize:          size,
				}
				report.EngineResources++
				report.Items = append(report.Items, item)
			}
		case "fromsoftware":
			if strings.Contains(lowerRel, "/font") || strings.Contains(lowerRel, "font_") {
				item := FontInventoryItem{
					Path:              rel,
					Engine:            engine,
					AssetType:         FontAssetCustomArchive,
					DetectedFormat:    "FromSoftware Glyph Atlas",
					Confidence:        0.75,
					CanExport:         false,
					CanReplace:        false,
					UnsupportedReason: "Custom FromSoftware glyph texture atlas; safe writer not available",
					FileSize:          size,
				}
				report.EngineResources++
				report.Items = append(report.Items, item)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk game directory: %w", err)
	}

	// For Unity games, scan compiled Font assets inside .assets and .bundle files
	if strings.ToLower(engine) == "unity" {
		unityFonts, _ := scanUnityCompiledFonts(absRoot)
		for _, uf := range unityFonts {
			report.Items = append(report.Items, uf)
			report.EngineResources++
		}
	}

	report.TotalFontsFound = len(report.Items)

	// Build human-readable diagnostic summary and actionable guidance
	if report.TotalFontsFound == 0 {
		report.DiagnosticSummary = "هیچ فایل یا دارایی فونت شناخته‌شده‌ای در پوشه بازی یافت نشد."
		report.RecommendedAction = "بررسی کنید که آیا فونت‌های بازی درون بسته‌های آرشیو فشرده (.pak, .bdt, .pck) قرار دارند یا خیر."
	} else if report.DirectFontFiles > 0 {
		report.DiagnosticSummary = fmt.Sprintf("%d فونت استاندارد مستقیم (TTF/OTF) یافت شد (%d مورد پشتیبانی کامل فارسی دارد).", report.DirectFontFiles, report.PersianReadyCount)
		report.RecommendedAction = "می‌توانید فونت‌های استاندارد مستقیم را با فونت فارسی دلخواه (مانند وزیرمتن) جایگزین کنید."
	} else {
		report.DiagnosticSummary = fmt.Sprintf("%d دارایی فونت کامپایل‌شده یا اطلس تصویری شناسایی شد.", report.TotalFontsFound)
		report.RecommendedAction = "فونت‌های شناسایی‌شده در فایل‌های کامپایل‌شده بازی آماده استخراج یا جایگزینی با فونت فارسی هستند."
	}

	return report, nil
}

type unityScannedFont struct {
	Path       string `json:"path"`
	File       string `json:"file"`
	PathID     int64  `json:"path_id"`
	Name       string `json:"name"`
	SizeBytes  int64  `json:"size_bytes"`
	Format     string `json:"format"`
	CanReplace bool   `json:"can_replace"`
	CanExport  bool   `json:"can_export"`
}

func scanUnityCompiledFonts(gameRoot string) ([]FontInventoryItem, error) {
	reg, err := tools.NewRegistry("")
	if err != nil {
		return nil, err
	}
	py := reg.GetPython()
	if py == "" {
		return nil, fmt.Errorf("python required")
	}

	tempDir, err := os.MkdirTemp("", "farsiforge_fonts_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	scriptPath, err := tools.WriteScript("inspect_asset.py", tempDir)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := tools.RunSilent(ctx, tempDir, py, scriptPath, "--scan-fonts", gameRoot)
	if err != nil {
		return nil, err
	}

	var parsed []unityScannedFont
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		return nil, err
	}

	var items []FontInventoryItem
	for _, f := range parsed {
		items = append(items, FontInventoryItem{
			Path:           f.Path,
			Engine:         "unity",
			Family:         f.Name,
			AssetType:      FontAssetEngineResource,
			DetectedFormat: "Unity Font (" + f.Format + ")",
			Confidence:     0.95,
			CanExport:      f.CanExport,
			CanReplace:     f.CanReplace,
			FileSize:       f.SizeBytes,
		})
	}
	return items, nil
}

