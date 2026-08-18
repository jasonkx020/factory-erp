-- v1.0.16: finance cash posting columns (farmer pay / prepay / return / order received)

ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
ALTER TABLE fin_prepay_prepaid ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
ALTER TABLE fin_sales_return_finance ADD COLUMN IF NOT EXISTS fund_account_id INTEGER;
ALTER TABLE sl_sales_order ADD COLUMN IF NOT EXISTS received_amount DOUBLE PRECISION NOT NULL DEFAULT 0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.16', 'finance cash posting columns', 'e1ad84710629d37b043517c050e4bbe9fc235700306d83b1100cfc8af2e33ce8')
ON CONFLICT (version) DO NOTHING;

