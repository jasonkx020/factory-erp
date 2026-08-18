package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

type postCashReq struct {
	AccountID    int64
	Direction    string
	Amount       float64
	Counterparty string
	SourceType   string
	SourceID     int64
	Remark       string
	BizDate      string
	Category     string
}

func (s *Services) ensureFinanceCashColumns() {
	for _, stmt := range []string{
		`ALTER TABLE pur_farmer_settlement ADD COLUMN IF NOT EXISTS fund_account_id INTEGER`,
		`ALTER TABLE fin_prepay_prepaid ADD COLUMN IF NOT EXISTS fund_account_id INTEGER`,
		`ALTER TABLE fin_sales_return_finance ADD COLUMN IF NOT EXISTS fund_account_id INTEGER`,
		`ALTER TABLE sl_sales_order ADD COLUMN IF NOT EXISTS received_amount DOUBLE PRECISION NOT NULL DEFAULT 0`,
	} {
		_, _ = s.DB.Exec(stmt)
	}
}

func (s *Services) defaultFundAccountID() int64 {
	var id int64
	_ = s.DB.QueryRow(`SELECT id FROM fin_fund_account WHERE COALESCE(status,'active')='active'
		ORDER BY CASE code WHEN 'CASH' THEN 0 WHEN 'BANK' THEN 1 ELSE 2 END, id LIMIT 1`).Scan(&id)
	return id
}

func (s *Services) resolveFundAccountID(raw int64) (int64, error) {
	id := raw
	if id <= 0 {
		id = s.defaultFundAccountID()
	}
	if id <= 0 {
		return 0, api.Fail("FUND_ACCOUNT_REQUIRED")
	}
	var st string
	err := s.DB.QueryRow(`SELECT COALESCE(status,'active') FROM fin_fund_account WHERE id=?`, id).Scan(&st)
	if err == sql.ErrNoRows {
		return 0, api.Fail("FUND_ACCOUNT_NOT_FOUND")
	}
	if err != nil {
		return 0, err
	}
	if st != "" && st != "active" {
		return 0, api.Fail("FUND_ACCOUNT_INACTIVE")
	}
	return id, nil
}

func (s *Services) requireOpenFinPeriod(bizDate string) error {
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	if s.finPeriodClosed(bizDate) {
		return api.Fail("PERIOD_CLOSED")
	}
	return nil
}

func (s *Services) existingLedgerID(sourceType string, sourceID int64) int64 {
	if sourceType == "" || sourceID <= 0 {
		return 0
	}
	var id int64
	_ = s.DB.QueryRow(`SELECT id FROM fin_ledger_entry WHERE source_doc_type=? AND source_doc_id=? ORDER BY id LIMIT 1`,
		sourceType, sourceID).Scan(&id)
	return id
}

func (s *Services) postCash(req postCashReq) (int64, error) {
	dir := strings.ToLower(strings.TrimSpace(req.Direction))
	if dir != "in" && dir != "out" {
		return 0, api.Fail("INVALID_DIRECTION")
	}
	if req.Amount <= 0.0001 {
		return 0, api.Fail("AMOUNT_REQUIRED")
	}
	accountID, err := s.resolveFundAccountID(req.AccountID)
	if err != nil {
		return 0, err
	}
	bizDate := strings.TrimSpace(req.BizDate)
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	if err := s.requireOpenFinPeriod(bizDate); err != nil {
		return 0, err
	}
	if exist := s.existingLedgerID(req.SourceType, req.SourceID); exist > 0 {
		return exist, nil
	}
	delta := req.Amount
	if dir == "out" {
		delta = -req.Amount
	}
	if err := s.adjustFundBalance(accountID, delta); err != nil {
		return 0, err
	}
	id, err := s.insertLedgerDated(accountID, 0, dir, req.Amount, req.Counterparty, req.SourceType, req.SourceID, req.Remark, bizDate)
	if err != nil {
		_ = s.adjustFundBalance(accountID, -delta)
		return 0, err
	}
	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		if dir == "in" {
			cat = "经营收入"
		} else {
			cat = "经营支出"
		}
	}
	_, _ = s.DB.Exec(`INSERT INTO fin_income_expense_detail(entry_id, category, amount, remark) VALUES(?,?,?,?)`,
		id, cat, req.Amount, req.Remark)
	return id, nil
}

func (s *Services) insertLedgerDated(accountID, subjectID int64, dir string, amount float64, counterparty, srcType string, srcID int64, remark, bizDate string) (int64, error) {
	if bizDate == "" {
		bizDate = time.Now().Format("2006-01-02")
	}
	docNo := finDocNo("LE")
	res, err := s.DB.Exec(`INSERT INTO fin_ledger_entry(doc_no, account_id, subject_id, direction, amount, biz_date, counterparty, source_doc_type, source_doc_id, remark)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, docNo, nullInt(accountID), nullInt(subjectID), dir, amount, bizDate, counterparty, srcType, nullInt(srcID), remark)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func failToJSON(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if be, ok := err.(*api.BusinessError); ok {
		api.FailJSON(c, be.Msg)
		return true
	}
	api.FailJSON(c, err.Error())
	return true
}

func (s *Services) postFarmerSettlementCash(id, fundAccountID int64, transferNo string) error {
	s.ensureFinanceCashColumns()
	var status string
	var amt float64
	var farmerName string
	err := s.DB.QueryRow(`SELECT s.status, s.amount, COALESCE(f.name,'')
		FROM pur_farmer_settlement s LEFT JOIN pur_farmer f ON f.id=s.farmer_id WHERE s.id=?`, id).
		Scan(&status, &amt, &farmerName)
	if err == sql.ErrNoRows {
		return api.Fail("NOT_FOUND")
	}
	if err != nil {
		return err
	}
	if status == "settle_paid" || status == "paid" {
		return api.Fail("ALREADY_PAID")
	}
	cp := farmerName
	if cp == "" {
		cp = fmt.Sprintf("farmer:%d", id)
	}
	remark := "农户货款"
	if transferNo != "" {
		remark = "农户货款 " + transferNo
	}
	_, err = s.postCash(postCashReq{
		AccountID:    fundAccountID,
		Direction:    "out",
		Amount:       amt,
		Counterparty: cp,
		SourceType:   "farmer_settlement",
		SourceID:     id,
		Remark:       remark,
		Category:     "农户货款",
	})
	if err != nil {
		return err
	}
	acc, _ := s.resolveFundAccountID(fundAccountID)
	_, _ = s.DB.Exec(`UPDATE pur_farmer_settlement SET fund_account_id=? WHERE id=?`, acc, id)
	return nil
}

func (s *Services) applyWriteoffToOrders(writeoffID int64) {
	rows, err := s.DB.Query(`SELECT sales_order_id, amount FROM fin_receipt_writeoff_line WHERE writeoff_id=?`, writeoffID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var oid int64
		var amt float64
		if err := rows.Scan(&oid, &amt); err != nil || oid <= 0 {
			continue
		}
		_, _ = s.DB.Exec(`UPDATE sl_sales_order SET received_amount=COALESCE(received_amount,0)+? WHERE id=?`, amt, oid)
	}
}

func (s *Services) customerHasConfirmedWriteoff(customerID int64) bool {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM fin_receipt_writeoff WHERE customer_id=? AND status='confirmed'`, customerID).Scan(&n)
	return n > 0
}

func bindFundAccountID(body map[string]interface{}) int64 {
	n, _ := asInt64(body["fund_account_id"])
	return n
}
