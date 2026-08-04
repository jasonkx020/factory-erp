import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useAuthStore } from './auth'
import { ERP_MENUS } from '../constants/menus'
import type { ModuleMeta } from '../types'
import { MODULES } from '../generated/modules'

export const usePermStore = defineStore('perm', () => {
  const auth = useAuthStore()

  /** 13 域菜单：若 me.menus 有裁剪则过滤；管理员看全量 */
  const visibleMenus = computed(() => {
    const isAdmin =
      auth.roles.includes('sys_admin') ||
      auth.permissions.includes('*:*:*') ||
      auth.menus.length === 0
    if (isAdmin) return ERP_MENUS

    const allowed = new Set(
      auth.menus.filter((m) => m.visible).map((m) => `${m.domain}|${m.module}`),
    )
    // menus from API may use English domain — also allow by module name alone
    const allowedMods = new Set(auth.menus.filter((m) => m.visible).map((m) => m.module))
    return ERP_MENUS.map((d) => ({
      domain: d.domain,
      modules: d.modules.filter(
        (m) => allowed.has(`${d.domain}|${m}`) || allowedMods.has(m) || allowed.size === 0,
      ),
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

  function isOnboardModule(module: string) {
    return module === '入职登记'
  }

  function isHrOpsModule(module: string) {
    return [
      '离职登记', '考勤管理', '班次管理', '绩效管理', '请假管理', '考勤明细',
      '加班补卡统计', '考勤月度统计', '考勤绩效汇总', '外访明细', '备忘录管理', '员工日志',
    ].includes(module)
  }

  return { visibleMenus, metaFor, isIamModule, isSupplierModule, isOnboardModule, isHrOpsModule }
})
