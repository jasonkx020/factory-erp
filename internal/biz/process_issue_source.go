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
		if trace == "" {
			api.FailJSON(c, "TRACE_CODE_REQUIRED")
			return true
		}
		trace = strings.ToUpper(strings.TrimSpace(trace))
		if fail = s.requireTraceProductionOpen(trace); fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		photo := strings.TrimSpace(strOrDef(body["photo_url"], strOr(body["image_url"])))
		out, fail = s.createWarehouseIssuePending(trace, workerID, processID, stepID, kg, opEmpID, photo)
		if fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		out["source"] = "warehouse"
	} else {
		if trace == "" {
			api.FailJSON(c, "TRACE_CODE_REQUIRED")
			return true
		}
		if fail = s.requireTraceProductionOpen(trace); fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		toProcessID := processID
		fromProcessID := asInt64Or0(body["from_process_id"])
		if fromProcessID <= 0 {
			if loc, ok := body["from_location"].(map[string]interface{}); ok {
				if strings.EqualFold(strings.TrimSpace(strOr(loc["location_type"])), "process") {
					fromProcessID = asInt64Or0(loc["process_id"])
				}
			}
		}
		if fromProcessID <= 0 {
			fromProcessID = toProcessID
		}
		out, fail = s.issueTraceProcessKg(trace, workerID, fromProcessID, toProcessID, stepID, kg, opEmpID)
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
	eventType := "issue"
	if source == "warehouse" {
		if p, ok := out["pending"].(bool); ok && p {
			eventType = "issue_apply"
		}
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: eventType, BoardID: boardID, BoardCode: boardCodeOut, TraceCode: trace,
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

// issueTraceProcessKg issues from trace-level WIP at fromProcess into worker holding at toProcess.
func (s *Services) issueTraceProcessKg(trace string, workerID, fromProcessID, toProcessID, stepID int64, kg float64, issuedBy int64) (gin.H, string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" || workerID <= 0 || toProcessID <= 0 || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if fromProcessID <= 0 {
		fromProcessID = toProcessID
	}
	if issuedBy <= 0 {
		issuedBy = workerID
	}
	if fail := s.assertTraceProcessWip(trace, fromProcessID, kg, true); fail != "" {
		return nil, fail
	}
	s.ensureProcessIssueLocationColumns()
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()

	if fail := takeTraceIssuableTx(tx, trace, fromProcessID, kg); fail != "" {
		return nil, fail
	}
	res, err := tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status, biz_status, issued_by_employee_id, source, from_location_type, from_process_id, to_process_id)
		VALUES(0,'',?,?,?,?,?,0,0,'open','open',?,'process','process',?,?)`,
		trace, toProcessID, stepID, workerID, kg, issuedBy, fromProcessID, toProcessID)
	if err != nil {
		res, err = tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status, biz_status, issued_by_employee_id, source)
			VALUES(0,'',?,?,?,?,?,0,0,'open','open',?,'process')`,
			trace, toProcessID, stepID, workerID, kg, issuedBy)
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
		"worker_id": workerID, "process_id": toProcessID, "from_process_id": fromProcessID,
		"to_process_id": toProcessID, "trace_code": trace, "source": "process",
	}
	s.attachIssueWagePreview(out, toProcessID, kg)
	return out, ""
}

func takeTraceIssuableTx(tx *sql.Tx, trace string, processID int64, kg float64) string {
	need := kg
	rows, err := tx.Query(`SELECT b.id, COALESCE(b.weight, b.qty, 0),
		COALESCE((SELECT SUM(l.qty) FROM inv_balance l WHERE l.box_code_id=b.id),0)
		FROM inv_box_code b
		WHERE COALESCE(b.is_deleted,0)=0 AND UPPER(COALESCE(b.trace_code,''))=UPPER(?)
		  AND COALESCE(b.current_process_id,0)=? AND COALESCE(b.status,'') NOT IN ('destroyed','void')
		ORDER BY b.id`, trace, processID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	type boardRow struct {
		id      int64
		w, bal  float64
	}
	boards := []boardRow{}
	for rows.Next() {
		var b boardRow
		if rows.Scan(&b.id, &b.w, &b.bal) != nil {
			continue
		}
		at := b.w
		if b.bal > at {
			at = b.bal
		}
		if at <= kgEps {
			continue
		}
		boards = append(boards, b)
	}
	rows.Close()
	for _, b := range boards {
		if need <= kgEps {
			break
		}
		at := b.w
		if b.bal > at {
			at = b.bal
		}
		take := math.Min(at, need)
		if fail := takeBoardProcessKgTx(tx, b.id, take); fail != "" {
			return fail
		}
		need = roundKg(need - take)
	}
	if need > kgEps {
		return takeTracePoolTx(tx, trace, processID, need)
	}
	return ""
}

func takeBoardProcessKgTx(tx *sql.Tx, boardID int64, take float64) string {
	if tx == nil || boardID <= 0 || take <= kgEps {
		return ""
	}
	var w float64
	if err := tx.QueryRow(`SELECT COALESCE(weight, qty, 0) FROM inv_box_code WHERE id=?`, boardID).Scan(&w); err != nil {
		return "DB_ERROR:" + err.Error()
	}
	fromW := math.Min(w, take)
	if fromW > kgEps {
		nw := roundKg(w - fromW)
		if _, err := tx.Exec(`UPDATE inv_box_code SET weight=?, qty=?, updated_at=NOW() WHERE id=?`, nw, nw, boardID); err != nil {
			return "DB_ERROR:" + err.Error()
		}
		take = roundKg(take - fromW)
	}
	if take <= kgEps {
		return ""
	}
	brows, err := tx.Query(`SELECT id, qty FROM inv_balance WHERE box_code_id=? AND qty > 0.0005 ORDER BY id`, boardID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	defer brows.Close()
	for brows.Next() && take > kgEps {
		var bid int64
		var qty float64
		if brows.Scan(&bid, &qty) != nil {
			continue
		}
		t := math.Min(qty, take)
		nq := roundKg(qty - t)
		if _, err := tx.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, nq, bid); err != nil {
			return "DB_ERROR:" + err.Error()
		}
		take = roundKg(take - t)
	}
	if take > kgEps {
		return "QTY_EXCEEDS_AVAILABLE"
	}
	return ""
}

func takeTracePoolTx(tx *sql.Tx, trace string, processID int64, kg float64) string {
	need := kg
	// Foreman-confirmed worker output (work_done) is issuable pool; consume via completed_kg.
	wrows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open'
		  AND COALESCE(biz_status,'')='work_done' AND COALESCE(worker_id,0)>0
		ORDER BY created_at, id`, trace, processID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	type poolRow struct {
		id                         int64
		issue, returned, completed float64
	}
	wlist := []poolRow{}
	for wrows.Next() {
		var r poolRow
		if err := wrows.Scan(&r.id, &r.issue, &r.returned, &r.completed); err != nil {
			continue
		}
		wlist = append(wlist, r)
	}
	wrows.Close()
	for _, r := range wlist {
		if need <= kgEps {
			break
		}
		rem := issueRemain(r.issue, r.returned, r.completed)
		if rem <= kgEps {
			_, _ = tx.Exec(`UPDATE pd_process_issue SET status='closed', updated_at=NOW() WHERE id=?`, r.id)
			continue
		}
		take := math.Min(rem, need)
		newDone := roundKg(r.completed + take)
		st := "open"
		if issueRemain(r.issue, r.returned, newDone) <= kgEps {
			st = "closed"
		}
		if _, err := tx.Exec(`UPDATE pd_process_issue SET completed_kg=?, status=?, updated_at=NOW() WHERE id=?`, newDone, st, r.id); err != nil {
			return "DB_ERROR:" + err.Error()
		}
		need = roundKg(need - take)
	}
	if need <= kgEps {
		return ""
	}
	rows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0
		  AND COALESCE(biz_status,'') NOT IN ('issue_pending_warehouse','return_pending','issue_rejected')
		ORDER BY created_at, id`, trace, processID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
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
		return takeTraceOccupiedTx(tx, trace, processID, need)
	}
	return ""
}

func takeTraceOccupiedTx(tx *sql.Tx, trace string, processID int64, kg float64) string {
	need := kg
	rows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open'
		  AND COALESCE(worker_id,0)>0
		  AND COALESCE(biz_status,'') NOT IN ('work_done','issue_pending_warehouse','return_pending','issue_rejected')
		ORDER BY created_at, id`, trace, processID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	type occRow struct {
		id                         int64
		issue, returned, completed float64
	}
	list := []occRow{}
	for rows.Next() {
		var r occRow
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
		newDone := roundKg(r.completed + take)
		st := "open"
		if issueRemain(r.issue, r.returned, newDone) <= kgEps {
			st = "closed"
		}
		if _, err := tx.Exec(`UPDATE pd_process_issue SET completed_kg=?, status=?, updated_at=NOW() WHERE id=?`, newDone, st, r.id); err != nil {
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

func (s *Services) createWarehouseIssuePending(trace string, workerID, processID, stepID int64, kg float64, issuedBy int64, photo string) (gin.H, string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" || workerID <= 0 || processID <= 0 || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if issuedBy <= 0 {
		issuedBy = workerID
	}
	var exist int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND worker_id=? AND biz_status='issue_pending_warehouse'`,
		trace, workerID).Scan(&exist)
	if exist > 0 {
		return nil, "ISSUE_PENDING_EXISTS"
	}
	s.ensureProcessIssueLocationColumns()
	res, err := s.DB.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id,
		issue_kg, returned_kg, completed_kg, status, biz_status, issued_by_employee_id, source,
		from_location_type, to_process_id, pending_reweigh_kg, pending_photo_url)
		VALUES(0,'',?,?,?,?,?,0,0,'open','issue_pending_warehouse',?,'warehouse','warehouse',?,?,?)`,
		trace, processID, stepID, workerID, kg, issuedBy, processID, kg, photo)
	if err != nil {
		res, err = s.DB.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id,
			issue_kg, returned_kg, completed_kg, status, biz_status, issued_by_employee_id, source, pending_reweigh_kg, pending_photo_url)
			VALUES(0,'',?,?,?,?,?,0,0,'open','issue_pending_warehouse',?,'warehouse',?,?)`,
			trace, processID, stepID, workerID, kg, issuedBy, kg, photo)
		if err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
	}
	iid, _ := res.LastInsertId()
	out := gin.H{
		"id": iid, "issue_kg": kg, "action": "issue_apply", "issued_by_employee_id": issuedBy,
		"worker_id": workerID, "process_id": processID, "to_process_id": processID,
		"trace_code": trace, "source": "warehouse", "biz_status": "issue_pending_warehouse", "pending": true,
	}
	s.attachIssueWagePreview(out, processID, kg)
	return out, ""
}

// completeWarehousePendingIssue deducts board/pool and finalizes a pending warehouse issue row.
func (s *Services) completeWarehousePendingIssue(pendingID int64, board *boardState, workerID, processID, stepID int64, kg float64, issuedBy int64) (string, *boardState) {
	if pendingID <= 0 || board == nil || workerID <= 0 || processID <= 0 || kg <= kgEps {
		return "INVALID_QTY", nil
	}
	if issuedBy <= 0 {
		issuedBy = workerID
	}
	s.ensureProcessIssueLocationColumns()
	tx, err := s.DB.Begin()
	if err != nil {
		return "DB_ERROR", nil
	}
	defer func() { _ = tx.Rollback() }()
	b, fail := s.reloadBoardTx(tx, board.ID)
	if fail != "" {
		return fail, nil
	}
	reentry := false
	if b.ProcessID != processID && b.Weight > kgEps {
		reentry = true
		if stepID <= 0 {
			stepID = b.StepID
		}
	}
	avail := 0.0
	if b.ProcessID == processID || reentry {
		avail = b.Weight
	}
	var pool float64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open'
		  AND (COALESCE(worker_id,0)=0 OR COALESCE(biz_status,'')='work_done')
		  AND COALESCE(biz_status,'') NOT IN ('issue_pending_warehouse','return_pending','issue_rejected')
		  AND id<>?`,
		b.ID, processID, pendingID).Scan(&pool)
	avail = roundKg(avail + pool)
	// 仓库领料：实物可能在 inv_balance（板码 weight 为 0）；与 box weight 取较大值避免重复计量。
	whKg := boardWarehouseKgTx(tx, b.ID)
	if whKg > avail {
		avail = whKg
	}
	if kg-avail > kgEps {
		return "QTY_EXCEEDS_AVAILABLE", nil
	}
	need := kg
	if (b.ProcessID == processID || reentry) && b.Weight > kgEps && need > kgEps {
		take := math.Min(b.Weight, need)
		b.Weight = roundKg(b.Weight - take)
		need = roundKg(need - take)
		if reentry || b.ProcessID != processID {
			if _, err := tx.Exec(`UPDATE inv_box_code SET current_process_id=?, current_step_id=?, weight=?, qty=?, updated_at=NOW() WHERE id=?`,
				processID, stepID, b.Weight, b.Weight, b.ID); err != nil {
				return "DB_ERROR:" + err.Error(), nil
			}
			b.ProcessID = processID
			b.StepID = stepID
		} else if _, err := tx.Exec(`UPDATE inv_box_code SET weight=?, qty=?, updated_at=NOW() WHERE id=?`, b.Weight, b.Weight, b.ID); err != nil {
			return "DB_ERROR:" + err.Error(), nil
		}
	}
	if need > kgEps {
		if whKg+kgEps >= need {
			// 库存台账在 inv_balance；出库扣减由 writeProcessIssueStockOut 完成。
			need = 0
		} else if fail := takePoolTx(tx, b.ID, processID, need); fail != "" {
			return fail, nil
		}
	}
	_, err = tx.Exec(`UPDATE pd_process_issue SET board_id=?, board_code=?, issue_kg=?, status='open', biz_status='open',
		source='warehouse', from_location_type='warehouse', to_process_id=?, assigned_board_code=?,
		pending_reweigh_kg=0, pending_photo_url='', updated_at=NOW() WHERE id=? AND biz_status='issue_pending_warehouse'`,
		b.ID, b.Code, kg, processID, b.Code, pendingID)
	if err != nil {
		return "DB_ERROR:" + err.Error(), nil
	}
	if err := tx.Commit(); err != nil {
		return "DB_ERROR:" + err.Error(), nil
	}
	board.Weight = b.Weight
	board.ProcessID = b.ProcessID
	board.StepID = b.StepID
	board.Code = b.Code
	board.Trace = b.Trace
	return "", board
}
