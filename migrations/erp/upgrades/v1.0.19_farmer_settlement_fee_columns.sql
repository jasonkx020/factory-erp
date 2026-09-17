-- v1.0.19: supplier settlement fee columns（兼容旧表名 pur_farmer_settlement）

DO $fee$
BEGIN
  IF to_regclass('pur_farmer_settlement') IS NOT NULL THEN
    ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS goods_amount DOUBLE PRECISION NOT NULL DEFAULT 0;
    ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS freight_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
    ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS loading_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
    ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS weigh_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
  END IF;
  IF to_regclass('pur_supplier_settlement') IS NOT NULL THEN
    ALTER TABLE pur_supplier_settlement ADD COLUMN IF NOT EXISTS goods_amount DOUBLE PRECISION NOT NULL DEFAULT 0;
    ALTER TABLE pur_supplier_settlement ADD COLUMN IF NOT EXISTS freight_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
    ALTER TABLE pur_supplier_settlement ADD COLUMN IF NOT EXISTS loading_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
    ALTER TABLE pur_supplier_settlement ADD COLUMN IF NOT EXISTS weigh_fee DOUBLE PRECISION NOT NULL DEFAULT 0;
  END IF;
END
$fee$;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.19', 'farmer settlement fee columns', '9a98ab451aa3d8661bd3e3f147275a385bdd94548ed6b5f2b730dc41d2f35991')
ON CONFLICT (version) DO NOTHING;
