<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fieldLedgerApi, ticketApi } from '@erp/shared'
import { EmployeeSelect } from '../../components/select'

type Row = Record<string, unknown>
const items = ref<Row[]>([])
const issues = ref<Row[]>([])
const pool = ref<Row[]>([])
const loading = ref(false)
const form = reactive({
  employee_id: null as number | null,
  tool_item_id: 1,
  issue_qty: 1,
  biz_date: new Date().toISOString().slice(0, 10),
  remark: '',
  next_assignee_user_id: null as number | null,
})

const statusLabel: Record<string, string> = {
  pending: '待审批发放',
  open: '在用',
  pending_return: '待确认归还',
  returned: '已还清',
  rejected: '已驳回',
}

async function loadPool() {
  const res = await ticketApi.handlerPool('category_code=tool_issue')
  pool.value = ((res.data as { pool?: Row[] })?.pool) || []
  if (pool.value.length && !form.next_assignee_user_id) {
    form.next_assignee_user_id = Number(pool.value[0].user_id)
  }
}

async function refresh() {
  loading.value = true
  try {
    await loadPool()
    const [a, b] = await Promise.all([fieldLedgerApi.toolItems(), fieldLedgerApi.toolIssues()])
    items.value = ((a.data as { list?: Row[] })?.list) || []
    issues.value = ((b.data as { list?: Row[] })?.list) || []
    if (items.value.length && !form.tool_item_id) form.tool_item_id = Number(items.value[0].id)
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!form.employee_id) return ElMessage.warning('请选择员工')
  if (!form.next_assignee_user_id) return ElMessage.warning('请指定下一手处理人')
  const res = await fieldLedgerApi.createToolIssue({ ...form })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交领取申请（工单已派发）')
  form.issue_qty = 1
  form.remark = ''
  await refresh()
}

async function approve(row: Row) {
  const res = await fieldLedgerApi.approveToolIssue(Number(row.id), {})
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已发放')
  await refresh()
}

async function reject(row: Row) {
  const res = await fieldLedgerApi.rejectToolIssue(Number(row.id), { comment: '驳回' })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已驳回')
  await refresh()
}

async function confirmReturn(row: Row) {
  const res = await fieldLedgerApi.returnConfirmToolIssue(Number(row.id), {})
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已确认归还')
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
    <p class="desc">
      申请后创建工单并派给指定处理人；也可在「系统管理 / 工单中心」跟踪。离职确认会校验未归还工具。
    </p>
    <el-card header="代登领取申请">
      <el-form inline size="small">
        <el-form-item label="员工">
          <EmployeeSelect v-model="form.employee_id" />
        </el-form-item>
        <el-form-item label="工具">
          <el-select v-model="form.tool_item_id" style="width:140px">
            <el-option v-for="t in items" :key="String(t.id)" :label="String(t.name)" :value="Number(t.id)" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量"><el-input-number v-model="form.issue_qty" :min="1" /></el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" style="width:150px" />
        </el-form-item>
        <el-form-item label="下一手">
          <el-select v-model="form.next_assignee_user_id" filterable style="width:140px">
            <el-option
              v-for="p in pool"
              :key="String(p.user_id)"
              :label="String(p.name || p.login_name)"
              :value="Number(p.user_id)"
            />
          </el-select>
        </el-form-item>
        <el-button type="primary" @click="create">提交申请</el-button>
        <el-button @click="refresh">刷新</el-button>
      </el-form>
    </el-card>
    <el-table :data="issues" size="small" style="margin-top:12px" border stripe>
      <el-table-column prop="biz_date" label="日期" width="110" />
      <el-table-column prop="seq_no" label="序号" width="70" />
      <el-table-column prop="doc_no" label="单号" width="150" />
      <el-table-column prop="employee_name" label="员工" width="100" />
      <el-table-column prop="tool_name" label="工具" width="100" />
      <el-table-column prop="issue_qty" label="领取" width="70" />
      <el-table-column prop="return_qty" label="交还" width="70" />
      <el-table-column prop="total_qty" label="合计" width="70" />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">{{ statusLabel[String(row.status)] || row.status }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.status==='pending'" link type="success" @click="approve(row)">发放</el-button>
          <el-button v-if="row.status==='pending'" link type="danger" @click="reject(row)">驳回</el-button>
          <el-button v-if="row.status==='pending_return'" link type="success" @click="confirmReturn(row)">确认归还</el-button>
          <el-button v-if="row.status==='open'" link type="warning" @click="doReturn(row)">快捷全还</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.page { padding: 16px; background: #fff; border-radius: 8px; border: 1px solid #d5dde3; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
</style>
