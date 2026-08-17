-- factory-erp PostgreSQL data-dev seed

-- 开发种子：组织 / 仓 / 工序 / 角色分组 / admin
-- 默认密码 admin123（bcrypt），仅用于本地开发

INSERT INTO sys_organization(id, code, name, status) VALUES (1, 'ORG001', '加工厂演示组织', 'active');

INSERT INTO sys_department(id, org_id, code, name) VALUES (1, 1, 'D001', '生产部')
ON CONFLICT DO NOTHING;

INSERT INTO pd_workshop(id, org_id, dept_id, code, name) VALUES (1, 1, 1, 'WS01', '一车间')
ON CONFLICT DO NOTHING;

INSERT INTO pd_work_team(id, workshop_id, code, name) VALUES (1, 1, 'T01', '去皮一组')
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
 (3, 'g_ops', '业务作业组', '销售采购', 30),
 (4, 'g_plant', '仓储生产组', '仓管计划车间质检', 40),
 (5, 'g_line', '一线作业组', '计件固定工', 50),
 (6, 'g_func', '职能管理组', '人事薪资财务', 60)
ON CONFLICT DO NOTHING;

INSERT INTO iam_role(id, code, name, data_scope_type, is_system, remark) VALUES
 (1, 'sys_admin', '系统管理员', 'all', 1, '系统管理+权限分配'),
 (2, 'boss', '老板', 'all', 1, '驾驶舱/财务只读'),
 (3, 'sales', '销售员', 'self', 1, '销售客户'),
 (4, 'purchase', '采购员', 'all', 1, '采购'),
 (5, 'warehouse', '仓管员', 'warehouse', 1, '库存按仓'),
 (6, 'planner', '生产计划', 'all', 1, '生产计划'),
 (7, 'foreman', '车间主任', 'workshop', 1, '车间调度'),
 (8, 'piece', '计件工', 'self', 1, '报工领料'),
 (9, 'fixed', '固定工', 'team', 1, '收货质检'),
 (10, 'qc', '质检员', 'all', 1, '质检'),
 (11, 'hr', '人事', 'all', 1, '人事'),
 (12, 'payroll', '薪资员', 'all', 1, '工资'),
 (13, 'finance', '财务', 'all', 1, '财务')
ON CONFLICT DO NOTHING;

INSERT INTO iam_admin_group_role(group_id, role_id) VALUES
 (1,1),(2,2),(3,3),(3,4),(4,5),(4,6),(4,7),(4,10),(5,8),(5,9),(6,11),(6,12),(6,13)
ON CONFLICT DO NOTHING;

INSERT INTO iam_permission(code, name, domain, module, action) VALUES
 ('人事管理:权限分配:查看', '查看权限分配', '人事管理', '权限分配', '查看'),
 ('人事管理:权限分配:编辑', '编辑权限分配', '人事管理', '权限分配', '编辑'),
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
 ('审批管理:任务管理:审批', '审批处理', '审批管理', '任务管理', '审批')
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 1, id FROM iam_permission
ON CONFLICT DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT 8, id FROM iam_permission WHERE code IN (
  '生产管理:扫码报工:新增','生产管理:扫码报工:查看','生产管理:联动式领料:新增','工资管理:工序工资:查看'
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
 (8, '生产管理', '扫码报工', '生产管理:扫码报工', 1, 10),
 (8, '生产管理', '联动式领料', '生产管理:联动式领料', 1, 20),
 (8, '工资管理', '工序工资', '工资管理:工序工资', 1, 30)
ON CONFLICT DO NOTHING;

INSERT INTO hr_employee(id, emp_no, name, org_id, dept_id, workshop_id, emp_type, status)
VALUES (1, 'E0001', '系统管理员', 1, 1, 1, 'office', 'active')
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
INSERT INTO hr_employee(id, emp_no, name, org_id, dept_id, workshop_id, emp_type, status) VALUES
 (10, 'E-PUR', '演示采购', 1, 1, 1, 'office', 'active'),
 (11, 'E-QC', '演示质检', 1, 1, 1, 'office', 'active'),
 (12, 'E-WH', '演示仓管', 1, 1, 1, 'warehouse', 'active'),
 (13, 'E-FM', '演示车间主任', 1, 1, 1, 'office', 'active'),
 (14, 'E-PC', '演示计件工', 1, 1, 1, 'piece', 'active'),
 (15, 'E-FX', '演示固定工', 1, 1, 1, 'fixed', 'active'),
 (16, 'E-SL', '演示销售', 1, 1, NULL, 'sales', 'active'),
 (17, 'E-FN', '演示财务', 1, 1, NULL, 'office', 'active'),
 (18, 'E-BS', '演示老板', 1, 1, NULL, 'office', 'active');

INSERT INTO iam_user(id, login_name, password_hash, employee_id, user_type, status) VALUES
 (10, 'u_purchase', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 10, 'biz', 'active'),
 (11, 'u_qc', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 11, 'biz', 'active'),
 (12, 'u_warehouse', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 12, 'biz', 'active'),
 (13, 'u_foreman', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 13, 'biz', 'active'),
 (14, 'u_piece', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 14, 'biz', 'active'),
 (15, 'u_fixed', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 15, 'biz', 'active'),
 (16, 'u_sales', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 16, 'biz', 'active'),
 (17, 'u_finance', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 17, 'biz', 'active'),
 (18, 'u_boss', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', 18, 'biz', 'active')
ON CONFLICT DO NOTHING;

UPDATE hr_employee SET user_id = id WHERE id BETWEEN 10 AND 18;

INSERT INTO iam_user_role(user_id, role_id) VALUES
 (10, 4), (11, 10), (12, 5), (13, 7), (14, 8), (15, 9), (16, 3), (17, 13), (18, 2)
ON CONFLICT DO NOTHING;

INSERT INTO prd_product(id, code, name, product_type, cost_price, sale_price, status) VALUES
 (1, 'RM-CASSAVA', '鲜木薯', 'raw', 1.2, NULL, 'active'),
 (2, 'SF-COREOUT', '去芯薯肉', 'semi', 2.5, 3.0, 'active'),
 (3, 'FG-DICED', '袋装木薯丁', 'finished', 4.0, 7.0, 'active')
ON CONFLICT DO NOTHING;

INSERT INTO prd_product_unit(product_id, unit_name, is_base, factor_to_base) VALUES
 (1, 'kg', 1, 1), (2, 'kg', 1, 1), (3, 'kg', 1, 1)
ON CONFLICT DO NOTHING;

INSERT INTO inv_balance(warehouse_id, product_id, batch_no, qty) VALUES
 (1, 1, 'B0801', 42000),
 (2, 2, 'B0803', 6500),
 (3, 3, 'B0802', 3200)
ON CONFLICT DO NOTHING;

INSERT INTO pay_process_wage_rate(process_id, rate, effective_from, status) VALUES
 (1, 0.18, '2026-07-01', 'active'),
 (4, 0.25, '2026-07-01', 'active'),
 (5, 0.22, '2026-07-01', 'active'),
 (10, 0.22, '2026-07-01', 'active')
ON CONFLICT DO NOTHING;

-- 木薯产线 12 步工艺路线（对齐 pic）
INSERT INTO pd_routing(id, code, name, product_id, version_no, status) VALUES
 (1, 'RT-CASSAVA', '木薯丁产线', 3, 'V1', 'active'),
 (2, 'RT-CASSAVA-RAW', '鲜木薯入厂产线', 1, 'V1', 'active'),
 (3, 'RT-CASSAVA-SEMI', '去芯薯肉入厂产线', 2, 'V1', 'active');

INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_id) VALUES
 (1, 1, 8, 'S1', '入库-原料', 0, 0, 1, 1, 0, 1, 1),
 (1, 2, 7, 'S2', '清洗', 0, 0, 1, 0, 0, 1, 1),
 (1, 3, 1, 'S3', '去皮-计件领料', 1, 0, 1, 0, 1, 1, 1),
 (1, 4, 2, 'S4', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1),
 (1, 5, 3, 'S5', '切断-固定工', 0, 0, 1, 0, 0, NULL, 1),
 (1, 6, 4, 'S6', '去芯-计件', 1, 0, 1, 0, 0, NULL, 1),
 (1, 7, 2, 'S7', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1),
 (1, 8, 9, 'S8', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (1, 9, 10, 'S9', '出库切块-计件', 1, 0, 1, 0, 1, 2, 1),
 (1, 10, 9, 'S10', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (1, 11, 6, 'S11', '过滤装袋', 0, 0, 1, 0, 0, NULL, 1),
 (1, 12, 11, 'S12', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 1),
 -- 鲜木薯入厂（routing 2）
 (2, 1, 8, 'S1', '入库-原料', 0, 0, 1, 1, 0, 1, 1),
 (2, 2, 7, 'S2', '清洗', 0, 0, 1, 0, 0, 1, 1),
 (2, 3, 1, 'S3', '去皮-计件领料', 1, 0, 1, 0, 1, 1, 1),
 (2, 4, 2, 'S4', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1),
 (2, 5, 3, 'S5', '切断-固定工', 0, 0, 1, 0, 0, NULL, 1),
 (2, 6, 4, 'S6', '去芯-计件', 1, 0, 1, 0, 0, NULL, 1),
 (2, 7, 2, 'S7', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1),
 (2, 8, 9, 'S8', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (2, 9, 10, 'S9', '出库切块-计件', 1, 0, 1, 0, 1, 2, 1),
 (2, 10, 9, 'S10', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (2, 11, 6, 'S11', '过滤装袋', 0, 0, 1, 0, 0, NULL, 1),
 (2, 12, 11, 'S12', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 1),
 -- 半成品入厂（routing 3）
 (3, 1, 9, 'S1', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (3, 2, 10, 'S2', '出库切块-计件', 1, 0, 1, 0, 1, 2, 1),
 (3, 3, 9, 'S3', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1),
 (3, 4, 6, 'S4', '过滤装袋', 0, 0, 1, 0, 0, NULL, 1),
 (3, 5, 11, 'S5', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 1)
ON CONFLICT DO NOTHING;

INSERT INTO hr_employee(id, emp_no, name, org_id, dept_id, workshop_id, emp_type, badge_code, status) VALUES
 (2, 'E0301', '陈某', 1, 1, 1, 'piece', 'EMP0301', 'active'),
 (3, 'E0205', '固定工甲', 1, 1, 1, 'fixed', 'EMP0205', 'active')
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
SELECT setval(pg_get_serial_sequence('pd_workshop', 'id'), COALESCE((SELECT MAX(id) FROM pd_workshop), 1));
SELECT setval(pg_get_serial_sequence('prd_product', 'id'), COALESCE((SELECT MAX(id) FROM prd_product), 1));
SELECT setval(pg_get_serial_sequence('pur_supplier', 'id'), COALESCE((SELECT MAX(id) FROM pur_supplier), 1));
SELECT setval(pg_get_serial_sequence('pur_supplier_license', 'id'), COALESCE((SELECT MAX(id) FROM pur_supplier_license), 1));
SELECT setval(pg_get_serial_sequence('pur_supplier_supply_item', 'id'), COALESCE((SELECT MAX(id) FROM pur_supplier_supply_item), 1));
SELECT setval(pg_get_serial_sequence('sys_department', 'id'), COALESCE((SELECT MAX(id) FROM sys_department), 1));
SELECT setval(pg_get_serial_sequence('sys_organization', 'id'), COALESCE((SELECT MAX(id) FROM sys_organization), 1));
