/** 左侧二/三级菜单分组：title=二级，modules=三级；未列入分组的模块归入「其他」 */
import type { ProductScopeEntry } from './productScope'

export type MenuGroup = { title: string; modules: string[] }

export const ERP_MENU_GROUPS: Record<string, MenuGroup[]> = {
  采购管理: [
    { title: '农户与入场', modules: ['农户档案', '过磅收货', '过磅流程编排', '过磅品种', '溯源批号', '原料溯源', '来料质检'] },
    { title: '结算', modules: ['农户结算'] },
  ],
  库存管理: [
    {
      title: '库存与判断',
      modules: ['库存查询', '仓管待入库', '箱码管理', '可用量分析', '在途量统计', '待用量统计', '亏料预警', '过量预警'],
    },
    { title: '流水', modules: ['出入库记录汇总'] },
  ],
  生产管理: [
    { title: '工艺与规则', modules: ['工序定义', '工艺流程', '产线班次', '例外派岗'] },
    { title: '现场与溯源', modules: ['溯源生产', '工序在制', '工序扣损'] },
    { title: '现场台账', modules: ['工序流水', '计件工资', '退库未用完还仓'] },
  ],
  产品管理: [{ title: '产品主数据', modules: ['产品档案', '产品单位管理', '生产规格绑定'] }],
  财务管理: [
    { title: '结算与资金', modules: ['农户应付', '资金管理', '交易流水账'] },
    { title: '成本', modules: ['成本核算', '成本明细溯源表'] },
  ],
  工资管理: [
    { title: '工价与档案', modules: ['工人信息管理', '工序工资'] },
    { title: '核算发放', modules: ['工资批量管理', '薪酬核算', '员工工作台账'] },
  ],
  人事管理: [{ title: '组织人事', modules: ['员工档案', '岗位管理', '公司架构', '角色管理'] }],
  统计报表: [
    { title: '经营看板', modules: ['生产看板', '生产实况', '三仓库存概览'] },
    { title: '日结对账', modules: ['日经营快照', '原料入场日报', '计件日结汇总'] },
    {
      title: '分析查询',
      modules: [
        '工序扣损收率分析',
        '收发存明细',
        '溯源批进度查询',
        '农户结算对账汇总',
        '薪酬核算对账',
        '成本期间汇总',
      ],
    },
  ],
  系统管理: [
    { title: '基础与权限', modules: ['基础设置', '自定义权限', '登录控制'] },
    { title: '产线运维', modules: ['生产设置', '批量核算工资', '操作日志'] },
  ],
}

/** 按可见模块裁剪分组；未分组模块并入末尾「其他」 */
export function buildSidebarGroups(domain: string, visibleModules: string[]): MenuGroup[] {
  const allowed = new Set(visibleModules)
  const defined = ERP_MENU_GROUPS[domain] || []
  const used = new Set<string>()
  const groups: MenuGroup[] = []
  for (const g of defined) {
    const mods = g.modules.filter((m) => allowed.has(m))
    mods.forEach((m) => used.add(m))
    if (mods.length) groups.push({ title: g.title, modules: mods })
  }
  const rest = visibleModules.filter((m) => !used.has(m))
  if (rest.length) groups.push({ title: groups.length ? '其他' : '功能菜单', modules: rest })
  return groups
}

export type { ProductScopeEntry }
