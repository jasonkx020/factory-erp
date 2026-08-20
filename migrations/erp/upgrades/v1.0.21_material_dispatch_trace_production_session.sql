-- v1.0.21: material dispatch + trace production session + process seeds

-- 派料业务单：扫板码+工牌，指定工序重量，仓/工序出料，完工拍照
CREATE TABLE IF NOT EXISTS pd_material_dispatch (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL DEFAULT 0,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  worker_id BIGINT NOT NULL DEFAULT 0,
  worker_name TEXT NOT NULL DEFAULT '',
  badge_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL DEFAULT 0,
  process_name TEXT NOT NULL DEFAULT '',
  weight_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  reweigh_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  source_kind TEXT NOT NULL DEFAULT 'warehouse',
  status TEXT NOT NULL DEFAULT 'in_progress',
  unit_price DOUBLE PRECISION NOT NULL DEFAULT 0,
  wage_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  pay_mode TEXT NOT NULL DEFAULT 'none',
  issue_id BIGINT NOT NULL DEFAULT 0,
  photo_url TEXT NOT NULL DEFAULT '',
  confirm_photo_url TEXT NOT NULL DEFAULT '',
  dispatched_by BIGINT NOT NULL DEFAULT 0,
  confirmed_by BIGINT NOT NULL DEFAULT 0,
  remark TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  confirmed_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_material_dispatch_worker ON pd_material_dispatch (worker_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_pd_material_dispatch_dispatcher ON pd_material_dispatch (dispatched_by, created_at);
CREATE INDEX IF NOT EXISTS idx_pd_material_dispatch_trace ON pd_material_dispatch (trace_code, status);
CREATE INDEX IF NOT EXISTS idx_pd_material_dispatch_board ON pd_material_dispatch (board_code, status);

-- 溯源码生产会话（生管启停）
CREATE TABLE IF NOT EXISTS pd_trace_production (
  id BIGSERIAL PRIMARY KEY,
  trace_code TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'in_progress',
  started_by BIGINT NOT NULL DEFAULT 0,
  completed_by BIGINT NOT NULL DEFAULT 0,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  remark TEXT NOT NULL DEFAULT '',
  loss_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  input_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  output_kg DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_pd_trace_production_open
  ON pd_trace_production (trace_code) WHERE status = 'in_progress';

CREATE INDEX IF NOT EXISTS idx_pd_trace_production_code ON pd_trace_production (trace_code, status);

-- 溯源工序启停日志
CREATE TABLE IF NOT EXISTS pd_trace_process_log (
  id BIGSERIAL PRIMARY KEY,
  session_id BIGINT NOT NULL DEFAULT 0,
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL DEFAULT 0,
  process_name TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL DEFAULT 'start',
  actor_user_id BIGINT NOT NULL DEFAULT 0,
  input_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  output_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  remark TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_trace_process_log_session ON pd_trace_process_log (session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_pd_trace_process_log_trace ON pd_trace_process_log (trace_code, created_at);

-- 种子工序补齐：入厂日志节点、切片（与清洗/切断/去芯/出入库并列）
INSERT INTO pd_process(code, name, process_type, is_piecework, is_handover_point)
SELECT 'GATE_IN', '入厂', 'gate', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM pd_process WHERE code='GATE_IN');

INSERT INTO pd_process(code, name, process_type, is_piecework, is_handover_point)
SELECT 'SLICE', '切片', 'slice', 1, 0
WHERE NOT EXISTS (SELECT 1 FROM pd_process WHERE code='SLICE');

INSERT INTO pd_process(code, name, process_type, is_piecework, is_handover_point)
SELECT 'OUT_RAW', '出库', 'outbound', 0, 0
WHERE NOT EXISTS (SELECT 1 FROM pd_process WHERE code='OUT_RAW');

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.21', 'material dispatch trace production session', '255f5b4160c9198baadfa1feb97474841bfa9a4c55abc621d297b55f7db970b8')
ON CONFLICT (version) DO NOTHING;
