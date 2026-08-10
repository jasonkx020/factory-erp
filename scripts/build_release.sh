#!/usr/bin/env bash
# Build web into embed dir and compile erp-api binary.
# Usage (from repo root):
#   ./scripts/build_release.sh
#   ./scripts/build_release.sh --skip-web
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
OUT_DIR="${OUT_DIR:-dist-release}"
SKIP_WEB=0
for a in "$@"; do
  case "$a" in
    --skip-web) SKIP_WEB=1 ;;
  esac
done

EMBED_DIST="$ROOT/internal/webui/dist"
WEB_DIST="$ROOT/web/dist"

if [[ "$SKIP_WEB" -eq 0 ]]; then
  echo "→ npm run build:dist"
  (cd "$ROOT/web" && npm run build:dist)
fi

if [[ ! -f "$WEB_DIST/index.html" ]]; then
  echo "missing web/dist/index.html" >&2
  exit 1
fi

echo "→ sync web/dist → internal/webui/dist"
rm -rf "$EMBED_DIST"
mkdir -p "$EMBED_DIST"
cp -a "$WEB_DIST"/. "$EMBED_DIST"/

mkdir -p "$OUT_DIR"
BIN="$OUT_DIR/erp-api"
echo "→ go build -o $BIN ./cmd/erp-api"
go build -o "$BIN" ./cmd/erp-api

echo ""
echo "Done: $BIN"
echo "Empty server.web_root uses embedded UI; set web_root or ERP_WEB_ROOT for external."
