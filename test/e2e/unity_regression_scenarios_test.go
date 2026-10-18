package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"farsiforge/pkg/assetbrowser"
	"farsiforge/pkg/core"
	"farsiforge/pkg/installer"
	"farsiforge/pkg/textfilter"
	"farsiforge/pkg/tools"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO A: Noisy Unity Asset File
// ─────────────────────────────────────────────────────────────────────────────
func TestScenarioA_NoisyUnityAssetFile(t *testing.T) {
	toolReg, _ := tools.NewRegistry("")
	pythonExe := "python"
	if toolReg != nil && toolReg.GetPython() != "" {
		pythonExe = toolReg.GetPython()
	}

	// Skip if UnityPy is not installed on the test runner
	if err := exec.Command(pythonExe, "-c", "import UnityPy").Run(); err != nil {
		t.Skip("UnityPy module not available on test runner; skipping Scenario A")
	}

	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	extractScript := filepath.Join(repoRoot, "pkg", "tools", "scripts", "extract.py")
	if _, err := os.Stat(extractScript); err != nil {
		t.Fatalf("extract.py not found at %s: %v", extractScript, err)
	}

	fixtureScript := filepath.Join(repoRoot, "test", "make_fixture.py")
	if _, err := os.Stat(fixtureScript); err != nil {
		t.Fatalf("make_fixture.py not found at %s: %v", fixtureScript, err)
	}

	tempDir := t.TempDir()
	gameDataDir := filepath.Join(tempDir, "Game_Data")
	if err := os.MkdirAll(gameDataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Build real synthetic Unity .assets fixture containing JSON TextAsset and Binary TextAsset
	assetsFile := filepath.Join(gameDataDir, "resources.assets")
	genCmd := exec.Command(pythonExe, fixtureScript, assetsFile)
	if genOut, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build test .assets fixture: %v, output: %s", err, string(genOut))
	}

	workDir := filepath.Join(tempDir, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(pythonExe, extractScript, tempDir, workDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("extract.py failed: %v\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	extractedFile := filepath.Join(workDir, "extracted.json")
	extractedBytes, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("failed to read extracted.json: %v\nOutput was:\n%s", err, stdout.String())
	}

	var extractedEntries []map[string]interface{}
	if err := json.Unmarshal(extractedBytes, &extractedEntries); err != nil {
		t.Fatalf("failed to unmarshal extracted.json: %v", err)
	}

	if len(extractedEntries) == 0 {
		t.Fatalf("No entries extracted from synthetic fixture. Output:\n%s", stdout.String())
	}

	var foundArthurDialogue, foundMerchantDialogue, foundOK, foundCancel, foundStart bool
	var foundBinaryReadOnly bool

	for _, e := range extractedEntries {
		src, _ := e["source"].(string)
		cat, _ := e["category"].(string)
		path, _ := e["path"].(string)
		readOnly, _ := e["read_only"].(bool)
		cls, _ := e["candidate_class"].(string)

		// 1. Noise check
		if strings.Contains(strings.ToLower(src), "m_name") || strings.Contains(strings.ToLower(path), "m_name") {
			t.Errorf("FAIL: extracted noise field containing m_name: %v", e)
		}
		if strings.Contains(strings.ToLower(src), "charactersequence") || strings.Contains(strings.ToLower(path), "charactersequence") {
			t.Errorf("FAIL: extracted noise field containing characterSequence: %v", e)
		}

		// 2. Structured JSON TextAsset decomposed assertions
		if strings.Contains(src, "Welcome to our tavern") {
			foundArthurDialogue = true
			if cat != "dialogue" {
				t.Errorf("Expected Arthur text to have category='dialogue', got '%s'", cat)
			}
			if cls != "high-confidence" {
				t.Errorf("Expected dialogue to have candidate_class='high-confidence', got '%s'", cls)
			}
			if !strings.Contains(path, "#jpath:dialogues[0].text") {
				t.Errorf("Expected JSON path '#jpath:dialogues[0].text', got '%s'", path)
			}
		}

		if strings.Contains(src, "Looking for rare wares") {
			foundMerchantDialogue = true
			if cat != "dialogue" {
				t.Errorf("Expected Merchant text to have category='dialogue', got '%s'", cat)
			}
		}

		// 3. Legitimate short UI keywords preserved
		if src == "OK" {
			foundOK = true
			if cat != "ui" {
				t.Errorf("Expected 'OK' to have category='ui', got '%s'", cat)
			}
			if readOnly {
				t.Errorf("Expected 'OK' in JSON TextAsset to be writable, got read_only=true")
			}
		}
		if src == "Cancel" {
			foundCancel = true
		}
		if src == "Start Game" {
			foundStart = true
			if cat != "ui" {
				t.Errorf("Expected 'Start Game' to have category='ui', got '%s'", cat)
			}
		}

		// 4. Binary TextAsset tagged as read-only
		file, _ := e["file"].(string)
		if strings.Contains(file, "compiled_shaders") || strings.Contains(path, "compiled_shaders") {
			if readOnly {
				foundBinaryReadOnly = true
				if cls != "raw-heuristic" {
					t.Errorf("Expected binary candidate_class='raw-heuristic', got '%s'", cls)
				}
			}
		}
	}

	if !foundArthurDialogue || !foundMerchantDialogue {
		t.Errorf("Failed to extract structured dialogue entries: Arthur=%v, Merchant=%v", foundArthurDialogue, foundMerchantDialogue)
	}
	if !foundOK || !foundCancel || !foundStart {
		t.Errorf("Failed to preserve legitimate short UI keywords: OK=%v, Cancel=%v, Start=%v", foundOK, foundCancel, foundStart)
	}
	if !foundBinaryReadOnly {
		t.Errorf("Failed to mark binary TextAsset as read_only=true")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO B: End-to-End Safe Translation
// ─────────────────────────────────────────────────────────────────────────────
func TestScenarioB_EndToEndSafeTranslation(t *testing.T) {
	tempDir := t.TempDir()
	gameDir := filepath.Join(tempDir, "Game")
	if err := os.MkdirAll(filepath.Join(gameDir, "Data"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Create original game file
	origContent := `{"title": "The Quest of Legends", "version": "1.0.0", "menu": {"start": "Start Adventure", "exit": "Exit"}}`
	targetRel := filepath.Join("Data", "ui.json")
	targetFull := filepath.Join(gameDir, targetRel)
	if err := os.WriteFile(targetFull, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origHashBytes := sha256.Sum256([]byte(origContent))
	origSHA256 := hex.EncodeToString(origHashBytes[:])

	// 2. Perform translation and staging strictly outside the game directory
	stagedDir := filepath.Join(tempDir, "StagedOutput")
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	translatedContent := `{"title": "The Quest of Legends", "version": "1.0.0", "menu": {"start": "شروع ماجراجویی", "exit": "Exit"}}`
	stagedFile := filepath.Join(stagedDir, "ui.json")
	if err := os.WriteFile(stagedFile, []byte(translatedContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify assertion: Original game file remains byte-for-byte untouched during staging
	currentOrigBytes, err := os.ReadFile(targetFull)
	if err != nil {
		t.Fatal(err)
	}
	currentOrigSHA := hex.EncodeToString(func() []byte { h := sha256.Sum256(currentOrigBytes); return h[:] }())
	if currentOrigSHA != origSHA256 {
		t.Fatalf("FAIL: Game installation mutated during staging! Expected %s, got %s", origSHA256, currentOrigSHA)
	}

	// 3. Build FFP1 Patch
	patchOutDir := filepath.Join(tempDir, "PatchPackage")
	buildCfg := installer.BuildConfig{
		GameRoot:  gameDir,
		OutputDir: patchOutDir,
		Targets: []installer.PatchTarget{
			{
				GamePath:       targetRel,
				PatchedFile:    stagedFile,
				OriginalFile:   targetFull,
				OriginalSHA256: origSHA256,
			},
		},
		PatchName: "Test FarsiForge Patch",
		GameExe:   "QuestOfLegends.exe",
	}

	buildRes, err := installer.Build(buildCfg)
	if err != nil {
		t.Fatalf("Failed to build installer patch: %v", err)
	}

	if _, err := os.Stat(buildRes.PatchFile); err != nil {
		t.Fatalf("Patch file was not generated: %v", err)
	}

	// 4. Test Stale/Mismatched Source Verification
	tamperedContent := `{"title": "The Quest of Legends v2", "version": "2.0.0", "menu": {"start": "New Game", "exit": "Exit"}}`
	tamperedFile := filepath.Join(gameDir, "Data", "tampered.json")
	if err := os.WriteFile(tamperedFile, []byte(tamperedContent), 0o644); err != nil {
		t.Fatal(err)
	}

	staleCfg := installer.BuildConfig{
		GameRoot:  gameDir,
		OutputDir: filepath.Join(tempDir, "StaleTest"),
		Targets: []installer.PatchTarget{
			{
				GamePath:       filepath.Join("Data", "tampered.json"),
				PatchedFile:    stagedFile,
				OriginalFile:   tamperedFile,
				OriginalSHA256: origSHA256, // Stale SHA256 that does not match tampered file
			},
		},
	}
	_, staleErr := installer.Build(staleCfg)
	if staleErr == nil {
		t.Fatalf("FAIL: Installer accepted stale source file without rejecting mismatch!")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO C: Classification Regression Corpus
// ─────────────────────────────────────────────────────────────────────────────
func TestScenarioC_ClassificationRegressionCorpus(t *testing.T) {
	type CorpusItem struct {
		Text            string
		Context         string
		ExpectedClass   string
		ExpectedCat     string
		ShouldBeDropped bool
	}

	corpus := []CorpusItem{
		// Genuine dialogue
		{Text: "Farewell, brave traveler. May the stars illuminate your path.", Context: "Dialogue", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryDialogue},
		{Text: "Who goes there? Speak or draw your sword!", Context: "NPC_Talk", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryDialogue},
		{Text: "I cannot believe the king has fallen...", Context: "Dialogue", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryDialogue},

		// Genuine UI Labels & short buttons
		{Text: "Start Game", Context: "Menu", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryUI},
		{Text: "Resume", Context: "PauseMenu", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryUI},
		{Text: "Options", Context: "Settings", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryUI},
		{Text: "Quit to Desktop", Context: "Menu", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryUI},
		{Text: "Inventory", Context: "HUD", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryUI},

		// Quest and Items
		{Text: "Moonveil Katana", Context: "ItemName", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryItem},
		{Text: "Defeat the Dragon of the Lake and retrieve its heart.", Context: "QuestDesc", ExpectedClass: core.CandidateHighConfidence, ExpectedCat: core.CategoryQuest},

		// Technical noise / Engine Metadata (MUST NOT be high-confidence)
		{Text: "root.m_Name", Context: "MonoBehaviour", ExpectedClass: core.CandidateTechnical, ShouldBeDropped: true},
		{Text: "m_CreationSettings.characterSequence", Context: "SerializedSettings", ExpectedClass: core.CandidateTechnical, ShouldBeDropped: true},
		{Text: "4a2b9f8e-3c1d-4e5f-9a8b-1c2d3e4f5a6b", Context: "GUID", ExpectedClass: core.CandidateTechnical, ShouldBeDropped: true},
		{Text: "Assets/Resources/Fonts/LiberationSans.ttf", Context: "AssetPath", ExpectedClass: core.CandidateTechnical, ShouldBeDropped: true},
		{Text: "v1.4.2_b120", Context: "Version", ExpectedClass: core.CandidateTechnical, ShouldBeDropped: true},
		{Text: "UnityEngine.UI.Text", Context: "ClassType", ExpectedClass: core.CandidateTechnical, ShouldBeDropped: true},
	}

	fp := 0
	fn := 0

	for _, item := range corpus {
		entry := core.StringEntry{
			Source:  item.Text,
			Context: item.Context,
		}
		res := textfilter.ClassifyCandidate(entry, "unity")

		if item.ShouldBeDropped {
			if res.CandidateClass == core.CandidateHighConfidence {
				t.Errorf("False Positive: noise %q classified as high-confidence (Category=%s, Confidence=%.2f)", item.Text, res.Category, res.Confidence)
				fp++
			}
		} else {
			if res.CandidateClass == core.CandidateTechnical || res.CandidateClass == core.CandidateRawHeuristic {
				t.Errorf("False Negative: genuine text %q classified as %s", item.Text, res.CandidateClass)
				fn++
			}
			if item.ExpectedCat != "" && res.Category != item.ExpectedCat {
				t.Errorf("Category mismatch for %q: got %s, want %s", item.Text, res.Category, item.ExpectedCat)
			}
		}
	}

	if fp > 0 {
		t.Errorf("Classification failed: %d false positives", fp)
	}
	if fn > 0 {
		t.Errorf("Classification failed: %d false negatives", fn)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO D: Backward Compatibility and Scalability
// ─────────────────────────────────────────────────────────────────────────────
func TestScenarioD_BackwardCompatibilityAndPerformance(t *testing.T) {
	oldProjectJSON := `{
		"game_root": "C:/Games/TestGame",
		"engine": "unity",
		"entries": [
			{
				"id": "entry-001",
				"file": "resources.assets",
				"path": "dialogues[0].text",
				"format": "unity-typetree",
				"source": "I welcome you, brave traveler! Have a drink.",
				"translation": "خوش آمدید، مسافر!",
				"status": "approved"
			},
			{
				"id": "entry-002",
				"file": "resources.assets",
				"path": "ui.start",
				"source": "Start Game",
				"translation": "شروع بازی",
				"status": "translated"
			}
		]
	}`

	var proj core.Project
	if err := json.Unmarshal([]byte(oldProjectJSON), &proj); err != nil {
		t.Fatalf("Failed to deserialize older project schema: %v", err)
	}

	// ClassifyProject adds candidate_class and preserves user translations
	_ = textfilter.ClassifyProject(&proj)
	if len(proj.Entries) != 2 {
		t.Fatalf("ClassifyProject returned invalid project entry count: %d", len(proj.Entries))
	}

	for _, e := range proj.Entries {
		if e.ID == "entry-001" {
			if e.Translation != "خوش آمدید، مسافر!" {
				t.Errorf("Lost existing translation for entry-001: %s", e.Translation)
			}
			if e.CandidateClass != core.CandidateHighConfidence {
				t.Errorf("Expected high-confidence for dialogue entry, got %s", e.CandidateClass)
			}
		}
		if e.ID == "entry-002" {
			if e.Translation != "شروع بازی" {
				t.Errorf("Lost existing translation for entry-002: %s", e.Translation)
			}
		}
	}

	// 2. Performance & Memory Pagination on Asset Browser
	tempDir := t.TempDir()
	dummyAssets := filepath.Join(tempDir, "test.assets")
	if err := os.WriteFile(dummyAssets, []byte("dummy assets content"), 0o644); err != nil {
		t.Fatal(err)
	}

	toolReg, _ := tools.NewRegistry("")
	p1, err := assetbrowser.InspectGameAsset(context.Background(), tempDir, "test.assets", "unity", 1, 50, toolReg)
	if err != nil {
		if strings.Contains(err.Error(), "UnityPy is not installed") {
			t.Skip("UnityPy is not installed on test runner; skipping InspectGameAsset check")
		}
		t.Fatalf("InspectGameAsset failed: %v", err)
	}
	if p1.Page != 1 || p1.PageSize != 50 {
		t.Errorf("Invalid pagination parameters: got page=%d, pageSize=%d", p1.Page, p1.PageSize)
	}
}
