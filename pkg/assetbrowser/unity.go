package assetbrowser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"farsiforge/pkg/core"
	"farsiforge/pkg/tools"
)

// InspectUnityAsset runs inspect_asset.py to inspect the serialized objects inside a Unity asset file.
func InspectUnityAsset(ctx context.Context, gameRoot, relPath string, page, pageSize int, reg core.ToolRegistry) (*AssetInspectionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cleanRel := filepath.Clean(filepath.ToSlash(relPath))
	if strings.HasPrefix(cleanRel, "..") || filepath.IsAbs(cleanRel) {
		return nil, fmt.Errorf("invalid asset relative path: %s", relPath)
	}

	fullPath := filepath.Join(gameRoot, filepath.FromSlash(cleanRel))
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("asset file not found: %s", fullPath)
	}

	if reg == nil {
		var initErr error
		reg, initErr = tools.NewRegistry("")
		if initErr != nil {
			return nil, fmt.Errorf("tool registry unavailable: %w", initErr)
		}
	}

	py := reg.GetPython()
	if py == "" {
		return nil, fmt.Errorf("python is required for Unity compiled asset inspection")
	}

	tempDir, err := os.MkdirTemp("", "farsiforge_inspect_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	scriptPath, err := tools.WriteScript("inspect_asset.py", tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to write inspect_asset.py: %w", err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	out, err := tools.RunSilent(
		ctxTimeout,
		tempDir,
		py,
		scriptPath,
		fullPath,
		strconv.Itoa(page),
		strconv.Itoa(pageSize),
	)
	if err != nil {
		return nil, fmt.Errorf("asset inspector failed: %w; output: %s", err, out)
	}

	var res AssetInspectionResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
		return nil, fmt.Errorf("failed to parse asset inspector output: %w; raw: %s", err, out)
	}

	res.FileRelPath = cleanRel
	return &res, nil
}
