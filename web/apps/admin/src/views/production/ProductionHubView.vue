<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  productionApi,
  productApi,
  hrApi,
  fieldLedgerApi,
  inventoryApi,
  PROCESS_TYPE_OPTIONS,
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
  RoleSelect,
  WarehouseSelect,
  EnumSelect,
} from '../../components/select'
import RoutingView from '../automation/RoutingView.vue'
import PieceIssueView from './PieceIssueView.vue'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const processCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'process_type', label: '类型' },
  { prop: 'is_piecework', label: '计件' },
  { prop: 'status', label: '状态' },
]
const shiftCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '班次号', primary: true },
  { prop: 'biz_date', label: '日期' },
  { prop: 'workshop_name', label: '车间' },
  { prop: 'member_count', label: '人数' },
  { prop: 'status', label: '状态' },
]
const shiftMemberCols: MobileCardColumn[] = [
  { prop: 'employee_name', label: '员工', primary: true },
  { prop: 'process_name', label: '工序' },
]
const taskCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'status', label: '状态' },
  { prop: 'plan_qty', label: '计划' },
  { prop: 'completed_qty', label: '完工' },
  { prop: 'progress_pct', label: '进度%' },
  { prop: 'created_at', label: '创建时间' },
]
const dispatchCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'task_id', label: '任务' },
  { prop: 'process_id', label: '工序' },
  { prop: 'worker_id', label: '工人' },
  { prop: 'qty', label: '数量' },
  { prop: 'status', label: '状态' },
]
const reportCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'process_id', label: '工序' },
  { prop: 'worker_id', label: '工人' },
  { prop: 'input_weight', label: '投料' },
  { prop: 'output_weight', label: '完工' },
  { prop: 'qty', label: '产量' },
  { prop: 'status', label: '状态' },
  { prop: 'reported_at', label: '时间' },
]
const processReportCols: MobileCardColumn[] = [
  { prop: 'process_name', label: '工序', primary: true },
  { prop: 'worker_name', label: '过站人' },
  { prop: 'operator_name', label: '操作人' },
  { prop: 'input_weight', label: '投料' },
  { prop: 'output_weight', label: '完工' },
  { prop: 'loss', label: '损耗' },
  { prop: 'bag_qty', label: '袋数' },
  { prop: 'scrap_type', label: '次品类型' },
  { prop: 'status', label: '状态' },
  { prop: 'reported_at', label: '时间' },
]
const pieceworkCols: MobileCardColumn[] = [
  { prop: 'id', label: 'ID', primary: true },
  { prop: 'worker_id', label: '工人' },
  { prop: 'process_id', label: '工序' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'qty', label: '产量' },
  { prop: 'amount', label: '金额' },
  { prop: 'status', label: '状态' },
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
  { prop: 'status', label: '状态' },
  { prop: 'created_at', label: '创建' },
]
const wipCols: MobileCardColumn[] = [
  { prop: 'step_name', label: '步骤', primary: true },
  { prop: 'seq_no', label: '序' },
  { prop: 'step_code', label: '步骤码' },
  { prop: 'process_name', label: '工序' },
  { prop: 'box_count', label: '箱数' },
  { prop: 'wip_weight', label: '在制重量 kg' },
]
const wipBoxCols: MobileCardColumn[] = [
  { prop: 'code', label: '箱码', primary: true },
  { prop: 'product_name', label: '产品' },
  { prop: 'weight', label: '重量' },
  { prop: 'trace_code', label: '溯源' },
  { prop: 'status', label: '状态' },
]
const workshopCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'status', label: '状态' },
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
const processReturnCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'box_code', label: '箱码' },
  { prop: 'return_weight', label: '退回kg' },
  { prop: 'reason', label: '原因' },
  { prop: 'status', label: '状态' },
]
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
  reports: '过站记录',
  'process-reports': '过站记录',
  piecework: '计件工资',
  'piece-issue': '计件领料表',
  boms: '自动BOM',
  mrp: 'MRP物料分析',
  requisitions: '联动式领料',
  workbench: '车间工作台',
  'process-wip': '工序在制',
  workshops: '车间管理',
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
const embedRoutings = computed(() => active.value === 'routings')
const embedPieceIssue = computed(() => active.value === 'piece-issue')
const reportTab = ref('ledger')

/** 默认 false：现场录入仅 App；管理员补单需 backfill_reason */
const fieldInputOnAdmin = import.meta.env.VITE_FIELD_INPUT_ON_ADMIN === 'true'

const backfillForm = reactive({ backfill_reason: '' })

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
  if (s === 'closed' || s === 'done' || s === 'finished') return 'success'
  if (s === 'in_progress' || s === 'released') return 'warning'
  if (s === 'cancelled' || s === 'void') return 'danger'
  return 'info'
}
const products = ref<Row[]>([])
const processes = ref<Row[]>([])
const workers = ref<Row[]>([])
const overview = ref<Row | null>(null)
const processReports = ref<Row[]>([])
const shiftDetail = ref<Row | null>(null)
const scrapTypeFilter = ref('')
const wipProductId = ref<number | null>(null)
const wipSummary = ref<Row | null>(null)
const wipDrawer = ref(false)
const wipBoxes = ref<Row[]>([])
const wipDrawerTitle = ref('')
const shiftMembers = computed(() =>
  ((shiftDetail.value?.members as Row[]) || []).map((m) => ({
    ...m,
    process_name: m.process_id === 0 ? '全工序' : m.process_name,
  })),
)
const shiftForm = reactive({ workshop_id: 1, remark: '产线开工' })
const shiftMemberForm = reactive({ employee_id: 2, process_id: 0 })

const taskForm = reactive({ product_id: 3, qty: 1000, routing_id: 1, workshop_id: 1, remark: '' })
const multiLines = ref<{ product_id: number; qty: number }[]>([{ product_id: 3, qty: 100 }])
const processEditDlg = ref(false)
const processEditForm = reactive({ id: 0, name: '', process_type: 'other', is_piecework: false, status: 'active' })
const dispatchForm = reactive({ task_id: null as number | null, process_id: 1, worker_id: 2, qty: 100 })
const reportForm = reactive({ process_id: 1, worker_id: 2, qty: 100, dispatch_id: null as number | null })
const reqForm = reactive({ product_id: 1, qty: 100, warehouse_id: 1 })
const processForm = reactive({ code: '', name: '', process_type: 'other', is_piecework: false })
const workshopForm = reactive({ code: '', name: '' })
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
    reportForm.process_id = id
    scrapForm.process_id = id
    qcForm.process_id = id
    reworkForm.process_id = id
    outForm.process_id = id
  }
  if (workers.value[0]) {
    dispatchForm.worker_id = Number(workers.value[0].id)
    reportForm.worker_id = Number(workers.value[0].id)
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
        res = await productionApi.listReportWorks()
        if (res && res.code !== 1) return ElMessage.error(res.msg)
        list.value = ((res?.data as { list?: Row[] })?.list) || []
        const q = scrapTypeFilter.value ? `scrap_type=${encodeURIComponent(scrapTypeFilter.value)}` : undefined
        const pr = await fieldLedgerApi.processReports(q)
        if (pr.code !== 1) return ElMessage.error(pr.msg)
        processReports.value = ((pr.data as { list?: Row[] })?.list) || []
        return
      case 'piecework':
        res = await productionApi.pieceworkSummaries()
        break
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
      case 'workshops':
        res = await productionApi.workshops()
        break
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

async function createTask() {
  const res = await productionApi.createTask({ ...taskForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`任务 ${(res.data as Row)?.doc_no}`)
  dispatchForm.task_id = Number((res.data as Row)?.id)
  await refresh()
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

async function createDispatch() {
  if (!dispatchForm.task_id) return ElMessage.warning('请选择任务或先建任务')
  const apiCall = active.value === 'flex' ? productionApi.createFlexDispatch : productionApi.createDispatch
  const res = await apiCall({ ...dispatchForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`派工 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function receiveDispatch(id: number) {
  const res = await productionApi.receiveDispatch(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已接收')
  await refresh()
}

async function createReport() {
  const reason = String(backfillForm.backfill_reason || '').trim()
  if (!reason) return ElMessage.warning('管理员补单须填写补单原因')
  const body: Record<string, unknown> = { ...reportForm, backfill_reason: reason, remark: reason }
  if (!reportForm.dispatch_id) delete body.dispatch_id
  const res = await productionApi.createReportWork(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('补单已提交')
  backfillForm.backfill_reason = ''
  await refresh()
}

async function confirmReport(id: number) {
  const res = await productionApi.confirmReportWork(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已确认')
  await refresh()
}

async function recalcPiece() {
  const res = await productionApi.recalcPiecework({})
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已重算')
  await refresh()
}

async function payPiece(id: number) {
  if (!payForm.transfer_no) return ElMessage.warning('请填转账单号')
  const res = await productionApi.payPiecework(id, { ...payForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已支付')
  await refresh()
}

async function createProcess() {
  if (!processForm.name) return ElMessage.warning('请填工序名')
  const code = processForm.code || `P${Date.now().toString().slice(-6)}`
  const res = await productionApi.createProcess({ ...processForm, code })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('工序已创建')
  await loadMeta()
  await refresh()
}

function openEditProcess(row: Row) {
  processEditForm.id = Number(row.id)
  processEditForm.name = String(row.name || '')
  processEditForm.process_type = String(row.process_type || 'other')
  processEditForm.is_piecework = Boolean(row.is_piecework)
  processEditForm.status = String(row.status || 'active')
  processEditDlg.value = true
}

async function saveEditProcess() {
  if (!processEditForm.id) return
  const res = await productionApi.updateProcess(processEditForm.id, {
    name: processEditForm.name,
    process_type: processEditForm.process_type,
    is_piecework: processEditForm.is_piecework,
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
    workshop_id: taskForm.workshop_id,
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

async function createWorkshop() {
  if (!workshopForm.name) return ElMessage.warning('请填车间名')
  const res = await productionApi.createWorkshop({
    code: workshopForm.code || `WS${Date.now().toString().slice(-4)}`,
    name: workshopForm.name,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('车间已创建')
  await refresh()
}

async function openWipBoxes(stepId: number, title: string, unassigned: boolean) {
  const parts = [
    unassigned ? 'unassigned=1' : `step_id=${stepId}`,
    wipProductId.value ? `product_id=${wipProductId.value}` : '',
  ].filter(Boolean)
  const res = await productionApi.processWipBoxes(parts.join('&'))
  if (res.code !== 1) return ElMessage.error(res.msg || '加载箱明细失败')
  wipBoxes.value = ((res.data as { list?: Row[] })?.list) || []
  wipDrawerTitle.value = `${title || '箱明细'}（${wipBoxes.value.length}）`
  wipDrawer.value = true
}

async function destroyWipBox(row: Row) {
  const id = Number(row.id)
  if (!id) return
  try {
    const { value } = await ElMessageBox.prompt('填写销毁原因（损耗等用不了的箱须标注销毁）', `销毁 ${row.code || ''}`, {
      confirmButtonText: '确认销毁',
      cancelButtonText: '取消',
      inputPattern: /\S+/,
      inputErrorMessage: '原因必填',
      type: 'warning',
    })
    const res = await inventoryApi.destroyBox(id, { reason: String(value || '').trim() })
    if (res.code !== 1) return ElMessage.error(res.msg)
    ElMessage.success('已销毁')
    wipBoxes.value = wipBoxes.value.filter((b) => Number(b.id) !== id)
    wipDrawerTitle.value = wipDrawerTitle.value.replace(/\（\d+\）/, `（${wipBoxes.value.length}）`)
    await refresh()
  } catch {
    /* cancel */
  }
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
  if (!returnForm.box_code.trim()) return ElMessage.warning('请填写箱码')
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

async function createShift() {
  const res = await productionApi.createShift({ ...shiftForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`班次 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function openShift(id: number) {
  const res = await productionApi.getShift(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  shiftDetail.value = (res.data as Row) || null
}

async function closeShiftRow(id: number) {
  const res = await productionApi.closeShift(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已收工')
  shiftDetail.value = null
  await refresh()
}

async function addShiftMember() {
  if (!shiftDetail.value?.id) return ElMessage.warning('请先选择班次')
  const res = await productionApi.addShiftMember(Number(shiftDetail.value.id), { ...shiftMemberForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已添加授权')
  await openShift(Number(shiftDetail.value.id))
}

async function removeShiftMember(memberId: number) {
  if (!shiftDetail.value?.id) return
  const res = await productionApi.removeShiftMember(Number(shiftDetail.value.id), memberId)
  if (res.code !== 1) return ElMessage.error(res.msg)
  await openShift(Number(shiftDetail.value.id))
}

watch(active, () => {
  reportTab.value = 'ledger'
  shiftDetail.value = null
  refresh()
})
watch(wipProductId, () => {
  if (active.value === 'process-wip') refresh()
})
onMounted(async () => {
  await loadMeta()
  await refresh()
})
</script>

<template>
  <div>
    <RoutingView v-if="embedRoutings" />
    <PieceIssueView v-else-if="embedPieceIssue" />

    <div v-else class="page" v-loading="loading">
      <div class="head">
        <h2>{{ title }}</h2>
        <p class="hint">管理端：配置、查询、结算与例外补单。日常过站/过磅请在 Flutter App 完成。</p>
      </div>

      <!-- 工序定义：新建 + 维护 -->
      <template v-if="active==='processes' || active==='process-mgmt'">
        <p class="mode-hint">工序主数据：编码、类型、是否计件。App 过站按工艺流程推进，此处为根配置。</p>
        <el-card header="新建工序" class="mb">
          <el-form inline size="small">
            <el-form-item label="编码"><el-input v-model="processForm.code" placeholder="可空自动" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="processForm.name" /></el-form-item>
            <el-form-item label="类型"><EnumSelect v-model="processForm.process_type" :options="PROCESS_TYPE_OPTIONS" style="width:140px" /></el-form-item>
            <el-form-item label="计件"><el-switch v-model="processForm.is_piecework" /></el-form-item>
            <el-button type="primary" @click="createProcess">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="processCols">
          <el-table :data="list" size="small">
            <el-table-column prop="code" label="编码" width="120" />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="process_type" label="类型" width="100" />
            <el-table-column prop="is_piecework" label="计件" width="80" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openEditProcess(row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openEditProcess(row)">编辑</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 产线班次：替代日常派工授权 -->
      <template v-else-if="active==='shifts'">
        <el-alert type="info" show-icon :closable="false" class="mb"
          title="产线开工班次授权：工人须在当日 open 班次成员中方可 App 过站。与「人事·考勤班次」不同。" />
        <el-row :gutter="12">
          <el-col :span="14" :xs="24">
            <el-card header="开工 / 班次列表" class="mb">
              <el-form inline size="small" class="mb">
                <el-form-item label="车间"><WorkshopSelect v-model="shiftForm.workshop_id" /></el-form-item>
                <el-form-item label="备注"><el-input v-model="shiftForm.remark" style="width:160px" /></el-form-item>
                <el-button type="primary" @click="createShift">开工</el-button>
              </el-form>
              <TableOrCards :data="list" :loading="loading" :columns="shiftCols">
                <el-table :data="list" size="small" highlight-current-row @row-click="(row: Row) => openShift(Number(row.id))">
                  <el-table-column prop="doc_no" label="班次号" width="150" />
                  <el-table-column prop="biz_date" label="日期" width="110" />
                  <el-table-column prop="workshop_name" label="车间" width="100" />
                  <el-table-column prop="member_count" label="人数" width="70" />
                  <el-table-column prop="status" label="状态" width="90" />
                  <el-table-column label="操作" width="100">
                    <template #default="{ row }">
                      <el-button v-if="row.status==='open'" link type="warning" @click.stop="closeShiftRow(Number(row.id))">收工</el-button>
                    </template>
                  </el-table-column>
                </el-table>
                <template #extra="{ row }">
                  <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
                </template>
                <template #actions="{ row }">
                  <el-button link type="primary" @click="openShift(Number(row.id))">成员</el-button>
                  <el-button v-if="row.status==='open'" link type="warning" @click="closeShiftRow(Number(row.id))">收工</el-button>
                </template>
              </TableOrCards>
            </el-card>
          </el-col>
          <el-col :span="10" :xs="24">
            <el-card header="成员授权" v-if="shiftDetail">
              <p class="hint mb">班次 {{ shiftDetail.doc_no }} · process_id=0 表示全工序</p>
              <el-form inline size="small" class="mb">
                <el-form-item label="员工">
                  <el-select v-model="shiftMemberForm.employee_id" style="width:120px">
                    <el-option v-for="w in workers" :key="String(w.id)" :label="String(w.name)" :value="Number(w.id)" />
                  </el-select>
                </el-form-item>
                <el-form-item label="工序">
                  <el-select v-model="shiftMemberForm.process_id" style="width:120px">
                    <el-option label="全工序" :value="0" />
                    <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
                  </el-select>
                </el-form-item>
                <el-button type="primary" @click="addShiftMember">添加</el-button>
              </el-form>
              <TableOrCards :data="shiftMembers" :columns="shiftMemberCols">
                <el-table :data="shiftMembers" size="small">
                  <el-table-column prop="employee_name" label="员工" />
                  <el-table-column prop="process_name" label="工序" width="100">
                    <template #default="{ row }">{{ row.process_id === 0 ? '全工序' : row.process_name }}</template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ row }">
                      <el-button link type="danger" @click="removeShiftMember(Number(row.id))">移除</el-button>
                    </template>
                  </el-table-column>
                </el-table>
                <template #actions="{ row }">
                  <el-button link type="danger" @click="removeShiftMember(Number(row.id))">移除</el-button>
                </template>
              </TableOrCards>
            </el-card>
            <el-empty v-else description="点击左侧班次查看/维护成员" />
          </el-col>
        </el-row>
      </template>

      <!-- 生产任务单：单商品任务 -->
      <template v-else-if="active==='tasks'">
        <p class="mode-hint">本页创建单商品生产任务；多商品请走「一单多商品」。</p>
        <el-card header="新建生产任务" class="mb">
          <el-form inline size="small">
            <el-form-item label="产品">
              <el-select v-model="taskForm.product_id" style="width:160px">
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="计划量"><el-input-number v-model="taskForm.qty" :min="1" /></el-form-item>
            <el-form-item label="工艺"><RoutingSelect v-model="taskForm.routing_id" /></el-form-item>
            <el-form-item label="车间"><WorkshopSelect v-model="taskForm.workshop_id" /></el-form-item>
            <el-button type="primary" @click="createTask">新建</el-button>
          </el-form>
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
                  :text-inside="false"
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

      <!-- 一单多商品：多行商品任务 -->
      <template v-else-if="active==='multi-products'">
        <p class="mode-hint">一张任务单挂多行商品；创建后可在明细中查看全部商品行。</p>
        <el-card header="新建多商品任务" class="mb">
          <el-form inline size="small" class="mb">
            <el-form-item label="工艺"><RoutingSelect v-model="taskForm.routing_id" /></el-form-item>
            <el-form-item label="车间"><WorkshopSelect v-model="taskForm.workshop_id" /></el-form-item>
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
        <el-alert type="info" show-icon :closable="false" class="mb" title="正常流转无需派工，App 扫工牌+箱码即可过站。此处仅用于例外派岗/灵活派发。" />
        <el-card v-if="fieldInputOnAdmin || active==='flex'" :header="active==='flex' ? '灵活派发（例外）' : '例外派岗'" class="mb">
          <el-form inline size="small">
            <el-form-item label="任务"><ProdTaskSelect v-model="dispatchForm.task_id" /></el-form-item>
            <el-form-item label="工序">
              <el-select v-model="dispatchForm.process_id" style="width:140px">
                <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="工人">
              <el-select v-model="dispatchForm.worker_id" style="width:140px">
                <el-option v-for="w in workers" :key="String(w.id)" :label="String(w.name)" :value="Number(w.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="dispatchForm.qty" :min="1" /></el-form-item>
            <el-button type="primary" @click="createDispatch">派工</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="dispatchCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="task_id" label="任务" width="80" />
            <el-table-column prop="process_id" label="工序" width="80" />
            <el-table-column prop="worker_id" label="工人" width="80" />
            <el-table-column prop="qty" label="数量" width="90" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button v-if="row.status==='dispatched'" link type="primary" @click="receiveDispatch(Number(row.id))">接收</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='dispatched'" link type="primary" @click="receiveDispatch(Number(row.id))">接收</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 过站记录：台账 + 加工明细 -->
      <template v-else-if="active==='reports' || active==='process-reports'">
        <el-alert type="warning" show-icon :closable="false" class="mb"
          title="现场过站请使用 App「工序过站」。本页仅查询；管理员补单需开启 VITE_FIELD_INPUT_ON_ADMIN 并填写补单原因。" />
        <el-tabs v-model="reportTab" class="mb">
          <el-tab-pane label="过站台账" name="ledger">
            <el-card v-if="fieldInputOnAdmin" header="管理员补单（须填原因）" class="mb">
              <el-form inline size="small">
                <el-form-item label="工序">
                  <el-select v-model="reportForm.process_id" style="width:140px">
                    <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
                  </el-select>
                </el-form-item>
                <el-form-item label="工人">
                  <el-select v-model="reportForm.worker_id" style="width:140px">
                    <el-option v-for="w in workers" :key="String(w.id)" :label="String(w.name)" :value="Number(w.id)" />
                  </el-select>
                </el-form-item>
                <el-form-item label="产量"><el-input-number v-model="reportForm.qty" :min="0.01" /></el-form-item>
                <el-form-item label="补单原因"><el-input v-model="backfillForm.backfill_reason" style="width:200px" placeholder="必填" /></el-form-item>
                <el-button type="primary" @click="createReport">提交补单</el-button>
              </el-form>
            </el-card>
            <TableOrCards :data="list" :loading="loading" :columns="reportCols">
              <el-table :data="list" size="small">
                <el-table-column prop="doc_no" label="单号" width="150" />
                <el-table-column prop="process_id" label="工序" width="80" />
                <el-table-column prop="worker_id" label="工人" width="80" />
                <el-table-column prop="input_weight" label="投料" width="80" />
                <el-table-column prop="output_weight" label="完工" width="80" />
                <el-table-column prop="qty" label="产量" width="90" />
                <el-table-column prop="status" label="状态" width="100" />
                <el-table-column prop="reported_at" label="时间" min-width="160" />
              </el-table>
              <template #extra="{ row }">
                <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
              </template>
            </TableOrCards>
          </el-tab-pane>
          <el-tab-pane label="加工明细" name="detail">
            <div style="margin-bottom:12px;display:flex;gap:8px;align-items:center">
              <el-select v-model="scrapTypeFilter" clearable placeholder="次品类型" style="width:200px" @change="refresh">
                <el-option label="切断次品" value="cut_defect" />
                <el-option label="去芯次品" value="core_defect" />
                <el-option label="切块次品" value="dice_defect" />
                <el-option label="筛选装袋次品" value="sieve_bag_defect" />
              </el-select>
            </div>
            <TableOrCards :data="processReports" :loading="loading" :columns="processReportCols">
              <el-table :data="processReports" size="small">
                <el-table-column prop="process_name" label="工序" width="120" />
                <el-table-column prop="worker_name" label="过站人" width="100" />
                <el-table-column prop="operator_name" label="操作人" width="100" />
                <el-table-column prop="scan_code" label="箱码" width="120" show-overflow-tooltip />
                <el-table-column prop="input_weight" label="投料" width="80" />
                <el-table-column prop="output_weight" label="完工" width="80" />
                <el-table-column prop="loss" label="损耗" width="80" />
                <el-table-column prop="bag_qty" label="袋数" width="70" />
                <el-table-column prop="scrap_type" label="次品类型" width="120" />
                <el-table-column prop="status" label="状态" width="100" />
                <el-table-column prop="reported_at" label="时间" min-width="160" />
              </el-table>
              <template #extra="{ row }">
                <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
              </template>
            </TableOrCards>
          </el-tab-pane>
        </el-tabs>
      </template>

      <!-- 计件 -->
      <template v-else-if="active==='piecework'">
        <el-card class="mb">
          <el-button type="primary" size="small" @click="recalcPiece">重算计件</el-button>
          <el-form inline size="small" style="margin-left:12px;display:inline">
            <el-form-item label="转账单号"><el-input v-model="payForm.transfer_no" /></el-form-item>
            <el-form-item label="回单"><el-input v-model="payForm.pay_evidence_url" /></el-form-item>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="pieceworkCols">
          <el-table :data="list" size="small">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="worker_id" label="工人" width="80" />
            <el-table-column prop="process_id" label="工序" width="80" />
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
        <el-row :gutter="12" class="mb" v-if="overview">
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">今日过站</div><div class="kpi-n">{{ overview.today_station_passes ?? overview.today_reports }}</div></el-card></el-col>
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">待确认</div><div class="kpi-n">{{ overview.pending_confirm ?? 0 }}</div></el-card></el-col>
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">开工班次</div><div class="kpi-n">{{ overview.open_shifts ?? 0 }}</div></el-card></el-col>
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">流转失败</div><div class="kpi-n">{{ overview.failed_flow_events }}</div></el-card></el-col>
        </el-row>
        <p v-if="overview" class="hint mb">例外派工 {{ overview.exception_dispatches ?? overview.open_dispatches ?? 0 }} 张 · 开立任务 {{ overview.open_tasks }}</p>
        <el-card header="今日/在制任务">
          <TableOrCards :data="list" :loading="loading" :columns="workbenchCols">
            <el-table :data="list" size="small">
              <el-table-column prop="doc_no" label="单号" />
              <el-table-column prop="status" label="状态" width="120" />
              <el-table-column prop="created_at" label="创建" />
            </el-table>
            <template #extra="{ row }">
              <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
            </template>
          </TableOrCards>
        </el-card>
      </template>

      <!-- 工序在制 -->
      <template v-else-if="active==='process-wip'">
        <div class="row mb">
          <ProductSelect v-model="wipProductId" clearable placeholder="按产品筛选（默认鲜木薯工艺）" style="width:240px" />
          <el-button @click="refresh">刷新</el-button>
          <span v-if="wipSummary" class="hint">工艺 {{ wipSummary.routing_code || '-' }}</span>
        </div>
        <el-row v-if="wipSummary" :gutter="12" class="mb">
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">在制箱数</div><div class="kpi-n">{{ wipSummary.total_boxes ?? 0 }}</div></el-card></el-col>
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">在制重量 kg</div><div class="kpi-n">{{ Number(wipSummary.total_weight || 0).toFixed(1) }}</div></el-card></el-col>
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">待确认过站</div><div class="kpi-n">{{ wipSummary.pending_confirm_reports ?? 0 }}</div></el-card></el-col>
          <el-col :span="6" :xs="24"><el-card shadow="never"><div class="kpi">待确认重量</div><div class="kpi-n">{{ Number(wipSummary.pending_confirm_weight || 0).toFixed(1) }}</div></el-card></el-col>
        </el-row>
        <p v-if="wipSummary?.unassigned" class="hint mb">
          未挂工序箱 {{ (wipSummary.unassigned as Row).box_count || 0 }} ·
          重量 {{ Number((wipSummary.unassigned as Row).wip_weight || 0).toFixed(1) }} kg
          <el-button
            v-if="Number((wipSummary.unassigned as Row).box_count || 0) > 0"
            link
            type="primary"
            @click="openWipBoxes(0, '未挂工序', true)"
          >查看</el-button>
        </p>
        <TableOrCards :data="list" :loading="loading" :columns="wipCols">
          <el-table :data="list" size="small" border stripe @row-click="(row: Row) => openWipBoxes(Number(row.step_id), String(row.step_name || ''), false)">
            <el-table-column prop="seq_no" label="序" width="60" />
            <el-table-column prop="step_code" label="步骤码" width="90" />
            <el-table-column prop="step_name" label="步骤" min-width="140" />
            <el-table-column prop="process_name" label="工序" width="120" />
            <el-table-column prop="box_count" label="箱数" width="80" />
            <el-table-column label="在制重量 kg" width="120">
              <template #default="{ row }">{{ Number(row.wip_weight || 0).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openWipBoxes(Number(row.step_id), String(row.step_name || ''), false)">箱明细</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openWipBoxes(Number(row.step_id), String(row.step_name || ''), false)">箱明细</el-button>
          </template>
        </TableOrCards>
        <el-drawer v-model="wipDrawer" :title="wipDrawerTitle" size="480px">
          <TableOrCards :data="wipBoxes" :columns="wipBoxCols">
            <el-table :data="wipBoxes" size="small" border>
              <el-table-column prop="code" label="箱码" min-width="140" />
              <el-table-column prop="product_name" label="产品" width="100" />
              <el-table-column label="重量" width="90">
                <template #default="{ row }">{{ Number(row.weight || 0).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column prop="trace_code" label="溯源" width="110" />
              <el-table-column prop="status" label="状态" width="80" />
              <el-table-column label="操作" width="80" fixed="right">
                <template #default="{ row }">
                  <el-button link type="danger" @click="destroyWipBox(row)">销毁</el-button>
                </template>
              </el-table-column>
            </el-table>
            <template #extra="{ row }">
              <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
            </template>
            <template #actions="{ row }">
              <el-button link type="danger" @click="destroyWipBox(row)">销毁</el-button>
            </template>
          </TableOrCards>
        </el-drawer>
      </template>

      <!-- 车间 -->
      <template v-else-if="active==='workshops'">
        <el-card header="新建车间" class="mb">
          <el-form inline size="small">
            <el-form-item label="编码"><el-input v-model="workshopForm.code" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="workshopForm.name" /></el-form-item>
            <el-button type="primary" @click="createWorkshop">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="workshopCols">
          <el-table :data="list" size="small">
            <el-table-column prop="code" label="编码" width="120" />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="status" label="状态" width="90" />
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
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
            <el-form-item label="箱码"><el-input v-model="returnForm.box_code" style="width:160px" /></el-form-item>
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
            <el-table-column prop="box_code" label="箱码" width="130" />
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
            <el-tag :type="statusTagType(detail.status)" size="small">{{ detail.status }}</el-tag>
            <span class="hint">创建 {{ detail.created_at || '-' }}</span>
            <span v-if="detail.routing_id" class="hint">工艺 #{{ detail.routing_id }}</span>
            <span v-if="detail.workshop_id" class="hint">车间 #{{ detail.workshop_id }}</span>
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

      <el-dialog v-model="processEditDlg" title="编辑工序" width="480px" destroy-on-close>
        <el-form label-width="90px">
          <el-form-item label="名称"><el-input v-model="processEditForm.name" /></el-form-item>
          <el-form-item label="类型"><EnumSelect v-model="processEditForm.process_type" :options="PROCESS_TYPE_OPTIONS" style="width:100%" /></el-form-item>
          <el-form-item label="计件"><el-switch v-model="processEditForm.is_piecework" /></el-form-item>
          <el-form-item label="状态">
            <el-select v-model="processEditForm.status" style="width:100%">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
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
</style>
