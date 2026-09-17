package pay

import (
	"context"
	"fmt"
)

// BankDirectGateway is a phase-2 stub for 银企直联 adapters (CMB/ICBC/...).
// Transfer always returns BANK_DIRECT_NOT_IMPLEMENTED until a bank is wired.
type BankDirectGateway struct {
	BankCode string // e.g. cmb, icbc
}

func (g *BankDirectGateway) Channel() string { return ChannelBankDirect }

func (g *BankDirectGateway) Transfer(_ context.Context, req TransferRequest) (TransferResult, error) {
	if stringsTrim(req.PayeeAccount) == "" {
		return TransferResult{}, fmt.Errorf("PAYEE_BANK_REQUIRED")
	}
	return TransferResult{
		Accepted: false,
		Message:  fmt.Sprintf("bank_direct stub bank=%s", g.BankCode),
	}, fmt.Errorf("BANK_DIRECT_NOT_IMPLEMENTED")
}

// ReconcileHint documents the phase-2 reconciliation hook.
func (g *BankDirectGateway) ReconcileHint() string {
	return "Pull bank statement files daily and match fin_payment_order.channel_trade_no / payment_no"
}
