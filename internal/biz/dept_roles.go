package biz

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
)

func ensureDeptRoleTable(db *sql.DB) {
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS sys_department_role (
  dept_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (dept_id, role_id)
)`)
}

func int64SliceFromBody(body map[string]interface{}, key string) ([]int64, bool) {
	raw, ok := body[key]
	if !ok {
		return nil, false
	}
	switch v := raw.(type) {
	case []interface{}:
		out := make([]int64, 0, len(v))
		for _, x := range v {
			if id, ok := asInt64(x); ok && id > 0 {
				out = append(out, id)
			}
		}
		return out, true
	case []int64:
		out := make([]int64, 0, len(v))
		for _, id := range v {
			if id > 0 {
				out = append(out, id)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func (s *Services) getDeptRoleIDs(deptID int64) ([]int64, error) {
	if deptID <= 0 {
		return nil, nil
	}
	ensureDeptRoleTable(s.DB)
	rows, err := s.DB.Query(`SELECT role_id FROM sys_department_role WHERE dept_id=? ORDER BY role_id`, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var rid int64
		if err := rows.Scan(&rid); err != nil {
			continue
		}
		out = append(out, rid)
	}
	return out, nil
}

func (s *Services) setDeptRoleIDs(deptID int64, roleIDs []int64) error {
	ensureDeptRoleTable(s.DB)
	_, err := s.DB.Exec(`DELETE FROM sys_department_role WHERE dept_id=?`, deptID)
	if err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if rid <= 0 {
			continue
		}
		if _, err := s.DB.Exec(`INSERT INTO sys_department_role(dept_id, role_id) VALUES(?,?)`, deptID, rid); err != nil {
			return err
		}
	}
	return nil
}

func (s *Services) listDeptMembers(deptID int64) ([]gin.H, error) {
	ensureEmployeeDeptTable(s.DB)
	rows, err := s.DB.Query(`SELECT e.id, COALESCE(e.emp_no,''), COALESCE(e.name,''), COALESCE(e.job_title,''),
		COALESCE(e.emp_type,''), COALESCE(e.status,'active'), COALESCE(e.user_id,0), COALESCE(u.login_name,''),
		COALESCE(ed.is_primary,0)
		FROM hr_employee_department ed
		JOIN hr_employee e ON e.id=ed.employee_id
		LEFT JOIN iam_user u ON u.id=e.user_id AND COALESCE(u.is_deleted,0)=0
		WHERE ed.dept_id=? AND COALESCE(e.is_deleted,0)=0 AND COALESCE(e.status,'')<>'left'
		ORDER BY ed.is_primary DESC, e.emp_no, e.id`, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, uid int64
		var no, name, job, typ, status, login string
		var isPrimary int
		if err := rows.Scan(&id, &no, &name, &job, &typ, &status, &uid, &login, &isPrimary); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "emp_no": no, "name": name, "job_title": job, "emp_type": typ,
			"status": status, "user_id": uid, "login_name": login, "has_account": uid > 0,
			"is_primary_dept": isPrimary == 1,
		})
	}
	return out, nil
}

func (s *Services) countDeptMembers(deptID int64) int {
	ensureEmployeeDeptTable(s.DB)
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee_department ed
		JOIN hr_employee e ON e.id=ed.employee_id
		WHERE ed.dept_id=? AND COALESCE(e.is_deleted,0)=0 AND COALESCE(e.status,'')<>'left'`, deptID).Scan(&n)
	return n
}

func (s *Services) applyDeptMembers(deptID int64, employeeIDs []int64) error {
	want := make(map[int64]struct{}, len(employeeIDs))
	for _, eid := range employeeIDs {
		if eid > 0 {
			want[eid] = struct{}{}
		}
	}
	ensureEmployeeDeptTable(s.DB)
	currentRows, err := s.DB.Query(`SELECT employee_id FROM hr_employee_department WHERE dept_id=?`, deptID)
	if err != nil {
		return err
	}
	current := map[int64]struct{}{}
	for currentRows.Next() {
		var eid int64
		if err := currentRows.Scan(&eid); err == nil && eid > 0 {
			current[eid] = struct{}{}
		}
	}
	currentRows.Close()

	for eid := range current {
		if _, ok := want[eid]; ok {
			continue
		}
		if err := s.removeEmployeeFromDepartment(eid, deptID); err != nil {
			return err
		}
	}

	for eid := range want {
		if _, ok := current[eid]; ok {
			continue
		}
		if err := s.addEmployeeToDepartment(eid, deptID, false); err != nil {
			return fmt.Errorf("assign employee %d: %w", eid, err)
		}
	}
	return nil
}

func (s *Services) syncEmployeeDeptRoles(empID, oldDeptID, newDeptID int64) {
	_ = oldDeptID
	_ = newDeptID
	s.rebuildUserEffectiveRolesByEmployee(empID)
}

func (s *Services) syncDeptBaseRolesForAllMembers(deptID int64) {
	if deptID <= 0 {
		return
	}
	ensureEmployeeDeptTable(s.DB)
	rows, err := s.DB.Query(`SELECT ed.employee_id FROM hr_employee_department ed
		JOIN hr_employee e ON e.id=ed.employee_id
		WHERE ed.dept_id=? AND COALESCE(e.user_id,0)>0 AND COALESCE(e.is_deleted,0)=0`, deptID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var empID int64
		if err := rows.Scan(&empID); err != nil || empID <= 0 {
			continue
		}
		s.rebuildUserEffectiveRolesByEmployee(empID)
	}
}

func (s *Services) syncDeptRolesAfterChange(deptID int64, oldRoleIDs, newRoleIDs []int64) {
	_ = oldRoleIDs
	_ = newRoleIDs
	s.syncDeptHierarchyRoleImpact(deptID)
}

func (s *Services) applyDeptBaseRolesForNewAccount(empID int64) {
	s.rebuildUserEffectiveRolesByEmployee(empID)
}

func (s *Services) loadDeptRoleDetails(deptID int64) []gin.H {
	ensureDeptRoleTable(s.DB)
	rows, err := s.DB.Query(`SELECT r.id, COALESCE(r.code,''), COALESCE(r.name,'')
		FROM sys_department_role dr
		JOIN iam_role r ON r.id=dr.role_id
		WHERE dr.dept_id=? AND COALESCE(r.is_deleted,0)=0
		ORDER BY r.id`, deptID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name string
		if err := rows.Scan(&id, &code, &name); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "code": code, "name": name})
	}
	return out
}
