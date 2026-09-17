-- v1.0.8: 溯源批号可复用生命周期（未启用/过站中/已结束）+ 首单锁定供应商/产品
-- used → in_progress；一码可挂多张 gate 过磅单（同供应商同产品）；结算仍按单张过磅单。
-- 兼容：旧列 farmer_id / 新列 supplier_id（schema 新装已用 supplier_id）

ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS supplier_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS farmer_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS product_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS variety TEXT;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS first_weigh_ticket_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS ended_at TEXT;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS ended_by INTEGER;

UPDATE pur_trace_batch_code SET status = 'in_progress' WHERE status = 'used';

-- 从首张 gate 过磅单回填绑定（按 batch_no）；优先 supplier_id，否则 farmer_id
DO $bf$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='pur_weigh_ticket' AND column_name='supplier_id'
  ) THEN
    UPDATE pur_trace_batch_code c
    SET
      supplier_id = COALESCE(NULLIF(c.supplier_id, 0), w.supplier_id),
      product_id = COALESCE(c.product_id, w.product_id),
      variety = COALESCE(NULLIF(c.variety, ''), w.variety),
      first_weigh_ticket_id = COALESCE(c.first_weigh_ticket_id, w.id)
    FROM (
      SELECT DISTINCT ON (UPPER(batch_no))
        id, UPPER(batch_no) AS bn, supplier_id, product_id, COALESCE(variety, '') AS variety
      FROM pur_weigh_ticket
      WHERE LOWER(COALESCE(receive_kind, '')) = 'gate'
        AND COALESCE(is_deleted, 0) = 0
        AND COALESCE(batch_no, '') <> ''
      ORDER BY UPPER(batch_no), id ASC
    ) w
    WHERE UPPER(c.code) = w.bn
      AND c.status IN ('in_progress', 'ended')
      AND (c.supplier_id IS NULL OR c.supplier_id = 0);
  ELSIF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='pur_weigh_ticket' AND column_name='farmer_id'
  ) THEN
    UPDATE pur_trace_batch_code c
    SET
      farmer_id = w.farmer_id,
      supplier_id = COALESCE(NULLIF(c.supplier_id, 0), w.farmer_id),
      product_id = w.product_id,
      variety = w.variety,
      first_weigh_ticket_id = COALESCE(c.first_weigh_ticket_id, w.id)
    FROM (
      SELECT DISTINCT ON (UPPER(batch_no))
        id, UPPER(batch_no) AS bn, farmer_id, product_id, COALESCE(variety, '') AS variety
      FROM pur_weigh_ticket
      WHERE LOWER(COALESCE(receive_kind, '')) = 'gate'
        AND COALESCE(is_deleted, 0) = 0
        AND COALESCE(batch_no, '') <> ''
      ORDER BY UPPER(batch_no), id ASC
    ) w
    WHERE UPPER(c.code) = w.bn
      AND c.status IN ('in_progress', 'ended')
      AND (COALESCE(c.supplier_id, 0) = 0)
      AND (c.farmer_id IS NULL OR c.farmer_id = 0);
  END IF;
END
$bf$;

-- 一码多单：pur_trace_lot.trace_code 不再全局唯一，改为按过磅单唯一
ALTER TABLE pur_trace_lot DROP CONSTRAINT IF EXISTS pur_trace_lot_trace_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_pur_trace_lot_weigh_ticket
  ON pur_trace_lot (weigh_ticket_id)
  WHERE weigh_ticket_id IS NOT NULL AND weigh_ticket_id > 0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.8', 'trace batch reuse lifecycle and lock farmer product', 'bf6571db3c088e4757eb494fc0e7d582b7f9cde24b1861386c1cd2cf1dc5415f')
ON CONFLICT (version) DO NOTHING;
