#!/usr/bin/env python3
"""FarsiForge Unity injection script — Memory-Optimized & Targeted.

Usage: python inject.py <game_dir> <work_dir> [tools_dir]

Reads <work_dir>/translated.json (an array of core.StringEntry-shaped
objects as emitted by extract.py), injects every entry that has a
non-empty `translation` field back into the matching Unity objects
(TextAsset and MonoBehaviour) via UnityPy, and writes the modified
*.assets files to <work_dir>/out/. Originals are never overwritten —
the Go side handles backup/staging/copy.

Memory Architecture:
- Inspects translated.json to identify ONLY the specific asset files that require modification.
- Never loads .resS, .resource, or raw media stream buffers into memory.
- Loads targeted asset files one at a time, saves modified output directly to work_dir/out/,
  and calls gc.collect() immediately after each file to prevent RAM bloat and freezing on large games.
"""

import gc
import json
import os
import pickle
import re
import subprocess
import sys
from typing import Dict, List, Optional, Set, Tuple

# --- UnityPy import guard ---------------------------------------------------
try:
    import UnityPy
except ImportError:
    print("ERROR: UnityPy is not installed. Run: pip install UnityPy")
    sys.exit(2)

_script_dir = os.path.dirname(os.path.abspath(__file__))
if _script_dir not in sys.path:
    sys.path.insert(0, _script_dir)

from unity_raw_fields import replace_serialized_strings

# Disable the C boost extension for typetree reading — it uses a different
# node attribute name ('children' vs 'm_Children') and fails on reconstructed
# TypeTreeNode objects. The pure-Python path works correctly.
try:
    import UnityPy.helpers.TypeTreeHelper as _tth

    _tth.read_typetree_boost = None
except Exception:
    pass


# --- Data dir discovery & candidate filtering --------------------------------


def _has_assets_files(d: str) -> bool:
    try:
        for name in os.listdir(d):
            nl = name.lower()
            if nl.endswith(".assets") and not nl.endswith(".ress"):
                return True
    except OSError:
        pass
    return False


def find_data_dir(game_dir: str) -> Optional[str]:
    """Locate the Unity *_Data directory using several strategies."""
    if not os.path.isdir(game_dir):
        return None

    base = os.path.basename(os.path.normpath(game_dir))
    if base.endswith("_Data") or _has_assets_files(game_dir):
        return game_dir

    try:
        for item in os.listdir(game_dir):
            full = os.path.join(game_dir, item)
            if item.endswith("_Data") and os.path.isdir(full):
                return full
    except OSError:
        pass

    try:
        for item in os.listdir(game_dir):
            full = os.path.join(game_dir, item)
            if not os.path.isdir(full):
                continue
            if item.endswith("_Data") or _has_assets_files(full):
                return full
            try:
                for sub in os.listdir(full):
                    sub_full = os.path.join(full, sub)
                    if sub.endswith("_Data") and os.path.isdir(sub_full):
                        return sub_full
            except OSError:
                continue
    except OSError:
        pass

    return None


def find_candidate_asset_files(data_dir: str) -> List[Tuple[str, str]]:
    """Return (full_path, rel_to_data_dir) for files containing serialized objects.

    Filters out massive media stream files (.resS, .resource) and temporary directories.
    """
    candidates = []
    ignore_suffixes = (
        ".ress",
        ".resource",
        ".dll",
        ".exe",
        ".pdb",
        ".config",
        ".manifest",
        ".log",
        ".ini",
        ".txt",
        ".png",
        ".jpg",
        ".jpeg",
        ".tga",
        ".dds",
        ".wav",
        ".ogg",
        ".mp3",
    )

    for root, dirs, files in os.walk(data_dir):
        dirs[:] = [
            d
            for d in dirs
            if not d.startswith(".farsiforge")
            and not d.startswith("WorkingDir")
            and d != "il2cpp_dump"
        ]

        for f in files:
            fl = f.lower()
            if any(fl.endswith(s) for s in ignore_suffixes):
                continue

            full_path = os.path.join(root, f)
            rel_path = os.path.relpath(full_path, data_dir).replace("\\", "/")

            if (
                fl.endswith(".assets")
                or fl.endswith(".bundle")
                or fl.endswith(".unity3d")
                or fl.startswith("level")
                or fl.startswith("sharedassets")
                or fl.startswith("globalgamemanagers")
            ):
                candidates.append((full_path, rel_path))

    return candidates


# --- Typetree helpers -------------------------------------------------------


def _try_read_typetree(obj) -> Optional[dict]:
    """Attempt read_typetree(); return dict or None."""
    try:
        return obj.read_typetree()
    except Exception:
        return None


def _try_read_object(obj):
    """Attempt obj.read(); return object or None."""
    try:
        return obj.read()
    except Exception:
        return None


def _inject_strings(
    value, path, candidate_base_ids, translations, entries_by_id, pending, stats=None
):
    """Recursively walk a typetree value, replacing strings whose id matches.

    Verifies source text before replacement to reject stale source states.
    Appends matched ids to `pending` so the caller can commit them only after
    save_typetree succeeds.
    """
    if isinstance(value, str):
        for base_id in candidate_base_ids:
            full_id = f"{base_id}_{path}"
            if full_id in translations:
                entry = entries_by_id.get(full_id, {})
                if entry.get("read_only"):
                    if stats is not None:
                        stats["skipped_no_typetree"] = (
                            stats.get("skipped_no_typetree", 0) + 1
                        )
                    return value
                expected_src = entry.get("source")
                if expected_src and value.strip() != expected_src.strip():
                    if stats is not None:
                        stats["errors"] = stats.get("errors", 0) + 1
                    print(
                        f"WARNING: source mismatch for {full_id}: expected {expected_src!r}, found {value!r}"
                    )
                    return value
                pending.append(full_id)
                return translations[full_id]
        return value
    if isinstance(value, dict):
        return {
            k: _inject_strings(
                v,
                f"{path}.{k}",
                candidate_base_ids,
                translations,
                entries_by_id,
                pending,
                stats,
            )
            for k, v in value.items()
        }
    if isinstance(value, list):
        return [
            _inject_strings(
                v,
                f"{path}[{i}]",
                candidate_base_ids,
                translations,
                entries_by_id,
                pending,
                stats,
            )
            for i, v in enumerate(value)
        ]
    return value


# --- TypeTreeGenerator helpers (shared logic with extract.py) ---------------


def _detect_il2cpp(game_dir: str, data_dir: str) -> bool:
    """Check if the game uses IL2CPP backend."""
    game_assembly = os.path.join(game_dir, "GameAssembly.dll")
    metadata = os.path.join(
        data_dir, "il2cpp_data", "Metadata", "global-metadata.dat"
    )
    return os.path.isfile(game_assembly) and os.path.isfile(metadata)


def _run_il2cpp_dumper(
    game_dir: str, data_dir: str, work_dir: str, tools_dir: str
) -> Optional[str]:
    """Run Il2CppDumper silently and return the output directory path."""
    dumper_exe = None
    if tools_dir:
        for candidate in [
            os.path.join(tools_dir, "Il2CppDumper", "Il2CppDumper.exe"),
            os.path.join(tools_dir, "Il2CppDumper.exe"),
        ]:
            if os.path.isfile(candidate):
                dumper_exe = candidate
                break
    if not dumper_exe:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        candidate = os.path.join(
            os.path.dirname(script_dir), "Il2CppDumper", "Il2CppDumper.exe"
        )
        if os.path.isfile(candidate):
            dumper_exe = candidate
    if not dumper_exe:
        return None

    game_assembly = os.path.join(game_dir, "GameAssembly.dll")
    metadata = os.path.join(
        data_dir, "il2cpp_data", "Metadata", "global-metadata.dat"
    )
    dump_dir = os.path.join(work_dir, "il2cpp_dump")
    os.makedirs(dump_dir, exist_ok=True)

    script_json = os.path.join(dump_dir, "script.json")
    if os.path.isfile(script_json):
        return dump_dir

    try:
        creationflags = 0
        startupinfo = None
        if sys.platform == "win32":
            startupinfo = subprocess.STARTUPINFO()
            startupinfo.dwFlags |= 0x00000001
            startupinfo.wShowWindow = 0

        proc = subprocess.run(
            [dumper_exe, game_assembly, metadata, dump_dir],
            stdin=None,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            creationflags=creationflags,
            startupinfo=startupinfo,
            timeout=300,
        )
        if proc.returncode != 0:
            return None
    except Exception:
        return None

    if not os.path.isfile(script_json):
        return None

    return dump_dir


def _create_generator(
    env, game_dir: str, data_dir: str, work_dir: str, tools_dir: str
):
    """Create and initialize a TypeTreeGenerator for the game."""
    try:
        from UnityPy.helpers.TypeTreeGenerator import TypeTreeGenerator
    except ImportError:
        return None, "TypeTreeGenerator not available"

    unity_version = None
    if env.assets:
        unity_version = str(env.assets[0].unity_version)
    if not unity_version:
        return None, "Could not determine Unity version"

    try:
        gen = TypeTreeGenerator(unity_version)
    except Exception as e:
        return None, f"TypeTreeGenerator init failed: {e}"

    if _detect_il2cpp(game_dir, data_dir):
        dump_dir = _run_il2cpp_dumper(game_dir, data_dir, work_dir, tools_dir)
        if dump_dir is None:
            return None, "Il2CppDumper failed"

        dummy_dll_dir = os.path.join(dump_dir, "DummyDll")
        if not os.path.isdir(dummy_dll_dir):
            return None, f"DummyDll directory not found: {dummy_dll_dir}"

        try:
            gen.load_local_dll_folder(dummy_dll_dir)
        except Exception as e:
            return None, f"Failed to load DummyDlls: {e}"
    else:
        managed_dir = os.path.join(data_dir, "Managed")
        if not os.path.isdir(managed_dir):
            return None, f"Managed directory not found: {managed_dir}"

        try:
            gen.load_local_dll_folder(managed_dir)
        except Exception as e:
            return None, f"Failed to load Managed DLLs: {e}"

    return gen, None


def _load_typetree_cache(work_dir: str) -> dict:
    """Load cached typetree nodes from work_dir."""
    cache_path = os.path.join(work_dir, "typetree_cache.pkl")
    if os.path.isfile(cache_path):
        try:
            with open(cache_path, "rb") as f:
                return pickle.load(f)
        except Exception:
            pass
    return {}


def _get_script_info_fast(obj, mono_script_map: dict) -> Optional[tuple]:
    """Extract (assembly, classname, namespace) using a pre-built MonoScript map."""
    from UnityPy.streams import EndianBinaryReader

    try:
        raw = obj.get_raw_data()
        reader = EndianBinaryReader(raw, endian="<")
        reader.read_int()
        reader.read_long()
        reader.read_boolean()
        reader.align_stream()
        script_file = reader.read_int()
        script_path = reader.read_long()

        if script_path in mono_script_map:
            return mono_script_map[script_path]
    except Exception:
        pass
    return None


def _get_reconstructed_nodes(
    gen, assembly: str, classname: str, namespace: str, cache: dict
):
    """Get reconstructed typetree nodes for a script class."""
    cache_key = (assembly, classname, namespace)
    if cache_key in cache:
        return cache[cache_key]

    fullname = classname
    if namespace:
        fullname = f"{namespace}.{classname}"

    try:
        node = gen.get_nodes_up(assembly, fullname)
        cache[cache_key] = node
        return node
    except Exception:
        cache[cache_key] = None
        return None


def _read_with_reconstructed_nodes(obj, node) -> Optional[dict]:
    """Read a MonoBehaviour's typetree using reconstructed nodes."""
    from UnityPy.helpers.TypeTreeHelper import read_typetree
    from UnityPy.streams import EndianBinaryReader

    try:
        raw = obj.get_raw_data()
        reader = EndianBinaryReader(raw, endian="<")
        tree = read_typetree(
            node,
            reader,
            as_dict=True,
            byte_size=len(raw),
            check_read=False,
        )
        return tree
    except Exception:
        return None


def _save_with_reconstructed_nodes(obj, tree, node) -> bool:
    """Save a MonoBehaviour's typetree using reconstructed nodes."""
    try:
        node_list = node.to_dict_list()
        obj.save_typetree(tree, nodes=node_list)
        return True
    except Exception:
        return False


# --- Main -------------------------------------------------------------------


def main():
    if len(sys.argv) < 3:
        print("Usage: inject.py <game_dir> <work_dir> [tools_dir]")
        sys.exit(1)

    game_dir = sys.argv[1]
    work_dir = sys.argv[2]
    tools_dir = sys.argv[3] if len(sys.argv) > 3 else ""

    # --- Load translated.json ------------------------------------------------
    in_json = os.path.join(work_dir, "translated.json")
    if not os.path.isfile(in_json):
        print(f"ERROR: translated.json not found: {in_json}")
        sys.exit(3)

    try:
        with open(in_json, "r", encoding="utf-8") as f:
            entries = json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        print(f"ERROR: could not read translated.json ({in_json}): {e}")
        sys.exit(3)

    if not isinstance(entries, list):
        print(
            f"ERROR: translated.json must be a JSON array, got {type(entries).__name__}"
        )
        sys.exit(3)

    # Build translation lookup: id -> translation (only non-empty)
    translations = {}
    entries_by_id = {}
    targeted_file_names = set()

    for entry in entries:
        if not isinstance(entry, dict):
            continue
        entry_id = entry.get("id", "")
        entries_by_id[entry_id] = entry
        translation = entry.get("translation") or entry.get("target")
        if translation:
            translations[entry_id] = translation
            # Track file references from entry
            file_ref = entry.get("file", "")
            if file_ref:
                targeted_file_names.add(file_ref)
                targeted_file_names.add(os.path.basename(file_ref))
            if "_" in entry_id:
                # Fallback: extract file_name from prefix of entry_id
                prefix = entry_id.split("_")[0]
                targeted_file_names.add(prefix)

    raw_fields_by_object = {}
    for entry_id, entry in entries_by_id.items():
        field_path = entry.get("path", "")
        if entry_id in translations and field_path.startswith("rawstr_"):
            object_id = entry_id[: -len("_" + field_path)]
            raw_fields_by_object.setdefault(object_id, []).append(entry)

    # No translations to inject — emit a summary and exit cleanly
    if not translations:
        _summary_and_exit(
            work_dir,
            injected=0,
            not_found=0,
            skipped_no_typetree=0,
            reconstructed=0,
            errors=0,
            files_modified=0,
        )

    # --- Resolve data dir ----------------------------------------------------
    data_dir = find_data_dir(game_dir)
    if not data_dir:
        print(
            f"ERROR: Could not find a Unity *_Data directory under: {game_dir}"
        )
        sys.exit(3)

    # --- Discover candidate asset files --------------------------------------
    candidate_files = find_candidate_asset_files(data_dir)

    # Filter to targeted files only
    files_to_process = []
    helper_files = []

    for full_path, rel_path in candidate_files:
        base_name = os.path.basename(full_path)
        is_targeted = (
            rel_path in targeted_file_names
            or base_name in targeted_file_names
            or any(t in rel_path for t in targeted_file_names)
        )
        if is_targeted:
            files_to_process.append((full_path, rel_path))
        elif base_name in ("globalgamemanagers.assets", "resources.assets"):
            # Potential MonoScript helper file
            helper_files.append((full_path, rel_path))

    # If no files matched targeted_file_names directly, fall back to checking all candidate files
    if not files_to_process:
        files_to_process = candidate_files

    print(
        f"Processing {len(files_to_process)} targeted Unity asset file(s) for injection"
    )

    stats = {
        "injected": 0,
        "not_found": 0,
        "skipped_no_typetree": 0,
        "reconstructed": 0,
        "errors": 0,
    }
    matched_ids = set()
    modified_files = set()
    out_dir = os.path.join(work_dir, "out")
    os.makedirs(out_dir, exist_ok=True)
    files_written = 0

    # Initialize TypeTreeGenerator and MonoScript map
    gen = None
    gen_error = ""
    tt_cache = _load_typetree_cache(work_dir)
    mono_script_map: Dict[int, Tuple[str, str, str]] = {}

    # Pre-populate mono_script_map from small helper files if available
    for h_full, _ in helper_files:
        try:
            h_env = UnityPy.load(h_full)
            for obj in h_env.objects:
                if obj.type.name == "MonoScript":
                    try:
                        ms = obj.read()
                        mono_script_map[obj.path_id] = (
                            ms.m_AssemblyName or "",
                            ms.m_ClassName or "",
                            ms.m_Namespace or "",
                        )
                    except Exception:
                        pass
            del h_env
        except Exception:
            pass

    # --- Process targeted files one-by-one -----------------------------------
    for full_path, rel_path in files_to_process:
        try:
            env = UnityPy.load(full_path)
        except Exception:
            continue

        file_has_changes = False
        base_filename = os.path.basename(full_path)

        # Collect any MonoScripts in this file too
        for obj in env.objects:
            if obj.type.name == "MonoScript":
                try:
                    ms = obj.read()
                    mono_script_map[obj.path_id] = (
                        ms.m_AssemblyName or "",
                        ms.m_ClassName or "",
                        ms.m_Namespace or "",
                    )
                except Exception:
                    pass

        # Inspect and inject into objects
        for obj in env.objects:
            try:
                type_name = obj.type.name
                if type_name not in ("TextAsset", "MonoBehaviour"):
                    continue

                asset_file_name = getattr(obj.assets_file, "name", "") or ""
                path_id = obj.path_id

                # Support matching against both relative path and asset filename
                candidate_base_ids = [
                    f"{rel_path}_{path_id}",
                    f"{base_filename}_{path_id}",
                ]
                if asset_file_name:
                    candidate_base_ids.append(f"{asset_file_name}_{path_id}")

                # --- TextAsset ----------------------------------------------
                if type_name == "TextAsset":
                    try:
                        data = obj.read()
                    except Exception:
                        stats["errors"] += 1
                        continue

                    asset_name = getattr(data, "m_Name", getattr(data, "name", "")) or ""
                    # Find matching translation candidates: full match or structured subpaths
                    matching_entries = {}
                    for bid in candidate_base_ids:
                        prefix = f"{bid}_TextAsset/{asset_name}"
                        for tid, val in translations.items():
                            if tid == prefix or tid.startswith(prefix + "#"):
                                matching_entries[tid] = (prefix, val)

                    if not matching_entries:
                        continue

                    # Read current text from TextAsset
                    raw = getattr(data, "m_Script", getattr(data, "script", getattr(data, "text", None)))
                    if raw is None:
                        raw = b""
                    if isinstance(raw, bytes):
                        current_text = raw.decode("utf-8", errors="ignore")
                    else:
                        current_text = str(raw) if raw else ""

                    # Case A: Exact full-file match
                    exact_tid = None
                    for tid, (pfx, _) in matching_entries.items():
                        if tid == pfx:
                            exact_tid = tid
                            break

                    if exact_tid and len(matching_entries) == 1:
                        entry = entries_by_id.get(exact_tid, {})
                        if entry.get("read_only"):
                            stats["skipped_no_typetree"] += 1
                            continue
                        expected_src = entry.get("source")
                        if expected_src and current_text.strip() != expected_src.strip():
                            print(
                                f"WARNING: source mismatch for {exact_tid}: expected {expected_src[:30]!r}, found {current_text[:30]!r}"
                            )
                            stats["errors"] += 1
                            continue
                        new_content = matching_entries[exact_tid][1]
                        try:
                            if hasattr(data, "m_Script"):
                                if isinstance(data.m_Script, bytes):
                                    data.m_Script = new_content.encode("utf-8")
                                else:
                                    data.m_Script = new_content
                            elif hasattr(data, "script"):
                                data.script = new_content.encode("utf-8")
                            elif hasattr(data, "text"):
                                data.text = new_content
                            else:
                                stats["skipped_no_typetree"] += 1
                                continue
                            data.save()
                            matched_ids.add(exact_tid)
                            file_has_changes = True
                        except Exception:
                            stats["errors"] += 1
                        continue

                    # Case B: Structured sub-path entries (JSON / CSV / KV)
                    json_entries = {}
                    csv_entries = {}
                    kv_entries = {}
                    for tid, (pfx, val) in matching_entries.items():
                        if "#" in tid:
                            subpath = tid.split("#", 1)[1]
                            if subpath.startswith("jpath:"):
                                subpath = subpath[len("jpath:") :]
                            entry = entries_by_id.get(tid, {})
                            if entry.get("read_only"):
                                continue
                            if entry.get("source_type") == "textasset-json" or (
                                subpath.startswith("{")
                                or "." in subpath
                                or ("[" in subpath and not subpath.startswith("row["))
                            ):
                                json_entries[subpath] = (
                                    tid,
                                    val,
                                    entry.get("source", ""),
                                )
                            elif entry.get(
                                "source_type"
                            ) == "textasset-csv" or subpath.startswith("row["):
                                csv_entries[subpath] = (
                                    tid,
                                    val,
                                    entry.get("source", ""),
                                )
                            else:
                                kv_entries[subpath] = (
                                    tid,
                                    val,
                                    entry.get("source", ""),
                                )

                    updated_text = None
                    if json_entries:
                        try:
                            parsed_json = json.loads(current_text)
                            for jpath, (tid, trans, exp_src) in json_entries.items():
                                parts = re.findall(r"([^.\[\]]+|\[\d+\])", jpath)
                                curr = parsed_json
                                for p_idx, part in enumerate(parts[:-1]):
                                    if part.startswith("[") and part.endswith("]"):
                                        idx = int(part[1:-1])
                                        curr = curr[idx]
                                    else:
                                        curr = curr[part]
                                last_part = parts[-1]
                                if last_part.startswith("[") and last_part.endswith("]"):
                                    idx = int(last_part[1:-1])
                                    if exp_src and curr[idx].strip() != exp_src.strip():
                                        continue
                                    curr[idx] = trans
                                else:
                                    if exp_src and curr[last_part].strip() != exp_src.strip():
                                        continue
                                    curr[last_part] = trans
                                matched_ids.add(tid)
                            updated_text = json.dumps(
                                parsed_json, ensure_ascii=False, indent=2
                            )
                        except Exception:
                            pass

                    elif csv_entries:
                        try:
                            import csv as _csv
                            import io as _io

                            first_line = (
                                current_text.splitlines()[0]
                                if current_text.splitlines()
                                else ""
                            )
                            delimiter = "\t" if "\t" in first_line else ","
                            lines = current_text.splitlines()
                            reader = list(_csv.reader(lines, delimiter=delimiter))
                            for cpath, (tid, trans, exp_src) in csv_entries.items():
                                m = re.match(r"row\[(\d+)\]\.col\[(\d+)\]", cpath)
                                if m:
                                    r_idx, c_idx = int(m.group(1)), int(m.group(2))
                                    if r_idx < len(reader) and c_idx < len(
                                        reader[r_idx]
                                    ):
                                        if (
                                            exp_src
                                            and reader[r_idx][c_idx].strip()
                                            != exp_src.strip()
                                        ):
                                            continue
                                        reader[r_idx][c_idx] = trans
                                        matched_ids.add(tid)
                            out_io = _io.StringIO()
                            writer = _csv.writer(
                                out_io, delimiter=delimiter, lineterminator="\n"
                            )
                            writer.writerows(reader)
                            updated_text = out_io.getvalue()
                        except Exception:
                            pass

                    elif kv_entries:
                        try:
                            lines = current_text.splitlines()
                            for kpath, (tid, trans, exp_src) in kv_entries.items():
                                for i, line in enumerate(lines):
                                    m = re.match(
                                        r"^(" + re.escape(kpath) + r")\s*([=:])\s*(.*)$",
                                        line,
                                    )
                                    if m:
                                        sep = m.group(2)
                                        lines[i] = f"{kpath}{sep} {trans}"
                                        matched_ids.add(tid)
                                        break
                            updated_text = "\n".join(lines)
                        except Exception:
                            pass

                    if updated_text is not None:
                        try:
                            if hasattr(data, "m_Script"):
                                if isinstance(data.m_Script, bytes):
                                    data.m_Script = updated_text.encode("utf-8")
                                else:
                                    data.m_Script = updated_text
                            elif hasattr(data, "script"):
                                data.script = updated_text.encode("utf-8")
                            elif hasattr(data, "text"):
                                data.text = updated_text
                            data.save()
                            file_has_changes = True
                        except Exception:
                            stats["errors"] += 1

                # --- MonoBehaviour -----------------------------------------
                elif type_name == "MonoBehaviour":
                    raw_entries = []
                    for bid in candidate_base_ids:
                        if bid in raw_fields_by_object:
                            raw_entries = raw_fields_by_object[bid]
                            break

                    if raw_entries:
                        raw = obj.get_raw_data()
                        endian = getattr(obj.reader, "endian", "<")
                        replacements = {}
                        for entry in raw_entries:
                            try:
                                offset = int(entry["path"][len("rawstr_") :], 16)
                            except (KeyError, TypeError, ValueError) as exc:
                                raise ValueError(
                                    f"invalid raw Unity field path for {entry.get('id')}"
                                ) from exc
                            if offset in replacements:
                                raise ValueError(
                                    f"duplicate raw Unity field offset {offset}"
                                )
                            replacements[offset] = (
                                entry.get("source", ""),
                                translations[entry["id"]],
                            )

                        obj.set_raw_data(
                            replace_serialized_strings(raw, replacements, endian)
                        )
                        for entry in raw_entries:
                            matched_ids.add(entry["id"])
                        file_has_changes = True
                        continue

                    # Typetree injection
                    nodes = None
                    try:
                        nodes = obj.serialized_type.nodes
                    except Exception:
                        nodes = None

                    tree = None
                    used_reconstruction = False

                    if nodes:
                        tree = _try_read_typetree(obj)

                    if not isinstance(tree, dict):
                        tree = _try_read_typetree(obj)

                    if not isinstance(tree, dict):
                        if gen is None and gen_error == "":
                            gen, gen_error = _create_generator(
                                env, game_dir, data_dir, work_dir, tools_dir
                            )

                        if gen is not None:
                            script_info = _get_script_info_fast(
                                obj, mono_script_map
                            )
                            if script_info:
                                assembly, classname, namespace = script_info
                                node = _get_reconstructed_nodes(
                                    gen,
                                    assembly,
                                    classname,
                                    namespace,
                                    tt_cache,
                                )
                                if node is not None:
                                    tree = _read_with_reconstructed_nodes(
                                        obj, node
                                    )
                                    if tree is not None:
                                        used_reconstruction = True

                    if isinstance(tree, dict):
                        pending = []
                        new_tree = _inject_strings(
                            tree,
                            "root",
                            candidate_base_ids,
                            translations,
                            entries_by_id,
                            pending,
                            stats,
                        )
                        if pending:
                            try:
                                if used_reconstruction:
                                    script_info = _get_script_info_fast(
                                        obj, mono_script_map
                                    )
                                    if script_info:
                                        assembly, classname, namespace = (
                                            script_info
                                        )
                                        node = _get_reconstructed_nodes(
                                            gen,
                                            assembly,
                                            classname,
                                            namespace,
                                            tt_cache,
                                        )
                                        if node is not None:
                                            if _save_with_reconstructed_nodes(
                                                obj, new_tree, node
                                            ):
                                                for pid in pending:
                                                    matched_ids.add(pid)
                                                file_has_changes = True
                                                stats["reconstructed"] += 1
                                            else:
                                                stats["errors"] += 1
                                                continue
                                        else:
                                            stats["errors"] += 1
                                            continue
                                    else:
                                        stats["errors"] += 1
                                        continue
                                else:
                                    obj.save_typetree(new_tree)
                                    for pid in pending:
                                        matched_ids.add(pid)
                                    file_has_changes = True
                            except Exception:
                                stats["errors"] += 1
                                continue
                    else:
                        stats["skipped_no_typetree"] += 1

            except Exception:
                stats["errors"] += 1
                continue

        # If any changes were made to objects in this file, save asset immediately
        if file_has_changes:
            for asset in env.assets:
                try:
                    saved = asset.save()
                    asset_name = getattr(asset, "name", "") or base_filename
                    safe_name = os.path.basename(asset_name)
                    if safe_name in ("", ".", ".."):
                        safe_name = base_filename

                    out_path = os.path.join(out_dir, safe_name)
                    with open(out_path, "wb") as f:
                        f.write(saved)

                    modified_files.add(safe_name)
                    files_written += 1
                except Exception:
                    stats["errors"] += 1

        # Explicitly release memory immediately
        del env
        gc.collect()

    stats["not_found"] = len(translations) - len(matched_ids)

    _summary_and_exit(
        work_dir,
        injected=len(matched_ids),
        not_found=stats["not_found"],
        skipped_no_typetree=stats["skipped_no_typetree"],
        reconstructed=stats["reconstructed"],
        errors=stats["errors"],
        files_modified=files_written,
        modified_files=sorted(modified_files),
    )


def _summary_and_exit(
    work_dir,
    injected,
    not_found,
    skipped_no_typetree,
    reconstructed,
    errors,
    files_modified,
    modified_files=None,
):
    """Print the machine-readable SUMMARY line and exit 0."""
    out_dir = os.path.join(work_dir, "out")
    summary = {
        "injected": injected,
        "not_found": not_found,
        "skipped_no_typetree": skipped_no_typetree,
        "reconstructed": reconstructed,
        "errors": errors,
        "files_modified": files_modified,
        "modified_files": modified_files or [],
        "output": out_dir,
    }
    print(
        f"Injected {injected} translation(s) into {files_modified} file(s) at {out_dir}"
    )
    print(f"SUMMARY: {json.dumps(summary, ensure_ascii=False)}")
    sys.exit(0)


if __name__ == "__main__":
    main()
