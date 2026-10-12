package extract

// Temporary diagnostic: prints the AES key and PRNG seed derived from the
// real GTA5.exe so the pipeline can be cross-checked against .NET.
// This test is skipped when the game executable is not present.

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
)

func TestDebugMagicPipelineValues(t *testing.T) {
	exePath := `D:\SteamLibrary\steamapps\common\Grand Theft Auto V\GTA5.exe`
	exe, err := os.ReadFile(exePath)
	if err != nil {
		t.Skip("GTA5.exe not available")
	}
	res := searchHashes(exe, [][20]byte{pcAESKeyHash}, 0x20)
	if res[0] == nil {
		t.Fatal("AES key not found")
	}
	fmt.Println("AESKEY:", hex.EncodeToString(res[0]))
	seed := int32(jenkHashBytes(res[0]))
	fmt.Println("SEED:", seed)

	magic, err := os.ReadFile(`D:\FarsiForge-dev\FarsiForge\Tools\CodeWalker\magic.dat`)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("MAGICLEN:", len(magic))
	rnd := newDotnetRandom(seed)
	n := len(magic)
	s1 := make([]byte, n)
	rnd.nextBytes(s1)
	fmt.Println("RB1_16:", hex.EncodeToString(s1[:16]))
}
