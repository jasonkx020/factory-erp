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
		return ""
	}
	routingID := s.resolveRoutingID(board.TaskID, board.ProductID)
	if routingID <= 0 {
		return ""
	}
	curProc := board.ProcessID
	if curProc <= 0 {
		first := s.firstStep(routingID)
		if first != nil && first.ProcessID == toProcessID {
			return ""
		}
		return "ROUTING_TRANSITION_FORBIDDEN"
	}
	curStep := s.stepByProcess(routingID, curProc)
	if curStep == nil {
		curStep = s.loadStep(board.StepID)
	}
	if curStep != nil {
		if curStep.ProcessID == toProcessID {
			return ""
		}
		next := s.nextStep(curStep)
		if next != nil && next.ProcessID == toProcessID {
			return ""
		}
	}
	return "ROUTING_TRANSITION_FORBIDDEN"
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
	case method == "GET" && strings.Contains(openapiPath, "/report"):
		return s.getTraceProductionReport(c)
	case method == "GET" && strings.Contains(openapiPath, "/material-locations"):
		return s.getTraceMaterialLocations(c)
	case method == "GET" && strings.Contains(openapiPath, "/routing-options"):
		return s.getTraceProductionRoutingOptions(c)
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
	case method == "POST" && strings.Contains(openapiPath, "/process-complete"):
		return s.completeTraceProcess(c)
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
	if !s.requireAnyRole(c, "foreman", "planner", "admin") {
		return true
	}
	body := bindBody(c)
	trace := strings.ToUpper(strings.TrimSpace(strOrDef(body["trace_code"], strOr(body["code"]))))
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	routingID := asInt64Or0(body["routing_id"])
	var exist int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_trace_production WHERE UPPER(trace_code)=? AND status='in_progress'`, trace).Scan(&exist)
	if exist > 0 {
		api.OK(c, s.loadTraceProduction(exist))
		return true
	}
	productID, fail := s.validateTraceRoutingStart(trace, routingID)
	if fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	rCode, rName, _ := s.loadRoutingMeta(routingID)
	uid := claimsUserID(c)
	remark := strOr(body["remark"])
	if rCode != "" {
		if remark != "" {
			remark += " | "
		}
		remark += "routing=" + rCode
		if rName != "" {
			remark += "(" + rName + ")"
		}
	}
	res, err := s.DB.Exec(`INSERT INTO pd_trace_production(trace_code, status, started_by, routing_id, product_id, remark)
		VALUES(?,'in_progress',?,?,?,?)`, trace, uid, routingID, productID, remark)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, remark)
		VALUES(?,?,?,?,?)`, id, trace, "session_start", uid, remark)
	api.OK(c, s.loadTraceProduction(id))
	return true
}

func (s *Services) getTraceProductionRoutingOptions(c *gin.Context) bool {
	trace := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	if trace == "" {
		trace = strings.ToUpper(strings.TrimSpace(c.Query("trace_code")))
	}
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	productID := s.resolveTraceProductID(trace)
	if productID <= 0 {
		api.FailJSON(c, "PRODUCT_REQUIRED")
		return true
	}
	_, pname, _ := s.productMeta(productID)
	options := s.listRoutingsForProduct(productID)
	suggested := s.resolveRoutingID(0, productID)
	api.OK(c, gin.H{
		"trace_code": trace, "product_id": productID, "product_name": pname,
		"suggested_routing_id": suggested, "routing_options": options,
	})
	return true
}

func (s *Services) completeTraceProduction(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman", "planner", "admin") {
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
	_, _, _, routingSteps := s.resolveTraceRoutingSteps(trace)
	if len(routingSteps) > 0 && !s.allRoutingStepsComplete(trace) {
		api.FailJSON(c, "ROUTING_STEPS_INCOMPLETE")
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
	var startedBy, completedBy, routingID, productID int64
	var trace, status, remark, startedAt, completedAt string
	var loss, inKg, outKg float64
	err := s.DB.QueryRow(`SELECT id, trace_code, status, started_by, completed_by, COALESCE(CAST(started_at AS TEXT),''), COALESCE(CAST(completed_at AS TEXT),''),
		COALESCE(remark,''), loss_rate, input_kg, output_kg, COALESCE(routing_id,0), COALESCE(product_id,0) FROM pd_trace_production WHERE id=?`, id).
		Scan(&id, &trace, &status, &startedBy, &completedBy, &startedAt, &completedAt, &remark, &loss, &inKg, &outKg, &routingID, &productID)
	if err != nil {
		// Fallback for DB without new columns yet.
		err = s.DB.QueryRow(`SELECT id, trace_code, status, started_by, completed_by, COALESCE(CAST(started_at AS TEXT),''), COALESCE(CAST(completed_at AS TEXT),''),
			COALESCE(remark,''), loss_rate, input_kg, output_kg FROM pd_trace_production WHERE id=?`, id).
			Scan(&id, &trace, &status, &startedBy, &completedBy, &startedAt, &completedAt, &remark, &loss, &inKg, &outKg)
		if err != nil {
			return nil
		}
	}
	rCode, rName, _ := s.loadRoutingMeta(routingID)
	out := gin.H{
		"id": id, "trace_code": trace, "status": status, "started_by": startedBy, "completed_by": completedBy,
		"started_at": startedAt, "completed_at": completedAt, "remark": remark,
		"loss_rate": loss, "input_kg": inKg, "output_kg": outKg,
		"routing_id": routingID, "product_id": productID,
	}
	if rCode != "" {
		out["routing_code"] = rCode
		out["routing_name"] = rName
	}
	return out
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

func (s *Services) finalizeTraceProduction(trace string, sessionID, actorUID int64, remark string) string {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if fail := s.assertTraceCanComplete(trace); fail != "" {
		return fail
	}
	_, _, _, routingSteps := s.resolveTraceRoutingSteps(trace)
	if len(routingSteps) > 0 && !s.allRoutingStepsComplete(trace) {
		return "ROUTING_STEPS_INCOMPLETE"
	}
	inputKg, outputKg, lossRate := s.calcTraceSessionYield(trace)
	_, err := s.DB.Exec(`UPDATE pd_trace_production SET status='done', completed_by=?, completed_at=NOW(),
		input_kg=?, output_kg=?, loss_rate=?, remark=COALESCE(NULLIF(?,''), remark) WHERE id=? AND status='in_progress'`,
		actorUID, inputKg, outputKg, lossRate, remark, sessionID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	s.snapshotTraceYield(trace)
	_, _ = s.DB.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, input_kg, output_kg, loss_rate, remark)
		VALUES(?,?,?,?,?,?,?,?)`, sessionID, trace, "session_complete", actorUID, inputKg, outputKg, lossRate, remark)
	return ""
}

func (s *Services) completeTraceProcess(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman", "planner", "admin") {
		return true
	}
	body := bindBody(c)
	trace := strings.ToUpper(strings.TrimSpace(strOrDef(body["trace_code"], strOr(body["code"]))))
	processID := asInt64Or0(body["process_id"])
	if processID <= 0 {
		stepID := asInt64Or0(body["routing_step_id"])
		if stepID > 0 {
			var pid int64
			_ = s.DB.QueryRow(`SELECT process_id FROM pd_routing_step WHERE id=?`, stepID).Scan(&pid)
			processID = pid
		}
	}
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	if processID <= 0 {
		api.FailJSON(c, "PROCESS_REQUIRED")
		return true
	}
	if fail := s.requireTraceProductionOpen(trace); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	if s.isTraceProcessComplete(trace, processID) {
		yields := s.traceProcessYields(trace)
		y := yields[processID]
		api.OK(c, gin.H{
			"trace_code": trace, "process_id": processID, "already_done": true,
			"input_kg": asFloatOr0(y["input_kg"]), "output_kg": asFloatOr0(y["output_kg"]),
			"loss_kg": asFloatOr0(y["loss_kg"]), "loss_rate": asFloatOr0(y["loss_rate"]),
		})
		return true
	}
	if fail := s.assertPriorRoutingStepsDone(trace, processID); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	wipMap := s.computeTraceProcessWip(trace)
	if fail := s.assertTraceProcessCanComplete(trace, processID, wipMap); fail != "" {
		api.FailJSON(c, fail)
		return true
	}
	var sessionID int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_trace_production WHERE UPPER(trace_code)=UPPER(?) AND status='in_progress'`, trace).Scan(&sessionID)
	inKg, outKg, lossKg, rate := s.snapshotTraceProcessYield(trace, processID)
	uid := claimsUserID(c)
	_, err := s.DB.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, process_id, process_name, event_type,
		actor_user_id, input_kg, output_kg, loss_rate, remark) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		sessionID, trace, processID, s.processName(processID), "process_complete", uid, inKg, outKg, rate, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	autoFinalized := false
	if s.isLastRoutingProcess(trace, processID) {
		if fail := s.finalizeTraceProduction(trace, sessionID, uid, strOr(body["remark"])); fail != "" {
			api.FailJSON(c, fail)
			return true
		}
		autoFinalized = true
	}
	routingSteps, _, canPID := s.buildTraceRoutingTimeline(trace, s.computeTraceProcessWip(trace))
	api.OK(c, gin.H{
		"trace_code": trace, "process_id": processID, "process_name": s.processName(processID),
		"input_kg": inKg, "output_kg": outKg, "loss_kg": lossKg, "loss_rate": rate,
		"auto_finalized": autoFinalized, "routing_steps": routingSteps, "can_complete_process_id": canPID,
	})
	return true
}

func (s *Services) getTraceProductionReport(c *gin.Context) bool {
	trace := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	if trace == "" {
		trace = strings.ToUpper(strings.TrimSpace(c.Query("trace_code")))
	}
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	uiStatus, sessionID, sessionStatus := s.traceUIStatus(trace)
	wipMap := s.computeTraceProcessWip(trace)
	routingSteps, currentIdx, canPID := s.buildTraceRoutingTimeline(trace, wipMap)
	productID, routingID, productName, _ := s.resolveTraceRoutingSteps(trace)
	farmerID, farmerName := s.traceFarmerInfo(trace)
	var stockKg float64
	var boardCnt int
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(COALESCE(weight,qty,0)),0), COUNT(1) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		  AND COALESCE(status,'') NOT IN ('destroyed','void')`, trace).Scan(&stockKg, &boardCnt)
	session := gin.H{"id": sessionID, "status": sessionStatus, "ui_status": uiStatus}
	var startedAt, completedAt, remark string
	var startedBy, completedBy int64
	var inKg, outKg, lossRate float64
	if sessionID > 0 {
		_ = s.DB.QueryRow(`SELECT status, started_by, completed_by, COALESCE(CAST(started_at AS TEXT),''),
			COALESCE(CAST(completed_at AS TEXT),''), COALESCE(remark,''), input_kg, output_kg, loss_rate
			FROM pd_trace_production WHERE id=?`, sessionID).
			Scan(&sessionStatus, &startedBy, &completedBy, &startedAt, &completedAt, &remark, &inKg, &outKg, &lossRate)
		session = gin.H{
			"id": sessionID, "trace_code": trace, "status": sessionStatus, "ui_status": uiStatus,
			"started_by": startedBy, "completed_by": completedBy, "started_at": startedAt, "completed_at": completedAt,
			"remark": remark, "input_kg": roundKg(inKg), "output_kg": roundKg(outKg), "loss_rate": lossRate,
		}
	}
	boards := []gin.H{}
	brows, err := s.DB.Query(`SELECT id, code, COALESCE(current_process_id,0), COALESCE(weight, qty, 0), COALESCE(status,'')
		FROM inv_box_code WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		  AND COALESCE(status,'') NOT IN ('destroyed','void')`, trace)
	if err == nil {
		for brows.Next() {
			var id, pid int64
			var code, st string
			var w float64
			if err := brows.Scan(&id, &code, &pid, &w, &st); err != nil {
				continue
			}
			boards = append(boards, gin.H{
				"id": id, "code": code, "process_id": pid, "process_name": s.processName(pid),
				"weight_kg": roundKg(w), "status": st,
			})
		}
		brows.Close()
	}
	yields := []gin.H{}
	yrows, err := s.DB.Query(`SELECT y.process_id, COALESCE(p.name,''), y.input_kg, y.output_kg, y.loss_kg, y.loss_rate, y.board_count, CAST(y.created_at AS TEXT)
		FROM pd_trace_process_yield y LEFT JOIN pd_process p ON p.id=y.process_id
		WHERE UPPER(y.trace_code)=UPPER(?) ORDER BY y.process_id`, trace)
	if err == nil {
		for yrows.Next() {
			var pid int64
			var pname, created string
			var inY, outY, lossY, rateY float64
			var bc int
			if err := yrows.Scan(&pid, &pname, &inY, &outY, &lossY, &rateY, &bc, &created); err != nil {
				continue
			}
			yields = append(yields, gin.H{
				"process_id": pid, "process_name": pname, "input_kg": roundKg(inY), "output_kg": roundKg(outY),
				"loss_kg": roundKg(lossY), "loss_rate": rateY, "board_count": bc, "created_at": created,
			})
		}
		yrows.Close()
	}
	issues := []gin.H{}
	irows, err := s.DB.Query(`SELECT id, board_code, process_id, worker_id, issue_kg, returned_kg, completed_kg, status, biz_status
		FROM pd_process_issue WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) ORDER BY id DESC LIMIT 200`, trace)
	if err == nil {
		for irows.Next() {
			var id, pid, wid int64
			var bcode, st, biz string
			var issue, ret, done float64
			if err := irows.Scan(&id, &bcode, &pid, &wid, &issue, &ret, &done, &st, &biz); err != nil {
				continue
			}
			issues = append(issues, gin.H{
				"id": id, "board_code": bcode, "process_id": pid, "process_name": s.processName(pid),
				"worker_id": wid, "issue_kg": issue, "returned_kg": ret, "completed_kg": done, "status": st, "biz_status": biz,
			})
		}
		irows.Close()
	}
	moves := []gin.H{}
	mrows, err := s.DB.Query(`SELECT id, from_process_id, to_process_id, kg, move_kind, COALESCE(CAST(created_at AS TEXT),'')
		FROM pd_process_move WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) ORDER BY id DESC LIMIT 200`, trace)
	if err == nil {
		for mrows.Next() {
			var id, fromP, toP int64
			var kg float64
			var mk, created string
			if err := mrows.Scan(&id, &fromP, &toP, &kg, &mk, &created); err != nil {
				continue
			}
			moves = append(moves, gin.H{
				"id": id, "from_process_id": fromP, "from_process_name": s.processName(fromP),
				"to_process_id": toP, "to_process_name": s.processName(toP), "kg": kg, "move_kind": mk, "created_at": created,
			})
		}
		mrows.Close()
	}
	logs := []gin.H{}
	lrows, err := s.DB.Query(`SELECT id, process_id, process_name, event_type, input_kg, output_kg, loss_rate, COALESCE(CAST(created_at AS TEXT),'')
		FROM pd_trace_process_log WHERE UPPER(trace_code)=UPPER(?) ORDER BY id`, trace)
	if err == nil {
		for lrows.Next() {
			var id, pid int64
			var pname, et, created string
			var inL, outL, lossL float64
			if err := lrows.Scan(&id, &pid, &pname, &et, &inL, &outL, &lossL, &created); err != nil {
				continue
			}
			logs = append(logs, gin.H{
				"id": id, "process_id": pid, "process_name": pname, "event_type": et,
				"input_kg": inL, "output_kg": outL, "loss_rate": lossL, "created_at": created,
			})
		}
		lrows.Close()
	}
	totalLossKg := 0.0
	totalInputKg := 0.0
	for _, y := range yields {
		totalLossKg = roundKg(totalLossKg + asFloatOr0(y["loss_kg"]))
		totalInputKg = roundKg(totalInputKg + asFloatOr0(y["input_kg"]))
	}
	isFinal := strings.EqualFold(uiStatus, "ended")
	api.OK(c, gin.H{
		"trace_code": trace, "session": session,
		"trace_meta": gin.H{
			"farmer_id": farmerID, "farmer_name": farmerName, "product_id": productID, "product_name": productName,
			"routing_id": routingID, "stock_kg": roundKg(stockKg), "board_count": boardCnt,
		},
		"routing_steps": routingSteps, "current_step_index": currentIdx, "can_complete_process_id": canPID,
		"process_yields": yields, "boards": boards, "issues": issues, "moves": moves, "logs": logs,
		"summary": gin.H{
			"is_final": isFinal, "total_process_loss_kg": totalLossKg, "total_process_input_kg": totalInputKg,
			"trace_input_kg": roundKg(inKg), "trace_output_kg": roundKg(outKg), "trace_loss_rate": lossRate,
			"finished_stock_in_kg": roundKg(outKg),
		},
	})
	return true
}
