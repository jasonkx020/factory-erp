-- v1.0.2: process yield snapshot
-- 整板完工后按工序×板写一次扣损；溯源下全部板完成后再写溯源快照。UNIQUE 防重算。

CREATE TABLE IF NOT EXISTS pd_board_process_yield (
  id BIGSERIAL PRIMARY KEY,
  board_id BIGINT NOT NULL,
  board_code TEXT NOT NULL DEFAULT '',
  trace_code TEXT NOT NULL DEFAULT '',
  process_id BIGINT NOT NULL,
  input_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  output_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (board_id, process_id)
);

CREATE INDEX IF NOT EXISTS idx_pd_board_process_yield_trace ON pd_board_process_yield (trace_code, process_id);

CREATE TABLE IF NOT EXISTS pd_trace_process_yield (
  id BIGSERIAL PRIMARY KEY,
  trace_code TEXT NOT NULL,
  process_id BIGINT NOT NULL,
  input_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  output_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  loss_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  board_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (trace_code, process_id)
);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.2', 'process yield snapshot', 'c42c35432b5b7fae95efd72e23ccbefb3b92713de778088069d0cb4179d41744')
ON CONFLICT (version) DO NOTHING;
