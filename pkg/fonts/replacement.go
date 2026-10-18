package fonts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"farsiforge/pkg/tools"
)

// StageFontReplacement stages a custom replacement font under Project.WorkingDir/out/<targetRelPath>.
// It validates the font, captures original and new SHA-256 hashes, and guarantees that the
// installed game directory is NEVER modified during staging.
func StageFontReplacement(gameRoot, workDir, targetFontRelPath, customFontPath string) (*FontReplacementResult, error) {
	if gameRoot == "" || workDir == "" || targetFontRelPath == "" || customFontPath == "" {
		return nil, fmt.Errorf("all paths must be non-empty")
	}

	var assetFileRel string
	var pathIDStr string
	isCompiledAsset := strings.Contains(targetFontRelPath, "#")
	if isCompiledAsset {
		parts := strings.SplitN(targetFontRelPath, "#", 2)
		assetFileRel = parts[0]
		pathIDStr = parts[1]
	} else {
		assetFileRel = targetFontRelPath
	}

	cleanRel := filepath.Clean(filepath.FromSlash(assetFileRel))
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

	// Capture original font / asset hash
	origHash, err := hashFile(targetGamePath)
	if err != nil {
		return nil, fmt.Errorf("hash original game font: %w", err)
	}

	// Stage under workDir/out/<cleanRel>
	stagedPath := filepath.Join(workDir, "out", cleanRel)
	if err := os.MkdirAll(filepath.Dir(stagedPath), 0o755); err != nil {
		return nil, fmt.Errorf("create staging font directory: %w", err)
	}

	if isCompiledAsset {
		if err := replaceCompiledUnityFont(targetGamePath, pathIDStr, customFontPath, stagedPath); err != nil {
			return nil, fmt.Errorf("replace compiled Unity font: %w", err)
		}
	} else {
		if err := copyFile(customFontPath, stagedPath); err != nil {
			return nil, fmt.Errorf("stage font file: %w", err)
		}
	}

	stagedHash, err := hashFile(stagedPath)
	if err != nil {
		return nil, fmt.Errorf("hash staged font: %w", err)
	}

	return &FontReplacementResult{
		TargetRelPath: filepath.ToSlash(cleanRel),
		StagedPath:    stagedPath,
		SourceSHA256:  origHash,
		TargetSHA256:  stagedHash,
		FontFamily:    parsed.Family,
	}, nil
}

func replaceCompiledUnityFont(assetPath, pathID, customFontPath, stagedAssetPath string) error {
	reg, err := tools.NewRegistry("")
	if err != nil {
		return err
	}
	py := reg.GetPython()
	if py == "" {
		return fmt.Errorf("python required")
	}

	tempDir, err := os.MkdirTemp("", "farsiforge_font_replace_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	scriptPath, err := tools.WriteScript("inspect_asset.py", tempDir)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	out, err := tools.RunSilent(ctx, tempDir, py, scriptPath, "--replace-font", assetPath, pathID, customFontPath, stagedAssetPath)
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	if !strings.Contains(out, `"success": true`) {
		return fmt.Errorf("replace font failed: %s", out)
	}
	return nil
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
