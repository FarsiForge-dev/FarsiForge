"""Tests for Unity extraction filtering, structured TextAsset parsing, and safe injection."""

import os
import sys
import json
import unittest

# Add scripts directory to sys.path
SCRIPT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "scripts"))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

from extract import is_noise, should_extract_string, _classify_semantic, _extract_text_asset
from inject import _inject_strings


class TestUnityExtractInject(unittest.TestCase):
    def test_noise_filtering(self):
        # Version strings
        self.assertTrue(is_noise("2021.3.15f1"))
        self.assertTrue(is_noise("1.0.4"))
        self.assertTrue(is_noise("v1.2.0"))

        # GUIDs & hex hashes
        self.assertTrue(is_noise("8d123456-789a-bcde-f012-3456789abcde"))
        self.assertTrue(is_noise("0123456789abcdef0123456789abcdef"))

        # Paths and asset filenames
        self.assertTrue(is_noise("Assets/UI/MainMenu.prefab"))
        self.assertTrue(is_noise("Textures/hero.png"))
        self.assertTrue(is_noise("hero.mat"))
        self.assertTrue(is_noise("custom_shader.shader"))

        # Unity internal classes
        self.assertTrue(is_noise("GameObject"))
        self.assertTrue(is_noise("Transform"))
        self.assertTrue(is_noise("RectTransform"))
        self.assertTrue(is_noise("Canvas"))
        self.assertTrue(is_noise("MonoBehaviour"))

        # Numbers and pure symbols
        self.assertTrue(is_noise("123"))
        self.assertTrue(is_noise("-"))
        self.assertTrue(is_noise(":::"))

        # Legitimate short UI
        self.assertFalse(is_noise("OK"))
        self.assertFalse(is_noise("Yes"))
        self.assertFalse(is_noise("No"))
        self.assertFalse(is_noise("Exit"))
        self.assertFalse(is_noise("Play"))
        self.assertFalse(is_noise("Back"))

        # Real natural text
        self.assertFalse(is_noise("Press Start"))
        self.assertFalse(is_noise("Hello, traveler! The kingdom is in grave danger."))

    def test_should_extract_string(self):
        # GameObject m_Name must NEVER be extracted
        self.assertFalse(should_extract_string("Main Camera", "m_Name", "root.m_Name"))
        self.assertFalse(should_extract_string("Directional Light", "m_Name", "root.m_Name"))
        self.assertFalse(should_extract_string("Canvas", "m_Name", "root.m_Name"))
        self.assertFalse(should_extract_string("Score Text", "m_Name", "root.m_Name"))

        # m_CreationSettings and character sequences
        self.assertFalse(should_extract_string(
            "abcdefghijklmnopqrstuvwxyz",
            "characterSequence",
            "root.m_CreationSettings.characterSequence"
        ))

        # Serialized version
        self.assertFalse(should_extract_string("2021.3.15f1", "m_Version", "root.m_Version"))

        # Real translatable UI and dialogue
        self.assertTrue(should_extract_string("Start Game", "m_Text", "root.m_Text"))
        self.assertTrue(should_extract_string("Hello traveler!", "text", "root.dialogueLines[0].text"))
        self.assertTrue(should_extract_string("Iron Sword", "name", "root.inventoryItems[0].name"))

    def test_semantic_classification(self):
        # Dialogue
        cat = _classify_semantic("Hello, traveler! Where are you going?", "dialogueLines[0].text", "MonoBehaviour")
        self.assertEqual(cat, "dialogue")

        # UI
        cat = _classify_semantic("Start Game", "root.m_Text", "MonoBehaviour")
        self.assertEqual(cat, "ui")
        cat = _classify_semantic("OK", "button.label", "MonoBehaviour")
        self.assertEqual(cat, "ui")

        # Item
        cat = _classify_semantic("Healing Potion", "inventory.items[0].name", "MonoBehaviour")
        self.assertEqual(cat, "item")

        # Quest
        cat = _classify_semantic("Defeat the Dragon", "quest.objective", "MonoBehaviour")
        self.assertEqual(cat, "quest")

    def test_structured_text_asset_json(self):
        class MockTextAsset:
            name = "localization_en"
            text = json.dumps({
                "menu": {
                    "start": "Start Game",
                    "options": "Options",
                    "quit": "Quit"
                },
                "dialogues": [
                    {"speaker": "Elder", "text": "Welcome, hero!"}
                ]
            })

        emitted = []
        def emit(file_name, path_id, path, source, context, **kwargs):
            emitted.append((path, source, kwargs.get("category"), kwargs.get("read_only"), kwargs.get("source_type")))

        stats = {"text_assets": 0, "errors": 0}
        _extract_text_asset(MockTextAsset(), "resources.assets", 100, emit, stats)

        self.assertEqual(stats["text_assets"], 1)
        # Verify discrete records were emitted with exact paths
        paths = [e[0] for e in emitted]
        self.assertIn("TextAsset/localization_en#menu.start", paths)
        self.assertIn("TextAsset/localization_en#menu.options", paths)
        self.assertIn("TextAsset/localization_en#menu.quit", paths)
        self.assertIn("TextAsset/localization_en#dialogues[0].text", paths)

        # Verify not marked read-only and categorized correctly
        for e in emitted:
            self.assertFalse(e[3], f"Entry {e[0]} should be writable")
            self.assertEqual(e[4], "textasset-json")

    def test_structured_text_asset_binary(self):
        class MockBinaryTextAsset:
            name = "data_bin"
            script = b"\x00\x01\x02\x03\xff\xfe"

        emitted = []
        def emit(file_name, path_id, path, source, context, **kwargs):
            emitted.append((path, source, kwargs.get("read_only"), kwargs.get("candidate_class")))

        stats = {"text_assets": 0, "errors": 0}
        _extract_text_asset(MockBinaryTextAsset(), "resources.assets", 101, emit, stats)

        self.assertEqual(len(emitted), 1)
        self.assertTrue(emitted[0][2], "Binary TextAsset must be read-only")
        self.assertEqual(emitted[0][3], "raw-heuristic")

    def test_inject_strings_source_verification(self):
        tree = {
            "title": "Old Title",
            "prompt": "Press Any Key"
        }
        candidate_ids = ["res_100"]
        translations = {
            "res_100_root.title": "عنوان جدید",
            "res_100_root.prompt": "کلیدی را فشار دهید",
        }
        entries_by_id = {
            # Matching source
            "res_100_root.title": {"source": "Old Title", "read_only": False},
            # Stale source mismatch (expected 'Click Here', but tree has 'Press Any Key')
            "res_100_root.prompt": {"source": "Click Here", "read_only": False},
        }
        pending = []
        stats = {"errors": 0, "skipped_no_typetree": 0}

        new_tree = _inject_strings(tree, "root", candidate_ids, translations, entries_by_id, pending, stats)

        # Title matched -> replaced!
        self.assertEqual(new_tree["title"], "عنوان جدید")
        self.assertIn("res_100_root.title", pending)

        # Prompt mismatched source -> refused replacement, kept original!
        self.assertEqual(new_tree["prompt"], "Press Any Key")
        self.assertNotIn("res_100_root.prompt", pending)
        self.assertEqual(stats["errors"], 1)


if __name__ == "__main__":
    unittest.main()
