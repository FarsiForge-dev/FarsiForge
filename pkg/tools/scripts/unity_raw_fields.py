"""Conservative reader/writer for Unity's length-prefixed serialized strings.

These helpers are only for MonoBehaviours whose typetree is unavailable. A
candidate must have a valid endian-aware byte length, strict UTF-8 payload,
and zero bytes through the Unity 4-byte alignment boundary. Offsets are local
to the object's raw data and become part of the project entry identity.
"""

import re
import struct

_MAX_STRING_BYTES = 64 * 1024
_MAX_RAW_MB_SCAN_BYTES = 2 * 1024 * 1024
_GUID_RE = re.compile(
    r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-"
    r"[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
)
_URL_RE = re.compile(r"^(https?://|www\.)", re.IGNORECASE)
_PATH_RE = re.compile(r"^[^\s]*[\\/][^\s]*\.[A-Za-z0-9]{1,8}$")
_NUMBER_RE = re.compile(r"^[+-]?\d+(\.\d+)?$")
_REPEAT_RE = re.compile(r"^(.)\1+$")
_ASSEMBLY_RE = re.compile(r"^[A-Za-z0-9_.]+, Assembly-[A-Za-z0-9]+$")
_UNITY_STYLE_RE = re.compile(
    r"(?:^|;)\s*(?:-unity-[\w-]+|display|visibility|overflow|background-color|"
    r"position|opacity|width|height|left|right|top|bottom)\s*:",
    re.IGNORECASE,
)
_REPEATED_PLACEHOLDER_RE = re.compile(r"^(?:newtext|testing|teststring|sampletext){3,}$", re.IGNORECASE)


def _looks_translatable(text):
    trimmed = text.strip()
    if len(trimmed) < 2 or _NUMBER_RE.match(trimmed):
        return False
    if _URL_RE.match(trimmed) or _PATH_RE.match(trimmed) or _GUID_RE.match(trimmed):
        return False
    if trimmed.startswith(("UnityEngine.", "System.", "TMPro.", "UnityEditor.")):
        return False
    if _REPEAT_RE.match(trimmed) or _ASSEMBLY_RE.match(trimmed):
        return False
    if any(ord(char) < 32 and char not in "\r\n\t" for char in trimmed):
        return False
    if "/" in trimmed or "\\" in trimmed:
        return False
    if ";" in trimmed and _UNITY_STYLE_RE.search(trimmed):
        return False
    if _REPEATED_PLACEHOLDER_RE.match(re.sub(r"\s+", "", trimmed)):
        return False
    if not any(char.isalpha() for char in trimmed):
        return False
    for unit_length in range(1, min(32, len(trimmed) // 3) + 1):
        if len(trimmed) % unit_length == 0:
            unit = trimmed[:unit_length]
            if len(trimmed) // unit_length >= 3 and unit * (len(trimmed) // unit_length) == trimmed:
                return False
    return True


def scan_serialized_strings(raw, endian="<"):
    """Return [(byte_offset, exact_text)] for verified Unity string fields."""
    if endian not in ("<", ">") or not raw:
        return []
    if len(raw) > _MAX_RAW_MB_SCAN_BYTES:
        return []

    fields = []
    covered_until = 0
    limit = len(raw) - 4
    for offset in range(0, limit + 1, 4):
        if offset < covered_until:
            continue
        length = struct.unpack_from(endian + "I", raw, offset)[0]
        if length < 2 or length > _MAX_STRING_BYTES:
            continue
        start = offset + 4
        end = start + length
        if end > len(raw):
            continue
        aligned_end = (end + 3) & ~3
        if aligned_end > len(raw) or any(raw[end:aligned_end]):
            continue
        try:
            text = raw[start:end].decode("utf-8", errors="strict")
        except UnicodeDecodeError:
            continue
        if not _looks_translatable(text):
            continue
        fields.append((offset, text))
        covered_until = aligned_end
    return fields


def replace_serialized_strings(raw, replacements, endian="<"):
    """Replace {offset: (expected_source, translation)} without shifting peers."""
    if not replacements:
        return raw
    if endian not in ("<", ">"):
        raise ValueError("unsupported Unity object byte order")

    fields = dict(scan_serialized_strings(raw, endian))
    ordered = sorted(replacements.items())
    result = bytearray()
    cursor = 0
    for offset, pair in ordered:
        expected, translation = pair
        if offset not in fields or fields[offset] != expected:
            raise ValueError(f"Unity serialized string source changed at byte offset {offset}")
        if offset < cursor:
            raise ValueError(f"overlapping Unity serialized string at byte offset {offset}")
        if "\x00" in translation:
            raise ValueError(f"translation contains a NUL byte at byte offset {offset}")
        encoded = translation.encode("utf-8", errors="strict")
        if len(encoded) > _MAX_STRING_BYTES:
            raise ValueError(f"translation is too large at byte offset {offset}")

        old_end = offset + 4 + len(expected.encode("utf-8"))
        old_aligned_end = (old_end + 3) & ~3
        result.extend(raw[cursor:offset])
        result.extend(struct.pack(endian + "I", len(encoded)))
        result.extend(encoded)
        new_end = offset + 4 + len(encoded)
        result.extend(b"\x00" * (((new_end + 3) & ~3) - new_end))
        cursor = old_aligned_end

    result.extend(raw[cursor:])
    return bytes(result)
