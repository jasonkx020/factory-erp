-- v1.0.4: inv_box_code farmer/trace columns
-- 板码补齐溯源/农户字段，与过站 WIP 列表、入库换码 SQL 对齐。

ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS farmer_id INTEGER;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS trace_code TEXT NOT NULL DEFAULT '';
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS origin TEXT;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS receive_date TEXT;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS source_type TEXT;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.4', 'inv_box_code farmer/trace columns', 'b6c8a3a56e2f310a284d387b76700887ebbbb643747210b48b9926ccebe63a98')
ON CONFLICT (version) DO NOTHING;
