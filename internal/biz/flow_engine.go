package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

type routingStep struct {
	ID                   int64
	RoutingID            int64
	SeqNo                int
	ProcessID            int64
	StepCode             string
	StepName             string
	IsPiecework          bool
	IsInboundCheckpoint  bool
	AutoNext             bool
	AutoStockIn          bool
	AutoStockOut         bool
	WarehouseID          int64
}

// AfterReportWork drives automation after a report-work is created.
func (s *Services) AfterReportWork(c *gin.Context, reportID, processID, workerID int64, qty float64, boxCode string, taskID, workOrderID, stepID int64) map[string]interface{} {
	traceID := middleware.TraceID(c)
	out := gin.H{"report_id": reportID}

	// piecework summary
	if processID > 0 && workerID > 0 {
		var rate float64
		_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, processID).Scan(&rate)
		amount := qty * rate
		bizDate := time.Now().Format("2006-01-02")
		_, _ = s.DB.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, weight, amount, source_report_ids)
			VALUES(?,?,?,?,?,?,?)`, workerID, processID, bizDate, qty, qty, amount, fmt.Sprintf("%d", reportID))
		out["wage_amount"] = amount
		out["rate"] = rate
	}

	// accumulate task completed qty
	if taskID > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_production_task_item SET completed_qty = completed_qty + ? WHERE task_id=?`, qty, taskID)
	}

	step := s.loadStep(stepID)
	if step == nil && workOrderID > 0 {
		var sid int64
		_ = s.DB.QueryRow(`SELECT COALESCE(routing_step_id,0) FROM pd_work_order WHERE id=?`, workOrderID).Scan(&sid)
		step = s.loadStep(sid)
	}

	nextInfo := gin.H{}
	status := "ok"
	errMsg := ""
	var toStepID int64

	if step != nil {
		payload := map[string]interface{}{"box_code": boxCode, "qty": qty, "process_id": processID}
		if step.AutoStockOut {
			if err := s.autoStock(boxCode, step.WarehouseID, processID, qty, "consume"); err != nil {
				status = "failed"
				errMsg = "stock_out:" + err.Error()
			}
		}
		if step.AutoStockIn {
			wh := step.WarehouseID
			if wh == 0 {
				wh = 2
			}
			newCode, err := s.autoStockInNewBox(boxCode, wh, processID, qty, taskID, workOrderID, step.ID)
			if err != nil {
				status = "failed"
				errMsg = "stock_in:" + err.Error()
			} else {
				payload["new_box_code"] = newCode
				nextInfo["new_box_code"] = newCode
				boxCode = newCode
			}
		}
		if step.IsInboundCheckpoint {
			woID, dID := s.spawnCheckpointWO(taskID, step)
			nextInfo["checkpoint_work_order_id"] = woID
			nextInfo["checkpoint_dispatch_id"] = dID
		}
		if step.AutoNext {
			next := s.nextStep(step)
			if next != nil {
				toStepID = next.ID
				woID, dID := s.spawnNextWO(taskID, next, qty)
				nextInfo["next_step"] = next.StepName
				nextInfo["next_step_id"] = next.ID
				nextInfo["next_work_order_id"] = woID
				nextInfo["next_dispatch_id"] = dID
				_, _ = s.DB.Exec(`UPDATE inv_box_code SET current_process_id=?, current_step_id=?, work_order_id=?, updated_at=datetime('now') WHERE code=?`,
					next.ProcessID, next.ID, woID, boxCode)
			} else {
				nextInfo["finished"] = true
				_, _ = s.DB.Exec(`UPDATE inv_box_code SET status='finished', updated_at=datetime('now') WHERE code=?`, boxCode)
			}
		} else {
			nextInfo["finished"] = true
			_, _ = s.DB.Exec(`UPDATE inv_box_code SET status='finished', updated_at=datetime('now') WHERE code=?`, boxCode)
		}
		b, _ := json.Marshal(payload)
		evID := s.writeFlowEvent("report_work", reportID, step.ID, toStepID, "after_report", traceID, status, errMsg, string(b))
		nextInfo["flow_event_id"] = evID
	}

	out["next"] = nextInfo
	out["trace_id"] = traceID
	return out
}

func (s *Services) loadStep(id int64) *routingStep {
	if id <= 0 {
		return nil
	}
	var st routingStep
	var piece, cp, an, asi, aso int
	err := s.DB.QueryRow(`SELECT id, routing_id, seq_no, process_id, COALESCE(step_code,''), COALESCE(step_name,''),
		COALESCE(is_piecework,0), COALESCE(is_inbound_checkpoint,0), COALESCE(auto_next,1),
		COALESCE(auto_stock_in,0), COALESCE(auto_stock_out,0), COALESCE(warehouse_id,0)
		FROM pd_routing_step WHERE id=?`, id).
		Scan(&st.ID, &st.RoutingID, &st.SeqNo, &st.ProcessID, &st.StepCode, &st.StepName, &piece, &cp, &an, &asi, &aso, &st.WarehouseID)
	if err != nil {
		return nil
	}
	st.IsPiecework = piece == 1
	st.IsInboundCheckpoint = cp == 1
	st.AutoNext = an == 1
	st.AutoStockIn = asi == 1
	st.AutoStockOut = aso == 1
	return &st
}

func (s *Services) nextStep(cur *routingStep) *routingStep {
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? AND seq_no>? ORDER BY seq_no LIMIT 1`, cur.RoutingID, cur.SeqNo).Scan(&id)
	if err != nil {
		return nil
	}
	return s.loadStep(id)
}

func (s *Services) spawnNextWO(taskID int64, step *routingStep, qty float64) (int64, int64) {
	if taskID == 0 {
		// create lightweight task if missing
		docNo := fmt.Sprintf("PT%d", time.Now().UnixNano()%1e12)
		res, err := s.DB.Exec(`INSERT INTO pd_production_task(doc_no, status, routing_id, workshop_id) VALUES(?,'in_progress',?,1)`, docNo, step.RoutingID)
		if err == nil {
			taskID, _ = res.LastInsertId()
			_, _ = s.DB.Exec(`INSERT INTO pd_production_task_item(task_id, product_id, plan_qty) VALUES(?,3,?)`, taskID, qty)
		}
	}
	woNo := fmt.Sprintf("WO%d", time.Now().UnixNano()%1e12)
	res, err := s.DB.Exec(`INSERT INTO pd_work_order(doc_no, task_id, process_id, routing_step_id, status, plan_qty) VALUES(?,?,?,?,'pending',?)`,
		woNo, taskID, step.ProcessID, step.ID, qty)
	if err != nil {
		return taskID, 0
	}
	woID, _ := res.LastInsertId()
	dpNo := fmt.Sprintf("DP%d", time.Now().UnixNano()%1e12)
	dres, _ := s.DB.Exec(`INSERT INTO pd_dispatch(doc_no, work_order_id, plan_qty, status, dispatched_at) VALUES(?,?,?,'dispatched',datetime('now'))`,
		dpNo, woID, qty)
	dID, _ := dres.LastInsertId()
	return woID, dID
}

func (s *Services) spawnCheckpointWO(taskID int64, step *routingStep) (int64, int64) {
	return s.spawnNextWO(taskID, step, 0)
}

func (s *Services) autoStock(boxCode string, warehouseID, processID int64, qty float64, txnType string) error {
	var productID, wh int64
	_ = s.DB.QueryRow(`SELECT product_id, COALESCE(warehouse_id,1) FROM inv_box_code WHERE code=?`, boxCode).Scan(&productID, &wh)
	if warehouseID > 0 {
		wh = warehouseID
	}
	if productID == 0 {
		productID = 1
	}
	docNo := fmt.Sprintf("ST%d", time.Now().UnixNano())
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,date('now'),'draft',?,?)`,
		docNo, txnType, wh, "auto:"+boxCode)
	if err != nil {
		return err
	}
	tid, _ := res.LastInsertId()
	dir := "out"
	if txnType == "purchase_in" || txnType == "produce_in" {
		dir = "in"
	}
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction) VALUES(?,?,?,?,?,?)`,
		tid, 1, productID, qty, qty, dir)
	// post immediately
	sign := -1.0
	if dir == "in" {
		sign = 1
	}
	return s.adjustBalance(wh, productID, sign*qty)
}

func (s *Services) autoStockInNewBox(oldCode string, warehouseID, processID int64, qty float64, taskID, woID, stepID int64) (string, error) {
	var productID int64 = 2
	if warehouseID == 3 {
		productID = 3
	} else if warehouseID == 1 {
		productID = 1
	}
	newCode := fmt.Sprintf("BX%d", time.Now().UnixNano()%1e12)
	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, parent_box_id, current_process_id, current_step_id, task_id, work_order_id, status)
		VALUES(?,?,?,?,?,?, (SELECT id FROM inv_box_code WHERE code=?), ?,?,?,?,'open')`,
		newCode, productID, warehouseID, time.Now().Format("20060102"), qty, qty, oldCode, processID, stepID, taskID, woID)
	if err != nil {
		return "", err
	}
	if err := s.autoStock(newCode, warehouseID, processID, qty, "produce_in"); err != nil {
		return newCode, err
	}
	_, _ = s.DB.Exec(`UPDATE inv_box_code SET qty=?, weight=?, warehouse_id=?, product_id=?, updated_at=datetime('now') WHERE code=?`,
		qty, qty, warehouseID, productID, newCode)
	return newCode, nil
}

func (s *Services) writeFlowEvent(sourceType string, sourceID, fromStep, toStep int64, trigger, traceID, status, errMsg, payload string) int64 {
	res, err := s.DB.Exec(`INSERT INTO pd_flow_event(source_type, source_id, from_step_id, to_step_id, trigger_action, trace_id, status, error, payload_json)
		VALUES(?,?,?,?,?,?,?,?,?)`, sourceType, sourceID, nullIf0(fromStep), nullIf0(toStep), trigger, traceID, status, errMsg, payload)
	if err != nil {
		return 0
	}
	id, _ := res.LastInsertId()
	return id
}

func nullIf0(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func (s *Services) handleFlowEvents(c *gin.Context, method, action, path string) bool {
	if strings.Contains(path, "/retry") && method == "POST" {
		id := paramID(c)
		var status, payload, trigger string
		var sourceID, fromStep int64
		err := s.DB.QueryRow(`SELECT source_id, COALESCE(from_step_id,0), trigger_action, status, COALESCE(payload_json,'{}') FROM pd_flow_event WHERE id=?`, id).
			Scan(&sourceID, &fromStep, &trigger, &status, &payload)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		var p map[string]interface{}
		_ = json.Unmarshal([]byte(payload), &p)
		qty, _ := asFloat(p["qty"])
		box, _ := p["box_code"].(string)
		step := s.loadStep(fromStep)
		if step == nil {
			api.FailJSON(c, "STEP_MISSING")
			return true
		}
		out := s.AfterReportWork(c, sourceID, step.ProcessID, 0, qty, box, 0, 0, fromStep)
		_, _ = s.DB.Exec(`UPDATE pd_flow_event SET status='ok', error='' WHERE id=?`, id)
		api.OK(c, gin.H{"retried": id, "result": out})
		return true
	}
	return s.handleTableCRUD(c, "production/flow-events", action)
}

func (s *Services) handleFlowRules(c *gin.Context, method, action string) bool {
	if method == "GET" || action == "list" || action == "get" {
		rows, err := s.DB.Query(`SELECT id, routing_id, seq_no, process_id, COALESCE(step_code,''), COALESCE(step_name,''),
			COALESCE(is_piecework,0), COALESCE(is_inbound_checkpoint,0), COALESCE(auto_next,1),
			COALESCE(auto_stock_in,0), COALESCE(auto_stock_out,0), COALESCE(warehouse_id,0)
			FROM pd_routing_step WHERE routing_id=1 ORDER BY seq_no`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, rid, pid, wh int64
			var seq, piece, cp, an, asi, aso int
			var code, name string
			_ = rows.Scan(&id, &rid, &seq, &pid, &code, &name, &piece, &cp, &an, &asi, &aso, &wh)
			list = append(list, gin.H{
				"id": id, "routing_id": rid, "seq_no": seq, "process_id": pid, "step_code": code, "step_name": name,
				"is_piecework": piece == 1, "is_inbound_checkpoint": cp == 1, "auto_next": an == 1,
				"auto_stock_in": asi == 1, "auto_stock_out": aso == 1, "warehouse_id": wh,
			})
		}
		api.OK(c, gin.H{"routing_id": 1, "steps": list})
		return true
	}
	if method == "PUT" || action == "replace" {
		body := bindBody(c)
		if steps, ok := body["steps"].([]interface{}); ok {
			for _, ln := range steps {
				m, _ := ln.(map[string]interface{})
				if m == nil {
					continue
				}
				id, _ := asInt64(m["id"])
				if id == 0 {
					continue
				}
				_, _ = s.DB.Exec(`UPDATE pd_routing_step SET is_piecework=?, is_inbound_checkpoint=?, auto_next=?, auto_stock_in=?, auto_stock_out=?, warehouse_id=? WHERE id=?`,
					boolInt(m["is_piecework"]), boolInt(m["is_inbound_checkpoint"]), boolInt(m["auto_next"]),
					boolInt(m["auto_stock_in"]), boolInt(m["auto_stock_out"]), nullIf0(mustInt(m["warehouse_id"])), id)
			}
		}
		api.OK(c, gin.H{"ok": true})
		return true
	}
	return true
}

func boolInt(v interface{}) int {
	switch t := v.(type) {
	case bool:
		if t {
			return 1
		}
	case float64:
		if t != 0 {
			return 1
		}
	case string:
		if t == "1" || t == "true" {
			return 1
		}
	}
	return 0
}

func mustInt(v interface{}) int64 {
	n, _ := asInt64(v)
	return n
}
