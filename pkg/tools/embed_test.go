package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteUnityScriptIncludesRawFieldHelper(t *testing.T) {
	for _, script := range []string{"extract.py", "inject.py"} {
		t.Run(script, func(t *testing.T) {
			dir := t.TempDir()
			path, err := WriteScript(script, dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("script was not written: %v", err)
			}
			helper := filepath.Join(dir, "unity_raw_fields.py")
			if _, err := os.Stat(helper); err != nil {
				t.Fatalf("Unity raw-field helper was not written: %v", err)
			}
		})
	}
}

func TestWriteScriptRejectsInvalidNames(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	for _, name := range []string{"", ".", "..", "../extract.py", "sub/extract.py", `sub\extract.py`} {
		if _, err := WriteScript(name, dir); err == nil {
			t.Errorf("WriteScript(%q) = nil error, want a validation error", name)
		}
	}

	// A valid name still works and lands directly inside destDir.
	path, err := WriteScript("extract.py", dir)
	if err != nil {
		t.Fatalf("WriteScript(extract.py) failed: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("WriteScript wrote to %q, want it inside %q", path, dir)
	}
}
