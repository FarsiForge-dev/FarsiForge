package fonts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// StageFontReplacement stages a custom replacement font under Project.WorkingDir/out/<targetRelPath>.
// It validates the font, captures original and new SHA-256 hashes, and guarantees that the
// installed game directory is NEVER modified during staging.
func StageFontReplacement(gameRoot, workDir, targetFontRelPath, customFontPath string) (*FontReplacementResult, error) {
	if gameRoot == "" || workDir == "" || targetFontRelPath == "" || customFontPath == "" {
		return nil, fmt.Errorf("all paths must be non-empty")
	}

	cleanRel := filepath.Clean(filepath.FromSlash(targetFontRelPath))
	if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
		return nil, fmt.Errorf("invalid game-relative font path %q", targetFontRelPath)
	}

	targetGamePath := filepath.Join(gameRoot, cleanRel)
	if !fileExists(targetGamePath) {
		return nil, fmt.Errorf("target font does not exist in game installation: %s", targetGamePath)
	}

	if !fileExists(customFontPath) {
		return nil, fmt.Errorf("custom replacement font file not found: %s", customFontPath)
	}

	// Validate custom font format and parse metadata
	parsed, err := ParseSFNT(customFontPath)
	if err != nil {
		return nil, fmt.Errorf("invalid replacement font: %w", err)
	}

	// Capture original font hash
	origHash, err := hashFile(targetGamePath)
	if err != nil {
		return nil, fmt.Errorf("hash original game font: %w", err)
	}

	// Capture custom font hash
	customHash, err := hashFile(customFontPath)
	if err != nil {
		return nil, fmt.Errorf("hash custom replacement font: %w", err)
	}

	// Stage under workDir/out/<cleanRel>
	stagedPath := filepath.Join(workDir, "out", cleanRel)
	if err := os.MkdirAll(filepath.Dir(stagedPath), 0o755); err != nil {
		return nil, fmt.Errorf("create staging font directory: %w", err)
	}

	if err := copyFile(customFontPath, stagedPath); err != nil {
		return nil, fmt.Errorf("stage font file: %w", err)
	}

	return &FontReplacementResult{
		TargetRelPath: filepath.ToSlash(cleanRel),
		StagedPath:    stagedPath,
		SourceSHA256:  origHash,
		TargetSHA256:  customHash,
		FontFamily:    parsed.Family,
	}, nil
}

// GenerateDiagnosticPreviewString returns a rich Persian test string for font rendering validation.
func GenerateDiagnosticPreviewString() string {
	return "فارسی‌ساز FarsiForge: ۱۲۳۴۵۶۷۸۹۰ — گچپژ آیا این متن به درستی و بدون جدایی حروف نمایش داده می‌شود؟"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
