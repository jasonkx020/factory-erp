package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/security"
)

// handleHRPermLifecycle covers onboard/offboard confirm and employee open-account.
func (s *Services) handleHRPermLifecycle(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.Contains(openapiPath, "/hr/employees") && strings.Contains(openapiPath, "/open-account"):
		return s.openEmployeeAccount(c)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/onboards"):
		return s.handleOnboards(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/offboards"):
		return s.handleOffboards(c, method, action)
	case openapiPath == "/api/v1/iam/hr-perm-overview":
		return s.hrPermOverview(c)
	case strings.Contains(openapiPath, "/bind-employee"):
		if method == "PUT" || action == "replace" || action == "update" {
			return s.bindEmployee(c)
		}
		if method == "DELETE" {
			return s.unbindEmployee(c)
		}
		return true
	case strings.Contains(openapiPath, "/data-scope"):
		if method == "GET" || action == "get" || action == "list" {
			return s.getUserDataScope(c)
		}
		if method == "PUT" || action == "replace" || action == "update" {
			return s.putUserDataScope(c)
		}
		return true
	}
	return false
}

func (s *Services) hrPermOverview(c *gin.Context) bool {
	var users, roles, bound, unboundEmp, frozen, sessions int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE COALESCE(is_deleted,0)=0`).Scan(&users)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_role WHERE COALESCE(is_deleted,0)=0`).Scan(&roles)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE COALESCE(is_deleted,0)=0 AND COALESCE(employee_id,0)>0`).Scan(&bound)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE COALESCE(is_deleted,0)=0 AND (user_id IS NULL OR user_id=0) AND status='active'`).Scan(&unboundEmp)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE status='frozen' AND COALESCE(is_deleted,0)=0`).Scan(&frozen)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user_session`).Scan(&sessions)
	api.OK(c, gin.H{
		"users": users, "roles": roles, "bound_users": bound,
		"unbound_employees": unboundEmp, "frozen_users": frozen, "active_sessions": sessions,
	})
	return true
}

func (s *Services) getUserDetailIAM(c *gin.Context) bool {
	uid := paramID(c)
	var login, ut, status string
	var empID int64
	err := s.DB.QueryRow(`SELECT login_name, user_type, status, COALESCE(employee_id,0) FROM iam_user WHERE id=? AND COALESCE(is_deleted,0)=0`, uid).
		Scan(&login, &ut, &status, &empID)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	var empName, empNo string
	if empID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(name,''), COALESCE(emp_no,'') FROM hr_employee WHERE id=?`, empID).Scan(&empName, &empNo)
	}
	roleRows, _ := s.DB.Query(`SELECT r.id, r.code, r.name FROM iam_user_role ur JOIN iam_role r ON r.id=ur.role_id WHERE ur.user_id=?`, uid)
	roles := []gin.H{}
	if roleRows != nil {
		defer roleRows.Close()
		for roleRows.Next() {
			var id int64
			var code, name string
			_ = roleRows.Scan(&id, &code, &name)
			roles = append(roles, gin.H{"id": id, "code": code, "name": name})
		}
	}
	scope := s.loadDataScope(uid)
	api.OK(c, gin.H{
		"id": uid, "login_name": login, "user_type": ut, "status": status,
		"employee_id": empID, "employee_name": empName, "emp_no": empNo,
		"roles": roles, "data_scope": scope,
	})
	return true
}

func (s *Services) bindEmployee(c *gin.Context) bool {
	uid := paramID(c)
	body := bindBody(c)
	empID, _ := asInt64(body["employee_id"])
	if empID <= 0 {
		api.FailJSON(c, "EMPLOYEE_REQUIRED")
		return true
	}
	var uExists, eExists int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE id=? AND COALESCE(is_deleted,0)=0`, uid).Scan(&uExists)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE id=? AND COALESCE(is_deleted,0)=0`, empID).Scan(&eExists)
	if uExists == 0 || eExists == 0 {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	var otherUser, otherEmp int64
	_ = s.DB.QueryRow(`SELECT COALESCE(id,0) FROM iam_user WHERE employee_id=? AND id<>? AND COALESCE(is_deleted,0)=0`, empID, uid).Scan(&otherUser)
	if otherUser > 0 {
		api.FailJSON(c, "EMPLOYEE_ALREADY_BOUND")
		return true
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, uid).Scan(&otherEmp)
	if otherEmp > 0 && otherEmp != empID {
		_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=NULL WHERE id=? AND user_id=?`, otherEmp, uid)
	}
	_, err := s.DB.Exec(`UPDATE iam_user SET employee_id=? WHERE id=?`, empID, uid)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, uid, empID)
	api.OK(c, gin.H{"user_id": uid, "employee_id": empID, "bound": true})
	return true
}

func (s *Services) unbindEmployee(c *gin.Context) bool {
	uid := paramID(c)
	var empID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, uid).Scan(&empID)
	_, _ = s.DB.Exec(`UPDATE iam_user SET employee_id=NULL WHERE id=?`, uid)
	if empID > 0 {
		_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=NULL WHERE id=? AND user_id=?`, empID, uid)
	}
	api.OK(c, gin.H{"user_id": uid, "unbound": true})
	return true
}

func (s *Services) loadDataScope(uid int64) gin.H {
	var scopeType string
	var workshopID, teamID int64
	err := s.DB.QueryRow(`SELECT data_scope_type, COALESCE(workshop_id,0), COALESCE(team_id,0) FROM iam_user_data_scope WHERE user_id=?`, uid).
		Scan(&scopeType, &workshopID, &teamID)
	if err != nil {
		return gin.H{"data_scope_type": "self", "workshop_id": 0, "team_id": 0}
	}
	return gin.H{"data_scope_type": scopeType, "workshop_id": workshopID, "team_id": teamID}
}

func (s *Services) getUserDataScope(c *gin.Context) bool {
	uid := paramID(c)
	api.OK(c, s.loadDataScope(uid))
	return true
}

func (s *Services) putUserDataScope(c *gin.Context) bool {
	uid := paramID(c)
	body := bindBody(c)
	scopeType, _ := body["data_scope_type"].(string)
	if scopeType == "" {
		scopeType = "self"
	}
	workshopID, _ := asInt64(body["workshop_id"])
	teamID, _ := asInt64(body["team_id"])
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user_data_scope WHERE user_id=?`, uid).Scan(&n)
	var err error
	if n == 0 {
		_, err = s.DB.Exec(`INSERT INTO iam_user_data_scope(user_id, data_scope_type, workshop_id, team_id) VALUES(?,?,?,?)`,
			uid, scopeType, nullIf0(workshopID), nullIf0(teamID))
	} else {
		_, err = s.DB.Exec(`UPDATE iam_user_data_scope SET data_scope_type=?, workshop_id=?, team_id=?, updated_at=datetime('now') WHERE user_id=?`,
			scopeType, nullIf0(workshopID), nullIf0(teamID), uid)
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadDataScope(uid))
	return true
}

func (s *Services) openEmployeeAccount(c *gin.Context) bool {
	empID := paramID(c)
	body := bindBody(c)
	var empNo, name, empType string
	var userID int64
	err := s.DB.QueryRow(`SELECT emp_no, name, COALESCE(emp_type,'piece'), COALESCE(user_id,0) FROM hr_employee WHERE id=? AND COALESCE(is_deleted,0)=0`, empID).
		Scan(&empNo, &name, &empType, &userID)
	if err != nil {
		api.FailJSON(c, "EMPLOYEE_NOT_FOUND")
		return true
	}
	if userID > 0 {
		api.FailJSON(c, "ACCOUNT_ALREADY_EXISTS")
		return true
	}
	login, _ := body["login_name"].(string)
	if login == "" {
		login = empNo
		if login == "" {
			login = fmt.Sprintf("u%d", empID)
		}
	}
	pass, _ := body["password"].(string)
	if pass == "" {
		pass = "ChangeMe123"
	}
	ut, _ := body["user_type"].(string)
	if ut == "" {
		ut = "biz"
	}
	hash, err := security.HashPassword(pass)
	if err != nil {
		api.FailJSON(c, "HASH_ERROR")
		return true
	}
	res, err := s.DB.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted) VALUES(?,?,?,?,'active',0)`,
		login, hash, empID, ut)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	uid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, uid, empID)

	roleIDs := []int64{}
	if raw, ok := body["role_ids"].([]interface{}); ok && len(raw) > 0 {
		for _, x := range raw {
			if id, ok := asInt64(x); ok && id > 0 {
				roleIDs = append(roleIDs, id)
			}
		}
	} else {
		rows, _ := s.DB.Query(`SELECT role_id FROM iam_onboard_role_template WHERE emp_type=?`, empType)
		if rows != nil {
			for rows.Next() {
				var rid int64
				_ = rows.Scan(&rid)
				roleIDs = append(roleIDs, rid)
			}
			rows.Close()
		}
		if len(roleIDs) == 0 {
			// fallback: line worker role if exists
			var rid int64
			_ = s.DB.QueryRow(`SELECT id FROM iam_role WHERE code IN ('line_worker','biz_user','sys_admin') ORDER BY CASE code WHEN 'line_worker' THEN 1 WHEN 'biz_user' THEN 2 ELSE 3 END LIMIT 1`).Scan(&rid)
			if rid > 0 {
				roleIDs = append(roleIDs, rid)
			}
		}
	}
	for _, rid := range roleIDs {
		_, _ = s.DB.Exec(`INSERT OR IGNORE INTO iam_user_role(user_id, role_id) VALUES(?,?)`, uid, rid)
	}
	// default data scope from employee workshop
	var workshopID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(workshop_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&workshopID)
	scopeType := "self"
	if workshopID > 0 {
		scopeType = "workshop"
	}
	_, _ = s.DB.Exec(`INSERT OR IGNORE INTO iam_user_data_scope(user_id, data_scope_type, workshop_id) VALUES(?,?,?)`,
		uid, scopeType, nullIf0(workshopID))

	api.OK(c, gin.H{
		"user_id": uid, "employee_id": empID, "login_name": login, "role_ids": roleIDs,
		"data_scope_type": scopeType, "workshop_id": workshopID,
	})
	return true
}

func (s *Services) handleOnboards(c *gin.Context, method, action string) bool {
	ensureHRPermTables(s.DB)
	switch {
	case action == "list":
		statusFilter := c.Query("status")
		q := `
			SELECT o.id, o.employee_id, COALESCE(o.status,'draft'), COALESCE(o.remark,''), COALESCE(o.onboard_date,''),
			       COALESCE(o.need_account,1), COALESCE(o.login_name,''), COALESCE(o.role_ids_json,'[]'), o.created_at,
			       COALESCE(e.emp_no,''), COALESCE(e.name,''), COALESCE(e.emp_type,''), COALESCE(e.status,''),
			       COALESCE(e.dept_id,0), COALESCE(e.workshop_id,0), COALESCE(e.team_id,0), COALESCE(e.job_title,''),
			       COALESCE(e.mobile,''), COALESCE(e.badge_code,''), COALESCE(e.user_id,0)
			FROM hr_onboard o
			LEFT JOIN hr_employee e ON e.id = o.employee_id
			WHERE 1=1`
		args := []interface{}{}
		if statusFilter != "" {
			q += ` AND o.status=?`
			args = append(args, statusFilter)
		}
		q += ` ORDER BY o.id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		var draftN, confirmedN, cancelledN int
		for rows.Next() {
			var id, empID, needAcc, dept, workshop, team, uid int64
			var st, remark, onboardDate, login, roleJSON, created string
			var empNo, empName, empType, empStatus, job, mobile, badge string
			_ = rows.Scan(&id, &empID, &st, &remark, &onboardDate, &needAcc, &login, &roleJSON, &created,
				&empNo, &empName, &empType, &empStatus, &dept, &workshop, &team, &job, &mobile, &badge, &uid)
			switch st {
			case "confirmed":
				confirmedN++
			case "cancelled":
				cancelledN++
			default:
				draftN++
			}
			list = append(list, gin.H{
				"id": id, "employee_id": empID, "status": st, "remark": remark, "onboard_date": onboardDate,
				"need_account": needAcc == 1, "login_name": login, "role_ids": parseJSONIntArr(roleJSON), "created_at": created,
				"emp_no": empNo, "name": empName, "emp_type": empType, "emp_status": empStatus,
				"dept_id": dept, "workshop_id": workshop, "team_id": team, "job_title": job,
				"mobile": mobile, "badge_code": badge, "user_id": uid, "has_account": uid > 0,
			})
		}
		api.OK(c, gin.H{
			"list": list, "total": len(list),
			"summary": gin.H{"draft": draftN, "confirmed": confirmedN, "cancelled": cancelledN},
		})
		return true
	case action == "create":
		body := bindBody(c)
		empID, _ := asInt64(body["employee_id"])
		var empCreated bool
		if empID == 0 {
			// inline 建档
			id, errMsg := s.createEmployeeFromBody(body, "pending")
			if errMsg != "" {
				api.FailJSON(c, errMsg)
				return true
			}
			empID = id
			empCreated = true
		} else {
			// 若员工仍无档案关键字段，允许补写
			if msg, _ := s.updateEmployeeFromBody(empID, body); msg == "INVALID_EMP_TYPE" {
				api.FailJSON(c, msg)
				return true
			}
		}
		onboardDate := strOr(body["onboard_date"])
		if onboardDate == "" {
			onboardDate = time.Now().Format("2006-01-02")
		}
		needAcc := 1
		if v, ok := body["need_account"].(bool); ok && !v {
			needAcc = 0
		}
		login := strOr(body["login_name"])
		res, err := s.DB.Exec(`INSERT INTO hr_onboard(employee_id, status, remark, role_ids_json, onboard_date, need_account, login_name)
			VALUES(?,'draft',?,?,?,?,?)`,
			empID, strOr(body["remark"]), jsonify(body["role_ids"]), onboardDate, needAcc, login)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "employee_id": empID, "status": "draft", "employee_created": empCreated, "onboard_date": onboardDate})
		return true
	case action == "update":
		id := paramID(c)
		var empID int64
		var status string
		err := s.DB.QueryRow(`SELECT employee_id, status FROM hr_onboard WHERE id=?`, id).Scan(&empID, &status)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status != "draft" {
			api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
			return true
		}
		body := bindBody(c)
		if msg, _ := s.updateEmployeeFromBody(empID, body); msg == "INVALID_EMP_TYPE" {
			api.FailJSON(c, msg)
			return true
		}
		onboardDate := strOr(body["onboard_date"])
		remark := strOr(body["remark"])
		login := strOr(body["login_name"])
		needAcc := 1
		if v, ok := body["need_account"].(bool); ok && !v {
			needAcc = 0
		}
		roleJSON := jsonify(body["role_ids"])
		_, err = s.DB.Exec(`UPDATE hr_onboard SET remark=?, onboard_date=COALESCE(NULLIF(?,''),onboard_date),
			need_account=?, login_name=?, role_ids_json=? WHERE id=?`,
			remark, onboardDate, needAcc, login, roleJSON, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadOnboardDetail(id))
		return true
	case action == "action:confirm":
		id := paramID(c)
		var empID int64
		var status, roleJSON, login string
		var needAcc int
		err := s.DB.QueryRow(`SELECT employee_id, status, COALESCE(role_ids_json,'[]'), COALESCE(need_account,1), COALESCE(login_name,'')
			FROM hr_onboard WHERE id=?`, id).Scan(&empID, &status, &roleJSON, &needAcc, &login)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "confirmed" {
			api.OK(c, s.loadOnboardDetail(id))
			return true
		}
		if status == "cancelled" {
			api.FailJSON(c, "ALREADY_CANCELLED")
			return true
		}
		body := bindBody(c)
		if v, ok := body["login_name"].(string); ok && v != "" {
			login = v
		}
		if v, ok := body["need_account"].(bool); ok {
			if v {
				needAcc = 1
			} else {
				needAcc = 0
			}
		}
		var uid int64
		if needAcc == 1 {
			if err := s.openAccountForEmployee(empID, roleJSON, login); err != nil {
				api.FailJSON(c, err.Error())
				return true
			}
		} else if roleJSON != "" && roleJSON != "[]" {
			// 已有账号则仅补角色
			_ = s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&uid)
			if uid > 0 {
				var arr []interface{}
				_ = jsonUnmarshal(roleJSON, &arr)
				for _, x := range arr {
					if rid, ok := asInt64(x); ok && rid > 0 {
						_, _ = s.DB.Exec(`INSERT OR IGNORE INTO iam_user_role(user_id, role_id) VALUES(?,?)`, uid, rid)
					}
				}
			}
		}
		_, _ = s.DB.Exec(`UPDATE hr_onboard SET status='confirmed' WHERE id=?`, id)
		_, _ = s.DB.Exec(`UPDATE hr_employee SET status='active' WHERE id=?`, empID)
		api.OK(c, s.loadOnboardDetail(id))
		return true
	case action == "action:cancel":
		id := paramID(c)
		var status string
		err := s.DB.QueryRow(`SELECT status FROM hr_onboard WHERE id=?`, id).Scan(&status)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "confirmed" {
			api.FailJSON(c, "CONFIRMED_CANNOT_CANCEL")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE hr_onboard SET status='cancelled' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "cancelled"})
		return true
	case action == "get":
		id := paramID(c)
		d := s.loadOnboardDetail(id)
		if d == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, d)
		return true
	}
	return true
}

func parseJSONIntArr(s string) []int64 {
	out := []int64{}
	var arr []interface{}
	_ = jsonUnmarshal(s, &arr)
	for _, x := range arr {
		if id, ok := asInt64(x); ok && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func (s *Services) loadOnboardDetail(id int64) gin.H {
	var empID int64
	var st, remark, onboardDate, login, roleJSON, created string
	var needAcc int
	err := s.DB.QueryRow(`SELECT employee_id, COALESCE(status,'draft'), COALESCE(remark,''), COALESCE(onboard_date,''),
		COALESCE(need_account,1), COALESCE(login_name,''), COALESCE(role_ids_json,'[]'), created_at
		FROM hr_onboard WHERE id=?`, id).Scan(&empID, &st, &remark, &onboardDate, &needAcc, &login, &roleJSON, &created)
	if err != nil {
		return nil
	}
	emp := s.loadEmployeeMap(empID)
	return gin.H{
		"id": id, "employee_id": empID, "status": st, "remark": remark, "onboard_date": onboardDate,
		"need_account": needAcc == 1, "login_name": login, "role_ids": parseJSONIntArr(roleJSON),
		"created_at": created, "employee": emp,
		"emp_no": emp["emp_no"], "name": emp["name"], "emp_type": emp["emp_type"], "emp_status": emp["status"],
		"has_account": emp["has_account"], "user_id": emp["user_id"],
	}
}

func (s *Services) loadEmployeeMap(id int64) gin.H {
	var no, name, typ, status, job, mobile, badge string
	var org, dept, workshop, team, uid int64
	err := s.DB.QueryRow(`SELECT emp_no, name, COALESCE(org_id,0), COALESCE(dept_id,0), COALESCE(workshop_id,0), COALESCE(team_id,0),
		COALESCE(job_title,''), COALESCE(emp_type,''), COALESCE(mobile,''), COALESCE(badge_code,''), COALESCE(status,''), COALESCE(user_id,0)
		FROM hr_employee WHERE id=?`, id).
		Scan(&no, &name, &org, &dept, &workshop, &team, &job, &typ, &mobile, &badge, &status, &uid)
	if err != nil {
		return gin.H{"id": id}
	}
	return gin.H{
		"id": id, "emp_no": no, "name": name, "org_id": org, "dept_id": dept, "workshop_id": workshop, "team_id": team,
		"job_title": job, "emp_type": typ, "mobile": mobile, "badge_code": badge, "status": status,
		"user_id": uid, "has_account": uid > 0,
	}
}

func (s *Services) createEmployeeFromBody(body map[string]interface{}, status string) (int64, string) {
	no := strOr(body["emp_no"])
	name := strOr(body["name"])
	if no == "" || name == "" {
		return 0, "EMP_NO_NAME_REQUIRED"
	}
	if status == "" {
		status = "active"
	}
	typ, ok := normalizeEmpType(strOr(body["emp_type"]))
	if !ok {
		return 0, "INVALID_EMP_TYPE"
	}
	orgID, _ := asInt64(body["org_id"])
	if orgID == 0 {
		orgID = 1
	}
	deptID, _ := asInt64(body["dept_id"])
	workshopID, _ := asInt64(body["workshop_id"])
	teamID, _ := asInt64(body["team_id"])
	job := strOr(body["job_title"])
	mobile := strOr(body["mobile"])
	badge := strOr(body["badge_code"])
	res, err := s.DB.Exec(`INSERT INTO hr_employee(emp_no, name, org_id, dept_id, workshop_id, team_id, job_title, emp_type, mobile, badge_code, status)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		no, name, orgID, nullIf0(deptID), nullIf0(workshopID), nullIf0(teamID), job, typ, mobile, badge, status)
	if err != nil {
		return 0, "DB_ERROR:" + err.Error()
	}
	id, _ := res.LastInsertId()
	return id, ""
}

func (s *Services) updateEmployeeFromBody(id int64, body map[string]interface{}) (string, error) {
	cur := s.loadEmployeeMap(id)
	if cur["emp_no"] == nil {
		return "NOT_FOUND", fmt.Errorf("not found")
	}
	name := strOrDef(body["name"], fmt.Sprint(cur["name"]))
	typ, typOK := normalizeEmpType(strOrDef(body["emp_type"], fmt.Sprint(cur["emp_type"])))
	if !typOK {
		return "INVALID_EMP_TYPE", fmt.Errorf("invalid emp_type")
	}
	job := strOrDef(body["job_title"], fmt.Sprint(cur["job_title"]))
	mobile := strOrDef(body["mobile"], fmt.Sprint(cur["mobile"]))
	badge := strOrDef(body["badge_code"], fmt.Sprint(cur["badge_code"]))
	deptID, ok := asInt64(body["dept_id"])
	if !ok {
		deptID, _ = asInt64(cur["dept_id"])
	}
	workshopID, ok := asInt64(body["workshop_id"])
	if !ok {
		workshopID, _ = asInt64(cur["workshop_id"])
	}
	teamID, ok := asInt64(body["team_id"])
	if !ok {
		teamID, _ = asInt64(cur["team_id"])
	}
	_, err := s.DB.Exec(`UPDATE hr_employee SET name=?, emp_type=?, job_title=?, mobile=?, badge_code=?,
		dept_id=?, workshop_id=?, team_id=?, updated_at=datetime('now') WHERE id=?`,
		name, typ, job, mobile, badge, nullIf0(deptID), nullIf0(workshopID), nullIf0(teamID), id)
	return "", err
}

func (s *Services) openAccountForEmployee(empID int64, roleJSON, loginHint string) error {
	var userID, workshopID int64
	var empNo, empType string
	if err := s.DB.QueryRow(`SELECT COALESCE(user_id,0), COALESCE(emp_no,''), COALESCE(emp_type,'piece'), COALESCE(workshop_id,0) FROM hr_employee WHERE id=?`, empID).
		Scan(&userID, &empNo, &empType, &workshopID); err != nil {
		return fmt.Errorf("EMPLOYEE_NOT_FOUND")
	}
	if userID > 0 {
		// still apply roles if provided
		var arr []interface{}
		_ = jsonUnmarshal(roleJSON, &arr)
		for _, x := range arr {
			if rid, ok := asInt64(x); ok && rid > 0 {
				_, _ = s.DB.Exec(`INSERT OR IGNORE INTO iam_user_role(user_id, role_id) VALUES(?,?)`, userID, rid)
			}
		}
		return nil
	}
	login := loginHint
	if login == "" {
		login = empNo
		if login == "" {
			login = fmt.Sprintf("u%d", empID)
		}
	}
	hash, err := security.HashPassword("ChangeMe123")
	if err != nil {
		return fmt.Errorf("HASH_ERROR")
	}
	res, err := s.DB.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted) VALUES(?,?,?,'biz','active',0)`,
		login, hash, empID)
	if err != nil {
		return fmt.Errorf("DB_ERROR:%s", err.Error())
	}
	uid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, uid, empID)
	roleIDs := []int64{}
	var arr []interface{}
	_ = jsonUnmarshal(roleJSON, &arr)
	for _, x := range arr {
		if id, ok := asInt64(x); ok {
			roleIDs = append(roleIDs, id)
		}
	}
	if len(roleIDs) == 0 {
		rows, _ := s.DB.Query(`SELECT role_id FROM iam_onboard_role_template WHERE emp_type=?`, empType)
		if rows != nil {
			for rows.Next() {
				var rid int64
				_ = rows.Scan(&rid)
				roleIDs = append(roleIDs, rid)
			}
			rows.Close()
		}
		if len(roleIDs) == 0 {
			var rid int64
			_ = s.DB.QueryRow(`SELECT id FROM iam_role WHERE code IN ('piece','fixed','line_worker','biz_user') ORDER BY id LIMIT 1`).Scan(&rid)
			if rid > 0 {
				roleIDs = append(roleIDs, rid)
			}
		}
	}
	for _, rid := range roleIDs {
		_, _ = s.DB.Exec(`INSERT OR IGNORE INTO iam_user_role(user_id, role_id) VALUES(?,?)`, uid, rid)
	}
	scopeType := "self"
	if workshopID > 0 {
		scopeType = "workshop"
	}
	_, _ = s.DB.Exec(`INSERT OR IGNORE INTO iam_user_data_scope(user_id, data_scope_type, workshop_id) VALUES(?,?,?)`,
		uid, scopeType, nullIf0(workshopID))
	return nil
}

func jsonUnmarshal(s string, v interface{}) error {
	if s == "" {
		s = "[]"
	}
	return json.Unmarshal([]byte(s), v)
}

func (s *Services) handleOffboards(c *gin.Context, method, action string) bool {
	ensureHRPermTables(s.DB)
	ensureHROpsTables(s.DB)
	switch {
	case action == "list":
		statusFilter := c.Query("status")
		q := `SELECT o.id, o.employee_id, COALESCE(o.status,'draft'), COALESCE(o.revoke_permission,1), COALESCE(o.reason,''),
			COALESCE(o.offboard_date,''), o.created_at, COALESCE(e.emp_no,''), COALESCE(e.name,''), COALESCE(e.user_id,0), COALESCE(e.status,'')
			FROM hr_offboard o LEFT JOIN hr_employee e ON e.id=o.employee_id WHERE 1=1`
		args := []interface{}{}
		if statusFilter != "" {
			q += ` AND o.status=?`
			args = append(args, statusFilter)
		}
		q += ` ORDER BY o.id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		var draftN, confirmedN int
		for rows.Next() {
			var id, empID, uid int64
			var revoke int
			var st, reason, od, created, empNo, name, empStatus string
			_ = rows.Scan(&id, &empID, &st, &revoke, &reason, &od, &created, &empNo, &name, &uid, &empStatus)
			if st == "confirmed" {
				confirmedN++
			} else {
				draftN++
			}
			list = append(list, gin.H{
				"id": id, "employee_id": empID, "status": st, "revoke_permission": revoke == 1, "reason": reason,
				"offboard_date": od, "created_at": created, "emp_no": empNo, "name": name, "user_id": uid, "emp_status": empStatus,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list), "summary": gin.H{"draft": draftN, "confirmed": confirmedN}})
		return true
	case action == "create":
		body := bindBody(c)
		empID, _ := asInt64(body["employee_id"])
		if empID == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		revoke := 1
		if v, ok := body["revoke_permission"].(bool); ok && !v {
			revoke = 0
		}
		od := strOrDef(body["offboard_date"], time.Now().Format("2006-01-02"))
		res, err := s.DB.Exec(`INSERT INTO hr_offboard(employee_id, status, revoke_permission, reason, offboard_date) VALUES(?,'draft',?,?,?)`,
			empID, revoke, strOr(body["reason"]), od)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "employee_id": empID, "status": "draft", "offboard_date": od})
		return true
	case action == "update":
		id := paramID(c)
		var status string
		err := s.DB.QueryRow(`SELECT status FROM hr_offboard WHERE id=?`, id).Scan(&status)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status != "draft" {
			api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
			return true
		}
		body := bindBody(c)
		revoke := 1
		if v, ok := body["revoke_permission"].(bool); ok && !v {
			revoke = 0
		}
		_, _ = s.DB.Exec(`UPDATE hr_offboard SET reason=?, offboard_date=COALESCE(NULLIF(?,''),offboard_date), revoke_permission=? WHERE id=?`,
			strOr(body["reason"]), strOr(body["offboard_date"]), revoke, id)
		api.OK(c, gin.H{"id": id, "status": "draft"})
		return true
	case action == "action:confirm":
		id := paramID(c)
		var empID int64
		var status string
		var revoke int
		err := s.DB.QueryRow(`SELECT employee_id, status, COALESCE(revoke_permission,1) FROM hr_offboard WHERE id=?`, id).Scan(&empID, &status, &revoke)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "confirmed" {
			api.OK(c, gin.H{"id": id, "status": "confirmed"})
			return true
		}
		body := bindBody(c)
		force, _ := body["force"].(bool)
		var openTools int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_tool_issue WHERE (employee_id=? OR employee_id IN (SELECT id FROM hr_employee WHERE id=?)) AND status='open'`, empID, empID).Scan(&openTools)
		// also match by employee name if issues only stored name
		if openTools == 0 {
			var ename string
			_ = s.DB.QueryRow(`SELECT name FROM hr_employee WHERE id=?`, empID).Scan(&ename)
			if ename != "" {
				_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_tool_issue WHERE status='open' AND employee_name=?`, ename).Scan(&openTools)
			}
		}
		if openTools > 0 && !force {
			api.FailJSON(c, fmt.Sprintf("OPEN_TOOLS:%d", openTools))
			return true
		}
		var uid int64
		_ = s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&uid)
		if uid == 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(id,0) FROM iam_user WHERE employee_id=?`, empID).Scan(&uid)
		}
		if revoke == 1 && uid > 0 {
			claims := middleware.Claims(c)
			var by interface{}
			if claims != nil {
				by = claims.UserID
			}
			now := time.Now().Format("2006-01-02 15:04:05")
			_, _ = s.DB.Exec(`DELETE FROM iam_user_role WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`DELETE FROM iam_admin_group_user WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`DELETE FROM iam_user_session WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`UPDATE iam_user SET status='frozen', freeze_reason=?, frozen_at=?, frozen_by=? WHERE id=?`,
				"offboard revoke", now, by, uid)
		}
		_, _ = s.DB.Exec(`UPDATE hr_employee SET status='left' WHERE id=?`, empID)
		_, _ = s.DB.Exec(`UPDATE hr_offboard SET status='confirmed' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "employee_id": empID, "user_id": uid, "status": "confirmed", "revoked": revoke == 1})
		return true
	case action == "get":
		id := paramID(c)
		var empID, uid int64
		var st, reason, od, created, empNo, name string
		var revoke int
		err := s.DB.QueryRow(`SELECT o.employee_id, COALESCE(o.status,'draft'), COALESCE(o.revoke_permission,1), COALESCE(o.reason,''),
			COALESCE(o.offboard_date,''), o.created_at, COALESCE(e.emp_no,''), COALESCE(e.name,''), COALESCE(e.user_id,0)
			FROM hr_offboard o LEFT JOIN hr_employee e ON e.id=o.employee_id WHERE o.id=?`, id).
			Scan(&empID, &st, &revoke, &reason, &od, &created, &empNo, &name, &uid)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{
			"id": id, "employee_id": empID, "status": st, "revoke_permission": revoke == 1, "reason": reason,
			"offboard_date": od, "created_at": created, "emp_no": empNo, "name": name, "user_id": uid,
		})
		return true
	}
	return true
}

func ensureHRPermTables(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS hr_onboard (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  role_ids_json TEXT,
  onboard_date TEXT,
  need_account INTEGER NOT NULL DEFAULT 1,
  login_name TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_offboard (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  revoke_permission INTEGER NOT NULL DEFAULT 1,
  reason TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS iam_onboard_role_template (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  emp_type TEXT NOT NULL,
  role_id INTEGER NOT NULL,
  UNIQUE(emp_type, role_id)
)`,
		`ALTER TABLE hr_onboard ADD COLUMN role_ids_json TEXT`,
		`ALTER TABLE hr_onboard ADD COLUMN onboard_date TEXT`,
		`ALTER TABLE hr_onboard ADD COLUMN need_account INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE hr_onboard ADD COLUMN login_name TEXT`,
		`ALTER TABLE hr_offboard ADD COLUMN revoke_permission INTEGER NOT NULL DEFAULT 1`,
	}
	for _, q := range stmts {
		_, _ = db.Exec(q)
	}
	// 按员工类型默认角色模板
	for _, t := range EmpTypes {
		_, _ = db.Exec(`INSERT OR IGNORE INTO iam_onboard_role_template(emp_type, role_id)
			SELECT ?, id FROM iam_role WHERE code=? LIMIT 1`, t.Code, t.RoleCode)
	}
}

// EnsureHRPermSchema is called at startup.
func EnsureHRPermSchema(db *sql.DB) {
	ensureHRPermTables(db)
}
