package biz

import (
	"database/sql"
	"strings"
)

// NormalizeWarehouseRole maps cold_store_type / legacy aliases → warehouse_role.
func NormalizeWarehouseRole(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "fresh", "raw", "原料", "保鲜":
		return "raw"
	case "semi", "半成品":
		return "semi"
	case "fg", "finished", "成品":
		return "fg"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

// ResolveWarehouseByStoreType looks up warehouse by role/type from master data.
// Prefer explicit warehouse_id on weigh variety; then warehouse_role; then warehouse_type.
// Returns 0 when not configured (callers must fail closed — no silent id=1/2/3).
func (s *Services) ResolveWarehouseByStoreType(kind string) int64 {
	if s == nil || s.DB == nil {
		return 0
	}
	role := NormalizeWarehouseRole(kind)
	if role == "" {
		return 0
	}
	var id int64
	_ = s.DB.QueryRow(`SELECT id FROM inv_warehouse
		WHERE COALESCE(is_deleted,0)=0 AND status='active'
		  AND (LOWER(COALESCE(warehouse_role,''))=? OR LOWER(COALESCE(warehouse_type,''))=?)
		ORDER BY id ASC LIMIT 1`, role, role).Scan(&id)
	return id
}

// ResolveWarehouseForWeighVariety resolves warehouse from variety row fields + cold_store_type.
func (s *Services) ResolveWarehouseForWeighVariety(varietyWarehouseID int64, varietyRole, coldStore string) int64 {
	if varietyWarehouseID > 0 {
		return varietyWarehouseID
	}
	if r := NormalizeWarehouseRole(varietyRole); r != "" {
		if id := s.ResolveWarehouseByStoreType(r); id > 0 {
			return id
		}
	}
	return s.ResolveWarehouseByStoreType(coldStore)
}

// ColdStoreWarehouse maps cold_store_type → warehouse_id via configured warehouse roles.
// Deprecated name kept for call sites; returns 0 when unconfigured.
func ColdStoreWarehouse(kind string) int64 {
	// Legacy package-level helper cannot see DB — prefer Services.ResolveWarehouseByStoreType.
	// Keep numeric fallback ONLY when no Services context (tests); production call sites updated.
	_ = kind
	return 0
}

func ColdStoreWarehouseDB(db *sql.DB, kind string) int64 {
	s := &Services{DB: db}
	return s.ResolveWarehouseByStoreType(kind)
}

func EnsureDefaultPlant(db *sql.DB) int64 {
	if db == nil {
		return 0
	}
	EnsureFactoryCoreColumns(db)
	var id int64
	_ = db.QueryRow(`SELECT id FROM sys_plant WHERE COALESCE(is_deleted,0)=0 ORDER BY is_default DESC, id ASC LIMIT 1`).Scan(&id)
	if id > 0 {
		return id
	}
	orgName := "加工厂"
	_ = db.QueryRow(`SELECT COALESCE(name,'加工厂') FROM sys_organization ORDER BY id LIMIT 1`).Scan(&orgName)
	res, err := db.Exec(`INSERT INTO sys_plant(code, name, status, is_default, remark) VALUES(?,?,?,?,?)`,
		"PLANT-01", orgName+"默认厂区", "active", 1, "系统默认厂区")
	if err != nil {
		_ = db.QueryRow(`SELECT id FROM sys_plant WHERE code='PLANT-01' AND COALESCE(is_deleted,0)=0`).Scan(&id)
		return id
	}
	id, _ = res.LastInsertId()
	return id
}

// EnsureFactoryCoreColumns applies lightweight DDL for factory-core columns/tables (idempotent).
func EnsureFactoryCoreColumns(db *sql.DB) {
	if db == nil {
		return
	}
	stmts := []string{
		`ALTER TABLE inv_warehouse ADD COLUMN IF NOT EXISTS warehouse_role TEXT`,
		`ALTER TABLE inv_warehouse ADD COLUMN IF NOT EXISTS plant_id INTEGER`,
		`ALTER TABLE pur_weigh_variety ADD COLUMN IF NOT EXISTS warehouse_id INTEGER`,
		`ALTER TABLE pur_weigh_variety ADD COLUMN IF NOT EXISTS warehouse_role TEXT`,
		`ALTER TABLE pur_weigh_variety ADD COLUMN IF NOT EXISTS cold_store_type TEXT`,
		`CREATE TABLE IF NOT EXISTS sys_plant (
		  id BIGSERIAL PRIMARY KEY, org_id INTEGER NOT NULL DEFAULT 1, code TEXT NOT NULL UNIQUE,
		  name TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active', is_default INTEGER NOT NULL DEFAULT 0,
		  remark TEXT, created_at TEXT NOT NULL DEFAULT NOW(), updated_at TEXT NOT NULL DEFAULT NOW(), is_deleted INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS iam_role_plant_scope (
		  id BIGSERIAL PRIMARY KEY, role_id INTEGER NOT NULL, plant_id INTEGER NOT NULL, UNIQUE(role_id, plant_id))`,
		`CREATE TABLE IF NOT EXISTS iam_user_plant_scope (
		  id BIGSERIAL PRIMARY KEY, user_id INTEGER NOT NULL, plant_id INTEGER NOT NULL, UNIQUE(user_id, plant_id))`,
		`UPDATE inv_warehouse SET warehouse_role = warehouse_type WHERE COALESCE(warehouse_role,'') = '' AND COALESCE(warehouse_type,'') <> ''`,
	}
	for _, q := range stmts {
		_, _ = db.Exec(q)
	}
}
