package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type indexRow struct {
	phase                  int
	domain, module, method string
	path                   string
}

type moduleMeta struct {
	domain, module                       string
	phase                                int
	list, create, detail, update, remove string
	actions                              []string
	readOnly, actionOnly                 bool
}

var preferredList = map[[2]string]string{
	{"产品管理", "产品档案"}:       "/product/products",
	{"产品管理", "产品单位管理"}:     "/product/products",
	{"产品管理", "APP产品排序"}:    "/product/app-sorts",
	{"产品管理", "生产规格绑定"}:     "/product/specs",
	{"库存管理", "库存查询"}:       "/inventory/balances",
	{"库存管理", "可用量分析"}:      "/inventory/availability",
	{"库存管理", "出入库记录汇总"}:    "/inventory/stock-txns",
	{"库存管理", "待用量统计"}:      "/inventory/reservations",
	{"库存管理", "在途量统计"}:      "/inventory/in-transits",
	{"生产管理", "生产任务单"}:      "/production/tasks",
	{"生产管理", "工序定义"}:       "/production/processes",
	{"生产管理", "工艺流程"}:       "/production/routings",
	{"生产管理", "产线班次"}:       "/production/shifts",
	{"生产管理", "例外派岗"}:       "/production/dispatches",
	{"生产管理", "过站记录"}:       "/production/report-works",
	{"生产管理", "生产派工"}:       "/production/dispatches",
	{"生产管理", "扫码报工"}:       "/production/report-works",
	{"生产管理", "工序设置"}:       "/production/processes",
	{"生产管理", "工序管理"}:       "/production/processes",
	{"生产管理", "联动式领料"}:      "/production/requisitions",
	{"生产管理", "质检管理"}:       "/production/qc-orders",
	{"生产管理", "废料管理"}:       "/production/scraps",
	{"生产管理", "车间管理"}:       "/production/workshops",
	{"生产管理", "车间工作台"}:      "/production/workshop-workbench/overview",
	{"生产管理", "进度跟踪"}:       "/production/progress",
	{"生产管理", "计件工资"}:       "/production/piecework-summaries",
	{"生产管理", "灵活派发工单"}:     "/production/flex-dispatches",
	{"生产管理", "多单整合管理"}:     "/production/task-merges",
	{"生产管理", "图纸分发"}:       "/production/drawing-links",
	{"生产管理", "一单多商品"}:      "/production/tasks",
	{"生产管理", "自动BOM"}:      "/production/boms",
	{"生产管理", "MRP物料分析"}:    "/production/mrp-runs",
	{"生产管理", "委外加工"}:       "/production/outsources",
	{"生产管理", "受托加工生产流程管控"}: "/production/consignments",
	{"生产管理", "成本隐藏"}:       "/production/cost-hide-policies",
	{"工资管理", "工序工资"}:       "/payroll/wage-rates",
	{"工资管理", "工资批量管理"}:     "/payroll/sheets",
	{"工资管理", "薪酬核算"}:       "/payroll/calculations",
	{"工资管理", "工人信息管理"}:     "/payroll/worker-profiles",
	{"工资管理", "销售提成"}:       "/payroll/commission-rules",
	{"人事管理", "权限分配"}:       "/iam/users",
	{"人事管理", "员工"}:         "/hr/employees",
	{"人事管理", "入职登记"}:       "/hr/onboards",
	{"人事管理", "离职登记"}:       "/hr/offboards",
	{"人事管理", "班次管理"}:       "/hr/shifts",
	{"人事管理", "考勤管理"}:       "/hr/attendance/rules",
	{"人事管理", "考勤明细"}:       "/hr/attendance/records",
	{"人事管理", "请假管理"}:       "/hr/leave-requests",
	{"审批管理", "任务管理"}:       "/approval/tasks",
	{"客户管理", "CRM客户管理"}:   "/crm/customers",
	{"客户管理", "商机管理"}:       "/crm/opportunities",
	{"客户管理", "客户档案"}:       "/crm/customers",
	{"客户管理", "客户跟进"}:       "/crm/follow-ups",
	{"客户管理", "资源分配"}:       "/crm/lead-assigns",
	{"客户管理", "保护机制"}:       "/crm/protect-rules",
	{"客户管理", "释放机制"}:       "/crm/releases",
	{"客户管理", "询价管理"}:       "/crm/inquiries",
	{"客户管理", "导入客户"}:       "/crm/imports",
	{"客户管理", "线索锁定"}:       "/crm/customers",
	{"客户管理", "线索隐藏"}:       "/crm/customers",
	{"客户管理", "任务提醒"}:       "/crm/task-reminders",
	{"销售管理", "销售订单"}:       "/sales/orders",
	{"销售管理", "询价管理"}:       "/sales/inquiries",
	{"销售管理", "预发货管理"}:      "/sales/pre-shipments",
	{"销售管理", "自助下单"}:       "/sales/self-orders",
	{"销售管理", "我的订单"}:       "/sales/my-orders",
	{"销售管理", "单据打印"}:       "/sales/prints",
	{"销售管理", "订单复购"}:       "/sales/orders",
	{"销售管理", "修改订单"}:       "/sales/orders",
	{"销售管理", "询价审批"}:       "/sales/inquiries",
	{"销售管理", "报价计算器"}:      "/sales/quote-calculator",
	{"采购管理", "供应商管理"}:      "/purchase/suppliers",
	{"采购管理", "采购申请"}:       "/purchase/requests",
	{"采购管理", "采购入库"}:       "/purchase/inbounds",
	{"采购管理", "来料质检"}:       "/purchase/incoming-qcs",
	{"统计报表", "老板驾驶舱"}:      "/report/dashboards/boss",
	{"财务管理", "凭证管理"}:       "/finance/vouchers",
	{"财务管理", "收款核单"}:       "/finance/receipt-writeoffs",
	{"财务管理", "账目管理"}:       "/finance/account-subjects",
	{"固定资产管理", "固定资产项目"}:   "/asset/fixed-assets",
	{"系统管理", "基础设置"}:       "/system/settings",
	{"系统管理", "操作日志"}:       "/system/operation-logs",
	{"系统管理", "自定义权限"}:      "/iam/permissions",
	{"系统管理", "自定义菜单"}:      "/iam/menus",
	{"系统管理", "登录控制"}:       "/iam/login-policy",
	{"系统管理", "账户冻结"}:       "/iam/users",
}

var readOnlyModules = map[string]struct{}{
	"库存查询": {}, "可用量分析": {}, "老板驾驶舱": {}, "生产看板": {}, "生产实况": {},
	"操作日志": {}, "在途量统计": {}, "待用量统计": {}, "车间工作台": {}, "进度跟踪": {},
	"历史报价查询": {}, "数据排行榜": {},
}

var menuBlockRe = regexp.MustCompile(`(?s)domain:\s*'([^']+)',\s*modules:\s*\[(.*?)\]`)
var menuModRe = regexp.MustCompile(`'([^']+)'`)

// GenWebMeta generates web/packages/shared/src/generated/modules.ts.
func GenWebMeta(root string) (int, error) {
	indexPath := filepath.Join(root, "docs", "openapi-路径全表.md")
	menusPath := filepath.Join(root, "web", "packages", "shared", "src", "constants", "menus.ts")
	outPath := filepath.Join(root, "web", "packages", "shared", "src", "generated", "modules.ts")

	indexText, err := os.ReadFile(indexPath)
	if err != nil {
		return 0, err
	}
	menusText, err := os.ReadFile(menusPath)
	if err != nil {
		return 0, err
	}

	byMod := map[[2]string][]indexRow{}
	for _, line := range strings.Split(string(indexText), "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		parts := splitTableRow(line)
		if len(parts) < 6 || parts[0] == "分期" || strings.HasPrefix(parts[0], "-") {
			continue
		}
		phase, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		path := strings.Trim(parts[4], "`")
		row := indexRow{
			phase:  phase,
			domain: parts[1],
			module: parts[2],
			method: strings.ToUpper(parts[3]),
			path:   path,
		}
		key := [2]string{row.domain, row.module}
		byMod[key] = append(byMod[key], row)
	}

	var menuPairs [][2]string
	for _, block := range menuBlockRe.FindAllStringSubmatch(string(menusText), -1) {
		domain := block[1]
		for _, m := range menuModRe.FindAllStringSubmatch(block[2], -1) {
			menuPairs = append(menuPairs, [2]string{domain, m[1]})
		}
	}

	seen := map[[2]string]struct{}{}
	var items []moduleMeta
	for _, key := range menuPairs {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ops := byMod[key]
		phase := 1
		if len(ops) > 0 {
			phase = ops[0].phase
		}
		pref := preferredList[key]

		var getCollections, postCollections, getDetails, putDetails, deleteDetails, actions []string
		for _, r := range ops {
			sp := StripAPI(r.path)
			switch {
			case r.method == "GET" && !strings.Contains(sp, "{"):
				getCollections = append(getCollections, sp)
			case r.method == "POST" && !strings.Contains(sp, "{"):
				postCollections = append(postCollections, sp)
			case r.method == "GET" && strings.Contains(sp, "{id}"):
				getDetails = append(getDetails, sp)
			case r.method == "PUT" && strings.Contains(sp, "{id}"):
				putDetails = append(putDetails, sp)
			case r.method == "DELETE" && strings.Contains(sp, "{id}"):
				deleteDetails = append(deleteDetails, sp)
			case r.method == "POST" && strings.Contains(sp, "/{"):
				act := filepath.Base(strings.TrimRight(sp, "/"))
				if act != "{id}" && !strings.Contains(act, "{") {
					actions = append(actions, act)
				}
			}
		}

		listPath := ""
		if pref != "" && !strings.Contains(pref, "{") {
			listPath = pref
		}
		if listPath == "" && len(getCollections) > 0 {
			listPath = getCollections[0]
		}
		if strings.Contains(listPath, "{") {
			listPath = ""
		}

		createPath := ""
		if listPath != "" && contains(postCollections, listPath) {
			createPath = listPath
		} else if len(postCollections) > 0 && (listPath == "" || contains(postCollections, listPath)) {
			createPath = postCollections[0]
		}
		if listPath != "" && createPath != "" && createPath != listPath && !contains(postCollections, listPath) {
			createPath = ""
		}

		detailPath := ""
		if listPath != "" {
			cand := strings.TrimRight(listPath, "/") + "/{id}"
			if contains(getDetails, cand) {
				detailPath = cand
			}
		} else if len(getDetails) > 0 {
			detailPath = getDetails[0]
		}

		updatePath := ""
		if detailPath != "" && contains(putDetails, detailPath) {
			updatePath = detailPath
		} else if listPath != "" {
			cand := strings.TrimRight(listPath, "/") + "/{id}"
			if contains(putDetails, cand) {
				updatePath = cand
			}
		}

		removePath := ""
		if detailPath != "" && contains(deleteDetails, detailPath) {
			removePath = detailPath
		} else if listPath != "" {
			cand := strings.TrimRight(listPath, "/") + "/{id}"
			if contains(deleteDetails, cand) {
				removePath = cand
			}
		}

		_, readOnly := readOnlyModules[key[1]]
		uniqActions := uniqueSorted(actions)
		actionOnly := listPath == "" && (createPath != "" || len(uniqActions) > 0)

		it := moduleMeta{
			domain: key[0], module: key[1], phase: phase,
			list: listPath, detail: detailPath, actions: uniqActions,
			readOnly: readOnly, actionOnly: actionOnly,
		}
		if !readOnly {
			it.create = createPath
			it.update = updatePath
			it.remove = removePath
		}
		items = append(items, it)
	}

	var b strings.Builder
	b.WriteString("// Code generated by go run ./cmd/erp-tools gen-web-meta — DO NOT EDIT\n")
	b.WriteString("import type { ModuleMeta } from '../types'\n\n")
	b.WriteString("export const MODULES: ModuleMeta[] = [\n")
	for _, it := range items {
		b.WriteString("  {\n")
		fmt.Fprintf(&b, "    domain: '%s',\n", escapeTS(it.domain))
		fmt.Fprintf(&b, "    module: '%s',\n", escapeTS(it.module))
		fmt.Fprintf(&b, "    phase: %d,\n", it.phase)
		fmt.Fprintf(&b, "    list: '%s',\n", escapeTS(it.list))
		if it.create != "" {
			fmt.Fprintf(&b, "    create: '%s',\n", escapeTS(it.create))
		}
		if it.detail != "" {
			fmt.Fprintf(&b, "    detail: '%s',\n", escapeTS(it.detail))
		}
		if it.update != "" {
			fmt.Fprintf(&b, "    update: '%s',\n", escapeTS(it.update))
		}
		if it.remove != "" {
			fmt.Fprintf(&b, "    remove: '%s',\n", escapeTS(it.remove))
		}
		b.WriteString("    actions: [")
		for i, a := range it.actions {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "'%s'", escapeTS(a))
		}
		b.WriteString("],\n")
		if it.readOnly {
			b.WriteString("    readOnly: true,\n")
		}
		if it.actionOnly {
			b.WriteString("    actionOnly: true,\n")
		}
		b.WriteString("  },\n")
	}
	b.WriteString("]\n\n")
	b.WriteString("export function findModule(domain: string, module: string) {\n")
	b.WriteString("  return MODULES.find((m) => m.domain === domain && m.module === module)\n")
	b.WriteString("}\n")

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return 0, err
	}
	return len(items), nil
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	raw := strings.Split(line, "|")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		parts = append(parts, strings.TrimSpace(p))
	}
	return parts
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func uniqueSorted(ss []string) []string {
	m := map[string]struct{}{}
	for _, s := range ss {
		m[s] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func escapeTS(s string) string {
	return strings.ReplaceAll(s, `'`, `\'`)
}
