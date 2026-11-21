#!/usr/bin/env python3
"""Comprehensive unit tests for lib/adopt-makefile."""

from __future__ import annotations

import os
import sys
import unittest
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
from pathlib import Path

os.environ["NO_COLOR"] = "1"

RETA_ROOT = Path(__file__).resolve().parent.parent
ADOPT_PATH = RETA_ROOT / "lib" / "adopt-makefile"

loader = SourceFileLoader("adopt_makefile", str(ADOPT_PATH))
spec = spec_from_loader("adopt_makefile", loader)
assert spec is not None
adopt = module_from_spec(spec)
loader.exec_module(adopt)


class TestDetectionHelpers(unittest.TestCase):
    def test_already_uses_reta_active_include(self) -> None:
        self.assertTrue(adopt.already_uses_reta("include $(RETA_BASE)\n"))
        self.assertTrue(adopt.already_uses_reta("include ~/.reta/lib/base.mk\n"))

    def test_already_uses_reta_ignores_comments(self) -> None:
        self.assertFalse(
            adopt.already_uses_reta("# include $(RETA_BASE)\nall:\n\techo hi\n")
        )
        self.assertFalse(adopt.already_uses_reta("APP := demo\nall:\n\techo hi\n"))

    def test_load_base_targets_from_repo(self) -> None:
        targets = adopt.load_base_targets(RETA_ROOT / "lib" / "base.mk")
        for expected in (
            "help",
            "up",
            "down",
            "logs",
            "clean",
            "clean-docker",
            "export",
            "git-sync",
            "check-secrets",
        ):
            self.assertIn(expected, targets)
        self.assertNotIn(".PHONY", targets)
        self.assertNotIn(".DEFAULT_GOAL", targets)


class TestSplitBlocks(unittest.TestCase):
    def test_preserves_multiline_variable_assignment(self) -> None:
        lines = [
            "SRCS := a.py \\",
            "        b.py \\",
            "        c.py",
            "build:",
            "\tpython $(SRCS)",
        ]
        blocks = adopt.split_blocks(lines)
        self.assertEqual(blocks[0][0], "directive")
        self.assertEqual(len(blocks[0][1]), 3)
        self.assertEqual(blocks[1][0], "target")
        self.assertEqual(blocks[1][1][1], "\tpython $(SRCS)")

    def test_preserves_nested_conditionals(self) -> None:
        lines = [
            "ifeq ($(ENV),prod)",
            "ifdef DEBUG",
            "FLAGS := -g",
            "endif",
            "else",
            "FLAGS := -O2",
            "endif",
            "run:",
            "\techo $(FLAGS)",
        ]
        blocks = adopt.split_blocks(lines)
        self.assertEqual(blocks[0][0], "directive")
        self.assertEqual(len(blocks[0][1]), 7)
        self.assertEqual(blocks[1][0], "target")

    def test_preserves_define_block_with_colons(self) -> None:
        lines = [
            "define HELPER_MSG",
            "usage: make run",
            "endef",
            "run:",
            "\techo ok",
        ]
        blocks = adopt.split_blocks(lines)
        self.assertEqual(blocks[0][0], "directive")
        self.assertEqual(len(blocks[0][1]), 3)
        self.assertEqual(blocks[1][0], "target")

    def test_keeps_blank_line_inside_multiline_recipe(self) -> None:
        lines = [
            "deploy:",
            "\techo step1",
            "",
            "\techo step2",
            "",
            "test:",
            "\tpytest",
        ]
        blocks = adopt.split_blocks(lines)
        target_blocks = [b for b in blocks if b[0] == "target"]
        self.assertEqual(len(target_blocks), 2)
        self.assertEqual(len(target_blocks[0][1]), 4)
        self.assertEqual(target_blocks[0][1][3], "\techo step2")

    def test_separates_dot_directives_from_targets(self) -> None:
        lines = [
            ".PHONY: all test",
            ".SILENT: test",
            "test:",
            "\tpytest",
        ]
        blocks = adopt.split_blocks(lines)
        self.assertEqual([b[0] for b in blocks], ["directive", "directive", "target"])


class TestExtractHelpDescriptions(unittest.TestCase):
    def test_extracts_echo_space_dash_colon_and_printf(self) -> None:
        lines = [
            "help:",
            '\t@echo "  make status   Show service status"',
            '\t@echo "  make lint - Run ruff checks"',
            '\t@echo "  make fmt : Format source files"',
            '\t@printf "  %-10s %s\\n" "build" "Build binary"',
            '\t@echo "Example: make train EPOCHS=30"',
        ]
        blocks = adopt.split_blocks(lines)
        descs = adopt.extract_help_descriptions(blocks)
        self.assertEqual(descs["status"], "Show service status")
        self.assertEqual(descs["lint"], "Run ruff checks")
        self.assertEqual(descs["fmt"], "Format source files")
        self.assertEqual(descs["build"], "Build binary")
        self.assertNotIn("train", descs)

    def test_extracts_without_make_prefix_for_known_targets(self) -> None:
        lines = [
            "help:",
            '\t@echo "  ── DOCKER ──"',
            '\t@echo "  n8n-publish       Publish all workflows"',
            '\t@echo "  check-workflows   Validate workflow JSONs"',
            '\t@echo "  cd ~/projects/demo"',
            "n8n-publish:",
            "\techo ok",
            "check-workflows:",
            "\techo ok",
        ]
        blocks = adopt.split_blocks(lines)
        descs = adopt.extract_help_descriptions(blocks)
        self.assertEqual(descs["n8n-publish"], "Publish all workflows")
        self.assertEqual(descs["check-workflows"], "Validate workflow JSONs")
        self.assertNotIn("cd", descs)


class TestClassifyHeading(unittest.TestCase):
    def _classify_all(self, lines: list[str]) -> list[tuple[str, str, str | None]]:
        blocks = adopt.split_blocks(lines)
        out = []
        for idx, (kind, chunk) in enumerate(blocks):
            if kind == "comment":
                status, title = adopt.classify_heading(blocks, idx)
                out.append((chunk[0], status, title))
        return out

    def test_decorated_and_banner_headings(self) -> None:
        lines = [
            "# === DATA MODEL ===",
            "a:",
            "",
            "# --- DOCKER ---",
            "b:",
            "",
            "# *** QUALITY ***",
            "c:",
            "",
            "# ### TESTING ###",
            "d:",
            "",
            "# ====================",
            "# PIPELINE",
            "# ====================",
            "e:",
        ]
        res = self._classify_all(lines)
        headings = [(s, t) for _, s, t in res if s == "heading"]
        self.assertEqual(
            headings,
            [
                ("heading", "DATA MODEL"),
                ("heading", "DOCKER"),
                ("heading", "QUALITY"),
                ("heading", "TESTING"),
                ("heading", "PIPELINE"),
            ],
        )

    def test_unicode_banner_and_mixed_case_long_heading(self) -> None:
        lines = [
            "# ═══════════════════════════════════════════════════════════",
            "# N8N WORKFLOWS — Export / Import (exports/ klasoru)",
            "# ═══════════════════════════════════════════════════════════",
            "export-workflows:",
            "\techo ok",
        ]
        res = self._classify_all(lines)
        headings = [(s, t) for _, s, t in res if s == "heading"]
        self.assertEqual(
            headings,
            [("heading", "N8N WORKFLOWS — Export / Import (exports/ klasoru)")],
        )

    def test_uppercase_heading_above_target_vs_trailing_uncertain(self) -> None:
        lines = [
            "# MIGRATIONS",
            "migrate:",
            "\talembic upgrade head",
            "",
            "# TRAILING HEADING",
        ]
        res = self._classify_all(lines)
        self.assertEqual(res[0][1:], ("heading", "MIGRATIONS"))
        self.assertEqual(res[1][1:], ("uncertain", "TRAILING HEADING"))

    def test_half_decorated_and_title_case_are_uncertain(self) -> None:
        lines = [
            "# --- Broken Header",
            "",
            "# Database Tasks",
            "migrate:",
            "\techo ok",
            "",
            "# This is a normal explanatory sentence.",
            "seed:",
            "\techo ok",
            "",
            "# APP_ENV=PRODUCTION",
            "run:",
            "\techo ok",
        ]
        res = self._classify_all(lines)
        by_text = {raw: (status, title) for raw, status, title in res}
        self.assertEqual(by_text["# --- Broken Header"], ("uncertain", "Broken Header"))
        self.assertEqual(by_text["# Database Tasks"], ("uncertain", "Database Tasks"))
        self.assertEqual(
            by_text["# This is a normal explanatory sentence."], ("normal", None)
        )
        self.assertEqual(by_text["# APP_ENV=PRODUCTION"], ("normal", None))


class TestConvertTargetHeader(unittest.TestCase):
    def test_keeps_existing_double_hash(self) -> None:
        status, line = adopt.convert_target_header("test: deps  ## Existing desc", {})
        self.assertEqual(status, "existing")
        self.assertEqual(line, "test: deps  ## Existing desc")

    def test_upgrades_single_hash_and_preserves_deps(self) -> None:
        status, line = adopt.convert_target_header(
            "setup: install # Prepare environment", {}
        )
        self.assertEqual(status, "annotated")
        self.assertEqual(line, "setup: install  ## Prepare environment")

    def test_injects_from_help_dict(self) -> None:
        status, line = adopt.convert_target_header(
            "migrate:", {"migrate": "Run migrations"}
        )
        self.assertEqual(status, "annotated")
        self.assertEqual(line, "migrate:  ## Run migrations")

    def test_adds_placeholder_when_missing(self) -> None:
        status, line = adopt.convert_target_header("deploy:", {})
        self.assertEqual(status, "placeholder")
        self.assertEqual(line, "deploy:  ## EDIT_HERE: add description")


class TestConvertMakefileEndToEnd(unittest.TestCase):
    def setUp(self) -> None:
        self.base_targets = adopt.load_base_targets(RETA_ROOT / "lib" / "base.mk")
        self.header = adopt.load_template_header(RETA_ROOT / "lib")

    def test_comprehensive_conversion(self) -> None:
        raw = "\n".join(
            [
                "# === Project Header ===",
                ".DEFAULT_GOAL := help",
                "SHELL := /bin/bash",
                "SHELL_CUSTOM := /bin/zsh",
                "APP := demo",
                "",
                "ifeq ($(ENV),prod)",
                "FLAGS := -O2",
                "else",
                "FLAGS := -O0",
                "endif",
                "",
                "# ====================",
                "# HELP",
                "# ====================",
                "help:",
                '\t@echo "  make status   Check status"',
                '\t@printf "  %-10s %s\\n" "build" "Build binary"',
                "",
                "status:",
                "\t@echo $(APP) $(FLAGS)",
                "",
                "prep:",
                "\t@echo prep",
                "",
                "# --- DATA ---",
                "migrate: # Run DB migrations",
                "\talembic upgrade head",
                "",
                "seed:  ## Seed database",
                "\tpython seed.py",
                "",
                "## DOCKER",
                "up:",
                "\tdocker compose up -d",
                "",
                "clean:",
                "\trm -rf dist",
                "",
                "# --- Half Heading",
                "",
                "# Database Tasks",
                "build:",
                "\tgo build ./...",
                "",
                "# TRAILING SECTION",
            ]
        )

        converted, report = adopt.convert_makefile(raw, self.base_targets, self.header)

        # Must be recognized as reta-enabled afterward
        self.assertTrue(adopt.already_uses_reta(converted))

        # Top header with no targets beneath it must be demoted from '# === ... ==='
        self.assertIn("# Project Header\n", converted)
        self.assertNotIn("# === Project Header ===", converted)

        # Redundant defaults commented out, custom variables preserved
        self.assertIn("# .DEFAULT_GOAL := help", converted)
        self.assertIn("# SHELL := /bin/bash", converted)
        self.assertIn("SHELL_CUSTOM := /bin/zsh", converted)
        self.assertIn("ifeq ($(ENV),prod)", converted)

        # HELP section had only commented 'help:' target -> demoted to '# HELP'
        self.assertIn("# HELP\n", converted)
        self.assertNotIn("# === HELP ===", converted)
        self.assertIn("# help:", converted)

        # DOCKER section had only colliding targets ('up', 'clean') -> demoted to '# DOCKER'
        self.assertIn("# DOCKER\n", converted)
        self.assertNotIn("# === DOCKER ===", converted)
        self.assertIn("# up:", converted)
        self.assertIn("# clean:", converted)
        self.assertIn("# \tdocker compose up -d", converted)

        # Active DATA section kept with normalized heading
        self.assertIn("# === DATA ===", converted)
        self.assertIn("DATA", report["headings"])
        self.assertNotIn("HELP", report["headings"])
        self.assertNotIn("DOCKER", report["headings"])

        # Target annotations
        self.assertIn("status:  ## Check status", converted)
        self.assertIn("prep:  ## EDIT_HERE: add description", converted)
        self.assertIn("migrate:  ## Run DB migrations", converted)
        self.assertIn("seed:  ## Seed database", converted)
        self.assertIn("build:  ## Build binary", converted)

        # Tabs preserved in recipes
        self.assertIn("\talembic upgrade head", converted)
        self.assertIn("\tgo build ./...", converted)

        # Uncertain headings flagged
        self.assertIn("# EDIT_HERE: verify if 'Half Heading'", converted)
        self.assertIn("# EDIT_HERE: verify if 'Database Tasks'", converted)
        self.assertIn("# EDIT_HERE: verify if 'TRAILING SECTION'", converted)

        # Report assertions
        self.assertCountEqual(report["commented_targets"], ["help", "up", "clean"])
        self.assertCountEqual(
            report["annotated_targets"], ["status", "migrate", "build"]
        )
        self.assertEqual(report["hidden_targets"], ["prep"])

    def test_preserves_non_default_shell_and_default_goal(self) -> None:
        raw = "\n".join(
            [
                "SHELL := /bin/zsh",
                ".DEFAULT_GOAL := build",
                "build:",
                "\techo ok",
            ]
        )
        converted, _ = adopt.convert_makefile(raw, self.base_targets, self.header)
        self.assertIn("\nSHELL := /bin/zsh\n", converted)
        self.assertIn("\n.DEFAULT_GOAL := build\n", converted)

    def test_collapses_excessive_blank_lines(self) -> None:
        raw = "APP := x\n\n\n\n\nbuild:\n\techo ok\n"
        converted, _ = adopt.convert_makefile(raw, self.base_targets, self.header)
        self.assertNotIn("\n\n\n\n", converted)

    def test_multi_target_line_with_collision_is_commented(self) -> None:
        raw = "up start:\n\techo starting\n"
        converted, report = adopt.convert_makefile(raw, self.base_targets, self.header)
        self.assertIn("# up start:", converted)
        self.assertIn("up", report["commented_targets"])


if __name__ == "__main__":
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(TestDetectionHelpers)
    for cls in (
        TestSplitBlocks,
        TestExtractHelpDescriptions,
        TestClassifyHeading,
        TestConvertTargetHeader,
        TestConvertMakefileEndToEnd,
    ):
        suite.addTests(unittest.defaultTestLoader.loadTestsFromTestCase(cls))
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    sys.exit(0 if result.wasSuccessful() else 1)
