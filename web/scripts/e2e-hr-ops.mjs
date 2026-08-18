/**
 * E2E 人事运维模块：班次/考勤/打卡/请假/加班/月统计/绩效/外访/备忘/日志/离职
 * Usage: node web/scripts/e2e-hr-ops.mjs
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
  try { data = text ? JSON.parse(text) : {} } catch {
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

  const empNo = `OPS-${Date.now()}`
  const emp = await req('POST', '/hr/employees', {
    emp_no: empNo, name: '人事运维工', emp_type: 'piece', status: 'active',
  }, token)
  assert(emp.code === 1 && emp.data?.id, `emp: ${emp.msg}`)
  const eid = emp.data.id
  log(`employee #${eid}`)

  const shift = await req('POST', '/hr/shifts', {
    code: `SX${Date.now().toString().slice(-5)}`, name: 'E2E班', start_time: '08:00', end_time: '17:00', workshop_dept_id: 0,
  }, token)
  assert(shift.code === 1 && shift.data?.id, `shift: ${shift.msg}`)
  log(`shift #${shift.data.id}`)

  const rule = await req('POST', '/hr/attendance/rules', {
    name: 'E2E规则', shift_id: shift.data.id, late_minutes: 5, early_minutes: 5,
  }, token)
  assert(rule.code === 1, `rule: ${rule.msg}`)
  log('attendance rule')

  const punchIn = await req('POST', '/hr/attendance/records/punch', {
    employee_id: eid, biz_date: '2026-08-04', punch_type: 'in',
  }, token)
  assert(punchIn.code === 1, `punch in: ${punchIn.msg}`)
  const punchOut = await req('POST', '/hr/attendance/records/punch', {
    employee_id: eid, biz_date: '2026-08-04', punch_type: 'out',
  }, token)
  assert(punchOut.code === 1, `punch out: ${punchOut.msg}`)
  const recs = await req('GET', '/hr/attendance/records?employee_id=' + eid, undefined, token)
  assert(recs.code === 1 && (recs.data?.list || []).length >= 1, `records: ${recs.msg}`)
  log('attendance punch + list')

  const leave = await req('POST', '/hr/leave-requests', {
    employee_id: eid, leave_type: 'personal',
    start_at: '2026-08-05 09:00:00', end_at: '2026-08-05 18:00:00',
  }, token)
  assert(leave.code === 1 && leave.data?.id, `leave: ${leave.msg}`)
  const cancel = await req('POST', `/hr/leave-requests/${leave.data.id}/cancel`, {}, token)
  assert(cancel.code === 1 && cancel.data?.status === 'cancelled', `cancel leave: ${cancel.msg}`)
  log('leave + cancel')

  const ot = await req('POST', '/hr/overtime-patches', {
    employee_id: eid, biz_type: 'overtime', biz_date: '2026-08-04', minutes: 90,
  }, token)
  assert(ot.code === 1, `overtime: ${ot.msg}`)
  const otStats = await req('GET', '/hr/overtime-patches/stats', undefined, token)
  assert(otStats.code === 1 && typeof otStats.data?.total_minutes === 'number', `ot stats: ${otStats.msg}`)
  log('overtime + stats')

  const recalc = await req('POST', '/hr/attendance/month-stats/recalc', { year: 2026, month: 8 }, token)
  assert(recalc.code === 1 && recalc.data?.updated >= 1, `recalc: ${recalc.msg}`)
  const ms = await req('GET', '/hr/attendance/month-stats?year=2026&month=8', undefined, token)
  assert(ms.code === 1 && (ms.data?.list || []).some((x) => x.employee_id === eid), `month stats: ${ms.msg}`)
  log('month stats recalc')

  const scheme = await req('POST', '/hr/performance/schemes', {
    name: 'E2E方案', scheme_json: { kpi: ['产量', '质量'] },
  }, token)
  assert(scheme.code === 1, `scheme: ${scheme.msg}`)
  const result = await req('POST', '/hr/performance/results', {
    scheme_id: scheme.data.id, employee_id: eid, period: '2026-08', score: 88, amount: 200,
  }, token)
  assert(result.code === 1, `perf result: ${result.msg}`)
  const sum = await req('GET', '/hr/attendance-perf-summaries?period=2026-08', undefined, token)
  assert(sum.code === 1, `perf summary: ${sum.msg}`)
  log('performance')

  const visit = await req('POST', '/hr/visits', {
    employee_id: eid, customer_id: 1, visit_at: '2026-08-04 14:00:00', content: '拜访客户', location: '厂区门口',
  }, token)
  assert(visit.code === 1, `visit: ${visit.msg}`)
  const memo = await req('POST', '/hr/memos', { title: '人事备忘', content: '跟进入职', biz_date: '2026-08-04' }, token)
  assert(memo.code === 1, `memo: ${memo.msg}`)
  const journal = await req('POST', '/hr/employee-journals', {
    employee_id: eid, biz_date: '2026-08-04', content: '今日完成去皮工序',
  }, token)
  assert(journal.code === 1, `journal: ${journal.msg}`)
  log('visit + memo + journal')

  const off = await req('POST', '/hr/offboards', {
    employee_id: eid, offboard_date: '2026-08-10', reason: 'e2e leave', revoke_permission: true,
  }, token)
  assert(off.code === 1 && off.data?.id, `offboard: ${off.msg}`)
  const offConfirm = await req('POST', `/hr/offboards/${off.data.id}/confirm`, {}, token)
  assert(offConfirm.code === 1 && offConfirm.data?.status === 'confirmed', `off confirm: ${offConfirm.msg}`)
  const emp2 = await req('GET', `/hr/employees/${eid}`, undefined, token)
  assert(emp2.data?.status === 'left', `emp left: ${emp2.data?.status}`)
  log('offboard confirm')

  console.log('\nALL GREEN — e2e-hr-ops passed')
}

main().catch((e) => {
  console.error('\nFAILED:', e.message)
  process.exit(1)
})
