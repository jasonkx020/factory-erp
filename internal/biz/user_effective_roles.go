package biz

import (
	"database/sql"

	"erp/internal/security"

	"github.com/gin-gonic/gin"
)

func ensureExtraRoleTable(db *sql.DB) {
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS iam_user_extra_role (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, role_id)
)`)
	services := &Services{DB: db}
	services.ensureExtraRoleBackfill()
}

func (s *Services) ensureExtraRoleBackfill() {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user_extra_role`).Scan(&n)
	if n > 0 {
		return
	}
	var roleCnt int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user_role`).Scan(&roleCnt)
	if roleCnt == 0 {
		return
	}
	_, _ = s.DB.Exec(`
INSERT INTO iam_user_extra_role(user_id, role_id)
SELECT ur.user_id, ur.role_id
FROM iam_user_role ur
WHERE NOT EXISTS (
  SELECT 1
  FROM hr_employee e
  JOIN hr_employee_department ed ON ed.employee_id = e.id
  JOIN sys_department_role dr ON dr.dept_id = ed.dept_id
  WHERE COALESCE(e.user_id, 0) = ur.user_id
    AND dr.role_id = ur.role_id
    AND COALESCE(e.is_deleted, 0) = 0
)
ON CONFLICT DO NOTHING`)
	rows, err := s.DB.Query(`SELECT id FROM iam_user WHERE COALESCE(is_deleted,0)=0`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil || uid <= 0 {
			continue
		}
		s.rebuildUserEffectiveRoles(uid)
	}
}

func (s *Services) getExtraRoleIDs(userID int64) ([]int64, error) {
	ensureExtraRoleTable(s.DB)
	rows, err := s.DB.Query(`SELECT role_id FROM iam_user_extra_role WHERE user_id=? ORDER BY role_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt64Rows(rows)
}

func (s *Services) setExtraRoleIDs(userID int64, roleIDs []int64) error {
	ensureExtraRoleTable(s.DB)
	if security.IsFounderUser(s.DB, userID) {
		var sysID int64
		_ = s.DB.QueryRow(`SELECT id FROM iam_role WHERE code='sys_admin' AND COALESCE(is_deleted,0)=0 LIMIT 1`).Scan(&sysID)
		if sysID > 0 {
			roleIDs = unionInt64(roleIDs, []int64{sysID})
		}
	}
	if _, err := s.DB.Exec(`DELETE FROM iam_user_extra_role WHERE user_id=?`, userID); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if rid <= 0 {
			continue
		}
		if _, err := s.DB.Exec(`INSERT INTO iam_user_extra_role(user_id, role_id) VALUES(?,?)`, userID, rid); err != nil {
			return err
		}
	}
	s.rebuildUserEffectiveRoles(userID)
	return nil
}

func (s *Services) getDeptBaseRoleIDsForUser(userID int64) ([]int64, error) {
	deptIDs, err := s.getEmployeeDeptIDsByUser(userID)
	if err != nil {
		return nil, err
	}
	if len(deptIDs) == 0 {
		var deptID int64
		_ = s.DB.QueryRow(`SELECT COALESCE(dept_id,0) FROM hr_employee WHERE user_id=? AND COALESCE(is_deleted,0)=0`, userID).Scan(&deptID)
		if deptID <= 0 {
			var empID int64
			_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, userID).Scan(&empID)
			if empID > 0 {
				_ = s.DB.QueryRow(`SELECT COALESCE(dept_id,0) FROM hr_employee WHERE id=? AND COALESCE(is_deleted,0)=0`, empID).Scan(&deptID)
			}
		}
		if deptID > 0 {
			deptIDs = []int64{deptID}
		}
	}
	seen := map[int64]struct{}{}
	out := []int64{}
	for _, deptID := range deptIDs {
		roleIDs, err := s.getDeptEffectiveRoleIDs(deptID)
		if err != nil {
			return nil, err
		}
		for _, rid := range roleIDs {
			if _, ok := seen[rid]; ok {
				continue
			}
			seen[rid] = struct{}{}
			out = append(out, rid)
		}
	}
	return out, nil
}

func (s *Services) rebuildUserEffectiveRoles(userID int64) {
	if userID <= 0 {
		return
	}
	extra, _ := s.getExtraRoleIDs(userID)
	deptBase, _ := s.getDeptBaseRoleIDsForUser(userID)
	merged := unionInt64(extra, deptBase)
	_, _ = s.DB.Exec(`DELETE FROM iam_user_role WHERE user_id=?`, userID)
	for _, rid := range merged {
		_, _ = s.DB.Exec(`INSERT INTO iam_user_role(user_id, role_id) VALUES(?,?)`, userID, rid)
	}
	s.keepFounderSysAdmin(userID)
	security.InvalidateUserRBAC(userID)
}

func (s *Services) rebuildUserEffectiveRolesByEmployee(empID int64) {
	var userID int64
	if err := s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&userID); err != nil || userID <= 0 {
		return
	}
	s.rebuildUserEffectiveRoles(userID)
}

func (s *Services) loadRoleDetailsByIDs(roleIDs []int64) []gin.H {
	if len(roleIDs) == 0 {
		return nil
	}
	out := make([]gin.H, 0, len(roleIDs))
	for _, rid := range roleIDs {
		var code, name string
		if err := s.DB.QueryRow(`SELECT COALESCE(code,''), COALESCE(name,'') FROM iam_role WHERE id=? AND COALESCE(is_deleted,0)=0`, rid).
			Scan(&code, &name); err != nil {
			continue
		}
		out = append(out, gin.H{"id": rid, "code": code, "name": name})
	}
	return out
}

func unionInt64(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, id := range append(a, b...) {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func scanInt64Rows(rows *sql.Rows) ([]int64, error) {
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func appendExtraRoleIDs(db *sql.DB, userID int64, roleIDs []int64) {
	ensureExtraRoleTable(db)
	for _, rid := range roleIDs {
		if rid <= 0 {
			continue
		}
		_, _ = db.Exec(`INSERT INTO iam_user_extra_role(user_id, role_id) VALUES(?,?) ON CONFLICT DO NOTHING`, userID, rid)
	}
}
