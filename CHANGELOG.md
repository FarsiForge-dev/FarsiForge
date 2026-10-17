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

### Changed
- **FromSoftware Staging Contract**: Routed all FromSoftware injector output to `Project.WorkingDir/out/` instead of mutating files inside `GameRoot`. Installed game files remain strictly read-only.
- **FromSoftware Archive Write-Back Safety**: Reject injection of archive-nested strings (`Container != ""`) with clear actionable guidance, ensuring only loose FMG and binder files can stage output.
- **Unreal Write-Back Contract**: Ensured `UnrealInjector` only stages loose `.locres` files with full original SHA-256 hash binding, explicitly refusing to stage phantom loose files for archive-resident entries.
- **Atomic Configuration Persistence**: `SaveConfig` in `pkg/core/config.go` now writes to a temporary sibling file, flushes/syncs to disk, and renames atomically to prevent corrupted files on crash or power loss.
- **Settings Path Validation**: `SaveSettings` in `app.go` now validates custom tools and output directories, ensuring tool registries initialize cleanly before persisting settings.
- **Release CI Hardening**: Pinned Go version to `1.26.0` matching `go.mod`, added mandatory test and frontend-check gates before build, and scoped permissions to `contents: read` globally with `contents: write` restricted to release jobs.
- **Bug Fix**: Fixed `fmtMatches` rune calculation in `pkg/validate` where counts >= 10 produced corrupted runes.
