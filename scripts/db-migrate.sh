#!/usr/bin/env bash
# factory-erp PostgreSQL migration CLI wrapper (Ubuntu/Linux)
#
# Usage:
#   ./scripts/db-migrate.sh baseline
#   ./scripts/db-migrate.sh status
#   ./scripts/db-migrate.sh upgrade --all
#   ./scripts/db-migrate.sh seed-dev
#   ./scripts/db-migrate.sh validate
#   ./scripts/db-migrate.sh create v1.0.1 "add foo column"
#
# Env:
#   ERP_DATABASE_DSN / DATABASE_URL / PG*   DSN
#   ERP_MIGRATIONS_DIR                     migrations root (default: ./migrations)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN="${REPO_ROOT}/bin/erp-db"

usage() {
  cat >&2 <<'EOF'
Usage:
  db-migrate.sh baseline [--dsn URL] [--migrations-dir PATH] [--dry-run]
  db-migrate.sh upgrade  [--all|--to VERSION|--file PATH] [--dry-run]
  db-migrate.sh status
  db-migrate.sh seed-dev
  db-migrate.sh validate
  db-migrate.sh create VERSION "description"

Environment:
  ERP_DATABASE_DSN, DATABASE_URL, ERP_MIGRATIONS_DIR
  PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE
EOF
  exit 1
}

ensure_erp_db() {
  if [[ -x "$BIN" ]]; then
    echo "$BIN"
    return 0
  fi
  if ! command -v go >/dev/null 2>&1; then
    echo "error: go not found; cannot build erp-db" >&2
    exit 1
  fi
  mkdir -p "${REPO_ROOT}/bin"
  (cd "$REPO_ROOT" && go build -trimpath -o bin/erp-db ./cmd/erp-db)
  echo "$BIN"
}

cmd="${1:-}"
if [[ -z "$cmd" || "$cmd" == "-h" || "$cmd" == "--help" || "$cmd" == "help" ]]; then
  usage
fi
shift || true

ERP_DB="$(ensure_erp_db)"
cd "$REPO_ROOT"
exec "$ERP_DB" "$cmd" "$@"
