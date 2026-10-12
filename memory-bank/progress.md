# Project Progress: FarsiForge

Updated: 2026-10-13

## Implemented Foundations
- **Core Architecture:** Engine detectors, native format extractors, staged injectors, Persian RTL text engine, FFP1 patch writer, and Wails desktop application.
- **FromSoftware Ecosystem:** Native pure Go FMG string table serializer, DCX compression/decompression, BND3/BND4 archive writer, and FromSoftwareInjector.
- **RAGE (GTA V) Decryption:** Full encrypted RPF7 archive decryption and traversal via AES-256-ECB and Knuth 32-bit PRNG table derivation using `magic.dat`.
- **Unreal Engine:** Locres v0–v3 roundtrip verified via `fftools`. IoStore `.utoc`/`.ucas` container extraction integrated via `retoc`; `.pak` extraction integrated via `repak`.
- **Godot 3/4 Ecosystem:** PCK v2–v4 text recovery integrated via `gdre_tools v2.7.4`. Binary `OptimizedTranslation` update and PCK rebuild verified on synthetic fixtures and MOLDRISE v1.0.5 (3,789 strings).
- **Patcher Hardening:** FFP1 v2 binds target replacements to original file size and SHA-256 hash. Directory symlink and reparse-point escapes prevented with preflight validation.
- **Automated CI/CD:** `.github/workflows/release.yml` automated workflow builds Wails desktop app, C# patchers, `farsiforge-cli`, `fftools`, stages helper binaries from `FarsiForge-dev`, and packages portable release bundles.

## Real-Game Validation Matrix
| Game | Engine / Family | Extracted Entries | Pipeline Status |
|------|-----------------|-------------------|-----------------|
| **Grand Theft Auto V** | RAGE | 8,669,936 strings across 43 `.rpf` | ✅ Extraction (Decrypted RPF7 + GXT2) |
| **ELDEN RING** | FromSoftware | 520,750 entries across 24 bundles | ✅ Extract + Write-Back (FMG / DCX / BND4) |
| **ASKA** | Unity IL2CPP | 324,899 entries | ✅ Extraction |
| **Deadlock** | Source 2 | 53,522 entries | ✅ Extraction |
| **Raft** | Unity Mono | 59,111 entries | ✅ Extraction |
| **Project Zomboid** | Java / Lua | 14,058 entries | ✅ Extraction |
| **Orc Massage** | Unity Mono | 13,148 entries | ✅ Extraction |
| **Hacker Simulator** | Unreal Engine | 11,676 entries | ✅ Extraction + Staging |
| **Red Alert 3** | SAGE | 10,956 entries | ✅ Extraction |
| **Factorio** | Custom | 7,019 entries | ✅ Extraction |
| **MOLDRISE v1.0.5** | Godot 4.7 | 3,789 strings across 467 files | ✅ Extraction (PCK v4 + GDRE tools) |
| **C&C Zero Hour** | SAGE | 1,532 entries | ✅ Extraction |
| **Fortnite** | Unreal Engine 5 | 85 `.pak` containers | ✅ Detection + Container Indexing |

## Quality Gate Status
- **Go Tests:** `go test ./... -count=1` passes 100% green.
- **Go Vet:** `go vet ./...` clean with 0 warnings.
- **Go Build:** `go build ./...` compiles cleanly.
- **Frontend Build:** `npm run build` passes cleanly.
- **Knowledge Graph:** 128 labeled semantic communities.
