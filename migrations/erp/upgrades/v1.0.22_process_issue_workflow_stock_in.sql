-- v1.0.22: process issue workflow stock in
-- 领料单退库申请态/结束确认 + 独立入库申请单

ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS biz_status TEXT NOT NULL DEFAULT 'open';
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS issued_by_employee_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS work_done_at TIMESTAMPTZ;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS work_done_by BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS pending_return_kg DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS pending_reweigh_kg DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS pending_photo_url TEXT NOT NULL DEFAULT '';
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS pending_return_by BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS pending_remark TEXT NOT NULL DEFAULT '';

UPDATE pd_process_issue SET biz_status='open' WHERE COALESCE(biz_status,'')='';
UPDATE pd_process_issue SET issued_by_employee_id=worker_id
  WHERE COALESCE(issued_by_employee_id,0)=0 AND COALESCE(worker_id,0)>0;

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_biz ON pd_process_issue (biz_status, worker_id);
CREATE INDEX IF NOT EXISTS idx_pd_process_issue_issuer ON pd_process_issue (issued_by_employee_id, created_at);

CREATE TABLE IF NOT EXISTS pd_process_stock_in (
  id BIGSERIAL PRIMARY KEY,
  doc_no TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL DEFAULT 0,
  board_id BIGINT NOT NULL DEFAULT 0,
  board_code TEXT NOT NULL DEFAULT '',
  applicant_employee_id BIGINT NOT NULL DEFAULT 0,
  worker_id BIGINT NOT NULL DEFAULT 0,
  apply_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  reweigh_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  photo_url TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending_warehouse',
  issue_ids TEXT NOT NULL DEFAULT '',
  warehouse_id BIGINT NOT NULL DEFAULT 0,
  approved_by BIGINT NOT NULL DEFAULT 0,
  approved_at TIMESTAMPTZ,
  remark TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_process_stock_in_status ON pd_process_stock_in (status, created_at);
CREATE INDEX IF NOT EXISTS idx_pd_process_stock_in_trace ON pd_process_stock_in (trace_code, process_id);
CREATE INDEX IF NOT EXISTS idx_pd_process_stock_in_applicant ON pd_process_stock_in (applicant_employee_id, created_at);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.22', 'process issue workflow stock in', '5f85f5c3aef2ed3c7db6fefb04490ef8c3383182827a9f2800908b2663729e1c')
ON CONFLICT (version) DO NOTHING;
