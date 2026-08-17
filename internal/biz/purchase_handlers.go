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

func (s *Services) handlePurchase(c *gin.Context, method, openapiPath, resourceKey, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/certificate-alerts"):
		return s.handleCertificateAlerts(c)
	case strings.Contains(openapiPath, "/purchase/suppliers") && strings.Contains(openapiPath, "/licenses"):
		return s.handleSupplierLicenses(c, method)
	case strings.Contains(openapiPath, "/purchase/suppliers") && strings.Contains(openapiPath, "/supply-items"):
		return s.handleSupplierSupplyItems(c, method)
	case strings.Contains(openapiPath, "/purchase/suppliers") && strings.Contains(openapiPath, "/performance"):
		return s.handleSupplierPerformanceOne(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/suppliers"):
		return s.handleSuppliers(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/inbound-arrivals"):
		return s.handleInboundArrivals(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/farmers"):
		return s.handleFarmers(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/weigh-varieties"):
		return s.handleWeighVarieties(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/trace-batch-codes"):
		return s.handleTraceBatchCodes(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/weigh-flow"):
		return s.handleWeighFlowNextOptions(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/role-users"):
		return s.handlePurchaseRoleUsers(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/weigh-tickets"):
		return s.handleWeighTickets(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/farmer-settlements"):
		return s.handleFarmerSettlements(c, method, action)
	case openapiPath == "/api/v1/purchase/trace-lots/verify" || strings.HasSuffix(openapiPath, "/trace-lots/verify"):
		return s.verifyTraceLot(c)
	case strings.Contains(openapiPath, "/purchase/trace-lots/"):
		return s.handleTraceLot(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/inbounds"):
		return s.handlePurchaseInbounds(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/incoming-qcs"):
		return s.handleIncomingQCs(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/returns"):
		return s.handlePurchaseReturns(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/price-histories"):
		return s.handlePriceHistories(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/analytics/volume-price"):
		return s.handleVolumePrice(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/analytics/supplier-performance"):
		return s.handleSupplierPerformanceList(c)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/requests"):
		if strings.Contains(openapiPath, "/to-plan") {
			return s.convertRequestToPlan(c)
		}
		if strings.Contains(openapiPath, "/approve") {
			id := paramID(c)
			_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET status='approved', updated_at=NOW() WHERE id=? AND status='submitted'`, id)
			api.OK(c, s.loadPurchaseRequest(id))
			return true
		}
		if strings.Contains(openapiPath, "/reject") {
			body := bindBody(c)
			id := paramID(c)
			_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET status='rejected', remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=? AND status='submitted'`,
				strOr(body["reason"]), id)
			api.OK(c, s.loadPurchaseRequest(id))
			return true
		}
		return s.handlePurchaseRequests(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/plans"):
		if strings.Contains(openapiPath, "/to-inbound") {
			return s.convertPlanToInbound(c)
		}
		if strings.Contains(openapiPath, "/approve") {
			id := paramID(c)
			_, _ = s.DB.Exec(`UPDATE pur_purchase_plan SET status='approved', updated_at=NOW() WHERE id=? AND status='submitted'`, id)
			api.OK(c, s.loadPurchasePlan(id))
			return true
		}
		return s.handlePurchasePlans(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/purchase/tasks"):
		return s.handlePurchaseTasks(c, method, action)
	default:
		return s.handleTableCRUD(c, resourceKey, action)
	}
}

func (s *Services) handleFinancePartyGuard(c *gin.Context, method, openapiPath, action string) bool {
	if method != "POST" || action != "create" {
		return false
	}
	if !(strings.HasPrefix(openapiPath, "/api/v1/finance/prepay-prepaids") || strings.HasPrefix(openapiPath, "/api/v1/finance/arap-adjusts")) {
		return false
	}
	body := bindBody(c)
	pt, _ := body["party_type"].(string)
	if pt != "supplier" {
		return false
	}
	pid, _ := asInt64(body["party_id"])
	if pid <= 0 || !s.supplierExists(pid) {
		api.FailJSON(c, "SUPPLIER_NOT_FOUND")
		return true
	}
	return false
}

// ---------- suppliers master ----------

func (s *Services) handleSuppliers(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action == "list"):
		return s.listSuppliers(c)
	case action == "create":
		return s.createSupplier(c)
	case action == "get":
		return s.getSupplier(c)
	case action == "update" || action == "replace":
		return s.updateSupplier(c)
	case action == "delete":
		return s.deleteSupplier(c)
	case strings.HasPrefix(action, "action:"):
		return s.supplierStatusAction(c, strings.TrimPrefix(action, "action:"))
	default:
		return true
	}
}

func (s *Services) listSuppliers(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := "COALESCE(is_deleted,0)=0"
	args := []interface{}{}
	if st := c.Query("status"); st != "" {
		where += " AND status=?"
		args = append(args, st)
	}
	if tp := c.Query("supplier_type"); tp != "" {
		where += " AND supplier_type=?"
		args = append(args, tp)
	}
	if q := c.Query("q"); q != "" {
		where += " AND (code LIKE ? OR name LIKE ? OR COALESCE(short_name,'') LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_supplier WHERE `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT * FROM pur_supplier WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	for _, m := range list {
		decodeJSONField(m, "contact_json")
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createSupplier(c *gin.Context) bool {
	body := bindBody(c)
	code, _ := body["code"].(string)
	name, _ := body["name"].(string)
	if code == "" || name == "" {
		api.FailJSON(c, "CODE_NAME_REQUIRED")
		return true
	}
	status, _ := body["status"].(string)
	if status == "" {
		status = "potential"
	}
	if status == "active" {
		status = "qualified"
	}
	if !validSupplierStatus(status) {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	pref := boolInt(body["is_preferred"])
	if pref == 1 && status != "qualified" {
		api.FailJSON(c, "PREFERRED_REQUIRES_QUALIFIED")
		return true
	}
	contact := jsonify(body["contact_json"])
	res, err := s.DB.Exec(`INSERT INTO pur_supplier(
		code,name,short_name,mnemonic,supplier_type,status,rating,is_preferred,uscc,legal_person,register_address,
		invoice_title,tax_no,bank_name,bank_account,settle_method,payment_days,credit_limit,currency,tax_rate,
		lead_time_days,moq,default_warehouse_id,contact_json,remark)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		code, name, strOr(body["short_name"]), strOr(body["mnemonic"]), strOrDef(body["supplier_type"], "raw"),
		status, strOr(body["rating"]), pref, strOr(body["uscc"]), strOr(body["legal_person"]), strOr(body["register_address"]),
		strOr(body["invoice_title"]), strOr(body["tax_no"]), strOr(body["bank_name"]), strOr(body["bank_account"]),
		strOr(body["settle_method"]), nullInt(body["payment_days"]), nullFloat(body["credit_limit"]),
		strOrDef(body["currency"], "CNY"), nullFloat(body["tax_rate"]), nullInt(body["lead_time_days"]),
		nullFloat(body["moq"]), nullInt(body["default_warehouse_id"]), contact, strOr(body["remark"]),
	)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadSupplier(id))
	return true
}

func (s *Services) getSupplier(c *gin.Context) bool {
	id := paramID(c)
	m := s.loadSupplier(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) updateSupplier(c *gin.Context) bool {
	id := paramID(c)
	if s.loadSupplier(id) == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	body := bindBody(c)
	status, _ := body["status"].(string)
	if status != "" && !validSupplierStatus(status) {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	pref := boolInt(body["is_preferred"])
	if status == "" {
		_ = s.DB.QueryRow(`SELECT status FROM pur_supplier WHERE id=?`, id).Scan(&status)
	}
	if pref == 1 && status != "qualified" {
		api.FailJSON(c, "PREFERRED_REQUIRES_QUALIFIED")
		return true
	}
	_, err := s.DB.Exec(`UPDATE pur_supplier SET
		name=COALESCE(?,name), short_name=?, mnemonic=?, supplier_type=COALESCE(?,supplier_type),
		status=COALESCE(NULLIF(?,''),status), rating=?, is_preferred=?, uscc=?, legal_person=?, register_address=?,
		invoice_title=?, tax_no=?, bank_name=?, bank_account=?, settle_method=?, payment_days=?, credit_limit=?,
		currency=COALESCE(?,currency), tax_rate=?, lead_time_days=?, moq=?, default_warehouse_id=?,
		contact_json=?, remark=?, updated_at=NOW()
		WHERE id=?`,
		strOr(body["name"]), strOr(body["short_name"]), strOr(body["mnemonic"]), strOr(body["supplier_type"]),
		status, strOr(body["rating"]), pref, strOr(body["uscc"]), strOr(body["legal_person"]), strOr(body["register_address"]),
		strOr(body["invoice_title"]), strOr(body["tax_no"]), strOr(body["bank_name"]), strOr(body["bank_account"]),
		strOr(body["settle_method"]), nullInt(body["payment_days"]), nullFloat(body["credit_limit"]),
		strOr(body["currency"]), nullFloat(body["tax_rate"]), nullInt(body["lead_time_days"]),
		nullFloat(body["moq"]), nullInt(body["default_warehouse_id"]), jsonify(body["contact_json"]), strOr(body["remark"]), id,
	)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadSupplier(id))
	return true
}

func (s *Services) deleteSupplier(c *gin.Context) bool {
	id := paramID(c)
	_, _ = s.DB.Exec(`UPDATE pur_supplier SET is_deleted=1, status='eliminated', updated_at=NOW() WHERE id=?`, id)
	api.OK(c, gin.H{"id": id, "deleted": true})
	return true
}

func (s *Services) supplierStatusAction(c *gin.Context, act string) bool {
	id := paramID(c)
	if s.loadSupplier(id) == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	var next string
	switch act {
	case "qualify":
		next = "qualified"
	case "freeze":
		next = "frozen"
	case "blacklist":
		next = "blacklist"
	case "activate":
		next = "qualified"
	default:
		api.FailJSON(c, "UNKNOWN_ACTION")
		return true
	}
	_, err := s.DB.Exec(`UPDATE pur_supplier SET status=?, is_preferred=CASE WHEN ? IN ('frozen','blacklist','eliminated') THEN 0 ELSE is_preferred END, updated_at=NOW() WHERE id=?`,
		next, next, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	api.OK(c, s.loadSupplier(id))
	return true
}

func (s *Services) loadSupplier(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pur_supplier WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := list[0]
	decodeJSONField(m, "contact_json")
	return gin.H(m)
}

func (s *Services) supplierExists(id int64) bool {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_supplier WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&n)
	return n > 0
}

func (s *Services) supplierStatus(id int64) string {
	var st string
	_ = s.DB.QueryRow(`SELECT status FROM pur_supplier WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&st)
	return st
}

func (s *Services) assertSupplierCanPurchase(id int64) string {
	if !s.supplierExists(id) {
		return "SUPPLIER_NOT_FOUND"
	}
	st := s.supplierStatus(id)
	if st == "frozen" || st == "blacklist" || st == "eliminated" {
		return "SUPPLIER_BLOCKED:" + st
	}
	if st != "qualified" {
		return "SUPPLIER_NOT_QUALIFIED"
	}
	return ""
}

func validSupplierStatus(st string) bool {
	switch st {
	case "potential", "qualified", "frozen", "blacklist", "eliminated", "active":
		return true
	}
	return false
}

func (s *Services) handleSupplierLicenses(c *gin.Context, method string) bool {
	id := paramID(c)
	if !s.supplierExists(id) {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if method == "GET" {
		rows, err := s.DB.Query(`SELECT * FROM pur_supplier_license WHERE supplier_id=? ORDER BY id`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		api.OK(c, gin.H{"supplier_id": id, "list": list})
		return true
	}
	body := bindBody(c)
	items, _ := body["items"].([]interface{})
	if items == nil {
		if arr, ok := body["list"].([]interface{}); ok {
			items = arr
		}
	}
	_, _ = s.DB.Exec(`DELETE FROM pur_supplier_license WHERE supplier_id=?`, id)
	for _, it := range items {
		m, _ := it.(map[string]interface{})
		if m == nil {
			continue
		}
		lt, _ := m["license_type"].(string)
		if lt == "" {
			continue
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_supplier_license(supplier_id, license_type, license_no, expire_date, attachment_url, remark)
			VALUES(?,?,?,?,?,?)`, id, lt, strOr(m["license_no"]), strOr(m["expire_date"]), strOr(m["attachment_url"]), strOr(m["remark"]))
	}
	rows, _ := s.DB.Query(`SELECT * FROM pur_supplier_license WHERE supplier_id=? ORDER BY id`, id)
	list := []map[string]interface{}{}
	if rows != nil {
		defer rows.Close()
		list, _ = rowsToMaps(rows)
	}
	api.OK(c, gin.H{"supplier_id": id, "list": list})
	return true
}

func (s *Services) handleSupplierSupplyItems(c *gin.Context, method string) bool {
	id := paramID(c)
	if !s.supplierExists(id) {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if method == "GET" {
		rows, err := s.DB.Query(`SELECT * FROM pur_supplier_supply_item WHERE supplier_id=? ORDER BY id`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		api.OK(c, gin.H{"supplier_id": id, "list": list})
		return true
	}
	body := bindBody(c)
	items, _ := body["items"].([]interface{})
	if items == nil {
		items, _ = body["list"].([]interface{})
	}
	_, _ = s.DB.Exec(`DELETE FROM pur_supplier_supply_item WHERE supplier_id=?`, id)
	for _, it := range items {
		m, _ := it.(map[string]interface{})
		if m == nil {
			continue
		}
		pid, _ := asInt64(m["product_id"])
		if pid == 0 {
			continue
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_supplier_supply_item(supplier_id, product_id, is_preferred, moq, lead_time_days, last_price, remark)
			VALUES(?,?,?,?,?,?,?)`, id, pid, boolInt(m["is_preferred"]), nullFloat(m["moq"]), nullInt(m["lead_time_days"]), nullFloat(m["last_price"]), strOr(m["remark"]))
	}
	rows, _ := s.DB.Query(`SELECT * FROM pur_supplier_supply_item WHERE supplier_id=? ORDER BY id`, id)
	list := []map[string]interface{}{}
	if rows != nil {
		defer rows.Close()
		list, _ = rowsToMaps(rows)
	}
	api.OK(c, gin.H{"supplier_id": id, "list": list})
	return true
}

func (s *Services) handleCertificateAlerts(c *gin.Context) bool {
	days := 60
	if v := c.Query("days"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			days = n
		}
	}
	rows, err := s.DB.Query(`SELECT l.*, s.code AS supplier_code, s.name AS supplier_name, s.status AS supplier_status
		FROM pur_supplier_license l
		JOIN pur_supplier s ON s.id=l.supplier_id AND COALESCE(s.is_deleted,0)=0
		WHERE l.expire_date IS NOT NULL AND l.expire_date <> ''
		  AND date(l.expire_date) <= date('now', ?)
		ORDER BY l.expire_date`, fmt.Sprintf("+%d day", days))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"days": days, "list": list, "total": len(list)})
	return true
}

func (s *Services) handleSupplierPerformanceOne(c *gin.Context) bool {
	id := paramID(c)
	if !s.supplierExists(id) {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, s.calcSupplierPerf(id))
	return true
}

func (s *Services) handleSupplierPerformanceList(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT id FROM pur_supplier WHERE COALESCE(is_deleted,0)=0 ORDER BY id`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		list = append(list, s.calcSupplierPerf(id))
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) calcSupplierPerf(id int64) gin.H {
	sup := s.loadSupplier(id)
	var purchaseAmt, purchaseQty float64
	var inboundCnt int
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(l.qty),0), COALESCE(SUM(COALESCE(l.amount, l.qty*COALESCE(l.price,0))),0)
		FROM pur_purchase_inbound h
		JOIN pur_purchase_inbound_line l ON l.inbound_id=h.id
		WHERE h.supplier_id=? AND h.status='posted'`, id).Scan(&inboundCnt, &purchaseQty, &purchaseAmt)
	var qcTotal, qcPass, qcFail float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty_check),0), COALESCE(SUM(qty_pass),0), COALESCE(SUM(qty_fail),0)
		FROM pur_incoming_qc WHERE supplier_id=? OR inbound_id IN (SELECT id FROM pur_purchase_inbound WHERE supplier_id=?)`, id, id).
		Scan(&qcTotal, &qcPass, &qcFail)
	var returnQty float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.qty),0) FROM pur_purchase_return h
		JOIN pur_purchase_return_line l ON l.return_id=h.id
		WHERE h.supplier_id=? AND h.status='posted'`, id).Scan(&returnQty)
	var lastDate string
	_ = s.DB.QueryRow(`SELECT COALESCE(MAX(biz_date),'') FROM pur_purchase_inbound WHERE supplier_id=? AND status='posted'`, id).Scan(&lastDate)
	passRate := 0.0
	if qcTotal > 0 {
		passRate = qcPass / qcTotal
	}
	returnRate := 0.0
	if purchaseQty > 0 {
		returnRate = returnQty / purchaseQty
	}
	code := ""
	name := ""
	if sup != nil {
		if v, ok := sup["code"].(string); ok {
			code = v
		}
		if v, ok := sup["name"].(string); ok {
			name = v
		}
	}
	return gin.H{
		"supplier_id": id, "supplier_code": code, "supplier_name": name,
		"inbound_count": inboundCnt, "purchase_qty": purchaseQty, "purchase_amount": purchaseAmt,
		"qc_qty_check": qcTotal, "qc_qty_pass": qcPass, "qc_qty_fail": qcFail, "pass_rate": passRate,
		"return_qty": returnQty, "return_rate": returnRate, "last_purchase_date": lastDate,
	}
}

// ---------- inbound / qc / return ----------

func (s *Services) handlePurchaseInbounds(c *gin.Context, method, action string) bool {
	switch {
	case action == "list":
		return s.listDocTable(c, `SELECT * FROM pur_purchase_inbound WHERE COALESCE(is_deleted,0)=0`)
	case action == "get":
		return s.getInbound(c)
	case action == "create":
		return s.createInbound(c)
	case action == "action:post":
		return s.postInbound(c)
	case action == "update" || action == "replace":
		return s.handleTableCRUD(c, "purchase/inbounds", action)
	case action == "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_inbound SET is_deleted=1 WHERE id=? AND status='draft'`, id)
		api.OK(c, gin.H{"id": id})
		return true
	default:
		return true
	}
}

func (s *Services) createInbound(c *gin.Context) bool {
	body := bindBody(c)
	sid, _ := asInt64(body["supplier_id"])
	if sid == 0 {
		if name, ok := body["supplier"].(string); ok && name != "" {
			_ = s.DB.QueryRow(`SELECT id FROM pur_supplier WHERE name=? AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, name).Scan(&sid)
		}
	}
	if sid == 0 {
		api.FailJSON(c, "SUPPLIER_REQUIRED")
		return true
	}
	if msg := s.assertSupplierCanPurchase(sid); msg != "" {
		api.FailJSON(c, msg)
		return true
	}
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(default_warehouse_id,1) FROM pur_supplier WHERE id=?`, sid).Scan(&wh)
		if wh == 0 {
			wh = 1
		}
	}
	bizDate, _ := body["biz_date"].(string)
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	docNo, _ := body["doc_no"].(string)
	if docNo == "" {
		docNo = fmt.Sprintf("PI%d", time.Now().UnixNano()%1e12)
	}
	res, err := s.DB.Exec(`INSERT INTO pur_purchase_inbound(doc_no, supplier_id, warehouse_id, status, biz_date, plan_id, remark)
		VALUES(?,?,?,'draft',?,?,?)`, docNo, sid, wh, bizDate, nullInt(body["plan_id"]), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	hid, _ := res.LastInsertId()
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		// allow flat qty/product for loop demos
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if pid == 0 {
			pid = 1
		}
		if qty <= 0 {
			qty = 1
		}
		price, _ := asFloat(body["price"])
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_inbound_line(inbound_id, product_id, qty, price, amount, batch_no) VALUES(?,?,?,?,?,?)`,
			hid, pid, qty, nullFloat(body["price"]), qty*price, strOr(body["batch_no"]))
	} else {
		for _, ln := range lines {
			m, _ := ln.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			qty, _ := asFloat(m["qty"])
			price, _ := asFloat(m["price"])
			amt, ok := asFloat(m["amount"])
			if !ok {
				amt = qty * price
			}
			_, _ = s.DB.Exec(`INSERT INTO pur_purchase_inbound_line(inbound_id, product_id, qty, price, amount, batch_no) VALUES(?,?,?,?,?,?)`,
				hid, pid, qty, nullFloat(m["price"]), amt, strOr(m["batch_no"]))
		}
	}
	api.OK(c, s.loadInbound(hid))
	return true
}

func (s *Services) getInbound(c *gin.Context) bool {
	m := s.loadInbound(paramID(c))
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) loadInbound(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pur_purchase_inbound WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM pur_purchase_inbound_line WHERE inbound_id=? ORDER BY id`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) postInbound(c *gin.Context) bool {
	id := paramID(c)
	in := s.loadInbound(id)
	if in == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if st, _ := in["status"].(string); st == "posted" {
		api.OK(c, in)
		return true
	}
	sid, _ := asInt64(in["supplier_id"])
	if msg := s.assertSupplierCanPurchase(sid); msg != "" {
		api.FailJSON(c, msg)
		return true
	}
	wh, _ := asInt64(in["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	bizDate, _ := in["biz_date"].(string)
	lines, _ := in["lines"].([]map[string]interface{})
	if lines == nil {
		if raw, ok := in["lines"].([]interface{}); ok {
			for _, x := range raw {
				if m, ok := x.(map[string]interface{}); ok {
					lines = append(lines, m)
				}
			}
		}
	}
	// also support typed slice from rowsToMaps embedded
	if len(lines) == 0 {
		lrows, _ := s.DB.Query(`SELECT * FROM pur_purchase_inbound_line WHERE inbound_id=?`, id)
		if lrows != nil {
			defer lrows.Close()
			lines, _ = rowsToMaps(lrows)
		}
	}
	txnNo := fmt.Sprintf("ST-PI-%d", id)
	tres, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'draft',?,?)`,
		txnNo, "purchase_in", bizDate, wh, fmt.Sprintf("purchase inbound #%d", id))
	if err != nil {
		api.FailJSON(c, "STOCK_TXN_ERROR:"+err.Error())
		return true
	}
	tid, _ := tres.LastInsertId()
	for i, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		price, _ := asFloat(ln["price"])
		batch := strOr(ln["batch_no"])
		_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'in',?)`,
			tid, i+1, pid, qty, qty, batch)
		if err := s.adjustBalanceBatch(wh, pid, batch, qty); err != nil {
			api.FailJSON(c, "BALANCE_ERROR:"+err.Error())
			return true
		}
		if price > 0 {
			_, _ = s.DB.Exec(`INSERT INTO pur_supplier_price_history(supplier_id, product_id, price, biz_date, source_doc_id) VALUES(?,?,?,?,?)`,
				sid, pid, price, bizDate, id)
			_, _ = s.DB.Exec(`UPDATE pur_supplier_supply_item SET last_price=?, updated_at=NOW() WHERE supplier_id=? AND product_id=?`,
				price, sid, pid)
		}
	}
	_, _ = s.DB.Exec(`UPDATE inv_stock_txn SET status='posted' WHERE id=?`, tid)
	_, _ = s.DB.Exec(`UPDATE pur_purchase_inbound SET status='posted', updated_at=NOW() WHERE id=?`, id)
	api.OK(c, s.loadInbound(id))
	return true
}

func (s *Services) handleIncomingQCs(c *gin.Context, method, action string) bool {
	switch {
	case action == "list":
		return s.listDocTable(c, `SELECT * FROM pur_incoming_qc WHERE COALESCE(is_deleted,0)=0`)
	case action == "get":
		id := paramID(c)
		rows, err := s.DB.Query(`SELECT * FROM pur_incoming_qc WHERE id=?`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, list[0])
		return true
	case action == "create":
		return s.createIncomingQC(c)
	case action == "action:pass":
		return s.finishQC(c, "pass")
	case action == "action:fail":
		return s.finishQC(c, "fail")
	default:
		return true
	}
}

func (s *Services) createIncomingQC(c *gin.Context) bool {
	body := bindBody(c)
	docNo, _ := body["doc_no"].(string)
	if docNo == "" {
		docNo = fmt.Sprintf("QC%d", time.Now().UnixNano()%1e12)
	}
	inboundID, _ := asInt64(body["inbound_id"])
	sid, _ := asInt64(body["supplier_id"])
	if sid == 0 && inboundID > 0 {
		_ = s.DB.QueryRow(`SELECT supplier_id FROM pur_purchase_inbound WHERE id=?`, inboundID).Scan(&sid)
	}
	pid, _ := asInt64(body["product_id"])
	if pid == 0 {
		pid = 1
	}
	qty, _ := asFloat(body["qty_check"])
	if qty <= 0 {
		qty, _ = asFloat(body["qty"])
	}
	if qty <= 0 {
		qty = 1
	}
	res, err := s.DB.Exec(`INSERT INTO pur_incoming_qc(doc_no, inbound_id, supplier_id, product_id, qty_check, qty_pass, qty_fail, status, remark)
		VALUES(?,?,?,?,?,0,0,'draft',?)`, docNo, nullIf0(inboundID), nullIf0(sid), pid, qty, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	rows, _ := s.DB.Query(`SELECT * FROM pur_incoming_qc WHERE id=?`, id)
	list := []map[string]interface{}{}
	if rows != nil {
		defer rows.Close()
		list, _ = rowsToMaps(rows)
	}
	if len(list) > 0 {
		api.OK(c, list[0])
	} else {
		api.OK(c, gin.H{"id": id})
	}
	return true
}

func (s *Services) finishQC(c *gin.Context, result string) bool {
	id := paramID(c)
	var qtyCheck float64
	err := s.DB.QueryRow(`SELECT qty_check FROM pur_incoming_qc WHERE id=?`, id).Scan(&qtyCheck)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	pass, fail := 0.0, 0.0
	if result == "pass" {
		pass = qtyCheck
	} else {
		fail = qtyCheck
	}
	_, _ = s.DB.Exec(`UPDATE pur_incoming_qc SET result=?, status=?, qty_pass=?, qty_fail=?, updated_at=NOW() WHERE id=?`,
		result, result, pass, fail, id)
	rows, _ := s.DB.Query(`SELECT * FROM pur_incoming_qc WHERE id=?`, id)
	list := []map[string]interface{}{}
	if rows != nil {
		defer rows.Close()
		list, _ = rowsToMaps(rows)
	}
	if len(list) > 0 {
		api.OK(c, list[0])
	} else {
		api.OK(c, gin.H{"id": id, "result": result})
	}
	return true
}

func (s *Services) handlePurchaseReturns(c *gin.Context, method, action string) bool {
	switch {
	case action == "list":
		return s.listDocTable(c, `SELECT * FROM pur_purchase_return WHERE COALESCE(is_deleted,0)=0`)
	case action == "get":
		api.OK(c, s.loadReturn(paramID(c)))
		return true
	case action == "create":
		return s.createReturn(c)
	case action == "action:post":
		return s.postReturn(c)
	default:
		return true
	}
}

func (s *Services) createReturn(c *gin.Context) bool {
	body := bindBody(c)
	sid, _ := asInt64(body["supplier_id"])
	if sid == 0 {
		api.FailJSON(c, "SUPPLIER_REQUIRED")
		return true
	}
	docNo, _ := body["doc_no"].(string)
	if docNo == "" {
		docNo = fmt.Sprintf("PR%d", time.Now().UnixNano()%1e12)
	}
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	res, err := s.DB.Exec(`INSERT INTO pur_purchase_return(doc_no, supplier_id, inbound_id, warehouse_id, status, reason)
		VALUES(?,?,?,?,'draft',?)`, docNo, sid, nullInt(body["inbound_id"]), wh, strOr(body["reason"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	rid, _ := res.LastInsertId()
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if pid == 0 {
			pid = 1
		}
		if qty <= 0 {
			qty = 1
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_return_line(return_id, product_id, qty, amount) VALUES(?,?,?,?)`, rid, pid, qty, nullFloat(body["amount"]))
	} else {
		for _, ln := range lines {
			m, _ := ln.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			qty, _ := asFloat(m["qty"])
			_, _ = s.DB.Exec(`INSERT INTO pur_purchase_return_line(return_id, product_id, qty, amount) VALUES(?,?,?,?)`,
				rid, pid, qty, nullFloat(m["amount"]))
		}
	}
	api.OK(c, s.loadReturn(rid))
	return true
}

func (s *Services) loadReturn(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pur_purchase_return WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM pur_purchase_return_line WHERE return_id=?`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) postReturn(c *gin.Context) bool {
	id := paramID(c)
	ret := s.loadReturn(id)
	if ret == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if st, _ := ret["status"].(string); st == "posted" {
		api.OK(c, ret)
		return true
	}
	wh, _ := asInt64(ret["warehouse_id"])
	if wh == 0 {
		wh = 1
	}
	type rl struct {
		pid int64
		qty float64
	}
	var rlines []rl
	lrows, _ := s.DB.Query(`SELECT product_id, qty FROM pur_purchase_return_line WHERE return_id=?`, id)
	if lrows != nil {
		for lrows.Next() {
			var pid int64
			var qty float64
			_ = lrows.Scan(&pid, &qty)
			rlines = append(rlines, rl{pid, qty})
		}
		lrows.Close()
	}
	for _, ln := range rlines {
		if err := s.adjustBalance(wh, ln.pid, -ln.qty); err != nil {
			api.FailJSON(c, "BALANCE_ERROR:"+err.Error())
			return true
		}
	}
	_, _ = s.DB.Exec(`UPDATE pur_purchase_return SET status='posted', updated_at=NOW() WHERE id=?`, id)
	api.OK(c, s.loadReturn(id))
	return true
}

func (s *Services) handlePriceHistories(c *gin.Context) bool {
	where := "1=1"
	args := []interface{}{}
	if v := c.Query("supplier_id"); v != "" {
		where += " AND supplier_id=?"
		args = append(args, v)
	}
	if v := c.Query("product_id"); v != "" {
		where += " AND product_id=?"
		args = append(args, v)
	}
	pageNum, pageSize := sqlutil.Page(c)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_supplier_price_history WHERE `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT * FROM pur_supplier_price_history WHERE `+where+` ORDER BY biz_date DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) handleVolumePrice(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT h.supplier_id, s.code AS supplier_code, s.name AS supplier_name,
		l.product_id, SUM(l.qty) AS qty, SUM(COALESCE(l.amount, l.qty*COALESCE(l.price,0))) AS amount,
		CASE WHEN SUM(l.qty)>0 THEN SUM(COALESCE(l.amount, l.qty*COALESCE(l.price,0)))/SUM(l.qty) ELSE 0 END AS avg_price
		FROM pur_purchase_inbound h
		JOIN pur_purchase_inbound_line l ON l.inbound_id=h.id
		LEFT JOIN pur_supplier s ON s.id=h.supplier_id
		WHERE h.status='posted'
		GROUP BY h.supplier_id, l.product_id
		ORDER BY amount DESC`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) handlePurchaseRequests(c *gin.Context, method, action string) bool {
	switch {
	case action == "create":
		body := bindBody(c)
		docNo, _ := body["doc_no"].(string)
		if docNo == "" {
			docNo = fmt.Sprintf("PRQ%s", time.Now().Format("060102150405"))
		}
		sug, _ := asInt64(body["suggest_supplier_id"])
		if sug == 0 {
			sug, _ = asInt64(body["supplier_id"])
		}
		if sug > 0 && !s.supplierExists(sug) {
			api.FailJSON(c, "SUPPLIER_NOT_FOUND")
			return true
		}
		applicant := claimsUserID(c)
		if v, ok := asInt64(body["applicant_id"]); ok && v > 0 {
			applicant = v
		}
		res, err := s.DB.Exec(`INSERT INTO pur_purchase_request(doc_no, applicant_id, title, qty, status, need_date, remark)
			VALUES(?,?,?,?,'draft',?,?)`, docNo, nullIf0(applicant), strOrDef(body["title"], strOr(body["name"])),
			nullFloat(body["qty"]), strOr(body["need_date"]), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		hid, _ := res.LastInsertId()
		s.insertRequestLines(hid, body, sug)
		api.OK(c, s.loadPurchaseRequest(hid))
		return true
	case action == "update" || action == "replace":
		id := paramID(c)
		body := bindBody(c)
		var status string
		_ = s.DB.QueryRow(`SELECT status FROM pur_purchase_request WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&status)
		if status == "" {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status != "draft" && status != "rejected" {
			api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
			return true
		}
		_, err := s.DB.Exec(`UPDATE pur_purchase_request SET
			title=COALESCE(NULLIF(?,''),title), qty=COALESCE(?,qty), need_date=COALESCE(NULLIF(?,''),need_date),
			remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
			strOr(body["title"]), nullFloat(body["qty"]), strOr(body["need_date"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		if _, ok := body["product_id"]; ok || body["lines"] != nil {
			_, _ = s.DB.Exec(`DELETE FROM pur_purchase_request_line WHERE request_id=?`, id)
			sug, _ := asInt64(body["suggest_supplier_id"])
			if sug == 0 {
				sug, _ = asInt64(body["supplier_id"])
			}
			s.insertRequestLines(id, body, sug)
		}
		api.OK(c, s.loadPurchaseRequest(id))
		return true
	case action == "action:submit":
		id := paramID(c)
		_, err := s.DB.Exec(`UPDATE pur_purchase_request SET status='submitted', updated_at=NOW() WHERE id=? AND status IN ('draft','rejected')`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, s.loadPurchaseRequest(id))
		return true
	case action == "action:approve":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET status='approved', updated_at=NOW() WHERE id=? AND status='submitted'`, id)
		api.OK(c, s.loadPurchaseRequest(id))
		return true
	case action == "action:reject":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET status='rejected', remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=? AND status='submitted'`,
			strOr(body["reason"]), id)
		api.OK(c, s.loadPurchaseRequest(id))
		return true
	case action == "action:to-plan" || strings.Contains(c.FullPath(), "/to-plan"):
		return s.convertRequestToPlan(c)
	case action == "list":
		return s.listDocTable(c, `SELECT * FROM pur_purchase_request WHERE COALESCE(is_deleted,0)=0`)
	case action == "get":
		m := s.loadPurchaseRequest(paramID(c))
		if m == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET is_deleted=1, status='cancelled', updated_at=NOW() WHERE id=? AND status='draft'`, id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	return true
}

func (s *Services) insertRequestLines(hid int64, body map[string]interface{}, sug int64) {
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if pid == 0 {
			pid = 1
		}
		if qty <= 0 {
			qty = 1
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_request_line(request_id, product_id, qty, suggest_supplier_id) VALUES(?,?,?,?)`,
			hid, pid, qty, nullIf0(sug))
		return
	}
	var total float64
	for _, ln := range lines {
		m, _ := ln.(map[string]interface{})
		if m == nil {
			continue
		}
		pid, _ := asInt64(m["product_id"])
		qty, _ := asFloat(m["qty"])
		lineSug, _ := asInt64(m["suggest_supplier_id"])
		if lineSug == 0 {
			lineSug = sug
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_request_line(request_id, product_id, qty, suggest_supplier_id) VALUES(?,?,?,?)`,
			hid, pid, qty, nullIf0(lineSug))
		total += qty
	}
	if total > 0 {
		_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET qty=? WHERE id=?`, total, hid)
	}
}

func (s *Services) loadPurchaseRequest(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pur_purchase_request WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM pur_purchase_request_line WHERE request_id=? ORDER BY id`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) convertRequestToPlan(c *gin.Context) bool {
	id := paramID(c)
	req := s.loadPurchaseRequest(id)
	if req == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(req["status"])
	if st != "approved" && st != "submitted" {
		api.FailJSON(c, "REQUEST_NOT_APPROVED")
		return true
	}
	docNo := fmt.Sprintf("PPL%s", time.Now().Format("060102150405"))
	res, err := s.DB.Exec(`INSERT INTO pur_purchase_plan(doc_no, status, plan_date, remark) VALUES(?,'draft',?,?)`,
		docNo, time.Now().Format("2006-01-02"), fmt.Sprintf("来自申请 %s", strOr(req["doc_no"])))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	pid, _ := res.LastInsertId()
	lines, _ := req["lines"].([]map[string]interface{})
	if lines == nil {
		if raw, ok := req["lines"].([]interface{}); ok {
			for _, x := range raw {
				if m, ok := x.(map[string]interface{}); ok {
					lines = append(lines, m)
				}
			}
		}
	}
	for _, ln := range lines {
		productID, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		sug, _ := asInt64(ln["suggest_supplier_id"])
		lineID, _ := asInt64(ln["id"])
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_plan_line(plan_id, product_id, qty, supplier_id, request_line_id) VALUES(?,?,?,?,?)`,
			pid, productID, qty, nullIf0(sug), nullIf0(lineID))
	}
	_, _ = s.DB.Exec(`UPDATE pur_purchase_request SET status='planned', updated_at=NOW() WHERE id=?`, id)
	out := s.loadPurchasePlan(pid)
	out["from_request_id"] = id
	api.OK(c, out)
	return true
}

func (s *Services) handlePurchasePlans(c *gin.Context, method, action string) bool {
	switch {
	case action == "create":
		body := bindBody(c)
		docNo, _ := body["doc_no"].(string)
		if docNo == "" {
			docNo = fmt.Sprintf("PPL%s", time.Now().Format("060102150405"))
		}
		sid, _ := asInt64(body["supplier_id"])
		if sid > 0 {
			if msg := s.assertSupplierCanPurchase(sid); msg != "" && msg != "SUPPLIER_NOT_QUALIFIED" {
				if strings.HasPrefix(msg, "SUPPLIER_BLOCKED") || msg == "SUPPLIER_NOT_FOUND" {
					api.FailJSON(c, msg)
					return true
				}
			}
		}
		res, err := s.DB.Exec(`INSERT INTO pur_purchase_plan(doc_no, status, plan_date, remark) VALUES(?,'draft',?,?)`,
			docNo, strOrDef(body["plan_date"], time.Now().Format("2006-01-02")), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		hid, _ := res.LastInsertId()
		s.insertPlanLines(hid, body, sid)
		api.OK(c, s.loadPurchasePlan(hid))
		return true
	case action == "update" || action == "replace":
		id := paramID(c)
		body := bindBody(c)
		var status string
		_ = s.DB.QueryRow(`SELECT status FROM pur_purchase_plan WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&status)
		if status == "" {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status != "draft" && status != "rejected" {
			api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
			return true
		}
		_, err := s.DB.Exec(`UPDATE pur_purchase_plan SET
			plan_date=COALESCE(NULLIF(?,''),plan_date), remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=?`,
			strOr(body["plan_date"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		if _, ok := body["product_id"]; ok || body["lines"] != nil {
			_, _ = s.DB.Exec(`DELETE FROM pur_purchase_plan_line WHERE plan_id=?`, id)
			sid, _ := asInt64(body["supplier_id"])
			s.insertPlanLines(id, body, sid)
		}
		api.OK(c, s.loadPurchasePlan(id))
		return true
	case action == "action:submit":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_plan SET status='submitted', updated_at=NOW() WHERE id=? AND status IN ('draft','rejected')`, id)
		api.OK(c, s.loadPurchasePlan(id))
		return true
	case action == "action:approve":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_plan SET status='approved', updated_at=NOW() WHERE id=? AND status='submitted'`, id)
		api.OK(c, s.loadPurchasePlan(id))
		return true
	case action == "action:to-inbound" || strings.Contains(c.FullPath(), "/to-inbound"):
		return s.convertPlanToInbound(c)
	case action == "list":
		return s.listDocTable(c, `SELECT * FROM pur_purchase_plan WHERE COALESCE(is_deleted,0)=0`)
	case action == "get":
		m := s.loadPurchasePlan(paramID(c))
		if m == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_plan SET is_deleted=1, status='cancelled', updated_at=NOW() WHERE id=? AND status='draft'`, id)
		api.OK(c, gin.H{"id": id})
		return true
	}
	_ = method
	return true
}

func (s *Services) insertPlanLines(hid int64, body map[string]interface{}, sid int64) {
	lines, _ := body["lines"].([]interface{})
	if len(lines) == 0 {
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if pid == 0 {
			pid = 1
		}
		if qty <= 0 {
			qty = 1
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_plan_line(plan_id, product_id, qty, supplier_id) VALUES(?,?,?,?)`, hid, pid, qty, nullIf0(sid))
		return
	}
	for _, ln := range lines {
		m, _ := ln.(map[string]interface{})
		if m == nil {
			continue
		}
		pid, _ := asInt64(m["product_id"])
		qty, _ := asFloat(m["qty"])
		lineSid, _ := asInt64(m["supplier_id"])
		if lineSid == 0 {
			lineSid = sid
		}
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_plan_line(plan_id, product_id, qty, supplier_id, request_line_id) VALUES(?,?,?,?,?)`,
			hid, pid, qty, nullIf0(lineSid), nullInt(m["request_line_id"]))
	}
}

func (s *Services) loadPurchasePlan(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pur_purchase_plan WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	m := gin.H(list[0])
	lrows, _ := s.DB.Query(`SELECT * FROM pur_purchase_plan_line WHERE plan_id=? ORDER BY id`, id)
	lines := []map[string]interface{}{}
	if lrows != nil {
		defer lrows.Close()
		lines, _ = rowsToMaps(lrows)
	}
	m["lines"] = lines
	return m
}

func (s *Services) convertPlanToInbound(c *gin.Context) bool {
	id := paramID(c)
	plan := s.loadPurchasePlan(id)
	if plan == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(plan["status"])
	if st != "approved" && st != "submitted" {
		api.FailJSON(c, "PLAN_NOT_APPROVED")
		return true
	}
	body := bindBody(c)
	sid, _ := asInt64(body["supplier_id"])
	lines, _ := plan["lines"].([]map[string]interface{})
	if lines == nil {
		if raw, ok := plan["lines"].([]interface{}); ok {
			for _, x := range raw {
				if m, ok := x.(map[string]interface{}); ok {
					lines = append(lines, m)
				}
			}
		}
	}
	if sid == 0 && len(lines) > 0 {
		sid, _ = asInt64(lines[0]["supplier_id"])
	}
	if sid == 0 {
		api.FailJSON(c, "SUPPLIER_REQUIRED")
		return true
	}
	if msg := s.assertSupplierCanPurchase(sid); msg != "" {
		api.FailJSON(c, msg)
		return true
	}
	wh, _ := asInt64(body["warehouse_id"])
	if wh == 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(default_warehouse_id,1) FROM pur_supplier WHERE id=?`, sid).Scan(&wh)
		if wh == 0 {
			wh = 1
		}
	}
	docNo := fmt.Sprintf("PI%s", time.Now().Format("060102150405"))
	bizDate := time.Now().Format("2006-01-02")
	res, err := s.DB.Exec(`INSERT INTO pur_purchase_inbound(doc_no, supplier_id, warehouse_id, status, biz_date, plan_id, remark)
		VALUES(?,?,?,'draft',?,?,?)`, docNo, sid, wh, bizDate, id, fmt.Sprintf("来自计划 %s", strOr(plan["doc_no"])))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	hid, _ := res.LastInsertId()
	price, _ := asFloat(body["price"])
	for _, ln := range lines {
		pid, _ := asInt64(ln["product_id"])
		qty, _ := asFloat(ln["qty"])
		_, _ = s.DB.Exec(`INSERT INTO pur_purchase_inbound_line(inbound_id, product_id, qty, price, amount, batch_no) VALUES(?,?,?,?,?,?)`,
			hid, pid, qty, nullFloat(price), qty*price, strOr(body["batch_no"]))
	}
	_, _ = s.DB.Exec(`UPDATE pur_purchase_plan SET status='ordered', updated_at=NOW() WHERE id=?`, id)
	api.OK(c, s.loadInbound(hid))
	return true
}

func (s *Services) handlePurchaseTasks(c *gin.Context, method, action string) bool {
	switch {
	case action == "list":
		return s.listDocTable(c, `SELECT * FROM pur_purchase_task WHERE COALESCE(is_deleted,0)=0`)
	case action == "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("PT%s", time.Now().Format("060102150405")))
		res, err := s.DB.Exec(`INSERT INTO pur_purchase_task(doc_no, title, assignee_id, product_id, qty, supplier_id, status, due_date, remark)
			VALUES(?,?,?,?,?,?,'open',?,?)`,
			docNo, strOrDef(body["title"], "采购任务"), nullInt(body["assignee_id"]), nullInt(body["product_id"]),
			nullFloat(body["qty"]), nullInt(body["supplier_id"]), strOr(body["due_date"]), strOr(body["remark"]))
		if err != nil {
			// fallback without extended columns
			res, err = s.DB.Exec(`INSERT INTO pur_purchase_task(doc_no, assignee_id, product_id, qty, status, due_date)
				VALUES(?,?,?,?,'open',?)`,
				docNo, nullInt(body["assignee_id"]), nullInt(body["product_id"]), nullFloat(body["qty"]), strOr(body["due_date"]))
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		}
		id, _ := res.LastInsertId()
		api.OK(c, s.loadPurchaseTask(id))
		return true
	case action == "get":
		m := s.loadPurchaseTask(paramID(c))
		if m == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case action == "update" || action == "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE pur_purchase_task SET
			assignee_id=COALESCE(?,assignee_id), product_id=COALESCE(?,product_id), qty=COALESCE(?,qty),
			due_date=COALESCE(NULLIF(?,''),due_date), status=COALESCE(NULLIF(?,''),status),
			updated_at=NOW() WHERE id=?`,
			nullInt(body["assignee_id"]), nullInt(body["product_id"]), nullFloat(body["qty"]),
			strOr(body["due_date"]), strOr(body["status"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		// best-effort extended fields
		_, _ = s.DB.Exec(`UPDATE pur_purchase_task SET title=COALESCE(NULLIF(?,''),title), remark=COALESCE(NULLIF(?,''),remark),
			supplier_id=COALESCE(?,supplier_id) WHERE id=?`,
			strOr(body["title"]), strOr(body["remark"]), nullInt(body["supplier_id"]), id)
		api.OK(c, s.loadPurchaseTask(id))
		return true
	case action == "action:assign":
		id := paramID(c)
		body := bindBody(c)
		aid, _ := asInt64(body["assignee_id"])
		if aid == 0 {
			aid = claimsUserID(c)
		}
		_, _ = s.DB.Exec(`UPDATE pur_purchase_task SET assignee_id=?, status='assigned', updated_at=NOW() WHERE id=?`, aid, id)
		api.OK(c, s.loadPurchaseTask(id))
		return true
	case action == "action:complete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_purchase_task SET status='done', updated_at=NOW() WHERE id=?`, id)
		api.OK(c, s.loadPurchaseTask(id))
		return true
	case action == "delete":
		_, _ = s.DB.Exec(`UPDATE pur_purchase_task SET is_deleted=1, updated_at=NOW() WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	_ = method
	return true
}

func (s *Services) loadPurchaseTask(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pur_purchase_task WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	return gin.H(list[0])
}

func (s *Services) listDocTable(c *gin.Context, baseSQL string) bool {
	pageNum, pageSize := sqlutil.Page(c)
	// naive count: wrap
	countSQL := "SELECT COUNT(1) FROM (" + baseSQL + ") t"
	var total int
	_ = s.DB.QueryRow(countSQL).Scan(&total)
	rows, err := s.DB.Query(baseSQL+` ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

// helpers

func strOr(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func strOrDef(v interface{}, def string) string {
	s := strOr(v)
	if s == "" {
		return def
	}
	return s
}

func nullInt(v interface{}) interface{} {
	n, ok := asInt64(v)
	if !ok || n == 0 {
		// allow explicit 0 for payment_days etc — use presence
		if v == nil {
			return nil
		}
		if f, ok := asFloat(v); ok {
			return int64(f)
		}
		return nil
	}
	return n
}

func nullFloat(v interface{}) interface{} {
	f, ok := asFloat(v)
	if !ok {
		return nil
	}
	return f
}

func jsonify(v interface{}) string {
	if v == nil {
		return "[]"
	}
	switch t := v.(type) {
	case string:
		if t == "" {
			return "[]"
		}
		return t
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "[]"
		}
		return string(b)
	}
}

func decodeJSONField(m map[string]interface{}, key string) {
	raw, _ := m[key].(string)
	if raw == "" {
		return
	}
	var v interface{}
	if json.Unmarshal([]byte(raw), &v) == nil {
		m[key] = v
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
