# FarsiForge Support-Matrix Audit — Detection vs Extraction vs Injection

- **Date:** 2026-10-12
- **Repo:** `D:\FarsiForge-dev\FarsiForge`
- **Type:** Read-only audit. No code was modified; this document is the only file created.
- **Method:** Direct inspection of `pkg/detection`, `pkg/extract`, `pkg/inject`, `app.go`, root `README.md`, plus a keyword sweep (`matrix`, `support`, `extraction`, `write-back`, `inject`, `engine`, `custom`, …) across every text file in `frontend/src`. Every reference below is an exact, verified `file:line` position in the current tree.

---

## 1. What each layer actually does

### 1.1 Detection — `pkg/detection` (13 possible engine strings)

- Registry runs all detectors, keeps the highest-confidence match and short-circuits at confidence ≥ 0.95 — `pkg/detection/registry.go:38-80` (best-match `:58-61`, short-circuit `:64-66`, error when nothing matched `:70-72`); detectors are priority-sorted on insert — `pkg/detection/registry.go:28-34`.
- Detector → engine string map (all verified):

| Detection name | Detector | Resulting `Engine` |
|---|---|---|
| `unity` | `pkg/detection/engines.go:17-84` (priority 100 `:20`) | `"unity"` `engines.go:24` |
| `unreal` | `pkg/detection/engines.go:143-216` (priority 90 `:146`) | `"unreal"` `engines.go:171` |
| `ue3` | `pkg/detection/engines.go:263-338` (priority 95 `:266`) | `"ue3"` `engines.go:296` |
| `godot` | `pkg/detection/engines.go:342-401` (priority 90 `:345`) | `"godot"` `engines.go:360` |
| `generic` (fallback detector) | `pkg/detection/engines.go:484-503` (priority 0 `:487`) | **`"custom"`** (confidence 0.1) `engines.go:492` |
| `sage` | `pkg/detection/engines_extra.go:83-164` (priority 95 `:86`) | `"sage"` `engines_extra.go:120` |
| `goldsrc` | `pkg/detection/engines_extra.go:168-240` | `"goldsrc"` `engines_extra.go:193` |
| `source` (Source 1) | `pkg/detection/engines_extra.go:244-317` | `"source"` `engines_extra.go:281` |
| `source2` | `pkg/detection/engines_extra.go:321-396` | `"source2"` `engines_extra.go:360` |
| `rage` | `pkg/detection/engines_extra.go:400-467` | `"rage"` `engines_extra.go:431` |
| `fromsoftware` | `pkg/detection/engines_extra.go:471-553` | `"fromsoftware"` `engines_extra.go:506` |
| `factorio` | `pkg/detection/engines_extra.go:557-620` | `"factorio"` `engines_extra.go:580` |
| `zomboid` | `pkg/detection/engines_extra.go:624-706` | `"zomboid"` `engines_extra.go:662` |

### 1.2 Extraction — `pkg/extract` (11 extractors + generic fallback)

- Registered extractors: `unity`, `unreal`, `godot`, `goldsrc`, `source2`, `factorio`, `zomboid`, `sage`, `rage`, `fromsoftware`, `generic` — `pkg/extract/registry.go:50-60`; planned-but-absent engines listed only in a comment — `pkg/extract/registry.go:62`.
- `GetExtractor` **does fall back** to the `generic` extractor for any unknown engine key — `pkg/extract/registry.go:31-43` (fallback `:37-40`).
- The pre-flight tool check validates **only the single `caps.ToolName`** — `pkg/extract/registry.go:76-82`.
- `SupportedEngine()` anchors (verified):
  - `pkg/extract/engines.go:159` (unity), `:248` (unreal), `:484` (godot), `:781` (generic)
  - `pkg/extract/engines_classic.go:364` (goldsrc), `:404` (source2), `:444` (factorio), `:480` (zomboid), `:521` (sage)
  - `pkg/extract/rage_extractor.go:20` (rage), `pkg/extract/fromsoftware_extractor.go:19` (fromsoftware)

### 1.3 Injection — `pkg/inject` (5 injectors registered, custom remapped to generic)

- Registered injectors: `unity`, `unreal`, `godot`, `generic`, `fromsoftware` — `pkg/inject/registry.go:61-71`.
- `GetInjector` maps `"custom"` to `"generic"` automatically (`pkg/inject/registry.go:42-47`).
- Extract-only engines (`sage`, `goldsrc`, `source2`, `factorio`, `zomboid`, `rage`) and detection-only engines (`ue3`, `source`) return explicit, actionable errors (`pkg/inject/registry.go:49-56`).
- SupportedEngine() anchors: `pkg/inject/engines.go:156` (unity), `:236` (unreal), `:327` (godot), `:344` (generic), `pkg/inject/fromsoftware.go:27` (fromsoftware).

### 1.4 Wiring — `app.go`

- `DetectEngine` passes the detector's engine string through untouched — `app.go:166-197` (`Engine` copy `:186`).
- `Extract` saves the project under the detected engine (`core.NewGameProject(..., info.Engine)` `app.go:212`) and calls `extract.Run` — `app.go:199-232` (`:221`).
- `Inject` verifies project/engine equality, then calls `inject.Run` — `app.go:234-270` (`:243-245`, `:261`). Because the project records `"custom"` (`app.go:212`), equality passes and the failure happens inside `inject.Run`.
- `BuildPatcher` requires staged `ModifiedFiles` from a successful injection — `app.go:376-447` (`:419-420`).
- Tool status reported to the UI covers only python / UnityPy / fftools / patcher — `app.go:116-139` (`ToolStatus` fields `:103-113`).

### 1.5 Frontend — `frontend/src`

- Fixed 6-step wizard detect → extract → translate → inject → patch → success — `frontend/src/app/page.tsx:26` and `:29`; step labels (شناسایی/استخراج/تزریق) `:196-200`; inject branch `:102-105`, inject button `:362`.
- Keyword sweep result: **there is no support-matrix UI**. The terms "matrix", "support", "extraction", "write-back" produce **0 hits** in `frontend/src`; all 37 matches in `page.tsx` are step-flow copy (e.g. `:264`, `:271`), and `lib/utils.ts` + all `components/ui/*` files have 0 relevant hits.
- The extract step hardcodes Unity wording ("در حال استخراج (resources.assets) …") shown for **every** engine — `page.tsx:271`; extract success copy — `page.tsx:264`.
- The UI never gates/disables inject or patch per engine — every detected game is offered the full pipeline — `page.tsx:362` (and patch step reachable from there).

### 1.6 README claims — `README.md`

- Support matrix table `README.md:29-42` — header `:31`, rows: Unity `:33`, Unreal `:34`, Godot `:35`, SAGE `:36`, GoldSrc `:37`, Source 2 `:38`, FromSoftware `:39`, Factorio `:40`, Project Zomboid `:41`, RAGE `:42`.
- Caveats: `README.md:14`, `:20`, `:24`, `:45`, `:47`, `:56`.
- Fixture evidence: Godot PCK `:51`, Factorio `:52`, DDPER `"custom engine"` generic path `:53`, Unity raw-byte candidates `:54`.
- Package claims: extract via UnityPy / Go parsers / `repak, gdre_tools, fftools` `:65`; inject limits `:66`; tools registry resolves "UnityPy, gdre_tools, repak, fftools" `:67`.
- Usage workflow promises Detect → Extract → Translate → Stage changes for "supported formats" — `README.md:106-111`.

---

## 2. Per-engine support matrix

Legend: ✅ implemented · ⚠️ conditional/narrow · ❌ absent. "Generic fallback" extraction = plain-text scan of `.txt/.ini/.json/.csv/.xml` — `pkg/extract/engines.go:781-794`.

| # | Engine (detection name) | Detectable? | Extractor? | Injector? | UI/docs claims | Gap |
|---|---|---|---|---|---|---|
| 1 | `unity` | ✅ `pkg/detection/engines.go:17-84` | ✅ Python + UnityPy — `pkg/extract/engines.go:159-176` | ✅ — `pkg/inject/engines.go:156-230` | README:33 — raw-scan candidates rejected, write-back unvalidated on shipped game | Extraction emits `raw_` MonoBehaviour candidates that injection categorically refuses (`pkg/inject/engines.go:173`); README:54 record shows 607/607 entries were raw candidates. Detected+extracted ≠ injectable; needs `py_UnityPy` at inject time (`pkg/inject/engines.go:182`) |
| 2 | `unreal` | ✅ `pkg/detection/engines.go:143-216` | ✅ — `pkg/extract/engines.go:248-470` | ✅ loose `.locres` only — `pkg/inject/engines.go:236-321` (`locres import` `:309`) | README:34 — `.pak` rebuild "not established"; README:65-67 list repak | Caps announce only `fftools` (`pkg/extract/engines.go:253`) but the `.pak` path hard-requires `repak.exe` (`:346-351`, `:371-373`) — the registry pre-check (`pkg/extract/registry.go:76-82`) validates only `caps.ToolName`, so the capability is mis-announced. No `.pak` write-back (loose `.locres` staging only, `pkg/inject/engines.go:293-314`); `.uexp` scan truncates at 8000 files (`pkg/extract/engines.go:470`) |
| 3 | `ue3` (UE3) | ✅ `pkg/detection/engines.go:263-338` | ❌ dedicated — silently falls back to generic (`pkg/extract/registry.go:37-40` → `pkg/extract/engines.go:781-794`) | ❌ (`pkg/inject/registry.go:36-42`) | **Absent from README matrix** (`README.md:29-42`) | Detect-only family: users get generic plain-text scans with no dedicated reader, and injection always fails; no documentation anywhere |
| 4 | `godot` | ✅ `pkg/detection/engines.go:342-401` | ✅ — `pkg/extract/engines.go:484-712` (prefers `gdre_tools` `:508-515`; fallback `extractPckOnly` `:584-595`; CSV-matched `.translation` export `:608-613`) | ⚠️ — `pkg/inject/godot.go:26-200` | README:35 — ⚠️ existing Persian `.translation` in standalone unencrypted PCK v2-v4 | Injection rewrites **only existing** `fa`/`fa_*` `.translation` entries (`pkg/inject/godot.go:46`, `:214-216`) with a matching CSV catalog (`:115`) inside unencrypted PCKs (`:90-93`); everything else (scenes/scripts text, new locale resources) is explicitly rejected (`:64`). Text recovered from `.tres/.gd/.tscn` is therefore extract-not-injectable; README:35 documents this but the UI still offers inject unconditionally (`page.tsx:362`) |
| 5 | `sage` (CnC Generals/ZH/RA3) | ✅ `pkg/detection/engines_extra.go:83-164` | ✅ — `pkg/extract/engines_classic.go:521-618` (`.big` walk `:526-549`, `.csf` `:562`, `.str` `:564`, `parseCsf` `:234`, `.str` parser `:597`) | ❌ | README:36 — ❌ honestly declared | Extract-only; every Inject attempt errors (`pkg/inject/registry.go:37-42`). No `.csf/.big` writer exists |
| 6 | `goldsrc` (HL/CS 1.6) | ✅ `pkg/detection/engines_extra.go:168-240` | ✅ — `pkg/extract/engines_classic.go:364-392` | ❌ | README:37 — ❌ honestly declared | Extract-only; no injector |
| 7 | `source` (Source 1) | ✅ `pkg/detection/engines_extra.go:244-317` | ❌ dedicated — generic fallback only (`pkg/extract/registry.go:37-40`) | ❌ | **Absent from README matrix** (only "Source 2" listed, `README.md:38`) | Detect-only family, undocumented, injection impossible |
| 8 | `source2` (CS2/Deadlock) | ✅ `pkg/detection/engines_extra.go:321-396` | ✅ — `pkg/extract/engines_classic.go:404-433` | ❌ | README:38 — ❌ honestly declared | Extract-only; no injector |
| 9 | `rage` (GTA V) | ✅ `pkg/detection/engines_extra.go:400-467` | ⚠️ TextExtraction-only caps — `pkg/extract/rage_extractor.go:20-23` | ❌ | README:42 — ❌ + "Encrypted RPF7 traversal is not implemented"; README:47 same | (a) No injector. (b) **Docs lag code**: RPF7 traversal IS implemented when key material from the user's own GTA5.exe is available (`rage_extractor.go:15-17`, `:32-50`, `:78-90`) with optional `codewalker_magic` (`:37-41`); only NG-encrypted archives without keys are skipped (`:82-83`, hard error `:99-100`); `mods/` trees skipped (`:62-64`). (c) GTA5.exe / codewalker_magic dependency is invisible to the tool status UI (`app.go:109-113`) and to README:67 |
| 10 | `fromsoftware` | ✅ `pkg/detection/engines_extra.go:471-553` | ⚠️ — `pkg/extract/fromsoftware_extractor.go:15-119` | ❌ | README:39 — focused on ELDEN RING Data0; README:47 historical 520,750-entry run (not revalidated) | No injector. BHD5 path supports only the ER `Data0` key (`fromsoftware_extractor.go:46-55`) — other titles/variants unsupported (`:117`, `:119`); DCX decompression needs the game's own `oo2core_6_win64.dll` (`:85`), which is not tracked by the tools registry (`README.md:67` omits it; `app.go:109-113`) |
| 11 | `factorio` | ✅ `pkg/detection/engines_extra.go:557-620` | ✅ — `pkg/extract/engines_classic.go:444-467` (locale `.cfg`) | ❌ | README:40 — ❌ honestly declared; fixture `:52` | Extract-only; no injector |
| 12 | `zomboid` | ✅ `pkg/detection/engines_extra.go:624-706` | ✅ — `pkg/extract/engines_classic.go:480-507` (Lua tables) | ❌ | README:41 — ❌ honestly declared | Extract-only; no injector |
| 13 | `generic` → result engine **`"custom"`** | ✅ `pkg/detection/engines.go:484-503` (fallback, confidence 0.1, `Engine:"custom"` `:492`) | ✅ via fallback — `pkg/extract/registry.go:37-40` → `GenericExtractor` `pkg/extract/engines.go:781-794` | ❌ **wiring bug** — `GenericInjector` registers as `"generic"` (`pkg/inject/engines.go:344`) but lookup uses `"custom"` with no mapping (`pkg/inject/registry.go:68`, `:36-42`) | README:53 mentions "custom engine" only as a fixture note; README:106-111 promise the full flow | **Critical:** every unrecognized game extracts fine but `Inject` **always fails** with `no injector available for engine: custom` even though a working `GenericInjector` exists. Extract has a fallback; Inject does not (`pkg/extract/registry.go:37-40` vs `pkg/inject/registry.go:36-42`) |

---

## 3. Top gaps (ranked)

1. **[High · functional] `"custom"` → `"generic"` injector key mismatch.** The generic detector reports engine `"custom"` (`pkg/detection/engines.go:492`), extraction falls back to `generic` (`pkg/extract/registry.go:37-40`), but `inject.Run` looks up `GetInjector(info.Engine)` with **no remapping** (`pkg/inject/registry.go:68`) and `GetInjector` has no fallback (`:36-42`) — while `GenericInjector` registers under `"generic"` (`pkg/inject/engines.go:344`). Net effect: the fully-implemented generic write-back path is unreachable from real UI flow; every custom-detected game fails injection (`app.go:261` → `inject/registry.go:69-71`) despite `README.md:53` confirming custom-engine extraction is an expected scenario.
2. **[High · functional/UX] Seven detectable engines are extract-only.** `sage`, `goldsrc`, `source2`, `rage`, `fromsoftware`, `factorio`, `zomboid` have extractors (`pkg/extract/registry.go:53-59`) but no injector (`pkg/inject/registry.go:49-52`) — any Inject attempt aborts (`pkg/inject/registry.go:37-42`). README:36-42 documents ❌ correctly, but the UI offers Injection unconditionally (`page.tsx:362`) with no per-engine gating anywhere (`page.tsx:26`, `:102-105`), so users discover the limit via runtime errors.
3. **[Medium] `ue3` and `source` are detect-only and undocumented.** Both detectors exist (`pkg/detection/engines.go:263-338`, `pkg/detection/engines_extra.go:244-317`) but no dedicated extractor exists — they silently degrade to the generic `.txt/.ini/.json/.csv/.xml` scan (`pkg/extract/registry.go:37-40`, `pkg/extract/engines.go:781-794`) — and no injector. Neither appears in the README matrix (`README.md:29-42`), unlike all other detected families.
4. **[Medium] Tool pre-check blind spots.** The pre-flight check validates only `caps.ToolName` (`pkg/extract/registry.go:76-82`). Unreal's caps announce `fftools` (`pkg/extract/engines.go:253`) but the `.pak` flow hard-requires `repak.exe` (`:371-373`). FromSoftware needs the game's `oo2core_6_win64.dll` (`pkg/extract/fromsoftware_extractor.go:85`). RAGE needs `GTA5.exe` key material plus optional `codewalker_magic` (`pkg/extract/rage_extractor.go:32-50`). None of these are surfaced by `GetToolStatus` (`app.go:116-139`, fields `:103-113`) or by README:67 (only UnityPy, gdre_tools, repak, fftools) — capability discovery fails until a runtime error.
5. **[Medium · product] Godot extraction promises text the injector cannot write back.** Extraction recovers `.tres/.gd/.tscn` text and `.translation` catalogs (`pkg/extract/engines.go:508-515`, `:608-613`), but injection only rewrites existing Persian `fa`/`fa_*` `.translation` entries (`pkg/inject/godot.go:46`, `:214-216`) with a companion CSV (`:115`) inside unencrypted PCKs (`:90-93`) and rejects all other content including new-locale resources (`:64`). README:35 warns about this; the UI does not (`page.tsx:362`).
6. **[Low · docs] README/code drift on RAGE.** README:42 and README:47 claim "Encrypted RPF7 traversal is not implemented", but the extractor implements RPF7 traversal whenever key material can be derived from the user's own GTA5.exe (`pkg/extract/rage_extractor.go:15-17`, `:32-50`, `:78-90`); only keyless NG archives are skipped (`:82-83`, `:99-100`). The README extraction bullet (`README.md:21`) also omits the RAGE GXT2 and FromSoftware FMG readers that the matrix rows (`:39`, `:42`) advertise.
7. **[Low · docs] Workflow text overbroad.** README:106-111 describes Detect → Extract → Translate → "Stage changes" as a universal flow (with a weak "supported formats" hedge at `:111`); combined with the ungated UI (`page.tsx:362`), the effective promise covers engines where injection is impossible or a key mismatch aborts the run (G1/G2/G3).

**What is accurate in the docs:** README:14 and `:45` explicitly warn that detection ≠ extraction ≠ write-back; the matrix rows for SAGE/GoldSrc/Source 2/FromSoftware/Factorio/Zomboid/RAGE correctly declare injection ❌ (README:36-42); Godot's narrow injection scope is correctly scoped at `README.md:35` and `:66`; Unity raw-candidate rejection is documented at `README.md:33` and `:54`. The documentation layer is more honest than the UI flow.

---

## 4. Appendix — verified function/registry index

- Detection registry: `pkg/detection/registry.go:28-34` (priority sort), `:37-80` (`Detect`).
- Extract registry: `pkg/extract/registry.go:26-29` (`Register`), `:31-43` (`GetExtractor` + generic fallback), `:47-65` (`DefaultRegistry`, 11 extractors `:50-60`, future engines `:62`), `:67-94` (`Run`, tool check `:76-82`).
- Inject registry: `pkg/inject/registry.go:31-34` (`Register` by `SupportedEngine` `:32`), `:36-42` (`GetInjector`, no fallback), `:46-55` (`DefaultRegistry`, 4 injectors), `:57-126` (`Run`, lookup `:68`, tool check `:73-79`, post-hash verification `:96-120`).
- Extractor format helpers (no separate engines): SAGE `.big` parser `pkg/extract/engines_classic.go:144-195`, CSF `:197-355`, `.str` `:597-618`; RAGE helpers: key derivation `pkg/extract/rage_extractor.go:32-50`, loose GXT2 `:105-123`; FromSoftware helpers: BHD5 `:46-68`, DCX/oodle `:80-88`.
- Injector internals: Unity `pkg/inject/engines.go:165-230` (`raw_` guard `:173`, UnityPy `:182`); Unreal `:245-321` (fftools required `:247-249`, staged import `:293-314`); Godot `pkg/inject/godot.go:26-200` (extract `:100`, catalog `:115`, export `:122`, import `:169-172`, rebuild `:183-186`); Generic `:352-395` + `injectIntoTextFile` `:401-474` (UTF-8-only `:406-407`).
- App wiring: `app.go:166-197` (DetectEngine), `:199-232` (Extract), `:234-270` (Inject), `:272-284` (LaunchGame), `:376-447` (BuildPatcher).