package biz

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/persistence/sqlutil"
	"erp/internal/security"
)

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func (s *Services) processWageRate(processID int64) float64 {
	if processID <= 0 {
		return 0
	}
	var rate float64
	_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, processID).Scan(&rate)
	return rate
}

func (s *Services) isPieceworkProcess(processID, stepID int64) bool {
	// Yield-pay process: pay_mode weight|piece (legacy is_piecework / step flag still honored via processPayMode).
	if processID > 0 {
		return s.processPaysYield(processID)
	}
	if stepID > 0 {
		if step := s.loadStep(stepID); step != nil {
			return step.IsPiecework
		}
	}
	return false
}

// workerLockedPieceworkKg 领料预锁定 kg（issue − 退库 − 已日结），未入日汇总；仅计件工。
func (s *Services) workerLockedPieceworkKg(workerID, processID int64) float64 {
	if workerID <= 0 || !s.workerYieldEligible(workerID) {
		return 0
	}
	q := `SELECT COALESCE(SUM(i.issue_kg - i.returned_kg - COALESCE(i.wage_settled_kg,0)),0)
		FROM pd_process_issue i
		LEFT JOIN pd_process p ON p.id=i.process_id
		WHERE i.worker_id=?
		  AND (i.issue_kg - i.returned_kg - COALESCE(i.wage_settled_kg,0)) > 0
		  AND (COALESCE(NULLIF(p.pay_mode,''),'') IN ('weight','piece') OR (COALESCE(NULLIF(p.pay_mode,''),'')='' AND COALESCE(p.is_piecework,0)=1))`
	args := []interface{}{workerID}
	if processID > 0 {
		q += ` AND i.process_id=?`
		args = append(args, processID)
	}
	var v float64
	_ = s.DB.QueryRow(q, args...).Scan(&v)
	if v < kgEps {
		return 0
	}
	return roundKg(v)
}

func (s *Services) attachPieceworkLockPreview(dst gin.H, workerID, processID, stepID int64) {
	_ = stepID
	eligible := s.shouldLockYieldWage(processID, workerID)
	dst["piecework"] = eligible
	dst["pay_mode"] = s.processPayMode(processID)
	dst["emp_type"] = s.workerEmpType(workerID)
	if !eligible {
		dst["locked_kg"] = 0.0
		dst["locked_wage_amount"] = 0.0
		dst["piecework_status"] = "none"
		return
	}
	rate := s.processWageRate(processID)
	lockedKg := s.workerLockedPieceworkKg(workerID, processID)
	dst["rate"] = rate
	dst["locked_kg"] = lockedKg
	dst["locked_wage_amount"] = roundMoney(lockedKg * rate)
	dst["piecework_status"] = "locked"
	dst["piecework_hint"] = "预估工钱，当日日结入账"
}

func (s *Services) listWorkerPieceworkLocks(workerID int64) ([]gin.H, float64, float64) {
	if !s.workerYieldEligible(workerID) {
		return []gin.H{}, 0, 0
	}
	rows, err := s.DB.Query(`SELECT i.process_id, COALESCE(p.name,''),
		COALESCE(SUM(i.issue_kg - i.returned_kg - COALESCE(i.wage_settled_kg,0)),0),
		COALESCE((SELECT rate FROM pay_process_wage_rate r WHERE r.process_id=i.process_id AND r.status='active' ORDER BY r.id DESC LIMIT 1),0)
		FROM pd_process_issue i
		LEFT JOIN pd_process p ON p.id=i.process_id
		WHERE i.worker_id=?
		  AND (i.issue_kg - i.returned_kg - COALESCE(i.wage_settled_kg,0)) > 0
		  AND (COALESCE(NULLIF(p.pay_mode,''),'') IN ('weight','piece') OR (COALESCE(NULLIF(p.pay_mode,''),'')='' AND COALESCE(p.is_piecework,0)=1))
		GROUP BY i.process_id, p.name`, workerID)
	if err != nil {
		return nil, 0, 0
	}
	defer rows.Close()
	list := []gin.H{}
	var totalKg, totalAmt float64
	for rows.Next() {
		var processID int64
		var processName string
		var kg, rate float64
		if err := rows.Scan(&processID, &processName, &kg, &rate); err != nil {
			continue
		}
		if kg <= kgEps {
			continue
		}
		kg = roundKg(kg)
		amt := roundMoney(kg * rate)
		list = append(list, gin.H{
			"process_id": processID, "process_name": processName,
			"locked_kg": kg, "rate": rate, "locked_wage_amount": amt,
			"piecework_status": "locked",
		})
		totalKg = roundKg(totalKg + kg)
		totalAmt = roundMoney(totalAmt + amt)
	}
	return list, totalKg, totalAmt
}

func (s *Services) handlePieceworkSummaries(c *gin.Context, method, action, openapiPath string) bool {
	if strings.Contains(openapiPath, "/mine") || strings.Contains(c.Request.URL.Path, "/mine") {
		return s.pieceworkMine(c)
	}
	if strings.Contains(openapiPath, "/recalc") || strings.Contains(c.Request.URL.Path, "/recalc") {
		return s.recalcPieceworkSummaries(c)
	}
	if strings.Contains(openapiPath, "/day-settle") || strings.Contains(c.Request.URL.Path, "/day-settle") {
		if method != "POST" {
			api.FailJSON(c, "METHOD_NOT_ALLOWED")
			return true
		}
		return s.handlePieceworkDaySettle(c)
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
		rrows, _ := s.DB.Query(`SELECT id, event_type, COALESCE(process_id,0), COALESCE(kg,0), COALESCE(amount,0),
		COALESCE(board_code,''), COALESCE(CAST(created_at AS TEXT),'')
		FROM pd_station_flow_log WHERE worker_id=? AND biz_date=? ORDER BY id DESC LIMIT 100`, workerID, bizDate)
	if rrows != nil {
		defer rrows.Close()
		for rrows.Next() {
			var id, processID int64
			var eventType, board, created string
			var kg, amount float64
			_ = rrows.Scan(&id, &eventType, &processID, &kg, &amount, &board, &created)
			reports = append(reports, gin.H{
				"id": id, "event_type": eventType, "process_id": processID, "qty": kg, "weight": kg,
				"amount": amount, "scan_code": board, "board_code": board, "reported_at": created,
			})
		}
	}
	var workerName string
	_ = s.DB.QueryRow(`SELECT name FROM hr_employee WHERE id=?`, workerID).Scan(&workerName)
	pending, pendingKg, pendingAmt := s.listWorkerPieceworkLocks(workerID)
	api.OK(c, gin.H{
		"worker_id": workerID, "worker_name": workerName, "biz_date": bizDate,
		"summaries": list, "reports": reports,
		"pending_locks": pending,
		"total_qty":     totalQty, "total_amount": totalAmount, "total_loss": totalLoss,
		"total_input_weight": totalIn, "total_output_weight": totalOut,
		"pending_locked_kg": pendingKg, "pending_locked_amount": pendingAmt,
		"settled_kg": totalQty, "settled_amount": totalAmount,
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
	// 旧报工重算已下线；产量以领料占用日结为准。
	api.FailJSON(c, "FEATURE_REMOVED:recalc_from_report_works;use_piecework_day_settle")
	return true
}

func (s *Services) upsertPieceworkSummary(workerID, processID, reportID int64, qty, inputWeight, outputWeight, loss, utilization float64) {
	s.upsertPieceworkSummaryKeyed(workerID, processID, fmt.Sprintf("%d", reportID), qty, inputWeight, outputWeight, loss, utilization)
}

func (s *Services) upsertPieceworkSummaryKeyed(workerID, processID int64, sourceKey string, qty, inputWeight, outputWeight, loss, utilization float64) {
	s.upsertPieceworkSummaryKeyedOnDate(workerID, processID, time.Now().Format("2006-01-02"), sourceKey, qty, inputWeight, outputWeight, loss, utilization)
}

func (s *Services) upsertPieceworkSummaryKeyedOnDate(workerID, processID int64, bizDate, sourceKey string, qty, inputWeight, outputWeight, loss, utilization float64) {
	if processID <= 0 || workerID <= 0 {
		return
	}
	if strings.TrimSpace(bizDate) == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
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
	rid := strings.TrimSpace(sourceKey)
	if rid == "" {
		return
	}
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

func (s *Services) handlePieceworkDaySettle(c *gin.Context) bool {
	body := bindBody(c)
	bizDate := strings.TrimSpace(strOrDef(body["biz_date"], time.Now().Format("2006-01-02")))
	batchNo := fmt.Sprintf("DS%s-%d", strings.ReplaceAll(bizDate, "-", ""), time.Now().Unix()%1e6)
	rows, err := s.DB.Query(`SELECT i.id, i.board_id, COALESCE(i.board_code,''), COALESCE(i.trace_code,''),
		i.process_id, COALESCE(i.step_id,0), i.worker_id,
		(i.issue_kg - i.returned_kg - COALESCE(i.wage_settled_kg,0)) AS rem
		FROM pd_process_issue i
		INNER JOIN hr_employee e ON e.id=i.worker_id AND COALESCE(e.emp_type,'')='piece'
		INNER JOIN pd_process p ON p.id=i.process_id
		WHERE COALESCE(i.worker_id,0)>0
		  AND COALESCE(i.biz_status,'open')='work_done'
		  AND (i.issue_kg - i.returned_kg - COALESCE(i.wage_settled_kg,0)) > 0
		  AND (COALESCE(NULLIF(p.pay_mode,''),'') IN ('weight','piece') OR (COALESCE(NULLIF(p.pay_mode,''),'')='' AND COALESCE(p.is_piecework,0)=1))
		ORDER BY i.worker_id, i.process_id, i.id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	type row struct {
		id, boardID, processID, stepID, workerID int64
		boardCode, trace                         string
		rem                                      float64
	}
	list := []row{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.boardID, &r.boardCode, &r.trace, &r.processID, &r.stepID, &r.workerID, &r.rem); err != nil {
			continue
		}
		r.rem = roundKg(r.rem)
		if r.rem > kgEps {
			list = append(list, r)
		}
	}
	settledN, settledKg, settledAmt := 0, 0.0, 0.0
	actor := claimsUserID(c)
	for _, r := range list {
		rate := s.processWageRate(r.processID)
		amt := roundMoney(r.rem * rate)
		src := fmt.Sprintf("DAY:%s:I%d", bizDate, r.id)
		s.upsertPieceworkSummaryKeyedOnDate(r.workerID, r.processID, bizDate, src, r.rem, r.rem, r.rem, 0, 1)
		var settled float64
		_ = s.DB.QueryRow(`SELECT COALESCE(wage_settled_kg,0) FROM pd_process_issue WHERE id=?`, r.id).Scan(&settled)
		newSettled := roundKg(settled + r.rem)
		_, _ = s.DB.Exec(`UPDATE pd_process_issue SET wage_settled_kg=?, updated_at=NOW() WHERE id=?`, newSettled, r.id)
		var summaryID int64
		_ = s.DB.QueryRow(`SELECT id FROM pd_piecework_summary WHERE worker_id=? AND process_id=? AND biz_date=?`,
			r.workerID, r.processID, bizDate).Scan(&summaryID)
		s.appendStationFlowLog(stationFlowEvent{
			EventType: "day_settle", BizDate: bizDate, BoardID: r.boardID, BoardCode: r.boardCode, TraceCode: r.trace,
			ProcessID: r.processID, StepID: r.stepID, WorkerID: r.workerID, ActorUserID: actor,
			Kg: r.rem, PayMode: s.processPayMode(r.processID), EmpType: "piece",
			Rate: rate, Amount: amt, RefType: "pd_piecework_summary", RefID: summaryID,
			Payload: gin.H{"batch_no": batchNo, "issue_id": r.id, "biz_date": bizDate},
		})
		settledN++
		settledKg = roundKg(settledKg + r.rem)
		settledAmt = roundMoney(settledAmt + amt)
	}
	s.writeAuditCtx(c, "piecework_day_settle", 0, "day_settle", bizDate, nil, gin.H{
		"biz_date": bizDate, "batch_no": batchNo, "rows": settledN, "kg": settledKg, "amount": settledAmt,
	})
	api.OK(c, gin.H{
		"biz_date": bizDate, "batch_no": batchNo, "settled_rows": settledN,
		"settled_kg": settledKg, "settled_amount": settledAmt,
	})
	return true
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
