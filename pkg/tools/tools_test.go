package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolsRegistryDiscoveryAndGetPath(t *testing.T) {
	tempTools := t.TempDir()

	// Create dummy tool executables and directories
	mustWrite := func(relPath, content string) {
		full := filepath.Join(tempTools, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("failed to mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	mustWrite("repak/repak.exe", "dummy repak")
	mustWrite("retoc/retoc.exe", "dummy retoc")
	mustWrite("fftools/fftools.exe", "dummy fftools")
	mustWrite("patcher/FarsiForgePatcher.exe", "dummy patcher")
	mustWrite("gdre_tools/gdre_tools.exe", "dummy gdre")
	mustWrite("CodeWalker/magic.dat", "dummy magic")
	mustWrite("vazirmatn/Vazirmatn-Regular.ttf", "dummy font")
	mustWrite("die/diec.exe", "dummy die")
	mustWrite("Il2CppDumper/Il2CppDumper.exe", "dummy il2cpp")
	mustWrite("AssetRipper/AssetRipper.exe", "dummy assetripper")
	mustWrite("UABEA/UABEA.exe", "dummy uabea")
	mustWrite("FModel/FModel.exe", "dummy fmodel")

	reg, err := NewRegistry(tempTools)
	if err != nil {
		t.Fatalf("NewRegistry returned unexpected error: %v", err)
	}

	// Verify discovery
	expectedTools := []string{
		"repak",
		"retoc",
		"fftools",
		"patcher",
		"gdre_tools",
		"codewalker_magic",
		"vazirmatn_font",
		"die",
		"il2cpp_dumper",
		"asset_ripper",
		"uabea",
		"fmodel",
	}

	for _, name := range expectedTools {
		if !reg.IsAvailable(name) {
			t.Errorf("expected tool %q to be available in registry", name)
		}
	}

	// Verify GetPath returns actual files rather than directory paths
	testPaths := []struct {
		lookup string
		expect string
	}{
		{"repak", filepath.Join(tempTools, "repak", "repak.exe")},
		{"retoc", filepath.Join(tempTools, "retoc", "retoc.exe")},
		{"fftools", filepath.Join(tempTools, "fftools", "fftools.exe")},
		{"ff_tools", filepath.Join(tempTools, "fftools", "fftools.exe")},
		{"patcher", filepath.Join(tempTools, "patcher", "FarsiForgePatcher.exe")},
		{"farsiforgepatcher", filepath.Join(tempTools, "patcher", "FarsiForgePatcher.exe")},
		{"gdre_tools", filepath.Join(tempTools, "gdre_tools", "gdre_tools.exe")},
		{"gdre", filepath.Join(tempTools, "gdre_tools", "gdre_tools.exe")},
		{"codewalker_magic", filepath.Join(tempTools, "CodeWalker", "magic.dat")},
		{"magic.dat", filepath.Join(tempTools, "CodeWalker", "magic.dat")},
		{"vazirmatn", filepath.Join(tempTools, "vazirmatn", "Vazirmatn-Regular.ttf")},
		{"die", filepath.Join(tempTools, "die", "diec.exe")},
		{"il2cpp_dumper", filepath.Join(tempTools, "Il2CppDumper", "Il2CppDumper.exe")},
		{"asset_ripper", filepath.Join(tempTools, "AssetRipper", "AssetRipper.exe")},
		{"uabea", filepath.Join(tempTools, "UABEA", "UABEA.exe")},
		{"fmodel", filepath.Join(tempTools, "FModel", "FModel.exe")},
	}

	for _, tc := range testPaths {
		got := reg.GetPath(tc.lookup)
		if !strings.EqualFold(got, tc.expect) {
			t.Errorf("GetPath(%q) = %q; expected %q", tc.lookup, got, tc.expect)
		}
	}

	summary := reg.Summary()
	if !strings.Contains(summary, "Tools directory:") {
		t.Errorf("Summary output missing header: %s", summary)
	}
}

func TestResolveToolsDirEnvVar(t *testing.T) {
	tempTools := t.TempDir()
	t.Setenv("FARSIFORGE_TOOLS_DIR", tempTools)

	resolved := resolveToolsDir("")
	if resolved != tempTools {
		t.Fatalf("resolveToolsDir() = %q; expected %q from FARSIFORGE_TOOLS_DIR", resolved, tempTools)
	}
}

func TestResolveToolsDirFallbackTypoEnvVar(t *testing.T) {
	tempTools := t.TempDir()
	t.Setenv("FARSIFORGE_TOOLS_DIR", "")
	t.Setenv("FARISIFORGE_TOOLS_DIR", tempTools)

	resolved := resolveToolsDir("")
	if resolved != tempTools {
		t.Fatalf("resolveToolsDir() = %q; expected %q from FARISIFORGE_TOOLS_DIR", resolved, tempTools)
	}
}
