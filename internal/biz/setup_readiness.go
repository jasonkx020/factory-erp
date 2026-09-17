package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

type setupCheckItem struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Required bool  `json:"required"`
}

// GetSetupReadiness reports whether factory master data is ready for field transactions.
func (s *Services) GetSetupReadiness(c *gin.Context) {
	plantID := asInt64Or0(c.Query("plant_id"))
	if plantID <= 0 {
		plantID = EnsureDefaultPlant(s.DB)
	}
	items := s.collectSetupChecks(plantID)
	ready := true
	missing := []string{}
	for _, it := range items {
		if it.Required && !it.OK {
			ready = false
			missing = append(missing, it.Code)
		}
	}
	api.OK(c, gin.H{
		"ready":            ready,
		"plant_id":         plantID,
		"product_profile":  s.ProductProfile,
		"industry_pack":    s.IndustryPack,
		"missing":          missing,
		"items":            items,
	})
}

func (s *Services) collectSetupChecks(plantID int64) []setupCheckItem {
	items := []setupCheckItem{}
	add := func(code, label string, ok bool, detail string, required bool) {
		items = append(items, setupCheckItem{Code: code, Label: label, OK: ok, Detail: detail, Required: required})
	}

	var orgN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sys_organization WHERE COALESCE(is_deleted,0)=0`).Scan(&orgN)
	add("organization", "组织", orgN > 0, "", true)

	var plantN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM sys_plant WHERE COALESCE(is_deleted,0)=0 AND status='active'`).Scan(&plantN)
	add("plant", "厂区", plantN > 0, "", true)

	roleOK := func(role string) bool {
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM inv_warehouse
			WHERE COALESCE(is_deleted,0)=0 AND status='active'
			  AND (LOWER(COALESCE(warehouse_role,''))=? OR LOWER(COALESCE(warehouse_type,''))=?)`,
			role, role).Scan(&n)
		return n > 0
	}
	add("warehouse_raw", "原料仓(role=raw)", roleOK("raw"), "", true)
	add("warehouse_semi", "半成品仓(role=semi)", roleOK("semi"), "", true)
	add("warehouse_fg", "成品仓(role=fg)", roleOK("fg"), "", true)

	var prodN, unitN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM prd_product WHERE COALESCE(is_deleted,0)=0 AND status='active'`).Scan(&prodN)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM prd_product_unit WHERE COALESCE(is_deleted,0)=0`).Scan(&unitN)
	add("product", "产品主数据", prodN > 0, "", true)
	add("unit", "计量单位", unitN > 0, "", false)

	var flowN, stepBad int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_flow_graph
		WHERE kind='production' AND status='active' AND COALESCE(is_deleted,0)=0`).Scan(&flowN)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing_step rs
		JOIN pd_flow_graph g ON g.routing_id=rs.routing_id AND g.kind='production' AND g.status='active' AND COALESCE(g.is_deleted,0)=0
		WHERE COALESCE(rs.output_product_id,0)<=0`).Scan(&stepBad)
	prodFlowOK := flowN > 0
	if flowN > 0 && stepBad > 0 {
		prodFlowOK = false
	}
	// also accept active routing without flow_graph for legacy
	if !prodFlowOK {
		var rn int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_routing WHERE status='active' AND COALESCE(is_deleted,0)=0`).Scan(&rn)
		prodFlowOK = rn > 0
	}
	add("production_flow", "已发布生产工艺", prodFlowOK, "", true)

	var varietyN, gateN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pur_weigh_variety WHERE status='active' AND COALESCE(is_deleted,0)=0`).Scan(&varietyN)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_flow_graph
		WHERE kind='purchase_gate' AND status='active' AND COALESCE(is_deleted,0)=0`).Scan(&gateN)
	weighRequired := s.purchaseFlowRequireWeigh("gate") || s.IsCassavaIndustryPack()
	add("weigh_variety", "采购品种", varietyN > 0, "", weighRequired)
	add("purchase_gate_flow", "采购入厂流程", gateN > 0, "", true)

	var shiftN, userN, rateN int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pd_shift WHERE COALESCE(is_deleted,0)=0`).Scan(&shiftN)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM iam_user WHERE status='active' AND COALESCE(is_deleted,0)=0`).Scan(&userN)
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM pay_process_wage_rate WHERE COALESCE(is_deleted,0)=0`).Scan(&rateN)
	add("shift", "产线班次", shiftN > 0, "", true)
	add("field_user", "现场用户", userN > 0, "", true)
	add("wage_rate", "工序工价", rateN > 0, "计件工序建议配置工价", false)

	if s.IsExtendedProfile() {
		var custN, subjN int
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM crm_customer WHERE COALESCE(is_deleted,0)=0`).Scan(&custN)
		_ = s.DB.QueryRow(`SELECT COUNT(1) FROM fin_account_subject WHERE COALESCE(is_deleted,0)=0`).Scan(&subjN)
		add("customer", "客户档案", custN > 0, "", false)
		add("finance_subject", "会计科目", subjN > 0, "", false)
	}
	_ = plantID
	return items
}

// RequireSetupReady blocks field transactions when setup incomplete.
// Returns true if the request was rejected (already written response).
func (s *Services) RequireSetupReady(c *gin.Context, codes ...string) bool {
	plantID := EnsureDefaultPlant(s.DB)
	items := s.collectSetupChecks(plantID)
	need := map[string]bool{}
	for _, c0 := range codes {
		need[c0] = true
	}
	if len(need) == 0 {
		for _, it := range items {
			if it.Required {
				need[it.Code] = true
			}
		}
	}
	for _, it := range items {
		if !need[it.Code] {
			continue
		}
		if it.Required && !it.OK {
			api.FailJSON(c, "SETUP_INCOMPLETE:"+it.Code)
			return true
		}
	}
	return false
}

// GetProductProfile returns deployment product profile for Admin UI.
func (s *Services) GetProductProfile(c *gin.Context) {
	brand, plantName := s.loadBrandSettings()
	api.OK(c, gin.H{
		"profile":            s.ProductProfile,
		"industry_pack":      s.IndustryPack,
		"brand_name":         brand,
		"plant_display_name": plantName,
		"extended":           s.IsExtendedProfile(),
		"cassava_pack":       s.IsCassavaIndustryPack(),
	})
}

func (s *Services) loadBrandSettings() (brand, plant string) {
	brand = "加工厂 ERP"
	plant = "加工厂"
	var bRaw, pRaw string
	_ = s.DB.QueryRow(`SELECT COALESCE(value_json,'') FROM sys_org_setting WHERE org_id=1 AND setting_key='brand_name'`).Scan(&bRaw)
	_ = s.DB.QueryRow(`SELECT COALESCE(value_json,'') FROM sys_org_setting WHERE org_id=1 AND setting_key='plant_display_name'`).Scan(&pRaw)
	brand = strings.Trim(strings.TrimSpace(bRaw), `"`)
	plant = strings.Trim(strings.TrimSpace(pRaw), `"`)
	if brand == "" {
		brand = "加工厂 ERP"
	}
	if plant == "" {
		plant = "加工厂"
	}
	return
}
