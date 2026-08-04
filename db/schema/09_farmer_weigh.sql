-- P0 农户过磅 / 溯源 / 结算（MySQL）
CREATE TABLE IF NOT EXISTS pur_farmer (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  mobile VARCHAR(32) NULL,
  origin VARCHAR(256) NULL,
  trace_code VARCHAR(128) NULL,
  trace_code_prefix VARCHAR(32) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_pur_farmer_code (code)
) ENGINE=InnoDB COMMENT='散户农户档案';

CREATE TABLE IF NOT EXISTS pur_weigh_ticket (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no VARCHAR(64) NOT NULL,
  farmer_id BIGINT NOT NULL,
  channel VARCHAR(16) NOT NULL DEFAULT 'internal' COMMENT 'external|internal',
  ticket_template VARCHAR(32) NULL,
  product_id BIGINT NOT NULL DEFAULT 1,
  variety VARCHAR(64) NULL,
  gross_weight DECIMAL(18,4) NOT NULL DEFAULT 0,
  deduct_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
  deduct_weight DECIMAL(18,4) NOT NULL DEFAULT 0,
  net_weight DECIMAL(18,4) NOT NULL DEFAULT 0,
  qc_result VARCHAR(16) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  trace_code VARCHAR(128) NULL,
  origin VARCHAR(256) NULL,
  biz_date DATE NOT NULL,
  source_type VARCHAR(16) NOT NULL DEFAULT 'self' COMMENT 'self|outsource',
  image_url VARCHAR(512) NULL,
  box_code VARCHAR(64) NULL,
  warehouse_id BIGINT NULL,
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_weigh_doc (doc_no),
  KEY idx_weigh_farmer (farmer_id),
  KEY idx_weigh_trace (trace_code)
) ENGINE=InnoDB COMMENT='原料过磅单';

CREATE TABLE IF NOT EXISTS pur_farmer_settlement (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no VARCHAR(64) NOT NULL,
  farmer_id BIGINT NOT NULL,
  weigh_ticket_id BIGINT NULL,
  biz_date DATE NOT NULL,
  net_weight DECIMAL(18,4) NOT NULL DEFAULT 0,
  unit_price DECIMAL(18,4) NOT NULL DEFAULT 0,
  amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'open',
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_farmer_settle_doc (doc_no),
  KEY idx_farmer_settle (farmer_id, biz_date)
) ENGINE=InnoDB COMMENT='农户货款结算依据';

ALTER TABLE inv_box_code ADD COLUMN farmer_id BIGINT NULL;
ALTER TABLE inv_box_code ADD COLUMN trace_code VARCHAR(128) NULL;
ALTER TABLE inv_box_code ADD COLUMN origin VARCHAR(256) NULL;
ALTER TABLE inv_box_code ADD COLUMN receive_date DATE NULL;
ALTER TABLE inv_box_code ADD COLUMN source_type VARCHAR(16) NULL;

ALTER TABLE pd_report_work ADD COLUMN input_weight DECIMAL(18,4) NULL;
ALTER TABLE pd_report_work ADD COLUMN output_weight DECIMAL(18,4) NULL;
ALTER TABLE pd_report_work ADD COLUMN loss DECIMAL(18,4) NULL;
ALTER TABLE pd_report_work ADD COLUMN utilization DECIMAL(8,4) NULL;

ALTER TABLE pd_piecework_summary ADD COLUMN input_weight DECIMAL(18,4) NULL;
ALTER TABLE pd_piecework_summary ADD COLUMN output_weight DECIMAL(18,4) NULL;
ALTER TABLE pd_piecework_summary ADD COLUMN loss DECIMAL(18,4) NULL;
ALTER TABLE pd_piecework_summary ADD COLUMN utilization DECIMAL(8,4) NULL;
