-- v1.0.6: appr_task display columns used by /approval/tasks
-- 基线表仅有 flow/doc 字段；列表/创建依赖 title、doc_no、amount 等扩展列。

ALTER TABLE appr_task ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE appr_task ADD COLUMN IF NOT EXISTS doc_no TEXT;
ALTER TABLE appr_task ADD COLUMN IF NOT EXISTS amount DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE appr_task ADD COLUMN IF NOT EXISTS applicant_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE appr_task ADD COLUMN IF NOT EXISTS remark TEXT;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.6', 'appr_task title and display columns', '422b984211b0381018544d89b5661970b5341530c364cf632cf490c62fed4ef7')
ON CONFLICT (version) DO NOTHING;
