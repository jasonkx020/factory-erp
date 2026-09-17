-- v1.0.31: 物理删除工序类型列。工序以编码+名称为准；出入库语义由工艺步骤属性驱动。

ALTER TABLE pd_process DROP COLUMN IF EXISTS process_type;

-- 边界/出入库工序补齐标志（不依赖已删除的 process_type）
UPDATE pd_process SET is_handover_point=1 WHERE code IN ('GATE_IN', 'HANDOVER') AND COALESCE(is_handover_point,0)=0;

-- 已有路由步骤：入库/出库工序强制带上步骤属性
UPDATE pd_routing_step SET auto_stock_in=1
WHERE process_id IN (SELECT id FROM pd_process WHERE code IN ('IN_RAW', 'IN_SEMI', 'IN_FG'))
  AND COALESCE(auto_stock_in,0)=0;

UPDATE pd_routing_step SET auto_stock_out=1
WHERE process_id IN (SELECT id FROM pd_process WHERE code IN ('OUT_RAW', 'OUT_DICE'))
  AND COALESCE(auto_stock_out,0)=0;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.31', 'drop process_type; process linkage via routing step attrs', '4604aa509828eb7c51ef47422808960206ab910d194b84b8f8942d56679b848c')
ON CONFLICT (version) DO NOTHING;
