package biz

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/persistence/sqlutil"
)

type flowGraphDoc struct {
	Nodes []flowGraphNode `json:"nodes"`
	Edges []flowGraphEdge `json:"edges"`
}

type flowGraphNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

type flowGraphEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Data   map[string]interface{} `json:"data"`
}

func (s *Services) handleFlowGraphs(c *gin.Context, method, action, openapiPath string) bool {
	_ = openapiPath
	switch {
	case method == "GET" && (action == "list" || action == ""):
		return s.listFlowGraphs(c)
	case method == "POST" && action == "create":
		return s.createFlowGraph(c)
	case method == "GET" && action == "get":
		return s.getFlowGraph(c)
	case method == "PUT" && (action == "update" || action == "replace"):
		return s.updateFlowGraph(c)
	case method == "DELETE" && action == "delete":
		return s.deleteFlowGraph(c)
	}
	return true
}

func (s *Services) listFlowGraphs(c *gin.Context) bool {
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE COALESCE(is_deleted,0)=0`
	args := []interface{}{}
	if kind := c.Query("kind"); kind != "" {
		where += ` AND kind=?`
		args = append(args, kind)
	}
	if st := c.Query("status"); st != "" {
		where += ` AND status=?`
		args = append(args, st)
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_flow_graph `+where, args...).Scan(&total)
	args2 := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, code, name, kind, status, COALESCE(routing_id,0), COALESCE(graph_json,'{}'),
		COALESCE(version_no,'V1'), created_at, updated_at FROM pd_flow_graph `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args2...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, rid int64
		var code, name, kind, status, gjson, ver, created, updated string
		_ = rows.Scan(&id, &code, &name, &kind, &status, &rid, &gjson, &ver, &created, &updated)
		list = append(list, gin.H{
			"id": id, "code": code, "name": name, "kind": kind, "status": status,
			"routing_id": rid, "graph_json": gjson, "version_no": ver, "created_at": created, "updated_at": updated,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) loadFlowGraphRow(id int64) gin.H {
	var rid int64
	var code, name, kind, status, gjson, ver, created, updated string
	err := s.DB.QueryRow(`SELECT id, code, name, kind, status, COALESCE(routing_id,0), COALESCE(graph_json,'{}'),
		COALESCE(version_no,'V1'), created_at, updated_at FROM pd_flow_graph WHERE id=? AND COALESCE(is_deleted,0)=0`, id).
		Scan(&id, &code, &name, &kind, &status, &rid, &gjson, &ver, &created, &updated)
	if err != nil {
		return nil
	}
	var productID int64
	if rid > 0 {
		_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0) FROM pd_routing WHERE id=?`, rid).Scan(&productID)
	}
	return gin.H{
		"id": id, "code": code, "name": name, "kind": kind, "status": status,
		"routing_id": rid, "product_id": productID, "graph_json": gjson, "version_no": ver,
		"created_at": created, "updated_at": updated,
	}
}

func (s *Services) getFlowGraph(c *gin.Context) bool {
	m := s.loadFlowGraphRow(paramID(c))
	if m == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, m)
	return true
}

func (s *Services) createFlowGraph(c *gin.Context) bool {
	body := bindBody(c)
	code := strings.TrimSpace(strOr(body["code"]))
	name := strings.TrimSpace(strOr(body["name"]))
	kind := strings.TrimSpace(strOr(body["kind"]))
	if code == "" || name == "" || kind == "" {
		api.FailJSON(c, "CODE_NAME_KIND_REQUIRED")
		return true
	}
	if kind != "production" && kind != "purchase_gate" && kind != "purchase_stockin" {
		api.FailJSON(c, "INVALID_KIND")
		return true
	}
	status := strOrDef(body["status"], "draft")
	gjson := graphJSONFromBody(body)
	ver := strOrDef(body["version_no"], "V1")
	routingID := asInt64Or0(body["routing_id"])

	if status == "active" {
		s.deactivateOtherFlowGraphs(kind, 0)
	}

	if kind == "production" && status == "active" {
		rid, errMsg := s.compileProductionGraph(routingID, code, name, gjson, body)
		if errMsg != "" {
			api.FailJSON(c, errMsg)
			return true
		}
		routingID = rid
	}

	res, err := s.DB.Exec(`INSERT INTO pd_flow_graph(code, name, kind, status, routing_id, graph_json, version_no)
		VALUES(?,?,?,?,?,?,?)`, code, name, kind, status, nullIf0(routingID), gjson, ver)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	id, _ := res.LastInsertId()
	if kind == "production" && routingID > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_routing SET graph_json=? WHERE id=?`, gjson, routingID)
	}
	api.OK(c, s.loadFlowGraphRow(id))
	return true
}

func (s *Services) updateFlowGraph(c *gin.Context) bool {
	id := paramID(c)
	cur := s.loadFlowGraphRow(id)
	if cur == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	body := bindBody(c)
	code := strOrDef(body["code"], strOr(cur["code"]))
	name := strOrDef(body["name"], strOr(cur["name"]))
	kind := strOrDef(body["kind"], strOr(cur["kind"]))
	status := strOrDef(body["status"], strOr(cur["status"]))
	ver := strOrDef(body["version_no"], strOr(cur["version_no"]))
	gjson := graphJSONFromBody(body)
	if gjson == "{}" || gjson == "" {
		gjson = strOr(cur["graph_json"])
	}
	routingID := asInt64Or0(cur["routing_id"])
	if v := asInt64Or0(body["routing_id"]); v > 0 {
		routingID = v
	}

	if status == "active" {
		s.deactivateOtherFlowGraphs(kind, id)
	}

	if kind == "production" && status == "active" {
		rid, errMsg := s.compileProductionGraph(routingID, code, name, gjson, body)
		if errMsg != "" {
			api.FailJSON(c, errMsg)
			return true
		}
		routingID = rid
	}

	_, err := s.DB.Exec(`UPDATE pd_flow_graph SET code=?, name=?, kind=?, status=?, routing_id=?, graph_json=?, version_no=?, updated_at=NOW() WHERE id=?`,
		code, name, kind, status, nullIf0(routingID), gjson, ver, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	if kind == "production" && routingID > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_routing SET graph_json=?, name=?, code=?, status=? WHERE id=?`, gjson, name, code, status, routingID)
	}
	api.OK(c, s.loadFlowGraphRow(id))
	return true
}

func (s *Services) deleteFlowGraph(c *gin.Context) bool {
	id := paramID(c)
	_, _ = s.DB.Exec(`UPDATE pd_flow_graph SET is_deleted=1, updated_at=NOW() WHERE id=?`, id)
	api.OK(c, gin.H{"id": id})
	return true
}

func (s *Services) deactivateOtherFlowGraphs(kind string, exceptID int64) {
	if exceptID > 0 {
		_, _ = s.DB.Exec(`UPDATE pd_flow_graph SET status='draft', updated_at=NOW() WHERE kind=? AND status='active' AND id<>? AND COALESCE(is_deleted,0)=0`, kind, exceptID)
	} else {
		_, _ = s.DB.Exec(`UPDATE pd_flow_graph SET status='draft', updated_at=NOW() WHERE kind=? AND status='active' AND COALESCE(is_deleted,0)=0`, kind)
	}
}

func graphJSONFromBody(body map[string]interface{}) string {
	if raw, ok := body["graph_json"]; ok {
		switch t := raw.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				return t
			}
		case map[string]interface{}:
			b, _ := json.Marshal(t)
			return string(b)
		}
	}
	if nodes, ok := body["nodes"]; ok {
		b, _ := json.Marshal(map[string]interface{}{"nodes": nodes, "edges": body["edges"]})
		return string(b)
	}
	return "{}"
}

func parseFlowGraph(gjson string) (*flowGraphDoc, error) {
	var doc flowGraphDoc
	if err := json.Unmarshal([]byte(gjson), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Services) compileProductionGraph(routingID int64, code, name, gjson string, body map[string]interface{}) (int64, string) {
	doc, err := parseFlowGraph(gjson)
	if err != nil {
		return 0, "GRAPH_JSON_INVALID"
	}
	steps, errMsg := compileProcessSteps(doc)
	if errMsg != "" {
		return 0, errMsg
	}
	productID := asInt64Or0(body["product_id"])
	ver := strOrDef(body["version_no"], "V1")
	status := strOrDef(body["status"], "active")
	if routingID <= 0 {
		res, err := s.DB.Exec(`INSERT INTO pd_routing(code, name, product_id, version_no, status, graph_json) VALUES(?,?,?,?,?,?)`,
			code, name, nullIf0(productID), ver, status, gjson)
		if err != nil {
			_ = s.DB.QueryRow(`SELECT id FROM pd_routing WHERE code=? AND COALESCE(is_deleted,0)=0 ORDER BY id LIMIT 1`, code).Scan(&routingID)
			if routingID <= 0 {
				return 0, "DB_ERROR:" + err.Error()
			}
			_, _ = s.DB.Exec(`UPDATE pd_routing SET name=?, version_no=?, status=?, graph_json=? WHERE id=?`, name, ver, status, gjson, routingID)
		} else {
			routingID, _ = res.LastInsertId()
		}
	} else {
		_, _ = s.DB.Exec(`UPDATE pd_routing SET code=?, name=?, version_no=?, status=?, graph_json=? WHERE id=?`,
			code, name, ver, status, gjson, routingID)
		if productID > 0 {
			_, _ = s.DB.Exec(`UPDATE pd_routing SET product_id=? WHERE id=?`, productID, routingID)
		}
	}
	if errMsg := s.upsertRoutingSteps(routingID, steps); errMsg != "" {
		return 0, errMsg
	}
	if errMsg := s.validateRoutingOutputProducts(routingID, steps, productID); errMsg != "" {
		return 0, errMsg
	}
	return routingID, ""
}

// upsertRoutingSteps updates steps by step_code (fallback seq_no) to preserve IDs referenced by inv_box_code.
func (s *Services) upsertRoutingSteps(routingID int64, steps []compiledStep) string {
	type exist struct {
		id   int64
		code string
		seq  int64
	}
	rows, err := s.DB.Query(`SELECT id, COALESCE(step_code,''), seq_no FROM pd_routing_step WHERE routing_id=?`, routingID)
	if err != nil {
		return "DB_ERROR:" + err.Error()
	}
	var existing []exist
	for rows.Next() {
		var e exist
		_ = rows.Scan(&e.id, &e.code, &e.seq)
		existing = append(existing, e)
	}
	_ = rows.Close()

	byCode := map[string]int64{}
	bySeq := map[int64]int64{}
	for _, e := range existing {
		if e.code != "" {
			byCode[e.code] = e.id
		}
		bySeq[e.seq] = e.id
	}

	keep := map[int64]bool{}
	for _, st := range steps {
		id := byCode[st.Code]
		if id <= 0 {
			id = bySeq[st.Seq]
		}
		if id > 0 {
			_, err := s.DB.Exec(`UPDATE pd_routing_step SET seq_no=?, process_id=?, step_code=?, step_name=?,
				is_piecework=?, is_inbound_checkpoint=?, checkpoint_bind_warehouse=?, auto_next=?, auto_stock_in=?, auto_stock_out=?, warehouse_id=?, output_product_id=? WHERE id=?`,
				st.Seq, st.ProcessID, st.Code, st.Name, boolToInt(st.Piece), boolToInt(st.Checkpoint), boolToInt(st.CheckpointBind),
				boolToInt(st.AutoNext), boolToInt(st.StockIn), boolToInt(st.StockOut), nullIf0(st.WarehouseID), nullIf0(st.OutputProductID), id)
			if err != nil {
				return "DB_ERROR:" + err.Error()
			}
			keep[id] = true
			continue
		}
		res, err := s.DB.Exec(`INSERT INTO pd_routing_step(routing_id, seq_no, process_id, step_code, step_name, is_piecework, is_inbound_checkpoint, checkpoint_bind_warehouse, auto_next, auto_stock_in, auto_stock_out, warehouse_id, output_product_id)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			routingID, st.Seq, st.ProcessID, st.Code, st.Name, boolToInt(st.Piece), boolToInt(st.Checkpoint), boolToInt(st.CheckpointBind),
			boolToInt(st.AutoNext), boolToInt(st.StockIn), boolToInt(st.StockOut), nullIf0(st.WarehouseID), nullIf0(st.OutputProductID))
		if err != nil {
			return "DB_ERROR:" + err.Error()
		}
		nid, _ := res.LastInsertId()
		if nid > 0 {
			keep[nid] = true
		}
	}

	for _, e := range existing {
		if keep[e.id] {
			continue
		}
		var refs int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_box_code WHERE current_step_id=? AND COALESCE(is_deleted,0)=0`, e.id).Scan(&refs)
		if refs > 0 {
			// keep orphaned step row so WIP boxes stay valid; clear from active path by bumping seq high
			_, _ = s.DB.Exec(`UPDATE pd_routing_step SET seq_no=seq_no+1000, step_name=COALESCE(step_name,'')||'(archived)' WHERE id=? AND seq_no<1000`, e.id)
			continue
		}
		_, _ = s.DB.Exec(`DELETE FROM pd_routing_step WHERE id=?`, e.id)
	}
	return ""
}

type compiledStep struct {
	Seq, ProcessID, WarehouseID, OutputProductID                 int64
	Code, Name                                                   string
	Piece, Checkpoint, CheckpointBind, AutoNext, StockIn, StockOut bool
}

func compileProcessSteps(doc *flowGraphDoc) ([]compiledStep, string) {
	byID := map[string]flowGraphNode{}
	var starts, ends int
	for _, n := range doc.Nodes {
		byID[n.ID] = n
		switch n.Type {
		case "start":
			starts++
		case "end":
			ends++
		}
	}
	if starts != 1 || ends != 1 {
		return nil, "GRAPH_START_END_REQUIRED"
	}
	outEdges := map[string][]flowGraphEdge{}
	for _, e := range doc.Edges {
		outEdges[e.Source] = append(outEdges[e.Source], e)
	}
	var startID string
	for _, n := range doc.Nodes {
		if n.Type == "start" {
			startID = n.ID
			break
		}
	}
	visited := map[string]bool{}
	pathSet := map[string]bool{}
	order := []string{}
	var walk func(id string) string
	walk = func(id string) string {
		if pathSet[id] {
			return "GRAPH_CYCLE"
		}
		if visited[id] {
			return ""
		}
		pathSet[id] = true
		visited[id] = true
		n := byID[id]
		if n.Type == "process_step" {
			order = append(order, id)
		}
		next := pickDefaultEdge(outEdges[id])
		if next != "" {
			if err := walk(next); err != "" {
				return err
			}
		}
		pathSet[id] = false
		return ""
	}
	if err := walk(startID); err != "" {
		return nil, err
	}
	steps := []compiledStep{}
	for i, nid := range order {
		n := byID[nid]
		d := n.Data
		if d == nil {
			d = map[string]interface{}{}
		}
		pid := asInt64Or0(d["process_id"])
		if pid <= 0 {
			return nil, "PROCESS_ID_REQUIRED"
		}
		asi := asBool(d["auto_stock_in"])
		aso := asBool(d["auto_stock_out"])
		wh := asInt64Or0(d["warehouse_id"])
		if (asi || aso) && wh <= 0 {
			return nil, "WAREHOUSE_REQUIRED"
		}
		code := strOr(d["step_code"])
		if code == "" {
			code = fmt.Sprintf("S%d", i+1)
		}
		name := strOrDef(d["step_name"], strOrDef(d["label"], code))
		an := true
		if _, ok := d["auto_next"]; ok {
			an = asBool(d["auto_next"])
		}
		steps = append(steps, compiledStep{
			Seq: int64(i + 1), ProcessID: pid, WarehouseID: wh, OutputProductID: asInt64Or0(d["output_product_id"]),
			Code: code, Name: name,
			Piece: asBool(d["is_piecework"]), Checkpoint: asBool(d["is_inbound_checkpoint"]),
			CheckpointBind: asBool(d["checkpoint_bind_warehouse"]),
			AutoNext: an, StockIn: asi, StockOut: aso,
		})
	}
	return steps, ""
}

func pickDefaultEdge(edges []flowGraphEdge) string {
	if len(edges) == 0 {
		return ""
	}
	for _, e := range edges {
		if e.Data != nil && asBool(e.Data["is_default"]) {
			return e.Target
		}
	}
	return edges[0].Target
}

func asBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}

func (s *Services) activeFlowGraphJSON(kind string) string {
	var gjson string
	_ = s.DB.QueryRow(`SELECT graph_json FROM pd_flow_graph WHERE kind=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, kind).Scan(&gjson)
	return gjson
}

func purchaseFlowKind(receiveKind string) string {
	if receiveKind == "stockin" {
		return "purchase_stockin"
	}
	return "purchase_gate"
}

func (s *Services) nextRoleAfterAction(receiveKind, fromAction, nextRole, nextNodeID string) (roleCode, action string) {
	kind := purchaseFlowKind(receiveKind)
	gjson := s.activeFlowGraphJSON(kind)
	if gjson == "" {
		return defaultNextRole(fromAction, receiveKind), ""
	}
	doc, err := parseFlowGraph(gjson)
	if err != nil {
		return defaultNextRole(fromAction, receiveKind), ""
	}
	byID := map[string]flowGraphNode{}
	for _, n := range doc.Nodes {
		byID[n.ID] = n
	}
	outEdges := map[string][]flowGraphEdge{}
	for _, e := range doc.Edges {
		outEdges[e.Source] = append(outEdges[e.Source], e)
	}
	if nextNodeID != "" {
		if n, ok := byID[nextNodeID]; ok && n.Type == "role_task" {
			d := n.Data
			if d == nil {
				d = map[string]interface{}{}
			}
			return strOr(d["role_code"]), strOr(d["action"])
		}
	}
	if nextRole != "" {
		return nextRole, ""
	}
	var fromID string
	for _, n := range doc.Nodes {
		if n.Type != "role_task" {
			continue
		}
		d := n.Data
		if d == nil {
			continue
		}
		act := strOr(d["action"])
		if act == fromAction || (fromAction == "submit" && (act == "submit" || act == "purchase_submit")) {
			fromID = n.ID
			break
		}
		if fromAction == "qc_deduct" && (act == "qc_deduct" || act == "qc") {
			fromID = n.ID
			break
		}
	}
	if fromID == "" {
		return defaultNextRole(fromAction, receiveKind), ""
	}
	next := pickDefaultEdge(outEdges[fromID])
	for next != "" {
		n, ok := byID[next]
		if !ok {
			break
		}
		if n.Type == "role_task" {
			d := n.Data
			if d == nil {
				d = map[string]interface{}{}
			}
			return strOr(d["role_code"]), strOr(d["action"])
		}
		if n.Type == "end" {
			return "", "end"
		}
		next = pickDefaultEdge(outEdges[n.ID])
	}
	return defaultNextRole(fromAction, receiveKind), ""
}

func defaultNextRole(fromAction, receiveKind string) string {
	switch fromAction {
	case "submit":
		// 入厂/入库均默认采购提交后直达仓管（质检非必经）
		return "warehouse"
	case "qc_deduct", "qc":
		return "warehouse"
	case "warehouse_confirm":
		return "finance"
	default:
		return ""
	}
}

func (s *Services) firstUserIDByRoleCode(roleCode string) int64 {
	if roleCode == "" {
		return 0
	}
	var uid int64
	_ = s.DB.QueryRow(`SELECT u.id FROM iam_user u
		JOIN iam_user_role ur ON ur.user_id=u.id
		JOIN iam_role r ON r.id=ur.role_id
		WHERE r.code=? AND COALESCE(u.is_deleted,0)=0 AND COALESCE(u.status,'active')='active'
		ORDER BY u.id LIMIT 1`, roleCode).Scan(&uid)
	return uid
}

func (s *Services) spawnWeighCollabTicketWithRole(c *gin.Context, kind string, weighID int64, docNo string, payload map[string]interface{}, preferredRole string, assigneeUserID int64) int64 {
	EnsureTicketSchema(s.DB)
	code := "farm_inbound"
	name := "过磅入厂"
	if kind == "stockin" {
		code = "stock_inbound"
		name = "过磅入库"
	}
	catID := s.categoryIDByCode(code)
	if catID <= 0 {
		return 0
	}
	cl := middleware.Claims(c)
	applicant := int64(0)
	if cl != nil {
		applicant = cl.UserID
	}
	if applicant <= 0 {
		return 0
	}
	assignee := int64(0)
	if assigneeUserID > 0 && s.assigneeInPool(catID, assigneeUserID) {
		assignee = assigneeUserID
	}
	if assignee <= 0 {
		assignee = s.firstUserIDByRoleCode(preferredRole)
	}
	if assignee <= 0 || assignee == applicant {
		assignee = s.firstPoolUserExcluding(catID, applicant)
	}
	if assignee <= 0 {
		assignee = s.firstUserIDByRoleCode(preferredRole)
		if assignee == applicant {
			assignee = 0
		}
	}
	if assignee <= 0 || assignee == applicant {
		return 0
	}
	title := fmt.Sprintf("%s · %s", name, docNo)
	if batch := strOr(payload["batch_no"]); batch != "" {
		title = fmt.Sprintf("%s · %s · %s", name, batch, docNo)
	}
	if preferredRole != "" {
		payload["flow_next_role"] = preferredRole
	}
	if assigneeUserID > 0 {
		payload["flow_next_assignee_user_id"] = assignee
	}
	payloadB, _ := json.Marshal(payload)
	tid, _, err := s.createTicket(c, catID, title, applicant, assignee, "weigh_ticket", weighID, string(payloadB), "")
	if err != nil {
		return 0
	}
	return tid
}

func roleDisplayName(code string) string {
	switch code {
	case "qc":
		return "质检"
	case "warehouse", "仓管", "仓管员":
		return "仓管"
	case "purchase", "采购":
		return "采购"
	case "finance", "财务":
		return "财务"
	case "sys_admin":
		return "系统管理员"
	default:
		if code == "" {
			return "未指定"
		}
		return code
	}
}

func (s *Services) listUsersByRoleCode(roleCode string) []gin.H {
	out := []gin.H{}
	if roleCode == "" {
		return out
	}
	rows, err := s.DB.Query(`SELECT u.id, u.login_name, COALESCE(e.name,'') FROM iam_user u
		JOIN iam_user_role ur ON ur.user_id=u.id
		JOIN iam_role r ON r.id=ur.role_id
		LEFT JOIN hr_employee e ON e.id=u.employee_id
		WHERE r.code=? AND COALESCE(u.is_deleted,0)=0 AND COALESCE(u.status,'active')='active'
		ORDER BY u.id`, roleCode)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var uid int64
		var login, name string
		_ = rows.Scan(&uid, &login, &name)
		disp := name
		if disp == "" {
			disp = login
		}
		out = append(out, gin.H{"user_id": uid, "login_name": login, "name": disp})
	}
	return out
}

func (s *Services) userHasRoleCode(userID int64, roleCode string) bool {
	if userID <= 0 || roleCode == "" {
		return false
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user_role ur
		JOIN iam_role r ON r.id=ur.role_id
		WHERE ur.user_id=? AND (r.code=? OR r.name=?)`, userID, roleCode, roleCode).Scan(&n)
	return n > 0
}

// handlePurchaseRoleUsers GET /purchase/role-users?role=purchase
func (s *Services) handlePurchaseRoleUsers(c *gin.Context) bool {
	if !s.requireAnyRole(c, "warehouse", "purchase") {
		return true
	}
	role := strings.TrimSpace(c.Query("role"))
	if role == "" {
		role = "purchase"
	}
	list := s.listUsersByRoleCode(role)
	if role == "purchase" && len(list) == 0 {
		// 兼容中文角色码
		for _, alt := range []string{"采购", "采购员"} {
			list = append(list, s.listUsersByRoleCode(alt)...)
		}
	}
	if role == "warehouse" && len(list) == 0 {
		for _, alt := range []string{"仓管", "仓管员"} {
			list = append(list, s.listUsersByRoleCode(alt)...)
		}
	}
	api.OK(c, gin.H{"list": list, "role": role})
	return true
}

// listNextRoleOptionsAfterAction returns selectable next role_task nodes after fromAction.
func (s *Services) listNextRoleOptionsAfterAction(receiveKind, fromAction string) []gin.H {
	kind := purchaseFlowKind(receiveKind)
	fallback := []string{"qc", "warehouse", "purchase"}
	if fromAction == "warehouse_confirm" {
		fallback = []string{"finance"}
	}
	gjson := s.activeFlowGraphJSON(kind)
	if gjson == "" {
		out := []gin.H{}
		for _, rc := range fallback {
			out = append(out, gin.H{
				"role_code": rc, "role_name": roleDisplayName(rc), "node_id": "", "action": "",
				"users": s.listUsersByRoleCode(rc),
			})
		}
		return out
	}
	doc, err := parseFlowGraph(gjson)
	if err != nil {
		out := []gin.H{}
		for _, rc := range fallback {
			out = append(out, gin.H{
				"role_code": rc, "role_name": roleDisplayName(rc), "node_id": "", "action": "",
				"users": s.listUsersByRoleCode(rc),
			})
		}
		return out
	}
	byID := map[string]flowGraphNode{}
	for _, n := range doc.Nodes {
		byID[n.ID] = n
	}
	outEdges := map[string][]flowGraphEdge{}
	for _, e := range doc.Edges {
		outEdges[e.Source] = append(outEdges[e.Source], e)
	}
	var fromID string
	for _, n := range doc.Nodes {
		if n.Type != "role_task" {
			continue
		}
		d := n.Data
		if d == nil {
			continue
		}
		act := strOr(d["action"])
		if act == fromAction || (fromAction == "submit" && (act == "submit" || act == "purchase_submit")) {
			fromID = n.ID
			break
		}
		if fromAction == "qc_deduct" && (act == "qc_deduct" || act == "qc") {
			fromID = n.ID
			break
		}
	}
	seenRole := map[string]bool{}
	out := []gin.H{}
	addRoleNode := func(n flowGraphNode) {
		d := n.Data
		if d == nil {
			d = map[string]interface{}{}
		}
		rc := strOr(d["role_code"])
		if rc == "" || seenRole[rc+":"+n.ID] {
			return
		}
		seenRole[rc+":"+n.ID] = true
		label := strOr(d["label"])
		if label == "" {
			label = roleDisplayName(rc)
		}
		out = append(out, gin.H{
			"role_code": rc, "role_name": label, "node_id": n.ID, "action": strOr(d["action"]),
			"users": s.listUsersByRoleCode(rc),
		})
	}
	if fromID != "" {
		for _, e := range outEdges[fromID] {
			n, ok := byID[e.Target]
			if !ok {
				continue
			}
			// walk through non-role nodes once
			for n.Type != "role_task" && n.Type != "end" {
				next := pickDefaultEdge(outEdges[n.ID])
				if next == "" {
					break
				}
				n, ok = byID[next]
				if !ok {
					break
				}
			}
			if n.Type == "role_task" {
				addRoleNode(n)
			}
		}
	}
	if len(out) == 0 {
		for _, rc := range fallback {
			out = append(out, gin.H{
				"role_code": rc, "role_name": roleDisplayName(rc), "node_id": "", "action": "",
				"users": s.listUsersByRoleCode(rc),
			})
		}
	}
	return out
}

func (s *Services) handleWeighFlowNextOptions(c *gin.Context) bool {
	kind := strings.ToLower(strings.TrimSpace(c.Query("receive_kind")))
	if kind == "" {
		kind = "gate"
	}
	from := strings.TrimSpace(c.Query("from_action"))
	if from == "" {
		from = "submit"
	}
	catCode := "farm_inbound"
	if kind == "stockin" {
		catCode = "stock_inbound"
	}
	catID := s.categoryIDByCode(catCode)
	options := s.listNextRoleOptionsAfterAction(kind, from)
	api.OK(c, gin.H{
		"receive_kind":  kind,
		"from_action":   from,
		"category_code": catCode,
		"category_id":   catID,
		"options":       options,
		"pool":          s.resolveHandlerPool(catID),
	})
	return true
}

func (s *Services) advanceWeighTicketAssignee(c *gin.Context, weighID int64, fromAction, nextRole, nextNodeID string) {
	body := map[string]interface{}{}
	if nextRole != "" {
		body["next_role_code"] = nextRole
	}
	_ = nextNodeID
	_ = s.handoffWeighOpenTicket(c, weighID, fromAction, body)
}

// handoffWeighOpenTicket assigns open weigh collab ticket to next handler (explicit user or role).
// Returns error code when next handler cannot be resolved.
func (s *Services) handoffWeighOpenTicket(c *gin.Context, weighID int64, fromAction string, body map[string]interface{}) string {
	var tid, catID int64
	var docNo string
	_ = s.DB.QueryRow(`SELECT t.id, COALESCE(t.category_id,0), COALESCE(w.doc_no,'')
		FROM wf_ticket t JOIN pur_weigh_ticket w ON w.id=t.biz_id
		WHERE t.biz_type='weigh_ticket' AND t.biz_id=? AND t.status IN ('open','in_progress') ORDER BY t.id DESC LIMIT 1`, weighID).
		Scan(&tid, &catID, &docNo)
	if tid <= 0 {
		return ""
	}
	nextUID, errCode := s.resolveNextHandoffUser(catID, body)
	if errCode != "" {
		role := strOr(body["next_role_code"])
		if role == "" {
			role = strOr(body["next_role"])
		}
		var kind string
		_ = s.DB.QueryRow(`SELECT COALESCE(receive_kind,'gate') FROM pur_weigh_ticket WHERE id=?`, weighID).Scan(&kind)
		if role == "" {
			role, _ = s.nextRoleAfterAction(kind, fromAction, role, strOr(body["next_node_id"]))
		}
		if role != "" {
			body2 := map[string]interface{}{"next_role_code": role}
			nextUID, errCode = s.resolveNextHandoffUser(catID, body2)
		}
	}
	if errCode != "" || nextUID <= 0 {
		return "NEXT_HANDLER_REQUIRED"
	}
	_, _ = s.DB.Exec(`UPDATE wf_ticket SET current_assignee_user_id=?, status='in_progress', updated_at=NOW() WHERE id=?`, nextUID, tid)
	cl := middleware.Claims(c)
	from := int64(0)
	if cl != nil {
		from = cl.UserID
	}
	s.appendTicketLog(tid, "assign", from, nextUID, "flow:"+fromAction)
	s.notifyTicketAssignee(c, tid, "workflow.ticket.assigned", docNo+" → next", nextUID, from)
	return ""
}
