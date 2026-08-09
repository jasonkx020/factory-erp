package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

// EnsureProductionExtSchema creates factory-delivery production satellite tables (SQLite).
func EnsureProductionExtSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pd_bom (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL,
  product_id INTEGER NOT NULL,
  version_no TEXT NOT NULL DEFAULT 'V1',
  name TEXT,
  is_auto_generated INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(code, version_no)
)`,
		`CREATE TABLE IF NOT EXISTS pd_bom_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bom_id INTEGER NOT NULL,
  component_product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  scrap_rate REAL NOT NULL DEFAULT 0,
  remark TEXT
)`,
		`CREATE TABLE IF NOT EXISTS pd_task_merge (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  merge_no TEXT NOT NULL UNIQUE,
  title TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  result_task_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_task_merge_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  merge_id INTEGER NOT NULL,
  source_doc_type TEXT NOT NULL DEFAULT 'production_task',
  source_doc_id INTEGER NOT NULL,
  task_id INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS pd_qc_order (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  qc_type TEXT NOT NULL DEFAULT 'process',
  source_doc_type TEXT,
  source_doc_id INTEGER,
  product_id INTEGER,
  process_id INTEGER,
  qty REAL NOT NULL DEFAULT 0,
  result TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_rework_order (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  source_qc_id INTEGER,
  task_id INTEGER,
  process_id INTEGER,
  qty REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_drawing_link (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  drawing_code TEXT,
  drawing_name TEXT,
  drawing_id INTEGER,
  task_id INTEGER,
  work_order_id INTEGER,
  process_id INTEGER,
  file_url TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_cost_hide_policy (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id INTEGER NOT NULL,
  name TEXT,
  field_scope TEXT NOT NULL DEFAULT '[]',
  is_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(role_id)
)`,
		`CREATE TABLE IF NOT EXISTS pd_outsource_order (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  supplier_id INTEGER,
  process_id INTEGER,
  product_id INTEGER,
  qty REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  received_qty REAL NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_consignment_order (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER,
  product_id INTEGER,
  qty REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  progress TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_mrp_run (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_no TEXT NOT NULL UNIQUE,
  run_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'done',
  params_json TEXT,
  remark TEXT
)`,
		`CREATE TABLE IF NOT EXISTS pd_mrp_result (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  demand_qty REAL NOT NULL DEFAULT 0,
  supply_qty REAL NOT NULL DEFAULT 0,
  shortage_qty REAL NOT NULL DEFAULT 0,
  suggest_action TEXT
)`,
		`CREATE TABLE IF NOT EXISTS pd_workshop (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`ALTER TABLE pd_scrap_record ADD COLUMN scrap_type TEXT`,
		`ALTER TABLE pd_scrap_record ADD COLUMN remark TEXT`,
		`ALTER TABLE pd_dispatch ADD COLUMN dispatch_type TEXT DEFAULT 'normal'`,
	}
	execSchemaRuns(db, "production-ext", stmts)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pd_workshop(id, code, name, status, remark) VALUES
 (1, 'WS-MAIN', '主车间', 'active', '木薯粗加工主车间')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pd_bom(id, code, product_id, version_no, name, status) VALUES
 (1, 'BOM-CASSAVA-DICE', 3, 'V1', '袋装木薯丁BOM', 'active')`)
	var bn int
	_ = db.QueryRow(`SELECT COUNT(1) FROM pd_bom_line WHERE bom_id=1`).Scan(&bn)
	if bn == 0 {
		_, _ = db.Exec(`INSERT INTO pd_bom_line(bom_id, component_product_id, qty, scrap_rate) VALUES (1, 1, 1.2, 0.05)`)
	}
}

func (s *Services) handleProductionExt(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/production/task-merges"):
		return s.handleTaskMerges(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/boms"):
		return s.handleProductionBOMs(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/mrp-runs"):
		return s.handleMRPRuns(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/qc-orders"):
		return s.handleQCOrders(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/reworks"):
		return s.handleReworks(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/scraps"):
		return s.handleScraps(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/drawing-links"):
		return s.handleDrawingLinks(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/cost-hide-policies"):
		return s.handleCostHidePolicies(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/outsources"):
		return s.handleOutsources(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/consignments"):
		if strings.Contains(openapiPath, "/progress") && method == "GET" {
			id := paramID(c)
			return s.getSimpleRow(c, `SELECT id, doc_no, progress, status FROM pd_consignment_order WHERE id=?`, id)
		}
		return s.handleConsignments(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/flex-dispatches"):
		return s.handleFlexDispatches(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/shifts"):
		return s.handleProductionShifts(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/workshop-workbench"):
		return s.handleWorkshopWorkbench(c, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/production/progress"):
		return s.handleProductionProgress(c)
	case strings.HasPrefix(openapiPath, "/api/v1/production/process-wip"):
		return s.handleProcessWip(c, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/production/workshops"):
		return s.handleWorkshopsCRUD(c, method, action)
	default:
		return false
	}
}

// ---------- workshops ----------

func (s *Services) handleWorkshopsCRUD(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_workshop WHERE COALESCE(is_deleted,0)=0`)
	case "create":
		body := bindBody(c)
		code := strOrDef(body["code"], fmt.Sprintf("WS%s", time.Now().Format("060102150405")))
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO pd_workshop(code, name, status, remark) VALUES(?,?,?,?)`,
			code, name, strOrDef(body["status"], "active"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "status": "active"})
		return true
	case "get":
		id := paramID(c)
		rows, _ := s.DB.Query(`SELECT * FROM pd_workshop WHERE id=?`, id)
		if rows == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, list[0])
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_workshop SET name=COALESCE(NULLIF(?,''),name), status=COALESCE(NULLIF(?,''),status),
			remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
			strOr(body["name"]), strOr(body["status"]), strOr(body["remark"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		_, _ = s.DB.Exec(`UPDATE pd_workshop SET is_deleted=1, status='inactive' WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	_ = method
	return true
}

// ---------- task merges ----------

func (s *Services) handleTaskMerges(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_task_merge`)
	case "create":
		body := bindBody(c)
		no := strOrDef(body["merge_no"], fmt.Sprintf("MG%s", time.Now().Format("060102150405")))
		res, err := s.DB.Exec(`INSERT INTO pd_task_merge(merge_no, title, status, remark) VALUES(?,?, 'draft',?)`,
			no, strOrDef(body["title"], "多单整合"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		mid, _ := res.LastInsertId()
		raw, _ := body["task_ids"].([]interface{})
		for _, x := range raw {
			tid, ok := asInt64(x)
			if !ok || tid == 0 {
				continue
			}
			_, _ = s.DB.Exec(`INSERT INTO pd_task_merge_line(merge_id, source_doc_type, source_doc_id, task_id) VALUES(?,?,?,?)`,
				mid, "production_task", tid, tid)
		}
		api.OK(c, s.loadTaskMerge(mid))
		return true
	case "get":
		m := s.loadTaskMerge(paramID(c))
		if m == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case "action:confirm":
		id := paramID(c)
		m := s.loadTaskMerge(id)
		if m == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		docNo := fmt.Sprintf("PTM%s", time.Now().Format("060102150405"))
		tres, err := s.DB.Exec(`INSERT INTO pd_production_task(doc_no, source_type, status, remark) VALUES(?,'merge','pending',?)`,
			docNo, fmt.Sprintf("整合自 %s", strOr(m["merge_no"])))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		tid, _ := tres.LastInsertId()
		lines, _ := m["lines"].([]map[string]interface{})
		for _, ln := range lines {
			srcTID := asInt64Or0(ln["task_id"])
			if srcTID == 0 {
				continue
			}
			rows, _ := s.DB.Query(`SELECT product_id, plan_qty FROM pd_production_task_item WHERE task_id=?`, srcTID)
			if rows != nil {
				for rows.Next() {
					var pid int64
					var qty float64
					_ = rows.Scan(&pid, &qty)
					_, _ = s.DB.Exec(`INSERT INTO pd_production_task_item(task_id, product_id, plan_qty) VALUES(?,?,?)`, tid, pid, qty)
				}
				rows.Close()
			}
			_, _ = s.DB.Exec(`UPDATE pd_production_task SET status='merged' WHERE id=?`, srcTID)
		}
		_, _ = s.DB.Exec(`UPDATE pd_task_merge SET status='confirmed', result_task_id=? WHERE id=?`, tid, id)
		out := s.loadTaskMerge(id)
		out["result_task"] = gin.H{"id": tid, "doc_no": docNo}
		api.OK(c, out)
		return true
	}
	_ = method
	return true
}

func (s *Services) loadTaskMerge(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pd_task_merge WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM pd_task_merge_line WHERE merge_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

// ---------- BOM ----------

func (s *Services) handleProductionBOMs(c *gin.Context, method, path, action string) bool {
	if strings.Contains(path, "/generate") || (action == "create" && strings.Contains(path, "generate")) {
		return s.generateBOM(c)
	}
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_bom`)
	case "create":
		body := bindBody(c)
		pid, _ := asInt64(body["product_id"])
		if pid == 0 {
			pid = 3
		}
		code := strOrDef(body["code"], fmt.Sprintf("BOM%s", time.Now().Format("060102150405")))
		res, err := s.DB.Exec(`INSERT INTO pd_bom(code, product_id, version_no, name, status, remark) VALUES(?,?,?,?,?,?)`,
			code, pid, strOrDef(body["version_no"], "V1"), strOrDef(body["name"], "生产BOM"),
			strOrDef(body["status"], "active"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		hid, _ := res.LastInsertId()
		s.insertBOMLines(hid, body)
		api.OK(c, s.loadBOM(hid))
		return true
	case "get":
		m := s.loadBOM(paramID(c))
		if m == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_bom SET name=COALESCE(NULLIF(?,''),name), status=COALESCE(NULLIF(?,''),status),
			remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
			strOr(body["name"]), strOr(body["status"]), strOr(body["remark"]), id)
		if body["lines"] != nil || body["component_product_id"] != nil {
			_, _ = s.DB.Exec(`DELETE FROM pd_bom_line WHERE bom_id=?`, id)
			s.insertBOMLines(id, body)
		}
		api.OK(c, s.loadBOM(id))
		return true
	case "delete":
		_, _ = s.DB.Exec(`UPDATE pd_bom SET status='inactive' WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	_ = method
	return true
}

func (s *Services) insertBOMLines(hid int64, body map[string]interface{}) {
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		cid, _ := asInt64(body["component_product_id"])
		qty, _ := asFloat(body["qty"])
		if cid == 0 {
			cid = 1
		}
		if qty <= 0 {
			qty = 1
		}
		_, _ = s.DB.Exec(`INSERT INTO pd_bom_line(bom_id, component_product_id, qty, scrap_rate) VALUES(?,?,?,?)`,
			hid, cid, qty, nullFloat(body["scrap_rate"]))
		return
	}
	for _, ln := range lines {
		m, _ := ln.(map[string]interface{})
		if m == nil {
			continue
		}
		cid, _ := asInt64(m["component_product_id"])
		qty, _ := asFloat(m["qty"])
		_, _ = s.DB.Exec(`INSERT INTO pd_bom_line(bom_id, component_product_id, qty, scrap_rate, remark) VALUES(?,?,?,?,?)`,
			hid, cid, qty, nullFloat(m["scrap_rate"]), strOr(m["remark"]))
	}
}

func (s *Services) loadBOM(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pd_bom WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM pd_bom_line WHERE bom_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) generateBOM(c *gin.Context) bool {
	body := bindBody(c)
	pid, _ := asInt64(body["product_id"])
	if pid == 0 {
		pid = 3
	}
	code := fmt.Sprintf("BOM-AUTO-%d-%s", pid, time.Now().Format("060102150405"))
	res, err := s.DB.Exec(`INSERT INTO pd_bom(code, product_id, version_no, name, is_auto_generated, status) VALUES(?,?,?,?,1,'active')`,
		code, pid, "V1", fmt.Sprintf("自动BOM-产品%d", pid))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	hid, _ := res.LastInsertId()
	// default: raw material product 1 with 1.2 factor
	_, _ = s.DB.Exec(`INSERT INTO pd_bom_line(bom_id, component_product_id, qty, scrap_rate) VALUES(?,?,?,?)`, hid, 1, 1.2, 0.05)
	api.OK(c, s.loadBOM(hid))
	return true
}

// ---------- MRP ----------

func (s *Services) handleMRPRuns(c *gin.Context, method, path, action string) bool {
	if strings.Contains(path, "/results") {
		id := paramID(c)
		rows, err := s.DB.Query(`SELECT * FROM pd_mrp_result WHERE run_id=? ORDER BY id`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		api.OK(c, gin.H{"list": list, "run_id": id})
		return true
	}
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_mrp_run`)
	case "create":
		body := bindBody(c)
		runNo := strOrDef(body["run_no"], fmt.Sprintf("MRP%s", time.Now().Format("060102150405")))
		now := time.Now().Format("2006-01-02 15:04:05")
		params, _ := json.Marshal(body)
		res, err := s.DB.Exec(`INSERT INTO pd_mrp_run(run_no, run_at, status, params_json, remark) VALUES(?,?,?,?,?)`,
			runNo, now, "done", string(params), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		rid, _ := res.LastInsertId()
		// simple MRP: open task demand vs inv balance
		type dem struct {
			pid int64
			qty float64
		}
		demands := map[int64]float64{}
		rows, _ := s.DB.Query(`SELECT i.product_id, SUM(i.plan_qty-COALESCE(i.completed_qty,0))
			FROM pd_production_task_item i
			JOIN pd_production_task t ON t.id=i.task_id
			WHERE COALESCE(t.is_deleted,0)=0 AND t.status IN ('pending','released','in_progress')
			GROUP BY i.product_id`)
		if rows != nil {
			for rows.Next() {
				var pid int64
				var qty float64
				_ = rows.Scan(&pid, &qty)
				demands[pid] = qty
			}
			rows.Close()
		}
		// also explode BOM for finished goods
		for pid, dq := range demands {
			brows, _ := s.DB.Query(`SELECT l.component_product_id, l.qty*(1+COALESCE(l.scrap_rate,0))
				FROM pd_bom_line l JOIN pd_bom b ON b.id=l.bom_id
				WHERE b.product_id=? AND b.status='active' ORDER BY b.id DESC LIMIT 20`, pid)
			if brows != nil {
				for brows.Next() {
					var cid int64
					var factor float64
					_ = brows.Scan(&cid, &factor)
					demands[cid] += dq * factor
				}
				brows.Close()
			}
		}
		results := []gin.H{}
		for pid, demand := range demands {
			var supply float64
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM inv_balance WHERE product_id=?`, pid).Scan(&supply)
			short := demand - supply
			actionSuggest := "ok"
			if short > 0 {
				actionSuggest = "purchase"
			}
			_, _ = s.DB.Exec(`INSERT INTO pd_mrp_result(run_id, product_id, demand_qty, supply_qty, shortage_qty, suggest_action) VALUES(?,?,?,?,?,?)`,
				rid, pid, demand, supply, short, actionSuggest)
			results = append(results, gin.H{
				"product_id": pid, "demand_qty": demand, "supply_qty": supply,
				"shortage_qty": short, "suggest_action": actionSuggest,
			})
		}
		api.OK(c, gin.H{"id": rid, "run_no": runNo, "run_at": now, "status": "done", "results": results})
		return true
	}
	_ = method
	return true
}

// ---------- QC / rework / scrap ----------

func (s *Services) handleQCOrders(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_qc_order`)
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("QC%s", time.Now().Format("060102150405")))
		res, err := s.DB.Exec(`INSERT INTO pd_qc_order(doc_no, qc_type, source_doc_type, source_doc_id, product_id, process_id, qty, status, remark)
			VALUES(?,?,?,?,?,?,?,'draft',?)`,
			docNo, strOrDef(body["qc_type"], "process"), strOr(body["source_doc_type"]), nullInt(body["source_doc_id"]),
			nullInt(body["product_id"]), nullInt(body["process_id"]), nullFloat(body["qty"]), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_qc_order WHERE id=?`, paramID(c))
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_qc_order SET result=COALESCE(NULLIF(?,''),result), remark=COALESCE(NULLIF(?,''),remark),
			qty=COALESCE(?,qty) WHERE id=?`, strOr(body["result"]), strOr(body["remark"]), nullFloat(body["qty"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case "action:complete":
		id := paramID(c)
		body := bindBody(c)
		result := strOrDef(body["result"], "pass")
		_, _ = s.DB.Exec(`UPDATE pd_qc_order SET status='completed', result=? WHERE id=?`, result, id)
		api.OK(c, gin.H{"id": id, "status": "completed", "result": result})
		return true
	}
	_ = method
	return true
}

func (s *Services) handleReworks(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_rework_order`)
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("RW%s", time.Now().Format("060102150405")))
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			qty = 1
		}
		res, err := s.DB.Exec(`INSERT INTO pd_rework_order(doc_no, source_qc_id, task_id, process_id, qty, status, remark)
			VALUES(?,?,?,?,?,'draft',?)`,
			docNo, nullInt(body["source_qc_id"]), nullInt(body["task_id"]), nullInt(body["process_id"]), qty, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_rework_order WHERE id=?`, paramID(c))
	case "action:close":
		_, _ = s.DB.Exec(`UPDATE pd_rework_order SET status='closed' WHERE id=?`, paramID(c))
		api.OK(c, gin.H{"id": paramID(c), "status": "closed"})
		return true
	}
	_ = method
	return true
}

func (s *Services) handleScraps(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_scrap_record`)
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("SC%s", time.Now().Format("060102150405")))
		pid, _ := asInt64(body["product_id"])
		if pid == 0 {
			pid = 1
		}
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			qty, _ = asFloat(body["weight"])
		}
		res, err := s.DB.Exec(`INSERT INTO pd_scrap_record(doc_no, task_id, process_id, product_id, qty, weight, disposition, scrap_type, status, remark)
			VALUES(?,?,?,?,?,?,?,?,'draft',?)`,
			docNo, nullInt(body["task_id"]), nullInt(body["process_id"]), pid, qty, nullFloat(body["weight"]),
			strOr(body["disposition"]), strOr(body["scrap_type"]), strOr(body["remark"]))
		if err != nil {
			// fallback without scrap_type/remark
			res, err = s.DB.Exec(`INSERT INTO pd_scrap_record(doc_no, task_id, process_id, product_id, qty, weight, disposition, status)
				VALUES(?,?,?,?,?,?,?,'draft')`,
				docNo, nullInt(body["task_id"]), nullInt(body["process_id"]), pid, qty, nullFloat(body["weight"]), strOr(body["disposition"]))
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_scrap_record WHERE id=?`, paramID(c))
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_scrap_record SET qty=COALESCE(?,qty), weight=COALESCE(?,weight),
			disposition=COALESCE(NULLIF(?,''),disposition), status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			nullFloat(body["qty"]), nullFloat(body["weight"]), strOr(body["disposition"]), strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	return true
}

// ---------- drawings / cost hide / outsource / consignment ----------

func (s *Services) handleDrawingLinks(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_drawing_link`)
	case "create":
		body := bindBody(c)
		res, err := s.DB.Exec(`INSERT INTO pd_drawing_link(drawing_code, drawing_name, drawing_id, task_id, work_order_id, process_id, file_url, status)
			VALUES(?,?,?,?,?,?,?,?)`,
			strOrDef(body["drawing_code"], strOr(body["code"])), strOrDef(body["drawing_name"], strOr(body["name"])),
			nullInt(body["drawing_id"]), nullInt(body["task_id"]), nullInt(body["work_order_id"]),
			nullInt(body["process_id"]), strOr(body["file_url"]), strOrDef(body["status"], "active"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_drawing_link WHERE id=?`, paramID(c))
	}
	_ = method
	return true
}

func (s *Services) handleCostHidePolicies(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_cost_hide_policy`)
	case "create":
		body := bindBody(c)
		rid, _ := asInt64(body["role_id"])
		if rid == 0 {
			api.FailJSON(c, "ROLE_REQUIRED")
			return true
		}
		scope := strOr(body["field_scope"])
		if scope == "" {
			if raw, ok := body["fields"]; ok {
				b, _ := json.Marshal(raw)
				scope = string(b)
			} else {
				scope = `["unit_cost","material_cost","labor_cost"]`
			}
		}
		res, err := s.DB.Exec(`INSERT INTO pd_cost_hide_policy(role_id, name, field_scope, is_enabled) VALUES(?,?,?,?)`,
			rid, strOrDef(body["name"], "成本隐藏"), scope, 1)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "role_id": rid})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_cost_hide_policy WHERE id=?`, paramID(c))
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_cost_hide_policy SET name=COALESCE(NULLIF(?,''),name), field_scope=COALESCE(NULLIF(?,''),field_scope),
			is_enabled=COALESCE(?,is_enabled), updated_at=datetime('now') WHERE id=?`,
			strOr(body["name"]), strOr(body["field_scope"]), nullInt(body["is_enabled"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	return true
}

func (s *Services) handleOutsources(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_outsource_order`)
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("OS%s", time.Now().Format("060102150405")))
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			qty = 1
		}
		res, err := s.DB.Exec(`INSERT INTO pd_outsource_order(doc_no, supplier_id, process_id, product_id, qty, status, remark)
			VALUES(?,?,?,?,?,'draft',?)`,
			docNo, nullInt(body["supplier_id"]), nullInt(body["process_id"]), nullInt(body["product_id"]), qty, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_outsource_order WHERE id=?`, paramID(c))
	case "action:receive":
		id := paramID(c)
		body := bindBody(c)
		qty, _ := asFloat(body["qty"])
		_, _ = s.DB.Exec(`UPDATE pd_outsource_order SET received_qty=COALESCE(received_qty,0)+?, status='received' WHERE id=?`, qty, id)
		api.OK(c, gin.H{"id": id, "status": "received"})
		return true
	}
	_ = method
	return true
}

func (s *Services) handleConsignments(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_consignment_order`)
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("CS%s", time.Now().Format("060102150405")))
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			qty = 1
		}
		res, err := s.DB.Exec(`INSERT INTO pd_consignment_order(doc_no, customer_id, product_id, qty, status, progress, remark)
			VALUES(?,?,?,?,'draft',?,?)`,
			docNo, nullInt(body["customer_id"]), nullInt(body["product_id"]), qty,
			strOrDef(body["progress"], "待投产"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_consignment_order WHERE id=?`, paramID(c))
	case "action:progress":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_consignment_order SET progress=?, status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			strOr(body["progress"]), strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	return true
}

func (s *Services) handleFlexDispatches(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_dispatch WHERE COALESCE(dispatch_type,'normal')='flex'`)
	case "create":
		body := bindBody(c)
		body["dispatch_type"] = "flex"
		taskID, _ := asInt64(body["task_id"])
		procID, _ := asInt64(body["process_id"])
		workerID, _ := asInt64(body["worker_id"])
		qty, _ := asFloat(body["qty"])
		docNo := fmt.Sprintf("FD%s", time.Now().Format("060102150405"))
		woNo := fmt.Sprintf("WO%s", time.Now().Format("060102150405"))
		woRes, err := s.DB.Exec(`INSERT INTO pd_work_order(doc_no, task_id, process_id, status, plan_qty) VALUES(?,?,?,'pending',?)`,
			woNo, taskID, procID, qty)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		woID, _ := woRes.LastInsertId()
		_, err = s.DB.Exec(`INSERT INTO pd_dispatch(doc_no, work_order_id, dispatch_type, worker_id, plan_qty, status, dispatched_at)
			VALUES(?,?,?,?,?,'dispatched',datetime('now'))`, docNo, woID, "flex", workerID, qty)
		if err != nil {
			_, err = s.DB.Exec(`INSERT INTO pd_dispatch(doc_no, work_order_id, worker_id, plan_qty, status) VALUES(?,?,?,?,'dispatched')`,
				docNo, woID, workerID, qty)
		}
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"doc_no": docNo, "work_order_id": woID, "dispatch_type": "flex", "status": "dispatched"})
		return true
	case "action:reassign":
		id := paramID(c)
		body := bindBody(c)
		wid, _ := asInt64(body["worker_id"])
		_, _ = s.DB.Exec(`UPDATE pd_dispatch SET worker_id=?, status='reassigned' WHERE id=?`, wid, id)
		api.OK(c, gin.H{"id": id, "worker_id": wid, "status": "reassigned"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_dispatch WHERE id=?`, paramID(c))
	}
	_ = method
	return true
}

// ---------- workbench / progress ----------

func (s *Services) handleWorkshopWorkbench(c *gin.Context, path string) bool {
	if strings.Contains(path, "today-tasks") {
		rows, err := s.DB.Query(`SELECT id, doc_no, status, created_at FROM pd_production_task
			WHERE COALESCE(is_deleted,0)=0 AND status IN ('pending','released','in_progress') ORDER BY id DESC LIMIT 50`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var docNo, status, created string
			_ = rows.Scan(&id, &docNo, &status, &created)
			list = append(list, gin.H{"id": id, "doc_no": docNo, "status": status, "created_at": created})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	}
	// overview — 过站导向 KPI（派工仅作例外参考）
	var tasksOpen, stationToday, pendingConfirm, flowFail, openShifts int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_production_task WHERE COALESCE(is_deleted,0)=0 AND status IN ('pending','released','in_progress')`).Scan(&tasksOpen)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_report_work WHERE status='posted' AND (date(reported_at)=date('now') OR date(created_at)=date('now'))`).Scan(&stationToday)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_report_work WHERE status='confirm_pending'`).Scan(&pendingConfirm)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_flow_event WHERE status IN ('error','failed')`).Scan(&flowFail)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_shift WHERE status='open' AND date(biz_date)=date('now')`).Scan(&openShifts)
	var exceptionDispatches int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_dispatch WHERE status IN ('dispatched','reassigned')`).Scan(&exceptionDispatches)
	api.OK(c, gin.H{
		"open_tasks":             tasksOpen,
		"today_station_passes":   stationToday,
		"pending_confirm":        pendingConfirm,
		"failed_flow_events":     flowFail,
		"open_shifts":            openShifts,
		"exception_dispatches":   exceptionDispatches,
		"today_reports":          stationToday,
		"open_dispatches":        exceptionDispatches,
		"hint":                   "车间工作台：今日过站/待确认/流转失败/产线班次（派工数仅作例外参考）",
	})
	return true
}

func (s *Services) handleProductionProgress(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	rows, err := s.DB.Query(`SELECT t.id, t.doc_no, t.status, t.created_at,
		COALESCE((SELECT SUM(plan_qty) FROM pd_production_task_item i WHERE i.task_id=t.id),0) AS plan_qty,
		COALESCE((SELECT SUM(completed_qty) FROM pd_production_task_item i WHERE i.task_id=t.id),0) AS completed_qty
		FROM pd_production_task t WHERE COALESCE(t.is_deleted,0)=0
		ORDER BY t.id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var docNo, status, created string
		var plan, done float64
		_ = rows.Scan(&id, &docNo, &status, &created, &plan, &done)
		pct := 0.0
		if plan > 0 {
			pct = done / plan * 100
		}
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "status": status, "created_at": created,
			"plan_qty": plan, "completed_qty": done, "progress_pct": pct,
		})
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_production_task WHERE COALESCE(is_deleted,0)=0`).Scan(&total)
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) getSimpleRow(c *gin.Context, sqlStr string, id int64) bool {
	rows, err := s.DB.Query(sqlStr, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, list[0])
	return true
}
