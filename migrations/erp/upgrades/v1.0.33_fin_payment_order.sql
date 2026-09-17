-- v1.0.33: 供应商结算在线支付单

CREATE TABLE IF NOT EXISTS fin_payment_order (
  id BIGSERIAL PRIMARY KEY,
  payment_no TEXT NOT NULL UNIQUE,
  settlement_id INTEGER NOT NULL,
  supplier_id INTEGER NOT NULL,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  currency TEXT NOT NULL DEFAULT 'CNY',
  channel TEXT NOT NULL DEFAULT 'alipay_bank',
  status TEXT NOT NULL DEFAULT 'pending_finance',
  payee_name TEXT,
  payee_bank_account TEXT,
  payee_bank_name TEXT,
  payee_mobile TEXT,
  channel_trade_no TEXT,
  fail_reason TEXT,
  fund_account_id INTEGER,
  finance_approved_by INTEGER,
  finance_approved_at TEXT,
  boss_approved_by INTEGER,
  boss_approved_at TEXT,
  paid_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fin_payment_order_settlement ON fin_payment_order(settlement_id);
CREATE INDEX IF NOT EXISTS idx_fin_payment_order_status ON fin_payment_order(status);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.33', 'fin_payment_order for supplier online pay', '2a4c21f09b84fa20ba364a16782ee0fb27013ae63bad941a36cfcc59a2908418')
ON CONFLICT (version) DO NOTHING;
