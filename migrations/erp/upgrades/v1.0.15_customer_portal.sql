-- v1.0.15: Portal 客户账号绑定 iam_user.customer_id，演示账号 cust01

ALTER TABLE iam_user ADD COLUMN IF NOT EXISTS customer_id INTEGER;

INSERT INTO iam_role(code, name, data_scope_type, is_system, remark, status)
VALUES ('customer', '客户自助', 'self', 1, 'Portal 客户自助账号', 'active')
ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, remark=EXCLUDED.remark, status='active', is_deleted=0;

INSERT INTO crm_customer(code, name, short_name, contact_name, mobile, address, level, source, status, is_public_sea, is_locked, is_hidden, settle_method, payment_days, credit_limit, remark)
SELECT 'CU-PORTAL-01', '门户演示客户', '门户客户', '王采购', '13900001901', '南宁', 'A', '门户', 'active', 0, 0, 0, '月结', 30, 50000, '客户自助演示'
WHERE NOT EXISTS (SELECT 1 FROM crm_customer WHERE code IN ('CU-DEMO-11', 'CU-PORTAL-01'));

INSERT INTO iam_user(login_name, password_hash, employee_id, customer_id, user_type, status, is_deleted)
SELECT 'cust01', '$2a$10$ZxLeZ1b51sNokCeBa.g24On0pDDLD2hL8xP9g74fa/k1hTxxT7V0.', NULL, c.id, 'customer', 'active', 0
FROM crm_customer c
WHERE c.code IN ('CU-DEMO-11', 'CU-PORTAL-01')
ORDER BY CASE WHEN c.code='CU-DEMO-11' THEN 0 ELSE 1 END
LIMIT 1
ON CONFLICT (login_name) DO UPDATE SET
  password_hash=EXCLUDED.password_hash,
  customer_id=EXCLUDED.customer_id,
  user_type='customer',
  status='active',
  is_deleted=0,
  employee_id=NULL;

INSERT INTO iam_user_role(user_id, role_id)
SELECT u.id, r.id
FROM iam_user u
JOIN iam_role r ON r.code='customer'
WHERE u.login_name='cust01'
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.15', 'customer portal bind', '11ea9222b63ba03a3fa44633016ab9f994dd631d79fb13c7ef243dbf4e79541c')
ON CONFLICT (version) DO NOTHING;
