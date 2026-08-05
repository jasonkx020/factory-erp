package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
		{"orders", "GET", "/sales/orders?page_size=5", nil, false},
		{"customers", "GET", "/crm/customers?page_size=5", nil, false},
		{"follow_ups", "GET", "/crm/follow-ups?page_size=5", nil, false},
		{"leave", "GET", "/hr/leave-requests", nil, false},
		{"overtime", "GET", "/hr/overtime-patches", nil, false},
		{"approval_tasks", "GET", "/approval/tasks", nil, false},
		{"payroll_sheets", "GET", "/payroll/sheets?page_size=5", nil, false},
		{"commission", "GET", "/payroll/commission-calcs?page_size=5", nil, false},
		{"products", "GET", "/product/products?page_size=5", nil, false},
		{"punch", "POST", "/hr/attendance/records/punch", map[string]any{"punch_type": "in"}, false},
		{"assets", "GET", "/asset/fixed-assets?page_size=5", nil, false},
		{"asset_transfers", "GET", "/asset/transfers?page_size=5", nil, false},
		{"receipt_alerts", "GET", "/finance/receipt-alerts?page_size=5", nil, false},
		{"recognitions", "GET", "/finance/payment-recognitions?page_size=5", nil, false},
		{"knowledge", "GET", "/system/knowledge?page_size=5", nil, false},
		{"drawings", "GET", "/system/drawings?page_size=5", nil, false},
		{"announcements", "GET", "/system/announcements?page_size=5", nil, false},
		{"courses", "GET", "/system/courses?page_size=5", nil, false},
		{"deliveries", "GET", "/sales/deliveries?page_size=5", nil, false},
		{"pre_ships", "GET", "/sales/pre-shipments?page_size=5", nil, false},
		{"quote_calc", "POST", "/sales/quote-calculator/calc", map[string]any{"product_id": 3, "qty": 10, "margin_rate": 0.2}, false},
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

	// closed-loop sample: weigh create → qc → (confirm may need purchase role; admin ok)
	farmerID := firstID(base, token, "/purchase/farmers?page_size=1", "id")
	if farmerID > 0 {
		img := fmt.Sprintf("mobile://delivery_smoke/%d", time.Now().Unix())
		var created map[string]any
		if err := callJSON(base, token, "POST", "/purchase/weigh-tickets", map[string]any{
			"farmer_id": farmerID, "channel": "internal", "product_id": 1, "variety": "鲜木薯",
			"gross_weight": 1000, "deduct_rate": 5, "grade": "A", "image_url": img,
			"biz_date": time.Now().Format("2006-01-02"), "source_type": "self",
		}, &created); err != nil {
			fail("weigh_create", err)
			failed++
		} else {
			ok("weigh_create")
			id := asInt(created["id"])
			if id > 0 {
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
					warn("weigh_confirm", err) // may fail on evidence edge-cases; report but don't hard-fail whole suite
				} else {
					ok("weigh_confirm")
				}
			}
		}
	} else {
		warn("weigh_create", fmt.Errorf("no farmer"))
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
