<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DEFAULT_EMP_TYPE, EMP_TYPE_OPTIONS, empTypeLabel, hrApi, iamApi } from '@erp/shared'
import { TeamSelect, JobTitleSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const onboardCols: MobileCardColumn[] = [
  { prop: 'name', label: '姓名', primary: true },
  { prop: 'id', label: '单号' },
  { prop: 'onboard_date', label: '入职日' },
  { prop: 'emp_no', label: '工号' },
  { prop: 'emp_type', label: '类型' },
  { prop: 'job_title_name', label: '岗位' },
  { prop: 'mobile', label: '手机' },
  { prop: 'id_card_no', label: '身份证号' },
  { prop: 'status', label: '状态' },
  { prop: 'remark', label: '备注' },
]

const loading = ref(false)
const list = ref<Row[]>([])
const summary = reactive({ draft: 0, confirmed: 0, cancelled: 0 })
const statusFilter = ref('')
const empTypeFilter = ref('')
const roles = ref<Row[]>([])
const deptMap = ref<Record<number, string>>({})
const deptTypeMap = ref<Record<number, string>>({})

const dialog = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({
  emp_no: '',
  name: '',
  emp_type: DEFAULT_EMP_TYPE,
  org_id: 1,
  dept_id: 1,
  dept_ids: [] as number[],
  primary_dept_id: 1,
  team_id: 0,
  job_title_id: 0,
  mobile: '',
  badge_code: '',
  id_card_no: '',
  onboard_date: '',
  need_account: true,
  login_name: '',
  remark: '',
  role_ids: [] as number[],
  bank_account: '',
  tax_no: '',
})

const statusLabel: Record<string, string> = {
  draft: '草稿',
  confirmed: '已入职',
  cancelled: '已取消',
}
const deptOptions = computed(() =>
  Object.entries(deptMap.value).map(([id, name]) => ({
    value: Number(id),
    label: deptTypeMap.value[Number(id)] === 'workshop' ? `${name}（车间）` : name,
  })),
)

const formWorkshopDeptId = computed(() => {
  for (const id of form.dept_ids) {
    if (deptTypeMap.value[id] === 'workshop') return id
  }
  return 0
})

const isEdit = computed(() => editingId.value != null)

const visibleList = computed(() => {
  if (!empTypeFilter.value) return list.value
  return list.value.filter((r) => String(r.emp_type || '') === empTypeFilter.value)
})

function today() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function resetForm() {
  Object.assign(form, {
    emp_no: `E${Date.now().toString().slice(-8)}`,
    name: '',
    emp_type: DEFAULT_EMP_TYPE,
    org_id: 1,
    dept_id: 1,
    dept_ids: [1],
    primary_dept_id: 1,
    team_id: 0,
    job_title_id: 0,
    mobile: '',
    badge_code: '',
    id_card_no: '',
    onboard_date: today(),
    need_account: true,
    login_name: '',
    remark: '',
    role_ids: [] as number[],
    bank_account: '',
    tax_no: '',
  })
}

async function load() {
  loading.value = true
  try {
    const qs = statusFilter.value ? `status=${encodeURIComponent(statusFilter.value)}` : ''
    const [ob, r, d] = await Promise.all([hrApi.onboards(qs), iamApi.roles(), hrApi.departments()])
    if (ob.code !== 1) return ElMessage.error(ob.msg)
    const depts = ((d.data as { list?: Row[] })?.list) || []
    const dm: Record<number, string> = {}
    const tm: Record<number, string> = {}
    for (const x of depts) {
      const id = Number(x.id) || 0
      if (id > 0) {
        dm[id] = String(x.path || x.name || x.code || id)
        tm[id] = String(x.dept_type || 'normal')
      }
    }
    deptMap.value = dm
    deptTypeMap.value = tm
    const data = ob.data as { list?: Row[]; summary?: Row }
    list.value = data?.list || []
    const s = data?.summary || {}
    summary.draft = Number(s.draft) || 0
    summary.confirmed = Number(s.confirmed) || 0
    summary.cancelled = Number(s.cancelled) || 0
    roles.value = ((r.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  resetForm()
  dialog.value = true
}

async function openEdit(row: Row) {
  if (row.status !== 'draft') return ElMessage.warning('仅草稿可编辑')
  editingId.value = Number(row.id)
  const res = await hrApi.getOnboard(editingId.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = res.data as Row
  const emp = (d.employee as Row) || {}
  const deptIds = ((emp.dept_ids as number[]) || []).map(Number)
  const primary = Number(emp.dept_id) || deptIds[0] || 1
  Object.assign(form, {
    emp_no: String(emp.emp_no || d.emp_no || ''),
    name: String(emp.name || d.name || ''),
    emp_type: String(emp.emp_type || DEFAULT_EMP_TYPE),
    org_id: Number(emp.org_id) || 1,
    dept_id: primary,
    dept_ids: deptIds.length ? deptIds : primary ? [primary] : [1],
    primary_dept_id: primary,
    team_id: Number(emp.team_id) || 0,
    job_title_id: Number(emp.job_title_id) || 0,
    mobile: String(emp.mobile || ''),
    badge_code: String(emp.badge_code || ''),
    id_card_no: String(emp.id_card_no || d.id_card_no || ''),
    onboard_date: String(d.onboard_date || today()),
    need_account: d.need_account !== false,
    login_name: String(d.login_name || ''),
    remark: String(d.remark || ''),
    role_ids: ((d.role_ids as number[]) || []).map(Number),
    bank_account: String(emp.bank_account || d.bank_account || ''),
    tax_no: String(emp.tax_no || d.tax_no || ''),
  })
  dialog.value = true
}

async function save() {
  if (!form.emp_no || !form.name) return ElMessage.warning('请填写工号与姓名')
  const body: Record<string, unknown> = {
    ...form,
    dept_ids: form.dept_ids,
    primary_dept_id: form.primary_dept_id || form.dept_ids[0] || form.dept_id,
    dept_id: form.primary_dept_id || form.dept_ids[0] || form.dept_id,
  }
  delete body.badge_code // 工牌由建档时后端自动生成
  if (!form.login_name) body.login_name = form.emp_no
  let res
  if (isEdit.value && editingId.value) {
    res = await hrApi.updateOnboard(editingId.value, body)
  } else {
    res = await hrApi.createOnboard(body)
  }
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(isEdit.value ? '草稿已更新' : '入职单已创建（草稿）')
  dialog.value = false
  await load()
}

watch(
  () => form.dept_ids.slice(),
  (ids) => {
    if (ids.length === 1) {
      form.primary_dept_id = ids[0]
      form.dept_id = ids[0]
    } else if (ids.length > 1 && !ids.includes(form.primary_dept_id)) {
      form.primary_dept_id = ids[0]
      form.dept_id = ids[0]
    } else if (ids.length === 0) {
      form.primary_dept_id = 0
      form.dept_id = 0
    }
  },
)

async function confirm(row: Row) {
  await ElMessageBox.confirm(
    row.need_account === false
      ? '确认入职？将激活员工档案（不开户）。'
      : '确认入职？将激活员工并开户赋权（初始密码 ChangeMe123）。',
    '确认入职',
  )
  const res = await hrApi.confirmOnboard(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  const data = res.data as Row
  ElMessage.success(data?.has_account || data?.user_id ? '已入职并开户' : '已入职')
  await load()
}

async function cancel(row: Row) {
  await ElMessageBox.confirm('取消该入职草稿？', '提示')
  const res = await hrApi.cancelOnboard(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已取消')
  await load()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="onboard">
    <h2 class="title">入职登记</h2>
    <p class="desc">
      闭环：员工建档 → 入职单（日期/岗位/角色）→ 确认开户赋权。确认后员工生效，并可登录系统。
    </p>

    <div class="stats">
      <div class="stat" @click="statusFilter = 'draft'; load()">
        <div class="label">草稿待确认</div>
        <div class="value">{{ summary.draft }}</div>
      </div>
      <div class="stat ok" @click="statusFilter = 'confirmed'; load()">
        <div class="label">已入职</div>
        <div class="value">{{ summary.confirmed }}</div>
      </div>
      <div class="stat muted" @click="statusFilter = 'cancelled'; load()">
        <div class="label">已取消</div>
        <div class="value">{{ summary.cancelled }}</div>
      </div>
      <div class="stat" @click="statusFilter = ''; load()">
        <div class="label">全部</div>
        <div class="value">{{ list.length }}</div>
      </div>
    </div>

    <div class="row">
      <el-radio-group v-model="statusFilter" size="small" @change="load">
        <el-radio-button label="">全部</el-radio-button>
        <el-radio-button label="draft">草稿</el-radio-button>
        <el-radio-button label="confirmed">已入职</el-radio-button>
        <el-radio-button label="cancelled">已取消</el-radio-button>
      </el-radio-group>
      <el-select v-model="empTypeFilter" size="small" clearable placeholder="全部类型" style="width:130px">
        <el-option v-for="t in EMP_TYPE_OPTIONS" :key="t.value" :label="t.label" :value="t.value" />
      </el-select>
      <el-button type="primary" @click="openCreate">新建入职</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <TableOrCards :data="visibleList" :loading="loading" :columns="onboardCols">
      <el-table :data="visibleList" border stripe>
        <el-table-column prop="id" label="单号" width="70" />
        <el-table-column prop="onboard_date" label="入职日" width="110" />
        <el-table-column prop="emp_no" label="工号" width="120" />
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.emp_type === 'temp' ? 'warning' : 'info'">
              {{ empTypeLabel(row.emp_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="job_title_name" label="岗位" width="100" />
        <el-table-column prop="mobile" label="手机" width="120" />
        <el-table-column prop="id_card_no" label="身份证号" width="160" show-overflow-tooltip />
        <el-table-column label="开户" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.has_account ? 'success' : row.need_account ? 'warning' : 'info'">
              {{ row.has_account ? '已开' : row.need_account ? '待开' : '不开' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.status === 'confirmed' ? 'success' : row.status === 'cancelled' ? 'info' : 'warning'"
            >{{ statusLabel[String(row.status)] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="120" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'draft'" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="row.status === 'draft'" link type="success" @click="confirm(row)">确认入职</el-button>
            <el-button v-if="row.status === 'draft'" link type="danger" @click="cancel(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #actions="{ row }">
        <el-button v-if="row.status === 'draft'" link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button v-if="row.status === 'draft'" link type="success" @click="confirm(row)">确认入职</el-button>
        <el-button v-if="row.status === 'draft'" link type="danger" @click="cancel(row)">取消</el-button>
      </template>
    </TableOrCards>

    <el-dialog v-model="dialog" :title="isEdit ? '编辑入职草稿' : '新建入职登记'" width="720px" destroy-on-close>
      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom:12px"
        title="一次完成：建档信息 + 入职日期 + 角色赋权；确认后员工生效并可开户登录。"
      />
      <el-form label-width="100px">
        <el-row :gutter="12">
          <el-col :span="12" :xs="24">
            <el-form-item label="工号" required>
              <el-input v-model="form.emp_no" :disabled="isEdit" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="姓名" required>
              <el-input v-model="form.name" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="员工类型">
              <el-select v-model="form.emp_type" style="width:100%">
                <el-option v-for="t in EMP_TYPE_OPTIONS" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="入职日期">
              <el-date-picker v-model="form.onboard_date" type="date" value-format="YYYY-MM-DD" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="岗位">
              <JobTitleSelect v-model="form.job_title_id" :emp-type="form.emp_type" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="手机">
              <el-input v-model="form.mobile" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="身份证号">
              <el-input v-model="form.id_card_no" placeholder="可手填；App 支持 OCR" maxlength="18" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="银行卡">
              <el-input v-model="form.bank_account" placeholder="工资卡号，与财务同源" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="税号">
              <el-input v-model="form.tax_no" placeholder="可选" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="24" :xs="24">
            <el-form-item label="所属部门">
              <el-select
                v-model="form.dept_ids"
                multiple
                filterable
                collapse-tags
                collapse-tags-tooltip
                placeholder="可选择多个部门"
                style="width:100%"
              >
                <el-option v-for="opt in deptOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
              </el-select>
              <p class="hint">可同时归属多个部门，权限为全部所属部门有效权限的并集。</p>
            </el-form-item>
          </el-col>
          <el-col v-if="form.dept_ids.length > 1" :span="12" :xs="24">
            <el-form-item label="主部门">
              <el-select v-model="form.primary_dept_id" placeholder="报表/调动默认部门" style="width:100%">
                <el-option
                  v-for="id in form.dept_ids"
                  :key="id"
                  :label="deptMap[id] || `#${id}`"
                  :value="id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8" :xs="24">
            <el-form-item label="班组">
              <TeamSelect
                v-model="form.team_id"
                :dept-id="formWorkshopDeptId"
                allow-zero
                zero-label="未设置"
                style="width:100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="工牌码">
              <span v-if="form.badge_code" class="badge-ro">{{ form.badge_code }}（自动生成）</span>
              <span v-else class="hint">确认入职建档时系统自动生成，无需填写</span>
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="登录名">
              <el-input v-model="form.login_name" placeholder="默认用工号" />
            </el-form-item>
          </el-col>
          <el-col :span="24" :xs="24">
            <el-form-item label="开户赋权">
              <el-switch v-model="form.need_account" active-text="确认时开户" inactive-text="仅建档不开户" />
            </el-form-item>
          </el-col>
          <el-col :span="24" :xs="24">
            <el-form-item label="赋予角色">
              <el-checkbox-group v-model="form.role_ids">
                <el-checkbox v-for="r in roles" :key="String(r.id)" :label="Number(r.id)">
                  {{ r.name }} ({{ r.code }})
                </el-checkbox>
              </el-checkbox-group>
              <p class="hint">不选则按员工类型使用入职角色模板（计件/临时→piece / 固定→fixed / 职能→hr）。</p>
            </el-form-item>
          </el-col>
          <el-col :span="24" :xs="24">
            <el-form-item label="备注">
              <el-input v-model="form.remark" type="textarea" :rows="2" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="save">保存草稿</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.onboard { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 12px; }
.stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-bottom: 14px; }
.stat {
  background: #f4f7f9; border: 1px solid #e2e8ec; border-radius: 8px; padding: 12px;
  cursor: pointer;
}
.stat:hover { border-color: #3d8b7a; }
.stat.ok { background: #eef8f4; }
.stat.muted { background: #f5f5f5; }
.stat .label { font-size: 12px; color: #5c6b75; }
.stat .value { font-size: 22px; font-weight: 600; margin-top: 4px; }
.hint { margin: 4px 0 0; font-size: 12px; color: #8a9aa3; }
.badge-ro { font-size: 13px; font-weight: 600; }
@media (max-width: 800px) {
  .stats { grid-template-columns: repeat(2, 1fr); }
}
</style>
