<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { financeApi, purchaseApi, CURRENCY_OPTIONS, PAY_CHANNEL_OPTIONS, useAuthStore } from '@erp/shared'
import {
  CustomerSelect,
  SupplierSelect,
  SalesOrderSelect,
  ProductSelect,
  EnumSelect,
} from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import {
  FIN_HINT,
  finDirLabel,
  finErrMsg,
  finPartyLabel,
  finStatusLabel,
  finStatusType,
  finSubjectTypeLabel,
  money,
} from './financeLabels'

type Row = Record<string, unknown>

const statusCol = (prop = 'status'): MobileCardColumn => ({
  prop,
  label: '状态',
  format: (v) => finStatusLabel(v),
})
const moneyCol = (prop: string, label: string): MobileCardColumn => ({
  prop,
  label,
  format: (v) => money(v),
})

const subjectCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'subject_type', label: '类型', format: (v) => finSubjectTypeLabel(v) },
  statusCol(),
]
const ledgerCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'direction', label: '方向', format: (v) => finDirLabel(v) },
  moneyCol('amount', '金额'),
  { prop: 'account_name', label: '账户' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'counterparty', label: '对方' },
  { prop: 'remark', label: '备注' },
]
const incomeExpenseCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '流水号', primary: true },
  { prop: 'category', label: '类别' },
  { prop: 'direction', label: '方向', format: (v) => finDirLabel(v) },
  moneyCol('amount', '金额'),
  { prop: 'biz_date', label: '日期' },
  { prop: 'counterparty', label: '对方' },
]
const orderCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'customer_name', label: '客户' },
  statusCol(),
  moneyCol('total_amount', '金额'),
  moneyCol('received_amount', '已收'),
  { prop: 'created_at', label: '创建时间' },
]
const mpCols: MobileCardColumn[] = [
  { prop: 'bill_no', label: '账单号', primary: true },
  { prop: 'channel', label: '渠道' },
  moneyCol('amount', '金额'),
  statusCol(),
  { prop: 'paid_at', label: '支付/对账时间' },
]
const voucherCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '凭证号', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'summary', label: '摘要' },
  statusCol(),
]
const invoiceCols: MobileCardColumn[] = [
  { prop: 'invoice_no', label: '票号', primary: true },
  { prop: 'direction', label: '方向', format: (v) => finDirLabel(v) },
  { prop: 'counterparty_name', label: '对方' },
  moneyCol('amount', '金额'),
  moneyCol('tax', '税额'),
  statusCol(),
]
const writeoffCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'customer_name', label: '客户' },
  moneyCol('amount', '金额'),
  { prop: 'account_name', label: '账户' },
  statusCol(),
]
const recognitionCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'customer_name', label: '客户' },
  moneyCol('amount', '金额'),
  { prop: 'account_name', label: '账户' },
  statusCol(),
]
const fxCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'currency', label: '币种' },
  moneyCol('amount_fx', '外币'),
  { prop: 'rate', label: '汇率' },
  moneyCol('amount_local', '本币'),
  statusCol(),
]
const allocCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  moneyCol('source_amount', '金额'),
  statusCol(),
]
const alertCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'overdue_days', label: '逾期天' },
  moneyCol('amount', '金额'),
  statusCol(),
]
const reconcileCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'account_name', label: '账户' },
  moneyCol('book_balance', '账面'),
  moneyCol('actual_balance', '实盘'),
  moneyCol('diff', '差额'),
  statusCol(),
]
const prepayCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'party_type', label: '类型', format: (v) => finPartyLabel(v) },
  { prop: 'party_name', label: '对方' },
  moneyCol('amount', '金额'),
  moneyCol('balance', '余额'),
  statusCol(),
]
const costCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'product_name', label: '产品' },
  moneyCol('material_cost', '物料'),
  moneyCol('labor_cost', '人工'),
  moneyCol('overhead', '制造'),
  moneyCol('total_cost', '合计'),
  statusCol(),
]
const contractProfitCols: MobileCardColumn[] = [
  { prop: 'contract_no', label: '合同', primary: true },
  { prop: 'customer_name', label: '客户' },
  moneyCol('revenue', '收入'),
  moneyCol('cost', '成本'),
  moneyCol('profit', '利润'),
  { prop: 'period', label: '期间' },
]
const returnCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'order_no', label: '订单' },
  moneyCol('amount', '金额'),
  { prop: 'account_name', label: '账户' },
  statusCol(),
]
const arapCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'party_type', label: '类型', format: (v) => finPartyLabel(v) },
  { prop: 'party_name', label: '对方' },
  moneyCol('amount', '金额'),
  { prop: 'direction', label: '方向', format: (v) => finDirLabel(v) },
  statusCol(),
]
const approvalCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'source', label: '来源', format: (v) => (String(v) === 'approval_item' ? '审批单' : '凭证') },
  { prop: 'biz_type', label: '类型' },
  { prop: 'title', label: '摘要' },
  statusCol(),
]
const fundAccountCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'currency', label: '币种' },
  moneyCol('balance', '余额'),
  statusCol(),
]
const fundTransferCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'from_account_name', label: '转出' },
  { prop: 'to_account_name', label: '转入' },
  moneyCol('amount', '金额'),
  statusCol(),
]
const statementCols: MobileCardColumn[] = [
  { prop: 'code', label: '报表', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'title', label: '标题' },
  { prop: 'generated_at', label: '生成时间' },
]
const costTraceCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '成本单', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'source_type', label: '来源类型' },
  { prop: 'source_id', label: '来源ID' },
  moneyCol('amount', '金额'),
]
const monthCloseCols: MobileCardColumn[] = [
  { prop: 'year', label: '年', primary: true },
  { prop: 'month', label: '月' },
  statusCol(),
  { prop: 'closed_at', label: '结转时间' },
]


const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const TITLE_MAP: Record<string, string> = {
  subjects: '账目管理',
  ledger: '交易流水账',
  'income-expenses': '收入支出明细',
  orders: '订单管理',
  miniprogram: '小程序管理',
  vouchers: '凭证管理',
  invoices: '发票管理',
  writeoffs: '收款核单',
  fx: '外币结汇',
  'fx-query': '结汇查询',
  allocations: '分摊撤销',
  alerts: '收款预警',
  reconciles: '出纳对账',
  prepays: '预收预付管理',
  'cost-accountings': '成本核算',
  'contract-profits': '合同利润',
  recognitions: '销售认款',
  'return-finances': '销售退货退单',
  arap: '往来调整单',
  approvals: '财务审批',
  funds: '资金管理',
  statements: '财务报表',
  'cost-traces': '成本明细溯源表',
  'month-closes': '月度结转',
}

const active = computed(() => String(route.params.section || 'vouchers'))
const title = computed(() => TITLE_MAP[active.value] || '财务管理')
const hint = computed(() => FIN_HINT[active.value] || '身份与资金配置。')
const loading = ref(false)
const list = ref<Row[]>([])
const funds = ref<Row[]>([])
const subjects = ref<Row[]>([])
const farmers = ref<Row[]>([])
const fundTab = ref<'accounts' | 'transfers'>('accounts')
const fundCols = computed(() => (fundTab.value === 'accounts' ? fundAccountCols : fundTransferCols))
const statementPreview = ref<Row | null>(null)
const keyword = ref('')
const farmerPending = ref(0)
const farmerPendingAmt = ref(0)

const filteredList = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list.value
  return list.value.filter((row) =>
    Object.values(row).some((v) => String(v ?? '').toLowerCase().includes(kw)),
  )
})

const showCost = computed(() => auth.fieldVisible('cost'))
const showProfit = computed(() => auth.fieldVisible('gross_profit'))

const stats = computed(() => {
  const rows = filteredList.value
  const sum = (k: string) => rows.reduce((s, r) => s + (Number(r[k]) || 0), 0)
  const countStatus = (st: string) => rows.filter((r) => String(r.status) === st).length
  if (active.value === 'ledger' || active.value === 'income-expenses') {
    const inn = rows.filter((r) => r.direction === 'in').reduce((s, r) => s + (Number(r.amount) || 0), 0)
    const out = rows.filter((r) => r.direction === 'out').reduce((s, r) => s + (Number(r.amount) || 0), 0)
    return [
      { label: '笔数', value: String(rows.length) },
      { label: '收入', value: money(inn), ok: true },
      { label: '支出', value: money(out), warn: true },
      { label: '净额', value: money(inn - out) },
    ]
  }
  if (active.value === 'funds' && fundTab.value === 'accounts') {
    return [
      { label: '账户数', value: String(rows.length) },
      { label: '余额合计', value: money(sum('balance')), ok: true },
      { label: '待付农户', value: String(farmerPending.value), warn: farmerPending.value > 0 },
      { label: '待付金额', value: money(farmerPendingAmt.value), warn: farmerPendingAmt.value > 0 },
    ]
  }
  if (active.value === 'writeoffs' || active.value === 'recognitions' || active.value === 'prepays') {
    return [
      { label: '单据', value: String(rows.length) },
      { label: '金额', value: money(sum('amount')) },
      { label: '待确认', value: String(countStatus('draft') + countStatus('open')), warn: true },
      { label: '待付农户', value: String(farmerPending.value), warn: farmerPending.value > 0 },
    ]
  }
  if (active.value === 'alerts') {
    return [
      { label: '预警', value: String(rows.length) },
      { label: '未处理', value: String(countStatus('open')), warn: true },
      { label: '逾期金额', value: money(sum('amount')), warn: true },
    ]
  }
  if (active.value === 'contract-profits') {
    return [
      { label: '记录', value: String(rows.length) },
      { label: '收入', value: money(sum('revenue')), ok: true },
      { label: '成本', value: showCost.value ? money(sum('cost')) : '***' },
      { label: '利润', value: showProfit.value ? money(sum('profit')) : '***' },
    ]
  }
  return [
    { label: '记录', value: String(rows.length) },
    { label: '待处理', value: String(countStatus('draft') + countStatus('pending') + countStatus('open') + countStatus('submitted')), warn: true },
    { label: '资金账户', value: String(funds.value.length) },
    { label: '待付农户', value: String(farmerPending.value), warn: farmerPending.value > 0 },
  ]
})

function goFarmerSettle() {
  router.push('/purchase/hub/settlements')
}

const subjectForm = reactive({ code: '', name: '', subject_type: 'asset' })
const ledgerForm = reactive({
  direction: 'in',
  amount: 100,
  account_id: 1,
  subject_id: 1,
  counterparty: '',
  category: '经营收支',
  remark: '',
})
const voucherForm = reactive({
  summary: '',
  debit_subject_id: 1,
  credit_subject_id: 1,
  amount: 100,
  period: new Date().toISOString().slice(0, 7),
})
const invoiceForm = reactive({
  direction: 'out',
  invoice_no: '',
  counterparty_name: '',
  amount: 1000,
  tax: 130,
})
const writeoffForm = reactive({ customer_id: 1, amount: 1000, fund_account_id: 1, sales_order_id: 0 })
const recognitionForm = reactive({ customer_id: 1, amount: 1000, fund_account_id: 1, remark: '' })
const fxForm = reactive({ currency: 'USD', amount_fx: 1000, rate: 7.2, fund_account_id: 1 })
const allocForm = reactive({ source_amount: 1000 })
const alertForm = reactive({ customer_id: 1, overdue_days: 30, amount: 5000, due_date: '' })
const reconcileForm = reactive({ fund_account_id: 1, actual_balance: 0 })
const prepayForm = reactive({
  party_type: 'customer',
  party_id: 1,
  direction: 'in',
  amount: 1000,
  fund_account_id: 1,
})
const costForm = reactive({
  period: new Date().toISOString().slice(0, 7),
  product_id: 1,
  task_id: 0,
  material_cost: 0,
  labor_cost: 0,
  overhead: 0,
})
const returnForm = reactive({ order_id: 0, amount: 500, fund_account_id: 1 })
const arapForm = reactive({
  party_type: 'customer',
  party_id: 1,
  amount: 100,
  direction: 'increase',
  remark: '',
})
const fundAccForm = reactive({ code: '', name: '', currency: 'CNY', balance: 0 })
const fundTfForm = reactive({ from_account_id: 1, to_account_id: 2, amount: 100, remark: '' })
const monthForm = reactive({
  year: new Date().getFullYear(),
  month: new Date().getMonth() + 1,
})
const mpForm = reactive({ bill_no: '', channel: 'wechat', amount: 100, order_id: 0 })

async function loadMeta() {
  const [f, s] = await Promise.all([financeApi.fundAccounts(), financeApi.subjects()])
  funds.value = ((f.data as { list?: Row[] })?.list) || []
  subjects.value = ((s.data as { list?: Row[] })?.list) || []
  try {
    const fr = await purchaseApi.farmers()
    farmers.value = ((fr.data as { list?: Row[] })?.list) || []
  } catch {
    farmers.value = []
  }
  if (funds.value[0]) {
    const id = Number(funds.value[0].id)
    ledgerForm.account_id = id
    writeoffForm.fund_account_id = id
    recognitionForm.fund_account_id = id
    fxForm.fund_account_id = id
    reconcileForm.fund_account_id = id
    prepayForm.fund_account_id = id
    returnForm.fund_account_id = id
    fundTfForm.from_account_id = id
    if (funds.value[1]) fundTfForm.to_account_id = Number(funds.value[1].id)
  }
  if (subjects.value[0]) {
    const sid = Number(subjects.value[0].id)
    ledgerForm.subject_id = sid
    voucherForm.debit_subject_id = sid
    voucherForm.credit_subject_id = subjects.value[1] ? Number(subjects.value[1].id) : sid
  }
  try {
    const fs = await purchaseApi.farmerSettlements()
    const rows = ((fs.data as { list?: Row[] })?.list) || []
    const pending = rows.filter((r) => {
      const st = String(r.status || '')
      return st !== 'settle_paid' && st !== 'paid' && st !== 'void'
    })
    farmerPending.value = pending.length
    farmerPendingAmt.value = pending.reduce((s, r) => s + (Number(r.amount) || 0), 0)
  } catch {
    farmerPending.value = 0
    farmerPendingAmt.value = 0
  }
}

async function refresh() {
  loading.value = true
  try {
    let res
    switch (active.value) {
      case 'subjects':
        res = await financeApi.subjects()
        break
      case 'ledger':
        res = await financeApi.ledger()
        break
      case 'income-expenses':
        res = await financeApi.incomeExpenses()
        break
      case 'orders':
        res = await financeApi.orders()
        break
      case 'miniprogram':
        res = await financeApi.miniprogramBills()
        break
      case 'vouchers':
        res = await financeApi.vouchers()
        break
      case 'invoices':
        res = await financeApi.invoices()
        break
      case 'writeoffs':
        res = await financeApi.writeoffs()
        break
      case 'fx':
        res = await financeApi.fxSettlements()
        break
      case 'fx-query':
        res = await financeApi.fxQuery()
        break
      case 'allocations':
        res = await financeApi.allocations()
        break
      case 'alerts':
        res = await financeApi.alerts()
        break
      case 'reconciles':
        res = await financeApi.reconciles()
        break
      case 'prepays':
        res = await financeApi.prepays()
        break
      case 'cost-accountings':
        res = await financeApi.costAccountings()
        break
      case 'contract-profits':
        res = await financeApi.contractProfits()
        break
      case 'recognitions':
        res = await financeApi.recognitions()
        break
      case 'return-finances':
        res = await financeApi.returnFinances()
        break
      case 'arap':
        res = await financeApi.arapAdjusts()
        break
      case 'approvals':
        res = await financeApi.approvals()
        break
      case 'funds':
        res = fundTab.value === 'accounts' ? await financeApi.fundAccounts() : await financeApi.fundTransfers()
        break
      case 'statements':
        res = await financeApi.statements()
        break
      case 'cost-traces':
        res = await financeApi.costTraces()
        break
      case 'month-closes':
        res = await financeApi.monthCloses()
        break
      default:
        res = await financeApi.vouchers()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
    if (active.value === 'funds' && fundTab.value === 'accounts') funds.value = list.value
  } finally {
    loading.value = false
  }
}

async function run(fn: () => Promise<{ code: number; msg: string }>, ok = '成功') {
  const res = await fn()
  if (res.code !== 1) {
    return ElMessage.error(finErrMsg(res.msg))
  }
  ElMessage.success(ok)
  await refresh()
  await loadMeta()
}

async function createVoucher() {
  if (voucherForm.debit_subject_id === voucherForm.credit_subject_id) {
    return ElMessage.warning('借贷须使用不同科目，不可同一科目对倒')
  }
  if (voucherForm.amount <= 0) return ElMessage.warning('金额须大于 0')
  await run(
    () =>
      financeApi.createVoucher({
        summary: voucherForm.summary,
        period: voucherForm.period,
        lines: [
          { subject_id: voucherForm.debit_subject_id, debit: voucherForm.amount, credit: 0 },
          { subject_id: voucherForm.credit_subject_id, debit: 0, credit: voucherForm.amount },
        ],
      }),
    '凭证已建',
  )
}

async function confirmReconcileRow(row: Row) {
  const diff = Math.abs(Number(row.diff ?? Number(row.actual_balance) - Number(row.book_balance)))
  let remark = String(row.remark || '')
  if (diff > 0.01) {
    try {
      const { value } = await ElMessageBox.prompt(
        `账面与实盘差额 ${money(row.diff ?? Number(row.actual_balance) - Number(row.book_balance))}，确认前须填写备注（本次不自动调余额）`,
        '对账差额说明',
        { inputValue: remark, inputPlaceholder: '差额原因', type: 'warning' },
      )
      remark = String(value || '').trim()
      if (!remark) return ElMessage.warning('对账有差额，请填写备注')
    } catch {
      return
    }
  }
  await run(() => financeApi.confirmReconcile(Number(row.id), { remark }), '已确认')
}

async function reopenMonthConfirm(row: Row) {
  try {
    await ElMessageBox.confirm(
      `反结转 ${row.year}-${row.month} 后该月可再次入账。确认？`,
      '反结转',
      { type: 'warning' },
    )
  } catch {
    return
  }
  await run(() => financeApi.reopenMonth(Number(row.id)), '已反结转')
}

async function generateStatements() {
  const r = await financeApi.generateStatements({})
  if (r.code !== 1) return ElMessage.error(finErrMsg(r.msg))
  statementPreview.value = r.data as Row
  ElMessage.success('已生成')
  await refresh()
}

async function closeMonthConfirm() {
  try {
    await ElMessageBox.confirm(
      `结转 ${monthForm.year}-${monthForm.month} 后，该月流水/核单/调拨/农户付款将不可入账。确认？`,
      '月度结转',
      { type: 'warning' },
    )
  } catch {
    return
  }
  await run(() => financeApi.closeMonth({ ...monthForm }), '期间已结转')
}

onMounted(async () => {
  await loadMeta()
  await refresh()
})
watch([active, fundTab], refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <header class="page-head">
      <div>
        <h2 class="title">{{ title }}</h2>
        <p class="desc">{{ hint }}</p>
      </div>
      <div class="head-actions">
        <el-button
          v-if="['funds','ledger','prepays','writeoffs'].includes(active)"
          type="warning"
          plain
          @click="goFarmerSettle"
        >待付农户 {{ farmerPending }}</el-button>
        <el-button @click="refresh">刷新</el-button>
      </div>
    </header>
    <div class="stats">
      <div v-for="s in stats" :key="s.label" class="stat" :class="{ ok: s.ok, warn: s.warn }">
        <div class="label">{{ s.label }}</div>
        <div class="value">{{ s.value }}</div>
      </div>
    </div>
    <div class="toolbar">
      <el-input v-model="keyword" clearable placeholder="筛选单号 / 名称 / 状态" style="width:220px" />
    </div>

    <!-- 账目 -->
    <template v-if="active === 'subjects'">
      <el-card class="mb" header="新建会计科目">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="subjectForm.code" style="width:120px" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="subjectForm.name" style="width:160px" /></el-form-item>
          <el-form-item label="类型">
            <el-select v-model="subjectForm.subject_type" style="width:110px">
              <el-option label="资产" value="asset" /><el-option label="负债" value="liability" />
              <el-option label="收入" value="income" /><el-option label="费用" value="expense" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createSubject({ ...subjectForm }), '已建科目')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="subjectCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="code" label="编码" width="100" /><el-table-column prop="name" label="名称" />
          <el-table-column label="类型" width="100">
            <template #default="{ row }">{{ finSubjectTypeLabel(row.subject_type) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <!-- 流水 -->
    <template v-else-if="active === 'ledger'">
      <el-card class="mb" header="登记流水（同步更新资金账户余额）">
        <el-form inline size="small">
          <el-form-item label="方向">
            <el-select v-model="ledgerForm.direction" style="width:90px"><el-option label="收入" value="in" /><el-option label="支出" value="out" /></el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="ledgerForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="ledgerForm.account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="对方"><el-input v-model="ledgerForm.counterparty" style="width:120px" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createLedger({ ...ledgerForm }), '流水已登记')">登记</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="ledgerCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column label="方向" width="80">
            <template #default="{ row }">{{ finDirLabel(row.direction) }}</template>
          </el-table-column>
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column prop="account_name" label="账户" width="120" />
          <el-table-column prop="biz_date" label="日期" width="110" />
          <el-table-column prop="counterparty" label="对方" /><el-table-column prop="remark" label="备注" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'income-expenses'">
      <TableOrCards :data="filteredList" :loading="loading" :columns="incomeExpenseCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="流水号" width="150" /><el-table-column prop="category" label="类别" width="120" />
          <el-table-column label="方向" width="80">
            <template #default="{ row }">{{ finDirLabel(row.direction) }}</template>
          </el-table-column>
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column prop="biz_date" label="日期" width="110" /><el-table-column prop="counterparty" label="对方" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'orders'">
      <el-alert class="mb" type="info" :closable="false" title="财务视角销售订单（只读）；业务办理请到销售管理" />
      <TableOrCards :data="filteredList" :loading="loading" :columns="orderCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="customer_name" label="客户" min-width="120" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.total_amount) }}</template></el-table-column>
          <el-table-column label="已收" width="110"><template #default="{ row }">{{ money(row.received_amount) }}</template></el-table-column>
          <el-table-column prop="created_at" label="创建时间" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'miniprogram'">
      <el-card class="mb" header="小程序账单">
        <el-form inline size="small">
          <el-form-item label="单号"><el-input v-model="mpForm.bill_no" style="width:140px" placeholder="可空" /></el-form-item>
          <el-form-item label="渠道"><EnumSelect v-model="mpForm.channel" :options="PAY_CHANNEL_OPTIONS" :clearable="false" style="width:120px" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="mpForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="订单"><SalesOrderSelect v-model="mpForm.order_id" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createMiniprogramBill({ ...mpForm }), '已建账单')">新建</el-button>
          <el-button @click="run(() => financeApi.reconcileMiniprogram({}), '已对账未付款账单')">一键对账</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="mpCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="bill_no" label="账单号" width="150" /><el-table-column prop="channel" label="渠道" width="100" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="paid_at" label="支付/对账时间" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='unpaid'" link type="primary" @click="run(() => financeApi.reconcileMiniprogram({ id: Number(row.id) }), '已对账')">对账</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='unpaid'" link type="primary" @click="run(() => financeApi.reconcileMiniprogram({ id: Number(row.id) }), '已对账')">对账</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'vouchers'">
      <el-card class="mb" header="新建凭证（借贷需平衡后可审批）">
        <el-form inline size="small">
          <el-form-item label="摘要"><el-input v-model="voucherForm.summary" style="width:180px" /></el-form-item>
          <el-form-item label="期间"><el-date-picker v-model="voucherForm.period" type="month" value-format="YYYY-MM" style="width:130px" /></el-form-item>
          <el-form-item label="借方科目">
            <el-select v-model="voucherForm.debit_subject_id" style="width:150px">
              <el-option v-for="s in subjects" :key="'d'+s.id" :label="`${s.code} ${s.name}`" :value="Number(s.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="贷方科目">
            <el-select v-model="voucherForm.credit_subject_id" style="width:150px">
              <el-option v-for="s in subjects" :key="'c'+s.id" :label="`${s.code} ${s.name}`" :value="Number(s.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="voucherForm.amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="createVoucher">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="voucherCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="凭证号" width="150" /><el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="biz_date" label="日期" width="110" /><el-table-column prop="summary" label="摘要" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="success" @click="run(() => financeApi.approveVoucher(Number(row.id)), '已审批')">审批</el-button>
              <el-button v-if="row.status==='approved' || row.status==='draft'" link type="primary" @click="run(() => financeApi.postVoucher(Number(row.id)), '已过账')">过账</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="success" @click="run(() => financeApi.approveVoucher(Number(row.id)), '已审批')">审批</el-button>
          <el-button v-if="row.status==='approved' || row.status==='draft'" link type="primary" @click="run(() => financeApi.postVoucher(Number(row.id)), '已过账')">过账</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'invoices'">
      <el-card class="mb" header="新建发票">
        <el-form inline size="small">
          <el-form-item label="方向">
            <el-select v-model="invoiceForm.direction" style="width:100px"><el-option label="销项" value="out" /><el-option label="进项" value="in" /></el-select>
          </el-form-item>
          <el-form-item label="票号"><el-input v-model="invoiceForm.invoice_no" style="width:140px" /></el-form-item>
          <el-form-item label="对方"><el-input v-model="invoiceForm.counterparty_name" style="width:140px" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="invoiceForm.amount" :min="0" /></el-form-item>
          <el-form-item label="税额"><el-input-number v-model="invoiceForm.tax" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createInvoice({ ...invoiceForm }), '发票已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="invoiceCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="invoice_no" label="票号" width="150" />
          <el-table-column label="方向" width="80"><template #default="{ row }">{{ finDirLabel(row.direction) }}</template></el-table-column>
          <el-table-column prop="counterparty_name" label="对方" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="税额" width="90"><template #default="{ row }">{{ money(row.tax) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'writeoffs'">
      <el-card class="mb" header="收款核单（确认后入资金账户）">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="writeoffForm.customer_id" :clearable="false" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="writeoffForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="writeoffForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="订单"><SalesOrderSelect v-model="writeoffForm.sales_order_id" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createWriteoff({ ...writeoffForm }), '核单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="writeoffCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="customer_name" label="客户" min-width="120" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column prop="account_name" label="账户" width="120" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmWriteoff(Number(row.id)), '已确认入账')">确认</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmWriteoff(Number(row.id)), '已确认入账')">确认</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'recognitions'">
      <el-card class="mb" header="销售认款">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="recognitionForm.customer_id" :clearable="false" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="recognitionForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="recognitionForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createRecognition({ ...recognitionForm }), '认款已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="recognitionCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="customer_name" label="客户" min-width="120" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column prop="account_name" label="账户" width="120" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmRecognition(Number(row.id)), '已认款')">确认</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmRecognition(Number(row.id)), '已认款')">确认</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'fx' || active === 'fx-query'">
      <el-card v-if="active==='fx'" class="mb" header="外币结汇">
        <el-form inline size="small">
          <el-form-item label="币种"><EnumSelect v-model="fxForm.currency" :options="CURRENCY_OPTIONS" :clearable="false" style="width:140px" /></el-form-item>
          <el-form-item label="外币金额"><el-input-number v-model="fxForm.amount_fx" :min="0.01" /></el-form-item>
          <el-form-item label="汇率"><el-input-number v-model="fxForm.rate" :min="0.0001" :step="0.01" /></el-form-item>
          <el-form-item label="入账账户">
            <el-select v-model="fxForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createFx({ ...fxForm }), '结汇单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="fxCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="currency" label="币种" width="80" />
          <el-table-column label="外币" width="100"><template #default="{ row }">{{ money(row.amount_fx) }}</template></el-table-column>
          <el-table-column prop="rate" label="汇率" width="90" />
          <el-table-column label="本币" width="110"><template #default="{ row }">{{ money(row.amount_local) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column v-if="active==='fx'" label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmFx(Number(row.id)), '结汇已确认')">确认</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template v-if="active==='fx'" #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmFx(Number(row.id)), '结汇已确认')">确认</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'allocations'">
      <el-card class="mb" header="费用分摊">
        <el-form inline size="small">
          <el-form-item label="源金额"><el-input-number v-model="allocForm.source_amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createAllocation({ ...allocForm }), '分摊已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="allocCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column label="金额" width="120"><template #default="{ row }">{{ money(row.source_amount) }}</template></el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status!=='revoked'" link type="danger" @click="run(() => financeApi.revokeAllocation(Number(row.id)), '已撤销')">撤销</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status!=='revoked'" link type="danger" @click="run(() => financeApi.revokeAllocation(Number(row.id)), '已撤销')">撤销</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'alerts'">
      <el-card class="mb" header="收款预警">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="alertForm.customer_id" :clearable="false" /></el-form-item>
          <el-form-item label="逾期天"><el-input-number v-model="alertForm.overdue_days" :min="0" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="alertForm.amount" :min="0" /></el-form-item>
          <el-form-item label="到期日"><el-date-picker v-model="alertForm.due_date" type="date" value-format="YYYY-MM-DD" style="width:150px" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createAlert({ ...alertForm }), '预警已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="alertCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="customer_name" label="客户" min-width="120" /><el-table-column prop="overdue_days" label="逾期天" width="90" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='open'" link type="primary" @click="run(() => financeApi.handleAlert(Number(row.id), { remark: '已跟进' }), '已处理')">处理</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='open'" link type="primary" @click="run(() => financeApi.handleAlert(Number(row.id), { remark: '已跟进' }), '已处理')">处理</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'reconciles'">
      <el-card class="mb" header="出纳对账（账面余额自动带出）">
        <el-form inline size="small">
          <el-form-item label="资金账户">
            <el-select v-model="reconcileForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="`${f.name}(${f.balance})`" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="实盘余额"><el-input-number v-model="reconcileForm.actual_balance" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createReconcile({ ...reconcileForm }), '对账单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="reconcileCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="account_name" label="账户" min-width="120" />
          <el-table-column label="账面" width="110"><template #default="{ row }">{{ money(row.book_balance) }}</template></el-table-column>
          <el-table-column label="实盘" width="110"><template #default="{ row }">{{ money(row.actual_balance) }}</template></el-table-column>
          <el-table-column label="差额" width="110"><template #default="{ row }">{{ money(row.diff) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="confirmReconcileRow(row)">确认</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="confirmReconcileRow(row)">确认</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'prepays'">
      <el-card class="mb" header="预收/预付">
        <el-form inline size="small">
          <el-form-item label="类型">
            <el-select v-model="prepayForm.party_type" style="width:110px">
              <el-option label="客户" value="customer" />
              <el-option label="供应商" value="supplier" />
              <el-option label="农户" value="farmer" />
            </el-select>
          </el-form-item>
          <el-form-item label="对方">
            <CustomerSelect v-if="prepayForm.party_type === 'customer'" v-model="prepayForm.party_id" :clearable="false" />
            <SupplierSelect v-else-if="prepayForm.party_type === 'supplier'" v-model="prepayForm.party_id" :clearable="false" />
            <el-select v-else v-model="prepayForm.party_id" style="width:160px" filterable>
              <el-option v-for="f in farmers" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="方向">
            <el-select v-model="prepayForm.direction" style="width:90px"><el-option label="预收" value="in" /><el-option label="预付" value="out" /></el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="prepayForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="prepayForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createPrepay({ ...prepayForm }), '已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="prepayCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column label="类型" width="90"><template #default="{ row }">{{ finPartyLabel(row.party_type) }}</template></el-table-column>
          <el-table-column prop="party_name" label="对方" min-width="120" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="余额" width="110"><template #default="{ row }">{{ money(row.balance) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'cost-accountings'">
      <el-card class="mb" header="成本核算">
        <el-form inline size="small">
          <el-form-item label="期间"><el-date-picker v-model="costForm.period" type="month" value-format="YYYY-MM" style="width:140px" /></el-form-item>
          <el-form-item label="产品"><ProductSelect v-model="costForm.product_id" :clearable="false" /></el-form-item>
          <el-form-item label="物料成本"><el-input-number v-model="costForm.material_cost" :min="0" /></el-form-item>
          <el-form-item label="人工"><el-input-number v-model="costForm.labor_cost" :min="0" /></el-form-item>
          <el-form-item label="制造费用"><el-input-number v-model="costForm.overhead" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createCostAccounting({ ...costForm }), '成本单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="costCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="product_name" label="产品" min-width="120" />
          <el-table-column label="物料" width="100"><template #default="{ row }">{{ money(row.material_cost) }}</template></el-table-column>
          <el-table-column label="人工" width="100"><template #default="{ row }">{{ money(row.labor_cost) }}</template></el-table-column>
          <el-table-column label="制造" width="100"><template #default="{ row }">{{ money(row.overhead) }}</template></el-table-column>
          <el-table-column label="合计" width="110"><template #default="{ row }">{{ money(row.total_cost) }}</template></el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.calcCost(Number(row.id)), '已核算')">核算</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.calcCost(Number(row.id)), '已核算')">核算</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'contract-profits'">
      <div class="mb"><el-button type="primary" size="small" @click="run(() => financeApi.recalcContractProfit({}), '利润已重算')">重算合同利润</el-button></div>
      <TableOrCards :data="filteredList" :loading="loading" :columns="contractProfitCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="contract_no" label="合同" min-width="140" />
          <el-table-column prop="customer_name" label="客户" min-width="120" />
          <el-table-column label="收入" width="120"><template #default="{ row }">{{ money(row.revenue) }}</template></el-table-column>
          <el-table-column v-if="showCost" label="成本" width="120"><template #default="{ row }">{{ money(row.cost) }}</template></el-table-column>
          <el-table-column v-if="showProfit" label="利润" width="120"><template #default="{ row }">{{ money(row.profit) }}</template></el-table-column>
          <el-table-column prop="period" label="期间" width="100" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'return-finances'">
      <el-card class="mb" header="销售退货退单（财务）">
        <el-form inline size="small">
          <el-form-item label="订单"><SalesOrderSelect v-model="returnForm.order_id" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="returnForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="returnForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="'rf'+f.id" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createReturnFinance({ ...returnForm }), '退单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="returnCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="order_no" label="订单" min-width="140" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column prop="account_name" label="账户" width="120" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmReturnFinance(Number(row.id)), '已确认')">确认</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmReturnFinance(Number(row.id)), '已确认')">确认</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'arap'">
      <el-card class="mb" header="往来调整">
        <el-form inline size="small">
          <el-form-item label="类型">
            <el-select v-model="arapForm.party_type" style="width:110px"><el-option label="客户" value="customer" /><el-option label="供应商" value="supplier" /></el-select>
          </el-form-item>
          <el-form-item label="对方">
            <CustomerSelect v-if="arapForm.party_type === 'customer'" v-model="arapForm.party_id" :clearable="false" />
            <SupplierSelect v-else v-model="arapForm.party_id" :clearable="false" />
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="arapForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="方向">
            <el-select v-model="arapForm.direction" style="width:110px"><el-option label="调增" value="increase" /><el-option label="调减" value="decrease" /></el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createArap({ ...arapForm }), '调整单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="arapCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column label="类型" width="90"><template #default="{ row }">{{ finPartyLabel(row.party_type) }}</template></el-table-column>
          <el-table-column prop="party_name" label="对方" min-width="120" />
          <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="方向" width="90"><template #default="{ row }">{{ finDirLabel(row.direction) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.postArap(Number(row.id)), '已过账')">过账</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.postArap(Number(row.id)), '已过账')">过账</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'approvals'">
      <TableOrCards :data="filteredList" :loading="loading" :columns="approvalCols">
        <el-table :data="filteredList" size="small">
          <el-table-column label="来源" width="90">
            <template #default="{ row }">{{ row.source === 'approval_item' ? '审批单' : '凭证' }}</template>
          </el-table-column>
          <el-table-column prop="biz_type" label="类型" width="100" /><el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="title" label="摘要" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft' || row.status==='pending' || row.status==='submitted'" link type="success" @click="run(() => financeApi.approveFinance(Number(row.id), { source: String(row.source || 'voucher') }), '已批准')">批准</el-button>
              <el-button v-if="row.status==='draft' || row.status==='pending' || row.status==='submitted'" link type="danger" @click="run(() => financeApi.rejectFinance(Number(row.id), { source: String(row.source || 'voucher') }), '已驳回')">驳回</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft' || row.status==='pending' || row.status==='submitted'" link type="success" @click="run(() => financeApi.approveFinance(Number(row.id), { source: String(row.source || 'voucher') }), '已批准')">批准</el-button>
          <el-button v-if="row.status==='draft' || row.status==='pending' || row.status==='submitted'" link type="danger" @click="run(() => financeApi.rejectFinance(Number(row.id), { source: String(row.source || 'voucher') }), '已驳回')">驳回</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'funds'">
      <el-radio-group v-model="fundTab" size="small" class="mb">
        <el-radio-button value="accounts">资金账户</el-radio-button>
        <el-radio-button value="transfers">资金调拨</el-radio-button>
      </el-radio-group>
      <el-card v-if="fundTab==='accounts'" class="mb" header="新建资金账户">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="fundAccForm.code" style="width:120px" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="fundAccForm.name" style="width:140px" /></el-form-item>
          <el-form-item label="币种"><EnumSelect v-model="fundAccForm.currency" :options="CURRENCY_OPTIONS" :clearable="false" style="width:140px" /></el-form-item>
          <el-form-item label="期初"><el-input-number v-model="fundAccForm.balance" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createFundAccount({ ...fundAccForm }), '账户已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-card v-else class="mb" header="资金调拨（过账双边余额）">
        <el-form inline size="small">
          <el-form-item label="转出">
            <el-select v-model="fundTfForm.from_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="'f'+f.id" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="转入">
            <el-select v-model="fundTfForm.to_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="'t'+f.id" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="fundTfForm.amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createFundTransfer({ ...fundTfForm }), '调拨单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="fundCols">
        <el-table :data="filteredList" size="small">
          <template v-if="fundTab==='accounts'">
            <el-table-column prop="code" label="编码" width="120" /><el-table-column prop="name" label="名称" />
            <el-table-column prop="currency" label="币种" width="80" />
            <el-table-column label="余额" width="120"><template #default="{ row }">{{ money(row.balance) }}</template></el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
            </el-table-column>
          </template>
          <template v-else>
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="from_account_name" label="转出" min-width="120" /><el-table-column prop="to_account_name" label="转入" min-width="120" />
            <el-table-column label="金额" width="110"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.postFundTransfer(Number(row.id)), '已过账')">过账</el-button>
              </template>
            </el-table-column>
          </template>
        </el-table>
        <template v-if="fundTab==='transfers'" #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.postFundTransfer(Number(row.id)), '已过账')">过账</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'statements'">
      <div class="mb">
        <el-button type="primary" size="small" @click="generateStatements">生成三大表</el-button>
      </div>
      <div v-if="statementPreview" class="stmt-preview">
        <div class="stmt-group">
          <h4>利润表</h4>
          <div class="stmt-row"><span>期间</span><b>{{ statementPreview.period }}</b></div>
          <div class="stmt-row"><span>收入</span><b>{{ money((statementPreview.profit_loss as Row)?.income) }}</b></div>
          <div class="stmt-row"><span>支出</span><b>{{ money((statementPreview.profit_loss as Row)?.expense) }}</b></div>
          <div class="stmt-row"><span>利润</span><b>{{ money((statementPreview.profit_loss as Row)?.profit) }}</b></div>
        </div>
        <div class="stmt-group">
          <h4>现金流量</h4>
          <div class="stmt-row"><span>流入</span><b>{{ money((statementPreview.cash_flow as Row)?.['in']) }}</b></div>
          <div class="stmt-row"><span>流出</span><b>{{ money((statementPreview.cash_flow as Row)?.['out']) }}</b></div>
          <div class="stmt-row"><span>净额</span><b>{{ money((statementPreview.cash_flow as Row)?.net) }}</b></div>
          <div class="stmt-row"><span>资金余额</span><b>{{ money((statementPreview.cash_flow as Row)?.fund_balance) }}</b></div>
        </div>
        <div class="stmt-group">
          <h4>资产负债表</h4>
          <div class="stmt-row"><span>资产</span><b>{{ money((statementPreview.balance_sheet as Row)?.assets) }}</b></div>
          <div class="stmt-row"><span>负债</span><b>{{ money((statementPreview.balance_sheet as Row)?.liabilities) }}</b></div>
          <div class="stmt-row"><span>权益</span><b>{{ money((statementPreview.balance_sheet as Row)?.equity) }}</b></div>
          <div class="stmt-row"><span>现金</span><b>{{ money((statementPreview.balance_sheet as Row)?.cash) }}</b></div>
        </div>
      </div>
      <TableOrCards :data="filteredList" :loading="loading" :columns="statementCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="code" label="报表" width="120" /><el-table-column prop="period" label="期间" width="110" />
          <el-table-column prop="title" label="标题" /><el-table-column prop="generated_at" label="生成时间" width="170" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'cost-traces'">
      <TableOrCards :data="filteredList" :loading="loading" :columns="costTraceCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="成本单" width="150" /><el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="source_type" label="来源类型" width="120" /><el-table-column prop="source_id" label="来源ID" width="90" />
          <el-table-column label="金额" width="120"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'month-closes'">
      <el-card class="mb" header="月度结转">
        <el-form inline size="small">
          <el-form-item label="年"><el-input-number v-model="monthForm.year" :min="2020" /></el-form-item>
          <el-form-item label="月"><el-input-number v-model="monthForm.month" :min="1" :max="12" /></el-form-item>
          <el-button type="primary" @click="closeMonthConfirm">结转</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="monthCloseCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="year" label="年" width="90" /><el-table-column prop="month" label="月" width="80" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="closed_at" label="结转时间" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='closed'" link type="warning" @click="reopenMonthConfirm(row)">反结转</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='closed'" link type="warning" @click="reopenMonthConfirm(row)">反结转</el-button>
        </template>
      </TableOrCards>
    </template>
  </div>
</template>

<style scoped>
.page {
  background: #fff;
  padding: 12px 16px 8px;
  border-radius: 8px;
  border: 1px solid #d5dde3;
  height: calc(100vh - 120px);
  min-height: 360px;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; flex-shrink: 0; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; }
.desc { color: #5c6b75; font-size: 13px; margin: 0; max-width: 720px; line-height: 1.5; }
.head-actions { display: flex; gap: 8px; flex-shrink: 0; }
.stats { display: grid; grid-template-columns: repeat(4, minmax(96px, 1fr)); gap: 10px; margin: 12px 0; flex-shrink: 0; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat.ok { background: #eef6f1; border-color: #d5eade; }
.stat.warn { background: #fff7f0; border-color: #f0e0d0; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 18px; font-weight: 600; font-variant-numeric: tabular-nums; }
.toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 10px; flex-shrink: 0; }
.mb { margin-bottom: 12px; flex-shrink: 0; }
.stmt-preview {
  display: grid; grid-template-columns: repeat(3, minmax(180px, 1fr)); gap: 12px;
  margin-bottom: 12px;
}
.stmt-group { background: #f6f8fa; padding: 12px; border-radius: 8px; display: grid; gap: 8px; }
.stmt-group h4 { margin: 0 0 4px; font-size: 13px; color: #1f2a33; }
.stmt-row { display: flex; justify-content: space-between; gap: 8px; font-size: 12px; color: #6b7a85; }
.stmt-row b { font-size: 14px; color: #1f2a33; font-variant-numeric: tabular-nums; }
.page :deep(.table-or-cards) { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.page :deep(.el-table__body-wrapper) { max-height: calc(100vh - 380px); overflow-y: auto; }
@media (max-width: 768px) {
  .page { height: auto; }
  .stats { grid-template-columns: repeat(2, 1fr); }
  .stmt-preview { grid-template-columns: 1fr; }
}
</style>
