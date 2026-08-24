<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { reportApi } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const SECTIONS = new Set([
  'production-board',
  'live',
  'warehouse',
  'daily',
  'inbound-daily',
  'piecework-daily',
  'yield-analysis',
  'stock-ledger',
  'trace-progress',
  'farmer-settlement-summary',
  'payroll-reconcile',
  'cost-period-summary',
])

const TITLE_MAP: Record<string, string> = {
  'production-board': '生产看板',
  live: '生产实况',
  warehouse: '三仓库存概览',
  daily: '日经营快照',
  'inbound-daily': '原料入场日报',
  'piecework-daily': '计件日结汇总',
  'yield-analysis': '工序扣损收率分析',
  'stock-ledger': '收发存明细',
  'trace-progress': '溯源批进度查询',
  'farmer-settlement-summary': '农户结算对账汇总',
  'payroll-reconcile': '薪酬核算对账',
  'cost-period-summary': '成本期间汇总',
}

const KPI_SECTIONS = new Set([
  'production-board',
  'live',
  'warehouse',
  'daily',
  'piecework-daily',
  'farmer-settlement-summary',
  'payroll-reconcile',
  'cost-period-summary',
])

const warehouseCols: MobileCardColumn[] = [
  { prop: 'warehouse_name', label: '仓库', primary: true },
  { prop: 'warehouse_type', label: '类型' },
  { prop: 'qty_kg', label: '结存kg' },
  { prop: 'sku_count', label: 'SKU数' },
]
const inboundCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '过磅单号', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'gross_weight', label: '毛重' },
  { prop: 'deduct_weight', label: '扣损' },
  { prop: 'net_weight', label: '净重' },
  { prop: 'qc_result', label: '质检' },
  { prop: 'status', label: '状态' },
]
const pieceworkCols: MobileCardColumn[] = [
  { prop: 'worker_name', label: '工人', primary: true },
  { prop: 'process_name', label: '工序' },
  { prop: 'qty', label: '完成kg' },
  { prop: 'amount', label: '金额' },
]
const yieldCols: MobileCardColumn[] = [
  { prop: 'process_name', label: '工序', primary: true },
  { prop: 'trace_count', label: '溯源批数' },
  { prop: 'input_kg', label: '投入kg' },
  { prop: 'output_kg', label: '产出kg' },
  { prop: 'loss_kg', label: '损耗kg' },
  { prop: 'loss_rate', label: '损耗率' },
]
const stockLedgerCols: MobileCardColumn[] = [
  { prop: 'warehouse_name', label: '仓库', primary: true },
  { prop: 'product_name', label: '产品' },
  { prop: 'product_code', label: '编码' },
  { prop: 'qty', label: '数量' },
  { prop: 'amount', label: '金额' },
]
const traceCols: MobileCardColumn[] = [
  { prop: 'trace_code', label: '溯源码', primary: true },
  { prop: 'status', label: '状态' },
  { prop: 'input_kg', label: '投入kg' },
  { prop: 'output_kg', label: '产出kg' },
  { prop: 'loss_rate', label: '损耗率' },
  { prop: 'open_issues', label: '在制领料' },
]
const settlementCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '结算单号', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'biz_date', label: '业务日' },
  { prop: 'net_weight', label: '净重' },
  { prop: 'amount', label: '金额' },
  { prop: 'status', label: '状态' },
  { prop: 'trace_code', label: '溯源码' },
]
const payrollReconcileCols: MobileCardColumn[] = [
  { prop: 'worker_name', label: '工人', primary: true },
  { prop: 'emp_no', label: '工号' },
  { prop: 'sheet_piece_amount', label: '工资单计件' },
  { prop: 'piecework_amount', label: '计件汇总' },
  { prop: 'diff', label: '差异' },
  { prop: 'sheet_total', label: '工资单合计' },
]
const costPeriodCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '成本单', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'product_name', label: '产品' },
  { prop: 'material_cost', label: '物料' },
  { prop: 'labor_cost', label: '人工' },
  { prop: 'overhead', label: '制造' },
  { prop: 'total_cost', label: '合计' },
  { prop: 'status', label: '状态' },
]
const flowCols: MobileCardColumn[] = [
  { prop: 'event_type', label: '事件', primary: true },
  { prop: 'trace_code', label: '溯源码' },
  { prop: 'process_name', label: '工序' },
  { prop: 'worker_name', label: '工人' },
  { prop: 'kg', label: 'kg' },
  { prop: 'created_at', label: '时间' },
]
const wipCols: MobileCardColumn[] = [
  { prop: 'process_name', label: '工序', primary: true },
  { prop: 'issue_count', label: '领料单数' },
  { prop: 'wip_kg', label: '在制kg' },
]

const route = useRoute()
const active = computed(() => {
  const s = String(route.params.section || 'production-board')
  return SECTIONS.has(s) ? s : 'production-board'
})
const title = computed(() => TITLE_MAP[active.value] || '统计报表')
const loading = ref(false)
const list = ref<Row[]>([])
const kpis = ref<Row[]>([])
const summary = ref<Row | null>(null)
const extraRows = ref<Row[]>([])
const asOf = ref('')
const bizDate = ref(new Date().toISOString().slice(0, 10))
const periodMonth = ref(new Date().toISOString().slice(0, 7))
const dateSections = new Set(['daily', 'inbound-daily', 'piecework-daily', 'yield-analysis', 'farmer-settlement-summary'])
const periodSections = new Set(['payroll-reconcile', 'cost-period-summary'])

function fmt(v: unknown) {
  if (v == null || v === '') return '—'
  if (typeof v === 'number') return Number.isInteger(v) ? String(v) : v.toFixed(2)
  return String(v)
}

function toKpisFromSummary(s: Row, labels: Record<string, string>) {
  return Object.entries(labels).map(([key, titleText]) => ({
    key,
    title: titleText,
    value: s[key],
  }))
}

async function refresh() {
  loading.value = true
  list.value = []
  kpis.value = []
  summary.value = null
  extraRows.value = []
  asOf.value = ''
  try {
    const sec = active.value
    let res: { code: number; msg?: string; data?: unknown }

    if (sec === 'production-board') {
      res = await reportApi.production()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      kpis.value = (data.list as Row[]) || []
      extraRows.value = (data.wip_by_process as Row[]) || []
      asOf.value = String(data.as_of || '')
      list.value = kpis.value
    } else if (sec === 'live') {
      res = await reportApi.live()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      kpis.value = (data.list as Row[]) || []
      extraRows.value = (data.recent_flow as Row[]) || []
      asOf.value = String(data.as_of || '')
      list.value = kpis.value
    } else if (sec === 'warehouse') {
      res = await reportApi.warehouse()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      list.value = (data.list as Row[]) || []
      summary.value = (data.summary as Row) || null
      asOf.value = String(data.as_of || '')
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          total_qty_kg: '总库存kg',
          shortage_alerts: '亏料预警',
          excess_alerts: '过量预警',
        })
      }
    } else if (sec === 'daily') {
      res = await reportApi.daily(bizDate.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      const s = (data.summary as Row) || ((data.list as Row[]) || [])[0] || {}
      summary.value = s
      kpis.value = toKpisFromSummary(s, {
        inbound_net_kg: '入场净重kg',
        inbound_tickets: '过磅单数',
        production_output_kg: '产出kg',
        flow_log_kg: '工序过账kg',
        piecework_amount: '计件支出',
        farmer_payable: '农户应付',
        farmer_paid: '农户已付',
        stock_in: '入库量',
        stock_out: '出库量',
      })
      list.value = (data.list as Row[]) || [s]
    } else if (sec === 'inbound-daily') {
      res = await reportApi.inboundDaily(bizDate.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          ticket_count: '过磅单数',
          gross_kg: '毛重kg',
          deduct_kg: '扣损kg',
          net_kg: '净重kg',
          settlement_amount: '结算金额',
          settlement_pending: '待付金额',
        })
      }
    } else if (sec === 'piecework-daily') {
      res = await reportApi.pieceworkDaily(bizDate.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          worker_count: '工人数',
          total_qty_kg: '完成kg',
          total_amount: '计件金额',
          flow_log_rows: '流水笔数',
        })
      }
    } else if (sec === 'yield-analysis') {
      res = await reportApi.yieldAnalysis(bizDate.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'stock-ledger') {
      res = await reportApi.stockLedger()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'trace-progress') {
      res = await reportApi.traceProgress()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'farmer-settlement-summary') {
      res = await reportApi.farmerSettlementSummary(bizDate.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          doc_count: '结算单数',
          total_amount: '结算总额',
          paid_amount: '已付',
          pending_amount: '待付',
        })
      }
    } else if (sec === 'payroll-reconcile') {
      res = await reportApi.payrollReconcile(periodMonth.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          sheet_no: '工资单号',
          worker_count: '对账人数',
          diff_count: '有差异人数',
          sheet_piece_total: '工资单计件合计',
          piecework_total: '计件汇总合计',
          diff_total: '差异合计',
        })
      }
    } else if (sec === 'cost-period-summary') {
      res = await reportApi.costPeriodSummary(periodMonth.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          doc_count: '成本单数',
          material_total: '物料合计',
          labor_total: '人工合计',
          overhead_total: '制造费用',
          cost_total: '总成本',
        })
      }
    }
  } finally {
    loading.value = false
  }
}

const showKpis = computed(() => KPI_SECTIONS.has(active.value) && kpis.value.length > 0)

onMounted(refresh)
watch(active, refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <div>
        <h2>{{ title }}</h2>
        <div v-if="asOf" class="sub">数据截止：{{ asOf }}</div>
      </div>
      <div class="actions">
        <template v-if="dateSections.has(active)">
          <el-date-picker v-model="bizDate" type="date" value-format="YYYY-MM-DD" size="small" style="width: 150px" />
          <el-button size="small" type="primary" @click="refresh">按日查询</el-button>
        </template>
        <template v-else-if="periodSections.has(active)">
          <el-date-picker v-model="periodMonth" type="month" value-format="YYYY-MM" size="small" style="width: 150px" />
          <el-button size="small" type="primary" @click="refresh">按期间查询</el-button>
        </template>
        <el-button size="small" @click="refresh">刷新</el-button>
      </div>
    </div>

    <el-row v-if="showKpis" :gutter="12" class="mb">
      <el-col v-for="k in kpis" :key="String(k.key || k.title)" :xs="24" :sm="8" :md="6">
        <el-card shadow="never" class="kpi">
          <div class="stat-label">{{ k.title }}</div>
          <div class="stat-value">{{ fmt(k.value) }}</div>
        </el-card>
      </el-col>
    </el-row>

    <template v-if="active === 'production-board' || active === 'live'">
      <el-empty v-if="!kpis.length" description="暂无指标数据" />
      <template v-if="active === 'production-board' && extraRows.length">
        <h4>工序在制</h4>
        <TableOrCards :data="extraRows" :loading="loading" :columns="wipCols" empty-text="暂无在制">
          <el-table :data="extraRows" size="small" empty-text="暂无在制">
            <el-table-column prop="process_name" label="工序" min-width="120" />
            <el-table-column prop="issue_count" label="领料单数" width="100" />
            <el-table-column prop="wip_kg" label="在制kg" width="120" />
          </el-table>
        </TableOrCards>
      </template>
      <template v-if="active === 'live' && extraRows.length">
        <h4>近1小时流水</h4>
        <TableOrCards :data="extraRows" :loading="loading" :columns="flowCols" empty-text="暂无流水">
          <el-table :data="extraRows" size="small" empty-text="暂无流水">
            <el-table-column prop="event_type" label="事件" width="100" />
            <el-table-column prop="trace_code" label="溯源码" width="140" />
            <el-table-column prop="process_name" label="工序" width="100" />
            <el-table-column prop="worker_name" label="工人" width="100" />
            <el-table-column prop="kg" label="kg" width="90" />
            <el-table-column prop="created_at" label="时间" min-width="160" />
          </el-table>
        </TableOrCards>
      </template>
    </template>

    <template v-else-if="active === 'warehouse'">
      <TableOrCards :data="list" :loading="loading" :columns="warehouseCols" empty-text="暂无库存数据">
        <el-table :data="list" size="small" empty-text="暂无库存数据">
          <el-table-column prop="warehouse_name" label="仓库" min-width="120" />
          <el-table-column prop="warehouse_type" label="类型" width="100" />
          <el-table-column prop="qty_kg" label="结存kg" width="120" />
          <el-table-column prop="sku_count" label="SKU数" width="90" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'daily'">
      <el-descriptions v-if="summary" :column="3" border size="small" title="当日汇总">
        <el-descriptions-item label="业务日">{{ summary.biz_date }}</el-descriptions-item>
        <el-descriptions-item label="入场净重">{{ fmt(summary.inbound_net_kg) }} kg</el-descriptions-item>
        <el-descriptions-item label="过磅单">{{ fmt(summary.inbound_tickets) }}</el-descriptions-item>
        <el-descriptions-item label="产出">{{ fmt(summary.production_output_kg) }} kg</el-descriptions-item>
        <el-descriptions-item label="计件支出">{{ fmt(summary.piecework_amount) }}</el-descriptions-item>
        <el-descriptions-item label="农户应付/已付">{{ fmt(summary.farmer_payable) }} / {{ fmt(summary.farmer_paid) }}</el-descriptions-item>
        <el-descriptions-item label="出入库">入 {{ fmt(summary.stock_in) }} / 出 {{ fmt(summary.stock_out) }}</el-descriptions-item>
      </el-descriptions>
    </template>

    <template v-else-if="active === 'inbound-daily'">
      <TableOrCards :data="list" :loading="loading" :columns="inboundCols" empty-text="暂无过磅记录">
        <el-table :data="list" size="small" empty-text="暂无过磅记录">
          <el-table-column prop="doc_no" label="过磅单号" width="150" />
          <el-table-column prop="farmer_name" label="农户" min-width="120" />
          <el-table-column prop="gross_weight" label="毛重" width="90" />
          <el-table-column prop="deduct_weight" label="扣损" width="90" />
          <el-table-column prop="net_weight" label="净重" width="90" />
          <el-table-column prop="qc_result" label="质检" width="90" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'piecework-daily'">
      <TableOrCards :data="list" :loading="loading" :columns="pieceworkCols" empty-text="暂无计件数据">
        <el-table :data="list" size="small" empty-text="暂无计件数据">
          <el-table-column prop="worker_name" label="工人" min-width="120" />
          <el-table-column prop="process_name" label="工序" width="120" />
          <el-table-column prop="qty" label="完成kg" width="100" />
          <el-table-column prop="amount" label="金额" width="100" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'yield-analysis'">
      <TableOrCards :data="list" :loading="loading" :columns="yieldCols" empty-text="暂无扣损数据">
        <el-table :data="list" size="small" empty-text="暂无扣损数据">
          <el-table-column prop="process_name" label="工序" min-width="120" />
          <el-table-column prop="trace_count" label="溯源批数" width="100" />
          <el-table-column prop="input_kg" label="投入kg" width="100" />
          <el-table-column prop="output_kg" label="产出kg" width="100" />
          <el-table-column prop="loss_kg" label="损耗kg" width="100" />
          <el-table-column prop="loss_rate" label="损耗率" width="100" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'stock-ledger'">
      <TableOrCards :data="list" :loading="loading" :columns="stockLedgerCols" empty-text="暂无库存余额">
        <el-table :data="list" size="small" empty-text="暂无库存余额">
          <el-table-column prop="warehouse_name" label="仓库" width="120" />
          <el-table-column prop="product_code" label="产品编码" width="120" />
          <el-table-column prop="product_name" label="产品名称" min-width="140" />
          <el-table-column prop="qty" label="数量" width="100" />
          <el-table-column prop="amount" label="金额" width="120" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'trace-progress'">
      <TableOrCards :data="list" :loading="loading" :columns="traceCols" empty-text="暂无溯源批">
        <el-table :data="list" size="small" empty-text="暂无溯源批">
          <el-table-column prop="trace_code" label="溯源码" width="150" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="input_kg" label="投入kg" width="100" />
          <el-table-column prop="output_kg" label="产出kg" width="100" />
          <el-table-column prop="loss_rate" label="损耗率" width="90" />
          <el-table-column prop="open_issues" label="在制领料" width="100" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'farmer-settlement-summary'">
      <TableOrCards :data="list" :loading="loading" :columns="settlementCols" empty-text="暂无结算单">
        <el-table :data="list" size="small" empty-text="暂无结算单">
          <el-table-column prop="doc_no" label="结算单号" width="150" />
          <el-table-column prop="farmer_name" label="农户" min-width="120" />
          <el-table-column prop="biz_date" label="业务日" width="110" />
          <el-table-column prop="net_weight" label="净重" width="90" />
          <el-table-column prop="amount" label="金额" width="100" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="trace_code" label="溯源码" width="140" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'payroll-reconcile'">
      <TableOrCards :data="list" :loading="loading" :columns="payrollReconcileCols" empty-text="暂无对账数据">
        <el-table :data="list" size="small" empty-text="暂无对账数据">
          <el-table-column prop="worker_name" label="工人" min-width="120" />
          <el-table-column prop="emp_no" label="工号" width="100" />
          <el-table-column prop="sheet_piece_amount" label="工资单计件" width="110" />
          <el-table-column prop="piecework_amount" label="计件汇总" width="110" />
          <el-table-column prop="diff" label="差异" width="100" />
          <el-table-column prop="sheet_total" label="工资单合计" width="110" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'cost-period-summary'">
      <TableOrCards :data="list" :loading="loading" :columns="costPeriodCols" empty-text="暂无成本核算单">
        <el-table :data="list" size="small" empty-text="暂无成本核算单">
          <el-table-column prop="doc_no" label="成本单" width="150" />
          <el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="product_name" label="产品" min-width="120" />
          <el-table-column prop="material_cost" label="物料" width="100" />
          <el-table-column prop="labor_cost" label="人工" width="100" />
          <el-table-column prop="overhead" label="制造" width="100" />
          <el-table-column prop="total_cost" label="合计" width="110" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
      </TableOrCards>
    </template>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.head { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.head h2 { margin: 0; font-size: 18px; }
.sub { margin-top: 4px; color: #888; font-size: 12px; }
.actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.mb { margin-bottom: 12px; }
.kpi { margin-bottom: 12px; }
.stat-label { font-size: 12px; color: #888; }
.stat-value { font-size: 22px; font-weight: 600; margin-top: 4px; }
h4 { margin: 16px 0 8px; font-size: 14px; }
</style>
