-- v1.0.18: inv_box_code.image_url for inbound box reweigh photos

ALTER TABLE inv_box_code ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.18', 'inv_box_code image_url', '25fec7946eafbaeb0d24e6f682a48993427058e3aa6e0deae77caad2ba629df6')
ON CONFLICT (version) DO NOTHING;
