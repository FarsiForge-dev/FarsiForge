package extract

// Temporary diagnostic for the GTA V RPF7 file-extraction pipeline: dumps the
// metadata and intermediate bytes of one real .gxt2 entry so the failing
// stage (read / NG decrypt / inflate / GXT2 parse) can be identified.
// Skipped when the real game install is not present.

import (
	"os"
	"path/filepath"
	"testing"

	"farsiforge/pkg/core"
)

func TestDebugRPF7FilePipeline(t *testing.T) {
	gameRoot := `D:\SteamLibrary\steamapps\common\Grand Theft Auto V`
	exePath := filepath.Join(gameRoot, "GTA5.exe")
	if _, err := os.Stat(exePath); err != nil {
		t.Skip("GTA V install not available")
	}
	magicData, err := os.ReadFile(`D:\FarsiForge-dev\FarsiForge\Tools\CodeWalker\magic.dat`)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := ExtractGTA5Keys(exePath, magicData)
	if err != nil {
		t.Fatal(err)
	}
	upPath := filepath.Join(gameRoot, "update", "update.rpf")
	uf, err := os.Open(upPath)
	if err != nil {
		t.Fatal(err)
	}
	defer uf.Close()
	ufi, _ := uf.Stat()
	proj := core.NewProject("test-gta", gameRoot, "rage")
	n, err := extractRPF7GXT2(uf, 0, "update.rpf", uint32(ufi.Size()), keys, "update.rpf", proj, 0)
	if err != nil {
		t.Fatalf("extractRPF7GXT2: %v", err)
	}
	if n == 0 || len(proj.Entries) < 10000 {
		t.Fatalf("expected >= 10000 entries from update.rpf, got n=%d entries=%d", n, len(proj.Entries))
	}
	t.Logf("Extracted %d GXT2 files from update.rpf, total entries in proj: %d", n, len(proj.Entries))
}
