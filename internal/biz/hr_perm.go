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
	ensureExtraRoleTable(s.DB)
	var login, ut, status string
	var empID int64
	err := s.DB.QueryRow(`SELECT login_name, user_type, status, COALESCE(employee_id,0) FROM iam_user WHERE id=? AND COALESCE(is_deleted,0)=0`, uid).
		Scan(&login, &ut, &status, &empID)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	var empName, empNo string
	var deptID int64
	var deptName string
	if empID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(name,''), COALESCE(emp_no,''), COALESCE(dept_id,0) FROM hr_employee WHERE id=?`, empID).
			Scan(&empName, &empNo, &deptID)
		if deptID > 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM sys_department WHERE id=?`, deptID).Scan(&deptName)
		}
	}
	extraIDs, _ := s.getExtraRoleIDs(uid)
	deptBaseIDs, _ := s.getDeptBaseRoleIDsForUser(uid)
	effectiveIDs := unionInt64(extraIDs, deptBaseIDs)
	api.OK(c, gin.H{
		"id": uid, "login_name": login, "user_type": ut, "status": status,
		"employee_id": empID, "employee_name": empName, "emp_no": empNo,
		"dept_id": deptID, "dept_name": deptName, "departments": s.loadEmployeeDepartments(empID),
		"extra_roles":     s.loadRoleDetailsByIDs(extraIDs),
		"dept_base_roles": s.loadRoleDetailsByIDs(deptBaseIDs),
		"roles":           s.loadRoleDetailsByIDs(effectiveIDs),
		"data_scope":      s.loadDataScope(uid),
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
	s.rebuildUserEffectiveRoles(uid)
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
	var deptID, teamID int64
	err := s.DB.QueryRow(`SELECT data_scope_type, COALESCE(dept_id,0), COALESCE(team_id,0) FROM iam_user_data_scope WHERE user_id=?`, uid).
		Scan(&scopeType, &deptID, &teamID)
	if err != nil {
		return gin.H{"data_scope_type": "self", "dept_id": 0, "team_id": 0}
	}
	return gin.H{"data_scope_type": scopeType, "dept_id": deptID, "team_id": teamID}
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
	deptID, _ := asInt64(body["dept_id"])
	if deptID == 0 {
		deptID = s.workshopDeptIDFromBody(body)
	}
	teamID, _ := asInt64(body["team_id"])
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user_data_scope WHERE user_id=?`, uid).Scan(&n)
	var err error
	if n == 0 {
		_, err = s.DB.Exec(`INSERT INTO iam_user_data_scope(user_id, data_scope_type, dept_id, team_id) VALUES(?,?,?,?)`,
			uid, scopeType, nullIf0(deptID), nullIf0(teamID))
	} else {
		_, err = s.DB.Exec(`UPDATE iam_user_data_scope SET data_scope_type=?, dept_id=?, team_id=?, updated_at=NOW() WHERE user_id=?`,
			scopeType, nullIf0(deptID), nullIf0(teamID), uid)
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
	var existingUID int64
	if err := s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=? AND COALESCE(is_deleted,0)=0`, empID).Scan(&existingUID); err != nil {
		api.FailJSON(c, "EMPLOYEE_NOT_FOUND")
		return true
	}
	if existingUID > 0 {
		api.FailJSON(c, "ACCOUNT_ALREADY_EXISTS")
		return true
	}
	loginHint := strOr(body["login_name"])
	pass := strOr(body["password"])
	roleJSON := "[]"
	if raw, ok := body["role_ids"].([]interface{}); ok && len(raw) > 0 {
		b, _ := json.Marshal(raw)
		roleJSON = string(b)
	}
	// 未传密码时：优先身份证后 6 位
	if pass == "" {
		var idCard string
		_ = s.DB.QueryRow(`SELECT COALESCE(id_card_no,'') FROM hr_employee WHERE id=?`, empID).Scan(&idCard)
		pass = initialPasswordFromIDCard(idCard)
	}
	login, initialPass, err := s.openAccountForEmployeeEx(empID, roleJSON, loginHint, pass)
	if err != nil {
		msg := err.Error()
		if strings.HasPrefix(msg, "DB_ERROR:") || msg == "EMPLOYEE_NOT_FOUND" || msg == "ACCOUNT_ALREADY_EXISTS" || msg == "HASH_ERROR" {
			api.FailJSON(c, msg)
			return true
		}
		api.FailJSON(c, "DB_ERROR:"+msg)
		return true
	}
	wsDeptID := s.employeePrimaryWorkshopDeptID(empID)
	scopeType := "self"
	if wsDeptID > 0 {
		scopeType = "dept_workshop"
	}
	var uid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&uid)
	api.OK(c, gin.H{
		"user_id": uid, "employee_id": empID, "login_name": login,
		"initial_password": initialPass,
		"data_scope_type":  scopeType, "dept_id": wsDeptID,
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
			       COALESCE(e.dept_id,0), COALESCE(e.team_id,0), COALESCE(e.job_title,''),
			       COALESCE(e.mobile,''), COALESCE(e.badge_code,''), COALESCE(e.user_id,0), COALESCE(e.id_card_no,''),
			       COALESCE(p.bank_account,''), COALESCE(p.tax_no,'')
			FROM hr_onboard o
			LEFT JOIN hr_employee e ON e.id = o.employee_id
			LEFT JOIN pay_worker_profile p ON p.employee_id = o.employee_id
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
			var id, empID, needAcc, dept, team, uid int64
			var st, remark, onboardDate, login, roleJSON, created string
			var empNo, empName, empType, empStatus, job, mobile, badge, idCard, bank, tax string
			_ = rows.Scan(&id, &empID, &st, &remark, &onboardDate, &needAcc, &login, &roleJSON, &created,
				&empNo, &empName, &empType, &empStatus, &dept, &team, &job, &mobile, &badge, &uid, &idCard, &bank, &tax)
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
				"dept_id": dept, "team_id": team, "job_title": job,
				"mobile": mobile, "badge_code": badge, "id_card_no": idCard, "user_id": uid, "has_account": uid > 0,
				"bank_account": bank, "tax_no": tax,
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
			// 已有账号则仅补个人特殊角色
			_ = s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&uid)
			if uid > 0 {
				var arr []interface{}
				_ = jsonUnmarshal(roleJSON, &arr)
				extra := []int64{}
				for _, x := range arr {
					if rid, ok := asInt64(x); ok && rid > 0 {
						extra = append(extra, rid)
					}
				}
				appendExtraRoleIDs(s.DB, uid, extra)
				s.rebuildUserEffectiveRoles(uid)
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
	emp := s.loadEmployeeMapEnriched(empID)
	return gin.H{
		"id": id, "employee_id": empID, "status": st, "remark": remark, "onboard_date": onboardDate,
		"need_account": needAcc == 1, "login_name": login, "role_ids": parseJSONIntArr(roleJSON),
		"created_at": created, "employee": emp,
		"emp_no": emp["emp_no"], "name": emp["name"], "emp_type": emp["emp_type"], "emp_status": emp["status"],
		"has_account": emp["has_account"], "user_id": emp["user_id"],
	}
}

func (s *Services) loadEmployeeMap(id int64) gin.H {
	var no, name, typ, status, job, mobile, badge, idCard, bank, tax string
	var org, dept, team, uid int64
	err := s.DB.QueryRow(`SELECT e.emp_no, e.name, COALESCE(e.org_id,0), COALESCE(e.dept_id,0), COALESCE(e.team_id,0),
		COALESCE(e.job_title,''), COALESCE(e.emp_type,''), COALESCE(e.mobile,''), COALESCE(e.badge_code,''), COALESCE(e.id_card_no,''),
		COALESCE(e.status,''), COALESCE(e.user_id,0),
		COALESCE(p.bank_account,''), COALESCE(p.tax_no,'')
		FROM hr_employee e
		LEFT JOIN pay_worker_profile p ON p.employee_id=e.id
		WHERE e.id=?`, id).
		Scan(&no, &name, &org, &dept, &team, &job, &typ, &mobile, &badge, &idCard, &status, &uid, &bank, &tax)
	if err != nil {
		return gin.H{"id": id}
	}
	return gin.H{
		"id": id, "emp_no": no, "name": name, "org_id": org, "dept_id": dept, "team_id": team,
		"job_title": job, "emp_type": typ, "mobile": mobile, "badge_code": badge, "id_card_no": idCard, "status": status,
		"user_id": uid, "has_account": uid > 0, "bank_account": bank, "tax_no": tax,
	}
}

func (s *Services) loadEmployeeMapEnriched(id int64) gin.H {
	return s.enrichEmployeeDeptFields(s.loadEmployeeMap(id))
}

// allocEmpNo 生成唯一工号：E + yyMMdd + 4 位序号。
func (s *Services) allocEmpNo() string {
	prefix := "E" + time.Now().Format("060102")
	var last string
	_ = s.DB.QueryRow(`SELECT emp_no FROM hr_employee WHERE emp_no LIKE ? ORDER BY emp_no DESC LIMIT 1`, prefix+"%").Scan(&last)
	seq := 1
	if strings.HasPrefix(last, prefix) && len(last) > len(prefix) {
		var n int
		if _, err := fmt.Sscanf(last[len(prefix):], "%d", &n); err == nil {
			seq = n + 1
		}
	}
	for i := 0; i < 200; i++ {
		no := fmt.Sprintf("%s%04d", prefix, seq+i)
		var cnt int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE emp_no=? AND COALESCE(is_deleted,0)=0`, no).Scan(&cnt)
		if cnt == 0 {
			return no
		}
	}
	return fmt.Sprintf("E%d", time.Now().UnixNano()%1e12)
}

func normalizeMobile(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

func normalizeIDCard(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func validMobileCN(mobile string) bool {
	if len(mobile) != 11 {
		return false
	}
	if mobile[0] != '1' {
		return false
	}
	for i := 0; i < 11; i++ {
		if mobile[i] < '0' || mobile[i] > '9' {
			return false
		}
	}
	return true
}

func validIDCardCN(id string) bool {
	n := len(id)
	if n != 15 && n != 18 {
		return false
	}
	for i := 0; i < n; i++ {
		c := id[i]
		if c >= '0' && c <= '9' {
			continue
		}
		if n == 18 && i == 17 && (c == 'X' || c == 'x') {
			continue
		}
		return false
	}
	return true
}

// initialPasswordFromIDCard 初始密码取身份证后 6 位；不足则 ChangeMe123。
func initialPasswordFromIDCard(idCard string) string {
	id := normalizeIDCard(idCard)
	if len(id) >= 6 {
		return id[len(id)-6:]
	}
	return "ChangeMe123"
}

// allocLoginName 优先手机号，冲突则用工号，再退回 u{id}。
func (s *Services) allocLoginName(mobile, empNo string, empID int64) string {
	candidates := []string{
		strings.TrimSpace(mobile),
		strings.TrimSpace(empNo),
		fmt.Sprintf("u%d", empID),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE login_name=? AND COALESCE(is_deleted,0)=0`, c).Scan(&n)
		if n == 0 {
			return c
		}
	}
	return fmt.Sprintf("u%d_%d", empID, time.Now().UnixNano()%1e6)
}

// allocBadgeCode 生成唯一工牌码（EMP-{工号}，冲突则加后缀；无工号则 EMP{id}）。
func (s *Services) allocBadgeCode(empNo string, empID int64) string {
	clean := strings.ToUpper(strings.TrimSpace(empNo))
	var b strings.Builder
	for _, r := range clean {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	base := b.String()
	if base == "" {
		if empID > 0 {
			base = fmt.Sprintf("%06d", empID)
		} else {
			base = fmt.Sprintf("%d", time.Now().UnixNano()%1e10)
		}
	}
	candidates := []string{
		"EMP-" + base,
		fmt.Sprintf("EMP-%s-%d", base, empID),
		fmt.Sprintf("EMP%d", time.Now().UnixNano()%1e12),
	}
	for _, code := range candidates {
		if code == "" || code == "EMP-" {
			continue
		}
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE badge_code=? AND id<>? AND COALESCE(is_deleted,0)=0`, code, empID).Scan(&n)
		if n == 0 {
			return code
		}
	}
	return fmt.Sprintf("EMP%d", time.Now().UnixNano())
}

// ensureEmployeeBadge 若员工无工牌则自动补全并写回。
func (s *Services) ensureEmployeeBadge(empID int64, empNo, curBadge string) string {
	curBadge = strings.TrimSpace(curBadge)
	if curBadge != "" {
		return curBadge
	}
	code := s.allocBadgeCode(empNo, empID)
	_, _ = s.DB.Exec(`UPDATE hr_employee SET badge_code=?, updated_at=NOW() WHERE id=?`, code, empID)
	return code
}

func (s *Services) createEmployeeFromBody(body map[string]interface{}, status string) (int64, string) {
	name := strings.TrimSpace(strOr(body["name"]))
	mobile := normalizeMobile(strOr(body["mobile"]))
	idCard := normalizeIDCard(strOr(body["id_card_no"]))
	if name == "" {
		return 0, "NAME_REQUIRED"
	}
	if idCard == "" {
		return 0, "ID_CARD_REQUIRED"
	}
	if !validIDCardCN(idCard) {
		return 0, "ID_CARD_INVALID"
	}
	if mobile == "" {
		return 0, "MOBILE_REQUIRED"
	}
	if !validMobileCN(mobile) {
		return 0, "MOBILE_INVALID"
	}
	var dupID int64
	_ = s.DB.QueryRow(`SELECT id FROM hr_employee WHERE id_card_no=? AND COALESCE(is_deleted,0)=0 LIMIT 1`, idCard).Scan(&dupID)
	if dupID > 0 {
		return 0, "ID_CARD_DUPLICATE"
	}

	no := strings.TrimSpace(strOr(body["emp_no"]))
	if no != "" {
		var exist int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE emp_no=? AND COALESCE(is_deleted,0)=0`, no).Scan(&exist)
		if exist > 0 {
			return 0, "EMP_NO_DUPLICATE"
		}
	} else {
		no = s.allocEmpNo()
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
	deptIDs, primaryDept := deptIDsFromBody(body)
	if len(deptIDs) == 0 && deptID == 0 {
		deptID = 1
		primaryDept = 1
		deptIDs = []int64{1}
	}
	teamID, _ := asInt64(body["team_id"])
	job := strOr(body["job_title"])
	// 工牌由系统自动生成
	badge := s.allocBadgeCode(no, 0)
	res, err := s.DB.Exec(`INSERT INTO hr_employee(emp_no, name, org_id, dept_id, team_id, job_title, emp_type, mobile, badge_code, id_card_no, status)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		no, name, orgID, nullIf0(primaryDept), nullIf0(teamID), job, typ, mobile, badge, idCard, status)
	if err != nil {
		return 0, "DB_ERROR:" + err.Error()
	}
	id, _ := res.LastInsertId()
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE badge_code=? AND id<>? AND COALESCE(is_deleted,0)=0`, badge, id).Scan(&n)
	if n > 0 || badge == "" {
		badge = s.allocBadgeCode(no, id)
		_, _ = s.DB.Exec(`UPDATE hr_employee SET badge_code=? WHERE id=?`, badge, id)
	}
	s.syncEmployeePayProfile(id, body, typ)
	if len(deptIDs) > 0 || primaryDept > 0 {
		_ = s.setEmployeeDepartments(id, deptIDs, primaryDept)
	}
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
	mobile := normalizeMobile(strOrDef(body["mobile"], fmt.Sprint(cur["mobile"])))
	if mobile != "" && !validMobileCN(mobile) {
		return "MOBILE_INVALID", fmt.Errorf("mobile invalid")
	}
	idCard := normalizeIDCard(strOrDef(body["id_card_no"], fmt.Sprint(cur["id_card_no"])))
	if idCard != "" {
		if !validIDCardCN(idCard) {
			return "ID_CARD_INVALID", fmt.Errorf("id card invalid")
		}
		var dupID int64
		_ = s.DB.QueryRow(`SELECT id FROM hr_employee WHERE id_card_no=? AND id<>? AND COALESCE(is_deleted,0)=0 LIMIT 1`, idCard, id).Scan(&dupID)
		if dupID > 0 {
			return "ID_CARD_DUPLICATE", fmt.Errorf("id card duplicate")
		}
	}
	// 工牌不随普通编辑改写；缺失时自动补全；工号不可通过本接口改写
	badge := s.ensureEmployeeBadge(id, fmt.Sprint(cur["emp_no"]), fmt.Sprint(cur["badge_code"]))
	deptID, ok := asInt64(body["dept_id"])
	if !ok {
		deptID, _ = asInt64(cur["dept_id"])
	}
	deptIDs, primaryDept := deptIDsFromBody(body)
	if len(deptIDs) == 0 && deptID > 0 {
		deptIDs = []int64{deptID}
		primaryDept = deptID
	}
	if primaryDept <= 0 && len(deptIDs) > 0 {
		primaryDept = deptIDs[0]
	}
	teamID, ok := asInt64(body["team_id"])
	if !ok {
		teamID, _ = asInt64(cur["team_id"])
	}
	_, err := s.DB.Exec(`UPDATE hr_employee SET name=?, emp_type=?, job_title=?, mobile=?, badge_code=?, id_card_no=?,
		dept_id=?, team_id=?, updated_at=NOW() WHERE id=?`,
		name, typ, job, mobile, badge, idCard, nullIf0(primaryDept), nullIf0(teamID), id)
	if err == nil {
		s.syncEmployeePayProfile(id, body, typ)
		if _, ok := int64SliceFromBody(body, "dept_ids"); ok || body["dept_id"] != nil || body["primary_dept_id"] != nil {
			_ = s.setEmployeeDepartments(id, deptIDs, primaryDept)
		}
	}
	return "", err
}

func (s *Services) openAccountForEmployee(empID int64, roleJSON, loginHint string) error {
	_, _, err := s.openAccountForEmployeeEx(empID, roleJSON, loginHint, "")
	return err
}

// openAccountForEmployeeEx 开通登录账号。password 空则用身份证后 6 位；loginHint 空则 allocLoginName。
// 返回实际登录名与本次使用的明文初始密码（仅供当次响应展示）。
func (s *Services) openAccountForEmployeeEx(empID int64, roleJSON, loginHint, password string) (loginName, initialPass string, err error) {
	var userID int64
	var empNo, empType, mobile, idCard string
	if err = s.DB.QueryRow(`SELECT COALESCE(user_id,0), COALESCE(emp_no,''), COALESCE(emp_type,'piece'),
		COALESCE(mobile,''), COALESCE(id_card_no,'') FROM hr_employee WHERE id=?`, empID).
		Scan(&userID, &empNo, &empType, &mobile, &idCard); err != nil {
		return "", "", fmt.Errorf("EMPLOYEE_NOT_FOUND")
	}
	if userID > 0 {
		var arr []interface{}
		_ = jsonUnmarshal(roleJSON, &arr)
		extraIDs := []int64{}
		for _, x := range arr {
			if rid, ok := asInt64(x); ok && rid > 0 {
				extraIDs = append(extraIDs, rid)
			}
		}
		if len(extraIDs) > 0 {
			appendExtraRoleIDs(s.DB, userID, extraIDs)
		}
		s.rebuildUserEffectiveRoles(userID)
		var existingLogin string
		_ = s.DB.QueryRow(`SELECT COALESCE(login_name,'') FROM iam_user WHERE id=?`, userID).Scan(&existingLogin)
		return existingLogin, "", nil
	}
	login := strings.TrimSpace(loginHint)
	if login == "" {
		login = s.allocLoginName(mobile, empNo, empID)
	} else {
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE login_name=? AND COALESCE(is_deleted,0)=0`, login).Scan(&n)
		if n > 0 {
			login = s.allocLoginName(mobile, empNo, empID)
		}
	}
	pass := password
	if pass == "" {
		pass = initialPasswordFromIDCard(idCard)
	}
	hash, herr := security.HashPassword(pass)
	if herr != nil {
		return "", "", fmt.Errorf("HASH_ERROR")
	}
	res, ierr := s.DB.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted) VALUES(?,?,?,'biz','active',0)`,
		login, hash, empID)
	if ierr != nil {
		return "", "", fmt.Errorf("DB_ERROR:%s", ierr.Error())
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
	appendExtraRoleIDs(s.DB, uid, roleIDs)
	s.rebuildUserEffectiveRoles(uid)
	wsDeptID := s.employeePrimaryWorkshopDeptID(empID)
	scopeType := "self"
	if wsDeptID > 0 {
		scopeType = "dept_workshop"
	}
	_, _ = s.DB.Exec(`INSERT INTO iam_user_data_scope(user_id, data_scope_type, dept_id) VALUES(?,?,?)`,
		uid, scopeType, nullIf0(wsDeptID))
	return login, pass, nil
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
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_tool_issue WHERE (employee_id=? OR employee_id IN (SELECT id FROM hr_employee WHERE id=?)) AND status IN ('open','pending_return','pending')`, empID, empID).Scan(&openTools)
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
			if security.IsFounderUser(s.DB, uid) {
				api.FailJSON(c, "SUPERUSER_PROTECTED")
				return true
			}
			claims := middleware.Claims(c)
			var by interface{}
			if claims != nil {
				by = claims.UserID
			}
			now := time.Now().Format("2006-01-02 15:04:05")
			_, _ = s.DB.Exec(`DELETE FROM iam_user_role WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`DELETE FROM iam_user_extra_role WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`DELETE FROM iam_admin_group_user WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`DELETE FROM iam_user_session WHERE user_id=?`, uid)
			_, _ = s.DB.Exec(`UPDATE iam_user SET status='frozen', freeze_reason=?, frozen_at=?, frozen_by=? WHERE id=?`,
				"offboard revoke", now, by, uid)
			security.InvalidateUserRBAC(uid)
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
	_ = db
	return
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
  created_at TEXT NOT NULL DEFAULT (NOW())
)`,
		`CREATE TABLE IF NOT EXISTS hr_offboard (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  revoke_permission INTEGER NOT NULL DEFAULT 1,
  reason TEXT,
  created_at TEXT NOT NULL DEFAULT (NOW())
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
		`ALTER TABLE hr_employee ADD COLUMN id_card_no TEXT`,
		`ALTER TABLE hr_employee ADD COLUMN badge_code TEXT`,
	}
	for _, q := range stmts {
		_, _ = db.Exec(q)
	}
	// 按员工类型默认角色模板
	for _, t := range EmpTypes {
		_, _ = db.Exec(`INSERT INTO iam_onboard_role_template(emp_type, role_id)
			SELECT ?, id FROM iam_role WHERE code=? LIMIT 1`, t.Code, t.RoleCode)
	}
}

// EnsureHRPermSchema is a no-op: schema owned by migrations/erp.
func EnsureHRPermSchema(db *sql.DB) {
	_ = db
}
