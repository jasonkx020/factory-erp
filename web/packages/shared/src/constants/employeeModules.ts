/** Employee-facing module keys shared by Web employee app and Flutter. */
export type EmployeeModuleKey = 'workshop' | 'worker' | 'sales' | 'warehouse'

export const EMPLOYEE_MODULES: Array<{
  key: EmployeeModuleKey
  title: string
  desc: string
}> = [
  { key: 'workshop', title: '车间工作台', desc: '扫码、流转、任务、派工、工序与库存' },
  { key: 'worker', title: '工人报工', desc: '双扫报工、今日核对与考勤提醒' },
  { key: 'warehouse', title: '仓管入库', desc: '采购推送待办、溯源核对、确认入库' },
  { key: 'sales', title: '销售外勤', desc: '订单、询价、客户跟进与任务提醒' },
]

function isAdmin(perms: string[], roles: string[]) {
  return perms.includes('*:*:*') || roles.includes('sys_admin') || roles.includes('系统管理员')
}

function matchAny(perms: string[], needles: string[]) {
  const joined = perms.join('\n')
  return needles.some((n) => joined.includes(n))
}

/** Decide module visibility from IAM permission codes (role union). */
export function canAccessEmployeeModule(
  module: EmployeeModuleKey,
  permissions: string[],
  roles: string[] = [],
): boolean {
  if (isAdmin(permissions, roles)) return true
  switch (module) {
    case 'workshop':
      return matchAny(permissions, [
        '生产管理',
        '派工',
        '车间',
        '库存管理',
        '扫码报工',
        'production',
        'inventory',
      ])
    case 'worker':
      return matchAny(permissions, [
        '生产管理',
        '扫码报工',
        '计件',
        '工资管理',
        'payroll',
        '报工',
      ])
    case 'warehouse':
      return (
        roles.includes('warehouse') ||
        roles.includes('仓管员') ||
        matchAny(permissions, ['库存管理', '仓管', 'warehouse', '入库'])
      )
    case 'sales':
      return matchAny(permissions, ['销售管理', '客户', '询价', '订单', 'sales', 'crm', 'CRM'])
    default:
      return false
  }
}

export function visibleEmployeeModules(permissions: string[], roles: string[] = []) {
  return EMPLOYEE_MODULES.filter((m) => canAccessEmployeeModule(m.key, permissions, roles))
}
