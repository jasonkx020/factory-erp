# 管理端交付边界（ADMIN_DELIVERY）

对齐员工 App 的 [`mobile/DELIVERY.md`](../mobile/DELIVERY.md)。本文约定**可对外上线运营**的交付范围。

## 范围内

| 能力 | 说明 |
|------|------|
| 产线试点闭环 | **App：过磅收货 → 工序过站 → 仓管确认 → 计件核对**；Admin：配置/查询/结算（**现场录入仅 App**） |
| 财务管理全量商用 | 科目/资金/流水/凭证（借贷平衡过账）/发票/核销认款/预收预付/往来/成本/月结/三表 |
| 系统管理 | IAM、菜单权限、登录策略、审计日志、打印与基础设置 |
| 人事/工资英文路由 | `/hr/*`、`/payroll/*` 专用页（去 `/m/中文/...`） |
| 监控运维 | `/api/v1/health`（含 DB 探活）、`/ready`、`/live`、`/metrics`；Docker/compose 与备份回滚 |

范围内菜单由 [`deliveryOnline.ts`](../web/packages/shared/src/constants/deliveryOnline.ts) 维护；侧栏范围外模块标 **未上线**。

## 范围外

- 客户自助 Web
- 非试点业务域（除财务外）的全量去占位（侧栏标「未上线」）
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
# 必须设置强 JWT
set ERP_JWT_SECRET=your-production-random-secret-32chars
docker compose up -d --build
# 可选 MQTT：
docker compose --profile mqtt up -d --build
```

配置模板：[`configs/erp.prod.yaml.example`](../configs/erp.prod.yaml.example)、[`configs/erp.prod.example.yaml`](../configs/erp.prod.example.yaml)。

### 备份与回滚

```bash
powershell -File scripts/backup.ps1
```

- **SQLite**：停服后拷贝 `data/erp.db`；回滚即还原文件并重启。
- **MySQL**：`mysqldump erp_factory > backup.sql`；回滚 `mysql erp_factory < backup.sql` 后重启 API。
- 发布回滚：保留上一镜像 tag，`docker compose up -d` 切回旧 tag。

### 发布门禁（执行）

```bash
powershell -File scripts/release_gate.ps1
# 通过后生成 docs/GATE_SIGN_OFF.md，签字归档
```

清单：

- [ ] `go test ./internal/biz/ -count=1`（含鉴权单测）
- [ ] OpenAPI/路由：`scripts/gin_routes.json` 与 openapi 对齐流程
- [ ] `scripts/release_gate.ps1`：登录→试点列表→财务闭环→403
- [ ] 生产配置：`demo:false`、强 JWT、CORS 非 `*`、debug 关闭
- [ ] `/ready` 返回 DB up；`/metrics` 可刮取
- [ ] `go run ./cmd/station_pass_smoke` → `STATION_PASS_SMOKE_OK`
- [ ] `bash scripts/delivery_loop.sh` 或 `powershell -File scripts/delivery_loop.ps1` → `DELIVERY_LOOP_OK`
- [ ] Admin 生产 Hub「过站记录」无默认创建表单（补单需 `VITE_FIELD_INPUT_ON_ADMIN=true`）

## Admin 模块必要性矩阵（木薯试点）

对齐 [木薯工序过站优化方案.md](./木薯工序过站优化方案.md) 第 12–13 章。Admin 定位为**配置 / 查询 / 结算**，现场录入仅 App。

| 判定 | 模块 |
|------|------|
| **必要·配置** | 工序定义、工艺流程、产线班次、生产任务单、车间管理、过磅流程编排、工序工资 |
| **必要·查询/结算** | 过站记录、计件工资、农户结算、箱码/库存台账、生产看板 |
| **降级·例外** | 例外派岗、灵活派发、Admin 补单（`VITE_FIELD_INPUT_ON_ADMIN`） |
| **试点未上线** | 自动BOM、MRP、计件领料、联动领料、委外/受托、多单整合、进度跟踪 |

侧栏分组见 [`menuGroups.ts`](../web/packages/shared/src/constants/menuGroups.ts)：**工艺与规则 → 计划与例外 → 现场台账 → 质量运维 → 未上线**。

产线班次（`pd_shift`）与「人事·考勤班次」不同：前者控制 App 过站授权，后者控制考勤打卡。

## 验收对照

- 业务 API（含财务）无对应权限码返回 **403 PERM_DENIED**
- 试点域与财务无「菜单有、实为 erp_doc 空壳」的对外入口
- 财务主链路可跑通并有审计
- 生产无默认弱密钥
- 非交付菜单侧栏标注「未上线」
