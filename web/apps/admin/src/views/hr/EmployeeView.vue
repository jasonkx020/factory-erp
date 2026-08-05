<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DEFAULT_EMP_TYPE, EMP_TYPE_OPTIONS, empTypeLabel, hrApi, iamApi } from '@erp/shared'

type Row = Record<string, unknown>

const loading = ref(false)
const list = ref<Row[]>([])
const roles = ref<Row[]>([])
const statusFilter = ref('')
const typeFilter = ref('')
const keyword = ref('')

const dlg = ref(false)
const badgeDlg = ref(false)
const accountDlg = ref(false)
const editingId = ref<number | null>(null)
const current = ref<Row | null>(null)

const form = reactive({
  emp_no: '',
  name: '',
  emp_type: DEFAULT_EMP_TYPE,
  job_title: '',
  mobile: '',
  org_id: 1,
  dept_id: 1,
  workshop_id: 1,
  team_id: 0,
  badge_code: '',
  status: 'active',
})

const badgeForm = reactive({ badge_code: '' })
const accountForm = reactive({ login_name: '', password: 'ChangeMe123', role_ids: [] as number[] })

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
        String(r.badge_code || '').toLowerCase().includes(k),
    )
  }
  return rows
})

const summary = computed(() => {
  const all = list.value
  return {
    total: all.length,
    active: all.filter((r) => r.status === 'active').length,
    left: all.filter((r) => r.status === 'left').length,
    withAccount: all.filter((r) => Number(r.user_id) > 0 || r.has_account).length,
  }
})

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
  Object.assign(form, {
    emp_no: `E${Date.now().toString().slice(-6)}`,
    name: '',
    emp_type: DEFAULT_EMP_TYPE,
    job_title: '',
    mobile: '',
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
  Object.assign(form, {
    emp_no: row.emp_no || '',
    name: row.name || '',
    emp_type: row.emp_type || DEFAULT_EMP_TYPE,
    job_title: row.job_title || '',
    mobile: row.mobile || '',
    org_id: Number(row.org_id) || 1,
    dept_id: Number(row.dept_id) || 1,
    workshop_id: Number(row.workshop_id) || 1,
    team_id: Number(row.team_id) || 0,
    badge_code: row.badge_code || '',
    status: row.status || 'active',
  })
  dlg.value = true
}

async function save() {
  if (!form.name || !form.emp_no) return ElMessage.warning('工号与姓名必填')
  const body = { ...form }
  let res
  if (editingId.value) res = await hrApi.updateEmployee(editingId.value, body)
  else res = await hrApi.createEmployee(body)
  if (res.code !== 1) return ElMessage.error(res.msg || '保存失败')
  ElMessage.success('已保存')
  dlg.value = false
  await load()
}

function openBadge(row: Row) {
  current.value = row
  badgeForm.badge_code = String(row.badge_code || '')
  badgeDlg.value = true
}

async function saveBadge() {
  if (!current.value) return
  const res = await hrApi.setBadge(Number(current.value.id), badgeForm.badge_code)
  if (res.code !== 1) return ElMessage.error(res.msg || '失败')
  ElMessage.success('工牌已更新')
  badgeDlg.value = false
  await load()
}

function openAccount(row: Row) {
  current.value = row
  accountForm.login_name = String(row.mobile || row.emp_no || '')
  accountForm.password = 'ChangeMe123'
  accountForm.role_ids = []
  accountDlg.value = true
}

async function saveAccount() {
  if (!current.value) return
  const res = await hrApi.openAccount(Number(current.value.id), {
    login_name: accountForm.login_name,
    password: accountForm.password,
    role_ids: accountForm.role_ids,
  })
  if (res.code !== 1) return ElMessage.error(res.msg || '开户失败')
  ElMessage.success('账号已开通')
  accountDlg.value = false
  await load()
}

async function deactivate(row: Row) {
  await ElMessageBox.confirm(`将员工 ${row.name} 设为停用？正式离职请走「离职登记」。`, '提示')
  const res = await hrApi.updateEmployee(Number(row.id), { status: 'inactive' })
  if (res.code !== 1) return ElMessage.error(res.msg || '失败')
  ElMessage.success('已停用')
  await load()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="emp">
    <h2 class="title">员工档案</h2>
    <p class="desc">工厂人事主档：工号/类型/工牌/账号；入职走「入职登记」，离职走「离职登记」。</p>

    <div class="stats">
      <div class="stat"><div class="label">在册</div><div class="value">{{ summary.total }}</div></div>
      <div class="stat ok"><div class="label">在职</div><div class="value">{{ summary.active }}</div></div>
      <div class="stat"><div class="label">已离职</div><div class="value">{{ summary.left }}</div></div>
      <div class="stat ok"><div class="label">已开户</div><div class="value">{{ summary.withAccount }}</div></div>
    </div>

    <div class="row">
      <el-button type="primary" @click="openCreate">新建员工</el-button>
      <el-button @click="load">刷新</el-button>
      <el-input v-model="keyword" clearable placeholder="工号/姓名/手机/工牌" style="width:200px" />
      <el-select v-model="statusFilter" clearable placeholder="状态" style="width:120px">
        <el-option label="在职" value="active" />
        <el-option label="离职" value="left" />
        <el-option label="停用" value="inactive" />
      </el-select>
      <el-select v-model="typeFilter" clearable placeholder="类型" style="width:130px">
        <el-option v-for="t in EMP_TYPE_OPTIONS" :key="t.value" :label="t.label" :value="t.value" />
      </el-select>
    </div>

    <el-table :data="filtered" border stripe>
      <el-table-column prop="emp_no" label="工号" width="110" />
      <el-table-column prop="name" label="姓名" width="100" />
      <el-table-column label="类型" width="100">
        <template #default="{ row }">{{ empTypeLabel(row.emp_type) }}</template>
      </el-table-column>
      <el-table-column prop="job_title" label="岗位" width="110" />
      <el-table-column prop="mobile" label="手机" width="120" />
      <el-table-column prop="badge_code" label="工牌" width="110" />
      <el-table-column prop="workshop_id" label="车间" width="70" />
      <el-table-column prop="dept_id" label="部门" width="70" />
      <el-table-column prop="status" label="状态" width="80" />
      <el-table-column label="账号" width="70">
        <template #default="{ row }">{{ Number(row.user_id) > 0 || row.has_account ? '有' : '无' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link @click="openBadge(row)">工牌</el-button>
          <el-button v-if="!(Number(row.user_id) > 0 || row.has_account)" link type="success" @click="openAccount(row)">开户</el-button>
          <el-button v-if="row.status === 'active'" link type="danger" @click="deactivate(row)">停用</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dlg" :title="editingId ? '编辑员工' : '新建员工'" width="560px">
      <el-form label-width="100px">
        <el-form-item label="工号"><el-input v-model="form.emp_no" /></el-form-item>
        <el-form-item label="姓名"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.emp_type" style="width:100%">
            <el-option v-for="t in EMP_TYPE_OPTIONS" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="岗位"><el-input v-model="form.job_title" /></el-form-item>
        <el-form-item label="手机"><el-input v-model="form.mobile" /></el-form-item>
        <el-form-item label="工牌"><el-input v-model="form.badge_code" /></el-form-item>
        <el-form-item label="部门ID"><el-input-number v-model="form.dept_id" :min="0" /></el-form-item>
        <el-form-item label="车间ID"><el-input-number v-model="form.workshop_id" :min="0" /></el-form-item>
        <el-form-item label="班组ID"><el-input-number v-model="form.team_id" :min="0" /></el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="在职" value="active" />
            <el-option label="停用" value="inactive" />
            <el-option label="离职" value="left" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="badgeDlg" title="工牌绑定" width="400px">
      <el-form label-width="80px">
        <el-form-item label="工牌码"><el-input v-model="badgeForm.badge_code" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="badgeDlg = false">取消</el-button>
        <el-button type="primary" @click="saveBadge">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="accountDlg" title="开通登录账号" width="480px">
      <el-form label-width="100px">
        <el-form-item label="登录名"><el-input v-model="accountForm.login_name" /></el-form-item>
        <el-form-item label="初始密码"><el-input v-model="accountForm.password" /></el-form-item>
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
.stats { display: grid; grid-template-columns: repeat(4, minmax(100px, 140px)); gap: 10px; margin-bottom: 12px; }
.stat { background: #f4f7f9; border: 1px solid #e2e8ec; border-radius: 8px; padding: 10px; }
.stat.ok { background: #eef8f4; }
.stat .label { font-size: 12px; color: #5c6b75; }
.stat .value { font-size: 20px; font-weight: 600; }
</style>
