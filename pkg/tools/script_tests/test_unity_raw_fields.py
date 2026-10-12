import struct
import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "scripts"))
from unity_raw_fields import replace_serialized_strings, scan_serialized_strings


def _serialized_string(text, endian="<"):
    encoded = text.encode("utf-8")
    result = struct.pack(endian + "I", len(encoded)) + encoded
    return result + b"\x00" * ((-len(result)) % 4)


class UnityRawFieldsTests(unittest.TestCase):
    def test_scans_exact_multiline_and_cjk_fields_with_offsets(self):
        raw = b"HEAD" + _serialized_string("  First line\nsecond line  ")
        raw += _serialized_string("新しい設定")

        fields = scan_serialized_strings(raw)

        second_offset = 4 + len(_serialized_string("  First line\nsecond line  "))
        self.assertEqual(fields, [(4, "  First line\nsecond line  "), (second_offset, "新しい設定")])

    def test_replaces_variable_length_values_and_keeps_neighbor_fields(self):
        raw = b"HEAD" + _serialized_string("Welcome") + _serialized_string("Settings") + b"TAIL"
        fields = scan_serialized_strings(raw)
        first_offset, second_offset = fields[0][0], fields[1][0]

        updated = replace_serialized_strings(
            raw,
            {
                first_offset: ("Welcome", "خوش آمدید"),
                second_offset: ("Settings", "تنظیمات"),
            },
        )

        next_offset = 4 + len(_serialized_string("خوش آمدید"))
        self.assertEqual(scan_serialized_strings(updated), [(4, "خوش آمدید"), (next_offset, "تنظیمات")])
        self.assertTrue(updated.endswith(b"TAIL"))

    def test_rejects_changed_source_and_invalid_layout(self):
        raw = b"HEAD" + _serialized_string("Start Game")
        offset = scan_serialized_strings(raw)[0][0]

        with self.assertRaisesRegex(ValueError, "source changed"):
            replace_serialized_strings(raw, {offset: ("Changed", "شروع")})

        malformed = b"HEAD" + struct.pack("<I", 9) + b"Not padded"
        self.assertEqual(scan_serialized_strings(malformed), [])

    def test_filters_unity_style_declarations(self):
        raw = b"HEAD" + _serialized_string(
            "left: 23%; opacity: 0.73; display: flex; background-color: rgb(255, 255, 255);"
        )
        self.assertEqual(scan_serialized_strings(raw), [])

    def test_filters_repeated_editor_placeholders(self):
        raw = b"HEAD" + _serialized_string("TestingTestingTesting\nTestingTesting")
        raw += _serialized_string("New TextNew TextNew Text")
        self.assertEqual(scan_serialized_strings(raw), [])


if __name__ == "__main__":
    unittest.main()
