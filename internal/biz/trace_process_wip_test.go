package biz

import "testing"

func TestSplitTraceIssueRemain(t *testing.T) {
	occ, pool := splitTraceIssueRemain(7, "open", 200)
	if occ != 200 || pool != 0 {
		t.Fatalf("open worker want occupied=200 pool=0 got %v %v", occ, pool)
	}
	occ, pool = splitTraceIssueRemain(7, "work_done", 200)
	if occ != 0 || pool != 200 {
		t.Fatalf("work_done want occupied=0 pool=200 got %v %v", occ, pool)
	}
	occ, pool = splitTraceIssueRemain(0, "open", 50)
	if occ != 0 || pool != 50 {
		t.Fatalf("unassigned pool want occupied=0 pool=50 got %v %v", occ, pool)
	}
}

func TestComputeTraceProcessWipWorkDoneIssuable(t *testing.T) {
	s := openIssueDB(t)
	const trace = "T-WIP-DONE"
	const pid int64 = 5
	_, err := s.DB.Exec(`INSERT INTO pd_process_issue(trace_code, process_id, worker_id, issue_kg, status, biz_status, source)
		VALUES(?, ?, 7, 200, 'open', 'work_done', 'process')`, trace, pid)
	if err != nil {
		t.Fatalf("insert issue: %v", err)
	}
	m := s.computeTraceProcessWip(trace)
	r, ok := m[pid]
	if !ok {
		t.Fatalf("missing process %d in wip map", pid)
	}
	if roundKg(r.IssuableKg) != 200 {
		t.Fatalf("issuable want 200 got %v (occupied=%v pool=%v)", r.IssuableKg, r.OccupiedKg, r.PoolKg)
	}
	if r.OccupiedKg > kgEps {
		t.Fatalf("work_done should not count as occupied, got %v", r.OccupiedKg)
	}
}

func TestCreditTraceProcessPoolKg(t *testing.T) {
	s := openIssueDB(t)
	const trace = "T-WIP-POOL"
	const pid int64 = 3
	s.creditTraceProcessPoolKg(trace, pid, 200)
	m := s.computeTraceProcessWip(trace)
	r := m[pid]
	if roundKg(r.IssuableKg) != 200 {
		t.Fatalf("pool credit issuable want 200 got %v", r.IssuableKg)
	}
	s.creditTraceProcessPoolKg(trace, pid, 50)
	m = s.computeTraceProcessWip(trace)
	if roundKg(m[pid].IssuableKg) != 250 {
		t.Fatalf("merged pool want 250 got %v", m[pid].IssuableKg)
	}
}

func TestComputeTraceProcessWipOccupiedLimit(t *testing.T) {
	s := openIssueDB(t)
	const trace = "T-WIP-OCC"
	const pid int64 = 5
	_, err := s.DB.Exec(`INSERT INTO pd_process_issue(trace_code, process_id, worker_id, issue_kg, status, biz_status)
		VALUES(?, ?, 7, 200, 'open', 'open')`, trace, pid)
	if err != nil {
		t.Fatalf("insert issue: %v", err)
	}
	m := s.computeTraceProcessWip(trace)
	r, ok := m[pid]
	if !ok {
		t.Fatalf("missing process %d in wip map", pid)
	}
	if roundKg(r.WipKg) != 200 {
		t.Fatalf("wip want 200 got %v (occupied=%v issuable=%v)", r.WipKg, r.OccupiedKg, r.IssuableKg)
	}
	if roundKg(r.IssuableKg) > kgEps {
		t.Fatalf("occupied-only want issuable=0 got %v", r.IssuableKg)
	}
	if fail := s.assertTraceProcessWip(trace, pid, 200, true); fail != "" {
		t.Fatalf("assert 200kg wip: %s", fail)
	}
	if fail := s.assertTraceProcessWip(trace, pid, 201, true); fail == "" {
		t.Fatalf("assert 201kg should fail")
	}
}

func TestIssueTraceProcessKgFromOccupied(t *testing.T) {
	s := openIssueDB(t)
	const trace = "T-ISSUE-OCC"
	const fromPID int64 = 1
	const toPID int64 = 2
	_, err := s.DB.Exec(`INSERT INTO pd_trace_production(trace_code, status) VALUES(?, 'in_progress')`, trace)
	if err != nil {
		t.Fatalf("trace production: %v", err)
	}
	res, err := s.DB.Exec(`INSERT INTO pd_process_issue(trace_code, process_id, worker_id, issue_kg, status, biz_status)
		VALUES(?, ?, 7, 200, 'open', 'open')`, trace, fromPID)
	if err != nil {
		t.Fatalf("insert source issue: %v", err)
	}
	srcID, _ := res.LastInsertId()
	out, fail := s.issueTraceProcessKg(trace, 8, fromPID, toPID, 10, 50, 8)
	if fail != "" {
		t.Fatalf("issue from occupied: %s", fail)
	}
	if asInt64Or0(out["id"]) <= 0 {
		t.Fatalf("missing new issue id: %v", out)
	}
	var done float64
	_ = s.DB.QueryRow(`SELECT completed_kg FROM pd_process_issue WHERE id=?`, srcID).Scan(&done)
	if roundKg(done) != 50 {
		t.Fatalf("source completed_kg want 50 got %v", done)
	}
	m := s.computeTraceProcessWip(trace)
	if roundKg(m[fromPID].WipKg) != 150 {
		t.Fatalf("source wip after issue want 150 got %v", m[fromPID].WipKg)
	}
}
