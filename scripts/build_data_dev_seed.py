#!/usr/bin/env python3
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "db" / "sqlite" / "seed.sql"
OUT = ROOT / "migrations" / "erp" / "data-dev.sql"


def main() -> None:
    seed_src = SRC.read_text(encoding="utf-8")
    seed = re.sub(r"INSERT\s+OR\s+IGNORE\s+INTO", "INSERT INTO", seed_src, flags=re.I)
    parts: list[str] = []
    for stmt in re.split(r";\s*\n", seed):
        s = stmt.strip()
        if not s:
            continue
        if re.match(r"(?is)INSERT\s+INTO", s) and "ON CONFLICT" not in s.upper():
            s = s.rstrip(";") + "\nON CONFLICT DO NOTHING"
        parts.append(s + ";")
    tables = sorted(
        set(
            re.findall(
                r"INSERT INTO (\w+)\s*\([^)]*\bid\b",
                "\n".join(parts),
                flags=re.I,
            )
        )
    )
    setvals = ["-- Reset sequences after explicit id inserts"]
    for t in tables:
        setvals.append(
            f"SELECT setval(pg_get_serial_sequence('{t}', 'id'), "
            f"COALESCE((SELECT MAX(id) FROM {t}), 1));"
        )
    out = (
        "-- factory-erp PostgreSQL data-dev seed\n\n"
        + "\n\n".join(parts)
        + "\n\n"
        + "\n".join(setvals)
        + "\n"
    )
    OUT.write_text(out, encoding="utf-8")
    print(f"wrote {OUT} tables={len(tables)} bytes={len(out)}")


if __name__ == "__main__":
    main()
