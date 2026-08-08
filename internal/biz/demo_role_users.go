package biz

import (
	"database/sql"
	"log"
)

// demoPasswordHash is bcrypt(admin123) cost=10 — same as db/sqlite/seed.sql admin.
const demoPasswordHash = "$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0."

type demoRoleUser struct {
	Login, EmpNo, Name, EmpType, RoleCode, Domain string
}

// EnsureDemoRoleUsers creates one login per app workbench role (password: admin123).
// Idempotent; safe to call on every demo boot after EnsureDomainPermissions.
func EnsureDemoRoleUsers(db *sql.DB) {
	if db == nil {
		return
	}
	users := []demoRoleUser{
		{"u_purchase", "E-PUR", "演示采购", "office", "purchase", "采购管理"},
		{"u_qc", "E-QC", "演示质检", "office", "qc", "采购管理"},
		{"u_warehouse", "E-WH", "演示仓管", "warehouse", "warehouse", "库存管理"},
		{"u_foreman", "E-FM", "演示车间主任", "office", "foreman", "生产管理"},
		{"u_piece", "E-PC", "演示计件工", "piece", "piece", "生产管理"},
		{"u_fixed", "E-FX", "演示固定工", "fixed", "fixed", "生产管理"},
		{"u_sales", "E-SL", "演示销售", "sales", "sales", "销售管理"},
		{"u_finance", "E-FN", "演示财务", "office", "finance", "财务管理"},
		{"u_boss", "E-BS", "演示老板", "office", "boss", "统计报表"},
	}
	for _, u := range users {
		ensureOneDemoRoleUser(db, u)
	}
	SeedOpenShiftForToday(db)
	log.Printf("demo role users ensured (password=admin123)")
}

func ensureOneDemoRoleUser(db *sql.DB, u demoRoleUser) {
	_, _ = db.Exec(`INSERT OR IGNORE INTO hr_employee(emp_no, name, org_id, dept_id, workshop_id, emp_type, status)
		VALUES(?,?,1,1,1,?,'active')`, u.EmpNo, u.Name, u.EmpType)

	var empID int64
	_ = db.QueryRow(`SELECT id FROM hr_employee WHERE emp_no=? LIMIT 1`, u.EmpNo).Scan(&empID)
	if empID == 0 {
		return
	}

	_, _ = db.Exec(`INSERT OR IGNORE INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted)
		VALUES(?,?,?,'biz','active',0)`, u.Login, demoPasswordHash, empID)
	// refresh hash/employee if user already existed with placeholder
	_, _ = db.Exec(`UPDATE iam_user SET password_hash=?, employee_id=?, status='active', is_deleted=0 WHERE login_name=?`,
		demoPasswordHash, empID, u.Login)

	var userID, roleID int64
	_ = db.QueryRow(`SELECT id FROM iam_user WHERE login_name=? LIMIT 1`, u.Login).Scan(&userID)
	_ = db.QueryRow(`SELECT id FROM iam_role WHERE code=? LIMIT 1`, u.RoleCode).Scan(&roleID)
	if userID == 0 || roleID == 0 {
		return
	}
	_, _ = db.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, userID, empID)
	// badge for scan station pass smoke / demo
	switch u.RoleCode {
	case "piece":
		_, _ = db.Exec(`UPDATE hr_employee SET badge_code='EMP-PC' WHERE id=?`, empID)
	case "fixed":
		_, _ = db.Exec(`UPDATE hr_employee SET badge_code='EMP-FX' WHERE id=?`, empID)
	}
	_, _ = db.Exec(`INSERT OR IGNORE INTO iam_user_role(user_id, role_id) VALUES(?,?)`, userID, roleID)

	// bind domain permissions (查看+编辑) so API CheckAPIPerm passes
	if u.Domain != "" {
		bindDomainPerms(db, roleID, u.Domain)
		extraDomains := []string{}
		switch u.RoleCode {
		case "piece", "fixed", "foreman":
			extraDomains = []string{"工资管理", "库存管理"}
		case "sales":
			extraDomains = []string{"客户管理", "财务管理"}
		case "purchase", "qc":
			extraDomains = []string{"库存管理", "产品管理"}
		case "warehouse":
			extraDomains = []string{"产品管理", "采购管理"}
		case "finance":
			extraDomains = []string{"销售管理"}
		case "boss":
			extraDomains = []string{"财务管理", "销售管理", "生产管理", "库存管理"}
		}
		for _, d := range extraDomains {
			bindDomainPerms(db, roleID, d)
		}
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
		_, _ = db.Exec(`INSERT OR IGNORE INTO iam_role_permission(role_id, permission_id) VALUES(?,?)`, roleID, pid)
	}
}
