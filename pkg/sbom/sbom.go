package sbom

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"farsiforge/pkg/tools"
)

// ToolSBOMEntry describes an external tool bundled or utilized in a release.
type ToolSBOMEntry struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	License      string `json:"license"`
	Architecture string `json:"architecture"`
	Path         string `json:"path,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Source       string `json:"source"`
}

// ArtifactSBOMEntry represents a built distributable file and its integrity hash.
type ArtifactSBOMEntry struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// ReleaseManifest contains all provenance, runtime, dependency, and artifact
// information needed for a verifiable release.
type ReleaseManifest struct {
	Application string              `json:"application"`
	Version     string              `json:"version"`
	GitCommit   string              `json:"git_commit"`
	GitBranch   string              `json:"git_branch"`
	BuildTime   string              `json:"build_time"`
	GoVersion   string              `json:"go_version"`
	NodeVersion string              `json:"node_version,omitempty"`
	OS          string              `json:"os"`
	Arch        string              `json:"arch"`
	Tools       []ToolSBOMEntry     `json:"tools"`
	Artifacts   []ArtifactSBOMEntry `json:"artifacts"`
}

// HashFile calculates the SHA-256 hex digest of a file.
func HashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// Generate creates a release manifest by inspecting the given distribution binary folder.
func Generate(binDir string, version, commit, branch string) (*ReleaseManifest, error) {
	manifest := &ReleaseManifest{
		Application: "FarsiForge",
		Version:     version,
		GitCommit:   commit,
		GitBranch:   branch,
		BuildTime:   time.Now().UTC().Format(time.RFC3339),
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Tools:       make([]ToolSBOMEntry, 0),
		Artifacts:   make([]ArtifactSBOMEntry, 0),
	}

	if manifest.Version == "" {
		manifest.Version = "v1.0.0-dev"
	}

	// 1. Scan external tools from binDir/Tools or default tools folder
	toolsDir := filepath.Join(binDir, "Tools")
	if _, err := os.Stat(toolsDir); err != nil {
		toolsDir = filepath.Join(binDir, "..", "Tools")
	}

	reg, _ := tools.NewRegistry(toolsDir)
	report := tools.GenerateDiagnosticReport(reg)
	for _, t := range report.Tools {
		toolEntry := ToolSBOMEntry{
			Name:         t.Name,
			Version:      t.Version,
			License:      t.License,
			Architecture: t.Architecture,
			Source:       t.Source,
		}
		if t.Path != "" && t.Found {
			rel, err := filepath.Rel(binDir, t.Path)
			if err == nil {
				toolEntry.Path = rel
			} else {
				toolEntry.Path = filepath.Base(t.Path)
			}
			if hash, _, err := HashFile(t.Path); err == nil {
				toolEntry.SHA256 = hash
			}
		}
		manifest.Tools = append(manifest.Tools, toolEntry)
	}

	// 2. Hash distributable artifacts inside binDir
	if binDir != "" && dirExists(binDir) {
		entries, err := os.ReadDir(binDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				// Only hash executable / zip / manifest files
				if strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".dll") {
					fullPath := filepath.Join(binDir, name)
					hash, size, hErr := HashFile(fullPath)
					if hErr == nil {
						manifest.Artifacts = append(manifest.Artifacts, ArtifactSBOMEntry{
							Name:      name,
							SizeBytes: size,
							SHA256:    hash,
						})
					}
				}
			}
		}
	}

	return manifest, nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// WriteJSON writes the release manifest as formatted JSON.
func (m *ReleaseManifest) WriteJSON(destPath string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0644)
}

// ToMarkdown generates human-readable release manifest markdown.
func (m *ReleaseManifest) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# FarsiForge Release Manifest & SBOM — %s\n\n", m.Version))
	sb.WriteString(fmt.Sprintf("- **Application:** %s\n", m.Application))
	sb.WriteString(fmt.Sprintf("- **Git Commit:** `%s`\n", m.GitCommit))
	sb.WriteString(fmt.Sprintf("- **Git Branch:** `%s`\n", m.GitBranch))
	sb.WriteString(fmt.Sprintf("- **Build Time (UTC):** `%s`\n", m.BuildTime))
	sb.WriteString(fmt.Sprintf("- **Go Compiler:** `%s`\n", m.GoVersion))
	sb.WriteString(fmt.Sprintf("- **Target Platform:** `%s / %s`\n\n", m.OS, m.Arch))

	sb.WriteString("## Release Artifacts & Checksums (SHA-256)\n\n")
	if len(m.Artifacts) > 0 {
		sb.WriteString("| Artifact | Size (Bytes) | SHA-256 Checksum |\n")
		sb.WriteString("| :--- | :---: | :--- |\n")
		for _, a := range m.Artifacts {
			sb.WriteString(fmt.Sprintf("| **%s** | %d | `%s` |\n", a.Name, a.SizeBytes, a.SHA256))
		}
	} else {
		sb.WriteString("_No binary artifacts recorded._\n")
	}

	sb.WriteString("\n## Bundled Helper Tools & Dependencies\n\n")
	sb.WriteString("| Tool | Version | License | Architecture | SHA-256 |\n")
	sb.WriteString("| :--- | :--- | :--- | :---: | :--- |\n")
	for _, t := range m.Tools {
		hash := t.SHA256
		if len(hash) > 16 {
			hash = hash[:16] + "..."
		}
		if hash == "" {
			hash = "N/A"
		}
		sb.WriteString(fmt.Sprintf("| **%s** | %s | %s | `%s` | `%s` |\n",
			t.Name, t.Version, t.License, t.Architecture, hash))
	}

	return sb.String()
}

// WriteMarkdown writes the release manifest as formatted Markdown.
func (m *ReleaseManifest) WriteMarkdown(destPath string) error {
	return os.WriteFile(destPath, []byte(m.ToMarkdown()), 0644)
}
