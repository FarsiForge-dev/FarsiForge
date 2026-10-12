#!/usr/bin/env python3
"""FarsiForge Unity extraction script.

Usage: python extract.py <game_dir> <work_dir> [tools_dir]

Walks a Unity game's *_Data directory, extracts translatable strings from
TextAsset and MonoBehaviour objects via UnityPy, filters noise, deduplicates,
and writes <work_dir>/extracted.json as an array of core.StringEntry-shaped
objects. Prints a machine-readable SUMMARY line as the final line of output.

For MonoBehaviour objects whose typetree has been stripped by Unity, this
script reconstructs the typetree from the game's managed DLLs (Mono) or
from Il2CppDumper output (IL2CPP) using UnityPy.helpers.TypeTreeGenerator.

Exit codes:
    0  success (even if 0 strings extracted)
    2  UnityPy not installed
    3  data dir not found
    4  work_dir not writable / other error
"""

import os
import re
import sys
import json
import pickle
import subprocess
from typing import Optional

# --- UnityPy import guard ---------------------------------------------------
try:
    import UnityPy
except ImportError:
    print("ERROR: UnityPy is not installed. Run: pip install UnityPy")
    sys.exit(2)

from unity_raw_fields import scan_serialized_strings

# Disable the C boost extension for typetree reading — it uses a different
# node attribute name ('children' vs 'm_Children') and fails on reconstructed
# TypeTreeNode objects. The pure-Python path works correctly.
try:
    import UnityPy.helpers.TypeTreeHelper as _tth

    _tth.read_typetree_boost = None
except Exception:
    pass


# --- Filters ----------------------------------------------------------------

_GUID_RE = re.compile(
    r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-"
    r"[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
)
_URL_RE = re.compile(r"^(https?://|www\.)", re.IGNORECASE)
_PATH_RE = re.compile(r"^[^\s]*[\\/][^\s]*\.[A-Za-z0-9]{1,8}$")
_NAMESPACE_RE = re.compile(r"^(UnityEngine|System|TMPro|UnityEditor)\.")
_NUM_RE = re.compile(r"^[+-]?\d+(\.\d+)?$")
_REPEAT_RE = re.compile(r"^(.)\1+$")

# Single-token identifier patterns (code identifiers, not human text)
_PASCAL_RE = re.compile(r"^[A-Z][a-zA-Z0-9]*$")
_CAMEL_RE = re.compile(r"^[a-z][a-zA-Z0-9]*$")
_SNAKE_RE = re.compile(r"^[A-Z][A-Z0-9_]*$")
_CONST_RE = re.compile(r"^[A-Z][A-Z0-9_]*$")

# Field names that indicate human-readable text content
_TEXT_FIELD_RE = re.compile(
    r"(text|label|title|name|description|message|tooltip|string)",
    re.IGNORECASE,
)


def is_noise(s: str) -> bool:
    """Return True if a string should be filtered out as non-translatable."""
    if s is None:
        return True
    s = s.strip()
    if len(s) < 2:
        return True
    if _NUM_RE.match(s):
        return True
    if _URL_RE.match(s):
        return True
    if _PATH_RE.match(s):
        return True
    if _GUID_RE.match(s):
        return True
    if _NAMESPACE_RE.match(s):
        return True
    if _REPEAT_RE.match(s):
        return True
    return False


def is_code_identifier(s: str) -> bool:
    """Return True if a string looks like a code identifier (not human text).

    Single-token PascalCase/camelCase/SNAKE_CASE strings are likely code
    identifiers (class names, enum values, method names) rather than
    human-readable text.
    """
    s = s.strip()
    # Multi-word strings (contain spaces) are likely human-readable
    if " " in s:
        return False
    # Single-token identifiers
    if _PASCAL_RE.match(s) or _CAMEL_RE.match(s) or _SNAKE_RE.match(s):
        return True
    return False


# Unity's base MonoBehaviour field m_Name is the object name — almost always
# a code identifier (class name, asset name), not translatable human text.
_MNAME_RE = re.compile(r"^m_Name$", re.IGNORECASE)


def should_extract_string(s: str, field_name: str = "") -> bool:
    """Decide whether a string should be extracted as translatable.

    Applies the noise filter, then the identifier-noise filter:
    skip single-token PascalCase/camelCase/SNAKE_CASE strings UNLESS the
    field name contains a text-related keyword. The Unity base field
    m_Name is excluded from the keyword exception — it holds the object
    name, which is almost always a code identifier.
    """
    if is_noise(s):
        return False
    # Identifier-noise filter: skip code identifiers unless field name
    # indicates text content
    if is_code_identifier(s):
        # m_Name (Unity base field) is the object name, not translatable text
        if field_name and _MNAME_RE.match(field_name):
            return False
        if field_name and _TEXT_FIELD_RE.search(field_name):
            return True
        return False
    return True


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


# --- Typetree / object walking ----------------------------------------------


def _walk_value(value, path, emit, file_name, path_id):
    """Recursively walk a parsed value, emitting string entries."""
    if isinstance(value, str):
        # Extract field name from path (last component after '.' or '/')
        field_name = ""
        if "." in path:
            field_name = path.rsplit(".", 1)[-1]
        elif "/" in path:
            field_name = path.rsplit("/", 1)[-1]
        # Strip array index from field name
        field_name = re.sub(r"\[\d+\]$", "", field_name)
        if should_extract_string(value, field_name):
            emit(file_name, path_id, path, value, "MonoBehaviour")
    elif isinstance(value, dict):
        for k, v in value.items():
            _walk_value(v, f"{path}.{k}", emit, file_name, path_id)
    elif isinstance(value, list):
        for i, v in enumerate(value):
            _walk_value(v, f"{path}[{i}]", emit, file_name, path_id)
    elif value is not None and not isinstance(value, (int, float, bool, bytes)):
        # Object with attributes (fallback for obj.read() results)
        try:
            attrs = vars(value)
        except TypeError:
            return
        for k, v in attrs.items():
            if k.startswith("_"):
                continue
            _walk_value(v, f"{path}.{k}", emit, file_name, path_id)


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


# --- TypeTreeGenerator helpers ----------------------------------------------


def _detect_il2cpp(game_dir: str, data_dir: str) -> bool:
    """Check if the game uses IL2CPP backend."""
    game_assembly = os.path.join(game_dir, "GameAssembly.dll")
    metadata = os.path.join(data_dir, "il2cpp_data", "Metadata", "global-metadata.dat")
    return os.path.isfile(game_assembly) and os.path.isfile(metadata)


def _run_il2cpp_dumper(
    game_dir: str, data_dir: str, work_dir: str, tools_dir: str
) -> Optional[str]:
    """Run Il2CppDumper silently and return the output directory path.

    Returns None if Il2CppDumper is not found or fails.
    """
    # Find Il2CppDumper.exe
    dumper_exe = None
    if tools_dir:
        candidate = os.path.join(tools_dir, "Il2CppDumper", "Il2CppDumper.exe")
        if os.path.isfile(candidate):
            dumper_exe = candidate
    if not dumper_exe:
        # Try relative to the script's parent Tools dir
        script_dir = os.path.dirname(os.path.abspath(__file__))
        candidate = os.path.join(
            os.path.dirname(script_dir), "Il2CppDumper", "Il2CppDumper.exe"
        )
        if os.path.isfile(candidate):
            dumper_exe = candidate
    if not dumper_exe:
        print("WARNING: Il2CppDumper.exe not found — IL2CPP typetrees unavailable")
        return None

    game_assembly = os.path.join(game_dir, "GameAssembly.dll")
    metadata = os.path.join(data_dir, "il2cpp_data", "Metadata", "global-metadata.dat")
    dump_dir = os.path.join(work_dir, "il2cpp_dump")
    os.makedirs(dump_dir, exist_ok=True)

    # Check if already dumped (script.json exists)
    script_json = os.path.join(dump_dir, "script.json")
    if os.path.isfile(script_json):
        return dump_dir

    try:
        # Il2CppDumper uses Console.ReadKey() which throws when stdin is
        # redirected. We must NOT redirect stdin. Instead, we hide any
        # console window via STARTUPINFO (SW_HIDE) and use a timeout to
        # prevent hanging on "Press any key to exit" prompts.
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
            stderr = (
                proc.stderr.decode("utf-8", errors="replace") if proc.stderr else ""
            )
            print(
                f"WARNING: Il2CppDumper exited with code {proc.returncode}: {stderr[:200]}"
            )
            return None
    except subprocess.TimeoutExpired:
        print("WARNING: Il2CppDumper timed out")
        return None
    except Exception as e:
        print(f"WARNING: Il2CppDumper failed: {e}")
        return None

    if not os.path.isfile(script_json):
        print("WARNING: Il2CppDumper did not produce script.json")
        return None

    return dump_dir


def _create_generator(env, game_dir: str, data_dir: str, work_dir: str, tools_dir: str):
    """Create and initialize a TypeTreeGenerator for the game.

    Returns (generator, None) on success, (None, error_msg) on failure.
    """
    try:
        from UnityPy.helpers.TypeTreeGenerator import TypeTreeGenerator
    except ImportError:
        return None, "TypeTreeGenerator not available"

    # Get Unity version
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
        # IL2CPP: run Il2CppDumper, then load DummyDlls
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
        # Mono: load Managed DLLs
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


def _save_typetree_cache(work_dir: str, cache: dict):
    """Save typetree nodes cache to work_dir."""
    cache_path = os.path.join(work_dir, "typetree_cache.pkl")
    try:
        with open(cache_path, "wb") as f:
            pickle.dump(cache, f)
    except Exception:
        pass


def _get_script_info(obj, env) -> Optional[tuple]:
    """Extract (assembly, classname, namespace) from a MonoBehaviour's m_Script.

    Reads the first 28 bytes of raw data to get the m_Script PPtr, then
    looks up the MonoScript object.
    """
    from UnityPy.streams import EndianBinaryReader

    try:
        raw = obj.get_raw_data()
        reader = EndianBinaryReader(raw, endian="<")
        # m_GameObject PPtr: int32 + int64 = 12 bytes
        reader.read_int()
        reader.read_long()
        # m_Enabled: uint8 + align = 4 bytes
        reader.read_boolean()
        reader.align_stream()
        # m_Script PPtr: int32 + int64 = 12 bytes
        script_file = reader.read_int()
        script_path = reader.read_long()

        # Find the MonoScript object
        for o2 in env.objects:
            if o2.type.name == "MonoScript" and o2.path_id == script_path:
                ms = o2.read()
                return (
                    ms.m_AssemblyName or "",
                    ms.m_ClassName or "",
                    ms.m_Namespace or "",
                )
    except Exception:
        pass
    return None


def _get_reconstructed_nodes(
    gen, assembly: str, classname: str, namespace: str, cache: dict
):
    """Get reconstructed typetree nodes for a script class.

    Returns a TypeTreeNode or None.
    """
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
    """Read a MonoBehaviour's typetree using reconstructed nodes.

    Reads directly from the object's raw data to avoid positioning issues
    with the internal asset-file reader.
    """
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


# --- Raw-data fallback (stripped / corrupt typetrees) -----------------------

def _raw_scan_mb(obj, emit, file_name, path_id):
    """Read exact length-prefixed Unity strings when a typetree is unavailable."""
    try:
        raw = obj.get_raw_data()
        endian = getattr(obj.reader, "endian", "<")
    except Exception:
        return

    for offset, text in scan_serialized_strings(raw, endian):
        emit(
            file_name,
            path_id,
            f"rawstr_{offset:08x}",
            text,
            "MonoBehaviour raw serialized string (typetree unavailable)",
        )


def _is_engine_file(file_name: str) -> bool:
    """Return True for Unity engine-internal files (globalgamemanagers*).

    These files' MonoBehaviours hold Unity's own UI names ("Light Layer
    default", etc.), not the game's text — skip them in raw-scan mode.
    """
    base = os.path.basename(file_name or "").lower()
    return base.startswith("globalgamemanagers")


# --- Main -------------------------------------------------------------------


def main():
    if len(sys.argv) < 3:
        print("Usage: extract.py <game_dir> <work_dir> [tools_dir]")
        sys.exit(1)

    game_dir = sys.argv[1]
    work_dir = sys.argv[2]
    tools_dir = sys.argv[3] if len(sys.argv) > 3 else ""

    # Resolve data dir
    data_dir = find_data_dir(game_dir)
    if not data_dir:
        print(f"ERROR: Could not find a Unity *_Data directory under: {game_dir}")
        sys.exit(3)

    # Ensure work_dir is writable
    try:
        os.makedirs(work_dir, exist_ok=True)
        test_path = os.path.join(work_dir, ".farsiforge_write_test")
        with open(test_path, "w", encoding="utf-8") as f:
            f.write("ok")
        os.remove(test_path)
    except OSError as e:
        print(f"ERROR: work_dir is not writable ({work_dir}): {e}")
        sys.exit(4)

    results = []
    seen = set()  # (file, path_id, path) dedup

    stats = {
        "text_assets": 0,
        "mono_behaviours": 0,
        "skipped_no_typetree": 0,
        "reconstructed": 0,
        "raw_scanned": 0,
        "errors": 0,
    }

    def emit(file_name, path_id, path, source, context):
        key = (file_name, path_id, path)
        if key in seen:
            return
        seen.add(key)
        results.append(
            {
                "id": f"{file_name}_{path_id}_{path}",
                "source": source,
                "file": file_name,
                "path": path,
                "context": context,
                "status": "untranslated",
            }
        )

    # Load the whole data dir (UnityPy handles files/bundles)
    try:
        env = UnityPy.load(data_dir)
    except Exception as e:
        print(f"ERROR: UnityPy failed to load {data_dir}: {e}")
        stats["errors"] += 1
        _write_and_summary(work_dir, results, stats)
        return

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

            if type_name == "TextAsset":
                try:
                    data = obj.read()
                except Exception:
                    stats["errors"] += 1
                    continue
                raw = getattr(data, "script", None)
                if raw is None:
                    # Some versions expose .text instead
                    raw = getattr(data, "text", None) or ""
                if isinstance(raw, bytes):
                    text = raw.decode("utf-8", errors="ignore")
                else:
                    text = str(raw) if raw else ""
                text = text.strip()
                if not text:
                    continue
                stats["text_assets"] += 1
                # TextAsset: emit the whole blob as one entry (no truncation)
                emit(file_name, path_id, f"TextAsset/{data.name}", text, "TextAsset")

            elif type_name == "MonoBehaviour":
                stats["mono_behaviours"] += 1
                handled = False
                used_reconstruction = False

                # Strategy 1: typetree nodes available (not stripped)
                nodes = None
                try:
                    nodes = obj.serialized_type.nodes
                except Exception:
                    nodes = None

                if nodes:
                    tree = _try_read_typetree(obj)
                    if tree is not None:
                        _walk_value(tree, "root", emit, file_name, path_id)
                        handled = True

                # Strategy 2: try read_typetree anyway (some IL2CPP builds)
                if not handled:
                    tree = _try_read_typetree(obj)
                    if isinstance(tree, dict):
                        _walk_value(tree, "root", emit, file_name, path_id)
                        handled = True

                # Strategy 3: reconstruct typetree from script DLLs
                if not handled:
                    # Lazy-init the generator on first need
                    if gen is None and gen_error == "":
                        gen, gen_error = _create_generator(
                            env, game_dir, data_dir, work_dir, tools_dir
                        )
                        if gen is not None:
                            print(
                                f"TypeTreeGenerator initialized ({'IL2CPP' if _detect_il2cpp(game_dir, data_dir) else 'Mono'})"
                            )

                    if gen is not None:
                        # Get script info from m_Script PPtr
                        script_info = _get_script_info_fast(obj, mono_script_map)
                        if script_info:
                            assembly, classname, namespace = script_info
                            node = _get_reconstructed_nodes(
                                gen, assembly, classname, namespace, tt_cache
                            )
                            if node is not None:
                                tree = _read_with_reconstructed_nodes(obj, node)
                                if tree is not None:
                                    _walk_value(
                                        tree,
                                        "root",
                                        emit,
                                        file_name,
                                        path_id,
                                    )
                                    handled = True
                                    used_reconstruction = True

                # Strategy 4: obj.read() and walk attributes (last resort)
                if not handled:
                    robj = _try_read_object(obj)
                    if robj is not None:
                        _walk_value(robj, "root", emit, file_name, path_id)
                        handled = True

                # Strategy 5: raw-data string scan (stripped/corrupt typetrees)
                # The game's text is still recoverable from the raw
                # serialized bytes even when the typetree cannot be parsed.
                # Engine-internal files (globalgamemanagers*) are skipped:
                # their MonoBehaviours hold Unity UI names, not game text.
                if not handled and not _is_engine_file(file_name):
                    _raw_scan_mb(obj, emit, file_name, path_id)
                    handled = True
                    stats["raw_scanned"] += 1

                if not handled:
                    stats["skipped_no_typetree"] += 1
                elif used_reconstruction:
                    stats["reconstructed"] += 1

        except Exception:
            stats["errors"] += 1
            continue

    # Save typetree cache for inject.py and repeat runs
    _save_typetree_cache(work_dir, tt_cache)

    _write_and_summary(work_dir, results, stats)


def _get_script_info_fast(obj, mono_script_map: dict) -> Optional[tuple]:
    """Extract (assembly, classname, namespace) using a pre-built MonoScript map.

    This is faster than _get_script_info because it avoids re-iterating
    env.objects for each MonoBehaviour.
    """
    from UnityPy.streams import EndianBinaryReader

    try:
        raw = obj.get_raw_data()
        reader = EndianBinaryReader(raw, endian="<")
        # m_GameObject PPtr: int32 + int64 = 12 bytes
        reader.read_int()
        reader.read_long()
        # m_Enabled: uint8 + align = 4 bytes
        reader.read_boolean()
        reader.align_stream()
        # m_Script PPtr: int32 + int64 = 12 bytes
        script_file = reader.read_int()
        script_path = reader.read_long()

        if script_path in mono_script_map:
            return mono_script_map[script_path]
    except Exception:
        pass
    return None


def _write_and_summary(work_dir, results, stats):
    out_path = os.path.join(work_dir, "extracted.json")
    try:
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump(results, f, ensure_ascii=False, indent=2)
    except OSError as e:
        print(f"ERROR: could not write {out_path}: {e}")
        sys.exit(4)

    summary = {
        "extracted": len(results),
        "text_assets": stats["text_assets"],
        "mono_behaviours": stats["mono_behaviours"],
        "skipped_no_typetree": stats["skipped_no_typetree"],
        "reconstructed": stats["reconstructed"],
        "raw_scanned": stats["raw_scanned"],
        "errors": stats["errors"],
        "output": out_path,
    }
    print(f"Extracted {len(results)} items to {out_path}")
    print(f"SUMMARY: {json.dumps(summary, ensure_ascii=False)}")
    sys.exit(0)


if __name__ == "__main__":
    main()
