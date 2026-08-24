-- factory-erp PostgreSQL data-dev seed

-- 开发种子：组织 / 仓 / 工序 / 角色分组 / admin
-- 默认密码 admin123（bcrypt），仅用于本地开发

INSERT INTO sys_organization(id, code, name, status) VALUES (1, 'ORG001', '桂南木薯加工厂', 'active');

-- 公司架构：一级总经办 → 二级职能/生产部门 → 三级车间（班组不进第 4 级树）
INSERT INTO sys_department(id, org_id, parent_id, code, name, dept_type) VALUES
 (1, 1, NULL, 'HQ01', '总经办', 'normal'),
 (3, 1, 1, 'D-PROD', '生产部', 'normal'),
 (4, 1, 1, 'D-WH', '仓储部', 'normal'),
 (5, 1, 1, 'D-PUR', '采购部', 'normal'),
 (7, 1, 1, 'D-QC', '质检部', 'normal'),
 (8, 1, 1, 'D-HR', '人事行政部', 'normal'),
 (9, 1, 1, 'D-FIN', '财务部', 'normal'),
 (2, 1, 3, 'WS01', '一车间', 'workshop'),
 (10, 1, 3, 'WS02', '二车间', 'workshop')
ON CONFLICT DO NOTHING;

INSERT INTO pd_work_team(id, dept_id, code, name) VALUES
 (1, 2, 'T01', '去皮一组'),
 (2, 2, 'T02', '切断一组'),
 (3, 10, 'T03', '切块一组')
ON CONFLICT DO NOTHING;

INSERT INTO inv_warehouse(id, org_id, code, name, warehouse_type) VALUES
 (1, 1, 'WH-RAW', '保鲜库', 'raw'),
 (2, 1, 'WH-SEMI', '半成品库', 'semi'),
 (3, 1, 'WH-FG', '成品冷库', 'finished')
ON CONFLICT DO NOTHING;

INSERT INTO pd_process(id, code, name, process_type, is_piecework, is_handover_point) VALUES
 (1, 'PEEL', '去皮', 'peel', 1, 0),
 (2, 'HANDOVER', '收货卡点', 'other', 0, 1),
 (3, 'CUT', '切断', 'cut', 0, 0),
 (4, 'CORE', '去芯', 'core', 1, 0),
 (5, 'DICE', '切块', 'dice', 1, 0),
 (6, 'BAG', '过筛装袋', 'bag', 0, 0),
 (7, 'WASH', '清洗', 'wash', 0, 0),
 (8, 'IN_RAW', '原料入库', 'inbound', 0, 0),
 (9, 'IN_SEMI', '半成品入库', 'inbound', 0, 0),
 (10, 'OUT_DICE', '出库切块', 'outbound', 1, 0),
 (11, 'IN_FG', '成品入库', 'inbound', 0, 0)
ON CONFLICT DO NOTHING;

INSERT INTO iam_login_policy(
  id, max_fail_count, lock_minutes, session_ttl_min, password_min_len,
  password_require_letter, password_require_digit, password_require_special, password_history
) VALUES (1, 5, 30, 120, 8, 1, 1, 0, 5)
ON CONFLICT DO NOTHING;

INSERT INTO iam_admin_group(id, code, name, remark, sort_no) VALUES
 (1, 'g_platform', '平台管理组', '系统管理员', 10),
 (2, 'g_biz', '经营决策组', '老板', 20),
 (3, 'g_ops', '业务作业组', '采购', 30),
 (4, 'g_plant', '仓储生产组', '仓管计划车间质检', 40),
 (5, 'g_line', '一线作业组', '计件固定工', 50),
 (6, 'g_func', '职能管理组', '人事薪资财务', 60)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role(id, code, name, data_scope_type, is_system, remark) VALUES
 (1, 'sys_admin', '系统管理员', 'all', 1, '系统管理+角色管理'),
 (2, 'boss', '老板', 'all', 1, '驾驶舱/成本只读'),
 (4, 'purchase', '采购员', 'all', 1, '采购'),
 (5, 'warehouse', '仓管员', 'warehouse', 1, '库存按仓'),
 (6, 'planner', '生产计划', 'all', 1, '生产计划'),
 (7, 'foreman', '车间主任', 'dept_workshop', 1, '车间调度'),
 (8, 'piece', '计件工', 'self', 1, '报工领料'),
 (9, 'fixed', '固定工', 'team', 1, '收货质检'),
 (10, 'qc', '质检员', 'all', 1, '质检'),
 (11, 'hr', '人事', 'all', 1, '人事'),
 (12, 'payroll', '薪资员', 'all', 1, '工资'),
 (13, 'finance', '财务', 'all', 1, '财务')
ON CONFLICT DO NOTHING;

INSERT INTO iam_admin_group_role(group_id, role_id) VALUES
 (1,1),(2,2),(3,4),(4,5),(4,6),(4,7),(4,10),(5,8),(5,9),(6,11),(6,12),(6,13)
ON CONFLICT DO NOTHING;

INSERT INTO iam_permission(code, name, domain, module, action) VALUES
 ('人事管理:角色管理:查看', '查看角色管理', '人事管理', '角色管理', '查看'),
 ('人事管理:角色管理:编辑', '编辑角色管理', '人事管理', '角色管理', '编辑'),
 ('系统管理:自定义权限:查看', '查看自定义权限', '系统管理', '自定义权限', '查看'),
 ('系统管理:自定义权限:编辑', '编辑自定义权限', '系统管理', '自定义权限', '编辑'),
 ('系统管理:自定义菜单:编辑', '编辑自定义菜单', '系统管理', '自定义菜单', '编辑'),
 ('系统管理:登录控制:编辑', '编辑登录控制', '系统管理', '登录控制', '编辑'),
 ('系统管理:账户冻结:编辑', '账户冻结', '系统管理', '账户冻结', '编辑'),
 ('生产管理:扫码报工:新增', '扫码报工新增', '生产管理', '扫码报工', '新增'),
 ('生产管理:扫码报工:查看', '扫码报工查看', '生产管理', '扫码报工', '查看'),
 ('生产管理:联动式领料:新增', '联动领料', '生产管理', '联动式领料', '新增'),
 ('生产管理:生产派工:新增', '生产派工', '生产管理', '生产派工', '新增'),
 ('库存管理:库存查询:查看', '库存查询', '库存管理', '库存查询', '查看'),
 ('产品管理:产品档案:查看', '产品档案查看', '产品管理', '产品档案', '查看'),
 ('产品管理:产品档案:新增', '产品档案新增', '产品管理', '产品档案', '新增'),
 ('工资管理:工序工资:查看', '工序工资', '工资管理', '工序工资', '查看'),
 ('财务管理:成本核算:查看', '成本核算', '财务管理', '成本核算', '查看'),
 ('财务管理:成本明细溯源表:查看', '成本明细溯源表', '财务管理', '成本明细溯源表', '查看')
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 1, id FROM iam_permission
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 8, id FROM iam_permission WHERE code IN (
  '生产管理:扫码报工:新增','生产管理:扫码报工:查看','生产管理:联动式领料:新增','工资管理:工序工资:查看'
)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 9, id FROM iam_permission WHERE code IN (
  '生产管理:扫码报工:新增','生产管理:扫码报工:查看'
)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 7, id FROM iam_permission WHERE code IN (
  '生产管理:扫码报工:新增','生产管理:扫码报工:查看','生产管理:联动式领料:新增','生产管理:生产派工:新增','工资管理:工序工资:查看'
)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 6, id FROM iam_permission WHERE code IN (
  '生产管理:生产派工:新增','工资管理:工序工资:查看'
)
ON CONFLICT DO NOTHING;

INSERT INTO iam_field_policy(role_id, field_key, field_name, visible, editable) VALUES
 (8, 'cost_price', '成本价', 0, 0),
 (8, 'gross_margin', '毛利', 0, 0),
 (8, 'other_wage', '他人工资', 0, 0),
 (8, 'own_wage', '本人工资', 1, 0),
 (13, 'cost_price', '成本价', 1, 0),
 (13, 'gross_margin', '毛利', 1, 0),
 (2, 'cost_price', '成本价', 1, 0),
 (2, 'gross_margin', '毛利', 1, 0)
ON CONFLICT DO NOTHING;

INSERT INTO iam_menu_custom(role_id, domain, module, menu_key, visible, sort_no) VALUES
 (8, '生产管理', '工序流水', '生产管理:工序流水', 1, 10),
 (8, '工资管理', '工序工资', '工资管理:工序工资', 1, 30)
ON CONFLICT DO NOTHING;

INSERT INTO hr_employee(id, emp_no, name, org_id, dept_id, team_id, job_title, emp_type, mobile, badge_code, id_card_no, status) VALUES
 (1, 'E0001', '系统管理员', 1, 1, NULL, '系统管理员', 'office', '13800001001', 'EMP-ADMIN', '450103198001011011', 'active')
ON CONFLICT DO NOTHING;

-- bcrypt(admin123) cost=10
INSERT INTO iam_user(id, login_name, password_hash, employee_id, user_type, status)
VALUES (1, 'admin', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 1, 'admin', 'active');

UPDATE hr_employee SET user_id = 1 WHERE id = 1;

INSERT INTO iam_user_role(user_id, role_id) VALUES (1, 1)
ON CONFLICT DO NOTHING;

INSERT INTO iam_admin_group_user(group_id, user_id) VALUES (1, 1)
ON CONFLICT DO NOTHING;

-- 各角色演示账号（密码均为 admin123，与 admin 相同 bcrypt）
INSERT INTO hr_employee(id, emp_no, name, org_id, dept_id, team_id, job_title, emp_type, mobile, badge_code, id_card_no, status) VALUES
 (10, 'E-PUR', '李采购', 1, 5, NULL, '采购员', 'office', '13800001010', 'EMP-PUR', '450103199104121010', 'active'),
 (11, 'E-QC', '孙质检', 1, 7, NULL, '质检员', 'office', '13800001011', 'EMP-QC', '450103199211091011', 'active'),
 (12, 'E-WH', '黄仓管', 1, 4, NULL, '仓管员', 'warehouse', '13800001012', 'EMP-WH', '450103199006081012', 'active'),
 (13, 'E-FM', '赵主任', 1, 3, NULL, '车间主任', 'office', '13800001013', 'EMP-FM', '450103198703221013', 'active'),
 (14, 'E-PC', '陈计件', 1, 2, 1, '去皮计件工', 'piece', '13800001014', 'EMP-PC', '450103199505051014', 'active'),
 (15, 'E-FX', '刘固定', 1, 2, 2, '收货固定工', 'fixed', '13800001015', 'EMP-FX', '450103199408181015', 'active'),
 (17, 'E-FN', '钱会计', 1, 9, NULL, '会计', 'office', '13800001017', 'EMP-FN', '450103198909091017', 'active'),
 (18, 'E-BS', '韦建国', 1, 1, NULL, '总经理', 'office', '13800001018', 'EMP-BS', '450103197501011018', 'active'),
 (19, 'E-PL', '吴计划', 1, 3, NULL, '生产计划员', 'office', '13800001019', 'EMP-PL', '450103198812011019', 'active'),
 (20, 'E-HR', '郑人事', 1, 8, NULL, '人事专员', 'office', '13800001020', 'EMP-HR', '450103199109211020', 'active'),
 (21, 'E-PAY', '冯薪资', 1, 9, NULL, '薪资员', 'office', '13800001021', 'EMP-PAY', '450103199307011021', 'active');

INSERT INTO iam_user(id, login_name, password_hash, employee_id, user_type, status) VALUES
 (10, 'u_purchase', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 10, 'biz', 'active'),
 (11, 'u_qc', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 11, 'biz', 'active'),
 (12, 'u_warehouse', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 12, 'biz', 'active'),
 (13, 'u_foreman', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 13, 'biz', 'active'),
 (14, 'u_piece', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 14, 'biz', 'active'),
 (15, 'u_fixed', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 15, 'biz', 'active'),
 (17, 'u_finance', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 17, 'biz', 'active'),
 (18, 'u_boss', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 18, 'biz', 'active'),
 (19, 'u_planner', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 19, 'biz', 'active'),
 (20, 'u_hr', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 20, 'biz', 'active'),
 (21, 'u_payroll', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 21, 'biz', 'active')
ON CONFLICT DO NOTHING;

UPDATE hr_employee SET user_id = id WHERE id BETWEEN 10 AND 21;

INSERT INTO iam_user_role(user_id, role_id) VALUES
 (10, 4), (11, 10), (12, 5), (13, 7), (14, 8), (15, 9), (17, 13), (18, 2),
 (19, 6), (20, 11), (21, 12)
ON CONFLICT DO NOTHING;

INSERT INTO iam_admin_group_user(group_id, user_id) VALUES
 (3, 10), (4, 11), (4, 12), (4, 13), (5, 14), (5, 15), (6, 17), (2, 18),
 (4, 19), (6, 20), (6, 21)
ON CONFLICT DO NOTHING;

INSERT INTO prd_product(id, code, name, product_type, cost_price, sale_price, status) VALUES
 (1, 'RM-CASSAVA', '鲜木薯', 'raw', 1.2, NULL, 'active'),
 (2, 'SF-COREOUT', '去芯薯肉', 'semi', 2.5, 3.0, 'active'),
 (3, 'FG-DICED', '袋装木薯丁', 'finished', 4.0, 7.0, 'active')
ON CONFLICT DO NOTHING;

-- 过磅品种（与产品绑定；新装 schema 已写入时此处幂等）
INSERT INTO pur_weigh_variety(code, name, sort_no, status, remark)
VALUES
 ('WV-FRESH', '鲜木薯', 10, 'active', '农户鲜薯过磅入厂，入保鲜库'),
 ('WV-SEMI', '半成品（去芯薯肉）', 20, 'active', '外购或厂内半成品过磅入厂，入半成品库'),
 ('WV-FG', '成品入库（袋装木薯丁）', 30, 'active', '成品过磅入库，入成品冷库')
ON CONFLICT (code) DO NOTHING;

UPDATE pur_weigh_variety v
SET default_product_id = p.id, updated_at = NOW()
FROM prd_product p
WHERE v.default_product_id IS NULL AND COALESCE(v.is_deleted,0)=0
  AND (
    (v.code = 'WV-FRESH' AND p.code = 'RM-CASSAVA')
    OR (v.code = 'WV-SEMI' AND p.code = 'SF-COREOUT')
    OR (v.code = 'WV-FG' AND p.code = 'FG-DICED')
  );

-- 农户档案（过磅入厂搜索姓名/手机号）
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

INSERT INTO prd_product_unit(product_id, unit_name, is_base, factor_to_base) VALUES
 (1, 'kg', 1, 1), (2, 'kg', 1, 1), (3, 'kg', 1, 1)
ON CONFLICT DO NOTHING;

INSERT INTO inv_balance(warehouse_id, product_id, batch_no, qty) VALUES
 (1, 1, 'B0801', 42000),
 (2, 2, 'B0803', 6500),
 (3, 3, 'B0802', 3200)
ON CONFLICT DO NOTHING;

INSERT INTO pay_process_wage_rate(process_id, rate, effective_from, status)
SELECT v.process_id, v.rate, v.effective_from, 'active'
FROM (VALUES
  (1::bigint, 0.18::double precision, '2026-07-01'::text),
  (4, 0.25, '2026-07-01'),
  (5, 0.22, '2026-07-01'),
  (10, 0.22, '2026-07-01')
) AS v(process_id, rate, effective_from)
WHERE NOT EXISTS (
  SELECT 1 FROM pay_process_wage_rate r WHERE r.process_id = v.process_id AND r.status = 'active'
);

-- 木薯产线 12 步工艺路线（对齐 pic）
INSERT INTO pd_routing(id, code, name, product_id, version_no, status) VALUES
 (1, 'RT-CASSAVA', '木薯丁产线', 3, 'V1', 'active'),
 (2, 'RT-CASSAVA-RAW', '鲜木薯入厂产线', 1, 'V1', 'active'),
 (3, 'RT-CASSAVA-SEMI', '去芯薯肉入厂产线', 2, 'V1', 'active');

INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (1, 1, 8, 'S1', '入库-原料', 0, 0, 1, 1, 0, 1, 2),
 (1, 2, 7, 'S2', '清洗', 0, 0, 1, 0, 0, 1, 2),
 (1, 3, 1, 'S3', '去皮-计件领料', 1, 0, 1, 0, 1, 1, 2),
 (1, 4, 2, 'S4', '收货-固定工', 0, 1, 1, 0, 0, NULL, 2),
 (1, 5, 3, 'S5', '切断-固定工', 0, 0, 1, 0, 0, NULL, 2),
 (1, 6, 4, 'S6', '去芯-计件', 1, 0, 1, 0, 0, NULL, 2),
 (1, 7, 2, 'S7', '收货-固定工', 0, 1, 1, 0, 0, NULL, 2),
 (1, 8, 9, 'S8', '入库-半成品库', 0, 0, 1, 1, 0, 2, 2),
 (1, 9, 10, 'S9', '出库切块-计件', 1, 0, 1, 0, 1, 2, 2),
 (1, 10, 9, 'S10', '入库-半成品库', 0, 0, 1, 1, 0, 2, 2),
 (1, 11, 6, 'S11', '过滤装袋', 0, 0, 1, 0, 0, NULL, 2),
 (1, 12, 11, 'S12', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 2),
 -- 鲜木薯入厂（routing 2）
 (2, 1, 8, 'S1', '入库-原料', 0, 0, 1, 1, 0, 1, 2),
 (2, 2, 7, 'S2', '清洗', 0, 0, 1, 0, 0, 1, 2),
 (2, 3, 1, 'S3', '去皮-计件领料', 1, 0, 1, 0, 1, 1, 2),
 (2, 4, 2, 'S4', '收货-固定工', 0, 1, 1, 0, 0, NULL, 2),
 (2, 5, 3, 'S5', '切断-固定工', 0, 0, 1, 0, 0, NULL, 2),
 (2, 6, 4, 'S6', '去芯-计件', 1, 0, 1, 0, 0, NULL, 2),
 (2, 7, 2, 'S7', '收货-固定工', 0, 1, 1, 0, 0, NULL, 2),
 (2, 8, 9, 'S8', '入库-半成品库', 0, 0, 1, 1, 0, 2, 2),
 (2, 9, 10, 'S9', '出库切块-计件', 1, 0, 1, 0, 1, 2, 2),
 (2, 10, 9, 'S10', '入库-半成品库', 0, 0, 1, 1, 0, 2, 2),
 (2, 11, 6, 'S11', '过滤装袋', 0, 0, 1, 0, 0, NULL, 2),
 (2, 12, 11, 'S12', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 2),
 -- 半成品入厂（routing 3）
 (3, 1, 9, 'S1', '入库-半成品库', 0, 0, 1, 1, 0, 2, 2),
 (3, 2, 10, 'S2', '出库切块-计件', 1, 0, 1, 0, 1, 2, 2),
 (3, 3, 9, 'S3', '入库-半成品库', 0, 0, 1, 1, 0, 2, 2),
 (3, 4, 6, 'S4', '过滤装袋', 0, 0, 1, 0, 0, NULL, 2),
 (3, 5, 11, 'S5', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 2)
ON CONFLICT DO NOTHING;

INSERT INTO hr_employee(id, emp_no, name, org_id, dept_id, team_id, job_title, emp_type, mobile, badge_code, id_card_no, status) VALUES
 (2, 'E0301', '陈某', 1, 2, 1, '去皮工', 'piece', '13800001002', 'EMP0301', '450103199601011002', 'active'),
 (3, 'E0205', '固定工甲', 1, 2, 2, '收货员', 'fixed', '13800001003', 'EMP0205', '450103199702021003', 'active')
ON CONFLICT DO NOTHING;

INSERT INTO inv_box_code(id, code, product_id, warehouse_id, batch_no, qty, weight, current_process_id, current_step_id, status) VALUES
 (1, 'BX-RAW-DEMO', 1, 1, 'B0801', 1000, 1000, 8, 1, 'open')
ON CONFLICT DO NOTHING;

INSERT INTO pur_supplier(id, code, name, short_name, supplier_type, status, rating, is_preferred, uscc, settle_method, payment_days, lead_time_days, moq, default_warehouse_id, contact_json, remark) VALUES
 (1, 'SUP-RAW-01', '广西木薯原料合作社', '桂薯原料', 'raw', 'qualified', 'A', 1, '91450000MA5XXXXX01', 'monthly', 30, 3, 1000, 1,
  '[{"name":"张经理","mobile":"13800001111","wechat":"zhang_sup","is_primary":true}]', '主原料供应商'),
 (2, 'SUP-AUX-01', '包装袋辅料厂', '包材厂', 'pack', 'qualified', 'B', 0, '91450000MA5XXXXX02', 'cash', 0, 7, 100, 1,
  '[{"name":"李主管","mobile":"13800002222","is_primary":true}]', '包装辅料'),
 (3, 'SUP-POT-01', '待准入产地商', '潜在产地', 'raw', 'potential', 'C', 0, NULL, 'cod', 0, 5, 500, 1,
  '[{"name":"王联系人","mobile":"13800003333","is_primary":true}]', '尚未准入')
ON CONFLICT DO NOTHING;

INSERT INTO pur_supplier_license(id, supplier_id, license_type, license_no, expire_date) VALUES
 (1, 1, '营业执照', '91450000MA5XXXXX01', '2027-12-31'),
 (2, 1, '食品经营许可', 'JY14500000001', '2026-09-30'),
 (3, 2, '营业执照', '91450000MA5XXXXX02', '2028-06-30')
ON CONFLICT DO NOTHING;

INSERT INTO pur_supplier_supply_item(id, supplier_id, product_id, is_preferred, moq, lead_time_days, last_price) VALUES
 (1, 1, 1, 1, 1000, 3, 1.85),
 (2, 2, 2, 1, 100, 7, 0.45),
 (3, 3, 1, 0, 500, 5, 1.90)
ON CONFLICT DO NOTHING;

INSERT INTO schema_meta(key, value) VALUES ('seeded', '1')
ON CONFLICT DO NOTHING;

INSERT INTO hr_employee_department(employee_id, dept_id, is_primary)
SELECT id, dept_id, 1 FROM hr_employee WHERE COALESCE(dept_id, 0) > 0
ON CONFLICT DO NOTHING;

INSERT INTO hr_employee_department(employee_id, dept_id, is_primary) VALUES
 (13, 2, 0)
ON CONFLICT DO NOTHING;

INSERT INTO sys_department_role(dept_id, role_id) VALUES
 (1, 1), (1, 2),
 (3, 6),
 (4, 5),
 (5, 4),
 (7, 10),
 (8, 11),
 (9, 13), (9, 12),
 (2, 7), (2, 8), (2, 9),
 (10, 7), (10, 8), (10, 9)
ON CONFLICT DO NOTHING;

INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, tax_no, status) VALUES
 (1, 'fixed', 0, '6222080000000001', '450103198001011', 'active'),
 (2, 'piece', 0, '6222080000000002', '450103199601011', 'active'),
 (3, 'fixed', 3600, '6222080000000003', '450103199702021', 'active'),
 (10, 'fixed', 4800, '6222080000000010', '450103199104121', 'active'),
 (11, 'fixed', 4600, '6222080000000011', '450103199211091', 'active'),
 (12, 'fixed', 4200, '6222080000000012', '450103199006081', 'active'),
 (13, 'fixed', 6200, '6222080000000013', '450103198703221', 'active'),
 (14, 'piece', 0, '6222080000000014', '450103199505051', 'active'),
 (15, 'fixed', 3800, '6222080000000015', '450103199408181', 'active'),
 (17, 'fixed', 5800, '6222080000000017', '450103198909091', 'active'),
 (18, 'fixed', 12000, '6222080000000018', '450103197501011', 'active'),
 (19, 'fixed', 5500, '6222080000000019', '450103198812011', 'active'),
 (20, 'fixed', 5000, '6222080000000020', '450103199109211', 'active'),
 (21, 'fixed', 5200, '6222080000000021', '450103199307011', 'active')
ON CONFLICT DO NOTHING;

-- Reset sequences after explicit id inserts
SELECT setval(pg_get_serial_sequence('hr_employee', 'id'), COALESCE((SELECT MAX(id) FROM hr_employee), 1));
SELECT setval(pg_get_serial_sequence('iam_admin_group', 'id'), COALESCE((SELECT MAX(id) FROM iam_admin_group), 1));
SELECT setval(pg_get_serial_sequence('iam_login_policy', 'id'), COALESCE((SELECT MAX(id) FROM iam_login_policy), 1));
SELECT setval(pg_get_serial_sequence('iam_role', 'id'), COALESCE((SELECT MAX(id) FROM iam_role), 1));
SELECT setval(pg_get_serial_sequence('iam_user', 'id'), COALESCE((SELECT MAX(id) FROM iam_user), 1));
SELECT setval(pg_get_serial_sequence('inv_box_code', 'id'), COALESCE((SELECT MAX(id) FROM inv_box_code), 1));
SELECT setval(pg_get_serial_sequence('inv_warehouse', 'id'), COALESCE((SELECT MAX(id) FROM inv_warehouse), 1));
SELECT setval(pg_get_serial_sequence('pd_process', 'id'), COALESCE((SELECT MAX(id) FROM pd_process), 1));
SELECT setval(pg_get_serial_sequence('pd_routing', 'id'), COALESCE((SELECT MAX(id) FROM pd_routing), 1));
SELECT setval(pg_get_serial_sequence('pd_work_team', 'id'), COALESCE((SELECT MAX(id) FROM pd_work_team), 1));
SELECT setval(pg_get_serial_sequence('prd_product', 'id'), COALESCE((SELECT MAX(id) FROM prd_product), 1));
SELECT setval(pg_get_serial_sequence('pur_farmer', 'id'), COALESCE((SELECT MAX(id) FROM pur_farmer), 1));
SELECT setval(pg_get_serial_sequence('pur_supplier', 'id'), COALESCE((SELECT MAX(id) FROM pur_supplier), 1));
SELECT setval(pg_get_serial_sequence('pur_supplier_license', 'id'), COALESCE((SELECT MAX(id) FROM pur_supplier_license), 1));
SELECT setval(pg_get_serial_sequence('pur_supplier_supply_item', 'id'), COALESCE((SELECT MAX(id) FROM pur_supplier_supply_item), 1));
SELECT setval(pg_get_serial_sequence('sys_department', 'id'), COALESCE((SELECT MAX(id) FROM sys_department), 1));
SELECT setval(pg_get_serial_sequence('sys_organization', 'id'), COALESCE((SELECT MAX(id) FROM sys_organization), 1));
