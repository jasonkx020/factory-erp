<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fieldLedgerApi } from '@erp/shared'

type Row = Record<string, unknown>
const items = ref<Row[]>([])
const issues = ref<Row[]>([])
const loading = ref(false)
const form = reactive({
  employee_name: '',
  tool_item_id: 1,
  issue_qty: 1,
  biz_date: new Date().toISOString().slice(0, 10),
})

async function refresh() {
  loading.value = true
  try {
    const [a, b] = await Promise.all([fieldLedgerApi.toolItems(), fieldLedgerApi.toolIssues()])
    items.value = ((a.data as { list?: Row[] })?.list) || []
    issues.value = ((b.data as { list?: Row[] })?.list) || []
    if (items.value.length && !form.tool_item_id) form.tool_item_id = Number(items.value[0].id)
  } finally {
    loading.value = false
  }
}

async function create() {
  const res = await fieldLedgerApi.createToolIssue({ ...form })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已领取')
  await refresh()
}

async function doReturn(row: Row) {
  const res = await fieldLedgerApi.returnToolIssue(Number(row.id), { return_qty: Number(row.issue_qty || 0) })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已交还')
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>物料工具领还</h2>
    <el-card header="领取">
      <el-form inline size="small">
        <el-form-item label="员工"><el-input v-model="form.employee_name" /></el-form-item>
        <el-form-item label="工具">
          <el-select v-model="form.tool_item_id" style="width:140px">
            <el-option v-for="t in items" :key="String(t.id)" :label="String(t.name)" :value="Number(t.id)" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量"><el-input-number v-model="form.issue_qty" :min="1" /></el-form-item>
        <el-button type="primary" @click="create">领取</el-button>
      </el-form>
    </el-card>
    <el-table :data="issues" size="small" style="margin-top:12px">
      <el-table-column prop="doc_no" label="单号" width="150" />
      <el-table-column prop="employee_name" label="员工" width="100" />
      <el-table-column prop="tool_name" label="工具" width="100" />
      <el-table-column prop="issue_qty" label="领取" width="70" />
      <el-table-column prop="return_qty" label="交还" width="70" />
      <el-table-column prop="total_qty" label="合计" width="70" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button v-if="row.status==='open'" link type="success" @click="doReturn(row)">全部交还</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
</style>
