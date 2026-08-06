-- SQLite 开发库：表名/字段与 MySQL 模型对齐
-- 金额骨架：生产切 MySQL 使用 DECIMAL(18,4)

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sys_organization (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_by INTEGER,
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  deleted_at TEXT,
  deleted_by INTEGER,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sys_department (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  parent_id INTEGER,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(org_id, code),
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS pd_workshop (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  dept_id INTEGER,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(org_id, code),
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS pd_work_team (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  workshop_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  leader_employee_id INTEGER,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(workshop_id, code),
  FOREIGN KEY(workshop_id) REFERENCES pd_workshop(id)
);

CREATE TABLE IF NOT EXISTS inv_warehouse (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  warehouse_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(org_id, code),
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS inv_location (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  warehouse_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(warehouse_id, code),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id)
);

CREATE TABLE IF NOT EXISTS sys_dict_type (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sys_dict_item (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  dict_type_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  sort_no INTEGER NOT NULL DEFAULT 0,
  ext_json TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(dict_type_id, code),
  FOREIGN KEY(dict_type_id) REFERENCES sys_dict_type(id)
);

CREATE TABLE IF NOT EXISTS sys_code_rule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  biz_type TEXT NOT NULL UNIQUE,
  prefix TEXT NOT NULL DEFAULT '',
  date_pattern TEXT NOT NULL DEFAULT 'yyyyMMdd',
  seq_length INTEGER NOT NULL DEFAULT 4,
  current_seq INTEGER NOT NULL DEFAULT 0,
  reset_policy TEXT NOT NULL DEFAULT 'daily',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 员工 / 用户 / 权限
CREATE TABLE IF NOT EXISTS hr_employee (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  emp_no TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  org_id INTEGER NOT NULL,
  dept_id INTEGER,
  workshop_id INTEGER,
  team_id INTEGER,
  job_title TEXT,
  emp_type TEXT NOT NULL DEFAULT 'office',
  mobile TEXT,
  badge_code TEXT,
  id_card_no TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  user_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS iam_user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  login_name TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  employee_id INTEGER,
  user_type TEXT NOT NULL DEFAULT 'biz',
  status TEXT NOT NULL DEFAULT 'active',
  freeze_reason TEXT,
  frozen_at TEXT,
  frozen_by INTEGER,
  last_login_at TEXT,
  login_fail_count INTEGER NOT NULL DEFAULT 0,
  lock_until TEXT,
  pwd_changed_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(employee_id) REFERENCES hr_employee(id)
);

CREATE TABLE IF NOT EXISTS iam_admin_group (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  remark TEXT,
  sort_no INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_role (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  data_scope_type TEXT NOT NULL DEFAULT 'self',
  remark TEXT,
  is_system INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_admin_group_role (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(group_id, role_id),
  FOREIGN KEY(group_id) REFERENCES iam_admin_group(id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_admin_group_user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(group_id, user_id),
  FOREIGN KEY(group_id) REFERENCES iam_admin_group(id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_user_role (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(user_id, role_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_permission (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  module TEXT NOT NULL,
  action TEXT NOT NULL,
  remark TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_role_permission (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id INTEGER NOT NULL,
  permission_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(role_id, permission_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id),
  FOREIGN KEY(permission_id) REFERENCES iam_permission(id)
);

CREATE TABLE IF NOT EXISTS pd_process (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  process_type TEXT,
  is_piecework INTEGER NOT NULL DEFAULT 0,
  is_handover_point INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_role_warehouse_scope (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id INTEGER NOT NULL,
  warehouse_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(role_id, warehouse_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id)
);

CREATE TABLE IF NOT EXISTS iam_role_process_scope (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  can_report INTEGER NOT NULL DEFAULT 1,
  can_dispatch INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(role_id, process_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS iam_user_warehouse_scope (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  warehouse_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(user_id, warehouse_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id)
);

CREATE TABLE IF NOT EXISTS iam_user_process_scope (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  can_report INTEGER NOT NULL DEFAULT 1,
  can_dispatch INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(user_id, process_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS iam_user_data_scope (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL UNIQUE,
  data_scope_type TEXT NOT NULL,
  workshop_id INTEGER,
  team_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_field_policy (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id INTEGER NOT NULL,
  field_key TEXT NOT NULL,
  field_name TEXT,
  visible INTEGER NOT NULL DEFAULT 0,
  editable INTEGER NOT NULL DEFAULT 0,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(role_id, field_key),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_menu_custom (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id INTEGER NOT NULL,
  domain TEXT NOT NULL,
  module TEXT NOT NULL,
  menu_key TEXT NOT NULL,
  visible INTEGER NOT NULL DEFAULT 1,
  sort_no INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(role_id, menu_key),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_login_policy (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER,
  max_fail_count INTEGER NOT NULL DEFAULT 5,
  lock_minutes INTEGER NOT NULL DEFAULT 30,
  session_ttl_min INTEGER NOT NULL DEFAULT 120,
  password_min_len INTEGER NOT NULL DEFAULT 8,
  password_require_letter INTEGER NOT NULL DEFAULT 1,
  password_require_digit INTEGER NOT NULL DEFAULT 1,
  password_require_special INTEGER NOT NULL DEFAULT 0,
  password_history INTEGER NOT NULL DEFAULT 5,
  force_change_days INTEGER,
  single_session INTEGER NOT NULL DEFAULT 0,
  password_rule_json TEXT,
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS iam_user_session (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  token_hash TEXT NOT NULL,
  client_type TEXT,
  ip TEXT,
  user_agent TEXT,
  login_at TEXT NOT NULL DEFAULT (datetime('now')),
  expire_at TEXT NOT NULL,
  revoked_at TEXT,
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_password_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_user_oauth (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  provider TEXT NOT NULL,
  open_id TEXT NOT NULL,
  union_id TEXT,
  bound_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(provider, open_id),
  UNIQUE(user_id, provider),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_onboard_role_template (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  emp_type TEXT NOT NULL,
  role_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(emp_type, role_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS sys_operation_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER,
  action TEXT NOT NULL,
  module TEXT,
  ref_type TEXT,
  ref_id INTEGER,
  detail_json TEXT,
  ip TEXT,
  trace_id TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sys_org_setting (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER,
  setting_key TEXT NOT NULL,
  value_json TEXT,
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(org_id, setting_key)
);

-- 产品 / 库存 / 生产一期最小表
CREATE TABLE IF NOT EXISTS prd_product (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category TEXT,
  product_type TEXT NOT NULL,
  base_unit_id INTEGER,
  spec_text TEXT,
  barcode TEXT,
  cost_price REAL,
  sale_price REAL,
  is_batch_managed INTEGER NOT NULL DEFAULT 1,
  is_box_managed INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS prd_product_unit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER NOT NULL,
  unit_name TEXT NOT NULL,
  is_base INTEGER NOT NULL DEFAULT 0,
  factor_to_base REAL NOT NULL DEFAULT 1,
  is_purchase INTEGER NOT NULL DEFAULT 1,
  is_sale INTEGER NOT NULL DEFAULT 1,
  is_stock INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(product_id, unit_name),
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS inv_balance (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  warehouse_id INTEGER NOT NULL,
  location_id INTEGER NOT NULL DEFAULT 0,
  product_id INTEGER NOT NULL,
  batch_no TEXT NOT NULL DEFAULT '',
  box_code_id INTEGER NOT NULL DEFAULT 0,
  qty REAL NOT NULL DEFAULT 0,
  weight REAL,
  avg_cost REAL,
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(warehouse_id, product_id, batch_no, location_id, box_code_id),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id),
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS inv_stock_txn (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  doc_type TEXT NOT NULL,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  warehouse_id INTEGER,
  counterparty_type TEXT,
  counterparty_id INTEGER,
  source_doc_type TEXT,
  source_doc_id INTEGER,
  posted_at TEXT,
  org_id INTEGER,
  owner_user_id INTEGER,
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inv_stock_txn_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  txn_id INTEGER NOT NULL,
  line_no INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  unit_id INTEGER,
  qty REAL NOT NULL,
  base_qty REAL NOT NULL,
  weight REAL,
  batch_no TEXT,
  box_code_id INTEGER,
  location_id INTEGER,
  direction TEXT NOT NULL,
  amount REAL,
  remark TEXT,
  UNIQUE(txn_id, line_no),
  FOREIGN KEY(txn_id) REFERENCES inv_stock_txn(id),
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS inv_reservation (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  warehouse_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  batch_no TEXT,
  qty REAL NOT NULL,
  source_doc_type TEXT NOT NULL,
  source_doc_id INTEGER NOT NULL,
  source_line_id INTEGER,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS pd_routing (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  product_id INTEGER,
  version_no TEXT NOT NULL DEFAULT 'V1',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(code, version_no)
);

CREATE TABLE IF NOT EXISTS pd_routing_step (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  routing_id INTEGER NOT NULL,
  seq_no INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  step_code TEXT,
  step_name TEXT,
  is_piecework INTEGER NOT NULL DEFAULT 0,
  is_inbound_checkpoint INTEGER NOT NULL DEFAULT 0,
  is_qc_required INTEGER NOT NULL DEFAULT 0,
  auto_next INTEGER NOT NULL DEFAULT 1,
  auto_stock_in INTEGER NOT NULL DEFAULT 0,
  auto_stock_out INTEGER NOT NULL DEFAULT 0,
  warehouse_id INTEGER,
  workshop_id INTEGER,
  UNIQUE(routing_id, seq_no),
  FOREIGN KEY(routing_id) REFERENCES pd_routing(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS inv_box_code (
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
  is_deleted INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS pd_flow_event (
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
);

CREATE TABLE IF NOT EXISTS pd_piecework_summary (
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
);

CREATE TABLE IF NOT EXISTS sys_data_repair (
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
);

CREATE TABLE IF NOT EXISTS pd_production_task (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  source_type TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL DEFAULT 'pending',
  plan_start TEXT,
  plan_end TEXT,
  routing_id INTEGER,
  workshop_id INTEGER,
  org_id INTEGER,
  owner_user_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pd_production_task_item (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  plan_qty REAL NOT NULL,
  plan_weight REAL,
  completed_qty REAL NOT NULL DEFAULT 0,
  FOREIGN KEY(task_id) REFERENCES pd_production_task(id)
);

CREATE TABLE IF NOT EXISTS pd_work_order (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  task_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  routing_step_id INTEGER,
  status TEXT NOT NULL DEFAULT 'pending',
  plan_qty REAL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(task_id) REFERENCES pd_production_task(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS pd_dispatch (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  work_order_id INTEGER NOT NULL,
  dispatch_type TEXT NOT NULL DEFAULT 'normal',
  worker_id INTEGER,
  team_id INTEGER,
  plan_qty REAL,
  status TEXT NOT NULL DEFAULT 'dispatched',
  dispatched_at TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(work_order_id) REFERENCES pd_work_order(id)
);

CREATE TABLE IF NOT EXISTS pd_report_work (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  dispatch_id INTEGER,
  work_order_id INTEGER,
  process_id INTEGER NOT NULL,
  worker_id INTEGER NOT NULL,
  report_type TEXT NOT NULL DEFAULT 'output',
  qty REAL NOT NULL,
  weight REAL,
  qty_net REAL,
  deduct_impurity REAL NOT NULL DEFAULT 0,
  deduct_water REAL NOT NULL DEFAULT 0,
  qc_result TEXT,
  status TEXT NOT NULL DEFAULT 'submitted',
  reported_at TEXT NOT NULL,
  scan_code TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS pay_process_wage_rate (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  process_id INTEGER NOT NULL,
  product_id INTEGER,
  product_spec_id INTEGER,
  unit_id INTEGER,
  rate REAL NOT NULL,
  effective_from TEXT NOT NULL,
  effective_to TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS appr_task (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  flow_id INTEGER,
  node_id INTEGER,
  doc_type TEXT NOT NULL,
  doc_id INTEGER NOT NULL,
  assignee_user_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  acted_at TEXT,
  comment TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO schema_meta(key, value) VALUES ('version', '2');

CREATE TABLE IF NOT EXISTS erp_doc (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  resource_key TEXT NOT NULL,
  doc_no TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  payload TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_erp_doc_key ON erp_doc(resource_key, is_deleted);

-- 采购 / 供应商
CREATE TABLE IF NOT EXISTS pur_supplier (
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
);

CREATE TABLE IF NOT EXISTS pur_supplier_license (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  supplier_id INTEGER NOT NULL,
  license_type TEXT NOT NULL,
  license_no TEXT,
  expire_date TEXT,
  attachment_url TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(supplier_id) REFERENCES pur_supplier(id)
);

CREATE TABLE IF NOT EXISTS pur_supplier_supply_item (
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
  UNIQUE(supplier_id, product_id),
  FOREIGN KEY(supplier_id) REFERENCES pur_supplier(id)
);

CREATE TABLE IF NOT EXISTS pur_purchase_request (
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
);

CREATE TABLE IF NOT EXISTS pur_purchase_request_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  request_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  unit_id INTEGER,
  suggest_supplier_id INTEGER,
  FOREIGN KEY(request_id) REFERENCES pur_purchase_request(id)
);

CREATE TABLE IF NOT EXISTS pur_purchase_plan (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'draft',
  plan_date TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_purchase_plan_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  supplier_id INTEGER,
  request_line_id INTEGER,
  FOREIGN KEY(plan_id) REFERENCES pur_purchase_plan(id)
);

CREATE TABLE IF NOT EXISTS pur_purchase_inbound (
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
);

CREATE TABLE IF NOT EXISTS pur_purchase_inbound_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  inbound_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  price REAL,
  amount REAL,
  batch_no TEXT,
  FOREIGN KEY(inbound_id) REFERENCES pur_purchase_inbound(id)
);

CREATE TABLE IF NOT EXISTS pur_incoming_qc (
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
);

CREATE TABLE IF NOT EXISTS pur_purchase_return (
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
);

CREATE TABLE IF NOT EXISTS pur_purchase_return_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  return_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty REAL NOT NULL,
  amount REAL,
  FOREIGN KEY(return_id) REFERENCES pur_purchase_return(id)
);

CREATE TABLE IF NOT EXISTS pur_supplier_price_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  supplier_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  price REAL NOT NULL,
  biz_date TEXT NOT NULL,
  source_doc_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS pur_purchase_task (
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
);
