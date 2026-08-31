package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/security"
)

func (s *Services) loadRoleIAMMeta(roleID int64) (code string, isSystem bool, ok bool) {
	var isSys int
	err := s.DB.QueryRow(`SELECT code, is_system FROM iam_role WHERE id=? AND COALESCE(is_deleted,0)=0`, roleID).Scan(&code, &isSys)
	if err != nil {
		return "", false, false
	}
	return code, isSys == 1, true
}

func permissionCodeSet(perms []string) map[string]struct{} {
	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[p] = struct{}{}
	}
	return set
}

func (s *Services) resolveSubmittedPermissionCodes(items []interface{}) ([]string, bool) {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, x := range items {
		if id, ok := asInt64(x); ok {
			var code string
			if err := s.DB.QueryRow(`SELECT code FROM iam_permission WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&code); err != nil {
				return nil, false
			}
			if _, dup := seen[code]; !dup {
				seen[code] = struct{}{}
				out = append(out, code)
			}
			continue
		}
		if code, ok := x.(string); ok && strings.TrimSpace(code) != "" {
			var exists int
			if err := s.DB.QueryRow(`SELECT COUNT(1) FROM iam_permission WHERE code=? AND COALESCE(is_deleted,0)=0`, code).Scan(&exists); err != nil || exists == 0 {
				return nil, false
			}
			if _, dup := seen[code]; !dup {
				seen[code] = struct{}{}
				out = append(out, code)
			}
		}
	}
	return out, true
}

func rejectSystemRoleReadonly(c *gin.Context) bool {
	api.FailJSON(c, "SYSTEM_ROLE_READONLY")
	return true
}

func (s *Services) handleProducts(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(category,''), product_type, status,
			COALESCE(spec_text,''), COALESCE(barcode,''), cost_price, sale_price,
			COALESCE(is_batch_managed,1), COALESCE(is_box_managed,0)
			FROM prd_product WHERE is_deleted=0 ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, cat, typ, status, spec, barcode string
			var cost, sale sql.NullFloat64
			var batch, box int
			_ = rows.Scan(&id, &code, &name, &cat, &typ, &status, &spec, &barcode, &cost, &sale, &batch, &box)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "category": cat, "product_type": typ, "status": status,
				"spec": spec, "spec_text": spec, "barcode": barcode,
				"cost_price": cost.Float64, "sale_price": sale.Float64,
				"is_batch_managed": batch == 1, "is_box_managed": box == 1,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		code, _ := body["code"].(string)
		name, _ := body["name"].(string)
		if code == "" || name == "" {
			api.FailJSON(c, "INVALID_REQUEST")
			return true
		}
		typ, _ := body["product_type"].(string)
		if typ == "" {
			typ = "finished"
		}
		spec, _ := body["spec"].(string)
		if spec == "" {
			spec, _ = body["spec_text"].(string)
		}
		cost, _ := asFloat(body["cost_price"])
		sale, _ := asFloat(body["sale_price"])
		res, err := s.DB.Exec(`INSERT INTO prd_product(code, name, category, product_type, status, spec_text, barcode, cost_price, sale_price, is_batch_managed, is_box_managed, is_deleted)
			VALUES(?,?,?,?,'active',?,?,?,?,?,?,0)`,
			code, name, strOr(body["category"]), typ, spec, strOr(body["barcode"]),
			cost, sale,
			boolIntDef(body["is_batch_managed"], 1), boolIntDef(body["is_box_managed"], 0))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		// default base unit kg
		unitName := strOrDef(body["base_unit"], "kg")
		_, _ = s.DB.Exec(`INSERT INTO prd_product_unit(product_id, unit_name, is_base, factor_to_base, is_purchase, is_sale, is_stock)
			VALUES(?,?,1,1,1,1,1)`, id, unitName)
		// app sort entry
		_, _ = s.DB.Exec(`INSERT INTO prd_product_app_sort(product_id, channel, sort_no, is_visible) VALUES(?,?,?,1)`,
			id, "app", id*10)
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "product_type": typ, "status": "active"})
		return true
	case "get":
		id := paramID(c)
		rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(category,''), product_type, status,
			COALESCE(spec_text,''), COALESCE(barcode,''), cost_price, sale_price,
			COALESCE(is_batch_managed,1), COALESCE(is_box_managed,0)
			FROM prd_product WHERE id=? AND is_deleted=0`, id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list, _ := rowsToMaps(rows)
		if len(list) == 0 {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		m := gin.H(list[0])
		m["spec"] = m["spec_text"]
		urows, _ := s.DB.Query(`SELECT id, unit_name, is_base, factor_to_base, is_purchase, is_sale, is_stock FROM prd_product_unit WHERE product_id=?`, id)
		units := []map[string]interface{}{}
		if urows != nil {
			defer urows.Close()
			units, _ = rowsToMaps(urows)
		}
		m["units"] = units
		api.OK(c, m)
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE prd_product SET
			name=COALESCE(NULLIF(?,''),name),
			category=COALESCE(NULLIF(?,''),category),
			product_type=COALESCE(NULLIF(?,''),product_type),
			spec_text=COALESCE(NULLIF(?,''),spec_text),
			barcode=COALESCE(NULLIF(?,''),barcode),
			cost_price=COALESCE(?,cost_price),
			sale_price=COALESCE(?,sale_price),
			is_batch_managed=COALESCE(?,is_batch_managed),
			is_box_managed=COALESCE(?,is_box_managed),
			status=COALESCE(NULLIF(?,''),status),
			updated_at=NOW()
			WHERE id=? AND is_deleted=0`,
			strOr(body["name"]), strOr(body["category"]), strOr(body["product_type"]),
			firstNonEmpty(strOr(body["spec"]), strOr(body["spec_text"])),
			strOr(body["barcode"]),
			nullFloat(body["cost_price"]), nullFloat(body["sale_price"]),
			nullableBoolInt(body["is_batch_managed"]), nullableBoolInt(body["is_box_managed"]),
			strOr(body["status"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE prd_product SET is_deleted=1, updated_at=NOW() WHERE id=?`, id)
		api.OK(c, gin.H{})
		return true
	case "action:activate":
		_, _ = s.DB.Exec(`UPDATE prd_product SET status='active', updated_at=NOW() WHERE id=?`, paramID(c))
		api.OK(c, gin.H{"id": paramID(c), "status": "active"})
		return true
	case "action:deactivate":
		_, _ = s.DB.Exec(`UPDATE prd_product SET status='inactive', updated_at=NOW() WHERE id=?`, paramID(c))
		api.OK(c, gin.H{"id": paramID(c), "status": "inactive"})
		return true
	}
	return false
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func nullableBoolInt(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	return boolInt(v)
}

func (s *Services) handleProcesses(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, code, name, process_type, COALESCE(NULLIF(pay_mode,''),'none'), is_piecework, is_handover_point,
			COALESCE(NULLIF(status,''),'active') FROM pd_process WHERE COALESCE(is_deleted,0)=0 ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, typ, payMode, status string
			var piece, hand int
			_ = rows.Scan(&id, &code, &name, &typ, &payMode, &piece, &hand, &status)
			payMode = normalizePayMode(payMode, piece == 1)
			if status != "inactive" {
				status = "active"
			}
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "process_type": typ, "pay_mode": payMode,
				"is_piecework": payModeToIsPiecework(payMode) == 1, "is_handover_point": hand == 1, "status": status,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		code, _ := body["code"].(string)
		name, _ := body["name"].(string)
		typ, _ := body["process_type"].(string)
		if typ == "" {
			typ = "other"
		}
		pieceBool := asBool(body["is_piecework"])
		payMode := normalizePayMode(strOr(body["pay_mode"]), pieceBool)
		piece := payModeToIsPiecework(payMode)
		status := strings.TrimSpace(strOrDef(body["status"], "active"))
		if status != "inactive" {
			status = "active"
		}
		res, err := s.DB.Exec(`INSERT INTO pd_process(code, name, process_type, pay_mode, is_piecework, is_handover_point, status) VALUES(?,?,?,?,?,0,?)`,
			code, name, typ, payMode, piece, status)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		id, _ := res.LastInsertId()
		after := gin.H{"id": id, "code": code, "name": name, "process_type": typ, "pay_mode": payMode, "is_piecework": piece == 1, "status": status}
		s.writeAuditCtx(c, "pd_process", id, "create", "", nil, after)
		api.OK(c, after)
		return true
	case "get", "update", "delete":
		id := paramID(c)
		if action == "get" {
			var code, name, typ, payMode, status string
			var piece, hand int
			err := s.DB.QueryRow(`SELECT code, name, process_type, COALESCE(NULLIF(pay_mode,''),'none'), is_piecework, is_handover_point,
				COALESCE(NULLIF(status,''),'active') FROM pd_process WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
				Scan(&code, &name, &typ, &payMode, &piece, &hand, &status)
			if err != nil {
				api.FailJSON(c, "NOT_FOUND")
				return true
			}
			payMode = normalizePayMode(payMode, piece == 1)
			if status != "inactive" {
				status = "active"
			}
			api.OK(c, gin.H{
				"id": id, "code": code, "name": name, "process_type": typ, "pay_mode": payMode,
				"is_piecework": payModeToIsPiecework(payMode) == 1, "is_handover_point": hand == 1, "status": status,
			})
			return true
		}
		if action == "update" {
			body := bindBody(c)
			var beforeCode, beforeName, beforeTyp, beforeMode, beforeStatus string
			var beforePiece, beforeHand int
			_ = s.DB.QueryRow(`SELECT code, name, process_type, COALESCE(NULLIF(pay_mode,''),'none'), is_piecework, is_handover_point,
				COALESCE(NULLIF(status,''),'active') FROM pd_process WHERE id=?`, id).
				Scan(&beforeCode, &beforeName, &beforeTyp, &beforeMode, &beforePiece, &beforeHand, &beforeStatus)
			before := gin.H{
				"id": id, "code": beforeCode, "name": beforeName, "process_type": beforeTyp,
				"pay_mode": normalizePayMode(beforeMode, beforePiece == 1), "is_piecework": beforePiece == 1,
				"is_handover_point": beforeHand == 1, "status": beforeStatus,
			}
			name := strOr(body["name"])
			typ := strOr(body["process_type"])
			sets := []string{}
			args := []interface{}{}
			if name != "" {
				sets = append(sets, "name=?")
				args = append(args, name)
			}
			if typ != "" {
				sets = append(sets, "process_type=?")
				args = append(args, typ)
			}
			if _, ok := body["pay_mode"]; ok {
				mode := normalizePayMode(strOr(body["pay_mode"]), asBool(body["is_piecework"]))
				sets = append(sets, "pay_mode=?", "is_piecework=?")
				args = append(args, mode, payModeToIsPiecework(mode))
			} else if _, ok := body["is_piecework"]; ok {
				mode := normalizePayMode("", asBool(body["is_piecework"]))
				sets = append(sets, "pay_mode=?", "is_piecework=?")
				args = append(args, mode, payModeToIsPiecework(mode))
			}
			if _, ok := body["is_handover_point"]; ok {
				sets = append(sets, "is_handover_point=?")
				args = append(args, boolToInt(asBool(body["is_handover_point"])))
			}
			if _, ok := body["status"]; ok {
				st := strings.TrimSpace(strOr(body["status"]))
				if st != "inactive" {
					st = "active"
				}
				sets = append(sets, "status=?")
				args = append(args, st)
			}
			if code := strings.TrimSpace(strOr(body["code"])); code != "" {
				sets = append(sets, "code=?")
				args = append(args, code)
			}
			if len(sets) > 0 {
				sets = append(sets, "updated_at=NOW()")
				args = append(args, id)
				_, _ = s.DB.Exec(`UPDATE pd_process SET `+strings.Join(sets, ",")+` WHERE id=?`, args...)
			}
			var afterCode, afterName, afterTyp, afterMode, afterStatus string
			var afterPiece, afterHand int
			_ = s.DB.QueryRow(`SELECT code, name, process_type, COALESCE(NULLIF(pay_mode,''),'none'), is_piecework, is_handover_point,
				COALESCE(NULLIF(status,''),'active') FROM pd_process WHERE id=?`, id).
				Scan(&afterCode, &afterName, &afterTyp, &afterMode, &afterPiece, &afterHand, &afterStatus)
			after := gin.H{
				"id": id, "code": afterCode, "name": afterName, "process_type": afterTyp,
				"pay_mode": normalizePayMode(afterMode, afterPiece == 1), "is_piecework": afterPiece == 1,
				"is_handover_point": afterHand == 1, "status": afterStatus,
			}
			s.writeAuditCtx(c, "pd_process", id, "update", "", before, after)
			api.OK(c, after)
			return true
		}
		api.OK(c, gin.H{})
		return true
	}
	return false
}

func (s *Services) handleProdTasks(c *gin.Context, method, action, path string) bool {
	if strings.Contains(path, "/items") {
		id := paramID(c)
		if method == "GET" {
			rows, err := s.DB.Query(`SELECT i.id, i.product_id, i.plan_qty, COALESCE(i.completed_qty,0),
				COALESCE(p.name,''), COALESCE(p.code,'')
				FROM pd_production_task_item i
				LEFT JOIN prd_product p ON p.id=i.product_id
				WHERE i.task_id=? ORDER BY i.id`, id)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var iid, pid int64
				var qty, done float64
				var pname, pcode string
				_ = rows.Scan(&iid, &pid, &qty, &done, &pname, &pcode)
				pct := 0.0
				if qty > 0 {
					pct = done / qty * 100
				}
				list = append(list, gin.H{
					"id": iid, "product_id": pid, "product_name": pname, "product_code": pcode,
					"qty": qty, "plan_qty": qty, "completed_qty": done, "progress_pct": pct,
				})
			}
			api.OK(c, gin.H{"list": list})
			return true
		}
		body := bindBody(c)
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if qty == 0 {
			qty, _ = asFloat(body["plan_qty"])
		}
		_, err := s.DB.Exec(`INSERT INTO pd_production_task_item(task_id, product_id, plan_qty) VALUES(?,?,?)`, id, pid, qty)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		api.OK(c, gin.H{"ok": true})
		return true
	}
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT t.id, t.doc_no, t.status, t.created_at,
			COALESCE(t.routing_id,0), COALESCE(t.workshop_dept_id,0),
			COALESCE((SELECT SUM(plan_qty) FROM pd_production_task_item i WHERE i.task_id=t.id),0),
			COALESCE((SELECT SUM(completed_qty) FROM pd_production_task_item i WHERE i.task_id=t.id),0)
			FROM pd_production_task t WHERE t.is_deleted=0 ORDER BY t.id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, rid, wid int64
			var docNo, status, created string
			var plan, done float64
			_ = rows.Scan(&id, &docNo, &status, &created, &rid, &wid, &plan, &done)
			pct := 0.0
			if plan > 0 {
				pct = done / plan * 100
			}
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "status": status, "created_at": created,
				"routing_id": rid, "workshop_dept_id": wid,
				"plan_qty": plan, "completed_qty": done, "progress_pct": pct,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		docNo, _ := body["doc_no"].(string)
		if docNo == "" {
			docNo = fmt.Sprintf("PT%s", time.Now().Format("060102150405"))
		}
		wid := s.resolveWorkshopDeptID(body, true)
		res, err := s.DB.Exec(`INSERT INTO pd_production_task(doc_no, source_type, status, routing_id, workshop_dept_id, remark, created_by)
			VALUES(?,?,?,?,?,?,?)`,
			docNo, strOrDef(body["source_type"], "manual"), "pending",
			nullInt(body["routing_id"]), nullIf0(wid), strOr(body["remark"]), claimsUserID(c))
		if err != nil {
			res, err = s.DB.Exec(`INSERT INTO pd_production_task(doc_no, status) VALUES(?,'pending')`, docNo)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
		}
		id, _ := res.LastInsertId()
		pid, _ := asInt64(body["product_id"])
		qty, _ := asFloat(body["qty"])
		if qty == 0 {
			qty, _ = asFloat(body["plan_qty"])
		}
		if pid > 0 {
			if qty <= 0 {
				qty = 1
			}
			_, _ = s.DB.Exec(`INSERT INTO pd_production_task_item(task_id, product_id, plan_qty) VALUES(?,?,?)`, id, pid, qty)
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "pending"})
		return true
	case "get":
		id := paramID(c)
		var docNo, status, created string
		var rid, wid int64
		var plan, done float64
		err := s.DB.QueryRow(`SELECT t.doc_no, t.status, t.created_at,
			COALESCE(t.routing_id,0), COALESCE(t.workshop_dept_id,0),
			COALESCE((SELECT SUM(plan_qty) FROM pd_production_task_item i WHERE i.task_id=t.id),0),
			COALESCE((SELECT SUM(completed_qty) FROM pd_production_task_item i WHERE i.task_id=t.id),0)
			FROM pd_production_task t WHERE t.id=? AND t.is_deleted=0`, id).
			Scan(&docNo, &status, &created, &rid, &wid, &plan, &done)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		pct := 0.0
		if plan > 0 {
			pct = done / plan * 100
		}
		api.OK(c, gin.H{
			"id": id, "doc_no": docNo, "status": status, "created_at": created,
			"routing_id": rid, "workshop_dept_id": wid,
			"plan_qty": plan, "completed_qty": done, "progress_pct": pct,
		})
		return true
	case "update":
		id := paramID(c)
		body := bindBody(c)
		status, _ := body["status"].(string)
		if status != "" {
			_, _ = s.DB.Exec(`UPDATE pd_production_task SET status=? WHERE id=?`, status, id)
		}
		api.OK(c, gin.H{"id": id})
		return true
	case "action:close", "action:release", "action:submit":
		id := paramID(c)
		st := "closed"
		if strings.Contains(action, "release") {
			st = "released"
		}
		if strings.Contains(action, "submit") {
			st = "in_progress"
		}
		_, _ = s.DB.Exec(`UPDATE pd_production_task SET status=? WHERE id=?`, st, id)
		api.OK(c, gin.H{"id": id, "status": st})
		return true
	}
	return false
}

func (s *Services) handleDispatches(c *gin.Context, method, action, path string) bool {
	_ = method
	_ = path
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_dispatch`)
	case "create":
		body := bindBody(c)
		taskID, _ := asInt64(body["task_id"])
		procID, _ := asInt64(body["process_id"])
		workerID, _ := asInt64(body["worker_id"])
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			qty = 1
		}
		docNo := fmt.Sprintf("DP%d", time.Now().UnixNano()%1e12)
		woNo := fmt.Sprintf("WO%d", time.Now().UnixNano()%1e12)
		woRes, err := s.DB.Exec(`INSERT INTO pd_work_order(doc_no, task_id, process_id, status, plan_qty) VALUES(?,?,?,'pending',?)`, woNo, taskID, procID, qty)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		woID, _ := woRes.LastInsertId()
		res, err := s.DB.Exec(`INSERT INTO pd_dispatch(doc_no, work_order_id, worker_id, plan_qty, status) VALUES(?,?,?,?,'dispatched')`, docNo, woID, workerID, qty)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "work_order_id": woID, "worker_id": workerID, "plan_qty": qty, "status": "dispatched"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_dispatch WHERE id=?`, paramID(c))
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_dispatch SET worker_id=COALESCE(NULLIF(?,0),worker_id), plan_qty=COALESCE(NULLIF(?,0),plan_qty),
			status=COALESCE(NULLIF(?,''),status) WHERE id=?`,
			nullInt64Or(body["worker_id"]), nullFloat(body["qty"]), strOr(body["status"]), id)
		return s.getSimpleRow(c, `SELECT * FROM pd_dispatch WHERE id=?`, id)
	case "action:confirm", "action:cancel", "action:receive":
		id := paramID(c)
		st := "confirmed"
		name := strings.TrimPrefix(action, "action:")
		if name == "cancel" {
			st = "cancelled"
		}
		if name == "receive" {
			st = "received"
		}
		_, _ = s.DB.Exec(`UPDATE pd_dispatch SET status=? WHERE id=?`, st, id)
		api.OK(c, gin.H{"id": id, "status": st})
		return true
	}
	return false
}

func (s *Services) handleReportWorks(c *gin.Context, method, action, path string) bool {
	_ = method
	_ = path
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_report_work`)
	case "create":
		body := bindBody(c)
		if !s.canCreateReportWorkBackfill(c, body) {
			return s.rejectReportWorkCreate(c)
		}
		did, _ := asInt64(body["dispatch_id"])
		pid, _ := asInt64(body["process_id"])
		wid, _ := asInt64(body["worker_id"])
		qty, _ := asFloat(body["qty"])
		if qty <= 0 {
			api.FailJSON(c, "INVALID_QTY")
			return true
		}
		docNo := fmt.Sprintf("RW%d", time.Now().UnixNano()%1e12)
		now := time.Now().Format("2006-01-02 15:04:05")
		res, err := s.DB.Exec(`INSERT INTO pd_report_work(doc_no, dispatch_id, process_id, worker_id, qty, weight, status, reported_at) VALUES(?,?,?,?,?,?,'submitted',?)`,
			docNo, did, pid, wid, qty, nullFloat(body["weight"]), now)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		rid, _ := res.LastInsertId()
		var rate float64
		_ = s.DB.QueryRow(`SELECT rate FROM pay_process_wage_rate WHERE process_id=? AND status='active' ORDER BY id DESC LIMIT 1`, pid).Scan(&rate)
		amount := qty * rate
		bizDate := time.Now().Format("2006-01-02")
		_, _ = s.DB.Exec(`INSERT INTO pd_piecework_summary(worker_id, process_id, biz_date, qty, amount, status)
			VALUES(?,?,?,?,?,'open')`, wid, pid, bizDate, qty, amount)
		api.OK(c, gin.H{"id": rid, "doc_no": docNo, "dispatch_id": did, "process_id": pid, "worker_id": wid,
			"qty": qty, "wage_amount": amount, "rate": rate, "status": "submitted"})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM pd_report_work WHERE id=?`, paramID(c))
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_report_work SET qty=COALESCE(NULLIF(?,0),qty), status=COALESCE(NULLIF(?,''),status) WHERE id=? AND status IN ('submitted','draft','confirm_pending')`,
			nullFloat(body["qty"]), strOr(body["status"]), id)
		return s.getSimpleRow(c, `SELECT * FROM pd_report_work WHERE id=?`, id)
	case "action:confirm", "action:cancel":
		id := paramID(c)
		st := "confirmed"
		if strings.Contains(action, "cancel") {
			st = "cancelled"
		}
		_, _ = s.DB.Exec(`UPDATE pd_report_work SET status=? WHERE id=?`, st, id)
		api.OK(c, gin.H{"id": id, "status": st})
		return true
	}
	return false
}

func (s *Services) handleRequisitions(c *gin.Context, method, action, path string) bool {
	_ = method
	_ = path
	switch action {
	case "list":
		return s.listDocTable(c, `SELECT * FROM pd_material_requisition WHERE COALESCE(is_deleted,0)=0`)
	case "create":
		body := bindBody(c)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("MR%d", time.Now().UnixNano()%1e12))
		res, err := s.DB.Exec(`INSERT INTO pd_material_requisition(doc_no, work_order_id, dispatch_id, warehouse_id, status)
			VALUES(?,?,?,?,'draft')`,
			docNo, nullInt64Or(body["work_order_id"]), nullInt64Or(body["dispatch_id"]), nullInt64Or(body["warehouse_id"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		lines, _ := body["lines"].([]interface{})
		for _, ln := range lines {
			m, _ := ln.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			qty, _ := asFloat(m["qty"])
			if pid <= 0 || qty <= 0 {
				continue
			}
			_, _ = s.DB.Exec(`INSERT INTO pd_material_requisition_line(requisition_id, product_id, qty, base_qty, batch_no) VALUES(?,?,?,?,?)`,
				id, pid, qty, qty, strOr(m["batch_no"]))
		}
		if pid, _ := asInt64(body["product_id"]); pid > 0 {
			qty, _ := asFloat(body["qty"])
			if qty <= 0 {
				qty = 1
			}
			_, _ = s.DB.Exec(`INSERT INTO pd_material_requisition_line(requisition_id, product_id, qty, base_qty) VALUES(?,?,?,?)`, id, pid, qty, qty)
		}
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "status": "draft"})
		return true
	case "get":
		id := paramID(c)
		rows, err := s.DB.Query(`SELECT * FROM pd_material_requisition WHERE id=? AND COALESCE(is_deleted,0)=0`, id)
		if err != nil {
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
		lrows, _ := s.DB.Query(`SELECT * FROM pd_material_requisition_line WHERE requisition_id=?`, id)
		lines := []map[string]interface{}{}
		if lrows != nil {
			defer lrows.Close()
			lines, _ = rowsToMaps(lrows)
		}
		m["lines"] = lines
		api.OK(c, m)
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE pd_material_requisition SET warehouse_id=COALESCE(NULLIF(?,0),warehouse_id) WHERE id=? AND status='draft'`,
			nullInt64Or(body["warehouse_id"]), id)
		return s.handleRequisitions(c, "GET", "get", path)
	case "action:post", "action:confirm":
		id := paramID(c)
		var status string
		var wh sql.NullInt64
		err := s.DB.QueryRow(`SELECT status, warehouse_id FROM pd_material_requisition WHERE id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&status, &wh)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if status == "posted" {
			api.OK(c, gin.H{"id": id, "status": "posted"})
			return true
		}
		lrows, _ := s.DB.Query(`SELECT product_id, qty FROM pd_material_requisition_line WHERE requisition_id=?`, id)
		txnLines := []txnLine{}
		if lrows != nil {
			defer lrows.Close()
			for lrows.Next() {
				var pid int64
				var qty float64
				if lrows.Scan(&pid, &qty) == nil && pid > 0 && qty > 0 {
					txnLines = append(txnLines, txnLine{pid: pid, qty: qty, dir: "out"})
				}
			}
		}
		whID := int64(1)
		if wh.Valid && wh.Int64 > 0 {
			whID = wh.Int64
		}
		if len(txnLines) > 0 {
			_, _ = s.insertPostedStockTxn("consume", whID, time.Now().Format("2006-01-02"), "", txnLines, fmt.Sprintf("requisition:%d", id))
		}
		_, _ = s.DB.Exec(`UPDATE pd_material_requisition SET status='posted', updated_at=NOW() WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "status": "posted"})
		return true
	}
	return false
}

func resourceKeyFromPath(path string) string {
	p := strings.TrimPrefix(path, "/api/v1/")
	parts := []string{}
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, "{") {
			break
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, "/")
}

func (s *Services) genericDoc(c *gin.Context, method, action, rk string) bool {
	switch action {
	case "list":
		pn, ps := 1, 50
		list, total, err := s.Store.List(rk, pn, ps)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		out := []map[string]interface{}{}
		for _, d := range list {
			out = append(out, d.Payload)
		}
		api.OK(c, gin.H{"list": out, "total": total})
		return true
	case "create":
		d, err := s.Store.Create(rk, bindBody(c), "draft")
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		api.OK(c, d.Payload)
		return true
	case "get":
		d, _ := s.Store.Get(paramID(c))
		if d == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, d.Payload)
		return true
	case "update", "replace":
		d, err := s.Store.Update(paramID(c), bindBody(c), "")
		if err != nil || d == nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, d.Payload)
		return true
	case "delete":
		_ = s.Store.Delete(paramID(c))
		api.OK(c, gin.H{})
		return true
	default:
		if strings.HasPrefix(action, "action:") {
			name := strings.TrimPrefix(action, "action:")
			// payroll sheet calc: aggregate piecework lines
			if name == "calc" || name == "generate" || name == "batch-generate" {
				body := bindBody(c)
				lines, _, _ := s.Store.List("payroll/piecework-lines", 1, 500)
				total := 0.0
				for _, ln := range lines {
					if a, ok := asFloat(ln.Payload["amount"]); ok {
						total += a
					}
				}
				body["piecework_total"] = total
				body["line_count"] = len(lines)
				d, err := s.Store.Create(rk, body, "calculated")
				if err != nil {
					api.FailJSON(c, "DB_ERROR")
					return true
				}
				api.OK(c, d.Payload)
				return true
			}
			d, err := s.ApplyDocAction(paramID(c), name, bindBody(c))
			if err != nil {
				api.FailJSON(c, err.Error())
				return true
			}
			api.OK(c, d)
			return true
		}
	}
	return false
}

func (s *Services) handleEmployees(c *gin.Context, method, action string) bool {
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT e.id, e.emp_no, e.name, COALESCE(e.org_id,0), COALESCE(e.dept_id,0), COALESCE(e.team_id,0),
			COALESCE(e.job_title_id,0), COALESCE(j.name,''), e.emp_type, e.status, COALESCE(e.user_id,0), COALESCE(e.badge_code,''), COALESCE(e.mobile,''),
			COALESCE(e.id_card_no,''), COALESCE(u.login_name,''), COALESCE(p.bank_account,''), COALESCE(p.tax_no,'')
			FROM hr_employee e
			LEFT JOIN hr_job_title j ON j.id=e.job_title_id AND COALESCE(j.is_deleted,0)=0
			LEFT JOIN iam_user u ON u.id=e.user_id AND COALESCE(u.is_deleted,0)=0
			LEFT JOIN pay_worker_profile p ON p.employee_id=e.id
			WHERE COALESCE(e.is_deleted,0)=0 ORDER BY e.id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, org, dept, team, uid, jobTitleID int64
			var no, name, jobTitleName, typ, status, badge, mobile, idCard, login, bank, tax string
			_ = rows.Scan(&id, &no, &name, &org, &dept, &team, &jobTitleID, &jobTitleName, &typ, &status, &uid, &badge, &mobile, &idCard, &login, &bank, &tax)
			if login == "" && uid > 0 {
				_ = s.DB.QueryRow(`SELECT COALESCE(login_name,'') FROM iam_user WHERE employee_id=? AND COALESCE(is_deleted,0)=0 LIMIT 1`, id).Scan(&login)
			}
			badge = s.ensureEmployeeBadge(id, no, badge)
			row := gin.H{
				"id": id, "emp_no": no, "name": name, "org_id": org, "dept_id": dept, "team_id": team,
				"job_title_id": jobTitleID, "job_title_name": jobTitleName,
				"emp_type": typ, "status": status, "user_id": uid, "badge_code": badge, "mobile": mobile,
				"id_card_no": idCard, "login_name": login, "has_account": uid > 0 || login != "",
				"bank_account": bank, "tax_no": tax,
			}
			list = append(list, s.enrichEmployeeDeptFields(row))
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "create":
		body := bindBody(c)
		id, errMsg := s.createEmployeeFromBody(body, strOrDef(body["status"], "active"))
		if errMsg != "" {
			api.FailJSON(c, errMsg)
			return true
		}
		m := s.loadEmployeeMapEnriched(id)
		// 默认自动开户；批量导入可传 open_account=false
		openAcc := true
		if v, ok := body["open_account"].(bool); ok {
			openAcc = v
		} else if flag := strings.TrimSpace(strOr(body["open_account"])); flag == "0" || strings.EqualFold(flag, "false") {
			openAcc = false
		}
		if openAcc {
			login, pass, err := s.openAccountForEmployeeEx(id, "[]", "", "")
			if err != nil {
				m["has_account"] = false
				m["account_error"] = err.Error()
			} else {
				m["has_account"] = true
				m["login_name"] = login
				if pass != "" {
					m["initial_password"] = pass
				}
			}
		} else {
			m["has_account"] = false
		}
		api.OK(c, m)
		return true
	case "get", "update", "delete":
		id := paramID(c)
		if action == "get" {
			m := s.loadEmployeeMapEnriched(id)
			if m["emp_no"] == nil {
				api.FailJSON(c, "NOT_FOUND")
				return true
			}
			m["badge_code"] = s.ensureEmployeeBadge(id, fmt.Sprint(m["emp_no"]), fmt.Sprint(m["badge_code"]))
			api.OK(c, m)
			return true
		}
		if action == "update" {
			body := bindBody(c)
			if msg, err := s.updateEmployeeFromBody(id, body); err != nil || msg != "" {
				if msg == "" {
					msg = "DB_ERROR"
				}
				api.FailJSON(c, msg)
				return true
			}
			if st := strOr(body["status"]); st != "" {
				_, _ = s.DB.Exec(`UPDATE hr_employee SET status=? WHERE id=?`, st, id)
			}
			api.OK(c, s.loadEmployeeMapEnriched(id))
			return true
		}
		_, _ = s.DB.Exec(`UPDATE hr_employee SET status='inactive' WHERE id=?`, id)
		api.OK(c, gin.H{})
		return true
	}
	return false
}

func (s *Services) handleIAM(c *gin.Context, method, action, path string) bool {
	// GET lists (generated routes)
	if method == "GET" {
		switch {
		case path == "/api/v1/iam/users" || (strings.HasPrefix(path, "/api/v1/iam/users") && action == "list"):
			rows, err := s.DB.Query(`
				SELECT u.id, u.login_name, u.user_type, u.status, COALESCE(e.name,''), COALESCE(u.employee_id,0)
				FROM iam_user u LEFT JOIN hr_employee e ON e.id = u.employee_id
				WHERE u.is_deleted = 0 ORDER BY u.id`)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id, empID int64
				var login, ut, status, name string
				_ = rows.Scan(&id, &login, &ut, &status, &name, &empID)
				list = append(list, gin.H{"id": id, "login_name": login, "user_type": ut, "status": status, "name": name, "employee_id": empID})
			}
			api.OK(c, gin.H{"list": list, "total": len(list)})
			return true
		case path == "/api/v1/iam/roles":
			rows, err := s.DB.Query(`SELECT id, code, name, data_scope_type, is_system, status FROM iam_role WHERE is_deleted = 0`)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id int64
				var code, name, scope, status string
				var isSys int
				_ = rows.Scan(&id, &code, &name, &scope, &isSys, &status)
				list = append(list, gin.H{"id": id, "code": code, "name": name, "data_scope_type": scope, "is_system": isSys == 1, "status": status})
			}
			api.OK(c, gin.H{"list": list})
			return true
		case path == "/api/v1/iam/admin-groups":
			rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(remark,''), sort_no, status FROM iam_admin_group WHERE is_deleted = 0 ORDER BY sort_no`)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id int64
				var code, name, remark, status string
				var sortNo int
				_ = rows.Scan(&id, &code, &name, &remark, &sortNo, &status)
				list = append(list, gin.H{"id": id, "code": code, "name": name, "remark": remark, "sort_no": sortNo, "status": status})
			}
			api.OK(c, gin.H{"list": list})
			return true
		case path == "/api/v1/iam/permissions":
			var n int
			_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_permission WHERE COALESCE(is_deleted,0)=0`).Scan(&n)
			if n == 0 {
				EnsureDomainPermissions(s.DB)
			}
			rows, err := s.DB.Query(`SELECT id, code, name, domain, module, action FROM iam_permission WHERE COALESCE(is_deleted,0)=0 ORDER BY domain, module, action, id`)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id int64
				var code, name, domain, module, act string
				_ = rows.Scan(&id, &code, &name, &domain, &module, &act)
				list = append(list, gin.H{"id": id, "code": code, "name": name, "domain": domain, "module": module, "action": act})
			}
			api.OK(c, gin.H{"list": list, "total": len(list)})
			return true
		case path == "/api/v1/iam/login-policy":
			var maxFail, lockMin, ttl, minLen, hist int
			var reqLetter, reqDigit, reqSpecial, single int
			err := s.DB.QueryRow(`SELECT max_fail_count, lock_minutes, session_ttl_min, password_min_len, password_require_letter, password_require_digit, password_require_special, password_history, single_session FROM iam_login_policy WHERE id=1`).
				Scan(&maxFail, &lockMin, &ttl, &minLen, &reqLetter, &reqDigit, &reqSpecial, &hist, &single)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			api.OK(c, gin.H{
				"max_fail_count": maxFail, "lock_minutes": lockMin, "session_ttl_min": ttl,
				"password_min_len": minLen, "password_require_letter": reqLetter == 1,
				"password_require_digit": reqDigit == 1, "password_require_special": reqSpecial == 1,
				"password_history": hist, "single_session": single == 1,
			})
			return true
		case path == "/api/v1/iam/field-policies":
			rows, err := s.DB.Query(`SELECT id, role_id, field_key, field_name, visible, editable FROM iam_field_policy`)
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id, rid int64
				var fk, fn string
				var vis, edit int
				_ = rows.Scan(&id, &rid, &fk, &fn, &vis, &edit)
				list = append(list, gin.H{"id": id, "role_id": rid, "field_key": fk, "field_name": fn, "visible": vis == 1, "editable": edit == 1})
			}
			api.OK(c, gin.H{"list": list})
			return true
		case path == "/api/v1/iam/menus":
			roleID := c.Query("role_id")
			q := `SELECT id, role_id, domain, module, menu_key, visible, sort_no FROM iam_menu_custom`
			var rows *sql.Rows
			var err error
			if roleID != "" {
				rows, err = s.DB.Query(q+` WHERE role_id=?`, roleID)
			} else {
				rows, err = s.DB.Query(q)
			}
			if err != nil {
				api.FailJSON(c, "DB_ERROR")
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id, rid int64
				var domain, module, key string
				var vis, sortNo int
				_ = rows.Scan(&id, &rid, &domain, &module, &key, &vis, &sortNo)
				list = append(list, gin.H{"id": id, "role_id": rid, "domain": domain, "module": module, "menu_key": key, "visible": vis == 1, "sort_no": sortNo})
			}
			api.OK(c, gin.H{"list": list})
			return true
		}
	}
	// Full IAM write paths — list endpoints may already be registered by iam package (first wins).
	// This handles generated duplicates and missing writes.
	switch {
	case path == "/api/v1/iam/users" && method == "POST":
		body := bindBody(c)
		login, _ := body["login_name"].(string)
		pass, _ := body["password"].(string)
		if login == "" || pass == "" {
			api.FailJSON(c, "INVALID_REQUEST")
			return true
		}
		hash, err := security.HashPassword(pass)
		if err != nil {
			api.FailJSON(c, "HASH_ERROR")
			return true
		}
		ut, _ := body["user_type"].(string)
		if ut == "" {
			ut = "admin"
		}
		empID, _ := asInt64(body["employee_id"])
		var res sql.Result
		if empID > 0 {
			res, err = s.DB.Exec(`INSERT INTO iam_user(login_name, password_hash, employee_id, user_type, status, is_deleted) VALUES(?,?,?,?,'active',0)`, login, hash, empID, ut)
		} else {
			res, err = s.DB.Exec(`INSERT INTO iam_user(login_name, password_hash, user_type, status, is_deleted) VALUES(?,?,?,'active',0)`, login, hash, ut)
		}
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		id, _ := res.LastInsertId()
		if empID > 0 {
			_, _ = s.DB.Exec(`UPDATE hr_employee SET user_id=? WHERE id=?`, id, empID)
		}
		if roleIDs, ok := body["role_ids"].([]interface{}); ok {
			extra := []int64{}
			for _, r := range roleIDs {
				if rid, ok := asInt64(r); ok && rid > 0 {
					extra = append(extra, rid)
				}
			}
			appendExtraRoleIDs(s.DB, id, extra)
		}
		s.rebuildUserEffectiveRoles(id)
		api.OK(c, gin.H{"id": id, "login_name": login, "status": "active", "employee_id": empID})
		return true
	case strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/roles") && method == "PUT":
		uid := paramID(c)
		body := bindBody(c)
		roleIDs, _ := body["role_ids"].([]interface{})
		extra := []int64{}
		for _, r := range roleIDs {
			if rid, ok := asInt64(r); ok && rid > 0 {
				extra = append(extra, rid)
			}
		}
		if err := s.setExtraRoleIDs(uid, extra); err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"user_id": uid, "extra_role_ids": extra})
		return true
	case strings.HasSuffix(path, "/freeze") && method == "POST":
		uid := paramID(c)
		if s.refuseIfFounderProtected(c, uid) {
			return true
		}
		body := bindBody(c)
		reason, _ := body["reason"].(string)
		if reason == "" {
			reason = "manual freeze"
		}
		var by interface{}
		if claims := middleware.Claims(c); claims != nil {
			by = claims.UserID
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		_, _ = s.DB.Exec(`UPDATE iam_user SET status='frozen', freeze_reason=?, frozen_at=?, frozen_by=? WHERE id=?`, reason, now, by, uid)
		_, _ = s.DB.Exec(`DELETE FROM iam_user_session WHERE user_id=?`, uid)
		api.OK(c, gin.H{"id": uid, "status": "frozen"})
		return true
	case strings.HasSuffix(path, "/unfreeze") && method == "POST":
		uid := paramID(c)
		_, _ = s.DB.Exec(`UPDATE iam_user SET status='active', freeze_reason=NULL, frozen_at=NULL, frozen_by=NULL WHERE id=?`, uid)
		api.OK(c, gin.H{"id": uid, "status": "active"})
		return true
	case path == "/api/v1/iam/users/{id}" && method == "PUT":
		uid := paramID(c)
		body := bindBody(c)
		if st, ok := body["status"].(string); ok {
			if st == "frozen" && s.refuseIfFounderProtected(c, uid) {
				return true
			}
			_, _ = s.DB.Exec(`UPDATE iam_user SET status=? WHERE id=?`, st, uid)
		}
		api.OK(c, gin.H{"id": uid})
		return true
	case path == "/api/v1/iam/roles" && method == "POST":
		return s.createRoleIAM(c)
	case path == "/api/v1/iam/roles/{id}" && method == "PUT":
		return s.updateRoleIAM(c)
	case path == "/api/v1/iam/permissions/sync" && method == "POST":
		EnsureDomainPermissions(s.DB)
		security.InvalidateAllRBAC()
		var total int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_permission WHERE COALESCE(is_deleted,0)=0`).Scan(&total)
		api.OK(c, gin.H{"synced": true, "total": total})
		return true
	case path == "/api/v1/iam/roles/{id}/permissions" && method == "PUT":
		rid := paramID(c)
		_, isSystem, ok := s.loadRoleIAMMeta(rid)
		if !ok {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if isSystem {
			return rejectSystemRoleReadonly(c)
		}
		body := bindBody(c)
		codes, _ := body["permission_ids"].([]interface{})
		if codes == nil {
			codes, _ = body["permission_codes"].([]interface{})
		}
		resolvedCodes, ok := s.resolveSubmittedPermissionCodes(codes)
		if !ok {
			api.FailJSON(c, "INVALID_PERMISSION_CODE")
			return true
		}
		claims := middleware.Claims(c)
		if claims == nil {
			api.FailJSON(c, "UNAUTHORIZED")
			return true
		}
		if !claimsIsSysAdmin(claims.Roles, claims.Permissions) {
			owned := permissionCodeSet(claims.Permissions)
			missing := make([]string, 0, 4)
			for _, code := range resolvedCodes {
				if _, ok := owned[code]; !ok {
					missing = append(missing, code)
				}
			}
			if len(missing) > 0 {
				api.OK(c, gin.H{"code": 0, "msg": "PERM_ESCALATION_DENIED", "missing_permission_codes": missing})
				return true
			}
		}
		_, _ = s.DB.Exec(`DELETE FROM iam_role_permission WHERE role_id=?`, rid)
		for _, x := range codes {
			if id, ok := asInt64(x); ok {
				_, _ = s.DB.Exec(`INSERT INTO iam_role_permission(role_id, permission_id) VALUES(?,?)`, rid, id)
				continue
			}
			if code, ok := x.(string); ok {
				var pid int64
				_ = s.DB.QueryRow(`SELECT id FROM iam_permission WHERE code=?`, code).Scan(&pid)
				if pid > 0 {
					_, _ = s.DB.Exec(`INSERT INTO iam_role_permission(role_id, permission_id) VALUES(?,?)`, rid, pid)
				}
			}
		}
		security.InvalidateAllRBAC()
		api.OK(c, gin.H{"role_id": rid})
		return true
	case path == "/api/v1/iam/menus" && method == "PUT":
		body := bindBody(c)
		items, _ := body["items"].([]interface{})
		roleCleared := map[int64]bool{}
		for _, it := range items {
			m, _ := it.(map[string]interface{})
			if m == nil {
				continue
			}
			rid, _ := asInt64(m["role_id"])
			if rid > 0 && !roleCleared[rid] {
				_, _ = s.DB.Exec(`DELETE FROM iam_menu_custom WHERE role_id=?`, rid)
				roleCleared[rid] = true
			}
			domain, _ := m["domain"].(string)
			module, _ := m["module"].(string)
			key, _ := m["menu_key"].(string)
			vis := 1
			switch v := m["visible"].(type) {
			case bool:
				if !v {
					vis = 0
				}
			case float64:
				if v == 0 {
					vis = 0
				}
			case string:
				if v == "0" || strings.EqualFold(v, "false") {
					vis = 0
				}
			}
			sortNo, _ := asInt64(m["sort_no"])
			_, _ = s.DB.Exec(`INSERT INTO iam_menu_custom(role_id, domain, module, menu_key, visible, sort_no) VALUES(?,?,?,?,?,?)`,
				rid, domain, module, key, vis, sortNo)
		}
		api.OK(c, gin.H{"ok": true, "count": len(items)})
		return true
	case path == "/api/v1/iam/field-policies" && method == "PUT":
		body := bindBody(c)
		items, _ := body["items"].([]interface{})
		roleCleared := map[int64]bool{}
		for _, it := range items {
			m, _ := it.(map[string]interface{})
			if m == nil {
				continue
			}
			rid, _ := asInt64(m["role_id"])
			if rid > 0 && !roleCleared[rid] {
				_, _ = s.DB.Exec(`DELETE FROM iam_field_policy WHERE role_id=?`, rid)
				roleCleared[rid] = true
			}
			fk, _ := m["field_key"].(string)
			fn, _ := m["field_name"].(string)
			vis, edit := 1, 0
			switch v := m["visible"].(type) {
			case bool:
				if !v {
					vis = 0
				}
			case float64:
				if v == 0 {
					vis = 0
				}
			}
			switch v := m["editable"].(type) {
			case bool:
				if v {
					edit = 1
				}
			case float64:
				if v != 0 {
					edit = 1
				}
			}
			_, _ = s.DB.Exec(`INSERT INTO iam_field_policy(role_id, field_key, field_name, visible, editable) VALUES(?,?,?,?,?)`,
				rid, fk, fn, vis, edit)
		}
		api.OK(c, gin.H{"ok": true, "count": len(items)})
		return true
	case path == "/api/v1/iam/login-policy" && method == "PUT":
		body := bindBody(c)
		maxFail, _ := asInt64(body["max_fail_count"])
		lockMin, _ := asInt64(body["lock_minutes"])
		ttl, _ := asInt64(body["session_ttl_min"])
		minLen, _ := asInt64(body["password_min_len"])
		_, _ = s.DB.Exec(`UPDATE iam_login_policy SET max_fail_count=COALESCE(NULLIF(?,0),max_fail_count), lock_minutes=COALESCE(NULLIF(?,0),lock_minutes), session_ttl_min=COALESCE(NULLIF(?,0),session_ttl_min), password_min_len=COALESCE(NULLIF(?,0),password_min_len) WHERE id=1`,
			maxFail, lockMin, ttl, minLen)
		api.OK(c, gin.H{"ok": true})
		return true
	case path == "/api/v1/iam/admin-groups" && method == "POST":
		body := bindBody(c)
		code, _ := body["code"].(string)
		name, _ := body["name"].(string)
		res, err := s.DB.Exec(`INSERT INTO iam_admin_group(code, name, sort_no, status) VALUES(?,?,10,'active')`, code, name)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name})
		return true
	case path == "/api/v1/iam/sessions/{id}/revoke" && method == "POST":
		id := paramID(c)
		_, _ = s.DB.Exec(`DELETE FROM iam_user_session WHERE id=?`, id)
		api.OK(c, gin.H{"id": id, "revoked": true})
		return true
	case strings.Contains(path, "warehouse-scope") && method == "PUT":
		rid := paramID(c)
		_, isSystem, ok := s.loadRoleIAMMeta(rid)
		if !ok {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if isSystem {
			return rejectSystemRoleReadonly(c)
		}
		body := bindBody(c)
		ids, _ := body["warehouse_ids"].([]interface{})
		_, _ = s.DB.Exec(`DELETE FROM iam_role_warehouse_scope WHERE role_id=?`, rid)
		for _, x := range ids {
			wid, _ := asInt64(x)
			if wid > 0 {
				_, _ = s.DB.Exec(`INSERT INTO iam_role_warehouse_scope(role_id, warehouse_id) VALUES(?,?)`, rid, wid)
			}
		}
		api.OK(c, gin.H{"role_id": rid, "warehouse_ids": ids})
		return true
	case strings.Contains(path, "process-scope") && method == "PUT":
		rid := paramID(c)
		_, isSystem, ok := s.loadRoleIAMMeta(rid)
		if !ok {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		if isSystem {
			return rejectSystemRoleReadonly(c)
		}
		body := bindBody(c)
		_, _ = s.DB.Exec(`DELETE FROM iam_role_process_scope WHERE role_id=?`, rid)
		if items, ok := body["items"].([]interface{}); ok && len(items) > 0 {
			for _, it := range items {
				m, _ := it.(map[string]interface{})
				if m == nil {
					continue
				}
				pid, _ := asInt64(m["process_id"])
				if pid <= 0 {
					continue
				}
				canReport, canDispatch := 1, 0
				if v, ok := m["can_report"].(bool); ok && !v {
					canReport = 0
				}
				if v, ok := m["can_dispatch"].(bool); ok && v {
					canDispatch = 1
				}
				_, _ = s.DB.Exec(`INSERT INTO iam_role_process_scope(role_id, process_id, can_report, can_dispatch) VALUES(?,?,?,?)`,
					rid, pid, canReport, canDispatch)
			}
		} else if ids, ok := body["process_ids"].([]interface{}); ok {
			for _, x := range ids {
				pid, _ := asInt64(x)
				if pid > 0 {
					_, _ = s.DB.Exec(`INSERT INTO iam_role_process_scope(role_id, process_id, can_report, can_dispatch) VALUES(?,?,1,0)`, rid, pid)
				}
			}
		}
		api.OK(c, gin.H{"role_id": rid})
		return true
	}
	// fallback lists for generated iam routes not in old package
	if method == "GET" && (action == "list" || action == "get") {
		if strings.Contains(path, "/sessions") {
			rows, err := s.DB.Query(`SELECT id, user_id, COALESCE(client_type,''), login_at FROM iam_user_session WHERE revoked_at IS NULL ORDER BY id DESC`)
			if err != nil {
				api.OK(c, gin.H{"list": []gin.H{}, "total": 0})
				return true
			}
			defer rows.Close()
			list := []gin.H{}
			for rows.Next() {
				var id, uid int64
				var ct, created string
				_ = rows.Scan(&id, &uid, &ct, &created)
				list = append(list, gin.H{"id": id, "user_id": uid, "client_type": ct, "created_at": created, "login_at": created})
			}
			api.OK(c, gin.H{"list": list, "total": len(list)})
			return true
		}
	}
	return false
}

func (s *Services) createRoleIAM(c *gin.Context) bool {
	body := bindBody(c)
	code, _ := body["code"].(string)
	name, _ := body["name"].(string)
	if code == "" || name == "" {
		api.FailJSON(c, "CODE_NAME_REQUIRED")
		return true
	}
	scope, _ := body["data_scope_type"].(string)
	if scope == "" {
		scope = "self"
	}
	remark, _ := body["remark"].(string)
	res, err := s.DB.Exec(`INSERT INTO iam_role(code, name, data_scope_type, remark, is_system, status, is_deleted) VALUES(?,?,?,?,0,'active',0)`,
		code, name, scope, remark)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	api.OK(c, gin.H{"id": id, "code": code, "name": name, "data_scope_type": scope, "is_system": false, "status": "active", "remark": remark})
	return true
}

func (s *Services) updateRoleIAM(c *gin.Context) bool {
	rid := paramID(c)
	body := bindBody(c)
	var code, name, scope, status, remark string
	var isSys int
	err := s.DB.QueryRow(`SELECT code, name, data_scope_type, status, COALESCE(remark,''), is_system FROM iam_role WHERE id=? AND COALESCE(is_deleted,0)=0`, rid).
		Scan(&code, &name, &scope, &status, &remark, &isSys)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if isSys == 1 {
		return rejectSystemRoleReadonly(c)
	}
	if v, ok := body["name"].(string); ok && v != "" {
		name = v
	}
	if v, ok := body["code"].(string); ok && v != "" && isSys == 0 {
		code = v
	}
	if v, ok := body["data_scope_type"].(string); ok && v != "" {
		scope = v
	}
	if v, ok := body["remark"].(string); ok {
		remark = v
	}
	if v, ok := body["status"].(string); ok && v != "" {
		status = v
	}
	_, err = s.DB.Exec(`UPDATE iam_role SET code=?, name=?, data_scope_type=?, remark=?, status=?, updated_at=NOW() WHERE id=?`,
		code, name, scope, remark, status, rid)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, gin.H{"id": rid, "code": code, "name": name, "data_scope_type": scope, "status": status, "remark": remark, "is_system": isSys == 1})
	return true
}

func (s *Services) getRoleDetailIAM(c *gin.Context) bool {
	rid := paramID(c)
	var code, name, scope, status, remark string
	var isSys int
	err := s.DB.QueryRow(`SELECT code, name, data_scope_type, status, COALESCE(remark,''), is_system FROM iam_role WHERE id=? AND COALESCE(is_deleted,0)=0`, rid).
		Scan(&code, &name, &scope, &status, &remark, &isSys)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	permIDs := []int64{}
	if rows, e := s.DB.Query(`SELECT permission_id FROM iam_role_permission WHERE role_id=?`, rid); e == nil {
		defer rows.Close()
		for rows.Next() {
			var pid int64
			_ = rows.Scan(&pid)
			permIDs = append(permIDs, pid)
		}
	}
	whIDs := []int64{}
	if rows, e := s.DB.Query(`SELECT warehouse_id FROM iam_role_warehouse_scope WHERE role_id=?`, rid); e == nil {
		defer rows.Close()
		for rows.Next() {
			var wid int64
			_ = rows.Scan(&wid)
			whIDs = append(whIDs, wid)
		}
	}
	procScopes := []gin.H{}
	if rows, e := s.DB.Query(`SELECT process_id, can_report, can_dispatch FROM iam_role_process_scope WHERE role_id=?`, rid); e == nil {
		defer rows.Close()
		for rows.Next() {
			var pid int64
			var cr, cd int
			_ = rows.Scan(&pid, &cr, &cd)
			procScopes = append(procScopes, gin.H{"process_id": pid, "can_report": cr == 1, "can_dispatch": cd == 1})
		}
	}
	boundUsers := []gin.H{}
	if rows, e := s.DB.Query(`
		SELECT u.id, u.login_name, u.status, COALESCE(e.name,'')
		FROM iam_user_role ur
		JOIN iam_user u ON u.id=ur.user_id AND COALESCE(u.is_deleted,0)=0
		LEFT JOIN hr_employee e ON e.id=u.employee_id
		WHERE ur.role_id=?`, rid); e == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var login, st, ename string
			_ = rows.Scan(&id, &login, &st, &ename)
			boundUsers = append(boundUsers, gin.H{"id": id, "login_name": login, "status": st, "name": ename})
		}
	}
	warehouses := []gin.H{}
	if rows, e := s.DB.Query(`SELECT id, code, name FROM inv_warehouse WHERE COALESCE(is_deleted,0)=0 ORDER BY id`); e == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var c, n string
			_ = rows.Scan(&id, &c, &n)
			warehouses = append(warehouses, gin.H{"id": id, "code": c, "name": n})
		}
	}
	processes := []gin.H{}
	if rows, e := s.DB.Query(`SELECT id, code, name FROM pd_process WHERE COALESCE(is_deleted,0)=0 ORDER BY id`); e == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var c, n string
			_ = rows.Scan(&id, &c, &n)
			processes = append(processes, gin.H{"id": id, "code": c, "name": n})
		}
	}
	api.OK(c, gin.H{
		"role": gin.H{
			"id": rid, "code": code, "name": name, "data_scope_type": scope,
			"status": status, "remark": remark, "is_system": isSys == 1,
		},
		"permission_ids": permIDs,
		"warehouse_ids":  whIDs,
		"process_scopes": procScopes,
		"bound_users":    boundUsers,
		"warehouses":     warehouses,
		"processes":      processes,
	})
	return true
}
