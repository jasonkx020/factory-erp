-- 现场台账：出厂结算、计件领料表、工具领还、地磅、入厂扩展列（MySQL）

ALTER TABLE pur_inbound_arrival ADD COLUMN plate_no VARCHAR(32) NULL;
ALTER TABLE pur_inbound_arrival ADD COLUMN receive_address VARCHAR(256) NULL;
ALTER TABLE pur_inbound_arrival ADD COLUMN pass_rate DECIMAL(8,4) NOT NULL DEFAULT 0;
ALTER TABLE pur_inbound_arrival ADD COLUMN reject_weight DECIMAL(18,4) NOT NULL DEFAULT 0;
ALTER TABLE pur_inbound_arrival ADD COLUMN freight_fee DECIMAL(18,4) NOT NULL DEFAULT 0;
ALTER TABLE pur_inbound_arrival ADD COLUMN loading_fee DECIMAL(18,4) NOT NULL DEFAULT 0;
ALTER TABLE pur_inbound_arrival ADD COLUMN weigh_fee DECIMAL(18,4) NOT NULL DEFAULT 0;

ALTER TABLE pd_report_work ADD COLUMN bag_qty DECIMAL(18,4) NOT NULL DEFAULT 0;
ALTER TABLE pd_scrap_record ADD COLUMN scrap_type VARCHAR(32) NULL COMMENT 'cut_defect|core_defect|dice_defect|sieve_bag_defect';
ALTER TABLE pay_process_wage_rate ADD COLUMN rate_unit VARCHAR(16) NOT NULL DEFAULT 'kg' COMMENT 'kg|hour';

CREATE TABLE IF NOT EXISTS sl_outbound_settle (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no VARCHAR(64) NOT NULL,
  biz_date DATE NOT NULL,
  product_id BIGINT NULL,
  product_name VARCHAR(128) NULL,
  plate_no VARCHAR(32) NULL,
  driver_name VARCHAR(64) NULL,
  trace_code VARCHAR(128) NULL,
  produce_date DATE NULL,
  qty DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight DECIMAL(18,4) NOT NULL DEFAULT 0,
  unit VARCHAR(16) NOT NULL DEFAULT 'kg',
  freight_fee DECIMAL(18,4) NOT NULL DEFAULT 0,
  loading_fee DECIMAL(18,4) NOT NULL DEFAULT 0,
  weigh_fee DECIMAL(18,4) NOT NULL DEFAULT 0,
  unit_price DECIMAL(18,4) NOT NULL DEFAULT 0,
  goods_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_sl_outbound_doc (doc_no)
) ENGINE=InnoDB COMMENT='销售出厂结算';

CREATE TABLE IF NOT EXISTS pd_piece_issue_sheet (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no VARCHAR(64) NOT NULL,
  biz_date DATE NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_piece_issue_doc (doc_no)
) ENGINE=InnoDB COMMENT='计件领料表头';

CREATE TABLE IF NOT EXISTS pd_piece_issue_line (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  sheet_id BIGINT NOT NULL,
  seq_no INT NOT NULL DEFAULT 1,
  employee_id BIGINT NULL,
  employee_name VARCHAR(64) NULL,
  process_id BIGINT NULL,
  process_name VARCHAR(64) NULL,
  process_kind VARCHAR(16) NOT NULL DEFAULT 'piece' COMMENT 'piece|time',
  unit_price DECIMAL(18,4) NOT NULL DEFAULT 0,
  qty DECIMAL(18,4) NOT NULL DEFAULT 0,
  reject_qty DECIMAL(18,4) NOT NULL DEFAULT 0,
  qty_total DECIMAL(18,4) NOT NULL DEFAULT 0,
  amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  remark VARCHAR(256) NULL,
  KEY idx_piece_issue_sheet (sheet_id)
) ENGINE=InnoDB COMMENT='计件领料表明细';

CREATE TABLE IF NOT EXISTS hr_tool_item (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_hr_tool_code (code)
) ENGINE=InnoDB COMMENT='工具品类';

CREATE TABLE IF NOT EXISTS hr_tool_issue (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no VARCHAR(64) NOT NULL,
  biz_date DATE NOT NULL,
  seq_no INT NOT NULL DEFAULT 1,
  employee_id BIGINT NULL,
  employee_name VARCHAR(64) NULL,
  tool_item_id BIGINT NOT NULL,
  tool_name VARCHAR(64) NULL,
  issue_qty DECIMAL(18,4) NOT NULL DEFAULT 0,
  return_qty DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_qty DECIMAL(18,4) NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'open',
  remark VARCHAR(256) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_hr_tool_issue_doc (doc_no)
) ENGINE=InnoDB COMMENT='工具领还';

CREATE TABLE IF NOT EXISTS inv_weighbridge (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  location VARCHAR(256) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_weighbridge_code (code)
) ENGINE=InnoDB COMMENT='地磅主数据';

INSERT IGNORE INTO hr_tool_item(id, code, name, status) VALUES
 (1,'TOOL-SCRAPER','刮刀','active'),
 (2,'TOOL-KNIFE','小刀','active'),
 (3,'TOOL-GLOVE-T','厚手套','active'),
 (4,'TOOL-GLOVE-S','薄手套','active'),
 (5,'TOOL-HAT','帽子','active'),
 (6,'TOOL-UNIFORM','工服','active'),
 (7,'TOOL-SHOES','鞋子','active');

INSERT IGNORE INTO inv_weighbridge(id, code, name, location, status) VALUES
 (1,'WB-GATE','大门地磅','厂区大门','active'),
 (2,'WB-FORK','叉车秤','原料区','active');
