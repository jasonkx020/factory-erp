package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleCRM(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/crm/customers"):
		return s.handleCRMCustomers(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/opportunities"):
		return s.handleCRMOpportunities(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/follow-ups"):
		return s.handleCRMFollowUps(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/lead-assigns"):
		return s.handleCRMLeadAssigns(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/protect-rules"):
		return s.handleCRMProtectRules(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/releases"):
		return s.handleCRMReleases(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/imports"):
		return s.handleCRMImports(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/task-reminders"):
		return s.handleCRMTaskReminders(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/crm/inquiries"):
		return s.handleCRMInquiries(c)
	default:
		return false
	}
}

func (s *Services) handleCRMCustomers(c *gin.Context, method, path, action string) bool {
	switch {
	case strings.Contains(path, "/hide"):
		id := paramID(c)
		hidden := 1
		if method == "DELETE" {
			hidden = 0
		}
		_, err := s.DB.Exec(`UPDATE crm_customer SET is_hidden=?, updated_at=NOW() WHERE id=? AND COALESCE(is_deleted,0)=0`, hidden, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadCustomer(id))
		return true
	case strings.Contains(path, "/lock"):
		id := paramID(c)
		locked := 1
		if method == "DELETE" {
			locked = 0
		}
		_, err := s.DB.Exec(`UPDATE crm_customer SET is_locked=?, updated_at=NOW() WHERE id=? AND COALESCE(is_deleted,0)=0`, locked, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadCustomer(id))
		return true
	case strings.Contains(path, "/profile"):
		id := paramID(c)
		if method == "GET" {
			m := s.loadCustomerProfile(id)
			if m["id"] == nil {
				api.FailJSON(c, "NOT_FOUND")
				return true
			}
			api.OK(c, m)
			return true
		}
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE crm_customer SET
			contact_name=COALESCE(NULLIF(?,''),contact_name),
			mobile=COALESCE(NULLIF(?,''),mobile),
			address=COALESCE(NULLIF(?,''),address),
			contact_json=COALESCE(NULLIF(?,''),contact_json),
			settle_method=COALESCE(NULLIF(?,''),settle_method),
			payment_days=COALESCE(?,payment_days),
			credit_limit=COALESCE(?,credit_limit),
			logistics_remark=COALESCE(NULLIF(?,''),logistics_remark),
			level=COALESCE(NULLIF(?,''),level),
			source=COALESCE(NULLIF(?,''),source),
			remark=COALESCE(NULLIF(?,''),remark),
			updated_at=NOW() WHERE id=?`,
			strOr(body["contact_name"]), strOr(body["mobile"]), strOr(body["address"]),
			contactJSONOr(body), strOr(body["settle_method"]),
			nullInt(body["payment_days"]), nullFloat(body["credit_limit"]),
			strOr(body["logistics_remark"]),
			levelOr(body), strOr(body["source"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadCustomerProfile(id))
		return true
	case action == "list":
		return s.listCRMCustomers(c)
	case action == "create":
		return s.createCRMCustomer(c)
	case action == "get":
		m := s.loadCustomer(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		return s.updateCRMCustomer(c)
	case action == "delete":
		_, _ = s.DB.Exec(`UPDATE crm_customer SET is_deleted=1, status='inactive', updated_at=NOW() WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	return false
}

func (s *Services) listCRMCustomers(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	kw := strings.TrimSpace(c.Query("keyword"))
	where := `WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	if c.Query("include_hidden") != "1" {
		where += ` AND COALESCE(is_hidden,0)=0`
	}
	if v := c.Query("is_public_sea"); v != "" {
		where += ` AND COALESCE(is_public_sea,0)=?`
		args = append(args, v)
	}
	if v := c.Query("is_locked"); v != "" {
		where += ` AND COALESCE(is_locked,0)=?`
		args = append(args, v)
	}
	if v := c.Query("owner_user_id"); v != "" {
		where += ` AND owner_user_id=?`
		args = append(args, v)
	}
	if v := c.Query("status"); v != "" {
		where += ` AND status=?`
		args = append(args, v)
	}
	if v := c.Query("level"); v != "" {
		where += ` AND level=?`
		args = append(args, v)
	}
	if kw != "" {
		where += ` AND (name LIKE ? OR code LIKE ? OR COALESCE(mobile,'') LIKE ? OR COALESCE(contact_name,'') LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_customer `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(short_name,''), COALESCE(contact_name,''), COALESCE(mobile,''),
		COALESCE(address,''), COALESCE(level,''), COALESCE(source,''), status, COALESCE(owner_user_id,0),
		COALESCE(protect_until,''), COALESCE(is_public_sea,0), COALESCE(is_hidden,0), COALESCE(is_locked,0),
		COALESCE(remark,''), created_at
		FROM crm_customer `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, owner int64
		var pub, hid, loc int
		var code, name, short, contact, mobile, addr, level, source, status, protect, remark, created string
		_ = rows.Scan(&id, &code, &name, &short, &contact, &mobile, &addr, &level, &source, &status, &owner,
			&protect, &pub, &hid, &loc, &remark, &created)
		list = append(list, gin.H{
			"id": id, "code": code, "name": name, "short_name": short, "contact_name": contact,
			"mobile": mobile, "address": addr, "level": level, "level_code": level, "source": source,
			"status": status, "owner_user_id": owner, "protect_until": protect,
			"is_public_sea": pub == 1, "is_hidden": hid == 1, "is_locked": loc == 1,
			"remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createCRMCustomer(c *gin.Context) bool {
	body := bindBody(c)
	name := strOr(body["name"])
	if name == "" {
		api.FailJSON(c, "NAME_REQUIRED")
		return true
	}
	code := strOrDef(body["code"], fmt.Sprintf("CU%s", time.Now().Format("060102150405")))
	owner := claimsUserID(c)
	if v, ok := asInt64(body["owner_user_id"]); ok && v > 0 {
		owner = v
	}
	pub := 0
	if b, ok := body["is_public_sea"].(bool); ok && b {
		pub = 1
	}
	var ownerArg interface{}
	if pub == 1 {
		ownerArg = nil
	} else if owner > 0 {
		ownerArg = owner
	} else {
		ownerArg = nil
	}
	protectDays := s.activeProtectDays()
	protectUntil := ""
	if pub == 0 && owner > 0 && protectDays > 0 {
		protectUntil = time.Now().AddDate(0, 0, protectDays).Format("2006-01-02")
	}
	res, err := s.DB.Exec(`INSERT INTO crm_customer(code, name, short_name, contact_name, mobile, address, level, source, status,
		owner_user_id, protect_until, is_public_sea, contact_json, settle_method, payment_days, credit_limit, logistics_remark, remark, created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		code, name, strOr(body["short_name"]), strOr(body["contact_name"]), strOr(body["mobile"]),
		strOr(body["address"]), levelOr(body), strOr(body["source"]), strOrDef(body["status"], "active"),
		ownerArg, nullStr(protectUntil), pub, contactJSONOr(body), strOr(body["settle_method"]),
		nullInt(body["payment_days"]), nullFloat(body["credit_limit"]), strOr(body["logistics_remark"]),
		strOr(body["remark"]), claimsUserID(c))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadCustomer(id))
	return true
}

func (s *Services) updateCRMCustomer(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	_, err := s.DB.Exec(`UPDATE crm_customer SET
		name=COALESCE(NULLIF(?,''),name), short_name=COALESCE(NULLIF(?,''),short_name),
		contact_name=COALESCE(NULLIF(?,''),contact_name), mobile=COALESCE(NULLIF(?,''),mobile),
		address=COALESCE(NULLIF(?,''),address), level=COALESCE(NULLIF(?,''),level),
		source=COALESCE(NULLIF(?,''),source), status=COALESCE(NULLIF(?,''),status),
		remark=COALESCE(NULLIF(?,''),remark), settle_method=COALESCE(NULLIF(?,''),settle_method),
		payment_days=COALESCE(?,payment_days), credit_limit=COALESCE(?,credit_limit),
		logistics_remark=COALESCE(NULLIF(?,''),logistics_remark),
		contact_json=COALESCE(NULLIF(?,''),contact_json),
		updated_at=NOW() WHERE id=?`,
		strOr(body["name"]), strOr(body["short_name"]), strOr(body["contact_name"]), strOr(body["mobile"]),
		strOr(body["address"]), levelOr(body), strOr(body["source"]), strOr(body["status"]),
		strOr(body["remark"]), strOr(body["settle_method"]), nullInt(body["payment_days"]),
		nullFloat(body["credit_limit"]), strOr(body["logistics_remark"]), contactJSONOr(body), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadCustomer(id))
	return true
}

func (s *Services) loadCustomer(id int64) gin.H {
	var code, name, short, contact, mobile, addr, level, source, status, protect, remark, created string
	var owner int64
	var pub, hid, loc int
	err := s.DB.QueryRow(`SELECT code, name, COALESCE(short_name,''), COALESCE(contact_name,''), COALESCE(mobile,''),
		COALESCE(address,''), COALESCE(level,''), COALESCE(source,''), status, COALESCE(owner_user_id,0),
		COALESCE(protect_until,''), COALESCE(is_public_sea,0), COALESCE(is_hidden,0), COALESCE(is_locked,0),
		COALESCE(remark,''), created_at FROM crm_customer WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&code, &name, &short, &contact, &mobile, &addr, &level, &source, &status, &owner, &protect, &pub, &hid, &loc, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "code": code, "name": name, "short_name": short, "contact_name": contact,
		"mobile": mobile, "address": addr, "level": level, "level_code": level, "source": source,
		"status": status, "owner_user_id": owner, "protect_until": protect,
		"is_public_sea": pub == 1, "is_hidden": hid == 1, "is_locked": loc == 1,
		"remark": remark, "created_at": created,
	}
}

func (s *Services) loadCustomerProfile(id int64) gin.H {
	base := s.loadCustomer(id)
	if base["id"] == nil {
		return gin.H{}
	}
	var contactJSON, settle, logistics string
	var payDays int64
	var credit float64
	_ = s.DB.QueryRow(`SELECT COALESCE(contact_json,''), COALESCE(settle_method,''), COALESCE(payment_days,0),
		COALESCE(credit_limit,0), COALESCE(logistics_remark,'') FROM crm_customer WHERE id=?`, id).
		Scan(&contactJSON, &settle, &payDays, &credit, &logistics)
	base["contact_json"] = contactJSON
	base["settle_method"] = settle
	base["payment_days"] = payDays
	base["credit_limit"] = credit
	base["logistics_remark"] = logistics
	if contactJSON != "" {
		var parsed interface{}
		if json.Unmarshal([]byte(contactJSON), &parsed) == nil {
			base["contacts"] = parsed
		}
	}
	// 360: recent follow-ups / opportunities / orders / quotes
	base["follow_ups"] = s.listFollowUpsByCustomer(id, 10)
	base["opportunities"] = s.listOppsByCustomer(id, 10)
	base["orders"] = s.listOrdersByCustomer(id, 10)
	base["quote_histories"] = s.listQuotesByCustomer(id, 10)
	return base
}

func (s *Services) listFollowUpsByCustomer(customerID int64, limit int) []gin.H {
	rows, err := s.DB.Query(`SELECT id, customer_id, COALESCE(opportunity_id,0), user_id, COALESCE(follow_type,''), follow_at,
		COALESCE(content,''), COALESCE(next_remind_at,''), created_at FROM crm_follow_up WHERE customer_id=? ORDER BY id DESC LIMIT ?`,
		customerID, limit)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, cid, oid, uid int64
		var ftype, fat, content, next, created string
		_ = rows.Scan(&id, &cid, &oid, &uid, &ftype, &fat, &content, &next, &created)
		out = append(out, gin.H{
			"id": id, "customer_id": cid, "opportunity_id": oid, "user_id": uid,
			"follow_type": ftype, "follow_at": fat, "content": content, "next_remind_at": next, "created_at": created,
		})
	}
	return out
}

func (s *Services) listOppsByCustomer(customerID int64, limit int) []gin.H {
	rows, err := s.DB.Query(`SELECT id, customer_id, COALESCE(title,''), stage, amount, COALESCE(expected_date,''),
		COALESCE(owner_user_id,0), status, COALESCE(remark,''), COALESCE(converted_order_id,0), created_at
		FROM crm_opportunity WHERE customer_id=? ORDER BY id DESC LIMIT ?`, customerID, limit)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, cid, owner, orderID int64
		var title, stage, expected, status, remark, created string
		var amount float64
		_ = rows.Scan(&id, &cid, &title, &stage, &amount, &expected, &owner, &status, &remark, &orderID, &created)
		out = append(out, gin.H{
			"id": id, "customer_id": cid, "title": title, "stage": stage, "amount": amount,
			"expected_date": expected, "owner_user_id": owner, "status": status, "remark": remark,
			"converted_order_id": orderID, "created_at": created,
		})
	}
	return out
}

func (s *Services) listOrdersByCustomer(customerID int64, limit int) []gin.H {
	rows, err := s.DB.Query(`SELECT id, doc_no, status, total_amount, created_at FROM sl_sales_order
		WHERE customer_id=? AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT ?`, customerID, limit)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var docNo, status, created string
		var amount float64
		_ = rows.Scan(&id, &docNo, &status, &amount, &created)
		out = append(out, gin.H{"id": id, "doc_no": docNo, "status": status, "total_amount": amount, "created_at": created})
	}
	return out
}

func (s *Services) listQuotesByCustomer(customerID int64, limit int) []gin.H {
	rows, err := s.DB.Query(`SELECT id, product_id, price, quoted_at, COALESCE(inquiry_id,0), COALESCE(order_id,0)
		FROM sl_quote_history WHERE customer_id=? ORDER BY id DESC LIMIT ?`, customerID, limit)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, pid, inq, oid int64
		var price float64
		var quoted string
		_ = rows.Scan(&id, &pid, &price, &quoted, &inq, &oid)
		out = append(out, gin.H{"id": id, "product_id": pid, "price": price, "quoted_at": quoted, "inquiry_id": inq, "order_id": oid})
	}
	return out
}

// ---------- opportunities ----------

func (s *Services) handleCRMOpportunities(c *gin.Context, method, path, action string) bool {
	if strings.Contains(path, "/convert") || action == "action:convert" {
		return s.convertOpportunity(c)
	}
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		where := `WHERE 1=1`
		args := []interface{}{}
		if v := c.Query("customer_id"); v != "" {
			where += ` AND customer_id=?`
			args = append(args, v)
		}
		if v := c.Query("status"); v != "" {
			where += ` AND status=?`
			args = append(args, v)
		}
		if v := c.Query("stage"); v != "" {
			where += ` AND stage=?`
			args = append(args, v)
		}
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_opportunity `+where, args...).Scan(&total)
		args = append(args, pageSize, (pageNum-1)*pageSize)
		rows, err := s.DB.Query(`SELECT o.id, o.customer_id, COALESCE(c.name,''), COALESCE(o.title,''), o.stage, o.amount,
			COALESCE(o.expected_date,''), COALESCE(o.owner_user_id,0), o.status, COALESCE(o.remark,''),
			COALESCE(o.converted_order_id,0), o.created_at
			FROM crm_opportunity o LEFT JOIN crm_customer c ON c.id=o.customer_id `+where+`
			ORDER BY o.id DESC LIMIT ? OFFSET ?`, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, cid, owner, orderID int64
			var cname, title, stage, expected, status, remark, created string
			var amount float64
			_ = rows.Scan(&id, &cid, &cname, &title, &stage, &amount, &expected, &owner, &status, &remark, &orderID, &created)
			list = append(list, gin.H{
				"id": id, "customer_id": cid, "customer_name": cname, "title": title, "stage": stage,
				"amount": amount, "expected_date": expected, "owner_user_id": owner, "status": status,
				"remark": remark, "converted_order_id": orderID, "created_at": created,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		cid, ok := asInt64(body["customer_id"])
		if !ok || cid == 0 {
			api.FailJSON(c, "CUSTOMER_REQUIRED")
			return true
		}
		if s.loadCustomer(cid)["id"] == nil {
			api.FailJSON(c, "CUSTOMER_NOT_FOUND")
			return true
		}
		owner := claimsUserID(c)
		if v, ok := asInt64(body["owner_user_id"]); ok && v > 0 {
			owner = v
		}
		amount, _ := asFloat(body["amount"])
		res, err := s.DB.Exec(`INSERT INTO crm_opportunity(customer_id, title, stage, amount, expected_date, owner_user_id, status, remark)
			VALUES(?,?,?,?,?,?,?,?)`,
			cid, strOrDef(body["title"], "商机"), strOrDef(body["stage"], "lead"), amount,
			strOr(body["expected_date"]), owner, strOrDef(body["status"], "open"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, s.loadOpportunity(id))
		return true
	case "get":
		m := s.loadOpportunity(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE crm_opportunity SET
			title=COALESCE(NULLIF(?,''),title), stage=COALESCE(NULLIF(?,''),stage),
			amount=COALESCE(?,amount), expected_date=COALESCE(NULLIF(?,''),expected_date),
			status=COALESCE(NULLIF(?,''),status), remark=COALESCE(NULLIF(?,''),remark),
			updated_at=NOW() WHERE id=?`,
			strOr(body["title"]), strOr(body["stage"]), nullFloat(body["amount"]),
			strOr(body["expected_date"]), strOr(body["status"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadOpportunity(id))
		return true
	}
	_ = method
	return false
}

func (s *Services) loadOpportunity(id int64) gin.H {
	var cid, owner, orderID int64
	var title, stage, expected, status, remark, created string
	var amount float64
	err := s.DB.QueryRow(`SELECT customer_id, COALESCE(title,''), stage, amount, COALESCE(expected_date,''),
		COALESCE(owner_user_id,0), status, COALESCE(remark,''), COALESCE(converted_order_id,0), created_at
		FROM crm_opportunity WHERE id=?`, id).
		Scan(&cid, &title, &stage, &amount, &expected, &owner, &status, &remark, &orderID, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "customer_id": cid, "title": title, "stage": stage, "amount": amount,
		"expected_date": expected, "owner_user_id": owner, "status": status, "remark": remark,
		"converted_order_id": orderID, "created_at": created,
	}
}

func (s *Services) convertOpportunity(c *gin.Context) bool {
	id := paramID(c)
	opp := s.loadOpportunity(id)
	if opp["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(opp["status"]) == "won" && asInt64Or0(opp["converted_order_id"]) > 0 {
		api.OK(c, opp)
		return true
	}
	body := bindBody(c)
	cid := asInt64Or0(opp["customer_id"])
	productID := asInt64Or0(body["product_id"])
	if productID == 0 {
		productID = 3
	}
	qty, _ := asFloat(body["qty"])
	if qty <= 0 {
		qty = 100
	}
	price, ok := asFloat(body["price"])
	if !ok || price <= 0 {
		if lp, _, found := s.resolveLockPrice(cid, productID); found {
			price = lp
		} else {
			price = s.productSalePrice(productID)
		}
	}
	docNo := fmt.Sprintf("SO%s", time.Now().Format("060102150405"))
	amount := qty * price
	res, err := s.DB.Exec(`INSERT INTO sl_sales_order(doc_no, customer_id, owner_user_id, status, source, warehouse_id, total_amount, remark, created_by)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		docNo, cid, asInt64Or0(opp["owner_user_id"]), "draft", "opportunity", 3, amount,
		fmt.Sprintf("商机转化#%d %s", id, strOr(opp["title"])), claimsUserID(c))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	orderID, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO sl_sales_order_line(order_id, product_id, qty, price, amount) VALUES(?,?,?,?,?)`,
		orderID, productID, qty, price, amount)
	_, _ = s.DB.Exec(`UPDATE crm_opportunity SET status='won', stage='won', converted_order_id=?, updated_at=NOW() WHERE id=?`,
		orderID, id)
	out := s.loadOpportunity(id)
	out["order"] = gin.H{"id": orderID, "doc_no": docNo, "total_amount": amount}
	api.OK(c, out)
	return true
}

// ---------- follow-ups ----------

func (s *Services) handleCRMFollowUps(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		where := `WHERE 1=1`
		args := []interface{}{}
		if v := c.Query("customer_id"); v != "" {
			where += ` AND f.customer_id=?`
			args = append(args, v)
		}
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_follow_up f `+where, args...).Scan(&total)
		args = append(args, pageSize, (pageNum-1)*pageSize)
		rows, err := s.DB.Query(`SELECT f.id, f.customer_id, COALESCE(c.name,''), COALESCE(f.opportunity_id,0), f.user_id,
			COALESCE(f.follow_type,''), f.follow_at, COALESCE(f.content,''), COALESCE(f.next_remind_at,''), f.created_at
			FROM crm_follow_up f LEFT JOIN crm_customer c ON c.id=f.customer_id `+where+`
			ORDER BY f.id DESC LIMIT ? OFFSET ?`, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, cid, oid, uid int64
			var cname, ftype, fat, content, next, created string
			_ = rows.Scan(&id, &cid, &cname, &oid, &uid, &ftype, &fat, &content, &next, &created)
			list = append(list, gin.H{
				"id": id, "customer_id": cid, "customer_name": cname, "opportunity_id": oid, "user_id": uid,
				"follow_type": ftype, "follow_at": fat, "content": content, "next_remind_at": next, "created_at": created,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		cid, ok := asInt64(body["customer_id"])
		if !ok || cid == 0 {
			api.FailJSON(c, "CUSTOMER_REQUIRED")
			return true
		}
		uid := claimsUserID(c)
		if v, ok := asInt64(body["user_id"]); ok && v > 0 {
			uid = v
		}
		followAt := strOrDef(body["follow_at"], time.Now().Format("2006-01-02 15:04:05"))
		next := strOr(body["next_remind_at"])
		res, err := s.DB.Exec(`INSERT INTO crm_follow_up(customer_id, opportunity_id, user_id, follow_type, follow_at, content, next_remind_at)
			VALUES(?,?,?,?,?,?,?)`,
			cid, nullInt(body["opportunity_id"]), uid, strOrDef(body["follow_type"], "visit"),
			followAt, strOr(body["content"]), nullStr(next))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		fid, _ := res.LastInsertId()
		// extend protect when followed
		if days := s.activeProtectDays(); days > 0 {
			until := time.Now().AddDate(0, 0, days).Format("2006-01-02")
			_, _ = s.DB.Exec(`UPDATE crm_customer SET protect_until=?, is_public_sea=0, updated_at=NOW()
				WHERE id=? AND COALESCE(is_deleted,0)=0`, until, cid)
		}
		if next != "" {
			_, _ = s.DB.Exec(`INSERT INTO crm_task_reminder(user_id, ref_type, ref_id, remind_at, content, status)
				VALUES(?,?,?,?,?,?)`, uid, "follow_up", fid, next, strOrDef(body["content"], "跟进提醒"), "pending")
		}
		api.OK(c, s.loadFollowUp(fid))
		return true
	case "get":
		m := s.loadFollowUp(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE crm_follow_up SET
			follow_type=COALESCE(NULLIF(?,''),follow_type), follow_at=COALESCE(NULLIF(?,''),follow_at),
			content=COALESCE(NULLIF(?,''),content), next_remind_at=COALESCE(NULLIF(?,''),next_remind_at)
			WHERE id=?`,
			strOr(body["follow_type"]), strOr(body["follow_at"]), strOr(body["content"]), strOr(body["next_remind_at"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadFollowUp(id))
		return true
	}
	_ = method
	return false
}

func (s *Services) loadFollowUp(id int64) gin.H {
	var cid, oid, uid int64
	var ftype, fat, content, next, created string
	err := s.DB.QueryRow(`SELECT customer_id, COALESCE(opportunity_id,0), user_id, COALESCE(follow_type,''), follow_at,
		COALESCE(content,''), COALESCE(next_remind_at,''), created_at FROM crm_follow_up WHERE id=?`, id).
		Scan(&cid, &oid, &uid, &ftype, &fat, &content, &next, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "customer_id": cid, "opportunity_id": oid, "user_id": uid,
		"follow_type": ftype, "follow_at": fat, "content": content, "next_remind_at": next, "created_at": created,
	}
}

// ---------- lead assigns ----------

func (s *Services) handleCRMLeadAssigns(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_lead_assign`).Scan(&total)
		rows, err := s.DB.Query(`SELECT a.id, a.customer_id, COALESCE(c.name,''), COALESCE(a.from_user_id,0), a.to_user_id,
			a.assigned_at, a.lock_flag, COALESCE(a.remark,'')
			FROM crm_lead_assign a LEFT JOIN crm_customer c ON c.id=a.customer_id
			ORDER BY a.id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, cid, from, to int64
			var lock int
			var cname, at, remark string
			_ = rows.Scan(&id, &cid, &cname, &from, &to, &at, &lock, &remark)
			list = append(list, gin.H{
				"id": id, "customer_id": cid, "customer_name": cname, "from_user_id": from, "to_user_id": to,
				"assigned_at": at, "lock_flag": lock == 1, "remark": remark,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		cid, ok := asInt64(body["customer_id"])
		if !ok || cid == 0 {
			api.FailJSON(c, "CUSTOMER_REQUIRED")
			return true
		}
		toUID, ok := asInt64(body["to_user_id"])
		if !ok || toUID == 0 {
			api.FailJSON(c, "TO_USER_REQUIRED")
			return true
		}
		cust := s.loadCustomer(cid)
		if cust["id"] == nil {
			api.FailJSON(c, "CUSTOMER_NOT_FOUND")
			return true
		}
		if b, _ := cust["is_locked"].(bool); b {
			api.FailJSON(c, "CUSTOMER_LOCKED")
			return true
		}
		fromUID := asInt64Or0(cust["owner_user_id"])
		lockFlag := 0
		if b, ok := body["lock_flag"].(bool); ok && b {
			lockFlag = 1
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		protectUntil := time.Now().AddDate(0, 0, s.activeProtectDays()).Format("2006-01-02")
		_, err := s.DB.Exec(`INSERT INTO crm_lead_assign(customer_id, from_user_id, to_user_id, assigned_at, lock_flag, remark)
			VALUES(?,?,?,?,?,?)`, cid, nullInt(fromUID), toUID, now, lockFlag, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		_, _ = s.DB.Exec(`UPDATE crm_customer SET owner_user_id=?, is_public_sea=0, protect_until=?, is_locked=?, updated_at=NOW() WHERE id=?`,
			toUID, protectUntil, lockFlag, cid)
		api.OK(c, gin.H{
			"customer_id": cid, "from_user_id": fromUID, "to_user_id": toUID,
			"assigned_at": now, "protect_until": protectUntil, "lock_flag": lockFlag == 1,
		})
		return true
	}
	_ = method
	return false
}

// ---------- protect rules ----------

func (s *Services) handleCRMProtectRules(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_lead_protect_rule`).Scan(&total)
		rows, err := s.DB.Query(`SELECT id, name, protect_days, COALESCE(release_rule_json,''), status, created_at
			FROM crm_lead_protect_rule ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var days int
			var name, rule, status, created string
			_ = rows.Scan(&id, &name, &days, &rule, &status, &created)
			list = append(list, gin.H{"id": id, "name": name, "protect_days": days, "release_rule_json": rule, "status": status, "created_at": created})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		days := 30
		if v, ok := asInt64(body["protect_days"]); ok && v > 0 {
			days = int(v)
		}
		res, err := s.DB.Exec(`INSERT INTO crm_lead_protect_rule(name, protect_days, release_rule_json, status) VALUES(?,?,?,?)`,
			name, days, strOrDef(body["release_rule_json"], `{"auto_release":true}`), strOrDef(body["status"], "active"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, s.loadProtectRule(id))
		return true
	case "get":
		m := s.loadProtectRule(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE crm_lead_protect_rule SET
			name=COALESCE(NULLIF(?,''),name), protect_days=COALESCE(?,protect_days),
			release_rule_json=COALESCE(NULLIF(?,''),release_rule_json),
			status=COALESCE(NULLIF(?,''),status), updated_at=NOW() WHERE id=?`,
			strOr(body["name"]), nullInt(body["protect_days"]), strOr(body["release_rule_json"]), strOr(body["status"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadProtectRule(id))
		return true
	case "delete":
		_, _ = s.DB.Exec(`DELETE FROM crm_lead_protect_rule WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	_ = method
	return false
}

func (s *Services) loadProtectRule(id int64) gin.H {
	var days int
	var name, rule, status, created string
	err := s.DB.QueryRow(`SELECT name, protect_days, COALESCE(release_rule_json,''), status, created_at FROM crm_lead_protect_rule WHERE id=?`, id).
		Scan(&name, &days, &rule, &status, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{"id": id, "name": name, "protect_days": days, "release_rule_json": rule, "status": status, "created_at": created}
}

func (s *Services) activeProtectDays() int {
	var days int
	err := s.DB.QueryRow(`SELECT protect_days FROM crm_lead_protect_rule WHERE status='active' ORDER BY id DESC LIMIT 1`).Scan(&days)
	if err != nil || days <= 0 {
		return 30
	}
	return days
}

// ---------- releases ----------

func (s *Services) handleCRMReleases(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_lead_release_log`).Scan(&total)
		rows, err := s.DB.Query(`SELECT r.id, r.customer_id, COALESCE(c.name,''), r.released_at, COALESCE(r.reason,''),
			r.to_public_sea, COALESCE(r.from_user_id,0), COALESCE(r.operator_user_id,0)
			FROM crm_lead_release_log r LEFT JOIN crm_customer c ON c.id=r.customer_id
			ORDER BY r.id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, cid, from, op int64
			var pub int
			var cname, at, reason string
			_ = rows.Scan(&id, &cid, &cname, &at, &reason, &pub, &from, &op)
			list = append(list, gin.H{
				"id": id, "customer_id": cid, "customer_name": cname, "released_at": at, "reason": reason,
				"to_public_sea": pub == 1, "from_user_id": from, "operator_user_id": op,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		// batch auto-release expired protections when no customer_id
		if body["customer_id"] == nil && (body["auto"] == true || strOr(body["mode"]) == "auto") {
			return s.autoReleaseExpired(c)
		}
		cid, ok := asInt64(body["customer_id"])
		if !ok || cid == 0 {
			api.FailJSON(c, "CUSTOMER_REQUIRED")
			return true
		}
		cust := s.loadCustomer(cid)
		if cust["id"] == nil {
			api.FailJSON(c, "CUSTOMER_NOT_FOUND")
			return true
		}
		fromUID := asInt64Or0(cust["owner_user_id"])
		now := time.Now().Format("2006-01-02 15:04:05")
		_, err := s.DB.Exec(`INSERT INTO crm_lead_release_log(customer_id, released_at, reason, to_public_sea, from_user_id, operator_user_id)
			VALUES(?,?,?,1,?,?)`, cid, now, strOrDef(body["reason"], "手动释放公海"), fromUID, claimsUserID(c))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		_, _ = s.DB.Exec(`UPDATE crm_customer SET owner_user_id=NULL, is_public_sea=1, protect_until=NULL, is_locked=0, updated_at=NOW() WHERE id=?`, cid)
		api.OK(c, gin.H{"customer_id": cid, "released_at": now, "to_public_sea": true})
		return true
	}
	_ = method
	return false
}

func (s *Services) autoReleaseExpired(c *gin.Context) bool {
	today := time.Now().Format("2006-01-02")
	rows, err := s.DB.Query(`SELECT id, COALESCE(owner_user_id,0) FROM crm_customer
		WHERE COALESCE(is_deleted,0)=0 AND COALESCE(is_public_sea,0)=0
		AND protect_until IS NOT NULL AND protect_until<>'' AND protect_until<?`, today)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	type item struct{ id, owner int64 }
	items := []item{}
	for rows.Next() {
		var id, owner int64
		_ = rows.Scan(&id, &owner)
		items = append(items, item{id, owner})
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	n := 0
	for _, it := range items {
		_, _ = s.DB.Exec(`INSERT INTO crm_lead_release_log(customer_id, released_at, reason, to_public_sea, from_user_id, operator_user_id)
			VALUES(?,?,?,1,?,?)`, it.id, now, "保护期到期自动释放", it.owner, claimsUserID(c))
		_, _ = s.DB.Exec(`UPDATE crm_customer SET owner_user_id=NULL, is_public_sea=1, protect_until=NULL, is_locked=0, updated_at=NOW() WHERE id=?`, it.id)
		n++
	}
	api.OK(c, gin.H{"released_count": n, "released_at": now})
	return true
}

// ---------- imports ----------

func (s *Services) handleCRMImports(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_customer_import_batch`).Scan(&total)
		rows, err := s.DB.Query(`SELECT id, COALESCE(file_name,''), imported_at, success_count, fail_count, COALESCE(fail_detail_json,''), COALESCE(created_by,0)
			FROM crm_customer_import_batch ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, by int64
			var okN, failN int
			var fname, at, detail string
			_ = rows.Scan(&id, &fname, &at, &okN, &failN, &detail, &by)
			list = append(list, gin.H{
				"id": id, "file_name": fname, "imported_at": at, "success_count": okN, "fail_count": failN,
				"fail_detail_json": detail, "created_by": by,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		rowsRaw, _ := body["rows"].([]interface{})
		if len(rowsRaw) == 0 {
			if items, ok := body["items"].([]interface{}); ok {
				rowsRaw = items
			}
		}
		if len(rowsRaw) == 0 {
			api.FailJSON(c, "ROWS_REQUIRED")
			return true
		}
		success, fail := 0, 0
		failDetails := []gin.H{}
		for i, raw := range rowsRaw {
			m, ok := raw.(map[string]interface{})
			if !ok {
				fail++
				failDetails = append(failDetails, gin.H{"row": i + 1, "error": "INVALID_ROW"})
				continue
			}
			name := strOr(m["name"])
			if name == "" {
				fail++
				failDetails = append(failDetails, gin.H{"row": i + 1, "error": "NAME_REQUIRED"})
				continue
			}
			code := strOrDef(m["code"], fmt.Sprintf("CU%s%02d", time.Now().Format("060102150405"), i+1))
			var exists int
			_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_customer WHERE code=? OR (name=? AND COALESCE(is_deleted,0)=0)`, code, name).Scan(&exists)
			if exists > 0 {
				fail++
				failDetails = append(failDetails, gin.H{"row": i + 1, "code": code, "name": name, "error": "DUPLICATE"})
				continue
			}
			_, err := s.DB.Exec(`INSERT INTO crm_customer(code, name, short_name, contact_name, mobile, address, level, source, status, is_public_sea, remark, created_by)
				VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
				code, name, strOr(m["short_name"]), strOr(m["contact_name"]), strOr(m["mobile"]),
				strOr(m["address"]), levelOr(m), strOrDef(m["source"], "导入"), strOrDef(m["status"], "active"),
				1, strOr(m["remark"]), claimsUserID(c))
			if err != nil {
				fail++
				failDetails = append(failDetails, gin.H{"row": i + 1, "error": err.Error()})
				continue
			}
			success++
		}
		detailBytes, _ := json.Marshal(failDetails)
		now := time.Now().Format("2006-01-02 15:04:05")
		res, err := s.DB.Exec(`INSERT INTO crm_customer_import_batch(file_name, imported_at, success_count, fail_count, fail_detail_json, created_by)
			VALUES(?,?,?,?,?,?)`,
			strOrDef(body["file_name"], "paste-import.json"), now, success, fail, string(detailBytes), claimsUserID(c))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{
			"id": id, "file_name": strOrDef(body["file_name"], "paste-import.json"), "imported_at": now,
			"success_count": success, "fail_count": fail, "fail_details": failDetails,
		})
		return true
	case "get":
		id := paramID(c)
		var by int64
		var okN, failN int
		var fname, at, detail string
		err := s.DB.QueryRow(`SELECT COALESCE(file_name,''), imported_at, success_count, fail_count, COALESCE(fail_detail_json,''), COALESCE(created_by,0)
			FROM crm_customer_import_batch WHERE id=?`, id).Scan(&fname, &at, &okN, &failN, &detail, &by)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{
			"id": id, "file_name": fname, "imported_at": at, "success_count": okN, "fail_count": failN,
			"fail_detail_json": detail, "created_by": by,
		})
		return true
	}
	_ = method
	return false
}

// ---------- task reminders ----------

func (s *Services) handleCRMTaskReminders(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		pageNum, pageSize := sqlutil.Page(c)
		where := `WHERE 1=1`
		args := []interface{}{}
		if v := c.Query("status"); v != "" {
			where += ` AND status=?`
			args = append(args, v)
		}
		if v := c.Query("user_id"); v != "" {
			where += ` AND user_id=?`
			args = append(args, v)
		}
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_task_reminder `+where, args...).Scan(&total)
		args = append(args, pageSize, (pageNum-1)*pageSize)
		rows, err := s.DB.Query(`SELECT id, user_id, COALESCE(ref_type,''), COALESCE(ref_id,0), remind_at, COALESCE(content,''), status, created_at
			FROM crm_task_reminder `+where+` ORDER BY remind_at ASC LIMIT ? OFFSET ?`, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, uid, refID int64
			var refType, remind, content, status, created string
			_ = rows.Scan(&id, &uid, &refType, &refID, &remind, &content, &status, &created)
			list = append(list, gin.H{
				"id": id, "user_id": uid, "ref_type": refType, "ref_id": refID,
				"remind_at": remind, "content": content, "status": status, "created_at": created,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	case "create":
		body := bindBody(c)
		uid := claimsUserID(c)
		if v, ok := asInt64(body["user_id"]); ok && v > 0 {
			uid = v
		}
		remind := strOr(body["remind_at"])
		if remind == "" {
			api.FailJSON(c, "REMIND_AT_REQUIRED")
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO crm_task_reminder(user_id, ref_type, ref_id, remind_at, content, status) VALUES(?,?,?,?,?,?)`,
			uid, strOrDef(body["ref_type"], "customer"), nullInt(body["ref_id"]), remind,
			strOr(body["content"]), strOrDef(body["status"], "pending"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, s.loadTaskReminder(id))
		return true
	case "get":
		m := s.loadTaskReminder(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE crm_task_reminder SET
			remind_at=COALESCE(NULLIF(?,''),remind_at), content=COALESCE(NULLIF(?,''),content),
			status=COALESCE(NULLIF(?,''),status), updated_at=NOW() WHERE id=?`,
			strOr(body["remind_at"]), strOr(body["content"]), strOr(body["status"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadTaskReminder(id))
		return true
	case "delete":
		_, _ = s.DB.Exec(`DELETE FROM crm_task_reminder WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	_ = method
	return false
}

func (s *Services) loadTaskReminder(id int64) gin.H {
	var uid, refID int64
	var refType, remind, content, status, created string
	err := s.DB.QueryRow(`SELECT user_id, COALESCE(ref_type,''), COALESCE(ref_id,0), remind_at, COALESCE(content,''), status, created_at
		FROM crm_task_reminder WHERE id=?`, id).Scan(&uid, &refType, &refID, &remind, &content, &status, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "user_id": uid, "ref_type": refType, "ref_id": refID,
		"remind_at": remind, "content": content, "status": status, "created_at": created,
	}
}

// ---------- CRM side inquiries (from sales inquiries) ----------

func (s *Services) handleCRMInquiries(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE COALESCE(i.is_deleted,0)=0`
	args := []interface{}{}
	if v := c.Query("customer_id"); v != "" {
		where += ` AND i.customer_id=?`
		args = append(args, v)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sl_inquiry i `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT i.id, i.doc_no, i.customer_id, COALESCE(c.name,''), i.status, i.source,
		COALESCE(i.expire_at,''), i.created_at
		FROM sl_inquiry i LEFT JOIN crm_customer c ON c.id=i.customer_id `+where+`
		ORDER BY i.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, cid int64
		var docNo, cname, status, source, expire, created string
		_ = rows.Scan(&id, &docNo, &cid, &cname, &status, &source, &expire, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "customer_id": cid, "customer_name": cname,
			"status": status, "source": source, "expire_at": expire, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func levelOr(body map[string]interface{}) string {
	if v := strOr(body["level"]); v != "" {
		return v
	}
	return strOrDef(body["level_code"], "B")
}

func contactJSONOr(body map[string]interface{}) string {
	if v := strOr(body["contact_json"]); v != "" {
		return v
	}
	if raw, ok := body["contacts"]; ok && raw != nil {
		b, err := json.Marshal(raw)
		if err == nil {
			return string(b)
		}
	}
	return ""
}
