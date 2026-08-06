/** Employee-facing module keys shared by Flutter employee app. */
export type EmployeeModuleKey =
  | 'workshop'
  | 'worker'
  | 'receiving'
  | 'warehouse'
  | 'sales'
  | 'assets'
  | 'collab'
  | 'knowledge'
  | 'mine'

export const EMPLOYEE_MODULES: Array<{
  key: EmployeeModuleKey
  title: string
  desc: string
}> = [
  { key: 'workshop', title: '车间工作台', desc: '扫码、派工、灵活派发、质检废料' },
  { key: 'worker', title: '工人报工', desc: '双扫报工、今日核对与联动领料' },
  { key: 'receiving', title: '过磅收货', desc: '农户过磅、质检、出码推仓' },
  { key: 'warehouse', title: '仓管作业', desc: '待入库、库存、箱码、盘点' },
  { key: 'sales', title: '销售外勤', desc: '订单、发货进度、报价与跟进' },
  { key: 'assets', title: '固定资产', desc: '资产查询与内部转移申请' },
  { key: 'collab', title: '收款协同', desc: '收款预警处理与销售认款' },
  { key: 'knowledge', title: '资料中心', desc: '知识库、图纸、公告、学堂' },
  { key: 'mine', title: '我的', desc: '打卡、假勤、审批、工资与消息' },
]

function isAdmin(perms: string[], roles: string[]) {
  return perms.includes('*:*:*') || roles.includes('sys_admin') || roles.includes('系统管理员')
}

function matchAny(perms: string[], needles: string[]) {
  const joined = perms.join('\n')
  return needles.some((n) => joined.includes(n))
}

export function canAccessEmployeeModule(
  module: EmployeeModuleKey,
  permissions: string[],
  roles: string[] = [],
): boolean {
  if (isAdmin(permissions, roles)) return true
  switch (module) {
    case 'workshop':
      return (
        matchAny(permissions, ['生产管理', '派工', '车间', '库存管理', '扫码报工', 'production', 'inventory']) ||
        roles.includes('foreman') ||
        roles.includes('车间主任')
      )
    case 'worker':
      return (
        matchAny(permissions, ['生产管理', '扫码报工', '计件', '工资管理', 'payroll', '报工']) ||
        roles.includes('piece') ||
        roles.includes('fixed') ||
        roles.includes('计件工') ||
        roles.includes('固定工')
      )
    case 'receiving':
      return (
        roles.includes('purchase') ||
        roles.includes('采购员') ||
        roles.includes('qc') ||
        roles.includes('质检') ||
        matchAny(permissions, ['采购管理', '过磅', '农户', 'purchase', 'weigh'])
      )
    case 'warehouse':
      return (
        roles.includes('warehouse') ||
        roles.includes('仓管员') ||
        matchAny(permissions, ['库存管理', '仓管', 'warehouse', '入库'])
      )
    case 'sales':
      return (
        roles.includes('sales') ||
        roles.includes('销售员') ||
        matchAny(permissions, ['销售管理', '客户', '询价', '订单', 'sales', 'crm', 'CRM'])
      )
    case 'assets':
      return (
        matchAny(permissions, ['固定资产', 'asset', 'fixed']) ||
        roles.includes('warehouse') ||
        roles.includes('仓管员') ||
        roles.includes('foreman')
      )
    case 'collab':
      return (
        roles.includes('sales') ||
        roles.includes('销售员') ||
        roles.includes('finance') ||
        roles.includes('财务') ||
        matchAny(permissions, ['财务', '认款', '收款', 'finance', 'sales'])
      )
    case 'knowledge':
    case 'mine':
      return true
    default:
      return false
  }
}

export function visibleEmployeeModules(permissions: string[], roles: string[] = []) {
  return EMPLOYEE_MODULES.filter((m) => canAccessEmployeeModule(m.key, permissions, roles))
}

/** 员工 App 主角色解析优先级（与 mobile/lib/core/role_codes.dart 对齐） */
export const EMPLOYEE_ROLE_PRIORITY = [
  'purchase',
  '采购员',
  'qc',
  '质检',
  'warehouse',
  '仓管员',
  'foreman',
  '车间主任',
  'piece',
  '计件工',
  'fixed',
  '固定工',
  'sales',
  '销售员',
  'finance',
  '财务',
  'sys_admin',
  '系统管理员',
] as const

export type EmployeeWorkbenchRole =
  | 'receiving'
  | 'warehouse'
  | 'workshop'
  | 'worker'
  | 'sales'
  | 'collab'
  | 'admin'
  | 'none'

export function workbenchRoleFromCode(code: string): EmployeeWorkbenchRole {
  switch (code) {
    case 'purchase':
    case '采购员':
    case 'qc':
    case '质检':
      return 'receiving'
    case 'warehouse':
    case '仓管员':
      return 'warehouse'
    case 'foreman':
    case '车间主任':
      return 'workshop'
    case 'piece':
    case '计件工':
    case 'fixed':
    case '固定工':
      return 'worker'
    case 'sales':
    case '销售员':
      return 'sales'
    case 'finance':
    case '财务':
      return 'collab'
    case 'sys_admin':
    case '系统管理员':
      return 'admin'
    default:
      return 'none'
  }
}

/** 从 IAM roles 解析员工 App 主作业角色 */
export function resolvePrimaryWorkbenchRole(roles: string[]): EmployeeWorkbenchRole {
  for (const code of EMPLOYEE_ROLE_PRIORITY) {
    if (roles.includes(code)) {
      const wr = workbenchRoleFromCode(code)
      if (wr !== 'none') return wr
    }
  }
  return 'none'
}
