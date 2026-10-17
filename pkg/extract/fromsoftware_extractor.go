package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"farsiforge/pkg/core"
	"farsiforge/pkg/scanner"
)

// FromSoftwareExtractor extracts loose FMG files and validates encrypted BHD5
// headers; supported BND/DCX wrappers are handled by the parsers in this package.
type FromSoftwareExtractor struct{}

func (*FromSoftwareExtractor) SupportedEngine() string { return "fromsoftware" }
func (*FromSoftwareExtractor) Capabilities() core.ExtractorCaps {
	return core.ExtractorCaps{TextExtraction: true}
}
func (*FromSoftwareExtractor) Extract(_ context.Context, info *core.GameInfo, proj *core.Project, _ core.ToolRegistry) error {
	seen := make(map[string]bool)

	// 1. Scan loose .fmg files
	fmgFiles := scanner.WalkDir(info.GameRoot, 12, func(p string) bool { return strings.EqualFold(filepath.Ext(p), ".fmg") })
	for _, p := range fmgFiles {
		b, e := os.ReadFile(p)
		if e != nil {
			continue
		}
		rows, e := ParseFMG(b)
		if e != nil {
			log.Warn("invalid FMG", "file", p, "error", e)
			continue
		}
		rel, _ := filepath.Rel(info.GameRoot, p)
		slashRel := filepath.ToSlash(rel)
		for _, row := range rows {
			if strings.TrimSpace(row.Text) != "" {
				key := slashRel + "::" + fmt.Sprint(row.ID)
				if seen[key] {
					continue
				}
				seen[key] = true
				proj.AddEntry(core.StringEntry{
					Source:  row.Text,
					File:    slashRel,
					Path:    fmt.Sprint(row.ID),
					Context: "FromSoftware FMG",
				})
			}
		}
		proj.ExtractedFiles = append(proj.ExtractedFiles, slashRel)
	}

	// 2. Scan loose message binder files (.msgbnd.dcx, .msgbnd, .bnd4, .bnd3)
	binderFiles := scanner.WalkDir(info.GameRoot, 12, func(p string) bool {
		lower := strings.ToLower(p)
		return strings.HasSuffix(lower, ".msgbnd.dcx") || strings.HasSuffix(lower, ".msgbnd") ||
			strings.HasSuffix(lower, ".bnd4") || strings.HasSuffix(lower, ".bnd3")
	})
	for _, p := range binderFiles {
		rel, _ := filepath.Rel(info.GameRoot, p)
		slashRel := filepath.ToSlash(rel)
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		bndData := raw
		if strings.HasSuffix(strings.ToLower(p), ".dcx") {
			decomp, err := DecompressDCX(raw)
			if err != nil {
				log.Warn("decompress loose DCX", "file", p, "error", err)
				continue
			}
			bndData = decomp
		}
		var innerFiles []BND4File
		if len(bndData) >= 4 && string(bndData[:4]) == "BND3" {
			bf, err := ParseBND3(bndData)
			if err == nil {
				innerFiles = bf
			}
		} else {
			bf, err := ParseBND4(bndData)
			if err == nil {
				innerFiles = bf
			}
		}
		injectedCount := 0
		for _, inner := range innerFiles {
			if !isFMGName(inner.Name) {
				continue
			}
			rows, err := ParseFMG(inner.Data)
			if err != nil {
				continue
			}
			innerPath := slashRel + "/" + inner.Name
			for _, row := range rows {
				if strings.TrimSpace(row.Text) != "" {
					key := innerPath + "::" + fmt.Sprint(row.ID)
					if seen[key] {
						continue
					}
					seen[key] = true
					proj.AddEntry(core.StringEntry{
						Source:  row.Text,
						File:    innerPath,
						Path:    fmt.Sprint(row.ID),
						Context: "FromSoftware FMG",
					})
					injectedCount++
				}
			}
		}
		if injectedCount > 0 {
			proj.ExtractedFiles = append(proj.ExtractedFiles, slashRel)
		}
	}

	// 3. Scan Data0.bhd/Data0.bdt archive — do not skip even if loose files exist.
	// Merge unique entries and record Container identity so archive-derived entries are traceable.
	bhdPath := filepath.Join(info.GameRoot, "Data0.bhd")
	if raw, err := os.ReadFile(bhdPath); err == nil {
		idx, err := DecryptERData0BHD(raw)
		if err != nil {
			return fmt.Errorf("decrypt Data0.bhd: %w", err)
		}
		relBhd, _ := filepath.Rel(info.GameRoot, bhdPath)
		containerName := filepath.ToSlash(relBhd)
		log.Info("Parsed FromSoftware BHD5 index", "file", bhdPath, "entries", len(idx.Files))
		bdtPath := filepath.Join(info.GameRoot, "Data0.bdt")
		bdt, err := os.Open(bdtPath)
		if err != nil {
			return fmt.Errorf("open Data0.bdt: %w", err)
		}
		defer bdt.Close()
		byHash := make(map[uint64]ERBHDFile, len(idx.Files))
		for _, f := range idx.Files {
			byHash[f.Hash] = f
		}
		candidates := make([]string, 0, 8)
		for _, lang := range []string{"engus", "jpn", "deude", "frafr", "ita", "spaes", "rus", "pol", "porbr", "kor", "zhocn"} {
			for _, base := range []string{"item", "menu", "item_dlc01", "menu_dlc01", "system", "talk"} {
				candidates = append(candidates, "/msg/"+lang+"/"+base+".msgbnd.dcx")
			}
		}
		sort.Strings(candidates)
		seenHashes := map[uint64]bool{}
		for _, name := range candidates {
			h := ERPathHash(name)
			f, ok := byHash[h]
			if !ok || seenHashes[h] {
				continue
			}
			seenHashes[h] = true
			dcx, err := ReadERBHDFile(idx, bdt, f)
			if err != nil {
				log.Warn("read FromSoftware BDT entry", "path", name, "error", err)
				continue
			}
			decoded, err := DecompressDCXWithOodle(dcx, filepath.Join(info.GameRoot, "oo2core_6_win64.dll"))
			if err != nil {
				log.Warn("decompress FromSoftware DCX", "path", name, "error", err)
				continue
			}
			files, err := ParseBND4(decoded)
			if err != nil {
				log.Warn("parse FromSoftware BND4", "path", name, "error", err)
				continue
			}
			archiveFileAdded := false
			for _, file := range files {
				if !isFMGName(file.Name) {
					continue
				}
				rows, err := ParseFMG(file.Data)
				if err != nil {
					log.Warn("parse FromSoftware FMG", "file", file.Name, "error", err)
					continue
				}
				innerPath := strings.TrimPrefix(name, "/") + "/" + file.Name
				for _, row := range rows {
					if strings.TrimSpace(row.Text) != "" {
						key := innerPath + "::" + fmt.Sprint(row.ID)
						if seen[key] {
							continue // Loose file already provided this entry; prioritize loose/editable copy
						}
						seen[key] = true
						proj.AddEntry(core.StringEntry{
							Source:    row.Text,
							File:      innerPath,
							Container: containerName,
							Path:      fmt.Sprint(row.ID),
							Context:   "FromSoftware BHD5 archive (read-only; BHD5/BDT repacking is not supported)",
							Notes:     "READ_ONLY: BHD5/BDT archive repacking is not supported",
						})
						archiveFileAdded = true
					}
				}
			}
			if archiveFileAdded {
				proj.ExtractedFiles = append(proj.ExtractedFiles, strings.TrimPrefix(name, "/"))
			}
		}
	}

	if len(proj.Entries) > 0 {
		return nil
	}
	return fmt.Errorf("no loose FMG files, binders, or supported BHD5 archive found")
}
