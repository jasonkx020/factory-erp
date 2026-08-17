package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// handleCorrections POST /api/v1/biz/corrections — 冲正/更正，禁止物理删除。
func (s *Services) handleCorrections(c *gin.Context) bool {
	return s.handleCorrectionsWithBody(c, bindBody(c))
}

func (s *Services) handleCorrectionsWithBody(c *gin.Context, body map[string]interface{}) bool {
	bizType := strOr(body["biz_type"])
	bizID, _ := asInt64(body["biz_id"])
	reason := strings.TrimSpace(strOr(body["reason"]))
	action := strOrDef(body["action"], "correct")
	if bizType == "" || bizID <= 0 {
		api.FailJSON(c, "INVALID_REQUEST")
		return true
	}
	if reason == "" {
		api.FailJSON(c, "REASON_REQUIRED")
		return true
	}
	before := s.snapshotBiz(bizType, bizID)
	if before == nil {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	switch action {
	case "void":
		if err := s.voidBiz(bizType, bizID); err != nil {
			api.FailJSON(c, err.Error())
			return true
		}
	case "correct":
		fields, _ := body["fields"].(map[string]interface{})
		if fields == nil {
			if raw, ok := body["fields"]; ok {
				b, _ := json.Marshal(raw)
				_ = json.Unmarshal(b, &fields)
			}
		}
		if len(fields) == 0 {
			api.FailJSON(c, "FIELDS_REQUIRED")
			return true
		}
		if err := s.applyCorrection(bizType, bizID, fields); err != nil {
			api.FailJSON(c, err.Error())
			return true
		}
	default:
		api.FailJSON(c, "ACTION_UNSUPPORTED")
		return true
	}
	after := s.snapshotBiz(bizType, bizID)
	s.writeAuditCtx(c, bizType, bizID, action, reason, before, after)
	api.OK(c, gin.H{
		"biz_type": bizType, "biz_id": bizID, "action": action, "reason": reason,
		"before": before, "after": after, "corrected_at": time.Now().Format(time.RFC3339),
	})
	return true
}

func (s *Services) snapshotBiz(bizType string, id int64) gin.H {
	switch bizType {
	case "weigh_ticket":
		return s.loadWeighTicket(id)
	case "inbound_arrival":
		return s.loadArrival(id)
	case "farmer_settlement":
		return s.loadSettlement(id)
	case "report_work":
		return s.loadReportWork(id)
	case "evidence":
		list := s.listEvidenceByID(id)
		if len(list) == 0 {
			return nil
		}
		return list[0]
	default:
		return nil
	}
}

func (s *Services) voidBiz(bizType string, id int64) error {
	switch bizType {
	case "weigh_ticket":
		_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET status='void', updated_at=NOW() WHERE id=?`, id)
		return err
	case "inbound_arrival":
		_, err := s.DB.Exec(`UPDATE pur_inbound_arrival SET status='void', updated_at=NOW() WHERE id=?`, id)
		return err
	case "farmer_settlement":
		_, err := s.DB.Exec(`UPDATE pur_farmer_settlement SET status='void', updated_at=NOW() WHERE id=?`, id)
		return err
	case "report_work":
		_, err := s.DB.Exec(`UPDATE pd_report_work SET status='void' WHERE id=?`, id)
		return err
	case "evidence":
		_, err := s.DB.Exec(`UPDATE biz_evidence SET voided_at=NOW() WHERE id=? AND COALESCE(voided_at,'')=''`, id)
		return err
	default:
		return fmt.Errorf("BIZ_TYPE_UNSUPPORTED")
	}
}

func (s *Services) applyCorrection(bizType string, id int64, fields map[string]interface{}) error {
	switch bizType {
	case "weigh_ticket":
		// 入库后锁定净重/溯源；纠错须冲正，不允许直接改 locked 字段
		var status, trace string
		_ = s.DB.QueryRow(`SELECT status, COALESCE(trace_code,'') FROM pur_weigh_ticket WHERE id=?`, id).Scan(&status, &trace)
		if status == "stocked" || (trace != "" && status == "weighed") {
			if _, ok := fields["net_weight"]; ok {
				return fmt.Errorf("LOCKED_FIELD:net_weight")
			}
			if _, ok := fields["trace_code"]; ok {
				return fmt.Errorf("LOCKED_FIELD:trace_code")
			}
			if _, ok := fields["grade"]; ok {
				return fmt.Errorf("LOCKED_FIELD:grade")
			}
		}
		gross, hasG := asFloat(fields["gross_weight"])
		deductRate, hasDR := asFloat(fields["deduct_rate"])
		deductW, hasDW := asFloat(fields["deduct_weight"])
		net, hasN := asFloat(fields["net_weight"])
		remark := strOr(fields["remark"])
		if hasG || hasDR || hasDW || hasN {
			m := s.loadWeighTicket(id)
			if !hasG {
				gross, _ = asFloat(m["gross_weight"])
			}
			if !hasDR {
				deductRate, _ = asFloat(m["deduct_rate"])
			}
			if !hasDW {
				deductW = gross * deductRate
			}
			if !hasN {
				net = gross - deductW
			}
			_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET gross_weight=?, deduct_rate=?, deduct_weight=?, net_weight=?,
				remark=COALESCE(NULLIF(?,''),remark), updated_at=NOW() WHERE id=? AND status NOT IN ('stocked','void')`,
				gross, deductRate, deductW, net, remark, id)
			return err
		}
		if remark != "" {
			_, err := s.DB.Exec(`UPDATE pur_weigh_ticket SET remark=?, updated_at=NOW() WHERE id=?`, remark, id)
			return err
		}
		return fmt.Errorf("NO_FIELDS")
	case "farmer_settlement":
		price, ok := asFloat(fields["unit_price"])
		if !ok {
			return fmt.Errorf("UNIT_PRICE_REQUIRED")
		}
		var net float64
		var status string
		_ = s.DB.QueryRow(`SELECT net_weight, status FROM pur_farmer_settlement WHERE id=?`, id).Scan(&net, &status)
		if status == "settle_paid" {
			return fmt.Errorf("ALREADY_PAID")
		}
		_, err := s.DB.Exec(`UPDATE pur_farmer_settlement SET unit_price=?, amount=?, updated_at=NOW() WHERE id=?`,
			price, net*price, id)
		return err
	case "report_work":
		outW, hasO := asFloat(fields["output_weight"])
		inW, hasI := asFloat(fields["input_weight"])
		if !hasO && !hasI {
			return fmt.Errorf("WEIGHT_REQUIRED")
		}
		m := s.loadReportWork(id)
		if strOr(m["status"]) == "posted" || strOr(m["confirmed_at"]) != "" {
			return fmt.Errorf("ALREADY_CONFIRMED")
		}
		if !hasI {
			inW, _ = asFloat(m["input_weight"])
		}
		if !hasO {
			outW, _ = asFloat(m["output_weight"])
		}
		loss := inW - outW
		if loss < 0 {
			loss = 0
		}
		util := 0.0
		if inW > 0 {
			util = outW / inW
		}
		_, err := s.DB.Exec(`UPDATE pd_report_work SET input_weight=?, output_weight=?, qty=?, weight=?, qty_net=?, loss=?, utilization=? WHERE id=?`,
			inW, outW, outW, outW, outW, loss, util, id)
		return err
	default:
		return fmt.Errorf("BIZ_TYPE_UNSUPPORTED")
	}
}

func (s *Services) loadSettlement(id int64) gin.H {
	var farmerID, wtID int64
	var docNo, fname, bizDate, status, remark, created, transfer, paidAt, payURL string
	var net, price, amount float64
	err := s.DB.QueryRow(`SELECT s.doc_no, s.farmer_id, COALESCE(f.name,''), s.weigh_ticket_id, s.biz_date,
		s.net_weight, s.unit_price, s.amount, s.status, COALESCE(s.remark,''), s.created_at,
		COALESCE(s.transfer_no,''), COALESCE(s.paid_at,''), COALESCE(s.pay_evidence_url,'')
		FROM pur_farmer_settlement s LEFT JOIN pur_farmer f ON f.id=s.farmer_id WHERE s.id=?`, id).
		Scan(&docNo, &farmerID, &fname, &wtID, &bizDate, &net, &price, &amount, &status, &remark, &created, &transfer, &paidAt, &payURL)
	if err != nil {
		return nil
	}
	return gin.H{
		"id": id, "doc_no": docNo, "farmer_id": farmerID, "farmer_name": fname, "weigh_ticket_id": wtID,
		"biz_date": bizDate, "net_weight": net, "unit_price": price, "amount": amount, "status": status,
		"remark": remark, "created_at": created, "transfer_no": transfer, "paid_at": paidAt, "pay_evidence_url": payURL,
	}
}

func (s *Services) listEvidenceByID(id int64) []gin.H {
	rows, err := s.DB.Query(`SELECT id, biz_type, biz_id, evidence_type, COALESCE(file_url,''), COALESCE(meta_json,'{}'),
		COALESCE(uploaded_by,0), uploaded_at, COALESCE(voided_at,'') FROM biz_evidence WHERE id=?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var eid, bizID, uid int64
		var bt, et, url, meta, uploaded, voided string
		_ = rows.Scan(&eid, &bt, &bizID, &et, &url, &meta, &uid, &uploaded, &voided)
		out = append(out, gin.H{
			"id": eid, "biz_type": bt, "biz_id": bizID, "evidence_type": et, "file_url": url,
			"meta_json": meta, "uploaded_by": uid, "uploaded_at": uploaded, "voided_at": voided,
		})
	}
	return out
}
