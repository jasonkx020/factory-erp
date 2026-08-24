-- v1.0.25: lock routing on trace production session start

ALTER TABLE pd_trace_production ADD COLUMN IF NOT EXISTS routing_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pd_trace_production ADD COLUMN IF NOT EXISTS product_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_pd_trace_production_routing ON pd_trace_production (routing_id);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.25', 'trace production session routing lock', '1d0ebdc830be6e021c2caf0932a35622be335eb2be9ccfb3215f0789d8d63c27')
ON CONFLICT (version) DO NOTHING;
