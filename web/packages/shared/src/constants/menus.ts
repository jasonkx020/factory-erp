import { CASSAVA_PRODUCT_SCOPE } from './productScope'

/**
 * 管理端侧栏菜单（木薯产线交付范围）。
 * 历史全量菜单已收敛为 CASSAVA_PRODUCT_SCOPE。
 */
export const ERP_MENUS = CASSAVA_PRODUCT_SCOPE

export const DOMAIN_LIST_PATH: Record<string, string> = {
  采购管理: '/purchase/hub/weigh',
  库存管理: '/inventory/hub/balances',
  生产管理: '/production/hub/trace-production',
  产品管理: '/product/hub/products',
  工资管理: '/payroll/wage-rates',
  人事管理: '/hr/employees',
  财务管理: '/finance/hub/cost-accountings',
  统计报表: '/report/hub/production-board',
  系统管理: '/system/settings',
}
