package pay

import (
	"context"
	"fmt"
	"time"
)

// MockGateway simulates successful payout (dev / no credentials).
type MockGateway struct{}

func (MockGateway) Channel() string { return ChannelMock }

func (MockGateway) Transfer(_ context.Context, req TransferRequest) (TransferResult, error) {
	if req.PaymentNo == "" || req.Amount <= 0 {
		return TransferResult{}, fmt.Errorf("PAYMENT_REQUEST_INVALID")
	}
	if stringsTrim(req.PayeeAccount) == "" || stringsTrim(req.PayeeName) == "" {
		return TransferResult{}, fmt.Errorf("PAYEE_BANK_REQUIRED")
	}
	return TransferResult{
		Accepted:       true,
		ChannelTradeNo: fmt.Sprintf("MOCK%s", time.Now().Format("150405")),
		Message:        "mock paid",
		SyncPaid:       true,
	}, nil
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
