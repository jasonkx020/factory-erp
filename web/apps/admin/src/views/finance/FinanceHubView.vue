<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { financeApi, purchaseApi } from '@erp/shared'
import { ProductSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import {
  FIN_HINT,
  finDirLabel,
  finErrMsg,
  finStatusLabel,
  finStatusType,
  money,
} from './financeLabels'

type Row = Record<string, unknown>

const SECTIONS = [
  { key: 'payables', title: '农户应付', hintKey: 'payables' },
  { key: 'funds', title: '资金管理', hintKey: 'funds' },
  { key: 'ledger', title: '交易流水账', hintKey: 'ledger' },
  { key: 'cost-accountings', title: '成本核算', hintKey: 'cost-accountings' },
  { key: 'cost-traces', title: '成本明细溯源表', hintKey: 'cost-traces' },
] as const

type SectionKey = (typeof SECTIONS)[number]['key']

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
const costTraceCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '成本单', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'source_label', label: '来源' },
  { prop: 'source_id', label: '来源ID' },
  moneyCol('amount', '金额'),
]
const fundCols: MobileCardColumn[] = [
  { prop: 'name', label: '账户', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'currency', label: '币种' },
  moneyCol('balance', '余额'),
  statusCol(),
]
const ledgerCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'biz_date', label: '日期' },
  { prop: 'account_name', label: '账户' },
  { prop: 'direction', label: '方向', format: (v) => finDirLabel(v) },
  moneyCol('amount', '金额'),
  { prop: 'counterparty', label: '对方' },
]
const payableCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '结算单', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'biz_date', label: '日期' },
  moneyCol('amount', '应付'),
  statusCol(),
]

const REPORT_LINKS = [
  { title: '日经营快照', path: '/report/hub/daily', desc: '入场、产出、计件、农户应付与库存' },
  { title: '计件日结汇总', path: '/report/hub/piecework-daily', desc: '与现场计件核对' },
  { title: '农户结算对账', path: '/report/hub/farmer-settlement-summary', desc: '原料款已付/待付' },
  { title: '薪酬核算对账', path: '/report/hub/payroll-reconcile', desc: '月工资 vs 计件差异' },
  { title: '成本期间汇总', path: '/report/hub/cost-period-summary', desc: '按期间汇总成本单' },
]

const route = useRoute()
const router = useRouter()

const active = computed<SectionKey>(() => {
  const s = String(route.params.section || 'payables')
  return (SECTIONS.find((x) => x.key === s)?.key || 'cost-accountings') as SectionKey
})
const title = computed(() => SECTIONS.find((x) => x.key === active.value)?.title || '财务管理')
const hint = computed(() => FIN_HINT[SECTIONS.find((x) => x.key === active.value)?.hintKey || ''] || '')
const loading = ref(false)
const list = ref<Row[]>([])
const keyword = ref('')
const farmerPending = ref(0)
const farmerPendingAmt = ref(0)
const fundBalanceTotal = ref(0)
const preview = ref<Row | null>(null)
const previewLoading = ref(false)

const costForm = reactive({
  period: new Date().toISOString().slice(0, 7),
  product_id: 1,
  material_cost: 0,
  labor_cost: 0,
  overhead: 0,
})

const fundForm = reactive({ code: '', name: '', currency: 'CNY', balance: 0 })
const transferForm = reactive({ from_account_id: 0, to_account_id: 0, amount: 0, remark: '' })
const ledgerForm = reactive({
  account_id: 0,
  direction: 'out',
  amount: 0,
  counterparty: '',
  remark: '',
  biz_date: new Date().toISOString().slice(0, 10),
  category: '',
})
const payForm = reactive({ transfer_no: '', pay_evidence_url: '', fund_account_id: 0 })
const payRow = ref<Row | null>(null)
const payVisible = ref(false)
const fundAccounts = ref<Row[]>([])
const transferList = ref<Row[]>([])

const filteredList = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list.value
  return list.value.filter((row) =>
    Object.values(row).some((v) => String(v ?? '').toLowerCase().includes(kw)),
  )
})

const stats = computed(() => {
  const rows = filteredList.value
  const sum = (k: string) => rows.reduce((s, r) => s + (Number(r[k]) || 0), 0)
  switch (active.value) {
    case 'cost-accountings':
      return [
        { label: '成本单', value: String(rows.length), tone: '' },
        { label: '物料', value: money(sum('material_cost')), tone: '' },
        { label: '人工', value: money(sum('labor_cost')), tone: '' },
        { label: '总成本', value: money(sum('total_cost')), tone: 'ok' },
      ]
    case 'cost-traces':
      return [
        { label: '溯源行', value: String(rows.length), tone: '' },
        { label: '金额合计', value: money(sum('amount')), tone: 'ok' },
      ]
    case 'funds':
      return [
        { label: '账户数', value: String(rows.length), tone: '' },
        { label: '余额合计', value: money(sum('balance')), tone: 'ok' },
        { label: '调拨单', value: String(transferList.value.length), tone: '' },
      ]
    case 'ledger': {
      const inn = rows.filter((r) => String(r.direction) === 'in').reduce((s, r) => s + (Number(r.amount) || 0), 0)
      const out = rows.filter((r) => String(r.direction) === 'out').reduce((s, r) => s + (Number(r.amount) || 0), 0)
      return [
        { label: '流水笔数', value: String(rows.length), tone: '' },
        { label: '收入', value: money(inn), tone: 'ok' },
        { label: '支出', value: money(out), tone: 'warn' },
        { label: '账户余额', value: money(fundBalanceTotal.value), tone: '' },
      ]
    }
    case 'payables':
    default:
      return [
        { label: '待付笔数', value: String(farmerPending.value), tone: farmerPending.value ? 'warn' : 'ok' },
        { label: '待付金额', value: money(farmerPendingAmt.value), tone: farmerPending.value ? 'warn' : '' },
        { label: '本页合计', value: money(sum('amount')), tone: '' },
        { label: '可用资金', value: money(fundBalanceTotal.value), tone: 'ok' },
      ]
  }
})

function goSection(key: string) {
  router.push(`/finance/hub/${key}`)
}

function goReport(path: string) {
  router.push(path)
}

function goTraceSource(row: Row) {
  const t = String(row.source_type || '')
  if (t === 'farmer_settlement') router.push('/finance/hub/payables')
  else if (t === 'piecework_day') router.push('/report/hub/piecework-daily')
  else if (t === 'requisition') router.push('/production/hub/process-wip')
  else router.push('/report/hub/cost-period-summary')
}

function isPayablePending(row: Row) {
  const st = String(row.status || '')
  return st !== 'settle_paid' && st !== 'paid' && st !== 'void'
}

async function loadOverview() {
  try {
    const [fs, fa] = await Promise.all([purchaseApi.farmerSettlements(), financeApi.fundAccounts()])
    const settles = ((fs.data as { list?: Row[] })?.list) || []
    const pending = settles.filter(isPayablePending)
    farmerPending.value = pending.length
    farmerPendingAmt.value = pending.reduce((s, r) => s + (Number(r.amount) || 0), 0)
    const funds = ((fa.data as { list?: Row[] })?.list) || []
    fundAccounts.value = funds
    fundBalanceTotal.value = funds.reduce((s, r) => s + (Number(r.balance) || 0), 0)
    if (!payForm.fund_account_id && funds[0]) payForm.fund_account_id = Number(funds[0].id)
    if (!ledgerForm.account_id && funds[0]) ledgerForm.account_id = Number(funds[0].id)
    if (!transferForm.from_account_id && funds[0]) transferForm.from_account_id = Number(funds[0].id)
    if (!transferForm.to_account_id && funds[1]) transferForm.to_account_id = Number(funds[1].id)
  } catch {
    farmerPending.value = 0
    farmerPendingAmt.value = 0
  }
}

async function refresh() {
  loading.value = true
  try {
    if (active.value === 'cost-traces') {
      const res = await financeApi.costTraces()
      if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'cost-accountings') {
      const res = await financeApi.costAccountings()
      if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'funds') {
      const [fa, ft] = await Promise.all([financeApi.fundAccounts(), financeApi.fundTransfers()])
      if (fa.code !== 1) return ElMessage.error(finErrMsg(fa.msg))
      list.value = ((fa.data as { list?: Row[] })?.list) || []
      transferList.value = ((ft.data as { list?: Row[] })?.list) || []
      fundAccounts.value = list.value
      fundBalanceTotal.value = list.value.reduce((s, r) => s + (Number(r.balance) || 0), 0)
    } else if (active.value === 'ledger') {
      const res = await financeApi.ledger()
      if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else {
      const res = await purchaseApi.farmerSettlements()
      if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
      const all = ((res.data as { list?: Row[] })?.list) || []
      list.value = all.filter(isPayablePending)
      farmerPending.value = list.value.length
      farmerPendingAmt.value = list.value.reduce((s, r) => s + (Number(r.amount) || 0), 0)
    }
  } finally {
    loading.value = false
  }
}

async function run(fn: () => Promise<{ code: number; msg: string }>, ok = '成功') {
  const res = await fn()
  if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
  ElMessage.success(ok)
  await loadOverview()
  await refresh()
}

async function fillFromPeriod() {
  previewLoading.value = true
  try {
    const res = await financeApi.previewCostPeriod({
      period: costForm.period,
      product_id: costForm.product_id,
    })
    if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
    const d = (res.data || {}) as Row
    preview.value = d
    costForm.material_cost = Number(d.material_cost) || 0
    costForm.labor_cost = Number(d.labor_cost) || 0
    ElMessage.success(
      `已汇入：农户已付 ${money(d.farmer_paid)}（${d.farmer_paid_count}笔），计件 ${money(d.piecework_amount)}`,
    )
  } finally {
    previewLoading.value = false
  }
}

async function createCost() {
  await run(
    () =>
      financeApi.createCostAccounting({
        ...costForm,
        auto_fill: costForm.material_cost <= 0 && costForm.labor_cost <= 0,
      }),
    '成本单已建',
  )
}

function openPay(row: Row) {
  payRow.value = row
  payForm.transfer_no = ''
  payForm.pay_evidence_url = ''
  payForm.fund_account_id = fundAccounts.value[0] ? Number(fundAccounts.value[0].id) : 0
  payVisible.value = true
}

async function confirmPay() {
  if (!payRow.value) return
  if (!payForm.fund_account_id) return ElMessage.warning('请选择资金账户')
  if (!payForm.transfer_no || !payForm.pay_evidence_url) return ElMessage.warning('转账单号与回单必填')
  await run(() => purchaseApi.payFarmerSettlement(Number(payRow.value!.id), { ...payForm }), '已支付关单')
  payVisible.value = false
}

onMounted(async () => {
  await loadOverview()
  await refresh()
})
watch(active, refresh)
</script>

<template>
  <div class="page factory-floor" v-loading="loading">
    <header class="page-head">
      <div>
        <div class="eyebrow">木薯加工 · 结算财务</div>
        <h2 class="title">{{ title }}</h2>
        <p class="desc">{{ hint }}</p>
      </div>
      <div class="head-actions">
        <el-button type="warning" plain @click="goSection('payables')">
          待付 {{ farmerPending }} · {{ money(farmerPendingAmt) }}
        </el-button>
        <el-button @click="refresh">刷新</el-button>
      </div>
    </header>

    <nav class="section-tabs">
      <button
        v-for="s in SECTIONS"
        :key="s.key"
        type="button"
        class="tab"
        :class="{ active: active === s.key }"
        @click="goSection(s.key)"
      >
        {{ s.title }}
        <span v-if="s.key === 'payables' && farmerPending" class="badge">{{ farmerPending }}</span>
      </button>
    </nav>

    <div class="stats">
      <div v-for="s in stats" :key="s.label" class="factory-kpi" :class="s.tone">
        <div class="kpi-label">{{ s.label }}</div>
        <div class="kpi-value">{{ s.value }}</div>
      </div>
    </div>

    <div class="factory-panel links-panel mb">
      <div class="panel-title">经营对账</div>
      <div class="report-links">
        <button v-for="link in REPORT_LINKS" :key="link.path" type="button" class="report-link" @click="goReport(link.path)">
          <div class="link-title">{{ link.title }}</div>
          <div class="link-desc">{{ link.desc }}</div>
        </button>
      </div>
    </div>

    <div class="toolbar">
      <el-input v-model="keyword" clearable placeholder="筛选单号 / 名称 / 期间" style="width:240px" />
    </div>

    <!-- 农户应付 -->
    <template v-if="active === 'payables'">
      <TableOrCards :data="filteredList" :loading="loading" :columns="payableCols">
        <el-table :data="filteredList" size="small" class="fin-table">
          <el-table-column prop="doc_no" label="结算单" width="150" />
          <el-table-column prop="farmer_name" label="农户" min-width="120" />
          <el-table-column prop="biz_date" label="日期" width="110" />
          <el-table-column label="净重" width="90">
            <template #default="{ row }">{{ row.net_weight ?? '-' }}</template>
          </el-table-column>
          <el-table-column label="应付" width="120">
            <template #default="{ row }">{{ money(row.amount) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="110">
            <template #default="{ row }">
              <el-button link type="primary" @click="openPay(row)">支付关单</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link type="primary" @click="openPay(row)">支付关单</el-button>
        </template>
      </TableOrCards>
      <p class="foot-hint">支付将扣减资金账户余额并写入交易流水；完整列表也可在采购「农户结算」处理。</p>
    </template>

    <!-- 资金账户 -->
    <template v-else-if="active === 'funds'">
      <div class="factory-panel mb">
        <div class="panel-title">开立账户</div>
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="fundForm.code" placeholder="如 BANK2" style="width:120px" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="fundForm.name" style="width:140px" /></el-form-item>
          <el-form-item label="期初">
            <el-input-number v-model="fundForm.balance" :min="0" :precision="2" />
          </el-form-item>
          <el-button
            type="primary"
            @click="run(() => financeApi.createFundAccount({ ...fundForm }), '账户已建')"
          >新建</el-button>
        </el-form>
      </div>
      <TableOrCards :data="filteredList" :loading="loading" :columns="fundCols">
        <el-table :data="filteredList" size="small" class="fin-table">
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="currency" label="币种" width="80" />
          <el-table-column label="余额" width="130">
            <template #default="{ row }"><span class="mono">{{ money(row.balance) }}</span></template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>

      <div class="factory-panel mb mt">
        <div class="panel-title">内部调拨</div>
        <el-form inline size="small">
          <el-form-item label="转出">
            <el-select v-model="transferForm.from_account_id" style="width:160px">
              <el-option
                v-for="f in fundAccounts"
                :key="String(f.id)"
                :label="String(f.name)"
                :value="Number(f.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="转入">
            <el-select v-model="transferForm.to_account_id" style="width:160px">
              <el-option
                v-for="f in fundAccounts"
                :key="String(f.id)"
                :label="String(f.name)"
                :value="Number(f.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="金额">
            <el-input-number v-model="transferForm.amount" :min="0" :precision="2" />
          </el-form-item>
          <el-form-item label="备注"><el-input v-model="transferForm.remark" style="width:140px" /></el-form-item>
          <el-button
            type="primary"
            @click="run(() => financeApi.createFundTransfer({ ...transferForm }), '调拨单已建')"
          >建调拨</el-button>
        </el-form>
        <el-table :data="transferList" size="small" class="fin-table mt-sm">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="from_account_name" label="转出" min-width="100" />
          <el-table-column prop="to_account_name" label="转入" min-width="100" />
          <el-table-column label="金额" width="110">
            <template #default="{ row }">{{ money(row.amount) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">{{ finStatusLabel(row.status) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button
                v-if="row.status === 'draft'"
                link
                type="primary"
                @click="run(() => financeApi.postFundTransfer(Number(row.id)), '已过账')"
              >过账</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </template>

    <!-- 流水 -->
    <template v-else-if="active === 'ledger'">
      <div class="factory-panel mb">
        <div class="panel-title">登记收支</div>
        <el-form inline size="small">
          <el-form-item label="账户">
            <el-select v-model="ledgerForm.account_id" style="width:160px">
              <el-option
                v-for="f in fundAccounts"
                :key="String(f.id)"
                :label="`${f.name}（${money(f.balance)}）`"
                :value="Number(f.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="方向">
            <el-select v-model="ledgerForm.direction" style="width:100px">
              <el-option label="收入" value="in" />
              <el-option label="支出" value="out" />
            </el-select>
          </el-form-item>
          <el-form-item label="金额">
            <el-input-number v-model="ledgerForm.amount" :min="0" :precision="2" />
          </el-form-item>
          <el-form-item label="日期">
            <el-date-picker v-model="ledgerForm.biz_date" type="date" value-format="YYYY-MM-DD" style="width:140px" />
          </el-form-item>
          <el-form-item label="对方"><el-input v-model="ledgerForm.counterparty" style="width:120px" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="ledgerForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createLedger({ ...ledgerForm }), '已入账')">登记</el-button>
        </el-form>
      </div>
      <TableOrCards :data="filteredList" :loading="loading" :columns="ledgerCols">
        <el-table :data="filteredList" size="small" class="fin-table">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="biz_date" label="日期" width="110" />
          <el-table-column prop="account_name" label="账户" width="120" />
          <el-table-column label="方向" width="80">
            <template #default="{ row }">
              <span :class="row.direction === 'in' ? 'dir-in' : 'dir-out'">{{ finDirLabel(row.direction) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="120">
            <template #default="{ row }"><span class="mono">{{ money(row.amount) }}</span></template>
          </el-table-column>
          <el-table-column prop="counterparty" label="对方" min-width="120" />
          <el-table-column prop="source_doc_type" label="来源" width="130" />
          <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        </el-table>
      </TableOrCards>
    </template>

    <!-- 成本核算 -->
    <template v-else-if="active === 'cost-accountings'">
      <div class="factory-panel mb">
        <div class="panel-title">新建成本核算单</div>
        <el-form inline size="small">
          <el-form-item label="期间">
            <el-date-picker v-model="costForm.period" type="month" value-format="YYYY-MM" style="width:140px" />
          </el-form-item>
          <el-form-item label="产品"><ProductSelect v-model="costForm.product_id" :clearable="false" /></el-form-item>
          <el-form-item label="物料"><el-input-number v-model="costForm.material_cost" :min="0" :precision="2" /></el-form-item>
          <el-form-item label="人工"><el-input-number v-model="costForm.labor_cost" :min="0" :precision="2" /></el-form-item>
          <el-form-item label="制造"><el-input-number v-model="costForm.overhead" :min="0" :precision="2" /></el-form-item>
          <el-button :loading="previewLoading" @click="fillFromPeriod">一键汇入期间</el-button>
          <el-button type="primary" @click="createCost">新建</el-button>
        </el-form>
        <div v-if="preview" class="preview-strip">
          <span>农户已付 {{ money(preview.farmer_paid) }}（{{ preview.farmer_paid_count }}）</span>
          <span>待付 {{ money(preview.farmer_pending) }}（{{ preview.farmer_pending_count }}）</span>
          <span>计件 {{ money(preview.piecework_amount) }}（{{ preview.piecework_count }}）</span>
          <span v-if="Number(preview.requisition_cost)">领料 {{ money(preview.requisition_cost) }}</span>
        </div>
      </div>
      <TableOrCards :data="filteredList" :loading="loading" :columns="costCols">
        <el-table :data="filteredList" size="small" class="fin-table">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="product_name" label="产品" min-width="120" />
          <el-table-column label="物料" width="100"><template #default="{ row }">{{ money(row.material_cost) }}</template></el-table-column>
          <el-table-column label="人工" width="100"><template #default="{ row }">{{ money(row.labor_cost) }}</template></el-table-column>
          <el-table-column label="制造" width="100"><template #default="{ row }">{{ money(row.overhead) }}</template></el-table-column>
          <el-table-column label="合计" width="110"><template #default="{ row }"><span class="mono">{{ money(row.total_cost) }}</span></template></el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button
                v-if="row.status === 'draft' || row.status === 'calculated'"
                link
                type="primary"
                @click="run(() => financeApi.calcCost(Number(row.id), { force_refresh: true }), '已核算')"
              >{{ row.status === 'draft' ? '核算' : '重算' }}</el-button>
              <el-button link @click="goSection('cost-traces')">溯源</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button
            v-if="row.status === 'draft' || row.status === 'calculated'"
            link
            type="primary"
            @click="run(() => financeApi.calcCost(Number(row.id), { force_refresh: true }), '已核算')"
          >核算</el-button>
        </template>
      </TableOrCards>
    </template>

    <!-- 溯源 -->
    <template v-else>
      <TableOrCards :data="filteredList" :loading="loading" :columns="costTraceCols">
        <el-table :data="filteredList" size="small" class="fin-table">
          <el-table-column prop="doc_no" label="成本单" width="150" />
          <el-table-column prop="period" label="期间" width="100" />
          <el-table-column label="来源" width="120">
            <template #default="{ row }">{{ row.source_label || row.source_type }}</template>
          </el-table-column>
          <el-table-column prop="source_id" label="来源ID" width="90" />
          <el-table-column label="金额" width="120"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="跳转" width="100">
            <template #default="{ row }">
              <el-button link type="primary" @click="goTraceSource(row)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link type="primary" @click="goTraceSource(row)">查看来源</el-button>
        </template>
      </TableOrCards>
    </template>

    <el-dialog v-model="payVisible" title="农户货款支付关单" width="480px" destroy-on-close>
      <el-form label-width="100px" size="small">
        <el-form-item label="结算单">{{ payRow?.doc_no }} · {{ money(payRow?.amount) }}</el-form-item>
        <el-form-item label="资金账户" required>
          <el-select v-model="payForm.fund_account_id" style="width:100%">
            <el-option
              v-for="f in fundAccounts"
              :key="String(f.id)"
              :label="`${f.name}（余额 ${money(f.balance)}）`"
              :value="Number(f.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="转账单号" required><el-input v-model="payForm.transfer_no" /></el-form-item>
        <el-form-item label="回单/截图URL" required><el-input v-model="payForm.pay_evidence_url" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="payVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmPay">确认支付</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page {
  padding: 12px 16px 16px;
  border-radius: 8px;
  border: 1px solid var(--border, #cfdcd4);
  min-height: 420px;
}
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}
.eyebrow {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent-soil, #a67c3d);
  margin-bottom: 2px;
}
.title {
  margin: 0 0 4px;
  font-size: 18px;
  font-weight: 650;
  color: var(--text, #1a2e24);
}
.desc {
  color: var(--muted, #5c6b75);
  font-size: 13px;
  margin: 0;
  max-width: 720px;
  line-height: 1.5;
}
.head-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.section-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
  margin-bottom: 12px;
  border: 1px solid var(--border, #cfdcd4);
  border-radius: 8px;
  overflow: hidden;
  background: var(--panel, #fff);
}
.section-tabs .tab {
  position: relative;
  border: 0;
  background: transparent;
  padding: 9px 14px;
  font-size: 13px;
  color: var(--muted, #5c6b75);
  cursor: pointer;
  border-right: 1px solid var(--border, #cfdcd4);
}
.section-tabs .tab:last-child {
  border-right: 0;
}
.section-tabs .tab.active {
  background: var(--accent-soft, #e8f5ee);
  color: var(--accent, #1f7a4d);
  font-weight: 600;
}
.section-tabs .badge {
  margin-left: 6px;
  background: var(--factory-warn, #c47a12);
  color: #fff;
  font-size: 11px;
  padding: 0 6px;
  border-radius: 8px;
}
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}
.kpi-label {
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--muted, #6b7a85);
  text-transform: uppercase;
}
.kpi-value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  font-family: var(--factory-mono, ui-monospace, monospace);
  color: var(--text, #1a2e24);
}
.factory-panel {
  background: var(--panel, #fff);
  border: 1px solid var(--border, #cfdcd4);
  border-radius: 8px;
  padding: 12px 14px;
}
.panel-title {
  font-size: 13px;
  font-weight: 650;
  margin-bottom: 10px;
  color: var(--accent, #1f7a4d);
}
.links-panel .panel-title {
  margin-bottom: 8px;
}
.report-links {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
}
.report-link {
  text-align: left;
  padding: 8px 10px;
  border: 1px solid var(--border, #e0e8e3);
  border-radius: 6px;
  cursor: pointer;
  background: var(--factory-panel, #f4f7f5);
  color: inherit;
}
.report-link:hover {
  border-color: var(--accent, #1f7a4d);
  background: var(--accent-soft, #e8f5ee);
}
.link-title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 2px;
}
.link-desc {
  font-size: 11px;
  color: var(--muted, #6b7a85);
  line-height: 1.35;
}
.toolbar {
  margin-bottom: 10px;
}
.mb {
  margin-bottom: 12px;
}
.mt {
  margin-top: 14px;
}
.mt-sm {
  margin-top: 8px;
}
.mono {
  font-variant-numeric: tabular-nums;
  font-family: var(--factory-mono, ui-monospace, monospace);
}
.dir-in {
  color: var(--ok, #1f7a3f);
  font-weight: 600;
}
.dir-out {
  color: var(--factory-warn, #c47a12);
  font-weight: 600;
}
.preview-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 8px;
  padding: 8px 10px;
  background: var(--accent-soft, #e8f5ee);
  border-radius: 6px;
  font-size: 12px;
  color: var(--text, #1a2e24);
}
.foot-hint {
  margin: 10px 0 0;
  font-size: 12px;
  color: var(--muted, #6b7a85);
}
:deep(.fin-table .el-table__header th) {
  background: linear-gradient(180deg, #1a3d30 0%, #14352a 100%) !important;
  color: #e8f5ee !important;
  font-weight: 600;
}
</style>
