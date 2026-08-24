package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// isCassavaReportPath 木薯产线允许的报表 API（销售/CRM/三表等仍下线）。
func isCassavaReportPath(openapiPath string) bool {
	allowed := []string{
		"/api/v1/report/dashboards/production",
		"/api/v1/report/dashboards/live",
		"/api/v1/report/dashboards/warehouse",
		"/api/v1/report/daily",
		"/api/v1/report/inbound-daily",
		"/api/v1/report/piecework-daily",
		"/api/v1/report/yield-analysis",
		"/api/v1/report/trace-progress",
		"/api/v1/report/farmer-settlement-summary",
		"/api/v1/report/stock-ledger",
		"/api/v1/report/qc",
		"/api/v1/report/payroll-reconcile",
		"/api/v1/report/cost-period-summary",
	}
	for _, p := range allowed {
		if openapiPath == p || strings.HasPrefix(openapiPath, p+"/") {
			return true
		}
	}
	return false
}

func (s *Services) reportWarehouseDashboard(c *gin.Context) bool {
	list := []gin.H{}
	rows, err := s.DB.Query(`SELECT w.id, COALESCE(w.name,''), COALESCE(w.warehouse_type,''),
		COALESCE(SUM(b.qty),0), COUNT(DISTINCT CASE WHEN b.qty>0 THEN b.product_id END)
		FROM inv_warehouse w
		LEFT JOIN inv_balance b ON b.warehouse_id=w.id
		WHERE COALESCE(w.is_deleted,0)=0
		GROUP BY w.id, w.name, w.warehouse_type
		ORDER BY w.id`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var name, whType string
			var qty float64
			var sku int
			_ = rows.Scan(&id, &name, &whType, &qty, &sku)
			list = append(list, gin.H{
				"warehouse_id": id, "warehouse_name": name, "warehouse_type": whType,
				"qty_kg": qty, "sku_count": sku,
			})
		}
	}
	shortage := s.queryCount(`SELECT COUNT(1) FROM inv_stock_alert_rule r
		JOIN inv_balance b ON b.product_id=r.product_id AND (r.warehouse_id IS NULL OR b.warehouse_id=r.warehouse_id)
		WHERE r.alert_type='shortage' AND COALESCE(r.is_enabled,1)=1 AND b.qty < COALESCE(r.min_qty,0)`)
	excess := s.queryCount(`SELECT COUNT(1) FROM inv_stock_alert_rule r
		JOIN inv_balance b ON b.product_id=r.product_id AND (r.warehouse_id IS NULL OR b.warehouse_id=r.warehouse_id)
		WHERE r.alert_type='excess' AND COALESCE(r.is_enabled,1)=1 AND r.max_qty IS NOT NULL AND b.qty > r.max_qty`)
	api.OK(c, gin.H{
		"title": "三仓库存概览", "list": list, "total": len(list),
		"summary": gin.H{
			"shortage_alerts": shortage,
			"excess_alerts":   excess,
			"total_qty_kg":    s.queryFloat(`SELECT COALESCE(SUM(qty),0) FROM inv_balance`),
		},
		"as_of": reportToday(),
	})
	return true
}

func (s *Services) reportInboundDaily(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	if bizDate == "" {
		bizDate = reportToday()
	}
	ticketCount := s.queryCount(`SELECT COUNT(1) FROM pur_weigh_ticket WHERE COALESCE(is_deleted,0)=0 AND biz_date=?`, bizDate)
	gross := s.queryFloat(`SELECT COALESCE(SUM(gross_weight),0) FROM pur_weigh_ticket WHERE COALESCE(is_deleted,0)=0 AND biz_date=?`, bizDate)
	deduct := s.queryFloat(`SELECT COALESCE(SUM(deduct_weight),0) FROM pur_weigh_ticket WHERE COALESCE(is_deleted,0)=0 AND biz_date=?`, bizDate)
	net := s.queryFloat(`SELECT COALESCE(SUM(net_weight),0) FROM pur_weigh_ticket WHERE COALESCE(is_deleted,0)=0 AND biz_date=?`, bizDate)
	settleAmt := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM pur_farmer_settlement WHERE biz_date=?`, bizDate)
	settlePending := s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM pur_farmer_settlement WHERE biz_date=? AND status NOT IN ('paid','settle_paid')`, bizDate)

	summary := gin.H{
		"biz_date": bizDate, "ticket_count": ticketCount,
		"gross_kg": gross, "deduct_kg": deduct, "net_kg": net,
		"settlement_amount": settleAmt, "settlement_pending": settlePending,
	}
	list := []gin.H{summary}

	rows, err := s.DB.Query(`SELECT t.id, t.doc_no, COALESCE(f.name,'') AS farmer_name, t.gross_weight, t.deduct_weight, t.net_weight,
		t.qc_result, t.status, COALESCE(t.trace_code,'')
		FROM pur_weigh_ticket t
		LEFT JOIN pur_farmer f ON f.id=t.farmer_id
		WHERE COALESCE(t.is_deleted,0)=0 AND t.biz_date=?
		ORDER BY t.id DESC LIMIT 200`, bizDate)
	if err == nil {
		defer rows.Close()
		detail, _ := rowsToMaps(rows)
		if len(detail) > 0 {
			for _, row := range detail {
				list = append(list, gin.H(row))
			}
		}
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": summary})
	return true
}

func (s *Services) reportPieceworkDaily(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	if bizDate == "" {
		bizDate = reportToday()
	}
	summary := gin.H{
		"biz_date":      bizDate,
		"worker_count":  s.queryCount(`SELECT COUNT(DISTINCT worker_id) FROM pd_piecework_summary WHERE biz_date=?`, bizDate),
		"total_qty_kg":  s.queryFloat(`SELECT COALESCE(SUM(qty),0) FROM pd_piecework_summary WHERE biz_date=?`, bizDate),
		"total_amount":  s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE biz_date=?`, bizDate),
		"flow_log_kg":   s.queryFloat(`SELECT COALESCE(SUM(kg),0) FROM pd_station_flow_log WHERE biz_date=?`, bizDate),
		"flow_log_rows": s.queryCount(`SELECT COUNT(1) FROM pd_station_flow_log WHERE biz_date=?`, bizDate),
	}
	list := []gin.H{}
	rows, err := s.DB.Query(`SELECT s.worker_id, COALESCE(e.name,'') AS worker_name, s.process_id, COALESCE(p.name,'') AS process_name,
		s.qty, s.amount
		FROM pd_piecework_summary s
		LEFT JOIN hr_employee e ON e.id=s.worker_id
		LEFT JOIN pd_process p ON p.id=s.process_id
		WHERE s.biz_date=?
		ORDER BY s.amount DESC, s.qty DESC LIMIT 500`, bizDate)
	if err == nil {
		defer rows.Close()
		maps, _ := rowsToMaps(rows)
		for _, row := range maps {
			list = append(list, gin.H(row))
		}
	}
	if len(list) == 0 {
		list = []gin.H{summary}
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": summary})
	return true
}

func (s *Services) reportYieldAnalysis(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	trace := strings.TrimSpace(c.Query("trace_code"))
	where := " WHERE 1=1 "
	args := []interface{}{}
	if bizDate != "" {
		where += " AND date(y.created_at)=? "
		args = append(args, bizDate)
	}
	if trace != "" {
		where += " AND UPPER(y.trace_code)=UPPER(?) "
		args = append(args, trace)
	}
	rows, err := s.DB.Query(`SELECT y.process_id, COALESCE(p.name,'') AS process_name,
		COUNT(DISTINCT y.trace_code) AS trace_count,
		COALESCE(SUM(y.input_kg),0) AS input_kg,
		COALESCE(SUM(y.output_kg),0) AS output_kg,
		COALESCE(SUM(y.loss_kg),0) AS loss_kg,
		CASE WHEN SUM(y.input_kg)>0 THEN SUM(y.loss_kg)/SUM(y.input_kg) ELSE 0 END AS loss_rate
		FROM pd_trace_process_yield y
		LEFT JOIN pd_process p ON p.id=y.process_id`+where+`
		GROUP BY y.process_id, p.name
		ORDER BY y.process_id`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportTraceProgress(c *gin.Context) bool {
	status := strings.TrimSpace(c.Query("status"))
	where := " WHERE 1=1 "
	args := []interface{}{}
	if status != "" {
		where += " AND tp.status=? "
		args = append(args, status)
	}
	rows, err := s.DB.Query(`SELECT tp.id, tp.trace_code, tp.status,
		COALESCE(tp.input_kg,0) AS input_kg, COALESCE(tp.output_kg,0) AS output_kg, COALESCE(tp.loss_rate,0) AS loss_rate,
		CAST(tp.started_at AS TEXT) AS started_at, CAST(tp.completed_at AS TEXT) AS completed_at,
		(SELECT COUNT(1) FROM pd_process_issue pi WHERE UPPER(pi.trace_code)=UPPER(tp.trace_code) AND pi.status='open') AS open_issues
		FROM pd_trace_production tp`+where+`
		ORDER BY tp.id DESC LIMIT 300`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) reportFarmerSettlementSummary(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	where := " WHERE 1=1 "
	args := []interface{}{}
	if bizDate != "" {
		where += " AND s.biz_date=? "
		args = append(args, bizDate)
	}
	summary := gin.H{
		"total_amount": s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM pur_farmer_settlement`+where, args...),
		"paid_amount": s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM pur_farmer_settlement`+where+` AND status IN ('paid','settle_paid')`, args...),
		"pending_amount": s.queryFloat(`SELECT COALESCE(SUM(amount),0) FROM pur_farmer_settlement`+where+` AND status NOT IN ('paid','settle_paid')`, args...),
		"doc_count": s.queryCount(`SELECT COUNT(1) FROM pur_farmer_settlement`+where, args...),
	}
	rows, err := s.DB.Query(`SELECT s.id, s.doc_no, s.biz_date, COALESCE(f.name,'') AS farmer_name,
		s.net_weight, s.unit_price, s.amount, s.status, COALESCE(wt.trace_code,'') AS trace_code
		FROM pur_farmer_settlement s
		LEFT JOIN pur_farmer f ON f.id=s.farmer_id
		LEFT JOIN pur_weigh_ticket wt ON wt.id=s.weigh_ticket_id`+where+`
		ORDER BY s.id DESC LIMIT 300`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": summary})
	return true
}

func (s *Services) reportIncomingQC(c *gin.Context) bool {
	list := []gin.H{}
	rows, err := s.DB.Query(`SELECT id, doc_no, product_id, qty_check, qty_pass, qty_fail, result, status, created_at
		FROM pur_incoming_qc WHERE COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 200`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, pid int64
			var docNo, result, status, created string
			var check, pass, fail float64
			var resultN, statusN sql.NullString
			_ = rows.Scan(&id, &docNo, &pid, &check, &pass, &fail, &resultN, &statusN, &created)
			if resultN.Valid {
				result = resultN.String
			}
			if statusN.Valid {
				status = statusN.String
			}
			list = append(list, gin.H{
				"id": id, "source": "incoming", "doc_no": docNo, "product_id": pid,
				"qty_check": check, "qty_pass": pass, "qty_fail": fail,
				"result": result, "status": status, "created_at": created,
			})
		}
	}
	passN := 0
	for _, row := range list {
		if strings.EqualFold(strOr(row["result"]), "pass") {
			passN++
		}
	}
	api.OK(c, gin.H{
		"list": list, "total": len(list),
		"summary": gin.H{"total": len(list), "pass_count": passN, "fail_count": len(list) - passN},
	})
	return true
}

func (s *Services) reportPayrollReconcile(c *gin.Context) bool {
	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	var year, month int
	_, _ = fmt.Sscanf(period, "%d-%d", &year, &month)
	if year == 0 || month == 0 {
		api.FailJSON(c, "INVALID_PERIOD")
		return true
	}
	var sheetID int64
	var sheetNo, sheetStatus string
	_ = s.DB.QueryRow(`SELECT id, doc_no, status FROM pay_payroll_sheet WHERE period_year=? AND period_month=? ORDER BY id DESC LIMIT 1`,
		year, month).Scan(&sheetID, &sheetNo, &sheetStatus)

	list := []gin.H{}
	diffCount := 0
	var sheetTotal, pieceworkTotal float64

	if sheetID > 0 {
		rows, err := s.DB.Query(`SELECT l.employee_id, COALESCE(e.name,'') AS worker_name, COALESCE(e.emp_no,'') AS emp_no,
			l.piece_amount AS sheet_piece_amount,
			(SELECT COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE worker_id=l.employee_id AND biz_date LIKE ?) AS piecework_amount,
			l.total_amount AS sheet_total
			FROM pay_payroll_sheet_line l
			LEFT JOIN hr_employee e ON e.id=l.employee_id
			WHERE l.sheet_id=?
			ORDER BY ABS(l.piece_amount - (SELECT COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE worker_id=l.employee_id AND biz_date LIKE ?)) DESC,
			l.piece_amount DESC`, period+"%", sheetID, period+"%")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var eid int64
				var name, empNo string
				var sheetPiece, pwAmt, sheetLineTotal float64
				_ = rows.Scan(&eid, &name, &empNo, &sheetPiece, &pwAmt, &sheetLineTotal)
				diff := sheetPiece - pwAmt
				if diff < -0.01 || diff > 0.01 {
					diffCount++
				}
				sheetTotal += sheetPiece
				pieceworkTotal += pwAmt
				list = append(list, gin.H{
					"employee_id": eid, "worker_name": name, "emp_no": empNo,
					"sheet_piece_amount": sheetPiece, "piecework_amount": pwAmt, "diff": diff,
					"sheet_total": sheetLineTotal,
				})
			}
		}
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(piece_amount),0) FROM pay_payroll_sheet_line WHERE sheet_id=?`,
			sheetID).Scan(&sheetTotal)
	} else {
		rows, err := s.DB.Query(`SELECT s.worker_id AS employee_id, COALESCE(e.name,'') AS worker_name, COALESCE(e.emp_no,'') AS emp_no,
			0 AS sheet_piece_amount, COALESCE(SUM(s.amount),0) AS piecework_amount, 0 AS sheet_total
			FROM pd_piecework_summary s
			LEFT JOIN hr_employee e ON e.id=s.worker_id
			WHERE s.biz_date LIKE ?
			GROUP BY s.worker_id, e.name, e.emp_no
			ORDER BY piecework_amount DESC`, period+"%")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var eid int64
				var name, empNo string
				var sheetPiece, pwAmt, sheetLineTotal float64
				_ = rows.Scan(&eid, &name, &empNo, &sheetPiece, &pwAmt, &sheetLineTotal)
				pieceworkTotal += pwAmt
				list = append(list, gin.H{
					"employee_id": eid, "worker_name": name, "emp_no": empNo,
					"sheet_piece_amount": sheetPiece, "piecework_amount": pwAmt, "diff": sheetPiece - pwAmt,
					"sheet_total": sheetLineTotal,
				})
			}
		}
	}

	summary := gin.H{
		"period": period, "sheet_id": sheetID, "sheet_no": sheetNo, "sheet_status": sheetStatus,
		"worker_count": len(list), "diff_count": diffCount,
		"sheet_piece_total": sheetTotal, "piecework_total": pieceworkTotal,
		"diff_total": sheetTotal - pieceworkTotal,
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": summary})
	return true
}

func (s *Services) reportCostPeriodSummary(c *gin.Context) bool {
	period := strings.TrimSpace(c.Query("period"))
	where := ""
	args := []interface{}{}
	if period != "" {
		where = " WHERE period=? "
		args = append(args, period)
	}
	summary := gin.H{
		"period":         period,
		"doc_count":      s.queryCount(`SELECT COUNT(1) FROM fin_cost_accounting`+where, args...),
		"material_total": s.queryFloat(`SELECT COALESCE(SUM(material_cost),0) FROM fin_cost_accounting`+where, args...),
		"labor_total":    s.queryFloat(`SELECT COALESCE(SUM(labor_cost),0) FROM fin_cost_accounting`+where, args...),
		"overhead_total": s.queryFloat(`SELECT COALESCE(SUM(overhead),0) FROM fin_cost_accounting`+where, args...),
		"cost_total":     s.queryFloat(`SELECT COALESCE(SUM(total_cost),0) FROM fin_cost_accounting`+where, args...),
	}
	rows, err := s.DB.Query(`SELECT c.id, c.doc_no, c.period, COALESCE(p.name,'') AS product_name,
		c.material_cost, c.labor_cost, c.overhead, c.total_cost, c.status
		FROM fin_cost_accounting c
		LEFT JOIN prd_product p ON p.id=c.product_id`+where+`
		ORDER BY c.period DESC, c.id DESC LIMIT 300`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	maps, _ := rowsToMaps(rows)
	list := make([]gin.H, 0, len(maps))
	for _, row := range maps {
		list = append(list, gin.H(row))
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "summary": summary})
	return true
}
