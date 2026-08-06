<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  approvalApi,
  APPROVAL_DOC_TYPE_OPTIONS,
  LEAVE_TYPE_OPTIONS,
  OVERTIME_BIZ_OPTIONS,
} from '@erp/shared'
import { EmployeeSelect, EnumSelect, SalesOrderSelect } from '../../components/select'

type Row = Record<string, unknown>

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  tasks: '任务管理',
  'doc-reviews': '单据审核',
  'expense-finance': '费用财务审批',
  'inquiry-finance': '询价财务审批',
  'inquiry-lines': '询价明细审批',
  purchases: '采购审批',
  'purchase-plans': '采购计划单审批',
  affairs: '事务申请审批',
  'expense-requests': '费用申请',
  attendance: '考勤审批',
}

const QUEUE_SECTIONS = new Set([
  'doc-reviews',
  'expense-finance',
  'inquiry-finance',
  'inquiry-lines',
  'purchases',
  'purchase-plans',
])

const active = computed(() => String(route.params.section || 'tasks'))
const title = computed(() => TITLE_MAP[active.value] || '审批管理')
const loading = ref(false)
const list = ref<Row[]>([])
const statusFilter = ref('')
const comment = ref('')

const taskForm = reactive({
  title: '',
  doc_type: 'sales_order',
  doc_id: 0,
  doc_no: '',
  amount: 0,
  remark: '',
})

const queueForm = reactive({
  title: '',
  doc_no: '',
  biz_type: '',
  biz_id: 0,
  amount: 0,
  remark: '',
})

const expenseForm = reactive({
  category: '办公费用',
  amount: 100,
  remark: '',
})

const affairForm = reactive({
  title: '',
  content: '',
  remark: '',
})

const attForm = reactive({
  kind: 'leave' as 'leave' | 'overtime_patch',
  employee_id: null as number | null,
  leave_type: 'annual',
  start_at: '',
  end_at: '',
  biz_type: 'overtime',
  biz_date: new Date().toISOString().slice(0, 10),
  minutes: 60,
  remark: '',
})

function isPending(row: Row) {
  const st = String(row.status || '')
  return st === 'pending' || st === 'submitted' || st === 'draft'
}

async function refresh() {
  loading.value = true
  list.value = []
  try {
    const qs = statusFilter.value ? `status=${encodeURIComponent(statusFilter.value)}` : undefined
    let res: { code: number; msg?: string; data?: unknown }
    const sec = active.value

    if (sec === 'tasks') {
      res = await approvalApi.tasks(qs)
    } else if (sec === 'doc-reviews') {
      res = await approvalApi.docReviews(qs)
    } else if (sec === 'expense-finance') {
      res = await approvalApi.expenseFinance(qs)
    } else if (sec === 'inquiry-finance') {
      res = await approvalApi.inquiryFinance(qs)
    } else if (sec === 'inquiry-lines') {
      res = await approvalApi.inquiryLines(qs)
    } else if (sec === 'purchases') {
      res = await approvalApi.purchases(qs)
    } else if (sec === 'purchase-plans') {
      res = await approvalApi.purchasePlans(qs)
    } else if (sec === 'affairs') {
      res = await approvalApi.affairs()
    } else if (sec === 'expense-requests') {
      res = await approvalApi.expenseRequests()
    } else if (sec === 'attendance') {
      res = await approvalApi.attendance()
    } else {
      return
    }
    if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function createTask() {
  if (!taskForm.title) return ElMessage.warning('填写标题')
  const res = await approvalApi.createTask({ ...taskForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已创建审批任务')
  taskForm.title = ''
  taskForm.doc_no = ''
  taskForm.amount = 0
  taskForm.remark = ''
  await refresh()
}

async function createQueue() {
  if (!queueForm.title) return ElMessage.warning('填写标题')
  const sec = active.value
  let res: { code: number; msg?: string }
  const body = { ...queueForm }
  if (sec === 'doc-reviews') res = await approvalApi.createDocReview(body)
  else if (sec === 'expense-finance') res = await approvalApi.createExpenseFinance(body)
  else if (sec === 'inquiry-finance') res = await approvalApi.createInquiryFinance(body)
  else if (sec === 'inquiry-lines') res = await approvalApi.createInquiryLine(body)
  else if (sec === 'purchases') res = await approvalApi.createPurchase(body)
  else if (sec === 'purchase-plans') res = await approvalApi.createPurchasePlan(body)
  else return
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交待审')
  queueForm.title = ''
  queueForm.doc_no = ''
  queueForm.amount = 0
  queueForm.remark = ''
  await refresh()
}

async function decideQueue(id: number, pass: boolean, kind?: string) {
  const tip = pass ? '确认通过？' : '确认驳回？'
  await ElMessageBox.confirm(tip, '审批确认', { type: pass ? 'success' : 'warning' })
  const cmt = comment.value || (pass ? '同意' : '驳回')
  const sec = active.value
  let res: { code: number; msg?: string }

  if (sec === 'tasks') {
    res = pass ? await approvalApi.approveTask(id, cmt) : await approvalApi.rejectTask(id, cmt)
  } else if (sec === 'doc-reviews') {
    res = pass ? await approvalApi.approveDocReview(id, cmt) : await approvalApi.rejectDocReview(id, cmt)
  } else if (sec === 'expense-finance') {
    res = pass
      ? await approvalApi.approveExpenseFinance(id, cmt)
      : await approvalApi.rejectExpenseFinance(id, cmt)
  } else if (sec === 'inquiry-finance') {
    res = pass
      ? await approvalApi.approveInquiryFinance(id, cmt)
      : await approvalApi.rejectInquiryFinance(id, cmt)
  } else if (sec === 'inquiry-lines') {
    res = pass ? await approvalApi.approveInquiryLine(id, cmt) : await approvalApi.rejectInquiryLine(id, cmt)
  } else if (sec === 'purchases') {
    res = pass ? await approvalApi.approvePurchase(id, cmt) : await approvalApi.rejectPurchase(id, cmt)
  } else if (sec === 'purchase-plans') {
    res = pass
      ? await approvalApi.approvePurchasePlan(id, cmt)
      : await approvalApi.rejectPurchasePlan(id, cmt)
  } else if (sec === 'affairs') {
    res = pass ? await approvalApi.approveAffair(id, cmt) : await approvalApi.rejectAffair(id, cmt)
  } else if (sec === 'attendance') {
    const body = { comment: cmt, kind: kind || 'leave' }
    res = pass ? await approvalApi.approveAttendance(id, body) : await approvalApi.rejectAttendance(id, body)
  } else {
    return
  }
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(pass ? '已通过' : '已驳回')
  comment.value = ''
  await refresh()
}

async function createExpense() {
  if (!expenseForm.amount) return ElMessage.warning('填写金额')
  const res = await approvalApi.createExpenseRequest({ ...expenseForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`费用申请 ${(res.data as Row)?.doc_no}`)
  expenseForm.amount = 100
  expenseForm.remark = ''
  await refresh()
}

async function submitExpense(id: number) {
  const res = await approvalApi.submitExpenseRequest(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交费用财务审批')
  await refresh()
}

async function createAffair() {
  if (!affairForm.title) return ElMessage.warning('填写标题')
  const res = await approvalApi.createAffair({ ...affairForm, submit: true })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('事务申请已提交')
  affairForm.title = ''
  affairForm.content = ''
  affairForm.remark = ''
  await refresh()
}

async function createAttendance() {
  if (!attForm.employee_id) return ElMessage.warning('请选择员工')
  const body: Record<string, unknown> = { ...attForm }
  const res = await approvalApi.createAttendance(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交考勤审批')
  attForm.remark = ''
  await refresh()
}

onMounted(refresh)
watch(active, () => {
  statusFilter.value = ''
  refresh()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <div class="actions">
        <el-select
          v-if="active !== 'expense-requests' && active !== 'affairs' && active !== 'attendance'"
          v-model="statusFilter"
          clearable
          size="small"
          placeholder="状态筛选"
          style="width: 120px"
          @change="refresh"
        >
          <el-option value="pending" label="待审" />
          <el-option value="approved" label="已通过" />
          <el-option value="rejected" label="已驳回" />
        </el-select>
        <el-input v-model="comment" size="small" placeholder="审批意见（可选）" style="width: 180px" />
        <el-button size="small" @click="refresh">刷新</el-button>
      </div>
    </div>

    <!-- 任务管理 -->
    <template v-if="active === 'tasks'">
      <el-card header="新建审批任务" class="mb">
        <el-form inline size="small">
          <el-form-item label="标题"><el-input v-model="taskForm.title" style="width:160px" /></el-form-item>
          <el-form-item label="单据类型">
            <EnumSelect v-model="taskForm.doc_type" :options="APPROVAL_DOC_TYPE_OPTIONS" :clearable="false" style="width:140px" />
          </el-form-item>
          <el-form-item label="单据">
            <SalesOrderSelect
              v-if="taskForm.doc_type === 'sales_order'"
              v-model="taskForm.doc_id"
              style="width:220px"
            />
            <el-input-number v-else v-model="taskForm.doc_id" :min="0" />
          </el-form-item>
          <el-form-item label="单号"><el-input v-model="taskForm.doc_no" style="width:130px" placeholder="可空自动" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="taskForm.amount" :min="0" :step="100" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="taskForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="createTask">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="140" />
        <el-table-column prop="title" label="标题" min-width="140" />
        <el-table-column prop="doc_type" label="类型" width="110" />
        <el-table-column prop="doc_id" label="单据ID" width="80" />
        <el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="comment" label="意见" min-width="120" show-overflow-tooltip />
        <el-table-column prop="created_at" label="创建" width="160" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button link type="success" @click="decideQueue(Number(row.id), true)">通过</el-button>
              <el-button link type="danger" @click="decideQueue(Number(row.id), false)">驳回</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 通用队列审批 -->
    <template v-else-if="QUEUE_SECTIONS.has(active)">
      <el-card header="提交待审单据" class="mb">
        <el-form inline size="small">
          <el-form-item label="标题"><el-input v-model="queueForm.title" style="width:160px" /></el-form-item>
          <el-form-item label="单号"><el-input v-model="queueForm.doc_no" style="width:130px" placeholder="可空自动" /></el-form-item>
          <el-form-item label="业务类型"><el-input v-model="queueForm.biz_type" style="width:120px" /></el-form-item>
          <el-form-item label="业务ID"><el-input-number v-model="queueForm.biz_id" :min="0" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="queueForm.amount" :min="0" :step="100" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="queueForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="createQueue">提交</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="140" />
        <el-table-column prop="title" label="标题" min-width="160" />
        <el-table-column prop="biz_type" label="业务" width="110" />
        <el-table-column prop="biz_id" label="业务ID" width="80" />
        <el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="comment" label="意见" min-width="120" show-overflow-tooltip />
        <el-table-column prop="acted_at" label="审批时间" width="160" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button link type="success" @click="decideQueue(Number(row.id), true)">通过</el-button>
              <el-button link type="danger" @click="decideQueue(Number(row.id), false)">驳回</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 费用申请 -->
    <template v-else-if="active === 'expense-requests'">
      <el-card header="新建费用申请（草稿→提交后进入费用财务审批）" class="mb">
        <el-form inline size="small">
          <el-form-item label="类别"><el-input v-model="expenseForm.category" style="width:120px" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="expenseForm.amount" :min="0" :step="50" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="expenseForm.remark" style="width:180px" /></el-form-item>
          <el-button type="primary" @click="createExpense">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" />
        <el-table-column prop="category" label="类别" width="110" />
        <el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column prop="created_at" label="创建" width="160" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'draft' || row.status === 'rejected'"
              link
              type="primary"
              @click="submitExpense(Number(row.id))"
            >提交</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 事务申请 -->
    <template v-else-if="active === 'affairs'">
      <el-card header="新建事务申请（默认直接提交审批）" class="mb">
        <el-form inline size="small">
          <el-form-item label="标题"><el-input v-model="affairForm.title" style="width:160px" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="affairForm.content" style="width:220px" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="affairForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="createAffair">提交申请</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="140" />
        <el-table-column prop="title" label="标题" min-width="140" />
        <el-table-column prop="content" label="内容" min-width="160" show-overflow-tooltip />
        <el-table-column prop="source" label="来源" width="80" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="created_at" label="创建" width="160" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <template v-if="isPending(row) && row.status !== 'approved' && row.status !== 'rejected'">
              <el-button link type="success" @click="decideQueue(Number(row.id), true)">通过</el-button>
              <el-button link type="danger" @click="decideQueue(Number(row.id), false)">驳回</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 考勤审批 -->
    <template v-else-if="active === 'attendance'">
      <el-card header="提交请假 / 加班补卡" class="mb">
        <el-form inline size="small">
          <el-form-item label="类型">
            <el-select v-model="attForm.kind" style="width:130px">
              <el-option value="leave" label="请假" />
              <el-option value="overtime_patch" label="加班补卡" />
            </el-select>
          </el-form-item>
          <el-form-item label="员工"><EmployeeSelect v-model="attForm.employee_id" /></el-form-item>
          <template v-if="attForm.kind === 'leave'">
            <el-form-item label="假别">
              <EnumSelect v-model="attForm.leave_type" :options="LEAVE_TYPE_OPTIONS" :clearable="false" style="width:120px" />
            </el-form-item>
            <el-form-item label="开始">
              <el-date-picker v-model="attForm.start_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:180px" placeholder="可空默认今日" />
            </el-form-item>
            <el-form-item label="结束">
              <el-date-picker v-model="attForm.end_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:180px" />
            </el-form-item>
          </template>
          <template v-else>
            <el-form-item label="业务">
              <EnumSelect v-model="attForm.biz_type" :options="OVERTIME_BIZ_OPTIONS" :clearable="false" style="width:120px" />
            </el-form-item>
            <el-form-item label="日期">
              <el-date-picker v-model="attForm.biz_date" type="date" value-format="YYYY-MM-DD" style="width:150px" />
            </el-form-item>
            <el-form-item label="分钟"><el-input-number v-model="attForm.minutes" :min="1" /></el-form-item>
          </template>
          <el-form-item label="备注"><el-input v-model="attForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="createAttendance">提交</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="kind" label="种类" width="110" />
        <el-table-column prop="doc_no" label="单号" width="140" />
        <el-table-column prop="title" label="标题" min-width="120" />
        <el-table-column prop="employee_name" label="员工" width="100" />
        <el-table-column prop="leave_type" label="假别" width="90" />
        <el-table-column prop="biz_date" label="日期" width="110" />
        <el-table-column prop="minutes" label="分钟" width="80" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending' || row.status === 'draft'">
              <el-button link type="success" @click="decideQueue(Number(row.id), true, String(row.kind || 'leave'))">通过</el-button>
              <el-button link type="danger" @click="decideQueue(Number(row.id), false, String(row.kind || 'leave'))">驳回</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </template>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.head h2 { margin: 0; font-size: 18px; }
.actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.mb { margin-bottom: 12px; }
</style>
