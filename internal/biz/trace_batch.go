package biz

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"
)

const (
	traceBatchPrefix = "TB"
	base36Digits     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// ColdStoreWarehouse maps cold_store_type → default warehouse_id.
func ColdStoreWarehouse(kind string) int64 {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "fresh":
		return 1
	case "semi":
		return 2
	case "fg":
		return 3
	default:
		return 0
	}
}

func padLot2(lot string) string {
	lot = strings.ToUpper(strings.TrimSpace(lot))
	if lot == "" {
		return "01"
	}
	var b strings.Builder
	for _, r := range lot {
		if unicode.IsDigit(r) || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return "01"
	}
	if len(s) == 1 {
		return "0" + s
	}
	if len(s) > 2 {
		return s[:2]
	}
	return s
}

func base36Index(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'A' && c <= 'Z' {
		return int(c-'A') + 10
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 10
	}
	return -1
}

// luhnBase36Check returns one Base36 check char for payload (防输错).
func luhnBase36Check(payload string) byte {
	sum := 0
	alt := false
	for i := len(payload) - 1; i >= 0; i-- {
		v := base36Index(payload[i])
		if v < 0 {
			continue
		}
		if alt {
			v *= 2
			if v >= 36 {
				v = v/36 + v%36
			}
		}
		sum += v
		alt = !alt
	}
	return base36Digits[(36-(sum%36))%36]
}

func hmacBase36Check(secret, payload string) byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	sum := mac.Sum(nil)
	return base36Digits[int(sum[0])%36]
}

// BuildTraceBatchCode builds TB+YYYYMMDD+SEQ4+LOT2+CHK2.
func BuildTraceBatchCode(secret, bizDate string, seq int, lot string) string {
	date := normalizeBizDate(bizDate)
	seqS := fmt.Sprintf("%04d", seq)
	if seq < 0 {
		seqS = "0000"
	}
	if seq > 9999 {
		seqS = fmt.Sprintf("%04d", seq%10000)
	}
	lot2 := padLot2(lot)
	payload := traceBatchPrefix + date + seqS + lot2
	chk := string([]byte{luhnBase36Check(payload), hmacBase36Check(secret, payload)})
	return payload + chk
}

// ParseTraceBatchCode validates format + CHK2. Returns parts if ok.
func ParseTraceBatchCode(secret, code string) (bizDate string, seq int, lot string, ok bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 18 || !strings.HasPrefix(code, traceBatchPrefix) {
		return "", 0, "", false
	}
	payload := code[:16]
	want := string([]byte{luhnBase36Check(payload), hmacBase36Check(secret, payload)})
	if code[16:] != want {
		return "", 0, "", false
	}
	bizDate = payload[2:10]
	lot = payload[14:16]
	_, _ = fmt.Sscanf(payload[10:14], "%d", &seq)
	return bizDate, seq, lot, true
}

func isValidSitePhotoURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	low := strings.ToLower(u)
	if strings.HasPrefix(low, "mobile://") || strings.HasPrefix(low, "file://") {
		return false
	}
	return strings.HasPrefix(u, "/files/") || strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://")
}

func collectImageURLs(body map[string]interface{}) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	add(strOr(body["image_url"]))
	if arr, ok := body["image_urls"].([]interface{}); ok {
		for _, v := range arr {
			add(fmt.Sprint(v))
		}
	}
	return out
}
