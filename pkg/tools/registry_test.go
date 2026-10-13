package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveToolsDirUsesExplicitPathAsGiven(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if got := resolveToolsDir(dir); got != dir {
		t.Errorf("resolveToolsDir(%q) = %q, want the explicit path", dir, got)
	}

	// Relative overrides are resolved against the working directory so the
	// registry never returns a path that depends on the current directory
	// later on.
	rel := filepath.Join("tools", "nested")
	got := resolveToolsDir(rel)
	if !filepath.IsAbs(got) {
		t.Errorf("resolveToolsDir(%q) = %q, want an absolute path", rel, got)
	}
	if filepath.Base(got) != "nested" {
		t.Errorf("resolveToolsDir(%q) = %q, want it to keep the last element", rel, got)
	}
}

func TestGetPythonFallbackIsPlatformAppropriate(t *testing.T) {
	t.Setenv(EnvPythonOverride, "")
	reg := &Registry{}
	if got := reg.GetPython(); got != defaultPythonName() {
		t.Errorf("GetPython() = %q, want %q", got, defaultPythonName())
	}

	t.Setenv(EnvPythonOverride, filepath.Join("custom", "python3.11"))
	if got := reg.GetPython(); got != filepath.Join("custom", "python3.11") {
		t.Errorf("GetPython() with override = %q, want the override", got)
	}

	reg.Python = "/usr/bin/python3"
	if got := reg.GetPython(); got != "/usr/bin/python3" {
		t.Errorf("GetPython() = %q, want the discovered interpreter", got)
	}
}

func TestGetPathRejectsEscapingNames(t *testing.T) {
	t.Parallel()
	reg := &Registry{RootDir: t.TempDir()}

	for _, name := range []string{"../outside", "..", filepath.Join("..", "..", "etc", "passwd")} {
		if got := reg.GetPath(name); got != "" {
			t.Errorf("GetPath(%q) = %q, want empty for an escaping name", name, got)
		}
	}
	if got := reg.GetPath("CodeWalker/magic.dat"); got != filepath.Join(reg.RootDir, "CodeWalker", "magic.dat") {
		t.Errorf("GetPath(%q) = %q, want it resolved under the tools root", "CodeWalker/magic.dat", got)
	}
}

func TestFileExistsRequiresRegularFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if fileExists(dir) {
		t.Error("fileExists() must be false for a directory")
	}
	file := filepath.Join(dir, "tool.exe")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(file) {
		t.Error("fileExists() must be true for a regular file")
	}
	if fileExists(filepath.Join(dir, "missing.exe")) {
		t.Error("fileExists() must be false for a missing file")
	}
}
