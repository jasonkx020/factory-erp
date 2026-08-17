package biz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsureProductExtSchema creates factory-delivery product satellite tables (SQLite).
func EnsureProductExtSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS prd_product_spec (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER NOT NULL,
  spec_code TEXT NOT NULL,
  routing_id INTEGER,
  process_wage_bind_json TEXT,
  remark TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT (NOW()),
  updated_at TEXT NOT NULL DEFAULT (NOW()),
  is_deleted INTEGER NOT NULL DEFAULT 0,
  UNIQUE(product_id, spec_code)
)`,
		`CREATE TABLE IF NOT EXISTS prd_product_app_sort (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER NOT NULL,
  channel TEXT NOT NULL DEFAULT 'app',
  sort_no INTEGER NOT NULL DEFAULT 0,
  is_visible INTEGER NOT NULL DEFAULT 1,
  UNIQUE(product_id, channel)
)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			fmt.Printf("[product-ext] schema warn: %v\n", err)
		}
	}
	// seed app sort for existing active products if empty
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM prd_product_app_sort`).Scan(&n)
	if n == 0 {
		_, _ = db.Exec(`INSERT INTO prd_product_app_sort(product_id, channel, sort_no, is_visible)
			SELECT id, 'app', id * 10, 1 FROM prd_product WHERE COALESCE(is_deleted,0)=0`)
	}
}

func (s *Services) handleProductDomain(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.Contains(openapiPath, "/units"):
		return s.handleProductUnits(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/product/app-sorts"):
		return s.handleProductAppSorts(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/product/specs"):
		return s.handleProductSpecs(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/product/products"):
		return s.handleProducts(c, method, action)
	default:
		return false
	}
}

// ---------- units ----------

func (s *Services) handleProductUnits(c *gin.Context, method, action string) bool {
	pid := paramID(c)
	if pid == 0 {
		api.FailJSON(c, "PRODUCT_REQUIRED")
		return true
	}
	switch {
	case action == "list" || method == "GET":
		rows, err := s.DB.Query(`SELECT id, product_id, unit_name, is_base, factor_to_base, is_purchase, is_sale, is_stock, created_at
			FROM prd_product_unit WHERE product_id=? ORDER BY is_base DESC, id`, pid)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, productID int64
			var unitName, created string
			var isBase, isPur, isSale, isStock int
			var factor float64
			_ = rows.Scan(&id, &productID, &unitName, &isBase, &factor, &isPur, &isSale, &isStock, &created)
			list = append(list, gin.H{
				"id": id, "product_id": productID, "unit_name": unitName,
				"is_base": isBase == 1, "factor_to_base": factor,
				"is_purchase": isPur == 1, "is_sale": isSale == 1, "is_stock": isStock == 1,
				"created_at": created,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list), "product_id": pid})
		return true
	case action == "replace" || method == "PUT":
		body := bindBody(c)
		units, _ := body["units"].([]interface{})
		if len(units) == 0 {
			// allow single unit payload
			if name := strOr(body["unit_name"]); name != "" {
				units = []interface{}{body}
			}
		}
		tx, err := s.DB.Begin()
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer func() { _ = tx.Rollback() }()
		_, _ = tx.Exec(`DELETE FROM prd_product_unit WHERE product_id=?`, pid)
		hasBase := false
		for _, u := range units {
			m, _ := u.(map[string]interface{})
			if m == nil {
				continue
			}
			name := strOr(m["unit_name"])
			if name == "" {
				continue
			}
			factor, _ := asFloat(m["factor_to_base"])
			if factor <= 0 {
				factor = 1
			}
			isBase := boolInt(m["is_base"])
			if isBase == 1 {
				if hasBase {
					isBase = 0
				} else {
					hasBase = true
					factor = 1
				}
			}
			_, err = tx.Exec(`INSERT INTO prd_product_unit(product_id, unit_name, is_base, factor_to_base, is_purchase, is_sale, is_stock)
				VALUES(?,?,?,?,?,?,?)`,
				pid, name, isBase, factor, boolIntDef(m["is_purchase"], 1), boolIntDef(m["is_sale"], 1), boolIntDef(m["is_stock"], 1))
			if err != nil {
				api.FailJSON(c, "DB_ERROR:"+err.Error())
				return true
			}
		}
		if !hasBase && len(units) > 0 {
			_, _ = tx.Exec(`UPDATE prd_product_unit SET is_base=1, factor_to_base=1 WHERE product_id=? AND id=(
				SELECT id FROM prd_product_unit WHERE product_id=? ORDER BY id LIMIT 1)`, pid, pid)
		}
		if err := tx.Commit(); err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		// return refreshed list
		return s.handleProductUnits(c, "GET", "list")
	}
	return true
}

func boolIntDef(v interface{}, def int) int {
	if v == nil {
		return def
	}
	return boolInt(v)
}

// ---------- app sorts ----------

func (s *Services) handleProductAppSorts(c *gin.Context, method, action string) bool {
	switch {
	case action == "list" || method == "GET":
		channel := c.Query("channel")
		if channel == "" {
			channel = "app"
		}
		rows, err := s.DB.Query(`
SELECT s.id, s.product_id, p.code, p.name, p.product_type, p.status,
  s.channel, s.sort_no, s.is_visible
FROM prd_product_app_sort s
JOIN prd_product p ON p.id=s.product_id AND COALESCE(p.is_deleted,0)=0
WHERE s.channel=?
ORDER BY s.sort_no ASC, s.id ASC`, channel)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, pid int64
			var code, name, typ, status, ch string
			var sortNo, visible int
			_ = rows.Scan(&id, &pid, &code, &name, &typ, &status, &ch, &sortNo, &visible)
			list = append(list, gin.H{
				"id": id, "product_id": pid, "product_code": code, "product_name": name,
				"product_type": typ, "product_status": status,
				"channel": ch, "sort_no": sortNo, "is_visible": visible == 1,
			})
		}
		// also include products without sort row
		orows, _ := s.DB.Query(`
SELECT p.id, p.code, p.name, p.product_type, p.status FROM prd_product p
WHERE COALESCE(p.is_deleted,0)=0 AND p.id NOT IN (
  SELECT product_id FROM prd_product_app_sort WHERE channel=?
) ORDER BY p.id`, channel)
		if orows != nil {
			defer orows.Close()
			for orows.Next() {
				var pid int64
				var code, name, typ, status string
				_ = orows.Scan(&pid, &code, &name, &typ, &status)
				list = append(list, gin.H{
					"id": 0, "product_id": pid, "product_code": code, "product_name": name,
					"product_type": typ, "product_status": status,
					"channel": channel, "sort_no": 9999, "is_visible": true, "unsaved": true,
				})
			}
		}
		api.OK(c, gin.H{"list": list, "total": len(list), "channel": channel})
		return true
	case action == "replace" || method == "PUT":
		body := bindBody(c)
		channel := strOrDef(body["channel"], "app")
		items, _ := body["items"].([]interface{})
		if len(items) == 0 {
			items, _ = body["list"].([]interface{})
		}
		if len(items) == 0 {
			// single item upsert
			if pid, ok := asInt64(body["product_id"]); ok && pid > 0 {
				items = []interface{}{body}
			}
		}
		tx, err := s.DB.Begin()
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		defer func() { _ = tx.Rollback() }()
		for _, it := range items {
			m, _ := it.(map[string]interface{})
			if m == nil {
				continue
			}
			pid, _ := asInt64(m["product_id"])
			if pid == 0 {
				continue
			}
			sortNo, _ := asInt64(m["sort_no"])
			vis := boolIntDef(m["is_visible"], 1)
			ch := strOrDef(m["channel"], channel)
			var existID int64
			_ = tx.QueryRow(`SELECT id FROM prd_product_app_sort WHERE product_id=? AND channel=?`, pid, ch).Scan(&existID)
			if existID > 0 {
				_, _ = tx.Exec(`UPDATE prd_product_app_sort SET sort_no=?, is_visible=? WHERE id=?`, sortNo, vis, existID)
			} else {
				_, _ = tx.Exec(`INSERT INTO prd_product_app_sort(product_id, channel, sort_no, is_visible) VALUES(?,?,?,?)`,
					pid, ch, sortNo, vis)
			}
		}
		if err := tx.Commit(); err != nil {
			api.FailJSON(c, "DB_ERROR")
			return true
		}
		api.OK(c, gin.H{"ok": true, "channel": channel, "count": len(items)})
		return true
	}
	return true
}

// ---------- specs ----------

func (s *Services) handleProductSpecs(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		pid := c.Query("product_id")
		sqlStr := `SELECT s.id, s.product_id, p.code, p.name, s.spec_code, s.routing_id, s.process_wage_bind_json,
			COALESCE(s.remark,''), s.status, s.created_at
			FROM prd_product_spec s
			JOIN prd_product p ON p.id=s.product_id
			WHERE COALESCE(s.is_deleted,0)=0`
		args := []interface{}{}
		if pid != "" {
			sqlStr += ` AND s.product_id=?`
			args = append(args, pid)
		}
		sqlStr += ` ORDER BY s.id DESC`
		rows, err := s.DB.Query(sqlStr, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, productID int64
			var routingID sql.NullInt64
			var code, name, specCode, wageJSON, remark, status, created string
			_ = rows.Scan(&id, &productID, &code, &name, &specCode, &routingID, &wageJSON, &remark, &status, &created)
			item := gin.H{
				"id": id, "product_id": productID, "product_code": code, "product_name": name,
				"spec_code": specCode, "routing_id": routingID.Int64, "remark": remark,
				"status": status, "created_at": created,
			}
			if wageJSON != "" {
				var wage interface{}
				if json.Unmarshal([]byte(wageJSON), &wage) == nil {
					item["process_wage_bind"] = wage
					item["process_wage_bind_json"] = wageJSON
				} else {
					item["process_wage_bind_json"] = wageJSON
				}
			}
			list = append(list, item)
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM prd_product_spec WHERE id=? AND COALESCE(is_deleted,0)=0`, paramID(c))
	case "create":
		body := bindBody(c)
		productID, _ := asInt64(body["product_id"])
		specCode := strOr(body["spec_code"])
		if productID == 0 || specCode == "" {
			api.FailJSON(c, "PRODUCT_SPEC_REQUIRED")
			return true
		}
		routingID, _ := asInt64(body["routing_id"])
		wageJSON := wageBindJSON(body)
		res, err := s.DB.Exec(`INSERT INTO prd_product_spec(product_id, spec_code, routing_id, process_wage_bind_json, remark, status)
			VALUES(?,?,?,?,?,'active')`, productID, specCode, nullInt(routingID), wageJSON, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "product_id": productID, "spec_code": specCode, "status": "active"})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		wageJSON := wageBindJSON(body)
		_, err := s.DB.Exec(`UPDATE prd_product_spec SET
			spec_code=COALESCE(NULLIF(?,''),spec_code),
			routing_id=COALESCE(NULLIF(?,0),routing_id),
			process_wage_bind_json=COALESCE(NULLIF(?,''),process_wage_bind_json),
			remark=COALESCE(NULLIF(?,''),remark),
			status=COALESCE(NULLIF(?,''),status),
			updated_at=NOW()
			WHERE id=? AND COALESCE(is_deleted,0)=0`,
			strOr(body["spec_code"]), nullInt64Or(body["routing_id"]), wageJSON, strOr(body["remark"]), strOr(body["status"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		return s.getSimpleRow(c, `SELECT * FROM prd_product_spec WHERE id=?`, id)
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE prd_product_spec SET is_deleted=1, updated_at=NOW() WHERE id=?`, id)
		api.OK(c, gin.H{})
		return true
	}
	return true
}

func wageBindJSON(body map[string]interface{}) string {
	if raw, ok := body["process_wage_bind_json"].(string); ok && raw != "" {
		return raw
	}
	if v, ok := body["process_wage_bind"]; ok && v != nil {
		b, err := json.Marshal(v)
		if err == nil {
			return string(b)
		}
	}
	// convenience: process_id + wage
	if pid, ok := asInt64(body["process_id"]); ok && pid > 0 {
		wage, _ := asFloat(body["wage"])
		b, _ := json.Marshal([]gin.H{{"process_id": pid, "wage": wage}})
		return string(b)
	}
	return ""
}