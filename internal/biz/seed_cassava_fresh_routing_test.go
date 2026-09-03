package biz

import "testing"

func TestEnsureFreshCassavaRouting(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	EnsureFreshCassavaRouting(s.DB)

	var rid int64
	var name string
	err := s.DB.QueryRow(`SELECT id, COALESCE(name,'') FROM pd_routing
		WHERE code='RT-CASSAVA-FRESH' AND status='active' AND COALESCE(is_deleted,0)=0`).Scan(&rid, &name)
	if err != nil || rid <= 0 {
		t.Fatalf("routing missing: %v", err)
	}
	if name != "鲜木薯完整加工" {
		t.Fatalf("routing name want 鲜木薯完整加工 got %q", name)
	}

	steps := s.loadRoutingStepsByID(rid)
	want := []string{"清洗", "去皮", "切段", "去芯", "切片", "烘干"}
	if len(steps) != len(want) {
		t.Fatalf("steps want %d got %d", len(want), len(steps))
	}
	for i, label := range want {
		got := steps[i].StepName
		if got == "" {
			got = steps[i].ProcessName
		}
		if got != label {
			t.Fatalf("step %d want %s got %s", i+1, label, got)
		}
		if steps[i].OutputProductID <= 0 {
			t.Fatalf("step %d missing output_product_id", i+1)
		}
	}

	var productID, suggested int64
	_ = s.DB.QueryRow(`SELECT id FROM prd_product WHERE code='RM-CASSAVA'`).Scan(&productID)
	if productID <= 0 {
		t.Fatal("RM-CASSAVA product missing")
	}
	suggested = s.resolveRoutingID(0, productID)
	if suggested != rid {
		t.Fatalf("suggested routing want %d got %d", rid, suggested)
	}

	// Idempotent second call.
	EnsureFreshCassavaRouting(s.DB)
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing WHERE code='RT-CASSAVA-FRESH'`).Scan(&n)
	if n != 1 {
		t.Fatalf("duplicate routing rows: %d", n)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id=?`, rid).Scan(&n)
	if n != 6 {
		t.Fatalf("step count after re-seed want 6 got %d", n)
	}
}
