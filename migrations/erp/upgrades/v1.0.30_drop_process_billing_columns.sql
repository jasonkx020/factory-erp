-- v1.0.30: 物理删除工序表计费字段（计费仅存 pay_process_wage_rate.pay_mode）
-- 工艺步骤 pd_routing_step.is_piecework 保留，与工序定义无关。

ALTER TABLE pd_process DROP COLUMN IF EXISTS pay_mode;
ALTER TABLE pd_process DROP COLUMN IF EXISTS is_piecework;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.30', 'drop process billing columns', '9b13c3668063e24a050d4bb0b4290e004a8565c0e9ef1725bbd534f791139483')
ON CONFLICT (version) DO NOTHING;
