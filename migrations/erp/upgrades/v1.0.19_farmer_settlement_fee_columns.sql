-- v1.0.19: farmer settlement fee columns

ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS goods_amount DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS freight_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS loading_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS weigh_fee DOUBLE PRECISION NOT NULL DEFAULT 0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.19', 'farmer settlement fee columns', '1d3bb3dbba6a4a70022a9d7cffc3e4b56a30af8ced4c28f7fb99b10349601839')
ON CONFLICT (version) DO NOTHING;
