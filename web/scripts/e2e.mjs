/**
 * E2E loop: login → 4 business loops → assert → logout (401)
 * Usage: node web/scripts/e2e.mjs
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
    return { status: res.status, code: 0, msg: `NON_JSON:${text.slice(0, 80)}` }
  }
  return { status: res.status, ...data }
}

function assert(cond, msg) {
  if (!cond) throw new Error(msg)
}

async function main() {
  const steps = []
  const log = (m) => { steps.push(m); console.log('✓', m) }

  // 1 login
  const login = await req('POST', '/auth/login', { login_name: 'admin', password: 'admin123', client_type: 'web' })
  assert(login.code === 1 && login.data?.access_token, `login failed: ${login.msg}`)
  const token = login.data.access_token
  log('login')

  const me = await req('GET', '/auth/me', undefined, token)
  assert(me.code === 1, `me failed: ${me.msg}`)
  log('me')

  // ensure product
  let productId
  const products = await req('GET', '/product/products', undefined, token)
  if (products.data?.list?.length) productId = products.data.list[0].id
  else {
    const c = await req('POST', '/product/products', { code: `E2E${Date.now()}`, name: 'E2E料', product_type: 'raw' }, token)
    assert(c.code === 1, `create product: ${c.msg}`)
    productId = c.data.id
  }
  log(`product #${productId}`)

  // --- piecework loop ---
  const wr = await req('POST', '/payroll/wage-rates', { process_id: 1, rate: 3, effective_from: '2026-01-01' }, token)
  if (wr.code !== 1) {
    // may already exist — ensure at least one rate
    const wlist = await req('GET', '/payroll/wage-rates', undefined, token)
    assert(wlist.code === 1 && (wlist.data?.list?.length || 0) > 0, `wage-rates: ${wr.msg}`)
  }
  let txn = await req('POST', '/inventory/stock-txns', {
    doc_no: `STE2E${Date.now()}A`,
    doc_type: 'purchase_in', warehouse_id: 1,
    lines: [{ product_id: productId, qty: 50, direction: 'in' }],
  }, token)
  assert(txn.code === 1, `txn create: ${txn.msg}`)
  let post = await req('POST', `/inventory/stock-txns/${txn.data.id}/post`, {}, token)
  assert(post.code === 1 && post.data?.status === 'posted', `txn post: ${post.msg}`)
  log('stock posted')

  const task = await req('POST', '/production/tasks', { doc_no: `E2EPT${Date.now()}` }, token)
  assert(task.code === 1, `task: ${task.msg}`)
  const disp = await req('POST', '/production/dispatches', {
    task_id: task.data.id, process_id: 1, worker_id: 1, qty: 8,
  }, token)
  assert(disp.code === 1, `dispatch: ${disp.msg}`)
  const reqn = await req('POST', '/production/requisitions', {
    warehouse_id: 1, txn_type: 'consume',
    lines: [{ product_id: productId, qty: 2 }],
  }, token)
  assert(reqn.code === 1, `requisition: ${reqn.msg}`)
  const reqPost = await req('POST', `/production/requisitions/${reqn.data.id}/post`, {}, token)
  assert(reqPost.code === 1, `requisition post: ${reqPost.msg}`)
  const settle = await req('POST', '/production/piecework-summaries/day-settle', {
    biz_date: new Date().toISOString().slice(0, 10),
  }, token)
  assert(settle.code === 1, `day-settle: ${settle.msg}`)
  const sheet = await req('POST', '/payroll/sheets', { remark: 'e2e', period: '2026-08' }, token)
  assert(sheet.code === 1, `payroll sheet: ${sheet.msg}`)
  const batch = await req('POST', '/payroll/sheets/batch-generate', { period: '2026-08' }, token)
  assert(batch.code === 1, `payroll batch: ${batch.msg}`)
  log(`piecework loop day-settle rows=${settle.data?.settled_rows ?? 0}`)

  // --- purchase loop ---
  const sup = await req('POST', '/purchase/suppliers', {
    code: `E2ES${Date.now()}`, name: 'E2E供应商', status: 'qualified', supplier_type: 'raw',
  }, token)
  assert(sup.code === 1, `supplier: ${sup.msg}`)
  const pr = await req('POST', '/purchase/requests', { title: 'E2E采购', qty: 10, supplier_id: sup.data.id }, token)
  assert(pr.code === 1 && pr.data?.id, `purchase request: ${pr.msg}`)
  const ib = await req('POST', '/purchase/inbounds', {
    qty: 10, supplier_id: sup.data.id, product_id: productId, price: 2.5, warehouse_id: 1,
  }, token)
  assert(ib.code === 1 && ib.data?.id, `inbound: ${ib.msg} ${JSON.stringify(ib)}`)
  const ibPost = await req('POST', `/purchase/inbounds/${ib.data.id}/post`, {}, token)
  assert(ibPost.code === 1 && ibPost.data?.status === 'posted', `inbound post: ${ibPost.msg}`)
  const qc = await req('POST', '/purchase/incoming-qcs', { inbound_id: ib.data.id, product_id: productId, qty_check: 10 }, token)
  assert(qc.code === 1, `qc: ${qc.msg}`)
  const qcPass = await req('POST', `/purchase/incoming-qcs/${qc.data.id}/pass`, {}, token)
  assert(qcPass.code === 1, `qc pass: ${qcPass.msg}`)
  log('purchase inbound loop')

  // --- sales loop ---
  const inq = await req('POST', '/sales/inquiries', { customer: 'E2E客户', product: '演示', qty: 2 }, token)
  assert(inq.code === 1, `inquiry: ${inq.msg}`)
  txn = await req('POST', '/inventory/stock-txns', {
    doc_no: `E2EST${Date.now()}C`,
    doc_type: 'purchase_in', warehouse_id: 3,
    lines: [{ product_id: productId, qty: 20, direction: 'in' }],
  }, token)
  assert(txn.code === 1 && txn.data?.id, `sales stock txn: ${txn.msg}`)
  post = await req('POST', `/inventory/stock-txns/${txn.data.id}/post`, {}, token)
  assert(post.code === 1, `sales stock post: ${post.msg}`)
  const so = await req('POST', '/sales/orders', {
    customer: 'E2E客户', warehouse_id: 3,
    lines: [{ product_id: productId, qty: 2 }],
  }, token)
  assert(so.code === 1 && so.data?.id, `sales order: ${so.msg} ${JSON.stringify(so.data)}`)
  const pre = await req('POST', '/sales/pre-shipments', {
    order_id: so.data.id, warehouse_id: 3, txn_type: 'sale_out',
    lines: [{ product_id: productId, qty: 2 }],
  }, token)
  assert(pre.code === 1 && pre.data?.id, `pre-ship: ${pre.msg}`)
  const conf = await req('POST', `/sales/pre-shipments/${pre.data.id}/confirm`, {
    warehouse_id: 3, txn_type: 'sale_out',
    lines: [{ product_id: productId, qty: 2 }],
  }, token)
  assert(conf.code === 1, `pre-ship confirm: ${conf.msg}`)
  const wo = await req('POST', '/finance/receipt-writeoffs', { order_id: so.data.id, amount: 99 }, token)
  assert(wo.code === 1, `writeoff: ${wo.msg}`)
  log('sales outbound loop')

  // --- perm loop ---
  const perms = await req('GET', '/iam/permissions', undefined, token)
  assert(perms.code === 1, `permissions: ${perms.msg}`)
  const users = await req('GET', '/iam/users', undefined, token)
  assert(users.code === 1 && users.data?.list?.length, 'users empty')
  const target = users.data.list.find((u) => u.login_name !== 'admin') || users.data.list[0]
  await req('PUT', `/iam/users/${target.id}/roles`, { role_ids: [1] }, token)
  await req('POST', `/iam/users/${target.id}/freeze`, {}, token)
  await req('POST', `/iam/users/${target.id}/unfreeze`, {}, token)
  await req('GET', '/system/operation-logs', undefined, token)
  log('perm loop')

  // logout = clear token client-side; request without token should 401 on /me
  const unauth = await req('GET', '/auth/me', undefined, '')
  assert(unauth.status === 401 || unauth.msg === 'UNAUTHORIZED', 'expected UNAUTHORIZED after logout')
  log('logout → UNAUTHORIZED')

  console.log('\nE2E OK', steps.length, 'steps')
}

main().catch((e) => {
  console.error('\nE2E FAIL:', e.message)
  process.exit(1)
})
