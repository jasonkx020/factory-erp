#!/usr/bin/env bash
# factory-erp Ubuntu maintenance helpers
#
# Usage:
#   ./scripts/maint.sh help
#   ./scripts/maint.sh doctor
#   ./scripts/maint.sh up-dev          # docker compose up postgres (+ optional api)
#   ./scripts/maint.sh migrate status|baseline|upgrade|seed-dev|validate
#   ./scripts/maint.sh backup|restore FILE
#   ./scripts/maint.sh psql             # open psql to ERP_DATABASE_DSN

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

resolve_dsn() {
  if [[ -n "${ERP_DATABASE_DSN:-}" ]]; then echo "$ERP_DATABASE_DSN"; return; fi
  if [[ -n "${DATABASE_URL:-}" ]]; then echo "$DATABASE_URL"; return; fi
  echo "postgres://${PGUSER:-erp}:${PGPASSWORD:-erp}@${PGHOST:-127.0.0.1}:${PGPORT:-5432}/${PGDATABASE:-erp_factory}?sslmode=${PGSSLMODE:-disable}"
}

usage() {
  cat <<'EOF'
factory-erp maint (Ubuntu)

  maint.sh doctor              Check go / psql / docker / DSN
  maint.sh up-dev [--api]      Start postgres (and api if --api)
  maint.sh down-dev            Stop compose stack
  maint.sh migrate <args...>   Proxy to db-migrate.sh
  maint.sh backup              pg_dump custom format
  maint.sh restore <file>      pg_restore
  maint.sh psql                Interactive psql
  maint.sh upgrade-prod        status → upgrade --all (explicit prod path)

Env: ERP_DATABASE_DSN, DATABASE_URL, PG*, COMPOSE_FILE
EOF
}

doctor() {
  echo "== factory-erp doctor =="
  echo "repo: $REPO_ROOT"
  echo "dsn:  $(resolve_dsn | sed -E 's#://([^:/]+):([^@]+)@#://\1:***@#')"
  command -v go >/dev/null && go version || echo "go: MISSING"
  command -v psql >/dev/null && psql --version || echo "psql: MISSING (sudo apt install postgresql-client)"
  command -v pg_dump >/dev/null && pg_dump --version || echo "pg_dump: MISSING"
  command -v docker >/dev/null && docker --version || echo "docker: optional"
  if [[ -f "$REPO_ROOT/migrations/erp/schema.sql" ]]; then
    echo "migrations/erp/schema.sql: OK"
  else
    echo "migrations/erp/schema.sql: MISSING"
  fi
  if command -v psql >/dev/null 2>&1; then
    if psql "$(resolve_dsn)" -c 'SELECT 1' >/dev/null 2>&1; then
      echo "postgres ping: OK"
    else
      echo "postgres ping: FAIL (start with: $0 up-dev)"
    fi
  fi
}

cmd="${1:-help}"
shift || true

case "$cmd" in
  help|-h|--help) usage ;;
  doctor) doctor ;;
  up-dev)
    cd "$REPO_ROOT"
    if [[ "${1:-}" == "--api" ]]; then
      docker compose up -d postgres api
    else
      docker compose up -d postgres
    fi
    ;;
  down-dev)
    cd "$REPO_ROOT"
    docker compose down
    ;;
  migrate)
    exec "$SCRIPT_DIR/db-migrate.sh" "$@"
    ;;
  backup)
    exec "$SCRIPT_DIR/backup.sh" dump
    ;;
  restore)
    exec "$SCRIPT_DIR/backup.sh" restore "$@"
    ;;
  psql)
    command -v psql >/dev/null 2>&1 || { echo "psql missing" >&2; exit 1; }
    exec psql "$(resolve_dsn)"
    ;;
  upgrade-prod)
    echo "Production upgrade: backup → status → upgrade --all → restart API"
    "$SCRIPT_DIR/backup.sh" dump
    "$SCRIPT_DIR/db-migrate.sh" status
    "$SCRIPT_DIR/db-migrate.sh" upgrade --all
    echo "Now rolling-restart your API process/service."
    ;;
  *)
    echo "unknown: $cmd" >&2
    usage
    exit 1
    ;;
esac
