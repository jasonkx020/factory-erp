-- ===== FILE: schema\00_init.sql =====
-- =============================================================================
-- 加工厂 ERP 数据库模型 · 物理 DDL（MySQL 8.0+）
-- 依据：加工厂ERP逻辑数据模型.md + 框架设计文档第7章权限
-- 字符集：utf8mb4 / InnoDB
-- PK：BIGINT 自增（生产可改为雪花，类型保持 BIGINT）
-- =============================================================================

CREATE DATABASE IF NOT EXISTS erp_factory
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE erp_factory;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;


-- ===== FILE: schema\01_common.sql =====
-- 00 公共约定字段说明（各表内联实现，不建物理基类表）
-- created_by, created_at, updated_by, updated_at, is_deleted, deleted_at, deleted_by, version

USE erp_factory;

-- ---------------------------------------------------------------------------
-- 组织主数据
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sys_organization (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64)  NOT NULL,
  name          VARCHAR(128) NOT NULL,
  status        VARCHAR(16)  NOT NULL DEFAULT 'active' COMMENT 'active/inactive',
  remark        VARCHAR(512) NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at    DATETIME(3) NULL,
  deleted_by    BIGINT NULL,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_org_code (code)
) ENGINE=InnoDB COMMENT='组织/公司';

CREATE TABLE IF NOT EXISTS sys_department (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  org_id        BIGINT NOT NULL,
  parent_id     BIGINT NULL,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at    DATETIME(3) NULL,
  deleted_by    BIGINT NULL,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_dept_org_code (org_id, code),
  KEY idx_dept_parent (parent_id),
  CONSTRAINT fk_dept_org FOREIGN KEY (org_id) REFERENCES sys_organization(id)
) ENGINE=InnoDB COMMENT='部门';

CREATE TABLE IF NOT EXISTS pd_workshop (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  org_id        BIGINT NOT NULL,
  dept_id       BIGINT NULL,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at    DATETIME(3) NULL,
  deleted_by    BIGINT NULL,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_ws_org_code (org_id, code),
  CONSTRAINT fk_ws_org FOREIGN KEY (org_id) REFERENCES sys_organization(id)
) ENGINE=InnoDB COMMENT='车间';

CREATE TABLE IF NOT EXISTS pd_work_team (
  id                   BIGINT PRIMARY KEY AUTO_INCREMENT,
  workshop_id          BIGINT NOT NULL,
  code                 VARCHAR(64) NOT NULL,
  name                 VARCHAR(128) NOT NULL,
  leader_employee_id   BIGINT NULL,
  status               VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by           BIGINT NULL,
  created_at           DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by           BIGINT NULL,
  updated_at           DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted           TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at           DATETIME(3) NULL,
  deleted_by           BIGINT NULL,
  version              INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_team_ws_code (workshop_id, code),
  CONSTRAINT fk_team_ws FOREIGN KEY (workshop_id) REFERENCES pd_workshop(id)
) ENGINE=InnoDB COMMENT='班组';

CREATE TABLE IF NOT EXISTS inv_warehouse (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  org_id          BIGINT NOT NULL,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  warehouse_type  VARCHAR(32) NOT NULL COMMENT 'raw/semi/finished/scrap/other',
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by      BIGINT NULL,
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at      DATETIME(3) NULL,
  deleted_by      BIGINT NULL,
  version         INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_wh_org_code (org_id, code),
  CONSTRAINT fk_wh_org FOREIGN KEY (org_id) REFERENCES sys_organization(id)
) ENGINE=InnoDB COMMENT='仓库';

CREATE TABLE IF NOT EXISTS inv_location (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  warehouse_id  BIGINT NOT NULL,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at    DATETIME(3) NULL,
  deleted_by    BIGINT NULL,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_loc_wh_code (warehouse_id, code),
  CONSTRAINT fk_loc_wh FOREIGN KEY (warehouse_id) REFERENCES inv_warehouse(id)
) ENGINE=InnoDB COMMENT='仓位/货位';

CREATE TABLE IF NOT EXISTS sys_dict_type (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  remark        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_dict_type_code (code)
) ENGINE=InnoDB COMMENT='字典类型';

CREATE TABLE IF NOT EXISTS sys_dict_item (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  dict_type_id  BIGINT NOT NULL,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  sort_no       INT NOT NULL DEFAULT 0,
  ext_json      JSON NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_dict_item (dict_type_id, code),
  CONSTRAINT fk_dict_item_type FOREIGN KEY (dict_type_id) REFERENCES sys_dict_type(id)
) ENGINE=InnoDB COMMENT='字典项';

CREATE TABLE IF NOT EXISTS sys_code_rule (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  biz_type      VARCHAR(64) NOT NULL COMMENT '单据业务类型',
  prefix        VARCHAR(32) NOT NULL DEFAULT '',
  date_pattern  VARCHAR(32) NOT NULL DEFAULT 'yyyyMMdd',
  seq_length    INT NOT NULL DEFAULT 4,
  current_seq   BIGINT NOT NULL DEFAULT 0,
  reset_policy  VARCHAR(16) NOT NULL DEFAULT 'daily' COMMENT 'daily/monthly/never',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_code_rule_biz (biz_type)
) ENGINE=InnoDB COMMENT='编码规则';

CREATE TABLE IF NOT EXISTS sys_biz_calendar (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  year          SMALLINT NOT NULL,
  period_no     TINYINT NOT NULL COMMENT '1-12',
  start_date    DATE NOT NULL,
  end_date      DATE NOT NULL,
  is_closed     TINYINT(1) NOT NULL DEFAULT 0,
  closed_at     DATETIME(3) NULL,
  closed_by     BIGINT NULL,
  UNIQUE KEY uk_calendar (year, period_no)
) ENGINE=InnoDB COMMENT='会计/业务期间';


-- ===== FILE: schema\02_iam.sql =====
-- =============================================================================
-- 权限全量模型（人事·权限分配 + 系统·自定义权限/菜单/登录/冻结）
-- 权限码格式：核心功能:功能模块:动作
-- =============================================================================
USE erp_factory;

-- 员工（人账一体，先建员工再挂用户；user_id 后补）
CREATE TABLE IF NOT EXISTS hr_employee (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  emp_no        VARCHAR(64) NOT NULL,
  name          VARCHAR(64) NOT NULL,
  org_id        BIGINT NOT NULL,
  dept_id       BIGINT NULL,
  workshop_id   BIGINT NULL,
  team_id       BIGINT NULL,
  job_title     VARCHAR(64) NULL,
  emp_type      VARCHAR(16) NOT NULL DEFAULT 'office' COMMENT 'piece/fixed/office',
  mobile        VARCHAR(32) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/inactive/left',
  user_id       BIGINT NULL COMMENT '关联登录用户，循环引用用应用层维护',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at    DATETIME(3) NULL,
  deleted_by    BIGINT NULL,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_emp_no (emp_no),
  KEY idx_emp_org (org_id),
  KEY idx_emp_user (user_id),
  CONSTRAINT fk_emp_org FOREIGN KEY (org_id) REFERENCES sys_organization(id)
) ENGINE=InnoDB COMMENT='员工';

-- ---------------------------------------------------------------------------
-- 用户 / 角色 / 分组
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS iam_user (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  login_name      VARCHAR(64) NOT NULL,
  password_hash   VARCHAR(255) NOT NULL,
  employee_id     BIGINT NULL,
  user_type       VARCHAR(16) NOT NULL DEFAULT 'biz' COMMENT 'admin/biz/line',
  status          VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/frozen',
  freeze_reason   VARCHAR(512) NULL,
  frozen_at       DATETIME(3) NULL,
  frozen_by       BIGINT NULL,
  last_login_at   DATETIME(3) NULL,
  login_fail_count INT NOT NULL DEFAULT 0,
  lock_until      DATETIME(3) NULL,
  pwd_changed_at  DATETIME(3) NULL,
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by      BIGINT NULL,
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at      DATETIME(3) NULL,
  deleted_by      BIGINT NULL,
  version         INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_login_name (login_name),
  KEY idx_user_emp (employee_id),
  KEY idx_user_status (status),
  CONSTRAINT fk_user_emp FOREIGN KEY (employee_id) REFERENCES hr_employee(id)
) ENGINE=InnoDB COMMENT='登录用户/管理员';

CREATE TABLE IF NOT EXISTS iam_admin_group (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  remark        VARCHAR(512) NULL,
  sort_no       INT NOT NULL DEFAULT 0,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at    DATETIME(3) NULL,
  deleted_by    BIGINT NULL,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_admin_group_code (code)
) ENGINE=InnoDB COMMENT='管理员分组';

CREATE TABLE IF NOT EXISTS iam_role (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  code              VARCHAR(64) NOT NULL,
  name              VARCHAR(128) NOT NULL,
  data_scope_type   VARCHAR(16) NOT NULL DEFAULT 'self' COMMENT 'self/team/workshop/warehouse/all',
  remark            VARCHAR(512) NULL,
  is_system         TINYINT(1) NOT NULL DEFAULT 0 COMMENT '预置角色不可删',
  status            VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by        BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by        BIGINT NULL,
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted        TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at        DATETIME(3) NULL,
  deleted_by        BIGINT NULL,
  version           INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_role_code (code)
) ENGINE=InnoDB COMMENT='角色';

CREATE TABLE IF NOT EXISTS iam_admin_group_role (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  group_id      BIGINT NOT NULL,
  role_id       BIGINT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_agr (group_id, role_id),
  CONSTRAINT fk_agr_group FOREIGN KEY (group_id) REFERENCES iam_admin_group(id),
  CONSTRAINT fk_agr_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='管理员分组-角色';

CREATE TABLE IF NOT EXISTS iam_admin_group_user (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  group_id      BIGINT NOT NULL,
  user_id       BIGINT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_agu (group_id, user_id),
  CONSTRAINT fk_agu_group FOREIGN KEY (group_id) REFERENCES iam_admin_group(id),
  CONSTRAINT fk_agu_user FOREIGN KEY (user_id) REFERENCES iam_user(id)
) ENGINE=InnoDB COMMENT='管理员分组-用户';

CREATE TABLE IF NOT EXISTS iam_user_role (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  role_id       BIGINT NOT NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_user_role (user_id, role_id),
  KEY idx_ur_role (role_id),
  CONSTRAINT fk_ur_user FOREIGN KEY (user_id) REFERENCES iam_user(id),
  CONSTRAINT fk_ur_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='用户-角色（一人多角色，权限并集）';

-- ---------------------------------------------------------------------------
-- 权限码 / 角色权限
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS iam_permission (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(256) NOT NULL COMMENT '核心功能:功能模块:动作',
  name          VARCHAR(128) NOT NULL,
  domain        VARCHAR(64) NOT NULL COMMENT '13大核心功能名',
  module        VARCHAR(128) NOT NULL COMMENT '功能模块名',
  action        VARCHAR(32) NOT NULL COMMENT '查看/新增/编辑/删除/审批/导出/打印等',
  remark        VARCHAR(512) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_perm_code (code),
  KEY idx_perm_domain_module (domain, module)
) ENGINE=InnoDB COMMENT='权限码（自定义权限）';

CREATE TABLE IF NOT EXISTS iam_role_permission (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id         BIGINT NOT NULL,
  permission_id   BIGINT NOT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rp (role_id, permission_id),
  CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES iam_role(id),
  CONSTRAINT fk_rp_perm FOREIGN KEY (permission_id) REFERENCES iam_permission(id)
) ENGINE=InnoDB COMMENT='角色-权限码';

-- ---------------------------------------------------------------------------
-- 数据范围细化：仓 / 工序（角色级默认；用户级可再收紧）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS iam_role_warehouse_scope (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT NOT NULL,
  warehouse_id  BIGINT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rws (role_id, warehouse_id),
  CONSTRAINT fk_rws_role FOREIGN KEY (role_id) REFERENCES iam_role(id),
  CONSTRAINT fk_rws_wh FOREIGN KEY (warehouse_id) REFERENCES inv_warehouse(id)
) ENGINE=InnoDB COMMENT='角色仓范围';

CREATE TABLE IF NOT EXISTS iam_role_process_scope (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT NOT NULL,
  process_id    BIGINT NOT NULL COMMENT '关联 pd_process.id，工序表建后约束',
  can_report    TINYINT(1) NOT NULL DEFAULT 1,
  can_dispatch  TINYINT(1) NOT NULL DEFAULT 0,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rps (role_id, process_id),
  CONSTRAINT fk_rps_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='角色工序范围';

CREATE TABLE IF NOT EXISTS iam_user_warehouse_scope (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  warehouse_id  BIGINT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_uws (user_id, warehouse_id),
  CONSTRAINT fk_uws_user FOREIGN KEY (user_id) REFERENCES iam_user(id),
  CONSTRAINT fk_uws_wh FOREIGN KEY (warehouse_id) REFERENCES inv_warehouse(id)
) ENGINE=InnoDB COMMENT='用户仓范围（相对角色再收紧，空则继承角色）';

CREATE TABLE IF NOT EXISTS iam_user_process_scope (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  process_id    BIGINT NOT NULL,
  can_report    TINYINT(1) NOT NULL DEFAULT 1,
  can_dispatch  TINYINT(1) NOT NULL DEFAULT 0,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ups (user_id, process_id),
  CONSTRAINT fk_ups_user FOREIGN KEY (user_id) REFERENCES iam_user(id)
) ENGINE=InnoDB COMMENT='用户工序范围';

CREATE TABLE IF NOT EXISTS iam_user_data_scope (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id           BIGINT NOT NULL,
  data_scope_type   VARCHAR(16) NOT NULL COMMENT 'self/team/workshop/warehouse/all',
  workshop_id       BIGINT NULL,
  team_id           BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_uds_user (user_id),
  CONSTRAINT fk_uds_user FOREIGN KEY (user_id) REFERENCES iam_user(id)
) ENGINE=InnoDB COMMENT='用户数据范围覆盖（空则用角色默认）';

-- ---------------------------------------------------------------------------
-- 字段策略 / 成本隐藏 / 自定义菜单
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS iam_field_policy (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT NOT NULL,
  field_key     VARCHAR(128) NOT NULL COMMENT '如 cost_price/gross_margin/other_wage',
  field_name    VARCHAR(128) NULL,
  visible       TINYINT(1) NOT NULL DEFAULT 0,
  editable      TINYINT(1) NOT NULL DEFAULT 0,
  remark        VARCHAR(256) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_field_policy (role_id, field_key),
  CONSTRAINT fk_fp_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='字段可见策略（成本隐藏等）';

CREATE TABLE IF NOT EXISTS iam_menu_custom (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT NOT NULL,
  domain        VARCHAR(64) NOT NULL,
  module        VARCHAR(128) NOT NULL,
  menu_key      VARCHAR(256) NOT NULL COMMENT 'domain:module',
  visible       TINYINT(1) NOT NULL DEFAULT 1,
  sort_no       INT NOT NULL DEFAULT 0,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_menu_role_key (role_id, menu_key),
  CONSTRAINT fk_menu_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='自定义菜单（按角色裁剪）';

CREATE TABLE IF NOT EXISTS iam_login_policy (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  org_id            BIGINT NULL COMMENT '空=全局策略',
  max_fail_count    INT NOT NULL DEFAULT 5,
  lock_minutes      INT NOT NULL DEFAULT 30,
  session_ttl_min   INT NOT NULL DEFAULT 120,
  password_min_len  INT NOT NULL DEFAULT 8,
  password_require_letter TINYINT(1) NOT NULL DEFAULT 1,
  password_require_digit  TINYINT(1) NOT NULL DEFAULT 1,
  password_require_special TINYINT(1) NOT NULL DEFAULT 0,
  password_history  INT NOT NULL DEFAULT 5,
  force_change_days INT NULL,
  single_session    TINYINT(1) NOT NULL DEFAULT 0,
  password_rule_json JSON NULL,
  updated_by        BIGINT NULL,
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='登录控制策略';

CREATE TABLE IF NOT EXISTS iam_user_session (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  token_hash    VARCHAR(128) NOT NULL,
  client_type   VARCHAR(32) NULL COMMENT 'web/mp_worker/mp_sales/pad/boss',
  ip            VARCHAR(64) NULL,
  user_agent    VARCHAR(512) NULL,
  login_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  expire_at     DATETIME(3) NOT NULL,
  revoked_at    DATETIME(3) NULL,
  KEY idx_session_user (user_id),
  KEY idx_session_token (token_hash),
  CONSTRAINT fk_session_user FOREIGN KEY (user_id) REFERENCES iam_user(id)
) ENGINE=InnoDB COMMENT='登录会话（强制下线）';

CREATE TABLE IF NOT EXISTS iam_password_history (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_pwd_hist_user (user_id),
  CONSTRAINT fk_pwd_hist_user FOREIGN KEY (user_id) REFERENCES iam_user(id)
) ENGINE=InnoDB COMMENT='历史密码';

-- 入职赋权模板（角色列表）
CREATE TABLE IF NOT EXISTS iam_onboard_role_template (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  emp_type      VARCHAR(16) NOT NULL COMMENT 'piece/fixed/office/admin',
  role_id       BIGINT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_onboard_tpl (emp_type, role_id),
  CONSTRAINT fk_onboard_tpl_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='入职默认角色模板';


-- ===== FILE: schema\03_product_inventory.sql =====
USE erp_factory;

-- ---------------------------------------------------------------------------
-- 产品管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS prd_product (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  code              VARCHAR(64) NOT NULL,
  name              VARCHAR(256) NOT NULL,
  category          VARCHAR(64) NULL,
  product_type      VARCHAR(32) NOT NULL COMMENT 'raw/semi/finished/aux/scrap',
  base_unit_id      BIGINT NULL,
  spec_text         VARCHAR(256) NULL,
  barcode           VARCHAR(64) NULL,
  cost_price        DECIMAL(18,4) NULL,
  sale_price        DECIMAL(18,4) NULL,
  is_batch_managed  TINYINT(1) NOT NULL DEFAULT 1,
  is_box_managed    TINYINT(1) NOT NULL DEFAULT 0,
  status            VARCHAR(16) NOT NULL DEFAULT 'active',
  created_by        BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by        BIGINT NULL,
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted        TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at        DATETIME(3) NULL,
  deleted_by        BIGINT NULL,
  version           INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_product_code (code)
) ENGINE=InnoDB COMMENT='产品/物料档案';

CREATE TABLE IF NOT EXISTS prd_product_unit (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id      BIGINT NOT NULL,
  unit_name       VARCHAR(32) NOT NULL,
  is_base         TINYINT(1) NOT NULL DEFAULT 0,
  factor_to_base  DECIMAL(18,8) NOT NULL DEFAULT 1,
  is_purchase     TINYINT(1) NOT NULL DEFAULT 1,
  is_sale         TINYINT(1) NOT NULL DEFAULT 1,
  is_stock        TINYINT(1) NOT NULL DEFAULT 1,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_prod_unit (product_id, unit_name),
  CONSTRAINT fk_pu_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='产品单位';

CREATE TABLE IF NOT EXISTS prd_product_spec (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id      BIGINT NOT NULL,
  spec_code       VARCHAR(64) NOT NULL,
  routing_id      BIGINT NULL,
  process_wage_bind_json JSON NULL,
  remark          VARCHAR(512) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_prod_spec (product_id, spec_code),
  CONSTRAINT fk_ps_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='生产规格绑定';

CREATE TABLE IF NOT EXISTS prd_product_app_sort (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id    BIGINT NOT NULL,
  channel       VARCHAR(32) NOT NULL DEFAULT 'app',
  sort_no       INT NOT NULL DEFAULT 0,
  is_visible    TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY uk_app_sort (product_id, channel),
  CONSTRAINT fk_pas_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='APP产品排序';

-- ---------------------------------------------------------------------------
-- 库存管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS inv_box_code (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  product_id    BIGINT NOT NULL,
  warehouse_id  BIGINT NULL,
  batch_no      VARCHAR(64) NULL,
  qty           DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight        DECIMAL(18,4) NULL,
  parent_box_id BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_box_code (code),
  CONSTRAINT fk_box_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='箱码';

CREATE TABLE IF NOT EXISTS inv_balance (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  warehouse_id  BIGINT NOT NULL,
  location_id   BIGINT NOT NULL DEFAULT 0,
  product_id    BIGINT NOT NULL,
  batch_no      VARCHAR(64) NOT NULL DEFAULT '',
  box_code_id   BIGINT NOT NULL DEFAULT 0,
  qty           DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight        DECIMAL(18,4) NULL,
  avg_cost      DECIMAL(18,4) NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_balance (warehouse_id, product_id, batch_no, location_id, box_code_id),
  KEY idx_bal_product (product_id),
  CONSTRAINT fk_bal_wh FOREIGN KEY (warehouse_id) REFERENCES inv_warehouse(id),
  CONSTRAINT fk_bal_product FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='库存结存';

CREATE TABLE IF NOT EXISTS inv_stock_txn (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  doc_type          VARCHAR(32) NOT NULL COMMENT 'opening/purchase_in/purchase_return/produce_in/requisition_out/transfer/stocktake_gain/stocktake_loss/sales_out/sales_return/consume',
  biz_date          DATE NOT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  warehouse_id      BIGINT NULL,
  counterparty_type VARCHAR(32) NULL,
  counterparty_id   BIGINT NULL,
  source_doc_type   VARCHAR(64) NULL,
  source_doc_id     BIGINT NULL,
  posted_at         DATETIME(3) NULL,
  org_id            BIGINT NULL,
  owner_user_id     BIGINT NULL,
  remark            VARCHAR(512) NULL,
  created_by        BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by        BIGINT NULL,
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted        TINYINT(1) NOT NULL DEFAULT 0,
  deleted_at        DATETIME(3) NULL,
  deleted_by        BIGINT NULL,
  version           INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_stock_txn_no (doc_no),
  KEY idx_txn_type_date (doc_type, biz_date),
  KEY idx_txn_source (source_doc_type, source_doc_id)
) ENGINE=InnoDB COMMENT='出入库流水头（唯一过账事实源）';

CREATE TABLE IF NOT EXISTS inv_stock_txn_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  txn_id        BIGINT NOT NULL,
  line_no       INT NOT NULL,
  product_id    BIGINT NOT NULL,
  unit_id       BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  base_qty      DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  batch_no      VARCHAR(64) NULL,
  box_code_id   BIGINT NULL,
  location_id   BIGINT NULL,
  direction     VARCHAR(8) NOT NULL COMMENT 'in/out',
  amount        DECIMAL(18,4) NULL,
  remark        VARCHAR(256) NULL,
  UNIQUE KEY uk_txn_line (txn_id, line_no),
  CONSTRAINT fk_txn_line_hdr FOREIGN KEY (txn_id) REFERENCES inv_stock_txn(id),
  CONSTRAINT fk_txn_line_prd FOREIGN KEY (product_id) REFERENCES prd_product(id)
) ENGINE=InnoDB COMMENT='出入库流水行';

CREATE TABLE IF NOT EXISTS inv_reservation (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  warehouse_id    BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  batch_no        VARCHAR(64) NULL,
  qty             DECIMAL(18,4) NOT NULL,
  source_doc_type VARCHAR(64) NOT NULL,
  source_doc_id   BIGINT NOT NULL,
  source_line_id  BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/released',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_rsv_product (warehouse_id, product_id, status),
  KEY idx_rsv_source (source_doc_type, source_doc_id)
) ENGINE=InnoDB COMMENT='待用/占用';

CREATE TABLE IF NOT EXISTS inv_in_transit (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id      BIGINT NOT NULL,
  warehouse_id    BIGINT NOT NULL COMMENT '目标仓',
  qty             DECIMAL(18,4) NOT NULL,
  transit_type    VARCHAR(32) NOT NULL COMMENT 'purchase/transfer',
  source_doc_type VARCHAR(64) NOT NULL,
  source_doc_id   BIGINT NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'open' COMMENT 'open/closed',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_transit (warehouse_id, product_id, status)
) ENGINE=InnoDB COMMENT='在途量';

CREATE TABLE IF NOT EXISTS inv_inbound_qc (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  stock_txn_id  BIGINT NULL,
  product_id    BIGINT NOT NULL,
  qty_check     DECIMAL(18,4) NOT NULL,
  qty_pass      DECIMAL(18,4) NOT NULL DEFAULT 0,
  qty_fail      DECIMAL(18,4) NOT NULL DEFAULT 0,
  result        VARCHAR(16) NULL,
  inspector_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_inbound_qc_no (doc_no)
) ENGINE=InnoDB COMMENT='入库质检';

CREATE TABLE IF NOT EXISTS inv_stocktake (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  stocktake_type  VARCHAR(16) NOT NULL COMMENT 'warehouse/workshop',
  warehouse_id    BIGINT NULL,
  workshop_id     BIGINT NULL,
  biz_date        DATE NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  version         INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_stocktake_no (doc_no)
) ENGINE=InnoDB COMMENT='盘点单头';

CREATE TABLE IF NOT EXISTS inv_stocktake_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  stocktake_id  BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  book_qty      DECIMAL(18,4) NOT NULL DEFAULT 0,
  count_qty     DECIMAL(18,4) NOT NULL DEFAULT 0,
  diff_qty      DECIMAL(18,4) NOT NULL DEFAULT 0,
  batch_no      VARCHAR(64) NULL,
  location_id   BIGINT NULL,
  CONSTRAINT fk_stl_hdr FOREIGN KEY (stocktake_id) REFERENCES inv_stocktake(id)
) ENGINE=InnoDB COMMENT='盘点行';

CREATE TABLE IF NOT EXISTS inv_transfer (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no              VARCHAR(64) NOT NULL,
  from_warehouse_id   BIGINT NOT NULL,
  to_warehouse_id     BIGINT NOT NULL,
  biz_date            DATE NOT NULL,
  status              VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by          BIGINT NULL,
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted          TINYINT(1) NOT NULL DEFAULT 0,
  version             INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_transfer_no (doc_no)
) ENGINE=InnoDB COMMENT='调拨单头';

CREATE TABLE IF NOT EXISTS inv_transfer_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  transfer_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  base_qty      DECIMAL(18,4) NOT NULL,
  batch_no      VARCHAR(64) NULL,
  CONSTRAINT fk_trl_hdr FOREIGN KEY (transfer_id) REFERENCES inv_transfer(id)
) ENGINE=InnoDB COMMENT='调拨行';

CREATE TABLE IF NOT EXISTS inv_assemble_split (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  biz_type      VARCHAR(16) NOT NULL COMMENT 'assemble/split',
  warehouse_id  BIGINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_asm_no (doc_no)
) ENGINE=InnoDB COMMENT='组装拆分单';

CREATE TABLE IF NOT EXISTS inv_assemble_split_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  header_id     BIGINT NOT NULL,
  role_type     VARCHAR(16) NOT NULL COMMENT 'parent/child',
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_asl_hdr FOREIGN KEY (header_id) REFERENCES inv_assemble_split(id)
) ENGINE=InnoDB COMMENT='组装拆分行';

CREATE TABLE IF NOT EXISTS inv_price_adjust (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  product_id    BIGINT NOT NULL,
  old_price     DECIMAL(18,4) NOT NULL,
  new_price     DECIMAL(18,4) NOT NULL,
  effective_at  DATETIME(3) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_price_adj_no (doc_no)
) ENGINE=InnoDB COMMENT='商品调价单';

CREATE TABLE IF NOT EXISTS inv_stock_alert_rule (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id    BIGINT NULL,
  warehouse_id  BIGINT NULL,
  alert_type    VARCHAR(16) NOT NULL COMMENT 'shortage/excess',
  min_qty       DECIMAL(18,4) NULL,
  max_qty       DECIMAL(18,4) NULL,
  is_enabled    TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='亏料/过量预警规则';

CREATE TABLE IF NOT EXISTS inv_sales_peel_return (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  sales_order_id  BIGINT NULL,
  product_id      BIGINT NOT NULL,
  peel_qty        DECIMAL(18,4) NOT NULL,
  weight          DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_peel_no (doc_no)
) ENGINE=InnoDB COMMENT='销售退皮';

CREATE TABLE IF NOT EXISTS inv_material_to_payable (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  consume_txn_id  BIGINT NULL,
  supplier_id     BIGINT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_mtp_no (doc_no)
) ENGINE=InnoDB COMMENT='物料转应付';


-- ===== FILE: schema\04_production_payroll.sql =====
USE erp_factory;

-- ---------------------------------------------------------------------------
-- 生产管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pd_process (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  code                VARCHAR(64) NOT NULL,
  name                VARCHAR(128) NOT NULL,
  process_type        VARCHAR(32) NULL COMMENT 'wash/peel/cut/core/dice/bag/other',
  is_piecework        TINYINT(1) NOT NULL DEFAULT 0,
  is_handover_point   TINYINT(1) NOT NULL DEFAULT 0 COMMENT '收货卡点',
  status              VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted          TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_process_code (code)
) ENGINE=InnoDB COMMENT='工序';

-- 补全 IAM 工序范围外键
ALTER TABLE iam_role_process_scope
  ADD CONSTRAINT fk_rps_process FOREIGN KEY (process_id) REFERENCES pd_process(id);
ALTER TABLE iam_user_process_scope
  ADD CONSTRAINT fk_ups_process FOREIGN KEY (process_id) REFERENCES pd_process(id);

CREATE TABLE IF NOT EXISTS pd_process_price (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  process_id      BIGINT NOT NULL,
  product_spec_id BIGINT NULL,
  unit_id         BIGINT NULL,
  price           DECIMAL(18,4) NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_pp_process (process_id),
  CONSTRAINT fk_pp_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工序工价入口（生产侧）';

CREATE TABLE IF NOT EXISTS pd_routing (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  product_id    BIGINT NULL,
  version_no    VARCHAR(32) NOT NULL DEFAULT 'V1',
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_routing_code_ver (code, version_no)
) ENGINE=InnoDB COMMENT='工艺流程';

CREATE TABLE IF NOT EXISTS pd_routing_step (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  routing_id            BIGINT NOT NULL,
  seq_no                INT NOT NULL,
  process_id            BIGINT NOT NULL,
  is_inbound_checkpoint TINYINT(1) NOT NULL DEFAULT 0,
  is_qc_required        TINYINT(1) NOT NULL DEFAULT 0,
  workshop_id           BIGINT NULL,
  UNIQUE KEY uk_routing_step (routing_id, seq_no),
  CONSTRAINT fk_rs_routing FOREIGN KEY (routing_id) REFERENCES pd_routing(id),
  CONSTRAINT fk_rs_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工艺步骤';

CREATE TABLE IF NOT EXISTS pd_bom (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  code              VARCHAR(64) NOT NULL,
  product_id        BIGINT NOT NULL,
  version_no        VARCHAR(32) NOT NULL DEFAULT 'V1',
  is_auto_generated TINYINT(1) NOT NULL DEFAULT 0,
  status            VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_bom (code, version_no)
) ENGINE=InnoDB COMMENT='BOM头';

CREATE TABLE IF NOT EXISTS pd_bom_line (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  bom_id                BIGINT NOT NULL,
  component_product_id  BIGINT NOT NULL,
  qty                   DECIMAL(18,4) NOT NULL,
  unit_id               BIGINT NULL,
  scrap_rate            DECIMAL(8,4) NOT NULL DEFAULT 0,
  CONSTRAINT fk_bl_bom FOREIGN KEY (bom_id) REFERENCES pd_bom(id)
) ENGINE=InnoDB COMMENT='BOM行';

CREATE TABLE IF NOT EXISTS pd_production_task (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  source_type   VARCHAR(16) NOT NULL DEFAULT 'manual' COMMENT 'sales/manual/merge',
  status        VARCHAR(16) NOT NULL DEFAULT 'pending',
  plan_start    DATETIME(3) NULL,
  plan_end      DATETIME(3) NULL,
  routing_id    BIGINT NULL,
  workshop_id   BIGINT NULL,
  org_id        BIGINT NULL,
  owner_user_id BIGINT NULL,
  remark        VARCHAR(512) NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_by    BIGINT NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  version       INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_task_no (doc_no)
) ENGINE=InnoDB COMMENT='生产任务单';

CREATE TABLE IF NOT EXISTS pd_production_task_item (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  task_id         BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  plan_qty        DECIMAL(18,4) NOT NULL,
  plan_weight     DECIMAL(18,4) NULL,
  completed_qty   DECIMAL(18,4) NOT NULL DEFAULT 0,
  CONSTRAINT fk_pti_task FOREIGN KEY (task_id) REFERENCES pd_production_task(id)
) ENGINE=InnoDB COMMENT='任务单商品行（一单多商品）';

CREATE TABLE IF NOT EXISTS pd_task_merge (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  merge_no      VARCHAR(64) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_merge_no (merge_no)
) ENGINE=InnoDB COMMENT='多单整合';

CREATE TABLE IF NOT EXISTS pd_task_merge_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  merge_id        BIGINT NOT NULL,
  source_doc_type VARCHAR(64) NOT NULL,
  source_doc_id   BIGINT NOT NULL,
  task_id         BIGINT NULL,
  CONSTRAINT fk_tml_merge FOREIGN KEY (merge_id) REFERENCES pd_task_merge(id)
) ENGINE=InnoDB COMMENT='多单整合来源';

CREATE TABLE IF NOT EXISTS pd_work_order (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  task_id           BIGINT NOT NULL,
  process_id        BIGINT NOT NULL,
  routing_step_id   BIGINT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'pending',
  plan_qty          DECIMAL(18,4) NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_wo_no (doc_no),
  CONSTRAINT fk_wo_task FOREIGN KEY (task_id) REFERENCES pd_production_task(id),
  CONSTRAINT fk_wo_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工单';

CREATE TABLE IF NOT EXISTS pd_dispatch (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  work_order_id   BIGINT NOT NULL,
  dispatch_type   VARCHAR(16) NOT NULL DEFAULT 'normal' COMMENT 'normal/flex',
  worker_id       BIGINT NULL COMMENT 'hr_employee.id',
  team_id         BIGINT NULL,
  plan_qty        DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'dispatched',
  dispatched_at   DATETIME(3) NULL,
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_dispatch_no (doc_no),
  CONSTRAINT fk_dp_wo FOREIGN KEY (work_order_id) REFERENCES pd_work_order(id)
) ENGINE=InnoDB COMMENT='派工/灵活派发';

CREATE TABLE IF NOT EXISTS pd_material_requisition (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  work_order_id   BIGINT NULL,
  dispatch_id     BIGINT NULL,
  warehouse_id    BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_mr_no (doc_no)
) ENGINE=InnoDB COMMENT='联动领料头';

CREATE TABLE IF NOT EXISTS pd_material_requisition_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  requisition_id  BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  qty             DECIMAL(18,4) NOT NULL,
  base_qty        DECIMAL(18,4) NOT NULL,
  batch_no        VARCHAR(64) NULL,
  CONSTRAINT fk_mrl_hdr FOREIGN KEY (requisition_id) REFERENCES pd_material_requisition(id)
) ENGINE=InnoDB COMMENT='领料行';

CREATE TABLE IF NOT EXISTS pd_report_work (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  dispatch_id     BIGINT NULL,
  work_order_id   BIGINT NULL,
  process_id      BIGINT NOT NULL,
  worker_id       BIGINT NOT NULL,
  report_type     VARCHAR(16) NOT NULL DEFAULT 'output' COMMENT 'output/handover',
  qty             DECIMAL(18,4) NOT NULL,
  weight          DECIMAL(18,4) NULL,
  qty_net         DECIMAL(18,4) NULL,
  deduct_impurity DECIMAL(18,4) NOT NULL DEFAULT 0,
  deduct_water    DECIMAL(18,4) NOT NULL DEFAULT 0,
  qc_result       VARCHAR(16) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'submitted',
  reported_at     DATETIME(3) NOT NULL,
  scan_code       VARCHAR(128) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rw_no (doc_no),
  KEY idx_rw_worker_date (worker_id, reported_at),
  CONSTRAINT fk_rw_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='扫码报工（含工序收货交接）';

CREATE TABLE IF NOT EXISTS pd_piecework_summary (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  worker_id     BIGINT NOT NULL,
  process_id    BIGINT NOT NULL,
  biz_date      DATE NOT NULL,
  qty           DECIMAL(18,4) NOT NULL DEFAULT 0,
  weight        DECIMAL(18,4) NULL,
  amount        DECIMAL(18,4) NOT NULL DEFAULT 0,
  source_report_ids JSON NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pws (worker_id, process_id, biz_date)
) ENGINE=InnoDB COMMENT='计件产量汇总';

CREATE TABLE IF NOT EXISTS pd_qc_order (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  qc_type         VARCHAR(32) NOT NULL COMMENT 'process/finished/incoming_link',
  source_doc_type VARCHAR(64) NULL,
  source_doc_id   BIGINT NULL,
  product_id      BIGINT NULL,
  result          VARCHAR(16) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_qc_no (doc_no)
) ENGINE=InnoDB COMMENT='质检单';

CREATE TABLE IF NOT EXISTS pd_rework_order (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  source_qc_id  BIGINT NULL,
  task_id       BIGINT NULL,
  process_id    BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rework_no (doc_no)
) ENGINE=InnoDB COMMENT='返修单';

CREATE TABLE IF NOT EXISTS pd_scrap_record (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  task_id       BIGINT NULL,
  process_id    BIGINT NULL,
  product_id    BIGINT NOT NULL COMMENT '废料料号',
  qty           DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  disposition   VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_scrap_no (doc_no)
) ENGINE=InnoDB COMMENT='废料登记';

CREATE TABLE IF NOT EXISTS pd_drawing_link (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  drawing_id    BIGINT NOT NULL,
  task_id       BIGINT NULL,
  work_order_id BIGINT NULL,
  process_id    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='图纸分发挂接';

CREATE TABLE IF NOT EXISTS pd_cost_hide_policy (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT NOT NULL,
  field_scope   JSON NOT NULL,
  is_enabled    TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cost_hide_role (role_id),
  CONSTRAINT fk_chp_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='成本隐藏策略（可与 iam_field_policy 并存）';

CREATE TABLE IF NOT EXISTS pd_outsource_order (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  supplier_id   BIGINT NULL,
  process_id    BIGINT NULL,
  product_id    BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_out_no (doc_no)
) ENGINE=InnoDB COMMENT='委外加工单';

CREATE TABLE IF NOT EXISTS pd_consignment_order (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  customer_id   BIGINT NULL,
  product_id    BIGINT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  progress      VARCHAR(64) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cons_no (doc_no)
) ENGINE=InnoDB COMMENT='受托加工单';

CREATE TABLE IF NOT EXISTS pd_mrp_run (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_no        VARCHAR(64) NOT NULL,
  run_at        DATETIME(3) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'done',
  params_json   JSON NULL,
  UNIQUE KEY uk_mrp_run (run_no)
) ENGINE=InnoDB COMMENT='MRP运算';

CREATE TABLE IF NOT EXISTS pd_mrp_result (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_id        BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  demand_qty    DECIMAL(18,4) NOT NULL DEFAULT 0,
  supply_qty    DECIMAL(18,4) NOT NULL DEFAULT 0,
  shortage_qty  DECIMAL(18,4) NOT NULL DEFAULT 0,
  suggest_action VARCHAR(64) NULL,
  CONSTRAINT fk_mrp_run FOREIGN KEY (run_id) REFERENCES pd_mrp_run(id)
) ENGINE=InnoDB COMMENT='MRP结果';

-- ---------------------------------------------------------------------------
-- 工资管理
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pay_worker_profile (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  pay_type      VARCHAR(16) NOT NULL DEFAULT 'piece' COMMENT 'piece/fixed/mixed',
  bank_account  VARCHAR(64) NULL,
  tax_no        VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pay_emp (employee_id),
  CONSTRAINT fk_pwp_emp FOREIGN KEY (employee_id) REFERENCES hr_employee(id)
) ENGINE=InnoDB COMMENT='工人工资档案';

CREATE TABLE IF NOT EXISTS pay_process_wage_rate (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  process_id      BIGINT NOT NULL,
  product_id      BIGINT NULL,
  product_spec_id BIGINT NULL,
  unit_id         BIGINT NULL,
  rate            DECIMAL(18,4) NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_pwr_process (process_id),
  CONSTRAINT fk_pwr_process FOREIGN KEY (process_id) REFERENCES pd_process(id)
) ENGINE=InnoDB COMMENT='工序工资（工价表）';

CREATE TABLE IF NOT EXISTS pay_payroll_sheet (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  period_year   SMALLINT NOT NULL,
  period_month  TINYINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  calc_at       DATETIME(3) NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_payroll_no (doc_no),
  UNIQUE KEY uk_payroll_period (period_year, period_month)
) ENGINE=InnoDB COMMENT='工资单头';

CREATE TABLE IF NOT EXISTS pay_payroll_sheet_line (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  sheet_id            BIGINT NOT NULL,
  employee_id         BIGINT NOT NULL,
  piece_amount        DECIMAL(18,4) NOT NULL DEFAULT 0,
  attendance_amount   DECIMAL(18,4) NOT NULL DEFAULT 0,
  commission_amount   DECIMAL(18,4) NOT NULL DEFAULT 0,
  adjust_amount       DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_amount        DECIMAL(18,4) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_psl (sheet_id, employee_id),
  CONSTRAINT fk_psl_sheet FOREIGN KEY (sheet_id) REFERENCES pay_payroll_sheet(id)
) ENGINE=InnoDB COMMENT='工资单行';

CREATE TABLE IF NOT EXISTS pay_payroll_adjust (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  sheet_id      BIGINT NOT NULL,
  employee_id   BIGINT NOT NULL,
  adjust_type   VARCHAR(32) NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  reason        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_pa_sheet FOREIGN KEY (sheet_id) REFERENCES pay_payroll_sheet(id)
) ENGINE=InnoDB COMMENT='工资调整';

CREATE TABLE IF NOT EXISTS pay_sales_commission_rule (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  name            VARCHAR(128) NOT NULL,
  rule_json       JSON NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='销售提成规则';

CREATE TABLE IF NOT EXISTS pay_commission_calc (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  rule_id           BIGINT NOT NULL,
  employee_id       BIGINT NOT NULL,
  period            VARCHAR(16) NOT NULL,
  base_amount       DECIMAL(18,4) NOT NULL DEFAULT 0,
  commission_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  source_doc_refs   JSON NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_cc_rule FOREIGN KEY (rule_id) REFERENCES pay_sales_commission_rule(id)
) ENGINE=InnoDB COMMENT='提成计算结果';


-- ===== FILE: schema\05_hr.sql =====
USE erp_factory;

CREATE TABLE IF NOT EXISTS hr_onboard (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  onboard_date  DATE NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  role_ids_json JSON NULL COMMENT '拟赋角色列表',
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_onboard_emp FOREIGN KEY (employee_id) REFERENCES hr_employee(id)
) ENGINE=InnoDB COMMENT='入职登记';

CREATE TABLE IF NOT EXISTS hr_offboard (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id         BIGINT NOT NULL,
  offboard_date       DATE NOT NULL,
  reason              VARCHAR(512) NULL,
  revoke_permission   TINYINT(1) NOT NULL DEFAULT 1,
  status              VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_by          BIGINT NULL,
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_offboard_emp FOREIGN KEY (employee_id) REFERENCES hr_employee(id)
) ENGINE=InnoDB COMMENT='离职登记（收回权限）';

CREATE TABLE IF NOT EXISTS hr_shift (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  start_time    TIME NOT NULL,
  end_time      TIME NOT NULL,
  workshop_id   BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_shift_code (code)
) ENGINE=InnoDB COMMENT='班次';

CREATE TABLE IF NOT EXISTS hr_attendance_rule (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(128) NOT NULL,
  shift_id      BIGINT NULL,
  late_minutes  INT NOT NULL DEFAULT 0,
  early_minutes INT NOT NULL DEFAULT 0,
  rule_json     JSON NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
) ENGINE=InnoDB COMMENT='考勤规则';

CREATE TABLE IF NOT EXISTS hr_attendance_record (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  biz_date      DATE NOT NULL,
  check_in_at   DATETIME(3) NULL,
  check_out_at  DATETIME(3) NULL,
  shift_id      BIGINT NULL,
  source        VARCHAR(32) NULL,
  UNIQUE KEY uk_att (employee_id, biz_date),
  CONSTRAINT fk_att_emp FOREIGN KEY (employee_id) REFERENCES hr_employee(id)
) ENGINE=InnoDB COMMENT='考勤打卡';

CREATE TABLE IF NOT EXISTS hr_leave_request (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  employee_id   BIGINT NOT NULL,
  leave_type    VARCHAR(32) NOT NULL,
  start_at      DATETIME(3) NOT NULL,
  end_at        DATETIME(3) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_leave_no (doc_no)
) ENGINE=InnoDB COMMENT='请假单';

CREATE TABLE IF NOT EXISTS hr_overtime_patch (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  employee_id   BIGINT NOT NULL,
  biz_type      VARCHAR(16) NOT NULL COMMENT 'overtime/patch',
  biz_date      DATE NOT NULL,
  minutes       INT NOT NULL DEFAULT 0,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ot_no (doc_no)
) ENGINE=InnoDB COMMENT='加班补卡';

CREATE TABLE IF NOT EXISTS hr_attendance_month_stat (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  year          SMALLINT NOT NULL,
  month         TINYINT NOT NULL,
  work_days     DECIMAL(8,2) NOT NULL DEFAULT 0,
  late_times    INT NOT NULL DEFAULT 0,
  ot_hours      DECIMAL(8,2) NOT NULL DEFAULT 0,
  leave_days    DECIMAL(8,2) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_ams (employee_id, year, month)
) ENGINE=InnoDB COMMENT='考勤月度统计';

CREATE TABLE IF NOT EXISTS hr_performance_scheme (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(128) NOT NULL,
  scheme_json   JSON NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='绩效方案';

CREATE TABLE IF NOT EXISTS hr_performance_result (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  scheme_id     BIGINT NOT NULL,
  employee_id   BIGINT NOT NULL,
  period        VARCHAR(16) NOT NULL,
  score         DECIMAL(8,2) NULL,
  amount        DECIMAL(18,4) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='绩效结果';

CREATE TABLE IF NOT EXISTS hr_attendance_perf_summary (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id       BIGINT NOT NULL,
  period            VARCHAR(16) NOT NULL,
  attendance_score  DECIMAL(8,2) NULL,
  perf_score        DECIMAL(8,2) NULL,
  summary_json      JSON NULL,
  UNIQUE KEY uk_aps (employee_id, period)
) ENGINE=InnoDB COMMENT='考勤绩效汇总';

CREATE TABLE IF NOT EXISTS hr_visit_record (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  customer_id   BIGINT NULL,
  visit_at      DATETIME(3) NOT NULL,
  content       VARCHAR(1024) NULL,
  location      VARCHAR(256) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='外访明细';

CREATE TABLE IF NOT EXISTS hr_memo (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  owner_user_id BIGINT NOT NULL,
  title         VARCHAR(256) NOT NULL,
  content       TEXT NULL,
  biz_date      DATE NULL,
  scope_type    VARCHAR(16) NOT NULL DEFAULT 'hr' COMMENT 'hr/system',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='备忘录';

CREATE TABLE IF NOT EXISTS hr_employee_journal (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  biz_date      DATE NOT NULL,
  content       TEXT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_ej (employee_id, biz_date)
) ENGINE=InnoDB COMMENT='员工日志';

CREATE TABLE IF NOT EXISTS hr_personnel_transfer (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  employee_id   BIGINT NOT NULL,
  from_dept_id  BIGINT NULL,
  to_dept_id    BIGINT NULL,
  from_post     VARCHAR(64) NULL,
  to_post       VARCHAR(64) NULL,
  effective_date DATE NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='人事调动';


-- ===== FILE: schema\06_crm_sales_purchase.sql =====
USE erp_factory;

-- 客户
CREATE TABLE IF NOT EXISTS crm_customer (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(256) NOT NULL,
  level_code      VARCHAR(32) NULL,
  source          VARCHAR(64) NULL,
  owner_user_id   BIGINT NULL,
  protect_until   DATE NULL,
  is_public_sea   TINYINT(1) NOT NULL DEFAULT 0,
  is_hidden       TINYINT(1) NOT NULL DEFAULT 0,
  is_locked       TINYINT(1) NOT NULL DEFAULT 0,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  contact_json    JSON NULL,
  address         VARCHAR(512) NULL,
  created_by      BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted      TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_customer_code (code)
) ENGINE=InnoDB COMMENT='客户档案';

CREATE TABLE IF NOT EXISTS crm_opportunity (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  stage           VARCHAR(32) NULL,
  amount          DECIMAL(18,4) NULL,
  expected_date   DATE NULL,
  owner_user_id   BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'open',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_opp_cust FOREIGN KEY (customer_id) REFERENCES crm_customer(id)
) ENGINE=InnoDB COMMENT='商机';

CREATE TABLE IF NOT EXISTS crm_follow_up (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  opportunity_id  BIGINT NULL,
  user_id         BIGINT NOT NULL,
  follow_at       DATETIME(3) NOT NULL,
  content         VARCHAR(1024) NULL,
  next_remind_at  DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='客户跟进';

CREATE TABLE IF NOT EXISTS crm_lead_assign (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id   BIGINT NOT NULL,
  from_user_id  BIGINT NULL,
  to_user_id    BIGINT NOT NULL,
  assigned_at   DATETIME(3) NOT NULL,
  lock_flag     TINYINT(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB COMMENT='资源/线索分配';

CREATE TABLE IF NOT EXISTS crm_lead_protect_rule (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  name              VARCHAR(128) NOT NULL,
  protect_days      INT NOT NULL DEFAULT 30,
  release_rule_json JSON NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'active'
) ENGINE=InnoDB COMMENT='保护机制规则';

CREATE TABLE IF NOT EXISTS crm_lead_release_log (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  released_at     DATETIME(3) NOT NULL,
  reason          VARCHAR(512) NULL,
  to_public_sea   TINYINT(1) NOT NULL DEFAULT 1
) ENGINE=InnoDB COMMENT='释放记录';

CREATE TABLE IF NOT EXISTS crm_task_reminder (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  ref_type      VARCHAR(64) NULL,
  ref_id        BIGINT NULL,
  remind_at     DATETIME(3) NOT NULL,
  content       VARCHAR(512) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'pending'
) ENGINE=InnoDB COMMENT='任务提醒';

CREATE TABLE IF NOT EXISTS crm_customer_import_batch (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  file_name       VARCHAR(256) NULL,
  imported_at     DATETIME(3) NOT NULL,
  success_count   INT NOT NULL DEFAULT 0,
  fail_count      INT NOT NULL DEFAULT 0
) ENGINE=InnoDB COMMENT='客户导入批次';

-- 销售
CREATE TABLE IF NOT EXISTS sl_contract (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  customer_id   BIGINT NOT NULL,
  amount        DECIMAL(18,4) NULL,
  start_date    DATE NULL,
  end_date      DATE NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  file_id       BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_contract_no (doc_no)
) ENGINE=InnoDB COMMENT='合同';

CREATE TABLE IF NOT EXISTS sl_price_lock (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  lock_price      DECIMAL(18,4) NOT NULL,
  effective_from  DATE NOT NULL,
  effective_to    DATE NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='销售锁价';

CREATE TABLE IF NOT EXISTS sl_inquiry (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  customer_id     BIGINT NOT NULL,
  owner_user_id   BIGINT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  source          VARCHAR(16) NOT NULL DEFAULT 'sales' COMMENT 'self/sales',
  expire_at       DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_inquiry_no (doc_no)
) ENGINE=InnoDB COMMENT='询价单头';

CREATE TABLE IF NOT EXISTS sl_inquiry_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  inquiry_id    BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  quote_price   DECIMAL(18,4) NULL,
  cost_ref      DECIMAL(18,4) NULL,
  remark        VARCHAR(256) NULL,
  CONSTRAINT fk_il_inq FOREIGN KEY (inquiry_id) REFERENCES sl_inquiry(id)
) ENGINE=InnoDB COMMENT='询价行';

CREATE TABLE IF NOT EXISTS sl_quote_history (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  price         DECIMAL(18,4) NOT NULL,
  quoted_at     DATETIME(3) NOT NULL,
  inquiry_id    BIGINT NULL,
  order_id      BIGINT NULL
) ENGINE=InnoDB COMMENT='历史报价';

CREATE TABLE IF NOT EXISTS sl_sales_order (
  id                      BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no                  VARCHAR(64) NOT NULL,
  customer_id             BIGINT NOT NULL,
  owner_user_id           BIGINT NULL,
  status                  VARCHAR(16) NOT NULL DEFAULT 'draft',
  source                  VARCHAR(16) NOT NULL DEFAULT 'manual' COMMENT 'manual/self/rebuy',
  contract_id             BIGINT NULL,
  price_lock_id           BIGINT NULL,
  reorder_from_id         BIGINT NULL,
  need_delivery_approval  TINYINT(1) NOT NULL DEFAULT 1,
  org_id                  BIGINT NULL,
  remark                  VARCHAR(512) NULL,
  created_by              BIGINT NULL,
  created_at              DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at              DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_deleted              TINYINT(1) NOT NULL DEFAULT 0,
  version                 INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_so_no (doc_no)
) ENGINE=InnoDB COMMENT='销售订单头';

CREATE TABLE IF NOT EXISTS sl_sales_order_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id        BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  qty             DECIMAL(18,4) NOT NULL,
  weight          DECIMAL(18,4) NULL,
  price           DECIMAL(18,4) NOT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  delivered_qty   DECIMAL(18,4) NOT NULL DEFAULT 0,
  CONSTRAINT fk_sol_so FOREIGN KEY (order_id) REFERENCES sl_sales_order(id)
) ENGINE=InnoDB COMMENT='销售订单行';

CREATE TABLE IF NOT EXISTS sl_pre_shipment (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  order_id      BIGINT NOT NULL,
  plan_ship_date DATE NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  reserved      TINYINT(1) NOT NULL DEFAULT 0,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pre_ship_no (doc_no)
) ENGINE=InnoDB COMMENT='预发货';

CREATE TABLE IF NOT EXISTS sl_pre_shipment_line (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  pre_shipment_id   BIGINT NOT NULL,
  order_line_id     BIGINT NULL,
  product_id        BIGINT NOT NULL,
  qty               DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_psl_hdr FOREIGN KEY (pre_shipment_id) REFERENCES sl_pre_shipment(id)
) ENGINE=InnoDB COMMENT='预发货行';

CREATE TABLE IF NOT EXISTS sl_delivery_approval (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  order_id          BIGINT NOT NULL,
  pre_shipment_id   BIGINT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  warehouse_id      BIGINT NULL,
  shipped_at        DATETIME(3) NULL,
  logistics_no      VARCHAR(64) NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_da_no (doc_no)
) ENGINE=InnoDB COMMENT='发货审批/发货单';

CREATE TABLE IF NOT EXISTS sl_delivery_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  delivery_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  weight        DECIMAL(18,4) NULL,
  batch_no      VARCHAR(64) NULL,
  box_code_id   BIGINT NULL,
  CONSTRAINT fk_dl_hdr FOREIGN KEY (delivery_id) REFERENCES sl_delivery_approval(id)
) ENGINE=InnoDB COMMENT='发货行';

CREATE TABLE IF NOT EXISTS sl_sales_bom (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id      BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  version_no    VARCHAR(32) NOT NULL DEFAULT 'V1',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='订单级BOM';

CREATE TABLE IF NOT EXISTS sl_sales_bom_line (
  id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
  sales_bom_id          BIGINT NOT NULL,
  component_product_id  BIGINT NOT NULL,
  qty                   DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_sbl_hdr FOREIGN KEY (sales_bom_id) REFERENCES sl_sales_bom(id)
) ENGINE=InnoDB COMMENT='销售BOM行';

CREATE TABLE IF NOT EXISTS sl_cost_budget (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id        BIGINT NOT NULL,
  material_cost   DECIMAL(18,4) NOT NULL DEFAULT 0,
  labor_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  other_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  margin          DECIMAL(18,4) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cb_order (order_id)
) ENGINE=InnoDB COMMENT='订单成本预算';

CREATE TABLE IF NOT EXISTS sl_quote_calculator_result (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  inquiry_id    BIGINT NULL,
  order_id      BIGINT NULL,
  input_json    JSON NULL,
  result_json   JSON NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='报价试算结果';

CREATE TABLE IF NOT EXISTS sl_sales_rank_config (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(128) NOT NULL,
  metric        VARCHAR(64) NOT NULL,
  period_type   VARCHAR(16) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active'
) ENGINE=InnoDB COMMENT='排行榜配置';

CREATE TABLE IF NOT EXISTS sl_order_change_log (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  order_id      BIGINT NOT NULL,
  change_json   JSON NOT NULL,
  changed_by    BIGINT NULL,
  changed_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='修改订单记录';

-- 采购
CREATE TABLE IF NOT EXISTS pur_supplier (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(256) NOT NULL,
  rating        VARCHAR(16) NULL,
  contact_json  JSON NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  is_deleted    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_supplier_code (code)
) ENGINE=InnoDB COMMENT='供应商';

CREATE TABLE IF NOT EXISTS pur_purchase_request (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  applicant_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  need_date     DATE NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pr_no (doc_no)
) ENGINE=InnoDB COMMENT='采购申请头';

CREATE TABLE IF NOT EXISTS pur_purchase_request_line (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  request_id          BIGINT NOT NULL,
  product_id          BIGINT NOT NULL,
  qty                 DECIMAL(18,4) NOT NULL,
  unit_id             BIGINT NULL,
  suggest_supplier_id BIGINT NULL,
  CONSTRAINT fk_prl_hdr FOREIGN KEY (request_id) REFERENCES pur_purchase_request(id)
) ENGINE=InnoDB COMMENT='采购申请行';

CREATE TABLE IF NOT EXISTS pur_purchase_plan (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  plan_date     DATE NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pp_no (doc_no)
) ENGINE=InnoDB COMMENT='采购计划单头';

CREATE TABLE IF NOT EXISTS pur_purchase_plan_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  plan_id         BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  qty             DECIMAL(18,4) NOT NULL,
  supplier_id     BIGINT NULL,
  request_line_id BIGINT NULL,
  CONSTRAINT fk_ppl_hdr FOREIGN KEY (plan_id) REFERENCES pur_purchase_plan(id)
) ENGINE=InnoDB COMMENT='采购计划行';

CREATE TABLE IF NOT EXISTS pur_purchase_inbound (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  supplier_id   BIGINT NOT NULL,
  warehouse_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  biz_date      DATE NOT NULL,
  plan_id       BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pi_no (doc_no)
) ENGINE=InnoDB COMMENT='采购入库头';

CREATE TABLE IF NOT EXISTS pur_purchase_inbound_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  inbound_id    BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  price         DECIMAL(18,4) NULL,
  amount        DECIMAL(18,4) NULL,
  batch_no      VARCHAR(64) NULL,
  CONSTRAINT fk_pil_hdr FOREIGN KEY (inbound_id) REFERENCES pur_purchase_inbound(id)
) ENGINE=InnoDB COMMENT='采购入库行';

CREATE TABLE IF NOT EXISTS pur_incoming_qc (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  inbound_id    BIGINT NULL,
  product_id    BIGINT NOT NULL,
  qty_check     DECIMAL(18,4) NOT NULL,
  qty_pass      DECIMAL(18,4) NOT NULL DEFAULT 0,
  qty_fail      DECIMAL(18,4) NOT NULL DEFAULT 0,
  result        VARCHAR(16) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_iqc_no (doc_no)
) ENGINE=InnoDB COMMENT='来料质检';

CREATE TABLE IF NOT EXISTS pur_purchase_return (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  supplier_id   BIGINT NOT NULL,
  inbound_id    BIGINT NULL,
  warehouse_id  BIGINT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  reason        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_prt_no (doc_no)
) ENGINE=InnoDB COMMENT='采购退货';

CREATE TABLE IF NOT EXISTS pur_purchase_return_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  return_id     BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  qty           DECIMAL(18,4) NOT NULL,
  amount        DECIMAL(18,4) NULL,
  CONSTRAINT fk_prtl_hdr FOREIGN KEY (return_id) REFERENCES pur_purchase_return(id)
) ENGINE=InnoDB COMMENT='采购退货行';

CREATE TABLE IF NOT EXISTS pur_supplier_price_history (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  supplier_id   BIGINT NOT NULL,
  product_id    BIGINT NOT NULL,
  price         DECIMAL(18,4) NOT NULL,
  biz_date      DATE NOT NULL,
  source_doc_id BIGINT NULL,
  KEY idx_sph (supplier_id, product_id, biz_date)
) ENGINE=InnoDB COMMENT='历史采购价';

CREATE TABLE IF NOT EXISTS pur_purchase_task (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  assignee_id   BIGINT NULL,
  product_id    BIGINT NULL,
  qty           DECIMAL(18,4) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  due_date      DATE NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ptask_no (doc_no)
) ENGINE=InnoDB COMMENT='采购任务';


-- ===== FILE: schema\07_finance_asset.sql =====
USE erp_factory;

-- 固定资产
CREATE TABLE IF NOT EXISTS ast_fixed_asset_category (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  parent_id     BIGINT NULL,
  UNIQUE KEY uk_fac_code (code)
) ENGINE=InnoDB COMMENT='固定资产类别';

CREATE TABLE IF NOT EXISTS ast_fixed_asset (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(256) NOT NULL,
  category_id     BIGINT NULL,
  dept_id         BIGINT NULL,
  location_text   VARCHAR(256) NULL,
  original_value  DECIMAL(18,4) NULL,
  net_value       DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  purchase_date   DATE NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_fa_code (code)
) ENGINE=InnoDB COMMENT='固定资产卡片';

CREATE TABLE IF NOT EXISTS ast_asset_transfer (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  asset_id        BIGINT NOT NULL,
  from_dept_id    BIGINT NULL,
  to_dept_id      BIGINT NULL,
  from_location   VARCHAR(256) NULL,
  to_location     VARCHAR(256) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  transferred_at  DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_at_no (doc_no)
) ENGINE=InnoDB COMMENT='固定资产内部转移';

-- 财务
CREATE TABLE IF NOT EXISTS fin_account_subject (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  parent_id     BIGINT NULL,
  subject_type  VARCHAR(32) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_subject_code (code)
) ENGINE=InnoDB COMMENT='会计科目/账目';

CREATE TABLE IF NOT EXISTS fin_fund_account (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  currency      VARCHAR(8) NOT NULL DEFAULT 'CNY',
  balance       DECIMAL(18,4) NOT NULL DEFAULT 0,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_fund_code (code)
) ENGINE=InnoDB COMMENT='资金账户';

CREATE TABLE IF NOT EXISTS fin_fund_transfer (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  from_account_id   BIGINT NOT NULL,
  to_account_id     BIGINT NOT NULL,
  amount            DECIMAL(18,4) NOT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ft_no (doc_no)
) ENGINE=InnoDB COMMENT='资金调拨';

CREATE TABLE IF NOT EXISTS fin_ledger_entry (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  account_id        BIGINT NULL,
  subject_id        BIGINT NULL,
  direction         VARCHAR(8) NOT NULL COMMENT 'in/out',
  amount            DECIMAL(18,4) NOT NULL,
  biz_date          DATE NOT NULL,
  counterparty      VARCHAR(256) NULL,
  source_doc_type   VARCHAR(64) NULL,
  source_doc_id     BIGINT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_le_no (doc_no)
) ENGINE=InnoDB COMMENT='交易流水账';

CREATE TABLE IF NOT EXISTS fin_income_expense_detail (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  entry_id      BIGINT NOT NULL,
  category      VARCHAR(64) NULL,
  amount        DECIMAL(18,4) NOT NULL,
  remark        VARCHAR(512) NULL,
  CONSTRAINT fk_ied_entry FOREIGN KEY (entry_id) REFERENCES fin_ledger_entry(id)
) ENGINE=InnoDB COMMENT='收入支出明细';

CREATE TABLE IF NOT EXISTS fin_voucher (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  period        VARCHAR(16) NULL,
  biz_date      DATE NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  summary       VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_voucher_no (doc_no)
) ENGINE=InnoDB COMMENT='凭证头';

CREATE TABLE IF NOT EXISTS fin_voucher_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  voucher_id    BIGINT NOT NULL,
  subject_id    BIGINT NOT NULL,
  debit         DECIMAL(18,4) NOT NULL DEFAULT 0,
  credit        DECIMAL(18,4) NOT NULL DEFAULT 0,
  remark        VARCHAR(256) NULL,
  CONSTRAINT fk_vl_hdr FOREIGN KEY (voucher_id) REFERENCES fin_voucher(id)
) ENGINE=InnoDB COMMENT='凭证行';

CREATE TABLE IF NOT EXISTS fin_invoice (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  invoice_no      VARCHAR(64) NOT NULL,
  direction       VARCHAR(8) NOT NULL COMMENT 'in/out',
  counterparty_id BIGINT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  tax             DECIMAL(18,4) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  biz_date        DATE NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_invoice_no (invoice_no)
) ENGINE=InnoDB COMMENT='发票';

CREATE TABLE IF NOT EXISTS fin_receipt_writeoff (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  customer_id   BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  received_at   DATETIME(3) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rw_no (doc_no)
) ENGINE=InnoDB COMMENT='收款核单';

CREATE TABLE IF NOT EXISTS fin_receipt_writeoff_line (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  writeoff_id     BIGINT NOT NULL,
  sales_order_id  BIGINT NOT NULL,
  amount          DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_rwl_hdr FOREIGN KEY (writeoff_id) REFERENCES fin_receipt_writeoff(id)
) ENGINE=InnoDB COMMENT='核销行';

CREATE TABLE IF NOT EXISTS fin_payment_recognition (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no            VARCHAR(64) NOT NULL,
  customer_id       BIGINT NOT NULL,
  amount            DECIMAL(18,4) NOT NULL,
  fund_account_id   BIGINT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pr_no (doc_no)
) ENGINE=InnoDB COMMENT='销售认款';

CREATE TABLE IF NOT EXISTS fin_prepay_prepaid (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  party_type    VARCHAR(16) NOT NULL COMMENT 'customer/supplier',
  party_id      BIGINT NOT NULL,
  direction     VARCHAR(8) NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  balance       DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pp_no (doc_no)
) ENGINE=InnoDB COMMENT='预收预付';

CREATE TABLE IF NOT EXISTS fin_fx_settlement (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  currency      VARCHAR(8) NOT NULL,
  amount_fx     DECIMAL(18,4) NOT NULL,
  rate          DECIMAL(18,8) NOT NULL,
  amount_local  DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_fx_no (doc_no)
) ENGINE=InnoDB COMMENT='外币结汇';

CREATE TABLE IF NOT EXISTS fin_cost_allocation (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  source_amount   DECIMAL(18,4) NOT NULL,
  alloc_json      JSON NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  revoked_from_id BIGINT NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ca_no (doc_no)
) ENGINE=InnoDB COMMENT='费用分摊';

CREATE TABLE IF NOT EXISTS fin_receipt_alert (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_id   BIGINT NOT NULL,
  order_id      BIGINT NULL,
  due_date      DATE NULL,
  overdue_days  INT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='收款预警';

CREATE TABLE IF NOT EXISTS fin_cashier_reconcile (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  fund_account_id BIGINT NOT NULL,
  biz_date        DATE NOT NULL,
  book_balance    DECIMAL(18,4) NOT NULL,
  actual_balance  DECIMAL(18,4) NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cr_no (doc_no)
) ENGINE=InnoDB COMMENT='出纳对账';

CREATE TABLE IF NOT EXISTS fin_cost_accounting (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no          VARCHAR(64) NOT NULL,
  period          VARCHAR(16) NOT NULL,
  task_id         BIGINT NULL,
  product_id      BIGINT NULL,
  material_cost   DECIMAL(18,4) NOT NULL DEFAULT 0,
  labor_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  overhead        DECIMAL(18,4) NOT NULL DEFAULT 0,
  total_cost      DECIMAL(18,4) NOT NULL DEFAULT 0,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cost_acc_no (doc_no)
) ENGINE=InnoDB COMMENT='成本核算单';

CREATE TABLE IF NOT EXISTS fin_cost_trace_line (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  cost_id       BIGINT NOT NULL,
  source_type   VARCHAR(32) NOT NULL COMMENT 'report/requisition/stock',
  source_id     BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  CONSTRAINT fk_ctl_cost FOREIGN KEY (cost_id) REFERENCES fin_cost_accounting(id)
) ENGINE=InnoDB COMMENT='成本明细溯源';

CREATE TABLE IF NOT EXISTS fin_contract_profit (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  contract_id   BIGINT NOT NULL,
  revenue       DECIMAL(18,4) NOT NULL DEFAULT 0,
  cost          DECIMAL(18,4) NOT NULL DEFAULT 0,
  profit        DECIMAL(18,4) NOT NULL DEFAULT 0,
  period        VARCHAR(16) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='合同利润';

CREATE TABLE IF NOT EXISTS fin_sales_return_finance (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  order_id      BIGINT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_srf_no (doc_no)
) ENGINE=InnoDB COMMENT='销售退货退单财务';

CREATE TABLE IF NOT EXISTS fin_arap_adjust (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  party_type    VARCHAR(16) NOT NULL,
  party_id      BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  direction     VARCHAR(8) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_arap_no (doc_no)
) ENGINE=InnoDB COMMENT='往来调整单';

CREATE TABLE IF NOT EXISTS fin_month_close (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  year          SMALLINT NOT NULL,
  month         TINYINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'open',
  closed_at     DATETIME(3) NULL,
  closed_by     BIGINT NULL,
  UNIQUE KEY uk_month_close (year, month)
) ENGINE=InnoDB COMMENT='月度结转';

CREATE TABLE IF NOT EXISTS fin_miniprogram_bill (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  bill_no       VARCHAR(64) NOT NULL,
  channel       VARCHAR(32) NULL,
  amount        DECIMAL(18,4) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'unpaid',
  order_id      BIGINT NULL,
  paid_at       DATETIME(3) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_mp_bill (bill_no)
) ENGINE=InnoDB COMMENT='小程序账单';


-- ===== FILE: schema\08_approval_system_report.sql =====
USE erp_factory;

-- 审批
CREATE TABLE IF NOT EXISTS appr_flow (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  doc_type      VARCHAR(64) NOT NULL,
  is_enabled    TINYINT(1) NOT NULL DEFAULT 1,
  version_no    INT NOT NULL DEFAULT 1,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_flow_code_ver (code, version_no)
) ENGINE=InnoDB COMMENT='审批流程定义';

CREATE TABLE IF NOT EXISTS appr_node (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  flow_id         BIGINT NOT NULL,
  seq_no          INT NOT NULL,
  node_name       VARCHAR(128) NOT NULL,
  approver_type   VARCHAR(16) NOT NULL COMMENT 'role/user',
  approver_ref    BIGINT NOT NULL COMMENT 'role_id或user_id',
  can_reject      TINYINT(1) NOT NULL DEFAULT 1,
  CONSTRAINT fk_node_flow FOREIGN KEY (flow_id) REFERENCES appr_flow(id)
) ENGINE=InnoDB COMMENT='审批节点';

CREATE TABLE IF NOT EXISTS appr_task (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  flow_id           BIGINT NULL,
  node_id           BIGINT NULL,
  doc_type          VARCHAR(64) NOT NULL,
  doc_id            BIGINT NOT NULL,
  assignee_user_id  BIGINT NOT NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/approved/rejected',
  acted_at          DATETIME(3) NULL,
  comment           VARCHAR(512) NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_appr_assignee (assignee_user_id, status),
  KEY idx_appr_doc (doc_type, doc_id)
) ENGINE=InnoDB COMMENT='审批任务';

CREATE TABLE IF NOT EXISTS appr_expense_request (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  applicant_id  BIGINT NOT NULL,
  amount        DECIMAL(18,4) NOT NULL,
  category      VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  remark        VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_exp_no (doc_no)
) ENGINE=InnoDB COMMENT='费用申请';

CREATE TABLE IF NOT EXISTS appr_affair_request (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_no        VARCHAR(64) NOT NULL,
  applicant_id  BIGINT NOT NULL,
  content       VARCHAR(1024) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_aff_no (doc_no)
) ENGINE=InnoDB COMMENT='事务申请';

-- 系统管理
CREATE TABLE IF NOT EXISTS sys_org_setting (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  org_id        BIGINT NULL,
  setting_key   VARCHAR(128) NOT NULL,
  value_json    JSON NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_org_setting (org_id, setting_key)
) ENGINE=InnoDB COMMENT='基础参数';

CREATE TABLE IF NOT EXISTS sys_production_setting (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  setting_key   VARCHAR(128) NOT NULL,
  value_json    JSON NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_prod_setting (setting_key)
) ENGINE=InnoDB COMMENT='生产设置';

CREATE TABLE IF NOT EXISTS sys_sales_setting (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  setting_key   VARCHAR(128) NOT NULL,
  value_json    JSON NULL,
  updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_sales_setting (setting_key)
) ENGINE=InnoDB COMMENT='销售设置';

CREATE TABLE IF NOT EXISTS sys_print_template (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  doc_type      VARCHAR(64) NOT NULL,
  template_body MEDIUMTEXT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_print_code (code)
) ENGINE=InnoDB COMMENT='自定义打印模板';

CREATE TABLE IF NOT EXISTS sys_table_custom (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  page_key      VARCHAR(128) NOT NULL,
  columns_json  JSON NOT NULL,
  UNIQUE KEY uk_table_custom (user_id, page_key)
) ENGINE=InnoDB COMMENT='表格自定义';

CREATE TABLE IF NOT EXISTS sys_formula (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  expression    TEXT NOT NULL,
  biz_domain    VARCHAR(64) NULL,
  UNIQUE KEY uk_formula_code (code)
) ENGINE=InnoDB COMMENT='公式设置';

CREATE TABLE IF NOT EXISTS sys_logistics_carrier (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  api_config_json JSON NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_carrier_code (code)
) ENGINE=InnoDB COMMENT='物流承运商';

CREATE TABLE IF NOT EXISTS sys_logistics_track (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  carrier_id      BIGINT NULL,
  tracking_no     VARCHAR(64) NOT NULL,
  status          VARCHAR(32) NULL,
  last_event_json JSON NULL,
  ref_doc_type    VARCHAR(64) NULL,
  ref_doc_id      BIGINT NULL,
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_track_no (tracking_no)
) ENGINE=InnoDB COMMENT='物流轨迹';

CREATE TABLE IF NOT EXISTS sys_doc_lock_rule (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_type          VARCHAR(64) NOT NULL,
  lock_when_status  VARCHAR(64) NOT NULL,
  allow_roles_json  JSON NULL,
  UNIQUE KEY uk_doc_lock (doc_type, lock_when_status)
) ENGINE=InnoDB COMMENT='单据锁定规则';

CREATE TABLE IF NOT EXISTS sys_doc_edit_rule (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_type            VARCHAR(64) NOT NULL,
  editable_statuses   VARCHAR(256) NULL,
  field_rules_json    JSON NULL,
  UNIQUE KEY uk_doc_edit (doc_type)
) ENGINE=InnoDB COMMENT='单据编辑策略';

CREATE TABLE IF NOT EXISTS sys_doc_delete_rule (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_type        VARCHAR(64) NOT NULL,
  allow_status    VARCHAR(128) NULL,
  need_approval   TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uk_doc_del (doc_type)
) ENGINE=InnoDB COMMENT='单据删除策略';

CREATE TABLE IF NOT EXISTS sys_doc_approve_switch (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  doc_type        VARCHAR(64) NOT NULL,
  need_approval   TINYINT(1) NOT NULL DEFAULT 0,
  flow_id         BIGINT NULL,
  UNIQUE KEY uk_doc_appr (doc_type)
) ENGINE=InnoDB COMMENT='单据是否需审';

CREATE TABLE IF NOT EXISTS sys_notify_rule (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  event_key       VARCHAR(128) NOT NULL,
  channel         VARCHAR(32) NULL,
  template_text   VARCHAR(1024) NULL,
  receivers_json  JSON NULL,
  UNIQUE KEY uk_notify_event (event_key)
) ENGINE=InnoDB COMMENT='通知规则';

CREATE TABLE IF NOT EXISTS sys_reminder (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  title         VARCHAR(256) NOT NULL,
  content       VARCHAR(1024) NULL,
  remind_at     DATETIME(3) NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'pending',
  ref_type      VARCHAR(64) NULL,
  ref_id        BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='事项提醒';

CREATE TABLE IF NOT EXISTS sys_announcement (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  title           VARCHAR(256) NOT NULL,
  content         TEXT NULL,
  publish_at      DATETIME(3) NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'draft',
  audience_json   JSON NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='公告';

CREATE TABLE IF NOT EXISTS sys_knowledge (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  title         VARCHAR(256) NOT NULL,
  content       MEDIUMTEXT NULL,
  category      VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='知识库';

CREATE TABLE IF NOT EXISTS sys_course (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  title         VARCHAR(256) NOT NULL,
  content       MEDIUMTEXT NULL,
  category      VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='学堂内容';

CREATE TABLE IF NOT EXISTS sys_drawing (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  code          VARCHAR(64) NOT NULL,
  name          VARCHAR(256) NOT NULL,
  file_url      VARCHAR(512) NULL,
  version_no    VARCHAR(32) NULL,
  product_id    BIGINT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_drawing_code (code)
) ENGINE=InnoDB COMMENT='图纸库';

CREATE TABLE IF NOT EXISTS sys_document_file (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(256) NOT NULL,
  file_url      VARCHAR(512) NULL,
  category      VARCHAR(64) NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='文档库';

CREATE TABLE IF NOT EXISTS sys_operation_log (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NULL,
  action        VARCHAR(64) NOT NULL,
  module        VARCHAR(128) NULL,
  ref_type      VARCHAR(64) NULL,
  ref_id        BIGINT NULL,
  detail_json   JSON NULL,
  ip            VARCHAR(64) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_oplog_user_time (user_id, created_at)
) ENGINE=InnoDB COMMENT='操作日志';

CREATE TABLE IF NOT EXISTS sys_search_config (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  page_key      VARCHAR(128) NOT NULL,
  fields_json   JSON NOT NULL,
  UNIQUE KEY uk_search_page (page_key)
) ENGINE=InnoDB COMMENT='多条件检索配置';

CREATE TABLE IF NOT EXISTS sys_finance_audit_control (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  control_key   VARCHAR(128) NOT NULL,
  enabled       TINYINT(1) NOT NULL DEFAULT 1,
  rule_json     JSON NULL,
  UNIQUE KEY uk_fac (control_key)
) ENGINE=InnoDB COMMENT='财审管控开关';

CREATE TABLE IF NOT EXISTS sys_batch_price_job (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  status          VARCHAR(16) NOT NULL DEFAULT 'pending',
  filter_json     JSON NULL,
  new_price_rule  JSON NULL,
  executed_at     DATETIME(3) NULL,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='批量改价任务';

CREATE TABLE IF NOT EXISTS sys_batch_payroll_job (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  period_year   SMALLINT NOT NULL,
  period_month  TINYINT NOT NULL,
  status        VARCHAR(16) NOT NULL DEFAULT 'pending',
  executed_at   DATETIME(3) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='批量核算工资任务';

-- 报表配置
CREATE TABLE IF NOT EXISTS rpt_report_definition (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  code              VARCHAR(64) NOT NULL,
  name              VARCHAR(128) NOT NULL,
  report_type       VARCHAR(64) NULL,
  query_config_json JSON NULL,
  status            VARCHAR(16) NOT NULL DEFAULT 'active',
  UNIQUE KEY uk_rpt_code (code)
) ENGINE=InnoDB COMMENT='报表定义';

CREATE TABLE IF NOT EXISTS rpt_dashboard_widget (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  dashboard_key   VARCHAR(32) NOT NULL COMMENT 'boss/production/live',
  title           VARCHAR(128) NOT NULL,
  metric_key      VARCHAR(64) NULL,
  layout_json     JSON NULL,
  refresh_sec     INT NOT NULL DEFAULT 60,
  status          VARCHAR(16) NOT NULL DEFAULT 'active'
) ENGINE=InnoDB COMMENT='驾驶舱/看板组件';

CREATE TABLE IF NOT EXISTS rpt_report_snapshot (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  report_code   VARCHAR(64) NOT NULL,
  biz_date      DATE NOT NULL,
  payload_json  JSON NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_rpt_snap (report_code, biz_date)
) ENGINE=InnoDB COMMENT='报表快照';

SET FOREIGN_KEY_CHECKS = 1;


-- ===== FILE: seed\01_iam_seed.sql =====
-- 权限种子数据：预置角色 / 管理员分组 / 登录策略 / 字段策略示例 / 权限码样例
USE erp_factory;

-- 组织占位（若未初始化）
INSERT INTO sys_organization (id, code, name, status)
SELECT 1, 'ORG001', '加工厂演示组织', 'active'
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM sys_organization WHERE id = 1);

INSERT INTO iam_login_policy (
  org_id, max_fail_count, lock_minutes, session_ttl_min,
  password_min_len, password_require_letter, password_require_digit,
  password_require_special, password_history, force_change_days, single_session
) SELECT NULL, 5, 30, 120, 8, 1, 1, 0, 5, 90, 0
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM iam_login_policy LIMIT 1);

-- 管理员分组
INSERT INTO iam_admin_group (code, name, remark, sort_no, status) VALUES
('g_platform', '平台管理组', '系统管理员等高权限后台账号', 10, 'active'),
('g_biz', '经营决策组', '老板驾驶舱与经营只读', 20, 'active'),
('g_ops', '业务作业组', '销售/采购等业务内勤与外勤', 30, 'active'),
('g_plant', '仓储生产组', '仓管/计划/车间/质检', 40, 'active'),
('g_line', '一线作业组', '计件工/固定工（前台为主）', 50, 'active'),
('g_func', '职能管理组', '人事/薪资/财务', 60, 'active')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 预置角色（文档 7.3）
INSERT INTO iam_role (code, name, data_scope_type, is_system, status, remark) VALUES
('sys_admin', '系统管理员', 'all', 1, 'active', '系统管理全模块 + 权限分配'),
('boss', '老板', 'all', 1, 'active', '统计报表/财务只读/审批'),
('sales', '销售员', 'self', 1, 'active', '销售+客户'),
('purchase', '采购员', 'all', 1, 'active', '采购+采购类审批'),
('warehouse', '仓管员', 'warehouse', 1, 'active', '库存按仓'),
('planner', '生产计划', 'all', 1, 'active', '任务/派工/工艺/MRP/BOM'),
('foreman', '车间主任', 'workshop', 1, 'active', '工作台/派工/进度/质检'),
('piece', '计件工', 'self', 1, 'active', '报工/领料/本人工资只读'),
('fixed', '固定工', 'team', 1, 'active', '授权工序报工/收货质检'),
('qc', '质检员', 'all', 1, 'active', '来料/入库/过程质检'),
('hr', '人事', 'all', 1, 'active', '人事+考勤审批'),
('payroll', '薪资员', 'all', 1, 'active', '工资+批量核算'),
('finance', '财务', 'all', 1, 'active', '财务+财审管控')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 分组绑定角色
INSERT INTO iam_admin_group_role (group_id, role_id)
SELECT g.id, r.id FROM iam_admin_group g JOIN iam_role r
WHERE (g.code, r.code) IN (
  ('g_platform','sys_admin'),
  ('g_biz','boss'),
  ('g_ops','sales'), ('g_ops','purchase'),
  ('g_plant','warehouse'), ('g_plant','planner'), ('g_plant','foreman'), ('g_plant','qc'),
  ('g_line','piece'), ('g_line','fixed'),
  ('g_func','hr'), ('g_func','payroll'), ('g_func','finance')
)
AND NOT EXISTS (
  SELECT 1 FROM iam_admin_group_role x WHERE x.group_id = g.id AND x.role_id = r.id
);

-- 入职模板
INSERT INTO iam_onboard_role_template (emp_type, role_id)
SELECT t.emp_type, r.id FROM (
  SELECT 'admin' emp_type, 'sys_admin' role_code UNION ALL
  SELECT 'office', 'hr' UNION ALL
  SELECT 'piece', 'piece' UNION ALL
  SELECT 'fixed', 'fixed'
) t JOIN iam_role r ON r.code = t.role_code
WHERE NOT EXISTS (
  SELECT 1 FROM iam_onboard_role_template x WHERE x.emp_type = t.emp_type AND x.role_id = r.id
);

-- 字段策略：成本隐藏（一线隐藏，财务/老板可见）
INSERT INTO iam_field_policy (role_id, field_key, field_name, visible, editable)
SELECT r.id, f.field_key, f.field_name, f.visible, 0
FROM iam_role r
JOIN (
  SELECT 'cost_price' field_key, '成本价' field_name, 0 visible, 'piece' role_code UNION ALL
  SELECT 'gross_margin', '毛利', 0, 'piece' UNION ALL
  SELECT 'other_wage', '他人工资', 0, 'piece' UNION ALL
  SELECT 'own_wage', '本人工资', 1, 'piece' UNION ALL
  SELECT 'cost_price', '成本价', 0, 'fixed' UNION ALL
  SELECT 'gross_margin', '毛利', 0, 'fixed' UNION ALL
  SELECT 'other_wage', '他人工资', 0, 'fixed' UNION ALL
  SELECT 'cost_price', '成本价', 0, 'foreman' UNION ALL
  SELECT 'gross_margin', '毛利', 0, 'foreman' UNION ALL
  SELECT 'cost_price', '成本价', 1, 'finance' UNION ALL
  SELECT 'gross_margin', '毛利', 1, 'finance' UNION ALL
  SELECT 'other_wage', '他人工资', 1, 'finance' UNION ALL
  SELECT 'cost_price', '成本价', 1, 'boss' UNION ALL
  SELECT 'gross_margin', '毛利', 1, 'boss' UNION ALL
  SELECT 'other_wage', '他人工资', 1, 'boss'
) f ON f.role_code = r.code
WHERE NOT EXISTS (
  SELECT 1 FROM iam_field_policy p WHERE p.role_id = r.id AND p.field_key = f.field_key
);

-- 权限码样例（全量可由程序按 13域×模块×动作 生成；此处植入关键样例）
INSERT INTO iam_permission (code, name, domain, module, action) VALUES
('人事管理:权限分配:查看', '查看权限分配', '人事管理', '权限分配', '查看'),
('人事管理:权限分配:编辑', '编辑权限分配', '人事管理', '权限分配', '编辑'),
('系统管理:自定义权限:查看', '查看自定义权限', '系统管理', '自定义权限', '查看'),
('系统管理:自定义权限:编辑', '编辑自定义权限', '系统管理', '自定义权限', '编辑'),
('系统管理:自定义菜单:编辑', '编辑自定义菜单', '系统管理', '自定义菜单', '编辑'),
('系统管理:登录控制:编辑', '编辑登录控制', '系统管理', '登录控制', '编辑'),
('系统管理:账户冻结:编辑', '账户冻结操作', '系统管理', '账户冻结', '编辑'),
('生产管理:扫码报工:新增', '扫码报工新增', '生产管理', '扫码报工', '新增'),
('生产管理:扫码报工:查看', '扫码报工查看', '生产管理', '扫码报工', '查看'),
('生产管理:联动式领料:新增', '联动领料新增', '生产管理', '联动式领料', '新增'),
('生产管理:生产派工:新增', '生产派工', '生产管理', '生产派工', '新增'),
('生产管理:灵活派发工单:新增', '灵活派发', '生产管理', '灵活派发工单', '新增'),
('生产管理:成本隐藏:编辑', '成本隐藏策略', '生产管理', '成本隐藏', '编辑'),
('库存管理:库存查询:查看', '库存查询', '库存管理', '库存查询', '查看'),
('库存管理:物料调拨耗用:新增', '调拨耗用', '库存管理', '物料调拨耗用', '新增'),
('工资管理:工序工资:查看', '工序工资查看', '工资管理', '工序工资', '查看'),
('工资管理:薪酬核算:编辑', '薪酬核算', '工资管理', '薪酬核算', '编辑'),
('财务管理:成本核算:查看', '成本核算查看', '财务管理', '成本核算', '查看'),
('财务管理:收款核单:编辑', '收款核单', '财务管理', '收款核单', '编辑'),
('审批管理:任务管理:审批', '审批处理', '审批管理', '任务管理', '审批'),
('销售管理:销售订单:新增', '销售订单新增', '销售管理', '销售订单', '新增'),
('采购管理:采购入库:新增', '采购入库', '采购管理', '采购入库', '新增')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 系统管理员拥有全部已植入权限码
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r CROSS JOIN iam_permission p
WHERE r.code = 'sys_admin'
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 计件工：报工/领料查看新增
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r JOIN iam_permission p
WHERE r.code = 'piece' AND p.code IN (
  '生产管理:扫码报工:新增','生产管理:扫码报工:查看','生产管理:联动式领料:新增','工资管理:工序工资:查看'
)
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 车间主任：派工相关
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r JOIN iam_permission p
WHERE r.code = 'foreman' AND p.code IN (
  '生产管理:扫码报工:查看','生产管理:生产派工:新增','生产管理:灵活派发工单:新增','生产管理:联动式领料:新增'
)
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 财务：成本可见相关
INSERT INTO iam_role_permission (role_id, permission_id)
SELECT r.id, p.id FROM iam_role r JOIN iam_permission p
WHERE r.code = 'finance' AND p.code IN (
  '财务管理:成本核算:查看','财务管理:收款核单:编辑','审批管理:任务管理:审批'
)
AND NOT EXISTS (SELECT 1 FROM iam_role_permission x WHERE x.role_id = r.id AND x.permission_id = p.id);

-- 演示管理员账号（密码占位 hash，勿用于生产）
INSERT INTO hr_employee (emp_no, name, org_id, emp_type, status)
SELECT 'E0001', '系统管理员', 1, 'office', 'active'
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM hr_employee WHERE emp_no = 'E0001');

INSERT INTO iam_user (login_name, password_hash, employee_id, user_type, status)
SELECT 'admin', '$2a$10$DEMO_ONLY_HASH_REPLACE_ME___________', e.id, 'admin', 'active'
FROM hr_employee e WHERE e.emp_no = 'E0001'
AND NOT EXISTS (SELECT 1 FROM iam_user WHERE login_name = 'admin');

UPDATE hr_employee e
JOIN iam_user u ON u.employee_id = e.id AND u.login_name = 'admin'
SET e.user_id = u.id;

INSERT INTO iam_user_role (user_id, role_id)
SELECT u.id, r.id FROM iam_user u JOIN iam_role r
WHERE u.login_name = 'admin' AND r.code = 'sys_admin'
AND NOT EXISTS (SELECT 1 FROM iam_user_role x WHERE x.user_id = u.id AND x.role_id = r.id);

INSERT INTO iam_admin_group_user (group_id, user_id)
SELECT g.id, u.id FROM iam_admin_group g JOIN iam_user u
WHERE g.code = 'g_platform' AND u.login_name = 'admin'
AND NOT EXISTS (SELECT 1 FROM iam_admin_group_user x WHERE x.group_id = g.id AND x.user_id = u.id);

-- 菜单裁剪示例：计件工仅可见部分生产/工资/人事模块
INSERT INTO iam_menu_custom (role_id, domain, module, menu_key, visible, sort_no)
SELECT r.id, m.domain, m.module, CONCAT(m.domain, ':', m.module), 1, m.sort_no
FROM iam_role r
JOIN (
  SELECT '生产管理' domain, '扫码报工' module, 10 sort_no UNION ALL
  SELECT '生产管理', '联动式领料', 20 UNION ALL
  SELECT '工资管理', '工序工资', 30 UNION ALL
  SELECT '人事管理', '考勤管理', 40 UNION ALL
  SELECT '人事管理', '请假管理', 50 UNION ALL
  SELECT '系统管理', '公告设置', 60
) m ON 1=1
WHERE r.code = 'piece'
AND NOT EXISTS (
  SELECT 1 FROM iam_menu_custom c WHERE c.role_id = r.id AND c.menu_key = CONCAT(m.domain, ':', m.module)
);


