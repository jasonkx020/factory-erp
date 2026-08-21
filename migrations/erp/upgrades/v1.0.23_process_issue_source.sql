-- v1.0.23: process issue source (warehouse|process) + trace indexes
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_trace_proc
  ON pd_process_issue (trace_code, process_id, worker_id, status);

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_source
  ON pd_process_issue (source, created_at);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.23', 'process issue source warehouse|process', 'v1.0.23_process_issue_source')
ON CONFLICT (version) DO NOTHING;
