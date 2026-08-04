USE erp_factory;

-- 客户
CREATE TABLE IF NOT EXISTS crm_customer (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(256) NOT NULL,
  level_code      VARCHAR(32) NULL,
  source          VARCHAR(64) NULL,
  owner_user_id   BIGINT NULL,
  protect_until   DATE NULL,
  is_public_sea   TINYINT(1) NOT NULL DEFAULT 0,
  is_hidden       TINYINT(1) NOT NULL DEFAULT 0,
  is_locked       TINYINT(1) NOT NULL DEFAULT 0,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  contact_json    JSON NULL,
  address         VARCHAR(512) NULL,
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_customer_code (code)
) ENGINE=InnoDB COMMENT='客户档案';

CREATE TABLE IF NOT EXISTS crm_opportunity (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  stage           VARCHAR(32) NULL,
  amount          DECIMAL(18,4) NULL,
  expected_date   DATE NULL,
  owner_user_id   BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'open',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_opp_cust FOREIGN KEY (customer_id) REFERENCES crm_customer(id)
) ENGINE=InnoDB COMMENT='商机';

CREATE TABLE IF NOT EXISTS crm_follow_up (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  opportunity_id  BIGINT NULL,
  user_id         BIGINT NOT NULL,
  follow_at       DATETIME(3) NOT NULL,
  content         VARCHAR(1024) NULL,
  next_remind_at  DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='客户跟进';

CREATE TABLE IF NOT EXISTS crm_lead_assign (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id   BIGINT NOT NULL,
  from_user_id  BIGINT NULL,
  to_user_id    BIGINT NOT NULL,
  assigned_at   DATETIME(3) NOT NULL,
  lock_flag     TINYINT(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB COMMENT='资源/线索分配';

CREATE TABLE IF NOT EXISTS crm_lead_protect_rule (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  name              VARCHAR(128) NOT NULL,
  protect_days      INT NOT NULL DEFAULT 30,
  release_rule_json JSON NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'active'
) ENGINE=InnoDB COMMENT='保护机制规则';

CREATE TABLE IF NOT EXISTS crm_lead_release_log (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  released_at     DATETIME(3) NOT NULL,
  reason          VARCHAR(512) NULL,
  to_public_sea   TINYINT(1) NOT NULL DEFAULT 1
) ENGINE=InnoDB COMMENT='释放记录';

CREATE TABLE IF NOT EXISTS crm_task_reminder (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  ref_type      VARCHAR(64) NULL,
  ref_id        BIGINT NULL,
  remind_at     DATETIME(3) NOT NULL,
  content       VARCHAR(512) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'pending'
) ENGINE=InnoDB COMMENT='任务提醒';

CREATE TABLE IF NOT EXISTS crm_customer_import_batch (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  file_name       VARCHAR(256) NULL,
  imported_at     DATETIME(3) NOT NULL,
  success_count   INT NOT NULL DEFAULT 0,
  fail_count      INT NOT NULL DEFAULT 0
) ENGINE=InnoDB COMMENT='客户导入批次';

-- 销售
CREATE TABLE IF NOT EXISTS sl_contract (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  customer_id   BIGINT NOT NULL,
  amount        DECIMAL(18,4) NULL,
  start_date    DATE NULL,
  end_date      DATE NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  file_id       BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_contract_no (doc_no)
) ENGINE=InnoDB COMMENT='合同';

CREATE TABLE IF NOT EXISTS sl_price_lock (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  lock_price      DECIMAL(18,4) NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='销售锁价';

CREATE TABLE IF NOT EXISTS sl_inquiry (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  customer_id     BIGINT NOT NULL,
  owner_user_id   BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  source          VARCHAR(16) NOT NULL DEFAULT 'sales' COMMENT 'self/sales',
  expire_at       DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_inquiry_no (doc_no)
) ENGINE=InnoDB COMMENT='询价单头';

CREATE TABLE IF NOT EXISTS sl_inquiry_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  inquiry_id    BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  quote_price   DECIMAL(18,4) NULL,
  cost_ref      DECIMAL(18,4) NULL,
  remark        VARCHAR(256) NULL,
  CONSTRAINT fk_il_inq FOREIGN KEY (inquiry_id) REFERENCES sl_inquiry(id)
) ENGINE=InnoDB COMMENT='询价行';

CREATE TABLE IF NOT EXISTS sl_quote_history (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  price         DECIMAL(18,4) NOT NULL,
  quoted_at     DATETIME(3) NOT NULL,
  inquiry_id    BIGINT NULL,
  order_id      BIGINT NULL
) ENGINE=InnoDB COMMENT='历史报价';

CREATE TABLE IF NOT EXISTS sl_sales_order (
  id                      BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no                  VARCHAR(64) NOT NULL,
  customer_id             BIGINT NOT NULL,
  owner_user_id           BIGINT NULL,
  status                  VARCHAR(16) NOT NULL DEFAULT 'draft',
  source                  VARCHAR(16) NOT NULL DEFAULT 'manual' COMMENT 'manual/self/rebuy',
  contract_id             BIGINT NULL,
  price_lock_id           BIGINT NULL,
  reorder_from_id         BIGINT NULL,
  need_delivery_approval  TINYINT(1) NOT NULL DEFAULT 1,
  org_id                  BIGINT NULL,
  remark                  VARCHAR(512) NULL,
  created_by              BIGINT NULL,
  created_at              DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at              DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted              TINYINT(1) NOT NULL DEFAULT 0,
  version                 INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_so_no (doc_no)
) ENGINE=InnoDB COMMENT='销售订单头';

CREATE TABLE IF NOT EXISTS sl_sales_order_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id        BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  qty             DECIMAL(18,4) NOT NULL,
  weight          DECIMAL(18,4) NULL,
  price           DECIMAL(18,4) NOT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  delivered_qty   DECIMAL(18,4) NOT NULL DEFAULT 0,
  CONSTRAINT fk_sol_so FOREIGN KEY (order_id) REFERENCES sl_sales_order(id)
) ENGINE=InnoDB COMMENT='销售订单行';

CREATE TABLE IF NOT EXISTS sl_pre_shipment (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  order_id      BIGINT NOT NULL,
  plan_ship_date DATE NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  reserved      TINYINT(1) NOT NULL DEFAULT 0,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pre_ship_no (doc_no)
) ENGINE=InnoDB COMMENT='预发货';

CREATE TABLE IF NOT EXISTS sl_pre_shipment_line (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  pre_shipment_id   BIGINT NOT NULL,
  order_line_id     BIGINT NULL,
  product_id        BIGINT NOT NULL,
  qty               DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_psl_hdr FOREIGN KEY (pre_shipment_id) REFERENCES sl_pre_shipment(id)
) ENGINE=InnoDB COMMENT='预发货行';

CREATE TABLE IF NOT EXISTS sl_delivery_approval (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  order_id          BIGINT NOT NULL,
  pre_shipment_id   BIGINT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  warehouse_id      BIGINT NULL,
  shipped_at        DATETIME(3) NULL,
  logistics_no      VARCHAR(64) NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_da_no (doc_no)
) ENGINE=InnoDB COMMENT='发货审批/发货单';

CREATE TABLE IF NOT EXISTS sl_delivery_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  delivery_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  batch_no      VARCHAR(64) NULL,
  box_code_id   BIGINT NULL,
  CONSTRAINT fk_dl_hdr FOREIGN KEY (delivery_id) REFERENCES sl_delivery_approval(id)
) ENGINE=InnoDB COMMENT='发货行';

CREATE TABLE IF NOT EXISTS sl_sales_bom (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id      BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  version_no    VARCHAR(32) NOT NULL DEFAULT 'V1',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='订单级BOM';

CREATE TABLE IF NOT EXISTS sl_sales_bom_line (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  sales_bom_id          BIGINT NOT NULL,
  component_product_id  BIGINT NOT NULL,
  qty                   DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_sbl_hdr FOREIGN KEY (sales_bom_id) REFERENCES sl_sales_bom(id)
) ENGINE=InnoDB COMMENT='销售BOM行';

CREATE TABLE IF NOT EXISTS sl_cost_budget (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id        BIGINT NOT NULL,
  material_cost   DECIMAL(18,4) NOT NULL DEFAULT 0,
  labor_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  other_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  margin          DECIMAL(18,4) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cb_order (order_id)
) ENGINE=InnoDB COMMENT='订单成本预算';

CREATE TABLE IF NOT EXISTS sl_quote_calculator_result (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  inquiry_id    BIGINT NULL,
  order_id      BIGINT NULL,
  input_json    JSON NULL,
  result_json   JSON NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='报价试算结果';

CREATE TABLE IF NOT EXISTS sl_sales_rank_config (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(128) NOT NULL,
  metric        VARCHAR(64) NOT NULL,
  period_type   VARCHAR(16) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
) ENGINE=InnoDB COMMENT='排行榜配置';

CREATE TABLE IF NOT EXISTS sl_order_change_log (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id      BIGINT NOT NULL,
  change_json   JSON NOT NULL,
  changed_by    BIGINT NULL,
  changed_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='修改订单记录';

-- 采购
CREATE TABLE IF NOT EXISTS pur_supplier (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(256) NOT NULL,
  short_name    VARCHAR(128) NULL,
  mnemonic      VARCHAR(64) NULL,
  supplier_type VARCHAR(32) NOT NULL DEFAULT 'raw' COMMENT 'raw/aux/pack/logistics/outsource/service',
  status        VARCHAR(16) NOT NULL DEFAULT 'potential' COMMENT 'potential/qualified/frozen/blacklist/eliminated',
  rating        VARCHAR(16) NULL,
  is_preferred  TINYINT(1) NOT NULL DEFAULT 0,
  uscc          VARCHAR(32) NULL,
  legal_person  VARCHAR(64) NULL,
  register_address VARCHAR(512) NULL,
  invoice_title VARCHAR(256) NULL,
  tax_no        VARCHAR(64) NULL,
  bank_name     VARCHAR(128) NULL,
  bank_account  VARCHAR(64) NULL,
  settle_method VARCHAR(32) NULL COMMENT 'cash/monthly/cod',
  payment_days  INT NULL,
  credit_limit  DECIMAL(18,2) NULL,
  currency      VARCHAR(8) NOT NULL DEFAULT 'CNY',
  tax_rate      DECIMAL(8,4) NULL,
  lead_time_days INT NULL,
  moq           DECIMAL(18,4) NULL,
  default_warehouse_id BIGINT NULL,
  contact_json  JSON NULL,
  remark        VARCHAR(1024) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_supplier_code (code)
) ENGINE=InnoDB COMMENT='供应商';

CREATE TABLE IF NOT EXISTS pur_supplier_license (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  supplier_id   BIGINT NOT NULL,
  license_type  VARCHAR(64) NOT NULL,
  license_no    VARCHAR(128) NULL,
  expire_date   DATE NULL,
  attachment_url VARCHAR(512) NULL,
  remark        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_psl_supplier (supplier_id),
  KEY idx_psl_expire (expire_date),
  CONSTRAINT fk_psl_sup FOREIGN KEY (supplier_id) REFERENCES pur_supplier(id)
) ENGINE=InnoDB COMMENT='供应商证照';

CREATE TABLE IF NOT EXISTS pur_supplier_supply_item (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  supplier_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  is_preferred  TINYINT(1) NOT NULL DEFAULT 0,
  moq           DECIMAL(18,4) NULL,
  lead_time_days INT NULL,
  last_price    DECIMAL(18,4) NULL,
  remark        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_psi (supplier_id, product_id),
  CONSTRAINT fk_psi_sup FOREIGN KEY (supplier_id) REFERENCES pur_supplier(id)
) ENGINE=InnoDB COMMENT='供应商可供物料';

CREATE TABLE IF NOT EXISTS pur_purchase_request (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  applicant_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  need_date     DATE NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pr_no (doc_no)
) ENGINE=InnoDB COMMENT='采购申请头';

CREATE TABLE IF NOT EXISTS pur_purchase_request_line (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  request_id          BIGINT NOT NULL,
  product_id          BIGINT NOT NULL,
  qty                 DECIMAL(18,4) NOT NULL,
  unit_id             BIGINT NULL,
  suggest_supplier_id BIGINT NULL,
  CONSTRAINT fk_prl_hdr FOREIGN KEY (request_id) REFERENCES pur_purchase_request(id)
) ENGINE=InnoDB COMMENT='采购申请行';

CREATE TABLE IF NOT EXISTS pur_purchase_plan (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  plan_date     DATE NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pp_no (doc_no)
) ENGINE=InnoDB COMMENT='采购计划单头';

CREATE TABLE IF NOT EXISTS pur_purchase_plan_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  plan_id         BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  qty             DECIMAL(18,4) NOT NULL,
  supplier_id     BIGINT NULL,
  request_line_id BIGINT NULL,
  CONSTRAINT fk_ppl_hdr FOREIGN KEY (plan_id) REFERENCES pur_purchase_plan(id)
) ENGINE=InnoDB COMMENT='采购计划行';

CREATE TABLE IF NOT EXISTS pur_purchase_inbound (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  supplier_id   BIGINT NOT NULL,
  warehouse_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  biz_date      DATE NOT NULL,
  plan_id       BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pi_no (doc_no)
) ENGINE=InnoDB COMMENT='采购入库头';

CREATE TABLE IF NOT EXISTS pur_purchase_inbound_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  inbound_id    BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  price         DECIMAL(18,4) NULL,
  amount        DECIMAL(18,4) NULL,
  batch_no      VARCHAR(64) NULL,
  CONSTRAINT fk_pil_hdr FOREIGN KEY (inbound_id) REFERENCES pur_purchase_inbound(id)
) ENGINE=InnoDB COMMENT='采购入库行';

CREATE TABLE IF NOT EXISTS pur_incoming_qc (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  inbound_id    BIGINT NULL,
  product_id    BIGINT NOT NULL,
  qty_check     DECIMAL(18,4) NOT NULL,
  qty_pass      DECIMAL(18,4) NOT NULL DEFAULT 0,
  qty_fail      DECIMAL(18,4) NOT NULL DEFAULT 0,
  result        VARCHAR(16) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_iqc_no (doc_no)
) ENGINE=InnoDB COMMENT='来料质检';

CREATE TABLE IF NOT EXISTS pur_purchase_return (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  supplier_id   BIGINT NOT NULL,
  inbound_id    BIGINT NULL,
  warehouse_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  reason        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_prt_no (doc_no)
) ENGINE=InnoDB COMMENT='采购退货';

CREATE TABLE IF NOT EXISTS pur_purchase_return_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  return_id     BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  amount        DECIMAL(18,4) NULL,
  CONSTRAINT fk_prtl_hdr FOREIGN KEY (return_id) REFERENCES pur_purchase_return(id)
) ENGINE=InnoDB COMMENT='采购退货行';

CREATE TABLE IF NOT EXISTS pur_supplier_price_history (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  supplier_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  price         DECIMAL(18,4) NOT NULL,
  biz_date      DATE NOT NULL,
  source_doc_id BIGINT NULL,
  KEY idx_sph (supplier_id, product_id, biz_date)
) ENGINE=InnoDB COMMENT='历史采购价';

CREATE TABLE IF NOT EXISTS pur_purchase_task (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  assignee_id   BIGINT NULL,
  product_id    BIGINT NULL,
  qty           DECIMAL(18,4) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  due_date      DATE NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ptask_no (doc_no)
) ENGINE=InnoDB COMMENT='采购任务';
