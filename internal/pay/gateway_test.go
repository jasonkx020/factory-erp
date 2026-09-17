package pay

import (
	"context"
	"testing"
)

func TestMockGatewayTransfer(t *testing.T) {
	res, err := MockGateway{}.Transfer(context.Background(), TransferRequest{
		PaymentNo:    "PO1",
		Amount:       100,
		PayeeName:    "张三",
		PayeeAccount: "622200001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.SyncPaid || res.ChannelTradeNo == "" {
		t.Fatalf("expected sync paid, got %+v", res)
	}
}

func TestBankDirectStub(t *testing.T) {
	_, err := (&BankDirectGateway{BankCode: "cmb"}).Transfer(context.Background(), TransferRequest{
		PaymentNo:    "PO2",
		Amount:       1,
		PayeeName:    "李四",
		PayeeAccount: "6222",
	})
	if err == nil || err.Error() != "BANK_DIRECT_NOT_IMPLEMENTED" {
		t.Fatalf("want BANK_DIRECT_NOT_IMPLEMENTED, got %v", err)
	}
	hint := (&BankDirectGateway{}).ReconcileHint()
	if hint == "" {
		t.Fatal("reconcile hint empty")
	}
}

func TestRegistryFallback(t *testing.T) {
	r := NewRegistry(MockGateway{}, &BankDirectGateway{})
	g, err := r.Get(ChannelAlipayBank)
	if err != nil {
		t.Fatal(err)
	}
	if g.Channel() != ChannelMock {
		t.Fatalf("fallback channel=%s", g.Channel())
	}
}
