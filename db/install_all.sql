-- 一键安装：按依赖顺序执行全部 DDL + 权限种子
-- 用法（MySQL客户端）：
--   mysql -uroot -p < db/install_all.sql
-- 或在本机逐个 source

SOURCE schema/00_init.sql;
SOURCE schema/01_common.sql;
SOURCE schema/02_iam.sql;
SOURCE schema/03_product_inventory.sql;
SOURCE schema/04_production_payroll.sql;
SOURCE schema/05_hr.sql;
SOURCE schema/06_crm_sales_purchase.sql;
SOURCE schema/07_finance_asset.sql;
SOURCE schema/08_approval_system_report.sql;
SOURCE schema/09_farmer_weigh.sql;
SOURCE seed/01_iam_seed.sql;

SELECT 'erp_factory schema installed' AS message;
