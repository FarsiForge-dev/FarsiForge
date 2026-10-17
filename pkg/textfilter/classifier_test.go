package textfilter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
)

type GoldenEntry struct {
	ID               string `json:"id"`
	Source           string `json:"source"`
	File             string `json:"file"`
	Context          string `json:"context"`
	Path             string `json:"path"`
	Speaker          string `json:"speaker"`
	Engine           string `json:"engine"`
	SourceType       string `json:"source_type"`
	ExpectedClass    string `json:"expected_class"`
	ExpectedCategory string `json:"expected_category"`
	IsTranslatable   bool   `json:"is_translatable"`
}

func TestClassifyCandidate_GoldenCorpusBenchmark(t *testing.T) {
	fixturePath := filepath.Join("testdata", "golden_corpus.json")
	bytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read golden corpus fixture: %v", err)
	}

	var corpus []GoldenEntry
	if err := json.Unmarshal(bytes, &corpus); err != nil {
		t.Fatalf("failed to parse golden corpus json: %v", err)
	}

	if len(corpus) == 0 {
		t.Fatal("golden corpus must not be empty")
	}

	var tp, fp, tn, fn int
	var categoryMatches, classMatches int

	for _, g := range corpus {
		entry := core.StringEntry{
			ID:         g.ID,
			Source:     g.Source,
			File:       g.File,
			Context:    g.Context,
			Path:       g.Path,
			Speaker:    g.Speaker,
			SourceType: g.SourceType,
		}

		res := ClassifyCandidate(entry, g.Engine)

		isPredictedTranslatable := res.CandidateClass != core.CandidateTechnical

		if g.IsTranslatable && isPredictedTranslatable {
			tp++
		} else if !g.IsTranslatable && isPredictedTranslatable {
			fp++
			t.Errorf("False Positive for %q (ID %s): expected non-translatable technical token, got class %q", g.Source, g.ID, res.CandidateClass)
		} else if !g.IsTranslatable && !isPredictedTranslatable {
			tn++
		} else if g.IsTranslatable && !isPredictedTranslatable {
			fn++
			t.Errorf("False Negative for %q (ID %s): expected translatable text, got technical filter %q", g.Source, g.ID, res.FilterReason)
		}

		if res.CandidateClass == g.ExpectedClass {
			classMatches++
		} else {
			t.Errorf("[%s] class mismatch for %q: got %q, want %q", g.ID, g.Source, res.CandidateClass, g.ExpectedClass)
		}

		if res.Category == g.ExpectedCategory {
			categoryMatches++
		} else {
			t.Errorf("[%s] category mismatch for %q: got %q, want %q", g.ID, g.Source, res.Category, g.ExpectedCategory)
		}
	}

	total := len(corpus)
	precision := float64(tp) / float64(tp+fp)
	recall := float64(tp) / float64(tp+fn)
	accuracy := float64(tp+tn) / float64(total)

	t.Logf("Golden Corpus Results: Total=%d, TP=%d, FP=%d, TN=%d, FN=%d", total, tp, fp, tn, fn)
	t.Logf("Metrics: Precision=%.4f, Recall=%.4f, Accuracy=%.4f, ClassAccuracy=%d/%d, CategoryAccuracy=%d/%d",
		precision, recall, accuracy, classMatches, total, categoryMatches, total)

	if precision < 0.95 {
		t.Errorf("Precision %.4f is below 0.95 gate threshold", precision)
	}
	if recall < 0.95 {
		t.Errorf("Recall %.4f is below 0.95 gate threshold", recall)
	}
	if accuracy < 0.95 {
		t.Errorf("Accuracy %.4f is below 0.95 gate threshold", accuracy)
	}
}

func TestClassifyCandidate_ContainerSafety(t *testing.T) {
	// FromSoftware BHD container entry must be strictly Read-Only
	bhdEntry := core.StringEntry{
		ID:        "fs_bhd_01",
		Source:    "Bearer of the curse, seek misery.",
		File:      "item.msg.bnd.dcx",
		Container: "Data0.bhd",
	}
	resBhd := ClassifyCandidate(bhdEntry, "fromsoftware")
	if !resBhd.ReadOnly {
		t.Fatal("expected FromSoftware BHD container string to be marked ReadOnly")
	}
	if resBhd.ReadOnlyReason == "" {
		t.Fatal("expected descriptive ReadOnlyReason for FromSoftware BHD container")
	}

	// Unreal PAK container entry must be strictly Read-Only
	pakEntry := core.StringEntry{
		ID:        "ue_pak_01",
		Source:    "Press E to Interact",
		File:      "Game.locres",
		Container: "Paks/Pakchunk0-WindowsNoEditor.pak",
	}
	resPak := ClassifyCandidate(pakEntry, "unreal")
	if !resPak.ReadOnly {
		t.Fatal("expected Unreal PAK container string to be marked ReadOnly")
	}

	// Loose file entry must NOT be marked Read-Only
	looseEntry := core.StringEntry{
		ID:     "loose_01",
		Source: "Press E to Interact",
		File:   "Game.locres",
	}
	resLoose := ClassifyCandidate(looseEntry, "unreal")
	if resLoose.ReadOnly {
		t.Fatal("loose file string should not be marked ReadOnly")
	}
}

func TestClassifyProject_StatsGeneration(t *testing.T) {
	proj := &core.Project{
		Engine: "unreal",
		Entries: []core.StringEntry{
			{ID: "1", Source: "Start Game", File: "Menu.locres", Context: "UI"},
			{ID: "2", Source: "Options", File: "Menu.locres", Context: "UI"},
			{ID: "3", Source: "e06d7363-2b99-4d6f-870a-706f9d2a3c11", File: "data.bin"},
			{ID: "4", Source: "A strange monolith in the distance.", File: "raw.bin", SourceType: "raw-binary"},
			{ID: "5", Source: "Press E to open", File: "game.locres", Container: "Paks/Game.pak"},
		},
	}

	stats := ClassifyProject(proj)

	if stats.TotalParsed != 5 {
		t.Fatalf("expected TotalParsed=5, got %d", stats.TotalParsed)
	}
	if stats.HighConfidenceCount < 2 {
		t.Errorf("expected at least 2 high confidence entries, got %d", stats.HighConfidenceCount)
	}
	if stats.TechnicalCount != 1 {
		t.Errorf("expected 1 technical entry, got %d", stats.TechnicalCount)
	}
	if stats.RawHeuristicCount != 1 {
		t.Errorf("expected 1 raw heuristic entry, got %d", stats.RawHeuristicCount)
	}
	if stats.ReadOnlyCount < 2 {
		t.Errorf("expected at least 2 read-only entries (raw-binary + container), got %d", stats.ReadOnlyCount)
	}
	if proj.ExtractionStats == nil {
		t.Fatal("proj.ExtractionStats must be populated")
	}
}
