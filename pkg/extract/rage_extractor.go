package extract

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"farsiforge/pkg/core"
	"farsiforge/pkg/scanner"
)

// RAGEExtractor reads GXT2 localization tables from loose files and from
// encrypted RPF7 archives. RPF7 support derives its key material directly
// from the user's own GTA5.exe by SHA1-hash identification.
type RAGEExtractor struct{}

func (*RAGEExtractor) SupportedEngine() string { return "rage" }
func (*RAGEExtractor) Capabilities() core.ExtractorCaps {
	return core.ExtractorCaps{TextExtraction: true}
}

// maxRPF7Depth bounds nested child-RPF recursion.
const maxRPF7Depth = 8

func (*RAGEExtractor) Extract(_ context.Context, info *core.GameInfo, proj *core.Project, tools core.ToolRegistry) error {
	loose := extractLooseGXT2(info, proj)

	var keys *GTA5Keys
	if exePath := findGTA5Exe(info.GameRoot); exePath != "" {
		// Modern GTA V builds (2024+) keep only the AES key in the
		// executable; the NG key material is reconstructed from CodeWalker's
		// magic.dat when that file is available in the tools directory.
		magicData := findMagicBlob(tools)
		k, err := ExtractGTA5Keys(exePath, magicData)
		if err != nil {
			log.Warn("RPF7 key extraction failed; encrypted archives will be skipped",
				"exe", exePath, "error", err)
		} else {
			keys = k
			log.Info("RPF7 NG keys derived from game executable", "exe", filepath.Base(exePath))
		}
	}

	archives := scanner.WalkDir(info.GameRoot, 8, func(p string) bool {
		return strings.EqualFold(filepath.Ext(p), ".rpf")
	})
	processed := 0
	var ngSkipped int
	for _, p := range archives {
		rel, _ := filepath.Rel(info.GameRoot, p)
		arcRel := filepath.ToSlash(rel)
		// OpenIV-style `mods/` trees mirror the base archives; skip them so
		// projects keep one canonical copy of each string.
		if strings.HasPrefix(strings.ToLower(arcRel), "mods/") {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			continue
		}
		var hdr [16]byte
		_, err = f.ReadAt(hdr[:], 0)
		if err != nil || binary.LittleEndian.Uint32(hdr[0:4]) != rpf7Magic {
			f.Close()
			continue
		}
		if binary.LittleEndian.Uint32(hdr[12:16]) == rpf7EncNG && keys == nil {
			ngSkipped++
			f.Close()
			continue
		}
		n, err := extractRPF7GXT2(f, 0, filepath.Base(p), uint32(fi.Size()), keys, arcRel, proj, 0)
		f.Close()
		if err != nil {
			log.Warn("RPF7 traversal failed", "archive", arcRel, "error", err)
			continue
		}
		if n > 0 {
			processed += n
			proj.ExtractedFiles = append(proj.ExtractedFiles, arcRel)
		}
	}

	if ngSkipped > 0 && loose == 0 && keys == nil {
		return fmt.Errorf("found %d NG-encrypted RPF7 archives but no key material (GTA5.exe missing or unrecognized build)", ngSkipped)
	}
	return nil
}

// extractLooseGXT2 reads standalone .gxt2 tables on disk.
func extractLooseGXT2(info *core.GameInfo, proj *core.Project) int {
	files := scanner.WalkDir(info.GameRoot, 12, func(p string) bool {
		return strings.EqualFold(filepath.Ext(p), ".gxt2")
	})
	count := 0
	for _, p := range files {
		b, e := os.ReadFile(p)
		if e != nil {
			continue
		}
		rows, e := ParseGXT2(b)
		if e != nil {
			log.Warn("invalid GXT2", "file", p, "error", e)
			continue
		}
		rel, _ := filepath.Rel(info.GameRoot, p)
		for _, row := range rows {
			if strings.TrimSpace(row.Text) == "" {
				continue
			}
			proj.AddEntry(core.StringEntry{Source: row.Text, File: filepath.ToSlash(rel), Path: fmt.Sprintf("0x%08x", row.Hash), Context: "RAGE GXT2"})
		}
		proj.ExtractedFiles = append(proj.ExtractedFiles, filepath.ToSlash(rel))
		count++
	}
	return count
}

// findGTA5Exe locates the game executable that carries RPF7 key material.
func findGTA5Exe(root string) string {
	candidates := []string{filepath.Join(root, "GTA5.exe")}
	for _, c := range candidates {
		if scanner.FileExists(c) {
			return c
		}
	}
	return ""
}

// extractRPF7GXT2 walks an RPF7 archive (and its nested child RPF7 archives)
// and parses every .gxt2 entry it finds into the project. archiveLabel is the
// project-relative "container!path" prefix for entries.
func extractRPF7GXT2(f *os.File, baseOffset int64, name string, size uint32, keys *GTA5Keys, archiveLabel string, proj *core.Project, depth int) (int, error) {
	if depth > maxRPF7Depth {
		return 0, fmt.Errorf("RPF7 nesting exceeds depth %d", maxRPF7Depth)
	}
	arc, err := openRPF7(f, baseOffset, name, size, keys)
	if err != nil {
		return 0, err
	}
	found := 0
	for _, e := range arc.entries {
		switch {
		case e.kind == rpfEntryBinary && strings.HasSuffix(e.nameLower, ".gxt2"):
			data, err := arc.readBinary(e, f, keys)
			if err != nil {
				log.Warn("GXT2 entry read failed", "entry", archiveLabel+"!"+e.path, "error", err)
				continue
			}
			rows, err := ParseGXT2(data)
			if err != nil {
				log.Warn("invalid GXT2 inside archive", "entry", archiveLabel+"!"+e.path, "error", err)
				continue
			}
			fileLabel := archiveLabel + "!" + e.path
			for _, row := range rows {
				if strings.TrimSpace(row.Text) == "" {
					continue
				}
				proj.AddEntry(core.StringEntry{
					Source:  row.Text,
					File:    fileLabel,
					Path:    fmt.Sprintf("0x%08x", row.Hash),
					Context: "RAGE GXT2",
				})
			}
			found++
		case e.kind == rpfEntryBinary && strings.HasSuffix(e.nameLower, ".rpf"):
			childSize := e.fileSize
			if childSize == 0 {
				childSize = e.fileUncompressed
			}
			if childSize == 0 {
				continue
			}
			n, err := extractRPF7GXT2(f, arc.baseOffset+int64(e.fileOffset)*512, e.name, childSize, keys,
				archiveLabel+"!"+strings.TrimSuffix(e.nameLower, ".rpf"), proj, depth+1)
			if err != nil {
				log.Warn("child RPF7 traversal failed", "entry", archiveLabel+"!"+e.path, "error", err)
				continue
			}
			found += n
		}
	}
	return found, nil
}
