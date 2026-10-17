package validate

import (
	"strings"
	"testing"

	"farsiforge/pkg/core"
)

// TestFmtMatches_SingleDigit verifies basic formatting for count=1.
func TestFmtMatches_SingleDigit(t *testing.T) {
	src := map[string]int{"{player}": 1}
	tr := map[string]int{}
	out := fmtMatches(src, tr)
	want := "{player}(x1)"
	if !strings.Contains(out, want) {
		t.Errorf("fmtMatches(%v, %v) = %q; want it to contain %q", src, tr, out, want)
	}
}

// TestFmtMatches_CountAboveTen is the regression test for the original bug:
// string(rune(c+'0')) produces a garbage rune for c >= 10.
// After the fix, counts >= 10 must be formatted correctly.
func TestFmtMatches_CountAboveTen(t *testing.T) {
	src := map[string]int{"%d": 12}
	tr := map[string]int{"%d": 12}
	out := fmtMatches(src, tr)
	want := "%d(x12)"
	if !strings.Contains(out, want) {
		t.Errorf("fmtMatches with count=12: got %q; want it to contain %q", out, want)
	}
}

// TestFmtMatches_Deterministic verifies output is sorted and stable across calls.
func TestFmtMatches_Deterministic(t *testing.T) {
	src := map[string]int{"{b}": 1, "{a}": 2, "{c}": 3}
	tr := map[string]int{}
	a := fmtMatches(src, tr)
	b := fmtMatches(src, tr)
	if a != b {
		t.Errorf("fmtMatches output is not stable: %q vs %q", a, b)
	}
	if !strings.Contains(a, "{a}(x2)") || !strings.Contains(a, "{b}(x1)") || !strings.Contains(a, "{c}(x3)") {
		t.Errorf("fmtMatches = %q; missing expected placeholders", a)
	}
}

// TestValidate_PlaceholderMismatch checks that a missing placeholder triggers an error.
func TestValidate_PlaceholderMismatch(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "e1",
			Source:      "Hello {player}",
			Translation: "سلام",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if len(errs) == 0 {
		t.Error("expected placeholder mismatch error, got none")
	}
	found := false
	for _, e := range errs {
		if e.Type == core.ValPlaceholderMismatch && e.Severity == core.SevError {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ValPlaceholderMismatch error with SevError, got: %v", errs)
	}
}

// TestValidate_MaxLength checks that translation exceeding MaxLength is warned.
func TestValidate_MaxLength(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "e2",
			Source:      "Short",
			Translation: "متن بسیار طولانی که از حد مجاز فراتر می‌رود",
			Status:      core.StatusTranslated,
			MaxLength:   5,
		},
	}
	errs := v.Validate(entries)
	found := false
	for _, e := range errs {
		if e.Type == core.ValLengthExceeded && e.Severity == core.SevWarning {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ValLengthExceeded warning, got none")
	}
}

// TestValidate_SkipsUntranslated ensures untranslated entries are not validated.
func TestValidate_SkipsUntranslated(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "e3",
			Source:      "Hello {player}",
			Translation: "",
			Status:      core.StatusUntranslated,
		},
	}
	errs := v.Validate(entries)
	if len(errs) != 0 {
		t.Errorf("expected no errors for untranslated entry, got: %v", errs)
	}
}

// TestValidate_PersianLatinMix verifies Latin placeholders within Persian text.
func TestValidate_PersianLatinMix(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "e4",
			Source:      "Level {level}: score is %d",
			Translation: "سطح {level}: امتیاز %d است",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if len(errs) != 0 {
		t.Errorf("expected no errors for correctly preserved Persian/Latin mix, got: %v", errs)
	}
}

// TestValidate_PrintfVariants verifies complex format specifiers: %1$s, %02d, %.2f, %lld.
func TestValidate_PrintfVariants(t *testing.T) {
	v := New()
	tests := []struct {
		name        string
		src         string
		tr          string
		expectError bool
	}{
		{
			name:        "positional matching",
			src:         "User %1$s scored %2$d points",
			tr:          "کاربر %1$s امتیاز %2$d کسب کرد",
			expectError: false,
		},
		{
			name:        "precision and width",
			src:         "Time: %02d:%02d, ratio: %.2f%%",
			tr:          "زمان: %02d:%02d، نسبت: %.2f%%",
			expectError: false,
		},
		{
			name:        "missing positional arg",
			src:         "Player %1$s has %2$d gold",
			tr:          "بازیکن %1$s طلا دارد",
			expectError: true,
		},
		{
			name:        "large integer format %lld",
			src:         "ID: %lld bytes",
			tr:          "شناسه: %lld بایت",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []core.StringEntry{
				{
					ID:          "printf-test",
					Source:      tt.src,
					Translation: tt.tr,
					Status:      core.StatusTranslated,
				},
			}
			errs := v.Validate(entries)
			hasErr := HasErrors(errs)
			if hasErr != tt.expectError {
				t.Errorf("HasErrors() = %v, want %v (errs: %v)", hasErr, tt.expectError, errs)
			}
		})
	}
}

// TestValidate_NamedAndNumericFormatArgs tests {player}, {0}, {name:s}.
func TestValidate_NamedAndNumericFormatArgs(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "named-1",
			Source:      "Welcome {player_name:s}! You have {0} items.",
			Translation: "خوش آمدید {player_name:s}! شما {0} آیتم دارید.",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if HasErrors(errs) {
		t.Errorf("expected no errors for valid named/numeric args, got: %v", errs)
	}

	// Missing argument
	entries[0].Translation = "خوش آمدید! شما آیتم دارید."
	errs = v.Validate(entries)
	if !HasErrors(errs) {
		t.Errorf("expected error for missing named/numeric args, got none")
	}
}

// TestValidate_MarkupTags tests Rich Text tags (<color>, </color>) and BBCode ([b], [/b]).
func TestValidate_MarkupTags(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "markup-1",
			Source:      "<color=#ff0000>Warning!</color> [b]Careful[/b]",
			Translation: "<color=#ff0000>هشدار!</color> [b]مراقب باشید[/b]",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid markup, got: %v", errs)
	}

	// Corrupt markup: missing closing tag
	entries[0].Translation = "<color=#ff0000>هشدار! [b]مراقب باشید[/b]"
	errs = v.Validate(entries)
	if !HasErrors(errs) {
		t.Errorf("expected error for missing tag, got none")
	}
}

// TestValidate_PlaceholderOrderDifference verifies warning (not hard error) for order diff.
func TestValidate_PlaceholderOrderDifference(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "order-1",
			Source:      "{A} then {B}",
			Translation: "{B} سپس {A}",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if HasErrors(errs) {
		t.Errorf("order difference should be a warning, not a hard error: %v", errs)
	}
	warnings := FilterBySeverity(errs, core.SevWarning)
	if len(warnings) == 0 {
		t.Errorf("expected a warning for placeholder order diff, got none")
	}
}

// TestValidate_UnbalancedBraces tests unbalanced braces detection.
func TestValidate_UnbalancedBraces(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "brace-1",
			Source:      "Value {val}",
			Translation: "مقدار {val",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if !HasErrors(errs) {
		t.Errorf("expected hard error for unbalanced braces, got none")
	}
}

// TestValidate_EmptyTranslation verifies empty translation produces a warning.
func TestValidate_EmptyTranslation(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{
			ID:          "empty-1",
			Source:      "Non-empty source",
			Translation: "   ",
			Status:      core.StatusTranslated,
		},
	}
	errs := v.Validate(entries)
	if len(errs) == 0 {
		t.Fatalf("expected warning for empty translation, got none")
	}
	if errs[0].Type != core.ValEmptyTranslation {
		t.Errorf("type = %v, want ValEmptyTranslation", errs[0].Type)
	}
}

// TestValidate_DeterministicSorting ensures returned errors are stably sorted.
func TestValidate_DeterministicSorting(t *testing.T) {
	v := New()
	entries := []core.StringEntry{
		{ID: "z-entry", Source: "Missing {x}", Translation: "ترجمه", Status: core.StatusTranslated},
		{ID: "a-entry", Source: "Missing {y}", Translation: "ترجمه", Status: core.StatusTranslated},
		{ID: "m-entry", Source: "Short", Translation: "طولانییییییییییییی", MaxLength: 3, Status: core.StatusTranslated},
	}
	errs1 := v.Validate(entries)
	errs2 := v.Validate(entries)

	if len(errs1) != len(errs2) {
		t.Fatalf("length mismatch: %d vs %d", len(errs1), len(errs2))
	}
	for i := range errs1 {
		if errs1[i].EntryID != errs2[i].EntryID || errs1[i].Type != errs2[i].Type {
			t.Errorf("non-deterministic order at %d: %+v vs %+v", i, errs1[i], errs2[i])
		}
	}
	// "a-entry" must be first
	if errs1[0].EntryID != "a-entry" {
		t.Errorf("expected first error to be for a-entry, got %s", errs1[0].EntryID)
	}
}

// TestFmtMatches_EmptyMaps verifies "none" labels appear for empty maps.
func TestFmtMatches_EmptyMaps(t *testing.T) {
	out := fmtMatches(map[string]int{}, map[string]int{})
	if !strings.Contains(out, "none") {
		t.Errorf("fmtMatches(empty, empty) = %q; expected 'none' labels", out)
	}
}
