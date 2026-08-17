package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/persistence/sqlutil"
	"erp/internal/security"
)

func (s *Services) handlePieceworkSummaries(c *gin.Context, method, action, openapiPath string) bool {
	if strings.Contains(openapiPath, "/mine") || strings.Contains(c.Request.URL.Path, "/mine") {
		return s.pieceworkMine(c)
	}
	if strings.Contains(openapiPath, "/recalc") || strings.Contains(c.Request.URL.Path, "/recalc") {
		return s.recalcPieceworkSummaries(c)
	}
	if method == "GET" && action == "list" {
		return s.listPieceworkSummaries(c)
	}
	return false
}

func (s *Services) listPieceworkSummaries(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	bizDate := c.Query("biz_date")
	where := `WHERE 1=1`
	args := []interface{}{}
	if bizDate != "" {
		where += ` AND s.biz_date=?`
		args = append(args, bizDate)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_piecework_summary s `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT s.id, s.worker_id, COALESCE(e.name,''), s.process_id, COALESCE(p.name,''),
		s.biz_date, s.qty, COALESCE(s.weight,0), COALESCE(s.input_weight,0), COALESCE(s.output_weight,0),
		COALESCE(s.loss,0), COALESCE(s.utilization,0), s.amount, COALESCE(s.source_report_ids,'')
		FROM pd_piecework_summary s
		LEFT JOIN hr_employee e ON e.id=s.worker_id
		LEFT JOIN pd_process p ON p.id=s.process_id
		`+where+` ORDER BY s.biz_date DESC, s.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, workerID, processID int64
		var workerName, processName, bizDate, sources string
		var qty, weight, inW, outW, loss, util, amount float64
		_ = rows.Scan(&id, &workerID, &workerName, &processID, &processName, &bizDate, &qty, &weight, &inW, &outW, &loss, &util, &amount, &sources)
		list = append(list, gin.H{
			"id": id, "worker_id": workerID, "worker_name": workerName, "process_id": processID, "process_name": processName,
			"biz_date": bizDate, "qty": qty, "weight": weight, "input_weight": inW, "output_weight": outW,
			"loss": loss, "utilization": util, "amount": amount, "source_report_ids": sources,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) pieceworkMine(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	workerID := s.resolveWorkerID(c)
	if workerID <= 0 {
		api.FailJSON(c, "WORKER_NOT_BOUND")
		return true
	}
	rows, err := s.DB.Query(`SELECT s.id, s.process_id, COALESCE(p.name,''), s.qty, COALESCE(s.weight,0),
		COALESCE(s.input_weight,0), COALESCE(s.output_weight,0), COALESCE(s.loss,0), COALESCE(s.utilization,0), s.amount,
		COALESCE((SELECT rate FROM pay_process_wage_rate r WHERE r.process_id=s.process_id AND r.status='active' ORDER BY r.id DESC LIMIT 1),0)
		FROM pd_piecework_summary s
		LEFT JOIN pd_process p ON p.id=s.process_id
		WHERE s.worker_id=? AND s.biz_date=? ORDER BY s.process_id`, workerID, bizDate)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	var totalQty, totalAmount, totalLoss, totalIn, totalOut float64
	for rows.Next() {
		var id, processID int64
		var processName string
		var qty, weight, inW, outW, loss, util, amount, rate float64
		_ = rows.Scan(&id, &processID, &processName, &qty, &weight, &inW, &outW, &loss, &util, &amount, &rate)
		list = append(list, gin.H{
			"id": id, "process_id": processID, "process_name": processName, "qty": qty, "weight": weight,
			"input_weight": inW, "output_weight": outW, "loss": loss, "utilization": util, "amount": amount, "rate": rate,
		})
		totalQty += qty
		totalAmount += amount
		totalLoss += loss
		totalIn += inW
		totalOut += outW
	}
	reports := []gin.H{}
	rrows, _ := s.DB.Query(`SELECT id, doc_no, process_id, COALESCE(qty,0), COALESCE(weight,0), COALESCE(input_weight,0),
		COALESCE(output_weight,0), COALESCE(loss,0), COALESCE(utilization,0), COALESCE(scan_code,''), reported_at
		FROM pd_report_work WHERE worker_id=? AND date(reported_at)=? ORDER BY id DESC`, workerID, bizDate)
	if rrows != nil {
		defer rrows.Close()
		for rrows.Next() {
			var id, processID int64
			var docNo, scan, reported string
			var qty, weight, inW, outW, loss, util float64
			_ = rrows.Scan(&id, &docNo, &processID, &qty, &weight, &inW, &outW, &loss, &util, &scan, &reported)
			reports = append(reports, gin.H{
				"id": id, "doc_no": docNo, "process_id": processID, "qty": qty, "weight": weight,
				"input_weight": inW, "output_weight": outW, "loss": loss, "utilization": util,
				"scan_code": scan, "reported_at": reported,
			})
		}
	}
	var workerName string
	_ = s.DB.QueryRow(`SELECT name FROM hr_employee WHERE id=?`, workerID).Scan(&workerName)
	api.OK(c, gin.H{
		"worker_id": workerID, "worker_name": workerName, "biz_date": bizDate,
		"summaries": list, "reports": reports,
		"total_qty": totalQty, "total_amount": totalAmount, "total_loss": totalLoss,
		"total_input_weight": totalIn, "total_output_weight": totalOut,
	})
	return true
}

func (s *Services) resolveWorkerID(c *gin.Context) int64 {
	if wid, ok := asInt64(c.Query("worker_id")); ok && wid > 0 {
		return wid
	}
	if badge := c.Query("badge_code"); badge != "" {
		var id int64
		_ = s.DB.QueryRow(`SELECT id FROM hr_employee WHERE badge_code=? AND COALESCE(is_deleted,0)=0`, badge).Scan(&id)
		if id > 0 {
			return id
		}
	}
	claims := middleware.Claims(c)
	if claims != nil && claims.UserID > 0 {
		var empID int64
		_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, claims.UserID).Scan(&empID)
		if empID > 0 {
			return empID
		}
	}
	return 0
}

func (s *Services) recalcPieceworkSummaries(c *gin.Context) bool {
	body := bindBody(c)
	bizDate := strOrDef(body["biz_date"], c.Query("biz_date"))
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	rows, err := s.DB.Query(`SELECT worker_id, process_id,
		SUM(COALESCE(output_weight, qty_net, qty, 0)),
		SUM(COALESCE(input_weight, 0)),
		SUM(COALESCE(loss, 0)),
		string_agg((id)::text, ',')
		FROM pd_report_work
		WHERE date(reported_at)=? AND worker_id IS NOT NULL AND process_id IS NOT NULL
		GROUP BY worker_id, process_id`, bizDate)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var workerID, processID int64
		var outW, inW, loss float64
		var ids string
		_ = rows.Scan(&workerID, &processID, &outW, &inW, &loss, &ids)
		var rate float64
		_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, processID).Scan(&rate)
		amount := outW * rate
		util := 0.0
		if inW > 0 {
			util = outW / inW
		}
		var exist int64
		_ = s.DB.QueryRow(`SELECT id FROM pd_piecework_summary WHERE worker_id=? AND process_id=? AND biz_date=?`, workerID, processID, bizDate).Scan(&exist)
		if exist > 0 {
			_, _ = s.DB.Exec(`UPDATE pd_piecework_summary SET qty=?, weight=?, input_weight=?, output_weight=?, loss=?, utilization=?, amount=?, source_report_ids=?, updated_at=NOW() WHERE id=?`,
				outW, outW, inW, outW, loss, util, amount, ids, exist)
		} else {
			_, _ = s.DB.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, weight, input_weight, output_weight, loss, utilization, amount, source_report_ids)
				VALUES(?,?,?,?,?,?,?,?,?,?,?)`, workerID, processID, bizDate, outW, outW, inW, outW, loss, util, amount, ids)
		}
		n++
	}
	api.OK(c, gin.H{"biz_date": bizDate, "recalculated": n})
	return true
}

func (s *Services) upsertPieceworkSummary(workerID, processID, reportID int64, qty, inputWeight, outputWeight, loss, utilization float64) {
	if processID <= 0 || workerID <= 0 {
		return
	}
	bizDate := time.Now().Format("2006-01-02")
	var rate float64
	_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, processID).Scan(&rate)
	amount := outputWeight * rate
	if outputWeight <= 0 {
		amount = qty * rate
		outputWeight = qty
	}
	var exist int64
	var src string
	_ = s.DB.QueryRow(`SELECT id, COALESCE(source_report_ids,'') FROM pd_piecework_summary WHERE worker_id=? AND process_id=? AND biz_date=?`,
		workerID, processID, bizDate).Scan(&exist, &src)
	rid := fmt.Sprintf("%d", reportID)
	if exist > 0 {
		if src != "" && !strings.Contains(","+src+",", ","+rid+",") {
			src = src + "," + rid
		} else if src == "" {
			src = rid
		}
		_, _ = s.DB.Exec(`UPDATE pd_piecework_summary SET
			qty = qty + ?, weight = COALESCE(weight,0) + ?, input_weight = COALESCE(input_weight,0) + ?,
			output_weight = COALESCE(output_weight,0) + ?, loss = COALESCE(loss,0) + ?,
			utilization = CASE WHEN (COALESCE(input_weight,0)+?)>0 THEN (COALESCE(output_weight,0)+?)/(COALESCE(input_weight,0)+?) ELSE 0 END,
			amount = amount + ?, source_report_ids=?, updated_at=NOW() WHERE id=?`,
			qty, outputWeight, inputWeight, outputWeight, loss, inputWeight, outputWeight, inputWeight, amount, src, exist)
	} else {
		_, _ = s.DB.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, weight, input_weight, output_weight, loss, utilization, amount, source_report_ids)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, workerID, processID, bizDate, qty, outputWeight, inputWeight, outputWeight, loss, utilization, amount, rid)
	}
}

func (s *Services) batchImportEmployees(c *gin.Context) bool {
	body := bindBody(c)
	raw, _ := body["rows"].([]interface{})
	if len(raw) == 0 {
		api.FailJSON(c, "ROWS_REQUIRED")
		return true
	}
	autoOpen := boolOr(body["auto_open_account"], false)
	created, skipped := 0, 0
	errors := []gin.H{}
	ids := []int64{}
	for i, item := range raw {
		m, _ := item.(map[string]interface{})
		if m == nil {
			skipped++
			errors = append(errors, gin.H{"index": i, "error": "INVALID_ROW"})
			continue
		}
		if _, ok := m["emp_type"]; !ok {
			m["emp_type"] = "piece"
		}
		if _, ok := m["status"]; !ok {
			m["status"] = "active"
		}
		// 批量走 createEmployeeFromBody，不开户；需要时由 auto_open_account 再开
		id, errMsg := s.createEmployeeFromBody(m, strOrDef(m["status"], "active"))
		if errMsg != "" {
			skipped++
			errors = append(errors, gin.H{"index": i, "emp_no": strOr(m["emp_no"]), "error": errMsg})
			continue
		}
		created++
		ids = append(ids, id)
		if autoOpen {
			_, _, _ = s.openAccountForEmployeeEx(id, "[]", strOr(m["login_name"]), strOr(m["password"]))
		}
	}
	api.OK(c, gin.H{"created": created, "skipped": skipped, "errors": errors, "ids": ids, "total": len(raw)})
	return true
}

func (s *Services) openEmployeeAccountByID(empID int64, body map[string]interface{}) (int64, string) {
	// lightweight: reuse open path if available via internal call pattern
	var uid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&uid)
	if uid > 0 {
		return uid, ""
	}
	login := strOr(body["login_name"])
	if login == "" {
		login = strOr(body["emp_no"])
	}
	if login == "" {
		return 0, "LOGIN_REQUIRED"
	}
	pass := strOrDef(body["password"], "123456")
	hash, err := security.HashPassword(pass)
	if err != nil {
		return 0, "HASH_ERROR"
	}
	res, err := s.DB.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted) VALUES(?,?,?,'biz','active',0)`,
		login, hash, empID)
	if err != nil {
		return 0, "USER_CREATE_ERROR"
	}
	uid, _ = res.LastInsertId()
	_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, uid, empID)
	return uid, ""
}
