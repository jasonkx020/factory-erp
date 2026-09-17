#!/usr/bin/env python3
"""Generate customer-facing PDF: system menus & features checklist."""

from __future__ import annotations

import os
from datetime import date
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import cm, mm
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import (
    HRFlowable,
    PageBreak,
    Paragraph,
    SimpleDocTemplate,
    Spacer,
    Table,
    TableStyle,
)

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "docs" / "木薯加工厂ERP-功能菜单核对清单.pdf"
FONT_PATH = Path(r"C:\Windows\Fonts\simhei.ttf")
if not FONT_PATH.exists():
    FONT_PATH = Path(r"C:\Windows\Fonts\msyh.ttc")

FONT = "SimHei"
pdfmetrics.registerFont(TTFont(FONT, str(FONT_PATH)))

PAGE_W, PAGE_H = A4
MARGIN = 2 * cm


def sty():
    base = getSampleStyleSheet()
    return {
        "title": ParagraphStyle(
            "title", fontName=FONT, fontSize=22, leading=28, alignment=TA_CENTER, spaceAfter=12
        ),
        "subtitle": ParagraphStyle(
            "subtitle", fontName=FONT, fontSize=11, leading=16, alignment=TA_CENTER, textColor=colors.HexColor("#444")
        ),
        "h1": ParagraphStyle("h1", fontName=FONT, fontSize=16, leading=22, spaceBefore=14, spaceAfter=8, textColor=colors.HexColor("#0D7A6F")),
        "h2": ParagraphStyle("h2", fontName=FONT, fontSize=13, leading=18, spaceBefore=10, spaceAfter=6, textColor=colors.HexColor("#1a5c55")),
        "body": ParagraphStyle("body", fontName=FONT, fontSize=10, leading=15, spaceAfter=6),
        "small": ParagraphStyle("small", fontName=FONT, fontSize=9, leading=13, textColor=colors.HexColor("#555")),
        "bullet": ParagraphStyle("bullet", fontName=FONT, fontSize=10, leading=15, leftIndent=12, spaceAfter=3),
        "check": ParagraphStyle("check", fontName=FONT, fontSize=10, leading=15, leftIndent=6, spaceAfter=4),
    }


def tbl(data, col_widths=None, header_rows=1):
    t = Table(data, colWidths=col_widths, repeatRows=header_rows)
    style = [
        ("FONT", (0, 0), (-1, -1), FONT, 9),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#c8d4d2")),
        ("BACKGROUND", (0, 0), (-1, header_rows - 1), colors.HexColor("#e8f4f2")),
        ("TEXTCOLOR", (0, 0), (-1, header_rows - 1), colors.HexColor("#0D7A6F")),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 5),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
    ]
    t.setStyle(TableStyle(style))
    return t


def footer(canvas, doc):
    canvas.saveState()
    canvas.setFont(FONT, 8)
    canvas.setFillColor(colors.HexColor("#888"))
    canvas.drawString(MARGIN, 12 * mm, f"木薯加工厂 ERP · 功能菜单核对清单 · 第 {doc.page} 页")
    canvas.drawRightString(PAGE_W - MARGIN, 12 * mm, date.today().isoformat())
    canvas.restoreState()


def build_story(s):
    today = date.today().strftime("%Y年%m月%d日")
    story = []

    # Cover
    story.append(Spacer(1, 4 * cm))
    story.append(Paragraph("木薯加工厂 ERP", s["title"]))
    story.append(Paragraph("系统功能与菜单核对清单", s["title"]))
    story.append(Spacer(1, 0.8 * cm))
    story.append(Paragraph("（合同签署 · 交付验收附件）", s["subtitle"]))
    story.append(Spacer(1, 1.2 * cm))
    story.append(Paragraph(f"编制日期：{today}", s["subtitle"]))
    story.append(Paragraph("版本：木薯产线交付版（CASSAVA_PRODUCT_SCOPE）", s["subtitle"]))
    story.append(Spacer(1, 1.5 * cm))
    story.append(tbl([
        ["项目", "说明"],
        ["适用终端", "管理端 Web Admin + 员工端 Flutter App"],
        ["业务域数量", "9 个"],
        ["管理端功能模块", "58 个"],
        ["App 默认现场模块", "5 个"],
        ["统计报表", "12 项"],
        ["核心闭环", "入厂过磅 → 分板入库 → 溯源生产 → 计件核对 → 结算 → 报表"],
    ], col_widths=[4.5 * cm, 12 * cm]))
    story.append(PageBreak())

    # 1. Overview
    story.append(Paragraph("一、文档说明", s["h1"]))
    story.append(Paragraph(
        "本文档列出木薯加工厂 ERP 产线交付版全部菜单与功能，供甲乙双方合同签署及项目验收核对使用。"
        "所列功能以系统当前交付白名单（CASSAVA_PRODUCT_SCOPE）为准。",
        s["body"],
    ))
    story.append(Paragraph("1.1 系统定位", s["h2"]))
    story.append(tbl([
        ["维度", "说明"],
        ["管理端（Admin）", "主数据配置、业务查询、结算对账、统计报表；非现场录入主通道"],
        ["员工端（App）", "现场过磅、仓管入库、工序领退料、班组管理、个人计件核对"],
        ["权限模型", "IAM 域/模块权限（域:模块:查看/编辑）+ 角色裁剪"],
    ], col_widths=[4 * cm, 12.5 * cm]))
    story.append(Spacer(1, 8))
    story.append(Paragraph("1.2 核心业务闭环", s["h2"]))
    story.append(Paragraph(
        "农户到货登记 → 过磅建单 → 入厂确认/质检 → 出码推仓/分板入库 → 库存过账 → "
        "溯源生产启停 → 工序领料/退库/入库 → 计件汇总 → 农户结算 / 薪酬核算 → 经营报表",
        s["body"],
    ))
    story.append(tbl([
        ["环节", "App（现场）", "Admin（后台）"],
        ["过磅收货", "✓ 主通道", "查询/配置/补单（可选）"],
        ["仓管入库", "✓ 主通道", "待入库列表、库存台账"],
        ["工序过站/领料", "✓ 主通道", "工序流水、在制、扣损查询"],
        ["溯源生产", "✓ 溯源生产台", "溯源批进度、生产看板"],
        ["农户结算", "—", "✓ 结算单、对账报表"],
        ["计件/工资", "✓ 今日核对", "✓ 计件汇总、薪酬核算"],
        ["成本核算", "—", "✓ 成本核算、溯源明细"],
    ], col_widths=[3.2 * cm, 5.5 * cm, 7.8 * cm]))
    story.append(PageBreak())

    # 2. Admin modules
    story.append(Paragraph("二、管理端 Web（Admin）功能清单", s["h1"]))
    story.append(Paragraph("共 9 个业务域、58 个功能模块。", s["body"]))

    admin_sections = [
        ("2.1 采购管理（8 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "农户档案", "/purchase/hub/farmers", "农户主数据、默认单价、溯源前缀"],
            ["2", "过磅收货", "/purchase/hub/weigh", "过磅单查询、流程状态（待入厂/待入库/待结算）"],
            ["3", "过磅流程编排", "/purchase/hub/flow-graphs", "入厂流程节点配置"],
            ["4", "过磅品种", "/purchase/hub/varieties", "品种/等级价维护"],
            ["5", "溯源批号", "/purchase/hub/trace-batches", "溯源码批次管理"],
            ["6", "农户结算", "/purchase/hub/settlements", "原料款结算、已付/待付"],
            ["7", "原料溯源", "/purchase/hub/trace", "溯源链查询"],
            ["8", "来料质检", "/purchase/hub/qcs", "来料质检记录"],
        ]),
        ("2.2 库存管理（9 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "库存查询", "/inventory/hub/balances", "按仓/品/批号查库存"],
            ["2", "仓管待入库", "/inventory/hub/inbound", "待入库过磅单、推仓状态"],
            ["3", "箱码管理", "/inventory/hub/boxes", "板码/箱码台账"],
            ["4", "出入库记录汇总", "/inventory/hub/stock-txns", "库存流水"],
            ["5", "可用量分析", "/inventory/hub/availability", "可发/可用量"],
            ["6", "亏料预警", "/inventory/hub/shortage", "缺料预警"],
            ["7", "过量预警", "/inventory/hub/excess", "超储预警"],
            ["8", "在途量统计", "/inventory/hub/in-transits", "在途库存"],
            ["9", "待用量统计", "/inventory/hub/reservations", "预留/待用量"],
        ]),
        ("2.3 生产管理（10 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "工序定义", "/production/hub/processes", "工序主数据"],
            ["2", "工艺流程", "/production/hub/routings", "工艺路线、工序顺序"],
            ["3", "产线班次", "/production/hub/shifts", "班次授权（控制 App 过站）"],
            ["4", "例外派岗", "/production/hub/dispatches", "灵活/例外派工"],
            ["5", "工序流水", "/production/hub/reports", "过站/报工记录"],
            ["6", "计件工资", "/production/hub/piecework", "计件汇总查询"],
            ["7", "工序在制", "/production/hub/process-wip", "在制品查询"],
            ["8", "溯源生产", "/production/hub/trace-production", "溯源批生产会话、启停结案"],
            ["9", "工序扣损", "/production/hub/process-yield", "工序收率/扣损"],
            ["10", "退库未用完还仓", "/production/hub/process-returns", "退库还仓记录"],
        ]),
        ("2.4 产品管理（3 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "产品档案", "/product/hub/products", "原料/半成品/成品档案"],
            ["2", "产品单位管理", "/product/hub/units", "计量单位"],
            ["3", "生产规格绑定", "/product/hub/specs", "产品与工艺规格绑定"],
        ]),
        ("2.5 工资管理（5 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "工人信息管理", "/payroll/workers", "计件工人档案"],
            ["2", "工序工资", "/payroll/wage-rates", "工序单价/工价"],
            ["3", "工资批量管理", "/payroll/batch", "批量工资处理"],
            ["4", "薪酬核算", "/payroll/sheets", "月工资单"],
            ["5", "员工工作台账", "/payroll/work-records", "员工工作记录"],
        ]),
        ("2.6 人事管理（3 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "员工档案", "/hr/employees", "员工主数据"],
            ["2", "公司架构", "/hr/departments", "部门/组织架构"],
            ["3", "角色管理", "/hr/roles", "角色与权限分配"],
        ]),
        ("2.7 财务管理（2 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "成本核算", "/finance/hub/cost-accountings", "按期间/产品成本归集"],
            ["2", "成本明细溯源表", "/finance/hub/cost-traces", "成本来源追溯"],
        ]),
        ("2.8 统计报表（12 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "生产看板", "/report/hub/production-board", "生产总览"],
            ["2", "生产实况", "/report/hub/live", "实时工序/溯源状态"],
            ["3", "三仓库存概览", "/report/hub/warehouse", "原料/半成品/成品仓"],
            ["4", "日经营快照", "/report/hub/daily", "日经营汇总"],
            ["5", "原料入场日报", "/report/hub/inbound-daily", "入场过磅日报"],
            ["6", "计件日结汇总", "/report/hub/piecework-daily", "计件日结"],
            ["7", "工序扣损收率分析", "/report/hub/yield-analysis", "收率分析"],
            ["8", "收发存明细", "/report/hub/stock-ledger", "库存收发存"],
            ["9", "溯源批进度查询", "/report/hub/trace-progress", "溯源批全流程进度"],
            ["10", "农户结算对账汇总", "/report/hub/farmer-settlement-summary", "农户款对账"],
            ["11", "薪酬核算对账", "/report/hub/payroll-reconcile", "工资单 vs 计件差异"],
            ["12", "成本期间汇总", "/report/hub/cost-period-summary", "期间成本汇总"],
        ]),
        ("2.9 系统管理（6 项）", [
            ["序号", "功能模块", "路径", "主要能力"],
            ["1", "基础设置", "/system/settings", "系统参数、载体码名称等"],
            ["2", "生产设置", "/system/production-settings", "产线相关配置"],
            ["3", "自定义权限", "/iam/permissions", "权限码管理"],
            ["4", "登录控制", "/iam/login-policy", "登录策略"],
            ["5", "批量核算工资", "/system/batch-payroll-jobs", "批量工资任务"],
            ["6", "操作日志", "/automation/logs", "审计/操作日志"],
        ]),
    ]

    cw = [1.2 * cm, 3.2 * cm, 5.5 * cm, 6.6 * cm]
    for title, rows in admin_sections:
        story.append(Paragraph(title, s["h2"]))
        story.append(tbl(rows, col_widths=cw))
        story.append(Spacer(1, 6))

    story.append(Paragraph("2.10 管理端公共能力", s["h2"]))
    story.append(tbl([
        ["能力", "说明"],
        ["登录/账号", "/login、个人账号 /account"],
        ["工作台首页", "/ — 各域模块入口、权限快捷"],
        ["工单中心", "/workflow/tickets（内部流转）"],
        ["工具领用", "/hr/tool-issues"],
        ["健康检查", "/api/v1/health、/ready、/live、/metrics"],
    ], col_widths=[4 * cm, 12.5 * cm]))
    story.append(PageBreak())

    # 3. App
    story.append(Paragraph("三、员工端 App 功能清单", s["h1"]))
    story.append(Paragraph("3.1 五大现场模块（默认交付）", s["h2"]))
    story.append(tbl([
        ["模块", "路由", "功能要点"],
        ["生产（过站）", "/station", "选溯源批 → 扫工牌+板码 → 领料/退库/入库；溯源生产台"],
        ["采购（收货）", "/receiving", "过磅入厂建单、过磅入库、单据列表；出码推仓"],
        ["仓管作业", "/warehouse", "待入库、扫码核对入库、板码、库存、盘点、出入库"],
        ["班组管理", "/workshop", "任务派工、灵活派发、质检/返修/废料、退库还仓"],
        ["我的", "Tab", "今日产量/工钱、打卡、假勤、审批、工资、工牌"],
    ], col_widths=[3 * cm, 3 * cm, 10.5 * cm]))
    story.append(Spacer(1, 8))
    story.append(Paragraph("3.2 按角色底部 Tab 配置", s["h2"]))
    story.append(tbl([
        ["角色", "底部 Tab"],
        ["计件工 / 固定工", "生产 · 我的"],
        ["采购员 / 过磅员", "采购 · 我的"],
        ["质检员", "待办工单 · 履历 · 我的（无采购建单）"],
        ["仓管员", "仓管 · 我的"],
        ["班组长 / 车间主任", "生产 · 班组 · 我的"],
        ["系统管理员", "生产 · 采购 · 仓管 · 班组 · 我的"],
    ], col_widths=[5 * cm, 11.5 * cm]))
    story.append(Spacer(1, 8))
    story.append(Paragraph("3.3 「我的」子功能", s["h2"]))
    story.append(tbl([
        ["功能", "说明"],
        ["今日计件核对", "对接个人计件汇总 API"],
        ["电子工牌", "展示员工 badge"],
        ["打卡", "上下班打卡"],
        ["假勤", "请假/加班申请与状态"],
        ["审批", "待审单据"],
        ["工资", "今日计件、月工资单"],
        ["知识库 / 工具领用 / 人事开户", "按权限可达"],
    ], col_widths=[4.5 * cm, 12 * cm]))
    story.append(Spacer(1, 8))
    story.append(Paragraph("3.4 App 通用能力", s["h2"]))
    for line in [
        "· 登录：无自助注册，账号由人事/管理端创建",
        "· 多角色切换：顶栏切换工作台角色",
        "· 溯源码：生产中统一用下拉列表选取溯源批",
        "· 扫码：支持手输；正式环境可接相机",
        "· 载体码：可配置为「板码」或「箱码」",
    ]:
        story.append(Paragraph(line, s["bullet"]))
    story.append(PageBreak())

    # 4. Boundary
    story.append(Paragraph("四、Admin 与 App 分工（交付边界）", s["h1"]))
    story.append(tbl([
        ["功能", "App", "Admin"],
        ["过磅收货（现场录入）", "✓ 主通道", "查询/配置；补单需环境开关"],
        ["仓管待入库", "✓ 主通道", "查询/监控"],
        ["工序领退料/过站", "✓ 主通道", "流水查询"],
        ["溯源生产启停", "✓ 溯源生产台", "会话查询/看板"],
        ["农户结算", "—", "✓"],
        ["薪酬核算", "今日核对", "✓ 全量核算"],
        ["成本核算", "—", "✓"],
        ["主数据配置", "—", "✓"],
        ["经营报表", "个人核对", "✓"],
    ], col_widths=[4.5 * cm, 4 * cm, 8 * cm]))
    story.append(Spacer(1, 12))
    story.append(Paragraph("五、合同范围外（本期不包含）", s["h1"]))
    story.append(tbl([
        ["类别", "不包含项"],
        ["销售域", "销售订单、客户 CRM、询价报价、销售外勤"],
        ["资产域", "固定资产全模块"],
        ["财务扩展", "完整总账（科目/凭证/发票/月结/三表等）；本期仅成本核算 2 项"],
        ["生产扩展", "MRP、自动 BOM、委外/受托、多单整合、进度跟踪"],
        ["客户侧", "客户自助 Web 门户"],
        ["平台级", "多租户 SaaS、复杂告警规则引擎"],
    ], col_widths=[3.5 * cm, 13 * cm]))
    story.append(PageBreak())

    # 6. Sign-off
    story.append(Paragraph("六、验收核对勾选表", s["h1"]))
    story.append(Paragraph("请甲乙双方在合同附件或验收单中逐项勾选确认：", s["body"]))
    checks = [
        "管理端：采购管理（8 项）",
        "管理端：库存管理（9 项）",
        "管理端：生产管理（10 项）",
        "管理端：产品管理（3 项）",
        "管理端：工资管理（5 项）",
        "管理端：人事管理（3 项）",
        "管理端：财务管理（2 项）",
        "管理端：统计报表（12 项）",
        "管理端：系统管理（6 项）",
        "员工端：过磅收货",
        "员工端：仓管作业",
        "员工端：工序过站/领料",
        "员工端：班组管理",
        "员工端：我的（计件/假勤/工资）",
        "员工端：质检工单",
        "IAM 权限与操作审计日志",
        "健康检查接口（/health、/ready、/metrics）",
        "确认不含：销售/CRM/固定资产/完整总账/MRP/客户门户",
    ]
    for c in checks:
        story.append(Paragraph(f"□  {c}", s["check"]))
    story.append(Spacer(1, 1.5 * cm))
    story.append(HRFlowable(width="100%", thickness=0.5, color=colors.HexColor("#ccc")))
    story.append(Spacer(1, 0.8 * cm))
    story.append(tbl([
        ["甲方（客户）", "乙方（交付方）"],
        ["单位名称：________________________", "单位名称：________________________"],
        ["授权代表：________________________", "授权代表：________________________"],
        ["签字：________________  日期：________", "签字：________________  日期：________"],
    ], col_widths=[8.25 * cm, 8.25 * cm], header_rows=0))
    story.append(Spacer(1, 12))
    story.append(Paragraph(
        "备注：本清单依据系统交付白名单 CASSAVA_PRODUCT_SCOPE 编制，与运行环境菜单一致。"
        "如有增删，以双方书面补充协议为准。",
        s["small"],
    ))
    return story


def main():
    OUT.parent.mkdir(parents=True, exist_ok=True)
    doc = SimpleDocTemplate(
        str(OUT),
        pagesize=A4,
        leftMargin=MARGIN,
        rightMargin=MARGIN,
        topMargin=MARGIN,
        bottomMargin=2 * cm,
        title="木薯加工厂ERP-功能菜单核对清单",
        author="YCWL ERP",
    )
    styles = sty()
    doc.build(build_story(styles), onFirstPage=footer, onLaterPages=footer)
    print(f"OK: {OUT}")
    print(f"SIZE: {OUT.stat().st_size} bytes")


if __name__ == "__main__":
    main()
