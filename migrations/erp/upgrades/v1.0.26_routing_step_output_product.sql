-- v1.0.26: output product per routing step (process material state)

ALTER TABLE pd_routing_step ADD COLUMN IF NOT EXISTS output_product_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_pd_routing_step_output_product ON pd_routing_step (routing_id, output_product_id);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.26', 'routing step output product', '837d5e0e18cb9423835a14a4add0d4c7a819ad238861f18dee5070e96ca02d02')
ON CONFLICT (version) DO NOTHING;
