package extract

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"farsiforge/pkg/core"
)

const (
	mbig  = 2147483647
	mseed = 161803398
)

// dotnetRandom reproduces the .NET Framework / .NET Core System.Random PRNG
// (Knuth subtractive algorithm) used by CodeWalker to de-obfuscate magic.dat.
type dotnetRandom struct {
	inext     int
	inextp    int
	seedArray [56]int32
}

func newDotnetRandom(seed int32) *dotnetRandom {
	r := &dotnetRandom{}
	var subtraction int32
	if seed == -2147483648 { // math.MinInt32
		subtraction = 2147483647 // math.MaxInt32
	} else if seed < 0 {
		subtraction = -seed
	} else {
		subtraction = seed
	}

	mj := int32(mseed) - subtraction
	r.seedArray[55] = mj
	var mk int32 = 1
	for i := 1; i < 55; i++ {
		ii := (21 * i) % 55
		r.seedArray[ii] = mk
		mk = mj - mk
		if mk < 0 {
			mk += mbig
		}
		mj = r.seedArray[ii]
	}
	for k := 1; k < 5; k++ {
		for i := 1; i < 56; i++ {
			r.seedArray[i] -= r.seedArray[1+(i+30)%55]
			if r.seedArray[i] < 0 {
				r.seedArray[i] += mbig
			}
		}
	}
	r.inext = 0
	r.inextp = 21
	return r
}

func (r *dotnetRandom) sample() int32 {
	locINext := r.inext + 1
	locINextp := r.inextp + 1
	if locINext >= 56 {
		locINext = 1
	}
	if locINextp >= 56 {
		locINextp = 1
	}
	retVal := r.seedArray[locINext] - r.seedArray[locINextp]
	if retVal == mbig {
		retVal--
	}
	if retVal < 0 {
		retVal += mbig
	}
	r.seedArray[locINext] = retVal
	r.inext = locINext
	r.inextp = locINextp
	return retVal
}

func (r *dotnetRandom) nextBytes(buffer []byte) {
	for i := 0; i < len(buffer); i++ {
		buffer[i] = byte(r.sample() % 256)
	}
}

// jenkHashBytes computes the 32-bit Jenkins one-at-a-time hash of a byte slice,
// matching CodeWalker's JenkHash.GenHash(byte[]).
func jenkHashBytes(data []byte) uint32 {
	var h uint32
	for _, b := range data {
		h += uint32(b)
		h += (h << 10)
		h ^= (h >> 6)
	}
	h += (h << 3)
	h ^= (h >> 11)
	h += (h << 15)
	return h
}

// KeysFromMagic de-obfuscates and decrypts CodeWalker's magic.dat blob using the
// AES key derived from the user's GTA5.exe, producing the full set of 101 NG
// keys, 272 NG decryption tables, and the filename hash lookup table.
func KeysFromMagic(magicData, aesKey []byte) (*GTA5Keys, error) {
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("invalid AES key length: %d (expected 32)", len(aesKey))
	}
	dbl := len(magicData)
	if dbl == 0 {
		return nil, fmt.Errorf("empty magic.dat data")
	}

	seed := int32(jenkHashBytes(aesKey))
	rnd := newDotnetRandom(seed)

	rb1 := make([]byte, dbl)
	rb2 := make([]byte, dbl)
	rb3 := make([]byte, dbl)
	rb4 := make([]byte, dbl)
	rnd.nextBytes(rb1)
	rnd.nextBytes(rb2)
	rnd.nextBytes(rb3)
	rnd.nextBytes(rb4)

	db := make([]byte, dbl)
	for i := 0; i < dbl; i++ {
		db[i] = byte((int(magicData[i]) - int(rb1[i]) - int(rb2[i]) - int(rb3[i]) - int(rb4[i])) & 0xFF)
	}

	db = decryptAESECB(db, aesKey)

	// Magic payload decompresses to 27472 (NG keys) + 278528 (tables) + 256 (LUT) + 16 (AWC key) = 306272 bytes.
	const expectedSize = 306272
	decompressed, err := inflateRawDeflate(db, expectedSize)
	if err != nil {
		return nil, fmt.Errorf("decompress magic data: %w", err)
	}
	if len(decompressed) < expectedSize {
		return nil, fmt.Errorf("decompressed magic data too short: %d < %d", len(decompressed), expectedSize)
	}

	keys := new(GTA5Keys)
	copy(keys.AESKey[:], aesKey)

	// 101 NG keys, 272 bytes each = 27472 bytes
	bp := 0
	for i := 0; i < 101; i++ {
		copy(keys.NGKeys[i][:], decompressed[bp:bp+272])
		bp += 272
	}

	// 17 rounds * 16 tables * 256 uint32s = 278528 bytes
	for round := 0; round < 17; round++ {
		for tbl := 0; tbl < 16; tbl++ {
			for entry := 0; entry < 256; entry++ {
				keys.NGDecryptTables[round][tbl][entry] = binary.LittleEndian.Uint32(decompressed[bp : bp+4])
				bp += 4
			}
		}
	}

	// 256-byte hash LUT
	copy(keys.LUT[:], decompressed[bp:bp+256])
	bp += 256

	// Trailing 16 bytes is AWC key (unused by RPF7)

	return keys, nil
}

// findMagicBlob searches for CodeWalker's magic.dat file across configured
// tool locations, executable neighbors, repository trees, and %APPDATA%.
func findMagicBlob(tools core.ToolRegistry) []byte {
	var candidates []string
	if tools != nil {
		if p := tools.GetPath("codewalker_magic"); p != "" {
			candidates = append(candidates, p)
		}
		if p := tools.GetPath("CodeWalker/magic.dat"); p != "" {
			candidates = append(candidates, p)
		}
	}

	// Conventional relative paths
	candidates = append(candidates,
		filepath.Join("Tools", "CodeWalker", "magic.dat"),
		filepath.Join("build", "bin", "Tools", "CodeWalker", "magic.dat"),
	)

	// Executable neighbors
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "Tools", "CodeWalker", "magic.dat"),
			filepath.Join(exeDir, "..", "Tools", "CodeWalker", "magic.dat"),
			filepath.Join(exeDir, "build", "bin", "Tools", "CodeWalker", "magic.dat"),
		)
	}

	// Cwd tree walk up
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			c := filepath.Join(dir, "Tools", "CodeWalker", "magic.dat")
			candidates = append(candidates, c)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// %APPDATA%\FarsiForge\keys\magic.dat
	if appData := os.Getenv("APPDATA"); appData != "" {
		candidates = append(candidates,
			filepath.Join(appData, "FarsiForge", "keys", "magic.dat"),
			filepath.Join(appData, "FarsiForge", "magic.dat"),
		)
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		if data, err := os.ReadFile(c); err == nil && len(data) > 0 {
			return data
		}
	}
	return nil
}
