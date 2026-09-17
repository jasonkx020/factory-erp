package biz

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/notify"
	"erp/internal/pay"
	"erp/internal/persistence/sqlutil"
)

func (s *Services) ensurePaymentOrderTable() {
	if s == nil || s.DB == nil {
		return
	}
	_, _ = s.DB.Exec(`CREATE TABLE IF NOT EXISTS fin_payment_order (
  id BIGSERIAL PRIMARY KEY,
  payment_no TEXT NOT NULL UNIQUE,
  settlement_id INTEGER NOT NULL,
  supplier_id INTEGER NOT NULL,
  amount DOUBLE PRECISION NOT NULL DEFAULT 0,
  currency TEXT NOT NULL DEFAULT 'CNY',
  channel TEXT NOT NULL DEFAULT 'alipay_bank',
  status TEXT NOT NULL DEFAULT 'pending_finance',
  payee_name TEXT,
  payee_bank_account TEXT,
  payee_bank_name TEXT,
  payee_mobile TEXT,
  channel_trade_no TEXT,
  fail_reason TEXT,
  fund_account_id INTEGER,
  finance_approved_by INTEGER,
  finance_approved_at TEXT,
  boss_approved_by INTEGER,
  boss_approved_at TEXT,
  paid_at TEXT,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
)`)
}

func (s *Services) paymentSettings() map[string]interface{} {
	m := s.loadSysSetting("payment")
	if m == nil {
		m = map[string]interface{}{}
	}
	return m
}

func (s *Services) onlinePayEnabled() bool {
	return boolish(s.paymentSettings()["online_enabled"])
}

func boolish(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true") || t == "yes"
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return false
	}
}

func floatish(v interface{}, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		var f float64
		_, _ = fmt.Sscanf(t, "%f", &f)
		return f
	default:
		return def
	}
}

func strMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	return strings.TrimSpace(strOr(m[key]))
}

func nestedMap(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func (s *Services) paymentDefaultChannel() string {
	ch := strMap(s.paymentSettings(), "default_channel")
	if ch == "" {
		return pay.ChannelAlipayBank
	}
	return ch
}

func (s *Services) requireBossApprove(amount float64) bool {
	m := s.paymentSettings()
	if v, ok := m["require_boss_approve"].(bool); ok && !v {
		return false
	}
	th := floatish(m["boss_amount_threshold"], 0)
	if th > 0 && amount < th {
		return false
	}
	// default: require boss when online pay settings present
	if m["require_boss_approve"] == nil {
		return true
	}
	return boolish(m["require_boss_approve"])
}

func (s *Services) supplierPayeeInfo(supplierID int64) (name, bankName, bankAcc, mobile string, errCode string) {
	var n, bn, ba, mob string
	err := s.DB.QueryRow(`SELECT COALESCE(name,''), COALESCE(bank_name,''), COALESCE(bank_account,''),
		COALESCE(mobile,'') FROM pur_supplier WHERE id=? AND COALESCE(is_deleted,0)=0`, supplierID).
		Scan(&n, &bn, &ba, &mob)
	if err != nil {
		return "", "", "", "", "SUPPLIER_NOT_FOUND"
	}
	if strings.TrimSpace(ba) == "" || strings.TrimSpace(n) == "" {
		return n, bn, ba, mob, "PAYEE_BANK_REQUIRED"
	}
	return n, bn, ba, mob, ""
}

func (s *Services) maybeCreatePaymentOrderForSettlement(settlementID int64) (int64, string) {
	if !s.onlinePayEnabled() || settlementID <= 0 {
		return 0, ""
	}
	s.ensurePaymentOrderTable()
	var exist int64
	_ = s.DB.QueryRow(`SELECT id FROM fin_payment_order WHERE settlement_id=? AND status NOT IN ('cancelled','failed','paid') ORDER BY id DESC LIMIT 1`, settlementID).Scan(&exist)
	if exist > 0 {
		return exist, ""
	}
	var supplierID int64
	var amount float64
	var status string
	if err := s.DB.QueryRow(`SELECT supplier_id, amount, status FROM pur_supplier_settlement WHERE id=?`, settlementID).
		Scan(&supplierID, &amount, &status); err != nil {
		return 0, "SETTLEMENT_NOT_FOUND"
	}
	if status == "settle_paid" || status == "paid" || status == "void" {
		return 0, ""
	}
	payee, bankName, bankAcc, mobile, errCode := s.supplierPayeeInfo(supplierID)
	channel := s.paymentDefaultChannel()
	alipay := nestedMap(s.paymentSettings(), "alipay")
	if strMap(alipay, "app_id") == "" && (channel == pay.ChannelAlipayBank || channel == "") {
		channel = pay.ChannelMock
	}
	paymentNo := fmt.Sprintf("PO%s%04d", time.Now().Format("060102150405"), settlementID%10000)
	remark := errCode
	res, err := s.DB.Exec(`INSERT INTO fin_payment_order(
		payment_no, settlement_id, supplier_id, amount, channel, status,
		payee_name, payee_bank_account, payee_bank_name, payee_mobile, remark)
		VALUES(?,?,?,?,?,'pending_finance',?,?,?,?,?)`,
		paymentNo, settlementID, supplierID, amount, channel,
		payee, bankAcc, bankName, mobile, remark)
	if err != nil {
		return 0, "PAYMENT_ORDER_CREATE_FAILED:" + err.Error()
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		_ = s.DB.QueryRow(`SELECT id FROM fin_payment_order WHERE payment_no=?`, paymentNo).Scan(&id)
	}
	if s.Notify != nil {
		s.Notify.NotifyNext(nil, notify.Event{
			Key: "finance.payment_pending", BizType: "payment_order", BizID: id,
			DocNo: paymentNo, FromRole: "system", ToRoles: []string{"finance"}, CreateTask: true,
			Title: "供应商货款待财务审批", Body: fmt.Sprintf("支付单 %s 金额 %.2f 待审批", paymentNo, amount),
			Payload: gin.H{"settlement_id": settlementID, "amount": amount},
		})
	}
	return id, ""
}

func (s *Services) handlePaymentOrders(c *gin.Context, method, openapiPath, action string) bool {
	s.ensurePaymentOrderTable()
	path := c.Request.URL.Path
	switch {
	case strings.Contains(path, "/alipay/notify") || strings.HasSuffix(action, "alipay-notify"):
		return s.handleAlipayPayNotify(c)
	case strings.HasSuffix(action, "approve-finance") || strings.Contains(path, "/approve-finance"):
		return s.approvePaymentFinance(c)
	case strings.HasSuffix(action, "approve-boss") || strings.Contains(path, "/approve-boss"):
		return s.approvePaymentBoss(c)
	case strings.HasSuffix(action, "reject") || strings.Contains(path, "/reject"):
		return s.rejectPaymentOrder(c)
	case strings.HasSuffix(action, "retry") || strings.Contains(path, "/retry"):
		return s.retryPaymentOrder(c)
	case action == "list" || (method == http.MethodGet && !strings.Contains(openapiPath, "{id}")):
		return s.listPaymentOrders(c)
	case action == "get" || method == http.MethodGet:
		return s.getPaymentOrder(c)
	case action == "create" || method == http.MethodPost:
		return s.createPaymentOrderAPI(c)
	}
	api.FailJSON(c, "UNKNOWN_ACTION")
	return true
}

func (s *Services) listPaymentOrders(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if st := c.Query("status"); st != "" {
		where += ` AND o.status=?`
		args = append(args, st)
	}
	if sid := c.Query("settlement_id"); sid != "" {
		where += ` AND o.settlement_id=?`
		args = append(args, sid)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM fin_payment_order o `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT o.id, o.payment_no, o.settlement_id, o.supplier_id, COALESCE(f.name,''),
		o.amount, o.channel, o.status, COALESCE(o.payee_name,''), COALESCE(o.payee_bank_account,''),
		COALESCE(o.payee_mobile,''), COALESCE(o.channel_trade_no,''), COALESCE(o.fail_reason,''),
		o.created_at, COALESCE(o.paid_at,'')
		FROM fin_payment_order o LEFT JOIN pur_supplier f ON f.id=o.supplier_id
		`+where+` ORDER BY o.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, sid, supplierID int64
		var pno, sname, channel, status, payee, bankAcc, mobile, tradeNo, fail, created, paidAt string
		var amount float64
		_ = rows.Scan(&id, &pno, &sid, &supplierID, &sname, &amount, &channel, &status, &payee, &bankAcc, &mobile, &tradeNo, &fail, &created, &paidAt)
		list = append(list, gin.H{
			"id": id, "payment_no": pno, "settlement_id": sid, "supplier_id": supplierID, "supplier_name": sname,
			"amount": amount, "channel": channel, "status": status, "payee_name": payee,
			"payee_bank_account": bankAcc, "payee_mobile": mobile, "channel_trade_no": tradeNo,
			"fail_reason": fail, "created_at": created, "paid_at": paidAt,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) loadPaymentOrder(id int64) gin.H {
	rows, err := s.DB.Query(`SELECT * FROM fin_payment_order WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	if len(list) == 0 {
		return nil
	}
	return gin.H(list[0])
}

func (s *Services) getPaymentOrder(c *gin.Context) bool {
	m := s.loadPaymentOrder(paramID(c))
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) createPaymentOrderAPI(c *gin.Context) bool {
	if !s.requireAnyRole(c, "finance", "sys_admin") {
		return true
	}
	if !s.onlinePayEnabled() {
		api.FailJSON(c, "ONLINE_PAY_DISABLED")
		return true
	}
	body := bindBody(c)
	sid, _ := asInt64(body["settlement_id"])
	if sid <= 0 {
		api.FailJSON(c, "SETTLEMENT_ID_REQUIRED")
		return true
	}
	id, errMsg := s.maybeCreatePaymentOrderForSettlement(sid)
	if id == 0 {
		api.FailJSON(c, strOrDef(errMsg, "PAYMENT_ORDER_CREATE_FAILED"))
		return true
	}
	api.OK(c, s.loadPaymentOrder(id))
	return true
}

func (s *Services) approvePaymentFinance(c *gin.Context) bool {
	if !s.requireAnyRole(c, "finance") {
		return true
	}
	id := paramID(c)
	m := s.loadPaymentOrder(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "pending_finance" {
		api.FailJSON(c, "INVALID_STATUS:"+strOr(m["status"]))
		return true
	}
	if strOr(m["payee_bank_account"]) == "" || strOr(m["payee_name"]) == "" {
		payee, bn, ba, mob, errCode := s.supplierPayeeInfo(asInt64Or0(m["supplier_id"]))
		if errCode != "" {
			api.FailJSON(c, errCode)
			return true
		}
		_, _ = s.DB.Exec(`UPDATE fin_payment_order SET payee_name=?, payee_bank_account=?, payee_bank_name=?, payee_mobile=?, updated_at=NOW() WHERE id=?`,
			payee, ba, bn, mob, id)
	}
	uid := int64(0)
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	amount := asFloatOr0(m["amount"])
	next := "pending_boss"
	if !s.requireBossApprove(amount) {
		next = "ready_to_pay"
	}
	if _, err := s.DB.Exec(`UPDATE fin_payment_order SET status=?, finance_approved_by=?, finance_approved_at=NOW(), updated_at=NOW() WHERE id=?`,
		next, uid, id); err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	if s.Notify != nil {
		s.Notify.CompleteTask("payment_order", id)
		if next == "pending_boss" {
			s.Notify.NotifyNext(c, notify.Event{
				Key: "finance.payment_boss", BizType: "payment_order", BizID: id,
				DocNo: strOr(m["payment_no"]), FromRole: "finance", ToRoles: []string{"boss", "sys_admin"}, CreateTask: true,
				Title: "供应商货款待总经理审批", Body: fmt.Sprintf("支付单 %s 金额 %.2f", strOr(m["payment_no"]), amount),
			})
		}
	}
	if next == "ready_to_pay" {
		s.dispatchPaymentOrder(id)
	}
	api.OK(c, s.loadPaymentOrder(id))
	return true
}

func (s *Services) approvePaymentBoss(c *gin.Context) bool {
	if !s.requireAnyRole(c, "boss", "sys_admin") {
		return true
	}
	id := paramID(c)
	m := s.loadPaymentOrder(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "pending_boss" {
		api.FailJSON(c, "INVALID_STATUS:"+strOr(m["status"]))
		return true
	}
	uid := int64(0)
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	if _, err := s.DB.Exec(`UPDATE fin_payment_order SET status='ready_to_pay', boss_approved_by=?, boss_approved_at=NOW(), updated_at=NOW() WHERE id=?`,
		uid, id); err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	if s.Notify != nil {
		s.Notify.CompleteTask("payment_order", id)
	}
	s.dispatchPaymentOrder(id)
	api.OK(c, s.loadPaymentOrder(id))
	return true
}

func (s *Services) rejectPaymentOrder(c *gin.Context) bool {
	if !s.requireAnyRole(c, "finance", "boss", "sys_admin") {
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	reason := strOrDef(body["reason"], strOr(body["remark"]))
	m := s.loadPaymentOrder(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(m["status"])
	if st == "paid" || st == "paying" {
		api.FailJSON(c, "CANNOT_REJECT:"+st)
		return true
	}
	_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='cancelled', fail_reason=?, updated_at=NOW() WHERE id=?`, reason, id)
	if s.Notify != nil {
		s.Notify.CompleteTask("payment_order", id)
	}
	api.OK(c, s.loadPaymentOrder(id))
	return true
}

func (s *Services) retryPaymentOrder(c *gin.Context) bool {
	if !s.requireAnyRole(c, "finance", "sys_admin") {
		return true
	}
	id := paramID(c)
	m := s.loadPaymentOrder(id)
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if strOr(m["status"]) != "failed" {
		api.FailJSON(c, "INVALID_STATUS")
		return true
	}
	sid := asInt64Or0(m["settlement_id"])
	_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='cancelled', updated_at=NOW() WHERE id=?`, id)
	newID, errMsg := s.maybeCreatePaymentOrderForSettlement(sid)
	if newID == 0 {
		api.FailJSON(c, strOrDef(errMsg, "PAYMENT_ORDER_CREATE_FAILED"))
		return true
	}
	api.OK(c, s.loadPaymentOrder(newID))
	return true
}

func (s *Services) buildPayRegistry() *pay.Registry {
	cfg := s.paymentSettings()
	alipay := nestedMap(cfg, "alipay")
	ag := &pay.AlipayBankGateway{Cfg: pay.AlipayConfig{
		AppID:           strMap(alipay, "app_id"),
		PrivateKeyPEM:   strMap(alipay, "private_key_pem"),
		AlipayPublicKey: strMap(alipay, "alipay_public_key"),
		NotifyURL:       strMap(alipay, "notify_url"),
		GatewayURL:      strMap(alipay, "gateway_url"),
	}}
	bank := &pay.BankDirectGateway{BankCode: strMap(nestedMap(cfg, "bank_direct"), "bank_code")}
	return pay.NewRegistry(ag, bank, pay.MockGateway{})
}

func (s *Services) dispatchPaymentOrder(id int64) {
	m := s.loadPaymentOrder(id)
	if m == nil {
		return
	}
	st := strOr(m["status"])
	if st != "ready_to_pay" && st != "paying" {
		return
	}
	_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='paying', updated_at=NOW() WHERE id=?`, id)
	channel := strOr(m["channel"])
	if channel == "" {
		channel = s.paymentDefaultChannel()
	}
	reg := s.buildPayRegistry()
	gw, err := reg.Get(channel)
	if err != nil {
		gw, err = reg.Get(pay.ChannelMock)
		if err != nil {
			_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='failed', fail_reason=?, updated_at=NOW() WHERE id=?`, err.Error(), id)
			return
		}
		channel = pay.ChannelMock
	}
	if ag, ok := gw.(*pay.AlipayBankGateway); ok && strings.TrimSpace(ag.Cfg.AppID) == "" {
		gw = pay.MockGateway{}
		channel = pay.ChannelMock
		_, _ = s.DB.Exec(`UPDATE fin_payment_order SET channel=? WHERE id=?`, channel, id)
	}
	res, err := gw.Transfer(context.Background(), pay.TransferRequest{
		PaymentNo:    strOr(m["payment_no"]),
		Amount:       asFloatOr0(m["amount"]),
		Currency:     strOrDef(m["currency"], "CNY"),
		PayeeName:    strOr(m["payee_name"]),
		PayeeAccount: strOr(m["payee_bank_account"]),
		PayeeBank:    strOr(m["payee_bank_name"]),
		Remark:       "供应商货款 " + strOr(m["payment_no"]),
	})
	if err != nil {
		_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='failed', fail_reason=?, channel_trade_no=COALESCE(NULLIF(?,''),channel_trade_no), updated_at=NOW() WHERE id=?`,
			err.Error(), res.ChannelTradeNo, id)
		return
	}
	_, _ = s.DB.Exec(`UPDATE fin_payment_order SET channel_trade_no=?, updated_at=NOW() WHERE id=?`, res.ChannelTradeNo, id)
	if res.SyncPaid {
		_ = s.completePaymentOrderPaid(id, res.ChannelTradeNo, 0)
	}
}

func (s *Services) completePaymentOrderPaid(paymentID int64, channelTradeNo string, fundAccountID int64) error {
	m := s.loadPaymentOrder(paymentID)
	if m == nil {
		return api.Fail("NOT_FOUND")
	}
	if strOr(m["status"]) == "paid" {
		return nil
	}
	sid := asInt64Or0(m["settlement_id"])
	transferNo := channelTradeNo
	if transferNo == "" {
		transferNo = strOr(m["payment_no"])
	}
	s.ensureFinanceCashColumns()
	if err := s.postFarmerSettlementCash(sid, fundAccountID, transferNo); err != nil {
		if !strings.Contains(fmt.Sprint(err), "ALREADY_PAID") {
			_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='failed', fail_reason=?, updated_at=NOW() WHERE id=?`, err.Error(), paymentID)
			return err
		}
	}
	_, _ = s.DB.Exec(`UPDATE pur_supplier_settlement SET status='settle_paid', transfer_no=?, paid_at=NOW() WHERE id=? AND COALESCE(status,'') NOT IN ('settle_paid','paid')`,
		transferNo, sid)
	_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='paid', channel_trade_no=COALESCE(NULLIF(?,''),channel_trade_no), paid_at=NOW(), updated_at=NOW() WHERE id=?`,
		channelTradeNo, paymentID)
	if s.Notify != nil {
		s.Notify.CompleteTask("payment_order", paymentID)
		s.Notify.CompleteTask("farmer_settlement", sid)
		sm := s.loadSettlement(sid)
		s.Notify.NotifyNext(nil, notify.Event{
			Key: "purchase.settle_paid", BizType: "farmer_settlement", BizID: sid,
			DocNo: strOr(sm["doc_no"]), FromRole: "finance", ToRoles: []string{"purchase"}, CreateTask: false,
			Payload: gin.H{"transfer_no": transferNo, "amount": m["amount"], "payment_no": m["payment_no"]},
		})
	}
	s.sendPaymentSuccessSMS(paymentID)
	return nil
}

func (s *Services) sendPaymentSuccessSMS(paymentID int64) {
	smsCfg := nestedMap(s.paymentSettings(), "sms")
	if !boolish(smsCfg["enabled"]) {
		return
	}
	m := s.loadPaymentOrder(paymentID)
	if m == nil {
		return
	}
	mobile := strOr(m["payee_mobile"])
	if mobile == "" {
		return
	}
	acc := strOr(m["payee_bank_account"])
	tail := acc
	if len(tail) > 4 {
		tail = tail[len(tail)-4:]
	}
	var sender pay.SmsSender = &pay.LogSmsSender{}
	if strings.EqualFold(strMap(smsCfg, "provider"), "aliyun") || strMap(smsCfg, "provider") == "" {
		if strMap(smsCfg, "access_key") != "" {
			sender = &pay.AliyunSmsSender{Cfg: pay.AliyunSmsConfig{
				AccessKeyID:     strMap(smsCfg, "access_key"),
				AccessKeySecret: strMap(smsCfg, "access_secret"),
				SignName:        strMap(smsCfg, "sign_name"),
				TemplateCode:    strMap(smsCfg, "template_pay_success"),
			}}
		}
	}
	_ = sender.Send(context.Background(), mobile, strMap(smsCfg, "template_pay_success"), map[string]string{
		"name":   strOr(m["payee_name"]),
		"amount": fmt.Sprintf("%.2f", asFloatOr0(m["amount"])),
		"no":     strOr(m["payment_no"]),
		"tail":   tail,
	})
}

func (s *Services) handleAlipayPayNotify(c *gin.Context) bool {
	_ = c.Request.ParseForm()
	values := c.Request.Form
	cfg := nestedMap(s.paymentSettings(), "alipay")
	pub := strMap(cfg, "alipay_public_key")
	if pub != "" && !pay.VerifyAlipayNotify(values, pub) {
		c.String(http.StatusBadRequest, "fail")
		return true
	}
	outBiz := values.Get("out_biz_no")
	if outBiz == "" {
		outBiz = values.Get("out_trade_no")
	}
	status := values.Get("status")
	if status == "" {
		status = values.Get("trade_status")
	}
	orderID := values.Get("order_id")
	if orderID == "" {
		orderID = values.Get("trade_no")
	}
	var id int64
	_ = s.DB.QueryRow(`SELECT id FROM fin_payment_order WHERE payment_no=?`, outBiz).Scan(&id)
	if id <= 0 {
		c.String(http.StatusOK, "success")
		return true
	}
	okStatus := strings.EqualFold(status, "SUCCESS") || strings.EqualFold(status, "TRADE_SUCCESS") || status == ""
	if okStatus {
		_ = s.completePaymentOrderPaid(id, orderID, 0)
	} else {
		_, _ = s.DB.Exec(`UPDATE fin_payment_order SET status='failed', fail_reason=?, updated_at=NOW() WHERE id=? AND status<>'paid'`,
			status, id)
	}
	c.String(http.StatusOK, "success")
	return true
}

func (s *Services) blockManualPayIfOnlinePending(c *gin.Context, settlementID int64) bool {
	if !s.onlinePayEnabled() {
		return false
	}
	s.ensurePaymentOrderTable()
	var st string
	_ = s.DB.QueryRow(`SELECT status FROM fin_payment_order WHERE settlement_id=? AND status NOT IN ('cancelled','failed') ORDER BY id DESC LIMIT 1`, settlementID).Scan(&st)
	if st == "" {
		return false
	}
	if st == "paid" {
		api.FailJSON(c, "ALREADY_PAID")
		return true
	}
	body := bindBody(c)
	if boolish(body["force_manual"]) {
		return false
	}
	api.FailJSON(c, "ONLINE_PAY_PENDING:"+st)
	return true
}

var _ = sql.ErrNoRows
