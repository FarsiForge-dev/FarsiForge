package fonts

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf16"
)

// SFNT tags
const (
	tagTrueType = 0x00010000
	tagOpenType = 0x4F54544F // 'OTTO'
	tagTTC      = 0x74746366 // 'ttcf'
	tagTrue     = 0x74727565 // 'true'
	tagTyp1     = 0x74797031 // 'typ1'

	tagCmap = 0x636D6170 // 'cmap'
	tagName = 0x6E616D65 // 'name'
	tagGSUB = 0x47535542 // 'GSUB'
)

// TableRecord describes an SFNT table location and size.
type TableRecord struct {
	Tag      uint32
	Checksum uint32
	Offset   uint32
	Length   uint32
}

// ParsedFont holds metadata, glyph coverage, and shaping capabilities extracted from an SFNT font.
type ParsedFont struct {
	Format        string
	Family        string
	Subfamily     string
	FullName      string
	HasGSUB       bool
	SupportedChars map[rune]bool
}

// ParseSFNT parses font metadata, name records, and cmap character coverage from a TTF/OTF file.
func ParseSFNT(filePath string) (*ParsedFont, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read font file: %w", err)
	}
	return ParseSFNTBytes(data)
}

// ParseSFNTBytes parses font metadata from in-memory byte slice.
func ParseSFNTBytes(data []byte) (*ParsedFont, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("font file too small (%d bytes)", len(data))
	}

	r := bytes.NewReader(data)
	var sfntVersion uint32
	if err := binary.Read(r, binary.BigEndian, &sfntVersion); err != nil {
		return nil, err
	}

	format := "TTF"
	switch sfntVersion {
	case tagTrueType, tagTrue:
		format = "TrueType (TTF)"
	case tagOpenType:
		format = "OpenType (OTF/CFF)"
	case tagTTC:
		format = "TrueType Collection (TTC)"
		// For TTC, read first font offset
		if len(data) >= 16 {
			firstOffset := binary.BigEndian.Uint32(data[12:16])
			if int(firstOffset) < len(data) {
				return ParseSFNTBytes(data[firstOffset:])
			}
		}
	default:
		return nil, fmt.Errorf("unrecognized font format signature: 0x%08X", sfntVersion)
	}

	var numTables uint16
	var searchRange, entrySelector, rangeShift uint16
	_ = binary.Read(r, binary.BigEndian, &numTables)
	_ = binary.Read(r, binary.BigEndian, &searchRange)
	_ = binary.Read(r, binary.BigEndian, &entrySelector)
	_ = binary.Read(r, binary.BigEndian, &rangeShift)

	if len(data) < 12+int(numTables)*16 {
		return nil, fmt.Errorf("corrupt font table directory")
	}

	tables := make(map[uint32]TableRecord, numTables)
	for i := 0; i < int(numTables); i++ {
		var tr TableRecord
		if err := binary.Read(r, binary.BigEndian, &tr); err != nil {
			break
		}
		tables[tr.Tag] = tr
	}

	pf := &ParsedFont{
		Format:         format,
		SupportedChars: make(map[rune]bool),
		HasGSUB:        tables[tagGSUB].Length > 0,
	}

	// 1. Parse 'name' table
	if nameRec, ok := tables[tagName]; ok {
		parseNameTable(data, nameRec, pf)
	}

	// 2. Parse 'cmap' table
	if cmapRec, ok := tables[tagCmap]; ok {
		parseCmapTable(data, cmapRec, pf)
	}

	return pf, nil
}

func parseNameTable(data []byte, rec TableRecord, pf *ParsedFont) {
	if int(rec.Offset+rec.Length) > len(data) || rec.Length < 6 {
		return
	}
	tableData := data[rec.Offset : rec.Offset+rec.Length]
	r := bytes.NewReader(tableData)

	var format, count, stringOffset uint16
	_ = binary.Read(r, binary.BigEndian, &format)
	_ = binary.Read(r, binary.BigEndian, &count)
	_ = binary.Read(r, binary.BigEndian, &stringOffset)

	strBase := int(stringOffset)
	for i := 0; i < int(count); i++ {
		var platformID, encodingID, languageID, nameID, length, offset uint16
		_ = binary.Read(r, binary.BigEndian, &platformID)
		_ = binary.Read(r, binary.BigEndian, &encodingID)
		_ = binary.Read(r, binary.BigEndian, &languageID)
		_ = binary.Read(r, binary.BigEndian, &nameID)
		_ = binary.Read(r, binary.BigEndian, &length)
		_ = binary.Read(r, binary.BigEndian, &offset)

		start := strBase + int(offset)
		end := start + int(length)
		if end > len(tableData) {
			continue
		}

		nameBytes := tableData[start:end]
		var str string

		// Windows Unicode (UTF-16BE)
		if platformID == 3 || platformID == 0 {
			if len(nameBytes)%2 == 0 {
				u16s := make([]uint16, len(nameBytes)/2)
				for j := 0; j < len(u16s); j++ {
					u16s[j] = binary.BigEndian.Uint16(nameBytes[j*2 : j*2+2])
				}
				str = string(utf16.Decode(u16s))
			}
		} else {
			str = string(nameBytes)
		}

		if str != "" {
			switch nameID {
			case 1:
				if pf.Family == "" || platformID == 3 {
					pf.Family = str
				}
			case 2:
				if pf.Subfamily == "" || platformID == 3 {
					pf.Subfamily = str
				}
			case 4:
				if pf.FullName == "" || platformID == 3 {
					pf.FullName = str
				}
			}
		}
	}
}

func parseCmapTable(data []byte, rec TableRecord, pf *ParsedFont) {
	if int(rec.Offset+rec.Length) > len(data) || rec.Length < 4 {
		return
	}
	tableData := data[rec.Offset : rec.Offset+rec.Length]
	r := bytes.NewReader(tableData)

	var version, numSubtables uint16
	_ = binary.Read(r, binary.BigEndian, &version)
	_ = binary.Read(r, binary.BigEndian, &numSubtables)

	type subtableEntry struct {
		platformID uint16
		encodingID uint16
		offset     uint32
	}

	var entries []subtableEntry
	for i := 0; i < int(numSubtables); i++ {
		var e subtableEntry
		_ = binary.Read(r, binary.BigEndian, &e.platformID)
		_ = binary.Read(r, binary.BigEndian, &e.encodingID)
		_ = binary.Read(r, binary.BigEndian, &e.offset)
		entries = append(entries, e)
	}

	// Try Windows Unicode BMP (3, 1) or Unicode (0, 3) or Unicode (0, 4)
	for _, e := range entries {
		if int(e.offset) >= len(tableData) {
			continue
		}
		sub := tableData[e.offset:]
		if len(sub) < 6 {
			continue
		}
		format := binary.BigEndian.Uint16(sub[0:2])
		if format == 4 {
			parseCmapFormat4(sub, pf)
			if len(pf.SupportedChars) > 0 {
				break
			}
		} else if format == 12 {
			parseCmapFormat12(sub, pf)
			if len(pf.SupportedChars) > 0 {
				break
			}
		}
	}
}

func parseCmapFormat4(sub []byte, pf *ParsedFont) {
	if len(sub) < 14 {
		return
	}
	segCountX2 := binary.BigEndian.Uint16(sub[6:8])
	segCount := int(segCountX2 / 2)
	if segCount == 0 || len(sub) < 14+segCount*8+2 {
		return
	}

	endCodeOffset := 14
	startCodeOffset := endCodeOffset + segCount*2 + 2
	idDeltaOffset := startCodeOffset + segCount*2
	idRangeOffset := idDeltaOffset + segCount*2

	for i := 0; i < segCount; i++ {
		endCode := binary.BigEndian.Uint16(sub[endCodeOffset+i*2 : endCodeOffset+i*2+2])
		startCode := binary.BigEndian.Uint16(sub[startCodeOffset+i*2 : startCodeOffset+i*2+2])
		idDelta := int16(binary.BigEndian.Uint16(sub[idDeltaOffset+i*2 : idDeltaOffset+i*2+2]))
		rangeOffset := binary.BigEndian.Uint16(sub[idRangeOffset+i*2 : idRangeOffset+i*2+2])

		if startCode == 0xFFFF {
			break
		}

		for c := startCode; c <= endCode; c++ {
			var glyphID uint16
			if rangeOffset == 0 {
				glyphID = uint16(int16(c) + idDelta)
			} else {
				glyphOffset := idRangeOffset + i*2 + int(rangeOffset) + int(c-startCode)*2
				if glyphOffset+2 <= len(sub) {
					glyphID = binary.BigEndian.Uint16(sub[glyphOffset : glyphOffset+2])
					if glyphID != 0 {
						glyphID = uint16(int16(glyphID) + idDelta)
					}
				}
			}
			if glyphID != 0 {
				pf.SupportedChars[rune(c)] = true
			}
		}
	}
}

func parseCmapFormat12(sub []byte, pf *ParsedFont) {
	if len(sub) < 16 {
		return
	}
	numGroups := binary.BigEndian.Uint32(sub[12:16])
	groupOffset := 16
	for i := 0; i < int(numGroups); i++ {
		pos := groupOffset + i*12
		if pos+12 > len(sub) {
			break
		}
		startChar := binary.BigEndian.Uint32(sub[pos : pos+4])
		endChar := binary.BigEndian.Uint32(sub[pos+4 : pos+8])
		startGlyph := binary.BigEndian.Uint32(sub[pos+8 : pos+12])

		for c := startChar; c <= endChar; c++ {
			if startGlyph+(c-startChar) != 0 {
				pf.SupportedChars[rune(c)] = true
			}
		}
	}
}

// PersianTestRunes represents standard Persian and Arabic characters needed for localization.
var PersianTestRunes = []rune{
	// Persian unique letters
	'گ', 'چ', 'پ', 'ژ', 'ی', 'ک',
	// Arabic/Persian shared letters
	'ا', 'ب', 'ت', 'ث', 'ج', 'ح', 'خ', 'د', 'ذ', 'ر', 'ز', 'س', 'ش',
	'ص', 'ض', 'ط', 'ظ', 'ع', 'غ', 'ف', 'ق', 'ل', 'م', 'ن', 'و', 'ه',
	// Persian digits
	'۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹',
}

// EvaluatePersianCoverage computes the glyph coverage metrics for Persian localization.
func EvaluatePersianCoverage(pf *ParsedFont) GlyphCoverageReport {
	if pf == nil || len(pf.SupportedChars) == 0 {
		return GlyphCoverageReport{
			TotalTestedRunes: len(PersianTestRunes),
			CoveragePercent:  0.0,
		}
	}

	supportedCount := 0
	var missing []string

	persianSpecific := []rune{'گ', 'چ', 'پ', 'ژ', 'ی', 'ک'}
	persianDigits := []rune{'۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'}

	persianSpecificSupported := 0
	persianDigitsSupported := 0

	for _, r := range PersianTestRunes {
		if pf.SupportedChars[r] {
			supportedCount++
		} else {
			missing = append(missing, string(r))
		}
	}

	for _, r := range persianSpecific {
		if pf.SupportedChars[r] {
			persianSpecificSupported++
		}
	}

	for _, r := range persianDigits {
		if pf.SupportedChars[r] {
			persianDigitsSupported++
		}
	}

	total := len(PersianTestRunes)
	pct := float64(supportedCount) / float64(total) * 100.0

	return GlyphCoverageReport{
		SupportsPersian:       persianSpecificSupported == len(persianSpecific),
		SupportsArabic:        supportedCount >= (total - len(persianSpecific) - len(persianDigits)),
		SupportsPersianDigits: persianDigitsSupported == len(persianDigits),
		HasOpenTypeShaping:    pf.HasGSUB,
		TotalTestedRunes:      total,
		SupportedRunesCount:   supportedCount,
		MissingRunes:          missing,
		CoveragePercent:       pct,
	}
}
