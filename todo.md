# FarsiForge Project Roadmap & TODO

Last updated: 2026-10-13

## Current Status & Verification Checkpoint

All P0 through P3 core engine, format, security, and persistence milestones have been successfully implemented and verified with 100% passing tests.

- **Go Test Suite:** `go test ./... -count=1` passed (100% green across main repository and FarsiForgeTools).
- **Go Build & Vet:** `go build ./...` and `go vet ./...` pass with zero errors/warnings.
- **Frontend Build:** `npm run build` compiles cleanly.
- **Knowledge Graph:** Knowledge graph updated with 128 labeled semantic communities.
- **Automated CI/CD:** GitHub Actions release workflow configured for automated cross-compilation and packaging.

---

## 🎯 Active & Future Milestones

### M1 — Engine Injection Expansion
- [ ] **Unity Typetree Generator**: Automate typetree definition extraction for stripped Unity games to convert candidate `raw_*` strings into safe writable candidates.
- [ ] **Unreal Engine Loose Staging**: Expand automatic discovery of loose `.locres` override directories to simplify patching without repackaging full `.pak` archives.
- [ ] **Godot Locale Auto-Creation**: Add capability to dynamically generate and register a new Persian (`fa`) locale catalog in PCKs that lack existing `.translation` resources.
- [ ] **FromSoftware Font Atlas Injection**: Build specialized font texture/glyph patcher for FromSoftware subtitle and UI renderers.

### M2 — UI & Workbench Enhancements
- [ ] **AI-Assisted Translation Suggestions**: Add optional pluggable LLM translator integration in the workbench UI for batch translating untranslated strings.
- [ ] **Glossary & Terminology Base**: Implement project-level terminology dictionaries to enforce consistent naming across item names, locations, and dialogue.
- [ ] **Visual Diff Inspector**: Add an in-app visual diff viewer to compare original and localized strings with live bidirectional formatting preview.

### M3 — Packaging & Distribution Ecosystem
- [ ] **Automated Release Validation**: Continuously monitor GitHub Actions release pipelines and verify generated `.zip` packages on clean Windows runner environments.
- [ ] **Auto-Updater Integration**: Integrate lightweight updater check in Wails UI to notify translators of new FarsiForge workbench versions.

---

## 📦 Completed Milestones Archive

<details>
<summary><b>Click to expand completed P0–P3 milestones</b></summary>

### P0 — Security, Reparse-Point Audits & Persistence Hardening
- [x] Fixed copy-from-base reparse-point escape in FFP1 v2 patcher. Base paths validated during preflight and right before opening.
- [x] Go-to-C# regression tests verify directory symlinks cannot bypass game root boundaries.
- [x] Hardened `SaveTranslations`: rejects missing/duplicate IDs, IDs absent from current project, duplicate project IDs, and invalid statuses before mutating project file.
- [x] Structured tuple identity for re-extraction translation matching (container, file, key, context, source), eliminating field-boundary collisions.
- [x] Windows case-insensitive project-path identity and volume alias normalization verified.
- [x] Deterministic Windows interrupted-save verification (no-delete-share handle blocks replacement; temporary files cleaned up).

### P1 — Format Extraction & Injection Validation
- [x] **Real-game extraction validated across 10 engines**: Unity (Mono/IL2CPP), Unreal (UE4/UE5), Godot (PCK v2–v4), FromSoftware, RAGE, SAGE, Source2, Project Zomboid, Factorio, and Generic structured text.
- [x] **FromSoftware**: Pure Go FMG encoder (v1/v2 wide/unicode), DCX DFLT compressor, BND4 writer, and FromSoftwareInjector verified on ELDEN RING Data0 (520,750 entries).
- [x] **RAGE / GTA V**: Decrypted RPF7 traversal via AES-256-ECB and Knuth 32-bit PRNG table derivation using `magic.dat`, verified on GTA V build 1.0.3725.0 (8,669,936 strings across 43 archives).
- [x] **Unreal Engine**: `.locres` v0–v3 roundtrip with `fftools`, `.pak` container extraction via `repak`, and IoStore `.utoc`/`.ucas` container extraction via `retoc`.
- [x] **Godot Engine**: PCK v2–v4 extraction via `gdre_tools v2.7.4`, binary `OptimizedTranslation` resource update, and standalone unencrypted PCK rebuild.
- [x] **Unity**: Raw byte scans flagged; typetree validation prevents unsafe writes.

### P2 — FarsiForgeTools Repository Alignment
- [x] Implemented Godot 4 binary `OptimizedTranslation` read/write in `FarsiForgeTools`.
- [x] Added `fftools translation export/import` with source-key hash resolution.
- [x] Added `fftools pck rebuild` for standalone unencrypted PCK v2/v3/v4.
- [x] End-to-end synthetic install test: inject → rebuild PCK → FFP1 package → apply → uninstall → byte-for-byte restore.

### P3 — Registry Alignment & Verification Matrix
- [x] Mapped "custom" engine to "generic" text extractor.
- [x] Explicit unsupported status errors for detection-only (UE3, Source 1) and extract-only (SAGE, GoldSrc, Source 2, Factorio, Project Zomboid, RAGE).
- [x] Frontend UI status badges and disabled buttons prevent invalid writes.
- [x] End-to-end test fixtures validate all 7 text categories (dialogue, subtitles, UI, journals/notes, items, tutorials, quests).

</details>