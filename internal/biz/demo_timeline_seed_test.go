package biz

import (
	"testing"
)

func TestEnsureDemoTimelineData(t *testing.T) {
	s := openIssueDB(t)
	ensureTraceProductionRoutingCols(t, s.DB)
	ensureRoutingStepOutputProductCol(t, s.DB)
	_, _ = s.DB.Exec(`DELETE FROM schema_meta WHERE key='demo_timeline_version'`)
	ensureDemoTimelineData(s.DB)
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_production WHERE trace_code='TR-DEMO-7D001'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected completed trace session, got %d", n)
	}
	var routingID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_trace_production WHERE trace_code='TR-DEMO-7D001'`).Scan(&routingID)
	if routingID <= 0 {
		t.Fatalf("expected locked routing_id on completed trace, got %d", routingID)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_production WHERE trace_code='TR-DEMO-TODAY001' AND status='in_progress'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected in-progress trace, got %d", n)
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_trace_production WHERE trace_code='TR-DEMO-TODAY001'`).Scan(&routingID)
	if routingID <= 0 {
		t.Fatalf("expected locked routing_id on in-progress trace, got %d", routingID)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_yield WHERE trace_code='TR-DEMO-7D001'`).Scan(&n)
	if n < 6 {
		t.Fatalf("expected >=6 yield rows for completed trace, got %d", n)
	}
	var steps int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id IN (SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE')`).Scan(&steps)
	if steps != 7 {
		t.Fatalf("expected 7 routing steps, got %d", steps)
	}
	var outCnt int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id IN (SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE') AND COALESCE(output_product_id,0)>0`).Scan(&outCnt)
	if outCnt < 7 {
		t.Fatalf("expected >=7 steps with output_product_id, got %d", outCnt)
	}
	var routingProductID, lastOutPID int64
	var lastName string
	_ = s.DB.QueryRow(`SELECT product_id FROM pd_routing WHERE code='RT-DEMO-TRACE'`).Scan(&routingProductID)
	_ = s.DB.QueryRow(`SELECT COALESCE(rs.output_product_id,0), COALESCE(p.name,'')
		FROM pd_routing_step rs LEFT JOIN prd_product p ON p.id=rs.output_product_id
		WHERE rs.routing_id IN (SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE')
		ORDER BY rs.seq_no DESC LIMIT 1`).Scan(&lastOutPID, &lastName)
	if routingProductID != lastOutPID {
		t.Fatalf("routing product_id should match last step output, got %d vs %d", routingProductID, lastOutPID)
	}
	if lastName != "烘干木薯" {
		t.Fatalf("expected final output 烘干木薯, got %q", lastName)
	}
	var st string
	_ = s.DB.QueryRow(`SELECT status FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-TR-7D'`).Scan(&st)
	if st != "stocked" {
		t.Fatalf("expected 7D weigh ticket stocked, got %q", st)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_stock_txn WHERE doc_no='DEMO-ST-WT-TR-7D' AND status='posted'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected posted stock txn for 7D trace, got %d", n)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_farmer_settlement WHERE doc_no='DEMO-FS-TR-7D' AND status='paid'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected paid farmer settlement for 7D, got %d", n)
	}
	_ = s.DB.QueryRow(`SELECT status FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-TR-T3'`).Scan(&st)
	if st != "weighed" {
		t.Fatalf("expected TODAY003 await gate (weighed), got %q", st)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_inbound_arrival WHERE doc_no LIKE 'DEMO-ARR-TR-%'`).Scan(&n)
	if n < 4 {
		t.Fatalf("expected >=4 inbound arrivals, got %d", n)
	}
}
