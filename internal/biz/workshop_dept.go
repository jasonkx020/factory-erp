package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	deptTypeNormal   = "normal"
	deptTypeWorkshop = "workshop"
)

func normalizeDeptType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case deptTypeWorkshop, "ws", "车间":
		return deptTypeWorkshop
	default:
		return deptTypeNormal
	}
}

func (s *Services) getDeptType(deptID int64) string {
	if deptID <= 0 {
		return ""
	}
	var t string
	_ = s.DB.QueryRow(`SELECT COALESCE(dept_type, 'normal') FROM sys_department WHERE id=? AND COALESCE(is_deleted,0)=0`, deptID).Scan(&t)
	if t == "" {
		return deptTypeNormal
	}
	return t
}

func defaultWorkshopDeptIDDB(db *sql.DB) int64 {
	if db == nil {
		return 0
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM sys_department WHERE COALESCE(dept_type,'normal')=? AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, deptTypeWorkshop).Scan(&id)
	return id
}

func (s *Services) defaultWorkshopDeptID() int64 {
	return defaultWorkshopDeptIDDB(s.DB)
}

func (s *Services) workshopDeptIDFromBody(body map[string]interface{}) int64 {
	id, _ := asInt64(body["workshop_dept_id"])
	return id
}

func (s *Services) resolveWorkshopDeptID(body map[string]interface{}, fallbackDefault bool) int64 {
	id := s.workshopDeptIDFromBody(body)
	if id > 0 {
		if s.getDeptType(id) != deptTypeWorkshop {
			return 0
		}
		return id
	}
	if fallbackDefault {
		return s.defaultWorkshopDeptID()
	}
	return 0
}

func (s *Services) employeeWorkshopDeptIDs(empID int64) []int64 {
	if empID <= 0 {
		return nil
	}
	rows, err := s.DB.Query(`SELECT ed.dept_id FROM hr_employee_department ed
		JOIN sys_department d ON d.id=ed.dept_id AND COALESCE(d.is_deleted,0)=0
		WHERE ed.employee_id=? AND COALESCE(d.dept_type,'normal')=?
		ORDER BY ed.is_primary DESC, ed.dept_id`, empID, deptTypeWorkshop)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out, _ := scanInt64Rows(rows)
	return out
}

func (s *Services) employeePrimaryWorkshopDeptID(empID int64) int64 {
	ids := s.employeeWorkshopDeptIDs(empID)
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func (s *Services) validateDeptTypeParent(deptID, parentID int64, deptType string) error {
	deptType = normalizeDeptType(deptType)
	if deptType == deptTypeWorkshop {
		if parentID <= 0 {
			return fmt.Errorf("WORKSHOP_PARENT_REQUIRED")
		}
		if s.getDeptType(parentID) != deptTypeNormal {
			return fmt.Errorf("WORKSHOP_PARENT_MUST_BE_NORMAL")
		}
	}
	if parentID > 0 && s.getDeptType(parentID) == deptTypeWorkshop {
		return fmt.Errorf("WORKSHOP_NO_CHILDREN")
	}
	if deptID > 0 && deptType == deptTypeWorkshop {
		var childCnt int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sys_department WHERE parent_id=? AND COALESCE(is_deleted,0)=0`, deptID).Scan(&childCnt)
		if childCnt > 0 {
			return fmt.Errorf("WORKSHOP_NO_CHILDREN")
		}
	}
	return nil
}

func (s *Services) listWorkTeamsByDept(deptID int64) []gin.H {
	q := `SELECT id, COALESCE(dept_id,0), COALESCE(code,''), COALESCE(name,''), COALESCE(status,'active')
		FROM pd_work_team WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	if deptID > 0 {
		q += ` AND dept_id=?`
		args = append(args, deptID)
	}
	q += ` ORDER BY id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, did int64
		var code, name, status string
		if err := rows.Scan(&id, &did, &code, &name, &status); err != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "dept_id": did, "code": code, "name": name, "status": status})
	}
	return out
}

func (s *Services) listTeamMemberIDs(teamID int64) []int64 {
	if teamID <= 0 {
		return nil
	}
	rows, err := s.DB.Query(`SELECT id FROM hr_employee
		WHERE team_id=? AND COALESCE(is_deleted,0)=0 AND COALESCE(status,'')<>'left'
		ORDER BY emp_no, id`, teamID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanInt64RowsMust(rows)
}

func scanInt64RowsMust(rows *sql.Rows) []int64 {
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func (s *Services) listWorkTeamsWithMembers(deptID int64) []gin.H {
	teams := s.listWorkTeamsByDept(deptID)
	for i, t := range teams {
		tid, _ := asInt64(t["id"])
		ids := s.listTeamMemberIDs(tid)
		t["employee_ids"] = ids
		teams[i] = t
	}
	return teams
}

type teamMemberBatch struct {
	TeamID      int64
	TeamCode    string
	EmployeeIDs []int64
}

func parseTeamMembersBody(body map[string]interface{}) []teamMemberBatch {
	raw, ok := body["team_members"].([]interface{})
	if !ok {
		return nil
	}
	out := []teamMemberBatch{}
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tid, _ := asInt64(m["team_id"])
		code := strings.TrimSpace(strOr(m["team_code"]))
		empIDs, _ := int64SliceFromBody(m, "employee_ids")
		if tid <= 0 && code == "" {
			continue
		}
		out = append(out, teamMemberBatch{TeamID: tid, TeamCode: code, EmployeeIDs: empIDs})
	}
	return out
}

func (s *Services) applyWorkshopTeamMembers(deptID int64, assigns []teamMemberBatch) error {
	if deptID <= 0 || len(assigns) == 0 {
		return nil
	}
	teams := s.listWorkTeamsByDept(deptID)
	teamByID := map[int64]struct{}{}
	teamByCode := map[string]int64{}
	for _, t := range teams {
		tid, _ := asInt64(t["id"])
		if tid <= 0 {
			continue
		}
		teamByID[tid] = struct{}{}
		if code := strings.TrimSpace(strOr(t["code"])); code != "" {
			teamByCode[code] = tid
		}
	}
	desired := map[int64]int64{}
	for _, a := range assigns {
		tid := a.TeamID
		if tid <= 0 && a.TeamCode != "" {
			tid = teamByCode[a.TeamCode]
		}
		if tid <= 0 {
			continue
		}
		if _, ok := teamByID[tid]; !ok {
			return fmt.Errorf("TEAM_NOT_IN_WORKSHOP")
		}
		for _, eid := range a.EmployeeIDs {
			if eid > 0 {
				desired[eid] = tid
			}
		}
	}
	memberSet := map[int64]struct{}{}
	for _, eid := range s.listDeptMemberIDs(deptID) {
		memberSet[eid] = struct{}{}
	}
	touch := map[int64]struct{}{}
	for eid := range memberSet {
		touch[eid] = struct{}{}
	}
	teamIDList := make([]int64, 0, len(teamByID))
	for tid := range teamByID {
		teamIDList = append(teamIDList, tid)
	}
	if len(teamIDList) > 0 {
		placeholders := strings.Repeat("?,", len(teamIDList))
		placeholders = placeholders[:len(placeholders)-1]
		q := fmt.Sprintf(`SELECT id FROM hr_employee WHERE COALESCE(is_deleted,0)=0 AND team_id IN (%s)`, placeholders)
		args := make([]interface{}, len(teamIDList))
		for i, tid := range teamIDList {
			args[i] = tid
		}
		rows, err := s.DB.Query(q, args...)
		if err == nil {
			for rows.Next() {
				var eid int64
				if rows.Scan(&eid) == nil && eid > 0 {
					touch[eid] = struct{}{}
				}
			}
			rows.Close()
		}
	}
	for eid := range touch {
		newTeam := desired[eid]
		if newTeam > 0 {
			if _, ok := memberSet[eid]; !ok {
				continue
			}
		}
		var cur int64
		_ = s.DB.QueryRow(`SELECT COALESCE(team_id,0) FROM hr_employee WHERE id=? AND COALESCE(is_deleted,0)=0`, eid).Scan(&cur)
		if newTeam == 0 {
			if _, ok := teamByID[cur]; !ok {
				continue
			}
		}
		if cur == newTeam {
			continue
		}
		if _, err := s.DB.Exec(`UPDATE hr_employee SET team_id=?, updated_at=NOW() WHERE id=?`, nullIf0(newTeam), eid); err != nil {
			return err
		}
	}
	return nil
}

func (s *Services) listDeptMemberIDs(deptID int64) []int64 {
	ensureEmployeeDeptTable(s.DB)
	rows, err := s.DB.Query(`SELECT ed.employee_id FROM hr_employee_department ed
		JOIN hr_employee e ON e.id=ed.employee_id
		WHERE ed.dept_id=? AND COALESCE(e.is_deleted,0)=0 AND COALESCE(e.status,'')<>'left'`, deptID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanInt64RowsMust(rows)
}

func (s *Services) syncDeptTeams(deptID int64, raw []interface{}) error {
	if deptID <= 0 {
		return nil
	}
	keep := map[int64]struct{}{}
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		code := strings.TrimSpace(strOr(m["code"]))
		name := strings.TrimSpace(strOr(m["name"]))
		if name == "" {
			continue
		}
		id, _ := asInt64(m["id"])
		status := strOrDef(m["status"], "active")
		if code == "" {
			code = fmt.Sprintf("T%d", id)
			if id == 0 {
				code = fmt.Sprintf("T%d", time.Now().UnixNano()%1e8)
			}
		}
		if id > 0 {
			_, err := s.DB.Exec(`UPDATE pd_work_team SET code=?, name=?, status=?, dept_id=?, is_deleted=0 WHERE id=?`,
				code, name, status, deptID, id)
			if err != nil {
				return err
			}
			keep[id] = struct{}{}
			continue
		}
		res, err := s.DB.Exec(`INSERT INTO pd_work_team(dept_id, code, name, status) VALUES(?,?,?,?)`, deptID, code, name, status)
		if err != nil {
			return err
		}
		nid, _ := res.LastInsertId()
		if nid > 0 {
			keep[nid] = struct{}{}
		}
	}
	rows, err := s.DB.Query(`SELECT id FROM pd_work_team WHERE dept_id=? AND COALESCE(is_deleted,0)=0`, deptID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if _, ok := keep[id]; !ok {
			_, _ = s.DB.Exec(`UPDATE pd_work_team SET is_deleted=1, status='inactive' WHERE id=?`, id)
		}
	}
	return nil
}
