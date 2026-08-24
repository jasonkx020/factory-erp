package biz

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type traceRoutingStep struct {
	SeqNo       int
	StepID      int64
	ProcessID   int64
	ProcessName string
	StepName    string
	StepCode    string
}

func (s *Services) resolveTraceProductID(trace string) int64 {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" {
		return 0
	}
	var productID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		  AND COALESCE(product_id,0)>0 LIMIT 1`, trace).Scan(&productID)
	if productID > 0 {
		return productID
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0) FROM pur_weigh_ticket
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND COALESCE(product_id,0)>0
		ORDER BY id DESC LIMIT 1`, trace).Scan(&productID)
	return productID
}

func (s *Services) resolveTraceRoutingSteps(trace string) (productID, routingID int64, productName string, steps []traceRoutingStep) {
	productID = s.resolveTraceProductID(trace)
	if productID <= 0 {
		return 0, 0, "", nil
	}
	_, pname, _ := s.productMeta(productID)
	productName = pname
	routingID = s.resolveRoutingID(0, productID)
	if routingID <= 0 {
		return productID, 0, productName, nil
	}
	rows, err := s.DB.Query(`SELECT rs.id, rs.seq_no, rs.process_id, COALESCE(rs.step_name,''), COALESCE(rs.step_code,''), COALESCE(p.name,'')
		FROM pd_routing_step rs LEFT JOIN pd_process p ON p.id=rs.process_id
		WHERE rs.routing_id=? ORDER BY rs.seq_no, rs.id`, routingID)
	if err != nil {
		return productID, routingID, productName, nil
	}
	defer rows.Close()
	for rows.Next() {
		var st traceRoutingStep
		if err := rows.Scan(&st.StepID, &st.SeqNo, &st.ProcessID, &st.StepName, &st.StepCode, &st.ProcessName); err != nil {
			continue
		}
		if st.StepName == "" {
			st.StepName = st.ProcessName
		}
		steps = append(steps, st)
	}
	return productID, routingID, productName, steps
}

func (s *Services) traceProcessCompleteTimes(trace string) map[int64]string {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	out := map[int64]string{}
	rows, err := s.DB.Query(`SELECT process_id, COALESCE(CAST(created_at AS TEXT),'')
		FROM pd_trace_process_log
		WHERE UPPER(trace_code)=UPPER(?) AND event_type='process_complete' AND process_id>0
		ORDER BY id`, trace)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var at string
		if err := rows.Scan(&pid, &at); err != nil {
			continue
		}
		out[pid] = at
	}
	return out
}

func (s *Services) isTraceProcessComplete(trace string, processID int64) bool {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" || processID <= 0 {
		return false
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_trace_process_log
		WHERE UPPER(trace_code)=UPPER(?) AND process_id=? AND event_type='process_complete'`, trace, processID).Scan(&n)
	return n > 0
}

func (s *Services) traceProcessYields(trace string) map[int64]gin.H {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	out := map[int64]gin.H{}
	rows, err := s.DB.Query(`SELECT process_id, input_kg, output_kg, loss_kg, loss_rate
		FROM pd_trace_process_yield WHERE UPPER(trace_code)=UPPER(?)`, trace)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var inKg, outKg, lossKg, rate float64
		if err := rows.Scan(&pid, &inKg, &outKg, &lossKg, &rate); err != nil {
			continue
		}
		out[pid] = gin.H{
			"input_kg": roundKg(inKg), "output_kg": roundKg(outKg),
			"loss_kg": roundKg(lossKg), "loss_rate": rate,
		}
	}
	return out
}

func (s *Services) buildTraceRoutingTimeline(trace string, wipMap map[int64]*traceProcessWipRow) (steps []gin.H, currentIdx int, canCompletePID int64) {
	_, _, _, routingSteps := s.resolveTraceRoutingSteps(trace)
	if len(routingSteps) == 0 {
		return nil, -1, 0
	}
	completeAt := s.traceProcessCompleteTimes(trace)
	yields := s.traceProcessYields(trace)
	currentIdx = -1
	allPriorDone := true
	for i, rs := range routingSteps {
		status := "pending"
		if s.isTraceProcessComplete(trace, rs.ProcessID) {
			status = "done"
		} else if allPriorDone {
			wip := wipMap[rs.ProcessID]
			wipKg, occKg := 0.0, 0.0
			if wip != nil {
				wipKg = wip.WipKg
				occKg = wip.OccupiedKg
			}
			if wipKg > kgEps || occKg > kgEps || s.traceProcessHasOpenIssue(trace, rs.ProcessID) {
				status = "in_progress"
			} else {
				status = "ready"
				if canCompletePID <= 0 {
					canCompletePID = rs.ProcessID
				}
			}
		}
		if status != "done" && currentIdx < 0 {
			currentIdx = i
		}
		if status != "done" {
			allPriorDone = false
		}
		item := gin.H{
			"seq_no": rs.SeqNo, "step_id": rs.StepID, "process_id": rs.ProcessID,
			"process_name": rs.ProcessName, "step_name": rs.StepName, "step_code": rs.StepCode,
			"step_status": status,
		}
		if wip := wipMap[rs.ProcessID]; wip != nil {
			item["available_kg"] = wip.AvailableKg
			item["occupied_kg"] = wip.OccupiedKg
			item["wip_kg"] = wip.WipKg
			item["pool_kg"] = wip.PoolKg
			item["issuable_kg"] = wip.IssuableKg
		}
		if y, ok := yields[rs.ProcessID]; ok {
			for k, v := range y {
				item[k] = v
			}
		}
		if at, ok := completeAt[rs.ProcessID]; ok {
			item["completed_at"] = at
		}
		steps = append(steps, item)
	}
	return steps, currentIdx, canCompletePID
}

func (s *Services) traceProcessHasOpenIssue(trace string, processID int64) bool {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" || processID <= 0 {
		return false
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='open'
		  AND (issue_kg - returned_kg - completed_kg) > 0.0005`, trace, processID).Scan(&n)
	if n > 0 {
		return true
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND biz_status='return_pending'`, trace, processID).Scan(&n)
	if n > 0 {
		return true
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_stock_in
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND process_id=? AND status='pending_warehouse'`, trace, processID).Scan(&n)
	return n > 0
}

func (s *Services) assertTraceProcessCanComplete(trace string, processID int64, wipMap map[int64]*traceProcessWipRow) string {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" {
		return "TRACE_CODE_REQUIRED"
	}
	if processID <= 0 {
		return "PROCESS_REQUIRED"
	}
	if s.traceProcessHasOpenIssue(trace, processID) {
		return "PROCESS_WIP_NOT_CLEAR"
	}
	wip := wipMap[processID]
	if wip != nil && (wip.WipKg > kgEps || wip.OccupiedKg > kgEps) {
		return "PROCESS_WIP_NOT_CLEAR"
	}
	return ""
}

func (s *Services) assertPriorRoutingStepsDone(trace string, targetProcessID int64) string {
	_, _, _, steps := s.resolveTraceRoutingSteps(trace)
	if len(steps) == 0 {
		return "ROUTING_REQUIRED"
	}
	found := false
	for _, st := range steps {
		if st.ProcessID == targetProcessID {
			found = true
			break
		}
		if !s.isTraceProcessComplete(trace, st.ProcessID) {
			return "PRIOR_STEP_NOT_DONE"
		}
	}
	if !found {
		return "PROCESS_NOT_IN_ROUTING"
	}
	return ""
}

func (s *Services) isLastRoutingProcess(trace string, processID int64) bool {
	_, _, _, steps := s.resolveTraceRoutingSteps(trace)
	if len(steps) == 0 {
		return false
	}
	last := steps[len(steps)-1]
	return last.ProcessID == processID
}

func (s *Services) allRoutingStepsComplete(trace string) bool {
	_, _, _, steps := s.resolveTraceRoutingSteps(trace)
	if len(steps) == 0 {
		return false
	}
	for _, st := range steps {
		if !s.isTraceProcessComplete(trace, st.ProcessID) {
			return false
		}
	}
	return true
}
