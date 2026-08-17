# 加工厂 ERP · PostgreSQL 安装与部署手册（运维）

面向运维人员：如何安装 PostgreSQL、初始化库环境、对接 `erp-api`，以及日常备份升级。  
**数据库仅 PostgreSQL**；结构真源为仓库内 `migrations/erp/`。

| 项 | 约定 |
|----|------|
| 推荐版本 | PostgreSQL **16+**（最低建议 14+） |
| 库名 | `erp_factory` |
| 业务用户 | `erp`（勿用超级用户跑业务） |
| 基线脚本 | `migrations/erp/schema.sql`（v1.0.0） |
| 开发种子 | `migrations/erp/data-dev.sql`（**生产勿用**） |
| 增量目录 | `migrations/erp/upgrades/` |
| 账本表 | `erp_schema_migration` |
| 迁移工具 | `cmd/erp-db`（包装脚本见下） |
| 生产原则 | **API 启动不改库**（`init_schema: false`）；升级靠 CLI |

配套脚本：

| 系统 | 脚本 |
|------|------|
| Ubuntu/Linux | `scripts/maint.sh`、`scripts/db-migrate.sh`、`scripts/backup.sh` |
| Windows | `scripts/db-migrate.ps1`、`scripts/backup.ps1` |

---

## 1. 架构与职责边界

```text
运维职责                          研发/应用职责
─────────────────                 ─────────────────
安装/加固 PostgreSQL              业务 SQL、migrations 内容
创建库与用户                        erp-api 配置 DSN
baseline / upgrade                开发环境 init_schema
备份 / 恢复                       应用发布与滚动重启
监控连接与磁盘
```

**禁止：**

- 生产环境配置 `init_schema: true`（会在启动时自动改库）
- 手工改已应用的 `upgrades/*.sql` 内容（checksum 会失败）
- 用开发种子 `data-dev.sql` 初始化生产库
- 指望 down migration；回滚一律靠备份

---

## 2. 安装 PostgreSQL

### 2.1 Ubuntu 22.04 / 24.04（推荐生产）

```bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib postgresql-client

# 确认版本
psql --version
sudo systemctl enable --now postgresql
sudo systemctl status postgresql --no-pager
```

如需官方 16 源（发行版自带版本偏旧时）：

```bash
# 示例：PostgreSQL APT（按官网当时文档微调）
sudo apt install -y curl ca-certificates
# 按 https://www.postgresql.org/download/linux/ubuntu/ 添加官方源后：
# sudo apt install -y postgresql-16
```

### 2.2 Docker（开发 / 测试，也可作轻量预发）

在仓库根目录：

```bash
docker compose up -d postgres
docker compose ps
# 健康检查通过后再初始化
```

默认（仅适合内网开发）：

- 用户 / 密码：`erp` / `erp`
- 库名：`erp_factory`
- 端口：`5432`
- 数据卷：`erp-pgdata`

生产若用 Docker，必须改强密码、限制端口暴露，并挂载持久卷与定期备份。

### 2.3 Windows（运维本机工具 / 临时环境）

1. 安装 [PostgreSQL Windows 安装包](https://www.postgresql.org/download/windows/)（勾选 Command Line Tools）。
2. 将 `bin` 加入 PATH（含 `psql`、`pg_dump`、`pg_restore`）。
3. 用「SQL Shell (psql)」或服务管理器确认服务已启动。

业务服务器仍建议 Linux。

---

## 3. 创建库、用户与权限（裸机 / 虚拟机）

以下以 Ubuntu 本机 Postgres 为例。

### 3.1 用系统用户进入

```bash
sudo -u postgres psql
```

### 3.2 创建角色与数据库

在 `psql` 中执行（**把密码换成强密码**）：

```sql
-- 业务角色（可登录）
CREATE ROLE erp WITH LOGIN PASSWORD '请替换为强密码' NOSUPERUSER NOCREATEDB NOCREATEROLE;

-- 业务库，属主为 erp
CREATE DATABASE erp_factory OWNER erp
  ENCODING 'UTF8'
  LC_COLLATE 'en_US.UTF-8'
  LC_CTYPE 'en_US.UTF-8'
  TEMPLATE template0;

-- 可选：限制连接数
ALTER ROLE erp CONNECTION LIMIT 50;

\q
```

若 locale 报错，可用：

```sql
CREATE DATABASE erp_factory OWNER erp ENCODING 'UTF8' TEMPLATE template0;
```

### 3.3 授权（属主通常已够用）

```bash
sudo -u postgres psql -d erp_factory -c "GRANT ALL ON SCHEMA public TO erp;"
sudo -u postgres psql -d erp_factory -c "ALTER SCHEMA public OWNER TO erp;"
```

PostgreSQL 15+ 上 `public` schema 默认权限更严，务必保证业务用户对 `public` 有 `CREATE`/`USAGE`。

### 3.4 网络与认证（生产必做）

编辑：

- `postgresql.conf`：`listen_addresses`（按需，勿对公网裸奔）
- `pg_hba.conf`：仅允许应用机 IP，生产建议 `scram-sha-256`

```bash
# 示例路径（随发行版变化）
sudo nano /etc/postgresql/*/main/postgresql.conf
sudo nano /etc/postgresql/*/main/pg_hba.conf
sudo systemctl reload postgresql
```

`pg_hba.conf` 示例（应用机 `10.0.0.20`）：

```text
# TYPE  DATABASE     USER  ADDRESS         METHOD
host    erp_factory  erp   10.0.0.20/32    scram-sha-256
host    erp_factory  erp   127.0.0.1/32    scram-sha-256
local   all          postgres              peer
```

防火墙仅放行可信来源的 `5432`（若库与 API 同机可只监听 `127.0.0.1`）。

### 3.5 连通性自检

```bash
export ERP_DATABASE_DSN='postgres://erp:强密码@127.0.0.1:5432/erp_factory?sslmode=disable'
# 若启用 TLS：sslmode=require 并配置证书

psql "$ERP_DATABASE_DSN" -c 'SELECT version();'
```

---

## 4. 初始化结构（新装）

新装只做一次：**baseline**（执行全量 `schema.sql`）。  
开发种子仅用于演示环境。

### 4.1 准备迁移工具

仓库根目录：

```bash
# Ubuntu
chmod +x scripts/*.sh
./scripts/maint.sh doctor

# 或直接构建
go build -o bin/erp-db ./cmd/erp-db
```

Windows：

```powershell
go build -o bin/erp-db.exe ./cmd/erp-db
```

### 4.2 生产新装（推荐流程）

```bash
cd /path/to/factory-erp
export ERP_DATABASE_DSN='postgres://erp:强密码@127.0.0.1:5432/erp_factory?sslmode=disable'

# 1) 空库建基线
./scripts/db-migrate.sh baseline

# 2) 应用已发布的增量（若有）
./scripts/db-migrate.sh upgrade --all

# 3) 查看账本
./scripts/db-migrate.sh status

# 4) 校验升级文件命名与 checksum（仓库侧）
./scripts/db-migrate.sh validate
```

等价直接调用：

```bash
./bin/erp-db baseline --dsn "$ERP_DATABASE_DSN"
./bin/erp-db upgrade --all --dsn "$ERP_DATABASE_DSN"
./bin/erp-db status --dsn "$ERP_DATABASE_DSN"
```

成功标志：

- `status` 中 `table_exists: true`
- `latest` 至少为 `v1.0.0`
- `psql` 能查到业务表，例如：

```bash
psql "$ERP_DATABASE_DSN" -c "\dt"
psql "$ERP_DATABASE_DSN" -c "SELECT version, description, applied_at FROM erp_schema_migration ORDER BY version;"
```

### 4.3 开发 / 演示环境初始化

**方式 A — Docker + 开发配置自动齐库**

```bash
docker compose up -d postgres
# configs/erp.dev.yaml 中 init_schema: true
go run ./cmd/erp-api
# 启动时会：baseline → upgrade → data-dev.sql
```

**方式 B — 手工**

```bash
./scripts/db-migrate.sh baseline
./scripts/db-migrate.sh upgrade --all
./scripts/db-migrate.sh seed-dev    # 仅开发！写入演示账号等
```

开发默认库账号见 `docker-compose.yml` / `configs/erp.dev.yaml`（`erp`/`erp`）。

### 4.4 生产配置要点

复制并修改：

```bash
cp configs/erp.prod.example.yaml configs/erp.prod.yaml
```

必须：

```yaml
database:
  driver: postgres
  dsn: "postgres://erp:强密码@DB主机:5432/erp_factory?sslmode=disable"
  init_schema: false          # 生产必须 false
  migrations_dir: migrations

seed:
  demo: false

jwt:
  secret: "足够长的随机密钥"   # 或环境变量 ERP_JWT_SECRET
```

环境变量覆盖示例：

```bash
export ERP_DATABASE_DSN='postgres://erp:...@db:5432/erp_factory?sslmode=require'
export ERP_DATABASE_INIT_SCHEMA=false
export ERP_JWT_SECRET='至少16位随机串'
```

先完成第 4.2 节初始化，再启动 `erp-api`。

---

## 5. 对接应用与验收

### 5.1 启动 API 前检查清单

1. Postgres 服务正常、`pg_isready` 成功  
2. `erp-db status` 已有 `v1.0.0`（及所需升级）  
3. 配置 `init_schema: false`（生产）  
4. DSN 用户为 `erp`，库为 `erp_factory`  
5. 上传目录等应用依赖目录可写（如 `data/uploads`）

### 5.2 健康检查

```bash
curl -s http://127.0.0.1:18080/api/v1/live
curl -s http://127.0.0.1:18080/api/v1/ready
# ready 通常会探测数据库
```

### 5.3 登录冒烟（开发种子环境）

若执行过 `seed-dev` / 开发 demo：按当前种子文档使用管理员账号（常见为 `admin` / `admin123`，以 `data-dev.sql` 与现场说明为准）。**生产账号由运维与管理员另行创建，勿沿用演示密码。**

---

## 6. 日常升级（存量库）

发布含数据库变更时：

```text
1. 维护窗口公告
2. 备份（必须）
3. erp-db status
4. erp-db upgrade --all
5. erp-db status（确认 latest）
6. 滚动重启 erp-api
7. 健康检查 + 核心业务冒烟
```

Ubuntu 一键（仍建议人工确认备份结果）：

```bash
./scripts/maint.sh upgrade-prod
```

Windows：

```powershell
.\scripts\backup.ps1
.\scripts\db-migrate.ps1 status
.\scripts\db-migrate.ps1 upgrade --all
# 然后重启 API 服务
```

规则摘要：

- 增量文件名：`vMAJOR.MINOR.PATCH_slug.sql`
- 新装环境应直接用最新 `schema.sql`，不必重放全部历史升级（baseline 已并入）
- 存量环境只跑 `upgrades/` 中尚未入账本的版本

---

## 7. 备份与恢复

### 7.1 备份

```bash
# Ubuntu
./scripts/backup.sh
# 产物默认：backups/erp_factory_YYYYMMDD_HHMMSS.dump

# 或手工
pg_dump --dbname="$ERP_DATABASE_DSN" -Fc -f /backup/erp_factory.dump
```

建议：每日全备 + 关键变更前临时备；备份文件异地保存。

### 7.2 恢复

```bash
# 警告：会清理并写回目标库
./scripts/backup.sh restore backups/erp_factory_XXXX.dump

# 或
pg_restore --clean --if-exists --dbname="$ERP_DATABASE_DSN" /backup/erp_factory.dump
```

恢复后执行 `erp-db status`，并重启 API。

### 7.3 回滚策略

**没有自动 down migration。**  
升级失败或业务异常：停 API → 恢复最近备份 → 重启 API → 联系研发排查升级脚本。

---

## 8. 常用运维命令速查

```bash
# 服务
sudo systemctl status postgresql
sudo systemctl restart postgresql

# 连通
pg_isready -h 127.0.0.1 -p 5432 -U erp
psql "$ERP_DATABASE_DSN" -c 'SELECT now();'

# 迁移
./scripts/db-migrate.sh status
./scripts/db-migrate.sh upgrade --all
./scripts/db-migrate.sh validate

# 环境自检
./scripts/maint.sh doctor
./scripts/maint.sh psql
```

查看连接与体积：

```sql
SELECT usename, application_name, client_addr, state
FROM pg_stat_activity
WHERE datname = 'erp_factory';

SELECT pg_size_pretty(pg_database_size('erp_factory'));
```

---

## 9. 故障排查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| `password authentication failed` | 密码/pg_hba 不匹配 | 核对角色密码与 `pg_hba.conf`，`reload` |
| `database "erp_factory" does not exist` | 未建库 | 见第 3 节 |
| `permission denied for schema public` | PG15+ 权限 | `GRANT`/`ALTER SCHEMA OWNER` 给 `erp` |
| `erp_schema_migration not found` | 未 baseline | 执行 `baseline` |
| `checksum mismatch` | 升级文件被改 | 恢复官方脚本或联系研发 |
| API 启动报缺表 | 生产未升级或 DSN 指错库 | `status` + `upgrade --all` |
| 连接数耗尽 | 连接泄漏/limit 过低 | 查 `pg_stat_activity`，调池与 `CONNECTION LIMIT` |
| 中文乱码 | 非 UTF8 库 | 重建 UTF8 库并恢复备份 |

---

## 10. 安全基线（生产）

1. 强密码；禁止默认 `erp/erp` 上生产  
2. `init_schema: false`；JWT / Trace HMAC 用环境变量注入  
3. 库不暴露公网；TLS（`sslmode=require`）优先  
4. 最小权限业务账号；备份加密与访问控制  
5. 定期演练恢复；升级窗口有回滚备份点  

---

## 11. 交付检查表（签字用）

- [ ] PostgreSQL 版本 ≥ 14（推荐 16）  
- [ ] 库 `erp_factory`、用户 `erp` 已创建，UTF8  
- [ ] `pg_hba` / 防火墙已按环境收紧  
- [ ] 已执行 `baseline`（及必要 `upgrade --all`）  
- [ ] `erp_schema_migration` 含 `v1.0.0`  
- [ ] **未**对生产执行 `seed-dev`  
- [ ] 应用配置 `init_schema: false`，DSN 正确  
- [ ] 首次备份已完成并可恢复演练  
- [ ] `/api/v1/live`、`/api/v1/ready` 正常  

---

## 12. 相关路径

| 路径 | 说明 |
|------|------|
| `migrations/erp/schema.sql` | 全量基线 |
| `migrations/erp/data-dev.sql` | 仅开发种子 |
| `migrations/erp/upgrades/README.md` | 增量规范 |
| `migrations/erp/archive/` | 历史 SQLite/MySQL 脚本（只读） |
| `configs/erp.dev.yaml` | 开发（可 `init_schema: true`） |
| `configs/erp.prod.example.yaml` | 生产模板 |
| `docker-compose.yml` | 本地 Postgres 服务 |
| `scripts/maint.sh` | Ubuntu 运维入口 |

修订记录：与仓库「全面转向 PostgreSQL」基线对齐；后续增量以 `erp-db` 与 `upgrades/` 为准。
