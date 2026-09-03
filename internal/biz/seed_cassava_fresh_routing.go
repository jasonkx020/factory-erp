package biz

import (
	"database/sql"
	"log"
)

const freshCassavaRoutingCode = "RT-CASSAVA-FRESH"

// EnsureFreshCassavaRouting seeds the canonical fresh-cassava shop-floor routing:
// 清洗 → 去皮 → 切段 → 去芯 → 切片 → 烘干（含逐步产出物料）。
// Idempotent: safe on every API start when demo seed is enabled.
func EnsureFreshCassavaRouting(db *sql.DB) {
	if db == nil {
		return
	}
	ensureProduct := func(code, name, ptype string, cost float64, sale *float64) int64 {
		var id int64
		_ = db.QueryRow(`SELECT id FROM prd_product WHERE code=?`, code).Scan(&id)
		if id > 0 {
			_, _ = db.Exec(`UPDATE prd_product SET name=?, product_type=?, cost_price=?, status='active' WHERE id=?`,
				name, ptype, cost, id)
			if sale != nil {
				_, _ = db.Exec(`UPDATE prd_product SET sale_price=? WHERE id=?`, *sale, id)
			}
			return id
		}
		if sale != nil {
			res, err := db.Exec(`INSERT INTO prd_product(code, name, product_type, cost_price, sale_price, status)
				VALUES(?,?,?,?,?,'active')`, code, name, ptype, cost, *sale)
			if err == nil {
				id, _ = res.LastInsertId()
			}
		} else {
			res, err := db.Exec(`INSERT INTO prd_product(code, name, product_type, cost_price, status)
				VALUES(?,?,?,?,'active')`, code, name, ptype, cost)
			if err == nil {
				id, _ = res.LastInsertId()
			}
		}
		if id <= 0 {
			_ = db.QueryRow(`SELECT id FROM prd_product WHERE code=?`, code).Scan(&id)
		}
		return id
	}

	saleDry := 5.5
	rawID := ensureProduct("RM-CASSAVA", "鲜木薯", "raw", 1.2, nil)
	if rawID <= 0 {
		// Fallback if code missing but classic id=1 exists.
		_ = db.QueryRow(`SELECT id FROM prd_product WHERE id=1`).Scan(&rawID)
	}
	washedID := ensureProduct("SF-CASSAVA-WASH", "清洗木薯", "semi", 1.5, nil)
	peeledID := ensureProduct("SF-CASSAVA-PEEL", "去皮木薯", "semi", 2.0, nil)
	cutID := ensureProduct("SF-CASSAVA-CUT", "切段木薯", "semi", 2.3, nil)
	coredID := ensureProduct("SF-CASSAVA-CORE", "去芯木薯", "semi", 2.6, nil)
	slicedID := ensureProduct("SF-CASSAVA-SLICE", "切片木薯", "semi", 2.9, nil)
	driedID := ensureProduct("FG-CASSAVA-DRY", "烘干木薯丁", "finished", 3.8, &saleDry)
	if rawID <= 0 || washedID <= 0 || peeledID <= 0 || cutID <= 0 || coredID <= 0 || slicedID <= 0 || driedID <= 0 {
		log.Printf("fresh cassava routing seed skipped: missing products")
		return
	}

	ensureProc := func(code, name, processType string, piecework int) int64 {
		_, _ = db.Exec(`INSERT INTO pd_process(code, name, process_type, is_piecework, is_handover_point, status)
			SELECT ?, ?, ?, ?, 0, 'active'
			WHERE NOT EXISTS (SELECT 1 FROM pd_process WHERE code=?)`, code, name, processType, piecework, code)
		var id int64
		_ = db.QueryRow(`SELECT id FROM pd_process WHERE code=?`, code).Scan(&id)
		if id > 0 {
			_, _ = db.Exec(`UPDATE pd_process SET name=?, process_type=?, is_piecework=?, status='active' WHERE id=?`,
				name, processType, piecework, id)
		}
		return id
	}
	washID := ensureProc("WASH", "清洗", "wash", 0)
	peelID := ensureProc("PEEL", "去皮", "peel", 1)
	cutPID := ensureProc("CUT", "切断", "cut", 0)
	coreID := ensureProc("CORE", "去芯", "core", 1)
	sliceID := ensureProc("SLICE", "切片", "slice", 1)
	dryID := ensureProc("DRY", "烘干", "dry", 0)
	if washID <= 0 || peelID <= 0 || cutPID <= 0 || coreID <= 0 || sliceID <= 0 || dryID <= 0 {
		log.Printf("fresh cassava routing seed skipped: missing processes")
		return
	}

	const routingName = "鲜木薯完整加工"
	var routingID int64
	_ = db.QueryRow(`SELECT id FROM pd_routing WHERE code=? AND COALESCE(is_deleted,0)=0`, freshCassavaRoutingCode).Scan(&routingID)
	if routingID <= 0 {
		res, err := db.Exec(`INSERT INTO pd_routing(code, name, product_id, version_no, status)
			VALUES(?,?,?,'V1','active')`, freshCassavaRoutingCode, routingName, rawID)
		if err != nil {
			log.Printf("fresh cassava routing insert: %v", err)
			return
		}
		routingID, _ = res.LastInsertId()
	} else {
		_, _ = db.Exec(`UPDATE pd_routing SET name=?, product_id=?, status='active', version_no=COALESCE(NULLIF(version_no,''),'V1') WHERE id=?`,
			routingName, rawID, routingID)
	}
	if routingID <= 0 {
		return
	}

	type stepDef struct {
		seq                 int
		processID           int64
		code, name          string
		piece, checkpoint   int
		outputProductID     int64
		autoStockIn, autoNext int
	}
	steps := []stepDef{
		{1, washID, "F1", "清洗", 0, 0, washedID, 0, 1},
		{2, peelID, "F2", "去皮", 1, 0, peeledID, 0, 1},
		{3, cutPID, "F3", "切段", 0, 0, cutID, 0, 1},
		{4, coreID, "F4", "去芯", 1, 0, coredID, 0, 1},
		{5, sliceID, "F5", "切片", 1, 0, slicedID, 0, 1},
		{6, dryID, "F6", "烘干", 0, 0, driedID, 1, 0},
	}

	ws := defaultWorkshopDeptIDDB(db)
	for _, st := range steps {
		var sid int64
		_ = db.QueryRow(`SELECT id FROM pd_routing_step WHERE routing_id=? AND seq_no=?`, routingID, st.seq).Scan(&sid)
		if sid <= 0 {
			_, err := db.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name,
				is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, warehouse_id, workshop_dept_id, output_product_id)
				VALUES(?,?,?,?,?,?,?,?,?,0,1,?,?)`,
				routingID, st.seq, st.processID, st.code, st.name, st.piece, st.checkpoint, st.autoNext, st.autoStockIn, nullIf0(ws), st.outputProductID)
			if err != nil {
				// warehouse_id / workshop may vary by schema; retry minimal columns
				_, _ = db.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name,
					is_piecework, is_inbound_checkpoint, auto_next, auto_stock_in, auto_stock_out, output_product_id)
					VALUES(?,?,?,?,?,?,?,?,?,0,?)`,
					routingID, st.seq, st.processID, st.code, st.name, st.piece, st.checkpoint, st.autoNext, st.autoStockIn, st.outputProductID)
			}
			continue
		}
		_, _ = db.Exec(`UPDATE pd_routing_step SET process_id=?, step_code=?, step_name=?, is_piecework=?,
			is_inbound_checkpoint=?, auto_next=?, auto_stock_in=?, output_product_id=? WHERE id=?`,
			st.processID, st.code, st.name, st.piece, st.checkpoint, st.autoNext, st.autoStockIn, st.outputProductID, sid)
	}
	// Drop stale extra steps if an older seed left more than 6.
	_, _ = db.Exec(`DELETE FROM pd_routing_step WHERE routing_id=? AND seq_no>?`, routingID, len(steps))

	// Prefer this routing as product-spec default for 鲜木薯.
	var specID int64
	_ = db.QueryRow(`SELECT id FROM prd_product_spec WHERE product_id=? AND spec_code='DEFAULT' AND COALESCE(is_deleted,0)=0`, rawID).Scan(&specID)
	if specID <= 0 {
		_, _ = db.Exec(`INSERT INTO prd_product_spec(product_id, spec_code, routing_id, remark, status)
			VALUES(?,'DEFAULT',?,'鲜木薯完整加工默认工艺','active')`, rawID, routingID)
	} else {
		_, _ = db.Exec(`UPDATE prd_product_spec SET routing_id=?, status='active', remark='鲜木薯完整加工默认工艺' WHERE id=?`, routingID, specID)
	}

	log.Printf("fresh cassava routing ensured: %s id=%d steps=%d", freshCassavaRoutingCode, routingID, len(steps))
}
