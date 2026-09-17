/**
 * 工厂加工系统交付范围：菜单 / IAM / deliveryOnline 单一白名单来源。
 * 对齐 docs/FACTORY_CORE.md、docs/ADMIN_DELIVERY.md。
 *
 * - FACTORY_CORE_SCOPE：采购过磅闭环 + 生产过站 + 库存 + 产品 + 人事工资 + 基础财务
 * - FACTORY_EXTENDED_SCOPE：销售出库、客户、完整总账等（product.profile=extended）
 * - INDUSTRY_CASSAVA_REPORTS：木薯行业包报表别名（不进引擎）
 */
export type ProductScopeEntry = { domain: string; modules: string[] }

export type ProductProfile = 'core' | 'extended'
export type IndustryPack = 'none' | 'cassava'

/** 核心闭环（所有 profile 默认包含） */
export const FACTORY_CORE_SCOPE: ProductScopeEntry[] = [
  {
    domain: '采购管理',
    modules: [
      '供应商管理',
      '采购记录',
      '采购流程编排',
      '采购品种',
      '溯源批号',
      '供应商结算',
      '原料溯源',
      '来料质检',
    ],
  },
  {
    domain: '库存管理',
    modules: [
      '库存查询',
      '仓管待入库',
      '箱码管理',
      '出入库记录汇总',
      '可用量分析',
      '亏料预警',
      '过量预警',
      '在途量统计',
      '待用量统计',
    ],
  },
  {
    domain: '生产管理',
    modules: [
      '工序定义',
      '工艺流程',
      '产线班次',
      '例外派岗',
      '工序流水',
      '计件工资',
      '工序在制',
      '溯源生产',
      '工序扣损',
      '退库未用完还仓',
    ],
  },
  {
    domain: '产品管理',
    modules: ['产品档案', '产品单位管理', '生产规格绑定'],
  },
  {
    domain: '工资管理',
    modules: ['工人信息管理', '工序工资', '工资批量管理', '薪酬核算', '员工工作台账'],
  },
  {
    domain: '人事管理',
    modules: ['员工档案', '岗位管理', '公司架构', '厂区管理', '角色管理'],
  },
  {
    domain: '财务管理',
    modules: ['成本核算', '成本明细溯源表', '资金管理', '交易流水账', '供应商应付', '在线支付审批'],
  },
  {
    domain: '统计报表',
    modules: [
      '生产看板',
      '生产实况',
      '仓库库存概览',
      '日经营快照',
      '原料入场日报',
      '计件日结汇总',
      '工序扣损收率分析',
      '收发存明细',
      '溯源批进度查询',
      '供应商结算对账汇总',
      '薪酬核算对账',
      '成本期间汇总',
    ],
  },
  {
    domain: '系统管理',
    modules: ['开厂配置', '基础设置', '支付配置', '生产设置', '自定义权限', '登录控制', '批量核算工资', '操作日志'],
  },
]

/** 扩展域（product.profile=extended） */
export const FACTORY_EXTENDED_SCOPE: ProductScopeEntry[] = [
  {
    domain: '销售管理',
    modules: ['客户档案', '销售订单', '销售出库', '出厂结算'],
  },
  {
    domain: '财务管理',
    modules: [
      '账目管理',
      '凭证管理',
      '发票管理',
      '收款核单',
      '预收预付管理',
      '往来调整单',
      '月度结转',
      '财务报表',
      '销售认款',
    ],
  },
]

/** 木薯行业包：报表显示名别名（菜单仍用核心「仓库库存概览」） */
export const INDUSTRY_CASSAVA_REPORT_ALIASES: Record<string, string> = {
  仓库库存概览: '三仓库存概览',
}

/** @deprecated 使用 resolveProductScope；保留兼容旧引用 */
export const CASSAVA_PRODUCT_SCOPE: ProductScopeEntry[] = mergeScopes(
  FACTORY_CORE_SCOPE,
  [],
)

function mergeScopes(...parts: ProductScopeEntry[][]): ProductScopeEntry[] {
  const map = new Map<string, Set<string>>()
  const order: string[] = []
  for (const part of parts) {
    for (const d of part) {
      if (!map.has(d.domain)) {
        map.set(d.domain, new Set())
        order.push(d.domain)
      }
      const set = map.get(d.domain)!
      for (const m of d.modules) set.add(m)
    }
  }
  return order.map((domain) => ({
    domain,
    modules: Array.from(map.get(domain)!),
  }))
}

/** 按部署 profile 解析有效菜单白名单 */
export function resolveProductScope(profile: ProductProfile = 'core'): ProductScopeEntry[] {
  if (profile === 'extended') {
    return mergeScopes(FACTORY_CORE_SCOPE, FACTORY_EXTENDED_SCOPE)
  }
  return FACTORY_CORE_SCOPE.map((d) => ({ domain: d.domain, modules: [...d.modules] }))
}

/** 运行时有效范围（构建期默认 core；Admin 可被 /system/product-profile 覆盖） */
let activeProfile: ProductProfile = 'core'

export function setActiveProductProfile(profile: ProductProfile) {
  activeProfile = profile === 'extended' ? 'extended' : 'core'
}

export function getActiveProductProfile(): ProductProfile {
  return activeProfile
}

function activeScope(): ProductScopeEntry[] {
  return resolveProductScope(activeProfile)
}

const scopeSet = () =>
  new Set(activeScope().flatMap((d) => d.modules.map((m) => `${d.domain}/${m}`)))

export function isProductScopeModule(domain: string, module: string): boolean {
  return scopeSet().has(`${domain}/${module}`)
}

export function productScopePairs(): Array<[string, string]> {
  return activeScope().flatMap((d) => d.modules.map((m) => [d.domain, m] as [string, string]))
}

/** 按产线白名单裁剪完整菜单树 */
export function filterMenusByProductScope(
  menus: ProductScopeEntry[],
): ProductScopeEntry[] {
  const allowed = new Map<string, Set<string>>()
  for (const d of activeScope()) {
    allowed.set(d.domain, new Set(d.modules))
  }
  return menus
    .map((d) => {
      const set = allowed.get(d.domain)
      if (!set) return null
      const modules = d.modules.filter((m) => set.has(m))
      if (!modules.length) return null
      return { domain: d.domain, modules }
    })
    .filter((d): d is ProductScopeEntry => d != null)
}

export function displayModuleTitle(
  module: string,
  industryPack: IndustryPack = 'none',
): string {
  if (industryPack === 'cassava' && INDUSTRY_CASSAVA_REPORT_ALIASES[module]) {
    return INDUSTRY_CASSAVA_REPORT_ALIASES[module]
  }
  return module
}
