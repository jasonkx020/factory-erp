package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) ensureSalesLoopColumns() {
	stmts := []string{
		`ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS reject_reason TEXT`,
		`ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS submitted_at TEXT`,
		`ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS approved_at TEXT`,
		`ALTER TABLE sl_inquiry ADD COLUMN IF NOT EXISTS rejected_at TEXT`,
		`ALTER TABLE sl_delivery_approval ADD COLUMN IF NOT EXISTS reject_reason TEXT`,
		`ALTER TABLE sl_delivery_approval ADD COLUMN IF NOT EXISTS received_at TEXT`,
		`ALTER TABLE sl_delivery_approval ADD COLUMN IF NOT EXISTS receive_remark TEXT`,
		`ALTER TABLE sl_outbound_settle ADD COLUMN IF NOT EXISTS order_id INTEGER`,
		`ALTER TABLE sl_outbound_settle ADD COLUMN IF NOT EXISTS delivery_id INTEGER`,
		`ALTER TABLE sl_outbound_settle ADD COLUMN IF NOT EXISTS closed_at TEXT`,
		`ALTER TABLE sl_contract ADD COLUMN IF NOT EXISTS attachment_url TEXT`,
		`ALTER TABLE sl_contract ADD COLUMN IF NOT EXISTS order_id INTEGER`,
		`ALTER TABLE sl_price_lock ADD COLUMN IF NOT EXISTS version_no INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE sl_sales_bom ADD COLUMN IF NOT EXISTS version_no INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE sl_sales_bom ADD COLUMN IF NOT EXISTS remark TEXT`,
		`ALTER TABLE sl_cost_budget ADD COLUMN IF NOT EXISTS updated_at TEXT NOT NULL DEFAULT NOW()`,
		`ALTER TABLE sl_self_order_rule ADD COLUMN IF NOT EXISTS max_amount DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE sl_self_order_rule ADD COLUMN IF NOT EXISTS max_qty DOUBLE PRECISION NOT NULL DEFAULT 0`,
	}
	for _, q := range stmts {
		_, _ = s.DB.Exec(q)
	}
}

func (s *Services) enqueueSalesApproval(category, bizType, docNo, title string, bizID, applicant int64, amount float64) {
	if bizID <= 0 {
		return
	}
	var existing int64
	_ = s.DB.QueryRow(`SELECT id FROM appr_queue WHERE category=? AND biz_type=? AND biz_id=? AND status='pending' ORDER BY id DESC LIMIT 1`,
		category, bizType, bizID).Scan(&existing)
	if existing > 0 {
		return
	}
	if applicant <= 0 {
		applicant = 1
	}
	if title == "" {
		title = docNo
	}
	_, _ = s.DB.Exec(`INSERT INTO appr_queue(category, doc_no, title, biz_type, biz_id, amount, applicant_id, assignee_user_id, status, remark)
		VALUES(?,?,?,?,?,?,?,1,'pending',?)`,
		category, docNo, title, bizType, bizID, amount, applicant, "由销售单据提交")
}

func (s *Services) closeSalesApprovalQueue(bizType string, bizID int64, status, comment string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, _ = s.DB.Exec(`UPDATE appr_queue SET status=?, comment=COALESCE(NULLIF(?,''),comment), acted_at=?, updated_at=?
		WHERE biz_id=? AND status='pending' AND biz_type IN (?,?)`,
		status, comment, now, now, bizID, bizType, "sl_"+bizType)
	_, _ = s.DB.Exec(`UPDATE appr_task SET status=?, comment=COALESCE(NULLIF(?,''),comment), acted_at=?
		WHERE doc_id=? AND status='pending' AND doc_type IN (?,?)`,
		status, comment, now, bizID, bizType, "sl_"+bizType)
}

func (s *Services) loadApprovalTrail(bizType string, bizID int64) []gin.H {
	out := []gin.H{}
	if bizID <= 0 {
		return out
	}
	rows, err := s.DB.Query(`SELECT id, category, COALESCE(doc_no,''), title, status, COALESCE(comment,''),
		COALESCE(acted_at,''), created_at, assignee_user_id
		FROM appr_queue WHERE biz_id=? AND biz_type IN (?,?) ORDER BY id DESC`,
		bizID, bizType, "sl_"+bizType)
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, assignee int64
			var cat, docNo, title, st, comment, acted, created string
			_ = rows.Scan(&id, &cat, &docNo, &title, &st, &comment, &acted, &created, &assignee)
			out = append(out, gin.H{
				"id": id, "source": "queue", "category": cat, "doc_no": docNo, "title": title,
				"status": st, "comment": comment, "acted_at": acted, "created_at": created,
				"assignee_user_id": assignee,
			})
		}
	}
	trows, err := s.DB.Query(`SELECT id, COALESCE(title,''), COALESCE(doc_no,''), status, COALESCE(comment,''),
		COALESCE(acted_at,''), created_at, assignee_user_id
		FROM appr_task WHERE doc_id=? AND doc_type IN (?,?) ORDER BY id DESC`,
		bizID, bizType, "sl_"+bizType)
	if err == nil && trows != nil {
		defer trows.Close()
		for trows.Next() {
			var id, assignee int64
			var title, docNo, st, comment, acted, created string
			_ = trows.Scan(&id, &title, &docNo, &st, &comment, &acted, &created, &assignee)
			out = append(out, gin.H{
				"id": id, "source": "task", "category": "task", "doc_no": docNo, "title": title,
				"status": st, "comment": comment, "acted_at": acted, "created_at": created,
				"assignee_user_id": assignee,
			})
		}
	}
	return out
}

func salesNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (s *Services) inquirySubmit(c *gin.Context) bool {
	id := paramID(c)
	inq := s.loadInquiry(id)
	if inq["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if s.refusePortalMismatch(c, ginHInt64(inq["customer_id"])) {
		return true
	}
	st := strOr(inq["status"])
	if st != "draft" && st != "rejected" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	now := salesNow()
	_, err := s.DB.Exec(`UPDATE sl_inquiry SET status='pending', submitted_at=?, reject_reason=NULL, updated_at=NOW() WHERE id=?`, now, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	var applicant int64
	if claims := middleware.Claims(c); claims != nil {
		applicant = claims.UserID
	}
	amount := 0.0
	if lines, ok := inq["lines"].([]gin.H); ok {
		for _, ln := range lines {
			qty, _ := asFloat(ln["qty"])
			price, _ := asFloat(ln["quote_price"])
			amount += qty * price
		}
	}
	s.enqueueSalesApproval("inquiry_finance", "inquiry", strOr(inq["doc_no"]),
		fmt.Sprintf("询价审批 %s", inq["doc_no"]), id, applicant, amount)
	api.OK(c, s.loadInquiry(id))
	return true
}

func (s *Services) inquiryReject(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	comment := strOrDef(body["comment"], strOr(body["reject_reason"]))
	res, err := s.DB.Exec(`UPDATE sl_inquiry SET status='rejected', reject_reason=?, rejected_at=?, updated_at=NOW()
		WHERE id=? AND status IN ('draft','pending')`, comment, salesNow(), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	s.closeSalesApprovalQueue("inquiry", id, "rejected", comment)
	api.OK(c, s.loadInquiry(id))
	return true
}

func (s *Services) inquiryWithdraw(c *gin.Context) bool {
	id := paramID(c)
	inq := s.loadInquiry(id)
	if inq["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if s.refusePortalMismatch(c, ginHInt64(inq["customer_id"])) {
		return true
	}
	res, err := s.DB.Exec(`UPDATE sl_inquiry SET status='draft', updated_at=NOW() WHERE id=? AND status='pending'`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	s.closeSalesApprovalQueue("inquiry", id, "withdrawn", "销售撤回")
	api.OK(c, s.loadInquiry(id))
	return true
}

func (s *Services) preShipCancel(c *gin.Context) bool {
	id := paramID(c)
	ps := s.loadPreShip(id)
	if ps["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(ps["status"])
	if st == "confirmed" || st == "cancelled" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='cancelled' WHERE source_doc_type='pre_shipment' AND source_doc_id=? AND status='active'`, id)
	_, _ = s.DB.Exec(`UPDATE sl_pre_shipment SET reserved=0, status='cancelled', updated_at=NOW() WHERE id=?`, id)
	api.OK(c, s.loadPreShip(id))
	return true
}

func (s *Services) deliveryResubmit(c *gin.Context) bool {
	id := paramID(c)
	res, err := s.DB.Exec(`UPDATE sl_delivery_approval SET status='pending', reject_reason=NULL, updated_at=NOW() WHERE id=? AND status='rejected'`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	d := s.loadDelivery(id)
	var applicant int64
	if claims := middleware.Claims(c); claims != nil {
		applicant = claims.UserID
	}
	s.enqueueSalesApproval("doc_review", "delivery", strOr(d["doc_no"]),
		fmt.Sprintf("发货重提 %s", d["doc_no"]), id, applicant, 0)
	api.OK(c, d)
	return true
}

func (s *Services) deliveryReceive(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	remark := strOrDef(body["receive_remark"], strOr(body["comment"]))
	res, err := s.DB.Exec(`UPDATE sl_delivery_approval SET status='received', received_at=?, receive_remark=?, updated_at=NOW()
		WHERE id=? AND status='shipped'`, salesNow(), remark, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	api.OK(c, s.loadDelivery(id))
	return true
}

func (s *Services) handleSelfOrderRules(c *gin.Context, method, action string) bool {
	if method == "GET" || action == "list" {
		return s.handleSelfOrders(c, "GET")
	}
	body := bindBody(c)
	name := strOrDef(body["name"], "默认规则")
	minQty, _ := asFloat(body["min_qty"])
	maxQty, _ := asFloat(body["max_qty"])
	maxAmt, _ := asFloat(body["max_amount"])
	enabled := 1
	if v, ok := body["enabled"]; ok {
		switch t := v.(type) {
		case bool:
			if !t {
				enabled = 0
			}
		case float64:
			if t == 0 {
				enabled = 0
			}
		case string:
			if t == "0" || strings.EqualFold(t, "false") {
				enabled = 0
			}
		}
	}
	id := paramID(c)
	if id > 0 {
		_, err := s.DB.Exec(`UPDATE sl_self_order_rule SET name=?, enabled=?, min_qty=?, max_qty=?, max_amount=?,
			allow_products_json=COALESCE(NULLIF(?,''),allow_products_json), remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
			name, enabled, minQty, maxQty, maxAmt, strOr(body["allow_products_json"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id, "name": name, "enabled": enabled == 1, "min_qty": minQty, "max_qty": maxQty, "max_amount": maxAmt})
		return true
	}
	res, err := s.DB.Exec(`INSERT INTO sl_self_order_rule(name, enabled, min_qty, max_qty, max_amount, allow_products_json, remark)
		VALUES(?,?,?,?,?,?,?)`, name, enabled, minQty, maxQty, maxAmt, strOr(body["allow_products_json"]), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	nid, _ := res.LastInsertId()
	api.OK(c, gin.H{"id": nid, "name": name, "enabled": enabled == 1, "min_qty": minQty, "max_qty": maxQty, "max_amount": maxAmt})
	return true
}

func (s *Services) recalcCostBudget(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	var oid int64
	var saleAmt float64
	if err := s.DB.QueryRow(`SELECT order_id, sale_amount FROM sl_cost_budget WHERE id=?`, id).Scan(&oid, &saleAmt); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	ord := s.loadSalesOrder(oid)
	if v, _ := asFloat(ord["total_amount"]); v > 0 {
		saleAmt = v
	}
	mat, hasMat := asFloat(body["material_cost"])
	labor, hasLabor := asFloat(body["labor_cost"])
	other, hasOther := asFloat(body["other_cost"])
	if !hasMat {
		_ = s.DB.QueryRow(`SELECT material_cost FROM sl_cost_budget WHERE id=?`, id).Scan(&mat)
	}
	if !hasLabor {
		_ = s.DB.QueryRow(`SELECT labor_cost FROM sl_cost_budget WHERE id=?`, id).Scan(&labor)
	}
	if !hasOther {
		_ = s.DB.QueryRow(`SELECT other_cost FROM sl_cost_budget WHERE id=?`, id).Scan(&other)
	}
	total := mat + labor + other
	margin := 0.0
	if saleAmt > 0 {
		margin = (saleAmt - total) / saleAmt
	}
	_, err := s.DB.Exec(`UPDATE sl_cost_budget SET material_cost=?, labor_cost=?, other_cost=?, total_cost=?, sale_amount=?, margin=?,
		remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
		mat, labor, other, total, saleAmt, margin, strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, gin.H{"id": id, "order_id": oid, "material_cost": mat, "labor_cost": labor, "other_cost": other,
		"total_cost": total, "sale_amount": saleAmt, "margin": margin})
	return true
}

func (s *Services) reopenOutboundSettle(c *gin.Context) bool {
	id := paramID(c)
	res, err := s.DB.Exec(`UPDATE sl_outbound_settle SET status='draft', closed_at=NULL, updated_at=NOW() WHERE id=? AND status='closed'`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	api.OK(c, s.loadOutboundSettle(id))
	return true
}
