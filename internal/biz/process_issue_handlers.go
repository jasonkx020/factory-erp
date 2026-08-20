package biz

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

const kgEps = 0.0005

func roundKg(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func issueRemain(issueKg, returnedKg, completedKg float64) float64 {
	r := roundKg(issueKg - returnedKg - completedKg)
	if r < kgEps {
		return 0
	}
	return r
}

type boardState struct {
	ID, ProductID, WarehouseID, ProcessID, StepID, TaskID, WoID int64
	Code, Trace, Status                                         string
	Weight                                                      float64
}

func (s *Services) handleBoardIssues(c *gin.Context, method, openapiPath, action string) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	if method != "POST" {
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
	if strings.Contains(openapiPath, "/return") {
		return s.handleBoardReturnHTTP(c)
	}
	return s.handleBoardIssueHTTP(c)
}

func (s *Services) handleBoardMoves(c *gin.Context, method, openapiPath, action string) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	if method != "POST" {
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
	_ = openapiPath
	_ = action
	return s.handleBoardMoveHTTP(c)
}

func (s *Services) handleBoardClose(c *gin.Context, method, openapiPath, action string) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	if !s.requireAnyRole(c, "foreman") {
		return true
	}
	if method != "POST" {
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
	_ = action
	if strings.Contains(openapiPath, "/preview") {
		return s.handleBoardClosePreviewHTTP(c)
	}
	return s.handleBoardCloseHTTP(c)
}

func (s *Services) loadBoardForCloseFromBody(body map[string]interface{}) (*boardState, string) {
	code := strings.TrimSpace(strOr(body["board_code"]))
	if code == "" {
		code = strings.TrimSpace(strOr(body["box_code"]))
	}
	if code == "" {
		return nil, "BOX_REQUIRED"
	}
	board, errMsg := s.loadBoardByCode(code)
	if errMsg != "" {
		return nil, errMsg
	}
	if strings.TrimSpace(board.Trace) == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	return board, ""
}

func (s *Services) handleBoardClosePreviewHTTP(c *gin.Context) bool {
	board, errMsg := s.loadBoardForCloseFromBody(bindBody(c))
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	api.OK(c, s.boardCloseRemain(board))
	return true
}

func (s *Services) handleBoardCloseHTTP(c *gin.Context) bool {
	body := bindBody(c)
	board, errMsg := s.loadBoardForCloseFromBody(body)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	confirmLoss := boolOr(body["confirm_loss"], false)
	out, fail := s.closeBoard(board, confirmLoss)
	if fail == "REMAIN_NEEDS_DECISION" {
		api.HandleBusiness(c, &api.BusinessError{Msg: fail, Data: out}, nil)
		return true
	}
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "board_close", BoardID: board.ID, BoardCode: board.Code, TraceCode: board.Trace,
		ProcessID: board.ProcessID, StepID: board.StepID, ActorUserID: claimsUserID(c),
		Payload: gin.H{"confirm_loss": confirmLoss, "writeoff_kg": out["writeoff_kg"]},
		After:   out,
	})
	api.OK(c, out)
	return true
}

func (s *Services) parseBoardAction(c *gin.Context) (body map[string]interface{}, board *boardState, workerID int64, workerName, badge string, kg float64, errMsg string) {
	body = bindBody(c)
	code := strings.TrimSpace(strOr(body["board_code"]))
	if code == "" {
		code = strings.TrimSpace(strOr(body["box_code"]))
	}
	if code == "" {
		return body, nil, 0, "", "", 0, "BOX_REQUIRED"
	}
	kg, _ = asFloat(body["kg"])
	if kg <= 0 {
		kg, _ = asFloat(body["qty"])
	}
	kg = roundKg(kg)
	if kg <= 0 {
		return body, nil, 0, "", "", 0, "INVALID_QTY"
	}
	workerID, workerName, badge, errMsg = s.resolveScanWorker(c, body)
	if errMsg != "" {
		return body, nil, 0, "", "", 0, errMsg
	}
	board, errMsg = s.loadBoardByCode(code)
	if errMsg != "" {
		return body, nil, 0, "", "", 0, errMsg
	}
	s.sanitizeBoardProcessRefs(board)
	if strings.TrimSpace(board.Trace) == "" {
		return body, nil, 0, "", "", 0, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return body, nil, 0, "", "", 0, "BOARD_FINISHED"
	}
	return body, board, workerID, workerName, badge, kg, ""
}

func (s *Services) requireBodyProcessID(body map[string]interface{}) (processID, stepID int64, errMsg string) {
	processID, _ = asInt64(body["process_id"])
	stepID, _ = asInt64(body["step_id"])
	if processID <= 0 {
		return 0, 0, "PROCESS_REQUIRED"
	}
	return processID, stepID, ""
}

func (s *Services) requireReweighFields(body map[string]interface{}) string {
	reweigh, _ := asFloat(body["reweigh_kg"])
	if reweigh <= kgEps {
		return "REWEIGH_REQUIRED"
	}
	photo := strings.TrimSpace(strOrDef(body["photo_url"], strOr(body["image_url"])))
	if photo == "" {
		return "REWEIGH_PHOTO_REQUIRED"
	}
	return ""
}

func (s *Services) handleBoardIssueHTTP(c *gin.Context) bool {
	body, board, workerID, workerName, badge, kg, errMsg := s.parseBoardAction(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	processID, stepID, errMsg := s.requireBodyProcessID(body)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	if fail := s.assertProcessTransitionAllowed(board, processID); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	if !s.workerShiftAuthorized(workerID, processID) {
		api.FailJSON(c, "SHIFT_NOT_AUTHORIZED")
		return true
	}
	out, fail := s.issueBoardKg(board, workerID, processID, stepID, kg)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	out["worker_id"] = workerID
	out["worker_name"] = workerName
	out["badge_code"] = badge
	s.attachBoardPreview(out, board.ID, board.Code, processID, stepID, workerID)
	rate, _ := asFloat(out["rate"])
	amt, _ := asFloat(out["issue_locked_wage_amount"])
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "issue", BoardID: board.ID, BoardCode: board.Code, TraceCode: board.Trace,
		ProcessID: processID, StepID: stepID, WorkerID: workerID, WorkerName: workerName, Badge: badge,
		ActorUserID: claimsUserID(c), Kg: kg, PayMode: s.processPayMode(processID), EmpType: s.workerEmpType(workerID),
		Rate: rate, Amount: amt, RefType: "pd_process_issue", RefID: asInt64Or0(out["id"]),
		After: out,
	})
	api.OK(c, out)
	return true
}

func (s *Services) handleBoardReturnHTTP(c *gin.Context) bool {
	body, board, workerID, workerName, badge, kg, errMsg := s.parseBoardAction(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	processID, stepID, errMsg := s.requireBodyProcessID(body)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	if !s.workerShiftAuthorized(workerID, processID) {
		api.FailJSON(c, "SHIFT_NOT_AUTHORIZED")
		return true
	}
	out, fail := s.returnBoardKg(board, workerID, kg)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	out["worker_id"] = workerID
	out["worker_name"] = workerName
	out["badge_code"] = badge
	s.attachBoardPreview(out, board.ID, board.Code, processID, stepID, workerID)
	rate, _ := asFloat(out["rate"])
	amt, _ := asFloat(out["released_locked_wage_amount"])
	if amt > 0 {
		amt = -amt
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "return", BoardID: board.ID, BoardCode: board.Code, TraceCode: board.Trace,
		ProcessID: processID, StepID: stepID, WorkerID: workerID, WorkerName: workerName, Badge: badge,
		ActorUserID: claimsUserID(c), Kg: kg, PayMode: s.processPayMode(processID), EmpType: s.workerEmpType(workerID),
		Rate: rate, Amount: amt, RefType: "pd_process_issue_return",
		After: out,
	})
	api.OK(c, out)
	return true
}

func (s *Services) handleBoardMoveHTTP(c *gin.Context) bool {
	body, board, workerID, workerName, badge, kg, errMsg := s.parseBoardAction(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	kind := strings.TrimSpace(strOrDef(body["move_kind"], "stock_in"))
	if kind == "finish" || kind == "finish_in" {
		kind = "stock_in"
	}
	if kind == "next" {
		api.FailJSON(c, "AUTO_ROUTING_DISABLED")
		return true
	}
	if kind != "stock_in" {
		api.FailJSON(c, "INVALID_MOVE_KIND")
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
	if !s.workerShiftAuthorized(workerID, processID) {
		api.FailJSON(c, "SHIFT_NOT_AUTHORIZED")
		return true
	}
	if pid := asInt64Or0(body["product_id"]); pid > 0 {
		board.ProductID = pid
	}
	createdBy := claimsUserID(c)
	out, fail := s.moveBoardKg(board, workerID, kg, kind, createdBy, processID, stepID)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	out["worker_id"] = workerID
	out["worker_name"] = workerName
	out["badge_code"] = badge
	s.attachBoardPreview(out, board.ID, board.Code, processID, stepID, workerID)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "stock_in", BoardID: board.ID, BoardCode: board.Code, TraceCode: board.Trace,
		ProcessID: processID, StepID: stepID, WorkerID: workerID, WorkerName: workerName, Badge: badge,
		ActorUserID: createdBy, Kg: kg, PayMode: s.processPayMode(processID), EmpType: s.workerEmpType(workerID),
		RefType: "pd_process_move", RefID: asInt64Or0(out["id"]),
		Payload: gin.H{"new_board_code": out["new_board_code"], "new_board_id": out["new_board_id"]},
		After:   out,
	})
	api.OK(c, out)
	return true
}

func (s *Services) loadBoardByCode(code string) (*boardState, string) {
	var b boardState
	err := s.DB.QueryRow(`SELECT id, COALESCE(code,''), COALESCE(product_id,0), COALESCE(warehouse_id,0),
		COALESCE(current_process_id,0), COALESCE(current_step_id,0), COALESCE(task_id,0), COALESCE(work_order_id,0),
		COALESCE(weight, qty, 0), COALESCE(trace_code,''), COALESCE(status,'')
		FROM inv_box_code WHERE code=? AND COALESCE(is_deleted,0)=0`, code).
		Scan(&b.ID, &b.Code, &b.ProductID, &b.WarehouseID, &b.ProcessID, &b.StepID, &b.TaskID, &b.WoID, &b.Weight, &b.Trace, &b.Status)
	if err != nil {
		return nil, "BOX_NOT_FOUND"
	}
	if b.Status == "destroyed" || b.Status == "void" {
		return nil, "BOX_NOT_FOUND"
	}
	return &b, ""
}

func (s *Services) reloadBoardTx(tx *sql.Tx, id int64) (*boardState, string) {
	var b boardState
	err := tx.QueryRow(`SELECT id, COALESCE(code,''), COALESCE(product_id,0), COALESCE(warehouse_id,0),
		COALESCE(current_process_id,0), COALESCE(current_step_id,0), COALESCE(task_id,0), COALESCE(work_order_id,0),
		COALESCE(weight, qty, 0), COALESCE(trace_code,''), COALESCE(status,'')
		FROM inv_box_code WHERE id=? FOR UPDATE`, id).
		Scan(&b.ID, &b.Code, &b.ProductID, &b.WarehouseID, &b.ProcessID, &b.StepID, &b.TaskID, &b.WoID, &b.Weight, &b.Trace, &b.Status)
	if err != nil {
		return nil, "BOX_NOT_FOUND"
	}
	return &b, ""
}

// sanitizeBoardProcessRefs clears stale routing step refs; does not auto-assign next/first step.
func (s *Services) sanitizeBoardProcessRefs(b *boardState) {
	if b == nil {
		return
	}
	if b.StepID > 0 && s.loadStep(b.StepID) == nil {
		b.StepID = 0
	}
}

// ensureBoardProcess kept for callers that only need stale-step cleanup (no routing auto-flow).
func (s *Services) ensureBoardProcess(b *boardState) {
	s.sanitizeBoardProcessRefs(b)
}

func (s *Services) boardAvailableKg(boardID, processID int64, currentProcessID int64, weight float64) float64 {
	avail := 0.0
	if currentProcessID == processID {
		avail += weight
	}
	avail += s.poolOpenKg(boardID, processID)
	return roundKg(avail)
}

func (s *Services) poolOpenKg(boardID, processID int64) float64 {
	var v float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`, boardID, processID).Scan(&v)
	return roundKg(v)
}

func (s *Services) processOpenKg(boardID, processID int64) float64 {
	var v float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0`, boardID, processID).Scan(&v)
	if v < kgEps {
		return 0
	}
	return roundKg(v)
}

func (s *Services) workerOpenKg(boardID, processID, workerID int64) float64 {
	if workerID <= 0 {
		return 0
	}
	var v float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND worker_id=? AND status='open'`, boardID, processID, workerID).Scan(&v)
	if v < kgEps {
		return 0
	}
	return roundKg(v)
}

func (s *Services) attachBoardPreview(dst gin.H, boardID int64, code string, processID, stepID, workerID int64) {
	var weight float64
	var trace string
	var curProc, curStep, productID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(weight, qty, 0), COALESCE(trace_code,''), COALESCE(current_process_id,0), COALESCE(current_step_id,0), COALESCE(product_id,0)
		FROM inv_box_code WHERE id=?`, boardID).Scan(&weight, &trace, &curProc, &curStep, &productID)
	if processID <= 0 {
		processID = curProc
	}
	if stepID <= 0 {
		stepID = curStep
	}
	step := s.loadStep(stepID)
	// Manual process selection: board weight is claimable into the chosen process (no routing auto-next).
	avail := roundKg(weight + s.poolOpenKg(boardID, processID))
	myOpen := s.workerOpenKg(boardID, processID, workerID)
	procOpen := s.processOpenKg(boardID, processID)
	dst["board_id"] = boardID
	dst["board_code"] = code
	dst["box_code"] = code
	dst["trace_code"] = trace
	dst["product_id"] = productID
	pname, pcat, pcode := s.productMeta(productID)
	dst["product_name"] = pname
	dst["product_category"] = pcat
	dst["product_code"] = pcode
	dst["process_id"] = processID
	dst["step_id"] = stepID
	dst["board_process_id"] = curProc
	dst["board_step_id"] = curStep
	dst["available_kg"] = avail
	dst["my_open_kg"] = myOpen
	dst["process_open_kg"] = procOpen
	dst["wip_kg"] = roundKg(avail + procOpen)
	dst["buffer_reentry"] = processID != curProc && weight > kgEps
	dst["has_next"] = false
	dst["can_next"] = false
	dst["can_stock_in"] = true
	if step != nil {
		dst["step_name"] = step.StepName
		dst["is_piecework"] = step.IsPiecework
		wh := step.WarehouseID
		if wh <= 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(warehouse_id,0) FROM inv_box_code WHERE id=?`, boardID).Scan(&wh)
		}
		if wh > 0 {
			dst["stock_in_warehouse_id"] = wh
		}
	} else if processID > 0 {
		dst["is_piecework"] = s.isPieceworkProcess(processID, 0)
		var wh int64
		_ = s.DB.QueryRow(`SELECT COALESCE(warehouse_id,0) FROM inv_box_code WHERE id=?`, boardID).Scan(&wh)
		if wh > 0 {
			dst["stock_in_warehouse_id"] = wh
		}
	}
	var processName string
	_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, processID).Scan(&processName)
	dst["process_name"] = processName
	if step == nil && processName != "" {
		dst["step_name"] = processName
	}
	s.attachPieceworkLockPreview(dst, workerID, processID, stepID)
}

func (s *Services) issueBoardKg(board *boardState, workerID, processID, stepID int64, kg float64) (gin.H, string) {
	if board == nil || workerID <= 0 || processID <= 0 || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if strings.TrimSpace(board.Trace) == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()
	b, fail := s.reloadBoardTx(tx, board.ID)
	if fail != "" {
		return nil, fail
	}
	// Manual process choice: board warehouse weight can be issued into any selected process.
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
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`, b.ID, processID).Scan(&pool)
	avail = roundKg(avail + pool)
	if kg-avail > kgEps {
		return nil, "QTY_EXCEEDS_AVAILABLE"
	}
	need := kg
	if (b.ProcessID == processID || reentry) && b.Weight > kgEps && need > kgEps {
		take := math.Min(b.Weight, need)
		b.Weight = roundKg(b.Weight - take)
		need = roundKg(need - take)
		if reentry || b.ProcessID != processID {
			if _, err := tx.Exec(`UPDATE inv_box_code SET current_process_id=?, current_step_id=?, weight=?, qty=?, updated_at=NOW() WHERE id=?`,
				processID, stepID, b.Weight, b.Weight, b.ID); err != nil {
				return nil, "DB_ERROR:" + err.Error()
			}
			b.ProcessID = processID
			b.StepID = stepID
		} else if _, err := tx.Exec(`UPDATE inv_box_code SET weight=?, qty=?, updated_at=NOW() WHERE id=?`, b.Weight, b.Weight, b.ID); err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
	}
	if need > kgEps {
		if fail := takePoolTx(tx, b.ID, processID, need); fail != "" {
			return nil, fail
		}
	}
	res, err := tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status)
		VALUES(?,?,?,?,?,?,?,0,0,'open')`, b.ID, b.Code, b.Trace, processID, stepID, workerID, kg)
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	iid, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	board.Weight = b.Weight
	board.ProcessID = b.ProcessID
	board.StepID = b.StepID
	out := gin.H{"id": iid, "issue_kg": kg, "action": "issue"}
	if s.shouldLockYieldWage(processID, workerID) {
		rate := s.processWageRate(processID)
		out["piecework"] = true
		out["pay_mode"] = s.processPayMode(processID)
		out["rate"] = rate
		out["issue_locked_kg"] = kg
		out["issue_locked_wage_amount"] = roundMoney(kg * rate)
		out["piecework_status"] = "locked"
		out["piecework_hint"] = "预估工钱，当日日结入账"
	}
	return out, ""
}

func takePoolTx(tx *sql.Tx, boardID, processID int64, kg float64) string {
	need := kg
	rows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0
		ORDER BY created_at, id`, boardID, processID)
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

func addPoolTx(tx *sql.Tx, b *boardState, processID, stepID int64, kg float64) string {
	if kg <= kgEps {
		return ""
	}
	var id int64
	var issue, returned, completed float64
	err := tx.QueryRow(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0
		ORDER BY id LIMIT 1`, b.ID, processID).Scan(&id, &issue, &returned, &completed)
	if err == nil && id > 0 {
		_, err = tx.Exec(`UPDATE pd_process_issue SET issue_kg=?, updated_at=NOW() WHERE id=?`, roundKg(issue+kg), id)
		if err != nil {
			return "DB_ERROR:" + err.Error()
		}
		return ""
	}
	_, err = tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status)
		VALUES(?,?,?,?,?,0,?,0,0,'open')`, b.ID, b.Code, b.Trace, processID, stepID, kg)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	return ""
}

func (s *Services) returnBoardKg(board *boardState, workerID int64, kg float64) (gin.H, string) {
	if board == nil || workerID <= 0 || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if strings.TrimSpace(board.Trace) == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()
	b, fail := s.reloadBoardTx(tx, board.ID)
	if fail != "" {
		return nil, fail
	}
	rows, err := tx.Query(`SELECT id, process_id, step_id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND worker_id=? AND status='open'
		ORDER BY created_at, id`, b.ID, workerID)
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	type iss struct {
		id, processID, stepID      int64
		issue, returned, completed float64
	}
	list := []iss{}
	for rows.Next() {
		var r iss
		if err := rows.Scan(&r.id, &r.processID, &r.stepID, &r.issue, &r.returned, &r.completed); err != nil {
			continue
		}
		list = append(list, r)
	}
	rows.Close()
	need := kg
	returnedTotal := 0.0
	byProcess := map[int64]float64{}
	stepByProcess := map[int64]int64{}
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
		newRet := roundKg(r.returned + take)
		st := "open"
		if issueRemain(r.issue, newRet, r.completed) <= kgEps {
			st = "closed"
		}
		if _, err := tx.Exec(`UPDATE pd_process_issue SET returned_kg=?, status=?, updated_at=NOW() WHERE id=?`, newRet, st, r.id); err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
		need = roundKg(need - take)
		returnedTotal = roundKg(returnedTotal + take)
		byProcess[r.processID] = roundKg(byProcess[r.processID] + take)
		stepByProcess[r.processID] = r.stepID
	}
	if need > kgEps {
		return nil, "QTY_EXCEEDS_OCCUPANCY"
	}
	for procID, addKg := range byProcess {
		if b.ProcessID == procID {
			b.Weight = roundKg(b.Weight + addKg)
			if _, err := tx.Exec(`UPDATE inv_box_code SET weight=?, qty=?, updated_at=NOW() WHERE id=?`, b.Weight, b.Weight, b.ID); err != nil {
				return nil, "DB_ERROR:" + err.Error()
			}
			continue
		}
		if fail := addPoolTx(tx, b, procID, stepByProcess[procID], addKg); fail != "" {
			return nil, fail
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	board.Weight = b.Weight
	out := gin.H{"returned_kg": returnedTotal, "action": "return"}
	if returnedTotal > kgEps {
		procID := int64(0)
		for pid := range byProcess {
			procID = pid
			break
		}
		if procID > 0 && s.shouldLockYieldWage(procID, workerID) {
			rate := s.processWageRate(procID)
			out["piecework"] = true
			out["pay_mode"] = s.processPayMode(procID)
			out["rate"] = rate
			out["released_locked_kg"] = returnedTotal
			out["released_locked_wage_amount"] = roundMoney(returnedTotal * rate)
			out["piecework_status"] = "locked"
			out["piecework_hint"] = "退库扣减预估工钱，未日结部分不入汇总"
		}
	}
	return out, ""
}

func (s *Services) boardProcessRemainKg(boardID, processID int64) float64 {
	var weight float64
	var curProc int64
	_ = s.DB.QueryRow(`SELECT COALESCE(weight, qty, 0), COALESCE(current_process_id,0)
		FROM inv_box_code WHERE id=? AND COALESCE(is_deleted,0)=0`, boardID).Scan(&weight, &curProc)
	remain := roundKg(s.processOpenKg(boardID, processID) + s.poolOpenKg(boardID, processID))
	if curProc == processID {
		remain = roundKg(remain + weight)
	}
	return remain
}

func boardProcessRemainKgTx(tx *sql.Tx, boardID, processID int64, curProc int64, weight float64) float64 {
	var occ, pool float64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0`, boardID, processID).Scan(&occ)
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`, boardID, processID).Scan(&pool)
	remain := roundKg(occ + pool)
	if curProc == processID {
		remain = roundKg(remain + weight)
	}
	return remain
}

// writeoffBoardProcessRemainTx closes all open occupancy at board+process without adding move output (loss, not piecework).
func writeoffBoardProcessRemainTx(tx *sql.Tx, b *boardState, processID int64) (float64, string) {
	rows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg, COALESCE(wage_settled_kg,0) FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' ORDER BY created_at, id`, b.ID, processID)
	if err != nil {
		return 0, "DB_ERROR:" + err.Error()
	}
	type iss struct {
		id                                  int64
		issue, returned, completed, settled float64
	}
	list := []iss{}
	for rows.Next() {
		var r iss
		if err := rows.Scan(&r.id, &r.issue, &r.returned, &r.completed, &r.settled); err != nil {
			continue
		}
		list = append(list, r)
	}
	rows.Close()
	writeoff := 0.0
	for _, r := range list {
		digest := roundKg(r.issue - r.returned)
		if digest < 0 {
			digest = 0
		}
		wageSettled := r.settled
		if digest > wageSettled {
			wageSettled = digest
		}
		rem := issueRemain(r.issue, r.returned, r.completed)
		if rem <= kgEps {
			_, _ = tx.Exec(`UPDATE pd_process_issue SET status='closed', wage_settled_kg=?, updated_at=NOW() WHERE id=?`, wageSettled, r.id)
			continue
		}
		newDone := digest
		if _, err := tx.Exec(`UPDATE pd_process_issue SET completed_kg=?, wage_settled_kg=?, status='closed', updated_at=NOW() WHERE id=?`,
			newDone, wageSettled, r.id); err != nil {
			return 0, "DB_ERROR:" + err.Error()
		}
		writeoff = roundKg(writeoff + rem)
	}
	if b.ProcessID == processID && b.Weight > kgEps {
		writeoff = roundKg(writeoff + b.Weight)
		b.Weight = 0
		if _, err := tx.Exec(`UPDATE inv_box_code SET weight=0, qty=0, updated_at=NOW() WHERE id=?`, b.ID); err != nil {
			return 0, "DB_ERROR:" + err.Error()
		}
	}
	return writeoff, ""
}

func (s *Services) moveBoardKg(board *boardState, toWorkerID int64, kg float64, kind string, createdBy, fromProcessID, fromStepIDHint int64) (gin.H, string) {
	if board == nil || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if strings.TrimSpace(board.Trace) == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	if kind == "finish" || kind == "finish_in" {
		kind = "stock_in"
	}
	if kind == "next" {
		return nil, "AUTO_ROUTING_DISABLED"
	}
	if kind != "stock_in" {
		return nil, "INVALID_MOVE_KIND"
	}
	fromProcess := fromProcessID
	if fromProcess <= 0 {
		fromProcess = board.ProcessID
	}
	if fromProcess <= 0 {
		return nil, "PROCESS_REQUIRED"
	}
	fromStepID := fromStepIDHint
	if fromStepID <= 0 {
		fromStepID = board.StepID
	}
	fromStep := s.loadStep(fromStepID)
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()
	productOverride := board.ProductID
	b, fail := s.reloadBoardTx(tx, board.ID)
	if fail != "" {
		return nil, fail
	}
	if productOverride > 0 {
		b.ProductID = productOverride
	}
	wip := 0.0
	if b.ProcessID == fromProcess {
		wip += b.Weight
	}
	var occ, pool float64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0`, b.ID, fromProcess).Scan(&occ)
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`, b.ID, fromProcess).Scan(&pool)
	wip = roundKg(wip + occ + pool)
	if kg-wip > kgEps {
		return nil, "QTY_EXCEEDS_WIP"
	}

	type alloc struct {
		issueID, workerID int64
		kg                float64
		piece             bool
	}
	allocs := []alloc{}
	need := kg
	rows, err := tx.Query(`SELECT id, worker_id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0
		ORDER BY created_at, id`, b.ID, fromProcess)
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	type iss struct {
		id, workerID               int64
		issue, returned, completed float64
	}
	list := []iss{}
	for rows.Next() {
		var r iss
		if err := rows.Scan(&r.id, &r.workerID, &r.issue, &r.returned, &r.completed); err != nil {
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
			return nil, "DB_ERROR:" + err.Error()
		}
		allocs = append(allocs, alloc{issueID: r.id, workerID: r.workerID, kg: take, piece: true})
		need = roundKg(need - take)
	}
	if need > kgEps && b.ProcessID == fromProcess && b.Weight > kgEps {
		take := math.Min(b.Weight, need)
		b.Weight = roundKg(b.Weight - take)
		need = roundKg(need - take)
		if _, err := tx.Exec(`UPDATE inv_box_code SET weight=?, qty=?, updated_at=NOW() WHERE id=?`, b.Weight, b.Weight, b.ID); err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
	}
	if need > kgEps {
		if fail := takePoolTx(tx, b.ID, fromProcess, need); fail != "" {
			return nil, fail
		}
		need = 0
	}

	issueIDs := make([]string, 0, len(allocs))
	for _, a := range allocs {
		issueIDs = append(issueIDs, fmt.Sprintf("%d", a.issueID))
	}
	res, err := tx.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, from_step_id, to_process_id, to_step_id, to_worker_id, kg, move_kind, issue_ids, created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.ID, b.Code, b.Trace, fromProcess, fromStepID, nil, nil, toWorkerID, kg, kind, strings.Join(issueIDs, ","), nullIf0(createdBy))
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	moveID, _ := res.LastInsertId()
	for _, a := range allocs {
		_, _ = tx.Exec(`INSERT INTO pd_process_move_alloc(move_id, issue_id, kg) VALUES(?,?,?)`, moveID, a.issueID, a.kg)
	}

	fromRemain := boardProcessRemainKgTx(tx, b.ID, fromProcess, b.ProcessID, b.Weight)
	if fromRemain <= kgEps {
		if _, err := tx.Exec(`UPDATE inv_box_code SET weight=0, qty=0, updated_at=NOW() WHERE id=?`, b.ID); err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
		b.Weight = 0
	}

	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}

	var newBoardCode string
	var newBoardID int64
	wh := b.WarehouseID
	if fromStep != nil && fromStep.WarehouseID > 0 {
		wh = fromStep.WarehouseID
	}
	code, nid, serr := s.stockInNewBoardFrom(&boardState{
		ID: b.ID, Code: b.Code, ProductID: b.ProductID, WarehouseID: b.WarehouseID,
		TaskID: b.TaskID, WoID: b.WoID, Trace: b.Trace,
	}, wh, fromProcess, fromStepID, kg)
	if serr != nil {
		msg := serr.Error()
		if msg == "PRODUCT_REQUIRED" || msg == "TRACE_CODE_REQUIRED" || msg == "INVALID_QTY" {
			return nil, msg
		}
		return nil, "STOCK_IN_FAILED:" + msg
	}
	newBoardCode, newBoardID = code, nid

	// Wage settles at day-end, not on stock_in.
	board.Weight = b.Weight
	board.ProcessID = b.ProcessID
	board.StepID = b.StepID
	board.Status = b.Status
	out := gin.H{
		"id": moveID, "action": kind, "kg": kg, "move_kind": kind,
		"from_process_id": fromProcess, "from_step_id": fromStepID,
		"settled_kg": 0.0, "settled_wage_amount": 0.0,
		"new_board_code": newBoardCode, "new_box_code": newBoardCode, "new_board_id": newBoardID,
		"piecework_hint": "入库仅换码；产量工钱请日结入账",
	}
	return out, ""
}

func (s *Services) boardWarehouseKg(boardID int64) float64 {
	if boardID <= 0 {
		return 0
	}
	var v float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM inv_balance WHERE box_code_id=?`, boardID).Scan(&v)
	return roundKg(v)
}

func (s *Services) boardCloseProcessIDs(boardID int64) []int64 {
	seen := map[int64]struct{}{}
	collect := func(q string, args ...interface{}) {
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			return
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil || id <= 0 {
				continue
			}
			seen[id] = struct{}{}
		}
		rows.Close()
	}
	collect(`SELECT DISTINCT process_id FROM pd_process_issue WHERE board_id=?`, boardID)
	collect(`SELECT DISTINCT from_process_id FROM pd_process_move WHERE board_id=?`, boardID)
	var cur int64
	_ = s.DB.QueryRow(`SELECT COALESCE(current_process_id,0) FROM inv_box_code WHERE id=?`, boardID).Scan(&cur)
	if cur > 0 {
		seen[cur] = struct{}{}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

func (s *Services) boardCloseRemain(board *boardState) gin.H {
	out := gin.H{
		"board_id": board.ID, "board_code": board.Code, "trace_code": board.Trace,
		"status": board.Status, "action": "close_preview",
	}
	processes := []gin.H{}
	wipTotal := 0.0
	for _, pid := range s.boardCloseProcessIDs(board.ID) {
		remain := s.boardProcessRemainKg(board.ID, pid)
		if remain <= kgEps {
			continue
		}
		var name string
		_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, pid).Scan(&name)
		processes = append(processes, gin.H{
			"process_id": pid, "process_name": name, "remain_kg": remain,
		})
		wipTotal = roundKg(wipTotal + remain)
	}
	wh := s.boardWarehouseKg(board.ID)
	total := roundKg(wipTotal + wh)
	out["processes"] = processes
	out["process_remain_kg"] = wipTotal
	out["warehouse_kg"] = wh
	out["total_remain_kg"] = total
	out["needs_decision"] = total > kgEps
	return out
}

func (s *Services) closeBoard(board *boardState, confirmLoss bool) (gin.H, string) {
	if board == nil {
		return nil, "BOX_REQUIRED"
	}
	if strings.TrimSpace(board.Trace) == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	remain := s.boardCloseRemain(board)
	total, _ := asFloat(remain["total_remain_kg"])
	if total > kgEps && !confirmLoss {
		return remain, "REMAIN_NEEDS_DECISION"
	}
	procIDs := s.boardCloseProcessIDs(board.ID)
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()
	b, fail := s.reloadBoardTx(tx, board.ID)
	if fail != "" {
		return nil, fail
	}
	if b.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	writeoff := 0.0
	for _, pid := range procIDs {
		kg, fail := writeoffBoardProcessRemainTx(tx, b, pid)
		if fail != "" {
			return nil, fail
		}
		writeoff = roundKg(writeoff + kg)
	}
	if _, err := tx.Exec(`UPDATE inv_balance SET qty=0 WHERE box_code_id=?`, b.ID); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	if _, err := tx.Exec(`UPDATE inv_box_code SET weight=0, qty=0, status='finished', updated_at=NOW() WHERE id=?`, b.ID); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	b.Weight = 0
	b.Status = "finished"
	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	board.Weight = 0
	board.Status = "finished"
	s.snapshotTraceYield(board.Trace)
	out := gin.H{
		"action": "close", "board_id": board.ID, "board_code": board.Code, "trace_code": board.Trace,
		"status": "finished", "writeoff_kg": writeoff, "confirm_loss": confirmLoss,
	}
	return out, ""
}

func (s *Services) enrichScanBoardPreview(preview gin.H, boxID int64, code string, processID, stepID, workerID int64) {
	s.attachBoardPreview(preview, boxID, code, processID, stepID, workerID)
}
