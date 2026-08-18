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

func (s *Services) handleSales(c *gin.Context, method, openapiPath, action string) bool {
	s.ensureSalesLoopColumns()
	if !s.bindPortalCustomer(c) {
		return true
	}
	if isCustomerClient(c) && !customerSalesAllowed(method, openapiPath, action) {
		api.FailJSON(c, "PERM_DENIED")
		return true
	}
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/sales/outbound-settles"):
		return s.handleOutboundSettles(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/self-order-rules"):
		return s.handleSelfOrderRules(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/orders"):
		return s.handleSalesOrders(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/inquiries"):
		return s.handleSalesInquiries(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/pre-shipments"):
		return s.handlePreShipments(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/deliveries"):
		return s.handleDeliveries(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/price-locks"):
		return s.handlePriceLocks(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/contracts"):
		return s.handleContracts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/quote-histories"):
		return s.handleQuoteHistories(c)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/rankings"):
		return s.handleSalesRankings(c, method, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/sales-boms"):
		return s.handleSalesBOMs(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/cost-budgets"):
		return s.handleCostBudgets(c, method, action, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/quote-calculator"):
		return s.handleQuoteCalculator(c, method, openapiPath)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/prints"):
		return s.handleSalesPrints(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/my-orders"):
		return s.handleMyOrders(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/sales/self-orders"):
		return s.handleSelfOrders(c, method)
	default:
		return false
	}
}

func (s *Services) resolveLockPrice(customerID, productID int64) (float64, int64, bool) {
	var price float64
	var id int64
	today := time.Now().Format("2006-01-02")
	err := s.DB.QueryRow(`SELECT id, lock_price FROM sl_price_lock
		WHERE customer_id=? AND product_id=? AND status='active'
		AND effective_from<=? AND (effective_to IS NULL OR effective_to='' OR effective_to>=?)
		ORDER BY id DESC LIMIT 1`, customerID, productID, today, today).Scan(&id, &price)
	return price, id, err == nil
}

func (s *Services) productSalePrice(productID int64) float64 {
	var p float64
	_ = s.DB.QueryRow(`SELECT COALESCE(sale_price,0) FROM prd_product WHERE id=?`, productID).Scan(&p)
	return p
}

func parseLines(body map[string]interface{}) []map[string]interface{} {
	out := []map[string]interface{}{}
	raw, ok := body["lines"].([]interface{})
	if !ok {
		return out
	}
	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// ---------- orders ----------

func (s *Services) handleSalesOrders(c *gin.Context, method, action, path string) bool {
	switch {
	case strings.Contains(path, "/cancel") || action == "action:cancel":
		return s.orderAction(c, "cancelled")
	case strings.Contains(path, "/submit") || action == "action:submit":
		return s.orderAction(c, "submitted")
	case strings.Contains(path, "/rebuy") || action == "action:rebuy":
		return s.rebuyOrder(c)
	case strings.Contains(path, "/changes"):
		if method == "GET" {
			return s.listOrderChanges(c)
		}
		return s.createOrderChange(c)
	case action == "list":
		return s.listSalesOrders(c, 0)
	case action == "create":
		return s.createSalesOrder(c, false)
	case action == "get":
		m := s.loadSalesOrder(paramID(c))
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
		return s.updateSalesOrder(c)
	}
	return false
}

func (s *Services) listSalesOrders(c *gin.Context, ownerUserID int64) bool {
	pageNum, pageSize := sqlutil.Page(c)
	status := c.Query("status")
	where := `WHERE COALESCE(o.is_deleted,0)=0`
	args := []interface{}{}
	if ownerUserID > 0 {
		where += ` AND o.owner_user_id=?`
		args = append(args, ownerUserID)
	}
	applyPortalCustomerSQL(c, "o.customer_id", &where, &args)
	if status != "" {
		where += ` AND o.status=?`
		args = append(args, status)
	}
	if cid, ok := asInt64(c.Query("customer_id")); ok && cid > 0 {
		where += ` AND o.customer_id=?`
		args = append(args, cid)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		where += ` AND (o.doc_no LIKE ? OR COALESCE(c.name,'') LIKE ? OR COALESCE(o.remark,'') LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_sales_order o LEFT JOIN crm_customer c ON c.id=o.customer_id `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT o.id, o.doc_no, o.customer_id, COALESCE(c.name,''), o.status, o.source,
		COALESCE(o.total_amount,0), COALESCE(o.warehouse_id,3), COALESCE(o.remark,''), o.created_at
		FROM sl_sales_order o LEFT JOIN crm_customer c ON c.id=o.customer_id
		`+where+` ORDER BY o.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, cid, wh int64
		var docNo, cname, st, source, remark, created string
		var amount float64
		_ = rows.Scan(&id, &docNo, &cid, &cname, &st, &source, &amount, &wh, &remark, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "customer_id": cid, "customer_name": cname, "status": st,
			"source": source, "total_amount": amount, "warehouse_id": wh, "remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createSalesOrder(c *gin.Context, fromSelf bool) bool {
	body := bindBody(c)
	customerID, _ := asInt64(body["customer_id"])
	if cid, ok := portalCustomerID(c); ok {
		customerID = cid
	}
	if customerID <= 0 {
		api.FailJSON(c, "CUSTOMER_REQUIRED")
		return true
	}
	if s.loadCustomer(customerID)["id"] == nil {
		api.FailJSON(c, "CUSTOMER_NOT_FOUND")
		return true
	}
	lines := parseLines(body)
	if len(lines) == 0 {
		api.FailJSON(c, "LINES_REQUIRED")
		return true
	}
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		wh = 3
	}
	source := strOrDef(body["source"], "manual")
	if fromSelf {
		source = "self"
		var minQty, maxQty, maxAmt float64
		var enabled int
		_ = s.DB.QueryRow(`SELECT enabled, min_qty, COALESCE(max_qty,0), COALESCE(max_amount,0) FROM sl_self_order_rule WHERE enabled=1 ORDER BY id DESC LIMIT 1`).
			Scan(&enabled, &minQty, &maxQty, &maxAmt)
		if enabled == 1 {
			var qtySum, amtGuess float64
			for _, ln := range lines {
				q, _ := asFloat(ln["qty"])
				p, _ := asFloat(ln["price"])
				qtySum += q
				amtGuess += q * p
			}
			if minQty > 0 && qtySum < minQty {
				api.FailJSON(c, "SELF_ORDER_MIN_QTY")
				return true
			}
			if maxQty > 0 && qtySum > maxQty {
				api.FailJSON(c, "SELF_ORDER_MAX_QTY")
				return true
			}
			if maxAmt > 0 && amtGuess > maxAmt {
				api.FailJSON(c, "SELF_ORDER_MAX_AMOUNT")
				return true
			}
		}
	}
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("SO%s", time.Now().Format("20060102150405")))
	var ownerID int64
	if claims := middleware.Claims(c); claims != nil {
		ownerID = claims.UserID
	}
	status := strOrDef(body["status"], "draft")
	if fromSelf {
		status = "submitted"
	}
	var total float64
	var lockID int64
	type lineCalc struct {
		pid, lockID        int64
		qty, price, amount float64
	}
	calcs := []lineCalc{}
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		if pid <= 0 || qty <= 0 {
			api.FailJSON(c, "INVALID_LINE")
			return true
		}
		price, _ := asFloat(ln["price"])
		lid := int64(0)
		if lp, id, ok := s.resolveLockPrice(customerID, pid); ok {
			price = lp
			lid = id
			lockID = id
		} else if price <= 0 {
			price = s.productSalePrice(pid)
		}
		amt := qty * price
		total += amt
		calcs = append(calcs, lineCalc{pid: pid, qty: qty, price: price, amount: amt, lockID: lid})
	}
	res, err := s.DB.Exec(`INSERT INTO sl_sales_order(doc_no, customer_id, owner_user_id, status, source, price_lock_id, warehouse_id, total_amount, remark, created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		docNo, customerID, ownerID, status, source, nullIf0(lockID), wh, total, strOr(body["remark"]), ownerID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	oid, _ := res.LastInsertId()
	for _, lc := range calcs {
		_, _ = s.DB.Exec(`INSERT INTO sl_sales_order_line(order_id, product_id, qty, price, amount) VALUES(?,?,?,?,?)`,
			oid, lc.pid, lc.qty, lc.price, lc.amount)
		_, _ = s.DB.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, qty, source_doc_type, source_doc_id, status)
			VALUES(?,?,?,'sales_order',?,'active')`, wh, lc.pid, lc.qty, oid)
		_, _ = s.DB.Exec(`INSERT INTO sl_quote_history(customer_id, product_id, price, quoted_at, order_id) VALUES(?,?,?,NOW(),?)`,
			customerID, lc.pid, lc.price, oid)
	}
	api.OK(c, s.loadSalesOrder(oid))
	return true
}

func (s *Services) updateSalesOrder(c *gin.Context) bool {
	id := paramID(c)
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM sl_sales_order WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status != "draft" && status != "submitted" {
		api.FailJSON(c, "ORDER_LOCKED")
		return true
	}
	before, _ := json.Marshal(s.loadSalesOrder(id))
	body := bindBody(c)
	_, _ = s.DB.Exec(`UPDATE sl_sales_order SET remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
		strOr(body["remark"]), id)
	if lines := parseLines(body); len(lines) > 0 {
		var wh int64 = 3
		_ = s.DB.QueryRow(`SELECT COALESCE(warehouse_id,3) FROM sl_sales_order WHERE id=?`, id).Scan(&wh)
		_, _ = s.DB.Exec(`DELETE FROM sl_sales_order_line WHERE order_id=?`, id)
		_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='cancelled' WHERE source_doc_type='sales_order' AND source_doc_id=? AND status='active'`, id)
		var customerID int64
		_ = s.DB.QueryRow(`SELECT customer_id FROM sl_sales_order WHERE id=?`, id).Scan(&customerID)
		var total float64
		for _, ln := range lines {
			pid, _ := asInt64(ln["product_id"])
			qty, _ := asFloat(ln["qty"])
			price, _ := asFloat(ln["price"])
			if lp, _, ok := s.resolveLockPrice(customerID, pid); ok {
				price = lp
			} else if price <= 0 {
				price = s.productSalePrice(pid)
			}
			amt := qty * price
			total += amt
			_, _ = s.DB.Exec(`INSERT INTO sl_sales_order_line(order_id, product_id, qty, price, amount) VALUES(?,?,?,?,?)`, id, pid, qty, price, amt)
			_, _ = s.DB.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, qty, source_doc_type, source_doc_id, status)
				VALUES(?,?,?,'sales_order',?,'active')`, wh, pid, qty, id)
		}
		_, _ = s.DB.Exec(`UPDATE sl_sales_order SET total_amount=? WHERE id=?`, total, id)
	}
	after, _ := json.Marshal(s.loadSalesOrder(id))
	var uid int64
	if claims := middleware.Claims(c); claims != nil {
		uid = claims.UserID
	}
	_, _ = s.DB.Exec(`INSERT INTO sl_order_change_log(order_id, change_type, before_json, after_json, reason, created_by)
		VALUES(?,'update',?,?,?,?)`, id, string(before), string(after), strOrDef(body["reason"], "修改订单"), uid)
	api.OK(c, s.loadSalesOrder(id))
	return true
}

func (s *Services) orderAction(c *gin.Context, toStatus string) bool {
	id := paramID(c)
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM sl_sales_order WHERE id=?`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if toStatus == "cancelled" {
		_, _ = s.DB.Exec(`UPDATE inv_reservation SET status='cancelled' WHERE source_doc_type='sales_order' AND source_doc_id=? AND status='active'`, id)
	}
	if toStatus == "submitted" && status != "draft" && status != "submitted" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	_, _ = s.DB.Exec(`UPDATE sl_sales_order SET status=?, updated_at=NOW() WHERE id=?`, toStatus, id)
	if toStatus == "submitted" {
		ord := s.loadSalesOrder(id)
		var applicant int64
		if claims := middleware.Claims(c); claims != nil {
			applicant = claims.UserID
		}
		amt, _ := asFloat(ord["total_amount"])
		s.enqueueSalesApproval("doc_review", "sales_order", strOr(ord["doc_no"]),
			fmt.Sprintf("销售订单 %s", ord["doc_no"]), id, applicant, amt)
		api.OK(c, ord)
		return true
	}
	api.OK(c, s.loadSalesOrder(id))
	return true
}

func (s *Services) rebuyOrder(c *gin.Context) bool {
	srcID := paramID(c)
	src := s.loadSalesOrder(srcID)
	if src["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if s.refusePortalMismatch(c, ginHInt64(src["customer_id"])) {
		return true
	}
	body := map[string]interface{}{
		"customer_id":  src["customer_id"],
		"warehouse_id": src["warehouse_id"],
		"source":       "rebuy",
		"remark":       fmt.Sprintf("复购自 %v", src["doc_no"]),
		"lines":        src["lines"],
	}
	// inject into context-like create
	c.Set("rebuy_body", body)
	customerID, _ := asInt64(src["customer_id"])
	wh, _ := asInt64(src["warehouse_id"])
	if wh == 0 {
		wh = 3
	}
	docNo := fmt.Sprintf("SO%s", time.Now().Format("20060102150405"))
	var ownerID int64
	if claims := middleware.Claims(c); claims != nil {
		ownerID = claims.UserID
	}
	lines, _ := src["lines"].([]gin.H)
	if lines == nil {
		if arr, ok := src["lines"].([]interface{}); ok {
			for _, a := range arr {
				if m, ok := a.(map[string]interface{}); ok {
					lines = append(lines, gin.H(m))
				} else if m, ok := a.(gin.H); ok {
					lines = append(lines, m)
				}
			}
		}
	}
	var total float64
	res, err := s.DB.Exec(`INSERT INTO sl_sales_order(doc_no, customer_id, owner_user_id, status, source, reorder_from_id, warehouse_id, total_amount, remark, created_by)
		VALUES(?,?,?,'draft','rebuy',?,?,0,?,?)`, docNo, customerID, ownerID, srcID, wh, fmt.Sprintf("复购自 %v", src["doc_no"]), ownerID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	oid, _ := res.LastInsertId()
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		price, _ := asFloat(ln["price"])
		if lp, _, ok := s.resolveLockPrice(customerID, pid); ok {
			price = lp
		}
		amt := qty * price
		total += amt
		_, _ = s.DB.Exec(`INSERT INTO sl_sales_order_line(order_id, product_id, qty, price, amount) VALUES(?,?,?,?,?)`, oid, pid, qty, price, amt)
		_, _ = s.DB.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, qty, source_doc_type, source_doc_id, status)
			VALUES(?,?,?,'sales_order',?,'active')`, wh, pid, qty, oid)
	}
	_, _ = s.DB.Exec(`UPDATE sl_sales_order SET total_amount=? WHERE id=?`, total, oid)
	_ = body
	api.OK(c, s.loadSalesOrder(oid))
	return true
}

func (s *Services) listOrderChanges(c *gin.Context) bool {
	id := paramID(c)
	rows, err := s.DB.Query(`SELECT id, change_type, COALESCE(reason,''), COALESCE(created_by,0), created_at FROM sl_order_change_log WHERE order_id=? ORDER BY id DESC`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id2, uid int64
		var typ, reason, created string
		_ = rows.Scan(&id2, &typ, &reason, &uid, &created)
		list = append(list, gin.H{"id": id2, "change_type": typ, "reason": reason, "created_by": uid, "created_at": created})
	}
	api.OK(c, gin.H{"order_id": id, "list": list, "total": len(list)})
	return true
}

func (s *Services) createOrderChange(c *gin.Context) bool {
	return s.updateSalesOrder(c)
}

func (s *Services) loadSalesOrder(id int64) gin.H {
	var customerID, ownerID, wh, reorderFrom, lockID int64
	var docNo, status, source, remark, created string
	var total float64
	err := s.DB.QueryRow(`SELECT doc_no, customer_id, COALESCE(owner_user_id,0), status, source, COALESCE(price_lock_id,0),
		COALESCE(reorder_from_id,0), COALESCE(warehouse_id,3), COALESCE(total_amount,0), COALESCE(remark,''), created_at
		FROM sl_sales_order WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&docNo, &customerID, &ownerID, &status, &source, &lockID, &reorderFrom, &wh, &total, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	cust := s.loadCustomer(customerID)
	rows, _ := s.DB.Query(`SELECT id, product_id, qty, COALESCE(weight,0), price, amount, COALESCE(delivered_qty,0) FROM sl_sales_order_line WHERE order_id=?`, id)
	lines := []gin.H{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, pid int64
			var qty, weight, price, amount, delivered float64
			_ = rows.Scan(&lid, &pid, &qty, &weight, &price, &amount, &delivered)
			var pname string
			_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM prd_product WHERE id=?`, pid).Scan(&pname)
			lines = append(lines, gin.H{
				"id": lid, "product_id": pid, "product_name": pname, "qty": qty, "weight": weight,
				"price": price, "amount": amount, "delivered_qty": delivered,
			})
		}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "customer_id": customerID, "customer_name": cust["name"],
		"owner_user_id": ownerID, "status": status, "source": source, "price_lock_id": lockID,
		"reorder_from_id": reorderFrom, "warehouse_id": wh, "total_amount": total, "remark": remark,
		"created_at": created, "lines": lines,
		"approvals":      s.loadApprovalTrail("sales_order", id),
		"approval_chain": "草稿/询价转单 → 提交 → 预发货占用 → 发货审批出库 → 出厂结算",
	}
}

func (s *Services) handleMyOrders(c *gin.Context, method, action string) bool {
	var uid int64
	if claims := middleware.Claims(c); claims != nil {
		uid = claims.UserID
	}
	if _, ok := portalCustomerID(c); ok {
		uid = 0
	}
	if action == "get" {
		m := s.loadSalesOrder(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if s.refusePortalMismatch(c, ginHInt64(m["customer_id"])) {
			return true
		}
		api.OK(c, m)
		return true
	}
	return s.listSalesOrders(c, uid)
}
