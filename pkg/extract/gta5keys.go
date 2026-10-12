package extract

// RPF7 "NG" decryption for RAGE games (GTA V), ported from the MIT-licensed
// Neodymium reference implementation used by CodeWalker
// (CodeWalker.Core/GameFiles/Utils/GTACrypto.cs + GTAKeys.cs,
// Copyright (c) 2015 Neodymium, MIT license).
//
// Key material is never shipped with FarsiForge. All AES keys, NG keys, NG
// decrypt tables, and the filename-hash LUT are discovered inside the user's
// own game executable by scanning for byte windows whose SHA1 matches the
// published identification hashes (see gta5keys_hashes.go).

import (
	"crypto/aes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"sync"
)

// GTA5Keys is the RPF7 crypto material extracted from a GTA5.exe.
type GTA5Keys struct {
	AESKey          [32]byte            // AES-256-ECB TOC key
	NGKeys          [101][272]byte      // NG expanded keys
	NGDecryptTables [17][16][256]uint32 // NG round decryption tables
	LUT             [256]byte           // filename hash lookup table
}

// ExtractGTA5Keys scans a game executable for RPF7 key material identified by
// its published SHA1 hashes. Modern builds (2024+, e.g. 1.0.3725) no longer
// embed the NG keys and decrypt tables in the executable; for those,
// magicData must carry CodeWalker's magic.dat blob (see KeysFromMagic). The
// executable is only read, never modified.
func ExtractGTA5Keys(exePath string, magicData []byte) (*GTA5Keys, error) {
	exe, err := os.ReadFile(exePath)
	if err != nil {
		return nil, fmt.Errorf("read executable: %w", err)
	}
	keys := new(GTA5Keys)

	aesRes := searchHashes(exe, [][20]byte{pcAESKeyHash}, 0x20)
	if aesRes[0] == nil {
		return nil, fmt.Errorf("AES key not found in %s (unrecognized executable build)", exePath)
	}
	copy(keys.AESKey[:], aesRes[0])

	ngRes := searchHashes(exe, pcNGKeyHashes[:], 0x110)
	if ngRes[0] == nil {
		// Modern GTA V builds removed the embedded NG key material; the AES
		// key is still present and unlocks CodeWalker's magic.dat instead.
		if magicData == nil {
			magicData = findMagicBlob(nil)
		}
		if magicData != nil {
			return KeysFromMagic(magicData, keys.AESKey[:])
		}
		return nil, fmt.Errorf("NG keys not found in %s (modern 2024+ build) and no CodeWalker magic.dat available — place magic.dat in Tools\\CodeWalker", exePath)
	}
	for i, r := range ngRes {
		if r == nil {
			return nil, fmt.Errorf("NG key %d/101 not found in %s (unrecognized executable build)", i, exePath)
		}
		copy(keys.NGKeys[i][:], r)
	}

	tabRes := searchHashes(exe, pcNGDecryptTableHashes[:], 0x400)
	for i, r := range tabRes {
		if r == nil {
			return nil, fmt.Errorf("NG decrypt table %d/272 not found in %s (unrecognized executable build)", i, exePath)
		}
		round, tbl := i/16, i%16
		for v := 0; v < 256; v++ {
			keys.NGDecryptTables[round][tbl][v] = binary.LittleEndian.Uint32(r[4*v:])
		}
	}

	lutRes := searchHashes(exe, [][20]byte{pcLUTHash}, 0x100)
	if lutRes[0] == nil {
		return nil, fmt.Errorf("hash LUT not found in %s (unrecognized executable build)", exePath)
	}
	copy(keys.LUT[:], lutRes[0])

	return keys, nil
}

// searchHashes scans data at 8-byte alignment for windows of the given length
// whose SHA1 matches one of the targets. The result slice is indexed like the
// targets; entries that were not found stay nil. Duplicated matches keep the
// first occurrence.
func searchHashes(data []byte, targets [][20]byte, length int) [][]byte {
	results := make([][]byte, len(targets))
	if length <= 0 || len(data) < length {
		return results
	}
	lookup := make(map[[20]byte]int, len(targets))
	for i := range targets {
		lookup[targets[i]] = i
	}

	steps := (len(data)-length)/8 + 1
	workers := runtime.NumCPU()
	if workers > 16 {
		workers = 16
	}
	if workers < 1 {
		workers = 1
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for s := start; s < steps; s += workers {
				p := s * 8
				sum := sha1.Sum(data[p : p+length])
				idx, ok := lookup[sum]
				if !ok {
					continue
				}
				mu.Lock()
				if results[idx] == nil {
					buf := make([]byte, length)
					copy(buf, data[p:p+length])
					results[idx] = buf
				}
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
	return results
}

// gta5FileNameHash reproduces RAGE's table-based filename hash used to select
// per-file NG keys.
func (k *GTA5Keys) gta5FileNameHash(name string) uint32 {
	var result uint32
	for i := 0; i < len(name); i++ {
		temp := 1025 * (uint32(k.LUT[name[i]]) + result)
		result = (temp >> 6) ^ temp
	}
	return 32769 * (((9 * result) >> 11) ^ (9 * result))
}

// ngKey selects the NG key used for a block of the named file with the given
// on-disk length.
func (k *GTA5Keys) ngKey(name string, length uint32) []byte {
	idx := (k.gta5FileNameHash(name) + length + 61) % 101
	return k.NGKeys[idx][:]
}

// decryptNG decrypts data that was encrypted with the NG key selected for the
// given file name and length.
func (k *GTA5Keys) decryptNG(data []byte, name string, length uint32) []byte {
	return k.decryptNGKey(data, k.ngKey(name, length))
}

// decryptNGKey decrypts a full buffer with a 272-byte NG key. The trailing
// partial block (if any) is left as-is, matching the reference behavior.
func (k *GTA5Keys) decryptNGKey(data, key []byte) []byte {
	out := make([]byte, len(data))
	var keyUints [68]uint32
	for i := range keyUints {
		keyUints[i] = binary.LittleEndian.Uint32(key[4*i : 4*i+4])
	}
	nblocks := len(data) / 16
	for b := 0; b < nblocks; b++ {
		k.decryptNGBlock(data[b*16:b*16+16], &keyUints, out[b*16:b*16+16])
	}
	copy(out[nblocks*16:], data[nblocks*16:])
	return out
}

// decryptNGBlock decrypts one 16-byte block through the 17 NG rounds.
func (k *GTA5Keys) decryptNGBlock(in []byte, key *[68]uint32, out []byte) {
	var cur, nxt [16]byte
	copy(cur[:], in)
	k.ngRoundA(&cur, &nxt, 0, key[0], key[1], key[2], key[3])
	cur, nxt = nxt, cur
	k.ngRoundA(&cur, &nxt, 1, key[4], key[5], key[6], key[7])
	cur, nxt = nxt, cur
	for r := 2; r <= 15; r++ {
		k.ngRoundB(&cur, &nxt, r, key[4*r], key[4*r+1], key[4*r+2], key[4*r+3])
		cur, nxt = nxt, cur
	}
	k.ngRoundA(&cur, &nxt, 16, key[64], key[65], key[66], key[67])
	copy(out, nxt[:])
}

// ngRoundA implements the NG decryption round used for rounds 1, 2, and 17.
func (k *GTA5Keys) ngRoundA(in, out *[16]byte, round int, k0, k1, k2, k3 uint32) {
	t := &k.NGDecryptTables[round]
	x1 := t[0][in[0]] ^ t[1][in[1]] ^ t[2][in[2]] ^ t[3][in[3]] ^ k0
	x2 := t[4][in[4]] ^ t[5][in[5]] ^ t[6][in[6]] ^ t[7][in[7]] ^ k1
	x3 := t[8][in[8]] ^ t[9][in[9]] ^ t[10][in[10]] ^ t[11][in[11]] ^ k2
	x4 := t[12][in[12]] ^ t[13][in[13]] ^ t[14][in[14]] ^ t[15][in[15]] ^ k3
	binary.LittleEndian.PutUint32(out[0:4], x1)
	binary.LittleEndian.PutUint32(out[4:8], x2)
	binary.LittleEndian.PutUint32(out[8:12], x3)
	binary.LittleEndian.PutUint32(out[12:16], x4)
}

// ngRoundB implements the NG decryption round used for rounds 3 through 16.
func (k *GTA5Keys) ngRoundB(in, out *[16]byte, round int, k0, k1, k2, k3 uint32) {
	t := &k.NGDecryptTables[round]
	x1 := t[0][in[0]] ^ t[7][in[7]] ^ t[10][in[10]] ^ t[13][in[13]] ^ k0
	x2 := t[1][in[1]] ^ t[4][in[4]] ^ t[11][in[11]] ^ t[14][in[14]] ^ k1
	x3 := t[2][in[2]] ^ t[5][in[5]] ^ t[8][in[8]] ^ t[15][in[15]] ^ k2
	x4 := t[3][in[3]] ^ t[6][in[6]] ^ t[9][in[9]] ^ t[12][in[12]] ^ k3
	binary.LittleEndian.PutUint32(out[0:4], x1)
	binary.LittleEndian.PutUint32(out[4:8], x2)
	binary.LittleEndian.PutUint32(out[8:12], x3)
	binary.LittleEndian.PutUint32(out[12:16], x4)
}

// decryptAESECB decrypts data with AES-256 in ECB mode without padding. Any
// trailing partial block is copied unchanged.
func decryptAESECB(data, key []byte) []byte {
	out := make([]byte, len(data))
	cipher, err := aes.NewCipher(key)
	if err != nil {
		copy(out, data)
		return out
	}
	n := len(data) &^ 0xF
	for off := 0; off < n; off += 16 {
		cipher.Decrypt(out[off:off+16], data[off:off+16])
	}
	copy(out[n:], data[n:])
	return out
}
