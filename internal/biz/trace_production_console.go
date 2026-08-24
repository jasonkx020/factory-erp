package biz

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

// listTraceProductionsConsole returns warehouse traces with UI status: in_stock|in_production|ended.
func (s *Services) listTraceProductionsConsole(c *gin.Context) bool {
	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))
	pageNum, pageSize := sqlutil.Page(c)
	if pageSize < 200 {
		// 生管台需要尽量一次拉全量可用溯源，避免点选前还要翻页/扫码。
		pageSize = 200
	}
	rows, err := s.DB.Query(`
		SELECT UPPER(TRIM(b.trace_code)) AS tc,
			COALESCE(SUM(COALESCE(b.weight, b.qty, 0)),0) AS stock_kg,
			COUNT(1) AS board_count
		FROM inv_box_code b
		WHERE COALESCE(b.is_deleted,0)=0 AND TRIM(COALESCE(b.trace_code,''))<>''
		  AND COALESCE(b.status,'') NOT IN ('destroyed','void')
		GROUP BY UPPER(TRIM(b.trace_code))
		ORDER BY tc
		LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	type row struct {
		tc, uiStatus, sessionStatus string
		stockKg                     float64
		boardCnt, sessionID         int64
	}
	raw := []row{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.tc, &r.stockKg, &r.boardCnt); err != nil {
			continue
		}
		r.uiStatus, r.sessionID, r.sessionStatus = s.traceUIStatus(r.tc)
		if statusFilter != "" {
			ok := statusFilter == r.uiStatus
			switch statusFilter {
			case "warehouse", "stock", "库中":
				ok = r.uiStatus == "in_stock"
			case "production", "producing", "生产中", "inproduction":
				ok = r.uiStatus == "in_production"
			case "done", "已结束":
				ok = r.uiStatus == "ended"
			}
			if !ok {
				continue
			}
		}
		raw = append(raw, r)
	}
	// 生产中优先，其次库中，已结束靠后
	rank := func(st string) int {
		switch st {
		case "in_production":
			return 0
		case "in_stock":
			return 1
		default:
			return 2
		}
	}
	sort.SliceStable(raw, func(i, j int) bool {
		ri, rj := rank(raw[i].uiStatus), rank(raw[j].uiStatus)
		if ri != rj {
			return ri < rj
		}
		return raw[i].tc < raw[j].tc
	})
	list := make([]gin.H, 0, len(raw))
	for _, r := range raw {
		fid, fname := s.traceFarmerInfo(r.tc)
		item := gin.H{
			"trace_code": r.tc, "ui_status": r.uiStatus, "status": r.uiStatus,
			"stock_kg": roundKg(r.stockKg), "board_count": r.boardCnt,
			"session_id": r.sessionID, "session_status": r.sessionStatus,
		}
		if fid > 0 {
			item["farmer_id"] = fid
		}
		if fname != "" {
			item["farmer_name"] = fname
		}
		list = append(list, item)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM (
		SELECT 1 FROM inv_box_code b
		WHERE COALESCE(b.is_deleted,0)=0 AND TRIM(COALESCE(b.trace_code,''))<>''
		  AND COALESCE(b.status,'') NOT IN ('destroyed','void')
		GROUP BY UPPER(TRIM(b.trace_code))) t`).Scan(&total)
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) traceUIStatus(trace string) (uiStatus string, sessionID int64, sessionStatus string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	_ = s.DB.QueryRow(`SELECT id, status FROM pd_trace_production
		WHERE UPPER(trace_code)=? AND status='in_progress' ORDER BY id DESC LIMIT 1`, trace).Scan(&sessionID, &sessionStatus)
	if sessionID > 0 {
		return "in_production", sessionID, sessionStatus
	}
	_ = s.DB.QueryRow(`SELECT id, status FROM pd_trace_production
		WHERE UPPER(trace_code)=? ORDER BY id DESC LIMIT 1`, trace).Scan(&sessionID, &sessionStatus)
	if sessionID > 0 && strings.EqualFold(sessionStatus, "done") {
		return "ended", sessionID, sessionStatus
	}
	return "in_stock", sessionID, sessionStatus
}

func (s *Services) traceFarmerInfo(trace string) (farmerID int64, farmerName string) {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if trace == "" {
		return 0, ""
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(l.farmer_id,0), COALESCE(f.name,'')
		FROM pur_trace_lot l LEFT JOIN pur_farmer f ON f.id=l.farmer_id
		WHERE UPPER(COALESCE(l.trace_code,''))=? ORDER BY l.id DESC LIMIT 1`, trace).Scan(&farmerID, &farmerName)
	if farmerID > 0 {
		return farmerID, farmerName
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(b.farmer_id,0), COALESCE(f.name,'')
		FROM inv_box_code b LEFT JOIN pur_farmer f ON f.id=b.farmer_id
		WHERE UPPER(COALESCE(b.trace_code,''))=? AND COALESCE(b.farmer_id,0)>0
		LIMIT 1`, trace).Scan(&farmerID, &farmerName)
	return farmerID, farmerName
}

func (s *Services) getTraceProductionWip(c *gin.Context) bool {
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
	productID, routingID, productName, _ := s.resolveTraceRoutingSteps(trace)
	routingSteps, currentIdx, canCompletePID := s.buildTraceRoutingTimeline(trace, wipMap)
	boardCountByProc := map[int64]int{}
	brows, err := s.DB.Query(`SELECT id, code, COALESCE(current_process_id,0), COALESCE(weight, qty, 0), COALESCE(status,'')
		FROM inv_box_code WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		  AND COALESCE(status,'') NOT IN ('destroyed','void')`, trace)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	boards := []gin.H{}
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
		if pid > 0 && w > kgEps {
			boardCountByProc[pid]++
		}
	}
	brows.Close()
	rCode, rName, _ := s.loadRoutingMeta(routingID)
	steps := []gin.H{}
	totalAvail, totalOcc, totalWip := 0.0, 0.0, 0.0
	for _, r := range wipMap {
		totalAvail = roundKg(totalAvail + r.AvailableKg)
		totalOcc = roundKg(totalOcc + r.OccupiedKg)
		totalWip = roundKg(totalWip + r.WipKg)
		steps = append(steps, gin.H{
			"process_id": r.ProcessID, "process_name": r.ProcessName,
			"available_kg": r.AvailableKg, "pool_kg": r.PoolKg, "occupied_kg": r.OccupiedKg,
			"wip_kg": r.WipKg, "issuable_kg": r.IssuableKg,
			"can_issue_from": r.CanIssueFrom, "can_stock_in": r.CanStockIn,
			"stock_kg": r.AvailableKg, "board_count": boardCountByProc[r.ProcessID],
		})
	}
	api.OK(c, gin.H{
		"trace_code": trace, "ui_status": uiStatus, "status": uiStatus,
		"session_id": sessionID, "session_status": sessionStatus,
		"product_id": productID, "routing_id": routingID, "product_name": productName,
		"routing_code": rCode, "routing_name": rName,
		"routing_steps": routingSteps, "current_step_index": currentIdx, "can_complete_process_id": canCompletePID,
		"total_available_kg": totalAvail, "total_occupied_kg": totalOcc, "total_wip_kg": totalWip,
		"steps": steps, "boards": boards,
	})
	return true
}

func (s *Services) assertTraceCanComplete(trace string) string {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	var openN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND status='open'
		  AND (issue_kg - returned_kg - completed_kg) > 0.0005`, trace).Scan(&openN)
	if openN > 0 {
		return "OPEN_ISSUES_REMAIN"
	}
	var pendRet int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND biz_status='return_pending'`, trace).Scan(&pendRet)
	if pendRet > 0 {
		return "RETURN_PENDING"
	}
	var pendIn int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_stock_in
		WHERE UPPER(COALESCE(trace_code,''))=UPPER(?) AND status='pending_warehouse'`, trace).Scan(&pendIn)
	if pendIn > 0 {
		return "STOCK_IN_PENDING"
	}
	return ""
}

func (s *Services) issueReturnableKg(issueKg, returnedKg, completedKg, pendingReturnKg float64) float64 {
	return issueRemain(issueKg, returnedKg+pendingReturnKg, completedKg)
}
