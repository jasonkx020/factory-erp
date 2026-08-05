<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { productionApi, productApi, hrApi } from '@erp/shared'
import RoutingView from '../automation/RoutingView.vue'
import ProcessReportView from './ProcessReportView.vue'
import PieceIssueView from './PieceIssueView.vue'

type Row = Record<string, unknown>

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  processes: '工序设置',
  routings: '工艺流程',
  tasks: '生产任务单',
  dispatches: '生产派工',
  flex: '灵活派发工单',
  reports: '扫码报工',
  piecework: '计件工资',
  'process-reports': '加工记录',
  'piece-issue': '计件领料表',
  boms: '自动BOM',
  mrp: 'MRP物料分析',
  requisitions: '联动式领料',
  workbench: '车间工作台',
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
}

const active = computed(() => String(route.params.section || 'tasks'))
const title = computed(() => TITLE_MAP[active.value] || '生产管理')
const embedRoutings = computed(() => active.value === 'routings')
const embedProcessReports = computed(() => active.value === 'process-reports')
const embedPieceIssue = computed(() => active.value === 'piece-issue')

const loading = ref(false)
const list = ref<Row[]>([])
const detail = ref<Row | null>(null)
const products = ref<Row[]>([])
const processes = ref<Row[]>([])
const workers = ref<Row[]>([])
const overview = ref<Row | null>(null)

const taskForm = reactive({ product_id: 3, qty: 1000, routing_id: 1, workshop_id: 1, remark: '' })
const dispatchForm = reactive({ task_id: 0 as number, process_id: 1, worker_id: 2, qty: 100 })
const reportForm = reactive({ process_id: 1, worker_id: 2, qty: 100, dispatch_id: 0 as number })
const reqForm = reactive({ product_id: 1, qty: 100, warehouse_id: 1 })
const processForm = reactive({ code: '', name: '', process_type: 'other', is_piecework: false })
const workshopForm = reactive({ code: '', name: '' })
const bomForm = reactive({ product_id: 3, name: '生产BOM', component_product_id: 1, qty: 1.2, scrap_rate: 0.05 })
const scrapForm = reactive({ product_id: 1, qty: 10, scrap_type: 'cut_defect', process_id: 1, remark: '' })
const qcForm = reactive({ qc_type: 'process', product_id: 3, process_id: 1, qty: 100 })
const reworkForm = reactive({ process_id: 1, qty: 10, remark: '' })
const mergeForm = reactive({ title: '多单整合', task_ids: '' })
const drawingForm = reactive({ drawing_code: '', drawing_name: '', task_id: 0 as number, file_url: '' })
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
  }
}

async function refresh() {
  if (embedRoutings.value || embedProcessReports.value || embedPieceIssue.value) return
  loading.value = true
  detail.value = null
  try {
    let res
    switch (active.value) {
      case 'processes':
        res = await productionApi.processes()
        break
      case 'tasks':
        res = await productionApi.listTasks()
        break
      case 'dispatches':
        res = await productionApi.listDispatches()
        break
      case 'flex':
        res = await productionApi.listFlexDispatches()
        break
      case 'reports':
        res = await productionApi.listReportWorks()
        break
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
  detail.value = { ...(res.data as Row), items: (items.data as { list?: Row[] })?.list || [] }
}

async function createDispatch() {
  if (!dispatchForm.task_id) return ElMessage.warning('请填写任务ID或先建任务')
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
  const body: Record<string, unknown> = { ...reportForm }
  if (!reportForm.dispatch_id) delete body.dispatch_id
  const res = await productionApi.createReportWork(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('报工已提交')
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
  const ids = mergeForm.task_ids.split(/[,，\s]+/).map(Number).filter((n) => n > 0)
  if (!ids.length) return ElMessage.warning('请填写任务ID，逗号分隔')
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

watch(active, () => refresh())
onMounted(async () => {
  await loadMeta()
  await refresh()
})
</script>

<template>
  <div>
    <RoutingView v-if="embedRoutings" />
    <ProcessReportView v-else-if="embedProcessReports" />
    <PieceIssueView v-else-if="embedPieceIssue" />

    <div v-else class="page" v-loading="loading">
      <div class="head">
        <h2>{{ title }}</h2>
        <p class="hint">工厂产线交付：工序/工艺 → 任务派工 → 扫码报工/计件 → 质检废料。现场双扫在 Flutter；本页为管理端台账。</p>
      </div>

      <!-- 工序 -->
      <template v-if="active==='processes'">
        <el-card header="新建工序" class="mb">
          <el-form inline size="small">
            <el-form-item label="编码"><el-input v-model="processForm.code" placeholder="可空自动" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="processForm.name" /></el-form-item>
            <el-form-item label="类型"><el-input v-model="processForm.process_type" style="width:100px" /></el-form-item>
            <el-form-item label="计件"><el-switch v-model="processForm.is_piecework" /></el-form-item>
            <el-button type="primary" @click="createProcess">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="process_type" label="类型" width="100" />
          <el-table-column prop="is_piecework" label="计件" width="80" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </template>

      <!-- 任务 / 一单多商品 -->
      <template v-else-if="active==='tasks'">
        <el-card header="新建生产任务（可带商品行）" class="mb">
          <el-form inline size="small">
            <el-form-item label="产品">
              <el-select v-model="taskForm.product_id" style="width:160px">
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="计划量"><el-input-number v-model="taskForm.qty" :min="1" /></el-form-item>
            <el-form-item label="工艺ID"><el-input-number v-model="taskForm.routing_id" :min="0" /></el-form-item>
            <el-form-item label="车间ID"><el-input-number v-model="taskForm.workshop_id" :min="0" /></el-form-item>
            <el-button type="primary" @click="createTask">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="160" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="created_at" label="创建时间" />
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button link @click="openTask(Number(row.id)); dispatchForm.task_id=Number(row.id)">明细</el-button>
              <el-button v-if="row.status!=='closed'" link type="warning" @click="closeTask(Number(row.id))">关闭</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 派工 / 灵活 -->
      <template v-else-if="active==='dispatches' || active==='flex'">
        <el-card :header="active==='flex' ? '灵活派发' : '新建派工'" class="mb">
          <el-form inline size="small">
            <el-form-item label="任务ID"><el-input-number v-model="dispatchForm.task_id" :min="1" /></el-form-item>
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
      </template>

      <!-- 报工 -->
      <template v-else-if="active==='reports'">
        <el-card header="管理端补录报工（现场请用 App 双扫）" class="mb">
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
            <el-form-item label="派工ID"><el-input-number v-model="reportForm.dispatch_id" :min="0" /></el-form-item>
            <el-button type="primary" @click="createReport">提交报工</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="process_id" label="工序" width="80" />
          <el-table-column prop="worker_id" label="工人" width="80" />
          <el-table-column prop="qty" label="产量" width="90" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='submitted' || row.status==='draft'" link type="success" @click="confirmReport(Number(row.id))">确认</el-button>
            </template>
          </el-table-column>
        </el-table>
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
            <el-form-item label="组件ID"><el-input-number v-model="bomForm.component_product_id" :min="1" /></el-form-item>
            <el-form-item label="用量"><el-input-number v-model="bomForm.qty" :min="0.01" :step="0.1" /></el-form-item>
            <el-form-item label="损耗率"><el-input-number v-model="bomForm.scrap_rate" :min="0" :max="1" :step="0.01" /></el-form-item>
            <el-button type="primary" @click="createBom">新建</el-button>
            <el-button @click="genBom">自动生成</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="code" label="编码" width="180" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="product_id" label="成品" width="90" />
          <el-table-column prop="version_no" label="版本" width="80" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </template>

      <!-- MRP -->
      <template v-else-if="active==='mrp'">
        <el-card class="mb">
          <el-button type="primary" @click="runMrp">运行 MRP</el-button>
          <span class="hint" style="margin-left:8px">按未完成任务需求 + BOM 展开，对比库存给出短缺建议。</span>
        </el-card>
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
      </template>

      <!-- 领料 -->
      <template v-else-if="active==='requisitions'">
        <el-card header="联动领料" class="mb">
          <el-form inline size="small">
            <el-form-item label="物料ID"><el-input-number v-model="reqForm.product_id" :min="1" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="reqForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="仓库"><el-input-number v-model="reqForm.warehouse_id" :min="1" /></el-form-item>
            <el-button type="primary" @click="createReq">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft' || row.status==='open'" link type="primary" @click="postReq(Number(row.id))">过账</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 工作台 -->
      <template v-else-if="active==='workbench'">
        <el-row :gutter="12" class="mb" v-if="overview">
          <el-col :span="6"><el-card shadow="never"><div class="kpi">开立任务</div><div class="kpi-n">{{ overview.open_tasks }}</div></el-card></el-col>
          <el-col :span="6"><el-card shadow="never"><div class="kpi">在派工</div><div class="kpi-n">{{ overview.open_dispatches }}</div></el-card></el-col>
          <el-col :span="6"><el-card shadow="never"><div class="kpi">今日报工</div><div class="kpi-n">{{ overview.today_reports }}</div></el-card></el-col>
          <el-col :span="6"><el-card shadow="never"><div class="kpi">流转失败</div><div class="kpi-n">{{ overview.failed_flow_events }}</div></el-card></el-col>
        </el-row>
        <el-card header="今日/在制任务">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" />
            <el-table-column prop="status" label="状态" width="120" />
            <el-table-column prop="created_at" label="创建" />
          </el-table>
        </el-card>
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
        <el-table :data="list" size="small">
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </template>

      <!-- 进度 -->
      <template v-else-if="active==='progress'">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="任务单" width="160" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="plan_qty" label="计划" width="100" />
          <el-table-column prop="completed_qty" label="完成" width="100" />
          <el-table-column prop="progress_pct" label="进度%" width="100" />
          <el-table-column prop="created_at" label="创建" />
        </el-table>
      </template>

      <!-- 多单整合 -->
      <template v-else-if="active==='merges'">
        <el-card header="多单整合" class="mb">
          <el-form inline size="small">
            <el-form-item label="标题"><el-input v-model="mergeForm.title" /></el-form-item>
            <el-form-item label="任务IDs"><el-input v-model="mergeForm.task_ids" placeholder="1,2,3" style="width:200px" /></el-form-item>
            <el-button type="primary" @click="createMerge">新建</el-button>
          </el-form>
        </el-card>
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
      </template>

      <!-- 图纸 -->
      <template v-else-if="active==='drawings'">
        <el-card header="图纸分发挂接" class="mb">
          <el-form inline size="small">
            <el-form-item label="图纸编码"><el-input v-model="drawingForm.drawing_code" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="drawingForm.drawing_name" /></el-form-item>
            <el-form-item label="任务ID"><el-input-number v-model="drawingForm.task_id" :min="0" /></el-form-item>
            <el-form-item label="文件URL"><el-input v-model="drawingForm.file_url" style="width:200px" /></el-form-item>
            <el-button type="primary" @click="createDrawing">挂接</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="drawing_code" label="编码" width="120" />
          <el-table-column prop="drawing_name" label="名称" />
          <el-table-column prop="task_id" label="任务" width="80" />
          <el-table-column prop="file_url" label="文件" min-width="160" show-overflow-tooltip />
        </el-table>
      </template>

      <!-- 质检 -->
      <template v-else-if="active==='qc'">
        <el-card header="质检单" class="mb">
          <el-form inline size="small">
            <el-form-item label="类型"><el-input v-model="qcForm.qc_type" style="width:100px" /></el-form-item>
            <el-form-item label="产品ID"><el-input-number v-model="qcForm.product_id" :min="1" /></el-form-item>
            <el-form-item label="工序">
              <el-select v-model="qcForm.process_id" style="width:140px">
                <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="qcForm.qty" :min="0" /></el-form-item>
            <el-button type="primary" @click="createQc">新建</el-button>
          </el-form>
        </el-card>
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
      </template>

      <!-- 废料 -->
      <template v-else-if="active==='scraps'">
        <el-card header="废料登记" class="mb">
          <el-form inline size="small">
            <el-form-item label="料号ID"><el-input-number v-model="scrapForm.product_id" :min="1" /></el-form-item>
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
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="product_id" label="料号" width="80" />
          <el-table-column prop="qty" label="数量" width="90" />
          <el-table-column prop="scrap_type" label="类型" width="120" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </template>

      <!-- 委外 -->
      <template v-else-if="active==='outsources'">
        <el-card header="委外加工" class="mb">
          <el-form inline size="small">
            <el-form-item label="供应商ID"><el-input-number v-model="outForm.supplier_id" :min="1" /></el-form-item>
            <el-form-item label="工序ID"><el-input-number v-model="outForm.process_id" :min="1" /></el-form-item>
            <el-form-item label="产品ID"><el-input-number v-model="outForm.product_id" :min="1" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="outForm.qty" :min="0.01" /></el-form-item>
            <el-button type="primary" @click="createOut">新建</el-button>
          </el-form>
        </el-card>
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
      </template>

      <!-- 受托 -->
      <template v-else-if="active==='consignments'">
        <el-card header="受托加工" class="mb">
          <el-form inline size="small">
            <el-form-item label="客户ID"><el-input-number v-model="consForm.customer_id" :min="1" /></el-form-item>
            <el-form-item label="产品ID"><el-input-number v-model="consForm.product_id" :min="1" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="consForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="进度"><el-input v-model="consForm.progress" /></el-form-item>
            <el-button type="primary" @click="createCons">新建</el-button>
          </el-form>
        </el-card>
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
      </template>

      <!-- 成本隐藏 -->
      <template v-else-if="active==='cost-hide'">
        <el-card header="成本隐藏策略" class="mb">
          <el-form inline size="small">
            <el-form-item label="角色ID"><el-input-number v-model="hideForm.role_id" :min="1" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="hideForm.name" /></el-form-item>
            <el-button type="primary" @click="createHide">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="role_id" label="角色" width="90" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="field_scope" label="字段范围" min-width="200" show-overflow-tooltip />
          <el-table-column prop="is_enabled" label="启用" width="80" />
        </el-table>
      </template>

      <el-card v-if="detail" header="明细" style="margin-top:16px">
        <pre class="detail">{{ JSON.stringify(detail, null, 2) }}</pre>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.head h2 { margin: 0 0 4px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
.mb { margin-bottom: 12px; }
.detail { background: #f6f8fa; padding: 12px; border-radius: 8px; max-height: 420px; overflow: auto; font-size: 12px; }
.kpi { color: #667; font-size: 12px; }
.kpi-n { font-size: 28px; font-weight: 600; margin-top: 4px; }
</style>
