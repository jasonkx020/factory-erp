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

const traceBatchReserveTTL = 15 * time.Minute

func (s *Services) handleTraceBatchCodes(c *gin.Context, method, action string) bool {
	path := c.Request.URL.Path
	switch {
	case strings.Contains(path, "/generate") || action == "action:generate":
		return s.generateTraceBatchCodes(c)
	case strings.Contains(path, "/validate") || action == "action:validate":
		return s.validateTraceBatchCodeAPI(c)
	case strings.Contains(path, "/void") || action == "action:void":
		return s.voidTraceBatchCode(c)
	case action == "list" || (method == "GET" && !strings.Contains(path, "/{") && !strings.Contains(c.FullPath(), ":id")):
		return s.listTraceBatchCodes(c)
	}
	return false
}

func (s *Services) listTraceBatchCodes(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	bizDate := normalizeBizDate(strOrDef(c.Query("biz_date"), time.Now().Format("2006-01-02")))
	status := strings.TrimSpace(c.Query("status"))
	lot := padLot2(c.Query("lot_no"))
	where := `WHERE biz_date=?`
	args := []interface{}{bizDate}
	if status != "" {
		where += ` AND status=?`
		args = append(args, status)
	}
	if strings.TrimSpace(c.Query("lot_no")) != "" {
		where += ` AND lot_no=?`
		args = append(args, lot)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_trace_batch_code `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, code, biz_date, seq_no, lot_no, status, COALESCE(weigh_ticket_id,0), created_at, COALESCE(used_at,'')
		FROM pur_trace_batch_code `+where+` ORDER BY seq_no ASC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, seq, wtID int64
		var code, bd, lotNo, st, created, used string
		_ = rows.Scan(&id, &code, &bd, &seq, &lotNo, &st, &wtID, &created, &used)
		list = append(list, gin.H{
			"id": id, "code": code, "biz_date": bd, "seq_no": seq, "lot_no": lotNo,
			"status": st, "weigh_ticket_id": wtID, "created_at": created, "used_at": used,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) generateTraceBatchCodes(c *gin.Context) bool {
	body := bindBody(c)
	bizDate := normalizeBizDate(strOrDef(body["biz_date"], time.Now().Format("2006-01-02")))
	lot := padLot2(strOrDef(body["lot_no"], "01"))
	qty, _ := asInt64(body["qty"])
	if qty <= 0 {
		qty = 1
	}
	if qty > 500 {
		api.FailJSON(c, "QTY_TOO_LARGE")
		return true
	}
	var maxSeq int
	_ = s.DB.QueryRow(`SELECT COALESCE(MAX(seq_no),0) FROM pur_trace_batch_code WHERE biz_date=? AND lot_no=?`, bizDate, lot).Scan(&maxSeq)
	secret := TraceHMACSecret(s.TraceHMACSecret)
	created := []gin.H{}
	for i := int64(0); i < qty; i++ {
		seq := maxSeq + int(i) + 1
		if seq > 9999 {
			api.FailJSON(c, "SEQ_OVERFLOW")
			return true
		}
		code := BuildTraceBatchCode(secret, bizDate, seq, lot)
		_, err := s.DB.Exec(`INSERT INTO pur_trace_batch_code(code, biz_date, seq_no, lot_no, status) VALUES(?,?,?,?,'available')`,
			code, bizDate, seq, lot)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		created = append(created, gin.H{"code": code, "biz_date": bizDate, "seq_no": seq, "lot_no": lot, "status": "available"})
	}
	api.OK(c, gin.H{"biz_date": bizDate, "lot_no": lot, "qty": len(created), "list": created})
	return true
}

func (s *Services) validateTraceBatchCodeAPI(c *gin.Context) bool {
	body := bindBody(c)
	code := strings.ToUpper(strings.TrimSpace(strOr(body["code"])))
	kind := strings.ToLower(strings.TrimSpace(strOr(body["receive_kind"])))
	if kind == "stockin" {
		out, errCode := s.validateTraceBatchForStockin(code)
		if errCode != "" {
			api.FailJSON(c, errCode)
			return true
		}
		api.OK(c, out)
		return true
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	out, errCode := s.reserveTraceBatchCodeForGate(code, uid)
	if errCode != "" {
		api.FailJSON(c, errCode)
		return true
	}
	api.OK(c, out)
	return true
}

// reserveTraceBatchCodeForGate atomically reserves an available pool code for gate inbound.
func (s *Services) reserveTraceBatchCodeForGate(code string, userID int64) (gin.H, string) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, "CODE_REQUIRED"
	}
	secret := TraceHMACSecret(s.TraceHMACSecret)
	if _, _, _, ok := ParseTraceBatchCode(secret, code); !ok {
		return nil, "BATCH_CODE_INVALID"
	}
	s.expireStaleTraceBatchReservations()
	if userID > 0 {
		// release other codes reserved by this user (swap code)
		_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET status='available', reserved_by=NULL, reserved_at=NULL
			WHERE status='reserved' AND reserved_by=? AND code<>?`, userID, code)
	}
	var status string
	var wtID, reservedBy int64
	var reservedAt string
	err := s.DB.QueryRow(`SELECT status, COALESCE(weigh_ticket_id,0), COALESCE(reserved_by,0), COALESCE(reserved_at,'')
		FROM pur_trace_batch_code WHERE code=?`, code).Scan(&status, &wtID, &reservedBy, &reservedAt)
	if err != nil {
		return nil, "BATCH_CODE_NOT_FOUND"
	}
	switch status {
	case "void":
		return nil, "BATCH_CODE_VOID"
	case "used":
		return nil, "BATCH_CODE_USED"
	case "reserved":
		if userID > 0 && reservedBy == userID {
			_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET reserved_at=NOW() WHERE code=? AND status='reserved'`, code)
			return gin.H{
				"code": code, "valid": true, "status": "reserved", "receive_kind": "gate",
				"expires_in_sec": int(traceBatchReserveTTL.Seconds()),
			}, ""
		}
		return nil, "BATCH_CODE_RESERVED"
	case "available":
		if userID <= 0 {
			// no auth context: read-only ok (admin tools); App always authenticated
			return gin.H{"code": code, "valid": true, "status": "available", "receive_kind": "gate"}, ""
		}
		res, err := s.DB.Exec(`UPDATE pur_trace_batch_code SET status='reserved', reserved_by=?, reserved_at=NOW()
			WHERE code=? AND status='available'`, userID, code)
		if err != nil {
			return nil, "DB_ERROR:" + err.Error()
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return nil, "BATCH_CODE_RESERVED"
		}
		return gin.H{
			"code": code, "valid": true, "status": "reserved", "receive_kind": "gate",
			"expires_in_sec": int(traceBatchReserveTTL.Seconds()),
		}, ""
	default:
		return nil, "BATCH_CODE_UNAVAILABLE"
	}
}

func (s *Services) expireStaleTraceBatchReservations() {
	mins := int(traceBatchReserveTTL.Minutes())
	if mins < 1 {
		mins = 15
	}
	_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET status='available', reserved_by=NULL, reserved_at=NULL
		WHERE status='reserved' AND (
			reserved_at IS NULL OR reserved_at='' OR
			datetime(reserved_at) < datetime('now', ?)
		)`, fmt.Sprintf("-%d minutes", mins))
}

// validateTraceBatchForStockin allows used codes that were bound at gate entry.
func (s *Services) validateTraceBatchForStockin(code string) (gin.H, string) {
	secret := TraceHMACSecret(s.TraceHMACSecret)
	if _, _, _, ok := ParseTraceBatchCode(secret, code); !ok {
		return nil, "BATCH_CODE_INVALID"
	}
	var status string
	err := s.DB.QueryRow(`SELECT status FROM pur_trace_batch_code WHERE code=?`, code).Scan(&status)
	if err != nil {
		return nil, "BATCH_CODE_NOT_FOUND"
	}
	if status == "void" {
		return nil, "BATCH_CODE_VOID"
	}
	bind, errCode := s.resolveGateBindingByBatch(code)
	if errCode != "" {
		return nil, errCode
	}
	out := gin.H{
		"code": code, "valid": true, "status": status, "receive_kind": "stockin",
		"gate_ticket_id": bind["gate_ticket_id"], "farmer_id": bind["farmer_id"],
		"farmer_name": bind["farmer_name"], "party_name": bind["party_name"],
		"party_mobile": bind["party_mobile"], "origin": bind["origin"],
		"channel": bind["channel"], "product_id": bind["product_id"],
		"variety_id": bind["variety_id"], "variety": bind["variety"],
		"grade": bind["grade"], "unit_price": bind["unit_price"],
		"plate_no": bind["plate_no"], "receive_address": bind["receive_address"],
	}
	return out, ""
}

// resolveGateBindingByBatch returns farmer/product binding from the latest gate weigh ticket for this batch.
func (s *Services) resolveGateBindingByBatch(batchNo string) (gin.H, string) {
	batchNo = strings.ToUpper(strings.TrimSpace(batchNo))
	if batchNo == "" {
		return nil, "BATCH_NO_REQUIRED"
	}
	var gateID, farmerID, productID int64
	var partyName, partyMobile, origin, channel, farmerName, variety, grade, plate, recvAddr string
	var unitPrice float64
	err := s.DB.QueryRow(`SELECT w.id, COALESCE(w.farmer_id,0), COALESCE(w.party_name,''), COALESCE(w.party_mobile,''),
		COALESCE(w.origin,''), COALESCE(w.channel,''), COALESCE(f.name,''),
		COALESCE(w.product_id,0), COALESCE(w.variety,''), COALESCE(w.grade,''),
		COALESCE(w.unit_price,0), COALESCE(w.plate_no,''), COALESCE(w.receive_address,'')
		FROM pur_weigh_ticket w
		LEFT JOIN pur_farmer f ON f.id=w.farmer_id
		WHERE UPPER(w.batch_no)=? AND LOWER(COALESCE(w.receive_kind,''))='gate'
		  AND COALESCE(w.is_deleted,0)=0
		ORDER BY CASE WHEN LOWER(w.status) IN ('weighed','stocked','gate_accepted') THEN 0 ELSE 1 END, w.id DESC
		LIMIT 1`, batchNo).
		Scan(&gateID, &farmerID, &partyName, &partyMobile, &origin, &channel, &farmerName,
			&productID, &variety, &grade, &unitPrice, &plate, &recvAddr)
	if err != nil || gateID <= 0 {
		return nil, "GATE_BINDING_REQUIRED"
	}
	if farmerName == "" {
		farmerName = partyName
	}
	// farmer master origin / mobile as fallback
	if farmerID > 0 {
		var fo, fm string
		_ = s.DB.QueryRow(`SELECT COALESCE(origin,''), COALESCE(mobile,'') FROM pur_farmer WHERE id=?`, farmerID).Scan(&fo, &fm)
		if origin == "" {
			origin = fo
		}
		if partyMobile == "" {
			partyMobile = fm
		}
	}
	// resolve variety_id from catalog by name / product
	var varietyID int64
	if productID > 0 {
		_ = s.DB.QueryRow(`SELECT id FROM pur_weigh_variety WHERE default_product_id=? AND status='active' ORDER BY id LIMIT 1`, productID).Scan(&varietyID)
	}
	if varietyID <= 0 && variety != "" {
		_ = s.DB.QueryRow(`SELECT id FROM pur_weigh_variety WHERE name=? AND status='active' ORDER BY id LIMIT 1`, variety).Scan(&varietyID)
	}
	return gin.H{
		"gate_ticket_id":  gateID,
		"farmer_id":       farmerID,
		"farmer_name":     farmerName,
		"party_name":      partyName,
		"party_mobile":    partyMobile,
		"origin":          origin,
		"channel":         channel,
		"product_id":      productID,
		"variety_id":      varietyID,
		"variety":         variety,
		"grade":           grade,
		"unit_price":      unitPrice,
		"plate_no":        plate,
		"receive_address": recvAddr,
	}, ""
}

// validateTraceBatchCode checks format+pool. If ticketID>0 allow already-used-by-same-ticket.
// On success returns status string; on failure returns error code.
func (s *Services) validateTraceBatchCode(code string, ticketID int64) (bool, string) {
	secret := TraceHMACSecret(s.TraceHMACSecret)
	if _, _, _, ok := ParseTraceBatchCode(secret, code); !ok {
		return false, "BATCH_CODE_INVALID"
	}
	var status string
	var wtID int64
	err := s.DB.QueryRow(`SELECT status, COALESCE(weigh_ticket_id,0) FROM pur_trace_batch_code WHERE code=?`, code).
		Scan(&status, &wtID)
	if err != nil {
		return false, "BATCH_CODE_NOT_FOUND"
	}
	if status == "void" {
		return false, "BATCH_CODE_VOID"
	}
	if status == "used" {
		if ticketID > 0 && wtID == ticketID {
			return true, status
		}
		return false, "BATCH_CODE_USED"
	}
	if status != "available" && status != "reserved" {
		return false, "BATCH_CODE_UNAVAILABLE"
	}
	return true, status
}

func (s *Services) occupyTraceBatchCode(code string, ticketID, userID int64) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	s.expireStaleTraceBatchReservations()
	// CAS: available, or reserved by current user, or expired reserved already cleared above
	res, err := s.DB.Exec(`UPDATE pur_trace_batch_code
		SET status='used', weigh_ticket_id=?, used_at=NOW(), reserved_by=NULL, reserved_at=NULL
		WHERE code=? AND (
			status='available'
			OR (status='reserved' AND (reserved_by=? OR COALESCE(reserved_by,0)=0))
		)`, ticketID, code, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("BATCH_CODE_USED")
	}
	return nil
}

func (s *Services) releaseTraceBatchCode(ticketID int64) {
	if ticketID <= 0 {
		return
	}
	_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET status='available', weigh_ticket_id=NULL, used_at=NULL,
		reserved_by=NULL, reserved_at=NULL
		WHERE weigh_ticket_id=? AND status='used'`, ticketID)
}

func (s *Services) voidTraceBatchCode(c *gin.Context) bool {
	body := bindBody(c)
	code := strings.ToUpper(strings.TrimSpace(strOr(body["code"])))
	if code == "" {
		api.FailJSON(c, "CODE_REQUIRED")
		return true
	}
	res, err := s.DB.Exec(`UPDATE pur_trace_batch_code SET status='void' WHERE code=? AND status='available'`, code)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "BATCH_CODE_NOT_AVAILABLE")
		return true
	}
	api.OK(c, gin.H{"code": code, "status": "void"})
	return true
}
