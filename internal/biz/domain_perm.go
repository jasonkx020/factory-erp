package biz

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
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
	"sales/deliveries": {"销售管理", "发货审批"}, "sales/price-locks": {"销售管理", "销售锁价"},
	"sales/quotes": {"销售管理", "历史报价查询"}, "sales/boms": {"销售管理", "销售BOM"},
	"sales/budgets": {"销售管理", "成本预算"}, "sales/outbound-settles": {"销售管理", "出厂结算"},
	// purchase
	"purchase/suppliers": {"采购管理", "供应商管理"}, "purchase/farmers": {"采购管理", "农户档案"},
	"purchase/weigh-tickets": {"采购管理", "过磅收货"}, "purchase/weigh-varieties": {"采购管理", "过磅品种"}, "purchase/trace-batch-codes": {"采购管理", "溯源批号"}, "purchase/farmer-settlements": {"采购管理", "农户结算"},
	"purchase/trace": {"采购管理", "原料溯源"}, "purchase/requests": {"采购管理", "采购申请"},
	"purchase/plans": {"采购管理", "采购计划单"}, "purchase/inbounds": {"采购管理", "采购入库"},
	"purchase/qcs": {"采购管理", "来料质检"}, "purchase/returns": {"采购管理", "采购退货"},
	"purchase/tasks": {"采购管理", "采购任务管理"},
	// production
	"production/tasks": {"生产管理", "生产任务单"}, "production/processes": {"生产管理", "工序设置"},
	"production/dispatches": {"生产管理", "生产派工"}, "production/report-works": {"生产管理", "扫码报工"},
	"production/piecework-summaries": {"生产管理", "计件工资"}, "production/requisitions": {"生产管理", "联动式领料"},
	"production/boms": {"生产管理", "自动BOM"}, "production/workshops": {"生产管理", "车间管理"},
	"production/qc": {"生产管理", "质检管理"}, "production/scraps": {"生产管理", "废料管理"},
	"production/reworks": {"生产管理", "返修单"}, "production/routings": {"生产管理", "工艺流程"},
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
	"hr/tool-issues": {"人事管理", "工具领还"}, "payroll/wage-rates": {"工资管理", "工序工资"},
	"payroll/sheets": {"工资管理", "薪酬核算"},
	// crm / product / asset / approval / report
	"crm/customers": {"客户管理", "CRM客户管理"}, "product/products": {"产品管理", "产品档案"},
	"asset/fixed-assets": {"固定资产管理", "固定资产项目"}, "approval/tasks": {"审批管理", "任务管理"},
	"report/dashboards": {"统计报表", "老板驾驶舱"},
	// IAM used by 人事管理/权限分配
	"iam/roles": {"人事管理", "权限分配"},
	"iam/admin-groups": {"人事管理", "权限分配"},
	"iam/hr-perm-overview": {"人事管理", "权限分配"},
}

var domainPrefixCN = map[string]string{
	"sales": "销售管理", "purchase": "采购管理", "production": "生产管理",
	"inventory": "库存管理", "finance": "财务管理", "hr": "人事管理",
	"payroll": "工资管理", "crm": "客户管理", "system": "系统管理",
	"iam": "系统管理", "approval": "审批管理", "report": "统计报表",
	"product": "产品管理", "asset": "固定资产管理", "notify": "", // notify 仅需登录
	"workflow": "", // 工单 API 在 handler 内做可见性/配置鉴权
}

// allDomainMenus 用于幂等 seed（与前端 ERP_MENUS 对齐）
var allDomainMenus = []struct {
	Domain  string
	Modules []string
}{
	{"销售管理", []string{"销售订单", "自助下单", "询价管理", "合同管理", "修改订单", "发货审批", "预发货管理", "单据打印", "订单复购", "数据排行榜", "销售锁价", "询价审批", "历史报价查询", "销售BOM", "我的订单", "成本预算", "报价计算器", "出厂结算"}},
	{"客户管理", []string{"CRM客户管理", "商机管理", "客户档案", "客户跟进", "资源分配", "保护机制", "释放机制", "询价管理", "导入客户", "线索锁定", "线索隐藏", "任务提醒"}},
	{"采购管理", []string{"供应商管理", "农户档案", "过磅收货", "过磅品种", "溯源批号", "农户结算", "原料溯源", "采购申请", "采购计划单", "采购入库", "来料质检", "采购退货", "采购分析", "历史价格查看", "采购任务管理"}},
	{"生产管理", []string{"多单整合管理", "生产任务单", "图纸分发", "工序设置", "工序管理", "工艺流程", "生产派工", "灵活派发工单", "扫码报工", "计件工资", "加工记录", "计件领料表", "自动BOM", "MRP物料分析", "联动式领料", "车间工作台", "车间管理", "委外加工", "受托加工生产流程管控", "成本隐藏", "一单多商品", "进度跟踪", "质检管理", "返修单", "废料管理"}},
	{"库存管理", []string{"库存查询", "仓管待入库", "地磅台账", "亏料预警", "过量预警", "入库质检", "仓库盘点", "车间盘点", "仓库盘点记录", "销售退皮", "物料调拨耗用", "商品调价组装拆分", "物料转应付", "在途量统计", "待用量统计", "可用量分析", "期初入库", "出入库记录汇总", "采购退货", "箱码管理"}},
	{"产品管理", []string{"产品档案", "产品单位管理", "APP产品排序", "生产规格绑定"}},
	{"固定资产管理", []string{"固定资产类别", "固定资产项目", "固定资产内部转移", "固定资产统计"}},
	{"财务管理", []string{"账目管理", "交易流水账", "收入支出明细", "订单管理", "小程序管理", "凭证管理", "发票管理", "收款核单", "外币结汇", "结汇查询", "分摊撤销", "收款预警", "出纳对账", "预收预付管理", "成本核算", "合同利润", "销售认款", "销售退货退单", "往来调整单", "财务审批", "资金管理", "财务报表", "成本明细溯源表", "月度结转"}},
	{"工资管理", []string{"工人信息管理", "工资批量管理", "工序工资", "薪酬核算", "销售提成"}},
	{"人事管理", []string{"员工档案", "权限分配", "入职登记", "离职登记", "工具领还", "考勤管理", "班次管理", "绩效管理", "请假管理", "考勤明细", "加班补卡统计", "考勤月度统计", "考勤绩效汇总", "外访明细", "备忘录管理", "员工日志"}},
	{"统计报表", []string{"企业报表", "老板驾驶舱", "生产看板", "生产实况", "客户询价查询", "CRM统计", "日统计报表", "毛利润统计", "质检报表", "账目统计", "出入库查询", "收发存明细", "跟进记录查询", "销售重量统计", "产品销售查询", "系统物流查询", "成本利润表", "资产负债表", "现金流量表", "利润表"}},
	{"审批管理", []string{"任务管理", "单据审核", "费用财务审批", "询价财务审批", "询价明细审批", "采购审批", "采购计划单审批", "事务申请审批", "费用申请", "考勤审批"}},
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

func isWeighWarehouseAPI(resourceKey, action string) bool {
	if resourceKey == "purchase/weigh-tickets/by-trace" {
		return true
	}
	if resourceKey != "purchase/weigh-tickets" {
		return false
	}
	switch action {
	case "action:warehouse-confirm", "action:stock-in", "get", "list":
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
	if claimsIsSysAdmin(claims.Roles, claims.Permissions) {
		return true
	}

	write := needWrite(action, method)
	// 仓管扫码定位 / 确认入库：仓管角色，或持有仓管待入库/过磅收货权限
	if isWeighWarehouseAPI(resourceKey, action) &&
		(claimsHasAnyRole(claims, "warehouse", "仓管", "仓管员") || claimsHasWeighWarehousePerm(claims.Permissions, write)) {
		return true
	}
	if dm.Module != "" {
		viewCode := dm.Domain + ":" + dm.Module + ":查看"
		editCode := dm.Domain + ":" + dm.Module + ":编辑"
		if write {
			if !claimsHasCode(claims.Permissions, editCode) {
				// 审批动作也可用审批码
				approve := dm.Domain + ":" + dm.Module + ":审批"
				if !claimsHasCode(claims.Permissions, approve) {
					c.AbortWithStatusJSON(http.StatusForbidden, api.Response{Code: 0, Msg: "PERM_DENIED"})
					return false
				}
			}
			return true
		}
		if !claimsHasCode(claims.Permissions, viewCode, editCode) {
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
				_, _ = db.Exec(`INSERT OR IGNORE INTO iam_permission(code, name, domain, module, action) VALUES(?,?,?,?,?)`,
					code, act+mod, d.Domain, mod, act)
			}
		}
	}
	// 财务审批动作
	for _, mod := range []string{"财务审批", "凭证管理", "收款核单"} {
		code := "财务管理:" + mod + ":审批"
		_, _ = db.Exec(`INSERT OR IGNORE INTO iam_permission(code, name, domain, module, action) VALUES(?,?,?,?,?)`,
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
		_, _ = db.Exec(`INSERT OR IGNORE INTO iam_role_permission(role_id, permission_id) VALUES(?,?)`, roleID, pid)
	}
}
