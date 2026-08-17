package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) writeAudit(bizType string, bizID int64, action, reason string, before, after interface{}) {
	bb, _ := json.Marshal(before)
	aa, _ := json.Marshal(after)
	_, _ = s.DB.Exec(`INSERT INTO biz_audit_log(biz_type, biz_id, action, reason, before_json, after_json, created_at)
		VALUES(?,?,?,?,?,?,NOW())`, bizType, bizID, action, reason, string(bb), string(aa))
}

func (s *Services) writeAuditCtx(c *gin.Context, bizType string, bizID int64, action, reason string, before, after interface{}) {
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	bb, _ := json.Marshal(before)
	aa, _ := json.Marshal(after)
	_, _ = s.DB.Exec(`INSERT INTO biz_audit_log(biz_type, biz_id, action, reason, before_json, after_json, actor_user_id, created_at)
		VALUES(?,?,?,?,?,?,?,NOW())`, bizType, bizID, action, reason, string(bb), string(aa), uid)
}

func (s *Services) addEvidence(c *gin.Context, bizType string, bizID int64, evidenceType, fileURL string, meta map[string]interface{}) (int64, error) {
	var uid int64
	if cl := middleware.Claims(c); cl != nil {
		uid = cl.UserID
	}
	metaJSON := "{}"
	if meta != nil {
		b, _ := json.Marshal(meta)
		metaJSON = string(b)
	}
	res, err := s.DB.Exec(`INSERT INTO biz_evidence(biz_type, biz_id, evidence_type, file_url, meta_json, uploaded_by, uploaded_at)
		VALUES(?,?,?,?,?,?,NOW())`, bizType, bizID, evidenceType, fileURL, metaJSON, uid)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Services) countEvidence(bizType string, bizID int64, evidenceType string) int {
	var n int
	q := `SELECT COUNT(1) FROM biz_evidence WHERE biz_type=? AND biz_id=? AND COALESCE(voided_at,'')='' `
	args := []interface{}{bizType, bizID}
	if evidenceType != "" {
		q += ` AND evidence_type=?`
		args = append(args, evidenceType)
	}
	_ = s.DB.QueryRow(q, args...).Scan(&n)
	return n
}

func (s *Services) requireEvidence(bizType string, bizID int64, types ...string) error {
	for _, t := range types {
		if s.countEvidence(bizType, bizID, t) < 1 {
			return fmt.Errorf("EVIDENCE_INCOMPLETE:%s", t)
		}
	}
	return nil
}

func (s *Services) listEvidence(bizType string, bizID int64) []gin.H {
	rows, err := s.DB.Query(`SELECT id, evidence_type, COALESCE(file_url,''), COALESCE(meta_json,'{}'), COALESCE(uploaded_by,0), uploaded_at, COALESCE(voided_at,'')
		FROM biz_evidence WHERE biz_type=? AND biz_id=? ORDER BY id`, bizType, bizID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, uid int64
		var et, url, meta, uploaded, voided string
		_ = rows.Scan(&id, &et, &url, &meta, &uid, &uploaded, &voided)
		out = append(out, gin.H{
			"id": id, "evidence_type": et, "file_url": url, "meta_json": meta,
			"uploaded_by": uid, "uploaded_at": uploaded, "voided_at": voided,
		})
	}
	return out
}

func (s *Services) handleEvidenceAPI(c *gin.Context, method string) bool {
	switch method {
	case "POST":
		body := bindBody(c)
		bizType := strOr(body["biz_type"])
		bizID, _ := asInt64(body["biz_id"])
		et := strOr(body["evidence_type"])
		url := strOr(body["file_url"])
		if bizType == "" || bizID <= 0 || et == "" || url == "" {
			api.FailJSON(c, "INVALID_REQUEST")
			return true
		}
		id, err := s.addEvidence(c, bizType, bizID, et, url, nil)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		s.writeAuditCtx(c, bizType, bizID, "evidence_add", et, nil, gin.H{"evidence_id": id, "file_url": url})
		api.OK(c, gin.H{"id": id})
		return true
	case "GET":
		bizType := c.Query("biz_type")
		bizID, _ := strconvParseInt(c.Query("biz_id"))
		api.OK(c, gin.H{"list": s.listEvidence(bizType, bizID)})
		return true
	}
	return false
}

func strconvParseInt(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	var n int64
	_, err := fmt.Sscan(s, &n)
	return n, err == nil
}

func (s *Services) refuseDelete(c *gin.Context) bool {
	api.FailJSON(c, "DELETE_FORBIDDEN")
	return true
}

func nowSnap(fields map[string]interface{}) string {
	b, _ := json.Marshal(gin.H{"at": time.Now().Format(time.RFC3339), "fields": fields})
	return string(b)
}
