import { resolveProductScope, getActiveProductProfile, type ProductScopeEntry } from './productScope'

/**
 * 管理端侧栏菜单（工厂加工核心 + 按 profile 扩展）。
 * 使用 getErpMenus() 获取当前 profile 下的菜单；ERP_MENUS 为 core 快照兼容旧引用。
 */
export function getErpMenus(): ProductScopeEntry[] {
  return resolveProductScope(getActiveProductProfile())
}

export const ERP_MENUS: ProductScopeEntry[] = resolveProductScope('core')

export const DOMAIN_LIST_PATH: Record<string, string> = {
  采购管理: '/purchase/hub/suppliers',
  库存管理: '/inventory/hub/balances',
  生产管理: '/production/hub/trace-production',
  产品管理: '/product/hub/products',
  销售管理: '/sales/hub/orders',
  工资管理: '/payroll/wage-rates',
  人事管理: '/hr/employees',
  财务管理: '/finance/hub/payables',
  统计报表: '/report/hub/production-board',
  系统管理: '/system/setup',
}
