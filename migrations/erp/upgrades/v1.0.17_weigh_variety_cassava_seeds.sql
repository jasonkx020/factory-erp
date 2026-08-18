-- v1.0.17: seed cassava weigh varieties for first install / existing DBs

INSERT INTO pur_weigh_variety(code, name, sort_no, status, remark)
VALUES
 ('WV-FRESH', '鲜木薯', 10, 'active', '农户鲜薯过磅入厂，入保鲜库'),
 ('WV-SEMI', '半成品（去芯薯肉）', 20, 'active', '外购或厂内半成品过磅入厂，入半成品库'),
 ('WV-FG', '成品入库（袋装木薯丁）', 30, 'active', '成品过磅入库，入成品冷库')
ON CONFLICT (code) DO NOTHING;

UPDATE pur_weigh_variety v
SET default_product_id = p.id, updated_at = NOW()
FROM prd_product p
WHERE v.default_product_id IS NULL AND COALESCE(v.is_deleted,0)=0
  AND (
    (v.code = 'WV-FRESH' AND p.code = 'RM-CASSAVA')
    OR (v.code = 'WV-SEMI' AND p.code = 'SF-COREOUT')
    OR (v.code = 'WV-FG' AND p.code = 'FG-DICED')
  );

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.17', 'seed cassava weigh varieties', 'fd2f8a72d2aebadccf22e794a9eae2ae9dccaf79e6620955b9f3b2c14f8a644d')
ON CONFLICT (version) DO NOTHING;
