package biz

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func openIssueDB(t *testing.T) *Services {
	t.Helper()
	db := openSmokeDB(t)
	stmts := []string{
		`CREATE TEMP TABLE IF NOT EXISTS pd_process_issue(
			id BIGSERIAL PRIMARY KEY, board_id BIGINT, board_code TEXT, trace_code TEXT,
			process_id BIGINT, step_id BIGINT, worker_id BIGINT DEFAULT 0,
			issue_kg DOUBLE PRECISION DEFAULT 0, returned_kg DOUBLE PRECISION DEFAULT 0, completed_kg DOUBLE PRECISION DEFAULT 0,
			wage_settled_kg DOUBLE PRECISION DEFAULT 0,
			status TEXT DEFAULT 'open', biz_status TEXT DEFAULT 'open', issued_by_employee_id BIGINT DEFAULT 0,
			work_done_at TIMESTAMPTZ, work_done_by BIGINT DEFAULT 0,
			pending_return_kg DOUBLE PRECISION DEFAULT 0, pending_reweigh_kg DOUBLE PRECISION DEFAULT 0,
			pending_photo_url TEXT DEFAULT '', pending_return_by BIGINT DEFAULT 0, pending_remark TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_process_move(
			id BIGSERIAL PRIMARY KEY, board_id BIGINT, board_code TEXT, trace_code TEXT,
			from_process_id BIGINT, from_step_id BIGINT, to_process_id BIGINT, to_step_id BIGINT,
			to_worker_id BIGINT DEFAULT 0, kg DOUBLE PRECISION, move_kind TEXT, issue_ids TEXT,
			created_by BIGINT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_process_move_alloc(
			id BIGSERIAL PRIMARY KEY, move_id BIGINT, issue_id BIGINT, kg DOUBLE PRECISION, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_board_process_yield(
			id BIGSERIAL PRIMARY KEY, board_id BIGINT, board_code TEXT, trace_code TEXT, process_id BIGINT,
			input_kg DOUBLE PRECISION DEFAULT 0, output_kg DOUBLE PRECISION DEFAULT 0, loss_kg DOUBLE PRECISION DEFAULT 0, loss_rate DOUBLE PRECISION DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(board_id, process_id))`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_trace_process_yield(
			id BIGSERIAL PRIMARY KEY, trace_code TEXT, process_id BIGINT,
			input_kg DOUBLE PRECISION DEFAULT 0, output_kg DOUBLE PRECISION DEFAULT 0, loss_kg DOUBLE PRECISION DEFAULT 0, loss_rate DOUBLE PRECISION DEFAULT 0,
			board_count INTEGER DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(trace_code, process_id))`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_station_flow_log(
			id BIGSERIAL PRIMARY KEY, event_type TEXT, biz_date TEXT DEFAULT '', board_id BIGINT DEFAULT 0,
			board_code TEXT DEFAULT '', trace_code TEXT DEFAULT '', process_id BIGINT DEFAULT 0, step_id BIGINT DEFAULT 0,
			process_name TEXT DEFAULT '', worker_id BIGINT DEFAULT 0, worker_name TEXT DEFAULT '', badge_code TEXT DEFAULT '',
			actor_user_id BIGINT DEFAULT 0, operator_employee_id BIGINT DEFAULT 0, kg DOUBLE PRECISION DEFAULT 0,
			pay_mode TEXT DEFAULT 'none', emp_type TEXT DEFAULT '', rate DOUBLE PRECISION DEFAULT 0, amount DOUBLE PRECISION DEFAULT 0,
			ref_type TEXT DEFAULT '', ref_id BIGINT DEFAULT 0, before_json TEXT, after_json TEXT, remark TEXT, payload_json TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW())`,
		`ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS is_deleted INTEGER DEFAULT 0`,
		`ALTER TABLE pd_process ADD COLUMN IF NOT EXISTS pay_mode TEXT DEFAULT 'none'`,
		`ALTER TABLE hr_employee ADD COLUMN IF NOT EXISTS emp_type TEXT DEFAULT ''`,
		`UPDATE inv_box_code SET current_process_id=1, current_step_id=10, weight=100, qty=100, trace_code='T-ISSUE' WHERE code='BX-SMOKE'`,
		`INSERT INTO pay_process_wage_rate(process_id, rate, status) SELECT 1, 0.5, 'active'
			WHERE NOT EXISTS (SELECT 1 FROM pay_process_wage_rate WHERE process_id=1)`,
		`UPDATE pd_process SET is_piecework=1, pay_mode='weight' WHERE id=1`,
		`UPDATE pd_routing_step SET is_piecework=1, auto_next=1 WHERE id=10`,
		`INSERT INTO hr_employee(id, name, badge_code, emp_no, emp_type, status) VALUES
			(7,'工人甲','BADGE-A','E7','piece','active') ON CONFLICT (id) DO UPDATE SET emp_type='piece'`,
		`INSERT INTO hr_employee(id, name, badge_code, emp_no, emp_type, status) VALUES
			(8,'工人乙','BADGE-B','E8','piece','active') ON CONFLICT (id) DO UPDATE SET emp_type='piece'`,
		`INSERT INTO hr_employee(id, name, badge_code, emp_no, emp_type, status) VALUES
			(9,'工人丙','BADGE-C','E9','piece','active') ON CONFLICT (id) DO UPDATE SET emp_type='piece'`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("issue schema: %v\n%s", err, s)
		}
	}
	return &Services{DB: db}
}

func TestBoardIssueReturnMovePiecework(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := openIssueDB(t)
	board, errMsg := s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" || board == nil {
		t.Fatalf("load board: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 30, 7); fail != "" {
		t.Fatalf("issue A: %s", fail)
	}
	var pwQty float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwQty)
	if pwQty != 0 {
		t.Fatalf("issue must not settle piecework, got summary qty %v", pwQty)
	}
	if got := s.workerLockedPieceworkKg(7, 1); roundKg(got) != 30 {
		t.Fatalf("after issue locked kg want 30 got %v", got)
	}
	if got := roundMoney(s.workerLockedPieceworkKg(7, 1) * 0.5); got != 15 {
		t.Fatalf("after issue locked wage want 15 got %v", got)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.issueBoardKg(board, 8, 1, 10, 20, 8); fail != "" {
		t.Fatalf("issue B: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if roundKg(board.Weight) != 50 {
		t.Fatalf("after two issues available want 50 got %v", board.Weight)
	}
	if got := s.workerOpenKg(board.ID, 1, 7); roundKg(got) != 30 {
		t.Fatalf("worker A open want 30 got %v", got)
	}
	if got := s.processOpenKg(board.ID, 1); roundKg(got) != 50 {
		t.Fatalf("process open want 50 got %v", got)
	}

	if _, fail := s.returnBoardKg(board, 7, 10); fail != "" {
		t.Fatalf("return: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if roundKg(board.Weight) != 60 {
		t.Fatalf("after return available want 60 got %v", board.Weight)
	}
	if got := s.workerOpenKg(board.ID, 1, 7); roundKg(got) != 20 {
		t.Fatalf("worker A open after return want 20 got %v", got)
	}
	if got := s.workerLockedPieceworkKg(7, 1); roundKg(got) != 20 {
		t.Fatalf("worker A locked after return want 20 got %v", got)
	}
	wip := roundKg(board.Weight + s.processOpenKg(board.ID, 1) + s.poolOpenKg(board.ID, 1))
	if wip != 100 {
		t.Fatalf("wip should stay 100 after return, got %v", wip)
	}

	if _, fail := s.moveBoardKg(board, 9, 30, "next", 1, 1, 10); fail != "AUTO_ROUTING_DISABLED" {
		t.Fatalf("next must be disabled, got %s", fail)
	}
	out, fail := s.moveBoardKg(board, 9, 30, "stock_in", 1, 1, 10)
	if fail != "" {
		t.Fatalf("stock_in: %s", fail)
	}
	if roundKg(asFloatOr0(out["settled_wage_amount"])) != 0 {
		t.Fatalf("stock_in must not settle wage, got %v", out["settled_wage_amount"])
	}
	newCode := strOr(out["new_board_code"])
	if newCode == "" {
		t.Fatal("stock_in must return new_board_code")
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	fromWip := roundKg(board.Weight + s.processOpenKg(board.ID, 1) + s.poolOpenKg(board.ID, 1))
	if fromWip != 70 {
		t.Fatalf("from wip want 70 got %v (process=%d weight=%v)", fromWip, board.ProcessID, board.Weight)
	}
	child, errMsg := s.loadBoardByCode(newCode)
	if errMsg != "" || child == nil {
		t.Fatalf("load new board: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(child, 9, 2, 11, 30, 9); fail != "" {
		t.Fatalf("manual issue into process 2: %s", fail)
	}
	toOpen := s.processOpenKg(child.ID, 2)
	if roundKg(toOpen) != 30 {
		t.Fatalf("to occupancy want 30 got %v", toOpen)
	}

	var pwAmt float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0), COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwQty, &pwAmt)
	if roundKg(pwQty) != 0 {
		t.Fatalf("stock_in must not write piecework summary, got qty %v", pwQty)
	}
	// 净占用=领取−退库；入库不减锁定。A=20，B=20
	if got := s.workerLockedPieceworkKg(7, 1); roundKg(got) != 20 {
		t.Fatalf("worker A locked after stock_in want 20 got %v", got)
	}
	if got := s.workerLockedPieceworkKg(8, 1); roundKg(got) != 20 {
		t.Fatalf("worker B locked after stock_in want 20 got %v", got)
	}

	s.upsertPieceworkSummaryKeyedOnDate(7, 1, "2026-08-17", "DAY:2026-08-17:testA", 20, 20, 20, 0, 1)
	_, _ = s.DB.Exec(`UPDATE pd_process_issue SET wage_settled_kg=issue_kg-returned_kg WHERE worker_id=7 AND process_id=1`)
	s.upsertPieceworkSummaryKeyedOnDate(8, 1, "2026-08-17", "DAY:2026-08-17:testB", 20, 20, 20, 0, 1)
	_, _ = s.DB.Exec(`UPDATE pd_process_issue SET wage_settled_kg=issue_kg-returned_kg WHERE worker_id=8 AND process_id=1`)
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0), COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwQty, &pwAmt)
	if roundKg(pwQty) != 40 {
		t.Fatalf("after day settle qty want 40 got %v", pwQty)
	}
	if roundKg(pwAmt) != 20 { // 40 * 0.5
		t.Fatalf("piecework amount want 20 got %v", pwAmt)
	}
	if got := s.workerLockedPieceworkKg(7, 1); roundKg(got) != 0 {
		t.Fatalf("worker A locked after settle want 0 got %v", got)
	}
	if got := s.workerLockedPieceworkKg(8, 1); roundKg(got) != 0 {
		t.Fatalf("worker B locked after settle want 0 got %v", got)
	}

	if _, fail := s.returnBoardKg(child, 9, 5); fail != "" {
		t.Fatalf("next process return: %s", fail)
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwQty)
	if roundKg(pwQty) != 40 {
		t.Fatalf("return must not change from-process piecework, got %v", pwQty)
	}
	var toPW float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=2`).Scan(&toPW)
	if toPW != 0 {
		t.Fatalf("to-process piecework should wait for day settle, got %v", toPW)
	}
	var yieldN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield WHERE board_id=?`, board.ID).Scan(&yieldN)
	if yieldN != 0 {
		t.Fatalf("board-level yield must not be written, got %d", yieldN)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_yield WHERE trace_code='T-ISSUE'`).Scan(&yieldN)
	if yieldN != 0 {
		t.Fatalf("unfinished board must not snapshot trace yield, got %d", yieldN)
	}
}

func TestBoardYieldSnapshotOnceAndNoDoubleCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := openIssueDB(t)
	board, errMsg := s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" || board == nil {
		t.Fatalf("load board: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 30, 7); fail != "" {
		t.Fatalf("issue A: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.issueBoardKg(board, 8, 1, 10, 20, 8); fail != "" {
		t.Fatalf("issue B: %s", fail)
	}
	var inKg float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg),0) FROM pd_process_issue WHERE board_id=? AND process_id=1 AND worker_id>0`, board.ID).Scan(&inKg)
	if roundKg(inKg) != 50 {
		t.Fatalf("two issues must sum kg=50, got %v", inKg)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 60, 7); fail != "QTY_EXCEEDS_AVAILABLE" {
		t.Fatalf("cannot re-issue beyond remaining, got %s", fail)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 50, 7); fail != "" {
		t.Fatalf("issue rest: %s", fail)
	}
	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, qty, weight, status, current_process_id, current_step_id, trace_code)
		VALUES('BX-SMOKE-2',1,1,80,80,'open',1,10,'T-ISSUE')`)
	if err != nil {
		t.Fatalf("insert board2: %v", err)
	}
	s.snapshotTraceYield("T-ISSUE")
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield`).Scan(&n)
	if n != 0 {
		t.Fatalf("board yield must stay empty, got %d", n)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_yield`).Scan(&n)
	if n != 0 {
		t.Fatalf("trace yield before all boards finished want 0 got %d", n)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	out, fail := s.moveBoardKg(board, 7, 100, "finish_in", 1, 1, 10)
	if fail != "" {
		t.Fatalf("finish move: %s", fail)
	}
	if code := strOr(out["new_board_code"]); code != "" {
		_, _ = s.DB.Exec(`UPDATE inv_box_code SET status='finished', weight=0, qty=0 WHERE code=?`, code)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if board.Status == "finished" {
		t.Fatalf("move must not finish board")
	}
	if _, fail := s.closeBoard(board, false); fail != "" {
		t.Fatalf("close empty remain: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if board.Status != "finished" {
		t.Fatalf("board status want finished got %s", board.Status)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield WHERE board_id=?`, board.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("close must not write board yield, got %d", n)
	}
	var traceN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_yield WHERE trace_code='T-ISSUE'`).Scan(&traceN)
	if traceN != 0 {
		t.Fatalf("trace snapshot must wait until all boards finished, got %d", traceN)
	}

	b2, errMsg := s.loadBoardByCode("BX-SMOKE-2")
	if errMsg != "" {
		t.Fatalf("load board2: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(b2, 7, 1, 10, 80, 7); fail != "" {
		t.Fatalf("issue board2: %s", fail)
	}
	b2, _ = s.loadBoardByCode("BX-SMOKE-2")
	out2, fail := s.moveBoardKg(b2, 7, 80, "finish_in", 1, 1, 10)
	if fail != "" {
		t.Fatalf("finish board2 move: %s", fail)
	}
	if code := strOr(out2["new_board_code"]); code != "" {
		_, _ = s.DB.Exec(`UPDATE inv_box_code SET status='finished', weight=0, qty=0 WHERE code=?`, code)
	}
	if _, fail := s.closeBoard(b2, false); fail != "" {
		t.Fatalf("close board2: %s", fail)
	}
	var traceIn float64
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0) FROM pd_trace_process_yield WHERE trace_code='T-ISSUE' AND process_id=1`).Scan(&traceN, &traceIn)
	if traceN != 1 {
		t.Fatalf("trace snapshot one row after all boards closed, got %d", traceN)
	}
	if roundKg(traceIn) != 180 {
		t.Fatalf("trace input want 100+80=180 got %v", traceIn)
	}
	s.snapshotTraceYield("T-ISSUE")
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0) FROM pd_trace_process_yield WHERE trace_code='T-ISSUE' AND process_id=1`).Scan(&traceN, &traceIn)
	if traceN != 1 || roundKg(traceIn) != 180 {
		t.Fatalf("resnapshot must not double trace, rows=%d input=%v", traceN, traceIn)
	}
}

func TestBoardIssueRequiresTraceAndRejectsFinished(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := openIssueDB(t)
	if _, err := s.DB.Exec(`UPDATE inv_box_code SET trace_code='' WHERE code='BX-SMOKE'`); err != nil {
		t.Fatalf("clear trace: %v", err)
	}
	board, errMsg := s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" || board == nil {
		t.Fatalf("load board: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 10, 7); fail != "TRACE_CODE_REQUIRED" {
		t.Fatalf("issue without trace want TRACE_CODE_REQUIRED got %s", fail)
	}
	if _, fail := s.returnBoardKg(board, 7, 1); fail != "TRACE_CODE_REQUIRED" {
		t.Fatalf("return without trace want TRACE_CODE_REQUIRED got %s", fail)
	}
	if _, fail := s.moveBoardKg(board, 7, 10, "finish_in", 1, 1, 10); fail != "TRACE_CODE_REQUIRED" {
		t.Fatalf("move without trace want TRACE_CODE_REQUIRED got %s", fail)
	}
	if _, err := s.DB.Exec(`UPDATE inv_box_code SET trace_code='T-ISSUE', status='finished' WHERE code='BX-SMOKE'`); err != nil {
		t.Fatalf("mark finished: %v", err)
	}
	board, errMsg = s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" {
		t.Fatalf("reload: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 10, 7); fail != "BOARD_FINISHED" {
		t.Fatalf("issue finished want BOARD_FINISHED got %s", fail)
	}
}

func TestBoardCloseYieldSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := openIssueDB(t)
	board, errMsg := s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" || board == nil {
		t.Fatalf("load board: %s", errMsg)
	}
	if _, err := s.DB.Exec(`UPDATE inv_box_code SET weight=90, qty=90 WHERE code='BX-SMOKE'`); err != nil {
		t.Fatalf("set weight: %v", err)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 90, 7); fail != "" {
		t.Fatalf("issue: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.moveBoardKg(board, 7, 80, "finish_in", 1, 1, 10); fail != "" {
		t.Fatalf("partial finish: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if board.Status == "finished" {
		t.Fatalf("partial move must not finish board")
	}
	if remain := s.boardProcessRemainKg(board.ID, 1); roundKg(remain) != 10 {
		t.Fatalf("remain want 10 got %v", remain)
	}
	var yieldN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield WHERE board_id=? AND process_id=1`, board.ID).Scan(&yieldN)
	if yieldN != 0 {
		t.Fatalf("no board snapshot before close, got %d", yieldN)
	}
	if locked := s.workerLockedPieceworkKg(7, 1); roundKg(locked) != 90 {
		t.Fatalf("locked before writeoff want 90 got %v", locked)
	}

	board, _ = s.loadBoardByCode("BX-SMOKE")
	out, fail := s.closeBoard(board, false)
	if fail != "REMAIN_NEEDS_DECISION" {
		t.Fatalf("close without confirm_loss want REMAIN_NEEDS_DECISION got %s", fail)
	}
	if roundKg(asFloatOr0(out["total_remain_kg"])) != 10 {
		t.Fatalf("remain preview want 10 got %v", out["total_remain_kg"])
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if board.Status == "finished" {
		t.Fatalf("unconfirmed close must not finish")
	}

	var pwBefore float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwBefore)

	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.closeBoard(board, true); fail != "" {
		t.Fatalf("close confirm_loss: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if board.Status != "finished" {
		t.Fatalf("confirm_loss must finish board, got %s", board.Status)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield WHERE board_id=?`, board.ID).Scan(&yieldN)
	if yieldN != 0 {
		t.Fatalf("close must not write board yield, got %d", yieldN)
	}
	if locked := s.workerLockedPieceworkKg(7, 1); roundKg(locked) != 0 {
		t.Fatalf("writeoff must clear piecework lock, got %v", locked)
	}
	var input, output, loss float64
	var traceN int
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0), COALESCE(SUM(output_kg),0), COALESCE(SUM(loss_kg),0)
		FROM pd_trace_process_yield WHERE trace_code='T-ISSUE' AND process_id=1`).Scan(&traceN, &input, &output, &loss)
	if traceN != 1 {
		t.Fatalf("single-board trace must snapshot after close, got %d", traceN)
	}
	if roundKg(input) != 90 {
		t.Fatalf("input want 90 got %v", input)
	}
	if roundKg(output) != 80 {
		t.Fatalf("output want 80 got %v", output)
	}
	if roundKg(loss) != 10 {
		t.Fatalf("loss want 10 got %v", loss)
	}
	var pwAfter float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwAfter)
	if roundKg(pwAfter) != roundKg(pwBefore) {
		t.Fatalf("writeoff must not add piecework, before=%v after=%v", pwBefore, pwAfter)
	}

	_, _ = s.DB.Exec(`DELETE FROM inv_box_code WHERE code='BX-LB-2'`)
	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, qty, weight, status, current_process_id, current_step_id, trace_code)
		VALUES('BX-LB-2',1,1,100,100,'open',1,10,'T-LB')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	b2, errMsg := s.loadBoardByCode("BX-LB-2")
	if errMsg != "" {
		t.Fatalf("load: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(b2, 7, 1, 10, 100, 7); fail != "" {
		t.Fatalf("issue b2: %s", fail)
	}
	b2, _ = s.loadBoardByCode("BX-LB-2")
	if _, fail := s.moveBoardKg(b2, 7, 80, "finish_in", 1, 1, 10); fail != "" {
		t.Fatalf("partial b2: %s", fail)
	}
	b2, _ = s.loadBoardByCode("BX-LB-2")
	if _, fail := s.closeBoard(b2, true); fail != "" {
		t.Fatalf("close b2: %s", fail)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield WHERE board_id=?`, b2.ID).Scan(&yieldN)
	if yieldN != 0 {
		t.Fatalf("b2 must not write board yield, got %d", yieldN)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0), COALESCE(SUM(output_kg),0), COALESCE(SUM(loss_kg),0)
		FROM pd_trace_process_yield WHERE trace_code='T-LB' AND process_id=1`).Scan(&traceN, &input, &output, &loss)
	if traceN != 1 || roundKg(input) != 100 || roundKg(output) != 80 || roundKg(loss) != 20 {
		t.Fatalf("trace T-LB want in100 out80 loss20 got n=%d in=%v out=%v loss=%v", traceN, input, output, loss)
	}
	s.snapshotTraceYield("T-LB")
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0) FROM pd_trace_process_yield WHERE trace_code='T-LB' AND process_id=1`).Scan(&traceN, &input)
	if traceN != 1 || roundKg(input) != 100 {
		t.Fatalf("resnapshot must not double, rows=%d input=%v", traceN, input)
	}
}

func TestBoardStockInNewCodeAndReissue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := openIssueDB(t)
	board, errMsg := s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" || board == nil {
		t.Fatalf("load board: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 40, 7); fail != "" {
		t.Fatalf("issue: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	out, fail := s.moveBoardKg(board, 7, 40, "stock_in", 1, 1, 10)
	if fail != "" {
		t.Fatalf("stock_in: %s", fail)
	}
	newCode := strOr(out["new_board_code"])
	if newCode == "" {
		t.Fatal("stock_in must return new_board_code")
	}
	if kind := strOr(out["move_kind"]); kind != "stock_in" {
		t.Fatalf("move_kind want stock_in got %s", kind)
	}
	child, errMsg := s.loadBoardByCode(newCode)
	if errMsg != "" || child == nil {
		t.Fatalf("load new board: %s", errMsg)
	}
	if child.ProcessID != 1 || child.StepID != 10 {
		t.Fatalf("new board must mark completed process/step 1/10 got %d/%d", child.ProcessID, child.StepID)
	}
	if roundKg(child.Weight) != 40 {
		t.Fatalf("new board weight want 40 got %v", child.Weight)
	}
	var parentID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(parent_box_id,0) FROM inv_box_code WHERE code=?`, newCode).Scan(&parentID)
	if parentID != board.ID {
		t.Fatalf("parent_box_id want %d got %d", board.ID, parentID)
	}
	var bal float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM inv_balance WHERE box_code_id=?`, child.ID).Scan(&bal)
	if roundKg(bal) != 40 {
		t.Fatalf("box balance want 40 got %v", bal)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if roundKg(board.Weight+s.processOpenKg(board.ID, 1)) > 60.0005 {
		t.Fatalf("source wip after stock_in should drop by 40, weight=%v open=%v", board.Weight, s.processOpenKg(board.ID, 1))
	}

	// Re-issue into next process from warehouse buffer board (manual process choice).
	if _, fail := s.issueBoardKg(child, 8, 2, 11, 40, 8); fail != "" {
		t.Fatalf("reissue next process: %s", fail)
	}
	child, _ = s.loadBoardByCode(newCode)
	if child.ProcessID != 2 || child.StepID != 11 {
		t.Fatalf("after reissue current want process 2 step 11 got %d/%d", child.ProcessID, child.StepID)
	}
	if got := s.workerOpenKg(child.ID, 2, 8); roundKg(got) != 40 {
		t.Fatalf("worker open on next want 40 got %v", got)
	}
}
