# 加工厂 ERP

契约先行的多端 ERP：Go/Gin 单二进制 `erp-api` + Vue Web（入口/管理/老板）+ Flutter 员工 App。**数据库仅 PostgreSQL**（正式研发基线）。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/用户使用手册.md](docs/用户使用手册.md) | **业务与现场用户使用手册（推荐先读）** |
| [docs/系统框架设计.md](docs/系统框架设计.md) | 架构与包结构 |
| [docs/木薯粗加工需求澄清与现状差距分析.md](docs/木薯粗加工需求澄清与现状差距分析.md) | 需求澄清与现状差距 |
| [docs/ERP-持续开发约束.md](docs/ERP-持续开发约束.md) | 工程约束与门禁 |
| [docs/openapi3.0-加工厂ERP.yaml](docs/openapi3.0-加工厂ERP.yaml) | OpenAPI 3.0 唯一契约 |
| [docs/openapi-使用说明.md](docs/openapi-使用说明.md) | 协议约定与域前缀 |
| [docs/openapi-路径全表.md](docs/openapi-路径全表.md) | 路径实现清单 |
| [加工厂ERP系统框架设计文档.md](加工厂ERP系统框架设计文档.md) | 功能边界（13 域） |
| [migrations/erp/upgrades/README.md](migrations/erp/upgrades/README.md) | DB 增量升级约定 |
| [docs/PostgreSQL运维手册.md](docs/PostgreSQL运维手册.md) | **运维：安装、初始化、备份升级** |

## 环境要求

- Go（与 [`go.mod`](go.mod) 一致，当前 `go 1.26.x`）
- **PostgreSQL 16+**（开发可用 Docker：`docker compose up -d postgres`）
- Python 3（可选；契约工具已改为 Go：`cmd/erp-tools`）
- Node.js + npm（Web monorepo：portal / admin / boss）
- Flutter（员工现场 App Android/iOS，必选现场端）

## 数据库（PostgreSQL）

| 项 | 说明 |
|----|------|
| 基线 | [`migrations/erp/schema.sql`](migrations/erp/schema.sql)（v1.0.0） |
| 开发种子 | [`migrations/erp/data-dev.sql`](migrations/erp/data-dev.sql) |
| 增量 | [`migrations/erp/upgrades/`](migrations/erp/upgrades/) |
| CLI | `cmd/erp-db`（`baseline` / `upgrade` / `status` / `seed-dev` / `validate` / `create`） |
| 开发 | `configs/erp.dev.yaml` 中 `init_schema: true` 启动时自动齐库 |
| 生产 | `init_schema: false`；运维显式升级后再重启 API |

### Ubuntu 维护脚本

```bash
chmod +x scripts/*.sh

./scripts/maint.sh doctor              # 检查 go / psql / docker / DSN
./scripts/maint.sh up-dev              # docker compose 起 PostgreSQL
./scripts/db-migrate.sh baseline       # 新库建基线
./scripts/db-migrate.sh upgrade --all  # 应用 pending 增量
./scripts/db-migrate.sh status
./scripts/backup.sh                    # pg_dump
./scripts/maint.sh upgrade-prod        # 备份 → status → upgrade --all
```

### Windows 维护脚本

```powershell
.\scripts\db-migrate.ps1 status
.\scripts\db-migrate.ps1 upgrade --all
.\scripts\backup.ps1
```

生产升级流程（无 down migration，回滚靠备份）：

```text
备份 → erp-db status → erp-db upgrade --all → 滚动重启 API
```

## 国内环境安装（Go / Flutter）

国内网络请务必配置模块代理与 Flutter 镜像，否则 `go mod` / `flutter pub` / 引擎包下载极易超时。

### Go 安装与代理（Windows）

1. 打开 [https://go.dev/dl/](https://go.dev/dl/) 或国内镜像（如 [https://golang.google.cn/dl/](https://golang.google.cn/dl/)）下载 Windows 安装包（`.msi`），按向导安装。
2. 新开 **PowerShell**，确认版本：

```powershell
go version
```

3. **安装后立刻改国内模块代理**（推荐七牛 / 官方中国代理）：

```powershell
# 当前用户永久生效
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
# 若公司有私有库再按需设置：
# go env -w GOPRIVATE=git.example.com

go env GOPROXY GOSUMDB
```

4. 拉本仓库依赖、起库并启动 API：

```powershell
cd D:\workplace\source\factory-erp
docker compose up -d postgres
go mod download
go run ./cmd/erp-api
```

可选：若仍走系统错误代理，可临时清空再编：

```powershell
$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; $env:ALL_PROXY=""
go env -w GOPROXY=https://goproxy.cn,direct
```

### Go 安装与代理（Linux / Ubuntu）

1. 下载并解压（版本号按 [go.dev/dl](https://go.dev/dl/) 调整，示例 `1.26.3`）：

```bash
# 官方中国站通常更快
curl -fsSL -o go.tgz https://golang.google.cn/dl/go1.26.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go.tgz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

2. 模块代理与依赖：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
cd /path/to/factory-erp
go mod download
```

3. PostgreSQL 客户端与本地库：

```bash
sudo apt update
sudo apt install -y postgresql-client docker.io docker-compose-v2
chmod +x scripts/*.sh
./scripts/maint.sh up-dev
./scripts/maint.sh doctor
go run ./cmd/erp-api
```

也可使用发行版包管理器（版本可能偏旧，需满足 `go.mod`）：

```bash
# Debian/Ubuntu 示例（确认版本够新再用）
# sudo apt update && sudo apt install golang-go
```

2. **安装后设置国内代理**：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
go env GOPROXY GOSUMDB
```

备选代理：`https://proxy.golang.com.cn,direct` 或 `https://mirrors.aliyun.com/goproxy/,direct`。

3. 启动：

```bash
cd /path/to/ycwl-erp-master
go mod download
go run ./cmd/erp-api
```

### Flutter 安装与收尾配置

员工 App 完整步骤（Windows/Linux、镜像、Android 编译踩坑）见：

**[mobile/README.md](mobile/README.md)** →「Flutter 安装与收尾配置」+「Android 编译踩坑」。

摘要（两台系统通用）：

```bash
# 1) 克隆 Flutter SDK 后把 bin 加入 PATH
# 2) 国内镜像（必配）
export FLUTTER_STORAGE_BASE_URL=https://storage.flutter-io.cn
export PUB_HOSTED_URL=https://pub.flutter-io.cn
# 3) 收尾
flutter doctor
cd mobile && flutter pub get
flutter run --dart-define=API_BASE=http://<电脑局域网IP>:18080/api/v1
```

Windows 持久化环境变量示例见 mobile README。

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

### 开发（多端口 Vite）

```bash
cd web
npm install
npm run dev:portal     # 入口 :5170（含客户自助 /shop）
npm run dev:admin      # 管理后台 :5173  client_type=admin
npm run dev:boss       # 老板驾驶舱 :5177  client_type=boss
```

员工现场作业**无 Web 前端**（原统一员工端已下线）；请使用 Flutter App。客户自助 Web 为 Portal `/shop`（登录 `client_type=customer`，演示账号 `cust01` / `admin123`）。

### 单端口发布（打进 exe + 可外置）

三端源码不合并；`build:dist` 拼到 `web/dist/`，再 embed 进 `erp-api`：

| 路径 | 内容 |
|------|------|
| `/` | 门户 portal |
| `/admin/` | 管理后台 |
| `/front/boss/` | 老板驾驶舱 |
| `/api/v1/*` | API（同端口） |

```powershell
# Windows：构建前端并输出 dist-release/erp-api.exe
powershell -File scripts/build_release.ps1
```

```bash
# Linux/macOS
chmod +x scripts/build_release.sh
./scripts/build_release.sh
```

- **内嵌（默认生产）**：`server.web_root` 留空 → 使用二进制内前端；拷贝 exe + 配置 + `data/` 即可。
- **外置**：配置 `server.web_root: web/dist`（或环境变量 `ERP_WEB_ROOT`）→ 优先读磁盘，改前端后无需重编。开发用 `configs/erp.dev.yaml` 已默认外置 `web/dist`（目录不存在时自动回退内嵌占位页）。

本地仅预览静态：`cd web && npm run build:dist && npm run preview:dist`（:4173，API 仍需另开）。

更细说明见 [docs/WEB_EMBED.md](docs/WEB_EMBED.md)。

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
| `web/` | portal / admin / boss（构建后 embed 或外置） |
| `internal/webui` | 静态站 go:embed + SPA 托管 |
| `mobile/` | Flutter 员工 App（Android/iOS） |
| `migrations/erp/` | PostgreSQL 基线、增量与历史归档 |

约定摘要：统一信封 `{ code, msg, data }`（`code=1` 成功）、Bearer JWT、权限码 `域:模块:动作`；员工与管理员共享 `iam_user` 等表，登录不同系统签发独立 token（claims 含 `client_type` + `jti`，写入 `iam_user_session`）。

## 生产 PostgreSQL

1. 用 `erp-db baseline` / `upgrade --all` 建库（见 [docs/PostgreSQL运维手册.md](docs/PostgreSQL运维手册.md)）  
2. 复制 `configs/erp.prod.example.yaml` 为本地配置并填写 DSN（`init_schema: false`）  
3. 启动：`go run ./cmd/erp-api -config <你的生产配置文件>`  
   或先 `scripts/build_release.*` 打出带内嵌 Web 的单二进制，`web_root` 留空即可只开 `:18080` 访问门户/Admin/Boss。

密钥、DSN、本机绝对路径不要写入仓库；仅提交 example 配置。
