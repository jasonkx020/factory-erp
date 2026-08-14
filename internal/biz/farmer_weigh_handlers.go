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
	mobileQ := strings.TrimSpace(c.Query("mobile"))
	nameQ := strings.TrimSpace(c.Query("name"))
	idQ := strings.TrimSpace(c.Query("id"))
	where := `WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	searching := idQ != "" || mobileQ != "" || nameQ != "" || kw != ""
	if searching {
		where += ` AND status='active'`
	}
	if idQ != "" {
		where += ` AND id=?`
		args = append(args, idQ)
	}
	if mobileQ != "" {
		// 手机号：精确或后缀/前缀模糊
		where += ` AND mobile LIKE ?`
		args = append(args, "%"+mobileQ+"%")
	}
	if nameQ != "" {
		where += ` AND name LIKE ?`
		args = append(args, "%"+nameQ+"%")
	}
	if kw != "" {
		where += ` AND (name LIKE ? OR mobile LIKE ? OR code LIKE ? OR origin LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like, like)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_farmer `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(mobile,''), COALESCE(origin,''), COALESCE(trace_code,''),
		COALESCE(trace_code_prefix,''), status, COALESCE(remark,''), created_at, COALESCE(default_unit_price,0)
		FROM pur_farmer `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		// fallback without default_unit_price
		rows, err = s.DB.Query(`SELECT id, code, name, COALESCE(mobile,''), COALESCE(origin,''), COALESCE(trace_code,''),
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
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var code, name, mobile, origin, trace, prefix, status, remark, created string
		var price float64
		_ = rows.Scan(&id, &code, &name, &mobile, &origin, &trace, &prefix, &status, &remark, &created, &price)
		list = append(list, gin.H{
			"id": id, "code": code, "name": name, "mobile": mobile, "origin": origin,
			"trace_code": trace, "trace_code_prefix": prefix, "status": status, "remark": remark, "created_at": created,
			"default_unit_price": price,
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
	var price float64
	err := s.DB.QueryRow(`SELECT code, name, COALESCE(mobile,''), COALESCE(origin,''), COALESCE(trace_code,''),
		COALESCE(trace_code_prefix,''), status, COALESCE(remark,''), created_at, COALESCE(default_unit_price,0)
		FROM pur_farmer WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&code, &name, &mobile, &origin, &trace, &prefix, &status, &remark, &created, &price)
	if err != nil {
		err = s.DB.QueryRow(`SELECT code, name, COALESCE(mobile,''), COALESCE(origin,''), COALESCE(trace_code,''),
			COALESCE(trace_code_prefix,''), status, COALESCE(remark,''), created_at
			FROM pur_farmer WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
			Scan(&code, &name, &mobile, &origin, &trace, &prefix, &status, &remark, &created)
		if err != nil {
			return gin.H{}
		}
		return gin.H{
			"id": id, "code": code, "name": name, "mobile": mobile, "origin": origin,
			"trace_code": trace, "trace_code_prefix": prefix, "status": status, "remark": remark, "created_at": created,
		}
	}
	return gin.H{
		"id": id, "code": code, "name": name, "mobile": mobile, "origin": origin,
		"trace_code": trace, "trace_code_prefix": prefix, "status": status, "remark": remark, "created_at": created,
		"default_unit_price": price,
	}
}

// ---------- weigh tickets ----------

func (s *Services) handleWeighTickets(c *gin.Context, method, action string) bool {
	path := c.Request.URL.Path
	switch {
	case strings.Contains(path, "/by-trace") || action == "by-trace":
		return s.resolveWeighByTraceCode(c)
	case action == "list":
		return s.listWeighTickets(c)
	case action == "create":
		return s.createWeighTicket(c)
	case action == "get":
		id := paramID(c)
		m := s.loadWeighTicket(id)
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m["evidences"] = s.listEvidence("weigh_ticket", id)
		s.attachWeighProcessTrail(m, id)
		s.attachWeighVerifyMedia(m, id, strOr(m["image_url"]))
		if claimsIsWarehouseOnly(middleware.Claims(c)) {
			m = maskWeighTicketForWarehouse(m)
		}
		api.OK(c, m)
		return true
	case strings.Contains(path, "/warehouse-confirm") || strings.HasSuffix(action, "warehouse-confirm"):
		return s.stockInWeighTicket(c)
	case strings.Contains(path, "/box-stock-in") || strings.HasSuffix(action, "box-stock-in"):
		return s.boxStockInWeighTicket(c)
	case strings.Contains(path, "/warehouse-return") || strings.HasSuffix(action, "warehouse-return"):
		return s.warehouseReturnWeighTicket(c)
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
	if pageSize > 100 {
		pageSize = 100
	}
	today := time.Now()
	defFrom := today.AddDate(0, 0, -2).Format("2006-01-02")
	defTo := today.Format("2006-01-02")
	// biz_date 存 YYYY-MM-DD；勿用 normalizeBizDate（会变成 YYYYMMDD，导致解析失败）
	dateFrom := strings.TrimSpace(strOrDef(c.Query("date_from"), defFrom))
	dateTo := strings.TrimSpace(strOrDef(c.Query("date_to"), defTo))
	fromT, errFrom := time.ParseInLocation("2006-01-02", dateFrom, time.Local)
	toT, errTo := time.ParseInLocation("2006-01-02", dateTo, time.Local)
	if errFrom != nil || errTo != nil {
		api.FailJSON(c, "DATE_RANGE_INVALID")
		return true
	}
	dateFrom = fromT.Format("2006-01-02")
	dateTo = toT.Format("2006-01-02")
	if fromT.After(toT) {
		api.FailJSON(c, "DATE_RANGE_INVALID")
		return true
	}
	if toT.Sub(fromT).Hours()/24 > 30 {
		// inclusive span > 31 days (0..30 = 31 days)
		api.FailJSON(c, "DATE_RANGE_TOO_LARGE")
		return true
	}
	where := `WHERE COALESCE(w.is_deleted,0)=0 AND w.biz_date>=? AND w.biz_date<=?`
	args := []interface{}{dateFrom, dateTo}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_weigh_ticket w `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT w.id, w.doc_no, w.farmer_id, COALESCE(f.name,''), w.channel, w.product_id,
		w.variety, w.gross_weight, w.deduct_rate, w.deduct_weight, w.net_weight, w.qc_result, w.status,
		COALESCE(w.trace_code,''), COALESCE(w.origin,''), w.biz_date, COALESCE(w.source_type,'self'),
		COALESCE(w.image_url,''), COALESCE(w.box_code,''), w.created_at,
		COALESCE(w.receive_kind,''), COALESCE(w.batch_no,''), COALESCE(w.unit_price,0), COALESCE(w.settle_amount,0),
		COALESCE(w.bag_qty,0), COALESCE(w.cold_store_type,''), COALESCE(w.party_name,''), COALESCE(w.party_mobile,''),
		COALESCE(p.name,''),
		COALESCE((SELECT s.status FROM pur_farmer_settlement s WHERE s.weigh_ticket_id=w.id AND COALESCE(s.status,'')!='void' ORDER BY s.id DESC LIMIT 1),''),
		COALESCE((SELECT COALESCE(NULLIF(e.name,''), u.login_name, '')
			FROM wf_ticket t
			LEFT JOIN iam_user u ON u.id=t.current_assignee_user_id
			LEFT JOIN hr_employee e ON e.id=u.employee_id
			WHERE t.biz_type='weigh_ticket' AND t.biz_id=w.id AND t.status IN ('open','in_progress')
			ORDER BY t.id DESC LIMIT 1),'')
		FROM pur_weigh_ticket w
		LEFT JOIN pur_farmer f ON f.id=w.farmer_id
		LEFT JOIN prd_product p ON p.id=w.product_id
		`+where+` ORDER BY w.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, farmerID, productID int64
		var docNo, farmerName, channel, variety, qc, status, trace, origin, bizDate, source, image, box, created string
		var kind, batch, cold, partyName, partyMobile, productName, settleStatus, assigneeName string
		var gross, deductRate, deductWeight, net, unitPrice, settle, bagQty float64
		_ = rows.Scan(&id, &docNo, &farmerID, &farmerName, &channel, &productID, &variety, &gross, &deductRate,
			&deductWeight, &net, &qc, &status, &trace, &origin, &bizDate, &source, &image, &box, &created,
			&kind, &batch, &unitPrice, &settle, &bagQty, &cold, &partyName, &partyMobile, &productName,
			&settleStatus, &assigneeName)
		phase := weighProcessPhase(kind, status, settleStatus)
		row := gin.H{
			"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName, "channel": channel,
			"product_id": productID, "product_name": productName, "variety": variety,
			"gross_weight": gross, "deduct_rate": deductRate, "deduct_weight": deductWeight, "net_weight": net,
			"qc_result": qc, "status": status, "trace_code": trace, "origin": origin, "biz_date": bizDate,
			"source_type": source, "image_url": image, "box_code": box, "created_at": created,
			"receive_kind": kind, "batch_no": batch, "unit_price": unitPrice, "settle_amount": settle,
			"bag_qty": bagQty, "cold_store_type": cold, "party_name": partyName, "party_mobile": partyMobile,
			"settlement_status": settleStatus, "process_phase": phase, "current_assignee_name": assigneeName,
			"date_from": dateFrom, "date_to": dateTo,
		}
		if claimsIsWarehouseOnly(middleware.Claims(c)) {
			row = maskWeighTicketForWarehouse(row)
		}
		list = append(list, row)
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

// weighProcessPhase maps ticket+settlement into a UI-facing phase code.
func weighProcessPhase(kind, status, settleStatus string) string {
	st := strings.ToLower(strings.TrimSpace(status))
	kind = strings.ToLower(strings.TrimSpace(kind))
	settleStatus = strings.ToLower(strings.TrimSpace(settleStatus))
	switch st {
	case "weighed":
		return "await_warehouse"
	case "returned":
		return "returned_by_warehouse"
	case "gate_accepted":
		if settleStatus == "settle_paid" || settleStatus == "paid" {
			return "settled"
		}
		return "await_finance"
	case "stocked", "posted":
		if kind == "gate" {
			if settleStatus == "settle_paid" || settleStatus == "paid" {
				return "settled"
			}
			return "await_finance"
		}
		return "stocked_done"
	case "pending_confirm", "draft", "qc_pass", "qc_pending":
		return "pending_bind"
	default:
		return st
	}
}

func (s *Services) createWeighTicket(c *gin.Context) bool {
	body := bindBody(c)
	kind := strings.ToLower(strings.TrimSpace(strOr(body["receive_kind"])))
	if kind == "stockin" {
		api.FailJSON(c, "USE_WAREHOUSE_BOX_STOCKIN")
		return true
	}
	if kind != "gate" {
		api.FailJSON(c, "RECEIVE_KIND_REQUIRED")
		return true
	}
	batchNo := strings.ToUpper(strings.TrimSpace(strOr(body["batch_no"])))
	if batchNo == "" {
		api.FailJSON(c, "BATCH_NO_REQUIRED")
		return true
	}
	if ok, errCode := s.validateTraceBatchCode(batchNo, 0); !ok {
		api.FailJSON(c, errCode)
		return true
	}
	s.expireStaleTraceBatchReservations()
	var st string
	var reservedBy int64
	_ = s.DB.QueryRow(`SELECT status, COALESCE(reserved_by,0) FROM pur_trace_batch_code WHERE code=?`, batchNo).Scan(&st, &reservedBy)
	if st == "used" {
		api.FailJSON(c, "BATCH_CODE_USED")
		return true
	}
	if st == "reserved" {
		var uid int64
		if cl := middleware.Claims(c); cl != nil {
			uid = cl.UserID
		}
		if reservedBy > 0 && uid > 0 && reservedBy != uid {
			api.FailJSON(c, "BATCH_CODE_RESERVED")
			return true
		}
	}
	var existID int64
	_ = s.DB.QueryRow(`SELECT id FROM pur_weigh_ticket WHERE UPPER(batch_no)=? AND LOWER(COALESCE(receive_kind,''))='gate'
		AND status IN ('weighed','stocked','posted','gate_accepted') AND COALESCE(is_deleted,0)=0 LIMIT 1`, batchNo).Scan(&existID)
	if existID > 0 {
		api.FailJSON(c, "BATCH_CODE_USED")
		return true
	}
	imgs := collectImageURLs(body)
	if len(imgs) == 0 {
		api.FailJSON(c, "EVIDENCE_INCOMPLETE:site_photo")
		return true
	}
	for _, u := range imgs {
		if !isValidSitePhotoURL(u) {
			api.FailJSON(c, "EVIDENCE_INVALID:site_photo")
			return true
		}
	}
	imageURL := imgs[0]

	arrivalID, _ := asInt64(body["arrival_id"])
	var farmerID int64
	var grade, origin, sourceType, channel, variety, bizDate string
	partyName := strOr(body["party_name"])
	partyMobile := strOr(body["party_mobile"])

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
		farmerID, _ = asInt64(body["farmer_id"])
		grade = strings.ToUpper(strOrDef(body["grade"], "A"))
		origin = strOr(body["origin"])
		sourceType = strOrDef(body["source_type"], "self")
		channel = strOrDef(body["channel"], "internal")
		variety = strOrDef(body["variety"], "鲜木薯")
		bizDate = strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	}

	if farmerID > 0 {
		var farmerName, farmerOrigin, farmerMobile string
		err := s.DB.QueryRow(`SELECT name, COALESCE(origin,''), COALESCE(mobile,'') FROM pur_farmer WHERE id=? AND status='active' AND COALESCE(is_deleted,0)=0`, farmerID).
			Scan(&farmerName, &farmerOrigin, &farmerMobile)
		if err != nil {
			api.FailJSON(c, "FARMER_NOT_FOUND")
			return true
		}
		if origin == "" {
			origin = farmerOrigin
		}
		if partyName == "" {
			partyName = farmerName
		}
		if partyMobile == "" {
			partyMobile = farmerMobile
		}
	} else if partyName == "" {
		api.FailJSON(c, "PARTY_REQUIRED")
		return true
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
	s.resolveWeighVariety(body, &variety, &productID)
	if productID <= 0 {
		productID = 1
	}
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}

	var gross, deductRate, deductWeight, net, unitPrice, settle, bagQty float64
	var coldStore string
	var warehouseID int64
	freight, loading, weighFee, passRate, reject, plate, recvAddr := feeFieldsFromBody(body)

	if kind == "gate" {
		if bizDate == "" {
			api.FailJSON(c, "BIZ_DATE_REQUIRED")
			return true
		}
		gross, _ = asFloat(body["gross_weight"])
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
		rejectW := reject
		net = gross - deductWeight - rejectW
		if n, ok := asFloat(body["net_weight"]); ok && n > 0 {
			net = n
		}
		if net < 0 {
			net = 0
		}
		unitPrice, _ = asFloat(body["unit_price"])
		_, settle = settleAmount(net, unitPrice, freight, loading, weighFee)
		if sa, ok := asFloat(body["settle_amount"]); ok && sa > 0 {
			settle = sa
		}
		// 鲜木薯→保鲜库；半成品→半成品库（可显式传 cold_store_type）
		coldStore = strings.ToLower(strOr(body["cold_store_type"]))
		if coldStore == "" {
			v := strings.ToLower(variety)
			if strings.Contains(v, "半成品") || strings.Contains(v, "semi") {
				coldStore = "semi"
			} else {
				coldStore = "fresh"
			}
		}
		warehouseID = ColdStoreWarehouse(coldStore)
		if wid, ok := asInt64(body["warehouse_id"]); ok && wid > 0 {
			warehouseID = wid
		}
	} else {
		net, _ = asFloat(body["net_weight"])
		if net <= 0 {
			if g, ok := asFloat(body["gross_weight"]); ok && g > 0 {
				net = g
				gross = g
			}
		} else {
			gross = net
		}
		if net <= 0 {
			api.FailJSON(c, "NET_WEIGHT_REQUIRED")
			return true
		}
		bagQty, _ = asFloat(body["bag_qty"])
		coldStore = strings.ToLower(strOr(body["cold_store_type"]))
		if coldStore != "fresh" && coldStore != "semi" && coldStore != "fg" {
			api.FailJSON(c, "COLD_STORE_TYPE_REQUIRED")
			return true
		}
		warehouseID = ColdStoreWarehouse(coldStore)
		if wid, ok := asInt64(body["warehouse_id"]); ok && wid > 0 {
			warehouseID = wid
		}
	}

	ocrDraft := "{}"
	if body["ocr_draft"] != nil {
		b, _ := json.Marshal(body["ocr_draft"])
		ocrDraft = string(b)
	} else if draftJSON := strOr(body["ocr_draft_json"]); draftJSON != "" {
		ocrDraft = draftJSON
	}
	template := strOrDef(body["ticket_template"], channel)
	docNo := fmt.Sprintf("WT%d", time.Now().UnixNano())
	qcResult := ""
	status := "draft"
	if arrivalID > 0 || kind == "gate" || kind == "stockin" {
		// 现场入厂/入库：批号池码即溯源码，建单后立即绑定生效（无「待出码」中间态）
		qcResult = "pass"
		status = "pending_confirm" // 插入后立刻 bind；失败则整单回滚语义由后续删除保证
	}
	res, err := s.DB.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, channel, ticket_template, product_id, variety,
		gross_weight, deduct_rate, deduct_weight, net_weight, qc_result, status, trace_code, origin, biz_date,
		source_type, image_url, remark, arrival_id, grade, ocr_draft_json, batch_no,
		plate_no, receive_address, pass_rate, reject_weight, freight_fee, loading_fee, weigh_fee,
		receive_kind, unit_price, settle_amount, bag_qty, cold_store_type, party_name, party_mobile, warehouse_id)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		docNo, nullIf0(farmerID), channel, template, productID, variety, gross, deductRate, deductWeight, net,
		qcResult, status, "",
		origin, bizDate, sourceType, imageURL, strOr(body["remark"]), nullIf0(arrivalID), grade, ocrDraft, batchNo,
		plate, recvAddr, passRate, reject, freight, loading, weighFee,
		kind, unitPrice, settle, bagQty, coldStore, partyName, partyMobile, nullIf0(warehouseID))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	// Gate occupies batch pool; stockin reuses the gate-bound code (no second occupy).
	if kind == "gate" {
		var occupyUID int64
		if cl := middleware.Claims(c); cl != nil {
			occupyUID = cl.UserID
		}
		if err := s.occupyTraceBatchCode(batchNo, id, occupyUID); err != nil {
			_, _ = s.DB.Exec(`DELETE FROM pur_weigh_ticket WHERE id=?`, id)
			api.FailJSON(c, err.Error())
			return true
		}
	}
	for _, u := range imgs {
		_, _ = s.addEvidence(c, "weigh_ticket", id, "site_photo", u, nil)
	}
	s.writeAuditCtx(c, "weigh_ticket", id, "create", "weigh_draft", nil, gin.H{"net_weight": net, "receive_kind": kind, "batch_no": batchNo})

	// 同步协作工单，便于采购员在「我发起的」看到刚提交的单
	channelLabel := "厂内秤"
	if channel == "external" {
		channelLabel = "外磅单"
	}
	coldLabel := coldStore
	switch coldStore {
	case "fresh":
		coldLabel = "保鲜库"
	case "semi":
		coldLabel = "半成品库"
	case "fg":
		coldLabel = "成品库"
	}
	ticketPayload := map[string]interface{}{
		"batch_no": batchNo, "party_name": partyName, "party_mobile": partyMobile, "origin": origin,
		"variety": variety, "channel": channelLabel, "gross_weight": gross, "deduct_rate": deductRate * 100,
		"reject_weight": reject, "unit_price": unitPrice, "plate_no": plate, "receive_address": recvAddr,
		"freight_fee": freight, "loading_fee": loading, "weigh_fee": weighFee, "grade": grade,
		"net_weight": net, "bag_qty": bagQty, "cold_store_type": coldLabel, "remark": strOr(body["remark"]),
		"doc_no": docNo, "receive_kind": kind, "image_url": imageURL, "image_urls": imgs, "verify_images": imgs,
	}
	nextRole, _ := s.nextRoleAfterAction(kind, "submit", strOr(body["next_role"]), strOr(body["next_node_id"]))
	nextAssignee, _ := asInt64(body["next_assignee_user_id"])
	cl := middleware.Claims(c)
	if cl != nil && nextAssignee > 0 && nextAssignee == cl.UserID {
		s.releaseTraceBatchCode(id)
		_, _ = s.DB.Exec(`DELETE FROM pur_weigh_ticket WHERE id=?`, id)
		api.FailJSON(c, "SELF_ASSIGN_FORBIDDEN")
		return true
	}
	ticketID := s.spawnWeighCollabTicketWithRole(c, kind, id, docNo, ticketPayload, nextRole, nextAssignee)
	if ticketID <= 0 {
		s.releaseTraceBatchCode(id)
		_, _ = s.DB.Exec(`DELETE FROM pur_weigh_ticket WHERE id=?`, id)
		api.FailJSON(c, "SELF_ASSIGN_FORBIDDEN")
		return true
	}

	out := s.loadWeighTicket(id)
	out["ticket_id"] = ticketID
	out["ticket"] = s.loadTicket(ticketID)

	// 扫码批号即溯源码：建单同请求绑定农户/本单并推仓管（activate 默认开启；显式 false 可仅建草稿）
	activate := true
	if v, ok := body["activate"]; ok {
		activate = asBool(v)
	}
	if activate {
		if code := s.ensureWeighIssued(c, id, body); code != "" {
			s.releaseTraceBatchCode(id)
			if ticketID > 0 {
				_, _ = s.DB.Exec(`UPDATE wf_ticket SET status='cancelled', updated_at=datetime('now') WHERE id=?`, ticketID)
			}
			_, _ = s.DB.Exec(`DELETE FROM pur_weigh_ticket WHERE id=?`, id)
			api.FailJSON(c, code)
			return true
		}
		out = s.loadWeighTicket(id)
		out["ticket_id"] = ticketID
		out["ticket"] = s.loadTicket(ticketID)
		out["label"] = s.buildLabel(out)
		s.notifyWeighConfirmed(c, id, out)
	}

	api.OK(c, out)
	return true
}

func (s *Services) notifyWeighConfirmed(c *gin.Context, id int64, out gin.H) {
	if s.Notify == nil {
		return
	}
	// 退回后再推：清掉旧待办，避免 dedupe 挡住新任务
	_, _ = s.DB.Exec(`DELETE FROM wf_task WHERE biz_type='weigh_ticket' AND biz_id=? AND event_key IN ('purchase.weigh_confirmed','purchase.weigh_returned')`, id)
	s.Notify.NotifyNext(c, notify.Event{
		Key: "purchase.weigh_confirmed", BizType: "weigh_ticket", BizID: id,
		DocNo: strOr(out["doc_no"]), TraceCode: strOr(out["trace_code"]),
		FromRole: "purchase", ToRoles: []string{"warehouse"}, CreateTask: true,
		Payload: gin.H{
			"net_weight": out["net_weight"], "batch_no": out["batch_no"], "biz_date": out["biz_date"],
			"variety": out["variety"], "product_name": out["product_name"], "plate_no": out["plate_no"],
			"reject_weight": out["reject_weight"], "gross_weight": out["gross_weight"],
			"deduct_weight": out["deduct_weight"], "trace_code": out["trace_code"],
			"cold_store_type": out["cold_store_type"], "receive_kind": out["receive_kind"],
			"image_url": out["image_url"], "verify_images": func() []string {
				urls, _ := s.collectWeighVerifyMedia(id, strOr(out["image_url"]))
				return urls
			}(),
		},
	})
}

func (s *Services) updateWeighTicket(c *gin.Context) bool {
	id := paramID(c)
	var status, curBatch string
	if err := s.DB.QueryRow(`SELECT status, COALESCE(batch_no,'') FROM pur_weigh_ticket WHERE id=?`, id).Scan(&status, &curBatch); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status != "draft" && status != "returned" {
		api.FailJSON(c, "ONLY_DRAFT_EDITABLE")
		return true
	}
	body := bindBody(c)
	if bn := strings.ToUpper(strings.TrimSpace(strOr(body["batch_no"]))); bn != "" && bn != strings.ToUpper(curBatch) {
		api.FailJSON(c, "BATCH_NO_LOCKED")
		return true
	}
	gross, _ := asFloat(body["gross_weight"])
	deductRate, _ := asFloat(body["deduct_rate"])
	deductWeight, _ := asFloat(body["deduct_weight"])
	if deductRate > 1 {
		deductRate = deductRate / 100
	}
	rejectW := asFloatOr0(body["reject_weight"])
	if deductRate > 0 && gross > 0 {
		deductWeight = gross * deductRate
	}
	net := gross - deductWeight - rejectW
	if n, ok := asFloat(body["net_weight"]); ok && n > 0 {
		net = n
	}
	unitPrice := asFloatOr0(body["unit_price"])
	freight, loading, weighFee, passRate, reject, plate, recvAddr := feeFieldsFromBody(body)
	_, settle := settleAmount(net, unitPrice, freight, loading, weighFee)
	if sa, ok := asFloat(body["settle_amount"]); ok && sa > 0 {
		settle = sa
	}
	bagQty := asFloatOr0(body["bag_qty"])
	cold := strOr(body["cold_store_type"])
	imgs := collectImageURLs(body)
	imageURL := strOr(body["image_url"])
	if len(imgs) > 0 {
		for _, u := range imgs {
			if !isValidSitePhotoURL(u) {
				api.FailJSON(c, "EVIDENCE_INVALID:site_photo")
				return true
			}
		}
		imageURL = imgs[0]
	}
	_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET gross_weight=COALESCE(NULLIF(?,0),gross_weight),
		deduct_rate=?, deduct_weight=?, net_weight=?, variety=COALESCE(NULLIF(?,''),variety),
		image_url=COALESCE(NULLIF(?,''),image_url), remark=COALESCE(NULLIF(?,''),remark),
		plate_no=COALESCE(NULLIF(?,''),plate_no), receive_address=COALESCE(NULLIF(?,''),receive_address),
		pass_rate=COALESCE(NULLIF(?,0),pass_rate), reject_weight=COALESCE(NULLIF(?,0),reject_weight),
		freight_fee=COALESCE(NULLIF(?,0),freight_fee), loading_fee=COALESCE(NULLIF(?,0),loading_fee),
		weigh_fee=COALESCE(NULLIF(?,0),weigh_fee),
		unit_price=COALESCE(NULLIF(?,0),unit_price), settle_amount=COALESCE(NULLIF(?,0),settle_amount),
		bag_qty=COALESCE(NULLIF(?,0),bag_qty), cold_store_type=COALESCE(NULLIF(?,''),cold_store_type),
		party_name=COALESCE(NULLIF(?,''),party_name), party_mobile=COALESCE(NULLIF(?,''),party_mobile),
		origin=COALESCE(NULLIF(?,''),origin),
		updated_at=datetime('now') WHERE id=?`,
		gross, deductRate, deductWeight, net, strOr(body["variety"]), imageURL, strOr(body["remark"]),
		plate, recvAddr, passRate, reject, freight, loading, weighFee,
		unitPrice, settle, bagQty, cold, strOr(body["party_name"]), strOr(body["party_mobile"]),
		strOr(body["origin"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	if len(imgs) > 0 {
		_, _ = s.DB.Exec(`UPDATE biz_evidence SET voided_at=datetime('now') WHERE biz_type='weigh_ticket' AND biz_id=? AND evidence_type='site_photo' AND COALESCE(voided_at,'')=''`, id)
		for _, u := range imgs {
			_, _ = s.addEvidence(c, "weigh_ticket", id, "site_photo", u, nil)
		}
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
	if result == "pass" {
		s.advanceWeighTicketAssignee(c, id, "qc_deduct", strOr(body["next_role"]), strOr(body["next_node_id"]))
	}
	api.OK(c, out)
	return true
}

// applyVerifiedWeighStockIn 仓管核对相符后：入厂仅接收（无箱码）；入库分箱建箱。already=true 表示幂等返回。
func (s *Services) applyVerifiedWeighStockIn(c *gin.Context, id int64, body map[string]interface{}) (out gin.H, errCode string, already bool) {
	verified := asBool(body["verified"]) || asBool(body["match_confirmed"])
	if !verified {
		return nil, "VERIFY_REQUIRED", false
	}
	var status, qc, trace, receiveKind string
	if err := s.DB.QueryRow(`SELECT status, COALESCE(qc_result,''), COALESCE(trace_code,''), COALESCE(receive_kind,'') FROM pur_weigh_ticket WHERE id=?`, id).
		Scan(&status, &qc, &trace, &receiveKind); err != nil {
		return nil, "NOT_FOUND", false
	}
	kind := strings.ToLower(strings.TrimSpace(receiveKind))
	if kind != "stockin" {
		kind = "gate"
	}
	if status == "stocked" || (kind == "gate" && status == "gate_accepted") {
		return s.loadWeighTicket(id), "", true
	}
	// 采购→仓管闭环：工单可能仍停在 pending_confirm，仓管确认时自动出码再入库
	if status != "weighed" || trace == "" {
		if code := s.ensureWeighIssued(c, id, body); code != "" {
			return nil, code, false
		}
		if err := s.DB.QueryRow(`SELECT status, COALESCE(qc_result,''), COALESCE(trace_code,''), COALESCE(receive_kind,'') FROM pur_weigh_ticket WHERE id=?`, id).
			Scan(&status, &qc, &trace, &receiveKind); err != nil {
			return nil, "NOT_FOUND", false
		}
		kind = strings.ToLower(strings.TrimSpace(receiveKind))
		if kind != "stockin" {
			kind = "gate"
		}
	}
	if status != "weighed" {
		return nil, "WEIGH_CONFIRM_REQUIRED", false
	}
	if qc != "pass" && qc != "" {
		return nil, "QC_PASS_REQUIRED", false
	}
	if trace == "" {
		return nil, "TRACE_CODE_REQUIRED", false
	}

	var inboundLoss float64
	auditAction := "stock_in"
	if kind == "gate" {
		auditAction = "gate_accept"
		_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET status='gate_accepted', purchase_completed_at=datetime('now'), updated_at=datetime('now') WHERE id=?`, id)
		_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET status='gate_accepted', updated_at=datetime('now') WHERE id=(SELECT arrival_id FROM pur_weigh_ticket WHERE id=?)`, id)
	} else {
		ok, msg, loss := s.doWeighStockIn(id, body)
		if !ok {
			return nil, msg, false
		}
		inboundLoss = loss
		_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET purchase_completed_at=datetime('now') WHERE id=?`, id)
		_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET status='stocked', updated_at=datetime('now') WHERE id=(SELECT arrival_id FROM pur_weigh_ticket WHERE id=?)`, id)
	}

	m := s.loadWeighTicket(id)
	var settleID int64
	var breakdown gin.H
	settlePoint := s.farmerSettlePoint()
	if kind == "gate" && settlePoint == "gate" {
		sid, bd, errMsg := s.ensureGateSettlement(id, m, nil)
		if errMsg != "" {
			return nil, errMsg, false
		}
		settleID = sid
		breakdown = bd
		s.patchWeighTicketPayload(id, map[string]interface{}{
			"settlement_id": settleID, "settle_breakdown": breakdown,
		})
	}
	s.writeAuditCtx(c, "weigh_ticket", id, "warehouse_confirm", auditAction, nil, m)
	if cl := middleware.Claims(c); cl != nil {
		var tid int64
		_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, id).Scan(&tid)
		if tid > 0 {
			s.appendTicketLog(tid, "warehouse_confirm", cl.UserID, 0, auditAction)
		}
	}
	s.advanceWeighTicketAssignee(c, id, "warehouse_confirm", strOr(body["next_role"]), strOr(body["next_node_id"]))
	if s.Notify != nil {
		s.Notify.CompleteTask("weigh_ticket", id, "purchase.weigh_confirmed")
		if kind == "gate" {
			s.Notify.NotifyNext(c, notify.Event{
				Key: "purchase.gate_accepted", BizType: "weigh_ticket", BizID: id,
				DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
				FromRole: "warehouse", ToRoles: []string{"purchase"}, CreateTask: false,
				Title: "入厂完成", Body: "仓管已确认入厂 " + strOr(m["doc_no"]) + "，可扫溯源分箱入库",
				Payload: gin.H{"trace_code": m["trace_code"], "status": "gate_accepted"},
			})
			if settleID > 0 {
				payload := gin.H{
					"net_weight": m["net_weight"], "doc_no": m["doc_no"], "trace_code": m["trace_code"],
					"settlement_id": settleID, "settle_breakdown": breakdown,
				}
				s.Notify.NotifyNext(c, notify.Event{
					Key: "purchase.gate_accepted", BizType: "weigh_ticket", BizID: id,
					DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
					FromRole: "warehouse", ToRoles: []string{"finance"}, CreateTask: true,
					Title: "待财务结算", Body: fmt.Sprintf("%s 入厂净重 %v 应付 %v", m["doc_no"], m["net_weight"], breakdown["amount"]),
					Payload: payload,
				})
			}
		} else {
			payload := gin.H{
				"box_code": m["box_code"], "net_weight": m["net_weight"], "doc_no": m["doc_no"],
				"inbound_loss_kg": inboundLoss,
			}
			s.Notify.NotifyNext(c, notify.Event{
				Key: "purchase.stocked", BizType: "weigh_ticket", BizID: id,
				DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
				FromRole: "warehouse", ToRoles: []string{"purchase"}, CreateTask: false,
				Title: "入库完成", Body: "仓管已确认入库 " + strOr(m["doc_no"]),
				Payload: payload,
			})
		}
	}
	out = s.loadWeighTicket(id)
	if settleID > 0 {
		out["settlement_id"] = settleID
		out["settle_breakdown"] = breakdown
	}
	if inboundLoss > 0 {
		out["inbound_loss_kg"] = inboundLoss
	}
	return out, "", false
}

func (s *Services) stockInWeighTicket(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse") {
		return true
	}
	if !s.requireMobileClient(c) {
		return true
	}
	id := paramID(c)
	// 去掉 App「认领」后：仓管核对确认时自动接管开着的协作工单
	s.takeOverWeighTicketForWarehouse(c, id)
	if !s.requireOpenTicketAssignee(c, "weigh_ticket", id) {
		return true
	}
	body := bindBody(c)
	out, errCode, _ := s.applyVerifiedWeighStockIn(c, id, body)
	if errCode != "" {
		api.FailJSON(c, errCode)
		return true
	}
	api.OK(c, out)
	return true
}

// boxStockInWeighTicket 仓管扫溯源后对已入厂批次分箱入库。
func (s *Services) boxStockInWeighTicket(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse") {
		return true
	}
	if !s.requireMobileClient(c) {
		return true
	}
	id := paramID(c)
	s.takeOverWeighTicketForWarehouse(c, id)
	body := bindBody(c)

	var status, trace, kind string
	if err := s.DB.QueryRow(`SELECT status, COALESCE(trace_code,''), COALESCE(receive_kind,'') FROM pur_weigh_ticket WHERE id=?`, id).
		Scan(&status, &trace, &kind); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status == "stocked" {
		out := s.loadWeighTicket(id)
		api.OK(c, out)
		return true
	}
	if status != "gate_accepted" {
		api.FailJSON(c, "GATE_ACCEPT_REQUIRED")
		return true
	}
	if strings.TrimSpace(trace) == "" {
		api.FailJSON(c, "TRACE_CODE_REQUIRED")
		return true
	}

	ok, msg, inboundLoss := s.doWeighStockIn(id, body)
	if !ok {
		api.FailJSON(c, msg)
		return true
	}
	_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET status='stocked', updated_at=datetime('now') WHERE id=(SELECT arrival_id FROM pur_weigh_ticket WHERE id=?)`, id)

	m := s.loadWeighTicket(id)
	ticketNet := asFloatOr0(m["net_weight"])
	boxSum := ticketNet - inboundLoss
	if boxSum < 0 {
		boxSum = 0
	}
	lossRate := 0.0
	if ticketNet > 0 && inboundLoss > 0 {
		lossRate = inboundLoss / ticketNet
	}

	var settleID int64
	var breakdown gin.H
	if s.farmerSettlePoint() == "box_stockin" {
		sid, bd, errMsg := s.ensureGateSettlement(id, m, &boxSum)
		if errMsg != "" {
			api.FailJSON(c, errMsg)
			return true
		}
		settleID = sid
		breakdown = bd
	}

	s.patchWeighTicketPayload(id, map[string]interface{}{
		"inbound_loss_kg": inboundLoss, "inbound_loss_rate": lossRate, "box_sum_kg": boxSum,
		"settlement_id": settleID, "settle_breakdown": breakdown,
	})
	s.writeAuditCtx(c, "weigh_ticket", id, "box_stock_in", "box_stockin", nil, m)
	if cl := middleware.Claims(c); cl != nil {
		var tid int64
		_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, id).Scan(&tid)
		if tid > 0 {
			s.appendTicketLog(tid, "box_stock_in", cl.UserID, 0, "box_stockin")
		}
	}

	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.stocked", BizType: "weigh_ticket", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
			FromRole: "warehouse", ToRoles: []string{"purchase"}, CreateTask: false,
			Title: "分箱入库完成", Body: fmt.Sprintf("%s 箱合计 %.2f 仓前损耗 %.2f", m["doc_no"], boxSum, inboundLoss),
			Payload: gin.H{
				"box_code": m["box_code"], "inbound_loss_kg": inboundLoss, "inbound_loss_rate": lossRate,
				"box_sum_kg": boxSum,
			},
		})
		if settleID > 0 {
			s.Notify.NotifyNext(c, notify.Event{
				Key: "purchase.stocked", BizType: "weigh_ticket", BizID: id,
				DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
				FromRole: "warehouse", ToRoles: []string{"finance"}, CreateTask: true,
				Title: "待财务结算", Body: fmt.Sprintf("%s 分箱净重 %.2f 应付 %v", m["doc_no"], boxSum, breakdown["amount"]),
				Payload: gin.H{"settlement_id": settleID, "settle_breakdown": breakdown, "box_sum_kg": boxSum},
			})
		}
	}

	out := s.loadWeighTicket(id)
	out["inbound_loss_kg"] = inboundLoss
	out["inbound_loss_rate"] = lossRate
	out["box_sum_kg"] = boxSum
	if settleID > 0 {
		out["settlement_id"] = settleID
		out["settle_breakdown"] = breakdown
	}
	api.OK(c, out)
	return true
}

// takeOverWeighTicketForWarehouse assigns open weigh collab ticket to current warehouse user.
func (s *Services) takeOverWeighTicketForWarehouse(c *gin.Context, weighID int64) {
	cl := middleware.Claims(c)
	if cl == nil || weighID <= 0 {
		return
	}
	var tid, assignee int64
	_ = s.DB.QueryRow(`SELECT id, COALESCE(current_assignee_user_id,0) FROM wf_ticket
		WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, weighID).
		Scan(&tid, &assignee)
	if tid <= 0 || assignee == cl.UserID {
		return
	}
	_, _ = s.DB.Exec(`UPDATE wf_ticket SET current_assignee_user_id=?, status='in_progress', updated_at=datetime('now') WHERE id=?`, cl.UserID, tid)
	s.appendTicketLog(tid, "assign", assignee, cl.UserID, "warehouse_takeover")
}

// warehouseReturnWeighTicket 仓管核对信息不符时退回采购（可指定采购员）。
func (s *Services) warehouseReturnWeighTicket(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse") {
		return true
	}
	if !s.requireMobileClient(c) {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	reason := strings.TrimSpace(strOr(body["reason"]))
	if reason == "" {
		api.FailJSON(c, "REASON_REQUIRED")
		return true
	}
	m := s.loadWeighTicket(id)
	if m["id"] == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	status := strings.ToLower(strOr(m["status"]))
	if status != "weighed" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	toUID, _ := asInt64(body["to_user_id"])
	if toUID <= 0 {
		var applicant, confirmedBy int64
		_ = s.DB.QueryRow(`SELECT COALESCE(applicant_user_id,0) FROM wf_ticket
			WHERE biz_type='weigh_ticket' AND biz_id=? ORDER BY id DESC LIMIT 1`, id).Scan(&applicant)
		_ = s.DB.QueryRow(`SELECT COALESCE(confirmed_by,0) FROM pur_weigh_ticket WHERE id=?`, id).Scan(&confirmedBy)
		toUID = applicant
		if toUID <= 0 {
			toUID = confirmedBy
		}
	}
	if toUID <= 0 {
		api.FailJSON(c, "PURCHASE_USER_REQUIRED")
		return true
	}
	if !s.userHasRoleCode(toUID, "purchase") && !s.userHasRoleCode(toUID, "采购") && !s.userHasRoleCode(toUID, "采购员") {
		api.FailJSON(c, "TARGET_NOT_PURCHASE")
		return true
	}
	before := m
	_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET status='returned', remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=?`,
		reason, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	var tid int64
	_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, id).Scan(&tid)
	var fromUID int64
	if cl := middleware.Claims(c); cl != nil {
		fromUID = cl.UserID
	}
	if tid > 0 {
		_, _ = s.DB.Exec(`UPDATE wf_ticket SET current_assignee_user_id=?, status='in_progress', updated_at=datetime('now') WHERE id=?`, toUID, tid)
		s.appendTicketLog(tid, "warehouse_return", fromUID, toUID, reason)
		s.notifyTicketAssignee(c, tid, "workflow.ticket.assigned", strOr(m["doc_no"])+" 仓管退回", toUID, fromUID)
	}
	if s.Notify != nil {
		s.Notify.CompleteTask("weigh_ticket", id, "purchase.weigh_confirmed")
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.weigh_returned", BizType: "weigh_ticket", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["trace_code"]),
			FromRole: "warehouse", ToRoles: []string{"purchase"}, CreateTask: true,
			Title: "仓管退回过磅单",
			Body:  fmt.Sprintf("%s：%s", m["doc_no"], reason),
			Payload: gin.H{
				"reason": reason, "receive_kind": m["receive_kind"], "doc_no": m["doc_no"],
				"trace_code": m["trace_code"], "batch_no": m["batch_no"], "to_user_id": toUID,
				"notify_user_ids": []int64{toUID},
			},
		})
		_, _ = s.DB.Exec(`UPDATE wf_task SET assignee_user_id=? WHERE biz_type='weigh_ticket' AND biz_id=? AND event_key='purchase.weigh_returned' AND status='pending'`,
			toUID, id)
	}
	out := s.loadWeighTicket(id)
	s.writeAuditCtx(c, "weigh_ticket", id, "warehouse_return", reason, before, out)
	api.OK(c, out)
	return true
}

// ensureWeighIssued 将过磅单推进到 weighed，并以批号作为溯源码与农户/本单唯一绑定；已 weighed/stocked 则幂等成功。
func (s *Services) ensureWeighIssued(c *gin.Context, id int64, body map[string]interface{}) string {
	m := s.loadWeighTicket(id)
	if m["id"] == nil {
		return "NOT_FOUND"
	}
	status := strOr(m["status"])
	if status == "weighed" || status == "stocked" || status == "gate_accepted" {
		if strOr(m["trace_code"]) != "" || status == "stocked" || status == "gate_accepted" {
			return ""
		}
	}
	// 仓管退回后采购修正再推：已有溯源绑定则直接恢复 weighed
	if status == "returned" && strOr(m["trace_code"]) != "" {
		_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET status='weighed', updated_at=datetime('now') WHERE id=?`, id)
		if err != nil {
			return "DB_ERROR:" + err.Error()
		}
		s.writeAuditCtx(c, "weigh_ticket", id, "repush_after_return", "returned_to_weighed", m, s.loadWeighTicket(id))
		return ""
	}
	kind := strings.ToLower(strOr(m["receive_kind"]))
	if status != "draft" && status != "pending_confirm" && status != "qc_pass" && status != "returned" {
		return "WEIGH_CONFIRM_REQUIRED"
	}
	if status == "draft" && kind != "gate" && kind != "stockin" {
		return "QC_PASS_REQUIRED"
	}
	if err := s.requireEvidence("weigh_ticket", id, "site_photo"); err != nil {
		return err.Error()
	}
	gross, _ := asFloat(m["gross_weight"])
	deductRate, _ := asFloat(m["deduct_rate"])
	deductWeight, _ := asFloat(m["deduct_weight"])
	net, _ := asFloat(m["net_weight"])
	if body != nil {
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
	}
	grade := strOr(m["grade"])
	if body != nil {
		grade = strOrDef(body["grade"], grade)
	}
	if grade == "" {
		grade = "A"
	}
	bizDate := strOr(m["biz_date"])
	batch := strings.ToUpper(strings.TrimSpace(strOr(m["batch_no"])))
	if batch == "" {
		return "BATCH_NO_REQUIRED"
	}
	// 扫码批号即溯源码（不再另签发 T1- 复合码）
	trace := batch
	farmerID, _ := asInt64(m["farmer_id"])
	arrivalID, _ := asInt64(m["arrival_id"])
	in := TraceIssueInput{
		BizDate: bizDate, BatchNo: batch, FarmerID: farmerID, Grade: grade,
		Channel: strOr(m["channel"]), SourceType: strOr(m["source_type"]), NetKg: net, ArrivalID: arrivalID,
	}
	secret := TraceHMACSecret(s.TraceHMACSecret)
	canonical := in.Canonical()
	sig := SignCanonical(secret, canonical)
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	snap := nowSnap(map[string]interface{}{
		"gross_weight": gross, "deduct_rate": deductRate, "deduct_weight": deductWeight, "net_weight": net,
		"grade": grade, "trace_code": trace, "batch_no": batch, "farmer_id": farmerID, "bind": "batch_as_trace",
	})
	_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET gross_weight=?, deduct_rate=?, deduct_weight=?, net_weight=?, grade=?,
		trace_code=?, batch_no=?, status='weighed', confirmed_by=?, confirmed_at=datetime('now'), confirmed_snapshot_json=?,
		updated_at=datetime('now') WHERE id=?`,
		gross, deductRate, deductWeight, net, grade, trace, batch, uid, snap, id)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	// 同一批号（溯源码）只建一条 lot；入库单复用入厂绑定，不重复插入
	var lotID int64
	_ = s.DB.QueryRow(`SELECT id FROM pur_trace_lot WHERE UPPER(trace_code)=? OR weigh_ticket_id=? ORDER BY id LIMIT 1`, trace, id).Scan(&lotID)
	if lotID <= 0 {
		_, err = s.DB.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, arrival_id, weigh_ticket_id, channel, source_type, net_weight, payload_canonical, signature, status)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'open')`,
			trace, normalizeBizDate(bizDate), batch, farmerID, grade, nullIf0(arrivalID), id, strOr(m["channel"]), strOr(m["source_type"]), net, canonical, sig)
		if err != nil {
			return "DB_ERROR:" + err.Error()
		}
	}
	if arrivalID > 0 {
		_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET status='weighed', updated_at=datetime('now') WHERE id=?`, arrivalID)
	}
	s.writeAuditCtx(c, "weigh_ticket", id, "bind_batch_trace", "batch_as_trace", m, s.loadWeighTicket(id))
	return ""
}

func (s *Services) confirmWeighTicket(c *gin.Context) bool {
	if !s.requireAnyRole(c, "purchase") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	wasReturned := false
	{
		var st string
		_ = s.DB.QueryRow(`SELECT status FROM pur_weigh_ticket WHERE id=?`, id).Scan(&st)
		wasReturned = strings.EqualFold(st, "returned")
	}
	if code := s.ensureWeighIssued(c, id, body); code != "" {
		api.FailJSON(c, code)
		return true
	}
	out := s.loadWeighTicket(id)
	out["label"] = s.buildLabel(out)
	// 退回后再推：改派仓管并重新发待办
	if wasReturned {
		s.advanceWeighTicketAssignee(c, id, "submit", "warehouse", "")
	}
	s.notifyWeighConfirmed(c, id, out)
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

func parseInboundBoxWeights(body map[string]interface{}) ([]float64, string) {
	if body == nil {
		return nil, "BOXES_REQUIRED"
	}
	raw, ok := body["boxes"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, "BOXES_REQUIRED"
	}
	weights := make([]float64, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]interface{})
		if m == nil {
			return nil, "BOX_WEIGHT_REQUIRED"
		}
		w, ok := asFloat(m["weight"])
		if !ok || w <= 0 {
			return nil, "BOX_WEIGHT_REQUIRED"
		}
		weights = append(weights, w)
	}
	return weights, ""
}

func weighBoxOverweight(ticketNet, boxSum float64) bool {
	if ticketNet <= 0 || boxSum <= ticketNet {
		return false
	}
	diff := boxSum - ticketNet
	tol := ticketNet * 0.03
	if tol < 5 {
		tol = 5
	}
	return diff > tol
}

func (s *Services) writeInboundLossTxn(weighID, warehouseID, productID int64, loss float64, bizDate string) error {
	if loss <= 0 || productID <= 0 {
		return nil
	}
	if warehouseID <= 0 {
		warehouseID = 1
	}
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	docNo := fmt.Sprintf("INLOSS-WT-%d", weighID)
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
		docNo, "inbound_loss", bizDate, warehouseID, fmt.Sprintf("weigh ticket #%d inbound_loss", weighID))
	if err != nil {
		// doc_no 冲突时换号重试一次
		docNo = fmt.Sprintf("INLOSS%d", time.Now().UnixNano()%1e12)
		res, err = s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'posted',?,?)`,
			docNo, "inbound_loss", bizDate, warehouseID, fmt.Sprintf("weigh ticket #%d inbound_loss", weighID))
		if err != nil {
			return err
		}
	}
	tid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'out',?)`,
		tid, 1, productID, loss, loss, bizDate)
	return s.adjustBalance(warehouseID, productID, -loss)
}

func (s *Services) allocInboundBoxCode(trace string, seq int) string {
	trace = strings.TrimSpace(trace)
	base := fmt.Sprintf("BX-%s-%02d", trace, seq)
	if len(base) > 64 {
		base = fmt.Sprintf("BX%d-%02d", time.Now().UnixNano()%1e10, seq)
	}
	code := base
	for i := 0; i < 20; i++ {
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_box_code WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&n)
		if n == 0 {
			return code
		}
		code = fmt.Sprintf("%s-%d", base, i+1)
	}
	return fmt.Sprintf("BX%d", time.Now().UnixNano()%1e12)
}

func (s *Services) doWeighStockIn(id int64, body map[string]interface{}) (bool, string, float64) {
	m := s.loadWeighTicket(id)
	if m["id"] == nil {
		return false, "NOT_FOUND", 0
	}
	if strOr(m["status"]) == "stocked" {
		return true, "", 0
	}
	weights, errMsg := parseInboundBoxWeights(body)
	if errMsg != "" {
		return false, errMsg, 0
	}
	productID, _ := asInt64(body["product_id"])
	if productID <= 0 {
		productID, _ = asInt64(m["product_id"])
	}
	if productID <= 0 {
		return false, "PRODUCT_REQUIRED", 0
	}
	procID, stepID, stepWH, errMsg := s.resolveInboundEntryStep(productID)
	if errMsg != "" {
		return false, errMsg, 0
	}
	var boxSum float64
	for _, w := range weights {
		boxSum += w
	}
	ticketNet, _ := asFloat(m["net_weight"])
	if weighBoxOverweight(ticketNet, boxSum) {
		return false, "WEIGHT_MISMATCH", 0
	}
	inboundLoss := 0.0
	if ticketNet > boxSum {
		inboundLoss = ticketNet - boxSum
	}

	farmerID, _ := asInt64(m["farmer_id"])
	trace := strings.TrimSpace(strOr(m["trace_code"]))
	if trace == "" {
		return false, "TRACE_CODE_REQUIRED", 0
	}
	origin := strOr(m["origin"])
	bizDate := strOr(m["biz_date"])
	sourceType := strOrDef(m["source_type"], "self")
	wh := stepWH
	if wh <= 0 {
		wh = 1
		if wid := asInt64Or0(m["warehouse_id"]); wid > 0 {
			wh = wid
		} else if cw := ColdStoreWarehouse(strOr(m["cold_store_type"])); cw > 0 {
			wh = cw
		} else {
			vname := strings.ToLower(strOr(m["variety"]) + " " + strOr(m["product_name"]))
			if strings.Contains(vname, "半成品") || strings.Contains(vname, "semi") || sourceType == "outsource" {
				wh = 2
			}
		}
	}

	boxCodes := make([]string, 0, len(weights))
	for i, w := range weights {
		code := s.allocInboundBoxCode(trace, i+1)
		_, err := s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, origin, receive_date, source_type, status, current_process_id, current_step_id)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,'open',?,?)`,
			code, productID, wh, bizDate, w, w, farmerID, trace, origin, bizDate, sourceType, procID, stepID)
		if err != nil {
			code = fmt.Sprintf("BX%d", time.Now().UnixNano()%1e12)
			_, err = s.DB.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, origin, receive_date, source_type, status, current_process_id, current_step_id)
				VALUES(?,?,?,?,?,?,?,?,?,?,?,'open',?,?)`,
				code, productID, wh, bizDate, w, w, farmerID, trace, origin, bizDate, sourceType, procID, stepID)
			if err != nil {
				return false, "BOX_CREATE_ERROR:" + err.Error(), 0
			}
		}
		boxCodes = append(boxCodes, code)
	}
	firstBox := ""
	if len(boxCodes) > 0 {
		firstBox = boxCodes[0]
	}

	txnNo := fmt.Sprintf("ST-WT-%d", id)
	docType := "purchase_in"
	if sourceType == "outsource" {
		docType = "outsource_in"
	}
	tres, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,?,'draft',?,?)`,
		txnNo, docType, bizDate, wh, fmt.Sprintf("weigh ticket #%d farmer=%d boxes=%d", id, farmerID, len(weights)))
	if err != nil {
		return false, "STOCK_TXN_ERROR:" + err.Error(), 0
	}
	tid, _ := tres.LastInsertId()
	for i, w := range weights {
		_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction, batch_no) VALUES(?,?,?,?,?,'in',?)`,
			tid, i+1, productID, w, w, bizDate)
	}
	if err := s.adjustBalanceBatch(wh, productID, bizDate, boxSum); err != nil {
		return false, "BALANCE_ERROR:" + err.Error(), 0
	}
	_, _ = s.DB.Exec(`UPDATE inv_stock_txn SET status='posted', posted_at=datetime('now') WHERE id=?`, tid)

	// 仓前损耗：票净重 − 分箱合计（欠重自动记损，不硬拒）
	if inboundLoss > 0 {
		if err := s.writeInboundLossTxn(id, wh, productID, inboundLoss, bizDate); err != nil {
			return false, "INBOUND_LOSS_ERROR:" + err.Error(), 0
		}
	}

	_, _ = s.DB.Exec(`UPDATE pur_weigh_ticket SET status='stocked', product_id=?, box_code=?, warehouse_id=?, updated_at=datetime('now') WHERE id=?`,
		productID, firstBox, wh, id)
	_, _ = s.DB.Exec(`UPDATE pur_trace_lot SET status='stocked' WHERE weigh_ticket_id=?`, id)
	return true, "", inboundLoss
}

func (s *Services) loadWeighTicket(id int64) gin.H {
	var farmerID, productID, warehouseID, arrivalID int64
	var docNo, channel, template, variety, qc, status, trace, origin, bizDate, source, image, box, remark, created, grade, batch string
	var plate, recvAddr, kind, cold, partyName, partyMobile, productName string
	var gross, deductRate, deductWeight, net, passRate, reject, freight, loading, weighFee, unitPrice, settle, bagQty float64
	var farmerName string
	err := s.DB.QueryRow(`SELECT w.doc_no, w.farmer_id, COALESCE(f.name,''), w.channel, COALESCE(w.ticket_template,''), w.product_id, w.variety,
		w.gross_weight, w.deduct_rate, w.deduct_weight, w.net_weight, COALESCE(w.qc_result,''), w.status,
		COALESCE(w.trace_code,''), COALESCE(w.origin,''), w.biz_date, COALESCE(w.source_type,'self'),
		COALESCE(w.image_url,''), COALESCE(w.box_code,''), COALESCE(w.warehouse_id,0), COALESCE(w.remark,''), w.created_at,
		COALESCE(w.arrival_id,0), COALESCE(w.grade,''), COALESCE(w.batch_no,''),
		COALESCE(w.plate_no,''), COALESCE(w.receive_address,''), COALESCE(w.pass_rate,0), COALESCE(w.reject_weight,0),
		COALESCE(w.freight_fee,0), COALESCE(w.loading_fee,0), COALESCE(w.weigh_fee,0),
		COALESCE(w.receive_kind,''), COALESCE(w.unit_price,0), COALESCE(w.settle_amount,0), COALESCE(w.bag_qty,0),
		COALESCE(w.cold_store_type,''), COALESCE(w.party_name,''), COALESCE(w.party_mobile,''), COALESCE(p.name,'')
		FROM pur_weigh_ticket w
		LEFT JOIN pur_farmer f ON f.id=w.farmer_id
		LEFT JOIN prd_product p ON p.id=w.product_id
		WHERE w.id=?`, id).
		Scan(&docNo, &farmerID, &farmerName, &channel, &template, &productID, &variety, &gross, &deductRate, &deductWeight, &net,
			&qc, &status, &trace, &origin, &bizDate, &source, &image, &box, &warehouseID, &remark, &created,
			&arrivalID, &grade, &batch, &plate, &recvAddr, &passRate, &reject, &freight, &loading, &weighFee,
			&kind, &unitPrice, &settle, &bagQty, &cold, &partyName, &partyMobile, &productName)
	if err != nil {
		return gin.H{}
	}
	out := gin.H{
		"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName, "channel": channel,
		"ticket_template": template, "product_id": productID, "product_name": productName, "variety": variety,
		"gross_weight": gross, "deduct_rate": deductRate, "deduct_weight": deductWeight, "net_weight": net,
		"qc_result": qc, "status": status, "trace_code": trace, "origin": origin, "biz_date": bizDate,
		"source_type": source, "image_url": image, "box_code": box, "warehouse_id": warehouseID,
		"remark": remark, "created_at": created, "arrival_id": arrivalID, "grade": grade, "batch_no": batch,
		"receive_kind": kind, "unit_price": unitPrice, "settle_amount": settle, "bag_qty": bagQty,
		"cold_store_type": cold, "party_name": partyName, "party_mobile": partyMobile,
		"evidences": s.listEvidence("weigh_ticket", id),
	}
	for k, v := range feeMap(freight, loading, weighFee, passRate, reject, plate, recvAddr) {
		out[k] = v
	}
	return out
}

// attachWeighProcessTrail 附带协同工单处理流水（谁在何时做了哪一步），供 App 单据展开查看。
func (s *Services) attachWeighProcessTrail(m gin.H, weighID int64) {
	rows, err := s.DB.Query(`SELECT id, COALESCE(doc_no,''), COALESCE(status,''), COALESCE(applicant_user_id,0),
		COALESCE(current_assignee_user_id,0), COALESCE(created_at,'')
		FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? ORDER BY id`, weighID)
	if err != nil {
		m["process_logs"] = []gin.H{}
		return
	}
	defer rows.Close()
	logs := []gin.H{}
	var latestTicketID int64
	for rows.Next() {
		var tid, applicant, assignee int64
		var docNo, status, created string
		if err := rows.Scan(&tid, &docNo, &status, &applicant, &assignee, &created); err != nil {
			continue
		}
		latestTicketID = tid
		m["wf_ticket_id"] = tid
		m["wf_ticket_status"] = status
		m["applicant_user_id"] = applicant
		m["applicant_name"] = s.userDisplayName(applicant)
		m["current_assignee_user_id"] = assignee
		m["current_assignee_name"] = s.userDisplayName(assignee)
		// 建单节点（log 里可能无 create 时补一条）
		logs = append(logs, gin.H{
			"id": 0, "action": "create", "action_label": "建单提交",
			"from_user_id": applicant, "to_user_id": assignee,
			"from_name": s.userDisplayName(applicant), "to_name": s.userDisplayName(assignee),
			"comment": docNo, "created_at": created, "ticket_id": tid,
		})
		for _, e := range s.listTicketLogs(tid) {
			act := strOr(e["action"])
			if act == "create" {
				// 已有建单 log 则去掉上面补的那条（保留真实 log）
				if len(logs) > 0 {
					if prev, ok := logs[len(logs)-1]["action"].(string); ok && prev == "create" && logs[len(logs)-1]["id"] == 0 {
						logs = logs[:len(logs)-1]
					}
				}
			}
			e["action_label"] = weighProcessActionLabel(act)
			e["ticket_id"] = tid
			logs = append(logs, e)
		}
	}
	if latestTicketID > 0 {
		m["wf_ticket_id"] = latestTicketID
	}
	var settleStatus, settleDoc string
	var settleAmt float64
	_ = s.DB.QueryRow(`SELECT COALESCE(status,''), COALESCE(doc_no,''), COALESCE(amount,0)
		FROM pur_farmer_settlement WHERE weigh_ticket_id=? AND COALESCE(status,'')!='void' ORDER BY id DESC LIMIT 1`, weighID).
		Scan(&settleStatus, &settleDoc, &settleAmt)
	if settleStatus != "" {
		m["settlement_status"] = settleStatus
		m["settlement_doc_no"] = settleDoc
		m["settlement_amount"] = settleAmt
	}
	m["process_logs"] = logs
	m["currency"] = "CNY"
	m["currency_label"] = "元人民币"
	// 下一步待办人（开着的工单当前处理人；已完结则标明）
	wfSt := strings.ToLower(strOr(m["wf_ticket_status"]))
	assigneeName := strings.TrimSpace(strOr(m["current_assignee_name"]))
	switch {
	case wfSt == "done" || wfSt == "rejected" || wfSt == "cancelled":
		m["next_handler_name"] = ""
		m["next_handler_hint"] = "流程已完结"
	case settleStatus == "settle_paid" || settleStatus == "paid":
		m["next_handler_name"] = ""
		m["next_handler_hint"] = "已付款结清"
	case assigneeName != "":
		m["next_handler_name"] = assigneeName
		m["next_handler_hint"] = "待 " + assigneeName + " 处理"
	default:
		m["next_handler_name"] = ""
		m["next_handler_hint"] = "待指派下一处理人"
	}
}

func weighProcessActionLabel(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "create":
		return "建单提交"
	case "assign":
		return "指派/认领"
	case "approve", "pass":
		return "审批通过"
	case "reject":
		return "驳回"
	case "comment":
		return "备注"
	case "warehouse_confirm", "stock_in":
		return "仓管确认入库"
	case "warehouse_return":
		return "仓管退回采购"
	case "settle_pay", "settle_paid":
		return "财务付款关单"
	case "confirm":
		return "确认出码"
	default:
		if strings.HasPrefix(action, "flow:") {
			return "流程流转"
		}
		if action == "" {
			return "处理"
		}
		return action
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
	var wtID int64
	if err := s.DB.QueryRow(`SELECT status, COALESCE(weigh_ticket_id,0) FROM pur_farmer_settlement WHERE id=?`, id).Scan(&status, &wtID); err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if wtID > 0 && !s.requireOpenTicketAssignee(c, "weigh_ticket", wtID) {
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
	cl := middleware.Claims(c)
	fromUID := int64(0)
	if cl != nil {
		fromUID = cl.UserID
	}
	var ticketID int64
	m := s.loadSettlement(id)
	if wtID > 0 {
		ticketID = s.closeWeighTicketByBiz(wtID, fromUID, "settle_paid:"+transferNo)
	}
	if s.Notify != nil {
		s.Notify.CompleteTask("farmer_settlement", id)
		if wtID > 0 {
			s.Notify.CompleteTask("weigh_ticket", wtID, "purchase.stocked")
		}
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.settle_paid", BizType: "farmer_settlement", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: "",
			FromRole: "finance", ToRoles: []string{"purchase"}, CreateTask: false,
			Payload: gin.H{"transfer_no": transferNo, "amount": m["amount"], "ticket_id": ticketID},
		})
	}
	api.OK(c, gin.H{"id": id, "status": "settle_paid", "transfer_no": transferNo, "ticket_id": ticketID})
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
