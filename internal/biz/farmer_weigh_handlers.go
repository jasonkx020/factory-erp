package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/notify"
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
		return s.refuseDelete(c)
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
	price, _ := asFloat(body["default_unit_price"])
	res, err := s.DB.Exec(`INSERT INTO pur_farmer(code, name, mobile, origin, trace_code, trace_code_prefix, status, remark, default_unit_price)
		VALUES(?,?,?,?,?,?,?,?,?)`, code, name, mobile, origin, trace, prefix, status, remark, price)
	if err != nil {
		// fallback without price column
		res, err = s.DB.Exec(`INSERT INTO pur_farmer(code, name, mobile, origin, trace_code, trace_code_prefix, status, remark)
			VALUES(?,?,?,?,?,?,?,?)`, code, name, mobile, origin, trace, prefix, status, remark)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
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
	path := c.Request.URL.Path
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
		m["evidences"] = s.listEvidence("weigh_ticket", paramID(c))
		api.OK(c, m)
		return true
	case strings.Contains(path, "/warehouse-confirm") || strings.HasSuffix(action, "warehouse-confirm"):
		return s.stockInWeighTicket(c)
	case strings.Contains(path, "/label") || strings.HasSuffix(action, "label"):
		return s.labelWeighTicket(c)
	case (strings.Contains(path, "/confirm") && !strings.Contains(path, "warehouse-confirm")) || action == "action:confirm":
		return s.confirmWeighTicket(c)
	case strings.HasSuffix(action, "qc") || (method == "POST" && strings.Contains(path, "/qc")):
		return s.qcWeighTicket(c)
	case strings.HasSuffix(action, "stock-in") || (method == "POST" && strings.Contains(path, "/stock-in")):
		return s.stockInWeighTicket(c)
	case action == "update" || action == "replace":
		return s.updateWeighTicket(c)
	case action == "delete":
		return s.refuseDelete(c)
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
	arrivalID, _ := asInt64(body["arrival_id"])
	var farmerID int64
	var grade, origin, sourceType, channel, variety, bizDate string
	if arrivalID > 0 {
		var status, qc string
		err := s.DB.QueryRow(`SELECT farmer_id, status, COALESCE(qc_result,''), COALESCE(grade,''), COALESCE(origin,''),
			source_type, channel, COALESCE(variety,''), biz_date FROM pur_inbound_arrival WHERE id=? AND COALESCE(is_deleted,0)=0`, arrivalID).
			Scan(&farmerID, &status, &qc, &grade, &origin, &sourceType, &channel, &variety, &bizDate)
		if err != nil {
			api.FailJSON(c, "ARRIVAL_NOT_FOUND")
			return true
		}
		if status != "qc_pass" || qc != "pass" {
			api.FailJSON(c, "QC_REQUIRED")
			return true
		}
		if grade == "" {
			api.FailJSON(c, "GRADE_REQUIRED")
			return true
		}
	} else {
		// legacy path: still require image; QC must be done before confirm (not stock-in)
		farmerID, _ = asInt64(body["farmer_id"])
		if farmerID <= 0 {
			api.FailJSON(c, "FARMER_REQUIRED")
			return true
		}
		grade = strings.ToUpper(strOr(body["grade"]))
		origin = strOr(body["origin"])
		sourceType = strOrDef(body["source_type"], "self")
		channel = strOrDef(body["channel"], "internal")
		variety = strOrDef(body["variety"], "鲜木薯")
		bizDate = strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	}
	var farmerName, farmerOrigin string
	err := s.DB.QueryRow(`SELECT name, COALESCE(origin,'') FROM pur_farmer WHERE id=? AND status='active' AND COALESCE(is_deleted,0)=0`, farmerID).
		Scan(&farmerName, &farmerOrigin)
	if err != nil {
		api.FailJSON(c, "FARMER_NOT_FOUND")
		return true
	}
	if origin == "" {
		origin = farmerOrigin
	}
	if channel != "external" && channel != "internal" {
		channel = "internal"
	}
	productID, _ := asInt64(body["product_id"])
	if productID <= 0 {
		productID = 1
	}
	if variety == "" {
		variety = strOrDef(body["variety"], "鲜木薯")
	}
	gross, _ := asFloat(body["gross_weight"])
	if gross <= 0 {
		api.FailJSON(c, "GROSS_WEIGHT_REQUIRED")
		return true
	}
	deductRate, hasRate := asFloat(body["deduct_rate"])
	deductWeight, hasDeduct := asFloat(body["deduct_weight"])
	if hasRate && deductRate > 0 {
		if deductRate > 1 {
			deductRate = deductRate / 100
		}
		deductWeight = gross * deductRate
	} else if hasDeduct && deductWeight > 0 && gross > 0 {
		deductRate = deductWeight / gross
	}
	net, hasNet := asFloat(body["net_weight"])
	if !hasNet || net <= 0 {
		net = gross - deductWeight
	}
	if net < 0 {
		net = 0
	}
	if bizDate == "" {
		bizDate = strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	}
	imageURL := strOr(body["image_url"])
	if imageURL == "" {
		api.FailJSON(c, "EVIDENCE_INCOMPLETE:weigh_photo")
		return true
	}
	// OCR draft optional
	ocrDraft := "{}"
	if body["ocr_draft"] != nil {
		b, _ := json.Marshal(body["ocr_draft"])
		ocrDraft = string(b)
	} else if s := strOr(body["ocr_draft_json"]); s != "" {
		ocrDraft = s
	}
	template := strOrDef(body["ticket_template"], channel)
	docNo := fmt.Sprintf("WT%s", time.Now().Format("20060102150405"))
	qcResult := ""
	status := "draft"
	if arrivalID > 0 {
		qcResult = "pass"
		status = "pending_confirm"
	}
	freight, loading, weighFee, passRate, reject, plate, recvAddr := feeFieldsFromBody(body)
	res, err := s.DB.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, channel, ticket_template, product_id, variety,
		gross_weight, deduct_rate, deduct_weight, net_weight, qc_result, status, trace_code, origin, biz_date,
		source_type, image_url, remark, arrival_id, grade, ocr_draft_json,
		plate_no, receive_address, pass_rate, reject_weight, freight_fee, loading_fee, weigh_fee)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		docNo, farmerID, channel, template, productID, variety, gross, deductRate, deductWeight, net,
		qcResult, status, "",
		origin, bizDate, sourceType, imageURL, strOr(body["remark"]), nullIf0(arrivalID), grade, ocrDraft,
		plate, recvAddr, passRate, reject, freight, loading, weighFee)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	_, _ = s.addEvidence(c, "weigh_ticket", id, "weigh_photo", imageURL, nil)
	s.writeAuditCtx(c, "weigh_ticket", id, "create", "weigh_draft", nil, gin.H{"net_weight": net, "arrival_id": arrivalID})
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
		plate_no=COALESCE(NULLIF(?,''),plate_no), receive_address=COALESCE(NULLIF(?,''),receive_address),
		pass_rate=COALESCE(NULLIF(?,0),pass_rate), reject_weight=COALESCE(NULLIF(?,0),reject_weight),
		freight_fee=COALESCE(NULLIF(?,0),freight_fee), loading_fee=COALESCE(NULLIF(?,0),loading_fee),
		weigh_fee=COALESCE(NULLIF(?,0),weigh_fee),
		updated_at=datetime('now') WHERE id=?`,
		gross, deductRate, deductWeight, net, strOr(body["variety"]), strOr(body["image_url"]), strOr(body["remark"]),
		strOr(body["plate_no"]), strOr(body["receive_address"]),
		asFloatOr0(body["pass_rate"]), asFloatOr0(body["reject_weight"]),
		asFloatOr0(body["freight_fee"]), asFloatOr0(body["loading_fee"]), asFloatOr0(body["weigh_fee"]), id)
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
		grade := strings.ToUpper(strOr(body["grade"]))
		if grade == "" {
			api.FailJSON(c, "GRADE_REQUIRED")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET grade=? WHERE id=?`, grade, id)
	}
	_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET qc_result=?, status=?, remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=?`,
		result, newStatus, strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	out := s.loadWeighTicket(id)
	out["stocked_in"] = false
	// no auto stock-in — warehouse must confirm after weigh confirm+trace
	s.writeAuditCtx(c, "weigh_ticket", id, "qc", result, nil, out)
	api.OK(c, out)
	return true
}

func (s *Services) stockInWeighTicket(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse") {
		return true
	}
	id := paramID(c)
	var status, qc, trace string
	if err := s.DB.QueryRow(`SELECT status, COALESCE(qc_result,''), COALESCE(trace_code,'') FROM pur_weigh_ticket WHERE id=?`, id).Scan(&status, &qc, &trace); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status == "stocked" {
		api.OK(c, s.loadWeighTicket(id))
		return true
	}
	if status != "weighed" {
		api.FailJSON(c, "WEIGH_CONFIRM_REQUIRED")
		return true
	}
	if qc != "pass" && qc != "" {
		api.FailJSON(c, "QC_PASS_REQUIRED")
		return true
	}
	if trace == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	if ok, msg := s.doWeighStockIn(id); !ok {
		api.FailJSON(c, msg)
		return true
	}
	_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET purchase_completed_at=datetime('now') WHERE id=?`, id)
	_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET status='stocked', updated_at=datetime('now') WHERE id=(SELECT arrival_id FROM pur_weigh_ticket WHERE id=?)`, id)
	s.writeAuditCtx(c, "weigh_ticket", id, "warehouse_confirm", "stock_in", nil, s.loadWeighTicket(id))
	if s.Notify != nil {
		s.Notify.CompleteTask("weigh_ticket", id, "purchase.weigh_confirmed")
		m := s.loadWeighTicket(id)
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.stocked", BizType: "weigh_ticket", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
			FromRole: "warehouse", ToRoles: []string{"finance"}, CreateTask: true,
			Payload: gin.H{"box_code": m["box_code"], "net_weight": m["net_weight"], "farmer_name": m["farmer_name"]},
		})
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.stocked", BizType: "weigh_ticket", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
			FromRole: "warehouse", ToRoles: []string{"purchase"}, CreateTask: false,
			Title: "入库完成", Body: "仓管已确认入库 " + strOr(m["doc_no"]),
			Payload: gin.H{"box_code": m["box_code"]},
		})
	}
	api.OK(c, s.loadWeighTicket(id))
	return true
}

func (s *Services) confirmWeighTicket(c *gin.Context) bool {
	if !s.requireAnyRole(c, "purchase") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	m := s.loadWeighTicket(id)
	if m["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	status := strOr(m["status"])
	if status != "draft" && status != "pending_confirm" && status != "qc_pass" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	if status == "draft" {
		api.FailJSON(c, "QC_PASS_REQUIRED")
		return true
	}
	if err := s.requireEvidence("weigh_ticket", id, "weigh_photo"); err != nil {
		api.FailJSON(c, err.Error())
		return true
	}
	// allow user to confirm adjusted weights
	gross, _ := asFloat(m["gross_weight"])
	deductRate, _ := asFloat(m["deduct_rate"])
	deductWeight, _ := asFloat(m["deduct_weight"])
	net, _ := asFloat(m["net_weight"])
	if g, ok := asFloat(body["gross_weight"]); ok && g > 0 {
		gross = g
	}
	if d, ok := asFloat(body["deduct_rate"]); ok {
		deductRate = d
		if deductRate > 1 {
			deductRate = deductRate / 100
		}
		deductWeight = gross * deductRate
		net = gross - deductWeight
	}
	if n, ok := asFloat(body["net_weight"]); ok && n > 0 {
		net = n
	}
	grade := strOrDef(body["grade"], strOr(m["grade"]))
	if grade == "" {
		api.FailJSON(c, "GRADE_REQUIRED")
		return true
	}
	bizDate := strOr(m["biz_date"])
	batch, err := NextBatchNo(s, bizDate)
	if err != nil {
		batch = fmt.Sprintf("%06d", time.Now().Unix()%1e6)
	}
	farmerID, _ := asInt64(m["farmer_id"])
	arrivalID, _ := asInt64(m["arrival_id"])
	in := TraceIssueInput{
		BizDate: bizDate, BatchNo: batch, FarmerID: farmerID, Grade: grade,
		Channel: strOr(m["channel"]), SourceType: strOr(m["source_type"]), NetKg: net, ArrivalID: arrivalID,
	}
	secret := TraceHMACSecret(s.TraceHMACSecret)
	trace, canonical, sig := IssueTraceCode(secret, in)
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	snap := nowSnap(map[string]interface{}{
		"gross_weight": gross, "deduct_rate": deductRate, "deduct_weight": deductWeight, "net_weight": net, "grade": grade, "trace_code": trace,
	})
	_, err = s.DB.Exec(`UPDATE pur_weigh_ticket SET gross_weight=?, deduct_rate=?, deduct_weight=?, net_weight=?, grade=?,
		batch_no=?, trace_code=?, status='weighed', confirmed_by=?, confirmed_at=datetime('now'), confirmed_snapshot_json=?,
		updated_at=datetime('now') WHERE id=?`,
		gross, deductRate, deductWeight, net, grade, batch, trace, uid, snap, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	_, _ = s.DB.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, arrival_id, weigh_ticket_id, channel, source_type, net_weight, payload_canonical, signature, status)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'open')`,
		trace, normalizeBizDate(bizDate), batch, farmerID, grade, nullIf0(arrivalID), id, strOr(m["channel"]), strOr(m["source_type"]), net, canonical, sig)
	if arrivalID > 0 {
		_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET status='weighed', updated_at=datetime('now') WHERE id=?`, arrivalID)
	}
	s.writeAuditCtx(c, "weigh_ticket", id, "confirm_issue_trace", "user_confirmed", m, s.loadWeighTicket(id))
	out := s.loadWeighTicket(id)
	out["label"] = s.buildLabel(out)
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.weigh_confirmed", BizType: "weigh_ticket", BizID: id,
			DocNo: strOr(out["doc_no"]), TraceCode: strOr(out["trace_code"]),
			FromRole: "purchase", ToRoles: []string{"warehouse"}, CreateTask: true,
			Payload: gin.H{
				"net_weight": out["net_weight"], "grade": out["grade"], "farmer_name": out["farmer_name"],
				"batch_no": out["batch_no"], "biz_date": out["biz_date"],
				"plate_no": out["plate_no"], "receive_address": out["receive_address"],
				"pass_rate": out["pass_rate"], "reject_weight": out["reject_weight"],
				"freight_fee": out["freight_fee"], "loading_fee": out["loading_fee"], "weigh_fee": out["weigh_fee"],
				"gross_weight": out["gross_weight"], "deduct_weight": out["deduct_weight"],
			},
		})
	}
	api.OK(c, out)
	return true
}

func (s *Services) labelWeighTicket(c *gin.Context) bool {
	m := s.loadWeighTicket(paramID(c))
	if m["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["trace_code"]) == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}
	api.OK(c, s.buildLabel(m))
	return true
}

func (s *Services) buildLabel(m gin.H) gin.H {
	code := strOr(m["trace_code"])
	return gin.H{
		"trace_code": code,
		"qr_content": code,
		"farmer_name": m["farmer_name"],
		"origin": m["origin"],
		"grade": m["grade"],
		"net_weight": m["net_weight"],
		"biz_date": m["biz_date"],
		"batch_no": m["batch_no"],
		"print_lines": []string{
			fmt.Sprintf("溯源码 %s", code),
			fmt.Sprintf("农户 %v  等级 %v", m["farmer_name"], m["grade"]),
			fmt.Sprintf("净重 %.2f kg  日期 %v", asFloatDef(m["net_weight"]), m["biz_date"]),
			fmt.Sprintf("批次 %v  产地 %v", m["batch_no"], m["origin"]),
		},
	}
}

func asFloatDef(v interface{}) float64 {
	f, _ := asFloat(v)
	return f
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
	_, _ = s.DB.Exec(`UPDATE inv_stock_txn SET status='posted', posted_at=datetime('now') WHERE id=?`, tid)
	_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET status='stocked', box_code=?, warehouse_id=?, updated_at=datetime('now') WHERE id=?`,
		boxCode, wh, id)
	// settlement basis with grade/default price
	var unitPrice float64
	grade := strOr(m["grade"])
	_ = s.DB.QueryRow(`SELECT unit_price FROM pur_grade_price WHERE grade=? AND status='active'`, grade).Scan(&unitPrice)
	if unitPrice <= 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(default_unit_price,0) FROM pur_farmer WHERE id=?`, farmerID).Scan(&unitPrice)
	}
	freight := 0.0
	loading := 0.0
	weighFee := 0.0
	if v, ok := asFloat(m["freight_fee"]); ok {
		freight = v
	}
	if v, ok := asFloat(m["loading_fee"]); ok {
		loading = v
	}
	if v, ok := asFloat(m["weigh_fee"]); ok {
		weighFee = v
	}
	goods, total := settleAmount(net, unitPrice, freight, loading, weighFee)
	amount := total
	_, _ = s.DB.Exec(`INSERT INTO pur_farmer_settlement(doc_no, farmer_id, weigh_ticket_id, biz_date, net_weight, unit_price, amount, status, remark,
		freight_fee, loading_fee, weigh_fee, goods_amount)
		VALUES(?,?,?,?,?,?,?,'settle_pending',?,?,?,?,?)`,
		fmt.Sprintf("FS%s", time.Now().Format("20060102150405")), farmerID, id, bizDate, net, unitPrice, amount, "auto from weigh net_weight",
		freight, loading, weighFee, goods)
	_, _ = s.DB.Exec(`UPDATE pur_trace_lot SET status='stocked' WHERE weigh_ticket_id=?`, id)
	return true, ""
}

func (s *Services) loadWeighTicket(id int64) gin.H {
	var farmerID, productID, warehouseID, arrivalID int64
	var docNo, channel, template, variety, qc, status, trace, origin, bizDate, source, image, box, remark, created, grade, batch string
	var plate, recvAddr string
	var gross, deductRate, deductWeight, net, passRate, reject, freight, loading, weighFee float64
	var farmerName string
	err := s.DB.QueryRow(`SELECT w.doc_no, w.farmer_id, COALESCE(f.name,''), w.channel, COALESCE(w.ticket_template,''), w.product_id, w.variety,
		w.gross_weight, w.deduct_rate, w.deduct_weight, w.net_weight, COALESCE(w.qc_result,''), w.status,
		COALESCE(w.trace_code,''), COALESCE(w.origin,''), w.biz_date, COALESCE(w.source_type,'self'),
		COALESCE(w.image_url,''), COALESCE(w.box_code,''), COALESCE(w.warehouse_id,0), COALESCE(w.remark,''), w.created_at,
		COALESCE(w.arrival_id,0), COALESCE(w.grade,''), COALESCE(w.batch_no,''),
		COALESCE(w.plate_no,''), COALESCE(w.receive_address,''), COALESCE(w.pass_rate,0), COALESCE(w.reject_weight,0),
		COALESCE(w.freight_fee,0), COALESCE(w.loading_fee,0), COALESCE(w.weigh_fee,0)
		FROM pur_weigh_ticket w LEFT JOIN pur_farmer f ON f.id=w.farmer_id WHERE w.id=?`, id).
		Scan(&docNo, &farmerID, &farmerName, &channel, &template, &productID, &variety, &gross, &deductRate, &deductWeight, &net,
			&qc, &status, &trace, &origin, &bizDate, &source, &image, &box, &warehouseID, &remark, &created,
			&arrivalID, &grade, &batch, &plate, &recvAddr, &passRate, &reject, &freight, &loading, &weighFee)
	if err != nil {
		return gin.H{}
	}
	out := gin.H{
		"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName, "channel": channel,
		"ticket_template": template, "product_id": productID, "variety": variety,
		"gross_weight": gross, "deduct_rate": deductRate, "deduct_weight": deductWeight, "net_weight": net,
		"qc_result": qc, "status": status, "trace_code": trace, "origin": origin, "biz_date": bizDate,
		"source_type": source, "image_url": image, "box_code": box, "warehouse_id": warehouseID,
		"remark": remark, "created_at": created, "arrival_id": arrivalID, "grade": grade, "batch_no": batch,
		"evidences": s.listEvidence("weigh_ticket", id),
	}
	for k, v := range feeMap(freight, loading, weighFee, passRate, reject, plate, recvAddr) {
		out[k] = v
	}
	return out
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
	path := c.Request.URL.Path
	if strings.Contains(path, "/pay") || strings.HasSuffix(action, "pay") {
		return s.payFarmerSettlement(c)
	}
	if strings.Contains(path, "/summary") || action == "summary" {
		return s.summaryFarmerSettlements(c)
	}
	if action == "delete" {
		return s.refuseDelete(c)
	}
	if action == "list" || method == "GET" {
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_farmer_settlement`).Scan(&total)
		rows, err := s.DB.Query(`SELECT s.id, s.doc_no, s.farmer_id, COALESCE(f.name,''), s.weigh_ticket_id, s.biz_date,
			s.net_weight, s.unit_price, s.amount, s.status, COALESCE(s.remark,''), s.created_at,
			COALESCE(s.transfer_no,''), COALESCE(s.paid_at,''), COALESCE(s.pay_evidence_url,''),
			COALESCE(s.freight_fee,0), COALESCE(s.loading_fee,0), COALESCE(s.weigh_fee,0), COALESCE(s.goods_amount,0)
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
			var docNo, fname, bizDate, status, remark, created, transfer, paidAt, payURL string
			var net, price, amount, freight, loading, weighFee, goods float64
			_ = rows.Scan(&id, &docNo, &farmerID, &fname, &wtID, &bizDate, &net, &price, &amount, &status, &remark, &created, &transfer, &paidAt, &payURL,
				&freight, &loading, &weighFee, &goods)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": fname, "weigh_ticket_id": wtID,
				"biz_date": bizDate, "net_weight": net, "unit_price": price, "amount": amount, "status": status,
				"remark": remark, "created_at": created, "transfer_no": transfer, "paid_at": paidAt, "pay_evidence_url": payURL,
				"freight_fee": freight, "loading_fee": loading, "weigh_fee": weighFee, "goods_amount": goods,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	}
	if action == "create" || method == "POST" {
		body := bindBody(c)
		wtID, _ := asInt64(body["weigh_ticket_id"])
		unitPrice, _ := asFloat(body["unit_price"])
		var farmerID int64
		var net float64
		var bizDate string
		if wtID > 0 {
			_ = s.DB.QueryRow(`SELECT farmer_id, net_weight, biz_date FROM pur_weigh_ticket WHERE id=?`, wtID).Scan(&farmerID, &net, &bizDate)
		}
		if farmerID == 0 {
			farmerID, _ = asInt64(body["farmer_id"])
			net, _ = asFloat(body["net_weight"])
			bizDate = strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		}
		amount := net * unitPrice
		freight, loading, weighFee, _, _, _, _ := feeFieldsFromBody(body)
		goods, total := settleAmount(net, unitPrice, freight, loading, weighFee)
		amount = total
		docNo := fmt.Sprintf("FS%s", time.Now().Format("20060102150405"))
		res, err := s.DB.Exec(`INSERT INTO pur_farmer_settlement(doc_no, farmer_id, weigh_ticket_id, biz_date, net_weight, unit_price, amount, status, remark,
			freight_fee, loading_fee, weigh_fee, goods_amount)
			VALUES(?,?,?,?,?,?,?,'settle_pending',?,?,?,?,?)`, docNo, farmerID, nullIf0(wtID), bizDate, net, unitPrice, amount, strOr(body["remark"]),
			freight, loading, weighFee, goods)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "amount": amount, "status": "settle_pending"})
		return true
	}
	return false
}

func (s *Services) payFarmerSettlement(c *gin.Context) bool {
	if !s.requireAnyRole(c, "finance") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	transferNo := strOr(body["transfer_no"])
	payURL := strOrDef(body["pay_evidence_url"], strOr(body["image_url"]))
	if transferNo == "" {
		api.FailJSON(c, "TRANSFER_NO_REQUIRED")
		return true
	}
	if payURL == "" {
		api.FailJSON(c, "EVIDENCE_INCOMPLETE:pay_receipt")
		return true
	}
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM pur_farmer_settlement WHERE id=?`, id).Scan(&status); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status == "settle_paid" {
		api.FailJSON(c, "ALREADY_PAID")
		return true
	}
	_, _ = s.addEvidence(c, "farmer_settlement", id, "pay_receipt", payURL, gin.H{"transfer_no": transferNo})
	_, err := s.DB.Exec(`UPDATE pur_farmer_settlement SET status='settle_paid', transfer_no=?, paid_at=datetime('now'), pay_evidence_url=?,
		unit_price=COALESCE(NULLIF(?,0),unit_price), amount=COALESCE(NULLIF(?,0),amount), remark=COALESCE(NULLIF(?,''),remark)
		WHERE id=?`, transferNo, payURL, asFloatOr0(body["unit_price"]), asFloatOr0(body["amount"]), strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.writeAuditCtx(c, "farmer_settlement", id, "pay", transferNo, nil, gin.H{"transfer_no": transferNo})
	if s.Notify != nil {
		s.Notify.CompleteTask("farmer_settlement", id)
		m := s.loadSettlement(id)
		wtID, _ := asInt64(m["weigh_ticket_id"])
		if wtID > 0 {
			s.Notify.CompleteTask("weigh_ticket", wtID, "purchase.stocked")
		}
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.settle_paid", BizType: "farmer_settlement", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: "",
			FromRole: "finance", ToRoles: []string{"purchase"}, CreateTask: false,
			Payload: gin.H{"transfer_no": transferNo, "amount": m["amount"]},
		})
	}
	api.OK(c, gin.H{"id": id, "status": "settle_paid", "transfer_no": transferNo})
	return true
}

func (s *Services) summaryFarmerSettlements(c *gin.Context) bool {
	bizDate := c.Query("biz_date")
	q := `SELECT farmer_id, COALESCE(f.name,''), SUM(net_weight), SUM(amount), COUNT(1)
		FROM pur_farmer_settlement s LEFT JOIN pur_farmer f ON f.id=s.farmer_id WHERE 1=1`
	args := []interface{}{}
	if bizDate != "" {
		q += ` AND s.biz_date=?`
		args = append(args, bizDate)
	}
	q += ` GROUP BY farmer_id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var fid, cnt int64
		var name string
		var net, amt float64
		_ = rows.Scan(&fid, &name, &net, &amt, &cnt)
		list = append(list, gin.H{"farmer_id": fid, "farmer_name": name, "net_weight": net, "amount": amt, "count": cnt})
	}
	api.OK(c, gin.H{"list": list})
	return true
}

func asFloatOr0(v interface{}) float64 {
	f, _ := asFloat(v)
	return f
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
	return s.traceTimeline(c, code)
}

func (s *Services) verifyTraceLot(c *gin.Context) bool {
	body := bindBody(c)
	code := strOr(body["trace_code"])
	canonical := strOr(body["payload_canonical"])
	sig := strOr(body["signature"])
	if code != "" {
		var storedCanon, storedSig string
		err := s.DB.QueryRow(`SELECT payload_canonical, signature FROM pur_trace_lot WHERE trace_code=?`, code).Scan(&storedCanon, &storedSig)
		if err == nil {
			canonical = storedCanon
			sig = storedSig
		} else if ok, s2 := ParseTraceSig(code); ok {
			sig = s2
		}
	}
	secrets := []string{TraceHMACSecret(s.TraceHMACSecret)}
	valid := canonical != "" && sig != "" && VerifyCanonical(secrets, canonical, sig)
	api.OK(c, gin.H{
		"valid": valid, "trace_code": code, "payload_canonical": canonical, "signature": sig,
		"legacy": !strings.HasPrefix(code, "T1-"),
	})
	return true
}

func (s *Services) traceTimeline(c *gin.Context, code string) bool {
	events := []gin.H{}
	var lot gin.H
	var lotID, farmerID, wtID, arrivalID int64
	var trace, bizDate, batch, grade, canon, sig, lotStatus string
	var net float64
	err := s.DB.QueryRow(`SELECT id, trace_code, biz_date, batch_no, farmer_id, COALESCE(grade,''), COALESCE(arrival_id,0), COALESCE(weigh_ticket_id,0),
		net_weight, payload_canonical, signature, status FROM pur_trace_lot WHERE trace_code=?`, code).
		Scan(&lotID, &trace, &bizDate, &batch, &farmerID, &grade, &arrivalID, &wtID, &net, &canon, &sig, &lotStatus)
	if err == nil {
		lot = gin.H{"id": lotID, "trace_code": trace, "biz_date": bizDate, "batch_no": batch, "farmer_id": farmerID,
			"grade": grade, "arrival_id": arrivalID, "weigh_ticket_id": wtID, "net_weight": net,
			"payload_canonical": canon, "signature": sig, "status": lotStatus,
			"signature_valid": VerifyCanonical([]string{TraceHMACSecret(s.TraceHMACSecret)}, canon, sig)}
		events = append(events, gin.H{"step": "trace_lot", "at": bizDate, "data": lot})
	}
	if arrivalID > 0 {
		events = append(events, gin.H{"step": "arrival", "data": s.loadArrival(arrivalID), "evidences": s.listEvidence("inbound_arrival", arrivalID)})
	}
	if wtID > 0 {
		events = append(events, gin.H{"step": "weigh", "data": s.loadWeighTicket(wtID), "evidences": s.listEvidence("weigh_ticket", wtID)})
	} else {
		var wid int64
		_ = s.DB.QueryRow(`SELECT id FROM pur_weigh_ticket WHERE trace_code=? LIMIT 1`, code).Scan(&wid)
		if wid > 0 {
			wtID = wid
			events = append(events, gin.H{"step": "weigh", "data": s.loadWeighTicket(wid), "evidences": s.listEvidence("weigh_ticket", wid)})
		}
	}
	// box
	var boxID int64
	var boxCode string
	_ = s.DB.QueryRow(`SELECT id, code FROM inv_box_code WHERE code=? OR trace_code=? LIMIT 1`, code, code).Scan(&boxID, &boxCode)
	if boxID > 0 {
		events = append(events, gin.H{"step": "box", "box_id": boxID, "box_code": boxCode})
		family := s.collectBoxFamily(boxCode)
		events = append(events, gin.H{"step": "box_family", "related_boxes": family})
	}
	if wtID > 0 {
		rows, _ := s.DB.Query(`SELECT id, doc_no, amount, status, COALESCE(transfer_no,''), COALESCE(paid_at,'') FROM pur_farmer_settlement WHERE weigh_ticket_id=?`, wtID)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var sid int64
				var docNo, st, tn, paid string
				var amt float64
				_ = rows.Scan(&sid, &docNo, &amt, &st, &tn, &paid)
				events = append(events, gin.H{"step": "farmer_settlement", "id": sid, "doc_no": docNo, "amount": amt, "status": st, "transfer_no": tn, "paid_at": paid,
					"evidences": s.listEvidence("farmer_settlement", sid)})
			}
		}
	}
	// audits
	auditRows, _ := s.DB.Query(`SELECT action, reason, created_at, COALESCE(actor_user_id,0) FROM biz_audit_log
		WHERE (biz_type='weigh_ticket' AND biz_id=?) OR (biz_type='inbound_arrival' AND biz_id=?) OR (biz_type='farmer_settlement' AND biz_id IN (SELECT id FROM pur_farmer_settlement WHERE weigh_ticket_id=?))
		ORDER BY id`, wtID, arrivalID, wtID)
	if auditRows != nil {
		defer auditRows.Close()
		for auditRows.Next() {
			var action, reason, at string
			var uid int64
			_ = auditRows.Scan(&action, &reason, &at, &uid)
			events = append(events, gin.H{"step": "audit", "action": action, "reason": reason, "at": at, "actor_user_id": uid})
		}
	}
	api.OK(c, gin.H{"trace_code": code, "lot": lot, "timeline": events})
	return true
}
