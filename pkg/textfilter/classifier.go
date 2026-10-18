package textfilter

import (
	"regexp"
	"strings"
	"unicode"

	"farsiforge/pkg/core"
)

// CandidateClassification holds the semantic and quality classification result for a string entry.
type CandidateClassification struct {
	CandidateClass string
	Category       string
	Confidence     float64
	ReadOnly       bool
	ReadOnlyReason string
	FilterReason   string
}

var (
	// snakeIdentifierRe matches snake_case code identifiers like dialogue_node_54, on_player_spawn.
	snakeIdentifierRe = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+){2,}$`)

	// camelIdentifierRe matches camelCase identifiers without spaces like bShowSubtitles, m_LocalScale.
	camelIdentifierRe = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*[A-Z][a-zA-Z0-9]*$`)

	// dottedIdentifierRe matches dotted variable names like npc.sister.name, player.stats.health.
	dottedIdentifierRe = regexp.MustCompile(`^[a-z0-9_]+(?:\.[a-z0-9_]+)+$`)

	// standardUIKeywords is a curated dictionary of short, high-value game UI terms.
	// This prevents legitimate 1-2 word UI labels ("Yes", "No", "Back", "Quit") from being filtered out.
	standardUIKeywords = map[string]bool{
		"ok": true, "yes": true, "no": true, "cancel": true, "apply": true,
		"save": true, "load": true, "back": true, "exit": true, "quit": true,
		"start": true, "continue": true, "retry": true, "play": true, "pause": true,
		"resume": true, "options": true, "settings": true, "audio": true, "video": true,
		"graphics": true, "controls": true, "gameplay": true, "language": true, "help": true,
		"credits": true, "inventory": true, "map": true, "quest": true, "quests": true,
		"items": true, "skills": true, "stats": true, "profile": true, "delete": true,
		"edit": true, "new": true, "next": true, "previous": true, "accept": true,
		"decline": true, "confirm": true, "close": true, "open": true, "buy": true,
		"sell": true, "equip": true, "unequip": true, "use": true, "discard": true,
		"craft": true, "upgrade": true, "select": true, "clear": true, "reset": true,
		"mute": true, "unmute": true, "default": true, "fullscreen": true, "windowed": true,
		"vsync": true, "brightness": true, "gamma": true, "volume": true, "master": true,
		"music": true, "sfx": true, "voice": true, "difficulty": true, "easy": true,
		"normal": true, "hard": true, "nightmare": true, "new game": true, "load game": true,
		"save game": true, "main menu": true, "restart": true, "abandon": true,
		"hp": true, "mp": true, "exp": true, "xp": true, "lvl": true, "level": true, "sp": true,
	}

	// technicalNoisePrefixes catches common debug / engine log prefixes.
	technicalNoisePrefixes = []string{
		"[debug]", "[error]", "[warn]", "[info]", "[trace]", "[assert]",
		"assertion failed", "fatal error:", "exception in", "stack trace:",
	}
)

// ClassifyCandidate evaluates a game string entry against structural evidence,
// semantic context, natural-language heuristics, and format write-back safety.
func ClassifyCandidate(entry core.StringEntry, engine string) CandidateClassification {
	src := strings.TrimSpace(entry.Source)
	cls := CandidateClassification{
		CandidateClass: core.CandidateHighConfidence,
		Category:       core.CategoryUnclassified,
		Confidence:     0.80,
		ReadOnly:       entry.ReadOnly,
		ReadOnlyReason: entry.ReadOnlyReason,
	}

	// 1. Container Write-Back Safety Contract (Task 2 & 3)
	// If the string resides in an archive container that does not support direct write-back,
	// mark it strictly Read-Only regardless of text quality.
	if entry.Container != "" {
		cLower := strings.ToLower(entry.Container)
		if strings.HasSuffix(cLower, ".bhd") || strings.HasSuffix(cLower, ".bdt") {
			cls.ReadOnly = true
			cls.ReadOnlyReason = "Archive container write-back not supported for FromSoftware BHD/BDT packages"
		} else if strings.HasSuffix(cLower, ".pak") || strings.HasSuffix(cLower, ".utoc") || strings.HasSuffix(cLower, ".ucas") {
			cls.ReadOnly = true
			cls.ReadOnlyReason = "Archive-resident string cannot be safely modified without repacking containing Unreal archive"
		}
	}

	// 2. Raw / Heuristic Byte Scan detection
	isRawScan := entry.SourceType == "raw-binary" || entry.SourceType == "raw-bytes" ||
		entry.SourceType == "uexp-strings" || entry.SourceType == "monobehaviour-bytes"
	if isRawScan {
		cls.ReadOnly = true
		if cls.ReadOnlyReason == "" {
			cls.ReadOnlyReason = "Heuristic byte scan without format AST; cannot safely write back"
		}
	}

	// 3. Obvious Technical Identifiers and Binary Noise
	if src == "" {
		cls.CandidateClass = core.CandidateTechnical
		cls.Category = core.CategorySystem
		cls.Confidence = 0.0
		cls.FilterReason = "empty_string"
		return cls
	}

	if isTechnicalNoise(src) {
		cls.CandidateClass = core.CandidateTechnical
		cls.Category = core.CategorySystem
		cls.Confidence = 0.05
		cls.FilterReason = "technical_identifier"
		return cls
	}

	// 4. Basic Translatability Check (markup, assembly refs, hex runs, GUIDs)
	if !IsTranslatable(src) {
		cls.CandidateClass = core.CandidateTechnical
		cls.Category = core.CategorySystem
		cls.Confidence = 0.05
		cls.FilterReason = "non_translatable_token"
		return cls
	}

	// 5. Short UI Vocabulary Match (Avoid over-filtering "Yes/No", "Back", etc.)
	cleanText := strings.ToLower(TranslatableText(src))
	if standardUIKeywords[cleanText] {
		cls.Category = core.CategoryUI
		cls.CandidateClass = core.CandidateHighConfidence
		cls.Confidence = 0.95
		if isRawScan {
			cls.CandidateClass = core.CandidateRawHeuristic
		}
		return cls
	}

	// 6. Persian / Arabic Source Script Detection
	if containsArabicRune(src) {
		cls.Confidence = 0.98
		cls.CandidateClass = core.CandidateHighConfidence
		cls.Category = detectSemanticCategory(entry, src)
		if isRawScan {
			cls.CandidateClass = core.CandidateRawHeuristic
		}
		return cls
	}

	// 7. Semantic Category Detection
	cls.Category = detectSemanticCategory(entry, src)

	// 8. Confidence Scoring based on Structural Evidence
	isStructuredFormat := isVerifiedLocalizationFormat(entry)
	wordCount := len(strings.Fields(cleanText))

	if isStructuredFormat {
		if wordCount >= 2 {
			cls.Confidence = 0.95
			cls.CandidateClass = core.CandidateHighConfidence
		} else if wordCount == 1 {
			// Single natural word not in standard UI dictionary
			cls.Confidence = 0.70
			cls.CandidateClass = core.CandidateNeedsReview
			cls.FilterReason = "single_word_unverified"
		}
	} else if isRawScan {
		cls.CandidateClass = core.CandidateRawHeuristic
		cls.Confidence = 0.50
	} else {
		// Generic or text scan without proven localization schema
		if wordCount >= 4 && (strings.ContainsAny(src, ".!?,;\"'") || cls.Category == core.CategoryDialogue) {
			cls.Confidence = 0.85
			cls.CandidateClass = core.CandidateHighConfidence
		} else if wordCount >= 2 {
			cls.Confidence = 0.70
			cls.CandidateClass = core.CandidateNeedsReview
			cls.FilterReason = "unstructured_short_phrase"
		} else {
			cls.Confidence = 0.40
			cls.CandidateClass = core.CandidateNeedsReview
			cls.FilterReason = "single_word_unstructured"
		}
	}

	return cls
}

// detectSemanticCategory inspects path, context, speaker, and text patterns to classify the string.
func detectSemanticCategory(entry core.StringEntry, src string) string {
	// 0. Pre-classified structural category from AST / schema extractor
	if entry.Category != "" && entry.Category != core.CategoryUnclassified {
		return entry.Category
	}

	// 1. Explicit Speaker indicates Dialogue
	if strings.TrimSpace(entry.Speaker) != "" {
		return core.CategoryDialogue
	}

	meta := strings.ToLower(entry.File + " " + entry.Path + " " + entry.Context)

	// 2. Dialogue / Subtitles metadata
	if strings.Contains(meta, "dialogue") || strings.Contains(meta, "dialog") ||
		strings.Contains(meta, "subtitle") || strings.Contains(meta, "speech") ||
		strings.Contains(meta, "talk") || strings.Contains(meta, "voice") ||
		strings.Contains(meta, "chat") || strings.Contains(meta, "cutscene") ||
		strings.Contains(meta, "conv") || strings.Contains(meta, "npc") {
		return core.CategoryDialogue
	}

	// 3. UI / Menu / HUD metadata
	if strings.Contains(meta, "menu") || strings.Contains(meta, "ui") ||
		strings.Contains(meta, "hud") || strings.Contains(meta, "btn") ||
		strings.Contains(meta, "button") || strings.Contains(meta, "title") ||
		strings.Contains(meta, "window") || strings.Contains(meta, "prompt") ||
		strings.Contains(meta, "tooltip") || strings.Contains(meta, "popup") {
		return core.CategoryUI
	}

	// 4. Items / Weapons / Equipment metadata
	if strings.Contains(meta, "item") || strings.Contains(meta, "weapon") ||
		strings.Contains(meta, "armor") || strings.Contains(meta, "potion") ||
		strings.Contains(meta, "inventory") || strings.Contains(meta, "gear") ||
		strings.Contains(meta, "spell") || strings.Contains(meta, "skill") ||
		strings.Contains(meta, "perk") || strings.Contains(meta, "artifact") {
		return core.CategoryItem
	}

	// 5. Quest / Mission / Objective metadata
	if strings.Contains(meta, "quest") || strings.Contains(meta, "mission") ||
		strings.Contains(meta, "objective") || strings.Contains(meta, "task") ||
		strings.Contains(meta, "bounty") || strings.Contains(meta, "journal") ||
		strings.Contains(meta, "log") {
		return core.CategoryQuest
	}

	// 6. Tutorial / Guide / Help metadata
	if strings.Contains(meta, "tutorial") || strings.Contains(meta, "help") ||
		strings.Contains(meta, "guide") || strings.Contains(meta, "hint") ||
		strings.Contains(meta, "tip") {
		return core.CategoryTutorial
	}

	// 7. System / Alert metadata
	if strings.Contains(meta, "system") || strings.Contains(meta, "network") ||
		strings.Contains(meta, "server") || strings.Contains(meta, "error") ||
		strings.Contains(meta, "warning") || strings.Contains(meta, "connection") ||
		strings.Contains(meta, "terminal") || strings.Contains(meta, "console") ||
		strings.Contains(meta, "command") || strings.Contains(meta, "notice") ||
		strings.Contains(meta, "notification") || strings.Contains(meta, "message") {
		return core.CategorySystem
	}

	// 8. Conversational text heuristic
	if isConversationalText(src) {
		return core.CategoryDialogue
	}

	return core.CategoryUnclassified
}

// isConversationalText checks for dialogue traits (first/second person pronouns, quotation, punctuation).
func isConversationalText(s string) bool {
	lower := strings.ToLower(s)
	hasPronoun := strings.Contains(lower, "you ") || strings.Contains(lower, "you're") ||
		strings.Contains(lower, "i ") || strings.Contains(lower, "i'm") ||
		strings.Contains(lower, "we ") || strings.Contains(lower, "we're") ||
		strings.Contains(lower, "my ") || strings.Contains(lower, "your ") ||
		strings.Contains(lower, "don't") || strings.Contains(lower, "can't") ||
		strings.Contains(lower, "please")

	hasPunct := strings.ContainsAny(s, "?!\"“”«»")
	words := len(strings.Fields(s))
	return hasPronoun && hasPunct && words >= 3
}

// isVerifiedLocalizationFormat checks whether the source file/context indicates a dedicated localization table.
func isVerifiedLocalizationFormat(entry core.StringEntry) bool {
	f := strings.ToLower(entry.File)
	ctx := strings.ToLower(entry.Context)
	st := strings.ToLower(entry.SourceType)

	if st == "locres" || st == "fmg" || st == "gxt2" || st == "po" ||
		st == "translation" || st == "strings" || st == "csv" || st == "json" ||
		st == "textasset-json" || st == "textasset-csv" || st == "textasset-kv" ||
		st == "textasset-text" || st == "unity-typetree" || st == "unity-reconstructed" {
		return true
	}

	if strings.HasSuffix(f, ".locres") || strings.HasSuffix(f, ".fmg") ||
		strings.HasSuffix(f, ".gxt2") || strings.HasSuffix(f, ".po") ||
		strings.HasSuffix(f, ".mo") || strings.HasSuffix(f, ".translation") ||
		strings.HasSuffix(f, ".str") || strings.HasSuffix(f, ".vtxt") {
		return true
	}

	if strings.Contains(ctx, "locres") || strings.Contains(ctx, "fmg") ||
		strings.Contains(ctx, "gxt2") || strings.Contains(ctx, "factorio") ||
		strings.Contains(ctx, "goldsrc") || strings.Contains(ctx, "source2") ||
		strings.Contains(ctx, "sage") {
		return true
	}

	return false
}

// isTechnicalNoise tests for obvious non-natural language identifier patterns.
func isTechnicalNoise(s string) bool {
	lower := strings.ToLower(s)

	for _, prefix := range technicalNoisePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	// Snake case identifiers (e.g. menu_btn_close_idle)
	if snakeIdentifierRe.MatchString(s) {
		return true
	}

	// Camel case identifiers without whitespace (e.g. bShowSubtitles, m_LocalScale)
	if len(s) > 4 && !strings.Contains(s, " ") && camelIdentifierRe.MatchString(s) {
		return true
	}

	// Dotted identifiers (e.g. system.render.backend)
	if dottedIdentifierRe.MatchString(s) {
		return true
	}

	// Check if string contains only digits and math/symbol runes
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return true
	}

	return false
}

// containsArabicRune tests for Persian/Arabic Unicode characters.
func containsArabicRune(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

// ClassifyProject processes all entries in a project, populating their classification fields
// and returning comprehensive extraction and filtering statistics.
func ClassifyProject(proj *core.Project) core.ExtractionStats {
	stats := core.ExtractionStats{
		TotalParsed:      len(proj.Entries),
		FilteredByReason: make(map[string]int),
	}

	for i := range proj.Entries {
		e := &proj.Entries[i]
		cls := ClassifyCandidate(*e, proj.Engine)

		e.CandidateClass = cls.CandidateClass
		e.Category = cls.Category
		e.Confidence = cls.Confidence
		if cls.ReadOnly {
			e.ReadOnly = true
			e.ReadOnlyReason = cls.ReadOnlyReason
		}

		switch cls.CandidateClass {
		case core.CandidateHighConfidence:
			stats.HighConfidenceCount++
			stats.AcceptedCount++
		case core.CandidateNeedsReview:
			stats.NeedsReviewCount++
			stats.AcceptedCount++
			if cls.FilterReason != "" {
				stats.FilteredByReason[cls.FilterReason]++
			}
		case core.CandidateTechnical:
			stats.TechnicalCount++
			if cls.FilterReason != "" {
				stats.FilteredByReason[cls.FilterReason]++
			}
		case core.CandidateRawHeuristic:
			stats.RawHeuristicCount++
			stats.AcceptedCount++
			if cls.FilterReason != "" {
				stats.FilteredByReason[cls.FilterReason]++
			}
		}

		if e.ReadOnly {
			stats.ReadOnlyCount++
		}
	}

	proj.ExtractionStats = &stats
	return stats
}
