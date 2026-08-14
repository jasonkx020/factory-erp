<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { inventoryApi, productApi } from '@erp/shared'
import {
  WarehouseSelect,
  SupplierSelect,
  SalesOrderSelect,
  StockTxnSelect,
  WorkshopSelect,
} from '../../components/select'
import WarehouseInboundView from '../warehouse/WarehouseInboundView.vue'
import StockLedgerView from './StockLedgerView.vue'
import BoxCodesView from './BoxCodesView.vue'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const balanceCols: MobileCardColumn[] = [
  { prop: 'product_name', label: '名称', primary: true },
  { prop: 'warehouse_name', label: '仓库' },
  { prop: 'product_code', label: '料号' },
  { prop: 'batch_no', label: '批次' },
  { prop: 'qty', label: '结存' },
]
const availabilityCols: MobileCardColumn[] = [
  { prop: 'product_id', label: '物料', primary: true },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'on_hand', label: '在手' },
  { prop: 'reserved', label: '占用' },
  { prop: 'available', label: '可用' },
]
const txnCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'doc_type', label: '类型' },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'status', label: '状态' },
  { prop: 'remark', label: '备注' },
]
const stocktakeCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'stocktake_type', label: '类型' },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'status', label: '状态' },
]
const hitCols: MobileCardColumn[] = [
  { prop: 'product_id', label: '物料', primary: true },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'qty', label: '结存' },
  { prop: 'min_qty', label: '下限' },
  { prop: 'max_qty', label: '上限' },
]
const alertRuleCols: MobileCardColumn[] = [
  { prop: 'id', label: 'ID', primary: true },
  { prop: 'product_id', label: '物料' },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'min_qty', label: '下限' },
  { prop: 'max_qty', label: '上限' },
  { prop: 'is_enabled', label: '启用' },
]
const inboundQcCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'product_id', label: '物料' },
  { prop: 'qty_check', label: '抽检' },
  { prop: 'qty_pass', label: '合格' },
  { prop: 'qty_fail', label: '不合格' },
  { prop: 'result', label: '结果' },
  { prop: 'status', label: '状态' },
]
const transitCols: MobileCardColumn[] = [
  { prop: 'id', label: 'ID', primary: true },
  { prop: 'product_id', label: '物料' },
  { prop: 'warehouse_id', label: '目标仓' },
  { prop: 'qty', label: '在途量' },
  { prop: 'transit_type', label: '类型' },
  { prop: 'source_doc_type', label: '来源' },
  { prop: 'status', label: '状态' },
]
const reservationCols: MobileCardColumn[] = [
  { prop: 'id', label: 'ID', primary: true },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'product_id', label: '物料' },
  { prop: 'qty', label: '占用' },
  { prop: 'source_doc_type', label: '来源类型' },
  { prop: 'source_doc_id', label: '来源ID' },
  { prop: 'status', label: '状态' },
]
const openingCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'status', label: '状态' },
  { prop: 'created_at', label: '创建时间' },
]
const peelCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'product_id', label: '物料' },
  { prop: 'peel_qty', label: '退皮量' },
  { prop: 'status', label: '状态' },
]
const payableCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'supplier_id', label: '供应商' },
  { prop: 'product_id', label: '物料' },
  { prop: 'amount', label: '金额' },
  { prop: 'status', label: '状态' },
]
const purchaseReturnCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'supplier_id', label: '供应商' },
  { prop: 'warehouse_id', label: '仓' },
  { prop: 'status', label: '状态' },
  { prop: 'reason', label: '原因' },
]

const route = useRoute()
const router = useRouter()

const TITLE_MAP: Record<string, string> = {
  balances: '库存查询',
  inbound: '仓管待入库',
  ledger: '地磅台账',
  shortage: '亏料预警',
  excess: '过量预警',
  'inbound-qc': '入库质检',
  stocktakes: '仓库盘点',
  'workshop-stocktakes': '车间盘点',
  'stocktake-records': '仓库盘点记录',
  peels: '销售退皮',
  transfers: '物料调拨耗用',
  assemble: '商品调价组装拆分',
  'to-payable': '物料转应付',
  'in-transits': '在途量统计',
  reservations: '待用量统计',
  availability: '可用量分析',
  openings: '期初入库',
  'stock-txns': '出入库记录汇总',
  'purchase-returns': '采购退货',
  boxes: '箱码管理',
}

const active = computed(() => String(route.params.section || 'balances'))
const title = computed(() => TITLE_MAP[active.value] || '库存管理')
const embedInbound = computed(() => active.value === 'inbound')
const embedLedger = computed(() => active.value === 'ledger')
const embedBoxes = computed(() => active.value === 'boxes')

const loading = ref(false)
const list = ref<Row[]>([])
const hits = ref<Row[]>([])
const detail = ref<Row | null>(null)
const products = ref<Row[]>([])
const transferTab = ref<'transfer' | 'consume'>('transfer')
const assembleTab = ref<'assemble' | 'price'>('assemble')

const transferCols = computed<MobileCardColumn[]>(() =>
  transferTab.value === 'transfer'
    ? [
        { prop: 'doc_no', label: '单号', primary: true },
        { prop: 'from_warehouse_id', label: '出仓' },
        { prop: 'to_warehouse_id', label: '入仓' },
        { prop: 'biz_date', label: '日期' },
        { prop: 'status', label: '状态' },
      ]
    : [
        { prop: 'doc_no', label: '单号', primary: true },
        { prop: 'warehouse_id', label: '仓' },
        { prop: 'biz_date', label: '日期' },
        { prop: 'status', label: '状态' },
      ],
)
const assembleCols = computed<MobileCardColumn[]>(() =>
  assembleTab.value === 'assemble'
    ? [
        { prop: 'doc_no', label: '单号', primary: true },
        { prop: 'biz_type', label: '类型' },
        { prop: 'status', label: '状态' },
      ]
    : [
        { prop: 'doc_no', label: '单号', primary: true },
        { prop: 'product_id', label: '物料' },
        { prop: 'old_price', label: '原价' },
        { prop: 'new_price', label: '新价' },
        { prop: 'status', label: '状态' },
      ],
)

const txnForm = reactive({
  warehouse_id: 1,
  product_id: 1,
  qty: 100,
  direction: 'in',
  doc_type: 'adjust',
  remark: '',
})
const stocktakeForm = reactive({
  warehouse_id: 1,
  workshop_id: 1,
  product_id: 1,
  count_qty: 0,
  remark: '',
})
const transferForm = reactive({
  from_warehouse_id: 1,
  to_warehouse_id: 2,
  product_id: 1,
  qty: 100,
  remark: '',
})
const consumeForm = reactive({
  warehouse_id: 1,
  product_id: 1,
  qty: 10,
  remark: '',
})
const alertForm = reactive({
  product_id: 0 as number,
  warehouse_id: 0 as number,
  min_qty: 100,
  max_qty: 10000,
})
const qcForm = reactive({ product_id: 1, qty_check: 100, stock_txn_id: 0 as number, remark: '' })
const openingForm = reactive({ warehouse_id: 1, product_id: 1, qty: 1000, remark: '' })
const peelForm = reactive({ warehouse_id: 1, product_id: 1, peel_qty: 10, sales_order_id: 0 as number })
const assembleForm = reactive({
  biz_type: 'assemble',
  warehouse_id: 1,
  parent_product_id: 1,
  parent_qty: 1,
  child_product_id: 1,
  child_qty: 2,
})
const priceForm = reactive({ product_id: 1, old_price: 0, new_price: 1, remark: '' })
const payableForm = reactive({
  supplier_id: 1,
  product_id: 1,
  qty: 10,
  amount: 100,
  consume_txn_id: 0 as number,
})

async function loadMeta() {
  const p = await productApi.list()
  products.value = ((p.data as { list?: Row[] })?.list) || []
  if (products.value[0]) {
    const pid = Number(products.value[0].id)
    txnForm.product_id = pid
    stocktakeForm.product_id = pid
    transferForm.product_id = pid
    consumeForm.product_id = pid
    alertForm.product_id = pid
    qcForm.product_id = pid
    openingForm.product_id = pid
    peelForm.product_id = pid
    assembleForm.parent_product_id = pid
    assembleForm.child_product_id = pid
    priceForm.product_id = pid
    payableForm.product_id = pid
  }
}

async function refresh() {
  if (embedInbound.value || embedLedger.value || embedBoxes.value) return
  loading.value = true
  detail.value = null
  hits.value = []
  try {
    let res
    switch (active.value) {
      case 'balances':
        res = await inventoryApi.balances()
        break
      case 'availability':
        res = await inventoryApi.availability()
        break
      case 'stock-txns':
        res = await inventoryApi.listTxns()
        break
      case 'stocktakes':
        res = await inventoryApi.stocktakes(false)
        break
      case 'workshop-stocktakes':
        res = await inventoryApi.stocktakes(true)
        break
      case 'stocktake-records':
        res = await inventoryApi.stocktakeRecords()
        break
      case 'transfers':
        res =
          transferTab.value === 'transfer'
            ? await inventoryApi.transfers()
            : await inventoryApi.consumes()
        break
      case 'shortage': {
        res = await inventoryApi.alertShortage()
        hits.value = ((res.data as { hits?: Row[] })?.hits) || []
        break
      }
      case 'excess': {
        res = await inventoryApi.alertExcess()
        hits.value = ((res.data as { hits?: Row[] })?.hits) || []
        break
      }
      case 'inbound-qc':
        res = await inventoryApi.inboundQcs()
        break
      case 'in-transits':
        res = await inventoryApi.inTransits()
        break
      case 'reservations':
        res = await inventoryApi.reservations()
        break
      case 'openings':
        res = await inventoryApi.openings()
        break
      case 'assemble':
        res =
          assembleTab.value === 'assemble'
            ? await inventoryApi.assembleSplits()
            : await inventoryApi.priceAdjusts()
        break
      case 'peels':
        res = await inventoryApi.peelReturns()
        break
      case 'to-payable':
        res = await inventoryApi.materialToPayables()
        break
      case 'purchase-returns':
        res = await inventoryApi.purchaseReturns()
        break
      default:
        res = await inventoryApi.balances()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function createTxn() {
  const res = await inventoryApi.createTxn({
    warehouse_id: txnForm.warehouse_id,
    doc_type: txnForm.doc_type,
    remark: txnForm.remark,
    lines: [{ product_id: txnForm.product_id, qty: txnForm.qty, direction: txnForm.direction }],
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已建单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postTxn(id: number) {
  const res = await inventoryApi.postTxn(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已过账')
  await refresh()
}

async function cancelTxn(id: number) {
  const res = await inventoryApi.cancelTxn(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已作废')
  await refresh()
}

async function openTxn(id: number) {
  const res = await inventoryApi.getTxn(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

async function createStocktake() {
  const workshop = active.value === 'workshop-stocktakes'
  const res = await inventoryApi.createStocktake(
    {
      warehouse_id: stocktakeForm.warehouse_id,
      workshop_id: stocktakeForm.workshop_id,
      product_id: stocktakeForm.product_id,
      count_qty: stocktakeForm.count_qty,
      remark: stocktakeForm.remark,
    },
    workshop,
  )
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`盘点单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function submitStocktake(id: number) {
  const workshop = active.value === 'workshop-stocktakes'
  const res = await inventoryApi.submitStocktake(id, workshop)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交')
  await refresh()
}

async function postStocktake(id: number) {
  const workshop = active.value === 'workshop-stocktakes'
  const res = await inventoryApi.postStocktake(id, workshop)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('盘点过账完成')
  await refresh()
}

async function openStocktake(id: number) {
  const workshop = active.value === 'workshop-stocktakes'
  const res = await inventoryApi.getStocktake(id, workshop)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

async function createTransfer() {
  const res = await inventoryApi.createTransfer({ ...transferForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`调拨单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postTransfer(id: number) {
  const res = await inventoryApi.postTransfer(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('调拨过账完成')
  await refresh()
}

async function createConsume() {
  const res = await inventoryApi.createConsume({ ...consumeForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`耗用单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postConsume(id: number) {
  const res = await inventoryApi.postConsume(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('耗用过账完成')
  await refresh()
}

async function saveAlert() {
  const body = {
    product_id: alertForm.product_id || undefined,
    warehouse_id: alertForm.warehouse_id || undefined,
    min_qty: alertForm.min_qty,
    max_qty: alertForm.max_qty,
  }
  const res =
    active.value === 'shortage'
      ? await inventoryApi.upsertAlertShortage(body)
      : await inventoryApi.upsertAlertExcess(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('规则已保存')
  await refresh()
}

async function createQc() {
  const res = await inventoryApi.createInboundQc({ ...qcForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`质检单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function passQc(id: number) {
  const res = await inventoryApi.passInboundQc(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('质检通过')
  await refresh()
}

async function failQc(id: number) {
  const res = await inventoryApi.failInboundQc(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已判不合格')
  await refresh()
}

async function createOpening() {
  const res = await inventoryApi.createOpening({ ...openingForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`期初单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postOpening(id: number) {
  const res = await inventoryApi.postOpening(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('期初过账完成')
  await refresh()
}

async function releaseRsv(id: number) {
  const res = await inventoryApi.releaseReservation(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已释放占用')
  await refresh()
}

async function createPeel() {
  const res = await inventoryApi.createPeelReturn({ ...peelForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`退皮单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postPeel(id: number) {
  const res = await inventoryApi.postPeelReturn(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('退皮入库完成')
  await refresh()
}

async function createAssemble() {
  const res = await inventoryApi.createAssembleSplit({ ...assembleForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`组装/拆分 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function postAssemble(id: number) {
  const res = await inventoryApi.postAssembleSplit(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('过账完成')
  await refresh()
}

async function createPrice() {
  const res = await inventoryApi.createPriceAdjust({ ...priceForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`调价单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function createPayable() {
  const res = await inventoryApi.createMaterialToPayable({ ...payableForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`转应付 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function submitPayable(id: number) {
  const res = await inventoryApi.submitMaterialToPayable(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交')
  await refresh()
}

onMounted(async () => {
  await loadMeta()
  await refresh()
})
watch([active, transferTab, assembleTab], refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <el-button size="small" @click="refresh" v-if="!embedInbound && !embedLedger && !embedBoxes">刷新</el-button>
    </div>

    <WarehouseInboundView v-if="embedInbound" />
    <StockLedgerView v-else-if="embedLedger" />
    <BoxCodesView v-else-if="embedBoxes" />

    <template v-else>
      <!-- 库存查询 -->
      <template v-if="active === 'balances'">
        <TableOrCards :data="list" :loading="loading" :columns="balanceCols">
          <el-table :data="list" size="small">
            <el-table-column prop="warehouse_name" label="仓库" width="120" />
            <el-table-column prop="product_code" label="料号" width="110" />
            <el-table-column prop="product_name" label="名称" min-width="160" />
            <el-table-column prop="batch_no" label="批次" width="100" />
            <el-table-column prop="qty" label="结存" width="100" />
          </el-table>
        </TableOrCards>
      </template>

      <!-- 可用量 -->
      <template v-else-if="active === 'availability'">
        <TableOrCards :data="list" :loading="loading" :columns="availabilityCols">
          <el-table :data="list" size="small">
            <el-table-column prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="on_hand" label="在手" width="100" />
            <el-table-column prop="reserved" label="占用" width="100" />
            <el-table-column prop="available" label="可用" width="100" />
          </el-table>
        </TableOrCards>
      </template>

      <!-- 出入库 -->
      <template v-else-if="active === 'stock-txns'">
        <el-card header="新建出入库（草稿→过账写结存）" class="mb">
          <el-form inline size="small">
            <el-form-item label="类型">
              <el-select v-model="txnForm.doc_type" style="width:140px">
                <el-option label="调整" value="adjust" />
                <el-option label="采购入库" value="purchase_in" />
                <el-option label="销售出库" value="sales_out" />
                <el-option label="生产入库" value="produce_in" />
                <el-option label="领料出库" value="requisition_out" />
              </el-select>
            </el-form-item>
            <el-form-item label="仓库"><WarehouseSelect v-model="txnForm.warehouse_id" /></el-form-item>
            <el-form-item label="物料">
              <el-select v-model="txnForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="txnForm.qty" :min="0.01" /></el-form-item>
            <el-form-item label="方向">
              <el-select v-model="txnForm.direction" style="width:90px">
                <el-option label="入" value="in" />
                <el-option label="出" value="out" />
              </el-select>
            </el-form-item>
            <el-button type="primary" @click="createTxn">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="txnCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="160" />
            <el-table-column prop="doc_type" label="类型" width="120" />
            <el-table-column prop="warehouse_id" label="仓" width="70" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="remark" label="备注" min-width="140" />
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button link @click="openTxn(Number(row.id))">明细</el-button>
                <el-button v-if="row.status==='draft'" link type="primary" @click="postTxn(Number(row.id))">过账</el-button>
                <el-button v-if="row.status==='draft'" link type="danger" @click="cancelTxn(Number(row.id))">作废</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link @click="openTxn(Number(row.id))">明细</el-button>
            <el-button v-if="row.status==='draft'" link type="primary" @click="postTxn(Number(row.id))">过账</el-button>
            <el-button v-if="row.status==='draft'" link type="danger" @click="cancelTxn(Number(row.id))">作废</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 盘点 -->
      <template v-else-if="active === 'stocktakes' || active === 'workshop-stocktakes'">
        <el-card :header="active === 'workshop-stocktakes' ? '新建车间盘点' : '新建仓库盘点'" class="mb">
          <el-form inline size="small">
            <el-form-item label="仓库"><WarehouseSelect v-model="stocktakeForm.warehouse_id" /></el-form-item>
            <el-form-item v-if="active==='workshop-stocktakes'" label="车间">
              <WorkshopSelect v-model="stocktakeForm.workshop_id" />
            </el-form-item>
            <el-form-item label="物料">
              <el-select v-model="stocktakeForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="实盘"><el-input-number v-model="stocktakeForm.count_qty" :min="0" /></el-form-item>
            <el-button type="primary" @click="createStocktake">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="stocktakeCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="stocktake_type" label="类型" width="100" />
            <el-table-column prop="warehouse_id" label="仓" width="70" />
            <el-table-column prop="biz_date" label="日期" width="110" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="260" fixed="right">
              <template #default="{ row }">
                <el-button link @click="openStocktake(Number(row.id))">明细</el-button>
                <el-button v-if="row.status==='draft'" link type="primary" @click="submitStocktake(Number(row.id))">提交</el-button>
                <el-button v-if="row.status==='draft' || row.status==='submitted'" link type="success" @click="postStocktake(Number(row.id))">过账</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link @click="openStocktake(Number(row.id))">明细</el-button>
            <el-button v-if="row.status==='draft'" link type="primary" @click="submitStocktake(Number(row.id))">提交</el-button>
            <el-button v-if="row.status==='draft' || row.status==='submitted'" link type="success" @click="postStocktake(Number(row.id))">过账</el-button>
          </template>
        </TableOrCards>
      </template>

      <template v-else-if="active === 'stocktake-records'">
        <TableOrCards :data="list" :loading="loading" :columns="stocktakeCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="stocktake_type" label="类型" width="100" />
            <el-table-column prop="warehouse_id" label="仓" width="70" />
            <el-table-column prop="biz_date" label="日期" width="110" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link @click="openStocktake(Number(row.id))">明细</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link @click="openStocktake(Number(row.id))">明细</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 调拨耗用 -->
      <template v-else-if="active === 'transfers'">
        <el-radio-group v-model="transferTab" size="small" class="mb">
          <el-radio-button value="transfer">仓间调拨</el-radio-button>
          <el-radio-button value="consume">物料耗用</el-radio-button>
        </el-radio-group>
        <el-card v-if="transferTab==='transfer'" header="新建调拨（过账：出原仓 + 入目标仓）" class="mb">
          <el-form inline size="small">
            <el-form-item label="出仓库"><WarehouseSelect v-model="transferForm.from_warehouse_id" /></el-form-item>
            <el-form-item label="入仓库"><WarehouseSelect v-model="transferForm.to_warehouse_id" /></el-form-item>
            <el-form-item label="物料">
              <el-select v-model="transferForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="transferForm.qty" :min="0.01" /></el-form-item>
            <el-button type="primary" @click="createTransfer">新建</el-button>
          </el-form>
        </el-card>
        <el-card v-else header="新建耗用（过账出库）" class="mb">
          <el-form inline size="small">
            <el-form-item label="仓库"><WarehouseSelect v-model="consumeForm.warehouse_id" /></el-form-item>
            <el-form-item label="物料">
              <el-select v-model="consumeForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="consumeForm.qty" :min="0.01" /></el-form-item>
            <el-button type="primary" @click="createConsume">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="transferCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column v-if="transferTab==='transfer'" prop="from_warehouse_id" label="出仓" width="80" />
            <el-table-column v-if="transferTab==='transfer'" prop="to_warehouse_id" label="入仓" width="80" />
            <el-table-column v-else prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="biz_date" label="日期" width="110" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button
                  v-if="row.status==='draft'"
                  link
                  type="primary"
                  @click="transferTab==='transfer' ? postTransfer(Number(row.id)) : postConsume(Number(row.id))"
                >过账</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button
              v-if="row.status==='draft'"
              link
              type="primary"
              @click="transferTab==='transfer' ? postTransfer(Number(row.id)) : postConsume(Number(row.id))"
            >过账</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 预警 -->
      <template v-else-if="active === 'shortage' || active === 'excess'">
        <el-card :header="active==='shortage' ? '亏料规则（低于 min_qty 触发）' : '过量规则（高于 max_qty 触发）'" class="mb">
          <el-form inline size="small">
            <el-form-item label="物料">
              <el-select v-model="alertForm.product_id" style="width:160px" clearable filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="仓库">
              <WarehouseSelect v-model="alertForm.warehouse_id" allow-zero zero-label="全部" />
            </el-form-item>
            <el-form-item v-if="active==='shortage'" label="下限"><el-input-number v-model="alertForm.min_qty" :min="0" /></el-form-item>
            <el-form-item v-else label="上限"><el-input-number v-model="alertForm.max_qty" :min="0" /></el-form-item>
            <el-button type="primary" @click="saveAlert">保存规则</el-button>
          </el-form>
        </el-card>
        <h4>当前命中</h4>
        <TableOrCards :data="hits" :loading="loading" :columns="hitCols" class="mb">
          <el-table :data="hits" size="small" class="mb">
            <el-table-column prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="qty" label="结存" width="100" />
            <el-table-column prop="min_qty" label="下限" width="100" />
            <el-table-column prop="max_qty" label="上限" width="100" />
          </el-table>
        </TableOrCards>
        <h4>规则列表</h4>
        <TableOrCards :data="list" :loading="loading" :columns="alertRuleCols">
          <el-table :data="list" size="small">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="min_qty" label="下限" width="100" />
            <el-table-column prop="max_qty" label="上限" width="100" />
            <el-table-column prop="is_enabled" label="启用" width="80" />
          </el-table>
        </TableOrCards>
      </template>

      <!-- 入库质检 -->
      <template v-else-if="active === 'inbound-qc'">
        <el-card header="新建入库质检" class="mb">
          <el-form inline size="small">
            <el-form-item label="物料">
              <el-select v-model="qcForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="抽检量"><el-input-number v-model="qcForm.qty_check" :min="0" /></el-form-item>
            <el-form-item label="库存流水"><StockTxnSelect v-model="qcForm.stock_txn_id" /></el-form-item>
            <el-button type="primary" @click="createQc">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="inboundQcCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="qty_check" label="抽检" width="90" />
            <el-table-column prop="qty_pass" label="合格" width="90" />
            <el-table-column prop="qty_fail" label="不合格" width="90" />
            <el-table-column prop="result" label="结果" width="80" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="160">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="success" @click="passQc(Number(row.id))">通过</el-button>
                <el-button v-if="row.status==='draft'" link type="danger" @click="failQc(Number(row.id))">不合格</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="success" @click="passQc(Number(row.id))">通过</el-button>
            <el-button v-if="row.status==='draft'" link type="danger" @click="failQc(Number(row.id))">不合格</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 在途 -->
      <template v-else-if="active === 'in-transits'">
        <TableOrCards :data="list" :loading="loading" :columns="transitCols">
          <el-table :data="list" size="small">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="warehouse_id" label="目标仓" width="90" />
            <el-table-column prop="qty" label="在途量" width="100" />
            <el-table-column prop="transit_type" label="类型" width="100" />
            <el-table-column prop="source_doc_type" label="来源" width="120" />
            <el-table-column prop="status" label="状态" width="90" />
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
        </TableOrCards>
      </template>

      <!-- 待用占用 -->
      <template v-else-if="active === 'reservations'">
        <TableOrCards :data="list" :loading="loading" :columns="reservationCols">
          <el-table :data="list" size="small">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="qty" label="占用" width="100" />
            <el-table-column prop="source_doc_type" label="来源类型" width="120" />
            <el-table-column prop="source_doc_id" label="来源ID" width="90" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='active'" link type="warning" @click="releaseRsv(Number(row.id))">释放</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='active'" link type="warning" @click="releaseRsv(Number(row.id))">释放</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 期初 -->
      <template v-else-if="active === 'openings'">
        <el-card header="期初入库（过账写结存）" class="mb">
          <el-form inline size="small">
            <el-form-item label="仓库"><WarehouseSelect v-model="openingForm.warehouse_id" /></el-form-item>
            <el-form-item label="物料">
              <el-select v-model="openingForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="openingForm.qty" :min="0.01" /></el-form-item>
            <el-button type="primary" @click="createOpening">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="openingCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="created_at" label="创建时间" width="160" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="primary" @click="postOpening(Number(row.id))">过账</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="postOpening(Number(row.id))">过账</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 退皮 -->
      <template v-else-if="active === 'peels'">
        <el-card header="销售退皮入库" class="mb">
          <el-form inline size="small">
            <el-form-item label="仓库"><WarehouseSelect v-model="peelForm.warehouse_id" /></el-form-item>
            <el-form-item label="物料">
              <el-select v-model="peelForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="退皮量"><el-input-number v-model="peelForm.peel_qty" :min="0.01" /></el-form-item>
            <el-form-item label="销售订单"><SalesOrderSelect v-model="peelForm.sales_order_id" /></el-form-item>
            <el-button type="primary" @click="createPeel">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="peelCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="peel_qty" label="退皮量" width="100" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="primary" @click="postPeel(Number(row.id))">过账入库</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="postPeel(Number(row.id))">过账入库</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 组装调价 -->
      <template v-else-if="active === 'assemble'">
        <el-radio-group v-model="assembleTab" size="small" class="mb">
          <el-radio-button value="assemble">组装/拆分</el-radio-button>
          <el-radio-button value="price">商品调价</el-radio-button>
        </el-radio-group>
        <el-card v-if="assembleTab==='assemble'" header="组装（子料出+母料入）/ 拆分（母料出+子料入）" class="mb">
          <el-form inline size="small">
            <el-form-item label="类型">
              <el-select v-model="assembleForm.biz_type" style="width:110px">
                <el-option label="组装" value="assemble" />
                <el-option label="拆分" value="split" />
              </el-select>
            </el-form-item>
            <el-form-item label="仓库"><WarehouseSelect v-model="assembleForm.warehouse_id" /></el-form-item>
            <el-form-item label="母料">
              <el-select v-model="assembleForm.parent_product_id" style="width:140px" filterable>
                <el-option v-for="p in products" :key="'p'+p.id" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="母量"><el-input-number v-model="assembleForm.parent_qty" :min="0.01" /></el-form-item>
            <el-form-item label="子料">
              <el-select v-model="assembleForm.child_product_id" style="width:140px" filterable>
                <el-option v-for="p in products" :key="'c'+p.id" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="子量"><el-input-number v-model="assembleForm.child_qty" :min="0.01" /></el-form-item>
            <el-button type="primary" @click="createAssemble">新建</el-button>
          </el-form>
        </el-card>
        <el-card v-else header="商品调价" class="mb">
          <el-form inline size="small">
            <el-form-item label="物料">
              <el-select v-model="priceForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="原价"><el-input-number v-model="priceForm.old_price" :min="0" :step="0.01" /></el-form-item>
            <el-form-item label="新价"><el-input-number v-model="priceForm.new_price" :min="0" :step="0.01" /></el-form-item>
            <el-button type="primary" @click="createPrice">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="assembleCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column v-if="assembleTab==='assemble'" prop="biz_type" label="类型" width="90" />
            <el-table-column v-if="assembleTab==='price'" prop="product_id" label="物料" width="90" />
            <el-table-column v-if="assembleTab==='price'" prop="old_price" label="原价" width="90" />
            <el-table-column v-if="assembleTab==='price'" prop="new_price" label="新价" width="90" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button
                  v-if="assembleTab==='assemble' && row.status==='draft'"
                  link
                  type="primary"
                  @click="postAssemble(Number(row.id))"
                >过账</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button
              v-if="assembleTab==='assemble' && row.status==='draft'"
              link
              type="primary"
              @click="postAssemble(Number(row.id))"
            >过账</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 转应付 -->
      <template v-else-if="active === 'to-payable'">
        <el-card header="物料转应付" class="mb">
          <el-form inline size="small">
            <el-form-item label="供应商"><SupplierSelect v-model="payableForm.supplier_id" /></el-form-item>
            <el-form-item label="物料">
              <el-select v-model="payableForm.product_id" style="width:160px" filterable>
                <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量"><el-input-number v-model="payableForm.qty" :min="0" /></el-form-item>
            <el-form-item label="金额"><el-input-number v-model="payableForm.amount" :min="0" :step="0.01" /></el-form-item>
            <el-button type="primary" @click="createPayable">新建</el-button>
          </el-form>
        </el-card>
        <TableOrCards :data="list" :loading="loading" :columns="payableCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="supplier_id" label="供应商" width="90" />
            <el-table-column prop="product_id" label="物料" width="90" />
            <el-table-column prop="amount" label="金额" width="100" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status==='draft'" link type="primary" @click="submitPayable(Number(row.id))">提交</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="submitPayable(Number(row.id))">提交</el-button>
          </template>
        </TableOrCards>
      </template>

      <!-- 采购退货视图 -->
      <template v-else-if="active === 'purchase-returns'">
        <el-alert
          class="mb"
          type="info"
          :closable="false"
          title="库存侧只读视图；办理退货过账请使用采购管理 → 采购退货"
        />
        <el-button class="mb" type="primary" size="small" @click="router.push('/purchase/hub/returns')">前往采购退货</el-button>
        <TableOrCards :data="list" :loading="loading" :columns="purchaseReturnCols">
          <el-table :data="list" size="small">
            <el-table-column prop="doc_no" label="单号" width="150" />
            <el-table-column prop="supplier_id" label="供应商" width="90" />
            <el-table-column prop="warehouse_id" label="仓" width="80" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="160" />
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
          </template>
        </TableOrCards>
      </template>
    </template>

    <el-drawer :model-value="!!detail" title="单据明细" size="420px" @update:model-value="(v: boolean) => { if (!v) detail = null }">
      <pre v-if="detail" class="trace">{{ detail }}</pre>
    </el-drawer>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.head h2 { margin: 0; font-size: 18px; }
.mb { margin-bottom: 12px; }
.trace {
  background: #f6f8fa;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
