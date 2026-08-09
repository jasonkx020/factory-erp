import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useAuthStore } from './auth'
import { ERP_MENUS } from '../constants/menus'
import { systemPermCode } from '../constants/systemPerms'
import type { ModuleMeta } from '../types'
import { MODULES } from '../generated/modules'

export const usePermStore = defineStore('perm', () => {
  const auth = useAuthStore()

  const isFullAdmin = computed(
    () =>
      auth.roles.includes('sys_admin') ||
      auth.roles.includes('系统管理员') ||
      auth.permissions.includes('*:*:*'),
  )

  function hasDomainModulePerm(domain: string, module: string, action?: string) {
    if (isFullAdmin.value) return true
    if (action) return auth.hasPerm(`${domain}:${module}:${action}`)
    const prefix = `${domain}:${module}:`
    return auth.permissions.some((p) => p.startsWith(prefix))
  }

  function canSeeModule(domain: string, module: string) {
    if (isFullAdmin.value) return true
    const fromMenu = auth.menus.some(
      (m) =>
        m.visible &&
        (`${m.domain}|${m.module}` === `${domain}|${module}` || m.module === module),
    )
    if (fromMenu) return true
    // 系统管理：权限码亦可显示（即使未配置菜单裁剪）
    if (domain === '系统管理') {
      return (
        auth.hasPerm(systemPermCode(module, '查看')) ||
        auth.hasPerm(systemPermCode(module, '编辑'))
      )
    }
    return false
  }

  /** 13 域菜单：管理员全量；否则按菜单裁剪 ∪ 权限码 */
  const visibleMenus = computed(() => {
    if (isFullAdmin.value) return ERP_MENUS

    return ERP_MENUS.map((d) => ({
      domain: d.domain,
      modules: d.modules.filter((m) => canSeeModule(d.domain, m)),
    })).filter((d) => d.modules.length > 0)
  })

  function metaFor(domain: string, module: string): ModuleMeta | undefined {
    return MODULES.find((m) => m.domain === domain && m.module === module)
  }

  function isIamModule(module: string) {
    return ['权限分配', '自定义权限', '自定义菜单', '登录控制', '账户冻结', '成本隐藏'].includes(module)
  }

  function isSupplierModule(module: string) {
    return module === '供应商管理'
  }

  function isFarmerInboundModule(module: string) {
    return module === '农户档案' || module === '过磅收货' || module === '农户结算'
  }

  function isOnboardModule(module: string) {
    return module === '入职登记'
  }

  function isEmployeeModule(module: string) {
    return module === '员工档案'
  }

  function isDeptModule(module: string) {
    return module === '部门管理'
  }

  function isHrOpsModule(module: string) {
    return [
      '离职登记', '考勤管理', '班次管理', '绩效管理', '请假管理', '考勤明细',
      '加班补卡统计', '考勤月度统计', '考勤绩效汇总', '外访明细', '备忘录管理', '员工日志',
    ].includes(module)
  }

  function isSystemAdminModule(module: string) {
    return [
      '基础设置', '自定义打印', '表格自定义', '公式设置', '销售设置', '生产设置',
      '物流信息管理', '审批流程设定', '人事调动', '批量改价', '批量核算工资',
      '单据审批', '单据锁定', '单据通知', '单据编辑', '单据删除', '事项提醒',
      '多条件检索', '财审管控', '学堂管理', '知识库', '图纸管理', '文档管理',
      '公告设置', '备忘录',
    ].includes(module)
  }

  function isPayrollModule(module: string) {
    return ['工人信息管理', '工资批量管理', '工序工资', '薪酬核算', '销售提成'].includes(module)
  }

  return {
    isFullAdmin,
    visibleMenus,
    metaFor,
    canSeeModule,
    hasDomainModulePerm,
    isIamModule,
    isSupplierModule,
    isFarmerInboundModule,
    isOnboardModule,
    isEmployeeModule,
    isDeptModule,
    isHrOpsModule,
    isSystemAdminModule,
    isPayrollModule,
  }
})
