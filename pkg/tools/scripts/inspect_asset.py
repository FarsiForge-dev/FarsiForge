#!/usr/bin/env python3
"""FarsiForge Unity compiled asset inspector.

Usage: python inspect_asset.py <asset_file_path> [page] [page_size]

Loads a single Unity asset file (.assets, .bundle) and inspects its serialized
objects table with pagination and memory safety. Emits JSON describing object
identities, Unity types, text-bearing fields, and exact property paths.
"""

import os
import sys
import json
import gc
from typing import List, Dict, Any, Optional

try:
    import UnityPy
except ImportError:
    print(json.dumps({"error": "UnityPy is not installed"}))
    sys.exit(2)

script_dir = os.path.dirname(os.path.abspath(__file__))
if script_dir not in sys.path:
    sys.path.insert(0, script_dir)

from unity_raw_fields import scan_serialized_strings

_EXCLUDED_FIELDS = {
    "m_name", "m_tagstring", "m_script", "m_version", "m_objecthideflags",
    "m_editorhideflags", "m_font", "m_material", "m_texture", "m_shader",
    "m_shaderkeywords", "m_guid", "guid", "m_path", "path", "m_fileid",
    "charactersequence", "m_charactersequence", "m_rootorder",
}


def _walk_tree_for_text(val, path: str, results: List[Dict[str, Any]], max_items: int = 30):
    if len(results) >= max_items:
        return
    if isinstance(val, str):
        trimmed = val.strip()
        field_name = path.rsplit(".", 1)[-1].lower() if "." in path else path.lower()
        if field_name in _EXCLUDED_FIELDS or "m_creationsettings" in path.lower():
            return
        if len(trimmed) >= 2 and any(c.isalpha() for c in trimmed):
            results.append({
                "path": path,
                "field_name": field_name,
                "value": trimmed[:200],
                "can_inject": True,
                "read_only": False,
            })
    elif isinstance(val, dict):
        for k, v in val.items():
            if not isinstance(k, str) or k.startswith("_"):
                continue
            _walk_tree_for_text(v, f"{path}.{k}" if path else k, results, max_items)
    elif isinstance(val, list):
        for i, item in enumerate(val[:50]):
            _walk_tree_for_text(item, f"{path}[{i}]", results, max_items)


def inspect_file(file_path: str, page: int = 1, page_size: int = 50) -> Dict[str, Any]:
    if not os.path.isfile(file_path):
        return {"error": f"File not found: {file_path}"}

    try:
        env = UnityPy.load(file_path)
    except Exception as e:
        return {"error": f"Failed to load Unity asset file: {e}"}

    objects = list(env.objects)
    total_objects = len(objects)

    page = max(1, page)
    page_size = max(1, min(page_size, 200))
    start_idx = (page - 1) * page_size
    end_idx = min(start_idx + page_size, total_objects)
    has_more = end_idx < total_objects

    slice_objects = objects[start_idx:end_idx]
    out_objects = []

    for obj in slice_objects:
        path_id = obj.path_id
        type_name = obj.type.name
        byte_size = getattr(obj, "byte_size", 0)

        obj_info = {
            "path_id": path_id,
            "type": type_name,
            "name": f"{type_name}_{path_id}",
            "size_bytes": byte_size,
            "text_bearing": False,
            "parsing_status": "parsed",
            "can_inject": False,
            "read_only": True,
            "read_only_reason": "",
            "properties": [],
        }

        if type_name == "TextAsset":
            obj_info["text_bearing"] = True
            obj_info["can_inject"] = True
            obj_info["read_only"] = False
            try:
                data = obj.read()
                obj_info["name"] = getattr(data, "m_Name", getattr(data, "name", "")) or f"TextAsset_{path_id}"
                raw = getattr(data, "m_Script", getattr(data, "script", getattr(data, "text", None)))
                if raw is None:
                    raw = b""
                if isinstance(raw, bytes):
                    text = raw.decode("utf-8", errors="ignore")
                else:
                    text = str(raw) if raw else ""
                
                text_trimmed = text.strip()
                if text_trimmed.startswith(("{", "[")):
                    obj_info["format"] = "json"
                    try:
                        parsed = json.loads(text_trimmed)
                        json_props = []
                        _walk_tree_for_text(parsed, "json", json_props, max_items=20)
                        obj_info["properties"] = json_props
                    except Exception:
                        pass
                elif "\n" in text_trimmed and ("," in text_trimmed or "\t" in text_trimmed):
                    obj_info["format"] = "csv"
                    sample_lines = text_trimmed.splitlines()[:5]
                    obj_info["properties"] = [{"path": f"row_{i}", "value": line[:150], "can_inject": True} for i, line in enumerate(sample_lines)]
                else:
                    obj_info["format"] = "text"
                    obj_info["properties"] = [{
                        "path": f"TextAsset/{obj_info['name']}",
                        "value": text_trimmed[:200],
                        "can_inject": True,
                        "read_only": False,
                    }]
            except Exception as e:
                obj_info["can_inject"] = False
                obj_info["read_only"] = True
                obj_info["read_only_reason"] = f"Failed to read TextAsset: {e}"

        elif type_name == "Font":
            obj_info["text_bearing"] = False
            obj_info["can_inject"] = True
            obj_info["read_only"] = False
            try:
                data = obj.read()
                font_name = getattr(data, "m_Name", "") or getattr(data, "name", "") or f"Font_{path_id}"
                obj_info["name"] = font_name
                obj_info["format"] = "font"
                font_size = len(getattr(data, "m_FontData", b"")) if hasattr(data, "m_FontData") else 0
                obj_info["properties"] = [
                    {
                        "path": "m_FontData",
                        "field_name": "m_FontData",
                        "value": f"Font Data ({font_size:,} bytes TTF/OTF)",
                        "can_inject": True,
                        "read_only": False,
                    },
                    {
                        "path": "m_FontSize",
                        "field_name": "m_FontSize",
                        "value": str(getattr(data, "m_FontSize", 0)),
                        "can_inject": False,
                        "read_only": True,
                    }
                ]
            except Exception as e:
                obj_info["can_inject"] = False
                obj_info["read_only"] = True
                obj_info["read_only_reason"] = f"Failed to read Font: {e}"

        elif type_name == "MonoBehaviour":
            tree = None
            try:
                tree = obj.read_typetree()
            except Exception:
                tree = None

            if isinstance(tree, dict):
                obj_info["name"] = tree.get("m_Name") or f"MonoBehaviour_{path_id}"
                props = []
                _walk_tree_for_text(tree, "root", props, max_items=25)
                if props:
                    obj_info["text_bearing"] = True
                    obj_info["can_inject"] = True
                    obj_info["read_only"] = False
                    obj_info["properties"] = props
                else:
                    obj_info["can_inject"] = True
                    obj_info["read_only"] = False
                    obj_info["read_only_reason"] = "No text properties found in typetree"
            else:
                # Raw byte scan fallback
                try:
                    raw = obj.get_raw_data()
                    endian = getattr(obj.reader, "endian", "<")
                    raw_strings = scan_serialized_strings(raw, endian)
                    if raw_strings:
                        obj_info["text_bearing"] = True
                        obj_info["can_inject"] = False
                        obj_info["read_only"] = True
                        obj_info["read_only_reason"] = "MonoBehaviour without typetree (heuristic raw byte scan)"
                        obj_info["properties"] = [
                            {
                                "path": f"rawstr_{offset:08x}",
                                "field_name": f"offset_0x{offset:08x}",
                                "value": text[:150],
                                "can_inject": False,
                                "read_only": True,
                            }
                            for offset, text in raw_strings[:15]
                        ]
                    else:
                        obj_info["can_inject"] = False
                        obj_info["read_only"] = True
                        obj_info["read_only_reason"] = "Typetree unavailable; no serialized strings detected"
                except Exception as e:
                    obj_info["can_inject"] = False
                    obj_info["read_only"] = True
                    obj_info["read_only_reason"] = f"Unparsed MonoBehaviour: {e}"
        else:
            try:
                data = obj.read()
                obj_info["name"] = getattr(data, "name", "") or getattr(data, "m_Name", "") or f"{type_name}_{path_id}"
            except Exception:
                pass
            obj_info["read_only_reason"] = f"Unity {type_name} does not contain translatable text fields"

        out_objects.append(obj_info)

    # Clean up
    del env
    gc.collect()

    return {
        "file_path": file_path,
        "total_objects": total_objects,
        "page": page,
        "page_size": page_size,
        "has_more": has_more,
        "objects": out_objects,
    }


def scan_unity_fonts(game_dir: str) -> List[Dict[str, Any]]:
    results = []
    ignore_suffixes = (".ress", ".resource", ".dll", ".exe", ".pdb", ".png", ".jpg", ".wav", ".ogg", ".mp3")
    for root, dirs, files in os.walk(game_dir):
        dirs[:] = [d for d in dirs if not d.startswith(".farsiforge") and not d.startswith("WorkingDir")]
        for f in files:
            fl = f.lower()
            if fl.endswith(".ress") or fl.endswith(".resource") or any(fl.endswith(s) for s in ignore_suffixes):
                continue
            if fl.endswith(".assets") or fl.endswith(".bundle") or fl.endswith(".unity3d"):
                full_path = os.path.join(root, f)
                try:
                    sz = os.path.getsize(full_path)
                    # Skip giant scene geometry/mesh files (> 500MB) that are not resources.assets
                    if sz > 500 * 1024 * 1024 and "resources" not in fl:
                        continue

                    env = UnityPy.load(full_path)
                    # Instant check: does any file in env have Font (class_id 128)?
                    has_font = False
                    for sf in env.files.values():
                        if hasattr(sf, "types") and any(getattr(t, "class_id", None) == 128 for t in sf.types):
                            has_font = True
                            break
                    if not has_font:
                        del env
                        continue

                    for obj in env.objects:
                        if obj.type.name == "Font":
                            try:
                                d = obj.read()
                                font_name = getattr(d, "m_Name", "") or getattr(d, "name", "") or f"Font_{obj.path_id}"
                                font_size = len(getattr(d, "m_FontData", b"")) if hasattr(d, "m_FontData") else 0
                                rel = os.path.relpath(full_path, game_dir).replace("\\", "/")
                                results.append({
                                    "path": f"{rel}#{obj.path_id}",
                                    "file": rel,
                                    "path_id": obj.path_id,
                                    "name": font_name,
                                    "size_bytes": font_size,
                                    "format": "TTF/OTF",
                                    "can_replace": True,
                                    "can_export": True,
                                })
                            except Exception:
                                pass
                    del env
                    gc.collect()
                except Exception:
                    pass
    return results


def export_unity_font(asset_file: str, path_id: int, output_file: str) -> bool:
    if not os.path.isfile(asset_file):
        return False
    env = UnityPy.load(asset_file)
    for obj in env.objects:
        if obj.path_id == path_id and obj.type.name == "Font":
            try:
                data = obj.read()
                font_bytes = getattr(data, "m_FontData", None)
                if font_bytes:
                    os.makedirs(os.path.dirname(os.path.abspath(output_file)), exist_ok=True)
                    with open(output_file, "wb") as f:
                        f.write(bytes(font_bytes))
                    return True
            except Exception:
                return False
    return False


def replace_unity_font(asset_file: str, path_id: int, new_font_file: str, output_asset_file: str) -> bool:
    if not os.path.isfile(asset_file) or not os.path.isfile(new_font_file):
        return False
    env = UnityPy.load(asset_file)
    with open(new_font_file, "rb") as f:
        new_bytes = f.read()

    found = False
    for obj in env.objects:
        if obj.path_id == path_id and obj.type.name == "Font":
            try:
                data = obj.read()
                data.m_FontData = list(new_bytes)
                data.save()
                found = True
                break
            except Exception:
                return False
    if not found:
        return False

    os.makedirs(os.path.dirname(os.path.abspath(output_asset_file)), exist_ok=True)
    with open(output_asset_file, "wb") as f:
        f.write(env.file.save())
    return True


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: inspect_asset.py <asset_file_path> [page] [page_size] | --scan-fonts <game_dir> | --export-font <asset> <path_id> <out> | --replace-font <asset> <path_id> <font> <out>"}))
        sys.exit(1)

    if sys.argv[1] == "--scan-fonts":
        game_dir = sys.argv[2] if len(sys.argv) > 2 else "."
        fonts = scan_unity_fonts(game_dir)
        print(json.dumps(fonts, ensure_ascii=False))
        return

    if sys.argv[1] == "--export-font":
        if len(sys.argv) < 5:
            print(json.dumps({"error": "Usage: --export-font <asset_file> <path_id> <output_file>"}))
            sys.exit(1)
        ok = export_unity_font(sys.argv[2], int(sys.argv[3]), sys.argv[4])
        print(json.dumps({"success": ok}))
        return

    if sys.argv[1] == "--replace-font":
        if len(sys.argv) < 6:
            print(json.dumps({"error": "Usage: --replace-font <asset_file> <path_id> <new_font_file> <output_asset_file>"}))
            sys.exit(1)
        ok = replace_unity_font(sys.argv[2], int(sys.argv[3]), sys.argv[4], sys.argv[5])
        print(json.dumps({"success": ok}))
        return

    file_path = sys.argv[1]
    page = int(sys.argv[2]) if len(sys.argv) > 2 else 1
    page_size = int(sys.argv[3]) if len(sys.argv) > 3 else 50

    result = inspect_file(file_path, page, page_size)
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()

