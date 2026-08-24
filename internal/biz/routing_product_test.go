package biz

import (
	"database/sql"
	"fmt"
	"testing"
)

func ensureRoutingStepOutputProductCol(t *testing.T, db *sql.DB) {
	t.Helper()
	_, _ = db.Exec(`ALTER TABLE pd_routing_step ADD COLUMN IF NOT EXISTS output_product_id BIGINT NOT NULL DEFAULT 0`)
}

func TestResolveNextProcessByProduct(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	ensureRoutingStepOutputProductCol(t, s.DB)

	const routingCode = "RT-ROUTING-PROD-TEST"
	_, _ = s.DB.Exec(`DELETE FROM pd_routing_step WHERE routing_id IN (SELECT id FROM pd_routing WHERE code=?)`, routingCode)
	_, _ = s.DB.Exec(`DELETE FROM pd_routing WHERE code=?`, routingCode)

	var p1, p2, p3 int64
	_ = s.DB.QueryRow(`SELECT id FROM prd_product ORDER BY id LIMIT 1`).Scan(&p1)
	_ = s.DB.QueryRow(`SELECT id FROM prd_product ORDER BY id OFFSET 1 LIMIT 1`).Scan(&p2)
	_ = s.DB.QueryRow(`SELECT id FROM prd_product ORDER BY id OFFSET 2 LIMIT 1`).Scan(&p3)
	if p1 <= 0 || p2 <= 0 || p3 <= 0 {
		t.Skip("need at least 3 products")
	}

	res, err := s.DB.Exec(`INSERT INTO pd_routing(code, name, product_id, version_no, status) VALUES(?,?,?,?,'active')`,
		routingCode, "产物测试工艺", p3, "V1")
	if err != nil {
		t.Fatalf("insert routing: %v", err)
	}
	rid, _ := res.LastInsertId()
	procs := []int64{7, 1, 3}
	outs := []int64{p1, p2, p3}
	for i, proc := range procs {
		_, err := s.DB.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, output_product_id)
			VALUES(?,?,?,?,?,?)`, rid, i+1, proc, fmt.Sprintf("S%d", i+1), fmt.Sprintf("步%d", i+1), outs[i])
		if err != nil {
			t.Fatalf("insert step: %v", err)
		}
	}

	next := s.resolveNextProcessByProduct(rid, p1)
	if next != 1 {
		t.Fatalf("want next process 1 got %d", next)
	}
	inPID, outPID := s.resolveProcessProducts(rid, 1)
	if inPID != p1 || outPID != p2 {
		t.Fatalf("process products want %d/%d got %d/%d", p1, p2, inPID, outPID)
	}
}

func TestValidateRoutingOutputProducts(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	steps := []compiledStep{
		{Seq: 1, Code: "S1", OutputProductID: 10},
		{Seq: 2, Code: "S2", OutputProductID: 20},
	}
	if code := s.validateRoutingOutputProducts(1, steps, 30); code != "ROUTING_FINAL_PRODUCT_MISMATCH" {
		t.Fatalf("want ROUTING_FINAL_PRODUCT_MISMATCH got %q", code)
	}
	steps[1].OutputProductID = 30
	if code := s.validateRoutingOutputProducts(1, steps, 30); code != "" {
		t.Fatalf("valid chain: %q", code)
	}
}

func TestAssertBoardProductForProcess(t *testing.T) {
	s := &Services{DB: openSmokeDB(t)}
	ensureRoutingStepOutputProductCol(t, s.DB)

	board := &boardState{ProductID: 99, Trace: "T-PROD-MISMATCH", ProcessID: 1}
	if fail := s.assertBoardProductForProcess(board, 0); fail != "" {
		t.Fatalf("skip when no process: %q", fail)
	}
}
