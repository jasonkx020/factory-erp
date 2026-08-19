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

func (s *Services) handleOperationLogs(c *gin.Context, method, action, path string) bool {
	if strings.Contains(path, "/trace/") {
		traceID := c.Param("trace_id")
		rows, err := s.DB.Query(`SELECT id, user_id, action, module, ref_type, ref_id, detail_json, ip, trace_id, created_at
			FROM sys_operation_log WHERE trace_id=? ORDER BY id`, traceID)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var uid, refID interface{}
			var act, mod, refType, detail, ip, tid, created string
			_ = rows.Scan(&id, &uid, &act, &mod, &refType, &refID, &detail, &ip, &tid, &created)
			list = append(list, gin.H{
				"id": id, "user_id": uid, "action": act, "module": mod, "ref_type": refType, "ref_id": refID,
				"detail_json": jsonRawOrStr(detail), "ip": ip, "trace_id": tid, "created_at": created,
			})
		}
		api.OK(c, gin.H{"trace_id": traceID, "list": list, "total": len(list)})
		return true
	}
	if action == "list" {
		pageNum, pageSize := sqlutil.Page(c)
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sys_operation_log`).Scan(&total)
		rows, err := s.DB.Query(`SELECT id, user_id, action, module, ref_type, ref_id, detail_json, ip, COALESCE(trace_id,''), created_at
			FROM sys_operation_log ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, (pageNum-1)*pageSize)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var uid, refID interface{}
			var act, mod, refType, detail, ip, tid, created string
			_ = rows.Scan(&id, &uid, &act, &mod, &refType, &refID, &detail, &ip, &tid, &created)
			list = append(list, gin.H{
				"id": id, "user_id": uid, "action": act, "module": mod, "ref_type": refType, "ref_id": refID,
				"detail_json": jsonRawOrStr(detail), "ip": ip, "trace_id": tid, "created_at": created,
			})
		}
		api.PageOK(c, list, total, pageNum, pageSize)
		return true
	}
	if action == "get" {
		id := paramID(c)
		var uid, refID interface{}
		var act, mod, refType, detail, ip, tid, created string
		err := s.DB.QueryRow(`SELECT user_id, action, module, ref_type, ref_id, detail_json, ip, COALESCE(trace_id,''), created_at
			FROM sys_operation_log WHERE id=?`, id).Scan(&uid, &act, &mod, &refType, &refID, &detail, &ip, &tid, &created)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{
			"id": id, "user_id": uid, "action": act, "module": mod, "ref_type": refType, "ref_id": refID,
			"detail_json": jsonRawOrStr(detail), "ip": ip, "trace_id": tid, "created_at": created,
		})
		return true
	}
	return true
}

func jsonRawOrStr(s string) interface{} {
	if s == "" {
		return nil
	}
	var v interface{}
	if json.Unmarshal([]byte(s), &v) == nil {
		return v
	}
	return s
}

func (s *Services) handleBoxTrace(c *gin.Context) bool {
	code := c.Param("code")
	codes := s.collectBoxFamily(code)
	if len(codes) == 0 {
		codes = []string{code}
	}
	var farmerID, productID int64
	var traceCode, origin, receiveDate, sourceType, status string
	var weight float64
	_ = s.DB.QueryRow(`SELECT COALESCE(farmer_id,0), COALESCE(product_id,0), COALESCE(trace_code,''), COALESCE(origin,''), COALESCE(receive_date,''), COALESCE(source_type,''),
		COALESCE(status,''), COALESCE(weight, qty, 0)
		FROM inv_box_code WHERE code=?`, code).Scan(&farmerID, &productID, &traceCode, &origin, &receiveDate, &sourceType, &status, &weight)
	pname, pcat, pcode := s.productMeta(productID)
	farmer := gin.H{}
	if farmerID > 0 {
		farmer = s.loadFarmer(farmerID)
	}
	likeParts := make([]string, 0, len(codes))
	args := make([]interface{}, 0, len(codes))
	for _, bc := range codes {
		likeParts = append(likeParts, "payload_json LIKE ?")
		args = append(args, "%"+bc+"%")
	}
	q := `SELECT id, source_type, source_id, from_step_id, to_step_id, trigger_action, trace_id, status, error, payload_json, created_at
		FROM pd_flow_event WHERE ` + strings.Join(likeParts, " OR ") + ` ORDER BY id`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer rows.Close()
	events := []gin.H{}
	seen := map[int64]bool{}
	for rows.Next() {
		var id, sourceID int64
		var fromStep, toStep interface{}
		var stype, trigger, tid, status, errmsg, payload, created string
		_ = rows.Scan(&id, &stype, &sourceID, &fromStep, &toStep, &trigger, &tid, &status, &errmsg, &payload, &created)
		if seen[id] {
			continue
		}
		seen[id] = true
		events = append(events, gin.H{
			"id": id, "source_type": stype, "source_id": sourceID, "from_step_id": fromStep, "to_step_id": toStep,
			"trigger": trigger, "trace_id": tid, "status": status, "error": errmsg, "payload": jsonRawOrStr(payload), "created_at": created,
		})
	}
	oplogs := []gin.H{}
	for _, bc := range codes {
		logs, err := s.DB.Query(`SELECT id, action, module, trace_id, created_at FROM sys_operation_log WHERE detail_json LIKE ? ORDER BY id`, "%"+bc+"%")
		if err != nil || logs == nil {
			continue
		}
		for logs.Next() {
			var id int64
			var act, mod, tid, created string
			_ = logs.Scan(&id, &act, &mod, &tid, &created)
			oplogs = append(oplogs, gin.H{"id": id, "action": act, "module": mod, "trace_id": tid, "created_at": created, "box_code": bc})
		}
		logs.Close()
	}
	api.OK(c, gin.H{
		"box_code": code, "code": code, "related_boxes": codes, "flow_events": events, "operation_logs": oplogs,
		"farmer": farmer, "trace_code": traceCode, "origin": origin, "receive_date": receiveDate, "source_type": sourceType,
		"product_id": productID, "product_name": pname, "product_category": pcat, "product_code": pcode,
		"status": status, "weight": weight, "qty": weight,
	})
	return true
}

func (s *Services) collectBoxFamily(code string) []string {
	var id, parentID int64
	err := s.DB.QueryRow(`SELECT id, COALESCE(parent_box_id,0) FROM inv_box_code WHERE code=?`, code).Scan(&id, &parentID)
	if err != nil {
		return nil
	}
	// walk up to root
	rootID := id
	for parentID > 0 {
		rootID = parentID
		var p int64
		if err := s.DB.QueryRow(`SELECT COALESCE(parent_box_id,0) FROM inv_box_code WHERE id=?`, parentID).Scan(&p); err != nil {
			break
		}
		parentID = p
	}
	codes := []string{}
	rows, err := s.DB.Query(`WITH RECURSIVE fam(id) AS (
		SELECT ? UNION ALL SELECT b.id FROM inv_box_code b JOIN fam ON b.parent_box_id = fam.id
	) SELECT code FROM inv_box_code WHERE id IN (SELECT id FROM fam)`, rootID)
	if err != nil {
		// sqlite without recursive fallback: just root + direct children
		_ = s.DB.QueryRow(`SELECT code FROM inv_box_code WHERE id=?`, rootID).Scan(&code)
		codes = append(codes, code)
		ch, _ := s.DB.Query(`SELECT code FROM inv_box_code WHERE parent_box_id=?`, rootID)
		if ch != nil {
			defer ch.Close()
			for ch.Next() {
				var c string
				_ = ch.Scan(&c)
				codes = append(codes, c)
			}
		}
		return codes
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		_ = rows.Scan(&c)
		codes = append(codes, c)
	}
	return codes
}

func (s *Services) handleDataRepair(c *gin.Context, action string) bool {
	if action == "action:apply" {
		claims := middleware.Claims(c)
		if claims == nil {
			api.FailJSON(c, "UNAUTHORIZED")
			return true
		}
		isAdmin := false
		for _, r := range claims.Roles {
			if r == "sys_admin" || r == "系统管理员" {
				isAdmin = true
			}
		}
		if !isAdmin {
			api.FailJSON(c, "PERM_DENIED")
			return true
		}
		id := paramID(c)
		var reason, targetType, act string
		var targetID int64
		err := s.DB.QueryRow(`SELECT reason, target_type, target_id, action FROM sys_data_repair WHERE id=?`, id).
			Scan(&reason, &targetType, &targetID, &act)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if strings.TrimSpace(reason) == "" {
			api.FailJSON(c, "REASON_REQUIRED")
			return true
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		switch act {
		case "reopen_box":
			_, _ = s.DB.Exec(`UPDATE inv_box_code SET status='open' WHERE id=?`, targetID)
		case "retry_flow":
			_, _ = s.DB.Exec(`UPDATE pd_flow_event SET status='ok', error='' WHERE id=?`, targetID)
		default:
			// no-op marker
		}
		_, _ = s.DB.Exec(`UPDATE sys_data_repair SET status='applied', applied_by=?, applied_at=? WHERE id=?`, claims.UserID, now, id)
		api.OK(c, gin.H{"id": id, "status": "applied", "reason": reason})
		return true
	}
	if action == "create" {
		body := bindBody(c)
		reason, _ := body["reason"].(string)
		if strings.TrimSpace(reason) == "" {
			api.FailJSON(c, "REASON_REQUIRED")
			return true
		}
		docNo, _ := body["doc_no"].(string)
		if docNo == "" {
			docNo = fmt.Sprintf("DR%d", time.Now().UnixNano()%1e12)
		}
		targetType, _ := body["target_type"].(string)
		if targetType == "" {
			targetType = "manual"
		}
		targetID, _ := asInt64(body["target_id"])
		act, _ := body["action"].(string)
		if act == "" {
			act, _ = body["action_json"].(string)
		}
		if act == "" {
			act = "noop"
		}
		payload, _ := body["payload_json"].(string)
		var createdBy interface{}
		if claims := middleware.Claims(c); claims != nil {
			createdBy = claims.UserID
		}
		res, err := s.DB.Exec(`INSERT INTO sys_data_repair(doc_no, target_type, target_id, action, reason, status, payload_json, created_by)
			VALUES(?,?,?,?,?,'draft',?,?)`, docNo, targetType, targetID, act, reason, payload, createdBy)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft", "reason": reason, "action": act, "target_type": targetType, "target_id": targetID})
		return true
	}
	return s.handleTableCRUD(c, "system/data-repairs", action)
}
