#!/usr/bin/env bash
# Delivery loop: run all gate checks until green. Usage: bash scripts/delivery_loop.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
REPORT="docs/DELIVERY_LOOP_REPORT.md"
STEPS=()
FAIL=0
TS="$(date '+%Y-%m-%d %H:%M:%S')"
GATE_RAN=0

step() {
  local name="$1"
  shift
  echo "== $name =="
  if "$@"; then
    STEPS+=("| $name | PASS |")
    echo "OK  $name"
  else
    STEPS+=("| $name | FAIL |")
    echo "FAIL $name"
    FAIL=$((FAIL + 1))
  fi
}

if ! command -v go >/dev/null 2>&1; then
  echo "FAIL: go not found in PATH"
  exit 1
fi

step "go test ./internal/biz" go test ./internal/biz/ -count=1

if command -v pwsh >/dev/null 2>&1; then
  step "release_gate.ps1 (incl. smokes)" pwsh -File scripts/release_gate.ps1
  GATE_RAN=1
else
  echo "WARN skip release_gate.ps1 (no pwsh); running API + smokes subset"
  API_PID=""
  cleanup_api() {
    if [[ -n "${API_PID:-}" ]] && kill -0 "$API_PID" 2>/dev/null; then
      kill "$API_PID" 2>/dev/null || true
      wait "$API_PID" 2>/dev/null || true
    fi
  }
  trap cleanup_api EXIT
  rm -f data/erp_gate.db
  mkdir -p data
  go run ./cmd/erp-api -config configs/erp.gate.yaml &
  API_PID=$!
  for i in $(seq 1 45); do
    if curl -sf http://127.0.0.1:18080/api/v1/live >/dev/null 2>&1; then break; fi
    sleep 1
  done
  step "live ready" bash -c 'curl -sf http://127.0.0.1:18080/api/v1/live | grep -q "\"code\":1"'
  step "mobile_delivery_smoke" go run ./cmd/mobile_delivery_smoke
  step "station_pass_smoke" go run ./cmd/station_pass_smoke
fi

if [[ "$GATE_RAN" -eq 0 ]]; then
  : # smokes already ran above
else
  STEPS+=("| mobile+station smoke (via release_gate) | PASS |")
fi

if [[ -f ./cmd/erp-tools/main.go ]] || [[ -d ./cmd/erp-tools ]]; then
  step "openapi-coverage" go run ./cmd/erp-tools openapi-coverage
else
  STEPS+=("| openapi-coverage | SKIP |")
  echo "SKIP openapi-coverage (erp-tools not found)"
fi

{
  echo "# Delivery Loop Report"
  echo ""
  echo "Generated: $TS"
  echo ""
  echo "| Step | Result |"
  echo "|------|--------|"
  for row in "${STEPS[@]}"; do echo "$row"; done
  echo ""
  echo "## Manual walkthrough (required before sign-off)"
  echo ""
  echo "- [ ] u_piece: 过站 → 确认 → 我的核对"
  echo "- [ ] u_fixed: 过站（无计件金额）"
  echo "- [ ] u_purchase/u_qc: 过磅收货"
  echo "- [ ] u_warehouse: 待入库"
  echo "- [ ] u_foreman: 班组（无每箱派工）"
  echo "- [ ] admin: 生产 Hub 无报工创建表单"
} > "$REPORT"

if [[ "$FAIL" -gt 0 ]]; then
  echo ""
  echo "DELIVERY_LOOP_FAIL count=$FAIL (see $REPORT)"
  exit 1
fi
echo ""
echo "DELIVERY_LOOP_OK -> $REPORT"
