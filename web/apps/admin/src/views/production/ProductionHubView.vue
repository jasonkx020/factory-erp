<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  productionApi,
  productApi,
  hrApi,
  PROCESS_TYPE_OPTIONS,
  PROCESS_PAY_MODE_OPTIONS,
  STATUS_ACTIVE_OPTIONS,
  STATION_FLOW_EVENT_OPTIONS,
  formOptionLabel,
  QC_TYPE_OPTIONS,
  CONSIGNMENT_PROGRESS_OPTIONS,
} from '@erp/shared'
import {
  ProductSelect,
  ProcessSelect,
  WorkshopSelect,
  RoutingSelect,
  ProdTaskSelect,
  DispatchSelect,
  SupplierSelect,
  CustomerSelect,
  EmployeeSelect,
  RoleSelect,
  WarehouseSelect,
  EnumSelect,
} from '../../components/select'
import PieceIssueView from './PieceIssueView.vue'
import RoutingView from '../automation/RoutingView.vue'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import { useCarrierCodeLabel } from '../../composables/useCarrierCodeLabel'

type Row = Record<string, unknown>

const { codeLabel, ensureLoaded: ensureCarrierLabel } = useCarrierCodeLabel()

const processCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'process_type_label', label: '类型' },
  { prop: 'pay_mode_label', label: '计费' },
  { prop: 'status_label', label: '状态' },
]
const shiftCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '班次号', primary: true },
  { prop: 'biz_date', label: '日期' },
  { prop: 'workshop_name', label: '车间' },
  { prop: 'member_count', label: '人数' },
  { prop: 'status_label', label: '状态' },
]
const shiftMemberCols: MobileCardColumn[] = [
  { prop: 'employee_name', label: '员工', primary: true },
  { prop: 'process_name', label: '工序' },
]
const taskCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'status_label', label: '状态' },
  { prop: 'plan_qty', label: '计划' },
  { prop: 'completed_qty', label: '完工' },
  { prop: 'progress_pct', label: '进度%' },
  { prop: 'created_at', label: '创建时间' },
]
const dispatchCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'task_id', label: '任务' },
  { prop: 'process_name', label: '工序' },
  { prop: 'worker_name', label: '工人' },
  { prop: 'qty', label: '数量' },
  { prop: 'status_label', label: '状态' },
]
const pieceworkCols: MobileCardColumn[] = [
  { prop: 'id', label: 'ID', primary: true },
  { prop: 'worker_name', label: '工人' },
  { prop: 'process_name', label: '工序' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'qty', label: '产量' },
  { prop: 'amount', label: '金额' },
  { prop: 'status', label: '状态' },
]
const stationFlowCols: MobileCardColumn[] = [
  { prop: 'created_at', label: '时间', primary: true },
  { prop: 'event_type', label: '类型' },
  { prop: 'board_code', label: '板码' },
  { prop: 'process_name', label: '工序' },
  { prop: 'worker_name', label: '工人' },
  { prop: 'emp_type', label: '工种' },
  { prop: 'kg', label: 'kg' },
  { prop: 'amount', label: '金额' },
]
const bomCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'product_id', label: '成品' },
  { prop: 'version_no', label: '版本' },
  { prop: 'status', label: '状态' },
]
const mrpCols: MobileCardColumn[] = [
  { prop: 'run_no', label: '运算号', primary: true },
  { prop: 'run_at', label: '时间' },
  { prop: 'status', label: '状态' },
]
const reqCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'status', label: '状态' },
]
const workbenchCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'status_label', label: '状态' },
  { prop: 'created_at', label: '创建' },
]
const wipCols: MobileCardColumn[] = [
  { prop: 'step_name', label: '步骤', primary: true },
  { prop: 'seq_no', label: '序' },
  { prop: 'step_code', label: '步骤码' },
  { prop: 'process_name', label: '工序' },
  { prop: 'board_count', label: '板数' },
  { prop: 'available_kg', label: '可领 kg' },
  { prop: 'occupied_kg', label: '领取未完 kg' },
  { prop: 'wip_weight', label: '在制重量 kg' },
  { prop: 'stock_kg', label: '在仓 kg' },
  { prop: 'stock_box_count', label: '在仓板数' },
]
const wipBoxCols: MobileCardColumn[] = [
  { prop: 'code', label: '板码', primary: true },
  { prop: 'product_name', label: '产品' },
  { prop: 'available_kg', label: '可领 kg' },
  { prop: 'occupied_kg', label: '领取未完 kg' },
  { prop: 'trace_code', label: '溯源' },
  { prop: 'status', label: '状态' },
]
const yieldTraceCols: MobileCardColumn[] = [
  { prop: 'process_name', label: '工序', primary: true },
  { prop: 'trace_code', label: '溯源' },
  { prop: 'board_count', label: '板数' },
  { prop: 'input_kg', label: '投料 kg' },
  { prop: 'output_kg', label: '完工 kg' },
  { prop: 'loss_kg', label: '扣损 kg' },
  { prop: 'loss_rate', label: '扣损率' },
]
const progressCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '任务单', primary: true },
  { prop: 'status', label: '状态' },
  { prop: 'plan_qty', label: '计划' },
  { prop: 'completed_qty', label: '完成' },
  { prop: 'progress_pct', label: '进度%' },
  { prop: 'created_at', label: '创建' },
]
const mergeCols: MobileCardColumn[] = [
  { prop: 'merge_no', label: '整合号', primary: true },
  { prop: 'title', label: '标题' },
  { prop: 'status', label: '状态' },
  { prop: 'result_task_id', label: '结果任务' },
]
const drawingCols: MobileCardColumn[] = [
  { prop: 'drawing_code', label: '编码', primary: true },
  { prop: 'drawing_name', label: '名称' },
  { prop: 'task_id', label: '任务' },
  { prop: 'file_url', label: '文件' },
]
const qcCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'qc_type', label: '类型' },
  { prop: 'result', label: '结果' },
  { prop: 'status', label: '状态' },
]
const reworkCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'process_id', label: '工序' },
  { prop: 'qty', label: '数量' },
  { prop: 'status', label: '状态' },
]
const scrapCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'product_id', label: '料号' },
  { prop: 'qty', label: '数量' },
  { prop: 'scrap_type', label: '类型' },
  { prop: 'status', label: '状态' },
]
const processReturnCols = computed<MobileCardColumn[]>(() => [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'box_code', label: codeLabel.value },
  { prop: 'return_weight', label: '退回kg' },
  { prop: 'reason', label: '原因' },
  { prop: 'status', label: '状态' },
])
const outsourceCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'supplier_id', label: '供应商' },
  { prop: 'qty', label: '数量' },
  { prop: 'received_qty', label: '收回' },
  { prop: 'status', label: '状态' },
]
const consignmentCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'customer_id', label: '客户' },
  { prop: 'qty', label: '数量' },
  { prop: 'progress', label: '进度' },
  { prop: 'status', label: '状态' },
]
const costHideCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'role_id', label: '角色' },
  { prop: 'field_scope', label: '字段范围' },
  { prop: 'is_enabled', label: '启用' },
]

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  processes: '工序定义',
  'process-mgmt': '工序定义',
  routings: '工艺流程',
  shifts: '产线班次',
  tasks: '生产任务单',
  'multi-products': '一单多商品',
  dispatches: '例外派岗',
  flex: '灵活派发工单',
  reports: '工序流水',
  'process-reports': '工序流水',
  piecework: '计件工资',
  'piece-issue': '计件领料表',
  boms: '自动BOM',
  mrp: 'MRP物料分析',
  requisitions: '联动式领料',
  workbench: '车间工作台',
  'process-wip': '工序在制',
  'trace-production': '溯源生产',
  'process-yield': '工序扣损',
  outsources: '委外加工',
  consignments: '受托加工',
  'cost-hide': '成本隐藏',
  progress: '进度跟踪',
  merges: '多单整合',
  drawings: '图纸分发',
  qc: '质检管理',
  reworks: '返修单',
  scraps: '废料管理',
  'process-returns': '退库（未用完还仓）',
}

const active = computed(() => String(route.params.section || 'tasks'))
const title = computed(() => TITLE_MAP[active.value] || '生产管理')
const useCustomHead = computed(() =>
  ['shifts', 'tasks', 'dispatches', 'flex', 'workbench', 'process-wip', 'trace-production'].includes(active.value),
)
const embedRoutings = computed(() => active.value === 'routings')
const embedPieceIssue = computed(() => active.value === 'piece-issue')

/** 默认 false：现场录入仅 App；例外派岗创建受此开关影响 */
const fieldInputOnAdmin = import.meta.env.VITE_FIELD_INPUT_ON_ADMIN === 'true'

const loading = ref(false)
const list = ref<Row[]>([])
const detail = ref<Row | null>(null)
const taskFlowSteps = ref<Row[]>([])
const taskDetailDrawer = ref(false)

const isTaskDetail = computed(() => Array.isArray(detail.value?.items))
const taskItems = computed(() => ((detail.value?.items as Row[]) || []))
const taskPlanTotal = computed(() =>
  taskItems.value.reduce((s, it) => s + Number(it.plan_qty ?? it.qty ?? 0), 0),
)
const taskDoneTotal = computed(() =>
  taskItems.value.reduce((s, it) => s + Number(it.completed_qty ?? 0), 0),
)
const taskProgressPct = computed(() => {
  const plan = taskPlanTotal.value
  if (plan <= 0) return 0
  return Math.min(100, Math.round((taskDoneTotal.value / plan) * 1000) / 10)
})

function productLabel(pid: number) {
  const p = products.value.find((x) => Number(x.id) === pid)
  return p ? String(p.name || p.code || pid) : String(pid)
}

function itemProgress(it: Row) {
  const plan = Number(it.plan_qty ?? it.qty ?? 0)
  const done = Number(it.completed_qty ?? 0)
  if (plan <= 0) return Number(it.progress_pct ?? 0)
  return Math.min(100, Math.round((done / plan) * 1000) / 10)
}

function statusTagType(st: unknown): 'success' | 'warning' | 'info' | 'danger' {
  const s = String(st || '')
  if (s === 'closed' || s === 'done' || s === 'finished' || s === 'received' || s === 'confirmed') return 'success'
  if (s === 'in_progress' || s === 'released' || s === 'dispatched' || s === 'reassigned' || s === 'open') return 'warning'
  if (s === 'cancelled' || s === 'void' || s === 'failed') return 'danger'
  return 'info'
}

function docStatusLabel(st: unknown): string {
  const map: Record<string, string> = {
    pending: '待开始',
    released: '已释放',
    in_progress: '进行中',
    closed: '已关闭',
    cancelled: '已取消',
    merged: '已整合',
    dispatched: '已派发',
    reassigned: '已改派',
    received: '已接收',
    confirmed: '已确认',
    done: '已完成',
    finished: '已完成',
    open: '开工中',
  }
  const s = String(st || '')
  return map[s] || s || '—'
}
const products = ref<Row[]>([])
const processes = ref<Row[]>([])
const workers = ref<Row[]>([])
const overview = ref<Row | null>(null)
const shiftDetail = ref<Row | null>(null)
const wipProductId = ref<number | null>(null)
const wipSummary = ref<Row | null>(null)
const wipDrawer = ref(false)
const wipBoxes = ref<Row[]>([])
const wipDrawerTitle = ref('')
const yieldTraceCode = ref('')
const traceProdFilter = ref('')
const traceProdWip = ref<Row | null>(null)
const traceProdReport = ref<Row | null>(null)
const traceProdCode = ref('')
const traceProdBusy = ref(false)
const traceStartDialogVisible = ref(false)
const traceStartOptions = ref<Row[]>([])
const traceStartProductName = ref('')
const traceStartSelectedRoutingId = ref<number | null>(null)
const traceStartSuggestedId = ref<number | null>(null)

function traceStepStatusLabel(st: unknown): string {
  switch (String(st || '')) {
    case 'done': return '已完成'
    case 'in_progress': return '进行中'
    case 'ready': return '可结束'
    default: return '待做'
  }
}
function traceStepStatusType(st: unknown): 'success' | 'warning' | 'info' | '' {
  switch (String(st || '')) {
    case 'done': return 'success'
    case 'in_progress': return 'warning'
    case 'ready': return 'info'
    default: return ''
  }
}
const shiftMembers = computed(() =>
  ((shiftDetail.value?.members as Row[]) || []).map((m) => ({
    ...m,
    process_name: m.process_id === 0 ? '全工序' : m.process_name,
  })),
)
const shiftForm = reactive({ workshop_dept_id: null as number | null, remark: '产线开工' })
const shiftMemberForm = reactive({ employee_id: null as number | null, process_id: 0 })
const shiftKeyword = ref('')
const shiftWorkshopFilter = ref<number | null>(null)
const shiftStatusFilter = ref('')
const shiftCreateDlg = ref(false)
const shiftDrawer = ref(false)

function shiftStatusLabel(st: unknown): string {
  const s = String(st || '')
  if (s === 'open') return '开工中'
  if (s === 'closed') return '已收工'
  return s || '—'
}
function shiftStatusTagType(st: unknown): 'success' | 'info' {
  return String(st) === 'open' ? 'success' : 'info'
}

const shiftFiltered = computed(() => {
  const kw = shiftKeyword.value.trim().toLowerCase()
  const ws = shiftWorkshopFilter.value
  const st = shiftStatusFilter.value
  return list.value
    .filter((r) => {
      if (ws && Number(r.workshop_dept_id) !== ws) return false
      if (st && String(r.status) !== st) return false
      if (kw) {
        const no = String(r.doc_no || '').toLowerCase()
        const shop = String(r.workshop_name || '').toLowerCase()
        const remark = String(r.remark || '').toLowerCase()
        if (!no.includes(kw) && !shop.includes(kw) && !remark.includes(kw)) return false
      }
      return true
    })
    .map((r) => ({ ...r, status_label: shiftStatusLabel(r.status) }))
})

const shiftStats = computed(() => {
  const rows = list.value
  const today = new Date().toISOString().slice(0, 10)
  return {
    open: rows.filter((r) => r.status === 'open').length,
    closed: rows.filter((r) => r.status === 'closed').length,
    today: rows.filter((r) => String(r.biz_date || '').slice(0, 10) === today).length,
    members: rows.reduce((s, r) => s + Number(r.member_count || 0), 0),
  }
})
const shiftOpen = computed(() => String(shiftDetail.value?.status) === 'open')

function shiftRowClass({ row }: { row: Row }) {
  return shiftDetail.value && Number(row.id) === Number(shiftDetail.value.id) ? 'is-current-shift' : ''
}

function processName(id: unknown) {
  const n = Number(id || 0)
  if (!n) return '—'
  const p = processes.value.find((x) => Number(x.id) === n)
  return p ? String(p.name || p.code || n) : `#${n}`
}
function workerName(id: unknown) {
  const n = Number(id || 0)
  if (!n) return '—'
  const w = workers.value.find((x) => Number(x.id) === n)
  return w ? String(w.name || w.emp_no || n) : `#${n}`
}

const taskKeyword = ref('')
const taskStatusFilter = ref('')
const taskCreateDlg = ref(false)
const dispatchKeyword = ref('')
const dispatchStatusFilter = ref('')
const dispatchCreateDlg = ref(false)

const taskFiltered = computed(() => {
  const kw = taskKeyword.value.trim().toLowerCase()
  const st = taskStatusFilter.value
  return list.value
    .filter((r) => {
      if (st && String(r.status) !== st) return false
      if (kw && !String(r.doc_no || '').toLowerCase().includes(kw) && !String(r.remark || '').toLowerCase().includes(kw)) return false
      return true
    })
    .map((r) => ({ ...r, status_label: docStatusLabel(r.status) }))
})
const taskStats = computed(() => {
  const rows = list.value
  return {
    pending: rows.filter((r) => r.status === 'pending').length,
    running: rows.filter((r) => r.status === 'in_progress' || r.status === 'released').length,
    closed: rows.filter((r) => r.status === 'closed').length,
    total: rows.length,
  }
})

const dispatchFiltered = computed(() => {
  const kw = dispatchKeyword.value.trim().toLowerCase()
  const st = dispatchStatusFilter.value
  return list.value
    .filter((r) => {
      if (st && String(r.status) !== st) return false
      if (kw && !String(r.doc_no || '').toLowerCase().includes(kw)) return false
      return true
    })
    .map((r) => ({
      ...r,
      qty: r.qty ?? r.plan_qty,
      process_name: r.process_name || processName(r.process_id),
      worker_name: r.worker_name || workerName(r.worker_id),
      status_label: docStatusLabel(r.status),
    }))
})
const dispatchStats = computed(() => {
  const rows = list.value
  return {
    dispatched: rows.filter((r) => r.status === 'dispatched').length,
    reassigned: rows.filter((r) => r.status === 'reassigned').length,
    received: rows.filter((r) => r.status === 'received' || r.status === 'closed' || r.status === 'done').length,
    total: rows.length,
  }
})
const canCreateDispatch = computed(() => active.value === 'flex' || fieldInputOnAdmin)
const workbenchDisplay = computed(() =>
  list.value.map((r) => ({ ...r, status_label: docStatusLabel(r.status) })),
)

const taskForm = reactive({ product_id: 3, qty: 1000, routing_id: 1, workshop_dept_id: 0, remark: '' })
const multiLines = ref<{ product_id: number; qty: number }[]>([{ product_id: 3, qty: 100 }])
const processEditDlg = ref(false)
const processEditForm = reactive({ id: 0, code: '', name: '', process_type: 'other', pay_mode: 'none', status: 'active' })
const dispatchForm = reactive({ task_id: null as number | null, process_id: null as number | null, worker_id: null as number | null, qty: 100 })
const reqForm = reactive({ product_id: 1, qty: 100, warehouse_id: 1 })
const processForm = reactive({ code: '', name: '', process_type: 'other', pay_mode: 'none', status: 'active' })
const processDlg = ref(false)
const bomForm = reactive({ product_id: 3, name: '生产BOM', component_product_id: 1, qty: 1.2, scrap_rate: 0.05 })
const scrapForm = reactive({ product_id: 1, qty: 10, scrap_type: 'cut_defect', process_id: 1, remark: '' })
const returnForm = reactive({ box_code: '', return_weight: 30, warehouse_id: 1, reason: '提前下班' })
const returnStatusFilter = ref('')
const transferUserId = ref<number | null>(null)
const qcForm = reactive({ qc_type: 'process', product_id: 3, process_id: 1, qty: 100 })
const reworkForm = reactive({ process_id: 1, qty: 10, remark: '' })
const mergeForm = reactive({ title: '多单整合', task_ids: [] as number[] })
const drawingForm = reactive({ drawing_code: '', drawing_name: '', task_id: null as number | null, file_url: '' })
const outForm = reactive({ supplier_id: 1, process_id: 1, product_id: 3, qty: 100 })
const consForm = reactive({ customer_id: 1, product_id: 3, qty: 100, progress: '待投产' })
const hideForm = reactive({ role_id: 1, name: '隐藏成本字段' })
const payForm = reactive({ transfer_no: '', pay_evidence_url: '' })
const pieceBizDate = ref(new Date().toISOString().slice(0, 10))
const stationFlowList = ref<Row[]>([])
const stationFlowFilter = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  board_code: '',
  event_type: '',
  has_amount: false,
})

async function loadMeta() {
  const [p, proc, emp] = await Promise.all([
    productApi.list(),
    productionApi.processes(),
    hrApi.employees(),
  ])
  products.value = ((p.data as { list?: Row[] })?.list) || []
  processes.value = ((proc.data as { list?: Row[] })?.list) || []
  workers.value = ((emp.data as { list?: Row[] })?.list) || []
  if (processes.value[0]) {
    const id = Number(processes.value[0].id)
    dispatchForm.process_id = id
    scrapForm.process_id = id
    qcForm.process_id = id
    reworkForm.process_id = id
    outForm.process_id = id
  }
  if (products.value[0]) {
    const pid = Number(products.value[0].id)
    taskForm.product_id = pid
    bomForm.product_id = pid
    if (multiLines.value[0]) multiLines.value[0].product_id = pid
  }
}

async function refresh() {
  if (embedRoutings.value || embedPieceIssue.value) return
  loading.value = true
  detail.value = null
  taskDetailDrawer.value = false
  taskFlowSteps.value = []
  try {
    let res
    switch (active.value) {
      case 'processes':
      case 'process-mgmt':
        res = await productionApi.processes()
        break
      case 'shifts':
        res = await productionApi.shifts()
        break
      case 'tasks':
      case 'multi-products':
        res = await productionApi.listTasks()
        break
      case 'dispatches':
        res = await productionApi.listDispatches()
        break
      case 'flex':
        res = await productionApi.listFlexDispatches()
        break
      case 'reports':
      case 'process-reports':
        await loadStationFlow()
        list.value = stationFlowList.value
        return
      case 'piecework': {
        const q = pieceBizDate.value ? `biz_date=${encodeURIComponent(pieceBizDate.value)}` : undefined
        res = await productionApi.pieceworkSummaries(q)
        break
      }
      case 'boms':
        res = await productionApi.boms()
        break
      case 'mrp':
        res = await productionApi.mrpRuns()
        break
      case 'requisitions':
        res = await productionApi.listRequisitions()
        break
      case 'workbench': {
        const [ov, today] = await Promise.all([
          productionApi.workbenchOverview(),
          productionApi.workbenchToday(),
        ])
        overview.value = (ov.data as Row) || null
        list.value = ((today.data as { list?: Row[] })?.list) || []
        return
      }
      case 'process-wip': {
        const qs = wipProductId.value ? `product_id=${wipProductId.value}` : ''
        res = await productionApi.processWip(qs || undefined)
        if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
        wipSummary.value = (res.data as Row) || null
        list.value = ((res.data as { steps?: Row[] })?.steps) || []
        return
      }
      case 'trace-production': {
        const qs = [
          'page_size=200',
          'page_num=1',
          ...(traceProdFilter.value ? [`status=${encodeURIComponent(traceProdFilter.value)}`] : []),
        ].join('&')
        res = await productionApi.traceProductions(qs)
        break
      }
      case 'process-yield': {
        const qs = yieldTraceCode.value.trim()
          ? `trace_code=${encodeURIComponent(yieldTraceCode.value.trim())}`
          : ''
        res = await productionApi.processYieldTraces(qs || undefined)
        break
      }
      case 'outsources':
        res = await productionApi.outsources()
        break
      case 'consignments':
        res = await productionApi.consignments()
        break
      case 'cost-hide':
        res = await productionApi.costHidePolicies()
        break
      case 'progress':
        res = await productionApi.progress()
        break
      case 'merges':
        res = await productionApi.taskMerges()
        break
      case 'drawings':
        res = await productionApi.drawingLinks()
        break
      case 'qc':
        res = await productionApi.qcOrders()
        break
      case 'reworks':
        res = await productionApi.reworks()
        break
      case 'scraps':
        res = await productionApi.scraps()
        break
      case 'process-returns': {
        const qs = returnStatusFilter.value ? `status=${encodeURIComponent(returnStatusFilter.value)}` : ''
        res = await productionApi.listProcessReturns(qs || undefined)
        break
      }
      default:
        res = await productionApi.listTasks()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function openCreateTask() {
  taskForm.remark = ''
  if (products.value[0] && !taskForm.product_id) taskForm.product_id = Number(products.value[0].id)
  taskCreateDlg.value = true
}

async function createTask() {
  if (!taskForm.product_id) return ElMessage.warning('请选择产品')
  if (!taskForm.qty) return ElMessage.warning('请填写计划量')
  const res = await productionApi.createTask({ ...taskForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`任务 ${(res.data as Row)?.doc_no}`)
  dispatchForm.task_id = Number((res.data as Row)?.id)
  taskCreateDlg.value = false
  await refresh()
  const id = Number((res.data as Row)?.id)
  if (id) await openTask(id)
}

async function closeTask(id: number) {
  const res = await productionApi.closeTask(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已关闭')
  await refresh()
}

async function openTask(id: number) {
  const res = await productionApi.getTask(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  const items = await productionApi.taskItems(id)
  const row = { ...(res.data as Row), items: (items.data as { list?: Row[] })?.list || [] }
  detail.value = row
  taskFlowSteps.value = []
  const rid = Number(row.routing_id || 0)
  if (rid > 0) {
    try {
      const fr = await productionApi.flowRules(rid)
      const data = fr.data as { steps?: Row[]; list?: Row[] } | Row[] | undefined
      if (Array.isArray(data)) {
        taskFlowSteps.value = data
      } else if (data && Array.isArray(data.steps)) {
        taskFlowSteps.value = data.steps
      } else if (data && Array.isArray(data.list)) {
        taskFlowSteps.value = data.list
      } else {
        taskFlowSteps.value = []
      }
    } catch {
      taskFlowSteps.value = []
    }
  }
  taskDetailDrawer.value = true
}

function closeTaskDetail() {
  taskDetailDrawer.value = false
}

function openCreateDispatch() {
  dispatchForm.task_id = null
  dispatchForm.process_id = processes.value[0] ? Number(processes.value[0].id) : null
  dispatchForm.worker_id = null
  dispatchForm.qty = 100
  dispatchCreateDlg.value = true
}

async function createDispatch() {
  if (!dispatchForm.task_id) return ElMessage.warning('请选择任务')
  if (!dispatchForm.process_id) return ElMessage.warning('请选择工序')
  if (!dispatchForm.worker_id) return ElMessage.warning('请选择工人')
  const apiCall = active.value === 'flex' ? productionApi.createFlexDispatch : productionApi.createDispatch
  const res = await apiCall({ ...dispatchForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`派工 ${(res.data as Row)?.doc_no}`)
  dispatchCreateDlg.value = false
  await refresh()
}

async function receiveDispatch(id: number) {
  const res = await productionApi.receiveDispatch(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已接收')
  await refresh()
}

async function daySettlePiece() {
  const res = await productionApi.daySettlePiecework({ biz_date: pieceBizDate.value })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = (res.data || {}) as Row
  ElMessage.success(`日结完成：${d.settled_rows || 0} 笔 / ${d.settled_kg || 0} kg / ¥${d.settled_amount || 0}`)
  await refresh()
}

async function loadTraceProdWip() {
  const code = String(traceProdCode.value || '').trim()
  if (!code) return ElMessage.warning('请输入溯源码')
  traceProdBusy.value = true
  const res = await productionApi.traceProductionWip(code)
  traceProdBusy.value = false
  if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
  traceProdWip.value = (res.data as Row) || null
  traceProdReport.value = null
  if (traceProdWip.value?.ui_status === 'ended') {
    await loadTraceProdReport()
  }
}

async function openTraceStartDialog() {
  const code = String(traceProdCode.value || '').trim()
  if (!code) return ElMessage.warning('请输入溯源码')
  traceProdBusy.value = true
  const res = await productionApi.traceProductionRoutingOptions(code)
  traceProdBusy.value = false
  if (res.code !== 1) return ElMessage.error(res.msg || '加载工艺选项失败')
  const d = (res.data || {}) as Row
  traceStartOptions.value = (d.routing_options as Row[]) || []
  traceStartProductName.value = String(d.product_name || '')
  const suggested = Number(d.suggested_routing_id || 0)
  traceStartSuggestedId.value = suggested > 0 ? suggested : null
  traceStartSelectedRoutingId.value =
    suggested > 0 ? suggested : (traceStartOptions.value[0]?.id as number | undefined) ?? null
  if (!traceStartOptions.value.length) {
    return ElMessage.warning('该原料产品暂无可用工艺，请先在「工艺流程」配置')
  }
  traceStartDialogVisible.value = true
}

const traceStartPreviewSteps = computed(() => {
  const rid = traceStartSelectedRoutingId.value
  const opt = traceStartOptions.value.find((o) => Number(o.id) === Number(rid))
  return (opt?.steps_preview as string[]) || []
})

async function confirmTraceStart() {
  const code = String(traceProdCode.value || '').trim()
  const routingId = traceStartSelectedRoutingId.value
  if (!code) return ElMessage.warning('请输入溯源码')
  if (!routingId) return ElMessage.warning('请选择工艺流程')
  traceProdBusy.value = true
  const res = await productionApi.startTraceProduction({ trace_code: code, routing_id: routingId })
  traceProdBusy.value = false
  if (res.code !== 1) return ElMessage.error(res.msg || '进入生产失败')
  traceStartDialogVisible.value = false
  ElMessage.success('已进入生产，工艺路线已锁定')
  await loadTraceProdWip()
  await refresh()
}

async function startTraceProduction() {
  await openTraceStartDialog()
}

function traceProdTimelineSteps(): Row[] {
  const wip = traceProdWip.value
  if (!wip) return []
  const routing = (wip.routing_steps as Row[]) || []
  if (routing.length) return routing
  return (wip.steps as Row[]) || []
}

function canCompleteTraceStep(row: Row): boolean {
  const wip = traceProdWip.value
  if (!wip || wip.ui_status !== 'in_production') return false
  const action = String(row.action || row.step_status || '')
  if (action === 'complete' || row.step_status === 'ready') {
    return Number(wip.can_complete_process_id) === Number(row.process_id)
  }
  return false
}

async function loadTraceProdReport() {
  const code = String(traceProdCode.value || '').trim()
  if (!code) return
  const res = await productionApi.traceProductionReport(code)
  if (res.code !== 1) return ElMessage.error(res.msg || '报表加载失败')
  traceProdReport.value = (res.data as Row) || null
}

async function completeTraceProcessStep(processId: number) {
  const code = String(traceProdCode.value || '').trim()
  if (!code || !processId) return
  traceProdBusy.value = true
  const res = await productionApi.completeTraceProcess({ trace_code: code, process_id: processId })
  traceProdBusy.value = false
  if (res.code !== 1) return ElMessage.error(res.msg || '结束工序失败')
  const d = (res.data || {}) as Row
  if (d.auto_finalized) ElMessage.success('末道工序已结束，溯源生产已自动结案')
  else ElMessage.success('工序已结束')
  await loadTraceProdWip()
  if (traceProdWip.value?.ui_status === 'ended') await loadTraceProdReport()
  await refresh()
}

async function loadStationFlow() {
  const qs = new URLSearchParams()
  if (stationFlowFilter.biz_date) qs.set('biz_date', stationFlowFilter.biz_date)
  if (stationFlowFilter.board_code) qs.set('board_code', stationFlowFilter.board_code)
  if (stationFlowFilter.event_type) qs.set('event_type', stationFlowFilter.event_type)
  if (stationFlowFilter.has_amount) qs.set('has_amount', '1')
  const res = await productionApi.stationFlowLogs(qs.toString())
  if (res.code !== 1) {
    stationFlowList.value = []
    return
  }
  stationFlowList.value = ((res.data as { list?: Row[] })?.list) || []
}

async function payPiece(id: number) {
  if (!payForm.transfer_no) return ElMessage.warning('请填转账单号')
  const res = await productionApi.payPiecework(id, { ...payForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已支付')
  await refresh()
}

function processTypeLabel(v: unknown) {
  return formOptionLabel(PROCESS_TYPE_OPTIONS, v)
}
function processPayModeLabel(v: unknown) {
  return formOptionLabel(PROCESS_PAY_MODE_OPTIONS, v)
}
function processStatusLabel(v: unknown) {
  return formOptionLabel(STATUS_ACTIVE_OPTIONS, v)
}

const processDisplayList = computed(() =>
  list.value.map((row) => ({
    ...row,
    process_type_label: processTypeLabel(row.process_type),
    pay_mode_label: processPayModeLabel(row.pay_mode || (row.is_piecework ? 'weight' : 'none')),
    status_label: processStatusLabel(row.status || 'active'),
  })),
)

function resetProcessForm() {
  processForm.code = ''
  processForm.name = ''
  processForm.process_type = 'other'
  processForm.pay_mode = 'none'
  processForm.status = 'active'
}

function openCreateProcess() {
  resetProcessForm()
  processDlg.value = true
}

async function createProcess() {
  if (!processForm.name.trim()) return ElMessage.warning('请填工序名称')
  const code = processForm.code.trim() || `P${Date.now().toString().slice(-6)}`
  const res = await productionApi.createProcess({
    code,
    name: processForm.name.trim(),
    process_type: processForm.process_type,
    pay_mode: processForm.pay_mode,
    status: processForm.status,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('工序已创建')
  processDlg.value = false
  resetProcessForm()
  await loadMeta()
  await refresh()
}

function openEditProcess(row: Row) {
  processEditForm.id = Number(row.id)
  processEditForm.code = String(row.code || '')
  processEditForm.name = String(row.name || '')
  processEditForm.process_type = String(row.process_type || 'other')
  processEditForm.pay_mode = String(row.pay_mode || (row.is_piecework ? 'weight' : 'none'))
  processEditForm.status = String(row.status || 'active')
  processEditDlg.value = true
}

async function saveEditProcess() {
  if (!processEditForm.id) return
  if (!processEditForm.name.trim()) return ElMessage.warning('请填工序名称')
  const res = await productionApi.updateProcess(processEditForm.id, {
    code: processEditForm.code.trim(),
    name: processEditForm.name.trim(),
    process_type: processEditForm.process_type,
    pay_mode: processEditForm.pay_mode,
    status: processEditForm.status,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('工序已更新')
  processEditDlg.value = false
  await loadMeta()
  await refresh()
}

function addMultiLine() {
  const pid = products.value[0] ? Number(products.value[0].id) : taskForm.product_id
  multiLines.value.push({ product_id: pid, qty: 100 })
}

function removeMultiLine(idx: number) {
  if (multiLines.value.length <= 1) return
  multiLines.value.splice(idx, 1)
}

async function createMultiTask() {
  if (!multiLines.value.length) return ElMessage.warning('请至少添加一行商品')
  const first = multiLines.value[0]
  const res = await productionApi.createTask({
    product_id: first.product_id,
    qty: first.qty,
    routing_id: taskForm.routing_id,
    workshop_dept_id: taskForm.workshop_dept_id,
    remark: taskForm.remark || '一单多商品',
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const taskId = Number((res.data as Row)?.id)
  for (let i = 1; i < multiLines.value.length; i++) {
    const line = multiLines.value[i]
    const itemRes = await productionApi.addTaskItem(taskId, { product_id: line.product_id, qty: line.qty })
    if (itemRes.code !== 1) return ElMessage.error(itemRes.msg || '添加商品行失败')
  }
  ElMessage.success(`多商品任务 ${(res.data as Row)?.doc_no}`)
  dispatchForm.task_id = taskId
  await refresh()
}

async function openWipBoxes(stepId: number, title: string, unassigned: boolean) {
  const parts = [
    unassigned ? 'unassigned=1' : `step_id=${stepId}`,
    wipProductId.value ? `product_id=${wipProductId.value}` : '',
  ].filter(Boolean)
  const res = await productionApi.processWipBoxes(parts.join('&'))
  if (res.code !== 1) return ElMessage.error(res.msg || '加载板明细失败')
  wipBoxes.value = ((res.data as { list?: Row[] })?.list) || []
  wipDrawerTitle.value = `${title || '板明细'}（${wipBoxes.value.length}）`
  wipDrawer.value = true
}

async function createBom() {
  const res = await productionApi.createBom({ ...bomForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`BOM ${(res.data as Row)?.code}`)
  await refresh()
}

async function genBom() {
  const res = await productionApi.generateBom({ product_id: bomForm.product_id })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已自动生成BOM')
  await refresh()
}

async function runMrp() {
  const res = await productionApi.createMrpRun({})
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`MRP ${(res.data as Row)?.run_no}`)
  detail.value = (res.data as Row) || null
  await refresh()
}

async function openMrp(id: number) {
  const res = await productionApi.mrpResults(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

async function createReq() {
  const res = await productionApi.createRequisition({ ...reqForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('领料单已创建')
  await refresh()
}

async function postReq(id: number) {
  const res = await productionApi.postRequisition(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已过账扣料')
  await refresh()
}

async function createScrap() {
  const res = await productionApi.createScrap({ ...scrapForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('废料已登记')
  await refresh()
}

async function createProcessReturn() {
  if (!returnForm.box_code.trim()) return ElMessage.warning(`请填写${codeLabel.value}`)
  const res = await productionApi.createProcessReturn({ ...returnForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`退库单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function actProcessReturn(id: number, act: 'submit' | 'approve' | 'reject' | 'warehouse-confirm' | 'transfer') {
  let res
  if (act === 'submit') res = await productionApi.submitProcessReturn(id)
  else if (act === 'approve') res = await productionApi.approveProcessReturn(id)
  else if (act === 'reject') res = await productionApi.rejectProcessReturn(id, { remark: '驳回' })
  else if (act === 'warehouse-confirm') res = await productionApi.warehouseConfirmProcessReturn(id)
  else {
    if (!transferUserId.value) return ElMessage.warning('请填写转交用户ID')
    res = await productionApi.transferProcessReturn(id, { to_user_id: transferUserId.value })
  }
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已处理')
  await refresh()
}

async function createQc() {
  const res = await productionApi.createQcOrder({ ...qcForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('质检单已建')
  await refresh()
}

async function completeQc(id: number) {
  const res = await productionApi.completeQcOrder(id, { result: 'pass' })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('质检完成')
  await refresh()
}

async function createRework() {
  const res = await productionApi.createRework({ ...reworkForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('返修单已建')
  await refresh()
}

async function closeRework(id: number) {
  const res = await productionApi.closeRework(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已关闭')
  await refresh()
}

async function createMerge() {
  const ids = (mergeForm.task_ids || []).filter((n) => n > 0)
  if (!ids.length) return ElMessage.warning('请选择任务')
  const res = await productionApi.createTaskMerge({ title: mergeForm.title, task_ids: ids })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('整合单已建')
  await refresh()
}

async function confirmMerge(id: number) {
  const res = await productionApi.confirmTaskMerge(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已生成任务 ${(res.data as Row)?.result_task && ((res.data as Row).result_task as Row).doc_no}`)
  await refresh()
}

async function createDrawing() {
  const res = await productionApi.createDrawingLink({ ...drawingForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('图纸已挂接')
  await refresh()
}

async function createOut() {
  const res = await productionApi.createOutsource({ ...outForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('委外单已建')
  await refresh()
}

async function receiveOut(id: number) {
  const res = await productionApi.receiveOutsource(id, { qty: outForm.qty })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已收回')
  await refresh()
}

async function createCons() {
  const res = await productionApi.createConsignment({ ...consForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('受托单已建')
  await refresh()
}

async function updateConsProgress(id: number) {
  const res = await productionApi.consignmentProgress(id, { progress: consForm.progress, status: 'in_progress' })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('进度已更新')
  await refresh()
}

async function createHide() {
  const res = await productionApi.createCostHidePolicy({ ...hideForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('策略已保存')
  await refresh()
}

function openCreateShift() {
  shiftForm.workshop_dept_id = null
  shiftForm.remark = '产线开工'
  shiftCreateDlg.value = true
}

async function createShift() {
  if (!shiftForm.workshop_dept_id) return ElMessage.warning('请选择车间')
  const res = await productionApi.createShift({ ...shiftForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`班次 ${(res.data as Row)?.doc_no}`)
  shiftCreateDlg.value = false
  await refresh()
  const id = Number((res.data as Row)?.id)
  if (id) await openShift(id)
}

async function openShift(id: number) {
  const res = await productionApi.getShift(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  shiftDetail.value = (res.data as Row) || null
  shiftMemberForm.employee_id = null
  shiftMemberForm.process_id = 0
  shiftDrawer.value = true
}

async function closeShiftRow(id: number) {
  const res = await productionApi.closeShift(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已收工')
  await refresh()
  if (shiftDetail.value && Number(shiftDetail.value.id) === id) {
    await openShift(id)
  }
}

async function addShiftMember() {
  if (!shiftDetail.value?.id) return ElMessage.warning('请先选择班次')
  if (String(shiftDetail.value.status) !== 'open') return ElMessage.warning('已收工班次不可改成员')
  if (!shiftMemberForm.employee_id) return ElMessage.warning('请选择员工')
  const res = await productionApi.addShiftMember(Number(shiftDetail.value.id), {
    employee_id: shiftMemberForm.employee_id,
    process_id: shiftMemberForm.process_id,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已添加授权')
  shiftMemberForm.employee_id = null
  await openShift(Number(shiftDetail.value.id))
}

async function removeShiftMember(memberId: number) {
  if (!shiftDetail.value?.id) return
  if (String(shiftDetail.value.status) !== 'open') return ElMessage.warning('已收工班次不可改成员')
  const res = await productionApi.removeShiftMember(Number(shiftDetail.value.id), memberId)
  if (res.code !== 1) return ElMessage.error(res.msg)
  await openShift(Number(shiftDetail.value.id))
}

watch(active, () => {
  shiftDetail.value = null
  shiftDrawer.value = false
  shiftKeyword.value = ''
  shiftWorkshopFilter.value = null
  shiftStatusFilter.value = ''
  taskKeyword.value = ''
  taskStatusFilter.value = ''
  dispatchKeyword.value = ''
  dispatchStatusFilter.value = ''
  refresh()
})
watch(wipProductId, () => {
  if (active.value === 'process-wip') refresh()
})
onMounted(async () => {
  await ensureCarrierLabel()
  await loadMeta()
  await refresh()
})
</script>

<template>
  <div>
    <RoutingView v-if="embedRoutings" />
    <PieceIssueView v-else-if="embedPieceIssue" />

    <div v-else class="page" v-loading="loading">
      <div v-if="!useCustomHead" class="head">
        <h2>{{ title }}</h2>
        <p class="hint">管理端：配置、查询、结算与例外补单。日常过站/过磅请在 Flutter App 完成。</p>
      </div>

      <!-- 工序定义：新建 + 维护 -->
      <template v-if="active==='processes' || active==='process-mgmt'">
        <p class="mode-hint">
          计费：不计费 / 按重量 / 按件。仅「按重量|按件 × 计件工」才预估与日结金额（当前均按 kg×工价）。App 过站须手动指定工序。
        </p>
        <el-card class="mb">
          <div class="row" style="justify-content:space-between;margin-bottom:8px">
            <strong>工序列表</strong>
            <el-button type="primary" size="small" @click="openCreateProcess">新增工序</el-button>
          </div>
          <TableOrCards :data="processDisplayList" :loading="loading" :columns="processCols">
            <el-table :data="processDisplayList" size="small" stripe>
              <el-table-column prop="code" label="编码" width="120" />
              <el-table-column prop="name" label="名称" min-width="140" />
              <el-table-column prop="process_type_label" label="类型" width="100" />
              <el-table-column prop="pay_mode_label" label="计费" width="100">
                <template #default="{ row }">
                  <el-tag
                    size="small"
                    :type="row.pay_mode === 'weight' ? 'warning' : row.pay_mode === 'piece' ? 'success' : 'info'"
                  >{{ row.pay_mode_label }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="status_label" label="状态" width="90">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.status === 'inactive' ? 'danger' : 'success'">{{ row.status_label }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openEditProcess(row)">配置</el-button>
                </template>
              </el-table-column>
            </el-table>
            <template #extra="{ row }">
              <el-tag size="small" :type="row.status === 'inactive' ? 'danger' : 'success'">{{ row.status_label }}</el-tag>
            </template>
            <template #actions="{ row }">
              <el-button link type="primary" @click="openEditProcess(row)">配置</el-button>
            </template>
          </TableOrCards>
        </el-card>
      </template>

      <!-- 产线班次：替代日常派工授权 -->
      <template v-else-if="active==='shifts'">
        <header class="page-head">
          <div>
            <h2 class="title">产线班次</h2>
            <p class="desc">工人须在当日开工班次成员中才能 App 领料。与人事「考勤班次」无关。</p>
          </div>
          <div class="head-meta">
            <span class="meta-pill">筛选 {{ shiftFiltered.length }} / 全部 {{ list.length }}</span>
          </div>
        </header>

        <div class="shift-stats">
          <div class="stat ok">
            <div class="label">开工中</div>
            <div class="value">{{ shiftStats.open }}</div>
          </div>
          <div class="stat">
            <div class="label">已收工</div>
            <div class="value">{{ shiftStats.closed }}</div>
          </div>
          <div class="stat">
            <div class="label">今日班次</div>
            <div class="value">{{ shiftStats.today }}</div>
          </div>
          <div class="stat">
            <div class="label">授权人数</div>
            <div class="value">{{ shiftStats.members }}</div>
          </div>
        </div>

        <div class="row shift-toolbar">
          <el-button type="primary" @click="openCreateShift">开工</el-button>
          <el-button @click="refresh">刷新</el-button>
          <WorkshopSelect v-model="shiftWorkshopFilter" placeholder="全部车间" clearable />
          <el-select v-model="shiftStatusFilter" clearable placeholder="状态" style="width:120px">
            <el-option label="开工中" value="open" />
            <el-option label="已收工" value="closed" />
          </el-select>
          <el-input v-model="shiftKeyword" clearable placeholder="班次号 / 车间 / 备注" style="width:220px" />
        </div>

        <TableOrCards :data="shiftFiltered" :loading="loading" :columns="shiftCols" empty-text="暂无班次，请点击「开工」">
          <el-table
            :data="shiftFiltered"
            border
            stripe
            class="shift-table"
            empty-text="暂无班次"
            highlight-current-row
            :row-class-name="shiftRowClass"
            @row-click="(row: Row) => openShift(Number(row.id))"
          >
            <el-table-column prop="doc_no" label="班次号" min-width="150">
              <template #default="{ row }">
                <div class="name-cell">
                  <span class="name">{{ row.doc_no || '—' }}</span>
                  <span v-if="row.id" class="id-hint">#{{ row.id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="biz_date" label="日期" width="120" />
            <el-table-column prop="workshop_name" label="车间" min-width="120">
              <template #default="{ row }">
                <span :class="{ muted: !row.workshop_name }">{{ row.workshop_name || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="member_count" label="人数" width="80" align="center" />
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="shiftStatusTagType(row.status)">{{ row.status_label }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <span :class="{ muted: !row.remark }">{{ row.remark || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openShift(Number(row.id))">成员</el-button>
                <el-button v-if="row.status==='open'" link type="warning" @click.stop="closeShiftRow(Number(row.id))">收工</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" :type="shiftStatusTagType(row.status)">{{ row.status_label }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openShift(Number(row.id))">成员</el-button>
            <el-button v-if="row.status==='open'" link type="warning" @click="closeShiftRow(Number(row.id))">收工</el-button>
          </template>
        </TableOrCards>

        <el-dialog v-model="shiftCreateDlg" title="开工" width="480px" destroy-on-close>
          <el-form label-width="80px">
            <el-form-item label="车间" required>
              <WorkshopSelect v-model="shiftForm.workshop_dept_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="shiftForm.remark" placeholder="如：白班 / 夜班" maxlength="64" show-word-limit />
            </el-form-item>
          </el-form>
          <p class="form-tip">开工后请在成员抽屉中授权当日可过站工人。工序选「全工序」表示该工人本班次可过任意站。</p>
          <template #footer>
            <el-button @click="shiftCreateDlg = false">取消</el-button>
            <el-button type="primary" @click="createShift">开工</el-button>
          </template>
        </el-dialog>

        <el-drawer v-model="shiftDrawer" size="480px">
          <template #header>
            <div v-if="shiftDetail" class="shift-drawer-head">
              <span class="name">{{ shiftDetail.doc_no }}</span>
              <el-tag size="small" :type="shiftStatusTagType(shiftDetail.status)">{{ shiftStatusLabel(shiftDetail.status) }}</el-tag>
            </div>
          </template>
          <template v-if="shiftDetail">
            <div class="shift-meta">
              <span>日期 {{ shiftDetail.biz_date || '—' }}</span>
              <span>车间 {{ shiftDetail.workshop_name || '—' }}</span>
              <span>人数 {{ shiftMembers.length }}</span>
            </div>
            <p v-if="!shiftOpen" class="form-tip">已收工，成员只读。</p>
            <div v-else class="row shift-member-bar">
              <EmployeeSelect v-model="shiftMemberForm.employee_id" placeholder="选择员工" style="width:160px" />
              <el-select v-model="shiftMemberForm.process_id" placeholder="工序" style="width:140px">
                <el-option label="全工序" :value="0" />
                <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
              <el-button type="primary" @click="addShiftMember">添加</el-button>
            </div>
            <TableOrCards :data="shiftMembers" :columns="shiftMemberCols" empty-text="尚未授权成员">
              <el-table :data="shiftMembers" border stripe size="small" empty-text="尚未授权成员">
                <el-table-column prop="employee_name" label="员工" min-width="120" />
                <el-table-column label="工序" width="120">
                  <template #default="{ row }">{{ row.process_id === 0 ? '全工序' : (row.process_name || '—') }}</template>
                </el-table-column>
                <el-table-column v-if="shiftOpen" label="操作" width="80" fixed="right">
                  <template #default="{ row }">
                    <el-button link type="danger" @click="removeShiftMember(Number(row.id))">移除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <template #actions="{ row }">
                <el-button v-if="shiftOpen" link type="danger" @click="removeShiftMember(Number(row.id))">移除</el-button>
              </template>
            </TableOrCards>
          </template>
          <template #footer>
            <el-button @click="shiftDrawer = false">关闭</el-button>
            <el-button
              v-if="shiftOpen && shiftDetail"
              type="warning"
              @click="closeShiftRow(Number(shiftDetail.id))"
            >收工</el-button>
          </template>
        </el-drawer>
      </template>

      <!-- 生产任务单：单商品任务 -->
      <template v-else-if="active==='tasks'">
        <header class="page-head">
          <div>
            <h2 class="title">生产任务单</h2>
            <p class="desc">创建单商品生产任务；日常过站在 App。点行查看明细与工艺路径。</p>
          </div>
          <div class="head-meta">
            <span class="meta-pill">筛选 {{ taskFiltered.length }} / 全部 {{ list.length }}</span>
          </div>
        </header>
        <div class="shift-stats">
          <div class="stat"><div class="label">全部</div><div class="value">{{ taskStats.total }}</div></div>
          <div class="stat"><div class="label">待开始</div><div class="value">{{ taskStats.pending }}</div></div>
          <div class="stat ok"><div class="label">进行中</div><div class="value">{{ taskStats.running }}</div></div>
          <div class="stat"><div class="label">已关闭</div><div class="value">{{ taskStats.closed }}</div></div>
        </div>
        <div class="row shift-toolbar">
          <el-button type="primary" @click="openCreateTask">新建任务</el-button>
          <el-button @click="refresh">刷新</el-button>
          <el-select v-model="taskStatusFilter" clearable placeholder="状态" style="width:130px">
            <el-option label="待开始" value="pending" />
            <el-option label="已释放" value="released" />
            <el-option label="进行中" value="in_progress" />
            <el-option label="已关闭" value="closed" />
          </el-select>
          <el-input v-model="taskKeyword" clearable placeholder="单号 / 备注" style="width:200px" />
        </div>
        <TableOrCards :data="taskFiltered" :loading="loading" :columns="taskCols" empty-text="暂无任务，请点击「新建任务」">
          <el-table
            :data="taskFiltered"
            border
            stripe
            class="hub-table"
            empty-text="暂无任务"
            @row-click="(row: Row) => openTask(Number(row.id))"
          >
            <el-table-column prop="doc_no" label="单号" min-width="150">
              <template #default="{ row }">
                <div class="name-cell">
                  <span class="name">{{ row.doc_no || '—' }}</span>
                  <span v-if="row.id" class="id-hint">#{{ row.id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="plan_qty" label="计划" width="90" align="right" />
            <el-table-column prop="completed_qty" label="完工" width="90" align="right" />
            <el-table-column label="进度" min-width="160">
              <template #default="{ row }">
                <el-progress :percentage="Math.min(100, Number(row.progress_pct || 0))" :stroke-width="10" />
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" min-width="160" />
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openTask(Number(row.id)); dispatchForm.task_id=Number(row.id)">明细</el-button>
                <el-button v-if="row.status!=='closed'" link type="warning" @click.stop="closeTask(Number(row.id))">关闭</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
            <span class="hint">{{ Number(row.completed_qty || 0) }}/{{ Number(row.plan_qty || 0) }} · {{ Number(row.progress_pct || 0).toFixed(0) }}%</span>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openTask(Number(row.id)); dispatchForm.task_id=Number(row.id)">明细</el-button>
            <el-button v-if="row.status!=='closed'" link type="warning" @click="closeTask(Number(row.id))">关闭</el-button>
          </template>
        </TableOrCards>
        <el-dialog v-model="taskCreateDlg" title="新建生产任务" width="520px" destroy-on-close>
          <el-form label-width="80px">
            <el-form-item label="产品" required>
              <ProductSelect v-model="taskForm.product_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="计划量" required>
              <el-input-number v-model="taskForm.qty" :min="1" style="width:100%" />
            </el-form-item>
            <el-form-item label="工艺">
              <RoutingSelect v-model="taskForm.routing_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="车间">
              <WorkshopSelect v-model="taskForm.workshop_dept_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="taskForm.remark" placeholder="可选" maxlength="64" />
            </el-form-item>
          </el-form>
          <p class="form-tip">多商品任务请走「一单多商品」。日常过站请在 App 完成。</p>
          <template #footer>
            <el-button @click="taskCreateDlg = false">取消</el-button>
            <el-button type="primary" @click="createTask">创建</el-button>
          </template>
        </el-dialog>
      </template>

      <!-- 一单多商品：多行商品任务 -->
      <template v-else-if="active==='multi-products'">
        <p class="mode-hint">一张任务单挂多行商品；创建后可在明细中查看全部商品行。</p>
        <el-card header="新建多商品任务" class="mb">
          <el-form inline size="small" class="mb">
            <el-form-item label="工艺"><RoutingSelect v-model="taskForm.routing_id" /></el-form-item>
            <el-form-item label="车间"><WorkshopSelect v-model="taskForm.workshop_dept_id" /></el-form-item>
            <el-button @click="addMultiLine">加一行</el-button>
            <el-button type="primary" @click="createMultiTask">创建任务</el-button>
          </el-form>
          <div v-for="(line, idx) in multiLines" :key="idx" class="multi-line">
            <el-select v-model="line.product_id" style="width:180px" size="small">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
            <el-input-number v-model="line.qty" :min="1" size="small" />
            <el-button link type="danger" :disabled="multiLines.length<=1" @click="removeMultiLine(idx)">删除</el-button>
          </div>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="taskCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="160" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column prop="plan_qty" label="计划" width="90" />
            <el-table-column prop="completed_qty" label="完工" width="90" />
            <el-table-column label="进度" min-width="140">
              <template #default="{ row }">
                <el-progress
                  :percentage="Math.min(100, Number(row.progress_pct || 0))"
                  :stroke-width="10"
                />
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="160" />
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button link type="primary" @click="openTask(Number(row.id)); dispatchForm.task_id=Number(row.id)">明细</el-button>
                <el-button v-if="row.status!=='closed'" link type="warning" @click="closeTask(Number(row.id))">关闭</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
            <span class="hint">{{ Number(row.completed_qty || 0) }}/{{ Number(row.plan_qty || 0) }} · {{ Number(row.progress_pct || 0).toFixed(0) }}%</span>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openTask(Number(row.id)); dispatchForm.task_id=Number(row.id)">明细</el-button>
            <el-button v-if="row.status!=='closed'" link type="warning" @click="closeTask(Number(row.id))">关闭</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 派工 / 灵活 -->
      <template v-else-if="active==='dispatches' || active==='flex'">
        <header class="page-head">
          <div>
            <h2 class="title">{{ title }}</h2>
            <p class="desc">{{ active==='flex' ? '灵活改派工人到指定任务/工序，属例外通道。日常过站请走 App。' : `正常流转无需派工，App 扫工牌+${codeLabel}即可过站。此处仅用于例外补派。` }}</p>
          </div>
          <div class="head-meta">
            <span class="meta-pill">筛选 {{ dispatchFiltered.length }} / 全部 {{ list.length }}</span>
          </div>
        </header>
        <div class="shift-stats">
          <div class="stat"><div class="label">全部</div><div class="value">{{ dispatchStats.total }}</div></div>
          <div class="stat warn"><div class="label">已派发</div><div class="value">{{ dispatchStats.dispatched }}</div></div>
          <div class="stat"><div class="label">已改派</div><div class="value">{{ dispatchStats.reassigned }}</div></div>
          <div class="stat ok"><div class="label">已接收</div><div class="value">{{ dispatchStats.received }}</div></div>
        </div>
        <div class="row shift-toolbar">
          <el-button v-if="canCreateDispatch" type="primary" @click="openCreateDispatch">{{ active==='flex' ? '灵活派发' : '例外派岗' }}</el-button>
          <el-button @click="refresh">刷新</el-button>
          <el-select v-model="dispatchStatusFilter" clearable placeholder="状态" style="width:130px">
            <el-option label="已派发" value="dispatched" />
            <el-option label="已改派" value="reassigned" />
            <el-option label="已接收" value="received" />
          </el-select>
          <el-input v-model="dispatchKeyword" clearable placeholder="单号" style="width:180px" />
        </div>
        <TableOrCards :data="dispatchFiltered" :loading="loading" :columns="dispatchCols" empty-text="暂无派工单">
          <el-table :data="dispatchFiltered" border stripe class="hub-table" empty-text="暂无派工单">
            <el-table-column prop="doc_no" label="单号" min-width="150">
              <template #default="{ row }">
                <div class="name-cell">
                  <span class="name">{{ row.doc_no || '—' }}</span>
                  <span v-if="row.id" class="id-hint">#{{ row.id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="task_id" label="任务" width="90">
              <template #default="{ row }">
                <span :class="{ muted: !row.task_id }">{{ row.task_id || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="process_name" label="工序" min-width="110" />
            <el-table-column prop="worker_name" label="工人" min-width="110" />
            <el-table-column label="数量" width="90" align="right">
              <template #default="{ row }">{{ Number(row.qty ?? row.plan_qty ?? 0) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button v-if="row.status==='dispatched'" link type="primary" @click="receiveDispatch(Number(row.id))">接收</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='dispatched'" link type="primary" @click="receiveDispatch(Number(row.id))">接收</el-button>
          </template>
        </TableOrCards>
        <el-dialog v-model="dispatchCreateDlg" :title="active==='flex' ? '灵活派发' : '例外派岗'" width="520px" destroy-on-close>
          <el-form label-width="80px">
            <el-form-item label="任务" required>
              <ProdTaskSelect v-model="dispatchForm.task_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="工序" required>
              <ProcessSelect v-model="dispatchForm.process_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="工人" required>
              <EmployeeSelect v-model="dispatchForm.worker_id" style="width:100%" />
            </el-form-item>
            <el-form-item label="数量">
              <el-input-number v-model="dispatchForm.qty" :min="1" style="width:100%" />
            </el-form-item>
          </el-form>
          <p class="form-tip">仅例外场景使用。正常过站由工人在 App 扫工牌完成。</p>
          <template #footer>
            <el-button @click="dispatchCreateDlg = false">取消</el-button>
            <el-button type="primary" @click="createDispatch">派工</el-button>
          </template>
        </el-dialog>
      </template>

      <!-- 工序流水（领料/退库/入库事件） -->
      <template v-else-if="active==='reports' || active==='process-reports'">
        <el-alert type="info" show-icon :closable="false" class="mb"
          title="现场请用 App「生产」领料（须溯源已进入生产）。本页只读查询流水；金额相关以确认结束+日结为准。" />
        <el-card class="mb">
          <el-form inline size="small" class="mb">
            <el-form-item label="日期"><el-date-picker v-model="stationFlowFilter.biz_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
            <el-form-item :label="codeLabel"><el-input v-model="stationFlowFilter.board_code" clearable style="width:140px" /></el-form-item>
            <el-form-item label="类型"><EnumSelect v-model="stationFlowFilter.event_type" :options="STATION_FLOW_EVENT_OPTIONS" clearable style="width:140px" /></el-form-item>
            <el-form-item label="仅有金额"><el-switch v-model="stationFlowFilter.has_amount" /></el-form-item>
            <el-button type="primary" @click="refresh">查询</el-button>
          </el-form>
          <TableOrCards :data="stationFlowList" :loading="loading" :columns="stationFlowCols">
            <el-table :data="stationFlowList" size="small">
              <el-table-column prop="created_at" label="时间" min-width="160" />
              <el-table-column prop="event_type" label="类型" width="100" />
              <el-table-column prop="board_code" :label="codeLabel" width="120" />
              <el-table-column prop="process_name" label="工序" width="100" />
              <el-table-column prop="worker_name" label="工人" width="90" />
              <el-table-column prop="emp_type" label="工种" width="70" />
              <el-table-column prop="kg" label="kg" width="80" />
              <el-table-column prop="pay_mode" label="计费" width="70" />
              <el-table-column prop="rate" label="单价" width="70" />
              <el-table-column prop="amount" label="金额" width="90" />
              <el-table-column prop="remark" label="备注" min-width="120" />
            </el-table>
          </TableOrCards>
        </el-card>
      </template>

      <!-- 计件日结 -->
      <template v-else-if="active==='piecework'">
        <el-card class="mb">
          <el-form inline size="small">
            <el-form-item label="业务日"><el-date-picker v-model="pieceBizDate" type="date" value-format="YYYY-MM-DD" /></el-form-item>
            <el-button type="warning" size="small" @click="daySettlePiece">日结</el-button>
            <el-button size="small" @click="refresh">刷新</el-button>
            <el-form-item label="转账单号"><el-input v-model="payForm.transfer_no" /></el-form-item>
            <el-form-item label="回单"><el-input v-model="payForm.pay_evidence_url" /></el-form-item>
          </el-form>
          <p class="mode-hint">日结：仅纳入已「确认结束」(work_done) 的领料净占用；主任确认前不进财务汇总。入库过账与下道领取记工序产出，不即时结工资。</p>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="pieceworkCols">
          <el-table :data="list" size="small">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="worker_name" label="工人" width="100" />
            <el-table-column prop="process_name" label="工序" width="120" />
            <el-table-column prop="biz_date" label="日期" width="110" />
            <el-table-column prop="qty" label="产量" width="90" />
            <el-table-column prop="amount" label="金额" width="100" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status!=='paid'" link type="primary" @click="payPiece(Number(row.id))">支付</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status!=='paid'" link type="primary" @click="payPiece(Number(row.id))">支付</el-button>
          </template>
        </TableOrCards>

      </template>

      <!-- BOM -->
      <template v-else-if="active==='boms'">
        <el-card header="BOM" class="mb">
          <el-form inline size="small">
            <el-form-item label="成品">
              <el-select v-model="bomForm.product_id" style="width:160px">
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="组件"><ProductSelect v-model="bomForm.component_product_id" /></el-form-item>
            <el-form-item label="用量"><el-input-number v-model="bomForm.qty" :min="0.01" :step="0.1" /></el-form-item>
            <el-form-item label="损耗率"><el-input-number v-model="bomForm.scrap_rate" :min="0" :max="1" :step="0.01" /></el-form-item>
            <el-button type="primary" @click="createBom">新建</el-button>
            <el-button @click="genBom">自动生成</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="bomCols">
          <el-table :data="list" size="small">
            <el-table-column prop="code" label="编码" width="180" />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="product_id" label="成品" width="90" />
            <el-table-column prop="version_no" label="版本" width="80" />
            <el-table-column prop="status" label="状态" width="90" />
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
        </TableOrCards>
      </template>

      <!-- MRP -->
      <template v-else-if="active==='mrp'">
        <el-card class="mb">
          <el-button type="primary" @click="runMrp">运行 MRP</el-button>
          <span class="hint" style="margin-left:8px">按未完成任务需求 + BOM 展开，对比库存给出短缺建议。</span>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="mrpCols">
          <el-table :data="list" size="small">
            <el-table-column prop="run_no" label="运算号" width="160" />
            <el-table-column prop="run_at" label="时间" width="160" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link type="primary" @click="openMrp(Number(row.id))">结果</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openMrp(Number(row.id))">结果</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 领料 -->
      <template v-else-if="active==='requisitions'">
        <el-card header="联动领料" class="mb">
          <el-form inline size="small">
            <el-form-item label="物料"><ProductSelect v-model="reqForm.product_id" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="reqForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="仓库"><WarehouseSelect v-model="reqForm.warehouse_id" /></el-form-item>
            <el-button type="primary" @click="createReq">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="reqCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft' || row.status==='open'" link type="primary" @click="postReq(Number(row.id))">过账</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft' || row.status==='open'" link type="primary" @click="postReq(Number(row.id))">过账</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 工作台 -->
      <template v-else-if="active==='workbench'">
        <header class="page-head">
          <div>
            <h2 class="title">车间工作台</h2>
            <p class="desc">今日领料流水、领取未完、开工班次与在制任务一览。点行可打开任务明细。</p>
          </div>
          <div class="head-meta">
            <span class="meta-pill">在制任务 {{ workbenchDisplay.length }}</span>
          </div>
        </header>
        <div class="shift-stats">
          <div class="stat ok">
            <div class="label">今日领料流水</div>
            <div class="value">{{ overview?.today_station_passes ?? overview?.today_reports ?? 0 }}</div>
          </div>
          <div class="stat warn">
            <div class="label">领取未完</div>
            <div class="value">{{ overview?.pending_confirm ?? 0 }}</div>
          </div>
          <div class="stat">
            <div class="label">开工班次</div>
            <div class="value">{{ overview?.open_shifts ?? 0 }}</div>
          </div>
          <div class="stat">
            <div class="label">流转失败</div>
            <div class="value">{{ overview?.failed_flow_events ?? 0 }}</div>
          </div>
        </div>
        <p class="form-tip">例外派工 {{ overview?.exception_dispatches ?? overview?.open_dispatches ?? 0 }} 张 · 开立任务 {{ overview?.open_tasks ?? 0 }}</p>
        <div class="row shift-toolbar">
          <el-button @click="refresh">刷新</el-button>
        </div>
        <TableOrCards :data="workbenchDisplay" :loading="loading" :columns="workbenchCols" empty-text="暂无在制任务">
          <el-table
            :data="workbenchDisplay"
            border
            stripe
            class="hub-table"
            empty-text="暂无在制任务"
            @row-click="(row: Row) => openTask(Number(row.id))"
          >
            <el-table-column prop="doc_no" label="单号" min-width="160">
              <template #default="{ row }">
                <div class="name-cell">
                  <span class="name">{{ row.doc_no || '—' }}</span>
                  <span v-if="row.id" class="id-hint">#{{ row.id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建" min-width="160" />
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openTask(Number(row.id))">明细</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openTask(Number(row.id))">明细</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 工序在制 -->
      <template v-else-if="active==='process-wip'">
        <header class="page-head">
          <div>
            <h2 class="title">工序在制</h2>
            <p class="desc">按工艺步骤查看在制板与重量。领取 / 退库请在 App 完成。</p>
          </div>
          <div class="head-meta">
            <span class="meta-pill">工艺 {{ wipSummary?.routing_code || '—' }}</span>
          </div>
        </header>
        <div class="shift-stats cols-5">
          <div class="stat">
            <div class="label">在制板数</div>
            <div class="value">{{ wipSummary?.total_boards ?? wipSummary?.total_boxes ?? 0 }}</div>
          </div>
          <div class="stat">
            <div class="label">在制 kg</div>
            <div class="value">{{ Number(wipSummary?.total_weight || 0).toFixed(1) }}</div>
          </div>
          <div class="stat ok">
            <div class="label">在仓 kg</div>
            <div class="value">{{ Number(wipSummary?.total_stock_kg || 0).toFixed(1) }}</div>
          </div>
          <div class="stat warn">
            <div class="label">领取未完</div>
            <div class="value">{{ wipSummary?.open_worker_issues ?? wipSummary?.pending_confirm_reports ?? 0 }}</div>
          </div>
          <div class="stat">
            <div class="label">领取未完 kg</div>
            <div class="value">{{ Number(wipSummary?.open_worker_issue_kg ?? wipSummary?.pending_confirm_weight ?? 0).toFixed(1) }}</div>
          </div>
        </div>
        <div class="row shift-toolbar">
          <ProductSelect v-model="wipProductId" clearable placeholder="按产品筛选" style="width:240px" />
          <el-button @click="refresh">刷新</el-button>
        </div>
        <p v-if="wipSummary?.unassigned" class="form-tip">
          未挂工序板 {{ (wipSummary.unassigned as Row).board_count || (wipSummary.unassigned as Row).box_count || 0 }}
          · 重量 {{ Number((wipSummary.unassigned as Row).wip_weight || 0).toFixed(1) }} kg
          <el-button
            v-if="Number((wipSummary.unassigned as Row).board_count || (wipSummary.unassigned as Row).box_count || 0) > 0"
            link
            type="primary"
            @click="openWipBoxes(0, '未挂工序', true)"
          >查看</el-button>
        </p>
        <TableOrCards :data="list" :loading="loading" :columns="wipCols" empty-text="暂无在制步骤">
          <el-table :data="list" border stripe class="hub-table" empty-text="暂无在制步骤" @row-click="(row: Row) => openWipBoxes(Number(row.step_id), String(row.step_name || ''), false)">
            <el-table-column prop="seq_no" label="序" width="60" align="center" />
            <el-table-column prop="step_code" label="步骤码" width="100" />
            <el-table-column prop="step_name" label="步骤" min-width="140">
              <template #default="{ row }">
                <span class="name">{{ row.step_name || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="process_name" label="工序" width="120" />
            <el-table-column label="板数" width="80" align="center">
              <template #default="{ row }">{{ row.board_count ?? row.box_count ?? 0 }}</template>
            </el-table-column>
            <el-table-column label="可领 kg" width="100" align="right">
              <template #default="{ row }">{{ Number(row.available_kg || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="领取未完 kg" width="120" align="right">
              <template #default="{ row }">{{ Number(row.occupied_kg || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="在制 kg" width="110" align="right">
              <template #default="{ row }">{{ Number(row.wip_weight || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="在仓 kg" width="100" align="right">
              <template #default="{ row }">{{ Number(row.stock_kg || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="在仓板数" width="90" align="center">
              <template #default="{ row }">{{ row.stock_box_count ?? 0 }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openWipBoxes(Number(row.step_id), String(row.step_name || ''), false)">板明细</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openWipBoxes(Number(row.step_id), String(row.step_name || ''), false)">板明细</el-button>
          </template>
        </TableOrCards>
        <el-drawer v-model="wipDrawer" :title="wipDrawerTitle" size="480px">
          <TableOrCards :data="wipBoxes" :columns="wipBoxCols" empty-text="该步骤暂无板">
            <el-table :data="wipBoxes" border stripe size="small" class="hub-table" empty-text="该步骤暂无板">
              <el-table-column prop="code" label="板码" min-width="140" />
              <el-table-column prop="product_name" label="产品" width="100" />
              <el-table-column label="可领 kg" width="90" align="right">
                <template #default="{ row }">{{ Number(row.available_kg ?? row.weight ?? 0).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column label="领取未完 kg" width="110" align="right">
                <template #default="{ row }">{{ Number(row.occupied_kg || 0).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column prop="trace_code" label="溯源" width="110" />
              <el-table-column label="状态" width="90" align="center">
                <template #default="{ row }">
                  <el-tag v-if="row.status != null" size="small" :type="statusTagType(row.status)">{{ docStatusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="工人占用" min-width="160">
                <template #default="{ row }">
                  <div v-if="Array.isArray(row.occupancies) && row.occupancies.length">
                    <div v-for="(o, i) in row.occupancies as Row[]" :key="i">
                      {{ o.worker_name || o.worker_id }} · {{ Number(o.open_kg || 0).toFixed(2) }} kg
                    </div>
                  </div>
                  <span v-else class="muted">—</span>
                </template>
              </el-table-column>
            </el-table>
            <template #extra="{ row }">
              <el-tag v-if="row.status != null" size="small" :type="statusTagType(row.status)">{{ docStatusLabel(row.status) }}</el-tag>
            </template>
          </TableOrCards>
        </el-drawer>
      </template>

      <!-- 溯源生产台 -->
      <template v-else-if="active==='trace-production'">
        <div class="page-head mb">
          <div>
            <h2 class="title">溯源生产</h2>
            <p class="desc">列出全部可用溯源码；点击行查看工艺工序时间线。生管可结束本工序；全部工序结束后自动结案并生成生产报表。</p>
          </div>
        </div>
        <div class="row mb">
          <el-select v-model="traceProdFilter" clearable placeholder="状态" style="width:140px" @change="refresh">
            <el-option label="全部" value="" />
            <el-option label="库中" value="in_stock" />
            <el-option label="生产中" value="in_production" />
            <el-option label="已结束" value="ended" />
          </el-select>
          <el-button @click="refresh">刷新</el-button>
          <el-input v-model="traceProdCode" clearable placeholder="溯源码详情" style="width:180px" />
          <el-button type="primary" @click="loadTraceProdWip">工序分布</el-button>
        </div>
        <TableOrCards :data="list" :loading="loading" :columns="[
          { prop: 'trace_code', label: '溯源', primary: true },
          { prop: 'ui_status', label: '状态' },
          { prop: 'stock_kg', label: '库存kg' },
          { prop: 'board_count', label: '板数' },
        ]">
          <el-table :data="list" size="small" border stripe @row-click="(row: Row) => { traceProdCode = String(row.trace_code||''); loadTraceProdWip() }">
            <el-table-column prop="trace_code" label="溯源" min-width="140" />
            <el-table-column prop="ui_status" label="状态" width="100">
              <template #default="{ row }">
                {{ row.ui_status === 'in_production' ? '生产中' : row.ui_status === 'ended' ? '已结束' : '库中' }}
              </template>
            </el-table-column>
            <el-table-column prop="stock_kg" label="库存 kg" width="100" />
            <el-table-column prop="board_count" label="板数" width="80" />
            <el-table-column prop="session_status" label="会话" width="100" />
          </el-table>
        </TableOrCards>
        <el-card v-if="traceProdWip" v-loading="traceProdBusy" class="mt mb">
          <template #header>
            <div class="row" style="justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
              <span>
                工序时间线 · {{ traceProdWip.trace_code }} ·
                {{ traceProdWip.ui_status === 'in_production' ? '生产中' : traceProdWip.ui_status === 'ended' ? '已结束' : '库中' }}
                <span v-if="traceProdWip.product_name" class="hint"> · {{ traceProdWip.product_name }}</span>
                <span v-if="traceProdWip.routing_code || traceProdWip.routing_name" class="hint">
                  · 工艺 {{ traceProdWip.routing_code }}<template v-if="traceProdWip.routing_name"> · {{ traceProdWip.routing_name }}</template>
                </span>
              </span>
              <span>
                <el-button
                  v-if="traceProdWip.ui_status === 'in_stock'"
                  type="primary"
                  size="small"
                  :loading="traceProdBusy"
                  @click="startTraceProduction"
                >进入生产</el-button>
                <el-tag v-else-if="traceProdWip.ui_status === 'in_production'" type="warning" size="small">生产中</el-tag>
                <el-tag v-else-if="traceProdWip.ui_status === 'ended'" type="success" size="small">已结案</el-tag>
              </span>
            </div>
          </template>
          <p class="form-tip">
            可领 {{ Number(traceProdWip.total_available_kg||0).toFixed(2) }} · 占用 {{ Number(traceProdWip.total_occupied_kg||0).toFixed(2) }}
            <span v-if="traceProdTimelineSteps().length"> · 全部工序结束后自动结案</span>
            <span v-else> · 未配置工艺路线，请先在「工艺流程」绑定产品工艺</span>
          </p>
          <el-alert
            v-if="traceProdWip.ui_status === 'in_stock'"
            type="info"
            :closable="false"
            show-icon
            class="mb"
            title="请先点击「进入生产」，再按工序顺序结束各工序"
          />
          <el-table :data="traceProdTimelineSteps()" size="small" border>
            <el-table-column prop="seq_no" label="#" width="50" />
            <el-table-column prop="process_name" label="工序" min-width="120" />
            <el-table-column prop="input_product_name" label="领取产物" min-width="110" />
            <el-table-column prop="output_product_name" label="产出产物" min-width="110" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag v-if="row.step_status" size="small" :type="traceStepStatusType(row.step_status)">{{ traceStepStatusLabel(row.step_status) }}</el-tag>
                <span v-else>—</span>
              </template>
            </el-table-column>
            <el-table-column label="WIP kg" width="90">
              <template #default="{ row }">{{ Number(row.wip_kg ?? row.available_kg ?? 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="投料" width="80">
              <template #default="{ row }">{{ row.input_kg != null ? Number(row.input_kg).toFixed(2) : '—' }}</template>
            </el-table-column>
            <el-table-column label="产出" width="80">
              <template #default="{ row }">{{ row.output_kg != null ? Number(row.output_kg).toFixed(2) : '—' }}</template>
            </el-table-column>
            <el-table-column label="扣损" width="80">
              <template #default="{ row }">{{ row.loss_kg != null ? Number(row.loss_kg).toFixed(2) : '—' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button
                  v-if="canCompleteTraceStep(row)"
                  type="primary" link size="small"
                  @click="completeTraceProcessStep(Number(row.process_id))"
                >结束本工序</el-button>
                <el-tooltip
                  v-else-if="(row.step_status === 'ready' || row.action === 'complete') && traceProdWip.ui_status !== 'in_production'"
                  content="请先点击「进入生产」"
                >
                  <el-button type="primary" link size="small" disabled>结束本工序</el-button>
                </el-tooltip>
                <span v-else-if="row.step_status === 'done' || row.action === 'done'" class="muted">已完成</span>
                <span v-else-if="row.step_status === 'in_progress' || row.action === 'in_progress'" class="hint">{{ row.action_hint || '在制中' }}</span>
                <span v-else class="muted">{{ row.action_hint || '待做' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card v-if="traceProdReport" class="mt mb">
          <template #header>溯源生产报表 · {{ traceProdReport.trace_code }}</template>
          <p class="form-tip">
            溯源投入 {{ Number((traceProdReport.summary as Row)?.trace_input_kg || 0).toFixed(2) }} kg ·
            产出 {{ Number((traceProdReport.summary as Row)?.trace_output_kg || 0).toFixed(2) }} kg ·
            耗损率 {{ (Number((traceProdReport.summary as Row)?.trace_loss_rate || 0) * 100).toFixed(1) }}%
          </p>
          <el-tabs>
            <el-tab-pane label="工序扣损">
              <el-table :data="(traceProdReport.process_yields as Row[]) || []" size="small" border>
                <el-table-column prop="process_name" label="工序" />
                <el-table-column label="投料 kg"><template #default="{ row }">{{ Number(row.input_kg||0).toFixed(2) }}</template></el-table-column>
                <el-table-column label="产出 kg"><template #default="{ row }">{{ Number(row.output_kg||0).toFixed(2) }}</template></el-table-column>
                <el-table-column label="扣损 kg"><template #default="{ row }">{{ Number(row.loss_kg||0).toFixed(2) }}</template></el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="领料摘要">
              <el-table :data="(traceProdReport.issues as Row[]) || []" size="small" border max-height="320">
                <el-table-column prop="board_code" label="板码" width="120" />
                <el-table-column prop="process_name" label="工序" />
                <el-table-column prop="issue_kg" label="领取 kg" width="90" />
                <el-table-column prop="status" label="状态" width="80" />
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="板明细">
              <el-table :data="(traceProdReport.boards as Row[]) || []" size="small" border>
                <el-table-column prop="code" label="板码" />
                <el-table-column prop="process_name" label="工序" />
                <el-table-column prop="weight_kg" label="kg" width="90" />
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="日志">
              <el-table :data="(traceProdReport.logs as Row[]) || []" size="small" border max-height="320">
                <el-table-column prop="event_type" label="事件" width="140" />
                <el-table-column prop="process_name" label="工序" />
                <el-table-column prop="created_at" label="时间" min-width="160" />
              </el-table>
            </el-tab-pane>
          </el-tabs>
        </el-card>
        <el-dialog v-model="traceStartDialogVisible" title="选择工艺流程" width="520px" destroy-on-close>
          <p class="form-tip mb">
            溯源码 <strong>{{ traceProdCode }}</strong>
            <span v-if="traceStartProductName"> · 原料 {{ traceStartProductName }}</span>
          </p>
          <el-form label-width="88px">
            <el-form-item label="工艺流程" required>
              <el-select v-model="traceStartSelectedRoutingId" placeholder="选择工艺" style="width:100%">
                <el-option
                  v-for="opt in traceStartOptions"
                  :key="Number(opt.id)"
                  :label="`${opt.code} · ${opt.name}（${opt.step_count} 道工序）`"
                  :value="Number(opt.id)"
                />
              </el-select>
            </el-form-item>
            <el-form-item v-if="traceStartPreviewSteps.length" label="工序预览">
              <el-tag v-for="(s, i) in traceStartPreviewSteps" :key="i" size="small" class="mr" style="margin:2px">
                {{ i + 1 }}. {{ s }}
              </el-tag>
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="traceStartDialogVisible = false">取消</el-button>
            <el-button type="primary" :loading="traceProdBusy" @click="confirmTraceStart">确认进入生产</el-button>
          </template>
        </el-dialog>
      </template>
      <template v-else-if="active==='process-yield'">
        <div class="row mb">
          <el-input v-model="yieldTraceCode" clearable placeholder="溯源码" style="width:180px" @keyup.enter="refresh" />
          <el-button @click="refresh">查询</el-button>
          <span class="hint">投入=领取净量；产出=下道移转+入库。逐工序结束或自动结案后可见扣损。</span>
        </div>
        <TableOrCards :data="list" :loading="loading" :columns="yieldTraceCols">
          <el-table :data="list" size="small" border stripe>
            <el-table-column prop="process_name" label="工序" width="120" />
            <el-table-column prop="trace_code" label="溯源" min-width="120" />
            <el-table-column prop="board_count" label="板数" width="80" />
            <el-table-column label="投料 kg" width="100">
              <template #default="{ row }">{{ Number(row.input_kg || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="完工 kg" width="100">
              <template #default="{ row }">{{ Number(row.output_kg || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="扣损 kg" width="100">
              <template #default="{ row }">{{ Number(row.loss_kg || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="扣损率" width="90">
              <template #default="{ row }">{{ (Number(row.loss_rate || 0) * 100).toFixed(1) }}%</template>
            </el-table-column>
            <el-table-column prop="created_at" label="计算时间" min-width="160" />
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small">{{ ((Number(row.loss_rate || 0) * 100).toFixed(1)) }}%</el-tag>
          </template>
        </TableOrCards>
      </template>

      <!-- 进度 -->
      <template v-else-if="active==='progress'">
        <TableOrCards :data="list" :loading="loading" :columns="progressCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="任务单" width="160" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column prop="plan_qty" label="计划" width="100" />
            <el-table-column prop="completed_qty" label="完成" width="100" />
            <el-table-column prop="progress_pct" label="进度%" width="100" />
            <el-table-column prop="created_at" label="创建" />
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
        </TableOrCards>
      </template>

      <!-- 多单整合 -->
      <template v-else-if="active==='merges'">
        <el-card header="多单整合" class="mb">
          <el-form inline size="small">
            <el-form-item label="标题"><el-input v-model="mergeForm.title" /></el-form-item>
            <el-form-item label="任务"><ProdTaskSelect v-model="mergeForm.task_ids" multiple style="width:280px" /></el-form-item>
            <el-button type="primary" @click="createMerge">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="mergeCols">
          <el-table :data="list" size="small">
            <el-table-column prop="merge_no" label="整合号" width="150" />
            <el-table-column prop="title" label="标题" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column prop="result_task_id" label="结果任务" width="100" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="primary" @click="confirmMerge(Number(row.id))">确认整合</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="confirmMerge(Number(row.id))">确认整合</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 图纸 -->
      <template v-else-if="active==='drawings'">
        <el-card header="图纸分发挂接" class="mb">
          <el-form inline size="small">
            <el-form-item label="图纸编码"><el-input v-model="drawingForm.drawing_code" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="drawingForm.drawing_name" /></el-form-item>
            <el-form-item label="任务"><ProdTaskSelect v-model="drawingForm.task_id" clearable /></el-form-item>
            <el-form-item label="文件URL"><el-input v-model="drawingForm.file_url" style="width:200px" /></el-form-item>
            <el-button type="primary" @click="createDrawing">挂接</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="drawingCols">
          <el-table :data="list" size="small">
            <el-table-column prop="drawing_code" label="编码" width="120" />
            <el-table-column prop="drawing_name" label="名称" />
            <el-table-column prop="task_id" label="任务" width="80" />
            <el-table-column prop="file_url" label="文件" min-width="160" show-overflow-tooltip />
          </el-table>
        </TableOrCards>
      </template>

      <!-- 质检 -->
      <template v-else-if="active==='qc'">
        <el-card header="质检单" class="mb">
          <el-form inline size="small">
            <el-form-item label="类型"><EnumSelect v-model="qcForm.qc_type" :options="QC_TYPE_OPTIONS" style="width:140px" /></el-form-item>
            <el-form-item label="产品"><ProductSelect v-model="qcForm.product_id" /></el-form-item>
            <el-form-item label="工序">
              <el-select v-model="qcForm.process_id" style="width:140px">
                <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="qcForm.qty" :min="0" /></el-form-item>
            <el-button type="primary" @click="createQc">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="qcCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="qc_type" label="类型" width="100" />
            <el-table-column prop="result" label="结果" width="90" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="success" @click="completeQc(Number(row.id))">完成</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="success" @click="completeQc(Number(row.id))">完成</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 返修 -->
      <template v-else-if="active==='reworks'">
        <el-card header="返修单" class="mb">
          <el-form inline size="small">
            <el-form-item label="工序">
              <el-select v-model="reworkForm.process_id" style="width:140px">
                <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="reworkForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="备注"><el-input v-model="reworkForm.remark" /></el-form-item>
            <el-button type="primary" @click="createRework">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="reworkCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="process_id" label="工序" width="80" />
            <el-table-column prop="qty" label="数量" width="90" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status!=='closed'" link @click="closeRework(Number(row.id))">关闭</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status!=='closed'" link @click="closeRework(Number(row.id))">关闭</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 废料 -->
      <template v-else-if="active==='scraps'">
        <el-card header="废料登记" class="mb">
          <el-form inline size="small">
            <el-form-item label="料号"><ProductSelect v-model="scrapForm.product_id" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="scrapForm.qty" :min="0" /></el-form-item>
            <el-form-item label="类型">
              <el-select v-model="scrapForm.scrap_type" style="width:140px">
                <el-option label="切断次品" value="cut_defect" />
                <el-option label="去芯次品" value="core_defect" />
                <el-option label="切块次品" value="dice_defect" />
                <el-option label="筛选装袋次品" value="sieve_bag_defect" />
              </el-select>
            </el-form-item>
            <el-button type="primary" @click="createScrap">登记</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="scrapCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="product_id" label="料号" width="80" />
            <el-table-column prop="qty" label="数量" width="90" />
            <el-table-column prop="scrap_type" label="类型" width="120" />
            <el-table-column prop="status" label="状态" width="90" />
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
        </TableOrCards>
      </template>

      <!-- 退库：领出未用完还仓（不回冲已确认计件） -->
      <template v-else-if="active==='process-returns'">
        <el-alert
          class="mb"
          type="info"
          :closable="false"
          title="退库 = 已领出未用完料还仓防丢失；班组长审批 → 仓管确认。不抹掉当日已确认完工计件。"
        />
        <el-card header="申请退未用完料" class="mb">
          <el-form inline size="small">
            <el-form-item :label="codeLabel"><el-input v-model="returnForm.box_code" style="width:160px" /></el-form-item>
            <el-form-item label="退回kg"><el-input-number v-model="returnForm.return_weight" :min="0.01" :step="1" /></el-form-item>
            <el-form-item label="仓库"><WarehouseSelect v-model="returnForm.warehouse_id" /></el-form-item>
            <el-form-item label="原因"><el-input v-model="returnForm.reason" style="width:120px" /></el-form-item>
            <el-button type="primary" @click="createProcessReturn">新建</el-button>
          </el-form>
        </el-card>
        <div class="row mb">
          <el-select v-model="returnStatusFilter" clearable placeholder="状态" style="width:160px" @change="refresh">
            <el-option label="草稿" value="draft" />
            <el-option label="待班组" value="pending_foreman" />
            <el-option label="待仓管" value="pending_warehouse" />
            <el-option label="已过账" value="posted" />
            <el-option label="已驳回" value="rejected" />
          </el-select>
          <el-input-number v-model="transferUserId" :min="1" placeholder="转交用户ID" controls-position="right" style="width:140px;margin-left:8px" />
          <el-button @click="refresh">刷新</el-button>
        </div>
        <TableOrCards :data="list" :loading="loading" :columns="processReturnCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="box_code" :label="codeLabel" width="130" />
            <el-table-column prop="return_weight" label="退回kg" width="90" />
            <el-table-column prop="warehouse_id" label="仓" width="60" />
            <el-table-column prop="reason" label="原因" min-width="100" />
            <el-table-column prop="status" label="状态" width="130" />
            <el-table-column label="操作" width="280" fixed="right">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="primary" @click="actProcessReturn(Number(row.id),'submit')">提交</el-button>
                <el-button v-if="row.status==='pending_foreman'" link type="primary" @click="actProcessReturn(Number(row.id),'approve')">班组通过</el-button>
                <el-button v-if="row.status==='pending_warehouse'" link type="success" @click="actProcessReturn(Number(row.id),'warehouse-confirm')">仓管确认</el-button>
                <el-button v-if="row.status==='pending_foreman' || row.status==='pending_warehouse'" link @click="actProcessReturn(Number(row.id),'transfer')">转交</el-button>
                <el-button v-if="row.status==='pending_foreman' || row.status==='pending_warehouse'" link type="danger" @click="actProcessReturn(Number(row.id),'reject')">驳回</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small">{{ row.status }}</el-tag>
            <span class="hint">{{ row.return_weight }}kg · {{ row.box_code }}</span>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="actProcessReturn(Number(row.id),'submit')">提交</el-button>
            <el-button v-if="row.status==='pending_foreman'" link type="primary" @click="actProcessReturn(Number(row.id),'approve')">班组通过</el-button>
            <el-button v-if="row.status==='pending_warehouse'" link type="success" @click="actProcessReturn(Number(row.id),'warehouse-confirm')">仓管确认</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 委外 -->
      <template v-else-if="active==='outsources'">
        <el-card header="委外加工" class="mb">
          <el-form inline size="small">
            <el-form-item label="供应商"><SupplierSelect v-model="outForm.supplier_id" /></el-form-item>
            <el-form-item label="工序"><ProcessSelect v-model="outForm.process_id" /></el-form-item>
            <el-form-item label="产品"><ProductSelect v-model="outForm.product_id" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="outForm.qty" :min="0.01" /></el-form-item>
            <el-button type="primary" @click="createOut">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="outsourceCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="supplier_id" label="供应商" width="90" />
            <el-table-column prop="qty" label="数量" width="90" />
            <el-table-column prop="received_qty" label="收回" width="90" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status!=='received'" link type="primary" @click="receiveOut(Number(row.id))">收回</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status!=='received'" link type="primary" @click="receiveOut(Number(row.id))">收回</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 受托 -->
      <template v-else-if="active==='consignments'">
        <el-card header="受托加工" class="mb">
          <el-form inline size="small">
            <el-form-item label="客户"><CustomerSelect v-model="consForm.customer_id" /></el-form-item>
            <el-form-item label="产品"><ProductSelect v-model="consForm.product_id" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="consForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="进度"><EnumSelect v-model="consForm.progress" :options="CONSIGNMENT_PROGRESS_OPTIONS" style="width:140px" /></el-form-item>
            <el-button type="primary" @click="createCons">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="consignmentCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="customer_id" label="客户" width="80" />
            <el-table-column prop="qty" label="数量" width="90" />
            <el-table-column prop="progress" label="进度" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button link type="primary" @click="updateConsProgress(Number(row.id))">更新进度</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="updateConsProgress(Number(row.id))">更新进度</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 成本隐藏 -->
      <template v-else-if="active==='cost-hide'">
        <el-card header="成本隐藏策略" class="mb">
          <el-form inline size="small">
            <el-form-item label="角色"><RoleSelect v-model="hideForm.role_id" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="hideForm.name" /></el-form-item>
            <el-button type="primary" @click="createHide">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="costHideCols">
          <el-table :data="list" size="small">
            <el-table-column prop="role_id" label="角色" width="90" />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="field_scope" label="字段范围" min-width="200" show-overflow-tooltip />
            <el-table-column prop="is_enabled" label="启用" width="80" />
          </el-table>
        </TableOrCards>
      </template>

      <el-drawer
        v-model="taskDetailDrawer"
        :title="`任务明细 · ${detail?.doc_no || ''}`"
        size="560px"
        destroy-on-close
        @close="closeTaskDetail"
      >
        <template v-if="isTaskDetail && detail">
          <el-row :gutter="12" class="mb">
            <el-col :span="8" :xs="24">
              <div class="kpi-card">
                <div class="kpi">计划量</div>
                <div class="kpi-n">{{ taskPlanTotal }}</div>
              </div>
            </el-col>
            <el-col :span="8" :xs="24">
              <div class="kpi-card">
                <div class="kpi">已完工</div>
                <div class="kpi-n">{{ taskDoneTotal }}</div>
              </div>
            </el-col>
            <el-col :span="8" :xs="24">
              <div class="kpi-card">
                <div class="kpi">完成率</div>
                <div class="kpi-n">{{ taskProgressPct }}%</div>
              </div>
            </el-col>
          </el-row>
          <div class="mb">
            <el-progress :percentage="taskProgressPct" :stroke-width="16" />
          </div>
          <div class="meta-row mb">
            <el-tag :type="statusTagType(detail.status)" size="small">{{ docStatusLabel(detail.status) }}</el-tag>
            <span class="hint">创建 {{ detail.created_at || '-' }}</span>
            <span v-if="detail.routing_id" class="hint">工艺 #{{ detail.routing_id }}</span>
            <span v-if="detail.workshop_dept_id" class="hint">车间 #{{ detail.workshop_dept_id }}</span>
          </div>

          <h4 class="sec-title">商品行</h4>
          <div v-if="!taskItems.length" class="hint mb">暂无商品行</div>
          <div v-for="it in taskItems" :key="String(it.id)" class="item-card">
            <div class="item-head">
              <strong>{{ it.product_name || productLabel(Number(it.product_id)) }}</strong>
              <span class="hint">#{{ it.product_id }} {{ it.product_code || '' }}</span>
            </div>
            <div class="item-nums">
              <span>计划 {{ Number(it.plan_qty ?? it.qty ?? 0) }}</span>
              <span>完工 {{ Number(it.completed_qty ?? 0) }}</span>
              <span>{{ itemProgress(it) }}%</span>
            </div>
            <el-progress :percentage="itemProgress(it)" :stroke-width="10" />
          </div>

          <h4 v-if="taskFlowSteps.length" class="sec-title">工艺路径</h4>
          <div v-if="taskFlowSteps.length" class="flow-track">
            <div
              v-for="(st, idx) in taskFlowSteps"
              :key="String(st.id || idx)"
              class="flow-node"
              :class="{
                piece: st.is_piecework,
                checkpoint: st.is_inbound_checkpoint,
              }"
            >
              <div class="flow-seq">{{ st.seq_no ?? idx + 1 }}</div>
              <div class="flow-name">{{ st.step_name || st.step_code || `步骤${idx + 1}` }}</div>
              <div class="flow-flags">
                <el-tag v-if="st.is_piecework" size="small" type="success">计件</el-tag>
                <el-tag v-if="st.is_inbound_checkpoint" size="small" type="warning">卡点</el-tag>
                <el-tag v-if="st.checkpoint_bind_warehouse" size="small">绑仓</el-tag>
              </div>
              <div v-if="idx < taskFlowSteps.length - 1" class="flow-arrow" aria-hidden="true">→</div>
            </div>
          </div>
        </template>
      </el-drawer>

      <el-card v-if="detail && !isTaskDetail" header="明细" style="margin-top:16px">
        <pre class="detail">{{ JSON.stringify(detail, null, 2) }}</pre>
      </el-card>

      <el-dialog v-model="processDlg" title="新增工序" width="520px" destroy-on-close>
        <el-form label-width="90px">
          <el-form-item label="编码"><el-input v-model="processForm.code" placeholder="可空，自动生成" /></el-form-item>
          <el-form-item label="名称" required><el-input v-model="processForm.name" placeholder="如：去皮、切断" /></el-form-item>
          <el-form-item label="类型"><EnumSelect v-model="processForm.process_type" :options="PROCESS_TYPE_OPTIONS" style="width:100%" /></el-form-item>
          <el-form-item label="计费">
            <EnumSelect v-model="processForm.pay_mode" :options="PROCESS_PAY_MODE_OPTIONS" style="width:100%" />
            <p class="hint" style="margin:6px 0 0">不计费 / 按重量 / 按件；按件与按重量目前均按 kg×工价核算</p>
          </el-form-item>
          <el-form-item label="状态"><EnumSelect v-model="processForm.status" :options="STATUS_ACTIVE_OPTIONS" style="width:100%" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="processDlg = false">取消</el-button>
          <el-button type="primary" @click="createProcess">保存</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="processEditDlg" title="配置工序" width="520px" destroy-on-close>
        <el-form label-width="90px">
          <el-form-item label="编码"><el-input v-model="processEditForm.code" /></el-form-item>
          <el-form-item label="名称" required><el-input v-model="processEditForm.name" /></el-form-item>
          <el-form-item label="类型"><EnumSelect v-model="processEditForm.process_type" :options="PROCESS_TYPE_OPTIONS" style="width:100%" /></el-form-item>
          <el-form-item label="计费">
            <EnumSelect v-model="processEditForm.pay_mode" :options="PROCESS_PAY_MODE_OPTIONS" style="width:100%" />
            <p class="hint" style="margin:6px 0 0">不计费 / 按重量 / 按件</p>
          </el-form-item>
          <el-form-item label="状态"><EnumSelect v-model="processEditForm.status" :options="STATUS_ACTIVE_OPTIONS" style="width:100%" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="processEditDlg = false">取消</el-button>
          <el-button type="primary" @click="saveEditProcess">保存</el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.head h2 { margin: 0 0 4px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
.mode-hint { color: #714b67; font-size: 13px; margin: 0 0 12px; background: #f5eef8; padding: 8px 12px; border-radius: 6px; }
.mb { margin-bottom: 12px; }
.row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.multi-line { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.detail { background: #f6f8fa; padding: 12px; border-radius: 8px; max-height: 420px; overflow: auto; font-size: 12px; }
.kpi { color: #667; font-size: 12px; }
.kpi-n { font-size: 28px; font-weight: 600; margin-top: 4px; }
.kpi-card {
  background: #f5f8fa;
  border-radius: 10px;
  padding: 12px 14px;
  margin-bottom: 8px;
}
.meta-row { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.sec-title { margin: 16px 0 10px; font-size: 14px; color: #334; }
.item-card {
  border: 1px solid #e6ebef;
  border-radius: 10px;
  padding: 12px;
  margin-bottom: 10px;
  background: #fff;
}
.item-head { display: flex; justify-content: space-between; gap: 8px; margin-bottom: 6px; }
.item-nums { display: flex; gap: 16px; font-size: 13px; color: #556; margin-bottom: 8px; }
.flow-track {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: stretch;
}
.flow-node {
  position: relative;
  min-width: 100px;
  max-width: 140px;
  flex: 1 1 100px;
  background: #f7fafb;
  border: 1px solid #dde5ea;
  border-radius: 10px;
  padding: 10px 10px 12px;
}
.flow-node.piece { border-color: #9fd4b3; background: #f3fbf6; }
.flow-node.checkpoint { border-color: #e6c56a; background: #fffaf0; }
.flow-seq {
  font-size: 11px;
  color: #889;
  margin-bottom: 4px;
}
.flow-name { font-weight: 600; font-size: 13px; line-height: 1.3; min-height: 2.4em; }
.flow-flags { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 6px; }
.flow-arrow {
  position: absolute;
  right: -10px;
  top: 40%;
  color: #99a;
  font-size: 14px;
  z-index: 1;
}
@media (max-width: 640px) {
  .flow-arrow { display: none; }
}
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 4px; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; color: #1f2a33; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; line-height: 1.5; max-width: 640px; }
.head-meta { flex-shrink: 0; padding-top: 2px; }
.meta-pill {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 999px;
  background: #eef6f1;
  color: #2f6b4f;
  font-size: 12px;
  font-weight: 500;
}
.shift-stats { display: grid; grid-template-columns: repeat(4, minmax(96px, 1fr)); gap: 10px; margin-bottom: 14px; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat.ok { background: #eef6f1; border-color: #d5eade; }
.stat.warn { background: #fff7f0; border-color: #f0e0d0; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; color: #1f2a33; }
.shift-toolbar { margin-bottom: 14px; }
.name-cell { display: flex; align-items: baseline; gap: 8px; }
.name { font-weight: 500; color: #1f2a33; }
.id-hint { font-size: 12px; color: #98a2a8; }
.muted { color: #98a2a8; }
.shift-table :deep(.el-table__header th) { background: #f6f8fa; color: #4a5a66; font-weight: 600; }
.shift-table :deep(.is-current-shift) { background: #eef6f1 !important; }
.hub-table :deep(.el-table__header th) { background: #f6f8fa; color: #4a5a66; font-weight: 600; }
.shift-stats.cols-5 { grid-template-columns: repeat(5, minmax(96px, 1fr)); }
.form-tip { margin: 0 0 12px; font-size: 13px; color: #5c6b75; background: #f6f8fa; padding: 8px 10px; border-radius: 6px; }
.shift-meta { display: flex; flex-wrap: wrap; gap: 12px; font-size: 13px; color: #5c6b75; margin-bottom: 14px; }
.shift-drawer-head { display: flex; align-items: center; gap: 8px; }
.shift-member-bar { margin-bottom: 12px; }
@media (max-width: 720px) {
  .shift-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .shift-stats.cols-5 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
