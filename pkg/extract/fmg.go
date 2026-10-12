package extract

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf16"
)

// FMGEntry contains a FromSoftware message ID and its decoded text.
type FMGEntry struct {
	ID   int32
	Text string
}

// ParseFMG decodes FMG v1/v2 tables. V2 uses 32-bit IDs and variable-width
// offsets; the byte-order and Unicode flags are read from the header.
func ParseFMG(data []byte) ([]FMGEntry, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("FMG header is truncated")
	}
	if data[0] != 0 || data[3] != 0 || (data[8] != 0 && data[8] != 1) {
		return nil, fmt.Errorf("invalid FMG signature/flags")
	}
	be := data[1] != 0
	var order binary.ByteOrder = binary.LittleEndian
	if be {
		order = binary.BigEndian
	}
	version := data[2]
	fileSize := int64(order.Uint32(data[4:8]))
	unicodeText := data[8] != 0
	if fileSize < 24 || fileSize > int64(len(data)) {
		return nil, fmt.Errorf("invalid FMG file size %d", fileSize)
	}
	groupCount := int(order.Uint32(data[12:16]))
	stringCount := int(order.Uint32(data[16:20]))
	if groupCount < 0 || groupCount > 1<<20 || stringCount < 0 || stringCount > 1<<22 {
		return nil, fmt.Errorf("invalid FMG counts")
	}
	wide := version == 2
	ptrSize, groupStart, groupSize := 4, 28, 12
	offAt := 20
	if wide {
		ptrSize, groupStart, groupSize, offAt = 8, 40, 16, 24
	}
	if groupStart+groupCount*groupSize > int(fileSize) {
		return nil, fmt.Errorf("FMG groups exceed file bounds")
	}
	var offsets int64
	if ptrSize == 8 {
		offsets = int64(order.Uint64(data[offAt : offAt+8]))
	} else {
		offsets = int64(order.Uint32(data[offAt : offAt+4]))
	}
	if offsets < 0 || offsets+int64(stringCount)*int64(ptrSize) > fileSize {
		return nil, fmt.Errorf("FMG string-offset table out of range")
	}
	entries := make([]FMGEntry, 0, stringCount)
	for g := 0; g < groupCount; g++ {
		p := groupStart + g*groupSize
		first, last := int32(order.Uint32(data[p+4:p+8])), int32(order.Uint32(data[p+8:p+12]))
		if last < first || int64(last)-int64(first) > 1<<22 {
			continue
		}
		for j := int64(0); j <= int64(last)-int64(first); j++ {
			id := int32(int64(first) + j)
			idx := int64(order.Uint32(data[p:p+4])) + j
			op := offsets + idx*int64(ptrSize)
			if op < 0 || op+int64(ptrSize) > fileSize {
				continue
			}
			var rel uint64
			if ptrSize == 8 {
				rel = order.Uint64(data[op : op+8])
			} else {
				rel = uint64(order.Uint32(data[op : op+4]))
			}
			if rel == 0 {
				continue
			}
			if rel > uint64(fileSize) {
				continue
			}
			start := int64(rel)
			if start < 0 || start >= fileSize {
				continue
			}
			text, ok := readFMGString(data[start:fileSize], unicodeText, be)
			if ok && text != "" {
				entries = append(entries, FMGEntry{ID: id, Text: text})
			}
		}
	}
	return entries, nil
}

func readFMGString(b []byte, wide, be bool) (string, bool) {
	if !wide {
		for i, c := range b {
			if c == 0 {
				return string(b[:i]), true
			}
		}
		return "", false
	}
	var u []uint16
	for i := 0; i+1 < len(b); i += 2 {
		v := binary.LittleEndian.Uint16(b[i : i+2])
		if be {
			v = binary.BigEndian.Uint16(b[i : i+2])
		}
		if v == 0 {
			return string(utf16.Decode(u)), true
		}
		u = append(u, v)
	}
	return "", false
}

// EncodeFMG encodes a slice of FMG entries into binary FMG format.
// Version 1 is used for DS1/DS2, version 2 (wide) is used for DS3/Sekiro/Elden Ring.
// If version is 0, it defaults to version 2 (wide).
func EncodeFMG(entries []FMGEntry, version int, unicode, bigEndian bool) ([]byte, error) {
	if version == 0 {
		version = 2
	}
	wide := version == 2

	var order binary.ByteOrder = binary.LittleEndian
	if bigEndian {
		order = binary.BigEndian
	}

	// Sort entries by ID ascending.
	sorted := make([]FMGEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	// Group contiguous IDs.
	type groupInfo struct {
		offsetIndex int32
		firstID     int32
		lastID      int32
	}
	var groups []groupInfo
	for i := 0; i < len(sorted); i++ {
		startIdx := int32(i)
		firstID := sorted[i].ID
		for i < len(sorted)-1 && sorted[i+1].ID == sorted[i].ID+1 {
			i++
		}
		lastID := sorted[i].ID
		groups = append(groups, groupInfo{
			offsetIndex: startIdx,
			firstID:     firstID,
			lastID:      lastID,
		})
	}

	buf := new(bytes.Buffer)

	// Header:
	buf.WriteByte(0)
	if bigEndian {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	buf.WriteByte(byte(version))
	buf.WriteByte(0)

	// Placeholder for FileSize (4 bytes)
	fileSizePos := buf.Len()
	var dummyU32 [4]byte
	buf.Write(dummyU32[:])

	if unicode {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	buf.WriteByte(0) // 0x00 (0xFF for DeS)
	buf.WriteByte(0)
	buf.WriteByte(0)

	// Placeholder for GroupCount (4 bytes)
	groupCountPos := buf.Len()
	buf.Write(dummyU32[:])

	// StringCount (4 bytes)
	order.PutUint32(dummyU32[:], uint32(len(sorted)))
	buf.Write(dummyU32[:])

	if wide {
		order.PutUint32(dummyU32[:], 0xFF)
		buf.Write(dummyU32[:])
	}

	// Placeholder for StringOffsets offset
	stringOffsetsOffsetPos := buf.Len()
	if wide {
		var dummyU64 [8]byte
		buf.Write(dummyU64[:])
		buf.Write(dummyU64[:]) // 8-byte zero
	} else {
		buf.Write(dummyU32[:])
		buf.Write(dummyU32[:]) // 4-byte zero
	}

	// Write groups
	for _, g := range groups {
		var b4 [4]byte
		order.PutUint32(b4[:], uint32(g.offsetIndex))
		buf.Write(b4[:])
		order.PutUint32(b4[:], uint32(g.firstID))
		buf.Write(b4[:])
		order.PutUint32(b4[:], uint32(g.lastID))
		buf.Write(b4[:])
		if wide {
			order.PutUint32(b4[:], 0)
			buf.Write(b4[:])
		}
	}

	// String offsets table position
	stringOffsetsStart := buf.Len()
	ptrSize := 4
	if wide {
		ptrSize = 8
	}

	strBuf := new(bytes.Buffer)
	stringOffsets := make([]byte, len(sorted)*ptrSize)
	for i, e := range sorted {
		if e.Text == "" {
			continue // offset remains 0
		}
		curOffset := uint64(stringOffsetsStart + len(stringOffsets) + strBuf.Len())
		offPos := i * ptrSize
		if wide {
			order.PutUint64(stringOffsets[offPos:offPos+8], curOffset)
		} else {
			order.PutUint32(stringOffsets[offPos:offPos+4], uint32(curOffset))
		}

		if unicode {
			runes := []rune(e.Text)
			u16 := utf16.Encode(runes)
			for _, val := range u16 {
				var b2 [2]byte
				if bigEndian {
					binary.BigEndian.PutUint16(b2[:], val)
				} else {
					binary.LittleEndian.PutUint16(b2[:], val)
				}
				strBuf.Write(b2[:])
			}
			strBuf.Write([]byte{0, 0}) // NUL terminator
		} else {
			strBuf.WriteString(e.Text)
			strBuf.WriteByte(0)
		}
	}

	buf.Write(stringOffsets)
	buf.Write(strBuf.Bytes())

	// Re-fetch data slice after appending strings
	data := buf.Bytes()
	totalSize := uint32(len(data))
	order.PutUint32(data[fileSizePos:fileSizePos+4], totalSize)
	order.PutUint32(data[groupCountPos:groupCountPos+4], uint32(len(groups)))
	if wide {
		order.PutUint64(data[stringOffsetsOffsetPos:stringOffsetsOffsetPos+8], uint64(stringOffsetsStart))
	} else {
		order.PutUint32(data[stringOffsetsOffsetPos:stringOffsetsOffsetPos+4], uint32(stringOffsetsStart))
	}

	return data, nil
}

