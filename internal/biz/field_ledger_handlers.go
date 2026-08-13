package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
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
	EnsureTicketSchema(s.DB)
	migrateToolIssueToLines(s.DB)
	path := c.Request.URL.Path
	if action == "delete" {
		return s.refuseDelete(c)
	}
	switch {
	case strings.Contains(path, "/approve") || strings.HasSuffix(action, "approve"):
		return s.approveToolIssue(c)
	case strings.Contains(path, "/reject") || strings.HasSuffix(action, "reject"):
		return s.rejectToolIssue(c)
	case strings.Contains(path, "/return-request") || strings.HasSuffix(action, "return-request"):
		return s.returnRequestToolIssue(c)
	case strings.Contains(path, "/return-confirm") || strings.HasSuffix(action, "return-confirm"):
		return s.returnConfirmToolIssue(c)
	case strings.Contains(path, "/return") || strings.HasSuffix(action, "return"):
		return s.returnToolIssue(c)
	case action == "list" || (method == "GET" && action != "get"):
		return s.listToolIssues(c)
	case action == "get":
		return s.getToolIssue(c)
	case action == "create":
		return s.createToolIssue(c)
	}
	return false
}

func (s *Services) listToolIssues(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if emp := c.Query("employee_id"); emp != "" {
		where += ` AND employee_id=?`
		args = append(args, emp)
	}
	if st := c.Query("status"); st != "" {
		where += ` AND status=?`
		args = append(args, st)
	}
	if c.Query("mine") == "1" {
		if cl := middleware.Claims(c); cl != nil {
			var empID int64
			_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, cl.UserID).Scan(&empID)
			if empID > 0 {
				where += ` AND employee_id=?`
				args = append(args, empID)
			}
		}
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM hr_tool_issue `+where, args...).Scan(&total)
	args2 := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, doc_no, biz_date, seq_no, COALESCE(employee_id,0), COALESCE(employee_name,''),
		status, COALESCE(remark,''), created_at, COALESCE(pending_return_qty,0), COALESCE(ticket_id,0)
		FROM hr_tool_issue `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args2...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	ids := []int64{}
	for rows.Next() {
		var id, seq, empID, ticketID int64
		var docNo, bizDate, ename, status, remark, created string
		var pending float64
		_ = rows.Scan(&id, &docNo, &bizDate, &seq, &empID, &ename, &status, &remark, &created, &pending, &ticketID)
		ids = append(ids, id)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "biz_date": bizDate, "seq_no": seq, "employee_id": empID, "employee_name": ename,
			"status": status, "remark": remark, "created_at": created, "pending_return_qty": pending, "ticket_id": ticketID,
		})
	}
	lineMap := s.loadToolIssueLinesMap(ids)
	for i := range list {
		id := asInt64Or0(list[i]["id"])
		lines := lineMap[id]
		if lines == nil {
			lines = []gin.H{}
		}
		list[i]["items"] = lines
		list[i]["items_summary"] = toolIssueItemsSummary(lines)
		issueQty, returnQty, totalQty := toolIssueQtyTotals(lines)
		list[i]["issue_qty"] = issueQty
		list[i]["return_qty"] = returnQty
		list[i]["total_qty"] = totalQty
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) getToolIssue(c *gin.Context) bool {
	id := paramID(c)
	m := s.loadToolIssue(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) loadToolIssue(id int64) gin.H {
	var seq, empID, ticketID int64
	var docNo, bizDate, ename, status, remark, created string
	var pending float64
	err := s.DB.QueryRow(`SELECT doc_no, biz_date, seq_no, COALESCE(employee_id,0), COALESCE(employee_name,''),
		status, COALESCE(remark,''), created_at, COALESCE(pending_return_qty,0), COALESCE(ticket_id,0)
		FROM hr_tool_issue WHERE id=?`, id).
		Scan(&docNo, &bizDate, &seq, &empID, &ename, &status, &remark, &created, &pending, &ticketID)
	if err != nil {
		return nil
	}
	lines := s.loadToolIssueLines(id)
	issueQty, returnQty, totalQty := toolIssueQtyTotals(lines)
	return gin.H{
		"id": id, "doc_no": docNo, "biz_date": bizDate, "seq_no": seq, "employee_id": empID, "employee_name": ename,
		"status": status, "remark": remark, "created_at": created, "pending_return_qty": pending, "ticket_id": ticketID,
		"items": lines, "items_summary": toolIssueItemsSummary(lines),
		"issue_qty": issueQty, "return_qty": returnQty, "total_qty": totalQty,
	}
}

func (s *Services) loadToolIssueLines(issueID int64) []gin.H {
	m := s.loadToolIssueLinesMap([]int64{issueID})
	if lines := m[issueID]; lines != nil {
		return lines
	}
	return []gin.H{}
}

func (s *Services) loadToolIssueLinesMap(issueIDs []int64) map[int64][]gin.H {
	out := map[int64][]gin.H{}
	if len(issueIDs) == 0 {
		return out
	}
	placeholders := make([]string, len(issueIDs))
	args := make([]interface{}, len(issueIDs))
	for i, id := range issueIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := s.DB.Query(`SELECT id, issue_id, tool_item_id, COALESCE(tool_name,''), issue_qty, return_qty, COALESCE(pending_return_qty,0)
		FROM hr_tool_issue_line WHERE issue_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY id`, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var lid, issueID, toolID int64
		var tname string
		var issue, ret, pending float64
		_ = rows.Scan(&lid, &issueID, &toolID, &tname, &issue, &ret, &pending)
		out[issueID] = append(out[issueID], gin.H{
			"id": lid, "line_id": lid, "issue_id": issueID, "tool_item_id": toolID, "tool_name": tname,
			"issue_qty": issue, "return_qty": ret, "pending_return_qty": pending,
			"total_qty": issue - ret,
		})
	}
	return out
}

func toolIssueItemsSummary(lines []gin.H) string {
	parts := make([]string, 0, len(lines))
	for _, ln := range lines {
		name := strOr(ln["tool_name"])
		qty := asFloatOr0(ln["issue_qty"])
		parts = append(parts, fmt.Sprintf("%s×%.0f", name, qty))
	}
	return strings.Join(parts, "、")
}

func toolIssueQtyTotals(lines []gin.H) (issueQty, returnQty, totalQty float64) {
	for _, ln := range lines {
		issueQty += asFloatOr0(ln["issue_qty"])
		returnQty += asFloatOr0(ln["return_qty"])
	}
	totalQty = issueQty - returnQty
	if totalQty < 0 {
		totalQty = 0
	}
	return
}

func (s *Services) nextToolSeq(bizDate string) int64 {
	var n int64
	_ = s.DB.QueryRow(`SELECT COALESCE(MAX(seq_no),0)+1 FROM hr_tool_issue WHERE biz_date=?`, bizDate).Scan(&n)
	if n <= 0 {
		n = 1
	}
	return n
}

type toolIssueLineInput struct {
	ToolItemID int64
	ToolName   string
	IssueQty   float64
}

func (s *Services) parseToolIssueItems(body map[string]interface{}) ([]toolIssueLineInput, string) {
	raw := coerceInterfaceSlice(body["items"])
	if len(raw) == 0 {
		// legacy single-tool body
		toolID := asInt64Or0(body["tool_item_id"])
		qty := asFloatOr0(body["issue_qty"])
		if toolID > 0 && qty > 0 {
			raw = []interface{}{map[string]interface{}{"tool_item_id": toolID, "issue_qty": qty, "tool_name": body["tool_name"]}}
		}
	}
	if len(raw) == 0 {
		return nil, "ITEMS_REQUIRED"
	}
	seen := map[int64]bool{}
	out := make([]toolIssueLineInput, 0, len(raw))
	for _, it := range raw {
		m := coerceStringMap(it)
		if m == nil {
			continue
		}
		toolID := asInt64Or0(m["tool_item_id"])
		qty := asFloatOr0(m["issue_qty"])
		if toolID <= 0 || qty <= 0 {
			return nil, "INVALID_ITEM"
		}
		if seen[toolID] {
			return nil, "DUPLICATE_TOOL"
		}
		seen[toolID] = true
		var toolName string
		_ = s.DB.QueryRow(`SELECT name FROM hr_tool_item WHERE id=?`, toolID).Scan(&toolName)
		if toolName == "" {
			toolName = strOr(m["tool_name"])
		}
		if toolName == "" {
			return nil, "TOOL_NOT_FOUND"
		}
		out = append(out, toolIssueLineInput{ToolItemID: toolID, ToolName: toolName, IssueQty: qty})
	}
	if len(out) == 0 {
		return nil, "ITEMS_REQUIRED"
	}
	return out, ""
}

func coerceInterfaceSlice(v interface{}) []interface{} {
	switch t := v.(type) {
	case []interface{}:
		return t
	case []map[string]interface{}:
		out := make([]interface{}, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out
	default:
		return nil
	}
}

func coerceStringMap(v interface{}) map[string]interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return t
	case gin.H:
		return map[string]interface{}(t)
	default:
		return nil
	}
}

func (s *Services) createToolIssue(c *gin.Context) bool {
	cl := middleware.Claims(c)
	body := bindBody(c)
	items, errCode := s.parseToolIssueItems(body)
	if errCode != "" {
		api.FailJSON(c, errCode)
		return true
	}
	empID := asInt64Or0(body["employee_id"])
	ename := strOr(body["employee_name"])
	if cl != nil && empID == 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(employee_id,0) FROM iam_user WHERE id=?`, cl.UserID).Scan(&empID)
	}
	if empID > 0 && ename == "" {
		_ = s.DB.QueryRow(`SELECT name FROM hr_employee WHERE id=?`, empID).Scan(&ename)
	}
	next, _ := asInt64(body["next_assignee_user_id"])
	if next <= 0 {
		api.FailJSON(c, "NEXT_ASSIGNEE_REQUIRED")
		return true
	}
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	seq := s.nextToolSeq(bizDate)
	docNo := fmt.Sprintf("TI%s%03d", time.Now().Format("20060102150405"), seq%1000)
	res, err := s.DB.Exec(`INSERT INTO hr_tool_issue(doc_no, biz_date, seq_no, employee_id, employee_name, status, remark, pending_return_qty)
		VALUES(?,?,?,?,?, 'pending',?,0)`,
		docNo, bizDate, seq, nullIf0(empID), ename, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	for _, it := range items {
		if _, err := s.DB.Exec(`INSERT INTO hr_tool_issue_line(issue_id, tool_item_id, tool_name, issue_qty, return_qty, pending_return_qty)
			VALUES(?,?,?,?,0,0)`, id, it.ToolItemID, it.ToolName, it.IssueQty); err != nil {
			_, _ = s.DB.Exec(`DELETE FROM hr_tool_issue_line WHERE issue_id=?`, id)
			_, _ = s.DB.Exec(`DELETE FROM hr_tool_issue WHERE id=?`, id)
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
	}
	applicant := int64(0)
	if cl != nil {
		applicant = cl.UserID
	}
	if applicant == 0 && empID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(user_id,0) FROM hr_employee WHERE id=?`, empID).Scan(&applicant)
	}
	catID := s.categoryIDByCode("tool_issue")
	if catID == 0 || !s.assigneeInPool(catID, next) {
		_, _ = s.DB.Exec(`DELETE FROM hr_tool_issue_line WHERE issue_id=?`, id)
		_, _ = s.DB.Exec(`DELETE FROM hr_tool_issue WHERE id=?`, id)
		api.FailJSON(c, "ASSIGNEE_NOT_IN_POOL")
		return true
	}
	lines := s.loadToolIssueLines(id)
	summary := toolIssueItemsSummary(lines)
	issueTotal, _, _ := toolIssueQtyTotals(lines)
	payloadB, _ := json.Marshal(gin.H{
		"items": lines, "items_summary": summary, "employee_name": ename, "biz_date": bizDate, "seq_no": seq,
		"issue_qty": issueTotal,
	})
	title := fmt.Sprintf("工具领取申请 · %s · %d种", ename, len(items))
	if ename == "" {
		title = fmt.Sprintf("工具领取申请 · %d种", len(items))
	}
	tid, _, err := s.createTicket(c, catID, title, applicant, next, "hr_tool_issue", id, string(payloadB), strOr(body["remark"]))
	if err != nil {
		_, _ = s.DB.Exec(`DELETE FROM hr_tool_issue_line WHERE issue_id=?`, id)
		_, _ = s.DB.Exec(`DELETE FROM hr_tool_issue WHERE id=?`, id)
		api.FailJSON(c, err.Error())
		return true
	}
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET ticket_id=? WHERE id=?`, tid, id)
	out := s.loadToolIssue(id)
	out["ticket"] = s.loadTicket(tid)
	api.OK(c, out)
	return true
}

func (s *Services) approveToolIssue(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	m := s.loadToolIssue(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "pending" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	ticketID := asInt64Or0(m["ticket_id"])
	if ticketID > 0 {
		body["action"] = "approve"
		if !s.actionTicketBody(c, ticketID, body) {
			return true
		}
		out := s.loadToolIssue(id)
		out["ticket"] = s.loadTicket(ticketID)
		api.OK(c, out)
		return true
	}
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='open' WHERE id=?`, id)
	api.OK(c, s.loadToolIssue(id))
	return true
}

func (s *Services) rejectToolIssue(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	m := s.loadToolIssue(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	ticketID := asInt64Or0(m["ticket_id"])
	if ticketID > 0 {
		body["action"] = "reject"
		if !s.actionTicketBody(c, ticketID, body) {
			return true
		}
		api.OK(c, s.loadToolIssue(id))
		return true
	}
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='rejected' WHERE id=?`, id)
	api.OK(c, s.loadToolIssue(id))
	return true
}

func (s *Services) returnRequestToolIssue(c *gin.Context) bool {
	cl := middleware.Claims(c)
	body := bindBody(c)
	id := paramID(c)
	m := s.loadToolIssue(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "open" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	next, _ := asInt64(body["next_assignee_user_id"])
	if next <= 0 {
		api.FailJSON(c, "NEXT_ASSIGNEE_REQUIRED")
		return true
	}
	catID := s.categoryIDByCode("tool_return")
	if !s.assigneeInPool(catID, next) {
		api.FailJSON(c, "ASSIGNEE_NOT_IN_POOL")
		return true
	}
	lines := s.loadToolIssueLines(id)
	retItems, _ := body["items"].([]interface{})
	pendingByLine := map[int64]float64{}
	var headerPending float64
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue_line SET pending_return_qty=0 WHERE issue_id=?`, id)
	if len(retItems) > 0 {
		for _, it := range retItems {
			rm, _ := it.(map[string]interface{})
			if rm == nil {
				continue
			}
			lineID := asInt64Or0(rm["line_id"])
			if lineID == 0 {
				lineID = asInt64Or0(rm["id"])
			}
			qty := asFloatOr0(rm["return_qty"])
			if lineID <= 0 || qty <= 0 {
				api.FailJSON(c, "INVALID_RETURN_ITEM")
				return true
			}
			pendingByLine[lineID] = qty
		}
		for _, ln := range lines {
			lid := asInt64Or0(ln["id"])
			qty, ok := pendingByLine[lid]
			if !ok {
				continue
			}
			remain := asFloatOr0(ln["issue_qty"]) - asFloatOr0(ln["return_qty"])
			if qty > remain {
				qty = remain
			}
			if qty <= 0 {
				continue
			}
			pendingByLine[lid] = qty
			headerPending += qty
			_, _ = s.DB.Exec(`UPDATE hr_tool_issue_line SET pending_return_qty=? WHERE id=? AND issue_id=?`, qty, lid, id)
		}
		if headerPending <= 0 {
			api.FailJSON(c, "RETURN_QTY_REQUIRED")
			return true
		}
	} else {
		// legacy: single return_qty or full remain
		legacyRet := asFloatOr0(body["return_qty"])
		remainTotal := asFloatOr0(m["total_qty"])
		if legacyRet <= 0 {
			legacyRet = remainTotal
		}
		if legacyRet <= 0 {
			api.FailJSON(c, "RETURN_QTY_REQUIRED")
			return true
		}
		left := legacyRet
		for _, ln := range lines {
			remain := asFloatOr0(ln["issue_qty"]) - asFloatOr0(ln["return_qty"])
			if remain <= 0 || left <= 0 {
				_, _ = s.DB.Exec(`UPDATE hr_tool_issue_line SET pending_return_qty=0 WHERE id=?`, asInt64Or0(ln["id"]))
				continue
			}
			qty := remain
			if qty > left {
				qty = left
			}
			left -= qty
			headerPending += qty
			_, _ = s.DB.Exec(`UPDATE hr_tool_issue_line SET pending_return_qty=? WHERE id=?`, qty, asInt64Or0(ln["id"]))
		}
	}
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='pending_return', pending_return_qty=? WHERE id=?`, headerPending, id)
	applicant := int64(0)
	if cl != nil {
		applicant = cl.UserID
	}
	linesAfter := s.loadToolIssueLines(id)
	summary := toolIssueItemsSummary(linesAfter)
	payloadB, _ := json.Marshal(gin.H{
		"items": linesAfter, "items_summary": summary, "return_qty": headerPending,
		"employee_name": m["employee_name"], "biz_date": m["biz_date"], "seq_no": m["seq_no"],
	})
	title := fmt.Sprintf("工具归还 · %v · %.0f件", m["employee_name"], headerPending)
	tid, _, err := s.createTicket(c, catID, title, applicant, next, "hr_tool_issue", id, string(payloadB), strOr(body["remark"]))
	if err != nil {
		_, _ = s.DB.Exec(`UPDATE hr_tool_issue_line SET pending_return_qty=0 WHERE issue_id=?`, id)
		_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='open', pending_return_qty=0 WHERE id=?`, id)
		api.FailJSON(c, err.Error())
		return true
	}
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET ticket_id=? WHERE id=?`, tid, id)
	out := s.loadToolIssue(id)
	out["ticket_id"] = tid
	api.OK(c, out)
	return true
}

func (s *Services) returnConfirmToolIssue(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	m := s.loadToolIssue(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	ticketID := asInt64Or0(m["ticket_id"])
	if ticketID > 0 {
		body["action"] = "return_confirm"
		if !s.actionTicketBody(c, ticketID, body) {
			return true
		}
		out := s.loadToolIssue(id)
		out["ticket"] = s.loadTicket(ticketID)
		api.OK(c, out)
		return true
	}
	return s.returnToolIssue(c)
}

func (s *Services) returnToolIssue(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	m := s.loadToolIssue(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if err := s.applyToolIssueReturn(id, body); err != "" {
		api.FailJSON(c, err)
		return true
	}
	api.OK(c, s.loadToolIssue(id))
	return true
}

// applyToolIssueReturn confirms pending line returns (or full remain / explicit items).
func (s *Services) applyToolIssueReturn(id int64, body map[string]interface{}) string {
	lines := s.loadToolIssueLines(id)
	if len(lines) == 0 {
		return "NOT_FOUND"
	}
	retItems, _ := body["items"].([]interface{})
	qtyByLine := map[int64]float64{}
	if len(retItems) > 0 {
		for _, it := range retItems {
			rm, _ := it.(map[string]interface{})
			if rm == nil {
				continue
			}
			lineID := asInt64Or0(rm["line_id"])
			if lineID == 0 {
				lineID = asInt64Or0(rm["id"])
			}
			qty := asFloatOr0(rm["return_qty"])
			if lineID > 0 && qty > 0 {
				qtyByLine[lineID] = qty
			}
		}
	} else {
		usePending := false
		for _, ln := range lines {
			if asFloatOr0(ln["pending_return_qty"]) > 0 {
				usePending = true
				break
			}
		}
		legacyRet := asFloatOr0(body["return_qty"])
		if usePending {
			for _, ln := range lines {
				qtyByLine[asInt64Or0(ln["id"])] = asFloatOr0(ln["pending_return_qty"])
			}
		} else if legacyRet > 0 {
			left := legacyRet
			for _, ln := range lines {
				remain := asFloatOr0(ln["issue_qty"]) - asFloatOr0(ln["return_qty"])
				if remain <= 0 || left <= 0 {
					continue
				}
				qty := remain
				if qty > left {
					qty = left
				}
				left -= qty
				qtyByLine[asInt64Or0(ln["id"])] = qty
			}
		} else {
			// full return remaining
			for _, ln := range lines {
				remain := asFloatOr0(ln["issue_qty"]) - asFloatOr0(ln["return_qty"])
				if remain > 0 {
					qtyByLine[asInt64Or0(ln["id"])] = remain
				}
			}
		}
	}
	allReturned := true
	var headerReturn, headerIssue float64
	for _, ln := range lines {
		lid := asInt64Or0(ln["id"])
		issue := asFloatOr0(ln["issue_qty"])
		curRet := asFloatOr0(ln["return_qty"])
		add := qtyByLine[lid]
		if add < 0 {
			add = 0
		}
		newRet := curRet + add
		if newRet > issue {
			newRet = issue
		}
		_, _ = s.DB.Exec(`UPDATE hr_tool_issue_line SET return_qty=?, pending_return_qty=0 WHERE id=?`, newRet, lid)
		headerIssue += issue
		headerReturn += newRet
		if newRet < issue {
			allReturned = false
		}
	}
	st := "open"
	if allReturned {
		st = "returned"
	}
	_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status=?, pending_return_qty=0 WHERE id=?`, st, id)
	_ = headerIssue
	_ = headerReturn
	return ""
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
		COALESCE(r.confirmed_at, r.reported_at,''), COALESCE(r.process_qc_result,''),
		COALESCE(r.worker_id,0), COALESCE(ew.name,''), COALESCE(r.operator_employee_id,0), COALESCE(eo.name,''),
		COALESCE(r.operator_user_id,0), COALESCE(r.scan_code,'')
		FROM pd_report_work r
		LEFT JOIN pd_process p ON p.id=r.process_id
		LEFT JOIN hr_employee ew ON ew.id=r.worker_id
		LEFT JOIN hr_employee eo ON eo.id=r.operator_employee_id
		`+where+`
		ORDER BY r.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, procID, workerID, opEmpID, opUserID int64
		var docNo, status, pname, at, qc, workerName, opName, scan string
		var inW, outW, loss, bag float64
		_ = rows.Scan(&id, &docNo, &status, &procID, &pname, &inW, &outW, &loss, &bag, &at, &qc,
			&workerID, &workerName, &opEmpID, &opName, &opUserID, &scan)
		if opName == "" && opUserID > 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(e.name,u.login_name,'') FROM iam_user u
				LEFT JOIN hr_employee e ON e.id=u.employee_id WHERE u.id=?`, opUserID).Scan(&opName)
		}
		var scrapTypeVal string
		_ = s.DB.QueryRow(`SELECT COALESCE(scrap_type,'') FROM pd_scrap_record WHERE doc_no LIKE ? ORDER BY id DESC LIMIT 1`, fmt.Sprintf("SCR-%d%%", id)).Scan(&scrapTypeVal)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "status": status, "process_id": procID, "process_name": pname,
			"input_weight": inW, "output_weight": outW, "loss": loss, "bag_qty": bag, "reported_at": at,
			"process_qc_result": qc, "scrap_type": scrapTypeVal,
			"worker_id": workerID, "worker_name": workerName,
			"operator_employee_id": opEmpID, "operator_user_id": opUserID, "operator_name": opName,
			"pass_for_other": opEmpID > 0 && workerID != opEmpID,
			"scan_code": scan,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}
