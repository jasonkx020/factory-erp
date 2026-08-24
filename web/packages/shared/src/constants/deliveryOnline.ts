/**
 * 对外交付上线范围内的菜单模块（域/模块）。
 * 与 productScope 白名单一致；专用页路径见 adminSpecialRoutes。
 */
import { ADMIN_SPECIAL_MODULE_PATHS } from './adminSpecialRoutes'
import { productScopePairs } from './productScope'

/** 现场业务录入仅 App；Admin 对应模块为查询/配置/补单 */
export const FIELD_INPUT_APP_ONLY: Array<[string, string]> = [
  ['采购管理', '过磅收货'],
  ['库存管理', '仓管待入库'],
]

const ONLINE: Array<[string, string]> = productScopePairs()

const onlineSet = new Set(ONLINE.map(([d, m]) => `${d}/${m}`))

/** 是否在对外交付上线范围内 */
export function isDeliveryOnlineModule(domain: string, module: string): boolean {
  if (onlineSet.has(`${domain}/${module}`)) return true
  return Boolean(ADMIN_SPECIAL_MODULE_PATHS[`${domain}/${module}`])
}

/** 侧栏展示用：未上线模块后缀（产线范围外一般不再展示） */
export const OFFLINE_MENU_BADGE = '未上线'
