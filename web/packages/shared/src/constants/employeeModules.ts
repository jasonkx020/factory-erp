/** 木薯产线试点：现场工序模块（销售/资产/协同已移出默认 App） */
export type EmployeeModuleKey =
  | 'station'
  | 'receiving'
  | 'warehouse'
  | 'workshop'
  | 'mine'

export const EMPLOYEE_MODULES: Array<{
  key: EmployeeModuleKey
  title: string
  desc: string
}> = [
  { key: 'station', title: '生产', desc: '扫工牌+板码，按 kg 领取/退库/进下道' },
  { key: 'receiving', title: '采购', desc: '农户过磅、质检、出码推仓' },
  { key: 'warehouse', title: '仓管作业', desc: '待入库、库存、板码、盘点' },
  { key: 'workshop', title: '班组管理', desc: '班次、异常、返工派岗' },
  { key: 'mine', title: '我的', desc: '今日核对、假勤、工具与消息' },
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
    case 'station':
      return (
        matchAny(permissions, ['生产管理', '工序流水', '过站记录', '扫码报工', 'production', '报工', '领料']) ||
        roles.includes('piece') ||
        roles.includes('fixed') ||
        roles.includes('计件工') ||
        roles.includes('固定工') ||
        roles.includes('foreman') ||
        roles.includes('车间主任')
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
    case 'workshop':
      return (
        matchAny(permissions, ['生产管理', '派工', '车间', 'production']) ||
        roles.includes('foreman') ||
        roles.includes('车间主任')
      )
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
