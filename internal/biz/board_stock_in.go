package biz

import (
	"fmt"
	"strings"
	"time"
)

// stockInNewBoardFrom creates a child board code marked with the completed process/step,
// posts produce_in, and links inv_balance to the new box when possible.
func (s *Services) stockInNewBoardFrom(old *boardState, warehouseID, processID, stepID int64, qty float64) (newCode string, newID int64, err error) {
	if old == nil || qty <= kgEps {
		return "", 0, fmt.Errorf("INVALID_QTY")
	}
	if warehouseID <= 0 {
		warehouseID = old.WarehouseID
	}
	if warehouseID <= 0 {
		warehouseID = 1
	}
	productID := old.ProductID
	if productID <= 0 {
		return "", 0, fmt.Errorf("PRODUCT_REQUIRED")
	}
	var farmerID int64
	var trace, origin, receiveDate, sourceType string
	_ = s.DB.QueryRow(`SELECT COALESCE(farmer_id,0), COALESCE(trace_code,''), COALESCE(origin,''), COALESCE(receive_date,''), COALESCE(source_type,'')
		FROM inv_box_code WHERE id=?`, old.ID).Scan(&farmerID, &trace, &origin, &receiveDate, &sourceType)
	if trace == "" {
		trace = old.Trace
	}
	if strings.TrimSpace(trace) == "" {
		return "", 0, fmt.Errorf("TRACE_CODE_REQUIRED")
	}
	newCode = fmt.Sprintf("BX%d", time.Now().UnixNano()%1e12)
	res, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, parent_box_id,
		current_process_id, current_step_id, task_id, work_order_id, farmer_id, trace_code, origin, receive_date, source_type, status)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'open')`,
		newCode, productID, warehouseID, time.Now().Format("20060102"), qty, qty, old.ID,
		processID, stepID, nullIf0(old.TaskID), nullIf0(old.WoID),
		nullIf0(farmerID), trace, origin, receiveDate, sourceType)
	if err != nil {
		return "", 0, err
	}
	newID, _ = res.LastInsertId()
	if newID <= 0 {
		_ = s.DB.QueryRow(`SELECT id FROM inv_box_code WHERE code=?`, newCode).Scan(&newID)
	}
	if err := s.autoStockProduceInBox(newCode, newID, warehouseID, productID, qty); err != nil {
		return newCode, newID, err
	}
	return newCode, newID, nil
}

func (s *Services) autoStockProduceInBox(boxCode string, boxID, warehouseID, productID int64, qty float64) error {
	if warehouseID <= 0 {
		warehouseID = 1
	}
	if productID <= 0 {
		productID = 1
	}
	docNo := fmt.Sprintf("ST%d", time.Now().UnixNano()%1e12)
	bizDate := time.Now().Format("2006-01-02")
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
		docNo, "produce_in", bizDate, warehouseID, "auto:"+boxCode)
	if err != nil {
		return err
	}
	tid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction) VALUES(?,?,?,?,?,'in')`,
		tid, 1, productID, qty, qty)
	if boxID > 0 {
		var bid int64
		var onHand float64
		err = s.DB.QueryRow(`SELECT id, qty FROM inv_balance WHERE warehouse_id=? AND product_id=? AND COALESCE(box_code_id,0)=? LIMIT 1`,
			warehouseID, productID, boxID).Scan(&bid, &onHand)
		if err != nil {
			_, err = s.DB.Exec(`INSERT INTO inv_balance(warehouse_id, location_id, product_id, batch_no, box_code_id, qty) VALUES(?,0,?,'',?,?)`,
				warehouseID, productID, boxID, qty)
			if err != nil {
				return s.adjustBalance(warehouseID, productID, qty)
			}
			return nil
		}
		_, err = s.DB.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, onHand+qty, bid)
		return err
	}
	return s.adjustBalance(warehouseID, productID, qty)
}
