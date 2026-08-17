package biz

import (
	"database/sql"
	"log"
)

// EnsureSalesSchema is a no-op: schema owned by migrations/erp.
func EnsureSalesSchema(db *sql.DB) {
	_ = db
	return
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS crm_customer (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  short_name TEXT,
  contact_name TEXT,
  mobile TEXT,
  address TEXT,
  level TEXT,
  source TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  owner_user_id INTEGER,
  protect_until TEXT,
  is_public_sea INTEGER NOT NULL DEFAULT 0,
  is_hidden INTEGER NOT NULL DEFAULT 0,
  is_locked INTEGER NOT NULL DEFAULT 0,
  contact_json TEXT,
  settle_method TEXT,
  payment_days INTEGER,
  credit_limit REAL,
  logistics_remark TEXT,
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW()),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS crm_opportunity (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER NOT NULL,
  title TEXT,
  stage TEXT NOT NULL DEFAULT 'lead',
  amount REAL NOT NULL DEFAULT 0,
  expected_date TEXT,
  owner_user_id INTEGER,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  converted_order_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS crm_follow_up (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER NOT NULL,
  opportunity_id INTEGER,
  user_id INTEGER NOT NULL,
  follow_type TEXT NOT NULL DEFAULT 'visit',
  follow_at TEXT NOT NULL,
  content TEXT,
  next_remind_at TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS crm_lead_assign (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER NOT NULL,
  from_user_id INTEGER,
  to_user_id INTEGER NOT NULL,
  assigned_at TEXT NOT NULL,
  lock_flag INTEGER NOT NULL DEFAULT 0,
  remark TEXT
)`,
		`CREATE TABLE IF NOT EXISTS crm_lead_protect_rule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  protect_days INTEGER NOT NULL DEFAULT 30,
  release_rule_json TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS crm_lead_release_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER NOT NULL,
  released_at TEXT NOT NULL,
  reason TEXT,
  to_public_sea INTEGER NOT NULL DEFAULT 1,
  from_user_id INTEGER,
  operator_user_id INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS crm_task_reminder (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  ref_type TEXT,
  ref_id INTEGER,
  remind_at TEXT NOT NULL,
  content TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS crm_customer_import_batch (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file_name TEXT,
  imported_at TEXT NOT NULL,
  success_count INTEGER NOT NULL DEFAULT 0,
  fail_count INTEGER NOT NULL DEFAULT 0,
  fail_detail_json TEXT,
  created_by INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS sl_contract (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER NOT NULL,
  title TEXT,
  amount REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  signed_at TEXT,
  expire_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW()),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS sl_price_lock (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  lock_price REAL NOT NULL,
  effective_from TEXT NOT NULL,
  effective_to TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_inquiry (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER NOT NULL,
  owner_user_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  source TEXT NOT NULL DEFAULT 'sales',
  expire_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW()),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS sl_inquiry_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  inquiry_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  weight REAL,
  quote_price REAL,
  cost_ref REAL,
  remark TEXT
)`,
		`CREATE TABLE IF NOT EXISTS sl_quote_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  price REAL NOT NULL,
  quoted_at TEXT NOT NULL,
  inquiry_id INTEGER,
  order_id INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS sl_sales_order (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER NOT NULL,
  owner_user_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  source TEXT NOT NULL DEFAULT 'manual',
  contract_id INTEGER,
  price_lock_id INTEGER,
  reorder_from_id INTEGER,
  warehouse_id INTEGER NOT NULL DEFAULT 3,
  need_delivery_approval INTEGER NOT NULL DEFAULT 1,
  total_amount REAL NOT NULL DEFAULT 0,
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW()),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS sl_sales_order_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  weight REAL,
  price REAL NOT NULL,
  amount REAL NOT NULL,
  delivered_qty REAL NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS sl_order_change_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id INTEGER NOT NULL,
  change_type TEXT NOT NULL,
  before_json TEXT,
  after_json TEXT,
  reason TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_pre_shipment (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  order_id INTEGER NOT NULL,
  plan_ship_date TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  reserved INTEGER NOT NULL DEFAULT 0,
  warehouse_id INTEGER NOT NULL DEFAULT 3,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_pre_shipment_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pre_shipment_id INTEGER NOT NULL,
  order_line_id INTEGER,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS sl_delivery_approval (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  order_id INTEGER NOT NULL,
  pre_shipment_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  warehouse_id INTEGER,
  shipped_at TEXT,
  logistics_no TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_delivery_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  delivery_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  weight REAL
)`,
		`CREATE TABLE IF NOT EXISTS sl_sales_bom (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  order_id INTEGER,
  product_id INTEGER NOT NULL,
  name TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_sales_bom_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bom_id INTEGER NOT NULL,
  material_product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  scrap_rate REAL NOT NULL DEFAULT 0,
  remark TEXT
)`,
		`CREATE TABLE IF NOT EXISTS sl_cost_budget (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id INTEGER NOT NULL,
  material_cost REAL NOT NULL DEFAULT 0,
  labor_cost REAL NOT NULL DEFAULT 0,
  other_cost REAL NOT NULL DEFAULT 0,
  total_cost REAL NOT NULL DEFAULT 0,
  sale_amount REAL NOT NULL DEFAULT 0,
  margin REAL NOT NULL DEFAULT 0,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_quote_calculator_result (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_id INTEGER,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL DEFAULT 1,
  base_cost REAL NOT NULL DEFAULT 0,
  margin_rate REAL NOT NULL DEFAULT 0.2,
  quote_price REAL NOT NULL DEFAULT 0,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_sales_rank_config (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  metric TEXT NOT NULL,
  period TEXT NOT NULL DEFAULT 'month',
  top_n INTEGER NOT NULL DEFAULT 10,
  updated_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS sl_print_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_type TEXT NOT NULL,
  doc_id INTEGER NOT NULL,
  doc_no TEXT,
  template_code TEXT,
  printed_by INTEGER,
  printed_at TEXT NOT NULL DEFAULT (NOW()),
  payload_json TEXT
)`,
		`CREATE TABLE IF NOT EXISTS sl_self_order_rule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  min_qty REAL NOT NULL DEFAULT 0,
  allow_products_json TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil && !isIdempotentSchemaErr(err) {
			log.Printf("sales schema ensure: %v", err)
		}
	}
	for _, alt := range []string{
		`ALTER TABLE crm_customer ADD COLUMN source TEXT`,
		`ALTER TABLE crm_customer ADD COLUMN protect_until TEXT`,
		`ALTER TABLE crm_customer ADD COLUMN is_public_sea INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE crm_customer ADD COLUMN is_hidden INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE crm_customer ADD COLUMN is_locked INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE crm_customer ADD COLUMN contact_json TEXT`,
		`ALTER TABLE crm_customer ADD COLUMN settle_method TEXT`,
		`ALTER TABLE crm_customer ADD COLUMN payment_days INTEGER`,
		`ALTER TABLE crm_customer ADD COLUMN credit_limit REAL`,
		`ALTER TABLE crm_customer ADD COLUMN logistics_remark TEXT`,
		`ALTER TABLE crm_customer ADD COLUMN created_by INTEGER`,
	} {
		if _, err := db.Exec(alt); err != nil && !isIdempotentSchemaErr(err) {
			log.Printf("crm schema migrate: %v", err)
		}
	}
	seedSales(db)
}

func seedSales(db *sql.DB) {
	_, _ = db.Exec(`INSERT INTO crm_customer(id, code, name, short_name, contact_name, mobile, address, level, source, status, owner_user_id, is_public_sea, remark) VALUES
 (1, 'CU-001', '南宁食品批发部', '南宁批发', '韦经理', '13800010001', '广西南宁市江南区', 'A', '展会', 'active', 1, 0, '演示客户'),
 (2, 'CU-002', '柳州餐饮连锁', '柳州餐饮', '周采购', '13800010002', '广西柳州市城中区', 'B', '转介绍', 'active', 1, 0, '演示客户'),
 (3, 'CU-003', '公海待分配客户', '公海样例', '待分配', '13800010003', '广西桂林市', 'C', '电话开发', 'active', NULL, 1, '公海演示')`)
	_, _ = db.Exec(`INSERT INTO crm_lead_protect_rule(id, name, protect_days, release_rule_json, status) VALUES
 (1, '默认保护30天', 30, '{"auto_release":true,"idle_days":30}', 'active')`)
	_, _ = db.Exec(`INSERT INTO crm_opportunity(id, customer_id, title, stage, amount, expected_date, owner_user_id, status, remark) VALUES
 (1, 1, '南宁批发袋装木薯丁年供', 'negotiation', 120000, '2026-09-30', 1, 'open', '演示商机')`)
	_, _ = db.Exec(`INSERT INTO crm_follow_up(id, customer_id, opportunity_id, user_id, follow_type, follow_at, content, next_remind_at) VALUES
 (1, 1, 1, 1, 'visit', NOW(), '到访确认锁价与冷链配送', datetime('now','+3 day'))`)
	_, _ = db.Exec(`INSERT INTO crm_task_reminder(id, user_id, ref_type, ref_id, remind_at, content, status) VALUES
 (1, 1, 'customer', 1, datetime('now','+1 day'), '跟进南宁批发部复购意向', 'pending')`)
	_, _ = db.Exec(`INSERT INTO sl_price_lock(id, customer_id, product_id, lock_price, effective_from, effective_to, status) VALUES
 (1, 1, 3, 6.80, '2026-01-01', '2026-12-31', 'active'),
 (2, 2, 3, 7.20, '2026-01-01', '2026-12-31', 'active')`)
	_, _ = db.Exec(`INSERT INTO sl_sales_rank_config(id, metric, period, top_n) VALUES
 (1, 'amount', 'month', 10),
 (2, 'qty', 'month', 10)`)
	_, _ = db.Exec(`INSERT INTO sl_self_order_rule(id, name, enabled, min_qty, allow_products_json, remark) VALUES
 (1, '默认自助下单规则', 1, 50, '[3]', '袋装木薯丁最小 50')`)
	// ensure finished goods stock for demo shipping
	_, _ = db.Exec(`INSERT INTO inv_balance(warehouse_id, location_id, product_id, batch_no, box_code_id, qty)
		VALUES(3, 0, 3, 'FG-SEED', 0, 5000)`)
}
