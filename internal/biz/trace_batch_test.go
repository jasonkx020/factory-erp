package biz

import "testing"

func TestTraceBatchCodeRoundTrip(t *testing.T) {
	secret := "test-secret"
	code := BuildTraceBatchCode(secret, "2026-08-06", 15, "03")
	if len(code) != 18 {
		t.Fatalf("len=%d code=%s", len(code), code)
	}
	if code[:2] != "TB" || code[2:10] != "20260806" {
		t.Fatalf("prefix/date bad: %s", code)
	}
	biz, seq, lot, ok := ParseTraceBatchCode(secret, code)
	if !ok || biz != "20260806" || seq != 15 || lot != "03" {
		t.Fatalf("parse got ok=%v biz=%s seq=%d lot=%s", ok, biz, seq, lot)
	}
	// typo
	bad := code[:17] + "X"
	if bad == code {
		bad = code[:16] + "00"
	}
	if _, _, _, ok := ParseTraceBatchCode(secret, bad); ok {
		t.Fatal("expected invalid for typo")
	}
	if isValidSitePhotoURL("mobile://x") {
		t.Fatal("mobile scheme should fail")
	}
	if !isValidSitePhotoURL("/files/uploads/a.jpg") {
		t.Fatal("files url should pass")
	}
}
