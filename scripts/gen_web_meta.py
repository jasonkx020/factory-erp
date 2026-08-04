# -*- coding: utf-8 -*-
"""Generate web/packages/shared/src/generated/modules.ts from OpenAPI path index + menus."""
from __future__ import annotations

import re
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
INDEX = ROOT / "docs" / "openapi-路径全表.md"
OUT = ROOT / "web" / "packages" / "shared" / "src" / "generated" / "modules.ts"

# module -> preferred list path overrides (when multiple resources)
# Must resolve to a Gin GET collection path (no {param}) whenever possible.
PREFERRED = {
    ("产品管理", "产品档案"): "/product/products",
    ("产品管理", "产品单位管理"): "/product/products",
    ("产品管理", "APP产品排序"): "/product/app-sorts",
    ("产品管理", "生产规格绑定"): "/product/specs",
    ("库存管理", "库存查询"): "/inventory/balances",
    ("库存管理", "可用量分析"): "/inventory/availability",
    ("库存管理", "出入库记录汇总"): "/inventory/stock-txns",
    ("库存管理", "待用量统计"): "/inventory/reservations",
    ("库存管理", "在途量统计"): "/inventory/in-transits",
    ("生产管理", "生产任务单"): "/production/tasks",
    ("生产管理", "生产派工"): "/production/dispatches",
    ("生产管理", "扫码报工"): "/production/report-works",
    ("生产管理", "联动式领料"): "/production/requisitions",
    ("生产管理", "工序设置"): "/production/processes",
    ("生产管理", "工序管理"): "/production/processes",
    ("生产管理", "工艺流程"): "/production/routings",
    ("生产管理", "质检管理"): "/production/qc-orders",
    ("生产管理", "返修单"): "/production/reworks",
    ("生产管理", "废料管理"): "/production/scraps",
    ("生产管理", "车间管理"): "/production/workshops",
    ("生产管理", "车间工作台"): "/production/workshop-workbench/overview",
    ("生产管理", "进度跟踪"): "/production/progress",
    ("生产管理", "计件工资"): "/production/piecework-summaries",
    ("生产管理", "灵活派发工单"): "/production/flex-dispatches",
    ("生产管理", "多单整合管理"): "/production/task-merges",
    ("生产管理", "图纸分发"): "/production/drawing-links",
    ("生产管理", "一单多商品"): "/production/tasks",
    ("生产管理", "自动BOM"): "/production/boms",
    ("生产管理", "MRP物料分析"): "/production/mrp-runs",
    ("生产管理", "委外加工"): "/production/outsources",
    ("生产管理", "受托加工生产流程管控"): "/production/consignments",
    ("生产管理", "成本隐藏"): "/production/cost-hide-policies",
    ("工资管理", "工序工资"): "/payroll/wage-rates",
    ("工资管理", "工资批量管理"): "/payroll/sheets",
    ("工资管理", "薪酬核算"): "/payroll/calculations",
    ("工资管理", "工人信息管理"): "/payroll/worker-profiles",
    ("工资管理", "销售提成"): "/payroll/commission-rules",
    ("人事管理", "权限分配"): "/iam/users",
    ("人事管理", "员工"): "/hr/employees",
    ("人事管理", "入职登记"): "/hr/onboards",
    ("人事管理", "离职登记"): "/hr/offboards",
    ("人事管理", "班次管理"): "/hr/shifts",
    ("人事管理", "考勤管理"): "/hr/attendance/rules",
    ("人事管理", "考勤明细"): "/hr/attendance/records",
    ("人事管理", "请假管理"): "/hr/leave-requests",
    ("审批管理", "任务管理"): "/approval/tasks",
    ("客户管理", "CRM客户管理"): "/crm/customers",
    ("客户管理", "客户档案"): "/crm/customers",
    ("客户管理", "线索锁定"): "/crm/customers",
    ("客户管理", "线索隐藏"): "/crm/customers",
    ("销售管理", "销售订单"): "/sales/orders",
    ("销售管理", "询价管理"): "/sales/inquiries",
    ("销售管理", "预发货管理"): "/sales/pre-shipments",
    ("销售管理", "自助下单"): "/sales/self-orders",
    ("销售管理", "我的订单"): "/sales/my-orders",
    ("销售管理", "单据打印"): "/sales/prints",
    ("销售管理", "订单复购"): "/sales/orders",
    ("销售管理", "修改订单"): "/sales/orders",
    ("销售管理", "询价审批"): "/sales/inquiries",
    ("销售管理", "报价计算器"): "/sales/quote-calculator",
    ("采购管理", "供应商管理"): "/purchase/suppliers",
    ("采购管理", "采购申请"): "/purchase/requests",
    ("采购管理", "采购入库"): "/purchase/inbounds",
    ("采购管理", "来料质检"): "/purchase/incoming-qcs",
    ("统计报表", "老板驾驶舱"): "/report/dashboards/boss",
    ("财务管理", "凭证管理"): "/finance/vouchers",
    ("财务管理", "收款核单"): "/finance/receipt-writeoffs",
    ("财务管理", "账目管理"): "/finance/account-subjects",
    ("固定资产管理", "固定资产项目"): "/asset/fixed-assets",
    ("系统管理", "基础设置"): "/system/settings",
    ("系统管理", "操作日志"): "/system/operation-logs",
    ("系统管理", "自定义权限"): "/iam/permissions",
    ("系统管理", "自定义菜单"): "/iam/menus",
    ("系统管理", "登录控制"): "/iam/login-policy",
    ("系统管理", "账户冻结"): "/iam/users",
}


def strip_api(p: str) -> str:
    return p.replace("/api/v1", "") if p.startswith("/api/v1") else p


def main():
    text = INDEX.read_text(encoding="utf-8")
    # | phase | domain | module | METHOD | `path` | summary |
    rows = []
    for line in text.splitlines():
        if not line.startswith("|"):
            continue
        parts = [p.strip() for p in line.strip("|").split("|")]
        if len(parts) < 6 or parts[0] in ("分期", "---") or parts[0].startswith("-"):
            continue
        try:
            phase = int(parts[0])
        except ValueError:
            continue
        domain, module, method, path = parts[1], parts[2], parts[3].upper(), parts[4].strip("`")
        rows.append((phase, domain, module, method, path))

    by_mod: dict[tuple, list] = defaultdict(list)
    for r in rows:
        by_mod[(r[1], r[2])].append(r)

    # also include menus from constants — ensure every menu module has an entry
    menus_src = (ROOT / "web" / "packages" / "shared" / "src" / "constants" / "menus.ts").read_text(encoding="utf-8")
    menu_pairs = []
    cur_domain = None
    for m in re.finditer(r'domain:\s*[\'"]([^\'"]+)[\'"]|\'([^\']+)\'|"([^"]+)"', menus_src):
        if m.group(1):
            cur_domain = m.group(1)
        elif cur_domain and (m.group(2) or m.group(3)):
            mod = m.group(2) or m.group(3)
            if mod not in ("销售管理",) and cur_domain:
                # skip domain names appearing as strings in modules wrongly — modules are Chinese
                if mod == cur_domain:
                    continue
                menu_pairs.append((cur_domain, mod))

    # Parse menus more reliably
    menu_pairs = []
    for block in re.finditer(
        r"domain:\s*'([^']+)',\s*modules:\s*\[(.*?)\]",
        menus_src,
        re.S,
    ):
        domain = block.group(1)
        mods = re.findall(r"'([^']+)'", block.group(2))
        for mod in mods:
            menu_pairs.append((domain, mod))

    out_items = []
    seen = set()
    for domain, module in menu_pairs:
        key = (domain, module)
        if key in seen:
            continue
        seen.add(key)
        ops = by_mod.get(key, [])
        phase = ops[0][0] if ops else 1
        pref = PREFERRED.get(key)

        # Collect real method coverage
        get_collections = []  # no {param}
        post_collections = []
        get_details = []
        put_details = []
        delete_details = []
        actions = []
        for ph, d, m, method, path in ops:
            sp = strip_api(path)
            if method == "GET" and "{" not in sp:
                get_collections.append(sp)
            if method == "POST" and "{" not in sp:
                post_collections.append(sp)
            if method == "GET" and "{id}" in sp:
                get_details.append(sp)
            if method == "PUT" and "{id}" in sp:
                put_details.append(sp)
            if method == "DELETE" and "{id}" in sp:
                delete_details.append(sp)
            if method == "POST" and "/{" in sp:
                act = sp.rstrip("/").split("/")[-1]
                if act not in ("{id}",) and "{" not in act:
                    actions.append(act)

        list_path = ""
        if pref and "{" not in pref:
            # Prefer override only when it is a real GET collection for this module,
            # or it is an intentional parent remap (orders/customers/products/...).
            if pref in get_collections or not get_collections:
                list_path = pref
            else:
                list_path = pref
        if not list_path and get_collections:
            list_path = get_collections[0]
        # Never invent list from POST-only / nested {id} paths
        if list_path and "{" in list_path:
            list_path = ""

        create_path = ""
        if list_path and list_path in post_collections:
            create_path = list_path
        elif (not list_path or list_path in post_collections) and post_collections:
            create_path = post_collections[0]
        # When remapped to a parent collection, do not attach unrelated POSTs
        if list_path and create_path and create_path != list_path and list_path not in post_collections:
            create_path = ""

        detail_path = ""
        if list_path:
            cand = list_path.rstrip("/") + "/{id}"
            if cand in get_details:
                detail_path = cand
        elif get_details:
            detail_path = get_details[0]

        update_path = ""
        if detail_path and detail_path in put_details:
            update_path = detail_path
        elif list_path:
            cand = list_path.rstrip("/") + "/{id}"
            if cand in put_details:
                update_path = cand

        remove_path = ""
        if detail_path and detail_path in delete_details:
            remove_path = detail_path
        elif list_path:
            cand = list_path.rstrip("/") + "/{id}"
            if cand in delete_details:
                remove_path = cand

        read_only = module in (
            "库存查询", "可用量分析", "老板驾驶舱", "生产看板", "生产实况",
            "操作日志", "在途量统计", "待用量统计", "车间工作台", "进度跟踪",
            "历史报价查询", "数据排行榜",
        )
        action_only = (not list_path) and bool(create_path or actions)

        out_items.append(
            {
                "domain": domain,
                "module": module,
                "phase": phase,
                "list": list_path,
                "create": "" if read_only else (create_path or ""),
                "detail": detail_path or "",
                "update": "" if read_only else (update_path or ""),
                "remove": "" if read_only else (remove_path or ""),
                "actions": sorted(set(actions)),
                "readOnly": read_only,
                "actionOnly": action_only,
            }
        )

    lines = [
        "// Code generated by scripts/gen_web_meta.py — DO NOT EDIT",
        "import type { ModuleMeta } from '../types'",
        "",
        "export const MODULES: ModuleMeta[] = [",
    ]
    for it in out_items:
        actions = ", ".join(f"'{a}'" for a in it["actions"])
        lines.append("  {")
        lines.append(f"    domain: '{it['domain']}',")
        lines.append(f"    module: '{it['module']}',")
        lines.append(f"    phase: {it['phase']},")
        lines.append(f"    list: '{it['list']}',")
        if it["create"]:
            lines.append(f"    create: '{it['create']}',")
        if it["detail"]:
            lines.append(f"    detail: '{it['detail']}',")
        if it["update"]:
            lines.append(f"    update: '{it['update']}',")
        if it["remove"]:
            lines.append(f"    remove: '{it['remove']}',")
        lines.append(f"    actions: [{actions}],")
        if it["readOnly"]:
            lines.append("    readOnly: true,")
        if it["actionOnly"]:
            lines.append("    actionOnly: true,")
        lines.append("  },")
    lines.append("]")
    lines.append("")
    lines.append("export function findModule(domain: string, module: string) {")
    lines.append("  return MODULES.find((m) => m.domain === domain && m.module === module)")
    lines.append("}")
    lines.append("")

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text("\n".join(lines), encoding="utf-8")
    print(f"modules={len(out_items)} -> {OUT}")


if __name__ == "__main__":
    main()
