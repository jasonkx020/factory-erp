/**
 * E2E supplier P0-P2: master → qualify → inbound post → qc/return → analytics → finance guard
 * Usage: node web/scripts/e2e-supplier.mjs
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

  const code = `SUP-E2E-${Date.now()}`
  const created = await req('POST', '/purchase/suppliers', {
    code,
    name: 'E2E木薯原料商',
    short_name: 'E2E原料',
    supplier_type: 'raw',
    status: 'potential',
    rating: 'A',
    settle_method: 'monthly',
    payment_days: 30,
    lead_time_days: 3,
    moq: 100,
    contact_json: [{ name: '测联', mobile: '13900000000', is_primary: true }],
  }, token)
  assert(created.code === 1 && created.data?.id, `create supplier: ${created.msg}`)
  const sid = created.data.id
  log(`supplier #${sid}`)

  const lic = await req('PUT', `/purchase/suppliers/${sid}/licenses`, {
    items: [{ license_type: '食品经营许可', license_no: 'JY-E2E', expire_date: '2026-08-20' }],
  }, token)
  assert(lic.code === 1, `licenses: ${lic.msg}`)
  const items = await req('PUT', `/purchase/suppliers/${sid}/supply-items`, {
    items: [{ product_id: 1, is_preferred: true, moq: 100, lead_time_days: 3, last_price: 1.5 }],
  }, token)
  assert(items.code === 1, `supply-items: ${items.msg}`)
  log('licenses + supply-items')

  const blocked = await req('POST', '/purchase/inbounds', {
    supplier_id: sid, product_id: 1, qty: 10, price: 1.5, warehouse_id: 1,
  }, token)
  assert(blocked.code !== 1, 'potential supplier should not create inbound')
  log('potential blocked')

  const q = await req('POST', `/purchase/suppliers/${sid}/qualify`, {}, token)
  assert(q.code === 1 && q.data?.status === 'qualified', `qualify: ${q.msg}`)
  log('qualified')

  const fr = await req('POST', `/purchase/suppliers/${sid}/freeze`, {}, token)
  assert(fr.code === 1 && fr.data?.status === 'frozen', `freeze: ${fr.msg}`)
  const blocked2 = await req('POST', '/purchase/inbounds', {
    supplier_id: sid, product_id: 1, qty: 10, price: 1.5, warehouse_id: 1,
  }, token)
  assert(blocked2.code !== 1, 'frozen supplier should not create inbound')
  const act = await req('POST', `/purchase/suppliers/${sid}/activate`, {}, token)
  assert(act.code === 1 && act.data?.status === 'qualified', `activate: ${act.msg}`)
  log('freeze/activate')

  // balance before
  const bal0 = await req('GET', '/inventory/balances', undefined, token)
  const before = ((bal0.data?.list) || []).find((x) => Number(x.warehouse_id) === 1 && Number(x.product_id) === 1)
  const beforeQty = Number(before?.qty || 0)

  const ib = await req('POST', '/purchase/inbounds', {
    supplier_id: sid, product_id: 1, qty: 25, price: 1.66, warehouse_id: 1, batch_no: 'E2EB',
  }, token)
  assert(ib.code === 1 && ib.data?.id, `inbound: ${ib.msg}`)
  const post = await req('POST', `/purchase/inbounds/${ib.data.id}/post`, {}, token)
  assert(post.code === 1 && post.data?.status === 'posted', `inbound post: ${post.msg}`)
  log(`inbound posted #${ib.data.id}`)

  const bal1 = await req('GET', '/inventory/balances', undefined, token)
  const after = ((bal1.data?.list) || []).find((x) => Number(x.warehouse_id) === 1 && Number(x.product_id) === 1)
  assert(Number(after?.qty || 0) >= beforeQty + 25, `balance expect >= ${beforeQty + 25} got ${after?.qty}`)
  log('stock increased')

  const hist = await req('GET', `/purchase/price-histories?supplier_id=${sid}`, undefined, token)
  assert(hist.code === 1 && (hist.data?.list || []).length >= 1, `price history: ${hist.msg}`)
  log('price history')

  const qc = await req('POST', '/purchase/incoming-qcs', {
    inbound_id: ib.data.id, supplier_id: sid, product_id: 1, qty_check: 25,
  }, token)
  assert(qc.code === 1, `qc: ${qc.msg}`)
  const pass = await req('POST', `/purchase/incoming-qcs/${qc.data.id}/pass`, {}, token)
  assert(pass.code === 1 && pass.data?.result === 'pass', `qc pass: ${pass.msg}`)

  const ret = await req('POST', '/purchase/returns', {
    supplier_id: sid, inbound_id: ib.data.id, warehouse_id: 1, product_id: 1, qty: 2, reason: 'e2e return',
  }, token)
  assert(ret.code === 1, `return: ${ret.msg}`)
  const retPost = await req('POST', `/purchase/returns/${ret.data.id}/post`, {}, token)
  assert(retPost.code === 1 && retPost.data?.status === 'posted', `return post: ${retPost.msg}`)
  log('qc + return')

  const perf = await req('GET', `/purchase/suppliers/${sid}/performance`, undefined, token)
  assert(perf.code === 1 && Number(perf.data?.purchase_qty) > 0, `performance: ${JSON.stringify(perf)}`)
  const alerts = await req('GET', '/purchase/certificate-alerts?days=90', undefined, token)
  assert(alerts.code === 1 && (alerts.data?.list || []).length >= 1, `certificate-alerts: ${alerts.msg}`)
  const vp = await req('GET', '/purchase/analytics/volume-price', undefined, token)
  assert(vp.code === 1 && (vp.data?.list || []).length >= 1, `volume-price: ${vp.msg}`)
  const spl = await req('GET', '/purchase/analytics/supplier-performance', undefined, token)
  assert(spl.code === 1 && (spl.data?.list || []).length >= 1, `supplier-performance: ${spl.msg}`)
  log('P2 analytics')

  const badFin = await req('POST', '/finance/prepay-prepaids', {
    party_type: 'supplier', party_id: 999999, amount: 100,
  }, token)
  assert(badFin.code !== 1, 'finance should reject unknown supplier')
  const okFin = await req('POST', '/finance/prepay-prepaids', {
    party_type: 'supplier', party_id: sid, amount: 100, doc_no: `PP${Date.now()}`,
  }, token)
  assert(okFin.code === 1, `finance prepaid: ${okFin.msg}`)
  log('finance party guard')

  console.log('\nALL GREEN — e2e-supplier passed')
}

main().catch((e) => {
  console.error('\nFAIL', e.message || e)
  process.exit(1)
})
