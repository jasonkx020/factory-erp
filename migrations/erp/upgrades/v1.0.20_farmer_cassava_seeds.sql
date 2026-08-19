-- v1.0.20: seed cassava farmers for inbound / weigh testing

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

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.20', 'seed cassava farmers for inbound / weigh testing', 'f8c403e39edbf6041b7d2a96fa70306877416dd75b15a19a00605967f4d91801')
ON CONFLICT (version) DO NOTHING;
