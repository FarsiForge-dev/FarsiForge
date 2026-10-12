<div align="center">
  
# 🇮🇷 FarsiForge (فارسی‌فورج)

**A Persian game localization workbench with format-specific extraction and reversible patch building**

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue.svg?style=flat-square)](https://golang.org)
[![React](https://img.shields.io/badge/React-18-blue.svg?style=flat-square)](https://reactjs.org/)
[![Next.js](https://img.shields.io/badge/Next.js-14-black.svg?style=flat-square)](https://nextjs.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-red.svg?style=flat-square)](https://wails.io)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)

![FarsiForge Banner](https://via.placeholder.com/800x200.png?text=FarsiForge+Persian+Game+Engine)

*FarsiForge helps locate game text, manage Persian translations, stage supported file changes, and build reversible patches. Engine detection, extraction, injection, and patching are distinct capabilities tailored per file format and game engine.*

</div>

## ✨ Features

- **🚀 Engine detection:** Robust heuristic detection across 13 engine families with priority sorting and confidence scoring.
- **📝 Format-specific text extraction:** High-fidelity native parsers for FromSoftware (`.fmg`/`.dcx`/`.bnd4`), RAGE (`.rpf`/`.gxt2`), Unreal Engine (`.locres`, `.pak`, IoStore `.utoc`/`.ucas`), Godot (`.pck`, `.translation`, `.tres`/`.tscn`), Unity (Mono/IL2CPP), Valve KeyValues, SAGE (`.big`/`.csf`), Factorio (`.cfg`), and Lua tables.
- **💉 Safe translation injection & write-back:** Dedicated write-back pipelines for FromSoftware (pure Go FMG encoder, DCX compressor, BND4 writer), Godot (`OptimizedTranslation` update & PCK rebuild), Unreal (`.locres` v0–v3 roundtrip), Unity (typetree-verified TextAssets), and generic structured text files.
- **🔄 Persian text shaping & typography:** Integrated Right-to-Left (RTL) reshaping, bidirectional Unicode processing, zero-width non-joiner (ZWNJ / نیم‌فاصله) handling, and Arabic-Indic / Persian digit conversion.
- **🎨 Interactive desktop UI:** Modern desktop workbench built with Wails, React, and TailwindCSS featuring real-time search, status filters, QA notes, and engine capability indicators.
- **📦 Reversible patch installer:** Standalone high-performance C# GUI patcher (`FarsiForgePatcher.exe`, FFP1 format) with preflight reparse-point protection, automatic original file backups, and byte-for-byte uninstall restore.

---

## 🎮 Engine Capability & Support Matrix

In game localization, **Detection**, **Extraction**, and **Injection (Write-Back)** are distinct stages. FarsiForge explicitly distinguishes these capabilities to ensure data integrity:

| Engine / Family | Detection | Extraction | Injection / Staging | Patching (FFP1) | Verification & Evidence |
|-----------------|-----------|------------|---------------------|-----------------|-------------------------|
| **FromSoftware** (Souls / Elden Ring) | ✅ | ✅ BHD5/BDT, DCX, BND3/BND4, FMG (v1/v2 wide/unicode) | ✅ Pure Go FMG serializer, DCX DFLT compressor, BND4 container writer | ✅ | Verified on *ELDEN RING* Data0 (520,750 entries across 24 message bundles; extraction & roundtrip write-back verified). |
| **Unreal Engine 4 / 5** | ✅ | ✅ `.locres` (v0–v3), `.pak` (repak), IoStore `.utoc`/`.ucas` (retoc), `.uexp` scanner | ⚠️ Loose `.locres` staging & write-back via `fftools` (compiled `.uexp` marked read-only) | ✅ | Verified on *Hacker Simulator* (11,676 entries) and *Fortnite* (85 `.pak` containers). |
| **Godot 3 / 4** | ✅ | ✅ Recovered `.tres/.gd/.tscn` text (gdre_tools v2.7.4), PCK v2–v4, `.translation` export | ⚠️ Updates CSV-resolved `.translation` resources in unencrypted PCK v2–v4 via `fftools` | ✅ | Verified on *MOLDRISE v1.0.5* (Godot 4.7, PCK v4, 3,789 strings) and synthetic PCK roundtrip tests. |
| **Unity** (Mono + IL2CPP) | ✅ | ✅ UnityPy TextAsset & recovered MonoBehaviour paths (raw byte scans flagged) | ⚠️ Requires verified writable typetree identity; raw-byte candidates rejected | ✅ | Verified on *ASKA* (324,899 entries), *Raft* (59,111 entries), and *Orc Massage* (13,148 entries). |
| **Generic / Custom** | ✅ | ✅ Structured text (`.json`, `.txt`, `.ini`, `.csv`, `.tsv`, `.xml`, `.yaml`) | ✅ Direct in-place staging for structured text | ✅ | Verified on *DDPER 8.4* and synthetic fixture matrices. |
| **RAGE** (GTA V) | ✅ | ✅ Encrypted RPF7 traversal (AES-256-ECB + Knuth PRNG tables via `magic.dat`), Deflate, GXT2 | ❌ Extract-only (re-encrypting RPF7 archives is not supported; loose-file modding recommended) | ❌ | Verified on *Grand Theft Auto V* build 1.0.3725.0 (8,669,936 strings across 43 archives). |
| **SAGE** (C&C Generals / ZH / RA3) | ✅ | ✅ `.big`, `.csf`, and `.str` archive & string table reader | ❌ Extract-only (write-back not implemented) | ❌ | Verified on *C&C Zero Hour* (1,532 strings) and *Red Alert 3* (10,956 strings). |
| **Source 2** (Deadlock / CS2) | ✅ | ✅ Valve KeyValues localization files | ❌ Extract-only | ❌ | Verified on *Deadlock* (53,522 entries). |
| **Factorio** | ✅ | ✅ `locale/*.cfg` localization files | ❌ Extract-only | ❌ | Verified on *Factorio* (7,019 entries). |
| **Project Zomboid** | ✅ | ✅ Lua translation table files | ❌ Extract-only | ❌ | Verified on *Project Zomboid* (14,058 entries). |
| **GoldSrc** (Half-Life / CS 1.6) | ✅ | ✅ Valve KeyValues localization files | ❌ Extract-only | ❌ | Verified format extraction path. |
| **UE3** (MK10 etc.) | ✅ | ❌ Generic text fallback only | ❌ Detection-only | ❌ | Heuristic detection only. |
| **Source 1** (HL2 / Portal) | ✅ | ❌ Generic text fallback only | ❌ Detection-only | ❌ | Heuristic detection only. |

> [!NOTE]
> Engines marked as **Extract-only** or **Detection-only** are clearly indicated in the UI with status badges and disabled buttons to prevent unexpected runtime errors.

---

## 📊 Real-Game Validation Matrix

The following real-game installations have been extracted and validated:

| Game | Engine | Extracted Entries / Scope | Pipeline Status |
|------|--------|---------------------------|-----------------|
| **Grand Theft Auto V** | RAGE | 8,669,936 strings across 43 `.rpf` archives | ✅ Extraction (Decrypted RPF7 + GXT2) |
| **ELDEN RING** | FromSoftware | 520,750 entries across 24 message bundles | ✅ Extract + Write-Back (FMG / DCX / BND4) |
| **ASKA** | Unity IL2CPP | 324,899 entries | ✅ Extraction |
| **Deadlock** | Source 2 | 53,522 entries | ✅ Extraction |
| **Raft** | Unity Mono | 59,111 entries | ✅ Extraction |
| **Orc Massage** | Unity Mono | 13,148 entries | ✅ Extraction |
| **Project Zomboid** | Java / Lua | 14,058 entries | ✅ Extraction |
| **Hacker Simulator** | Unreal Engine | 11,676 entries | ✅ Extraction + Staging |
| **Red Alert 3** | SAGE | 10,956 entries | ✅ Extraction |
| **Factorio** | Custom | 7,019 entries | ✅ Extraction |
| **MOLDRISE v1.0.5** | Godot 4.7 | 3,789 strings across 467 files | ✅ Extraction (PCK v4 + GDRE tools) |
| **C&C Zero Hour** | SAGE | 1,532 entries | ✅ Extraction |
| **Fortnite** | Unreal Engine 5 | 85 `.pak` containers | ✅ Detection + Container Indexing |

---

## 🏗️ Architecture

```
FarsiForge Workspace
├── cmd/
│   └── farsiforge-cli/     # Command-line interface for headless automation
├── frontend/               # Next.js 14 + TailwindCSS + Framer Motion workbench UI
├── patcher/                # Standalone C# .NET GUI patcher (FFP1 format)
├── pkg/
│   ├── core/               # Domain models, project persistence, translation entries
│   ├── detection/          # Priority-ranked heuristic game engine detectors
│   ├── extract/            # Format extractors (BND4, DCX, FMG, GXT2, RPF7, locres, etc.)
│   ├── inject/             # Safe format encoders & staging injectors
│   ├── ffpatch/            # FFP1 binary package builder & unpacker
│   ├── persian/            # RTL text reshaping, ZWNJ, and digit normalization
│   ├── tools/              # Smart helper tool discovery (fftools, gdre, repak, retoc)
│   └── validate/           # Pre-injection validation and tag/placeholder checks
└── Tools/                  # Precompiled helper tool binaries (managed/discovered)
```

---

## 🛠️ Integrated Helper Tools

FarsiForge incorporates and orchestrates specialized external utilities via `pkg/tools`:

- **`fftools.exe`**: In-house companion utility for Unreal Engine `.locres` (v0–v3) encoding/decoding and Godot `.translation` (`OptimizedTranslation`) CSV export/import and standalone PCK rebuilding.
- **`gdre_tools.exe` (v2.7.4)**: Godot reverse-engineering toolkit (built with Godot 4.8.dev runtime) for PCK v2–v4 text recovery.
- **`repak.exe`**: High-performance Unreal Engine `.pak` archive extractor.
- **`retoc.exe`**: Unreal Engine 5 IoStore container extractor (`.utoc` / `.ucas`).
- **`CodeWalker` `magic.dat`**: Lookup table for GTA V RPF7 AES key derivation and table decryption.
- **`UnityPy`**: Python module for Unity typetree asset extraction and serialization.

---

## 🚀 Getting Started

### Prerequisites

- [Go 1.26+](https://golang.org/dl/)
- [Node.js 20.9+](https://nodejs.org/)
- [Python 3.10+](https://www.python.org/) with `pip` (optional, for Unity typetree extraction)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

### Installation & Build

1. **Clone the repository:**
   ```bash
   git clone https://github.com/FarsiForge-dev/FarsiForge.git
   cd FarsiForge
   ```

2. **Install Python dependencies (optional for Unity):**
   ```bash
   pip install UnityPy
   ```

3. **Build the Desktop Application (Wails):**
   ```bash
   wails build -s
   ```
   *The portable binary will be generated at `build/bin/FarsiForge.exe`. Ensure the `Tools/` folder is placed adjacent to the executable.*

4. **Build the CLI tool:**
   ```bash
   go build -o build/farsiforge-cli.exe ./cmd/farsiforge-cli
   ```

5. **Run the Test Suite:**
   ```bash
   go test ./... -count=1
   ```

---

## 💡 Recommended Workflow

1. **Detect**: Select the game root folder. FarsiForge identifies the engine, displays detection confidence, and indicates available capabilities (Full Pipeline vs. Extract-Only).
2. **Extract**: Unpack archives and extract strings into the project database (`.ffproj`).
3. **Translate**: Edit strings using the responsive grid with search, status filters (Untranslated, In Progress, QA, Approved), and QA notes.
4. **Stage Changes**: Run injection to shape Persian text (RTL, ZWNJ, digit normalization) and stage format-compliant modified files inside the project workspace (the installed game remains untouched).
5. **Build & Apply Patch**: Package staged files into an FFP1 translation installer. The end-user patcher validates the game installation, backs up original files, and applies the translated assets.

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](../../issues).

## 📄 License

This project is licensed under the [MIT License](LICENSE).

<div align="center">
  <i>Developed with ❤️ for the Persian Gaming Community</i>
</div>
