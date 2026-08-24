# 员工 App 交付清单（木薯产线 · 现场可交付）

## 登录与默认壳

- **无自助注册**：账号由人事 App「人事开户」或管理端入职登记创建
- 登录后默认进入 **FactoryShell**（按角色底部 Tab），非通用工单 FAB
- 多角色可 `ChoiceChip` / 顶栏切换；资料中心入口在「我的」内（知识库链接保留路由）

## 模块（木薯产线）

| 模块 | 路由 | 验收要点 |
|------|------|----------|
| 工序过站/领料 | `/station` | 溯源列表选取；工牌+板码领料/退库/入库 |
| 过磅收货 | `/receiving` | 建单→质检→出码推仓 |
| 仓管作业 | `/warehouse` | 待入库、扫码过账、箱码 |
| 班组管理 | `/workshop` | 异常、返工派岗 |
| 我的 | Tab | **首屏今日产量/工钱核对**；打卡、假勤、工具/工单 |

**已移出默认 App**：销售外勤、固定资产、收款协同、客户门户。

## 角色 Tab 裁剪

| 角色 | 底部 Tab |
|------|----------|
| 计件工/固定工 | 过站 · 我的 |
| 过磅/质检 | 收货 · 我的 |
| 仓管 | 仓管 · 我的 |
| 班组长 | 过站 · 班组 · 我的 |
| 管理员 | 过站 · 收货 · 仓管 · 班组 · 我的 |

## Debug 账号（密码 `admin123`）

`u_piece`、`u_fixed`、`u_foreman`、`u_purchase`、`u_qc`、`u_warehouse`、`u_planner`、`u_payroll`、`u_boss`、`admin`

计件工 badge：`EMP-PC`（demo 自动写入）

生产中除「溯源生产台」外，溯源码统一用下拉列表选取（`ActiveTraceDropdown`）。

## 自动化冒烟

```bash
go run ./cmd/erp-api          # :18080
go run ./cmd/mobile_delivery_smoke   # DELIVERY_SMOKE_OK
go run ./cmd/station_pass_smoke      # STATION_PASS_SMOKE_OK
bash scripts/delivery_loop.sh        # DELIVERY_LOOP_OK
```

## 已知边界

- 日常过站/过磅 **仅 App**；Admin 对应模块为查询/配置
- 扫码可手输；正式环境接相机
- 第三方 OAuth / OCR 需配置后开通
