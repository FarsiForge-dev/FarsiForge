#!/usr/bin/env python3
"""FarsiForge Unity extraction script — Memory-Optimized & Streamlined.

Usage: python extract.py <game_dir> <work_dir> [tools_dir]

Walks a Unity game's *_Data directory, extracts translatable strings from
TextAsset and MonoBehaviour objects via UnityPy, filters noise, deduplicates,
and writes <work_dir>/extracted.json as an array of core.StringEntry-shaped
objects. Prints a machine-readable SUMMARY line as the final line of output.

Memory Architecture:
- Scans candidate serialized files (.assets, level*, bundles) individually.
- Never loads .resS, .resource or raw media buffers into memory.
- Calls gc.collect() between files to prevent memory leaks and out-of-memory hangs on 5GB-20GB games.
"""

import os
import re
import sys
import gc
import json
import pickle
import subprocess
from typing import Optional, List, Dict, Tuple, Set

# --- UnityPy import guard ---------------------------------------------------
try:
    import UnityPy
except ImportError:
    print("ERROR: UnityPy is not installed. Run: pip install UnityPy")
    sys.exit(2)

_script_dir = os.path.dirname(os.path.abspath(__file__))
if _script_dir not in sys.path:
    sys.path.insert(0, _script_dir)

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
_VERSION_RE = re.compile(r"^v?\d+\.\d+(\.\d+)?[a-zA-Z0-9_\-\.]*$", re.IGNORECASE)
_HEX_RUN_RE = re.compile(r"^[0-9a-fA-F]{32,}$")
_ASSET_PATH_RE = re.compile(
    r"^[a-zA-Z0-9_\-/\\]+\.(prefab|asset|mat|shader|dll|exe|png|jpg|jpeg|tga|dds|wav|ogg|mp3|unity3d|bytes|txt|json)$",
    re.IGNORECASE,
)

# Single-token identifier patterns (code identifiers, not human text)
_PASCAL_RE = re.compile(r"^[A-Z][a-z0-9]+[A-Z][a-zA-Z0-9]*$")
_CAMEL_RE = re.compile(r"^[a-z]+[A-Z][a-zA-Z0-9]*$")
_SNAKE_RE = re.compile(r"^[a-zA-Z0-9]+_[a-zA-Z0-9_]+$")

# Field names that indicate human-readable text content
_TEXT_FIELD_RE = re.compile(
    r"(text|label|title|description|message|tooltip|string|dialogue|subtitle|content|prompt|caption|story|quest|lore|ui|item|comm|name|speech|talk|body|cell)",
    re.IGNORECASE,
)

_MNAME_RE = re.compile(r"^m_Name$", re.IGNORECASE)

_EXCLUDED_FIELDS = {
    "m_name", "m_tagstring", "m_script", "m_version", "m_objecthideflags",
    "m_editorhideflags", "m_font", "m_material", "m_texture", "m_shader",
    "m_shaderkeywords", "m_guid", "guid", "m_path", "path", "m_fileid",
    "charactersequence", "m_charactersequence", "m_rootorder", "m_localscale",
    "m_localposition", "m_localrotation", "m_anchormin", "m_anchormax",
    "m_anchoredposition", "m_sizeoffset", "m_pivot", "m_layermask", "m_lightmapindex",
}

_EXCLUDED_SUBSTRINGS = (
    "m_creationsettings",
    "charactersettings",
    "charactersequence",
)

_COMMON_UNITY_CLASSES = {
    "gameobject", "transform", "recttransform", "canvas", "camera", "light",
    "audiosource", "monobehaviour", "eventsystem", "standaloneinputmodule",
    "graphicraycaster", "canvasscaler", "boxcollider", "meshrenderer", "meshfilter",
    "rigidbody", "rigidbody2d", "boxcollider2d", "circlecollider2d",
}

_STANDARD_SHORT_UI = {
    "ok", "yes", "no", "up", "go", "hp", "mp", "xp", "sp", "lv", "on", "in", "to", "at", "by", "or", "ui"
}

_STANDARD_UI_KEYWORDS = {
    "ok", "yes", "no", "cancel", "apply", "save", "load", "back", "exit", "quit",
    "start", "continue", "retry", "play", "pause", "resume", "options", "settings",
    "audio", "video", "graphics", "controls", "gameplay", "language", "help",
    "credits", "inventory", "map", "quest", "quests", "items", "skills", "stats",
    "profile", "delete", "edit", "new", "next", "previous", "accept", "decline",
    "confirm", "close", "open", "buy", "sell", "equip", "unequip", "use", "discard",
    "craft", "upgrade", "select", "clear", "reset", "mute", "unmute", "default",
    "fullscreen", "windowed", "vsync", "brightness", "gamma", "volume", "master",
    "music", "sfx", "voice", "difficulty", "easy", "normal", "hard", "nightmare",
    "new game", "load game", "save game", "start game", "main menu", "restart", "abandon",
}


def is_noise(s: str) -> bool:
    """Return True if a string should be filtered out as non-translatable."""
    if s is None:
        return True
    s = s.strip()
    if len(s) < 2:
        return True
    lower = s.lower()
    if lower in _STANDARD_SHORT_UI or lower in _STANDARD_UI_KEYWORDS:
        return False
    if _NUM_RE.match(s):
        return True
    if _VERSION_RE.match(s):
        return True
    if _URL_RE.match(s):
        return True
    if _PATH_RE.match(s) or _ASSET_PATH_RE.match(s):
        return True
    if _GUID_RE.match(s) or _HEX_RUN_RE.match(s):
        return True
    if _NAMESPACE_RE.match(s):
        return True
    if _REPEAT_RE.match(s):
        return True
    if lower in _COMMON_UNITY_CLASSES:
        return True
    if not any(c.isalpha() for c in s):
        return True
    return False


def is_code_identifier(s: str) -> bool:
    """Return True if a string looks like a code identifier."""
    s = s.strip()
    if " " in s:
        return False
    if _PASCAL_RE.match(s) or _CAMEL_RE.match(s) or _SNAKE_RE.match(s):
        return True
    return False


def should_extract_string(s: str, field_name: str = "", path: str = "") -> bool:
    """Decide whether a string should be extracted as translatable."""
    if is_noise(s):
        return False
    path_lower = path.lower() if path else ""
    field_lower = field_name.lower() if field_name else ""
    trimmed_lower = s.strip().lower()

    if field_lower in _EXCLUDED_FIELDS:
        return False
    if any(sub in path_lower for sub in _EXCLUDED_SUBSTRINGS):
        return False
    if path_lower.endswith(".m_name") or path_lower == "root.m_name":
        return False

    # Standard UI vocabulary words are always legitimate translatable strings!
    if trimmed_lower in _STANDARD_UI_KEYWORDS or trimmed_lower in _STANDARD_SHORT_UI:
        return True

    if is_code_identifier(s):
        if field_name and _MNAME_RE.match(field_name):
            return False
        if field_name and _TEXT_FIELD_RE.search(field_name):
            return True
        return False

    return True


def _classify_semantic(text: str, path: str, context: str) -> str:
    """Classify semantic category based on structural context and text traits."""
    meta = f"{path} {context}".lower()
    text_lower = text.lower().strip()

    # 1. Dialogue evidence
    if any(kw in meta for kw in ("dialogue", "dialog", "subtitle", "speech", "talk", "voice", "chat", "npc", "conversation", "manga", "cutscene", "story", "scenario", "actor", "timeline")):
        return "dialogue"
    has_pronoun = any(p in text_lower for p in ("you ", "you're", "i ", "i'm", "we ", "we're", "my ", "your ", "don't", "can't", "please"))
    has_punct = any(c in text for c in ("?", "!", '"', "“", "”", "…"))
    words = len(text.split())
    if has_pronoun and has_punct and words >= 3:
        return "dialogue"

    # 2. UI evidence
    if any(kw in meta for kw in ("menu", "ui", "btn", "button", "title", "label", "hud", "prompt", "window", "tooltip", "popup", "m_text")):
        return "ui"
    if text_lower in _STANDARD_UI_KEYWORDS or text_lower in _STANDARD_SHORT_UI:
        return "ui"

    # 3. Item evidence
    if any(kw in meta for kw in ("item", "weapon", "armor", "inventory", "gear", "skill", "spell", "potion", "perk", "artifact")):
        return "item"

    # 4. Quest evidence
    if any(kw in meta for kw in ("quest", "mission", "objective", "task", "journal", "bounty", "log")):
        return "quest"

    # 5. System evidence
    if any(kw in meta for kw in ("system", "network", "server", "error", "warning", "notice", "notification")):
        return "system"

    return "unclassified"


# --- Data dir discovery & candidate filtering --------------------------------

def _has_assets_files(d: str) -> bool:
    try:
        for name in os.listdir(d):
            nl = name.lower()
            if nl.endswith(".assets") and not nl.endswith(".resS"):
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
        # Skip temporary directories
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

            # Only accept files known to store Unity serialized objects/bundles
            if (
                fl.endswith(".assets")
                or fl.endswith(".bundle")
                or fl.endswith(".unity3d")
                or fl.startswith("level")
                or fl.startswith("sharedassets")
                or fl.startswith("globalgamemanagers")
            ):
                # Filter out giant compressed bundles (>250MB) and non-text media bundles (characters, animations, shaders, audio)
                if fl.endswith(".bundle") or fl.endswith(".unity3d"):
                    try:
                        sz = os.path.getsize(full_path)
                        if sz > 250 * 1024 * 1024:
                            continue
                    except OSError:
                        continue

                    skip_kw = (
                        "character",
                        "animator",
                        "animation",
                        "texture",
                        "shader",
                        "sound",
                        "audio",
                        "music",
                        "video",
                        "movie",
                        "mesh",
                        "model",
                        "terrain",
                        "environment",
                        "lighting",
                        "lightmap",
                        "particle",
                        "vfx",
                    )
                    if any(kw in fl for kw in skip_kw):
                        continue

                candidates.append((full_path, rel_path))

    # Sort files: prioritize globalgamemanagers and resources so MonoScripts are parsed early
    def sort_key(item):
        path_lower = item[1].lower()
        if "globalgamemanagers" in path_lower:
            return 0
        if "resources.assets" in path_lower:
            return 1
        if "sharedassets" in path_lower:
            return 2
        return 3

    candidates.sort(key=sort_key)
    return candidates


# --- Typetree / object walking ----------------------------------------------

def _walk_value(value, path, emit, file_name, path_id, depth=0, max_depth=10, source_type="unity-typetree"):
    """Recursively walk a parsed value, emitting string entries with recursion guards."""
    if depth > max_depth:
        return

    if isinstance(value, str):
        field_name = ""
        if "." in path:
            field_name = path.rsplit(".", 1)[-1]
        elif "/" in path:
            field_name = path.rsplit("/", 1)[-1]
        field_name = re.sub(r"\[\d+\]$", "", field_name)
        if should_extract_string(value, field_name, path):
            cat = _classify_semantic(value, path, "MonoBehaviour")
            emit(
                file_name,
                path_id,
                path,
                value,
                "MonoBehaviour",
                category=cat,
                candidate_class="high-confidence",
                confidence=0.90,
                read_only=False,
                source_type=source_type,
            )
    elif isinstance(value, dict):
        for k, v in value.items():
            if not isinstance(k, str) or k.startswith("_"):
                continue
            _walk_value(v, f"{path}.{k}", emit, file_name, path_id, depth + 1, max_depth, source_type)
    elif isinstance(value, list):
        # Cap list iteration to avoid memory thrashing on massive numeric vertex/geometry buffers
        for i, v in enumerate(value[:250]):
            _walk_value(v, f"{path}[{i}]", emit, file_name, path_id, depth + 1, max_depth, source_type)
    elif value is not None and not isinstance(value, (int, float, bool, bytes)):
        try:
            attrs = vars(value)
        except TypeError:
            return
        for k, v in attrs.items():
            if k.startswith("_"):
                continue
            _walk_value(v, f"{path}.{k}", emit, file_name, path_id, depth + 1, max_depth, source_type)


def _try_read_typetree(obj) -> Optional[dict]:
    byte_size = getattr(obj, "byte_size", 0)
    if byte_size > 5 * 1024 * 1024:
        return None
    try:
        return obj.read_typetree()
    except Exception:
        return None


def _try_read_object(obj):
    byte_size = getattr(obj, "byte_size", 0)
    if byte_size > 5 * 1024 * 1024:
        return None
    try:
        return obj.read()
    except Exception:
        return None


# --- TypeTreeGenerator helpers ----------------------------------------------

def _detect_il2cpp(game_dir: str, data_dir: str) -> bool:
    game_assembly = os.path.join(game_dir, "GameAssembly.dll")
    metadata = os.path.join(data_dir, "il2cpp_data", "Metadata", "global-metadata.dat")
    return os.path.isfile(game_assembly) and os.path.isfile(metadata)


def _run_il2cpp_dumper(
    game_dir: str, data_dir: str, work_dir: str, tools_dir: str
) -> Optional[str]:
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
        print("WARNING: Il2CppDumper.exe not found — IL2CPP typetrees unavailable")
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
            startupinfo.dwFlags |= 0x00000001
            startupinfo.wShowWindow = 0

        proc = subprocess.run(
            [dumper_exe, game_assembly, metadata, dump_dir],
            stdin=None,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            creationflags=creationflags,
            startupinfo=startupinfo,
            timeout=180,
        )
        if proc.returncode != 0:
            return None
    except Exception:
        return None

    if not os.path.isfile(script_json):
        return None

    return dump_dir


def _create_generator(env, game_dir: str, data_dir: str, work_dir: str, tools_dir: str):
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
    cache_path = os.path.join(work_dir, "typetree_cache.pkl")
    if os.path.isfile(cache_path):
        try:
            with open(cache_path, "rb") as f:
                return pickle.load(f)
        except Exception:
            pass
    return {}


def _save_typetree_cache(work_dir: str, cache: dict):
    cache_path = os.path.join(work_dir, "typetree_cache.pkl")
    try:
        with open(cache_path, "wb") as f:
            pickle.dump(cache, f)
    except Exception:
        pass


def _get_reconstructed_nodes(gen, assembly: str, classname: str, namespace: str, cache: dict):
    if not assembly or not classname:
        return None
    as_lower = assembly.lower()
    if "editor" in as_lower or as_lower.startswith(("unityengine", "unityeditor", "system", "testrunner")):
        return None

    cache_key = (assembly, classname, namespace)
    if cache_key in cache:
        return cache[cache_key]

    fullname = classname
    if namespace:
        fullname = f"{namespace}.{classname}"

    try:
        node = gen.get_nodes_up(assembly, fullname)
        if len(cache) > 500:
            # Bound cache size to prevent memory leaks on large games
            for k in list(cache.keys())[:100]:
                del cache[k]
        cache[cache_key] = node
        return node
    except Exception:
        cache[cache_key] = None
        return None


def _read_with_reconstructed_nodes(obj, node) -> Optional[dict]:
    byte_size = getattr(obj, "byte_size", 0)
    if byte_size > 5 * 1024 * 1024:
        return None

    from UnityPy.streams import EndianBinaryReader
    from UnityPy.helpers.TypeTreeHelper import read_typetree

    try:
        raw = obj.get_raw_data()
        if not raw or len(raw) > 5 * 1024 * 1024:
            return None
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


def _extract_text_asset(data, file_name: str, path_id: int, emit, stats):
    asset_name = getattr(data, "m_Name", getattr(data, "name", "TextAsset")) or "TextAsset"
    raw = getattr(data, "m_Script", getattr(data, "script", getattr(data, "text", None)))
    if raw is None:
        raw = b""

    # Detect binary or non-UTF-8 payload
    is_binary = False
    if isinstance(raw, (bytes, bytearray, memoryview)):
        is_binary = b"\x00" in raw[:1024]
        text = bytes(raw).decode("utf-8", errors="ignore")
    else:
        text = str(raw) if raw else ""
        is_binary = "\x00" in text[:1024] or any(0xD800 <= ord(c) <= 0xDFFF for c in text[:100])

    if is_binary:
        stats["errors"] += 1
        emit(
            file_name,
            path_id,
            f"TextAsset/{asset_name}",
            "<binary TextAsset>",
            "TextAsset/Binary",
            category="unclassified",
            candidate_class="raw-heuristic",
            confidence=0.10,
            read_only=True,
            read_only_reason="Binary payload in TextAsset; unparsed and read-only",
            source_type="textasset-binary",
        )
        return

    text = text.strip()
    if not text:
        return

    # Check 1: Structured JSON TextAsset
    if text.startswith(("{", "[")) and text.endswith(("}", "]")):
        try:
            parsed = json.loads(text)
            extracted_count = 0

            def _walk_json(val, jpath):
                nonlocal extracted_count
                if isinstance(val, str):
                    fname = jpath.rsplit(".", 1)[-1] if "." in jpath else jpath
                    if should_extract_string(val, fname, jpath):
                        cat = _classify_semantic(val, jpath, "TextAsset/JSON")
                        emit(
                            file_name,
                            path_id,
                            f"TextAsset/{asset_name}#jpath:{jpath}",
                            val,
                            "TextAsset/JSON",
                            category=cat,
                            candidate_class="high-confidence",
                            confidence=0.95,
                            read_only=False,
                            source_type="textasset-json",
                        )
                        extracted_count += 1
                elif isinstance(val, dict):
                    for k, v in val.items():
                        if isinstance(k, str):
                            _walk_json(v, f"{jpath}.{k}" if jpath else k)
                elif isinstance(val, list):
                    for idx, item in enumerate(val):
                        _walk_json(item, f"{jpath}[{idx}]")

            _walk_json(parsed, "")
            if extracted_count > 0:
                stats["text_assets"] += 1
                return
        except Exception:
            pass

    # Check 2: Structured CSV / TSV TextAsset
    lines = text.splitlines()
    if len(lines) >= 2:
        delimiter = "\t" if "\t" in lines[0] else ("," if "," in lines[0] else None)
        if delimiter:
            try:
                import csv as _csv
                reader = list(_csv.reader(lines, delimiter=delimiter))
                if len(reader) >= 2 and len(reader[0]) >= 2:
                    header = reader[0]
                    text_cols = []
                    for c_idx, h in enumerate(header):
                        h_low = h.lower()
                        if any(kw in h_low for kw in ("text", "desc", "english", "en", "source", "dialogue", "name", "value", "trans", "msg", "content", "string", "label", "caption", "title", "ui", "sub", "line", "speech", "talk", "body", "item")):
                            text_cols.append(c_idx)
                    if not text_cols:
                        text_cols = list(range(1, len(header)))

                    extracted_count = 0
                    for r_idx in range(1, len(reader)):
                        row = reader[r_idx]
                        for c_idx in text_cols:
                            if c_idx < len(row):
                                cell_val = row[c_idx]
                                col_name = header[c_idx] if c_idx < len(header) else str(c_idx)
                                if should_extract_string(cell_val, col_name, f"{asset_name}#{col_name}"):
                                    cat = _classify_semantic(cell_val, f"{asset_name} {col_name}", "TextAsset/CSV")
                                    emit(
                                        file_name,
                                        path_id,
                                        f"TextAsset/{asset_name}#row[{r_idx}].col[{c_idx}]",
                                        cell_val,
                                        "TextAsset/CSV",
                                        category=cat,
                                        candidate_class="high-confidence",
                                        confidence=0.95,
                                        read_only=False,
                                        source_type="textasset-csv",
                                    )
                                    extracted_count += 1
                    if extracted_count > 0:
                        stats["text_assets"] += 1
                        return
            except Exception:
                pass

    # Check 3: Line-based Key-Value (.ini, .strings, .txt)
    kv_pattern = re.compile(r"^([a-zA-Z0-9_\-\.]+)\s*[=:]\s*(.+)$")
    matched_kv = []
    for line in lines:
        m = kv_pattern.match(line.strip())
        if m:
            matched_kv.append(m.groups())
    if len(matched_kv) >= 3 and len(matched_kv) >= len(lines) * 0.4:
        extracted_count = 0
        for k, v in matched_kv:
            if should_extract_string(v, k, f"kv_{k}"):
                cat = _classify_semantic(v, k, "TextAsset/KeyValue")
                emit(
                    file_name,
                    path_id,
                    f"TextAsset/{asset_name}#{k}",
                    v,
                    "TextAsset/KeyValue",
                    category=cat,
                    candidate_class="high-confidence",
                    confidence=0.95,
                    read_only=False,
                    source_type="textasset-kv",
                )
                extracted_count += 1
        if extracted_count > 0:
            stats["text_assets"] += 1
            return

    # Check 4: Plain Text / Narrative
    if should_extract_string(text, asset_name, f"TextAsset/{asset_name}"):
        cat = _classify_semantic(text, asset_name, "TextAsset")
        emit(
            file_name,
            path_id,
            f"TextAsset/{asset_name}",
            text,
            "TextAsset",
            category=cat,
            candidate_class="high-confidence",
            confidence=0.80,
            read_only=False,
            source_type="textasset-plain",
        )
        stats["text_assets"] += 1


def _raw_scan_mb(obj, emit, file_name, path_id):
    byte_size = getattr(obj, "byte_size", 0)
    if byte_size > 2 * 1024 * 1024:
        return

    try:
        raw = obj.get_raw_data()
        if not raw or len(raw) > 2 * 1024 * 1024:
            return
        endian = getattr(obj.reader, "endian", "<")
    except Exception:
        return

    for offset, text in scan_serialized_strings(raw, endian):
        cat = _classify_semantic(text, f"rawstr_{offset:08x}", "MonoBehaviour raw")
        emit(
            file_name,
            path_id,
            f"rawstr_{offset:08x}",
            text,
            "MonoBehaviour raw serialized string (typetree unavailable)",
            category=cat,
            candidate_class="raw-heuristic",
            confidence=0.50,
            read_only=True,
            read_only_reason="Raw byte scan without verified AST schema; read-only",
            source_type="monobehaviour-bytes",
        )


def _is_engine_file(file_name: str) -> bool:
    base = os.path.basename(file_name or "").lower()
    return base.startswith("globalgamemanagers")


def _get_script_info_fast(obj, mono_script_map: dict) -> Optional[tuple]:
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


# --- Main -------------------------------------------------------------------

def main():
    if len(sys.argv) < 3:
        print("Usage: extract.py <game_dir> <work_dir> [tools_dir]")
        sys.exit(1)

    game_dir = sys.argv[1]
    work_dir = sys.argv[2]
    tools_dir = sys.argv[3] if len(sys.argv) > 3 else ""

    data_dir = find_data_dir(game_dir)
    if not data_dir:
        print(f"ERROR: Could not find a Unity *_Data directory under: {game_dir}")
        sys.exit(3)

    try:
        os.makedirs(work_dir, exist_ok=True)
        test_path = os.path.join(work_dir, ".farsiforge_write_test")
        with open(test_path, "w", encoding="utf-8") as f:
            f.write("ok")
        os.remove(test_path)
    except OSError as e:
        print(f"ERROR: work_dir is not writable ({work_dir}): {e}")
        sys.exit(4)

    out_path = os.path.join(work_dir, "extracted.json")
    try:
        out_f = open(out_path, "w", encoding="utf-8")
        out_f.write("[\n")
    except OSError as e:
        print(f"ERROR: could not create output file {out_path}: {e}")
        sys.exit(4)

    is_first_entry = True
    extracted_count = 0
    seen_hashes = set()

    stats = {
        "text_assets": 0,
        "mono_behaviours": 0,
        "skipped_no_typetree": 0,
        "reconstructed": 0,
        "raw_scanned": 0,
        "errors": 0,
    }

    def emit(
        file_name,
        path_id,
        path,
        source,
        context,
        category="unclassified",
        candidate_class="high-confidence",
        confidence=0.80,
        read_only=False,
        read_only_reason="",
        source_type="unity-typetree",
    ):
        nonlocal is_first_entry, extracted_count
        key_hash = hash((file_name, path_id, path))
        if key_hash in seen_hashes:
            return
        seen_hashes.add(key_hash)
        entry = {
            "id": f"{file_name}_{path_id}_{path}",
            "source": source,
            "file": file_name,
            "path": path,
            "context": context,
            "status": "untranslated",
            "category": category,
            "candidate_class": candidate_class,
            "confidence": confidence,
            "read_only": read_only,
            "read_only_reason": read_only_reason,
            "source_type": source_type,
        }
        prefix = "  " if is_first_entry else ",\n  "
        out_f.write(prefix + json.dumps(entry, ensure_ascii=False))
        is_first_entry = False
        extracted_count += 1

    # 1. Discover candidate serialized files
    candidate_files = find_candidate_asset_files(data_dir)
    print(f"Found {len(candidate_files)} candidate Unity asset file(s) for extraction")

    # Global lookup of MonoScript objects across files
    mono_script_map: Dict[int, Tuple[str, str, str]] = {}
    gen = None
    gen_error = ""
    tt_cache = _load_typetree_cache(work_dir)

    # 2. Process file-by-file with active memory cleanup
    for idx, (full_path, rel_path) in enumerate(candidate_files, 1):
        try:
            env = UnityPy.load(full_path)
        except Exception as e:
            # Skip unreadable or corrupted single files gracefully
            continue

        # Cache any MonoScripts found in this file
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

        # Process TextAssets and MonoBehaviours
        obj_count = 0
        for obj in env.objects:
            obj_count += 1
            if obj_count % 200 == 0:
                gc.collect()

            try:
                type_name = obj.type.name
                if type_name not in ("TextAsset", "MonoBehaviour"):
                    continue

                file_name = rel_path
                path_id = obj.path_id

                if type_name == "TextAsset":
                    try:
                        data = obj.read()
                    except Exception:
                        stats["errors"] += 1
                        continue
                    _extract_text_asset(data, file_name, path_id, emit, stats)

                elif type_name == "MonoBehaviour":
                    stats["mono_behaviours"] += 1
                    handled = False
                    used_reconstruction = False

                    # Strategy 1: typetree nodes available
                    nodes = None
                    try:
                        nodes = obj.serialized_type.nodes
                    except Exception:
                        nodes = None

                    if nodes:
                        tree = _try_read_typetree(obj)
                        if tree is not None:
                            _walk_value(tree, "root", emit, file_name, path_id, source_type="unity-typetree")
                            handled = True

                    # Strategy 2: read_typetree direct
                    if not handled:
                        tree = _try_read_typetree(obj)
                        if isinstance(tree, dict):
                            _walk_value(tree, "root", emit, file_name, path_id, source_type="unity-typetree")
                            handled = True

                    # Strategy 3: TypeTreeGenerator reconstruction
                    if not handled:
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
                                        _walk_value(
                                            tree,
                                            "root",
                                            emit,
                                            file_name,
                                            path_id,
                                            source_type="unity-reconstructed",
                                        )
                                        handled = True
                                        used_reconstruction = True

                    # Strategy 4: obj.read() fallback
                    if not handled:
                        robj = _try_read_object(obj)
                        if robj is not None:
                            _walk_value(robj, "root", emit, file_name, path_id, source_type="unity-typetree")
                            handled = True

                    # Strategy 5: raw serialized string scan
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

        # Immediately free the loaded asset file and break cyclic references
        try:
            if hasattr(env, "files") and isinstance(env.files, dict):
                env.files.clear()
            if hasattr(env, "objects") and isinstance(env.objects, dict):
                env.objects.clear()
        except Exception:
            pass
        del env
        gc.collect()

    # Final cleanup & save
    out_f.write("\n]\n")
    out_f.close()

    _save_typetree_cache(work_dir, tt_cache)

    summary = {
        "extracted": extracted_count,
        "text_assets": stats["text_assets"],
        "mono_behaviours": stats["mono_behaviours"],
        "skipped_no_typetree": stats["skipped_no_typetree"],
        "reconstructed": stats["reconstructed"],
        "raw_scanned": stats["raw_scanned"],
        "errors": stats["errors"],
        "output": out_path,
    }
    print(f"Extracted {extracted_count} items to {out_path}")
    print(f"SUMMARY: {json.dumps(summary, ensure_ascii=False)}")
    sys.exit(0)


if __name__ == "__main__":
    main()
