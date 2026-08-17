USE erp_factory;

-- 固定资产
CREATE TABLE IF NOT EXISTS ast_fixed_asset_category (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  parent_id     BIGINT NULL,
  UNIQUE KEY uk_fac_code (code)
) ENGINE=InnoDB COMMENT='固定资产类别';

CREATE TABLE IF NOT EXISTS ast_fixed_asset (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(256) NOT NULL,
  category_id     BIGINT NULL,
  dept_id         BIGINT NULL,
  location_text   VARCHAR(256) NULL,
  original_value  DECIMAL(18,4) NULL,
  net_value       DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  purchase_date   DATE NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_fa_code (code)
) ENGINE=InnoDB COMMENT='固定资产卡片';

CREATE TABLE IF NOT EXISTS ast_asset_transfer (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  asset_id        BIGINT NOT NULL,
  from_dept_id    BIGINT NULL,
  to_dept_id      BIGINT NULL,
  from_location   VARCHAR(256) NULL,
  to_location     VARCHAR(256) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  transferred_at  DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_at_no (doc_no)
) ENGINE=InnoDB COMMENT='固定资产内部转移';

-- 财务
CREATE TABLE IF NOT EXISTS fin_account_subject (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  parent_id     BIGINT NULL,
  subject_type  VARCHAR(32) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_subject_code (code)
) ENGINE=InnoDB COMMENT='会计科目/账目';

CREATE TABLE IF NOT EXISTS fin_fund_account (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  currency      VARCHAR(8) NOT NULL DEFAULT 'CNY',
  balance       DECIMAL(18,4) NOT NULL DEFAULT 0,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_fund_code (code)
) ENGINE=InnoDB COMMENT='资金账户';

CREATE TABLE IF NOT EXISTS fin_fund_transfer (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  from_account_id   BIGINT NOT NULL,
  to_account_id     BIGINT NOT NULL,
  amount            DECIMAL(18,4) NOT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ft_no (doc_no)
) ENGINE=InnoDB COMMENT='资金调拨';

CREATE TABLE IF NOT EXISTS fin_ledger_entry (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  account_id        BIGINT NULL,
  subject_id        BIGINT NULL,
  direction         VARCHAR(8) NOT NULL COMMENT 'in/out',
  amount            DECIMAL(18,4) NOT NULL,
  biz_date          DATE NOT NULL,
  counterparty      VARCHAR(256) NULL,
  source_doc_type   VARCHAR(64) NULL,
  source_doc_id     BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_le_no (doc_no)
) ENGINE=InnoDB COMMENT='交易流水账';

CREATE TABLE IF NOT EXISTS fin_income_expense_detail (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  entry_id      BIGINT NOT NULL,
  category      VARCHAR(64) NULL,
  amount        DECIMAL(18,4) NOT NULL,
  remark        VARCHAR(512) NULL,
  CONSTRAINT fk_ied_entry FOREIGN KEY (entry_id) REFERENCES fin_ledger_entry(id)
) ENGINE=InnoDB COMMENT='收入支出明细';

CREATE TABLE IF NOT EXISTS fin_voucher (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  period        VARCHAR(16) NULL,
  biz_date      DATE NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  summary       VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_voucher_no (doc_no)
) ENGINE=InnoDB COMMENT='凭证头';

CREATE TABLE IF NOT EXISTS fin_voucher_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  voucher_id    BIGINT NOT NULL,
  subject_id    BIGINT NOT NULL,
  debit         DECIMAL(18,4) NOT NULL DEFAULT 0,
  credit        DECIMAL(18,4) NOT NULL DEFAULT 0,
  remark        VARCHAR(256) NULL,
  CONSTRAINT fk_vl_hdr FOREIGN KEY (voucher_id) REFERENCES fin_voucher(id)
) ENGINE=InnoDB COMMENT='凭证行';

CREATE TABLE IF NOT EXISTS fin_invoice (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  invoice_no      VARCHAR(64) NOT NULL,
  direction       VARCHAR(8) NOT NULL COMMENT 'in/out',
  counterparty_id BIGINT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  tax             DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  biz_date        DATE NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_invoice_no (invoice_no)
) ENGINE=InnoDB COMMENT='发票';

CREATE TABLE IF NOT EXISTS fin_receipt_writeoff (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  customer_id   BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  received_at   DATETIME(3) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rw_no (doc_no)
) ENGINE=InnoDB COMMENT='收款核单';

CREATE TABLE IF NOT EXISTS fin_receipt_writeoff_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  writeoff_id     BIGINT NOT NULL,
  sales_order_id  BIGINT NOT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_rwl_hdr FOREIGN KEY (writeoff_id) REFERENCES fin_receipt_writeoff(id)
) ENGINE=InnoDB COMMENT='核销行';

CREATE TABLE IF NOT EXISTS fin_payment_recognition (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  customer_id       BIGINT NOT NULL,
  amount            DECIMAL(18,4) NOT NULL,
  fund_account_id   BIGINT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pr_no (doc_no)
) ENGINE=InnoDB COMMENT='销售认款';

CREATE TABLE IF NOT EXISTS fin_prepay_prepaid (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  party_type    VARCHAR(16) NOT NULL COMMENT 'customer/supplier',
  party_id      BIGINT NOT NULL,
  direction     VARCHAR(8) NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  balance       DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pp_no (doc_no)
) ENGINE=InnoDB COMMENT='预收预付';

CREATE TABLE IF NOT EXISTS fin_fx_settlement (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  currency      VARCHAR(8) NOT NULL,
  amount_fx     DECIMAL(18,4) NOT NULL,
  rate          DECIMAL(18,8) NOT NULL,
  amount_local  DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_fx_no (doc_no)
) ENGINE=InnoDB COMMENT='外币结汇';

CREATE TABLE IF NOT EXISTS fin_cost_allocation (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  source_amount   DECIMAL(18,4) NOT NULL,
  alloc_json      JSON NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  revoked_from_id BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ca_no (doc_no)
) ENGINE=InnoDB COMMENT='费用分摊';

CREATE TABLE IF NOT EXISTS fin_receipt_alert (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id   BIGINT NOT NULL,
  order_id      BIGINT NULL,
  due_date      DATE NULL,
  overdue_days  INT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='收款预警';

CREATE TABLE IF NOT EXISTS fin_cashier_reconcile (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  fund_account_id BIGINT NOT NULL,
  biz_date        DATE NOT NULL,
  book_balance    DECIMAL(18,4) NOT NULL,
  actual_balance  DECIMAL(18,4) NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cr_no (doc_no)
) ENGINE=InnoDB COMMENT='出纳对账';

CREATE TABLE IF NOT EXISTS fin_cost_accounting (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  period          VARCHAR(16) NOT NULL,
  task_id         BIGINT NULL,
  product_id      BIGINT NULL,
  material_cost   DECIMAL(18,4) NOT NULL DEFAULT 0,
  labor_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  overhead        DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cost_acc_no (doc_no)
) ENGINE=InnoDB COMMENT='成本核算单';

CREATE TABLE IF NOT EXISTS fin_cost_trace_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  cost_id       BIGINT NOT NULL,
  source_type   VARCHAR(32) NOT NULL COMMENT 'report/requisition/stock',
  source_id     BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_ctl_cost FOREIGN KEY (cost_id) REFERENCES fin_cost_accounting(id)
) ENGINE=InnoDB COMMENT='成本明细溯源';

CREATE TABLE IF NOT EXISTS fin_contract_profit (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  contract_id   BIGINT NOT NULL,
  revenue       DECIMAL(18,4) NOT NULL DEFAULT 0,
  cost          DECIMAL(18,4) NOT NULL DEFAULT 0,
  profit        DECIMAL(18,4) NOT NULL DEFAULT 0,
  period        VARCHAR(16) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='合同利润';

CREATE TABLE IF NOT EXISTS fin_sales_return_finance (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  order_id      BIGINT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_srf_no (doc_no)
) ENGINE=InnoDB COMMENT='销售退货退单财务';

CREATE TABLE IF NOT EXISTS fin_arap_adjust (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  party_type    VARCHAR(16) NOT NULL,
  party_id      BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  direction     VARCHAR(8) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_arap_no (doc_no)
) ENGINE=InnoDB COMMENT='往来调整单';

CREATE TABLE IF NOT EXISTS fin_month_close (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  year          SMALLINT NOT NULL,
  month         TINYINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  closed_at     DATETIME(3) NULL,
  closed_by     BIGINT NULL,
  UNIQUE KEY uk_month_close (year, month)
) ENGINE=InnoDB COMMENT='月度结转';

CREATE TABLE IF NOT EXISTS fin_miniprogram_bill (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  bill_no       VARCHAR(64) NOT NULL,
  channel       VARCHAR(32) NULL,
  amount        DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'unpaid',
  order_id      BIGINT NULL,
  paid_at       DATETIME(3) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_mp_bill (bill_no)
) ENGINE=InnoDB COMMENT='小程序账单';
