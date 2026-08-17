USE erp_factory;

-- ---------------------------------------------------------------------------
-- 产品管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS prd_product (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  code              VARCHAR(64) NOT NULL,
  name              VARCHAR(256) NOT NULL,
  category          VARCHAR(64) NULL,
  product_type      VARCHAR(32) NOT NULL COMMENT 'raw/semi/finished/aux/scrap',
  base_unit_id      BIGINT NULL,
  spec_text         VARCHAR(256) NULL,
  barcode           VARCHAR(64) NULL,
  cost_price        DECIMAL(18,4) NULL,
  sale_price        DECIMAL(18,4) NULL,
  is_batch_managed  TINYINT(1) NOT NULL DEFAULT 1,
  is_box_managed    TINYINT(1) NOT NULL DEFAULT 0,
  status            VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by        BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by        BIGINT NULL,
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted        TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at        DATETIME(3) NULL,
  deleted_by        BIGINT NULL,
  version           INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_product_code (code)
) ENGINE=InnoDB COMMENT='产品/物料档案';

CREATE TABLE IF NOT EXISTS prd_product_unit (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id      BIGINT NOT NULL,
  unit_name       VARCHAR(32) NOT NULL,
  is_base         TINYINT(1) NOT NULL DEFAULT 0,
  factor_to_base  DECIMAL(18,8) NOT NULL DEFAULT 1,
  is_purchase     TINYINT(1) NOT NULL DEFAULT 1,
  is_sale         TINYINT(1) NOT NULL DEFAULT 1,
  is_stock        TINYINT(1) NOT NULL DEFAULT 1,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_prod_unit (product_id, unit_name),
  CONSTRAINT fk_pu_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='产品单位';

CREATE TABLE IF NOT EXISTS prd_product_spec (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id      BIGINT NOT NULL,
  spec_code       VARCHAR(64) NOT NULL,
  routing_id      BIGINT NULL,
  process_wage_bind_json JSON NULL,
  remark          VARCHAR(512) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_prod_spec (product_id, spec_code),
  CONSTRAINT fk_ps_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='生产规格绑定';

CREATE TABLE IF NOT EXISTS prd_product_app_sort (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id    BIGINT NOT NULL,
  channel       VARCHAR(32) NOT NULL DEFAULT 'app',
  sort_no       INT NOT NULL DEFAULT 0,
  is_visible    TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY uk_app_sort (product_id, channel),
  CONSTRAINT fk_pas_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='APP产品排序';

-- ---------------------------------------------------------------------------
-- 库存管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS inv_box_code (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  product_id    BIGINT NOT NULL,
  warehouse_id  BIGINT NULL,
  batch_no      VARCHAR(64) NULL,
  qty           DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight        DECIMAL(18,4) NULL,
  parent_box_id BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_box_code (code),
  CONSTRAINT fk_box_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='箱码';

CREATE TABLE IF NOT EXISTS inv_balance (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  warehouse_id  BIGINT NOT NULL,
  location_id   BIGINT NOT NULL DEFAULT 0,
  product_id    BIGINT NOT NULL,
  batch_no      VARCHAR(64) NOT NULL DEFAULT '',
  box_code_id   BIGINT NOT NULL DEFAULT 0,
  qty           DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight        DECIMAL(18,4) NULL,
  avg_cost      DECIMAL(18,4) NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_balance (warehouse_id, product_id, batch_no, location_id, box_code_id),
  KEY idx_bal_product (product_id),
  CONSTRAINT fk_bal_wh FOREIGN KEY (warehouse_id) REFERENCES inv_warehouse(id),
  CONSTRAINT fk_bal_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='库存结存';

CREATE TABLE IF NOT EXISTS inv_stock_txn (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  doc_type          VARCHAR(32) NOT NULL COMMENT 'opening/purchase_in/purchase_return/produce_in/requisition_out/transfer/stocktake_gain/stocktake_loss/sales_out/sales_return/consume',
  biz_date          DATE NOT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  warehouse_id      BIGINT NULL,
  counterparty_type VARCHAR(32) NULL,
  counterparty_id   BIGINT NULL,
  source_doc_type   VARCHAR(64) NULL,
  source_doc_id     BIGINT NULL,
  posted_at         DATETIME(3) NULL,
  org_id            BIGINT NULL,
  owner_user_id     BIGINT NULL,
  remark            VARCHAR(512) NULL,
  created_by        BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by        BIGINT NULL,
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted        TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at        DATETIME(3) NULL,
  deleted_by        BIGINT NULL,
  version           INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_stock_txn_no (doc_no),
  KEY idx_txn_type_date (doc_type, biz_date),
  KEY idx_txn_source (source_doc_type, source_doc_id)
) ENGINE=InnoDB COMMENT='出入库流水头（唯一过账事实源）';

CREATE TABLE IF NOT EXISTS inv_stock_txn_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  txn_id        BIGINT NOT NULL,
  line_no       INT NOT NULL,
  product_id    BIGINT NOT NULL,
  unit_id       BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  base_qty      DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  batch_no      VARCHAR(64) NULL,
  box_code_id   BIGINT NULL,
  location_id   BIGINT NULL,
  direction     VARCHAR(8) NOT NULL COMMENT 'in/out',
  amount        DECIMAL(18,4) NULL,
  remark        VARCHAR(256) NULL,
  UNIQUE KEY uk_txn_line (txn_id, line_no),
  CONSTRAINT fk_txn_line_hdr FOREIGN KEY (txn_id) REFERENCES inv_stock_txn(id),
  CONSTRAINT fk_txn_line_prd FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='出入库流水行';

CREATE TABLE IF NOT EXISTS inv_reservation (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  warehouse_id    BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  batch_no        VARCHAR(64) NULL,
  qty             DECIMAL(18,4) NOT NULL,
  source_doc_type VARCHAR(64) NOT NULL,
  source_doc_id   BIGINT NOT NULL,
  source_line_id  BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/released',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_rsv_product (warehouse_id, product_id, status),
  KEY idx_rsv_source (source_doc_type, source_doc_id)
) ENGINE=InnoDB COMMENT='待用/占用';

CREATE TABLE IF NOT EXISTS inv_in_transit (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id      BIGINT NOT NULL,
  warehouse_id    BIGINT NOT NULL COMMENT '目标仓',
  qty             DECIMAL(18,4) NOT NULL,
  transit_type    VARCHAR(32) NOT NULL COMMENT 'purchase/transfer',
  source_doc_type VARCHAR(64) NOT NULL,
  source_doc_id   BIGINT NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'open' COMMENT 'open/closed',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_transit (warehouse_id, product_id, status)
) ENGINE=InnoDB COMMENT='在途量';

CREATE TABLE IF NOT EXISTS inv_inbound_qc (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  stock_txn_id  BIGINT NULL,
  product_id    BIGINT NOT NULL,
  qty_check     DECIMAL(18,4) NOT NULL,
  qty_pass      DECIMAL(18,4) NOT NULL DEFAULT 0,
  qty_fail      DECIMAL(18,4) NOT NULL DEFAULT 0,
  result        VARCHAR(16) NULL,
  inspector_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_inbound_qc_no (doc_no)
) ENGINE=InnoDB COMMENT='入库质检';

CREATE TABLE IF NOT EXISTS inv_stocktake (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  stocktake_type  VARCHAR(16) NOT NULL COMMENT 'warehouse/workshop',
  warehouse_id    BIGINT NULL,
  workshop_id     BIGINT NULL,
  biz_date        DATE NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  version         INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_stocktake_no (doc_no)
) ENGINE=InnoDB COMMENT='盘点单头';

CREATE TABLE IF NOT EXISTS inv_stocktake_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  stocktake_id  BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  book_qty      DECIMAL(18,4) NOT NULL DEFAULT 0,
  count_qty     DECIMAL(18,4) NOT NULL DEFAULT 0,
  diff_qty      DECIMAL(18,4) NOT NULL DEFAULT 0,
  batch_no      VARCHAR(64) NULL,
  location_id   BIGINT NULL,
  CONSTRAINT fk_stl_hdr FOREIGN KEY (stocktake_id) REFERENCES inv_stocktake(id)
) ENGINE=InnoDB COMMENT='盘点行';

CREATE TABLE IF NOT EXISTS inv_transfer (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no              VARCHAR(64) NOT NULL,
  from_warehouse_id   BIGINT NOT NULL,
  to_warehouse_id     BIGINT NOT NULL,
  biz_date            DATE NOT NULL,
  status              VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by          BIGINT NULL,
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted          TINYINT(1) NOT NULL DEFAULT 0,
  version             INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_transfer_no (doc_no)
) ENGINE=InnoDB COMMENT='调拨单头';

CREATE TABLE IF NOT EXISTS inv_transfer_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  transfer_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  base_qty      DECIMAL(18,4) NOT NULL,
  batch_no      VARCHAR(64) NULL,
  CONSTRAINT fk_trl_hdr FOREIGN KEY (transfer_id) REFERENCES inv_transfer(id)
) ENGINE=InnoDB COMMENT='调拨行';

CREATE TABLE IF NOT EXISTS inv_assemble_split (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  biz_type      VARCHAR(16) NOT NULL COMMENT 'assemble/split',
  warehouse_id  BIGINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_asm_no (doc_no)
) ENGINE=InnoDB COMMENT='组装拆分单';

CREATE TABLE IF NOT EXISTS inv_assemble_split_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  header_id     BIGINT NOT NULL,
  role_type     VARCHAR(16) NOT NULL COMMENT 'parent/child',
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_asl_hdr FOREIGN KEY (header_id) REFERENCES inv_assemble_split(id)
) ENGINE=InnoDB COMMENT='组装拆分行';

CREATE TABLE IF NOT EXISTS inv_price_adjust (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  product_id    BIGINT NOT NULL,
  old_price     DECIMAL(18,4) NOT NULL,
  new_price     DECIMAL(18,4) NOT NULL,
  effective_at  DATETIME(3) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_price_adj_no (doc_no)
) ENGINE=InnoDB COMMENT='商品调价单';

CREATE TABLE IF NOT EXISTS inv_stock_alert_rule (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id    BIGINT NULL,
  warehouse_id  BIGINT NULL,
  alert_type    VARCHAR(16) NOT NULL COMMENT 'shortage/excess',
  min_qty       DECIMAL(18,4) NULL,
  max_qty       DECIMAL(18,4) NULL,
  is_enabled    TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='亏料/过量预警规则';

CREATE TABLE IF NOT EXISTS inv_sales_peel_return (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  sales_order_id  BIGINT NULL,
  product_id      BIGINT NOT NULL,
  peel_qty        DECIMAL(18,4) NOT NULL,
  weight          DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_peel_no (doc_no)
) ENGINE=InnoDB COMMENT='销售退皮';

CREATE TABLE IF NOT EXISTS inv_material_to_payable (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  consume_txn_id  BIGINT NULL,
  supplier_id     BIGINT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_mtp_no (doc_no)
) ENGINE=InnoDB COMMENT='物料转应付';
