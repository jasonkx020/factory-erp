-- v1.0.13: 人事「权限分配」更名为「角色管理」——补齐新权限码并复制角色绑定

INSERT INTO iam_permission(code, name, domain, module, action)
VALUES
  ('人事管理:角色管理:查看', '查看角色管理', '人事管理', '角色管理', '查看'),
  ('人事管理:角色管理:编辑', '编辑角色管理', '人事管理', '角色管理', '编辑')
ON CONFLICT (code) DO NOTHING;

INSERT INTO iam_role_permission(role_id, permission_id)
SELECT rp.role_id, np.id
FROM iam_role_permission rp
JOIN iam_permission op ON op.id = rp.permission_id
JOIN iam_permission np ON np.code = REPLACE(op.code, ':权限分配:', ':角色管理:')
WHERE op.domain = '人事管理'
  AND op.module = '权限分配'
  AND np.domain = '人事管理'
  AND np.module = '角色管理'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.13', 'rename hr permission assignment to role management', '8e5a71612d49eb6f6b7d07854dca825602527c59ec6d34464adcbdd60f21b27e')
ON CONFLICT (version) DO NOTHING;
