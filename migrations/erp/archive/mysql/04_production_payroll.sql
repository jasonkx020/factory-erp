USE erp_factory;

-- ---------------------------------------------------------------------------
-- 生产管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pd_process (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  code                VARCHAR(64) NOT NULL,
  name                VARCHAR(128) NOT NULL,
  process_type        VARCHAR(32) NULL COMMENT 'wash/peel/cut/core/dice/bag/other',
  is_piecework        TINYINT(1) NOT NULL DEFAULT 0,
  is_handover_point   TINYINT(1) NOT NULL DEFAULT 0 COMMENT '收货卡点',
  status              VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted          TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_process_code (code)
) ENGINE=InnoDB COMMENT='工序';

-- 补全 IAM 工序范围外键
ALTER TABLE iam_role_process_scope
  ADD CONSTRAINT fk_rps_process FOREIGN KEY (process_id) REFERENCES pd_process(id);
ALTER TABLE iam_user_process_scope
  ADD CONSTRAINT fk_ups_process FOREIGN KEY (process_id) REFERENCES pd_process(id);

CREATE TABLE IF NOT EXISTS pd_process_price (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  process_id      BIGINT NOT NULL,
  product_spec_id BIGINT NULL,
  unit_id         BIGINT NULL,
  price           DECIMAL(18,4) NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_pp_process (process_id),
  CONSTRAINT fk_pp_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工序工价入口（生产侧）';

CREATE TABLE IF NOT EXISTS pd_routing (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  product_id    BIGINT NULL,
  version_no    VARCHAR(32) NOT NULL DEFAULT 'V1',
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_routing_code_ver (code, version_no)
) ENGINE=InnoDB COMMENT='工艺流程';

CREATE TABLE IF NOT EXISTS pd_routing_step (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  routing_id            BIGINT NOT NULL,
  seq_no                INT NOT NULL,
  process_id            BIGINT NOT NULL,
  is_inbound_checkpoint TINYINT(1) NOT NULL DEFAULT 0,
  is_qc_required        TINYINT(1) NOT NULL DEFAULT 0,
  workshop_id           BIGINT NULL,
  UNIQUE KEY uk_routing_step (routing_id, seq_no),
  CONSTRAINT fk_rs_routing FOREIGN KEY (routing_id) REFERENCES pd_routing(id),
  CONSTRAINT fk_rs_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工艺步骤';

CREATE TABLE IF NOT EXISTS pd_bom (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  code              VARCHAR(64) NOT NULL,
  product_id        BIGINT NOT NULL,
  version_no        VARCHAR(32) NOT NULL DEFAULT 'V1',
  is_auto_generated TINYINT(1) NOT NULL DEFAULT 0,
  status            VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_bom (code, version_no)
) ENGINE=InnoDB COMMENT='BOM头';

CREATE TABLE IF NOT EXISTS pd_bom_line (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  bom_id                BIGINT NOT NULL,
  component_product_id  BIGINT NOT NULL,
  qty                   DECIMAL(18,4) NOT NULL,
  unit_id               BIGINT NULL,
  scrap_rate            DECIMAL(8,4) NOT NULL DEFAULT 0,
  CONSTRAINT fk_bl_bom FOREIGN KEY (bom_id) REFERENCES pd_bom(id)
) ENGINE=InnoDB COMMENT='BOM行';

CREATE TABLE IF NOT EXISTS pd_production_task (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  source_type   VARCHAR(16) NOT NULL DEFAULT 'manual' COMMENT 'sales/manual/merge',
  status        VARCHAR(16) NOT NULL DEFAULT 'pending',
  plan_start    DATETIME(3) NULL,
  plan_end      DATETIME(3) NULL,
  routing_id    BIGINT NULL,
  workshop_id   BIGINT NULL,
  org_id        BIGINT NULL,
  owner_user_id BIGINT NULL,
  remark        VARCHAR(512) NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_task_no (doc_no)
) ENGINE=InnoDB COMMENT='生产任务单';

CREATE TABLE IF NOT EXISTS pd_production_task_item (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  task_id         BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  plan_qty        DECIMAL(18,4) NOT NULL,
  plan_weight     DECIMAL(18,4) NULL,
  completed_qty   DECIMAL(18,4) NOT NULL DEFAULT 0,
  CONSTRAINT fk_pti_task FOREIGN KEY (task_id) REFERENCES pd_production_task(id)
) ENGINE=InnoDB COMMENT='任务单商品行（一单多商品）';

CREATE TABLE IF NOT EXISTS pd_task_merge (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  merge_no      VARCHAR(64) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_merge_no (merge_no)
) ENGINE=InnoDB COMMENT='多单整合';

CREATE TABLE IF NOT EXISTS pd_task_merge_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  merge_id        BIGINT NOT NULL,
  source_doc_type VARCHAR(64) NOT NULL,
  source_doc_id   BIGINT NOT NULL,
  task_id         BIGINT NULL,
  CONSTRAINT fk_tml_merge FOREIGN KEY (merge_id) REFERENCES pd_task_merge(id)
) ENGINE=InnoDB COMMENT='多单整合来源';

CREATE TABLE IF NOT EXISTS pd_work_order (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  task_id           BIGINT NOT NULL,
  process_id        BIGINT NOT NULL,
  routing_step_id   BIGINT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'pending',
  plan_qty          DECIMAL(18,4) NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_wo_no (doc_no),
  CONSTRAINT fk_wo_task FOREIGN KEY (task_id) REFERENCES pd_production_task(id),
  CONSTRAINT fk_wo_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工单';

CREATE TABLE IF NOT EXISTS pd_dispatch (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  work_order_id   BIGINT NOT NULL,
  dispatch_type   VARCHAR(16) NOT NULL DEFAULT 'normal' COMMENT 'normal/flex',
  worker_id       BIGINT NULL COMMENT 'hr_employee.id',
  team_id         BIGINT NULL,
  plan_qty        DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'dispatched',
  dispatched_at   DATETIME(3) NULL,
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_dispatch_no (doc_no),
  CONSTRAINT fk_dp_wo FOREIGN KEY (work_order_id) REFERENCES pd_work_order(id)
) ENGINE=InnoDB COMMENT='派工/灵活派发';

CREATE TABLE IF NOT EXISTS pd_material_requisition (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  work_order_id   BIGINT NULL,
  dispatch_id     BIGINT NULL,
  warehouse_id    BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_mr_no (doc_no)
) ENGINE=InnoDB COMMENT='联动领料头';

CREATE TABLE IF NOT EXISTS pd_material_requisition_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  requisition_id  BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  qty             DECIMAL(18,4) NOT NULL,
  base_qty        DECIMAL(18,4) NOT NULL,
  batch_no        VARCHAR(64) NULL,
  CONSTRAINT fk_mrl_hdr FOREIGN KEY (requisition_id) REFERENCES pd_material_requisition(id)
) ENGINE=InnoDB COMMENT='领料行';

CREATE TABLE IF NOT EXISTS pd_report_work (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  dispatch_id     BIGINT NULL,
  work_order_id   BIGINT NULL,
  process_id      BIGINT NOT NULL,
  worker_id       BIGINT NOT NULL,
  report_type     VARCHAR(16) NOT NULL DEFAULT 'output' COMMENT 'output/handover',
  qty             DECIMAL(18,4) NOT NULL,
  weight          DECIMAL(18,4) NULL,
  qty_net         DECIMAL(18,4) NULL,
  deduct_impurity DECIMAL(18,4) NOT NULL DEFAULT 0,
  deduct_water    DECIMAL(18,4) NOT NULL DEFAULT 0,
  qc_result       VARCHAR(16) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'submitted',
  reported_at     DATETIME(3) NOT NULL,
  scan_code       VARCHAR(128) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rw_no (doc_no),
  KEY idx_rw_worker_date (worker_id, reported_at),
  CONSTRAINT fk_rw_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='扫码报工（含工序收货交接）';

CREATE TABLE IF NOT EXISTS pd_piecework_summary (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  worker_id     BIGINT NOT NULL,
  process_id    BIGINT NOT NULL,
  biz_date      DATE NOT NULL,
  qty           DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight        DECIMAL(18,4) NULL,
  amount        DECIMAL(18,4) NOT NULL DEFAULT 0,
  source_report_ids JSON NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pws (worker_id, process_id, biz_date)
) ENGINE=InnoDB COMMENT='计件产量汇总';

CREATE TABLE IF NOT EXISTS pd_qc_order (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  qc_type         VARCHAR(32) NOT NULL COMMENT 'process/finished/incoming_link',
  source_doc_type VARCHAR(64) NULL,
  source_doc_id   BIGINT NULL,
  product_id      BIGINT NULL,
  result          VARCHAR(16) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_qc_no (doc_no)
) ENGINE=InnoDB COMMENT='质检单';

CREATE TABLE IF NOT EXISTS pd_rework_order (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  source_qc_id  BIGINT NULL,
  task_id       BIGINT NULL,
  process_id    BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rework_no (doc_no)
) ENGINE=InnoDB COMMENT='返修单';

CREATE TABLE IF NOT EXISTS pd_scrap_record (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  task_id       BIGINT NULL,
  process_id    BIGINT NULL,
  product_id    BIGINT NOT NULL COMMENT '废料料号',
  qty           DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  disposition   VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_scrap_no (doc_no)
) ENGINE=InnoDB COMMENT='废料登记';

CREATE TABLE IF NOT EXISTS pd_drawing_link (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  drawing_id    BIGINT NOT NULL,
  task_id       BIGINT NULL,
  work_order_id BIGINT NULL,
  process_id    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='图纸分发挂接';

CREATE TABLE IF NOT EXISTS pd_cost_hide_policy (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT NOT NULL,
  field_scope   JSON NOT NULL,
  is_enabled    TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cost_hide_role (role_id),
  CONSTRAINT fk_chp_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='成本隐藏策略（可与 iam_field_policy 并存）';

CREATE TABLE IF NOT EXISTS pd_outsource_order (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  supplier_id   BIGINT NULL,
  process_id    BIGINT NULL,
  product_id    BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_out_no (doc_no)
) ENGINE=InnoDB COMMENT='委外加工单';

CREATE TABLE IF NOT EXISTS pd_consignment_order (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  customer_id   BIGINT NULL,
  product_id    BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  progress      VARCHAR(64) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cons_no (doc_no)
) ENGINE=InnoDB COMMENT='受托加工单';

CREATE TABLE IF NOT EXISTS pd_mrp_run (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_no        VARCHAR(64) NOT NULL,
  run_at        DATETIME(3) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'done',
  params_json   JSON NULL,
  UNIQUE KEY uk_mrp_run (run_no)
) ENGINE=InnoDB COMMENT='MRP运算';

CREATE TABLE IF NOT EXISTS pd_mrp_result (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_id        BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  demand_qty    DECIMAL(18,4) NOT NULL DEFAULT 0,
  supply_qty    DECIMAL(18,4) NOT NULL DEFAULT 0,
  shortage_qty  DECIMAL(18,4) NOT NULL DEFAULT 0,
  suggest_action VARCHAR(64) NULL,
  CONSTRAINT fk_mrp_run FOREIGN KEY (run_id) REFERENCES pd_mrp_run(id)
) ENGINE=InnoDB COMMENT='MRP结果';

-- ---------------------------------------------------------------------------
-- 工资管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pay_worker_profile (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  pay_type      VARCHAR(16) NOT NULL DEFAULT 'piece' COMMENT 'piece/fixed/mixed',
  bank_account  VARCHAR(64) NULL,
  tax_no        VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pay_emp (employee_id),
  CONSTRAINT fk_pwp_emp FOREIGN KEY (employee_id) REFERENCES hr_employee(id)
) ENGINE=InnoDB COMMENT='工人工资档案';

CREATE TABLE IF NOT EXISTS pay_process_wage_rate (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  process_id      BIGINT NOT NULL,
  product_id      BIGINT NULL,
  product_spec_id BIGINT NULL,
  unit_id         BIGINT NULL,
  rate            DECIMAL(18,4) NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_pwr_process (process_id),
  CONSTRAINT fk_pwr_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工序工资（工价表）';

CREATE TABLE IF NOT EXISTS pay_payroll_sheet (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  period_year   SMALLINT NOT NULL,
  period_month  TINYINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  calc_at       DATETIME(3) NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_payroll_no (doc_no),
  UNIQUE KEY uk_payroll_period (period_year, period_month)
) ENGINE=InnoDB COMMENT='工资单头';

CREATE TABLE IF NOT EXISTS pay_payroll_sheet_line (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  sheet_id            BIGINT NOT NULL,
  employee_id         BIGINT NOT NULL,
  piece_amount        DECIMAL(18,4) NOT NULL DEFAULT 0,
  attendance_amount   DECIMAL(18,4) NOT NULL DEFAULT 0,
  commission_amount   DECIMAL(18,4) NOT NULL DEFAULT 0,
  adjust_amount       DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_amount        DECIMAL(18,4) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_psl (sheet_id, employee_id),
  CONSTRAINT fk_psl_sheet FOREIGN KEY (sheet_id) REFERENCES pay_payroll_sheet(id)
) ENGINE=InnoDB COMMENT='工资单行';

CREATE TABLE IF NOT EXISTS pay_payroll_adjust (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  sheet_id      BIGINT NOT NULL,
  employee_id   BIGINT NOT NULL,
  adjust_type   VARCHAR(32) NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  reason        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_pa_sheet FOREIGN KEY (sheet_id) REFERENCES pay_payroll_sheet(id)
) ENGINE=InnoDB COMMENT='工资调整';

CREATE TABLE IF NOT EXISTS pay_sales_commission_rule (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  name            VARCHAR(128) NOT NULL,
  rule_json       JSON NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='销售提成规则';

CREATE TABLE IF NOT EXISTS pay_commission_calc (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  rule_id           BIGINT NOT NULL,
  employee_id       BIGINT NOT NULL,
  period            VARCHAR(16) NOT NULL,
  base_amount       DECIMAL(18,4) NOT NULL DEFAULT 0,
  commission_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  source_doc_refs   JSON NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_cc_rule FOREIGN KEY (rule_id) REFERENCES pay_sales_commission_rule(id)
) ENGINE=InnoDB COMMENT='提成计算结果';
