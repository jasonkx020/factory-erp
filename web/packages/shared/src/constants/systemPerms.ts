/** 系统管理模块清单（与 menus.ts 对齐） */
export const SYSTEM_ADMIN_MODULES: string[] = [
  '基础设置',
  '自定义打印',
  '自定义菜单',
  '自定义权限',
  '表格自定义',
  '公式设置',
  '销售设置',
  '生产设置',
  '物流信息管理',
  '审批流程设定',
  '人事调动',
  '登录控制',
  '批量改价',
  '批量核算工资',
  '单据审批',
  '单据锁定',
  '单据通知',
  '单据编辑',
  '单据删除',
  '事项提醒',
  '多条件检索',
  '账户冻结',
  '财审管控',
  '学堂管理',
  '知识库',
  '图纸管理',
  '文档管理',
  '员工日志',
  '操作日志',
  '数据修复',
  '业务闭环',
  '公告设置',
  '备忘录',
  '工单中心',
]

/** API resourceKey → 系统管理中文模块名 */
export const SYSTEM_RESOURCE_MODULE: Record<string, string> = {
  'system/settings': '基础设置',
  'system/print-templates': '自定义打印',
  'iam/menus': '自定义菜单',
  'iam/permissions': '自定义权限',
  'system/table-customs': '表格自定义',
  'system/formulas': '公式设置',
  'system/sales-settings': '销售设置',
  'system/production-settings': '生产设置',
  'system/logistics/carriers': '物流信息管理',
  'system/approval-flows': '审批流程设定',
  'system/personnel-transfers': '人事调动',
  'iam/login-policy': '登录控制',
  'system/batch-price-jobs': '批量改价',
  'system/batch-payroll-jobs': '批量核算工资',
  'system/doc-approve-switches': '单据审批',
  'system/doc-lock-rules': '单据锁定',
  'system/notify-rules': '单据通知',
  'system/doc-edit-rules': '单据编辑',
  'system/doc-delete-rules': '单据删除',
  'system/reminders': '事项提醒',
  'system/search-configs': '多条件检索',
  'iam/users': '账户冻结',
  'system/finance-audit-controls': '财审管控',
  'system/courses': '学堂管理',
  'system/knowledge': '知识库',
  'system/drawings': '图纸管理',
  'system/documents': '文档管理',
  'system/employee-journals': '员工日志',
  'hr/employee-journals': '员工日志',
  'system/operation-logs': '操作日志',
  'system/data-repairs': '数据修复',
  'system/announcements': '公告设置',
  'system/memos': '备忘录',
  'workflow/ticket-categories': '工单中心',
}

export function systemPermCode(module: string, action: '查看' | '编辑'): string {
  return `系统管理:${module}:${action}`
}

export function isSystemResourceKey(resourceKey: string): boolean {
  if (SYSTEM_RESOURCE_MODULE[resourceKey]) return true
  if (resourceKey.startsWith('system/')) return true
  return ['iam/menus', 'iam/permissions', 'iam/login-policy', 'iam/users'].includes(resourceKey)
}
