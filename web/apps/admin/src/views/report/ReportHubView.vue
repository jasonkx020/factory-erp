<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { reportApi } from '@erp/shared'

type Row = Record<string, unknown>

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  enterprise: '企业报表',
  boss: '老板驾驶舱',
  'production-board': '生产看板',
  live: '生产实况',
  inquiries: '客户询价查询',
  'crm-stats': 'CRM统计',
  daily: '日统计报表',
  'gross-profit': '毛利润统计',
  qc: '质检报表',
  accounts: '账目统计',
  'stock-txns': '出入库查询',
  'stock-ledger': '收发存明细',
  'follow-ups': '跟进记录查询',
  'sales-weight': '销售重量统计',
  'product-sales': '产品销售查询',
  logistics: '系统物流查询',
  'cost-profit': '成本利润表',
  'balance-sheet': '资产负债表',
  'cash-flow': '现金流量表',
  'income-statement': '利润表',
}

const KPI_SECTIONS = new Set([
  'boss',
  'production-board',
  'live',
  'enterprise',
  'crm-stats',
  'daily',
  'gross-profit',
  'accounts',
  'balance-sheet',
  'cash-flow',
  'income-statement',
])

const active = computed(() => String(route.params.section || 'boss'))
const title = computed(() => TITLE_MAP[active.value] || '统计报表')
const loading = ref(false)
const list = ref<Row[]>([])
const kpis = ref<Row[]>([])
const summary = ref<Row | null>(null)
const extraRows = ref<Row[]>([])
const asOf = ref('')
const bizDate = ref(new Date().toISOString().slice(0, 10))

function fmt(v: unknown) {
  if (v == null || v === '') return '—'
  if (typeof v === 'number') {
    return Number.isInteger(v) ? String(v) : v.toFixed(2)
  }
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

    if (sec === 'boss') {
      res = await reportApi.boss()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      kpis.value = (data.kpis as Row[]) || (data.list as Row[]) || []
      summary.value = (data.summary as Row) || null
      asOf.value = String(data.as_of || '')
      list.value = kpis.value
    } else if (sec === 'production-board') {
      res = await reportApi.production()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      kpis.value = (data.list as Row[]) || []
      asOf.value = String(data.as_of || '')
      list.value = kpis.value
    } else if (sec === 'live') {
      res = await reportApi.live()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      kpis.value = (data.list as Row[]) || []
      asOf.value = String(data.as_of || '')
      list.value = kpis.value
    } else if (sec === 'enterprise') {
      res = await reportApi.enterprise()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      const overview = (data.overview as Row) || {}
      summary.value = overview
      kpis.value = toKpisFromSummary(overview, {
        sales_orders: '销售订单数',
        sales_amount: '销售总额',
        customers: '客户数',
        products: '产品数',
        stock_sku: '有库存SKU',
        fund_balance: '资金余额',
        open_tasks: '在制任务',
        fixed_assets: '固定资产',
      })
      asOf.value = String(overview.generated_at || '')
      list.value = ((data.list as Row[]) || []).map((r) => ({
        ...r,
        ...(typeof r === 'object' ? {} : {}),
      }))
      if (!list.value.length) list.value = kpis.value
    } else if (sec === 'daily') {
      res = await reportApi.daily(bizDate.value)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      const s = (data.summary as Row) || ((data.list as Row[]) || [])[0] || {}
      summary.value = s
      kpis.value = toKpisFromSummary(s, {
        sales_amount: '销售额',
        sales_orders: '销售单数',
        report_works: '报工单数',
        report_qty: '报工量',
        stock_in: '入库量',
        stock_out: '出库量',
        cash_in: '现金流入',
        cash_out: '现金流出',
        follow_ups: '跟进数',
      })
      list.value = (data.list as Row[]) || [s]
    } else if (sec === 'crm-stats') {
      res = await reportApi.crmStats()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      kpis.value = (data.list as Row[]) || []
      extraRows.value = (data.by_level as Row[]) || []
      list.value = kpis.value
    } else if (sec === 'inquiries') {
      res = await reportApi.inquiries()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'follow-ups') {
      res = await reportApi.followUps()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'gross-profit') {
      res = await reportApi.grossProfit()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      kpis.value = (data.list as Row[]) || []
      list.value = kpis.value
    } else if (sec === 'qc') {
      res = await reportApi.qc()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'accounts') {
      res = await reportApi.accounts()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      extraRows.value = (data.funds as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          income: '收入合计',
          expense: '支出合计',
          net: '净额',
        })
      }
    } else if (sec === 'stock-txns') {
      res = await reportApi.stockTxns()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'stock-ledger') {
      res = await reportApi.stockLedger()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'sales-weight') {
      res = await reportApi.salesWeight()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'product-sales') {
      res = await reportApi.productSales()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'logistics') {
      res = await reportApi.logistics()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (sec === 'cost-profit') {
      res = await reportApi.costProfit()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          total_cost: '成本合计',
          revenue: '收入',
          profit: '利润',
        })
      }
    } else if (sec === 'balance-sheet') {
      res = await reportApi.balanceSheet()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      asOf.value = String(data.as_of || '')
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          total_assets: '资产合计',
          total_liabilities: '负债合计',
          equity: '净资产',
        })
      }
    } else if (sec === 'cash-flow') {
      res = await reportApi.cashFlow()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          in: '现金流入',
          out: '现金流出',
          net: '净现金流',
        })
      }
    } else if (sec === 'income-statement') {
      res = await reportApi.incomeStatement()
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      const data = (res.data as Row) || {}
      summary.value = (data.summary as Row) || null
      list.value = (data.list as Row[]) || []
      if (summary.value) {
        kpis.value = toKpisFromSummary(summary.value, {
          income: '营业收入',
          cost: '营业成本',
          expense: '期间费用',
          profit: '利润总额',
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
        <template v-if="active === 'daily'">
          <el-input v-model="bizDate" size="small" style="width: 140px" placeholder="YYYY-MM-DD" />
          <el-button size="small" type="primary" @click="refresh">按日查询</el-button>
        </template>
        <el-button size="small" @click="refresh">刷新</el-button>
      </div>
    </div>

    <el-row v-if="showKpis" :gutter="12" class="mb">
      <el-col v-for="k in kpis" :key="String(k.key || k.metric || k.title)" :xs="12" :sm="8" :md="6">
        <el-card shadow="never" class="kpi">
          <div class="stat-label">{{ k.title || k.item || k.metric || k.key }}</div>
          <div class="stat-value">{{ fmt(k.value ?? k.amount) }}<span v-if="k.unit" class="unit">{{ k.unit }}</span></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 老板/生产看板/实况：仅 KPI -->
    <template v-if="active === 'boss' || active === 'production-board' || active === 'live'">
      <el-empty v-if="!kpis.length" description="暂无指标数据" />
    </template>

    <!-- 企业报表定义表 -->
    <template v-else-if="active === 'enterprise'">
      <h4>报表定义</h4>
      <el-table :data="list" size="small" empty-text="暂无定义">
        <el-table-column prop="code" label="编码" width="160" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="report_type" label="类型" width="120" />
        <el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <!-- CRM 分级 -->
    <template v-else-if="active === 'crm-stats'">
      <h4>客户分级</h4>
      <el-table :data="extraRows" size="small" empty-text="暂无分级数据">
        <el-table-column prop="level" label="级别" min-width="140" />
        <el-table-column prop="count" label="数量" width="120" />
      </el-table>
    </template>

    <!-- 日统计明细 -->
    <template v-else-if="active === 'daily'">
      <el-descriptions v-if="summary" :column="3" border size="small" title="当日汇总">
        <el-descriptions-item label="业务日">{{ summary.biz_date }}</el-descriptions-item>
        <el-descriptions-item label="销售额">{{ fmt(summary.sales_amount) }}</el-descriptions-item>
        <el-descriptions-item label="销售单">{{ fmt(summary.sales_orders) }}</el-descriptions-item>
        <el-descriptions-item label="报工">{{ fmt(summary.report_works) }} / {{ fmt(summary.report_qty) }}</el-descriptions-item>
        <el-descriptions-item label="出入库">入 {{ fmt(summary.stock_in) }} / 出 {{ fmt(summary.stock_out) }}</el-descriptions-item>
        <el-descriptions-item label="现金流">入 {{ fmt(summary.cash_in) }} / 出 {{ fmt(summary.cash_out) }}</el-descriptions-item>
      </el-descriptions>
    </template>

    <!-- 询价 -->
    <template v-else-if="active === 'inquiries'">
      <el-table :data="list" size="small" empty-text="暂无询价">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="customer_id" label="客户ID" width="90" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="created_at" label="创建时间" min-width="160" />
      </el-table>
    </template>

    <!-- 跟进 -->
    <template v-else-if="active === 'follow-ups'">
      <el-table :data="list" size="small" empty-text="暂无跟进">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="customer_name" label="客户" min-width="140" />
        <el-table-column prop="follow_type" label="类型" width="100" />
        <el-table-column prop="follow_at" label="跟进时间" width="160" />
        <el-table-column prop="content" label="内容" min-width="200" show-overflow-tooltip />
        <el-table-column prop="next_remind_at" label="下次提醒" width="160" />
      </el-table>
    </template>

    <!-- 毛利仅 KPI -->
    <template v-else-if="active === 'gross-profit'">
      <el-empty v-if="!kpis.length" description="暂无毛利数据" />
    </template>

    <!-- 质检 -->
    <template v-else-if="active === 'qc'">
      <el-table :data="list" size="small" empty-text="暂无质检记录">
        <el-table-column prop="source" label="来源" width="90" />
        <el-table-column prop="doc_no" label="单号" width="150" />
        <el-table-column prop="qc_type" label="类型" width="100" />
        <el-table-column prop="product_id" label="产品ID" width="90" />
        <el-table-column prop="qty" label="数量" width="90" />
        <el-table-column prop="qty_pass" label="合格" width="80" />
        <el-table-column prop="qty_fail" label="不合格" width="80" />
        <el-table-column prop="result" label="结果" width="90" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="created_at" label="时间" min-width="160" />
      </el-table>
    </template>

    <!-- 账目 -->
    <template v-else-if="active === 'accounts'">
      <h4>收支方向汇总</h4>
      <el-table :data="list" size="small" class="mb" empty-text="暂无流水">
        <el-table-column prop="direction" label="方向" width="100" />
        <el-table-column prop="count" label="笔数" width="100" />
        <el-table-column prop="amount" label="金额" min-width="140" />
      </el-table>
      <h4>资金账户</h4>
      <el-table :data="extraRows" size="small" empty-text="暂无账户">
        <el-table-column prop="code" label="编码" width="120" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="currency" label="币种" width="80" />
        <el-table-column prop="balance" label="余额" width="140" />
      </el-table>
    </template>

    <!-- 出入库 -->
    <template v-else-if="active === 'stock-txns'">
      <el-table :data="list" size="small" empty-text="暂无出入库单据">
        <el-table-column prop="doc_no" label="单号" width="150" />
        <el-table-column prop="doc_type" label="类型" width="100" />
        <el-table-column prop="warehouse_id" label="仓库" width="80" />
        <el-table-column prop="biz_date" label="业务日" width="110" />
        <el-table-column prop="qty_in" label="入库量" width="100" />
        <el-table-column prop="qty_out" label="出库量" width="100" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
      </el-table>
    </template>

    <!-- 收发存 -->
    <template v-else-if="active === 'stock-ledger'">
      <el-table :data="list" size="small" empty-text="暂无库存余额">
        <el-table-column prop="warehouse_name" label="仓库" width="120" />
        <el-table-column prop="product_code" label="产品编码" width="120" />
        <el-table-column prop="product_name" label="产品名称" min-width="140" />
        <el-table-column prop="batch_no" label="批次" width="100" />
        <el-table-column prop="qty" label="数量" width="100" />
        <el-table-column prop="avg_cost" label="平均成本" width="110" />
        <el-table-column prop="amount" label="金额" width="120" />
      </el-table>
    </template>

    <!-- 销售重量 -->
    <template v-else-if="active === 'sales-weight'">
      <el-table :data="list" size="small" empty-text="暂无销售重量">
        <el-table-column prop="product_code" label="产品编码" width="120" />
        <el-table-column prop="product_name" label="产品名称" min-width="140" />
        <el-table-column prop="qty" label="数量" width="100" />
        <el-table-column prop="weight" label="重量" width="120" />
        <el-table-column prop="amount" label="金额" width="120" />
      </el-table>
    </template>

    <!-- 产品销售 -->
    <template v-else-if="active === 'product-sales'">
      <el-table :data="list" size="small" empty-text="暂无产品销售">
        <el-table-column prop="product_code" label="产品编码" width="120" />
        <el-table-column prop="product_name" label="产品名称" min-width="140" />
        <el-table-column prop="order_count" label="订单数" width="90" />
        <el-table-column prop="qty" label="数量" width="100" />
        <el-table-column prop="amount" label="金额" width="120" />
        <el-table-column prop="avg_price" label="均价" width="110" />
      </el-table>
    </template>

    <!-- 物流 -->
    <template v-else-if="active === 'logistics'">
      <el-table :data="list" size="small" empty-text="暂无物流轨迹（可在系统物流表维护）">
        <el-table-column prop="track_no" label="运单号" width="160" />
        <el-table-column prop="carrier_name" label="承运商" width="120" />
        <el-table-column prop="order_id" label="订单ID" width="90" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="location" label="位置" min-width="140" />
        <el-table-column prop="updated_at" label="更新时间" width="160" />
      </el-table>
    </template>

    <!-- 成本利润明细 -->
    <template v-else-if="active === 'cost-profit'">
      <el-table :data="list" size="small" empty-text="暂无成本核算">
        <el-table-column prop="doc_no" label="单号" width="140" />
        <el-table-column prop="period" label="期间" width="100" />
        <el-table-column prop="product_id" label="产品ID" width="90" />
        <el-table-column prop="material_cost" label="材料" width="100" />
        <el-table-column prop="labor_cost" label="人工" width="100" />
        <el-table-column prop="overhead" label="制造费用" width="100" />
        <el-table-column prop="total_cost" label="总成本" width="110" />
        <el-table-column prop="status" label="状态" width="100" />
      </el-table>
    </template>

    <!-- 资产负债表 / 现金流 / 利润表 -->
    <template v-else-if="active === 'balance-sheet' || active === 'cash-flow' || active === 'income-statement'">
      <el-table :data="list" size="small" empty-text="暂无报表行">
        <el-table-column v-if="active === 'balance-sheet'" prop="section" label="类别" width="110" />
        <el-table-column prop="item" label="项目" min-width="180" />
        <el-table-column prop="amount" label="金额" width="160">
          <template #default="{ row }">{{ fmt(row.amount) }}</template>
        </el-table-column>
      </el-table>
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
.stat-label { color: #888; font-size: 13px; }
.stat-value { font-size: 22px; font-weight: 600; margin-top: 6px; }
.unit { margin-left: 4px; font-size: 13px; font-weight: 400; color: #666; }
h4 { margin: 8px 0 8px; font-size: 14px; }
</style>
