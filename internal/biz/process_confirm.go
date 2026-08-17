package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/notify"
)

func (s *Services) loadReportWork(id int64) gin.H {
	var dispatchID, woID, processID, workerID int64
	var docNo, status, reported, scan, snap, qc, confirmedAt string
	var qty, weight, inW, outW, loss, util float64
	var confirmedBy, opUserID, opEmpID int64
	err := s.DB.QueryRow(`SELECT doc_no, COALESCE(dispatch_id,0), COALESCE(work_order_id,0), process_id, worker_id,
		qty, COALESCE(weight,0), COALESCE(input_weight,0), COALESCE(output_weight,0), COALESCE(loss,0), COALESCE(utilization,0),
		status, COALESCE(reported_at,''), COALESCE(scan_code,''), COALESCE(confirmed_by,0), COALESCE(confirmed_at,''),
		COALESCE(confirmed_snapshot_json,''), COALESCE(process_qc_result,''),
		COALESCE(operator_user_id,0), COALESCE(operator_employee_id,0)
		FROM pd_report_work WHERE id=?`, id).
		Scan(&docNo, &dispatchID, &woID, &processID, &workerID, &qty, &weight, &inW, &outW, &loss, &util,
			&status, &reported, &scan, &confirmedBy, &confirmedAt, &snap, &qc, &opUserID, &opEmpID)
	if err != nil {
		return nil
	}
	var workerName, opName string
	_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM hr_employee WHERE id=?`, workerID).Scan(&workerName)
	if opEmpID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(name,'') FROM hr_employee WHERE id=?`, opEmpID).Scan(&opName)
	} else if opUserID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(e.name,u.login_name,'') FROM iam_user u
			LEFT JOIN hr_employee e ON e.id=u.employee_id WHERE u.id=?`, opUserID).Scan(&opName)
	}
	return gin.H{
		"id": id, "doc_no": docNo, "dispatch_id": dispatchID, "work_order_id": woID, "process_id": processID,
		"worker_id": workerID, "worker_name": workerName, "qty": qty, "weight": weight, "input_weight": inW, "output_weight": outW,
		"loss": loss, "utilization": util, "status": status, "reported_at": reported, "scan_code": scan,
		"confirmed_by": confirmedBy, "confirmed_at": confirmedAt, "confirmed_snapshot_json": snap,
		"process_qc_result": qc, "operator_user_id": opUserID, "operator_employee_id": opEmpID, "operator_name": opName,
		"pass_for_other": opEmpID > 0 && workerID != opEmpID,
		"evidences": s.listEvidence("report_work", id),
	}
}

// applyReportWorkConfirm 将待确认报工过账；失败返回错误码（不写 HTTP）。
func (s *Services) applyReportWorkConfirm(c *gin.Context, id int64, body map[string]interface{}) (gin.H, string) {
	m := s.loadReportWork(id)
	if m == nil {
		return nil, "NOT_FOUND"
	}
	if strOr(m["status"]) == "posted" || strOr(m["confirmed_at"]) != "" {
		return nil, "ALREADY_CONFIRMED"
	}
	if strOr(m["status"]) == "void" {
		return nil, "VOIDED"
	}

	inW, _ := asFloat(m["input_weight"])
	outW, _ := asFloat(m["output_weight"])
	if v, ok := asFloat(body["input_weight"]); ok && v > 0 {
		inW = v
	}
	if v, ok := asFloat(body["output_weight"]); ok && v > 0 {
		outW = v
	}
	loss := inW - outW
	if loss < 0 {
		loss = 0
	}
	util := 0.0
	if inW > 0 {
		util = outW / inW
	}

	qc := strings.ToLower(strOrDef(body["process_qc_result"], strOr(m["process_qc_result"])))
	if qc == "" {
		qc = "pass"
	}
	if qc == "合格" {
		qc = "pass"
	}
	if qc == "不合格" {
		qc = "fail"
	}
	if qc != "pass" {
		return nil, "PROCESS_QC_FAIL"
	}
	if img := strOrDef(body["qc_image_url"], strOr(body["image_url"])); img != "" {
		_, _ = s.addEvidence(c, "report_work", id, "process_qc_photo", img, nil)
	}

	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	snap := nowSnap(map[string]interface{}{
		"input_weight": inW, "output_weight": outW, "loss": loss, "utilization": util, "process_qc_result": qc,
		"bag_qty": asFloatOr0(body["bag_qty"]), "scrap_type": strOr(body["scrap_type"]),
	})
	bagQty := asFloatOr0(body["bag_qty"])
	_, err := s.DB.Exec(`UPDATE pd_report_work SET input_weight=?, output_weight=?, qty=?, weight=?, qty_net=?, loss=?, utilization=?,
		process_qc_result=?, confirmed_by=?, confirmed_at=NOW(), confirmed_snapshot_json=?, status='posted', bag_qty=? WHERE id=?`,
		inW, outW, outW, outW, outW, loss, util, qc, uid, snap, bagQty, id)
	if err != nil {
		return nil, "DB_ERROR:" + err.Error()
	}
	if loss > 0 {
		pid, _ := asInt64(m["process_id"])
		scrapType := strOrDef(body["scrap_type"], "process_loss")
		_, _ = s.DB.Exec(`INSERT INTO pd_scrap_record(doc_no, process_id, product_id, qty, weight, disposition, status, scrap_type)
			VALUES(?,?,1,?,?, 'process_loss','recorded',?)`,
			fmt.Sprintf("SCR-%d-%d", id, time.Now().UnixNano()%1e6), pid, loss, loss, scrapType)
		box := strOr(m["scan_code"])
		if box != "" {
			_ = s.writeProcessLossTxn(box, loss)
		}
	}

	processID, _ := asInt64(m["process_id"])
	workerID, _ := asInt64(m["worker_id"])
	woID, _ := asInt64(m["work_order_id"])
	box := strOr(m["scan_code"])
	var taskID, stepID int64
	if woID > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(task_id,0), COALESCE(routing_step_id,0) FROM pd_work_order WHERE id=?`, woID).Scan(&taskID, &stepID)
	}
	if stepID == 0 && box != "" {
		_ = s.DB.QueryRow(`SELECT COALESCE(current_step_id,0), COALESCE(task_id,0) FROM inv_box_code WHERE code=?`, box).Scan(&stepID, &taskID)
	}
	flow := s.AfterReportWork(c, id, processID, workerID, outW, box, taskID, woID, stepID, inW, loss, util)
	s.writeAuditCtx(c, "report_work", id, "confirm", "worker_confirm", m, s.loadReportWork(id))
	out := s.loadReportWork(id)
	for k, v := range flow {
		out[k] = v
	}
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "production.report_confirmed", BizType: "report_work", BizID: id,
			DocNo: strOr(out["doc_no"]), TraceCode: box,
			FromRole: "piece", ToRoles: []string{"foreman"}, CreateTask: true,
			Payload: gin.H{"scan_code": box, "loss": out["loss"], "output_weight": out["output_weight"], "next": flow["next"]},
		})
	}
	return out, ""
}

// confirmReportWork 工人确认报工/定损/工序 QC 后过账工序环。
func (s *Services) confirmReportWork(c *gin.Context) bool {
	if !s.requireAnyRole(c, "piece", "fixed", "foreman", "line_worker") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	out, failCode := s.applyReportWorkConfirm(c, id, body)
	if failCode != "" {
		api.FailJSON(c, failCode)
		return true
	}
	api.OK(c, out)
	return true
}

// voidReportWorkDraft 作废未确认过站草稿（与「领出未用完退库」分开，不计件、无库存回冲）。
func (s *Services) voidReportWorkDraft(c *gin.Context) bool {
	if !s.requireAnyRole(c, "piece", "fixed", "foreman", "line_worker", "admin") {
		return true
	}
	id := paramID(c)
	m := s.loadReportWork(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(m["status"])
	if st != "confirm_pending" && st != "draft" && st != "submitted" {
		api.FailJSON(c, "ONLY_DRAFT_VOIDABLE")
		return true
	}
	if strOr(m["confirmed_at"]) != "" || st == "posted" {
		api.FailJSON(c, "ALREADY_CONFIRMED")
		return true
	}
	body := bindBody(c)
	_, err := s.DB.Exec(`UPDATE pd_report_work SET status='void' WHERE id=? AND status IN ('confirm_pending','draft','submitted')`, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	s.writeAuditCtx(c, "report_work", id, "void", strOrDef(body["remark"], "void_draft"), m, s.loadReportWork(id))
	api.OK(c, gin.H{"id": id, "status": "void"})
	return true
}

func (s *Services) writeProcessLossTxn(boxCode string, loss float64) error {
	var productID, wh int64
	_ = s.DB.QueryRow(`SELECT product_id, COALESCE(warehouse_id,1) FROM inv_box_code WHERE code=?`, boxCode).Scan(&productID, &wh)
	if productID == 0 {
		productID = 1
	}
	docNo := fmt.Sprintf("LOSS%d", time.Now().UnixNano()%1e12)
	res, err := s.DB.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark) VALUES(?,?,date('now'),'posted',?,?)`,
		docNo, "process_loss", wh, "loss:"+boxCode)
	if err != nil {
		return err
	}
	tid, _ := res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, direction) VALUES(?,?,?,?,?,'out')`,
		tid, 1, productID, loss, loss)
	return s.adjustBalance(wh, productID, -loss)
}

// payLaborSummary 劳动结算支付：须转账单号 + 回单证据。
func (s *Services) payLaborSummary(c *gin.Context) bool {
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
	var amount float64
	err := s.DB.QueryRow(`SELECT COALESCE(status,'open'), COALESCE(amount,0) FROM pd_piecework_summary WHERE id=?`, id).Scan(&status, &amount)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status == "labor_paid" {
		api.FailJSON(c, "ALREADY_PAID")
		return true
	}
	_, _ = s.addEvidence(c, "piecework_summary", id, "pay_receipt", payURL, gin.H{"transfer_no": transferNo})
	_, err = s.DB.Exec(`UPDATE pd_piecework_summary SET status='labor_paid', transfer_no=?, paid_at=NOW(), pay_evidence_url=? WHERE id=?`,
		transferNo, payURL, id)
	if err != nil {
		// columns may need ensure
		_, err = s.DB.Exec(`UPDATE pd_piecework_summary SET status='labor_paid' WHERE id=?`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
	}
	s.writeAuditCtx(c, "piecework_summary", id, "labor_pay", transferNo, gin.H{"status": status}, gin.H{"status": "labor_paid", "transfer_no": transferNo})
	if s.Notify != nil {
		s.Notify.NotifyNext(c, notify.Event{
			Key: "payroll.labor_paid", BizType: "piecework_summary", BizID: id,
			DocNo: fmt.Sprintf("PW%d", id), FromRole: "finance", ToRoles: []string{"piece", "fixed"},
			CreateTask: false, Payload: gin.H{"transfer_no": transferNo, "amount": amount},
		})
	}
	api.OK(c, gin.H{"id": id, "status": "labor_paid", "transfer_no": transferNo, "amount": amount, "pay_evidence_url": payURL})
	return true
}
