/**
 * E2E HR permission workbench:
 * employee → open-account → bind/scope/roles → freeze → onboard/offboard revoke
 * Usage: node web/scripts/e2e-hr-perm.mjs
 */
const BASE = process.env.ERP_API || 'http://127.0.0.1:18080/api/v1'

async function req(method, path, body, token) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await res.text()
  let data = {}
  try {
    data = text ? JSON.parse(text) : {}
  } catch {
    return { status: res.status, code: 0, msg: `NON_JSON:${text.slice(0, 120)}` }
  }
  return { status: res.status, ...data }
}

function assert(cond, msg) {
  if (!cond) throw new Error(msg)
}

async function main() {
  const log = (m) => console.log('✓', m)

  const login = await req('POST', '/auth/login', { login_name: 'admin', password: 'admin123', client_type: 'web' })
  assert(login.code === 1, `login: ${login.msg}`)
  const token = login.data.access_token
  log('login')

  const ov = await req('GET', '/iam/hr-perm-overview', undefined, token)
  assert(ov.code === 1 && typeof ov.data?.users === 'number', `overview: ${ov.msg}`)
  log(`overview users=${ov.data.users} unbound=${ov.data.unbound_employees}`)

  const empNo = `E2E-HR-${Date.now()}`
  const emp = await req('POST', '/hr/employees', {
    emp_no: empNo,
    name: '权限E2E员工',
    emp_type: 'piece',
    status: 'active',
    workshop_id: 1,
  }, token)
  assert(emp.code === 1 && emp.data?.id, `create employee: ${emp.msg}`)
  const empID = emp.data.id
  log(`employee #${empID}`)

  const opened = await req('POST', `/hr/employees/${empID}/open-account`, {
    login_name: empNo,
    password: 'ChangeMe123',
  }, token)
  assert(opened.code === 1 && opened.data?.user_id, `open-account: ${opened.msg}`)
  const uid = opened.data.user_id
  log(`open-account user #${uid}`)

  const detail = await req('GET', `/iam/users/${uid}`, undefined, token)
  assert(detail.code === 1 && detail.data?.employee_id === empID, `user detail: ${detail.msg}`)
  log('user detail bound')

  const roles = await req('GET', '/iam/roles', undefined, token)
  assert(roles.code === 1, `roles: ${roles.msg}`)
  const roleID = (roles.data?.list || [])[0]?.id
  assert(roleID, 'need at least one role')
  const setR = await req('PUT', `/iam/users/${uid}/roles`, { role_ids: [roleID] }, token)
  assert(setR.code === 1, `set roles: ${setR.msg}`)
  log(`roles -> ${roleID}`)

  const scope = await req('PUT', `/iam/users/${uid}/data-scope`, {
    data_scope_type: 'workshop',
    workshop_id: 1,
    team_id: 0,
  }, token)
  assert(scope.code === 1 && scope.data?.data_scope_type === 'workshop', `data-scope: ${scope.msg}`)
  const getScope = await req('GET', `/iam/users/${uid}/data-scope`, undefined, token)
  assert(getScope.code === 1 && getScope.data?.workshop_id === 1, `get data-scope: ${getScope.msg}`)
  log('data-scope workshop')

  // create second employee + bind/unbind on a fresh user via create user
  const emp2No = `E2E-HR2-${Date.now()}`
  const emp2 = await req('POST', '/hr/employees', {
    emp_no: emp2No, name: '权限E2E员工2', emp_type: 'piece', status: 'active',
  }, token)
  assert(emp2.code === 1, `emp2: ${emp2.msg}`)
  const emp2ID = emp2.data.id

  const user2 = await req('POST', '/iam/users', {
    login_name: `u-${emp2No}`,
    password: 'ChangeMe123',
    user_type: 'biz',
  }, token)
  assert(user2.code === 1 && user2.data?.id, `create user: ${user2.msg}`)
  const uid2 = user2.data.id
  const bind = await req('PUT', `/iam/users/${uid2}/bind-employee`, { employee_id: emp2ID }, token)
  assert(bind.code === 1 && bind.data?.bound === true, `bind: ${bind.msg}`)
  const unbind = await req('DELETE', `/iam/users/${uid2}/bind-employee`, undefined, token)
  assert(unbind.code === 1, `unbind: ${unbind.msg}`)
  const rebind = await req('PUT', `/iam/users/${uid2}/bind-employee`, { employee_id: emp2ID }, token)
  assert(rebind.code === 1, `rebind: ${rebind.msg}`)
  log('bind/unbind')

  const fr = await req('POST', `/iam/users/${uid2}/freeze`, { reason: 'e2e freeze' }, token)
  assert(fr.code === 1 && fr.data?.status === 'frozen', `freeze: ${fr.msg}`)
  const uf = await req('POST', `/iam/users/${uid2}/unfreeze`, {}, token)
  assert(uf.code === 1, `unfreeze: ${uf.msg}`)
  log('freeze/unfreeze')

  const onb = await req('POST', '/hr/onboards', {
    employee_id: emp2ID,
    remark: 'e2e onboard',
    role_ids: [roleID],
  }, token)
  // emp2 already has account after rebind — confirm may no-op open
  // create third employee for onboard confirm path
  const emp3No = `E2E-HR3-${Date.now()}`
  const emp3 = await req('POST', '/hr/employees', {
    emp_no: emp3No, name: '权限E2E入职', emp_type: 'piece', status: 'active',
  }, token)
  assert(emp3.code === 1, `emp3: ${emp3.msg}`)
  const onb3 = await req('POST', '/hr/onboards', {
    employee_id: emp3.data.id, remark: 'e2e', role_ids: [roleID],
  }, token)
  assert(onb3.code === 1 && onb3.data?.id, `onboard create: ${onb3.msg}${onb.code}`)
  const onbConfirm = await req('POST', `/hr/onboards/${onb3.data.id}/confirm`, {}, token)
  assert(onbConfirm.code === 1 && onbConfirm.data?.status === 'confirmed', `onboard confirm: ${onbConfirm.msg}`)
  log(`onboard confirm emp#${emp3.data.id}`)

  const off = await req('POST', '/hr/offboards', {
    employee_id: empID,
    reason: 'e2e leave',
    revoke_permission: true,
  }, token)
  assert(off.code === 1 && off.data?.id, `offboard create: ${off.msg}`)
  const offConfirm = await req('POST', `/hr/offboards/${off.data.id}/confirm`, {}, token)
  assert(offConfirm.code === 1 && offConfirm.data?.status === 'confirmed', `offboard confirm: ${offConfirm.msg}`)
  const after = await req('GET', `/iam/users/${uid}`, undefined, token)
  assert(after.code === 1 && after.data?.status === 'frozen', `user frozen after offboard: ${after.data?.status}`)
  log('offboard revoke + freeze')

  const sessions = await req('GET', '/iam/sessions', undefined, token)
  assert(sessions.code === 1, `sessions: ${sessions.msg}`)
  log(`sessions ${(sessions.data?.list || []).length}`)

  // role management
  const code = `e2e_role_${Date.now()}`
  const createdRole = await req('POST', '/iam/roles', {
    code, name: 'E2E角色', data_scope_type: 'workshop', remark: 'e2e',
  }, token)
  assert(createdRole.code === 1 && createdRole.data?.id, `create role: ${createdRole.msg}`)
  const rid = createdRole.data.id
  const roleDetail = await req('GET', `/iam/roles/${rid}`, undefined, token)
  assert(roleDetail.code === 1 && roleDetail.data?.role?.code === code, `role detail: ${roleDetail.msg}`)
  const upd = await req('PUT', `/iam/roles/${rid}`, { name: 'E2E角色改', data_scope_type: 'warehouse' }, token)
  assert(upd.code === 1, `update role: ${upd.msg}`)
  const perms = await req('GET', '/iam/permissions', undefined, token)
  const pid = (perms.data?.list || [])[0]?.id
  if (pid) {
    const sp = await req('PUT', `/iam/roles/${rid}/permissions`, { permission_ids: [pid] }, token)
    assert(sp.code === 1, `set perms: ${sp.msg}`)
  }
  const wh = await req('PUT', `/iam/roles/${rid}/warehouse-scope`, { warehouse_ids: [1, 2] }, token)
  assert(wh.code === 1, `warehouse-scope: ${wh.msg}`)
  const ps = await req('PUT', `/iam/roles/${rid}/process-scope`, {
    items: [{ process_id: 1, can_report: true, can_dispatch: true }],
  }, token)
  assert(ps.code === 1, `process-scope: ${ps.msg}`)
  const roleDetail2 = await req('GET', `/iam/roles/${rid}`, undefined, token)
  assert(roleDetail2.data?.warehouse_ids?.includes(1), 'warehouse scope persisted')
  assert(roleDetail2.data?.process_scopes?.some((x) => x.process_id === 1 && x.can_dispatch), 'process scope persisted')
  log(`role #${rid} manage`)

  console.log('\nALL GREEN — e2e-hr-perm passed')
}

main().catch((e) => {
  console.error('\nFAILED:', e.message)
  process.exit(1)
})
