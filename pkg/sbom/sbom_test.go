package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashFile(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	content := "FarsiForge integrity test content"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hash, size, err := HashFile(tempFile)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), size)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64 char hex hash, got %d chars: %s", len(hash), hash)
	}
}

func TestGenerateManifestAndSerialization(t *testing.T) {
	tempDir := t.TempDir()

	// Create dummy binary
	binPath := filepath.Join(tempDir, "FarsiForge.exe")
	if err := os.WriteFile(binPath, []byte("dummy exe bytes"), 0755); err != nil {
		t.Fatalf("failed to write dummy bin: %v", err)
	}

	manifest, err := Generate(tempDir, "v1.2.0", "abcdef123456", "main")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if manifest.Application != "FarsiForge" {
		t.Errorf("expected Application FarsiForge, got %s", manifest.Application)
	}
	if manifest.Version != "v1.2.0" {
		t.Errorf("expected Version v1.2.0, got %s", manifest.Version)
	}
	if manifest.GitCommit != "abcdef123456" {
		t.Errorf("expected commit abcdef123456, got %s", manifest.GitCommit)
	}
	if len(manifest.Artifacts) == 0 {
		t.Errorf("expected at least 1 artifact in manifest")
	}

	// Test JSON write
	jsonPath := filepath.Join(tempDir, "sbom.json")
	if err := manifest.WriteJSON(jsonPath); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read written JSON: %v", err)
	}
	var loaded ReleaseManifest
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if loaded.Version != manifest.Version {
		t.Errorf("loaded version %s != %s", loaded.Version, manifest.Version)
	}

	// Test Markdown write
	mdPath := filepath.Join(tempDir, "RELEASE_MANIFEST.md")
	if err := manifest.WriteMarkdown(mdPath); err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read written Markdown: %v", err)
	}
	if !strings.Contains(string(mdContent), "# FarsiForge Release Manifest & SBOM — v1.2.0") {
		t.Errorf("markdown output missing title: %s", string(mdContent))
	}
}
