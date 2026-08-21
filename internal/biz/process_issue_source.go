package biz

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// issueSourceOf returns warehouse|process.
func issueSourceOf(body map[string]interface{}, hasBoard bool) string {
	src := strings.ToLower(strings.TrimSpace(strOrDef(body["source"], strOr(body["issue_source"]))))
	switch src {
	case "warehouse", "wh", "stock":
		return "warehouse"
	case "process", "wip", "line":
		return "process"
	}
	if hasBoard {
		return "warehouse"
	}
	return "process"
}

func (s *Services) handleBoardIssueHTTP(c *gin.Context) bool {
	body := bindBody(c)
	boardCode := strings.TrimSpace(strOrDef(body["board_code"], strOr(body["box_code"])))
	trace := strings.ToUpper(strings.TrimSpace(strOrDef(body["trace_code"], strOr(body["code"]))))
	source := issueSourceOf(body, boardCode != "")

	kg, _ := asFloat(body["kg"])
	if kg <= 0 {
		kg, _ = asFloat(body["qty"])
	}
	kg = roundKg(kg)
	if kg <= 0 {
		api.FailJSON(c, "INVALID_QTY")
		return true
	}
	workerID, workerName, badge, errMsg := s.resolveScanWorker(c, body)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	processID, stepID, errMsg := s.requireBodyProcessID(body)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	opEmpID, _, _, _ := s.currentEmployeeInfo(c)
	if opEmpID <= 0 {
		api.FailJSON(c, "EMPLOYEE_REQUIRED")
		return true
	}
	if workerID != opEmpID && !s.claimsHasAnyRole(c, "foreman") {
		api.FailJSON(c, "ROLE_FORBIDDEN")
		return true
	}
	if !s.workerShiftAuthorized(workerID, processID) {
		api.FailJSON(c, "SHIFT_NOT_AUTHORIZED")
		return true
	}

	var out gin.H
	var fail string
	var board *boardState

	if source == "warehouse" {
		if boardCode == "" {
			api.FailJSON(c, "BOX_REQUIRED")
			return true
		}
		board, errMsg = s.loadBoardByCode(boardCode)
		if errMsg != "" {
			api.FailJSON(c, errMsg)
			return true
		}
		s.sanitizeBoardProcessRefs(board)
		if strings.TrimSpace(board.Trace) == "" {
			api.FailJSON(c, "TRACE_CODE_REQUIRED")
			return true
		}
		if board.Status == "finished" {
			api.FailJSON(c, "BOARD_FINISHED")
			return true
		}
		if trace != "" && strings.ToUpper(strings.TrimSpace(board.Trace)) != trace {
			api.FailJSON(c, "TRACE_MISMATCH")
			return true
		}
		trace = strings.ToUpper(strings.TrimSpace(board.Trace))
		if fail = s.requireTraceProductionOpen(trace); fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		if fail = s.assertProcessTransitionAllowed(board, processID); fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		prevProcess := board.ProcessID
		out, fail = s.issueBoardKg(board, workerID, processID, stepID, kg, opEmpID)
		if fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		issueID := asInt64Or0(out["id"])
		_, _ = s.DB.Exec(`UPDATE pd_process_issue SET source='warehouse' WHERE id=?`, issueID)
		if err := s.writeProcessIssueStockOut(board, kg, issueID, trace); err != nil {
			api.FailJSON(c, "STOCK_OUT_FAILED:"+err.Error())
			return true
		}
		if prevProcess > 0 && prevProcess != processID {
			s.creditUpstreamOutputOnIssue(&boardState{
				ID: board.ID, Code: board.Code, Trace: board.Trace, ProcessID: prevProcess, StepID: board.StepID,
			}, processID, kg, workerID)
		}
		out["source"] = "warehouse"
		s.attachBoardPreview(out, board.ID, board.Code, processID, stepID, workerID)
	} else {
		if trace == "" {
			api.FailJSON(c, "TRACE_CODE_REQUIRED")
			return true
		}
		if fail = s.requireTraceProductionOpen(trace); fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		out, fail = s.issueTraceProcessKg(trace, workerID, processID, stepID, kg, opEmpID)
		if fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		out["source"] = "process"
		out["trace_code"] = trace
		out["board_id"] = 0
		out["board_code"] = ""
	}

	out["worker_id"] = workerID
	out["worker_name"] = workerName
	out["badge_code"] = badge
	out["issued_by_employee_id"] = opEmpID
	out["is_proxy"] = opEmpID != workerID
	rate, _ := asFloat(out["rate"])
	amt, _ := asFloat(out["issue_locked_wage_amount"])
	boardID := int64(0)
	boardCodeOut := ""
	if board != nil {
		boardID = board.ID
		boardCodeOut = board.Code
		trace = board.Trace
	} else {
		trace = strings.ToUpper(strings.TrimSpace(strOrDef(out["trace_code"], trace)))
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "issue", BoardID: boardID, BoardCode: boardCodeOut, TraceCode: trace,
		ProcessID: processID, StepID: stepID, WorkerID: workerID, WorkerName: workerName, Badge: badge,
		ActorUserID: claimsUserID(c), OperatorEmployeeID: opEmpID, Kg: kg, Rate: rate, Amount: amt,
		RefType: "pd_process_issue", RefID: asInt64Or0(out["id"]),
		Payload: gin.H{"source": source, "is_proxy": opEmpID != workerID},
		After:   out,
	})
	api.OK(c, out)
	return true
}

func (s *Services) attachIssueWagePreview(dst gin.H, processID int64, kg float64) {
	if dst == nil {
		return
	}
	workerID := asInt64Or0(dst["worker_id"])
	if !s.shouldLockYieldWage(processID, workerID) && workerID == 0 {
		// still show rate when worker known later; try without worker gate
		if !s.shouldLockYieldWage(processID, 1) {
			return
		}
	}
	if s.shouldLockYieldWage(processID, workerID) || s.shouldLockYieldWage(processID, asInt64Or0(dst["issued_by_employee_id"])) {
		rate := s.processWageRate(processID)
		dst["piecework"] = true
		dst["pay_mode"] = s.processPayMode(processID)
		dst["rate"] = rate
		dst["issue_locked_kg"] = kg
		dst["issue_locked_wage_amount"] = roundMoney(kg * rate)
		dst["piecework_status"] = "locked"
		dst["piecework_hint"] = "预估工钱，确认结束后日结入账"
	}
}

// issueTraceProcessKg issues from trace-level WIP pool (worker_id=0, any board including 0).
func (s *Services) issueTraceProcessKg(trace string, workerID, processID, stepID int64, kg float64, issuedBy int64) (gin.H, string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" || workerID <= 0 || processID <= 0 || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if issuedBy <= 0 {
		issuedBy = workerID
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()

	var pool float64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`,
		trace, processID).Scan(&pool)
	if kg-roundKg(pool) > kgEps {
		return nil, "QTY_EXCEEDS_AVAILABLE"
	}
	if fail := takeTracePoolTx(tx, trace, processID, kg); fail != "" {
		return nil, fail
	}
	res, err := tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status, biz_status, issued_by_employee_id, source)
		VALUES(0,'',?,?,?,?,?,0,0,'open','open',?,'process')`,
		trace, processID, stepID, workerID, kg, issuedBy)
	if err != nil {
		res, err = tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status, biz_status, issued_by_employee_id)
			VALUES(0,'',?,?,?,?,?,0,0,'open','open',?)`,
			trace, processID, stepID, workerID, kg, issuedBy)
		if err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
	}
	iid, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	out := gin.H{
		"id": iid, "issue_kg": kg, "action": "issue", "issued_by_employee_id": issuedBy,
		"worker_id": workerID, "process_id": processID, "trace_code": trace, "source": "process",
	}
	s.attachIssueWagePreview(out, processID, kg)
	return out, ""
}

func takeTracePoolTx(tx *sql.Tx, trace string, processID int64, kg float64) string {
	need := kg
	rows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0
		ORDER BY created_at, id`, trace, processID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	type poolRow struct {
		id                         int64
		issue, returned, completed float64
	}
	list := []poolRow{}
	for rows.Next() {
		var r poolRow
		if err := rows.Scan(&r.id, &r.issue, &r.returned, &r.completed); err != nil {
			continue
		}
		list = append(list, r)
	}
	rows.Close()
	for _, r := range list {
		if need <= kgEps {
			break
		}
		rem := issueRemain(r.issue, r.returned, r.completed)
		if rem <= kgEps {
			_, _ = tx.Exec(`UPDATE pd_process_issue SET status='closed', updated_at=NOW() WHERE id=?`, r.id)
			continue
		}
		take := math.Min(rem, need)
		newIssue := roundKg(r.issue - take)
		st := "open"
		if issueRemain(newIssue, r.returned, r.completed) <= kgEps {
			st = "closed"
		}
		if _, err := tx.Exec(`UPDATE pd_process_issue SET issue_kg=?, status=?, updated_at=NOW() WHERE id=?`, newIssue, st, r.id); err != nil {
			return "DB_ERROR:" + err.Error()
		}
		need = roundKg(need - take)
	}
	if need > kgEps {
		return "QTY_EXCEEDS_AVAILABLE"
	}
	return ""
}

func (s *Services) writeProcessIssueStockOut(board *boardState, kg float64, issueID int64, trace string) error {
	if board == nil || kg <= kgEps {
		return nil
	}
	wh := board.WarehouseID
	if wh <= 0 {
		wh = 1
	}
	pid := board.ProductID
	if pid <= 0 {
		pid = 1
	}
	bizDate := time.Now().Format("2006-01-02")
	docNo := fmt.Sprintf("PIO%d", time.Now().UnixNano()%1e12)
	remark := fmt.Sprintf("process_issue_out board=%s trace=%s issue=%d", board.Code, trace, issueID)
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
		docNo, "process_issue_out", bizDate, wh, remark)
	if err != nil {
		return err
	}
	tid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'out',?)`,
		tid, 1, pid, kg, kg, trace)
	if board.ID > 0 {
		var bid int64
		var onHand float64
		err = s.DB.QueryRow(`SELECT id, qty FROM inv_balance WHERE warehouse_id=? AND product_id=? AND COALESCE(box_code_id,0)=? LIMIT 1`,
			wh, pid, board.ID).Scan(&bid, &onHand)
		if err == nil {
			newQty := onHand - kg
			if newQty < -0.0001 {
				return fmt.Errorf("INSUFFICIENT_STOCK")
			}
			_, err = s.DB.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, newQty, bid)
			return err
		}
	}
	return s.adjustBalance(wh, pid, -kg)
}
