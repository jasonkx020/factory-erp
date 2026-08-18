package biz

import (
	"database/sql"
	"log"
	"strings"
)

// demoPasswordHash is bcrypt(admin123) cost=10 — same as migrations/erp/data-dev.sql admin.
const demoPasswordHash = "$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0."

type demoRoleUser struct {
	Login, EmpNo, Name, EmpType, RoleCode, Domain string
	DeptCode                                      string
	ExtraDeptCodes                                []string
	TeamCode                                      string
	JobTitle, Mobile, IDCard, Badge, Bank, Tax    string
	Monthly                                       float64
	PayType                                       string
	GroupID                                       int64
}

type presetGrant struct {
	views      [][2]string
	edits      [][2]string
	extraCodes []string
}

func pair(domain string, modules ...string) [][2]string {
	out := make([][2]string, 0, len(modules))
	for _, m := range modules {
		out = append(out, [2]string{domain, m})
	}
	return out
}

func presetRoleGrant(roleCode string) presetGrant {
	switch roleCode {
	case "boss":
		return presetGrant{
			views: append(
				append(
					append(pair("统计报表",
						"企业报表", "老板驾驶舱", "生产看板", "生产实况", "客户询价查询", "CRM统计", "日统计报表",
						"毛利润统计", "质检报表", "账目统计", "出入库查询", "收发存明细", "跟进记录查询",
						"销售重量统计", "产品销售查询", "系统物流查询", "成本利润表", "资产负债表", "现金流量表", "利润表",
					), pair("财务管理",
						"账目管理", "交易流水账", "收入支出明细", "订单管理", "小程序管理", "凭证管理", "发票管理",
						"收款核单", "外币结汇", "结汇查询", "分摊撤销", "收款预警", "出纳对账", "预收预付管理",
						"成本核算", "合同利润", "销售认款", "销售退货退单", "往来调整单", "财务审批", "资金管理",
						"财务报表", "成本明细溯源表", "月度结转",
					)...),
					pair("销售管理", "销售订单", "询价管理", "历史报价查询", "订单复购", "数据排行榜")...,
				),
				pair("审批管理", "任务管理", "单据审核", "费用财务审批", "询价财务审批", "询价明细审批", "采购审批", "采购计划单审批", "考勤审批")...,
			),
		}
	case "sales":
		return presetGrant{
			edits: append(
				pair("销售管理",
					"销售订单", "自助下单", "询价管理", "合同管理", "修改订单", "发货审批", "预发货管理", "单据打印",
					"订单复购", "数据排行榜", "销售锁价", "询价审批", "历史报价查询", "销售BOM", "我的订单", "成本预算", "报价计算器", "出厂结算",
				),
				pair("客户管理",
					"CRM客户管理", "商机管理", "客户档案", "客户跟进", "资源分配", "保护机制", "释放机制",
					"询价管理", "导入客户", "线索锁定", "线索隐藏", "任务提醒",
				)...,
			),
		}
	case "purchase":
		return presetGrant{
			edits: append(
				pair("采购管理",
					"供应商管理", "农户档案", "过磅收货", "过磅品种", "溯源批号", "农户结算", "原料溯源", "采购申请",
					"采购计划单", "采购入库", "来料质检", "采购退货", "采购分析", "历史价格查看", "采购任务管理",
				),
				pair("审批管理", "任务管理", "采购审批", "采购计划单审批")...,
			),
			views: pair("库存管理", "库存查询", "出入库记录汇总"),
		}
	case "warehouse":
		return presetGrant{
			edits: append(
				pair("库存管理",
					"库存查询", "仓管待入库", "地磅台账", "亏料预警", "过量预警", "入库质检", "仓库盘点", "车间盘点",
					"仓库盘点记录", "销售退皮", "物料调拨耗用", "商品调价组装拆分", "物料转应付", "在途量统计",
					"待用量统计", "可用量分析", "期初入库", "出入库记录汇总", "采购退货", "箱码管理",
				),
				pair("采购管理", "采购入库", "采购退货", "来料质检", "过磅收货")...,
			),
			views: pair("产品管理", "产品档案"),
		}
	case "planner":
		return presetGrant{
			edits: pair("生产管理",
				"多单整合管理", "生产任务单", "图纸分发", "工序定义", "产线班次", "例外派岗", "灵活派发工单",
				"过站记录", "计件工资", "计件领料表", "自动BOM", "MRP物料分析", "联动式领料", "车间工作台",
				"工序在制", "委外加工", "受托加工生产流程管控", "一单多商品", "进度跟踪", "质检管理", "返修单", "废料管理", "退库未用完还仓",
			),
			views: append(
				pair("库存管理", "库存查询", "可用量分析", "在途量统计", "待用量统计"),
				pair("工资管理", "工序工资")...,
			),
			extraCodes: []string{"生产管理:生产派工:新增"},
		}
	case "foreman":
		return presetGrant{
			edits: pair("生产管理",
				"生产任务单", "例外派岗", "灵活派发工单", "过站记录", "联动式领料", "车间工作台", "工序在制", "进度跟踪", "质检管理", "返修单",
			),
			views: append(
				pair("库存管理", "库存查询"),
				pair("工资管理", "工序工资", "员工工作台账")...,
			),
			extraCodes: []string{"生产管理:扫码报工:新增", "生产管理:扫码报工:查看", "生产管理:联动式领料:新增", "生产管理:生产派工:新增"},
		}
	case "piece":
		return presetGrant{
			edits:      pair("生产管理", "过站记录", "联动式领料"),
			views:      pair("工资管理", "工序工资"),
			extraCodes: []string{"生产管理:扫码报工:新增", "生产管理:扫码报工:查看", "生产管理:联动式领料:新增"},
		}
	case "fixed":
		return presetGrant{
			edits:      pair("生产管理", "过站记录"),
			views:      pair("生产管理", "质检管理"),
			extraCodes: []string{"生产管理:扫码报工:新增", "生产管理:扫码报工:查看"},
		}
	case "qc":
		return presetGrant{
			edits: append(
				pair("采购管理", "来料质检"),
				pair("库存管理", "入库质检")...,
			),
			views: append(
				pair("生产管理", "质检管理"),
				pair("统计报表", "质检报表")...,
			),
		}
	case "hr":
		return presetGrant{
			edits: pair("人事管理",
				"员工档案", "公司架构", "角色管理", "入职登记", "离职登记", "人事调动", "工具领还",
				"考勤管理", "班次管理", "绩效管理", "请假管理", "考勤明细", "加班补卡统计", "考勤月度统计",
				"考勤绩效汇总", "外访明细", "备忘录管理", "员工日志",
			),
			views: pair("审批管理", "任务管理", "考勤审批"),
		}
	case "payroll":
		return presetGrant{
			edits: append(
				pair("工资管理", "工人信息管理", "工资批量管理", "工序工资", "薪酬核算", "员工工作台账", "销售提成"),
				pair("系统管理", "批量核算工资")...,
			),
		}
	case "finance":
		return presetGrant{
			edits: append(
				pair("财务管理",
					"账目管理", "交易流水账", "收入支出明细", "订单管理", "小程序管理", "凭证管理", "发票管理", "收款核单",
					"外币结汇", "结汇查询", "分摊撤销", "收款预警", "出纳对账", "预收预付管理", "成本核算", "合同利润",
					"销售认款", "销售退货退单", "往来调整单", "财务审批", "资金管理", "财务报表", "成本明细溯源表", "月度结转",
				),
				pair("审批管理", "任务管理", "费用财务审批", "询价财务审批", "询价明细审批")...,
			),
			views:      pair("工资管理", "工资批量管理", "薪酬核算", "员工工作台账"),
			extraCodes: []string{"审批管理:任务管理:审批"},
		}
	case "customer":
		return presetGrant{
			edits: pair("销售管理", "询价管理", "自助下单", "我的订单", "订单复购", "报价计算器"),
			views: append(
				pair("销售管理", "历史报价查询", "销售锁价", "合同管理", "发货审批", "预发货管理"),
				pair("产品管理", "产品档案")...,
			),
		}
	default:
		return presetGrant{}
	}
}

func demoRoleUserSeeds() []demoRoleUser {
	return []demoRoleUser{
		{"admin", "E0001", "系统管理员", "office", "sys_admin", "系统管理",
			"HQ01", nil, "", "系统管理员", "13800001001", "450103198001011011", "EMP-ADMIN", "6222080000000001", "450103198001011", 0, "fixed", 1},
		{"u_boss", "E-BS", "韦建国", "office", "boss", "统计报表",
			"HQ01", nil, "", "总经理", "13800001018", "450103197501011018", "EMP-BS", "6222080000000018", "450103197501011", 12000, "fixed", 2},
		{"u_sales", "E-SL", "周海燕", "sales", "sales", "销售管理",
			"D-SALES", nil, "", "销售员", "13800001016", "450103199203151016", "EMP-SL", "6222080000000016", "450103199203151", 4500, "fixed", 3},
		{"u_purchase", "E-PUR", "李采购", "office", "purchase", "采购管理",
			"D-PUR", nil, "", "采购员", "13800001010", "450103199104121010", "EMP-PUR", "6222080000000010", "450103199104121", 4800, "fixed", 3},
		{"u_warehouse", "E-WH", "黄仓管", "warehouse", "warehouse", "库存管理",
			"D-WH", nil, "", "仓管员", "13800001012", "450103199006081012", "EMP-WH", "6222080000000012", "450103199006081", 4200, "fixed", 4},
		{"u_planner", "E-PL", "吴计划", "office", "planner", "生产管理",
			"D-PROD", nil, "", "生产计划员", "13800001019", "450103198812011019", "EMP-PL", "6222080000000019", "450103198812011", 5500, "fixed", 4},
		{"u_foreman", "E-FM", "赵主任", "office", "foreman", "生产管理",
			"D-PROD", []string{"WS01"}, "", "车间主任", "13800001013", "450103198703221013", "EMP-FM", "6222080000000013", "450103198703221", 6200, "fixed", 4},
		{"u_piece", "E-PC", "陈计件", "piece", "piece", "生产管理",
			"WS01", nil, "T01", "去皮计件工", "13800001014", "450103199505051014", "EMP-PC", "6222080000000014", "450103199505051", 0, "piece", 5},
		{"u_fixed", "E-FX", "刘固定", "fixed", "fixed", "生产管理",
			"WS01", nil, "T02", "收货固定工", "13800001015", "450103199408181015", "EMP-FX", "6222080000000015", "450103199408181", 3800, "fixed", 5},
		{"u_qc", "E-QC", "孙质检", "office", "qc", "采购管理",
			"D-QC", nil, "", "质检员", "13800001011", "450103199211091011", "EMP-QC", "6222080000000011", "450103199211091", 4600, "fixed", 4},
		{"u_hr", "E-HR", "郑人事", "office", "hr", "人事管理",
			"D-HR", nil, "", "人事专员", "13800001020", "450103199109211020", "EMP-HR", "6222080000000020", "450103199109211", 5000, "fixed", 6},
		{"u_payroll", "E-PAY", "冯薪资", "office", "payroll", "工资管理",
			"D-FIN", nil, "", "薪资员", "13800001021", "450103199307011021", "EMP-PAY", "6222080000000021", "450103199307011", 5200, "fixed", 6},
		{"u_finance", "E-FN", "钱会计", "office", "finance", "财务管理",
			"D-FIN", nil, "", "会计", "13800001017", "450103198909091017", "EMP-FN", "6222080000000017", "450103198909091", 5800, "fixed", 6},
	}
}

// EnsureDemoRoleUsers creates one login per app workbench role (password: admin123).
// Idempotent; safe to call on every demo boot after EnsureDomainPermissions.
func EnsureDemoRoleUsers(db *sql.DB) {
	if db == nil {
		return
	}
	ensureFactoryOrgTree(db)
	ensureLineWorkerSeeds(db)
	for _, u := range demoRoleUserSeeds() {
		ensureOneDemoRoleUser(db, u)
	}
	ensureDemoCustomerPortal(db)
	SeedOpenShiftForToday(db)
	log.Printf("demo role users ensured (password=admin123)")
}

func ensureLineWorkerSeeds(db *sql.DB) {
	ws := deptIDByCode(db, "WS01")
	team := teamIDByCode(db, "T01")
	ensureLineWorker(db, "E0301", "陈某", "piece", "去皮工", "13800001002", "450103199601011002", "EMP0301", ws, team, "piece", 0, "6222080000000002", "450103199601011")
	ensureLineWorker(db, "E0205", "固定工甲", "fixed", "收货员", "13800001003", "450103199702021003", "EMP0205", ws, teamIDByCode(db, "T02"), "fixed", 3600, "6222080000000003", "450103199702021")
}

func ensureLineWorker(db *sql.DB, empNo, name, empType, job, mobile, idCard, badge string, deptID, teamID int64, payType string, monthly float64, bank, tax string) {
	_, _ = db.Exec(`INSERT INTO hr_employee(emp_no, name, org_id, dept_id, team_id, job_title, emp_type, mobile, badge_code, id_card_no, status)
		VALUES(?,?,1,?,?,?,?,?,?,?,'active')
		ON CONFLICT (emp_no) DO UPDATE SET
			name=EXCLUDED.name, dept_id=EXCLUDED.dept_id, team_id=EXCLUDED.team_id, job_title=EXCLUDED.job_title,
			emp_type=EXCLUDED.emp_type, mobile=EXCLUDED.mobile, badge_code=EXCLUDED.badge_code, id_card_no=EXCLUDED.id_card_no, status='active'`,
		empNo, name, nullIf0(deptID), nullIf0(teamID), job, empType, mobile, badge, idCard)
	var empID int64
	_ = db.QueryRow(`SELECT id FROM hr_employee WHERE emp_no=? LIMIT 1`, empNo).Scan(&empID)
	if empID == 0 {
		return
	}
	ensureEmployeeDeptMembership(db, empID, deptID, true)
	ensureWorkerPayProfile(db, empID, payType, bank, tax, monthly)
}

func bindPresetRolePerms(db *sql.DB, roleID int64, roleCode string) {
	if db == nil || roleID <= 0 || roleCode == "" {
		return
	}
	if roleCode == "sys_admin" {
		return
	}
	grant := presetRoleGrant(roleCode)
	codes := make([]string, 0, 256)
	seen := map[string]struct{}{}
	add := func(code string) {
		if strings.TrimSpace(code) == "" {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	for _, dm := range grant.views {
		for _, code := range modulePermCodes(dm[0], dm[1], false) {
			add(code)
		}
	}
	for _, dm := range grant.edits {
		for _, code := range modulePermCodes(dm[0], dm[1], false) {
			add(code)
		}
		for _, code := range modulePermCodes(dm[0], dm[1], true) {
			add(code)
		}
	}
	for _, code := range grant.extraCodes {
		add(code)
	}
	_, _ = db.Exec(`DELETE FROM iam_role_permission WHERE role_id=?`, roleID)
	for _, code := range codes {
		var pid int64
		if err := db.QueryRow(`SELECT id FROM iam_permission WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&pid); err == nil && pid > 0 {
			_, _ = db.Exec(`INSERT INTO iam_role_permission(role_id, permission_id) VALUES(?,?)
ON CONFLICT (role_id, permission_id) DO NOTHING`, roleID, pid)
		}
	}
}

func ensureOneDemoRoleUser(db *sql.DB, u demoRoleUser) {
	deptID := deptIDByCode(db, u.DeptCode)
	teamID := teamIDByCode(db, u.TeamCode)
	badge := u.Badge
	if badge == "" {
		badge = fmtBadge(u.EmpNo)
	}
	_, _ = db.Exec(`INSERT INTO hr_employee(emp_no, name, org_id, dept_id, team_id, job_title, emp_type, mobile, badge_code, id_card_no, status)
		VALUES(?,?,1,?,?,?,?,?,?,?,'active')
		ON CONFLICT (emp_no) DO UPDATE SET
			name=EXCLUDED.name, dept_id=EXCLUDED.dept_id, team_id=EXCLUDED.team_id, job_title=EXCLUDED.job_title,
			emp_type=EXCLUDED.emp_type, mobile=EXCLUDED.mobile, badge_code=EXCLUDED.badge_code, id_card_no=EXCLUDED.id_card_no, status='active'`,
		u.EmpNo, u.Name, nullIf0(deptID), nullIf0(teamID), u.JobTitle, u.EmpType, u.Mobile, badge, u.IDCard)

	var empID int64
	_ = db.QueryRow(`SELECT id FROM hr_employee WHERE emp_no=? LIMIT 1`, u.EmpNo).Scan(&empID)
	if empID == 0 {
		return
	}
	ensureEmployeeDeptMembership(db, empID, deptID, true)
	for _, extra := range u.ExtraDeptCodes {
		ensureEmployeeDeptMembership(db, empID, deptIDByCode(db, extra), false)
	}
	ensureWorkerPayProfile(db, empID, u.PayType, u.Bank, u.Tax, u.Monthly)

	_, _ = db.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted)
		VALUES(?,?,?,'biz','active',0)
		ON CONFLICT (login_name) DO NOTHING`, u.Login, demoPasswordHash, empID)
	userType := "biz"
	if u.RoleCode == "sys_admin" {
		userType = "admin"
	}
	_, _ = db.Exec(`UPDATE iam_user SET password_hash=?, employee_id=?, user_type=?, status='active', is_deleted=0 WHERE login_name=?`,
		demoPasswordHash, empID, userType, u.Login)

	var userID int64
	roleID := lookupRoleID(db, u.RoleCode)
	_ = db.QueryRow(`SELECT id FROM iam_user WHERE login_name=? LIMIT 1`, u.Login).Scan(&userID)
	if userID == 0 {
		return
	}
	_, _ = db.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, userID, empID)
	if roleID > 0 {
		_, _ = db.Exec(`INSERT INTO iam_user_role(user_id, role_id) VALUES(?,?)
ON CONFLICT (user_id, role_id) DO NOTHING`, userID, roleID)
	}
	if u.GroupID > 0 {
		_, _ = db.Exec(`INSERT INTO iam_admin_group_user(group_id, user_id) VALUES(?,?)
ON CONFLICT (group_id, user_id) DO NOTHING`, u.GroupID, userID)
	}

	if roleID > 0 {
		bindPresetRolePerms(db, roleID, u.RoleCode)
	}
}

func bindDomainPerms(db *sql.DB, roleID int64, domain string) {
	rows, err := db.Query(`SELECT id FROM iam_permission WHERE domain=? AND COALESCE(is_deleted,0)=0`, domain)
	if err != nil {
		return
	}
	ids := make([]int64, 0, 64)
	for rows.Next() {
		var pid int64
		if rows.Scan(&pid) == nil && pid > 0 {
			ids = append(ids, pid)
		}
	}
	_ = rows.Close()
	for _, pid := range ids {
		_, _ = db.Exec(`INSERT INTO iam_role_permission(role_id, permission_id) VALUES(?,?)
ON CONFLICT (role_id, permission_id) DO NOTHING`, roleID, pid)
	}
}

// ensureDemoCustomerPortal creates Portal demo login cust01 bound to a CRM customer.
func ensureDemoCustomerPortal(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`INSERT INTO iam_role(code, name, data_scope_type, is_system, remark, status)
		VALUES('customer','客户自助','self',1,'Portal 客户自助账号','active')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, remark=EXCLUDED.remark, status='active', is_deleted=0`)

	var customerID int64
	_ = db.QueryRow(`SELECT id FROM crm_customer WHERE COALESCE(is_deleted,0)=0 AND code='CU-DEMO-11' LIMIT 1`).Scan(&customerID)
	if customerID == 0 {
		_ = db.QueryRow(`SELECT id FROM crm_customer WHERE COALESCE(is_deleted,0)=0 AND code='CU-PORTAL-01' LIMIT 1`).Scan(&customerID)
	}
	if customerID == 0 {
		_ = db.QueryRow(`SELECT id FROM crm_customer WHERE COALESCE(is_deleted,0)=0 AND status='active' ORDER BY id LIMIT 1`).Scan(&customerID)
	}
	if customerID == 0 {
		_, _ = db.Exec(`INSERT INTO crm_customer(code, name, short_name, contact_name, mobile, address, level, source, status, is_public_sea, is_locked, is_hidden, settle_method, payment_days, credit_limit, remark)
			VALUES('CU-PORTAL-01','门户演示客户','门户客户','王采购','13900001901','南宁','A','门户','active',0,0,0,'月结',30,50000,'客户自助演示')
			ON CONFLICT (code) DO NOTHING`)
		_ = db.QueryRow(`SELECT id FROM crm_customer WHERE code='CU-PORTAL-01' LIMIT 1`).Scan(&customerID)
	}
	if customerID == 0 {
		return
	}

	_, _ = db.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, customer_id, user_type, status, is_deleted)
		VALUES('cust01',?,NULL,?,'customer','active',0)
		ON CONFLICT (login_name) DO UPDATE SET
			password_hash=EXCLUDED.password_hash, customer_id=EXCLUDED.customer_id, user_type='customer',
			status='active', is_deleted=0, employee_id=NULL`, demoPasswordHash, customerID)

	var userID, roleID int64
	_ = db.QueryRow(`SELECT id FROM iam_user WHERE login_name='cust01' LIMIT 1`).Scan(&userID)
	roleID = lookupRoleID(db, "customer")
	if userID == 0 || roleID == 0 {
		return
	}
	_, _ = db.Exec(`INSERT INTO iam_user_role(user_id, role_id) VALUES(?,?)
ON CONFLICT (user_id, role_id) DO NOTHING`, userID, roleID)
	bindPresetRolePerms(db, roleID, "customer")
}
