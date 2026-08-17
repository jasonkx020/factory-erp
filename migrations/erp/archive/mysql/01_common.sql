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
