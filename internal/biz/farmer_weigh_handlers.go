package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleFarmers(c *gin.Context, method, action string) bool {
	switch {
	case action == "list":
		return s.listFarmers(c)
	case action == "create":
		return s.createFarmer(c)
	case action == "get":
		return s.getFarmer(c)
	case action == "update" || action == "replace":
		return s.updateFarmer(c)
	case action == "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE pur_farmer SET status='inactive', is_deleted=1, updated_at=datetime('now') WHERE id=?`, id)
		api.OK(c, gin.H{})
		return true
	}
	return false
}

func (s *Services) listFarmers(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	kw := strings.TrimSpace(c.Query("keyword"))
	where := `WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	if kw != "" {
		where += ` AND (name LIKE ? OR mobile LIKE ? OR code LIKE ? OR origin LIKE ? OR trace_code LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_farmer `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(mobile,''), COALESCE(origin,''), COALESCE(trace_code,''),
		COALESCE(trace_code_prefix,''), status, COALESCE(remark,''), created_at
		FROM pur_farmer `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, mobile, origin, trace, prefix, status, remark, created string
		_ = rows.Scan(&id, &code, &name, &mobile, &origin, &trace, &prefix, &status, &remark, &created)
		list = append(list, gin.H{
			"id": id, "code": code, "name": name, "mobile": mobile, "origin": origin,
			"trace_code": trace, "trace_code_prefix": prefix, "status": status, "remark": remark, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createFarmer(c *gin.Context) bool {
	body := bindBody(c)
	name := strOr(body["name"])
	if name == "" {
		api.FailJSON(c, "NAME_REQUIRED")
		return true
	}
	code := strOr(body["code"])
	if code == "" {
		code = fmt.Sprintf("F%s", time.Now().Format("060102150405"))
	}
	mobile := strOr(body["mobile"])
	origin := strOr(body["origin"])
	prefix := strOrDef(body["trace_code_prefix"], "TR")
	trace := strOr(body["trace_code"])
	if trace == "" {
		trace = fmt.Sprintf("%s-%s-%d", prefix, time.Now().Format("20060102"), time.Now().UnixNano()%1e6)
	}
	status := strOrDef(body["status"], "active")
	remark := strOr(body["remark"])
	res, err := s.DB.Exec(`INSERT INTO pur_farmer(code, name, mobile, origin, trace_code, trace_code_prefix, status, remark)
		VALUES(?,?,?,?,?,?,?,?)`, code, name, mobile, origin, trace, prefix, status, remark)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadFarmer(id))
	return true
}

func (s *Services) getFarmer(c *gin.Context) bool {
	m := s.loadFarmer(paramID(c))
	if m["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) updateFarmer(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	_, err := s.DB.Exec(`UPDATE pur_farmer SET name=COALESCE(NULLIF(?,''),name), mobile=COALESCE(NULLIF(?,''),mobile),
		origin=COALESCE(NULLIF(?,''),origin), trace_code=COALESCE(NULLIF(?,''),trace_code),
		trace_code_prefix=COALESCE(NULLIF(?,''),trace_code_prefix), status=COALESCE(NULLIF(?,''),status),
		remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=? AND COALESCE(is_deleted,0)=0`,
		strOr(body["name"]), strOr(body["mobile"]), strOr(body["origin"]), strOr(body["trace_code"]),
		strOr(body["trace_code_prefix"]), strOr(body["status"]), strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadFarmer(id))
	return true
}

func (s *Services) loadFarmer(id int64) gin.H {
	var code, name, mobile, origin, trace, prefix, status, remark, created string
	err := s.DB.QueryRow(`SELECT code, name, COALESCE(mobile,''), COALESCE(origin,''), COALESCE(trace_code,''),
		COALESCE(trace_code_prefix,''), status, COALESCE(remark,''), created_at FROM pur_farmer WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&code, &name, &mobile, &origin, &trace, &prefix, &status, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "code": code, "name": name, "mobile": mobile, "origin": origin,
		"trace_code": trace, "trace_code_prefix": prefix, "status": status, "remark": remark, "created_at": created,
	}
}

// ---------- weigh tickets ----------

func (s *Services) handleWeighTickets(c *gin.Context, method, action string) bool {
	switch {
	case action == "list":
		return s.listWeighTickets(c)
	case action == "create":
		return s.createWeighTicket(c)
	case action == "get":
		m := s.loadWeighTicket(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case strings.HasSuffix(action, "qc") || (method == "POST" && strings.Contains(c.Request.URL.Path, "/qc")):
		return s.qcWeighTicket(c)
	case strings.HasSuffix(action, "stock-in") || (method == "POST" && strings.Contains(c.Request.URL.Path, "/stock-in")):
		return s.stockInWeighTicket(c)
	case action == "update" || action == "replace":
		return s.updateWeighTicket(c)
	}
	return false
}

func (s *Services) listWeighTickets(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_weigh_ticket WHERE COALESCE(is_deleted,0)=0`).Scan(&total)
	rows, err := s.DB.Query(`SELECT w.id, w.doc_no, w.farmer_id, COALESCE(f.name,''), w.channel, w.product_id,
		w.variety, w.gross_weight, w.deduct_rate, w.deduct_weight, w.net_weight, w.qc_result, w.status,
		COALESCE(w.trace_code,''), COALESCE(w.origin,''), w.biz_date, COALESCE(w.source_type,'self'),
		COALESCE(w.image_url,''), COALESCE(w.box_code,''), w.created_at
		FROM pur_weigh_ticket w LEFT JOIN pur_farmer f ON f.id=w.farmer_id
		WHERE COALESCE(w.is_deleted,0)=0 ORDER BY w.id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, farmerID, productID int64
		var docNo, farmerName, channel, variety, qc, status, trace, origin, bizDate, source, image, box, created string
		var gross, deductRate, deductWeight, net float64
		_ = rows.Scan(&id, &docNo, &farmerID, &farmerName, &channel, &productID, &variety, &gross, &deductRate,
			&deductWeight, &net, &qc, &status, &trace, &origin, &bizDate, &source, &image, &box, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName, "channel": channel,
			"product_id": productID, "variety": variety, "gross_weight": gross, "deduct_rate": deductRate,
			"deduct_weight": deductWeight, "net_weight": net, "qc_result": qc, "status": status,
			"trace_code": trace, "origin": origin, "biz_date": bizDate, "source_type": source,
			"image_url": image, "box_code": box, "created_at": created,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createWeighTicket(c *gin.Context) bool {
	body := bindBody(c)
	farmerID, _ := asInt64(body["farmer_id"])
	if farmerID <= 0 {
		api.FailJSON(c, "FARMER_REQUIRED")
		return true
	}
	var farmerName, farmerOrigin, farmerTrace string
	err := s.DB.QueryRow(`SELECT name, COALESCE(origin,''), COALESCE(trace_code,'') FROM pur_farmer WHERE id=? AND status='active' AND COALESCE(is_deleted,0)=0`, farmerID).
		Scan(&farmerName, &farmerOrigin, &farmerTrace)
	if err != nil {
		api.FailJSON(c, "FARMER_NOT_FOUND")
		return true
	}
	channel := strOrDef(body["channel"], "internal") // external | internal
	if channel != "external" && channel != "internal" {
		api.FailJSON(c, "INVALID_CHANNEL")
		return true
	}
	productID, _ := asInt64(body["product_id"])
	if productID <= 0 {
		productID = 1
	}
	variety := strOrDef(body["variety"], "鲜木薯")
	gross, _ := asFloat(body["gross_weight"])
	if gross <= 0 {
		api.FailJSON(c, "GROSS_WEIGHT_REQUIRED")
		return true
	}
	deductRate, hasRate := asFloat(body["deduct_rate"])
	deductWeight, hasDeduct := asFloat(body["deduct_weight"])
	if hasRate && deductRate > 0 {
		if deductRate > 1 {
			deductRate = deductRate / 100 // allow 5 meaning 5%
		}
		deductWeight = gross * deductRate
	} else if hasDeduct && deductWeight > 0 {
		deductRate = deductWeight / gross
	}
	net, hasNet := asFloat(body["net_weight"])
	if !hasNet || net <= 0 {
		net = gross - deductWeight
	}
	if net < 0 {
		net = 0
	}
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	origin := strOrDef(body["origin"], farmerOrigin)
	trace := strOr(body["trace_code"])
	if trace == "" {
		trace = fmt.Sprintf("%s-%s", farmerTrace, time.Now().Format("150405"))
		if farmerTrace == "" {
			trace = fmt.Sprintf("TR-%s-%d", bizDate, time.Now().UnixNano()%1e6)
		}
	}
	sourceType := strOrDef(body["source_type"], "self") // self | outsource
	imageURL := strOr(body["image_url"])
	template := strOrDef(body["ticket_template"], channel) // external_std | handwritten | internal
	docNo := fmt.Sprintf("WT%s", time.Now().Format("20060102150405"))
	res, err := s.DB.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, channel, ticket_template, product_id, variety,
		gross_weight, deduct_rate, deduct_weight, net_weight, qc_result, status, trace_code, origin, biz_date,
		source_type, image_url, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?,'','draft',?,?,?,?,?,?)`,
		docNo, farmerID, channel, template, productID, variety, gross, deductRate, deductWeight, net,
		trace, origin, bizDate, sourceType, imageURL, strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, s.loadWeighTicket(id))
	return true
}

func (s *Services) updateWeighTicket(c *gin.Context) bool {
	id := paramID(c)
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM pur_weigh_ticket WHERE id=?`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status != "draft" {
		api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
		return true
	}
	body := bindBody(c)
	gross, _ := asFloat(body["gross_weight"])
	deductRate, _ := asFloat(body["deduct_rate"])
	deductWeight, _ := asFloat(body["deduct_weight"])
	if deductRate > 1 {
		deductRate = deductRate / 100
	}
	if deductRate > 0 && gross > 0 {
		deductWeight = gross * deductRate
	}
	net := gross - deductWeight
	if n, ok := asFloat(body["net_weight"]); ok && n > 0 {
		net = n
	}
	_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET gross_weight=COALESCE(NULLIF(?,0),gross_weight),
		deduct_rate=?, deduct_weight=?, net_weight=?, variety=COALESCE(NULLIF(?,''),variety),
		image_url=COALESCE(NULLIF(?,''),image_url), remark=COALESCE(NULLIF(?,''),remark),
		updated_at=datetime('now') WHERE id=?`,
		gross, deductRate, deductWeight, net, strOr(body["variety"]), strOr(body["image_url"]), strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, s.loadWeighTicket(id))
	return true
}

func (s *Services) qcWeighTicket(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	result := strings.ToLower(strOrDef(body["qc_result"], strOr(body["result"])))
	if result != "pass" && result != "fail" && result != "合格" && result != "不合格" {
		api.FailJSON(c, "QC_RESULT_REQUIRED")
		return true
	}
	if result == "合格" {
		result = "pass"
	}
	if result == "不合格" {
		result = "fail"
	}
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM pur_weigh_ticket WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status != "draft" && status != "qc_pending" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	newStatus := "qc_fail"
	if result == "pass" {
		newStatus = "qc_pass"
	}
	_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET qc_result=?, status=?, remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=?`,
		result, newStatus, strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	out := s.loadWeighTicket(id)
	// fail: archive only, no stock-in
	out["stocked_in"] = false
	if result == "pass" && boolOr(body["auto_stock_in"], true) {
		if ok, msg := s.doWeighStockIn(id); !ok {
			api.FailJSON(c, msg)
			return true
		}
		out = s.loadWeighTicket(id)
		out["stocked_in"] = true
	}
	api.OK(c, out)
	return true
}

func (s *Services) stockInWeighTicket(c *gin.Context) bool {
	id := paramID(c)
	var status, qc string
	if err := s.DB.QueryRow(`SELECT status, COALESCE(qc_result,'') FROM pur_weigh_ticket WHERE id=?`, id).Scan(&status, &qc); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if qc != "pass" && status != "qc_pass" {
		api.FailJSON(c, "QC_PASS_REQUIRED")
		return true
	}
	if status == "stocked" {
		api.OK(c, s.loadWeighTicket(id))
		return true
	}
	if ok, msg := s.doWeighStockIn(id); !ok {
		api.FailJSON(c, msg)
		return true
	}
	api.OK(c, s.loadWeighTicket(id))
	return true
}

func (s *Services) doWeighStockIn(id int64) (bool, string) {
	m := s.loadWeighTicket(id)
	if m["id"] == nil {
		return false, "NOT_FOUND"
	}
	if strOr(m["status"]) == "stocked" {
		return true, ""
	}
	farmerID, _ := asInt64(m["farmer_id"])
	productID, _ := asInt64(m["product_id"])
	net, _ := asFloat(m["net_weight"])
	trace := strOr(m["trace_code"])
	origin := strOr(m["origin"])
	bizDate := strOr(m["biz_date"])
	sourceType := strOrDef(m["source_type"], "self")
	wh := int64(1) // 保鲜库 / 原料仓
	if sourceType == "outsource" {
		wh = 2 // 外购半成品入半成品库
	}
	boxCode := fmt.Sprintf("BX-%s", trace)
	if len(boxCode) > 64 {
		boxCode = fmt.Sprintf("BX%d", time.Now().UnixNano()%1e12)
	}
	_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, origin, receive_date, source_type, status)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,'open')`,
		boxCode, productID, wh, bizDate, net, net, farmerID, trace, origin, bizDate, sourceType)
	if err != nil {
		// unique conflict: append suffix
		boxCode = fmt.Sprintf("BX%d", time.Now().UnixNano()%1e12)
		_, err = s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, origin, receive_date, source_type, status)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,'open')`,
			boxCode, productID, wh, bizDate, net, net, farmerID, trace, origin, bizDate, sourceType)
		if err != nil {
			return false, "BOX_CREATE_ERROR:" + err.Error()
		}
	}
	txnNo := fmt.Sprintf("ST-WT-%d", id)
	docType := "purchase_in"
	if sourceType == "outsource" {
		docType = "outsource_in"
	}
	tres, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'draft',?,?)`,
		txnNo, docType, bizDate, wh, fmt.Sprintf("weigh ticket #%d farmer=%d", id, farmerID))
	if err != nil {
		return false, "STOCK_TXN_ERROR:" + err.Error()
	}
	tid, _ := tres.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'in',?)`,
		tid, 1, productID, net, net, bizDate)
	if err := s.adjustBalanceBatch(wh, productID, bizDate, net); err != nil {
		return false, "BALANCE_ERROR:" + err.Error()
	}
	_, _ = s.DB.Exec(`UPDATE inv_stock_txn SET status='posted' WHERE id=?`, tid)
	_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET status='stocked', box_code=?, warehouse_id=?, updated_at=datetime('now') WHERE id=?`,
		boxCode, wh, id)
	// settlement basis
	_, _ = s.DB.Exec(`INSERT INTO pur_farmer_settlement(doc_no, farmer_id, weigh_ticket_id, biz_date, net_weight, amount, status, remark)
		VALUES(?,?,?,?,?,0,'open',?)`,
		fmt.Sprintf("FS%s", time.Now().Format("20060102150405")), farmerID, id, bizDate, net, "auto from weigh net_weight")
	return true, ""
}

func (s *Services) loadWeighTicket(id int64) gin.H {
	var farmerID, productID, warehouseID int64
	var docNo, channel, template, variety, qc, status, trace, origin, bizDate, source, image, box, remark, created string
	var gross, deductRate, deductWeight, net float64
	var farmerName string
	err := s.DB.QueryRow(`SELECT w.doc_no, w.farmer_id, COALESCE(f.name,''), w.channel, COALESCE(w.ticket_template,''), w.product_id, w.variety,
		w.gross_weight, w.deduct_rate, w.deduct_weight, w.net_weight, COALESCE(w.qc_result,''), w.status,
		COALESCE(w.trace_code,''), COALESCE(w.origin,''), w.biz_date, COALESCE(w.source_type,'self'),
		COALESCE(w.image_url,''), COALESCE(w.box_code,''), COALESCE(w.warehouse_id,0), COALESCE(w.remark,''), w.created_at
		FROM pur_weigh_ticket w LEFT JOIN pur_farmer f ON f.id=w.farmer_id WHERE w.id=?`, id).
		Scan(&docNo, &farmerID, &farmerName, &channel, &template, &productID, &variety, &gross, &deductRate, &deductWeight, &net,
			&qc, &status, &trace, &origin, &bizDate, &source, &image, &box, &warehouseID, &remark, &created)
	if err != nil {
		return gin.H{}
	}
	return gin.H{
		"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName, "channel": channel,
		"ticket_template": template, "product_id": productID, "variety": variety,
		"gross_weight": gross, "deduct_rate": deductRate, "deduct_weight": deductWeight, "net_weight": net,
		"qc_result": qc, "status": status, "trace_code": trace, "origin": origin, "biz_date": bizDate,
		"source_type": source, "image_url": image, "box_code": box, "warehouse_id": warehouseID,
		"remark": remark, "created_at": created,
	}
}

func boolOr(v interface{}, def bool) bool {
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true") || t == "yes"
	case float64:
		return t != 0
	}
	return def
}

func (s *Services) handleFarmerSettlements(c *gin.Context, method, action string) bool {
	if action == "list" || method == "GET" {
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_farmer_settlement`).Scan(&total)
		rows, err := s.DB.Query(`SELECT s.id, s.doc_no, s.farmer_id, COALESCE(f.name,''), s.weigh_ticket_id, s.biz_date,
			s.net_weight, s.unit_price, s.amount, s.status, COALESCE(s.remark,''), s.created_at
			FROM pur_farmer_settlement s LEFT JOIN pur_farmer f ON f.id=s.farmer_id
			ORDER BY s.id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, farmerID, wtID int64
			var docNo, farmerName, bizDate, status, remark, created string
			var net, price, amount float64
			_ = rows.Scan(&id, &docNo, &farmerID, &farmerName, &wtID, &bizDate, &net, &price, &amount, &status, &remark, &created)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName,
				"weigh_ticket_id": wtID, "biz_date": bizDate, "net_weight": net, "unit_price": price,
				"amount": amount, "status": status, "remark": remark, "created_at": created,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	}
	if action == "create" || method == "POST" {
		body := bindBody(c)
		id, _ := asInt64(body["id"])
		if id == 0 {
			id, _ = asInt64(body["settlement_id"])
		}
		price, _ := asFloat(body["unit_price"])
		if id > 0 && price >= 0 {
			var net float64
			_ = s.DB.QueryRow(`SELECT net_weight FROM pur_farmer_settlement WHERE id=?`, id).Scan(&net)
			amount := net * price
			_, _ = s.DB.Exec(`UPDATE pur_farmer_settlement SET unit_price=?, amount=?, status=COALESCE(NULLIF(?,''),status), updated_at=datetime('now') WHERE id=?`,
				price, amount, strOr(body["status"]), id)
			api.OK(c, gin.H{"id": id, "unit_price": price, "amount": amount, "net_weight": net})
			return true
		}
		api.FailJSON(c, "SETTLEMENT_ID_REQUIRED")
		return true
	}
	return false
}

func (s *Services) handleTraceLot(c *gin.Context) bool {
	code := c.Param("code")
	if code == "" {
		code = c.Param("trace_code")
	}
	if code == "" {
		api.FailJSON(c, "CODE_REQUIRED")
		return true
	}
	var boxID, farmerID, productID, warehouseID int64
	var boxCode, batch, status, trace, origin, receiveDate, sourceType string
	var qty, weight float64
	err := s.DB.QueryRow(`SELECT id, code, product_id, COALESCE(warehouse_id,0), COALESCE(batch_no,''), qty, COALESCE(weight,0),
		status, COALESCE(farmer_id,0), COALESCE(trace_code,''), COALESCE(origin,''), COALESCE(receive_date,''), COALESCE(source_type,'')
		FROM inv_box_code WHERE (code=? OR trace_code=?) AND COALESCE(is_deleted,0)=0 LIMIT 1`, code, code).
		Scan(&boxID, &boxCode, &productID, &warehouseID, &batch, &qty, &weight, &status, &farmerID, &trace, &origin, &receiveDate, &sourceType)
	if err != nil {
		// try weigh ticket
		m := gin.H{}
		var wtID, fID int64
		var docNo, farmerName, bizDate, wtTrace, wtOrigin, wtStatus, box string
		var net float64
		err2 := s.DB.QueryRow(`SELECT w.id, w.doc_no, w.farmer_id, COALESCE(f.name,''), w.biz_date, w.net_weight,
			COALESCE(w.trace_code,''), COALESCE(w.origin,''), w.status, COALESCE(w.box_code,'')
			FROM pur_weigh_ticket w LEFT JOIN pur_farmer f ON f.id=w.farmer_id
			WHERE w.trace_code=? OR w.box_code=? OR w.doc_no=? LIMIT 1`, code, code, code).
			Scan(&wtID, &docNo, &fID, &farmerName, &bizDate, &net, &wtTrace, &wtOrigin, &wtStatus, &box)
		if err2 != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m["weigh_ticket"] = gin.H{
			"id": wtID, "doc_no": docNo, "farmer_id": fID, "farmer_name": farmerName, "biz_date": bizDate,
			"net_weight": net, "trace_code": wtTrace, "origin": wtOrigin, "status": wtStatus, "box_code": box,
		}
		m["farmer"] = s.loadFarmer(fID)
		m["trace_code"] = wtTrace
		api.OK(c, m)
		return true
	}
	farmer := s.loadFarmer(farmerID)
	family := s.collectBoxFamily(boxCode)
	events := []gin.H{}
	if len(family) > 0 {
		// reuse light query
		for _, bc := range family {
			rows, err := s.DB.Query(`SELECT id, source_type, source_id, trigger_action, status, created_at FROM pd_flow_event
				WHERE payload_json LIKE ? ORDER BY id`, "%"+bc+"%")
			if err != nil {
				continue
			}
			for rows.Next() {
				var id, sid int64
				var stype, trigger, st, created string
				_ = rows.Scan(&id, &stype, &sid, &trigger, &st, &created)
				events = append(events, gin.H{"id": id, "source_type": stype, "source_id": sid, "trigger": trigger, "status": st, "created_at": created, "box_code": bc})
			}
			rows.Close()
		}
	}
	var wt gin.H
	var wtID int64
	_ = s.DB.QueryRow(`SELECT id FROM pur_weigh_ticket WHERE trace_code=? OR box_code=? LIMIT 1`, trace, boxCode).Scan(&wtID)
	if wtID > 0 {
		wt = s.loadWeighTicket(wtID)
	}
	api.OK(c, gin.H{
		"trace_code": trace, "box_code": boxCode, "related_boxes": family,
		"box": gin.H{
			"id": boxID, "code": boxCode, "product_id": productID, "warehouse_id": warehouseID,
			"batch_no": batch, "qty": qty, "weight": weight, "status": status,
			"farmer_id": farmerID, "origin": origin, "receive_date": receiveDate, "source_type": sourceType,
		},
		"farmer": farmer, "weigh_ticket": wt, "flow_events": events,
	})
	return true
}
