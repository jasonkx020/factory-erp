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

func TestParseGateWeighPhotos(t *testing.T) {
	okURL := "/files/uploads/a.jpg"
	photos, err := parseGateWeighPhotos(map[string]interface{}{
		"photos": map[string]interface{}{
			"material": okURL, "scale_display": okURL, "closeup": okURL,
		},
	})
	if err != "" || len(photos) != 3 {
		t.Fatalf("three slots: err=%s n=%d", err, len(photos))
	}
	if photos[0].EvidenceType != "weigh_material" || photos[1].EvidenceType != "scale_display" || photos[2].EvidenceType != "site_photo" {
		t.Fatalf("types %+v", photos)
	}
	if _, err := parseGateWeighPhotos(map[string]interface{}{
		"photos": map[string]interface{}{"material": okURL},
	}); err != "EVIDENCE_INCOMPLETE:scale_display" {
		t.Fatalf("incomplete got %s", err)
	}
	legacy, err := parseGateWeighPhotos(map[string]interface{}{"image_url": okURL})
	if err != "" || len(legacy) != 1 {
		t.Fatalf("legacy err=%s n=%d", err, len(legacy))
	}
}

func TestWeighProcessPhaseGateAcceptedIsAwaitStockin(t *testing.T) {
	if got := weighProcessPhase("gate", "weighed", ""); got != "await_gate" {
		t.Fatalf("weighed gate: %s", got)
	}
	if got := weighProcessPhase("gate", "gate_accepted", ""); got != "await_stockin" {
		t.Fatalf("gate_accepted: %s", got)
	}
	if got := weighProcessPhase("gate", "stocked", ""); got != "await_finance" {
		t.Fatalf("stocked gate: %s", got)
	}
	if got := weighProcessPhase("stockin", "weighed", ""); got != "await_warehouse" {
		t.Fatalf("weighed stockin: %s", got)
	}
}
