# Research:  ELDEN RING (FromSoftware) text extraction — 2026-10-09

ELDEN RING `D:\games\ELDEN RING\Game`. Reference implementations: dexyfex/CodeWalker (GPL-3), soulsmods/SoulsFormatsNEXT + ividyon/WitchyBND (GPL-3).

> **Scope note (user directive, 2026-10-09)**: FarsiForge targets **ALL games**, not a fixed set.ELDEN RING are only test cases for understanding engine families. The deliverables below must be engine-family-generic:
> - `bhd5.go`/`dcx.go`/`fmg.go` → **FromSoftware family** (DS1–DS3, Sekiro, ER share BHD5/DCX/BND/FMG; version-switch on BHD5 header variant, path-hash width u32/u64, FMG version 1/2, DCX type).
> Both slot into the existing pkg\detection + pkg\extract registry architecture as family extractors, not per-game hacks.

## ELDEN RING — FromSoftware (GRB-era)

### Verified on the real install
- `Data0.bdt` (11.1 GB) + `Data0.bhd` (1 MB), Data1 (13.3 GB / 5.9 MB), Data2 (19.7 GB / 6.9 MB), Data3 (2.7 GB / 0.4 MB), `DLC.bdt` (15 GB / 1.3 MB). No loose `msg` folder — all text inside archives.
- **`oo2core_6_win64.dll` (1 MB) ships in the game dir** → Oodle decompression can use the game's own DLL (same pattern as existing pkg/tools registry).
- Probed `Data0.bhd` / `DLC.bhd` first 16 bytes: **encrypted garbage** (no BHD5 magic) → header is RSA-encrypted.

### Format chain (from SoulsFormatsNEXT BHD5.cs, WitchyBND DVDBND.cs, FMG.cs)
1. **BHD5 header (ER variant)**: RSA-encrypted with the community-known public key (PEM in WitchyBND `SoulsOodleLib/DVDBND.cs`); decrypt = raw RSA public-key op per block — Go stdlib `crypto/rsa`/`math/big` can do it. After decryption: magic "BHD5", bucketCount, salt, buckets of file headers.
2. **ER file header (64-bit)**: fileNameHash u64, paddedFileSize i32, unpaddedFileSize i32, fileOffset i64, shaHashOffset i64, aesKeyOffset i64.
3. **Path hash (ER)**: normalize path (`\`→`/`, lowercase, leading `/`), then `hash = hash*0x85 + c` (u64). Pre-ER games: u32 with prime 37.
4. **Per-file AES**: header stores 16-byte AES-ECB key + encrypted ranges per file — decrypt ranges in place.
5. **BDT**: raw data at fileOffset (padded size). BDF3/BDF4 magic on data files.
6. **DCX compression**: ER uses Oodle (via game's oo2core dll) and/or zlib variants; SoulsFormats DCX.cs handles all types.
7. **MSGBND**: BND3/BND4 container holding `.fmg` files (item.msgbnd.dcx → menu/item fmg set).
8. **FMG (text)**: version 2 ("wide", DS3/BB/ER layout): byte0, bigEndian, version, 0; fileSize i32, unicode bool, 3 bytes; groupCount i32, stringCount i32, 0xFF i32; varint stringOffsetsOffset; groups (offsetIndex i32, firstID i32, lastID i32, 0 i32); strings at varint offsets, **UTF-16LE** null-terminated (Unicode=true for ER).

### Feasibility in Go
- **Extraction (read)**: RSA header decrypt (~80 lines with math/big), BHD5 parse (~150), path-hash (~15), per-file AES (~40), DCX-Oodle decompress via DLL call (syscall — pattern exists in pkg/tools; or zlib variant via stdlib), BND4 parse (~150), FMG parse (~120). Total ≈ 600–800 lines. Oodle DLL loaded from game dir at runtime.
- **Injection (write)**: rebuild FMG (UTF-16), rebuild MSGBND, DCX recompress (**Oodle compress via DLL** — dll exports compression too), then **rebuild BDT+BHD pair** (padded sizes change → full archive rewrite; UXM-style repack ≈ 300 lines). RSA header re-encryption is the blocker: we only have the PUBLIC key — but the header encryption is the same RSA op with the public key (FromSoft "encrypts" with private, tools "decrypt" with public; writing back = apply public-key op again — WitchyBND repacks .bhd successfully, proving this works).
- **Risk**: Oodle compress via DLL needs correct export signatures (oo2core_6: OodleLZ_Decompress + OodleLZ_Compress); ER DLC.bdt pairs with DLC.bhd — handle all 5 pairs.

## Recommendation
- **Phase 1 (extraction only, both games)**: native Go readers — RPF7(NG/AES/NONE)+GXT2 and BHD5(RSA)+BDT+DCX+BND4+FMG. Both are stdlib-only except ER Oodle (game's own DLL). Estimated ~1,500 lines total + tests against the real installs (read-only, outputs to %TEMP% per scope rules).
- **Phase 2 (injection)**: GXT2 write-back into RPF7 (port CodeWalker space management); ER FMG→MSGBND→DCX→BDT/BHD repack. Larger; do after Phase 1 proves extraction.
- **Alternative**: bundle WitchyBND.exe (CLI) for ER instead of native code — but it's GPL-3 and adds a tool dependency; native Go keeps FarsiForge self-contained and matches the existing engine-extractor architecture (pkg/extract/engines.go pattern).
- **Licensing note**: CodeWalker and SoulsFormatsNEXT are GPL-3 — porting logic to Go (clean-room from format docs, which the above research now documents) avoids license entanglement; do not copy code verbatim.

## Next steps (implementation plan)
1. `pkg/extract/rpf7.go` — RPF7 reader (NG/AES/NONE TOC decrypt, entry tree, Deflate) + `gxt2.go` parser; wire as RAGE extractor strategy; validate: list update.rpf tree, extract american.gxt2, count entries.
2. `pkg/extract/bhd5.go` — RSA header decrypt + BHD5 parse + path hash; `dcx.go` (zlib first, Oodle via game DLL second); `fmg.go` parser; wire as FromSoftware extractor strategy; validate: enumerate Data0.bhd entries, find `/msg/` paths, extract + parse one FMG.
3. Register both in pkg/extract registry + detection matrix update (ELDEN RING fromsoftware 95% → extraction ✓).