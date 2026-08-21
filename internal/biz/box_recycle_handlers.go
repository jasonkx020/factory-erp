package biz

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

// recycleBoxCode clears a physical board so its code can be reused (keeps code, resets content).
// Unlike destroy, the QR/code stays valid and status returns to open.
func (s *Services) recycleBoxCode(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "foreman", "admin", "sys_admin") {
		return true
	}
	id := paramID(c)
	if id <= 0 {
		api.FailJSON(c, "ID_REQUIRED")
		return true
	}
	body := bindBody(c)
	reason := strings.TrimSpace(strOrDef(body["reason"], "仓管回收板码"))
	confirmClear := boolOr(body["confirm_clear"], false)

	var code, status, trace string
	var productID, whID, processID int64
	var weight, qty float64
	err := s.DB.QueryRow(`SELECT code, COALESCE(status,''), COALESCE(trace_code,''), COALESCE(product_id,0),
		COALESCE(warehouse_id,1), COALESCE(weight,0), COALESCE(qty,0), COALESCE(current_process_id,0)
		FROM inv_box_code WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&code, &status, &trace, &productID, &whID, &weight, &qty, &processID)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "destroyed", "void":
		api.FailJSON(c, "BOX_STATUS_INVALID")
		return true
	}

	var openIssues int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_issue
		WHERE board_id=? AND status='open' AND (issue_kg - returned_kg - completed_kg) > 0.0005`, id).Scan(&openIssues)
	if openIssues > 0 {
		api.FailJSON(c, "OPEN_ISSUES_REMAIN")
		return true
	}
	var pendIn int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_process_stock_in
		WHERE board_id=? AND status='pending_warehouse'`, id).Scan(&pendIn)
	if pendIn > 0 {
		api.FailJSON(c, "STOCK_IN_PENDING")
		return true
	}

	remain := weight
	if remain <= kgEps {
		remain = qty
	}
	remain = roundKg(remain)
	if remain > kgEps && !confirmClear {
		api.HandleBusiness(c, &api.BusinessError{
			Msg: "BOARD_NOT_EMPTY",
			Data: gin.H{
				"id": id, "code": code, "weight": remain, "trace_code": trace,
				"hint": "板上仍有重量，确认清空后可回收（将记仓库出库并清零）",
			},
		}, nil)
		return true
	}

	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	before := gin.H{
		"id": id, "code": code, "status": status, "weight": weight, "qty": qty,
		"trace_code": trace, "product_id": productID, "process_id": processID,
	}

	if remain > kgEps && productID > 0 {
		if err := s.writeBoxRecycleClearTxn(id, code, whID, productID, remain, reason); err != nil {
			api.FailJSON(c, "STOCK_ADJUST_ERROR:"+err.Error())
			return true
		}
	}
	_, _ = s.DB.Exec(`UPDATE inv_balance SET qty=0 WHERE COALESCE(box_code_id,0)=?`, id)

	_, err = s.DB.Exec(`UPDATE inv_box_code SET
		weight=0, qty=0, status='open',
		trace_code='', current_process_id=NULL, current_step_id=NULL,
		task_id=NULL, work_order_id=NULL, parent_box_id=NULL,
		updated_at=NOW()
		WHERE id=?`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}

	after := gin.H{
		"id": id, "code": code, "status": "open", "weight": 0.0, "qty": 0.0,
		"trace_code": "", "cleared_kg": remain, "recycle_reason": reason, "recycled_by": uid,
	}
	s.writeAuditCtx(c, "box_code", id, "recycle", reason, before, after)
	s.appendStationFlowLog(stationFlowEvent{
		EventType: "board_recycle", BoardID: id, BoardCode: code, TraceCode: trace,
		ProcessID: processID, ActorUserID: uid, Kg: remain,
		RefType: "inv_box_code", RefID: id,
		Payload: gin.H{"reason": reason, "confirm_clear": confirmClear},
		After:   after,
	})
	api.OK(c, after)
	return true
}

func (s *Services) writeBoxRecycleClearTxn(boxID int64, boxCode string, warehouseID, productID int64, qty float64, reason string) error {
	if qty <= kgEps || productID <= 0 {
		return nil
	}
	if warehouseID <= 0 {
		warehouseID = 1
	}
	bizDate := time.Now().Format("2006-01-02")
	docNo := fmt.Sprintf("BXRCY%d", time.Now().UnixNano()%1e12)
	remark := fmt.Sprintf("recycle clear %s: %s", boxCode, reason)
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
		docNo, "box_recycle_clear", bizDate, warehouseID, remark)
	if err != nil {
		return err
	}
	tid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'out',?)`,
		tid, 1, productID, qty, qty, boxCode)
	var bid int64
	var onHand float64
	err = s.DB.QueryRow(`SELECT id, qty FROM inv_balance WHERE warehouse_id=? AND product_id=? AND COALESCE(box_code_id,0)=? LIMIT 1`,
		warehouseID, productID, boxID).Scan(&bid, &onHand)
	if err == nil {
		newQty := onHand - qty
		if newQty < 0 {
			newQty = 0
		}
		_, err = s.DB.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, newQty, bid)
		return err
	}
	if math.Abs(qty) > kgEps {
		return s.adjustBalance(warehouseID, productID, -qty)
	}
	return nil
}
