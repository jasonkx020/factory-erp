#!/usr/bin/env python3
from pathlib import Path

p = Path(__file__).resolve().parents[1] / "internal" / "biz" / "schema_ensure.go"
text = p.read_text(encoding="utf-8")
start2 = text.find("// seedInboundProductRoutings")
start = text.find("// SeedOpenShiftForToday")
if start2 < 0:
    start2 = start
end = text.find("// EnsurePurchaseSchema creates")
if end < 0:
    end = text.find("func EnsurePurchaseSchema")
seed_block = text[start2:end]
seed_block = seed_block.replace("date('now')", "CURRENT_DATE")
seed_block = seed_block.replace(
    "INSERT INTO pd_shift_member(shift_id, employee_id, process_id) VALUES(?,?,0)",
    "INSERT INTO pd_shift_member(shift_id, employee_id, process_id) VALUES(?,?,0) ON CONFLICT DO NOTHING",
)

out = '''package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Schema DDL is owned by migrations/erp (erp-db / init_schema). Ensure* below are no-ops.

func execSchemaRuns(db *sql.DB, label string, stmts []string) {
	_ = db
	_ = label
	_ = stmts
}

func isIdempotentSchemaErr(err error) bool {
	if err == nil {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate")
}

func EnsureAutomationSchema(db *sql.DB) { _ = db }
func EnsureClosedLoopSchema(db *sql.DB) { _ = db }
func EnsureFarmerSchema(db *sql.DB)     { _ = db }
func EnsurePurchaseSchema(db *sql.DB)   { _ = db }
func EnsureFieldLedgerSchema(db *sql.DB) { _ = db }

''' + seed_block

p.write_text(out, encoding="utf-8")
print("wrote", p, "bytes", len(out))
