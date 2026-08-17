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
		stepID, processID, warehouseID             int64
		seqNo                                      int
		stepCode, stepName, processName            string
		boxCount                                   int
		availableKg, occupiedKg, wipWeight, wipQty float64
		stockKg                                    float64
		stockBoxCount                              int
		boards                                     map[int64]struct{}
		stockBoards                                map[int64]struct{}
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
	processToStep := map[int64]int{}
	for rows.Next() {
		var st stepAgg
		if err := rows.Scan(&st.stepID, &st.seqNo, &st.stepCode, &st.stepName, &st.processID, &st.processName, &st.warehouseID); err != nil {
			continue
		}
		st.boards = map[int64]struct{}{}
		st.stockBoards = map[int64]struct{}{}
		stepIndex[st.stepID] = len(steps)
		if st.processID > 0 {
			if _, ok := processToStep[st.processID]; !ok {
				processToStep[st.processID] = len(steps)
			}
		}
		steps = append(steps, st)
	}
	rows.Close()

	mark := func(idx int, boardID int64) {
		if idx < 0 || idx >= len(steps) {
			return
		}
		if boardID > 0 {
			steps[idx].boards[boardID] = struct{}{}
		}
	}

	q := `SELECT b.id, COALESCE(b.current_step_id,0), COALESCE(b.current_process_id,0), COALESCE(b.weight, b.qty, 0)
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
	brows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	unassignedBoxes, unassignedWeight := 0, 0.0
	unassignedIDs := map[int64]struct{}{}
	for brows.Next() {
		var boardID, stepID, procID int64
		var w float64
		if err := brows.Scan(&boardID, &stepID, &procID, &w); err != nil {
			continue
		}
		idx, ok := stepIndex[stepID]
		if !ok && procID > 0 {
			idx, ok = processToStep[procID]
		}
		if ok {
			if w > kgEps {
				steps[idx].availableKg = roundKg(steps[idx].availableKg + w)
			}
			mark(idx, boardID)
		} else if stepID == 0 && procID == 0 {
			unassignedIDs[boardID] = struct{}{}
			unassignedWeight += w
		}
	}
	brows.Close()
	unassignedBoxes = len(unassignedIDs)

	iq := `SELECT i.board_id, COALESCE(i.step_id,0), COALESCE(i.process_id,0), COALESCE(i.worker_id,0),
		COALESCE(SUM(i.issue_kg - i.returned_kg - i.completed_kg),0)
		FROM pd_process_issue i
		JOIN inv_box_code b ON b.id=i.board_id
		WHERE i.status='open' AND COALESCE(b.is_deleted,0)=0 AND b.status IN ('open','active','finished')`
	iargs := []interface{}{}
	if productFilter > 0 {
		iq += ` AND b.product_id=?`
		iargs = append(iargs, productFilter)
	}
	iq += ` GROUP BY i.board_id, COALESCE(i.step_id,0), COALESCE(i.process_id,0), COALESCE(i.worker_id,0)`
	irows, err := s.DB.Query(iq, iargs...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	for irows.Next() {
		var boardID, stepID, procID, workerID int64
		var kg float64
		if err := irows.Scan(&boardID, &stepID, &procID, &workerID, &kg); err != nil {
			continue
		}
		if kg <= kgEps {
			continue
		}
		idx, ok := stepIndex[stepID]
		if !ok && procID > 0 {
			idx, ok = processToStep[procID]
		}
		if !ok {
			continue
		}
		if workerID <= 0 {
			steps[idx].availableKg = roundKg(steps[idx].availableKg + kg)
		} else {
			steps[idx].occupiedKg = roundKg(steps[idx].occupiedKg + kg)
		}
		mark(idx, boardID)
	}
	irows.Close()

	// In-warehouse buffer boards created by stock_in (have parent_box_id), keyed by completed process.
	sq := `SELECT b.id, COALESCE(b.current_step_id,0), COALESCE(b.current_process_id,0), COALESCE(b.weight, b.qty, 0)
		FROM inv_box_code b
		WHERE COALESCE(b.is_deleted,0)=0 AND b.status IN ('open','active')
		  AND COALESCE(b.parent_box_id,0)>0`
	sargs := []interface{}{}
	if productFilter > 0 {
		sq += ` AND b.product_id=?`
		sargs = append(sargs, productFilter)
	}
	srows, err := s.DB.Query(sq, sargs...)
	if err == nil {
		for srows.Next() {
			var boardID, stepID, procID int64
			var w float64
			if err := srows.Scan(&boardID, &stepID, &procID, &w); err != nil {
				continue
			}
			if w <= kgEps {
				continue
			}
			idx, ok := stepIndex[stepID]
			if !ok && procID > 0 {
				idx, ok = processToStep[procID]
			}
			if !ok {
				continue
			}
			steps[idx].stockKg = roundKg(steps[idx].stockKg + w)
			steps[idx].stockBoards[boardID] = struct{}{}
		}
		srows.Close()
	}

	var pendingCnt int
	var pendingWeight float64
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(COALESCE(r.weight, r.qty, 0)),0)
		FROM pd_report_work r
		WHERE r.status='confirm_pending'`).Scan(&pendingCnt, &pendingWeight)

	totalBoxes := 0
	totalWeight := 0.0
	totalStock := 0.0
	list := make([]gin.H, 0, len(steps))
	for i := range steps {
		st := &steps[i]
		st.boxCount = len(st.boards)
		st.stockBoxCount = len(st.stockBoards)
		st.wipWeight = roundKg(st.availableKg + st.occupiedKg)
		st.wipQty = st.wipWeight
		totalBoxes += st.boxCount
		totalWeight += st.wipWeight
		totalStock += st.stockKg
		list = append(list, gin.H{
			"step_id": st.stepID, "seq_no": st.seqNo, "step_code": st.stepCode, "step_name": st.stepName,
			"process_id": st.processID, "process_name": st.processName, "warehouse_id": st.warehouseID,
			"box_count": st.boxCount, "board_count": st.boxCount,
			"available_kg": st.availableKg, "occupied_kg": st.occupiedKg,
			"wip_weight": st.wipWeight, "wip_qty": st.wipQty,
			"stock_kg": st.stockKg, "stock_box_count": st.stockBoxCount,
		})
	}
	totalWeight = roundKg(totalWeight)
	totalStock = roundKg(totalStock)
	api.OK(c, gin.H{
		"routing_id": routingID, "routing_code": routingCode,
		"product_id":  productFilter,
		"total_boxes": totalBoxes, "total_boards": totalBoxes, "total_weight": totalWeight,
		"total_stock_kg": totalStock,
		"pending_confirm_reports": pendingCnt, "pending_confirm_weight": pendingWeight,
		"unassigned": gin.H{"box_count": unassignedBoxes, "board_count": unassignedBoxes, "wip_weight": roundKg(unassignedWeight)},
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

	var processID int64
	if stepID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(process_id,0) FROM pd_routing_step WHERE id=?`, stepID).Scan(&processID)
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
		q += ` AND COALESCE(b.current_step_id,0)=0 AND COALESCE(b.current_process_id,0)=0`
	} else if stepID > 0 {
		q += ` AND (
			b.current_step_id=?
			OR EXISTS (
				SELECT 1 FROM pd_process_issue i
				WHERE i.board_id=b.id AND i.status='open'
				  AND (i.step_id=? OR (COALESCE(i.step_id,0)=0 AND i.process_id=?))
				  AND (i.issue_kg - i.returned_kg - i.completed_kg) > 0
			)
		)`
		args = append(args, stepID, stepID, processID)
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
	ids := []int64{}
	for rows.Next() {
		var id, productID, wh, farmerID, procID, curStep int64
		var code, productName, status, trace, recvDate string
		var qty, weight float64
		if err := rows.Scan(&id, &code, &productID, &productName, &wh, &qty, &weight, &status, &trace, &recvDate, &farmerID, &procID, &curStep); err != nil {
			continue
		}
		avail := 0.0
		if unassigned || curStep == stepID {
			avail = weight
		}
		ids = append(ids, id)
		wip := avail
		list = append(list, gin.H{
			"id": id, "code": code, "product_id": productID, "product_name": productName,
			"warehouse_id": wh, "qty": qty, "weight": weight, "available_kg": avail, "status": status,
			"trace_code": trace, "receive_date": recvDate, "farmer_id": farmerID,
			"current_process_id": procID, "current_step_id": curStep,
			"occupancies": []gin.H{}, "occupied_kg": 0.0, "wip_kg": roundKg(wip),
		})
	}

	if len(ids) > 0 && !unassigned {
		occQ := `SELECT i.board_id, i.worker_id, COALESCE(e.name,''),
			COALESCE(SUM(i.issue_kg - i.returned_kg - i.completed_kg),0)
			FROM pd_process_issue i
			LEFT JOIN hr_employee e ON e.id=i.worker_id
			WHERE i.status='open' AND COALESCE(i.worker_id,0)>0 AND i.board_id IN (` + placeholders(len(ids)) + `)`
		occArgs := make([]interface{}, 0, len(ids)+2)
		for _, id := range ids {
			occArgs = append(occArgs, id)
		}
		if stepID > 0 {
			occQ += ` AND (i.step_id=? OR (COALESCE(i.step_id,0)=0 AND i.process_id=?))`
			occArgs = append(occArgs, stepID, processID)
		}
		occQ += ` GROUP BY i.board_id, i.worker_id, COALESCE(e.name,'')`
		orows, err := s.DB.Query(occQ, occArgs...)
		if err == nil {
			byBoard := map[int64][]gin.H{}
			byKg := map[int64]float64{}
			for orows.Next() {
				var boardID, workerID int64
				var name string
				var kg float64
				if err := orows.Scan(&boardID, &workerID, &name, &kg); err != nil {
					continue
				}
				if kg <= kgEps {
					continue
				}
				byBoard[boardID] = append(byBoard[boardID], gin.H{"worker_id": workerID, "worker_name": name, "open_kg": roundKg(kg)})
				byKg[boardID] = roundKg(byKg[boardID] + kg)
			}
			orows.Close()
			for i := range list {
				id, _ := list[i]["id"].(int64)
				if occ, ok := byBoard[id]; ok {
					list[i]["occupancies"] = occ
					list[i]["occupied_kg"] = byKg[id]
				}
				avail, _ := list[i]["available_kg"].(float64)
				occKg, _ := list[i]["occupied_kg"].(float64)
				list[i]["wip_kg"] = roundKg(avail + occKg)
			}
		}
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	s := "?"
	for i := 1; i < n; i++ {
		s += ",?"
	}
	return s
}
