package inject

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"farsiforge/pkg/core"
	"farsiforge/pkg/extract"
)

// FromSoftwareInjector injects translated Persian strings into FromSoftware
// FMG tables, BND4/BND3 binders, and DCX archives.
type FromSoftwareInjector struct{}

func (i *FromSoftwareInjector) SupportedEngine() string { return "fromsoftware" }
func (i *FromSoftwareInjector) Capabilities() core.InjectorCaps {
	return core.InjectorCaps{
		TextInjection:     true,
		NeedsExternalTool: false,
	}
}

func (i *FromSoftwareInjector) Inject(ctx context.Context, info *core.GameInfo, proj *core.Project, reg core.ToolRegistry, opts core.PersianOptions) (*core.InjectionResult, error) {
	res := &core.InjectionResult{}
	translatedEntries := proj.FindTranslated()
	if len(translatedEntries) == 0 {
		return res, core.NewError("inject", "no translated entries are ready for injection")
	}

	// Verify entries: archive-nested entries (Container != "") are read-only and cannot be injected
	for _, entry := range translatedEntries {
		if entry.Container != "" {
			return res, core.NewError("inject", fmt.Sprintf(
				"entry %s (%s) is inside archive %q; BHD5/BDT archive write-back is not supported. Use loose files or a mod loader for injection",
				entry.ID, entry.File, entry.Container,
			))
		}
	}

	groups := make(map[string]map[string][]core.StringEntry) // outerFile -> innerFMG -> entries
	for _, entry := range translatedEntries {
		rawFile := strings.TrimPrefix(filepath.ToSlash(entry.File), "/")
		outer := rawFile
		inner := ""

		for _, marker := range []string{".msgbnd.dcx/", ".msgbnd/", ".bnd4/", ".bnd3/", ".bnd/"} {
			if idx := strings.Index(strings.ToLower(rawFile), marker); idx >= 0 {
				outer = rawFile[:idx+len(marker)-1]
				inner = rawFile[idx+len(marker):]
				break
			}
		}

		if groups[outer] == nil {
			groups[outer] = make(map[string][]core.StringEntry)
		}
		groups[outer][inner] = append(groups[outer][inner], entry)
	}

	for outerRel, inners := range groups {
		cleanOuter, err := cleanGameRelativePath(outerRel)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("invalid path %s: %v", outerRel, err))
			continue
		}
		sourcePath, err := gameFilePath(info.GameRoot, cleanOuter)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("target path %s: %v", cleanOuter, err))
			continue
		}

		data, err := os.ReadFile(sourcePath)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("read source file %s: %v", cleanOuter, err))
			continue
		}

		// Capture original hash BEFORE any transformation.
		origHash := sha256File(data)

		// Compute staged output path under WorkingDir/out/ — never write to GameRoot.
		stagedPath, err := stagedOutputPath(proj.WorkingDir, cleanOuter)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("staged path %s: %v", cleanOuter, err))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(stagedPath), 0o755); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("create staged dir %s: %v", cleanOuter, err))
			continue
		}

		isDCX := strings.HasSuffix(strings.ToLower(cleanOuter), ".dcx")
		isBND := strings.Contains(strings.ToLower(cleanOuter), ".bnd") || strings.Contains(strings.ToLower(cleanOuter), ".msgbnd")

		if !isBND {
			// Loose FMG — write to staged path only.
			entries := inners[""]
			if len(entries) == 0 {
				for _, list := range inners {
					entries = append(entries, list...)
				}
			}
			fmgData := data
			if isDCX {
				decomp, err := extract.DecompressDCX(data)
				if err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("decompress %s: %v", cleanOuter, err))
					continue
				}
				fmgData = decomp
			}
			updatedFMG, count, err := injectFMGData(fmgData, entries, opts)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("inject FMG %s: %v", cleanOuter, err))
				continue
			}
			outData := updatedFMG
			if isDCX {
				comp, err := extract.CompressDCXDFLT(updatedFMG)
				if err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("compress DCX %s: %v", cleanOuter, err))
					continue
				}
				outData = comp
			}
			// Write to staged output, NOT to GameRoot.
			if err := atomicWriteFile(stagedPath, outData, 0o644); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("write staged %s: %v", cleanOuter, err))
				continue
			}
			slashRel := filepath.ToSlash(cleanOuter)
			res.ModifiedFiles = append(res.ModifiedFiles, slashRel)
			if proj.ModifiedFileHashes == nil {
				proj.ModifiedFileHashes = make(map[string]string)
			}
			proj.ModifiedFileHashes[slashRel] = origHash
			res.StringCount += count
			continue
		}

		// Container (BND4 or BND3)
		bndData := data
		if isDCX {
			decomp, err := extract.DecompressDCX(data)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("decompress %s: %v", cleanOuter, err))
				continue
			}
			bndData = decomp
		}

		isBND3 := len(bndData) >= 4 && string(bndData[:4]) == "BND3"
		var bndFiles []extract.BND4File
		if isBND3 {
			bf, err := extract.ParseBND3(bndData)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("parse BND3 %s: %v", cleanOuter, err))
				continue
			}
			bndFiles = bf
		} else {
			bf, err := extract.ParseBND4(bndData)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("parse BND4 %s: %v", cleanOuter, err))
				continue
			}
			bndFiles = bf
		}

		totalContainerInjected := 0
		for i := range bndFiles {
			innerName := filepath.Base(bndFiles[i].Name)
			// Check if we have translations for this inner file (by full name or basename)
			var targetEntries []core.StringEntry
			for key, list := range inners {
				if strings.EqualFold(key, bndFiles[i].Name) || strings.EqualFold(key, innerName) {
					targetEntries = append(targetEntries, list...)
				}
			}
			if len(targetEntries) == 0 {
				continue
			}
			updatedFMG, count, err := injectFMGData(bndFiles[i].Data, targetEntries, opts)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("inject inner %s in %s: %v", bndFiles[i].Name, cleanOuter, err))
				continue
			}
			bndFiles[i].Data = updatedFMG
			totalContainerInjected += count
		}

		if totalContainerInjected == 0 {
			continue
		}

		var repacked []byte
		if isBND3 {
			rb, err := extract.WriteBND3(bndFiles, extract.DefaultBND3Options())
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("repack BND3 %s: %v", cleanOuter, err))
				continue
			}
			repacked = rb
		} else {
			rb, err := extract.WriteBND4(bndFiles, extract.DefaultBND4Options())
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("repack BND4 %s: %v", cleanOuter, err))
				continue
			}
			repacked = rb
		}

		outData := repacked
		if isDCX {
			comp, err := extract.CompressDCXDFLT(repacked)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("compress DCX %s: %v", cleanOuter, err))
				continue
			}
			outData = comp
		}

		// Write to staged output, NOT to GameRoot.
		if err := atomicWriteFile(stagedPath, outData, 0o644); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("write staged %s: %v", cleanOuter, err))
			continue
		}
		slashRel := filepath.ToSlash(cleanOuter)
		res.ModifiedFiles = append(res.ModifiedFiles, slashRel)
		if proj.ModifiedFileHashes == nil {
			proj.ModifiedFileHashes = make(map[string]string)
		}
		proj.ModifiedFileHashes[slashRel] = origHash
		res.StringCount += totalContainerInjected
	}

	return res, nil
}

// stagedOutputPath returns the path under WorkingDir/out/ mirroring the
// game-relative file path. This ensures injectors never write back to GameRoot.
func stagedOutputPath(workingDir, gameRelPath string) (string, error) {
	if workingDir == "" {
		return "", fmt.Errorf("project WorkingDir is empty; cannot determine staging path")
	}
	clean := filepath.Clean(gameRelPath)
	// Guard against path traversal: staged path must stay inside WorkingDir/out.
	outRoot := filepath.Join(workingDir, "out")
	full := filepath.Join(outRoot, clean)
	if !strings.HasPrefix(filepath.Clean(full)+string(os.PathSeparator), filepath.Clean(outRoot)+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes staging directory", gameRelPath)
	}
	return full, nil
}

// sha256File returns the hex-encoded SHA-256 of data.
func sha256File(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// atomicWriteFile writes data to path atomically using a temp-file + rename
// pattern, preventing partial writes from leaving a corrupt staged file.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ffstage-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(data)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if writeErr != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", writeErr)
	}
	if syncErr != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("sync temp file: %w", syncErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", closeErr)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename to target: %w", err)
	}
	return nil
}

// Ensure io is used (sha256File uses crypto/sha256, not io directly; kept for
// potential future streaming hash use).
var _ = io.Discard

func injectFMGData(rawFMG []byte, entries []core.StringEntry, opts core.PersianOptions) ([]byte, int, error) {
	rows, err := extract.ParseFMG(rawFMG)
	if err != nil {
		return nil, 0, err
	}

	transMap := make(map[int32]string, len(entries))
	for _, e := range entries {
		id, err := strconv.ParseInt(e.Path, 10, 32)
		if err != nil {
			continue
		}
		transMap[int32(id)] = processPersianTranslation(e.Translation, opts)
	}

	count := 0
	for i := range rows {
		if t, ok := transMap[rows[i].ID]; ok {
			rows[i].Text = t
			count++
		}
	}

	// Detect version: if byte 2 is 2, it's wide (DS3/Sekiro/ER)
	version := int(rawFMG[2])
	if version == 0 {
		version = 2
	}
	unicode := rawFMG[8] != 0
	bigEndian := rawFMG[1] != 0

	encoded, err := extract.EncodeFMG(rows, version, unicode, bigEndian)
	if err != nil {
		return nil, 0, err
	}
	return encoded, count, nil
}
