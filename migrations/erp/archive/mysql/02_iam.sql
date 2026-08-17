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
  emp_type      VARCHAR(16) NOT NULL DEFAULT 'office' COMMENT 'piece计件工/temp临时工/fixed固定工/office职能内勤',
  mobile        VARCHAR(32) NULL,
  badge_code    VARCHAR(64) NULL,
  id_card_no    VARCHAR(32) NULL COMMENT '身份证号',
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
  client_type   VARCHAR(32) NULL COMMENT 'admin/boss/employee/mobile (legacy web/pad/mp_* mapped)',
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

CREATE TABLE IF NOT EXISTS iam_user_oauth (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT NOT NULL,
  provider      VARCHAR(32) NOT NULL,
  open_id       VARCHAR(128) NOT NULL,
  union_id      VARCHAR(128) NULL,
  bound_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_oauth_provider_open (provider, open_id),
  UNIQUE KEY uk_oauth_user_provider (user_id, provider),
  KEY idx_oauth_user (user_id),
  CONSTRAINT fk_oauth_user FOREIGN KEY (user_id) REFERENCES iam_user(id)
) ENGINE=InnoDB COMMENT='用户第三方绑定';

-- 入职赋权模板（角色列表）
CREATE TABLE IF NOT EXISTS iam_onboard_role_template (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  emp_type      VARCHAR(16) NOT NULL COMMENT 'piece/temp/fixed/office/admin',
  role_id       BIGINT NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_onboard_tpl (emp_type, role_id),
  CONSTRAINT fk_onboard_tpl_role FOREIGN KEY (role_id) REFERENCES iam_role(id)
) ENGINE=InnoDB COMMENT='入职默认角色模板';
