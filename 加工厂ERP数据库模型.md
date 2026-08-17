# 加工厂 ERP 数据库模型（PostgreSQL）

> 正式研发阶段以 [`migrations/erp/schema.sql`](../migrations/erp/schema.sql) 为唯一结构基线。  
> 引擎：**PostgreSQL 16+**。历史 SQLite/MySQL 脚本见 `migrations/erp/archive/`（只读）。

## 约定

| 项 | 取值 |
|----|------|
| PK | `BIGSERIAL` |
| 时间 | `TIMESTAMPTZ` / `TEXT`+`NOW()`（基线过渡期并存，新表优先 TIMESTAMPTZ） |
| 金额 | 优先 `NUMERIC(18,4)` |
| 升级 | `erp-db` / `scripts/db-migrate.sh`；账本表 `erp_schema_migration` |
| 生产 | `init_schema: false`；先备份再 `upgrade --all` |

## 维护

```bash
# Ubuntu
./scripts/maint.sh doctor
./scripts/db-migrate.sh status
./scripts/backup.sh

# Windows
.\scripts\db-migrate.ps1 status
.\scripts\backup.ps1
```
