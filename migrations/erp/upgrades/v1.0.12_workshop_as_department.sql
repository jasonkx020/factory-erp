-- v1.0.12: 车间并入公司架构（dept_type=workshop），去掉独立 pd_workshop

ALTER TABLE sys_department ADD COLUMN IF NOT EXISTS dept_type TEXT NOT NULL DEFAULT 'normal';
ALTER TABLE sys_department ADD COLUMN IF NOT EXISTS _mig_workshop_id INTEGER;

DO $$
BEGIN
  IF to_regclass('public.pd_workshop') IS NOT NULL THEN
    EXECUTE $m$
      INSERT INTO sys_department (org_id, parent_id, code, name, status, dept_type, is_deleted, created_at, updated_at, _mig_workshop_id)
      SELECT
        COALESCE(w.org_id, 1),
        CASE
          WHEN COALESCE(w.dept_id, 0) > 0 THEN w.dept_id
          ELSE (SELECT d.id FROM sys_department d WHERE COALESCE(d.is_deleted, 0) = 0 AND COALESCE(d.dept_type, 'normal') = 'normal' ORDER BY d.id LIMIT 1)
        END,
        CASE
          WHEN EXISTS (
            SELECT 1 FROM sys_department d
            WHERE d.org_id = COALESCE(w.org_id, 1) AND d.code = w.code AND COALESCE(d.is_deleted, 0) = 0
          ) THEN w.code || '-WS'
          ELSE w.code
        END,
        w.name,
        COALESCE(w.status, 'active'),
        'workshop',
        COALESCE(w.is_deleted, 0),
        COALESCE(w.created_at::text, NOW()::text),
        COALESCE(w.updated_at::text, NOW()::text),
        w.id
      FROM pd_workshop w
      WHERE NOT EXISTS (SELECT 1 FROM sys_department d WHERE d._mig_workshop_id = w.id)
    $m$;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'hr_employee' AND column_name = 'workshop_id'
  ) THEN
    EXECUTE $m$
      INSERT INTO hr_employee_department(employee_id, dept_id, is_primary)
      SELECT e.id, d.id, 0
      FROM hr_employee e
      JOIN sys_department d ON d._mig_workshop_id = e.workshop_id
      WHERE COALESCE(e.workshop_id, 0) > 0
        AND COALESCE(e.is_deleted, 0) = 0
      ON CONFLICT DO NOTHING
    $m$;
  END IF;
END $$;

DO $$
DECLARE
  def_ws BIGINT;
BEGIN
  SELECT id INTO def_ws FROM sys_department WHERE dept_type = 'workshop' AND COALESCE(is_deleted, 0) = 0 ORDER BY id LIMIT 1;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'pd_work_team' AND column_name = 'workshop_id') THEN
    ALTER TABLE pd_work_team ADD COLUMN IF NOT EXISTS dept_id INTEGER;
    UPDATE pd_work_team t SET dept_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    UPDATE pd_work_team SET dept_id = def_ws WHERE COALESCE(dept_id, 0) = 0 AND def_ws IS NOT NULL;
    ALTER TABLE pd_work_team DROP CONSTRAINT IF EXISTS pd_work_team_workshop_id_fkey;
    ALTER TABLE pd_work_team DROP CONSTRAINT IF EXISTS pd_work_team_workshop_id_code_key;
    ALTER TABLE pd_work_team DROP COLUMN IF EXISTS workshop_id;
    ALTER TABLE pd_work_team ALTER COLUMN dept_id SET NOT NULL;
    CREATE UNIQUE INDEX IF NOT EXISTS pd_work_team_dept_id_code_key ON pd_work_team(dept_id, code);
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'pd_production_task' AND column_name = 'workshop_id') THEN
    UPDATE pd_production_task t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE pd_production_task RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'pd_routing_step' AND column_name = 'workshop_id') THEN
    UPDATE pd_routing_step t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE pd_routing_step RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'pd_shift' AND column_name = 'workshop_id') THEN
    UPDATE pd_shift t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    UPDATE pd_shift SET workshop_id = def_ws WHERE COALESCE(workshop_id, 0) = 0 AND def_ws IS NOT NULL;
    ALTER TABLE pd_shift ALTER COLUMN workshop_id DROP DEFAULT;
    ALTER TABLE pd_shift RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'hr_shift' AND column_name = 'workshop_id') THEN
    UPDATE hr_shift t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE hr_shift RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'inv_stocktake' AND column_name = 'workshop_id') THEN
    UPDATE inv_stocktake t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE inv_stocktake RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'pay_payroll_sheet' AND column_name = 'workshop_id') THEN
    UPDATE pay_payroll_sheet t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE pay_payroll_sheet RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'sys_batch_payroll_job' AND column_name = 'workshop_id') THEN
    UPDATE sys_batch_payroll_job t SET workshop_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE sys_batch_payroll_job RENAME COLUMN workshop_id TO workshop_dept_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'iam_user_data_scope' AND column_name = 'workshop_id') THEN
    ALTER TABLE iam_user_data_scope ADD COLUMN IF NOT EXISTS dept_id INTEGER;
    UPDATE iam_user_data_scope t SET dept_id = d.id FROM sys_department d WHERE d._mig_workshop_id = t.workshop_id;
    ALTER TABLE iam_user_data_scope DROP COLUMN IF EXISTS workshop_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'sys_personnel_transfer' AND column_name = 'to_workshop_id') THEN
    UPDATE sys_personnel_transfer t
    SET to_dept_id = COALESCE(NULLIF(t.to_dept_id, 0), d.id)
    FROM sys_department d
    WHERE d._mig_workshop_id = t.to_workshop_id AND COALESCE(t.to_dept_id, 0) = 0;
    ALTER TABLE sys_personnel_transfer DROP COLUMN IF EXISTS from_workshop_id;
    ALTER TABLE sys_personnel_transfer DROP COLUMN IF EXISTS to_workshop_id;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'hr_employee' AND column_name = 'workshop_id') THEN
    ALTER TABLE hr_employee DROP COLUMN IF EXISTS workshop_id;
  END IF;

  UPDATE iam_user_data_scope SET data_scope_type = 'dept_workshop' WHERE data_scope_type = 'workshop';
  UPDATE iam_role SET data_scope_type = 'dept_workshop' WHERE data_scope_type = 'workshop';
END $$;

ALTER TABLE sys_department DROP COLUMN IF EXISTS _mig_workshop_id;
DROP TABLE IF EXISTS pd_workshop;

INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('v1.0.12', 'unify workshop as department node', 'e3d6caaa3ae3362447edbd623372fb53aefcfd05c34fb25d9074db8f67eb3828')
ON CONFLICT (version) DO NOTHING;
