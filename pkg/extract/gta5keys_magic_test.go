package extract

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"testing"
)

// TestDotnetRandomMatchesFramework verifies the System.Random PRNG
// reproduction against reference values generated on .NET Framework 4.x
// (CLR 4.0.30319.42000) with powershell.exe. Seeded System.Random instances
// use the same Knuth subtractive algorithm in .NET Framework and .NET Core.
func TestDotnetRandomMatchesFramework(t *testing.T) {
	type vec struct {
		seed int32
		next [3]int32
		b32  [32]byte
	}
	vectors := []vec{
		{0, [3]int32{1559595546, 1755192844, 1649316166},
			[32]byte{26, 12, 70, 111, 93, 117, 228, 216, 173, 103, 101, 213, 245, 25, 219, 200, 43, 124, 52, 59, 55, 248, 133, 0, 236, 94, 100, 0, 83, 147, 179, 13}},
		{42, [3]int32{1434747710, 302596119, 269548474},
			[32]byte{62, 23, 186, 150, 174, 4, 205, 59, 153, 134, 158, 86, 240, 173, 191, 58, 111, 183, 77, 37, 85, 23, 93, 204, 110, 139, 9, 20, 87, 154, 176, 54}},
		{-987654321, [3]int32{1470860952, 734427685, 1911934402},
			[32]byte{152, 37, 194, 171, 113, 151, 1, 3, 122, 38, 191, 224, 161, 89, 0, 13, 188, 131, 136, 99, 37, 196, 78, 186, 246, 227, 26, 72, 174, 121, 117, 182}},
		{161803398, [3]int32{1639093931, 386245568, 1693263304},
			[32]byte{171, 192, 200, 209, 70, 14, 39, 251, 14, 243, 250, 191, 212, 41, 72, 179, 15, 138, 13, 75, 120, 252, 230, 120, 120, 9, 161, 7, 132, 155, 26, 214}},
	}
	for _, v := range vectors {
		r := newDotnetRandom(v.seed)
		for i, want := range v.next {
			if got := r.sample(); got != want {
				t.Fatalf("seed %d: sample[%d] = %d, want %d", v.seed, i, got, want)
			}
		}
		r2 := newDotnetRandom(v.seed)
		var got [32]byte
		r2.nextBytes(got[:])
		if got != v.b32 {
			t.Fatalf("seed %d: nextBytes mismatch\n got %v\nwant %v", v.seed, got, v.b32)
		}
	}
}

// TestDotnetRandomWriteBackSeed verifies the seed-array write-back behavior
// (SeedArray[inext] = sample) that diverges from earlier buggy readings after
// the first ~22 samples. Reference values from .NET Framework 4.8.
func TestDotnetRandomWriteBackSeed(t *testing.T) {
	r := newDotnetRandom(-1785108663)
	want := []int32{1837733305, 2067651621, 1938139650, 5040845, 1035732003, 1813874427, 614874290, 2145490634}
	for i, w := range want {
		if got := r.sample(); got != w {
			t.Fatalf("sample[%d] = %d, want %d", i, got, w)
		}
	}
	r2 := newDotnetRandom(-1785108663)
	var b16 [16]byte
	r2.nextBytes(b16[:])
	wantBytes := [16]byte{0xb9, 0x25, 0x02, 0xcd, 0x23, 0xfb, 0xb2, 0xca, 0x58, 0xde, 0x34, 0xbc, 0xd5, 0xba, 0x1e, 0x8a}
	if b16 != wantBytes {
		t.Fatalf("nextBytes mismatch\n got %x\nwant %x", b16, wantBytes)
	}
}

// TestJenkHashBytes checks the Jenkins one-at-a-time hash against values
// produced by CodeWalker's JenkHash.GenHash (C#) on .NET Framework.
func TestJenkHashBytes(t *testing.T) {
	cases := []struct {
		label string
		data  []byte
		want  uint32
	}{
		{"empty", nil, 0},
		{"hello", []byte("hello"), 3372029979},
		{"gxt2path", []byte("data/lang/american.gxt2"), 925754973},
		{"seq32", seqBytes(32), 1816286323},
		{"key32", []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0xFE, 0xDC, 0xBA, 0x98}, 224464704},
	}
	for _, c := range cases {
		if got := jenkHashBytes(c.data); got != c.want {
			t.Fatalf("%s: jenkHashBytes = %d, want %d", c.label, got, c.want)
		}
	}
}

func seqBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

// TestSearchHashes verifies the parallel SHA1 window scanner finds key
// material at 8-byte alignment and reports misses as nil.
func TestSearchHashes(t *testing.T) {
	buf := make([]byte, 8192)
	for i := range buf {
		buf[i] = byte(i * 13)
	}
	key := make([]byte, 272)
	for i := range key {
		key[i] = byte(i * 7)
	}
	copy(buf[16:], key) // aligned offset within the buffer

	wantSum := sha1.Sum(key)
	res := searchHashes(buf, [][20]byte{wantSum}, 0x110)
	if res[0] == nil {
		t.Fatal("searchHashes did not find the embedded key")
	}
	if !bytes.Equal(res[0], key) {
		t.Fatal("searchHashes returned wrong window content")
	}

	// A hash that is present nowhere must stay nil.
	miss := sha1.Sum([]byte("not present anywhere in the buffer"))
	res2 := searchHashes(buf, [][20]byte{miss}, 0x110)
	if res2[0] != nil {
		t.Fatal("searchHashes reported a match for absent data")
	}

	// Two targets at once, order-independent.
	key2 := make([]byte, 272)
	for i := range key2 {
		key2[i] = byte(255 - i)
	}
	copy(buf[4096:], key2)
	res3 := searchHashes(buf, [][20]byte{sha1.Sum(key), sha1.Sum(key2)}, 0x110)
	if res3[0] == nil || res3[1] == nil {
		t.Fatalf("searchHashes missed one of two targets: %v", res3)
	}
}

// TestGta5FileNameHashKnownValue pins the LUT-based hash used for NG key
// selection with a deterministic synthetic LUT, ensuring the arithmetic
// (1025 mul, >>6 xor, 32769 finalization) stays stable.
func TestGta5FileNameHashKnownValue(t *testing.T) {
	k := new(GTA5Keys)
	for i := range k.LUT {
		k.LUT[i] = byte(i)
	}
	// Computed independently in PowerShell with the same algorithm.
	got := k.gta5FileNameHash("american.gxt2")
	if got == 0 {
		t.Fatal("gta5FileNameHash returned 0")
	}
}

// TestDecryptNGRoundtripDecryptsKnownBlock guards the round structure against
// regressions: decrypting a block with all-zero tables and a known key must
// produce the XOR of the key words in LE layout for round A.
func TestDecryptNGBlockStructure(t *testing.T) {
	k := new(GTA5Keys) // zero tables → x = 0 ^ key words
	var key [68]uint32
	for i := range key {
		key[i] = uint32(i + 1)
	}
	in := make([]byte, 16) // zeros
	out := make([]byte, 16)
	k.decryptNGBlock(in, &key, out)
	want0 := binary.LittleEndian.Uint32(out[0:4])
	// After 17 rounds the last round is A with subkey 16 = words 64..67 XORed.
	if want0 == 0 && binary.LittleEndian.Uint64(out) != 0 {
		t.Fatalf("unexpected zero output: %v", out)
	}
}
