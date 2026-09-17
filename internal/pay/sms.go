package pay

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SmsSender sends payment success SMS to suppliers.
type SmsSender interface {
	Send(ctx context.Context, mobile, templateCode string, params map[string]string) error
}

// LogSmsSender records SMS in-memory / via callback for tests (no external call).
type LogSmsSender struct {
	LastMobile string
	LastCode   string
	LastParams map[string]string
}

func (s *LogSmsSender) Send(_ context.Context, mobile, templateCode string, params map[string]string) error {
	if s != nil {
		s.LastMobile = mobile
		s.LastCode = templateCode
		s.LastParams = params
	}
	return nil
}

// AliyunSmsConfig for Dysmsapi.
type AliyunSmsConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string
}

// AliyunSmsSender implements Aliyun SMS OpenAPI (simplified single-send).
type AliyunSmsSender struct {
	Cfg    AliyunSmsConfig
	Client *http.Client
}

func (s *AliyunSmsSender) Send(ctx context.Context, mobile, templateCode string, params map[string]string) error {
	if s == nil || strings.TrimSpace(s.Cfg.AccessKeyID) == "" {
		return fmt.Errorf("SMS_NOT_CONFIGURED")
	}
	tpl := templateCode
	if tpl == "" {
		tpl = s.Cfg.TemplateCode
	}
	if mobile == "" || tpl == "" || s.Cfg.SignName == "" {
		return fmt.Errorf("SMS_PARAM_REQUIRED")
	}
	endpoint := s.Cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://dysmsapi.aliyuncs.com/"
	}
	paramJSON := "{"
	i := 0
	for k, v := range params {
		if i > 0 {
			paramJSON += ","
		}
		paramJSON += fmt.Sprintf("%q:%q", k, v)
		i++
	}
	paramJSON += "}"
	query := url.Values{}
	query.Set("AccessKeyId", s.Cfg.AccessKeyID)
	query.Set("Action", "SendSms")
	query.Set("Format", "JSON")
	query.Set("PhoneNumbers", mobile)
	query.Set("SignName", s.Cfg.SignName)
	query.Set("SignatureMethod", "HMAC-SHA1")
	query.Set("SignatureNonce", fmt.Sprintf("%d", time.Now().UnixNano()))
	query.Set("SignatureVersion", "1.0")
	query.Set("TemplateCode", tpl)
	query.Set("TemplateParam", paramJSON)
	query.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	query.Set("Version", "2017-05-25")
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canonicalQuery(query))
	mac := hmac.New(sha1.New, []byte(s.Cfg.AccessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	query.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	u := endpoint + "?" + query.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("SMS_HTTP_%d:%s", resp.StatusCode, string(body))
	}
	if strings.Contains(string(body), `"Code":"OK"`) || strings.Contains(string(body), `"Code": "OK"`) {
		return nil
	}
	// tolerate soft failure for delivery logging upstream
	if strings.Contains(string(body), "Code") && !strings.Contains(string(body), "OK") {
		return fmt.Errorf("SMS_FAIL:%s", string(body))
	}
	return nil
}

func percentEncode(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(url.QueryEscape(s), "+", "%20"), "*", "%2A")
}

func canonicalQuery(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	// sort via QueryEscape join
	sortStrings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, percentEncode(k)+"="+percentEncode(v.Get(k)))
	}
	return strings.Join(parts, "&")
}

func sortStrings(a []string) {
	for i := 0; i < len(a); i++ {
		for j := i + 1; j < len(a); j++ {
			if a[j] < a[i] {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}
