package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/notify"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) handleProcessReturns(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasSuffix(openapiPath, "/submit") && method == "POST":
		return s.submitProcessReturn(c)
	case strings.HasSuffix(openapiPath, "/approve") && method == "POST":
		return s.approveProcessReturn(c)
	case strings.HasSuffix(openapiPath, "/reject") && method == "POST":
		return s.rejectProcessReturn(c)
	case strings.HasSuffix(openapiPath, "/transfer") && method == "POST":
		return s.transferProcessReturn(c)
	case strings.HasSuffix(openapiPath, "/warehouse-confirm") && method == "POST":
		return s.warehouseConfirmProcessReturn(c)
	case action == "list" || (method == "GET" && !strings.Contains(openapiPath, "{id}")):
		return s.listProcessReturns(c)
	case action == "create" || method == "POST":
		return s.createProcessReturn(c)
	case action == "get" || method == "GET":
		return s.getProcessReturn(c)
	default:
		return false
	}
}

func (s *Services) listProcessReturns(c *gin.Context) bool {
	status := strings.TrimSpace(c.Query("status"))
	q := `SELECT * FROM pd_process_return WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	if status != "" {
		q += ` AND status=?`
		args = append(args, status)
	}
	if mine := c.Query("mine"); mine == "1" || mine == "true" {
		if cl := middleware.Claims(c); cl != nil {
			q += ` AND (applicant_user_id=? OR current_assignee_user_id=? OR foreman_user_id=? OR warehouse_user_id=?)`
			args = append(args, cl.UserID, cl.UserID, cl.UserID, cl.UserID)
		}
	}
	pageNum, pageSize := sqlutil.Page(c)
	countSQL := "SELECT COUNT(1) FROM (" + q + ") t"
	var total int
	_ = s.DB.QueryRow(countSQL, args...).Scan(&total)
	args2 := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(q+` ORDER BY id DESC LIMIT ? OFFSET ?`, args2...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) getProcessReturn(c *gin.Context) bool {
	id := paramID(c)
	m := s.loadProcessReturn(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	rows, err := s.DB.Query(`SELECT id, action, COALESCE(from_user_id,0), COALESCE(to_user_id,0), COALESCE(remark,''), created_at
		FROM pd_process_return_log WHERE return_id=? ORDER BY id`, id)
	logs := []gin.H{}
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var lid, fromU, toU int64
			var act, remark, created string
			_ = rows.Scan(&lid, &act, &fromU, &toU, &remark, &created)
			logs = append(logs, gin.H{"id": lid, "action": act, "from_user_id": fromU, "to_user_id": toU, "remark": remark, "created_at": created})
		}
	}
	m["logs"] = logs
	m["returnable_weight"] = s.processReturnAvailable(strOr(m["box_code"]), id)
	api.OK(c, m)
	return true
}

func (s *Services) loadProcessReturn(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM pd_process_return WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
	if err != nil || rows == nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	return gin.H(list[0])
}

func (s *Services) processReturnAvailable(boxCode string, excludeID int64) float64 {
	if boxCode == "" {
		return 0
	}
	var boxW float64
	_ = s.DB.QueryRow(`SELECT COALESCE(NULLIF(weight,0), COALESCE(qty,0), 0) FROM inv_box_code WHERE code=?`, boxCode).Scan(&boxW)
	var issued float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.qty),0) FROM inv_stock_txn t
		JOIN inv_stock_txn_line l ON l.txn_id=t.id
		WHERE l.direction='out' AND t.doc_type IN ('consume','requisition_out','produce_out')
		  AND (t.remark LIKE ? OR t.remark LIKE ?)`, "%:"+boxCode, "%"+boxCode+"%").Scan(&issued)
	capW := boxW
	if issued > 0 {
		capW = issued
		var consumed float64
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(COALESCE(NULLIF(input_weight,0), qty)),0) FROM pd_report_work
			WHERE scan_code=? AND status='posted'`, boxCode).Scan(&consumed)
		if consumed > 0 && consumed < capW {
			capW = capW - consumed
		}
	}
	var used float64
	if excludeID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(return_weight),0) FROM pd_process_return
			WHERE box_code=? AND COALESCE(is_deleted,0)=0 AND id!=?
			  AND status IN ('draft','pending_foreman','pending_warehouse','posted')`, boxCode, excludeID).Scan(&used)
	} else {
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(return_weight),0) FROM pd_process_return
			WHERE box_code=? AND COALESCE(is_deleted,0)=0
			  AND status IN ('draft','pending_foreman','pending_warehouse','posted')`, boxCode).Scan(&used)
	}
	avail := capW - used
	if avail < 0 {
		avail = 0
	}
	return avail
}

func (s *Services) createProcessReturn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "piece", "fixed", "foreman", "line_worker", "warehouse") {
		return true
	}
	body := bindBody(c)
	box := strings.TrimSpace(strOr(body["box_code"]))
	if box == "" {
		api.FailJSON(c, "BOX_CODE_REQUIRED")
		return true
	}
	retW, _ := asFloat(body["return_weight"])
	if retW <= 0 {
		api.FailJSON(c, "RETURN_WEIGHT_REQUIRED")
		return true
	}
	var productID, whID, processID, stepID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0), COALESCE(warehouse_id,0), COALESCE(current_process_id,0), COALESCE(current_step_id,0)
		FROM inv_box_code WHERE code=?`, box).Scan(&productID, &whID, &processID, &stepID)
	if productID == 0 {
		api.FailJSON(c, "BOX_NOT_FOUND")
		return true
	}
	if v, ok := asInt64(body["warehouse_id"]); ok && v > 0 {
		whID = v
	}
	if whID <= 0 {
		whID = 1
	}
	if v, ok := asInt64(body["process_id"]); ok && v > 0 {
		processID = v
	}
	if v, ok := asInt64(body["step_id"]); ok && v > 0 {
		stepID = v
	}
	avail := s.processReturnAvailable(box, 0)
	if retW > avail+1e-6 && avail > 0 {
		api.FailJSON(c, fmt.Sprintf("RETURN_WEIGHT_EXCEEDS:%.3f", avail))
		return true
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	docNo := strOrDef(body["doc_no"], fmt.Sprintf("PR%s", time.Now().Format("060102150405")))
	reportID, _ := asInt64(body["report_work_id"])
	res, err := s.DB.Exec(`INSERT INTO pd_process_return(doc_no, box_code, process_id, step_id, warehouse_id, return_weight, reason, status,
		applicant_user_id, current_assignee_user_id, report_work_id, remark)
		VALUES(?,?,?,?,?,?,?,'draft',?,?,?,?)`,
		docNo, box, nullIf0(processID), nullIf0(stepID), whID, retW, strOrDef(body["reason"], "提前下班"),
		nullIf0(uid), nullIf0(uid), nullIf0(reportID), strOr(body["remark"]))
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	s.logProcessReturn(id, "create", uid, uid, strOr(body["reason"]))
	api.OK(c, s.loadProcessReturn(id))
	return true
}

func (s *Services) submitProcessReturn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "piece", "fixed", "foreman", "line_worker", "warehouse") {
		return true
	}
	id := paramID(c)
	m := s.loadProcessReturn(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "draft" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	foreman := s.firstUserIDByRoleCode("foreman")
	if foreman <= 0 {
		foreman = uid
	}
	_, err := s.DB.Exec(`UPDATE pd_process_return SET status='pending_foreman', foreman_user_id=?, current_assignee_user_id=?, updated_at=datetime('now') WHERE id=?`,
		nullIf0(foreman), nullIf0(foreman), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.logProcessReturn(id, "submit", uid, foreman, "")
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "production.process_return_submit", BizType: "process_return", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["box_code"]),
			FromRole: "piece", ToRoles: []string{"foreman"}, CreateTask: true,
			Payload: gin.H{"box_code": m["box_code"], "return_weight": m["return_weight"], "reason": m["reason"]},
		})
	}
	api.OK(c, s.loadProcessReturn(id))
	return true
}

func (s *Services) approveProcessReturn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman", "admin") {
		return true
	}
	id := paramID(c)
	m := s.loadProcessReturn(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "pending_foreman" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	whUser := s.firstUserIDByRoleCode("warehouse")
	if whUser <= 0 {
		whUser = uid
	}
	_, err := s.DB.Exec(`UPDATE pd_process_return SET status='pending_warehouse', foreman_user_id=?, warehouse_user_id=?, current_assignee_user_id=?, updated_at=datetime('now') WHERE id=?`,
		nullIf0(uid), nullIf0(whUser), nullIf0(whUser), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.logProcessReturn(id, "approve", uid, whUser, strOr(bindBody(c)["remark"]))
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "production.process_return_approve", BizType: "process_return", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["box_code"]),
			FromRole: "foreman", ToRoles: []string{"warehouse"}, CreateTask: true,
			Payload: gin.H{"box_code": m["box_code"], "return_weight": m["return_weight"]},
		})
	}
	api.OK(c, s.loadProcessReturn(id))
	return true
}

func (s *Services) rejectProcessReturn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "foreman", "warehouse", "admin") {
		return true
	}
	id := paramID(c)
	m := s.loadProcessReturn(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(m["status"])
	if st != "pending_foreman" && st != "pending_warehouse" && st != "draft" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	body := bindBody(c)
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	_, err := s.DB.Exec(`UPDATE pd_process_return SET status='rejected', remark=COALESCE(NULLIF(?,''),remark), current_assignee_user_id=NULL, updated_at=datetime('now') WHERE id=?`,
		strOr(body["remark"]), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.logProcessReturn(id, "reject", uid, 0, strOr(body["remark"]))
	api.OK(c, s.loadProcessReturn(id))
	return true
}

func (s *Services) transferProcessReturn(c *gin.Context) bool {
	id := paramID(c)
	m := s.loadProcessReturn(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(m["status"])
	if st != "pending_foreman" && st != "pending_warehouse" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	body := bindBody(c)
	toUID, _ := asInt64(body["to_user_id"])
	if toUID <= 0 {
		api.FailJSON(c, "TO_USER_REQUIRED")
		return true
	}
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	// 转交不跳过阶段：仅换当前处理人
	if st == "pending_foreman" {
		_, _ = s.DB.Exec(`UPDATE pd_process_return SET foreman_user_id=?, current_assignee_user_id=?, updated_at=datetime('now') WHERE id=?`,
			toUID, toUID, id)
	} else {
		_, _ = s.DB.Exec(`UPDATE pd_process_return SET warehouse_user_id=?, current_assignee_user_id=?, updated_at=datetime('now') WHERE id=?`,
			toUID, toUID, id)
	}
	s.logProcessReturn(id, "transfer", uid, toUID, strOr(body["remark"]))
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "production.process_return_transfer", BizType: "process_return", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: strOr(m["box_code"]),
			FromRole: "foreman", ToRoles: []string{"foreman", "warehouse"}, CreateTask: true,
			Payload: gin.H{"to_user_id": toUID, "status": st, "box_code": m["box_code"]},
		})
	}
	api.OK(c, s.loadProcessReturn(id))
	return true
}

func (s *Services) warehouseConfirmProcessReturn(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "admin") {
		return true
	}
	id := paramID(c)
	m := s.loadProcessReturn(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "pending_warehouse" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	box := strOr(m["box_code"])
	retW, _ := asFloat(m["return_weight"])
	whID, _ := asInt64(m["warehouse_id"])
	if whID <= 0 {
		whID = 1
	}
	var productID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0) FROM inv_box_code WHERE code=?`, box).Scan(&productID)
	if productID == 0 {
		productID = 1
	}
	docNo := fmt.Sprintf("PRIN%s", time.Now().Format("060102150405"))
	tid, err := s.insertPostedStockTxn("process_return", whID, time.Now().Format("2006-01-02"), docNo,
		[]txnLine{{pid: productID, qty: retW, dir: "in"}}, fmt.Sprintf("process_return:%d:%s", id, box))
	if err != nil {
		api.FailJSON(c, "STOCK_ERROR:"+err.Error())
		return true
	}
	// 箱码挂回仓库，次日可再出库；不回冲已确认计件
	_, _ = s.DB.Exec(`UPDATE inv_box_code SET warehouse_id=?, qty=?, weight=?, status='open', updated_at=datetime('now') WHERE code=?`,
		whID, retW, retW, box)
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	_, err = s.DB.Exec(`UPDATE pd_process_return SET status='posted', stock_txn_id=?, warehouse_user_id=?, current_assignee_user_id=NULL,
		posted_at=datetime('now'), updated_at=datetime('now') WHERE id=?`, tid, nullIf0(uid), id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.logProcessReturn(id, "warehouse_confirm", uid, 0, "posted")
	s.writeAuditCtx(c, "process_return", id, "warehouse_confirm", "unused_material_return", m, s.loadProcessReturn(id))
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "production.process_return_posted", BizType: "process_return", BizID: id,
			DocNo: strOr(m["doc_no"]), TraceCode: box,
			FromRole: "warehouse", ToRoles: []string{"foreman", "piece"}, CreateTask: false,
			Payload: gin.H{"box_code": box, "return_weight": retW, "stock_txn_id": tid, "piecework_reversed": false},
		})
	}
	out := s.loadProcessReturn(id)
	out["piecework_reversed"] = false
	out["stock_txn_id"] = tid
	api.OK(c, out)
	return true
}

func (s *Services) logProcessReturn(returnID int64, action string, fromUID, toUID int64, remark string) {
	_, _ = s.DB.Exec(`INSERT INTO pd_process_return_log(return_id, action, from_user_id, to_user_id, remark) VALUES(?,?,?,?,?)`,
		returnID, action, nullIf0(fromUID), nullIf0(toUID), remark)
}
