#!/usr/bin/env python3
"""Append ADD COLUMN IF NOT EXISTS from Go Ensure* ALTER statements."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "migrations" / "erp" / "schema.sql"
INTERNAL = ROOT / "internal"

pat = re.compile(
    r"ALTER\s+TABLE\s+(\w+)\s+ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s+([^;`\"]+)",
    re.I,
)


def convert_type(t: str) -> str:
    t = t.strip().rstrip(",")
    t = re.sub(r"datetime\('now'\)", "NOW()", t, flags=re.I)
    t = re.sub(r"DEFAULT\s*\(\s*NOW\(\)\s*\)", "DEFAULT NOW()", t, flags=re.I)
    t = re.sub(r"\bREAL\b", "DOUBLE PRECISION", t)
    t = re.sub(r"\bAUTOINCREMENT\b", "", t, flags=re.I)
    return t.strip()


def main() -> None:
    schema = SCHEMA.read_text(encoding="utf-8")
    seen: set[tuple[str, str]] = set()
    alters: list[str] = []
    for p in INTERNAL.rglob("*.go"):
        text = p.read_text(encoding="utf-8", errors="ignore")
        for m in pat.finditer(text):
            table, col, typ = m.group(1).lower(), m.group(2).lower(), convert_type(m.group(3))
            key = (table, col)
            if key in seen:
                continue
            seen.add(key)
            # skip if column already appears in CREATE for that table (rough)
            if re.search(rf"CREATE TABLE IF NOT EXISTS\s+{table}\b[\s\S]*?\b{col}\b", schema, re.I):
                continue
            alters.append(f"ALTER TABLE {table} ADD COLUMN IF NOT EXISTS {col} {typ};")
    if not alters:
        print("no alters to add")
        return
    footer_mark = "INSERT INTO erp_schema_migration"
    idx = schema.rfind(footer_mark)
    block = "\n-- Columns from Ensure* ALTER ADD COLUMN\n" + "\n".join(sorted(alters)) + "\n\n"
    SCHEMA.write_text(schema[:idx] + block + schema[idx:], encoding="utf-8")
    print(f"added {len(alters)} ALTER COLUMN statements")


if __name__ == "__main__":
    main()
