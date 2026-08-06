package biz

import (
	"database/sql"
	"log"
	"strings"
)

// execSchemaRuns runs DDL; ignores idempotent "already exists" errors, logs real failures once.
func execSchemaRuns(db *sql.DB, label string, stmts []string) {
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil && !isIdempotentSchemaErr(err) {
			log.Printf("%s schema ensure: %v", label, err)
		}
	}
}

func isIdempotentSchemaErr(err error) bool {
	if err == nil {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate key name") ||
		// MySQL: Duplicate column name / Table already exists
		strings.Contains(msg, "1050") ||
		strings.Contains(msg, "1060") ||
		strings.Contains(msg, "1061")
}

// EnsureAutomationSchema creates/alters tables needed for flow/scan/audit on existing DBs.
func EnsureAutomationSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS inv_box_code (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  product_id INTEGER NOT NULL,
  warehouse_id INTEGER,
  batch_no TEXT,
  qty REAL NOT NULL DEFAULT 0,
  weight REAL,
  parent_box_id INTEGER,
  current_process_id INTEGER,
  current_step_id INTEGER,
  task_id INTEGER,
  work_order_id INTEGER,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pd_flow_event (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_type TEXT NOT NULL,
  source_id INTEGER NOT NULL,
  from_step_id INTEGER,
  to_step_id INTEGER,
  trigger_action TEXT NOT NULL,
  trace_id TEXT,
  status TEXT NOT NULL DEFAULT 'ok',
  error TEXT,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_piecework_summary (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  worker_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  qty REAL NOT NULL DEFAULT 0,
  weight REAL,
  amount REAL NOT NULL DEFAULT 0,
  source_report_ids TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_material_requisition (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  work_order_id INTEGER,
  dispatch_id INTEGER,
  warehouse_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pd_material_requisition_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  requisition_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  base_qty REAL NOT NULL,
  batch_no TEXT
)`,
		`CREATE TABLE IF NOT EXISTS sys_data_repair (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  target_type TEXT NOT NULL,
  target_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  reason TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  payload_json TEXT,
  applied_by INTEGER,
  applied_at TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`ALTER TABLE sys_operation_log ADD COLUMN trace_id TEXT`,
		`ALTER TABLE hr_employee ADD COLUMN badge_code TEXT`,
		`ALTER TABLE pd_routing_step ADD COLUMN step_code TEXT`,
		`ALTER TABLE pd_routing_step ADD COLUMN step_name TEXT`,
		`ALTER TABLE pd_routing_step ADD COLUMN is_piecework INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE pd_routing_step ADD COLUMN auto_next INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE pd_routing_step ADD COLUMN auto_stock_in INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE pd_routing_step ADD COLUMN auto_stock_out INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE pd_routing_step ADD COLUMN warehouse_id INTEGER`,
		`ALTER TABLE pd_report_work ADD COLUMN input_weight REAL`,
		`ALTER TABLE pd_report_work ADD COLUMN output_weight REAL`,
		`ALTER TABLE pd_report_work ADD COLUMN loss REAL`,
		`ALTER TABLE pd_report_work ADD COLUMN utilization REAL`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN input_weight REAL`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN output_weight REAL`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN loss REAL`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN utilization REAL`,
		`ALTER TABLE inv_box_code ADD COLUMN farmer_id INTEGER`,
		`ALTER TABLE inv_box_code ADD COLUMN trace_code TEXT`,
		`ALTER TABLE inv_box_code ADD COLUMN origin TEXT`,
		`ALTER TABLE inv_box_code ADD COLUMN receive_date TEXT`,
		`ALTER TABLE inv_box_code ADD COLUMN source_type TEXT`,
		`UPDATE inv_warehouse SET name='保鲜库' WHERE code='WH-RAW' AND name='原料仓'`,
		`UPDATE inv_warehouse SET name='半成品库' WHERE code='WH-SEMI' AND name='半成品仓'`,
		`UPDATE inv_warehouse SET name='成品冷库' WHERE code='WH-FG' AND name='成品仓'`,
	}
	execSchemaRuns(db, "automation", stmts)
	seedAutomation(db)
	EnsureFarmerSchema(db)
	EnsureClosedLoopSchema(db)
	EnsureFieldLedgerSchema(db)
	EnsureSalesSchema(db)
	EnsureProductionExtSchema(db)
	EnsureInventoryExtSchema(db)
	EnsureProductExtSchema(db)
	EnsureAssetSchema(db)
	EnsureFinanceSchema(db)
	EnsureReportSchema(db)
	EnsureApprovalSchema(db)
}

// EnsureClosedLoopSchema evidence, audit, arrival, trace lot, confirm columns.
func EnsureClosedLoopSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS biz_evidence (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  biz_type TEXT NOT NULL,
  biz_id INTEGER NOT NULL,
  evidence_type TEXT NOT NULL,
  file_url TEXT NOT NULL,
  meta_json TEXT,
  uploaded_by INTEGER,
  uploaded_at TEXT NOT NULL DEFAULT (datetime('now')),
  voided_at TEXT
)`,
		`CREATE TABLE IF NOT EXISTS biz_audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  biz_type TEXT NOT NULL,
  biz_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  reason TEXT,
  before_json TEXT,
  after_json TEXT,
  actor_user_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pur_inbound_arrival (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id INTEGER NOT NULL,
  origin TEXT,
  variety TEXT,
  estimate_weight REAL DEFAULT 0,
  source_type TEXT NOT NULL DEFAULT 'self',
  channel TEXT NOT NULL DEFAULT 'internal',
  qc_result TEXT,
  grade TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  qc_image_url TEXT,
  confirmed_by INTEGER,
  confirmed_at TEXT,
  confirmed_snapshot_json TEXT,
  remark TEXT,
  biz_date TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_trace_lot (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  trace_code TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  batch_no TEXT NOT NULL,
  farmer_id INTEGER NOT NULL,
  grade TEXT,
  arrival_id INTEGER,
  weigh_ticket_id INTEGER,
  channel TEXT,
  source_type TEXT,
  net_weight REAL NOT NULL DEFAULT 0,
  payload_canonical TEXT NOT NULL,
  signature TEXT NOT NULL,
  algo TEXT NOT NULL DEFAULT 'HmacSHA256_v1',
  replaces_trace_code TEXT,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pur_grade_price (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  grade TEXT NOT NULL UNIQUE,
  unit_price REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active'
)`,
	}
	execSchemaRuns(db, "closed-loop", stmts)
	alters := []string{
		`ALTER TABLE pur_farmer ADD COLUMN default_unit_price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN arrival_id INTEGER`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN grade TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN batch_no TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN ocr_draft_json TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN confirmed_by INTEGER`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN confirmed_at TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN confirmed_snapshot_json TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN purchase_completed_at TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN plate_no TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN receive_address TEXT`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN pass_rate REAL DEFAULT 0`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN reject_weight REAL DEFAULT 0`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN freight_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN loading_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN weigh_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_weigh_ticket ADD COLUMN weighbridge_id INTEGER`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN plate_no TEXT`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN receive_address TEXT`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN pass_rate REAL DEFAULT 0`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN reject_weight REAL DEFAULT 0`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN freight_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN loading_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_inbound_arrival ADD COLUMN weigh_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN transfer_no TEXT`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN paid_at TEXT`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN pay_evidence_url TEXT`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN freight_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN loading_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN weigh_fee REAL DEFAULT 0`,
		`ALTER TABLE pur_farmer_settlement ADD COLUMN goods_amount REAL DEFAULT 0`,
		`ALTER TABLE pd_report_work ADD COLUMN confirmed_by INTEGER`,
		`ALTER TABLE pd_report_work ADD COLUMN confirmed_at TEXT`,
		`ALTER TABLE pd_report_work ADD COLUMN confirmed_snapshot_json TEXT`,
		`ALTER TABLE pd_report_work ADD COLUMN process_qc_result TEXT`,
		`ALTER TABLE pd_report_work ADD COLUMN bag_qty REAL DEFAULT 0`,
		`ALTER TABLE pd_scrap_record ADD COLUMN scrap_type TEXT`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN transfer_no TEXT`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN paid_at TEXT`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN pay_evidence_url TEXT`,
		`ALTER TABLE pd_piecework_summary ADD COLUMN status TEXT DEFAULT 'open'`,
		`ALTER TABLE pay_process_wage_rate ADD COLUMN rate_unit TEXT DEFAULT 'kg'`,
	}
	execSchemaRuns(db, "closed-loop-alter", alters)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pur_grade_price(grade, unit_price, status) VALUES
 ('A', 1.20, 'active'), ('B', 1.00, 'active'), ('C', 0.80, 'active')`)
}

// EnsureFarmerSchema creates farmer inbound / weigh / settlement tables.
func EnsureFarmerSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pur_farmer (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  mobile TEXT,
  origin TEXT,
  trace_code TEXT,
  trace_code_prefix TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_weigh_ticket (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id INTEGER NOT NULL,
  channel TEXT NOT NULL DEFAULT 'internal',
  ticket_template TEXT,
  product_id INTEGER NOT NULL DEFAULT 1,
  variety TEXT,
  gross_weight REAL NOT NULL DEFAULT 0,
  deduct_rate REAL NOT NULL DEFAULT 0,
  deduct_weight REAL NOT NULL DEFAULT 0,
  net_weight REAL NOT NULL DEFAULT 0,
  qc_result TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  trace_code TEXT,
  origin TEXT,
  biz_date TEXT NOT NULL,
  source_type TEXT NOT NULL DEFAULT 'self',
  image_url TEXT,
  box_code TEXT,
  warehouse_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_farmer_settlement (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id INTEGER NOT NULL,
  weigh_ticket_id INTEGER,
  biz_date TEXT NOT NULL,
  net_weight REAL NOT NULL DEFAULT 0,
  unit_price REAL NOT NULL DEFAULT 0,
  amount REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_scrap_record (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  task_id INTEGER,
  process_id INTEGER,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL DEFAULT 0,
  weight REAL,
  disposition TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	}
	execSchemaRuns(db, "farmer", stmts)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pur_farmer(id, code, name, mobile, origin, trace_code, trace_code_prefix, status, remark) VALUES
 (1, 'F-DEMO-01', '黄农户', '13900001111', '广西田东产地', 'TR-HUANG-DEMO', 'TR', 'active', '演示散户'),
 (2, 'F-DEMO-02', '李农户', '13900002222', '广西武鸣产地', 'TR-LI-DEMO', 'TR', 'active', '演示散户')`)
}

func seedAutomation(db *sql.DB) {
	_, _ = db.Exec(`INSERT OR IGNORE INTO pd_process(id, code, name, process_type, is_piecework, is_handover_point) VALUES
 (7, 'WASH', '清洗', 'wash', 0, 0),
 (8, 'IN_RAW', '原料入库', 'inbound', 0, 0),
 (9, 'IN_SEMI', '半成品入库', 'inbound', 0, 0),
 (10, 'OUT_DICE', '出库切块', 'outbound', 1, 0),
 (11, 'IN_FG', '成品入库', 'inbound', 0, 0)`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pd_routing(id, code, name, product_id, version_no, status) VALUES
 (1, 'RT-CASSAVA', '木薯丁产线', 3, 'V1', 'active')`)
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id=1`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_id) VALUES
 (1, 1, 8, 'S1', '入库-原料', 0, 0, 1, 1, 0, 1, 1),
 (1, 2, 7, 'S2', '清洗', 0, 0, 1, 0, 0, 1, 1),
 (1, 3, 1, 'S3', '去皮-计件领料', 1, 0, 1, 0, 1, 1, 1),
 (1, 4, 2, 'S4', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1),
 (1, 5, 3, 'S5', '切断-固定工', 0, 0, 1, 0, 0, NULL, 1),
 (1, 6, 4, 'S6', '去芯-计件', 1, 0, 1, 0, 0, NULL, 1),
 (1, 7, 2, 'S7', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1),
 (1, 8, 9, 'S8', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (1, 9, 10, 'S9', '出库切块-计件', 1, 0, 1, 0, 1, 2, 1),
 (1, 10, 9, 'S10', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (1, 11, 6, 'S11', '过滤装袋', 0, 0, 1, 0, 0, NULL, 1),
 (1, 12, 11, 'S12', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 1)`)
	}
	_, _ = db.Exec(`INSERT OR IGNORE INTO pay_process_wage_rate(process_id, rate, effective_from, status) VALUES (10, 0.22, '2026-07-01', 'active')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO hr_employee(id, emp_no, name, org_id, dept_id, workshop_id, emp_type, badge_code, status) VALUES
 (2, 'E0301', '陈某', 1, 1, 1, 'piece', 'EMP0301', 'active'),
 (3, 'E0205', '固定工甲', 1, 1, 1, 'fixed', 'EMP0205', 'active')`)
	_, _ = db.Exec(`UPDATE hr_employee SET badge_code='EMP0001' WHERE id=1 AND (badge_code IS NULL OR badge_code='')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO inv_box_code(id, code, product_id, warehouse_id, batch_no, qty, weight, current_process_id, current_step_id, status) VALUES
 (1, 'BX-RAW-DEMO', 1, 1, 'B0801', 1000, 1000, 8, NULL, 'open')`)
	EnsurePurchaseSchema(db)
}

// EnsurePurchaseSchema creates purchase/supplier tables on existing DBs.
func EnsurePurchaseSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pur_supplier (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  short_name TEXT,
  mnemonic TEXT,
  supplier_type TEXT NOT NULL DEFAULT 'raw',
  status TEXT NOT NULL DEFAULT 'potential',
  rating TEXT,
  is_preferred INTEGER NOT NULL DEFAULT 0,
  uscc TEXT,
  legal_person TEXT,
  register_address TEXT,
  invoice_title TEXT,
  tax_no TEXT,
  bank_name TEXT,
  bank_account TEXT,
  settle_method TEXT,
  payment_days INTEGER,
  credit_limit REAL,
  currency TEXT NOT NULL DEFAULT 'CNY',
  tax_rate REAL,
  lead_time_days INTEGER,
  moq REAL,
  default_warehouse_id INTEGER,
  contact_json TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_supplier_license (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  supplier_id INTEGER NOT NULL,
  license_type TEXT NOT NULL,
  license_no TEXT,
  expire_date TEXT,
  attachment_url TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pur_supplier_supply_item (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  supplier_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  is_preferred INTEGER NOT NULL DEFAULT 0,
  moq REAL,
  lead_time_days INTEGER,
  last_price REAL,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(supplier_id, product_id)
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_request (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  applicant_id INTEGER,
  title TEXT,
  qty REAL,
  status TEXT NOT NULL DEFAULT 'draft',
  need_date TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_request_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  request_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  unit_id INTEGER,
  suggest_supplier_id INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_plan (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'draft',
  plan_date TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_plan_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  supplier_id INTEGER,
  request_line_id INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_inbound (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  supplier_id INTEGER NOT NULL,
  warehouse_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  biz_date TEXT NOT NULL,
  plan_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_inbound_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  inbound_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  price REAL,
  amount REAL,
  batch_no TEXT
)`,
		`CREATE TABLE IF NOT EXISTS pur_incoming_qc (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  inbound_id INTEGER,
  supplier_id INTEGER,
  product_id INTEGER NOT NULL,
  qty_check REAL NOT NULL,
  qty_pass REAL NOT NULL DEFAULT 0,
  qty_fail REAL NOT NULL DEFAULT 0,
  result TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_return (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  supplier_id INTEGER NOT NULL,
  inbound_id INTEGER,
  warehouse_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  reason TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_return_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  return_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  amount REAL
)`,
		`CREATE TABLE IF NOT EXISTS pur_supplier_price_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  supplier_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  price REAL NOT NULL,
  biz_date TEXT NOT NULL,
  source_doc_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pur_purchase_task (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  assignee_id INTEGER,
  product_id INTEGER,
  qty REAL,
  status TEXT NOT NULL DEFAULT 'open',
  due_date TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`ALTER TABLE pur_supplier ADD COLUMN short_name TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN mnemonic TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN supplier_type TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN is_preferred INTEGER`,
		`ALTER TABLE pur_supplier ADD COLUMN uscc TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN legal_person TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN register_address TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN invoice_title TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN tax_no TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN bank_name TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN bank_account TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN settle_method TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN payment_days INTEGER`,
		`ALTER TABLE pur_supplier ADD COLUMN credit_limit REAL`,
		`ALTER TABLE pur_supplier ADD COLUMN currency TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN tax_rate REAL`,
		`ALTER TABLE pur_supplier ADD COLUMN lead_time_days INTEGER`,
		`ALTER TABLE pur_supplier ADD COLUMN moq REAL`,
		`ALTER TABLE pur_supplier ADD COLUMN default_warehouse_id INTEGER`,
		`ALTER TABLE pur_supplier ADD COLUMN remark TEXT`,
		`ALTER TABLE pur_supplier ADD COLUMN updated_at TEXT`,
		`ALTER TABLE pur_incoming_qc ADD COLUMN supplier_id INTEGER`,
		`ALTER TABLE pur_purchase_task ADD COLUMN title TEXT`,
		`ALTER TABLE pur_purchase_task ADD COLUMN supplier_id INTEGER`,
		`ALTER TABLE pur_purchase_task ADD COLUMN remark TEXT`,
	}
	execSchemaRuns(db, "purchase", stmts)
	seedPurchase(db)
}

func seedPurchase(db *sql.DB) {
	_, _ = db.Exec(`INSERT OR IGNORE INTO pur_supplier(id, code, name, short_name, supplier_type, status, rating, is_preferred, uscc, settle_method, payment_days, lead_time_days, moq, default_warehouse_id, contact_json, remark) VALUES
 (1, 'SUP-RAW-01', '广西木薯原料合作社', '桂薯原料', 'raw', 'qualified', 'A', 1, '91450000MA5XXXXX01', 'monthly', 30, 3, 1000, 1,
  '[{"name":"张经理","mobile":"13800001111","wechat":"zhang_sup","is_primary":true}]', '主原料供应商'),
 (2, 'SUP-AUX-01', '包装袋辅料厂', '包材厂', 'pack', 'qualified', 'B', 0, '91450000MA5XXXXX02', 'cash', 0, 7, 100, 1,
  '[{"name":"李主管","mobile":"13800002222","is_primary":true}]', '包装辅料'),
 (3, 'SUP-POT-01', '待准入产地商', '潜在产地', 'raw', 'potential', 'C', 0, NULL, 'cod', 0, 5, 500, 1,
  '[{"name":"王联系人","mobile":"13800003333","is_primary":true}]', '尚未准入')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pur_supplier_license(id, supplier_id, license_type, license_no, expire_date) VALUES
 (1, 1, '营业执照', '91450000MA5XXXXX01', '2027-12-31'),
 (2, 1, '食品经营许可', 'JY14500000001', '2026-09-30'),
 (3, 2, '营业执照', '91450000MA5XXXXX02', '2028-06-30')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO pur_supplier_supply_item(id, supplier_id, product_id, is_preferred, moq, lead_time_days, last_price) VALUES
 (1, 1, 1, 1, 1000, 3, 1.85),
 (2, 2, 2, 1, 100, 7, 0.45),
 (3, 3, 1, 0, 500, 5, 1.90)`)
}

// EnsureFieldLedgerSchema 现场台账：出厂结算、计件领料表、工具领还、地磅。
func EnsureFieldLedgerSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sl_outbound_settle (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  product_id INTEGER,
  product_name TEXT,
  plate_no TEXT,
  driver_name TEXT,
  trace_code TEXT,
  produce_date TEXT,
  qty REAL DEFAULT 0,
  weight REAL DEFAULT 0,
  unit TEXT DEFAULT 'kg',
  freight_fee REAL DEFAULT 0,
  loading_fee REAL DEFAULT 0,
  weigh_fee REAL DEFAULT 0,
  unit_price REAL DEFAULT 0,
  goods_amount REAL DEFAULT 0,
  amount REAL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS pd_piece_issue_sheet (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS pd_piece_issue_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sheet_id INTEGER NOT NULL,
  seq_no INTEGER NOT NULL DEFAULT 1,
  employee_id INTEGER,
  employee_name TEXT,
  process_id INTEGER,
  process_name TEXT,
  process_kind TEXT DEFAULT 'piece',
  unit_price REAL DEFAULT 0,
  qty REAL DEFAULT 0,
  reject_qty REAL DEFAULT 0,
  qty_total REAL DEFAULT 0,
  amount REAL DEFAULT 0,
  remark TEXT
)`,
		`CREATE TABLE IF NOT EXISTS hr_tool_item (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active'
)`,
		`CREATE TABLE IF NOT EXISTS hr_tool_issue (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  seq_no INTEGER DEFAULT 1,
  employee_id INTEGER,
  employee_name TEXT,
  tool_item_id INTEGER NOT NULL,
  tool_name TEXT,
  issue_qty REAL DEFAULT 0,
  return_qty REAL DEFAULT 0,
  total_qty REAL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS inv_weighbridge (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  location TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	}
	execSchemaRuns(db, "field-ledger", stmts)
	_, _ = db.Exec(`INSERT OR IGNORE INTO hr_tool_item(id, code, name, status) VALUES
 (1,'TOOL-SCRAPER','刮刀','active'),
 (2,'TOOL-KNIFE','小刀','active'),
 (3,'TOOL-GLOVE-T','厚手套','active'),
 (4,'TOOL-GLOVE-S','薄手套','active'),
 (5,'TOOL-HAT','帽子','active'),
 (6,'TOOL-UNIFORM','工服','active'),
 (7,'TOOL-SHOES','鞋子','active')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO inv_weighbridge(id, code, name, location, status) VALUES
 (1,'WB-GATE','大门地磅','厂区大门','active'),
 (2,'WB-FORK','叉车秤','原料区','active')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO prd_product_unit(product_id, unit_name, is_base, factor_to_base, is_purchase, is_sale, is_stock)
 SELECT id, '袋', 0, 25, 1, 1, 1 FROM prd_product WHERE product_type IN ('semi','finished') AND id NOT IN (
  SELECT product_id FROM prd_product_unit WHERE unit_name='袋'
 )`)
}

