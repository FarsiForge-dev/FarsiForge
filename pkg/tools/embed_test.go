package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteUnityScriptIncludesRawFieldHelper(t *testing.T) {
	for _, script := range []string{"extract.py", "inject.py", "inspect_asset.py"} {
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
