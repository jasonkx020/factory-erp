package biz

import "testing"

func TestWeighTraceLotDocLabel(t *testing.T) {
	if got := (weighTraceLot{Count: 1, DocNos: []string{"WT1"}}).docLabel(); got != "WT1" {
		t.Fatalf("single ticket label = %q", got)
	}
	if got := (weighTraceLot{Count: 3, DocNos: []string{"WT1", "WT2", "WT3"}}).docLabel(); got != "WT1 等3车" {
		t.Fatalf("merged ticket label = %q", got)
	}
}

func TestWeighTraceLotContains(t *testing.T) {
	lot := weighTraceLot{TicketIDs: []int64{2, 5, 9}}
	if !lot.contains(5) || lot.contains(1) {
		t.Fatal("contains mismatch")
	}
}
