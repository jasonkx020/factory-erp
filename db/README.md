# 旧 `db/` 目录说明

正式研发阶段 **请勿再使用** 本目录下的 SQLite/MySQL 脚本建库。

- 权威基线：[`../migrations/erp/schema.sql`](../migrations/erp/schema.sql)
- 增量升级：[`../migrations/erp/upgrades/`](../migrations/erp/upgrades/)
- 归档副本：[`../migrations/erp/archive/`](../migrations/erp/archive/)
- 工具：`./scripts/db-migrate.sh`（Ubuntu） / `.\scripts\db-migrate.ps1`（Windows）
