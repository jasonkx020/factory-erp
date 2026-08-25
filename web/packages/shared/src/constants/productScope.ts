/**
 * 木薯产线交付范围：菜单 / IAM / deliveryOnline 单一白名单来源。
 * 对齐 docs/ADMIN_DELIVERY.md 与产线平台精简整理计划。
 */
export type ProductScopeEntry = { domain: string; modules: string[] }

/** 产线平台保留的域与模块（不含销售/CRM/固定资产/完整总账） */
export const CASSAVA_PRODUCT_SCOPE: ProductScopeEntry[] = [
  {
    domain: '采购管理',
    modules: [
      '农户档案',
      '过磅收货',
      '过磅流程编排',
      '过磅品种',
      '溯源批号',
      '农户结算',
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
    modules: ['员工档案', '岗位管理', '公司架构', '角色管理'],
  },
  {
    domain: '财务管理',
    modules: ['成本核算', '成本明细溯源表'],
  },
  {
    domain: '统计报表',
    modules: [
      '生产看板',
      '生产实况',
      '三仓库存概览',
      '日经营快照',
      '原料入场日报',
      '计件日结汇总',
      '工序扣损收率分析',
      '收发存明细',
      '溯源批进度查询',
      '农户结算对账汇总',
      '薪酬核算对账',
      '成本期间汇总',
    ],
  },
  {
    domain: '系统管理',
    modules: ['基础设置', '生产设置', '自定义权限', '登录控制', '批量核算工资', '操作日志'],
  },
]

const scopeSet = new Set(
  CASSAVA_PRODUCT_SCOPE.flatMap((d) => d.modules.map((m) => `${d.domain}/${m}`)),
)

export function isProductScopeModule(domain: string, module: string): boolean {
  return scopeSet.has(`${domain}/${module}`)
}

export function productScopePairs(): Array<[string, string]> {
  return CASSAVA_PRODUCT_SCOPE.flatMap((d) => d.modules.map((m) => [d.domain, m] as [string, string]))
}

/** 按产线白名单裁剪完整菜单树 */
export function filterMenusByProductScope(
  menus: ProductScopeEntry[],
): ProductScopeEntry[] {
  const allowed = new Map<string, Set<string>>()
  for (const d of CASSAVA_PRODUCT_SCOPE) {
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
