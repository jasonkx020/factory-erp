-- v1.0.29: 工序计费方式落在工价表（无工序表回退）
-- pay_mode 仅存于 pay_process_wage_rate；启用工价且 pay_mode 为 weight|piece 才计费。

ALTER TABLE pay_process_wage_rate ADD COLUMN IF NOT EXISTS pay_mode TEXT NOT NULL DEFAULT 'none';

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.29', 'wage rate pay_mode', 'ad8835be31416de276f873ef3e6b3864b7d09ae841a2b53f213ec7eabdcb0607')
ON CONFLICT (version) DO NOTHING;
