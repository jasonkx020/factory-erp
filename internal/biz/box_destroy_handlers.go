package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

// destroyBoxCode marks an unused box as destroyed (loss/scrap), removes it from WIP, and adjusts stock.
func (s *Services) destroyBoxCode(c *gin.Context) bool {
	id := paramID(c)
	if id <= 0 {
		api.FailJSON(c, "ID_REQUIRED")
		return true
	}
	body := bindBody(c)
	reason := strings.TrimSpace(strOr(body["reason"]))
	if reason == "" {
		api.FailJSON(c, "REASON_REQUIRED")
		return true
	}

	var code, status, trace string
	var productID, whID int64
	var weight, qty float64
	err := s.DB.QueryRow(`SELECT code, COALESCE(status,''), COALESCE(trace_code,''), COALESCE(product_id,0),
		COALESCE(warehouse_id,1), COALESCE(weight,0), COALESCE(qty,0)
		FROM inv_box_code WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&code, &status, &trace, &productID, &whID, &weight, &qty)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "destroyed":
		api.FailJSON(c, "ALREADY_DESTROYED")
		return true
	case "finished", "void", "closed":
		api.FailJSON(c, "BOX_STATUS_INVALID")
		return true
	}

	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	before := gin.H{"id": id, "code": code, "status": status, "weight": weight, "trace_code": trace}

	_, err = s.DB.Exec(`UPDATE inv_box_code SET status='destroyed', destroyed_at=NOW(), destroyed_by=?,
		destroy_reason=?, current_process_id=NULL, current_step_id=NULL, updated_at=NOW() WHERE id=?`,
		uid, reason, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}

	outW := weight
	if outW <= 0 {
		outW = qty
	}
	if outW > 0 && productID > 0 {
		if err := s.writeBoxDestroyTxn(id, code, whID, productID, outW, reason); err != nil {
			api.FailJSON(c, "STOCK_ADJUST_ERROR:"+err.Error())
			return true
		}
	}

	after := gin.H{
		"id": id, "code": code, "status": "destroyed", "weight": weight,
		"trace_code": trace, "destroy_reason": reason, "destroyed_by": uid,
	}
	s.writeAuditCtx(c, "box_code", id, "destroy", reason, before, after)
	api.OK(c, after)
	return true
}

func (s *Services) writeBoxDestroyTxn(boxID int64, boxCode string, warehouseID, productID int64, qty float64, reason string) error {
	if qty <= 0 || productID <= 0 {
		return nil
	}
	if warehouseID <= 0 {
		warehouseID = 1
	}
	bizDate := time.Now().Format("2006-01-02")
	docNo := fmt.Sprintf("BXDEST-%d", boxID)
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
		docNo, "box_destroy", bizDate, warehouseID, fmt.Sprintf("destroy %s: %s", boxCode, reason))
	if err != nil {
		docNo = fmt.Sprintf("BXDEST%d", time.Now().UnixNano()%1e12)
		res, err = s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
			docNo, "box_destroy", bizDate, warehouseID, fmt.Sprintf("destroy %s: %s", boxCode, reason))
		if err != nil {
			return err
		}
	}
	tid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'out',?)`,
		tid, 1, productID, qty, qty, bizDate)
	return s.adjustBalance(warehouseID, productID, -qty)
}
