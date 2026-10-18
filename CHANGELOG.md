# Changelog

All notable changes to FarsiForge will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v2.0.0] - 2026-10-18

### Added
- **WinRAR-Style Archive Explorer in Workbench**: Replaced flat string list with a dual-pane WinRAR-like split-view containing an archive/package tree on the left, an archive-aware file table with monospace source typography and RTL Persian columns on the right, and a docked bottom Dialogue Inspector & Editor with quick navigation and copy actions.
- **Live Unpacking Exploration Feedback**: Display live unpacking container and asset paths in Step 2 during extraction.

- **Engine-Aware Compiled Unity Asset Inspector**:
  - Implemented real serialized object browsing via `UnityPy` with lazy pagination, exact field paths, object types, and write-back safety indicators in `pkg/tools/scripts/inspect_asset.py` and `pkg/assetbrowser/unity.go`.
  - Decoupled physical compiled asset explorer tree from translation candidates view in `frontend/src/app/page.tsx`, eliminating false-positive dialogue labelling on raw TextAssets.
- **Compiled Unity Font Discovery & Safe Replacement**:
  - Upgraded `pkg/fonts/scanner.go` to inspect compiled `Font` objects (ClassID 128) inside `.assets` files (`resources.assets`, `sharedassets0.assets`), discovering embedded TTF/OTF fonts (`LiberationSans`, `PerfectDOSVGA437`, `NotoSansKR`, `GenYoGothic`) identical to AssetRipper.
  - Implemented transactional font export and font replacement staging (`inspect_asset.py --replace-font`, `StageFontReplacement`) keeping original game roots byte-for-byte untouched.
- **Automated Regression Suite (Scenarios A, B, C, D)**:
  - Added full test coverage in `test/e2e/unity_regression_scenarios_test.go` validating synthetic noisy asset extraction (A), safe staging outside game root with patch build & stale source rejection (B), candidate classification precision/recall (C), and project schema backward compatibility with pagination (D).

### Fixed
- **Unity Real-Game Extraction Architecture Overhaul**:
  - Fixed silent drop of all `TextAsset` objects in `pkg/tools/scripts/extract.py` and `inject.py`: replaced `data.name` and `data.script` with `getattr(data, "m_Name", ...)` and `getattr(data, "m_Script", ...)`, eliminating uncaught `AttributeError` on UnityPy generated classes.
  - Fixed overly aggressive identifier regex `_PASCAL_RE`: eliminated `^[A-Z][a-zA-Z0-9]*$` which previously misclassified capitalized ordinary English words (`Options`, `Sound`, `Video`, `Thalia`, `Comics`, `Back`) as code variables and discarded them.
  - Structured TextAsset decomposition: broke JSON and CSV assets into discrete translatable units (`#jpath` and `#row[r].col[c]`) instead of giant multi-kilobyte monoliths.
  - Verified 100% extraction recall across all 15 game screenshots on real Unity title (`Orc Massage`, 22,634 items extracted).
- **Critical Unity Python RAM 100% Exhaustion**:
  - Replaced in-memory array accumulation with direct streaming JSON writer to disk.
  - Switched duplicate tracking to 64-bit integer hashes (`seen_hashes`).
  - Filtered out non-text multi-gigabyte media bundles (`character`, `animator`, `texture`, `audio`) and bundles `>250MB`.
  - Added LRU cache cap (500) on `tt_cache` and skipped engine/editor assembly loops.
  - Lowered raw byte string size limit from 16MB to 64KB and added 2MB guard on serialized scans.
  - Added periodic garbage collection every 200 objects and explicit file/object cleanup.
  - Switched Go `UnityExtractor.Extract` to streaming `json.NewDecoder`.
- **Pre-Extraction Asset Browser Streamlining**: Removed redundant pre-extraction file structure browse button from detection step.

- **Atomic CSV/XLSX Exchange**: Added validate-all-first import policy for Excel (`.xlsx`) and `.csv` files. Zero modifications are applied if any row contains duplicate IDs, unknown entries, or invalid statuses.
- **Export & Import UI**: Added direct Export and Import buttons in the translation workbench with native file picker dialogs (`SelectSaveFile` and `SelectOpenFile`) and atomic import reporting.
- **Unified Validation Pipeline**: Reconciled `pkg/validate.StandardValidator` with `pkg/textfilter.QA` as the single source of truth for placeholder, tag, bracket, and brace integrity.
- **Pre-Injection Validation Gate**: Injection automatically validates all translated entries and blocks with an actionable summary if any hard errors (`SevError`: missing placeholders, unclosed tags, unbalanced braces) are detected.
- **Workbench Virtualized Pagination**: Replaced the previous 200-item slice with full pagination (50 items/page) and navigation controls, allowing translators to reach every entry in projects with 100k+ strings.
- **Explicit Save & QA Counters**: Added a "Save Changes" button with feedback state and a live QA issues counter badge that filters directly to problematic entries.
- **Persian Options Persistence**: Added `SavePersianOptions` in `app.go` and wired UI toggles (RTL, shaping, Persian digits) so user selections persist and pass directly into the injection pipeline.
- **Engine Capability Drift Tests**: Added `TestEngineSupportMatrixDrift` in `test/e2e` to verify that all detected engines map to explicit injector or extract-only/detect-only contracts.
- **FromSoftware & Unreal Archive Container Identity**: Preserved `Container` identity on all strings extracted from FromSoftware `Data0.bhd` archives and Unreal `.pak` / `.utoc` containers, marking archive-nested members clearly as read-only.
- **Patcher FFP1 Bounds & Bomb Hardening**: Hardened `FarsiForgePatcher.cs` against malicious or corrupt patches with string length limits (64 KB), target limits (100k), record limits (1M), stream bound checks, and decompression bomb guards in `Inflate`.
- **CI C# Patcher Build Gate**: Added automatic C# patcher compilation in GitHub Actions `test` job before running Go tests, guaranteeing cross-validation regressions run on every PR and commit.
- **Transactional Package Creation**: `installer.Build` now stages all package components (`FarsiForgePatch.ffp1`, patcher binary, optional font, and README) in a temporary staging directory and publishes them atomically only after every step succeeds.
- **Portable Patch Mode**: `installer.Build` generates context-aware bilingual README instructions distinguishing standalone self-extracting packages (when `FarsiForgePatcher.exe` is bundled) from portable patch-only distributions (when no patcher is bundled).
- **Filesystem & Backup Safety**: Hardened `BackupManager` against Windows device names (`CON`, `PRN`, `AUX`, `NUL`, `COM1-9`, `LPT1-9`), NTFS data streams, and symlink escapes with `verifyWithinRoot`. Added nil guards to `RollbackAll`.
- **Unified Engine Capability Matrix**: Centralized engine capabilities in `pkg/core/capabilities.go` and exposed via `GetEngineCapabilities` / `GetEngineCapability`. Replaced hardcoded frontend arrays with live capability query reporting detection, extraction, injection, archive rebuild, and font readiness with contextual guidance notes.
- **Scanner Diagnostics & Scan Visibility**: Added `ScanDiagnostics` and `WalkDirWithDiagnostics` in `pkg/scanner/scanner.go` to explicitly surface skipped directories, permission denials, and scan truncation.
- **Granular Pipeline Progress Events**: Added `pipeline:progress` Wails runtime event emission across extraction, injection, and patch building with real percentage values, stage names, and file counters.
- **Rich In-App Diagnostics Toast**: Replaced browser `alert()` popups with non-blocking error banners featuring actionable error explanations and a "کپی گزارش کامل خطا (Diagnostics)" button for instant clipboard export.
- **Interactive Settings Modal**: Added full Settings configuration dialog in workbench header wired to `GetSettings` and `SaveSettings`, persisting `tools_dir`, `output_dir`, and default author.
- **Pip Package Installation Allowlist**: Restriced `InstallPythonPackage` in `app.go` strictly to pre-approved FarsiForge dependencies (`UnityPy`, `Pillow`, `brotli`), rejecting unapproved packages before accessing the environment.
- **Precision-First Text Candidate Classification (Task B)**: Built a comprehensive extraction classification pipeline in `pkg/textfilter/classifier.go` combining structural evidence (localization tables, Dialogue/UI/Item metadata), syntax checks (placeholders, markup tags, brackets), natural-language heuristics, and write-back safety contracts.
- **Semantic Categories & Quality Classes**: Defined core candidate classes (`high-confidence`, `needs-review`, `technical`, `raw-heuristic`) and semantic categories (`dialogue`, `ui`, `item`, `quest`, `tutorial`, `system`, `unclassified`) with 0.0-1.0 confidence scoring.
- **Container Write-Back Protection**: Enforced strict `ReadOnly` marking with descriptive `ReadOnlyReason` on archive-resident strings (Unreal `.pak`/`.utoc` and FromSoftware `Data0.bhd`) and unparsed binary byte scans.
- **Labelled Golden Fixture Benchmark**: Added `pkg/textfilter/testdata/golden_corpus.json` and `classifier_test.go` regression test harness achieving 100% precision and 100% recall across engines and noise scenarios.
- **Extraction Statistics Reporting**: Added `ExtractionStats` attached to `core.Project` and populated during extraction, tracking accepted, reviewable, technical, and read-only candidate metrics.
- **Workbench Candidate Quality & Category UI**: Added quality filter chips (`همه`, `با اطمینان بالا`, `نیازمند بازبینی`, `خام`, `فقط‌خواندنی`, `فنی`), semantic category dropdown selector, row category badges, read-only lock notices, and interactive promotion controls.
- **Game Launch Safety**: `LaunchGame` in `app.go` now validates executable paths against traversal, root boundary escape, directory execution, and empty parameters.
- **Game-Font Inventory & SFNT Parser (Task C)**: Built a native, zero-dependency SFNT OpenType parser in `pkg/fonts/sfnt.go` extracting family/subfamily metadata from the `name` table, Unicode character map coverage from `cmap` (format 4 and 12), and Arabic shaping feature presence (`GSUB`).
- **Persian Glyph Coverage Diagnostics**: Implemented `EvaluatePersianCoverage` testing Persian unique letters (گ چ پ ژ ی ک), Arabic base characters, and Persian digits (۰-۹), outputting detailed coverage percentages and missing rune lists.
- **Multi-Engine Font Scanner**: Built `pkg/fonts/scanner.go` detecting direct vector fonts (`.ttf`, `.otf`, `.ttc`), BMFont atlases (`.fnt`), and engine-compiled resources across Unreal (`.uasset`), Unity (`.fontsettings`, `.assets`), Godot (`.tres`, `.res`), and FromSoftware glyph archives (`.bnd`, `.dcx`).
- **Transactional Font Replacement Staging**: Implemented `pkg/fonts/replacement.go` to safely stage custom font replacements under `Project.WorkingDir/out/<targetRelPath>` with source and target SHA-256 hash tracking, guaranteeing that installed game files are never mutated.
- **End-to-End Font Packaging**: Connected staged fonts into `Project.FontPath` and `installer.BuildConfig.FontFile` during patch creation in `BuildPatcher`.
- **Workbench Font Studio & Diagnostics Modal**: Added a dedicated Font Typography sidebar card and interactive modal displaying font inventory, Persian glyph readiness badges, GSUB shaping detection, and direct custom font replacement file selection.
- **Engine-Aware Compiled-Asset Browser & Source Provenance (Task A)**: Built `pkg/assetbrowser` providing hierarchical exploration of game directories, containers, asset files, and parsed resources before or alongside translation extraction.
- **Multi-Engine Browse Providers**: Built specialized providers for Unreal Engine (`.pak`, `.utoc`, `.locres`, `.uexp`), FromSoftware (`Data0.bhd`, `.msgbnd.dcx`, `.fmg`), Godot (`.pck`, `.tres`, `.translation`), Unity (`.assets`, `.bundle`, `.resS`, text assets), and generic structured files.
- **Write-Back Safety Contracts in Browser**: Every discovered asset explicitly reports its parsing status (`parsed`, `opaque_container`, `raw_heuristic`, `unsupported`) and editability contracts (`can_extract`, `can_inject`, `read_only`, and `read_only_reason`).
- **Interactive Asset Browser Modal & Tree Navigation**: Added interactive split-view modal in `frontend/src/app/page.tsx` with folder/container tree expand/collapse, quick filters (`all`, `containers`, `extractable`, `readonly`), real-time search, and a metadata/provenance detail inspector.
- **Fine-Grained Tool Diagnostics & Per-Operation Verification (Task 12)**:
  - Bounded subprocess output buffer (`RunSilentBounded`) with `DefaultMaxOutputBytes = 2MB` to protect against memory exhaustion from runaway subprocesses.
  - Native PE header architecture inspection via `debug/pe` (`DetectPEArchitecture`) accurately identifying `x86_64`, `x86`, and `arm64` binary targets and classifying corrupt/damaged executables.
  - Per-operation prerequisite checker (`CheckOperationRequirements`) determining whether tools needed for `extract`, `inject`, `rebuild_archive`, or `build_patch` are present, compatible, and healthy for each engine (Unreal `repak`/`retoc`, Unity `python`/`unitypy`, Godot `fftools`/`gdre_tools`, FromSoftware `fftools`/`oo2core`, RAGE `magic.dat`, and standalone `FarsiForgePatcher.exe`).
  - Diagnostic report generator (`GenerateDiagnosticReport`) producing structured metadata and copyable/downloadable Markdown reports.
  - Exposed via `App` API: `GetDetailedToolDiagnostics`, `CheckOperationPrerequisites`, and `RefreshTools`.
- **Release Manifest & SBOM Generator (Task 6)**:
  - Built `pkg/sbom/sbom.go` and standalone CLI `scripts/generate_sbom.go` capturing tool versions, licenses, source revisions, SHA-256 hashes, Go/Node versions, and target binary digests.
  - Emits both structured `sbom.json` and human-readable `RELEASE_MANIFEST.md`.
  - Incorporated into `.github/workflows/release.yml` with artifact publishing.
- **Clean-Machine Smoke Test (Task 6)**:
  - Added `test/smoke/smoke_test.go` verifying clean tool discovery, project setup, staged translation, FFP1 patch building, transactional apply, and byte-for-byte uninstall rollback with 100% pass.
- **Expanded Real-Games Validation Suite (`D:\games`)**:
  - Expanded `test/e2e/real_games_test.go` to test 10 real game installations in `D:\games` (`ELDEN RING`, `Sekiro`, `Factorio`, `Project.Zomboid`, `Raft`, `Orc Massage` [Unity], `MOLDRISE` [Godot], `Hacker Simulator` [Unreal], `C&C Generals Zero Hour`, `Command and Conquer Red Alert 3`).
  - Tested engine detection, asset browser scans, font scans, and operation requirement validation with 100% pass and zero game file mutations.
- **Large Game Memory Benchmarks (`test/bench/memory_bench_test.go`) (Task 16)**:
  - Added comprehensive benchmark suite for real games in `D:\games`.
  - Automated peak memory tracing with Python's `tracemalloc`, verifying memory stays under 350 MB on full 7.5 GB games (tested on `Raft`, dropping peak RAM from 4+ GB to 243 MB with 59,097 strings).
  - Benchmarked engine detection (8 real games in ~11s) and asset browser browsing (12.9 ms/op).

### Changed
- **Unity Extraction & Injection Memory Architecture (Task 16)**:
  - Fixed root cause of Python memory leaks and machine freezes. Replaced whole-directory `UnityPy.load(data_dir)` with candidate-filtered streaming.
  - Excluded non-serialized media stream buffers (`.resS`, `.resource`), audio, textures, and executables.
  - Added file-by-file streaming with unconditional immediate garbage collection (`del env; gc.collect()`) after every file.
  - In `inject.py`, targeted ONLY the specific asset files that contain translated strings from `translated.json`.
  - Added recursion depth (`max_depth=10`) and array length (`[:250]`) caps in `_walk_value` to prevent infinite loops and memory blowup on binary vertex/geometry buffers.
- **Subprocess Safety Timeouts**:
  - Added 15-minute fallback timeout and explicit context deadline cancellation handling to `runPythonScript` in `pkg/extract/engines.go` and `pkg/inject/engines.go`.
- **FromSoftware Staging Contract**: Routed all FromSoftware injector output to `Project.WorkingDir/out/` instead of mutating files inside `GameRoot`. Installed game files remain strictly read-only.
- **FromSoftware Archive Write-Back Safety**: Reject injection of archive-nested strings (`Container != ""`) with clear actionable guidance, ensuring only loose FMG and binder files can stage output.
- **Unreal Write-Back Contract**: Ensured `UnrealInjector` only stages loose `.locres` files with full original SHA-256 hash binding, explicitly refusing to stage phantom loose files for archive-resident entries.
- **Atomic Configuration Persistence**: `SaveConfig` in `pkg/core/config.go` now writes to a temporary sibling file, flushes/syncs to disk, and renames atomically to prevent corrupted files on crash or power loss.
- **Settings Path Validation**: `SaveSettings` in `app.go` now validates custom tools and output directories, ensuring tool registries initialize cleanly before persisting settings.
- **Release CI Hardening**: Pinned Go version to `1.26.0` matching `go.mod`, added mandatory test and frontend-check gates before build, and scoped permissions to `contents: read` globally with `contents: write` restricted to release jobs.
- **Bug Fix**: Fixed `fmtMatches` rune calculation in `pkg/validate` where counts >= 10 produced corrupted runes.
