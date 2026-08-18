<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { salesApi, type FormOption } from '@erp/shared'
import {
  CustomerSelect,
  ProductSelect,
  WarehouseSelect,
  SalesOrderSelect,
  EnumSelect,
} from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import { dash, money, statusLabel, statusType, type SalesRow } from './salesUi'

type Row = SalesRow

const PRINT_DOC_OPTIONS: FormOption[] = [
  { value: 'sales_order', label: '销售订单' },
  { value: 'pre_shipment', label: '预发货' },
  { value: 'delivery', label: '发货单' },
]
const RANK_PERIODS: FormOption[] = [
  { value: 'month', label: '近一月' },
  { value: 'quarter', label: '近一季' },
  { value: 'year', label: '近一年' },
  { value: 'all', label: '全部' },
]

const props = defineProps<{ module?: string }>()
const route = useRoute()
const router = useRouter()

const MODULE_MAP: Record<string, string> = {
  销售订单: 'orders',
  修改订单: 'order-edit',
  订单复购: 'order-rebuy',
  我的订单: 'my-orders',
  询价管理: 'inquiries',
  询价审批: 'inquiry-approve',
  预发货管理: 'pre-ships',
  发货审批: 'deliveries',
  销售锁价: 'price-locks',
  合同管理: 'contracts',
  历史报价查询: 'quotes',
  数据排行榜: 'rankings',
  销售BOM: 'boms',
  成本预算: 'budgets',
  报价计算器: 'calculator',
  单据打印: 'prints',
  自助下单: 'self-orders',
  出厂结算: 'outbound',
}

const SECTION_META: Record<string, { title: string; hint: string }> = {
  orders: { title: '销售订单', hint: '客户下单 → 提交占用库存 → 预发货/发货。单价填 0 时自动取锁价或牌价。' },
  'order-edit': { title: '修改订单', hint: '仅草稿/已提交订单可改备注与明细；已发货订单只读。' },
  'order-rebuy': { title: '订单复购', hint: '按历史订单一键复制生成新单，不改原单。' },
  'my-orders': { title: '我的订单', hint: '当前登录人作为跟进人的订单，可按状态筛选并跟进。' },
  inquiries: { title: '询价管理', hint: '草稿可编辑；提交后进入询价审批/询价财务审批，通过后可转订单。' },
  'inquiry-approve': { title: '询价审批', hint: '处理待审询价。通过/驳回会同步审批队列。新建请走「询价管理」。' },
  'pre-ships': { title: '预发货管理', hint: '按订单占用成品仓；确认后生成发货审批单。可释放占用或取消。' },
  deliveries: { title: '发货审批', hint: '审批通过后出库发货，可驳回重提、登记物流并签收。' },
  'price-locks': { title: '销售锁价', hint: '客户+产品锁价，有效期内优先于牌价。可停用/重新生效。' },
  contracts: { title: '合同管理', hint: '维护合同状态、附件与关联订单，生效后可在财务合同利润中对照。' },
  quotes: { title: '历史报价查询', hint: '按客户/产品/时间钻取询价或订单报价记录。' },
  rankings: { title: '数据排行榜', hint: '按销售额排客户，可切换近一月/季/年口径。' },
  boms: { title: '销售BOM', hint: '维护成品配料行，可多行编辑、查看版本并停用。' },
  budgets: { title: '成本预算', hint: '按订单测算材料/人工/其他成本，支持重算对比毛利率。' },
  calculator: { title: '报价计算器', hint: '按成本与毛利率试算报价，可回写历史报价。' },
  prints: { title: '单据打印', hint: '生成销售订单/预发货/发货单预览。' },
  'self-orders': { title: '自助下单', hint: '维护限量/限额规则，代客户提交自助订单。' },
  outbound: { title: '出厂结算', hint: '出厂单关联订单与发货，关单后可反关单。' },
}

const active = computed(() => {
  const section = String(route.params.section || '')
  if (section) return section
  if (props.module && MODULE_MAP[props.module]) return MODULE_MAP[props.module]
  return 'orders'
})
const title = computed(() => SECTION_META[active.value]?.title || '销售管理')
const hint = computed(
  () => SECTION_META[active.value]?.hint || '客户 → 询价/锁价 → 订单占用 → 预发货 → 发货出库 → 出厂结算。',
)

const loading = ref(false)
const list = ref<Row[]>([])
const detail = ref<Row | null>(null)
const drawer = ref(false)
const createDlg = ref(false)
const editDlg = ref(false)
const bomDlg = ref(false)
const ruleDlg = ref(false)
const calcResult = ref<Row | null>(null)
const bomLines = ref<Row[]>([])
const editingBomId = ref(0)

const filter = reactive({
  keyword: '',
  status: '',
  customer_id: null as number | null,
  product_id: null as number | null,
  date_from: '',
  date_to: '',
  period: 'month',
})

const orderForm = reactive({
  customer_id: null as number | null,
  warehouse_id: 3,
  remark: '',
  product_id: null as number | null,
  qty: 100,
  price: 0,
})
const editForm = reactive({ id: 0, remark: '', product_id: 3, qty: 100, price: 0 })
const inquiryForm = reactive({
  id: 0,
  customer_id: null as number | null,
  product_id: null as number | null,
  qty: 100,
  quote_price: 0,
  remark: '',
})
const lockForm = reactive({
  customer_id: null as number | null,
  product_id: null as number | null,
  lock_price: 6.8,
  effective_from: new Date().toISOString().slice(0, 10),
  effective_to: '2026-12-31',
})
const contractForm = reactive({
  id: 0,
  customer_id: null as number | null,
  order_id: null as number | null,
  title: '年度供货合同',
  amount: 100000,
  attachment_url: '',
  remark: '',
})
const preShipForm = reactive({ order_id: null as number | null, plan_ship_date: new Date().toISOString().slice(0, 10) })
const deliveryForm = reactive({ order_id: null as number | null, logistics_no: '' })
const calcForm = reactive({ product_id: 3, qty: 100, base_cost: 4, margin_rate: 0.2 })
const bomForm = reactive({ product_id: 3, name: '袋装木薯丁BOM', material_product_id: null as number | null, qty: 1.2 })
const budgetForm = reactive({ order_id: null as number | null, material_cost: 0, labor_cost: 0, other_cost: 0 })
const printForm = reactive({ doc_type: 'sales_order', doc_id: null as number | null })
const selfForm = reactive({ customer_id: null as number | null, product_id: null as number | null, qty: 50, price: 0 })
const ruleForm = reactive({ id: 0, name: '默认自助规则', enabled: true, min_qty: 10, max_qty: 0, max_amount: 0, remark: '' })

const orderListCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'customer_name', label: '客户' },
  { prop: 'status', label: '状态' },
  { prop: 'total_amount', label: '金额' },
  { prop: 'created_at', label: '时间' },
]
const inquiryCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'customer_name', label: '客户' },
  { prop: 'status', label: '状态' },
  { prop: 'created_at', label: '时间' },
]
const preShipCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'order_no', label: '订单' },
  { prop: 'plan_ship_date', label: '计划日' },
  { prop: 'status', label: '状态' },
]
const deliveryCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'order_no', label: '订单' },
  { prop: 'status', label: '状态' },
  { prop: 'logistics_no', label: '物流' },
]
const priceLockCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'product_name', label: '产品' },
  { prop: 'lock_price', label: '锁价' },
  { prop: 'status', label: '状态' },
]
const contractCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '合同号', primary: true },
  { prop: 'customer_name', label: '客户' },
  { prop: 'title', label: '标题' },
  { prop: 'amount', label: '金额' },
  { prop: 'status', label: '状态' },
]
const quoteCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'product_name', label: '产品' },
  { prop: 'price', label: '报价' },
  { prop: 'quoted_at', label: '时间' },
]
const rankingCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'rank', label: '排名' },
  { prop: 'order_count', label: '订单数' },
  { prop: 'amount', label: '销售额' },
]
const bomCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'product_name', label: '成品' },
  { prop: 'status', label: '状态' },
]
const budgetCols: MobileCardColumn[] = [
  { prop: 'order_id', label: '订单', primary: true },
  { prop: 'sale_amount', label: '销售额' },
  { prop: 'total_cost', label: '成本' },
  { prop: 'margin', label: '毛利率' },
]
const printCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'doc_type', label: '类型' },
  { prop: 'printed_at', label: '打印时间' },
]
const selfOrderCols: MobileCardColumn[] = [
  { prop: 'name', label: '规则', primary: true },
  { prop: 'min_qty', label: '最小量' },
  { prop: 'max_qty', label: '最大量' },
  { prop: 'enabled', label: '启用' },
]

const statusOptions = computed<FormOption[]>(() => {
  const map: Record<string, string[]> = {
    orders: ['draft', 'submitted', 'shipped', 'cancelled'],
    'order-edit': ['draft', 'submitted'],
    'my-orders': ['draft', 'submitted', 'shipped', 'cancelled'],
    inquiries: ['draft', 'pending', 'approved', 'rejected', 'ordered'],
    'inquiry-approve': ['draft', 'pending'],
    'pre-ships': ['draft', 'reserved', 'confirmed', 'cancelled'],
    deliveries: ['draft', 'pending', 'approved', 'rejected', 'shipped', 'received'],
    'price-locks': ['active', 'inactive'],
    contracts: ['draft', 'active', 'expired'],
    boms: ['active', 'inactive'],
  }
  return (map[active.value] || []).map((v) => ({ value: v, label: statusLabel(v) }))
})

const orderEditList = computed(() =>
  list.value.filter((r) => ['draft', 'open', 'submitted'].includes(String(r.status))),
)
const inquiryApproveList = computed(() =>
  list.value.filter((r) => ['draft', 'pending', 'submitted'].includes(String(r.status))),
)
const myTodoList = computed(() =>
  list.value.filter((r) => ['draft', 'submitted'].includes(String(r.status))),
)

const stats = computed(() => {
  const rows = active.value === 'order-edit' ? orderEditList.value
    : active.value === 'inquiry-approve' ? inquiryApproveList.value
    : list.value
  const countBy = (st: string) => rows.filter((r) => String(r.status) === st).length
  const amount = rows.reduce((s, r) => s + Number(r.total_amount || r.amount || r.sale_amount || 0), 0)
  if (['orders', 'order-edit', 'order-rebuy', 'my-orders'].includes(active.value)) {
    return [
      { label: '单据', value: String(rows.length) },
      { label: '待提交', value: String(countBy('draft')), warn: true },
      { label: '已提交', value: String(countBy('submitted')) },
      { label: '金额', value: money(amount), ok: true },
    ]
  }
  if (active.value === 'inquiries' || active.value === 'inquiry-approve') {
    return [
      { label: '询价单', value: String(rows.length) },
      { label: '待审', value: String(countBy('pending') + countBy('draft')), warn: true },
      { label: '已通过', value: String(countBy('approved')), ok: true },
      { label: '已驳回', value: String(countBy('rejected')) },
    ]
  }
  if (active.value === 'pre-ships') {
    return [
      { label: '预发货', value: String(rows.length) },
      { label: '已占用', value: String(rows.filter((r) => r.reserved).length), warn: true },
      { label: '已确认', value: String(countBy('confirmed')), ok: true },
    ]
  }
  if (active.value === 'deliveries') {
    return [
      { label: '发货单', value: String(rows.length) },
      { label: '待审', value: String(countBy('pending') + countBy('draft')), warn: true },
      { label: '已发货', value: String(countBy('shipped')), ok: true },
      { label: '已签收', value: String(countBy('received')) },
    ]
  }
  return [
    { label: '记录', value: String(rows.length) },
    { label: '金额/销售额', value: money(amount), ok: true },
  ]
})

function qs() {
  const p = new URLSearchParams()
  if (filter.keyword) p.set('keyword', filter.keyword)
  if (filter.status) p.set('status', filter.status)
  if (filter.customer_id) p.set('customer_id', String(filter.customer_id))
  if (filter.product_id) p.set('product_id', String(filter.product_id))
  if (filter.date_from) p.set('date_from', filter.date_from)
  if (filter.date_to) p.set('date_to', filter.date_to)
  if (active.value === 'rankings') p.set('period', filter.period)
  const s = p.toString()
  return s
}

function showCreate() {
  return [
    'orders', 'inquiries', 'pre-ships', 'deliveries', 'price-locks', 'contracts',
    'boms', 'budgets', 'self-orders', 'prints',
  ].includes(active.value)
}

function openCreate() {
  if (active.value === 'inquiries') inquiryForm.id = 0
  if (active.value === 'contracts') contractForm.id = 0
  createDlg.value = true
}

async function refresh() {
  loading.value = true
  try {
    const q = qs()
    let res
    switch (active.value) {
      case 'orders':
      case 'order-edit':
      case 'order-rebuy':
        res = await salesApi.orders(q)
        break
      case 'my-orders':
        res = await salesApi.myOrders(q)
        break
      case 'inquiries':
      case 'inquiry-approve':
        res = await salesApi.inquiries(q)
        break
      case 'pre-ships':
        res = await salesApi.preShipments(q)
        break
      case 'deliveries':
        res = await salesApi.deliveries(q)
        break
      case 'price-locks':
        res = await salesApi.priceLocks(q)
        break
      case 'contracts':
        res = await salesApi.contracts(q)
        break
      case 'quotes':
        res = await salesApi.quoteHistories(q)
        break
      case 'rankings':
        res = await salesApi.rankings(q)
        list.value = ((res.data as { list?: Row[] })?.list) || []
        return
      case 'boms':
        res = await salesApi.salesBoms()
        break
      case 'budgets':
        res = await salesApi.costBudgets()
        break
      case 'prints':
        res = await salesApi.prints()
        break
      case 'self-orders':
        res = await salesApi.selfOrders()
        list.value = ((res.data as { rules?: Row[] })?.rules) || []
        return
      case 'calculator':
        res = await salesApi.quoteCalculator()
        list.value = []
        detail.value = (res.data as Row) || null
        return
      case 'outbound':
        router.replace('/sales/outbound-settle')
        return
      default:
        res = await salesApi.orders(q)
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function ok(res: { code?: number; msg?: string }, msg: string) {
  if (res.code !== 1) {
    ElMessage.error(res.msg || '操作失败')
    return false
  }
  ElMessage.success(msg)
  return true
}

async function openDetail(kind: string, id: number) {
  let res
  if (kind === 'order') res = await salesApi.getOrder(id)
  else if (kind === 'inquiry') res = await salesApi.getInquiry(id)
  else if (kind === 'preship') res = await salesApi.getPreShip(id)
  else if (kind === 'delivery') res = await salesApi.getDelivery(id)
  else if (kind === 'contract') res = await salesApi.getContract(id)
  else if (kind === 'bom') res = await salesApi.getSalesBom(id)
  else if (kind === 'settle') res = await salesApi.getOutboundSettle(id)
  else return
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
  drawer.value = true
}

function linePayload(productId: number, qty: number, price: number) {
  return [{ product_id: productId, qty, price }]
}

async function createOrder() {
  if (!orderForm.customer_id || !orderForm.product_id) return ElMessage.warning('请选择客户和产品')
  const res = await salesApi.createOrder({
    customer_id: orderForm.customer_id,
    warehouse_id: orderForm.warehouse_id,
    remark: orderForm.remark,
    lines: linePayload(orderForm.product_id, orderForm.qty, orderForm.price),
  })
  if (!ok(res, `订单已创建 ${(res.data as Row)?.doc_no}`)) return
  preShipForm.order_id = Number((res.data as Row)?.id)
  deliveryForm.order_id = Number((res.data as Row)?.id)
  budgetForm.order_id = Number((res.data as Row)?.id)
  createDlg.value = false
  await refresh()
}

async function submitOrder(id: number) {
  const res = await salesApi.submitOrder(id)
  if (ok(res, '已提交')) await refresh()
}
async function cancelOrder(id: number) {
  await ElMessageBox.confirm('确认取消订单并释放库存占用？')
  const res = await salesApi.cancelOrder(id)
  if (ok(res, '已取消')) await refresh()
}
async function rebuyOrder(id: number) {
  const res = await salesApi.rebuyOrder(id)
  if (ok(res, `复购成功 ${(res.data as Row)?.doc_no}`)) await refresh()
}

async function openEditOrder(row: Row) {
  editForm.id = Number(row.id)
  editForm.remark = String(row.remark || '')
  const res = await salesApi.getOrder(Number(row.id))
  if (res.code === 1) {
    const d = res.data as Row
    editForm.remark = String(d.remark || '')
    const dl = (d.lines as Row[]) || []
    if (dl[0]) {
      editForm.product_id = Number(dl[0].product_id || 3)
      editForm.qty = Number(dl[0].qty || 100)
      editForm.price = Number(dl[0].price || 0)
    }
  }
  editDlg.value = true
}
async function saveEditOrder() {
  const res = await salesApi.updateOrder(editForm.id, {
    remark: editForm.remark,
    lines: linePayload(editForm.product_id, editForm.qty, editForm.price),
  })
  if (!ok(res, '订单已修改')) return
  editDlg.value = false
  await refresh()
}

async function createInquiry() {
  if (!inquiryForm.customer_id || !inquiryForm.product_id) return ElMessage.warning('请选择客户和产品')
  const payload = {
    customer_id: inquiryForm.customer_id,
    remark: inquiryForm.remark,
    lines: [{ product_id: inquiryForm.product_id, qty: inquiryForm.qty, quote_price: inquiryForm.quote_price }],
  }
  const res = inquiryForm.id
    ? await salesApi.updateInquiry(inquiryForm.id, payload)
    : await salesApi.createInquiry(payload)
  if (!ok(res, inquiryForm.id ? '询价已保存' : `询价单 ${(res.data as Row)?.doc_no}`)) return
  createDlg.value = false
  inquiryForm.id = 0
  await refresh()
}
async function openEditInquiry(row: Row) {
  const res = await salesApi.getInquiry(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = res.data as Row
  const ln = ((d.lines as Row[]) || [])[0] || {}
  inquiryForm.id = Number(d.id)
  inquiryForm.customer_id = Number(d.customer_id)
  inquiryForm.product_id = Number(ln.product_id || 0)
  inquiryForm.qty = Number(ln.qty || 100)
  inquiryForm.quote_price = Number(ln.quote_price || 0)
  inquiryForm.remark = String(d.remark || '')
  createDlg.value = true
}
async function submitInquiry(id: number) {
  const res = await salesApi.submitInquiry(id)
  if (ok(res, '已提交审批')) await refresh()
}
async function approveInquiry(id: number) {
  const res = await salesApi.approveInquiry(id)
  if (ok(res, '询价已通过')) await refresh()
}
async function rejectInquiry(id: number) {
  const { value } = await ElMessageBox.prompt('请填写驳回原因', '驳回询价', { inputPlaceholder: '原因' })
  const res = await salesApi.rejectInquiry(id, { comment: value })
  if (ok(res, '已驳回')) await refresh()
}
async function withdrawInquiry(id: number) {
  const res = await salesApi.withdrawInquiry(id)
  if (ok(res, '已撤回')) await refresh()
}
async function inquiryToOrder(id: number) {
  const res = await salesApi.inquiryToOrder(id)
  if (ok(res, `已转订单 ${((res.data as Row)?.order as Row)?.doc_no}`)) await refresh()
}

async function createLock() {
  if (!lockForm.customer_id || !lockForm.product_id) return ElMessage.warning('请选择客户和产品')
  const res = await salesApi.createPriceLock({ ...lockForm })
  if (!ok(res, '锁价已生效')) return
  createDlg.value = false
  await refresh()
}
async function toggleLock(row: Row) {
  const id = Number(row.id)
  const res = String(row.status) === 'active'
    ? await salesApi.deactivatePriceLock(id)
    : await salesApi.activatePriceLock(id)
  if (ok(res, String(row.status) === 'active' ? '已停用' : '已生效')) await refresh()
}

async function createContract() {
  if (!contractForm.customer_id) return ElMessage.warning('请选择客户')
  const payload = { ...contractForm }
  const res = contractForm.id
    ? await salesApi.updateContract(contractForm.id, payload)
    : await salesApi.createContract(payload)
  if (!ok(res, contractForm.id ? '合同已保存' : `合同 ${(res.data as Row)?.doc_no}`)) return
  createDlg.value = false
  contractForm.id = 0
  await refresh()
}
async function openEditContract(row: Row) {
  const res = await salesApi.getContract(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = res.data as Row
  contractForm.id = Number(d.id)
  contractForm.customer_id = Number(d.customer_id)
  contractForm.order_id = d.order_id ? Number(d.order_id) : null
  contractForm.title = String(d.title || '')
  contractForm.amount = Number(d.amount || 0)
  contractForm.attachment_url = String(d.attachment_url || '')
  contractForm.remark = String(d.remark || '')
  createDlg.value = true
}
async function activateContract(id: number) {
  const res = await salesApi.activateContract(id)
  if (ok(res, '合同已生效')) await refresh()
}

async function createPreShip() {
  if (!preShipForm.order_id) return ElMessage.warning('请选择订单')
  const res = await salesApi.createPreShip({ ...preShipForm })
  if (!ok(res, `预发货 ${(res.data as Row)?.doc_no}`)) return
  createDlg.value = false
  await refresh()
}
async function reservePre(id: number) {
  const res = await salesApi.reservePreShip(id)
  if (ok(res, '已占用库存')) await refresh()
}
async function releasePre(id: number) {
  const res = await salesApi.releasePreShip(id)
  if (ok(res, '已释放占用')) await refresh()
}
async function cancelPre(id: number) {
  await ElMessageBox.confirm('确认取消预发货并释放占用？')
  const res = await salesApi.cancelPreShip(id)
  if (ok(res, '已取消')) await refresh()
}
async function confirmPre(id: number) {
  const res = await salesApi.confirmPreShip(id)
  if (ok(res, '已确认并生成发货单')) await refresh()
}

async function createDelivery() {
  if (!deliveryForm.order_id) return ElMessage.warning('请选择订单')
  const res = await salesApi.createDelivery({ order_id: deliveryForm.order_id })
  if (!ok(res, `发货单 ${(res.data as Row)?.doc_no}`)) return
  createDlg.value = false
  await refresh()
}
async function approveDelivery(id: number) {
  const res = await salesApi.approveDelivery(id)
  if (ok(res, '发货已审批')) await refresh()
}
async function rejectDelivery(id: number) {
  const { value } = await ElMessageBox.prompt('请填写驳回原因', '驳回发货', { inputPlaceholder: '原因' })
  const res = await salesApi.rejectDelivery(id, { comment: value })
  if (ok(res, '已驳回')) await refresh()
}
async function resubmitDelivery(id: number) {
  const res = await salesApi.resubmitDelivery(id)
  if (ok(res, '已重提')) await refresh()
}
async function shipDelivery(id: number) {
  const res = await salesApi.shipDelivery(id, { logistics_no: deliveryForm.logistics_no })
  if (ok(res, '已出库发货')) await refresh()
}
async function receiveDelivery(id: number) {
  const { value } = await ElMessageBox.prompt('签收备注（可空）', '客户签收', { inputPlaceholder: '备注', inputValue: '' })
  const res = await salesApi.receiveDelivery(id, { receive_remark: value })
  if (ok(res, '已签收')) await refresh()
}

async function doCalc() {
  const res = await salesApi.calcQuote({ ...calcForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  calcResult.value = (res.data as Row) || null
}
async function applyCalc() {
  if (!calcResult.value) return
  const res = await salesApi.applyQuote({
    customer_id: orderForm.customer_id,
    ...calcForm,
    quote_price: calcResult.value.quote_price,
  })
  if (ok(res, '已写入历史报价')) await refresh()
}

async function createBom() {
  if (!bomForm.material_product_id) return ElMessage.warning('请选择原料')
  const res = await salesApi.createSalesBom({ product_id: bomForm.product_id, name: bomForm.name })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const id = Number((res.data as Row)?.id)
  await salesApi.saveSalesBomLines(id, {
    lines: [{ material_product_id: bomForm.material_product_id, qty: bomForm.qty }],
  })
  ElMessage.success('销售BOM已保存')
  createDlg.value = false
  await refresh()
}
async function openBom(row: Row) {
  const res = await salesApi.getSalesBom(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = res.data as Row
  editingBomId.value = Number(d.id)
  bomLines.value = ((d.lines as Row[]) || []).map((x) => ({ ...x }))
  if (!bomLines.value.length) bomLines.value = [{ material_product_id: null, qty: 1, scrap_rate: 0 }]
  bomDlg.value = true
}
function addBomLine() {
  bomLines.value.push({ material_product_id: null, qty: 1, scrap_rate: 0 })
}
async function saveBomLines() {
  const res = await salesApi.saveSalesBomLines(editingBomId.value, { lines: bomLines.value })
  if (ok(res, '明细已保存')) {
    bomDlg.value = false
    await refresh()
  }
}
async function deactivateBom(id: number) {
  const res = await salesApi.deactivateSalesBom(id)
  if (ok(res, '已停用')) await refresh()
}

async function createBudget() {
  if (!budgetForm.order_id) return ElMessage.warning('请选择订单')
  const res = await salesApi.createCostBudget({ ...budgetForm })
  if (!ok(res, `毛利率 ${((Number((res.data as Row)?.margin) || 0) * 100).toFixed(1)}%`)) return
  createDlg.value = false
  await refresh()
}
async function recalcBudget(row: Row) {
  const res = await salesApi.recalcCostBudget(Number(row.id), {
    material_cost: budgetForm.material_cost || row.material_cost,
    labor_cost: budgetForm.labor_cost || row.labor_cost,
    other_cost: budgetForm.other_cost || row.other_cost,
  })
  if (ok(res, `已重算 毛利率 ${((Number((res.data as Row)?.margin) || 0) * 100).toFixed(1)}%`)) await refresh()
}

async function doPrint() {
  if (!printForm.doc_id) return ElMessage.warning('请选择单据')
  const res = await salesApi.createPrint({ ...printForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = ((res.data as Row)?.preview as Row) || (res.data as Row)
  drawer.value = true
  ElMessage.success('打印预览已生成')
  await refresh()
}

async function submitSelf() {
  if (!selfForm.customer_id || !selfForm.product_id) return ElMessage.warning('请选择客户和产品')
  const res = await salesApi.submitSelfOrder({
    customer_id: selfForm.customer_id,
    source: 'self',
    lines: linePayload(selfForm.product_id, selfForm.qty, selfForm.price),
  })
  if (ok(res, `自助订单 ${(res.data as Row)?.doc_no}`)) {
    createDlg.value = false
    await refresh()
  }
}
function openRule(row?: Row) {
  if (row) {
    ruleForm.id = Number(row.id)
    ruleForm.name = String(row.name || '')
    ruleForm.enabled = Boolean(row.enabled)
    ruleForm.min_qty = Number(row.min_qty || 0)
    ruleForm.max_qty = Number(row.max_qty || 0)
    ruleForm.max_amount = Number(row.max_amount || 0)
    ruleForm.remark = String(row.remark || '')
  } else {
    ruleForm.id = 0
    ruleForm.name = '默认自助规则'
    ruleForm.enabled = true
    ruleForm.min_qty = 10
    ruleForm.max_qty = 0
    ruleForm.max_amount = 0
    ruleForm.remark = ''
  }
  ruleDlg.value = true
}
async function saveRule() {
  const res = await salesApi.saveSelfOrderRule({ ...ruleForm }, ruleForm.id || undefined)
  if (ok(res, '规则已保存')) {
    ruleDlg.value = false
    await refresh()
  }
}

function goFinance(path: string) {
  router.push(path)
}
function goQuoteDrill(row: Row) {
  if (row.inquiry_id) openDetail('inquiry', Number(row.inquiry_id))
  else if (row.order_id) openDetail('order', Number(row.order_id))
}
function goRankCustomer(row: Row) {
  filter.customer_id = Number(row.customer_id)
  router.push('/sales/hub/orders')
}

watch(active, () => {
  detail.value = null
  drawer.value = false
  filter.keyword = ''
  filter.status = ''
  refresh()
})
watch(() => [filter.status, filter.customer_id, filter.product_id, filter.date_from, filter.date_to, filter.period], () => {
  refresh()
})

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <header class="page-head">
      <div>
        <h2 class="title">{{ title }}</h2>
        <p class="desc">{{ hint }}</p>
      </div>
      <div class="head-meta">
        <span class="meta-pill">成品默认仓 · 成品冷库</span>
      </div>
    </header>

    <div class="stats">
      <div v-for="s in stats" :key="s.label" class="stat" :class="{ ok: s.ok, warn: s.warn }">
        <div class="label">{{ s.label }}</div>
        <div class="value">{{ s.value }}</div>
      </div>
    </div>

    <div class="toolbar">
      <el-input v-model="filter.keyword" clearable placeholder="关键词 / 单号" style="width:180px" @keyup.enter="refresh" @clear="refresh" />
      <el-select v-if="statusOptions.length" v-model="filter.status" clearable placeholder="状态" style="width:130px">
        <el-option v-for="o in statusOptions" :key="o.value" :label="o.label" :value="o.value" />
      </el-select>
      <CustomerSelect v-if="['orders','inquiries','quotes','price-locks','contracts','my-orders'].includes(active)" v-model="filter.customer_id" />
      <ProductSelect v-if="active==='quotes' || active==='price-locks'" v-model="filter.product_id" />
      <el-date-picker v-if="active==='quotes'" v-model="filter.date_from" type="date" value-format="YYYY-MM-DD" placeholder="起" style="width:140px" />
      <el-date-picker v-if="active==='quotes'" v-model="filter.date_to" type="date" value-format="YYYY-MM-DD" placeholder="止" style="width:140px" />
      <EnumSelect v-if="active==='rankings'" v-model="filter.period" :options="RANK_PERIODS" :clearable="false" style="width:120px" />
      <el-button @click="refresh">刷新</el-button>
      <el-button v-if="showCreate()" type="primary" @click="openCreate">新建</el-button>
      <el-button v-if="active==='self-orders'" @click="openRule()">规则</el-button>
      <el-button v-if="active==='orders'" link type="primary" @click="goFinance('/finance/hub/writeoffs')">收款核单</el-button>
      <el-button v-if="active==='contracts'" link type="primary" @click="goFinance('/finance/hub/contract-profits')">合同利润</el-button>
      <el-button v-if="active==='budgets'" link type="primary" @click="goFinance('/report/hub/gross-profit')">毛利润统计</el-button>
      <el-button v-if="active==='inquiry-approve'" link @click="router.push('/approval/hub/inquiry-finance')">询价财务审批</el-button>
    </div>

    <p v-if="active==='order-edit' || active==='order-rebuy' || active==='inquiry-approve' || active==='my-orders'" class="mode-hint">
      {{ hint }}
    </p>

    <!-- 订单类 -->
    <template v-if="['orders','order-edit','order-rebuy','my-orders'].includes(active)">
      <TableOrCards :data="active==='order-edit' ? orderEditList : list" :loading="loading" :columns="orderListCols">
        <el-table :data="active==='order-edit' ? orderEditList : list" size="small" @row-click="(r: Row) => openDetail('order', Number(r.id))">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="customer_name" label="客户" width="140" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="90" />
          <el-table-column label="金额" width="100"><template #default="{ row }">{{ money(row.total_amount) }}</template></el-table-column>
          <el-table-column prop="created_at" label="时间" min-width="150" />
          <el-table-column label="操作" width="280" fixed="right">
            <template #default="{ row }">
              <el-button v-if="active==='orders' && row.status==='draft'" link type="primary" @click.stop="submitOrder(Number(row.id))">提交</el-button>
              <el-button v-if="active!=='order-rebuy'" link @click.stop="openEditOrder(row)">修改</el-button>
              <el-button v-if="active==='orders'" link type="danger" @click.stop="cancelOrder(Number(row.id))">取消</el-button>
              <el-button v-if="active==='order-rebuy'" link type="success" @click.stop="rebuyOrder(Number(row.id))">一键复购</el-button>
              <el-button link @click.stop="openDetail('order', Number(row.id))">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link @click="openDetail('order', Number(row.id))">详情</el-button>
        </template>
      </TableOrCards>
      <el-alert v-if="active==='my-orders'" type="info" :closable="false" class="mt" :title="`待跟进 ${myTodoList.length} 单（草稿/已提交）`" />
    </template>

    <!-- 询价 -->
    <template v-else-if="active==='inquiries' || active==='inquiry-approve'">
      <TableOrCards :data="active==='inquiry-approve' ? inquiryApproveList : list" :loading="loading" :columns="inquiryCols">
        <el-table :data="active==='inquiry-approve' ? inquiryApproveList : list" size="small">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="customer_name" label="客户" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="submitted_at" label="提交" width="160" />
          <el-table-column prop="created_at" label="创建" width="160" />
          <el-table-column label="操作" width="280">
            <template #default="{ row }">
              <el-button v-if="active==='inquiries' && ['draft','rejected'].includes(String(row.status))" link @click="openEditInquiry(row)">编辑</el-button>
              <el-button v-if="active==='inquiries' && ['draft','rejected'].includes(String(row.status))" link type="primary" @click="submitInquiry(Number(row.id))">提交</el-button>
              <el-button v-if="row.status==='pending'" link @click="withdrawInquiry(Number(row.id))">撤回</el-button>
              <el-button v-if="['draft','pending'].includes(String(row.status))" link type="success" @click="approveInquiry(Number(row.id))">通过</el-button>
              <el-button v-if="['draft','pending'].includes(String(row.status))" link type="danger" @click="rejectInquiry(Number(row.id))">驳回</el-button>
              <el-button v-if="row.status==='approved'" link type="primary" @click="inquiryToOrder(Number(row.id))">转订单</el-button>
              <el-button link @click="openDetail('inquiry', Number(row.id))">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <!-- 预发货 -->
    <template v-else-if="active==='pre-ships'">
      <TableOrCards :data="list" :loading="loading" :columns="preShipCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="order_no" label="订单" width="160" />
          <el-table-column prop="plan_ship_date" label="计划日" width="120" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="占用" width="80"><template #default="{ row }">{{ row.reserved ? '是' : '否' }}</template></el-table-column>
          <el-table-column label="操作" width="280">
            <template #default="{ row }">
              <el-button v-if="!row.reserved && row.status!=='cancelled' && row.status!=='confirmed'" link @click="reservePre(Number(row.id))">占用</el-button>
              <el-button v-if="row.reserved" link @click="releasePre(Number(row.id))">释放</el-button>
              <el-button v-if="row.status!=='confirmed' && row.status!=='cancelled'" link type="primary" @click="confirmPre(Number(row.id))">确认→发货</el-button>
              <el-button v-if="row.status!=='confirmed' && row.status!=='cancelled'" link type="danger" @click="cancelPre(Number(row.id))">取消</el-button>
              <el-button link @click="openDetail('preship', Number(row.id))">明细</el-button>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <!-- 发货 -->
    <template v-else-if="active==='deliveries'">
      <TableOrCards :data="list" :loading="loading" :columns="deliveryCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="order_no" label="订单" width="160" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="logistics_no" label="物流" />
          <el-table-column prop="shipped_at" label="发货" width="160" />
          <el-table-column prop="received_at" label="签收" width="160" />
          <el-table-column label="操作" width="300">
            <template #default="{ row }">
              <el-button v-if="['draft','pending'].includes(String(row.status))" link type="success" @click="approveDelivery(Number(row.id))">通过</el-button>
              <el-button v-if="['draft','pending','approved'].includes(String(row.status))" link type="danger" @click="rejectDelivery(Number(row.id))">驳回</el-button>
              <el-button v-if="row.status==='rejected'" link @click="resubmitDelivery(Number(row.id))">重提</el-button>
              <el-button v-if="['approved','pending','draft'].includes(String(row.status))" link type="primary" @click="shipDelivery(Number(row.id))">出库发货</el-button>
              <el-button v-if="row.status==='shipped'" link type="success" @click="receiveDelivery(Number(row.id))">签收</el-button>
              <el-button link @click="openDetail('delivery', Number(row.id))">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <!-- 锁价 -->
    <template v-else-if="active==='price-locks'">
      <TableOrCards :data="list" :loading="loading" :columns="priceLockCols">
        <el-table :data="list" size="small">
          <el-table-column prop="customer_name" label="客户" />
          <el-table-column prop="product_name" label="产品" />
          <el-table-column prop="lock_price" label="锁价" width="100" />
          <el-table-column prop="effective_from" label="起" width="120" />
          <el-table-column prop="effective_to" label="止" width="120" />
          <el-table-column prop="version_no" label="版本" width="70" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link @click="toggleLock(row)">{{ row.status==='active' ? '停用' : '生效' }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <!-- 合同 -->
    <template v-else-if="active==='contracts'">
      <TableOrCards :data="list" :loading="loading" :columns="contractCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="合同号" width="160" />
          <el-table-column prop="customer_name" label="客户" />
          <el-table-column prop="title" label="标题" />
          <el-table-column label="金额" width="120"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button link @click="openEditContract(row)">编辑</el-button>
              <el-button v-if="row.status==='draft'" link type="success" @click="activateContract(Number(row.id))">生效</el-button>
              <el-button link @click="openDetail('contract', Number(row.id))">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active==='quotes'">
      <TableOrCards :data="list" :loading="loading" :columns="quoteCols">
        <el-table :data="list" size="small">
          <el-table-column prop="customer_name" label="客户" />
          <el-table-column prop="product_name" label="产品" />
          <el-table-column prop="price" label="报价" width="100" />
          <el-table-column prop="quoted_at" label="时间" width="180" />
          <el-table-column label="来源" width="160">
            <template #default="{ row }">
              <el-button v-if="row.inquiry_id" link type="primary" @click="goQuoteDrill(row)">询价#{{ row.inquiry_id }}</el-button>
              <el-button v-else-if="row.order_id" link type="primary" @click="goQuoteDrill(row)">订单#{{ row.order_id }}</el-button>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active==='rankings'">
      <TableOrCards :data="list" :loading="loading" :columns="rankingCols">
        <el-table :data="list" size="small">
          <el-table-column prop="rank" label="排名" width="80" />
          <el-table-column prop="customer_name" label="客户" />
          <el-table-column prop="order_count" label="订单数" width="100" />
          <el-table-column label="销售额" width="120"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }"><el-button link type="primary" @click="goRankCustomer(row)">看订单</el-button></template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active==='boms'">
      <TableOrCards :data="list" :loading="loading" :columns="bomCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="product_name" label="成品" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button link @click="openBom(row)">编辑明细</el-button>
              <el-button v-if="row.status==='active'" link type="danger" @click="deactivateBom(Number(row.id))">停用</el-button>
              <el-button link @click="openDetail('bom', Number(row.id))">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active==='budgets'">
      <TableOrCards :data="list" :loading="loading" :columns="budgetCols">
        <el-table :data="list" size="small">
          <el-table-column prop="order_id" label="订单" width="90" />
          <el-table-column label="销售额" width="100"><template #default="{ row }">{{ money(row.sale_amount) }}</template></el-table-column>
          <el-table-column label="成本" width="100"><template #default="{ row }">{{ money(row.total_cost) }}</template></el-table-column>
          <el-table-column label="毛利率" width="100"><template #default="{ row }">{{ ((Number(row.margin)||0)*100).toFixed(1) }}%</template></el-table-column>
          <el-table-column prop="created_at" label="时间" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }"><el-button link @click="recalcBudget(row)">重算</el-button></template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active==='calculator'">
      <el-card header="报价试算">
        <el-form inline size="small">
          <el-form-item label="产品"><ProductSelect v-model="calcForm.product_id" /></el-form-item>
          <el-form-item label="数量"><el-input-number v-model="calcForm.qty" :min="1" /></el-form-item>
          <el-form-item label="成本"><el-input-number v-model="calcForm.base_cost" :min="0" :step="0.1" /></el-form-item>
          <el-form-item label="毛利率"><el-input-number v-model="calcForm.margin_rate" :min="0" :max="1" :step="0.01" /></el-form-item>
          <el-button type="primary" @click="doCalc">试算</el-button>
          <el-button @click="applyCalc" :disabled="!calcResult">回写报价</el-button>
        </el-form>
        <el-descriptions v-if="calcResult" :column="3" border style="margin-top:12px">
          <el-descriptions-item label="单价">{{ calcResult.quote_price }}</el-descriptions-item>
          <el-descriptions-item label="金额">{{ calcResult.amount }}</el-descriptions-item>
          <el-descriptions-item label="成本">{{ calcResult.base_cost }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </template>

    <template v-else-if="active==='prints'">
      <TableOrCards :data="list" :loading="loading" :columns="printCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_type" label="类型" width="120" />
          <el-table-column prop="doc_no" label="单号" />
          <el-table-column prop="printed_at" label="打印时间" width="180" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active==='self-orders'">
      <TableOrCards :data="list" :loading="loading" :columns="selfOrderCols">
        <el-table :data="list" size="small">
          <el-table-column prop="name" label="规则" />
          <el-table-column prop="min_qty" label="最小量" width="90" />
          <el-table-column prop="max_qty" label="最大量" width="90" />
          <el-table-column prop="max_amount" label="限额" width="90" />
          <el-table-column label="启用" width="80"><template #default="{ row }">{{ row.enabled ? '是' : '否' }}</template></el-table-column>
          <el-table-column prop="remark" label="说明" />
          <el-table-column label="操作" width="80">
            <template #default="{ row }"><el-button link @click="openRule(row)">编辑</el-button></template>
          </el-table-column>
        </el-table>
      </TableOrCards>
    </template>

    <!-- 详情抽屉 -->
    <el-drawer v-model="drawer" title="单据详情" size="520px">
      <template v-if="detail">
        <el-alert v-if="detail.approval_chain" type="info" :closable="false" :title="String(detail.approval_chain)" class="mb" />
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item v-for="k in ['doc_no','customer_name','status','source','total_amount','order_no','logistics_no','plan_ship_date','shipped_at','received_at','reject_reason','remark','created_at','submitted_at','approved_at']" :key="k" :label="k">
            <el-tag v-if="k==='status'" size="small" :type="statusType(detail[k])">{{ statusLabel(detail[k]) }}</el-tag>
            <span v-else>{{ dash(detail[k]) }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <h4 v-if="Array.isArray(detail.lines) && (detail.lines as Row[]).length" class="sub">明细</h4>
        <el-table v-if="Array.isArray(detail.lines)" :data="detail.lines as Row[]" size="small">
          <el-table-column prop="product_name" label="产品" />
          <el-table-column prop="qty" label="数量" width="80" />
          <el-table-column prop="quote_price" label="报价" width="80" />
          <el-table-column prop="price" label="单价" width="80" />
          <el-table-column prop="amount" label="金额" width="80" />
        </el-table>
        <h4 v-if="Array.isArray(detail.approvals) && (detail.approvals as Row[]).length" class="sub">审批记录</h4>
        <el-timeline v-if="Array.isArray(detail.approvals)">
          <el-timeline-item v-for="a in (detail.approvals as Row[])" :key="String(a.id)" :timestamp="String(a.acted_at || a.created_at || '')">
            {{ a.title || a.category }} · {{ statusLabel(a.status) }}
            <div v-if="a.comment" class="muted">{{ a.comment }}</div>
          </el-timeline-item>
        </el-timeline>
        <h4 v-if="Array.isArray(detail.related_orders) && (detail.related_orders as Row[]).length" class="sub">关联订单</h4>
        <el-table v-if="Array.isArray(detail.related_orders)" :data="detail.related_orders as Row[]" size="small">
          <el-table-column prop="doc_no" label="单号" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="total_amount" label="金额" width="90" />
        </el-table>
        <div v-if="(detail.finance_links as Row)" class="mt">
          <el-button link type="primary" @click="goFinance('/finance/hub/writeoffs')">收款核单</el-button>
          <el-button link type="primary" @click="goFinance('/finance/hub/contract-profits')">合同利润</el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 新建弹窗 -->
    <el-dialog v-model="createDlg" :title="'新建' + title" width="560px" destroy-on-close>
      <el-form v-if="active==='orders'" label-width="90px">
        <el-form-item label="客户"><CustomerSelect v-model="orderForm.customer_id" /></el-form-item>
        <el-form-item label="仓库"><WarehouseSelect v-model="orderForm.warehouse_id" :clearable="false" /></el-form-item>
        <el-form-item label="产品"><ProductSelect v-model="orderForm.product_id" /></el-form-item>
        <el-form-item label="数量"><el-input-number v-model="orderForm.qty" :min="1" /></el-form-item>
        <el-form-item label="单价"><el-input-number v-model="orderForm.price" :min="0" :step="0.1" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="orderForm.remark" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='inquiries'" label-width="90px">
        <el-form-item label="客户"><CustomerSelect v-model="inquiryForm.customer_id" /></el-form-item>
        <el-form-item label="产品"><ProductSelect v-model="inquiryForm.product_id" /></el-form-item>
        <el-form-item label="数量"><el-input-number v-model="inquiryForm.qty" :min="1" /></el-form-item>
        <el-form-item label="报价"><el-input-number v-model="inquiryForm.quote_price" :min="0" :step="0.1" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="inquiryForm.remark" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='pre-ships'" label-width="100px">
        <el-form-item label="订单"><SalesOrderSelect v-model="preShipForm.order_id" /></el-form-item>
        <el-form-item label="计划发货日"><el-date-picker v-model="preShipForm.plan_ship_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='deliveries'" label-width="90px">
        <el-form-item label="订单"><SalesOrderSelect v-model="deliveryForm.order_id" /></el-form-item>
        <el-form-item label="物流单号"><el-input v-model="deliveryForm.logistics_no" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='price-locks'" label-width="90px">
        <el-form-item label="客户"><CustomerSelect v-model="lockForm.customer_id" /></el-form-item>
        <el-form-item label="产品"><ProductSelect v-model="lockForm.product_id" /></el-form-item>
        <el-form-item label="锁价"><el-input-number v-model="lockForm.lock_price" :min="0" :step="0.1" /></el-form-item>
        <el-form-item label="起"><el-date-picker v-model="lockForm.effective_from" type="date" value-format="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="止"><el-date-picker v-model="lockForm.effective_to" type="date" value-format="YYYY-MM-DD" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='contracts'" label-width="90px">
        <el-form-item label="客户"><CustomerSelect v-model="contractForm.customer_id" /></el-form-item>
        <el-form-item label="关联订单"><SalesOrderSelect v-model="contractForm.order_id" /></el-form-item>
        <el-form-item label="标题"><el-input v-model="contractForm.title" /></el-form-item>
        <el-form-item label="金额"><el-input-number v-model="contractForm.amount" :min="0" /></el-form-item>
        <el-form-item label="附件URL"><el-input v-model="contractForm.attachment_url" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="contractForm.remark" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='boms'" label-width="90px">
        <el-form-item label="成品"><ProductSelect v-model="bomForm.product_id" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="bomForm.name" /></el-form-item>
        <el-form-item label="原料"><ProductSelect v-model="bomForm.material_product_id" /></el-form-item>
        <el-form-item label="用量"><el-input-number v-model="bomForm.qty" :min="0" :step="0.1" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='budgets'" label-width="90px">
        <el-form-item label="订单"><SalesOrderSelect v-model="budgetForm.order_id" /></el-form-item>
        <el-form-item label="材料"><el-input-number v-model="budgetForm.material_cost" :min="0" /></el-form-item>
        <el-form-item label="人工"><el-input-number v-model="budgetForm.labor_cost" :min="0" /></el-form-item>
        <el-form-item label="其他"><el-input-number v-model="budgetForm.other_cost" :min="0" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='prints'" label-width="90px">
        <el-form-item label="类型"><EnumSelect v-model="printForm.doc_type" :options="PRINT_DOC_OPTIONS" :clearable="false" /></el-form-item>
        <el-form-item label="单据"><SalesOrderSelect v-model="printForm.doc_id" /></el-form-item>
      </el-form>
      <el-form v-else-if="active==='self-orders'" label-width="90px">
        <el-form-item label="客户"><CustomerSelect v-model="selfForm.customer_id" /></el-form-item>
        <el-form-item label="产品"><ProductSelect v-model="selfForm.product_id" /></el-form-item>
        <el-form-item label="数量"><el-input-number v-model="selfForm.qty" :min="1" /></el-form-item>
        <el-form-item label="单价"><el-input-number v-model="selfForm.price" :min="0" :step="0.1" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDlg = false">取消</el-button>
        <el-button v-if="active==='orders'" type="primary" @click="createOrder">下单</el-button>
        <el-button v-else-if="active==='inquiries'" type="primary" @click="createInquiry">保存</el-button>
        <el-button v-else-if="active==='pre-ships'" type="primary" @click="createPreShip">创建</el-button>
        <el-button v-else-if="active==='deliveries'" type="primary" @click="createDelivery">创建</el-button>
        <el-button v-else-if="active==='price-locks'" type="primary" @click="createLock">保存</el-button>
        <el-button v-else-if="active==='contracts'" type="primary" @click="createContract">保存</el-button>
        <el-button v-else-if="active==='boms'" type="primary" @click="createBom">保存</el-button>
        <el-button v-else-if="active==='budgets'" type="primary" @click="createBudget">测算</el-button>
        <el-button v-else-if="active==='prints'" type="primary" @click="doPrint">生成预览</el-button>
        <el-button v-else-if="active==='self-orders'" type="primary" @click="submitSelf">提交自助单</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editDlg" title="修改订单" width="520px" destroy-on-close>
      <el-form label-width="90px">
        <el-form-item label="产品"><ProductSelect v-model="editForm.product_id" /></el-form-item>
        <el-form-item label="数量"><el-input-number v-model="editForm.qty" :min="1" style="width:100%" /></el-form-item>
        <el-form-item label="单价"><el-input-number v-model="editForm.price" :min="0" :step="0.1" style="width:100%" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="editForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDlg = false">取消</el-button>
        <el-button type="primary" @click="saveEditOrder">保存修改</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="bomDlg" title="编辑 BOM 明细" width="640px">
      <el-button size="small" @click="addBomLine">加行</el-button>
      <el-table :data="bomLines" size="small" class="mt">
        <el-table-column label="原料">
          <template #default="{ row }"><ProductSelect v-model="row.material_product_id" /></template>
        </el-table-column>
        <el-table-column label="用量" width="140">
          <template #default="{ row }"><el-input-number v-model="row.qty" :min="0" :step="0.1" /></template>
        </el-table-column>
        <el-table-column label="损耗" width="140">
          <template #default="{ row }"><el-input-number v-model="row.scrap_rate" :min="0" :step="0.01" /></template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="bomDlg = false">取消</el-button>
        <el-button type="primary" @click="saveBomLines">保存明细</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="ruleDlg" title="自助下单规则" width="480px">
      <el-form label-width="90px">
        <el-form-item label="名称"><el-input v-model="ruleForm.name" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="ruleForm.enabled" /></el-form-item>
        <el-form-item label="最小量"><el-input-number v-model="ruleForm.min_qty" :min="0" /></el-form-item>
        <el-form-item label="最大量"><el-input-number v-model="ruleForm.max_qty" :min="0" /></el-form-item>
        <el-form-item label="限额"><el-input-number v-model="ruleForm.max_amount" :min="0" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="ruleForm.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDlg = false">取消</el-button>
        <el-button type="primary" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { background: #fff; padding: 16px 18px; border-radius: 10px; border: 1px solid #e2e8ee; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 4px; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; color: #1f2a33; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; line-height: 1.5; max-width: 720px; }
.head-meta { flex-shrink: 0; padding-top: 2px; }
.meta-pill {
  display: inline-block; padding: 4px 10px; border-radius: 999px;
  background: #eef6f1; color: #2f6b4f; font-size: 12px; font-weight: 500;
}
.stats { display: grid; grid-template-columns: repeat(4, minmax(96px, 1fr)); gap: 10px; margin-bottom: 14px; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat.ok { background: #eef6f1; border-color: #d5eade; }
.stat.warn { background: #fff7f0; border-color: #f0e0d0; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; color: #1f2a33; }
.toolbar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 12px; }
.mode-hint { color: #714b67; font-size: 13px; margin: 0 0 12px; background: #f5eef8; padding: 8px 12px; border-radius: 6px; }
.muted { color: #98a2a8; font-size: 12px; }
.mb { margin-bottom: 12px; }
.mt { margin-top: 12px; }
.sub { margin: 16px 0 8px; font-size: 14px; }
</style>
