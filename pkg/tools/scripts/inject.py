#!/usr/bin/env python3
"""FarsiForge Unity injection script.

Usage: python inject.py <game_dir> <work_dir> [tools_dir]

Reads <work_dir>/translated.json (an array of core.StringEntry-shaped
objects as emitted by extract.py), injects every entry that has a
non-empty `translation` field back into the matching Unity objects
(TextAsset and MonoBehaviour) via UnityPy, and writes the modified
*.assets files to <work_dir>/out/. Originals are never overwritten —
the Go side handles backup/copy.

Object matching uses the same id scheme extract.py emits:
    <file_name>_<path_id>_<path>
where <path> is `TextAsset/<asset_name>` for TextAsset objects and a
dotted walk path like `root.field.sub[0].name` for MonoBehaviour fields.

For MonoBehaviour objects whose typetree has been stripped by Unity, this
script reconstructs the typetree from the game's managed DLLs (Mono) or
from Il2CppDumper output (IL2CPP) using UnityPy.helpers.TypeTreeGenerator,
then uses save_typetree with the reconstructed nodes.

Prints a machine-readable SUMMARY line as the final line of output.

Exit codes:
    0  success (even if 0 translations injected)
    2  UnityPy not installed
    3  translated.json missing/invalid or data dir not found
    4  unexpected runtime failure
"""

import os
import sys
import json
import pickle
import re
import subprocess
from typing import Optional

# --- UnityPy import guard ---------------------------------------------------
try:
    import UnityPy
except ImportError:
    print("ERROR: UnityPy is not installed. Run: pip install UnityPy")
    sys.exit(2)

from unity_raw_fields import replace_serialized_strings

# Disable the C boost extension for typetree reading — it uses a different
# node attribute name ('children' vs 'm_Children') and fails on reconstructed
# TypeTreeNode objects. The pure-Python path works correctly.
try:
    import UnityPy.helpers.TypeTreeHelper as _tth

    _tth.read_typetree_boost = None
except Exception:
    pass


# --- Data dir discovery -----------------------------------------------------


def _has_assets_files(d: str) -> bool:
    try:
        for name in os.listdir(d):
            if name.lower().endswith(".assets"):
                return True
    except OSError:
        pass
    return False


def find_data_dir(game_dir: str) -> Optional[str]:
    """Locate the Unity *_Data directory using several strategies."""
    if not os.path.isdir(game_dir):
        return None

    base = os.path.basename(os.path.normpath(game_dir))

    # (b) game_dir itself is the data dir
    if base.endswith("_Data") or _has_assets_files(game_dir):
        return game_dir

    # (a) *_Data directly under game_dir
    try:
        for item in os.listdir(game_dir):
            full = os.path.join(game_dir, item)
            if item.endswith("_Data") and os.path.isdir(full):
                return full
    except OSError:
        pass

    # (c) one level deep
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


def _inject_strings(value, path, base_id, translations, pending):
    """Recursively walk a typetree value, replacing strings whose id matches.

    Returns the (possibly new) value. Appends matched ids to `pending` so the
    caller can commit them only after save_typetree succeeds.
    """
    if isinstance(value, str):
        full_id = f"{base_id}_{path}"
        if full_id in translations:
            pending.append(full_id)
            return translations[full_id]
        return value
    if isinstance(value, dict):
        return {
            k: _inject_strings(v, f"{path}.{k}", base_id, translations, pending)
            for k, v in value.items()
        }
    if isinstance(value, list):
        return [
            _inject_strings(v, f"{path}[{i}]", base_id, translations, pending)
            for i, v in enumerate(value)
        ]
    return value


# --- TypeTreeGenerator helpers (shared logic with extract.py) ---------------


def _detect_il2cpp(game_dir: str, data_dir: str) -> bool:
    """Check if the game uses IL2CPP backend."""
    game_assembly = os.path.join(game_dir, "GameAssembly.dll")
    metadata = os.path.join(data_dir, "il2cpp_data", "Metadata", "global-metadata.dat")
    return os.path.isfile(game_assembly) and os.path.isfile(metadata)


def _run_il2cpp_dumper(
    game_dir: str, data_dir: str, work_dir: str, tools_dir: str
) -> Optional[str]:
    """Run Il2CppDumper silently and return the output directory path."""
    dumper_exe = None
    if tools_dir:
        candidate = os.path.join(tools_dir, "Il2CppDumper", "Il2CppDumper.exe")
        if os.path.isfile(candidate):
            dumper_exe = candidate
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
    metadata = os.path.join(data_dir, "il2cpp_data", "Metadata", "global-metadata.dat")
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
            startupinfo.dwFlags |= 0x00000001  # STARTF_USESHOWWINDOW
            startupinfo.wShowWindow = 0  # SW_HIDE

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


def _create_generator(env, game_dir: str, data_dir: str, work_dir: str, tools_dir: str):
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
    from UnityPy.streams import EndianBinaryReader
    from UnityPy.helpers.TypeTreeHelper import read_typetree

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
    """Save a MonoBehaviour's typetree using reconstructed nodes.

    Uses obj.save_typetree which accepts custom nodes and calls
    set_raw_data() internally.
    """
    try:
        # Convert node to dict list for save_typetree compatibility
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
    for entry in entries:
        if not isinstance(entry, dict):
            continue
        entry_id = entry.get("id", "")
        entries_by_id[entry_id] = entry
        translation = entry.get("translation")
        if translation:
            translations[entry_id] = translation

    raw_fields_by_object = {}
    for entry_id, entry in entries_by_id.items():
        field_path = entry.get("path", "")
        if entry_id in translations and field_path.startswith("rawstr_"):
            object_id = entry_id[: -len("_" + field_path)]
            raw_fields_by_object.setdefault(object_id, []).append(entry)

    # No translations to inject — still emit a summary and exit cleanly.
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
        print(f"ERROR: Could not find a Unity *_Data directory under: {game_dir}")
        sys.exit(3)

    # --- Load Unity environment ---------------------------------------------
    try:
        env = UnityPy.load(data_dir)
    except Exception as e:
        print(f"ERROR: UnityPy failed to load {data_dir}: {e}")
        sys.exit(4)

    stats = {
        "injected": 0,
        "not_found": 0,
        "skipped_no_typetree": 0,
        "reconstructed": 0,
        "errors": 0,
    }
    matched_ids = set()
    modified_files = set()

    # Initialize TypeTreeGenerator for reconstructed typetrees
    gen = None
    gen_error = ""
    tt_cache = _load_typetree_cache(work_dir)

    # Build a lookup of MonoScript objects by path_id for efficiency
    mono_script_map = {}
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

    for obj in env.objects:
        try:
            type_name = obj.type.name
            file_name = getattr(obj.assets_file, "name", "unknown")
            path_id = obj.path_id
            base_id = f"{file_name}_{path_id}"

            # --- TextAsset --------------------------------------------------
            if type_name == "TextAsset":
                try:
                    data = obj.read()
                except Exception:
                    stats["errors"] += 1
                    continue

                asset_name = getattr(data, "name", "") or ""
                text_id = f"{base_id}_TextAsset/{asset_name}"
                if text_id not in translations:
                    continue

                translation = translations[text_id]
                try:
                    # Replace the script/text bytes with the UTF-8 translation.
                    if hasattr(data, "script"):
                        data.script = translation.encode("utf-8")
                    elif hasattr(data, "text"):
                        data.text = translation
                    else:
                        # No writable text field — skip this object.
                        stats["skipped_no_typetree"] += 1
                        continue
                    data.save()
                    matched_ids.add(text_id)
                    modified_files.add(file_name)
                except Exception:
                    stats["errors"] += 1
                    continue

            # --- MonoBehaviour ---------------------------------------------
            elif type_name == "MonoBehaviour":
                raw_entries = raw_fields_by_object.get(base_id, [])
                if raw_entries:
                    raw = obj.get_raw_data()
                    endian = getattr(obj.reader, "endian", "<")
                    replacements = {}
                    for entry in raw_entries:
                        try:
                            offset = int(entry["path"][len("rawstr_") :], 16)
                        except (KeyError, TypeError, ValueError) as exc:
                            raise ValueError(f"invalid raw Unity field path for {entry.get('id')}") from exc
                        if offset in replacements:
                            raise ValueError(f"duplicate raw Unity field offset {offset}")
                        replacements[offset] = (
                            entry.get("source", ""),
                            translations[entry["id"]],
                        )

                    obj.set_raw_data(replace_serialized_strings(raw, replacements, endian))
                    for entry in raw_entries:
                        matched_ids.add(entry["id"])
                    modified_files.add(file_name)
                    continue

                # Strategy 1: typetree nodes available (not stripped)
                nodes = None
                try:
                    nodes = obj.serialized_type.nodes
                except Exception:
                    nodes = None

                tree = None
                used_reconstruction = False

                if nodes:
                    tree = _try_read_typetree(obj)

                # Strategy 2: try read_typetree directly (some IL2CPP builds)
                if not isinstance(tree, dict):
                    tree = _try_read_typetree(obj)

                # Strategy 3: reconstruct typetree from script DLLs
                if not isinstance(tree, dict):
                    # Lazy-init the generator on first need
                    if gen is None and gen_error == "":
                        gen, gen_error = _create_generator(
                            env, game_dir, data_dir, work_dir, tools_dir
                        )

                    if gen is not None:
                        script_info = _get_script_info_fast(obj, mono_script_map)
                        if script_info:
                            assembly, classname, namespace = script_info
                            node = _get_reconstructed_nodes(
                                gen, assembly, classname, namespace, tt_cache
                            )
                            if node is not None:
                                tree = _read_with_reconstructed_nodes(obj, node)
                                if tree is not None:
                                    used_reconstruction = True

                if isinstance(tree, dict):
                    pending = []
                    new_tree = _inject_strings(
                        tree, "root", base_id, translations, pending
                    )
                    if pending:
                        try:
                            if used_reconstruction:
                                # Save with reconstructed nodes
                                script_info = _get_script_info_fast(
                                    obj, mono_script_map
                                )
                                if script_info:
                                    assembly, classname, namespace = script_info
                                    node = _get_reconstructed_nodes(
                                        gen, assembly, classname, namespace, tt_cache
                                    )
                                    if node is not None:
                                        if _save_with_reconstructed_nodes(
                                            obj, new_tree, node
                                        ):
                                            for pid in pending:
                                                matched_ids.add(pid)
                                            modified_files.add(file_name)
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
                                # Save with original typetree nodes
                                obj.save_typetree(new_tree)
                                for pid in pending:
                                    matched_ids.add(pid)
                                modified_files.add(file_name)
                        except Exception:
                            stats["errors"] += 1
                            continue
                else:
                    # No typetree available — skip
                    stats["skipped_no_typetree"] += 1

        except Exception:
            stats["errors"] += 1
            continue

    stats["not_found"] = len(translations) - len(matched_ids)

    # --- Write modified .assets files to <work_dir>/out/ --------------------
    out_dir = os.path.join(work_dir, "out")
    files_written = 0
    try:
        os.makedirs(out_dir, exist_ok=True)
        for asset in env.assets:
            asset_name = getattr(asset, "name", None)
            if asset_name is None or asset_name not in modified_files:
                continue
            try:
                saved = asset.save()
                # Names originate in the game archive; keep all output staged.
                safe_name = os.path.basename(asset_name)
                if safe_name in ("", ".", ".."):
                    stats["errors"] += 1
                    continue
                out_path = os.path.join(out_dir, safe_name)
                with open(out_path, "wb") as f:
                    f.write(saved)
                files_written += 1
            except Exception:
                stats["errors"] += 1
    except OSError as e:
        print(f"ERROR: could not write output to {out_dir}: {e}")
        sys.exit(4)

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
