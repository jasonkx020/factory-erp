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
