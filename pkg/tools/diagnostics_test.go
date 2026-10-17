package tools

import (
	"debug/pe"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createMinimalPE(machine uint16) []byte {
	b := make([]byte, 0x200)
	// DOS header
	b[0] = 'M'
	b[1] = 'Z'
	// e_lfanew at 0x3C -> 0x80
	b[0x3c] = 0x80
	// PE signature at 0x80
	b[0x80] = 'P'
	b[0x81] = 'E'
	b[0x82] = 0
	b[0x83] = 0
	// File header at 0x84
	b[0x84] = byte(machine)
	b[0x85] = byte(machine >> 8)
	// NumberOfSections at 0x86: 0
	// TimeDateStamp at 0x88: 0
	// SizeOfOptionalHeader at 0x94: 0
	return b
}

func TestPEArchitectureDetection(t *testing.T) {
	tempDir := t.TempDir()

	// 1. AMD64
	amd64Path := filepath.Join(tempDir, "amd64.exe")
	if err := os.WriteFile(amd64Path, createMinimalPE(pe.IMAGE_FILE_MACHINE_AMD64), 0755); err != nil {
		t.Fatalf("failed to write amd64 pe: %v", err)
	}
	arch, err := DetectPEArchitecture(amd64Path)
	if err != nil {
		t.Fatalf("DetectPEArchitecture failed: %v", err)
	}
	if arch != "x86_64" {
		t.Errorf("expected x86_64, got %s", arch)
	}

	// 2. I386
	i386Path := filepath.Join(tempDir, "i386.exe")
	if err := os.WriteFile(i386Path, createMinimalPE(pe.IMAGE_FILE_MACHINE_I386), 0755); err != nil {
		t.Fatalf("failed to write i386 pe: %v", err)
	}
	arch, err = DetectPEArchitecture(i386Path)
	if err != nil {
		t.Fatalf("DetectPEArchitecture failed: %v", err)
	}
	if arch != "x86" {
		t.Errorf("expected x86, got %s", arch)
	}

	// 3. Corrupt
	corruptPath := filepath.Join(tempDir, "corrupt.exe")
	if err := os.WriteFile(corruptPath, []byte("not a valid pe"), 0755); err != nil {
		t.Fatalf("failed to write corrupt pe: %v", err)
	}
	_, err = DetectPEArchitecture(corruptPath)
	if err == nil {
		t.Errorf("expected error on corrupt PE")
	}
}

func TestCheckOperationRequirements(t *testing.T) {
	tempTools := t.TempDir()

	// 1. Empty registry (tools missing)
	reg, err := NewRegistry(tempTools)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Test Unreal extract without repak
	resUnreal := CheckOperationRequirements(reg, "unreal", "extract")
	if resUnreal.Allowed {
		t.Errorf("expected unreal extract to be disallowed when repak missing")
	}
	if len(resUnreal.Blockers) == 0 {
		t.Errorf("expected blockers for missing repak")
	}

	// Test Godot extract without fftools/gdre
	resGodot := CheckOperationRequirements(reg, "godot", "extract")
	if resGodot.Allowed {
		t.Errorf("expected godot extract to be disallowed when fftools and gdre_tools missing")
	}

	// Test Build Patch without patcher
	resPatchEmpty := CheckOperationRequirements(reg, "general", "build_patch")
	if resPatchEmpty.Allowed {
		t.Errorf("expected build_patch to be disallowed when patcher missing")
	}

	// 2. Populate tools with valid PE binaries
	mustWritePE := func(relPath string, machine uint16) {
		full := filepath.Join(tempTools, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("failed to mkdir: %v", err)
		}
		if err := os.WriteFile(full, createMinimalPE(machine), 0o755); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	mustWritePE("patcher/FarsiForgePatcher.exe", pe.IMAGE_FILE_MACHINE_AMD64)
	mustWritePE("FromSoftware/oo2core_6_win64.dll", pe.IMAGE_FILE_MACHINE_AMD64)

	regWithPatcher, err := NewRegistry(tempTools)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	resPatchOK := CheckOperationRequirements(regWithPatcher, "general", "build_patch")
	if !resPatchOK.Allowed {
		t.Errorf("expected build_patch to be allowed with valid patcher present: %v", resPatchOK.Blockers)
	}

	// 3. Test Corrupt Tool Detection
	mustWriteCorrupt := func(relPath string) {
		full := filepath.Join(tempTools, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("failed to mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("NOT_A_PE_HEADER"), 0o755); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	mustWriteCorrupt("patcher/FarsiForgePatcher.exe")
	regCorrupt, _ := NewRegistry(tempTools)
	resPatchCorrupt := CheckOperationRequirements(regCorrupt, "general", "build_patch")
	if resPatchCorrupt.Allowed {
		t.Errorf("expected build_patch to be disallowed with corrupt patcher")
	}
}

func TestGenerateDiagnosticReport(t *testing.T) {
	tempTools := t.TempDir()
	reg, _ := NewRegistry(tempTools)

	report := GenerateDiagnosticReport(reg)
	if report == nil {
		t.Fatalf("expected non-nil report")
	}

	if len(report.Tools) == 0 {
		t.Errorf("expected tools in report")
	}

	md := report.ToMarkdown()
	if !strings.Contains(md, "# FarsiForge Tool & Environment Diagnostic Report") {
		t.Errorf("missing title in markdown report: %s", md)
	}
	if !strings.Contains(md, "| Tool | Category | Status |") {
		t.Errorf("missing matrix table in markdown report: %s", md)
	}
}

func TestDetectPEArchitecture_NonExistent(t *testing.T) {
	_, err := DetectPEArchitecture("C:\\non_existent_file_xyz.exe")
	if err == nil {
		t.Errorf("expected error for non-existent file")
	}
}
