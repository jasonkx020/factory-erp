package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleInboundArrivals(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		if strings.Contains(c.Request.URL.Path, "/qc") {
			return false
		}
		return s.listArrivals(c)
	case action == "create":
		return s.createArrival(c)
	case action == "get":
		m := s.loadArrival(paramID(c))
		if m["id"] == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, m)
		return true
	case strings.Contains(action, "qc") || strings.Contains(c.Request.URL.Path, "/qc"):
		return s.qcArrival(c)
	case action == "delete":
		return s.refuseDelete(c)
	}
	return false
}

func (s *Services) listArrivals(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	status := c.Query("status")
	qc := c.Query("qc_result")
	where := `WHERE COALESCE(a.is_deleted,0)=0`
	args := []interface{}{}
	if status != "" {
		where += ` AND a.status=?`
		args = append(args, status)
	}
	if qc != "" {
		where += ` AND a.qc_result=?`
		args = append(args, qc)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_inbound_arrival a `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT a.id, a.doc_no, a.farmer_id, COALESCE(f.name,''), COALESCE(a.origin,''), COALESCE(a.variety,''),
		a.estimate_weight, a.source_type, a.channel, COALESCE(a.qc_result,''), COALESCE(a.grade,''), a.status,
		COALESCE(a.qc_image_url,''), a.biz_date, a.created_at
		FROM pur_inbound_arrival a LEFT JOIN pur_farmer f ON f.id=a.farmer_id `+where+`
		ORDER BY a.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, farmerID int64
		var docNo, fname, origin, variety, source, channel, qcRes, grade, status, img, bizDate, created string
		var est float64
		_ = rows.Scan(&id, &docNo, &farmerID, &fname, &origin, &variety, &est, &source, &channel, &qcRes, &grade, &status, &img, &bizDate, &created)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": fname, "origin": origin, "variety": variety,
			"estimate_weight": est, "source_type": source, "channel": channel, "qc_result": qcRes, "grade": grade,
			"status": status, "qc_image_url": img, "biz_date": bizDate, "created_at": created,
			"evidences": s.listEvidence("inbound_arrival", id),
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) createArrival(c *gin.Context) bool {
	body := bindBody(c)
	farmerID, _ := asInt64(body["farmer_id"])
	if farmerID <= 0 {
		api.FailJSON(c, "FARMER_REQUIRED")
		return true
	}
	docNo := fmt.Sprintf("AR%s", time.Now().Format("20060102150405"))
	bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
	est, _ := asFloat(body["estimate_weight"])
	freight, loading, weighFee, passRate, reject, plate, recvAddr := feeFieldsFromBody(body)
	res, err := s.DB.Exec(`INSERT INTO pur_inbound_arrival(doc_no, farmer_id, origin, variety, estimate_weight, source_type, channel, status, biz_date, remark,
		plate_no, receive_address, pass_rate, reject_weight, freight_fee, loading_fee, weigh_fee)
		VALUES(?,?,?,?,?,?,?,'qc_pending',?,?,?,?,?,?,?,?,?)`,
		docNo, farmerID, strOr(body["origin"]), strOrDef(body["variety"], "鲜木薯"), est,
		strOrDef(body["source_type"], "self"), strOrDef(body["channel"], "internal"), bizDate, strOr(body["remark"]),
		plate, recvAddr, passRate, reject, freight, loading, weighFee)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	if img := strOr(body["qc_image_url"]); img != "" {
		_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET qc_image_url=? WHERE id=?`, img, id)
		_, _ = s.addEvidence(c, "inbound_arrival", id, "qc_photo", img, nil)
	}
	api.OK(c, s.loadArrival(id))
	return true
}

func (s *Services) qcArrival(c *gin.Context) bool {
	if !s.requireAnyRole(c, "purchase", "qc") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	result := strings.ToLower(strOrDef(body["qc_result"], strOr(body["result"])))
	if result == "合格" {
		result = "pass"
	}
	if result == "不合格" {
		result = "fail"
	}
	if result != "pass" && result != "fail" {
		api.FailJSON(c, "QC_RESULT_REQUIRED")
		return true
	}
	grade := strings.ToUpper(strOr(body["grade"]))
	if result == "pass" && grade == "" {
		api.FailJSON(c, "GRADE_REQUIRED")
		return true
	}
	img := strOr(body["qc_image_url"])
	if img == "" {
		img = strOr(body["image_url"])
	}
	if img != "" {
		_, _ = s.DB.Exec(`UPDATE pur_inbound_arrival SET qc_image_url=? WHERE id=?`, img, id)
		_, _ = s.addEvidence(c, "inbound_arrival", id, "qc_photo", img, nil)
	}
	if err := s.requireEvidence("inbound_arrival", id, "qc_photo"); err != nil && img == "" {
		// also accept if already on row
		var existing string
		_ = s.DB.QueryRow(`SELECT COALESCE(qc_image_url,'') FROM pur_inbound_arrival WHERE id=?`, id).Scan(&existing)
		if existing == "" {
			api.FailJSON(c, "EVIDENCE_INCOMPLETE:qc_photo")
			return true
		}
		_, _ = s.addEvidence(c, "inbound_arrival", id, "qc_photo", existing, nil)
	}
	newStatus := "qc_rejected"
	if result == "pass" {
		newStatus = "qc_pass"
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	snap := nowSnap(map[string]interface{}{"qc_result": result, "grade": grade})
	_, err := s.DB.Exec(`UPDATE pur_inbound_arrival SET qc_result=?, grade=?, status=?, confirmed_by=?, confirmed_at=datetime('now'),
		confirmed_snapshot_json=?, remark=COALESCE(NULLIF(?,''),remark), updated_at=datetime('now') WHERE id=?`,
		result, grade, newStatus, uid, snap, strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.writeAuditCtx(c, "inbound_arrival", id, "qc", result, nil, gin.H{"status": newStatus, "grade": grade})
	api.OK(c, s.loadArrival(id))
	return true
}

func (s *Services) loadArrival(id int64) gin.H {
	var farmerID int64
	var docNo, origin, variety, source, channel, qcRes, grade, status, img, bizDate, remark, created string
	var plate, recvAddr string
	var est, passRate, reject, freight, loading, weighFee float64
	var farmerName string
	err := s.DB.QueryRow(`SELECT a.doc_no, a.farmer_id, COALESCE(f.name,''), COALESCE(a.origin,''), COALESCE(a.variety,''),
		a.estimate_weight, a.source_type, a.channel, COALESCE(a.qc_result,''), COALESCE(a.grade,''), a.status,
		COALESCE(a.qc_image_url,''), a.biz_date, COALESCE(a.remark,''), a.created_at,
		COALESCE(a.plate_no,''), COALESCE(a.receive_address,''), COALESCE(a.pass_rate,0), COALESCE(a.reject_weight,0),
		COALESCE(a.freight_fee,0), COALESCE(a.loading_fee,0), COALESCE(a.weigh_fee,0)
		FROM pur_inbound_arrival a LEFT JOIN pur_farmer f ON f.id=a.farmer_id WHERE a.id=?`, id).
		Scan(&docNo, &farmerID, &farmerName, &origin, &variety, &est, &source, &channel, &qcRes, &grade, &status, &img, &bizDate, &remark, &created,
			&plate, &recvAddr, &passRate, &reject, &freight, &loading, &weighFee)
	if err != nil {
		return gin.H{}
	}
	out := gin.H{
		"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": farmerName, "origin": origin, "variety": variety,
		"estimate_weight": est, "source_type": source, "channel": channel, "qc_result": qcRes, "grade": grade,
		"status": status, "qc_image_url": img, "biz_date": bizDate, "remark": remark, "created_at": created,
		"evidences": s.listEvidence("inbound_arrival", id),
	}
	for k, v := range feeMap(freight, loading, weighFee, passRate, reject, plate, recvAddr) {
		out[k] = v
	}
	return out
}
