-- v1.0.24: process issue from/to process + warehouse pending issue
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS from_location_type TEXT NOT NULL DEFAULT '';
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS from_process_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS to_process_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS warehouse_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS assigned_board_code TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_from_proc
  ON pd_process_issue (trace_code, from_process_id, status);

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_to_proc
  ON pd_process_issue (trace_code, to_process_id, status);

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_biz_pending
  ON pd_process_issue (biz_status, created_at);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.24', 'process issue from/to process and warehouse pending', '548c7cc791a0c7b47fb39fffa0314370ed9d57b3a58d4eb8bc9752c6d35c9e9c')
ON CONFLICT (version) DO NOTHING;
