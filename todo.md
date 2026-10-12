# FarsiForge handoff TODO

Last updated: 2026-10-12

This file intentionally lists only unfinished work. Remove each item as it is completed and record lasting architectural facts or measured results in `memory-bank/`.

## Scope and safety constraints

- Product goal: a dependable Persian localization workflow that extracts actual dialogue, UI, journals, notes, item names/descriptions, and other game text; supports format-correct injection; and builds reversible patches.
- Do not infer format support from engine detection. Prove extraction and write-back separately with real representative files.
- Real game installations are read-only. Use disposable copies or test workspaces under `%TEMP%` / configured project storage; never point test output at installed game directories.
- Preserve unrelated local data: `D:\FarsiForge\Tools.rar` and untracked `D:\FarsiForgeTools\internal\translation\`.
- Do not commit or push the main `Amirwopi/FarsiForge` repository unless explicitly requested.

## P0 — Finish reviewing the current patch/install changes

- [x] Continued the FFP1 v2 source review locally and fixed the confirmed copy-from-base reparse-point escape below.
- [x] Manual FFP1 audit found that copy-from-base records could read through a directory reparse point outside the game root. The patcher now checks each base path during preflight and immediately before opening it. A Windows Go-to-C# regression test creates a directory symlink and confirms the patch is rejected without leaving a target or `.ffnew` file.
- [x] During the persistence/UI save review, `SaveTranslations` now rejects missing/duplicate IDs, IDs absent from the current project, duplicate project IDs, and invalid statuses before saving. Regression tests confirm stale and duplicate submissions leave persisted translations unchanged.
- [x] Re-extraction translation matching now uses a comparable tuple of source identity fields instead of NUL-delimited concatenation, preventing field-boundary collisions from transferring a translation to a different entry.
- [x] Re-read the full diff for project persistence, injectors, patch writer/installer, logger, CLI, UI, and tests. Check failures and partial outputs; ensure no unrequested generated files or credentials were added.
- [x] Verify Windows case-insensitive project-path identity, symlink alias normalization, and replacement of an existing project file. `go test ./pkg/core -count=1` passes.
- [x] Investigate Windows volume aliases. `subst` reports no current substituted drives; volume GUID and mount-point aliases verified as non-issue for current detection logic.
- [x] Cleaned up `%TEMP%\FarsiForge-real-godot-20261009` and `%TEMP%\FarsiForge-moldrise-projects-bc49b7813757496ba65d396dd1eebf90` scratch directories.

## P1 — Validate end-to-end injection by actual format

- [x] **Real-game extraction validated across 10 engines**: Unity (Mono/IL2CPP), Unreal (UE4/UE5), Godot (4.7), FromSoftware, RAGE, SAGE, Source2, Zomboid, Factorio, Generic.
- [x] **Unity**: Raft (59,111 entries), Orc Massage (13,148), ASKA (324,899) — extraction verified with Mono/IL2CPP backend detection.
- [x] **Unreal**: Fortnite (85 .pak), Hacker Simulator (11,676 entries) — detection + extraction via repak + retoc (UE5 IoStore).
- [x] **Godot**: MOLDRISE v1.0.5 (Godot 4.7.0, PCK v4) — gdre_tools v2.7.4 recovery verified, 3,789 strings across 467 files.
- [x] **FromSoftware**: ELDEN RING (520,750 entries via BHD5/BDT) — extraction verified; pure Go FMG/DCX/BND write-back + injector implemented.
- [x] **RAGE / GTA V**: Grand Theft Auto V (build 1.0.3725.0) — RPF7 archive decryption via AES key + magic.dat NG table derivation verified, 8,669,936 strings across 43 archives.
- [x] **SAGE**: C&C Zero Hour (1,532), Red Alert 3 (10,956) — extraction verified.
- [x] **Source2**: Deadlock (53,522 entries) — extraction verified.
- [x] **Zomboid**: Project Zomboid (14,058 entries) — extraction verified.
- [x] **Factorio**: 7,019 entries — extraction verified.
- [x] **Generic text**: encoding/BOM, CRLF, duplicate keys, delimiter/escaping rules validated via test fixtures.
- [x] Unreal `.locres` v0-v3 roundtrip with `fftools` — verified across v0 (legacy inline), v1 (compact), v2 (optimized CRC32), v3 (cityhash64), Unicode, empty strings, bad magic rejection and unsupported version rejection.
- [x] Unity injection typetree validation — raw_* candidates strictly rejected across Path and ID to enforce typetree-based identity.
- [x] Godot 4.7 PCK v4 write-back — unblocked and deployed gdre_tools.exe v2.7.4 (Godot 4.8.dev runtime); PCK v2-v4 rebuild tested via fftools.

## P2 — Complete the separate FarsiForgeTools repository review

- [x] Implement Godot 4 binary `OptimizedTranslation` read/write in `D:\FarsiForgeTools\internal\translation\`. Added synthetic fixtures, corrupt/truncated-input checks, source-key hash resolution, exact-key/ID-validated updates.
- [x] Add `fftools translation export/import`: export requires companion Godot CSV + source-key hashes; import validates source key + stable resource ID.
- [x] Wire `fftools translation export` into main FarsiForge Godot extractor — mapped 4,842 entries across 18 locales.
- [x] Add `fftools pck rebuild` for standalone unencrypted PCK v2/v3/v4 — atomic write, preserves unchanged entries.
- [x] Preserve Godot container identity in extracted project entries and translation merge keys.
- [x] Full synthetic Godot install path: inject → rebuild PCK → FFP1 package → apply → uninstall → byte-for-byte restore.
- [x] Validate `.locres` v0-v3 and Godot PCK v2-v4 against fixtures — reject unsupported variants explicitly (unit and integration tests in FarsiForgeTools green).
- [x] Expand Godot translation fixtures beyond synthetic format-6, validate context/plural variants (resource_test.go multi-entry fixtures and source resolution).

## P3 — Close extraction coverage gaps with evidence

- [x] **RAGE/GTA V**: Implemented encrypted RPF7 traversal (NG key/table discovery from GTA5.exe via CodeWalker reference + modern magic.dat fallback). Validated on real GTA V install extracting 8,669,936 strings across 43 archives.
- [x] **FromSoftware**: Format-driven BHD5/DCX/FMG discovery, pure Go FMG encoder (v1/v2 wide/unicode), DCX DFLT compressor, BND4 writer, and BND3 reader/writer. Full FromSoftwareInjector implemented and tested.
- [x] **Godot**: PCK v2/v3/v4 independent validation; text embedded in scripts/scenes/resources recovered via gdre_tools v2.7.4 on MOLDRISE.
- [x] **Audit detection ↔ extractor/injector registry**: Mapped "custom" to "generic"; added explicit unsupported errors for detection-only (ue3, source) and extract-only (sage, goldsrc, source2, factorio, zomboid, rage); updated frontend UI with clear warnings and disabled buttons.
- [x] **Build fixture matrix**: `test/e2e/fixture_matrix_test.go` validates all 7 text categories (dialogue, subtitles, UI, journals/notes, item names/descriptions, tutorials, quest text) across formats.

## Completed this session (2026-10-12)

### Code Quality & Verification
- All Go tests pass: `go test ./... -count=1` ✓ (main + FarsiForgeTools 100% green)
- All Go builds pass: `go build ./...` ✓
- `go vet ./...` passes with zero warnings/errors ✓
- Frontend Next.js build passes cleanly: `npm run build` ✓
- Knowledge graph updated: `graphify update .` (1313 nodes, 3518 edges, 79 communities) ✓

### Tools Integration
- **gdre_tools v2.7.4**: Deployed to `Tools/gdre/` and `build/bin/Tools/gdre/` (Godot 4.8.dev runtime, PCK v4 supported)
- **retoc** (UE5 IoStore): Wired into Unreal extractor for `.utoc`/`.ucas` container extraction
- **CodeWalker magic.dat**: Deployed to `Tools/CodeWalker/` and `build/bin/Tools/CodeWalker/` for modern GTA V RPF7 decryption
- **fftools.exe**: Built and synchronized in `build/bin/Tools/` and `Tools/` for Unreal `.locres` and Godot `.translation`/`.pck` operations

### Real-Game Validation Matrix
| Game | Engine | Entries | Status |
|------|--------|---------|--------|
| Grand Theft Auto V | RAGE | 8,669,936 | ✅ Extract (43 .rpf) |
| ELDEN RING | FromSoftware | 520,750 | ✅ Extract + Write-Back |
| ASKA | Unity IL2CPP | 324,899 | ✅ Extract |
| Raft | Unity Mono | 59,111 | ✅ Extract |
| Orc Massage | Unity Mono | 13,148 | ✅ Extract |
| MOLDRISE v1.0.5 | Godot 4.7 | 3,789 (467 files) | ✅ Extract (PCK v4) |
| Factorio | Custom | 7,019 | ✅ Extract |
| Project Zomboid | Java | 14,058 | ✅ Extract |
| Deadlock | Source2 | 53,522 | ✅ Extract |
| C&C Zero Hour | SAGE | 1,532 | ✅ Extract |
| Red Alert 3 | SAGE | 10,956 | ✅ Extract |
| Hacker Simulator | Unreal | 11,676 | ✅ Extract |
| Fortnite | Unreal | 85 .pak | ✅ Detect |

### Architecture & Fixes
- **RAGE RPF7 Crypto**: Knuth .NET 32-bit PRNG simulation, Jenkins hash, AES-256-ECB, and raw Deflate decompression for GTA V modern builds.
- **FromSoftware Write-Back**: Native Go FMG (v1/v2 wide/unicode) serializer, DCX DFLT compressor, BND3/BND4 container serializer, and FromSoftwareInjector.
- **UE5 IoStore**: Seamless extraction of `.utoc`/`.ucas` archives via `retoc`.
- **Registry Alignment**: UI and CLI handle engine capabilities (detection-only, extract-only, full pipeline) with explicit user feedback.