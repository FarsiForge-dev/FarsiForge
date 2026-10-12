# Project Brief: FarsiForge

Updated: 2026-10-13

## Overview
FarsiForge (فارسی‌فورج) is an open-source, evidence-driven game localization workbench engineered specifically for Persian (Farsi) game translation. It addresses the unique linguistic challenges of Right-to-Left (RTL) text, character shaping, zero-width non-joiners (ZWNJ / نیم‌فاصله), and bidirectional mixed content, while providing format-specific extractors and safe, reversible patch installers.

## Core Value Proposition
- **Automated Game Discovery:** Scans game installations, heuristic engine detection across 13 families with capability scoring.
- **High-Fidelity String Extraction:** Deep extraction of player-facing text (dialogue, menus, items, quests, subtitles) across proprietary archive formats without corrupting binary tags or markup.
- **Linguistic Precision:** Full Persian RTL shaping, BiDi reordering, Arabic-Indic digit normalization, and tag protection.
- **Safety by Design:** Original game files are strictly read-only; modified assets are staged in an isolated workspace with SHA-256 validation.
- **Reversible Patching (FFP1):** Zero-dependency C# GUI installer with original file verification, automatic backups, and 1-click byte-for-byte uninstall.

## Repository Ecosystem
- **Main Workbench:** `FarsiForge-dev/FarsiForge` (Go backend + Wails v2 + Next.js 14 frontend)
- **Format Tools:** `FarsiForge-dev/FarsiForgeTools` (Companion Go CLI `fftools` for Unreal `.locres` and Godot `.translation`/`.pck`)
- **Engine Helper Utilities:** Curated forks under `FarsiForge-dev` organization (`repak`, `retoc`, `gdsdecomp`, `CodeWalker`, `UnityPy`, `vazirmatn`).

## Scope & Non-Goals
- **In Scope:** Text extraction, translation management, bidirectional text processing, format-safe staging, reversible patch creation.
- **Out of Scope:** Direct executable memory hooking / runtime translation injection (e.g. XUnity AutoTranslator), invasive DRM bypasses, or modifying multiplayer anti-cheat protected binaries.
