<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { plantingApi, purchaseApi } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>
type Kpi = { key: string; title: string; value: unknown }

const route = useRoute()
const router = useRouter()

const TITLE_MAP: Record<string, string> = {
  overview: '木薯种植总览',
  plots: '地块档案',
  contracts: '种植合同',
  'field-logs': '田间作业',
  'harvest-plans': '采收计划',
}

const SECTIONS = [
  { key: 'overview', label: '总览' },
  { key: 'plots', label: '地块档案' },
  { key: 'contracts', label: '种植合同' },
  { key: 'field-logs', label: '田间作业' },
  { key: 'harvest-plans', label: '采收计划' },
]

const LOG_TYPE_OPTIONS = [
  { value: 'planting', label: '播种' },
  { value: 'fertilize', label: '施肥' },
  { value: 'weed', label: '除草' },
  { value: 'pest', label: '病虫害' },
  { value: 'irrigation', label: '灌溉' },
  { value: 'other', label: '其他' },
]

const plotCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '地块名' },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'area_mu', label: '面积(亩)' },
  { prop: 'variety', label: '品种' },
  { prop: 'location', label: '位置' },
  { prop: 'status', label: '状态' },
]
const contractCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '合同号', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'plot_name', label: '地块' },
  { prop: 'area_mu', label: '面积(亩)' },
  { prop: 'unit_price', label: '单价' },
  { prop: 'start_date', label: '开始' },
  { prop: 'status', label: '状态' },
]
const logCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'plot_name', label: '地块' },
  { prop: 'log_type', label: '类型' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'operator_name', label: '操作人' },
  { prop: 'content', label: '内容' },
]
const harvestCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '计划号', primary: true },
  { prop: 'farmer_name', label: '农户' },
  { prop: 'plot_name', label: '地块' },
  { prop: 'plan_date', label: '采收日' },
  { prop: 'estimate_weight', label: '预估(kg)' },
  { prop: 'status', label: '状态' },
]

const active = computed(() => String(route.params.section || 'overview'))
const title = computed(() => TITLE_MAP[active.value] || '种植管理')

const loading = ref(false)
const list = ref<Row[]>([])
const kpis = ref<Kpi[]>([])
const farmers = ref<Row[]>([])
const plots = ref<Row[]>([])

const plotForm = reactive({
  code: '',
  name: '',
  farmer_id: null as number | null,
  area_mu: 0,
  location: '',
  variety: '鲜木薯',
  soil_type: '',
  irrigation_type: '',
  remark: '',
})

const contractForm = reactive({
  farmer_id: null as number | null,
  plot_id: null as number | null,
  area_mu: 0,
  unit_price: 0,
  start_date: '',
  end_date: '',
  variety: '鲜木薯',
  remark: '',
})

const logForm = reactive({
  plot_id: null as number | null,
  log_type: 'fertilize',
  biz_date: new Date().toISOString().slice(0, 10),
  operator_name: '',
  content: '',
  qty: 0,
  unit: 'kg',
  remark: '',
})

const harvestForm = reactive({
  plot_id: null as number | null,
  farmer_id: null as number | null,
  plan_date: new Date().toISOString().slice(0, 10),
  estimate_weight: 0,
  variety: '鲜木薯',
  remark: '',
})

function logTypeLabel(v: string) {
  return LOG_TYPE_OPTIONS.find((o) => o.value === v)?.label || v
}

async function loadMeta() {
  const [fr, pl] = await Promise.all([purchaseApi.farmers(), plantingApi.plots('page_size=500')])
  farmers.value = ((fr.data as { list?: Row[] })?.list) || []
  plots.value = ((pl.data as { list?: Row[] })?.list) || []
}

async function refresh() {
  loading.value = true
  try {
    if (active.value === 'overview') {
      const res = await plantingApi.dashboard()
      if (res.code !== 1) return ElMessage.error(res.msg)
      const data = (res.data as { kpis?: Kpi[] }) || {}
      kpis.value = data.kpis || []
      return
    }
    if (active.value === 'plots') {
      const res = await plantingApi.plots()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
      plots.value = list.value
    } else if (active.value === 'contracts') {
      const res = await plantingApi.contracts()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'field-logs') {
      const res = await plantingApi.fieldLogs()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'harvest-plans') {
      const res = await plantingApi.harvestPlans()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
    }
  } finally {
    loading.value = false
  }
}

function goSection(key: string) {
  router.push(`/planting/hub/${key}`)
}

async function createPlot() {
  if (!plotForm.name) return ElMessage.warning('请填写地块名称')
  const res = await plantingApi.createPlot({ ...plotForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('地块已创建')
  Object.assign(plotForm, { code: '', name: '', farmer_id: null, area_mu: 0, location: '', remark: '' })
  await loadMeta()
  await refresh()
}

async function createContract() {
  if (!contractForm.farmer_id) return ElMessage.warning('请选择农户')
  const res = await plantingApi.createContract({ ...contractForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('合同已创建')
  Object.assign(contractForm, {
    farmer_id: null, plot_id: null, area_mu: 0, unit_price: 0, start_date: '', end_date: '', remark: '',
  })
  await refresh()
}

async function createFieldLog() {
  if (!logForm.plot_id) return ElMessage.warning('请选择地块')
  const res = await plantingApi.createFieldLog({ ...logForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('田间作业已记录')
  logForm.content = ''
  logForm.qty = 0
  await refresh()
}

async function createHarvestPlan() {
  if (!harvestForm.plot_id) return ElMessage.warning('请选择地块')
  if (!harvestForm.farmer_id) return ElMessage.warning('请选择农户')
  const res = await plantingApi.createHarvestPlan({ ...harvestForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('采收计划已创建')
  harvestForm.estimate_weight = 0
  harvestForm.remark = ''
  await refresh()
}

async function confirmPlan(row: Row) {
  const id = Number(row.id)
  const res = await plantingApi.confirmHarvestPlan(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('计划已确认')
  await refresh()
}

async function toArrival(row: Row) {
  const id = Number(row.id)
  await ElMessageBox.confirm('将为该采收计划生成到货登记，供后续过磅收货使用。', '生成到货登记', { type: 'info' })
  const res = await plantingApi.harvestPlanToArrival(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  const arrivalId = (res.data as { arrival_id?: number })?.arrival_id
  ElMessage.success(`已生成到货登记 #${arrivalId}`)
  await refresh()
}

function onPlotSelectForHarvest(plotId: number) {
  harvestForm.plot_id = plotId
  const plot = plots.value.find((p) => Number(p.id) === plotId)
  if (plot) {
    harvestForm.farmer_id = Number(plot.farmer_id) || null
    harvestForm.variety = String(plot.variety || '鲜木薯')
  }
}

watch(active, () => refresh())
onMounted(async () => {
  await loadMeta()
  await refresh()
})
</script>

<template>
  <div v-loading="loading" class="planting-hub">
    <header class="hub-head">
      <div>
        <h2>{{ title }}</h2>
        <p class="desc">木薯种植全流程：地块 → 合同 → 田间作业 → 采收计划 → 到货过磅</p>
      </div>
      <div class="section-tabs">
        <el-button
          v-for="s in SECTIONS"
          :key="s.key"
          :type="active === s.key ? 'primary' : 'default'"
          size="small"
          @click="goSection(s.key)"
        >
          {{ s.label }}
        </el-button>
      </div>
    </header>

    <template v-if="active === 'overview'">
      <el-row :gutter="12">
        <el-col v-for="k in kpis" :key="k.key" :xs="12" :sm="8" :md="4">
          <el-card shadow="never" class="kpi-card">
            <div class="kpi-label">{{ k.title }}</div>
            <div class="kpi-value">{{ k.value ?? '—' }}</div>
          </el-card>
        </el-col>
      </el-row>
      <el-card shadow="never" class="flow-card">
        <template #header>业务闭环</template>
        <el-steps :active="4" align-center>
          <el-step title="登记地块" description="关联农户与面积" />
          <el-step title="签订种植合同" description="约定品种与单价" />
          <el-step title="记录田间作业" description="播种施肥除草等" />
          <el-step title="制定采收计划" description="预估产量与日期" />
          <el-step title="生成到货登记" description="衔接过磅收货" />
        </el-steps>
      </el-card>
    </template>

    <template v-else-if="active === 'plots'">
      <el-card shadow="never" class="form-card">
        <template #header>新建地块</template>
        <el-form inline label-width="72px">
          <el-form-item label="编码"><el-input v-model="plotForm.code" placeholder="自动生成" style="width:120px" /></el-form-item>
          <el-form-item label="名称" required><el-input v-model="plotForm.name" style="width:140px" /></el-form-item>
          <el-form-item label="农户">
            <el-select v-model="plotForm.farmer_id" clearable filterable placeholder="选择农户" style="width:160px">
              <el-option v-for="f in farmers" :key="f.id" :label="f.name" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="面积(亩)"><el-input-number v-model="plotForm.area_mu" :min="0" :step="0.1" /></el-form-item>
          <el-form-item label="位置"><el-input v-model="plotForm.location" style="width:160px" /></el-form-item>
          <el-form-item label="品种"><el-input v-model="plotForm.variety" style="width:100px" /></el-form-item>
          <el-form-item><el-button type="primary" @click="createPlot">保存</el-button></el-form-item>
        </el-form>
      </el-card>
      <TableOrCards :columns="plotCols" :data="list" row-key="id">
        <el-table :data="list" size="small">
          <el-table-column prop="code" label="编码" width="110" />
          <el-table-column prop="name" label="地块名" min-width="120" />
          <el-table-column prop="farmer_name" label="农户" width="100" />
          <el-table-column prop="area_mu" label="面积(亩)" width="90" />
          <el-table-column prop="variety" label="品种" width="90" />
          <el-table-column prop="location" label="位置" min-width="120" />
          <el-table-column prop="status" label="状态" width="80" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'contracts'">
      <el-card shadow="never" class="form-card">
        <template #header>新建种植合同</template>
        <el-form inline label-width="72px">
          <el-form-item label="农户" required>
            <el-select v-model="contractForm.farmer_id" clearable filterable style="width:160px">
              <el-option v-for="f in farmers" :key="f.id" :label="f.name" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="地块">
            <el-select v-model="contractForm.plot_id" clearable filterable style="width:160px">
              <el-option v-for="p in plots" :key="p.id" :label="p.name" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="面积(亩)"><el-input-number v-model="contractForm.area_mu" :min="0" :step="0.1" /></el-form-item>
          <el-form-item label="单价"><el-input-number v-model="contractForm.unit_price" :min="0" :step="0.01" /></el-form-item>
          <el-form-item label="开始日"><el-date-picker v-model="contractForm.start_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
          <el-form-item><el-button type="primary" @click="createContract">保存</el-button></el-form-item>
        </el-form>
      </el-card>
      <TableOrCards :columns="contractCols" :data="list" row-key="id">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="合同号" width="140" />
          <el-table-column prop="farmer_name" label="农户" width="100" />
          <el-table-column prop="plot_name" label="地块" width="120" />
          <el-table-column prop="area_mu" label="面积(亩)" width="90" />
          <el-table-column prop="unit_price" label="单价" width="80" />
          <el-table-column prop="start_date" label="开始" width="110" />
          <el-table-column prop="status" label="状态" width="80" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'field-logs'">
      <el-card shadow="never" class="form-card">
        <template #header>记录田间作业</template>
        <el-form inline label-width="72px">
          <el-form-item label="地块" required>
            <el-select v-model="logForm.plot_id" clearable filterable style="width:160px">
              <el-option v-for="p in plots" :key="p.id" :label="p.name" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="logForm.log_type" style="width:120px">
              <el-option v-for="o in LOG_TYPE_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="日期"><el-date-picker v-model="logForm.biz_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
          <el-form-item label="操作人"><el-input v-model="logForm.operator_name" style="width:100px" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="logForm.content" style="width:200px" /></el-form-item>
          <el-form-item><el-button type="primary" @click="createFieldLog">保存</el-button></el-form-item>
        </el-form>
      </el-card>
      <TableOrCards :columns="logCols" :data="list" row-key="id">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="140" />
          <el-table-column prop="plot_name" label="地块" width="120" />
          <el-table-column label="类型" width="90">
            <template #default="{ row }">{{ logTypeLabel(String(row.log_type)) }}</template>
          </el-table-column>
          <el-table-column prop="biz_date" label="日期" width="110" />
          <el-table-column prop="operator_name" label="操作人" width="90" />
          <el-table-column prop="content" label="内容" min-width="160" />
        </el-table>
        <template #log_type="{ row }">{{ logTypeLabel(String(row.log_type)) }}</template>
      </TableOrCards>
    </template>

    <template v-else-if="active === 'harvest-plans'">
      <el-card shadow="never" class="form-card">
        <template #header>新建采收计划</template>
        <el-form inline label-width="72px">
          <el-form-item label="地块" required>
            <el-select v-model="harvestForm.plot_id" clearable filterable style="width:160px" @change="onPlotSelectForHarvest">
              <el-option v-for="p in plots" :key="p.id" :label="p.name" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="农户" required>
            <el-select v-model="harvestForm.farmer_id" clearable filterable style="width:160px">
              <el-option v-for="f in farmers" :key="f.id" :label="f.name" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="采收日"><el-date-picker v-model="harvestForm.plan_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
          <el-form-item label="预估(kg)"><el-input-number v-model="harvestForm.estimate_weight" :min="0" :step="100" /></el-form-item>
          <el-form-item><el-button type="primary" @click="createHarvestPlan">保存</el-button></el-form-item>
        </el-form>
      </el-card>
      <TableOrCards :columns="harvestCols" :data="list" row-key="id">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="计划号" width="140" />
          <el-table-column prop="farmer_name" label="农户" width="100" />
          <el-table-column prop="plot_name" label="地块" width="120" />
          <el-table-column prop="plan_date" label="采收日" width="110" />
          <el-table-column prop="estimate_weight" label="预估(kg)" width="100" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status === 'draft'" link type="primary" @click="confirmPlan(row)">确认</el-button>
              <el-button v-if="!row.arrival_id" link type="success" @click="toArrival(row)">生成到货</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status === 'draft'" link type="primary" @click="confirmPlan(row)">确认</el-button>
          <el-button v-if="!row.arrival_id" link type="success" @click="toArrival(row)">生成到货</el-button>
        </template>
      </TableOrCards>
    </template>
  </div>
</template>

<style scoped>
.planting-hub { padding: 4px 2px 24px; }
.hub-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.hub-head h2 { margin: 0 0 4px; font-size: 20px; }
.desc { margin: 0; color: #8a9aa3; font-size: 13px; }
.section-tabs { display: flex; gap: 6px; flex-wrap: wrap; }
.kpi-card { margin-bottom: 12px; border: 1px solid #e8eef2; }
.kpi-label { color: #8a9aa3; font-size: 12px; }
.kpi-value { margin-top: 8px; font-size: 24px; font-weight: 600; color: #0d7a6f; }
.flow-card { margin-top: 16px; }
.form-card { margin-bottom: 16px; }
</style>
