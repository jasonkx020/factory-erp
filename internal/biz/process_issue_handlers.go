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
	s.ensureBoardProcess(board)
	if strings.TrimSpace(board.Trace) == "" {
		return body, nil, 0, "", "", 0, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return body, nil, 0, "", "", 0, "BOARD_FINISHED"
	}
	return body, board, workerID, workerName, badge, kg, ""
}

func (s *Services) handleBoardIssueHTTP(c *gin.Context) bool {
	body, board, workerID, workerName, badge, kg, errMsg := s.parseBoardAction(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	processID, _ := asInt64(body["process_id"])
	stepID, _ := asInt64(body["step_id"])
	if processID <= 0 {
		processID = board.ProcessID
	}
	if stepID <= 0 {
		stepID = board.StepID
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
	api.OK(c, out)
	return true
}

func (s *Services) handleBoardReturnHTTP(c *gin.Context) bool {
	_, board, workerID, workerName, badge, kg, errMsg := s.parseBoardAction(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
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
	s.attachBoardPreview(out, board.ID, board.Code, board.ProcessID, board.StepID, workerID)
	api.OK(c, out)
	return true
}

func (s *Services) handleBoardMoveHTTP(c *gin.Context) bool {
	body, board, workerID, workerName, badge, kg, errMsg := s.parseBoardAction(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	kind := strings.TrimSpace(strOrDef(body["move_kind"], "next"))
	if kind == "finish" {
		kind = "finish_in"
	}
	if kind != "next" && kind != "finish_in" {
		api.FailJSON(c, "INVALID_MOVE_KIND")
		return true
	}
	fromProcess := board.ProcessID
	if !s.workerShiftAuthorized(workerID, fromProcess) {
		api.FailJSON(c, "SHIFT_NOT_AUTHORIZED")
		return true
	}
	createdBy := claimsUserID(c)
	out, fail := s.moveBoardKg(board, workerID, kg, kind, createdBy)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	out["worker_id"] = workerID
	out["worker_name"] = workerName
	out["badge_code"] = badge
	previewProc := board.ProcessID
	previewStep := board.StepID
	if v, ok := out["to_process_id"].(int64); ok && v > 0 {
		previewProc = v
	}
	if v, ok := out["to_step_id"].(int64); ok && v > 0 {
		previewStep = v
	}
	s.attachBoardPreview(out, board.ID, board.Code, previewProc, previewStep, workerID)
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

func (s *Services) ensureBoardProcess(b *boardState) {
	if b == nil {
		return
	}
	if b.StepID > 0 && s.loadStep(b.StepID) == nil {
		b.StepID = 0
	}
	if b.ProcessID > 0 && b.StepID > 0 {
		return
	}
	rid := s.resolveRoutingID(b.TaskID, b.ProductID)
	var step *routingStep
	if b.ProcessID > 0 {
		step = s.stepByProcess(rid, b.ProcessID)
	}
	if step == nil {
		step = s.firstStep(rid)
	}
	if step == nil {
		return
	}
	b.ProcessID = step.ProcessID
	b.StepID = step.ID
	s.advanceBoxToStep(b.ID, step)
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
	var curProc, curStep int64
	_ = s.DB.QueryRow(`SELECT COALESCE(weight, qty, 0), COALESCE(trace_code,''), COALESCE(current_process_id,0), COALESCE(current_step_id,0)
		FROM inv_box_code WHERE id=?`, boardID).Scan(&weight, &trace, &curProc, &curStep)
	if processID <= 0 {
		processID = curProc
	}
	if stepID <= 0 {
		stepID = curStep
	}
	avail := s.boardAvailableKg(boardID, processID, curProc, weight)
	myOpen := s.workerOpenKg(boardID, processID, workerID)
	procOpen := s.processOpenKg(boardID, processID)
	dst["board_id"] = boardID
	dst["board_code"] = code
	dst["box_code"] = code
	dst["trace_code"] = trace
	dst["process_id"] = processID
	dst["step_id"] = stepID
	dst["available_kg"] = avail
	dst["my_open_kg"] = myOpen
	dst["process_open_kg"] = procOpen
	dst["wip_kg"] = roundKg(avail + procOpen)
	if step := s.loadStep(stepID); step != nil {
		dst["step_name"] = step.StepName
		dst["is_piecework"] = step.IsPiecework
		next := s.nextStep(step)
		if next != nil {
			dst["has_next"] = true
			dst["next_process_id"] = next.ProcessID
			dst["next_step_id"] = next.ID
			dst["next_step_name"] = next.StepName
		} else {
			dst["has_next"] = false
		}
	}
	var processName string
	_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, processID).Scan(&processName)
	dst["process_name"] = processName
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
	avail := b.Weight
	if b.ProcessID != processID {
		avail = 0
	}
	var pool float64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`, b.ID, processID).Scan(&pool)
	avail = roundKg(avail + pool)
	if kg-avail > kgEps {
		return nil, "QTY_EXCEEDS_AVAILABLE"
	}
	need := kg
	if b.ProcessID == processID && b.Weight > kgEps && need > kgEps {
		take := math.Min(b.Weight, need)
		b.Weight = roundKg(b.Weight - take)
		need = roundKg(need - take)
		if _, err := tx.Exec(`UPDATE inv_box_code SET weight=?, qty=?, updated_at=NOW() WHERE id=?`, b.Weight, b.Weight, b.ID); err != nil {
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
	if s.isPieceworkProcess(processID, stepID) {
		rate := s.processWageRate(processID)
		out["piecework"] = true
		out["rate"] = rate
		out["issue_locked_kg"] = kg
		out["issue_locked_wage_amount"] = roundMoney(kg * rate)
		out["piecework_status"] = "locked"
		out["piecework_hint"] = "计件已预锁定，进下道或完工入库后结算"
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
		if procID > 0 && s.isPieceworkProcess(procID, stepByProcess[procID]) {
			rate := s.processWageRate(procID)
			out["piecework"] = true
			out["rate"] = rate
			out["released_locked_kg"] = returnedTotal
			out["released_locked_wage_amount"] = roundMoney(returnedTotal * rate)
			out["piecework_status"] = "locked"
			out["piecework_hint"] = "退库扣减计件锁定，未结算部分不入日汇总"
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
	rows, err := tx.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' ORDER BY created_at, id`, b.ID, processID)
	if err != nil {
		return 0, "DB_ERROR:" + err.Error()
	}
	type iss struct {
		id                         int64
		issue, returned, completed float64
	}
	list := []iss{}
	for rows.Next() {
		var r iss
		if err := rows.Scan(&r.id, &r.issue, &r.returned, &r.completed); err != nil {
			continue
		}
		list = append(list, r)
	}
	rows.Close()
	writeoff := 0.0
	for _, r := range list {
		rem := issueRemain(r.issue, r.returned, r.completed)
		if rem <= kgEps {
			_, _ = tx.Exec(`UPDATE pd_process_issue SET status='closed', updated_at=NOW() WHERE id=?`, r.id)
			continue
		}
		newDone := roundKg(r.issue - r.returned)
		if _, err := tx.Exec(`UPDATE pd_process_issue SET completed_kg=?, status='closed', updated_at=NOW() WHERE id=?`, newDone, r.id); err != nil {
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

func (s *Services) moveBoardKg(board *boardState, toWorkerID int64, kg float64, kind string, createdBy int64) (gin.H, string) {
	if board == nil || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	if strings.TrimSpace(board.Trace) == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	if board.Status == "finished" {
		return nil, "BOARD_FINISHED"
	}
	fromStep := s.loadStep(board.StepID)
	var toStep *routingStep
	if kind == "next" {
		if fromStep == nil {
			return nil, "ROUTING_REQUIRED"
		}
		toStep = s.nextStep(fromStep)
		if toStep == nil {
			return nil, "NO_NEXT_STEP"
		}
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
	fromProcess := b.ProcessID
	fromStepID := b.StepID
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

	var toProcessID, toStepID int64
	if toStep != nil {
		toProcessID = toStep.ProcessID
		toStepID = toStep.ID
	}
	issueIDs := make([]string, 0, len(allocs))
	for _, a := range allocs {
		issueIDs = append(issueIDs, fmt.Sprintf("%d", a.issueID))
	}
	var toProcArg, toStepArg interface{}
	if toProcessID > 0 {
		toProcArg, toStepArg = toProcessID, toStepID
	}
	res, err := tx.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, from_step_id, to_process_id, to_step_id, to_worker_id, kg, move_kind, issue_ids, created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.ID, b.Code, b.Trace, fromProcess, fromStepID, toProcArg, toStepArg, toWorkerID, kg, kind, strings.Join(issueIDs, ","), nullIf0(createdBy))
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	moveID, _ := res.LastInsertId()
	for _, a := range allocs {
		_, _ = tx.Exec(`INSERT INTO pd_process_move_alloc(move_id, issue_id, kg) VALUES(?,?,?)`, moveID, a.issueID, a.kg)
	}
	if kind == "next" && toProcessID > 0 && toWorkerID > 0 {
		_, err = tx.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status)
			VALUES(?,?,?,?,?,?,?,0,0,'open')`, b.ID, b.Code, b.Trace, toProcessID, toStepID, toWorkerID, kg)
		if err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
	}

	fromRemain := boardProcessRemainKgTx(tx, b.ID, fromProcess, b.ProcessID, b.Weight)
	if fromRemain <= kgEps {
		if kind == "finish_in" {
			if _, err := tx.Exec(`UPDATE inv_box_code SET weight=0, qty=0, updated_at=NOW() WHERE id=?`, b.ID); err != nil {
				return nil, "DB_ERROR:" + err.Error()
			}
			b.Weight = 0
		} else if toStep != nil {
			var poolTo float64
			_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
				FROM pd_process_issue WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`, b.ID, toProcessID).Scan(&poolTo)
			if poolTo > kgEps {
				if fail := takePoolTx(tx, b.ID, toProcessID, poolTo); fail != "" {
					return nil, fail
				}
				b.Weight = roundKg(poolTo)
			} else {
				b.Weight = 0
			}
			if _, err := tx.Exec(`UPDATE inv_box_code SET current_process_id=?, current_step_id=?, weight=?, qty=?, updated_at=NOW() WHERE id=?`,
				toProcessID, toStepID, b.Weight, b.Weight, b.ID); err != nil {
				return nil, "DB_ERROR:" + err.Error()
			}
			b.ProcessID = toProcessID
			b.StepID = toStepID
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}

	doPiece := fromProcess > 0 && (fromStep == nil || fromStep.IsPiecework)
	if fromStep == nil && fromProcess > 0 {
		var pPiece int
		_ = s.DB.QueryRow(`SELECT COALESCE(is_piecework,0) FROM pd_process WHERE id=?`, fromProcess).Scan(&pPiece)
		doPiece = pPiece == 1
	}
	pieceKg := 0.0
	settledAmt := 0.0
	if doPiece {
		src := fmt.Sprintf("M%d", moveID)
		rate := s.processWageRate(fromProcess)
		for _, a := range allocs {
			if !a.piece || a.workerID <= 0 || a.kg <= kgEps {
				continue
			}
			s.upsertPieceworkSummaryKeyed(a.workerID, fromProcess, src, a.kg, a.kg, a.kg, 0, 1)
			pieceKg = roundKg(pieceKg + a.kg)
			settledAmt = roundMoney(settledAmt + a.kg*rate)
		}
	}

	board.Weight = b.Weight
	board.ProcessID = b.ProcessID
	board.StepID = b.StepID
	board.Status = b.Status
	out := gin.H{
		"id": moveID, "action": kind, "kg": kg, "move_kind": kind,
		"from_process_id": fromProcess, "from_step_id": fromStepID,
		"settled_kg": pieceKg, "settled_wage_amount": settledAmt,
	}
	if doPiece && pieceKg > kgEps {
		out["piecework"] = true
		out["piecework_status"] = "settled"
		out["piecework_hint"] = "本道工序计件已结算入日汇总"
	}
	if kind == "next" && toProcessID > 0 && toWorkerID > 0 && s.isPieceworkProcess(toProcessID, toStepID) {
		rateTo := s.processWageRate(toProcessID)
		out["to_locked_kg"] = kg
		out["to_locked_wage_amount"] = roundMoney(kg * rateTo)
		out["to_piecework_status"] = "locked"
	}
	if toProcessID > 0 {
		out["to_process_id"] = toProcessID
		out["to_step_id"] = toStepID
		if toStep != nil {
			out["to_step_name"] = toStep.StepName
		}
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
	s.snapshotBoardYield(board.ID)
	out := gin.H{
		"action": "close", "board_id": board.ID, "board_code": board.Code, "trace_code": board.Trace,
		"status": "finished", "writeoff_kg": writeoff, "confirm_loss": confirmLoss,
	}
	return out, ""
}

func (s *Services) enrichScanBoardPreview(preview gin.H, boxID int64, code string, processID, stepID, workerID int64) {
	s.attachBoardPreview(preview, boxID, code, processID, stepID, workerID)
}
