package biz

import (
	"testing"
)

func TestEnsureDemoTimelineData(t *testing.T) {
	s := openIssueDB(t)
	_, _ = s.DB.Exec(`DELETE FROM schema_meta WHERE key='demo_timeline_version'`)
	ensureDemoTimelineData(s.DB)
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_production WHERE trace_code='TR-DEMO-7D001'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected completed trace session, got %d", n)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_production WHERE trace_code='TR-DEMO-TODAY001' AND status='in_progress'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected in-progress trace, got %d", n)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_yield WHERE trace_code='TR-DEMO-7D001'`).Scan(&n)
	if n < 4 {
		t.Fatalf("expected >=4 yield rows for completed trace, got %d", n)
	}
	var steps int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id IN (SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE')`).Scan(&steps)
	if steps != 4 {
		t.Fatalf("expected 4 routing steps, got %d", steps)
	}
}
