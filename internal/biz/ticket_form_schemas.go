package biz

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormFieldDef describes one dynamic form field on a ticket category.
type FormFieldDef struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // text|number|date|select|textarea
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Unit     string   `json:"unit,omitempty"`
}

func fieldText(key, label string, req bool) FormFieldDef {
	return FormFieldDef{Key: key, Label: label, Type: "text", Required: req}
}
func fieldNum(key, label string, req bool, unit string) FormFieldDef {
	return FormFieldDef{Key: key, Label: label, Type: "number", Required: req, Unit: unit}
}
func fieldDate(key, label string, req bool) FormFieldDef {
	return FormFieldDef{Key: key, Label: label, Type: "date", Required: req}
}
func fieldSelect(key, label string, req bool, opts ...string) FormFieldDef {
	return FormFieldDef{Key: key, Label: label, Type: "select", Required: req, Options: opts}
}

type ticketCategorySeed struct {
	Code    string
	Name    string
	Remark  string
	BizHint string
	Fields  []FormFieldDef
}

func defaultTicketCategorySeeds() []ticketCategorySeed {
	return []ticketCategorySeed{
		{
			Code: "farm_inbound", Name: "原料半成品收购入厂", Remark: "农户/供应商收购入厂申报",
			BizHint: "/purchase/hub/weigh",
			Fields: []FormFieldDef{
				fieldDate("biz_date", "日期", true),
				fieldText("product_name", "产品名称", true),
				fieldText("plate_no", "车牌号", true),
				fieldText("origin_addr", "产地地址", false),
				fieldText("trace_lot", "溯源批号", true),
				fieldText("supplier_name", "供应商/加工厂姓名", true),
				fieldText("phone", "电话", false),
				fieldText("recv_addr", "收货地址", false),
				fieldNum("in_weight_kg", "入场重量", true, "kg"),
				fieldNum("unit_price", "单价", false, "元/kg"),
				fieldNum("sample_pass_pct", "规格抽检合格率", false, "%"),
				fieldNum("reject_weight_kg", "不合格重量", false, "kg"),
				fieldNum("loss_rate_pct", "扣损率", false, "%"),
				fieldNum("net_weight_kg", "实际入厂净重", true, "kg"),
				fieldNum("freight", "运费", false, "元"),
				fieldNum("loading_fee", "装卸费", false, "元"),
				fieldNum("weigh_fee", "过磅费", false, "元"),
				fieldNum("settle_amount", "结算金额", false, "元"),
			},
		},
		{
			Code: "stock_inbound", Name: "原料半成品成品入库", Remark: "入库申报（协作工单）",
			BizHint: "/inventory/hub/inbound",
			Fields: []FormFieldDef{
				fieldText("product_name", "产品名称", true),
				fieldText("product_addr", "产品地址", false),
				fieldText("trace_lot", "溯源批号", true),
				fieldText("supplier_name", "供应商/姓名", true),
				fieldNum("qty", "重量/数量", true, ""),
				fieldSelect("qty_unit", "单位", true, "kg", "袋"),
				fieldSelect("warehouse", "冷库", true, "保鲜库", "半成品库", "成品库"),
			},
		},
		{
			Code: "prod_process", Name: "产品加工", Remark: "加工申报（协作工单）",
			BizHint: "/production/hub/process-reports",
			Fields: []FormFieldDef{
				fieldDate("process_date", "加工日期", true),
				fieldSelect("material_name", "产品名称", true, "原料", "白条", "断块", "切块"),
				fieldNum("inout_qty", "出/入库量", true, ""),
				fieldSelect("inout_unit", "出/入库单位", true, "kg", "袋"),
				fieldText("lot_no", "产品批号", true),
				fieldSelect("process_type", "加工类型", true, "去皮", "切断", "去芯", "过筛"),
				fieldNum("process_qty_kg", "加工数量", true, "kg"),
				fieldNum("scrap_unusable_kg", "不可用损耗", false, "kg"),
				fieldSelect("scrap_type", "次品类型", false, "切断次品", "去芯次品", "切块次品", "筛选装袋次品"),
				fieldSelect("finished_name", "成品名称", true, "白条", "断块", "切块", "成品"),
				fieldNum("finished_qty", "成品重量/数量", true, ""),
				fieldSelect("finished_unit", "成品单位", true, "kg", "袋"),
			},
		},
		{
			Code: "sales_outbound", Name: "销售出厂", Remark: "销售出厂结算协作单",
			BizHint: "/sales/outbound-settle",
			Fields: []FormFieldDef{
				fieldDate("biz_date", "日期", true),
				fieldSelect("product_name", "产品名称", true, "原料", "去头尾白条", "去芯切断", "去芯切块"),
				fieldText("plate_no", "车牌号", true),
				fieldText("driver_name", "司机名字", false),
				fieldText("trace_lot", "溯源批号", true),
				fieldDate("produce_date", "生产日期", false),
				fieldNum("qty", "重量/数量", true, ""),
				fieldSelect("qty_unit", "单位", true, "kg", "袋"),
				fieldNum("freight", "运费", false, "元"),
				fieldNum("weigh_fee", "过磅费", false, "元"),
				fieldNum("loading_fee", "装卸费", false, "元"),
				fieldNum("settle_amount", "结算金额", true, "元"),
			},
		},
		{
			Code: "tool_issue", Name: "物料工具领用", Remark: "员工申请领取物料/劳保工具",
			BizHint: "/hr/tool-issues",
			Fields: []FormFieldDef{
				fieldDate("biz_date", "日期", true),
				fieldNum("seq_no", "序号", false, ""),
				fieldText("employee_name", "员工姓名", true),
				fieldText("items_summary", "工具明细", true),
				fieldNum("issue_qty", "领取合计", true, ""),
			},
		},
		{
			Code: "tool_return", Name: "物料工具归还", Remark: "员工申请归还已领工具",
			BizHint: "/hr/tool-issues",
			Fields: []FormFieldDef{
				fieldDate("biz_date", "日期", true),
				fieldNum("seq_no", "序号", false, ""),
				fieldText("employee_name", "员工姓名", true),
				fieldText("items_summary", "工具明细", true),
				fieldNum("return_qty", "交还合计", true, ""),
			},
		},
		{
			Code: "piece_issue", Name: "计件工领料", Remark: "计件工领料协作单",
			BizHint: "/production/hub/piece-issue",
			Fields: []FormFieldDef{
				fieldDate("biz_date", "日期", true),
				fieldNum("seq_no", "序号", false, ""),
				fieldText("employee_name", "员工姓名", true),
				fieldSelect("process", "工序", true, "去皮", "去芯", "切块", "计时"),
				fieldNum("unit_price", "工序单价", true, "元"),
				fieldNum("qty", "重量/数量", true, ""),
				fieldSelect("qty_unit", "单位", true, "斤", "时"),
				fieldNum("deduct_qty", "扣减不合格数量", false, "斤"),
				fieldNum("qty_total", "重量/数量合计", true, ""),
				fieldNum("amount", "合计金额", true, "元"),
			},
		},
	}
}

func marshalFormSchema(fields []FormFieldDef) string {
	b, _ := json.Marshal(fields)
	return string(b)
}

func parseFormSchema(raw string) []FormFieldDef {
	out := []FormFieldDef{}
	if raw == "" || raw == "null" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func validatePayloadAgainstSchema(schema []FormFieldDef, payload map[string]interface{}) string {
	for _, f := range schema {
		if !f.Required {
			continue
		}
		v, ok := payload[f.Key]
		if !ok || v == nil {
			return "FIELD_REQUIRED:" + f.Key
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			return "FIELD_REQUIRED:" + f.Key
		}
	}
	return ""
}

func normalizeFormSchemaJSON(v interface{}) string {
	switch x := v.(type) {
	case string:
		if x == "" {
			return "[]"
		}
		var tmp []FormFieldDef
		if err := json.Unmarshal([]byte(x), &tmp); err != nil {
			return "[]"
		}
		return marshalFormSchema(tmp)
	case []interface{}:
		b, _ := json.Marshal(x)
		var tmp []FormFieldDef
		_ = json.Unmarshal(b, &tmp)
		return marshalFormSchema(tmp)
	case []FormFieldDef:
		return marshalFormSchema(x)
	default:
		if v == nil {
			return "[]"
		}
		b, err := json.Marshal(v)
		if err != nil {
			return "[]"
		}
		var tmp []FormFieldDef
		if err := json.Unmarshal(b, &tmp); err != nil {
			return "[]"
		}
		return marshalFormSchema(tmp)
	}
}

func autoTitleFromPayload(catName string, payload map[string]interface{}) string {
	for _, k := range []string{"product_name", "material_name", "tool_name", "employee_name", "title"} {
		if v, ok := payload[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return catName + " · " + s
			}
		}
	}
	return catName
}
