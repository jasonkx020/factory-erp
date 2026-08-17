#!/usr/bin/env python3
"""Append missing CREATE TABLE DDL from Go Ensure* sources into PG baseline."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "migrations" / "erp" / "schema.sql"
INTERNAL = ROOT / "internal"


def convert_ddl(src: str) -> str:
    src = re.sub(
        r"INTEGER\s+PRIMARY\s+KEY\s+AUTOINCREMENT",
        "BIGSERIAL PRIMARY KEY",
        src,
        flags=re.I,
    )
    src = re.sub(r"INTEGER\s+PRIMARY\s+KEY", "BIGSERIAL PRIMARY KEY", src, flags=re.I)
    src = re.sub(
        r"DEFAULT\s*\(\s*datetime\('now'\)\s*\)",
        "DEFAULT NOW()",
        src,
        flags=re.I,
    )
    src = re.sub(r"DEFAULT\s+datetime\('now'\)", "DEFAULT NOW()", src, flags=re.I)
    src = re.sub(r"datetime\('now'\)", "NOW()", src, flags=re.I)
    src = re.sub(r"\bREAL\b", "DOUBLE PRECISION", src)
    return src.strip()


def extract_creates(text: str) -> dict[str, str]:
    out: dict[str, str] = {}
    # raw string literals in Go: `CREATE TABLE ...`
    for m in re.finditer(r"`(CREATE TABLE IF NOT EXISTS\s+(\w+)[\s\S]*?)`", text, re.I):
        ddl, name = m.group(1), m.group(2).lower()
        # trim to closing paren of CREATE — naive: until first backtick already
        # stop at semicolon if present inside
        ddl = ddl.strip()
        if not ddl.endswith(";"):
            ddl = ddl + ";"
        out[name] = convert_ddl(ddl)
    # also double-quoted Exec(`...`) shorter forms without nested backticks already handled
    return out


def main() -> None:
    schema = SCHEMA.read_text(encoding="utf-8")
    have = {
        m.group(1).lower()
        for m in re.finditer(r"CREATE TABLE IF NOT EXISTS\s+(\w+)", schema, re.I)
    }
    found: dict[str, str] = {}
    for p in INTERNAL.rglob("*.go"):
        if "test" in p.name:
            continue
        text = p.read_text(encoding="utf-8", errors="ignore")
        for name, ddl in extract_creates(text).items():
            if name not in have:
                found[name] = ddl
    if not found:
        print("no missing tables")
        return
    # insert before baseline footer
    footer_mark = "INSERT INTO erp_schema_migration"
    idx = schema.rfind(footer_mark)
    if idx < 0:
        raise SystemExit("footer not found")
    extras = ["\n-- Tables from Ensure* (merged into baseline)\n"]
    for name in sorted(found):
        extras.append(found[name])
        extras.append("")
        print("add", name)
    new_schema = schema[:idx] + "\n".join(extras) + "\n" + schema[idx:]
    SCHEMA.write_text(new_schema, encoding="utf-8")
    print(f"added {len(found)} tables -> {SCHEMA}")


if __name__ == "__main__":
    main()
