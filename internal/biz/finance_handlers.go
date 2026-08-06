package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsureFinanceSchema creates factory-delivery finance tables (SQLite).
func EnsureFinanceSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS fin_account_subject (
  id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
  parent_id INTEGER, subject_type TEXT, status TEXT NOT NULL DEFAULT 'active')`,
		`CREATE TABLE IF NOT EXISTS fin_fund_account (
  id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
  currency TEXT NOT NULL DEFAULT 'CNY', balance REAL NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'active')`,
		`CREATE TABLE IF NOT EXISTS fin_fund_transfer (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE,
  from_account_id INTEGER NOT NULL, to_account_id INTEGER NOT NULL, amount REAL NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', remark TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_ledger_entry (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, account_id INTEGER, subject_id INTEGER,
  direction TEXT NOT NULL, amount REAL NOT NULL, biz_date TEXT NOT NULL, counterparty TEXT,
  source_doc_type TEXT, source_doc_id INTEGER, remark TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_income_expense_detail (
  id INTEGER PRIMARY KEY AUTOINCREMENT, entry_id INTEGER NOT NULL, category TEXT, amount REAL NOT NULL, remark TEXT)`,
		`CREATE TABLE IF NOT EXISTS fin_voucher (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, period TEXT, biz_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', summary TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_voucher_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT, voucher_id INTEGER NOT NULL, subject_id INTEGER NOT NULL,
  debit REAL NOT NULL DEFAULT 0, credit REAL NOT NULL DEFAULT 0, remark TEXT)`,
		`CREATE TABLE IF NOT EXISTS fin_invoice (
  id INTEGER PRIMARY KEY AUTOINCREMENT, invoice_no TEXT NOT NULL UNIQUE, direction TEXT NOT NULL,
  counterparty_id INTEGER, counterparty_name TEXT, amount REAL NOT NULL, tax REAL,
  status TEXT NOT NULL DEFAULT 'draft', biz_date TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_receipt_writeoff (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, customer_id INTEGER NOT NULL,
  amount REAL NOT NULL, fund_account_id INTEGER, status TEXT NOT NULL DEFAULT 'draft',
  received_at TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_receipt_writeoff_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT, writeoff_id INTEGER NOT NULL, sales_order_id INTEGER NOT NULL, amount REAL NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS fin_payment_recognition (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, customer_id INTEGER NOT NULL,
  amount REAL NOT NULL, fund_account_id INTEGER, status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_prepay_prepaid (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, party_type TEXT NOT NULL,
  party_id INTEGER NOT NULL, direction TEXT NOT NULL, amount REAL NOT NULL, balance REAL NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_fx_settlement (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, currency TEXT NOT NULL,
  amount_fx REAL NOT NULL, rate REAL NOT NULL, amount_local REAL NOT NULL,
  fund_account_id INTEGER, status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_cost_allocation (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, source_amount REAL NOT NULL,
  alloc_json TEXT, status TEXT NOT NULL DEFAULT 'draft', revoked_from_id INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_receipt_alert (
  id INTEGER PRIMARY KEY AUTOINCREMENT, customer_id INTEGER NOT NULL, order_id INTEGER,
  due_date TEXT, overdue_days INTEGER, amount REAL, status TEXT NOT NULL DEFAULT 'open',
  handled_remark TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_cashier_reconcile (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, fund_account_id INTEGER NOT NULL,
  biz_date TEXT NOT NULL, book_balance REAL NOT NULL, actual_balance REAL NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', remark TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_cost_accounting (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, period TEXT NOT NULL,
  task_id INTEGER, product_id INTEGER, material_cost REAL NOT NULL DEFAULT 0,
  labor_cost REAL NOT NULL DEFAULT 0, overhead REAL NOT NULL DEFAULT 0, total_cost REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_cost_trace_line (
  id INTEGER PRIMARY KEY AUTOINCREMENT, cost_id INTEGER NOT NULL, source_type TEXT NOT NULL,
  source_id INTEGER NOT NULL, amount REAL NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS fin_contract_profit (
  id INTEGER PRIMARY KEY AUTOINCREMENT, contract_id INTEGER NOT NULL, revenue REAL NOT NULL DEFAULT 0,
  cost REAL NOT NULL DEFAULT 0, profit REAL NOT NULL DEFAULT 0, period TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_sales_return_finance (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, order_id INTEGER,
  amount REAL NOT NULL, status TEXT NOT NULL DEFAULT 'draft', created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_arap_adjust (
  id INTEGER PRIMARY KEY AUTOINCREMENT, doc_no TEXT NOT NULL UNIQUE, party_type TEXT NOT NULL,
  party_id INTEGER NOT NULL, amount REAL NOT NULL, direction TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft', remark TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_month_close (
  id INTEGER PRIMARY KEY AUTOINCREMENT, year INTEGER NOT NULL, month INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', closed_at TEXT, closed_by INTEGER, UNIQUE(year, month))`,
		`CREATE TABLE IF NOT EXISTS fin_miniprogram_bill (
  id INTEGER PRIMARY KEY AUTOINCREMENT, bill_no TEXT NOT NULL UNIQUE, channel TEXT,
  amount REAL NOT NULL, status TEXT NOT NULL DEFAULT 'unpaid', order_id INTEGER,
  paid_at TEXT, created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_approval_item (
  id INTEGER PRIMARY KEY AUTOINCREMENT, biz_type TEXT NOT NULL, biz_id INTEGER NOT NULL,
  doc_no TEXT, title TEXT, amount REAL, status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS fin_statement_cache (
  id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT NOT NULL, period TEXT, title TEXT,
  content_json TEXT, generated_at TEXT NOT NULL DEFAULT (datetime('now')))`,
	}
	for _, stmt := range stmts {
		_, _ = db.Exec(stmt)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM fin_account_subject`).Scan(&n)
	if n == 0 {
		for _, s := range [][3]string{
			{"1001", "库存现金", "asset"}, {"1002", "银行存款", "asset"},
			{"1122", "应收账款", "asset"}, {"2202", "应付账款", "liability"},
			{"5001", "主营业务收入", "income"}, {"5401", "主营业务成本", "expense"},
			{"5601", "管理费用", "expense"}, {"5602", "销售费用", "expense"},
		} {
			_, _ = db.Exec(`INSERT OR IGNORE INTO fin_account_subject(code, name, subject_type) VALUES(?,?,?)`, s[0], s[1], s[2])
		}
	}
	_ = db.QueryRow(`SELECT COUNT(1) FROM fin_fund_account`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT OR IGNORE INTO fin_fund_account(code, name, currency, balance) VALUES('CASH','现金','CNY',0)`)
		_, _ = db.Exec(`INSERT OR IGNORE INTO fin_fund_account(code, name, currency, balance) VALUES('BANK','基本户','CNY',0)`)
	}
}

func (s *Services) handleFinanceDomain(c *gin.Context, method, openapiPath, action string) bool {
	// keep supplier party guard for create
	if s.handleFinancePartyGuard(c, method, openapiPath, action) {
		return true
	}
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/finance/account-subjects"):
		return s.handleFinSubjects(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/fund-accounts"):
		return s.handleFinFundAccounts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/fund-transfers"):
		return s.handleFinFundTransfers(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/ledger-entries"):
		return s.handleFinLedger(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/income-expenses"):
		return s.handleFinIncomeExpenses(c)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/vouchers"):
		return s.handleFinVouchers(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/invoices"):
		return s.handleFinInvoices(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/receipt-writeoffs"):
		return s.handleFinWriteoffs(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/fx-settlements"):
		return s.handleFinFX(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/cost-allocations"):
		return s.handleFinAllocations(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/receipt-alerts"):
		return s.handleFinAlerts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/cashier-reconciles"):
		return s.handleFinReconciles(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/prepay-prepaids"):
		return s.handleFinPrepays(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/cost-accountings"):
		return s.handleFinCostAccountings(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/contract-profits"):
		return s.handleFinContractProfits(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/payment-recognitions"):
		return s.handleFinRecognitions(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/sales-return-finances"):
		return s.handleFinReturnFinances(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/arap-adjusts"):
		return s.handleFinArap(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/approvals"):
		return s.handleFinApprovals(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/statements"):
		return s.handleFinStatements(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/cost-traces"):
		return s.handleFinCostTraces(c, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/month-closes"):
		return s.handleFinMonthCloses(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/miniprogram-bills"):
		return s.handleFinMiniprogram(c, method, openapiPath, action)
	case strings.HasPrefix(openapiPath, "/api/v1/finance/orders"):
		return s.handleFinOrders(c, action)
	default:
		return false
	}
}

func finDocNo(prefix string) string {
	return fmt.Sprintf("%s%s", prefix, time.Now().Format("060102150405"))
}

func (s *Services) adjustFundBalance(accountID int64, delta float64) error {
	var bal float64
	err := s.DB.QueryRow(`SELECT balance FROM fin_fund_account WHERE id=?`, accountID).Scan(&bal)
	if err == sql.ErrNoRows {
		return api.Fail("FUND_ACCOUNT_NOT_FOUND")
	}
	if err != nil {
		return err
	}
	nq := bal + delta
	if nq < -0.0001 {
		return api.Fail("INSUFFICIENT_FUND")
	}
	_, err = s.DB.Exec(`UPDATE fin_fund_account SET balance=? WHERE id=?`, nq, accountID)
	return err
}

func (s *Services) insertLedger(accountID, subjectID int64, dir string, amount float64, counterparty, srcType string, srcID int64, remark string) (int64, error) {
	docNo := finDocNo("LE")
	bizDate := time.Now().Format("2006-01-02")
	res, err := s.DB.Exec(`INSERT INTO fin_ledger_entry(doc_no, account_id, subject_id, direction, amount, biz_date, counterparty, source_doc_type, source_doc_id, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, docNo, nullInt(accountID), nullInt(subjectID), dir, amount, bizDate, counterparty, srcType, nullInt(srcID), remark)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ---------- subjects ----------
func (s *Services) handleFinSubjects(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_account_subject`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_account_subject WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		code := strOrDef(body["code"], finDocNo("SUB"))
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO fin_account_subject(code, name, parent_id, subject_type, status) VALUES(?,?,?,?,?)`,
			code, name, nullInt64Or(body["parent_id"]), strOrDef(body["subject_type"], "asset"), strOrDef(body["status"], "active"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_account_subject SET name=COALESCE(NULLIF(?,''),name), subject_type=COALESCE(NULLIF(?,''),subject_type),
			status=COALESCE(NULLIF(?,''),status), parent_id=COALESCE(NULLIF(?,0),parent_id) WHERE id=?`,
			strOr(body["name"]), strOr(body["subject_type"]), strOr(body["status"]), nullInt64Or(body["parent_id"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_account_subject WHERE id=?`, id)
	case "delete":
		_, _ = s.DB.Exec(`DELETE FROM fin_account_subject WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	return true
}

// ---------- fund accounts / transfers ----------
func (s *Services) handleFinFundAccounts(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_fund_account`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_fund_account WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		code := strOrDef(body["code"], finDocNo("FA"))
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		bal, _ := asFloat(body["balance"])
		res, err := s.DB.Exec(`INSERT INTO fin_fund_account(code, name, currency, balance, status) VALUES(?,?,?,?,?)`,
			code, name, strOrDef(body["currency"], "CNY"), bal, strOrDef(body["status"], "active"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "balance": bal})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_fund_account SET name=COALESCE(NULLIF(?,''),name), currency=COALESCE(NULLIF(?,''),currency),
			status=COALESCE(NULLIF(?,''),status) WHERE id=?`, strOr(body["name"]), strOr(body["currency"]), strOr(body["status"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_fund_account WHERE id=?`, id)
	case "delete":
		_, _ = s.DB.Exec(`DELETE FROM fin_fund_account WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	return true
}

func (s *Services) handleFinFundTransfers(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_fund_transfer`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_fund_transfer WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		fromID, _ := asInt64(body["from_account_id"])
		toID, _ := asInt64(body["to_account_id"])
		amt, _ := asFloat(body["amount"])
		if fromID == 0 || toID == 0 || amt <= 0 || fromID == toID {
			api.FailJSON(c, "INVALID_TRANSFER")
			return true
		}
		docNo := strOrDef(body["doc_no"], finDocNo("FT"))
		res, err := s.DB.Exec(`INSERT INTO fin_fund_transfer(doc_no, from_account_id, to_account_id, amount, status, remark) VALUES(?,?,?,?,'draft',?)`,
			docNo, fromID, toID, amt, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_fund_transfer SET remark=COALESCE(NULLIF(?,''),remark), amount=COALESCE(?,amount) WHERE id=? AND status='draft'`,
			strOr(body["remark"]), nullFloat(body["amount"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_fund_transfer WHERE id=?`, id)
	case "action:post":
		id := paramID(c)
		var fromID, toID int64
		var amt float64
		var status string
		err := s.DB.QueryRow(`SELECT from_account_id, to_account_id, amount, status FROM fin_fund_transfer WHERE id=?`, id).
			Scan(&fromID, &toID, &amt, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "posted" {
			return s.getSimpleRow(c, `SELECT * FROM fin_fund_transfer WHERE id=?`, id)
		}
		if err := s.adjustFundBalance(fromID, -amt); err != nil {
			api.FailJSON(c, err.Error())
			return true
		}
		if err := s.adjustFundBalance(toID, amt); err != nil {
			_ = s.adjustFundBalance(fromID, amt) // rollback best-effort
			api.FailJSON(c, err.Error())
			return true
		}
		_, _ = s.insertLedger(fromID, 0, "out", amt, "", "fund_transfer", id, "资金调出")
		_, _ = s.insertLedger(toID, 0, "in", amt, "", "fund_transfer", id, "资金调入")
		_, _ = s.DB.Exec(`UPDATE fin_fund_transfer SET status='posted' WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_fund_transfer WHERE id=?`, id)
	}
	return true
}

// ---------- ledger / income-expense ----------
func (s *Services) handleFinLedger(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_ledger_entry`)
	case "get":
		id := paramID(c)
		rows, _ := s.DB.Query(`SELECT * FROM fin_ledger_entry WHERE id=?`, id)
		if rows == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m := gin.H(list[0])
		drows, _ := s.DB.Query(`SELECT * FROM fin_income_expense_detail WHERE entry_id=?`, id)
		details := []map[string]interface{}{}
		if drows != nil {
			defer drows.Close()
			details, _ = rowsToMaps(drows)
		}
		m["details"] = details
		api.OK(c, m)
		return true
	case "create":
		body := bindBody(c)
		dir := strOrDef(body["direction"], "in")
		amt, _ := asFloat(body["amount"])
		if amt <= 0 {
			api.FailJSON(c, "AMOUNT_REQUIRED")
			return true
		}
		accountID, _ := asInt64(body["account_id"])
		subjectID, _ := asInt64(body["subject_id"])
		docNo := strOrDef(body["doc_no"], finDocNo("LE"))
		bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		res, err := s.DB.Exec(`INSERT INTO fin_ledger_entry(doc_no, account_id, subject_id, direction, amount, biz_date, counterparty, remark)
			VALUES(?,?,?,?,?,?,?,?)`, docNo, nullInt(accountID), nullInt(subjectID), dir, amt, bizDate, strOr(body["counterparty"]), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		if accountID > 0 {
			delta := amt
			if dir == "out" {
				delta = -amt
			}
			_ = s.adjustFundBalance(accountID, delta)
		}
		cat := strOrDef(body["category"], dir)
		_, _ = s.DB.Exec(`INSERT INTO fin_income_expense_detail(entry_id, category, amount, remark) VALUES(?,?,?,?)`,
			id, cat, amt, strOr(body["remark"]))
		api.OK(c, gin.H{"id": id, "doc_no": docNo})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_ledger_entry SET counterparty=COALESCE(NULLIF(?,''),counterparty), remark=COALESCE(NULLIF(?,''),remark) WHERE id=?`,
			strOr(body["counterparty"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_ledger_entry WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleFinIncomeExpenses(c *gin.Context) bool {
	rows, err := s.DB.Query(`SELECT d.id, d.entry_id, d.category, d.amount, COALESCE(d.remark,''),
		e.doc_no, e.direction, e.biz_date, COALESCE(e.counterparty,'')
		FROM fin_income_expense_detail d
		JOIN fin_ledger_entry e ON e.id=d.entry_id
		ORDER BY d.id DESC`)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, entryID int64
		var cat, remark, docNo, dir, bizDate, cp string
		var amt float64
		_ = rows.Scan(&id, &entryID, &cat, &amt, &remark, &docNo, &dir, &bizDate, &cp)
		list = append(list, gin.H{
			"id": id, "entry_id": entryID, "category": cat, "amount": amt, "remark": remark,
			"doc_no": docNo, "direction": dir, "biz_date": bizDate, "counterparty": cp,
		})
	}
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

// ---------- vouchers ----------
func (s *Services) handleFinVouchers(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_voucher`)
	case "get":
		id := paramID(c)
		rows, _ := s.DB.Query(`SELECT * FROM fin_voucher WHERE id=?`, id)
		if rows == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m := gin.H(list[0])
		lrows, _ := s.DB.Query(`SELECT * FROM fin_voucher_line WHERE voucher_id=?`, id)
		lines := []map[string]interface{}{}
		if lrows != nil {
			defer lrows.Close()
			lines, _ = rowsToMaps(lrows)
		}
		m["lines"] = lines
		api.OK(c, m)
		return true
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], finDocNo("VCH"))
		bizDate := strOrDef(body["biz_date"], time.Now().Format("2006-01-02"))
		period := strOrDef(body["period"], time.Now().Format("2006-01"))
		if s.finPeriodClosed(period) {
			api.FailJSON(c, "PERIOD_CLOSED")
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO fin_voucher(doc_no, period, biz_date, status, summary) VALUES(?,?,?,'draft',?)`,
			docNo, period, bizDate, strOr(body["summary"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		vid, _ := res.LastInsertId()
		lines, _ := body["lines"].([]interface{})
		if len(lines) == 0 {
			sid, _ := asInt64(body["subject_id"])
			debit, _ := asFloat(body["debit"])
			credit, _ := asFloat(body["credit"])
			if sid == 0 {
				sid = 1
			}
			_, _ = s.DB.Exec(`INSERT INTO fin_voucher_line(voucher_id, subject_id, debit, credit, remark) VALUES(?,?,?,?,?)`,
				vid, sid, debit, credit, strOr(body["remark"]))
			if debit > 0 || credit > 0 {
				// balancing line stub if only one side
				if debit > 0 && credit == 0 {
					_, _ = s.DB.Exec(`INSERT INTO fin_voucher_line(voucher_id, subject_id, debit, credit, remark) VALUES(?,?,0,?,'自动平衡')`,
						vid, sid, debit)
				}
			}
		} else {
			for _, ln := range lines {
				m, _ := ln.(map[string]interface{})
				if m == nil {
					continue
				}
				sid, _ := asInt64(m["subject_id"])
				debit, _ := asFloat(m["debit"])
				credit, _ := asFloat(m["credit"])
				_, _ = s.DB.Exec(`INSERT INTO fin_voucher_line(voucher_id, subject_id, debit, credit, remark) VALUES(?,?,?,?,?)`,
					vid, sid, debit, credit, strOr(m["remark"]))
			}
		}
		api.OK(c, gin.H{"id": vid, "doc_no": docNo, "status": "draft"})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_voucher SET summary=COALESCE(NULLIF(?,''),summary) WHERE id=? AND status='draft'`,
			strOr(body["summary"]), id)
		return s.handleFinVouchers(c, "GET", "get")
	case "action:approve":
		id := paramID(c)
		if s.finPeriodClosedForVoucher(id) {
			api.FailJSON(c, "PERIOD_CLOSED")
			return true
		}
		var debit, credit float64
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(debit),0), COALESCE(SUM(credit),0) FROM fin_voucher_line WHERE voucher_id=?`, id).Scan(&debit, &credit)
		if debit-credit > 0.01 || credit-debit > 0.01 {
			api.FailJSON(c, "VOUCHER_UNBALANCED")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE fin_voucher SET status='approved' WHERE id=? AND status='draft'`, id)
		_, _ = s.DB.Exec(`INSERT INTO fin_approval_item(biz_type, biz_id, doc_no, title, amount, status)
			SELECT 'voucher', id, doc_no, COALESCE(summary,''), ?, 'approved' FROM fin_voucher WHERE id=?`, debit, id)
		api.OK(c, gin.H{"id": id, "status": "approved", "debit": debit, "credit": credit})
		return true
	case "action:post":
		id := paramID(c)
		if s.finPeriodClosedForVoucher(id) {
			api.FailJSON(c, "PERIOD_CLOSED")
			return true
		}
		var status string
		_ = s.DB.QueryRow(`SELECT status FROM fin_voucher WHERE id=?`, id).Scan(&status)
		if status != "approved" && status != "draft" {
			api.FailJSON(c, "INVALID_STATUS")
			return true
		}
		var debit, credit float64
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(debit),0), COALESCE(SUM(credit),0) FROM fin_voucher_line WHERE voucher_id=?`, id).Scan(&debit, &credit)
		if debit-credit > 0.01 || credit-debit > 0.01 {
			api.FailJSON(c, "VOUCHER_UNBALANCED")
			return true
		}
		if debit <= 0 {
			api.FailJSON(c, "VOUCHER_EMPTY")
			return true
		}
		_, _ = s.DB.Exec(`UPDATE fin_voucher SET status='posted' WHERE id=?`, id)
		s.writeAuditCtx(c, "fin_voucher", id, "post", "", gin.H{"status": status}, gin.H{"status": "posted", "debit": debit, "credit": credit})
		api.OK(c, gin.H{"id": id, "status": "posted", "debit": debit, "credit": credit})
		return true
	}
	return true
}

// ---------- invoices ----------
func (s *Services) handleFinInvoices(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_invoice`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_invoice WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		no := strOrDef(body["invoice_no"], finDocNo("INV"))
		amt, _ := asFloat(body["amount"])
		res, err := s.DB.Exec(`INSERT INTO fin_invoice(invoice_no, direction, counterparty_id, counterparty_name, amount, tax, status, biz_date)
			VALUES(?,?,?,?,?,?,?,?)`,
			no, strOrDef(body["direction"], "out"), nullInt64Or(body["counterparty_id"]), strOr(body["counterparty_name"]),
			amt, nullFloat(body["tax"]), strOrDef(body["status"], "draft"),
			strOrDef(body["biz_date"], time.Now().Format("2006-01-02")))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "invoice_no": no})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_invoice SET amount=COALESCE(?,amount), tax=COALESCE(?,tax), status=COALESCE(NULLIF(?,''),status),
			counterparty_name=COALESCE(NULLIF(?,''),counterparty_name) WHERE id=?`,
			nullFloat(body["amount"]), nullFloat(body["tax"]), strOr(body["status"]), strOr(body["counterparty_name"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_invoice WHERE id=?`, id)
	case "delete":
		_, _ = s.DB.Exec(`DELETE FROM fin_invoice WHERE id=?`, paramID(c))
		api.OK(c, gin.H{})
		return true
	}
	return true
}

// ---------- writeoffs / recognitions ----------
func (s *Services) handleFinWriteoffs(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_receipt_writeoff`)
	case "get":
		id := paramID(c)
		rows, _ := s.DB.Query(`SELECT * FROM fin_receipt_writeoff WHERE id=?`, id)
		if rows == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m := gin.H(list[0])
		lrows, _ := s.DB.Query(`SELECT * FROM fin_receipt_writeoff_line WHERE writeoff_id=?`, id)
		lines := []map[string]interface{}{}
		if lrows != nil {
			defer lrows.Close()
			lines, _ = rowsToMaps(lrows)
		}
		m["lines"] = lines
		api.OK(c, m)
		return true
	case "create":
		body := bindBody(c)
		cid, _ := asInt64(body["customer_id"])
		amt, _ := asFloat(body["amount"])
		if cid == 0 || amt <= 0 {
			api.FailJSON(c, "CUSTOMER_AMOUNT_REQUIRED")
			return true
		}
		docNo := strOrDef(body["doc_no"], finDocNo("RW"))
		res, err := s.DB.Exec(`INSERT INTO fin_receipt_writeoff(doc_no, customer_id, amount, fund_account_id, status) VALUES(?,?,?,?,'draft')`,
			docNo, cid, amt, nullInt64Or(body["fund_account_id"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		oid, _ := asInt64(body["sales_order_id"])
		lineAmt, _ := asFloat(body["line_amount"])
		if lineAmt == 0 {
			lineAmt = amt
		}
		if oid > 0 {
			if lineAmt-amt > 0.01 {
				api.FailJSON(c, "WRITEOFF_LINE_EXCEEDS")
				return true
			}
			_, _ = s.DB.Exec(`INSERT INTO fin_receipt_writeoff_line(writeoff_id, sales_order_id, amount) VALUES(?,?,?)`, id, oid, lineAmt)
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_receipt_writeoff SET amount=COALESCE(?,amount) WHERE id=? AND status='draft'`, nullFloat(body["amount"]), id)
		return s.handleFinWriteoffs(c, "GET", "get")
	case "action:confirm":
		id := paramID(c)
		var cid, fundID sql.NullInt64
		var amt float64
		var status string
		err := s.DB.QueryRow(`SELECT customer_id, amount, fund_account_id, status FROM fin_receipt_writeoff WHERE id=?`, id).
			Scan(&cid, &amt, &fundID, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "confirmed" {
			return s.handleFinWriteoffs(c, "GET", "get")
		}
		var lineSum float64
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM fin_receipt_writeoff_line WHERE writeoff_id=?`, id).Scan(&lineSum)
		if lineSum > 0 && lineSum-amt > 0.01 {
			api.FailJSON(c, "WRITEOFF_OVER_AMOUNT")
			return true
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		if fundID.Valid && fundID.Int64 > 0 {
			_ = s.adjustFundBalance(fundID.Int64, amt)
			_, _ = s.insertLedger(fundID.Int64, 0, "in", amt, fmt.Sprintf("customer:%d", cid.Int64), "receipt_writeoff", id, "收款核单")
		}
		_, _ = s.DB.Exec(`UPDATE fin_receipt_writeoff SET status='confirmed', received_at=? WHERE id=?`, now, id)
		s.writeAuditCtx(c, "fin_receipt_writeoff", id, "confirm", "", gin.H{"status": status}, gin.H{"status": "confirmed", "amount": amt})
		return s.handleFinWriteoffs(c, "GET", "get")
	}
	return true
}

func (s *Services) handleFinRecognitions(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_payment_recognition`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_payment_recognition WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		cid, _ := asInt64(body["customer_id"])
		amt, _ := asFloat(body["amount"])
		if cid == 0 || amt <= 0 {
			api.FailJSON(c, "CUSTOMER_AMOUNT_REQUIRED")
			return true
		}
		docNo := strOrDef(body["doc_no"], finDocNo("PR"))
		res, err := s.DB.Exec(`INSERT INTO fin_payment_recognition(doc_no, customer_id, amount, fund_account_id, status, remark) VALUES(?,?,?,?,'draft',?)`,
			docNo, cid, amt, nullInt64Or(body["fund_account_id"]), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_payment_recognition SET amount=COALESCE(?,amount), remark=COALESCE(NULLIF(?,''),remark) WHERE id=? AND status='draft'`,
			nullFloat(body["amount"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_payment_recognition WHERE id=?`, id)
	case "action:confirm":
		id := paramID(c)
		var cid int64
		var fundID sql.NullInt64
		var amt float64
		var status string
		err := s.DB.QueryRow(`SELECT customer_id, amount, fund_account_id, status FROM fin_payment_recognition WHERE id=?`, id).
			Scan(&cid, &amt, &fundID, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "confirmed" {
			return s.getSimpleRow(c, `SELECT * FROM fin_payment_recognition WHERE id=?`, id)
		}
		fid := fundID.Int64
		if fid == 0 {
			_ = s.DB.QueryRow(`SELECT id FROM fin_fund_account ORDER BY id LIMIT 1`).Scan(&fid)
		}
		if fid > 0 {
			_ = s.adjustFundBalance(fid, amt)
			_, _ = s.insertLedger(fid, 0, "in", amt, fmt.Sprintf("customer:%d", cid), "payment_recognition", id, "销售认款")
		}
		_, _ = s.DB.Exec(`UPDATE fin_payment_recognition SET status='confirmed', fund_account_id=? WHERE id=?`, fid, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_payment_recognition WHERE id=?`, id)
	}
	return true
}

// ---------- fx / allocations / alerts / reconciles ----------
func (s *Services) handleFinFX(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/query") || (action == "list" && strings.Contains(openapiPath, "query")) {
		return s.listDocTable(c, `SELECT * FROM fin_fx_settlement WHERE status='confirmed'`)
	}
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_fx_settlement`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_fx_settlement WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		fx, _ := asFloat(body["amount_fx"])
		rate, _ := asFloat(body["rate"])
		if rate <= 0 {
			rate = 1
		}
		local, ok := asFloat(body["amount_local"])
		if !ok {
			local = fx * rate
		}
		docNo := strOrDef(body["doc_no"], finDocNo("FX"))
		res, err := s.DB.Exec(`INSERT INTO fin_fx_settlement(doc_no, currency, amount_fx, rate, amount_local, fund_account_id, status)
			VALUES(?,?,?,?,?,?,'draft')`, docNo, strOrDef(body["currency"], "USD"), fx, rate, local, nullInt64Or(body["fund_account_id"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "amount_local": local})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_fx_settlement SET rate=COALESCE(?,rate), amount_local=COALESCE(?,amount_local) WHERE id=? AND status='draft'`,
			nullFloat(body["rate"]), nullFloat(body["amount_local"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_fx_settlement WHERE id=?`, id)
	case "action:confirm":
		id := paramID(c)
		var local float64
		var fundID sql.NullInt64
		var status string
		err := s.DB.QueryRow(`SELECT amount_local, fund_account_id, status FROM fin_fx_settlement WHERE id=?`, id).Scan(&local, &fundID, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "confirmed" {
			return s.getSimpleRow(c, `SELECT * FROM fin_fx_settlement WHERE id=?`, id)
		}
		if fundID.Valid && fundID.Int64 > 0 {
			_ = s.adjustFundBalance(fundID.Int64, local)
			_, _ = s.insertLedger(fundID.Int64, 0, "in", local, "", "fx_settlement", id, "外币结汇")
		}
		_, _ = s.DB.Exec(`UPDATE fin_fx_settlement SET status='confirmed' WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_fx_settlement WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleFinAllocations(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_cost_allocation`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_cost_allocation WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		amt, _ := asFloat(body["source_amount"])
		docNo := strOrDef(body["doc_no"], finDocNo("CA"))
		alloc := "{}"
		if v, ok := body["alloc_json"]; ok {
			b, _ := json.Marshal(v)
			alloc = string(b)
		} else if raw, ok := body["alloc_json"].(string); ok {
			alloc = raw
		}
		res, err := s.DB.Exec(`INSERT INTO fin_cost_allocation(doc_no, source_amount, alloc_json, status) VALUES(?,?,?,'posted')`,
			docNo, amt, alloc)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "posted"})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_cost_allocation SET source_amount=COALESCE(?,source_amount) WHERE id=? AND status!='revoked'`,
			nullFloat(body["source_amount"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_cost_allocation WHERE id=?`, id)
	case "action:revoke":
		id := paramID(c)
		var amt float64
		var docNo, status string
		err := s.DB.QueryRow(`SELECT doc_no, source_amount, status FROM fin_cost_allocation WHERE id=?`, id).Scan(&docNo, &amt, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "revoked" {
			api.OK(c, gin.H{"id": id, "status": "revoked"})
			return true
		}
		revNo := finDocNo("CAR")
		res, _ := s.DB.Exec(`INSERT INTO fin_cost_allocation(doc_no, source_amount, alloc_json, status, revoked_from_id)
			VALUES(?,?,'{}','revoked',?)`, revNo, -amt, id)
		rid, _ := res.LastInsertId()
		_, _ = s.DB.Exec(`UPDATE fin_cost_allocation SET status='revoked' WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "revoked", "revoke_id": rid, "revoke_doc_no": revNo})
		return true
	}
	return true
}

func (s *Services) handleFinAlerts(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_receipt_alert`)
	case "create":
		body := bindBody(c)
		cid, _ := asInt64(body["customer_id"])
		if cid == 0 {
			api.FailJSON(c, "CUSTOMER_REQUIRED")
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO fin_receipt_alert(customer_id, order_id, due_date, overdue_days, amount, status)
			VALUES(?,?,?,?,?,'open')`, cid, nullInt64Or(body["order_id"]), strOr(body["due_date"]),
			nullInt64Or(body["overdue_days"]), nullFloat(body["amount"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "status": "open"})
		return true
	case "action:handle":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_receipt_alert SET status='handled', handled_remark=? WHERE id=?`, strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_receipt_alert WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleFinReconciles(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_cashier_reconcile`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_cashier_reconcile WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		fid, _ := asInt64(body["fund_account_id"])
		if fid == 0 {
			api.FailJSON(c, "FUND_ACCOUNT_REQUIRED")
			return true
		}
		var book float64
		_ = s.DB.QueryRow(`SELECT balance FROM fin_fund_account WHERE id=?`, fid).Scan(&book)
		if v, ok := asFloat(body["book_balance"]); ok {
			book = v
		}
		actual, _ := asFloat(body["actual_balance"])
		docNo := strOrDef(body["doc_no"], finDocNo("CR"))
		res, err := s.DB.Exec(`INSERT INTO fin_cashier_reconcile(doc_no, fund_account_id, biz_date, book_balance, actual_balance, status, remark)
			VALUES(?,?,?,?,?,'draft',?)`, docNo, fid, strOrDef(body["biz_date"], time.Now().Format("2006-01-02")),
			book, actual, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "book_balance": book, "diff": actual - book})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_cashier_reconcile SET actual_balance=COALESCE(?,actual_balance), remark=COALESCE(NULLIF(?,''),remark)
			WHERE id=? AND status='draft'`, nullFloat(body["actual_balance"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_cashier_reconcile WHERE id=?`, id)
	case "action:confirm":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE fin_cashier_reconcile SET status='confirmed' WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_cashier_reconcile WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleFinPrepays(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_prepay_prepaid`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_prepay_prepaid WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		pid, _ := asInt64(body["party_id"])
		amt, _ := asFloat(body["amount"])
		if pid == 0 || amt <= 0 {
			api.FailJSON(c, "PARTY_AMOUNT_REQUIRED")
			return true
		}
		docNo := strOrDef(body["doc_no"], finDocNo("PP"))
		bal := amt
		if v, ok := asFloat(body["balance"]); ok {
			bal = v
		}
		res, err := s.DB.Exec(`INSERT INTO fin_prepay_prepaid(doc_no, party_type, party_id, direction, amount, balance, status)
			VALUES(?,?,?,?,?,?,'open')`, docNo, strOrDef(body["party_type"], "customer"), pid,
			strOrDef(body["direction"], "in"), amt, bal)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_prepay_prepaid SET balance=COALESCE(?,balance), status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			nullFloat(body["balance"]), strOr(body["status"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_prepay_prepaid WHERE id=?`, id)
	}
	return true
}

// ---------- cost / contract / returns / arap ----------
func (s *Services) handleFinCostAccountings(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_cost_accounting`)
	case "get":
		id := paramID(c)
		rows, _ := s.DB.Query(`SELECT * FROM fin_cost_accounting WHERE id=?`, id)
		if rows == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m := gin.H(list[0])
		trows, _ := s.DB.Query(`SELECT * FROM fin_cost_trace_line WHERE cost_id=?`, id)
		traces := []map[string]interface{}{}
		if trows != nil {
			defer trows.Close()
			traces, _ = rowsToMaps(trows)
		}
		m["traces"] = traces
		api.OK(c, m)
		return true
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], finDocNo("COST"))
		res, err := s.DB.Exec(`INSERT INTO fin_cost_accounting(doc_no, period, task_id, product_id, material_cost, labor_cost, overhead, total_cost, status)
			VALUES(?,?,?,?,?,?,?,?,?)`,
			docNo, strOrDef(body["period"], time.Now().Format("2006-01")), nullInt64Or(body["task_id"]), nullInt64Or(body["product_id"]),
			nullFloat(body["material_cost"]), nullFloat(body["labor_cost"]), nullFloat(body["overhead"]),
			nullFloat(body["total_cost"]), strOrDef(body["status"], "draft"))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_cost_accounting SET material_cost=COALESCE(?,material_cost), labor_cost=COALESCE(?,labor_cost),
			overhead=COALESCE(?,overhead) WHERE id=? AND status='draft'`,
			nullFloat(body["material_cost"]), nullFloat(body["labor_cost"]), nullFloat(body["overhead"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_cost_accounting WHERE id=?`, id)
	case "action:calc":
		id := paramID(c)
		var mat, labor, oh float64
		var productID, taskID sql.NullInt64
		_ = s.DB.QueryRow(`SELECT COALESCE(material_cost,0), COALESCE(labor_cost,0), COALESCE(overhead,0), product_id, task_id
			FROM fin_cost_accounting WHERE id=?`, id).Scan(&mat, &labor, &oh, &productID, &taskID)
		// pull piecework / report costs if available
		if taskID.Valid {
			var extra float64
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM pd_piecework_summary WHERE task_id=?`, taskID.Int64).Scan(&extra)
			if extra > 0 {
				labor = extra
				_, _ = s.DB.Exec(`INSERT INTO fin_cost_trace_line(cost_id, source_type, source_id, amount) VALUES(?,?,?,?)`,
					id, "piecework", taskID.Int64, extra)
			}
		}
		if productID.Valid {
			var reqCost float64
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM pd_requisition_line WHERE product_id=?`, productID.Int64).Scan(&reqCost)
			if reqCost > 0 && mat == 0 {
				mat = reqCost
				_, _ = s.DB.Exec(`INSERT INTO fin_cost_trace_line(cost_id, source_type, source_id, amount) VALUES(?,?,?,?)`,
					id, "requisition", productID.Int64, reqCost)
			}
		}
		total := mat + labor + oh
		_, _ = s.DB.Exec(`UPDATE fin_cost_accounting SET material_cost=?, labor_cost=?, overhead=?, total_cost=?, status='calculated' WHERE id=?`,
			mat, labor, oh, total, id)
		api.OK(c, gin.H{"id": id, "material_cost": mat, "labor_cost": labor, "overhead": oh, "total_cost": total, "status": "calculated"})
		return true
	}
	return true
}

func (s *Services) handleFinCostTraces(c *gin.Context, action string) bool {
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT t.id, t.cost_id, t.source_type, t.source_id, t.amount, COALESCE(c.doc_no,''), COALESCE(c.period,'')
			FROM fin_cost_trace_line t LEFT JOIN fin_cost_accounting c ON c.id=t.cost_id ORDER BY t.id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, costID, srcID int64
			var srcType, docNo, period string
			var amt float64
			_ = rows.Scan(&id, &costID, &srcType, &srcID, &amt, &docNo, &period)
			list = append(list, gin.H{
				"id": id, "cost_id": costID, "source_type": srcType, "source_id": srcID,
				"amount": amt, "doc_no": docNo, "period": period,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "get":
		costID := paramID(c)
		if v := c.Param("cost_id"); v != "" {
			fmt.Sscanf(v, "%d", &costID)
		}
		rows, err := s.DB.Query(`SELECT * FROM fin_cost_trace_line WHERE cost_id=?`, costID)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		api.OK(c, gin.H{"cost_id": costID, "list": list, "total": len(list)})
		return true
	}
	return true
}

func (s *Services) handleFinContractProfits(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/recalc") || (method == "POST" && action == "create" && strings.Contains(openapiPath, "recalc")) {
		// simple recalc: upsert sample from body or aggregate existing
		body := bindBody(c)
		cid, _ := asInt64(body["contract_id"])
		if cid == 0 {
			cid = 1
		}
		rev, _ := asFloat(body["revenue"])
		cost, _ := asFloat(body["cost"])
		if rev == 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM fin_payment_recognition WHERE status='confirmed'`).Scan(&rev)
		}
		if cost == 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(total_cost),0) FROM fin_cost_accounting WHERE status='calculated'`).Scan(&cost)
		}
		profit := rev - cost
		var exist int64
		_ = s.DB.QueryRow(`SELECT id FROM fin_contract_profit WHERE contract_id=? ORDER BY id DESC LIMIT 1`, cid).Scan(&exist)
		if exist > 0 {
			_, _ = s.DB.Exec(`UPDATE fin_contract_profit SET revenue=?, cost=?, profit=?, period=? WHERE id=?`,
				rev, cost, profit, strOrDef(body["period"], time.Now().Format("2006-01")), exist)
			api.OK(c, gin.H{"id": exist, "contract_id": cid, "revenue": rev, "cost": cost, "profit": profit})
			return true
		}
		res, _ := s.DB.Exec(`INSERT INTO fin_contract_profit(contract_id, revenue, cost, profit, period) VALUES(?,?,?,?,?)`,
			cid, rev, cost, profit, strOrDef(body["period"], time.Now().Format("2006-01")))
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "contract_id": cid, "revenue": rev, "cost": cost, "profit": profit})
		return true
	}
	if action == "list" {
		return s.listDocTable(c, `SELECT * FROM fin_contract_profit`)
	}
	return true
}

func (s *Services) handleFinReturnFinances(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_sales_return_finance`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_sales_return_finance WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		amt, _ := asFloat(body["amount"])
		docNo := strOrDef(body["doc_no"], finDocNo("SRF"))
		res, err := s.DB.Exec(`INSERT INTO fin_sales_return_finance(doc_no, order_id, amount, status) VALUES(?,?,?,'draft')`,
			docNo, nullInt64Or(body["order_id"]), amt)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_sales_return_finance SET amount=COALESCE(?,amount) WHERE id=? AND status='draft'`, nullFloat(body["amount"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_sales_return_finance WHERE id=?`, id)
	case "action:confirm":
		id := paramID(c)
		var amt float64
		var status string
		_ = s.DB.QueryRow(`SELECT amount, status FROM fin_sales_return_finance WHERE id=?`, id).Scan(&amt, &status)
		if status == "confirmed" {
			return s.getSimpleRow(c, `SELECT * FROM fin_sales_return_finance WHERE id=?`, id)
		}
		_, _ = s.insertLedger(0, 0, "out", amt, "", "sales_return_finance", id, "销售退货退单")
		_, _ = s.DB.Exec(`UPDATE fin_sales_return_finance SET status='confirmed' WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_sales_return_finance WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleFinArap(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_arap_adjust`)
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM fin_arap_adjust WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		pid, _ := asInt64(body["party_id"])
		amt, _ := asFloat(body["amount"])
		if pid == 0 || amt == 0 {
			api.FailJSON(c, "PARTY_AMOUNT_REQUIRED")
			return true
		}
		docNo := strOrDef(body["doc_no"], finDocNo("ARAP"))
		res, err := s.DB.Exec(`INSERT INTO fin_arap_adjust(doc_no, party_type, party_id, amount, direction, status, remark)
			VALUES(?,?,?,?,?,'draft',?)`, docNo, strOrDef(body["party_type"], "customer"), pid, amt,
			strOrDef(body["direction"], "increase"), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE fin_arap_adjust SET amount=COALESCE(?,amount), remark=COALESCE(NULLIF(?,''),remark) WHERE id=? AND status='draft'`,
			nullFloat(body["amount"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM fin_arap_adjust WHERE id=?`, id)
	case "action:post":
		id := paramID(c)
		var amt float64
		var dir, status, pt string
		var pid int64
		err := s.DB.QueryRow(`SELECT party_type, party_id, amount, direction, status FROM fin_arap_adjust WHERE id=?`, id).
			Scan(&pt, &pid, &amt, &dir, &status)
		if err == sql.ErrNoRows {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "posted" {
			return s.getSimpleRow(c, `SELECT * FROM fin_arap_adjust WHERE id=?`, id)
		}
		ledgerDir := "in"
		if dir == "decrease" {
			ledgerDir = "out"
		}
		_, _ = s.insertLedger(0, 0, ledgerDir, amt, fmt.Sprintf("%s:%d", pt, pid), "arap_adjust", id, "往来调整")
		_, _ = s.DB.Exec(`UPDATE fin_arap_adjust SET status='posted' WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_arap_adjust WHERE id=?`, id)
	}
	return true
}

// ---------- approvals / statements / month / mini / orders ----------
func (s *Services) handleFinApprovals(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		// aggregate pending from vouchers + approval items
		rows, err := s.DB.Query(`SELECT id, 'voucher' AS biz_type, id AS biz_id, doc_no, COALESCE(summary,'') AS title, status, created_at
			FROM fin_voucher WHERE status IN ('draft','submitted')
			UNION ALL
			SELECT id, biz_type, biz_id, COALESCE(doc_no,''), COALESCE(title,''), status, created_at FROM fin_approval_item WHERE status='pending'
			ORDER BY id DESC`)
		if err != nil {
			return s.listDocTable(c, `SELECT * FROM fin_approval_item`)
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "action:approve":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE fin_approval_item SET status='approved' WHERE id=?`, id)
		_, _ = s.DB.Exec(`UPDATE fin_voucher SET status='approved' WHERE id=? AND status IN ('draft','submitted')`, id)
		api.OK(c, gin.H{"id": id, "status": "approved"})
		return true
	case "action:reject":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE fin_approval_item SET status='rejected' WHERE id=?`, id)
		_, _ = s.DB.Exec(`UPDATE fin_voucher SET status='rejected' WHERE id=? AND status IN ('draft','submitted')`, id)
		api.OK(c, gin.H{"id": id, "status": "rejected"})
		return true
	}
	return true
}

func (s *Services) handleFinStatements(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/generate") || (method == "POST" && strings.Contains(openapiPath, "generate")) {
		body := bindBody(c)
		period := strOrDef(body["period"], time.Now().Format("2006-01"))
		var income, expense, assetDebit, assetCredit, liabCredit, cashDebit, cashCredit float64
		// 已过账凭证 + 科目类型汇总（三表由凭证生成）
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.credit),0) FROM fin_voucher_line l
			JOIN fin_voucher v ON v.id=l.voucher_id JOIN fin_account_subject s ON s.id=l.subject_id
			WHERE v.status='posted' AND v.period=? AND s.subject_type='income'`, period).Scan(&income)
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.debit),0) FROM fin_voucher_line l
			JOIN fin_voucher v ON v.id=l.voucher_id JOIN fin_account_subject s ON s.id=l.subject_id
			WHERE v.status='posted' AND v.period=? AND s.subject_type='expense'`, period).Scan(&expense)
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.debit),0), COALESCE(SUM(l.credit),0) FROM fin_voucher_line l
			JOIN fin_voucher v ON v.id=l.voucher_id JOIN fin_account_subject s ON s.id=l.subject_id
			WHERE v.status='posted' AND v.period=? AND s.subject_type='asset'`, period).Scan(&assetDebit, &assetCredit)
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.credit),0) FROM fin_voucher_line l
			JOIN fin_voucher v ON v.id=l.voucher_id JOIN fin_account_subject s ON s.id=l.subject_id
			WHERE v.status='posted' AND v.period=? AND s.subject_type='liability'`, period).Scan(&liabCredit)
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(l.debit),0), COALESCE(SUM(l.credit),0) FROM fin_voucher_line l
			JOIN fin_voucher v ON v.id=l.voucher_id JOIN fin_account_subject s ON s.id=l.subject_id
			WHERE v.status='posted' AND v.period=? AND (s.code LIKE '1001%' OR s.code LIKE '1002%')`, period).Scan(&cashDebit, &cashCredit)
		// 无凭证时回退流水账
		if income == 0 && expense == 0 {
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='in' AND biz_date LIKE ?`, period+"%").Scan(&income)
			_ = s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM fin_ledger_entry WHERE direction='out' AND biz_date LIKE ?`, period+"%").Scan(&expense)
		}
		var fundBal float64
		_ = s.DB.QueryRow(`SELECT COALESCE(SUM(balance),0) FROM fin_fund_account`).Scan(&fundBal)
		payload := gin.H{
			"period": period,
			"source": "posted_vouchers",
			"profit_loss": gin.H{"income": income, "expense": expense, "profit": income - expense},
			"cash_flow":   gin.H{"in": cashDebit, "out": cashCredit, "net": cashDebit - cashCredit, "fund_balance": fundBal},
			"balance_sheet": gin.H{
				"assets":      assetDebit - assetCredit,
				"liabilities": liabCredit,
				"equity":      (assetDebit - assetCredit) - liabCredit,
				"cash":        fundBal,
			},
		}
		b, _ := json.Marshal(payload)
		for _, code := range []string{"profit", "cashflow", "balance"} {
			_, _ = s.DB.Exec(`INSERT INTO fin_statement_cache(code, period, title, content_json) VALUES(?,?,?,?)`,
				code, period, code+"-"+period, string(b))
		}
		api.OK(c, payload)
		return true
	}
	if strings.Contains(openapiPath, "/export") {
		code := c.Param("code")
		rows, _ := s.DB.Query(`SELECT * FROM fin_statement_cache WHERE code=? ORDER BY id DESC LIMIT 1`, code)
		if rows != nil {
			defer rows.Close()
			list, _ := rowsToMaps(rows)
			if len(list) > 0 {
				api.OK(c, list[0])
				return true
			}
		}
		api.OK(c, gin.H{"code": code, "hint": "请先生成报表"})
		return true
	}
	_ = action
	return s.listDocTable(c, `SELECT * FROM fin_statement_cache`)
}

func (s *Services) handleFinMonthCloses(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM fin_month_close`)
	case "create":
		body := bindBody(c)
		year, _ := asInt64(body["year"])
		month, _ := asInt64(body["month"])
		if year == 0 {
			year = int64(time.Now().Year())
		}
		if month == 0 {
			month = int64(time.Now().Month())
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		res, err := s.DB.Exec(`INSERT INTO fin_month_close(year, month, status, closed_at) VALUES(?,?, 'closed', ?)
			ON CONFLICT(year, month) DO UPDATE SET status='closed', closed_at=excluded.closed_at`, year, month, now)
		if err != nil {
			// sqlite without ON CONFLICT update fallback
			var id int64
			_ = s.DB.QueryRow(`SELECT id FROM fin_month_close WHERE year=? AND month=?`, year, month).Scan(&id)
			if id > 0 {
				_, _ = s.DB.Exec(`UPDATE fin_month_close SET status='closed', closed_at=? WHERE id=?`, now, id)
				api.OK(c, gin.H{"id": id, "year": year, "month": month, "status": "closed"})
				return true
			}
			res, err = s.DB.Exec(`INSERT INTO fin_month_close(year, month, status, closed_at) VALUES(?,?, 'closed', ?)`, year, month, now)
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "year": year, "month": month, "status": "closed"})
		return true
	case "action:reopen":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE fin_month_close SET status='open', closed_at=NULL WHERE id=?`, id)
		return s.getSimpleRow(c, `SELECT * FROM fin_month_close WHERE id=?`, id)
	}
	return true
}

func (s *Services) handleFinMiniprogram(c *gin.Context, method, openapiPath, action string) bool {
	if strings.Contains(openapiPath, "/reconcile") || (method == "POST" && strings.Contains(openapiPath, "reconcile")) {
		body := bindBody(c)
		id, _ := asInt64(body["id"])
		if id == 0 {
			id = paramID(c)
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		if id > 0 {
			_, _ = s.DB.Exec(`UPDATE fin_miniprogram_bill SET status='reconciled', paid_at=? WHERE id=?`, now, id)
			api.OK(c, gin.H{"id": id, "status": "reconciled"})
			return true
		}
		// reconcile all unpaid
		res, _ := s.DB.Exec(`UPDATE fin_miniprogram_bill SET status='reconciled', paid_at=? WHERE status='unpaid'`, now)
		n, _ := res.RowsAffected()
		api.OK(c, gin.H{"reconciled": n})
		return true
	}
	if action == "list" {
		return s.listDocTable(c, `SELECT * FROM fin_miniprogram_bill`)
	}
	if action == "create" {
		body := bindBody(c)
		no := strOrDef(body["bill_no"], finDocNo("MP"))
		amt, _ := asFloat(body["amount"])
		res, err := s.DB.Exec(`INSERT INTO fin_miniprogram_bill(bill_no, channel, amount, status, order_id) VALUES(?,?,?,'unpaid',?)`,
			no, strOrDef(body["channel"], "wechat"), amt, nullInt64Or(body["order_id"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "bill_no": no})
		return true
	}
	return true
}

func (s *Services) handleFinOrders(c *gin.Context, action string) bool {
	// 财务侧销售订单视图：只读真实表 sl_sales_order
	if action == "get" {
		id := paramID(c)
		rows, err := s.DB.Query(`SELECT id, doc_no, status, created_at FROM sl_sales_order WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, list[0])
		return true
	}
	rows, err := s.DB.Query(`SELECT id, doc_no, status, created_at FROM sl_sales_order WHERE COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 200`)
	if err != nil {
		api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return true
	}
	defer rows.Close()
	list, _ := rowsToMaps(rows)
	api.OK(c, gin.H{"list": list, "total": len(list)})
	return true
}

func (s *Services) finPeriodClosed(period string) bool {
	if period == "" || len(period) < 7 {
		return false
	}
	var y, m int
	_, _ = fmt.Sscanf(period[:7], "%d-%d", &y, &m)
	if y == 0 || m == 0 {
		return false
	}
	var st string
	err := s.DB.QueryRow(`SELECT status FROM fin_month_close WHERE year=? AND month=?`, y, m).Scan(&st)
	return err == nil && st == "closed"
}

func (s *Services) finPeriodClosedForVoucher(id int64) bool {
	var period string
	_ = s.DB.QueryRow(`SELECT period FROM fin_voucher WHERE id=?`, id).Scan(&period)
	return s.finPeriodClosed(period)
}
