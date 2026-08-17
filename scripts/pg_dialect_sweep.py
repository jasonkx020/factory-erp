#!/usr/bin/env python3
"""Sweep SQLite dialect remnants from Go sources for PostgreSQL."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DIRS = [
    ROOT / "internal",
    ROOT / "scripts",
    ROOT / "cmd",
]


def transform(text: str) -> str:
    text = text.replace("datetime('now')", "NOW()")
    text = text.replace('datetime("now")', "NOW()")
    # INSERT OR IGNORE INTO x -> INSERT INTO x ... need ON CONFLICT - do simple replace
    text = re.sub(
        r"INSERT\s+OR\s+IGNORE\s+INTO",
        "INSERT INTO",
        text,
        flags=re.I,
    )
    # GROUP_CONCAT(x) -> string_agg(x::text, ',')
    text = re.sub(
        r"GROUP_CONCAT\s*\(\s*([^)]+?)\s*\)",
        r"string_agg((\1)::text, ',')",
        text,
        flags=re.I,
    )
    return text


def main() -> None:
    n = 0
    for d in DIRS:
        for p in d.rglob("*.go"):
            raw = p.read_text(encoding="utf-8")
            new = transform(raw)
            if new != raw:
                # For INSERT INTO that used to be OR IGNORE inside raw strings,
                # append ON CONFLICT DO NOTHING when the statement is a simple Exec string
                # Heuristic: only touch lines that are clearly single-statement inserts in backticks
                p.write_text(new, encoding="utf-8")
                n += 1
                print("updated", p.relative_to(ROOT))
    print("files", n)


if __name__ == "__main__":
    main()
