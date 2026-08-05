package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ginRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// CheckCoverage compares OpenAPI ops vs gin_routes.json (or live health/routes).
// Returns missing ops and prints summary; non-nil error if incomplete or IO failure.
func CheckCoverage(root string) error {
	opsPath := filepath.Join(root, "scripts", "openapi_ops.json")
	routesPath := filepath.Join(root, "scripts", "gin_routes.json")

	raw, err := os.ReadFile(opsPath)
	if err != nil {
		return fmt.Errorf("missing openapi_ops.json — run: go run ./cmd/erp-tools gen-routes (%w)", err)
	}
	var ops []Op
	if err := json.Unmarshal(raw, &ops); err != nil {
		return err
	}

	ginSet, err := loadGinRoutes(routesPath)
	if err != nil {
		return err
	}

	var missing []Op
	for _, o := range ops {
		key := o.Method + "\t" + o.Path
		if _, ok := ginSet[key]; !ok {
			missing = append(missing, o)
		}
	}
	covered := len(ops) - len(missing)
	pct := 0.0
	if len(ops) > 0 {
		pct = 100.0 * float64(covered) / float64(len(ops))
	}
	fmt.Printf("coverage %d/%d (%.1f%%)\n", covered, len(ops), pct)
	if len(missing) > 0 {
		fmt.Printf("missing %d:\n", len(missing))
		limit := len(missing)
		if limit > 40 {
			limit = 40
		}
		for _, m := range missing[:limit] {
			fmt.Printf("  %s %s\n", m.Method, m.Path)
		}
		if len(missing) > 40 {
			fmt.Printf("  ... %d more\n", len(missing)-40)
		}
		return fmt.Errorf("openapi coverage incomplete")
	}
	fmt.Println("OK: all OpenAPI operations registered")
	return nil
}

func loadGinRoutes(routesPath string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if raw, err := os.ReadFile(routesPath); err == nil {
		var data []ginRoute
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		for _, r := range data {
			p := NormGinToOpenAPI(r.Path)
			out[strings.ToUpper(r.Method)+"\t"+p] = struct{}{}
		}
		return out, nil
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:18080/api/v1/health/routes")
	if err != nil {
		return nil, fmt.Errorf("cannot load gin routes: %w (start erp-api or write scripts/gin_routes.json)", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var wrap map[string]interface{}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	var items []interface{}
	if data, ok := wrap["data"]; ok {
		switch d := data.(type) {
		case []interface{}:
			items = d
		case map[string]interface{}:
			if routes, ok := d["routes"].([]interface{}); ok {
				items = routes
			}
		}
	} else if routes, ok := wrap["routes"].([]interface{}); ok {
		items = routes
	}
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		method, _ := m["method"].(string)
		path, _ := m["path"].(string)
		out[strings.ToUpper(method)+"\t"+NormGinToOpenAPI(path)] = struct{}{}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty gin routes from live endpoint")
	}
	return out, nil
}
