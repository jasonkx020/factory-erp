package biz

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsureReportSchema creates report definition / widget / snapshot tables (SQLite).
func EnsureReportSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS rpt_report_definition (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  report_type TEXT,
  query_config_json TEXT,
  status TEXT NOT NULL DEFAULT 'active'
)`,
		`CREATE TABLE IF NOT EXISTS rpt_dashboard_widget (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  dashboard_key TEXT NOT NULL,
  title TEXT NOT NULL,
  metric_key TEXT,
  layout_json TEXT,
  refresh_sec INTEGER NOT NULL DEFAULT 60,
  status TEXT NOT NULL DEFAULT 'active'
)`,
		`CREATE TABLE IF NOT EXISTS rpt_report_snapshot (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  report_code TEXT NOT NULL,
  biz_date TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(report_code, biz_date)
)`,
		`CREATE TABLE IF NOT EXISTS sys_logistics_carrier (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  contact TEXT,
  status TEXT NOT NULL DEFAULT 'active'
)`,
		`CREATE TABLE IF NOT EXISTS sys_logistics_track (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  track_no TEXT NOT NULL,
  carrier_id INTEGER,
  order_id INTEGER,
  status TEXT NOT NULL DEFAULT 'in_transit',
  location TEXT,
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	}
	for _, s := range stmts {
		_, _ = db.Exec(s)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM rpt_report_definition`).Scan(&n)
	if n == 0 {
		defs := [][3]string{
			{"enterprise_overview", "企业总览", "enterprise"},
			{"daily", "日统计", "daily"},
			{"gross_profit", "毛利润", "finance"},
			{"balance_sheet", "资产负债表", "finance"},
			{"cash_flow", "现金流量表", "finance"},
			{"income_statement", "利润表", "finance"},
			{"cost_profit", "成本利润表", "finance"},
		}
		for _, d := range defs {
			_, _ = db.Exec(`INSERT OR IGNORE INTO rpt_report_definition(code, name, report_type) VALUES(?,?,?)`, d[0], d[1], d[2])
		}
	}
	_ = db.QueryRow(`SELECT COUNT(1) FROM rpt_dashboard_widget`).Scan(&n)
	if n == 0 {
		widgets := [][3]string{
			{"boss", "今日销售额", "sales_today"},
			{"boss", "在制任务", "open_tasks"},
			{"boss", "库存SKU", "stock_sku"},
			{"boss", "待审批", "pending_approvals"},
			{"production", "今日报工", "reports_today"},
			{"production", "未完成派工", "open_dispatches"},
			{"live", "流转失败", "flow_fail"},
		}
		for _, w := range widgets {
			_, _ = db.Exec(`INSERT INTO rpt_dashboard_widget(dashboard_key, title, metric_key, refresh_sec) VALUES(?,?,?,60)`,
				w[0], w[1], w[2])
		}
	}
	_ = db.QueryRow(`SELECT COUNT(1) FROM sys_logistics_carrier`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO sys_logistics_carrier(code, name, contact) VALUES('SF','顺丰速运','400-111-1111')`)
		_, _ = db.Exec(`INSERT INTO sys_logistics_carrier(code, name, contact) VALUES('YD','韵达快递','95546')`)
	}
}

func (s *Services) handleReportDomain(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/report/dashboards/boss/widgets"):
		return s.handleBossWidgets(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/report/dashboards/boss"):
		return s.reportBossDashboard(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/dashboards/production"):
		return s.reportProductionDashboard(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/dashboards/live"):
		return s.reportLiveDashboard(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/enterprise"):
		return s.reportEnterprise(c, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/report/crm-stats"):
		return s.reportCRMStats(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/daily"):
		return s.reportDaily(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/inquiry-queries"):
		return s.reportInquiries(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/follow-ups"):
		return s.reportFollowUps(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/gross-profit"):
		return s.reportGrossProfit(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/qc"):
		return s.reportQC(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/accounts"):
		return s.reportAccounts(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/stock-txns"):
		return s.reportStockTxns(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/stock-ledger"):
		return s.reportStockLedger(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/sales-weight"):
		return s.reportSalesWeight(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/product-sales"):
		return s.reportProductSales(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/logistics"):
		return s.reportLogistics(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/cost-profit"):
		return s.reportCostProfit(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/balance-sheet"):
		return s.reportBalanceSheet(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/cash-flow"):
		return s.reportCashFlow(c)
	case strings.HasPrefix(openapiPath, "/api/v1/report/income-statement"):
		return s.reportIncomeStatement(c)
	default:
		return false
	}
}

func (s *Services) queryCount(sqlStr string, args ...interface{}) int {
	var n int
	_ = s.DB.QueryRow(sqlStr, args...).Scan(&n)
	return n
}

func (s *Services) queryFloat(sqlStr string, args ...interface{}) float64 {
	var f float64
	_ = s.DB.QueryRow(sqlStr, args...).Scan(&f)
	return f
}

func reportToday() string { return time.Now().Format("2006-01-02") }

func (s *Services) reportBossDashboard(c *gin.Context) bool {
	salesToday := s.queryFloat(`SELECT COALESCE(SUM(total_amount),0) FROM sl_sales_order
		WHERE COALESCE(is_deleted,0)=0 AND date(created_at)=date('now')`)
	openOrders := s.queryCount(`SELECT COUNT(1) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0 AND status NOT IN ('closed','cancelled','shipped')`)
	openTasks := s.queryCount(`SELECT COUNT(1) FROM pd_production_task WHERE COALESCE(is_deleted,0)=0 AND status IN ('pending','released','in_progress')`)
	stockSKU := s.queryCount(`SELECT COUNT(1) FROM inv_balance WHERE qty>0`)
	fundBal := s.queryFloat(`SELECT COALESCE(SUM(balance),0) FROM fin_fund_account WHERE status='active'`)
	customers := s.queryCount(`SELECT COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0`)
	pendingAppr := s.queryCount(`SELECT COUNT(1) FROM fin_voucher WHERE status IN ('draft','submitted')`)
	reportsToday := s.queryCount(`SELECT COUNT(1) FROM pd_report_work WHERE date(COALESCE(reported_at,created_at))=date('now')`)

	kpis := []gin.H{
		{"key": "sales_today", "title": "今日销售额", "value": salesToday, "unit": "元"},
		{"key": "open_orders", "title": "在途订单", "value": openOrders},
		{"key": "open_tasks", "title": "在制任务", "value": openTasks},
		{"key": "stock_sku", "title": "有库存SKU", "value": stockSKU},
		{"key": "fund_balance", "title": "资金余额", "value": fundBal, "unit": "元"},
		{"key": "customers", "title": "客户数", "value": customers},
		{"key": "pending_approvals", "title": "待审凭证", "value": pendingAppr},
		{"key": "reports_today", "title": "今日报工", "value": reportsToday},
	}
	api.OK(c, gin.H{
		"title": "老板驾驶舱", "as_of": time.Now().Format("2006-01-02 15:04:05"),
		"kpis": kpis, "list": kpis, "total": len(kpis),
		"summary": gin.H{
			"sales_today": salesToday, "open_orders": openOrders, "open_tasks": openTasks,
			"stock_sku": stockSKU, "fund_balance": fundBal,
		},
	})
	return true
}

func (s *Services) handleBossWidgets(c *gin.Context, method, action string) bool {
	if method == "PUT" || action == "replace" {
		body := bindBody(c)
		items, _ := body["widgets"].([]interface{})
		if len(items) == 0 {
			items, _ = body["list"].([]interface{})
		}
		for _, it := range items {
			m, _ := it.(map[string]interface{})
			if m == nil {
				continue
			}
			if id, ok := asInt64(m["id"]); ok && id > 0 {
				_, _ = s.DB.Exec(`UPDATE rpt_dashboard_widget SET title=COALESCE(NULLIF(?,''),title), metric_key=COALESCE(NULLIF(?,''),metric_key),
					refresh_sec=COALESCE(NULLIF(?,0),refresh_sec), status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
					strOr(m["title"]), strOr(m["metric_key"]), nullInt64Or(m["refresh_sec"]), strOr(m["status"]), id)
			} else {
				_, _ = s.DB.Exec(`INSERT INTO rpt_dashboard_widget(dashboard_key, title, metric_key, refresh_sec) VALUES(?,?,?,?)`,
					strOrDef(m["dashboard_key"], "boss"), strOrDef(m["title"], "组件"), strOr(m["metric_key"]), 60)
			}
		}
		api.OK(c, gin.H{"ok": true})
		return true
	}
	rows, err := s.DB.Query(`SELECT id, dashboard_key, title, metric_key, COALESCE(layout_json,''), refresh_sec, status
		FROM rpt_dashboard_widget WHERE dashboard_key='boss' AND status='active' ORDER BY id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var key, title, metric, layout, status string
		var refresh int
		_ = rows.Scan(&id, &key, &title, &metric, &layout, &refresh, &status)
		list = append(list, gin.H{
			"id": id, "dashboard_key": key, "title": title, "metric_key": metric,
			"layout_json": layout, "refresh_sec": refresh, "status": status,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportProductionDashboard(c *gin.Context) bool {
	list := []gin.H{
		{"key": "open_tasks", "title": "在制任务", "value": s.queryCount(`SELECT COUNT(1) FROM pd_production_task WHERE COALESCE(is_deleted,0)=0 AND status IN ('pending','released','in_progress')`)},
		{"key": "open_dispatches", "title": "未完成派工", "value": s.queryCount(`SELECT COUNT(1) FROM pd_dispatch WHERE status IN ('dispatched','reassigned')`)},
		{"key": "reports_today", "title": "今日报工单", "value": s.queryCount(`SELECT COUNT(1) FROM pd_report_work WHERE date(COALESCE(reported_at,created_at))=date('now')`)},
		{"key": "qty_today", "title": "今日报工量", "value": s.queryFloat(`SELECT COALESCE(SUM(qty),0) FROM pd_report_work WHERE date(COALESCE(reported_at,created_at))=date('now')`)},
		{"key": "qc_pending", "title": "待质检", "value": s.queryCount(`SELECT COUNT(1) FROM pd_qc_order WHERE status IN ('draft','pending')`)},
	}
	api.OK(c, gin.H{"title": "生产看板", "list": list, "total": len(list), "as_of": time.Now().Format(time.RFC3339)})
	return true
}

func (s *Services) reportLiveDashboard(c *gin.Context) bool {
	list := []gin.H{
		{"key": "flow_fail", "title": "流转失败", "value": s.queryCount(`SELECT COUNT(1) FROM pd_flow_event WHERE status IN ('error','failed')`)},
		{"key": "reports_1h", "title": "近1小时报工", "value": s.queryCount(`SELECT COUNT(1) FROM pd_report_work WHERE datetime(COALESCE(reported_at,created_at))>=datetime('now','-1 hour')`)},
		{"key": "open_dispatches", "title": "待接收派工", "value": s.queryCount(`SELECT COUNT(1) FROM pd_dispatch WHERE status IN ('dispatched','reassigned')`)},
	}
	api.OK(c, gin.H{"title": "生产实况", "list": list, "total": len(list), "as_of": time.Now().Format(time.RFC3339)})
	return true
}

func (s *Services) reportEnterprise(c *gin.Context, openapiPath, action string) bool {
	if action == "get" || strings.Contains(openapiPath, "{code}") {
		code := c.Param("code")
		if code == "" {
			code = "enterprise_overview"
		}
		payload := s.buildEnterprisePayload()
		rows, _ := s.DB.Query(`SELECT * FROM rpt_report_definition WHERE code=?`, code)
		var def interface{}
		if rows != nil {
			defer rows.Close()
			list, _ := rowsToMaps(rows)
			if len(list) > 0 {
				def = list[0]
			}
		}
		api.OK(c, gin.H{"code": code, "definition": def, "payload": payload})
		return true
	}
	rows, err := s.DB.Query(`SELECT * FROM rpt_report_definition WHERE status='active' ORDER BY id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list), "overview": s.buildEnterprisePayload()})
	return true
}

func (s *Services) buildEnterprisePayload() gin.H {
	return gin.H{
		"sales_orders": s.queryCount(`SELECT COUNT(1) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0`),
		"sales_amount": s.queryFloat(`SELECT COALESCE(SUM(total_amount),0) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0`),
		"customers":    s.queryCount(`SELECT COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0`),
		"products":     s.queryCount(`SELECT COUNT(1) FROM prd_product WHERE COALESCE(is_deleted,0)=0`),
		"stock_sku":    s.queryCount(`SELECT COUNT(1) FROM inv_balance WHERE qty>0`),
		"fund_balance": s.queryFloat(`SELECT COALESCE(SUM(balance),0) FROM fin_fund_account`),
		"open_tasks":   s.queryCount(`SELECT COUNT(1) FROM pd_production_task WHERE COALESCE(is_deleted,0)=0 AND status IN ('pending','released','in_progress')`),
		"fixed_assets": s.queryCount(`SELECT COUNT(1) FROM ast_fixed_asset WHERE COALESCE(is_deleted,0)=0 AND status!='scrapped'`),
		"generated_at": time.Now().Format("2006-01-02 15:04:05"),
	}
}

func (s *Services) reportDaily(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	if bizDate == "" {
		bizDate = reportToday()
	}
	payload := gin.H{
		"biz_date":     bizDate,
		"sales_amount": s.queryFloat(`SELECT COALESCE(SUM(total_amount),0) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0 AND date(created_at)=?`, bizDate),
		"sales_orders": s.queryCount(`SELECT COUNT(1) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0 AND date(created_at)=?`, bizDate),
		"report_works": s.queryCount(`SELECT COUNT(1) FROM pd_report_work WHERE date(COALESCE(reported_at,created_at))=?`, bizDate),
		"report_qty":   s.queryFloat(`SELECT COALESCE(SUM(qty),0) FROM pd_report_work WHERE date(COALESCE(reported_at,created_at))=?`, bizDate),
		"stock_in":     s.queryFloat(`SELECT COALESCE(SUM(l.qty),0) FROM inv_stock_txn_line l JOIN inv_stock_txn t ON t.id=l.txn_id WHERE t.status='posted' AND l.direction='in' AND date(t.biz_date)=?`, bizDate),
		"stock_out":    s.queryFloat(`SELECT COALESCE(SUM(l.qty),0) FROM inv_stock_txn_line l JOIN inv_stock_txn t ON t.id=l.txn_id WHERE t.status='posted' AND l.direction='out' AND date(t.biz_date)=?`, bizDate),
		"cash_in":      s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='in' AND biz_date=?`, bizDate),
		"cash_out":     s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='out' AND biz_date=?`, bizDate),
		"follow_ups":   s.queryCount(`SELECT COUNT(1) FROM crm_follow_up WHERE date(follow_at)=?`, bizDate),
	}
	b, _ := json.Marshal(payload)
	_, _ = s.DB.Exec(`INSERT INTO rpt_report_snapshot(report_code, biz_date, payload_json) VALUES('daily',?,?)
		ON CONFLICT(report_code, biz_date) DO UPDATE SET payload_json=excluded.payload_json`, bizDate, string(b))
	api.OK(c, gin.H{"list": []gin.H{payload}, "total": 1, "summary": payload})
	return true
}

func (s *Services) reportCRMStats(c *gin.Context) bool {
	list := []gin.H{
		{"metric": "customers_total", "title": "客户总数", "value": s.queryCount(`SELECT COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0`)},
		{"metric": "public_sea", "title": "公海客户", "value": s.queryCount(`SELECT COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0 AND COALESCE(is_public_sea,0)=1`)},
		{"metric": "locked", "title": "锁定线索", "value": s.queryCount(`SELECT COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0 AND COALESCE(is_locked,0)=1`)},
		{"metric": "opportunities", "title": "商机数", "value": s.queryCount(`SELECT COUNT(1) FROM crm_opportunity WHERE COALESCE(is_deleted,0)=0`)},
		{"metric": "follow_ups", "title": "跟进记录", "value": s.queryCount(`SELECT COUNT(1) FROM crm_follow_up`)},
		{"metric": "follow_ups_7d", "title": "近7日跟进", "value": s.queryCount(`SELECT COUNT(1) FROM crm_follow_up WHERE date(follow_at)>=date('now','-7 day')`)},
	}
	byLevel := []gin.H{}
	rows, err := s.DB.Query(`SELECT COALESCE(NULLIF(level,''),'未分级'), COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0 GROUP BY COALESCE(NULLIF(level,''),'未分级')`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lv string
			var cnt int
			_ = rows.Scan(&lv, &cnt)
			byLevel = append(byLevel, gin.H{"level": lv, "count": cnt})
		}
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "by_level": byLevel})
	return true
}

func (s *Services) reportInquiries(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT id, doc_no, customer_id, status, created_at FROM sl_inquiry ORDER BY id DESC LIMIT 200`)
	if err != nil {
		api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportFollowUps(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT f.id, f.customer_id, COALESCE(c.name,'') AS customer_name, f.user_id, f.follow_type, f.follow_at, f.content, f.next_remind_at
		FROM crm_follow_up f LEFT JOIN crm_customer c ON c.id=f.customer_id ORDER BY f.id DESC LIMIT 200`)
	if err != nil {
		api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportStockTxns(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT t.id, t.doc_no, t.doc_type, t.warehouse_id, t.status, t.biz_date, COALESCE(t.remark,''), t.created_at,
		COALESCE((SELECT SUM(qty) FROM inv_stock_txn_line l WHERE l.txn_id=t.id AND l.direction='in'),0),
		COALESCE((SELECT SUM(qty) FROM inv_stock_txn_line l WHERE l.txn_id=t.id AND l.direction='out'),0)
		FROM inv_stock_txn t WHERE COALESCE(t.is_deleted,0)=0 ORDER BY t.id DESC LIMIT 300`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, wh int64
		var docNo, docType, status, bizDate, remark, created string
		var qtyIn, qtyOut float64
		_ = rows.Scan(&id, &docNo, &docType, &wh, &status, &bizDate, &remark, &created, &qtyIn, &qtyOut)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "doc_type": docType, "warehouse_id": wh, "status": status,
			"biz_date": bizDate, "remark": remark, "created_at": created, "qty_in": qtyIn, "qty_out": qtyOut,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportStockLedger(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT b.id, b.warehouse_id, COALESCE(w.name,''), b.product_id, COALESCE(p.code,''), COALESCE(p.name,''),
		b.qty, COALESCE(b.batch_no,''), COALESCE(b.avg_cost,0)
		FROM inv_balance b
		LEFT JOIN inv_warehouse w ON w.id=b.warehouse_id
		LEFT JOIN prd_product p ON p.id=b.product_id
		ORDER BY b.warehouse_id, b.product_id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, wh, pid int64
		var whName, code, name, batch string
		var qty, cost float64
		_ = rows.Scan(&id, &wh, &whName, &pid, &code, &name, &qty, &batch, &cost)
		list = append(list, gin.H{
			"id": id, "warehouse_id": wh, "warehouse_name": whName, "product_id": pid,
			"product_code": code, "product_name": name, "qty": qty, "batch_no": batch,
			"avg_cost": cost, "amount": qty * cost,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportSalesWeight(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT l.product_id, COALESCE(p.code,''), COALESCE(p.name,''),
		COALESCE(SUM(l.qty),0), COALESCE(SUM(l.weight),0), COALESCE(SUM(l.amount),0)
		FROM sl_sales_order_line l
		JOIN sl_sales_order o ON o.id=l.order_id AND COALESCE(o.is_deleted,0)=0
		LEFT JOIN prd_product p ON p.id=l.product_id
		GROUP BY l.product_id, p.code, p.name ORDER BY COALESCE(SUM(l.weight),0) DESC`)
	if err != nil {
		api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var pid int64
		var code, name string
		var qty, weight, amount float64
		_ = rows.Scan(&pid, &code, &name, &qty, &weight, &amount)
		list = append(list, gin.H{"product_id": pid, "product_code": code, "product_name": name, "qty": qty, "weight": weight, "amount": amount})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportProductSales(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT l.product_id, COALESCE(p.code,''), COALESCE(p.name,''),
		COUNT(DISTINCT l.order_id), COALESCE(SUM(l.qty),0), COALESCE(SUM(l.amount),0),
		CASE WHEN SUM(l.qty)>0 THEN SUM(l.amount)/SUM(l.qty) ELSE 0 END
		FROM sl_sales_order_line l
		JOIN sl_sales_order o ON o.id=l.order_id AND COALESCE(o.is_deleted,0)=0
		LEFT JOIN prd_product p ON p.id=l.product_id
		GROUP BY l.product_id, p.code, p.name ORDER BY COALESCE(SUM(l.amount),0) DESC`)
	if err != nil {
		api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var pid int64
		var code, name string
		var orders int
		var qty, amount, avgPrice float64
		_ = rows.Scan(&pid, &code, &name, &orders, &qty, &amount, &avgPrice)
		list = append(list, gin.H{
			"product_id": pid, "product_code": code, "product_name": name,
			"order_count": orders, "qty": qty, "amount": amount, "avg_price": avgPrice,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportQC(c *gin.Context) bool {
	list := []gin.H{}
	rows, err := s.DB.Query(`SELECT id, doc_no, qc_type, product_id, process_id, qty, result, status, created_at FROM pd_qc_order ORDER BY id DESC LIMIT 200`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var docNo, qcType, result, status, created string
			var pid, procID sql.NullInt64
			var qty float64
			var resultN, statusN sql.NullString
			_ = rows.Scan(&id, &docNo, &qcType, &pid, &procID, &qty, &resultN, &statusN, &created)
			if resultN.Valid {
				result = resultN.String
			}
			if statusN.Valid {
				status = statusN.String
			}
			list = append(list, gin.H{
				"id": id, "source": "production", "doc_no": docNo, "qc_type": qcType,
				"product_id": pid.Int64, "process_id": procID.Int64, "qty": qty,
				"result": result, "status": status, "created_at": created,
			})
		}
	}
	irows, err := s.DB.Query(`SELECT id, doc_no, product_id, qty_check, qty_pass, qty_fail, result, status, created_at
		FROM inv_inbound_qc WHERE COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 200`)
	if err == nil {
		defer irows.Close()
		for irows.Next() {
			var id, pid int64
			var docNo, result, status, created string
			var check, pass, fail float64
			var resultN, statusN sql.NullString
			_ = irows.Scan(&id, &docNo, &pid, &check, &pass, &fail, &resultN, &statusN, &created)
			if resultN.Valid {
				result = resultN.String
			}
			if statusN.Valid {
				status = statusN.String
			}
			list = append(list, gin.H{
				"id": id, "source": "inbound", "doc_no": docNo, "product_id": pid,
				"qty_check": check, "qty_pass": pass, "qty_fail": fail,
				"result": result, "status": status, "created_at": created,
			})
		}
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportLogistics(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT t.id, t.track_no, t.carrier_id, COALESCE(c.name,''), t.order_id, t.status, COALESCE(t.location,''), t.updated_at
		FROM sys_logistics_track t LEFT JOIN sys_logistics_carrier c ON c.id=t.carrier_id ORDER BY t.id DESC LIMIT 200`)
	if err != nil {
		api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var carrierID, orderID sql.NullInt64
		var trackNo, carrierName, status, loc, updated string
		_ = rows.Scan(&id, &trackNo, &carrierID, &carrierName, &orderID, &status, &loc, &updated)
		list = append(list, gin.H{
			"id": id, "track_no": trackNo, "carrier_id": carrierID.Int64, "carrier_name": carrierName,
			"order_id": orderID.Int64, "status": status, "location": loc, "updated_at": updated,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportAccounts(c *gin.Context) bool {
	list := []gin.H{}
	var income, expense float64
	rows, err := s.DB.Query(`SELECT COALESCE(direction,'in'), COUNT(1), COALESCE(SUM(amount),0) FROM fin_ledger_entry GROUP BY COALESCE(direction,'in')`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dir string
			var cnt int
			var amt float64
			_ = rows.Scan(&dir, &cnt, &amt)
			list = append(list, gin.H{"direction": dir, "count": cnt, "amount": amt})
			if dir == "in" {
				income = amt
			} else {
				expense = amt
			}
		}
	}
	funds := []map[string]interface{}{}
	frows, _ := s.DB.Query(`SELECT id, code, name, currency, balance FROM fin_fund_account WHERE status='active'`)
	if frows != nil {
		defer frows.Close()
		funds, _ = rowsToMaps(frows)
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"income": income, "expense": expense, "net": income - expense}, "funds": funds})
	return true
}

func (s *Services) reportGrossProfit(c *gin.Context) bool {
	revenue := s.queryFloat(`SELECT COALESCE(SUM(total_amount),0) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0 AND status NOT IN ('cancelled')`)
	if revenue == 0 {
		revenue = s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='in'`)
	}
	cost := s.queryFloat(`SELECT COALESCE(SUM(total_cost),0) FROM fin_cost_accounting WHERE status IN ('calculated','posted')`)
	if cost == 0 {
		cost = s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='out'`)
	}
	profit := revenue - cost
	rate := 0.0
	if revenue > 0 {
		rate = profit / revenue * 100
	}
	list := []gin.H{
		{"metric": "revenue", "title": "收入", "value": revenue},
		{"metric": "cost", "title": "成本", "value": cost},
		{"metric": "gross_profit", "title": "毛利", "value": profit},
		{"metric": "gross_margin_pct", "title": "毛利率%", "value": rate},
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"revenue": revenue, "cost": cost, "gross_profit": profit, "margin_pct": rate}})
	return true
}

func (s *Services) reportCostProfit(c *gin.Context) bool {
	list := []map[string]interface{}{}
	rows, err := s.DB.Query(`SELECT id, doc_no, period, product_id, material_cost, labor_cost, overhead, total_cost, status FROM fin_cost_accounting ORDER BY id DESC LIMIT 200`)
	if err == nil {
		defer rows.Close()
		list, _ = rowsToMaps(rows)
	}
	totalCost := s.queryFloat(`SELECT COALESCE(SUM(total_cost),0) FROM fin_cost_accounting`)
	revenue := s.queryFloat(`SELECT COALESCE(SUM(total_amount),0) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0`)
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"total_cost": totalCost, "revenue": revenue, "profit": revenue - totalCost}})
	return true
}

func (s *Services) reportBalanceSheet(c *gin.Context) bool {
	cash := s.queryFloat(`SELECT COALESCE(SUM(balance),0) FROM fin_fund_account`)
	assetNet := s.queryFloat(`SELECT COALESCE(SUM(net_value),0) FROM ast_fixed_asset WHERE COALESCE(is_deleted,0)=0 AND status!='scrapped'`)
	assetOrig := s.queryFloat(`SELECT COALESCE(SUM(original_value),0) FROM ast_fixed_asset WHERE COALESCE(is_deleted,0)=0 AND status!='scrapped'`)
	stockQty := s.queryFloat(`SELECT COALESCE(SUM(qty),0) FROM inv_balance`)
	liabilities := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_arap_adjust WHERE party_type='supplier' AND status='posted'`)
	assets := cash + assetNet
	equity := assets - liabilities
	list := []gin.H{
		{"section": "asset", "item": "货币资金", "amount": cash},
		{"section": "asset", "item": "存货数量(参考)", "amount": stockQty},
		{"section": "asset", "item": "固定资产原值", "amount": assetOrig},
		{"section": "asset", "item": "固定资产净值", "amount": assetNet},
		{"section": "liability", "item": "应付/往来", "amount": liabilities},
		{"section": "equity", "item": "净资产(估)", "amount": equity},
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"total_assets": assets, "total_liabilities": liabilities, "equity": equity}, "as_of": reportToday()})
	return true
}

func (s *Services) reportCashFlow(c *gin.Context) bool {
	cashIn := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='in'`)
	cashOut := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='out'`)
	list := []gin.H{
		{"item": "经营活动现金流入", "amount": cashIn},
		{"item": "经营活动现金流出", "amount": cashOut},
		{"item": "净现金流", "amount": cashIn - cashOut},
		{"item": "期末资金余额", "amount": s.queryFloat(`SELECT COALESCE(SUM(balance),0) FROM fin_fund_account`)},
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"in": cashIn, "out": cashOut, "net": cashIn - cashOut}})
	return true
}

func (s *Services) reportIncomeStatement(c *gin.Context) bool {
	income := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='in'`)
	if income == 0 {
		income = s.queryFloat(`SELECT COALESCE(SUM(total_amount),0) FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0`)
	}
	expense := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='out'`)
	cost := s.queryFloat(`SELECT COALESCE(SUM(total_cost),0) FROM fin_cost_accounting WHERE status IN ('calculated','posted')`)
	list := []gin.H{
		{"item": "营业收入", "amount": income},
		{"item": "营业成本", "amount": cost},
		{"item": "期间费用/支出", "amount": expense},
		{"item": "利润总额(估)", "amount": income - cost - expense},
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"income": income, "cost": cost, "expense": expense, "profit": income - cost - expense}})
	return true
}
