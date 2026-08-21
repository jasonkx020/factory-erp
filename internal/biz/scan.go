package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) currentEmployeeInfo(c *gin.Context) (empID int64, name, badge, empNo string) {
	cl := middleware.Claims(c)
	if cl == nil || cl.UserID <= 0 {
		return
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(e.id,0), COALESCE(e.name,''), COALESCE(e.badge_code,''), COALESCE(e.emp_no,'')
		FROM iam_user u LEFT JOIN hr_employee e ON e.id=u.employee_id
		WHERE u.id=? AND COALESCE(u.is_deleted,0)=0`, cl.UserID).Scan(&empID, &name, &badge, &empNo)
	return
}

// resolveScanWorker 解析真正过站人：工牌码/工号优先；未填则回落到当前登录员工。
func (s *Services) resolveScanWorker(c *gin.Context, body map[string]interface{}) (workerID int64, workerName, badgeUsed string, errMsg string) {
	badge := strings.TrimSpace(strOr(body["badge_code"]))
	if badge == "" {
		badge = strings.TrimSpace(strOr(body["worker_badge"]))
	}
	workerID, _ = asInt64(body["worker_id"])
	if badge != "" {
		err := s.DB.QueryRow(`SELECT id, name, COALESCE(badge_code,'') FROM hr_employee
			WHERE status='active' AND COALESCE(is_deleted,0)=0
			AND (badge_code=? OR emp_no=? OR lower(badge_code)=lower(?) OR lower(emp_no)=lower(?))
			ORDER BY CASE WHEN badge_code=? THEN 0 WHEN emp_no=? THEN 1 ELSE 2 END LIMIT 1`,
			badge, badge, badge, badge, badge, badge).Scan(&workerID, &workerName, &badgeUsed)
		if err != nil || workerID <= 0 {
			return 0, "", "", "BADGE_NOT_FOUND"
		}
		if badgeUsed == "" {
			badgeUsed = badge
		}
		return workerID, workerName, badgeUsed, ""
	}
	if workerID > 0 {
		err := s.DB.QueryRow(`SELECT id, name, COALESCE(badge_code,'') FROM hr_employee
			WHERE id=? AND status='active' AND COALESCE(is_deleted,0)=0`, workerID).Scan(&workerID, &workerName, &badgeUsed)
		if err != nil || workerID <= 0 {
			return 0, "", "", "BADGE_NOT_FOUND"
		}
		return workerID, workerName, badgeUsed, ""
	}
	empID, name, curBadge, _ := s.currentEmployeeInfo(c)
	if empID <= 0 {
		return 0, "", "", "BADGE_AND_BOX_REQUIRED"
	}
	return empID, name, curBadge, ""
}

func (s *Services) handleScan(c *gin.Context, resolveOnly bool) bool {
	body := bindBody(c)
	box := strings.TrimSpace(strOr(body["box_code"]))
	inputWeight, _ := asFloat(body["input_weight"])
	outputWeight, _ := asFloat(body["output_weight"])
	qty, _ := asFloat(body["net_weight"])
	if qty <= 0 {
		qty, _ = asFloat(body["qty"])
	}
	if outputWeight <= 0 {
		outputWeight = qty
	}
	if inputWeight <= 0 {
		inputWeight, _ = asFloat(body["feed_weight"])
	}
	if inputWeight <= 0 {
		inputWeight = outputWeight
	}
	loss, hasLoss := asFloat(body["loss"])
	if !hasLoss || loss < 0 {
		loss = inputWeight - outputWeight
		if loss < 0 {
			loss = 0
		}
	}
	utilization := 0.0
	if inputWeight > 0 {
		utilization = outputWeight / inputWeight
	}
	if qty <= 0 {
		qty = outputWeight
	}
	if box == "" {
		api.FailJSON(c, "BOX_REQUIRED")
		return true
	}
	workerID, workerName, badge, errMsg := s.resolveScanWorker(c, body)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}

	opUserID := claimsUserID(c)
	opEmpID, opName, _, _ := s.currentEmployeeInfo(c)

	var boxID, productID, warehouseID, processID, stepID, taskID, woID int64
	var boxStatus string
	err := s.DB.QueryRow(`SELECT id, product_id, COALESCE(warehouse_id,0), COALESCE(current_process_id,0), COALESCE(current_step_id,0),
		COALESCE(task_id,0), COALESCE(work_order_id,0), status FROM inv_box_code WHERE code=? AND COALESCE(is_deleted,0)=0`, box).
		Scan(&boxID, &productID, &warehouseID, &processID, &stepID, &taskID, &woID, &boxStatus)
	if err != nil {
		api.FailJSON(c, "BOX_NOT_FOUND")
		return true
	}

	reqProcessID, _ := asInt64(body["process_id"])
	reqStepID, _ := asInt64(body["step_id"])
	if reqProcessID <= 0 {
		api.FailJSON(c, "PROCESS_REQUIRED")
		return true
	}
	processID = reqProcessID
	if reqStepID > 0 {
		stepID = reqStepID
	} else {
		stepID = 0
	}
	if stepID > 0 && s.loadStep(stepID) == nil {
		stepID = 0
	}

	// locate open dispatch for current process
	var dispatchID int64
	if woID > 0 {
		_ = s.DB.QueryRow(`SELECT id FROM pd_dispatch WHERE work_order_id=? AND status IN ('dispatched','received','open') ORDER BY id DESC LIMIT 1`, woID).Scan(&dispatchID)
	}
	if !s.workerShiftAuthorized(workerID, processID) {
		api.FailJSON(c, "SHIFT_NOT_AUTHORIZED")
		return true
	}

	preview := gin.H{
		"worker_id": workerID, "worker_name": workerName, "badge_code": badge,
		"operator_user_id": opUserID, "operator_employee_id": opEmpID, "operator_name": opName,
		"pass_for_other": opEmpID > 0 && workerID != opEmpID,
		"box_code":       box, "product_id": productID, "warehouse_id": warehouseID,
		"process_id": processID, "step_id": stepID, "task_id": taskID,
		"work_order_id": woID, "dispatch_id": dispatchID, "net_weight": qty,
		"input_weight": inputWeight, "output_weight": outputWeight, "loss": loss, "utilization": utilization,
	}
	if step := s.loadStep(stepID); step != nil {
		preview["step_name"] = step.StepName
		preview["is_piecework"] = step.IsPiecework
		preview["is_inbound_checkpoint"] = step.IsInboundCheckpoint
	}
	s.enrichScanBoardPreview(preview, boxID, box, processID, stepID, workerID)
	if strings.TrimSpace(strOr(preview["trace_code"])) == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	if boxStatus == "finished" {
		api.FailJSON(c, "BOARD_FINISHED")
		return true
	}
	if resolveOnly {
		api.OK(c, preview)
		return true
	}
	// 旧扫码写报工路径已下线；现场提交走 /production/board-issues。
	api.FailJSON(c, "FEATURE_REMOVED:scan_submit;use_board_issues")
	return true
}

func (s *Services) resolveRoutingID(taskID, productID int64) int64 {
	if taskID > 0 {
		var rid int64
		_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_production_task WHERE id=?`, taskID).Scan(&rid)
		if rid > 0 {
			return rid
		}
	}
	if productID > 0 {
		var rid int64
		_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE product_id=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, productID).Scan(&rid)
		if rid > 0 {
			return rid
		}
	}
	var rid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_flow_graph WHERE kind='production' AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`).Scan(&rid)
	if rid > 0 {
		return rid
	}
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE code='RT-CASSAVA' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`).Scan(&rid)
	if rid > 0 {
		return rid
	}
	return 1
}

func (s *Services) stepByProcess(routingID, processID int64) *routingStep {
	if routingID <= 0 || processID <= 0 {
		return nil
	}
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? AND process_id=? ORDER BY seq_no LIMIT 1`, routingID, processID).Scan(&id)
	if err != nil {
		return nil
	}
	return s.loadStep(id)
}

func (s *Services) firstStep(routingID int64) *routingStep {
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? ORDER BY seq_no LIMIT 1`, routingID).Scan(&id)
	if err != nil {
		return nil
	}
	return s.loadStep(id)
}

// resolveInboundEntryStep 按产品绑定的 active 工艺取首步；无产品/无工艺则失败（不静默回退成品线）。
func (s *Services) resolveInboundEntryStep(productID int64) (processID, stepID, warehouseID int64, errMsg string) {
	if productID <= 0 {
		return 0, 0, 0, "PRODUCT_REQUIRED"
	}
	var rid int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE product_id=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, productID).Scan(&rid)
	if rid <= 0 {
		return 0, 0, 0, "ROUTING_REQUIRED"
	}
	step := s.firstStep(rid)
	if step == nil {
		return 0, 0, 0, "ROUTING_REQUIRED"
	}
	return step.ProcessID, step.ID, step.WarehouseID, ""
}

func (s *Services) handleBadge(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	var empNo, cur string
	err := s.DB.QueryRow(`SELECT COALESCE(emp_no,''), COALESCE(badge_code,'') FROM hr_employee WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&empNo, &cur)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	// 默认自动生成/重新生成；仅当显式传入非空 badge_code 且未要求 regenerate 时沿用（兼容旧调用）
	code := strings.TrimSpace(strOr(body["badge_code"]))
	regenerate := boolOr(body["regenerate"], false) || code == ""
	if regenerate {
		code = s.allocBadgeCode(empNo, id)
	}
	_, err = s.DB.Exec(`UPDATE hr_employee SET badge_code=?, updated_at=NOW() WHERE id=?`, code, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	api.OK(c, gin.H{"id": id, "badge_code": code, "previous_badge_code": cur})
	return true
}
