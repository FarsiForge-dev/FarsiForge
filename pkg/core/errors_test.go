package core

import (
	"errors"
	"strings"
	"testing"
)

func TestFarsiForgeErrorFormatting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  *FarsiForgeError
		want string
	}{
		{
			name: "op and message only",
			err:  NewError("detect", "nothing found"),
			want: "[detect]: nothing found",
		},
		{
			name: "with engine",
			err:  &FarsiForgeError{Op: "extract", Engine: "unity", Message: "boom"},
			want: "[extract] engine=unity: boom",
		},
		{
			name: "with file and underlying error",
			err:  WrapFile("inject", "data.pak", errors.New("disk full"), "write failed"),
			want: "[inject] file=data.pak: write failed (disk full)",
		},
		{
			name: "engine and file both present",
			err:  &FarsiForgeError{Op: "inject", Engine: "unreal", File: "a.uasset", Message: "m"},
			want: "[inject] engine=unreal file=a.uasset: m",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWrapPreservesErrorChain(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	wrapped := Wrap("extract", sentinel, "outer")

	if !errors.Is(wrapped, sentinel) {
		t.Fatalf("errors.Is should find the wrapped sentinel")
	}
	if wrapped.Unwrap() != sentinel {
		t.Fatalf("Unwrap() = %v, want sentinel", wrapped.Unwrap())
	}

	var ffe *FarsiForgeError
	if !errors.As(error(wrapped), &ffe) {
		t.Fatalf("errors.As should match *FarsiForgeError")
	}
	if ffe.Op != "extract" || ffe.Message != "outer" {
		t.Fatalf("unexpected fields: op=%q msg=%q", ffe.Op, ffe.Message)
	}

	// A nil underlying error must not be rendered.
	if got := NewError("x", "y").Error(); strings.Contains(got, "(") {
		t.Fatalf("nil Err should not add a parenthesised suffix, got %q", got)
	}
}

func TestErrorConstructors(t *testing.T) {
	t.Parallel()

	t.Run("ErrToolNotFound names tool and offers fixes", func(t *testing.T) {
		t.Parallel()
		err := ErrToolNotFound("UABEA")
		if err.Op != "tools" {
			t.Fatalf("Op = %q, want tools", err.Op)
		}
		if !strings.Contains(err.Error(), "UABEA") {
			t.Fatalf("message should mention the tool: %q", err.Error())
		}
		if len(err.Fixes) == 0 {
			t.Fatalf("expected at least one fix suggestion")
		}
		if !strings.Contains(err.Fixes[0], "UABEA") {
			t.Fatalf("first fix should name the tool: %q", err.Fixes[0])
		}
		if err.CanRetry {
			t.Fatalf("missing tool should not be marked retryable")
		}
	})

	t.Run("ErrExtractionFailed is retryable and wraps cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("parse error")
		err := ErrExtractionFailed("unity", cause)
		if !err.CanRetry {
			t.Fatalf("expected CanRetry=true")
		}
		if err.Engine != "unity" || err.Op != "extract" {
			t.Fatalf("unexpected op/engine: %q/%q", err.Op, err.Engine)
		}
		if !errors.Is(err, cause) {
			t.Fatalf("cause should be reachable via errors.Is")
		}
	})

	t.Run("ErrInjectionFailed is retryable with fixes", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("permission denied")
		err := ErrInjectionFailed("unreal", cause)
		if !err.CanRetry {
			t.Fatalf("expected CanRetry=true")
		}
		if len(err.Fixes) < 1 {
			t.Fatalf("expected fix suggestions")
		}
		if !errors.Is(err, cause) {
			t.Fatalf("cause should be reachable via errors.Is")
		}
		if !strings.HasPrefix(err.Error(), "[inject] engine=unreal:") {
			t.Fatalf("unexpected prefix: %q", err.Error())
		}
	})

	t.Run("ErrEngineNotDetected is a stable sentinel", func(t *testing.T) {
		t.Parallel()
		wrapped := Wrap("pipeline", ErrEngineNotDetected, "detection stage failed")
		if !errors.Is(wrapped, ErrEngineNotDetected) {
			t.Fatalf("sentinel should survive wrapping")
		}
		if ErrEngineNotDetected.Op != "detect" {
			t.Fatalf("Op = %q, want detect", ErrEngineNotDetected.Op)
		}
	})
}

func TestDefaultPersianOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultPersianOptions()
	if opts == (PersianOptions{}) {
		t.Fatalf("defaults must not be the zero value; the CLI uses the zero value as 'unset'")
	}
	if !opts.Reshape || !opts.BidiReorder || !opts.FixYeh || !opts.PersianDigits {
		t.Fatalf("expected reshape/bidi/fix-yeh/digits enabled by default, got %+v", opts)
	}
	if opts.ConvertPunct || opts.DropDiacritics {
		t.Fatalf("punctuation conversion and diacritic dropping should be off by default, got %+v", opts)
	}
}
