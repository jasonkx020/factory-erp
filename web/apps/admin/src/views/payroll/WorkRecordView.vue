<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  empTypeLabel,
  formOptionLabel,
  payrollApi,
  paySheetStatusLabel,
  STATION_FLOW_EVENT_OPTIONS,
} from '@erp/shared'
import { EmployeeSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

function today() {
  return new Date().toISOString().slice(0, 10)
}

const loading = ref(false)
const employeeId = ref<number | null>(null)
const dateRange = ref<[string, string]>([today(), today()])
const tab = ref('flows')
const employee = ref<Row>({})
const kpi = ref<Row>({})
const flows = ref<Row[]>([])
const piecework = ref<Row[]>([])
const wages = ref<Row[]>([])

const flowCols: MobileCardColumn[] = [
  { prop: 'created_at', label: '时间', primary: true },
  { prop: 'event_type_label', label: '类型' },
  { prop: 'board_code', label: '板码' },
  { prop: 'process_name', label: '工序' },
  { prop: 'kg', label: 'kg' },
  { prop: 'amount', label: '金额' },
]
const pieceCols: MobileCardColumn[] = [
  { prop: 'biz_date', label: '日期', primary: true },
  { prop: 'process_name', label: '工序' },
  { prop: 'qty', label: '产量' },
  { prop: 'weight', label: '重量' },
  { prop: 'amount', label: '金额' },
]
const wageCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '工资单', primary: true },
  { prop: 'period_ym', label: '期间' },
  { prop: 'status_label', label: '状态' },
  { prop: 'piece_amount', label: '计件' },
  { prop: 'total_amount', label: '合计' },
]

const empTitle = computed(() => {
  const n = String(employee.value.name || '')
  const no = String(employee.value.emp_no || '')
  if (n && no) return `${n} · ${no}`
  return n || no || '未选员工'
})

function num(v: unknown, digits = 2) {
  const n = Number(v ?? 0)
  if (!Number.isFinite(n)) return '0'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: digits })
}

function money(v: unknown) {
  return num(v, 2)
}

function sheetTagType(status: unknown): 'info' | 'warning' | 'success' {
  const s = String(status || '')
  if (s === 'paid') return 'success'
  if (s === 'confirmed') return 'warning'
  return 'info'
}

async function load() {
  if (!employeeId.value) {
    employee.value = {}
    kpi.value = {}
    flows.value = []
    piecework.value = []
    wages.value = []
    return
  }
  const [from, to] = dateRange.value || [today(), today()]
  loading.value = true
  try {
    const qs = `employee_id=${employeeId.value}&date_from=${encodeURIComponent(from)}&date_to=${encodeURIComponent(to)}`
    const res = await payrollApi.workRecords(qs)
    if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
    const data = (res.data || {}) as Row
    employee.value = (data.employee as Row) || {}
    kpi.value = (data.kpi as Row) || {}
    flows.value = ((data.flows as Row[]) || []).map((r) => ({
      ...r,
      event_type_label: formOptionLabel(STATION_FLOW_EVENT_OPTIONS, r.event_type),
    }))
    piecework.value = (data.piecework as Row[]) || []
    wages.value = ((data.wages as Row[]) || []).map((r) => ({
      ...r,
      status_label: paySheetStatusLabel(r.status),
      emp_type_label: empTypeLabel(r.emp_type),
    }))
  } finally {
    loading.value = false
  }
}

watch(employeeId, load)
watch(dateRange, load)
onMounted(load)
</script>

<template>
  <div v-loading="loading" class="rec">
    <header class="page-head">
      <div>
        <h2 class="title">员工工作台账</h2>
        <p class="desc">按人查看过站领取/退库流水、计件日结与工资单行。生产现场台账仍在生产管理维护。</p>
      </div>
      <div class="head-meta">
        <span class="meta-pill">{{ empTitle }}</span>
      </div>
    </header>

    <div class="row">
      <EmployeeSelect v-model="employeeId" placeholder="选择员工" style="width:220px" />
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        value-format="YYYY-MM-DD"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        :clearable="false"
      />
      <el-button @click="load">查询</el-button>
    </div>

    <div class="stats">
      <div class="stat ok">
        <div class="label">领取 kg</div>
        <div class="value">{{ num(kpi.issue_kg) }}</div>
      </div>
      <div class="stat warn">
        <div class="label">退库 kg</div>
        <div class="value">{{ num(kpi.return_kg) }}</div>
      </div>
      <div class="stat">
        <div class="label">计件产量</div>
        <div class="value">{{ num(kpi.piece_qty) }}</div>
      </div>
      <div class="stat">
        <div class="label">计件金额</div>
        <div class="value">{{ money(kpi.piece_amount) }}</div>
      </div>
      <div class="stat">
        <div class="label">工资单合计</div>
        <div class="value">{{ money(kpi.wage_amount) }}</div>
      </div>
    </div>

    <el-empty v-if="!employeeId" description="请选择员工后查询日工作记录" />

    <el-tabs v-else v-model="tab">
      <el-tab-pane :label="`过站流水 ${flows.length}`" name="flows">
        <TableOrCards :data="flows" :columns="flowCols" empty-text="该区间无过站流水">
          <el-table :data="flows" border stripe class="rec-table" empty-text="该区间无过站流水">
            <el-table-column prop="created_at" label="时间" min-width="160" />
            <el-table-column label="类型" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" effect="plain">{{ row.event_type_label }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="board_code" label="板码" min-width="120" />
            <el-table-column prop="process_name" label="工序" min-width="110" />
            <el-table-column label="kg" width="90" align="right">
              <template #default="{ row }">{{ num(row.kg) }}</template>
            </el-table-column>
            <el-table-column label="单价" width="80" align="right">
              <template #default="{ row }">{{ num(row.rate) }}</template>
            </el-table-column>
            <el-table-column label="金额" width="100" align="right">
              <template #default="{ row }">{{ money(row.amount) }}</template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <span :class="{ muted: !row.remark }">{{ row.remark || '—' }}</span>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" effect="plain">{{ row.event_type_label }}</el-tag>
            <span class="hint">{{ num(row.kg) }} kg</span>
          </template>
        </TableOrCards>
      </el-tab-pane>

      <el-tab-pane :label="`计件日结 ${piecework.length}`" name="piecework">
        <TableOrCards :data="piecework" :columns="pieceCols" empty-text="该区间无计件日结">
          <el-table :data="piecework" border stripe class="rec-table" empty-text="该区间无计件日结">
            <el-table-column prop="biz_date" label="日期" width="120" />
            <el-table-column prop="process_name" label="工序" min-width="140" />
            <el-table-column label="产量" width="100" align="right">
              <template #default="{ row }">{{ num(row.qty) }}</template>
            </el-table-column>
            <el-table-column label="重量 kg" width="110" align="right">
              <template #default="{ row }">{{ num(row.weight) }}</template>
            </el-table-column>
            <el-table-column label="金额" width="120" align="right">
              <template #default="{ row }">{{ money(row.amount) }}</template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <span class="hint">{{ row.process_name }} · {{ money(row.amount) }}</span>
          </template>
        </TableOrCards>
      </el-tab-pane>

      <el-tab-pane :label="`工资凭证 ${wages.length}`" name="wages">
        <TableOrCards :data="wages" :columns="wageCols" empty-text="该区间无工资单行">
          <el-table :data="wages" border stripe class="rec-table" empty-text="该区间无工资单行">
            <el-table-column prop="doc_no" label="工资单" min-width="140" />
            <el-table-column prop="period_ym" label="期间" width="100" />
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="sheetTagType(row.status)">{{ row.status_label }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="用工" width="100">
              <template #default="{ row }">{{ row.emp_type_label || '—' }}</template>
            </el-table-column>
            <el-table-column label="计件" width="100" align="right">
              <template #default="{ row }">{{ money(row.piece_amount) }}</template>
            </el-table-column>
            <el-table-column label="考勤/月薪" width="110" align="right">
              <template #default="{ row }">{{ money(row.attendance_amount) }}</template>
            </el-table-column>
            <el-table-column label="提成" width="100" align="right">
              <template #default="{ row }">{{ money(row.commission_amount) }}</template>
            </el-table-column>
            <el-table-column label="调整" width="90" align="right">
              <template #default="{ row }">{{ money(row.adjust_amount) }}</template>
            </el-table-column>
            <el-table-column label="合计" width="110" align="right">
              <template #default="{ row }">{{ money(row.total_amount) }}</template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" :type="sheetTagType(row.status)">{{ row.status_label }}</el-tag>
            <span class="hint">{{ money(row.total_amount) }}</span>
          </template>
        </TableOrCards>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.rec { background: #fff; padding: 16px 18px; border-radius: 10px; border: 1px solid #e2e8ee; }
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
.row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; align-items: center; }
.hint { font-size: 12px; color: #5c6b75; }
.stats { display: grid; grid-template-columns: repeat(5, minmax(96px, 1fr)); gap: 10px; margin-bottom: 14px; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat.ok { background: #eef6f1; border-color: #d5eade; }
.stat.warn { background: #fff7f0; border-color: #f0e0d0; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; color: #1f2a33; }
.muted { color: #98a2a8; }
.rec-table :deep(.el-table__header th) { background: #f6f8fa; color: #4a5a66; font-weight: 600; }
@media (max-width: 720px) {
  .stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
