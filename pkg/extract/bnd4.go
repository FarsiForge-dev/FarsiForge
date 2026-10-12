package extract

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"
)

// BND4File is a file extracted from a FromSoftware BND4 container.
type BND4File struct {
	ID   int32
	Name string
	Data []byte
}

// ParseBND4 reads common BND4 variants used for message bundles.
func ParseBND4(data []byte) ([]BND4File, error) {
	if len(data) < 64 || string(data[:4]) != "BND4" {
		return nil, fmt.Errorf("invalid BND4 header")
	}
	be := data[8] != 0
	var order binary.ByteOrder = binary.LittleEndian
	if be {
		order = binary.BigEndian
	}
	bitBigEndian := data[10] == 0
	rawFmt := data[49]
	reverse := bitBigEndian || (rawFmt&1 != 0 && rawFmt&0x80 == 0)
	if !reverse {
		rawFmt = bits.Reverse8(rawFmt)
	}
	format := rawFmt
	count := uint64(order.Uint32(data[12:16]))
	headerSize := order.Uint64(data[32:40])
	longOffsets := format&0x10 != 0
	hasCompression := format&0x20 != 0
	hasIDs := format&2 != 0
	hasNames := format&0x0c != 0
	per := uint64(16)
	if longOffsets {
		per += 8
	} else {
		per += 4
	}
	if hasCompression {
		per += 8
	}
	if hasIDs {
		per += 4
	}
	if hasNames {
		per += 4
	}
	if format == 4 {
		per += 8
	}
	if count > 1<<20 || headerSize != per || 64+count*per > uint64(len(data)) {
		return nil, fmt.Errorf("unsupported or invalid BND4 file-header table")
	}
	files := make([]BND4File, 0, count)
	p := uint64(64)
	for n := uint64(0); n < count; n++ {
		flags := data[p]
		if !bitBigEndian {
			flags = bits.Reverse8(flags)
		}
		p += 4
		if int32(order.Uint32(data[p:p+4])) != -1 {
			return nil, fmt.Errorf("invalid BND4 file-header sentinel")
		}
		p += 4
		compressed := order.Uint64(data[p : p+8])
		p += 8
		uncompressed := compressed
		if hasCompression {
			uncompressed = order.Uint64(data[p : p+8])
			p += 8
		}
		var off uint64
		if longOffsets {
			off = order.Uint64(data[p : p+8])
			p += 8
		} else {
			off = uint64(order.Uint32(data[p : p+4]))
			p += 4
		}
		id := int32(-1)
		if hasIDs {
			id = int32(order.Uint32(data[p : p+4]))
			p += 4
		}
		name := ""
		if hasNames {
			no := uint64(order.Uint32(data[p : p+4]))
			p += 4
			if no < uint64(len(data)) {
				name = readBND4Name(data[no:], data[48] != 0, be)
			}
		}
		if format == 4 {
			if p+8 > uint64(len(data)) {
				return nil, fmt.Errorf("truncated BND4 Names1 header")
			}
			id = int32(order.Uint32(data[p : p+4]))
			p += 8
		}
		if compressed > 1<<31 || off > uint64(len(data)) || compressed > uint64(len(data))-off {
			return nil, fmt.Errorf("BND4 file %d data range is invalid (offset=%d compressed=%d length=%d format=%02x)", n, off, compressed, len(data), format)
		}
		payload := append([]byte(nil), data[off:off+compressed]...)
		if flags&1 != 0 && uncompressed > 0 {
			raw, err := inflateBND4(payload, int(uncompressed))
			if err != nil {
				return nil, fmt.Errorf("decompress BND4 file %d: %w", n, err)
			}
			payload = raw
		}
		files = append(files, BND4File{ID: id, Name: name, Data: payload})
	}
	return files, nil
}

func readBND4Name(b []byte, wide, be bool) string {
	if !wide {
		if i := bytes.IndexByte(b, 0); i >= 0 {
			b = b[:i]
		}
		return string(b)
	}
	var u []uint16
	for i := 0; i+1 < len(b); i += 2 {
		v := binary.LittleEndian.Uint16(b[i : i+2])
		if be {
			v = binary.BigEndian.Uint16(b[i : i+2])
		}
		if v == 0 {
			break
		}
		u = append(u, v)
	}
	return string(utf16.Decode(u))
}

func inflateBND4(data []byte, size int) ([]byte, error) {
	if r, e := zlib.NewReader(bytes.NewReader(data)); e == nil {
		defer r.Close()
		out, e := io.ReadAll(io.LimitReader(r, int64(size)+1))
		if e == nil && len(out) == size {
			return out, nil
		}
	}
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	out, e := io.ReadAll(io.LimitReader(r, int64(size)+1))
	if e != nil {
		return nil, e
	}
	if len(out) != size {
		return nil, fmt.Errorf("size mismatch: got %d, want %d", len(out), size)
	}
	return out, nil
}

func isFMGName(name string) bool { return strings.EqualFold(filepath.Ext(name), ".fmg") }

// BND4Options configures serialization of BND4 containers.
type BND4Options struct {
	Version      string // 8-char version string, e.g. "00000001" or "GR210000"
	Format       byte   // Default: 0x2E (IDs | Names1 | Names2 | Compression)
	Unicode      bool   // Default: true
	BigEndian    bool   // Default: false
	BitBigEndian bool   // Default: true
	Extended     byte   // Default: 4 (with hash table) or 0
}

// DefaultBND4Options returns the standard configuration used for Dark Souls 3 and Elden Ring message bundles.
func DefaultBND4Options() BND4Options {
	return BND4Options{
		Version:      "00000001",
		Format:       0x2E,
		Unicode:      true,
		BigEndian:    false,
		BitBigEndian: true,
		Extended:     4,
	}
}

// FromPathHash computes FromSoftware's 32-bit path hash for BND hash tables.
func FromPathHash(path string) uint32 {
	h := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	if !strings.HasPrefix(h, "/") {
		h = "/" + h
	}
	var hash uint32
	for _, c := range []byte(h) {
		hash = hash*37 + uint32(c)
	}
	return hash
}

func isPrime(candidate uint32) bool {
	if candidate < 2 {
		return false
	}
	if candidate == 2 {
		return true
	}
	if candidate%2 == 0 {
		return false
	}
	for i := uint32(3); i*i <= candidate; i += 2 {
		if candidate%i == 0 {
			return false
		}
	}
	return true
}

func bnd4FileHeaderSize(format byte) uint64 {
	per := uint64(16)
	if format&0x10 != 0 { // longOffsets
		per += 8
	} else {
		per += 4
	}
	if format&0x20 != 0 { // hasCompression
		per += 8
	}
	if format&0x02 != 0 { // hasIDs
		per += 4
	}
	if format&0x0C != 0 { // hasNames
		per += 4
	}
	if format == 4 { // Names1
		per += 8
	}
	return per
}

// WriteBND4 serializes files into a FromSoftware BND4 container.
func WriteBND4(files []BND4File, opts BND4Options) ([]byte, error) {
	if opts.Format == 0 {
		opts = DefaultBND4Options()
	}
	var order binary.ByteOrder = binary.LittleEndian
	if opts.BigEndian {
		order = binary.BigEndian
	}

	buf := new(bytes.Buffer)

	// 0x00: "BND4" (4 bytes)
	buf.WriteString("BND4")
	// 0x04..0x08: Unk04, Unk05, 0, 0
	buf.Write([]byte{0, 0, 0, 0})
	// 0x08..0x0C: 0, BigEndian, !BitBigEndian, 0
	beByte := byte(0)
	if opts.BigEndian {
		beByte = 1
	}
	notBbeByte := byte(0)
	if !opts.BitBigEndian {
		notBbeByte = 1
	}
	buf.Write([]byte{0, beByte, notBbeByte, 0})

	// 0x0C: FileCount (4 bytes)
	var u32 [4]byte
	order.PutUint32(u32[:], uint32(len(files)))
	buf.Write(u32[:])

	// 0x10: HeaderSize = 0x40 (8 bytes)
	var u64 [8]byte
	order.PutUint64(u64[:], 0x40)
	buf.Write(u64[:])

	// 0x18: Version (8 bytes)
	vBytes := make([]byte, 8)
	copy(vBytes, []byte(opts.Version))
	buf.Write(vBytes)

	// 0x20: FileHeaderSize (8 bytes)
	per := bnd4FileHeaderSize(opts.Format)
	order.PutUint64(u64[:], per)
	buf.Write(u64[:])

	// 0x28: HeadersEnd placeholder (8 bytes)
	headersEndPos := buf.Len()
	buf.Write(u64[:])

	// 0x30: Unicode, Format, Extended, 0
	uByte := byte(0)
	if opts.Unicode {
		uByte = 1
	}
	rawFmt := opts.Format
	if !opts.BitBigEndian {
		rawFmt = bits.Reverse8(rawFmt)
	}
	buf.Write([]byte{uByte, rawFmt, opts.Extended, 0})

	// 0x34: int32 0 (4 bytes)
	order.PutUint32(u32[:], 0)
	buf.Write(u32[:])

	// 0x38: HashTableOffset placeholder (8 bytes)
	hashTableOffsetPos := buf.Len()
	buf.Write(u64[:])

	format := opts.Format
	longOffsets := format&0x10 != 0
	hasCompression := format&0x20 != 0
	hasIDs := format&2 != 0
	hasNames := format&0x0c != 0

	type fileHeaderPos struct {
		compressedPos   int
		uncompressedPos int
		dataOffsetPos   int
		nameOffsetPos   int
	}
	headerPositions := make([]fileHeaderPos, len(files))

	// Write file headers table
	for i, f := range files {
		startHeader := buf.Len()
		// Flags: 0x40 (standard) or 0
		flagByte := byte(0x40)
		if !opts.BitBigEndian {
			flagByte = bits.Reverse8(flagByte)
		}
		buf.WriteByte(flagByte)
		buf.Write([]byte{0, 0, 0})
		// Sentinel -1 (0xFFFFFFFF)
		order.PutUint32(u32[:], 0xFFFFFFFF)
		buf.Write(u32[:])

		// CompressedSize (8 bytes)
		headerPositions[i].compressedPos = buf.Len()
		buf.Write(u64[:])

		// UncompressedSize (8 bytes)
		if hasCompression {
			headerPositions[i].uncompressedPos = buf.Len()
			buf.Write(u64[:])
		}

		// FileDataOffset (8 or 4 bytes)
		headerPositions[i].dataOffsetPos = buf.Len()
		if longOffsets {
			buf.Write(u64[:])
		} else {
			buf.Write(u32[:])
		}

		// ID (4 bytes)
		if hasIDs {
			order.PutUint32(u32[:], uint32(f.ID))
			buf.Write(u32[:])
		}

		// FileNameOffset (4 bytes)
		if hasNames {
			headerPositions[i].nameOffsetPos = buf.Len()
			buf.Write(u32[:])
		}

		if format == 4 {
			order.PutUint32(u32[:], uint32(f.ID))
			buf.Write(u32[:])
			order.PutUint32(u32[:], 0)
			buf.Write(u32[:])
		}

		writtenHeader := uint64(buf.Len() - startHeader)
		if writtenHeader != per {
			return nil, fmt.Errorf("internal error: header size mismatch %d != %d", writtenHeader, per)
		}
	}

	// Write file names
	data := buf.Bytes()
	for i, f := range files {
		if !hasNames {
			continue
		}
		namePos := uint32(buf.Len())
		// Fill name offset in header
		order.PutUint32(data[headerPositions[i].nameOffsetPos:headerPositions[i].nameOffsetPos+4], namePos)

		if opts.Unicode {
			u16 := utf16.Encode([]rune(f.Name))
			for _, v := range u16 {
				var b2 [2]byte
				if opts.BigEndian {
					binary.BigEndian.PutUint16(b2[:], v)
				} else {
					binary.LittleEndian.PutUint16(b2[:], v)
				}
				buf.Write(b2[:])
			}
			buf.Write([]byte{0, 0}) // NUL terminator
		} else {
			buf.WriteString(f.Name)
			buf.WriteByte(0)
		}
		data = buf.Bytes()
	}

	// Write hash table if Extended == 4
	var hashTableOffset uint64
	if opts.Extended == 4 && len(files) > 0 {
		for buf.Len()%8 != 0 {
			buf.WriteByte(0)
		}
		hashTableOffset = uint64(buf.Len())

		bucketCount := uint32(len(files) / 7)
		if bucketCount < 7 {
			bucketCount = 7
		}
		for !isPrime(bucketCount) {
			bucketCount++
		}

		type pathHashItem struct {
			index int
			hash  uint32
		}
		hashLists := make([][]pathHashItem, bucketCount)
		for i, f := range files {
			h := FromPathHash(f.Name)
			g := h % bucketCount
			hashLists[g] = append(hashLists[g], pathHashItem{index: i, hash: h})
		}
		for g := range hashLists {
			sort.Slice(hashLists[g], func(a, b int) bool {
				return hashLists[g][a].hash < hashLists[g][b].hash
			})
		}

		type hashGroup struct {
			index  int32
			length int32
		}
		var hashGroups []hashGroup
		var flatHashes []pathHashItem
		count := 0
		for _, list := range hashLists {
			idx := count
			for _, item := range list {
				flatHashes = append(flatHashes, item)
				count++
			}
			hashGroups = append(hashGroups, hashGroup{index: int32(idx), length: int32(count - idx)})
		}

		// HashesOffset placeholder (8 bytes)
		hashesOffsetPos := buf.Len()
		buf.Write(u64[:])

		order.PutUint32(u32[:], bucketCount)
		buf.Write(u32[:])
		buf.Write([]byte{0x10, 8, 8, 0})

		// Write hash groups
		for _, grp := range hashGroups {
			order.PutUint32(u32[:], uint32(grp.length))
			buf.Write(u32[:])
			order.PutUint32(u32[:], uint32(grp.index))
			buf.Write(u32[:])
		}

		// Write flat hashes at hashesOffset
		data = buf.Bytes()
		order.PutUint64(data[hashesOffsetPos:hashesOffsetPos+8], uint64(buf.Len()))
		for _, ph := range flatHashes {
			order.PutUint32(u32[:], ph.hash)
			buf.Write(u32[:])
			order.PutUint32(u32[:], uint32(ph.index))
			buf.Write(u32[:])
		}
	}

	// Update headersEnd and hashTableOffset in BND4 header
	data = buf.Bytes()
	order.PutUint64(data[headersEndPos:headersEndPos+8], uint64(buf.Len()))
	order.PutUint64(data[hashTableOffsetPos:hashTableOffsetPos+8], hashTableOffset)

	// Write file payload data (aligned to 16 bytes)
	for i, f := range files {
		if len(f.Data) > 0 {
			for buf.Len()%16 != 0 {
				buf.WriteByte(0)
			}
		}
		dataOffset := uint64(buf.Len())
		buf.Write(f.Data)
		data = buf.Bytes()

		size := uint64(len(f.Data))
		order.PutUint64(data[headerPositions[i].compressedPos:headerPositions[i].compressedPos+8], size)
		if hasCompression {
			order.PutUint64(data[headerPositions[i].uncompressedPos:headerPositions[i].uncompressedPos+8], size)
		}
		if longOffsets {
			order.PutUint64(data[headerPositions[i].dataOffsetPos:headerPositions[i].dataOffsetPos+8], dataOffset)
		} else {
			order.PutUint32(data[headerPositions[i].dataOffsetPos:headerPositions[i].dataOffsetPos+4], uint32(dataOffset))
		}
	}

	return buf.Bytes(), nil
}

// BND3Options configures serialization of BND3 containers (Dark Souls 1 / Demon's Souls).
type BND3Options struct {
	Version      string // 8-char version, e.g. "00000001"
	Format       byte   // Default: 0x74 or 0x2E
	BigEndian    bool   // Default: false
	BitBigEndian bool   // Default: false
}

// DefaultBND3Options returns default BND3 options for Dark Souls 1.
func DefaultBND3Options() BND3Options {
	return BND3Options{
		Version:      "00000001",
		Format:       0x2E,
		BigEndian:    false,
		BitBigEndian: false,
	}
}

// ParseBND3 reads files from a FromSoftware BND3 container.
func ParseBND3(data []byte) ([]BND4File, error) {
	if len(data) < 32 || string(data[:4]) != "BND3" {
		return nil, fmt.Errorf("invalid BND3 header")
	}
	bitBigEndian := data[14] != 0
	rawFmt := data[12]
	if !bitBigEndian {
		rawFmt = bits.Reverse8(rawFmt)
	}
	format := rawFmt
	be := data[13] != 0 || (format&1 != 0)
	var order binary.ByteOrder = binary.LittleEndian
	if be {
		order = binary.BigEndian
	}
	count := int(order.Uint32(data[16:20]))
	if count < 0 || count > 1<<20 {
		return nil, fmt.Errorf("invalid BND3 file count: %d", count)
	}

	longOffsets := format&0x10 != 0
	hasCompression := format&0x20 != 0
	hasIDs := format&2 != 0
	hasNames := format&0x0c != 0

	per := 8
	if longOffsets {
		per += 8
	} else {
		per += 4
	}
	if hasIDs {
		per += 4
	}
	if hasNames {
		per += 4
	}
	if hasCompression {
		per += 4
	}

	if 32+count*per > len(data) {
		return nil, fmt.Errorf("BND3 header table exceeds file size")
	}

	files := make([]BND4File, 0, count)
	p := 32
	for i := 0; i < count; i++ {
		flags := data[p]
		if !bitBigEndian {
			flags = bits.Reverse8(flags)
		}
		p += 4
		compressed := int(order.Uint32(data[p : p+4]))
		p += 4
		var off int64
		if longOffsets {
			off = int64(order.Uint64(data[p : p+8]))
			p += 8
		} else {
			off = int64(order.Uint32(data[p : p+4]))
			p += 4
		}
		id := int32(-1)
		if hasIDs {
			id = int32(order.Uint32(data[p : p+4]))
			p += 4
		}
		name := ""
		if hasNames {
			no := int(order.Uint32(data[p : p+4]))
			p += 4
			if no >= 0 && no < len(data) {
				name = readBND4Name(data[no:], false, be)
			}
		}
		uncompressed := compressed
		if hasCompression {
			uncompressed = int(order.Uint32(data[p : p+4]))
			p += 4
		}
		if off < 0 || int(off)+compressed > len(data) {
			return nil, fmt.Errorf("BND3 file %d range invalid: offset=%d len=%d total=%d", i, off, compressed, len(data))
		}
		payload := append([]byte(nil), data[off:int(off)+compressed]...)
		if flags&1 != 0 && uncompressed > 0 {
			raw, err := inflateBND4(payload, uncompressed)
			if err != nil {
				return nil, fmt.Errorf("decompress BND3 file %d: %w", i, err)
			}
			payload = raw
		}
		files = append(files, BND4File{ID: id, Name: name, Data: payload})
	}
	return files, nil
}

// WriteBND3 serializes files into a FromSoftware BND3 container.
func WriteBND3(files []BND4File, opts BND3Options) ([]byte, error) {
	if opts.Format == 0 {
		opts = DefaultBND3Options()
	}
	var order binary.ByteOrder = binary.LittleEndian
	if opts.BigEndian {
		order = binary.BigEndian
	}

	buf := new(bytes.Buffer)
	buf.WriteString("BND3")
	vBytes := make([]byte, 8)
	copy(vBytes, []byte(opts.Version))
	buf.Write(vBytes)

	rawFmt := opts.Format
	if !opts.BitBigEndian {
		rawFmt = bits.Reverse8(rawFmt)
	}
	buf.WriteByte(rawFmt)
	beByte := byte(0)
	if opts.BigEndian {
		beByte = 1
	}
	buf.WriteByte(beByte)
	bbeByte := byte(0)
	if opts.BitBigEndian {
		bbeByte = 1
	}
	buf.WriteByte(bbeByte)
	buf.WriteByte(0)

	var u32 [4]byte
	order.PutUint32(u32[:], uint32(len(files)))
	buf.Write(u32[:])

	fileHeadersEndPos := buf.Len()
	buf.Write(u32[:]) // placeholder for fileHeadersEnd
	order.PutUint32(u32[:], 0)
	buf.Write(u32[:]) // Unk18
	buf.Write(u32[:]) // 0

	format := opts.Format
	longOffsets := format&0x10 != 0
	hasCompression := format&0x20 != 0
	hasIDs := format&2 != 0
	hasNames := format&0x0c != 0

	type fileHdrPos struct {
		compressedPos   int
		dataOffsetPos   int
		nameOffsetPos   int
		uncompressedPos int
	}
	hdrPositions := make([]fileHdrPos, len(files))

	for i, f := range files {
		flagByte := byte(0x40)
		if !opts.BitBigEndian {
			flagByte = bits.Reverse8(flagByte)
		}
		buf.WriteByte(flagByte)
		buf.Write([]byte{0, 0, 0})

		hdrPositions[i].compressedPos = buf.Len()
		buf.Write(u32[:]) // compressed size

		hdrPositions[i].dataOffsetPos = buf.Len()
		if longOffsets {
			var u64 [8]byte
			buf.Write(u64[:])
		} else {
			buf.Write(u32[:])
		}

		if hasIDs {
			order.PutUint32(u32[:], uint32(f.ID))
			buf.Write(u32[:])
		}

		if hasNames {
			hdrPositions[i].nameOffsetPos = buf.Len()
			buf.Write(u32[:])
		}

		if hasCompression {
			hdrPositions[i].uncompressedPos = buf.Len()
			buf.Write(u32[:])
		}
	}

	// Write file names
	data := buf.Bytes()
	for i, f := range files {
		if !hasNames {
			continue
		}
		namePos := uint32(buf.Len())
		order.PutUint32(data[hdrPositions[i].nameOffsetPos:hdrPositions[i].nameOffsetPos+4], namePos)
		buf.WriteString(f.Name)
		buf.WriteByte(0)
		data = buf.Bytes()
	}

	// FileHeadersEnd
	order.PutUint32(data[fileHeadersEndPos:fileHeadersEndPos+4], uint32(buf.Len()))

	// Write payload data
	for i, f := range files {
		if len(f.Data) > 0 {
			for buf.Len()%16 != 0 {
				buf.WriteByte(0)
			}
		}
		dataOffset := uint32(buf.Len())
		buf.Write(f.Data)
		data = buf.Bytes()

		size := uint32(len(f.Data))
		order.PutUint32(data[hdrPositions[i].compressedPos:hdrPositions[i].compressedPos+4], size)
		if hasCompression {
			order.PutUint32(data[hdrPositions[i].uncompressedPos:hdrPositions[i].uncompressedPos+4], size)
		}
		if longOffsets {
			order.PutUint64(data[hdrPositions[i].dataOffsetPos:hdrPositions[i].dataOffsetPos+8], uint64(dataOffset))
		} else {
			order.PutUint32(data[hdrPositions[i].dataOffsetPos:hdrPositions[i].dataOffsetPos+4], dataOffset)
		}
	}

	return buf.Bytes(), nil
}
