package biz

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

const employeeJobTitleJoin = `LEFT JOIN hr_job_title j ON j.id=e.job_title_id AND COALESCE(j.is_deleted,0)=0`

type jobTitleSeed struct {
	Code, Name, EmpType string
	Sort                int
}

var factoryJobTitleSeeds = []jobTitleSeed{
	{Code: "JT-SYS-ADMIN", Name: "系统管理员", EmpType: "", Sort: 1},
	{Code: "JT-BOSS", Name: "总经理", EmpType: "office", Sort: 2},
	{Code: "JT-PURCHASE", Name: "采购员", EmpType: "office", Sort: 3},
	{Code: "JT-QC", Name: "质检员", EmpType: "office", Sort: 4},
	{Code: "JT-WH", Name: "仓管员", EmpType: "warehouse", Sort: 5},
	{Code: "JT-FOREMAN", Name: "车间主任", EmpType: "office", Sort: 6},
	{Code: "JT-PLANNER", Name: "生产计划员", EmpType: "office", Sort: 7},
	{Code: "JT-HR", Name: "人事专员", EmpType: "office", Sort: 8},
	{Code: "JT-PAYROLL", Name: "薪资员", EmpType: "office", Sort: 9},
	{Code: "JT-FINANCE", Name: "会计", EmpType: "office", Sort: 10},
	{Code: "JT-SALES", Name: "销售员", EmpType: "office", Sort: 11},
	{Code: "JT-PEEL", Name: "去皮工", EmpType: "piece", Sort: 20},
	{Code: "JT-PEEL-PC", Name: "去皮计件工", EmpType: "piece", Sort: 21},
	{Code: "JT-CORE", Name: "去芯工", EmpType: "piece", Sort: 22},
	{Code: "JT-DICE", Name: "切块工", EmpType: "piece", Sort: 23},
	{Code: "JT-RECEIVE", Name: "收货员", EmpType: "fixed", Sort: 30},
	{Code: "JT-RECEIVE-FX", Name: "收货固定工", EmpType: "fixed", Sort: 31},
	{Code: "JT-CUT", Name: "切断工", EmpType: "fixed", Sort: 32},
	{Code: "JT-WASH", Name: "清洗工", EmpType: "fixed", Sort: 33},
	{Code: "JT-TEMP", Name: "临时工", EmpType: "temp", Sort: 40},
}

var slugNonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func ensureJobTitleTable(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS hr_job_title (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL,
		name TEXT NOT NULL,
		emp_type TEXT NOT NULL DEFAULT '',
		sort_no INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		is_deleted INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS hr_job_title_code_uq ON hr_job_title(code) WHERE COALESCE(is_deleted,0)=0`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS hr_job_title_name_uq ON hr_job_title(name) WHERE COALESCE(is_deleted,0)=0`)
}

func ensureFactoryJobTitles(db *sql.DB) {
	if db == nil {
		return
	}
	ensureJobTitleTable(db)
	for _, s := range factoryJobTitleSeeds {
		ensureJobTitleRow(db, s.Name, s.EmpType, s.Code, s.Sort)
	}
}

func jobTitleCodeFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Sprintf("JT-%d", timeNowUnixNano()%1e9)
	}
	var b strings.Builder
	b.WriteString("JT-")
	for _, r := range name {
		if unicode.Is(unicode.Han, r) {
			b.WriteRune(r)
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	code := strings.Trim(slugNonAlnum.ReplaceAllString(b.String(), "-"), "-")
	if code == "JT" || len(code) < 4 {
		return fmt.Sprintf("JT-%d", timeNowUnixNano()%1e9)
	}
	if len(code) > 48 {
		code = code[:48]
	}
	return code
}

func timeNowUnixNano() int64 {
	return time.Now().UnixNano()
}

func ensureJobTitleRow(db *sql.DB, name, empType, code string, sortNo int) int64 {
	if db == nil {
		return 0
	}
	ensureJobTitleTable(db)
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	empType = strings.TrimSpace(empType)
	if code == "" {
		code = jobTitleCodeFromName(name)
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM hr_job_title WHERE name=? AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, name).Scan(&id)
	if id > 0 {
		_, _ = db.Exec(`UPDATE hr_job_title SET emp_type=CASE WHEN COALESCE(emp_type,'')='' THEN ? ELSE emp_type END,
			sort_no=CASE WHEN sort_no=0 THEN ? ELSE sort_no END, status='active', updated_at=NOW() WHERE id=?`,
			empType, sortNo, id)
		return id
	}
	_ = db.QueryRow(`SELECT id FROM hr_job_title WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&id)
	if id > 0 {
		return id
	}
	res, err := db.Exec(`INSERT INTO hr_job_title(code, name, emp_type, sort_no, status) VALUES(?,?,?,?, 'active')`,
		code, name, empType, sortNo)
	if err != nil {
		_ = db.QueryRow(`SELECT id FROM hr_job_title WHERE name=? AND COALESCE(is_deleted,0)=0`, name).Scan(&id)
		return id
	}
	id, _ = res.LastInsertId()
	return id
}

func jobTitleIDByName(db *sql.DB, name string) int64 {
	if db == nil || strings.TrimSpace(name) == "" {
		return 0
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM hr_job_title WHERE name=? AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, strings.TrimSpace(name)).Scan(&id)
	return id
}

func jobTitleIDByCode(db *sql.DB, code string) int64 {
	if db == nil || strings.TrimSpace(code) == "" {
		return 0
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM hr_job_title WHERE code=? AND COALESCE(is_deleted,0)=0`, strings.TrimSpace(code)).Scan(&id)
	return id
}

func (s *Services) validateJobTitleID(jobTitleID int64, empType string) string {
	if jobTitleID <= 0 {
		return ""
	}
	ensureJobTitleTable(s.DB)
	var status, jtEmpType string
	err := s.DB.QueryRow(`SELECT COALESCE(status,''), COALESCE(emp_type,'') FROM hr_job_title WHERE id=? AND COALESCE(is_deleted,0)=0`, jobTitleID).
		Scan(&status, &jtEmpType)
	if err != nil {
		return "JOB_TITLE_NOT_FOUND"
	}
	if status != "active" {
		return "JOB_TITLE_INACTIVE"
	}
	empType = strings.TrimSpace(empType)
	if empType != "" && jtEmpType != "" && jtEmpType != empType {
		return "JOB_TITLE_EMP_TYPE_MISMATCH"
	}
	return ""
}

func parseJobTitleIDFromBody(body map[string]interface{}, cur gin.H) int64 {
	if body != nil {
		if _, ok := body["job_title_id"]; ok {
			v, _ := asInt64(body["job_title_id"])
			return v
		}
	}
	if cur != nil {
		if v, ok := asInt64(cur["job_title_id"]); ok {
			return v
		}
	}
	return 0
}

func (s *Services) handleJobTitles(c *gin.Context, method, action, openapiPath string) bool {
	ensureJobTitleTable(s.DB)
	if strings.HasSuffix(openapiPath, "/ensure") && method == "POST" {
		return s.ensureJobTitleAPI(c)
	}
	switch action {
	case "list":
		if method != "GET" {
			api.FailJSON(c, "METHOD_NOT_ALLOWED")
			return true
		}
		list := s.listJobTitles(c.Query("emp_type"), c.Query("status"))
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		if method != "POST" {
			api.FailJSON(c, "METHOD_NOT_ALLOWED")
			return true
		}
		row, msg := s.createJobTitleFromBody(bindBody(c))
		if msg != "" {
			api.FailJSON(c, msg)
			return true
		}
		api.OK(c, row)
		return true
	case "get":
		if method != "GET" {
			api.FailJSON(c, "METHOD_NOT_ALLOWED")
			return true
		}
		row := s.loadJobTitleMap(paramID(c))
		if row == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, row)
		return true
	case "update":
		if method != "PUT" {
			api.FailJSON(c, "METHOD_NOT_ALLOWED")
			return true
		}
		row, msg := s.updateJobTitleFromBody(paramID(c), bindBody(c))
		if msg != "" {
			api.FailJSON(c, msg)
			return true
		}
		api.OK(c, row)
		return true
	case "remove":
		if method != "DELETE" {
			api.FailJSON(c, "METHOD_NOT_ALLOWED")
			return true
		}
		if msg := s.removeJobTitle(paramID(c)); msg != "" {
			api.FailJSON(c, msg)
			return true
		}
		api.OK(c, gin.H{"id": paramID(c)})
		return true
	default:
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
}

func (s *Services) listJobTitles(empType, status string) []gin.H {
	q := `SELECT id, code, name, COALESCE(emp_type,''), sort_no, COALESCE(status,'active')
		FROM hr_job_title WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	statusFilter := strings.TrimSpace(status)
	switch statusFilter {
	case "all":
	case "inactive":
		q += ` AND status='inactive'`
	default:
		q += ` AND status='active'`
	}
	empType = strings.TrimSpace(empType)
	if empType != "" {
		q += ` AND (emp_type='' OR emp_type=?)`
		args = append(args, empType)
	}
	q += ` ORDER BY sort_no, id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, et, st string
		var sort int
		if rows.Scan(&id, &code, &name, &et, &sort, &st) != nil {
			continue
		}
		out = append(out, gin.H{"id": id, "code": code, "name": name, "emp_type": et, "sort_no": sort, "status": st})
	}
	return out
}

func (s *Services) loadJobTitleMap(id int64) gin.H {
	if id <= 0 {
		return nil
	}
	var code, name, et, st string
	var sort int
	err := s.DB.QueryRow(`SELECT code, name, COALESCE(emp_type,''), sort_no, COALESCE(status,'active')
		FROM hr_job_title WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&code, &name, &et, &sort, &st)
	if err != nil {
		return nil
	}
	return gin.H{"id": id, "code": code, "name": name, "emp_type": et, "sort_no": sort, "status": st}
}

func (s *Services) createJobTitleFromBody(body map[string]interface{}) (gin.H, string) {
	name := strings.TrimSpace(strOr(body["name"]))
	if name == "" {
		return nil, "NAME_REQUIRED"
	}
	empType := strings.TrimSpace(strOr(body["emp_type"]))
	code := strings.TrimSpace(strOr(body["code"]))
	if code == "" {
		code = jobTitleCodeFromName(name)
	}
	sortNo, _ := asInt64(body["sort_no"])
	id := ensureJobTitleRow(s.DB, name, empType, code, int(sortNo))
	if id <= 0 {
		return nil, "DB_ERROR"
	}
	if sortNo > 0 {
		_, _ = s.DB.Exec(`UPDATE hr_job_title SET sort_no=? WHERE id=?`, sortNo, id)
	}
	return s.loadJobTitleMap(id), ""
}

func (s *Services) updateJobTitleFromBody(id int64, body map[string]interface{}) (gin.H, string) {
	cur := s.loadJobTitleMap(id)
	if cur == nil {
		return nil, "NOT_FOUND"
	}
	name := strings.TrimSpace(strOrDef(body["name"], fmt.Sprint(cur["name"])))
	empType := strings.TrimSpace(strOrDef(body["emp_type"], fmt.Sprint(cur["emp_type"])))
	code := strings.TrimSpace(strOrDef(body["code"], fmt.Sprint(cur["code"])))
	status := strings.TrimSpace(strOrDef(body["status"], fmt.Sprint(cur["status"])))
	if status == "" {
		status = "active"
	}
	sortNo, ok := asInt64(body["sort_no"])
	if !ok {
		sortNo, _ = asInt64(cur["sort_no"])
	}
	var dup int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_job_title WHERE name=? AND id<>? AND COALESCE(is_deleted,0)=0`, name, id).Scan(&dup)
	if dup > 0 {
		return nil, "JOB_TITLE_NAME_DUPLICATE"
	}
	_, err := s.DB.Exec(`UPDATE hr_job_title SET code=?, name=?, emp_type=?, sort_no=?, status=?, updated_at=NOW() WHERE id=?`,
		code, name, empType, sortNo, status, id)
	if err != nil {
		return nil, "DB_ERROR"
	}
	return s.loadJobTitleMap(id), ""
}

func (s *Services) removeJobTitle(id int64) string {
	if id <= 0 {
		return "NOT_FOUND"
	}
	var cnt int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_employee WHERE job_title_id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&cnt)
	if cnt > 0 {
		_, _ = s.DB.Exec(`UPDATE hr_job_title SET status='inactive', updated_at=NOW() WHERE id=?`, id)
		return ""
	}
	_, _ = s.DB.Exec(`UPDATE hr_job_title SET is_deleted=1, status='inactive', updated_at=NOW() WHERE id=?`, id)
	return ""
}

func (s *Services) ensureJobTitleAPI(c *gin.Context) bool {
	body := bindBody(c)
	name := strings.TrimSpace(strOr(body["name"]))
	if name == "" {
		api.FailJSON(c, "NAME_REQUIRED")
		return true
	}
	empType := strings.TrimSpace(strOr(body["emp_type"]))
	id := ensureJobTitleRow(s.DB, name, empType, "", 0)
	if id <= 0 {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	row := s.loadJobTitleMap(id)
	api.OK(c, row)
	return true
}
