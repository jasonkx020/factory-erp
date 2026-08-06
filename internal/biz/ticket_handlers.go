package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/notify"
	"erp/internal/persistence/sqlutil"
)

// EnsureTicketSchema creates workflow ticket tables and seeds tool categories.
func EnsureTicketSchema(db *sql.DB) {
	if db == nil {
		return
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS wf_ticket_category (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS wf_ticket_category_handler (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  category_id INTEGER NOT NULL,
  handler_type TEXT NOT NULL,
  handler_ref INTEGER NOT NULL,
  UNIQUE(category_id, handler_type, handler_ref)
)`,
		`CREATE TABLE IF NOT EXISTS wf_ticket (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  category_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  applicant_user_id INTEGER NOT NULL,
  current_assignee_user_id INTEGER,
  biz_type TEXT,
  biz_id INTEGER,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  closed_at TEXT
)`,
		`CREATE TABLE IF NOT EXISTS wf_ticket_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  from_user_id INTEGER,
  to_user_id INTEGER,
  comment TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`ALTER TABLE hr_tool_issue ADD COLUMN pending_return_qty REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE hr_tool_issue ADD COLUMN ticket_id INTEGER`,
		`ALTER TABLE wf_ticket_category ADD COLUMN form_schema_json TEXT`,
		`ALTER TABLE wf_ticket_category ADD COLUMN biz_hint TEXT`,
	}
	for _, q := range stmts {
		_, _ = db.Exec(q)
	}
	seedTicketCategories(db)
}

func seedTicketCategories(db *sql.DB) {
	for _, seed := range defaultTicketCategorySeeds() {
		schema := marshalFormSchema(seed.Fields)
		_, _ = db.Exec(`INSERT OR IGNORE INTO wf_ticket_category(code, name, remark, form_schema_json, biz_hint) VALUES(?,?,?,?,?)`,
			seed.Code, seed.Name, seed.Remark, schema, seed.BizHint)
		// backfill schema if empty on existing row
		_, _ = db.Exec(`UPDATE wf_ticket_category SET form_schema_json=?, biz_hint=COALESCE(NULLIF(biz_hint,''), ?), name=?, remark=?
			WHERE code=? AND (form_schema_json IS NULL OR form_schema_json='' OR form_schema_json='[]' OR form_schema_json='null')`,
			schema, seed.BizHint, seed.Name, seed.Remark, seed.Code)
		var catID int64
		_ = db.QueryRow(`SELECT id FROM wf_ticket_category WHERE code=?`, seed.Code).Scan(&catID)
		if catID <= 0 {
			continue
		}
		for _, roleCode := range []string{"hr", "sys_admin"} {
			var roleID int64
			_ = db.QueryRow(`SELECT id FROM iam_role WHERE code=? LIMIT 1`, roleCode).Scan(&roleID)
			if roleID > 0 {
				_, _ = db.Exec(`INSERT OR IGNORE INTO wf_ticket_category_handler(category_id, handler_type, handler_ref) VALUES(?,'role',?)`, catID, roleID)
			}
		}
	}
}

func (s *Services) handleTicketDomain(c *gin.Context, method, openapiPath, action string) bool {
	EnsureTicketSchema(s.DB)
	switch {
	case strings.Contains(openapiPath, "/workflow/ticket-handler-pool"):
		return s.handleTicketCategoryPool(c)
	case strings.Contains(openapiPath, "/workflow/ticket-categories") && strings.Contains(openapiPath, "/handlers"):
		return s.handleTicketCategoryHandlers(c, method)
	case strings.HasPrefix(openapiPath, "/api/v1/workflow/ticket-categories"):
		return s.handleTicketCategories(c, method, action)
	case strings.Contains(openapiPath, "/workflow/tickets/") && strings.HasSuffix(openapiPath, "/assign"):
		return s.assignTicket(c)
	case strings.Contains(openapiPath, "/workflow/tickets/") && strings.HasSuffix(openapiPath, "/action"):
		return s.actionTicket(c)
	case strings.Contains(openapiPath, "/workflow/tickets/") && strings.HasSuffix(openapiPath, "/handlers-pool"):
		return s.ticketHandlersPoolByTicket(c)
	case strings.HasPrefix(openapiPath, "/api/v1/workflow/tickets"):
		return s.handleTickets(c, method, action)
	}
	return false
}

func (s *Services) handleTicketCategories(c *gin.Context, method, action string) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	needEdit := action == "create" || action == "update" || method == "PUT" || method == "POST"
	if needEdit && !claimsIsSysAdmin(cl.Roles, cl.Permissions) && !claimsHasCode(cl.Permissions, "系统管理:工单中心:编辑") {
		api.FailJSON(c, "PERM_DENIED")
		return true
	}
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		rows, err := s.DB.Query(`SELECT id, code, name, COALESCE(enabled,1), COALESCE(remark,''), created_at,
			COALESCE(form_schema_json,'[]'), COALESCE(biz_hint,'') FROM wf_ticket_category ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, remark, created, schema, hint string
			var enabled int
			_ = rows.Scan(&id, &code, &name, &enabled, &remark, &created, &schema, &hint)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "enabled": enabled == 1, "remark": remark, "created_at": created,
				"form_schema": parseFormSchema(schema), "form_schema_json": schema, "biz_hint": hint,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case action == "get":
		id := paramID(c)
		var code, name, remark, created, schema, hint string
		var enabled int
		err := s.DB.QueryRow(`SELECT code, name, COALESCE(enabled,1), COALESCE(remark,''), created_at,
			COALESCE(form_schema_json,'[]'), COALESCE(biz_hint,'') FROM wf_ticket_category WHERE id=?`, id).
			Scan(&code, &name, &enabled, &remark, &created, &schema, &hint)
		if err != nil {
			api.FailJSON(c, "NOT_FOUND")
			return true
		}
		api.OK(c, gin.H{
			"id": id, "code": code, "name": name, "enabled": enabled == 1, "remark": remark, "created_at": created,
			"form_schema": parseFormSchema(schema), "form_schema_json": schema, "biz_hint": hint,
			"handlers": s.listCategoryHandlers(id), "pool": s.resolveHandlerPool(id),
		})
		return true
	case action == "create":
		body := bindBody(c)
		code := strings.TrimSpace(strOr(body["code"]))
		name := strings.TrimSpace(strOr(body["name"]))
		if code == "" || name == "" {
			api.FailJSON(c, "CODE_NAME_REQUIRED")
			return true
		}
		schema := normalizeFormSchemaJSON(body["form_schema"])
		if schema == "[]" {
			schema = normalizeFormSchemaJSON(body["form_schema_json"])
		}
		res, err := s.DB.Exec(`INSERT INTO wf_ticket_category(code, name, enabled, remark, form_schema_json, biz_hint) VALUES(?,?,?,?,?,?)`,
			code, name, boolToInt(body["enabled"] == nil || body["enabled"] == true), strOr(body["remark"]), schema, strOr(body["biz_hint"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "form_schema": parseFormSchema(schema)})
		return true
	case action == "update":
		id := paramID(c)
		body := bindBody(c)
		name := strOr(body["name"])
		remark := strOr(body["remark"])
		hint := strOr(body["biz_hint"])
		enabled := 1
		if v, ok := body["enabled"].(bool); ok && !v {
			enabled = 0
		}
		schema := ""
		if body["form_schema"] != nil {
			schema = normalizeFormSchemaJSON(body["form_schema"])
		} else if body["form_schema_json"] != nil {
			schema = normalizeFormSchemaJSON(body["form_schema_json"])
		}
		if schema != "" {
			_, err := s.DB.Exec(`UPDATE wf_ticket_category SET name=COALESCE(NULLIF(?,''),name), remark=?, enabled=?, form_schema_json=?, biz_hint=COALESCE(NULLIF(?,''),biz_hint) WHERE id=?`,
				name, remark, enabled, schema, hint, id)
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		} else {
			_, err := s.DB.Exec(`UPDATE wf_ticket_category SET name=COALESCE(NULLIF(?,''),name), remark=?, enabled=?, biz_hint=COALESCE(NULLIF(?,''),biz_hint) WHERE id=?`,
				name, remark, enabled, hint, id)
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		}
		api.OK(c, s.loadCategory(id))
		return true
	}
	return true
}

func (s *Services) loadCategory(id int64) gin.H {
	var code, name, remark, created, schema, hint string
	var enabled int
	err := s.DB.QueryRow(`SELECT code, name, COALESCE(enabled,1), COALESCE(remark,''), created_at,
		COALESCE(form_schema_json,'[]'), COALESCE(biz_hint,'') FROM wf_ticket_category WHERE id=?`, id).
		Scan(&code, &name, &enabled, &remark, &created, &schema, &hint)
	if err != nil {
		return gin.H{"id": id}
	}
	return gin.H{
		"id": id, "code": code, "name": name, "enabled": enabled == 1, "remark": remark, "created_at": created,
		"form_schema": parseFormSchema(schema), "form_schema_json": schema, "biz_hint": hint,
	}
}

func (s *Services) handleTicketCategoryHandlers(c *gin.Context, method string) bool {
	id := paramID(c)
	if method == "GET" {
		api.OK(c, gin.H{"category_id": id, "handlers": s.listCategoryHandlers(id), "pool": s.resolveHandlerPool(id)})
		return true
	}
	if method != "PUT" && method != "POST" {
		api.FailJSON(c, "METHOD_NOT_ALLOWED")
		return true
	}
	cl := middleware.Claims(c)
	if cl == nil || (!claimsIsSysAdmin(cl.Roles, cl.Permissions) && !claimsHasCode(cl.Permissions, "系统管理:工单中心:编辑")) {
		api.FailJSON(c, "PERM_DENIED")
		return true
	}
	body := bindBody(c)
	raw, _ := body["handlers"].([]interface{})
	_, _ = s.DB.Exec(`DELETE FROM wf_ticket_category_handler WHERE category_id=?`, id)
	for _, x := range raw {
		m, _ := x.(map[string]interface{})
		if m == nil {
			continue
		}
		ht := strings.ToLower(strOr(m["handler_type"]))
		ref, _ := asInt64(m["handler_ref"])
		if (ht != "user" && ht != "role") || ref <= 0 {
			continue
		}
		_, _ = s.DB.Exec(`INSERT OR IGNORE INTO wf_ticket_category_handler(category_id, handler_type, handler_ref) VALUES(?,?,?)`, id, ht, ref)
	}
	api.OK(c, gin.H{"category_id": id, "handlers": s.listCategoryHandlers(id), "pool": s.resolveHandlerPool(id)})
	return true
}

func (s *Services) listCategoryHandlers(catID int64) []gin.H {
	rows, err := s.DB.Query(`SELECT id, handler_type, handler_ref FROM wf_ticket_category_handler WHERE category_id=?`, catID)
	out := []gin.H{}
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, ref int64
		var ht string
		_ = rows.Scan(&id, &ht, &ref)
		label := ""
		if ht == "user" {
			_ = s.DB.QueryRow(`SELECT login_name FROM iam_user WHERE id=?`, ref).Scan(&label)
		} else {
			_ = s.DB.QueryRow(`SELECT code FROM iam_role WHERE id=?`, ref).Scan(&label)
		}
		out = append(out, gin.H{"id": id, "handler_type": ht, "handler_ref": ref, "label": label})
	}
	return out
}

// resolveHandlerPool expands role handlers to users + direct users.
func (s *Services) resolveHandlerPool(catID int64) []gin.H {
	seen := map[int64]bool{}
	out := []gin.H{}
	rows, err := s.DB.Query(`SELECT handler_type, handler_ref FROM wf_ticket_category_handler WHERE category_id=?`, catID)
	if err != nil {
		return out
	}
	defer rows.Close()
	type pair struct{ ht string; ref int64 }
	pairs := []pair{}
	for rows.Next() {
		var ht string
		var ref int64
		_ = rows.Scan(&ht, &ref)
		pairs = append(pairs, pair{ht, ref})
	}
	addUser := func(uid int64) {
		if uid <= 0 || seen[uid] {
			return
		}
		var login, name string
		_ = s.DB.QueryRow(`SELECT u.login_name, COALESCE(e.name,'') FROM iam_user u
			LEFT JOIN hr_employee e ON e.id=u.employee_id WHERE u.id=? AND COALESCE(u.is_deleted,0)=0 AND COALESCE(u.status,'active')='active'`, uid).
			Scan(&login, &name)
		if login == "" {
			return
		}
		seen[uid] = true
		disp := name
		if disp == "" {
			disp = login
		}
		out = append(out, gin.H{"user_id": uid, "login_name": login, "name": disp})
	}
	for _, p := range pairs {
		if p.ht == "user" {
			addUser(p.ref)
			continue
		}
		rrows, err := s.DB.Query(`SELECT u.id FROM iam_user u
			JOIN iam_user_role ur ON ur.user_id=u.id WHERE ur.role_id=? AND COALESCE(u.status,'active')='active' AND COALESCE(u.is_deleted,0)=0`, p.ref)
		if err != nil {
			continue
		}
		for rrows.Next() {
			var uid int64
			_ = rrows.Scan(&uid)
			addUser(uid)
		}
		rrows.Close()
	}
	return out
}

func (s *Services) categoryIDByCode(code string) int64 {
	var id int64
	_ = s.DB.QueryRow(`SELECT id FROM wf_ticket_category WHERE code=? AND COALESCE(enabled,1)=1`, code).Scan(&id)
	return id
}

func (s *Services) assigneeInPool(catID, userID int64) bool {
	for _, p := range s.resolveHandlerPool(catID) {
		if id, _ := asInt64(p["user_id"]); id == userID {
			return true
		}
	}
	return false
}

func (s *Services) handleTickets(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || (method == "GET" && action != "get"):
		return s.listTickets(c)
	case action == "get":
		return s.getTicket(c)
	case action == "create":
		return s.createTicketHTTP(c)
	}
	return true
}

func (s *Services) listTickets(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	pageNum, pageSize := sqlutil.Page(c)
	where := `WHERE 1=1`
	args := []interface{}{}
	if st := c.Query("status"); st != "" {
		where += ` AND t.status=?`
		args = append(args, st)
	}
	if cat := c.Query("category_id"); cat != "" {
		where += ` AND t.category_id=?`
		args = append(args, cat)
	}
	if code := c.Query("category_code"); code != "" {
		where += ` AND c.code=?`
		args = append(args, code)
	}
	scope := c.Query("scope")
	admin := claimsIsSysAdmin(cl.Roles, cl.Permissions)
	switch scope {
	case "mine_applicant":
		where += ` AND t.applicant_user_id=?`
		args = append(args, cl.UserID)
	case "mine_assignee":
		where += ` AND t.current_assignee_user_id=? AND t.status IN ('open','in_progress')`
		args = append(args, cl.UserID)
	default:
		if !admin {
			where += ` AND (t.applicant_user_id=? OR t.current_assignee_user_id=?)`
			args = append(args, cl.UserID, cl.UserID)
		}
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM wf_ticket t LEFT JOIN wf_ticket_category c ON c.id=t.category_id `+where, args...).Scan(&total)
	args2 := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT t.id, t.doc_no, t.category_id, COALESCE(c.code,''), COALESCE(c.name,''), t.title, t.status,
		t.applicant_user_id, COALESCE(t.current_assignee_user_id,0), COALESCE(t.biz_type,''), COALESCE(t.biz_id,0),
		COALESCE(t.payload_json,'{}'), t.created_at, t.updated_at, COALESCE(t.closed_at,'')
		FROM wf_ticket t LEFT JOIN wf_ticket_category c ON c.id=t.category_id `+where+`
		ORDER BY t.id DESC LIMIT ? OFFSET ?`, args2...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, catID, applicant, assignee, bizID int64
		var docNo, catCode, catName, title, status, bizType, payload, created, updated, closed string
		_ = rows.Scan(&id, &docNo, &catID, &catCode, &catName, &title, &status, &applicant, &assignee, &bizType, &bizID, &payload, &created, &updated, &closed)
		list = append(list, gin.H{
			"id": id, "doc_no": docNo, "category_id": catID, "category_code": catCode, "category_name": catName,
			"title": title, "status": status, "applicant_user_id": applicant, "current_assignee_user_id": assignee,
			"applicant_name": s.userDisplayName(applicant), "assignee_name": s.userDisplayName(assignee),
			"biz_type": bizType, "biz_id": bizID, "payload_json": payload, "created_at": created, "updated_at": updated, "closed_at": closed,
		})
	}
	api.PageOK(c, list, total, pageNum, pageSize)
	return true
}

func (s *Services) userDisplayName(uid int64) string {
	if uid <= 0 {
		return ""
	}
	var login, name string
	_ = s.DB.QueryRow(`SELECT u.login_name, COALESCE(e.name,'') FROM iam_user u LEFT JOIN hr_employee e ON e.id=u.employee_id WHERE u.id=?`, uid).
		Scan(&login, &name)
	if name != "" {
		return name
	}
	return login
}

func (s *Services) getTicket(c *gin.Context) bool {
	id := paramID(c)
	d := s.loadTicket(id)
	if d == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	d["logs"] = s.listTicketLogs(id)
	d["pool"] = s.resolveHandlerPool(asInt64Or0(d["category_id"]))
	api.OK(c, d)
	return true
}

func (s *Services) loadTicket(id int64) gin.H {
	var catID, applicant, assignee, bizID int64
	var docNo, catCode, catName, title, status, bizType, payload, created, updated, closed string
	err := s.DB.QueryRow(`SELECT t.doc_no, t.category_id, COALESCE(c.code,''), COALESCE(c.name,''), t.title, t.status,
		t.applicant_user_id, COALESCE(t.current_assignee_user_id,0), COALESCE(t.biz_type,''), COALESCE(t.biz_id,0),
		COALESCE(t.payload_json,'{}'), t.created_at, t.updated_at, COALESCE(t.closed_at,'')
		FROM wf_ticket t LEFT JOIN wf_ticket_category c ON c.id=t.category_id WHERE t.id=?`, id).
		Scan(&docNo, &catID, &catCode, &catName, &title, &status, &applicant, &assignee, &bizType, &bizID, &payload, &created, &updated, &closed)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "doc_no": docNo, "category_id": catID, "category_code": catCode, "category_name": catName,
		"title": title, "status": status, "applicant_user_id": applicant, "current_assignee_user_id": assignee,
		"applicant_name": s.userDisplayName(applicant), "assignee_name": s.userDisplayName(assignee),
		"biz_type": bizType, "biz_id": bizID, "payload_json": payload, "payload": parsePayloadMap(payload),
		"form_schema": s.categoryFormSchema(catID),
		"created_at": created, "updated_at": updated, "closed_at": closed,
	}
}

func parsePayloadMap(raw string) map[string]interface{} {
	m := map[string]interface{}{}
	if raw == "" {
		return m
	}
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func (s *Services) categoryFormSchema(catID int64) []FormFieldDef {
	var schema string
	_ = s.DB.QueryRow(`SELECT COALESCE(form_schema_json,'[]') FROM wf_ticket_category WHERE id=?`, catID).Scan(&schema)
	return parseFormSchema(schema)
}

func (s *Services) listTicketLogs(ticketID int64) []gin.H {
	rows, err := s.DB.Query(`SELECT id, action, COALESCE(from_user_id,0), COALESCE(to_user_id,0), COALESCE(comment,''), created_at
		FROM wf_ticket_log WHERE ticket_id=? ORDER BY id`, ticketID)
	out := []gin.H{}
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, from, to int64
		var action, comment, created string
		_ = rows.Scan(&id, &action, &from, &to, &comment, &created)
		out = append(out, gin.H{
			"id": id, "action": action, "from_user_id": from, "to_user_id": to, "comment": comment, "created_at": created,
			"from_name": s.userDisplayName(from), "to_name": s.userDisplayName(to),
		})
	}
	return out
}

func (s *Services) createTicketHTTP(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	body := bindBody(c)
	catID, _ := asInt64(body["category_id"])
	if catID == 0 {
		catID = s.categoryIDByCode(strOr(body["category_code"]))
	}
	next, _ := asInt64(body["next_assignee_user_id"])
	if catID == 0 || next <= 0 {
		api.FailJSON(c, "CATEGORY_ASSIGNEE_REQUIRED")
		return true
	}
	if !s.assigneeInPool(catID, next) {
		api.FailJSON(c, "ASSIGNEE_NOT_IN_POOL")
		return true
	}
	cat := s.loadCategory(catID)
	schema := parseFormSchema(strOr(cat["form_schema_json"]))
	payloadMap := map[string]interface{}{}
	if body["payload"] != nil {
		if m, ok := body["payload"].(map[string]interface{}); ok {
			payloadMap = m
		} else {
			b, _ := json.Marshal(body["payload"])
			_ = json.Unmarshal(b, &payloadMap)
		}
	} else if raw := strOr(body["payload_json"]); raw != "" {
		_ = json.Unmarshal([]byte(raw), &payloadMap)
	}
	if msg := validatePayloadAgainstSchema(schema, payloadMap); msg != "" {
		api.FailJSON(c, msg)
		return true
	}
	title := strings.TrimSpace(strOr(body["title"]))
	if title == "" {
		title = autoTitleFromPayload(strOr(cat["name"]), payloadMap)
	}
	bizType := strOr(body["biz_type"])
	bizID, _ := asInt64(body["biz_id"])
	payloadB, _ := json.Marshal(payloadMap)
	if len(payloadMap) == 0 {
		payloadB = []byte("{}")
	}
	id, _, err := s.createTicket(c, catID, title, cl.UserID, next, bizType, bizID, string(payloadB), strOr(body["comment"]))
	if err != nil {
		api.FailJSON(c, err.Error())
		return true
	}
	api.OK(c, s.loadTicket(id))
	return true
}

func (s *Services) createTicket(c *gin.Context, catID int64, title string, applicant, assignee int64, bizType string, bizID int64, payload, comment string) (int64, string, error) {
	docNo := fmt.Sprintf("TK%s%04d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%10000)
	res, err := s.DB.Exec(`INSERT INTO wf_ticket(doc_no, category_id, title, status, applicant_user_id, current_assignee_user_id, biz_type, biz_id, payload_json)
		VALUES(?,?,?,'open',?,?,?,?,?)`, docNo, catID, title, applicant, assignee, nullStr(bizType), nullIf0(bizID), payload)
	if err != nil {
		return 0, "", fmt.Errorf("DB_ERROR:%s", err.Error())
	}
	id, _ := res.LastInsertId()
	s.appendTicketLog(id, "create", applicant, assignee, comment)
	s.notifyTicketAssignee(c, id, "workflow.ticket.assigned", title, assignee, applicant)
	return id, docNo, nil
}

func (s *Services) appendTicketLog(ticketID int64, action string, from, to int64, comment string) {
	_, _ = s.DB.Exec(`INSERT INTO wf_ticket_log(ticket_id, action, from_user_id, to_user_id, comment) VALUES(?,?,?,?,?)`,
		ticketID, action, nullIf0(from), nullIf0(to), comment)
}

func (s *Services) assignTicket(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	id := paramID(c)
	body := bindBody(c)
	next, _ := asInt64(body["next_assignee_user_id"])
	t := s.loadTicket(id)
	if t == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	st := strOr(t["status"])
	if st == "done" || st == "rejected" || st == "cancelled" {
		api.FailJSON(c, "TICKET_CLOSED")
		return true
	}
	catID := asInt64Or0(t["category_id"])
	if next <= 0 || !s.assigneeInPool(catID, next) {
		api.FailJSON(c, "ASSIGNEE_NOT_IN_POOL")
		return true
	}
	cur := asInt64Or0(t["current_assignee_user_id"])
	admin := claimsIsSysAdmin(cl.Roles, cl.Permissions)
	if !admin && cur != cl.UserID && asInt64Or0(t["applicant_user_id"]) != cl.UserID {
		api.FailJSON(c, "PERM_DENIED")
		return true
	}
	_, _ = s.DB.Exec(`UPDATE wf_ticket SET current_assignee_user_id=?, status='in_progress', updated_at=datetime('now') WHERE id=?`, next, id)
	s.appendTicketLog(id, "assign", cl.UserID, next, strOr(body["comment"]))
	if s.Notify != nil {
		s.Notify.CompleteTask("wf_ticket", id)
	}
	s.notifyTicketAssignee(c, id, "workflow.ticket.assigned", strOr(t["title"]), next, cl.UserID)
	api.OK(c, s.loadTicket(id))
	return true
}

func (s *Services) actionTicket(c *gin.Context) bool {
	id := paramID(c)
	body := bindBody(c)
	if !s.actionTicketBody(c, id, body) {
		return true
	}
	api.OK(c, s.loadTicket(id))
	return true
}

// actionTicketBody mutates ticket; returns false if it already wrote an error response.
func (s *Services) actionTicketBody(c *gin.Context, id int64, body map[string]interface{}) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return false
	}
	action := strings.ToLower(strOr(body["action"]))
	t := s.loadTicket(id)
	if t == nil {
		api.FailJSON(c, "NOT_FOUND")
		return false
	}
	st := strOr(t["status"])
	if st == "done" || st == "rejected" || st == "cancelled" {
		api.FailJSON(c, "TICKET_CLOSED")
		return false
	}
	cur := asInt64Or0(t["current_assignee_user_id"])
	admin := claimsIsSysAdmin(cl.Roles, cl.Permissions)
	if !admin && cur != cl.UserID {
		api.FailJSON(c, "PERM_DENIED")
		return false
	}
	comment := strOr(body["comment"])
	next, _ := asInt64(body["next_assignee_user_id"])
	catID := asInt64Or0(t["category_id"])
	bizType := strOr(t["biz_type"])
	bizID := asInt64Or0(t["biz_id"])

	switch action {
	case "approve", "return_confirm", "close":
		if next > 0 {
			if !s.assigneeInPool(catID, next) {
				api.FailJSON(c, "ASSIGNEE_NOT_IN_POOL")
				return false
			}
			_, _ = s.DB.Exec(`UPDATE wf_ticket SET current_assignee_user_id=?, status='in_progress', updated_at=datetime('now') WHERE id=?`, next, id)
			s.appendTicketLog(id, action, cl.UserID, next, comment)
			if s.Notify != nil {
				s.Notify.CompleteTask("wf_ticket", id)
			}
			s.notifyTicketAssignee(c, id, "workflow.ticket.assigned", strOr(t["title"]), next, cl.UserID)
		} else {
			_, _ = s.DB.Exec(`UPDATE wf_ticket SET status='done', closed_at=datetime('now'), updated_at=datetime('now'), current_assignee_user_id=NULL WHERE id=?`, id)
			s.appendTicketLog(id, action, cl.UserID, 0, comment)
			if s.Notify != nil {
				s.Notify.CompleteTask("wf_ticket", id)
			}
			s.notifyTicketApplicant(c, id, "workflow.ticket.done", strOr(t["title"]), asInt64Or0(t["applicant_user_id"]))
		}
		if bizType == "hr_tool_issue" && bizID > 0 {
			s.applyToolBizFromTicketAction(c, action, bizID, body)
		}
	case "reject":
		_, _ = s.DB.Exec(`UPDATE wf_ticket SET status='rejected', closed_at=datetime('now'), updated_at=datetime('now'), current_assignee_user_id=NULL WHERE id=?`, id)
		s.appendTicketLog(id, "reject", cl.UserID, 0, comment)
		if s.Notify != nil {
			s.Notify.CompleteTask("wf_ticket", id)
		}
		s.notifyTicketApplicant(c, id, "workflow.ticket.rejected", strOr(t["title"]), asInt64Or0(t["applicant_user_id"]))
		if bizType == "hr_tool_issue" && bizID > 0 {
			var curSt string
			_ = s.DB.QueryRow(`SELECT status FROM hr_tool_issue WHERE id=?`, bizID).Scan(&curSt)
			if curSt == "pending_return" {
				_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='open', pending_return_qty=0 WHERE id=?`, bizID)
			} else {
				_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='rejected' WHERE id=? AND status='pending'`, bizID)
			}
		}
	case "comment":
		s.appendTicketLog(id, "comment", cl.UserID, 0, comment)
	default:
		api.FailJSON(c, "INVALID_ACTION")
		return false
	}
	return true
}

func (s *Services) applyToolBizFromTicketAction(c *gin.Context, action string, bizID int64, body map[string]interface{}) {
	switch action {
	case "approve":
		_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET status='open' WHERE id=? AND status='pending'`, bizID)
	case "return_confirm":
		var issue, pending float64
		_ = s.DB.QueryRow(`SELECT issue_qty, COALESCE(pending_return_qty,0) FROM hr_tool_issue WHERE id=?`, bizID).Scan(&issue, &pending)
		ret := asFloatOr0(body["return_qty"])
		if ret <= 0 {
			ret = pending
		}
		if ret <= 0 {
			ret = issue
		}
		var curRet float64
		_ = s.DB.QueryRow(`SELECT return_qty FROM hr_tool_issue WHERE id=?`, bizID).Scan(&curRet)
		newRet := curRet + ret
		if newRet > issue {
			newRet = issue
		}
		total := issue - newRet
		st := "open"
		if total <= 0 {
			total = 0
			st = "returned"
		}
		_, _ = s.DB.Exec(`UPDATE hr_tool_issue SET return_qty=?, total_qty=?, status=?, pending_return_qty=0 WHERE id=?`, newRet, total, st, bizID)
	}
}

func (s *Services) ticketHandlersPoolByTicket(c *gin.Context) bool {
	id := paramID(c)
	t := s.loadTicket(id)
	if t == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	api.OK(c, gin.H{"pool": s.resolveHandlerPool(asInt64Or0(t["category_id"]))})
	return true
}

func (s *Services) notifyTicketAssignee(c *gin.Context, ticketID int64, eventKey, title string, toUID, fromUID int64) {
	if s.Notify == nil || toUID <= 0 {
		return
	}
	t := s.loadTicket(ticketID)
	docNo := ""
	if t != nil {
		docNo = strOr(t["doc_no"])
	}
	s.Notify.NotifyNext(c, notify.Event{
		Key: eventKey, BizType: "wf_ticket", BizID: ticketID, DocNo: docNo,
		Title: "工单待处理", Body: title + " " + docNo, CreateTask: true, ToRoles: []string{"_user"},
		Payload: map[string]interface{}{
			"notify_user_ids": []int64{toUID},
			"ticket_id":       ticketID,
			"from_user_id":    fromUID,
			"admin_route":     "/workflow/tickets",
			"employee_route":  "/tickets",
		},
	})
	// ensure assignee on wf_task
	_, _ = s.DB.Exec(`UPDATE wf_task SET assignee_user_id=? WHERE biz_type='wf_ticket' AND biz_id=? AND status='pending'`, toUID, ticketID)
}

func (s *Services) notifyTicketApplicant(c *gin.Context, ticketID int64, eventKey, title string, applicant int64) {
	if s.Notify == nil || applicant <= 0 {
		return
	}
	t := s.loadTicket(ticketID)
	docNo := ""
	if t != nil {
		docNo = strOr(t["doc_no"])
	}
	label := "工单已办结"
	if strings.Contains(eventKey, "reject") {
		label = "工单已驳回"
	}
	s.Notify.NotifyNext(c, notify.Event{
		Key: eventKey, BizType: "wf_ticket", BizID: ticketID, DocNo: docNo,
		Title: label, Body: title + " " + docNo, CreateTask: false, ToRoles: []string{"_user"},
		Payload: map[string]interface{}{
			"notify_user_ids": []int64{applicant},
			"ticket_id":       ticketID,
			"admin_route":     "/workflow/tickets",
			"employee_route":  "/tickets",
		},
	})
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// PoolByCategoryCode returns handler pool for App/Admin pickers.
func (s *Services) poolByCategoryCodeHTTP(c *gin.Context, code string) bool {
	EnsureTicketSchema(s.DB)
	id := s.categoryIDByCode(code)
	if id == 0 {
		api.FailJSON(c, "CATEGORY_NOT_FOUND")
		return true
	}
	api.OK(c, gin.H{"category_id": id, "category_code": code, "pool": s.resolveHandlerPool(id)})
	return true
}

// helper for query category pool without ticket id
func (s *Services) handleTicketCategoryPool(c *gin.Context) bool {
	code := c.Query("category_code")
	if code == "" {
		idStr := c.Query("category_id")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if id > 0 {
			api.OK(c, gin.H{"category_id": id, "pool": s.resolveHandlerPool(id)})
			return true
		}
		api.FailJSON(c, "CATEGORY_REQUIRED")
		return true
	}
	return s.poolByCategoryCodeHTTP(c, code)
}
