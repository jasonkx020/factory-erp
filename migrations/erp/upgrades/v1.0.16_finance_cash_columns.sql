-- v1.0.16: finance cash columns（结算/预付挂资金账户）

DO $cash$
BEGIN
  IF to_regclass('pur_farmer_settlement') IS NOT NULL THEN
    ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
  END IF;
  IF to_regclass('pur_supplier_settlement') IS NOT NULL THEN
    ALTER TABLE pur_supplier_settlement ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
  END IF;
END
$cash$;

ALTER TABLE fin_prepay_prepaid ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
ALTER TABLE fin_sales_return_finance ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
ALTER TABLE sl_sales_order ADD COLUMN IF NOT EXISTS received_amount DOUBLE PRECISION NOT NULL DEFAULT 0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.16', 'finance cash columns', 'df5c4575b37494d2078d698c495094abcbc4de2290e66079654b67a1a1a16732')
ON CONFLICT (version) DO NOTHING;
