/** 管理端：域菜单模块 → 专用页面（木薯产线交付范围） */
export const ADMIN_SPECIAL_MODULE_PATHS: Record<string, string> = {
  '采购管理/农户档案': '/purchase/hub/farmers',
  '采购管理/过磅收货': '/purchase/hub/weigh',
  '采购管理/过磅流程编排': '/purchase/hub/flow-graphs',
  '采购管理/过磅品种': '/purchase/hub/varieties',
  '采购管理/溯源批号': '/purchase/hub/trace-batches',
  '采购管理/农户结算': '/purchase/hub/settlements',
  '采购管理/原料溯源': '/purchase/hub/trace',
  '采购管理/来料质检': '/purchase/hub/qcs',
  '库存管理/库存查询': '/inventory/hub/balances',
  '库存管理/仓管待入库': '/inventory/hub/inbound',
  '库存管理/箱码管理': '/inventory/hub/boxes',
  '库存管理/出入库记录汇总': '/inventory/hub/stock-txns',
  '库存管理/可用量分析': '/inventory/hub/availability',
  '库存管理/亏料预警': '/inventory/hub/shortage',
  '库存管理/过量预警': '/inventory/hub/excess',
  '库存管理/在途量统计': '/inventory/hub/in-transits',
  '库存管理/待用量统计': '/inventory/hub/reservations',
  '生产管理/工序定义': '/production/hub/processes',
  '生产管理/工艺流程': '/production/hub/routings',
  '生产管理/产线班次': '/production/hub/shifts',
  '生产管理/例外派岗': '/production/hub/dispatches',
  '生产管理/工序流水': '/production/hub/reports',
  '生产管理/计件工资': '/production/hub/piecework',
  '生产管理/工序在制': '/production/hub/process-wip',
  '生产管理/溯源生产': '/production/hub/trace-production',
  '生产管理/工序扣损': '/production/hub/process-yield',
  '生产管理/退库未用完还仓': '/production/hub/process-returns',
  '产品管理/产品档案': '/product/hub/products',
  '产品管理/产品单位管理': '/product/hub/units',
  '产品管理/生产规格绑定': '/product/hub/specs',
  '财务管理/成本核算': '/finance/hub/cost-accountings',
  '财务管理/成本明细溯源表': '/finance/hub/cost-traces',
  '财务管理/资金管理': '/finance/hub/funds',
  '财务管理/交易流水账': '/finance/hub/ledger',
  '财务管理/农户应付': '/finance/hub/payables',
  '工资管理/工人信息管理': '/payroll/workers',
  '工资管理/工资批量管理': '/payroll/batch',
  '工资管理/工序工资': '/payroll/wage-rates',
  '工资管理/薪酬核算': '/payroll/sheets',
  '工资管理/员工工作台账': '/payroll/work-records',
  '人事管理/员工档案': '/hr/employees',
  '人事管理/岗位管理': '/hr/job-titles',
  '人事管理/公司架构': '/hr/departments',
  '人事管理/角色管理': '/hr/roles',
  '统计报表/生产看板': '/report/hub/production-board',
  '统计报表/生产实况': '/report/hub/live',
  '统计报表/三仓库存概览': '/report/hub/warehouse',
  '统计报表/日经营快照': '/report/hub/daily',
  '统计报表/原料入场日报': '/report/hub/inbound-daily',
  '统计报表/计件日结汇总': '/report/hub/piecework-daily',
  '统计报表/工序扣损收率分析': '/report/hub/yield-analysis',
  '统计报表/收发存明细': '/report/hub/stock-ledger',
  '统计报表/溯源批进度查询': '/report/hub/trace-progress',
  '统计报表/农户结算对账汇总': '/report/hub/farmer-settlement-summary',
  '统计报表/薪酬核算对账': '/report/hub/payroll-reconcile',
  '统计报表/成本期间汇总': '/report/hub/cost-period-summary',
  '系统管理/基础设置': '/system/settings',
  '系统管理/生产设置': '/system/production-settings',
  '系统管理/自定义权限': '/iam/permissions',
  '系统管理/登录控制': '/iam/login-policy',
  '系统管理/批量核算工资': '/system/batch-payroll-jobs',
  '系统管理/操作日志': '/automation/logs',
}

export function adminSpecialPath(domain: string, module: string): string | undefined {
  return ADMIN_SPECIAL_MODULE_PATHS[`${domain}/${module}`]
}

/** 当前路由是否对应该菜单项（含专用页） */
export function isAdminModuleActive(
  domain: string,
  module: string,
  routePath: string,
  routeDomain?: string,
  routeModule?: string,
): boolean {
  const special = adminSpecialPath(domain, module)
  if (special) {
    if (special === '/') return routePath === '/' || routePath === ''
    return routePath === special || routePath.startsWith(special + '/')
  }
  return routeDomain === domain && routeModule === module
}

/** path → { domain, module } 反查；同前缀时优先最长精确匹配 */
export function adminModuleForPath(path: string): { domain: string; module: string } | null {
  let best: { domain: string; module: string; len: number } | null = null
  for (const [key, p] of Object.entries(ADMIN_SPECIAL_MODULE_PATHS)) {
    if (path === p || path.startsWith(p + '/')) {
      if (!best || p.length > best.len) {
        const [domain, module] = key.split('/')
        best = { domain, module, len: p.length }
      }
    }
  }
  return best ? { domain: best.domain, module: best.module } : null
}
