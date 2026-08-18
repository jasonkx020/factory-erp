-- v1.0.10: 个人特殊角色与部门基础角色分离（iam_user_extra_role）

CREATE TABLE IF NOT EXISTS iam_user_extra_role (
  user_id INTEGER NOT NULL,
  role_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY(user_id) REFERENCES iam_user(id),
  FOREIGN KEY(role_id) REFERENCES iam_role(id)
);

-- 已有角色中，不属于当前部门基础角色的部分视为个人特殊角色
INSERT INTO iam_user_extra_role(user_id, role_id)
SELECT ur.user_id, ur.role_id
FROM iam_user_role ur
WHERE NOT EXISTS (
  SELECT 1
  FROM hr_employee e
  JOIN sys_department_role dr ON dr.dept_id = e.dept_id
  WHERE COALESCE(e.user_id, 0) = ur.user_id
    AND dr.role_id = ur.role_id
    AND COALESCE(e.is_deleted, 0) = 0
)
ON CONFLICT DO NOTHING;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.10', 'separate user extra roles from department base roles', 'fdf403b581b0c4285948d4bcb1ccc0b6b0af4c1432d9eeb630da956bd0be6523')
ON CONFLICT (version) DO NOTHING;
