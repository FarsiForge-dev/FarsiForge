# Technical Context: FarsiForge

Updated: 2026-10-13

## Technology Stack
- **Backend:** Go 1.22+ (`main.go`, `app.go`, `pkg/*`)
- **Desktop Shell:** Wails v2 (Go + Webview2 bridge)
- **Frontend:** Next.js 14, React 18, TypeScript, TailwindCSS, Framer Motion, Radix UI primitives
- **Installer:** C# .NET Framework 4.5+ (`patcher/FarsiForgePatcher.cs`), compiled via Windows built-in `csc.exe` (zero SDK dependencies)
- **Companion CLI:** `FarsiForgeTools` (`fftools.exe`), pure Go binary for `.locres` and Godot `.translation`/`.pck` operations
- **Knowledge Graph:** Graphify (128 semantic communities, persistent graph at `graphify-out/`)

## Organization Helper Tools Ecosystem (`FarsiForge-dev`)
| Tool | Location | Role in Pipeline |
|------|----------|------------------|
| `fftools.exe` | `Tools/fftools/` | Unreal locres v0–v3 roundtrip, Godot `OptimizedTranslation` CSV/hash import/export, and PCK v2–v4 rebuild. |
| `FarsiForgePatcher.exe` | `Tools/patcher/` | Standalone C# Windows Forms installer (FFP1 v1 & v2). |
| `repak.exe` | `Tools/repak/` | High-performance Unreal Engine `.pak` archive extractor. |
| `retoc.exe` | `Tools/retoc/` | Unreal Engine 5 IoStore container (`.utoc` / `.ucas`) unpacker. |
| `gdre_tools.exe` | `Tools/gdre/` | Godot PCK v2–v4 reverse-engineering and text asset recovery (v2.7.4). |
| `magic.dat` | `Tools/CodeWalker/` | Modern GTA V RPF7 AES key derivation table. |
| `vazirmatn.ttf` | `Tools/vazirmatn/` | High-legibility Persian typeface bundled for game UI injection. |

## Verification & Build Commands
```powershell
# 1. Run full backend unit and regression tests
go test ./... -count=1

# 2. Run static analysis
go vet ./...

# 3. Build desktop application
wails build -platform windows/amd64 -s

# 4. Build headless CLI
go build -o build/bin/farsiforge-cli.exe ./cmd/farsiforge-cli

# 5. Build C# patchers (GUI & CLI)
pwsh -File patcher/build.ps1 -NoCopy

# 6. Verify frontend
cd frontend; npm run lint; npx tsc --noEmit; npm run build; cd ..

# 7. Update repository knowledge graph
graphify update .
```
