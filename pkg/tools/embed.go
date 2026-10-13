package tools

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// scriptsFS embeds the Python scripts shipped under ./scripts so that they are
// always available inside the compiled binary regardless of where the app is
// installed. go:embed cannot cross package directories, which is why the
// scripts live inside this package.
//
//go:embed all:scripts
var scriptsFS embed.FS

// WriteScript writes an embedded Python script (e.g. "extract.py") into destDir
// and returns the full path of the written file. If destDir does not exist it
// is created. The file is overwritten if it already exists.
func WriteScript(name, destDir string) (string, error) {
	// The script name comes from callers, but it is still validated so a
	// malformed name can never read outside the embedded scripts or write
	// outside destDir.
	if name == "" || name != filepath.Base(name) || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid script name: %q", name)
	}
	if destDir != "" {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return "", err
		}
	}

	data, err := scriptsFS.ReadFile(path.Join("scripts", name))
	if err != nil {
		return "", err
	}

	dest := name
	if destDir != "" {
		dest = filepath.Join(destDir, name)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	if name == "extract.py" || name == "inject.py" {
		helperData, err := scriptsFS.ReadFile("scripts/unity_raw_fields.py")
		if err != nil {
			return "", err
		}
		helperPath := "unity_raw_fields.py"
		if destDir != "" {
			helperPath = filepath.Join(destDir, helperPath)
		}
		if err := os.WriteFile(helperPath, helperData, 0o644); err != nil {
			return "", err
		}
	}
	return dest, nil
}

// ListScripts returns the names of the embedded Python scripts (e.g.
// "extract.py", "inject.py") sorted alphabetically.
func ListScripts() []string {
	entries, err := fs.ReadDir(scriptsFS, "scripts")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}
