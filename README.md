# 加工厂 ERP

契约先行的多端 ERP：Go/Gin 单二进制 `erp-api` + Vue 多端前端。开发默认 SQLite，生产可切 MySQL。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/系统框架设计.md](docs/系统框架设计.md) | 架构与包结构 |
| [docs/ERP-持续开发约束.md](docs/ERP-持续开发约束.md) | 工程约束与门禁 |
| [docs/openapi3.0-加工厂ERP.yaml](docs/openapi3.0-加工厂ERP.yaml) | OpenAPI 3.0 唯一契约 |
| [docs/openapi-使用说明.md](docs/openapi-使用说明.md) | 协议约定与域前缀 |
| [docs/openapi-路径全表.md](docs/openapi-路径全表.md) | 路径实现清单 |
| [加工厂ERP系统框架设计文档.md](加工厂ERP系统框架设计文档.md) | 功能边界（13 域） |

## 环境要求

- Go（与 `go.mod` 一致）
- Python 3（契约生成 / 覆盖率脚本）
- Node.js + npm（前端 monorepo，可选）

无需本机预装数据库即可开发：默认使用仓库内 SQLite 文件库。

## 快速启动（API）

在仓库根目录：

```bash
go run ./cmd/erp-api
```

默认读取 `configs/erp.dev.yaml`：

- 监听地址：`:18080`（API 前缀 `/api/v1`）
- 数据库：`data/erp_dev.db`（首次自动 schema + seed）

演示账号：`admin` / `admin123`

指定配置：

```bash
go run ./cmd/erp-api -config configs/erp.dev.yaml
```

健康检查：`GET /api/v1/health`

## 前端（可选）

```bash
cd web
npm install
npm run dev:admin    # 管理端；另有 workshop / worker / sales / boss / customer
```

开发时将 API 指到本机 `erp-api`（见各端代理或 `erp_api_base` 约定）。静态效果页也可直接打开 `demo/index.html`（需先启动 API）。

## 契约与门禁

改 OpenAPI 后在根目录执行：

```bash
python docs/gen_openapi_paths.py   # 如有路径生成步骤
python scripts/gen_routes.py       # 生成 Gin 路由
python scripts/openapi_coverage.py # 契约操作须 100% 注册
```

冒烟 / 业务 e2e（需 API 已启动）：

```bash
# 可选：scripts/smoke_api.ps1（Windows）或等价 curl 登录探测
cd web
npm run e2e
npm run e2e:flow
npm run e2e:supplier
npm run e2e:hr-perm
npm run e2e:onboard
```

覆盖率与操作数以脚本输出为准，勿在文档中写死操作条数。

## 架构要点

| 位置 | 职责 |
|------|------|
| `internal/apigen` | OpenAPI 全量路由（生成）与分发 |
| `internal/biz` | 业务规则（IAM / 采购 / 生产 / 人事权限等） |
| `internal/store` | 通用单据落库 |
| `internal/auth` | 登录 / refresh / me |
| `web/` | 多端 Vue 应用与共享包 |
| `db/` | SQLite 开发脚本与 MySQL 生产 DDL |

约定摘要：统一信封 `{ code, msg, data }`（`code=1` 成功）、Bearer JWT、权限码 `域:模块:动作`。

## 生产 MySQL

1. 按 `db/schema`（及说明文档）建库并导入  
2. 复制 `configs/erp.prod.example.yaml` 为本地配置并填写 DSN  
3. 启动：`go run ./cmd/erp-api -config <你的生产配置文件>`

密钥、DSN、本机绝对路径不要写入仓库；仅提交 example 配置。
