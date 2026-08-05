-- 权限种子数据：预置角色 / 管理员分组 / 登录策略 / 字段策略示例 / 权限码样例
USE erp_factory;

-- 组织占位（若未初始化）
INSERT INTO sys_organization (id, code, name, status)
SELECT 1, 'ORG001', '加工厂演示组织', 'active'
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM sys_organization WHERE id = 1);

INSERT INTO iam_login_policy (
  org_id, max_fail_count, lock_minutes, session_ttl_min,
  password_min_len, password_require_letter, password_require_digit,
  password_require_special, password_history, force_change_days, single_session
) SELECT NULL, 5, 30, 120, 8, 1, 1, 0, 5, 90, 0
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM iam_login_policy LIMIT 1);

-- 管理员分组
INSERT INTO iam_admin_group (code, name, remark, sort_no, status) VALUES
('g_platform', '平台管理组', '系统管理员等高权限后台账号', 10, 'active'),
('g_biz', '经营决策组', '老板驾驶舱与经营只读', 20, 'active'),
('g_ops', '业务作业组', '销售/采购等业务内勤与外勤', 30, 'active'),
('g_plant', '仓储生产组', '仓管/计划/车间/质检', 40, 'active'),
('g_line', '一线作业组', '计件工/固定工（前台为主）', 50, 'active'),
('g_func', '职能管理组', '人事/薪资/财务', 60, 'active')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 预置角色（文档 7.3）
INSERT INTO iam_role (code, name, data_scope_type, is_system, status, remark) VALUES
('sys_admin', '系统管理员', 'all', 1, 'active', '系统管理全模块 + 权限分配'),
('boss', '老板', 'all', 1, 'active', '统计报表/财务只读/审批'),
('sales', '销售员', 'self', 1, 'active', '销售+客户'),
('purchase', '采购员', 'all', 1, 'active', '采购+采购类审批'),
('warehouse', '仓管员', 'warehouse', 1, 'active', '库存按仓'),
('planner', '生产计划', 'all', 1, 'active', '任务/派工/工艺/MRP/BOM'),
('foreman', '车间主任', 'workshop', 1, 'active', '工作台/派工/进度/质检'),
('piece', '计件工', 'self', 1, 'active', '报工/领料/本人工资只读'),
('fixed', '固定工', 'team', 1, 'active', '授权工序报工/收货质检'),
('qc', '质检员', 'all', 1, 'active', '来料/入库/过程质检'),
('hr', '人事', 'all', 1, 'active', '人事+考勤审批'),
('payroll', '薪资员', 'all', 1, 'active', '工资+批量核算'),
('finance', '财务', 'all', 1, 'active', '财务+财审管控')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 分组绑定角色
INSERT INTO iam_admin_group_role (group_id, role_id)
SELECT g.id, r.id FROM iam_admin_group g JOIN iam_role r
WHERE (g.code, r.code) IN (
  ('g_platform','sys_admin'),
  ('g_biz','boss'),
  ('g_ops','sales'), ('g_ops','purchase'),
  ('g_plant','warehouse'), ('g_plant','planner'), ('g_plant','foreman'), ('g_plant','qc'),
  ('g_line','piece'), ('g_line','fixed'),
  ('g_func','hr'), ('g_func','payroll'), ('g_func','finance')
)
AND NOT EXISTS (
  SELECT 1 FROM iam_admin_group_role x WHERE x.group_id = g.id AND x.role_id = r.id
);

-- 入职模板
INSERT INTO iam_onboard_role_template (emp_type, role_id)
SELECT t.emp_type, r.id FROM (
  SELECT 'admin' emp_type, 'sys_admin' role_code UNION ALL
  SELECT 'office', 'hr' UNION ALL
  SELECT 'piece', 'piece' UNION ALL
  SELECT 'temp', 'piece' UNION ALL
  SELECT 'fixed', 'fixed'
) t JOIN iam_role r ON r.code = t.role_code
WHERE NOT EXISTS (
  SELECT 1 FROM iam_onboard_role_template x WHERE x.emp_type = t.emp_type AND x.role_id = r.id
);

-- 字段策略：成本隐藏（一线隐藏，财务/老板可见）
INSERT INTO iam_field_policy (role_id, field_key, field_name, visible, editable)
SELECT r.id, f.field_key, f.field_name, f.visible, 0
FROM iam_role r
JOIN (
  SELECT 'cost_price' field_key, '成本价' field_name, 0 visible, 'piece' role_code UNION ALL
  SELECT 'gross_margin', '毛利', 0, 'piece' UNION ALL
  SELECT 'other_wage', '他人工资', 0, 'piece' UNION ALL
  SELECT 'own_wage', '本人工资', 1, 'piece' UNION ALL
  SELECT 'cost_price', '成本价', 0, 'fixed' UNION ALL
  SELECT 'gross_margin', '毛利', 0, 'fixed' UNION ALL
  SELECT 'other_wage', '他人工资', 0, 'fixed' UNION ALL
  SELECT 'cost_price', '成本价', 0, 'foreman' UNION ALL
  SELECT 'gross_margin', '毛利', 0, 'foreman' UNION ALL
  SELECT 'cost_price', '成本价', 1, 'finance' UNION ALL
  SELECT 'gross_margin', '毛利', 1, 'finance' UNION ALL
  SELECT 'other_wage', '他人工资', 1, 'finance' UNION ALL
  SELECT 'cost_price', '成本价', 1, 'boss' UNION ALL
  SELECT 'gross_margin', '毛利', 1, 'boss' UNION ALL
  SELECT 'other_wage', '他人工资', 1, 'boss'
) f ON f.role_code = r.code
WHERE NOT EXISTS (
  SELECT 1 FROM iam_field_policy p WHERE p.role_id = r.id AND p.field_key = f.field_key
);

-- 权限码样例（全量可由程序按 13域×模块×动作 生成；此处植入关键样例）
INSERT INTO iam_permission (code, name, domain, module, action) VALUES
('人事管理:权限分配:查看', '查看权限分配', '人事管理', '权限分配', '查看'),
('人事管理:权限分配:编辑', '编辑权限分配', '人事管理', '权限分配', '编辑'),
('系统管理:自定义权限:查看', '查看自定义权限', '系统管理', '自定义权限', '查看'),
('系统管理:自定义权限:编辑', '编辑自定义权限', '系统管理', '自定义权限', '编辑'),
('系统管理:自定义菜单:编辑', '编辑自定义菜单', '系统管理', '自定义菜单', '编辑'),
('系统管理:登录控制:编辑', '编辑登录控制', '系统管理', '登录控制', '编辑'),
('系统管理:账户冻结:编辑', '账户冻结操作', '系统管理', '账户冻结', '编辑'),
('生产管理:扫码报工:新增', '扫码报工新增', '生产管理', '扫码报工', '新增'),
('生产管理:扫码报工:查看', '扫码报工查看', '生产管理', '扫码报工', '查看'),
('生产管理:联动式领料:新增', '联动领料新增', '生产管理', '联动式领料', '新增'),
('生产管理:生产派工:新增', '生产派工', '生产管理', '生产派工', '新增'),
('生产管理:灵活派发工单:新增', '灵活派发', '生产管理', '灵活派发工单', '新增'),
('生产管理:成本隐藏:编辑', '成本隐藏策略', '生产管理', '成本隐藏', '编辑'),
('库存管理:库存查询:查看', '库存查询', '库存管理', '库存查询', '查看'),
('库存管理:物料调拨耗用:新增', '调拨耗用', '库存管理', '物料调拨耗用', '新增'),
('工资管理:工序工资:查看', '工序工资查看', '工资管理', '工序工资', '查看'),
('工资管理:薪酬核算:编辑', '薪酬核算', '工资管理', '薪酬核算', '编辑'),
('财务管理:成本核算:查看', '成本核算查看', '财务管理', '成本核算', '查看'),
('财务管理:收款核单:编辑', '收款核单', '财务管理', '收款核单', '编辑'),
('审批管理:任务管理:审批', '审批处理', '审批管理', '任务管理', '审批'),
('销售管理:销售订单:新增', '销售订单新增', '销售管理', '销售订单', '新增'),
('采购管理:采购入库:新增', '采购入库', '采购管理', '采购入库', '新增')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 系统管理员拥有全部已植入权限码
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r CROSS JOIN iam_permission p
WHERE r.code = 'sys_admin'
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 计件工：报工/领料查看新增
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r JOIN iam_permission p
WHERE r.code = 'piece' AND p.code IN (
  '生产管理:扫码报工:新增','生产管理:扫码报工:查看','生产管理:联动式领料:新增','工资管理:工序工资:查看'
)
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 车间主任：派工相关
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r JOIN iam_permission p
WHERE r.code = 'foreman' AND p.code IN (
  '生产管理:扫码报工:查看','生产管理:生产派工:新增','生产管理:灵活派发工单:新增','生产管理:联动式领料:新增'
)
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 财务：成本可见相关
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r JOIN iam_permission p
WHERE r.code = 'finance' AND p.code IN (
  '财务管理:成本核算:查看','财务管理:收款核单:编辑','审批管理:任务管理:审批'
)
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 演示管理员账号（密码占位 hash，勿用于生产）
INSERT INTO hr_employee (emp_no, name, org_id, emp_type, status)
SELECT 'E0001', '系统管理员', 1, 'office', 'active'
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM hr_employee WHERE emp_no = 'E0001');

INSERT INTO iam_user (login_name, password_hash, employee_id, user_type, status)
SELECT 'admin', '$2a$10$DEMO_ONLY_HASH_REPLACE_ME___________', e.id, 'admin', 'active'
FROM hr_employee e WHERE e.emp_no = 'E0001'
AND NOT EXISTS (SELECT 1 FROM iam_user WHERE login_name = 'admin');

UPDATE hr_employee e
JOIN iam_user u ON u.employee_id = e.id AND u.login_name = 'admin'
SET e.user_id = u.id;

INSERT INTO iam_user_role (user_id, role_id)
SELECT u.id, r.id FROM iam_user u JOIN iam_role r
WHERE u.login_name = 'admin' AND r.code = 'sys_admin'
AND NOT EXISTS (SELECT 1 FROM iam_user_role x WHERE x.user_id = u.id AND x.role_id = r.id);

INSERT INTO iam_admin_group_user (group_id, user_id)
SELECT g.id, u.id FROM iam_admin_group g JOIN iam_user u
WHERE g.code = 'g_platform' AND u.login_name = 'admin'
AND NOT EXISTS (SELECT 1 FROM iam_admin_group_user x WHERE x.group_id = g.id AND x.user_id = u.id);

-- 菜单裁剪示例：计件工仅可见部分生产/工资/人事模块
INSERT INTO iam_menu_custom (role_id, domain, module, menu_key, visible, sort_no)
SELECT r.id, m.domain, m.module, CONCAT(m.domain, ':', m.module), 1, m.sort_no
FROM iam_role r
JOIN (
  SELECT '生产管理' domain, '扫码报工' module, 10 sort_no UNION ALL
  SELECT '生产管理', '联动式领料', 20 UNION ALL
  SELECT '工资管理', '工序工资', 30 UNION ALL
  SELECT '人事管理', '考勤管理', 40 UNION ALL
  SELECT '人事管理', '请假管理', 50 UNION ALL
  SELECT '系统管理', '公告设置', 60
) m ON 1=1
WHERE r.code = 'piece'
AND NOT EXISTS (
  SELECT 1 FROM iam_menu_custom c WHERE c.role_id = r.id AND c.menu_key = CONCAT(m.domain, ':', m.module)
);
