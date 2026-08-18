<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { purchaseApi, bizApi, financeApi } from '@erp/shared'
import ConfirmSnapshotCompare from '../../components/closed-loop/ConfirmSnapshotCompare.vue'
import TraceLotPanel from '../../components/trace/TraceLotPanel.vue'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import { useCarrierCodeLabel } from '../../composables/useCarrierCodeLabel'

const route = useRoute()

type Row = Record<string, unknown>

const { codeLabel, ensureLoaded: ensureCarrierLabel } = useCarrierCodeLabel()

const farmerCols: MobileCardColumn[] = [
  { prop: 'name', label: '姓名', primary: true },
  { prop: 'id', label: 'ID' },
  { prop: 'mobile', label: '电话' },
  { prop: 'origin', label: '产地' },
  { prop: 'default_unit_price', label: '默认单价' },
  { prop: 'status', label: '状态' },
]
const arrivalCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'estimate_weight', label: '估重' },
  { prop: 'qc_result', label: '质检' },
  { prop: 'grade', label: '等级' },
  { prop: 'status', label: '状态' },
]
const ticketCols = computed<MobileCardColumn[]>(() => [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'receive_kind', label: '模式' },
  { prop: 'batch_no', label: '溯源批号' },
  { prop: 'party_name', label: '姓名' },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'gross_weight', label: '入场重量' },
  { prop: 'net_weight', label: '净重' },
  { prop: 'settle_amount', label: '结算' },
  { prop: 'cold_store_type', label: '冷库' },
  { prop: 'status_label', label: '状态' },
  { prop: 'trace_code', label: '溯源码' },
  { prop: 'box_code', label: codeLabel.value },
])
const settlementCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '结算单', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'net_weight', label: '净重' },
  { prop: 'goods_amount', label: '货款' },
  { prop: 'freight_fee', label: '运费' },
  { prop: 'loading_fee', label: '装卸' },
  { prop: 'weigh_fee', label: '过磅费' },
  { prop: 'amount', label: '结算金额' },
  { prop: 'status', label: '状态' },
  { prop: 'transfer_no', label: '转账单号' },
]

const props = withDefaults(defineProps<{ section?: string }>(), { section: 'all' })

const showFarmers = computed(() => ['all', 'farmers'].includes(props.section || 'all'))
const showWeigh = computed(() => ['all', 'weigh'].includes(props.section || 'all'))
const showSettlements = computed(() => ['all', 'settlements'].includes(props.section || 'all'))
const showTrace = computed(() => ['all', 'trace'].includes(props.section || 'all'))

const pageTitle = computed(() => {
  switch (props.section) {
    case 'farmers': return '农户档案'
    case 'weigh': return '过磅收货'
    case 'settlements': return '农户结算'
    case 'trace': return '原料溯源'
    default: return '农户采购闭环'
  }
})

const headStats = computed(() => {
  const items = [
    showFarmers.value ? { label: '农户档案', value: farmers.value.length, tone: 'primary' } : null,
    showWeigh.value ? { label: '到货单', value: arrivals.value.length, tone: 'warning' } : null,
    showWeigh.value ? { label: '过磅单', value: tickets.value.length, tone: 'success' } : null,
    showSettlements.value ? { label: '结算单', value: settlements.value.length, tone: 'info' } : null,
  ].filter(Boolean) as { label: string; value: number; tone: string }[]
  return items
})

const farmers = ref<Row[]>([])
const arrivals = ref<Row[]>([])
const tickets = ref<Row[]>([])
const settlements = ref<Row[]>([])
const loading = ref(false)
const farmerDlg = ref(false)
const farmerForm = reactive({ name: '', mobile: '', origin: '', remark: '', default_unit_price: 1 })
const arrivalForm = reactive({
  farmer_id: 0,
  origin: '',
  variety: '鲜木薯',
  estimate_weight: 1000,
  source_type: 'self',
  channel: 'internal',
  qc_image_url: '',
  plate_no: '',
  receive_address: '',
  pass_rate: 100,
  reject_weight: 0,
  freight_fee: 0,
  loading_fee: 0,
  weigh_fee: 0,
})
const weighForm = reactive({
  receive_kind: 'gate' as 'gate' | 'stockin',
  arrival_id: 0,
  farmer_id: 0,
  channel: 'internal',
  gross_weight: 1000,
  deduct_rate: 0.05,
  reject_weight: 0,
  unit_price: 1.2,
  net_weight: 100,
  bag_qty: 0,
  cold_store_type: 'fresh',
  variety: '鲜木薯',
  variety_id: 0,
  product_id: 1,
  source_type: 'self',
  batch_no: '',
  image_url: '',
  image_urls: [] as string[],
  ocr_draft_json: '',
  plate_no: '',
  receive_address: '',
  origin: '',
  party_name: '',
  party_mobile: '',
  pass_rate: 100,
  freight_fee: 0,
  loading_fee: 0,
  weigh_fee: 0,
})
const batchValid = ref(false)
const batchBoundFarmer = ref('')
const batchInputMode = ref<'scan' | 'manual'>('manual')
const farmerOptions = ref<Row[]>([])
const farmerSearchLoading = ref(false)
const onsiteFarmerDlg = ref(false)
const onsiteFarmer = reactive({ name: '', mobile: '', origin: '' })
const varieties = ref<Row[]>([])
const confirmDlg = ref(false)
const confirmTicket = ref<Row | null>(null)
const confirmModel = ref<Record<string, unknown>>({
  gross_weight: '',
  deduct_rate: '',
  deduct_weight: '',
  net_weight: '',
})
const payDlg = ref(false)
const payRow = ref<Row | null>(null)
const payForm = reactive({ transfer_no: '', pay_evidence_url: '', fund_account_id: 0 })
const fundAccounts = ref<Row[]>([])
const correctDlg = ref(false)
const correctForm = reactive({ biz_type: 'weigh_ticket', biz_id: 0, reason: '', unit_price: '', net_weight: '' })
const labelPreview = ref<Row | null>(null)
const traceCode = ref('')
const traceListKeyword = ref('')
const selectedTraceTicketId = ref<number | null>(null)

const traceTicketCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'trace_code', label: '溯源码' },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'net_weight', label: '净重' },
  { prop: 'status_label', label: '状态' },
]

const ticketsWithTrace = computed(() => {
  const kw = traceListKeyword.value.trim().toLowerCase()
  return tickets.value.filter((t) => {
    const code = String(t.trace_code || '').trim()
    if (!code && !String(t.batch_no || '').trim() && !String(t.doc_no || '').trim()) return false
    if (!kw) return Boolean(code || t.batch_no || t.doc_no)
    const hay = [t.doc_no, t.trace_code, t.batch_no, t.farmer_name, t.party_name, t.status, weighTicketStatusLabel(t)]
      .map((x) => String(x || '').toLowerCase())
      .join(' ')
    return hay.includes(kw)
  }).map(withWeighStatusLabel)
})

function weighTicketStatusLabel(row: Row) {
  const st = String(row.status || '').toLowerCase()
  const kind = String(row.receive_kind || 'gate').toLowerCase()
  const phase = String(row.process_phase || '')
  if (phase === 'await_gate' || (st === 'weighed' && kind !== 'stockin')) return '待入厂'
  if (phase === 'await_stockin' || st === 'gate_accepted') return '待入库'
  if (phase === 'await_warehouse' || st === 'weighed') return '待仓管确认'
  if (phase === 'await_finance') return '已入仓·待结算'
  if (phase === 'settled') return '已结清'
  if (phase === 'stocked_done' || st === 'stocked') return kind === 'gate' ? '已入仓·待结算' : '已入库'
  if (st === 'returned') return '仓管已退回'
  if (st === 'draft') return '草稿'
  if (st === 'pending_confirm' || st === 'qc_pass') return '待绑定'
  return st || '-'
}

function withWeighStatusLabel(row: Row): Row {
  return { ...row, status_label: weighTicketStatusLabel(row) }
}

const ticketsView = computed(() => tickets.value.map(withWeighStatusLabel))

const labelPreviewFields = computed(() => {
  const m = labelPreview.value
  if (!m) return [] as { label: string; value: string }[]
  const kv = (label: string, v: unknown) =>
    v != null && v !== '' ? { label, value: String(v) } : null
  return [
    kv('溯源码', m.trace_code || m.code),
    kv('批号', m.batch_no),
    kv('品种', m.variety || m.product_name),
    kv('净重', m.net_weight != null ? `${m.net_weight} kg` : null),
    kv('农户', m.farmer_name || m.party_name),
    kv('单号', m.doc_no),
  ].filter(Boolean) as { label: string; value: string }[]
})

const ocrDraft = computed(() => {
  const raw = String(confirmTicket.value?.ocr_draft_json || weighForm.ocr_draft_json || '')
  if (!raw) return null
  try {
    return JSON.parse(raw) as Record<string, unknown>
  } catch {
    return { raw }
  }
})

async function refresh() {
  loading.value = true
  try {
    const [f, a, t, s, v] = await Promise.all([
      purchaseApi.farmers(),
      purchaseApi.arrivals(),
      purchaseApi.weighTickets(),
      purchaseApi.farmerSettlements(),
      purchaseApi.weighVarieties('status=active'),
    ])
    farmers.value = ((f.data as { list?: Row[] })?.list) || []
    farmerOptions.value = farmers.value.slice(0, 30)
    arrivals.value = ((a.data as { list?: Row[] })?.list) || []
    tickets.value = ((t.data as { list?: Row[] })?.list) || []
    settlements.value = ((s.data as { list?: Row[] })?.list) || []
    varieties.value = ((v.data as { list?: Row[] })?.list) || []
    if (farmers.value.length) {
      if (!arrivalForm.farmer_id) arrivalForm.farmer_id = Number(farmers.value[0].id)
    }
    if (varieties.value.length && !weighForm.variety_id) {
      onVarietyChange(Number(varieties.value[0].id))
    }
  } finally {
    loading.value = false
  }
}

async function createFarmer() {
  if (!farmerForm.name) return ElMessage.warning('请填写农户姓名')
  const res = await purchaseApi.createFarmer({ ...farmerForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('农户已建档')
  farmerDlg.value = false
  farmerForm.name = ''
  farmerForm.mobile = ''
  farmerForm.origin = ''
  farmerForm.remark = ''
  farmerForm.default_unit_price = 1
  await refresh()
}

function openFarmerDialog() {
  farmerDlg.value = true
}

async function createArrival() {
  if (!arrivalForm.farmer_id) return ElMessage.warning('请选择农户')
  const res = await purchaseApi.createArrival({ ...arrivalForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('到货单已创建，请质检定级')
  await refresh()
}

async function qcArrival(id: number, pass: boolean) {
  const grade = pass
    ? ((await ElMessageBox.prompt('请输入等级 A/B/C', '质检定级', { inputValue: 'A' })).value || 'A')
    : ''
  const img = arrivalForm.qc_image_url
  const res = await purchaseApi.qcArrival(id, {
    qc_result: pass ? 'pass' : 'fail',
    grade: String(grade).toUpperCase(),
    qc_image_url: img,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(pass ? '质检合格，可过磅出码' : '质检不合格，不可过磅')
  await refresh()
}

async function searchFarmers(q: string) {
  const kw = String(q || '').trim()
  if (!kw) {
    farmerOptions.value = farmers.value.slice(0, 30)
    return
  }
  farmerSearchLoading.value = true
  try {
    let params = `page_size=30&keyword=${encodeURIComponent(kw)}`
    if (/^\d+$/.test(kw) && kw.length <= 6) params = `page_size=30&id=${encodeURIComponent(kw)}`
    else if (/^1\d{10}$/.test(kw) || /^\d{7,}$/.test(kw)) params = `page_size=30&mobile=${encodeURIComponent(kw)}`
    const res = await purchaseApi.farmers(params)
    farmerOptions.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    farmerSearchLoading.value = false
  }
}

function applyFarmer(row: Row | undefined) {
  if (!row) {
    weighForm.farmer_id = 0
    return
  }
  weighForm.farmer_id = Number(row.id || 0)
  weighForm.party_name = String(row.name || '')
  weighForm.party_mobile = String(row.mobile || '')
  weighForm.origin = String(row.origin || weighForm.origin || '')
  const price = Number(row.default_unit_price || 0)
  if (price > 0) weighForm.unit_price = price
}

function onFarmerSelect(id: number) {
  const row = farmerOptions.value.find((x) => Number(x.id) === id) || farmers.value.find((x) => Number(x.id) === id)
  applyFarmer(row)
}

function openOnsiteFarmer() {
  onsiteFarmer.name = weighForm.party_name || ''
  onsiteFarmer.mobile = weighForm.party_mobile || ''
  onsiteFarmer.origin = weighForm.origin || ''
  onsiteFarmerDlg.value = true
}

async function saveOnsiteFarmer() {
  if (!onsiteFarmer.name.trim()) return ElMessage.warning('请填写农户姓名')
  const res = await purchaseApi.createFarmer({
    name: onsiteFarmer.name.trim(),
    mobile: onsiteFarmer.mobile.trim(),
    origin: onsiteFarmer.origin.trim(),
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const row = (res.data as Row) || {}
  farmerOptions.value = [row, ...farmerOptions.value]
  farmers.value = [row, ...farmers.value]
  applyFarmer(row)
  onsiteFarmerDlg.value = false
  ElMessage.success(`已现场建档并关联 #${row.id}`)
}

async function validateBatch() {
  const code = String(weighForm.batch_no || '').trim().toUpperCase()
  weighForm.batch_no = code
  batchBoundFarmer.value = ''
  if (!code) {
    batchValid.value = false
    return ElMessage.warning('请填写溯源批号')
  }
  const res = await purchaseApi.validateTraceBatchCode({
    code,
    receive_kind: weighForm.receive_kind,
  })
  batchValid.value = res.code === 1
  if (res.code !== 1) return ElMessage.error(res.msg)
  const data = (res.data || {}) as Row
  batchBoundFarmer.value = String(data.farmer_name || data.party_name || '')
  ElMessage.success(
    weighForm.receive_kind === 'stockin' && batchBoundFarmer.value
      ? `批号校验通过 · 关联农户 ${batchBoundFarmer.value}`
      : '批号校验通过',
  )
}

function onReceiveKindChange() {
  batchValid.value = false
  batchBoundFarmer.value = ''
  if (weighForm.receive_kind === 'stockin') {
    weighForm.farmer_id = 0
    weighForm.party_name = ''
    weighForm.party_mobile = ''
    weighForm.origin = ''
  }
}

function onBatchScanEnter() {
  void validateBatch()
}

async function uploadSitePhoto(file: File) {
  const res = await bizApi.upload(file)
  if (res.code !== 1) {
    ElMessage.error(res.msg)
    return false
  }
  const url = String((res.data as Row)?.url || (res.data as Row)?.file_url || '')
  if (!url) {
    ElMessage.error('上传无返回地址')
    return false
  }
  weighForm.image_urls = [...weighForm.image_urls, url].slice(0, 3)
  weighForm.image_url = weighForm.image_urls[0] || ''
  ElMessage.success('照片已上传')
  return false
}

async function createWeigh() {
  weighForm.receive_kind = 'gate'
  if (!weighForm.batch_no) return ElMessage.warning('溯源批号必填')
  if (!batchValid.value) {
    await validateBatch()
    if (!batchValid.value) return
  }
  if (!weighForm.image_url) return ElMessage.warning('请上传现场照片')
  if (!weighForm.variety_id && !weighForm.variety) return ElMessage.warning('请选择过磅品种')
  if (!weighForm.farmer_id && !String(weighForm.party_name || '').trim()) {
    return ElMessage.warning('入厂须选择或现场录入农户')
  }
  const body: Record<string, unknown> = {
    receive_kind: 'gate',
    batch_no: weighForm.batch_no,
    channel: weighForm.channel,
    product_id: weighForm.product_id,
    variety_id: weighForm.variety_id || undefined,
    variety: weighForm.variety,
    source_type: weighForm.source_type,
    image_url: weighForm.image_url,
    image_urls: weighForm.image_urls,
    pass_rate: weighForm.pass_rate,
    reject_weight: weighForm.reject_weight,
  }
  if (weighForm.arrival_id) body.arrival_id = weighForm.arrival_id
  body.farmer_id = weighForm.farmer_id || 0
  body.party_name = weighForm.party_name
  body.party_mobile = weighForm.party_mobile
  body.origin = weighForm.origin
  body.gross_weight = weighForm.gross_weight
  body.deduct_rate = weighForm.deduct_rate
  body.unit_price = weighForm.unit_price
  body.plate_no = weighForm.plate_no
  body.receive_address = weighForm.receive_address
  body.freight_fee = weighForm.freight_fee
  body.loading_fee = weighForm.loading_fee
  body.weigh_fee = weighForm.weigh_fee
  if (weighForm.ocr_draft_json) {
    try {
      body.ocr_draft_json = JSON.parse(weighForm.ocr_draft_json)
    } catch {
      body.ocr_draft_json = { text: weighForm.ocr_draft_json }
    }
  }
  const res = await purchaseApi.createWeighTicket(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`草稿已建入厂，净重 ${(res.data as Row)?.net_weight}`)
  weighForm.batch_no = ''
  weighForm.image_url = ''
  weighForm.image_urls = []
  batchValid.value = false
  batchBoundFarmer.value = ''
  await refresh()
}

function onVarietyChange(id: number) {
  weighForm.variety_id = id
  const row = varieties.value.find((x) => Number(x.id) === id)
  if (!row) return
  weighForm.variety = String(row.name || '')
  const pid = Number(row.default_product_id || 0)
  weighForm.product_id = pid > 0 ? pid : 1
}

function openConfirm(row: Row) {
  confirmTicket.value = row
  confirmModel.value = {
    gross_weight: row.gross_weight,
    deduct_rate: row.deduct_rate,
    deduct_weight: row.deduct_weight,
    net_weight: row.net_weight,
  }
  confirmDlg.value = true
}

async function doConfirm() {
  const id = Number(confirmTicket.value?.id)
  const m = confirmModel.value
  const res = await purchaseApi.confirmWeighTicket(id, {
    gross_weight: Number(m.gross_weight),
    deduct_rate: Number(m.deduct_rate),
    deduct_weight: Number(m.deduct_weight),
    net_weight: Number(m.net_weight),
    confirmed: true,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已确认出码 ${(res.data as Row)?.trace_code || ''}`)
  confirmDlg.value = false
  const label = await purchaseApi.labelWeighTicket(id)
  if (label.code === 1) labelPreview.value = (label.data as Row) || null
  await refresh()
}

async function loadFundAccounts() {
  try {
    const r = await financeApi.fundAccounts()
    fundAccounts.value = ((r.data as { list?: Row[] })?.list) || []
    if (!payForm.fund_account_id && fundAccounts.value[0]) {
      payForm.fund_account_id = Number(fundAccounts.value[0].id)
    }
  } catch {
    fundAccounts.value = []
  }
}

function openPay(row: Row) {
  payRow.value = row
  payForm.transfer_no = ''
  payForm.pay_evidence_url = ''
  payForm.fund_account_id = fundAccounts.value[0] ? Number(fundAccounts.value[0].id) : 0
  payDlg.value = true
  void loadFundAccounts()
}

async function uploadPayEvidence(file: File) {
  const res = await bizApi.upload(file)
  if (res.code !== 1) {
    ElMessage.error(res.msg)
    return false
  }
  const url = String((res.data as Row)?.url || (res.data as Row)?.file_url || '')
  if (!url) {
    ElMessage.error('上传无返回地址')
    return false
  }
  payForm.pay_evidence_url = url
  ElMessage.success('发票/转账截图已上传')
  return false
}

async function doPay() {
  if (!payRow.value) return
  if (!payForm.transfer_no || !payForm.pay_evidence_url) return ElMessage.warning('转账单号与发票/转账截图必填')
  if (!payForm.fund_account_id) return ElMessage.warning('请选择资金账户')
  const res = await purchaseApi.payFarmerSettlement(Number(payRow.value.id), { ...payForm })
  if (res.code !== 1) {
    const msg = res.msg === 'PERIOD_CLOSED' ? '该期间已月结，不可入账' : res.msg === 'FUND_ACCOUNT_REQUIRED' ? '请选择资金账户' : res.msg
    return ElMessage.error(msg)
  }
  ElMessage.success(
    res.data && (res.data as Row).ticket_id
      ? `已支付关单，关联工单 #${(res.data as Row).ticket_id} 已办结`
      : '已支付关单',
  )
  payDlg.value = false
  await refresh()
}

function openCorrect(row: Row, bizType: string) {
  correctForm.biz_type = bizType
  correctForm.biz_id = Number(row.id)
  correctForm.reason = ''
  correctForm.unit_price = ''
  correctForm.net_weight = ''
  correctDlg.value = true
}

async function doCorrect() {
  if (!correctForm.reason) return ElMessage.warning('请填写纠错原因')
  const fields: Record<string, unknown> = {}
  if (correctForm.biz_type === 'farmer_settlement' && correctForm.unit_price !== '') {
    fields.unit_price = Number(correctForm.unit_price)
  }
  if (correctForm.biz_type === 'weigh_ticket' && correctForm.net_weight !== '') {
    fields.net_weight = Number(correctForm.net_weight)
  }
  const res = await bizApi.correct({
    biz_type: correctForm.biz_type,
    biz_id: correctForm.biz_id,
    action: 'correct',
    reason: correctForm.reason,
    fields,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已记录纠错审计')
  correctDlg.value = false
  await refresh()
}

function pickTraceCode(row: Row): string {
  return String(row.trace_code || row.batch_no || row.doc_no || '').trim()
}

function selectTraceTicket(row: Row) {
  selectedTraceTicketId.value = Number(row.id) || null
  const code = pickTraceCode(row)
  if (!code) {
    ElMessage.warning('该单据暂无溯源码/批号')
    return
  }
  traceCode.value = code
}

async function applyTraceQueryCode() {
  if (!showTrace.value) return
  const code = String(route.query.code || '').trim()
  if (!code) return
  selectedTraceTicketId.value = null
  traceCode.value = code
}

onMounted(async () => {
  await ensureCarrierLabel()
  await refresh()
  await applyTraceQueryCode()
})

watch(
  () => [route.query.code, props.section] as const,
  () => {
    void applyTraceQueryCode()
  },
)
</script>

<template>
  <div class="page" v-loading="loading">
    <section class="page-head">
      <div>
        <h2 class="page-title">{{ pageTitle }}</h2>
        <p class="hint">
          到货拍照质检定级 → 过磅图+自动预填 → 对照原图确认出码贴标 → 仓管扫码入库 → 财务转账回单关单。业务可查可改不可删。
        </p>
      </div>
      <div v-if="headStats.length" class="head-stats">
        <div v-for="item in headStats" :key="item.label" class="stat-card" :data-tone="item.tone">
          <span class="stat-label">{{ item.label }}</span>
          <strong class="stat-value">{{ item.value }}</strong>
        </div>
      </div>
    </section>
    <el-alert v-if="showWeigh" type="warning" show-icon :closable="false" class="mb" style="margin-bottom:12px"
      title="日常过磅收货请优先使用 Flutter App「过磅收货」。管理端用于审核、查询与补单。" />

    <el-row v-if="showWeigh" :gutter="16" class="top-panels">
      <template v-if="showWeigh">
        <el-col :span="12" :xs="24">
          <el-card class="section-card" shadow="hover">
            <template #header>
              <div class="card-head">
                <span>到货 + 质检照</span>
                <small>先建到货单，再做质检定级</small>
              </div>
            </template>
            <el-form label-width="90px" size="small" class="stack-form">
              <el-form-item label="农户">
                <el-select v-model="arrivalForm.farmer_id" style="width:100%">
                  <el-option v-for="f in farmers" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
                </el-select>
              </el-form-item>
              <el-form-item label="估重"><el-input-number v-model="arrivalForm.estimate_weight" :min="0" /></el-form-item>
              <el-form-item label="车牌"><el-input v-model="arrivalForm.plate_no" /></el-form-item>
              <el-form-item label="收货地址"><el-input v-model="arrivalForm.receive_address" /></el-form-item>
              <el-form-item label="合格率%"><el-input-number v-model="arrivalForm.pass_rate" :min="0" :max="100" /></el-form-item>
              <el-form-item label="不合格重"><el-input-number v-model="arrivalForm.reject_weight" :min="0" /></el-form-item>
              <el-form-item label="运费"><el-input-number v-model="arrivalForm.freight_fee" :min="0" :step="1" /></el-form-item>
              <el-form-item label="装卸费"><el-input-number v-model="arrivalForm.loading_fee" :min="0" :step="1" /></el-form-item>
              <el-form-item label="过磅费"><el-input-number v-model="arrivalForm.weigh_fee" :min="0" :step="1" /></el-form-item>
              <el-form-item label="质检照URL"><el-input v-model="arrivalForm.qc_image_url" placeholder="必填证据" /></el-form-item>
              <el-button type="primary" @click="createArrival">创建到货单</el-button>
            </el-form>
          </el-card>
        </el-col>
        <el-col :span="12" :xs="24">
          <el-card class="section-card" shadow="hover">
            <template #header>
              <div class="card-head">
                <span>过磅草稿（入厂）</span>
                <small>支持批号校验、现场录入与照片留痕</small>
              </div>
            </template>
            <el-form label-width="100px" size="small" class="stack-form">
              <el-alert
                type="info"
                :closable="false"
                show-icon
                class="mb"
                title="入厂后由仓管扫溯源分板入库；农户结算环节可在「系统管理 → 基础设置」配置"
                style="margin-bottom:12px"
              />
              <el-form-item label="溯源批号">
                <div style="width:100%">
                  <el-radio-group v-model="batchInputMode" size="small" style="margin-bottom:8px" @change="batchValid = false; batchBoundFarmer = ''">
                    <el-radio-button value="scan">扫描输入</el-radio-button>
                    <el-radio-button value="manual">手动输入</el-radio-button>
                  </el-radio-group>
                  <div style="display:flex;gap:8px;width:100%">
                    <el-input
                      v-model="weighForm.batch_no"
                      :placeholder="
                        batchInputMode === 'scan'
                          ? '扫码枪扫入后回车校验'
                          : '手输批号后点校验'
                      "
                      @change="batchValid = false; batchBoundFarmer = ''"
                      @keyup.enter="onBatchScanEnter"
                    />
                    <el-button @click="validateBatch">校验</el-button>
                  </div>
                  <div v-if="batchInputMode === 'scan'" class="hint">扫描模式：光标在输入框内，扫码枪楔入后回车自动校验</div>
                </div>
              </el-form-item>
              <el-form-item label="到货单">
                <el-select v-model="weighForm.arrival_id" clearable style="width:100%" placeholder="可选">
                  <el-option
                    v-for="a in arrivals.filter((x) => x.status === 'qc_pass')"
                    :key="String(a.id)"
                    :label="`${a.doc_no} ${a.farmer_name} ${a.grade}`"
                    :value="Number(a.id)"
                  />
                </el-select>
              </el-form-item>
              <template>
                <el-form-item label="农户搜索">
                  <div style="display:flex;gap:8px;width:100%">
                    <el-select
                      v-model="weighForm.farmer_id"
                      filterable
                      remote
                      clearable
                      reserve-keyword
                      placeholder="手机号 / 姓名 / ID"
                      :remote-method="searchFarmers"
                      :loading="farmerSearchLoading"
                      style="flex:1"
                      @change="onFarmerSelect"
                    >
                      <el-option
                        v-for="f in farmerOptions"
                        :key="String(f.id)"
                        :label="`${f.name} ${f.mobile || ''} (#${f.id})`"
                        :value="Number(f.id)"
                      />
                    </el-select>
                    <el-button @click="openOnsiteFarmer">现场录入</el-button>
                  </div>
                </el-form-item>
                <el-form-item label="姓名"><el-input v-model="weighForm.party_name" /></el-form-item>
                <el-form-item label="电话"><el-input v-model="weighForm.party_mobile" /></el-form-item>
                <el-form-item label="产地"><el-input v-model="weighForm.origin" /></el-form-item>
              </template>
              <el-form-item label="品种">
                <el-select
                  :model-value="weighForm.variety_id || undefined"
                  style="width:100%"
                  placeholder="请选择过磅品种"
                  @update:model-value="(v: number) => onVarietyChange(v)"
                >
                  <el-option v-for="x in varieties" :key="String(x.id)" :label="String(x.name)" :value="Number(x.id)" />
                </el-select>
              </el-form-item>
              <template>
                <el-form-item label="入场重量"><el-input-number v-model="weighForm.gross_weight" :min="0" /></el-form-item>
                <el-form-item label="扣损率"><el-input-number v-model="weighForm.deduct_rate" :min="0" :max="1" :step="0.01" /></el-form-item>
                <el-form-item label="不合格重"><el-input-number v-model="weighForm.reject_weight" :min="0" /></el-form-item>
                <el-form-item label="单价"><el-input-number v-model="weighForm.unit_price" :min="0" :step="0.1" /></el-form-item>
                <el-form-item label="车牌"><el-input v-model="weighForm.plate_no" /></el-form-item>
                <el-form-item label="收货地址"><el-input v-model="weighForm.receive_address" /></el-form-item>
                <el-form-item label="运费"><el-input-number v-model="weighForm.freight_fee" :min="0" /></el-form-item>
                <el-form-item label="装卸费"><el-input-number v-model="weighForm.loading_fee" :min="0" /></el-form-item>
                <el-form-item label="过磅费"><el-input-number v-model="weighForm.weigh_fee" :min="0" /></el-form-item>
              </template>
              <el-form-item label="现场照片">
                <el-upload :show-file-list="false" :before-upload="uploadSitePhoto" accept="image/*">
                  <el-button>上传照片</el-button>
                </el-upload>
                <div v-if="weighForm.image_urls.length" class="photos">
                  <el-image
                    v-for="(u, i) in weighForm.image_urls"
                    :key="u"
                    :src="u"
                    style="width: 56px; height: 56px; margin-right: 6px"
                    fit="cover"
                    :preview-src-list="weighForm.image_urls"
                    :initial-index="i"
                  />
                </div>
              </el-form-item>
              <el-button type="primary" @click="createWeigh">创建过磅草稿</el-button>
            </el-form>
          </el-card>
        </el-col>
      </template>
    </el-row>

    <el-card v-if="showFarmers" class="section-card" shadow="hover" style="margin-top:16px">
      <template #header>
        <div class="card-head card-head-row">
          <div>
            <span>农户列表</span>
            <small>快速查看建档状态与默认单价</small>
          </div>
          <el-button type="primary" plain size="small" @click="openFarmerDialog">新建农户</el-button>
        </div>
      </template>
      <TableOrCards :data="farmers" :loading="loading" :columns="farmerCols">
        <el-table :data="farmers" size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="name" label="姓名" width="120" />
          <el-table-column prop="mobile" label="电话" width="130" />
          <el-table-column prop="origin" label="产地" />
          <el-table-column prop="default_unit_price" label="默认单价" width="100" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </TableOrCards>
    </el-card>

    <el-card v-if="showWeigh" class="section-card" shadow="hover" style="margin-top:16px">
      <template #header>
        <div class="card-head">
          <span>到货质检</span>
          <small>定级后才能进入后续过磅流程</small>
        </div>
      </template>
      <TableOrCards :data="arrivals" :loading="loading" :columns="arrivalCols">
        <el-table :data="arrivals" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="farmer_name" label="农户" width="100" />
          <el-table-column prop="estimate_weight" label="估重" width="80" />
          <el-table-column prop="qc_result" label="质检" width="70" />
          <el-table-column prop="grade" label="等级" width="60" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status==='qc_pending' || row.status==='draft'" link type="success" @click="qcArrival(Number(row.id), true)">合格定级</el-button>
              <el-button v-if="row.status==='qc_pending' || row.status==='draft'" link type="danger" @click="qcArrival(Number(row.id), false)">不合格</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='qc_pending' || row.status==='draft'" link type="success" @click="qcArrival(Number(row.id), true)">合格定级</el-button>
          <el-button v-if="row.status==='qc_pending' || row.status==='draft'" link type="danger" @click="qcArrival(Number(row.id), false)">不合格</el-button>
        </template>
      </TableOrCards>
    </el-card>

    <el-card v-if="showWeigh" class="section-card" shadow="hover" style="margin-top:16px">
      <template #header>
        <div class="card-head">
          <span>过磅确认 / 出码</span>
          <small>入库仍需到仓管待办继续处理</small>
        </div>
      </template>
      <p class="hint">确认出码后系统将溯源码与单号推送给仓管；仓管确认后方为采购完成。</p>
      <TableOrCards :data="ticketsView" :loading="loading" :columns="ticketCols">
        <el-table :data="ticketsView" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="receive_kind" label="模式" width="70">
            <template #default="{ row }">{{ row.receive_kind === 'stockin' ? '入库' : '入厂' }}</template>
          </el-table-column>
          <el-table-column prop="batch_no" label="溯源批号" min-width="160" />
          <el-table-column prop="party_name" label="姓名" width="90" />
          <el-table-column prop="farmer_name" label="农户" width="90" />
          <el-table-column prop="gross_weight" label="入场重量" width="90" />
          <el-table-column prop="net_weight" label="净重" width="70" />
          <el-table-column prop="settle_amount" label="结算" width="80" />
          <el-table-column prop="cold_store_type" label="冷库" width="80" />
          <el-table-column prop="image_url" label="照片" width="70">
            <template #default="{ row }">
              <el-image v-if="row.image_url" :src="String(row.image_url)" style="width:36px;height:36px" fit="cover" :preview-src-list="[String(row.image_url)]" />
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">{{ weighTicketStatusLabel(row) }}</template>
          </el-table-column>
          <el-table-column prop="trace_code" label="溯源码" min-width="160" />
          <el-table-column prop="box_code" :label="codeLabel" min-width="120" />
          <el-table-column label="操作" width="280" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft' || row.status==='qc_pass'" link type="primary" @click="openConfirm(row)">对照确认出码</el-button>
              <el-button v-if="row.status==='weighed'" link type="info" disabled>{{ row.receive_kind === 'stockin' ? '等待仓管确认入库' : '待入厂' }}</el-button>
              <el-button v-if="row.status==='gate_accepted'" link type="warning" disabled>待入库</el-button>
              <el-button v-if="row.status==='stocked'" link type="success" disabled>已分板入库</el-button>
              <el-button link type="warning" @click="openCorrect(row, 'weigh_ticket')">纠错</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft' || row.status==='qc_pass'" link type="primary" @click="openConfirm(row)">对照确认出码</el-button>
          <el-button v-if="row.status==='weighed'" link type="info" disabled>{{ row.receive_kind === 'stockin' ? '等待仓管确认入库' : '待入厂' }}</el-button>
          <el-button v-if="row.status==='gate_accepted'" link type="warning" disabled>待入库</el-button>
          <el-button v-if="row.status==='stocked'" link type="success" disabled>已分板入库</el-button>
          <el-button link type="warning" @click="openCorrect(row, 'weigh_ticket')">纠错</el-button>
        </template>
      </TableOrCards>
      <div v-if="labelPreview" class="label-preview">
        <h4 class="sec">标签预览</h4>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item v-for="f in labelPreviewFields" :key="f.label" :label="f.label">
            {{ f.value }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-card>

    <!-- 原料溯源：列表 + 详情 -->
    <el-row v-if="showTrace && props.section === 'trace'" :gutter="16" style="margin-top:16px">
      <el-col :span="10" :xs="24">
        <el-card class="section-card" shadow="hover">
          <template #header>
            <div class="card-head">
              <span>过磅单据</span>
              <small>点选后在右侧做溯源倒查</small>
            </div>
          </template>
          <el-input
            v-model="traceListKeyword"
            clearable
            placeholder="过滤单号/溯源码/农户"
            size="small"
            style="margin-bottom:10px"
          />
          <TableOrCards :data="ticketsWithTrace" :loading="loading" :columns="traceTicketCols">
            <el-table
              :data="ticketsWithTrace"
              size="small"
              highlight-current-row
              :row-class-name="({ row }: { row: Row }) => (Number(row.id) === selectedTraceTicketId ? 'is-selected' : '')"
              @row-click="selectTraceTicket"
            >
              <el-table-column prop="doc_no" label="单号" min-width="120" show-overflow-tooltip />
              <el-table-column prop="trace_code" label="溯源码" min-width="140" show-overflow-tooltip />
              <el-table-column label="农户" width="90">
                <template #default="{ row }">{{ row.farmer_name || row.party_name || '-' }}</template>
              </el-table-column>
              <el-table-column prop="net_weight" label="净重" width="70" />
              <el-table-column label="状态" width="90">
                <template #default="{ row }">{{ weighTicketStatusLabel(row) }}</template>
              </el-table-column>
                <template #default="{ row }">
                  <el-button link type="primary" @click.stop="selectTraceTicket(row)">倒查</el-button>
                </template>
              </el-table-column>
            </el-table>
            <template #actions="{ row }">
              <el-button link type="primary" @click="selectTraceTicket(row)">倒查</el-button>
            </template>
          </TableOrCards>
        </el-card>
      </el-col>
      <el-col :span="14" :xs="24">
        <el-card class="section-card" shadow="hover">
          <template #header>
            <div class="card-head">
              <span>溯源详情</span>
              <small>支持按单号 / 溯源码追溯</small>
            </div>
          </template>
          <TraceLotPanel v-model:code="traceCode" />
        </el-card>
      </el-col>
    </el-row>

    <el-row v-if="showSettlements || (showTrace && props.section !== 'trace')" :gutter="16" style="margin-top:16px">
      <el-col v-if="showSettlements" :span="showTrace && props.section !== 'trace' ? 14 : 24" :xs="24">
        <el-card class="section-card" shadow="hover">
          <template #header>
            <div class="card-head">
              <span>农户结算</span>
              <small>财务支付需补齐转账单号与回单凭证</small>
            </div>
          </template>
          <TableOrCards :data="settlements" :loading="loading" :columns="settlementCols">
            <el-table :data="settlements" size="small">
              <el-table-column prop="doc_no" label="结算单" width="140" />
              <el-table-column prop="farmer_name" label="农户" width="90" />
              <el-table-column prop="net_weight" label="净重" width="80" />
              <el-table-column prop="goods_amount" label="货款" width="80" />
              <el-table-column prop="freight_fee" label="运费" width="70" />
              <el-table-column prop="loading_fee" label="装卸" width="70" />
              <el-table-column prop="weigh_fee" label="过磅费" width="70" />
              <el-table-column prop="amount" label="结算金额" width="90" />
              <el-table-column prop="status" label="状态" width="110" />
              <el-table-column prop="transfer_no" label="转账单号" min-width="120" />
              <el-table-column label="操作" width="160" fixed="right">
                <template #default="{ row }">
                  <el-button v-if="row.status!=='settle_paid'" link type="primary" @click="openPay(row)">支付关单</el-button>
                  <el-button link type="warning" @click="openCorrect(row, 'farmer_settlement')">纠错</el-button>
                </template>
              </el-table-column>
            </el-table>
            <template #actions="{ row }">
              <el-button v-if="row.status!=='settle_paid'" link type="primary" @click="openPay(row)">支付关单</el-button>
              <el-button link type="warning" @click="openCorrect(row, 'farmer_settlement')">纠错</el-button>
            </template>
          </TableOrCards>
        </el-card>
      </el-col>
      <el-col v-if="showTrace && props.section !== 'trace'" :span="showSettlements ? 10 : 24" :xs="24">
        <el-card class="section-card" shadow="hover">
          <template #header>
            <div class="card-head">
              <span>溯源倒查</span>
              <small>辅助核对批次与单据链路</small>
            </div>
          </template>
          <TraceLotPanel v-model:code="traceCode" />
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="confirmDlg" title="原图与数值并排确认" width="860px">
      <ConfirmSnapshotCompare
        v-model="confirmModel"
        :image-url="String(confirmTicket?.image_url || '')"
        :draft="ocrDraft"
        :fields="[
          { key: 'gross_weight', label: '毛重', highlight: true },
          { key: 'deduct_rate', label: '扣损率' },
          { key: 'deduct_weight', label: '扣损重' },
          { key: 'net_weight', label: '净重', highlight: true },
        ]"
      />
      <template #footer>
        <el-button @click="confirmDlg = false">取消</el-button>
        <el-button type="primary" @click="doConfirm">确认无误并出码</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="payDlg" title="财务支付关单" width="480px">
      <el-form label-width="110px">
        <el-form-item label="资金账户" required>
          <el-select v-model="payForm.fund_account_id" style="width:100%" placeholder="付款账户">
            <el-option v-for="f in fundAccounts" :key="String(f.id)" :label="`${f.name}（余额 ${f.balance}）`" :value="Number(f.id)" />
          </el-select>
        </el-form-item>
        <el-form-item label="转账单号" required><el-input v-model="payForm.transfer_no" /></el-form-item>
        <el-form-item label="发票/转账截图" required>
          <el-upload :show-file-list="false" :http-request="(opt: any) => uploadPayEvidence(opt.file)" accept="image/*">
            <el-button type="primary" plain>上传图片</el-button>
          </el-upload>
          <div v-if="payForm.pay_evidence_url" style="margin-top:8px">
            <el-image :src="payForm.pay_evidence_url" style="width:160px;height:100px" fit="contain" />
            <div class="muted" style="font-size:12px;word-break:break-all">{{ payForm.pay_evidence_url }}</div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="payDlg = false">取消</el-button>
        <el-button type="primary" @click="doPay">确认已付</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="correctDlg" title="纠错（审计必填原因）" width="480px">
      <el-form label-width="100px">
        <el-form-item label="原因"><el-input v-model="correctForm.reason" type="textarea" /></el-form-item>
        <el-form-item v-if="correctForm.biz_type==='farmer_settlement'" label="单价">
          <el-input v-model="correctForm.unit_price" />
        </el-form-item>
        <el-form-item v-if="correctForm.biz_type==='weigh_ticket'" label="净重">
          <el-input v-model="correctForm.net_weight" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="correctDlg = false">取消</el-button>
        <el-button type="warning" @click="doCorrect">提交纠错</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="onsiteFarmerDlg" title="现场录入农户（平台共享）" width="480px">
      <el-form label-width="90px">
        <el-form-item label="姓名" required><el-input v-model="onsiteFarmer.name" /></el-form-item>
        <el-form-item label="手机号"><el-input v-model="onsiteFarmer.mobile" /></el-form-item>
        <el-form-item label="产地地址"><el-input v-model="onsiteFarmer.origin" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="onsiteFarmerDlg = false">取消</el-button>
        <el-button type="primary" @click="saveOnsiteFarmer">保存并关联</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="farmerDlg" title="新建农户" width="460px">
      <el-form label-width="90px" class="stack-form">
        <el-form-item label="姓名" required><el-input v-model="farmerForm.name" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="farmerForm.mobile" /></el-form-item>
        <el-form-item label="产地"><el-input v-model="farmerForm.origin" /></el-form-item>
        <el-form-item label="默认单价"><el-input-number v-model="farmerForm.default_unit_price" :min="0" :step="0.1" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="farmerDlg = false">取消</el-button>
        <el-button type="primary" @click="createFarmer">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; background: #f6f8fb; min-height: 100%; }
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 14px;
}
.page-title { margin: 0 0 6px; font-size: 22px; font-weight: 600; color: #1f2d3d; }
.hint { color: #667; font-size: 13px; margin: 0; line-height: 1.65; }
.head-stats { display: flex; flex-wrap: wrap; gap: 10px; }
.stat-card {
  min-width: 110px;
  padding: 10px 12px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid #e6ebf2;
  box-shadow: 0 6px 18px rgba(31, 45, 61, 0.05);
}
.stat-card[data-tone='primary'] { border-color: #d9e8ff; }
.stat-card[data-tone='warning'] { border-color: #f6dfb7; }
.stat-card[data-tone='success'] { border-color: #cfe8d7; }
.stat-card[data-tone='info'] { border-color: #dce8f5; }
.stat-label { display: block; color: #6b7785; font-size: 12px; margin-bottom: 4px; }
.stat-value { font-size: 20px; color: #1f2d3d; }
.top-panels { margin-bottom: 4px; }
.section-card { border-radius: 14px; border: 1px solid #e7edf5; }
.card-head { display: flex; flex-direction: column; gap: 2px; }
.card-head-row { flex-direction: row; align-items: center; justify-content: space-between; gap: 12px; }
.card-head span { font-size: 15px; font-weight: 600; color: #1f2d3d; }
.card-head small { color: #7a8797; font-size: 12px; }
.stack-form :deep(.el-form-item) { margin-bottom: 14px; }
.stack-form :deep(.el-input-number) { width: 100%; }
.sec { margin: 12px 0 8px; font-size: 14px; font-weight: 600; }
.label-preview { margin-top: 12px; padding: 12px; border-radius: 12px; background: #f8fbff; border: 1px dashed #d5e3f4; }
:deep(.el-card__header) { padding: 14px 18px; border-bottom: 1px solid #eef2f7; }
:deep(.el-card__body) { padding: 18px; }
:deep(.el-table) { --el-table-header-bg-color: #f8fafc; border-radius: 10px; }
:deep(.el-table .is-selected > td) { background: #ecf8f6 !important; }
@media (max-width: 900px) {
  .page-head { flex-direction: column; }
  .head-stats { width: 100%; }
  .stat-card { flex: 1; min-width: 0; }
}
</style>
