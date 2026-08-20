package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/notify"
)

func (s *Services) handleMaterialDispatches(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case method == "GET" && strings.Contains(openapiPath, "/wage-summary"):
		return s.listMaterialDispatchWageSummary(c)
	case method == "GET":
		return s.listMaterialDispatches(c)
	case method == "POST" && strings.Contains(openapiPath, "/complete"):
		return s.completeMaterialDispatch(c)
	case method == "POST":
		return s.createMaterialDispatch(c)
	default:
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
}

func (s *Services) createMaterialDispatch(c *gin.Context) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	if !s.requireAnyRole(c, "foreman", "planner", "sys_admin", "admin") {
		return true
	}
	body := bindBody(c)
	boardCode := strings.TrimSpace(strOrDef(body["board_code"], strOr(body["box_code"])))
	if boardCode == "" {
		api.FailJSON(c, "BOX_REQUIRED")
		return true
	}
	board, errMsg := s.loadBoardByCode(boardCode)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
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
	if fail := s.assertProcessTransitionAllowed(board, processID); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	kg, ok := asFloat(body["weight_kg"])
	if !ok || kg <= kgEps {
		kg, ok = asFloat(body["kg"])
	}
	if !ok || kg <= kgEps {
		api.FailJSON(c, "INVALID_QTY")
		return true
	}
	reweigh, _ := asFloat(body["reweigh_kg"])
	if reweigh <= kgEps {
		api.FailJSON(c, "REWEIGH_REQUIRED")
		return true
	}
	photo := strings.TrimSpace(strOrDef(body["photo_url"], strOr(body["image_url"])))
	if photo == "" {
		api.FailJSON(c, "REWEIGH_PHOTO_REQUIRED")
		return true
	}
	source := strings.ToLower(strings.TrimSpace(strOrDef(body["source_kind"], "warehouse")))
	if source != "warehouse" && source != "process" {
		api.FailJSON(c, "INVALID_SOURCE_KIND")
		return true
	}
	whAvail := roundKg(board.Weight)
	procAvail := s.poolOpenKg(board.ID, processID) + s.processOpenKg(board.ID, processID)
	if source == "warehouse" && kg-whAvail > kgEps {
		if procAvail+kgEps >= kg {
			api.FailJSON(c, "WAREHOUSE_INSUFFICIENT_USE_PROCESS")
			return true
		}
		api.FailJSON(c, "QTY_EXCEEDS_AVAILABLE")
		return true
	}
	if source == "process" && kg-(procAvail+whAvail) > kgEps {
		api.FailJSON(c, "QTY_EXCEEDS_AVAILABLE")
		return true
	}

	out, fail := s.issueBoardKg(board, workerID, processID, stepID, kg)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	issueID := asInt64Or0(out["id"])
	rate := s.processWageRate(processID)
	payMode := s.processPayMode(processID)
	wage := 0.0
	if s.shouldLockYieldWage(processID, workerID) {
		wage = roundMoney(kg * rate)
	}
	pname := s.processName(processID)
	dispatcher := claimsUserID(c)
	res, err := s.DB.Exec(`INSERT INTO pd_material_dispatch(
		board_id, board_code, trace_code, worker_id, worker_name, badge_code,
		process_id, process_name, weight_kg, reweigh_kg, source_kind, status,
		unit_price, wage_amount, pay_mode, issue_id, photo_url, dispatched_by, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,'in_progress',?,?,?,?,?,?,?)`,
		board.ID, board.Code, board.Trace, workerID, workerName, badge,
		processID, pname, kg, reweigh, source,
		rate, wage, payMode, issueID, photo, dispatcher, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	row := s.loadMaterialDispatch(id)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "dispatch", BoardID: board.ID, BoardCode: board.Code, TraceCode: board.Trace,
		ProcessID: processID, StepID: stepID, WorkerID: workerID, WorkerName: workerName, Badge: badge,
		ActorUserID: dispatcher, Kg: kg, PayMode: payMode, EmpType: s.workerEmpType(workerID),
		Rate: rate, Amount: wage, RefType: "pd_material_dispatch", RefID: id,
		Payload: gin.H{"source_kind": source, "reweigh_kg": reweigh, "photo_url": photo},
		After:   row,
	})
	api.OK(c, row)
	return true
}

func (s *Services) completeMaterialDispatch(c *gin.Context) bool {
	if !s.requireMobileClient(c) {
		return true
	}
	if !s.requireAnyRole(c, "foreman", "planner", "sys_admin", "admin") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	if id <= 0 {
		id = asInt64Or0(body["id"])
	}
	photo := strings.TrimSpace(strOrDef(body["confirm_photo_url"], strOr(body["photo_url"])))
	if photo == "" {
		api.FailJSON(c, "CONFIRM_PHOTO_REQUIRED")
		return true
	}
	row := s.loadMaterialDispatch(id)
	if row == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(row["status"]) != "in_progress" {
		api.FailJSON(c, "ALREADY_DONE")
		return true
	}
	uid := claimsUserID(c)
	_, err := s.DB.Exec(`UPDATE pd_material_dispatch SET status='done', confirm_photo_url=?, confirmed_by=?,
		confirmed_at=NOW(), updated_at=NOW() WHERE id=? AND status='in_progress'`, photo, uid, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	issueID := asInt64Or0(row["issue_id"])
	if issueID > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_process_issue SET status='closed', completed_kg=GREATEST(0, issue_kg-returned_kg),
			updated_at=NOW() WHERE id=? AND status='open'`, issueID)
	}
	out := s.loadMaterialDispatch(id)
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "production.dispatch_done", BizType: "material_dispatch", BizID: id,
			DocNo: fmt.Sprintf("MD%d", id), TraceCode: strOr(out["trace_code"]),
			FromRole: "foreman", ToRoles: []string{"finance", "payroll"}, CreateTask: true,
			Title: "派料完工·待计薪",
			Body: fmt.Sprintf("%s · %s · %.2fkg · 工价 %.2f",
				strOr(out["worker_name"]), strOr(out["process_name"]), asFloatOr0(out["weight_kg"]), asFloatOr0(out["wage_amount"])),
			Payload: gin.H{
				"dispatch_id": id, "worker_id": out["worker_id"], "worker_name": out["worker_name"],
				"process_id": out["process_id"], "weight_kg": out["weight_kg"],
				"unit_price": out["unit_price"], "wage_amount": out["wage_amount"],
				"biz_date": time.Now().Format("2006-01-02"),
			},
		})
	}
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "dispatch_done", BoardID: asInt64Or0(out["board_id"]), BoardCode: strOr(out["board_code"]),
		TraceCode: strOr(out["trace_code"]), ProcessID: asInt64Or0(out["process_id"]),
		WorkerID: asInt64Or0(out["worker_id"]), WorkerName: strOr(out["worker_name"]),
		ActorUserID: uid, Kg: asFloatOr0(out["weight_kg"]), Amount: asFloatOr0(out["wage_amount"]),
		RefType: "pd_material_dispatch", RefID: id, After: out,
	})
	api.OK(c, out)
	return true
}

func (s *Services) listMaterialDispatches(c *gin.Context) bool {
	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	status := strings.TrimSpace(c.Query("status"))
	dateFrom := strings.TrimSpace(c.Query("date_from"))
	dateTo := strings.TrimSpace(c.Query("date_to"))
	cl := middleware.Claims(c)
	uid := int64(0)
	if cl != nil {
		uid = cl.UserID
	}
	where := `WHERE 1=1`
	args := []interface{}{}
	if status != "" {
		where += ` AND status=?`
		args = append(args, status)
	}
	if dateFrom != "" {
		where += ` AND CAST(created_at AS TEXT)>=?`
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		where += ` AND CAST(created_at AS TEXT)<?`
		args = append(args, dateTo+"~")
	}
	switch scope {
	case "mine_dispatch":
		where += ` AND dispatched_by=?`
		args = append(args, uid)
	case "mine_work":
		var empID int64
		_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, uid).Scan(&empID)
		where += ` AND worker_id=?`
		args = append(args, empID)
	}
	rows, err := s.DB.Query(`SELECT id, board_id, board_code, trace_code, worker_id, worker_name, badge_code,
		process_id, process_name, weight_kg, reweigh_kg, source_kind, status, unit_price, wage_amount, pay_mode,
		issue_id, photo_url, confirm_photo_url, dispatched_by, confirmed_by, remark,
		COALESCE(created_at,''), COALESCE(confirmed_at,''), COALESCE(updated_at,'')
		FROM pd_material_dispatch `+where+` ORDER BY id DESC LIMIT 200`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, boardID, workerID, processID, issueID, dispatchedBy, confirmedBy int64
		var boardCode, trace, workerName, badge, processName, source, st, payMode, photo, confirmPhoto, remark, created, confirmed, updated string
		var weight, reweigh, unitPrice, wage float64
		if err := rows.Scan(&id, &boardID, &boardCode, &trace, &workerID, &workerName, &badge,
			&processID, &processName, &weight, &reweigh, &source, &st, &unitPrice, &wage, &payMode,
			&issueID, &photo, &confirmPhoto, &dispatchedBy, &confirmedBy, &remark,
			&created, &confirmed, &updated); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "board_id": boardID, "board_code": boardCode, "trace_code": trace,
			"worker_id": workerID, "worker_name": workerName, "badge_code": badge,
			"process_id": processID, "process_name": processName, "weight_kg": weight, "reweigh_kg": reweigh,
			"source_kind": source, "status": st, "unit_price": unitPrice, "wage_amount": wage, "pay_mode": payMode,
			"issue_id": issueID, "photo_url": photo, "confirm_photo_url": confirmPhoto,
			"dispatched_by": dispatchedBy, "confirmed_by": confirmedBy, "remark": remark,
			"created_at": created, "confirmed_at": confirmed, "updated_at": updated,
		})
	}
	api.OK(c, gin.H{"items": list, "total": len(list)})
	return true
}

func (s *Services) listMaterialDispatchWageSummary(c *gin.Context) bool {
	dateFrom := strings.TrimSpace(c.Query("date_from"))
	dateTo := strings.TrimSpace(c.Query("date_to"))
	if dateFrom == "" {
		dateFrom = time.Now().Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = dateFrom
	}
	workerID := asInt64Or0(c.Query("worker_id"))
	where := `WHERE status='done' AND CAST(COALESCE(confirmed_at, created_at) AS TEXT)>=? AND CAST(COALESCE(confirmed_at, created_at) AS TEXT)<?`
	args := []interface{}{dateFrom, dateTo + "~"}
	if workerID > 0 {
		where += ` AND worker_id=?`
		args = append(args, workerID)
	}
	rows, err := s.DB.Query(`SELECT worker_id, MAX(worker_name), COALESCE(SUM(weight_kg),0), COALESCE(SUM(wage_amount),0), COUNT(1)
		FROM pd_material_dispatch `+where+` GROUP BY worker_id ORDER BY worker_id`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var wid, cnt int64
		var name string
		var kg, wage float64
		if err := rows.Scan(&wid, &name, &kg, &wage, &cnt); err != nil {
			continue
		}
		list = append(list, gin.H{
			"worker_id": wid, "worker_name": name, "weight_kg": kg, "wage_amount": wage, "count": cnt,
			"date_from": dateFrom, "date_to": dateTo,
		})
	}
	api.OK(c, gin.H{"items": list, "date_from": dateFrom, "date_to": dateTo})
	return true
}

func (s *Services) loadMaterialDispatch(id int64) gin.H {
	if id <= 0 {
		return nil
	}
	var boardID, workerID, processID, issueID, dispatchedBy, confirmedBy int64
	var boardCode, trace, workerName, badge, processName, source, st, payMode, photo, confirmPhoto, remark, created, confirmed, updated string
	var weight, reweigh, unitPrice, wage float64
	err := s.DB.QueryRow(`SELECT id, board_id, board_code, trace_code, worker_id, worker_name, badge_code,
		process_id, process_name, weight_kg, reweigh_kg, source_kind, status, unit_price, wage_amount, pay_mode,
		issue_id, photo_url, confirm_photo_url, dispatched_by, confirmed_by, remark,
		COALESCE(created_at,''), COALESCE(confirmed_at,''), COALESCE(updated_at,'')
		FROM pd_material_dispatch WHERE id=?`, id).
		Scan(&id, &boardID, &boardCode, &trace, &workerID, &workerName, &badge,
			&processID, &processName, &weight, &reweigh, &source, &st, &unitPrice, &wage, &payMode,
			&issueID, &photo, &confirmPhoto, &dispatchedBy, &confirmedBy, &remark,
			&created, &confirmed, &updated)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "board_id": boardID, "board_code": boardCode, "trace_code": trace,
		"worker_id": workerID, "worker_name": workerName, "badge_code": badge,
		"process_id": processID, "process_name": processName, "weight_kg": weight, "reweigh_kg": reweigh,
		"source_kind": source, "status": st, "unit_price": unitPrice, "wage_amount": wage, "pay_mode": payMode,
		"issue_id": issueID, "photo_url": photo, "confirm_photo_url": confirmPhoto,
		"dispatched_by": dispatchedBy, "confirmed_by": confirmedBy, "remark": remark,
		"created_at": created, "confirmed_at": confirmed, "updated_at": updated,
	}
}

func (s *Services) processName(processID int64) string {
	if processID <= 0 {
		return ""
	}
	var name string
	_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, processID).Scan(&name)
	return name
}

func (s *Services) processType(processID int64) string {
	if processID <= 0 {
		return ""
	}
	var t string
	_ = s.DB.QueryRow(`SELECT COALESCE(process_type,'') FROM pd_process WHERE id=?`, processID).Scan(&t)
	return strings.ToLower(strings.TrimSpace(t))
}

func (s *Services) assertProcessTransitionAllowed(board *boardState, toProcessID int64) string {
	if board == nil || toProcessID <= 0 {
		return ""
	}
	toType := s.processType(toProcessID)
	if toType == "inbound" {
		st := strings.ToLower(strings.TrimSpace(board.Status))
		if st == "in_stock" || st == "stocked" || st == "stored" {
			return "ALREADY_IN_STOCK"
		}
		fromType := s.processType(board.ProcessID)
		if fromType == "inbound" && board.ProcessID == toProcessID {
			return "SAME_PROCESS_FORBIDDEN"
		}
		if fromType == "inbound" && board.Weight > kgEps && board.ProcessID > 0 && board.ProcessID != toProcessID {
			return "ALREADY_IN_STOCK"
		}
	}
	if board.ProcessID > 0 && board.ProcessID == toProcessID {
		fromType := s.processType(board.ProcessID)
		if fromType == "inbound" || fromType == "outbound" || fromType == "gate" {
			return "SAME_PROCESS_FORBIDDEN"
		}
	}
	return ""
}

func (s *Services) requireTraceProductionOpen(trace string) string {
	trace = strings.TrimSpace(trace)
	if trace == "" {
		return "TRACE_CODE_REQUIRED"
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_production WHERE UPPER(trace_code)=UPPER(?) AND status='in_progress'`, trace).Scan(&n)
	if n == 0 {
		return "TRACE_PRODUCTION_NOT_STARTED"
	}
	return ""
}

func (s *Services) handleTraceProduction(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case method == "GET" && strings.Contains(openapiPath, "/logs"):
		return s.listTraceProcessLogs(c)
	case method == "GET":
		return s.getTraceProduction(c)
	case method == "POST" && strings.Contains(openapiPath, "/start"):
		return s.startTraceProduction(c)
	case method == "POST" && strings.Contains(openapiPath, "/complete"):
		return s.completeTraceProduction(c)
	case method == "POST" && strings.Contains(openapiPath, "/process-start"):
		return s.logTraceProcessEvent(c, "start")
	case method == "POST" && strings.Contains(openapiPath, "/process-stop"):
		return s.logTraceProcessEvent(c, "stop")
	default:
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
}

func (s *Services) startTraceProduction(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman", "planner", "sys_admin", "admin") {
		return true
	}
	body := bindBody(c)
	trace := strings.ToUpper(strings.TrimSpace(strOrDef(body["trace_code"], strOr(body["code"]))))
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	var exist int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_trace_production WHERE UPPER(trace_code)=? AND status='in_progress'`, trace).Scan(&exist)
	if exist > 0 {
		api.OK(c, s.loadTraceProduction(exist))
		return true
	}
	uid := claimsUserID(c)
	res, err := s.DB.Exec(`INSERT INTO pd_trace_production(trace_code, status, started_by, remark)
		VALUES(?,'in_progress',?,?)`, trace, uid, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, remark)
		VALUES(?,?,?,?,?)`, id, trace, "session_start", uid, strOr(body["remark"]))
	api.OK(c, s.loadTraceProduction(id))
	return true
}

func (s *Services) completeTraceProduction(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman", "planner", "sys_admin", "admin") {
		return true
	}
	body := bindBody(c)
	trace := strings.ToUpper(strings.TrimSpace(strOrDef(body["trace_code"], strOr(body["code"]))))
	id := asInt64Or0(body["id"])
	if id <= 0 && trace != "" {
		_ = s.DB.QueryRow(`SELECT id FROM pd_trace_production WHERE UPPER(trace_code)=? AND status='in_progress'`, trace).Scan(&id)
	}
	if id <= 0 {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	row := s.loadTraceProduction(id)
	if row == nil || strOr(row["status"]) != "in_progress" {
		api.FailJSON(c, "NOT_IN_PROGRESS")
		return true
	}
	trace = strOr(row["trace_code"])
	inputKg, outputKg, lossRate := s.calcTraceSessionYield(trace)
	uid := claimsUserID(c)
	_, err := s.DB.Exec(`UPDATE pd_trace_production SET status='done', completed_by=?, completed_at=NOW(),
		input_kg=?, output_kg=?, loss_rate=?, remark=COALESCE(NULLIF(?,''), remark) WHERE id=?`,
		uid, inputKg, outputKg, lossRate, strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	_, _ = s.DB.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, input_kg, output_kg, loss_rate, remark)
		VALUES(?,?,?,?,?,?,?,?)`, id, trace, "session_complete", uid, inputKg, outputKg, lossRate, strOr(body["remark"]))
	api.OK(c, s.loadTraceProduction(id))
	return true
}

func (s *Services) logTraceProcessEvent(c *gin.Context, eventType string) bool {
	if !s.requireAnyRole(c, "foreman", "planner", "sys_admin", "admin") {
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
	var sessionID int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_trace_production WHERE UPPER(trace_code)=? AND status='in_progress'`, trace).Scan(&sessionID)
	inputKg, _ := asFloat(body["input_kg"])
	outputKg, _ := asFloat(body["output_kg"])
	loss := 0.0
	if inputKg > kgEps && outputKg >= 0 && inputKg >= outputKg {
		loss = (inputKg - outputKg) / inputKg
	}
	uid := claimsUserID(c)
	res, err := s.DB.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, process_id, process_name, event_type,
		actor_user_id, input_kg, output_kg, loss_rate, remark) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		sessionID, trace, processID, s.processName(processID), eventType, uid, inputKg, outputKg, loss, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	lid, _ := res.LastInsertId()
	api.OK(c, gin.H{
		"id": lid, "session_id": sessionID, "trace_code": trace, "process_id": processID,
		"process_name": s.processName(processID), "event_type": eventType,
		"input_kg": inputKg, "output_kg": outputKg, "loss_rate": loss,
	})
	return true
}

func (s *Services) getTraceProduction(c *gin.Context) bool {
	trace := strings.ToUpper(strings.TrimSpace(c.Query("trace_code")))
	if trace == "" {
		trace = strings.ToUpper(strings.TrimSpace(c.Query("code")))
	}
	id := paramID(c)
	if id <= 0 && trace != "" {
		_ = s.DB.QueryRow(`SELECT id FROM pd_trace_production WHERE UPPER(trace_code)=? ORDER BY id DESC LIMIT 1`, trace).Scan(&id)
	}
	if id <= 0 {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, s.loadTraceProduction(id))
	return true
}

func (s *Services) listTraceProcessLogs(c *gin.Context) bool {
	trace := strings.ToUpper(strings.TrimSpace(c.Query("trace_code")))
	sessionID := asInt64Or0(c.Query("session_id"))
	where := `WHERE 1=1`
	args := []interface{}{}
	if sessionID > 0 {
		where += ` AND session_id=?`
		args = append(args, sessionID)
	}
	if trace != "" {
		where += ` AND UPPER(trace_code)=?`
		args = append(args, trace)
	}
	rows, err := s.DB.Query(`SELECT id, session_id, trace_code, process_id, process_name, event_type, actor_user_id,
		input_kg, output_kg, loss_rate, remark, COALESCE(created_at,'')
		FROM pd_trace_process_log `+where+` ORDER BY id DESC LIMIT 300`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, sid, pid, actor int64
		var code, pname, et, remark, created string
		var inKg, outKg, loss float64
		if err := rows.Scan(&id, &sid, &code, &pid, &pname, &et, &actor, &inKg, &outKg, &loss, &remark, &created); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "session_id": sid, "trace_code": code, "process_id": pid, "process_name": pname,
			"event_type": et, "actor_user_id": actor, "input_kg": inKg, "output_kg": outKg,
			"loss_rate": loss, "remark": remark, "created_at": created,
		})
	}
	api.OK(c, gin.H{"items": list})
	return true
}

func (s *Services) loadTraceProduction(id int64) gin.H {
	if id <= 0 {
		return nil
	}
	var startedBy, completedBy int64
	var trace, status, remark, startedAt, completedAt string
	var loss, inKg, outKg float64
	err := s.DB.QueryRow(`SELECT id, trace_code, status, started_by, completed_by, COALESCE(started_at,''), COALESCE(completed_at,''),
		COALESCE(remark,''), loss_rate, input_kg, output_kg FROM pd_trace_production WHERE id=?`, id).
		Scan(&id, &trace, &status, &startedBy, &completedBy, &startedAt, &completedAt, &remark, &loss, &inKg, &outKg)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "trace_code": trace, "status": status, "started_by": startedBy, "completed_by": completedBy,
		"started_at": startedAt, "completed_at": completedAt, "remark": remark,
		"loss_rate": loss, "input_kg": inKg, "output_kg": outKg,
	}
}

func (s *Services) calcTraceSessionYield(trace string) (inputKg, outputKg, lossRate float64) {
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(weight),0) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)`, trace).Scan(&inputKg)
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(kg),0) FROM pd_process_move
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND move_kind='stock_in'`, trace).Scan(&outputKg)
	if inputKg > kgEps && inputKg >= outputKg {
		lossRate = (inputKg - outputKg) / inputKg
	}
	return
}
