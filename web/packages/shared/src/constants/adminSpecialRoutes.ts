/** 管理端：域菜单模块 → 专用页面（非通用 ModuleCrud） */
export const ADMIN_SPECIAL_MODULE_PATHS: Record<string, string> = {
  '销售管理/出厂结算': '/sales/outbound-settle',
  '库存管理/仓管待入库': '/warehouse/inbound',
  '库存管理/地磅台账': '/inventory/ledger',
  '生产管理/加工记录': '/production/process-reports',
  '生产管理/计件领料表': '/production/piece-issue',
  '生产管理/工艺流程': '/automation/routing',
  '人事管理/工具领还': '/hr/tool-issues',
  '采购管理/原料溯源': '/automation/trace',
  '系统管理/操作日志': '/automation/logs',
  '系统管理/数据修复': '/automation/repairs',
  '系统管理/业务闭环': '/loops',
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

/** path → { domain, module } 反查，用于展开侧栏 */
export function adminModuleForPath(path: string): { domain: string; module: string } | null {
  for (const [key, p] of Object.entries(ADMIN_SPECIAL_MODULE_PATHS)) {
    if (path === p || path.startsWith(p + '/')) {
      const [domain, module] = key.split('/')
      return { domain, module }
    }
  }
  return null
}
