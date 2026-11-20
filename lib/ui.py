"""Shared UI constants (colors) for reta Python scripts.

Colors are disabled automatically when:
    - stdout is not a TTY (piped/redirected)
    - NO_COLOR environment variable is set
"""

from __future__ import annotations

import os
import sys
from pathlib import Path


def _use_color() -> bool:
    if os.environ.get("NO_COLOR") is not None:
        return False
    return hasattr(sys.stdout, "isatty") and sys.stdout.isatty()


if _use_color():
    BOLD = "\033[1m"
    DIM = "\033[2m"
    CYAN = "\033[36m"
    GREEN = "\033[32m"
    YELLOW = "\033[1;33m"
    MAGENTA = "\033[1;35m"
    RED = "\033[31m"
    RESET = "\033[0m"
else:
    BOLD = DIM = CYAN = GREEN = YELLOW = MAGENTA = RED = RESET = ""


def _hints_enabled() -> bool:
    value = os.environ.get("RETA_HINTS", "").lower()
    if value in ("off", "false", "0", "no"):
        return False
    config = Path.home() / ".config" / "reta" / "config"
    if config.exists():
        try:
            for line in config.read_text(encoding="utf-8").splitlines():
                if line.strip() == "hints=off":
                    return False
        except OSError:
            pass
    return True


def next_steps(steps: list[str]) -> None:
    if not steps or not _hints_enabled():
        return
    print(f"\n{BOLD}Next steps:{RESET}")
    for step in steps:
        print(f"  → {step}")
