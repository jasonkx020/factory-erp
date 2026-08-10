<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { jsPDF } from 'jspdf'
import QRCode from 'qrcode'
import { DEFAULT_EMP_TYPE, EMP_TYPE_OPTIONS, empTypeLabel, hrApi, iamApi } from '@erp/shared'
import { DeptSelect, TeamSelect, WorkshopSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const empCols: MobileCardColumn[] = [
  { prop: 'name', label: '姓名', primary: true },
  { prop: 'emp_no', label: '工号' },
  { prop: 'login_name', label: '登录账号' },
  { prop: 'emp_type', label: '类型' },
  { prop: 'job_title', label: '岗位' },
  { prop: 'mobile', label: '手机' },
  { prop: 'workshop_id', label: '车间' },
  { prop: 'dept_id', label: '部门' },
  { prop: 'status', label: '状态' },
]

const loading = ref(false)
const exporting = ref(false)
const list = ref<Row[]>([])
const roles = ref<Row[]>([])
const statusFilter = ref('')
const typeFilter = ref('')
const keyword = ref('')
/** badge_code -> dataURL */
const qrMap = ref<Record<string, string>>({})

const dlg = ref(false)
const badgeDlg = ref(false)
const accountDlg = ref(false)
const editingId = ref<number | null>(null)
const current = ref<Row | null>(null)
const badgePreviewQr = ref('')

const form = reactive({
  emp_no: '',
  name: '',
  emp_type: DEFAULT_EMP_TYPE,
  job_title: '',
  mobile: '',
  id_card_no: '',
  org_id: 1,
  dept_id: 1,
  workshop_id: 1,
  team_id: 0,
  badge_code: '',
  status: 'active',
})
const showOrgAdvanced = ref(false)

const accountForm = reactive({ login_name: '', password: '', role_ids: [] as number[] })

const errLabel: Record<string, string> = {
  NAME_REQUIRED: '请填写姓名',
  ID_CARD_REQUIRED: '请填写身份证号',
  ID_CARD_INVALID: '身份证号格式不正确',
  ID_CARD_DUPLICATE: '身份证号已存在',
  MOBILE_REQUIRED: '请填写手机号',
  MOBILE_INVALID: '手机号须为 11 位数字',
  EMP_NO_DUPLICATE: '工号已存在',
  INVALID_EMP_TYPE: '员工类型无效',
}

const filtered = computed(() => {
  let rows = list.value
  if (statusFilter.value) rows = rows.filter((r) => String(r.status) === statusFilter.value)
  if (typeFilter.value) rows = rows.filter((r) => String(r.emp_type) === typeFilter.value)
  if (keyword.value) {
    const k = keyword.value.trim().toLowerCase()
    rows = rows.filter(
      (r) =>
        String(r.emp_no || '').toLowerCase().includes(k) ||
        String(r.name || '').toLowerCase().includes(k) ||
        String(r.mobile || '').includes(k) ||
        String(r.id_card_no || '').toLowerCase().includes(k) ||
        String(r.badge_code || '').toLowerCase().includes(k) ||
        String(r.login_name || '').toLowerCase().includes(k),
    )
  }
  return rows
})

/** 当前筛选中有工牌码的员工（批量导出用） */
const exportable = computed(() =>
  filtered.value.filter((r) => String(r.badge_code || '').trim() !== ''),
)

const summary = computed(() => {
  const all = list.value
  return {
    total: all.length,
    active: all.filter((r) => r.status === 'active').length,
    left: all.filter((r) => r.status === 'left').length,
    withAccount: all.filter((r) => Number(r.user_id) > 0 || r.has_account).length,
  }
})

async function buildQr(code: string, force = false) {
  const c = code.trim()
  if (!c || (qrMap.value[c] && !force)) return
  try {
    const url = await QRCode.toDataURL(c, {
      errorCorrectionLevel: 'M',
      margin: 1,
      width: 320,
      color: { dark: '#000000', light: '#ffffff' },
    })
    qrMap.value = { ...qrMap.value, [c]: url }
  } catch {
    /* ignore */
  }
}

async function refreshBadgePreview(code: string) {
  const c = code.trim()
  badgePreviewQr.value = ''
  if (!c) return
  await buildQr(c)
  badgePreviewQr.value = qrMap.value[c] || ''
}

async function load() {
  loading.value = true
  try {
    const [e, r] = await Promise.all([hrApi.employees(), iamApi.roles()])
    list.value = ((e.data as { list?: Row[] })?.list) || []
    roles.value = ((r.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  showOrgAdvanced.value = false
  Object.assign(form, {
    emp_no: '',
    name: '',
    emp_type: DEFAULT_EMP_TYPE,
    job_title: '',
    mobile: '',
    id_card_no: '',
    org_id: 1,
    dept_id: 1,
    workshop_id: 1,
    team_id: 0,
    badge_code: '',
    status: 'active',
  })
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id)
  showOrgAdvanced.value = false
  Object.assign(form, {
    emp_no: row.emp_no || '',
    name: row.name || '',
    emp_type: row.emp_type || DEFAULT_EMP_TYPE,
    job_title: row.job_title || '',
    mobile: row.mobile || '',
    id_card_no: row.id_card_no || '',
    org_id: Number(row.org_id) || 1,
    dept_id: Number(row.dept_id) || 1,
    workshop_id: Number(row.workshop_id) || 1,
    team_id: Number(row.team_id) || 0,
    badge_code: row.badge_code || '',
    status: row.status || 'active',
  })
  dlg.value = true
}

async function showCreatedCredentials(data: Row) {
  const empNo = String(data.emp_no || '')
  const badge = String(data.badge_code || '')
  const login = String(data.login_name || '')
  const pass = String(data.initial_password || '')
  const hasAcc = data.has_account === true || !!login
  const lines = [
    `工号：${empNo || '-'}`,
    `工牌：${badge || '-'}`,
    hasAcc ? `App 登录名：${login || '-'}` : 'App 账号：开通失败，请稍后点「开户」',
    hasAcc && pass ? `初始密码：${pass}` : '',
    data.account_error ? `开户备注：${data.account_error}` : '',
    '',
    '请将登录名与初始密码告知员工；建议首次登录后修改密码。',
  ].filter((x, i, arr) => x !== '' || (i > 0 && arr[i - 1] !== ''))
  const text = lines.filter(Boolean).join('\n')
  try {
    await ElMessageBox.alert(
      `<div style="white-space:pre-wrap;line-height:1.6;font-size:14px">${text.replace(/</g, '&lt;')}</div>
       <p style="margin:12px 0 0;color:#5c6b75;font-size:12px">明文密码仅本次显示，关闭后无法再从系统查看。</p>`,
      'App 账号信息',
      {
        dangerouslyUseHTMLString: true,
        confirmButtonText: '复制账号信息',
        cancelButtonText: '关闭',
        showCancelButton: true,
        distinguishCancelAndClose: true,
      },
    )
    await navigator.clipboard.writeText(text)
    ElMessage.success('账号信息已复制')
  } catch {
    /* 关闭 */
  }
}

async function save() {
  if (!form.name.trim()) return ElMessage.warning('请填写姓名')
  if (!form.id_card_no.trim()) return ElMessage.warning('请填写身份证号')
  if (!form.mobile.trim()) return ElMessage.warning('请填写手机号')
  let body: Record<string, unknown>
  if (editingId.value) {
    body = {
      name: form.name.trim(),
      id_card_no: form.id_card_no.trim(),
      mobile: form.mobile.trim(),
      emp_type: form.emp_type,
      job_title: form.job_title,
      status: form.status,
      dept_id: form.dept_id,
      workshop_id: form.workshop_id,
      team_id: form.team_id,
      org_id: form.org_id,
    }
  } else {
    // 新建只提交元数据；工号/工牌/组织/App 账号由后端生成
    body = {
      name: form.name.trim(),
      id_card_no: form.id_card_no.trim(),
      mobile: form.mobile.trim(),
      emp_type: form.emp_type,
      job_title: form.job_title,
      open_account: true,
    }
  }
  let res
  if (editingId.value) res = await hrApi.updateEmployee(editingId.value, body)
  else res = await hrApi.createEmployee(body)
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '保存失败')
  const data = (res.data || {}) as Row
  dlg.value = false
  if (!editingId.value) {
    await showCreatedCredentials(data)
  } else {
    ElMessage.success('已保存')
  }
  await load()
}

async function openBadge(row: Row) {
  current.value = row
  badgeDlg.value = true
  await refreshBadgePreview(String(row.badge_code || ''))
}

async function regenerateBadge() {
  if (!current.value) return
  await ElMessageBox.confirm('重新生成后，旧工牌二维码将失效，确认继续？', '重新生成工牌')
  const res = await hrApi.setBadge(Number(current.value.id), '', { regenerate: true })
  if (res.code !== 1) return ElMessage.error(res.msg || '失败')
  const code = String((res.data as Row)?.badge_code || '')
  current.value = { ...current.value, badge_code: code }
  await refreshBadgePreview(code)
  ElMessage.success(`工牌已重新生成：${code}`)
  await load()
}

function openAccount(row: Row) {
  current.value = row
  accountForm.login_name = String(row.mobile || row.emp_no || '')
  accountForm.password = ''
  accountForm.role_ids = []
  accountDlg.value = true
}

async function saveAccount() {
  if (!current.value) return
  const body: Record<string, unknown> = {
    login_name: accountForm.login_name,
    role_ids: accountForm.role_ids,
  }
  const pass = accountForm.password.trim()
  if (pass) body.password = pass
  const res = await hrApi.openAccount(Number(current.value.id), body)
  if (res.code !== 1) return ElMessage.error(res.msg || '开户失败')
  accountDlg.value = false
  const data = (res.data || {}) as Row
  await showCreatedCredentials({
    ...current.value,
    login_name: data.login_name,
    initial_password: data.initial_password,
    has_account: true,
  })
  await load()
}

async function deactivate(row: Row) {
  await ElMessageBox.confirm(`将员工 ${row.name} 设为停用？正式离职请走「离职登记」。`, '提示')
  const res = await hrApi.updateEmployee(Number(row.id), { status: 'inactive' })
  if (res.code !== 1) return ElMessage.error(res.msg || '失败')
  ElMessage.success('已停用')
  await load()
}

/** A4 批量 PDF：4 列 × 5 行 / 页，姓名 + 二维码 + 工牌码 + 工号 */
async function exportBadgePdf() {
  const rows = exportable.value
  const skipped = filtered.value.length - rows.length
  if (!rows.length) {
    return ElMessage.warning(skipped > 0 ? `当前筛选 ${skipped} 人均无工牌码，无法导出` : '无员工可导出')
  }
  if (skipped > 0) {
    ElMessage.info(`已跳过 ${skipped} 名无工牌码员工`)
  }
  exporting.value = true
  try {
    await Promise.all(rows.map((r) => buildQr(String(r.badge_code || '').trim(), true)))
    const doc = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4' })
    const pageW = doc.internal.pageSize.getWidth()
    const pageH = doc.internal.pageSize.getHeight()
    const marginX = 10
    const marginY = 12
    const cols = 4
    const rowCount = 5
    const gapX = 4
    const gapY = 4
    const cellW = (pageW - marginX * 2 - gapX * (cols - 1)) / cols
    const cellH = (pageH - marginY * 2 - gapY * (rowCount - 1) - 8) / rowCount
    const qrSize = Math.min(cellW - 6, cellH - 18)
    const perPage = cols * rowCount
    const today = new Date().toISOString().slice(0, 10)
    const title = `Employee Badge  ${today}  Total ${rows.length}`
    const fname = `employee-badges_${today}_${rows.length}.pdf`

    rows.forEach((emp, i) => {
      const code = String(emp.badge_code || '').trim()
      const name = String(emp.name || '-')
      const empNo = String(emp.emp_no || '')
      const pageIdx = Math.floor(i / perPage)
      const onPage = i % perPage
      if (onPage === 0) {
        if (pageIdx > 0) doc.addPage()
        doc.setFontSize(10)
        doc.setTextColor(60)
        doc.text(title, marginX, 8)
        doc.setFontSize(8)
        doc.text(`Page ${pageIdx + 1}`, pageW - marginX, 8, { align: 'right' })
      }
      const col = onPage % cols
      const row = Math.floor(onPage / cols)
      const x = marginX + col * (cellW + gapX)
      const y = marginY + row * (cellH + gapY)

      doc.setDrawColor(180)
      doc.setLineWidth(0.2)
      doc.roundedRect(x, y, cellW, cellH, 1.5, 1.5, 'S')

      doc.setTextColor(0)
      doc.setFontSize(8)
      doc.text(name, x + cellW / 2, y + 4, { align: 'center', baseline: 'top' })

      const img = qrMap.value[code]
      const imgX = x + (cellW - qrSize) / 2
      const imgY = y + 9
      if (img) {
        doc.addImage(img, 'PNG', imgX, imgY, qrSize, qrSize)
      }

      doc.setFontSize(7)
      const textY = imgY + qrSize + 2.5
      doc.text(code, x + cellW / 2, textY, { align: 'center', baseline: 'top' })
      if (empNo) {
        doc.setTextColor(80)
        doc.setFontSize(6.5)
        doc.text(empNo, x + cellW / 2, textY + 3.5, { align: 'center', baseline: 'top' })
      }
    })

    doc.save(fname)
    ElMessage.success(`已导出 PDF：${fname}（A4 · 每页 4×5 · ${rows.length} 人）`)
  } catch (e) {
    ElMessage.error(`导出失败：${e instanceof Error ? e.message : e}`)
  } finally {
    exporting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading || exporting" class="emp" :element-loading-text="exporting ? '正在生成 PDF…' : ''">
    <h2 class="title">员工档案</h2>
    <p class="desc">快速建档只需姓名、身份证、手机；工号与工牌自动生成。工牌可在操作列查看或批量导出 PDF。</p>

    <div class="stats">
      <div class="stat"><div class="label">在册</div><div class="value">{{ summary.total }}</div></div>
      <div class="stat ok"><div class="label">在职</div><div class="value">{{ summary.active }}</div></div>
      <div class="stat"><div class="label">已离职</div><div class="value">{{ summary.left }}</div></div>
      <div class="stat ok"><div class="label">已开户</div><div class="value">{{ summary.withAccount }}</div></div>
    </div>

    <div class="row">
      <el-button type="primary" @click="openCreate">新建员工</el-button>
      <el-button type="success" :disabled="!exportable.length" @click="exportBadgePdf">批量导出工牌 PDF</el-button>
      <el-button @click="load">刷新</el-button>
      <el-input v-model="keyword" clearable placeholder="工号/姓名/手机/身份证/工牌" style="width:260px" />
      <el-select v-model="statusFilter" clearable placeholder="状态" style="width:120px">
        <el-option label="在职" value="active" />
        <el-option label="离职" value="left" />
        <el-option label="停用" value="inactive" />
      </el-select>
      <el-select v-model="typeFilter" clearable placeholder="类型" style="width:130px">
        <el-option v-for="t in EMP_TYPE_OPTIONS" :key="t.value" :label="t.label" :value="t.value" />
      </el-select>
      <span class="hint">可导出 {{ exportable.length }} / 筛选 {{ filtered.length }}</span>
    </div>

    <TableOrCards :data="filtered" :loading="loading" :columns="empCols">
      <el-table :data="filtered" border stripe>
        <el-table-column prop="emp_no" label="工号" width="110" />
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column prop="login_name" label="登录账号" width="120" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">{{ empTypeLabel(row.emp_type) }}</template>
        </el-table-column>
        <el-table-column prop="job_title" label="岗位" width="110" />
        <el-table-column prop="mobile" label="手机" width="120" />
        <el-table-column prop="workshop_id" label="车间" width="70" />
        <el-table-column prop="dept_id" label="部门" width="70" />
        <el-table-column prop="status" label="状态" width="80" />
        <el-table-column label="账号" width="70">
          <template #default="{ row }">{{ Number(row.user_id) > 0 || row.has_account || row.login_name ? '有' : '无' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link @click="openBadge(row)">工牌</el-button>
            <el-button v-if="!(Number(row.user_id) > 0 || row.has_account)" link type="success" @click="openAccount(row)">开户</el-button>
            <el-button v-if="row.status === 'active'" link type="danger" @click="deactivate(row)">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button link @click="openBadge(row)">工牌</el-button>
        <el-button v-if="!(Number(row.user_id) > 0 || row.has_account)" link type="success" @click="openAccount(row)">开户</el-button>
        <el-button v-if="row.status === 'active'" link type="danger" @click="deactivate(row)">停用</el-button>
      </template>
    </TableOrCards>

    <el-dialog v-model="dlg" :title="editingId ? '编辑员工' : '新建员工'" width="520px">
      <p v-if="!editingId" class="form-tip">
        只需填写姓名、身份证、手机。系统将自动生成工号、工牌，并开通 App 账号（登录名=手机号，初始密码=身份证后 6 位）。
      </p>
      <el-form label-width="100px">
        <el-form-item v-if="editingId" label="工号">
          <span class="badge-readonly">{{ form.emp_no }}</span>
        </el-form-item>
        <el-form-item label="姓名" required>
          <el-input v-model="form.name" placeholder="真实姓名" maxlength="40" />
        </el-form-item>
        <el-form-item label="身份证号" required>
          <el-input v-model="form.id_card_no" placeholder="15 或 18 位" maxlength="18" />
        </el-form-item>
        <el-form-item label="手机" required>
          <el-input v-model="form.mobile" placeholder="11 位手机号" maxlength="11" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.emp_type" style="width:100%">
            <el-option v-for="t in EMP_TYPE_OPTIONS" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="岗位">
          <el-input v-model="form.job_title" placeholder="可选" />
        </el-form-item>
        <el-form-item v-if="editingId && form.badge_code" label="工牌">
          <span class="badge-readonly">{{ form.badge_code }}</span>
          <el-button link type="primary" style="margin-left:8px" @click="openBadge({ id: editingId, ...form })">预览</el-button>
        </el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="在职" value="active" />
            <el-option label="停用" value="inactive" />
            <el-option label="离职" value="left" />
          </el-select>
        </el-form-item>
        <template v-if="editingId">
          <el-divider content-position="left">
            <el-button link type="primary" @click="showOrgAdvanced = !showOrgAdvanced">
              {{ showOrgAdvanced ? '收起归属设置' : '高级：部门 / 车间 / 班组' }}
            </el-button>
          </el-divider>
          <template v-if="showOrgAdvanced">
            <el-form-item label="部门">
              <DeptSelect v-model="form.dept_id" allow-zero zero-label="未设置" style="width:100%" />
            </el-form-item>
            <el-form-item label="车间">
              <WorkshopSelect v-model="form.workshop_id" allow-zero zero-label="未设置" style="width:100%" />
            </el-form-item>
            <el-form-item label="班组">
              <TeamSelect
                v-model="form.team_id"
                :workshop-id="form.workshop_id"
                allow-zero
                zero-label="未设置"
                style="width:100%"
              />
            </el-form-item>
          </template>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">{{ editingId ? '保存' : '建档' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="badgeDlg" title="工牌二维码" width="420px">
      <div v-if="badgePreviewQr || current?.badge_code" class="preview-box">
        <img v-if="badgePreviewQr" :src="badgePreviewQr" alt="badge qr" class="qr-lg" />
        <div class="preview-meta">{{ current?.badge_code }}</div>
        <div class="preview-sub">{{ current?.name }} · {{ current?.emp_no }}</div>
        <p class="hint center">工牌码由系统自动生成，过站扫此码即可识别员工</p>
      </div>
      <p v-else class="muted center">暂无工牌，可点击重新生成</p>
      <template #footer>
        <el-button @click="badgeDlg = false">关闭</el-button>
        <el-button type="warning" @click="regenerateBadge">重新生成</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="accountDlg" title="开通登录账号" width="480px">
      <el-form label-width="100px">
        <el-form-item label="登录名"><el-input v-model="accountForm.login_name" /></el-form-item>
        <el-form-item label="初始密码">
          <el-input v-model="accountForm.password" placeholder="留空则用身份证后 6 位" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="accountForm.role_ids" multiple filterable style="width:100%">
            <el-option v-for="r in roles" :key="String(r.id)" :label="`${r.code || ''} ${r.name || ''}`" :value="Number(r.id)" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDlg = false">取消</el-button>
        <el-button type="primary" @click="saveAccount">开通</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.emp { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; align-items: center; }
.hint { font-size: 12px; color: #5c6b75; }
.stats { display: grid; grid-template-columns: repeat(4, minmax(100px, 140px)); gap: 10px; margin-bottom: 12px; }
.stat { background: #f4f7f9; border: 1px solid #e2e8ec; border-radius: 8px; padding: 10px; }
.stat.ok { background: #eef8f4; }
.stat .label { font-size: 12px; color: #5c6b75; }
.stat .value { font-size: 20px; font-weight: 600; }
.muted { color: #9aa7b0; font-size: 12px; }
.center { text-align: center; }
.preview-box { text-align: center; padding: 8px 0 4px; }
.qr-lg { width: 220px; height: 220px; display: block; margin: 0 auto 10px; border: 1px solid #e2e8ec; border-radius: 8px; }
.preview-meta { font-size: 14px; font-weight: 600; letter-spacing: 0.02em; margin-top: 4px; word-break: break-all; }
.preview-sub { font-size: 12px; color: #5c6b75; margin-top: 4px; }
.badge-readonly { font-size: 13px; font-weight: 600; }
.form-tip { margin: 0 0 12px; font-size: 13px; color: #5c6b75; background: #f4f7f9; padding: 8px 10px; border-radius: 6px; }
</style>
