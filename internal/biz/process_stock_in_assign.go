package biz

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// resolveOrCreateBoardForTrace loads board by code or creates one hanging the given trace.
func (s *Services) resolveOrCreateBoardForTrace(code, trace string, qty float64) (*boardState, string) {
	code = strings.TrimSpace(code)
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if code == "" {
		return nil, "BOX_REQUIRED"
	}
	if trace == "" {
		return nil, "TRACE_CODE_REQUIRED"
	}
	board, errMsg := s.loadBoardByCode(code)
	if errMsg == "" && board != nil {
		bt := strings.ToUpper(strings.TrimSpace(board.Trace))
		if bt != "" && bt != trace {
			return nil, "TRACE_MISMATCH"
		}
		if bt == "" {
			_, _ = s.DB.Exec(`UPDATE inv_box_code SET trace_code=?, updated_at=NOW() WHERE id=?`, trace, board.ID)
			board.Trace = trace
		}
		return board, ""
	}
	if errMsg != "" && errMsg != "BOX_NOT_FOUND" && !strings.Contains(errMsg, "NOT_FOUND") {
		return nil, errMsg
	}
	// Create new board for this trace
	whID := int64(1)
	productID := int64(1)
	_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0), COALESCE(warehouse_id,0) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?) AND COALESCE(product_id,0)>0
		ORDER BY id DESC LIMIT 1`, trace).Scan(&productID, &whID)
	if productID <= 0 {
		productID = 1
	}
	if whID <= 0 {
		whID = 1
	}
	if qty < 0 {
		qty = 0
	}
	res, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, trace_code, status)
		VALUES(?,?,?,?,?,?,?,'open')`,
		code, productID, whID, time.Now().Format("20060102"), 0, 0, trace)
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	id, _ := res.LastInsertId()
	if id <= 0 {
		_ = s.DB.QueryRow(`SELECT id FROM inv_box_code WHERE code=?`, code).Scan(&id)
	}
	board, errMsg = s.loadBoardByCode(code)
	if errMsg != "" {
		return nil, errMsg
	}
	return board, ""
}

// stockInTraceProcessKg completes open issues for (trace, process) including board_id=0, then stocks into board.
func (s *Services) stockInTraceProcessKg(board *boardState, toWorkerID int64, kg float64, kind string, createdBy, fromProcessID int64) (gin.H, string) {
	if board == nil || kg <= kgEps {
		return nil, "INVALID_QTY"
	}
	trace := strings.ToUpper(strings.TrimSpace(board.Trace))
	if trace == "" {
		return nil, "TRACE_CODE_REQUIRED"
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
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, "DB_ERROR"
	}
	defer func() { _ = tx.Rollback() }()
	b, fail := s.reloadBoardTx(tx, board.ID)
	if fail != "" {
		return nil, fail
	}

	var occ, pool float64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0`,
		trace, fromProcess).Scan(&occ)
	_ = tx.QueryRow(`SELECT COALESCE(SUM(issue_kg - returned_kg - completed_kg),0)
		FROM pd_process_issue WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open' AND COALESCE(worker_id,0)=0`,
		trace, fromProcess).Scan(&pool)
	wip := roundKg(occ + pool)
	if b.ProcessID == fromProcess {
		wip = roundKg(wip + b.Weight)
	}
	if kg-wip > kgEps {
		return nil, "QTY_EXCEEDS_WIP"
	}

	type alloc struct {
		issueID, workerID int64
		kg                float64
	}
	allocs := []alloc{}
	need := kg
	rows, err := tx.Query(`SELECT id, worker_id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0
		ORDER BY CASE WHEN board_id=? THEN 0 WHEN COALESCE(board_id,0)=0 THEN 1 ELSE 2 END, created_at, id`,
		trace, fromProcess, b.ID)
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
		if _, err := tx.Exec(`UPDATE pd_process_issue SET completed_kg=?, status=?, board_id=COALESCE(NULLIF(board_id,0),?), board_code=CASE WHEN COALESCE(board_code,'')='' THEN ? ELSE board_code END, updated_at=NOW() WHERE id=?`,
			newDone, st, b.ID, b.Code, r.id); err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
		allocs = append(allocs, alloc{issueID: r.id, workerID: r.workerID, kg: take})
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
		if fail := takeTracePoolTx(tx, trace, fromProcess, need); fail != "" {
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
		b.ID, b.Code, trace, fromProcess, b.StepID, nil, nil, toWorkerID, kg, kind, strings.Join(issueIDs, ","), nullIf0(createdBy))
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	moveID, _ := res.LastInsertId()
	for _, a := range allocs {
		_, _ = tx.Exec(`INSERT INTO pd_process_move_alloc(move_id, issue_id, kg) VALUES(?,?,?)`, moveID, a.issueID, a.kg)
	}
	if err := tx.Commit(); err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}

	wh := b.WarehouseID
	if wh <= 0 {
		wh = 1
	}
	newCode, newID, serr := s.stockInNewBoardFrom(&boardState{
		ID: b.ID, Code: b.Code, Trace: trace, ProductID: b.ProductID, WarehouseID: wh,
		ProcessID: fromProcess, StepID: b.StepID, TaskID: b.TaskID, WoID: b.WoID,
	}, wh, fromProcess, b.StepID, kg)
	if serr != nil {
		return nil, "STOCK_IN_FAILED:" + serr.Error()
	}
	out := gin.H{
		"action": "stock_in", "board_id": b.ID, "board_code": b.Code, "trace_code": trace,
		"from_process_id": fromProcess, "kg": kg, "move_id": moveID,
		"new_board_code": newCode, "new_box_code": newCode, "new_board_id": newID,
	}
	return out, ""
}
