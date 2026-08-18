-- v1.0.14: 销售主链闭环字段（询价审批时间线、发货签收、出厂结算关联、合同/锁价/BOM 版本）

ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS reject_reason TEXT;
ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS submitted_at TEXT;
ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS approved_at TEXT;
ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS rejected_at TEXT;

ALTER TABLE sl_delivery_approval ADD COLUMN IF NOT EXISTS reject_reason TEXT;
ALTER TABLE sl_delivery_approval ADD COLUMN IF NOT EXISTS received_at TEXT;
ALTER TABLE sl_delivery_approval ADD COLUMN IF NOT EXISTS receive_remark TEXT;

ALTER TABLE sl_outbound_settle ADD COLUMN IF NOT EXISTS order_id INTEGER;
ALTER TABLE sl_outbound_settle ADD COLUMN IF NOT EXISTS delivery_id INTEGER;
ALTER TABLE sl_outbound_settle ADD COLUMN IF NOT EXISTS closed_at TEXT;

ALTER TABLE sl_contract ADD COLUMN IF NOT EXISTS attachment_url TEXT;
ALTER TABLE sl_contract ADD COLUMN IF NOT EXISTS order_id INTEGER;

ALTER TABLE sl_price_lock ADD COLUMN IF NOT EXISTS version_no INTEGER NOT NULL DEFAULT 1;

ALTER TABLE sl_sales_bom ADD COLUMN IF NOT EXISTS version_no INTEGER NOT NULL DEFAULT 1;
ALTER TABLE sl_sales_bom ADD COLUMN IF NOT EXISTS remark TEXT;

ALTER TABLE sl_cost_budget ADD COLUMN IF NOT EXISTS updated_at TEXT NOT NULL DEFAULT NOW();

ALTER TABLE sl_self_order_rule ADD COLUMN IF NOT EXISTS max_amount DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE sl_self_order_rule ADD COLUMN IF NOT EXISTS max_qty DOUBLE PRECISION NOT NULL DEFAULT 0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.14', 'sales domain loop columns', 'c43af2be037fd2a707c79438053150b28b20a8d3f638ca303b93d575510e21df')
ON CONFLICT (version) DO NOTHING;
