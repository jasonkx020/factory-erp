package biz

import (
	"database/sql"
	"fmt"
)

type factoryDeptSeed struct {
	Code, Name, DeptType, ParentCode string
	RoleCodes                        []string
}

type factoryTeamSeed struct {
	Code, Name, DeptCode string
}

// factoryOrgDepts 加工厂演示架构：一级总经办，二级职能/生产部门，三级车间节点。
var factoryOrgDepts = []factoryDeptSeed{
	{Code: "HQ01", Name: "总经办", DeptType: deptTypeNormal, RoleCodes: []string{"sys_admin", "boss"}},
	{Code: "D-SALES", Name: "销售部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"sales"}},
	{Code: "D-PUR", Name: "采购部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"purchase"}},
	{Code: "D-WH", Name: "仓储部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"warehouse"}},
	{Code: "D-PROD", Name: "生产部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"planner"}},
	{Code: "D-QC", Name: "质检部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"qc"}},
	{Code: "D-HR", Name: "人事行政部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"hr"}},
	{Code: "D-FIN", Name: "财务部", DeptType: deptTypeNormal, ParentCode: "HQ01", RoleCodes: []string{"finance", "payroll"}},
	{Code: "WS01", Name: "一车间", DeptType: deptTypeWorkshop, ParentCode: "D-PROD", RoleCodes: []string{"foreman", "piece", "fixed"}},
	{Code: "WS02", Name: "二车间", DeptType: deptTypeWorkshop, ParentCode: "D-PROD", RoleCodes: []string{"foreman", "piece", "fixed"}},
}

var factoryOrgTeams = []factoryTeamSeed{
	{Code: "T01", Name: "去皮一组", DeptCode: "WS01"},
	{Code: "T02", Name: "切断一组", DeptCode: "WS01"},
	{Code: "T03", Name: "切块一组", DeptCode: "WS02"},
}

func ensureFactoryOrgTree(db *sql.DB) {
	if db == nil {
		return
	}
	ensureDeptRoleTable(db)
	ensureEmployeeDeptTable(db)
	var orgID int64
	_ = db.QueryRow(`SELECT id FROM sys_organization WHERE COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`).Scan(&orgID)
	if orgID == 0 {
		_, _ = db.Exec(`INSERT INTO sys_organization(code, name, status) VALUES('ORG001','桂南木薯加工厂','active')`)
		_ = db.QueryRow(`SELECT id FROM sys_organization WHERE COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`).Scan(&orgID)
	} else {
		_, _ = db.Exec(`UPDATE sys_organization SET name='桂南木薯加工厂' WHERE id=? AND (name='' OR name LIKE '%演示组织%')`, orgID)
	}
	if orgID == 0 {
		orgID = 1
	}
	for _, d := range factoryOrgDepts {
		ensureFactoryDept(db, orgID, d)
	}
	for _, t := range factoryOrgTeams {
		ensureFactoryTeam(db, t)
	}
}

func ensureFactoryDept(db *sql.DB, orgID int64, d factoryDeptSeed) int64 {
	var id int64
	_ = db.QueryRow(`SELECT id FROM sys_department WHERE org_id=? AND code=? AND COALESCE(is_deleted,0)=0`, orgID, d.Code).Scan(&id)
	var parentID int64
	if d.ParentCode != "" {
		_ = db.QueryRow(`SELECT id FROM sys_department WHERE org_id=? AND code=? AND COALESCE(is_deleted,0)=0`, orgID, d.ParentCode).Scan(&parentID)
	}
	if id == 0 {
		res, err := db.Exec(`INSERT INTO sys_department(org_id, parent_id, code, name, dept_type, status) VALUES(?,?,?,?,?,'active')`,
			orgID, nullIf0(parentID), d.Code, d.Name, d.DeptType)
		if err == nil {
			id, _ = res.LastInsertId()
		}
		if id == 0 {
			_ = db.QueryRow(`SELECT id FROM sys_department WHERE org_id=? AND code=? AND COALESCE(is_deleted,0)=0`, orgID, d.Code).Scan(&id)
		}
	} else {
		_, _ = db.Exec(`UPDATE sys_department SET name=?, dept_type=?, parent_id=?, status='active' WHERE id=?`,
			d.Name, d.DeptType, nullIf0(parentID), id)
	}
	if id > 0 {
		for _, code := range d.RoleCodes {
			var rid int64
			_ = db.QueryRow(`SELECT id FROM iam_role WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&rid)
			if rid > 0 {
				_, _ = db.Exec(`INSERT INTO sys_department_role(dept_id, role_id) VALUES(?,?) ON CONFLICT DO NOTHING`, id, rid)
			}
		}
	}
	return id
}

func ensureFactoryTeam(db *sql.DB, t factoryTeamSeed) int64 {
	deptID := deptIDByCode(db, t.DeptCode)
	if deptID == 0 {
		return 0
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM pd_work_team WHERE dept_id=? AND code=? AND COALESCE(is_deleted,0)=0`, deptID, t.Code).Scan(&id)
	if id > 0 {
		_, _ = db.Exec(`UPDATE pd_work_team SET name=?, status='active' WHERE id=?`, t.Name, id)
		return id
	}
	res, err := db.Exec(`INSERT INTO pd_work_team(dept_id, code, name, status) VALUES(?,?,?,'active')`, deptID, t.Code, t.Name)
	if err != nil {
		return 0
	}
	id, _ = res.LastInsertId()
	if id == 0 {
		_ = db.QueryRow(`SELECT id FROM pd_work_team WHERE dept_id=? AND code=? AND COALESCE(is_deleted,0)=0`, deptID, t.Code).Scan(&id)
	}
	return id
}

func deptIDByCode(db *sql.DB, code string) int64 {
	if db == nil || code == "" {
		return 0
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM sys_department WHERE code=? AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, code).Scan(&id)
	return id
}

func teamIDByCode(db *sql.DB, code string) int64 {
	if db == nil || code == "" {
		return 0
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM pd_work_team WHERE code=? AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, code).Scan(&id)
	return id
}

func ensureEmployeeDeptMembership(db *sql.DB, empID, deptID int64, primary bool) {
	if empID <= 0 || deptID <= 0 {
		return
	}
	flag := 0
	if primary {
		flag = 1
	}
	_, _ = db.Exec(`INSERT INTO hr_employee_department(employee_id, dept_id, is_primary) VALUES(?,?,?)
		ON CONFLICT (employee_id, dept_id) DO UPDATE SET is_primary=EXCLUDED.is_primary`, empID, deptID, flag)
}

func ensureWorkerPayProfile(db *sql.DB, empID int64, payType, bank, tax string, monthly float64) {
	if empID <= 0 {
		return
	}
	if payType == "" {
		payType = "piece"
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM pay_worker_profile WHERE employee_id=?`, empID).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, tax_no, status)
			VALUES(?,?,?,?,?,'active')`, empID, payType, monthly, bank, tax)
		return
	}
	_, _ = db.Exec(`UPDATE pay_worker_profile SET pay_type=?, monthly_base=?, bank_account=?, tax_no=?, status='active' WHERE employee_id=?`,
		payType, monthly, bank, tax, empID)
}

func lookupRoleID(db *sql.DB, code string) int64 {
	var id int64
	_ = db.QueryRow(`SELECT id FROM iam_role WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&id)
	return id
}

func fmtBadge(empNo string) string {
	if empNo == "" {
		return ""
	}
	return fmt.Sprintf("EMP-%s", empNo)
}
