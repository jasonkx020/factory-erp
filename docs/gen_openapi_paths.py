# -*- coding: utf-8 -*-
"""Generate complete OpenAPI paths for ERP (all 13 domains, phase 1-3)."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "docs" / "openapi3.0-加工厂ERP.yaml"
BASE_OPENAPI = ROOT / "docs" / "_restore_openapi_v1.yaml"  # 含完整 auth/iam + 一期富 schema
AUTH_FRAGMENT = ROOT / "docs" / "_auth_iam_paths.fragment.yaml"
OUT_PATHS = ROOT / "docs" / "_generated_paths.yaml"
INDEX_MD = ROOT / "docs" / "openapi-路径全表.md"

OK = '#/components/schemas/OkAnyResponse'
EMPTY = '#/components/schemas/OkEmptyResponse'

# Each entry: (phase, tag, module_cn, base_path, ops)
# ops: list of (method, subpath, summary, extra_desc)
# subpath '' means collection; '{id}' detail; 'action:xxx' => /{id}/xxx or special

def crud(base_summary: str, with_delete: bool = False):
    ops = [
        ("get", "", f"{base_summary}-列表", None),
        ("post", "", f"{base_summary}-新建", None),
        ("get", "{id}", f"{base_summary}-详情", None),
        ("put", "{id}", f"{base_summary}-更新", None),
    ]
    if with_delete:
        ops.append(("delete", "{id}", f"{base_summary}-删除", None))
    return ops


def action(method: str, sub: str, summary: str):
    return (method, sub, summary, None)


MODULES = []

def add(phase, tag, module, path, ops):
    MODULES.append((phase, tag, module, path, ops))

# ---- 产品管理 一期 ----
add(1, "产品管理", "产品档案", "/api/v1/product/products", crud("产品档案", True) + [
    action("post", "{id}/activate", "产品档案-启用"),
    action("post", "{id}/deactivate", "产品档案-停用"),
])
add(1, "产品管理", "产品单位管理", "/api/v1/product/products/{id}/units", [
    ("get", "", "产品单位-列表", None),
    ("put", "", "产品单位-保存", None),
])
add(1, "产品管理", "APP产品排序", "/api/v1/product/app-sorts", [
    ("get", "", "APP产品排序-查询", None),
    ("put", "", "APP产品排序-保存", None),
])
add(1, "产品管理", "生产规格绑定", "/api/v1/product/specs", crud("生产规格绑定", True))

# ---- 库存管理 一期 ----
add(1, "库存管理", "库存查询", "/api/v1/inventory/balances", [
    ("get", "", "库存查询-列表", None),
])
add(1, "库存管理", "可用量分析", "/api/v1/inventory/availability", [
    ("get", "", "可用量分析", None),
])
add(1, "库存管理", "在途量统计", "/api/v1/inventory/in-transits", [
    ("get", "", "在途量统计", None),
])
add(1, "库存管理", "待用量统计", "/api/v1/inventory/reservations", [
    ("get", "", "待用量统计", None),
    ("post", "{id}/release", "待用占用-释放", None),
])
add(1, "库存管理", "亏料预警", "/api/v1/inventory/alert-rules/shortage", [
    ("get", "", "亏料预警规则-列表", None),
    ("put", "", "亏料预警规则-保存", None),
])
add(1, "库存管理", "过量预警", "/api/v1/inventory/alert-rules/excess", [
    ("get", "", "过量预警规则-列表", None),
    ("put", "", "过量预警规则-保存", None),
])
add(1, "库存管理", "出入库记录汇总", "/api/v1/inventory/stock-txns", crud("出入库流水") + [
    action("post", "{id}/post", "出入库-过账"),
    action("post", "{id}/cancel", "出入库-作废"),
])
add(1, "库存管理", "期初入库", "/api/v1/inventory/openings", [
    ("get", "", "期初入库-列表", None),
    ("post", "", "期初入库-新建", None),
    ("post", "{id}/post", "期初入库-过账", None),
])
add(1, "库存管理", "入库质检", "/api/v1/inventory/inbound-qcs", crud("入库质检") + [
    action("post", "{id}/pass", "入库质检-合格"),
    action("post", "{id}/fail", "入库质检-不合格"),
])
add(1, "库存管理", "仓库盘点", "/api/v1/inventory/stocktakes", crud("仓库盘点") + [
    action("post", "{id}/submit", "盘点-提交"),
    action("post", "{id}/post", "盘点-过账盈亏"),
])
add(1, "库存管理", "车间盘点", "/api/v1/inventory/workshop-stocktakes", crud("车间盘点") + [
    action("post", "{id}/submit", "车间盘点-提交"),
    action("post", "{id}/post", "车间盘点-过账"),
])
add(1, "库存管理", "仓库盘点记录", "/api/v1/inventory/stocktake-records", [
    ("get", "", "仓库盘点记录-查询", None),
    ("get", "{id}", "仓库盘点记录-详情", None),
])
add(1, "库存管理", "物料调拨耗用", "/api/v1/inventory/transfers", crud("物料调拨") + [
    action("post", "{id}/post", "调拨-过账"),
])
add(1, "库存管理", "物料调拨耗用", "/api/v1/inventory/consumes", crud("生产耗用") + [
    action("post", "{id}/post", "耗用-过账"),
])
add(1, "库存管理", "商品调价组装拆分", "/api/v1/inventory/price-adjusts", crud("商品调价"))
add(1, "库存管理", "商品调价组装拆分", "/api/v1/inventory/assemble-splits", crud("组装拆分") + [
    action("post", "{id}/post", "组装拆分-过账"),
])
add(1, "库存管理", "销售退皮", "/api/v1/inventory/sales-peel-returns", crud("销售退皮") + [
    action("post", "{id}/post", "销售退皮-过账"),
])
add(1, "库存管理", "物料转应付", "/api/v1/inventory/material-to-payables", crud("物料转应付") + [
    action("post", "{id}/submit", "物料转应付-提交"),
])
add(1, "库存管理", "采购退货", "/api/v1/inventory/purchase-returns", [
    ("get", "", "库存侧采购退货过账记录", None),
])
add(1, "库存管理", "箱码管理", "/api/v1/inventory/box-codes", [
    ("get", "", "箱码-列表/查询", None),
    ("post", "", "箱码-生成", None),
    ("get", "{id}", "箱码-详情", None),
    ("post", "{id}/bind", "箱码-绑定", None),
    ("get", "trace/{code}", "箱码-追溯", None),
])

# ---- 生产管理 ----
add(1, "生产管理", "工序设置", "/api/v1/production/processes", crud("工序", True))
add(1, "生产管理", "工序管理", "/api/v1/production/processes/{id}/status", [
    ("put", "", "工序-启用停用", None),
])
add(1, "生产管理", "工艺流程", "/api/v1/production/routings", crud("工艺流程", True) + [
    action("get", "{id}/steps", "工艺步骤-查询"),
    action("put", "{id}/steps", "工艺步骤-保存"),
])
add(1, "生产管理", "生产任务单", "/api/v1/production/tasks", crud("生产任务单") + [
    action("post", "{id}/close", "生产任务单-关闭"),
    action("get", "{id}/items", "任务商品行-查询"),
])
add(1, "生产管理", "一单多商品", "/api/v1/production/tasks/{id}/items", [
    ("put", "", "一单多商品-保存行", None),
])
add(1, "生产管理", "多单整合管理", "/api/v1/production/task-merges", crud("多单整合") + [
    action("post", "{id}/confirm", "多单整合-确认"),
])
add(1, "生产管理", "生产派工", "/api/v1/production/dispatches", crud("生产派工") + [
    action("post", "{id}/receive", "派工-接收"),
])
add(1, "生产管理", "灵活派发工单", "/api/v1/production/flex-dispatches", [
    ("get", "", "灵活派发-列表", None),
    ("post", "", "灵活派发-新建", None),
    ("get", "{id}", "灵活派发-详情", None),
    ("post", "{id}/reassign", "灵活派发-改派", None),
])
add(1, "生产管理", "扫码报工", "/api/v1/production/report-works", crud("扫码报工") + [
    action("post", "{id}/correct", "报工-纠错"),
])
add(1, "生产管理", "联动式领料", "/api/v1/production/requisitions", crud("联动领料") + [
    action("post", "{id}/post", "领料-过账出库"),
])
add(1, "生产管理", "计件工资", "/api/v1/production/piecework-summaries", [
    ("get", "", "计件产量汇总-查询", None),
    ("post", "recalc", "计件产量-重算", None),
])
add(1, "生产管理", "计件工资", "/api/v1/production/piecework-summaries/mine", [
    ("get", "", "计件日结-我的今日核对", None),
])
add(1, "人事管理", "员工", "/api/v1/hr/employee-imports", [
    ("post", "", "员工-批量导入", None),
])
add(1, "生产管理", "进度跟踪", "/api/v1/production/progress", [
    ("get", "", "进度跟踪-查询", None),
])
add(1, "生产管理", "质检管理", "/api/v1/production/qc-orders", crud("质检管理") + [
    action("post", "{id}/complete", "质检-完成"),
])
add(1, "生产管理", "返修单", "/api/v1/production/reworks", crud("返修单") + [
    action("post", "{id}/close", "返修单-关闭"),
])
add(1, "生产管理", "废料管理", "/api/v1/production/scraps", crud("废料登记"))
add(1, "生产管理", "车间管理", "/api/v1/production/workshops", crud("车间档案", True))
add(1, "生产管理", "车间工作台", "/api/v1/production/workshop-workbench", [
    ("get", "overview", "车间工作台-总览", None),
    ("get", "today-tasks", "车间工作台-今日任务", None),
])
add(1, "生产管理", "图纸分发", "/api/v1/production/drawing-links", [
    ("get", "", "图纸分发-列表", None),
    ("post", "", "图纸分发-挂接", None),
    ("delete", "{id}", "图纸分发-取消", None),
])
add(3, "生产管理", "自动BOM", "/api/v1/production/boms", crud("自动BOM", True) + [
    action("post", "generate", "BOM-自动生成"),
])
add(3, "生产管理", "MRP物料分析", "/api/v1/production/mrp-runs", [
    ("get", "", "MRP运算-列表", None),
    ("post", "", "MRP运算-执行", None),
    ("get", "{id}/results", "MRP结果-查询", None),
])
add(3, "生产管理", "委外加工", "/api/v1/production/outsources", crud("委外加工") + [
    action("post", "{id}/receive", "委外-收回"),
])
add(3, "生产管理", "受托加工生产流程管控", "/api/v1/production/consignments", crud("受托加工") + [
    action("get", "{id}/progress", "受托-进度"),
])
add(3, "生产管理", "成本隐藏", "/api/v1/production/cost-hide-policies", [
    ("get", "", "成本隐藏策略-查询", None),
    ("put", "", "成本隐藏策略-保存", None),
])
# 产线自动化：双扫码 + 流转引擎
add(1, "生产管理", "扫码报工", "/api/v1/production/scan", [
    ("post", "", "双扫报工-提交", None),
])
add(1, "生产管理", "扫码报工", "/api/v1/production/scan/resolve", [
    ("post", "", "双扫报工-预解析", None),
])
add(1, "生产管理", "进度跟踪", "/api/v1/production/flow-events", [
    ("get", "", "流转事件-列表", None),
    ("get", "{id}", "流转事件-详情", None),
    ("post", "{id}/retry", "流转事件-重试", None),
])
add(1, "生产管理", "工艺流程", "/api/v1/production/flow-rules", [
    ("get", "", "流转规则-查询", None),
    ("put", "", "流转规则-保存", None),
])

# ---- 工资管理 一期 ----
add(1, "工资管理", "工人信息管理", "/api/v1/payroll/worker-profiles", crud("工人工资档案"))
add(1, "工资管理", "工序工资", "/api/v1/payroll/wage-rates", crud("工序工资", True))
add(1, "工资管理", "工资批量管理", "/api/v1/payroll/sheets", crud("工资单") + [
    action("post", "batch-generate", "工资批量-生成"),
    action("post", "{id}/adjust", "工资单-调整"),
])
add(1, "工资管理", "薪酬核算", "/api/v1/payroll/calculations", [
    ("get", "", "薪酬核算-列表", None),
    ("post", "", "薪酬核算-执行", None),
    ("get", "{id}", "薪酬核算-结果", None),
])
add(1, "工资管理", "销售提成", "/api/v1/payroll/commission-rules", crud("销售提成规则", True))
add(1, "工资管理", "销售提成", "/api/v1/payroll/commission-calcs", [
    ("get", "", "提成计算-列表", None),
    ("post", "", "提成计算-执行", None),
])

# ---- 人事（权限分配走 iam）----
add(1, "人事管理", "入职登记", "/api/v1/hr/onboards", crud("入职登记") + [
    action("post", "{id}/confirm", "入职-确认赋权"),
    action("post", "{id}/cancel", "入职-取消草稿"),
])
add(1, "人事管理", "离职登记", "/api/v1/hr/offboards", crud("离职登记") + [
    action("post", "{id}/confirm", "离职-确认收回权限"),
])
add(1, "人事管理", "员工", "/api/v1/hr/employees", crud("员工档案", True) + [
    action("put", "{id}/badge", "员工-绑定工牌"),
    action("post", "{id}/open-account", "员工-开户赋权"),
])
add(1, "人事管理", "权限分配", "/api/v1/iam/users/{id}/bind-employee", [
    ("put", "", "用户-绑定员工", None),
    ("delete", "", "用户-解绑员工", None),
])
add(1, "人事管理", "权限分配", "/api/v1/iam/users/{id}/data-scope", [
    ("get", "", "用户数据范围-查询", None),
])
add(1, "人事管理", "权限分配", "/api/v1/iam/hr-perm-overview", [
    ("get", "", "人事权限工作台-总览", None),
])
add(1, "人事管理", "班次管理", "/api/v1/hr/shifts", crud("班次", True))
add(1, "人事管理", "考勤管理", "/api/v1/hr/attendance/rules", crud("考勤规则", True))
add(1, "人事管理", "考勤明细", "/api/v1/hr/attendance/records", [
    ("get", "", "考勤明细-查询", None),
    ("post", "punch", "考勤-打卡", None),
    ("post", "patch", "考勤-补卡申请", None),
])
add(1, "人事管理", "请假管理", "/api/v1/hr/leave-requests", crud("请假单") + [
    action("post", "{id}/cancel", "请假-撤销"),
])
add(1, "人事管理", "加班补卡统计", "/api/v1/hr/overtime-patches", [
    ("get", "", "加班补卡-列表", None),
    ("post", "", "加班补卡-申请", None),
    ("get", "stats", "加班补卡-统计", None),
])
add(1, "人事管理", "考勤月度统计", "/api/v1/hr/attendance/month-stats", [
    ("get", "", "考勤月度统计", None),
    ("post", "recalc", "考勤月度-重算", None),
])
add(1, "人事管理", "绩效管理", "/api/v1/hr/performance/schemes", crud("绩效方案", True))
add(1, "人事管理", "绩效管理", "/api/v1/hr/performance/results", [
    ("get", "", "绩效结果-列表", None),
    ("post", "", "绩效结果-录入", None),
])
add(1, "人事管理", "考勤绩效汇总", "/api/v1/hr/attendance-perf-summaries", [
    ("get", "", "考勤绩效汇总", None),
])
add(2, "人事管理", "外访明细", "/api/v1/hr/visits", crud("外访明细"))
add(1, "人事管理", "备忘录管理", "/api/v1/hr/memos", crud("人事备忘录", True))
add(1, "人事管理", "员工日志", "/api/v1/hr/employee-journals", [
    ("get", "", "员工日志-列表", None),
    ("post", "", "员工日志-新建", None),
    ("get", "{id}", "员工日志-详情", None),
])
# 权限分配已在 /api/v1/iam/*

# ---- 审批 ----
add(1, "审批管理", "任务管理", "/api/v1/approval/tasks", [
    ("get", "", "审批任务-待办已办", None),
    ("get", "{id}", "审批任务-详情", None),
    ("post", "{id}/approve", "审批-通过", None),
    ("post", "{id}/reject", "审批-驳回", None),
])
add(1, "审批管理", "单据审核", "/api/v1/approval/doc-reviews", [
    ("get", "", "单据审核-入口列表", None),
    ("post", "{id}/approve", "单据审核-通过", None),
    ("post", "{id}/reject", "单据审核-驳回", None),
])
add(2, "审批管理", "费用财务审批", "/api/v1/approval/expense-finance", [
    ("get", "", "费用财务审批-列表", None),
    ("post", "{id}/approve", "费用财务审批-通过", None),
    ("post", "{id}/reject", "费用财务审批-驳回", None),
])
add(2, "审批管理", "询价财务审批", "/api/v1/approval/inquiry-finance", [
    ("get", "", "询价财务审批-列表", None),
    ("post", "{id}/approve", "询价财务审批-通过", None),
    ("post", "{id}/reject", "询价财务审批-驳回", None),
])
add(2, "审批管理", "询价明细审批", "/api/v1/approval/inquiry-lines", [
    ("get", "", "询价明细审批-列表", None),
    ("post", "{id}/approve", "询价明细审批-通过", None),
    ("post", "{id}/reject", "询价明细审批-驳回", None),
])
add(2, "审批管理", "采购审批", "/api/v1/approval/purchases", [
    ("get", "", "采购审批-列表", None),
    ("post", "{id}/approve", "采购审批-通过", None),
    ("post", "{id}/reject", "采购审批-驳回", None),
])
add(2, "审批管理", "采购计划单审批", "/api/v1/approval/purchase-plans", [
    ("get", "", "采购计划单审批-列表", None),
    ("post", "{id}/approve", "采购计划审批-通过", None),
    ("post", "{id}/reject", "采购计划审批-驳回", None),
])
add(2, "审批管理", "事务申请审批", "/api/v1/approval/affairs", [
    ("get", "", "事务申请审批-列表", None),
    ("post", "{id}/approve", "事务审批-通过", None),
    ("post", "{id}/reject", "事务审批-驳回", None),
])
add(2, "审批管理", "费用申请", "/api/v1/approval/expense-requests", crud("费用申请") + [
    action("post", "{id}/submit", "费用申请-提交审批"),
])
add(1, "审批管理", "考勤审批", "/api/v1/approval/attendance", [
    ("get", "", "考勤审批-列表", None),
    ("post", "{id}/approve", "考勤审批-通过", None),
    ("post", "{id}/reject", "考勤审批-驳回", None),
])

# ---- 系统管理（自定义权限/菜单/登录/冻结 → iam）----
add(1, "系统管理", "基础设置", "/api/v1/system/settings", [
    ("get", "", "基础设置-查询", None),
    ("put", "", "基础设置-保存", None),
])
add(2, "系统管理", "销售设置", "/api/v1/system/sales-settings", [
    ("get", "", "销售设置-查询", None),
    ("put", "", "销售设置-保存", None),
])
add(1, "系统管理", "生产设置", "/api/v1/system/production-settings", [
    ("get", "", "生产设置-查询", None),
    ("put", "", "生产设置-保存", None),
])
add(2, "系统管理", "自定义打印", "/api/v1/system/print-templates", crud("打印模板", True))
add(1, "系统管理", "表格自定义", "/api/v1/system/table-customs", [
    ("get", "", "表格自定义-查询", None),
    ("put", "", "表格自定义-保存", None),
])
add(2, "系统管理", "公式设置", "/api/v1/system/formulas", crud("公式设置", True))
add(2, "系统管理", "物流信息管理", "/api/v1/system/logistics/carriers", crud("物流承运商", True))
add(2, "系统管理", "物流信息管理", "/api/v1/system/logistics/tracks", [
    ("get", "", "物流轨迹-查询", None),
    ("get", "{tracking_no}", "物流轨迹-详情", None),
])
add(2, "系统管理", "审批流程设定", "/api/v1/system/approval-flows", crud("审批流程", True) + [
    action("get", "{id}/nodes", "审批节点-查询"),
    action("put", "{id}/nodes", "审批节点-保存"),
])
add(1, "系统管理", "人事调动", "/api/v1/system/personnel-transfers", crud("人事调动") + [
    action("post", "{id}/confirm", "人事调动-确认"),
])
add(2, "系统管理", "批量改价", "/api/v1/system/batch-price-jobs", [
    ("get", "", "批量改价-任务列表", None),
    ("post", "", "批量改价-创建任务", None),
    ("get", "{id}", "批量改价-任务详情", None),
])
add(1, "系统管理", "批量核算工资", "/api/v1/system/batch-payroll-jobs", [
    ("get", "", "批量核算工资-列表", None),
    ("post", "", "批量核算工资-触发", None),
])
add(1, "系统管理", "单据审批", "/api/v1/system/doc-approve-switches", [
    ("get", "", "单据是否需审-查询", None),
    ("put", "", "单据是否需审-保存", None),
])
add(1, "系统管理", "单据锁定", "/api/v1/system/doc-lock-rules", [
    ("get", "", "单据锁定规则-查询", None),
    ("put", "", "单据锁定规则-保存", None),
])
add(1, "系统管理", "单据通知", "/api/v1/system/notify-rules", [
    ("get", "", "单据通知规则-查询", None),
    ("put", "", "单据通知规则-保存", None),
])
add(1, "系统管理", "单据编辑", "/api/v1/system/doc-edit-rules", [
    ("get", "", "单据编辑策略-查询", None),
    ("put", "", "单据编辑策略-保存", None),
])
add(1, "系统管理", "单据删除", "/api/v1/system/doc-delete-rules", [
    ("get", "", "单据删除策略-查询", None),
    ("put", "", "单据删除策略-保存", None),
])
add(1, "系统管理", "事项提醒", "/api/v1/system/reminders", crud("事项提醒", True))
add(2, "系统管理", "多条件检索", "/api/v1/system/search-configs", [
    ("get", "", "多条件检索配置-查询", None),
    ("put", "", "多条件检索配置-保存", None),
])
add(3, "系统管理", "财审管控", "/api/v1/system/finance-audit-controls", [
    ("get", "", "财审管控-查询", None),
    ("put", "", "财审管控-保存", None),
])
add(3, "系统管理", "学堂管理", "/api/v1/system/courses", crud("学堂内容", True))
add(3, "系统管理", "知识库", "/api/v1/system/knowledge", crud("知识库", True))
add(3, "系统管理", "图纸管理", "/api/v1/system/drawings", crud("图纸库", True))
add(3, "系统管理", "文档管理", "/api/v1/system/documents", crud("文档库", True))
add(1, "系统管理", "操作日志", "/api/v1/system/operation-logs", [
    ("get", "", "操作日志-查询", None),
    ("get", "{id}", "操作日志-详情", None),
    ("get", "trace/{trace_id}", "操作日志-链路追溯", None),
])
add(1, "系统管理", "操作日志", "/api/v1/system/data-repairs", crud("数据修复单") + [
    action("post", "{id}/apply", "数据修复-执行"),
])
add(1, "系统管理", "公告设置", "/api/v1/system/announcements", crud("公告", True) + [
    action("post", "{id}/publish", "公告-发布"),
])
add(1, "系统管理", "备忘录", "/api/v1/system/memos", crud("系统备忘录", True))
# 自定义菜单/权限/登录控制/账户冻结 → iam 已覆盖，加索引说明路径
add(1, "系统管理", "自定义菜单", "/api/v1/iam/menus", [
    ("get", "", "自定义菜单-查询(见权限中心)", None),
    ("put", "", "自定义菜单-保存(见权限中心)", None),
])
add(1, "系统管理", "自定义权限", "/api/v1/iam/permissions", [
    ("get", "", "自定义权限-字典(见权限中心)", None),
])
add(1, "系统管理", "登录控制", "/api/v1/iam/login-policy", [
    ("get", "", "登录控制-查询(见权限中心)", None),
    ("put", "", "登录控制-保存(见权限中心)", None),
])
add(1, "系统管理", "账户冻结", "/api/v1/iam/users/{id}/freeze", [
    ("post", "", "账户冻结(见权限中心)", None),
])

# ---- 客户管理 二期 ----
add(2, "客户管理", "CRM客户管理", "/api/v1/crm/customers", crud("CRM客户", True))
add(2, "客户管理", "客户档案", "/api/v1/crm/customers/{id}/profile", [
    ("get", "", "客户档案-详情", None),
    ("put", "", "客户档案-更新", None),
])
add(2, "客户管理", "商机管理", "/api/v1/crm/opportunities", crud("商机") + [
    action("post", "{id}/convert", "商机-转化"),
])
add(2, "客户管理", "客户跟进", "/api/v1/crm/follow-ups", crud("客户跟进"))
add(2, "客户管理", "资源分配", "/api/v1/crm/lead-assigns", [
    ("get", "", "资源分配-记录", None),
    ("post", "", "资源分配-执行", None),
])
add(2, "客户管理", "保护机制", "/api/v1/crm/protect-rules", crud("保护机制规则", True))
add(2, "客户管理", "释放机制", "/api/v1/crm/releases", [
    ("get", "", "释放记录-列表", None),
    ("post", "", "释放-执行/回收公海", None),
])
add(2, "客户管理", "询价管理", "/api/v1/crm/inquiries", [
    ("get", "", "客户侧询价关联-列表", None),
])
add(2, "客户管理", "导入客户", "/api/v1/crm/imports", [
    ("post", "", "导入客户-上传", None),
    ("get", "", "导入批次-列表", None),
    ("get", "{id}", "导入批次-详情", None),
])
add(2, "客户管理", "线索锁定", "/api/v1/crm/customers/{id}/lock", [
    ("post", "", "线索锁定", None),
    ("delete", "", "线索解锁", None),
])
add(2, "客户管理", "线索隐藏", "/api/v1/crm/customers/{id}/hide", [
    ("post", "", "线索隐藏", None),
    ("delete", "", "线索取消隐藏", None),
])
add(2, "客户管理", "任务提醒", "/api/v1/crm/task-reminders", crud("CRM任务提醒", True))

# ---- 销售管理 二期 ----
add(2, "销售管理", "销售订单", "/api/v1/sales/orders", crud("销售订单") + [
    action("post", "{id}/submit", "销售订单-提交"),
    action("post", "{id}/cancel", "销售订单-取消"),
])
add(2, "销售管理", "自助下单", "/api/v1/sales/self-orders", [
    ("post", "", "自助下单-提交", None),
    ("get", "", "自助下单-规则/来源查询", None),
])
add(2, "销售管理", "我的订单", "/api/v1/sales/my-orders", [
    ("get", "", "我的订单-列表", None),
    ("get", "{id}", "我的订单-详情", None),
])
add(2, "销售管理", "询价管理", "/api/v1/sales/inquiries", crud("询价单") + [
    action("post", "{id}/to-order", "询价-转订单"),
])
add(2, "销售管理", "询价审批", "/api/v1/sales/inquiries/{id}/approve", [
    ("post", "", "询价审批-处理", None),
])
add(2, "销售管理", "合同管理", "/api/v1/sales/contracts", crud("合同", True))
add(2, "销售管理", "修改订单", "/api/v1/sales/orders/{id}/changes", [
    ("get", "", "订单变更记录", None),
    ("post", "", "修改订单-提交变更", None),
])
add(2, "销售管理", "发货审批", "/api/v1/sales/deliveries", crud("发货单") + [
    action("post", "{id}/approve", "发货审批-通过"),
    action("post", "{id}/reject", "发货审批-驳回"),
    action("post", "{id}/ship", "发货-确认出库"),
])
add(2, "销售管理", "预发货管理", "/api/v1/sales/pre-shipments", crud("预发货") + [
    action("post", "{id}/reserve", "预发货-占用可用量"),
    action("post", "{id}/release", "预发货-释放占用"),
    action("post", "{id}/confirm", "预发货-确认出库"),
])
add(2, "销售管理", "单据打印", "/api/v1/sales/prints", [
    ("get", "", "销售单据打印-列表", None),
    ("post", "", "销售单据-打印预览/登记", None),
    ("get", "{id}", "销售单据打印-详情", None),
])
add(2, "销售管理", "订单复购", "/api/v1/sales/orders/{id}/rebuy", [
    ("post", "", "订单复购-一键生成", None),
])
add(2, "销售管理", "数据排行榜", "/api/v1/sales/rankings", [
    ("get", "configs", "排行榜配置-查询", None),
    ("put", "configs", "排行榜配置-保存", None),
    ("get", "", "排行榜数据-查询", None),
])
add(2, "销售管理", "销售锁价", "/api/v1/sales/price-locks", crud("销售锁价", True))
add(2, "销售管理", "历史报价查询", "/api/v1/sales/quote-histories", [
    ("get", "", "历史报价-查询", None),
])
add(2, "销售管理", "销售BOM", "/api/v1/sales/sales-boms", crud("销售BOM") + [
    action("get", "{id}/lines", "销售BOM行-查询"),
    action("put", "{id}/lines", "销售BOM行-保存"),
])
add(2, "销售管理", "成本预算", "/api/v1/sales/cost-budgets", [
    ("get", "", "成本预算-列表", None),
    ("post", "", "成本预算-测算", None),
    ("get", "{order_id}", "成本预算-按订单", None),
])
add(2, "销售管理", "报价计算器", "/api/v1/sales/quote-calculator", [
    ("get", "", "报价计算器-页面数据", None),
    ("post", "calc", "报价计算器-试算", None),
    ("post", "apply", "报价计算器-回写询价/订单", None),
])

# ---- 采购 二期 ----
add(2, "采购管理", "供应商管理", "/api/v1/purchase/suppliers", crud("供应商", True) + [
    action("post", "{id}/qualify", "供应商-准入合格"),
    action("post", "{id}/freeze", "供应商-冻结"),
    action("post", "{id}/blacklist", "供应商-拉黑"),
    action("post", "{id}/activate", "供应商-恢复启用"),
    action("get", "{id}/licenses", "供应商证照-查询"),
    action("put", "{id}/licenses", "供应商证照-保存"),
    action("get", "{id}/supply-items", "供应商可供物料-查询"),
    action("put", "{id}/supply-items", "供应商可供物料-保存"),
    action("get", "{id}/performance", "供应商绩效-查询"),
])
add(2, "采购管理", "供应商管理", "/api/v1/purchase/certificate-alerts", [
    ("get", "", "供应商证照到期预警", None),
])
add(2, "采购管理", "农户档案", "/api/v1/purchase/farmers", crud("农户档案", True))
add(2, "采购管理", "过磅收货", "/api/v1/purchase/weigh-tickets", crud("过磅单") + [
    action("post", "{id}/qc", "过磅单-质检"),
    action("post", "{id}/stock-in", "过磅单-入库"),
])
add(2, "采购管理", "农户结算", "/api/v1/purchase/farmer-settlements", [
    ("get", "", "农户结算-列表", None),
    ("post", "", "农户结算-核价", None),
])
add(2, "采购管理", "原料溯源", "/api/v1/purchase/trace-lots/{code}", [
    ("get", "", "追溯码-反查", None),
])
add(2, "采购管理", "采购申请", "/api/v1/purchase/requests", crud("采购申请") + [
    action("post", "{id}/submit", "采购申请-提交审批"),
])
add(2, "采购管理", "采购计划单", "/api/v1/purchase/plans", crud("采购计划单") + [
    action("post", "{id}/submit", "采购计划-提交审批"),
])
add(2, "采购管理", "采购入库", "/api/v1/purchase/inbounds", crud("采购入库") + [
    action("post", "{id}/post", "采购入库-过账"),
])
add(2, "采购管理", "来料质检", "/api/v1/purchase/incoming-qcs", crud("来料质检") + [
    action("post", "{id}/pass", "来料质检-合格"),
    action("post", "{id}/fail", "来料质检-不合格"),
])
add(2, "采购管理", "采购退货", "/api/v1/purchase/returns", crud("采购退货") + [
    action("post", "{id}/post", "采购退货-过账"),
])
add(2, "采购管理", "采购分析", "/api/v1/purchase/analytics", [
    ("get", "volume-price", "采购量价分析", None),
    ("get", "supplier-performance", "供应商绩效汇总", None),
])
add(2, "采购管理", "历史价格查看", "/api/v1/purchase/price-histories", [
    ("get", "", "供应商历史价-查询", None),
])
add(2, "采购管理", "采购任务管理", "/api/v1/purchase/tasks", crud("采购任务") + [
    action("post", "{id}/assign", "采购任务-分派"),
    action("post", "{id}/complete", "采购任务-完成"),
])

# ---- 统计报表 ----
add(2, "统计报表", "企业报表", "/api/v1/report/enterprise", [
    ("get", "", "企业报表-中心列表", None),
    ("get", "{code}", "企业报表-查询", None),
])
add(3, "统计报表", "老板驾驶舱", "/api/v1/report/dashboards/boss", [
    ("get", "", "老板驾驶舱-指标", None),
    ("get", "widgets", "老板驾驶舱-组件配置", None),
    ("put", "widgets", "老板驾驶舱-保存配置", None),
])
add(2, "统计报表", "生产看板", "/api/v1/report/dashboards/production", [
    ("get", "", "生产看板", None),
])
add(2, "统计报表", "生产实况", "/api/v1/report/dashboards/live", [
    ("get", "", "生产实况", None),
])
add(2, "统计报表", "客户询价查询", "/api/v1/report/inquiry-queries", [
    ("get", "", "客户询价查询", None),
])
add(2, "统计报表", "CRM统计", "/api/v1/report/crm-stats", [
    ("get", "", "CRM统计", None),
])
add(2, "统计报表", "日统计报表", "/api/v1/report/daily", [
    ("get", "", "日统计报表", None),
])
add(3, "统计报表", "毛利润统计", "/api/v1/report/gross-profit", [
    ("get", "", "毛利润统计", None),
])
add(2, "统计报表", "质检报表", "/api/v1/report/qc", [
    ("get", "", "质检报表", None),
])
add(3, "统计报表", "账目统计", "/api/v1/report/accounts", [
    ("get", "", "账目统计", None),
])
add(2, "统计报表", "出入库查询", "/api/v1/report/stock-txns", [
    ("get", "", "出入库查询", None),
])
add(2, "统计报表", "收发存明细", "/api/v1/report/stock-ledger", [
    ("get", "", "收发存明细", None),
])
add(2, "统计报表", "跟进记录查询", "/api/v1/report/follow-ups", [
    ("get", "", "跟进记录查询", None),
])
add(2, "统计报表", "销售重量统计", "/api/v1/report/sales-weight", [
    ("get", "", "销售重量统计", None),
])
add(2, "统计报表", "产品销售查询", "/api/v1/report/product-sales", [
    ("get", "", "产品销售查询", None),
])
add(2, "统计报表", "系统物流查询", "/api/v1/report/logistics", [
    ("get", "", "系统物流查询", None),
])
add(3, "统计报表", "成本利润表", "/api/v1/report/cost-profit", [
    ("get", "", "成本利润表", None),
])
add(3, "统计报表", "资产负债表", "/api/v1/report/balance-sheet", [
    ("get", "", "资产负债表", None),
])
add(3, "统计报表", "现金流量表", "/api/v1/report/cash-flow", [
    ("get", "", "现金流量表", None),
])
add(3, "统计报表", "利润表", "/api/v1/report/income-statement", [
    ("get", "", "利润表", None),
])

# ---- 财务 三期 ----
add(3, "财务管理", "账目管理", "/api/v1/finance/account-subjects", crud("会计科目", True))
add(3, "财务管理", "资金管理", "/api/v1/finance/fund-accounts", crud("资金账户", True))
add(3, "财务管理", "资金管理", "/api/v1/finance/fund-transfers", crud("资金调拨") + [
    action("post", "{id}/post", "资金调拨-过账"),
])
add(3, "财务管理", "交易流水账", "/api/v1/finance/ledger-entries", crud("交易流水"))
add(3, "财务管理", "收入支出明细", "/api/v1/finance/income-expenses", [
    ("get", "", "收入支出明细-查询", None),
])
add(3, "财务管理", "订单管理", "/api/v1/finance/orders", [
    ("get", "", "订单财务视角-列表", None),
    ("get", "{id}", "订单财务视角-详情", None),
])
add(3, "财务管理", "小程序管理", "/api/v1/finance/miniprogram-bills", [
    ("get", "", "小程序账单-列表", None),
    ("post", "reconcile", "小程序-对账", None),
])
add(3, "财务管理", "凭证管理", "/api/v1/finance/vouchers", crud("凭证") + [
    action("post", "{id}/approve", "凭证-审核"),
])
add(3, "财务管理", "发票管理", "/api/v1/finance/invoices", crud("发票", True))
add(3, "财务管理", "收款核单", "/api/v1/finance/receipt-writeoffs", crud("收款核单") + [
    action("post", "{id}/confirm", "收款核单-确认"),
])
add(3, "财务管理", "销售认款", "/api/v1/finance/payment-recognitions", crud("销售认款") + [
    action("post", "{id}/confirm", "销售认款-确认"),
])
add(3, "财务管理", "外币结汇", "/api/v1/finance/fx-settlements", crud("外币结汇") + [
    action("post", "{id}/confirm", "结汇-确认"),
])
add(3, "财务管理", "结汇查询", "/api/v1/finance/fx-settlements/query", [
    ("get", "", "结汇查询", None),
])
add(3, "财务管理", "分摊撤销", "/api/v1/finance/cost-allocations", crud("费用分摊") + [
    action("post", "{id}/revoke", "分摊-撤销"),
])
add(3, "财务管理", "收款预警", "/api/v1/finance/receipt-alerts", [
    ("get", "", "收款预警-列表", None),
    ("post", "{id}/handle", "收款预警-处理", None),
])
add(3, "财务管理", "出纳对账", "/api/v1/finance/cashier-reconciles", crud("出纳对账") + [
    action("post", "{id}/confirm", "出纳对账-确认"),
])
add(3, "财务管理", "预收预付管理", "/api/v1/finance/prepay-prepaids", crud("预收预付"))
add(3, "财务管理", "成本核算", "/api/v1/finance/cost-accountings", crud("成本核算") + [
    action("post", "{id}/calc", "成本核算-计算"),
])
add(3, "财务管理", "成本明细溯源表", "/api/v1/finance/cost-traces", [
    ("get", "", "成本明细溯源-查询", None),
    ("get", "{cost_id}", "成本明细溯源-详情", None),
])
add(3, "财务管理", "合同利润", "/api/v1/finance/contract-profits", [
    ("get", "", "合同利润-列表", None),
    ("post", "recalc", "合同利润-重算", None),
])
add(3, "财务管理", "销售退货退单", "/api/v1/finance/sales-return-finances", crud("销售退货退单财务") + [
    action("post", "{id}/confirm", "退货退单-确认"),
])
add(3, "财务管理", "往来调整单", "/api/v1/finance/arap-adjusts", crud("往来调整单") + [
    action("post", "{id}/post", "往来调整-过账"),
])
add(3, "财务管理", "财务审批", "/api/v1/finance/approvals", [
    ("get", "", "财务审批-入口列表", None),
    ("post", "{id}/approve", "财务审批-通过", None),
    ("post", "{id}/reject", "财务审批-驳回", None),
])
add(3, "财务管理", "财务报表", "/api/v1/finance/statements", [
    ("get", "", "财务报表-列表", None),
    ("post", "generate", "财务报表-生成", None),
    ("get", "{code}/export", "财务报表-导出", None),
])
add(3, "财务管理", "月度结转", "/api/v1/finance/month-closes", [
    ("get", "", "月度结转-状态", None),
    ("post", "", "月度结转-执行", None),
    ("post", "{id}/reopen", "月度结转-反结", None),
])

# ---- 固定资产 三期 ----
add(3, "固定资产管理", "固定资产类别", "/api/v1/asset/categories", crud("固定资产类别", True))
add(3, "固定资产管理", "固定资产项目", "/api/v1/asset/fixed-assets", crud("固定资产项目", True))
add(3, "固定资产管理", "固定资产内部转移", "/api/v1/asset/transfers", crud("资产内部转移") + [
    action("post", "{id}/confirm", "资产转移-确认"),
])
add(3, "固定资产管理", "固定资产统计", "/api/v1/asset/stats", [
    ("get", "", "固定资产统计", None),
])


def op_id(method, path):
    p = path.strip("/").replace("/", "_").replace("{", "").replace("}", "").replace("-", "_")
    return f"{method}_{p}"


def render_op(method, full_path, summary, tag, phase, module):
    lines = []
    lines.append(f"    {method}:")
    lines.append(f"      tags: [{tag}]")
    lines.append(f"      summary: {summary}")
    lines.append(f"      operationId: {op_id(method, full_path)}")
    lines.append(f"      description: |")
    lines.append(f"        功能模块：{module}；分期：{phase}")
    lines.append(f"      x-erp-phase: {phase}")
    lines.append(f"      x-erp-module: {module}")
    path_params = re.findall(r"\{([^}]+)\}", full_path)
    # list-like GET：路径不以 } 结尾（非按 id 取详情）时带分页参数
    need_page = method == "get" and not full_path.rstrip("/").endswith("}")
    if path_params or need_page:
        lines.append("      parameters:")
        for name in path_params:
            lines.append(f"        - name: {name}")
            lines.append("          in: path")
            lines.append("          required: true")
            lines.append("          schema:")
            if name == "id" or name.endswith("_id"):
                lines.append("            type: integer")
                lines.append("            format: int64")
            else:
                lines.append("            type: string")
        if need_page:
            lines.append('        - $ref: "#/components/parameters/PageNum"')
            lines.append('        - $ref: "#/components/parameters/PageSize"')
    if method in ("post", "put", "patch"):
        lines.append("      requestBody:")
        lines.append("        required: false")
        lines.append("        content:")
        lines.append("          application/json:")
        lines.append("            schema:")
        lines.append("              type: object")
        lines.append("              additionalProperties: true")
    resp_ref = EMPTY if method == "delete" else OK
    lines.append("      responses:")
    lines.append('        "200":')
    lines.append("          description: OK")
    lines.append("          content:")
    lines.append("            application/json:")
    lines.append("              schema:")
    lines.append(f'                $ref: "{resp_ref}"')
    return "\n".join(lines)


def build_paths():
    # merge ops by full path
    path_map = {}  # path -> list of (method, summary, tag, phase, module)
    index_rows = []
    for phase, tag, module, base, ops in MODULES:
        for method, sub, summary, _ in ops:
            if sub == "":
                full = base
            elif sub.startswith("{") or "/" in sub or sub[0].isalnum():
                # relative join
                if sub.startswith("{"):
                    full = base.rstrip("/") + "/" + sub
                elif "{id}" in base and sub.startswith("{id}/"):
                    full = base
                else:
                    # action on collection or item
                    if sub.startswith("{id}/") or sub.startswith("{id}"):
                        full = base.rstrip("/") + "/" + sub
                    elif re.match(r"^[a-z0-9_\-]+$", sub) and method != "get":
                        # could be collection action or item action - if looks like action word after id pattern in ops
                        # check: post id/action vs post action
                        if any(sub.startswith(p) for p in ("batch-", "recalc", "generate", "calc", "apply", "punch", "patch", "overview", "today-", "volume-", "configs", "widgets", "reconcile")):
                            full = base.rstrip("/") + "/" + sub
                        elif method == "get" and sub in ("stats", "overview", "today-tasks", "volume-price", "configs", "widgets"):
                            full = base.rstrip("/") + "/" + sub
                        else:
                            # item action: /{id}/sub OR collection /sub
                            # Heuristic: if summary contains 列表/查询 without 详情 and sub has no id -> collection action
                            if "列表" in summary or "查询" in summary or "统计" in summary or "配置" in summary or "总览" in summary or "试算" in summary or "打卡" in summary or "生成" in summary or "执行" in summary or "触发" in summary or "上传" in summary or "对账" in summary or "重算" in summary or "测算" in summary:
                                if sub in ("stats", "recalc", "generate", "calc", "apply", "punch", "patch", "batch-generate", "volume-price", "configs", "widgets", "reconcile", "overview", "today-tasks"):
                                    full = base.rstrip("/") + "/" + sub
                                elif "{id}" in sub or sub.startswith("trace/"):
                                    full = base.rstrip("/") + "/" + sub
                                else:
                                    full = base.rstrip("/") + "/{id}/" + sub if method in ("post", "put", "delete") and sub not in ("",) and not sub.startswith("{") else base.rstrip("/") + "/" + sub
                            else:
                                full = base.rstrip("/") + "/{id}/" + sub
                    else:
                        full = base.rstrip("/") + "/" + sub
            else:
                full = base.rstrip("/") + "/" + sub

            # normalize double slashes and fix common patterns from ops like "{id}/post"
            full = re.sub(r"/+", "/", full)
            if not full.startswith("/"):
                full = "/" + full

            key = (full, method)
            path_map.setdefault(full, {})
            # later ops override same method on same path if duplicate - keep first
            if method not in path_map[full]:
                path_map[full][method] = (summary, tag, phase, module)
                index_rows.append((phase, tag, module, method.upper(), full, summary))

    # Better rebuild with clearer join rules
    return path_map, index_rows


def build_paths_v2():
    path_map = {}
    index_rows = []

    def add_op(phase, tag, module, full, method, summary):
        full = re.sub(r"/+", "/", full)
        path_map.setdefault(full, {})
        if method in path_map[full]:
            return
        path_map[full][method] = (summary, tag, phase, module)
        index_rows.append((phase, tag, module, method.upper(), full, summary))

    for phase, tag, module, base, ops in MODULES:
        for method, sub, summary, _ in ops:
            if sub == "":
                full = base
            elif sub.startswith("{id}") or sub.startswith("{") or sub.startswith("trace/"):
                full = f"{base.rstrip('/')}/{sub}"
            else:
                # collection-level action keywords
                coll_actions = {
                    "batch-generate", "recalc", "generate", "calc", "apply", "punch", "patch",
                    "stats", "overview", "today-tasks", "volume-price", "supplier-performance",
                    "configs", "widgets", "reconcile", "query",
                }
                if sub in coll_actions or sub.startswith("batch-"):
                    full = f"{base.rstrip('/')}/{sub}"
                else:
                    # resource action under id
                    full = f"{base.rstrip('/')}/{{id}}/{sub}"
            add_op(phase, tag, module, full, method, summary)

    return path_map, index_rows


def split_path_blocks(paths_body: str):
    """Parse OpenAPI paths body into {path: block_text_without_key_line}."""
    blocks = {}
    cur = None
    cur_lines = []

    def flush():
        nonlocal cur, cur_lines
        if cur:
            blocks[cur] = "\n".join(cur_lines).rstrip()
        cur, cur_lines = None, []

    for line in paths_body.splitlines():
        if line.startswith("  /"):
            flush()
            # "  /api/...:" or "  /api/...: "
            cur = line.strip().rstrip(":")
            cur_lines = []
        elif cur is not None:
            cur_lines.append(line)
    flush()
    return blocks


def methods_in_block(block: str):
    return set(re.findall(r"(?m)^    (get|post|put|patch|delete):", block))


def index_from_block(full: str, block: str, default_phase=1, default_tag="", default_module=""):
    rows = []
    # try x-erp-module / tags / summary per operation — lightweight parse
    ops = re.split(r"(?m)^    (get|post|put|patch|delete):", block)
    # ops[0]=preamble, then pairs (method, body)
    i = 1
    while i + 1 < len(ops):
        method = ops[i]
        body = ops[i + 1]
        i += 2
        tag_m = re.search(r"tags:\s*\[([^\]]+)\]", body)
        tag = tag_m.group(1).strip() if tag_m else default_tag
        sum_m = re.search(r"summary:\s*(.+)", body)
        summary = sum_m.group(1).strip() if sum_m else method
        ph_m = re.search(r"x-erp-phase:\s*(\d+)", body)
        phase = int(ph_m.group(1)) if ph_m else default_phase
        mod_m = re.search(r"x-erp-module:\s*(.+)", body)
        module = mod_m.group(1).strip() if mod_m else default_module
        if full.startswith("/api/v1/health"):
            tag, module, phase = "健康检查", "健康检查", 1
        elif full.startswith("/api/v1/auth"):
            tag, module, phase = "认证", "认证", 1
        elif full.startswith("/api/v1/iam"):
            tag, module, phase = "权限中心", "权限分配", 1
        rows.append((phase, tag, module, method.upper(), full, summary))
    return rows


def main():
    path_map, index_rows = build_paths_v2()

    # dump pure generated stubs for diff/debug
    stub_chunks = ["paths:"]
    for full in sorted(path_map.keys()):
        stub_chunks.append(f"  {full}:")
        for method in ("get", "post", "put", "delete", "patch"):
            if method not in path_map[full]:
                continue
            summary, tag, phase, module = path_map[full][method]
            stub_chunks.append(render_op(method, full, summary, tag, phase, module))
        stub_chunks.append("")
    OUT_PATHS.write_text("\n".join(stub_chunks), encoding="utf-8")

    # Base = restored v1 (rich auth/iam + early phase1 schemas). Fallback: AUTH fragment only.
    if BASE_OPENAPI.exists():
        base_text = BASE_OPENAPI.read_text(encoding="utf-8")
    else:
        raise SystemExit(f"missing base openapi: {BASE_OPENAPI}")

    bm = re.search(r"(?ms)^paths:\n(.*)^components:\n", base_text)
    if not bm:
        raise SystemExit("base openapi: cannot find paths/components")
    base_blocks = split_path_blocks(bm.group(1))
    components = "components:\n" + base_text[bm.end() :]
    head = base_text[: bm.start()]
    # normalize server port for local erp-api
    head = head.replace('default: "8080"', 'default: "18080"')
    head = head.replace('x-erp-protocol-version: "1.0.0"', 'x-erp-protocol-version: "1.1.0"')
    head = head.replace("version: 1.0.0", "version: 1.1.0", 1)
    if "协议 1.1.0" not in head:
        head = head.replace(
            "权限码格式：`核心功能:功能模块:动作`。\n",
            "权限码格式：`核心功能:功能模块:动作`。\n"
            "    协议 1.1.0：13 域功能模块 paths 已全量展开（含二/三期）；实现按 x-erp-phase 分期落地。\n",
            1,
        )

    # Prefer base block when path exists; else generate stub. Also add missing methods onto base.
    final_blocks = dict(base_blocks)
    for full, methods in path_map.items():
        if full in final_blocks:
            existing = methods_in_block(final_blocks[full])
            extras = []
            for method in ("get", "post", "put", "delete", "patch"):
                if method in methods and method not in existing:
                    summary, tag, phase, module = methods[method]
                    extras.append(render_op(method, full, summary, tag, phase, module))
            if extras:
                final_blocks[full] = final_blocks[full].rstrip() + "\n" + "\n".join(extras)
            continue
        parts = []
        for method in ("get", "post", "put", "delete", "patch"):
            if method not in methods:
                continue
            summary, tag, phase, module = methods[method]
            parts.append(render_op(method, full, summary, tag, phase, module))
        final_blocks[full] = "\n".join(parts)

    # Ensure auth fragment paths always present (idempotent)
    if AUTH_FRAGMENT.exists():
        frag_blocks = split_path_blocks(AUTH_FRAGMENT.read_text(encoding="utf-8") + "\n")
        for k, v in frag_blocks.items():
            final_blocks[k] = v  # fragment wins for auth/iam/health

    path_chunks = ["paths:"]
    for full in sorted(final_blocks.keys()):
        path_chunks.append(f"  {full}:")
        body = final_blocks[full]
        if body and not body.startswith("\n"):
            path_chunks.append(body if body.startswith("    ") else body)
        else:
            path_chunks.append(body)
        path_chunks.append("")
    new_paths = "\n".join(path_chunks).replace("paths:\n\n  /", "paths:\n  /")

    OPENAPI.write_text(head + new_paths + "\n" + components, encoding="utf-8")

    # Full index = generated module rows + auth/iam from final blocks
    all_rows = list(index_rows)
    seen = {(r[4], r[3]) for r in all_rows}
    for full, block in final_blocks.items():
        for row in index_from_block(full, block):
            key = (row[4], row[3])
            if key not in seen:
                all_rows.append(row)
                seen.add(key)

    lines = [
        "# OpenAPI 路径全表（实现依据）",
        "",
        f"> 协议 **1.1.0**；共 **{len(all_rows)}** 条操作、**{len(final_blocks)}** 条路径。主文件 [openapi3.0-加工厂ERP.yaml](./openapi3.0-加工厂ERP.yaml)。",
        "",
        "| 分期 | 域 | 功能模块 | 方法 | 路径 | 摘要 |",
        "|------|----|----------|------|------|------|",
    ]
    for phase, tag, module, method, full, summary in sorted(all_rows, key=lambda x: (x[0], x[1], x[4], x[3])):
        lines.append(f"| {phase} | {tag} | {module} | {method} | `{full}` | {summary} |")
    lines.append("")
    lines.append("## 说明")
    lines.append("")
    lines.append("- 权限分配 / 自定义权限 / 自定义菜单 / 登录控制 / 账户冻结：以 `/api/v1/iam/*` 与 `/api/v1/auth/*` 为准（完整 schema）。")
    lines.append("- 一期部分资源保留 v1 富 schema；其余与二/三期为契约骨架（requestBody 自由 object），实现时再收紧。")
    lines.append("- 重新生成：`python docs/gen_openapi_paths.py`（依赖 `_restore_openapi_v1.yaml` + `_auth_iam_paths.fragment.yaml`）。")
    lines.append("- `x-erp-phase` / `x-erp-module` 须与核心功能表一致。")
    INDEX_MD.write_text("\n".join(lines), encoding="utf-8")
    print(f"paths={len(final_blocks)} ops={len(all_rows)} written={OPENAPI}")


if __name__ == "__main__":
    main()

