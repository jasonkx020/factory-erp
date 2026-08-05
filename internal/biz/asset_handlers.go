package biz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// EnsureAssetSchema creates factory-delivery fixed-asset tables (SQLite).
func EnsureAssetSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ast_fixed_asset_category (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  parent_id INTEGER,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`CREATE TABLE IF NOT EXISTS ast_fixed_asset (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category_id INTEGER,
  dept_id INTEGER,
  dept_name TEXT,
  location_text TEXT,
  original_value REAL,
  net_value REAL,
  status TEXT NOT NULL DEFAULT 'active',
  purchase_date TEXT,
  useful_life_months INTEGER,
  residual_rate REAL DEFAULT 0.05,
  remark TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  is_deleted INTEGER NOT NULL DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS ast_asset_transfer (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_no TEXT NOT NULL UNIQUE,
  asset_id INTEGER NOT NULL,
  from_dept_id INTEGER,
  to_dept_id INTEGER,
  from_dept_name TEXT,
  to_dept_name TEXT,
  from_location TEXT,
  to_location TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  remark TEXT,
  transferred_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`ALTER TABLE ast_fixed_asset ADD COLUMN dept_name TEXT`,
		`ALTER TABLE ast_fixed_asset ADD COLUMN useful_life_months INTEGER`,
		`ALTER TABLE ast_fixed_asset ADD COLUMN residual_rate REAL DEFAULT 0.05`,
		`ALTER TABLE ast_fixed_asset ADD COLUMN remark TEXT`,
		`ALTER TABLE ast_fixed_asset ADD COLUMN updated_at TEXT`,
		`ALTER TABLE ast_fixed_asset ADD COLUMN is_deleted INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE ast_fixed_asset_category ADD COLUMN remark TEXT`,
		`ALTER TABLE ast_asset_transfer ADD COLUMN from_dept_name TEXT`,
		`ALTER TABLE ast_asset_transfer ADD COLUMN to_dept_name TEXT`,
		`ALTER TABLE ast_asset_transfer ADD COLUMN remark TEXT`,
	}
	for _, stmt := range stmts {
		_, _ = db.Exec(stmt)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM ast_fixed_asset_category`).Scan(&n)
	if n == 0 {
		seeds := [][2]string{
			{"FAC-MACH", "生产设备"},
			{"FAC-TRANS", "运输工具"},
			{"FAC-BUILD", "房屋建筑物"},
			{"FAC-OFFICE", "办公设备"},
			{"FAC-OTHER", "其他固定资产"},
		}
		for _, s := range seeds {
			_, _ = db.Exec(`INSERT OR IGNORE INTO ast_fixed_asset_category(code, name) VALUES(?,?)`, s[0], s[1])
		}
	}
}

func (s *Services) handleAssetDomain(c *gin.Context, method, openapiPath, action string) bool {
	switch {
	case strings.HasPrefix(openapiPath, "/api/v1/asset/categories"):
		return s.handleAssetCategories(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/asset/fixed-assets"):
		return s.handleFixedAssets(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/asset/transfers"):
		return s.handleAssetTransfers(c, method, action)
	case strings.HasPrefix(openapiPath, "/api/v1/asset/stats"):
		return s.handleAssetStats(c)
	default:
		return false
	}
}

// ---------- categories ----------

func (s *Services) handleAssetCategories(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT id, code, name, parent_id, COALESCE(remark,''), COALESCE(created_at,'')
			FROM ast_fixed_asset_category ORDER BY id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var parent sql.NullInt64
			var code, name, remark, created string
			_ = rows.Scan(&id, &code, &name, &parent, &remark, &created)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name,
				"parent_id": parent.Int64, "remark": remark, "created_at": created,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list), "tree": buildCategoryTree(list)})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM ast_fixed_asset_category WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		code := strOrDef(body["code"], fmt.Sprintf("FAC%s", time.Now().Format("060102150405")))
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		parentID, _ := asInt64(body["parent_id"])
		res, err := s.DB.Exec(`INSERT INTO ast_fixed_asset_category(code, name, parent_id, remark) VALUES(?,?,?,?)`,
			code, name, nullInt(parentID), strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE ast_fixed_asset_category SET
			name=COALESCE(NULLIF(?,''),name),
			parent_id=COALESCE(NULLIF(?,0),parent_id),
			remark=COALESCE(NULLIF(?,''),remark)
			WHERE id=?`, strOr(body["name"]), nullInt64Or(body["parent_id"]), strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		return s.getSimpleRow(c, `SELECT * FROM ast_fixed_asset_category WHERE id=?`, id)
	case "delete":
		id := paramID(c)
		var cnt int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM ast_fixed_asset WHERE category_id=? AND COALESCE(is_deleted,0)=0`, id).Scan(&cnt)
		if cnt > 0 {
			api.FailJSON(c, "CATEGORY_IN_USE")
			return true
		}
		_, _ = s.DB.Exec(`DELETE FROM ast_fixed_asset_category WHERE id=?`, id)
		api.OK(c, gin.H{})
		return true
	}
	return true
}

func buildCategoryTree(flat []gin.H) []gin.H {
	byID := map[int64]gin.H{}
	for _, item := range flat {
		id, _ := asInt64(item["id"])
		cp := gin.H{}
		for k, v := range item {
			cp[k] = v
		}
		cp["children"] = []gin.H{}
		byID[id] = cp
	}
	roots := []gin.H{}
	for _, item := range flat {
		id, _ := asInt64(item["id"])
		pid, _ := asInt64(item["parent_id"])
		node := byID[id]
		if pid > 0 {
			if parent, ok := byID[pid]; ok {
				children, _ := parent["children"].([]gin.H)
				parent["children"] = append(children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	return roots
}

// ---------- fixed assets ----------

func (s *Services) handleFixedAssets(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		q := `SELECT a.id, a.code, a.name, a.category_id, COALESCE(c.name,''), COALESCE(c.code,''),
			a.dept_id, COALESCE(a.dept_name,''), COALESCE(a.location_text,''),
			a.original_value, a.net_value, a.status, COALESCE(a.purchase_date,''),
			a.useful_life_months, a.residual_rate, COALESCE(a.remark,''), a.created_at
			FROM ast_fixed_asset a
			LEFT JOIN ast_fixed_asset_category c ON c.id=a.category_id
			WHERE COALESCE(a.is_deleted,0)=0`
		args := []interface{}{}
		if cid := c.Query("category_id"); cid != "" {
			q += ` AND a.category_id=?`
			args = append(args, cid)
		}
		if st := c.Query("status"); st != "" {
			q += ` AND a.status=?`
			args = append(args, st)
		}
		if kw := c.Query("keyword"); kw != "" {
			q += ` AND (a.code LIKE ? OR a.name LIKE ? OR a.location_text LIKE ?)`
			like := "%" + kw + "%"
			args = append(args, like, like, like)
		}
		q += ` ORDER BY a.id DESC`
		rows, err := s.DB.Query(q, args...)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var catID, deptID, life sql.NullInt64
			var code, name, catName, catCode, deptName, loc, status, purchase, remark, created string
			var orig, net, residual sql.NullFloat64
			_ = rows.Scan(&id, &code, &name, &catID, &catName, &catCode, &deptID, &deptName, &loc,
				&orig, &net, &status, &purchase, &life, &residual, &remark, &created)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name,
				"category_id": catID.Int64, "category_name": catName, "category_code": catCode,
				"dept_id": deptID.Int64, "dept_name": deptName, "location_text": loc,
				"original_value": orig.Float64, "net_value": net.Float64,
				"status": status, "purchase_date": purchase,
				"useful_life_months": life.Int64, "residual_rate": residual.Float64,
				"remark": remark, "created_at": created,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "get":
		id := paramID(c)
		return s.getSimpleRow(c, `SELECT a.*, COALESCE(c.name,'') AS category_name
			FROM ast_fixed_asset a LEFT JOIN ast_fixed_asset_category c ON c.id=a.category_id
			WHERE a.id=? AND COALESCE(a.is_deleted,0)=0`, id)
	case "create":
		body := bindBody(c)
		code := strOrDef(body["code"], fmt.Sprintf("FA%s", time.Now().Format("060102150405")))
		name := strOr(body["name"])
		if name == "" {
			api.FailJSON(c, "NAME_REQUIRED")
			return true
		}
		orig, _ := asFloat(body["original_value"])
		net, ok := asFloat(body["net_value"])
		if !ok {
			net = orig
		}
		res, err := s.DB.Exec(`INSERT INTO ast_fixed_asset(
			code, name, category_id, dept_id, dept_name, location_text,
			original_value, net_value, status, purchase_date, useful_life_months, residual_rate, remark)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			code, name, nullInt64Or(body["category_id"]), nullInt64Or(body["dept_id"]),
			strOr(body["dept_name"]), strOr(body["location_text"]),
			orig, net, strOrDef(body["status"], "active"),
			strOrDef(body["purchase_date"], time.Now().Format("2006-01-02")),
			nullInt64Or(body["useful_life_months"]), nullFloat(body["residual_rate"]),
			strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "code": code, "name": name, "status": "active", "original_value": orig, "net_value": net})
		return true
	case "update", "replace":
		id := paramID(c)
		body := bindBody(c)
		_, err := s.DB.Exec(`UPDATE ast_fixed_asset SET
			name=COALESCE(NULLIF(?,''),name),
			category_id=COALESCE(NULLIF(?,0),category_id),
			dept_id=COALESCE(NULLIF(?,0),dept_id),
			dept_name=COALESCE(NULLIF(?,''),dept_name),
			location_text=COALESCE(NULLIF(?,''),location_text),
			original_value=COALESCE(?,original_value),
			net_value=COALESCE(?,net_value),
			status=COALESCE(NULLIF(?,''),status),
			purchase_date=COALESCE(NULLIF(?,''),purchase_date),
			useful_life_months=COALESCE(NULLIF(?,0),useful_life_months),
			residual_rate=COALESCE(?,residual_rate),
			remark=COALESCE(NULLIF(?,''),remark),
			updated_at=datetime('now')
			WHERE id=? AND COALESCE(is_deleted,0)=0`,
			strOr(body["name"]), nullInt64Or(body["category_id"]), nullInt64Or(body["dept_id"]),
			strOr(body["dept_name"]), strOr(body["location_text"]),
			nullFloat(body["original_value"]), nullFloat(body["net_value"]),
			strOr(body["status"]), strOr(body["purchase_date"]),
			nullInt64Or(body["useful_life_months"]), nullFloat(body["residual_rate"]),
			strOr(body["remark"]), id)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		api.OK(c, gin.H{"id": id})
		return true
	case "delete":
		id := paramID(c)
		_, _ = s.DB.Exec(`UPDATE ast_fixed_asset SET is_deleted=1, status='scrapped', updated_at=datetime('now') WHERE id=?`, id)
		api.OK(c, gin.H{})
		return true
	}
	return true
}

// ---------- transfers ----------

func (s *Services) handleAssetTransfers(c *gin.Context, method, action string) bool {
	_ = method
	switch action {
	case "list":
		rows, err := s.DB.Query(`SELECT t.id, t.doc_no, t.asset_id, a.code, a.name,
			t.from_dept_id, COALESCE(t.from_dept_name,''), t.to_dept_id, COALESCE(t.to_dept_name,''),
			COALESCE(t.from_location,''), COALESCE(t.to_location,''),
			t.status, COALESCE(t.remark,''), COALESCE(t.transferred_at,''), t.created_at
			FROM ast_asset_transfer t
			LEFT JOIN ast_fixed_asset a ON a.id=t.asset_id
			ORDER BY t.id DESC`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id, assetID int64
			var fromDept, toDept sql.NullInt64
			var docNo, assetCode, assetName, fromDN, toDN, fromLoc, toLoc, status, remark, transferred, created string
			_ = rows.Scan(&id, &docNo, &assetID, &assetCode, &assetName,
				&fromDept, &fromDN, &toDept, &toDN, &fromLoc, &toLoc, &status, &remark, &transferred, &created)
			list = append(list, gin.H{
				"id": id, "doc_no": docNo, "asset_id": assetID,
				"asset_code": assetCode, "asset_name": assetName,
				"from_dept_id": fromDept.Int64, "from_dept_name": fromDN,
				"to_dept_id": toDept.Int64, "to_dept_name": toDN,
				"from_location": fromLoc, "to_location": toLoc,
				"status": status, "remark": remark, "transferred_at": transferred, "created_at": created,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
		return true
	case "get":
		return s.getSimpleRow(c, `SELECT * FROM ast_asset_transfer WHERE id=?`, paramID(c))
	case "create":
		body := bindBody(c)
		assetID, _ := asInt64(body["asset_id"])
		if assetID == 0 {
			api.FailJSON(c, "ASSET_REQUIRED")
			return true
		}
		var curDept sql.NullInt64
		var curDeptName, curLoc string
		_ = s.DB.QueryRow(`SELECT dept_id, COALESCE(dept_name,''), COALESCE(location_text,'') FROM ast_fixed_asset WHERE id=? AND COALESCE(is_deleted,0)=0`,
			assetID).Scan(&curDept, &curDeptName, &curLoc)
		docNo := strOrDef(body["doc_no"], fmt.Sprintf("AT%s", time.Now().Format("060102150405")))
		fromDept, ok := asInt64(body["from_dept_id"])
		if !ok {
			fromDept = curDept.Int64
		}
		fromDeptName := strOrDef(body["from_dept_name"], curDeptName)
		fromLoc := strOrDef(body["from_location"], curLoc)
		toDept, _ := asInt64(body["to_dept_id"])
		toDeptName := strOr(body["to_dept_name"])
		toLoc := strOr(body["to_location"])
		if toLoc == "" && toDeptName == "" && toDept == 0 {
			api.FailJSON(c, "TO_LOCATION_REQUIRED")
			return true
		}
		res, err := s.DB.Exec(`INSERT INTO ast_asset_transfer(
			doc_no, asset_id, from_dept_id, to_dept_id, from_dept_name, to_dept_name,
			from_location, to_location, status, remark)
			VALUES(?,?,?,?,?,?,?,?,'draft',?)`,
			docNo, assetID, nullInt(fromDept), nullInt(toDept), fromDeptName, toDeptName, fromLoc, toLoc, strOr(body["remark"]))
		if err != nil {
			api.FailJSON(c, "DB_ERROR:"+err.Error())
			return true
		}
		id, _ := res.LastInsertId()
		api.OK(c, gin.H{"id": id, "doc_no": docNo, "asset_id": assetID, "status": "draft"})
		return true
	case "update", "replace":
		id := paramID(c)
		var status string
		_ = s.DB.QueryRow(`SELECT status FROM ast_asset_transfer WHERE id=?`, id).Scan(&status)
		if status != "draft" {
			api.FailJSON(c, "DOC_LOCKED")
			return true
		}
		body := bindBody(c)
		_, _ = s.DB.Exec(`UPDATE ast_asset_transfer SET
			to_dept_id=COALESCE(NULLIF(?,0),to_dept_id),
			to_dept_name=COALESCE(NULLIF(?,''),to_dept_name),
			to_location=COALESCE(NULLIF(?,''),to_location),
			remark=COALESCE(NULLIF(?,''),remark)
			WHERE id=?`, nullInt64Or(body["to_dept_id"]), strOr(body["to_dept_name"]),
			strOr(body["to_location"]), strOr(body["remark"]), id)
		return s.getSimpleRow(c, `SELECT * FROM ast_asset_transfer WHERE id=?`, id)
	case "action:confirm":
		return s.confirmAssetTransfer(c)
	}
	return true
}

func (s *Services) confirmAssetTransfer(c *gin.Context) bool {
	id := paramID(c)
	var assetID int64
	var status, toLoc, toDeptName string
	var toDept sql.NullInt64
	err := s.DB.QueryRow(`SELECT asset_id, status, COALESCE(to_location,''), COALESCE(to_dept_name,''), to_dept_id
		FROM ast_asset_transfer WHERE id=?`, id).Scan(&assetID, &status, &toLoc, &toDeptName, &toDept)
	if err == sql.ErrNoRows {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	if status == "confirmed" {
		return s.getSimpleRow(c, `SELECT * FROM ast_asset_transfer WHERE id=?`, id)
	}
	if status != "draft" && status != "submitted" {
		api.FailJSON(c, "DOC_LOCKED")
		return true
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	tx, err := s.DB.Begin()
	if err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.Exec(`UPDATE ast_fixed_asset SET
		dept_id=?, dept_name=?, location_text=?, updated_at=?
		WHERE id=?`, nullInt(toDept.Int64), toDeptName, toLoc, now, assetID)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	_, err = tx.Exec(`UPDATE ast_asset_transfer SET status='confirmed', transferred_at=? WHERE id=?`, now, id)
	if err != nil {
		api.FailJSON(c, "DB_ERROR:"+err.Error())
		return true
	}
	if err := tx.Commit(); err != nil {
		api.FailJSON(c, "DB_ERROR")
		return true
	}
	api.OK(c, gin.H{"id": id, "status": "confirmed", "asset_id": assetID, "transferred_at": now})
	return true
}

// ---------- stats ----------

func (s *Services) handleAssetStats(c *gin.Context) bool {
	var totalCount int
	var totalOrig, totalNet float64
	_ = s.DB.QueryRow(`SELECT COUNT(1), COALESCE(SUM(original_value),0), COALESCE(SUM(net_value),0)
		FROM ast_fixed_asset WHERE COALESCE(is_deleted,0)=0 AND status!='scrapped'`).Scan(&totalCount, &totalOrig, &totalNet)

	byCat := []gin.H{}
	rows, err := s.DB.Query(`SELECT COALESCE(c.id,0), COALESCE(c.code,'未分类'), COALESCE(c.name,'未分类'),
		COUNT(1), COALESCE(SUM(a.original_value),0), COALESCE(SUM(a.net_value),0)
		FROM ast_fixed_asset a
		LEFT JOIN ast_fixed_asset_category c ON c.id=a.category_id
		WHERE COALESCE(a.is_deleted,0)=0 AND a.status!='scrapped'
		GROUP BY COALESCE(c.id,0), COALESCE(c.code,'未分类'), COALESCE(c.name,'未分类')
		ORDER BY COALESCE(SUM(a.original_value),0) DESC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int64
			var code, name string
			var cnt int
			var orig, net float64
			_ = rows.Scan(&cid, &code, &name, &cnt, &orig, &net)
			byCat = append(byCat, gin.H{
				"category_id": cid, "category_code": code, "category_name": name,
				"count": cnt, "original_value": orig, "net_value": net,
			})
		}
	}

	byDept := []gin.H{}
	drows, err := s.DB.Query(`SELECT COALESCE(dept_id,0), COALESCE(NULLIF(dept_name,''),'未分配'),
		COUNT(1), COALESCE(SUM(original_value),0), COALESCE(SUM(net_value),0)
		FROM ast_fixed_asset
		WHERE COALESCE(is_deleted,0)=0 AND status!='scrapped'
		GROUP BY COALESCE(dept_id,0), COALESCE(NULLIF(dept_name,''),'未分配')
		ORDER BY COALESCE(SUM(original_value),0) DESC`)
	if err == nil {
		defer drows.Close()
		for drows.Next() {
			var deptID int64
			var deptName string
			var cnt int
			var orig, net float64
			_ = drows.Scan(&deptID, &deptName, &cnt, &orig, &net)
			byDept = append(byDept, gin.H{
				"dept_id": deptID, "dept_name": deptName,
				"count": cnt, "original_value": orig, "net_value": net,
			})
		}
	}

	byStatus := []gin.H{}
	srows, _ := s.DB.Query(`SELECT status, COUNT(1), COALESCE(SUM(original_value),0), COALESCE(SUM(net_value),0)
		FROM ast_fixed_asset WHERE COALESCE(is_deleted,0)=0 GROUP BY status`)
	if srows != nil {
		defer srows.Close()
		for srows.Next() {
			var st string
			var cnt int
			var orig, net float64
			_ = srows.Scan(&st, &cnt, &orig, &net)
			byStatus = append(byStatus, gin.H{"status": st, "count": cnt, "original_value": orig, "net_value": net})
		}
	}

	var transferDraft, transferDone int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM ast_asset_transfer WHERE status='draft'`).Scan(&transferDraft)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM ast_asset_transfer WHERE status='confirmed'`).Scan(&transferDone)

	api.OK(c, gin.H{
		"summary": gin.H{
			"asset_count": totalCount, "original_value": totalOrig, "net_value": totalNet,
			"depreciation_value": totalOrig - totalNet,
			"transfer_draft":    transferDraft, "transfer_confirmed": transferDone,
		},
		"by_category": byCat,
		"by_dept":     byDept,
		"by_status":   byStatus,
		"list": []gin.H{{
			"asset_count": totalCount, "original_value": totalOrig, "net_value": totalNet,
		}},
		"total": 1,
	})
	return true
}
