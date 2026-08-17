-- factory-erp PostgreSQL baseline v1.0.0
-- Converted from db/sqlite/schema.sql (formal R&D baseline)

CREATE TABLE IF NOT EXISTS erp_schema_migration (
  version     VARCHAR(32) PRIMARY KEY,
  description VARCHAR(255) NOT NULL DEFAULT '',
  applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  checksum    VARCHAR(64) NOT NULL DEFAULT ''
);

-- SQLite 开发库：表名/字段与 MySQL 模型对齐
-- 金额骨架：生产切 MySQL 使用 DECIMAL(18,4)



CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sys_organization (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_by INTEGER,
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  deleted_at TEXT,
  deleted_by INTEGER,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sys_department (
  id BIGSERIAL PRIMARY KEY,
  org_id INTEGER NOT NULL,
  parent_id INTEGER,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(org_id, code),
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS pd_workshop (
  id BIGSERIAL PRIMARY KEY,
  org_id INTEGER NOT NULL,
  dept_id INTEGER,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(org_id, code),
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS pd_work_team (
  id BIGSERIAL PRIMARY KEY,
  workshop_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  leader_employee_id INTEGER,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(workshop_id, code),
  FOREIGN KEY(workshop_id) REFERENCES pd_workshop(id)
);

CREATE TABLE IF NOT EXISTS inv_warehouse (
  id BIGSERIAL PRIMARY KEY,
  org_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  warehouse_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(org_id, code),
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS inv_location (
  id BIGSERIAL PRIMARY KEY,
  warehouse_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(warehouse_id, code),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id)
);

CREATE TABLE IF NOT EXISTS sys_dict_type (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sys_dict_item (
  id BIGSERIAL PRIMARY KEY,
  dict_type_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  sort_no INTEGER NOT NULL DEFAULT 0,
  ext_json TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(dict_type_id, code),
  FOREIGN KEY(dict_type_id) REFERENCES sys_dict_type(id)
);

CREATE TABLE IF NOT EXISTS sys_code_rule (
  id BIGSERIAL PRIMARY KEY,
  biz_type TEXT NOT NULL UNIQUE,
  prefix TEXT NOT NULL DEFAULT '',
  date_pattern TEXT NOT NULL DEFAULT 'yyyyMMdd',
  seq_length INTEGER NOT NULL DEFAULT 4,
  current_seq INTEGER NOT NULL DEFAULT 0,
  reset_policy TEXT NOT NULL DEFAULT 'daily',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

-- 员工 / 用户 / 权限
CREATE TABLE IF NOT EXISTS hr_employee (
  id BIGSERIAL PRIMARY KEY,
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
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(org_id) REFERENCES sys_organization(id)
);

CREATE TABLE IF NOT EXISTS iam_user (
  id BIGSERIAL PRIMARY KEY,
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
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(employee_id) REFERENCES hr_employee(id)
);

CREATE TABLE IF NOT EXISTS iam_admin_group (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  remark TEXT,
  sort_no INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_role (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  data_scope_type TEXT NOT NULL DEFAULT 'self',
  remark TEXT,
  is_system INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_admin_group_role (
  id BIGSERIAL PRIMARY KEY,
  group_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(group_id, role_id),
  FOREIGN KEY(group_id) REFERENCES iam_admin_group(id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_admin_group_user (
  id BIGSERIAL PRIMARY KEY,
  group_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(group_id, user_id),
  FOREIGN KEY(group_id) REFERENCES iam_admin_group(id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_user_role (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, role_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_permission (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  module TEXT NOT NULL,
  action TEXT NOT NULL,
  remark TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_role_permission (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  permission_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(role_id, permission_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id),
  FOREIGN KEY(permission_id) REFERENCES iam_permission(id)
);

CREATE TABLE IF NOT EXISTS pd_process (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  process_type TEXT,
  is_piecework INTEGER NOT NULL DEFAULT 0,
  is_handover_point INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iam_role_warehouse_scope (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  warehouse_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(role_id, warehouse_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id)
);

CREATE TABLE IF NOT EXISTS iam_role_process_scope (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  can_report INTEGER NOT NULL DEFAULT 1,
  can_dispatch INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(role_id, process_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS iam_user_warehouse_scope (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  warehouse_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, warehouse_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id)
);

CREATE TABLE IF NOT EXISTS iam_user_process_scope (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  can_report INTEGER NOT NULL DEFAULT 1,
  can_dispatch INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, process_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS iam_user_data_scope (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL UNIQUE,
  data_scope_type TEXT NOT NULL,
  workshop_id INTEGER,
  team_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_field_policy (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  field_key TEXT NOT NULL,
  field_name TEXT,
  visible INTEGER NOT NULL DEFAULT 0,
  editable INTEGER NOT NULL DEFAULT 0,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(role_id, field_key),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_menu_custom (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  domain TEXT NOT NULL,
  module TEXT NOT NULL,
  menu_key TEXT NOT NULL,
  visible INTEGER NOT NULL DEFAULT 1,
  sort_no INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(role_id, menu_key),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS iam_login_policy (
  id BIGSERIAL PRIMARY KEY,
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
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam_user_session (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  token_hash TEXT NOT NULL,
  client_type TEXT,
  ip TEXT,
  user_agent TEXT,
  login_at TEXT NOT NULL DEFAULT NOW(),
  expire_at TEXT NOT NULL,
  revoked_at TEXT,
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_password_history (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_user_oauth (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  provider TEXT NOT NULL,
  open_id TEXT NOT NULL,
  union_id TEXT,
  bound_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(provider, open_id),
  UNIQUE(user_id, provider),
  FOREIGN KEY(user_id) REFERENCES iam_user(id)
);

CREATE TABLE IF NOT EXISTS iam_onboard_role_template (
  id BIGSERIAL PRIMARY KEY,
  emp_type TEXT NOT NULL,
  role_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(emp_type, role_id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

CREATE TABLE IF NOT EXISTS sys_operation_log (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER,
  action TEXT NOT NULL,
  module TEXT,
  ref_type TEXT,
  ref_id INTEGER,
  detail_json TEXT,
  ip TEXT,
  trace_id TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sys_org_setting (
  id BIGSERIAL PRIMARY KEY,
  org_id INTEGER,
  setting_key TEXT NOT NULL,
  value_json TEXT,
  updated_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(org_id, setting_key)
);

-- 产品 / 库存 / 生产一期最小表
CREATE TABLE IF NOT EXISTS prd_product (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category TEXT,
  product_type TEXT NOT NULL,
  base_unit_id INTEGER,
  spec_text TEXT,
  barcode TEXT,
  cost_price DOUBLE PRECISION,
  sale_price DOUBLE PRECISION,
  is_batch_managed INTEGER NOT NULL DEFAULT 1,
  is_box_managed INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS prd_product_unit (
  id BIGSERIAL PRIMARY KEY,
  product_id INTEGER NOT NULL,
  unit_name TEXT NOT NULL,
  is_base INTEGER NOT NULL DEFAULT 0,
  factor_to_base DOUBLE PRECISION NOT NULL DEFAULT 1,
  is_purchase INTEGER NOT NULL DEFAULT 1,
  is_sale INTEGER NOT NULL DEFAULT 1,
  is_stock INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(product_id, unit_name),
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS inv_balance (
  id BIGSERIAL PRIMARY KEY,
  warehouse_id INTEGER NOT NULL,
  location_id INTEGER NOT NULL DEFAULT 0,
  product_id INTEGER NOT NULL,
  batch_no TEXT NOT NULL DEFAULT '',
  box_code_id INTEGER NOT NULL DEFAULT 0,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  weight DOUBLE PRECISION,
  avg_cost DOUBLE PRECISION,
  updated_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(warehouse_id, product_id, batch_no, location_id, box_code_id),
  FOREIGN KEY(warehouse_id) REFERENCES inv_warehouse(id),
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS inv_stock_txn (
  id BIGSERIAL PRIMARY KEY,
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
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inv_stock_txn_line (
  id BIGSERIAL PRIMARY KEY,
  txn_id INTEGER NOT NULL,
  line_no INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  unit_id INTEGER,
  qty DOUBLE PRECISION NOT NULL,
  base_qty DOUBLE PRECISION NOT NULL,
  weight DOUBLE PRECISION,
  batch_no TEXT,
  box_code_id INTEGER,
  location_id INTEGER,
  direction TEXT NOT NULL,
  amount DOUBLE PRECISION,
  remark TEXT,
  UNIQUE(txn_id, line_no),
  FOREIGN KEY(txn_id) REFERENCES inv_stock_txn(id),
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS inv_reservation (
  id BIGSERIAL PRIMARY KEY,
  warehouse_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  batch_no TEXT,
  qty DOUBLE PRECISION NOT NULL,
  source_doc_type TEXT NOT NULL,
  source_doc_id INTEGER NOT NULL,
  source_line_id INTEGER,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_routing (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  product_id INTEGER,
  version_no TEXT NOT NULL DEFAULT 'V1',
  status TEXT NOT NULL DEFAULT 'active',
  graph_json TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(code, version_no)
);

CREATE TABLE IF NOT EXISTS pd_flow_graph (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  routing_id INTEGER,
  graph_json TEXT NOT NULL DEFAULT '{}',
  version_no TEXT NOT NULL DEFAULT 'V1',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pd_routing_step (
  id BIGSERIAL PRIMARY KEY,
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
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  product_id INTEGER NOT NULL,
  warehouse_id INTEGER,
  batch_no TEXT,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  weight DOUBLE PRECISION,
  parent_box_id INTEGER,
  current_process_id INTEGER,
  current_step_id INTEGER,
  task_id INTEGER,
  work_order_id INTEGER,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(product_id) REFERENCES prd_product(id)
);

CREATE TABLE IF NOT EXISTS pd_process_issue (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL,
  step_id BIGINT NOT NULL DEFAULT 0,
  worker_id BIGINT NOT NULL DEFAULT 0,
  issue_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  returned_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  completed_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pd_process_issue_kg_chk CHECK (returned_kg + completed_kg <= issue_kg + 0.0001)
);

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_board ON pd_process_issue (board_id, process_id, status);
CREATE INDEX IF NOT EXISTS idx_pd_process_issue_worker ON pd_process_issue (worker_id, status);

CREATE TABLE IF NOT EXISTS pd_process_move (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  from_process_id BIGINT NOT NULL DEFAULT 0,
  from_step_id BIGINT NOT NULL DEFAULT 0,
  to_process_id BIGINT,
  to_step_id BIGINT,
  to_worker_id BIGINT NOT NULL DEFAULT 0,
  kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  move_kind TEXT NOT NULL DEFAULT 'next',
  issue_ids TEXT NOT NULL DEFAULT '',
  created_by BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_process_move_board ON pd_process_move (board_id, created_at);

CREATE TABLE IF NOT EXISTS pd_process_move_alloc (
  id BIGSERIAL PRIMARY KEY,
  move_id BIGINT NOT NULL,
  issue_id BIGINT NOT NULL,
  kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_process_move_alloc_move ON pd_process_move_alloc (move_id);

CREATE TABLE IF NOT EXISTS pd_board_process_yield (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL,
  input_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  output_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (board_id, process_id)
);

CREATE INDEX IF NOT EXISTS idx_pd_board_process_yield_trace ON pd_board_process_yield (trace_code, process_id);

CREATE TABLE IF NOT EXISTS pd_trace_process_yield (
  id BIGSERIAL PRIMARY KEY,
  trace_code TEXT NOT NULL,
  process_id BIGINT NOT NULL,
  input_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  output_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  board_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (trace_code, process_id)
);

CREATE TABLE IF NOT EXISTS pd_flow_event (
  id BIGSERIAL PRIMARY KEY,
  source_type TEXT NOT NULL,
  source_id INTEGER NOT NULL,
  from_step_id INTEGER,
  to_step_id INTEGER,
  trigger_action TEXT NOT NULL,
  trace_id TEXT,
  status TEXT NOT NULL DEFAULT 'ok',
  error TEXT,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_piecework_summary (
  id BIGSERIAL PRIMARY KEY,
  worker_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  weight DOUBLE PRECISION,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  source_report_ids TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sys_data_repair (
  id BIGSERIAL PRIMARY KEY,
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
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_production_task (
  id BIGSERIAL PRIMARY KEY,
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
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pd_production_task_item (
  id BIGSERIAL PRIMARY KEY,
  task_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  plan_qty DOUBLE PRECISION NOT NULL,
  plan_weight DOUBLE PRECISION,
  completed_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  FOREIGN KEY(task_id) REFERENCES pd_production_task(id)
);

CREATE TABLE IF NOT EXISTS pd_work_order (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  task_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL,
  routing_step_id INTEGER,
  status TEXT NOT NULL DEFAULT 'pending',
  plan_qty DOUBLE PRECISION,
  created_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(task_id) REFERENCES pd_production_task(id),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS pd_dispatch (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  work_order_id INTEGER NOT NULL,
  dispatch_type TEXT NOT NULL DEFAULT 'normal',
  worker_id INTEGER,
  team_id INTEGER,
  plan_qty DOUBLE PRECISION,
  status TEXT NOT NULL DEFAULT 'dispatched',
  dispatched_at TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(work_order_id) REFERENCES pd_work_order(id)
);

CREATE TABLE IF NOT EXISTS pd_report_work (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  dispatch_id INTEGER,
  work_order_id INTEGER,
  process_id INTEGER NOT NULL,
  worker_id INTEGER NOT NULL,
  report_type TEXT NOT NULL DEFAULT 'output',
  qty DOUBLE PRECISION NOT NULL,
  weight DOUBLE PRECISION,
  qty_net DOUBLE PRECISION,
  deduct_impurity DOUBLE PRECISION NOT NULL DEFAULT 0,
  deduct_water DOUBLE PRECISION NOT NULL DEFAULT 0,
  qc_result TEXT,
  status TEXT NOT NULL DEFAULT 'submitted',
  reported_at TEXT NOT NULL,
  scan_code TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS pay_process_wage_rate (
  id BIGSERIAL PRIMARY KEY,
  process_id INTEGER NOT NULL,
  product_id INTEGER,
  product_spec_id INTEGER,
  unit_id INTEGER,
  rate DOUBLE PRECISION NOT NULL,
  effective_from TEXT NOT NULL,
  effective_to TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(process_id) REFERENCES pd_process(id)
);

CREATE TABLE IF NOT EXISTS appr_task (
  id BIGSERIAL PRIMARY KEY,
  flow_id INTEGER,
  node_id INTEGER,
  doc_type TEXT NOT NULL,
  doc_id INTEGER NOT NULL,
  assignee_user_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  acted_at TEXT,
  comment TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

INSERT INTO schema_meta(key, value) VALUES ('version', '2')
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS erp_doc (
  id BIGSERIAL PRIMARY KEY,
  resource_key TEXT NOT NULL,
  doc_no TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  payload TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_erp_doc_key ON erp_doc(resource_key, is_deleted);

-- 采购 / 供应商
CREATE TABLE IF NOT EXISTS pur_supplier (
  id BIGSERIAL PRIMARY KEY,
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
  credit_limit DOUBLE PRECISION,
  currency TEXT NOT NULL DEFAULT 'CNY',
  tax_rate DOUBLE PRECISION,
  lead_time_days INTEGER,
  moq DOUBLE PRECISION,
  default_warehouse_id INTEGER,
  contact_json TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_supplier_license (
  id BIGSERIAL PRIMARY KEY,
  supplier_id INTEGER NOT NULL,
  license_type TEXT NOT NULL,
  license_no TEXT,
  expire_date TEXT,
  attachment_url TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  FOREIGN KEY(supplier_id) REFERENCES pur_supplier(id)
);

CREATE TABLE IF NOT EXISTS pur_supplier_supply_item (
  id BIGSERIAL PRIMARY KEY,
  supplier_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  is_preferred INTEGER NOT NULL DEFAULT 0,
  moq DOUBLE PRECISION,
  lead_time_days INTEGER,
  last_price DOUBLE PRECISION,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(supplier_id, product_id),
  FOREIGN KEY(supplier_id) REFERENCES pur_supplier(id)
);

CREATE TABLE IF NOT EXISTS pur_purchase_request (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  applicant_id INTEGER,
  title TEXT,
  qty DOUBLE PRECISION,
  status TEXT NOT NULL DEFAULT 'draft',
  need_date TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_purchase_request_line (
  id BIGSERIAL PRIMARY KEY,
  request_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  unit_id INTEGER,
  suggest_supplier_id INTEGER,
  FOREIGN KEY(request_id) REFERENCES pur_purchase_request(id)
);

CREATE TABLE IF NOT EXISTS pur_purchase_plan (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'draft',
  plan_date TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_purchase_plan_line (
  id BIGSERIAL PRIMARY KEY,
  plan_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  supplier_id INTEGER,
  request_line_id INTEGER,
  FOREIGN KEY(plan_id) REFERENCES pur_purchase_plan(id)
);

CREATE TABLE IF NOT EXISTS pur_purchase_inbound (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  supplier_id INTEGER NOT NULL,
  warehouse_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  biz_date TEXT NOT NULL,
  plan_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_purchase_inbound_line (
  id BIGSERIAL PRIMARY KEY,
  inbound_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  price DOUBLE PRECISION,
  amount DOUBLE PRECISION,
  batch_no TEXT,
  FOREIGN KEY(inbound_id) REFERENCES pur_purchase_inbound(id)
);

CREATE TABLE IF NOT EXISTS pur_incoming_qc (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  inbound_id INTEGER,
  supplier_id INTEGER,
  product_id INTEGER NOT NULL,
  qty_check DOUBLE PRECISION NOT NULL,
  qty_pass DOUBLE PRECISION NOT NULL DEFAULT 0,
  qty_fail DOUBLE PRECISION NOT NULL DEFAULT 0,
  result TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_purchase_return (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  supplier_id INTEGER NOT NULL,
  inbound_id INTEGER,
  warehouse_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  reason TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_purchase_return_line (
  id BIGSERIAL PRIMARY KEY,
  return_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  amount DOUBLE PRECISION,
  FOREIGN KEY(return_id) REFERENCES pur_purchase_return(id)
);

CREATE TABLE IF NOT EXISTS pur_supplier_price_history (
  id BIGSERIAL PRIMARY KEY,
  supplier_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  biz_date TEXT NOT NULL,
  source_doc_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pur_purchase_task (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  assignee_id INTEGER,
  product_id INTEGER,
  qty DOUBLE PRECISION,
  status TEXT NOT NULL DEFAULT 'open',
  due_date TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);



-- Tables from Ensure* (merged into baseline)

CREATE TABLE IF NOT EXISTS appr_affair_request (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  applicant_id INTEGER NOT NULL DEFAULT 1,
  title TEXT,
  content TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  queue_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS appr_expense_request (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  applicant_id INTEGER NOT NULL DEFAULT 1,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  category TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  queue_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS appr_queue (
  id BIGSERIAL PRIMARY KEY,
  category TEXT NOT NULL,
  doc_no TEXT,
  title TEXT NOT NULL,
  biz_type TEXT,
  biz_id INTEGER NOT NULL DEFAULT 0,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  applicant_id INTEGER NOT NULL DEFAULT 0,
  assignee_user_id INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'pending',
  remark TEXT,
  comment TEXT,
  payload_json TEXT,
  acted_at TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ast_asset_transfer (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  asset_id INTEGER NOT NULL,
  from_dept_id INTEGER,
  to_dept_id INTEGER,
  from_dept_name TEXT,
  to_dept_name TEXT,
  from_location TEXT,
  to_location TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  transferred_at TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ast_fixed_asset (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category_id INTEGER,
  dept_id INTEGER,
  dept_name TEXT,
  location_text TEXT,
  original_value DOUBLE PRECISION,
  net_value DOUBLE PRECISION,
  status TEXT NOT NULL DEFAULT 'active',
  purchase_date TEXT,
  useful_life_months INTEGER,
  residual_rate DOUBLE PRECISION DEFAULT 0.05,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ast_fixed_asset_category (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  parent_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS biz_audit_log (
  id BIGSERIAL PRIMARY KEY,
  biz_type TEXT NOT NULL,
  biz_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  reason TEXT,
  before_json TEXT,
  after_json TEXT,
  actor_user_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS biz_evidence (
  id BIGSERIAL PRIMARY KEY,
  biz_type TEXT NOT NULL,
  biz_id INTEGER NOT NULL,
  evidence_type TEXT NOT NULL,
  file_url TEXT NOT NULL,
  meta_json TEXT,
  uploaded_by INTEGER,
  uploaded_at TEXT NOT NULL DEFAULT NOW(),
  voided_at TEXT
);

CREATE TABLE IF NOT EXISTS crm_customer (
  id BIGSERIAL PRIMARY KEY,
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
  credit_limit DOUBLE PRECISION,
  logistics_remark TEXT,
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS crm_customer_import_batch (
  id BIGSERIAL PRIMARY KEY,
  file_name TEXT,
  imported_at TEXT NOT NULL,
  success_count INTEGER NOT NULL DEFAULT 0,
  fail_count INTEGER NOT NULL DEFAULT 0,
  fail_detail_json TEXT,
  created_by INTEGER
);

CREATE TABLE IF NOT EXISTS crm_follow_up (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  opportunity_id INTEGER,
  user_id INTEGER NOT NULL,
  follow_type TEXT NOT NULL DEFAULT 'visit',
  follow_at TEXT NOT NULL,
  content TEXT,
  next_remind_at TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crm_lead_assign (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  from_user_id INTEGER,
  to_user_id INTEGER NOT NULL,
  assigned_at TEXT NOT NULL,
  lock_flag INTEGER NOT NULL DEFAULT 0,
  remark TEXT
);

CREATE TABLE IF NOT EXISTS crm_lead_protect_rule (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  protect_days INTEGER NOT NULL DEFAULT 30,
  release_rule_json TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crm_lead_release_log (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  released_at TEXT NOT NULL,
  reason TEXT,
  to_public_sea INTEGER NOT NULL DEFAULT 1,
  from_user_id INTEGER,
  operator_user_id INTEGER
);

CREATE TABLE IF NOT EXISTS crm_opportunity (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  title TEXT,
  stage TEXT NOT NULL DEFAULT 'lead',
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  expected_date TEXT,
  owner_user_id INTEGER,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  converted_order_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crm_task_reminder (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  ref_type TEXT,
  ref_id INTEGER,
  remind_at TEXT NOT NULL,
  content TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fin_account_subject (
  id BIGSERIAL PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
  parent_id INTEGER, subject_type TEXT, status TEXT NOT NULL DEFAULT 'active');

CREATE TABLE IF NOT EXISTS fin_approval_item (
  id BIGSERIAL PRIMARY KEY, biz_type TEXT NOT NULL, biz_id INTEGER NOT NULL,
  doc_no TEXT, title TEXT, amount DOUBLE PRECISION, status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_arap_adjust (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, party_type TEXT NOT NULL,
  party_id INTEGER NOT NULL, amount DOUBLE PRECISION NOT NULL, direction TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', remark TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_cashier_reconcile (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, fund_account_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL, book_balance DOUBLE PRECISION NOT NULL, actual_balance DOUBLE PRECISION NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', remark TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_contract_profit (
  id BIGSERIAL PRIMARY KEY, contract_id INTEGER NOT NULL, revenue DOUBLE PRECISION NOT NULL DEFAULT 0,
  cost DOUBLE PRECISION NOT NULL DEFAULT 0, profit DOUBLE PRECISION NOT NULL DEFAULT 0, period TEXT,
  created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_cost_accounting (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, period TEXT NOT NULL,
  task_id INTEGER, product_id INTEGER, material_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  labor_cost DOUBLE PRECISION NOT NULL DEFAULT 0, overhead DOUBLE PRECISION NOT NULL DEFAULT 0, total_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_cost_allocation (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, source_amount DOUBLE PRECISION NOT NULL,
  alloc_json TEXT, status TEXT NOT NULL DEFAULT 'draft', revoked_from_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_cost_trace_line (
  id BIGSERIAL PRIMARY KEY, cost_id INTEGER NOT NULL, source_type TEXT NOT NULL,
  source_id INTEGER NOT NULL, amount DOUBLE PRECISION NOT NULL);

CREATE TABLE IF NOT EXISTS fin_fund_account (
  id BIGSERIAL PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
  currency TEXT NOT NULL DEFAULT 'CNY', balance DOUBLE PRECISION NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'active');

CREATE TABLE IF NOT EXISTS fin_fund_transfer (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE,
  from_account_id INTEGER NOT NULL, to_account_id INTEGER NOT NULL, amount DOUBLE PRECISION NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', remark TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_fx_settlement (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, currency TEXT NOT NULL,
  amount_fx DOUBLE PRECISION NOT NULL, rate DOUBLE PRECISION NOT NULL, amount_local DOUBLE PRECISION NOT NULL,
  fund_account_id INTEGER, status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_income_expense_detail (
  id BIGSERIAL PRIMARY KEY, entry_id INTEGER NOT NULL, category TEXT, amount DOUBLE PRECISION NOT NULL, remark TEXT);

CREATE TABLE IF NOT EXISTS fin_invoice (
  id BIGSERIAL PRIMARY KEY, invoice_no TEXT NOT NULL UNIQUE, direction TEXT NOT NULL,
  counterparty_id INTEGER, counterparty_name TEXT, amount DOUBLE PRECISION NOT NULL, tax DOUBLE PRECISION,
  status TEXT NOT NULL DEFAULT 'draft', biz_date TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_ledger_entry (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, account_id INTEGER, subject_id INTEGER,
  direction TEXT NOT NULL, amount DOUBLE PRECISION NOT NULL, biz_date TEXT NOT NULL, counterparty TEXT,
  source_doc_type TEXT, source_doc_id INTEGER, remark TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_miniprogram_bill (
  id BIGSERIAL PRIMARY KEY, bill_no TEXT NOT NULL UNIQUE, channel TEXT,
  amount DOUBLE PRECISION NOT NULL, status TEXT NOT NULL DEFAULT 'unpaid', order_id INTEGER,
  paid_at TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_month_close (
  id BIGSERIAL PRIMARY KEY, year INTEGER NOT NULL, month INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', closed_at TEXT, closed_by INTEGER, UNIQUE(year, month));

CREATE TABLE IF NOT EXISTS fin_payment_recognition (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, customer_id INTEGER NOT NULL,
  amount DOUBLE PRECISION NOT NULL, fund_account_id INTEGER, status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_prepay_prepaid (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, party_type TEXT NOT NULL,
  party_id INTEGER NOT NULL, direction TEXT NOT NULL, amount DOUBLE PRECISION NOT NULL, balance DOUBLE PRECISION NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_receipt_alert (
  id BIGSERIAL PRIMARY KEY, customer_id INTEGER NOT NULL, order_id INTEGER,
  due_date TEXT, overdue_days INTEGER, amount DOUBLE PRECISION, status TEXT NOT NULL DEFAULT 'open',
  handled_remark TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_receipt_writeoff (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, customer_id INTEGER NOT NULL,
  amount DOUBLE PRECISION NOT NULL, fund_account_id INTEGER, status TEXT NOT NULL DEFAULT 'draft',
  received_at TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_receipt_writeoff_line (
  id BIGSERIAL PRIMARY KEY, writeoff_id INTEGER NOT NULL, sales_order_id INTEGER NOT NULL, amount DOUBLE PRECISION NOT NULL);

CREATE TABLE IF NOT EXISTS fin_sales_return_finance (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, order_id INTEGER,
  amount DOUBLE PRECISION NOT NULL, status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_statement_cache (
  id BIGSERIAL PRIMARY KEY, code TEXT NOT NULL, period TEXT, title TEXT,
  content_json TEXT, generated_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_voucher (
  id BIGSERIAL PRIMARY KEY, doc_no TEXT NOT NULL UNIQUE, period TEXT, biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', summary TEXT, created_at TEXT NOT NULL DEFAULT NOW());

CREATE TABLE IF NOT EXISTS fin_voucher_line (
  id BIGSERIAL PRIMARY KEY, voucher_id INTEGER NOT NULL, subject_id INTEGER NOT NULL,
  debit DOUBLE PRECISION NOT NULL DEFAULT 0, credit DOUBLE PRECISION NOT NULL DEFAULT 0, remark TEXT);

CREATE TABLE IF NOT EXISTS hr_attendance_month_stat (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  year INTEGER NOT NULL,
  month INTEGER NOT NULL,
  work_days DOUBLE PRECISION NOT NULL DEFAULT 0,
  late_times INTEGER NOT NULL DEFAULT 0,
  ot_hours DOUBLE PRECISION NOT NULL DEFAULT 0,
  leave_days DOUBLE PRECISION NOT NULL DEFAULT 0,
  UNIQUE(employee_id, year, month)
);

CREATE TABLE IF NOT EXISTS hr_attendance_perf_summary (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  period TEXT NOT NULL,
  attendance_score DOUBLE PRECISION,
  perf_score DOUBLE PRECISION,
  summary_json TEXT,
  UNIQUE(employee_id, period)
);

CREATE TABLE IF NOT EXISTS hr_attendance_record (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  check_in_at TEXT,
  check_out_at TEXT,
  shift_id INTEGER,
  source TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(employee_id, biz_date)
);

CREATE TABLE IF NOT EXISTS hr_attendance_rule (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  shift_id INTEGER,
  late_minutes INTEGER NOT NULL DEFAULT 0,
  early_minutes INTEGER NOT NULL DEFAULT 0,
  rule_json TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_employee_journal (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_leave_request (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  employee_id INTEGER NOT NULL,
  leave_type TEXT NOT NULL,
  start_at TEXT NOT NULL,
  end_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_memo (
  id BIGSERIAL PRIMARY KEY,
  owner_user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  content TEXT,
  biz_date TEXT,
  scope_type TEXT NOT NULL DEFAULT 'hr',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_offboard (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  revoke_permission INTEGER NOT NULL DEFAULT 1,
  reason TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_onboard (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  role_ids_json TEXT,
  onboard_date TEXT,
  need_account INTEGER NOT NULL DEFAULT 1,
  login_name TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_overtime_patch (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  employee_id INTEGER NOT NULL,
  biz_type TEXT NOT NULL,
  biz_date TEXT NOT NULL,
  minutes INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_performance_result (
  id BIGSERIAL PRIMARY KEY,
  scheme_id INTEGER NOT NULL,
  employee_id INTEGER NOT NULL,
  period TEXT NOT NULL,
  score DOUBLE PRECISION,
  amount DOUBLE PRECISION,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_performance_scheme (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  scheme_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_shift (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  workshop_id INTEGER,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_tool_issue (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  seq_no INTEGER DEFAULT 1,
  employee_id INTEGER,
  employee_name TEXT,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  pending_return_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  ticket_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_tool_issue__hdr (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  seq_no INTEGER DEFAULT 1,
  employee_id INTEGER,
  employee_name TEXT,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  pending_return_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  ticket_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_tool_issue_line (
  id BIGSERIAL PRIMARY KEY,
  issue_id INTEGER NOT NULL,
  tool_item_id INTEGER NOT NULL,
  tool_name TEXT,
  issue_qty DOUBLE PRECISION DEFAULT 0,
  return_qty DOUBLE PRECISION DEFAULT 0,
  pending_return_qty DOUBLE PRECISION DEFAULT 0,
  UNIQUE(issue_id, tool_item_id)
);

CREATE TABLE IF NOT EXISTS hr_tool_item (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS hr_visit_record (
  id BIGSERIAL PRIMARY KEY,
  employee_id INTEGER NOT NULL,
  customer_id INTEGER,
  visit_at TEXT NOT NULL,
  content TEXT,
  location TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_assemble_split (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  biz_type TEXT NOT NULL DEFAULT 'assemble',
  warehouse_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_assemble_split_line (
  id BIGSERIAL PRIMARY KEY,
  header_id INTEGER NOT NULL,
  role_type TEXT NOT NULL DEFAULT 'child',
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS inv_consume (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  warehouse_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inv_consume_line (
  id BIGSERIAL PRIMARY KEY,
  consume_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  batch_no TEXT
);

CREATE TABLE IF NOT EXISTS inv_in_transit (
  id BIGSERIAL PRIMARY KEY,
  product_id INTEGER NOT NULL,
  warehouse_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  transit_type TEXT NOT NULL DEFAULT 'purchase',
  source_doc_type TEXT NOT NULL DEFAULT '',
  source_doc_id INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_inbound_qc (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  stock_txn_id INTEGER,
  product_id INTEGER NOT NULL,
  qty_check DOUBLE PRECISION NOT NULL DEFAULT 0,
  qty_pass DOUBLE PRECISION NOT NULL DEFAULT 0,
  qty_fail DOUBLE PRECISION NOT NULL DEFAULT 0,
  result TEXT,
  inspector_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inv_material_to_payable (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  consume_txn_id INTEGER,
  supplier_id INTEGER,
  product_id INTEGER,
  qty DOUBLE PRECISION,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_price_adjust (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  product_id INTEGER NOT NULL,
  old_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  new_price DOUBLE PRECISION NOT NULL,
  effective_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_sales_peel_return (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  sales_order_id INTEGER,
  product_id INTEGER NOT NULL,
  peel_qty DOUBLE PRECISION NOT NULL,
  weight DOUBLE PRECISION,
  warehouse_id INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_stock_alert_rule (
  id BIGSERIAL PRIMARY KEY,
  product_id INTEGER,
  warehouse_id INTEGER,
  alert_type TEXT NOT NULL,
  min_qty DOUBLE PRECISION,
  max_qty DOUBLE PRECISION,
  is_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inv_stocktake (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  stocktake_type TEXT NOT NULL DEFAULT 'warehouse',
  warehouse_id INTEGER,
  workshop_id INTEGER,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inv_stocktake_line (
  id BIGSERIAL PRIMARY KEY,
  stocktake_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  book_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  count_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  diff_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  batch_no TEXT,
  location_id INTEGER
);

CREATE TABLE IF NOT EXISTS inv_transfer (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  from_warehouse_id INTEGER NOT NULL,
  to_warehouse_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inv_transfer_line (
  id BIGSERIAL PRIMARY KEY,
  transfer_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  base_qty DOUBLE PRECISION NOT NULL,
  batch_no TEXT
);

CREATE TABLE IF NOT EXISTS inv_weighbridge (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  location TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notify_inbox (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  body TEXT,
  event_key TEXT,
  task_id INTEGER,
  payload_json TEXT,
  read_at TEXT,
  acked_at TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notify_outbox (
  id BIGSERIAL PRIMARY KEY,
  topic TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_retry_at TEXT,
  dedupe_key TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT NOW(),
  sent_at TEXT
);

CREATE TABLE IF NOT EXISTS pay_commission_calc (
			id BIGSERIAL PRIMARY KEY,
			rule_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			period TEXT NOT NULL,
			base_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			commission_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			source_doc_refs TEXT,
			created_at TEXT NOT NULL DEFAULT NOW()
		);

CREATE TABLE IF NOT EXISTS pay_payroll_adjust (
			id BIGSERIAL PRIMARY KEY,
			sheet_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			adjust_type TEXT NOT NULL,
			amount DOUBLE PRECISION NOT NULL,
			reason TEXT,
			created_at TEXT NOT NULL DEFAULT NOW()
		);

CREATE TABLE IF NOT EXISTS pay_payroll_calc_log (
			id BIGSERIAL PRIMARY KEY,
			doc_no TEXT,
			period_ym TEXT,
			sheet_id INTEGER,
			status TEXT,
			summary_json TEXT,
			created_at TEXT NOT NULL DEFAULT NOW()
		);

CREATE TABLE IF NOT EXISTS pay_payroll_sheet (
			id BIGSERIAL PRIMARY KEY,
			doc_no TEXT NOT NULL UNIQUE,
			period_year INTEGER NOT NULL,
			period_month INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			workshop_id INTEGER,
			calc_at TEXT,
			paid_at TEXT,
			remark TEXT,
			created_by INTEGER,
			created_at TEXT NOT NULL DEFAULT NOW(),
			UNIQUE(period_year, period_month)
		);

CREATE TABLE IF NOT EXISTS pay_payroll_sheet_line (
			id BIGSERIAL PRIMARY KEY,
			sheet_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			emp_type TEXT,
			piece_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			attendance_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			commission_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			adjust_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			total_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			UNIQUE(sheet_id, employee_id)
		);

CREATE TABLE IF NOT EXISTS pay_sales_commission_rule (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			rule_json TEXT NOT NULL DEFAULT '{}',
			effective_from TEXT NOT NULL,
			effective_to TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT NOW()
		);

CREATE TABLE IF NOT EXISTS pay_worker_profile (
			id BIGSERIAL PRIMARY KEY,
			employee_id INTEGER NOT NULL UNIQUE,
			pay_type TEXT NOT NULL DEFAULT 'piece',
			monthly_base DOUBLE PRECISION NOT NULL DEFAULT 0,
			bank_account TEXT,
			tax_no TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT NOW()
		);

CREATE TABLE IF NOT EXISTS pd_bom (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL,
  product_id INTEGER NOT NULL,
  version_no TEXT NOT NULL DEFAULT 'V1',
  name TEXT,
  is_auto_generated INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(code, version_no)
);

CREATE TABLE IF NOT EXISTS pd_bom_line (
  id BIGSERIAL PRIMARY KEY,
  bom_id INTEGER NOT NULL,
  component_product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  scrap_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  remark TEXT
);

CREATE TABLE IF NOT EXISTS pd_consignment_order (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER,
  product_id INTEGER,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  progress TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_cost_hide_policy (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  name TEXT,
  field_scope TEXT NOT NULL DEFAULT '[]',
  is_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(role_id)
);

CREATE TABLE IF NOT EXISTS pd_drawing_link (
  id BIGSERIAL PRIMARY KEY,
  drawing_code TEXT,
  drawing_name TEXT,
  drawing_id INTEGER,
  task_id INTEGER,
  work_order_id INTEGER,
  process_id INTEGER,
  file_url TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_material_requisition (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  work_order_id INTEGER,
  dispatch_id INTEGER,
  warehouse_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pd_material_requisition_line (
  id BIGSERIAL PRIMARY KEY,
  requisition_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  base_qty DOUBLE PRECISION NOT NULL,
  batch_no TEXT
);

CREATE TABLE IF NOT EXISTS pd_mrp_result (
  id BIGSERIAL PRIMARY KEY,
  run_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  demand_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  supply_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  shortage_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  suggest_action TEXT
);

CREATE TABLE IF NOT EXISTS pd_mrp_run (
  id BIGSERIAL PRIMARY KEY,
  run_no TEXT NOT NULL UNIQUE,
  run_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'done',
  params_json TEXT,
  remark TEXT
);

CREATE TABLE IF NOT EXISTS pd_outsource_order (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  supplier_id INTEGER,
  process_id INTEGER,
  product_id INTEGER,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  received_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_piece_issue_line (
  id BIGSERIAL PRIMARY KEY,
  sheet_id INTEGER NOT NULL,
  seq_no INTEGER NOT NULL DEFAULT 1,
  employee_id INTEGER,
  employee_name TEXT,
  process_id INTEGER,
  process_name TEXT,
  process_kind TEXT DEFAULT 'piece',
  unit_price DOUBLE PRECISION DEFAULT 0,
  qty DOUBLE PRECISION DEFAULT 0,
  reject_qty DOUBLE PRECISION DEFAULT 0,
  qty_total DOUBLE PRECISION DEFAULT 0,
  amount DOUBLE PRECISION DEFAULT 0,
  remark TEXT
);

CREATE TABLE IF NOT EXISTS pd_piece_issue_sheet (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_process_return (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  box_code TEXT NOT NULL,
  process_id INTEGER,
  step_id INTEGER,
  warehouse_id INTEGER NOT NULL,
  return_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  reason TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  applicant_user_id INTEGER,
  foreman_user_id INTEGER,
  warehouse_user_id INTEGER,
  current_assignee_user_id INTEGER,
  report_work_id INTEGER,
  stock_txn_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  posted_at TEXT,
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pd_process_return_log (
  id BIGSERIAL PRIMARY KEY,
  return_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  from_user_id INTEGER,
  to_user_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_qc_order (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  qc_type TEXT NOT NULL DEFAULT 'process',
  source_doc_type TEXT,
  source_doc_id INTEGER,
  product_id INTEGER,
  process_id INTEGER,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  result TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_rework_order (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  source_qc_id INTEGER,
  task_id INTEGER,
  process_id INTEGER,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_scrap_record (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  task_id INTEGER,
  process_id INTEGER,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  weight DOUBLE PRECISION,
  disposition TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_shift (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL,
  workshop_id INTEGER NOT NULL DEFAULT 1,
  biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_shift_member (
  id BIGSERIAL PRIMARY KEY,
  shift_id INTEGER NOT NULL,
  employee_id INTEGER NOT NULL,
  process_id INTEGER NOT NULL DEFAULT 0,
  UNIQUE(shift_id, employee_id, process_id)
);

CREATE TABLE IF NOT EXISTS pd_task_merge (
  id BIGSERIAL PRIMARY KEY,
  merge_no TEXT NOT NULL UNIQUE,
  title TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  result_task_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pd_task_merge_line (
  id BIGSERIAL PRIMARY KEY,
  merge_id INTEGER NOT NULL,
  source_doc_type TEXT NOT NULL DEFAULT 'production_task',
  source_doc_id INTEGER NOT NULL,
  task_id INTEGER
);

CREATE TABLE IF NOT EXISTS prd_product_app_sort (
  id BIGSERIAL PRIMARY KEY,
  product_id INTEGER NOT NULL,
  channel TEXT NOT NULL DEFAULT 'app',
  sort_no INTEGER NOT NULL DEFAULT 0,
  is_visible INTEGER NOT NULL DEFAULT 1,
  UNIQUE(product_id, channel)
);

CREATE TABLE IF NOT EXISTS prd_product_spec (
  id BIGSERIAL PRIMARY KEY,
  product_id INTEGER NOT NULL,
  spec_code TEXT NOT NULL,
  routing_id INTEGER,
  process_wage_bind_json TEXT,
  remark TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(product_id, spec_code)
);

CREATE TABLE IF NOT EXISTS pur_farmer (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  mobile TEXT,
  origin TEXT,
  trace_code TEXT,
  trace_code_prefix TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_farmer_settlement (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id INTEGER NOT NULL,
  weigh_ticket_id INTEGER,
  biz_date TEXT NOT NULL,
  net_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  unit_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pur_grade_price (
  id BIGSERIAL PRIMARY KEY,
  grade TEXT NOT NULL UNIQUE,
  unit_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS pur_inbound_arrival (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id INTEGER NOT NULL,
  origin TEXT,
  variety TEXT,
  estimate_weight DOUBLE PRECISION DEFAULT 0,
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
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_trace_batch_code (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  seq_no INTEGER NOT NULL,
  lot_no TEXT NOT NULL DEFAULT '01',
  status TEXT NOT NULL DEFAULT 'available',
  weigh_ticket_id INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  used_at TEXT
);

CREATE TABLE IF NOT EXISTS pur_trace_lot (
  id BIGSERIAL PRIMARY KEY,
  trace_code TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  batch_no TEXT NOT NULL,
  farmer_id INTEGER NOT NULL,
  grade TEXT,
  arrival_id INTEGER,
  weigh_ticket_id INTEGER,
  channel TEXT,
  source_type TEXT,
  net_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  payload_canonical TEXT NOT NULL,
  signature TEXT NOT NULL,
  algo TEXT NOT NULL DEFAULT 'HmacSHA256_v1',
  replaces_trace_code TEXT,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pur_weigh_ticket (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id INTEGER NOT NULL,
  channel TEXT NOT NULL DEFAULT 'internal',
  ticket_template TEXT,
  product_id INTEGER NOT NULL DEFAULT 1,
  variety TEXT,
  gross_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  deduct_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  deduct_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  net_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  qc_result TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  trace_code TEXT,
  origin TEXT,
  biz_date TEXT NOT NULL,
  source_type TEXT NOT NULL DEFAULT 'self',
  image_url TEXT,
  box_code TEXT,
  warehouse_id INTEGER,
  plate_no TEXT,
  freight_fee DOUBLE PRECISION NOT NULL DEFAULT 0,
  loading_fee DOUBLE PRECISION NOT NULL DEFAULT 0,
  weigh_fee DOUBLE PRECISION NOT NULL DEFAULT 0,
  unit_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  confirmed_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pur_weigh_variety (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  sort_no INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  default_product_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS rpt_dashboard_widget (
  id BIGSERIAL PRIMARY KEY,
  dashboard_key TEXT NOT NULL,
  title TEXT NOT NULL,
  metric_key TEXT,
  layout_json TEXT,
  refresh_sec INTEGER NOT NULL DEFAULT 60,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS rpt_report_definition (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  report_type TEXT,
  query_config_json TEXT,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS rpt_report_snapshot (
  id BIGSERIAL PRIMARY KEY,
  report_code TEXT NOT NULL,
  biz_date TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT NOW(),
  UNIQUE(report_code, biz_date)
);

CREATE TABLE IF NOT EXISTS sl_contract (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER NOT NULL,
  title TEXT,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  signed_at TEXT,
  expire_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sl_cost_budget (
  id BIGSERIAL PRIMARY KEY,
  order_id INTEGER NOT NULL,
  material_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  labor_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  other_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  sale_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  margin DOUBLE PRECISION NOT NULL DEFAULT 0,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_delivery_approval (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  order_id INTEGER NOT NULL,
  pre_shipment_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  warehouse_id INTEGER,
  shipped_at TEXT,
  logistics_no TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_delivery_line (
  id BIGSERIAL PRIMARY KEY,
  delivery_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  weight DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS sl_inquiry (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  customer_id INTEGER NOT NULL,
  owner_user_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  source TEXT NOT NULL DEFAULT 'sales',
  expire_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sl_inquiry_line (
  id BIGSERIAL PRIMARY KEY,
  inquiry_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  weight DOUBLE PRECISION,
  quote_price DOUBLE PRECISION,
  cost_ref DOUBLE PRECISION,
  remark TEXT
);

CREATE TABLE IF NOT EXISTS sl_order_change_log (
  id BIGSERIAL PRIMARY KEY,
  order_id INTEGER NOT NULL,
  change_type TEXT NOT NULL,
  before_json TEXT,
  after_json TEXT,
  reason TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_outbound_settle (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  biz_date TEXT NOT NULL,
  product_id INTEGER,
  product_name TEXT,
  plate_no TEXT,
  driver_name TEXT,
  trace_code TEXT,
  produce_date TEXT,
  qty DOUBLE PRECISION DEFAULT 0,
  weight DOUBLE PRECISION DEFAULT 0,
  unit TEXT DEFAULT 'kg',
  freight_fee DOUBLE PRECISION DEFAULT 0,
  loading_fee DOUBLE PRECISION DEFAULT 0,
  weigh_fee DOUBLE PRECISION DEFAULT 0,
  unit_price DOUBLE PRECISION DEFAULT 0,
  goods_amount DOUBLE PRECISION DEFAULT 0,
  amount DOUBLE PRECISION DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sl_pre_shipment (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  order_id INTEGER NOT NULL,
  plan_ship_date TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  reserved INTEGER NOT NULL DEFAULT 0,
  warehouse_id INTEGER NOT NULL DEFAULT 3,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_pre_shipment_line (
  id BIGSERIAL PRIMARY KEY,
  pre_shipment_id INTEGER NOT NULL,
  order_line_id INTEGER,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS sl_price_lock (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  lock_price DOUBLE PRECISION NOT NULL,
  effective_from TEXT NOT NULL,
  effective_to TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_print_log (
  id BIGSERIAL PRIMARY KEY,
  doc_type TEXT NOT NULL,
  doc_id INTEGER NOT NULL,
  doc_no TEXT,
  template_code TEXT,
  printed_by INTEGER,
  printed_at TEXT NOT NULL DEFAULT NOW(),
  payload_json TEXT
);

CREATE TABLE IF NOT EXISTS sl_quote_calculator_result (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL DEFAULT 1,
  base_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
  margin_rate DOUBLE PRECISION NOT NULL DEFAULT 0.2,
  quote_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_quote_history (
  id BIGSERIAL PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  quoted_at TEXT NOT NULL,
  inquiry_id INTEGER,
  order_id INTEGER
);

CREATE TABLE IF NOT EXISTS sl_sales_bom (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  order_id INTEGER,
  product_id INTEGER NOT NULL,
  name TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_sales_bom_line (
  id BIGSERIAL PRIMARY KEY,
  bom_id INTEGER NOT NULL,
  material_product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  scrap_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  remark TEXT
);

CREATE TABLE IF NOT EXISTS sl_sales_order (
  id BIGSERIAL PRIMARY KEY,
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
  total_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  remark TEXT,
  created_by INTEGER,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sl_sales_order_line (
  id BIGSERIAL PRIMARY KEY,
  order_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  weight DOUBLE PRECISION,
  price DOUBLE PRECISION NOT NULL,
  amount DOUBLE PRECISION NOT NULL,
  delivered_qty DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sl_sales_rank_config (
  id BIGSERIAL PRIMARY KEY,
  metric TEXT NOT NULL,
  period TEXT NOT NULL DEFAULT 'month',
  top_n INTEGER NOT NULL DEFAULT 10,
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sl_self_order_rule (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  min_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  allow_products_json TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sys_announcement (
			id BIGSERIAL PRIMARY KEY,
			title TEXT, content TEXT, status TEXT DEFAULT 'draft', published_at TEXT,
			created_by INTEGER, is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_approval_flow (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, name TEXT, doc_type TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_approval_flow_node (
			id BIGSERIAL PRIMARY KEY,
			flow_id INTEGER NOT NULL, seq_no INTEGER DEFAULT 1, node_name TEXT,
			approver_role TEXT, approver_user_id INTEGER, require_all INTEGER DEFAULT 0
		);

CREATE TABLE IF NOT EXISTS sys_batch_payroll_job (
			id BIGSERIAL PRIMARY KEY,
			doc_no TEXT, period_ym TEXT, workshop_id INTEGER, status TEXT DEFAULT 'draft',
			result_msg TEXT, created_by INTEGER, applied_at TEXT, is_deleted INTEGER DEFAULT 0, created_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_batch_price_job (
			id BIGSERIAL PRIMARY KEY,
			doc_no TEXT, target_type TEXT, adjust_type TEXT, adjust_value DOUBLE PRECISION,
			scope_json TEXT, status TEXT DEFAULT 'draft', result_msg TEXT,
			created_by INTEGER, applied_at TEXT, is_deleted INTEGER DEFAULT 0, created_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_carrier (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, name TEXT, contact TEXT, phone TEXT, remark TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_course (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, title TEXT, category TEXT, content TEXT, duration_min INTEGER, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_document (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, title TEXT, category TEXT, content TEXT, file_url TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_drawing (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, title TEXT, product_id INTEGER, version_no TEXT, file_url TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_formula (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, name TEXT, scope TEXT, expression TEXT, remark TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_knowledge (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, title TEXT, category TEXT, content TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_logistics_carrier (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  contact TEXT,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS sys_logistics_track (
  id BIGSERIAL PRIMARY KEY,
  track_no TEXT NOT NULL,
  carrier_id INTEGER,
  order_id INTEGER,
  status TEXT NOT NULL DEFAULT 'in_transit',
  location TEXT,
  updated_at TEXT NOT NULL DEFAULT NOW(),
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sys_memo (
			id BIGSERIAL PRIMARY KEY,
			title TEXT, content TEXT, owner_id INTEGER, status TEXT DEFAULT 'open',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_notify_rule (
  id BIGSERIAL PRIMARY KEY,
  event_key TEXT NOT NULL UNIQUE,
  channel TEXT,
  template_text TEXT,
  receivers_json TEXT
);

CREATE TABLE IF NOT EXISTS sys_personnel_transfer (
			id BIGSERIAL PRIMARY KEY,
			doc_no TEXT, employee_id INTEGER, from_dept_id INTEGER, to_dept_id INTEGER,
			from_workshop_id INTEGER, to_workshop_id INTEGER, reason TEXT,
			status TEXT DEFAULT 'draft', effective_date TEXT, confirmed_at TEXT, created_by INTEGER,
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_print_template (
			id BIGSERIAL PRIMARY KEY,
			code TEXT, name TEXT, doc_type TEXT, content TEXT, status TEXT DEFAULT 'active',
			is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_reminder (
			id BIGSERIAL PRIMARY KEY,
			title TEXT, content TEXT, remind_at TEXT, target_user_id INTEGER, target_role TEXT,
			status TEXT DEFAULT 'open', is_deleted INTEGER DEFAULT 0, created_at TEXT, updated_at TEXT
		);

CREATE TABLE IF NOT EXISTS sys_setting (
			setting_key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			updated_at TEXT,
			updated_by INTEGER
		);

CREATE TABLE IF NOT EXISTS wf_task (
  id BIGSERIAL PRIMARY KEY,
  event_key TEXT NOT NULL,
  biz_type TEXT NOT NULL,
  biz_id INTEGER NOT NULL,
  doc_no TEXT,
  trace_code TEXT,
  from_role TEXT,
  to_role TEXT NOT NULL,
  assignee_user_id INTEGER,
  payload_json TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  dedupe_key TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT NOW(),
  done_at TEXT
);

CREATE TABLE IF NOT EXISTS wf_ticket (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  category_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  applicant_user_id INTEGER NOT NULL,
  current_assignee_user_id INTEGER,
  biz_type TEXT,
  biz_id INTEGER,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  closed_at TEXT
);

CREATE TABLE IF NOT EXISTS wf_ticket_category (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wf_ticket_category_handler (
  id BIGSERIAL PRIMARY KEY,
  category_id INTEGER NOT NULL,
  handler_type TEXT NOT NULL,
  handler_ref INTEGER NOT NULL,
  UNIQUE(category_id, handler_type, handler_ref)
);

CREATE TABLE IF NOT EXISTS wf_ticket_log (
  id BIGSERIAL PRIMARY KEY,
  ticket_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  from_user_id INTEGER,
  to_user_id INTEGER,
  comment TEXT,
  created_at TEXT NOT NULL DEFAULT NOW()
);


-- Columns from Ensure* ALTER ADD COLUMN
ALTER TABLE hr_offboard ADD COLUMN IF NOT EXISTS offboard_date TEXT;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS destroy_reason TEXT;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS destroyed_at TEXT;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS destroyed_by INTEGER;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS receive_date TEXT;
ALTER TABLE pay_process_wage_rate ADD COLUMN IF NOT EXISTS rate_unit TEXT DEFAULT 'kg';
ALTER TABLE pd_piecework_summary ADD COLUMN IF NOT EXISTS input_weight DOUBLE PRECISION;
ALTER TABLE pd_piecework_summary ADD COLUMN IF NOT EXISTS loss DOUBLE PRECISION;
ALTER TABLE pd_piecework_summary ADD COLUMN IF NOT EXISTS output_weight DOUBLE PRECISION;
ALTER TABLE pd_piecework_summary ADD COLUMN IF NOT EXISTS pay_evidence_url TEXT;
ALTER TABLE pd_piecework_summary ADD COLUMN IF NOT EXISTS transfer_no TEXT;
ALTER TABLE pd_piecework_summary ADD COLUMN IF NOT EXISTS utilization DOUBLE PRECISION;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS bag_qty DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS input_weight DOUBLE PRECISION;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS loss DOUBLE PRECISION;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS operator_employee_id INTEGER;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS output_weight DOUBLE PRECISION;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS process_qc_result TEXT;
ALTER TABLE pd_report_work ADD COLUMN IF NOT EXISTS utilization DOUBLE PRECISION;
ALTER TABLE pd_routing_step ADD COLUMN IF NOT EXISTS checkpoint_bind_warehouse INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pd_scrap_record ADD COLUMN IF NOT EXISTS scrap_type TEXT;
ALTER TABLE pur_farmer ADD COLUMN IF NOT EXISTS default_unit_price DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS paid_at TEXT;
ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS pay_evidence_url TEXT;
ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS transfer_no TEXT;
ALTER TABLE pur_inbound_arrival ADD COLUMN IF NOT EXISTS pass_rate DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pur_inbound_arrival ADD COLUMN IF NOT EXISTS receive_address TEXT;
ALTER TABLE pur_inbound_arrival ADD COLUMN IF NOT EXISTS reject_weight DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS reserved_at TEXT;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS reserved_by INTEGER;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS arrival_id INTEGER;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS bag_qty DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS batch_no TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS cold_store_type TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS confirmed_at TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS confirmed_by INTEGER;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS confirmed_snapshot_json TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS freight_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS grade TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS loading_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS ocr_draft_json TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS party_mobile TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS party_name TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS pass_rate DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS plate_no TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS purchase_completed_at TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS receive_address TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS receive_kind TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS reject_weight DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS settle_amount DOUBLE PRECISION DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS unit_price DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS weigh_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS weighbridge_id INTEGER;
ALTER TABLE wf_ticket_category ADD COLUMN IF NOT EXISTS biz_hint TEXT;
ALTER TABLE wf_ticket_category ADD COLUMN IF NOT EXISTS form_schema_json TEXT;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.0', 'postgresql baseline', '')
ON CONFLICT (version) DO NOTHING;
