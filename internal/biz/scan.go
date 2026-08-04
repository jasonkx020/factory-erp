package biz

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) handleScan(c *gin.Context, resolveOnly bool) bool {
	body := bindBody(c)
	badge, _ := body["badge_code"].(string)
	box, _ := body["box_code"].(string)
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
	if badge == "" || box == "" {
		api.FailJSON(c, "BADGE_AND_BOX_REQUIRED")
		return true
	}
	var workerID int64
	var workerName string
	err := s.DB.QueryRow(`SELECT id, name FROM hr_employee WHERE badge_code=? AND status='active' AND COALESCE(is_deleted,0)=0`, badge).
		Scan(&workerID, &workerName)
	if err != nil {
		api.FailJSON(c, "BADGE_NOT_FOUND")
		return true
	}
	var boxID, productID, warehouseID, processID, stepID, taskID, woID int64
	var boxStatus string
	err = s.DB.QueryRow(`SELECT id, product_id, COALESCE(warehouse_id,0), COALESCE(current_process_id,0), COALESCE(current_step_id,0),
		COALESCE(task_id,0), COALESCE(work_order_id,0), status FROM inv_box_code WHERE code=? AND COALESCE(is_deleted,0)=0`, box).
		Scan(&boxID, &productID, &warehouseID, &processID, &stepID, &taskID, &woID, &boxStatus)
	if err != nil {
		api.FailJSON(c, "BOX_NOT_FOUND")
		return true
	}

	// locate open dispatch for current process
	var dispatchID int64
	if woID > 0 {
		_ = s.DB.QueryRow(`SELECT id FROM pd_dispatch WHERE work_order_id=? AND status IN ('dispatched','received','open') ORDER BY id DESC LIMIT 1`, woID).Scan(&dispatchID)
	}
	if processID == 0 {
		// bootstrap first step
		step := s.firstStep(1)
		if step != nil {
			processID = step.ProcessID
			stepID = step.ID
			woID, dispatchID = s.spawnNextWO(taskID, step, qty)
			taskID = 0
			if woID > 0 {
				_ = s.DB.QueryRow(`SELECT task_id FROM pd_work_order WHERE id=?`, woID).Scan(&taskID)
			}
			_, _ = s.DB.Exec(`UPDATE inv_box_code SET current_process_id=?, current_step_id=?, task_id=?, work_order_id=? WHERE id=?`,
				processID, stepID, taskID, woID, boxID)
		}
	}

	preview := gin.H{
		"worker_id": workerID, "worker_name": workerName,
		"box_code": box, "product_id": productID, "warehouse_id": warehouseID,
		"process_id": processID, "step_id": stepID, "task_id": taskID,
		"work_order_id": woID, "dispatch_id": dispatchID, "net_weight": qty,
		"input_weight": inputWeight, "output_weight": outputWeight, "loss": loss, "utilization": utilization,
	}
	if step := s.loadStep(stepID); step != nil {
		preview["step_name"] = step.StepName
		preview["is_piecework"] = step.IsPiecework
	}
	if resolveOnly {
		api.OK(c, preview)
		return true
	}
	if qty <= 0 {
		api.FailJSON(c, "INVALID_QTY")
		return true
	}

	docNo := fmt.Sprintf("RW%d", time.Now().UnixNano()%1e12)
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`INSERT INTO pd_report_work(doc_no, dispatch_id, work_order_id, process_id, worker_id, qty, weight, qty_net,
		input_weight, output_weight, loss, utilization, status, reported_at, scan_code)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'submitted',?,?)`,
		docNo, nullIf0(dispatchID), nullIf0(woID), processID, workerID, qty, outputWeight, outputWeight,
		inputWeight, outputWeight, loss, utilization, now, box)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	rid, _ := res.LastInsertId()
	if loss > 0 {
		_, _ = s.DB.Exec(`INSERT INTO pd_scrap_record(doc_no, process_id, product_id, qty, weight, disposition, status)
			VALUES(?,?,?,?,?,'process_loss','recorded')`,
			fmt.Sprintf("SC%d", time.Now().UnixNano()%1e12), processID, productID, loss, loss)
	}
	var rate float64
	_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, processID).Scan(&rate)
	amount := outputWeight * rate
	body = map[string]interface{}{
		"id": rid, "doc_no": docNo, "dispatch_id": dispatchID, "work_order_id": woID,
		"process_id": processID, "worker_id": workerID, "qty": qty, "qty_net": outputWeight,
		"input_weight": inputWeight, "output_weight": outputWeight, "loss": loss, "utilization": utilization,
		"wage_amount": amount, "rate": rate, "scan_code": box, "badge_code": badge,
	}
	d, _ := s.Store.Create("production/report-works", body, "submitted")
	if d != nil {
		body = d.Payload
	}
	flow := s.AfterReportWork(c, rid, processID, workerID, outputWeight, box, taskID, woID, stepID, inputWeight, loss, utilization)
	for k, v := range flow {
		body[k] = v
	}
	body["trace_id"] = middleware.TraceID(c)
	api.OK(c, body)
	return true
}

func (s *Services) firstStep(routingID int64) *routingStep {
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? ORDER BY seq_no LIMIT 1`, routingID).Scan(&id)
	if err != nil {
		return nil
	}
	return s.loadStep(id)
}

func (s *Services) handleBadge(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	code, _ := body["badge_code"].(string)
	if code == "" {
		api.FailJSON(c, "BADGE_REQUIRED")
		return true
	}
	_, err := s.DB.Exec(`UPDATE hr_employee SET badge_code=? WHERE id=?`, code, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	api.OK(c, gin.H{"id": id, "badge_code": code})
	return true
}
