package biz

import (
	"database/sql"
	"log"
)

// EnsureCassavaIndustryPack seeds cassava weigh varieties and binds warehouse roles.
// Called only when product.industry_pack=cassava.
func EnsureCassavaIndustryPack(db *sql.DB) {
	if db == nil {
		return
	}
	_, _ = db.Exec(`INSERT INTO pur_weigh_variety(code, name, sort_no, status, warehouse_role, cold_store_type, remark)
		VALUES
		 ('WV-FRESH', '鲜木薯', 10, 'active', 'raw', 'fresh', '农户鲜薯过磅入厂，入原料/保鲜库'),
		 ('WV-SEMI', '半成品（去芯薯肉）', 20, 'active', 'semi', 'semi', '外购或厂内半成品过磅入厂'),
		 ('WV-FG', '成品入库（袋装木薯丁）', 30, 'active', 'fg', 'fg', '成品过磅入库')
		ON CONFLICT (code) DO NOTHING`)
	_, _ = db.Exec(`UPDATE pur_weigh_variety v
		SET default_product_id = p.id, updated_at = NOW()
		FROM prd_product p
		WHERE v.default_product_id IS NULL AND COALESCE(v.is_deleted,0)=0
		  AND (
		    (v.code = 'WV-FRESH' AND p.code = 'RM-CASSAVA')
		    OR (v.code = 'WV-SEMI' AND p.code = 'SF-COREOUT')
		    OR (v.code = 'WV-FG' AND p.code = 'FG-DICED')
		  )`)
	_, _ = db.Exec(`UPDATE pur_weigh_variety v
		SET warehouse_id = w.id, warehouse_role = COALESCE(NULLIF(v.warehouse_role,''), w.warehouse_role, w.warehouse_type)
		FROM inv_warehouse w
		WHERE COALESCE(v.is_deleted,0)=0 AND COALESCE(v.warehouse_id,0)=0
		  AND COALESCE(w.is_deleted,0)=0 AND w.status='active'
		  AND LOWER(COALESCE(w.warehouse_role, w.warehouse_type, '')) = LOWER(COALESCE(v.warehouse_role, ''))`)
	EnsureFreshCassavaRouting(db)
	log.Printf("[seed] cassava industry pack ensured")
}

// EnsureOrgBrandDefaults writes factory display name into sys_org_setting when missing.
func EnsureOrgBrandDefaults(db *sql.DB, cassavaPack bool) {
	if db == nil {
		return
	}
	brand := "加工厂 ERP"
	orgName := "加工厂"
	if cassavaPack {
		brand = "木薯加工厂 ERP"
		_ = db.QueryRow(`SELECT COALESCE(name,'桂南木薯加工厂') FROM sys_organization ORDER BY id LIMIT 1`).Scan(&orgName)
	} else {
		_ = db.QueryRow(`SELECT COALESCE(name,'加工厂') FROM sys_organization ORDER BY id LIMIT 1`).Scan(&orgName)
	}
	_, _ = db.Exec(`INSERT INTO sys_org_setting(org_id, setting_key, value_json)
		SELECT 1, 'brand_name', ? WHERE NOT EXISTS (
			SELECT 1 FROM sys_org_setting WHERE org_id=1 AND setting_key='brand_name')`, `"`+brand+`"`)
	_, _ = db.Exec(`INSERT INTO sys_org_setting(org_id, setting_key, value_json)
		SELECT 1, 'plant_display_name', ? WHERE NOT EXISTS (
			SELECT 1 FROM sys_org_setting WHERE org_id=1 AND setting_key='plant_display_name')`, `"`+orgName+`"`)
}
