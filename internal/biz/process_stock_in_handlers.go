package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleProcessStockIns(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/approve") && method == "POST" {
		return s.approveProcessStockIn(c)
	}
	if strings.Contains(openapiPath, "/reject") && method == "POST" {
		return s.rejectProcessStockIn(c)
	}
	if method == "POST" && (action == "create" || strings.HasSuffix(openapiPath, "/process-stock-ins")) {
		return s.createProcessStockIn(c)
	}
	if method == "GET" && paramID(c) > 0 {
		return s.getProcessStockIn(c)
	}
	if method == "GET" {
		return s.listProcessStockIns(c)
	}
	api.FailJSON(c, "METHOD_NOT_ALLOWED")
	return true
}

func (s *Services) createProcessStockIn(c *gin.Context) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	body := bindBody(c)
	trace := strings.ToUpper(strings.TrimSpace(strOrDef(body["trace_code"], strOr(body["code"]))))
	processID := asInt64Or0(body["process_id"])
	if fail := s.requireTraceProductionOpen(trace); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	if processID <= 0 {
		api.FailJSON(c, "PROCESS_REQUIRED")
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	applyKg, _ := asFloat(body["apply_kg"])
	if applyKg <= kgEps {
		applyKg, _ = asFloat(body["kg"])
	}
	reweigh, _ := asFloat(body["reweigh_kg"])
	if applyKg <= kgEps {
		applyKg = reweigh
	}
	applyKg = roundKg(applyKg)
	if applyKg <= kgEps {
		api.FailJSON(c, "INVALID_QTY")
		return true
	}
	// 申请不填板码：由仓管接收时分配
	boardID := int64(0)
	boardCode := ""
	applicant, _, _, _ := s.currentEmployeeInfo(c)
	workerID := asInt64Or0(body["worker_id"])
	if workerID <= 0 {
		workerID = applicant
	}
	photo := strings.TrimSpace(strOrDef(body["photo_url"], strOr(body["image_url"])))
	docNo := fmt.Sprintf("SI%s", time.Now().Format("060102150405"))
	issueIDs := strings.TrimSpace(strOr(body["issue_ids"]))
	res, err := s.DB.Exec(`INSERT INTO pd_process_stock_in(doc_no, trace_code, process_id, board_id, board_code,
		applicant_employee_id, worker_id, apply_kg, reweigh_kg, photo_url, status, issue_ids, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?,'pending_warehouse',?,?)`,
		docNo, trace, processID, boardID, boardCode, applicant, workerID, applyKg, reweigh, photo, issueIDs, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "stock_in_apply", BoardID: boardID, BoardCode: boardCode, TraceCode: trace,
		ProcessID: processID, WorkerID: workerID, ActorUserID: claimsUserID(c), OperatorEmployeeID: applicant,
		Kg: applyKg, RefType: "pd_process_stock_in", RefID: id,
	})
	api.OK(c, s.loadProcessStockIn(id))
	return true
}

func (s *Services) listProcessStockIns(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		where += ` AND status=?`
		args = append(args, st)
	}
	if tr := strings.ToUpper(strings.TrimSpace(c.Query("trace_code"))); tr != "" {
		where += ` AND UPPER(trace_code)=?`
		args = append(args, tr)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_stock_in `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, doc_no, trace_code, process_id, board_id, board_code,
		applicant_employee_id, worker_id, apply_kg, reweigh_kg, photo_url, status, issue_ids,
		warehouse_id, approved_by, COALESCE(CAST(approved_at AS TEXT),''), COALESCE(remark,''),
		COALESCE(CAST(created_at AS TEXT),'')
		FROM pd_process_stock_in `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, pid, bid, appID, wid, whID, apprBy int64
		var doc, trace, bcode, photo, st, issueIDs, apprAt, remark, created string
		var applyKg, reweigh float64
		if err := rows.Scan(&id, &doc, &trace, &pid, &bid, &bcode, &appID, &wid, &applyKg, &reweigh, &photo, &st, &issueIDs,
			&whID, &apprBy, &apprAt, &remark, &created); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "doc_no": doc, "trace_code": trace, "process_id": pid, "process_name": s.processName(pid),
			"board_id": bid, "board_code": bcode, "applicant_employee_id": appID, "worker_id": wid,
			"apply_kg": applyKg, "reweigh_kg": reweigh, "photo_url": photo, "status": st, "issue_ids": issueIDs,
			"warehouse_id": whID, "approved_by": apprBy, "approved_at": apprAt, "remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) getProcessStockIn(c *gin.Context) bool {
	row := s.loadProcessStockIn(paramID(c))
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, row)
	return true
}

func (s *Services) loadProcessStockIn(id int64) gin.H {
	if id <= 0 {
		return nil
	}
	var pid, bid, appID, wid, whID, apprBy int64
	var doc, trace, bcode, photo, st, issueIDs, apprAt, remark, created string
	var applyKg, reweigh float64
	err := s.DB.QueryRow(`SELECT id, doc_no, trace_code, process_id, board_id, board_code,
		applicant_employee_id, worker_id, apply_kg, reweigh_kg, photo_url, status, issue_ids,
		warehouse_id, approved_by, COALESCE(CAST(approved_at AS TEXT),''), COALESCE(remark,''),
		COALESCE(CAST(created_at AS TEXT),'')
		FROM pd_process_stock_in WHERE id=?`, id).Scan(&id, &doc, &trace, &pid, &bid, &bcode, &appID, &wid, &applyKg, &reweigh, &photo, &st, &issueIDs,
		&whID, &apprBy, &apprAt, &remark, &created)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "doc_no": doc, "trace_code": trace, "process_id": pid, "process_name": s.processName(pid),
		"board_id": bid, "board_code": bcode, "applicant_employee_id": appID, "worker_id": wid,
		"apply_kg": applyKg, "reweigh_kg": reweigh, "photo_url": photo, "status": st, "issue_ids": issueIDs,
		"warehouse_id": whID, "approved_by": apprBy, "approved_at": apprAt, "remark": remark, "created_at": created,
	}
}

func (s *Services) approveProcessStockIn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	row := s.loadProcessStockIn(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["status"]) != "pending_warehouse" {
		api.FailJSON(c, "NOT_PENDING")
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	kg, _ := asFloat(body["reweigh_kg"])
	if kg <= kgEps {
		kg = asFloatOr0(row["apply_kg"])
	}
	kg = roundKg(kg)
	boardCode := strings.TrimSpace(strOrDef(body["board_code"], strOr(body["box_code"])))
	trace := strings.ToUpper(strings.TrimSpace(strOr(row["trace_code"])))
	processID := asInt64Or0(row["process_id"])
	if boardCode == "" {
		api.FailJSON(c, "BOX_REQUIRED")
		return true
	}
	board, errMsg := s.resolveOrCreateBoardForTrace(boardCode, trace, kg)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	_, _ = s.DB.Exec(`UPDATE pd_process_stock_in SET board_id=?, board_code=?, updated_at=NOW() WHERE id=?`,
		board.ID, board.Code, id)
	workerID := asInt64Or0(row["worker_id"])

	out, fail := s.stockInTraceProcessKg(board, workerID, kg, "stock_in", claimsUserID(c), processID)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	empID, _, _, _ := s.currentEmployeeInfo(c)
	_, _ = s.DB.Exec(`UPDATE pd_process_stock_in SET status='posted', reweigh_kg=?, photo_url=COALESCE(NULLIF(?,''), photo_url),
		approved_by=?, approved_at=NOW(), updated_at=NOW() WHERE id=?`,
		kg, strOrDef(body["photo_url"], strOr(body["image_url"])), empID, id)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "stock_in_approve", BoardID: board.ID, BoardCode: board.Code, TraceCode: trace,
		ProcessID: processID, WorkerID: workerID, ActorUserID: claimsUserID(c), OperatorEmployeeID: empID,
		Kg: kg, RefType: "pd_process_stock_in", RefID: id, After: out,
	})
	api.OK(c, gin.H{"stock_in": s.loadProcessStockIn(id), "move": out})
	return true
}

func (s *Services) rejectProcessStockIn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin") {
		return true
	}
	id := paramID(c)
	_, err := s.DB.Exec(`UPDATE pd_process_stock_in SET status='rejected', remark=COALESCE(NULLIF(?,''), remark), updated_at=NOW() WHERE id=? AND status='pending_warehouse'`,
		strOr(bindBody(c)["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadProcessStockIn(id))
	return true
}

func (s *Services) completeIssueKgForStockIn(boardID, processID, workerID int64, kg float64) {
	need := kg
	q := `SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0`
	args := []interface{}{boardID, processID}
	if workerID > 0 {
		q += ` AND worker_id=?`
		args = append(args, workerID)
	}
	q += ` ORDER BY created_at, id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() && need > kgEps {
		var id int64
		var issue, ret, done float64
		if err := rows.Scan(&id, &issue, &ret, &done); err != nil {
			continue
		}
		rem := issueRemain(issue, ret, done)
		if rem <= kgEps {
			continue
		}
		take := rem
		if take > need {
			take = need
		}
		newDone := roundKg(done + take)
		st := "open"
		if issueRemain(issue, ret, newDone) <= kgEps {
			st = "closed"
		}
		_, _ = s.DB.Exec(`UPDATE pd_process_issue SET completed_kg=?, status=?, updated_at=NOW() WHERE id=?`, newDone, st, id)
		need = roundKg(need - take)
	}
}
