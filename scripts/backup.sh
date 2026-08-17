#!/usr/bin/env bash
# factory-erp PostgreSQL backup / restore helpers (Ubuntu/Linux)
#
# Usage:
#   ./scripts/backup.sh                  # dump
#   ./scripts/backup.sh dump
#   ./scripts/backup.sh restore FILE.dump
#   ./scripts/backup.sh status           # erp-db status
#
# Env:
#   ERP_DATABASE_DSN / DATABASE_URL / PG*

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKUP_DIR="${ERP_BACKUP_DIR:-$REPO_ROOT/backups}"

resolve_dsn() {
  if [[ -n "${ERP_DATABASE_DSN:-}" ]]; then
    echo "$ERP_DATABASE_DSN"
    return
  fi
  if [[ -n "${DATABASE_URL:-}" ]]; then
    echo "$DATABASE_URL"
    return
  fi
  local host="${PGHOST:-127.0.0.1}"
  local port="${PGPORT:-5432}"
  local user="${PGUSER:-erp}"
  local pass="${PGPASSWORD:-erp}"
  local db="${PGDATABASE:-erp_factory}"
  local ssl="${PGSSLMODE:-disable}"
  echo "postgres://${user}:${pass}@${host}:${port}/${db}?sslmode=${ssl}"
}

usage() {
  cat >&2 <<'EOF'
Usage:
  backup.sh [dump]
  backup.sh restore <file.dump>
  backup.sh status

Environment:
  ERP_DATABASE_DSN, DATABASE_URL, ERP_BACKUP_DIR
  PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE PGSSLMODE
EOF
  exit 1
}

cmd="${1:-dump}"
dsn="$(resolve_dsn)"

case "$cmd" in
  dump|"")
    command -v pg_dump >/dev/null 2>&1 || { echo "error: pg_dump not found (apt install postgresql-client)" >&2; exit 1; }
    mkdir -p "$BACKUP_DIR"
    stamp="$(date +%Y%m%d_%H%M%S)"
    out="${BACKUP_DIR}/erp_factory_${stamp}.dump"
    echo "pg_dump -> $out"
    pg_dump --dbname="$dsn" -Fc -f "$out"
    echo "done. restore: $0 restore $out"
    ;;
  restore)
    file="${2:-}"
    [[ -n "$file" && -f "$file" ]] || { echo "error: restore requires an existing .dump file" >&2; usage; }
    command -v pg_restore >/dev/null 2>&1 || { echo "error: pg_restore not found (apt install postgresql-client)" >&2; exit 1; }
    echo "WARNING: restores into $dsn (clean+if-exists)"
    pg_restore --clean --if-exists --dbname="$dsn" "$file"
    echo "restore done"
    ;;
  status)
    exec "$SCRIPT_DIR/db-migrate.sh" status --dsn "$dsn"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "unknown command: $cmd" >&2
    usage
    ;;
esac
