-- v1.0.25: cassava planting management (plots, contracts, field logs, harvest plans)

CREATE TABLE IF NOT EXISTS plant_plot (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  farmer_id BIGINT NOT NULL DEFAULT 0,
  area_mu DOUBLE PRECISION NOT NULL DEFAULT 0,
  location TEXT,
  soil_type TEXT,
  irrigation_type TEXT,
  variety TEXT NOT NULL DEFAULT '鲜木薯',
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS plant_contract (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  farmer_id BIGINT NOT NULL,
  plot_id BIGINT NOT NULL DEFAULT 0,
  variety TEXT NOT NULL DEFAULT '鲜木薯',
  area_mu DOUBLE PRECISION NOT NULL DEFAULT 0,
  unit_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  start_date TEXT NOT NULL,
  end_date TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS plant_field_log (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  plot_id BIGINT NOT NULL,
  farmer_id BIGINT NOT NULL DEFAULT 0,
  log_type TEXT NOT NULL DEFAULT 'other',
  biz_date TEXT NOT NULL,
  operator_name TEXT,
  content TEXT,
  qty DOUBLE PRECISION NOT NULL DEFAULT 0,
  unit TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS plant_harvest_plan (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL UNIQUE,
  plot_id BIGINT NOT NULL,
  farmer_id BIGINT NOT NULL,
  variety TEXT NOT NULL DEFAULT '鲜木薯',
  plan_date TEXT NOT NULL,
  estimate_weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  arrival_id BIGINT NOT NULL DEFAULT 0,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW(),
  is_deleted INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_plant_plot_farmer ON plant_plot (farmer_id, status);
CREATE INDEX IF NOT EXISTS idx_plant_contract_farmer ON plant_contract (farmer_id, status);
CREATE INDEX IF NOT EXISTS idx_plant_field_log_plot ON plant_field_log (plot_id, biz_date);
CREATE INDEX IF NOT EXISTS idx_plant_harvest_plan_date ON plant_harvest_plan (plan_date, status);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.25', 'cassava planting management tables', '376ea85ef616deabe4ae56bc0d345f949e65ff6627cc1323d519754224448b65')
ON CONFLICT (version) DO NOTHING;
