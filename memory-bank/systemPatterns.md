# System Patterns: FarsiForge

Updated: 2026-10-13

## Core Architectural Layers
- **Desktop Host (`app.go` & Wails v2):** Coordinates frontend events, project persistence, file dialogues, and lifecycle orchestration.
- **Headless CLI (`cmd/farsiforge-cli/`):** Headless interface for automated detection, extraction, injection, and search.
- **Engine Detectors (`pkg/detection/`):** Priority-ranked heuristic detectors mapping folder signatures and file headers to engine families.
- **Extractors (`pkg/extract/`):** Format-specific decoders (BHD5/BDT, DCX, BND4, FMG, GXT2, RPF7, locres, PCK, Valve KV, SAGE CSF).
- **Injectors & Staging (`pkg/inject/`):** Safe format serializers and staged workspace writers.
- **Persian Text Engine (`pkg/persian/`):** Bidirectional reordering, Arabic/Persian letter shaping, ZWNJ handling, and tag protection.
- **Installer & Package (`pkg/ffpatch/`, `pkg/installer/`, `patcher/`):** FFP1 binary package format writer and zero-dependency C# GUI installer with rollback protection.
- **Tool Registry (`pkg/tools/`):** Dynamic discovery of engine helper binaries (`Tools/fftools/`, `Tools/repak/`, `Tools/retoc/`, `Tools/patcher/`).

## Architectural Invariants
1. **Separation of Capabilities:** Detection, extraction, injection, and patching are distinct capabilities. Detection never implies write-back support.
2. **Read-Only Originals:** Real game folders are strictly read-only. Staged outputs and projects are stored externally (in `%LOCALAPPDATA%\FarsiForge` or custom workspace).
3. **Identity Preservation:** Translation entries are uniquely keyed by a composite tuple (`container`, `file`, `key`, `context`, `source`). Re-extraction carries over existing translations only when source identity matches.
4. **Security & Reparse-Point Guards:** Patcher preflight rejects copy-from-base directory symlinks and path traversals escaping game roots.
5. **Atomic Operations:** Project saves write to temporary files first and replace atomically; patch installation backs up originals and rolls back on failure.
6. **No Silent Fallbacks:** Unsupported engines or ambiguous raw byte scans fail explicitly with informative error messages.
