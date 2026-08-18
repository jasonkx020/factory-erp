export type SalesRow = Record<string, unknown>

export const STATUS_LABEL: Record<string, string> = {
  draft: '草稿',
  pending: '待审批',
  submitted: '已提交',
  approved: '已通过',
  rejected: '已驳回',
  ordered: '已转订单',
  reserved: '已占用',
  confirmed: '已确认',
  cancelled: '已取消',
  shipped: '已发货',
  received: '已签收',
  closed: '已关单',
  active: '生效',
  inactive: '停用',
  open: '进行中',
}

export function statusLabel(v: unknown) {
  const s = String(v || '')
  return STATUS_LABEL[s] || s || '—'
}

export function statusType(v: unknown): 'success' | 'warning' | 'danger' | 'info' {
  const s = String(v || '')
  if (['approved', 'active', 'shipped', 'received', 'ordered', 'confirmed', 'closed'].includes(s)) return 'success'
  if (['pending', 'submitted', 'reserved', 'draft', 'open'].includes(s)) return 'warning'
  if (['rejected', 'cancelled', 'inactive'].includes(s)) return 'danger'
  return 'info'
}

export function money(v: unknown) {
  const n = Number(v || 0)
  return Number.isFinite(n) ? n.toFixed(2) : '0.00'
}

export function dash(v: unknown) {
  const s = String(v ?? '').trim()
  return s || '—'
}
