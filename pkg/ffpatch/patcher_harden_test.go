package ffpatch

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatcherRejectsMalformedHeadersAndLimits(t *testing.T) {
	cli := cliExe()
	if cli == "" {
		t.Skip("FarsiForgePatcherCli.exe not built; run patcher/build.ps1 first")
	}

	tests := []struct {
		name       string
		data       []byte
		errPattern string
	}{
		{
			name:       "invalid magic",
			data:       []byte("XYZ1\x01\x00\x00\x00"),
			errPattern: "not a valid FFP1 patch",
		},
		{
			name:       "unsupported version 0",
			data:       []byte("FFP1\x00\x00\x00\x00"),
			errPattern: "unsupported patch version",
		},
		{
			name:       "unsupported version 99",
			data:       []byte("FFP1\x63\x00\x00\x00"),
			errPattern: "unsupported patch version",
		},
		{
			name: "oversized string length",
			data: func() []byte {
				buf := []byte("FFP1\x02\x00\x00\x00")
				// Write string with length 1,000,000 bytes (exceeds 64 KB limit)
				lenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(lenBytes, 1000000)
				return append(buf, lenBytes...)
			}(),
			errPattern: "invalid or truncated string",
		},
		{
			name: "massive target count",
			data: func() []byte {
				// Valid magic + ver 2 + 8 empty strings (each len 0)
				buf := []byte("FFP1\x02\x00\x00\x00")
				for i := 0; i < 8; i++ {
					buf = append(buf, 0, 0, 0, 0)
				}
				// targetCount = 0xFFFFFFFF
				buf = append(buf, 0xFF, 0xFF, 0xFF, 0xFF)
				// blobSize = 0
				buf = append(buf, 0, 0, 0, 0, 0, 0, 0, 0)
				return buf
			}(),
			errPattern: "target count exceeds patch limits",
		},
		{
			name: "invalid negative blob size",
			data: func() []byte {
				buf := []byte("FFP1\x02\x00\x00\x00")
				for i := 0; i < 8; i++ {
					buf = append(buf, 0, 0, 0, 0)
				}
				// targetCount = 0
				buf = append(buf, 0, 0, 0, 0)
				// blobSize = -1 (0xFFFFFFFFFFFFFFFF)
				buf = append(buf, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)
				return buf
			}(),
			errPattern: "invalid patch blob size",
		},
		{
			name: "blob size exceeding file length",
			data: func() []byte {
				buf := []byte("FFP1\x02\x00\x00\x00")
				for i := 0; i < 8; i++ {
					buf = append(buf, 0, 0, 0, 0)
				}
				// targetCount = 0
				buf = append(buf, 0, 0, 0, 0)
				// blobSize = 1000000000
				bs := make([]byte, 8)
				binary.LittleEndian.PutUint64(bs, 1000000000)
				buf = append(buf, bs...)
				return buf
			}(),
			errPattern: "invalid patch blob size",
		},
		{
			name: "truncated target entry",
			data: func() []byte {
				buf := []byte("FFP1\x02\x00\x00\x00")
				for i := 0; i < 8; i++ {
					buf = append(buf, 0, 0, 0, 0)
				}
				// targetCount = 1
				buf = append(buf, 1, 0, 0, 0)
				// blobSize = 0
				buf = append(buf, 0, 0, 0, 0, 0, 0, 0, 0)
				// Only 5 bytes of target data (needs path, mode, sizes, hashes)
				buf = append(buf, 1, 2, 3, 4, 5)
				return buf
			}(),
			errPattern: "too short",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			patchPath := filepath.Join(t.TempDir(), "malformed.ffpatch")
			if err := os.WriteFile(patchPath, tc.data, 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(cli, "--info", patchPath)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected patcher to reject malformed input, but succeeded: %s", out)
			}
			if !strings.Contains(strings.ToLower(string(out)), strings.ToLower(tc.errPattern)) {
				t.Fatalf("expected error containing %q, got: %s", tc.errPattern, out)
			}
		})
	}
}
