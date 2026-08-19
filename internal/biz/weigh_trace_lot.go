package biz

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/middleware"
	"erp/internal/notify"
)

// weighTraceLot 同一溯源码下尚未完结入库的过磅单合集（入厂接收后待分板）。
type weighTraceLot struct {
	Trace     string
	TicketIDs []int64
	PrimaryID int64
	Net       float64
	Gross     float64
	Count     int
	DocNos    []string
	Since     string
}

func (s *Services) loadOpenTraceLot(trace string) weighTraceLot {
	out := weighTraceLot{Trace: strings.TrimSpace(trace)}
	if out.Trace == "" {
		return out
	}
	rows, err := s.DB.Query(`SELECT id, COALESCE(doc_no,''), COALESCE(net_weight,0), COALESCE(gross_weight,0),
		COALESCE(purchase_completed_at,''), COALESCE(created_at,'')
		FROM pur_weigh_ticket
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		AND LOWER(COALESCE(status,''))='gate_accepted'
		ORDER BY id`, out.Trace)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var docNo, completed, created string
		var net, gross float64
		if err := rows.Scan(&id, &docNo, &net, &gross, &completed, &created); err != nil {
			continue
		}
		out.TicketIDs = append(out.TicketIDs, id)
		out.DocNos = append(out.DocNos, docNo)
		out.Net += net
		out.Gross += gross
		if out.PrimaryID == 0 {
			out.PrimaryID = id
		}
		since := strings.TrimSpace(completed)
		if since == "" {
			since = strings.TrimSpace(created)
		}
		if since != "" && (out.Since == "" || since < out.Since) {
			out.Since = since
		}
	}
	out.Count = len(out.TicketIDs)
	return out
}

func (lot weighTraceLot) contains(id int64) bool {
	for _, x := range lot.TicketIDs {
		if x == id {
			return true
		}
	}
	return false
}

func (s *Services) weighBoxedProgressForLot(trace, since string) (count int, sum float64) {
	trace = strings.TrimSpace(trace)
	if trace == "" {
		return 0, 0
	}
	var stockedN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_weigh_ticket
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		AND LOWER(COALESCE(status,''))='stocked'`, trace).Scan(&stockedN)
	if stockedN == 0 || strings.TrimSpace(since) == "" {
		return s.weighBoxedProgress(trace)
	}
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(weight),0) FROM inv_box_code
		WHERE COALESCE(is_deleted,0)=0 AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		AND CAST(created_at AS TEXT)>=?`, trace, strings.TrimSpace(since)).Scan(&count, &sum)
	return count, sum
}

func (lot weighTraceLot) docLabel() string {
	if lot.Count <= 0 {
		return ""
	}
	if lot.Count == 1 {
		return lot.DocNos[0]
	}
	return fmt.Sprintf("%s 等%d车", lot.DocNos[0], lot.Count)
}

func (s *Services) closeInboundCollabOnGateAccept(c *gin.Context, weighID int64) {
	if weighID <= 0 {
		return
	}
	var tid int64
	_ = s.DB.QueryRow(`SELECT id FROM wf_ticket WHERE biz_type='weigh_ticket' AND biz_id=? AND status IN ('open','in_progress') ORDER BY id DESC LIMIT 1`, weighID).Scan(&tid)
	if tid <= 0 {
		return
	}
	_, _ = s.DB.Exec(`UPDATE wf_ticket SET status='done', closed_at=NOW(), updated_at=NOW(), current_assignee_user_id=NULL WHERE id=?`, tid)
	fromUID := int64(0)
	if cl := middleware.Claims(c); cl != nil {
		fromUID = cl.UserID
	}
	s.appendTicketLog(tid, "gate_accept_done", fromUID, 0, "inbound_closed")
	if s.Notify != nil {
		s.Notify.CompleteTask("wf_ticket", tid)
	}
}

func (s *Services) upsertWarehouseAwaitStockin(c *gin.Context, id int64, m gin.H) {
	if s.Notify == nil {
		return
	}
	trace := strings.TrimSpace(strOr(m["trace_code"]))
	lot := s.loadOpenTraceLot(trace)
	if lot.Count == 0 {
		lot = weighTraceLot{
			Trace: trace, TicketIDs: []int64{id}, PrimaryID: id, Count: 1,
			Net: asFloatOr0(m["net_weight"]), Gross: asFloatOr0(m["gross_weight"]),
			DocNos: []string{strOr(m["doc_no"])},
		}
	}
	boxedCnt, boxedSum := s.weighBoxedProgressForLot(trace, lot.Since)
	var assignee int64
	_ = s.DB.QueryRow(`SELECT COALESCE(assignee_user_id,0) FROM wf_task
		WHERE biz_type='weigh_ticket' AND biz_id=? AND event_key='purchase.weigh_confirmed'
		ORDER BY id DESC LIMIT 1`, id).Scan(&assignee)
	if assignee <= 0 {
		if cl := middleware.Claims(c); cl != nil {
			assignee = cl.UserID
		}
	}
	remain := lot.Net - boxedSum
	if remain < 0 {
		remain = 0
	}
	payload := gin.H{
		"net_weight": lot.Net, "trace_net_weight": lot.Net, "ticket_count": lot.Count,
		"weigh_ticket_ids": lot.TicketIDs, "weigh_ticket_id": lot.PrimaryID,
		"batch_no": m["batch_no"], "biz_date": m["biz_date"],
		"variety": m["variety"], "product_name": m["product_name"], "plate_no": m["plate_no"],
		"gross_weight": lot.Gross, "deduct_weight": m["deduct_weight"],
		"trace_code": trace, "receive_kind": m["receive_kind"],
		"status": "gate_accepted", "box_stockin_ready": true,
		"image_url": m["image_url"], "boxed_qty": boxedCnt, "boxed_weight": boxedSum,
		"remaining_weight": remain,
	}
	if assignee > 0 {
		payload["notify_user_ids"] = []int64{assignee}
	}
	docNo := lot.docLabel()
	if docNo == "" {
		docNo = strOr(m["doc_no"])
	}
	body := fmt.Sprintf("溯源 %s 已接收入厂合单（%d 车，净重 %.2f kg），请扫溯源分板入库", trace, lot.Count, lot.Net)
	if lot.Count == 1 {
		body = "已接收入厂 " + docNo + "，请扫溯源分板入库"
	}

	var taskID int64
	_ = s.DB.QueryRow(`SELECT id FROM wf_task
		WHERE event_key='purchase.await_stockin' AND status='pending'
		AND UPPER(COALESCE(trace_code,''))=UPPER(?)
		ORDER BY id LIMIT 1`, trace).Scan(&taskID)
	if taskID > 0 {
		pj, _ := json.Marshal(payload)
		_, _ = s.DB.Exec(`UPDATE wf_task SET payload_json=?, doc_no=?, biz_id=? WHERE id=?`,
			string(pj), docNo, lot.PrimaryID, taskID)
		if assignee > 0 {
			_, _ = s.DB.Exec(`UPDATE wf_task SET assignee_user_id=? WHERE id=? AND status='pending'`, assignee, taskID)
		}
		_, _ = s.DB.Exec(`UPDATE wf_task SET status='done', done_at=NOW()
			WHERE event_key='purchase.await_stockin' AND status='pending'
			AND UPPER(COALESCE(trace_code,''))=UPPER(?) AND id<>?`, trace, taskID)
		s.Notify.NotifyNext(c, notify.Event{
			Key: "purchase.await_stockin", BizType: "weigh_ticket", BizID: lot.PrimaryID,
			DocNo: docNo, TraceCode: trace,
			FromRole: "warehouse", ToRoles: []string{"warehouse"}, CreateTask: false,
			Title: "待入库", Body: body, Payload: payload,
		})
		return
	}

	s.Notify.NotifyNext(c, notify.Event{
		Key: "purchase.await_stockin", BizType: "weigh_ticket", BizID: lot.PrimaryID,
		DocNo: docNo, TraceCode: trace,
		FromRole: "warehouse", ToRoles: []string{"warehouse"}, CreateTask: true,
		Title: "待入库", Body: body, Payload: payload,
	})
	if assignee > 0 {
		_, _ = s.DB.Exec(`UPDATE wf_task SET assignee_user_id=?
			WHERE event_key='purchase.await_stockin' AND status='pending'
			AND UPPER(COALESCE(trace_code,''))=UPPER(?)`, assignee, trace)
	}
}

func (s *Services) completeAwaitStockinByTrace(trace string, ticketIDs []int64) {
	if s.Notify == nil {
		return
	}
	trace = strings.TrimSpace(trace)
	if trace != "" {
		s.Notify.CompletePendingByTrace("purchase.await_stockin", trace)
	}
	seen := map[int64]bool{}
	for _, tid := range ticketIDs {
		if tid <= 0 || seen[tid] {
			continue
		}
		seen[tid] = true
		s.Notify.CompleteTask("weigh_ticket", tid, "purchase.await_stockin")
		s.Notify.CompleteTask("weigh_ticket", tid, "purchase.weigh_confirmed")
	}
}
