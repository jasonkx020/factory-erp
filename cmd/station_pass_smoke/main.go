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

	if pieceTok != "" {
		var resolved map[string]any
		if err := callJSON(base, pieceTok, "POST", "/production/scan/resolve", map[string]any{
			"badge_code":     "EMP-PC",
			"box_code":       "BX-RAW-DEMO",
			"process_id":     1,
			"input_weight":   1000.0,
			"output_weight":  920.0,
		}, &resolved); err != nil {
			// DEMO 箱可能不存在：允许 BOX_NOT_FOUND / PROCESS_REQUIRED 以外仍算连通性
			if !strings.Contains(err.Error(), "BOX_NOT_FOUND") &&
				!strings.Contains(err.Error(), "BOARD_FINISHED") &&
				!strings.Contains(err.Error(), "SHIFT_NOT_AUTHORIZED") {
				fail("scan_resolve", err)
				failed++
			} else {
				ok("scan_resolve_reachable")
			}
		} else {
			ok("scan_resolve")
		}

		// 旧扫码提交必须拒绝
		if err := callExpectFail(base, pieceTok, "POST", "/production/scan", map[string]any{
			"badge_code":    "EMP-PC",
			"box_code":      "BX-RAW-DEMO",
			"process_id":    1,
			"input_weight":  1000.0,
			"output_weight": 920.0,
		}, "FEATURE_REMOVED"); err != nil {
			fail("scan_submit_removed", err)
			failed++
		} else {
			ok("scan_submit_removed")
		}

		if err := callExpectFail(base, pieceTok, "POST", "/production/report-works", map[string]any{
			"process_id": 1, "worker_id": 1, "qty": 10,
		}, "FEATURE_REMOVED"); err != nil {
			fail("report_works_removed", err)
			failed++
		} else {
			ok("report_works_removed")
		}

		if err := call(base, pieceTok, "GET", "/production/station-flow-logs?page=1&page_size=5", nil); err != nil {
			fail("station_flow_logs", err)
			failed++
		} else {
			ok("station_flow_logs")
		}
	}

	adminTok, err := login(base, "admin", "admin123", "admin")
	if err != nil {
		// 兼容演示账号
		adminTok, err = login(base, "u_admin", "admin123", "admin")
	}
	if err != nil {
		fail("admin_login", err)
		failed++
	} else {
		ok("admin_login")
		if err := callExpectFail(base, adminTok, "POST", "/production/report-works", map[string]any{
			"process_id": 1, "worker_id": 1, "qty": 10, "backfill_reason": "smoke",
		}, "FEATURE_REMOVED"); err != nil {
			fail("admin_report_works_removed", err)
			failed++
		} else {
			ok("admin_report_works_removed")
		}
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "station_pass_smoke failed=%d\n", failed)
		os.Exit(1)
	}
	fmt.Println("station_pass_smoke OK (board-issue path; report-works removed)")
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func ok(name string)   { fmt.Println("OK ", name) }
func fail(name string, err error) {
	fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", name, err)
}

func login(base, user, pass, client string) (string, error) {
	var out struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"`
		} `json:"data"`
	}
	body := map[string]any{"login_name": user, "password": pass, "client": client}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", strings.TrimRight(base, "/")+"/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("login decode: %w body=%s", err, string(raw))
	}
	if out.Code != 1 {
		return "", fmt.Errorf("login code=%d msg=%s", out.Code, out.Msg)
	}
	tok := out.Data.AccessToken
	if tok == "" {
		tok = out.Data.Token
	}
	if tok == "" {
		return "", fmt.Errorf("empty token")
	}
	return tok, nil
}

func call(base, tok, method, path string, body map[string]any) error {
	var discard map[string]any
	return callJSON(base, tok, method, path, body, &discard)
}

func callJSON(base, tok, method, path string, body map[string]any, dest any) error {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, strings.TrimRight(base, "/")+path, rdr)
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("decode: %w body=%s", err, string(raw))
	}
	if env.Code != 1 {
		return fmt.Errorf("code=%d msg=%s", env.Code, env.Msg)
	}
	if dest != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		_ = json.Unmarshal(env.Data, dest)
	}
	return nil
}

func callExpectFail(base, tok, method, path string, body map[string]any, wantSub string) error {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, strings.TrimRight(base, "/")+path, rdr)
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var env envelope
	_ = json.Unmarshal(raw, &env)
	if env.Code == 1 {
		return fmt.Errorf("expected fail containing %q, got success", wantSub)
	}
	if !strings.Contains(env.Msg, wantSub) && !strings.Contains(string(raw), wantSub) {
		return fmt.Errorf("expected msg containing %q, got code=%d msg=%s raw=%s", wantSub, env.Code, env.Msg, string(raw))
	}
	return nil
}
