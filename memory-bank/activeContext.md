# Active Context: FarsiForge

Updated: 2026-10-13

## Current Checkpoint
- **Engine Capabilities & Real-Game Evidence:**
  - **FromSoftware / Elden Ring:** Full roundtrip write-back verified on Elden Ring Data0 (520,750 entries across 24 bundles). Pure Go FMG encoder (v1/v2 wide/unicode), DCX DFLT compressor, BND4 writer, and FromSoftwareInjector are implemented and tested.
  - **RAGE / GTA V:** Encrypted RPF7 archive traversal implemented with Knuth PRNG + AES-256-ECB table derivation via `magic.dat`. Extracted 8,669,936 strings across 43 archives.
  - **Unreal Engine:** Locres v0–v3 roundtrip verified via `fftools`. IoStore `.utoc`/`.ucas` extraction integrated via `retoc`; `.pak` extraction integrated via `repak`.
  - **Godot Engine:** PCK v2–v4 text recovery integrated via `gdre_tools v2.7.4`. Binary `OptimizedTranslation` update and PCK rebuild verified on synthetic fixtures and MOLDRISE v1.0.5 (3,789 strings).
  - **10 Engine Families Validated:** Unity (Mono/IL2CPP), Unreal (UE4/UE5), Godot (4.7), FromSoftware, RAGE, SAGE, Source2, Project Zomboid, Factorio, and Generic text.
- **CI/CD & Release Pipeline:**
  - GitHub Actions release workflow (`.github/workflows/release.yml`) updated with cross-repository tool builds (`fftools` from `FarsiForgeTools`), C# patcher compilation, helper tool fetching from `FarsiForge-dev` releases, and portable `.zip` artifact packaging.
- **Organization Helper Releases:**
  - Official releases created on `FarsiForge-dev/repak` (v0.2.3), `FarsiForge-dev/retoc` (v0.1.5), and `FarsiForge-dev/CodeWalker` (v1.0.0) for reliable CI dependency downloads.
- **Security & Integrity:**
  - Reparse-point directory escape vulnerability patched and verified with Go-to-C# symlink tests.
  - Staged patch files bound to SHA-256 hashes of base files; stale staged outputs rejected before packaging.
  - Project saves use atomic temporary replacement with deterministic Windows file locking coverage.

## Next Work
- Monitor GitHub Actions release builds via `gh` to verify that automated CI compiles and packages cleanly.
- Explore font texture generation for FromSoftware UI/subtitles.
- Continue tracking roadmap items documented in `todo.md`.
