package notify

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/config"
	"erp/internal/middleware"
	erpmqtt "erp/internal/mqtt"
)

type Service struct {
	DB  *sql.DB
	Cfg *config.Config
	Hub *erpmqtt.Hub
}

func New(db *sql.DB, cfg *config.Config, hub *erpmqtt.Hub) *Service {
	return &Service{DB: db, Cfg: cfg, Hub: hub}
}

// Event workflow handoff notification.
type Event struct {
	Key       string
	BizType   string
	BizID     int64
	DocNo     string
	TraceCode string
	FromRole  string
	ToRoles   []string
	Title     string
	Body      string
	Payload   map[string]interface{}
	CreateTask bool
}

func EnsureSchema(db *sql.DB) {
	_ = db // schema owned by migrations/erp
}

func (s *Service) NotifyNext(c *gin.Context, ev Event) {
	if s == nil || s.DB == nil {
		return
	}
	if ev.Payload == nil {
		ev.Payload = map[string]interface{}{}
	}
	ev.Payload["event_key"] = ev.Key
	ev.Payload["biz_type"] = ev.BizType
	ev.Payload["biz_id"] = ev.BizID
	ev.Payload["doc_no"] = ev.DocNo
	ev.Payload["trace_code"] = ev.TraceCode
	title := ev.Title
	body := ev.Body
	if title == "" {
		title, body = s.renderTemplate(ev.Key, ev.DocNo, ev.TraceCode)
	}
	fromUID := int64(0)
	if cl := middleware.Claims(c); cl != nil {
		fromUID = cl.UserID
		ev.Payload["from_user_id"] = fromUID
	}

	var taskID int64
	if ev.CreateTask && len(ev.ToRoles) > 0 {
		toRole := ev.ToRoles[0]
		dedupe := fmt.Sprintf("%s:%s:%d:%s", ev.Key, ev.BizType, ev.BizID, toRole)
		pj, _ := json.Marshal(ev.Payload)
		res, err := s.DB.Exec(`INSERT INTO wf_task(event_key, biz_type, biz_id, doc_no, trace_code, from_role, to_role, payload_json, status, dedupe_key)
			VALUES(?,?,?,?,?,?,?,?,'pending',?)`,
			ev.Key, ev.BizType, ev.BizID, ev.DocNo, ev.TraceCode, ev.FromRole, toRole, string(pj), dedupe)
		if err == nil {
			taskID, _ = res.LastInsertId()
			if taskID == 0 {
				_ = s.DB.QueryRow(`SELECT id FROM wf_task WHERE dedupe_key=?`, dedupe).Scan(&taskID)
			}
		}
		ev.Payload["task_id"] = taskID
	}

	userIDs := s.resolveReceivers(ev.ToRoles, ev.Payload)
	pj, _ := json.Marshal(ev.Payload)
	for _, uid := range userIDs {
		_, _ = s.DB.Exec(`INSERT INTO notify_inbox(user_id, title, body, event_key, task_id, payload_json)
			VALUES(?,?,?,?,?,?)`, uid, title, body, ev.Key, nullIf0(taskID), string(pj))
		s.enqueueOutbox(erpmqtt.UserTopic(erpmqtt.Tenant(s.Cfg), uid), gin.H{
			"title": title, "body": body, "event_key": ev.Key, "task_id": taskID, "payload": ev.Payload,
		}, fmt.Sprintf("inbox-user-%s-%d-%d", ev.Key, ev.BizID, uid))
	}
	for _, role := range ev.ToRoles {
		s.enqueueOutbox(erpmqtt.RoleTopic(erpmqtt.Tenant(s.Cfg), role), gin.H{
			"title": title, "body": body, "event_key": ev.Key, "task_id": taskID, "payload": ev.Payload,
		}, fmt.Sprintf("role-%s-%s-%d", role, ev.Key, ev.BizID))
	}
}

func (s *Service) CompleteTask(bizType string, bizID int64, eventKeys ...string) {
	if s == nil || s.DB == nil {
		return
	}
	q := `UPDATE wf_task SET status='done', done_at=NOW() WHERE biz_type=? AND biz_id=? AND status='pending'`
	args := []interface{}{bizType, bizID}
	if len(eventKeys) > 0 {
		q += ` AND event_key=?`
		args = append(args, eventKeys[0])
	}
	_, _ = s.DB.Exec(q, args...)
}

func (s *Service) renderTemplate(eventKey, docNo, traceCode string) (title, body string) {
	var tpl string
	_ = s.DB.QueryRow(`SELECT COALESCE(template_text,'') FROM sys_notify_rule WHERE event_key=?`, eventKey).Scan(&tpl)
	if tpl == "" {
		tpl = eventKey + " {{doc_no}} {{trace_code}}"
	}
	body = strings.ReplaceAll(tpl, "{{doc_no}}", docNo)
	body = strings.ReplaceAll(body, "{{trace_code}}", traceCode)
	title = eventKey
	switch eventKey {
	case "purchase.weigh_confirmed":
		title = "待仓管入库"
	case "purchase.weigh_returned":
		title = "仓管退回过磅单"
	case "purchase.stocked":
		title = "待财务支付"
	case "purchase.settle_paid":
		title = "结算已完成"
	case "production.report_confirmed":
		title = "报工已过账"
	case "payroll.labor_paid":
		title = "劳动已支付"
	}
	return title, body
}

func (s *Service) resolveReceivers(roles []string, payload map[string]interface{}) []int64 {
	ids := []int64{}
	seen := map[int64]bool{}
	explicitUsers := false
	if raw, ok := payload["notify_user_ids"]; ok {
		explicitUsers = true
		switch v := raw.(type) {
		case []int64:
			for _, id := range v {
				if id > 0 && !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		case []interface{}:
			for _, x := range v {
				id, _ := toInt64(x)
				if id > 0 && !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}
	for _, role := range roles {
		if role == "" || strings.HasPrefix(role, "_") {
			continue
		}
		rows, err := s.DB.Query(`SELECT DISTINCT u.id FROM iam_user u
			JOIN iam_user_role ur ON ur.user_id=u.id
			JOIN iam_role r ON r.id=ur.role_id
			WHERE r.code=? AND COALESCE(u.status,'active')='active'`, role)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id int64
			_ = rows.Scan(&id)
			if id > 0 && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		rows.Close()
	}
	// fallback: admin users so inbox is never empty in seed DB (skip when callers target explicit users)
	if len(ids) == 0 && !explicitUsers {
		rows, err := s.DB.Query(`SELECT id FROM iam_user WHERE COALESCE(status,'active')='active' ORDER BY id LIMIT 5`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				_ = rows.Scan(&id)
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

func (s *Service) enqueueOutbox(topic string, payload interface{}, dedupe string) {
	b, _ := json.Marshal(payload)
	_, _ = s.DB.Exec(`INSERT INTO notify_outbox(topic, payload_json, status, next_retry_at, dedupe_key)
		VALUES(?,?,'pending',NOW(),?)`, topic, string(b), dedupe)
}

func nullIf0(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// StartPublisher scans outbox and publishes via MQTT hub.
func (s *Service) StartPublisher(stop <-chan struct{}) {
	if s == nil {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.flushOutbox()
		}
	}
}

func (s *Service) flushOutbox() {
	rows, err := s.DB.Query(`SELECT id, topic, payload_json, attempts FROM notify_outbox
		WHERE status='pending' AND (next_retry_at IS NULL OR next_retry_at<=NOW())
		ORDER BY id LIMIT 50`)
	if err != nil {
		return
	}
	defer rows.Close()
	type item struct {
		id       int64
		topic    string
		payload  string
		attempts int
	}
	list := []item{}
	for rows.Next() {
		var it item
		_ = rows.Scan(&it.id, &it.topic, &it.payload, &it.attempts)
		list = append(list, it)
	}
	for _, it := range list {
		var payload interface{}
		_ = json.Unmarshal([]byte(it.payload), &payload)
		err := error(nil)
		if s.Hub != nil && s.Cfg != nil && s.Cfg.Mqtt.Enabled {
			err = s.Hub.Publish(it.topic, payload)
		}
		if err != nil {
			next := time.Now().Add(time.Duration(it.attempts+1) * 5 * time.Second).Format("2006-01-02 15:04:05")
			status := "pending"
			if it.attempts+1 >= 10 {
				status = "dead"
			}
			_, _ = s.DB.Exec(`UPDATE notify_outbox SET attempts=attempts+1, next_retry_at=?, status=? WHERE id=?`, next, status, it.id)
			continue
		}
		_, _ = s.DB.Exec(`UPDATE notify_outbox SET status='sent', sent_at=NOW() WHERE id=?`, it.id)
	}
}
