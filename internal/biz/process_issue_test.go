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
			status TEXT DEFAULT 'open', created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW())`,
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
		`ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS is_deleted INTEGER DEFAULT 0`,
		`UPDATE inv_box_code SET current_process_id=1, current_step_id=10, weight=100, qty=100, trace_code='T-ISSUE' WHERE code='BX-SMOKE'`,
		`INSERT INTO pay_process_wage_rate(process_id, rate, status) SELECT 1, 0.5, 'active'
			WHERE NOT EXISTS (SELECT 1 FROM pay_process_wage_rate WHERE process_id=1)`,
		`UPDATE pd_process SET is_piecework=1 WHERE id=1`,
		`UPDATE pd_routing_step SET is_piecework=1, auto_next=1 WHERE id=10`,
		`INSERT INTO hr_employee(id, name, badge_code, emp_no, status) VALUES
			(7,'工人甲','BADGE-A','E7','active') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO hr_employee(id, name, badge_code, emp_no, status) VALUES
			(8,'工人乙','BADGE-B','E8','active') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO hr_employee(id, name, badge_code, emp_no, status) VALUES
			(9,'工人丙','BADGE-C','E9','active') ON CONFLICT (id) DO NOTHING`,
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
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 30); fail != "" {
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
	if _, fail := s.issueBoardKg(board, 8, 1, 10, 20); fail != "" {
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

	out, fail := s.moveBoardKg(board, 9, 30, "next", 1)
	if fail != "" {
		t.Fatalf("move: %s", fail)
	}
	if roundKg(asFloatOr0(out["settled_kg"])) != 30 {
		t.Fatalf("settled kg want 30 got %v", out["settled_kg"])
	}
	if roundKg(asFloatOr0(out["settled_wage_amount"])) != 15 {
		t.Fatalf("settled wage want 15 got %v", out["settled_wage_amount"])
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	fromWip := roundKg(board.Weight + s.processOpenKg(board.ID, 1) + s.poolOpenKg(board.ID, 1))
	if board.ProcessID == 1 && fromWip != 70 {
		t.Fatalf("from wip want 70 got %v (process=%d weight=%v)", fromWip, board.ProcessID, board.Weight)
	}
	toOpen := s.processOpenKg(board.ID, 2)
	if roundKg(toOpen) != 30 {
		t.Fatalf("to occupancy want 30 got %v", toOpen)
	}

	var pwAmt float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0), COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwQty, &pwAmt)
	if roundKg(pwQty) != 30 {
		t.Fatalf("piecework summary qty want 30 (completed only) got %v", pwQty)
	}
	if roundKg(pwAmt) != 15 { // 30 * 0.5
		t.Fatalf("piecework amount want 15 got %v", pwAmt)
	}

	if _, fail := s.returnBoardKg(board, 9, 5); fail != "" {
		t.Fatalf("next worker return: %s", fail)
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=1`).Scan(&pwQty)
	if roundKg(pwQty) != 30 {
		t.Fatalf("return must not change from-process piecework, got %v", pwQty)
	}
	var toPW float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE process_id=2`).Scan(&toPW)
	if toPW != 0 {
		t.Fatalf("to-process piecework should wait for later move, got %v", toPW)
	}
	var yieldN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield WHERE board_id=?`, board.ID).Scan(&yieldN)
	if yieldN != 0 {
		t.Fatalf("unfinished board must not snapshot yield, got %d", yieldN)
	}
}

func TestBoardYieldSnapshotOnceAndNoDoubleCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := openIssueDB(t)
	board, errMsg := s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" || board == nil {
		t.Fatalf("load board: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 30); fail != "" {
		t.Fatalf("issue A: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.issueBoardKg(board, 8, 1, 10, 20); fail != "" {
		t.Fatalf("issue B: %s", fail)
	}
	var inKg float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg),0) FROM pd_process_issue WHERE board_id=? AND process_id=1 AND worker_id>0`, board.ID).Scan(&inKg)
	if roundKg(inKg) != 50 {
		t.Fatalf("two issues must sum kg=50, got %v", inKg)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 60); fail != "QTY_EXCEEDS_AVAILABLE" {
		t.Fatalf("cannot re-issue beyond remaining, got %s", fail)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 50); fail != "" {
		t.Fatalf("issue rest: %s", fail)
	}
	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, qty, weight, status, current_process_id, current_step_id, trace_code)
		VALUES('BX-SMOKE-2',1,1,80,80,'open',1,10,'T-ISSUE')`)
	if err != nil {
		t.Fatalf("insert board2: %v", err)
	}
	s.snapshotBoardYield(board.ID)
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_board_process_yield`).Scan(&n)
	if n != 0 {
		t.Fatalf("yield before close want 0 got %d", n)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.moveBoardKg(board, 7, 100, "finish_in", 1); fail != "" {
		t.Fatalf("finish move: %s", fail)
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
	var input, output, loss float64
	_ = s.DB.QueryRow(`SELECT input_kg, output_kg, loss_kg FROM pd_board_process_yield WHERE board_id=? AND process_id=1`, board.ID).
		Scan(&input, &output, &loss)
	if roundKg(input) != 100 {
		t.Fatalf("snapshot input want 100 (30+20+50) got %v", input)
	}
	if roundKg(output) != 100 {
		t.Fatalf("snapshot output want 100 got %v", output)
	}
	if roundKg(loss) != 0 {
		t.Fatalf("snapshot loss want 0 got %v", loss)
	}
	s.snapshotBoardYield(board.ID)
	s.snapshotBoardYield(board.ID)
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0) FROM pd_board_process_yield WHERE board_id=? AND process_id=1`, board.ID).Scan(&n, &input)
	if n != 1 {
		t.Fatalf("snapshot must write once, rows=%d", n)
	}
	if roundKg(input) != 100 {
		t.Fatalf("resnapshot must not double, input=%v", input)
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
	if _, fail := s.issueBoardKg(b2, 7, 1, 10, 80); fail != "" {
		t.Fatalf("issue board2: %s", fail)
	}
	b2, _ = s.loadBoardByCode("BX-SMOKE-2")
	if _, fail := s.moveBoardKg(b2, 7, 80, "finish_in", 1); fail != "" {
		t.Fatalf("finish board2 move: %s", fail)
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
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 10); fail != "TRACE_CODE_REQUIRED" {
		t.Fatalf("issue without trace want TRACE_CODE_REQUIRED got %s", fail)
	}
	if _, fail := s.returnBoardKg(board, 7, 1); fail != "TRACE_CODE_REQUIRED" {
		t.Fatalf("return without trace want TRACE_CODE_REQUIRED got %s", fail)
	}
	if _, fail := s.moveBoardKg(board, 7, 10, "finish_in", 1); fail != "TRACE_CODE_REQUIRED" {
		t.Fatalf("move without trace want TRACE_CODE_REQUIRED got %s", fail)
	}
	if _, err := s.DB.Exec(`UPDATE inv_box_code SET trace_code='T-ISSUE', status='finished' WHERE code='BX-SMOKE'`); err != nil {
		t.Fatalf("mark finished: %v", err)
	}
	board, errMsg = s.loadBoardByCode("BX-SMOKE")
	if errMsg != "" {
		t.Fatalf("reload: %s", errMsg)
	}
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 10); fail != "BOARD_FINISHED" {
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
	if _, fail := s.issueBoardKg(board, 7, 1, 10, 90); fail != "" {
		t.Fatalf("issue: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.moveBoardKg(board, 7, 80, "finish_in", 1); fail != "" {
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
		t.Fatalf("no snapshot before close, got %d", yieldN)
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
	if roundKg(pwBefore) != 80 {
		t.Fatalf("piecework after 80kg move want 80 got %v", pwBefore)
	}

	board, _ = s.loadBoardByCode("BX-SMOKE")
	if _, fail := s.closeBoard(board, true); fail != "" {
		t.Fatalf("close confirm_loss: %s", fail)
	}
	board, _ = s.loadBoardByCode("BX-SMOKE")
	if board.Status != "finished" {
		t.Fatalf("confirm_loss must finish board, got %s", board.Status)
	}
	var input, output, loss float64
	_ = s.DB.QueryRow(`SELECT input_kg, output_kg, loss_kg FROM pd_board_process_yield WHERE board_id=? AND process_id=1`, board.ID).
		Scan(&input, &output, &loss)
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
	if roundKg(pwAfter) != 80 {
		t.Fatalf("writeoff must not add piecework, got %v", pwAfter)
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
	if _, fail := s.issueBoardKg(b2, 7, 1, 10, 100); fail != "" {
		t.Fatalf("issue b2: %s", fail)
	}
	b2, _ = s.loadBoardByCode("BX-LB-2")
	if _, fail := s.moveBoardKg(b2, 7, 80, "finish_in", 1); fail != "" {
		t.Fatalf("partial b2: %s", fail)
	}
	b2, _ = s.loadBoardByCode("BX-LB-2")
	if _, fail := s.closeBoard(b2, true); fail != "" {
		t.Fatalf("close b2: %s", fail)
	}
	_ = s.DB.QueryRow(`SELECT input_kg, output_kg, loss_kg FROM pd_board_process_yield WHERE board_id=? AND process_id=1`, b2.ID).
		Scan(&input, &output, &loss)
	if roundKg(input) != 100 || roundKg(output) != 80 || roundKg(loss) != 20 {
		t.Fatalf("close want in100 out80 loss20 got in=%v out=%v loss=%v", input, output, loss)
	}
	s.snapshotBoardYield(b2.ID)
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(input_kg),0) FROM pd_board_process_yield WHERE board_id=? AND process_id=1`, b2.ID).Scan(&n, &input)
	if n != 1 || roundKg(input) != 100 {
		t.Fatalf("resnapshot must not double, rows=%d input=%v", n, input)
	}
}
