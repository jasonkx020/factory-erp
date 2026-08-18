-- v1.0.9: 部门基础角色（部门成员自动继承）

CREATE TABLE IF NOT EXISTS sys_department_role (
  dept_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (dept_id, role_id),
  FOREIGN KEY(dept_id) REFERENCES sys_department(id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.9', 'department base roles for members', 'b1984c4c73758fe0d5a72d126179d70e05a04d3a16eec7757da5352731e4bc8c')
ON CONFLICT (version) DO NOTHING;
