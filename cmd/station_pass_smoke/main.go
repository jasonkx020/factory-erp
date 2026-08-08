package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func main() {
	base := env("API_BASE", "http://127.0.0.1:18080/api/v1")
	failed := 0

	pieceTok, err := login(base, "u_piece", "admin123", "mobile")
	if err != nil {
		fail("piece_login", err)
		failed++
	} else {
		ok("piece_login")
	}

	var reportID int64
	if pieceTok != "" {
		var resolved map[string]any
		if err := callJSON(base, pieceTok, "POST", "/production/scan/resolve", map[string]any{
			"badge_code": "EMP-PC",
			"box_code":   "BX-RAW-DEMO",
			"input_weight": 1000.0,
			"output_weight": 920.0,
		}, &resolved); err != nil {
			fail("scan_resolve", err)
			failed++
		} else {
			ok("scan_resolve")
		}

		var draft map[string]any
		if err := callJSON(base, pieceTok, "POST", "/production/scan", map[string]any{
			"badge_code":    "EMP-PC",
			"box_code":      "BX-RAW-DEMO",
			"input_weight":  1000.0,
			"output_weight": 920.0,
			"net_weight":    920.0,
		}, &draft); err != nil {
			fail("scan_submit", err)
			failed++
		} else {
			ok("scan_submit")
			st := fmt.Sprint(draft["status"])
			if st != "confirm_pending" {
				fail("scan_status", fmt.Errorf("want confirm_pending got %s", st))
				failed++
			} else {
				ok("scan_status")
			}
			reportID = asInt(draft["id"])
		}

		if reportID > 0 {
			var posted map[string]any
			if err := callJSON(base, pieceTok, "POST", fmt.Sprintf("/production/report-works/%d/confirm", reportID), map[string]any{
				"input_weight":      1000.0,
				"output_weight":   920.0,
				"process_qc_result": "pass",
			}, &posted); err != nil {
				fail("confirm_pass", err)
				failed++
			} else {
				ok("confirm_pass")
				if fmt.Sprint(posted["status"]) != "posted" {
					fail("confirm_status", fmt.Errorf("want posted"))
					failed++
				} else {
					ok("confirm_status")
				}
				loss := asFloat(posted["loss"])
				if loss < 0 || loss > 100 {
					fail("loss_calc", fmt.Errorf("loss=%v", loss))
					failed++
				} else {
					ok("loss_calc")
				}
			}

			// QC fail on new draft
			var draft2 map[string]any
			_ = callJSON(base, pieceTok, "POST", "/production/scan", map[string]any{
				"badge_code": "EMP-PC", "box_code": "BX-RAW-DEMO",
				"input_weight": 900.0, "output_weight": 880.0, "net_weight": 880.0,
			}, &draft2)
			id2 := asInt(draft2["id"])
			if id2 > 0 {
				if err := call(base, pieceTok, "POST", fmt.Sprintf("/production/report-works/%d/confirm", id2), map[string]any{
					"process_qc_result": "fail",
				}); err == nil {
					fail("qc_fail_block", fmt.Errorf("expected PROCESS_QC_FAIL"))
					failed++
				} else if !strings.Contains(err.Error(), "PROCESS_QC_FAIL") {
					fail("qc_fail_block", err)
					failed++
				} else {
					ok("qc_fail_block")
				}
			}
		}

		if err := call(base, pieceTok, "GET", "/production/piecework-summaries/mine?badge_code=EMP-PC", nil); err != nil {
			fail("piecework_mine", err)
			failed++
		} else {
			ok("piecework_mine")
		}

		if err := call(base, pieceTok, "GET", "/inventory/stock-txns", nil); err != nil {
			fail("stock_txns", err)
			failed++
		} else {
			ok("stock_txns")
		}
	}

	foremanTok, err := login(base, "u_foreman", "admin123", "web")
	if err != nil {
		fail("foreman_login", err)
		failed++
	} else {
		ok("foreman_login")
		if err := call(base, foremanTok, "POST", "/production/report-works", map[string]any{
			"process_id": 1, "worker_id": 2, "qty": 10,
		}); err == nil {
			fail("foreman_create_denied", fmt.Errorf("expected FIELD_INPUT_USE_APP"))
			failed++
		} else if !strings.Contains(err.Error(), "FIELD_INPUT_USE_APP") {
			fail("foreman_create_denied", err)
			failed++
		} else {
			ok("foreman_create_denied")
		}
	}

	adminTok, err := login(base, "admin", "admin123", "web")
	if err != nil {
		fail("admin_login", err)
		failed++
	} else {
		ok("admin_login")
		if err := call(base, adminTok, "POST", "/production/report-works", map[string]any{
			"process_id": 1, "worker_id": 2, "qty": 5,
		}); err == nil {
			fail("admin_backfill_no_reason", fmt.Errorf("expected reject without reason"))
			failed++
		} else {
			ok("admin_backfill_no_reason")
		}
		var back map[string]any
		if err := callJSON(base, adminTok, "POST", "/production/report-works", map[string]any{
			"process_id": 1, "worker_id": 2, "qty": 5, "backfill_reason": "smoke补单测试",
		}, &back); err != nil {
			fail("admin_backfill", err)
			failed++
		} else {
			ok("admin_backfill")
		}
	}

	if failed > 0 {
		fmt.Printf("\nSTATION_PASS_SMOKE_FAIL count=%d\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nSTATION_PASS_SMOKE_OK")
}

func login(base, user, pass, clientType string) (string, error) {
	b, _ := json.Marshal(map[string]any{"login_name": user, "password": pass, "client_type": clientType})
	req, _ := http.NewRequest("POST", base+"/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
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
		return fmt.Errorf("parse: %s", truncate(string(raw)))
	}
	if env.Code != 1 {
		return fmt.Errorf("%s", env.Msg)
	}
	if into != nil && len(env.Data) > 0 {
		_ = json.Unmarshal(env.Data, into)
	}
	return nil
}

func asInt(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	default:
		return 0
	}
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
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
func fail(name string, err error) { fmt.Printf("FAIL %s: %v\n", name, err) }
func truncate(s string) string {
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
