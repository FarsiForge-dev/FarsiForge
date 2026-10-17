package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/assetbrowser"
	"farsiforge/pkg/detection"
	"farsiforge/pkg/fonts"
)

func TestRealGames_SanityAudit(t *testing.T) {
	gamesRoot := `D:\games`
	if fi, err := os.Stat(gamesRoot); err != nil || !fi.IsDir() {
		t.Skip("D:\\games directory not found; skipping real games test")
	}

	reg := detection.DefaultRegistry()

	games := []struct {
		folder         string
		expectedEngine string
	}{
		{"ELDEN RING", "fromsoftware"},
		{"Sekiro Shadows Die Twice GOTY Edition", "fromsoftware"},
		{"Factorio", "factorio"},
		{"Project.Zomboid", "zomboid"},
		{"Raft", "unity"},
		{"CnC Zero Hour - R2P Edition", "sage"},
		{"Command and Conquer Red Alert 3", "sage"},
	}

	for _, g := range games {
		gamePath := filepath.Join(gamesRoot, g.folder)
		if fi, err := os.Stat(gamePath); err != nil || !fi.IsDir() {
			t.Logf("Game folder %s not present; skipping", g.folder)
			continue
		}

		t.Run(g.folder, func(t *testing.T) {
			// 1. Detection
			res, err := reg.Detect(gamePath)
			if err != nil {
				t.Fatalf("detection failed for %s: %v", g.folder, err)
			}
			t.Logf("[%s] Detected Engine: %s (Confidence: %.2f)", g.folder, res.Engine, res.Confidence)

			// 2. Asset Browser
			view, err := assetbrowser.BrowseGameAssets(gamePath, res.Engine)
			if err != nil {
				t.Fatalf("BrowseGameAssets failed for %s: %v", g.folder, err)
			}
			t.Logf("[%s] Asset Browser: %d files, %d containers, %d parsed resources, %d text records",
				g.folder, view.TotalFiles, view.TotalContainers, view.TotalParsedResources, view.TotalTextRecords)

			// 3. Font Inventory
			fontRep, err := fonts.ScanGameFonts(gamePath, res.Engine)
			if err != nil {
				t.Fatalf("ScanGameFonts failed for %s: %v", g.folder, err)
			}
			t.Logf("[%s] Font Scan: %d fonts found (%d direct, %d resources, %d atlases, %d Persian-ready)",
				g.folder, fontRep.TotalFontsFound, fontRep.DirectFontFiles, fontRep.EngineResources, fontRep.BitmapAtlases, fontRep.PersianReadyCount)
		})
	}
}
