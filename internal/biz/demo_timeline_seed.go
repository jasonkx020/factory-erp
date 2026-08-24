package biz

import (
	"database/sql"
	"log"
	"time"
)

const demoTimelineVersion = "v1"

// ensureDemoTimelineData seeds trace-production showcase + 7-day timeline rows for full UI demo.
func ensureDemoTimelineData(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT)`)
	var ver string
	_ = db.QueryRow(`SELECT value FROM schema_meta WHERE key='demo_timeline_version'`).Scan(&ver)
	if ver == demoTimelineVersion {
		return
	}
	clearDemoTimeline(db)
	seedDemoTimeline(db)
	_, _ = db.Exec(`INSERT INTO schema_meta(key, value) VALUES('demo_timeline_version', ?)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, demoTimelineVersion)
	log.Printf("demo timeline data ensured (%s)", demoTimelineVersion)
}

func clearDemoTimeline(db *sql.DB) {
	stmts := []string{
		`DELETE FROM pd_trace_process_log WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_trace_process_yield WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_process_move_alloc WHERE move_id IN (SELECT id FROM pd_process_move WHERE UPPER(trace_code) LIKE 'TR-DEMO-%')`,
		`DELETE FROM pd_process_move WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_process_issue WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_trace_production WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM inv_box_code WHERE UPPER(trace_code) LIKE 'TR-DEMO-%' OR code LIKE 'BX-TR-DEMO-%'`,
		`DELETE FROM pur_trace_lot WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pur_weigh_ticket WHERE doc_no LIKE 'DEMO-WT-TR-%' OR UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_station_flow_log WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_routing_step WHERE routing_id IN (SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE')`,
		`DELETE FROM pd_routing WHERE code='RT-DEMO-TRACE'`,
		`DELETE FROM prd_product WHERE code='RM-DEMO-TRACE'`,
	}
	for _, s := range stmts {
		_, _ = db.Exec(s)
	}
}

func demoTS(daysAgo int, hour, min int) string {
	t := time.Now().AddDate(0, 0, -daysAgo)
	return time.Date(t.Year(), t.Month(), t.Day(), hour, min, 0, 0, t.Location()).Format("2006-01-02 15:04:05")
}

func demoDate(daysAgo int) string {
	return time.Now().AddDate(0, 0, -daysAgo).Format("2006-01-02")
}

func seedDemoTimeline(db *sql.DB) {
	today := demoDate(0)
	day7 := demoDate(7)

	farmerID := demoID(db, `SELECT id FROM pur_farmer WHERE code='FM01'`)
	if farmerID == 0 {
		farmerID = 1
	}
	workerID := demoID(db, `SELECT id FROM hr_employee WHERE badge_code='EMP0301'`)
	if workerID == 0 {
		workerID = 2
	}

	_, _ = db.Exec(`INSERT INTO prd_product(code, name, product_type, cost_price, sale_price, status, spec_text)
		VALUES('RM-DEMO-TRACE','演示溯源鲜薯','raw',1.2,NULL,'active','溯源工序演示专用')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, status='active'`)
	prodID := demoID(db, `SELECT id FROM prd_product WHERE code='RM-DEMO-TRACE'`)
	if prodID <= 0 {
		return
	}
	var routingID int64
	_ = db.QueryRow(`SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE' AND COALESCE(is_deleted,0)=0`).Scan(&routingID)
	if routingID <= 0 {
		res, err := db.Exec(`INSERT INTO pd_routing(code, name, product_id, version_no, status)
			VALUES('RT-DEMO-TRACE','演示溯源短工艺',?,'V1','active')`, prodID)
		if err == nil {
			routingID, _ = res.LastInsertId()
		}
	}
	if routingID <= 0 {
		return
	}
	stepDefs := []struct {
		seq, proc       int
		code, name      string
		piece           int
	}{
		{1, 7, "D-WASH", "清洗", 0},
		{2, 1, "D-PEEL", "去皮", 1},
		{3, 3, "D-CUT", "切断", 0},
		{4, 6, "D-BAG", "装袋", 0},
	}
	stepIDs := map[int]int64{}
	for _, sd := range stepDefs {
		var sid int64
		_ = db.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? AND seq_no=?`, routingID, sd.seq).Scan(&sid)
		if sid <= 0 {
			res, err := db.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id)
				VALUES(?,?,?,?,?,?,0,1,0,0,1)`, routingID, sd.seq, sd.proc, sd.code, sd.name, sd.piece)
			if err == nil {
				sid, _ = res.LastInsertId()
			}
		}
		stepIDs[sd.seq] = sid
	}

	seedTraceCompleted7D(db, prodID, farmerID, workerID, stepIDs, day7)
	seedTraceInProgressToday(db, prodID, farmerID, workerID, stepIDs, today)
	seedTraceInStockToday(db, prodID, farmerID, today)
	seedTimelineHistory(db, workerID)
}

func seedTraceCompleted7D(db *sql.DB, prodID, farmerID, workerID int64, stepIDs map[int]int64, bizDate string) {
	const trace = "TR-DEMO-7D001"
	startTS := demoTS(7, 8, 0)
	endTS := demoTS(7, 17, 30)

	_, _ = db.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, product_id, gross_weight, deduct_weight, net_weight, qc_result, status, biz_date, remark, receive_kind, batch_no, trace_code)
		VALUES('DEMO-WT-TR-7D',?,?,1250,250,1000,'pass','weighed',?,'7天前已完成溯源演示','gate',?,?)`,
		farmerID, prodID, bizDate, trace, trace)
	wtID := demoID(db, `SELECT id FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-TR-7D'`)
	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, payload_canonical, signature, status)
		VALUES(?,?,?,?,'A',?,1000,'{"demo":true}','demo','closed')`, trace, bizDate, trace, farmerID, nullIf0(wtID))

	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id, current_step_id)
		VALUES('BX-TR-DEMO-7D-A',?,1,?,850,850,?,?,'open',6,?)`,
		prodID, trace, farmerID, trace, nullIf0(stepIDs[4]))
	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id, current_step_id)
		VALUES('BX-TR-DEMO-7D-B',?,3,?,0,0,?,?,'in_stock',11,?)`,
		prodID, trace, farmerID, trace, nullIf0(stepIDs[4]))
	boardA := demoID(db, `SELECT id FROM inv_box_code WHERE code='BX-TR-DEMO-7D-A'`)

	res, _ := db.Exec(`INSERT INTO pd_trace_production(trace_code, status, started_by, completed_by, started_at, completed_at, input_kg, output_kg, loss_rate, remark)
		VALUES(?,'done',1,1,?,?,1000,850,0.15,'演示：全部工序已结束并自动结案')`, trace, startTS, endTS)
	sessID, _ := res.LastInsertId()

	type procYield struct {
		pid                       int64
		inKg, outKg, lossKg, rate float64
		at                        string
	}
	yields := []procYield{
		{7, 1000, 980, 20, 0.02, demoTS(7, 9, 30)},
		{1, 980, 850, 130, 0.1327, demoTS(7, 11, 0)},
		{3, 850, 840, 10, 0.0118, demoTS(7, 14, 0)},
		{6, 840, 850, 0, 0, demoTS(7, 16, 30)},
	}
	for _, y := range yields {
		pname := ""
		_ = db.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, y.pid).Scan(&pname)
		_, _ = db.Exec(`INSERT INTO pd_trace_process_yield(trace_code, process_id, input_kg, output_kg, loss_kg, loss_rate, board_count, created_at)
			VALUES(?,?,?,?,?,?,2,?)
			ON CONFLICT (trace_code, process_id) DO UPDATE SET input_kg=EXCLUDED.input_kg, output_kg=EXCLUDED.output_kg, loss_kg=EXCLUDED.loss_kg, loss_rate=EXCLUDED.loss_rate`,
			trace, y.pid, y.inKg, y.outKg, y.lossKg, y.rate, y.at)
		_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, process_id, process_name, event_type, actor_user_id, input_kg, output_kg, loss_rate, created_at)
			VALUES(?,?,?,?,'process_complete',1,?,?,?,?)`,
			sessID, trace, y.pid, pname, y.inKg, y.outKg, y.rate, y.at)
	}

	if boardA > 0 {
		_, _ = db.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status, source)
			VALUES(?,'BX-TR-DEMO-7D-A',?,?,?,?,800,0,800,'closed','process')`,
			boardA, trace, 1, nullIf0(stepIDs[2]), workerID)
		_, _ = db.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, from_step_id, to_process_id, to_step_id, kg, move_kind, created_by, created_at)
			VALUES(?,'BX-TR-DEMO-7D-A',?,7,?,?,1,?,980,'next',1,?)`,
			boardA, trace, nullIf0(stepIDs[1]), nullIf0(stepIDs[2]), nullIf0(stepIDs[2]), demoTS(7, 10, 0))
		_, _ = db.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, from_step_id, kg, move_kind, created_by, created_at)
			VALUES(?,'BX-TR-DEMO-7D-A',?,6,?,850,'stock_in',1,?)`,
			boardA, trace, nullIf0(stepIDs[4]), demoTS(7, 17, 0))
	}
	_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, remark, created_at)
		VALUES(?,?,'session_start',1,'进入生产',?)`, sessID, trace, startTS)
	_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, input_kg, output_kg, loss_rate, remark, created_at)
		VALUES(?,?,'session_complete',1,1000,850,0.15,'自动结案',?)`, sessID, trace, endTS)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, board_code, trace_code, process_id, worker_id, kg, remark, created_at)
		VALUES('issue', ?, 'BX-TR-DEMO-7D-A', ?, 1, ?, 800, '演示领料', ?)`, bizDate, trace, workerID, demoTS(7, 11, 0))
}

func seedTraceInProgressToday(db *sql.DB, prodID, farmerID, workerID int64, stepIDs map[int]int64, bizDate string) {
	const trace = "TR-DEMO-TODAY001"
	startTS := demoTS(0, 8, 30)

	_, _ = db.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, product_id, gross_weight, deduct_weight, net_weight, qc_result, status, biz_date, remark, receive_kind, batch_no, trace_code)
		VALUES('DEMO-WT-TR-T1',?,?,600,100,500,'pass','weighed',?,'今日生产中溯源演示','gate',?,?)`,
		farmerID, prodID, bizDate, trace, trace)
	wtID := demoID(db, `SELECT id FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-TR-T1'`)
	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, status)
		VALUES(?,?,?,?,'A',?,500,'open')`, trace, bizDate, trace, farmerID, nullIf0(wtID))

	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id, current_step_id)
		VALUES('BX-TR-DEMO-T1-A',?,1,?,180,180,?,?,'open',3,?)`,
		prodID, trace, farmerID, trace, nullIf0(stepIDs[3]))
	boardA := demoID(db, `SELECT id FROM inv_box_code WHERE code='BX-TR-DEMO-T1-A'`)

	res, _ := db.Exec(`INSERT INTO pd_trace_production(trace_code, status, started_by, started_at, remark)
		VALUES(?,'in_progress',1,?,'演示：前两道工序已结束，切断工序进行中')`, trace, startTS)
	sessID, _ := res.LastInsertId()

	for _, row := range []struct {
		pid                       int64
		inKg, outKg, loss, rate   float64
		at                        string
	}{
		{7, 500, 490, 10, 0.02, demoTS(0, 9, 0)},
		{1, 490, 420, 70, 0.1429, demoTS(0, 10, 30)},
	} {
		pname := ""
		_ = db.QueryRow(`SELECT COALESCE(name,'') FROM pd_process WHERE id=?`, row.pid).Scan(&pname)
		_, _ = db.Exec(`INSERT INTO pd_trace_process_yield(trace_code, process_id, input_kg, output_kg, loss_kg, loss_rate, board_count)
			VALUES(?,?,?,?,?,?,1) ON CONFLICT (trace_code, process_id) DO UPDATE SET input_kg=EXCLUDED.input_kg, output_kg=EXCLUDED.output_kg, loss_kg=EXCLUDED.loss_kg, loss_rate=EXCLUDED.loss_rate`,
			trace, row.pid, row.inKg, row.outKg, row.loss, row.rate)
		_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, process_id, process_name, event_type, actor_user_id, input_kg, output_kg, loss_rate, created_at)
			VALUES(?,?,?,?,'process_complete',1,?,?,?,?)`,
			sessID, trace, row.pid, pname, row.inKg, row.outKg, row.rate, row.at)
	}

	if boardA > 0 {
		_, _ = db.Exec(`INSERT INTO pd_process_issue(board_id, board_code, trace_code, process_id, step_id, worker_id, issue_kg, returned_kg, completed_kg, status, source)
			VALUES(?,'BX-TR-DEMO-T1-A',?,?,?,?,180,0,0,'open','process')`,
			boardA, trace, 3, nullIf0(stepIDs[3]), workerID)
	}
	_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, remark, created_at)
		VALUES(?,?,'session_start',1,'今日进入生产',?)`, sessID, trace, startTS)
}

func seedTraceInStockToday(db *sql.DB, prodID, farmerID int64, bizDate string) {
	const trace = "TR-DEMO-TODAY002"
	_, _ = db.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, product_id, gross_weight, deduct_weight, net_weight, qc_result, status, biz_date, remark, receive_kind, batch_no, trace_code)
		VALUES('DEMO-WT-TR-T2',?,?,360,60,300,'pass','weighed',?,'今日库中溯源（未进入生产）','gate',?,?)`,
		farmerID, prodID, bizDate, trace, trace)
	wtID := demoID(db, `SELECT id FROM pur_weigh_ticket WHERE doc_no='DEMO-WT-TR-T2'`)
	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, status)
		VALUES(?,?,?,?,'B',?,300,'open')`, trace, bizDate, trace, farmerID, nullIf0(wtID))
	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id)
		VALUES('BX-TR-DEMO-T2-A',?,1,?,300,300,?,?,'open',8)`,
		prodID, trace, farmerID, trace)
}

func seedTimelineHistory(db *sql.DB, workerID int64) {
	day7 := demoDate(7)
	today := demoDate(0)
	for d := 7; d >= 0; d-- {
		biz := demoDate(d)
		_, _ = db.Exec(`INSERT INTO hr_attendance_record(employee_id, biz_date, check_in_at, check_out_at, source)
			SELECT ?, ?, ? || ' 08:05:00', ? || ' 18:10:00', 'demo_timeline'
			WHERE NOT EXISTS (SELECT 1 FROM hr_attendance_record WHERE employee_id=? AND biz_date=? AND source='demo_timeline')`,
			workerID, biz, biz, biz, workerID, biz)
		qty := 400.0 + float64(d*15)
		amt := qty * 0.18
		_, _ = db.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, amount)
			SELECT ?, 1, ?, ?, ?
			WHERE NOT EXISTS (SELECT 1 FROM pd_piecework_summary WHERE worker_id=? AND process_id=1 AND biz_date=? AND qty=?)`,
			workerID, biz, qty, amt, workerID, biz, qty)
	}
	_, _ = db.Exec(`INSERT INTO rpt_report_snapshot(report_code, biz_date, payload_json)
		SELECT 'daily', ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM rpt_report_snapshot WHERE report_code='daily' AND biz_date=? AND payload_json LIKE '%demo_timeline%')`,
		day7, `{"note":"demo_timeline","sales_amount":9800,"trace_completed":"TR-DEMO-7D001"}`, day7)
	_, _ = db.Exec(`INSERT INTO rpt_report_snapshot(report_code, biz_date, payload_json)
		SELECT 'daily', ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM rpt_report_snapshot WHERE report_code='daily' AND biz_date=? AND payload_json LIKE '%demo_timeline%')`,
		today, `{"note":"demo_timeline","sales_amount":13600,"trace_in_progress":"TR-DEMO-TODAY001"}`, today)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, process_name, kg, remark, created_at)
		VALUES('trace_start', ?, 'TR-DEMO-7D001', '溯源生产', 1000, '7天前启动', ?)`, day7, demoTS(7, 8, 0))
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, process_name, kg, remark, created_at)
		VALUES('trace_start', ?, 'TR-DEMO-TODAY001', '溯源生产', 500, '今日启动', ?)`, today, demoTS(0, 8, 30))
}
