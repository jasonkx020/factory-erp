# Release Gate Sign-off

Generated: 2026-08-06 12:36:40
Config: `configs/erp.gate.yaml` (seed.demo=false, strong JWT, CORS locked)

| Check | Result |
|-------|--------|
| go test ./internal/biz | PASS |
| /live /ready(db=up) /health /metrics | PASS |
| login + weigh/inventory/report/finance lists | PASS |
| finance loop subject->fund/ledger->voucher post->writeoff->month close->statements | PASS |
| unbalanced voucher post rejected | PASS |
| request without token rejected | PASS |

Sign: _______________  Date: _______________

