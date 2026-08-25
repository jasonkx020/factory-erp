/**
 * E2E 入职登记：建档+入职单 → 草稿编辑 → 确认开户赋权 → 取消草稿
 * Usage: node web/scripts/e2e-onboard.mjs
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

async function jobTitleId(token, name, empType = '') {
  const q = empType ? `?emp_type=${encodeURIComponent(empType)}` : ''
  const res = await req('GET', `/hr/job-titles${q}`, undefined, token)
  assert(res.code === 1, `job-titles: ${res.msg}`)
  const hit = (res.data?.list || []).find((x) => x.name === name)
  if (hit?.id) return hit.id
  const ensured = await req('POST', '/hr/job-titles/ensure', { name, emp_type: empType }, token)
  assert(ensured.code === 1 && ensured.data?.id, `ensure job title ${name}: ${ensured.msg}`)
  return ensured.data.id
}

async function main() {
  const log = (m) => console.log('✓', m)

  const login = await req('POST', '/auth/login', { login_name: 'admin', password: 'admin123', client_type: 'web' })
  assert(login.code === 1, `login: ${login.msg}`)
  const token = login.data.access_token
  log('login')

  const peelId = await jobTitleId(token, '去皮工', 'piece')
  const cutId = await jobTitleId(token, '切断工', 'fixed')
  log('job titles loaded')

  const roles = await req('GET', '/iam/roles', undefined, token)
  assert(roles.code === 1, `roles: ${roles.msg}`)
  const roleID = (roles.data?.list || []).find((r) => r.code === 'piece')?.id
    || (roles.data?.list || [])[0]?.id
  assert(roleID, 'need role')

  const empNo = `ONB-${Date.now()}`
  const created = await req('POST', '/hr/onboards', {
    emp_no: empNo,
    name: '入职E2E工人',
    emp_type: 'piece',
    dept_id: 1,
    team_id: 1,
    job_title_id: peelId,
    mobile: '13800001111',
    badge_code: `BD-${Date.now().toString().slice(-6)}`,
    onboard_date: '2026-08-04',
    need_account: true,
    login_name: empNo,
    role_ids: [roleID],
    remark: 'e2e onboard',
  }, token)
  assert(created.code === 1 && created.data?.id, `create onboard: ${created.msg}`)
  const oid = created.data.id
  const empID = created.data.employee_id
  assert(created.data.status === 'draft', 'should be draft')
  assert(empID > 0, 'employee should be created')
  log(`onboard #${oid} emp #${empID}`)

  const detail = await req('GET', `/hr/onboards/${oid}`, undefined, token)
  assert(detail.code === 1 && detail.data?.employee?.name === '入职E2E工人', `detail: ${detail.msg}`)
  assert(detail.data?.need_account === true, 'need_account')
  log('detail + employee archive')

  const upd = await req('PUT', `/hr/onboards/${oid}`, {
    name: '入职E2E工人改',
    job_title_id: cutId,
    remark: 'e2e updated',
    role_ids: [roleID],
    need_account: true,
  }, token)
  assert(upd.code === 1 && upd.data?.employee?.name === '入职E2E工人改', `update: ${upd.msg}`)
  log('draft update')

  const conf = await req('POST', `/hr/onboards/${oid}/confirm`, {}, token)
  assert(conf.code === 1 && conf.data?.status === 'confirmed', `confirm: ${conf.msg}`)
  assert(conf.data?.has_account === true || conf.data?.user_id > 0, 'should open account')
  const emp = await req('GET', `/hr/employees/${empID}`, undefined, token)
  assert(emp.code === 1 && emp.data?.status === 'active', `emp active: ${emp.data?.status}`)
  assert(emp.data?.user_id > 0, 'user bound')
  log('confirm → active + account')

  // cancel path on another draft
  const empNo2 = `ONB2-${Date.now()}`
  const d2 = await req('POST', '/hr/onboards', {
    emp_no: empNo2, name: '取消草稿', emp_type: 'fixed', need_account: false,
  }, token)
  assert(d2.code === 1, `draft2: ${d2.msg}`)
  const cancel = await req('POST', `/hr/onboards/${d2.data.id}/cancel`, {}, token)
  assert(cancel.code === 1 && cancel.data?.status === 'cancelled', `cancel: ${cancel.msg}`)
  log('cancel draft')

  // 临时工：与计件工采集字段一致，仅类别不同
  const empNo3 = `ONB3-${Date.now()}`
  const d3 = await req('POST', '/hr/onboards', {
    emp_no: empNo3, name: '临时工E2E', emp_type: '临时工', need_account: false,
  }, token)
  assert(d3.code === 1, `draft3: ${d3.msg}`)
  const d3Detail = await req('GET', `/hr/onboards/${d3.data.id}`, undefined, token)
  assert(d3Detail.data?.emp_type === 'temp', `temp emp_type: ${d3Detail.data?.emp_type}`)
  log('temp employee type (中文名归一化)')

  const bad = await req('POST', '/hr/employees', {
    emp_no: `BAD-${Date.now()}`, name: '非法类型', emp_type: 'unknown_type',
  }, token)
  assert(bad.code !== 1 && String(bad.msg).includes('INVALID_EMP_TYPE'), `reject bad emp_type: ${bad.msg}`)
  log('reject invalid emp_type')

  const list = await req('GET', '/hr/onboards?status=confirmed', undefined, token)
  assert(list.code === 1 && (list.data?.list || []).some((x) => x.id === oid), 'list filter confirmed')
  assert(typeof list.data?.summary?.confirmed === 'number', 'summary')
  log('list + summary')

  console.log('\nALL GREEN — e2e-onboard passed')
}

main().catch((e) => {
  console.error('\nFAILED:', e.message)
  process.exit(1)
})
