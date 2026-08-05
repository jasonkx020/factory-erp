package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsureApprovalSchema creates factory approval tables (SQLite).
func EnsureApprovalSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS appr_task (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  flow_id INTEGER,
  node_id INTEGER,
  doc_type TEXT NOT NULL,
  doc_id INTEGER NOT NULL DEFAULT 0,
  assignee_user_id INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'pending',
  acted_at TEXT,
  comment TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`ALTER TABLE appr_task ADD COLUMN title TEXT`,
		`ALTER TABLE appr_task ADD COLUMN doc_no TEXT`,
		`ALTER TABLE appr_task ADD COLUMN amount REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE appr_task ADD COLUMN applicant_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE appr_task ADD COLUMN remark TEXT`,
		`CREATE TABLE IF NOT EXISTS appr_queue (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  category TEXT NOT NULL,
  doc_no TEXT,
  title TEXT NOT NULL,
  biz_type TEXT,
  biz_id INTEGER NOT NULL DEFAULT 0,
  amount REAL NOT NULL DEFAULT 0,
  applicant_id INTEGER NOT NULL DEFAULT 0,
  assignee_user_id INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'pending',
  remark TEXT,
  comment TEXT,
  payload_json TEXT,
  acted_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE INDEX IF NOT EXISTS idx_appr_queue_cat ON appr_queue(category, status)`,
		`CREATE TABLE IF NOT EXISTS appr_expense_request (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  applicant_id INTEGER NOT NULL DEFAULT 1,
  amount REAL NOT NULL DEFAULT 0,
  category TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  queue_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS appr_affair_request (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  applicant_id INTEGER NOT NULL DEFAULT 1,
  title TEXT,
  content TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  queue_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	}
	for _, s := range stmts {
		_, _ = db.Exec(s)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM appr_queue`).Scan(&n)
	if n == 0 {
		seeds := []struct {
			cat, title, biz string
			amount          float64
		}{
			{"doc_review", "销售订单审核", "sales_order", 12800},
			{"expense_finance", "差旅报销财审", "expense", 860},
			{"inquiry_finance", "客户询价财审", "inquiry", 0},
			{"inquiry_line", "询价明细审批", "inquiry_line", 3200},
			{"purchase", "原料采购审批", "purchase_order", 56000},
			{"purchase_plan", "月度采购计划审批", "purchase_plan", 120000},
			{"affair", "设备检修事务申请", "affair", 0},
			{"attendance", "请假/补卡审批", "leave", 0},
		}
		for i, s := range seeds {
			docNo := fmt.Sprintf("AQ%s%02d", time.Now().Format("060102"), i+1)
			_, _ = db.Exec(`INSERT INTO appr_queue(category, doc_no, title, biz_type, biz_id, amount, applicant_id, assignee_user_id, status, remark)
				VALUES(?,?,?,?,?,?,1,1,'pending',?)`,
				s.cat, docNo, s.title, s.biz, i+1, s.amount, "工厂演示待办")
		}
	}
	_ = db.QueryRow(`SELECT COUNT(1) FROM appr_task`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO appr_task(doc_type, doc_id, assignee_user_id, status, title, doc_no, amount, applicant_id, remark)
			VALUES('sales_order',1,1,'pending','销售订单待审','SO-DEMO-001',9800,1,'演示任务')`)
		_, _ = db.Exec(`INSERT INTO appr_task(doc_type, doc_id, assignee_user_id, status, title, doc_no, amount, applicant_id, remark)
			VALUES('purchase_order',1,1,'pending','采购单待审','PO-DEMO-001',45000,1,'演示任务')`)
	}
}

func apprDocNo(prefix string) string {
	return fmt.Sprintf("%s%s", prefix, time.Now().Format("060102150405"))
}

func (s *Services) handleApprovalDomain(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/approval/tasks"):
		return s.handleApprovalTasksExt(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/approval/expense-requests"):
		return s.handleApprExpenseRequests(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/approval/doc-reviews"):
		return s.handleApprQueue(c, method, action, openapiPath, "doc_review", "单据审核")
	case strings.HasPrefix(openapiPath, "/api/v1/approval/expense-finance"):
		return s.handleApprQueue(c, method, action, openapiPath, "expense_finance", "费用财务审批")
	case strings.HasPrefix(openapiPath, "/api/v1/approval/inquiry-finance"):
		return s.handleApprQueue(c, method, action, openapiPath, "inquiry_finance", "询价财务审批")
	case strings.HasPrefix(openapiPath, "/api/v1/approval/inquiry-lines"):
		return s.handleApprQueue(c, method, action, openapiPath, "inquiry_line", "询价明细审批")
	case strings.HasPrefix(openapiPath, "/api/v1/approval/purchases"):
		return s.handleApprQueue(c, method, action, openapiPath, "purchase", "采购审批")
	case strings.HasPrefix(openapiPath, "/api/v1/approval/purchase-plans"):
		return s.handleApprQueue(c, method, action, openapiPath, "purchase_plan", "采购计划单审批")
	case strings.HasPrefix(openapiPath, "/api/v1/approval/affairs"):
		return s.handleApprAffairs(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/approval/attendance"):
		return s.handleApprAttendance(c, method, action, openapiPath)
	default:
		return false
	}
}

// handleApprovalTasksExt replaces legacy task handler with richer fields.
func (s *Services) handleApprovalTasksExt(c *gin.Context, method, action, path string) bool {
	if strings.HasSuffix(path, "/approve") && method == "POST" {
		return s.decideApprTask(c, "approved")
	}
	if strings.HasSuffix(path, "/reject") && method == "POST" {
		return s.decideApprTask(c, "rejected")
	}
	switch action {
	case "list":
		status := c.Query("status")
		q := `SELECT id, doc_type, doc_id, assignee_user_id, status, COALESCE(comment,''), created_at,
			COALESCE(title,''), COALESCE(doc_no,''), COALESCE(amount,0), COALESCE(applicant_id,0), COALESCE(remark,''), COALESCE(acted_at,'')
			FROM appr_task`
		args := []interface{}{}
		if status != "" {
			q += ` WHERE status=?`
			args = append(args, status)
		}
		q += ` ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, docID, uid, applicant int64
			var docType, st, comment, created, title, docNo, remark, acted string
			var amount float64
			_ = rows.Scan(&id, &docType, &docID, &uid, &st, &comment, &created, &title, &docNo, &amount, &applicant, &remark, &acted)
			list = append(list, gin.H{
				"id": id, "doc_type": docType, "doc_id": docID, "assignee_user_id": uid,
				"status": st, "comment": comment, "created_at": created, "title": title,
				"doc_no": docNo, "amount": amount, "applicant_id": applicant, "remark": remark, "acted_at": acted,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		docType := strOrDef(body["doc_type"], "general")
		docID, _ := asInt64(body["doc_id"])
		uid, _ := asInt64(body["assignee_user_id"])
		if uid == 0 {
			uid = 1
		}
		applicant, _ := asInt64(body["applicant_id"])
		if applicant == 0 {
			applicant = 1
		}
		amount, _ := asFloat(body["amount"])
		title := strOrDef(body["title"], "审批任务")
		docNo := strOrDef(body["doc_no"], apprDocNo("AT"))
		remark := strOr(body["remark"])
		res, err := s.DB.Exec(`INSERT INTO appr_task(doc_type, doc_id, assignee_user_id, status, title, doc_no, amount, applicant_id, remark)
			VALUES(?,?,?,'pending',?,?,?,?,?)`, docType, docID, uid, title, docNo, amount, applicant, remark)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		if docID > 0 && s.Store != nil {
			_, _ = s.Store.SetStatus(docID, "pending_approval")
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "pending", "title": title})
		return true
	case "get":
		id := paramID(c)
		row := s.scanApprTask(id)
		if row == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, row)
		return true
	}
	return false
}

func (s *Services) scanApprTask(id int64) gin.H {
	var docID, uid, applicant int64
	var docType, st, comment, created, title, docNo, remark, acted string
	var amount float64
	err := s.DB.QueryRow(`SELECT doc_type, doc_id, assignee_user_id, status, COALESCE(comment,''), created_at,
		COALESCE(title,''), COALESCE(doc_no,''), COALESCE(amount,0), COALESCE(applicant_id,0), COALESCE(remark,''), COALESCE(acted_at,'')
		FROM appr_task WHERE id=?`, id).
		Scan(&docType, &docID, &uid, &st, &comment, &created, &title, &docNo, &amount, &applicant, &remark, &acted)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "doc_type": docType, "doc_id": docID, "assignee_user_id": uid,
		"status": st, "comment": comment, "created_at": created, "title": title,
		"doc_no": docNo, "amount": amount, "applicant_id": applicant, "remark": remark, "acted_at": acted,
	}
}

func (s *Services) decideApprTask(c *gin.Context, status string) bool {
	id := paramID(c)
	body := bindBody(c)
	comment := strOr(body["comment"])
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`UPDATE appr_task SET status=?, acted_at=?, comment=? WHERE id=? AND status='pending'`,
		status, now, comment, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "NOT_PENDING_OR_NOT_FOUND")
		return true
	}
	var docType string
	var docID int64
	_ = s.DB.QueryRow(`SELECT doc_type, doc_id FROM appr_task WHERE id=?`, id).Scan(&docType, &docID)
	if docID > 0 && s.Store != nil {
		_, _ = s.Store.SetStatus(docID, status)
	}
	s.syncBizAfterApproval(docType, docID, status)
	api.OK(c, gin.H{"id": id, "status": status, "doc_type": docType, "doc_id": docID, "acted_at": now})
	return true
}

func (s *Services) handleApprQueue(c *gin.Context, method, action, path, category, label string) bool {
	if strings.HasSuffix(path, "/approve") && method == "POST" {
		return s.decideApprQueue(c, category, "approved")
	}
	if strings.HasSuffix(path, "/reject") && method == "POST" {
		return s.decideApprQueue(c, category, "rejected")
	}
	if method == "GET" || action == "list" {
		return s.listApprQueue(c, category)
	}
	if method == "POST" || action == "create" {
		return s.createApprQueue(c, category, label)
	}
	return false
}

func (s *Services) listApprQueue(c *gin.Context, category string) bool {
	status := c.Query("status")
	q := `SELECT id, category, COALESCE(doc_no,''), title, COALESCE(biz_type,''), biz_id, amount,
		applicant_id, assignee_user_id, status, COALESCE(remark,''), COALESCE(comment,''),
		COALESCE(acted_at,''), created_at
		FROM appr_queue WHERE category=?`
	args := []interface{}{category}
	if status != "" {
		q += ` AND status=?`
		args = append(args, status)
	}
	q += ` ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, id DESC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, bizID, applicant, assignee int64
		var cat, docNo, title, bizType, st, remark, comment, acted, created string
		var amount float64
		_ = rows.Scan(&id, &cat, &docNo, &title, &bizType, &bizID, &amount, &applicant, &assignee, &st, &remark, &comment, &acted, &created)
		list = append(list, gin.H{
			"id": id, "category": cat, "doc_no": docNo, "title": title, "biz_type": bizType, "biz_id": bizID,
			"amount": amount, "applicant_id": applicant, "assignee_user_id": assignee, "status": st,
			"remark": remark, "comment": comment, "acted_at": acted, "created_at": created,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list), "category": category})
	return true
}

func (s *Services) createApprQueue(c *gin.Context, category, label string) bool {
	body := bindBody(c)
	title := strOrDef(body["title"], label)
	docNo := strOrDef(body["doc_no"], apprDocNo("AQ"))
	bizType := strOr(body["biz_type"])
	bizID, _ := asInt64(body["biz_id"])
	amount, _ := asFloat(body["amount"])
	applicant, _ := asInt64(body["applicant_id"])
	if applicant == 0 {
		applicant = 1
	}
	assignee, _ := asInt64(body["assignee_user_id"])
	if assignee == 0 {
		assignee = 1
	}
	remark := strOr(body["remark"])
	res, err := s.DB.Exec(`INSERT INTO appr_queue(category, doc_no, title, biz_type, biz_id, amount, applicant_id, assignee_user_id, status, remark)
		VALUES(?,?,?,?,?,?,?,?,'pending',?)`,
		category, docNo, title, bizType, bizID, amount, applicant, assignee, remark)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, gin.H{"id": id, "doc_no": docNo, "category": category, "status": "pending", "title": title})
	return true
}

func (s *Services) decideApprQueue(c *gin.Context, category, status string) bool {
	id := paramID(c)
	body := bindBody(c)
	comment := strOr(body["comment"])
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`UPDATE appr_queue SET status=?, comment=?, acted_at=?, updated_at=? WHERE id=? AND category=? AND status='pending'`,
		status, comment, now, now, id, category)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "NOT_PENDING_OR_NOT_FOUND")
		return true
	}
	var bizType string
	var bizID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(biz_type,''), biz_id FROM appr_queue WHERE id=?`, id).Scan(&bizType, &bizID)
	s.syncBizAfterApproval(bizType, bizID, status)
	if category == "expense_finance" && bizID > 0 {
		_, _ = s.DB.Exec(`UPDATE appr_expense_request SET status=?, updated_at=? WHERE id=?`, status, now, bizID)
	}
	if category == "affair" && bizID > 0 {
		_, _ = s.DB.Exec(`UPDATE appr_affair_request SET status=?, updated_at=? WHERE id=?`, status, now, bizID)
	}
	api.OK(c, gin.H{"id": id, "category": category, "status": status, "acted_at": now})
	return true
}

func (s *Services) syncBizAfterApproval(bizType string, bizID int64, status string) {
	if bizID <= 0 || bizType == "" {
		return
	}
	st := status
	switch bizType {
	case "sales_order", "sl_sales_order":
		_, _ = s.DB.Exec(`UPDATE sl_sales_order SET status=? WHERE id=?`, st, bizID)
	case "purchase_order", "pur_order", "purchase":
		_, _ = s.DB.Exec(`UPDATE pur_order SET status=? WHERE id=?`, st, bizID)
	case "purchase_plan":
		_, _ = s.DB.Exec(`UPDATE pur_plan SET status=? WHERE id=?`, st, bizID)
	case "inquiry", "sl_inquiry":
		_, _ = s.DB.Exec(`UPDATE sl_inquiry SET status=? WHERE id=?`, st, bizID)
	case "leave", "hr_leave_request":
		_, _ = s.DB.Exec(`UPDATE hr_leave_request SET status=? WHERE id=?`, st, bizID)
	case "overtime_patch", "hr_overtime_patch":
		_, _ = s.DB.Exec(`UPDATE hr_overtime_patch SET status=? WHERE id=?`, st, bizID)
	case "fin_voucher", "voucher":
		_, _ = s.DB.Exec(`UPDATE fin_voucher SET status=? WHERE id=?`, st, bizID)
	}
}

func (s *Services) handleApprExpenseRequests(c *gin.Context, method, action, path string) bool {
	if strings.HasSuffix(path, "/submit") && method == "POST" {
		id := paramID(c)
		var amount float64
		var docNo, category, remark string
		var applicant int64
		var st string
		err := s.DB.QueryRow(`SELECT doc_no, applicant_id, amount, COALESCE(category,''), status, COALESCE(remark,'') FROM appr_expense_request WHERE id=?`, id).
			Scan(&docNo, &applicant, &amount, &category, &st, &remark)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if st != "draft" && st != "rejected" {
			api.FailJSON(c, "INVALID_STATUS")
			return true
		}
		title := "费用申请-" + docNo
		if category != "" {
			title = category + "-" + docNo
		}
		res, err := s.DB.Exec(`INSERT INTO appr_queue(category, doc_no, title, biz_type, biz_id, amount, applicant_id, status, remark)
			VALUES('expense_finance',?,?, 'expense', ?, ?, ?, 'pending', ?)`,
			docNo, title, id, amount, applicant, remark)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		qid, _ := res.LastInsertId()
		now := time.Now().Format("2006-01-02 15:04:05")
		_, _ = s.DB.Exec(`UPDATE appr_expense_request SET status='submitted', queue_id=?, updated_at=? WHERE id=?`, qid, now, id)
		api.OK(c, gin.H{"id": id, "status": "submitted", "queue_id": qid})
		return true
	}
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, doc_no, applicant_id, amount, COALESCE(category,''), status, COALESCE(remark,''), COALESCE(queue_id,0), created_at
			FROM appr_expense_request ORDER BY id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, applicant, qid int64
			var docNo, cat, st, remark, created string
			var amount float64
			_ = rows.Scan(&id, &docNo, &applicant, &amount, &cat, &st, &remark, &qid, &created)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "applicant_id": applicant, "amount": amount,
				"category": cat, "status": st, "remark": remark, "queue_id": qid, "created_at": created,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], apprDocNo("ER"))
		applicant, _ := asInt64(body["applicant_id"])
		if applicant == 0 {
			applicant = 1
		}
		amount, _ := asFloat(body["amount"])
		cat := strOrDef(body["category"], "办公费用")
		remark := strOr(body["remark"])
		res, err := s.DB.Exec(`INSERT INTO appr_expense_request(doc_no, applicant_id, amount, category, status, remark)
			VALUES(?,?,?,?,'draft',?)`, docNo, applicant, amount, cat, remark)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft", "amount": amount})
		return true
	case "get":
		id := paramID(c)
		var applicant, qid int64
		var docNo, cat, st, remark, created string
		var amount float64
		err := s.DB.QueryRow(`SELECT doc_no, applicant_id, amount, COALESCE(category,''), status, COALESCE(remark,''), COALESCE(queue_id,0), created_at
			FROM appr_expense_request WHERE id=?`, id).Scan(&docNo, &applicant, &amount, &cat, &st, &remark, &qid, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{
			"id": id, "doc_no": docNo, "applicant_id": applicant, "amount": amount,
			"category": cat, "status": st, "remark": remark, "queue_id": qid, "created_at": created,
		})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		var st string
		_ = s.DB.QueryRow(`SELECT status FROM appr_expense_request WHERE id=?`, id).Scan(&st)
		if st != "draft" && st != "rejected" {
			api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
			return true
		}
		amount, ok := asFloat(body["amount"])
		if !ok {
			_ = s.DB.QueryRow(`SELECT amount FROM appr_expense_request WHERE id=?`, id).Scan(&amount)
		}
		_, err := s.DB.Exec(`UPDATE appr_expense_request SET
			amount=?, category=COALESCE(NULLIF(?,''),category), remark=COALESCE(NULLIF(?,''),remark),
			updated_at=datetime('now') WHERE id=?`,
			amount, strOr(body["category"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		api.OK(c, gin.H{"id": id, "ok": true})
		return true
	}
	return false
}

func (s *Services) handleApprAffairs(c *gin.Context, method, action, path string) bool {
	if strings.HasSuffix(path, "/approve") && method == "POST" {
		return s.decideAffair(c, "approved")
	}
	if strings.HasSuffix(path, "/reject") && method == "POST" {
		return s.decideAffair(c, "rejected")
	}
	if method == "GET" || action == "list" {
		rows, err := s.DB.Query(`SELECT id, doc_no, applicant_id, COALESCE(title,''), COALESCE(content,''), status, COALESCE(remark,''), created_at
			FROM appr_affair_request ORDER BY CASE status WHEN 'submitted' THEN 0 WHEN 'pending' THEN 0 ELSE 1 END, id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, applicant int64
			var docNo, title, content, st, remark, created string
			_ = rows.Scan(&id, &docNo, &applicant, &title, &content, &st, &remark, &created)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "applicant_id": applicant, "title": title,
				"content": content, "status": st, "remark": remark, "created_at": created, "source": "affair",
			})
		}
		qrows, _ := s.DB.Query(`SELECT id, COALESCE(doc_no,''), title, amount, status, COALESCE(remark,''), created_at
			FROM appr_queue WHERE category='affair' ORDER BY id DESC`)
		if qrows != nil {
			defer qrows.Close()
			for qrows.Next() {
				var id int64
				var docNo, title, st, remark, created string
				var amount float64
				_ = qrows.Scan(&id, &docNo, &title, &amount, &st, &remark, &created)
				list = append(list, gin.H{
					"id": id, "doc_no": docNo, "title": title, "amount": amount,
					"status": st, "remark": remark, "created_at": created, "source": "queue",
				})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	}
	if method == "POST" || action == "create" {
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], apprDocNo("AF"))
		applicant, _ := asInt64(body["applicant_id"])
		if applicant == 0 {
			applicant = 1
		}
		title := strOrDef(body["title"], "事务申请")
		content := strOr(body["content"])
		if content == "" {
			content = strOr(body["remark"])
		}
		submit := true
		if v, ok := body["submit"].(bool); ok {
			submit = v
		}
		st := "draft"
		if submit {
			st = "submitted"
		}
		res, err := s.DB.Exec(`INSERT INTO appr_affair_request(doc_no, applicant_id, title, content, status, remark)
			VALUES(?,?,?,?,?,?)`, docNo, applicant, title, content, st, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		var qid int64
		if submit {
			qr, _ := s.DB.Exec(`INSERT INTO appr_queue(category, doc_no, title, biz_type, biz_id, applicant_id, status, remark)
				VALUES('affair',?,?, 'affair', ?, ?, 'pending', ?)`, docNo, title, id, applicant, content)
			if qr != nil {
				qid, _ = qr.LastInsertId()
				_, _ = s.DB.Exec(`UPDATE appr_affair_request SET queue_id=? WHERE id=?`, qid, id)
			}
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": st, "queue_id": qid})
		return true
	}
	return false
}

func (s *Services) decideAffair(c *gin.Context, status string) bool {
	id := paramID(c)
	body := bindBody(c)
	comment := strOr(body["comment"])
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(`UPDATE appr_affair_request SET status=?, remark=CASE WHEN ?!='' THEN ? ELSE remark END, updated_at=?
		WHERE id=? AND status IN ('submitted','pending','draft')`, status, comment, comment, now, id)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			var qid sql.NullInt64
			_ = s.DB.QueryRow(`SELECT queue_id FROM appr_affair_request WHERE id=?`, id).Scan(&qid)
			if qid.Valid && qid.Int64 > 0 {
				_, _ = s.DB.Exec(`UPDATE appr_queue SET status=?, comment=?, acted_at=?, updated_at=? WHERE id=?`,
					status, comment, now, now, qid.Int64)
			}
			api.OK(c, gin.H{"id": id, "status": status, "source": "affair"})
			return true
		}
	}
	return s.decideApprQueue(c, "affair", status)
}

func (s *Services) handleApprAttendance(c *gin.Context, method, action, path string) bool {
	if strings.HasSuffix(path, "/approve") && method == "POST" {
		return s.decideAttendance(c, "approved")
	}
	if strings.HasSuffix(path, "/reject") && method == "POST" {
		return s.decideAttendance(c, "rejected")
	}
	if method == "GET" || action == "list" {
		list := []gin.H{}
		lrows, err := s.DB.Query(`SELECT l.id, l.doc_no, l.employee_id, COALESCE(e.name,''), l.leave_type, l.start_at, l.end_at, l.status, COALESCE(l.remark,''), l.created_at
			FROM hr_leave_request l LEFT JOIN hr_employee e ON e.id=l.employee_id
			ORDER BY CASE l.status WHEN 'pending' THEN 0 ELSE 1 END, l.id DESC`)
		if err == nil {
			defer lrows.Close()
			for lrows.Next() {
				var id, eid int64
				var docNo, ename, typ, start, end, st, remark, created string
				_ = lrows.Scan(&id, &docNo, &eid, &ename, &typ, &start, &end, &st, &remark, &created)
				list = append(list, gin.H{
					"id": id, "kind": "leave", "doc_no": docNo, "employee_id": eid, "employee_name": ename,
					"leave_type": typ, "start_at": start, "end_at": end, "status": st, "remark": remark, "created_at": created,
					"title": "请假-" + typ,
				})
			}
		}
		orows, err := s.DB.Query(`SELECT o.id, o.doc_no, o.employee_id, COALESCE(e.name,''), o.biz_type, o.biz_date, o.minutes, o.status, COALESCE(o.remark,''), o.created_at
			FROM hr_overtime_patch o LEFT JOIN hr_employee e ON e.id=o.employee_id
			ORDER BY CASE o.status WHEN 'pending' THEN 0 WHEN 'draft' THEN 0 ELSE 1 END, o.id DESC`)
		if err == nil {
			defer orows.Close()
			for orows.Next() {
				var id, eid int64
				var docNo, ename, bizType, bizDate, st, remark, created string
				var minutes int
				_ = orows.Scan(&id, &docNo, &eid, &ename, &bizType, &bizDate, &minutes, &st, &remark, &created)
				list = append(list, gin.H{
					"id": id, "kind": "overtime_patch", "doc_no": docNo, "employee_id": eid, "employee_name": ename,
					"biz_type": bizType, "biz_date": bizDate, "minutes": minutes, "status": st, "remark": remark, "created_at": created,
					"title": "加班补卡-" + bizType,
				})
			}
		}
		qrows, _ := s.DB.Query(`SELECT id, COALESCE(doc_no,''), title, status, COALESCE(remark,''), created_at FROM appr_queue WHERE category='attendance' ORDER BY id DESC`)
		if qrows != nil {
			defer qrows.Close()
			for qrows.Next() {
				var id int64
				var docNo, title, st, remark, created string
				_ = qrows.Scan(&id, &docNo, &title, &st, &remark, &created)
				list = append(list, gin.H{
					"id": id, "kind": "queue", "doc_no": docNo, "title": title, "status": st, "remark": remark, "created_at": created,
				})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	}
	if method == "POST" || action == "create" {
		body := bindBody(c)
		kind := strOrDef(body["kind"], "leave")
		if kind == "overtime_patch" || kind == "patch" {
			docNo := strOrDef(body["doc_no"], apprDocNo("OT"))
			eid, _ := asInt64(body["employee_id"])
			if eid == 0 {
				eid = 1
			}
			bizType := strOrDef(body["biz_type"], "overtime")
			bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
			minutes := 0
			if m, ok := asInt64(body["minutes"]); ok {
				minutes = int(m)
			}
			res, err := s.DB.Exec(`INSERT INTO hr_overtime_patch(doc_no, employee_id, biz_type, biz_date, minutes, status, remark)
				VALUES(?,?,?,?,?,'pending',?)`, docNo, eid, bizType, bizDate, minutes, strOr(body["remark"]))
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
			id, _ := res.LastInsertId()
			api.OK(c, gin.H{"id": id, "doc_no": docNo, "kind": "overtime_patch", "status": "pending"})
			return true
		}
		docNo := strOrDef(body["doc_no"], apprDocNo("LV"))
		eid, _ := asInt64(body["employee_id"])
		if eid == 0 {
			eid = 1
		}
		typ := strOrDef(body["leave_type"], "annual")
		start := strOrDef(body["start_at"], time.Now().Format("2006-01-02")+" 09:00:00")
		end := strOrDef(body["end_at"], time.Now().Format("2006-01-02")+" 18:00:00")
		res, err := s.DB.Exec(`INSERT INTO hr_leave_request(doc_no, employee_id, leave_type, start_at, end_at, status, remark)
			VALUES(?,?,?,?,?,'pending',?)`, docNo, eid, typ, start, end, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "kind": "leave", "status": "pending"})
		return true
	}
	return false
}

func (s *Services) decideAttendance(c *gin.Context, status string) bool {
	id := paramID(c)
	body := bindBody(c)
	kind := strOrDef(body["kind"], "leave")
	comment := strOr(body["comment"])
	now := time.Now().Format("2006-01-02 15:04:05")

	if kind == "overtime_patch" || kind == "patch" {
		res, err := s.DB.Exec(`UPDATE hr_overtime_patch SET status=? WHERE id=? AND status IN ('draft','pending')`, status, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		if n, _ := res.RowsAffected(); n == 0 {
			api.FailJSON(c, "NOT_PENDING_OR_NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "kind": "overtime_patch", "status": status, "comment": comment, "acted_at": now})
		return true
	}
	if kind == "queue" {
		return s.decideApprQueue(c, "attendance", status)
	}
	res, err := s.DB.Exec(`UPDATE hr_leave_request SET status=? WHERE id=? AND status IN ('draft','pending')`, status, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return s.decideApprQueue(c, "attendance", status)
	}
	api.OK(c, gin.H{"id": id, "kind": "leave", "status": status, "comment": comment, "acted_at": now})
	return true
}
