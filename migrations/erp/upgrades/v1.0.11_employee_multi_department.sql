-- v1.0.11: 员工可归属多个部门

CREATE TABLE IF NOT EXISTS hr_employee_department (
  employee_id INTEGER NOT NULL,
  dept_id INTEGER NOT NULL,
  is_primary INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (employee_id, dept_id),
  FOREIGN KEY(employee_id) REFERENCES hr_employee(id),
  FOREIGN KEY(dept_id) REFERENCES sys_department(id)
);

CREATE INDEX IF NOT EXISTS idx_hr_employee_department_dept ON hr_employee_department(dept_id);

INSERT INTO hr_employee_department(employee_id, dept_id, is_primary)
SELECT id, dept_id, 1
FROM hr_employee
WHERE COALESCE(dept_id, 0) > 0 AND COALESCE(is_deleted, 0) = 0
ON CONFLICT DO NOTHING;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.11', 'employee multi-department membership', '22881672f19b470d19eab13655f3e5e4b34d7dbf145faf170760daf07ba7736f')
ON CONFLICT (version) DO NOTHING;
