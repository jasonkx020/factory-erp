package biz

import (
	"database/sql"
	"testing"
)

func ensureTraceProductionRoutingCols(t *testing.T, db *sql.DB) {
	t.Helper()
	_, _ = db.Exec(`ALTER TABLE pd_trace_production ADD COLUMN IF NOT EXISTS routing_id BIGINT NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE pd_trace_production ADD COLUMN IF NOT EXISTS product_id BIGINT NOT NULL DEFAULT 0`)
}

func TestValidateTraceRoutingStart(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	ensureTraceProductionRoutingCols(t, s.DB)

	const trace = "T-ROUTING-START"
	const productID int64 = 1
	_, _ = s.DB.Exec(`DELETE FROM pd_trace_production WHERE UPPER(trace_code)=?`, trace)
	_, _ = s.DB.Exec(`DELETE FROM inv_box_code WHERE UPPER(trace_code)=?`, trace)

	var routingID int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE product_id=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, productID).Scan(&routingID)
	if routingID <= 0 {
		t.Skip("no active routing for product 1")
	}

	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, qty, weight, trace_code, status)
		VALUES('BX-ROUTING-TEST', ?, 1, 100, 100, ?, 'open')`, productID, trace)
	if err != nil {
		t.Fatalf("insert box: %v", err)
	}

	if _, code := s.validateTraceRoutingStart(trace, 0); code != "ROUTING_REQUIRED" {
		t.Fatalf("want ROUTING_REQUIRED got %q", code)
	}
	if _, code := s.validateTraceRoutingStart(trace, routingID+99999); code != "ROUTING_NOT_FOUND" {
		t.Fatalf("want ROUTING_NOT_FOUND got %q", code)
	}

	var wrongProductRouting int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE product_id<>? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, productID).Scan(&wrongProductRouting)
	if wrongProductRouting > 0 {
		if _, code := s.validateTraceRoutingStart(trace, wrongProductRouting); code != "ROUTING_PRODUCT_MISMATCH" {
			t.Fatalf("want ROUTING_PRODUCT_MISMATCH got %q", code)
		}
	}

	pid, code := s.validateTraceRoutingStart(trace, routingID)
	if code != "" {
		t.Fatalf("valid routing: %s", code)
	}
	if pid != productID {
		t.Fatalf("product_id want %d got %d", productID, pid)
	}
}

func TestResolveTraceSessionRoutingLocked(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	ensureTraceProductionRoutingCols(t, s.DB)

	const trace = "T-ROUTING-LOCK"
	const productID int64 = 1
	_, _ = s.DB.Exec(`DELETE FROM pd_trace_production WHERE UPPER(trace_code)=?`, trace)
	_, _ = s.DB.Exec(`DELETE FROM inv_box_code WHERE UPPER(trace_code)=?`, trace)

	var routingA int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE product_id=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, productID).Scan(&routingA)
	if routingA <= 0 {
		t.Skip("no routing for product 1")
	}

	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, qty, weight, trace_code, status)
		VALUES('BX-ROUTING-LOCK', ?, 1, 50, 50, ?, 'open')`, productID, trace)
	if err != nil {
		t.Fatalf("insert box: %v", err)
	}
	_, err = s.DB.Exec(`INSERT INTO pd_trace_production(trace_code, status, routing_id, product_id)
		VALUES(?, 'in_progress', ?, ?)`, trace, routingA, productID)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	got := s.resolveTraceSessionRoutingID(trace)
	if got != routingA {
		t.Fatalf("locked routing want %d got %d", routingA, got)
	}
	_, rid, _, steps := s.resolveTraceRoutingSteps(trace)
	if rid != routingA {
		t.Fatalf("resolve steps routing want %d got %d", routingA, rid)
	}
	if len(steps) == 0 {
		t.Fatalf("expected routing steps")
	}
}

func TestResolveRoutingIDNoSilentCassavaFallback(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	// Unknown product must not silently fall back to RT-CASSAVA / id=1.
	got := s.resolveRoutingID(0, 999999001)
	if got != 0 {
		t.Fatalf("want 0 without product routing, got %d", got)
	}
}

func TestAssertProcessInSessionRouting(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	ensureTraceProductionRoutingCols(t, s.DB)

	const trace = "T-PROC-IN-ROUTING"
	const productID int64 = 1
	_, _ = s.DB.Exec(`DELETE FROM pd_trace_production WHERE UPPER(trace_code)=?`, trace)

	var routingID, processID int64
	_ = s.DB.QueryRow(`SELECT r.id, rs.process_id FROM pd_routing r
		JOIN pd_routing_step rs ON rs.routing_id=r.id
		WHERE r.product_id=? AND r.status='active' AND COALESCE(r.is_deleted,0)=0
		ORDER BY r.id, rs.seq_no LIMIT 1`, productID).Scan(&routingID, &processID)
	if routingID <= 0 || processID <= 0 {
		t.Skip("no routing steps for product 1")
	}

	_, err := s.DB.Exec(`INSERT INTO pd_trace_production(trace_code, status, routing_id, product_id)
		VALUES(?, 'in_progress', ?, ?)`, trace, routingID, productID)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if code := s.assertProcessInSessionRouting(trace, processID); code != "" {
		t.Fatalf("in-routing process: %s", code)
	}
	if code := s.assertProcessInSessionRouting(trace, processID+999999); code != "PROCESS_NOT_IN_ROUTING" {
		t.Fatalf("want PROCESS_NOT_IN_ROUTING got %q", code)
	}
}

func TestPlantLinePreviewHasConfiguredSteps(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	var cnt int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing WHERE status='active' AND COALESCE(is_deleted,0)=0`).Scan(&cnt)
	if cnt <= 0 {
		t.Skip("no active routing")
	}
	var rid int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`).Scan(&rid)
	steps := s.loadRoutingStepsByID(rid)
	if len(steps) == 0 {
		t.Fatalf("active routing %d should have steps for plant-line preview", rid)
	}
}
