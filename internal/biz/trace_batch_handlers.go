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

func traceBatchStatusLabel(st string) string {
	switch strings.ToLower(strings.TrimSpace(st)) {
	case "available":
		return "未启用"
	case "reserved":
		return "预占"
	case "in_progress", "used":
		return "过站中"
	case "ended":
		return "已结束"
	case "void":
		return "作废"
	default:
		return st
	}
}

func isTraceBatchInProgress(st string) bool {
	st = strings.ToLower(strings.TrimSpace(st))
	return st == "in_progress" || st == "used"
}

func (s *Services) handleTraceBatchCodes(c *gin.Context, method, action string) bool {
	path := c.Request.URL.Path
	switch {
	case strings.Contains(path, "/generate") || action == "action:generate":
		return s.generateTraceBatchCodes(c)
	case strings.Contains(path, "/validate") || action == "action:validate":
		return s.validateTraceBatchCodeAPI(c)
	case strings.Contains(path, "/void") || action == "action:void":
		return s.voidTraceBatchCode(c)
	case strings.Contains(path, "/end") || action == "action:end":
		return s.endTraceBatchCode(c)
	case action == "list" || (method == "GET" && !strings.Contains(path, "/{") && !strings.Contains(c.FullPath(), ":id")):
		return s.listTraceBatchCodes(c)
	}
	return false
}

func (s *Services) listTraceBatchCodes(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	farmerID, _ := asInt64(c.Query("farmer_id"))
	status := strings.TrimSpace(c.Query("status"))
	if status == "used" {
		status = "in_progress"
	}
	lot := padLot2(c.Query("lot_no"))
	usableOnly := strings.EqualFold(strings.TrimSpace(c.Query("usable")), "1") ||
		strings.EqualFold(strings.TrimSpace(c.Query("usable")), "true")

	where := `WHERE 1=1`
	args := []interface{}{}
	if farmerID > 0 {
		// 倒查：池表锁定农户，或曾挂过该农户的 gate 过磅单
		where += ` AND (COALESCE(c.farmer_id,0)=? OR EXISTS (
			SELECT 1 FROM pur_weigh_ticket w
			WHERE UPPER(w.batch_no)=UPPER(c.code)
			  AND LOWER(COALESCE(w.receive_kind,''))='gate'
			  AND COALESCE(w.is_deleted,0)=0
			  AND COALESCE(w.farmer_id,0)=?
		))`
		args = append(args, farmerID, farmerID)
		if bd := strings.TrimSpace(c.Query("biz_date")); bd != "" {
			where += ` AND c.biz_date=?`
			args = append(args, normalizeBizDate(bd))
		}
	} else {
		bizDate := normalizeBizDate(strOrDef(c.Query("biz_date"), time.Now().Format("2006-01-02")))
		where += ` AND c.biz_date=?`
		args = append(args, bizDate)
	}
	if usableOnly {
		where += ` AND c.status IN ('in_progress','used')`
	}
	if status != "" {
		where += ` AND c.status=?`
		args = append(args, status)
	}
	if strings.TrimSpace(c.Query("lot_no")) != "" {
		where += ` AND c.lot_no=?`
		args = append(args, lot)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_trace_batch_code c `+where, args...).Scan(&total)
	orderBy := `c.seq_no ASC`
	if farmerID > 0 {
		orderBy = `CASE WHEN c.status IN ('in_progress','used') THEN 0 WHEN c.status='ended' THEN 1 ELSE 2 END, COALESCE(c.used_at,c.created_at) DESC, c.id DESC`
	}
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT c.id, c.code, c.biz_date, c.seq_no, c.lot_no, c.status,
		COALESCE(c.weigh_ticket_id,0), COALESCE(c.first_weigh_ticket_id,0),
		COALESCE(c.farmer_id,0), COALESCE(f.name,''), COALESCE(c.product_id,0), COALESCE(p.name,''),
		COALESCE(c.variety,''), c.created_at, COALESCE(c.used_at,''), COALESCE(c.ended_at,'')
		FROM pur_trace_batch_code c
		LEFT JOIN pur_farmer f ON f.id=c.farmer_id
		LEFT JOIN prd_product p ON p.id=c.product_id
		`+where+` ORDER BY `+orderBy+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, seq, wtID, firstID, fid, productID int64
		var code, bd, lotNo, st, farmerName, productName, variety, created, used, ended string
		_ = rows.Scan(&id, &code, &bd, &seq, &lotNo, &st, &wtID, &firstID,
			&fid, &farmerName, &productID, &productName, &variety, &created, &used, &ended)
		canAppend := isTraceBatchInProgress(st)
		list = append(list, gin.H{
			"id": id, "code": code, "biz_date": bd, "seq_no": seq, "lot_no": lotNo,
			"status": st, "status_label": traceBatchStatusLabel(st),
			"can_append": canAppend, "selectable": canAppend,
			"weigh_ticket_id": wtID, "first_weigh_ticket_id": firstID,
			"farmer_id": fid, "farmer_name": farmerName,
			"product_id": productID, "product_name": productName, "variety": variety,
			"created_at": created, "used_at": used, "ended_at": ended,
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

func (s *Services) mergeTraceBindingInto(out gin.H, code string) {
	bind, errCode := s.resolveGateBindingByBatch(code)
	if errCode != "" {
		// pool-level lock as fallback
		var farmerID, productID int64
		var variety, farmerName, productName string
		_ = s.DB.QueryRow(`SELECT COALESCE(c.farmer_id,0), COALESCE(f.name,''), COALESCE(c.product_id,0), COALESCE(p.name,''), COALESCE(c.variety,'')
			FROM pur_trace_batch_code c
			LEFT JOIN pur_farmer f ON f.id=c.farmer_id
			LEFT JOIN prd_product p ON p.id=c.product_id
			WHERE c.code=?`, code).Scan(&farmerID, &farmerName, &productID, &productName, &variety)
		if farmerID > 0 {
			out["farmer_id"] = farmerID
			out["farmer_name"] = farmerName
			out["party_name"] = farmerName
			out["product_id"] = productID
			out["product_name"] = productName
			out["variety"] = variety
			out["binding_locked"] = true
		}
		return
	}
	for k, v := range bind {
		out[k] = v
	}
	out["binding_locked"] = true
}

// reserveTraceBatchCodeForGate atomically reserves an available pool code for gate inbound,
// or returns locked binding when code is already in_progress (append allowed).
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
	switch {
	case status == "void":
		return nil, "BATCH_CODE_VOID"
	case status == "ended":
		return nil, "BATCH_CODE_ENDED"
	case isTraceBatchInProgress(status):
		out := gin.H{
			"code": code, "valid": true, "status": "in_progress", "status_label": "过站中",
			"receive_kind": "gate", "can_append": true,
		}
		s.mergeTraceBindingInto(out, code)
		return out, ""
	case status == "reserved":
		if userID > 0 && reservedBy == userID {
			_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET reserved_at=NOW() WHERE code=? AND status='reserved'`, code)
			return gin.H{
				"code": code, "valid": true, "status": "reserved", "status_label": "预占",
				"receive_kind": "gate", "expires_in_sec": int(traceBatchReserveTTL.Seconds()),
			}, ""
		}
		return nil, "BATCH_CODE_RESERVED"
	case status == "available":
		if userID <= 0 {
			return gin.H{"code": code, "valid": true, "status": "available", "status_label": "未启用", "receive_kind": "gate"}, ""
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
			"code": code, "valid": true, "status": "reserved", "status_label": "预占",
			"receive_kind": "gate", "expires_in_sec": int(traceBatchReserveTTL.Seconds()),
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

// validateTraceBatchForStockin allows in_progress/ended codes that were bound at gate entry.
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
	if status == "available" || status == "reserved" {
		return nil, "GATE_BINDING_REQUIRED"
	}
	bind, errCode := s.resolveGateBindingByBatch(code)
	if errCode != "" {
		return nil, errCode
	}
	out := gin.H{
		"code": code, "valid": true, "status": status, "status_label": traceBatchStatusLabel(status),
		"receive_kind": "stockin",
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

// resolveGateBindingByBatch returns farmer/product binding from pool lock or latest gate weigh ticket.
func (s *Services) resolveGateBindingByBatch(batchNo string) (gin.H, string) {
	batchNo = strings.ToUpper(strings.TrimSpace(batchNo))
	if batchNo == "" {
		return nil, "BATCH_NO_REQUIRED"
	}
	// Prefer pool-level lock when present
	var lockFarmer, lockProduct, firstTicket int64
	var lockVariety string
	_ = s.DB.QueryRow(`SELECT COALESCE(farmer_id,0), COALESCE(product_id,0), COALESCE(variety,''), COALESCE(first_weigh_ticket_id,0)
		FROM pur_trace_batch_code WHERE code=?`, batchNo).Scan(&lockFarmer, &lockProduct, &lockVariety, &firstTicket)

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
		if lockFarmer > 0 {
			var fn string
			_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pur_farmer WHERE id=?`, lockFarmer).Scan(&fn)
			farmerID, farmerName, variety, productID = lockFarmer, fn, lockVariety, lockProduct
			gateID = firstTicket
		} else {
			return nil, "GATE_BINDING_REQUIRED"
		}
	}
	if lockFarmer > 0 {
		farmerID = lockFarmer
		if lockVariety != "" {
			variety = lockVariety
		}
		if lockProduct > 0 {
			productID = lockProduct
		}
		var fn string
		_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM pur_farmer WHERE id=?`, lockFarmer).Scan(&fn)
		if fn != "" {
			farmerName = fn
		}
	}
	if farmerName == "" {
		farmerName = partyName
	}
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

// validateTraceBatchCode checks format+pool. Allows available/reserved/in_progress; rejects ended/void.
func (s *Services) validateTraceBatchCode(code string, ticketID int64) (bool, string) {
	_ = ticketID
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
	if status == "ended" {
		return false, "BATCH_CODE_ENDED"
	}
	if isTraceBatchInProgress(status) {
		return true, "in_progress"
	}
	if status != "available" && status != "reserved" {
		return false, "BATCH_CODE_UNAVAILABLE"
	}
	return true, status
}

func (s *Services) occupyTraceBatchCode(code string, ticketID, userID, farmerID, productID int64, variety string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	s.expireStaleTraceBatchReservations()
	var st string
	var lockFarmer, lockProduct int64
	var lockVariety string
	err := s.DB.QueryRow(`SELECT status, COALESCE(farmer_id,0), COALESCE(product_id,0), COALESCE(variety,'')
		FROM pur_trace_batch_code WHERE code=?`, code).Scan(&st, &lockFarmer, &lockProduct, &lockVariety)
	if err != nil {
		return fmt.Errorf("BATCH_CODE_NOT_FOUND")
	}
	if st == "ended" {
		return fmt.Errorf("BATCH_CODE_ENDED")
	}
	if st == "void" {
		return fmt.Errorf("BATCH_CODE_VOID")
	}
	if isTraceBatchInProgress(st) {
		if lockFarmer > 0 && farmerID > 0 && farmerID != lockFarmer {
			return fmt.Errorf("TRACE_FARMER_LOCKED")
		}
		if lockProduct > 0 && productID > 0 && productID != lockProduct {
			return fmt.Errorf("TRACE_PRODUCT_LOCKED")
		}
		if lockVariety != "" && variety != "" && !strings.EqualFold(strings.TrimSpace(variety), strings.TrimSpace(lockVariety)) {
			return fmt.Errorf("TRACE_PRODUCT_LOCKED")
		}
		res, err := s.DB.Exec(`UPDATE pur_trace_batch_code
			SET status='in_progress', weigh_ticket_id=?, used_at=NOW(), reserved_by=NULL, reserved_at=NULL
			WHERE code=? AND status IN ('in_progress','used')`, ticketID, code)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("BATCH_CODE_UNAVAILABLE")
		}
		return nil
	}
	// First occupy: available or reserved by current user
	res, err := s.DB.Exec(`UPDATE pur_trace_batch_code
		SET status='in_progress', weigh_ticket_id=?, first_weigh_ticket_id=COALESCE(NULLIF(first_weigh_ticket_id,0),?),
			farmer_id=?, product_id=?, variety=?, used_at=NOW(), reserved_by=NULL, reserved_at=NULL
		WHERE code=? AND (
			status='available'
			OR (status='reserved' AND (reserved_by=? OR COALESCE(reserved_by,0)=0))
		)`, ticketID, ticketID, nullIf0(farmerID), nullIf0(productID), variety, code, userID)
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
	var code, st string
	err := s.DB.QueryRow(`SELECT code, status FROM pur_trace_batch_code WHERE weigh_ticket_id=? OR first_weigh_ticket_id=? ORDER BY id LIMIT 1`,
		ticketID, ticketID).Scan(&code, &st)
	if err != nil || code == "" {
		// fallback: match by ticket column only
		_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET status='available', weigh_ticket_id=NULL, used_at=NULL,
			reserved_by=NULL, reserved_at=NULL, farmer_id=NULL, product_id=NULL, variety=NULL, first_weigh_ticket_id=NULL
			WHERE weigh_ticket_id=? AND status IN ('in_progress','used')`, ticketID)
		return
	}
	if st == "ended" || st == "void" {
		return
	}
	var remain int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_weigh_ticket
		WHERE UPPER(batch_no)=? AND LOWER(COALESCE(receive_kind,''))='gate'
		  AND COALESCE(is_deleted,0)=0 AND id<>?`, code, ticketID).Scan(&remain)
	if remain > 0 {
		// keep binding; point last ticket to another remaining gate ticket
		var otherID int64
		_ = s.DB.QueryRow(`SELECT id FROM pur_weigh_ticket
			WHERE UPPER(batch_no)=? AND LOWER(COALESCE(receive_kind,''))='gate'
			  AND COALESCE(is_deleted,0)=0 AND id<>? ORDER BY id DESC LIMIT 1`, code, ticketID).Scan(&otherID)
		if otherID > 0 {
			_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET weigh_ticket_id=? WHERE code=?`, otherID, code)
		}
		return
	}
	_, _ = s.DB.Exec(`UPDATE pur_trace_batch_code SET status='available', weigh_ticket_id=NULL, used_at=NULL,
		reserved_by=NULL, reserved_at=NULL, farmer_id=NULL, product_id=NULL, variety=NULL,
		first_weigh_ticket_id=NULL, ended_at=NULL, ended_by=NULL
		WHERE code=? AND status IN ('in_progress','used')`, code)
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
	api.OK(c, gin.H{"code": code, "status": "void", "status_label": "作废"})
	return true
}

// endTraceBatchCodeInternal locks a trace after warehouse stock-in completes (or admin end).
// Idempotent: already-ended codes return ok=true without error.
func (s *Services) endTraceBatchCodeInternal(code string, userID int64) (ok bool, errCode string) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return false, "CODE_REQUIRED"
	}
	var st string
	err := s.DB.QueryRow(`SELECT status FROM pur_trace_batch_code WHERE code=?`, code).Scan(&st)
	if err != nil {
		return false, "BATCH_CODE_NOT_FOUND"
	}
	if st == "ended" {
		return true, ""
	}
	if st == "void" {
		return false, "BATCH_CODE_VOID"
	}
	res, err := s.DB.Exec(`UPDATE pur_trace_batch_code SET status='ended', ended_at=NOW(), ended_by=?
		WHERE code=? AND status IN ('in_progress','used')`, nullIf0(userID), code)
	if err != nil {
		return false, "DB_ERROR:" + err.Error()
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, "BATCH_CODE_NOT_IN_PROGRESS"
	}
	return true, ""
}

func (s *Services) endTraceBatchCode(c *gin.Context) bool {
	body := bindBody(c)
	code := strings.ToUpper(strings.TrimSpace(strOr(body["code"])))
	if code == "" {
		api.FailJSON(c, "CODE_REQUIRED")
		return true
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	ok, errCode := s.endTraceBatchCodeInternal(code, uid)
	if !ok {
		api.FailJSON(c, errCode)
		return true
	}
	api.OK(c, gin.H{"code": code, "status": "ended", "status_label": "已结束"})
	return true
}
