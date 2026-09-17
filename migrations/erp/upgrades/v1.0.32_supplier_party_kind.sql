-- v1.0.32: 农户并入供应商（party_kind=person|enterprise）；单据 farmer_id→supplier_id
-- 幂等：新装 baseline 已含最终结构时本脚本可跳过数据迁移；存量库有 pur_farmer 时执行迁入。

-- 1) 供应商扩展
ALTER TABLE pur_supplier ADD COLUMN IF NOT EXISTS party_kind TEXT NOT NULL DEFAULT 'enterprise';
ALTER TABLE pur_supplier ADD COLUMN IF NOT EXISTS mobile TEXT;
ALTER TABLE pur_supplier ADD COLUMN IF NOT EXISTS origin TEXT;
ALTER TABLE pur_supplier ADD COLUMN IF NOT EXISTS trace_code_prefix TEXT;
ALTER TABLE pur_supplier ADD COLUMN IF NOT EXISTS default_unit_price DOUBLE PRECISION NOT NULL DEFAULT 0;

UPDATE pur_supplier SET party_kind='enterprise' WHERE COALESCE(NULLIF(party_kind,''),'')='';

-- 2) 单据列：始终确保 supplier_id 存在
ALTER TABLE pur_weigh_ticket ADD COLUMN IF NOT EXISTS supplier_id INTEGER;
ALTER TABLE pur_inbound_arrival ADD COLUMN IF NOT EXISTS supplier_id INTEGER;
ALTER TABLE pur_trace_lot ADD COLUMN IF NOT EXISTS supplier_id INTEGER;
ALTER TABLE pur_trace_batch_code ADD COLUMN IF NOT EXISTS supplier_id INTEGER;
ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS supplier_id INTEGER;

-- 3) 仅当 pur_farmer 仍存在时迁入个人供应商并回填单据
DO $mig$
BEGIN
  IF to_regclass('pur_farmer') IS NULL THEN
    RETURN;
  END IF;

  CREATE TABLE IF NOT EXISTS _mig_farmer_supplier_map (
    farmer_id BIGINT PRIMARY KEY,
    supplier_id BIGINT NOT NULL
  );

  INSERT INTO pur_supplier(
    code, name, short_name, supplier_type, status, party_kind,
    mobile, origin, trace_code_prefix, default_unit_price, remark, is_deleted
  )
  SELECT
    CASE WHEN EXISTS (SELECT 1 FROM pur_supplier s WHERE s.code=f.code)
      THEN 'P-' || f.code ELSE f.code END,
    f.name,
    NULL,
    'raw',
    CASE WHEN COALESCE(f.status,'active') IN ('active','qualified') THEN 'qualified' ELSE COALESCE(f.status,'qualified') END,
    'person',
    f.mobile,
    f.origin,
    COALESCE(NULLIF(f.trace_code_prefix,''), NULLIF(f.trace_code,''), f.code),
    COALESCE(f.default_unit_price, 0),
    COALESCE(f.remark, ''),
    COALESCE(f.is_deleted, 0)
  FROM pur_farmer f
  WHERE NOT EXISTS (
    SELECT 1 FROM _mig_farmer_supplier_map m WHERE m.farmer_id=f.id
  )
  AND NOT EXISTS (
    SELECT 1 FROM pur_supplier s
    WHERE s.party_kind='person' AND s.code IN (f.code, 'P-' || f.code)
  );

  INSERT INTO _mig_farmer_supplier_map(farmer_id, supplier_id)
  SELECT f.id, s.id
  FROM pur_farmer f
  JOIN pur_supplier s ON s.party_kind='person'
    AND s.code IN (f.code, 'P-' || f.code)
  ON CONFLICT (farmer_id) DO NOTHING;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='pur_weigh_ticket' AND column_name='farmer_id'
  ) THEN
    UPDATE pur_weigh_ticket w SET supplier_id=m.supplier_id
    FROM _mig_farmer_supplier_map m WHERE w.farmer_id=m.farmer_id AND COALESCE(w.supplier_id,0)=0;
    UPDATE pur_weigh_ticket SET supplier_id=COALESCE(NULLIF(supplier_id,0), farmer_id) WHERE COALESCE(supplier_id,0)=0;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='pur_inbound_arrival' AND column_name='farmer_id'
  ) THEN
    UPDATE pur_inbound_arrival a SET supplier_id=m.supplier_id
    FROM _mig_farmer_supplier_map m WHERE a.farmer_id=m.farmer_id AND COALESCE(a.supplier_id,0)=0;
    UPDATE pur_inbound_arrival SET supplier_id=COALESCE(NULLIF(supplier_id,0), farmer_id) WHERE COALESCE(supplier_id,0)=0;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='pur_trace_lot' AND column_name='farmer_id'
  ) THEN
    UPDATE pur_trace_lot t SET supplier_id=m.supplier_id
    FROM _mig_farmer_supplier_map m WHERE t.farmer_id=m.farmer_id AND COALESCE(t.supplier_id,0)=0;
    UPDATE pur_trace_lot SET supplier_id=COALESCE(NULLIF(supplier_id,0), farmer_id) WHERE COALESCE(supplier_id,0)=0;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='pur_trace_batch_code' AND column_name='farmer_id'
  ) THEN
    UPDATE pur_trace_batch_code c SET supplier_id=m.supplier_id
    FROM _mig_farmer_supplier_map m WHERE c.farmer_id=m.farmer_id AND COALESCE(c.supplier_id,0)=0;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name='inv_box_code' AND column_name='farmer_id'
  ) THEN
    UPDATE inv_box_code b SET supplier_id=m.supplier_id
    FROM _mig_farmer_supplier_map m WHERE b.farmer_id=m.farmer_id AND COALESCE(b.supplier_id,0)=0;
  END IF;

  IF to_regclass('pur_farmer_settlement') IS NOT NULL THEN
    ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS supplier_id INTEGER;
    IF EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = current_schema() AND table_name='pur_farmer_settlement' AND column_name='farmer_id'
    ) THEN
      UPDATE pur_farmer_settlement s SET supplier_id=m.supplier_id
      FROM _mig_farmer_supplier_map m WHERE s.farmer_id=m.farmer_id AND COALESCE(s.supplier_id,0)=0;
      UPDATE pur_farmer_settlement SET supplier_id=COALESCE(NULLIF(supplier_id,0), farmer_id) WHERE COALESCE(supplier_id,0)=0;
      ALTER TABLE pur_farmer_settlement DROP COLUMN IF EXISTS farmer_id;
    END IF;
    IF to_regclass('pur_supplier_settlement') IS NULL THEN
      ALTER TABLE pur_farmer_settlement RENAME TO pur_supplier_settlement;
    END IF;
  END IF;

  DROP TABLE IF EXISTS pur_farmer;
  DROP TABLE IF EXISTS _mig_farmer_supplier_map;
END
$mig$;

-- 4) 清理残留 farmer_id（无论是否走过 pur_farmer 迁入）
ALTER TABLE pur_weigh_ticket DROP COLUMN IF EXISTS farmer_id;
ALTER TABLE pur_inbound_arrival DROP COLUMN IF EXISTS farmer_id;
ALTER TABLE pur_trace_lot DROP COLUMN IF EXISTS farmer_id;
ALTER TABLE pur_trace_batch_code DROP COLUMN IF EXISTS farmer_id;
ALTER TABLE inv_box_code DROP COLUMN IF EXISTS farmer_id;

-- 5) 财务 party_type
UPDATE fin_prepay_prepaid SET party_type='supplier' WHERE party_type='farmer';
UPDATE fin_arap_adjust SET party_type='supplier' WHERE party_type='farmer';

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.32', 'supplier party_kind person/enterprise; merge farmer', '6a7fa7230f401709ce102e18689df0e61eb6ca4c66c9bcbb08d067e70066c423')
ON CONFLICT (version) DO NOTHING;
