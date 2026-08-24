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
  finErrMsg,
  finStatusLabel,
  finStatusType,
  money,
} from './financeLabels'

type Row = Record<string, unknown>

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
  { prop: 'source_type', label: '来源类型' },
  { prop: 'source_id', label: '来源ID' },
  moneyCol('amount', '金额'),
]

const REPORT_LINKS = [
  { title: '日经营快照', path: '/report/hub/daily', desc: '入场、产出、计件、农户应付与库存' },
  { title: '计件日结汇总', path: '/report/hub/piecework-daily', desc: '与现场计件/App 今日核对一致' },
  { title: '农户结算对账', path: '/report/hub/farmer-settlement-summary', desc: '原料款已付/待付' },
  { title: '薪酬核算对账', path: '/report/hub/payroll-reconcile', desc: '月工资单 vs 计件汇总差异' },
  { title: '成本期间汇总', path: '/report/hub/cost-period-summary', desc: '按期间汇总成本核算单' },
]

const route = useRoute()
const router = useRouter()

const active = computed(() => {
  const s = String(route.params.section || 'cost-accountings')
  return s === 'cost-traces' ? 'cost-traces' : 'cost-accountings'
})
const title = computed(() => (active.value === 'cost-traces' ? '成本明细溯源表' : '成本核算'))
const hint = computed(() => FIN_HINT[active.value] || '')
const loading = ref(false)
const list = ref<Row[]>([])
const keyword = ref('')
const farmerPending = ref(0)
const farmerPendingAmt = ref(0)

const costForm = reactive({
  period: new Date().toISOString().slice(0, 7),
  product_id: 1,
  material_cost: 0,
  labor_cost: 0,
  overhead: 0,
})

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
  if (active.value === 'cost-accountings') {
    return [
      { label: '成本单数', value: rows.length },
      { label: '物料合计', value: money(sum('material_cost')) },
      { label: '人工合计', value: money(sum('labor_cost')) },
      { label: '总成本', value: money(sum('total_cost')) },
    ]
  }
  return [
    { label: '溯源行数', value: rows.length },
    { label: '金额合计', value: money(sum('amount')) },
  ]
})

function goFarmerSettle() {
  router.push('/purchase/hub/settlements')
}

function goReport(path: string) {
  router.push(path)
}

async function loadFarmerPending() {
  try {
    const fs = await purchaseApi.farmerSettlements()
    const rows = ((fs.data as { list?: Row[] })?.list) || []
    const pending = rows.filter((r) => {
      const st = String(r.status || '')
      return st !== 'settle_paid' && st !== 'paid' && st !== 'void'
    })
    farmerPending.value = pending.length
    farmerPendingAmt.value = pending.reduce((s, r) => s + (Number(r.amount) || 0), 0)
  } catch {
    farmerPending.value = 0
    farmerPendingAmt.value = 0
  }
}

async function refresh() {
  loading.value = true
  try {
    const res =
      active.value === 'cost-traces'
        ? await financeApi.costTraces()
        : await financeApi.costAccountings()
    if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function run(fn: () => Promise<{ code: number; msg: string }>, ok = '成功') {
  const res = await fn()
  if (res.code !== 1) return ElMessage.error(finErrMsg(res.msg))
  ElMessage.success(ok)
  await refresh()
}

onMounted(async () => {
  await loadFarmerPending()
  await refresh()
})
watch(active, refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <header class="page-head">
      <div>
        <h2 class="title">{{ title }}</h2>
        <p class="desc">{{ hint }}</p>
        <p class="scope-note">产线版财务管理仅保留批次成本；总账/凭证/资金账户已裁剪。经营总览请用下方报表快捷入口。</p>
      </div>
      <div class="head-actions">
        <el-button type="warning" plain @click="goFarmerSettle">
          待付农户 {{ farmerPending }}（{{ money(farmerPendingAmt) }}）
        </el-button>
        <el-button @click="refresh">刷新</el-button>
      </div>
    </header>

    <el-card class="mb links-card" header="经营与对账（统计报表）">
      <div class="report-links">
        <div v-for="link in REPORT_LINKS" :key="link.path" class="report-link" @click="goReport(link.path)">
          <div class="link-title">{{ link.title }}</div>
          <div class="link-desc">{{ link.desc }}</div>
        </div>
      </div>
    </el-card>

    <div class="stats">
      <div v-for="s in stats" :key="s.label" class="stat">
        <div class="label">{{ s.label }}</div>
        <div class="value">{{ s.value }}</div>
      </div>
    </div>
    <div class="toolbar">
      <el-input v-model="keyword" clearable placeholder="筛选单号 / 产品 / 期间" style="width:220px" />
    </div>

    <template v-if="active === 'cost-accountings'">
      <el-card class="mb" header="新建成本核算单">
        <el-form inline size="small">
          <el-form-item label="期间">
            <el-date-picker v-model="costForm.period" type="month" value-format="YYYY-MM" style="width:140px" />
          </el-form-item>
          <el-form-item label="产品"><ProductSelect v-model="costForm.product_id" :clearable="false" /></el-form-item>
          <el-form-item label="物料成本"><el-input-number v-model="costForm.material_cost" :min="0" /></el-form-item>
          <el-form-item label="人工"><el-input-number v-model="costForm.labor_cost" :min="0" /></el-form-item>
          <el-form-item label="制造费用"><el-input-number v-model="costForm.overhead" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createCostAccounting({ ...costForm }), '成本单已建')">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="filteredList" :loading="loading" :columns="costCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="product_name" label="产品" min-width="120" />
          <el-table-column label="物料" width="100"><template #default="{ row }">{{ money(row.material_cost) }}</template></el-table-column>
          <el-table-column label="人工" width="100"><template #default="{ row }">{{ money(row.labor_cost) }}</template></el-table-column>
          <el-table-column label="制造" width="100"><template #default="{ row }">{{ money(row.overhead) }}</template></el-table-column>
          <el-table-column label="合计" width="110"><template #default="{ row }">{{ money(row.total_cost) }}</template></el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="finStatusType(row.status)">{{ finStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.calcCost(Number(row.id)), '已核算')">核算</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.calcCost(Number(row.id)), '已核算')">核算</el-button>
        </template>
      </TableOrCards>
    </template>

    <template v-else>
      <TableOrCards :data="filteredList" :loading="loading" :columns="costTraceCols">
        <el-table :data="filteredList" size="small">
          <el-table-column prop="doc_no" label="成本单" width="150" />
          <el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="source_type" label="来源类型" width="120" />
          <el-table-column prop="source_id" label="来源ID" width="90" />
          <el-table-column label="金额" width="120"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
        </el-table>
      </TableOrCards>
    </template>
  </div>
</template>

<style scoped>
.page {
  background: #fff;
  padding: 12px 16px 8px;
  border-radius: 8px;
  border: 1px solid #d5dde3;
  min-height: 360px;
}
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; }
.desc { color: #5c6b75; font-size: 13px; margin: 0; max-width: 720px; line-height: 1.5; }
.scope-note { color: #8a6d3b; font-size: 12px; margin: 6px 0 0; }
.head-actions { display: flex; gap: 8px; flex-shrink: 0; flex-wrap: wrap; }
.stats { display: grid; grid-template-columns: repeat(4, minmax(96px, 1fr)); gap: 10px; margin: 12px 0; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 18px; font-weight: 600; font-variant-numeric: tabular-nums; }
.toolbar { margin-bottom: 10px; }
.mb { margin-bottom: 12px; }
.report-links { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
.report-link {
  padding: 10px 12px;
  border: 1px solid #e8eef2;
  border-radius: 8px;
  cursor: pointer;
  background: #fafbfc;
}
.report-link:hover { border-color: #409eff; background: #f0f7ff; }
.link-title { font-weight: 600; font-size: 14px; margin-bottom: 4px; }
.link-desc { font-size: 12px; color: #6b7a85; line-height: 1.4; }
</style>
