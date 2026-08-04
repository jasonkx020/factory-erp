package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
	"erp/internal/store"
)

type Services struct {
	DB     *sql.DB
	Driver string
	Store  *store.Store
}

func New(db *sql.DB, driver string, st *store.Store) *Services {
	return &Services{DB: db, Driver: driver, Store: st}
}

// Handle returns true if the request was fully handled.
func (s *Services) Handle(c *gin.Context, method, openapiPath, resourceKey, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/product/products") && !strings.Contains(openapiPath, "/units") && !strings.Contains(openapiPath, "/activate") && !strings.Contains(openapiPath, "/deactivate"):
		return s.handleProducts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/balances") && method == "GET":
		s.listBalances(c)
		return true
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/availability") && method == "GET":
		s.listAvailability(c)
		return true
	case openapiPath == "/api/v1/inventory/stock-txns/{id}/post" && method == "POST":
		s.postStockTxn(c)
		return true
	case openapiPath == "/api/v1/inventory/stock-txns" || strings.HasPrefix(openapiPath, "/api/v1/inventory/stock-txns/{id}"):
		return s.handleStockTxns(c, method, action)
	case openapiPath == "/api/v1/production/processes" || openapiPath == "/api/v1/production/processes/{id}":
		return s.handleProcesses(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/tasks"):
		return s.handleProdTasks(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/production/dispatches"):
		return s.handleDispatches(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/production/report-works"):
		return s.handleReportWorks(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/production/requisitions"):
		return s.handleRequisitions(c, method, action, openapiPath)
	case openapiPath == "/api/v1/production/scan" && method == "POST":
		return s.handleScan(c, false)
	case openapiPath == "/api/v1/production/scan/resolve" && method == "POST":
		return s.handleScan(c, true)
	case strings.HasPrefix(openapiPath, "/api/v1/production/flow-events"):
		return s.handleFlowEvents(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/production/flow-rules"):
		return s.handleFlowRules(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/system/operation-logs"):
		return s.handleOperationLogs(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/system/data-repairs"):
		return s.handleDataRepair(c, action)
	case strings.Contains(openapiPath, "/inventory/box-codes/trace/"):
		return s.handleBoxTrace(c)
	case strings.HasPrefix(openapiPath, "/api/v1/payroll/"):
		return s.handlePayroll(c, method, action, openapiPath)
	case strings.Contains(openapiPath, "/hr/employees/{id}/badge"):
		return s.handleBadge(c)
	case strings.Contains(openapiPath, "/hr/employees") && strings.Contains(openapiPath, "/open-account"):
		return s.openEmployeeAccount(c)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/onboards"):
		return s.handleOnboards(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/offboards"):
		return s.handleOffboards(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/employees"):
		return s.handleEmployees(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/"):
		return s.handleHROps(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/approval/tasks"):
		return s.handleApprovalTasks(c, method, action, openapiPath)
	case openapiPath == "/api/v1/iam/hr-perm-overview":
		return s.hrPermOverview(c)
	case strings.Contains(openapiPath, "/iam/users/") && strings.Contains(openapiPath, "/bind-employee"):
		return s.handleHRPermLifecycle(c, method, openapiPath, action)
	case strings.Contains(openapiPath, "/iam/users/") && strings.Contains(openapiPath, "/data-scope"):
		return s.handleHRPermLifecycle(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/iam/"):
		// enhance get user / role detail
		if method == "GET" && action == "get" && strings.HasPrefix(openapiPath, "/api/v1/iam/users/{id}") {
			return s.getUserDetailIAM(c)
		}
		if method == "GET" && action == "get" && strings.HasPrefix(openapiPath, "/api/v1/iam/roles/{id}") {
			return s.getRoleDetailIAM(c)
		}
		return s.handleIAM(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/"):
		return s.handlePurchase(c, method, openapiPath, resourceKey, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/prepay-prepaids") || strings.HasPrefix(openapiPath, "/api/v1/finance/arap-adjusts"):
		if s.handleFinancePartyGuard(c, method, openapiPath, action) {
			return true
		}
	case strings.HasPrefix(openapiPath, "/api/v1/sales/orders") && method == "POST" && action == "create":
		return s.handleSalesOrderCreate(c)
	case openapiPath == "/api/v1/sales/pre-shipments/{id}/confirm" || (strings.Contains(openapiPath, "pre-shipments") && strings.HasSuffix(action, "confirm")):
		return s.handlePreShipConfirm(c)
	case openapiPath == "/api/v1/health":
		return false // keep health package
	case strings.HasPrefix(openapiPath, "/api/v1/auth/"):
		return false
	}
	// real-table CRUD engine for registered resources
	if s.handleTableCRUD(c, resourceKey, action) {
		return true
	}
	return false
}

func (s *Services) handleSalesOrderCreate(c *gin.Context) bool {
	body := bindBody(c)
	body["doc_type"] = "sales_order"
	d, err := s.Store.Create("sales/orders", body, "open")
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	// create reservations for lines
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		wh = 3 // finished goods
	}
	if lines, ok := body["lines"].([]interface{}); ok {
		for _, ln := range lines {
			m, _ := ln.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			qty, _ := asFloat(m["qty"])
			_, _ = s.DB.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, qty, source_doc_type, source_doc_id, status) VALUES(?,?,?,'sales_order',?,'active')`,
				wh, pid, qty, d.ID)
		}
	}
	api.OK(c, d.Payload)
	return true
}

func (s *Services) handlePreShipConfirm(c *gin.Context) bool {
	id := paramID(c)
	d, _ := s.Store.Get(id)
	if d == nil {
		// try create from body
		body := bindBody(c)
		nd, err := s.Store.Update(id, body, "confirmed")
		if err != nil || nd == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, nd.Payload)
		return true
	}
	// convert reservation to outbound draft
	d.Payload["txn_type"] = "sale_out"
	d.Payload["doc_type"] = "sale_out"
	_ = s.applyPayloadStock(d)
	_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='closed' WHERE source_doc_type='sales_order' AND source_doc_id=?`, id)
	nd, _ := s.Store.SetStatus(id, "shipped")
	api.OK(c, nd.Payload)
	return true
}

func (s *Services) ApplyDocAction(id int64, action string, body map[string]interface{}) (map[string]interface{}, error) {
	d, err := s.Store.Get(id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, api.Fail("NOT_FOUND")
	}
	statusMap := map[string]string{
		"post": "posted", "submit": "submitted", "approve": "approved", "reject": "rejected",
		"cancel": "cancelled", "confirm": "confirmed", "activate": "active", "deactivate": "inactive",
		"pass": "passed", "fail": "failed", "release": "released", "close": "closed",
	}
	if st, ok := statusMap[action]; ok {
		// inventory post via doc: also try stock effects if payload has lines
		if action == "post" && strings.Contains(d.ResourceKey, "inventory") {
			if err := s.applyPayloadStock(d); err != nil {
				return nil, err
			}
		}
		nd, err := s.Store.SetStatus(id, st)
		if err != nil {
			return nil, err
		}
		if body != nil {
			nd, _ = s.Store.Update(id, body, st)
		}
		return nd.Payload, nil
	}
	payload := d.Payload
	if body != nil {
		for k, v := range body {
			payload[k] = v
		}
	}
	payload["last_action"] = action
	payload["last_action_at"] = time.Now().Format("2006-01-02 15:04:05")
	nd, err := s.Store.Update(id, payload, d.Status)
	if err != nil {
		return nil, err
	}
	return nd.Payload, nil
}

func (s *Services) applyPayloadStock(d *store.Doc) error {
	wh, _ := asInt64(d.Payload["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	lines, _ := d.Payload["lines"].([]interface{})
	txnType, _ := d.Payload["txn_type"].(string)
	if txnType == "" {
		txnType = "adjust"
	}
	sign := 1.0
	if strings.Contains(txnType, "out") || txnType == "sale_out" || txnType == "consume" {
		sign = -1
	}
	for _, ln := range lines {
		m, ok := ln.(map[string]interface{})
		if !ok {
			continue
		}
		pid, _ := asInt64(m["product_id"])
		qty, _ := asFloat(m["qty"])
		if pid == 0 || qty == 0 {
			continue
		}
		if err := s.adjustBalance(wh, pid, sign*qty); err != nil {
			return err
		}
	}
	return nil
}

func (s *Services) adjustBalance(warehouseID, productID int64, delta float64) error {
	return s.adjustBalanceBatch(warehouseID, productID, "", delta)
}

func (s *Services) adjustBalanceBatch(warehouseID, productID int64, batchNo string, delta float64) error {
	var id int64
	var qty float64
	var err error
	if batchNo != "" {
		err = s.DB.QueryRow(`SELECT id, qty FROM inv_balance WHERE warehouse_id=? AND product_id=? AND COALESCE(batch_no,'')=? LIMIT 1`,
			warehouseID, productID, batchNo).Scan(&id, &qty)
	}
	if err == sql.ErrNoRows || batchNo == "" {
		err = s.DB.QueryRow(`SELECT id, qty FROM inv_balance WHERE warehouse_id=? AND product_id=? ORDER BY id LIMIT 1`,
			warehouseID, productID).Scan(&id, &qty)
	}
	if err == sql.ErrNoRows {
		_, err = s.DB.Exec(`INSERT INTO inv_balance(warehouse_id, location_id, product_id, batch_no, box_code_id, qty) VALUES(?,0,?,?,0,?)`,
			warehouseID, productID, batchNo, delta)
		return err
	}
	if err != nil {
		return err
	}
	newQty := qty + delta
	if newQty < -0.0001 {
		return api.Fail("INSUFFICIENT_STOCK")
	}
	_, err = s.DB.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, newQty, id)
	return err
}

func asInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	case json.Number:
		i, err := t.Int64()
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(t, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func asFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int64:
		return float64(t), true
	case int:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func paramID(c *gin.Context) int64 {
	for _, k := range []string{"id", "product_id"} {
		if v := c.Param(k); v != "" {
			id, _ := strconv.ParseInt(v, 10, 64)
			return id
		}
	}
	return 0
}

func (s *Services) listBalances(c *gin.Context) {
	rows, err := s.DB.Query(`
SELECT b.id, b.warehouse_id, w.name, b.product_id, p.code, p.name, b.qty, COALESCE(b.batch_no,'')
FROM inv_balance b
JOIN inv_warehouse w ON w.id=b.warehouse_id
JOIN prd_product p ON p.id=b.product_id
ORDER BY b.id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, whID, pid int64
		var whName, code, name, batch string
		var onHand float64
		_ = rows.Scan(&id, &whID, &whName, &pid, &code, &name, &onHand, &batch)
		list = append(list, gin.H{
			"id": id, "warehouse_id": whID, "warehouse_name": whName,
			"product_id": pid, "product_code": code, "product_name": name,
			"batch_no": batch, "qty": onHand, "qty_on_hand": onHand,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
}

func (s *Services) listAvailability(c *gin.Context) {
	rows, err := s.DB.Query(`
SELECT b.warehouse_id, b.product_id, b.qty,
  COALESCE((SELECT SUM(qty) FROM inv_reservation r WHERE r.warehouse_id=b.warehouse_id AND r.product_id=b.product_id AND r.status='active'),0)
FROM inv_balance b`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var wh, pid int64
		var onHand, reserved float64
		_ = rows.Scan(&wh, &pid, &onHand, &reserved)
		list = append(list, gin.H{
			"warehouse_id": wh, "product_id": pid,
			"on_hand": onHand, "reserved": reserved, "available": onHand - reserved,
		})
	}
	api.OK(c, gin.H{"list": list})
}

func (s *Services) postStockTxn(c *gin.Context) {
	id := paramID(c)
	tx, err := s.DB.Begin()
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var status, docType string
	var whID sql.NullInt64
	err = tx.QueryRow(`SELECT status, doc_type, warehouse_id FROM inv_stock_txn WHERE id=? AND is_deleted=0`, id).Scan(&status, &docType, &whID)
	if err == sql.ErrNoRows {
		api.FailJSON(c, "NOT_FOUND")
		return
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	if status == "posted" {
		api.FailJSON(c, "ALREADY_POSTED")
		return
	}
	if status == "cancelled" {
		api.FailJSON(c, "DOC_LOCKED")
		return
	}
	wh := whID.Int64
	if wh == 0 {
		wh = 1
	}
	rows, err := tx.Query(`SELECT product_id, qty, direction FROM inv_stock_txn_line WHERE txn_id=?`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	type line struct {
		pid int64
		qty float64
		dir string
	}
	var lines []line
	for rows.Next() {
		var l line
		_ = rows.Scan(&l.pid, &l.qty, &l.dir)
		lines = append(lines, l)
	}
	rows.Close()
	for _, l := range lines {
		delta := l.qty
		if l.dir == "out" || strings.Contains(docType, "out") {
			delta = -l.qty
		}
		var bid int64
		var onHand float64
		err = tx.QueryRow(`SELECT id, qty FROM inv_balance WHERE warehouse_id=? AND product_id=? AND batch_no='' AND location_id=0 AND box_code_id=0`, wh, l.pid).Scan(&bid, &onHand)
		if err == sql.ErrNoRows {
			if delta < 0 {
				api.FailJSON(c, "INSUFFICIENT_STOCK")
				return
			}
			_, err = tx.Exec(`INSERT INTO inv_balance(warehouse_id, location_id, product_id, batch_no, box_code_id, qty) VALUES(?,0,?,'',0,?)`, wh, l.pid, delta)
		} else if err == nil {
			nq := onHand + delta
			if nq < -0.0001 {
				api.FailJSON(c, "INSUFFICIENT_STOCK")
				return
			}
			_, err = tx.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, nq, bid)
		}
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = tx.Exec(`UPDATE inv_stock_txn SET status='posted', posted_at=? WHERE id=?`, now, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	if err := tx.Commit(); err != nil {
		api.FailJSON(c, "DB_ERROR")
		return
	}
	api.OK(c, gin.H{"id": id, "status": "posted"})
}

func bindBody(c *gin.Context) map[string]interface{} {
	var body map[string]interface{}
	_ = c.ShouldBindJSON(&body)
	if body == nil {
		body = map[string]interface{}{}
	}
	return body
}

func (s *Services) handleStockTxns(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, doc_no, doc_type, COALESCE(warehouse_id,0), status, COALESCE(remark,''), created_at FROM inv_stock_txn WHERE is_deleted=0 ORDER BY id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, wh int64
			var docNo, typ, status, remark, created string
			_ = rows.Scan(&id, &docNo, &typ, &wh, &status, &remark, &created)
			list = append(list, gin.H{"id": id, "doc_no": docNo, "doc_type": typ, "txn_type": typ, "warehouse_id": wh, "status": status, "remark": remark, "created_at": created})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		docNo, _ := body["doc_no"].(string)
		if docNo == "" {
			docNo = fmt.Sprintf("ST%d", time.Now().UnixNano())
		}
		docType, _ := body["doc_type"].(string)
		if docType == "" {
			docType, _ = body["txn_type"].(string)
		}
		if docType == "" {
			docType = "adjust"
		}
		wh, _ := asInt64(body["warehouse_id"])
		if wh == 0 {
			wh = 1
		}
		remark, _ := body["remark"].(string)
		bizDate := time.Now().Format("2006-01-02")
		res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'draft',?,?)`,
			docNo, docType, bizDate, wh, remark)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		tid, _ := res.LastInsertId()
		lineNo := 1
		if lines, ok := body["lines"].([]interface{}); ok {
			for _, ln := range lines {
				m, _ := ln.(map[string]interface{})
				if m == nil {
					continue
				}
				pid, _ := asInt64(m["product_id"])
				qty, _ := asFloat(m["qty"])
				dir, _ := m["direction"].(string)
				if dir == "" {
					dir = "in"
				}
				_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction) VALUES(?,?,?,?,?,?)`, tid, lineNo, pid, qty, qty, dir)
				lineNo++
			}
		}
		api.OK(c, gin.H{"id": tid, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		id := paramID(c)
		var docNo, typ, status, remark, created string
		var wh int64
		err := s.DB.QueryRow(`SELECT doc_no, doc_type, COALESCE(warehouse_id,0), status, COALESCE(remark,''), created_at FROM inv_stock_txn WHERE id=? AND is_deleted=0`, id).
			Scan(&docNo, &typ, &wh, &status, &remark, &created)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		rows, _ := s.DB.Query(`SELECT id, product_id, qty, direction FROM inv_stock_txn_line WHERE txn_id=?`, id)
		lines := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var lid, pid int64
				var qty float64
				var dir string
				_ = rows.Scan(&lid, &pid, &qty, &dir)
				lines = append(lines, gin.H{"id": lid, "product_id": pid, "qty": qty, "direction": dir})
			}
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "doc_type": typ, "warehouse_id": wh, "status": status, "remark": remark, "created_at": created, "lines": lines})
		return true
	case "action:cancel":
		id := paramID(c)
		var status string
		_ = s.DB.QueryRow(`SELECT status FROM inv_stock_txn WHERE id=?`, id).Scan(&status)
		if status == "posted" {
			api.FailJSON(c, "DOC_LOCKED")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE inv_stock_txn SET status='cancelled' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "cancelled"})
		return true
	}
	return false
}

func pageOK(c *gin.Context, list interface{}, total int) {
	pn, ps := sqlutil.Page(c)
	api.PageOK(c, list, total, pn, ps)
}
