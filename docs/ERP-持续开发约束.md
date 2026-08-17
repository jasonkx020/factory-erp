# 加工厂 ERP 持续开发约束

> 从 `d:\workplace\bus\docs\YCWL-持续开发约束.md` 摘取可迁移项，去除 FreeTV 双二进制 / MQTT / 报站相关条款。

| 项目 | 说明 |
|------|------|
| 文档版本 | v1.0 |
| 适用范围 | 本仓库 `erp-api`、OpenAPI、PostgreSQL、demo |
| 契约主文件 | [openapi3.0-加工厂ERP.yaml](./openapi3.0-加工厂ERP.yaml) |

---

## 一、协议（C-PROTO）

| 编号 | 约束 |
|------|------|
| C-PROTO-01 | OpenAPI YAML 为唯一 HTTP 契约；先改 YAML，再改 handler / Markdown |
| C-PROTO-02 | 字段只增不减；破坏性变更升 `info.version` 主版本 |
| C-PROTO-03 | 草案可标 `x-erp-phase`；禁止接口未实现却对外宣称已上线 |
| C-PROTO-04 | 响应信封固定 `{ code, msg, data }`：`code=1` 成功，`code=0` 失败 |

## 二、代码（C-CODE）

| 编号 | 约束 |
|------|------|
| C-CODE-01 | 单二进制 `cmd/erp-api`；业务在 `internal/{domain}`（handler/service/repo） |
| C-CODE-02 | 中间件：Recovery → Logger → CORS → JWT → 路由级 `RequirePerm` |
| C-CODE-03 | DB 表/列 `snake_case`；仅 PostgreSQL；升级走 `migrations/erp` + `erp-db`；方言经 `sqlutil` / rebind |
| C-CODE-04 | 禁止跨域把业务堆在 `system.RegisterSkeleton`；占位须迁回本域 |
| C-CODE-05 | 业务错误优先 HTTP 200 + `code=0`；鉴权失败用 401/403 |

## 三、门禁（C-GATE）

提交前须满足：

- [ ] OpenAPI 变更已 lint（可选 Redocly）
- [ ] `go run ./cmd/erp-tools openapi-coverage` 退出码 0（路径全覆盖且无 `NOT_IMPLEMENTED`）
- [ ] `go build ./...` 通过
- [ ] 冒烟：`scripts/smoke_api.ps1`（health + login + 抽样写读）

## 四、明确不做

- MQTT / 设备双平面 / tenant-platform 隔离
- oapi-codegen 全量生成强制
- 强制本机安装非 PostgreSQL 数据库（开发可用 Docker Postgres）
