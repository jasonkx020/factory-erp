# 加工厂 ERP

契约先行的多端 ERP：Go/Gin 单二进制 `erp-api` + Vue Web（入口/管理/老板）+ Flutter 员工 App。开发默认 SQLite，生产可切 MySQL。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/系统框架设计.md](docs/系统框架设计.md) | 架构与包结构 |
| [docs/木薯粗加工需求澄清与现状差距分析.md](docs/木薯粗加工需求澄清与现状差距分析.md) | 需求澄清与现状差距 |
| [docs/ERP-持续开发约束.md](docs/ERP-持续开发约束.md) | 工程约束与门禁 |
| [docs/openapi3.0-加工厂ERP.yaml](docs/openapi3.0-加工厂ERP.yaml) | OpenAPI 3.0 唯一契约 |
| [docs/openapi-使用说明.md](docs/openapi-使用说明.md) | 协议约定与域前缀 |
| [docs/openapi-路径全表.md](docs/openapi-路径全表.md) | 路径实现清单 |
| [加工厂ERP系统框架设计文档.md](加工厂ERP系统框架设计文档.md) | 功能边界（13 域） |

## 环境要求

- Go（与 `go.mod` 一致）
- Python 3（可选；契约工具已改为 Go：`cmd/erp-tools`）
- Node.js + npm（Web monorepo：portal / admin / boss）
- Flutter（员工现场 App Android/iOS，必选现场端）

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

## Web 前端

```bash
cd web
npm install
npm run dev:portal     # 入口
npm run dev:admin      # 管理后台  client_type=admin
npm run dev:boss       # 老板驾驶舱  client_type=boss
```

员工现场作业**无 Web 前端**（原统一员工端已下线）；请使用 Flutter App。客户自助下单亦无 Web；相关销售/客户 API 仍保留在 OpenAPI 与后端。

## Flutter 员工 App

```bash
cd mobile
flutter pub get
flutter run
# 可选：flutter run --dart-define=API_BASE=http://<host>:18080/api/v1
```

登录 `client_type=mobile`；按 IAM 显隐车间 / 工人 / 仓管 / 销售。详见 [mobile/README.md](mobile/README.md)。

## 契约与门禁

改 OpenAPI 后在根目录执行：

```bash
go run ./cmd/erp-tools gen-routes         # OpenAPI → Gin 路由
go run ./cmd/erp-tools openapi-coverage   # 契约操作须 100% 注册
go run ./cmd/erp-tools gen-web-meta       # 管理端 modules.ts（可选）
```

OpenAPI 主文件 [`docs/openapi3.0-加工厂ERP.yaml`](docs/openapi3.0-加工厂ERP.yaml) 为契约真相源；直接编辑 YAML，勿再整文件批量重生成。

冒烟 / 业务 e2e（需 API 已启动）：

```bash
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
| `internal/auth` | 登录 / refresh / me（分端 `client_type` + 会话） |
| `web/` | portal / admin / employee / boss |
| `mobile/` | Flutter 员工 App（Android/iOS） |
| `db/` | SQLite 开发脚本与 MySQL 生产 DDL |

约定摘要：统一信封 `{ code, msg, data }`（`code=1` 成功）、Bearer JWT、权限码 `域:模块:动作`；员工与管理员共享 `iam_user` 等表，登录不同系统签发独立 token（claims 含 `client_type` + `jti`，写入 `iam_user_session`）。

## 生产 MySQL

1. 按 `db/schema`（及说明文档）建库并导入  
2. 复制 `configs/erp.prod.example.yaml` 为本地配置并填写 DSN  
3. 启动：`go run ./cmd/erp-api -config <你的生产配置文件>`

密钥、DSN、本机绝对路径不要写入仓库；仅提交 example 配置。
