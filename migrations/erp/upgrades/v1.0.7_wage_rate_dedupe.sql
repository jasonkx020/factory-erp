-- v1.0.7: dedupe process wage rates; one active rate per process
-- 开发反复 seed / 多次新建导致同工序多条 active，列表看起来「重复」。

-- 保留每个 process_id 最新一条 active，其余 active 置为 inactive
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY process_id ORDER BY id DESC) AS rn
  FROM pay_process_wage_rate
  WHERE status = 'active'
)
UPDATE pay_process_wage_rate r
SET status = 'inactive'
FROM ranked x
WHERE r.id = x.id AND x.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_pay_process_wage_rate_active_process
  ON pay_process_wage_rate (process_id)
  WHERE status = 'active';

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.7', 'dedupe process wage rates one active per process', 'ae69d2e81f5bfbee879ee5ff9c19fc99405e1227c9a264412dd063efcc438fdf')
ON CONFLICT (version) DO NOTHING;
