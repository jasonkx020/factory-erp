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
				pair("统计报表",
					"生产看板", "生产实况", "三仓库存概览",
					"日经营快照", "原料入场日报", "计件日结汇总",
					"工序扣损收率分析", "收发存明细", "溯源批进度查询", "农户结算对账汇总",
					"薪酬核算对账", "成本期间汇总",
				),
				pair("财务管理", "成本核算", "成本明细溯源表")...,
			),
			edits: append(
				pair("工资管理", "工人信息管理", "工资批量管理", "工序工资", "薪酬核算", "员工工作台账"),
				pair("系统管理", "批量核算工资")...,
			),
		}
	case "purchase":
		return presetGrant{
			edits: pair("采购管理",
				"农户档案", "过磅收货", "过磅流程编排", "过磅品种", "溯源批号", "农户结算", "原料溯源", "来料质检",
			),
			views: pair("库存管理", "库存查询", "出入库记录汇总"),
		}
	case "warehouse":
		return presetGrant{
			edits: pair("库存管理",
				"库存查询", "仓管待入库", "箱码管理", "出入库记录汇总",
				"可用量分析", "亏料预警", "过量预警", "在途量统计", "待用量统计",
			),
			views: append(
				pair("采购管理", "过磅收货"),
				pair("产品管理", "产品档案")...,
			),
		}
	case "planner":
		return presetGrant{
			edits: pair("生产管理",
				"工序定义", "工艺流程", "产线班次", "例外派岗",
				"工序流水", "计件工资", "工序在制", "溯源生产", "工序扣损", "退库未用完还仓",
			),
			views: append(
				pair("库存管理", "库存查询", "可用量分析", "在途量统计", "待用量统计"),
				pair("工资管理", "工序工资")...,
			),
		}
	case "foreman":
		return presetGrant{
			edits: pair("生产管理",
				"例外派岗", "工序流水", "工序在制", "溯源生产", "退库未用完还仓",
			),
			views: append(
				pair("库存管理", "库存查询"),
				pair("工资管理", "工序工资", "员工工作台账")...,
			),
		}
	case "piece":
		return presetGrant{
			edits: pair("生产管理", "工序流水"),
			views: pair("工资管理", "工序工资"),
		}
	case "fixed":
		return presetGrant{
			edits: pair("生产管理", "工序流水"),
			views: pair("生产管理", "溯源生产"),
		}
	case "qc":
		return presetGrant{
			edits: pair("采购管理", "来料质检"),
			views: pair("采购管理", "过磅收货", "原料溯源"),
		}
	case "hr":
		return presetGrant{
			edits: pair("人事管理", "员工档案", "公司架构", "角色管理"),
		}
	case "payroll":
		return presetGrant{
			edits: append(
				pair("工资管理", "工人信息管理", "工资批量管理", "工序工资", "薪酬核算", "员工工作台账"),
				pair("系统管理", "批量核算工资")...,
			),
			views: append(
				pair("财务管理", "成本核算", "成本明细溯源表"),
				pair("统计报表", "薪酬核算对账", "计件日结汇总")...,
			),
		}
	case "finance":
		return presetGrant{
			views: append(
				append(
					pair("财务管理", "成本核算", "成本明细溯源表", "资金管理", "交易流水账", "农户应付"),
					pair("采购管理", "农户结算")...,
				),
				pair("统计报表", "日经营快照", "成本期间汇总", "农户结算对账汇总")...,
			),
			edits: append(
				append(
					pair("财务管理", "成本核算", "资金管理", "交易流水账", "农户应付"),
					pair("采购管理", "农户结算")...,
				),
				pair("工资管理", "工资批量管理", "薪酬核算", "员工工作台账")...,
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
