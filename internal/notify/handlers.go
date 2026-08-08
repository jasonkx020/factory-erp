package notify

import (
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
	_, err := s.DB.Exec(`UPDATE notify_inbox SET read_at=datetime('now') WHERE id=? AND user_id=?`, id, cl.UserID)
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
	where := `WHERE status=?`
	args := []interface{}{status}
	if !isAdmin {
		where += ` AND (assignee_user_id=? OR to_role IN (` + placeholders(len(cl.Roles)) + `))`
		args = append(args, cl.UserID)
		for _, r := range cl.Roles {
			args = append(args, r)
		}
		if len(cl.Roles) == 0 {
			where = `WHERE status=? AND assignee_user_id=?`
			args = []interface{}{status, cl.UserID}
		}
	}
	var total int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM wf_task `+where, args...).Scan(&total)
	args = append(args, pageSize, (pageNum-1)*pageSize)
	rows, err := s.DB.Query(`SELECT id, event_key, biz_type, biz_id, COALESCE(doc_no,''), COALESCE(trace_code,''),
		COALESCE(from_role,''), to_role, COALESCE(assignee_user_id,0), COALESCE(payload_json,'{}'), status, created_at
		FROM wf_task `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, bizID, assignee int64
		var ek, bt, doc, trace, fr, tr, pj, st, created string
		_ = rows.Scan(&id, &ek, &bt, &bizID, &doc, &trace, &fr, &tr, &assignee, &pj, &st, &created)
		var payload interface{}
		_ = json.Unmarshal([]byte(pj), &payload)
		if strings.EqualFold(tr, "warehouse") {
			if m, ok := payload.(map[string]interface{}); ok {
				payload = maskTaskPayloadForWarehouse(m)
			}
		}
		list = append(list, gin.H{
			"id": id, "event_key": ek, "biz_type": bt, "biz_id": bizID, "doc_no": doc, "trace_code": trace,
			"from_role": fr, "to_role": tr, "assignee_user_id": assignee, "payload": payload, "status": st, "created_at": created,
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
