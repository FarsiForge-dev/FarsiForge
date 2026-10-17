package core

import "strings"

// EngineCapability represents the full operational capability of an engine in FarsiForge.
type EngineCapability struct {
	Engine          string   `json:"engine"`
	DisplayName     string   `json:"display_name"`
	CanDetect       bool     `json:"can_detect"`
	CanExtract      bool     `json:"can_extract"`
	CanInject       bool     `json:"can_inject"`
	ArchiveRebuild  bool     `json:"archive_rebuild"`
	FontSupport     bool     `json:"font_support"`
	PatchInstall    bool     `json:"patch_install"`
	Status          string   `json:"status"` // "full", "extract-only", "detection-only", "custom"
	ExtractionNotes string   `json:"extraction_notes,omitempty"`
	InjectionNotes  string   `json:"injection_notes,omitempty"`
	RequiredTools   []string `json:"required_tools,omitempty"`
}

// AllEngineCapabilities returns the authoritative capability matrix for all supported engines.
func AllEngineCapabilities() []EngineCapability {
	return []EngineCapability{
		{
			Engine:          "unity",
			DisplayName:     "Unity",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       true,
			ArchiveRebuild:  false,
			FontSupport:     true,
			PatchInstall:    true,
			Status:          "full",
			ExtractionNotes: "Supports TextAsset, MonoBehaviour, and UnityPy extraction; unverified raw bytes are guarded.",
			InjectionNotes:  "Safe loose asset staging in WorkingDir/out/; unverified raw byte mutation is blocked.",
			RequiredTools:   []string{"UnityPy", "FarsiForgePatcher"},
		},
		{
			Engine:          "unreal",
			DisplayName:     "Unreal Engine",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       true,
			ArchiveRebuild:  false,
			FontSupport:     true,
			PatchInstall:    true,
			Status:          "full",
			ExtractionNotes: "Supports loose .locres, .pak, and IoStore .utoc extractions with container preservation.",
			InjectionNotes:  "Supports loose .locres injection; archive-resident .locres marked read-only to prevent phantom overrides.",
			RequiredTools:   []string{"repak", "retoc", "FarsiForgePatcher"},
		},
		{
			Engine:          "godot",
			DisplayName:     "Godot",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       true,
			ArchiveRebuild:  true,
			FontSupport:     true,
			PatchInstall:    true,
			Status:          "full",
			ExtractionNotes: "Supports PCK v1-v4 packages and Godot 4 OptimizedTranslation catalogs.",
			InjectionNotes:  "Supports PCK rebuild and standalone translation injection with Persian bidi and shaping.",
			RequiredTools:   []string{"fftools", "FarsiForgePatcher"},
		},
		{
			Engine:          "fromsoftware",
			DisplayName:     "FromSoftware",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       true,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    true,
			Status:          "full",
			ExtractionNotes: "Supports loose FMG, BND/DCX binders, and Data0.bhd/bdt extraction with container preservation.",
			InjectionNotes:  "Loose FMG and BND/DCX stage to WorkingDir/out/; Data0.bhd/bdt archive entries are strictly read-only.",
			RequiredTools:   []string{"FarsiForgePatcher"},
		},
		{
			Engine:          "generic",
			DisplayName:     "Generic / Plain Text",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       true,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    true,
			Status:          "full",
			ExtractionNotes: "Plain text, INI, JSON, CSV files with key-value or localized strings.",
			InjectionNotes:  "Non-destructive replacement with strict original hash binding and staging in WorkingDir/out/.",
			RequiredTools:   []string{"FarsiForgePatcher"},
		},
		{
			Engine:          "custom",
			DisplayName:     "Custom Engine",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       true,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    true,
			Status:          "custom",
			ExtractionNotes: "Fallback generic parser scanning text resources in custom engine installations.",
			InjectionNotes:  "Generic text staging with FFP1 patch creation.",
			RequiredTools:   []string{"FarsiForgePatcher"},
		},
		{
			Engine:          "rage",
			DisplayName:     "RAGE (Rockstar)",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "extract-only",
			ExtractionNotes: "RPF7 and GXT2 text extraction supported with key-material resolution.",
			InjectionNotes:  "Injection is disabled (extract-only); requires custom RPF archive signing key material.",
			RequiredTools:   []string{"fftools"},
		},
		{
			Engine:          "sage",
			DisplayName:     "SAGE (EA RTS)",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "extract-only",
			ExtractionNotes: "BIG archive and CSF string table extraction supported.",
			InjectionNotes:  "Injection is disabled (extract-only); binary CSF packaging not currently enabled.",
		},
		{
			Engine:          "goldsrc",
			DisplayName:     "GoldSrc (Half-Life 1)",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "extract-only",
			ExtractionNotes: "PAK archives and titles.txt extraction supported.",
			InjectionNotes:  "Injection is disabled (extract-only).",
		},
		{
			Engine:          "source2",
			DisplayName:     "Source 2 (Valve)",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "extract-only",
			ExtractionNotes: "VPK and compiled localization files supported.",
			InjectionNotes:  "Injection is disabled (extract-only).",
		},
		{
			Engine:          "factorio",
			DisplayName:     "Factorio",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "extract-only",
			ExtractionNotes: "CFG and locale script parsing supported.",
			InjectionNotes:  "Injection is disabled (extract-only).",
		},
		{
			Engine:          "zomboid",
			DisplayName:     "Project Zomboid",
			CanDetect:       true,
			CanExtract:      true,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "extract-only",
			ExtractionNotes: "Translation script tables supported.",
			InjectionNotes:  "Injection is disabled (extract-only).",
		},
		{
			Engine:          "ue3",
			DisplayName:     "Unreal Engine 3",
			CanDetect:       true,
			CanExtract:      false,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "detection-only",
			ExtractionNotes: "Detection-only; UPK package extraction requires external tooling not currently integrated.",
			InjectionNotes:  "Injection is not supported.",
		},
		{
			Engine:          "source",
			DisplayName:     "Source 1 (Valve)",
			CanDetect:       true,
			CanExtract:      false,
			CanInject:       false,
			ArchiveRebuild:  false,
			FontSupport:     false,
			PatchInstall:    false,
			Status:          "detection-only",
			ExtractionNotes: "Detection-only; Source 1 extraction not implemented (Source 2 is supported for extraction).",
			InjectionNotes:  "Injection is not supported.",
		},
	}
}

// FindEngineCapability returns capability for a given engine identifier.
func FindEngineCapability(engine string) EngineCapability {
	target := strings.ToLower(strings.TrimSpace(engine))
	for _, cap := range AllEngineCapabilities() {
		if strings.EqualFold(cap.Engine, target) {
			return cap
		}
	}
	// Fallback generic capability
	return EngineCapability{
		Engine:          engine,
		DisplayName:     engine,
		CanDetect:       true,
		CanExtract:      true,
		CanInject:       true,
		ArchiveRebuild:  false,
		FontSupport:     false,
		PatchInstall:    true,
		Status:          "custom",
		ExtractionNotes: "Custom engine using generic fallback extraction.",
		InjectionNotes:  "Generic text replacement staging.",
	}
}
