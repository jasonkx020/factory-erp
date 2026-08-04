# -*- coding: utf-8 -*-
"""Compare OpenAPI ops vs Gin routes dumped by erp-api /debug/routes or routes.json."""
from __future__ import annotations

import json
import re
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OPS = ROOT / "scripts" / "openapi_ops.json"
ROUTES_FILE = ROOT / "scripts" / "gin_routes.json"


def norm_gin(p: str) -> str:
    p = p.split("?")[0]
    # gin uses :id ; openapi uses {id}
    return re.sub(r":([A-Za-z0-9_]+)", r"{\1}", p)


def load_ops():
    if not OPS.exists():
        print("missing openapi_ops.json — run gen_routes.py first", file=sys.stderr)
        sys.exit(2)
    return [(o["method"], o["path"]) for o in json.loads(OPS.read_text(encoding="utf-8"))]


def load_gin_routes():
    if ROUTES_FILE.exists():
        data = json.loads(ROUTES_FILE.read_text(encoding="utf-8"))
        return [(r["method"].upper(), norm_gin(r["path"])) for r in data]
    # try live endpoint
    try:
        with urllib.request.urlopen("http://127.0.0.1:18080/api/v1/health/routes", timeout=2) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            items = data.get("data", data)
            if isinstance(items, dict):
                items = items.get("routes", [])
            return [(r["method"].upper(), norm_gin(r["path"])) for r in items]
    except Exception as e:
        print(f"cannot load gin routes: {e}", file=sys.stderr)
        print("Start erp-api or write scripts/gin_routes.json", file=sys.stderr)
        sys.exit(2)


def main():
    ops = load_ops()
    gin = set(load_gin_routes())
    missing = []
    for m, p in ops:
        if (m, p) not in gin:
            missing.append((m, p))
    covered = len(ops) - len(missing)
    print(f"coverage {covered}/{len(ops)} ({100.0 * covered / len(ops):.1f}%)")
    if missing:
        print(f"missing {len(missing)}:")
        for m, p in missing[:40]:
            print(f"  {m} {p}")
        if len(missing) > 40:
            print(f"  ... {len(missing) - 40} more")
        sys.exit(1)
    print("OK: all OpenAPI operations registered")
    sys.exit(0)


if __name__ == "__main__":
    main()
