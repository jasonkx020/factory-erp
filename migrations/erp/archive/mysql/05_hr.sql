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
