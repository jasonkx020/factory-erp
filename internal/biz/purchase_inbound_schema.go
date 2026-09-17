package biz

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// purchaseFlowMeta is stored in pd_flow_graph.graph_json.meta for purchase kinds.
type purchaseFlowMeta struct {
	Capabilities []string `json:"capabilities,omitempty"`
	FieldPack    string   `json:"field_pack,omitempty"`
	ProductID    int64    `json:"product_id,omitempty"`
}

type purchaseFlowConfig struct {
	Kind         string   `json:"kind"`
	GraphCode    string   `json:"graph_code"`
	Capabilities []string `json:"capabilities"`
	FieldPack    string   `json:"field_pack"`
	RequireWeigh bool     `json:"require_weigh"`
}

func purchaseInboundReceiveKind(receiveKind string) string {
	if strings.EqualFold(strings.TrimSpace(receiveKind), "stockin") {
		return "purchase_stockin"
	}
	return "purchase_gate"
}

func parsePurchaseFlowMeta(gjson string) purchaseFlowMeta {
	var wrap struct {
		Meta purchaseFlowMeta `json:"meta"`
	}
	_ = json.Unmarshal([]byte(gjson), &wrap)
	return wrap.Meta
}

func metaHasCapability(m purchaseFlowMeta, cap string) bool {
	cap = strings.ToLower(strings.TrimSpace(cap))
	for _, c := range m.Capabilities {
		if strings.EqualFold(strings.TrimSpace(c), cap) {
			return true
		}
	}
	return false
}

func (s *Services) loadPurchaseFlowConfig(receiveKind string) purchaseFlowConfig {
	kind := purchaseInboundReceiveKind(receiveKind)
	var gjson, code string
	_ = s.DB.QueryRow(`SELECT COALESCE(graph_json,'{}'), COALESCE(code,'') FROM pd_flow_graph
		WHERE kind=? AND status='active' AND COALESCE(is_deleted,0)=0 ORDER BY id DESC LIMIT 1`, kind).Scan(&gjson, &code)
	meta := parsePurchaseFlowMeta(gjson)
	// Legacy graphs without meta: cassava pack defaults to weigh+cassava_gate; others qty-only.
	if gjson != "" && !strings.Contains(gjson, `"capabilities"`) && len(meta.Capabilities) == 0 && meta.FieldPack == "" {
		if s.IsCassavaIndustryPack() {
			meta.Capabilities = []string{"weigh"}
			meta.FieldPack = "cassava_gate"
		}
	}
	cfg := purchaseFlowConfig{
		Kind:         kind,
		GraphCode:    code,
		Capabilities: meta.Capabilities,
		FieldPack:    strings.TrimSpace(meta.FieldPack),
		RequireWeigh: metaHasCapability(meta, "weigh"),
	}
	if cfg.Capabilities == nil {
		cfg.Capabilities = []string{}
	}
	return cfg
}

func (s *Services) purchaseFlowRequireWeigh(receiveKind string) bool {
	return s.loadPurchaseFlowConfig(receiveKind).RequireWeigh
}

func (s *Services) purchaseFlowFieldPack(receiveKind string) string {
	return s.loadPurchaseFlowConfig(receiveKind).FieldPack
}

// EnsurePurchaseFlowGraphMeta writes default meta for purchase graphs missing capabilities
// (does not overwrite graphs that already declare capabilities).
func EnsurePurchaseFlowGraphMeta(db *sql.DB, industryPack string) {
	if db == nil {
		return
	}
	cassava := strings.EqualFold(strings.TrimSpace(industryPack), "cassava")
	for _, kind := range []string{"purchase_gate", "purchase_stockin"} {
		rows, err := db.Query(`SELECT id, COALESCE(graph_json,'{}') FROM pd_flow_graph
			WHERE kind=? AND COALESCE(is_deleted,0)=0`, kind)
		if err != nil {
			continue
		}
		type row struct {
			id    int64
			gjson string
		}
		var list []row
		for rows.Next() {
			var r row
			if rows.Scan(&r.id, &r.gjson) == nil {
				list = append(list, r)
			}
		}
		_ = rows.Close()
		for _, r := range list {
			if strings.Contains(r.gjson, `"capabilities"`) {
				continue
			}
			var doc map[string]interface{}
			if err := json.Unmarshal([]byte(r.gjson), &doc); err != nil || doc == nil {
				doc = map[string]interface{}{"nodes": []interface{}{}, "edges": []interface{}{}}
			}
			meta, _ := doc["meta"].(map[string]interface{})
			if meta == nil {
				meta = map[string]interface{}{}
			}
			if _, ok := meta["capabilities"]; ok {
				continue
			}
			if cassava {
				meta["capabilities"] = []string{"weigh"}
				meta["field_pack"] = "cassava_gate"
			} else {
				meta["capabilities"] = []interface{}{}
				meta["field_pack"] = ""
			}
			doc["meta"] = meta
			b, err := json.Marshal(doc)
			if err != nil {
				continue
			}
			_, _ = db.Exec(`UPDATE pd_flow_graph SET graph_json=?, updated_at=NOW() WHERE id=?`, string(b), r.id)
			if kind == "purchase_gate" {
				_, _ = db.Exec(`UPDATE pd_flow_graph SET name=CASE WHEN name LIKE '%过磅%' THEN '采购入厂流程' ELSE name END WHERE id=?`, r.id)
			}
		}
	}
}

func baseInboundFormFields(requireWeigh bool) []FormFieldDef {
	fields := []FormFieldDef{
		fieldText("batch_no", "溯源批号", true),
		fieldText("party_name", "供应商", true),
		fieldText("party_mobile", "电话", false),
		fieldText("origin", "产地", false),
		fieldText("variety", "品种/产品", true),
	}
	if requireWeigh {
		fields = append(fields,
			fieldSelect("channel", "称重渠道", true, "厂内秤", "外磅单"),
			fieldNum("gross_weight", "毛重", true, "kg"),
		)
	} else {
		fields = append(fields, fieldNum("net_weight", "数量/净重", true, "kg"))
	}
	fields = append(fields,
		fieldNum("unit_price", "单价", false, "元/kg"),
		fieldText("plate_no", "车牌号", false),
		fieldText("receive_address", "收货地址", false),
		fieldText("remark", "备注", false),
	)
	return fields
}

func cassavaGateFormFields(requireWeigh bool) []FormFieldDef {
	fields := baseInboundFormFields(requireWeigh)
	extra := []FormFieldDef{
		fieldNum("deduct_rate", "扣损率", false, "%"),
		fieldNum("reject_weight", "不合格重量", false, "kg"),
		fieldNum("freight_fee", "运费", false, "元"),
		fieldNum("loading_fee", "装卸费", false, "元"),
		fieldNum("weigh_fee", "计量费", false, "元"),
		fieldSelect("grade", "等级", false, "A", "B", "C"),
		fieldSelect("cold_store_type", "冷库类型", false, "保鲜库", "半成品库", "成品库"),
	}
	if requireWeigh {
		extra = append(extra,
			fieldText("photo_material", "材料称重照", true),
			fieldText("photo_scale_display", "磅显特写", true),
			fieldText("photo_closeup", "近景照", true),
		)
	} else {
		extra = append(extra, fieldText("photo_closeup", "现场照片", false))
	}
	return append(fields, extra...)
}

func (s *Services) inboundFormFields(receiveKind string) []FormFieldDef {
	cfg := s.loadPurchaseFlowConfig(receiveKind)
	pack := cfg.FieldPack
	if pack == "" && s.IsCassavaIndustryPack() && cfg.RequireWeigh {
		pack = "cassava_gate"
	}
	switch pack {
	case "cassava_gate":
		return cassavaGateFormFields(cfg.RequireWeigh)
	default:
		return baseInboundFormFields(cfg.RequireWeigh)
	}
}

func (s *Services) handlePurchaseInboundFormSchema(c *gin.Context) bool {
	kind := strings.ToLower(strings.TrimSpace(c.Query("receive_kind")))
	if kind == "" {
		kind = "gate"
	}
	cfg := s.loadPurchaseFlowConfig(kind)
	api.OK(c, gin.H{
		"receive_kind":  kind,
		"capabilities":  cfg.Capabilities,
		"field_pack":    cfg.FieldPack,
		"require_weigh": cfg.RequireWeigh,
		"industry_pack": s.IndustryPack,
		"fields":        s.inboundFormFields(kind),
	})
	return true
}

func (s *Services) handlePurchaseWeighFlow(c *gin.Context) bool {
	path := c.Request.URL.Path
	if strings.Contains(path, "/inbound-form-schema") {
		return s.handlePurchaseInboundFormSchema(c)
	}
	if strings.Contains(path, "/config") {
		kind := strings.ToLower(strings.TrimSpace(c.Query("receive_kind")))
		if kind == "" {
			kind = "gate"
		}
		cfg := s.loadPurchaseFlowConfig(kind)
		api.OK(c, gin.H{
			"receive_kind":  kind,
			"kind":          cfg.Kind,
			"graph_code":    cfg.GraphCode,
			"capabilities":  cfg.Capabilities,
			"field_pack":    cfg.FieldPack,
			"require_weigh": cfg.RequireWeigh,
			"industry_pack": s.IndustryPack,
			"fields":        s.inboundFormFields(kind),
		})
		return true
	}
	return s.handleWeighFlowNextOptions(c)
}
