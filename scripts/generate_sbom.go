package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"farsiforge/pkg/sbom"
)

func getGitInfo() (commit string, branch string) {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err == nil {
		commit = strings.TrimSpace(string(out))
	}
	out, err = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		branch = strings.TrimSpace(string(out))
	}
	return
}

func main() {
	binDir := flag.String("bin", "build/bin", "Directory containing built release binaries")
	outDir := flag.String("out", "build/bin", "Directory where sbom.json and RELEASE_MANIFEST.md will be saved")
	version := flag.String("version", "", "Release version tag (e.g. v1.0.0)")
	commit := flag.String("commit", "", "Git commit SHA")
	branch := flag.String("branch", "", "Git branch name")
	flag.Parse()

	gitCommit, gitBranch := getGitInfo()
	if *commit == "" {
		*commit = gitCommit
	}
	if *branch == "" {
		*branch = gitBranch
	}
	if *version == "" {
		*version = os.Getenv("GITHUB_REF_NAME")
		if *version == "" {
			*version = "v1.0.0-dev"
		}
	}

	fmt.Printf("Generating FarsiForge Release Manifest & SBOM (%s)...\n", *version)
	manifest, err := sbom.Generate(*binDir, *version, *commit, *branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating SBOM: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		os.Exit(1)
	}

	jsonPath := filepath.Join(*outDir, "sbom.json")
	if err := manifest.WriteJSON(jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing sbom.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully generated: %s\n", jsonPath)

	mdPath := filepath.Join(*outDir, "RELEASE_MANIFEST.md")
	if err := manifest.WriteMarkdown(mdPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing RELEASE_MANIFEST.md: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully generated: %s\n", mdPath)
}
