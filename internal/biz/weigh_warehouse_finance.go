package biz

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/security"
)

func claimsHasAnyRole(cl *security.Claims, roles ...string) bool {
	if cl == nil {
		return false
	}
	set := map[string]bool{}
	for _, r := range cl.Roles {
		set[strings.ToLower(strings.TrimSpace(r))] = true
	}
	for _, want := range roles {
		if set[strings.ToLower(want)] {
			return true
		}
	}
	return false
}

func claimsIsFinanceViewer(cl *security.Claims) bool {
	if cl == nil {
		return false
	}
	return claimsHasAnyRole(cl, "finance", "purchase", "sys_admin", "admin") ||
		claimsIsSysAdmin(cl.Roles, cl.Permissions)
}

func claimsIsWarehouseOnly(cl *security.Claims) bool {
	if cl == nil {
		return false
	}
	if claimsIsFinanceViewer(cl) {
		return false
	}
	return claimsHasAnyRole(cl, "warehouse")
}

func maskWeighPayloadForWarehouse(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return map[string]interface{}{}
	}
	allow := map[string]bool{
		"doc_no": true, "batch_no": true, "trace_code": true, "variety": true, "product_name": true,
		"product_id": true, "plate_no": true, "gross_weight": true, "deduct_rate": true, "deduct_weight": true,
		"reject_weight": true, "net_weight": true, "image_url": true, "image_urls": true, "cold_store_type": true,
		"warehouse_id": true, "receive_kind": true, "status": true, "bag_qty": true, "channel": true,
		"biz_date": true, "ticket_id": true, "id": true, "weigh_ticket_id": true, "stockin_ready": true, "reason": true,
		"box_code": true, "qc_result": true, "grade": true,
		"evidences": true, "verify_images": true, "site_photos": true,
		"process_phase": true, "settlement_status": true, "current_assignee_name": true,
		"party_name": true, "farmer_name": true, "settle_amount": true, "date_from": true, "date_to": true,
		"process_logs": true, "applicant_name": true, "currency": true, "currency_label": true,
		"settlement_doc_no": true, "settlement_amount": true, "wf_ticket_id": true, "wf_ticket_status": true,
		"next_handler_name": true, "next_handler_hint": true,
	}
	out := map[string]interface{}{}
	for k, v := range src {
		if allow[strings.ToLower(k)] {
			out[k] = v
		}
	}
	return out
}

func maskWeighPayloadJSON(raw string) string {
	m := parsePayloadMap(raw)
	b, _ := json.Marshal(maskWeighPayloadForWarehouse(m))
	return string(b)
}

func maskWeighTicketForWarehouse(m gin.H) gin.H {
	raw := map[string]interface{}{}
	for k, v := range m {
		raw[k] = v
	}
	out := gin.H{}
	for k, v := range maskWeighPayloadForWarehouse(raw) {
		out[k] = v
	}
	return out
}

// collectWeighVerifyMedia 现场核对用图片（主图 + site_photo 凭证）。
func (s *Services) collectWeighVerifyMedia(weighID int64, imageURL string) (urls []string, evidences []gin.H) {
	seen := map[string]bool{}
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		urls = append(urls, u)
	}
	add(imageURL)
	for _, e := range s.listEvidence("weigh_ticket", weighID) {
		if strOr(e["voided_at"]) != "" {
			continue
		}
		et := strings.ToLower(strOr(e["evidence_type"]))
		switch et {
		case "site_photo", "weigh_photo", "image", "photo", "":
			u := strOr(e["file_url"])
			add(u)
			if u != "" {
				evidences = append(evidences, gin.H{
					"id": e["id"], "evidence_type": et, "file_url": u, "uploaded_at": e["uploaded_at"],
				})
			}
		}
	}
	return urls, evidences
}

func (s *Services) attachWeighVerifyMedia(out gin.H, weighID int64, imageURL string) {
	urls, evs := s.collectWeighVerifyMedia(weighID, imageURL)
	if len(urls) == 0 {
		return
	}
	out["image_url"] = urls[0]
	out["image_urls"] = urls
	out["verify_images"] = urls
	out["site_photos"] = urls
	if len(evs) > 0 {
		out["evidences"] = evs
	}
}

func (s *Services) ensureGateSettlement(weighID int64, m gin.H) (int64, gin.H, string) {
	kind := strings.ToLower(strOr(m["receive_kind"]))
	if kind != "gate" {
		return 0, nil, ""
	}
	var existID int64
	_ = s.DB.QueryRow(`SELECT id FROM pur_farmer_settlement WHERE weigh_ticket_id=? ORDER BY id DESC LIMIT 1`, weighID).Scan(&existID)
	if existID > 0 {
		return existID, s.settlementBreakdownFromRow(existID), ""
	}
	farmerID := asInt64Or0(m["farmer_id"])
	net := asFloatOr0(m["net_weight"])
	bizDate := strOr(m["biz_date"])
	var unitPrice float64
	grade := strOr(m["grade"])
	_ = s.DB.QueryRow(`SELECT unit_price FROM pur_grade_price WHERE grade=? AND status='active'`, grade).Scan(&unitPrice)
	if unitPrice <= 0 {
		if up := asFloatOr0(m["unit_price"]); up > 0 {
			unitPrice = up
		} else {
			_ = s.DB.QueryRow(`SELECT COALESCE(default_unit_price,0) FROM pur_farmer WHERE id=?`, farmerID).Scan(&unitPrice)
		}
	}
	freight := asFloatOr0(m["freight_fee"])
	loading := asFloatOr0(m["loading_fee"])
	weighFee := asFloatOr0(m["weigh_fee"])
	goods, total := settleAmount(net, unitPrice, freight, loading, weighFee)
	docNo := fmt.Sprintf("FS%d%04d", weighID, weighID%10000)
	res, err := s.DB.Exec(`INSERT INTO pur_farmer_settlement(doc_no, farmer_id, weigh_ticket_id, biz_date, net_weight, unit_price, amount, status, remark,
		freight_fee, loading_fee, weigh_fee, goods_amount)
		VALUES(?,?,?,?,?,?,?,'settle_pending',?,?,?,?,?)`,
		docNo, farmerID, weighID, bizDate, net, unitPrice, total, "auto from weigh net_weight",
		freight, loading, weighFee, goods)
	if err != nil {
		return 0, nil, "SETTLEMENT_ERROR:" + err.Error()
	}
	sid, _ := res.LastInsertId()
	bd := gin.H{
		"settlement_id": sid, "doc_no": docNo, "net_weight": net, "unit_price": unitPrice,
		"goods_amount": goods, "freight_fee": freight, "loading_fee": loading, "weigh_fee": weighFee,
		"amount": total, "farmer_id": farmerID, "farmer_name": m["farmer_name"], "status": "settle_pending",
	}
	return sid, bd, ""
}

func (s *Services) settlementBreakdownFromRow(id int64) gin.H {
	var farmerID int64
	var docNo, status, farmerName string
	var net, price, amt, freight, loading, weighFee, goods float64
	_ = s.DB.QueryRow(`SELECT s.doc_no, s.farmer_id, COALESCE(f.name,''), s.net_weight, s.unit_price, s.amount, s.status,
		COALESCE(s.freight_fee,0), COALESCE(s.loading_fee,0), COALESCE(s.weigh_fee,0), COALESCE(s.goods_amount,0)
		FROM pur_farmer_settlement s LEFT JOIN pur_farmer f ON f.id=s.farmer_id WHERE s.id=?`, id).
		Scan(&docNo, &farmerID, &farmerName, &net, &price, &amt, &status, &freight, &loading, &weighFee, &goods)
	return gin.H{
		"settlement_id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName,
		"net_weight": net, "unit_price": price, "goods_amount": goods, "freight_fee": freight,
		"loading_fee": loading, "weigh_fee": weighFee, "amount": amt, "status": status,
	}
}

func (s *Services) patchWeighTicketPayload(weighID int64, patch map[string]interface{}) {
	var tid int64
	var payload string
	_ = s.DB.QueryRow(`SELECT id, COALESCE(payload_json,'{}') FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, weighID).
		Scan(&tid, &payload)
	if tid <= 0 {
		return
	}
	m := map[string]interface{}{}
	_ = json.Unmarshal([]byte(payload), &m)
	for k, v := range patch {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	_, _ = s.DB.Exec(`UPDATE wf_ticket SET payload_json=?, updated_at=datetime('now') WHERE id=?`, string(b), tid)
}

func (s *Services) closeWeighTicketByBiz(weighID int64, fromUID int64, comment string) int64 {
	var tid int64
	_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, weighID).Scan(&tid)
	if tid <= 0 {
		return 0
	}
	_, _ = s.DB.Exec(`UPDATE wf_ticket SET status='done', closed_at=datetime('now'), updated_at=datetime('now'), current_assignee_user_id=NULL WHERE id=?`, tid)
	s.appendTicketLog(tid, "settle_paid", fromUID, 0, comment)
	if s.Notify != nil {
		s.Notify.CompleteTask("wf_ticket", tid)
	}
	return tid
}

func (s *Services) resolveWeighByTraceCode(c *gin.Context) bool {
	code := strings.ToUpper(strings.TrimSpace(c.Query("code")))
	if code == "" {
		body := bindBody(c)
		code = strings.ToUpper(strings.TrimSpace(strOr(body["code"])))
	}
	if code == "" {
		api.FailJSON(c, "CODE_REQUIRED")
		return true
	}
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM pur_weigh_ticket WHERE COALESCE(is_deleted,0)=0 AND (UPPER(COALESCE(trace_code,''))=? OR UPPER(COALESCE(batch_no,''))=?) ORDER BY id DESC LIMIT 1`, code, code).Scan(&id)
	if err != nil || id <= 0 {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	m := s.loadWeighTicket(id)
	st := strOr(m["status"])
	if st == "stocked" {
		api.FailJSON(c, "ALREADY_STOCKED")
		return true
	}
	var ticketID int64
	_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, id).Scan(&ticketID)
	out := maskWeighTicketForWarehouse(m)
	out["ticket_id"] = ticketID
	out["weigh_ticket_id"] = id
	s.attachWeighVerifyMedia(out, id, strOr(m["image_url"]))
	// weighed 可直接入库；pending_confirm/qc_pass 允许仓管一键出码+入库
	ready := st == "weighed" || st == "pending_confirm" || st == "qc_pass"
	if st == "draft" && strings.ToLower(strOr(m["receive_kind"])) == "gate" {
		ready = true
	}
	if !ready {
		out["stockin_ready"] = false
		if strOr(m["trace_code"]) == "" && st != "weighed" {
			out["reason"] = "WEIGH_CONFIRM_REQUIRED"
		} else {
			out["reason"] = "TRACE_NOT_READY"
		}
		api.OK(c, out)
		return true
	}
	out["stockin_ready"] = true
	api.OK(c, out)
	return true
}
