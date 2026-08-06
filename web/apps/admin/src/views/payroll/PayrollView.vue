<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { empTypeLabel, payrollApi, PAY_ADJUST_TYPE_OPTIONS, RATE_UNIT_OPTIONS } from '@erp/shared'
import { EmployeeSelect, WorkshopSelect, ProcessSelect, EnumSelect } from '../../components/select'

type Row = Record<string, unknown>
const props = defineProps<{ module: string }>()

const loading = ref(false)
const list = ref<Row[]>([])
const sheetDetail = ref<Row | null>(null)
const calcs = ref<Row[]>([])

const form = reactive<Row>({})
const dlg = ref(false)
const detailDlg = ref(false)
const adjustDlg = ref(false)
const editingId = ref<number | null>(null)

function todayYM() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

async function load() {
  loading.value = true
  try {
    const m = props.module
    if (m === '工人信息管理') {
      const res = await payrollApi.workerProfiles()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '工资批量管理') {
      const res = await payrollApi.listSheets()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '工序工资') {
      const res = await payrollApi.wageRates()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '薪酬核算') {
      const res = await payrollApi.calculations()
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (m === '销售提成') {
      const [rules, c] = await Promise.all([payrollApi.commissionRules(), payrollApi.commissionCalcs()])
      list.value = ((rules.data as { list?: Row[] })?.list) || []
      calcs.value = ((c.data as { list?: Row[] })?.list) || []
    }
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.keys(form).forEach((k) => delete form[k])
  const m = props.module
  if (m === '工人信息管理') Object.assign(form, { employee_id: null, pay_type: 'piece', monthly_base: 0, bank_account: '', tax_no: '' })
  else if (m === '工资批量管理' || m === '薪酬核算') Object.assign(form, { period_ym: todayYM(), workshop_id: null, force: false })
  else if (m === '工序工资') Object.assign(form, { process_id: null, rate: 0.22, rate_unit: 'yuan/kg', effective_from: new Date().toISOString().slice(0, 10) })
  else if (m === '销售提成') Object.assign(form, { mode: 'rule', name: '', rate: 0.01, effective_from: new Date().toISOString().slice(0, 10), period: todayYM(), employee_id: null, base_amount: 0 })
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id || 0) || Number(row.employee_id)
  Object.keys(form).forEach((k) => delete form[k])
  Object.assign(form, { ...row, employee_id: Number(row.employee_id) })
  if (props.module === '销售提成') form.mode = 'rule'
  dlg.value = true
}

async function save() {
  const m = props.module
  let res
  if (m === '工人信息管理') {
    res = await payrollApi.saveWorkerProfile({ ...form })
  } else if (m === '工资批量管理' || m === '薪酬核算') {
    res = await payrollApi.calcSheet({ period_ym: form.period_ym, workshop_id: form.workshop_id || 0, force: !!form.force })
  } else if (m === '工序工资') {
    if (editingId.value) res = await payrollApi.updateWageRate(editingId.value, { ...form })
    else res = await payrollApi.createWageRate({ ...form })
  } else if (m === '销售提成') {
    if (form.mode === 'calc') {
      res = await payrollApi.createCommissionCalc({
        employee_id: form.employee_id,
        period: form.period,
        base_amount: form.base_amount,
        run: false,
      })
    } else if (editingId.value) {
      res = await payrollApi.updateCommissionRule(editingId.value, { name: form.name, rate: form.rate, status: form.status })
    } else {
      res = await payrollApi.createCommissionRule({ name: form.name, rate: form.rate, effective_from: form.effective_from })
    }
  } else return
  if (!res || res.code !== 1) return ElMessage.error(res?.msg || '失败')
  ElMessage.success('已保存')
  dlg.value = false
  await load()
}

async function openSheet(row: Row) {
  const res = await payrollApi.getSheet(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  sheetDetail.value = (res.data as Row) || null
  detailDlg.value = true
}

async function confirmSheet(row: Row) {
  const res = await payrollApi.confirmSheet(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已确认')
  await load()
}

async function paySheet(row: Row) {
  await ElMessageBox.confirm('确认标记该工资单为已发放？', '发放确认')
  const res = await payrollApi.paySheet(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已发放')
  await load()
}

function openAdjust(row: Row) {
  editingId.value = Number(row.id)
  Object.assign(form, { employee_id: null, amount: 0, adjust_type: 'manual', reason: '' })
  adjustDlg.value = true
}

async function saveAdjust() {
  if (!editingId.value || !form.employee_id) return ElMessage.warning('请选择员工')
  const res = await payrollApi.adjustSheet(editingId.value, { ...form })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已调整')
  adjustDlg.value = false
  await load()
}

async function removeRate(row: Row) {
  await ElMessageBox.confirm('停用该工价？', '提示')
  const res = await payrollApi.removeWageRate(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  await load()
}

async function runCommission() {
  const res = await payrollApi.runCommission({ period: todayYM() })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已生成 ${(res.data as Row)?.created ?? 0} 条提成`)
  await load()
}

const titleBtn = computed(() => {
  if (props.module === '工资批量管理' || props.module === '薪酬核算') return '按月生成工资单'
  if (props.module === '销售提成') return '新建规则'
  return '新建'
})

watch(() => props.module, load)
onMounted(load)
</script>

<template>
  <div v-loading="loading" class="pay">
    <h2 class="title">{{ module }}</h2>
    <p class="desc">工厂工资：计件汇总 + 固定月薪 + 提成 → 月结工资单（确认/发放）。</p>

    <div class="row">
      <el-button type="primary" @click="openCreate">{{ titleBtn }}</el-button>
      <el-button v-if="module === '销售提成'" @click="runCommission">按规则跑提成</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <!-- 工人档案 -->
    <el-table v-if="module === '工人信息管理'" :data="list" border stripe>
      <el-table-column prop="emp_no" label="工号" width="110" />
      <el-table-column prop="name" label="姓名" width="100" />
      <el-table-column label="用工类型" width="100">
        <template #default="{ row }">{{ empTypeLabel(row.emp_type) }}</template>
      </el-table-column>
      <el-table-column prop="pay_type" label="计薪方式" width="100" />
      <el-table-column prop="monthly_base" label="月薪基数" width="110" />
      <el-table-column prop="bank_account" label="银行卡" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="90">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 工资单 -->
    <el-table v-else-if="module === '工资批量管理'" :data="list" border stripe>
      <el-table-column prop="doc_no" label="单号" width="120" />
      <el-table-column prop="period_ym" label="期间" width="100" />
      <el-table-column prop="line_count" label="人数" width="80" />
      <el-table-column prop="total_amount" label="合计" width="110" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column prop="calc_at" label="核算时间" width="160" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openSheet(row)">明细</el-button>
          <el-button v-if="row.status === 'draft'" link @click="openAdjust(row)">调整</el-button>
          <el-button v-if="row.status === 'draft'" link type="success" @click="confirmSheet(row)">确认</el-button>
          <el-button v-if="row.status === 'confirmed' || row.status === 'draft'" link type="warning" @click="paySheet(row)">发放</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 工价 -->
    <el-table v-else-if="module === '工序工资'" :data="list" border stripe>
      <el-table-column prop="process_id" label="工序ID" width="90" />
      <el-table-column prop="process_code" label="编码" width="100" />
      <el-table-column prop="process_name" label="工序" />
      <el-table-column prop="rate" label="工价" width="100" />
      <el-table-column prop="rate_unit" label="单位" width="80" />
      <el-table-column prop="effective_from" label="生效" width="120" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="removeRate(row)">停用</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 核算日志 -->
    <el-table v-else-if="module === '薪酬核算'" :data="list" border stripe>
      <el-table-column prop="doc_no" label="单号" width="120" />
      <el-table-column prop="period_ym" label="期间" width="100" />
      <el-table-column prop="sheet_id" label="工资单ID" width="100" />
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column prop="summary_json" label="摘要" />
      <el-table-column prop="created_at" label="时间" width="160" />
    </el-table>

    <!-- 提成 -->
    <template v-else-if="module === '销售提成'">
      <h3 class="sub">提成规则</h3>
      <el-table :data="list" border size="small" style="margin-bottom:12px">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="rate" label="比例" width="100" />
        <el-table-column prop="effective_from" label="生效" width="120" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <h3 class="sub">提成结果</h3>
      <el-table :data="calcs" border stripe>
        <el-table-column prop="period" label="期间" width="100" />
        <el-table-column prop="emp_no" label="工号" width="100" />
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column prop="rule_name" label="规则" />
        <el-table-column prop="base_amount" label="基数" width="110" />
        <el-table-column prop="commission_amount" label="提成" width="110" />
      </el-table>
    </template>

    <el-dialog v-model="dlg" :title="module" width="520px">
      <el-form label-width="110px">
        <template v-if="module === '工人信息管理'">
          <el-form-item label="员工">
            <EmployeeSelect v-model="form.employee_id" style="width:100%" />
          </el-form-item>
          <el-form-item label="计薪方式">
            <el-select v-model="form.pay_type" style="width:100%">
              <el-option label="计件" value="piece" />
              <el-option label="固定月薪" value="fixed" />
              <el-option label="混合" value="mixed" />
            </el-select>
          </el-form-item>
          <el-form-item label="月薪基数"><el-input-number v-model="form.monthly_base" :min="0" :step="100" style="width:100%" /></el-form-item>
          <el-form-item label="银行卡"><el-input v-model="form.bank_account" /></el-form-item>
          <el-form-item label="税号"><el-input v-model="form.tax_no" /></el-form-item>
        </template>
        <template v-else-if="module === '工资批量管理' || module === '薪酬核算'">
          <el-form-item label="期间"><el-date-picker v-model="form.period_ym" type="month" value-format="YYYY-MM" style="width:100%" /></el-form-item>
          <el-form-item label="车间"><WorkshopSelect v-model="form.workshop_id" style="width:100%" /></el-form-item>
          <el-form-item label="强制重算"><el-switch v-model="form.force" /></el-form-item>
          <p class="hint">计件工：汇总当月计件金额；固定/职能：取月薪基数（可按迟到微调）；提成：取已算提成。</p>
        </template>
        <template v-else-if="module === '工序工资'">
          <el-form-item label="工序"><ProcessSelect v-model="form.process_id" style="width:100%" /></el-form-item>
          <el-form-item label="工价"><el-input-number v-model="form.rate" :min="0" :step="0.01" :precision="4" style="width:100%" /></el-form-item>
          <el-form-item label="单位"><EnumSelect v-model="form.rate_unit" :options="RATE_UNIT_OPTIONS" :clearable="false" style="width:100%" /></el-form-item>
          <el-form-item label="生效日"><el-date-picker v-model="form.effective_from" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
        </template>
        <template v-else-if="module === '销售提成'">
          <el-form-item label="类型">
            <el-radio-group v-model="form.mode">
              <el-radio label="rule">规则</el-radio>
              <el-radio label="calc">手工提成</el-radio>
            </el-radio-group>
          </el-form-item>
          <template v-if="form.mode === 'rule'">
            <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
            <el-form-item label="比例"><el-input-number v-model="form.rate" :min="0" :max="1" :step="0.001" :precision="4" style="width:100%" /></el-form-item>
            <el-form-item label="生效日"><el-date-picker v-model="form.effective_from" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item>
          </template>
          <template v-else>
            <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
            <el-form-item label="期间"><el-date-picker v-model="form.period" type="month" value-format="YYYY-MM" style="width:100%" /></el-form-item>
            <el-form-item label="销售基数"><el-input-number v-model="form.base_amount" :min="0" style="width:100%" /></el-form-item>
          </template>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="adjustDlg" title="工资调整" width="460px">
      <el-form label-width="100px">
        <el-form-item label="员工"><EmployeeSelect v-model="form.employee_id" style="width:100%" /></el-form-item>
        <el-form-item label="金额"><el-input-number v-model="form.amount" style="width:100%" /></el-form-item>
        <el-form-item label="类型"><EnumSelect v-model="form.adjust_type" :options="PAY_ADJUST_TYPE_OPTIONS" :clearable="false" style="width:100%" /></el-form-item>
        <el-form-item label="原因"><el-input v-model="form.reason" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustDlg = false">取消</el-button>
        <el-button type="primary" @click="saveAdjust">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="detailDlg" title="工资单明细" width="820px">
      <template v-if="sheetDetail">
        <p>{{ sheetDetail.doc_no }} · {{ sheetDetail.period_ym }} · {{ sheetDetail.status }} · 合计 {{ sheetDetail.total_amount }}</p>
        <el-table :data="(sheetDetail.lines as Row[]) || []" border size="small" max-height="420">
          <el-table-column prop="emp_no" label="工号" width="100" />
          <el-table-column prop="name" label="姓名" width="90" />
          <el-table-column prop="emp_type" label="类型" width="80" />
          <el-table-column prop="piece_amount" label="计件" width="90" />
          <el-table-column prop="attendance_amount" label="考勤/月薪" width="110" />
          <el-table-column prop="commission_amount" label="提成" width="90" />
          <el-table-column prop="adjust_amount" label="调整" width="90" />
          <el-table-column prop="total_amount" label="合计" width="100" />
        </el-table>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.pay { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.sub { margin: 8px 0; font-size: 14px; }
.hint { color: #5c6b75; font-size: 12px; margin: 0 0 0 110px; }
</style>
