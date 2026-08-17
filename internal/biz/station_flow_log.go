package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

// processPayMode returns weight|piece|none from pd_process (falls back to is_piecework→weight).
func (s *Services) processPayMode(processID int64) string {
	if processID <= 0 {
		return "none"
	}
	var mode string
	var piece int
	err := s.DB.QueryRow(`SELECT COALESCE(NULLIF(pay_mode,''),'none'), COALESCE(is_piecework,0) FROM pd_process WHERE id=?`, processID).
		Scan(&mode, &piece)
	if err != nil {
		return "none"
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "weight" || mode == "piece" || mode == "none" {
		return mode
	}
	if piece == 1 {
		return "weight"
	}
	return "none"
}

func (s *Services) processPaysYield(processID int64) bool {
	m := s.processPayMode(processID)
	return m == "weight" || m == "piece"
}

func (s *Services) workerEmpType(workerID int64) string {
	if workerID <= 0 {
		return ""
	}
	var t string
	_ = s.DB.QueryRow(`SELECT COALESCE(emp_type,'') FROM hr_employee WHERE id=?`, workerID).Scan(&t)
	return strings.ToLower(strings.TrimSpace(t))
}

func (s *Services) workerYieldEligible(workerID int64) bool {
	return s.workerEmpType(workerID) == "piece"
}

func (s *Services) shouldLockYieldWage(processID, workerID int64) bool {
	return s.processPaysYield(processID) && s.workerYieldEligible(workerID)
}

func normalizePayMode(raw string, isPiecework bool) string {
	m := strings.ToLower(strings.TrimSpace(raw))
	switch m {
	case "weight", "piece", "none":
		return m
	}
	if isPiecework {
		return "weight"
	}
	return "none"
}

func payModeToIsPiecework(mode string) int {
	if mode == "weight" || mode == "piece" {
		return 1
	}
	return 0
}

type stationFlowEvent struct {
	EventType            string
	BizDate              string
	BoardID              int64
	BoardCode, TraceCode string
	ProcessID, StepID    int64
	ProcessName          string
	WorkerID             int64
	WorkerName, Badge    string
	ActorUserID          int64
	OperatorEmployeeID   int64
	Kg                   float64
	PayMode, EmpType     string
	Rate, Amount         float64
	RefType              string
	RefID                int64
	Before, After        interface{}
	Remark               string
	Payload              interface{}
}

func (s *Services) appendStationFlowLog(ev stationFlowEvent) {
	if ev.EventType == "" {
		return
	}
	bizDate := strings.TrimSpace(ev.BizDate)
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	bb, _ := json.Marshal(ev.Before)
	aa, _ := json.Marshal(ev.After)
	pp, _ := json.Marshal(ev.Payload)
	if string(bb) == "null" {
		bb = []byte("{}")
	}
	if string(aa) == "null" {
		aa = []byte("{}")
	}
	if string(pp) == "null" {
		pp = []byte("{}")
	}
	if ev.ProcessName == "" && ev.ProcessID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, ev.ProcessID).Scan(&ev.ProcessName)
	}
	if ev.PayMode == "" && ev.ProcessID > 0 {
		ev.PayMode = s.processPayMode(ev.ProcessID)
	}
	if ev.EmpType == "" && ev.WorkerID > 0 {
		ev.EmpType = s.workerEmpType(ev.WorkerID)
	}
	if (ev.WorkerName == "" || ev.Badge == "") && ev.WorkerID > 0 {
		var n, b string
		_ = s.DB.QueryRow(`SELECT COALESCE(name,''), COALESCE(badge_code,'') FROM hr_employee WHERE id=?`, ev.WorkerID).Scan(&n, &b)
		if ev.WorkerName == "" {
			ev.WorkerName = n
		}
		if ev.Badge == "" {
			ev.Badge = b
		}
	}
	payMode := strings.TrimSpace(ev.PayMode)
	if payMode == "" {
		payMode = "none"
	}
	_, _ = s.DB.Exec(`INSERT INTO pd_station_flow_log(
		event_type, biz_date, board_id, board_code, trace_code, process_id, step_id, process_name,
		worker_id, worker_name, badge_code, actor_user_id, operator_employee_id,
		kg, pay_mode, emp_type, rate, amount, ref_type, ref_id, before_json, after_json, remark, payload_json)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.EventType, bizDate, ev.BoardID, ev.BoardCode, ev.TraceCode, ev.ProcessID, ev.StepID, ev.ProcessName,
		ev.WorkerID, ev.WorkerName, ev.Badge, ev.ActorUserID, ev.OperatorEmployeeID,
		ev.Kg, payMode, ev.EmpType, ev.Rate, ev.Amount, ev.RefType, ev.RefID,
		string(bb), string(aa), ev.Remark, string(pp))
}

func (s *Services) handleStationFlowLogs(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	bizDate := strings.TrimSpace(c.Query("biz_date"))
	board := strings.TrimSpace(c.Query("board_code"))
	if board == "" {
		board = strings.TrimSpace(c.Query("box_code"))
	}
	eventType := strings.TrimSpace(c.Query("event_type"))
	var processID, workerID int64
	fmt.Sscanf(c.Query("process_id"), "%d", &processID)
	fmt.Sscanf(c.Query("worker_id"), "%d", &workerID)
	onlyAmount := c.Query("has_amount") == "1" || strings.EqualFold(c.Query("has_amount"), "true")

	where := `WHERE 1=1`
	args := []interface{}{}
	if bizDate != "" {
		where += ` AND biz_date=?`
		args = append(args, bizDate)
	}
	if board != "" {
		where += ` AND board_code=?`
		args = append(args, board)
	}
	if eventType != "" {
		where += ` AND event_type=?`
		args = append(args, eventType)
	}
	if processID > 0 {
		where += ` AND process_id=?`
		args = append(args, processID)
	}
	if workerID > 0 {
		where += ` AND worker_id=?`
		args = append(args, workerID)
	}
	if onlyAmount {
		where += ` AND ABS(amount) > 0.0005`
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_station_flow_log `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, event_type, biz_date, board_id, board_code, trace_code, process_id, step_id, process_name,
		worker_id, worker_name, badge_code, actor_user_id, kg, pay_mode, emp_type, rate, amount, ref_type, ref_id,
		COALESCE(remark,''), COALESCE(payload_json,''), created_at
		FROM pd_station_flow_log `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, boardID, procID, stepID, wid, actorID, refID int64
		var et, bd, boardCode, trace, processName, workerName, badge, payMode, empType, refType, remark, payload, created string
		var kg, rate, amount float64
		_ = rows.Scan(&id, &et, &bd, &boardID, &boardCode, &trace, &procID, &stepID, &processName,
			&wid, &workerName, &badge, &actorID, &kg, &payMode, &empType, &rate, &amount, &refType, &refID,
			&remark, &payload, &created)
		list = append(list, gin.H{
			"id": id, "event_type": et, "biz_date": bd, "board_id": boardID, "board_code": boardCode,
			"trace_code": trace, "process_id": procID, "step_id": stepID, "process_name": processName,
			"worker_id": wid, "worker_name": workerName, "badge_code": badge, "actor_user_id": actorID,
			"kg": kg, "pay_mode": payMode, "emp_type": empType, "rate": rate, "amount": amount,
			"ref_type": refType, "ref_id": refID, "remark": remark, "payload_json": payload, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}
