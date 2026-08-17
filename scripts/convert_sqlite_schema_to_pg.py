#!/usr/bin/env python3
"""Convert db/sqlite/schema.sql + seed.sql into PostgreSQL baseline files."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "db" / "sqlite" / "schema.sql"
SEED = ROOT / "db" / "sqlite" / "seed.sql"
OUT_SCHEMA = ROOT / "migrations" / "erp" / "schema.sql"
OUT_SEED = ROOT / "migrations" / "erp" / "data-dev.sql"


def convert_sql(src: str) -> str:
    src = re.sub(r"PRAGMA[^;]*;", "", src, flags=re.I)
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
    src = re.sub(
        r"INSERT\s+OR\s+IGNORE\s+INTO",
        "INSERT INTO",
        src,
        flags=re.I,
    )
    # naive: append ON CONFLICT DO NOTHING for simple seed inserts without it
    return src


def main() -> None:
    raw = SRC.read_text(encoding="utf-8")
    body = convert_sql(raw)
    header = """-- factory-erp PostgreSQL baseline v1.0.0
-- Converted from db/sqlite/schema.sql (formal R&D baseline)

CREATE TABLE IF NOT EXISTS erp_schema_migration (
  version     VARCHAR(32) PRIMARY KEY,
  description VARCHAR(255) NOT NULL DEFAULT '',
  applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  checksum    VARCHAR(64) NOT NULL DEFAULT ''
);

"""
    footer = """
INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.0', 'postgresql baseline', '')
ON CONFLICT (version) DO NOTHING;
"""
    OUT_SCHEMA.parent.mkdir(parents=True, exist_ok=True)
    OUT_SCHEMA.write_text(header + body + "\n" + footer, encoding="utf-8")
    print(f"wrote {OUT_SCHEMA} ({OUT_SCHEMA.stat().st_size} bytes)")

    if SEED.exists():
        seed = convert_sql(SEED.read_text(encoding="utf-8"))
        # Add ON CONFLICT DO NOTHING to INSERT INTO ... VALUES without conflict clause
        lines = []
        for stmt in re.split(r";\s*\n", seed):
            s = stmt.strip()
            if not s:
                continue
            if re.match(r"(?is)INSERT\s+INTO", s) and "ON CONFLICT" not in s.upper():
                s = s.rstrip(";") + "\nON CONFLICT DO NOTHING"
            lines.append(s + ";")
        OUT_SEED.write_text(
            "-- factory-erp PostgreSQL data-dev seed\n\n" + "\n\n".join(lines) + "\n",
            encoding="utf-8",
        )
        print(f"wrote {OUT_SEED} ({OUT_SEED.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
