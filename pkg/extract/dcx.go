package extract

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// DecompressDCX unwraps common zlib-backed DCX/DCP containers. Oodle-backed
// payloads are deliberately returned as a typed unsupported error until a
// compatible Oodle decoder is available from the game installation.
func DecompressDCX(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("DCX data is truncated")
	}
	if !bytes.Contains(data[:min(len(data), 64)], []byte("DCX")) {
		return nil, fmt.Errorf("invalid DCX signature")
	}
	// Souls formats place the compressed stream after DCA/DCS metadata. Search
	// only a bounded header region for standard zlib framing.
	limit := min(len(data), 256)
	for i := 0; i+2 <= limit; i++ {
		if data[i] != 0x78 || (data[i+1] != 0x01 && data[i+1] != 0x5e && data[i+1] != 0x9c && data[i+1] != 0xda) {
			continue
		}
		r, e := zlib.NewReader(bytes.NewReader(data[i:]))
		if e != nil {
			continue
		}
		out, e := io.ReadAll(r)
		_ = r.Close()
		if e == nil {
			return out, nil
		}
	}
	return nil, fmt.Errorf("DCX compression is not zlib; Oodle support requires the game's oo2core DLL")
}

// DecompressDCXWithOodle also handles KRAK/other Oodle DCX variants using the
// game's own oo2core DLL. The expected size is read from the DCS header.
func DecompressDCXWithOodle(data []byte, dllPath string) ([]byte, error) {
	if out, err := DecompressDCX(data); err == nil {
		return out, nil
	}
	if dllPath == "" {
		return nil, fmt.Errorf("Oodle DCX requires oo2core DLL path")
	}
	if len(data) < 16 {
		return nil, fmt.Errorf("DCX data is truncated")
	}
	codec := ""
	var size uint32
	var compressedSize uint32
	start := -1
	for i := 0; i+12 <= min(len(data), 256); i++ {
		if string(data[i:i+4]) == "DCS\x00" {
			size = binary.BigEndian.Uint32(data[i+4 : i+8])
			compressedSize = binary.BigEndian.Uint32(data[i+8 : i+12])
		}
		for _, tag := range []string{"KRAK", "OODL", "EDGE", "LEVI"} {
			if string(data[i:i+4]) == tag {
				codec = tag
				break
			}
		}
		if codec != "" && size > 0 {
			break
		}
	}
	if codec == "" || size == 0 || compressedSize == 0 {
		return nil, fmt.Errorf("unsupported DCX Oodle header")
	}
	if uint64(size) > 1<<30 {
		return nil, fmt.Errorf("DCX output size exceeds limit")
	}
	for i := 0; i+8 <= min(len(data), 256); i++ {
		if string(data[i:i+4]) == "DCA\x00" {
			start = i + 8
			break
		}
	}
	if start < 0 || uint64(start)+uint64(compressedSize) > uint64(len(data)) {
		return nil, fmt.Errorf("DCX compressed size is invalid")
	}
	return decompressOodle(data[start:start+int(compressedSize)], int(size), dllPath)
}

// CompressDCXDFLT wraps raw data into a standard DCX DFLT (zlib-compressed) container,
// matching the format expected by FromSoftware games (DS1, DS2, DS3, Sekiro, Elden Ring).
func CompressDCXDFLT(data []byte) ([]byte, error) {
	var compressed bytes.Buffer
	zw, err := zlib.NewWriterLevel(&compressed, zlib.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("create zlib writer: %w", err)
	}
	if _, err := zw.Write(data); err != nil {
		return nil, fmt.Errorf("write zlib data: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zlib writer: %w", err)
	}
	cmpBytes := compressed.Bytes()

	var buf bytes.Buffer
	// DCX header (24 bytes)
	buf.WriteString("DCX\x00")
	var u32 [4]byte
	binary.BigEndian.PutUint32(u32[:], 0x10000) // Unk04
	buf.Write(u32[:])
	binary.BigEndian.PutUint32(u32[:], 0x18) // 0x18
	buf.Write(u32[:])
	binary.BigEndian.PutUint32(u32[:], 0x24) // 0x24
	buf.Write(u32[:])
	binary.BigEndian.PutUint32(u32[:], 0x24) // Unk10
	buf.Write(u32[:])
	binary.BigEndian.PutUint32(u32[:], 0x2C) // Unk14
	buf.Write(u32[:])

	// DCS header (12 bytes)
	buf.WriteString("DCS\x00")
	binary.BigEndian.PutUint32(u32[:], uint32(len(data)))
	buf.Write(u32[:])
	binary.BigEndian.PutUint32(u32[:], uint32(len(cmpBytes)))
	buf.Write(u32[:])

	// DCP header (32 bytes)
	buf.WriteString("DCP\x00")
	buf.WriteString("DFLT")
	binary.BigEndian.PutUint32(u32[:], 0x20)
	buf.Write(u32[:])
	buf.WriteByte(9) // compression level
	buf.Write([]byte{0, 0, 0})
	binary.BigEndian.PutUint32(u32[:], 0)
	buf.Write(u32[:])
	buf.WriteByte(0)
	buf.Write([]byte{0, 0, 0})
	binary.BigEndian.PutUint32(u32[:], 0)
	buf.Write(u32[:])
	binary.BigEndian.PutUint32(u32[:], 0x00010100)
	buf.Write(u32[:])

	// DCA header (8 bytes)
	buf.WriteString("DCA\x00")
	binary.BigEndian.PutUint32(u32[:], 8)
	buf.Write(u32[:])

	// Compressed payload
	buf.Write(cmpBytes)

	return buf.Bytes(), nil
}

