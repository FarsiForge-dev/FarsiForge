// Package validate provides translation string validation.
//
// Ensures placeholders, tags, and structure remain intact after translation
// to prevent game crashes, corrupt UI layouts, or runtime formatting exceptions.
package validate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"farsiforge/pkg/core"
	"farsiforge/pkg/textfilter"
)

// StandardValidator implements core.IValidator and serves as the single source
// of truth for translation QA and pre-injection validation.
type StandardValidator struct {
	// Custom placeholder patterns (in addition to textfilter built-in intelligence).
	Placeholders []*regexp.Regexp
}

// New creates a new StandardValidator.
func New() *StandardValidator {
	return &StandardValidator{
		Placeholders: []*regexp.Regexp{
			// Extended printf variants: %s, %d, %1$s, %02d, %.2f, %+5d, %lld, %zu, %I64d
			regexp.MustCompile(`%(?:\d+\$)?[0-9.+#\-]*(?:ll|l|h|hh|z|j|t|I64|I32)?[dsfpxXeEgGciou]`),
			// Named and numeric format arguments: {player}, {0}, {name:s}, {count, number}
			regexp.MustCompile(`\{[a-zA-Z0-9_]+(?::[a-zA-Z0-9_]+)?(?:,\s*[a-zA-Z0-9_]+)?\}`),
			// Unity Rich Text / XML tags: <color=#ff0000>, </color>, <b>, </i>, <size=20>
			regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9_:.!-]*(?:\s[^<>]*)?/?>`),
			// BBCode tags: [b], [/b], [color=red], [size=12]
			regexp.MustCompile(`\[/?[a-zA-Z0-9_]+(?:=[^\]]+)?\]`),
			// Escaped control characters: \n, \r, \t, \0
			regexp.MustCompile(`\\[nrt0"'\\]`),
		},
	}
}

// Validate checks all entries and returns a list of validation errors.
// Output is sorted deterministically for stable reporting across runs.
func (v *StandardValidator) Validate(entries []core.StringEntry) []core.ValidationError {
	var errs []core.ValidationError

	for _, entry := range entries {
		// Only validate translated strings
		if entry.Status != core.StatusTranslated && entry.Status != core.StatusApproved {
			continue
		}

		// 1. Empty translation check
		if strings.TrimSpace(entry.Translation) == "" {
			errs = append(errs, core.ValidationError{
				EntryID:  entry.ID,
				Type:     core.ValEmptyTranslation,
				Message:  "Translation is empty for a translated entry",
				Severity: core.SevWarning,
			})
			continue
		}

		// 2. Length validation (checked by character/rune count)
		if entry.MaxLength > 0 {
			runeCount := utf8.RuneCountInString(entry.Translation)
			if runeCount > entry.MaxLength {
				errs = append(errs, core.ValidationError{
					EntryID:  entry.ID,
					Type:     core.ValLengthExceeded,
					Message:  fmt.Sprintf("Translation length (%d runes) exceeds maximum length (%d)", runeCount, entry.MaxLength),
					Severity: core.SevWarning,
					Details:  fmt.Sprintf("limit=%d actual=%d", entry.MaxLength, runeCount),
				})
			}
		}

		// 3. QA analysis via textfilter (reconciled single source of truth)
		qaNotes := textfilter.QA(entry.Source, entry.Translation)
		for _, note := range qaNotes {
			valErr := mapQANoteToValidationError(entry.ID, note)
			errs = append(errs, valErr)
		}

		// 4. Custom regex placeholder checks (if any additional patterns configured)
		for _, pattern := range v.Placeholders {
			srcMatches := extractMatches(entry.Source, pattern)
			trMatches := extractMatches(entry.Translation, pattern)

			if !compareMatches(srcMatches, trMatches) {
				// Only append if not already captured by textfilter QA
				formattedDetails := fmtMatches(srcMatches, trMatches)
				if !containsDetails(errs, entry.ID, formattedDetails) {
					errs = append(errs, core.ValidationError{
						EntryID:  entry.ID,
						Type:     core.ValPlaceholderMismatch,
						Message:  "Placeholders/tags do not match original string",
						Severity: core.SevError,
						Details:  formattedDetails,
					})
				}
			}
		}
	}

	// Sort deterministically: EntryID -> Severity (Error first) -> Type -> Message
	sort.SliceStable(errs, func(i, j int) bool {
		if errs[i].EntryID != errs[j].EntryID {
			return errs[i].EntryID < errs[j].EntryID
		}
		if errs[i].Severity != errs[j].Severity {
			// SevError comes before SevWarning
			return errs[i].Severity == core.SevError
		}
		if errs[i].Type != errs[j].Type {
			return errs[i].Type < errs[j].Type
		}
		return errs[i].Message < errs[j].Message
	})

	return errs
}

// mapQANoteToValidationError maps machine-readable textfilter.QA notes to structured ValidationError.
func mapQANoteToValidationError(entryID, note string) core.ValidationError {
	switch {
	case strings.HasPrefix(note, "MISMATCH_PLACEHOLDERS:"):
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValPlaceholderMismatch,
			Message:  "Placeholders do not match original string",
			Severity: core.SevError,
			Details:  note,
		}
	case note == "PLACEHOLDER_ORDER_DIFFERS":
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValPlaceholderMismatch,
			Message:  "Placeholder order differs from original string",
			Severity: core.SevWarning,
			Details:  note,
		}
	case strings.HasPrefix(note, "MISMATCH_TAGS:"), strings.HasPrefix(note, "MISSING_TAG:"), strings.HasPrefix(note, "EXTRA_TAG:"):
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValMarkupError,
			Message:  "Markup tags do not match original string",
			Severity: core.SevError,
			Details:  note,
		}
	case note == "TAG_ORDER_DIFFERS":
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValMarkupError,
			Message:  "Markup tag order differs from original string",
			Severity: core.SevWarning,
			Details:  note,
		}
	case strings.HasPrefix(note, "MISMATCH_BRACKETS:"):
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValMarkupError,
			Message:  "Brackets do not match original string",
			Severity: core.SevError,
			Details:  note,
		}
	case note == "BRACKET_ORDER_DIFFERS":
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValMarkupError,
			Message:  "Bracket order differs from original string",
			Severity: core.SevWarning,
			Details:  note,
		}
	case strings.HasPrefix(note, "UNBALANCED_BRACES:"):
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValEscapeError,
			Message:  "Unbalanced braces in translation",
			Severity: core.SevError,
			Details:  note,
		}
	case strings.HasPrefix(note, "LINE_COUNT_MISMATCH:"):
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValEscapeError,
			Message:  "Line count does not match original string",
			Severity: core.SevWarning,
			Details:  note,
		}
	case note == "EMPTY_TRANSLATION":
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValEmptyTranslation,
			Message:  "Translation is empty",
			Severity: core.SevWarning,
			Details:  note,
		}
	default:
		return core.ValidationError{
			EntryID:  entryID,
			Type:     core.ValPlaceholderMismatch,
			Message:  note,
			Severity: core.SevWarning,
			Details:  note,
		}
	}
}

// HasErrors returns true if any validation error has severity SevError (hard failure).
func HasErrors(errs []core.ValidationError) bool {
	for _, e := range errs {
		if e.Severity == core.SevError {
			return true
		}
	}
	return false
}

// FilterBySeverity filters validation errors by severity level.
func FilterBySeverity(errs []core.ValidationError, sev core.Severity) []core.ValidationError {
	var filtered []core.ValidationError
	for _, e := range errs {
		if e.Severity == sev {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// ValidateProject validates all entries in a project using StandardValidator.
func ValidateProject(proj *core.Project) []core.ValidationError {
	if proj == nil {
		return nil
	}
	v := New()
	return v.Validate(proj.Entries)
}

func containsDetails(errs []core.ValidationError, entryID, details string) bool {
	for _, e := range errs {
		if e.EntryID == entryID && e.Details == details {
			return true
		}
	}
	return false
}

func extractMatches(s string, pattern *regexp.Regexp) map[string]int {
	matches := pattern.FindAllString(s, -1)
	counts := make(map[string]int)
	for _, m := range matches {
		counts[m]++
	}
	return counts
}

func compareMatches(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, count := range a {
		if b[k] != count {
			return false
		}
	}
	return true
}

// fmtMatches formats source vs translated match maps for diagnostic output.
// Ordering is deterministic (sorted by key) so test assertions are stable.
func fmtMatches(src, tr map[string]int) string {
	var parts []string
	parts = append(parts, "Source:")
	srcKeys := sortedKeys(src)
	for _, k := range srcKeys {
		// Use Sprintf — string(rune(c+'0')) only works for single digits.
		parts = append(parts, fmt.Sprintf("%s(x%d)", k, src[k]))
	}
	if len(src) == 0 {
		parts = append(parts, "none")
	}

	parts = append(parts, "| Translated:")
	trKeys := sortedKeys(tr)
	for _, k := range trKeys {
		parts = append(parts, fmt.Sprintf("%s(x%d)", k, tr[k]))
	}
	if len(tr) == 0 {
		parts = append(parts, "none")
	}

	return strings.Join(parts, " ")
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
