# Product Context: FarsiForge

Updated: 2026-10-13

## Problem Statement
Game localization into Persian has historically relied on ad-hoc scripts, manual hex editing, destructive file overwrites, and unverified string replacements that break game executables, crash fonts, or render text in reverse (LToR or disconnected letters). Translators lack a unified workbench that guarantees file integrity, preserves variables, and produces reliable end-user installers.

## Intended Localization Workflow
1. **Detect Engine:** User selects game folder; FarsiForge detects the engine and reports supported capabilities (Full Pipeline, Extract-Only, or Detection-Only).
2. **Extract Text:** Unpacks archives into a project file (`.ffproj`) stored outside the game directory. Strings retain stable IDs, file paths, contexts, and line numbers.
3. **Translate & QA:** Translators edit strings using a responsive grid with status workflows (Untranslated, In Progress, QA, Approved) and QA validation warnings.
4. **Stage Changes:** The format-specific injector shapes Persian text (RTL + ZWNJ) and stages format-correct files in the workspace. The original game remains untouched.
5. **Build & Apply Patch:** Generates an FFP1 installer (`FarsiForgePatcher.exe`). The patcher verifies original game files, creates backups, applies translations, and supports 1-click atomic uninstall.

## Evidence & Honesty Principles
- **No False Claims:** Engine detection does not imply write-back support. Capabilities are tracked and labeled separately per format.
- **Roundtrip Verification:** Every supported format must prove roundtrip fidelity (original → extract → unchanged write-back → byte/functional equality).
- **Read-Only Game Files:** Game installations are treated strictly as read-only.
- **Explicit Failure Modes:** Unsupported engines and ambiguous raw candidates (e.g. Unity `raw_*`) are rejected with explicit user feedback rather than corrupted in-place.
