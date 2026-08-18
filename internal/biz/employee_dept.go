package biz

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
)

func ensureEmployeeDeptTable(db *sql.DB) {
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS hr_employee_department (
  employee_id INTEGER NOT NULL,
  dept_id INTEGER NOT NULL,
  is_primary INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (employee_id, dept_id)
)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_hr_employee_department_dept ON hr_employee_department(dept_id)`)
	services := &Services{DB: db}
	services.ensureEmployeeDeptBackfill()
}

func (s *Services) ensureEmployeeDeptBackfill() {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee_department`).Scan(&n)
	if n > 0 {
		return
	}
	_, _ = s.DB.Exec(`
INSERT INTO hr_employee_department(employee_id, dept_id, is_primary)
SELECT id, dept_id, 1 FROM hr_employee
WHERE COALESCE(dept_id, 0) > 0 AND COALESCE(is_deleted, 0) = 0
ON CONFLICT DO NOTHING`)
}

func deptIDsFromBody(body map[string]interface{}) ([]int64, int64) {
	var deptIDs []int64
	if ids, ok := int64SliceFromBody(body, "dept_ids"); ok {
		deptIDs = ids
	}
	primary, hasPrimary := asInt64(body["primary_dept_id"])
	if !hasPrimary || primary <= 0 {
		primary, _ = asInt64(body["dept_id"])
	}
	if len(deptIDs) == 0 && primary > 0 {
		deptIDs = []int64{primary}
	}
	if primary <= 0 && len(deptIDs) > 0 {
		primary = deptIDs[0]
	}
	return deptIDs, primary
}

func (s *Services) getEmployeeDeptIDs(empID int64) ([]int64, error) {
	ensureEmployeeDeptTable(s.DB)
	rows, err := s.DB.Query(`SELECT dept_id FROM hr_employee_department WHERE employee_id=? ORDER BY is_primary DESC, dept_id`, empID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt64Rows(rows)
}

func (s *Services) getEmployeeDeptIDsByUser(userID int64) ([]int64, error) {
	var empID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, userID).Scan(&empID)
	if empID <= 0 {
		_ = s.DB.QueryRow(`SELECT id FROM hr_employee WHERE user_id=? AND COALESCE(is_deleted,0)=0 LIMIT 1`, userID).Scan(&empID)
	}
	if empID <= 0 {
		return nil, nil
	}
	return s.getEmployeeDeptIDs(empID)
}

func (s *Services) loadEmployeeDepartments(empID int64) []gin.H {
	ensureEmployeeDeptTable(s.DB)
	rows, err := s.DB.Query(`SELECT d.id, COALESCE(d.code,''), COALESCE(d.name,''), COALESCE(ed.is_primary,0)
		FROM hr_employee_department ed
		JOIN sys_department d ON d.id=ed.dept_id AND COALESCE(d.is_deleted,0)=0
		WHERE ed.employee_id=?
		ORDER BY ed.is_primary DESC, d.id`, empID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name string
		var isPrimary int
		if err := rows.Scan(&id, &code, &name, &isPrimary); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "code": code, "name": name, "is_primary": isPrimary == 1,
		})
	}
	return out
}

func (s *Services) setEmployeeDepartments(empID int64, deptIDs []int64, primaryDeptID int64) error {
	ensureEmployeeDeptTable(s.DB)
	seen := map[int64]struct{}{}
	unique := []int64{}
	for _, id := range deptIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if primaryDeptID > 0 {
		if _, ok := seen[primaryDeptID]; !ok {
			unique = append([]int64{primaryDeptID}, unique...)
			seen[primaryDeptID] = struct{}{}
		}
	} else if len(unique) > 0 {
		primaryDeptID = unique[0]
	}
	if _, err := s.DB.Exec(`DELETE FROM hr_employee_department WHERE employee_id=?`, empID); err != nil {
		return err
	}
	for _, deptID := range unique {
		isPrimary := 0
		if deptID == primaryDeptID {
			isPrimary = 1
		}
		if _, err := s.DB.Exec(`INSERT INTO hr_employee_department(employee_id, dept_id, is_primary) VALUES(?,?,?)`,
			empID, deptID, isPrimary); err != nil {
			return err
		}
	}
	_, _ = s.DB.Exec(`UPDATE hr_employee SET dept_id=?, updated_at=NOW() WHERE id=?`, nullIf0(primaryDeptID), empID)
	s.rebuildUserEffectiveRolesByEmployee(empID)
	return nil
}

func (s *Services) addEmployeeToDepartment(empID, deptID int64, asPrimary bool) error {
	if empID <= 0 || deptID <= 0 {
		return nil
	}
	ensureEmployeeDeptTable(s.DB)
	isPrimary := 0
	if asPrimary {
		isPrimary = 1
		_, _ = s.DB.Exec(`UPDATE hr_employee_department SET is_primary=0 WHERE employee_id=?`, empID)
		_, _ = s.DB.Exec(`UPDATE hr_employee SET dept_id=?, updated_at=NOW() WHERE id=?`, deptID, empID)
	}
	_, err := s.DB.Exec(`INSERT INTO hr_employee_department(employee_id, dept_id, is_primary) VALUES(?,?,?)
		ON CONFLICT(employee_id, dept_id) DO UPDATE SET is_primary=EXCLUDED.is_primary`, empID, deptID, isPrimary)
	if err != nil {
		return err
	}
	s.rebuildUserEffectiveRolesByEmployee(empID)
	return nil
}

func (s *Services) removeEmployeeFromDepartment(empID, deptID int64) error {
	if empID <= 0 || deptID <= 0 {
		return nil
	}
	ensureEmployeeDeptTable(s.DB)
	var wasPrimary int
	_ = s.DB.QueryRow(`SELECT COALESCE(is_primary,0) FROM hr_employee_department WHERE employee_id=? AND dept_id=?`, empID, deptID).Scan(&wasPrimary)
	if _, err := s.DB.Exec(`DELETE FROM hr_employee_department WHERE employee_id=? AND dept_id=?`, empID, deptID); err != nil {
		return err
	}
	if wasPrimary == 1 {
		var nextDept int64
		_ = s.DB.QueryRow(`SELECT dept_id FROM hr_employee_department WHERE employee_id=? ORDER BY dept_id LIMIT 1`, empID).Scan(&nextDept)
		if nextDept > 0 {
			_, _ = s.DB.Exec(`UPDATE hr_employee_department SET is_primary=1 WHERE employee_id=? AND dept_id=?`, empID, nextDept)
		}
		_, _ = s.DB.Exec(`UPDATE hr_employee SET dept_id=?, updated_at=NOW() WHERE id=?`, nullIf0(nextDept), empID)
	}
	s.rebuildUserEffectiveRolesByEmployee(empID)
	return nil
}

func (s *Services) employeeDeptNames(empID int64) string {
	depts := s.loadEmployeeDepartments(empID)
	if len(depts) == 0 {
		return ""
	}
	names := make([]string, 0, len(depts))
	for _, d := range depts {
		name := fmt.Sprint(d["name"])
		if d["is_primary"] == true {
			name += "（主）"
		}
		names = append(names, name)
	}
	return joinStrings(names, "、")
}

func (s *Services) enrichEmployeeDeptFields(m gin.H) gin.H {
	id, _ := asInt64(m["id"])
	if id <= 0 {
		return m
	}
	depts := s.loadEmployeeDepartments(id)
	deptIDs := make([]int64, 0, len(depts))
	for _, d := range depts {
		if did, ok := asInt64(d["id"]); ok && did > 0 {
			deptIDs = append(deptIDs, did)
		}
	}
	var primary int64
	for _, d := range depts {
		if d["is_primary"] == true {
			primary, _ = asInt64(d["id"])
			break
		}
	}
	if primary <= 0 && len(deptIDs) > 0 {
		primary = deptIDs[0]
	}
	m["departments"] = depts
	m["dept_ids"] = deptIDs
	m["dept_id"] = primary
	m["dept_name"] = s.employeeDeptNames(id)
	return m
}
