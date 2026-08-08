package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func main() {
	base := env("API_BASE", "http://127.0.0.1:18080/api/v1")
	login := env("ERP_LOGIN", "admin")
	pass := env("ERP_PASSWORD", "admin123")

	token, err := doLogin(base, login, pass)
	if err != nil {
		fail("login", err)
	}
	ok("login")

	checks := []struct {
		name, method, path string
		body               map[string]any
		optional           bool
	}{
		{"me", "GET", "/auth/me", nil, false},
		{"farmers", "GET", "/purchase/farmers?page_size=5", nil, false},
		{"weigh_list", "GET", "/purchase/weigh-tickets?page_size=5", nil, false},
		{"weigh_varieties", "GET", "/purchase/weigh-varieties?status=active", nil, false},
		{"purchase_tasks", "GET", "/purchase/tasks?page_size=5", nil, false},
		{"balances", "GET", "/inventory/balances?page_size=5", nil, false},
		{"box_codes", "GET", "/inventory/box-codes?page_size=5", nil, false},
		{"stocktakes", "GET", "/inventory/stocktakes?page_size=5", nil, false},
		{"tasks", "GET", "/production/tasks?page_size=5", nil, false},
		{"dispatches", "GET", "/production/dispatches?page_size=5", nil, false},
		{"flex", "GET", "/production/flex-dispatches?page_size=5", nil, false},
		{"qc", "GET", "/production/qc-orders?page_size=5", nil, false},
		{"scraps", "GET", "/production/scraps?page_size=5", nil, false},
		{"reworks", "GET", "/production/reworks?page_size=5", nil, false},
		{"requisitions", "GET", "/production/requisitions", nil, false},
		{"workbench", "GET", "/production/workshop-workbench/overview", nil, false},
		{"piecework_mine", "GET", "/production/piecework-summaries/mine", nil, false},
		{"orders", "GET", "/sales/orders?page_size=5", nil, true},
		{"customers", "GET", "/crm/customers?page_size=5", nil, true},
		{"follow_ups", "GET", "/crm/follow-ups?page_size=5", nil, true},
		{"leave", "GET", "/hr/leave-requests", nil, false},
		{"overtime", "GET", "/hr/overtime-patches", nil, false},
		{"approval_tasks", "GET", "/approval/tasks", nil, false},
		{"payroll_sheets", "GET", "/payroll/sheets?page_size=5", nil, false},
		{"commission", "GET", "/payroll/commission-calcs?page_size=5", nil, false},
		{"products", "GET", "/product/products?page_size=5", nil, false},
		{"punch", "POST", "/hr/attendance/records/punch", map[string]any{"punch_type": "in"}, false},
		{"assets", "GET", "/asset/fixed-assets?page_size=5", nil, true},
		{"asset_transfers", "GET", "/asset/transfers?page_size=5", nil, true},
		{"receipt_alerts", "GET", "/finance/receipt-alerts?page_size=5", nil, true},
		{"recognitions", "GET", "/finance/payment-recognitions?page_size=5", nil, true},
		{"knowledge", "GET", "/system/knowledge?page_size=5", nil, true},
		{"drawings", "GET", "/system/drawings?page_size=5", nil, true},
		{"announcements", "GET", "/system/announcements?page_size=5", nil, true},
		{"courses", "GET", "/system/courses?page_size=5", nil, true},
		{"deliveries", "GET", "/sales/deliveries?page_size=5", nil, true},
		{"pre_ships", "GET", "/sales/pre-shipments?page_size=5", nil, true},
		{"quote_calc", "POST", "/sales/quote-calculator/calc", map[string]any{"product_id": 3, "qty": 10, "margin_rate": 0.2}, true},
		{"alert_shortage", "GET", "/inventory/alert-rules/shortage", nil, false},
		{"alert_excess", "GET", "/inventory/alert-rules/excess", nil, false},
		{"stock_txns", "GET", "/inventory/stock-txns", nil, false},
		{"drawing_links", "GET", "/production/drawing-links", nil, false},
		{"memos", "GET", "/hr/memos", nil, false},
		{"journals", "GET", "/hr/employee-journals", nil, false},
	}

	failed := 0
	for _, c := range checks {
		if err := call(base, token, c.method, c.path, c.body); err != nil {
			if c.optional {
				warn(c.name, err)
				continue
			}
			fail(c.name, err)
			failed++
			continue
		}
		ok(c.name)
	}

	// closed-loop: generate batch → upload → gate + stockin create → qc → confirm
	farmerID := firstID(base, token, "/purchase/farmers?page_size=1", "id")
	bizDate := time.Now().Format("2006-01-02")
	var gen map[string]any
	if err := callJSON(base, token, "POST", "/purchase/trace-batch-codes/generate", map[string]any{
		"biz_date": bizDate, "lot_no": "01", "qty": 3,
	}, &gen); err != nil {
		fail("trace_batch_generate", err)
		failed++
	} else {
		ok("trace_batch_generate")
	}
	codes := listCodes(gen)
	if len(codes) < 2 {
		warn("trace_batch_generate", fmt.Errorf("need >=2 codes, got %d", len(codes)))
	} else {
		// bad checksum
		bad := codes[0][:len(codes[0])-1] + "0"
		if bad == codes[0] {
			bad = codes[0][:len(codes[0])-1] + "1"
		}
		if err := call(base, token, "POST", "/purchase/trace-batch-codes/validate", map[string]any{"code": bad}); err == nil {
			fail("trace_batch_bad_chk", fmt.Errorf("expected reject"))
			failed++
		} else {
			ok("trace_batch_bad_chk")
		}
	}

	imgURL, err := uploadSmokeImage(base, token)
	if err != nil {
		fail("upload", err)
		failed++
	} else {
		ok("upload")
	}

	if farmerID > 0 && len(codes) >= 2 && imgURL != "" {
		var created map[string]any
		if err := callJSON(base, token, "POST", "/purchase/weigh-tickets", map[string]any{
			"receive_kind": "gate", "batch_no": codes[0], "farmer_id": farmerID,
			"channel": "internal", "product_id": 1, "variety": "鲜木薯",
			"gross_weight": 1000, "deduct_rate": 5, "unit_price": 1.2, "grade": "A",
			"image_url": imgURL, "biz_date": bizDate, "source_type": "self",
		}, &created); err != nil {
			fail("weigh_create_gate", err)
			failed++
		} else {
			ok("weigh_create_gate")
			id := asInt(created["id"])
			if id > 0 {
				if err := call(base, token, "PUT", fmt.Sprintf("/purchase/weigh-tickets/%d", id), map[string]any{
					"batch_no": codes[1], "gross_weight": 1000,
				}); err == nil {
					fail("batch_locked", fmt.Errorf("expected BATCH_NO_LOCKED"))
					failed++
				} else {
					ok("batch_locked")
				}
				if err := call(base, token, "POST", fmt.Sprintf("/purchase/weigh-tickets/%d/qc", id), map[string]any{
					"qc_result": "pass", "grade": "A",
				}); err != nil {
					fail("weigh_qc", err)
					failed++
				} else {
					ok("weigh_qc")
				}
				if err := call(base, token, "POST", fmt.Sprintf("/purchase/weigh-tickets/%d/confirm", id), map[string]any{
					"confirmed": true, "grade": "A",
				}); err != nil {
					warn("weigh_confirm", err)
				} else {
					ok("weigh_confirm")
				}
			}
		}
		var stockin map[string]any
		if err := callJSON(base, token, "POST", "/purchase/weigh-tickets", map[string]any{
			"receive_kind": "stockin", "batch_no": codes[1], "farmer_id": farmerID,
			"product_id": 1, "variety": "鲜木薯", "net_weight": 200, "bag_qty": 10,
			"cold_store_type": "fresh", "origin": "广西田东", "image_url": imgURL,
			"biz_date": bizDate, "source_type": "self",
		}, &stockin); err != nil {
			fail("weigh_create_stockin", err)
			failed++
		} else {
			ok("weigh_create_stockin")
		}
		// missing photo
		if err := call(base, token, "POST", "/purchase/weigh-tickets", map[string]any{
			"receive_kind": "gate", "batch_no": codes[2], "farmer_id": farmerID,
			"gross_weight": 100, "image_url": "mobile://fake",
		}); err == nil {
			fail("reject_fake_photo", fmt.Errorf("expected reject"))
			failed++
		} else {
			ok("reject_fake_photo")
		}
	} else {
		warn("weigh_create", fmt.Errorf("farmer=%d codes=%d img=%s", farmerID, len(codes), imgURL))
	}

	if failed > 0 {
		fmt.Printf("\nDELIVERY_SMOKE_FAIL count=%d\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nDELIVERY_SMOKE_OK")
}

func doLogin(base, login, pass string) (string, error) {
	var out struct {
		Code int `json:"code"`
		Msg  string
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := callJSON(base, "", "POST", "/auth/login", map[string]any{
		"login_name": login, "password": pass, "client_type": "mobile",
	}, &out.Data); err != nil {
		// callJSON expects data only when decoding into interface - redo
	}
	b, _ := json.Marshal(map[string]any{"login_name": login, "password": pass, "client_type": "mobile"})
	req, _ := http.NewRequest("POST", base+"/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return "", err
	}
	if env.Code != 1 {
		return "", fmt.Errorf("%s", env.Msg)
	}
	var data struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(env.Data, &data)
	if data.AccessToken == "" {
		return "", fmt.Errorf("empty token")
	}
	return data.AccessToken, nil
}

func call(base, token, method, path string, body map[string]any) error {
	return callJSON(base, token, method, path, body, nil)
}

func callJSON(base, token, method, path string, body map[string]any, into any) error {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, base+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("http_%d parse: %s", res.StatusCode, truncate(string(raw)))
	}
	if env.Code != 1 {
		return fmt.Errorf("%s", env.Msg)
	}
	if into != nil && len(env.Data) > 0 {
		_ = json.Unmarshal(env.Data, into)
	}
	return nil
}

func firstID(base, token, path, key string) int64 {
	var data map[string]any
	if err := callJSON(base, token, "GET", path, nil, &data); err != nil {
		return 0
	}
	list, _ := data["list"].([]any)
	if len(list) == 0 {
		return 0
	}
	row, _ := list[0].(map[string]any)
	return asInt(row[key])
}

func listCodes(gen map[string]any) []string {
	out := []string{}
	list, _ := gen["list"].([]any)
	for _, it := range list {
		m, _ := it.(map[string]any)
		if c, _ := m["code"].(string); c != "" {
			out = append(out, c)
		}
	}
	return out
}

func uploadSmokeImage(base, token string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "smoke.jpg")
	if err != nil {
		return "", err
	}
	// minimal valid-ish JPEG header + padding
	_, _ = part.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
	_, _ = part.Write([]byte("smoke-photo"))
	_ = w.Close()
	apiRoot := strings.TrimSuffix(base, "/api/v1")
	if apiRoot == base {
		apiRoot = strings.TrimSuffix(base, "/api/v1/")
	}
	url := base + "/biz/uploads"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("parse: %s", truncate(string(raw)))
	}
	if env.Code != 1 {
		return "", fmt.Errorf("%s", env.Msg)
	}
	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	u, _ := data["url"].(string)
	if u == "" {
		u, _ = data["file_url"].(string)
	}
	if u == "" {
		return "", fmt.Errorf("empty url")
	}
	_ = apiRoot
	return u, nil
}

func asInt(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		return 0
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func ok(name string)   { fmt.Printf("OK   %s\n", name) }
func warn(name string, err error) { fmt.Printf("WARN %s: %v\n", name, err) }
func fail(name string, err error) { fmt.Printf("FAIL %s: %v\n", name, err) }
func truncate(s string) string {
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
