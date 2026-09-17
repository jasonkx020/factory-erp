package pay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AlipayConfig credentials for alipay.fund.trans.uni.transfer (to bank card).
type AlipayConfig struct {
	AppID           string
	PrivateKeyPEM   string
	AlipayPublicKey string
	NotifyURL       string
	GatewayURL      string // default https://openapi.alipay.com/gateway.do
}

// AlipayBankGateway transfers to a bank card via Alipay open API.
// When AppID is empty, Transfer returns an error directing callers to use mock.
type AlipayBankGateway struct {
	Cfg    AlipayConfig
	Client *http.Client
}

func (g *AlipayBankGateway) Channel() string { return ChannelAlipayBank }

func (g *AlipayBankGateway) Transfer(ctx context.Context, req TransferRequest) (TransferResult, error) {
	if g == nil || strings.TrimSpace(g.Cfg.AppID) == "" {
		return TransferResult{}, fmt.Errorf("ALIPAY_NOT_CONFIGURED")
	}
	if strings.TrimSpace(req.PayeeAccount) == "" || strings.TrimSpace(req.PayeeName) == "" {
		return TransferResult{}, fmt.Errorf("PAYEE_BANK_REQUIRED")
	}
	biz := map[string]interface{}{
		"out_biz_no":   req.PaymentNo,
		"trans_amount": fmt.Sprintf("%.2f", req.Amount),
		"product_code": "TRANS_BANKCARD_NO_PWD",
		"biz_scene":    "DIRECT_TRANSFER",
		"order_title":  strDef(req.Remark, "供应商货款"),
		"payee_info": map[string]string{
			"identity":      req.PayeeAccount,
			"identity_type": "BANKCARD_ACCOUNT",
			"name":          req.PayeeName,
		},
	}
	if g.Cfg.NotifyURL != "" {
		biz["remark"] = req.Remark
	}
	bizJSON, _ := json.Marshal(biz)
	params := map[string]string{
		"app_id":      g.Cfg.AppID,
		"method":      "alipay.fund.trans.uni.transfer",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizJSON),
	}
	if g.Cfg.NotifyURL != "" {
		params["notify_url"] = g.Cfg.NotifyURL
	}
	sign, err := signRSA2(params, g.Cfg.PrivateKeyPEM)
	if err != nil {
		return TransferResult{}, fmt.Errorf("ALIPAY_SIGN_ERROR:%v", err)
	}
	params["sign"] = sign

	gw := g.Cfg.GatewayURL
	if gw == "" {
		gw = "https://openapi.alipay.com/gateway.do"
	}
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, gw, strings.NewReader(form.Encode()))
	if err != nil {
		return TransferResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := client.Do(httpReq)
	if err != nil {
		return TransferResult{}, fmt.Errorf("ALIPAY_HTTP:%v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var wrap map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrap); err != nil {
		return TransferResult{}, fmt.Errorf("ALIPAY_RESP_INVALID")
	}
	raw, ok := wrap["alipay_fund_trans_uni_transfer_response"]
	if !ok {
		return TransferResult{Accepted: false, Message: string(body)}, fmt.Errorf("ALIPAY_RESP_MISSING")
	}
	var out struct {
		Code     string `json:"code"`
		Msg      string `json:"msg"`
		SubCode  string `json:"sub_code"`
		SubMsg   string `json:"sub_msg"`
		OrderID  string `json:"order_id"`
		OutBizNo string `json:"out_biz_no"`
		Status   string `json:"status"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Code != "10000" {
		msg := out.SubMsg
		if msg == "" {
			msg = out.Msg
		}
		return TransferResult{Accepted: false, Message: msg}, fmt.Errorf("ALIPAY_FAIL:%s", msg)
	}
	syncPaid := strings.EqualFold(out.Status, "SUCCESS")
	return TransferResult{
		Accepted:       true,
		ChannelTradeNo: out.OrderID,
		Message:        out.Status,
		SyncPaid:       syncPaid,
	}, nil
}

func strDef(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func signRSA2(params map[string]string, privateKeyPEM string) (string, error) {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	block, _ := pem.Decode([]byte(normalizePEM(privateKeyPEM)))
	if block == nil {
		return "", fmt.Errorf("invalid private key pem")
	}
	var key *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		key = k
	} else if k8, err2 := x509.ParsePKCS8PrivateKey(block.Bytes); err2 == nil {
		var ok bool
		key, ok = k8.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("not rsa private key")
		}
	} else {
		return "", err
	}
	h := sha256.Sum256([]byte(b.String()))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func normalizePEM(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "BEGIN") {
		return s
	}
	return "-----BEGIN RSA PRIVATE KEY-----\n" + s + "\n-----END RSA PRIVATE KEY-----"
}

// VerifyAlipayNotify verifies RSA2 notify signature (simplified: checks sign present + reconstruct).
func VerifyAlipayNotify(values url.Values, alipayPublicKeyPEM string) bool {
	sign := values.Get("sign")
	if sign == "" || strings.TrimSpace(alipayPublicKeyPEM) == "" {
		return false
	}
	params := map[string]string{}
	for k := range values {
		if k == "sign" || k == "sign_type" {
			continue
		}
		params[k] = values.Get(k)
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	block, _ := pem.Decode([]byte(normalizePublicPEM(alipayPublicKeyPEM)))
	if block == nil {
		return false
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}
	h := sha256.Sum256([]byte(b.String()))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig) == nil
}

func normalizePublicPEM(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "BEGIN") {
		return s
	}
	return "-----BEGIN PUBLIC KEY-----\n" + s + "\n-----END PUBLIC KEY-----"
}
