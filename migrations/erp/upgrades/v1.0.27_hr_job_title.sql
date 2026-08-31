-- v1.0.27: 岗位主数据 hr_job_title；hr_employee.job_title 文本改为 job_title_id 外键

CREATE TABLE IF NOT EXISTS hr_job_title (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  emp_type TEXT NOT NULL DEFAULT '',
  sort_no INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  is_deleted INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT NOW(),
  updated_at TEXT NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS hr_job_title_code_uq ON hr_job_title(code) WHERE COALESCE(is_deleted, 0) = 0;
CREATE UNIQUE INDEX IF NOT EXISTS hr_job_title_name_uq ON hr_job_title(name) WHERE COALESCE(is_deleted, 0) = 0;

INSERT INTO hr_job_title(code, name, emp_type, sort_no) VALUES
 ('JT-SYS-ADMIN', '系统管理员', '', 1),
 ('JT-BOSS', '总经理', 'office', 2),
 ('JT-PURCHASE', '采购员', 'office', 3),
 ('JT-QC', '质检员', 'office', 4),
 ('JT-WH', '仓管员', 'warehouse', 5),
 ('JT-FOREMAN', '车间主任', 'office', 6),
 ('JT-PLANNER', '生产计划员', 'office', 7),
 ('JT-HR', '人事专员', 'office', 8),
 ('JT-PAYROLL', '薪资员', 'office', 9),
 ('JT-FINANCE', '会计', 'office', 10),
 ('JT-PEEL', '去皮工', 'piece', 20),
 ('JT-PEEL-PC', '去皮计件工', 'piece', 21),
 ('JT-CORE', '去芯工', 'piece', 22),
 ('JT-DICE', '切块工', 'piece', 23),
 ('JT-RECEIVE', '收货员', 'fixed', 30),
 ('JT-RECEIVE-FX', '收货固定工', 'fixed', 31),
 ('JT-CUT', '切断工', 'fixed', 32),
 ('JT-WASH', '清洗工', 'fixed', 33)
ON CONFLICT DO NOTHING;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'hr_employee' AND column_name = 'job_title_id'
  ) THEN
    ALTER TABLE hr_employee ADD COLUMN job_title_id INTEGER;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'hr_employee' AND column_name = 'job_title'
  ) THEN
    INSERT INTO hr_job_title(code, name, emp_type, sort_no)
    SELECT
      'JT-MIG-' || ROW_NUMBER() OVER (ORDER BY e.job_title, e.emp_type),
      e.job_title,
      COALESCE(NULLIF(e.emp_type, ''), ''),
      900 + ROW_NUMBER() OVER (ORDER BY e.job_title, e.emp_type)
    FROM hr_employee e
    WHERE COALESCE(e.job_title, '') <> ''
      AND NOT EXISTS (
        SELECT 1 FROM hr_job_title j
        WHERE j.name = e.job_title AND COALESCE(j.is_deleted, 0) = 0
      )
    GROUP BY e.job_title, e.emp_type;

    UPDATE hr_employee e
    SET job_title_id = j.id
    FROM hr_job_title j
    WHERE COALESCE(e.job_title, '') <> ''
      AND j.name = e.job_title
      AND COALESCE(j.is_deleted, 0) = 0
      AND (j.emp_type = '' OR j.emp_type = COALESCE(e.emp_type, '') OR NOT EXISTS (
        SELECT 1 FROM hr_job_title j2
        WHERE j2.name = e.job_title AND j2.emp_type = COALESCE(e.emp_type, '') AND COALESCE(j2.is_deleted, 0) = 0
      ));

    ALTER TABLE hr_employee DROP COLUMN job_title;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE constraint_name = 'hr_employee_job_title_id_fkey'
  ) THEN
    ALTER TABLE hr_employee
      ADD CONSTRAINT hr_employee_job_title_id_fkey
      FOREIGN KEY (job_title_id) REFERENCES hr_job_title(id);
  END IF;
END $$;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.27', 'hr job title master data', '47cf887d35720845d7a2fbd2bf26264bd5e7d5033ba80e0fb1ceadfdcbdbd9bd')
ON CONFLICT (version) DO NOTHING;
