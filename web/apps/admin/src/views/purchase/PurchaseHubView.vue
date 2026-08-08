<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { purchaseApi, productApi } from '@erp/shared'
import {
  ProductSelect,
  SupplierSelect,
  WarehouseSelect,
  PurchaseInboundSelect,
  UserSelect,
} from '../../components/select'
import SupplierView from './SupplierView.vue'
import FarmerInboundView from './FarmerInboundView.vue'
import WeighVarietyView from './WeighVarietyView.vue'
import TraceBatchView from './TraceBatchView.vue'
import FlowGraphEditorView from '../automation/FlowGraphEditorView.vue'

type Row = Record<string, unknown>

const route = useRoute()

const TITLE_MAP: Record<string, string> = {
  suppliers: '供应商管理',
  farmers: '农户档案',
  weigh: '过磅收货',
  'flow-graphs': '过磅流程编排',
  varieties: '过磅品种',
  'trace-batches': '溯源批号',
  settlements: '农户结算',
  trace: '原料溯源',
  requests: '采购申请',
  plans: '采购计划单',
  inbounds: '采购入库',
  qcs: '来料质检',
  returns: '采购退货',
  analytics: '采购分析',
  prices: '历史价格查看',
  tasks: '采购任务管理',
}

const active = computed(() => String(route.params.section || 'suppliers'))
const title = computed(() => TITLE_MAP[active.value] || '采购管理')
const isFarmerSection = computed(() => ['farmers', 'weigh', 'settlements', 'trace'].includes(active.value))
const isFlowGraphSection = computed(() => active.value === 'flow-graphs')
const isVarietySection = computed(() => active.value === 'varieties')
const isTraceBatchSection = computed(() => active.value === 'trace-batches')
const isSupplierSection = computed(() => active.value === 'suppliers')

const loading = ref(false)
const list = ref<Row[]>([])
const detail = ref<Row | null>(null)
const suppliers = ref<Row[]>([])
const products = ref<Row[]>([])

const reqForm = reactive({
  title: '原料采购申请',
  product_id: 1,
  qty: 1000,
  supplier_id: 1,
  need_date: new Date().toISOString().slice(0, 10),
  remark: '',
})
const planForm = reactive({
  product_id: 1,
  qty: 1000,
  supplier_id: 1,
  plan_date: new Date().toISOString().slice(0, 10),
  remark: '',
})
const inboundForm = reactive({
  supplier_id: 1,
  warehouse_id: 1,
  product_id: 1,
  qty: 1000,
  price: 1.85,
  batch_no: '',
  plan_id: 0 as number,
  remark: '',
})
const qcForm = reactive({
  supplier_id: 1,
  inbound_id: 0 as number,
  product_id: 1,
  qty_check: 1000,
  remark: '',
})
const returnForm = reactive({
  supplier_id: 1,
  inbound_id: 0 as number,
  warehouse_id: 1,
  product_id: 1,
  qty: 100,
  reason: '质量不合格',
})
const taskForm = reactive({
  title: '采购跟进任务',
  product_id: 1,
  qty: 1000,
  supplier_id: 1,
  assignee_id: 1,
  due_date: '',
  remark: '',
})

async function loadMeta() {
  const [s, p] = await Promise.all([purchaseApi.suppliers('page_size=100'), productApi.list()])
  suppliers.value = ((s.data as { list?: Row[] })?.list) || []
  products.value = ((p.data as { list?: Row[] })?.list) || []
  if (suppliers.value[0]) {
    const id = Number(suppliers.value[0].id)
    reqForm.supplier_id = id
    planForm.supplier_id = id
    inboundForm.supplier_id = id
    qcForm.supplier_id = id
    returnForm.supplier_id = id
    taskForm.supplier_id = id
  }
  if (products.value[0]) {
    const pid = Number(products.value[0].id)
    reqForm.product_id = pid
    planForm.product_id = pid
    inboundForm.product_id = pid
    qcForm.product_id = pid
    returnForm.product_id = pid
    taskForm.product_id = pid
  }
}

async function refresh() {
  if (isSupplierSection.value || isFarmerSection.value || isVarietySection.value || isTraceBatchSection.value || isFlowGraphSection.value) return
  loading.value = true
  detail.value = null
  try {
    let res
    switch (active.value) {
      case 'requests':
        res = await purchaseApi.requests()
        break
      case 'plans':
        res = await purchaseApi.plans()
        break
      case 'inbounds':
        res = await purchaseApi.inbounds()
        break
      case 'qcs':
        res = await purchaseApi.qcs()
        break
      case 'returns':
        res = await purchaseApi.returns()
        break
      case 'prices':
        res = await purchaseApi.priceHistories()
        break
      case 'analytics': {
        const [vp, sp] = await Promise.all([
          purchaseApi.volumePrice(),
          purchaseApi.supplierPerformance(),
        ])
        const volume = ((vp.data as { list?: Row[] })?.list) || []
        const perf = ((sp.data as { list?: Row[] })?.list) || []
        list.value = [
          ...volume.map((r) => ({ ...r, _kind: '量价' })),
          ...perf.map((r) => ({ ...r, _kind: '绩效' })),
        ]
        return
      }
      case 'tasks':
        res = await purchaseApi.tasks()
        break
      default:
        res = await purchaseApi.requests()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function createRequest() {
  const res = await purchaseApi.createRequest({ ...reqForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`申请 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function submitReq(id: number) {
  const res = await purchaseApi.submitRequest(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交')
  await refresh()
}

async function approveReq(id: number) {
  const res = await purchaseApi.approveRequest(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已批准')
  await refresh()
}

async function toPlan(id: number) {
  const res = await purchaseApi.requestToPlan(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已生成计划 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function createPlan() {
  const res = await purchaseApi.createPlan({ ...planForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`计划 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function submitPlan(id: number) {
  const res = await purchaseApi.submitPlan(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交')
  await refresh()
}

async function approvePlan(id: number) {
  const res = await purchaseApi.approvePlan(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已批准')
  await refresh()
}

async function toInbound(id: number) {
  const res = await purchaseApi.planToInbound(id, {
    supplier_id: planForm.supplier_id,
    warehouse_id: inboundForm.warehouse_id,
    price: inboundForm.price,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已生成入库单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function createInbound() {
  const body: Record<string, unknown> = {
    supplier_id: inboundForm.supplier_id,
    warehouse_id: inboundForm.warehouse_id,
    product_id: inboundForm.product_id,
    qty: inboundForm.qty,
    price: inboundForm.price,
    batch_no: inboundForm.batch_no,
    remark: inboundForm.remark,
  }
  if (inboundForm.plan_id) body.plan_id = inboundForm.plan_id
  const res = await purchaseApi.createInbound(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`入库单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postInbound(id: number) {
  const res = await purchaseApi.postInbound(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已过账入库')
  await refresh()
}

async function createQc() {
  const body: Record<string, unknown> = { ...qcForm }
  if (!qcForm.inbound_id) delete body.inbound_id
  const res = await purchaseApi.createQc(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`质检单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function passQc(id: number) {
  const res = await purchaseApi.passQc(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('质检通过')
  await refresh()
}

async function failQc(id: number) {
  const res = await purchaseApi.failQc(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('质检不合格')
  await refresh()
}

async function createReturn() {
  const body: Record<string, unknown> = { ...returnForm }
  if (!returnForm.inbound_id) delete body.inbound_id
  const res = await purchaseApi.createReturn(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`退货单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postReturn(id: number) {
  const res = await purchaseApi.postReturn(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('退货已过账扣库存')
  await refresh()
}

async function createTask() {
  const res = await purchaseApi.createTask({ ...taskForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`任务 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function assignTask(id: number) {
  const res = await purchaseApi.assignTask(id, { assignee_id: taskForm.assignee_id })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已分配')
  await refresh()
}

async function completeTask(id: number) {
  const res = await purchaseApi.completeTask(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已完成')
  await refresh()
}

async function openDetail(kind: string, id: number) {
  let res
  if (kind === 'request') res = await purchaseApi.getRequest(id)
  else if (kind === 'plan') res = await purchaseApi.getPlan(id)
  else if (kind === 'inbound') res = await purchaseApi.getInbound(id)
  else if (kind === 'return') res = await purchaseApi.getReturn(id)
  else return
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

watch(active, async () => {
  await refresh()
})

onMounted(async () => {
  await loadMeta()
  await refresh()
})
</script>

<template>
  <div>
    <SupplierView v-if="isSupplierSection" />
    <WeighVarietyView v-else-if="isVarietySection" />
    <TraceBatchView v-else-if="isTraceBatchSection" />
    <FarmerInboundView v-else-if="isFarmerSection" :section="active" />
    <div v-else-if="isFlowGraphSection" class="page">
      <h2>过磅流程编排</h2>
      <p class="hint">配置入厂/入库过磅岗序；运行时按图推送下一角色待办。</p>
      <FlowGraphEditorView kind-filter="purchase" />
    </div>

    <div v-else class="page" v-loading="loading">
      <div class="head">
        <h2>{{ title }}</h2>
        <p class="hint">
          工厂采购双轨：农户过磅闭环 + 供应商正式采购（申请→计划→入库过账→质检/退货）。仓管入库见「库存管理/仓管待入库」。
        </p>
      </div>

      <!-- 采购申请 -->
      <template v-if="active === 'requests'">
        <el-card header="新建采购申请" class="mb">
          <el-form inline size="small">
            <el-form-item label="标题"><el-input v-model="reqForm.title" /></el-form-item>
            <el-form-item label="物料"><ProductSelect v-model="reqForm.product_id" :clearable="false" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="reqForm.qty" :min="1" /></el-form-item>
            <el-form-item label="建议供应商"><SupplierSelect v-model="reqForm.supplier_id" :clearable="false" /></el-form-item>
            <el-form-item label="需用日"><el-date-picker v-model="reqForm.need_date" type="date" value-format="YYYY-MM-DD" style="width:150px" /></el-form-item>
            <el-button type="primary" @click="createRequest">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="title" label="标题" min-width="140" />
          <el-table-column prop="qty" label="数量" width="90" />
          <el-table-column prop="need_date" label="需用日" width="110" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column label="操作" width="320" fixed="right">
            <template #default="{ row }">
              <el-button link @click="openDetail('request', Number(row.id))">明细</el-button>
              <el-button v-if="row.status==='draft' || row.status==='rejected'" link type="primary" @click="submitReq(Number(row.id))">提交</el-button>
              <el-button v-if="row.status==='submitted'" link type="success" @click="approveReq(Number(row.id))">批准</el-button>
              <el-button v-if="row.status==='approved' || row.status==='submitted'" link type="warning" @click="toPlan(Number(row.id))">转计划</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 采购计划 -->
      <template v-else-if="active === 'plans'">
        <el-card header="新建采购计划" class="mb">
          <el-form inline size="small">
            <el-form-item label="物料"><ProductSelect v-model="planForm.product_id" :clearable="false" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="planForm.qty" :min="1" /></el-form-item>
            <el-form-item label="供应商"><SupplierSelect v-model="planForm.supplier_id" :clearable="false" /></el-form-item>
            <el-form-item label="计划日"><el-date-picker v-model="planForm.plan_date" type="date" value-format="YYYY-MM-DD" style="width:150px" /></el-form-item>
            <el-button type="primary" @click="createPlan">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="plan_date" label="计划日" width="110" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="remark" label="备注" min-width="160" />
          <el-table-column label="操作" width="340" fixed="right">
            <template #default="{ row }">
              <el-button link @click="openDetail('plan', Number(row.id))">明细</el-button>
              <el-button v-if="row.status==='draft' || row.status==='rejected'" link type="primary" @click="submitPlan(Number(row.id))">提交</el-button>
              <el-button v-if="row.status==='submitted'" link type="success" @click="approvePlan(Number(row.id))">批准</el-button>
              <el-button v-if="row.status==='approved' || row.status==='submitted'" link type="warning" @click="toInbound(Number(row.id))">转入库</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 采购入库 -->
      <template v-else-if="active === 'inbounds'">
        <el-card header="新建采购入库（供应商，过账写库存）" class="mb">
          <el-form inline size="small">
            <el-form-item label="供应商"><SupplierSelect v-model="inboundForm.supplier_id" :clearable="false" /></el-form-item>
            <el-form-item label="物料"><ProductSelect v-model="inboundForm.product_id" :clearable="false" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="inboundForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="单价"><el-input-number v-model="inboundForm.price" :min="0" :step="0.01" /></el-form-item>
            <el-form-item label="仓库"><WarehouseSelect v-model="inboundForm.warehouse_id" :clearable="false" /></el-form-item>
            <el-form-item label="批次"><el-input v-model="inboundForm.batch_no" style="width:120px" /></el-form-item>
            <el-button type="primary" @click="createInbound">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="supplier_id" label="供应商" width="90" />
          <el-table-column prop="warehouse_id" label="仓库" width="80" />
          <el-table-column prop="biz_date" label="业务日" width="110" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button link @click="openDetail('inbound', Number(row.id))">明细</el-button>
              <el-button v-if="row.status==='draft'" link type="primary" @click="postInbound(Number(row.id))">过账入库</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 来料质检 -->
      <template v-else-if="active === 'qcs'">
        <el-card header="新建来料质检" class="mb">
          <el-form inline size="small">
            <el-form-item label="供应商"><SupplierSelect v-model="qcForm.supplier_id" :clearable="false" /></el-form-item>
            <el-form-item label="物料"><ProductSelect v-model="qcForm.product_id" :clearable="false" /></el-form-item>
            <el-form-item label="抽检量"><el-input-number v-model="qcForm.qty_check" :min="0" /></el-form-item>
            <el-form-item label="入库单"><PurchaseInboundSelect v-model="qcForm.inbound_id" /></el-form-item>
            <el-button type="primary" @click="createQc">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="product_id" label="物料" width="90" />
          <el-table-column prop="qty_check" label="抽检" width="90" />
          <el-table-column prop="qty_pass" label="合格" width="90" />
          <el-table-column prop="qty_fail" label="不合格" width="90" />
          <el-table-column prop="result" label="结果" width="90" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft' || row.status==='pending'" link type="success" @click="passQc(Number(row.id))">通过</el-button>
              <el-button v-if="row.status==='draft' || row.status==='pending'" link type="danger" @click="failQc(Number(row.id))">不合格</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 退货 -->
      <template v-else-if="active === 'returns'">
        <el-card header="新建采购退货" class="mb">
          <el-form inline size="small">
            <el-form-item label="供应商"><SupplierSelect v-model="returnForm.supplier_id" :clearable="false" /></el-form-item>
            <el-form-item label="物料"><ProductSelect v-model="returnForm.product_id" :clearable="false" /></el-form-item>
            <el-form-item label="入库单"><PurchaseInboundSelect v-model="returnForm.inbound_id" /></el-form-item>
            <el-form-item label="仓库"><WarehouseSelect v-model="returnForm.warehouse_id" :clearable="false" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="returnForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="原因"><el-input v-model="returnForm.reason" style="width:160px" /></el-form-item>
            <el-button type="primary" @click="createReturn">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="supplier_id" label="供应商" width="90" />
          <el-table-column prop="reason" label="原因" min-width="140" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="180">
            <template #default="{ row }">
              <el-button link @click="openDetail('return', Number(row.id))">明细</el-button>
              <el-button v-if="row.status==='draft'" link type="warning" @click="postReturn(Number(row.id))">过账退货</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- 历史价 -->
      <template v-else-if="active === 'prices'">
        <el-table :data="list" size="small">
          <el-table-column prop="supplier_id" label="供应商" width="100" />
          <el-table-column prop="product_id" label="物料" width="100" />
          <el-table-column prop="price" label="价格" width="100" />
          <el-table-column prop="biz_date" label="日期" width="120" />
          <el-table-column prop="source_doc_id" label="来源入库单" width="120" />
        </el-table>
      </template>

      <!-- 分析 -->
      <template v-else-if="active === 'analytics'">
        <el-table :data="list" size="small">
          <el-table-column prop="_kind" label="类型" width="80" />
          <el-table-column prop="supplier_name" label="供应商" min-width="140" />
          <el-table-column prop="supplier_code" label="编码" width="120" />
          <el-table-column prop="product_id" label="物料" width="90" />
          <el-table-column prop="purchase_qty" label="采购量" width="100" />
          <el-table-column prop="purchase_amount" label="采购额" width="100" />
          <el-table-column prop="avg_price" label="均价" width="90" />
          <el-table-column prop="pass_rate" label="合格率" width="90" />
          <el-table-column prop="return_rate" label="退货率" width="90" />
        </el-table>
      </template>

      <!-- 任务 -->
      <template v-else-if="active === 'tasks'">
        <el-card header="新建采购任务" class="mb">
          <el-form inline size="small">
            <el-form-item label="标题"><el-input v-model="taskForm.title" /></el-form-item>
            <el-form-item label="物料"><ProductSelect v-model="taskForm.product_id" :clearable="false" /></el-form-item>
            <el-form-item label="数量"><el-input-number v-model="taskForm.qty" :min="1" /></el-form-item>
            <el-form-item label="供应商"><SupplierSelect v-model="taskForm.supplier_id" :clearable="false" /></el-form-item>
            <el-form-item label="负责人"><UserSelect v-model="taskForm.assignee_id" :clearable="false" /></el-form-item>
            <el-form-item label="截止日期"><el-date-picker v-model="taskForm.due_date" type="date" value-format="YYYY-MM-DD" style="width:150px" /></el-form-item>
            <el-button type="primary" @click="createTask">新建</el-button>
          </el-form>
        </el-card>
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="140" />
          <el-table-column prop="title" label="标题" min-width="140" />
          <el-table-column prop="product_id" label="物料" width="80" />
          <el-table-column prop="qty" label="数量" width="90" />
          <el-table-column prop="assignee_id" label="负责人" width="90" />
          <el-table-column prop="due_date" label="截止" width="110" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="180">
            <template #default="{ row }">
              <el-button v-if="row.status==='open'" link type="primary" @click="assignTask(Number(row.id))">分配</el-button>
              <el-button v-if="row.status!=='done'" link type="success" @click="completeTask(Number(row.id))">完成</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <el-card v-if="detail" header="单据明细" style="margin-top:16px">
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
</style>
