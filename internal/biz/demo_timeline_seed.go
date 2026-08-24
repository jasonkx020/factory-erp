package biz

import (
	"database/sql"
	"log"
	"time"
)

const demoTimelineVersion = "v6"

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
		`DELETE FROM pur_inbound_arrival WHERE doc_no LIKE 'DEMO-ARR-TR-%'`,
		`DELETE FROM pur_farmer_settlement WHERE doc_no LIKE 'DEMO-FS-TR-%'`,
		`DELETE FROM inv_stock_txn_line WHERE txn_id IN (SELECT id FROM inv_stock_txn WHERE doc_no LIKE 'DEMO-ST-WT-TR-%')`,
		`DELETE FROM inv_stock_txn WHERE doc_no LIKE 'DEMO-ST-WT-TR-%'`,
		`DELETE FROM inv_balance WHERE product_id IN (SELECT id FROM prd_product WHERE code='RM-DEMO-TRACE') AND batch_no LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_station_flow_log WHERE UPPER(trace_code) LIKE 'TR-DEMO-%'`,
		`DELETE FROM pd_routing_step WHERE routing_id IN (SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE')`,
		`DELETE FROM pd_routing WHERE code='RT-DEMO-TRACE'`,
		`DELETE FROM prd_product WHERE code IN ('RM-DEMO-TRACE','SF-DEMO-WASHED','SF-DEMO-PEELED','SF-DEMO-CUT','FG-DEMO-TRACE','SF-DEMO-WASH','SF-DEMO-PEEL','SF-DEMO-CORE','SF-DEMO-SLICE','FG-DEMO-DRY')`,
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

func ensureDemoProcess(db *sql.DB, code, name, processType string, piecework int) int64 {
	_, _ = db.Exec(`INSERT INTO pd_process(code, name, process_type, is_piecework, is_handover_point, status)
		SELECT ?, ?, ?, ?, 0, 'active'
		WHERE NOT EXISTS (SELECT 1 FROM pd_process WHERE code=?)`, code, name, processType, piecework, code)
	var id int64
	_ = db.QueryRow(`SELECT id FROM pd_process WHERE code=?`, code).Scan(&id)
	return id
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
	_, _ = db.Exec(`INSERT INTO prd_product(code, name, product_type, cost_price, sale_price, status)
		VALUES('SF-DEMO-WASH','清洗木薯','semi',1.5,NULL,'active'),
		('SF-DEMO-PEEL','去皮木薯','semi',2.0,NULL,'active'),
		('SF-DEMO-CUT','切段木薯','semi',2.3,NULL,'active'),
		('SF-DEMO-CORE','去芯木薯','semi',2.6,NULL,'active'),
		('SF-DEMO-SLICE','切片木薯','semi',2.9,NULL,'active'),
		('FG-DEMO-DRY','烘干木薯','finished',3.5,5.0,'active')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, status='active'`)
	prodID := demoID(db, `SELECT id FROM prd_product WHERE code='RM-DEMO-TRACE'`)
	finalProdID := demoID(db, `SELECT id FROM prd_product WHERE code='FG-DEMO-DRY'`)
	outWash := demoID(db, `SELECT id FROM prd_product WHERE code='SF-DEMO-WASH'`)
	outPeel := demoID(db, `SELECT id FROM prd_product WHERE code='SF-DEMO-PEEL'`)
	outCut := demoID(db, `SELECT id FROM prd_product WHERE code='SF-DEMO-CUT'`)
	outCore := demoID(db, `SELECT id FROM prd_product WHERE code='SF-DEMO-CORE'`)
	outSlice := demoID(db, `SELECT id FROM prd_product WHERE code='SF-DEMO-SLICE'`)
	if prodID <= 0 || finalProdID <= 0 {
		return
	}
	procSlice := ensureDemoProcess(db, "SLICE", "切片", "slice", 1)
	procDry := ensureDemoProcess(db, "DRY", "烘干", "dry", 0)
	if procSlice <= 0 {
		procSlice = demoID(db, `SELECT id FROM pd_process WHERE code='SLICE'`)
	}
	if procDry <= 0 {
		procDry = demoID(db, `SELECT id FROM pd_process WHERE code='DRY'`)
	}
	var routingID int64
	_ = db.QueryRow(`SELECT id FROM pd_routing WHERE code='RT-DEMO-TRACE' AND COALESCE(is_deleted,0)=0`).Scan(&routingID)
	if routingID <= 0 {
		res, err := db.Exec(`INSERT INTO pd_routing(code, name, product_id, version_no, status)
			VALUES('RT-DEMO-TRACE','演示木薯加工短工艺',?,'V1','active')`, finalProdID)
		if err == nil {
			routingID, _ = res.LastInsertId()
		}
	} else {
		_, _ = db.Exec(`UPDATE pd_routing SET product_id=?, name='演示木薯加工短工艺' WHERE id=?`, finalProdID, routingID)
	}
	if routingID <= 0 {
		return
	}
	stepDefs := []struct {
		seq, proc, outProd int
		code, name         string
		piece, checkpoint  int
	}{
		{1, 8, int(prodID), "D-IN", "原料入库", 0, 1},
		{2, 7, int(outWash), "D-WASH", "清洗", 0, 0},
		{3, 1, int(outPeel), "D-PEEL", "去皮", 1, 0},
		{4, 3, int(outCut), "D-CUT", "切段", 0, 0},
		{5, 4, int(outCore), "D-CORE", "去芯", 1, 0},
		{6, int(procSlice), int(outSlice), "D-SLICE", "切片", 1, 0},
		{7, int(procDry), int(finalProdID), "D-DRY", "烘干", 0, 0},
	}
	for i := range stepDefs {
		if stepDefs[i].outProd <= 0 {
			stepDefs[i].outProd = int(prodID)
		}
		if stepDefs[i].proc <= 0 {
			return
		}
	}
	stepIDs := map[int]int64{}
	for _, sd := range stepDefs {
		var sid int64
		_ = db.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? AND seq_no=?`, routingID, sd.seq).Scan(&sid)
		if sid <= 0 {
			res, err := db.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, output_product_id)
				VALUES(?,?,?,?,?,?,?,1,0,0,1,?)`, routingID, sd.seq, sd.proc, sd.code, sd.name, sd.piece, sd.checkpoint, sd.outProd)
			if err == nil {
				sid, _ = res.LastInsertId()
			}
		} else {
			_, _ = db.Exec(`UPDATE pd_routing_step SET process_id=?, step_code=?, step_name=?, is_piecework=?, is_inbound_checkpoint=?, output_product_id=? WHERE id=?`,
				sd.proc, sd.code, sd.name, sd.piece, sd.checkpoint, sd.outProd, sid)
		}
		stepIDs[sd.seq] = sid
	}

	seedTraceCompleted7D(db, prodID, finalProdID, routingID, farmerID, workerID, stepIDs, procSlice, procDry, day7)
	seedTraceInProgressToday(db, prodID, outPeel, routingID, farmerID, workerID, stepIDs, today)
	seedTraceInStockToday(db, prodID, farmerID, today)
	seedTraceAwaitGateToday(db, prodID, farmerID, today)
	seedTimelineHistory(db, workerID)
}

func seedTraceCompleted7D(db *sql.DB, prodID, finalProdID, routingID, farmerID, workerID int64, stepIDs map[int]int64, procSlice, procDry int64, bizDate string) {
	const trace = "TR-DEMO-7D001"
	startTS := demoTS(7, 8, 0)
	endTS := demoTS(7, 17, 30)
	gateTS := demoTS(7, 7, 30)
	stockTS := demoTS(7, 8, 15)

	arrID := seedDemoInboundArrival(db, "DEMO-ARR-TR-7D", farmerID, bizDate, 1250, "stocked", "7天前入厂到货")
	wtID := seedDemoWeighTicket(db, demoWeighSeed{
		docNo: "DEMO-WT-TR-7D", trace: trace, farmerID: farmerID, prodID: prodID, arrivalID: arrID,
		bizDate: bizDate, gross: 1250, deduct: 250, net: 1000, unitPrice: 1.2, whID: 1,
		status: "stocked", remark: "7天前已完成：入厂→入库→木薯加工全工序", confirmedAt: gateTS,
	})
	seedDemoFarmerSettlement(db, "DEMO-FS-TR-7D", farmerID, wtID, bizDate, 1000, 1.2, "paid", "7天前已付款")
	seedDemoStockTxnIn(db, "DEMO-ST-WT-TR-7D", prodID, 1, trace, 850, bizDate, stockTS, "过磅分板入库")
	seedDemoInvBalance(db, 1, prodID, trace, 850)

	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, payload_canonical, signature, status)
		VALUES(?,?,?,?,'A',?,1000,'{"demo":true}','demo','closed')`, trace, bizDate, trace, farmerID, nullIf0(wtID))

	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id, current_step_id)
		VALUES('BX-TR-DEMO-7D-A',?,1,?,50,50,?,?,'open',?,?)`,
		finalProdID, trace, farmerID, trace, procDry, nullIf0(stepIDs[7]))
	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id, current_step_id)
		VALUES('BX-TR-DEMO-7D-B',?,3,?,750,750,?,?,'in_stock',?,?)`,
		finalProdID, trace, farmerID, trace, procDry, nullIf0(stepIDs[7]))
	boardA := demoID(db, `SELECT id FROM inv_box_code WHERE code='BX-TR-DEMO-7D-A'`)

	res, _ := db.Exec(`INSERT INTO pd_trace_production(trace_code, status, started_by, completed_by, started_at, completed_at, input_kg, output_kg, loss_rate, routing_id, product_id, remark)
		VALUES(?,'done',1,1,?,?,1000,750,0.25,?,?,'演示：清洗至烘干全部工序已结束并自动结案')`, trace, startTS, endTS, nullIf0(routingID), prodID)
	sessID, _ := res.LastInsertId()

	type procYield struct {
		pid                       int64
		inKg, outKg, lossKg, rate float64
		at                        string
	}
	yields := []procYield{
		{7, 1000, 980, 20, 0.02, demoTS(7, 9, 30)},
		{1, 980, 850, 130, 0.1327, demoTS(7, 10, 30)},
		{3, 850, 820, 30, 0.0353, demoTS(7, 12, 0)},
		{4, 820, 780, 40, 0.0488, demoTS(7, 13, 30)},
		{procSlice, 780, 760, 20, 0.0256, demoTS(7, 15, 0)},
		{procDry, 760, 750, 10, 0.0132, demoTS(7, 16, 30)},
	}
	for _, y := range yields {
		if y.pid <= 0 {
			continue
		}
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
			boardA, trace, 1, nullIf0(stepIDs[3]), workerID)
		_, _ = db.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, from_step_id, to_process_id, to_step_id, kg, move_kind, created_by, created_at)
			VALUES(?,'BX-TR-DEMO-7D-A',?,7,?,?,1,?,980,'next',1,?)`,
			boardA, trace, nullIf0(stepIDs[2]), nullIf0(stepIDs[3]), nullIf0(stepIDs[3]), demoTS(7, 10, 0))
		_, _ = db.Exec(`INSERT INTO pd_process_move(board_id, board_code, trace_code, from_process_id, from_step_id, kg, move_kind, created_by, created_at)
			VALUES(?,'BX-TR-DEMO-7D-A',?,?,?,750,'stock_in',1,?)`,
			boardA, trace, procDry, nullIf0(stepIDs[7]), demoTS(7, 17, 0))
	}
	_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, remark, created_at)
		VALUES(?,?,'session_start',1,'进入生产',?)`, sessID, trace, startTS)
	_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, input_kg, output_kg, loss_rate, remark, created_at)
		VALUES(?,?,'session_complete',1,1000,750,0.25,'自动结案',?)`, sessID, trace, endTS)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, board_code, trace_code, process_id, worker_id, kg, remark, created_at)
		VALUES('issue', ?, 'BX-TR-DEMO-7D-A', ?, 1, ?, 800, '演示领料', ?)`, bizDate, trace, workerID, demoTS(7, 11, 0))
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('gate_accept', ?, ?, 1000, '入厂确认', ?)`, bizDate, trace, gateTS)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('stock_in', ?, ?, 850, '分板入库', ?)`, bizDate, trace, stockTS)
}

func seedTraceInProgressToday(db *sql.DB, prodID, boardProdID, routingID, farmerID, workerID int64, stepIDs map[int]int64, bizDate string) {
	const trace = "TR-DEMO-TODAY001"
	startTS := demoTS(0, 8, 30)
	gateTS := demoTS(0, 8, 0)
	stockTS := demoTS(0, 8, 20)

	arrID := seedDemoInboundArrival(db, "DEMO-ARR-TR-T1", farmerID, bizDate, 600, "stocked", "今日入厂到货")
	wtID := seedDemoWeighTicket(db, demoWeighSeed{
		docNo: "DEMO-WT-TR-T1", trace: trace, farmerID: farmerID, prodID: prodID, arrivalID: arrID,
		bizDate: bizDate, gross: 600, deduct: 100, net: 500, unitPrice: 1.2, whID: 1,
		status: "stocked", remark: "今日生产中：已入库待结算", confirmedAt: gateTS,
	})
	seedDemoFarmerSettlement(db, "DEMO-FS-TR-T1", farmerID, wtID, bizDate, 500, 1.2, "settle_pending", "今日待财务付款")
	seedDemoStockTxnIn(db, "DEMO-ST-WT-TR-T1", prodID, 1, trace, 500, bizDate, stockTS, "过磅分板入库")
	seedDemoInvBalance(db, 1, prodID, trace, 500)

	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, status)
		VALUES(?,?,?,?,'A',?,500,'open')`, trace, bizDate, trace, farmerID, nullIf0(wtID))

	if boardProdID <= 0 {
		boardProdID = prodID
	}
	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id, current_step_id)
		VALUES('BX-TR-DEMO-T1-A',?,1,?,180,180,?,?,'open',3,?)`,
		boardProdID, trace, farmerID, trace, nullIf0(stepIDs[4]))
	boardA := demoID(db, `SELECT id FROM inv_box_code WHERE code='BX-TR-DEMO-T1-A'`)

	res, _ := db.Exec(`INSERT INTO pd_trace_production(trace_code, status, started_by, started_at, routing_id, product_id, remark)
		VALUES(?,'in_progress',1,?,?,?,'演示：清洗、去皮已完成，切段工序进行中')`, trace, startTS, nullIf0(routingID), prodID)
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
			boardA, trace, 3, nullIf0(stepIDs[4]), workerID)
	}
	_, _ = db.Exec(`INSERT INTO pd_trace_process_log(session_id, trace_code, event_type, actor_user_id, remark, created_at)
		VALUES(?,?,'session_start',1,'今日进入生产',?)`, sessID, trace, startTS)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('gate_accept', ?, ?, 500, '入厂确认', ?)`, bizDate, trace, gateTS)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('stock_in', ?, ?, 500, '分板入库', ?)`, bizDate, trace, stockTS)
}

func seedTraceInStockToday(db *sql.DB, prodID, farmerID int64, bizDate string) {
	const trace = "TR-DEMO-TODAY002"
	gateTS := demoTS(0, 9, 0)
	stockTS := demoTS(0, 9, 30)

	arrID := seedDemoInboundArrival(db, "DEMO-ARR-TR-T2", farmerID, bizDate, 360, "stocked", "今日库中待投产")
	wtID := seedDemoWeighTicket(db, demoWeighSeed{
		docNo: "DEMO-WT-TR-T2", trace: trace, farmerID: farmerID, prodID: prodID, arrivalID: arrID,
		bizDate: bizDate, gross: 360, deduct: 60, net: 300, unitPrice: 1.2, whID: 1,
		status: "stocked", remark: "今日库中溯源（未进入生产）", confirmedAt: gateTS,
	})
	seedDemoFarmerSettlement(db, "DEMO-FS-TR-T2", farmerID, wtID, bizDate, 300, 1.2, "settle_pending", "入库完成待付款")
	seedDemoStockTxnIn(db, "DEMO-ST-WT-TR-T2", prodID, 1, trace, 300, bizDate, stockTS, "过磅分板入库")
	seedDemoInvBalance(db, 1, prodID, trace, 300)

	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, status)
		VALUES(?,?,?,?,'B',?,300,'open')`, trace, bizDate, trace, farmerID, nullIf0(wtID))
	_, _ = db.Exec(`INSERT INTO inv_box_code(code, product_id, warehouse_id, batch_no, qty, weight, farmer_id, trace_code, status, current_process_id)
		VALUES('BX-TR-DEMO-T2-A',?,1,?,300,300,?,?,'open',8)`,
		prodID, trace, farmerID, trace)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('gate_accept', ?, ?, 300, '入厂确认', ?)`, bizDate, trace, gateTS)
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('stock_in', ?, ?, 300, '分板入库待投产', ?)`, bizDate, trace, stockTS)
}

func seedTraceAwaitGateToday(db *sql.DB, prodID, farmerID int64, bizDate string) {
	const trace = "TR-DEMO-TODAY003"
	arrID := seedDemoInboundArrival(db, "DEMO-ARR-TR-T3", farmerID, bizDate, 420, "confirmed", "今日待到厂确认")
	wtID := seedDemoWeighTicket(db, demoWeighSeed{
		docNo: "DEMO-WT-TR-T3", trace: trace, farmerID: farmerID, prodID: prodID, arrivalID: arrID,
		bizDate: bizDate, gross: 420, deduct: 70, net: 350, unitPrice: 1.2, whID: 0,
		status: "weighed", remark: "今日过磅完成，待入厂确认", confirmedAt: "",
	})
	_, _ = db.Exec(`INSERT INTO pur_trace_lot(trace_code, biz_date, batch_no, farmer_id, grade, weigh_ticket_id, net_weight, status)
		VALUES(?,?,?,?,'A',?,350,'open')`, trace, bizDate, trace, farmerID, nullIf0(wtID))
	_, _ = db.Exec(`INSERT INTO pd_station_flow_log(event_type, biz_date, trace_code, kg, remark, created_at)
		VALUES('weigh', ?, ?, 350, '过磅待入厂', ?)`, bizDate, trace, demoTS(0, 7, 45))
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

type demoWeighSeed struct {
	docNo, trace, bizDate, remark, confirmedAt, status string
	farmerID, prodID, arrivalID, whID                     int64
	gross, deduct, net, unitPrice                          float64
}

func seedDemoInboundArrival(db *sql.DB, docNo string, farmerID int64, bizDate string, estimate float64, status, remark string) int64 {
	_, _ = db.Exec(`INSERT INTO pur_inbound_arrival(doc_no, farmer_id, origin, variety, estimate_weight, status, biz_date, remark)
		VALUES(?, ?, '广西武鸣', '鲜木薯', ?, ?, ?, ?)`,
		docNo, farmerID, estimate, status, bizDate, remark)
	return demoID(db, `SELECT id FROM pur_inbound_arrival WHERE doc_no=?`, docNo)
}

func seedDemoWeighTicket(db *sql.DB, p demoWeighSeed) int64 {
	_, _ = db.Exec(`INSERT INTO pur_weigh_ticket(doc_no, farmer_id, product_id, gross_weight, deduct_weight, net_weight, qc_result, status, biz_date, remark, receive_kind, batch_no, trace_code, arrival_id, warehouse_id, unit_price, confirmed_at)
		VALUES(?, ?, ?, ?, ?, ?, 'pass', ?, ?, ?, 'gate', ?, ?, ?, ?, ?, NULLIF(?,''))`,
		p.docNo, p.farmerID, p.prodID, p.gross, p.deduct, p.net, p.status, p.bizDate, p.remark,
		p.trace, p.trace, nullIf0(p.arrivalID), nullIf0(p.whID), p.unitPrice, p.confirmedAt)
	return demoID(db, `SELECT id FROM pur_weigh_ticket WHERE doc_no=?`, p.docNo)
}

func seedDemoStockTxnIn(db *sql.DB, docNo string, prodID, whID int64, batchNo string, qty float64, bizDate, postedAt, remark string) {
	res, err := db.Exec(`INSERT INTO inv_stock_txn(doc_no, doc_type, biz_date, status, warehouse_id, remark, posted_at, created_by)
		VALUES(?, 'purchase_in', ?, 'posted', ?, ?, ?, 1)`, docNo, bizDate, whID, remark, postedAt)
	if err != nil {
		return
	}
	tid, _ := res.LastInsertId()
	amt := qty * 1.2
	_, _ = db.Exec(`INSERT INTO inv_stock_txn_line(txn_id, line_no, product_id, qty, base_qty, weight, batch_no, direction, amount)
		VALUES(?, 1, ?, ?, ?, ?, ?, 'in', ?)`, tid, prodID, qty, qty, qty, batchNo, amt)
}

func seedDemoInvBalance(db *sql.DB, whID, prodID int64, batchNo string, qty float64) {
	var id int64
	err := db.QueryRow(`SELECT id FROM inv_balance WHERE warehouse_id=? AND product_id=? AND COALESCE(batch_no,'')=?`,
		whID, prodID, batchNo).Scan(&id)
	if err == sql.ErrNoRows {
		_, _ = db.Exec(`INSERT INTO inv_balance(warehouse_id, location_id, product_id, batch_no, box_code_id, qty) VALUES(?,?,?,?,0,?)`,
			whID, 0, prodID, batchNo, qty)
		return
	}
	_, _ = db.Exec(`UPDATE inv_balance SET qty=? WHERE id=?`, qty, id)
}

func seedDemoFarmerSettlement(db *sql.DB, docNo string, farmerID, wtID int64, bizDate string, net, unitPrice float64, status, remark string) {
	amount := net * unitPrice
	_, _ = db.Exec(`INSERT INTO pur_farmer_settlement(doc_no, farmer_id, weigh_ticket_id, biz_date, net_weight, unit_price, amount, status, remark, goods_amount)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		docNo, farmerID, wtID, bizDate, net, unitPrice, amount, status, remark, amount)
}
