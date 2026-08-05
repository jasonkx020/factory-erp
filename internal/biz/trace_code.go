package biz

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TraceHMACSecret returns signing secret (config/env).
func TraceHMACSecret(override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	if v := os.Getenv("ERP_TRACE_HMAC_SECRET"); v != "" {
		return v
	}
	return "dev-trace-hmac-change-me"
}

type TraceIssueInput struct {
	BizDate    string  // YYYY-MM-DD or YYYYMMDD
	BatchNo    string  // 6 digits
	FarmerID   int64
	Grade      string
	Channel    string
	SourceType string
	NetKg      float64
	ArrivalID  int64
}

func normalizeBizDate(d string) string {
	d = strings.ReplaceAll(d, "-", "")
	if len(d) >= 8 {
		return d[:8]
	}
	return time.Now().Format("20060102")
}

func farmerCode4(id int64) string {
	const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if id <= 0 {
		return "0000"
	}
	n := id
	out := make([]byte, 0, 8)
	for n > 0 {
		out = append(out, digits[n%36])
		n /= 36
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	s := string(out)
	if len(s) > 4 {
		sum := sha256.Sum256([]byte(strconv.FormatInt(id, 10)))
		return strings.ToUpper(hex.EncodeToString(sum[:2]))
	}
	return fmt.Sprintf("%04s", s)
}

func (in TraceIssueInput) Canonical() string {
	biz := normalizeBizDate(in.BizDate)
	batch := in.BatchNo
	if len(batch) < 6 {
		batch = fmt.Sprintf("%06s", batch)
	}
	grade := strings.ToUpper(strings.TrimSpace(in.Grade))
	if grade == "" {
		grade = "NA"
	}
	ch := strOrDef(in.Channel, "internal")
	src := strOrDef(in.SourceType, "self")
	return fmt.Sprintf("v=1|biz_date=%s|batch=%s|farmer_id=%d|grade=%s|channel=%s|source_type=%s|net_kg=%.2f|arrival_id=%d",
		biz, batch, in.FarmerID, grade, ch, src, in.NetKg, in.ArrivalID)
}

func SignCanonical(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	sum := mac.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(sum[:4])) // 8 hex chars
}

func VerifyCanonical(secrets []string, canonical, sig string) bool {
	sig = strings.ToUpper(strings.TrimSpace(sig))
	for _, sec := range secrets {
		if hmac.Equal([]byte(SignCanonical(sec, canonical)), []byte(sig)) {
			return true
		}
	}
	return false
}

func IssueTraceCode(secret string, in TraceIssueInput) (traceCode, canonical, signature string) {
	biz := normalizeBizDate(in.BizDate)
	batch := in.BatchNo
	if len(batch) < 6 {
		batch = fmt.Sprintf("%06s", batch)
	}
	canonical = in.Canonical()
	signature = SignCanonical(secret, canonical)
	traceCode = fmt.Sprintf("T1-%s-%s-%s-%s", biz, batch, farmerCode4(in.FarmerID), signature)
	return
}

func ParseTraceSig(traceCode string) (ok bool, sig string) {
	parts := strings.Split(traceCode, "-")
	if len(parts) < 5 || parts[0] != "T1" {
		return false, ""
	}
	return true, parts[len(parts)-1]
}

func NextBatchNo(s *Services, bizDate string) (string, error) {
	biz := normalizeBizDate(bizDate)
	var maxN int
	_ = s.DB.QueryRow(`SELECT COALESCE(MAX(CAST(batch_no AS INTEGER)),0) FROM pur_trace_lot WHERE biz_date=? OR biz_date=?`,
		biz, bizDate).Scan(&maxN)
	return fmt.Sprintf("%06d", maxN+1), nil
}
