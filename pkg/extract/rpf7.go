package extract

// RPF7 archive reader for RAGE games (GTA V). The format is a 16-byte header
// (magic "RPF7", entry count, names length, encryption flag) followed by the
// table of contents (entry count * 16 bytes) and a names blob. The TOC may be
// plaintext, AES-encrypted, or "NG"-encrypted; NG TOCs are keyed by the
// archive's own file name and file size. File entries are 512-byte aligned and
// optionally raw-deflate compressed.

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const rpf7Magic = 0x52504637 // "RPF7"

// RPF7 header encryption flags.
const (
	rpf7EncNone uint32 = 0
	rpf7EncOpen uint32 = 0x4E45504F // "OPEN" — OpenIV style, plaintext TOC
	rpf7EncAES  uint32 = 0x0FFFFFF9
	rpf7EncNG   uint32 = 0x0FEFFFFF
)

type rpfEntryKind uint8

const (
	rpfEntryDir rpfEntryKind = iota
	rpfEntryBinary
	rpfEntryResource
)

type rpfEntry struct {
	kind      rpfEntryKind
	nameOff   uint32
	name      string // stored case, used for NG key derivation
	nameLower string
	path      string // full path inside the archive, lowercase

	// directory entries
	dirIndex uint32
	dirCount uint32

	// file entries
	fileOffset       uint32 // in 512-byte units
	fileSize         uint32 // compressed size; 0 means stored uncompressed
	fileUncompressed uint32
	encrypted        bool
}

// rpfArchive is a parsed RPF7 table of contents. Data blocks are read from
// the underlying *os.File relative to baseOffset.
type rpfArchive struct {
	name       string // archive name (NG key derivation)
	size       uint32 // archive size (NG key derivation)
	isAES      bool   // TOC was AES-encrypted; entry data uses AES too
	entries    []*rpfEntry
	baseOffset int64
}

// openRPF7 parses the RPF7 header and TOC at baseOffset within f. name and size
// identify the archive for NG TOC decryption. keys may be nil for plaintext
// archives.
func openRPF7(f *os.File, baseOffset int64, name string, size uint32, keys *GTA5Keys) (*rpfArchive, error) {
	var hdr [16]byte
	if _, err := f.ReadAt(hdr[:], baseOffset); err != nil {
		return nil, fmt.Errorf("read RPF7 header: %w", err)
	}
	if binary.LittleEndian.Uint32(hdr[0:4]) != rpf7Magic {
		return nil, fmt.Errorf("not an RPF7 archive (magic mismatch)")
	}
	entryCount := binary.LittleEndian.Uint32(hdr[4:8])
	namesLength := binary.LittleEndian.Uint32(hdr[8:12])
	encryption := binary.LittleEndian.Uint32(hdr[12:16])
	if entryCount == 0 {
		return nil, fmt.Errorf("RPF7 archive has no entries")
	}
	if uint64(entryCount) > 4_000_000 || namesLength > 256<<20 {
		return nil, fmt.Errorf("implausible RPF7 TOC dimensions (entries=%d names=%d)", entryCount, namesLength)
	}

	entriesRaw := make([]byte, int(entryCount)*16)
	if _, err := f.ReadAt(entriesRaw, baseOffset+16); err != nil {
		return nil, fmt.Errorf("read RPF7 TOC entries: %w", err)
	}
	namesRaw := make([]byte, namesLength)
	if namesLength > 0 {
		if _, err := f.ReadAt(namesRaw, baseOffset+16+int64(entryCount)*16); err != nil {
			return nil, fmt.Errorf("read RPF7 TOC names: %w", err)
		}
	}

	arc := &rpfArchive{name: name, size: size, baseOffset: baseOffset}
	switch encryption {
	case rpf7EncNone, rpf7EncOpen:
		// plaintext TOC
	case rpf7EncAES:
		if keys == nil {
			return nil, fmt.Errorf("AES-encrypted RPF7 TOC but no keys available")
		}
		entriesRaw = decryptAESECB(entriesRaw, keys.AESKey[:])
		namesRaw = decryptAESECB(namesRaw, keys.AESKey[:])
		arc.isAES = true
	case rpf7EncNG:
		if keys == nil {
			return nil, fmt.Errorf("NG-encrypted RPF7 TOC but no keys available")
		}
		entriesRaw = keys.decryptNG(entriesRaw, name, size)
		namesRaw = keys.decryptNG(namesRaw, name, size)
	default:
		return nil, fmt.Errorf("unknown RPF7 encryption flag 0x%08x", encryption)
	}

	arc.entries = make([]*rpfEntry, 0, entryCount)
	for i := uint32(0); i < entryCount; i++ {
		off := int(i) * 16
		x := binary.LittleEndian.Uint32(entriesRaw[off+4 : off+8])
		e := new(rpfEntry)
		switch {
		case x == 0x7FFFFF00: // directory entry
			e.kind = rpfEntryDir
			e.nameOff = binary.LittleEndian.Uint32(entriesRaw[off : off+4])
			e.dirIndex = binary.LittleEndian.Uint32(entriesRaw[off+8 : off+12])
			e.dirCount = binary.LittleEndian.Uint32(entriesRaw[off+12 : off+16])
		case x&0x80000000 == 0: // binary file entry
			buf := binary.LittleEndian.Uint64(entriesRaw[off : off+8])
			e.kind = rpfEntryBinary
			e.nameOff = uint32(buf & 0xFFFF)
			e.fileSize = uint32((buf >> 16) & 0xFFFFFF)
			e.fileOffset = uint32((buf >> 40) & 0xFFFFFF)
			e.fileUncompressed = binary.LittleEndian.Uint32(entriesRaw[off+8 : off+12])
			encType := binary.LittleEndian.Uint32(entriesRaw[off+12 : off+16])
			if encType > 1 {
				return nil, fmt.Errorf("RPF7 entry %d: unsupported encryption type %d", i, encType)
			}
			e.encrypted = encType == 1
		default: // resource file entry
			e.kind = rpfEntryResource
			e.nameOff = uint32(binary.LittleEndian.Uint16(entriesRaw[off : off+2]))
			e.fileSize = uint32(entriesRaw[off+2]) | uint32(entriesRaw[off+3])<<8 | uint32(entriesRaw[off+4])<<16
			e.fileOffset = (uint32(entriesRaw[off+5]) | uint32(entriesRaw[off+6])<<8 | uint32(entriesRaw[off+7])<<16) & 0x7FFFFF
		}
		e.name = rpfReadName(namesRaw, e.nameOff)
		e.nameLower = strings.ToLower(e.name)
		arc.entries = append(arc.entries, e)
	}

	root := arc.entries[0]
	if root.kind != rpfEntryDir {
		return nil, fmt.Errorf("RPF7 root entry is not a directory")
	}
	root.path = ""
	stack := []*rpfEntry{root}
	for len(stack) > 0 {
		d := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		end := d.dirIndex + d.dirCount
		if end > uint32(len(arc.entries)) {
			end = uint32(len(arc.entries))
		}
		for i := d.dirIndex; i < end; i++ {
			e := arc.entries[i]
			if e == nil || e == d {
				continue
			}
			if d.path == "" {
				e.path = e.nameLower
			} else {
				e.path = d.path + "/" + e.nameLower
			}
			if e.kind == rpfEntryDir {
				stack = append(stack, e)
			}
		}
	}
	return arc, nil
}

// rpfReadName reads a NUL-terminated name from the names blob.
func rpfReadName(names []byte, off uint32) string {
	if int(off) >= len(names) {
		return ""
	}
	rest := names[off:]
	if z := bytes.IndexByte(rest, 0); z >= 0 {
		rest = rest[:z]
	}
	if len(rest) > 256 {
		rest = rest[:256]
	}
	return string(rest)
}

// readBinary extracts and decrypts a binary file entry's data.
func (a *rpfArchive) readBinary(e *rpfEntry, f *os.File, keys *GTA5Keys) ([]byte, error) {
	size := e.fileSize
	if size == 0 {
		size = e.fileUncompressed
	}
	if size == 0 {
		return nil, nil
	}
	raw := make([]byte, size)
	if _, err := f.ReadAt(raw, a.baseOffset+int64(e.fileOffset)*512); err != nil {
		return nil, fmt.Errorf("read entry data: %w", err)
	}
	data := raw
	if e.encrypted {
		if keys == nil {
			return nil, fmt.Errorf("entry data is encrypted but keys are unavailable")
		}
		if a.isAES {
			data = decryptAESECB(raw, keys.AESKey[:])
		} else {
			data = keys.decryptNG(raw, e.name, e.fileUncompressed)
		}
	}
	if e.fileSize > 0 { // compressed → raw deflate stream
		out, err := inflateRawDeflate(data, e.fileUncompressed)
		if err != nil {
			return nil, fmt.Errorf("decompress: %w", err)
		}
		return out, nil
	}
	return data, nil
}

// inflateRawDeflate decompresses a raw DEFLATE stream. If the stream carries
// trailing garbage the output is still accepted as long as at least `expected`
// bytes were produced; anything beyond the expected size is discarded.
func inflateRawDeflate(data []byte, expected uint32) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		if expected > 0 && uint32(len(out)) >= expected {
			return out[:expected], nil
		}
		return nil, err
	}
	if expected > 0 && uint32(len(out)) > expected {
		out = out[:expected]
	}
	return out, nil
}
