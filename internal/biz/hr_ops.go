package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

// EnsureHROpsSchema creates attendance / leave / performance / visit / memo tables.
func EnsureHROpsSchema(db *sql.DB) {
	ensureHROpsTables(db)
}

func ensureHROpsTables(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS hr_shift (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  workshop_id INTEGER,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_attendance_rule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  shift_id INTEGER,
  late_minutes INTEGER NOT NULL DEFAULT 0,
  early_minutes INTEGER NOT NULL DEFAULT 0,
  rule_json TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_attendance_record (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  check_in_at TEXT,
  check_out_at TEXT,
  shift_id INTEGER,
  source TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(employee_id, biz_date)
)`,
		`CREATE TABLE IF NOT EXISTS hr_leave_request (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  employee_id INTEGER NOT NULL,
  leave_type TEXT NOT NULL,
  start_at TEXT NOT NULL,
  end_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_overtime_patch (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  employee_id INTEGER NOT NULL,
  biz_type TEXT NOT NULL,
  biz_date TEXT NOT NULL,
  minutes INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_attendance_month_stat (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  year INTEGER NOT NULL,
  month INTEGER NOT NULL,
  work_days REAL NOT NULL DEFAULT 0,
  late_times INTEGER NOT NULL DEFAULT 0,
  ot_hours REAL NOT NULL DEFAULT 0,
  leave_days REAL NOT NULL DEFAULT 0,
  UNIQUE(employee_id, year, month)
)`,
		`CREATE TABLE IF NOT EXISTS hr_performance_scheme (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  scheme_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_performance_result (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scheme_id INTEGER NOT NULL,
  employee_id INTEGER NOT NULL,
  period TEXT NOT NULL,
  score REAL,
  amount REAL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_attendance_perf_summary (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  period TEXT NOT NULL,
  attendance_score REAL,
  perf_score REAL,
  summary_json TEXT,
  UNIQUE(employee_id, period)
)`,
		`CREATE TABLE IF NOT EXISTS hr_visit_record (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  customer_id INTEGER,
  visit_at TEXT NOT NULL,
  content TEXT,
  location TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_memo (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  content TEXT,
  biz_date TEXT,
  scope_type TEXT NOT NULL DEFAULT 'hr',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS hr_employee_journal (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`ALTER TABLE hr_offboard ADD COLUMN offboard_date TEXT`,
	}
	for _, q := range stmts {
		_, _ = db.Exec(q)
	}
	// seed default shift + rule if empty
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM hr_shift`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO hr_shift(code, name, start_time, end_time, workshop_id, status) VALUES('S1','白班','08:00','17:00',1,'active')`)
		_, _ = db.Exec(`INSERT INTO hr_shift(code, name, start_time, end_time, workshop_id, status) VALUES('S2','夜班','20:00','05:00',1,'active')`)
	}
	_ = db.QueryRow(`SELECT COUNT(1) FROM hr_attendance_rule`).Scan(&n)
	if n == 0 {
		var sid int64
		_ = db.QueryRow(`SELECT id FROM hr_shift ORDER BY id LIMIT 1`).Scan(&sid)
		_, _ = db.Exec(`INSERT INTO hr_attendance_rule(name, shift_id, late_minutes, early_minutes, status) VALUES('默认考勤规则',?,10,10,'active')`, sid)
	}
}

// handleHROps dispatches remaining HR modules (shifts, attendance, leave, etc.).
func (s *Services) handleHROps(c *gin.Context, method, openapiPath, action string) bool {
	ensureHROpsTables(s.DB)
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/hr/shifts"):
		return s.handleShifts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/attendance/rules"):
		return s.handleAttendanceRules(c, method, action)
	case strings.Contains(openapiPath, "/attendance/records"):
		return s.handleAttendanceRecords(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/leave-requests"):
		return s.handleLeaveRequests(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/overtime-patches"):
		return s.handleOvertimePatches(c, method, action, openapiPath)
	case strings.Contains(openapiPath, "/attendance/month-stats"):
		return s.handleMonthStats(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/performance/schemes"):
		return s.handlePerfSchemes(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/performance/results"):
		return s.handlePerfResults(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/attendance-perf-summaries"):
		return s.handleAttPerfSummaries(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/visits"):
		return s.handleVisits(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/memos"):
		return s.handleMemos(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/hr/employee-journals"):
		return s.handleJournals(c, method, action)
	}
	return false
}

func (s *Services) handleShifts(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, code, name, start_time, end_time, COALESCE(workshop_id,0), status FROM hr_shift ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, wid int64
			var code, name, st, et, status string
			_ = rows.Scan(&id, &code, &name, &st, &et, &wid, &status)
			list = append(list, gin.H{"id": id, "code": code, "name": name, "start_time": st, "end_time": et, "workshop_id": wid, "status": status})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		code, name := strOr(body["code"]), strOr(body["name"])
		if code == "" || name == "" {
			api.FailJSON(c, "CODE_NAME_REQUIRED")
			return true
		}
		st := strOrDef(body["start_time"], "08:00")
		et := strOrDef(body["end_time"], "17:00")
		wid, _ := asInt64(body["workshop_id"])
		res, err := s.DB.Exec(`INSERT INTO hr_shift(code, name, start_time, end_time, workshop_id, status) VALUES(?,?,?,?,?,'active')`,
			code, name, st, et, nullIf0(wid))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "start_time": st, "end_time": et, "workshop_id": wid, "status": "active"})
		return true
	case "get":
		id := paramID(c)
		var code, name, st, et, status string
		var wid int64
		err := s.DB.QueryRow(`SELECT code, name, start_time, end_time, COALESCE(workshop_id,0), status FROM hr_shift WHERE id=?`, id).
			Scan(&code, &name, &st, &et, &wid, &status)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "start_time": st, "end_time": et, "workshop_id": wid, "status": status})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE hr_shift SET name=COALESCE(NULLIF(?,''),name), start_time=COALESCE(NULLIF(?,''),start_time),
			end_time=COALESCE(NULLIF(?,''),end_time), workshop_id=COALESCE(?,workshop_id), status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			strOr(body["name"]), strOr(body["start_time"]), strOr(body["end_time"]), nullIf0(hrInt(body["workshop_id"])), strOr(body["status"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE hr_shift SET status='inactive' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "inactive"})
		return true
	}
	return true
}

func hrInt(v interface{}) int64 {
	n, _ := asInt64(v)
	return n
}

func (s *Services) handleAttendanceRules(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, name, COALESCE(shift_id,0), late_minutes, early_minutes, COALESCE(rule_json,''), status FROM hr_attendance_rule ORDER BY id`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, sid int64
				var name, rj, status string
				var late, early int
				_ = rows.Scan(&id, &name, &sid, &late, &early, &rj, &status)
				list = append(list, gin.H{"id": id, "name": name, "shift_id": sid, "late_minutes": late, "early_minutes": early, "rule_json": rj, "status": status})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		sid, _ := asInt64(body["shift_id"])
		late, _ := asInt64(body["late_minutes"])
		early, _ := asInt64(body["early_minutes"])
		res, err := s.DB.Exec(`INSERT INTO hr_attendance_rule(name, shift_id, late_minutes, early_minutes, rule_json, status) VALUES(?,?,?,?,?,'active')`,
			name, nullIf0(sid), late, early, strOrDef(body["rule_json"], "{}"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "name": name, "shift_id": sid, "late_minutes": late, "early_minutes": early, "status": "active"})
		return true
	case "get":
		id := paramID(c)
		var name, rj, status string
		var sid int64
		var late, early int
		err := s.DB.QueryRow(`SELECT name, COALESCE(shift_id,0), late_minutes, early_minutes, COALESCE(rule_json,''), status FROM hr_attendance_rule WHERE id=?`, id).
			Scan(&name, &sid, &late, &early, &rj, &status)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "name": name, "shift_id": sid, "late_minutes": late, "early_minutes": early, "rule_json": rj, "status": status})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE hr_attendance_rule SET name=COALESCE(NULLIF(?,''),name), shift_id=COALESCE(?,shift_id),
			late_minutes=COALESCE(?,late_minutes), early_minutes=COALESCE(?,early_minutes), status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			strOr(body["name"]), nullIf0(hrInt(body["shift_id"])), hrInt(body["late_minutes"]), hrInt(body["early_minutes"]), strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE hr_attendance_rule SET status='inactive' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	return true
}

func (s *Services) handleAttendanceRecords(c *gin.Context, method, action, path string) bool {
	if strings.HasSuffix(path, "/punch") || action == "action:punch" {
		return s.punchAttendance(c, "punch")
	}
	if strings.HasSuffix(path, "/patch") || action == "action:patch" {
		return s.punchAttendance(c, "patch")
	}
	switch action {
	case "list":
		empID := c.Query("employee_id")
		from, to := c.Query("from"), c.Query("to")
		q := `SELECT id, employee_id, biz_date, COALESCE(check_in_at,''), COALESCE(check_out_at,''), COALESCE(shift_id,0), COALESCE(source,'')
			FROM hr_attendance_record WHERE 1=1`
		args := []interface{}{}
		if empID != "" {
			q += ` AND employee_id=?`
			args = append(args, empID)
		}
		if from != "" {
			q += ` AND biz_date>=?`
			args = append(args, from)
		}
		if to != "" {
			q += ` AND biz_date<=?`
			args = append(args, to)
		}
		q += ` ORDER BY biz_date DESC, id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, eid, sid int64
			var date, cin, cout, src string
			_ = rows.Scan(&id, &eid, &date, &cin, &cout, &sid, &src)
			list = append(list, gin.H{"id": id, "employee_id": eid, "biz_date": date, "check_in_at": cin, "check_out_at": cout, "shift_id": sid, "source": src})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		eid, _ := asInt64(body["employee_id"])
		date := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		if eid == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		sid, _ := asInt64(body["shift_id"])
		cin, cout := strOr(body["check_in_at"]), strOr(body["check_out_at"])
		src := strOrDef(body["source"], "manual")
		res, err := s.DB.Exec(`INSERT INTO hr_attendance_record(employee_id, biz_date, check_in_at, check_out_at, shift_id, source)
			VALUES(?,?,?,?,?,?) ON CONFLICT(employee_id, biz_date) DO UPDATE SET
			check_in_at=COALESCE(NULLIF(excluded.check_in_at,''),hr_attendance_record.check_in_at),
			check_out_at=COALESCE(NULLIF(excluded.check_out_at,''),hr_attendance_record.check_out_at),
			shift_id=COALESCE(excluded.shift_id,hr_attendance_record.shift_id),
			source=excluded.source`,
			eid, date, nullStr(cin), nullStr(cout), nullIf0(sid), src)
		if err != nil {
			// fallback without upsert for older sqlite
			res, err = s.DB.Exec(`INSERT OR REPLACE INTO hr_attendance_record(employee_id, biz_date, check_in_at, check_out_at, shift_id, source) VALUES(?,?,?,?,?,?)`,
				eid, date, nullStr(cin), nullStr(cout), nullIf0(sid), src)
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "employee_id": eid, "biz_date": date, "source": src})
		return true
	}
	return true
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (s *Services) punchAttendance(c *gin.Context, source string) bool {
	body := bindBody(c)
	eid, _ := asInt64(body["employee_id"])
	if eid == 0 {
		if claims := middleware.Claims(c); claims != nil {
			_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, claims.UserID).Scan(&eid)
		}
	}
	if eid == 0 {
		api.FailJSON(c, "EMPLOYEE_REQUIRED")
		return true
	}
	date := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	now := time.Now().Format("2006-01-02 15:04:05")
	punchType := strOrDef(body["punch_type"], "in") // in/out
	var exists int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_attendance_record WHERE employee_id=? AND biz_date=?`, eid, date).Scan(&exists)
	if exists == 0 {
		cin, cout := "", ""
		if punchType == "out" {
			cout = now
		} else {
			cin = now
		}
		_, err := s.DB.Exec(`INSERT INTO hr_attendance_record(employee_id, biz_date, check_in_at, check_out_at, source) VALUES(?,?,?,?,?)`,
			eid, date, nullStr(cin), nullStr(cout), source)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
	} else if punchType == "out" {
		_, _ = s.DB.Exec(`UPDATE hr_attendance_record SET check_out_at=?, source=? WHERE employee_id=? AND biz_date=?`, now, source, eid, date)
	} else {
		_, _ = s.DB.Exec(`UPDATE hr_attendance_record SET check_in_at=COALESCE(check_in_at,?), source=? WHERE employee_id=? AND biz_date=?`, now, source, eid, date)
	}
	api.OK(c, gin.H{"employee_id": eid, "biz_date": date, "punch_type": punchType, "at": now, "source": source})
	return true
}

func (s *Services) handleLeaveRequests(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, doc_no, employee_id, leave_type, start_at, end_at, status, COALESCE(remark,''), created_at FROM hr_leave_request ORDER BY id DESC`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, eid int64
				var no, typ, start, end, st, remark, created string
				_ = rows.Scan(&id, &no, &eid, &typ, &start, &end, &st, &remark, &created)
				list = append(list, gin.H{"id": id, "doc_no": no, "employee_id": eid, "leave_type": typ, "start_at": start, "end_at": end, "status": st, "remark": remark, "created_at": created})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		eid, _ := asInt64(body["employee_id"])
		if eid == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		typ := strOrDef(body["leave_type"], "annual")
		start, end := strOr(body["start_at"]), strOr(body["end_at"])
		if start == "" || end == "" {
			api.FailJSON(c, "TIME_REQUIRED")
			return true
		}
		docNo := strOr(body["doc_no"])
		if docNo == "" {
			docNo = fmt.Sprintf("LV%s%d", time.Now().Format("20060102"), time.Now().Unix()%100000)
		}
		res, err := s.DB.Exec(`INSERT INTO hr_leave_request(doc_no, employee_id, leave_type, start_at, end_at, status, remark) VALUES(?,?,?,?,?,'pending',?)`,
			docNo, eid, typ, start, end, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "employee_id": eid, "leave_type": typ, "status": "pending"})
		return true
	case "get":
		id := paramID(c)
		var eid int64
		var no, typ, start, end, st, remark, created string
		err := s.DB.QueryRow(`SELECT doc_no, employee_id, leave_type, start_at, end_at, status, COALESCE(remark,''), created_at FROM hr_leave_request WHERE id=?`, id).
			Scan(&no, &eid, &typ, &start, &end, &st, &remark, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "doc_no": no, "employee_id": eid, "leave_type": typ, "start_at": start, "end_at": end, "status": st, "remark": remark, "created_at": created})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE hr_leave_request SET leave_type=COALESCE(NULLIF(?,''),leave_type), start_at=COALESCE(NULLIF(?,''),start_at),
			end_at=COALESCE(NULLIF(?,''),end_at), status=COALESCE(NULLIF(?,''),status), remark=COALESCE(?,remark) WHERE id=? AND status IN ('draft','pending')`,
			strOr(body["leave_type"]), strOr(body["start_at"]), strOr(body["end_at"]), strOr(body["status"]), strOr(body["remark"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case "action:cancel":
		id := paramID(c)
		_, err := s.DB.Exec(`UPDATE hr_leave_request SET status='cancelled' WHERE id=? AND status IN ('draft','pending')`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		api.OK(c, gin.H{"id": id, "status": "cancelled"})
		return true
	}
	return true
}

func (s *Services) handleOvertimePatches(c *gin.Context, method, action, path string) bool {
	if strings.HasSuffix(path, "/stats") || (action == "list" && strings.Contains(path, "/stats")) || (method == "GET" && strings.Contains(c.Request.URL.Path, "/stats")) {
		return s.overtimeStats(c)
	}
	// also catch when gin path is /overtime-patches/stats with action list
	if strings.Contains(path, "overtime-patches/stats") {
		return s.overtimeStats(c)
	}
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, doc_no, employee_id, biz_type, biz_date, minutes, status, COALESCE(remark,''), created_at FROM hr_overtime_patch ORDER BY id DESC`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, eid int64
				var no, typ, date, st, remark, created string
				var minutes int
				_ = rows.Scan(&id, &no, &eid, &typ, &date, &minutes, &st, &remark, &created)
				list = append(list, gin.H{"id": id, "doc_no": no, "employee_id": eid, "biz_type": typ, "biz_date": date, "minutes": minutes, "status": st, "remark": remark, "created_at": created})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		eid, _ := asInt64(body["employee_id"])
		if eid == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		bizType := strOrDef(body["biz_type"], "overtime")
		date := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		minutes, _ := asInt64(body["minutes"])
		docNo := strOr(body["doc_no"])
		if docNo == "" {
			docNo = fmt.Sprintf("OT%s%d", time.Now().Format("20060102"), time.Now().Unix()%100000)
		}
		res, err := s.DB.Exec(`INSERT INTO hr_overtime_patch(doc_no, employee_id, biz_type, biz_date, minutes, status, remark) VALUES(?,?,?,?,?,'pending',?)`,
			docNo, eid, bizType, date, minutes, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "employee_id": eid, "biz_type": bizType, "minutes": minutes, "status": "pending"})
		return true
	}
	return true
}

func (s *Services) overtimeStats(c *gin.Context) bool {
	from, to := c.Query("from"), c.Query("to")
	q := `SELECT biz_type, COUNT(1), COALESCE(SUM(minutes),0) FROM hr_overtime_patch WHERE status!='cancelled'`
	args := []interface{}{}
	if from != "" {
		q += ` AND biz_date>=?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND biz_date<=?`
		args = append(args, to)
	}
	q += ` GROUP BY biz_type`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	var totalMin int64
	for rows.Next() {
		var typ string
		var cnt, mins int64
		_ = rows.Scan(&typ, &cnt, &mins)
		totalMin += mins
		list = append(list, gin.H{"biz_type": typ, "count": cnt, "minutes": mins, "hours": float64(mins) / 60.0})
	}
	api.OK(c, gin.H{"list": list, "total_minutes": totalMin, "total_hours": float64(totalMin) / 60.0})
	return true
}

func (s *Services) handleMonthStats(c *gin.Context, method, action, path string) bool {
	if strings.Contains(path, "/recalc") || action == "action:recalc" || method == "POST" && (action == "create" && strings.Contains(path, "recalc") || strings.Contains(c.Request.URL.Path, "recalc")) {
		return s.recalcMonthStats(c)
	}
	year := c.Query("year")
	month := c.Query("month")
	q := `SELECT id, employee_id, year, month, work_days, late_times, ot_hours, leave_days FROM hr_attendance_month_stat WHERE 1=1`
	args := []interface{}{}
	if year != "" {
		q += ` AND year=?`
		args = append(args, year)
	}
	if month != "" {
		q += ` AND month=?`
		args = append(args, month)
	}
	q += ` ORDER BY year DESC, month DESC, employee_id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eid int64
		var y, m, late int
		var work, ot, leave float64
		_ = rows.Scan(&id, &eid, &y, &m, &work, &late, &ot, &leave)
		list = append(list, gin.H{"id": id, "employee_id": eid, "year": y, "month": m, "work_days": work, "late_times": late, "ot_hours": ot, "leave_days": leave})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) recalcMonthStats(c *gin.Context) bool {
	body := bindBody(c)
	year, _ := asInt64(body["year"])
	month, _ := asInt64(body["month"])
	if year == 0 || month == 0 {
		now := time.Now()
		year, month = int64(now.Year()), int64(now.Month())
	}
	prefix := fmt.Sprintf("%04d-%02d", year, month)
	empRows, err := s.DB.Query(`SELECT id FROM hr_employee WHERE COALESCE(is_deleted,0)=0 AND status IN ('active','pending','left')`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	eids := []int64{}
	for empRows.Next() {
		var eid int64
		_ = empRows.Scan(&eid)
		eids = append(eids, eid)
	}
	empRows.Close()
	updated := 0
	for _, eid := range eids {
		var workDays, lateTimes int
		var otHours, leaveDays float64
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_attendance_record WHERE employee_id=? AND biz_date LIKE ? AND check_in_at IS NOT NULL AND check_in_at!=''`,
			eid, prefix+"%").Scan(&workDays)
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_attendance_record WHERE employee_id=? AND biz_date LIKE ? AND check_in_at > (biz_date || ' 08:10:00')`,
			eid, prefix+"%").Scan(&lateTimes)
		var otMin int64
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(minutes),0) FROM hr_overtime_patch WHERE employee_id=? AND biz_type='overtime' AND biz_date LIKE ? AND status!='cancelled'`,
			eid, prefix+"%").Scan(&otMin)
		otHours = float64(otMin) / 60.0
		_ = s.DB.QueryRow(`SELECT COUNT(1)*1.0 FROM hr_leave_request WHERE employee_id=? AND status IN ('pending','approved') AND start_at LIKE ?`,
			eid, prefix+"%").Scan(&leaveDays)
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_attendance_month_stat WHERE employee_id=? AND year=? AND month=?`, eid, year, month).Scan(&n)
		var execErr error
		if n == 0 {
			_, execErr = s.DB.Exec(`INSERT INTO hr_attendance_month_stat(employee_id, year, month, work_days, late_times, ot_hours, leave_days) VALUES(?,?,?,?,?,?,?)`,
				eid, year, month, workDays, lateTimes, otHours, leaveDays)
		} else {
			_, execErr = s.DB.Exec(`UPDATE hr_attendance_month_stat SET work_days=?, late_times=?, ot_hours=?, leave_days=? WHERE employee_id=? AND year=? AND month=?`,
				workDays, lateTimes, otHours, leaveDays, eid, year, month)
		}
		if execErr != nil {
			api.FailJSON(c, "DB_ERROR:"+execErr.Error())
			return true
		}
		updated++
	}
	api.OK(c, gin.H{"year": year, "month": month, "updated": updated})
	return true
}

func (s *Services) handlePerfSchemes(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, name, COALESCE(scheme_json,'{}'), status, created_at FROM hr_performance_scheme ORDER BY id DESC`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var name, sj, st, created string
				_ = rows.Scan(&id, &name, &sj, &st, &created)
				list = append(list, gin.H{"id": id, "name": name, "scheme_json": sj, "status": st, "created_at": created})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		sj := strOrDef(body["scheme_json"], "{}")
		if m, ok := body["scheme_json"].(map[string]interface{}); ok {
			sj = jsonify(m)
		}
		res, err := s.DB.Exec(`INSERT INTO hr_performance_scheme(name, scheme_json, status) VALUES(?,?,'active')`, name, sj)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "name": name, "status": "active"})
		return true
	case "get":
		id := paramID(c)
		var name, sj, st, created string
		err := s.DB.QueryRow(`SELECT name, COALESCE(scheme_json,'{}'), status, created_at FROM hr_performance_scheme WHERE id=?`, id).Scan(&name, &sj, &st, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "name": name, "scheme_json": sj, "status": st, "created_at": created})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		sj := strOr(body["scheme_json"])
		if m, ok := body["scheme_json"].(map[string]interface{}); ok {
			sj = jsonify(m)
		}
		_, _ = s.DB.Exec(`UPDATE hr_performance_scheme SET name=COALESCE(NULLIF(?,''),name), scheme_json=COALESCE(NULLIF(?,''),scheme_json), status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			strOr(body["name"]), sj, strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE hr_performance_scheme SET status='inactive' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	return true
}

func (s *Services) handlePerfResults(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		period := c.Query("period")
		q := `SELECT id, scheme_id, employee_id, period, COALESCE(score,0), COALESCE(amount,0), created_at FROM hr_performance_result WHERE 1=1`
		args := []interface{}{}
		if period != "" {
			q += ` AND period=?`
			args = append(args, period)
		}
		q += ` ORDER BY id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, sid, eid int64
			var period, created string
			var score, amount float64
			_ = rows.Scan(&id, &sid, &eid, &period, &score, &amount, &created)
			list = append(list, gin.H{"id": id, "scheme_id": sid, "employee_id": eid, "period": period, "score": score, "amount": amount, "created_at": created})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		sid, _ := asInt64(body["scheme_id"])
		eid, _ := asInt64(body["employee_id"])
		period := strOr(body["period"])
		if sid == 0 || eid == 0 || period == "" {
			api.FailJSON(c, "SCHEME_EMP_PERIOD_REQUIRED")
			return true
		}
		score, _ := body["score"].(float64)
		if score == 0 {
			if v, ok := asInt64(body["score"]); ok {
				score = float64(v)
			}
		}
		amount, _ := body["amount"].(float64)
		res, err := s.DB.Exec(`INSERT INTO hr_performance_result(scheme_id, employee_id, period, score, amount) VALUES(?,?,?,?,?)`,
			sid, eid, period, score, amount)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		// upsert attendance-perf summary
		_, _ = s.DB.Exec(`INSERT INTO hr_attendance_perf_summary(employee_id, period, perf_score, attendance_score, summary_json)
			VALUES(?,?,?,0,'{}') ON CONFLICT(employee_id, period) DO UPDATE SET perf_score=excluded.perf_score`, eid, period, score)
		api.OK(c, gin.H{"id": id, "scheme_id": sid, "employee_id": eid, "period": period, "score": score, "amount": amount})
		return true
	}
	return true
}

func (s *Services) handleAttPerfSummaries(c *gin.Context, method, action string) bool {
	period := c.Query("period")
	q := `SELECT id, employee_id, period, COALESCE(attendance_score,0), COALESCE(perf_score,0), COALESCE(summary_json,'') FROM hr_attendance_perf_summary WHERE 1=1`
	args := []interface{}{}
	if period != "" {
		q += ` AND period=?`
		args = append(args, period)
	}
	q += ` ORDER BY period DESC, employee_id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eid int64
		var period, sj string
		var ascore, pscore float64
		_ = rows.Scan(&id, &eid, &period, &ascore, &pscore, &sj)
		list = append(list, gin.H{"id": id, "employee_id": eid, "period": period, "attendance_score": ascore, "perf_score": pscore, "summary_json": sj})
	}
	// auto-fill from month stats if empty and period like 2026-08
	if len(list) == 0 && period != "" && len(period) >= 7 {
		var y, m int
		_, _ = fmt.Sscanf(period, "%d-%d", &y, &m)
		if y > 0 && m > 0 {
			srows, _ := s.DB.Query(`SELECT employee_id, work_days, late_times, ot_hours, leave_days FROM hr_attendance_month_stat WHERE year=? AND month=?`, y, m)
			if srows != nil {
				defer srows.Close()
				for srows.Next() {
					var eid int64
					var work, ot, leave float64
					var late int
					_ = srows.Scan(&eid, &work, &late, &ot, &leave)
					attScore := work*2 - float64(late) + ot
					_, _ = s.DB.Exec(`INSERT OR IGNORE INTO hr_attendance_perf_summary(employee_id, period, attendance_score, perf_score, summary_json) VALUES(?,?,?,0,?)`,
						eid, period, attScore, fmt.Sprintf(`{"work_days":%v,"late":%d,"ot":%v,"leave":%v}`, work, late, ot, leave))
					list = append(list, gin.H{"employee_id": eid, "period": period, "attendance_score": attScore, "perf_score": 0})
				}
			}
		}
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) handleVisits(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, employee_id, COALESCE(customer_id,0), visit_at, COALESCE(content,''), COALESCE(location,''), created_at FROM hr_visit_record ORDER BY visit_at DESC`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, eid, cid int64
				var at, content, loc, created string
				_ = rows.Scan(&id, &eid, &cid, &at, &content, &loc, &created)
				list = append(list, gin.H{"id": id, "employee_id": eid, "customer_id": cid, "visit_at": at, "content": content, "location": loc, "created_at": created})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		eid, _ := asInt64(body["employee_id"])
		if eid == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		cid, _ := asInt64(body["customer_id"])
		at := strOrDef(body["visit_at"], time.Now().Format("2006-01-02 15:04:05"))
		res, err := s.DB.Exec(`INSERT INTO hr_visit_record(employee_id, customer_id, visit_at, content, location) VALUES(?,?,?,?,?)`,
			eid, nullIf0(cid), at, strOr(body["content"]), strOr(body["location"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "employee_id": eid, "visit_at": at})
		return true
	case "get":
		id := paramID(c)
		var eid, cid int64
		var at, content, loc, created string
		err := s.DB.QueryRow(`SELECT employee_id, COALESCE(customer_id,0), visit_at, COALESCE(content,''), COALESCE(location,''), created_at FROM hr_visit_record WHERE id=?`, id).
			Scan(&eid, &cid, &at, &content, &loc, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "employee_id": eid, "customer_id": cid, "visit_at": at, "content": content, "location": loc, "created_at": created})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE hr_visit_record SET content=COALESCE(?,content), location=COALESCE(?,location), visit_at=COALESCE(NULLIF(?,''),visit_at) WHERE id=?`,
			strOr(body["content"]), strOr(body["location"]), strOr(body["visit_at"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	return true
}

func (s *Services) handleMemos(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, owner_user_id, title, COALESCE(content,''), COALESCE(biz_date,''), scope_type, created_at FROM hr_memo WHERE scope_type='hr' ORDER BY id DESC`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, oid int64
				var title, content, date, scope, created string
				_ = rows.Scan(&id, &oid, &title, &content, &date, &scope, &created)
				list = append(list, gin.H{"id": id, "owner_user_id": oid, "title": title, "content": content, "biz_date": date, "scope_type": scope, "created_at": created})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		title := strOr(body["title"])
		if title == "" {
			api.FailJSON(c, "TITLE_REQUIRED")
			return true
		}
		oid, _ := asInt64(body["owner_user_id"])
		if oid == 0 {
			if claims := middleware.Claims(c); claims != nil {
				oid = claims.UserID
			}
		}
		res, err := s.DB.Exec(`INSERT INTO hr_memo(owner_user_id, title, content, biz_date, scope_type) VALUES(?,?,?,?,?)`,
			oid, title, strOr(body["content"]), strOr(body["biz_date"]), strOrDef(body["scope_type"], "hr"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "title": title, "owner_user_id": oid})
		return true
	case "get":
		id := paramID(c)
		var oid int64
		var title, content, date, scope, created string
		err := s.DB.QueryRow(`SELECT owner_user_id, title, COALESCE(content,''), COALESCE(biz_date,''), scope_type, created_at FROM hr_memo WHERE id=?`, id).
			Scan(&oid, &title, &content, &date, &scope, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "owner_user_id": oid, "title": title, "content": content, "biz_date": date, "scope_type": scope, "created_at": created})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE hr_memo SET title=COALESCE(NULLIF(?,''),title), content=COALESCE(?,content), biz_date=COALESCE(NULLIF(?,''),biz_date) WHERE id=?`,
			strOr(body["title"]), strOr(body["content"]), strOr(body["biz_date"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`DELETE FROM hr_memo WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "deleted": true})
		return true
	}
	return true
}

func (s *Services) handleJournals(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		eid := c.Query("employee_id")
		q := `SELECT id, employee_id, biz_date, content, created_at FROM hr_employee_journal WHERE 1=1`
		args := []interface{}{}
		if eid != "" {
			q += ` AND employee_id=?`
			args = append(args, eid)
		}
		q += ` ORDER BY biz_date DESC, id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, emp int64
			var date, content, created string
			_ = rows.Scan(&id, &emp, &date, &content, &created)
			list = append(list, gin.H{"id": id, "employee_id": emp, "biz_date": date, "content": content, "created_at": created})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		eid, _ := asInt64(body["employee_id"])
		content := strOr(body["content"])
		if eid == 0 || content == "" {
			api.FailJSON(c, "EMP_CONTENT_REQUIRED")
			return true
		}
		date := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		res, err := s.DB.Exec(`INSERT INTO hr_employee_journal(employee_id, biz_date, content) VALUES(?,?,?)`, eid, date, content)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "employee_id": eid, "biz_date": date})
		return true
	case "get":
		id := paramID(c)
		var eid int64
		var date, content, created string
		err := s.DB.QueryRow(`SELECT employee_id, biz_date, content, created_at FROM hr_employee_journal WHERE id=?`, id).Scan(&eid, &date, &content, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "employee_id": eid, "biz_date": date, "content": content, "created_at": created})
		return true
	}
	return true
}
