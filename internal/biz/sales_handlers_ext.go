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

// ---------- inquiries ----------

func (s *Services) handleSalesInquiries(c *gin.Context, method, action, path string) bool {
	switch {
	case strings.Contains(path, "/approve") || action == "action:approve":
		return s.inquiryApprove(c)
	case strings.Contains(path, "/to-order") || action == "action:to-order":
		return s.inquiryToOrder(c)
	case strings.Contains(path, "/submit") || action == "action:submit":
		return s.inquirySubmit(c)
	case strings.Contains(path, "/reject") || action == "action:reject":
		return s.inquiryReject(c)
	case strings.Contains(path, "/withdraw") || action == "action:withdraw":
		return s.inquiryWithdraw(c)
	case action == "list":
		return s.listInquiries(c)
	case action == "create":
		return s.createInquiry(c)
	case action == "get":
		m := s.loadInquiry(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if s.refusePortalMismatch(c, ginHInt64(m["customer_id"])) {
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		return s.updateInquiry(c)
	}
	return false
}

func (s *Services) listInquiries(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE COALESCE(i.is_deleted,0)=0`
	args := []interface{}{}
	if st := c.Query("status"); st != "" {
		where += ` AND i.status=?`
		args = append(args, st)
	}
	if cid, ok := asInt64(c.Query("customer_id")); ok && cid > 0 {
		where += ` AND i.customer_id=?`
		args = append(args, cid)
	}
	applyPortalCustomerSQL(c, "i.customer_id", &where, &args)
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		where += ` AND (i.doc_no LIKE ? OR COALESCE(c.name,'') LIKE ? OR COALESCE(i.remark,'') LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_inquiry i LEFT JOIN crm_customer c ON c.id=i.customer_id `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT i.id, i.doc_no, i.customer_id, COALESCE(c.name,''), i.status, i.source, COALESCE(i.remark,''), i.created_at,
		COALESCE(i.reject_reason,''), COALESCE(i.submitted_at,''), COALESCE(i.approved_at,''), COALESCE(i.rejected_at,'')
		FROM sl_inquiry i LEFT JOIN crm_customer c ON c.id=i.customer_id
		`+where+` ORDER BY i.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, cid int64
		var docNo, cname, st, source, remark, created, rejectReason, submitted, approved, rejected string
		_ = rows.Scan(&id, &docNo, &cid, &cname, &st, &source, &remark, &created, &rejectReason, &submitted, &approved, &rejected)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "customer_id": cid, "customer_name": cname,
			"status": st, "source": source, "remark": remark, "created_at": created,
			"reject_reason": rejectReason, "submitted_at": submitted, "approved_at": approved, "rejected_at": rejected,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createInquiry(c *gin.Context) bool {
	body := bindBody(c)
	customerID, _ := asInt64(body["customer_id"])
	if cid, ok := portalCustomerID(c); ok {
		customerID = cid
	}
	if customerID <= 0 {
		api.FailJSON(c, "CUSTOMER_REQUIRED")
		return true
	}
	lines := parseLines(body)
	if len(lines) == 0 {
		api.FailJSON(c, "LINES_REQUIRED")
		return true
	}
	docNo := fmt.Sprintf("IQ%s", time.Now().Format("20060102150405"))
	var ownerID int64
	if claims := middleware.Claims(c); claims != nil {
		ownerID = claims.UserID
	}
	source := strOrDef(body["source"], "sales")
	if _, ok := portalCustomerID(c); ok {
		source = "portal"
	}
	res, err := s.DB.Exec(`INSERT INTO sl_inquiry(doc_no, customer_id, owner_user_id, status, source, remark)
		VALUES(?,?,?,'draft',?,?)`, docNo, customerID, ownerID, source, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		price, _ := asFloat(ln["quote_price"])
		if price <= 0 {
			if lp, _, ok := s.resolveLockPrice(customerID, pid); ok {
				price = lp
			} else {
				price = s.productSalePrice(pid)
			}
		}
		_, _ = s.DB.Exec(`INSERT INTO sl_inquiry_line(inquiry_id, product_id, qty, quote_price, remark) VALUES(?,?,?,?,?)`,
			id, pid, qty, price, strOr(ln["remark"]))
		_, _ = s.DB.Exec(`INSERT INTO sl_quote_history(customer_id, product_id, price, quoted_at, inquiry_id) VALUES(?,?,?,NOW(),?)`,
			customerID, pid, price, id)
	}
	api.OK(c, s.loadInquiry(id))
	return true
}

func (s *Services) updateInquiry(c *gin.Context) bool {
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
		api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
		return true
	}
	body := bindBody(c)
	_, _ = s.DB.Exec(`UPDATE sl_inquiry SET remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
		strOr(body["remark"]), id)
	lines := parseLines(body)
	if len(lines) > 0 {
		_, _ = s.DB.Exec(`DELETE FROM sl_inquiry_line WHERE inquiry_id=?`, id)
		customerID, _ := asInt64(inq["customer_id"])
		for _, ln := range lines {
			pid, _ := asInt64(ln["product_id"])
			qty, _ := asFloat(ln["qty"])
			price, _ := asFloat(ln["quote_price"])
			if price <= 0 {
				price, _ = asFloat(ln["price"])
			}
			if price <= 0 {
				if lp, _, ok := s.resolveLockPrice(customerID, pid); ok {
					price = lp
				} else {
					price = s.productSalePrice(pid)
				}
			}
			_, _ = s.DB.Exec(`INSERT INTO sl_inquiry_line(inquiry_id, product_id, qty, quote_price, remark) VALUES(?,?,?,?,?)`,
				id, pid, qty, price, strOr(ln["remark"]))
		}
	}
	api.OK(c, s.loadInquiry(id))
	return true
}

func (s *Services) inquiryApprove(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	comment := strOr(body["comment"])
	res, err := s.DB.Exec(`UPDATE sl_inquiry SET status='approved', approved_at=?, updated_at=NOW() WHERE id=? AND status IN ('draft','pending')`, salesNow(), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	s.closeSalesApprovalQueue("inquiry", id, "approved", comment)
	api.OK(c, s.loadInquiry(id))
	return true
}

func (s *Services) inquiryToOrder(c *gin.Context) bool {
	id := paramID(c)
	inq := s.loadInquiry(id)
	if inq["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(inq["status"])
	if st != "approved" {
		api.FailJSON(c, "INQUIRY_NOT_APPROVED")
		return true
	}
	customerID, _ := asInt64(inq["customer_id"])
	docNo := fmt.Sprintf("SO%s", time.Now().Format("20060102150405"))
	var ownerID int64
	if claims := middleware.Claims(c); claims != nil {
		ownerID = claims.UserID
	}
	res, err := s.DB.Exec(`INSERT INTO sl_sales_order(doc_no, customer_id, owner_user_id, status, source, warehouse_id, total_amount, remark, created_by)
		VALUES(?,?,?,'submitted','inquiry',3,0,?,?)`, docNo, customerID, ownerID, fmt.Sprintf("来自询价 %v", inq["doc_no"]), ownerID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	oid, _ := res.LastInsertId()
	var total float64
	lines, _ := inq["lines"].([]gin.H)
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		price, _ := asFloat(ln["quote_price"])
		if price <= 0 {
			price, _ = asFloat(ln["price"])
		}
		amt := qty * price
		total += amt
		_, _ = s.DB.Exec(`INSERT INTO sl_sales_order_line(order_id, product_id, qty, price, amount) VALUES(?,?,?,?,?)`, oid, pid, qty, price, amt)
		_, _ = s.DB.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, qty, source_doc_type, source_doc_id, status)
			VALUES(3,?,?, 'sales_order',?,'active')`, pid, qty, oid)
	}
	_, _ = s.DB.Exec(`UPDATE sl_sales_order SET total_amount=? WHERE id=?`, total, oid)
	_, _ = s.DB.Exec(`UPDATE sl_inquiry SET status='ordered', updated_at=NOW() WHERE id=?`, id)
	api.OK(c, gin.H{"inquiry_id": id, "order": s.loadSalesOrder(oid)})
	return true
}

func (s *Services) loadInquiry(id int64) gin.H {
	var customerID int64
	var docNo, status, source, remark, created, rejectReason, submitted, approved, rejected string
	err := s.DB.QueryRow(`SELECT doc_no, customer_id, status, source, COALESCE(remark,''), created_at,
		COALESCE(reject_reason,''), COALESCE(submitted_at,''), COALESCE(approved_at,''), COALESCE(rejected_at,'')
		FROM sl_inquiry WHERE id=?`, id).
		Scan(&docNo, &customerID, &status, &source, &remark, &created, &rejectReason, &submitted, &approved, &rejected)
	if err != nil {
		return gin.H{}
	}
	cust := s.loadCustomer(customerID)
	rows, _ := s.DB.Query(`SELECT id, product_id, qty, COALESCE(quote_price,0), COALESCE(remark,'') FROM sl_inquiry_line WHERE inquiry_id=?`, id)
	lines := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, pid int64
			var qty, price float64
			var rmk string
			_ = rows.Scan(&lid, &pid, &qty, &price, &rmk)
			var pname string
			_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM prd_product WHERE id=?`, pid).Scan(&pname)
			lines = append(lines, gin.H{"id": lid, "product_id": pid, "product_name": pname, "qty": qty, "quote_price": price, "price": price, "remark": rmk})
		}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "customer_id": customerID, "customer_name": cust["name"],
		"status": status, "source": source, "remark": remark, "created_at": created, "lines": lines,
		"reject_reason": rejectReason, "submitted_at": submitted, "approved_at": approved, "rejected_at": rejected,
		"approvals":      s.loadApprovalTrail("inquiry", id),
		"approval_chain": "询价管理提交 → 询价审批/询价财务审批 → 转销售订单",
	}
}

// ---------- pre-shipments ----------

func (s *Services) handlePreShipments(c *gin.Context, method, action, path string) bool {
	switch {
	case strings.Contains(path, "/reserve") || action == "action:reserve":
		return s.preShipReserve(c, true)
	case strings.Contains(path, "/release") || action == "action:release":
		return s.preShipReserve(c, false)
	case strings.Contains(path, "/confirm") || action == "action:confirm":
		return s.preShipConfirmReal(c)
	case strings.Contains(path, "/cancel") || action == "action:cancel":
		return s.preShipCancel(c)
	case action == "list":
		return s.listPreShips(c)
	case action == "create":
		return s.createPreShip(c)
	case action == "get":
		m := s.loadPreShip(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if s.refusePortalMismatch(c, s.preShipCustomerID(paramID(c))) {
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE sl_pre_shipment SET plan_ship_date=COALESCE(NULLIF(?,''),plan_ship_date), remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
			strOr(body["plan_ship_date"]), strOr(body["remark"]), id)
		api.OK(c, s.loadPreShip(id))
		return true
	}
	return false
}

func (s *Services) listPreShips(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if st := c.Query("status"); st != "" {
		where += ` AND p.status=?`
		args = append(args, st)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		where += ` AND (p.doc_no LIKE ? OR COALESCE(o.doc_no,'') LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like)
	}
	applyPortalCustomerSQL(c, "o.customer_id", &where, &args)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_pre_shipment p LEFT JOIN sl_sales_order o ON o.id=p.order_id `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT p.id, p.doc_no, p.order_id, COALESCE(o.doc_no,''), COALESCE(p.plan_ship_date,''), p.status, p.reserved, p.created_at
		FROM sl_pre_shipment p LEFT JOIN sl_sales_order o ON o.id=p.order_id
		`+where+` ORDER BY p.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, oid int64
		var docNo, odoc, plan, st, created string
		var reserved int
		_ = rows.Scan(&id, &docNo, &oid, &odoc, &plan, &st, &reserved, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "order_id": oid, "order_no": odoc, "plan_ship_date": plan,
			"status": st, "reserved": reserved == 1, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createPreShip(c *gin.Context) bool {
	body := bindBody(c)
	orderID, _ := asInt64(body["order_id"])
	if orderID <= 0 {
		api.FailJSON(c, "ORDER_REQUIRED")
		return true
	}
	ord := s.loadSalesOrder(orderID)
	if ord["id"] == nil {
		api.FailJSON(c, "ORDER_NOT_FOUND")
		return true
	}
	wh, _ := asInt64(ord["warehouse_id"])
	if wh == 0 {
		wh = 3
	}
	docNo := fmt.Sprintf("PS%s", time.Now().Format("20060102150405"))
	res, err := s.DB.Exec(`INSERT INTO sl_pre_shipment(doc_no, order_id, plan_ship_date, status, reserved, warehouse_id, remark)
		VALUES(?,?,?,'draft',0,?,?)`, docNo, orderID, strOrDef(body["plan_ship_date"], time.Now().Format("2006-01-02")), wh, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	pid, _ := res.LastInsertId()
	lines := parseLines(body)
	if len(lines) == 0 {
		if ols, ok := ord["lines"].([]gin.H); ok {
			for _, ln := range ols {
				lines = append(lines, map[string]interface{}{
					"product_id": ln["product_id"], "qty": ln["qty"], "order_line_id": ln["id"],
				})
			}
		}
	}
	for _, ln := range lines {
		prid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		olid, _ := asInt64(ln["order_line_id"])
		_, _ = s.DB.Exec(`INSERT INTO sl_pre_shipment_line(pre_shipment_id, order_line_id, product_id, qty) VALUES(?,?,?,?)`,
			pid, nullIf0(olid), prid, qty)
	}
	api.OK(c, s.loadPreShip(pid))
	return true
}

func (s *Services) preShipReserve(c *gin.Context, reserve bool) bool {
	id := paramID(c)
	ps := s.loadPreShip(id)
	if ps["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	orderID, _ := asInt64(ps["order_id"])
	wh, _ := asInt64(ps["warehouse_id"])
	if wh == 0 {
		wh = 3
	}
	if reserve {
		lines, _ := ps["lines"].([]gin.H)
		for _, ln := range lines {
			prid, _ := asInt64(ln["product_id"])
			qty, _ := asFloat(ln["qty"])
			_, _ = s.DB.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, qty, source_doc_type, source_doc_id, status)
				VALUES(?,?,?,'pre_shipment',?,'active')`, wh, prid, qty, id)
		}
		_, _ = s.DB.Exec(`UPDATE sl_pre_shipment SET reserved=1, status='reserved', updated_at=NOW() WHERE id=?`, id)
	} else {
		_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='cancelled' WHERE source_doc_type='pre_shipment' AND source_doc_id=? AND status='active'`, id)
		_, _ = s.DB.Exec(`UPDATE sl_pre_shipment SET reserved=0, status='draft', updated_at=NOW() WHERE id=?`, id)
	}
	_ = orderID
	api.OK(c, s.loadPreShip(id))
	return true
}

func (s *Services) preShipConfirmReal(c *gin.Context) bool {
	id := paramID(c)
	ps := s.loadPreShip(id)
	if ps["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	// create delivery draft from pre-ship
	orderID, _ := asInt64(ps["order_id"])
	wh, _ := asInt64(ps["warehouse_id"])
	if wh == 0 {
		wh = 3
	}
	docNo := fmt.Sprintf("DL%s", time.Now().Format("20060102150405"))
	res, err := s.DB.Exec(`INSERT INTO sl_delivery_approval(doc_no, order_id, pre_shipment_id, status, warehouse_id, remark)
		VALUES(?,?,?,'pending',?,?)`, docNo, orderID, id, wh, "由预发货确认生成")
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	did, _ := res.LastInsertId()
	lines, _ := ps["lines"].([]gin.H)
	for _, ln := range lines {
		prid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		_, _ = s.DB.Exec(`INSERT INTO sl_delivery_line(delivery_id, product_id, qty) VALUES(?,?,?)`, did, prid, qty)
	}
	_, _ = s.DB.Exec(`UPDATE sl_pre_shipment SET status='confirmed', updated_at=NOW() WHERE id=?`, id)
	d := s.loadDelivery(did)
	var applicant int64
	if claims := middleware.Claims(c); claims != nil {
		applicant = claims.UserID
	}
	s.enqueueSalesApproval("doc_review", "delivery", strOr(d["doc_no"]),
		fmt.Sprintf("发货审批 %s", d["doc_no"]), did, applicant, 0)
	api.OK(c, gin.H{"pre_shipment": s.loadPreShip(id), "delivery": d})
	return true
}

func (s *Services) loadPreShip(id int64) gin.H {
	var orderID, wh int64
	var docNo, plan, status, remark, created string
	var reserved int
	err := s.DB.QueryRow(`SELECT doc_no, order_id, COALESCE(plan_ship_date,''), status, reserved, COALESCE(warehouse_id,3), COALESCE(remark,''), created_at
		FROM sl_pre_shipment WHERE id=?`, id).Scan(&docNo, &orderID, &plan, &status, &reserved, &wh, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	rows, _ := s.DB.Query(`SELECT id, COALESCE(order_line_id,0), product_id, qty FROM sl_pre_shipment_line WHERE pre_shipment_id=?`, id)
	lines := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, olid, pid int64
			var qty float64
			_ = rows.Scan(&lid, &olid, &pid, &qty)
			var pname string
			_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM prd_product WHERE id=?`, pid).Scan(&pname)
			lines = append(lines, gin.H{"id": lid, "order_line_id": olid, "product_id": pid, "product_name": pname, "qty": qty})
		}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "order_id": orderID, "plan_ship_date": plan, "status": status,
		"reserved": reserved == 1, "warehouse_id": wh, "remark": remark, "created_at": created, "lines": lines,
		"order": s.loadSalesOrder(orderID),
	}
}

// ---------- deliveries ----------

func (s *Services) handleDeliveries(c *gin.Context, method, action, path string) bool {
	switch {
	case strings.Contains(path, "/approve") || action == "action:approve":
		return s.deliverySetStatus(c, "approved")
	case strings.Contains(path, "/reject") || action == "action:reject":
		return s.deliveryReject(c)
	case strings.Contains(path, "/ship") || action == "action:ship":
		return s.deliveryShip(c)
	case strings.Contains(path, "/resubmit") || action == "action:resubmit":
		return s.deliveryResubmit(c)
	case strings.Contains(path, "/receive") || action == "action:receive":
		return s.deliveryReceive(c)
	case action == "list":
		return s.listDeliveries(c)
	case action == "create":
		return s.createDelivery(c)
	case action == "get":
		m := s.loadDelivery(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if s.refusePortalMismatch(c, s.deliveryCustomerID(paramID(c))) {
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE sl_delivery_approval SET logistics_no=COALESCE(NULLIF(?,''),logistics_no), remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
			strOr(body["logistics_no"]), strOr(body["remark"]), id)
		api.OK(c, s.loadDelivery(id))
		return true
	}
	return false
}

func (s *Services) listDeliveries(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if st := c.Query("status"); st != "" {
		where += ` AND d.status=?`
		args = append(args, st)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		where += ` AND (d.doc_no LIKE ? OR COALESCE(o.doc_no,'') LIKE ? OR COALESCE(d.logistics_no,'') LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like)
	}
	applyPortalCustomerSQL(c, "o.customer_id", &where, &args)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_delivery_approval d LEFT JOIN sl_sales_order o ON o.id=d.order_id `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT d.id, d.doc_no, d.order_id, COALESCE(o.doc_no,''), d.status, COALESCE(d.warehouse_id,3),
		COALESCE(d.logistics_no,''), COALESCE(d.shipped_at,''), d.created_at,
		COALESCE(d.reject_reason,''), COALESCE(d.received_at,''), COALESCE(d.receive_remark,'')
		FROM sl_delivery_approval d LEFT JOIN sl_sales_order o ON o.id=d.order_id
		`+where+` ORDER BY d.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, oid, wh int64
		var docNo, odoc, st, logistics, shipped, created, rejectReason, received, receiveRemark string
		_ = rows.Scan(&id, &docNo, &oid, &odoc, &st, &wh, &logistics, &shipped, &created, &rejectReason, &received, &receiveRemark)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "order_id": oid, "order_no": odoc, "status": st,
			"warehouse_id": wh, "logistics_no": logistics, "shipped_at": shipped, "created_at": created,
			"reject_reason": rejectReason, "received_at": received, "receive_remark": receiveRemark,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createDelivery(c *gin.Context) bool {
	body := bindBody(c)
	orderID, _ := asInt64(body["order_id"])
	if orderID <= 0 {
		api.FailJSON(c, "ORDER_REQUIRED")
		return true
	}
	ord := s.loadSalesOrder(orderID)
	if ord["id"] == nil {
		api.FailJSON(c, "ORDER_NOT_FOUND")
		return true
	}
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		wh, _ = asInt64(ord["warehouse_id"])
	}
	if wh == 0 {
		wh = 3
	}
	docNo := fmt.Sprintf("DL%s", time.Now().Format("20060102150405"))
	psID, _ := asInt64(body["pre_shipment_id"])
	res, err := s.DB.Exec(`INSERT INTO sl_delivery_approval(doc_no, order_id, pre_shipment_id, status, warehouse_id, remark)
		VALUES(?,?,?,'draft',?,?)`, docNo, orderID, nullIf0(psID), wh, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	lines := parseLines(body)
	if len(lines) == 0 {
		if ols, ok := ord["lines"].([]gin.H); ok {
			for _, ln := range ols {
				lines = append(lines, map[string]interface{}{"product_id": ln["product_id"], "qty": ln["qty"]})
			}
		}
	}
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		_, _ = s.DB.Exec(`INSERT INTO sl_delivery_line(delivery_id, product_id, qty) VALUES(?,?,?)`, id, pid, qty)
	}
	d := s.loadDelivery(id)
	var applicant int64
	if claims := middleware.Claims(c); claims != nil {
		applicant = claims.UserID
	}
	s.enqueueSalesApproval("doc_review", "delivery", strOr(d["doc_no"]),
		fmt.Sprintf("发货审批 %s", d["doc_no"]), id, applicant, 0)
	api.OK(c, d)
	return true
}

func (s *Services) deliverySetStatus(c *gin.Context, st string) bool {
	id := paramID(c)
	_, err := s.DB.Exec(`UPDATE sl_delivery_approval SET status=?, updated_at=NOW() WHERE id=?`, st, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	if st == "approved" {
		s.closeSalesApprovalQueue("delivery", id, "approved", "")
	}
	api.OK(c, s.loadDelivery(id))
	return true
}

func (s *Services) deliveryReject(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	comment := strOrDef(body["comment"], strOr(body["reject_reason"]))
	res, err := s.DB.Exec(`UPDATE sl_delivery_approval SET status='rejected', reject_reason=?, updated_at=NOW()
		WHERE id=? AND status IN ('draft','pending','approved')`, comment, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	s.closeSalesApprovalQueue("delivery", id, "rejected", comment)
	api.OK(c, s.loadDelivery(id))
	return true
}

func (s *Services) deliveryShip(c *gin.Context) bool {
	id := paramID(c)
	d := s.loadDelivery(id)
	if d["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(d["status"])
	if st != "approved" && st != "pending" && st != "draft" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	wh, _ := asInt64(d["warehouse_id"])
	if wh == 0 {
		wh = 3
	}
	orderID, _ := asInt64(d["order_id"])
	txnNo := fmt.Sprintf("ST-SO-%d", id)
	tres, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,date('now'),'draft',?,?)`,
		txnNo, "sales_out", wh, fmt.Sprintf("delivery #%d", id))
	if err != nil {
		api.FailJSON(c, "STOCK_TXN_ERROR:"+err.Error())
		return true
	}
	tid, _ := tres.LastInsertId()
	lines, _ := d["lines"].([]gin.H)
	for i, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction) VALUES(?,?,?,?,?,'out')`,
			tid, i+1, pid, qty, qty)
		if err := s.adjustBalance(wh, pid, -qty); err != nil {
			api.FailJSON(c, "INSUFFICIENT_STOCK")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE sl_sales_order_line SET delivered_qty = delivered_qty + ? WHERE order_id=? AND product_id=?`,
			qty, orderID, pid)
	}
	_, _ = s.DB.Exec(`UPDATE inv_stock_txn SET status='posted' WHERE id=?`, tid)
	_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='closed' WHERE source_doc_type='sales_order' AND source_doc_id=? AND status='active'`, orderID)
	psID, _ := asInt64(d["pre_shipment_id"])
	if psID > 0 {
		_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='closed' WHERE source_doc_type='pre_shipment' AND source_doc_id=? AND status='active'`, psID)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	body := bindBody(c)
	_, _ = s.DB.Exec(`UPDATE sl_delivery_approval SET status='shipped', shipped_at=?, logistics_no=COALESCE(NULLIF(?,''),logistics_no), updated_at=NOW() WHERE id=?`,
		now, strOr(body["logistics_no"]), id)
	_, _ = s.DB.Exec(`UPDATE sl_sales_order SET status='shipped', updated_at=NOW() WHERE id=?`, orderID)
	api.OK(c, s.loadDelivery(id))
	return true
}

func (s *Services) loadDelivery(id int64) gin.H {
	var orderID, psID, wh int64
	var docNo, status, logistics, shipped, remark, created, rejectReason, received, receiveRemark string
	err := s.DB.QueryRow(`SELECT doc_no, order_id, COALESCE(pre_shipment_id,0), status, COALESCE(warehouse_id,3),
		COALESCE(logistics_no,''), COALESCE(shipped_at,''), COALESCE(remark,''), created_at,
		COALESCE(reject_reason,''), COALESCE(received_at,''), COALESCE(receive_remark,'')
		FROM sl_delivery_approval WHERE id=?`, id).
		Scan(&docNo, &orderID, &psID, &status, &wh, &logistics, &shipped, &remark, &created, &rejectReason, &received, &receiveRemark)
	if err != nil {
		return gin.H{}
	}
	rows, _ := s.DB.Query(`SELECT id, product_id, qty, COALESCE(weight,0) FROM sl_delivery_line WHERE delivery_id=?`, id)
	lines := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, pid int64
			var qty, weight float64
			_ = rows.Scan(&lid, &pid, &qty, &weight)
			var pname string
			_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM prd_product WHERE id=?`, pid).Scan(&pname)
			lines = append(lines, gin.H{"id": lid, "product_id": pid, "product_name": pname, "qty": qty, "weight": weight})
		}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "order_id": orderID, "pre_shipment_id": psID, "status": status,
		"warehouse_id": wh, "logistics_no": logistics, "shipped_at": shipped, "remark": remark,
		"created_at": created, "lines": lines,
		"reject_reason": rejectReason, "received_at": received, "receive_remark": receiveRemark,
		"approvals":      s.loadApprovalTrail("delivery", id),
		"approval_chain": "预发货确认/手工建单 → 发货审批 → 出库发货 → 签收 → 出厂结算",
	}
}

// ---------- price locks / contracts / quotes / rankings / bom / cost / calc / prints / self ----------

func (s *Services) handlePriceLocks(c *gin.Context, method, action string) bool {
	path := c.Request.URL.Path
	if strings.Contains(path, "/activate") || action == "action:activate" {
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE sl_price_lock SET status='active' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "active"})
		return true
	}
	if strings.Contains(path, "/deactivate") || action == "action:deactivate" {
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE sl_price_lock SET status='inactive' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "inactive"})
		return true
	}
	switch {
	case action == "list":
		pageNum, pageSize := sqlutil.Page(c)
		where := `WHERE 1=1`
		args := []interface{}{}
		if st := c.Query("status"); st != "" {
			where += ` AND p.status=?`
			args = append(args, st)
		}
		if cid, ok := asInt64(c.Query("customer_id")); ok && cid > 0 {
			where += ` AND p.customer_id=?`
			args = append(args, cid)
		}
		applyPortalCustomerSQL(c, "p.customer_id", &where, &args)
		if pid, ok := asInt64(c.Query("product_id")); ok && pid > 0 {
			where += ` AND p.product_id=?`
			args = append(args, pid)
		}
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_price_lock p `+where, args...).Scan(&total)
		args = append(args, pageSize, (pageNum-1)*pageSize)
		rows, _ := s.DB.Query(`SELECT p.id, p.customer_id, COALESCE(c.name,''), p.product_id, COALESCE(pr.name,''),
			p.lock_price, p.effective_from, COALESCE(p.effective_to,''), p.status, p.created_at, COALESCE(p.version_no,1)
			FROM sl_price_lock p
			LEFT JOIN crm_customer c ON c.id=p.customer_id
			LEFT JOIN prd_product pr ON pr.id=p.product_id
			`+where+` ORDER BY p.id DESC LIMIT ? OFFSET ?`, args...)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, cid, pid, ver int64
				var cname, pname, from, to, st, created string
				var price float64
				_ = rows.Scan(&id, &cid, &cname, &pid, &pname, &price, &from, &to, &st, &created, &ver)
				list = append(list, gin.H{
					"id": id, "customer_id": cid, "customer_name": cname, "product_id": pid, "product_name": pname,
					"lock_price": price, "effective_from": from, "effective_to": to, "status": st, "created_at": created,
					"version_no": ver,
				})
			}
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case action == "create":
		body := bindBody(c)
		cid, _ := asInt64(body["customer_id"])
		pid, _ := asInt64(body["product_id"])
		price, _ := asFloat(body["lock_price"])
		if cid <= 0 || pid <= 0 || price <= 0 {
			api.FailJSON(c, "INVALID_PAYLOAD")
			return true
		}
		from := strOrDef(body["effective_from"], time.Now().Format("2006-01-02"))
		res, err := s.DB.Exec(`INSERT INTO sl_price_lock(customer_id, product_id, lock_price, effective_from, effective_to, status, remark)
			VALUES(?,?,?,?,?,?,?)`, cid, pid, price, from, strOr(body["effective_to"]), strOrDef(body["status"], "active"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "customer_id": cid, "product_id": pid, "lock_price": price, "effective_from": from, "status": "active"})
		return true
	case action == "update" || action == "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE sl_price_lock SET lock_price=COALESCE(NULLIF(?,0),lock_price), effective_to=COALESCE(NULLIF(?,''),effective_to),
			status=COALESCE(NULLIF(?,''),status), remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
			func() float64 { f, _ := asFloat(body["lock_price"]); return f }(), strOr(body["effective_to"]), strOr(body["status"]), strOr(body["remark"]), id)
		api.OK(c, gin.H{"id": id})
		return true
	case action == "delete":
		_, _ = s.DB.Exec(`UPDATE sl_price_lock SET status='inactive' WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	case action == "get":
		id := paramID(c)
		var cid, pid int64
		var price float64
		var from, to, st string
		err := s.DB.QueryRow(`SELECT customer_id, product_id, lock_price, effective_from, COALESCE(effective_to,''), status FROM sl_price_lock WHERE id=?`, id).
			Scan(&cid, &pid, &price, &from, &to, &st)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if s.refusePortalMismatch(c, cid) {
			return true
		}
		api.OK(c, gin.H{"id": id, "customer_id": cid, "product_id": pid, "lock_price": price, "effective_from": from, "effective_to": to, "status": st})
		return true
	}
	return false
}

func (s *Services) handleContracts(c *gin.Context, method, action string) bool {
	path := c.Request.URL.Path
	if strings.Contains(path, "/activate") || action == "action:activate" {
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE sl_contract SET status='active', signed_at=COALESCE(NULLIF(signed_at,''),?), updated_at=NOW() WHERE id=? AND COALESCE(is_deleted,0)=0`,
			time.Now().Format("2006-01-02"), id)
		return s.contractGet(c, id)
	}
	switch {
	case action == "list":
		pageNum, pageSize := sqlutil.Page(c)
		where := `WHERE COALESCE(ct.is_deleted,0)=0`
		args := []interface{}{}
		if st := c.Query("status"); st != "" {
			where += ` AND ct.status=?`
			args = append(args, st)
		}
		if cid, ok := asInt64(c.Query("customer_id")); ok && cid > 0 {
			where += ` AND ct.customer_id=?`
			args = append(args, cid)
		}
		applyPortalCustomerSQL(c, "ct.customer_id", &where, &args)
		if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
			where += ` AND (ct.doc_no LIKE ? OR COALESCE(ct.title,'') LIKE ? OR COALESCE(c.name,'') LIKE ?)`
			like := "%" + kw + "%"
			args = append(args, like, like, like)
		}
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_contract ct LEFT JOIN crm_customer c ON c.id=ct.customer_id `+where, args...).Scan(&total)
		args = append(args, pageSize, (pageNum-1)*pageSize)
		rows, _ := s.DB.Query(`SELECT ct.id, ct.doc_no, ct.customer_id, COALESCE(c.name,''), COALESCE(ct.title,''), ct.amount, ct.status, ct.created_at,
			COALESCE(ct.order_id,0), COALESCE(ct.attachment_url,''), COALESCE(ct.signed_at,''), COALESCE(ct.expire_at,'')
			FROM sl_contract ct LEFT JOIN crm_customer c ON c.id=ct.customer_id
			`+where+` ORDER BY ct.id DESC LIMIT ? OFFSET ?`, args...)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, cid, oid int64
				var docNo, cname, title, st, created, attach, signed, expire string
				var amount float64
				_ = rows.Scan(&id, &docNo, &cid, &cname, &title, &amount, &st, &created, &oid, &attach, &signed, &expire)
				list = append(list, gin.H{
					"id": id, "doc_no": docNo, "customer_id": cid, "customer_name": cname, "title": title,
					"amount": amount, "status": st, "created_at": created, "order_id": oid,
					"attachment_url": attach, "signed_at": signed, "expire_at": expire,
				})
			}
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case action == "create":
		body := bindBody(c)
		cid, _ := asInt64(body["customer_id"])
		if cid <= 0 {
			api.FailJSON(c, "CUSTOMER_REQUIRED")
			return true
		}
		docNo := fmt.Sprintf("CT%s", time.Now().Format("20060102150405"))
		amount, _ := asFloat(body["amount"])
		oid, _ := asInt64(body["order_id"])
		res, err := s.DB.Exec(`INSERT INTO sl_contract(doc_no, customer_id, title, amount, status, signed_at, expire_at, remark, order_id, attachment_url)
			VALUES(?,?,?,?,?,?,?,?,?,?)`, docNo, cid, strOr(body["title"]), amount, strOrDef(body["status"], "draft"),
			strOr(body["signed_at"]), strOr(body["expire_at"]), strOr(body["remark"]), nullIf0(oid), strOr(body["attachment_url"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		return s.contractGet(c, id)
	case action == "get", action == "update", action == "replace", action == "delete":
		id := paramID(c)
		if action == "delete" {
			_, _ = s.DB.Exec(`UPDATE sl_contract SET is_deleted=1 WHERE id=?`, id)
			api.OK(c, gin.H{})
			return true
		}
		if action == "update" || action == "replace" {
			body := bindBody(c)
			oid, _ := asInt64(body["order_id"])
			_, _ = s.DB.Exec(`UPDATE sl_contract SET title=COALESCE(NULLIF(?,''),title), amount=COALESCE(NULLIF(?,0),amount),
				status=COALESCE(NULLIF(?,''),status), remark=COALESCE(NULLIF(?,''),remark),
				attachment_url=COALESCE(NULLIF(?,''),attachment_url), order_id=COALESCE(NULLIF(?,0),order_id),
				signed_at=COALESCE(NULLIF(?,''),signed_at), expire_at=COALESCE(NULLIF(?,''),expire_at), updated_at=NOW() WHERE id=?`,
				strOr(body["title"]), func() float64 { f, _ := asFloat(body["amount"]); return f }(), strOr(body["status"]), strOr(body["remark"]),
				strOr(body["attachment_url"]), oid, strOr(body["signed_at"]), strOr(body["expire_at"]), id)
		}
		return s.contractGet(c, id)
	}
	return false
}

func (s *Services) contractGet(c *gin.Context, id int64) bool {
	var cid, oid int64
	var docNo, title, st, remark, attach, signed, expire, created string
	var amount float64
	err := s.DB.QueryRow(`SELECT doc_no, customer_id, COALESCE(title,''), amount, status, COALESCE(remark,''),
		COALESCE(order_id,0), COALESCE(attachment_url,''), COALESCE(signed_at,''), COALESCE(expire_at,''), created_at
		FROM sl_contract WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&docNo, &cid, &title, &amount, &st, &remark, &oid, &attach, &signed, &expire, &created)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if s.refusePortalMismatch(c, cid) {
		return true
	}
	cust := s.loadCustomer(cid)
	related := []gin.H{}
	orows, _ := s.DB.Query(`SELECT id, doc_no, status, COALESCE(total_amount,0), created_at FROM sl_sales_order
		WHERE COALESCE(is_deleted,0)=0 AND (contract_id=? OR customer_id=?) ORDER BY id DESC LIMIT 20`, id, cid)
	if orows != nil {
		defer orows.Close()
		for orows.Next() {
			var oid2 int64
			var odoc, ost, ocreated string
			var amt float64
			_ = orows.Scan(&oid2, &odoc, &ost, &amt, &ocreated)
			related = append(related, gin.H{"id": oid2, "doc_no": odoc, "status": ost, "total_amount": amt, "created_at": ocreated})
		}
	}
	api.OK(c, gin.H{
		"id": id, "doc_no": docNo, "customer_id": cid, "customer_name": cust["name"], "title": title,
		"amount": amount, "status": st, "remark": remark, "order_id": oid, "attachment_url": attach,
		"signed_at": signed, "expire_at": expire, "created_at": created, "related_orders": related,
	})
	return true
}

func (s *Services) handleQuoteHistories(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if cid, ok := asInt64(c.Query("customer_id")); ok && cid > 0 {
		where += ` AND q.customer_id=?`
		args = append(args, cid)
	}
	applyPortalCustomerSQL(c, "q.customer_id", &where, &args)
	if pid, ok := asInt64(c.Query("product_id")); ok && pid > 0 {
		where += ` AND q.product_id=?`
		args = append(args, pid)
	}
	if from := c.Query("date_from"); from != "" {
		where += ` AND q.quoted_at>=?`
		args = append(args, from)
	}
	if to := c.Query("date_to"); to != "" {
		where += ` AND q.quoted_at<=?`
		args = append(args, to+" 23:59:59")
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_quote_history q `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT q.id, q.customer_id, COALESCE(c.name,''), q.product_id, COALESCE(p.name,''), q.price, q.quoted_at,
		COALESCE(q.inquiry_id,0), COALESCE(q.order_id,0)
		FROM sl_quote_history q
		LEFT JOIN crm_customer c ON c.id=q.customer_id
		LEFT JOIN prd_product p ON p.id=q.product_id
		`+where+` ORDER BY q.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, cid, pid, iq, oid int64
		var cname, pname, quoted string
		var price float64
		_ = rows.Scan(&id, &cid, &cname, &pid, &pname, &price, &quoted, &iq, &oid)
		list = append(list, gin.H{
			"id": id, "customer_id": cid, "customer_name": cname, "product_id": pid, "product_name": pname,
			"price": price, "quoted_at": quoted, "inquiry_id": iq, "order_id": oid,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) handleSalesRankings(c *gin.Context, method, path string) bool {
	if strings.Contains(path, "/configs") {
		if method == "PUT" {
			body := bindBody(c)
			metric := strOrDef(body["metric"], "amount")
			topN, _ := asInt64(body["top_n"])
			if topN <= 0 {
				topN = 10
			}
			_, _ = s.DB.Exec(`UPDATE sl_sales_rank_config SET metric=?, top_n=?, updated_at=NOW() WHERE id=1`, metric, topN)
			api.OK(c, gin.H{"metric": metric, "top_n": topN})
			return true
		}
		rows, _ := s.DB.Query(`SELECT id, metric, period, top_n FROM sl_sales_rank_config ORDER BY id`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, topN int64
				var metric, period string
				_ = rows.Scan(&id, &metric, &period, &topN)
				list = append(list, gin.H{"id": id, "metric": metric, "period": period, "top_n": topN})
			}
		}
		api.OK(c, gin.H{"list": list})
		return true
	}
	period := strOrDef(c.Query("period"), "all")
	where := `WHERE COALESCE(o.is_deleted,0)=0 AND o.status NOT IN ('cancelled','draft')`
	args := []interface{}{}
	switch period {
	case "month":
		where += ` AND o.created_at>=?`
		args = append(args, time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	case "quarter":
		where += ` AND o.created_at>=?`
		args = append(args, time.Now().AddDate(0, -3, 0).Format("2006-01-02"))
	case "year":
		where += ` AND o.created_at>=?`
		args = append(args, time.Now().AddDate(-1, 0, 0).Format("2006-01-02"))
	}
	q := `SELECT o.customer_id, COALESCE(c.name,''), COUNT(1), SUM(o.total_amount)
		FROM sl_sales_order o LEFT JOIN crm_customer c ON c.id=o.customer_id
		` + where + ` GROUP BY o.customer_id ORDER BY SUM(o.total_amount) DESC LIMIT 20`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	rank := 1
	for rows.Next() {
		var cid, cnt int64
		var name string
		var amount float64
		_ = rows.Scan(&cid, &name, &cnt, &amount)
		list = append(list, gin.H{"rank": rank, "customer_id": cid, "customer_name": name, "order_count": cnt, "amount": amount})
		rank++
	}
	api.OK(c, gin.H{"list": list, "metric": "amount", "period": period})
	return true
}

func (s *Services) handleSalesBOMs(c *gin.Context, method, action, path string) bool {
	if strings.Contains(path, "/deactivate") || action == "action:deactivate" {
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE sl_sales_bom SET status='inactive' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "inactive"})
		return true
	}
	if strings.Contains(path, "/lines") {
		id := paramID(c)
		if method == "GET" {
			rows, _ := s.DB.Query(`SELECT id, material_product_id, qty, scrap_rate, COALESCE(remark,'') FROM sl_sales_bom_line WHERE bom_id=?`, id)
			list := []gin.H{}
			if rows != nil {
				defer rows.Close()
				for rows.Next() {
					var lid, mid int64
					var qty, scrap float64
					var remark string
					_ = rows.Scan(&lid, &mid, &qty, &scrap, &remark)
					list = append(list, gin.H{"id": lid, "material_product_id": mid, "qty": qty, "scrap_rate": scrap, "remark": remark})
				}
			}
			api.OK(c, gin.H{"bom_id": id, "lines": list})
			return true
		}
		body := bindBody(c)
		_, _ = s.DB.Exec(`DELETE FROM sl_sales_bom_line WHERE bom_id=?`, id)
		for _, ln := range parseLines(body) {
			mid, _ := asInt64(ln["material_product_id"])
			if mid == 0 {
				mid, _ = asInt64(ln["product_id"])
			}
			qty, _ := asFloat(ln["qty"])
			scrap, _ := asFloat(ln["scrap_rate"])
			_, _ = s.DB.Exec(`INSERT INTO sl_sales_bom_line(bom_id, material_product_id, qty, scrap_rate, remark) VALUES(?,?,?,?,?)`,
				id, mid, qty, scrap, strOr(ln["remark"]))
		}
		api.OK(c, gin.H{"bom_id": id, "saved": true})
		return true
	}
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_sales_bom`).Scan(&total)
		rows, _ := s.DB.Query(`SELECT b.id, b.doc_no, COALESCE(b.order_id,0), b.product_id, COALESCE(p.name,''), COALESCE(b.name,''), b.status, b.created_at
			FROM sl_sales_bom b LEFT JOIN prd_product p ON p.id=b.product_id
			ORDER BY b.id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, oid, pid int64
				var docNo, pname, name, st, created string
				_ = rows.Scan(&id, &docNo, &oid, &pid, &pname, &name, &st, &created)
				list = append(list, gin.H{"id": id, "doc_no": docNo, "order_id": oid, "product_id": pid, "product_name": pname, "name": name, "status": st, "created_at": created})
			}
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		pid, _ := asInt64(body["product_id"])
		if pid <= 0 {
			pid = 3
		}
		docNo := fmt.Sprintf("SB%s", time.Now().Format("20060102150405"))
		oid, _ := asInt64(body["order_id"])
		res, err := s.DB.Exec(`INSERT INTO sl_sales_bom(doc_no, order_id, product_id, name, status) VALUES(?,?,?,?,'active')`,
			docNo, nullIf0(oid), pid, strOrDef(body["name"], "销售BOM"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "product_id": pid})
		return true
	case "get", "update", "replace":
		id := paramID(c)
		if action != "get" {
			body := bindBody(c)
			_, _ = s.DB.Exec(`UPDATE sl_sales_bom SET name=COALESCE(NULLIF(?,''),name), status=COALESCE(NULLIF(?,''),status),
				remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
				strOr(body["name"]), strOr(body["status"]), strOr(body["remark"]), id)
			if lines := parseLines(body); len(lines) > 0 {
				_, _ = s.DB.Exec(`DELETE FROM sl_sales_bom_line WHERE bom_id=?`, id)
				for _, ln := range lines {
					mid, _ := asInt64(ln["material_product_id"])
					if mid == 0 {
						mid, _ = asInt64(ln["product_id"])
					}
					qty, _ := asFloat(ln["qty"])
					scrap, _ := asFloat(ln["scrap_rate"])
					_, _ = s.DB.Exec(`INSERT INTO sl_sales_bom_line(bom_id, material_product_id, qty, scrap_rate, remark) VALUES(?,?,?,?,?)`,
						id, mid, qty, scrap, strOr(ln["remark"]))
				}
			}
		}
		var oid, pid, ver int64
		var docNo, name, st, remark string
		err := s.DB.QueryRow(`SELECT doc_no, COALESCE(order_id,0), product_id, COALESCE(name,''), status, COALESCE(version_no,1), COALESCE(remark,'') FROM sl_sales_bom WHERE id=?`, id).
			Scan(&docNo, &oid, &pid, &name, &st, &ver, &remark)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		lrows, _ := s.DB.Query(`SELECT id, material_product_id, qty, scrap_rate, COALESCE(remark,'') FROM sl_sales_bom_line WHERE bom_id=?`, id)
		lines := []gin.H{}
		if lrows != nil {
			defer lrows.Close()
			for lrows.Next() {
				var lid, mid int64
				var qty, scrap float64
				var rmk string
				_ = lrows.Scan(&lid, &mid, &qty, &scrap, &rmk)
				var pname string
				_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM prd_product WHERE id=?`, mid).Scan(&pname)
				lines = append(lines, gin.H{"id": lid, "material_product_id": mid, "product_name": pname, "qty": qty, "scrap_rate": scrap, "remark": rmk})
			}
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "order_id": oid, "product_id": pid, "name": name, "status": st, "version_no": ver, "remark": remark, "lines": lines})
		return true
	}
	return false
}

func (s *Services) handleCostBudgets(c *gin.Context, method, action, path string) bool {
	if strings.Contains(path, "/recalc") || action == "action:recalc" {
		return s.recalcCostBudget(c)
	}
	if method == "POST" || action == "create" {
		body := bindBody(c)
		oid, _ := asInt64(body["order_id"])
		if oid <= 0 {
			api.FailJSON(c, "ORDER_REQUIRED")
			return true
		}
		ord := s.loadSalesOrder(oid)
		saleAmt, _ := asFloat(ord["total_amount"])
		mat, _ := asFloat(body["material_cost"])
		labor, _ := asFloat(body["labor_cost"])
		other, _ := asFloat(body["other_cost"])
		if mat <= 0 {
			mat = saleAmt * 0.55
		}
		if labor <= 0 {
			labor = saleAmt * 0.15
		}
		total := mat + labor + other
		margin := 0.0
		if saleAmt > 0 {
			margin = (saleAmt - total) / saleAmt
		}
		res, err := s.DB.Exec(`INSERT INTO sl_cost_budget(order_id, material_cost, labor_cost, other_cost, total_cost, sale_amount, margin, remark)
			VALUES(?,?,?,?,?,?,?,?)`, oid, mat, labor, other, total, saleAmt, margin, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "order_id": oid, "material_cost": mat, "labor_cost": labor, "other_cost": other, "total_cost": total, "sale_amount": saleAmt, "margin": margin})
		return true
	}
	if strings.Contains(path, "{order_id}") || c.Param("order_id") != "" {
		oid := paramID(c)
		if v := c.Param("order_id"); v != "" {
			fmt.Sscan(v, &oid)
		}
		var id int64
		var mat, labor, other, total, sale, margin float64
		err := s.DB.QueryRow(`SELECT id, material_cost, labor_cost, other_cost, total_cost, sale_amount, margin FROM sl_cost_budget WHERE order_id=? ORDER BY id DESC LIMIT 1`, oid).
			Scan(&id, &mat, &labor, &other, &total, &sale, &margin)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{"id": id, "order_id": oid, "material_cost": mat, "labor_cost": labor, "other_cost": other, "total_cost": total, "sale_amount": sale, "margin": margin})
		return true
	}
	pageNum, pageSize := sqlutil.Page(c)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_cost_budget`).Scan(&total)
	rows, _ := s.DB.Query(`SELECT id, order_id, material_cost, labor_cost, other_cost, total_cost, sale_amount, margin, created_at
		FROM sl_cost_budget ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	list := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, oid int64
			var mat, labor, other, tot, sale, margin float64
			var created string
			_ = rows.Scan(&id, &oid, &mat, &labor, &other, &tot, &sale, &margin, &created)
			list = append(list, gin.H{"id": id, "order_id": oid, "material_cost": mat, "labor_cost": labor, "other_cost": other, "total_cost": tot, "sale_amount": sale, "margin": margin, "created_at": created})
		}
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) handleQuoteCalculator(c *gin.Context, method, path string) bool {
	if strings.Contains(path, "/calc") || (method == "POST" && strings.Contains(c.Request.URL.Path, "/calc")) {
		body := bindBody(c)
		pid, _ := asInt64(body["product_id"])
		if pid <= 0 {
			pid = 3
		}
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			qty = 1
		}
		base, _ := asFloat(body["base_cost"])
		if base <= 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(cost_price,0) FROM prd_product WHERE id=?`, pid).Scan(&base)
		}
		margin, _ := asFloat(body["margin_rate"])
		if margin <= 0 {
			margin = 0.2
		}
		quote := base * (1 + margin)
		api.OK(c, gin.H{"product_id": pid, "qty": qty, "base_cost": base, "margin_rate": margin, "quote_price": quote, "amount": quote * qty})
		return true
	}
	if strings.Contains(path, "/apply") || strings.Contains(c.Request.URL.Path, "/apply") {
		body := bindBody(c)
		pid, _ := asInt64(body["product_id"])
		cid, _ := asInt64(body["customer_id"])
		if bound, ok := portalCustomerID(c); ok {
			cid = bound
		}
		qty, _ := asFloat(body["qty"])
		base, _ := asFloat(body["base_cost"])
		margin, _ := asFloat(body["margin_rate"])
		quote, _ := asFloat(body["quote_price"])
		b, _ := json.Marshal(body)
		res, err := s.DB.Exec(`INSERT INTO sl_quote_calculator_result(customer_id, product_id, qty, base_cost, margin_rate, quote_price, payload_json)
			VALUES(?,?,?,?,?,?,?)`, nullIf0(cid), pid, qty, base, margin, quote, string(b))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		if cid > 0 && quote > 0 {
			_, _ = s.DB.Exec(`INSERT INTO sl_quote_history(customer_id, product_id, price, quoted_at) VALUES(?,?,?,NOW())`, cid, pid, quote)
		}
		api.OK(c, gin.H{"id": id, "saved": true, "quote_price": quote})
		return true
	}
	api.OK(c, gin.H{
		"defaults": gin.H{"margin_rate": 0.2, "product_id": 3},
		"hint":     "POST /calc 试算，POST /apply 回写历史报价",
	})
	return true
}

func (s *Services) handleSalesPrints(c *gin.Context, method, action string) bool {
	if method == "POST" || action == "create" {
		body := bindBody(c)
		docType := strOrDef(body["doc_type"], "sales_order")
		docID, _ := asInt64(body["doc_id"])
		docNo := strOr(body["doc_no"])
		payload := gin.H{}
		switch docType {
		case "sales_order":
			payload = s.loadSalesOrder(docID)
			docNo = strOr(payload["doc_no"])
		case "delivery":
			payload = s.loadDelivery(docID)
			docNo = strOr(payload["doc_no"])
		case "pre_shipment":
			payload = s.loadPreShip(docID)
			docNo = strOr(payload["doc_no"])
		}
		b, _ := json.Marshal(payload)
		var uid int64
		if claims := middleware.Claims(c); claims != nil {
			uid = claims.UserID
		}
		res, err := s.DB.Exec(`INSERT INTO sl_print_log(doc_type, doc_id, doc_no, template_code, printed_by, payload_json)
			VALUES(?,?,?,?,?,?)`, docType, docID, docNo, strOrDef(body["template_code"], "default"), uid, string(b))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_type": docType, "doc_id": docID, "doc_no": docNo, "preview": payload})
		return true
	}
	pageNum, pageSize := sqlutil.Page(c)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_print_log`).Scan(&total)
	rows, _ := s.DB.Query(`SELECT id, doc_type, doc_id, COALESCE(doc_no,''), COALESCE(template_code,''), printed_at
		FROM sl_print_log ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	list := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, docID int64
			var docType, docNo, tpl, printed string
			_ = rows.Scan(&id, &docType, &docID, &docNo, &tpl, &printed)
			list = append(list, gin.H{"id": id, "doc_type": docType, "doc_id": docID, "doc_no": docNo, "template_code": tpl, "printed_at": printed})
		}
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) handleSelfOrders(c *gin.Context, method string) bool {
	if method == "GET" {
		rows, _ := s.DB.Query(`SELECT id, name, enabled, min_qty, COALESCE(max_qty,0), COALESCE(max_amount,0), COALESCE(allow_products_json,''), COALESCE(remark,'') FROM sl_self_order_rule ORDER BY id`)
		list := []gin.H{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var name, products, remark string
				var enabled int
				var minQty, maxQty, maxAmt float64
				_ = rows.Scan(&id, &name, &enabled, &minQty, &maxQty, &maxAmt, &products, &remark)
				list = append(list, gin.H{"id": id, "name": name, "enabled": enabled == 1, "min_qty": minQty, "max_qty": maxQty, "max_amount": maxAmt, "allow_products_json": products, "remark": remark})
			}
		}
		api.OK(c, gin.H{"rules": list, "customers": "使用 CRM 客户下单"})
		return true
	}
	// POST submit -> create sales order source=self
	return s.createSalesOrder(c, true)
}
