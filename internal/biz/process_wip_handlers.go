package biz

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func (s *Services) handleProcessWip(c *gin.Context, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/boxes") || strings.HasSuffix(openapiPath, "/boxes") {
		return s.listProcessWipBoxes(c)
	}
	if action == "list" || action == "get" {
		return s.getProcessWip(c)
	}
	api.FailJSON(c, "METHOD_NOT_ALLOWED")
	return true
}

func (s *Services) resolveWipRoutingID(c *gin.Context) (int64, string, string) {
	productID := int64(0)
	if v := strings.TrimSpace(c.Query("product_id")); v != "" {
		fmt.Sscanf(v, "%d", &productID)
	}
	routingID := int64(0)
	if v := strings.TrimSpace(c.Query("routing_id")); v != "" {
		fmt.Sscanf(v, "%d", &routingID)
	}
	code, name := "", ""
	if productID > 0 {
		_ = s.DB.QueryRow(`SELECT id, COALESCE(code,''), COALESCE(name,'') FROM pd_routing
			WHERE product_id=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, productID).
			Scan(&routingID, &code, &name)
		if routingID <= 0 {
			return 0, "", "ROUTING_REQUIRED"
		}
		return routingID, code, ""
	}
	if routingID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(code,''), COALESCE(name,'') FROM pd_routing WHERE id=?`, routingID).Scan(&code, &name)
		return routingID, code, ""
	}
	_ = s.DB.QueryRow(`SELECT id, COALESCE(code,''), COALESCE(name,'') FROM pd_routing
		WHERE code='RT-CASSAVA-RAW' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`).Scan(&routingID, &code, &name)
	if routingID <= 0 {
		_ = s.DB.QueryRow(`SELECT id, COALESCE(code,''), COALESCE(name,'') FROM pd_routing
			WHERE status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`).Scan(&routingID, &code, &name)
	}
	if routingID <= 0 {
		return 0, "", "ROUTING_REQUIRED"
	}
	return routingID, code, ""
}

func (s *Services) getProcessWip(c *gin.Context) bool {
	routingID, routingCode, errMsg := s.resolveWipRoutingID(c)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	productFilter := int64(0)
	if v := strings.TrimSpace(c.Query("product_id")); v != "" {
		fmt.Sscanf(v, "%d", &productFilter)
	}

	type stepAgg struct {
		stepID, processID, warehouseID int64
		seqNo                          int
		stepCode, stepName, processName string
		boxCount                       int
		wipWeight, wipQty              float64
	}
	steps := []stepAgg{}
	rows, err := s.DB.Query(`SELECT rs.id, rs.seq_no, COALESCE(rs.step_code,''), COALESCE(rs.step_name,''),
		COALESCE(rs.process_id,0), COALESCE(p.name,''), COALESCE(rs.warehouse_id,0)
		FROM pd_routing_step rs
		LEFT JOIN pd_process p ON p.id=rs.process_id
		WHERE rs.routing_id=? ORDER BY rs.seq_no`, routingID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	stepIndex := map[int64]int{}
	for rows.Next() {
		var st stepAgg
		if err := rows.Scan(&st.stepID, &st.seqNo, &st.stepCode, &st.stepName, &st.processID, &st.processName, &st.warehouseID); err != nil {
			continue
		}
		stepIndex[st.stepID] = len(steps)
		steps = append(steps, st)
	}
	rows.Close()

	q := `SELECT COALESCE(b.current_step_id,0), COUNT(1),
		COALESCE(SUM(COALESCE(b.weight, b.qty, 0)),0), COALESCE(SUM(COALESCE(b.qty,0)),0)
		FROM inv_box_code b
		WHERE COALESCE(b.is_deleted,0)=0 AND b.status IN ('open','active')
		  AND NOT EXISTS (
			SELECT 1 FROM inv_box_code c
			WHERE c.parent_box_id=b.id AND COALESCE(c.is_deleted,0)=0 AND c.status IN ('open','active')
		  )`
	args := []interface{}{}
	if productFilter > 0 {
		q += ` AND b.product_id=?`
		args = append(args, productFilter)
	}
	q += ` GROUP BY COALESCE(b.current_step_id,0)`
	arows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	unassignedBoxes, unassignedWeight := 0, 0.0
	totalBoxes := 0
	totalWeight := 0.0
	for arows.Next() {
		var stepID int64
		var cnt int
		var w, qty float64
		if err := arows.Scan(&stepID, &cnt, &w, &qty); err != nil {
			continue
		}
		if idx, ok := stepIndex[stepID]; ok {
			steps[idx].boxCount = cnt
			steps[idx].wipWeight = w
			steps[idx].wipQty = qty
			totalBoxes += cnt
			totalWeight += w
		} else if stepID == 0 {
			unassignedBoxes = cnt
			unassignedWeight = w
			totalBoxes += cnt
			totalWeight += w
		}
	}
	arows.Close()

	var pendingCnt int
	var pendingWeight float64
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(COALESCE(r.weight, r.qty, 0)),0)
		FROM pd_report_work r
		WHERE r.status='confirm_pending'`).Scan(&pendingCnt, &pendingWeight)

	list := make([]gin.H, 0, len(steps))
	for _, st := range steps {
		list = append(list, gin.H{
			"step_id": st.stepID, "seq_no": st.seqNo, "step_code": st.stepCode, "step_name": st.stepName,
			"process_id": st.processID, "process_name": st.processName, "warehouse_id": st.warehouseID,
			"box_count": st.boxCount, "wip_weight": st.wipWeight, "wip_qty": st.wipQty,
		})
	}
	api.OK(c, gin.H{
		"routing_id": routingID, "routing_code": routingCode,
		"product_id": productFilter,
		"total_boxes": totalBoxes, "total_weight": totalWeight,
		"pending_confirm_reports": pendingCnt, "pending_confirm_weight": pendingWeight,
		"unassigned": gin.H{"box_count": unassignedBoxes, "wip_weight": unassignedWeight},
		"steps":      list,
	})
	return true
}

func (s *Services) listProcessWipBoxes(c *gin.Context) bool {
	stepID := int64(0)
	unassigned := false
	if strings.TrimSpace(c.Query("unassigned")) == "1" || strings.EqualFold(c.Query("unassigned"), "true") {
		unassigned = true
	} else if v := strings.TrimSpace(c.Query("step_id")); v != "" {
		fmt.Sscanf(v, "%d", &stepID)
	}
	productFilter := int64(0)
	if v := strings.TrimSpace(c.Query("product_id")); v != "" {
		fmt.Sscanf(v, "%d", &productFilter)
	}

	q := `SELECT b.id, b.code, COALESCE(b.product_id,0), COALESCE(p.name,''), COALESCE(b.warehouse_id,0),
		COALESCE(b.qty,0), COALESCE(b.weight,0), COALESCE(b.status,''), COALESCE(b.trace_code,''),
		COALESCE(b.receive_date,''), COALESCE(b.farmer_id,0), COALESCE(b.current_process_id,0), COALESCE(b.current_step_id,0)
		FROM inv_box_code b
		LEFT JOIN prd_product p ON p.id=b.product_id
		WHERE COALESCE(b.is_deleted,0)=0 AND b.status IN ('open','active')
		  AND NOT EXISTS (
			SELECT 1 FROM inv_box_code c
			WHERE c.parent_box_id=b.id AND COALESCE(c.is_deleted,0)=0 AND c.status IN ('open','active')
		  )`
	args := []interface{}{}
	if unassigned {
		q += ` AND COALESCE(b.current_step_id,0)=0`
	} else if stepID > 0 {
		q += ` AND b.current_step_id=?`
		args = append(args, stepID)
	} else {
		api.FailJSON(c, "STEP_ID_REQUIRED")
		return true
	}
	if productFilter > 0 {
		q += ` AND b.product_id=?`
		args = append(args, productFilter)
	}
	q += ` ORDER BY b.id DESC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, productID, wh, farmerID, procID, curStep int64
		var code, productName, status, trace, recvDate string
		var qty, weight float64
		if err := rows.Scan(&id, &code, &productID, &productName, &wh, &qty, &weight, &status, &trace, &recvDate, &farmerID, &procID, &curStep); err != nil {
			continue
		}
		list = append(list, gin.H{
			"id": id, "code": code, "product_id": productID, "product_name": productName,
			"warehouse_id": wh, "qty": qty, "weight": weight, "status": status,
			"trace_code": trace, "receive_date": recvDate, "farmer_id": farmerID,
			"current_process_id": procID, "current_step_id": curStep,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}
