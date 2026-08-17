# Archive（只读）

历史 SQLite / MySQL 建库脚本副本，**不参与运行时**。

- `sqlite/`：原 `db/sqlite/`（schema + seed）
- `mysql/`：原 `db/schema/`、`install_all.sql`、`install.ps1`、`seed/01_iam_seed.sql`

正式研发真源：

- [`../schema.sql`](../schema.sql) — PostgreSQL baseline v1.0.0
- [`../data-dev.sql`](../data-dev.sql)
- [`../upgrades/`](../upgrades/) — 增量（对齐 ycwl-freetv `erp-db` 账本模式）
