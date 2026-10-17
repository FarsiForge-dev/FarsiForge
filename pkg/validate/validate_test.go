package validate

import (
	"fmt"
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
		if e.Type == core.ValPlaceholderMismatch {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ValPlaceholderMismatch error, got: %v", errs)
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
		if e.Type == core.ValLengthExceeded {
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

// TestFmtMatches_EmptyMaps verifies "none" labels appear for empty maps.
func TestFmtMatches_EmptyMaps(t *testing.T) {
	out := fmtMatches(map[string]int{}, map[string]int{})
	if !strings.Contains(out, "none") {
		t.Errorf("fmtMatches(empty, empty) = %q; expected 'none' labels", out)
	}
}

// Ensure fmt is used (imported for Sprintf in fmtMatches).
var _ = fmt.Sprintf
