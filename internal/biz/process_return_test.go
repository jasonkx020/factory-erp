package biz

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"erp/internal/config"
	"erp/internal/middleware"
	"erp/internal/persistence"
	"erp/internal/security"
)

func openSmokeDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("ERP_TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = os.Getenv("ERP_DATABASE_DSN")
	}
	if dsn == "" {
		t.Skip("set ERP_TEST_DATABASE_DSN for PostgreSQL smoke tests")
	}
	cfg := &config.Config{}
	cfg.Database.Driver = "postgres"
	cfg.Database.DSN = dsn
	cfg.Database.InitSchema = false
	pdb, err := persistence.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	db := pdb.SQL
	t.Cleanup(func() { _ = pdb.Close() })
	stmts := []string{
		`CREATE TEMP TABLE IF NOT EXISTS inv_box_code(
			id BIGSERIAL PRIMARY KEY, code TEXT UNIQUE, product_id INTEGER, warehouse_id INTEGER,
			qty DOUBLE PRECISION, weight DOUBLE PRECISION, current_process_id INTEGER, current_step_id INTEGER,
			task_id INTEGER, work_order_id INTEGER, farmer_id INTEGER, trace_code TEXT,
			origin TEXT, receive_date TEXT, source_type TEXT, status TEXT, parent_box_id INTEGER,
			batch_no TEXT, updated_at TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS inv_stock_txn(
			id BIGSERIAL PRIMARY KEY, doc_no TEXT, doc_type TEXT, biz_date TEXT,
			status TEXT, warehouse_id INTEGER, remark TEXT, posted_at TEXT, is_deleted INTEGER DEFAULT 0)`,
		`CREATE TEMP TABLE IF NOT EXISTS inv_stock_txn_line(
			id BIGSERIAL PRIMARY KEY, txn_id INTEGER, line_no INTEGER, product_id INTEGER,
			qty DOUBLE PRECISION, base_qty DOUBLE PRECISION, direction TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS inv_balance(
			id BIGSERIAL PRIMARY KEY, warehouse_id INTEGER, location_id INTEGER,
			product_id INTEGER, batch_no TEXT, box_code_id INTEGER, qty DOUBLE PRECISION)`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_routing_step(
			id BIGSERIAL PRIMARY KEY, routing_id INTEGER, seq_no INTEGER, process_id INTEGER,
			step_code TEXT, step_name TEXT, is_piecework INTEGER, is_inbound_checkpoint INTEGER,
			checkpoint_bind_warehouse INTEGER, auto_next INTEGER, auto_stock_in INTEGER,
			auto_stock_out INTEGER, warehouse_id INTEGER)`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_flow_event(
			id BIGSERIAL PRIMARY KEY, source_type TEXT, source_id INTEGER,
			from_step_id INTEGER, to_step_id INTEGER, trigger_action TEXT, trace_id TEXT,
			status TEXT, error TEXT, payload_json TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_piecework_summary(
			id BIGSERIAL PRIMARY KEY, worker_id INTEGER, process_id INTEGER, biz_date TEXT,
			qty DOUBLE PRECISION, weight DOUBLE PRECISION, input_weight DOUBLE PRECISION, output_weight DOUBLE PRECISION, loss DOUBLE PRECISION, utilization DOUBLE PRECISION,
			amount DOUBLE PRECISION, source_report_ids TEXT, status TEXT, updated_at TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS pay_process_wage_rate(id BIGSERIAL PRIMARY KEY, process_id INTEGER, rate DOUBLE PRECISION, status TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_process(id BIGSERIAL PRIMARY KEY, is_piecework INTEGER)`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_process_return(
			id BIGSERIAL PRIMARY KEY, doc_no TEXT UNIQUE, box_code TEXT, process_id INTEGER,
			step_id INTEGER, warehouse_id INTEGER, return_weight DOUBLE PRECISION, reason TEXT, status TEXT,
			applicant_user_id INTEGER, foreman_user_id INTEGER, warehouse_user_id INTEGER,
			current_assignee_user_id INTEGER, report_work_id INTEGER, stock_txn_id INTEGER,
			remark TEXT, created_at TEXT DEFAULT NOW(), updated_at TEXT DEFAULT NOW(),
			posted_at TEXT, is_deleted INTEGER DEFAULT 0)`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_process_return_log(
			id BIGSERIAL PRIMARY KEY, return_id INTEGER, action TEXT,
			from_user_id INTEGER, to_user_id INTEGER, remark TEXT, created_at TEXT DEFAULT NOW())`,
		`CREATE TEMP TABLE IF NOT EXISTS pd_report_work(
			id BIGSERIAL PRIMARY KEY, doc_no TEXT, dispatch_id INTEGER, work_order_id INTEGER,
			process_id INTEGER, worker_id INTEGER, qty DOUBLE PRECISION, weight DOUBLE PRECISION, input_weight DOUBLE PRECISION,
			output_weight DOUBLE PRECISION, loss DOUBLE PRECISION, utilization DOUBLE PRECISION, status TEXT, reported_at TEXT,
			scan_code TEXT, confirmed_by INTEGER, confirmed_at TEXT, confirmed_snapshot_json TEXT,
			process_qc_result TEXT, bag_qty DOUBLE PRECISION, operator_user_id INTEGER, operator_employee_id INTEGER, created_by INTEGER)`,
		`CREATE TEMP TABLE IF NOT EXISTS hr_employee(id BIGSERIAL PRIMARY KEY, name TEXT, badge_code TEXT, emp_no TEXT, status TEXT DEFAULT 'active', is_deleted INTEGER DEFAULT 0)`,
		`CREATE TEMP TABLE IF NOT EXISTS biz_audit_log(
			id BIGSERIAL PRIMARY KEY, biz_type TEXT, biz_id INTEGER, action TEXT,
			reason TEXT, before_json TEXT, after_json TEXT, actor_user_id INTEGER, created_at TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS biz_evidence(id BIGSERIAL PRIMARY KEY, biz_type TEXT, biz_id INTEGER, voided_at TEXT)`,
		`CREATE TEMP TABLE IF NOT EXISTS iam_user(id BIGSERIAL PRIMARY KEY, is_deleted INTEGER DEFAULT 0, status TEXT DEFAULT 'active')`,
		`CREATE TEMP TABLE IF NOT EXISTS iam_user_role(user_id INTEGER, role_id INTEGER)`,
		`CREATE TEMP TABLE IF NOT EXISTS iam_role(id BIGSERIAL PRIMARY KEY, code TEXT)`,
		`INSERT INTO inv_box_code(id,code,product_id,warehouse_id,qty,weight,status) VALUES(1,'BX-SMOKE',1,1,100,100,'open')`,
		`INSERT INTO inv_balance(warehouse_id,location_id,product_id,batch_no,box_code_id,qty) VALUES(1,0,1,'',0,500)`,
		`INSERT INTO inv_balance(warehouse_id,location_id,product_id,batch_no,box_code_id,qty) VALUES(2,0,1,'',0,0)`,
		`INSERT INTO pd_routing_step(id,routing_id,seq_no,process_id,step_code,step_name,is_piecework,is_inbound_checkpoint,checkpoint_bind_warehouse,auto_next,auto_stock_in,auto_stock_out,warehouse_id)
			VALUES(10,1,1,1,'CP','卡点绑仓',0,1,1,0,1,1,2)`,
		`INSERT INTO pd_routing_step(id,routing_id,seq_no,process_id,step_code,step_name,is_piecework,is_inbound_checkpoint,checkpoint_bind_warehouse,auto_next,auto_stock_in,auto_stock_out,warehouse_id)
			VALUES(11,1,2,2,'PW','计件步',1,0,0,0,0,1,1)`,
		`INSERT INTO pd_routing_step(id,routing_id,seq_no,process_id,step_code,step_name,is_piecework,is_inbound_checkpoint,checkpoint_bind_warehouse,auto_next,auto_stock_in,auto_stock_out,warehouse_id)
			VALUES(12,1,3,3,'FX','非计件',0,0,0,0,0,0,NULL)`,
		`INSERT INTO pay_process_wage_rate(id,process_id,rate,status) VALUES(1,2,0.2,'active')`,
		`INSERT INTO pd_process(id,is_piecework) VALUES(1,0),(2,1),(3,0)`,
		`INSERT INTO iam_user(id,status) VALUES(8,'active'),(9,'active')`,
		`INSERT INTO iam_role(id,code) VALUES(1,'foreman'),(2,'warehouse')`,
		`INSERT INTO iam_user_role(user_id,role_id) VALUES(9,1),(8,2)`,
		`INSERT INTO pd_report_work(id,doc_no,status,confirmed_at,scan_code,input_weight,qty) VALUES(1,'RW1','posted',NOW(),'BX-SMOKE',70,70)`,
		`INSERT INTO inv_stock_txn(doc_no,doc_type,biz_date,status,warehouse_id,remark) VALUES('C1','consume',CURRENT_DATE::text,'posted',1,'auto:BX-SMOKE')`,
		`INSERT INTO inv_stock_txn_line(txn_id,line_no,product_id,qty,base_qty,direction) VALUES(1,1,1,100,100,'out')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("schema: %v\n%s", err, s)
		}
	}
	return db
}

func TestCheckpointBindStockOrderAndPieceworkGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openSmokeDB(t)
	defer db.Close()
	s := &Services{DB: db}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxClaims, &security.Claims{UserID: 1, Roles: []string{"piece"}})

	out := s.AfterReportWork(c, 100, 1, 0, 50, "BX-SMOKE", 0, 0, 10, 50, 0, 1)
	next, _ := out["next"].(gin.H)
	if next["stock_order"] != "in_then_out" {
		t.Fatalf("checkpoint bind expect in_then_out, got %#v", next["stock_order"])
	}
	if out["piecework"] != false {
		t.Fatalf("checkpoint non-piecework should not write piecework")
	}

	out2 := s.AfterReportWork(c, 101, 2, 7, 40, "BX-SMOKE", 0, 0, 11, 40, 0, 1)
	if out2["piecework"] != true {
		t.Fatalf("piecework step should write piecework")
	}
	var pwQty float64
	_ = db.QueryRow(`SELECT qty FROM pd_piecework_summary WHERE worker_id=7 AND process_id=2`).Scan(&pwQty)
	if pwQty != 40 {
		t.Fatalf("piecework qty want 40 got %v", pwQty)
	}

	out3 := s.AfterReportWork(c, 102, 3, 7, 10, "BX-SMOKE", 0, 0, 12, 10, 0, 1)
	if out3["piecework"] != false {
		t.Fatalf("non-piecework step must skip piecework")
	}
}

func withBody(c *gin.Context, body map[string]interface{}) {
	b, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")
}

func TestProcessReturnTwoStageKeepsPiecework(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openSmokeDB(t)
	defer db.Close()
	s := &Services{DB: db}

	_, _ = db.Exec(`INSERT INTO pd_piecework_summary(worker_id,process_id,biz_date,qty,amount,status) VALUES(7,2,date('now'),70,14,'open')`)
	var before float64
	_ = db.QueryRow(`SELECT qty FROM pd_piecework_summary WHERE worker_id=7 AND process_id=2`).Scan(&before)

	mk := func(roles []string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(middleware.CtxClaims, &security.Claims{UserID: 1, Roles: roles})
		return c
	}

	c := mk([]string{"piece"})
	withBody(c, map[string]interface{}{
		"box_code": "BX-SMOKE", "return_weight": 30.0, "warehouse_id": 1.0, "reason": "提前下班",
	})
	if !s.createProcessReturn(c) {
		t.Fatal("create not handled")
	}
	var id int64
	_ = db.QueryRow(`SELECT id FROM pd_process_return ORDER BY id DESC LIMIT 1`).Scan(&id)
	if id == 0 {
		t.Fatal("return not created")
	}

	c = mk([]string{"piece"})
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", id)}}
	withBody(c, map[string]interface{}{})
	if !s.submitProcessReturn(c) {
		t.Fatal("submit")
	}
	c = mk([]string{"foreman"})
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", id)}}
	withBody(c, map[string]interface{}{})
	if !s.approveProcessReturn(c) {
		t.Fatal("approve")
	}
	var balBefore float64
	_ = db.QueryRow(`SELECT qty FROM inv_balance WHERE warehouse_id=1 AND product_id=1`).Scan(&balBefore)

	c = mk([]string{"warehouse"})
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", id)}}
	withBody(c, map[string]interface{}{})
	if !s.warehouseConfirmProcessReturn(c) {
		t.Fatal("warehouse confirm")
	}
	var st string
	var txnID int64
	_ = db.QueryRow(`SELECT status, COALESCE(stock_txn_id,0) FROM pd_process_return WHERE id=?`, id).Scan(&st, &txnID)
	if st != "posted" || txnID == 0 {
		t.Fatalf("want posted with txn, got %s txn=%d", st, txnID)
	}
	var balAfter float64
	_ = db.QueryRow(`SELECT qty FROM inv_balance WHERE warehouse_id=1 AND product_id=1`).Scan(&balAfter)
	if balAfter < balBefore+29.9 {
		t.Fatalf("stock should increase by ~30: before=%v after=%v", balBefore, balAfter)
	}
	var after float64
	_ = db.QueryRow(`SELECT qty FROM pd_piecework_summary WHERE worker_id=7 AND process_id=2`).Scan(&after)
	if after != before {
		t.Fatalf("piecework must not reverse: before=%v after=%v", before, after)
	}
	var docType string
	_ = db.QueryRow(`SELECT doc_type FROM inv_stock_txn WHERE id=?`, txnID).Scan(&docType)
	if docType != "process_return" {
		t.Fatalf("doc_type want process_return got %s", docType)
	}
}

func TestVoidReportWorkDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openSmokeDB(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO pd_report_work(id,doc_no,status,scan_code,process_id,worker_id,qty) VALUES(9,'RW-VOID','confirm_pending','BX-SMOKE',1,1,0)`)
	s := &Services{DB: db}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxClaims, &security.Claims{UserID: 1, Roles: []string{"piece"}})
	c.Params = gin.Params{{Key: "id", Value: "9"}}
	withBody(c, map[string]interface{}{"remark": "test"})
	if !s.voidReportWorkDraft(c) {
		t.Fatal("void not handled")
	}
	var st string
	_ = db.QueryRow(`SELECT status FROM pd_report_work WHERE id=9`).Scan(&st)
	if st != "void" {
		t.Fatalf("want void got %s (resp=%s)", st, w.Body.String())
	}
}
