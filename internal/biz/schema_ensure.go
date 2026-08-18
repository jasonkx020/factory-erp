package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Schema DDL is owned by migrations/erp (erp-db / init_schema). Ensure* below are no-ops.

func execSchemaRuns(db *sql.DB, label string, stmts []string) {
	_ = db
	_ = label
	_ = stmts
}

func isIdempotentSchemaErr(err error) bool {
	if err == nil {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate")
}

func EnsureAutomationSchema(db *sql.DB) { _ = db }
func EnsureClosedLoopSchema(db *sql.DB) { _ = db }
func EnsureFarmerSchema(db *sql.DB)     { _ = db }
func EnsurePurchaseSchema(db *sql.DB)   { _ = db }
func EnsureFieldLedgerSchema(db *sql.DB) { _ = db }

// migrateToolIssueToLines is a no-op: line table is in migrations/erp baseline.
func migrateToolIssueToLines(db *sql.DB) { _ = db }

// seedInboundProductRoutings 原料/半成品入厂工艺（按 product_id 解析首步）。
func seedInboundProductRoutings(db *sql.DB) {
	_, _ = db.Exec(`INSERT INTO prd_product(id, code, name, product_type, cost_price, sale_price, status) VALUES
 (1, 'RM-CASSAVA', '鲜木薯', 'raw', 1.2, NULL, 'active'),
 (2, 'SF-COREOUT', '去芯薯肉', 'semi', 2.5, 3.0, 'active'),
 (3, 'FG-DICED', '袋装木薯丁', 'finished', 4.0, 7.0, 'active')`)

	ensureRoutingWithSteps := func(code, name string, productID int64, steps []string) {
		var rid int64
		_ = db.QueryRow(`SELECT id FROM pd_routing WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&rid)
		if rid == 0 {
			res, err := db.Exec(`INSERT INTO pd_routing(code, name, product_id, version_no, status) VALUES(?,?,?,'V1','active')`, code, name, productID)
			if err != nil {
				return
			}
			rid, _ = res.LastInsertId()
		} else {
			_, _ = db.Exec(`UPDATE pd_routing SET product_id=?, name=?, status='active' WHERE id=?`, productID, name, rid)
		}
		var n int
		_ = db.QueryRow(`SELECT COUNT(1) FROM pd_routing_step WHERE routing_id=?`, rid).Scan(&n)
		if n > 0 || rid <= 0 {
			return
		}
		for _, sql := range steps {
			_, _ = db.Exec(strings.ReplaceAll(sql, "__RID__", fmt.Sprintf("%d", rid)))
		}
	}

	rawSteps := []string{
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 1, 8, 'S1', '入库-原料', 0, 0, 1, 1, 0, 1, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 2, 7, 'S2', '清洗', 0, 0, 1, 0, 0, 1, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 3, 1, 'S3', '去皮-计件领料', 1, 0, 1, 0, 1, 1, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 4, 2, 'S4', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 5, 3, 'S5', '切断-固定工', 0, 0, 1, 0, 0, NULL, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 6, 4, 'S6', '去芯-计件', 1, 0, 1, 0, 0, NULL, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 7, 2, 'S7', '收货-固定工', 0, 1, 1, 0, 0, NULL, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 8, 9, 'S8', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 9, 10, 'S9', '出库切块-计件', 1, 0, 1, 0, 1, 2, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 10, 9, 'S10', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 11, 6, 'S11', '过滤装袋', 0, 0, 1, 0, 0, NULL, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 12, 11, 'S12', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 1)`,
	}
	semiSteps := []string{
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 1, 9, 'S1', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 2, 10, 'S2', '出库切块-计件', 1, 0, 1, 0, 1, 2, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 3, 9, 'S3', '入库-半成品库', 0, 0, 1, 1, 0, 2, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 4, 6, 'S4', '过滤装袋', 0, 0, 1, 0, 0, NULL, 1)`,
		`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id) VALUES
 (__RID__, 5, 11, 'S5', '入库-成品库销售', 0, 0, 0, 1, 0, 3, 1)`,
	}
	ensureRoutingWithSteps("RT-CASSAVA-RAW", "鲜木薯入厂产线", 1, rawSteps)
	ensureRoutingWithSteps("RT-CASSAVA-SEMI", "去芯薯肉入厂产线", 2, semiSteps)
	if ws := defaultWorkshopDeptIDDB(db); ws > 0 {
		_, _ = db.Exec(`UPDATE pd_routing_step SET workshop_dept_id=? WHERE COALESCE(workshop_dept_id,0) IN (0,1)`, ws)
	}
}

// SeedOpenShiftForToday ensures demo workers can pass shift gate when shifts are enabled.
func SeedOpenShiftForToday(db *sql.DB) {
	seedOpenShift(db)
}

func seedOpenShift(db *sql.DB) {
	var sid int64
	_ = db.QueryRow(`SELECT id FROM pd_shift WHERE status='open' AND (biz_date::date)=CURRENT_DATE ORDER BY id DESC LIMIT 1`).Scan(&sid)
	if sid == 0 {
		ws := defaultWorkshopDeptIDDB(db)
		res, err := db.Exec(`INSERT INTO pd_shift(doc_no, workshop_dept_id, biz_date, status, remark) VALUES(?,?,CURRENT_DATE,'open','demo open shift')`,
			fmt.Sprintf("SHF%d", time.Now().UnixNano()%1e9), nullIf0(ws))
		if err != nil {
			return
		}
		sid, _ = res.LastInsertId()
	}
	if sid == 0 {
		return
	}
	seedShiftMembers(db, sid, `SELECT id FROM hr_employee WHERE badge_code IN ('EMP-PC','EMP-FX','EMP0301','EMP0205') AND status='active'`)
	seedShiftMembers(db, sid, `SELECT e.id FROM hr_employee e
		INNER JOIN iam_user u ON u.employee_id=e.id AND COALESCE(u.is_deleted,0)=0
		INNER JOIN iam_user_role ur ON ur.user_id=u.id
		INNER JOIN iam_role r ON r.id=ur.role_id
		WHERE r.code IN ('piece','fixed','foreman') AND e.status='active'`)
}

func seedShiftMembers(db *sql.DB, shiftID int64, query string) {
	rows, err := db.Query(query)
	if err != nil || rows == nil {
		return
	}
	ids := make([]int64, 0, 16)
	for rows.Next() {
		var eid int64
		if rows.Scan(&eid) == nil && eid > 0 {
			ids = append(ids, eid)
		}
	}
	_ = rows.Close()
	for _, eid := range ids {
		_, _ = db.Exec(`INSERT INTO pd_shift_member(shift_id, employee_id, process_id) VALUES(?,?,0) ON CONFLICT DO NOTHING`, shiftID, eid)
	}
}

// looksLikeEncodingCorruption detects Windows/latin1-style CJK→'?' replacement
// (ASCII '?' / '-' / spaces only, and at least one '?').
func looksLikeEncodingCorruption(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !strings.Contains(s, "?") {
		return false
	}
	for _, r := range s {
		if r != '?' && r != '-' && r != ' ' && r != '_' && r != '/' {
			return false
		}
	}
	return true
}

// cassavaRoutingStepNames is the canonical demo RT-CASSAVA step labels (UTF-8).
var cassavaRoutingStepNames = map[string]string{
	"S1":  "入库-原料",
	"S2":  "清洗",
	"S3":  "去皮-计件领料",
	"S4":  "收货-固定工",
	"S5":  "切断-固定工",
	"S6":  "去芯-计件",
	"S7":  "收货-固定工",
	"S8":  "入库-半成品库",
	"S9":  "出库切块-计件",
	"S10": "入库-半成品库",
	"S11": "过滤装袋",
	"S12": "入库-成品库销售",
}

// repairCassavaRoutingSeed fixes demo routing/step Chinese that was stored as '?' on some Windows DBs.
// Returns true when the production flow graph should be rebuilt from steps.
func repairCassavaRoutingSeed(db *sql.DB) bool {
	if db == nil {
		return false
	}
	rebuild := false
	var routingName string
	_ = db.QueryRow(`SELECT COALESCE(name,'') FROM pd_routing WHERE code='RT-CASSAVA'`).Scan(&routingName)
	if looksLikeEncodingCorruption(routingName) {
		_, _ = db.Exec(`UPDATE pd_routing SET name=? WHERE code='RT-CASSAVA'`, "木薯丁产线")
		rebuild = true
	}
	type stepFix struct {
		id   int64
		want string
	}
	rows, err := db.Query(`SELECT id, COALESCE(step_code,''), COALESCE(step_name,'') FROM pd_routing_step WHERE routing_id=1`)
	if err != nil {
		return rebuild
	}
	var fixes []stepFix
	for rows.Next() {
		var id int64
		var code, name string
		if err := rows.Scan(&id, &code, &name); err != nil {
			continue
		}
		want, ok := cassavaRoutingStepNames[code]
		if !ok {
			continue
		}
		if looksLikeEncodingCorruption(name) || name == "" {
			fixes = append(fixes, stepFix{id: id, want: want})
		}
	}
	_ = rows.Close()
	for _, f := range fixes {
		if _, err := db.Exec(`UPDATE pd_routing_step SET step_name=? WHERE id=?`, f.want, f.id); err == nil {
			rebuild = true
		}
	}
	return rebuild
}

func productionFlowGraphCorrupt(db *sql.DB) bool {
	var name, gjson string
	err := db.QueryRow(`SELECT COALESCE(name,''), COALESCE(graph_json,'') FROM pd_flow_graph
		WHERE code='RT-CASSAVA' AND COALESCE(is_deleted,0)=0`).Scan(&name, &gjson)
	if err != nil {
		return false
	}
	if looksLikeEncodingCorruption(name) {
		return true
	}
	// node labels baked at seed time from corrupted step_name
	return strings.Contains(gjson, `"label":"??`) || strings.Contains(gjson, `"step_name":"??`)
}

func seedFlowGraphs(db *sql.DB) {
	if db == nil {
		return
	}
	gateJSON := purchaseGateFlowJSON()
	stockJSON := purchaseStockinFlowJSON()
	rebuildProd := repairCassavaRoutingSeed(db) || productionFlowGraphCorrupt(db)

	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM pd_flow_graph WHERE COALESCE(is_deleted,0)=0`).Scan(&n)
	if n == 0 {
		seedFlowGraphsInitial(db, gateJSON, stockJSON)
	} else if rebuildProd {
		rebuildProductionFlowGraph(db)
	}
	// migrate: 入厂默认图改为 采购→仓管→财务（去掉质检必经）
	_, _ = db.Exec(`UPDATE pd_flow_graph SET graph_json=?, name='过磅入厂流程', version_no='V2', updated_at=NOW()
		WHERE code='PURCHASE_GATE' AND COALESCE(is_deleted,0)=0`, gateJSON)
	_, _ = db.Exec(`UPDATE pd_flow_graph SET status='active' WHERE code='PURCHASE_GATE' AND COALESCE(is_deleted,0)=0`)
}

func purchaseGateFlowJSON() string {
	return `{"nodes":[
{"id":"start","type":"start","position":{"x":40,"y":80},"data":{}},
{"id":"submit","type":"role_task","position":{"x":220,"y":80},"data":{"role_code":"purchase","action":"submit","label":"采购提交"}},
{"id":"wh","type":"role_task","position":{"x":420,"y":80},"data":{"role_code":"warehouse","action":"warehouse_confirm","label":"仓管入库"}},
{"id":"fin","type":"role_task","position":{"x":620,"y":80},"data":{"role_code":"finance","action":"pay","label":"财务付款"}},
{"id":"end","type":"end","position":{"x":820,"y":80},"data":{}}
],"edges":[
{"id":"e1","source":"start","target":"submit","data":{"is_default":true}},
{"id":"e2","source":"submit","target":"wh","data":{"is_default":true}},
{"id":"e3","source":"wh","target":"fin","data":{"is_default":true}},
{"id":"e4","source":"fin","target":"end","data":{"is_default":true}}
]}`
}

func purchaseStockinFlowJSON() string {
	return `{"nodes":[
{"id":"start","type":"start","position":{"x":40,"y":80},"data":{}},
{"id":"submit","type":"role_task","position":{"x":220,"y":80},"data":{"role_code":"purchase","action":"submit","label":"采购提交"}},
{"id":"wh","type":"role_task","position":{"x":420,"y":80},"data":{"role_code":"warehouse","action":"warehouse_confirm","label":"仓管入库"}},
{"id":"end","type":"end","position":{"x":620,"y":80},"data":{}}
],"edges":[
{"id":"e1","source":"start","target":"submit","data":{"is_default":true}},
{"id":"e2","source":"submit","target":"wh","data":{"is_default":true}},
{"id":"e3","source":"wh","target":"end","data":{"is_default":true}}
]}`
}

func buildProductionFlowJSONFromRouting(db *sql.DB, routingID int64) string {
	prodJSON := `{"nodes":[{"id":"start","type":"start","position":{"x":80,"y":200},"data":{}},{"id":"end","type":"end","position":{"x":80,"y":200},"data":{}}],"edges":[]}`
	type stepRow struct {
		seq                     int
		pid, wh                 int64
		code, name              string
		piece, cp, an, asi, aso int
	}
	rows, err := db.Query(`SELECT seq_no, process_id, COALESCE(step_code,''), COALESCE(step_name,''),
		COALESCE(is_piecework,0), COALESCE(is_inbound_checkpoint,0), COALESCE(auto_next,1),
		COALESCE(auto_stock_in,0), COALESCE(auto_stock_out,0), COALESCE(warehouse_id,0)
		FROM pd_routing_step WHERE routing_id=? ORDER BY seq_no`, routingID)
	if err != nil {
		return prodJSON
	}
	defer rows.Close()
	var steps []stepRow
	for rows.Next() {
		var r stepRow
		_ = rows.Scan(&r.seq, &r.pid, &r.code, &r.name, &r.piece, &r.cp, &r.an, &r.asi, &r.aso, &r.wh)
		if want, ok := cassavaRoutingStepNames[r.code]; ok && (looksLikeEncodingCorruption(r.name) || r.name == "") {
			r.name = want
		}
		steps = append(steps, r)
	}
	if len(steps) == 0 {
		return prodJSON
	}
	nodes := []map[string]interface{}{
		{"id": "start", "type": "start", "position": map[string]float64{"x": 40, "y": 40}, "data": map[string]interface{}{}},
	}
	edges := []map[string]interface{}{}
	prev := "start"
	for i, st := range steps {
		nid := fmt.Sprintf("ps%d", st.seq)
		nodes = append(nodes, map[string]interface{}{
			"id": nid, "type": "process_step",
			"position": map[string]float64{"x": float64(40 + (i%4)*220), "y": float64(140 + (i/4)*140)},
			"data": map[string]interface{}{
				"process_id": st.pid, "step_code": st.code, "step_name": st.name,
				"is_piecework": st.piece == 1, "is_inbound_checkpoint": st.cp == 1,
				"auto_next": st.an == 1, "auto_stock_in": st.asi == 1, "auto_stock_out": st.aso == 1,
				"warehouse_id": st.wh, "label": st.name,
			},
		})
		edges = append(edges, map[string]interface{}{
			"id": fmt.Sprintf("e_%s_%s", prev, nid), "source": prev, "target": nid,
			"data": map[string]interface{}{"is_default": true},
		})
		prev = nid
	}
	nodes = append(nodes, map[string]interface{}{
		"id": "end", "type": "end", "position": map[string]float64{"x": 40, "y": float64(140 + ((len(steps)+3)/4)*140)},
		"data": map[string]interface{}{},
	})
	edges = append(edges, map[string]interface{}{
		"id": fmt.Sprintf("e_%s_end", prev), "source": prev, "target": "end",
		"data": map[string]interface{}{"is_default": true},
	})
	b, _ := json.Marshal(map[string]interface{}{"nodes": nodes, "edges": edges})
	return string(b)
}

func rebuildProductionFlowGraph(db *sql.DB) {
	prodJSON := buildProductionFlowJSONFromRouting(db, 1)
	_, _ = db.Exec(`UPDATE pd_flow_graph SET name=?, graph_json=?, version_no='V1-fixed', updated_at=NOW()
		WHERE code='RT-CASSAVA' AND COALESCE(is_deleted,0)=0`, "木薯丁产线", prodJSON)
	_, _ = db.Exec(`UPDATE pd_routing SET name=?, graph_json=? WHERE code='RT-CASSAVA'`, "木薯丁产线", prodJSON)
}

func seedFlowGraphsInitial(db *sql.DB, gateJSON, stockJSON string) {
	prodJSON := buildProductionFlowJSONFromRouting(db, 1)
	_, _ = db.Exec(`INSERT INTO pd_flow_graph(code, name, kind, status, routing_id, graph_json, version_no) VALUES(?,?,?,?,?,?,?)`,
		"RT-CASSAVA", "木薯丁产线", "production", "active", 1, prodJSON, "V1")
	_, _ = db.Exec(`UPDATE pd_routing SET graph_json=? WHERE id=1`, prodJSON)
	_, _ = db.Exec(`INSERT INTO pd_flow_graph(code, name, kind, status, graph_json, version_no) VALUES(?,?,?,?,?,?)`,
		"PURCHASE_GATE", "过磅入厂流程", "purchase_gate", "active", gateJSON, "V2")
	_, _ = db.Exec(`INSERT INTO pd_flow_graph(code, name, kind, status, graph_json, version_no) VALUES(?,?,?,?,?,?)`,
		"PURCHASE_STOCKIN", "过磅入库流程", "purchase_stockin", "active", stockJSON, "V1")
}

