package biz

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

const demoShowcaseVersion = "v4"

// EnsureDemoData fills showcase rows for every admin menu domain so hubs are not empty.
// Idempotent via schema_meta version + DEMO-* unique keys (no fixed IDs, avoids clashing with live data).
func EnsureDemoData(db *sql.DB) {
	if db == nil {
		return
	}
	ensureDemoFarmers(db)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT)`)
	var ver string
	_ = db.QueryRow(`SELECT value FROM schema_meta WHERE key='demo_showcase_version'`).Scan(&ver)
	if ver == demoShowcaseVersion {
		ensureDemoCustomerPortal(db)
		return
	}

	clearDemoShowcase(db)

	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("2006-01-02 15:04:05")
	period := today[:7]

	seedDemoCRM(db, today, now)
	seedDemoSales(db, today, now)
	seedDemoPurchase(db, today, now)
	seedDemoProduction(db, today, now)
	seedDemoInventory(db, today, now)
	seedDemoProduct(db)
	seedDemoAsset(db, today)
	seedDemoFinance(db, today, now, period)
	seedDemoHR(db, today, now, period)
	seedDemoPayroll(db, today, now)
	seedDemoApproval(db)
	seedDemoSystem(db, today, now, period)
	seedDemoReport(db, today, now)

	_, _ = db.Exec(`INSERT OR REPLACE INTO schema_meta(key, value) VALUES('demo_showcase_version', ?)`, demoShowcaseVersion)
	ensureDemoCustomerPortal(db)
	log.Printf("demo showcase data ensured (%s)", demoShowcaseVersion)
}

func clearDemoShowcase(db *sql.DB) {
	// children first where possible; ignore errors on missing tables
	stmts := []string{
		`DELETE FROM sl_sales_order_line WHERE order_id IN (SELECT id FROM sl_sales_order WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_pre_shipment_line WHERE pre_shipment_id IN (SELECT id FROM sl_pre_shipment WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_delivery_line WHERE delivery_id IN (SELECT id FROM sl_delivery_approval WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_sales_bom_line WHERE bom_id IN (SELECT id FROM sl_sales_bom WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_inquiry_line WHERE inquiry_id IN (SELECT id FROM sl_inquiry WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_cost_budget WHERE order_id IN (SELECT id FROM sl_sales_order WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_order_change_log WHERE order_id IN (SELECT id FROM sl_sales_order WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM sl_pre_shipment WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_delivery_approval WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_sales_bom WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_print_log WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_outbound_settle WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_sales_order WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_inquiry WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sl_contract WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_purchase_request_line WHERE request_id IN (SELECT id FROM pur_purchase_request WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pur_purchase_plan_line WHERE plan_id IN (SELECT id FROM pur_purchase_plan WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pur_purchase_inbound_line WHERE inbound_id IN (SELECT id FROM pur_purchase_inbound WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pur_purchase_return_line WHERE return_id IN (SELECT id FROM pur_purchase_return WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pur_incoming_qc WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_purchase_return WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_purchase_inbound WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_purchase_plan WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_purchase_request WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_purchase_task WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_weigh_ticket WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_farmer_settlement WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_inbound_arrival WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pur_trace_lot WHERE trace_code LIKE 'LOT-DEMO-%'`,
		`DELETE FROM pd_piece_issue_line WHERE sheet_id IN (SELECT id FROM pd_piece_issue_sheet WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pd_task_merge_line WHERE merge_id IN (SELECT id FROM pd_task_merge WHERE merge_no LIKE 'DEMO-%')`,
		`DELETE FROM pd_production_task_item WHERE task_id IN (SELECT id FROM pd_production_task WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pd_dispatch WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_report_work WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_work_order WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_production_task WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_qc_order WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_rework_order WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_scrap_record WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_outsource_order WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_consignment_order WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_mrp_result WHERE run_id IN (SELECT id FROM pd_mrp_run WHERE run_no LIKE 'DEMO-%')`,
		`DELETE FROM pd_mrp_run WHERE run_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_task_merge WHERE merge_no LIKE 'DEMO-%'`,
		`DELETE FROM pd_piece_issue_sheet WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_stock_txn_line WHERE txn_id IN (SELECT id FROM inv_stock_txn WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM inv_stocktake_line WHERE stocktake_id IN (SELECT id FROM inv_stocktake WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM inv_transfer_line WHERE transfer_id IN (SELECT id FROM inv_transfer WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM inv_consume_line WHERE consume_id IN (SELECT id FROM inv_consume WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM inv_assemble_split_line WHERE header_id IN (SELECT id FROM inv_assemble_split WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM inv_stock_txn WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_inbound_qc WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_stocktake WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_transfer WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_consume WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_assemble_split WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_price_adjust WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_sales_peel_return WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM inv_material_to_payable WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_voucher_line WHERE voucher_id IN (SELECT id FROM fin_voucher WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM fin_receipt_writeoff_line WHERE writeoff_id IN (SELECT id FROM fin_receipt_writeoff WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM fin_cost_trace_line WHERE cost_id IN (SELECT id FROM fin_cost_accounting WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM fin_voucher WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_ledger_entry WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_invoice WHERE invoice_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_receipt_writeoff WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_payment_recognition WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_prepay_prepaid WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_fx_settlement WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_cost_allocation WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_cashier_reconcile WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_cost_accounting WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_sales_return_finance WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_arap_adjust WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_fund_transfer WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM fin_miniprogram_bill WHERE bill_no LIKE 'DEMO-%'`,
		`DELETE FROM ast_asset_transfer WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM ast_fixed_asset WHERE code LIKE 'FA-DEMO-%'`,
		`DELETE FROM appr_expense_request WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM appr_affair_request WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM appr_queue WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM appr_task WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM hr_leave_request WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM hr_overtime_patch WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM hr_tool_issue_line WHERE issue_id IN (SELECT id FROM hr_tool_issue WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM hr_tool_issue WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM pay_payroll_sheet_line WHERE sheet_id IN (SELECT id FROM pay_payroll_sheet WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pay_payroll_adjust WHERE sheet_id IN (SELECT id FROM pay_payroll_sheet WHERE doc_no LIKE 'DEMO-%')`,
		`DELETE FROM pay_payroll_sheet WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sys_data_repair WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sys_personnel_transfer WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sys_batch_price_job WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM sys_batch_payroll_job WHERE doc_no LIKE 'DEMO-%'`,
		`DELETE FROM crm_customer WHERE code LIKE 'CU-DEMO-%'`,
		`DELETE FROM inv_box_code WHERE code LIKE 'BX-%-DEMO' OR code LIKE 'BX-%DEMO%'`,
		`DELETE FROM prd_product WHERE code IN ('FG-CHIPS','PK-BAG')`,
		`DELETE FROM sys_logistics_track WHERE track_no IN ('SF1234567890','YD99887766')`,
		`DELETE FROM rpt_report_snapshot WHERE report_code IN ('daily','enterprise_overview') AND payload_json LIKE '%demo%'`,
	}
	for _, s := range stmts {
		_, _ = db.Exec(s)
	}
}

func demoID(db *sql.DB, q string, args ...interface{}) int64 {
	var id int64
	_ = db.QueryRow(q, args...).Scan(&id)
	return id
}

func seedDemoCRM(db *sql.DB, today, now string) {
	_, _ = db.Exec(`INSERT INTO crm_customer(id, code, name, short_name, contact_name, mobile, address, level, source, status, owner_user_id, is_public_sea, is_locked, is_hidden, settle_method, payment_days, credit_limit, remark) VALUES
 (11, 'CU-DEMO-11', '桂林零食商贸', '桂林零食', '刘总', '13900001101', '广西桂林', 'A', '展会', 'active', 1, 0, 0, 0, '月结', 30, 80000, '演示客户'),
 (12, 'CU-DEMO-12', '北海水产餐饮', '北海餐饮', '陈采购', '13900001102', '广西北海', 'B', '转介绍', 'active', 1, 0, 1, 0, '现结', 0, 20000, '锁定线索演示'),
 (13, 'CU-DEMO-13', '隐藏测试客户', '隐藏样例', '隐藏', '13900001103', '南宁', 'C', '电话', 'active', NULL, 1, 0, 1, '月结', 15, 5000, '隐藏线索演示')`)

	_, _ = db.Exec(`INSERT INTO crm_opportunity(id, customer_id, title, stage, amount, expected_date, owner_user_id, status, remark) VALUES
 (11, 11, '桂林零食年框供货', 'proposal', 86000, ?, 1, 'open', '演示商机'),
 (12, 1, '南宁批发扩量', 'won', 50000, ?, 1, 'closed', '已赢单演示')`, today, today)

	_, _ = db.Exec(`INSERT INTO crm_follow_up(id, customer_id, opportunity_id, user_id, follow_type, follow_at, content, next_remind_at) VALUES
 (11, 11, 11, 1, 'call', ?, '电话确认配送时效与冷链要求', datetime('now','+2 day')),
 (12, 1, 1, 1, 'wechat', ?, '微信发送报价单与样品照片', datetime('now','+5 day'))`, now, now)

	_, _ = db.Exec(`INSERT INTO crm_lead_assign(id, customer_id, from_user_id, to_user_id, assigned_at, lock_flag, remark) VALUES
 (1, 3, NULL, 1, ?, 0, '公海领取演示'),
 (2, 11, 1, 1, ?, 1, '资源分配演示')`, now, now)

	_, _ = db.Exec(`INSERT INTO crm_lead_release_log(id, customer_id, released_at, reason, to_public_sea, from_user_id, operator_user_id) VALUES
 (1, 13, ?, '超期未跟进自动释放演示', 1, 1, 1)`, now)

	_, _ = db.Exec(`INSERT INTO crm_task_reminder(id, user_id, ref_type, ref_id, remind_at, content, status) VALUES
 (11, 1, 'opportunity', 11, datetime('now','+1 day'), '跟进桂林零食商机报价', 'pending'),
 (12, 1, 'customer', 12, datetime('now','+3 day'), '解锁北海客户后回访', 'pending')`)

	_, _ = db.Exec(`INSERT INTO crm_customer_import_batch(id, file_name, imported_at, success_count, fail_count, fail_detail_json, created_by) VALUES
 (1, 'customers_demo.xlsx', ?, 18, 2, '[{"row":3,"err":"手机号重复"}]', 1)`, now)
}

func seedDemoSales(db *sql.DB, today, now string) {
	_, _ = db.Exec(`INSERT INTO sl_inquiry(doc_no, customer_id, status, owner_user_id, remark, created_at) VALUES
 ('DEMO-INQ-001', 1, 'quoted', 1, '客户询价演示', ?),
 ('DEMO-INQ-002', 11, 'pending', 1, '待询价审批', ?)`, now, now)
	inq1 := demoID(db, `SELECT id FROM sl_inquiry WHERE doc_no='DEMO-INQ-001'`)
	inq2 := demoID(db, `SELECT id FROM sl_inquiry WHERE doc_no='DEMO-INQ-002'`)
	if inq1 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_inquiry_line(inquiry_id, product_id, qty, quote_price, remark)
			SELECT ?, 3, 500, 6.8, '袋装木薯丁' WHERE NOT EXISTS (SELECT 1 FROM sl_inquiry_line WHERE inquiry_id=? AND product_id=3)`, inq1, inq1)
		_, _ = db.Exec(`INSERT INTO sl_inquiry_line(inquiry_id, product_id, qty, quote_price, remark)
			SELECT ?, 2, 200, 3.0, '去芯薯肉' WHERE NOT EXISTS (SELECT 1 FROM sl_inquiry_line WHERE inquiry_id=? AND product_id=2)`, inq1, inq1)
	}
	if inq2 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_inquiry_line(inquiry_id, product_id, qty, quote_price, remark)
			SELECT ?, 3, 1000, 6.5, '批量询价' WHERE NOT EXISTS (SELECT 1 FROM sl_inquiry_line WHERE inquiry_id=? AND product_id=3)`, inq2, inq2)
	}

	_, _ = db.Exec(`INSERT INTO sl_quote_history(customer_id, product_id, price, quoted_at, inquiry_id) VALUES
 (1, 3, 6.8, ?, ?), (11, 3, 6.5, ?, ?)`, today, nullIf0(inq1), today, nullIf0(inq2))

	_, _ = db.Exec(`INSERT INTO sl_contract(doc_no, customer_id, title, amount, status, signed_at, remark) VALUES
 ('DEMO-CT-001', 1, '南宁批发2026框架合同', 200000, 'active', ?, '演示合同'),
 ('DEMO-CT-002', 11, '桂林零食供货合同', 86000, 'draft', ?, '草稿合同')`, today, today)
	ct1 := demoID(db, `SELECT id FROM sl_contract WHERE doc_no='DEMO-CT-001'`)

	_, _ = db.Exec(`INSERT INTO sl_sales_order(doc_no, customer_id, owner_user_id, status, source, contract_id, warehouse_id, total_amount, remark, created_by) VALUES
 ('DEMO-SO-001', 1, 1, 'confirmed', 'manual', ?, 3, 13600, '演示销售订单', 1),
 ('DEMO-SO-002', 11, 1, 'draft', 'self', NULL, 3, 6500, '自助下单演示', 1),
 ('DEMO-SO-003', 2, 1, 'shipped', 'manual', NULL, 3, 9800, '已发货复购样例', 1)`, nullIf0(ct1))
	so1 := demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-001'`)
	so2 := demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-002'`)
	so3 := demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-003'`)
	for _, row := range []struct {
		oid             int64
		pid             int64
		qty, price, amt float64
		delivered       float64
	}{
		{so1, 3, 2000, 6.8, 13600, 500},
		{so2, 3, 1000, 6.5, 6500, 0},
		{so3, 3, 1400, 7.0, 9800, 1400},
	} {
		if row.oid == 0 {
			continue
		}
		_, _ = db.Exec(`INSERT INTO sl_sales_order_line(order_id, product_id, qty, weight, price, amount, delivered_qty)
			SELECT ?, ?, ?, ?, ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM sl_sales_order_line WHERE order_id=? AND product_id=?)`,
			row.oid, row.pid, row.qty, row.qty, row.price, row.amt, row.delivered, row.oid, row.pid)
	}
	ol1 := demoID(db, `SELECT id FROM sl_sales_order_line WHERE order_id=? AND product_id=3`, so1)

	if so1 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_order_change_log(order_id, change_type, before_json, after_json, reason, created_by)
			SELECT ?, 'qty', '{"qty":1800}', '{"qty":2000}', '客户加量', 1
			WHERE NOT EXISTS (SELECT 1 FROM sl_order_change_log WHERE order_id=? AND change_type='qty')`, so1, so1)
	}

	_, _ = db.Exec(`INSERT INTO sl_pre_shipment(doc_no, order_id, plan_ship_date, status, reserved, warehouse_id, remark) VALUES
 ('DEMO-PS-001', ?, ?, 'approved', 1, 3, '预发货演示')`, nullIf0(so1), today)
	ps1 := demoID(db, `SELECT id FROM sl_pre_shipment WHERE doc_no='DEMO-PS-001'`)
	if ps1 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_pre_shipment_line(pre_shipment_id, order_line_id, product_id, qty)
			SELECT ?, ?, 3, 500 WHERE NOT EXISTS (SELECT 1 FROM sl_pre_shipment_line WHERE pre_shipment_id=?)`, ps1, nullIf0(ol1), ps1)
	}

	_, _ = db.Exec(`INSERT INTO sl_delivery_approval(doc_no, order_id, pre_shipment_id, status, warehouse_id, logistics_no, remark) VALUES
 ('DEMO-DA-001', ?, ?, 'approved', 3, 'SF1234567890', '发货审批演示'),
 ('DEMO-DA-002', ?, NULL, 'shipped', 3, 'YD99887766', '已发运')`, nullIf0(so1), nullIf0(ps1), nullIf0(so3))
	da1 := demoID(db, `SELECT id FROM sl_delivery_approval WHERE doc_no='DEMO-DA-001'`)
	da2 := demoID(db, `SELECT id FROM sl_delivery_approval WHERE doc_no='DEMO-DA-002'`)
	if da1 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_delivery_line(delivery_id, product_id, qty, weight) SELECT ?, 3, 500, 500 WHERE NOT EXISTS (SELECT 1 FROM sl_delivery_line WHERE delivery_id=?)`, da1, da1)
	}
	if da2 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_delivery_line(delivery_id, product_id, qty, weight) SELECT ?, 3, 1400, 1400 WHERE NOT EXISTS (SELECT 1 FROM sl_delivery_line WHERE delivery_id=?)`, da2, da2)
	}

	_, _ = db.Exec(`INSERT INTO sl_sales_bom(doc_no, order_id, product_id, name, status) VALUES
 ('DEMO-SBOM-001', ?, 3, '袋装木薯丁销售BOM', 'active')`, nullIf0(so1))
	bom := demoID(db, `SELECT id FROM sl_sales_bom WHERE doc_no='DEMO-SBOM-001'`)
	if bom > 0 {
		_, _ = db.Exec(`INSERT INTO sl_sales_bom_line(bom_id, material_product_id, qty, scrap_rate, remark)
			SELECT ?, 1, 1.2, 0.05, '鲜木薯' WHERE NOT EXISTS (SELECT 1 FROM sl_sales_bom_line WHERE bom_id=? AND material_product_id=1)`, bom, bom)
		_, _ = db.Exec(`INSERT INTO sl_sales_bom_line(bom_id, material_product_id, qty, scrap_rate, remark)
			SELECT ?, 2, 1.05, 0.02, '去芯薯肉' WHERE NOT EXISTS (SELECT 1 FROM sl_sales_bom_line WHERE bom_id=? AND material_product_id=2)`, bom, bom)
	}

	if so1 > 0 {
		_, _ = db.Exec(`INSERT INTO sl_cost_budget(order_id, material_cost, labor_cost, other_cost, total_cost, sale_amount, margin, remark)
			SELECT ?, 8000, 1200, 400, 9600, 13600, 4000, '成本预算演示'
			WHERE NOT EXISTS (SELECT 1 FROM sl_cost_budget WHERE order_id=?)`, so1, so1)
	}

	_, _ = db.Exec(`INSERT INTO sl_quote_calculator_result(customer_id, product_id, qty, base_cost, margin_rate, quote_price, payload_json) VALUES
 (1, 3, 1000, 4.2, 0.25, 5.25, '{"freight":0.3,"pack":0.2}')`)

	_, _ = db.Exec(`INSERT INTO sl_print_log(doc_type, doc_id, doc_no, template_code, printed_by, printed_at) VALUES
 ('sales_order', ?, 'DEMO-SO-001', 'SO_A4', 1, ?)`, nullIf0(so1), now)

	_, _ = db.Exec(`INSERT INTO sl_outbound_settle(doc_no, biz_date, product_id, product_name, plate_no, driver_name, qty, weight, unit_price, goods_amount, amount, status, remark) VALUES
 ('DEMO-OS-001', ?, 3, '袋装木薯丁', '桂A88D88', '张司机', 1400, 1400, 7.0, 9800, 9800, 'posted', '出厂结算演示')`, today)
}

func seedDemoPurchase(db *sql.DB, today, now string) {
	_, _ = db.Exec(`INSERT INTO pur_purchase_request(doc_no, applicant_id, title, qty, status, need_date, remark) VALUES
 ('DEMO-PR-001', 1, '鲜木薯补货申请', 10000, 'approved', ?, '采购申请演示')`, today)
	pr := demoID(db, `SELECT id FROM pur_purchase_request WHERE doc_no='DEMO-PR-001'`)
	if pr > 0 {
		_, _ = db.Exec(`INSERT INTO pur_purchase_request_line(request_id, product_id, qty, suggest_supplier_id)
			SELECT ?, 1, 10000, 1 WHERE NOT EXISTS (SELECT 1 FROM pur_purchase_request_line WHERE request_id=?)`, pr, pr)
	}

	_, _ = db.Exec(`INSERT INTO pur_purchase_plan(doc_no, status, plan_date, remark) VALUES
 ('DEMO-PP-001', 'approved', ?, '月度采购计划演示')`, today)
	pp := demoID(db, `SELECT id FROM pur_purchase_plan WHERE doc_no='DEMO-PP-001'`)
	prl := demoID(db, `SELECT id FROM pur_purchase_request_line WHERE request_id=?`, pr)
	if pp > 0 {
		_, _ = db.Exec(`INSERT INTO pur_purchase_plan_line(plan_id, product_id, qty, supplier_id, request_line_id)
			SELECT ?, 1, 10000, 1, ? WHERE NOT EXISTS (SELECT 1 FROM pur_purchase_plan_line WHERE plan_id=?)`, pp, nullIf0(prl), pp)
	}

	_, _ = db.Exec(`INSERT INTO pur_purchase_inbound(doc_no, supplier_id, warehouse_id, status, biz_date, remark) VALUES
 ('DEMO-PI-001', 1, 1, 'posted', ?, '采购入库演示'),
 ('DEMO-PI-002', 2, 1, 'draft', ?, '包装袋待入')`, today, today)
	pi1 := demoID(db, `SELECT id FROM pur_purchase_inbound WHERE doc_no='DEMO-PI-001'`)
	pi2 := demoID(db, `SELECT id FROM pur_purchase_inbound WHERE doc_no='DEMO-PI-002'`)
	if pi1 > 0 {
		_, _ = db.Exec(`INSERT INTO pur_purchase_inbound_line(inbound_id, product_id, qty, price, amount)
			SELECT ?, 1, 5000, 1.85, 9250 WHERE NOT EXISTS (SELECT 1 FROM pur_purchase_inbound_line WHERE inbound_id=?)`, pi1, pi1)
	}
	if pi2 > 0 {
		_, _ = db.Exec(`INSERT INTO pur_purchase_inbound_line(inbound_id, product_id, qty, price, amount)
			SELECT ?, 2, 200, 0.45, 90 WHERE NOT EXISTS (SELECT 1 FROM pur_purchase_inbound_line WHERE inbound_id=?)`, pi2, pi2)
	}

	_, _ = db.Exec(`INSERT INTO pur_incoming_qc(doc_no, inbound_id, supplier_id, product_id, qty_check, qty_pass, qty_fail, result, status, remark) VALUES
 ('DEMO-IQC-001', ?, 1, 1, 5000, 4850, 150, 'pass', 'done', '来料质检演示')`, nullIf0(pi1))

	_, _ = db.Exec(`INSERT INTO pur_purchase_return(doc_no, supplier_id, inbound_id, warehouse_id, status, remark) VALUES
 ('DEMO-PRT-001', 1, ?, 1, 'draft', '采购退货演示')`, nullIf0(pi1))
	prt := demoID(db, `SELECT id FROM pur_purchase_return WHERE doc_no='DEMO-PRT-001'`)
	if prt > 0 {
		_, _ = db.Exec(`INSERT INTO pur_purchase_return_line(return_id, product_id, qty, amount)
			SELECT ?, 1, 150, 277.5 WHERE NOT EXISTS (SELECT 1 FROM pur_purchase_return_line WHERE return_id=?)`, prt, prt)
	}

	_, _ = db.Exec(`INSERT INTO pur_purchase_task(doc_no, assignee_id, product_id, qty, status, due_date) VALUES
 ('DEMO-PT-001', 1, 1, 10000, 'open', ?)`, today)

	_, _ = db.Exec(`INSERT INTO pur_supplier_price_history(supplier_id, product_id, price, biz_date)
		SELECT 1, 1, 1.85, ? WHERE NOT EXISTS (SELECT 1 FROM pur_supplier_price_history WHERE supplier_id=1 AND product_id=1 AND price=1.85)`, today)
	_, _ = db.Exec(`INSERT INTO pur_supplier_price_history(supplier_id, product_id, price, biz_date)
		SELECT 1, 1, 1.78, date('now','-30 day') WHERE NOT EXISTS (SELECT 1 FROM pur_supplier_price_history WHERE supplier_id=1 AND product_id=1 AND price=1.78)`)

	farmerID := demoID(db, `SELECT id FROM pur_farmer WHERE code='FM01'`)
	if farmerID == 0 {
		farmerID = 1
	}

	// 清理旧流程演示单（draft/待确认出码等），新流程仅保留已生效演示
	_, _ = db.Exec(`DELETE FROM pur_trace_lot WHERE weigh_ticket_id IN (SELECT id FROM pur_weigh_ticket WHERE doc_no IN ('DEMO-WT-002') OR (doc_no LIKE 'DEMO-%' AND status IN ('draft','pending_confirm','qc_pending','qc_pass')))`)
	_, _ = db.Exec(`DELETE FROM pur_farmer_settlement WHERE weigh_ticket_id IN (SELECT id FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-002')`)
	_, _ = db.Exec(`DELETE FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-002' OR (doc_no LIKE 'DEMO-%' AND status IN ('draft','pending_confirm','qc_pending','qc_pass'))`)
	_, _ = db.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, product_id, gross_weight, deduct_weight, net_weight, qc_result, status, biz_date, remark, receive_kind, batch_no, trace_code) VALUES
 ('DEMO-WT-001', ?, 1, 12500, 2500, 10000, 'pass', 'weighed', ?, '过磅收货演示（批号即溯源码已绑定）', 'gate', 'DEMO-B001', 'DEMO-B001')`, farmerID, today)
	wt1 := demoID(db, `SELECT id FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-001'`)

	_, _ = db.Exec(`INSERT INTO pur_farmer_settlement(doc_no, farmer_id, weigh_ticket_id, biz_date, net_weight, unit_price, amount, status, remark) VALUES
 ('DEMO-FS-001', ?, ?, ?, 10000, 1.85, 18500, 'paid', '农户结算演示')`, farmerID, nullIf0(wt1), today)

	_, _ = db.Exec(`INSERT INTO pur_inbound_arrival(doc_no, farmer_id, origin, variety, estimate_weight, status, biz_date, remark) VALUES
 ('DEMO-ARR-001', ?, '广西武鸣', '鲜木薯', 10000, 'confirmed', ?, '到货登记演示')`, farmerID, today)

	_, _ = db.Exec(`DELETE FROM pur_trace_lot WHERE trace_code IN ('LOT-DEMO-001','DEMO-B001')`)
	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, payload_canonical, signature, status) VALUES
 ('DEMO-B001', ?, 'DEMO-B001', ?, 'A', ?, 10000, '{"demo":true,"bind":"batch_as_trace"}', 'demo-sig', 'open')`, today, farmerID, nullIf0(wt1))
}

func seedDemoProduction(db *sql.DB, today, now string) {
	ws := defaultWorkshopDeptIDDB(db)
	_, _ = db.Exec(`INSERT INTO pd_production_task(doc_no, source_type, status, plan_start, plan_end, routing_id, workshop_dept_id, owner_user_id, remark) VALUES
 ('DEMO-TK-001', 'sales_order', 'in_progress', ?, date('now','+3 day'), 1, ?, 1, '生产任务演示'),
 ('DEMO-TK-002', 'manual', 'pending', ?, date('now','+5 day'), 1, ?, 1, '待释放任务')`, today, nullIf0(ws), today, nullIf0(ws))
	tk1 := demoID(db, `SELECT id FROM pd_production_task WHERE doc_no='DEMO-TK-001'`)
	tk2 := demoID(db, `SELECT id FROM pd_production_task WHERE doc_no='DEMO-TK-002'`)
	if tk1 > 0 {
		_, _ = db.Exec(`INSERT INTO pd_production_task_item(task_id, product_id, plan_qty, plan_weight, completed_qty)
			SELECT ?, 3, 2000, 2000, 800 WHERE NOT EXISTS (SELECT 1 FROM pd_production_task_item WHERE task_id=?)`, tk1, tk1)
	}
	if tk2 > 0 {
		_, _ = db.Exec(`INSERT INTO pd_production_task_item(task_id, product_id, plan_qty, plan_weight, completed_qty)
			SELECT ?, 3, 1000, 1000, 0 WHERE NOT EXISTS (SELECT 1 FROM pd_production_task_item WHERE task_id=?)`, tk2, tk2)
	}

	_, _ = db.Exec(`INSERT INTO pd_work_order(doc_no, task_id, process_id, routing_step_id, status, plan_qty) VALUES
 ('DEMO-WO-001', ?, 1, 3, 'in_progress', 2000),
 ('DEMO-WO-002', ?, 4, 6, 'pending', 2000)`, nullIf0(tk1), nullIf0(tk1))
	wo1 := demoID(db, `SELECT id FROM pd_work_order WHERE doc_no='DEMO-WO-001'`)

	_, _ = db.Exec(`INSERT INTO pd_dispatch(doc_no, work_order_id, dispatch_type, worker_id, plan_qty, status, dispatched_at, created_by) VALUES
 ('DEMO-DP-001', ?, 'normal', 2, 800, 'dispatched', ?, 1),
 ('DEMO-DP-002', ?, 'flex', 3, 500, 'reassigned', ?, 1)`, nullIf0(wo1), now, nullIf0(wo1), now)
	dp1 := demoID(db, `SELECT id FROM pd_dispatch WHERE doc_no='DEMO-DP-001'`)
	dp2 := demoID(db, `SELECT id FROM pd_dispatch WHERE doc_no='DEMO-DP-002'`)

	_, _ = db.Exec(`INSERT INTO pd_report_work(doc_no, dispatch_id, work_order_id, process_id, worker_id, report_type, qty, weight, status, reported_at, scan_code) VALUES
 ('DEMO-RW-001', ?, ?, 1, 2, 'output', 800, 800, 'confirmed', ?, 'BX-RAW-DEMO'),
 ('DEMO-RW-002', ?, ?, 1, 3, 'output', 500, 500, 'submitted', ?, 'SCAN-DEMO-02')`,
		nullIf0(dp1), nullIf0(wo1), now, nullIf0(dp2), nullIf0(wo1), now)

	_, _ = db.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, amount)
		SELECT 2, 1, ?, 800, 144 WHERE NOT EXISTS (SELECT 1 FROM pd_piecework_summary WHERE worker_id=2 AND biz_date=? AND process_id=1)`, today, today)
	_, _ = db.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, amount)
		SELECT 3, 4, ?, 500, 125 WHERE NOT EXISTS (SELECT 1 FROM pd_piecework_summary WHERE worker_id=3 AND biz_date=? AND process_id=4)`, today, today)

	_, _ = db.Exec(`INSERT INTO pd_qc_order(doc_no, qc_type, product_id, process_id, qty, result, status) VALUES
 ('DEMO-QC-001', 'process', 3, 6, 800, 'pass', 'done'),
 ('DEMO-QC-002', 'final', 3, 11, 500, 'pending', 'pending')`)
	qc1 := demoID(db, `SELECT id FROM pd_qc_order WHERE doc_no='DEMO-QC-001'`)

	_, _ = db.Exec(`INSERT INTO pd_rework_order(doc_no, source_qc_id, task_id, process_id, qty, status, remark) VALUES
 ('DEMO-RWK-001', ?, ?, 6, 50, 'doing', '袋装破损返修演示')`, nullIf0(qc1), nullIf0(tk1))

	_, _ = db.Exec(`INSERT INTO pd_scrap_record(doc_no, task_id, process_id, product_id, qty, disposition, status) VALUES
 ('DEMO-SCR-001', ?, 1, 1, 120, 'waste', 'posted')`, nullIf0(tk1))

	_, _ = db.Exec(`INSERT INTO pd_drawing_link(drawing_code, drawing_name, task_id, process_id, file_url, status) VALUES
 ('DWG-FG-001', '袋装木薯丁图纸', ?, 11, '/files/demo/fg_diced.pdf', 'active')`, nullIf0(tk1))

	_, _ = db.Exec(`INSERT INTO pd_task_merge(merge_no, title, status, result_task_id, remark) VALUES
 ('DEMO-MG-001', '多单整合演示', 'done', ?, '合并 DEMO-TK-001/002')`, nullIf0(tk1))
	mg := demoID(db, `SELECT id FROM pd_task_merge WHERE merge_no='DEMO-MG-001'`)
	if mg > 0 && tk1 > 0 {
		_, _ = db.Exec(`INSERT INTO pd_task_merge_line(merge_id, source_doc_type, source_doc_id, task_id)
			SELECT ?, 'production_task', ?, ? WHERE NOT EXISTS (SELECT 1 FROM pd_task_merge_line WHERE merge_id=? AND task_id=?)`, mg, tk1, tk1, mg, tk1)
	}
	if mg > 0 && tk2 > 0 {
		_, _ = db.Exec(`INSERT INTO pd_task_merge_line(merge_id, source_doc_type, source_doc_id, task_id)
			SELECT ?, 'production_task', ?, ? WHERE NOT EXISTS (SELECT 1 FROM pd_task_merge_line WHERE merge_id=? AND task_id=?)`, mg, tk2, tk2, mg, tk2)
	}

	_, _ = db.Exec(`INSERT INTO pd_outsource_order(doc_no, supplier_id, process_id, product_id, qty, status, remark) VALUES
 ('DEMO-OSR-001', 2, 6, 3, 300, 'doing', '委外装袋演示')`)

	_, _ = db.Exec(`INSERT INTO pd_consignment_order(doc_no, customer_id, product_id, qty, status, progress, remark) VALUES
 ('DEMO-CSG-001', 1, 3, 500, 'doing', '半成品入库', '受托加工演示')`)

	_, _ = db.Exec(`INSERT INTO pd_mrp_run(run_no, run_at, status, params_json, remark) VALUES
 ('DEMO-MRP-001', ?, 'done', '{"horizon_days":7}', 'MRP演示')`, now)
	mrp := demoID(db, `SELECT id FROM pd_mrp_run WHERE run_no='DEMO-MRP-001'`)
	if mrp > 0 {
		_, _ = db.Exec(`INSERT INTO pd_mrp_result(run_id, product_id, demand_qty, supply_qty, shortage_qty)
			SELECT ?, 1, 12000, 42000, 0 WHERE NOT EXISTS (SELECT 1 FROM pd_mrp_result WHERE run_id=? AND product_id=1)`, mrp, mrp)
		_, _ = db.Exec(`INSERT INTO pd_mrp_result(run_id, product_id, demand_qty, supply_qty, shortage_qty)
			SELECT ?, 3, 3000, 3200, 0 WHERE NOT EXISTS (SELECT 1 FROM pd_mrp_result WHERE run_id=? AND product_id=3)`, mrp, mrp)
	}

	_, _ = db.Exec(`INSERT INTO pd_cost_hide_policy(role_id, name, field_scope, is_enabled) VALUES
 (1, '管理员可见成本', '["cost","margin"]', 1)`)

	_, _ = db.Exec(`INSERT INTO pd_piece_issue_sheet(doc_no, biz_date, status, remark) VALUES
 ('DEMO-PIS-001', ?, 'posted', '计件领料演示')`, today)
	pis := demoID(db, `SELECT id FROM pd_piece_issue_sheet WHERE doc_no='DEMO-PIS-001'`)
	if pis > 0 {
		_, _ = db.Exec(`INSERT INTO pd_piece_issue_line(sheet_id, seq_no, employee_id, employee_name, process_id, process_name, unit_price, qty, amount)
			SELECT ?, 1, 2, '陈某', 1, '去皮', 0.18, 800, 144 WHERE NOT EXISTS (SELECT 1 FROM pd_piece_issue_line WHERE sheet_id=?)`, pis, pis)
	}

	_, _ = db.Exec(`INSERT INTO pd_flow_event(source_type, source_id, from_step_id, to_step_id, trigger_action, status, error, payload_json)
		SELECT 'box', 1, 1, 2, 'scan', 'ok', NULL, '{"qty":800}'
		WHERE NOT EXISTS (SELECT 1 FROM pd_flow_event WHERE trigger_action='scan' AND payload_json LIKE '%800%')`)
	_, _ = db.Exec(`INSERT INTO pd_flow_event(source_type, source_id, from_step_id, to_step_id, trigger_action, status, error, payload_json)
		SELECT 'box', 1, 2, 3, 'handover', 'failed', '演示失败事件', '{}'
		WHERE NOT EXISTS (SELECT 1 FROM pd_flow_event WHERE trigger_action='handover' AND status='failed')`)
}

func seedDemoInventory(db *sql.DB, today, now string) {
	_, _ = db.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark, posted_at, created_by) VALUES
 ('DEMO-ST-IN-001', 'inbound', ?, 'posted', 1, '采购入库演示', ?, 1),
 ('DEMO-ST-OUT-001', 'outbound', ?, 'posted', 3, '销售出库演示', ?, 1),
 ('DEMO-ST-TR-001', 'transfer', ?, 'draft', 1, '调拨草稿', NULL, 1)`, today, now, today, now, today)
	inID := demoID(db, `SELECT id FROM inv_stock_txn WHERE doc_no='DEMO-ST-IN-001'`)
	outID := demoID(db, `SELECT id FROM inv_stock_txn WHERE doc_no='DEMO-ST-OUT-001'`)
	trID := demoID(db, `SELECT id FROM inv_stock_txn WHERE doc_no='DEMO-ST-TR-001'`)
	if inID > 0 {
		_, _ = db.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, weight, batch_no, direction, amount)
			SELECT ?, 1, 1, 5000, 5000, 5000, 'B0801', 'in', 9250 WHERE NOT EXISTS (SELECT 1 FROM inv_stock_txn_line WHERE txn_id=? AND line_no=1)`, inID, inID)
	}
	if outID > 0 {
		_, _ = db.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, weight, batch_no, direction, amount)
			SELECT ?, 1, 3, 500, 500, 500, 'FG-SEED', 'out', 3400 WHERE NOT EXISTS (SELECT 1 FROM inv_stock_txn_line WHERE txn_id=? AND line_no=1)`, outID, outID)
	}
	if trID > 0 {
		_, _ = db.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, weight, batch_no, direction, amount)
			SELECT ?, 1, 1, 200, 200, 200, 'B0801', 'out', 0 WHERE NOT EXISTS (SELECT 1 FROM inv_stock_txn_line WHERE txn_id=? AND line_no=1)`, trID, trID)
	}

	_, _ = db.Exec(`INSERT INTO inv_reservation(warehouse_id, product_id, batch_no, qty, source_doc_type, source_doc_id, status)
		SELECT 3, 3, 'FG-SEED', 500, 'sales_order', id, 'active' FROM sl_sales_order WHERE doc_no='DEMO-SO-001'
		AND NOT EXISTS (SELECT 1 FROM inv_reservation WHERE source_doc_type='sales_order' AND batch_no='FG-SEED' AND qty=500)`)

	_, _ = db.Exec(`INSERT INTO inv_in_transit(product_id, warehouse_id, qty, transit_type, source_doc_type, source_doc_id, status)
		SELECT 1, 1, 2000, 'purchase', 'purchase_plan', COALESCE((SELECT id FROM pur_purchase_plan WHERE doc_no='DEMO-PP-001'),1), 'open'
		WHERE NOT EXISTS (SELECT 1 FROM inv_in_transit WHERE transit_type='purchase' AND qty=2000 AND status='open')`)

	_, _ = db.Exec(`INSERT INTO inv_inbound_qc(doc_no, stock_txn_id, product_id, qty_check, qty_pass, qty_fail, result, status, remark) VALUES
 ('DEMO-IIQC-001', ?, 1, 5000, 4850, 150, 'pass', 'done', '入库质检演示')`, nullIf0(inID))

	ws := defaultWorkshopDeptIDDB(db)
	_, _ = db.Exec(`INSERT INTO inv_stocktake(doc_no, stocktake_type, warehouse_id, workshop_dept_id, biz_date, status, remark) VALUES
 ('DEMO-TKW-001', 'warehouse', 1, NULL, ?, 'posted', '仓库盘点演示'),
 ('DEMO-TKP-001', 'workshop', NULL, ?, ?, 'draft', '车间盘点演示')`, today, nullIf0(ws), today)
	stw := demoID(db, `SELECT id FROM inv_stocktake WHERE doc_no='DEMO-TKW-001'`)
	stp := demoID(db, `SELECT id FROM inv_stocktake WHERE doc_no='DEMO-TKP-001'`)
	if stw > 0 {
		_, _ = db.Exec(`INSERT INTO inv_stocktake_line(stocktake_id, product_id, book_qty, count_qty, diff_qty, batch_no)
			SELECT ?, 1, 42000, 41950, -50, 'B0801' WHERE NOT EXISTS (SELECT 1 FROM inv_stocktake_line WHERE stocktake_id=?)`, stw, stw)
	}
	if stp > 0 {
		_, _ = db.Exec(`INSERT INTO inv_stocktake_line(stocktake_id, product_id, book_qty, count_qty, diff_qty, batch_no)
			SELECT ?, 2, 6500, 6500, 0, 'B0803' WHERE NOT EXISTS (SELECT 1 FROM inv_stocktake_line WHERE stocktake_id=?)`, stp, stp)
	}

	_, _ = db.Exec(`INSERT INTO inv_transfer(doc_no, from_warehouse_id, to_warehouse_id, biz_date, status, remark) VALUES
 ('DEMO-TF-001', 1, 2, ?, 'posted', '原料转半成品库')`, today)
	tf := demoID(db, `SELECT id FROM inv_transfer WHERE doc_no='DEMO-TF-001'`)
	if tf > 0 {
		_, _ = db.Exec(`INSERT INTO inv_transfer_line(transfer_id, product_id, qty, base_qty, batch_no)
			SELECT ?, 1, 500, 500, 'B0801' WHERE NOT EXISTS (SELECT 1 FROM inv_transfer_line WHERE transfer_id=?)`, tf, tf)
	}

	_, _ = db.Exec(`INSERT INTO inv_consume(doc_no, warehouse_id, biz_date, status, remark) VALUES
 ('DEMO-CS-001', 1, ?, 'posted', '生产耗用演示')`, today)
	cs := demoID(db, `SELECT id FROM inv_consume WHERE doc_no='DEMO-CS-001'`)
	if cs > 0 {
		_, _ = db.Exec(`INSERT INTO inv_consume_line(consume_id, product_id, qty)
			SELECT ?, 1, 800 WHERE NOT EXISTS (SELECT 1 FROM inv_consume_line WHERE consume_id=?)`, cs, cs)
	}

	_, _ = db.Exec(`INSERT INTO inv_assemble_split(doc_no, biz_type, warehouse_id, status, remark) VALUES
 ('DEMO-AS-001', 'assemble', 3, 'draft', '组装拆分演示')`)
	as := demoID(db, `SELECT id FROM inv_assemble_split WHERE doc_no='DEMO-AS-001'`)
	if as > 0 {
		_, _ = db.Exec(`INSERT INTO inv_assemble_split_line(header_id, role_type, product_id, qty)
			SELECT ?, 'parent', 2, 100 WHERE NOT EXISTS (SELECT 1 FROM inv_assemble_split_line WHERE header_id=? AND role_type='parent')`, as, as)
		_, _ = db.Exec(`INSERT INTO inv_assemble_split_line(header_id, role_type, product_id, qty)
			SELECT ?, 'child', 3, 95 WHERE NOT EXISTS (SELECT 1 FROM inv_assemble_split_line WHERE header_id=? AND role_type='child')`, as, as)
	}

	_, _ = db.Exec(`INSERT INTO inv_price_adjust(doc_no, product_id, old_price, new_price, effective_at, status, remark) VALUES
 ('DEMO-PA-001', 3, 7.0, 7.2, ?, 'posted', '调价演示')`, today)

	_, _ = db.Exec(`INSERT INTO inv_stock_alert_rule(product_id, warehouse_id, alert_type, min_qty, max_qty, is_enabled)
		SELECT 1, 1, 'shortage', 5000, NULL, 1 WHERE NOT EXISTS (SELECT 1 FROM inv_stock_alert_rule WHERE product_id=1 AND alert_type='shortage')`)
	_, _ = db.Exec(`INSERT INTO inv_stock_alert_rule(product_id, warehouse_id, alert_type, min_qty, max_qty, is_enabled)
		SELECT 3, 3, 'excess', NULL, 20000, 1 WHERE NOT EXISTS (SELECT 1 FROM inv_stock_alert_rule WHERE product_id=3 AND alert_type='excess')`)

	so3 := demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-003'`)
	_, _ = db.Exec(`INSERT INTO inv_sales_peel_return(doc_no, sales_order_id, product_id, peel_qty, weight, warehouse_id, status, remark) VALUES
 ('DEMO-PEEL-001', ?, 3, 20, 20, 3, 'posted', '销售退皮演示')`, nullIf0(so3))

	_, _ = db.Exec(`INSERT INTO inv_material_to_payable(doc_no, supplier_id, product_id, qty, amount, status, remark) VALUES
 ('DEMO-MTP-001', 1, 1, 150, 277.5, 'draft', '物料转应付演示')`)

	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, status) VALUES
 ('BX-SEMI-DEMO', 2, 2, 'B0803', 500, 500, 'open'),
 ('BX-FG-DEMO', 3, 3, 'FG-SEED', 200, 200, 'open')`)
}

func seedDemoProduct(db *sql.DB) {
	_, _ = db.Exec(`INSERT INTO prd_product(id, code, name, product_type, cost_price, sale_price, status, spec_text) VALUES
 (11, 'FG-CHIPS', '木薯脆片', 'finished', 5.5, 9.8, 'active', '200g/袋'),
 (12, 'PK-BAG', '包装袋', 'pack', 0.3, NULL, 'active', '透明袋')`)
	_, _ = db.Exec(`INSERT INTO prd_product_unit(product_id, unit_name, is_base, factor_to_base) VALUES
 (11, '袋', 1, 1), (12, '个', 1, 1)`)
	_, _ = db.Exec(`INSERT INTO prd_product_spec(id, product_id, spec_code, routing_id, remark, status) VALUES
 (1, 3, 'SPEC-DICED-STD', 1, '标准丁规格绑定工艺', 'active'),
 (2, 11, 'SPEC-CHIPS-A', 1, '脆片规格演示', 'active')`)
	_, _ = db.Exec(`INSERT INTO prd_product_app_sort(product_id, channel, sort_no, is_visible) VALUES
 (3, 'app', 1, 1), (11, 'app', 2, 1), (2, 'app', 3, 1), (1, 'app', 4, 1)`)
}

func seedDemoAsset(db *sql.DB, today string) {
	var catID int64 = 1
	_ = db.QueryRow(`SELECT id FROM ast_fixed_asset_category ORDER BY id LIMIT 1`).Scan(&catID)
	_, _ = db.Exec(`INSERT INTO ast_fixed_asset(code, name, category_id, dept_id, dept_name, location_text, original_value, net_value, status, purchase_date, useful_life_months, residual_rate, remark) VALUES
 ('FA-DEMO-001', '去皮机A号', ?, 1, '生产部', '一车间', 128000, 98000, 'active', '2024-03-01', 60, 0.05, '固定资产演示'),
 ('FA-DEMO-002', '冷库压缩机', ?, 1, '仓储部', '成品冷库', 86000, 72000, 'active', '2023-08-15', 84, 0.05, '冷库设备'),
 ('FA-DEMO-003', '叉车01', ?, 2, '仓储部', '原料仓门口', 45000, 30000, 'active', '2022-01-10', 60, 0.05, '运输工具')`, catID, catID, catID)
	a1 := demoID(db, `SELECT id FROM ast_fixed_asset WHERE code='FA-DEMO-001'`)
	a3 := demoID(db, `SELECT id FROM ast_fixed_asset WHERE code='FA-DEMO-003'`)
	_, _ = db.Exec(`INSERT INTO ast_asset_transfer(doc_no, asset_id, from_dept_id, to_dept_id, from_dept_name, to_dept_name, from_location, to_location, status, remark, transferred_at) VALUES
 ('DEMO-FAT-001', ?, 1, 1, '生产部', '维修间', '一车间', '维修间', 'draft', '内部转移待确认', NULL),
 ('DEMO-FAT-002', ?, 2, 1, '仓储部', '生产部', '原料仓门口', '一车间通道', 'confirmed', '已确认转移', ?)`, nullIf0(a1), nullIf0(a3), today)
}

func seedDemoFinance(db *sql.DB, today, now, period string) {
	var sub1, sub2, acc1, acc2 int64 = 1, 2, 1, 2
	_ = db.QueryRow(`SELECT id FROM fin_account_subject ORDER BY id LIMIT 1`).Scan(&sub1)
	_ = db.QueryRow(`SELECT id FROM fin_account_subject ORDER BY id LIMIT 1 OFFSET 1`).Scan(&sub2)
	_ = db.QueryRow(`SELECT id FROM fin_fund_account ORDER BY id LIMIT 1`).Scan(&acc1)
	_ = db.QueryRow(`SELECT id FROM fin_fund_account ORDER BY id LIMIT 1 OFFSET 1`).Scan(&acc2)
	if sub2 == 0 {
		sub2 = sub1
	}
	if acc2 == 0 {
		acc2 = acc1
	}

	_, _ = db.Exec(`INSERT INTO fin_voucher(doc_no, period, biz_date, status, summary) VALUES
 ('DEMO-V-001', ?, ?, 'submitted', '销售收款凭证演示'),
 ('DEMO-V-002', ?, ?, 'draft', '采购付款草稿')`, period, today, period, today)
	v1 := demoID(db, `SELECT id FROM fin_voucher WHERE doc_no='DEMO-V-001'`)
	if v1 > 0 {
		_, _ = db.Exec(`INSERT INTO fin_voucher_line(voucher_id, subject_id, debit, credit, remark)
			SELECT ?, ?, 13600, 0, '银行存款' WHERE NOT EXISTS (SELECT 1 FROM fin_voucher_line WHERE voucher_id=? AND debit>0)`, v1, sub1, v1)
		_, _ = db.Exec(`INSERT INTO fin_voucher_line(voucher_id, subject_id, debit, credit, remark)
			SELECT ?, ?, 0, 13600, '主营业务收入' WHERE NOT EXISTS (SELECT 1 FROM fin_voucher_line WHERE voucher_id=? AND credit>0)`, v1, sub2, v1)
	}

	so1 := demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-001'`)
	_, _ = db.Exec(`INSERT INTO fin_ledger_entry(doc_no, account_id, subject_id, direction, amount, biz_date, counterparty, source_doc_type, source_doc_id, remark) VALUES
 ('DEMO-LE-IN-001', ?, ?, 'in', 13600, ?, '南宁食品批发部', 'sales_order', ?, '收款'),
 ('DEMO-LE-OUT-001', ?, ?, 'out', 9250, ?, '广西木薯原料合作社', 'purchase_inbound', 0, '付款'),
 ('DEMO-LE-IN-002', ?, ?, 'in', 5000, ?, '柳州餐饮连锁', 'recognition', 0, '认款')`,
		acc1, sub1, today, so1, acc1, sub2, today, acc2, sub1, today)

	_, _ = db.Exec(`INSERT INTO fin_invoice(invoice_no, direction, counterparty_id, counterparty_name, amount, tax, status, biz_date) VALUES
 ('DEMO-INV-OUT-001', 'out', 1, '南宁食品批发部', 13600, 1768, 'issued', ?),
 ('DEMO-INV-IN-001', 'in', 1, '广西木薯原料合作社', 9250, 1202.5, 'received', ?)`, today, today)

	_, _ = db.Exec(`INSERT INTO fin_receipt_writeoff(doc_no, customer_id, amount, fund_account_id, status, received_at) VALUES
 ('DEMO-WO-001', 1, 5000, ?, 'posted', ?)`, acc1, today)
	_, _ = db.Exec(`INSERT INTO fin_payment_recognition(doc_no, customer_id, amount, fund_account_id, status, remark) VALUES
 ('DEMO-RG-001', 2, 3000, ?, 'posted', '销售认款演示')`, acc1)
	_, _ = db.Exec(`INSERT INTO fin_prepay_prepaid(doc_no, party_type, party_id, direction, amount, balance, status) VALUES
 ('DEMO-PPY-001', 'customer', 1, 'in', 10000, 5000, 'open'),
 ('DEMO-PPY-002', 'supplier', 1, 'out', 8000, 8000, 'open')`)
	_, _ = db.Exec(`INSERT INTO fin_fx_settlement(doc_no, currency, amount_fx, rate, amount_local, fund_account_id, status) VALUES
 ('DEMO-FX-001', 'USD', 1000, 7.2, 7200, ?, 'posted'),
 ('DEMO-FX-002', 'USD', 500, 7.15, 3575, ?, 'draft')`, acc1, acc1)
	_, _ = db.Exec(`INSERT INTO fin_cost_allocation(doc_no, source_amount, alloc_json, status) VALUES
 ('DEMO-CA-001', 3000, '{"workshop":1,"rate":1}', 'posted')`)
	_, _ = db.Exec(`INSERT INTO fin_cashier_reconcile(doc_no, fund_account_id, biz_date, book_balance, actual_balance, status, remark) VALUES
 ('DEMO-CR-001', ?, ?, 50000, 49850, 'draft', '出纳对账演示')`, acc1, today)
	_, _ = db.Exec(`INSERT INTO fin_cost_accounting(doc_no, period, task_id, product_id, material_cost, labor_cost, overhead, total_cost, status) VALUES
 ('DEMO-COST-001', ?, ?, 3, 8000, 1200, 800, 10000, 'posted')`, period, demoID(db, `SELECT id FROM pd_production_task WHERE doc_no='DEMO-TK-001'`))
	_, _ = db.Exec(`INSERT INTO fin_sales_return_finance(doc_no, order_id, amount, status) VALUES
 ('DEMO-SRF-001', ?, 140, 'draft')`, demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-003'`))
	_, _ = db.Exec(`INSERT INTO fin_arap_adjust(doc_no, party_type, party_id, amount, direction, status, remark) VALUES
 ('DEMO-ARAP-001', 'customer', 1, 200, 'increase', 'posted', '往来调整演示'),
 ('DEMO-ARAP-002', 'supplier', 1, 150, 'decrease', 'draft', '应付调减')`)
	_, _ = db.Exec(`INSERT INTO fin_month_close(year, month, status) VALUES (?, ?, 'open')`, time.Now().Year(), int(time.Now().Month()))
	_, _ = db.Exec(`INSERT INTO fin_miniprogram_bill(bill_no, channel, amount, status, order_id) VALUES
 ('DEMO-MP-001', 'wechat', 6500, 'paid', ?)`, demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-002'`))
	_, _ = db.Exec(`INSERT INTO fin_fund_transfer(doc_no, from_account_id, to_account_id, amount, status, remark) VALUES
 ('DEMO-FT-001', ?, ?, 10000, 'posted', '资金调拨演示')`, acc1, acc2)
	_, _ = db.Exec(`INSERT INTO fin_statement_cache(code, period, title, content_json) VALUES
 ('income', ?, '利润表演示缓存', '{"income":18600,"cost":10000,"profit":8600}')`, period)
	if v1 > 0 {
		_, _ = db.Exec(`INSERT INTO fin_approval_item(biz_type, biz_id, doc_no, title, amount, status)
			SELECT 'voucher', ?, 'DEMO-V-001', '销售收款凭证审批', 13600, 'pending'
			WHERE NOT EXISTS (SELECT 1 FROM fin_approval_item WHERE doc_no='DEMO-V-001')`, v1)
	}
	_, _ = db.Exec(`INSERT INTO fin_receipt_alert(customer_id, order_id, due_date, overdue_days, amount, status)
		SELECT 1, ?, date('now','-5 day'), 5, 8600, 'open'
		WHERE NOT EXISTS (SELECT 1 FROM fin_receipt_alert WHERE amount=8600 AND overdue_days=5)`, nullIf0(so1))
}

func seedDemoHR(db *sql.DB, today, now, period string) {
	_, _ = db.Exec(`INSERT INTO hr_attendance_record(employee_id, biz_date, check_in_at, check_out_at, source)
		SELECT 2, ?, ? || ' 08:02:00', ? || ' 18:05:00', 'demo'
		WHERE NOT EXISTS (SELECT 1 FROM hr_attendance_record WHERE employee_id=2 AND biz_date=? AND source='demo')`, today, today, today, today)
	_, _ = db.Exec(`INSERT INTO hr_attendance_record(employee_id, biz_date, check_in_at, check_out_at, source)
		SELECT 3, ?, ? || ' 08:20:00', ? || ' 18:00:00', 'demo'
		WHERE NOT EXISTS (SELECT 1 FROM hr_attendance_record WHERE employee_id=3 AND biz_date=? AND source='demo')`, today, today, today, today)

	_, _ = db.Exec(`INSERT INTO hr_leave_request(doc_no, employee_id, leave_type, start_at, end_at, status, remark) VALUES
 ('DEMO-LV-001', 2, 'annual', ? || ' 09:00:00', ? || ' 18:00:00', 'pending', '考勤/请假审批演示'),
 ('DEMO-LV-002', 3, 'sick', date('now','-2 day') || ' 09:00:00', date('now','-2 day') || ' 18:00:00', 'approved', '已批病假')`, today, today)

	_, _ = db.Exec(`INSERT INTO hr_overtime_patch(doc_no, employee_id, biz_type, biz_date, minutes, status, remark) VALUES
 ('DEMO-OT-001', 2, 'overtime', ?, 90, 'pending', '加班补卡演示'),
 ('DEMO-OT-002', 3, 'patch', ?, 30, 'approved', '漏打卡补卡')`, today, today)

	y, m := time.Now().Year(), int(time.Now().Month())
	_, _ = db.Exec(`INSERT INTO hr_attendance_month_stat(employee_id, year, month, work_days, late_times, ot_hours, leave_days) VALUES
 (2, ?, ?, 21, 1, 6, 1),
 (3, ?, ?, 20, 3, 2, 0)`, y, m, y, m)

	_, _ = db.Exec(`INSERT INTO hr_tool_issue(doc_no, biz_date, employee_id, employee_name, status, remark) VALUES
 ('DEMO-TI-001', ?, 2, '陈某', 'open', '工具领用演示'),
 ('DEMO-TI-002', ?, 3, '固定工甲', 'returned', '已归还')`, today, today)
	var demoTI1, demoTI2 int64
	_ = db.QueryRow(`SELECT id FROM hr_tool_issue WHERE doc_no='DEMO-TI-001'`).Scan(&demoTI1)
	_ = db.QueryRow(`SELECT id FROM hr_tool_issue WHERE doc_no='DEMO-TI-002'`).Scan(&demoTI2)
	if demoTI1 > 0 {
		_, _ = db.Exec(`INSERT INTO hr_tool_issue_line(issue_id, tool_item_id, tool_name, issue_qty, return_qty) VALUES
 (?,1,'刮刀',1,0), (?,3,'厚手套',2,0)`, demoTI1, demoTI1)
	}
	if demoTI2 > 0 {
		_, _ = db.Exec(`INSERT INTO hr_tool_issue_line(issue_id, tool_item_id, tool_name, issue_qty, return_qty) VALUES
 (?,2,'小刀',1,1)`, demoTI2)
	}

	eSales := demoID(db, `SELECT id FROM hr_employee WHERE emp_no='E-SL'`)
	_, _ = db.Exec(`INSERT INTO hr_visit_record(employee_id, customer_id, visit_at, content, location)
		SELECT ?, 1, ?, '商务拜访确认下周提货', '南宁'
		WHERE ? > 0 AND NOT EXISTS (SELECT 1 FROM hr_visit_record WHERE content LIKE '%下周提货%')`, nullIf0(eSales), now, eSales)

	_, _ = db.Exec(`INSERT INTO hr_memo(owner_user_id, title, content, biz_date, scope_type)
		SELECT 1, '车间安全检查', '周五下午安全巡检', ?, 'hr'
		WHERE NOT EXISTS (SELECT 1 FROM hr_memo WHERE title='车间安全检查')`, today)

	_, _ = db.Exec(`INSERT INTO hr_employee_journal(employee_id, biz_date, content)
		SELECT 2, ?, '完成去皮工序800kg，设备运行正常'
		WHERE NOT EXISTS (SELECT 1 FROM hr_employee_journal WHERE employee_id=2 AND biz_date=? AND content LIKE '%去皮工序%')`, today, today)

	eWH := demoID(db, `SELECT id FROM hr_employee WHERE emp_no='E-WH'`)
	_, _ = db.Exec(`INSERT INTO hr_onboard(employee_id, status, remark, onboard_date, need_account, login_name)
		SELECT ?, 'done', '入职登记演示', ?, 1, 'cangguan'
		WHERE ? > 0 AND NOT EXISTS (SELECT 1 FROM hr_onboard WHERE login_name='cangguan')`, nullIf0(eWH), today, eWH)
	_, _ = db.Exec(`INSERT INTO hr_offboard(employee_id, status, reason, offboard_date)
		SELECT ?, 'draft', '个人原因离职演示', ?
		WHERE ? > 0 AND NOT EXISTS (SELECT 1 FROM hr_offboard WHERE employee_id=? AND reason LIKE '%离职演示%')`, nullIf0(eSales), today, eSales, eSales)

	_, _ = db.Exec(`INSERT INTO hr_performance_scheme(name, scheme_json, status)
		SELECT '默认绩效方案', '{"punctuality":30,"output":50,"quality":20}', 'active'
		WHERE NOT EXISTS (SELECT 1 FROM hr_performance_scheme WHERE name='默认绩效方案')`)
	_, _ = db.Exec(`INSERT INTO hr_attendance_perf_summary(employee_id, period, attendance_score, perf_score, summary_json) VALUES
 (2, ?, 92, 88, '{"demo":true}')`, period)
}

func seedDemoPayroll(db *sql.DB, today, now string) {
	eSales := demoID(db, `SELECT id FROM hr_employee WHERE emp_no='E-SL'`)
	_, _ = db.Exec(`INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, status)
		SELECT 2, 'piece', 0, '622202******1234', 'active'
		WHERE NOT EXISTS (SELECT 1 FROM pay_worker_profile WHERE employee_id=2)`)
	_, _ = db.Exec(`INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, status)
		SELECT 3, 'fixed', 4500, '622202******5678', 'active'
		WHERE NOT EXISTS (SELECT 1 FROM pay_worker_profile WHERE employee_id=3)`)
	if eSales > 0 {
		_, _ = db.Exec(`INSERT INTO pay_worker_profile(employee_id, pay_type, monthly_base, bank_account, status)
			SELECT ?, 'commission', 3000, '622202******9012', 'active'
			WHERE NOT EXISTS (SELECT 1 FROM pay_worker_profile WHERE employee_id=?)`, eSales, eSales)
	}

	y, m := time.Now().AddDate(0, -1, 0).Year(), int(time.Now().AddDate(0, -1, 0).Month())
	_, _ = db.Exec(`INSERT INTO pay_payroll_sheet(doc_no, period_year, period_month, status, workshop_dept_id, calc_at, remark, created_by) VALUES
 ('DEMO-PAY-001', ?, ?, 'confirmed', ?, ?, '薪酬核算演示', 1)`, y, m, nullIf0(defaultWorkshopDeptIDDB(db)), now)
	sheet := demoID(db, `SELECT id FROM pay_payroll_sheet WHERE doc_no='DEMO-PAY-001'`)
	if sheet > 0 {
		_, _ = db.Exec(`INSERT INTO pay_payroll_sheet_line(sheet_id, employee_id, emp_type, piece_amount, attendance_amount, commission_amount, adjust_amount, total_amount)
			SELECT ?, 2, 'piece', 1440, 200, 0, 50, 1690
			WHERE NOT EXISTS (SELECT 1 FROM pay_payroll_sheet_line WHERE sheet_id=? AND employee_id=2)`, sheet, sheet)
		_, _ = db.Exec(`INSERT INTO pay_payroll_sheet_line(sheet_id, employee_id, emp_type, piece_amount, attendance_amount, commission_amount, adjust_amount, total_amount)
			SELECT ?, 3, 'fixed', 0, 4500, 0, -100, 4400
			WHERE NOT EXISTS (SELECT 1 FROM pay_payroll_sheet_line WHERE sheet_id=? AND employee_id=3)`, sheet, sheet)
		_, _ = db.Exec(`INSERT INTO pay_payroll_adjust(sheet_id, employee_id, adjust_type, amount, reason)
			SELECT ?, 2, 'bonus', 50, '质量奖励'
			WHERE NOT EXISTS (SELECT 1 FROM pay_payroll_adjust WHERE sheet_id=? AND employee_id=2 AND adjust_type='bonus')`, sheet, sheet)
		_, _ = db.Exec(`INSERT INTO pay_payroll_adjust(sheet_id, employee_id, adjust_type, amount, reason)
			SELECT ?, 3, 'deduct', -100, '迟到扣款'
			WHERE NOT EXISTS (SELECT 1 FROM pay_payroll_adjust WHERE sheet_id=? AND employee_id=3 AND adjust_type='deduct')`, sheet, sheet)
	}

	var ruleID int64 = 1
	_ = db.QueryRow(`SELECT id FROM pay_sales_commission_rule ORDER BY id LIMIT 1`).Scan(&ruleID)
	if eSales > 0 {
		_, _ = db.Exec(`INSERT INTO pay_commission_calc(rule_id, employee_id, period, base_amount, commission_amount, source_doc_refs)
			SELECT ?, ?, ?, 9800, 294, 'DEMO-SO-003'
			WHERE NOT EXISTS (SELECT 1 FROM pay_commission_calc WHERE source_doc_refs='DEMO-SO-003')`, ruleID, eSales, today[:7])
	}
}

func seedDemoApproval(db *sql.DB) {
	_, _ = db.Exec(`INSERT INTO appr_expense_request(doc_no, applicant_id, amount, category, status, remark) VALUES
 ('DEMO-ER-001', 1, 860, '差旅费', 'submitted', '费用申请已提交'),
 ('DEMO-ER-002', 1, 320, '办公费用', 'draft', '费用申请草稿')`)
	_, _ = db.Exec(`INSERT INTO appr_affair_request(doc_no, applicant_id, title, content, status, remark) VALUES
 ('DEMO-AF-001', 1, '申请更换车间照明', '一车间灯管老化需更换', 'submitted', '事务申请演示')`)
	er1 := demoID(db, `SELECT id FROM appr_expense_request WHERE doc_no='DEMO-ER-001'`)
	so1 := demoID(db, `SELECT id FROM sl_sales_order WHERE doc_no='DEMO-SO-001'`)
	_, _ = db.Exec(`INSERT INTO appr_queue(category, doc_no, title, biz_type, biz_id, amount, applicant_id, status, remark) VALUES
 ('expense_finance', 'DEMO-ER-001', '差旅费-DEMO-ER-001', 'expense', ?, 860, 1, 'pending', '由费用申请转入'),
 ('doc_review', 'DEMO-SO-001', '销售订单审核 DEMO-SO-001', 'sales_order', ?, 13600, 1, 'pending', '单据审核演示')`, nullIf0(er1), nullIf0(so1))
	_, _ = db.Exec(`INSERT INTO appr_task(doc_type, doc_id, assignee_user_id, status, title, doc_no, amount, applicant_id, remark) VALUES
 ('sales_order', ?, 1, 'pending', '销售订单 DEMO-SO-001', 'DEMO-SO-001', 13600, 1, '任务管理演示')`, nullIf0(so1))
}

func seedDemoSystem(db *sql.DB, today, now, period string) {
	_, _ = db.Exec(`INSERT INTO sys_print_template(id, code, name, doc_type, content, status) VALUES
 (1, 'SO_A4', '销售订单A4', 'sales_order', '<h1>销售订单</h1>', 'active'),
 (2, 'PO_A4', '采购入库单', 'purchase_inbound', '<h1>采购入库</h1>', 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_formula(id, code, name, scope, expression, remark, status) VALUES
 (1, 'MARGIN', '毛利率', 'sales', '(sale-cost)/sale', '报价计算', 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_carrier(id, code, name, contact, phone, status) VALUES
 (1, 'SF', '顺丰速运', '客服', '95338', 'active'),
 (2, 'YD', '韵达快递', '客服', '95546', 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_approval_flow(id, code, name, doc_type, status) VALUES
 (1, 'FLOW-SO', '销售订单审批流', 'sales_order', 'active'),
 (2, 'FLOW-PO', '采购入库审批流', 'purchase_inbound', 'active')`)
	_, _ = db.Exec(`INSERT INTO sys_approval_flow_node(id, flow_id, seq_no, node_name, approver_role, require_all) VALUES
 (1, 1, 1, '销售主管', 'sales_manager', 0),
 (2, 1, 2, '财务审核', 'finance', 0),
 (3, 2, 1, '采购主管', 'purchase_manager', 0)`)

	_, _ = db.Exec(`INSERT INTO sys_personnel_transfer(id, doc_no, employee_id, from_dept_id, to_dept_id, reason, status, effective_date) VALUES
 (1, 'DEMO-PTF-001', 3, 1, 1, '岗位轮换演示', 'draft', ?)`, today)

	_, _ = db.Exec(`INSERT INTO sys_batch_price_job(id, doc_no, target_type, adjust_type, adjust_value, status, result_msg) VALUES
 (1, 'DEMO-BP-001', 'product', 'percent', 0.05, 'done', '已调价3个SKU')`)

	_, _ = db.Exec(`INSERT INTO sys_batch_payroll_job(id, doc_no, period_ym, workshop_dept_id, status, result_msg) VALUES
 (1, 'DEMO-BPAY-001', ?, ?, 'done', '已核算21人')`, period, nullIf0(defaultWorkshopDeptIDDB(db)))

	_, _ = db.Exec(`INSERT INTO sys_reminder(id, title, content, remind_at, target_user_id, status) VALUES
 (1, '月结提醒', '请完成本月财务月结', datetime('now','+2 day'), 1, 'open')`)

	_, _ = db.Exec(`INSERT INTO sys_announcement(id, title, content, status, published_at, created_by) VALUES
 (1, '安全生产月通知', '全厂开展安全检查，请各部门配合。', 'published', ?, 1)`, now)

	_, _ = db.Exec(`INSERT INTO sys_memo(id, title, content, owner_id, status) VALUES
 (1, '系统演示备忘', '客户验收走查清单', 1, 'open')`)

	_, _ = db.Exec(`INSERT INTO sys_document(id, code, title, category, content, status) VALUES
 (1, 'DOC-DEMO-01', '木薯加工工艺说明', '工艺', '演示文档内容', 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_drawing(id, code, title, product_id, version_no, file_url, status) VALUES
 (1, 'DWG-DEMO-01', '袋装木薯丁包装图', 3, 'V1', '/files/demo/pack.png', 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_knowledge(id, code, title, category, content, status) VALUES
 (1, 'KB-DEMO-01', '去皮工序作业指导', '生产', '演示知识库条目', 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_course(id, code, title, category, content, duration_min, status) VALUES
 (1, 'CRS-DEMO-01', '新员工入职安全培训', '学堂', '演示课程', 45, 'active')`)

	_, _ = db.Exec(`INSERT INTO sys_data_repair(id, doc_no, target_type, target_id, action, reason, status, applied_at, created_by) VALUES
 (1, 'DEMO-DR-001', 'inv_balance', 1, 'rebuild', '库存余额重建演示', 'done', ?, 1)`, now)
}

func seedDemoReport(db *sql.DB, today, now string) {
	var carrierID int64 = 1
	_ = db.QueryRow(`SELECT id FROM sys_logistics_carrier ORDER BY id LIMIT 1`).Scan(&carrierID)
	_, _ = db.Exec(`INSERT INTO sys_logistics_track(id, track_no, carrier_id, order_id, status, location, updated_at) VALUES
 (1, 'SF1234567890', ?, 1, 'in_transit', '南宁转运中心', ?),
 (2, 'YD99887766', ?, 3, 'delivered', '柳州已签收', ?)`, carrierID, now, carrierID, now)

	payload := fmt.Sprintf(`{"biz_date":%q,"sales_amount":13600,"note":"demo snapshot"}`, today)
	_, _ = db.Exec(`INSERT INTO rpt_report_snapshot(report_code, biz_date, payload_json) VALUES
 ('daily', ?, ?)`, today, payload)
	_, _ = db.Exec(`INSERT INTO rpt_report_snapshot(report_code, biz_date, payload_json) VALUES
 ('enterprise_overview', ?, ?)`, today, fmt.Sprintf(`{"sales_orders":3,"generated_at":%q}`, now))
}

func ensureDemoFarmers(db *sql.DB) {
	if db == nil {
		return
	}
	_, err := db.Exec(`INSERT INTO pur_farmer(code, name, mobile, origin, trace_code, trace_code_prefix, status, remark, default_unit_price)
VALUES
 ('FM01', '黄桂生', '13807710001', '南宁武鸣', 'FM01', 'FM01', 'active', '开发种子·鲜薯入厂', 1.20),
 ('FM02', '李秀兰', '13807710002', '南宁横州', 'FM02', 'FM02', 'active', '开发种子·鲜薯入厂', 1.18),
 ('FM03', '韦建国', '13907710003', '南宁宾阳', 'FM03', 'FM03', 'active', '开发种子·鲜薯入厂', 1.22),
 ('FM04', '覃金莲', '13707710004', '钦州灵山', 'FM04', 'FM04', 'active', '开发种子·鲜薯入厂', 1.15),
 ('FM05', '陈木生', '13607710005', '北海合浦', 'FM05', 'FM05', 'active', '开发种子·鲜薯入厂', 1.25),
 ('FM06', '农福田', '13507710006', '崇左扶绥', 'FM06', 'FM06', 'active', '开发种子·鲜薯入厂', 1.16),
 ('FM07', '陆阿婆', '13407710007', '贵港桂平', 'FM07', 'FM07', 'active', '开发种子·鲜薯入厂', 1.10),
 ('FM08', '门口过磅点', '13307710008', '厂区地磅', 'FM08', 'FM08', 'active', '开发种子·现场临时户', 1.20)
ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		_, _ = db.Exec(`INSERT INTO pur_farmer(code, name, mobile, origin, trace_code, trace_code_prefix, status, remark)
VALUES
 ('FM01', '黄桂生', '13807710001', '南宁武鸣', 'FM01', 'FM01', 'active', '开发种子·鲜薯入厂'),
 ('FM02', '李秀兰', '13807710002', '南宁横州', 'FM02', 'FM02', 'active', '开发种子·鲜薯入厂'),
 ('FM03', '韦建国', '13907710003', '南宁宾阳', 'FM03', 'FM03', 'active', '开发种子·鲜薯入厂'),
 ('FM04', '覃金莲', '13707710004', '钦州灵山', 'FM04', 'FM04', 'active', '开发种子·鲜薯入厂'),
 ('FM05', '陈木生', '13607710005', '北海合浦', 'FM05', 'FM05', 'active', '开发种子·鲜薯入厂'),
 ('FM06', '农福田', '13507710006', '崇左扶绥', 'FM06', 'FM06', 'active', '开发种子·鲜薯入厂'),
 ('FM07', '陆阿婆', '13407710007', '贵港桂平', 'FM07', 'FM07', 'active', '开发种子·鲜薯入厂'),
 ('FM08', '门口过磅点', '13307710008', '厂区地磅', 'FM08', 'FM08', 'active', '开发种子·现场临时户')
ON CONFLICT (code) DO NOTHING`)
	}
}
