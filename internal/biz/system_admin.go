package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/persistence/sqlutil"
)

// singleton setting keys under sys_setting.setting_key
var sysSettingKeys = map[string]string{
	"system/settings":               "base",
	"system/sales-settings":         "sales",
	"system/production-settings":    "production",
	"system/table-customs":          "table_customs",
	"system/doc-approve-switches":   "doc_approve",
	"system/doc-lock-rules":         "doc_lock",
	"system/notify-rules":           "notify",
	"system/doc-edit-rules":         "doc_edit",
	"system/doc-delete-rules":       "doc_delete",
	"system/search-configs":         "search",
	"system/finance-audit-controls": "finance_audit",
}

var sysSettingDefaults = map[string]map[string]interface{}{
	"base": {
		"company_name": "加工厂ERP", "timezone": "Asia/Shanghai", "currency": "CNY",
		"date_format": "YYYY-MM-DD", "default_page_size": 20, "enable_mqtt_notify": true,
		// gate=入厂确认后结算；box_stockin=分箱入库后按箱合计结算
		"farmer_settle_point": "gate",
	},
	"sales": {
		"default_tax_rate": 0.13, "allow_negative_stock": false, "require_pre_ship": true,
		"default_warehouse_id": 3, "price_precision": 2,
	},
	"production": {
		"auto_inbound_on_qc": true, "require_box_code": true, "default_workshop_id": 1,
		"piecework_confirm_required": true,
	},
	"table_customs": {
		"dense": false, "stripe": true, "show_id": false, "page_sizes": []interface{}{10, 20, 50, 100},
	},
	"doc_approve": {
		"sales_order": true, "purchase_inbound": true, "stock_txn": false, "payroll_sheet": true,
	},
	"doc_lock": {
		"lock_after_approve": true, "lock_after_days": 30, "allow_admin_unlock": true,
	},
	"notify": {
		"on_approve": true, "on_reject": true, "on_assign": true, "channels": []interface{}{"inbox", "mqtt"},
	},
	"doc_edit": {
		"allow_edit_draft": true, "allow_edit_after_approve": false, "track_versions": true,
	},
	"doc_delete": {
		"allow_delete_draft": true, "soft_delete_only": true, "require_reason": true,
	},
	"search": {
		"enable_advanced": true, "max_conditions": 8, "default_operator": "AND",
	},
	"finance_audit": {
		"require_finance_approve": true, "cost_visible_roles": []interface{}{"sys_admin", "finance"},
		"amount_threshold": 10000,
	},
}

// EnsureSystemAdminSchema creates system-management tables used by tablespec + settings KV.
func EnsureSystemAdminSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sys_setting (
			setting_key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			updated_at TEXT,
			updated_by INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS sys_print_template (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, name TEXT, doc_type TEXT, content TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_formula (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, name TEXT, scope TEXT, expression TEXT, remark TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_carrier (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, name TEXT, contact TEXT, phone TEXT, remark TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_approval_flow (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, name TEXT, doc_type TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_approval_flow_node (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			flow_id INTEGER NOT NULL, seq_no INTEGER DEFAULT 1, node_name TEXT,
			approver_role TEXT, approver_user_id INTEGER, require_all INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS sys_personnel_transfer (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_no TEXT, employee_id INTEGER, from_dept_id INTEGER, to_dept_id INTEGER,
			from_workshop_id INTEGER, to_workshop_id INTEGER, reason TEXT,
			status TEXT DEFAULT 'draft', effective_date TEXT, confirmed_at TEXT, created_by INTEGER,
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_batch_price_job (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_no TEXT, target_type TEXT, adjust_type TEXT, adjust_value REAL,
			scope_json TEXT, status TEXT DEFAULT 'draft', result_msg TEXT,
			created_by INTEGER, applied_at TEXT, is_deleted INTEGER DEFAULT 0, created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_batch_payroll_job (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_no TEXT, period_ym TEXT, workshop_id INTEGER, status TEXT DEFAULT 'draft',
			result_msg TEXT, created_by INTEGER, applied_at TEXT, is_deleted INTEGER DEFAULT 0, created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_reminder (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT, content TEXT, remind_at TEXT, target_user_id INTEGER, target_role TEXT,
			status TEXT DEFAULT 'open', is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_announcement (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT, content TEXT, status TEXT DEFAULT 'draft', published_at TEXT,
			created_by INTEGER, is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_memo (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT, content TEXT, owner_id INTEGER, status TEXT DEFAULT 'open',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_document (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, title TEXT, category TEXT, content TEXT, file_url TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_drawing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, title TEXT, product_id INTEGER, version_no TEXT, file_url TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_knowledge (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, title TEXT, category TEXT, content TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_course (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT, title TEXT, category TEXT, content TEXT, duration_min INTEGER, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil && !isIdempotentSchemaErr(err) {
			_ = err
		}
	}
	for k, def := range sysSettingDefaults {
		b, _ := json.Marshal(def)
		_, _ = db.Exec(`INSERT OR IGNORE INTO sys_setting(setting_key, value_json, updated_at) VALUES(?,?,?)`,
			k, string(b), time.Now().Format("2006-01-02 15:04:05"))
	}
	EnsureSystemAdminPermissions(db)
}

func (s *Services) handleSystemAdmin(c *gin.Context, method, openapiPath, resourceKey, action string) bool {
	if key, ok := sysSettingKeys[resourceKey]; ok {
		return s.handleSysSetting(c, key, method, action)
	}
	// batch job apply side-effects after create when status run
	if resourceKey == "system/batch-price-jobs" && action == "create" && method == "POST" {
		return s.createBatchPriceJob(c)
	}
	if resourceKey == "system/batch-payroll-jobs" && action == "create" && method == "POST" {
		return s.createBatchPayrollJob(c)
	}
	if resourceKey == "system/announcements" && action == "action:publish" {
		return s.publishAnnouncement(c)
	}
	if resourceKey == "system/personnel-transfers" && action == "action:confirm" {
		return s.confirmPersonnelTransfer(c)
	}
	if resourceKey == "system/approval-flows" && strings.Contains(openapiPath, "/nodes") {
		return s.handleApprovalFlowNodes(c, method, action)
	}
	return false
}

func (s *Services) handleSysSetting(c *gin.Context, key, method, action string) bool {
	switch {
	case method == "GET" && (action == "list" || action == "replace" || action == "get"):
		m := s.loadSysSetting(key)
		pageNum, pageSize := sqlutil.Page(c)
		api.PageOK(c, []map[string]interface{}{m}, 1, pageNum, pageSize)
		return true
	case method == "PUT" && (action == "replace" || action == "update" || action == "create"):
		body := bindBody(c)
		if body == nil {
			body = map[string]interface{}{}
		}
		delete(body, "id")
		delete(body, "setting_key")
		s.saveSysSetting(key, body, claimsUserID(c))
		api.OK(c, s.loadSysSetting(key))
		return true
	case method == "POST" && action == "create":
		body := bindBody(c)
		cur := s.loadSysSetting(key)
		for k, v := range body {
			cur[k] = v
		}
		s.saveSysSetting(key, cur, claimsUserID(c))
		api.OK(c, cur)
		return true
	default:
		return false
	}
}

func (s *Services) loadSysSetting(key string) map[string]interface{} {
	var raw string
	err := s.DB.QueryRow(`SELECT value_json FROM sys_setting WHERE setting_key=?`, key).Scan(&raw)
	out := map[string]interface{}{}
	if err == nil && raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	} else if def, ok := sysSettingDefaults[key]; ok {
		for k, v := range def {
			out[k] = v
		}
	}
	// merge missing defaults so new keys appear after upgrades
	if def, ok := sysSettingDefaults[key]; ok {
		for k, v := range def {
			if _, exists := out[k]; !exists {
				out[k] = v
			}
		}
	}
	out["setting_key"] = key
	out["id"] = 1
	return out
}

// farmerSettlePoint returns gate | box_stockin from system base settings.
func (s *Services) farmerSettlePoint() string {
	m := s.loadSysSetting("base")
	v := strings.ToLower(strings.TrimSpace(strOr(m["farmer_settle_point"])))
	if v == "box_stockin" {
		return "box_stockin"
	}
	return "gate"
}

func (s *Services) saveSysSetting(key string, body map[string]interface{}, userID int64) {
	clean := map[string]interface{}{}
	for k, v := range body {
		if k == "id" || k == "setting_key" {
			continue
		}
		clean[k] = v
	}
	b, _ := json.Marshal(clean)
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`UPDATE sys_setting SET value_json=?, updated_at=?, updated_by=? WHERE setting_key=?`, string(b), now, userID, key)
	if err == nil {
		n, _ := res.RowsAffected()
		if n > 0 {
			return
		}
	}
	_, _ = s.DB.Exec(`INSERT INTO sys_setting(setting_key, value_json, updated_at, updated_by) VALUES(?,?,?,?)`, key, string(b), now, userID)
}

func claimsUserID(c *gin.Context) int64 {
	if claims := middleware.Claims(c); claims != nil {
		return claims.UserID
	}
	return 0
}

func (s *Services) createBatchPriceJob(c *gin.Context) bool {
	body := bindBody(c)
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("BPJ-%d", time.Now().Unix()%1e10))
	target, _ := body["target_type"].(string)
	adjType, _ := body["adjust_type"].(string)
	adjVal, _ := asFloat(body["adjust_value"])
	scope, _ := json.Marshal(body["scope_json"])
	if body["scope_json"] == nil {
		scope, _ = json.Marshal(body)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`INSERT INTO sys_batch_price_job(doc_no, target_type, adjust_type, adjust_value, scope_json, status, result_msg, created_by, applied_at, created_at)
		VALUES(?,?,?,?,?,'done',?,?,?,?)`,
		docNo, target, adjType, adjVal, string(scope), "已记录改价任务（演示执行完成）", claimsUserID(c), now, now)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "done", "result_msg": "已记录改价任务（演示执行完成）"})
	return true
}

func (s *Services) createBatchPayrollJob(c *gin.Context) bool {
	body := bindBody(c)
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("PAY-%d", time.Now().Unix()%1e10))
	period := strOrDef(body["period_ym"], time.Now().Format("2006-01"))
	ws, _ := asInt64(body["workshop_id"])
	now := time.Now().Format("2006-01-02 15:04:05")
	var year, month int64
	fmt.Sscanf(period, "%d-%d", &year, &month)
	force, _ := body["force"].(bool)
	sheetID, sheetNo, n, errMsg := s.generatePayrollSheet(int(year), int(month), ws, claimsUserID(c), force)
	result := "已触发工资单批次生成"
	if errMsg != "" {
		result = "生成失败:" + errMsg
	} else {
		result = fmt.Sprintf("已生成工资单 %s，共 %d 人", sheetNo, n)
	}
	res, err := s.DB.Exec(`INSERT INTO sys_batch_payroll_job(doc_no, period_ym, workshop_id, status, result_msg, created_by, applied_at, created_at)
		VALUES(?,?,?,?,?,?,?,?)`, docNo, period, ws, map[bool]string{true: "done", false: "failed"}[errMsg == ""], result, claimsUserID(c), now, now)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, gin.H{"id": id, "doc_no": docNo, "period_ym": period, "status": "done", "result_msg": result, "sheet_id": sheetID, "line_count": n})
	return true
}

func (s *Services) publishAnnouncement(c *gin.Context) bool {
	id := paramID(c)
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.DB.Exec(`UPDATE sys_announcement SET status='published', published_at=? WHERE id=? AND COALESCE(is_deleted,0)=0`, now, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	api.OK(c, gin.H{"id": id, "status": "published", "published_at": now})
	return true
}

func (s *Services) confirmPersonnelTransfer(c *gin.Context) bool {
	id := paramID(c)
	now := time.Now().Format("2006-01-02 15:04:05")
	var empID, toDept, toWs int64
	_ = s.DB.QueryRow(`SELECT employee_id, COALESCE(to_dept_id,0), COALESCE(to_workshop_id,0) FROM sys_personnel_transfer WHERE id=?`, id).
		Scan(&empID, &toDept, &toWs)
	if empID > 0 {
		if toDept > 0 {
			_, _ = s.DB.Exec(`UPDATE hr_employee SET dept_id=? WHERE id=?`, toDept, empID)
		}
		if toWs > 0 {
			_, _ = s.DB.Exec(`UPDATE hr_employee SET workshop_id=? WHERE id=?`, toWs, empID)
		}
	}
	_, err := s.DB.Exec(`UPDATE sys_personnel_transfer SET status='confirmed', confirmed_at=? WHERE id=?`, now, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	api.OK(c, gin.H{"id": id, "status": "confirmed", "confirmed_at": now})
	return true
}

func (s *Services) handleApprovalFlowNodes(c *gin.Context, method, action string) bool {
	flowID := paramID(c)
	if method == "GET" {
		rows, err := s.DB.Query(`SELECT id, flow_id, seq_no, node_name, approver_role, approver_user_id, require_all FROM sys_approval_flow_node WHERE flow_id=? ORDER BY seq_no, id`, flowID)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		pageNum, pageSize := sqlutil.Page(c)
		api.PageOK(c, list, len(list), pageNum, pageSize)
		return true
	}
	if method == "PUT" && (action == "replace" || action == "update") {
		body := bindBody(c)
		nodes, _ := body["nodes"].([]interface{})
		if nodes == nil {
			nodes, _ = body["items"].([]interface{})
		}
		if nodes == nil {
			// body itself is array? unlikely with gin bind map
			if arr, ok := body["list"].([]interface{}); ok {
				nodes = arr
			}
		}
		_, _ = s.DB.Exec(`DELETE FROM sys_approval_flow_node WHERE flow_id=?`, flowID)
		for i, n := range nodes {
			m, _ := n.(map[string]interface{})
			if m == nil {
				continue
			}
			seq, _ := asInt64(m["seq_no"])
			if seq == 0 {
				seq = int64(i + 1)
			}
			name, _ := m["node_name"].(string)
			role, _ := m["approver_role"].(string)
			uid, _ := asInt64(m["approver_user_id"])
			reqAll := 0
			if v, ok := m["require_all"].(bool); ok && v {
				reqAll = 1
			}
			_, _ = s.DB.Exec(`INSERT INTO sys_approval_flow_node(flow_id, seq_no, node_name, approver_role, approver_user_id, require_all) VALUES(?,?,?,?,?,?)`,
				flowID, seq, name, role, uid, reqAll)
		}
		api.OK(c, gin.H{"flow_id": flowID, "count": len(nodes)})
		return true
	}
	return false
}
