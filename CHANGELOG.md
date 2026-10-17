# Changelog

All notable changes to FarsiForge will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-10-17

### Added
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

### Changed
- **FromSoftware Staging Contract**: Routed all FromSoftware injector output to `Project.WorkingDir/out/` instead of mutating files inside `GameRoot`. Installed game files remain strictly read-only.
- **FromSoftware Archive Write-Back Safety**: Reject injection of archive-nested strings (`Container != ""`) with clear actionable guidance, ensuring only loose FMG and binder files can stage output.
- **Unreal Write-Back Contract**: Ensured `UnrealInjector` only stages loose `.locres` files with full original SHA-256 hash binding, explicitly refusing to stage phantom loose files for archive-resident entries.
- **Atomic Configuration Persistence**: `SaveConfig` in `pkg/core/config.go` now writes to a temporary sibling file, flushes/syncs to disk, and renames atomically to prevent corrupted files on crash or power loss.
- **Settings Path Validation**: `SaveSettings` in `app.go` now validates custom tools and output directories, ensuring tool registries initialize cleanly before persisting settings.
- **Release CI Hardening**: Pinned Go version to `1.26.0` matching `go.mod`, added mandatory test and frontend-check gates before build, and scoped permissions to `contents: read` globally with `contents: write` restricted to release jobs.
- **Bug Fix**: Fixed `fmtMatches` rune calculation in `pkg/validate` where counts >= 10 produced corrupted runes.
