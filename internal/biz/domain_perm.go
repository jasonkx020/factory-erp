package biz

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
	"erp/internal/security"
)

type domainModule struct {
	Domain string
	Module string
}

// resourceDomainModule 精确映射（一对多时取主模块）
var resourceDomainModule = map[string]domainModule{
	// sales
	"sales/orders": {"销售管理", "销售订单"}, "sales/inquiries": {"销售管理", "询价管理"},
	"sales/contracts": {"销售管理", "合同管理"}, "sales/pre-ships": {"销售管理", "预发货管理"},
	"sales/pre-shipments": {"销售管理", "预发货管理"},
	"sales/deliveries":    {"销售管理", "发货审批"}, "sales/price-locks": {"销售管理", "销售锁价"},
	"sales/quotes": {"销售管理", "历史报价查询"}, "sales/quote-histories": {"销售管理", "历史报价查询"},
	"sales/quote-calculator": {"销售管理", "报价计算器"},
	"sales/boms":             {"销售管理", "销售BOM"}, "sales/sales-boms": {"销售管理", "销售BOM"},
	"sales/budgets": {"销售管理", "成本预算"}, "sales/cost-budgets": {"销售管理", "成本预算"},
	"sales/outbound-settles": {"销售管理", "出厂结算"},
	"sales/self-orders":      {"销售管理", "自助下单"}, "sales/my-orders": {"销售管理", "我的订单"},
	// purchase
	"purchase/suppliers": {"采购管理", "供应商管理"}, "purchase/farmers": {"采购管理", "农户档案"},
	"purchase/weigh-tickets": {"采购管理", "过磅收货"}, "purchase/weigh-varieties": {"采购管理", "过磅品种"}, "purchase/trace-batch-codes": {"采购管理", "溯源批号"}, "purchase/farmer-settlements": {"采购管理", "农户结算"},
	"purchase/flow-graphs": {"采购管理", "过磅流程编排"},
	"purchase/role-users": {"采购管理", "过磅收货"},
	"purchase/trace":      {"采购管理", "原料溯源"}, "purchase/requests": {"采购管理", "采购申请"},
	"purchase/plans": {"采购管理", "采购计划单"}, "purchase/inbounds": {"采购管理", "采购入库"},
	"purchase/qcs": {"采购管理", "来料质检"}, "purchase/returns": {"采购管理", "采购退货"},
	"purchase/tasks": {"采购管理", "采购任务管理"},
	// production
	"production/tasks": {"生产管理", "生产任务单"}, "production/processes": {"生产管理", "工序定义"},
	"production/shifts":     {"生产管理", "产线班次"},
	"production/dispatches": {"生产管理", "例外派岗"}, "production/report-works": {"生产管理", "工序流水"},
	"production/workshop-workbench":  {"生产管理", "车间工作台"},
	"production/process-wip":         {"生产管理", "工序在制"},
	"production/scan":                {"生产管理", "工序流水"},
	"production/scan/resolve":        {"生产管理", "工序流水"},
	"production/piecework-summaries": {"生产管理", "计件工资"}, "production/requisitions": {"生产管理", "联动式领料"},
	"production/station-flow-logs": {"生产管理", "工序流水"},
	"production/boms":              {"生产管理", "自动BOM"},
	"production/qc":                {"生产管理", "质检管理"}, "production/scraps": {"生产管理", "废料管理"},
	"production/process-returns": {"生产管理", "退库未用完还仓"},
	"production/board-issues":    {"生产管理", "工序流水"},
	"production/process-issues":  {"生产管理", "工序流水"},
	"production/process-stock-ins": {"生产管理", "工序流水"},
	"production/board-moves":     {"生产管理", "工序流水"},
	"production/board-close":     {"生产管理", "工序流水"},
	"production/trace-productions":   {"生产管理", "溯源生产"},
	"production/process-yields":  {"生产管理", "工序扣损"},
	"production/reworks":         {"生产管理", "返修单"}, "production/routings": {"生产管理", "工艺流程"},
	"production/flow-graphs": {"生产管理", "工艺流程"}, "production/flow-rules": {"生产管理", "工艺流程"},
	// inventory
	"inventory/balances": {"库存管理", "库存查询"}, "inventory/stock-txns": {"库存管理", "出入库记录汇总"},
	"inventory/warehouses": {"库存管理", "库存查询"}, "inventory/stocktakes": {"库存管理", "仓库盘点"},
	"inventory/transfers": {"库存管理", "物料调拨耗用"}, "inventory/box-codes": {"库存管理", "箱码管理"},
	"inventory/openings": {"库存管理", "期初入库"}, "inventory/inbound-qcs": {"库存管理", "入库质检"},
	// finance
	"finance/subjects": {"财务管理", "账目管理"}, "finance/ledger": {"财务管理", "交易流水账"},
	"finance/income-expenses": {"财务管理", "收入支出明细"}, "finance/orders": {"财务管理", "订单管理"},
	"finance/miniprogram": {"财务管理", "小程序管理"}, "finance/vouchers": {"财务管理", "凭证管理"},
	"finance/invoices": {"财务管理", "发票管理"}, "finance/writeoffs": {"财务管理", "收款核单"},
	"finance/fx": {"财务管理", "外币结汇"}, "finance/fx-settlements": {"财务管理", "外币结汇"},
	"finance/allocations": {"财务管理", "分摊撤销"}, "finance/receipt-alerts": {"财务管理", "收款预警"},
	"finance/reconciles": {"财务管理", "出纳对账"}, "finance/prepays": {"财务管理", "预收预付管理"},
	"finance/cost-accountings": {"财务管理", "成本核算"}, "finance/contract-profits": {"财务管理", "合同利润"},
	"finance/recognitions": {"财务管理", "销售认款"}, "finance/return-finances": {"财务管理", "销售退货退单"},
	"finance/arap": {"财务管理", "往来调整单"}, "finance/approvals": {"财务管理", "财务审批"},
	"finance/funds": {"财务管理", "资金管理"}, "finance/fund-accounts": {"财务管理", "资金管理"},
	"finance/fund-transfers": {"财务管理", "资金管理"}, "finance/statements": {"财务管理", "财务报表"},
	"finance/cost-traces": {"财务管理", "成本明细溯源表"}, "finance/month-closes": {"财务管理", "月度结转"},
	// hr / payroll
	"hr/employees": {"人事管理", "员工档案"}, "hr/onboards": {"人事管理", "入职登记"},
	"hr/departments": {"人事管理", "公司架构"}, "hr/work-teams": {"人事管理", "员工档案"},
	"hr/job-titles":  {"人事管理", "岗位管理"},
	"hr/tool-issues": {"人事管理", "工具领还"}, "system/personnel-transfers": {"人事管理", "人事调动"},
	"payroll/wage-rates":   {"工资管理", "工序工资"},
	"payroll/sheets":       {"工资管理", "薪酬核算"},
	"payroll/work-records": {"工资管理", "员工工作台账"},
	// crm / product / asset / approval / report
	"crm/customers": {"客户管理", "CRM客户管理"}, "product/products": {"产品管理", "产品档案"},
	"asset/fixed-assets": {"固定资产管理", "固定资产项目"}, "approval/tasks": {"审批管理", "任务管理"},
	"report/dashboards/production":           {"统计报表", "生产看板"},
	"report/dashboards/live":               {"统计报表", "生产实况"},
	"report/dashboards/warehouse":          {"统计报表", "三仓库存概览"},
	"report/daily":                         {"统计报表", "日经营快照"},
	"report/inbound-daily":                 {"统计报表", "原料入场日报"},
	"report/piecework-daily":               {"统计报表", "计件日结汇总"},
	"report/yield-analysis":                {"统计报表", "工序扣损收率分析"},
	"report/trace-progress":                {"统计报表", "溯源批进度查询"},
	"report/farmer-settlement-summary":     {"统计报表", "农户结算对账汇总"},
	"report/payroll-reconcile":             {"统计报表", "薪酬核算对账"},
	"report/cost-period-summary":           {"统计报表", "成本期间汇总"},
	"report/stock-ledger":                  {"统计报表", "收发存明细"},
	"report/qc":                            {"统计报表", "生产看板"},
	"report/dashboards":                    {"统计报表", "生产看板"},
	// IAM used by 人事管理/角色管理
	"iam/roles":            {"人事管理", "角色管理"},
	"iam/admin-groups":     {"人事管理", "角色管理"},
	"iam/hr-perm-overview": {"人事管理", "角色管理"},
}

// modulePermAliases 模块重命名后兼容旧权限码
var modulePermAliases = map[string][]string{
	"公司架构": {"部门管理"},
	"工序流水": {"过站记录", "扫码报工", "加工记录"},
	"过站记录": {"扫码报工", "加工记录", "工序流水"},
	"例外派岗": {"生产派工"},
	"工序定义": {"工序设置", "工序管理"},
	"角色管理": {"权限分配"},
}

var domainPrefixCN = map[string]string{
	"sales": "销售管理", "purchase": "采购管理", "production": "生产管理",
	"inventory": "库存管理", "finance": "财务管理", "hr": "人事管理",
	"payroll": "工资管理", "crm": "客户管理", "system": "系统管理",
	"iam": "系统管理", "approval": "审批管理", "report": "统计报表",
	"product": "产品管理", "asset": "固定资产管理", "notify": "", // notify 仅需登录
	"workflow": "", // 工单 API 在 handler 内做可见性/配置鉴权
}

// allDomainMenus 用于幂等 seed（与前端 CASSAVA_PRODUCT_SCOPE 对齐）
var allDomainMenus = []struct {
	Domain  string
	Modules []string
}{
	{"采购管理", []string{"农户档案", "过磅收货", "过磅流程编排", "过磅品种", "溯源批号", "农户结算", "原料溯源", "来料质检"}},
	{"库存管理", []string{"库存查询", "仓管待入库", "箱码管理", "出入库记录汇总", "可用量分析", "亏料预警", "过量预警", "在途量统计", "待用量统计"}},
	{"生产管理", []string{"工序定义", "工艺流程", "产线班次", "例外派岗", "工序流水", "计件工资", "工序在制", "溯源生产", "工序扣损", "退库未用完还仓"}},
	{"产品管理", []string{"产品档案", "产品单位管理", "生产规格绑定"}},
	{"工资管理", []string{"工人信息管理", "工资批量管理", "工序工资", "薪酬核算", "员工工作台账"}},
	{"人事管理", []string{"员工档案", "公司架构", "角色管理"}},
	{"财务管理", []string{"成本核算", "成本明细溯源表"}},
	{"统计报表", []string{
		"生产看板", "生产实况", "三仓库存概览",
		"日经营快照", "原料入场日报", "计件日结汇总",
		"工序扣损收率分析", "收发存明细", "溯源批进度查询", "农户结算对账汇总",
		"薪酬核算对账", "成本期间汇总",
	}},
	{"系统管理", []string{"基础设置", "生产设置", "自定义权限", "登录控制", "批量核算工资", "操作日志"}},
}

func resolveDomainModule(resourceKey string) (domainModule, bool) {
	if m, ok := resourceDomainModule[resourceKey]; ok {
		return m, true
	}
	// longest prefix match
	best := ""
	var hit domainModule
	for k, v := range resourceDomainModule {
		if strings.HasPrefix(resourceKey, k+"/") || resourceKey == k {
			if len(k) > len(best) {
				best = k
				hit = v
			}
		}
	}
	if best != "" {
		return hit, true
	}
	seg := strings.Split(resourceKey, "/")
	if len(seg) == 0 {
		return domainModule{}, false
	}
	dom, ok := domainPrefixCN[seg[0]]
	if !ok || dom == "" {
		return domainModule{}, false
	}
	return domainModule{Domain: dom, Module: ""}, true
}

func needWrite(action, method string) bool {
	if action == "list" || action == "get" {
		return false
	}
	if action == "replace" && (method == "GET" || method == "") {
		return false
	}
	if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
		return true
	}
	if strings.HasPrefix(action, "action:") || action == "create" || action == "update" || action == "delete" || action == "replace" {
		return true
	}
	return false
}

func modulePermCodes(domain, module string, write bool) []string {
	suffix := "查看"
	if write {
		suffix = "编辑"
	}
	codes := []string{domain + ":" + module + ":" + suffix}
	for _, alias := range modulePermAliases[module] {
		codes = append(codes, domain+":"+alias+":"+suffix)
	}
	return codes
}

func isProductionFieldAPI(resourceKey, action string) bool {
	if strings.HasPrefix(resourceKey, "production/scan") {
		return true
	}
	if strings.HasPrefix(resourceKey, "production/board-issues") || strings.HasPrefix(resourceKey, "production/board-moves") ||
		strings.HasPrefix(resourceKey, "production/process-issues") || strings.HasPrefix(resourceKey, "production/process-stock-ins") {
		return true
	}
	if strings.HasPrefix(resourceKey, "production/trace-productions") {
		return true
	}
	if strings.Contains(resourceKey, "piecework-summaries/mine") {
		return true
	}
	if strings.Contains(resourceKey, "report-works") && (strings.Contains(action, "confirm") || action == "action:confirm") {
		return true
	}
	return false
}

func claimsHasProductionFieldRole(claims *security.Claims) bool {
	return claimsHasAnyRole(claims, "piece", "fixed", "foreman", "line_worker", "计件工", "固定工", "车间主任")
}

func isWeighWarehouseAPI(resourceKey, action string) bool {
	if resourceKey == "purchase/weigh-tickets/by-trace" {
		return true
	}
	if resourceKey == "purchase/role-users" && (action == "list" || action == "get") {
		return true
	}
	if resourceKey != "purchase/weigh-tickets" {
		return false
	}
	switch action {
	case "action:warehouse-confirm", "action:warehouse-return", "action:stock-in", "action:box-stock-in", "action:box-stock-in-complete", "get", "list":
		return true
	default:
		return false
	}
}

// isWeighQcAPI 质检仅判定：查过磅 + action:qc；不含入厂/分板。
func isWeighQcAPI(resourceKey, action string) bool {
	if resourceKey != "purchase/weigh-tickets" {
		return false
	}
	switch action {
	case "action:qc", "get", "list":
		return true
	default:
		return false
	}
}

// isWarehouseProductionPendingAPI 仓管待审队列：领料/退库/入库过账，不依赖「工序流水」权限。
func isWarehouseProductionPendingAPI(resourceKey, action string) bool {
	if !strings.HasPrefix(resourceKey, "production/process-issues") &&
		!strings.HasPrefix(resourceKey, "production/process-stock-ins") {
		return false
	}
	switch action {
	case "list", "get",
		"action:issue-approve", "action:issue-reject",
		"action:return-approve", "action:return-reject",
		"action:approve", "action:reject":
		return true
	default:
		return false
	}
}

func claimsHasWeighWarehousePerm(perms []string, write bool) bool {
	if write {
		return claimsHasCode(perms,
			"库存管理:仓管待入库:编辑",
			"采购管理:过磅收货:编辑",
		)
	}
	return claimsHasCode(perms,
		"库存管理:仓管待入库:查看",
		"库存管理:仓管待入库:编辑",
		"采购管理:过磅收货:查看",
		"采购管理:过磅收货:编辑",
	)
}

func domainHasAnyPerm(perms []string, domain string, write bool) bool {
	prefix := domain + ":"
	for _, p := range perms {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		if write {
			if strings.HasSuffix(p, ":编辑") || strings.HasSuffix(p, ":审批") || strings.HasSuffix(p, ":新增") {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

// CheckAPIPerm 全域 API 二次鉴权。系统管理走原逻辑；业务域按 域:模块:查看|编辑。
// notify 等仅需登录的资源直接放行。返回 false 表示已写拒绝响应。
func CheckAPIPerm(c *gin.Context, resourceKey, action, method string) bool {
	if isSystemProtectedResource(resourceKey) {
		return CheckSystemAPIPerm(c, resourceKey, action, method)
	}
	dm, ok := resolveDomainModule(resourceKey)
	if !ok {
		// 未映射且无域前缀：仅 JWT（如 notify）
		return true
	}

	claims := middleware.Claims(c)
	if claims == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.Response{Code: 0, Msg: "UNAUTHORIZED"})
		return false
	}
	if claims.UserID == 1 || claimsIsSysAdmin(claims.Roles, claims.Permissions) {
		return true
	}

	write := needWrite(action, method)
	// 仓管扫码定位 / 确认入库：仓管角色，或持有仓管待入库/过磅收货权限
	if isWeighWarehouseAPI(resourceKey, action) &&
		(claimsHasAnyRole(claims, "warehouse", "仓管", "仓管员") || claimsHasWeighWarehousePerm(claims.Permissions, write)) {
		return true
	}
	// 工序领退料/入库待审：仓管角色或仓管待入库权限，无需「生产管理:工序流水」
	if isWarehouseProductionPendingAPI(resourceKey, action) &&
		(claimsHasAnyRole(claims, "warehouse", "foreman", "admin", "仓管", "仓管员") ||
			claimsHasWeighWarehousePerm(claims.Permissions, write)) {
		return true
	}
	// 质检判定：可查过磅与提交 qc；入厂类 action 不走此旁路（仍由 warehouse 拦截）
	if isWeighQcAPI(resourceKey, action) &&
		claimsHasAnyRole(claims, "qc", "质检", "质检员", "purchase") {
		return true
	}
	// App 现场过站：计件/固定/班组长角色可访问 scan / confirm / 我的计件
	if isProductionFieldAPI(resourceKey, action) && claimsHasProductionFieldRole(claims) {
		return true
	}
	// 部门下拉：员工档案/入职可查看列表；写操作仍走「公司架构」
	if resourceKey == "hr/departments" && !write {
		codes := append(modulePermCodes("人事管理", "公司架构", false), modulePermCodes("人事管理", "公司架构", true)...)
		codes = append(codes, modulePermCodes("人事管理", "员工档案", false)...)
		codes = append(codes, modulePermCodes("人事管理", "员工档案", true)...)
		codes = append(codes, modulePermCodes("人事管理", "入职登记", false)...)
		codes = append(codes, modulePermCodes("人事管理", "入职登记", true)...)
		if claimsHasCode(claims.Permissions, codes...) {
			return true
		}
	}
	if dm.Module != "" {
		viewCodes := modulePermCodes(dm.Domain, dm.Module, false)
		editCodes := modulePermCodes(dm.Domain, dm.Module, true)
		// 人事调动从系统管理迁入：兼容旧权限码
		if dm.Domain == "人事管理" && dm.Module == "人事调动" {
			viewCodes = append(viewCodes, "系统管理:人事调动:查看")
			editCodes = append(editCodes, "系统管理:人事调动:编辑")
		}
		if write {
			if !claimsHasCode(claims.Permissions, editCodes...) {
				approve := dm.Domain + ":" + dm.Module + ":审批"
				approveAliases := []string{approve}
				for _, alias := range modulePermAliases[dm.Module] {
					approveAliases = append(approveAliases, dm.Domain+":"+alias+":审批")
				}
				if !claimsHasCode(claims.Permissions, approveAliases...) {
					c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
					return false
				}
			}
			return true
		}
		if !claimsHasCode(claims.Permissions, append(viewCodes, editCodes...)...) {
			c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
			return false
		}
		return true
	}
	// 仅域前缀：持有该域任意相关权限即可
	if !domainHasAnyPerm(claims.Permissions, dm.Domain, write) {
		c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
		return false
	}
	return true
}

// EnsureDomainPermissions 幂等植入各域模块 查看/编辑 权限并绑定 sys_admin。
func EnsureDomainPermissions(db *sql.DB) {
	EnsureSystemAdminPermissions(db)
	for _, d := range allDomainMenus {
		for _, mod := range d.Modules {
			for _, act := range []string{"查看", "编辑"} {
				code := d.Domain + ":" + mod + ":" + act
				_, _ = db.Exec(`INSERT INTO iam_permission(code, name, domain, module, action) VALUES(?,?,?,?,?)
ON CONFLICT (code) DO NOTHING`,
					code, act+mod, d.Domain, mod, act)
			}
		}
	}
	// 财务审批动作
	for _, mod := range []string{"财务审批", "凭证管理", "收款核单"} {
		code := "财务管理:" + mod + ":审批"
		_, _ = db.Exec(`INSERT INTO iam_permission(code, name, domain, module, action) VALUES(?,?,?,?,?)
ON CONFLICT (code) DO NOTHING`,
			code, "审批"+mod, "财务管理", mod, "审批")
	}
	bindAllPermissionsToSysAdmin(db)
}

// bindAllPermissionsToSysAdmin 先收集 id 再写入，避免 SQLite 在 Rows 未关闭时嵌套 Exec 失败。
func bindAllPermissionsToSysAdmin(db *sql.DB) {
	var roleID int64
	if err := db.QueryRow(`SELECT id FROM iam_role WHERE code='sys_admin' LIMIT 1`).Scan(&roleID); err != nil || roleID == 0 {
		return
	}
	rows, err := db.Query(`SELECT id FROM iam_permission WHERE COALESCE(is_deleted,0)=0`)
	if err != nil {
		return
	}
	ids := make([]int64, 0, 512)
	for rows.Next() {
		var pid int64
		if rows.Scan(&pid) == nil && pid > 0 {
			ids = append(ids, pid)
		}
	}
	_ = rows.Close()
	for _, pid := range ids {
		_, _ = db.Exec(`INSERT INTO iam_role_permission(role_id, permission_id) VALUES(?,?)
ON CONFLICT (role_id, permission_id) DO NOTHING`, roleID, pid)
	}
}
