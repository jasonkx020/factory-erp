# Upgrades (增量迁移)

正式研发阶段：**存量库**靠本目录增量脚本升级；**新装**直接执行上级 `schema.sql`（已并入最新结构）。

## 命名

```
v{MAJOR}.{MINOR}.{PATCH}_{slug}.sql
例如: v1.0.1_add_box_image_url.sql
```

## 文件结构

1. 幂等 DDL（`IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS`）
2. 末尾 footer（checksum = body 的 SHA256，不含 footer）:

```sql
INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.1', 'description', '<sha256>')
ON CONFLICT (version) DO NOTHING;
```

用工具生成模板（自动算 checksum）:

```bash
go run ./cmd/erp-db create v1.0.1 "add foo"
```

## 应用

```bash
# 开发：配置 init_schema: true 启动时自动 baseline + upgrades + seed
# 生产：禁止启动改库，运维显式执行

go run ./cmd/erp-db status
go run ./cmd/erp-db upgrade --all
go run ./cmd/erp-db validate
```

## 规则

- 无 down migration；回滚靠 `pg_dump` 备份恢复
- 改结构后把变更**并回** `schema.sql`，保证新装一次齐库
- 已应用脚本勿改 body（checksum 会失败）
