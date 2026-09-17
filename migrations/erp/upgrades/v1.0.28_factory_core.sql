-- v1.0.28 factory core: warehouse role, weigh variety warehouse bind, plant, plant scope

ALTER TABLE inv_warehouse ADD COLUMN IF NOT EXISTS warehouse_role TEXT;
UPDATE inv_warehouse SET warehouse_role = warehouse_type
WHERE COALESCE(warehouse_role,'') = '' AND COALESCE(warehouse_type,'') <> '';

ALTER TABLE pur_weigh_variety ADD COLUMN IF NOT EXISTS warehouse_id INTEGER;
ALTER TABLE pur_weigh_variety ADD COLUMN IF NOT EXISTS warehouse_role TEXT;
ALTER TABLE pur_weigh_variety ADD COLUMN IF NOT EXISTS cold_store_type TEXT;

UPDATE pur_weigh_variety SET cold_store_type = 'fresh', warehouse_role = 'raw'
WHERE code = 'WV-FRESH' AND COALESCE(cold_store_type,'') = '';
UPDATE pur_weigh_variety SET cold_store_type = 'semi', warehouse_role = 'semi'
WHERE code = 'WV-SEMI' AND COALESCE(cold_store_type,'') = '';
UPDATE pur_weigh_variety SET cold_store_type = 'fg', warehouse_role = 'fg'
WHERE code = 'WV-FG' AND COALESCE(cold_store_type,'') = '';

CREATE TABLE IF NOT EXISTS sys_plant (
  id BIGSERIAL PRIMARY KEY,
  org_id INTEGER NOT NULL DEFAULT 1,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  is_default INTEGER NOT NULL DEFAULT 0,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

INSERT INTO sys_plant(code, name, status, is_default, remark)
SELECT 'PLANT-01', COALESCE((SELECT name FROM sys_organization WHERE id=1 LIMIT 1), '默认厂区'), 'active', 1, '开厂默认厂区'
WHERE NOT EXISTS (SELECT 1 FROM sys_plant WHERE COALESCE(is_deleted,0)=0);

ALTER TABLE inv_warehouse ADD COLUMN IF NOT EXISTS plant_id INTEGER;
UPDATE inv_warehouse w SET plant_id = (SELECT id FROM sys_plant WHERE is_default=1 AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1)
WHERE plant_id IS NULL;

ALTER TABLE pd_routing ADD COLUMN IF NOT EXISTS plant_id INTEGER;
ALTER TABLE pd_flow_graph ADD COLUMN IF NOT EXISTS plant_id INTEGER;
ALTER TABLE pd_shift ADD COLUMN IF NOT EXISTS plant_id INTEGER;
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS plant_id INTEGER;

CREATE TABLE IF NOT EXISTS iam_role_plant_scope (
  id BIGSERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL,
  plant_id INTEGER NOT NULL,
  UNIQUE(role_id, plant_id)
);

CREATE TABLE IF NOT EXISTS iam_user_plant_scope (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  plant_id INTEGER NOT NULL,
  UNIQUE(user_id, plant_id)
);

CREATE INDEX IF NOT EXISTS idx_inv_warehouse_role ON inv_warehouse (warehouse_role);
CREATE INDEX IF NOT EXISTS idx_pur_weigh_variety_wh ON pur_weigh_variety (warehouse_id);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.28', 'factory core warehouse role plant scope', '3f367f7c4fd63515479e42b7c09f003cf61f72157343614b7318d57e6745ea1f')
ON CONFLICT (version) DO NOTHING;
