package biz

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type traceRoutingStep struct {
	SeqNo             int
	StepID            int64
	ProcessID         int64
	ProcessName       string
	StepName          string
	StepCode          string
	InputProductID    int64
	OutputProductID   int64
	InputProductName  string
	OutputProductName string
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

// resolveTraceSessionRoutingID returns routing locked on trace production session.
func (s *Services) resolveTraceSessionRoutingID(trace string) int64 {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" {
		return 0
	}
	var rid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_trace_production
		WHERE UPPER(trace_code)=? AND status='in_progress' ORDER BY id DESC LIMIT 1`, trace).Scan(&rid)
	if rid > 0 {
		return rid
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(routing_id,0) FROM pd_trace_production
		WHERE UPPER(trace_code)=? AND COALESCE(routing_id,0)>0 ORDER BY id DESC LIMIT 1`, trace).Scan(&rid)
	return rid
}

func (s *Services) loadRoutingMeta(routingID int64) (code, name string, productID int64) {
	if routingID <= 0 {
		return "", "", 0
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(code,''), COALESCE(name,''), COALESCE(product_id,0)
		FROM pd_routing WHERE id=? AND COALESCE(is_deleted,0)=0`, routingID).Scan(&code, &name, &productID)
	return code, name, productID
}

func (s *Services) loadRoutingStepsByID(routingID int64) []traceRoutingStep {
	if routingID <= 0 {
		return nil
	}
	rows, err := s.DB.Query(`SELECT rs.id, rs.seq_no, rs.process_id, COALESCE(rs.step_name,''), COALESCE(rs.step_code,''), COALESCE(p.name,''),
		COALESCE(rs.output_product_id,0)
		FROM pd_routing_step rs LEFT JOIN pd_process p ON p.id=rs.process_id
		WHERE rs.routing_id=? ORDER BY rs.seq_no, rs.id`, routingID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	steps := []traceRoutingStep{}
	for rows.Next() {
		var st traceRoutingStep
		if err := rows.Scan(&st.StepID, &st.SeqNo, &st.ProcessID, &st.StepName, &st.StepCode, &st.ProcessName, &st.OutputProductID); err != nil {
			continue
		}
		if st.StepName == "" {
			st.StepName = st.ProcessName
		}
		steps = append(steps, st)
	}
	for i := range steps {
		if steps[i].OutputProductID > 0 {
			_, steps[i].OutputProductName, _ = s.productMeta(steps[i].OutputProductID)
		}
		if i > 0 && steps[i-1].OutputProductID > 0 {
			steps[i].InputProductID = steps[i-1].OutputProductID
			steps[i].InputProductName = steps[i-1].OutputProductName
		}
	}
	return steps
}

func (s *Services) resolveTraceRoutingSteps(trace string) (productID, routingID int64, productName string, steps []traceRoutingStep) {
	productID = s.resolveTraceProductID(trace)
	if productID > 0 {
		_, pname, _ := s.productMeta(productID)
		productName = pname
	}
	routingID = s.resolveTraceSessionRoutingID(trace)
	if routingID <= 0 && productID > 0 {
		// Legacy sessions without locked routing: fallback to product default.
		routingID = s.resolveRoutingID(0, productID)
	}
	if routingID <= 0 {
		return productID, 0, productName, nil
	}
	steps = s.loadRoutingStepsByID(routingID)
	return productID, routingID, productName, steps
}

func (s *Services) listRoutingsForProduct(productID int64) []gin.H {
	if productID <= 0 {
		return nil
	}
	rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(version_no,''), COALESCE(status,'')
		FROM pd_routing WHERE COALESCE(is_deleted,0)=0 AND status='active'
		  AND (product_id=? OR EXISTS (
		    SELECT 1 FROM pd_routing_step rs WHERE rs.routing_id=pd_routing.id AND rs.output_product_id=?
		  ))
		ORDER BY id DESC`, productID, productID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, ver, status string
		if err := rows.Scan(&id, &code, &name, &ver, &status); err != nil {
			continue
		}
		steps := s.loadRoutingStepsByID(id)
		preview := make([]string, 0, len(steps))
		for _, st := range steps {
			label := st.StepName
			if label == "" {
				label = st.ProcessName
			}
			if label != "" {
				preview = append(preview, label)
			}
		}
		out = append(out, gin.H{
			"id": id, "code": code, "name": name, "version_no": ver, "status": status,
			"step_count": len(steps), "steps_preview": preview,
		})
	}
	return out
}

func (s *Services) validateTraceRoutingStart(trace string, routingID int64) (productID int64, errCode string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" {
		return 0, "TRACE_CODE_REQUIRED"
	}
	productID = s.resolveTraceProductID(trace)
	if productID <= 0 {
		return 0, "PRODUCT_REQUIRED"
	}
	if routingID <= 0 {
		return productID, "ROUTING_REQUIRED"
	}
	var rid int64
	var rProductID int64
	var status string
	var stepCnt int
	err := s.DB.QueryRow(`SELECT id, COALESCE(product_id,0), COALESCE(status,'')
		FROM pd_routing WHERE id=? AND COALESCE(is_deleted,0)=0`, routingID).Scan(&rid, &rProductID, &status)
	if err != nil || rid <= 0 {
		return productID, "ROUTING_NOT_FOUND"
	}
	if !strings.EqualFold(status, "active") {
		return productID, "ROUTING_INACTIVE"
	}
	if rProductID != productID && !s.routingMatchesTraceProduct(routingID, productID) {
		return productID, "ROUTING_PRODUCT_MISMATCH"
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id=?`, routingID).Scan(&stepCnt)
	if stepCnt <= 0 {
		return productID, "ROUTING_STEPS_REQUIRED"
	}
	return productID, ""
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
	_, routingID, _, routingSteps := s.resolveTraceRoutingSteps(trace)
	if len(routingSteps) == 0 && routingID > 0 {
		routingSteps = s.loadRoutingStepsByID(routingID)
	}
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
		if rs.InputProductID > 0 {
			item["input_product_id"] = rs.InputProductID
			item["input_product_name"] = rs.InputProductName
		}
		if rs.OutputProductID > 0 {
			item["output_product_id"] = rs.OutputProductID
			item["output_product_name"] = rs.OutputProductName
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
		switch status {
		case "done":
			item["action"] = "done"
			item["action_hint"] = "已完成"
		case "ready":
			item["action"] = "complete"
			item["action_hint"] = "可结束本工序"
		case "in_progress":
			item["action"] = "in_progress"
			item["action_hint"] = "在制中，待领退料清零"
		default:
			item["action"] = "pending"
			item["action_hint"] = "等待上道工序"
		}
		steps = append(steps, item)
	}
	if len(steps) == 0 && len(wipMap) > 0 {
		pids := make([]int64, 0, len(wipMap))
		for pid := range wipMap {
			if pid > 0 {
				pids = append(pids, pid)
			}
		}
		sort.Slice(pids, func(i, j int) bool { return pids[i] < pids[j] })
		for i, pid := range pids {
			wip := wipMap[pid]
			if wip == nil || wip.WipKg <= kgEps {
				continue
			}
			status := "in_progress"
			action := "in_progress"
			hint := "在制中，待领退料清零"
			if canCompletePID <= 0 && i == 0 {
				status = "ready"
				action = "complete"
				hint = "可结束本工序"
				canCompletePID = pid
			}
			if currentIdx < 0 {
				currentIdx = i
			}
			item := gin.H{
				"seq_no": i + 1, "process_id": pid, "process_name": wip.ProcessName,
				"step_status": status, "action": action, "action_hint": hint,
				"available_kg": wip.AvailableKg, "occupied_kg": wip.OccupiedKg,
				"wip_kg": wip.WipKg, "pool_kg": wip.PoolKg, "issuable_kg": wip.IssuableKg,
			}
			if y, ok := yields[pid]; ok {
				for k, v := range y {
					item[k] = v
				}
			}
			steps = append(steps, item)
		}
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
