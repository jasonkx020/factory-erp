<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fieldLedgerApi } from '@erp/shared'

type Row = Record<string, unknown>
const sheets = ref<Row[]>([])
const detail = ref<Row | null>(null)
const loading = ref(false)
const genDate = ref(new Date().toISOString().slice(0, 10))
const form = reactive({
  biz_date: genDate.value,
  lines: [{ employee_name: '', process_name: '去皮', process_kind: 'piece', unit_price: 0.5, qty: 0, reject_qty: 0 }],
})

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
  const res = await fieldLedgerApi.createPieceIssueSheet({ ...form })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('领料表已建')
  detail.value = (res.data as Row) || null
  await refresh()
}

async function generate() {
  const res = await fieldLedgerApi.generatePieceIssue({ biz_date: genDate.value })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已从报工生成')
  detail.value = (res.data as Row) || null
  await refresh()
}

async function openSheet(id: number) {
  const res = await fieldLedgerApi.getPieceIssueSheet(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>计件工领料表</h2>
    <p class="hint">金额 = (数量 − 扣减不合格) × 单价；支持计时 process_kind=time</p>
    <el-card header="手工建一行">
      <el-form inline size="small">
        <el-form-item label="日期"><el-input v-model="form.biz_date" /></el-form-item>
        <el-form-item label="员工"><el-input v-model="form.lines[0].employee_name" /></el-form-item>
        <el-form-item label="工序"><el-input v-model="form.lines[0].process_name" /></el-form-item>
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
    <el-card header="从报工生成" style="margin-top:12px">
      <el-input v-model="genDate" style="width:160px;margin-right:8px" />
      <el-button @click="generate">生成草稿</el-button>
    </el-card>
    <el-table :data="sheets" size="small" style="margin-top:12px" @row-click="(row: Row) => openSheet(Number(row.id))">
      <el-table-column prop="doc_no" label="单号" />
      <el-table-column prop="biz_date" label="日期" width="120" />
      <el-table-column prop="status" label="状态" width="100" />
    </el-table>
    <el-card v-if="detail" header="明细" style="margin-top:12px">
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
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.hint { color: #667; font-size: 13px; }
</style>
