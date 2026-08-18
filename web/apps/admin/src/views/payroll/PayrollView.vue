<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  empTypeLabel,
  normalizeRateUnit,
  payrollApi,
  PAY_ADJUST_TYPE_OPTIONS,
  PAY_TYPE_OPTIONS,
  paySheetStatusLabel,
  payTypeLabel,
  RATE_UNIT_OPTIONS,
  rateUnitLabel,
  STATUS_ACTIVE_OPTIONS,
  statusActiveLabel,
} from '@erp/shared'
import { EmployeeSelect, WorkshopSelect, ProcessSelect, EnumSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import { downloadExcel } from '../../utils/exportExcel'

type Row = Record<string, unknown>
const props = defineProps<{ module: string }>()

const workerCols: MobileCardColumn[] = [
  { prop: 'name', label: '姓名', primary: true },
  { prop: 'emp_no', label: '工号' },
  { prop: 'emp_type_label', label: '用工类型' },
  { prop: 'pay_type_label', label: '计薪方式' },
  { prop: 'monthly_base_display', label: '月薪基数' },
  { prop: 'bank_account', label: '银行卡' },
  { prop: 'tax_no', label: '税号' },
  { prop: 'status_label', label: '状态' },
]
const sheetCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'period_ym', label: '期间' },
  { prop: 'line_count', label: '人数' },
  { prop: 'total_amount_display', label: '合计' },
  { prop: 'status_label', label: '状态' },
  { prop: 'calc_at', label: '核算时间' },
]
const rateCols: MobileCardColumn[] = [
  { prop: 'process_name', label: '工序', primary: true },
  { prop: 'process_code', label: '编码' },
  { prop: 'rate_display', label: '工价' },
  { prop: 'rate_unit_label', label: '单位' },
  { prop: 'effective_from', label: '生效日' },
  { prop: 'status_label', label: '状态' },
]
const calcCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'period_ym', label: '期间' },
  { prop: 'sheet_id', label: '工资单ID' },
  { prop: 'status', label: '状态' },
  { prop: 'summary_json', label: '摘要' },
  { prop: 'created_at', label: '时间' },
]
const commissionRuleCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'rate', label: '比例' },
  { prop: 'effective_from', label: '生效' },
  { prop: 'status', label: '状态' },
]
const commissionCalcCols: MobileCardColumn[] = [
  { prop: 'name', label: '姓名', primary: true },
  { prop: 'period', label: '期间' },
  { prop: 'emp_no', label: '工号' },
  { prop: 'rule_name', label: '规则' },
  { prop: 'base_amount', label: '基数' },
  { prop: 'commission_amount', label: '提成' },
]
const sheetLineCols: MobileCardColumn[] = [
  { prop: 'name', label: '姓名', primary: true },
  { prop: 'emp_no', label: '工号' },
  { prop: 'emp_type_label', label: '用工类型' },
  { prop: 'piece_amount_display', label: '计件' },
  { prop: 'attendance_amount_display', label: '考勤/月薪' },
  { prop: 'commission_amount_display', label: '提成' },
  { prop: 'adjust_amount_display', label: '调整' },
  { prop: 'total_amount_display', label: '合计' },
  { prop: 'bank_account', label: '银行卡' },
]

const loading = ref(false)
const exporting = ref(false)
const list = ref<Row[]>([])
const sheetDetail = ref<Row | null>(null)
const sheetDetailLines = ref<Row[]>([])
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

function formatMoney(v: unknown) {
  const n = Number(v ?? 0)
  if (!Number.isFinite(n)) return '—'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
}

function moneyNum(v: unknown) {
  const n = Number(v ?? 0)
  return Number.isFinite(n) ? n : 0
}

function mapSheetRow(r: Row): Row {
  return {
    ...r,
    status_label: paySheetStatusLabel(r.status),
    total_amount_display: formatMoney(r.total_amount),
  }
}

function mapSheetLine(r: Row): Row {
  return {
    ...r,
    emp_type_label: empTypeLabel(r.emp_type),
    pay_type_label: payTypeLabel(r.pay_type),
    piece_amount_display: formatMoney(r.piece_amount),
    attendance_amount_display: formatMoney(r.attendance_amount),
    commission_amount_display: formatMoney(r.commission_amount),
    adjust_amount_display: formatMoney(r.adjust_amount),
    total_amount_display: formatMoney(r.total_amount),
  }
}

function sheetStatusTagType(status: unknown): 'info' | 'warning' | 'success' {
  const s = String(status || '')
  if (s === 'paid') return 'success'
  if (s === 'confirmed') return 'warning'
  return 'info'
}

function payTypeTagType(v: unknown): 'warning' | 'success' | 'info' | '' {
  const s = String(v || '').toLowerCase()
  if (s === 'piece') return 'warning'
  if (s === 'fixed') return 'success'
  if (s === 'mixed') return 'info'
  if (s === 'commission') return 'warning'
  return ''
}

function empTypeTagType(v: unknown): 'warning' | 'success' | 'info' | '' {
  const s = String(v || '').toLowerCase()
  if (s === 'piece' || s === 'temp') return 'warning'
  if (s === 'fixed') return 'success'
  if (s === 'office' || s === 'admin' || s === 'sys_admin') return 'info'
  if (s === 'warehouse' || s === 'sales' || s === 'purchase' || s === 'qc' || s === 'foreman' || s === 'finance' || s === 'hr') {
    return 'info'
  }
  return ''
}

async function load() {
  loading.value = true
  try {
    const m = props.module
    if (m === '工人信息管理') {
      const res = await payrollApi.workerProfiles()
      const rows = ((res.data as { list?: Row[] })?.list) || []
      list.value = rows.map((r) => ({
        ...r,
        emp_type_label: empTypeLabel(r.emp_type),
        pay_type_label: payTypeLabel(r.pay_type),
        status_label: statusActiveLabel(r.status),
        monthly_base_display: formatMoney(r.monthly_base),
      }))
    } else if (m === '工资批量管理') {
      const res = await payrollApi.listSheets()
      const rows = ((res.data as { list?: Row[] })?.list) || []
      list.value = rows.map(mapSheetRow)
    } else if (m === '工序工资') {
      const res = await payrollApi.wageRates()
      const rows = ((res.data as { list?: Row[] })?.list) || []
      list.value = rows.map((r) => {
        const unit = normalizeRateUnit(r.rate_unit)
        const rate = Number(r.rate ?? 0)
        return {
          ...r,
          rate_unit: unit,
          rate_unit_label: rateUnitLabel(unit),
          status_label: statusActiveLabel(r.status),
          rate_display: `${Number.isFinite(rate) ? rate.toFixed(4).replace(/\.?0+$/, '') : r.rate} ${rateUnitLabel(unit)}`,
        }
      })
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
  if (m === '工人信息管理') Object.assign(form, { employee_id: null, pay_type: 'piece', monthly_base: 0, bank_account: '', tax_no: '', status: 'active' })
  else if (m === '工资批量管理' || m === '薪酬核算') Object.assign(form, { period_ym: todayYM(), workshop_dept_id: null, force: false })
  else if (m === '工序工资') Object.assign(form, { process_id: null, rate: 0.22, rate_unit: 'yuan/kg', effective_from: new Date().toISOString().slice(0, 10) })
  else if (m === '销售提成') Object.assign(form, { mode: 'rule', name: '', rate: 0.01, effective_from: new Date().toISOString().slice(0, 10), period: todayYM(), employee_id: null, base_amount: 0 })
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id || 0) || Number(row.employee_id)
  Object.keys(form).forEach((k) => delete form[k])
  Object.assign(form, { ...row, employee_id: Number(row.employee_id) })
  if (props.module === '工人信息管理') {
    editingId.value = Number(row.employee_id) || null
    form.employee_id = Number(row.employee_id) || null
    form.pay_type = String(row.pay_type || 'piece')
    form.monthly_base = Number(row.monthly_base) || 0
    form.bank_account = String(row.bank_account || '')
    form.tax_no = String(row.tax_no || '')
    form.status = String(row.status || 'active')
  }
  if (props.module === '工序工资') {
    form.rate_unit = normalizeRateUnit(row.rate_unit)
    form.process_id = Number(row.process_id) || null
    form.rate = Number(row.rate) || 0
  }
  if (props.module === '销售提成') form.mode = 'rule'
  dlg.value = true
}

async function save() {
  const m = props.module
  let res
  if (m === '工人信息管理') {
    if (!form.employee_id) return ElMessage.warning('请选择员工')
    res = await payrollApi.saveWorkerProfile({
      employee_id: Number(form.employee_id),
      pay_type: String(form.pay_type || 'piece'),
      monthly_base: Number(form.monthly_base) || 0,
      bank_account: String(form.bank_account || ''),
      tax_no: String(form.tax_no || ''),
      status: String(form.status || 'active'),
    })
  } else if (m === '工资批量管理' || m === '薪酬核算') {
    res = await payrollApi.calcSheet({ period_ym: form.period_ym, workshop_dept_id: form.workshop_dept_id || 0, force: !!form.force })
  } else if (m === '工序工资') {
    if (!form.process_id) return ElMessage.warning('请选择工序')
    const payload = {
      process_id: Number(form.process_id),
      rate: Number(form.rate) || 0,
      rate_unit: normalizeRateUnit(form.rate_unit),
      effective_from: form.effective_from,
    }
    if (editingId.value) res = await payrollApi.updateWageRate(editingId.value, payload)
    else res = await payrollApi.createWageRate(payload)
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
  const data = (res.data as Row) || {}
  const lines = ((data.lines as Row[]) || []).map(mapSheetLine)
  sheetDetail.value = {
    ...mapSheetRow(data),
    line_count: lines.length,
  }
  sheetDetailLines.value = lines
  detailDlg.value = true
}

function buildSheetDetailRows(detail: Row, lines: Row[]) {
  const summary: (string | number)[][] = [
    ['工资单明细'],
    ['单号', String(detail.doc_no || '')],
    ['期间', String(detail.period_ym || '')],
    ['状态', paySheetStatusLabel(detail.status)],
    ['人数', lines.length],
    ['合计金额', moneyNum(detail.total_amount)],
    ['核算时间', String(detail.calc_at || '')],
    ['发放时间', String(detail.paid_at || '')],
    [],
    ['工号', '姓名', '用工类型', '计薪方式', '计件', '考勤/月薪', '提成', '调整', '合计', '银行卡', '税号'],
  ]
  for (const r of lines) {
    summary.push([
      String(r.emp_no || ''),
      String(r.name || ''),
      empTypeLabel(r.emp_type),
      payTypeLabel(r.pay_type) === '—' ? '' : payTypeLabel(r.pay_type),
      moneyNum(r.piece_amount),
      moneyNum(r.attendance_amount),
      moneyNum(r.commission_amount),
      moneyNum(r.adjust_amount),
      moneyNum(r.total_amount),
      String(r.bank_account || ''),
      String(r.tax_no || ''),
    ])
  }
  return summary
}

async function exportSheetExcel(row?: Row) {
  exporting.value = true
  try {
    let detail = sheetDetail.value
    let lines = sheetDetailLines.value
    const id = Number(row?.id || detail?.id || 0)
    if (row || !detail) {
      if (!id) return ElMessage.warning('请选择工资单')
      const res = await payrollApi.getSheet(id)
      if (res.code !== 1) return ElMessage.error(res.msg || '加载失败')
      detail = mapSheetRow((res.data as Row) || {})
      lines = (((res.data as Row)?.lines as Row[]) || []).map(mapSheetLine)
      detail.line_count = lines.length
      detail.total_amount = (res.data as Row)?.total_amount
    }
    if (!detail) return
    const docNo = String(detail.doc_no || id)
    const period = String(detail.period_ym || '')
    downloadExcel(
      [{ name: '工资明细', rows: buildSheetDetailRows(detail, lines) }],
      `工资单_${docNo}_${period || 'export'}`,
    )
    ElMessage.success('已导出 Excel')
  } catch (e) {
    ElMessage.error(`导出失败：${e instanceof Error ? e.message : e}`)
  } finally {
    exporting.value = false
  }
}

function exportSheetsListExcel() {
  if (!list.value.length) return ElMessage.warning('暂无工资单可导出')
  const rows: (string | number)[][] = [
    ['工资单列表'],
    ['单号', '期间', '人数', '合计金额', '状态', '核算时间', '发放时间', '备注'],
  ]
  for (const r of list.value) {
    rows.push([
      String(r.doc_no || ''),
      String(r.period_ym || ''),
      moneyNum(r.line_count),
      moneyNum(r.total_amount),
      paySheetStatusLabel(r.status),
      String(r.calc_at || ''),
      String(r.paid_at || ''),
      String(r.remark || ''),
    ])
  }
  downloadExcel([{ name: '工资单列表', rows }], `工资单列表_${todayYM()}`)
  ElMessage.success('已导出 Excel')
}

const sheetDetailTotals = computed(() => {
  const lines = sheetDetailLines.value
  const sum = (key: string) => lines.reduce((a, r) => a + moneyNum(r[key]), 0)
  return {
    piece: sum('piece_amount'),
    attendance: sum('attendance_amount'),
    commission: sum('commission_amount'),
    adjust: sum('adjust_amount'),
    total: sum('total_amount'),
    count: lines.length,
  }
})

function sheetLineSummary({ columns }: { columns: { property?: string }[] }) {
  const sums: string[] = []
  const t = sheetDetailTotals.value
  columns.forEach((col, i) => {
    if (i === 0) {
      sums[i] = '合计'
      return
    }
    const p = col.property
    if (p === 'name') {
      sums[i] = `${t.count} 人`
      return
    }
    if (p === 'piece_amount') sums[i] = formatMoney(t.piece)
    else if (p === 'attendance_amount') sums[i] = formatMoney(t.attendance)
    else if (p === 'commission_amount') sums[i] = formatMoney(t.commission)
    else if (p === 'adjust_amount') sums[i] = formatMoney(t.adjust)
    else if (p === 'total_amount') sums[i] = formatMoney(t.total)
    else sums[i] = ''
  })
  return sums
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
  if (props.module === '工序工资') return '新建工价'
  if (props.module === '工人信息管理') return '新建档案'
  return '新建'
})

const pageDesc = computed(() => {
  if (props.module === '工序工资') {
    return '按工序维护计件/计重工价；过站日结时按启用中的工价核算。同工序新建会自动停用旧费率。'
  }
  if (props.module === '工人信息管理') {
    return '维护工人计薪方式、月薪基数与收款银行卡；与人事档案银行卡同源，供月结工资单使用。'
  }
  if (props.module === '工资批量管理') {
    return '按月汇总计件、月薪与提成生成工资单；确认后可发放，支持导出 Excel 明细（含银行卡）。'
  }
  return '工厂工资：计件汇总 + 固定月薪 + 提成 → 月结工资单（确认/发放）。'
})

const dialogTitle = computed(() => {
  if (props.module === '工序工资') return editingId.value ? '编辑工序工价' : '新建工序工价'
  if (props.module === '工人信息管理') return editingId.value ? '编辑薪资档案' : '新建薪资档案'
  if (props.module === '工资批量管理' || props.module === '薪酬核算') return '按月生成工资单'
  return props.module
})

const headMetaText = computed(() => {
  if (props.module === '工序工资') return `启用 ${list.value.length} 条`
  if (props.module === '工人信息管理') return `共 ${list.value.length} 人`
  if (props.module === '工资批量管理') return `共 ${list.value.length} 单`
  return ''
})

function statusTagType(status: unknown): 'success' | 'info' | 'danger' {
  const s = String(status || '')
  if (s === 'active') return 'success'
  if (s === 'inactive') return 'info'
  return 'danger'
}

watch(() => props.module, load)
onMounted(load)
</script>

<template>
  <div v-loading="loading" class="pay">
    <header class="page-head">
      <div>
        <h2 class="title">{{ module }}</h2>
        <p class="desc">{{ pageDesc }}</p>
      </div>
      <div v-if="headMetaText" class="head-meta">
        <span class="meta-pill">{{ headMetaText }}</span>
      </div>
    </header>

    <div class="row">
      <el-button type="primary" @click="openCreate">{{ titleBtn }}</el-button>
      <el-button
        v-if="module === '工资批量管理'"
        type="success"
        plain
        :disabled="!list.length"
        :loading="exporting"
        @click="exportSheetsListExcel"
      >导出列表 Excel</el-button>
      <el-button v-if="module === '销售提成'" @click="runCommission">按规则跑提成</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <!-- 工人档案 -->
    <TableOrCards
      v-if="module === '工人信息管理'"
      :data="list"
      :loading="loading"
      :columns="workerCols"
      empty-text="暂无工人薪资档案，请点击「新建档案」"
    >
      <el-table :data="list" border stripe class="worker-table" empty-text="暂无工人薪资档案">
        <el-table-column prop="emp_no" label="工号" width="110" />
        <el-table-column prop="name" label="姓名" min-width="110">
          <template #default="{ row }">
            <div class="proc-cell">
              <span class="proc-name">{{ row.name || '—' }}</span>
              <span v-if="row.employee_id" class="proc-id">#{{ row.employee_id }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="用工类型" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" :type="empTypeTagType(row.emp_type)">{{ row.emp_type_label || empTypeLabel(row.emp_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="计薪方式" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="payTypeTagType(row.pay_type)">{{ row.pay_type_label || payTypeLabel(row.pay_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="月薪基数" width="120" align="right">
          <template #default="{ row }">
            <span class="rate-num">{{ row.monthly_base_display || formatMoney(row.monthly_base) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="银行卡" min-width="160">
          <template #default="{ row }">
            <span :class="{ 'muted-cell': !row.bank_account }">{{ row.bank_account || '未填写' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="税号" min-width="120">
          <template #default="{ row }">
            <span :class="{ 'muted-cell': !row.tax_no }">{{ row.tax_no || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label || statusActiveLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #field-emp_type_label="{ row }">
        <el-tag size="small" effect="plain" :type="empTypeTagType(row.emp_type)">{{ row.emp_type_label }}</el-tag>
      </template>
      <template #field-pay_type_label="{ row }">
        <el-tag size="small" :type="payTypeTagType(row.pay_type)">{{ row.pay_type_label }}</el-tag>
      </template>
      <template #field-status_label="{ row }">
        <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
      </template>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
      </template>
    </TableOrCards>

    <!-- 工资单 -->
    <TableOrCards
      v-else-if="module === '工资批量管理'"
      :data="list"
      :loading="loading"
      :columns="sheetCols"
      empty-text="暂无工资单，请点击「按月生成工资单」"
    >
      <el-table :data="list" border stripe class="sheet-table" empty-text="暂无工资单">
        <el-table-column prop="doc_no" label="单号" min-width="130">
          <template #default="{ row }">
            <span class="proc-name">{{ row.doc_no || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="period_ym" label="期间" width="100" />
        <el-table-column prop="line_count" label="人数" width="80" align="center" />
        <el-table-column label="合计" width="130" align="right">
          <template #default="{ row }">
            <span class="rate-num">{{ row.total_amount_display || formatMoney(row.total_amount) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="sheetStatusTagType(row.status)">{{ row.status_label || paySheetStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="calc_at" label="核算时间" min-width="160" />
        <el-table-column prop="paid_at" label="发放时间" min-width="160">
          <template #default="{ row }">
            <span :class="{ 'muted-cell': !row.paid_at }">{{ row.paid_at || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openSheet(row)">明细</el-button>
            <el-button link type="success" :loading="exporting" @click="exportSheetExcel(row)">导出</el-button>
            <el-button v-if="row.status === 'draft'" link @click="openAdjust(row)">调整</el-button>
            <el-button v-if="row.status === 'draft'" link type="success" @click="confirmSheet(row)">确认</el-button>
            <el-button v-if="row.status === 'confirmed' || row.status === 'draft'" link type="warning" @click="paySheet(row)">发放</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #field-status_label="{ row }">
        <el-tag size="small" :type="sheetStatusTagType(row.status)">{{ row.status_label }}</el-tag>
      </template>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openSheet(row)">明细</el-button>
        <el-button link type="success" :loading="exporting" @click="exportSheetExcel(row)">导出</el-button>
        <el-button v-if="row.status === 'draft'" link @click="openAdjust(row)">调整</el-button>
        <el-button v-if="row.status === 'draft'" link type="success" @click="confirmSheet(row)">确认</el-button>
        <el-button v-if="row.status === 'confirmed' || row.status === 'draft'" link type="warning" @click="paySheet(row)">发放</el-button>
      </template>
    </TableOrCards>

    <!-- 工价 -->
    <TableOrCards v-else-if="module === '工序工资'" :data="list" :loading="loading" :columns="rateCols" empty-text="暂无启用中的工序工价，请点击「新建工价」">
      <el-table :data="list" border stripe class="rate-table" empty-text="暂无启用中的工序工价">
        <el-table-column prop="process_code" label="工序编码" width="120" />
        <el-table-column prop="process_name" label="工序名称" min-width="140">
          <template #default="{ row }">
            <div class="proc-cell">
              <span class="proc-name">{{ row.process_name || '—' }}</span>
              <span v-if="row.process_id" class="proc-id">#{{ row.process_id }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="工价" width="160" align="right">
          <template #default="{ row }">
            <span class="rate-num">{{ Number(row.rate ?? 0).toFixed(4).replace(/\.?0+$/, '') }}</span>
          </template>
        </el-table-column>
        <el-table-column label="单位" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" type="warning">{{ row.rate_unit_label || rateUnitLabel(row.rate_unit) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="effective_from" label="生效日" width="120" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label || statusActiveLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="row.status === 'active'" link type="danger" @click="removeRate(row)">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #field-rate_unit_label="{ row }">
        <el-tag size="small" effect="plain" type="warning">{{ row.rate_unit_label }}</el-tag>
      </template>
      <template #field-status_label="{ row }">
        <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
      </template>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button v-if="row.status === 'active'" link type="danger" @click="removeRate(row)">停用</el-button>
      </template>
    </TableOrCards>

    <!-- 核算日志 -->
    <TableOrCards v-else-if="module === '薪酬核算'" :data="list" :loading="loading" :columns="calcCols">
      <el-table :data="list" border stripe>
        <el-table-column prop="doc_no" label="单号" width="120" />
        <el-table-column prop="period_ym" label="期间" width="100" />
        <el-table-column prop="sheet_id" label="工资单ID" width="100" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="summary_json" label="摘要" />
        <el-table-column prop="created_at" label="时间" width="160" />
      </el-table>
    </TableOrCards>

    <!-- 提成 -->
    <template v-else-if="module === '销售提成'">
      <h3 class="sub">提成规则</h3>
      <TableOrCards :data="list" :loading="loading" :columns="commissionRuleCols" style="margin-bottom:12px">
        <el-table :data="list" border size="small">
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
        <template #actions="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        </template>
      </TableOrCards>
      <h3 class="sub">提成结果</h3>
      <TableOrCards :data="calcs" :loading="loading" :columns="commissionCalcCols">
        <el-table :data="calcs" border stripe>
          <el-table-column prop="period" label="期间" width="100" />
          <el-table-column prop="emp_no" label="工号" width="100" />
          <el-table-column prop="name" label="姓名" width="100" />
          <el-table-column prop="rule_name" label="规则" />
          <el-table-column prop="base_amount" label="基数" width="110" />
          <el-table-column prop="commission_amount" label="提成" width="110" />
        </el-table>
      </TableOrCards>
    </template>

    <el-dialog v-model="dlg" :title="dialogTitle" width="520px" destroy-on-close>
      <el-form label-width="110px">
        <template v-if="module === '工人信息管理'">
          <el-form-item label="员工" required>
            <EmployeeSelect v-model="form.employee_id" style="width:100%" :disabled="!!editingId" />
          </el-form-item>
          <el-form-item label="计薪方式" required>
            <EnumSelect v-model="form.pay_type" :options="PAY_TYPE_OPTIONS" :clearable="false" style="width:100%" />
          </el-form-item>
          <el-form-item label="月薪基数">
            <el-input-number v-model="form.monthly_base" :min="0" :step="100" :precision="2" controls-position="right" style="width:100%" />
          </el-form-item>
          <el-form-item label="银行卡">
            <el-input v-model="form.bank_account" placeholder="收款账号，与人事档案同源" maxlength="64" show-word-limit />
          </el-form-item>
          <el-form-item label="税号">
            <el-input v-model="form.tax_no" placeholder="可选" maxlength="64" />
          </el-form-item>
          <el-form-item label="状态">
            <EnumSelect v-model="form.status" :options="STATUS_ACTIVE_OPTIONS" :clearable="false" style="width:100%" />
          </el-form-item>
          <p class="hint">计件：按工序日结；固定月薪：取月薪基数；混合：计件 + 月薪基数；提成：销售提成核算。</p>
        </template>
        <template v-else-if="module === '工资批量管理' || module === '薪酬核算'">
          <el-form-item label="期间"><el-date-picker v-model="form.period_ym" type="month" value-format="YYYY-MM" style="width:100%" /></el-form-item>
          <el-form-item label="车间"><WorkshopSelect v-model="form.workshop_dept_id" style="width:100%" /></el-form-item>
          <el-form-item label="强制重算"><el-switch v-model="form.force" /></el-form-item>
          <p class="hint">计件工：汇总当月计件金额；固定/职能：取月薪基数（可按迟到微调）；提成：取已算提成。</p>
        </template>
        <template v-else-if="module === '工序工资'">
          <el-form-item label="工序" required>
            <ProcessSelect v-model="form.process_id" style="width:100%" :disabled="!!editingId" />
          </el-form-item>
          <el-form-item label="工价" required>
            <el-input-number v-model="form.rate" :min="0" :step="0.01" :precision="4" controls-position="right" style="width:100%" />
          </el-form-item>
          <el-form-item label="单位" required>
            <EnumSelect v-model="form.rate_unit" :options="RATE_UNIT_OPTIONS" :clearable="false" style="width:100%" />
          </el-form-item>
          <el-form-item label="生效日">
            <el-date-picker v-model="form.effective_from" type="date" value-format="YYYY-MM-DD" style="width:100%" />
          </el-form-item>
          <p class="hint rate-hint">展示示例：{{ Number(form.rate || 0).toFixed(4).replace(/\.?0+$/, '') || '0' }} {{ rateUnitLabel(form.rate_unit) }}</p>
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

    <el-dialog v-model="detailDlg" title="工资单明细" width="960px" destroy-on-close class="sheet-detail-dlg">
      <template v-if="sheetDetail">
        <div class="detail-head">
          <div class="detail-title">
            <span class="proc-name">{{ sheetDetail.doc_no }}</span>
            <el-tag size="small" :type="sheetStatusTagType(sheetDetail.status)">{{ sheetDetail.status_label || paySheetStatusLabel(sheetDetail.status) }}</el-tag>
          </div>
          <el-button type="success" plain :loading="exporting" @click="exportSheetExcel()">导出明细 Excel</el-button>
        </div>
        <div class="detail-meta">
          <span>期间 {{ sheetDetail.period_ym }}</span>
          <span>人数 {{ sheetDetailTotals.count }}</span>
          <span>核算 {{ sheetDetail.calc_at || '—' }}</span>
          <span>发放 {{ sheetDetail.paid_at || '—' }}</span>
        </div>
        <div class="detail-stats">
          <div class="stat"><div class="stat-lab">计件</div><div class="stat-val">{{ formatMoney(sheetDetailTotals.piece) }}</div></div>
          <div class="stat"><div class="stat-lab">考勤/月薪</div><div class="stat-val">{{ formatMoney(sheetDetailTotals.attendance) }}</div></div>
          <div class="stat"><div class="stat-lab">提成</div><div class="stat-val">{{ formatMoney(sheetDetailTotals.commission) }}</div></div>
          <div class="stat"><div class="stat-lab">调整</div><div class="stat-val">{{ formatMoney(sheetDetailTotals.adjust) }}</div></div>
          <div class="stat stat-total"><div class="stat-lab">合计</div><div class="stat-val">{{ formatMoney(sheetDetailTotals.total) }}</div></div>
        </div>
        <TableOrCards :data="sheetDetailLines" :columns="sheetLineCols" empty-text="暂无明细行">
          <el-table :data="sheetDetailLines" border size="small" max-height="420" class="sheet-table" show-summary :summary-method="sheetLineSummary">
            <el-table-column prop="emp_no" label="工号" width="100" />
            <el-table-column prop="name" label="姓名" width="100" />
            <el-table-column label="用工类型" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" effect="plain">{{ row.emp_type_label || empTypeLabel(row.emp_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="piece_amount" label="计件" width="100" align="right">
              <template #default="{ row }"><span class="rate-num">{{ row.piece_amount_display }}</span></template>
            </el-table-column>
            <el-table-column prop="attendance_amount" label="考勤/月薪" width="110" align="right">
              <template #default="{ row }"><span class="rate-num">{{ row.attendance_amount_display }}</span></template>
            </el-table-column>
            <el-table-column prop="commission_amount" label="提成" width="100" align="right">
              <template #default="{ row }"><span class="rate-num">{{ row.commission_amount_display }}</span></template>
            </el-table-column>
            <el-table-column prop="adjust_amount" label="调整" width="100" align="right">
              <template #default="{ row }"><span class="rate-num">{{ row.adjust_amount_display }}</span></template>
            </el-table-column>
            <el-table-column prop="total_amount" label="合计" width="110" align="right">
              <template #default="{ row }"><span class="rate-num">{{ row.total_amount_display }}</span></template>
            </el-table-column>
            <el-table-column label="银行卡" min-width="140">
              <template #default="{ row }">
                <span :class="{ 'muted-cell': !row.bank_account }">{{ row.bank_account || '未填写' }}</span>
              </template>
            </el-table-column>
          </el-table>
          <template #field-emp_type_label="{ row }">{{ row.emp_type_label }}</template>
        </TableOrCards>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.pay { background: #fff; padding: 16px 18px; border-radius: 10px; border: 1px solid #e2e8ee; }
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
.row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
.sub { margin: 8px 0; font-size: 14px; }
.hint { color: #5c6b75; font-size: 12px; margin: 0 0 0 110px; }
.rate-hint { margin-top: 4px; }
.proc-cell { display: flex; align-items: baseline; gap: 8px; }
.proc-name { font-weight: 500; color: #1f2a33; }
.proc-id { font-size: 12px; color: #98a2a8; }
.rate-num { font-variant-numeric: tabular-nums; font-weight: 600; color: #1f2a33; }
.rate-table :deep(.el-table__header th),
.worker-table :deep(.el-table__header th),
.sheet-table :deep(.el-table__header th) { background: #f6f8fa; color: #4a5a66; font-weight: 600; }
.muted-cell { color: #98a2a8; }
.detail-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 10px; }
.detail-title { display: flex; align-items: center; gap: 10px; }
.detail-meta {
  display: flex; flex-wrap: wrap; gap: 14px;
  color: #5c6b75; font-size: 13px; margin-bottom: 12px;
}
.detail-stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 14px;
}
.stat {
  background: #f6f8fa;
  border: 1px solid #e8eef2;
  border-radius: 8px;
  padding: 10px 12px;
}
.stat-total { background: #eef6f1; border-color: #d5eade; }
.stat-lab { font-size: 12px; color: #6b7a85; margin-bottom: 4px; }
.stat-val { font-variant-numeric: tabular-nums; font-weight: 600; color: #1f2a33; font-size: 15px; }
@media (max-width: 720px) {
  .detail-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
