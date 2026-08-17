package notify

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	erpmqtt "erp/internal/mqtt"
	"erp/internal/persistence/sqlutil"
)

func (s *Service) HandleAPI(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.Contains(openapiPath, "/notify/mqtt-connect") && method == "GET":
		return s.handleMqttConnect(c)
	case strings.Contains(openapiPath, "/notify/inbox") && strings.HasSuffix(openapiPath, "/read") && method == "POST":
		return s.handleInboxRead(c)
	case strings.HasPrefix(openapiPath, "/api/v1/notify/inbox") && method == "GET":
		return s.handleInboxList(c)
	case strings.Contains(openapiPath, "/workflow/tasks") && strings.HasSuffix(openapiPath, "/assign") && method == "POST":
		return s.handleTaskAssign(c)
	case strings.Contains(openapiPath, "/workflow/tasks") && strings.HasSuffix(openapiPath, "/claim") && method == "POST":
		return s.handleTaskClaim(c)
	case strings.HasPrefix(openapiPath, "/api/v1/workflow/tasks") && method == "GET":
		return s.handleTaskList(c)
	}
	return false
}

func (s *Service) handleMqttConnect(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	out, err := erpmqtt.IssueUserConnectInfo(s.Cfg, cl.UserID, cl.Roles)
	if err != nil {
		api.FailJSON(c, "MQTT_TOKEN_ERROR:"+err.Error())
		return true
	}
	api.OK(c, out)
	return true
}

func (s *Service) handleInboxList(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	pageNum, pageSize := sqlutil.Page(c)
	unreadOnly := c.Query("unread") == "1"
	where := `WHERE user_id=?`
	args := []interface{}{cl.UserID}
	if unreadOnly {
		where += ` AND COALESCE(read_at,'')=''`
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM notify_inbox `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, title, COALESCE(body,''), COALESCE(event_key,''), COALESCE(task_id,0),
		COALESCE(payload_json,'{}'), COALESCE(read_at,''), created_at FROM notify_inbox `+where+`
		ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, taskID int64
		var title, body, ek, pj, readAt, created string
		_ = rows.Scan(&id, &title, &body, &ek, &taskID, &pj, &readAt, &created)
		list = append(list, gin.H{
			"id": id, "title": title, "body": body, "event_key": ek, "task_id": taskID,
			"payload_json": pj, "read_at": readAt, "created_at": created,
		})
	}
	var unread int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM notify_inbox WHERE user_id=? AND COALESCE(read_at,'')=''`, cl.UserID).Scan(&unread)
	api.OK(c, gin.H{"list": list, "total": total, "page_num": pageNum, "page_size": pageSize, "unread": unread})
	return true
}

func (s *Service) handleInboxRead(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	id := paramID(c)
	_, err := s.DB.Exec(`UPDATE notify_inbox SET read_at=NOW() WHERE id=? AND user_id=?`, id, cl.UserID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	api.OK(c, gin.H{"id": id, "read": true})
	return true
}

func (s *Service) handleTaskList(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	status := c.Query("status")
	if status == "" {
		status = "pending"
	}
	pageNum, pageSize := sqlutil.Page(c)
	roleSet := map[string]bool{}
	for _, r := range cl.Roles {
		roleSet[r] = true
	}
	isAdmin := roleSet["sys_admin"] || roleSet["admin"] || roleSet["系统管理员"]
	where := `WHERE t.status=?`
	args := []interface{}{status}
	if !isAdmin {
		// 已指定处理人：仅本人可见；未指定：按 to_role 对角色池可见
		where += ` AND (
			(COALESCE(t.assignee_user_id,0)>0 AND t.assignee_user_id=?)
			OR (COALESCE(t.assignee_user_id,0)=0 AND t.to_role IN (` + placeholders(len(cl.Roles)) + `))
		)`
		args = append(args, cl.UserID)
		for _, r := range cl.Roles {
			args = append(args, r)
		}
		if len(cl.Roles) == 0 {
			where = `WHERE t.status=? AND t.assignee_user_id=?`
			args = []interface{}{status, cl.UserID}
		}
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM wf_task t `+where, args...).Scan(&total)
	listArgs := append(append([]interface{}{}, args...), pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT t.id, t.event_key, t.biz_type, t.biz_id, COALESCE(t.doc_no,''), COALESCE(t.trace_code,''),
		COALESCE(t.from_role,''), t.to_role, COALESCE(t.assignee_user_id,0), COALESCE(t.payload_json,'{}'), t.status, t.created_at,
		COALESCE(NULLIF(e.name,''), u.login_name, '')
		FROM wf_task t
		LEFT JOIN iam_user u ON u.id=t.assignee_user_id
		LEFT JOIN hr_employee e ON e.id=u.employee_id
		`+where+` ORDER BY t.id DESC LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, bizID, assignee int64
		var ek, bt, doc, trace, fr, tr, pj, st, created, assigneeName string
		_ = rows.Scan(&id, &ek, &bt, &bizID, &doc, &trace, &fr, &tr, &assignee, &pj, &st, &created, &assigneeName)
		var payload interface{}
		_ = json.Unmarshal([]byte(pj), &payload)
		if strings.EqualFold(tr, "warehouse") {
			if m, ok := payload.(map[string]interface{}); ok {
				payload = maskTaskPayloadForWarehouse(m)
			}
		}
		list = append(list, gin.H{
			"id": id, "event_key": ek, "biz_type": bt, "biz_id": bizID, "doc_no": doc, "trace_code": trace,
			"from_role": fr, "to_role": tr, "assignee_user_id": assignee, "assignee_name": assigneeName,
			"payload": payload, "status": st, "created_at": created,
		})
	}
	api.OK(c, gin.H{"list": list, "total": total, "page_num": pageNum, "page_size": pageSize})
	return true
}

func maskTaskPayloadForWarehouse(src map[string]interface{}) map[string]interface{} {
	allow := map[string]bool{
		"doc_no": true, "batch_no": true, "trace_code": true, "variety": true, "product_name": true,
		"product_id": true, "plate_no": true, "gross_weight": true, "deduct_rate": true, "deduct_weight": true,
		"reject_weight": true, "net_weight": true, "image_url": true, "cold_store_type": true,
		"warehouse_id": true, "receive_kind": true, "status": true, "bag_qty": true, "channel": true,
		"biz_date": true, "box_code": true, "qc_result": true, "grade": true,
		"image_urls": true, "verify_images": true, "site_photos": true,
	}
	out := map[string]interface{}{}
	for k, v := range src {
		if allow[strings.ToLower(k)] {
			out[k] = v
		}
	}
	return out
}

func (s *Service) handleTaskClaim(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	id := paramID(c)
	res, err := s.DB.Exec(`UPDATE wf_task SET assignee_user_id=? WHERE id=? AND status='pending'
		AND (assignee_user_id IS NULL OR assignee_user_id=0 OR assignee_user_id=?)`, cl.UserID, id, cl.UserID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		api.FailJSON(c, "CLAIM_FAILED")
		return true
	}
	api.OK(c, gin.H{"id": id, "assignee_user_id": cl.UserID})
	return true
}

// handleTaskAssign 后台指定待办处理人（仓管等），并同步 weigh 协作单。
func (s *Service) handleTaskAssign(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return true
	}
	id := paramID(c)
	var body struct {
		ToUserID int64  `json:"to_user_id"`
		Comment  string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.ToUserID <= 0 {
		api.FailJSON(c, "TO_USER_REQUIRED")
		return true
	}

	var status, toRole, bizType, docNo, trace string
	var bizID, curAssignee int64
	err := s.DB.QueryRow(`SELECT status, COALESCE(to_role,''), COALESCE(biz_type,''), COALESCE(biz_id,0),
		COALESCE(doc_no,''), COALESCE(trace_code,''), COALESCE(assignee_user_id,0)
		FROM wf_task WHERE id=?`, id).
		Scan(&status, &toRole, &bizType, &bizID, &docNo, &trace, &curAssignee)
	if err != nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if status != "pending" {
		api.FailJSON(c, "TASK_NOT_PENDING")
		return true
	}

	roleNeed := strings.ToLower(strings.TrimSpace(toRole))
	if roleNeed == "" {
		roleNeed = "warehouse"
	}
	if !userHasRoleCode(s.DB, body.ToUserID, roleNeed) {
		// 仓管中文别名
		ok := false
		if roleNeed == "warehouse" {
			for _, alt := range []string{"仓管", "仓管员"} {
				if userHasRoleCode(s.DB, body.ToUserID, alt) {
					ok = true
					break
				}
			}
		}
		if !ok {
			api.FailJSON(c, "ASSIGNEE_ROLE_MISMATCH")
			return true
		}
	}

	_, err = s.DB.Exec(`UPDATE wf_task SET assignee_user_id=? WHERE id=? AND status='pending'`, body.ToUserID, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}

	if strings.EqualFold(bizType, "weigh_ticket") && bizID > 0 {
		var tid int64
		_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, bizID).Scan(&tid)
		if tid > 0 {
			_, _ = s.DB.Exec(`UPDATE wf_ticket SET current_assignee_user_id=?, status='in_progress', updated_at=NOW() WHERE id=?`, body.ToUserID, tid)
			_, _ = s.DB.Exec(`INSERT INTO wf_ticket_log(ticket_id, action, from_user_id, to_user_id, comment) VALUES(?,?,?,?,?)`,
				tid, "assign", nullIf0(cl.UserID), body.ToUserID, strings.TrimSpace(body.Comment))
		}
	}

	assigneeName := ""
	_ = s.DB.QueryRow(`SELECT COALESCE(NULLIF(e.name,''), u.login_name, '') FROM iam_user u
		LEFT JOIN hr_employee e ON e.id=u.employee_id WHERE u.id=?`, body.ToUserID).Scan(&assigneeName)

	title := "待办已指派给你"
	bodyText := docNo
	if trace != "" {
		bodyText = docNo + " · " + trace
	}
	if bodyText == "" {
		bodyText = "请打开 App 处理仓管待办"
	}
	pj, _ := json.Marshal(gin.H{
		"event_key": "workflow.task.assigned", "task_id": id, "biz_type": bizType, "biz_id": bizID,
		"doc_no": docNo, "trace_code": trace, "from_user_id": cl.UserID,
	})
	_, _ = s.DB.Exec(`INSERT INTO notify_inbox(user_id, title, body, event_key, task_id, payload_json) VALUES(?,?,?,?,?,?)`,
		body.ToUserID, title, bodyText, "workflow.task.assigned", id, string(pj))
	s.enqueueOutbox(erpmqtt.UserTopic(erpmqtt.Tenant(s.Cfg), body.ToUserID), gin.H{
		"title": title, "body": bodyText, "event_key": "workflow.task.assigned", "task_id": id,
	}, "task-assign-"+strconv.FormatInt(id, 10)+"-"+strconv.FormatInt(body.ToUserID, 10))

	api.OK(c, gin.H{
		"id": id, "assignee_user_id": body.ToUserID, "assignee_name": assigneeName,
		"prev_assignee_user_id": curAssignee,
	})
	return true
}

func userHasRoleCode(db *sql.DB, userID int64, roleCode string) bool {
	if db == nil || userID <= 0 || roleCode == "" {
		return false
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM iam_user_role ur
		JOIN iam_role r ON r.id=ur.role_id
		WHERE ur.user_id=? AND (r.code=? OR r.name=?)`, userID, roleCode, roleCode).Scan(&n)
	return n > 0
}

func paramID(c *gin.Context) int64 {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func placeholders(n int) string {
	if n <= 0 {
		return "''"
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}
