-- v1.0.5: process pay_mode + station flow log
-- 工序计费方式（计重/计件/默认）与过站全流水；产量工钱与物料完成解耦。

ALTER TABLE pd_process ADD COLUMN IF NOT EXISTS pay_mode TEXT NOT NULL DEFAULT 'none';

UPDATE pd_process SET pay_mode='weight' WHERE COALESCE(is_piecework,0)=1 AND (pay_mode='' OR pay_mode='none');
UPDATE pd_process SET is_piecework=1 WHERE pay_mode IN ('weight','piece');
UPDATE pd_process SET is_piecework=0 WHERE pay_mode NOT IN ('weight','piece');

ALTER TABLE pd_process_issue ADD COLUMN IF NOT EXISTS wage_settled_kg DOUBLE PRECISION NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS pd_station_flow_log (
  id BIGSERIAL PRIMARY KEY,
  event_type TEXT NOT NULL,
  biz_date TEXT NOT NULL DEFAULT '',
  board_id BIGINT NOT NULL DEFAULT 0,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL DEFAULT 0,
  step_id BIGINT NOT NULL DEFAULT 0,
  process_name TEXT NOT NULL DEFAULT '',
  worker_id BIGINT NOT NULL DEFAULT 0,
  worker_name TEXT NOT NULL DEFAULT '',
  badge_code TEXT NOT NULL DEFAULT '',
  actor_user_id BIGINT NOT NULL DEFAULT 0,
  operator_employee_id BIGINT NOT NULL DEFAULT 0,
  kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  pay_mode TEXT NOT NULL DEFAULT 'none',
  emp_type TEXT NOT NULL DEFAULT '',
  rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  ref_type TEXT NOT NULL DEFAULT '',
  ref_id BIGINT NOT NULL DEFAULT 0,
  before_json TEXT,
  after_json TEXT,
  remark TEXT,
  payload_json TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_station_flow_biz_date ON pd_station_flow_log (biz_date, id DESC);
CREATE INDEX IF NOT EXISTS idx_pd_station_flow_board ON pd_station_flow_log (board_code, id DESC);
CREATE INDEX IF NOT EXISTS idx_pd_station_flow_worker ON pd_station_flow_log (worker_id, biz_date);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.5', 'process pay_mode + station flow log', '3521f748257f60cda84c6290ec3996d4b05d89d8421fce48b80d344f6ec99361')
ON CONFLICT (version) DO NOTHING;
