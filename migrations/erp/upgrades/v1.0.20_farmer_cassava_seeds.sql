-- v1.0.20: seed cassava person suppliers for inbound / weigh testing
-- 兼容：旧库写 pur_farmer；新装（无 pur_farmer）写 pur_supplier.party_kind=person

DO $seed$
BEGIN
  IF to_regclass('pur_farmer') IS NOT NULL THEN
    INSERT INTO pur_farmer(code, name, mobile, origin, trace_code, trace_code_prefix, status, remark, default_unit_price)
    VALUES
     ('FM01', '黄桂生', '13807710001', '南宁武鸣', 'FM01', 'FM01', 'active', '开发种子·鲜薯入厂', 1.20),
     ('FM02', '李秀兰', '13807710002', '南宁横州', 'FM02', 'FM02', 'active', '开发种子·鲜薯入厂', 1.18),
     ('FM03', '韦建国', '13907710003', '南宁宾阳', 'FM03', 'FM03', 'active', '开发种子·鲜薯入厂', 1.22),
     ('FM04', '覃金莲', '13707710004', '钦州灵山', 'FM04', 'FM04', 'active', '开发种子·鲜薯入厂', 1.15),
     ('FM05', '陈木生', '13607710005', '北海合浦', 'FM05', 'FM05', 'active', '开发种子·鲜薯入厂', 1.25),
     ('FM06', '农福田', '13507710006', '崇左扶绥', 'FM06', 'FM06', 'active', '开发种子·鲜薯入厂', 1.16),
     ('FM07', '陆阿婆', '13407710007', '贵港桂平', 'FM07', 'FM07', 'active', '开发种子·鲜薯入厂', 1.10),
     ('FM08', '门口过磅点', '13307710008', '厂区地磅', 'FM08', 'FM08', 'active', '开发种子·现场临时户', 1.20)
    ON CONFLICT (code) DO NOTHING;
  ELSE
    INSERT INTO pur_supplier(code, name, party_kind, supplier_type, status, mobile, origin, trace_code_prefix, remark, default_unit_price)
    VALUES
     ('FM01', '黄桂生', 'person', 'raw', 'qualified', '13807710001', '南宁武鸣', 'FM01', '开发种子·鲜薯入厂', 1.20),
     ('FM02', '李秀兰', 'person', 'raw', 'qualified', '13807710002', '南宁横州', 'FM02', '开发种子·鲜薯入厂', 1.18),
     ('FM03', '韦建国', 'person', 'raw', 'qualified', '13907710003', '南宁宾阳', 'FM03', '开发种子·鲜薯入厂', 1.22),
     ('FM04', '覃金莲', 'person', 'raw', 'qualified', '13707710004', '钦州灵山', 'FM04', '开发种子·鲜薯入厂', 1.15),
     ('FM05', '陈木生', 'person', 'raw', 'qualified', '13607710005', '北海合浦', 'FM05', '开发种子·鲜薯入厂', 1.25),
     ('FM06', '农福田', 'person', 'raw', 'qualified', '13507710006', '崇左扶绥', 'FM06', '开发种子·鲜薯入厂', 1.16),
     ('FM07', '陆阿婆', 'person', 'raw', 'qualified', '13407710007', '贵港桂平', 'FM07', '开发种子·鲜薯入厂', 1.10),
     ('FM08', '门口过磅点', 'person', 'raw', 'qualified', '13307710008', '厂区地磅', 'FM08', '开发种子·现场临时户', 1.20)
    ON CONFLICT (code) DO NOTHING;
  END IF;
END
$seed$;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.20', 'seed cassava farmers for inbound / weigh testing', '5e26311a49fcde31072db07e40342b3b7029dc0ae471f94f8ecd1b16e0e41731')
ON CONFLICT (version) DO NOTHING;
