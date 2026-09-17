# 管理端交付边界（ADMIN_DELIVERY）

对齐员工 App 的 [`mobile/DELIVERY.md`](../mobile/DELIVERY.md) 与架构说明 [`FACTORY_CORE.md`](./FACTORY_CORE.md)。本文约定**可对外上线运营**的交付范围。

产品分层：

| 层 | 说明 |
|----|------|
| **核心（core）** | 过磅收货 → 工序过站 → 仓管确认 → 计件核对；Admin 配置/查询/结算 |
| **扩展（extended）** | 销售出库/客户、完整财务总账（科目/凭证/月结/三表等） |
| **行业包** | 如 `cassava`：种子主数据与报表别名，不改引擎 |

所有主数据须经后台配置；现场事务受 **开厂就绪门禁** 约束（见 FACTORY_CORE.md）。

## 范围内

| 能力 | 说明 |
|------|------|
| 产线核心闭环 | **App：过磅收货 → 工序过站 → 仓管确认 → 计件核对**；Admin：配置/查询/结算（**现场录入仅 App**） |
| 财务管理 | core：成本/资金/流水/农户应付；extended：科目/凭证/发票/核销/预收预付/往来/月结/三表 |
| 销售（extended） | 客户档案、销售订单、销售出库、出厂结算 |
| 系统管理 | IAM、开厂配置、菜单权限、登录策略、审计日志、打印与基础设置 |
| 人事/工资英文路由 | `/hr/*`、`/payroll/*` 专用页（去 `/m/中文/...`） |
| 监控运维 | `/api/v1/health`（含 DB 探活）、`/ready`、`/live`、`/metrics`；Docker/compose 与备份回滚 |

范围内菜单由 [`deliveryOnline.ts`](../web/packages/shared/src/constants/deliveryOnline.ts) / [`productScope.ts`](../web/packages/shared/src/constants/productScope.ts) 维护；侧栏范围外模块标 **未上线**。

## 范围外

- 客户自助 Web（可选后续）
- 未纳入当前 profile 的业务域（侧栏标「未上线」）
- 复杂告警规则引擎 / 多租户 SaaS 隔离

## 已知限制

- 生产环境必须：`seed.demo: false`、强 `jwt.secret` / `ERP_JWT_SECRET`（≥16 且非占位符）、收紧 CORS；`/_debug/routes` 仅 demo 且需 sys_admin
- 已月结期间禁止凭证录入/过账；借贷不平衡不可过账；核销行金额不可超额
- 三表由已过账凭证（缺省时回退流水）生成，需先「生成报表」
- MQTT 异常、DB 探活失败会写入 `[ALERT]` 日志（见 `internal/alert`）

## 运维手册要点

### 健康与指标

```bash
curl -s http://127.0.0.1:18080/api/v1/live
curl -s http://127.0.0.1:18080/api/v1/ready
curl -s http://127.0.0.1:18080/api/v1/health
curl -s http://127.0.0.1:18080/api/v1/metrics
```

### Docker 部署

```bash
set ERP_JWT_SECRET=your-production-random-secret-32chars
docker compose up -d --build
docker compose --profile mqtt up -d --build
```

配置模板：[`configs/erp.prod.yaml.example`](../configs/erp.prod.yaml.example)、[`configs/erp.prod.example.yaml`](../configs/erp.prod.example.yaml)。

### 备份与回滚

```bash
powershell -File scripts/backup.ps1
```

### 发布门禁（执行）

```bash
powershell -File scripts/release_gate.ps1
```

清单：

- [ ] `go test ./internal/biz/ -count=1`
- [ ] 生产配置：`demo:false`、强 JWT、CORS 非 `*`
- [ ] `/ready` / `/metrics` 正常
- [ ] 开厂就绪检查通过后再跑现场闭环

## Admin 模块必要性矩阵（工厂核心）

Admin 定位为**配置 / 查询 / 结算**，现场录入仅 App。

| 判定 | 模块 |
|------|------|
| **必要·配置** | 开厂配置、工序定义、工艺流程、产线班次、公司架构/厂区、过磅流程编排、过磅品种、工序工资、仓库角色 |
| **必要·查询/结算** | 过站记录、计件工资、农户结算、箱码/库存台账、生产看板 |
| **扩展·销售/财务** | 客户、销售出库、凭证、科目、月结、三表（`profile=extended`） |
| **降级·例外** | 例外派岗、灵活派发、Admin 补单（`VITE_FIELD_INPUT_ON_ADMIN`） |

## 验收对照

- 业务 API 无权限码返回 **403 PERM_DENIED**
- 核心/扩展域无空壳对外入口
- `industry_pack=none` 时无强制木薯种子
- 非交付菜单侧栏标注「未上线」
