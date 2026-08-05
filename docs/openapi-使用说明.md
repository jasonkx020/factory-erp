# OpenAPI 使用说明（加工厂 ERP）

**唯一规范主文件**：[openapi3.0-加工厂ERP.yaml](./openapi3.0-加工厂ERP.yaml)（协议 **1.1.0**）

**路径全表（实现依据）**：[openapi-路径全表.md](./openapi-路径全表.md)

框架说明：[系统框架设计.md](./系统框架设计.md)

---

## 1. 统一约定

| 项 | 说明 |
|----|------|
| 路径前缀 | `/api/v1` |
| 响应 | `{ "code": 1\|0, "msg": "...", "data": ... }`，`code=1` 成功 |
| 鉴权 | `Authorization: Bearer <access_token>` |
| 白名单 | `/api/v1/health`、`/api/v1/auth/login`、`/api/v1/auth/refresh` |
| 字段 | snake_case；分页 `page_num` / `page_size` |
| 权限码 | `核心功能:功能模块:动作` |
| 分期标注 | `x-erp-phase`：`1` / `2` / `3`；`x-erp-module` 对齐核心功能表模块名 |
| 数据库 | 开发 SQLite，生产 MySQL（与契约无关） |

---

## 2. 路径覆盖（1.1.0）

| 范围 | 说明 |
|------|------|
| 健康检查 / 认证 / 权限中心 | 完整 schema（登录、用户、分组、角色、权限码、菜单、字段策略、登录策略、会话） |
| 一期业务域 | 产品/库存/生产/工资/人事/审批/系统：按功能模块展开 CRUD + 业务动作 |
| 二期 | 客户(CRM)/销售/采购/统计报表 + 相关审批与系统配置：路径已全量 |
| 三期 | 财务/固定资产 + 生产(BOM/MRP/委外等) + 报表财务报表类：路径已全量 |

当前约 **427** 条路径、**690+** 条操作；功能模块与《加工厂ERP系统框架设计文档》第 5 章表 **一一对应**（权限能力落在 IAM，不另立核心功能名）。

二/三期及部分一期扩展接口的 `requestBody` 暂为自由 `object`，实现时按实体收紧 schema。一期早期资源（产品/出入库过账等）保留 v1 富 schema。

---

## 3. 权限中心路径速查

| 方法 | 路径 | 能力 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录 |
| GET | `/api/v1/auth/me` | 当前用户+权限+菜单 |
| GET/POST | `/api/v1/iam/users` | 管理员/用户管理 |
| PUT | `/api/v1/iam/users/{id}/roles` | 用户授权 |
| POST | `/api/v1/iam/users/{id}/freeze` | 账户冻结 |
| CRUD | `/api/v1/iam/admin-groups` | 管理员分组 |
| CRUD | `/api/v1/iam/roles` | 角色管理 |
| PUT | `/api/v1/iam/roles/{id}/permissions` | 角色权限码 |
| PUT | `/api/v1/iam/roles/{id}/warehouse-scope` | 仓范围 |
| PUT | `/api/v1/iam/roles/{id}/process-scope` | 工序范围 |
| GET | `/api/v1/iam/permissions` | 权限码字典 |
| GET/PUT | `/api/v1/iam/menus` | 自定义菜单 |
| GET/PUT | `/api/v1/iam/field-policies` | 字段策略/成本隐藏 |
| GET/PUT | `/api/v1/iam/login-policy` | 登录控制 |
| POST | `/api/v1/iam/sessions/{id}/revoke` | 强制下线 |

完整清单见 [openapi-路径全表.md](./openapi-路径全表.md)。

---

## 4. 域路径前缀

| 域 | 前缀 | 分期 |
|----|------|------|
| 认证 / 权限 | `/api/v1/auth`、`/api/v1/iam` | 1 |
| 产品管理 | `/api/v1/product` | 1 |
| 库存管理 | `/api/v1/inventory` | 1 |
| 生产管理 | `/api/v1/production` | 1（BOM/MRP/委外等为 3） |
| 工资管理 | `/api/v1/payroll` | 1 |
| 人事管理 | `/api/v1/hr` | 1（外访等为 2） |
| 审批管理 | `/api/v1/approval` | 1–2 |
| 系统管理 | `/api/v1/system`（菜单/权限等走 iam） | 1–3 |
| 客户管理 | `/api/v1/crm` | 2 |
| 销售管理 | `/api/v1/sales` | 2 |
| 采购管理 | `/api/v1/purchase` | 2 |
| 统计报表 | `/api/v1/report` | 2–3 |
| 财务管理 | `/api/v1/finance` | 3 |
| 固定资产 | `/api/v1/asset` | 3 |

---

## 5. 业务闭环（接口序）

### 生产计件

`POST /production/tasks` → `POST /production/dispatches` → `POST /production/requisitions` → `POST /production/report-works` → `POST /payroll/sheets`

### 采购入库（二期）

申请/审批 → `POST /purchase/inbounds` → 库存过账 `POST /inventory/stock-txns/{id}/post`

### 销售出库（二期）

询价 → 订单 → 预发货占用 → 发货 → 库存出库 → 财务核销（三期）

### 权限

自定义权限/菜单 → 权限分配（用户角色） → 登录控制/冻结 → 操作日志 → 离职收回

---

## 6. 契约与路由生成

OpenAPI 主文件为真相源，**直接编辑** `docs/openapi3.0-加工厂ERP.yaml`（原 `gen_openapi_paths.py` 整文件重生成已退役）。

改契约后执行：

```bash
go run ./cmd/erp-tools gen-routes         # → internal/apigen/routes_gen.go
go run ./cmd/erp-tools openapi-coverage   # 对照 scripts/gin_routes.json 或运行中 API
go run ./cmd/erp-tools gen-web-meta       # 可选：管理端 modules.ts
```

历史基底文件 `_restore_openapi_v1.yaml` / `_auth_iam_paths.fragment.yaml` 仅作参考归档，日常不必再跑合并脚本。

---

## 7. 本地预览

```bash
npx @redocly/cli preview-docs docs/openapi3.0-加工厂ERP.yaml
npx @redocly/cli lint docs/openapi3.0-加工厂ERP.yaml
```

---

## 8. 开发 SOP

1. 先改本 YAML，再 `go run ./cmd/erp-tools gen-routes`
2. 再改 `internal/{domain}` handler；未实现接口可返回 `NOT_IMPLEMENTED`
3. 本地默认 SQLite：`go run ./cmd/erp-api`（端口 **18080**）
4. 演示账号：`admin` / `admin123`（仅开发种子）
5. 门禁：`go run ./cmd/erp-tools openapi-coverage` 须退出码 0

---

## 9. 常见错误码（msg）

| msg | 含义 |
|-----|------|
| `UNAUTHORIZED` | 未登录或 token 无效 |
| `PERM_DENIED` | 无权限码 |
| `USER_FROZEN` | 账户已冻结 |
| `INVALID_CREDENTIAL` | 用户名或密码错误 |
| `DOC_LOCKED` | 单据锁定不可编辑 |
| `NOT_IMPLEMENTED` | 骨架接口尚未实现 |
