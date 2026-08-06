<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hrApi, LEAVE_TYPE_OPTIONS, OVERTIME_BIZ_OPTIONS } from '@erp/shared'
import { EmployeeSelect, WorkshopSelect, CustomerSelect, EnumSelect } from '../../components/select'

type Row = Record<string, unknown>
const props = defineProps<{ module: string }>()

const loading = ref(false)
const list = ref<Row[]>([])
const shifts = ref<Row[]>([])
const schemes = ref<Row[]>([])
const stats = ref<Row | null>(null)
const summary = ref<Row>({})

const form = reactive<Row>({})
const dlg = ref(false)
const editingId = ref<number | null>(null)

function today() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function shiftName(id: unknown) {
  const s = shifts.value.find((x) => Number(x.id) === Number(id))
  return s ? String(s.name) : String(id || '')
}

async function load() {
  loading.value = true
  try {
    const m = props.module
    if (m === '离职登记') {
      const res = await hrApi.offboards()
      list.value = ((res.data as { list?: Row[] })?.list) || []
      summary.value = ((res.data as { summary?: Row })?.summary) || {}
    } else if (m === '班次管理') {
      const res = await hrApi.shifts()
      list.value = ((res.data as { list?: Row[] })?.list) || []
      shifts.value = list.value
    } else if (m === '考勤管理') {
      const [r, s] = await Promise.all([hrApi.attendanceRules(), hrApi.shifts()])
      list.value = ((r.data as { list?: Row[] })?.list) || []
      shifts.value = ((s.data as { list?: Row[] })?.list) || []
    } else if (m === '考勤明细') {
      const [r, s] = await Promise.all([hrApi.attendanceRecords(), hrApi.shifts()])
      list.value = ((r.data as { list?: Row[] })?.list) || []
      shifts.value = ((s.data as { list?: Row[] })?.list) || []
    } else if (m === '请假管理') {
      const res = await hrApi.leaveRequests()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '加班补卡统计') {
      const [r, st] = await Promise.all([hrApi.overtimePatches(), hrApi.overtimeStats()])
      list.value = ((r.data as { list?: Row[] })?.list) || []
      stats.value = (st.data as Row) || null
    } else if (m === '考勤月度统计') {
      const res = await hrApi.monthStats()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '绩效管理') {
      const [sc, rs] = await Promise.all([hrApi.perfSchemes(), hrApi.perfResults()])
      schemes.value = ((sc.data as { list?: Row[] })?.list) || []
      list.value = ((rs.data as { list?: Row[] })?.list) || []
    } else if (m === '考勤绩效汇总') {
      const res = await hrApi.attPerfSummaries()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '外访明细') {
      const res = await hrApi.visits()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '备忘录管理') {
      const res = await hrApi.memos()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '员工日志') {
      const res = await hrApi.journals()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    }
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  const m = props.module
  Object.keys(form).forEach((k) => delete form[k])
  if (m === '离职登记') Object.assign(form, { employee_id: null, offboard_date: today(), reason: '', revoke_permission: true })
  else if (m === '班次管理') Object.assign(form, { code: `S${Date.now().toString().slice(-4)}`, name: '', start_time: '08:00', end_time: '17:00', workshop_id: 1 })
  else if (m === '考勤管理') Object.assign(form, { name: '', shift_id: shifts.value[0] ? Number(shifts.value[0].id) : 0, late_minutes: 10, early_minutes: 10 })
  else if (m === '考勤明细') Object.assign(form, { employee_id: null, biz_date: today(), punch_type: 'in', shift_id: shifts.value[0] ? Number(shifts.value[0].id) : null })
  else if (m === '请假管理') Object.assign(form, { employee_id: null, leave_type: 'annual', start_at: `${today()} 09:00:00`, end_at: `${today()} 18:00:00`, remark: '' })
  else if (m === '加班补卡统计') Object.assign(form, { employee_id: null, biz_type: 'overtime', biz_date: today(), minutes: 60, remark: '' })
  else if (m === '绩效管理') Object.assign(form, { mode: 'result', scheme_id: schemes.value[0] ? Number(schemes.value[0].id) : null, employee_id: null, period: today().slice(0, 7), score: 80, amount: 0, name: '', scheme_json: '{}' })
  else if (m === '外访明细') Object.assign(form, { employee_id: null, customer_id: null, visit_at: `${today()} 10:00:00`, content: '', location: '' })
  else if (m === '备忘录管理') Object.assign(form, { title: '', content: '', biz_date: today() })
  else if (m === '员工日志') Object.assign(form, { employee_id: null, biz_date: today(), content: '' })
  else if (m === '考勤月度统计' || m === '考勤绩效汇总') Object.assign(form, { period: today().slice(0, 7) })
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id)
  Object.keys(form).forEach((k) => delete form[k])
  Object.assign(form, { ...row })
  dlg.value = true
}

async function save() {
  const m = props.module
  let res
  if (m === '离职登记') res = await hrApi.createOffboard({ ...form })
  else if (m === '班次管理') {
    if (editingId.value) res = await hrApi.updateShift(editingId.value, { ...form })
    else res = await hrApi.createShift({ ...form })
  } else if (m === '考勤管理') {
    if (editingId.value) res = await hrApi.updateAttendanceRule(editingId.value, { ...form })
    else res = await hrApi.createAttendanceRule({ ...form })
  } else if (m === '考勤明细') res = await hrApi.punch({ ...form })
  else if (m === '请假管理') res = await hrApi.createLeave({ ...form })
  else if (m === '加班补卡统计') res = await hrApi.createOvertimePatch({ ...form })
  else if (m === '绩效管理') {
    if (form.mode === 'scheme') res = await hrApi.createPerfScheme({ name: form.name, scheme_json: form.scheme_json })
    else res = await hrApi.createPerfResult({ ...form })
  } else if (m === '外访明细') {
    if (editingId.value) res = await hrApi.updateVisit(editingId.value, { ...form })
    else res = await hrApi.createVisit({ ...form })
  } else if (m === '备忘录管理') {
    if (editingId.value) res = await hrApi.updateMemo(editingId.value, { ...form })
    else res = await hrApi.createMemo({ ...form })
  } else if (m === '员工日志') res = await hrApi.createJournal({ ...form })
  else if (m === '考勤月度统计') {
    const p = String(form.period || '')
    const [y, mo] = p.split('-').map(Number)
    res = await hrApi.recalcMonthStats({ year: y, month: mo })
  } else if (m === '考勤绩效汇总') res = await hrApi.recalcAttPerf({ period: form.period })
  else return
  if (!res || res.code !== 1) return ElMessage.error(res?.msg || '失败')
  ElMessage.success('已保存')
  dlg.value = false
  await load()
}

async function confirmOffboard(row: Row, force = false) {
  if (!force) await ElMessageBox.confirm('确认离职？将冻结账号并收回权限（若勾选）。未归还工具将拦截。', '离职确认')
  const res = await hrApi.confirmOffboard(Number(row.id), force ? { force: true } : {})
  if (res.code !== 1) {
    const msg = String(res.msg || '')
    if (msg.startsWith('OPEN_TOOLS:')) {
      const n = msg.split(':')[1]
      try {
        await ElMessageBox.confirm(`该员工仍有 ${n} 笔未归还工具，是否强制确认离职？`, '工具未还')
        return confirmOffboard(row, true)
      } catch {
        return
      }
    }
    return ElMessage.error(res.msg)
  }
  ElMessage.success('已确认离职')
  await load()
}

async function cancelLeave(row: Row) {
  const res = await hrApi.cancelLeave(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已撤销')
  await load()
}

async function approveLeave(row: Row) {
  const res = await hrApi.approveLeave(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已批准')
  await load()
}

async function rejectLeave(row: Row) {
  const res = await hrApi.rejectLeave(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已驳回')
  await load()
}

async function approveOT(row: Row) {
  const res = await hrApi.approveOvertime(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已批准')
  await load()
}

async function rejectOT(row: Row) {
  const res = await hrApi.rejectOvertime(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已驳回')
  await load()
}

async function cancelOT(row: Row) {
  const res = await hrApi.cancelOvertime(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已撤销')
  await load()
}

async function removeShift(row: Row) {
  await ElMessageBox.confirm('停用该班次？', '提示')
  const res = await hrApi.removeShift(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  await load()
}

async function removeMemo(row: Row) {
  await ElMessageBox.confirm('删除该备忘？', '提示')
  const res = await hrApi.removeMemo(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  await load()
}

const primaryBtn = () => {
  if (props.module === '考勤月度统计') return '重算月统'
  if (props.module === '考勤绩效汇总') return '重算汇总'
  if (props.module === '考勤明细') return '打卡/补卡'
  return '新建'
}

watch(() => props.module, load)
onMounted(load)
</script>

<template>
  <div v-loading="loading" class="hr-ops">
    <h2 class="title">{{ module }}</h2>
    <p class="desc">工厂人事工作台：考勤/请假/加班可审批，月统按班次规则计算迟到。</p>

    <div v-if="module === '离职登记' && summary" class="stats">
      <div class="stat"><div class="label">草稿</div><div class="value">{{ summary.draft ?? 0 }}</div></div>
      <div class="stat ok"><div class="label">已确认</div><div class="value">{{ summary.confirmed ?? 0 }}</div></div>
    </div>
    <div v-if="module === '加班补卡统计' && stats" class="stats">
      <div class="stat"><div class="label">总分钟</div><div class="value">{{ stats.total_minutes ?? 0 }}</div></div>
      <div class="stat ok"><div class="label">总小时</div><div class="value">{{ Number(stats.total_hours || 0).toFixed(1) }}</div></div>
    </div>

    <div class="row">
      <el-button type="primary" @click="openCreate">{{ primaryBtn() }}</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <el-table v-if="module === '离职登记'" :data="list" border stripe>
      <el-table-column prop="id" label="单号" width="70" />
      <el-table-column prop="offboard_date" label="离职日" width="110" />
      <el-table-column prop="emp_no" label="工号" width="110" />
      <el-table-column prop="name" label="姓名" />
      <el-table-column prop="reason" label="原因" />
      <el-table-column label="收回权限" width="90">
        <template #default="{ row }">{{ row.revoke_permission ? '是' : '否' }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="110">
        <template #default="{ row }">
          <el-button v-if="row.status === 'draft'" link type="danger" @click="confirmOffboard(row)">确认离职</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '班次管理'" :data="list" border stripe>
      <el-table-column prop="code" label="编码" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="start_time" label="开始" width="90" />
      <el-table-column prop="end_time" label="结束" width="90" />
      <el-table-column prop="workshop_id" label="车间" width="80" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="removeShift(row)">停用</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '考勤管理'" :data="list" border stripe>
      <el-table-column prop="name" label="规则名" />
      <el-table-column label="班次" width="120">
        <template #default="{ row }">{{ shiftName(row.shift_id) }}</template>
      </el-table-column>
      <el-table-column prop="late_minutes" label="迟到阈值(分)" width="120" />
      <el-table-column prop="early_minutes" label="早退阈值(分)" width="120" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="90">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '考勤明细'" :data="list" border stripe>
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="biz_date" label="日期" width="110" />
      <el-table-column prop="check_in_at" label="上班" />
      <el-table-column prop="check_out_at" label="下班" />
      <el-table-column label="班次" width="100">
        <template #default="{ row }">{{ shiftName(row.shift_id) }}</template>
      </el-table-column>
      <el-table-column prop="source" label="来源" width="90" />
    </el-table>

    <el-table v-else-if="module === '请假管理'" :data="list" border stripe>
      <el-table-column prop="doc_no" label="单号" width="140" />
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="leave_type" label="类型" width="90" />
      <el-table-column prop="start_at" label="开始" />
      <el-table-column prop="end_at" label="结束" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <template v-if="row.status === 'pending' || row.status === 'draft'">
            <el-button link type="success" @click="approveLeave(row)">批准</el-button>
            <el-button link type="warning" @click="rejectLeave(row)">驳回</el-button>
            <el-button link type="danger" @click="cancelLeave(row)">撤销</el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '加班补卡统计'" :data="list" border stripe>
      <el-table-column prop="doc_no" label="单号" width="140" />
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="biz_type" label="类型" width="90" />
      <el-table-column prop="biz_date" label="日期" width="110" />
      <el-table-column prop="minutes" label="分钟" width="80" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column prop="remark" label="备注" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <template v-if="row.status === 'pending' || row.status === 'draft'">
            <el-button link type="success" @click="approveOT(row)">批准</el-button>
            <el-button link type="warning" @click="rejectOT(row)">驳回</el-button>
            <el-button link type="danger" @click="cancelOT(row)">撤销</el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '考勤月度统计'" :data="list" border stripe>
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="year" label="年" width="70" />
      <el-table-column prop="month" label="月" width="60" />
      <el-table-column prop="work_days" label="出勤天" width="90" />
      <el-table-column prop="late_times" label="迟到次" width="90" />
      <el-table-column prop="ot_hours" label="加班时" width="90" />
      <el-table-column prop="leave_days" label="请假天" width="90" />
    </el-table>

    <template v-else-if="module === '绩效管理'">
      <h3 class="sub">绩效方案</h3>
      <el-table :data="schemes" border size="small" style="margin-bottom:12px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="status" label="状态" width="90" />
      </el-table>
      <h3 class="sub">绩效结果</h3>
      <el-table :data="list" border stripe>
        <el-table-column prop="scheme_id" label="方案" width="80" />
        <el-table-column prop="employee_id" label="员工ID" width="80" />
        <el-table-column prop="period" label="周期" width="100" />
        <el-table-column prop="score" label="得分" width="80" />
        <el-table-column prop="amount" label="金额" width="100" />
      </el-table>
    </template>

    <el-table v-else-if="module === '考勤绩效汇总'" :data="list" border stripe>
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="period" label="周期" width="110" />
      <el-table-column prop="attendance_score" label="考勤分" width="100" />
      <el-table-column prop="perf_score" label="绩效分" width="100" />
      <el-table-column prop="summary_json" label="摘要" />
    </el-table>

    <el-table v-else-if="module === '外访明细'" :data="list" border stripe>
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="customer_id" label="客户ID" width="90" />
      <el-table-column prop="visit_at" label="时间" width="160" />
      <el-table-column prop="location" label="地点" width="120" />
      <el-table-column prop="content" label="内容" />
      <el-table-column label="操作" width="80">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '备忘录管理'" :data="list" border stripe>
      <el-table-column prop="title" label="标题" />
      <el-table-column prop="biz_date" label="日期" width="110" />
      <el-table-column prop="content" label="内容" />
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="removeMemo(row)">删</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-table v-else-if="module === '员工日志'" :data="list" border stripe>
      <el-table-column prop="emp_no" label="工号" width="100" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="biz_date" label="日期" width="110" />
      <el-table-column prop="content" label="内容" />
      <el-table-column prop="created_at" label="创建时间" width="160" />
    </el-table>

    <el-dialog v-model="dlg" :title="editingId ? `编辑 · ${module}` : module" width="520px">
      <el-form label-width="100px">
        <template v-if="module === '离职登记'">
          <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
          <el-form-item label="离职日"><el-date-picker v-model="form.offboard_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
          <el-form-item label="原因"><el-input v-model="form.reason" /></el-form-item>
          <el-form-item label="收回权限"><el-switch v-model="form.revoke_permission" /></el-form-item>
        </template>
        <template v-else-if="module === '班次管理'">
          <el-form-item v-if="!editingId" label="编码"><el-input v-model="form.code" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="开始"><el-time-picker v-model="form.start_time" value-format="HH:mm" format="HH:mm" style="width:100%" /></el-form-item>
          <el-form-item label="结束"><el-time-picker v-model="form.end_time" value-format="HH:mm" format="HH:mm" style="width:100%" /></el-form-item>
          <el-form-item label="车间"><WorkshopSelect v-model="form.workshop_id" style="width:100%" /></el-form-item>
          <el-form-item v-if="editingId" label="状态"><el-select v-model="form.status" style="width:100%"><el-option label="启用" value="active" /><el-option label="停用" value="inactive" /></el-select></el-form-item>
        </template>
        <template v-else-if="module === '考勤管理'">
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="班次"><el-select v-model="form.shift_id" style="width:100%"><el-option v-for="s in shifts" :key="String(s.id)" :label="String(s.name)" :value="Number(s.id)" /></el-select></el-form-item>
          <el-form-item label="迟到阈值"><el-input-number v-model="form.late_minutes" :min="0" /></el-form-item>
          <el-form-item label="早退阈值"><el-input-number v-model="form.early_minutes" :min="0" /></el-form-item>
        </template>
        <template v-else-if="module === '考勤明细'">
          <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
          <el-form-item label="日期"><el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
          <el-form-item label="班次"><el-select v-model="form.shift_id" clearable style="width:100%"><el-option v-for="s in shifts" :key="String(s.id)" :label="String(s.name)" :value="Number(s.id)" /></el-select></el-form-item>
          <el-form-item label="类型"><el-select v-model="form.punch_type" style="width:100%"><el-option label="上班打卡" value="in" /><el-option label="下班打卡" value="out" /></el-select></el-form-item>
        </template>
        <template v-else-if="module === '请假管理'">
          <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
          <el-form-item label="类型"><EnumSelect v-model="form.leave_type" :options="LEAVE_TYPE_OPTIONS" :clearable="false" style="width:100%" /></el-form-item>
          <el-form-item label="开始"><el-date-picker v-model="form.start_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:100%" /></el-form-item>
          <el-form-item label="结束"><el-date-picker v-model="form.end_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:100%" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        </template>
        <template v-else-if="module === '加班补卡统计'">
          <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
          <el-form-item label="类型"><EnumSelect v-model="form.biz_type" :options="OVERTIME_BIZ_OPTIONS" :clearable="false" style="width:100%" /></el-form-item>
          <el-form-item label="日期"><el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
          <el-form-item label="分钟"><el-input-number v-model="form.minutes" :min="0" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        </template>
        <template v-else-if="module === '考勤月度统计'">
          <el-form-item label="年月"><el-date-picker v-model="form.period" type="month" value-format="YYYY-MM" style="width:100%" /></el-form-item>
        </template>
        <template v-else-if="module === '考勤绩效汇总'">
          <el-form-item label="周期"><el-date-picker v-model="form.period" type="month" value-format="YYYY-MM" style="width:100%" /></el-form-item>
        </template>
        <template v-else-if="module === '绩效管理'">
          <el-form-item label="录入类型"><el-radio-group v-model="form.mode"><el-radio label="result">结果</el-radio><el-radio label="scheme">方案</el-radio></el-radio-group></el-form-item>
          <template v-if="form.mode === 'scheme'">
            <el-form-item label="方案名"><el-input v-model="form.name" /></el-form-item>
          </template>
          <template v-else>
            <el-form-item label="方案"><el-select v-model="form.scheme_id" style="width:100%"><el-option v-for="s in schemes" :key="String(s.id)" :label="String(s.name)" :value="Number(s.id)" /></el-select></el-form-item>
            <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
            <el-form-item label="周期"><el-date-picker v-model="form.period" type="month" value-format="YYYY-MM" style="width:100%" /></el-form-item>
            <el-form-item label="得分"><el-input-number v-model="form.score" :min="0" :max="100" /></el-form-item>
            <el-form-item label="金额"><el-input-number v-model="form.amount" :min="0" /></el-form-item>
          </template>
        </template>
        <template v-else-if="module === '外访明细'">
          <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
          <el-form-item label="客户"><CustomerSelect v-model="form.customer_id" style="width:100%" /></el-form-item>
          <el-form-item label="时间"><el-date-picker v-model="form.visit_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:100%" /></el-form-item>
          <el-form-item label="地点"><el-input v-model="form.location" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="form.content" type="textarea" /></el-form-item>
        </template>
        <template v-else-if="module === '备忘录管理'">
          <el-form-item label="标题"><el-input v-model="form.title" /></el-form-item>
          <el-form-item label="日期"><el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="form.content" type="textarea" /></el-form-item>
        </template>
        <template v-else-if="module === '员工日志'">
          <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
          <el-form-item label="日期"><el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="form.content" type="textarea" /></el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hr-ops { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; }
.sub { margin: 8px 0; font-size: 14px; }
.stats { display: grid; grid-template-columns: repeat(2, 160px); gap: 10px; margin-bottom: 12px; }
.stat { background: #f4f7f9; border: 1px solid #e2e8ec; border-radius: 8px; padding: 10px; }
.stat.ok { background: #eef8f4; }
.stat .label { font-size: 12px; color: #5c6b75; }
.stat .value { font-size: 20px; font-weight: 600; }
</style>
