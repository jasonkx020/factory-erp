package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleOutboundSettles(c *gin.Context, method, action string) bool {
	if action == "delete" {
		return s.refuseDelete(c)
	}
	if strings.Contains(c.Request.URL.Path, "/close") || strings.HasSuffix(action, "close") {
		return s.closeOutboundSettle(c)
	}
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listOutboundSettles(c)
	case action == "create":
		return s.createOutboundSettle(c)
	case action == "get":
		m := s.loadOutboundSettle(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "update":
		return s.updateOutboundSettle(c)
	}
	return false
}

func (s *Services) listOutboundSettles(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_outbound_settle WHERE COALESCE(is_deleted,0)=0`).Scan(&total)
	rows, err := s.DB.Query(`SELECT id, doc_no, biz_date, COALESCE(product_id,0), COALESCE(product_name,''), COALESCE(plate_no,''),
		COALESCE(driver_name,''), COALESCE(trace_code,''), COALESCE(produce_date,''), qty, weight, COALESCE(unit,'kg'),
		freight_fee, loading_fee, weigh_fee, unit_price, goods_amount, amount, status, COALESCE(remark,''), created_at
		FROM sl_outbound_settle WHERE COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, productID int64
		var docNo, bizDate, pname, plate, driver, trace, produce, unit, status, remark, created string
		var qty, weight, freight, loading, weighFee, price, goods, amount float64
		_ = rows.Scan(&id, &docNo, &bizDate, &productID, &pname, &plate, &driver, &trace, &produce, &qty, &weight, &unit,
			&freight, &loading, &weighFee, &price, &goods, &amount, &status, &remark, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "biz_date": bizDate, "product_id": productID, "product_name": pname,
			"plate_no": plate, "driver_name": driver, "trace_code": trace, "produce_date": produce,
			"qty": qty, "weight": weight, "unit": unit, "freight_fee": freight, "loading_fee": loading, "weigh_fee": weighFee,
			"unit_price": price, "goods_amount": goods, "amount": amount, "status": status, "remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createOutboundSettle(c *gin.Context) bool {
	body := bindBody(c)
	weight, _ := asFloat(body["weight"])
	qty, _ := asFloat(body["qty"])
	if qty <= 0 {
		qty = weight
	}
	unitPrice, _ := asFloat(body["unit_price"])
	freight, loading, weighFee, _, _, plate, _ := feeFieldsFromBody(body)
	goods, total := settleAmount(weight, unitPrice, freight, loading, weighFee)
	docNo := fmt.Sprintf("OS%s", time.Now().Format("20060102150405"))
	res, err := s.DB.Exec(`INSERT INTO sl_outbound_settle(doc_no, biz_date, product_id, product_name, plate_no, driver_name,
		trace_code, produce_date, qty, weight, unit, freight_fee, loading_fee, weigh_fee, unit_price, goods_amount, amount, status, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'draft',?)`,
		docNo, strOrDef(body["biz_date"], time.Now().Format("2006-01-02")), nullIf0(asInt64Or0(body["product_id"])),
		strOr(body["product_name"]), plate, strOr(body["driver_name"]), strOr(body["trace_code"]), strOr(body["produce_date"]),
		qty, weight, strOrDef(body["unit"], "kg"), freight, loading, weighFee, unitPrice, goods, total, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadOutboundSettle(id))
	return true
}

func asInt64Or0(v interface{}) int64 {
	n, _ := asInt64(v)
	return n
}

func (s *Services) updateOutboundSettle(c *gin.Context) bool {
	id := paramID(c)
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM sl_outbound_settle WHERE id=?`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status != "draft" {
		api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
		return true
	}
	body := bindBody(c)
	weight := asFloatOr0(body["weight"])
	unitPrice := asFloatOr0(body["unit_price"])
	freight, loading, weighFee, _, _, plate, _ := feeFieldsFromBody(body)
	goods, total := settleAmount(weight, unitPrice, freight, loading, weighFee)
	_, err := s.DB.Exec(`UPDATE sl_outbound_settle SET product_name=COALESCE(NULLIF(?,''),product_name), plate_no=?,
		driver_name=COALESCE(NULLIF(?,''),driver_name), trace_code=COALESCE(NULLIF(?,''),trace_code),
		produce_date=COALESCE(NULLIF(?,''),produce_date), qty=?, weight=?, unit=COALESCE(NULLIF(?,''),unit),
		freight_fee=?, loading_fee=?, weigh_fee=?, unit_price=?, goods_amount=?, amount=?,
		remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=?`,
		strOr(body["product_name"]), plate, strOr(body["driver_name"]), strOr(body["trace_code"]), strOr(body["produce_date"]),
		asFloatOr0(body["qty"]), weight, strOr(body["unit"]), freight, loading, weighFee, unitPrice, goods, total,
		strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadOutboundSettle(id))
	return true
}

func (s *Services) closeOutboundSettle(c *gin.Context) bool {
	id := paramID(c)
	_, err := s.DB.Exec(`UPDATE sl_outbound_settle SET status='closed', updated_at=datetime('now') WHERE id=? AND status='draft'`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadOutboundSettle(id))
	return true
}

func (s *Services) loadOutboundSettle(id int64) gin.H {
	var productID int64
	var docNo, bizDate, pname, plate, driver, trace, produce, unit, status, remark, created string
	var qty, weight, freight, loading, weighFee, price, goods, amount float64
	err := s.DB.QueryRow(`SELECT doc_no, biz_date, COALESCE(product_id,0), COALESCE(product_name,''), COALESCE(plate_no,''),
		COALESCE(driver_name,''), COALESCE(trace_code,''), COALESCE(produce_date,''), qty, weight, COALESCE(unit,'kg'),
		freight_fee, loading_fee, weigh_fee, unit_price, goods_amount, amount, status, COALESCE(remark,''), created_at
		FROM sl_outbound_settle WHERE id=?`, id).
		Scan(&docNo, &bizDate, &productID, &pname, &plate, &driver, &trace, &produce, &qty, &weight, &unit,
			&freight, &loading, &weighFee, &price, &goods, &amount, &status, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "biz_date": bizDate, "product_id": productID, "product_name": pname,
		"plate_no": plate, "driver_name": driver, "trace_code": trace, "produce_date": produce,
		"qty": qty, "weight": weight, "unit": unit, "freight_fee": freight, "loading_fee": loading, "weigh_fee": weighFee,
		"unit_price": price, "goods_amount": goods, "amount": amount, "status": status, "remark": remark, "created_at": created,
	}
}

// --- piece issue sheet ---

func (s *Services) handlePieceIssueSheets(c *gin.Context, method, action string) bool {
	path := c.Request.URL.Path
	if strings.Contains(path, "/generate") || strings.HasSuffix(action, "generate") {
		return s.generatePieceIssueFromReports(c)
	}
	if action == "delete" {
		return s.refuseDelete(c)
	}
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listPieceIssueSheets(c)
	case action == "create":
		return s.createPieceIssueSheet(c)
	case action == "get":
		api.OK(c, s.loadPieceIssueSheet(paramID(c)))
		return true
	}
	return false
}

func (s *Services) listPieceIssueSheets(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_piece_issue_sheet`).Scan(&total)
	rows, err := s.DB.Query(`SELECT id, doc_no, biz_date, status, COALESCE(remark,''), created_at FROM pd_piece_issue_sheet
		ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var docNo, bizDate, status, remark, created string
		_ = rows.Scan(&id, &docNo, &bizDate, &status, &remark, &created)
		list = append(list, gin.H{"id": id, "doc_no": docNo, "biz_date": bizDate, "status": status, "remark": remark, "created_at": created})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createPieceIssueSheet(c *gin.Context) bool {
	body := bindBody(c)
	docNo := fmt.Sprintf("PI%s", time.Now().Format("20060102150405"))
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	res, err := s.DB.Exec(`INSERT INTO pd_piece_issue_sheet(doc_no, biz_date, status, remark) VALUES(?,?, 'draft',?)`,
		docNo, bizDate, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	sheetID, _ := res.LastInsertId()
	lines, _ := body["lines"].([]interface{})
	for i, raw := range lines {
		m, _ := raw.(map[string]interface{})
		if m == nil {
			continue
		}
		qty := asFloatOr0(m["qty"])
		reject := asFloatOr0(m["reject_qty"])
		price := asFloatOr0(m["unit_price"])
		totalQty := qty - reject
		if totalQty < 0 {
			totalQty = 0
		}
		amt := totalQty * price
		_, _ = s.DB.Exec(`INSERT INTO pd_piece_issue_line(sheet_id, seq_no, employee_id, employee_name, process_id, process_name,
			process_kind, unit_price, qty, reject_qty, qty_total, amount, remark)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			sheetID, i+1, nullIf0(asInt64Or0(m["employee_id"])), strOr(m["employee_name"]), nullIf0(asInt64Or0(m["process_id"])),
			strOr(m["process_name"]), strOrDef(m["process_kind"], "piece"), price, qty, reject, totalQty, amt, strOr(m["remark"]))
	}
	api.OK(c, s.loadPieceIssueSheet(sheetID))
	return true
}

func (s *Services) generatePieceIssueFromReports(c *gin.Context) bool {
	body := bindBody(c)
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	docNo := fmt.Sprintf("PI%s", time.Now().Format("20060102150405"))
	res, err := s.DB.Exec(`INSERT INTO pd_piece_issue_sheet(doc_no, biz_date, status, remark) VALUES(?,?, 'draft',?)`,
		docNo, bizDate, "generated from report-works")
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	sheetID, _ := res.LastInsertId()
	rows, err := s.DB.Query(`SELECT COALESCE(r.worker_id,0), COALESCE(e.name,''), COALESCE(r.process_id,0), COALESCE(p.name,''),
		COALESCE(r.output_weight, r.qty_net, r.qty, 0), COALESCE(r.loss,0)
		FROM pd_report_work r
		LEFT JOIN hr_employee e ON e.id=r.worker_id
		LEFT JOIN pd_process p ON p.id=r.process_id
		WHERE date(COALESCE(r.confirmed_at, r.reported_at))=date(?)
		LIMIT 200`, bizDate)
	if err == nil {
		defer rows.Close()
		seq := 1
		for rows.Next() {
			var empID, procID int64
			var ename, pname string
			var qty, loss float64
			_ = rows.Scan(&empID, &ename, &procID, &pname, &qty, &loss)
			var price float64
			_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, procID).Scan(&price)
			totalQty := qty - loss
			if totalQty < 0 {
				totalQty = 0
			}
			_, _ = s.DB.Exec(`INSERT INTO pd_piece_issue_line(sheet_id, seq_no, employee_id, employee_name, process_id, process_name,
				process_kind, unit_price, qty, reject_qty, qty_total, amount) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
				sheetID, seq, nullIf0(empID), ename, nullIf0(procID), pname, "piece", price, qty, loss, totalQty, totalQty*price)
			seq++
		}
	}
	api.OK(c, s.loadPieceIssueSheet(sheetID))
	return true
}

func (s *Services) loadPieceIssueSheet(id int64) gin.H {
	var docNo, bizDate, status, remark, created string
	err := s.DB.QueryRow(`SELECT doc_no, biz_date, status, COALESCE(remark,''), created_at FROM pd_piece_issue_sheet WHERE id=?`, id).
		Scan(&docNo, &bizDate, &status, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	rows, _ := s.DB.Query(`SELECT id, seq_no, COALESCE(employee_id,0), COALESCE(employee_name,''), COALESCE(process_id,0),
		COALESCE(process_name,''), COALESCE(process_kind,'piece'), unit_price, qty, reject_qty, qty_total, amount, COALESCE(remark,'')
		FROM pd_piece_issue_line WHERE sheet_id=? ORDER BY seq_no`, id)
	lines := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, seq, empID, procID int64
			var ename, pname, kind, rem string
			var price, qty, reject, total, amt float64
			_ = rows.Scan(&lid, &seq, &empID, &ename, &procID, &pname, &kind, &price, &qty, &reject, &total, &amt, &rem)
			lines = append(lines, gin.H{
				"id": lid, "seq_no": seq, "employee_id": empID, "employee_name": ename, "process_id": procID,
				"process_name": pname, "process_kind": kind, "unit_price": price, "qty": qty, "reject_qty": reject,
				"qty_total": total, "amount": amt, "remark": rem,
			})
		}
	}
	return gin.H{"id": id, "doc_no": docNo, "biz_date": bizDate, "status": status, "remark": remark, "created_at": created, "lines": lines}
}

// --- tools ---

func (s *Services) handleToolItems(c *gin.Context, method, action string) bool {
	if action == "list" || method == "GET" {
		rows, err := s.DB.Query(`SELECT id, code, name, status FROM hr_tool_item WHERE status='active' ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, status string
			_ = rows.Scan(&id, &code, &name, &status)
			list = append(list, gin.H{"id": id, "code": code, "name": name, "status": status})
		}
		api.PageOK(c, list, len(list), 1, len(list)+1)
		return true
	}
	return false
}

func (s *Services) handleToolIssues(c *gin.Context, method, action string) bool {
	if action == "delete" {
		return s.refuseDelete(c)
	}
	if strings.Contains(c.Request.URL.Path, "/return") || strings.HasSuffix(action, "return") {
		return s.returnToolIssue(c)
	}
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_tool_issue`).Scan(&total)
		rows, err := s.DB.Query(`SELECT id, doc_no, biz_date, seq_no, COALESCE(employee_id,0), COALESCE(employee_name,''),
			tool_item_id, COALESCE(tool_name,''), issue_qty, return_qty, total_qty, status, COALESCE(remark,''), created_at
			FROM hr_tool_issue ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, seq, empID, toolID int64
			var docNo, bizDate, ename, tname, status, remark, created string
			var issue, ret, totalQty float64
			_ = rows.Scan(&id, &docNo, &bizDate, &seq, &empID, &ename, &toolID, &tname, &issue, &ret, &totalQty, &status, &remark, &created)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "biz_date": bizDate, "seq_no": seq, "employee_id": empID, "employee_name": ename,
				"tool_item_id": toolID, "tool_name": tname, "issue_qty": issue, "return_qty": ret, "total_qty": totalQty,
				"status": status, "remark": remark, "created_at": created,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case action == "create":
		body := bindBody(c)
		toolID, _ := asInt64(body["tool_item_id"])
		var toolName string
		_ = s.DB.QueryRow(`SELECT name FROM hr_tool_item WHERE id=?`, toolID).Scan(&toolName)
		if toolName == "" {
			toolName = strOr(body["tool_name"])
		}
		issue := asFloatOr0(body["issue_qty"])
		docNo := fmt.Sprintf("TI%s", time.Now().Format("20060102150405"))
		res, err := s.DB.Exec(`INSERT INTO hr_tool_issue(doc_no, biz_date, seq_no, employee_id, employee_name, tool_item_id, tool_name,
			issue_qty, return_qty, total_qty, status, remark) VALUES(?,?,?,?,?,?,?,?,0,?, 'open',?)`,
			docNo, strOrDef(body["biz_date"], time.Now().Format("2006-01-02")), asInt64Or0(body["seq_no"])+1,
			nullIf0(asInt64Or0(body["employee_id"])), strOr(body["employee_name"]), toolID, toolName, issue, issue, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "issue_qty": issue, "total_qty": issue})
		return true
	}
	return false
}

func (s *Services) returnToolIssue(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	ret := asFloatOr0(body["return_qty"])
	var issue float64
	_ = s.DB.QueryRow(`SELECT issue_qty FROM hr_tool_issue WHERE id=?`, id).Scan(&issue)
	total := issue - ret
	if total < 0 {
		total = 0
	}
	status := "open"
	if total == 0 {
		status = "returned"
	}
	_, err := s.DB.Exec(`UPDATE hr_tool_issue SET return_qty=?, total_qty=?, status=? WHERE id=?`, ret, total, status, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, gin.H{"id": id, "return_qty": ret, "total_qty": total, "status": status})
	return true
}

// --- weighbridge ---

func (s *Services) handleWeighbridges(c *gin.Context, method, action string) bool {
	if action == "delete" {
		return s.refuseDelete(c)
	}
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(location,''), status, COALESCE(remark,''), created_at FROM inv_weighbridge ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, loc, status, remark, created string
			_ = rows.Scan(&id, &code, &name, &loc, &status, &remark, &created)
			list = append(list, gin.H{"id": id, "code": code, "name": name, "location": loc, "status": status, "remark": remark, "created_at": created})
		}
		api.PageOK(c, list, len(list), 1, len(list)+1)
		return true
	case action == "create":
		body := bindBody(c)
		code := strOrDef(body["code"], fmt.Sprintf("WB%d", time.Now().Unix()%100000))
		res, err := s.DB.Exec(`INSERT INTO inv_weighbridge(code, name, location, status, remark) VALUES(?,?,?,?,?)`,
			code, strOr(body["name"]), strOr(body["location"]), strOrDef(body["status"], "active"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code})
		return true
	case action == "update":
		body := bindBody(c)
		id := paramID(c)
		_, err := s.DB.Exec(`UPDATE inv_weighbridge SET name=COALESCE(NULLIF(?,''),name), location=COALESCE(NULLIF(?,''),location),
			status=COALESCE(NULLIF(?,''),status), remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
			strOr(body["name"]), strOr(body["location"]), strOr(body["status"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id})
		return true
	}
	return false
}

// list report works for process ledger
func (s *Services) listProcessReports(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	scrapType := c.Query("scrap_type")
	where := `WHERE 1=1`
	args := []interface{}{}
	if scrapType != "" {
		where += ` AND EXISTS (SELECT 1 FROM pd_scrap_record s WHERE s.doc_no LIKE 'SCR-'||r.id||'%' AND s.scrap_type=?)`
		args = append(args, scrapType)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_report_work r `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT r.id, COALESCE(r.doc_no,''), COALESCE(r.status,''), COALESCE(r.process_id,0), COALESCE(p.name,''),
		COALESCE(r.input_weight,0), COALESCE(r.output_weight,0), COALESCE(r.loss,0), COALESCE(r.bag_qty,0),
		COALESCE(r.confirmed_at, r.reported_at,''), COALESCE(r.process_qc_result,'')
		FROM pd_report_work r LEFT JOIN pd_process p ON p.id=r.process_id `+where+`
		ORDER BY r.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, procID int64
		var docNo, status, pname, at, qc string
		var inW, outW, loss, bag float64
		_ = rows.Scan(&id, &docNo, &status, &procID, &pname, &inW, &outW, &loss, &bag, &at, &qc)
		var scrapTypeVal string
		_ = s.DB.QueryRow(`SELECT COALESCE(scrap_type,'') FROM pd_scrap_record WHERE doc_no LIKE ? ORDER BY id DESC LIMIT 1`, fmt.Sprintf("SCR-%d%%", id)).Scan(&scrapTypeVal)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "status": status, "process_id": procID, "process_name": pname,
			"input_weight": inW, "output_weight": outW, "loss": loss, "bag_qty": bag, "reported_at": at,
			"process_qc_result": qc, "scrap_type": scrapTypeVal,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}
