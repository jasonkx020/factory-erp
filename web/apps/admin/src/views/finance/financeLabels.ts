export const FIN_STATUS: Record<string, string> = {
  draft: '草稿',
  submitted: '已提交',
  pending: '待审',
  approved: '已审批',
  rejected: '已驳回',
  posted: '已过账',
  confirmed: '已确认',
  open: '未结',
  closed: '已结转',
  active: '启用',
  unpaid: '未付',
  reconciled: '已对账',
  revoked: '已撤销',
  calculated: '已核算',
  handled: '已处理',
}

export const FIN_STATUS_TYPE: Record<string, 'success' | 'warning' | 'info' | 'danger' | 'primary'> = {
  draft: 'info',
  submitted: 'warning',
  pending: 'warning',
  approved: 'success',
  rejected: 'danger',
  posted: 'success',
  confirmed: 'success',
  open: 'warning',
  closed: 'success',
  active: 'success',
  unpaid: 'warning',
  reconciled: 'success',
  revoked: 'info',
  calculated: 'success',
  handled: 'success',
}

export const FIN_DIRECTION: Record<string, string> = {
  in: '收入',
  out: '支出',
  increase: '调增',
  decrease: '调减',
}

export const FIN_SUBJECT_TYPE: Record<string, string> = {
  asset: '资产',
  liability: '负债',
  income: '收入',
  expense: '费用',
}

export const FIN_PARTY_TYPE: Record<string, string> = {
  customer: '客户',
  supplier: '供应商',
  farmer: '农户',
}

export const FIN_INVOICE_DIR: Record<string, string> = {
  out: '销项',
  in: '进项',
}

export const FIN_HINT: Record<string, string> = {
  subjects: '维护现金、银行、应收应付等科目，供凭证选用。',
  ledger: '登记资金进出并同步账户余额；已月结期间不可登记。',
  'income-expenses': '由流水自动生成的收支明细，只读查询。',
  orders: '财务视角销售订单（只读）。收款请到收款核单。',
  miniprogram: '小程序账单（本厂日常少用）。',
  vouchers: '借贷须平衡后审批、过账。过账不另造总账流水。',
  invoices: '进项/销项发票登记。',
  writeoffs: '客户回款核销到订单，确认后入资金账户。',
  fx: '外币结汇（本厂日常少用）。',
  'fx-query': '已确认结汇查询。',
  allocations: '制造费用分摊；撤销生成反向记录。',
  alerts: '客户逾期收款跟进。差额处理须填备注。',
  reconciles: '出纳账面与实盘核对，差额须填备注后确认。',
  prepays: '预收客户或预付供应商/农户，登记即入账。',
  'cost-accountings': '按期间汇入农户已付货款与计件日结，生成可审计成本单。',
  'contract-profits': '按合同或期间出厂汇总收入与成本，禁止全厂塞进一张单。',
  recognitions: '销售认款。若该客户已有核单入账，认款不再重复加余额。',
  'return-finances': '销售退货退款，确认后从资金账户扣出。',
  arap: '往来应收应付调整，过账不动现金。',
  approvals: '待审凭证与审批单；按来源分别批准，避免串单。',
  funds: '现金/银行账户余额与内部调拨；农户付款从账户扣减。',
  ledger: '资金进出流水（含农户货款支付）；可手工登记其他收支。',
  payables: '待付农户结算单：选资金账户、填转账单号与回单后支付关单。',
  statements: '由已过账凭证生成三大表；无凭证时回退流水。',
  'cost-traces': '成本核算来源明细（农户货款/计件/领料），可跳转对账。',
  'month-closes': '结转后该月禁止流水、核单、调拨、农户付款。',
}

export function finStatusLabel(v: unknown) {
  const s = String(v || '')
  return FIN_STATUS[s] || s || '-'
}

export function finStatusType(v: unknown) {
  return FIN_STATUS_TYPE[String(v || '')] || 'info'
}

export function finDirLabel(v: unknown) {
  const s = String(v || '')
  return FIN_DIRECTION[s] || FIN_INVOICE_DIR[s] || s || '-'
}

export function money(v: unknown) {
  const n = Number(v)
  if (!Number.isFinite(n)) return '-'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function finPartyLabel(v: unknown) {
  const s = String(v || '')
  return FIN_PARTY_TYPE[s] || s || '-'
}

export function finSubjectTypeLabel(v: unknown) {
  const s = String(v || '')
  return FIN_SUBJECT_TYPE[s] || s || '-'
}

export const FIN_ERR: Record<string, string> = {
  PERIOD_CLOSED: '该期间已月结，不可入账',
  FUND_ACCOUNT_REQUIRED: '请选择资金账户',
  VOUCHER_LINE_BOTH_SIDES: '同一分录不能同时填借贷',
  VOUCHER_SAME_SUBJECT: '借贷须使用不同科目，不可同一科目对倒',
  VOUCHER_UNBALANCED: '凭证借贷不平衡',
  VOUCHER_EMPTY: '凭证没有分录',
  RECONCILE_DIFF_REMARK_REQUIRED: '对账有差额，请填写备注',
  UNKNOWN_APPROVAL_SOURCE: '审批来源无效，请刷新后重试',
  INVALID_TRANSFER: '调拨账户无效',
  WRITEOFF_OVER_AMOUNT: '核销金额超出单据',
  WRITEOFF_LINE_EXCEEDS: '核销行金额超出',
}

export function finErrMsg(msg: string) {
  if (FIN_ERR[msg]) return FIN_ERR[msg]
  if (msg.startsWith('DB_ERROR')) return '保存失败，请稍后重试'
  return msg || '操作失败'
}
