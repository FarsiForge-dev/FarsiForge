# FarsiForge — Code-Reviewed Engineering Roadmap

**Review date:** 2026-10-17  
**Primary repository:** `FarsiForge-dev/FarsiForge`  
**Companion repository:** `FarsiForge-dev/FarsiForgeTools`

This roadmap is based on a source-level review of the application, Wails/Next.js workbench, engine registries and adapters, patch writer/installer, C# patcher, helper-tool CLI/parsers, tests, GitHub Actions workflows, and organization guidance—not on README claims alone.

**Verification note:** this was a static review of the current GitHub source. The test suites and release builds were not executed as part of this review. The older checkpoint in the archive records the previous project's reported results; it is not a fresh verification of the current branch.

## Priority definitions

- **P0 — Release blocker:** data loss, unintended game-file mutation, a patch that cannot be installed, unsafe malformed-input handling, or a non-reproducible/unchecked release.
- **P1 — Required for a dependable workbench:** reliable editing, truthful capability reporting, diagnostics, CI coverage, and correct import/export.
- **P2 — High-value product improvements:** broader engine support, translation productivity, and large-project performance.
- **P3 — Optional polish:** conveniences that should follow correctness, safety, and test coverage.

---

## P0 — Correctness, data safety, and trustworthy releases

### 1. Make every injector stage output instead of modifying the installed game

- [x] **Fix direct writes in the FromSoftware injector.** In `pkg/inject/fromsoftware.go`, the loose-FMG and BND/DCX branches call `os.WriteFile(targetPath, ...)`, where `targetPath` points into `info.GameRoot`. This violates the staging contract used by the other injectors and can overwrite installed game files before the patch is built.
- [x] Route all modified bytes to `Project.WorkingDir/out/<game-relative-path>`; never write to the game installation during extraction, translation, injection, or patch creation.
- [x] Capture original-file hashes **before** any transformation and make `ModifiedFiles`/`ModifiedFileHashes` refer to the same staged targets consistently.
- [x] Add regression tests that hash original files before and after injection and prove they remain byte-for-byte identical; then build, apply, and uninstall an FFP1 patch and prove the original bytes are restored.

**Acceptance criteria:** every built-in injector follows the same staging-only contract; a failed injection cannot change game files or leave a previous staged result buildable.

### 2. Correct FromSoftware archive identity and write-back

- [ ] Preserve the actual archive/container identity for strings extracted from `Data0.bhd`/`Data0.bdt`. In `pkg/extract/fromsoftware_extractor.go`, archive-derived entries currently use a virtual message-archive path as `File` without enough container metadata for the injector to locate the real installed target.
- [ ] Ensure the injector can distinguish a loose `.fmg`, a loose `.dcx`/BND, and an FMG nested inside an archive. Implement a verified BHD5/BDT rebuild path only when the archive layout and offsets can be preserved safely; otherwise mark those entries read-only and explain why.
- [ ] Do not return after discovering loose `.fmg` files if supported archive content remains to be scanned. Merge unique entries from all supported sources instead.
- [ ] Add tests with mixed loose and archived fixtures, duplicate IDs, non-English catalogs, compressed/uncompressed variants, and round-trip validation against the format parsers.

**Acceptance criteria:** every editable entry maps to a real source target and exact archive member; unsupported cases are visibly read-only rather than failing late or targeting a phantom path.

### 3. Make Unreal archive extraction compatible with patch creation

- [ ] Define and implement the write-back contract for `.locres` files found inside `.pak` archives. `pkg/extract/engines.go` records the archive member as the entry's file path, while `UnrealInjector` may stage a loose file at that path; `installer.Build` then validates the original as `GameRoot/GamePath`. An archive-only source is not a loose file, so the end-to-end flow can fail or produce an overlay that the game never reads.
- [ ] For each Unreal content type, choose one explicitly tested strategy: rebuild the containing archive with compatible tooling, generate a supported loose override, or expose extraction-only/read-only capability. Do not silently package a loose replacement for an archive member.
- [ ] Preserve container, member path, archive identity, game version, and encryption/signing requirements in project entries/patch metadata.
- [ ] Add end-to-end fixtures for loose `.locres`, `.pak`-resident `.locres`, and UE5 IoStore. Verify the game-relative installation result and clean uninstall.

**Acceptance criteria:** the UI only offers patch creation when the selected content can actually be applied to the target game using the generated package.

### 4. Harden and test the FFP1 reader/apply/uninstall boundary

- [ ] Add a dedicated C# test harness for `patcher/FarsiForgePatcher.cs` and run it in CI. Current workflow builds the patcher but does not run a comparable malformed-patch/apply/uninstall regression suite.
- [ ] Validate attacker-controlled lengths and counts before allocating memory or looping: target count, record count, UTF-8 strings, blob boundaries, offsets, integer overflow, final sizes, decompressed sizes, and aggregate resource limits. Include malformed, truncated, duplicate, and oversized FFP1 v1/v2 files.
- [ ] Add property/fuzz tests for Go FFP1 writer ↔ C# reader compatibility, base hashes, rollback failures, interrupted apply/uninstall, and reparse-point races.
- [ ] Keep the existing safe-relative-path, base-file hash, rollback, and reparse-point checks; expand tests rather than weakening them.

**Acceptance criteria:** malformed patches fail closed with actionable errors, before touching target files; any failed apply/uninstall either rolls back cleanly or preserves all recoverable backups and reports the paths.

### 5. Make imports validated and atomic

- [x] Validate imported CSV/XLSX status values against `untranslated`, `translated`, `approved`, and `skipped`. `pkg/exchange/exchange.go` currently converts a supplied status into `core.Status` and passes it to `SetTranslation`, which does not enforce the allowed set.
- [x] Validate all rows first (unknown/duplicate IDs, invalid status, malformed rows, conflicting translations) before mutating the project. If validation fails, apply **zero** imported changes and return a report of rejected rows.
- [x] Avoid trimming meaningful leading/trailing whitespace from translations silently; define a format-specific policy and preserve intentional whitespace.
- [x] Add tests for invalid statuses, duplicated and stale IDs, BOMs, embedded commas/newlines, blank translations, and partial-import rollback.

**Acceptance criteria:** an invalid import never corrupts project state and always reports exactly what was accepted, skipped, or rejected.

### 6. Make releases reproducible, least-privilege, and verifiable

- [x] Align `.github/workflows/release.yml` with the Go version declared by `go.mod` (`go 1.26.0`); the current workflow's `>=1.22` lower bound is inconsistent with the module requirement.
- [x] Add a CI job on pull requests and pushes for `go test ./... -count=1`, `go vet ./...`, frontend dependency install/build/lint/type-check, companion `FarsiForgeTools` tests, and patcher tests. Build/release success must depend on these gates.
- [x] Separate build/test permissions from publishing permissions; grant `contents: write` only to the release job.
- [x] Pin the companion repository to a tagged version or commit SHA instead of cloning its moving default branch during release builds.
- [x] Verify SHA-256 checksums and, where available, signatures for downloaded tools/assets (`repak`, `retoc`, GDRE, font artifacts, and other binaries); fail the release when verification fails.
- [ ] Generate a release manifest/SBOM with tool versions, source revisions, hashes, licenses, and the exact Go/Node/runtime versions used. Add dependency and secret scanning.
- [ ] Run a clean-machine smoke test on the produced ZIP: launch the desktop app, inspect tool discovery, extract a fixture, stage a patch, apply it, and uninstall it.

**Acceptance criteria:** the same tagged source and pinned dependencies produce traceable artifacts; a failed test, missing dependency, checksum mismatch, or broken packaged path prevents publication.

---

## P1 — Workbench reliability and truthful capabilities

### 7. Use one capability matrix across backend, frontend, CLI, and docs

- [ ] Expose engine capabilities from the registered detector/extractor/injector implementations instead of repeating engine lists in `frontend/src/app/page.tsx`.
- [ ] Represent detection, extraction, translation, validation, injection, archive rebuild, font support, and patch-install support separately. Include conditional requirements and precise reasons (e.g. Unity raw-byte candidates without a verified typetree, Unreal `.uexp` strings, Godot's existing Persian `.translation` constraints).
- [ ] Display the actual capability state and reason before long operations; never rely only on a generic success/failure alert.
- [x] Re-audit `docs/support-matrix-audit.md`: it conflicts with the current registries/UI in places, including custom-to-generic injector mapping and current injection registrations. Update it from current code and add a test that catches documentation/capability drift.
- [ ] Make discovery failures and truncated scans visible. `pkg/scanner/scanner.go` currently suppresses walk/callback errors, while some engine paths have explicit result limits; show skipped files, inaccessible directories, missing keys/tools, and partial extraction counts.

**Acceptance criteria:** the displayed support status is derived from the same capability source the pipeline uses, and each unsupported/partial result explains what can and cannot be done.

### 8. Finish the translation editor as a real workbench

- [x] Replace the `MAX_VISIBLE = 200` slice in `frontend/src/app/page.tsx` with pagination or virtualized rendering, so translators can reach every item in large projects without relying on search to reveal hidden rows.
- [x] Add explicit row status controls (translated/approved/skipped), select/bulk actions, next/previous issue navigation, and visible QA summaries.
- [x] Autosave drafts or provide explicit Save/Save All before leaving the translation step. Handle app shutdown, operation failure, and project switching without silently discarding edits.
- [x] Add Back navigation and correctly reset all project-specific state when a new game is selected. Prevent overlapping operations and stale results from earlier runs.
- [ ] Replace `alert(String(error))` and console-only diagnostics with readable in-app errors, recovery suggestions, operation logs, and copyable diagnostic reports.
- [ ] Replace the hard-coded 50%/100% progress indicator with real progress events, stage names, cancellation where supported, and a clear partial-failure state.

**Acceptance criteria:** projects with 100,000+ strings remain usable; edits are not silently lost; long operations can be understood and safely retried.

### 9. Connect settings and Persian options to real behavior

- [ ] Wire the existing `GetSettings`/`SaveSettings` APIs into the UI, including tools/output directories, game search paths, default language/author, and persisted Persian settings.
- [x] The RTL, shaping, Persian digits, and Vazirmatn controls in the current page are local React state only. Persist them and pass the selected values to the project/injection pipeline; a visual toggle must not pretend to change output when it does not.
- [ ] Wire `Project.FontPath` and the optional `BuildConfig.FontFile` into patch building if a font is supported. Otherwise label the font option as unavailable until font packaging works.
- [x] Validate custom paths before saving settings and report a failed tool-registry refresh instead of persisting a path while retaining the old registry.
- [x] Add atomic config writes (temporary file, flush, replace) in `pkg/core/config.go`, with invalid/corrupt configuration recovery and migration tests.

**Acceptance criteria:** a restart preserves settings, each displayed toggle demonstrably affects generated output, and invalid settings cannot leave the UI and backend using different values.

### 10. Make CSV/XLSX exchange available in the product

- [x] Expose the existing `pkg/exchange` CSV/XLSX import/export operations through the Wails `App` API and add corresponding UI actions.
- [x] Include a schema/version marker and stable entry IDs; report stale IDs and ambiguous matches instead of guessing based on source text.
- [x] Keep import/export non-destructive and round-trip test Persian text, RTL content, newlines, placeholders, status, context, and very large projects.

**Acceptance criteria:** translators can safely collaborate in Excel/CSV and re-import work without losing entry identity, QA state, or formatting.

### 11. Consolidate translation validation and QA

- [x] Add tests to `pkg/validate` and fix `fmtMatches`: `string(rune(c+'0'))` does not format occurrence counts correctly at 10 or above. Make diagnostic ordering deterministic.
- [x] Reconcile `pkg/validate.StandardValidator` and `pkg/textfilter.QA` so placeholder/tag/newline/brace/markup checks have one clearly documented source of truth and are consistently invoked on editor changes, imports, approvals, and before injection.
- [x] Expand format-aware placeholder parsing for printf variants, named/numeric format arguments, escaped braces, Rich Text/BBCode, control codes, plural/select tokens, and engine-specific markers.
- [x] Make validation severity actionable: block injection on hard errors, allow warnings with explicit acknowledgement, and show exact source/translation differences.
- [x] Add tests for mixed Persian/Latin text, emoji, combining marks, repeated placeholders, whitespace, line breaks, markup nesting, maximum length, and RTL text surrounding variables.

**Acceptance criteria:** dangerous placeholder loss is caught before patch creation, false positives are explainable, and validation results are stable across runs.

### 12. Improve tool setup, diagnostics, and external process handling

- [ ] Build a per-operation requirements check instead of checking only a single `Capabilities.ToolName`. Account for Unreal `repak`/`retoc`, UnityPy, FFTools, GDRE, RAGE key-material requirements, and FromSoftware's Oodle dependency as applicable.
- [ ] Show detected path, version, architecture, license/source, and a test/run result for each required tool. Add refresh and a downloadable/copyable diagnostic report.
- [ ] Replace arbitrary `pip install <name>` behavior with a documented, pinned requirements set in an isolated environment; avoid silently installing arbitrary packages from UI input.
- [ ] Bound captured subprocess output, consistently report exit code/stdout/stderr, preserve cancellation, and use explicit timeouts for tool discovery and operations that may hang.
- [ ] Add fixture-based tests for missing, corrupt, incompatible, wrong-architecture, and non-executable helper tools.

**Acceptance criteria:** prerequisites are detected before work begins and failures identify the missing/incompatible dependency without forcing users to reverse-engineer logs.

### 13. Strengthen filesystem safety and recovery

- [ ] Audit every read/write path—not just the patcher—against symlinks/reparse points and canonical root boundaries, especially backup/restore, generic text injection, FromSoftware staging, launch-game paths, and temporary directories.
- [ ] Add tests for traversal, Windows drive/UNC paths, junctions, symbolic links, case-insensitive duplicates, path races, interrupted writes, and a target replaced while a process is running.
- [ ] Make package creation transactional: write the patch, patcher, font, manifest, and README to a temporary staging directory; publish only after every step succeeds; clean up or clearly quarantine failed partial output.
- [ ] Require the patcher executable for a user-installable package, or explicitly emit a documented portable patch-only artifact. Never generate installation instructions for a file that was not packaged.
- [ ] Add nil/empty-input checks and failure-recovery tests to public APIs such as backup rollback.

**Acceptance criteria:** failed or interrupted operations do not corrupt the source game/project and never leave a misleading “successful” partial installer.

---

## P1/P2 — Source-aware asset browser, precise text extraction, and game-specific fonts

These features should be implemented as one source-aware workflow: inspect game files and their parsed contents first, review reliable text candidates, then edit only assets whose structure and write-back strategy are known. Do not turn a raw byte scan into an authoritative translation list, and do not assume every game can use the same font replacement method.

### A — P1: Engine-aware compiled-asset browser and source provenance

- [ ] Replace the translation-first experience with an **asset browser** that starts from the selected game root and presents a hierarchical tree where supported: folders → archives/containers → files → parsed resources/objects → text-bearing fields/records. Preserve original game-relative paths and container/member identity.
- [ ] Add engine-specific browse providers rather than pretending that every binary format can be opened generically. Start with formats that already have parsers: Unity assets/objects (UnityPy and verified type-tree/property data), Unreal `.pak`/IoStore entries and `.locres`/structured assets, Godot PCK/resources/catalogs, FromSoftware FMG/BND/DCX where the parser provides member identity, and generic structured text files. Unknown or opaque formats must say **not parsed / unsupported** instead of displaying invented structure.
- [ ] Selecting a file or resource should show metadata and a content preview: engine, resource/object type, object/key/field ID, container/member path, source offset when known, text count, extraction method, confidence, and whether the candidate can be edited and injected. Show the exact reason when a candidate is read-only (for example, unverified Unity raw bytes or heuristic-only Unreal `.uexp` strings).
- [ ] Group content by its **real source resource**; do not flatten every string from every compiled file into one undifferentiated dialogue list. Provide distinct views/categories for dialogue/subtitles, menus/UI, items, quests/journal, tutorials/help, system/technical text, and unclassified candidates. A sentence found in a binary blob is not automatically dialogue.
- [ ] Add search, filtering by engine/file/resource/category/confidence/editability, file-level string counts, and a details panel. Virtualize or paginate long file/resource/entry lists; no candidate may be silently inaccessible because only the first 200 rows render.
- [ ] Maintain provenance from every displayed string back to a stable source identity. Extend the project schema/versioned metadata as needed without changing IDs in a way that discards existing translations; preserve container, object/field/key, context, source text, and write-back capability across project reload and re-extraction.
- [ ] Keep a separate advanced **raw candidates** view for printable strings recovered without structural parsing. Make such records reviewable and exportable, but visibly lower-confidence/read-only until a format-specific parser establishes a reliable field identity. Do not make them writable merely because they look like natural language.
- [ ] Update the Wails API and frontend to browse project files/resources without forcing a full translation extraction first; support lazy parsing of the selected resource and cancellation/progress for expensive archive scans.
- [ ] Add fixture-based tests for tree structure, container/member identity, stable IDs, unsupported formats, path safety, large resources, and preservation of prior translations. Include nested archives and similarly named assets so paths cannot collide.

**Acceptance scenario — compiled game assets:** open a Unity game containing text assets and raw MonoBehaviour byte candidates, an Unreal game containing both `.locres` and heuristic `.uexp` strings, and a Godot PCK. The browser shows the actual files/resources and available metadata. Verified structured text is grouped under its owning resource; raw/heuristic strings appear separately with confidence and read-only reasons. Only actual dialogue records are labelled dialogue; menu, item, technical, and unknown text are not mislabelled. Selecting any item reveals exactly where it came from and whether it is safe to write back.

**Acceptance criteria:** every editable entry has a stable, traceable source location and a supported write-back route. Unknown formats are clearly marked as unknown, and no text is labelled dialogue solely because it contains sentence-like words.

### B — P1: Precision-first text candidate classification and filtering

- [ ] Create a shared extraction/classification pipeline used consistently by all extractors, instead of allowing each parser to use unrelated thresholds such as “contains letters” or a raw printable-byte scan as its final acceptance rule.
- [ ] Combine structural evidence (engine resource/field type, localization table, dialogue node, UI/widget, item/quest record), source context/path, placeholder/tag parsing, natural-language heuristics, duplicate signals, and confidence. The shared `textfilter.IsTranslatable` check is useful but is not, by itself, proof that a string is intended for translation or dialogue.
- [ ] Add explicit candidate classes such as `high-confidence translatable`, `likely technical/identifier`, `raw/heuristic`, and `needs review`. Use **high-confidence candidates** as the default translation view, with a visible count of filtered/reviewable candidates and an advanced “show all” control; never silently discard unknown candidates.
- [ ] Filter obvious binary/engine noise, hashes, paths, type names, debug/log strings, property identifiers, empty/markup-only records, and non-language payloads when evidence supports that decision. Avoid over-filtering legitimate short UI labels, item names, options such as “Yes/No”, key prompts, placeholders mixed with text, Persian/Arabic source strings, or strings whose meaning depends on context.
- [ ] Treat semantic category and write-back capability as separate dimensions: a record can be readable but unclassified, categorized but read-only, or translatable but not injectable for that engine/version. Never infer editability from confidence alone.
- [ ] Deduplicate only records proven to represent the same stable source identity. Identical visible wording in different files, dialogue speakers, languages, keys, or contexts must remain distinct.
- [ ] Provide user controls to include/exclude categories, mark false positives, promote a candidate to translatable, or keep a useful short/non-dialogue string. Persist review decisions against stable source identity and carry them forward only when that identity still matches after re-extraction.
- [ ] Build a labelled golden fixture corpus containing real dialogue, UI labels, item/quest names, tutorial/help text, source-code identifiers, font/resource metadata, debug text, hashes, placeholders, markup, and binary noise across supported engines. Measure precision/recall (or an equivalent reviewed false-positive/false-negative report) and regressions per engine before changing heuristics.
- [ ] Report extraction statistics per file/resource: parsed records, accepted candidates, filtered candidates by reason, unclassified/raw records, failures, and unsupported data. Make any truncation/cap explicit and actionable.

**Acceptance scenario — a noisy extraction:** a compiled widget contains “Start Game”, an internal class name, a GUID, a font asset path, a debug message, and a dialogue line. The normal translation view shows likely translatable records in their actual categories; the raw/review view retains uncertain candidates with reasons. “Start Game” is not labelled dialogue without structural evidence, the GUID and class name are not shown as ordinary translation work, and the user can reveal filtered candidates instead of losing them.

**Acceptance criteria:** default results are materially cleaner without silently losing short but meaningful UI strings. Per-engine precision/recall is tracked against reviewed fixtures, and raw heuristic candidates never become writable unless a verified parser supports their source identity.

### C — P1: Game-font inventory and diagnostic report; P2: safe, game-specific replacement

- [ ] Add a **Font Inventory / Font Scan** operation separate from the current Tools-directory discovery. Scan the selected game installation and supported archives/resources for font assets and their references, not just files whose names end in `.ttf`; classify direct font files, engine font resources, bitmap/glyph atlases, and font metadata as different asset types.
- [ ] Start with engine adapters where parsers or metadata are available: loose `.ttf/.otf/.ttc` files, Unity Font/TextMeshPro resources and atlases, Unreal Font/FontFace resources and associated atlases, Godot font resources/`FontFile` inside supported PCK/resources, and documented read-only detection for formats such as custom FromSoftware glyph atlases until a safe writer exists. Do not claim universal discovery for opaque/encrypted/unsupported containers.
- [ ] Produce a browsable/exportable inventory report with asset path, containing archive/PCK, resource/object ID, engine/version, detected format, font family/style where recoverable, atlas/texture and material dependencies where known, reference/usage evidence, detection confidence, glyph-coverage results, and whether export/replacement is supported. Distinguish **not found**, **not parsed**, and **no font detected**.
- [ ] Allow the user to inspect/export a copy of a font resource to the project work area and choose a custom replacement manually. Keep original assets untouched; record source hash, target hash, resource identity, and replacement decision in the project/patch manifest.
- [ ] Add a font preview/diagnostic test string containing Persian and Arabic letters, presentation forms as applicable, Persian/Latin digits, punctuation, mixed RTL/LTR text, and representative glyphs from the game. Report missing glyphs and, when possible, family/style/metrics differences before replacement. Font glyph coverage does not guarantee that the game supports RTL directionality or Arabic shaping; report these as separate capabilities.
- [ ] Replace the single mandatory Vazirmatn assumption with an optional suggested font, not an unconditional replacement. Respect the target game's existing font system, encoding, glyph atlas, layout, line height, fallback chain, and licensing constraints. Importing a TTF/OTF must not be treated as equivalent to rebuilding a bitmap glyph atlas or an engine-serialized font object.
- [ ] Implement replacement only through an explicit engine-specific font adapter with a declared `FontDiscovery`, `FontExport`, and `FontInjection` capability (or equivalent); unsupported fonts/resources stay analysis-only. Where the engine requires atlas repacking or character-map/metric changes, implement and validate those transformations before enabling the UI action.
- [ ] Connect `Project.FontPath`, font-selection settings, `InjectorCaps.FontInjection`, and `BuildConfig.FontFile` end-to-end, or hide/disable unsupported controls with an explanation. The current Vazirmatn UI toggle is local frontend state; tool discovery looks for Vazirmatn in the Tools folder and does not constitute scanning the selected game's fonts. Do not show “font installed/replaced” unless the generated patch actually contains and applies the selected asset.
- [ ] Stage every replacement under the project work directory and include it in the final patch with all required companion assets. Extend uninstall/apply tests so font and atlas changes are restored along with text assets byte-for-byte. Never overwrite installed game files during scan, preview, export, or injection staging.
- [ ] Add fixture-based tests for font metadata parsing, nested archive/PCK discovery, missing glyph detection, unsupported/proprietary font formats, corrupt font files, custom font selection, duplicate resource names, target hash checks, patch packaging, and uninstall restoration. Document redistribution/license checks for any font bundled by default.

**Acceptance scenario — two games, two font systems:** Game A stores a standard font resource inside a Unity asset bundle; Game B uses a custom bitmap glyph atlas. FarsiForge reports each asset, its container and resource identity, what is actually known about the font and glyph coverage, and whether an adapter can safely replace it. The translator chooses a licensed custom font for Game A; a staged patch is built only if the Unity replacement adapter and validation pass. Game B remains read-only until an atlas adapter can preserve glyph metrics/mapping. The original installed files remain unchanged and uninstall restores them exactly.

**Acceptance criteria:** font scanning returns an auditable inventory rather than assuming Vazirmatn is the right font. A replacement is offered only when the engine/resource adapter can safely perform and verify it; otherwise the report provides useful information without risking the game installation.

---

## P2 — Engine depth, scale, and translator productivity

### 14. Expand engine write-back only with end-to-end fixtures

- [ ] **Unity typetree recovery:** recover/validate field identity for stripped builds, retain the raw-scan safety gate, and only enable writes after a candidate passes structural verification.
- [ ] **Unreal archives:** support a fully tested `.pak` rebuild or supported override workflow for eligible `.locres` files; add explicit version/encryption/IoStore limits.
- [ ] **Godot locale creation:** create/register Persian locale resources where format/version permits; keep encrypted/embedded or unsupported PCKs read-only with a clear explanation.
- [ ] **FromSoftware fonts:** build glyph/atlas editing only after text archives have safe staged write-back and restoration tests. Preserve atlas layout, glyph metrics, and fallback fonts.
- [ ] Evaluate extractors for RPG Maker, Ren'Py, GameMaker, and additional engines based on user demand and the available safe write-back path. Avoid advertising support based on detection/extraction alone.

**Acceptance criteria:** every new supported engine or sub-format has sample fixtures, round-trip parser tests, real-game validation evidence, safe staging, installer/uninstaller coverage, and a documented support level.

### 15. Improve FarsiForgeTools text fidelity and format parsing

- [ ] Replace the generic line-based parser in `FarsiForgeTools/internal/text/text.go` with format-aware handlers: normal CSV files are not reliably handled by the current `key=value` line parser, and XML should not be treated as arbitrary lines.
- [ ] Handle JSON arrays of objects/nested arrays and report unsupported structures instead of silently omitting strings.
- [ ] Preserve UTF-8 BOM, line endings, comments, encoding, ordering, and unrelated fields whenever possible; write outputs atomically.
- [ ] Check all CSV writer flush/close errors and add tests for disk-full/short-write failures, BOMs, escaped quotes, commas/newlines, and large records.
- [ ] Add corpus-based and fuzz tests for `.locres`, Godot PCK/`.translation`, FMG, BND/DCX, RPF7/GXT2, JSON/CSV/XML, and the FFP1 parser/writer; include malformed/truncated files and resource limits.

**Acceptance criteria:** formats preserve non-translated content byte-for-byte where promised, malformed files fail safely, and release confidence comes from regression fixtures rather than manual success reports alone.

### 16. Support large games and large translation projects

- [ ] Measure extraction and injection time, peak memory, allocations, and output size on representative large games. Publish baseline benchmarks.
- [ ] Stream large patch payloads through the FFP1 writer/installer instead of repeatedly holding complete patched files and the compressed blob in memory where practical.
- [ ] Add cancellation and visible progress to long scans, archive extraction, and patch generation; make any hard cap configurable or show exactly what was skipped.
- [ ] Add incremental rescanning and stable identity-based merge reports (new, changed, removed, unchanged, and preserved translations).
- [ ] Add backup/project cleanup tools with explicit confirmation and disk-space estimates; never automatically delete original backups or translator work.

**Acceptance criteria:** performance improvements are backed by repeatable benchmarks and do not weaken integrity verification or translation preservation.

### 17. Improve translator productivity

- [ ] Add project-level glossary/terminology rules, protected terms, character/placeholder rules, and optional reusable translation memory.
- [ ] Add optional, provider-agnostic AI translation suggestions with explicit user opt-in, configurable provider/model, batch limits, retry/rate-limit handling, and a preview before changes are applied. Never overwrite approved translations silently.
- [ ] Add a diff inspector for source vs translation and before/after extraction, with live RTL/Latin/placeholder preview.
- [ ] Add duplicate-string detection and suggested bulk translation only where context is compatible; preserve separate entries when meaning may differ.
- [ ] Add configurable filters for untranslated, QA failures, context, file/container, and maximum-length warnings.

**Acceptance criteria:** suggestions remain optional and reversible; glossary and translation memory never merge unrelated entries by source text alone.

---

## P3 — Distribution and community polish

### 18. Make project and patch formats maintainable over time

- [ ] Add explicit project schema versions and tested migrations; reject or back up unsupported future versions without overwriting user data.
- [ ] Add patch metadata for target game build/version, content-container strategy, required tools/runtime, and known limitations.
- [ ] Include user-facing checksums and a patch integrity verification command; consider signing releases and published patch metadata.
- [ ] Publish supported platform/runtime requirements and a clean-install guide with troubleshooting for missing tools, blocked executables, antivirus false positives, and write-protected games.

### 19. Complete organization-level engineering hygiene

- [ ] Keep the main application and `FarsiForgeTools` versions coordinated: pin helper-tool releases in the main release workflow and document a compatibility table.
- [ ] Add consistent PR templates/checklists for parser changes, safety boundaries, real-game evidence, fixtures, release notes, and license review.
- [ ] Add an automated stale-document check for support matrix, dependency versions, and packaging claims. Treat large upstream forks such as AssetRipper as separately versioned dependencies rather than folding their entire upstream codebase into the app's own support claims.
- [ ] Maintain third-party notices for every bundled executable, font, parser, and downloaded asset, including redistribution rights and the exact upstream version.

---

## Recommended implementation order

1. **First:** FromSoftware staging/data preservation; FromSoftware archive identity; Unreal archive-to-installer semantics.
2. **Second:** FFP1 malformed-input and rollback tests; import atomicity/status validation; transactional package output.
3. **Third:** reproducible CI, pinned tool downloads, Go version alignment, checksums, and least-privilege release permissions.
4. **Fourth:** shared extraction/classification baseline and labelled false-positive/false-negative fixtures; engine-aware asset browser with source provenance.
5. **Fifth:** game-font inventory/report and custom-font preview/coverage validation (read-only first); add replacement adapters only when their staging, package, and uninstall path is verified.
6. **Sixth:** capability-driven UI, persisted settings/options, workbench autosave/pagination, and CSV/XLSX integration.
7. **Seventh:** unified QA, helper text parser fidelity, diagnostics, fuzz tests, and large-game benchmarks.
8. **Then:** engine expansions, glossary/translation memory, optional AI suggestions, visual diff, and auto-updater.

Do not mark a task complete solely because it compiles. Mark it complete after its acceptance criteria pass in automated tests and, for game-format work, the documented fixture or real-game workflow.

---

## Completed milestone archive

The items below were recorded as completed in the previous roadmap. This review did not rerun those tests, and the archive should remain historical evidence rather than an assumption that every current code path is fully covered.

<details>
<summary><b>Previously reported P0–P3 milestones</b></summary>

### P0 — Security, reparse-point audits, persistence
- [x] FFP1 v2 copy-from-base checks and reparse-point escape protections.
- [x] Go/C# regression coverage for game-root boundaries.
- [x] `SaveTranslations` ID/status validation before persisted mutation.
- [x] Structured identity for re-extraction translation matching.
- [x] Windows path normalization and interrupted-save verification.

### P1 — Format extraction and injection
- [x] Real-game extraction validation recorded across Unity, Unreal, Godot, FromSoftware, RAGE, SAGE, Source 2, Project Zomboid, Factorio, and generic text.
- [x] FMG/BND/DCX, RAGE RPF7/GXT2, Unreal `.locres`/archive, and Godot PCK processing milestones recorded.
- [x] Unity raw-byte candidates are flagged and unsafe unverified writes are rejected.

### P2 — Companion repository alignment
- [x] Godot 4 binary `OptimizedTranslation` read/write in `FarsiForgeTools`.
- [x] `fftools translation export/import` with source-key hash resolution.
- [x] Standalone unencrypted Godot PCK v2/v3/v4 rebuild.
- [x] Synthetic install/apply/uninstall restoration scenario recorded.

### P3 — Registry and support status
- [x] Custom-engine mapping to generic extraction/injection.
- [x] Explicit detection-only and extract-only errors for known unsupported engines.
- [x] Frontend status badges/disabled actions for known unsupported states.
- [x] Synthetic coverage for dialogue, subtitles, UI, notes, items, tutorials, and quests.

</details>
