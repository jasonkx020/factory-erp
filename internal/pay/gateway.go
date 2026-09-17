package pay

import (
	"context"
	"fmt"
	"strings"
)

// Channel identifiers.
const (
	ChannelAlipayBank = "alipay_bank"
	ChannelWechatBank = "wechat_bank"
	ChannelBankDirect = "bank_direct"
	ChannelManual     = "manual"
	ChannelMock       = "mock"
)

// TransferRequest is a single outbound payment to a supplier bank account.
type TransferRequest struct {
	PaymentNo   string
	Amount      float64
	Currency    string
	PayeeName   string
	PayeeAccount string
	PayeeBank   string
	Remark      string
}

// TransferResult is returned by the gateway after submit (may still be async).
type TransferResult struct {
	Accepted       bool
	ChannelTradeNo string
	Message        string
	SyncPaid       bool // true when gateway confirms paid immediately (mock)
}

// Gateway is the pluggable payout adapter.
type Gateway interface {
	Channel() string
	Transfer(ctx context.Context, req TransferRequest) (TransferResult, error)
}

// Registry picks a gateway by channel name.
type Registry struct {
	gateways map[string]Gateway
	fallback Gateway
}

func NewRegistry(gs ...Gateway) *Registry {
	r := &Registry{gateways: map[string]Gateway{}}
	for _, g := range gs {
		if g == nil {
			continue
		}
		r.gateways[g.Channel()] = g
		if r.fallback == nil {
			r.fallback = g
		}
	}
	return r
}

func (r *Registry) Get(channel string) (Gateway, error) {
	if r == nil {
		return nil, fmt.Errorf("PAYMENT_GATEWAY_UNAVAILABLE")
	}
	ch := strings.TrimSpace(channel)
	if g, ok := r.gateways[ch]; ok {
		return g, nil
	}
	if r.fallback != nil {
		return r.fallback, nil
	}
	return nil, fmt.Errorf("PAYMENT_CHANNEL_UNSUPPORTED:%s", ch)
}
