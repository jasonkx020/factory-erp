<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { fieldLedgerApi } from '@erp/shared'
import { EmployeeSelect, ProcessSelect } from '../../components/select'
import { loadEmployees, loadProcesses } from '../../components/select/entitySelects'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const sheetCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'biz_date', label: '日期' },
  { prop: 'status', label: '状态' },
]
const lineCols: MobileCardColumn[] = [
  { prop: 'employee_name', label: '员工', primary: true },
  { prop: 'seq_no', label: '序号' },
  { prop: 'process_name', label: '工序' },
  { prop: 'process_kind', label: '类型' },
  { prop: 'unit_price', label: '单价' },
  { prop: 'qty', label: '数量' },
  { prop: 'reject_qty', label: '扣减' },
  { prop: 'qty_total', label: '合计' },
  { prop: 'amount', label: '金额' },
]
const sheets = ref<Row[]>([])
const detail = ref<Row | null>(null)
const loading = ref(false)
const genDate = ref(new Date().toISOString().slice(0, 10))
const employees = ref<Row[]>([])
const processes = ref<Row[]>([])

const form = reactive({
  biz_date: genDate.value,
  lines: [{
    employee_id: null as number | null,
    employee_name: '',
    process_id: null as number | null,
    process_name: '',
    process_kind: 'piece',
    unit_price: 0.5,
    qty: 0,
    reject_qty: 0,
  }],
})

watch(
  () => form.lines[0].employee_id,
  (id) => {
    const row = employees.value.find((e) => Number(e.id) === id)
    form.lines[0].employee_name = row ? String(row.name || '') : ''
  },
)

watch(
  () => form.lines[0].process_id,
  (id) => {
    const row = processes.value.find((p) => Number(p.id) === id)
    form.lines[0].process_name = row ? String(row.name || '') : ''
  },
)

async function refresh() {
  loading.value = true
  try {
    const res = await fieldLedgerApi.pieceIssueSheets()
    if (res.code !== 1) return ElMessage.error(res.msg)
    sheets.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function create() {
  const line = form.lines[0]
  if (!line.employee_id && !line.employee_name) return ElMessage.warning('请选择员工')
  if (!line.process_id && !line.process_name) return ElMessage.warning('请选择工序')
  const res = await fieldLedgerApi.createPieceIssueSheet({
    biz_date: form.biz_date,
    lines: [{
      employee_id: line.employee_id || 0,
      employee_name: line.employee_name,
      process_id: line.process_id || 0,
      process_name: line.process_name,
      process_kind: line.process_kind,
      unit_price: line.unit_price,
      qty: line.qty,
      reject_qty: line.reject_qty,
    }],
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('领料表已建')
  detail.value = (res.data as Row) || null
  await refresh()
}

async function openSheet(id: number) {
  const res = await fieldLedgerApi.getPieceIssueSheet(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

onMounted(async () => {
  ;[employees.value, processes.value] = await Promise.all([loadEmployees(), loadProcesses()])
  await refresh()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>计件工领料表</h2>
    <p class="hint">金额 = (数量 − 扣减不合格) × 单价；支持计时 process_kind=time</p>
    <el-card header="手工建一行">
      <el-form inline size="small">
        <el-form-item label="日期">
          <el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" style="width:160px" />
        </el-form-item>
        <el-form-item label="员工">
          <EmployeeSelect v-model="form.lines[0].employee_id" />
        </el-form-item>
        <el-form-item label="工序">
          <ProcessSelect v-model="form.lines[0].process_id" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.lines[0].process_kind" style="width:100px">
            <el-option label="计件" value="piece" /><el-option label="计时" value="time" />
          </el-select>
        </el-form-item>
        <el-form-item label="单价"><el-input-number v-model="form.lines[0].unit_price" :min="0" :step="0.1" /></el-form-item>
        <el-form-item label="数量"><el-input-number v-model="form.lines[0].qty" :min="0" /></el-form-item>
        <el-form-item label="扣减"><el-input-number v-model="form.lines[0].reject_qty" :min="0" /></el-form-item>
        <el-button type="primary" @click="create">保存</el-button>
      </el-form>
    </el-card>
    <el-card header="从报工生成（已下线）" style="margin-top:12px">
      <p class="hint">旧报工链路已移除。产量请以 App 领料 + 计件日结为准。</p>
    </el-card>
    <TableOrCards :data="sheets" :loading="loading" :columns="sheetCols" style="margin-top:12px">
      <el-table :data="sheets" size="small" @row-click="(row: Row) => openSheet(Number(row.id))">
        <el-table-column prop="doc_no" label="单号" />
        <el-table-column prop="biz_date" label="日期" width="120" />
        <el-table-column prop="status" label="状态" width="100" />
      </el-table>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openSheet(Number(row.id))">查看明细</el-button>
      </template>
    </TableOrCards>
    <el-card v-if="detail" header="明细" style="margin-top:12px">
      <TableOrCards :data="(detail.lines as Row[]) || []" :columns="lineCols">
        <el-table :data="(detail.lines as Row[]) || []" size="small">
          <el-table-column prop="seq_no" label="序号" width="60" />
          <el-table-column prop="employee_name" label="员工" />
          <el-table-column prop="process_name" label="工序" />
          <el-table-column prop="process_kind" label="类型" width="70" />
          <el-table-column prop="unit_price" label="单价" width="80" />
          <el-table-column prop="qty" label="数量" width="80" />
          <el-table-column prop="reject_qty" label="扣减" width="80" />
          <el-table-column prop="qty_total" label="合计" width="80" />
          <el-table-column prop="amount" label="金额" width="90" />
        </el-table>
      </TableOrCards>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.hint { color: #667; font-size: 13px; }
</style>
