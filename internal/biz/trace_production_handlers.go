package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

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
	case method == "GET" && strings.Contains(openapiPath, "/material-locations"):
		return s.getTraceMaterialLocations(c)
	case method == "GET" && (strings.Contains(openapiPath, "/wip") || strings.HasSuffix(openapiPath, "/wip")):
		return s.getTraceProductionWip(c)
	case method == "GET" && (action == "list" || openapiPath == "/api/v1/production/trace-productions"):
		if strings.TrimSpace(c.Query("trace_code")) != "" || strings.TrimSpace(c.Query("code")) != "" || paramID(c) > 0 {
			return s.getTraceProduction(c)
		}
		return s.listTraceProductionsConsole(c)
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
	if !s.requireAnyRole(c, "foreman") {
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
	if !s.requireAnyRole(c, "foreman") {
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
	if fail := s.assertTraceCanComplete(trace); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
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
	s.snapshotTraceYield(trace)
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
	if eventType == "stop" && outputKg > kgEps {
		s.creditTraceProcessPoolKg(trace, processID, outputKg)
	}
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
		input_kg, output_kg, loss_rate, remark, COALESCE(CAST(created_at AS TEXT),'')
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
	err := s.DB.QueryRow(`SELECT id, trace_code, status, started_by, completed_by, COALESCE(CAST(started_at AS TEXT),''), COALESCE(CAST(completed_at AS TEXT),''),
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
