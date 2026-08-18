package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsurePayrollSchema is a no-op: schema owned by migrations/erp.
func EnsurePayrollSchema(db *sql.DB) {
	_ = db
	return
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pay_worker_profile (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL UNIQUE,
			pay_type TEXT NOT NULL DEFAULT 'piece',
			monthly_base REAL NOT NULL DEFAULT 0,
			bank_account TEXT,
			tax_no TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (NOW())
		)`,
		`CREATE TABLE IF NOT EXISTS pay_payroll_sheet (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_no TEXT NOT NULL UNIQUE,
			period_year INTEGER NOT NULL,
			period_month INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			workshop_dept_id INTEGER,
			calc_at TEXT,
			paid_at TEXT,
			remark TEXT,
			created_by INTEGER,
			created_at TEXT NOT NULL DEFAULT (NOW()),
			UNIQUE(period_year, period_month)
		)`,
		`CREATE TABLE IF NOT EXISTS pay_payroll_sheet_line (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sheet_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			emp_type TEXT,
			piece_amount REAL NOT NULL DEFAULT 0,
			attendance_amount REAL NOT NULL DEFAULT 0,
			commission_amount REAL NOT NULL DEFAULT 0,
			adjust_amount REAL NOT NULL DEFAULT 0,
			total_amount REAL NOT NULL DEFAULT 0,
			UNIQUE(sheet_id, employee_id)
		)`,
		`CREATE TABLE IF NOT EXISTS pay_payroll_adjust (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sheet_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			adjust_type TEXT NOT NULL,
			amount REAL NOT NULL,
			reason TEXT,
			created_at TEXT NOT NULL DEFAULT (NOW())
		)`,
		`CREATE TABLE IF NOT EXISTS pay_sales_commission_rule (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			rule_json TEXT NOT NULL DEFAULT '{}',
			effective_from TEXT NOT NULL,
			effective_to TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (NOW())
		)`,
		`CREATE TABLE IF NOT EXISTS pay_commission_calc (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rule_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			period TEXT NOT NULL,
			base_amount REAL NOT NULL DEFAULT 0,
			commission_amount REAL NOT NULL DEFAULT 0,
			source_doc_refs TEXT,
			created_at TEXT NOT NULL DEFAULT (NOW())
		)`,
		`CREATE TABLE IF NOT EXISTS pay_payroll_calc_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_no TEXT,
			period_ym TEXT,
			sheet_id INTEGER,
			status TEXT,
			summary_json TEXT,
			created_at TEXT NOT NULL DEFAULT (NOW())
		)`,
		`ALTER TABLE pay_worker_profile ADD COLUMN monthly_base REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pay_payroll_sheet ADD COLUMN workshop_dept_id INTEGER`,
		`ALTER TABLE pay_payroll_sheet ADD COLUMN paid_at TEXT`,
		`ALTER TABLE pay_payroll_sheet ADD COLUMN remark TEXT`,
		`ALTER TABLE pay_payroll_sheet_line ADD COLUMN emp_type TEXT`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil && !isIdempotentSchemaErr(err) {
			_ = err
		}
	}
	// seed default commission rule
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM pay_sales_commission_rule`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO pay_sales_commission_rule(name, rule_json, effective_from, status)
			VALUES('默认销售提成','{"rate":0.01,"base":"order_amount"}',date('now'),'active')`)
	}
}

// handlePayroll is the payroll domain dispatcher (real tables).
func (s *Services) handlePayroll(c *gin.Context, method, action, path string) bool {
	EnsurePayrollSchema(s.DB)
	switch {
	case strings.Contains(path, "work-records"):
		return s.handlePayrollWorkRecords(c)
	case strings.Contains(path, "wage-rates"):
		return s.handleWageRates(c, action)
	case strings.Contains(path, "worker-profiles"):
		return s.handleWorkerProfiles(c, action)
	case strings.Contains(path, "batch-generate"):
		return s.batchGeneratePayrollSheet(c)
	case strings.Contains(path, "/sheets"):
		return s.handlePayrollSheets(c, method, action, path)
	case strings.Contains(path, "calculations"):
		return s.handlePayrollCalculations(c, action)
	case strings.Contains(path, "commission-rules"):
		return s.handleCommissionRules(c, action)
	case strings.Contains(path, "commission-calcs"):
		return s.handleCommissionCalcs(c, method, action, path)
	}
	return false
}

func (s *Services) handleWageRates(c *gin.Context, action string) bool {
	switch action {
	case "list":
		showAll := c.Query("all") == "1" || strings.EqualFold(c.Query("all"), "true")
		where := `WHERE r.status='active'`
		if showAll {
			where = ``
		}
		rows, err := s.DB.Query(`SELECT r.id, r.process_id, COALESCE(p.code,''), COALESCE(p.name,''), r.rate,
			COALESCE(r.rate_unit,'kg'), r.effective_from, COALESCE(r.effective_to,''), r.status
			FROM pay_process_wage_rate r
			LEFT JOIN pd_process p ON p.id=r.process_id
			`+where+`
			ORDER BY r.process_id, r.id DESC`)
		if err != nil {
			// fallback without rate_unit / join
			q2 := `SELECT id, process_id, rate, effective_from, COALESCE(effective_to,''), status FROM pay_process_wage_rate`
			if !showAll {
				q2 += ` WHERE status='active'`
			}
			q2 += ` ORDER BY process_id, id DESC`
			rows, err = s.DB.Query(q2)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id, pid int64
				var rate float64
				var from, to, status string
				_ = rows.Scan(&id, &pid, &rate, &from, &to, &status)
				list = append(list, gin.H{"id": id, "process_id": pid, "rate": rate, "effective_from": from, "effective_to": to, "status": status})
			}
			api.OK(c, gin.H{"list": list, "total": len(list)})
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, pid int64
			var rate float64
			var pcode, pname, unit, from, to, status string
			_ = rows.Scan(&id, &pid, &pcode, &pname, &rate, &unit, &from, &to, &status)
			list = append(list, gin.H{
				"id": id, "process_id": pid, "process_code": pcode, "process_name": pname,
				"rate": rate, "rate_unit": unit, "effective_from": from, "effective_to": to, "status": status,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		pid, _ := asInt64(body["process_id"])
		rate, _ := asFloat(body["rate"])
		if pid == 0 {
			api.FailJSON(c, "PROCESS_REQUIRED")
			return true
		}
		from := strOrDef(body["effective_from"], time.Now().Format("2006-01-02"))
		unit := strOrDef(body["rate_unit"], "yuan/kg")
		// 同工序仅保留一条 active：新建前先停用旧费率
		_, _ = s.DB.Exec(`UPDATE pay_process_wage_rate SET status='inactive' WHERE process_id=? AND status='active'`, pid)
		res, err := s.DB.Exec(`INSERT INTO pay_process_wage_rate(process_id, rate, effective_from, status, rate_unit) VALUES(?,?,?,'active',?)`, pid, rate, from, unit)
		if err != nil {
			res, err = s.DB.Exec(`INSERT INTO pay_process_wage_rate(process_id, rate, effective_from, status) VALUES(?,?,?,'active')`, pid, rate, from)
		}
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "process_id": pid, "rate": rate, "rate_unit": unit, "status": "active"})
		return true
	case "get", "update", "delete":
		id := paramID(c)
		if action == "delete" {
			_, _ = s.DB.Exec(`UPDATE pay_process_wage_rate SET status='inactive' WHERE id=?`, id)
			api.OK(c, gin.H{"id": id, "status": "inactive"})
			return true
		}
		if action == "get" {
			var pid int64
			var rate float64
			var from, to, status string
			err := s.DB.QueryRow(`SELECT process_id, rate, effective_from, COALESCE(effective_to,''), status FROM pay_process_wage_rate WHERE id=?`, id).
				Scan(&pid, &rate, &from, &to, &status)
			if err != nil {
				api.FailJSON(c, "NOT_FOUND")
				return true
			}
			api.OK(c, gin.H{"id": id, "process_id": pid, "rate": rate, "effective_from": from, "effective_to": to, "status": status})
			return true
		}
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pay_process_wage_rate SET
			rate=COALESCE(?,rate),
			process_id=COALESCE(NULLIF(?,0),process_id),
			effective_from=COALESCE(NULLIF(?,''),effective_from),
			effective_to=COALESCE(?,effective_to),
			status=COALESCE(NULLIF(?,''),status)
			WHERE id=?`,
			payrollNullFloat(body["rate"]), asInt64Or0(body["process_id"]), strOr(body["effective_from"]),
			nullStr(strOr(body["effective_to"])), strOr(body["status"]), id)
		if unit := strOr(body["rate_unit"]); unit != "" {
			_, _ = s.DB.Exec(`UPDATE pay_process_wage_rate SET rate_unit=? WHERE id=?`, unit, id)
		}
		api.OK(c, gin.H{"id": id})
		return true
	}
	return true
}

func payrollNullFloat(v interface{}) interface{} {
	f, ok := asFloat(v)
	if !ok {
		return nil
	}
	return f
}

// payTypeFromEmpType 人事工种 → 工资档案计薪方式（与工人信息管理列表默认规则一致）。
func payTypeFromEmpType(empType string) string {
	switch strings.ToLower(strings.TrimSpace(empType)) {
	case "fixed", "office", "func":
		return "fixed"
	default:
		return "piece"
	}
}

// syncEmployeePayProfile 将人事侧银行卡/税号写入 pay_worker_profile，与财务「工人信息管理」同源。
func (s *Services) syncEmployeePayProfile(empID int64, body map[string]interface{}, empType string) {
	if empID <= 0 {
		return
	}
	payType := payTypeFromEmpType(empType)
	bank := strings.TrimSpace(strOr(body["bank_account"]))
	tax := strings.TrimSpace(strOr(body["tax_no"]))
	var exist int64
	_ = s.DB.QueryRow(`SELECT id FROM pay_worker_profile WHERE employee_id=?`, empID).Scan(&exist)
	if exist == 0 {
		_, _ = s.DB.Exec(`INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, tax_no, status)
			VALUES(?,?,0,?,?,'active')`, empID, payType, bank, tax)
		return
	}
	// 有传字段则覆盖（允许清空）；未传则只同步 pay_type
	_, hasBank := body["bank_account"]
	_, hasTax := body["tax_no"]
	if hasBank || hasTax {
		if hasBank && hasTax {
			_, _ = s.DB.Exec(`UPDATE pay_worker_profile SET pay_type=?, bank_account=?, tax_no=? WHERE employee_id=?`,
				payType, bank, tax, empID)
		} else if hasBank {
			_, _ = s.DB.Exec(`UPDATE pay_worker_profile SET pay_type=?, bank_account=? WHERE employee_id=?`, payType, bank, empID)
		} else {
			_, _ = s.DB.Exec(`UPDATE pay_worker_profile SET pay_type=?, tax_no=? WHERE employee_id=?`, payType, tax, empID)
		}
		return
	}
	_, _ = s.DB.Exec(`UPDATE pay_worker_profile SET pay_type=? WHERE employee_id=?`, payType, empID)
}

func (s *Services) handleWorkerProfiles(c *gin.Context, action string) bool {
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT COALESCE(p.id,0), e.id, e.emp_no, e.name, e.emp_type, e.status,
			COALESCE(p.pay_type, CASE WHEN e.emp_type IN ('fixed','office') THEN 'fixed' ELSE 'piece' END),
			COALESCE(p.monthly_base,0), COALESCE(p.bank_account,''), COALESCE(p.tax_no,''), COALESCE(p.status,'active')
			FROM hr_employee e
			LEFT JOIN pay_worker_profile p ON p.employee_id=e.id
			WHERE COALESCE(e.is_deleted,0)=0
			ORDER BY e.id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var pid, eid int64
			var empNo, name, empType, empStatus, payType, bank, tax, pstatus string
			var base float64
			_ = rows.Scan(&pid, &eid, &empNo, &name, &empType, &empStatus, &payType, &base, &bank, &tax, &pstatus)
			list = append(list, gin.H{
				"id": pid, "employee_id": eid, "emp_no": empNo, "name": name, "emp_type": empType, "emp_status": empStatus,
				"pay_type": payType, "monthly_base": base, "bank_account": bank, "tax_no": tax, "status": pstatus,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create", "update":
		body := bindBody(c)
		eid, _ := asInt64(body["employee_id"])
		if eid == 0 && action == "update" {
			// profile id -> employee
			id := paramID(c)
			_ = s.DB.QueryRow(`SELECT employee_id FROM pay_worker_profile WHERE id=?`, id).Scan(&eid)
		}
		if eid == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		payType := strOrDef(body["pay_type"], "piece")
		base, _ := asFloat(body["monthly_base"])
		bank, tax := strOr(body["bank_account"]), strOr(body["tax_no"])
		status := strOrDef(body["status"], "active")
		var exist int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pay_worker_profile WHERE employee_id=?`, eid).Scan(&exist)
		if exist == 0 {
			res, err := s.DB.Exec(`INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, tax_no, status) VALUES(?,?,?,?,?,?)`,
				eid, payType, base, bank, tax, status)
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
			id, _ := res.LastInsertId()
			api.OK(c, gin.H{"id": id, "employee_id": eid, "pay_type": payType, "monthly_base": base})
			return true
		}
		_, err := s.DB.Exec(`UPDATE pay_worker_profile SET pay_type=?, monthly_base=?, bank_account=?, tax_no=?, status=? WHERE employee_id=?`,
			payType, base, bank, tax, status, eid)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"employee_id": eid, "pay_type": payType, "monthly_base": base})
		return true
	case "get":
		id := paramID(c)
		var eid, pid int64
		var payType, bank, tax, status string
		var base float64
		err := s.DB.QueryRow(`SELECT id, employee_id, pay_type, COALESCE(monthly_base,0), COALESCE(bank_account,''), COALESCE(tax_no,''), status FROM pay_worker_profile WHERE id=?`, id).
			Scan(&pid, &eid, &payType, &base, &bank, &tax, &status)
		if err != nil {
			err = s.DB.QueryRow(`SELECT id, employee_id, pay_type, COALESCE(monthly_base,0), COALESCE(bank_account,''), COALESCE(tax_no,''), status FROM pay_worker_profile WHERE employee_id=?`, id).
				Scan(&pid, &eid, &payType, &base, &bank, &tax, &status)
			if err != nil {
				api.FailJSON(c, "NOT_FOUND")
				return true
			}
		}
		api.OK(c, gin.H{"id": pid, "employee_id": eid, "pay_type": payType, "monthly_base": base, "bank_account": bank, "tax_no": tax, "status": status})
		return true
	}
	return true
}

func (s *Services) handlePayrollSheets(c *gin.Context, method, action, path string) bool {
	switch {
	case action == "action:adjust":
		return s.adjustPayrollSheet(c)
	case action == "action:confirm":
		return s.setPayrollSheetStatus(c, "confirmed")
	case action == "action:pay":
		return s.setPayrollSheetStatus(c, "paid")
	case action == "list":
		rows, err := s.DB.Query(`SELECT id, doc_no, period_year, period_month, status, COALESCE(workshop_dept_id,0),
			COALESCE(calc_at,''), COALESCE(paid_at,''), COALESCE(remark,''), created_at
			FROM pay_payroll_sheet ORDER BY period_year DESC, period_month DESC, id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, ws int64
			var y, m int
			var docNo, status, calcAt, paidAt, remark, created string
			_ = rows.Scan(&id, &docNo, &y, &m, &status, &ws, &calcAt, &paidAt, &remark, &created)
			var lineCnt int
			var total float64
			_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(total_amount),0) FROM pay_payroll_sheet_line WHERE sheet_id=?`, id).Scan(&lineCnt, &total)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "period_year": y, "period_month": m, "period_ym": fmt.Sprintf("%04d-%02d", y, m),
				"status": status, "workshop_dept_id": ws, "calc_at": calcAt, "paid_at": paidAt, "remark": remark,
				"created_at": created, "line_count": lineCnt, "total_amount": total,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case action == "get":
		return s.getPayrollSheet(c)
	case action == "create":
		// create empty draft then generate, or just generate
		return s.batchGeneratePayrollSheet(c)
	case action == "update":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pay_payroll_sheet SET remark=COALESCE(?,remark), status=COALESCE(NULLIF(?,''),status) WHERE id=? AND status='draft'`,
			strOr(body["remark"]), strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	_ = path
	return true
}

func (s *Services) getPayrollSheet(c *gin.Context) bool {
	id := paramID(c)
	var docNo, status, calcAt, paidAt, remark, created string
	var y, m int
	var ws int64
	err := s.DB.QueryRow(`SELECT doc_no, period_year, period_month, status, COALESCE(workshop_dept_id,0),
		COALESCE(calc_at,''), COALESCE(paid_at,''), COALESCE(remark,''), created_at FROM pay_payroll_sheet WHERE id=?`, id).
		Scan(&docNo, &y, &m, &status, &ws, &calcAt, &paidAt, &remark, &created)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	rows, qerr := s.DB.Query(`SELECT l.id, l.employee_id, COALESCE(e.emp_no,''), COALESCE(e.name,''), COALESCE(l.emp_type,''),
		l.piece_amount, l.attendance_amount, l.commission_amount, l.adjust_amount, l.total_amount,
		COALESCE(p.bank_account,''), COALESCE(p.tax_no,''), COALESCE(p.pay_type,'')
		FROM pay_payroll_sheet_line l
		LEFT JOIN hr_employee e ON e.id=l.employee_id
		LEFT JOIN pay_worker_profile p ON p.employee_id=l.employee_id
		WHERE l.sheet_id=? ORDER BY l.id`, id)
	withProfile := qerr == nil
	if !withProfile {
		rows, _ = s.DB.Query(`SELECT l.id, l.employee_id, COALESCE(e.emp_no,''), COALESCE(e.name,''), COALESCE(l.emp_type,''),
			l.piece_amount, l.attendance_amount, l.commission_amount, l.adjust_amount, l.total_amount
			FROM pay_payroll_sheet_line l LEFT JOIN hr_employee e ON e.id=l.employee_id
			WHERE l.sheet_id=? ORDER BY l.id`, id)
	}
	lines := []gin.H{}
	var sum float64
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, eid int64
			var empNo, name, et, bank, tax, payType string
			var piece, att, comm, adj, total float64
			if withProfile {
				_ = rows.Scan(&lid, &eid, &empNo, &name, &et, &piece, &att, &comm, &adj, &total, &bank, &tax, &payType)
			} else {
				_ = rows.Scan(&lid, &eid, &empNo, &name, &et, &piece, &att, &comm, &adj, &total)
			}
			sum += total
			lines = append(lines, gin.H{
				"id": lid, "employee_id": eid, "emp_no": empNo, "name": name, "emp_type": et,
				"piece_amount": piece, "attendance_amount": att, "commission_amount": comm,
				"adjust_amount": adj, "total_amount": total,
				"bank_account": bank, "tax_no": tax, "pay_type": payType,
			})
		}
	}
	api.OK(c, gin.H{
		"id": id, "doc_no": docNo, "period_year": y, "period_month": m, "period_ym": fmt.Sprintf("%04d-%02d", y, m),
		"status": status, "workshop_dept_id": ws, "calc_at": calcAt, "paid_at": paidAt, "remark": remark,
		"created_at": created, "lines": lines, "total_amount": sum,
	})
	return true
}

func (s *Services) batchGeneratePayrollSheet(c *gin.Context) bool {
	body := bindBody(c)
	period := strOr(body["period_ym"])
	year, _ := asInt64(body["period_year"])
	month, _ := asInt64(body["period_month"])
	if period != "" {
		fmt.Sscanf(period, "%d-%d", &year, &month)
	}
	if year == 0 || month == 0 {
		now := time.Now()
		year, month = int64(now.Year()), int64(now.Month())
	}
	ws, _ := asInt64(body["workshop_dept_id"])
	force, _ := body["force"].(bool)
	sheetID, docNo, n, errMsg := s.generatePayrollSheet(int(year), int(month), ws, claimsUserID(c), force)
	if errMsg != "" {
		api.FailJSON(c, errMsg)
		return true
	}
	_, _ = s.DB.Exec(`INSERT INTO pay_payroll_calc_log(doc_no, period_ym, sheet_id, status, summary_json)
		VALUES(?,?,?, 'done', ?)`, docNo, fmt.Sprintf("%04d-%02d", year, month), sheetID,
		fmt.Sprintf(`{"lines":%d}`, n))
	api.OK(c, gin.H{
		"id": sheetID, "doc_no": docNo, "period_year": year, "period_month": month,
		"period_ym": fmt.Sprintf("%04d-%02d", year, month), "line_count": n, "status": "draft",
	})
	return true
}

func (s *Services) generatePayrollSheet(year, month int, workshopID, createdBy int64, force bool) (sheetID int64, docNo string, lineCount int, errMsg string) {
	EnsurePayrollSchema(s.DB)
	var existID int64
	var st string
	_ = s.DB.QueryRow(`SELECT id, status FROM pay_payroll_sheet WHERE period_year=? AND period_month=?`, year, month).Scan(&existID, &st)
	if existID > 0 {
		if st != "draft" && !force {
			return 0, "", 0, "SHEET_LOCKED"
		}
		if st == "draft" || force {
			_, _ = s.DB.Exec(`DELETE FROM pay_payroll_adjust WHERE sheet_id=?`, existID)
			_, _ = s.DB.Exec(`DELETE FROM pay_payroll_sheet_line WHERE sheet_id=?`, existID)
			sheetID = existID
			_ = s.DB.QueryRow(`SELECT doc_no FROM pay_payroll_sheet WHERE id=?`, existID).Scan(&docNo)
		}
	}
	if sheetID == 0 {
		docNo = fmt.Sprintf("PS%04d%02d", year, month)
		res, err := s.DB.Exec(`INSERT INTO pay_payroll_sheet(doc_no, period_year, period_month, status, workshop_dept_id, calc_at, created_by)
			VALUES(?,?,?,'draft',?,?,?)`, docNo, year, month, nullIf0(workshopID), time.Now().Format("2006-01-02 15:04:05"), nullIf0(createdBy))
		if err != nil {
			docNo = fmt.Sprintf("PS%04d%02d-%d", year, month, time.Now().Unix()%10000)
			res, err = s.DB.Exec(`INSERT INTO pay_payroll_sheet(doc_no, period_year, period_month, status, workshop_dept_id, calc_at, created_by)
				VALUES(?,?,?,'draft',?,?,?)`, docNo, year, month, nullIf0(workshopID), time.Now().Format("2006-01-02 15:04:05"), nullIf0(createdBy))
			if err != nil {
				return 0, "", 0, "DB_ERROR:" + err.Error()
			}
		}
		sheetID, _ = res.LastInsertId()
	} else {
		_, _ = s.DB.Exec(`UPDATE pay_payroll_sheet SET calc_at=?, workshop_dept_id=COALESCE(?,workshop_dept_id), status='draft' WHERE id=?`,
			time.Now().Format("2006-01-02 15:04:05"), nullIf0(workshopID), sheetID)
	}

	prefix := fmt.Sprintf("%04d-%02d", year, month)
	// employees in scope
	q := `SELECT e.id, e.emp_type, COALESCE(p.pay_type,''), COALESCE(p.monthly_base,0)
		FROM hr_employee e
		LEFT JOIN pay_worker_profile p ON p.employee_id=e.id
		WHERE COALESCE(e.is_deleted,0)=0 AND e.status='active'`
	args := []interface{}{}
	if workshopID > 0 {
		q += ` AND EXISTS (SELECT 1 FROM hr_employee_department ed WHERE ed.employee_id=e.id AND ed.dept_id=?)`
		args = append(args, workshopID)
	}
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return sheetID, docNo, 0, "DB_ERROR:" + err.Error()
	}
	defer rows.Close()
	type empPay struct {
		eid     int64
		empType string
		payType string
		monthly float64
	}
	emps := []empPay{}
	for rows.Next() {
		var eid int64
		var empType, payType string
		var monthly float64
		_ = rows.Scan(&eid, &empType, &payType, &monthly)
		if payType == "" {
			if empType == "fixed" || empType == "office" {
				payType = "fixed"
			} else {
				payType = "piece"
			}
		}
		emps = append(emps, empPay{eid: eid, empType: empType, payType: payType, monthly: monthly})
	}

	for _, e := range emps {
		var piece, att, comm float64
		if e.payType == "piece" || e.payType == "mixed" || e.empType == "piece" || e.empType == "temp" {
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM pd_piecework_summary
				WHERE worker_id=? AND biz_date LIKE ? AND COALESCE(status,'open')!='void'`, e.eid, prefix+"%").Scan(&piece)
		}
		if e.payType == "fixed" || e.payType == "mixed" || e.empType == "fixed" || e.empType == "office" {
			att = e.monthly
			// optional attendance factor from month stats
			var workDays, late float64
			_ = s.DB.QueryRow(`SELECT COALESCE(work_days,0), COALESCE(late_times,0) FROM hr_attendance_month_stat WHERE employee_id=? AND year=? AND month=?`,
				e.eid, year, month).Scan(&workDays, &late)
			if workDays > 0 && e.monthly > 0 {
				// soft deduction: 1 late = 0.5% of base, cap 10%
				deduct := late * 0.005
				if deduct > 0.1 {
					deduct = 0.1
				}
				att = e.monthly * (1 - deduct)
			}
		}
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(commission_amount),0) FROM pay_commission_calc WHERE employee_id=? AND period=?`,
			e.eid, prefix).Scan(&comm)
		total := piece + att + comm
		if total == 0 && piece == 0 && att == 0 && comm == 0 {
			// skip zero lines unless they have profile
			if e.monthly == 0 {
				continue
			}
		}
		_, err = s.DB.Exec(`INSERT INTO pay_payroll_sheet_line(sheet_id, employee_id, emp_type, piece_amount, attendance_amount, commission_amount, adjust_amount, total_amount)
			VALUES(?,?,?,?,?,?,0,?)`, sheetID, e.eid, e.empType, piece, att, comm, total)
		if err != nil {
			_, _ = s.DB.Exec(`UPDATE pay_payroll_sheet_line SET piece_amount=?, attendance_amount=?, commission_amount=?, total_amount=?, emp_type=? WHERE sheet_id=? AND employee_id=?`,
				piece, att, comm, total, e.empType, sheetID, e.eid)
		}
		lineCount++
	}
	return sheetID, docNo, lineCount, ""
}

func (s *Services) adjustPayrollSheet(c *gin.Context) bool {
	id := paramID(c)
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM pay_payroll_sheet WHERE id=?`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status == "paid" {
		api.FailJSON(c, "SHEET_PAID")
		return true
	}
	body := bindBody(c)
	eid, _ := asInt64(body["employee_id"])
	amount, _ := asFloat(body["amount"])
	adjType := strOrDef(body["adjust_type"], "manual")
	if eid == 0 {
		api.FailJSON(c, "EMPLOYEE_REQUIRED")
		return true
	}
	_, err := s.DB.Exec(`INSERT INTO pay_payroll_adjust(sheet_id, employee_id, adjust_type, amount, reason) VALUES(?,?,?,?,?)`,
		id, eid, adjType, amount, strOr(body["reason"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	// upsert line adjust
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pay_payroll_sheet_line WHERE sheet_id=? AND employee_id=?`, id, eid).Scan(&n)
	if n == 0 {
		_, _ = s.DB.Exec(`INSERT INTO pay_payroll_sheet_line(sheet_id, employee_id, adjust_amount, total_amount) VALUES(?,?,?,?)`,
			id, eid, amount, amount)
	} else {
		_, _ = s.DB.Exec(`UPDATE pay_payroll_sheet_line SET adjust_amount=adjust_amount+?, total_amount=piece_amount+attendance_amount+commission_amount+adjust_amount+? WHERE sheet_id=? AND employee_id=?`,
			amount, amount, id, eid)
	}
	api.OK(c, gin.H{"sheet_id": id, "employee_id": eid, "amount": amount})
	return true
}

func (s *Services) setPayrollSheetStatus(c *gin.Context, to string) bool {
	id := paramID(c)
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM pay_payroll_sheet WHERE id=?`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	switch to {
	case "confirmed":
		if status != "draft" && status != "confirmed" {
			api.FailJSON(c, "INVALID_STATUS")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE pay_payroll_sheet SET status='confirmed' WHERE id=?`, id)
	case "paid":
		if status != "confirmed" && status != "draft" {
			api.FailJSON(c, "CONFIRM_FIRST")
			return true
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		_, _ = s.DB.Exec(`UPDATE pay_payroll_sheet SET status='paid', paid_at=? WHERE id=?`, now, id)
	}
	api.OK(c, gin.H{"id": id, "status": to})
	return true
}

func (s *Services) handlePayrollCalculations(c *gin.Context, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, COALESCE(doc_no,''), COALESCE(period_ym,''), COALESCE(sheet_id,0), COALESCE(status,''), COALESCE(summary_json,''), created_at
			FROM pay_payroll_calc_log ORDER BY id DESC LIMIT 100`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, sid int64
				var docNo, period, status, sj, created string
				_ = rows.Scan(&id, &docNo, &period, &sid, &status, &sj, &created)
				list = append(list, gin.H{"id": id, "doc_no": docNo, "period_ym": period, "sheet_id": sid, "status": status, "summary_json": sj, "created_at": created})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		return s.batchGeneratePayrollSheet(c)
	case "get":
		id := paramID(c)
		var sid int64
		var docNo, period, status, sj, created string
		err := s.DB.QueryRow(`SELECT COALESCE(doc_no,''), COALESCE(period_ym,''), COALESCE(sheet_id,0), COALESCE(status,''), COALESCE(summary_json,''), created_at FROM pay_payroll_calc_log WHERE id=?`, id).
			Scan(&docNo, &period, &sid, &status, &sj, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "period_ym": period, "sheet_id": sid, "status": status, "summary_json": sj, "created_at": created})
		return true
	}
	return true
}

func (s *Services) handleCommissionRules(c *gin.Context, action string) bool {
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT id, name, COALESCE(rule_json,'{}'), effective_from, COALESCE(effective_to,''), status, created_at FROM pay_sales_commission_rule ORDER BY id DESC`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var name, rj, from, to, status, created string
				_ = rows.Scan(&id, &name, &rj, &from, &to, &status, &created)
				rate := 0.0
				var m map[string]interface{}
				_ = json.Unmarshal([]byte(rj), &m)
				if m != nil {
					if f, ok := asFloat(m["rate"]); ok {
						rate = f
					}
				}
				list = append(list, gin.H{"id": id, "name": name, "rule_json": rj, "rate": rate, "effective_from": from, "effective_to": to, "status": status, "created_at": created})
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
		rj := strOrDef(body["rule_json"], "{}")
		if m, ok := body["rule_json"].(map[string]interface{}); ok {
			b, _ := json.Marshal(m)
			rj = string(b)
		} else if rate, ok := asFloat(body["rate"]); ok {
			b, _ := json.Marshal(map[string]interface{}{"rate": rate, "base": "order_amount"})
			rj = string(b)
		}
		from := strOrDef(body["effective_from"], time.Now().Format("2006-01-02"))
		res, err := s.DB.Exec(`INSERT INTO pay_sales_commission_rule(name, rule_json, effective_from, status) VALUES(?,?,?,'active')`, name, rj, from)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "name": name})
		return true
	case "get", "update", "delete":
		id := paramID(c)
		if action == "delete" {
			_, _ = s.DB.Exec(`UPDATE pay_sales_commission_rule SET status='inactive' WHERE id=?`, id)
			api.OK(c, gin.H{"id": id})
			return true
		}
		if action == "get" {
			var name, rj, from, to, status string
			err := s.DB.QueryRow(`SELECT name, COALESCE(rule_json,'{}'), effective_from, COALESCE(effective_to,''), status FROM pay_sales_commission_rule WHERE id=?`, id).
				Scan(&name, &rj, &from, &to, &status)
			if err != nil {
				api.FailJSON(c, "NOT_FOUND")
				return true
			}
			api.OK(c, gin.H{"id": id, "name": name, "rule_json": rj, "effective_from": from, "effective_to": to, "status": status})
			return true
		}
		body := bindBody(c)
		rj := strOr(body["rule_json"])
		if rate, ok := asFloat(body["rate"]); ok {
			b, _ := json.Marshal(map[string]interface{}{"rate": rate, "base": "order_amount"})
			rj = string(b)
		}
		_, _ = s.DB.Exec(`UPDATE pay_sales_commission_rule SET name=COALESCE(NULLIF(?,''),name), rule_json=COALESCE(NULLIF(?,''),rule_json),
			status=COALESCE(NULLIF(?,''),status) WHERE id=?`, strOr(body["name"]), rj, strOr(body["status"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	return true
}

func (s *Services) handleCommissionCalcs(c *gin.Context, method, action, path string) bool {
	if action == "action:run" || (method == "POST" && strings.Contains(path, "/run")) || (action == "create" && strings.Contains(c.Request.URL.Path, "run")) {
		return s.runCommissionCalc(c)
	}
	switch action {
	case "list":
		rows, _ := s.DB.Query(`SELECT c.id, c.rule_id, COALESCE(r.name,''), c.employee_id, COALESCE(e.emp_no,''), COALESCE(e.name,''),
			c.period, c.base_amount, c.commission_amount, COALESCE(c.source_doc_refs,''), c.created_at
			FROM pay_commission_calc c
			LEFT JOIN pay_sales_commission_rule r ON r.id=c.rule_id
			LEFT JOIN hr_employee e ON e.id=c.employee_id
			ORDER BY c.id DESC LIMIT 200`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, rid, eid int64
				var rname, empNo, name, period, refs, created string
				var base, comm float64
				_ = rows.Scan(&id, &rid, &rname, &eid, &empNo, &name, &period, &base, &comm, &refs, &created)
				list = append(list, gin.H{
					"id": id, "rule_id": rid, "rule_name": rname, "employee_id": eid, "emp_no": empNo, "name": name,
					"period": period, "base_amount": base, "commission_amount": comm, "source_doc_refs": refs, "created_at": created,
				})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		if _, ok := body["run"].(bool); ok && body["run"] == true {
			return s.runCommissionCalc(c)
		}
		rid, _ := asInt64(body["rule_id"])
		eid, _ := asInt64(body["employee_id"])
		period := strOrDef(body["period"], time.Now().Format("2006-01"))
		base, _ := asFloat(body["base_amount"])
		rate := 0.01
		if rid > 0 {
			var rj string
			_ = s.DB.QueryRow(`SELECT COALESCE(rule_json,'{}') FROM pay_sales_commission_rule WHERE id=?`, rid).Scan(&rj)
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(rj), &m)
			if f, ok := asFloat(m["rate"]); ok {
				rate = f
			}
		}
		comm, _ := asFloat(body["commission_amount"])
		if !okFloat(body["commission_amount"]) {
			comm = base * rate
		}
		if eid == 0 {
			api.FailJSON(c, "EMPLOYEE_REQUIRED")
			return true
		}
		if rid == 0 {
			_ = s.DB.QueryRow(`SELECT id FROM pay_sales_commission_rule WHERE status='active' ORDER BY id LIMIT 1`).Scan(&rid)
		}
		res, err := s.DB.Exec(`INSERT INTO pay_commission_calc(rule_id, employee_id, period, base_amount, commission_amount, source_doc_refs)
			VALUES(?,?,?,?,?,?)`, rid, eid, period, base, comm, strOr(body["source_doc_refs"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "employee_id": eid, "period": period, "base_amount": base, "commission_amount": comm})
		return true
	}
	return true
}

func okFloat(v interface{}) bool {
	_, ok := asFloat(v)
	return ok
}

func (s *Services) runCommissionCalc(c *gin.Context) bool {
	body := bindBody(c)
	period := strOrDef(body["period"], time.Now().Format("2006-01"))
	var ruleID int64
	var rj string
	_ = s.DB.QueryRow(`SELECT id, COALESCE(rule_json,'{}') FROM pay_sales_commission_rule WHERE status='active' ORDER BY id LIMIT 1`).Scan(&ruleID, &rj)
	rate := 0.01
	var m map[string]interface{}
	_ = json.Unmarshal([]byte(rj), &m)
	if f, ok := asFloat(m["rate"]); ok {
		rate = f
	}
	// sales employees: office or emp with sales in job_title, or all with user
	rows, err := s.DB.Query(`SELECT id FROM hr_employee WHERE COALESCE(is_deleted,0)=0 AND status='active'
		AND (emp_type='office' OR COALESCE(job_title,'') LIKE '%销售%' OR emp_type='fixed')`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var eid int64
		_ = rows.Scan(&eid)
		base, _ := asFloat(body["base_amount"])
		if base == 0 {
			// try outbound settle amount for period if table exists
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM sl_outbound_settle WHERE COALESCE(sales_employee_id,0)=? AND biz_date LIKE ?`,
				eid, period+"%").Scan(&base)
			if base == 0 {
				continue
			}
		}
		comm := base * rate
		_, _ = s.DB.Exec(`INSERT INTO pay_commission_calc(rule_id, employee_id, period, base_amount, commission_amount, source_doc_refs)
			VALUES(?,?,?,?,?,'auto')`, ruleID, eid, period, base, comm)
		n++
	}
	// if none matched, allow manual single from body
	if n == 0 {
		if eid, _ := asInt64(body["employee_id"]); eid > 0 {
			base, _ := asFloat(body["base_amount"])
			comm := base * rate
			_, _ = s.DB.Exec(`INSERT INTO pay_commission_calc(rule_id, employee_id, period, base_amount, commission_amount, source_doc_refs)
				VALUES(?,?,?,?,?,'manual')`, ruleID, eid, period, base, comm)
			n = 1
		}
	}
	api.OK(c, gin.H{"period": period, "rule_id": ruleID, "rate": rate, "created": n})
	return true
}

func (s *Services) handlePayrollWorkRecords(c *gin.Context) bool {
	var empID int64
	fmt.Sscanf(strings.TrimSpace(c.Query("employee_id")), "%d", &empID)
	if empID <= 0 {
		api.FailJSON(c, "EMPLOYEE_ID_REQUIRED")
		return true
	}
	today := time.Now().Format("2006-01-02")
	from := strings.TrimSpace(c.Query("date_from"))
	if from == "" {
		from = strings.TrimSpace(c.Query("biz_date"))
	}
	to := strings.TrimSpace(c.Query("date_to"))
	if from == "" {
		from = today
	}
	if to == "" {
		to = from
	}
	if from > to {
		from, to = to, from
	}

	emp := gin.H{"id": empID, "emp_no": "", "name": ""}
	var empNo, name string
	if err := s.DB.QueryRow(`SELECT COALESCE(emp_no,''), COALESCE(name,'') FROM hr_employee WHERE id=?`, empID).Scan(&empNo, &name); err == nil {
		emp["emp_no"] = empNo
		emp["name"] = name
	}

	var issueKg, returnKg, pieceQty, pieceAmt, wageAmt float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(kg),0) FROM pd_station_flow_log
		WHERE worker_id=? AND biz_date>=? AND biz_date<=? AND event_type='issue'`, empID, from, to).Scan(&issueKg)
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(kg),0) FROM pd_station_flow_log
		WHERE worker_id=? AND biz_date>=? AND biz_date<=? AND event_type='return'`, empID, from, to).Scan(&returnKg)
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0), COALESCE(SUM(amount),0) FROM pd_piecework_summary
		WHERE worker_id=? AND biz_date>=? AND biz_date<=?`, empID, from, to).Scan(&pieceQty, &pieceAmt)

	fromT, errFrom := time.Parse("2006-01-02", from)
	toT, errTo := time.Parse("2006-01-02", to)
	now := time.Now()
	if errFrom != nil {
		fromT = now
	}
	if errTo != nil {
		toT = fromT
	}
	fromYM := fromT.Year()*100 + int(fromT.Month())
	toYM := toT.Year()*100 + int(toT.Month())
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.total_amount),0)
		FROM pay_payroll_sheet_line l
		JOIN pay_payroll_sheet s ON s.id=l.sheet_id
		WHERE l.employee_id=? AND (s.period_year*100+s.period_month) BETWEEN ? AND ?`,
		empID, fromYM, toYM).Scan(&wageAmt)

	flows := []gin.H{}
	if rows, err := s.DB.Query(`SELECT id, event_type, biz_date, COALESCE(board_code,''), COALESCE(process_id,0), COALESCE(process_name,''),
		COALESCE(kg,0), COALESCE(pay_mode,''), COALESCE(emp_type,''), COALESCE(rate,0), COALESCE(amount,0), COALESCE(remark,''), CAST(created_at AS TEXT)
		FROM pd_station_flow_log
		WHERE worker_id=? AND biz_date>=? AND biz_date<=?
		ORDER BY id DESC LIMIT 500`, empID, from, to); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, procID int64
			var et, bd, board, processName, payMode, empType, remark, created string
			var kg, rate, amount float64
			if err := rows.Scan(&id, &et, &bd, &board, &procID, &processName, &kg, &payMode, &empType, &rate, &amount, &remark, &created); err != nil {
				continue
			}
			flows = append(flows, gin.H{
				"id": id, "event_type": et, "biz_date": bd, "board_code": board,
				"process_id": procID, "process_name": processName,
				"kg": kg, "pay_mode": payMode, "emp_type": empType, "rate": rate, "amount": amount,
				"remark": remark, "created_at": created,
			})
		}
	}

	piecework := []gin.H{}
	if rows, err := s.DB.Query(`SELECT s.id, s.process_id, COALESCE(p.name,''), s.biz_date, s.qty, COALESCE(s.weight,0), s.amount
		FROM pd_piecework_summary s
		LEFT JOIN pd_process p ON p.id=s.process_id
		WHERE s.worker_id=? AND s.biz_date>=? AND s.biz_date<=?
		ORDER BY s.biz_date DESC, s.id DESC LIMIT 500`, empID, from, to); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, procID int64
			var processName, bizDate string
			var qty, weight, amount float64
			if err := rows.Scan(&id, &procID, &processName, &bizDate, &qty, &weight, &amount); err != nil {
				continue
			}
			piecework = append(piecework, gin.H{
				"id": id, "process_id": procID, "process_name": processName,
				"biz_date": bizDate, "qty": qty, "weight": weight, "amount": amount,
			})
		}
	}

	wages := []gin.H{}
	if rows, err := s.DB.Query(`SELECT l.id, s.id, s.doc_no, s.period_year, s.period_month, COALESCE(s.status,''),
		COALESCE(s.calc_at,''), COALESCE(s.paid_at,''), COALESCE(l.emp_type,''),
		l.piece_amount, l.attendance_amount, l.commission_amount, l.adjust_amount, l.total_amount
		FROM pay_payroll_sheet_line l
		JOIN pay_payroll_sheet s ON s.id=l.sheet_id
		WHERE l.employee_id=? AND (s.period_year*100+s.period_month) BETWEEN ? AND ?
		ORDER BY s.period_year DESC, s.period_month DESC, l.id DESC
		LIMIT 100`, empID, fromYM, toYM); err == nil {
		defer rows.Close()
		for rows.Next() {
			var lineID, sheetID int64
			var year, month int
			var docNo, status, calcAt, paidAt, empType string
			var piece, att, comm, adj, total float64
			if err := rows.Scan(&lineID, &sheetID, &docNo, &year, &month, &status, &calcAt, &paidAt, &empType,
				&piece, &att, &comm, &adj, &total); err != nil {
				continue
			}
			wages = append(wages, gin.H{
				"id": lineID, "sheet_id": sheetID, "doc_no": docNo,
				"period_year": year, "period_month": month,
				"period_ym": fmt.Sprintf("%04d-%02d", year, month),
				"status": status, "calc_at": calcAt, "paid_at": paidAt, "emp_type": empType,
				"piece_amount": piece, "attendance_amount": att, "commission_amount": comm,
				"adjust_amount": adj, "total_amount": total,
			})
		}
	}

	api.OK(c, gin.H{
		"employee": emp,
		"date_from": from, "date_to": to,
		"kpi": gin.H{
			"issue_kg": issueKg, "return_kg": returnKg,
			"piece_qty": pieceQty, "piece_amount": pieceAmt, "wage_amount": wageAmt,
			"flow_count": len(flows), "piecework_count": len(piecework), "wage_count": len(wages),
		},
		"flows": flows, "piecework": piecework, "wages": wages,
	})
	return true
}
