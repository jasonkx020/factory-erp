-- v1.0.8: 溯源批号可复用生命周期（未启用/过站中/已结束）+ 首单锁定农户/产品
-- used → in_progress；一码可挂多张 gate 过磅单（同农户同产品）；结算仍按单张过磅单。

ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS farmer_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS product_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS variety TEXT;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS first_weigh_ticket_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS ended_at TEXT;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS ended_by INTEGER;

UPDATE pur_trace_batch_code SET status = 'in_progress' WHERE status = 'used';

-- 从首张 gate 过磅单回填绑定（按 batch_no）
UPDATE pur_trace_batch_code c
SET
  farmer_id = w.farmer_id,
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
  AND (c.farmer_id IS NULL OR c.farmer_id = 0);

-- 一码多单：pur_trace_lot.trace_code 不再全局唯一，改为按过磅单唯一
ALTER TABLE pur_trace_lot DROP CONSTRAINT IF EXISTS pur_trace_lot_trace_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_pur_trace_lot_weigh_ticket
  ON pur_trace_lot (weigh_ticket_id)
  WHERE weigh_ticket_id IS NOT NULL AND weigh_ticket_id > 0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.8', 'trace batch reuse lifecycle and lock farmer product', '1a1dcc1aec71679c95441a2de2a96bd9da1d630727faafa6a45961b7916850e8')
ON CONFLICT (version) DO NOTHING;
