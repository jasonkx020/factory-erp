package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleProcessIssuesAPI(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/return-apply") && method == "POST" {
		return s.handleIssueReturnApply(c)
	}
	if strings.Contains(openapiPath, "/return-approve") && method == "POST" {
		return s.handleIssueReturnApprove(c)
	}
	if strings.Contains(openapiPath, "/return-reject") && method == "POST" {
		return s.handleIssueReturnReject(c)
	}
	if strings.Contains(openapiPath, "/issue-approve") && method == "POST" {
		return s.handleIssueWarehouseApprove(c)
	}
	if strings.Contains(openapiPath, "/issue-reject") && method == "POST" {
		return s.handleIssueWarehouseReject(c)
	}
	if strings.Contains(openapiPath, "/confirm-done") && method == "POST" {
		return s.handleIssueConfirmDone(c)
	}
	if method == "GET" && (action == "get" || paramID(c) > 0) && !strings.Contains(openapiPath, "process-issues?") {
		if id := paramID(c); id > 0 {
			return s.getProcessIssue(c, id)
		}
	}
	if method == "GET" {
		return s.listProcessIssues(c)
	}
	api.FailJSON(c, "METHOD_NOT_ALLOWED")
	return true
}

func (s *Services) listProcessIssues(c *gin.Context) bool {
	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	if scope == "" {
		scope = "related" // App 默认：本人领 + 我代领
	}
	empID, _, _, _ := s.currentEmployeeInfo(c)
	if empID <= 0 && (scope == "mine" || scope == "for_me" || scope == "proxy_by_me" || scope == "related" || scope == "all_mine") {
		api.FailJSON(c, "EMPLOYEE_REQUIRED")
		return true
	}
	// 补全历史代领单操作人（从流水回填），避免「代领成功但历史看不见」
	if empID > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_process_issue i SET issued_by_employee_id=l.operator_employee_id
			FROM (
				SELECT DISTINCT ON (ref_id) ref_id, operator_employee_id
				FROM pd_station_flow_log
				WHERE ref_type='pd_process_issue' AND event_type='issue' AND COALESCE(operator_employee_id,0)>0
				ORDER BY ref_id, id DESC
			) l
			WHERE i.id=l.ref_id AND COALESCE(i.issued_by_employee_id,0)=0`)
	}
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	switch scope {
	case "warehouse_queue", "warehouse_pending":
		if !s.claimsHasAnyRole(c, "warehouse", "foreman", "admin", "sys_admin") {
			api.FailJSON(c, "ROLE_FORBIDDEN")
			return true
		}
	case "mine", "for_me":
		where += ` AND i.worker_id=?`
		args = append(args, empID)
	case "proxy_by_me":
		where += ` AND COALESCE(i.issued_by_employee_id,0)=? AND i.worker_id<>i.issued_by_employee_id AND COALESCE(i.issued_by_employee_id,0)>0`
		args = append(args, empID)
	case "related", "all_mine":
		where += ` AND (i.worker_id=? OR COALESCE(i.issued_by_employee_id,0)=?)`
		args = append(args, empID, empID)
	}
	if st := strings.TrimSpace(c.Query("biz_status")); st != "" {
		where += ` AND i.biz_status=?`
		args = append(args, st)
	}
	// 工牌号：匹配领料人（工人）工牌；大小写不敏感，支持模糊
	if badge := strings.TrimSpace(c.Query("badge_code")); badge != "" {
		where += ` AND EXISTS (
			SELECT 1 FROM hr_employee e
			WHERE e.id=i.worker_id AND lower(COALESCE(e.badge_code,'')) LIKE lower(?)
		)`
		args = append(args, "%"+badge+"%")
	}
	if from := strings.TrimSpace(c.Query("date_from")); from != "" {
		where += ` AND i.created_at>=?::timestamptz`
		args = append(args, from)
	}
	if to := strings.TrimSpace(c.Query("date_to")); to != "" {
		// 含当天整天
		if len(to) <= 10 {
			to = to + " 23:59:59"
		}
		where += ` AND i.created_at<=?::timestamptz`
		args = append(args, to)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue i `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT i.id, i.board_id, COALESCE(i.board_code,''), COALESCE(i.trace_code,''),
		i.process_id, COALESCE(p.name,''), i.worker_id, COALESCE(ew.name,''), COALESCE(ew.badge_code,''),
		COALESCE(i.issued_by_employee_id,0), COALESCE(ei.name,''),
		i.issue_kg, i.returned_kg, i.completed_kg, COALESCE(i.pending_return_kg,0),
		COALESCE(i.pending_reweigh_kg,0), COALESCE(i.pending_photo_url,''),
		COALESCE(i.status,'open'), COALESCE(i.biz_status,'open'), COALESCE(i.source,''),
		COALESCE(i.work_done_by,0), COALESCE(CAST(i.work_done_at AS TEXT),''),
		COALESCE(CAST(i.created_at AS TEXT),'')
		FROM pd_process_issue i
		LEFT JOIN pd_process p ON p.id=i.process_id
		LEFT JOIN hr_employee ew ON ew.id=i.worker_id
		LEFT JOIN hr_employee ei ON ei.id=i.issued_by_employee_id
		`+where+` ORDER BY i.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, boardID, processID, workerID, issuerID, doneBy int64
		var boardCode, trace, processName, workerName, badgeCode, issuerName, status, bizStatus, source, doneAt, created string
		var pendPhoto string
		var issueKg, retKg, doneKg, pendKg, pendReweigh float64
		if err := rows.Scan(&id, &boardID, &boardCode, &trace, &processID, &processName, &workerID, &workerName, &badgeCode,
			&issuerID, &issuerName, &issueKg, &retKg, &doneKg, &pendKg, &pendReweigh, &pendPhoto, &status, &bizStatus, &source, &doneBy, &doneAt, &created); err != nil {
			continue
		}
		proxy := issuerID > 0 && workerID > 0 && issuerID != workerID
		_, farmerName := s.traceFarmerInfo(trace)
		item := gin.H{
			"id": id, "board_id": boardID, "board_code": boardCode, "trace_code": trace,
			"process_id": processID, "process_name": processName,
			"worker_id": workerID, "worker_name": workerName, "badge_code": badgeCode,
			"issued_by_employee_id": issuerID, "issuer_name": issuerName,
			"is_proxy": proxy, "issue_kg": issueKg, "returned_kg": retKg, "completed_kg": doneKg,
			"pending_return_kg": pendKg, "pending_reweigh_kg": pendReweigh, "pending_photo_url": pendPhoto,
			"returnable_kg": s.issueReturnableKg(issueKg, retKg, doneKg, pendKg),
			"status": status, "biz_status": bizStatus, "source": source, "ended": bizStatus == "work_done",
			"work_done_by": doneBy, "work_done_at": doneAt, "created_at": created,
		}
		if farmerName != "" {
			item["farmer_name"] = farmerName
		}
		list = append(list, item)
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) getProcessIssue(c *gin.Context, id int64) bool {
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, row)
	return true
}

func (s *Services) loadProcessIssueRow(id int64) gin.H {
	var boardID, processID, workerID, issuerID, doneBy, pendBy int64
	var boardCode, trace, processName, workerName, badgeCode, issuerName, status, bizStatus, doneAt, created, pendPhoto, pendRemark string
	var issueKg, retKg, doneKg, pendKg, pendReweigh float64
	err := s.DB.QueryRow(`SELECT i.id, i.board_id, COALESCE(i.board_code,''), COALESCE(i.trace_code,''),
		i.process_id, COALESCE(p.name,''), i.worker_id, COALESCE(ew.name,''), COALESCE(ew.badge_code,''),
		COALESCE(i.issued_by_employee_id,0), COALESCE(ei.name,''),
		i.issue_kg, i.returned_kg, i.completed_kg, COALESCE(i.pending_return_kg,0), COALESCE(i.pending_reweigh_kg,0),
		COALESCE(i.pending_photo_url,''), COALESCE(i.pending_return_by,0), COALESCE(i.pending_remark,''),
		COALESCE(i.status,'open'), COALESCE(i.biz_status,'open'),
		COALESCE(i.work_done_by,0), COALESCE(CAST(i.work_done_at AS TEXT),''),
		COALESCE(CAST(i.created_at AS TEXT),'')
		FROM pd_process_issue i
		LEFT JOIN pd_process p ON p.id=i.process_id
		LEFT JOIN hr_employee ew ON ew.id=i.worker_id
		LEFT JOIN hr_employee ei ON ei.id=i.issued_by_employee_id
		WHERE i.id=?`, id).Scan(&id, &boardID, &boardCode, &trace, &processID, &processName, &workerID, &workerName, &badgeCode,
		&issuerID, &issuerName, &issueKg, &retKg, &doneKg, &pendKg, &pendReweigh, &pendPhoto, &pendBy, &pendRemark,
		&status, &bizStatus, &doneBy, &doneAt, &created)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "board_id": boardID, "board_code": boardCode, "trace_code": trace,
		"process_id": processID, "process_name": processName,
		"worker_id": workerID, "worker_name": workerName, "badge_code": badgeCode,
		"issued_by_employee_id": issuerID, "issuer_name": issuerName,
		"is_proxy": issuerID > 0 && workerID > 0 && issuerID != workerID,
		"issue_kg": issueKg, "returned_kg": retKg, "completed_kg": doneKg,
		"pending_return_kg": pendKg, "pending_reweigh_kg": pendReweigh,
		"pending_photo_url": pendPhoto, "pending_return_by": pendBy, "pending_remark": pendRemark,
		"returnable_kg": s.issueReturnableKg(issueKg, retKg, doneKg, pendKg),
		"status": status, "biz_status": bizStatus, "ended": bizStatus == "work_done",
		"work_done_by": doneBy, "work_done_at": doneAt, "created_at": created,
	}
}

func (s *Services) canApplyIssueReturn(c *gin.Context, workerID, issuerID int64) bool {
	empID, _, _, _ := s.currentEmployeeInfo(c)
	if empID <= 0 {
		return false
	}
	if empID == workerID || (issuerID > 0 && empID == issuerID) {
		return true
	}
	return s.claimsHasAnyRole(c, "foreman")
}

func (s *Services) handleIssueReturnApply(c *gin.Context) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["biz_status"]) == "work_done" {
		api.FailJSON(c, "ISSUE_ALREADY_ENDED")
		return true
	}
	if strOr(row["biz_status"]) == "return_pending" {
		api.FailJSON(c, "RETURN_ALREADY_PENDING")
		return true
	}
	if strOr(row["biz_status"]) == "issue_pending_warehouse" {
		api.FailJSON(c, "NOT_ISSUE_PENDING")
		return true
	}
	workerID := asInt64Or0(row["worker_id"])
	issuerID := asInt64Or0(row["issued_by_employee_id"])
	if !s.canApplyIssueReturn(c, workerID, issuerID) {
		api.FailJSON(c, "ROLE_FORBIDDEN")
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	retKg, _ := asFloat(body["return_kg"])
	if retKg <= kgEps {
		retKg, _ = asFloat(body["kg"])
	}
	reweigh, _ := asFloat(body["reweigh_kg"])
	if retKg <= kgEps {
		retKg = reweigh
	}
	retKg = roundKg(retKg)
	maxRet := asFloatOr0(row["returnable_kg"])
	if retKg <= kgEps || retKg-maxRet > kgEps {
		api.FailJSON(c, "QTY_EXCEEDS_RETURNABLE")
		return true
	}
	photo := strings.TrimSpace(strOrDef(body["photo_url"], strOr(body["image_url"])))
	empID, _, _, _ := s.currentEmployeeInfo(c)
	_, err := s.DB.Exec(`UPDATE pd_process_issue SET biz_status='return_pending', pending_return_kg=?, pending_reweigh_kg=?,
		pending_photo_url=?, pending_return_by=?, pending_remark=?, updated_at=NOW() WHERE id=?`,
		retKg, reweigh, photo, empID, strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "return_apply", BoardID: asInt64Or0(row["board_id"]), BoardCode: strOr(row["board_code"]),
		TraceCode: strOr(row["trace_code"]), ProcessID: asInt64Or0(row["process_id"]),
		WorkerID: workerID, ActorUserID: claimsUserID(c), OperatorEmployeeID: empID,
		Kg: retKg, RefType: "pd_process_issue", RefID: id, Remark: strOr(body["remark"]),
		Payload: gin.H{"reweigh_kg": reweigh, "photo_url": photo},
	})
	api.OK(c, s.loadProcessIssueRow(id))
	return true
}

func (s *Services) handleIssueReturnApprove(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["biz_status"]) != "return_pending" {
		api.FailJSON(c, "NOT_RETURN_PENDING")
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	approveKg, _ := asFloat(body["reweigh_kg"])
	pend := asFloatOr0(row["pending_return_kg"])
	if approveKg <= kgEps {
		approveKg = pend
	}
	approveKg = roundKg(approveKg)
	if approveKg <= kgEps || approveKg-pend > kgEps {
		api.FailJSON(c, "INVALID_QTY")
		return true
	}
	retKg := asFloatOr0(row["returned_kg"])
	newRet := roundKg(retKg + approveKg)
	issueKg := asFloatOr0(row["issue_kg"])
	doneKg := asFloatOr0(row["completed_kg"])
	st := "open"
	if issueRemain(issueKg, newRet, doneKg) <= kgEps {
		st = "closed"
	}
	_, err := s.DB.Exec(`UPDATE pd_process_issue SET returned_kg=?, status=?, biz_status='open',
		pending_return_kg=0, pending_reweigh_kg=0, pending_photo_url='', pending_return_by=0, pending_remark='',
		updated_at=NOW() WHERE id=?`, newRet, st, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	// return weight to board available pool
	boardID := asInt64Or0(row["board_id"])
	processID := asInt64Or0(row["process_id"])
	_, _ = s.DB.Exec(`UPDATE inv_box_code SET weight=COALESCE(weight,0)+?, qty=COALESCE(qty,0)+?,
		current_process_id=COALESCE(NULLIF(current_process_id,0),?), updated_at=NOW() WHERE id=?`,
		approveKg, approveKg, processID, boardID)
	empID, _, _, _ := s.currentEmployeeInfo(c)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "return_approve", BoardID: boardID, BoardCode: strOr(row["board_code"]),
		TraceCode: strOr(row["trace_code"]), ProcessID: processID,
		WorkerID: asInt64Or0(row["worker_id"]), ActorUserID: claimsUserID(c), OperatorEmployeeID: empID,
		Kg: approveKg, RefType: "pd_process_issue", RefID: id,
		Payload: gin.H{"photo_url": strOrDef(body["photo_url"], strOr(body["image_url"]))},
	})
	api.OK(c, s.loadProcessIssueRow(id))
	return true
}

func (s *Services) handleIssueReturnReject(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin") {
		return true
	}
	id := paramID(c)
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["biz_status"]) != "return_pending" {
		api.FailJSON(c, "NOT_RETURN_PENDING")
		return true
	}
	_, err := s.DB.Exec(`UPDATE pd_process_issue SET biz_status='open', pending_return_kg=0, pending_reweigh_kg=0,
		pending_photo_url='', pending_return_by=0, pending_remark=?, updated_at=NOW() WHERE id=?`,
		strOr(bindBody(c)["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadProcessIssueRow(id))
	return true
}

func (s *Services) handleIssueConfirmDone(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman") {
		return true
	}
	id := paramID(c)
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["biz_status"]) == "work_done" {
		api.OK(c, row)
		return true
	}
	if strOr(row["biz_status"]) == "return_pending" {
		api.FailJSON(c, "RETURN_PENDING")
		return true
	}
	empID, _, _, _ := s.currentEmployeeInfo(c)
	_, err := s.DB.Exec(`UPDATE pd_process_issue SET biz_status='work_done', work_done_by=?, work_done_at=NOW(), updated_at=NOW() WHERE id=?`,
		empID, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "work_done", BoardID: asInt64Or0(row["board_id"]), BoardCode: strOr(row["board_code"]),
		TraceCode: strOr(row["trace_code"]), ProcessID: asInt64Or0(row["process_id"]),
		WorkerID: asInt64Or0(row["worker_id"]), ActorUserID: claimsUserID(c), OperatorEmployeeID: empID,
		RefType: "pd_process_issue", RefID: id,
	})
	api.OK(c, s.loadProcessIssueRow(id))
	return true
}

func (s *Services) creditUpstreamOutputOnIssue(board *boardState, toProcessID int64, kg float64, toWorkerID int64) {
	if board == nil || kg <= kgEps || board.ProcessID <= 0 || board.ProcessID == toProcessID {
		return
	}
	fromProcess := board.ProcessID
	need := kg
	rows, err := s.DB.Query(`SELECT id, issue_kg, returned_kg, completed_kg FROM pd_process_issue
		WHERE board_id=? AND process_id=? AND status='open' AND COALESCE(worker_id,0)>0
		ORDER BY created_at, id`, board.ID, fromProcess)
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
	_, _ = s.DB.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, to_process_id, to_worker_id, kg, move_kind, created_by)
		VALUES(?,?,?,?,?,?,?,'next',?)`, board.ID, board.Code, board.Trace, fromProcess, toProcessID, toWorkerID, kg, 0)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "next_claim", BoardID: board.ID, BoardCode: board.Code, TraceCode: board.Trace,
		ProcessID: fromProcess, WorkerID: toWorkerID, Kg: kg,
		Payload: gin.H{"to_process_id": toProcessID, "as_upstream_output": true},
	})
}

func fmtDocNo(prefix string) string {
	return fmt.Sprintf("%s%s", prefix, time.Now().Format("060102150405"))
}

func (s *Services) handleIssueWarehouseApprove(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["biz_status"]) != "issue_pending_warehouse" {
		api.FailJSON(c, "NOT_ISSUE_PENDING")
		return true
	}
	boardCode := strings.TrimSpace(strOrDef(body["board_code"], strOr(body["box_code"])))
	if boardCode == "" {
		api.FailJSON(c, "BOX_REQUIRED")
		return true
	}
	if fail := s.requireReweighFields(body); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	approveKg, _ := asFloat(body["reweigh_kg"])
	if approveKg <= kgEps {
		approveKg = asFloatOr0(row["issue_kg"])
	}
	approveKg = roundKg(approveKg)
	if approveKg <= kgEps {
		api.FailJSON(c, "INVALID_QTY")
		return true
	}
	board, errMsg := s.loadBoardByCode(boardCode)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	s.sanitizeBoardProcessRefs(board)
	trace := strings.ToUpper(strings.TrimSpace(strOr(row["trace_code"])))
	if be := assertBoardTraceForIssue(board, trace); be != nil {
		api.HandleBusiness(c, be, nil)
		return true
	}
	if board.Status == "finished" {
		api.FailJSON(c, "BOARD_FINISHED")
		return true
	}
	processID := asInt64Or0(row["process_id"])
	workerID := asInt64Or0(row["worker_id"])
	if fail := s.assertProcessTransitionAllowed(board, processID); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	prevProcess := board.ProcessID
	opEmpID, _, _, _ := s.currentEmployeeInfo(c)
	fail, boardOut := s.completeWarehousePendingIssue(id, board, workerID, processID, 0, approveKg, opEmpID)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	if err := s.writeProcessIssueStockOut(boardOut, approveKg, id, trace); err != nil {
		api.FailJSON(c, "STOCK_OUT_FAILED:"+err.Error())
		return true
	}
	if prevProcess > 0 && prevProcess != processID {
		s.creditUpstreamOutputOnIssue(&boardState{
			ID: boardOut.ID, Code: boardOut.Code, Trace: boardOut.Trace, ProcessID: prevProcess, StepID: boardOut.StepID,
		}, processID, approveKg, workerID)
	}
	photo := strings.TrimSpace(strOrDef(body["photo_url"], strOr(body["image_url"])))
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "issue_approve", BoardID: boardOut.ID, BoardCode: boardOut.Code, TraceCode: trace,
		ProcessID: processID, WorkerID: workerID, ActorUserID: claimsUserID(c), OperatorEmployeeID: opEmpID,
		Kg: approveKg, RefType: "pd_process_issue", RefID: id,
		Payload: gin.H{"photo_url": photo, "assigned_board_code": boardCode},
	})
	out := s.loadProcessIssueRow(id)
	s.attachIssueWagePreview(out, processID, approveKg)
	api.OK(c, out)
	return true
}

func (s *Services) handleIssueWarehouseReject(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin") {
		return true
	}
	id := paramID(c)
	row := s.loadProcessIssueRow(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["biz_status"]) != "issue_pending_warehouse" {
		api.FailJSON(c, "NOT_ISSUE_PENDING")
		return true
	}
	_, err := s.DB.Exec(`UPDATE pd_process_issue SET biz_status='issue_rejected', status='closed',
		pending_reweigh_kg=0, pending_photo_url='', pending_remark=?, updated_at=NOW() WHERE id=?`,
		strOr(bindBody(c)["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	empID, _, _, _ := s.currentEmployeeInfo(c)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "issue_reject", TraceCode: strOr(row["trace_code"]), ProcessID: asInt64Or0(row["process_id"]),
		WorkerID: asInt64Or0(row["worker_id"]), ActorUserID: claimsUserID(c), OperatorEmployeeID: empID,
		RefType: "pd_process_issue", RefID: id,
	})
	api.OK(c, s.loadProcessIssueRow(id))
	return true
}
