-- v1.0.1: process issue move
-- 工序板领料：占用(pd_process_issue) + 下道/入库事件(pd_process_move)
-- inv_box_code.weight = 当前工序可领剩余 kg（同一板码贯穿多工序）

CREATE TABLE IF NOT EXISTS pd_process_issue (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL,
  step_id BIGINT NOT NULL DEFAULT 0,
  worker_id BIGINT NOT NULL DEFAULT 0,
  issue_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  returned_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  completed_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pd_process_issue_kg_chk CHECK (returned_kg + completed_kg <= issue_kg + 0.0001)
);

CREATE INDEX IF NOT EXISTS idx_pd_process_issue_board ON pd_process_issue (board_id, process_id, status);
CREATE INDEX IF NOT EXISTS idx_pd_process_issue_worker ON pd_process_issue (worker_id, status);

CREATE TABLE IF NOT EXISTS pd_process_move (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  from_process_id BIGINT NOT NULL DEFAULT 0,
  from_step_id BIGINT NOT NULL DEFAULT 0,
  to_process_id BIGINT,
  to_step_id BIGINT,
  to_worker_id BIGINT NOT NULL DEFAULT 0,
  kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  move_kind TEXT NOT NULL DEFAULT 'next',
  issue_ids TEXT NOT NULL DEFAULT '',
  created_by BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_process_move_board ON pd_process_move (board_id, created_at);

CREATE TABLE IF NOT EXISTS pd_process_move_alloc (
  id BIGSERIAL PRIMARY KEY,
  move_id BIGINT NOT NULL,
  issue_id BIGINT NOT NULL,
  kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pd_process_move_alloc_move ON pd_process_move_alloc (move_id);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.1', 'process issue move', 'e43dcc8a68fa6819c019cb87b23c5ee890685c202be03e9e4e4ca2691bc3431b')
ON CONFLICT (version) DO NOTHING;
