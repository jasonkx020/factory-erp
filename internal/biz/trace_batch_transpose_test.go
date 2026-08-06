package biz

import "testing"

func TestTraceBatchTransposition10vs01(t *testing.T) {
	secret := "test-secret"
	code := BuildTraceBatchCode(secret, "2026-08-06", 15, "10")
	t.Logf("valid code: %s", code)
	tampered := code[:14] + "01" + code[16:]
	if _, _, _, ok := ParseTraceBatchCode(secret, tampered); ok {
		t.Fatal("expected reject when LOT 10 mistyped as 01 with original check digits")
	}
	code2 := BuildTraceBatchCode(secret, "2026-08-06", 10, "03")
	seqSwap := code2[:10] + "0001" + code2[14:]
	if _, _, _, ok := ParseTraceBatchCode(secret, seqSwap); ok {
		t.Fatal("expected reject SEQ 0010->0001 with original CHK")
	}
	dateSwap := "TB" + "20268006" + code2[10:]
	if _, _, _, ok := ParseTraceBatchCode(secret, dateSwap); ok {
		t.Fatal("expected reject date transposition")
	}
}
