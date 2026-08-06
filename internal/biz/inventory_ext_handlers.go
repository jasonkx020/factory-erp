package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsureInventoryExtSchema creates factory-delivery inventory satellite tables (SQLite).
func EnsureInventoryExtSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS inv_in_transit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER NOT NULL,
  warehouse_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  transit_type TEXT NOT NULL DEFAULT 'purchase',
  source_doc_type TEXT NOT NULL DEFAULT '',
  source_doc_id INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS inv_inbound_qc (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  stock_txn_id INTEGER,
  product_id INTEGER NOT NULL,
  qty_check REAL NOT NULL DEFAULT 0,
  qty_pass REAL NOT NULL DEFAULT 0,
  qty_fail REAL NOT NULL DEFAULT 0,
  result TEXT,
  inspector_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS inv_stocktake (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  stocktake_type TEXT NOT NULL DEFAULT 'warehouse',
  warehouse_id INTEGER,
  workshop_id INTEGER,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS inv_stocktake_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  stocktake_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  book_qty REAL NOT NULL DEFAULT 0,
  count_qty REAL NOT NULL DEFAULT 0,
  diff_qty REAL NOT NULL DEFAULT 0,
  batch_no TEXT,
  location_id INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS inv_transfer (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  from_warehouse_id INTEGER NOT NULL,
  to_warehouse_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS inv_transfer_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transfer_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  base_qty REAL NOT NULL,
  batch_no TEXT
)`,
		`CREATE TABLE IF NOT EXISTS inv_consume (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  warehouse_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS inv_consume_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  consume_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  batch_no TEXT
)`,
		`CREATE TABLE IF NOT EXISTS inv_assemble_split (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  biz_type TEXT NOT NULL DEFAULT 'assemble',
  warehouse_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS inv_assemble_split_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  header_id INTEGER NOT NULL,
  role_type TEXT NOT NULL DEFAULT 'child',
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS inv_price_adjust (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  product_id INTEGER NOT NULL,
  old_price REAL NOT NULL DEFAULT 0,
  new_price REAL NOT NULL,
  effective_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS inv_stock_alert_rule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER,
  warehouse_id INTEGER,
  alert_type TEXT NOT NULL,
  min_qty REAL,
  max_qty REAL,
  is_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS inv_sales_peel_return (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  sales_order_id INTEGER,
  product_id INTEGER NOT NULL,
  peel_qty REAL NOT NULL,
  weight REAL,
  warehouse_id INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS inv_material_to_payable (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  consume_txn_id INTEGER,
  supplier_id INTEGER,
  product_id INTEGER,
  qty REAL,
  amount REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			fmt.Printf("[inventory-ext] schema warn: %v\n", err)
		}
	}
}

func (s *Services) handleInventoryExt(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case openapiPath == "/api/v1/inventory/warehouses" && method == "GET":
		return s.listWarehouses(c)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/stocktakes") || strings.HasPrefix(openapiPath, "/api/v1/inventory/workshop-stocktakes"):
		stType := "warehouse"
		if strings.Contains(openapiPath, "workshop-stocktakes") {
			stType = "workshop"
		}
		return s.handleStocktakes(c, method, action, stType)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/stocktake-records"):
		return s.handleStocktakeRecords(c, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/transfers"):
		return s.handleTransfers(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/consumes"):
		return s.handleConsumes(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/inbound-qcs"):
		return s.handleInboundQCs(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/alert-rules/"):
		return s.handleAlertRules(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/reservations"):
		return s.handleReservations(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/in-transits"):
		return s.handleInTransits(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/openings"):
		return s.handleOpenings(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/assemble-splits"):
		return s.handleAssembleSplits(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/price-adjusts"):
		return s.handlePriceAdjusts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/sales-peel-returns"):
		return s.handleSalesPeelReturns(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/material-to-payables"):
		return s.handleMaterialToPayables(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/inventory/purchase-returns"):
		return s.handleInventoryPurchaseReturns(c, action)
	default:
		return false
	}
}

func (s *Services) listWarehouses(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(warehouse_type,'') FROM inv_warehouse WHERE COALESCE(is_deleted,0)=0 ORDER BY id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, wtype string
		if err := rows.Scan(&id, &code, &name, &wtype); err != nil {
			continue
		}
		list = append(list, gin.H{"id": id, "code": code, "name": name, "warehouse_type": wtype})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

// ---------- stocktakes ----------

func (s *Services) handleStocktakes(c *gin.Context, method, action, stType string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, fmt.Sprintf(`SELECT * FROM inv_stocktake WHERE COALESCE(is_deleted,0)=0 AND stocktake_type='%s'`, stType))
	case "get":
		api.OK(c, s.loadStocktake(paramID(c)))
		return true
	case "create":
		return s.createStocktake(c, stType)
	case "update", "replace":
		return s.updateStocktake(c)
	case "action:submit":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE inv_stocktake SET status='submitted', updated_at=datetime('now') WHERE id=? AND status='draft'`, id)
		api.OK(c, s.loadStocktake(id))
		return true
	case "action:post":
		return s.postStocktake(c)
	}
	return true
}

func (s *Services) createStocktake(c *gin.Context, stType string) bool {
	body := bindBody(c)
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("STK%s", time.Now().Format("060102150405")))
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	ws, _ := asInt64(body["workshop_id"])
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	res, err := s.DB.Exec(`INSERT INTO inv_stocktake(doc_no, stocktake_type, warehouse_id, workshop_id, biz_date, status, remark)
		VALUES(?,?,?,?,?,'draft',?)`, docNo, stType, wh, nullInt(ws), bizDate, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	s.insertStocktakeLines(id, wh, body)
	api.OK(c, s.loadStocktake(id))
	return true
}

func (s *Services) insertStocktakeLines(id, wh int64, body map[string]interface{}) {
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		pid, _ := asInt64(body["product_id"])
		countQty, _ := asFloat(body["count_qty"])
		if pid == 0 {
			return
		}
		book := s.bookQty(wh, pid)
		_, _ = s.DB.Exec(`INSERT INTO inv_stocktake_line(stocktake_id, product_id, book_qty, count_qty, diff_qty, batch_no)
			VALUES(?,?,?,?,?,?)`, id, pid, book, countQty, countQty-book, strOr(body["batch_no"]))
		return
	}
	for _, ln := range lines {
		m, _ := ln.(map[string]interface{})
		if m == nil {
			continue
		}
		pid, _ := asInt64(m["product_id"])
		countQty, _ := asFloat(m["count_qty"])
		book, ok := asFloat(m["book_qty"])
		if !ok {
			book = s.bookQty(wh, pid)
		}
		_, _ = s.DB.Exec(`INSERT INTO inv_stocktake_line(stocktake_id, product_id, book_qty, count_qty, diff_qty, batch_no)
			VALUES(?,?,?,?,?,?)`, id, pid, book, countQty, countQty-book, strOr(m["batch_no"]))
	}
}

func (s *Services) bookQty(wh, pid int64) float64 {
	var q float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM inv_balance WHERE warehouse_id=? AND product_id=?`, wh, pid).Scan(&q)
	return q
}

func (s *Services) loadStocktake(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM inv_stocktake WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM inv_stocktake_line WHERE stocktake_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) updateStocktake(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	_, _ = s.DB.Exec(`UPDATE inv_stocktake SET remark=COALESCE(NULLIF(?,''),remark), warehouse_id=COALESCE(NULLIF(?,0),warehouse_id),
		updated_at=datetime('now') WHERE id=? AND status IN ('draft','submitted')`,
		strOr(body["remark"]), nullInt64Or(body["warehouse_id"]), id)
	if lines, ok := body["lines"].([]interface{}); ok && len(lines) > 0 {
		_, _ = s.DB.Exec(`DELETE FROM inv_stocktake_line WHERE stocktake_id=?`, id)
		var wh int64 = 1
		_ = s.DB.QueryRow(`SELECT COALESCE(warehouse_id,1) FROM inv_stocktake WHERE id=?`, id).Scan(&wh)
		s.insertStocktakeLines(id, wh, body)
	}
	api.OK(c, s.loadStocktake(id))
	return true
}

func (s *Services) postStocktake(c *gin.Context) bool {
	id := paramID(c)
	st := s.loadStocktake(id)
	if st == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status, _ := st["status"].(string); status == "posted" {
		api.OK(c, st)
		return true
	}
	wh, _ := asInt64(st["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	lines, _ := st["lines"].([]map[string]interface{})
	if lines == nil {
		if raw, ok := st["lines"].([]interface{}); ok {
			for _, r := range raw {
				if m, ok := r.(map[string]interface{}); ok {
					lines = append(lines, m)
				}
			}
		}
	}
	bizDate := strOrDef(st["biz_date"], time.Now().Format("2006-01-02"))
	docNo := strOrDef(st["doc_no"], fmt.Sprintf("STK%d", id))
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		diff, _ := asFloat(ln["diff_qty"])
		if pid == 0 || diff == 0 {
			continue
		}
		dir := "in"
		docType := "stocktake_gain"
		qty := diff
		if diff < 0 {
			dir = "out"
			docType = "stocktake_loss"
			qty = -diff
		}
		txnID, err := s.insertPostedStockTxn(docType, wh, bizDate, fmt.Sprintf("%s-%s-%d", docNo, docType, pid),
			[]txnLine{{pid: pid, qty: qty, dir: dir}}, fmt.Sprintf("stocktake:%d", id))
		if err != nil {
			api.FailJSON(c, "POST_ERROR:"+err.Error())
			return true
		}
		_ = txnID
	}
	_, _ = s.DB.Exec(`UPDATE inv_stocktake SET status='posted', updated_at=datetime('now') WHERE id=?`, id)
	api.OK(c, s.loadStocktake(id))
	return true
}

func (s *Services) handleStocktakeRecords(c *gin.Context, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_stocktake WHERE COALESCE(is_deleted,0)=0 AND status='posted'`)
	case "get":
		api.OK(c, s.loadStocktake(paramID(c)))
		return true
	}
	return true
}

type txnLine struct {
	pid int64
	qty float64
	dir string
}

func (s *Services) insertPostedStockTxn(docType string, wh int64, bizDate, docNo string, lines []txnLine, remark string) (int64, error) {
	if docNo == "" {
		docNo = fmt.Sprintf("ST%d", time.Now().UnixNano()%1e12)
	}
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark)
		VALUES(?,?,?,'draft',?,?)`, docNo, docType, bizDate, wh, remark)
	if err != nil {
		return 0, err
	}
	tid, _ := res.LastInsertId()
	for i, ln := range lines {
		_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction)
			VALUES(?,?,?,?,?,?)`, tid, i+1, ln.pid, ln.qty, ln.qty, ln.dir)
	}
	// post balances
	for _, ln := range lines {
		delta := ln.qty
		if ln.dir == "out" {
			delta = -ln.qty
		}
		if err := s.adjustBalance(wh, ln.pid, delta); err != nil {
			return tid, err
		}
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = s.DB.Exec(`UPDATE inv_stock_txn SET status='posted', posted_at=? WHERE id=?`, now, tid)
	return tid, err
}

func nullInt64Or(v interface{}) int64 {
	i, _ := asInt64(v)
	return i
}

// ---------- transfers ----------

func (s *Services) handleTransfers(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_transfer WHERE COALESCE(is_deleted,0)=0`)
	case "get":
		api.OK(c, s.loadTransfer(paramID(c)))
		return true
	case "create":
		return s.createTransfer(c)
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_transfer SET remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=? AND status='draft'`,
			strOr(body["remark"]), id)
		api.OK(c, s.loadTransfer(id))
		return true
	case "action:post":
		return s.postTransfer(c)
	}
	return true
}

func (s *Services) createTransfer(c *gin.Context) bool {
	body := bindBody(c)
	fromWh, _ := asInt64(body["from_warehouse_id"])
	toWh, _ := asInt64(body["to_warehouse_id"])
	if fromWh == 0 {
		fromWh = 1
	}
	if toWh == 0 {
		toWh = 2
	}
	if fromWh == toWh {
		api.FailJSON(c, "WAREHOUSE_SAME")
		return true
	}
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("TF%s", time.Now().Format("060102150405")))
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	res, err := s.DB.Exec(`INSERT INTO inv_transfer(doc_no, from_warehouse_id, to_warehouse_id, biz_date, status, remark)
		VALUES(?,?,?,?,'draft',?)`, docNo, fromWh, toWh, bizDate, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if pid > 0 && qty > 0 {
			_, _ = s.DB.Exec(`INSERT INTO inv_transfer_line(transfer_id, product_id, qty, base_qty, batch_no) VALUES(?,?,?,?,?)`,
				id, pid, qty, qty, strOr(body["batch_no"]))
		}
	} else {
		for _, ln := range lines {
			m, _ := ln.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			qty, _ := asFloat(m["qty"])
			_, _ = s.DB.Exec(`INSERT INTO inv_transfer_line(transfer_id, product_id, qty, base_qty, batch_no) VALUES(?,?,?,?,?)`,
				id, pid, qty, qty, strOr(m["batch_no"]))
		}
	}
	api.OK(c, s.loadTransfer(id))
	return true
}

func (s *Services) loadTransfer(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM inv_transfer WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM inv_transfer_line WHERE transfer_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) postTransfer(c *gin.Context) bool {
	id := paramID(c)
	tf := s.loadTransfer(id)
	if tf == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if st, _ := tf["status"].(string); st == "posted" {
		api.OK(c, tf)
		return true
	}
	fromWh, _ := asInt64(tf["from_warehouse_id"])
	toWh, _ := asInt64(tf["to_warehouse_id"])
	bizDate := strOrDef(tf["biz_date"], time.Now().Format("2006-01-02"))
	docNo := strOrDef(tf["doc_no"], fmt.Sprintf("TF%d", id))
	lrows, _ := s.DB.Query(`SELECT product_id, qty FROM inv_transfer_line WHERE transfer_id=?`, id)
	var outLines, inLines []txnLine
	if lrows != nil {
		for lrows.Next() {
			var pid int64
			var qty float64
			_ = lrows.Scan(&pid, &qty)
			outLines = append(outLines, txnLine{pid: pid, qty: qty, dir: "out"})
			inLines = append(inLines, txnLine{pid: pid, qty: qty, dir: "in"})
		}
		lrows.Close()
	}
	if _, err := s.insertPostedStockTxn("transfer", fromWh, bizDate, docNo+"-OUT", outLines, fmt.Sprintf("transfer:%d", id)); err != nil {
		api.FailJSON(c, "POST_ERROR:"+err.Error())
		return true
	}
	if _, err := s.insertPostedStockTxn("transfer", toWh, bizDate, docNo+"-IN", inLines, fmt.Sprintf("transfer:%d", id)); err != nil {
		api.FailJSON(c, "POST_ERROR:"+err.Error())
		return true
	}
	// close transit if any open for this transfer
	_, _ = s.DB.Exec(`UPDATE inv_in_transit SET status='closed', updated_at=datetime('now')
		WHERE source_doc_type='inv_transfer' AND source_doc_id=?`, id)
	_, _ = s.DB.Exec(`UPDATE inv_transfer SET status='posted', updated_at=datetime('now') WHERE id=?`, id)
	api.OK(c, s.loadTransfer(id))
	return true
}

// ---------- consumes ----------

func (s *Services) handleConsumes(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_consume WHERE COALESCE(is_deleted,0)=0`)
	case "get":
		api.OK(c, s.loadConsume(paramID(c)))
		return true
	case "create":
		return s.createConsume(c)
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_consume SET remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=? AND status='draft'`,
			strOr(body["remark"]), id)
		api.OK(c, s.loadConsume(id))
		return true
	case "action:post":
		return s.postConsume(c)
	}
	return true
}

func (s *Services) createConsume(c *gin.Context) bool {
	body := bindBody(c)
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("CS%s", time.Now().Format("060102150405")))
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	res, err := s.DB.Exec(`INSERT INTO inv_consume(doc_no, warehouse_id, biz_date, status, remark) VALUES(?,?,?,'draft',?)`,
		docNo, wh, bizDate, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if pid > 0 && qty > 0 {
			_, _ = s.DB.Exec(`INSERT INTO inv_consume_line(consume_id, product_id, qty, batch_no) VALUES(?,?,?,?)`,
				id, pid, qty, strOr(body["batch_no"]))
		}
	} else {
		for _, ln := range lines {
			m, _ := ln.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			qty, _ := asFloat(m["qty"])
			_, _ = s.DB.Exec(`INSERT INTO inv_consume_line(consume_id, product_id, qty, batch_no) VALUES(?,?,?,?)`,
				id, pid, qty, strOr(m["batch_no"]))
		}
	}
	api.OK(c, s.loadConsume(id))
	return true
}

func (s *Services) loadConsume(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM inv_consume WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM inv_consume_line WHERE consume_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) postConsume(c *gin.Context) bool {
	id := paramID(c)
	cs := s.loadConsume(id)
	if cs == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if st, _ := cs["status"].(string); st == "posted" {
		api.OK(c, cs)
		return true
	}
	wh, _ := asInt64(cs["warehouse_id"])
	bizDate := strOrDef(cs["biz_date"], time.Now().Format("2006-01-02"))
	docNo := strOrDef(cs["doc_no"], fmt.Sprintf("CS%d", id))
	lrows, _ := s.DB.Query(`SELECT product_id, qty FROM inv_consume_line WHERE consume_id=?`, id)
	var lines []txnLine
	if lrows != nil {
		for lrows.Next() {
			var pid int64
			var qty float64
			_ = lrows.Scan(&pid, &qty)
			lines = append(lines, txnLine{pid: pid, qty: qty, dir: "out"})
		}
		lrows.Close()
	}
	txnID, err := s.insertPostedStockTxn("consume", wh, bizDate, docNo, lines, fmt.Sprintf("consume:%d", id))
	if err != nil {
		api.FailJSON(c, "POST_ERROR:"+err.Error())
		return true
	}
	_, _ = s.DB.Exec(`UPDATE inv_consume SET status='posted', updated_at=datetime('now') WHERE id=?`, id)
	out := s.loadConsume(id)
	out["stock_txn_id"] = txnID
	api.OK(c, out)
	return true
}

// ---------- inbound QC ----------

func (s *Services) handleInboundQCs(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_inbound_qc WHERE COALESCE(is_deleted,0)=0`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM inv_inbound_qc WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("IQC%s", time.Now().Format("060102150405")))
		pid, _ := asInt64(body["product_id"])
		if pid == 0 {
			pid = 1
		}
		qtyCheck, _ := asFloat(body["qty_check"])
		res, err := s.DB.Exec(`INSERT INTO inv_inbound_qc(doc_no, stock_txn_id, product_id, qty_check, status, remark)
			VALUES(?,?,?,?,'draft',?)`, docNo, nullInt64Or(body["stock_txn_id"]), pid, qtyCheck, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		return s.getSimpleRow(c, `SELECT * FROM inv_inbound_qc WHERE id=?`, id)
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_inbound_qc SET qty_check=COALESCE(?,qty_check), remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now')
			WHERE id=? AND status='draft'`, nullFloat(body["qty_check"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM inv_inbound_qc WHERE id=?`, id)
	case "action:pass":
		return s.finishInboundQC(c, true)
	case "action:fail":
		return s.finishInboundQC(c, false)
	}
	return true
}

func (s *Services) finishInboundQC(c *gin.Context, pass bool) bool {
	id := paramID(c)
	body := bindBody(c)
	var qtyCheck float64
	_ = s.DB.QueryRow(`SELECT qty_check FROM inv_inbound_qc WHERE id=?`, id).Scan(&qtyCheck)
	qtyPass, _ := asFloat(body["qty_pass"])
	qtyFail, _ := asFloat(body["qty_fail"])
	result := "pass"
	status := "passed"
	if !pass {
		result = "fail"
		status = "failed"
		if qtyFail == 0 {
			qtyFail = qtyCheck
		}
	} else if qtyPass == 0 {
		qtyPass = qtyCheck
	}
	_, _ = s.DB.Exec(`UPDATE inv_inbound_qc SET qty_pass=?, qty_fail=?, result=?, status=?, updated_at=datetime('now') WHERE id=?`,
		qtyPass, qtyFail, result, status, id)
	return s.getSimpleRow(c, `SELECT * FROM inv_inbound_qc WHERE id=?`, id)
}

// ---------- alert rules ----------

func (s *Services) handleAlertRules(c *gin.Context, method, openapiPath, action string) bool {
	alertType := "shortage"
	if strings.Contains(openapiPath, "/excess") {
		alertType = "excess"
	}
	switch {
	case method == "GET" || action == "list":
		rows, err := s.DB.Query(`SELECT * FROM inv_stock_alert_rule WHERE alert_type=? AND is_enabled=1 ORDER BY id DESC`, alertType)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		rules, _ := rowsToMaps(rows)
		// compute hits against balances
		hits := []gin.H{}
		for _, r := range rules {
			pid, _ := asInt64(r["product_id"])
			wh, _ := asInt64(r["warehouse_id"])
			minQ, _ := asFloat(r["min_qty"])
			maxQ, _ := asFloat(r["max_qty"])
			qSQL := `SELECT b.warehouse_id, b.product_id, SUM(b.qty) FROM inv_balance b WHERE 1=1`
			args := []interface{}{}
			if wh > 0 {
				qSQL += ` AND b.warehouse_id=?`
				args = append(args, wh)
			}
			if pid > 0 {
				qSQL += ` AND b.product_id=?`
				args = append(args, pid)
			}
			qSQL += ` GROUP BY b.warehouse_id, b.product_id`
			brows, _ := s.DB.Query(qSQL, args...)
			if brows == nil {
				continue
			}
			for brows.Next() {
				var bwh, bpid int64
				var qty float64
				_ = brows.Scan(&bwh, &bpid, &qty)
				hit := false
				if alertType == "shortage" && minQ > 0 && qty < minQ {
					hit = true
				}
				if alertType == "excess" && maxQ > 0 && qty > maxQ {
					hit = true
				}
				if hit {
					hits = append(hits, gin.H{
						"rule_id": r["id"], "warehouse_id": bwh, "product_id": bpid,
						"qty": qty, "min_qty": minQ, "max_qty": maxQ, "alert_type": alertType,
					})
				}
			}
			brows.Close()
		}
		api.OK(c, gin.H{"list": rules, "hits": hits, "total": len(rules)})
		return true
	case method == "PUT" || action == "replace":
		body := bindBody(c)
		// upsert single rule
		pid, _ := asInt64(body["product_id"])
		wh, _ := asInt64(body["warehouse_id"])
		minQ := nullFloat(body["min_qty"])
		maxQ := nullFloat(body["max_qty"])
		if id, ok := asInt64(body["id"]); ok && id > 0 {
			_, _ = s.DB.Exec(`UPDATE inv_stock_alert_rule SET product_id=?, warehouse_id=?, min_qty=?, max_qty=?, is_enabled=? WHERE id=?`,
				nullInt(pid), nullInt(wh), minQ, maxQ, 1, id)
			api.OK(c, gin.H{"id": id, "alert_type": alertType})
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO inv_stock_alert_rule(product_id, warehouse_id, alert_type, min_qty, max_qty, is_enabled)
			VALUES(?,?,?,?,?,1)`, nullInt(pid), nullInt(wh), alertType, minQ, maxQ)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		nid, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": nid, "alert_type": alertType})
		return true
	}
	return true
}

// ---------- reservations / in-transits ----------

func (s *Services) handleReservations(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_reservation`)
	case "action:release":
		id := paramID(c)
		_, err := s.DB.Exec(`UPDATE inv_reservation SET status='released', updated_at=datetime('now') WHERE id=?`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id, "status": "released"})
		return true
	}
	return true
}

func (s *Services) handleInTransits(c *gin.Context, method, action string) bool {
	_ = method
	_ = action
	return s.listDocTable(c, `SELECT * FROM inv_in_transit`)
}

// ---------- openings (stock_txn doc_type=opening) ----------

func (s *Services) handleOpenings(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_stock_txn WHERE COALESCE(is_deleted,0)=0 AND doc_type='opening'`)
	case "create":
		body := bindBody(c)
		body["doc_type"] = "opening"
		body["txn_type"] = "opening"
		if lines, ok := body["lines"].([]interface{}); !ok || len(lines) == 0 {
			pid, _ := asInt64(body["product_id"])
			qty, _ := asFloat(body["qty"])
			if pid > 0 && qty > 0 {
				body["lines"] = []interface{}{
					map[string]interface{}{"product_id": pid, "qty": qty, "direction": "in"},
				}
			}
		}
		// reuse create path
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("OP%s", time.Now().Format("060102150405")))
		wh, _ := asInt64(body["warehouse_id"])
		if wh == 0 {
			wh = 1
		}
		bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark)
			VALUES(?,?,?,'draft',?,?)`, docNo, "opening", bizDate, wh, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
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
				_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction)
					VALUES(?,?,?,?,?,'in')`, tid, lineNo, pid, qty, qty)
				lineNo++
			}
		}
		api.OK(c, gin.H{"id": tid, "doc_no": docNo, "doc_type": "opening", "status": "draft"})
		return true
	case "action:post":
		s.postStockTxn(c)
		return true
	}
	return true
}

// ---------- assemble / price ----------

func (s *Services) handleAssembleSplits(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_assemble_split`)
	case "get":
		api.OK(c, s.loadAssemble(paramID(c)))
		return true
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("AS%s", time.Now().Format("060102150405")))
		bizType := strOrDef(body["biz_type"], "assemble")
		wh, _ := asInt64(body["warehouse_id"])
		if wh == 0 {
			wh = 1
		}
		res, err := s.DB.Exec(`INSERT INTO inv_assemble_split(doc_no, biz_type, warehouse_id, status, remark) VALUES(?,?,?,'draft',?)`,
			docNo, bizType, wh, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		parentID, _ := asInt64(body["parent_product_id"])
		parentQty, _ := asFloat(body["parent_qty"])
		if parentID == 0 {
			parentID, _ = asInt64(body["product_id"])
			parentQty, _ = asFloat(body["qty"])
		}
		if parentQty == 0 {
			parentQty = 1
		}
		if parentID > 0 {
			_, _ = s.DB.Exec(`INSERT INTO inv_assemble_split_line(header_id, role_type, product_id, qty) VALUES(?,?,?,?)`,
				id, "parent", parentID, parentQty)
		}
		childID, _ := asInt64(body["child_product_id"])
		childQty, _ := asFloat(body["child_qty"])
		if childID > 0 {
			if childQty == 0 {
				childQty = 1
			}
			_, _ = s.DB.Exec(`INSERT INTO inv_assemble_split_line(header_id, role_type, product_id, qty) VALUES(?,?,?,?)`,
				id, "child", childID, childQty)
		}
		if lines, ok := body["lines"].([]interface{}); ok {
			for _, ln := range lines {
				m, _ := ln.(map[string]interface{})
				if m == nil {
					continue
				}
				pid, _ := asInt64(m["product_id"])
				qty, _ := asFloat(m["qty"])
				role := strOrDef(m["role_type"], "child")
				_, _ = s.DB.Exec(`INSERT INTO inv_assemble_split_line(header_id, role_type, product_id, qty) VALUES(?,?,?,?)`,
					id, role, pid, qty)
			}
		}
		api.OK(c, s.loadAssemble(id))
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_assemble_split SET remark=COALESCE(NULLIF(?,''),remark) WHERE id=? AND status='draft'`,
			strOr(body["remark"]), id)
		api.OK(c, s.loadAssemble(id))
		return true
	case "action:post":
		return s.postAssemble(c)
	}
	return true
}

func (s *Services) loadAssemble(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM inv_assemble_split WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM inv_assemble_split_line WHERE header_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) postAssemble(c *gin.Context) bool {
	id := paramID(c)
	doc := s.loadAssemble(id)
	if doc == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if st, _ := doc["status"].(string); st == "posted" {
		api.OK(c, doc)
		return true
	}
	wh, _ := asInt64(doc["warehouse_id"])
	bizType := strOrDef(doc["biz_type"], "assemble")
	bizDate := time.Now().Format("2006-01-02")
	docNo := strOrDef(doc["doc_no"], fmt.Sprintf("AS%d", id))
	lrows, _ := s.DB.Query(`SELECT role_type, product_id, qty FROM inv_assemble_split_line WHERE header_id=?`, id)
	var lines []txnLine
	if lrows != nil {
		for lrows.Next() {
			var role string
			var pid int64
			var qty float64
			_ = lrows.Scan(&role, &pid, &qty)
			dir := "out"
			// assemble: consume children(out), produce parent(in)
			// split: consume parent(out), produce children(in)
			if bizType == "assemble" {
				if role == "parent" {
					dir = "in"
				} else {
					dir = "out"
				}
			} else {
				if role == "parent" {
					dir = "out"
				} else {
					dir = "in"
				}
			}
			lines = append(lines, txnLine{pid: pid, qty: qty, dir: dir})
		}
		lrows.Close()
	}
	if _, err := s.insertPostedStockTxn(bizType, wh, bizDate, docNo, lines, fmt.Sprintf("assemble:%d", id)); err != nil {
		api.FailJSON(c, "POST_ERROR:"+err.Error())
		return true
	}
	_, _ = s.DB.Exec(`UPDATE inv_assemble_split SET status='posted' WHERE id=?`, id)
	api.OK(c, s.loadAssemble(id))
	return true
}

func (s *Services) handlePriceAdjusts(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_price_adjust`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM inv_price_adjust WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("PA%s", time.Now().Format("060102150405")))
		pid, _ := asInt64(body["product_id"])
		if pid == 0 {
			api.FailJSON(c, "PRODUCT_REQUIRED")
			return true
		}
		oldP, _ := asFloat(body["old_price"])
		newP, _ := asFloat(body["new_price"])
		eff := strOrDef(body["effective_at"], time.Now().Format("2006-01-02 15:04:05"))
		res, err := s.DB.Exec(`INSERT INTO inv_price_adjust(doc_no, product_id, old_price, new_price, effective_at, status, remark)
			VALUES(?,?,?,?,?,'draft',?)`, docNo, pid, oldP, newP, eff, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		return s.getSimpleRow(c, `SELECT * FROM inv_price_adjust WHERE id=?`, id)
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_price_adjust SET new_price=COALESCE(?,new_price), remark=COALESCE(NULLIF(?,''),remark)
			WHERE id=? AND status='draft'`, nullFloat(body["new_price"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM inv_price_adjust WHERE id=?`, id)
	}
	return true
}

// ---------- peel / material-to-payable / purchase-returns view ----------

func (s *Services) handleSalesPeelReturns(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_sales_peel_return`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM inv_sales_peel_return WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("PL%s", time.Now().Format("060102150405")))
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["peel_qty"])
		if qty == 0 {
			qty, _ = asFloat(body["qty"])
		}
		if pid == 0 || qty == 0 {
			api.FailJSON(c, "PRODUCT_QTY_REQUIRED")
			return true
		}
		wh, _ := asInt64(body["warehouse_id"])
		if wh == 0 {
			wh = 1
		}
		res, err := s.DB.Exec(`INSERT INTO inv_sales_peel_return(doc_no, sales_order_id, product_id, peel_qty, weight, warehouse_id, status, remark)
			VALUES(?,?,?,?,?,?,'draft',?)`, docNo, nullInt64Or(body["sales_order_id"]), pid, qty, nullFloat(body["weight"]), wh, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		return s.getSimpleRow(c, `SELECT * FROM inv_sales_peel_return WHERE id=?`, id)
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_sales_peel_return SET peel_qty=COALESCE(?,peel_qty), remark=COALESCE(NULLIF(?,''),remark)
			WHERE id=? AND status='draft'`, nullFloat(body["peel_qty"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM inv_sales_peel_return WHERE id=?`, id)
	case "action:post":
		id := paramID(c)
		var pid, wh int64
		var qty float64
		var status string
		err := s.DB.QueryRow(`SELECT product_id, peel_qty, COALESCE(warehouse_id,1), status FROM inv_sales_peel_return WHERE id=?`, id).
			Scan(&pid, &qty, &wh, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "posted" {
			return s.getSimpleRow(c, `SELECT * FROM inv_sales_peel_return WHERE id=?`, id)
		}
		docNo := fmt.Sprintf("PL%d", id)
		if _, err := s.insertPostedStockTxn("sales_return", wh, time.Now().Format("2006-01-02"), docNo,
			[]txnLine{{pid: pid, qty: qty, dir: "in"}}, fmt.Sprintf("peel:%d", id)); err != nil {
			api.FailJSON(c, "POST_ERROR:"+err.Error())
			return true
		}
		_, _ = s.DB.Exec(`UPDATE inv_sales_peel_return SET status='posted' WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM inv_sales_peel_return WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleMaterialToPayables(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM inv_material_to_payable`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM inv_material_to_payable WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("MTP%s", time.Now().Format("060102150405")))
		amt, _ := asFloat(body["amount"])
		res, err := s.DB.Exec(`INSERT INTO inv_material_to_payable(doc_no, consume_txn_id, supplier_id, product_id, qty, amount, status, remark)
			VALUES(?,?,?,?,?,?,'draft',?)`,
			docNo, nullInt64Or(body["consume_txn_id"]), nullInt64Or(body["supplier_id"]),
			nullInt64Or(body["product_id"]), nullFloat(body["qty"]), amt, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		return s.getSimpleRow(c, `SELECT * FROM inv_material_to_payable WHERE id=?`, id)
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE inv_material_to_payable SET amount=COALESCE(?,amount), remark=COALESCE(NULLIF(?,''),remark)
			WHERE id=? AND status='draft'`, nullFloat(body["amount"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM inv_material_to_payable WHERE id=?`, id)
	case "action:submit":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE inv_material_to_payable SET status='submitted' WHERE id=? AND status='draft'`, id)
		return s.getSimpleRow(c, `SELECT * FROM inv_material_to_payable WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleInventoryPurchaseReturns(c *gin.Context, action string) bool {
	// 库存菜单「采购退货」复用采购真表，避免双套空壳
	if action == "list" || action == "get" {
		return s.handlePurchaseReturns(c, "GET", action)
	}
	api.OK(c, gin.H{"list": []gin.H{}, "hint": "请到采购管理/采购退货办理退货过账"})
	return true
}
