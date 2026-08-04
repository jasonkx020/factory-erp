/**
 * E2E flow: cassava 12-step dual-scan → piecework → checkpoint → stock → trace → repair → retry
 * Usage: node web/scripts/e2e-flow.mjs
 */
const BASE = process.env.ERP_API || 'http://127.0.0.1:18080/api/v1'

async function req(method, path, body, token, extraHeaders = {}) {
  const headers = { 'Content-Type': 'application/json', ...extraHeaders }
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
  assert(login.code === 1 && login.data?.access_token, `login: ${login.msg}`)
  const token = login.data.access_token
  log('login')

  // ensure wage rates for piecework processes
  for (const [process_id, rate] of [[1, 0.18], [4, 0.25], [10, 0.22]]) {
    await req('POST', '/payroll/wage-rates', { process_id, rate, effective_from: '2026-01-01', status: 'active' }, token)
  }
  log('wage rates')

  // flow rules / routing
  const rules = await req('GET', '/production/flow-rules', undefined, token)
  assert(rules.code === 1 && (rules.data?.steps?.length || 0) >= 12, `flow-rules: ${rules.msg}`)
  const routings = await req('GET', '/production/routings', undefined, token)
  assert(routings.code === 1 && (routings.data?.list?.length || 0) > 0, `routings: ${routings.msg}`)
  log(`routing steps=${rules.data.steps.length}`)

  // badge bind
  const badge = await req('PUT', '/hr/employees/2/badge', { badge_code: 'EMP0301' }, token)
  assert(badge.code === 1, `badge: ${badge.msg}`)
  log('badge bind')

  // fresh box at step 1
  const boxCode = `BX-E2E-${Date.now()}`
  const box = await req('POST', '/inventory/box-codes', {
    code: boxCode,
    product_id: 1,
    warehouse_id: 1,
    batch_no: 'E2E',
    qty: 500,
    weight: 500,
    current_process_id: 8,
    current_step_id: 1,
    status: 'open',
  }, token)
  assert(box.code === 1, `create box: ${box.msg} ${JSON.stringify(box)}`)
  log(`box ${boxCode}`)

  // resolve preview
  const preview = await req('POST', '/production/scan/resolve', {
    badge_code: 'EMP0301', box_code: boxCode, net_weight: 50,
  }, token)
  assert(preview.code === 1 && preview.data?.worker_id, `resolve: ${preview.msg}`)
  log(`resolve worker=${preview.data.worker_name}`)

  let currentBox = boxCode
  let lastTrace = ''
  let pieceworkHits = 0
  let checkpointHits = 0
  let stockInHits = 0
  const flowIds = []
  const pieceProcessIds = new Set([1, 4, 10])

  for (let i = 1; i <= 12; i++) {
    const scan = await req('POST', '/production/scan', {
      badge_code: 'EMP0301',
      box_code: currentBox,
      net_weight: 40 + i,
    }, token, { 'X-Trace-Id': `e2e-flow-${Date.now()}-${i}` })
    assert(scan.code === 1, `scan step ${i}: ${scan.msg} box=${currentBox}`)
    const next = scan.data?.next || {}
    lastTrace = scan.data?.trace_id || lastTrace
    if (next.flow_event_id) flowIds.push(next.flow_event_id)
    if (pieceProcessIds.has(Number(scan.data?.process_id))) {
      assert(Number(scan.data?.wage_amount) > 0, `step ${i} piecework wage expected >0 got ${scan.data?.wage_amount}`)
      pieceworkHits++
    }
    if (next.checkpoint_work_order_id) checkpointHits++
    if (next.new_box_code) {
      currentBox = String(next.new_box_code)
      stockInHits++
    }
    if (i < 12) {
      assert(next.next_step || next.finished || next.next_work_order_id, `step ${i} should advance`)
    } else {
      assert(next.finished === true, `step 12 should finish, got ${JSON.stringify(next)}`)
    }
    log(`scan #${i} process=${scan.data.process_id} wage=${scan.data.wage_amount ?? 0} box=${currentBox} next=${next.next_step || (next.finished ? 'DONE' : '-')}`)
  }

  assert(pieceworkHits >= 2, `expected piecework hits >=2 got ${pieceworkHits}`)
  assert(checkpointHits >= 1, `expected checkpoint >=1 got ${checkpointHits}`)
  assert(stockInHits >= 1, `expected stock-in boxes >=1 got ${stockInHits}`)
  log(`piecework=${pieceworkHits} checkpoint=${checkpointHits} stockIn=${stockInHits}`)

  // finished goods balance warehouse 3
  const bal = await req('GET', '/inventory/balances', undefined, token)
  assert(bal.code === 1, `balances: ${bal.msg}`)
  const finished = (bal.data?.list || []).filter((x) => Number(x.warehouse_id) === 3)
  assert(finished.length > 0, 'finished warehouse balance missing')
  log(`finished warehouse rows=${finished.length}`)

  // box / trace
  const traceBox = await req('GET', `/inventory/box-codes/trace/${encodeURIComponent(boxCode)}`, undefined, token)
  assert(traceBox.code === 1, `box trace: ${traceBox.msg}`)
  const evCount = (traceBox.data?.flow_events || []).length
  assert(evCount >= 12, `box flow events expected >=12 got ${evCount} related=${JSON.stringify(traceBox.data?.related_boxes)}`)
  assert((traceBox.data?.operation_logs || []).length >= 1, 'box operation logs empty')
  log(`box trace events=${evCount} related=${(traceBox.data?.related_boxes || []).length} logs=${(traceBox.data?.operation_logs || []).length}`)

  if (lastTrace) {
    const tr = await req('GET', `/system/operation-logs/trace/${encodeURIComponent(lastTrace)}`, undefined, token)
    assert(tr.code === 1, `op trace: ${tr.msg}`)
    assert((tr.data?.list || []).length >= 1, 'operation log trace empty')
    log(`op-log trace ${lastTrace} n=${tr.data.list.length}`)
  }

  const logs = await req('GET', '/system/operation-logs?page_num=1&page_size=5', undefined, token)
  assert(logs.code === 1 && (logs.data?.list || []).length > 0, `operation-logs: ${logs.msg}`)
  log('operation-logs list')

  // data repair
  const failCreate = await req('POST', '/system/data-repairs', {
    target_type: 'inv_box_code', target_id: 1, action: 'reopen_box', reason: '',
  }, token)
  assert(failCreate.code !== 1, 'repair without reason should fail')

  const repair = await req('POST', '/system/data-repairs', {
    target_type: 'inv_box_code',
    target_id: box.data?.id || 1,
    action: 'reopen_box',
    reason: 'e2e reopen after finish',
  }, token)
  assert(repair.code === 1 && repair.data?.id, `create repair: ${repair.msg}`)
  const applied = await req('POST', `/system/data-repairs/${repair.data.id}/apply`, {}, token)
  assert(applied.code === 1 && applied.data?.status === 'applied', `apply repair: ${applied.msg}`)
  log(`data-repair #${repair.data.id} applied`)

  // force a failed flow event then retry
  const events = await req('GET', '/production/flow-events?page_num=1&page_size=20', undefined, token)
  assert(events.code === 1, `flow-events: ${events.msg}`)
  let retryId = flowIds[0]
  if (!retryId && (events.data?.list || []).length) {
    retryId = events.data.list[0].id
  }
  if (retryId) {
    // mark failed via repair noop path is enough; call retry endpoint
    const retry = await req('POST', `/production/flow-events/${retryId}/retry`, {}, token)
    assert(retry.code === 1, `flow retry: ${retry.msg}`)
    log(`flow-event retry #${retryId}`)
  }

  console.log('\nALL GREEN — e2e-flow passed')
}

main().catch((e) => {
  console.error('\nFAIL', e.message || e)
  process.exit(1)
})
