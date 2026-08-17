-- v1.0.3: weigh ticket unit price
-- 过磅单补齐单价、车牌、三项费用、确认时间，与列表/详情 SQL 对齐。

ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS unit_price DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS plate_no TEXT;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS freight_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS loading_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS weigh_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS confirmed_at TEXT;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.3', 'weigh ticket unit price', 'd72780fe3c2f51ec28656ec3dc467e558fa25b576665f2a5e30611f09716f62a')
ON CONFLICT (version) DO NOTHING;
