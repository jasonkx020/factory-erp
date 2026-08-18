<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { iamApi, ticketApi } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const ticketCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'category_name', label: '分类' },
  { prop: 'title', label: '标题' },
  { prop: 'applicant_name', label: '申请人' },
  { prop: 'assignee_name', label: '当前处理人' },
  { prop: 'status', label: '状态', hideOnCard: true },
  { prop: 'created_at', label: '创建时间' },
]

const categoryCols: MobileCardColumn[] = [
  { prop: 'name', label: '业务', primary: true },
  { prop: 'remark', label: '做什么用' },
  { prop: 'pool_count', label: '可指派人数' },
]

type FieldDef = {
  key: string
  label: string
  type: string
  required?: boolean
  options?: string[]
  unit?: string
}

const tab = ref<'tickets' | 'categories'>('categories')
const loading = ref(false)
const tickets = ref<Row[]>([])
const categories = ref<Row[]>([])
const roles = ref<Row[]>([])
const users = ref<Row[]>([])
const scope = ref('mine_assignee')
const statusFilter = ref('')

const detail = ref<Row | null>(null)
const detailVisible = ref(false)
const pool = ref<Row[]>([])
const createOpen = ref<string[]>([])

const createForm = reactive({
  category_id: 0,
  title: '',
  next_assignee_user_id: null as number | null,
  payload: {} as Record<string, unknown>,
})
const createSchema = ref<FieldDef[]>([])

const actionForm = reactive({
  action: 'approve',
  comment: '',
  next_assignee_user_id: null as number | null,
})

const handlerEdit = reactive({
  category_id: 0,
  category_name: '',
  handlers: [] as { handler_type: string; handler_ref: number }[],
})
const handlerDialog = ref(false)
const handlerPoolPreview = ref<Row[]>([])

const typeDialog = ref(false)
const typeForm = reactive({
  id: 0,
  code: '',
  name: '',
  remark: '',
  biz_hint: '',
  enabled: true,
  form_schema: [] as FieldDef[],
})
const isEditType = computed(() => typeForm.id > 0)
const codeTouched = ref(false)
const typeSaving = ref(false)

const BUILTIN_TYPE_CODES = new Set([
  'farm_inbound',
  'stock_inbound',
  'prod_process',
  'sales_outbound',
  'tool_issue',
  'tool_return',
  'piece_issue',
])
const isBuiltinType = computed(() => BUILTIN_TYPE_CODES.has(typeForm.code))

const activeUsers = computed(() =>
  users.value.filter((u) => {
    if (u.is_deleted === 1 || u.is_deleted === true) return false
    const st = String(u.status || 'active').toLowerCase()
    return st === '' || st === 'active'
  }),
)

function handlerUserOptions(currentRef: number): Row[] {
  const list = [...activeUsers.value]
  const ids = new Set(list.map((u) => Number(u.id)))
  if (currentRef > 0 && !ids.has(currentRef)) {
    const stale = users.value.find((u) => Number(u.id) === currentRef)
    if (stale) list.unshift(stale)
  }
  return list
}

function userOptionLabel(u: Row): string {
  const name = String(u.name || u.login_name || u.id)
  const st = String(u.status || 'active').toLowerCase()
  const gone = u.is_deleted === 1 || u.is_deleted === true || (st !== '' && st !== 'active')
  return gone ? `${name}（已停用/离职）` : name
}

const FIELD_TYPES = [
  { value: 'text', label: '文本', hint: '短内容，如姓名、车牌' },
  { value: 'number', label: '数字', hint: '重量、金额、数量' },
  { value: 'date', label: '日期', hint: '业务发生日期' },
  { value: 'select', label: '下拉', hint: '从固定选项里选' },
  { value: 'textarea', label: '多行', hint: '备注、情况说明' },
]

const CODE_EXAMPLES = [
  { code: 'equipment_repair', name: '设备报修' },
  { code: 'quality_abnormal', name: '质量异常' },
  { code: 'other_collab', name: '其他协作' },
]

function suggestCode(name: string): string {
  const ascii = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 48)
  if (ascii && /^[a-z]/.test(ascii)) return ascii
  return ''
}

function suggestFieldKey(label: string, index: number): string {
  const ascii = label
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 32)
  if (ascii && /^[a-z]/.test(ascii)) return ascii
  return `field_${index + 1}`
}

function onTypeNameInput(v: string) {
  typeForm.name = v
  if (!isEditType.value && !codeTouched.value) {
    typeForm.code = suggestCode(v)
  }
}

function applyCodeExample(ex: { code: string; name: string }) {
  if (isEditType.value) return
  typeForm.name = ex.name
  typeForm.code = ex.code
  codeTouched.value = true
  if (!typeForm.remark) typeForm.remark = `${ex.name}协作单，派给对应岗位处理`
}

function fieldTypeHint(type: string): string {
  return FIELD_TYPES.find((t) => t.value === type)?.hint || ''
}

const statusLabel: Record<string, string> = {
  open: '待处理',
  in_progress: '处理中',
  done: '已办结',
  rejected: '已驳回',
  cancelled: '已取消',
}

const detailIsTool = computed(() => {
  const code = String(detail.value?.category_code || '')
  const name = String(detail.value?.category_name || '')
  return code.includes('tool') || name.includes('工具')
})

function schemaOf(cat: Row | undefined): FieldDef[] {
  if (!cat) return []
  const arr = cat.form_schema
  if (Array.isArray(arr)) return arr as FieldDef[]
  return []
}

function handlerLabels(row: Row): string[] {
  const raw = row.handler_labels
  if (Array.isArray(raw)) return raw.map((x) => String(x)).filter(Boolean)
  return []
}

function poolCount(row: Row): number {
  return Number(row.pool_count || 0)
}

function poolPreviewNames(row: Row): string {
  const list = Array.isArray(row.pool) ? (row.pool as Row[]) : []
  return list
    .map((p) => String(p.name || p.login_name || ''))
    .filter(Boolean)
    .join('、')
}

function initPayload(schema: FieldDef[]) {
  const p: Record<string, unknown> = {}
  for (const f of schema) {
    if (f.type === 'number') p[f.key] = null
    else if (f.type === 'date') p[f.key] = new Date().toISOString().slice(0, 10)
    else if (f.type === 'select' && f.options?.length) p[f.key] = f.options[0]
    else p[f.key] = ''
  }
  return p
}

async function loadMeta() {
  const [c, r, u] = await Promise.all([ticketApi.categories(), iamApi.roles(), iamApi.users()])
  categories.value = ((c.data as { list?: Row[] })?.list) || []
  roles.value = ((r.data as { list?: Row[] })?.list) || []
  users.value = ((u.data as { list?: Row[] })?.list) || []
  if (!createForm.category_id && categories.value.length) {
    createForm.category_id = Number(categories.value[0].id)
  }
  await onCategoryChange()
}

async function onCategoryChange() {
  const cat = categories.value.find((x) => Number(x.id) === createForm.category_id)
  createSchema.value = schemaOf(cat)
  createForm.payload = initPayload(createSchema.value)
  createForm.title = ''
  await loadPool(createForm.category_id)
}

async function loadPool(catId: number) {
  if (!catId) {
    pool.value = []
    return
  }
  const res = await ticketApi.handlerPool(`category_id=${catId}`)
  pool.value = ((res.data as { pool?: Row[] })?.pool) || []
  if (pool.value.length) {
    createForm.next_assignee_user_id = Number(pool.value[0].user_id)
  } else {
    createForm.next_assignee_user_id = null
  }
}

async function loadTickets() {
  loading.value = true
  try {
    const qs = new URLSearchParams()
    if (scope.value) qs.set('scope', scope.value)
    if (statusFilter.value) qs.set('status', statusFilter.value)
    const res = await ticketApi.tickets(qs.toString())
    tickets.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await loadMeta()
  await loadTickets()
}

async function createTicket() {
  if (!createForm.category_id || !createForm.next_assignee_user_id) {
    return ElMessage.warning(pool.value.length ? '请选择分类并指定下一手处理人' : '请先到「业务类型与处理人」配置处理人池')
  }
  const res = await ticketApi.createTicket({
    category_id: createForm.category_id,
    title: createForm.title || undefined,
    next_assignee_user_id: createForm.next_assignee_user_id,
    payload: createForm.payload,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('工单已创建')
  createForm.title = ''
  createForm.payload = initPayload(createSchema.value)
  await loadTickets()
}

async function openDetail(row: Row) {
  const res = await ticketApi.getTicket(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = res.data as Row
  pool.value = ((res.data as { pool?: Row[] })?.pool) || []
  actionForm.action = 'approve'
  actionForm.comment = ''
  actionForm.next_assignee_user_id = null
  detailVisible.value = true
}

const detailSchema = computed(() => ((detail.value?.form_schema as FieldDef[]) || []) as FieldDef[])
const detailPayload = computed(() => {
  const p = detail.value?.payload
  if (p && typeof p === 'object') return p as Record<string, unknown>
  return {}
})

async function doAction() {
  if (!detail.value) return
  const body: Record<string, unknown> = {
    action: actionForm.action,
    comment: actionForm.comment,
  }
  if (actionForm.next_assignee_user_id) body.next_assignee_user_id = actionForm.next_assignee_user_id
  const res = await ticketApi.actionTicket(Number(detail.value.id), body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已处理')
  detailVisible.value = false
  await loadTickets()
}

async function doAssign() {
  if (!detail.value || !actionForm.next_assignee_user_id) return ElMessage.warning('请选择下一手')
  const res = await ticketApi.assignTicket(Number(detail.value.id), {
    next_assignee_user_id: actionForm.next_assignee_user_id,
    comment: actionForm.comment,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已转交')
  detailVisible.value = false
  await loadTickets()
}

async function openHandlers(row: Row) {
  handlerEdit.category_id = Number(row.id)
  handlerEdit.category_name = String(row.name || '')
  const res = await ticketApi.getHandlers(handlerEdit.category_id)
  const list = ((res.data as { handlers?: Row[] })?.handlers) || []
  handlerPoolPreview.value = ((res.data as { pool?: Row[] })?.pool) || []
  handlerEdit.handlers = list.map((h) => ({
    handler_type: String(h.handler_type || 'role'),
    handler_ref: Number(h.handler_ref),
  }))
  if (!handlerEdit.handlers.length) handlerEdit.handlers.push({ handler_type: 'role', handler_ref: 0 })
  handlerDialog.value = true
}

function addHandlerRow() {
  handlerEdit.handlers.push({ handler_type: 'role', handler_ref: 0 })
}

const selectedHandlerLabels = computed(() => {
  return handlerEdit.handlers
    .filter((h) => h.handler_ref > 0)
    .map((h) => {
      if (h.handler_type === 'role') {
        const r = roles.value.find((x) => Number(x.id) === h.handler_ref)
        return r ? String(r.name || r.code) : ''
      }
      const u = users.value.find((x) => Number(x.id) === h.handler_ref)
      return u ? String(u.name || u.login_name || '') : ''
    })
    .filter(Boolean)
})

async function saveHandlers() {
  const handlers = handlerEdit.handlers.filter((h) => h.handler_ref > 0)
  if (!handlers.length) {
    return ElMessage.warning('至少选一个角色或指定用户')
  }
  const res = await ticketApi.putHandlers(handlerEdit.category_id, handlers)
  if (res.code !== 1) return ElMessage.error(res.msg)
  handlerPoolPreview.value = ((res.data as { pool?: Row[] })?.pool) || []
  ElMessage.success('处理人池已保存')
  handlerDialog.value = false
  await loadMeta()
}

function openNewType() {
  codeTouched.value = false
  Object.assign(typeForm, {
    id: 0,
    code: '',
    name: '',
    remark: '',
    biz_hint: '',
    enabled: true,
    form_schema: [
      { key: 'biz_date', label: '日期', type: 'date', required: true, options: [] },
      { key: 'remark', label: '情况说明', type: 'textarea', required: false, options: [] },
    ],
  })
  typeDialog.value = true
}

function openEditType(row: Row) {
  codeTouched.value = true
  Object.assign(typeForm, {
    id: Number(row.id),
    code: String(row.code || ''),
    name: String(row.name || ''),
    remark: String(row.remark || ''),
    biz_hint: String(row.biz_hint || ''),
    enabled: row.enabled !== false,
    form_schema: schemaOf(row).map((f) => ({ ...f, options: [...(f.options || [])] })),
  })
  if (!typeForm.form_schema.length) {
    typeForm.form_schema.push({ key: 'remark', label: '备注', type: 'textarea', required: false, options: [] })
  }
  typeDialog.value = true
}

function uniqueFieldKey(base: string): string {
  const used = new Set(typeForm.form_schema.map((f) => f.key))
  if (!used.has(base)) return base
  let n = 2
  while (used.has(`${base}_${n}`)) n += 1
  return `${base}_${n}`
}

function addField(preset?: Partial<FieldDef>) {
  const i = typeForm.form_schema.length
  const label = preset?.label || '新字段'
  const rawKey = preset?.key || suggestFieldKey(label, i)
  typeForm.form_schema.push({
    key: uniqueFieldKey(rawKey),
    label,
    type: preset?.type || 'text',
    required: preset?.required ?? false,
    options: [...(preset?.options || [])],
    unit: preset?.unit || '',
  })
}

function onFieldLabelInput(f: FieldDef, i: number, v: string) {
  f.label = v
  if (!f.key || /^field_\d+$/.test(f.key)) {
    f.key = suggestFieldKey(v, i)
  }
}

function typeFormError(): string {
  const name = typeForm.name.trim()
  const code = typeForm.code.trim()
  if (!name) return '请填写名称，员工建单时会看到这个名字'
  if (!code) return '请填写编码，例如 equipment_repair'
  if (!/^[a-z][a-z0-9_]{1,47}$/.test(code)) {
    return '编码须为小写字母开头，仅含字母、数字、下划线，最长 48 位'
  }
  if (!isEditType.value && categories.value.some((c) => String(c.code) === code)) {
    return `编码「${code}」已存在，请换一个`
  }
  if (!typeForm.form_schema.length) return '至少添加一个填报字段'
  const keys = new Set<string>()
  for (let i = 0; i < typeForm.form_schema.length; i++) {
    const f = typeForm.form_schema[i]
    const lab = (f.label || '').trim()
    const key = (f.key || '').trim()
    if (!lab) return `第 ${i + 1} 个字段请填写「显示名称」`
    if (!key) return `第 ${i + 1} 个字段请填写「字段编码」`
    if (!/^[a-z][a-z0-9_]{0,31}$/.test(key)) {
      return `「${lab}」的字段编码须为小写字母开头，仅含字母、数字、下划线`
    }
    if (keys.has(key)) return `字段编码「${key}」重复了`
    keys.add(key)
    if (f.type === 'select' && !(f.options || []).filter(Boolean).length) {
      return `「${lab}」是下拉，请填写选项，用逗号分隔`
    }
  }
  return ''
}

async function saveType() {
  const err = typeFormError()
  if (err) return ElMessage.warning(err)
  typeSaving.value = true
  try {
    const body = {
      code: typeForm.code.trim(),
      name: typeForm.name.trim(),
      remark: typeForm.remark.trim(),
      biz_hint: typeForm.biz_hint.trim(),
      enabled: typeForm.enabled,
      form_schema: typeForm.form_schema.map((f) => ({
        ...f,
        key: f.key.trim(),
        label: f.label.trim(),
        options: f.type === 'select' ? (f.options || []).filter(Boolean) : undefined,
        unit: f.unit?.trim() || undefined,
      })),
    }
    const res = isEditType.value
      ? await ticketApi.updateCategory(typeForm.id, body)
      : await ticketApi.createCategory(body)
    if (res.code !== 1) return ElMessage.error(res.msg)
    ElMessage.success(isEditType.value ? '类型已更新，记得配置处理人' : '类型已创建，请接着配置处理人')
    typeDialog.value = false
    await loadMeta()
  } finally {
    typeSaving.value = false
  }
}

const logs = computed(() => ((detail.value?.logs as Row[]) || []) as Row[])

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>工单中心</h2>

    <el-alert type="info" show-icon :closable="false" class="howto">
      <template #title>怎么配置（本页）</template>
      <ol class="howto-list">
        <li>打开类型右侧「配置处理人」（内置类型也可以改，不是锁死的名单）。</li>
        <li>添加角色：仓管、质检（可选采购）。勾选角色即可带出该角色全部<strong>在职</strong>账号。</li>
        <li>保存后看「可指派人数」：大于 0 且不能只剩提交人自己，员工 App 才能指派成功。</li>
        <li>人离职了：先停用其账号，可指派名单会自动去掉；给新人挂上同一角色，或在「配置处理人」改成指定用户。</li>
      </ol>
      <p class="howto-sub">怎么使用</p>
      <ol class="howto-list">
        <li>员工在 App 做「过磅入厂」，预览步选择下一部门和处理人（必须是上面池里的人）。</li>
        <li>提交成功后，单据出现在本页「工单跟踪」（待我处理 / 我发起的）。</li>
        <li>仓管在 App 或本页详情里接单处理。不必在工单中心手建过磅单。</li>
      </ol>
    </el-alert>

    <el-tabs v-model="tab">
      <el-tab-pane label="业务类型与处理人" name="categories">
        <div class="toolbar mb">
          <el-button size="small" @click="openNewType">新建工单类型</el-button>
        </div>
        <TableOrCards :data="categories" :loading="loading" :columns="categoryCols" empty-text="暂无业务类型，请点击「新建工单类型」或刷新本页以自动补齐">
          <el-table :data="categories" border stripe size="small" empty-text="暂无业务类型">
            <el-table-column prop="name" label="业务" width="140" />
            <el-table-column prop="remark" label="做什么用" min-width="180" />
            <el-table-column label="已配处理角色" min-width="160">
              <template #default="{ row }">
                <span v-if="!handlerLabels(row).length" class="muted">未配</span>
                <el-tag v-for="lab in handlerLabels(row)" :key="lab" size="small" class="role-tag">{{ lab }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="可指派人数" width="200">
              <template #default="{ row }">
                <el-tag v-if="poolCount(row) <= 0" type="warning" size="small">未配置，App 提交会提示没有可指派的人</el-tag>
                <span v-else>
                  {{ poolCount(row) }}
                  <span v-if="poolPreviewNames(row)" class="muted"> · {{ poolPreviewNames(row) }}</span>
                </span>
              </template>
            </el-table-column>
            <el-table-column label="启用" width="70">
              <template #default="{ row }">{{ row.enabled ? '是' : '否' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link @click="openHandlers(row)">
                  {{ poolCount(row) <= 0 ? '去配置' : '配置处理人' }}
                </el-button>
                <el-button link @click="openEditType(row)">编辑字段</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag v-if="poolCount(row) <= 0" size="small" type="warning">未配置处理人</el-tag>
            <el-tag v-else size="small" type="success">可指派 {{ poolCount(row) }} 人</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button type="primary" link @click="openHandlers(row)">
              {{ poolCount(row) <= 0 ? '去配置' : '配置处理人' }}
            </el-button>
            <el-button link @click="openEditType(row)">编辑字段</el-button>
          </template>
        </TableOrCards>
      </el-tab-pane>

      <el-tab-pane label="工单跟踪" name="tickets">
        <p class="desc">这里是已提交工单的跟踪，不是配处理人的地方；处理人请到「业务类型与处理人」。</p>

        <div class="toolbar">
          <el-radio-group v-model="scope" size="small" @change="loadTickets">
            <el-radio-button value="mine_assignee">待我处理</el-radio-button>
            <el-radio-button value="mine_applicant">我发起的</el-radio-button>
            <el-radio-button value="">全部可见</el-radio-button>
          </el-radio-group>
          <el-select v-model="statusFilter" clearable placeholder="状态" style="width:120px;margin-left:8px" @change="loadTickets">
            <el-option v-for="(lab, k) in statusLabel" :key="k" :label="lab" :value="k" />
          </el-select>
          <el-button size="small" style="margin-left:8px" @click="loadTickets">刷新</el-button>
        </div>

        <TableOrCards
          :data="tickets"
          :loading="loading"
          :columns="ticketCols"
          style="margin-top:12px"
          empty-text="暂无工单。请先在上一页签配好处理人，再在员工 App 提交过磅入厂。"
        >
          <el-table :data="tickets" border stripe size="small" empty-text="暂无工单。请先在上一页签配好处理人，再在员工 App 提交过磅入厂。">
            <el-table-column prop="doc_no" label="单号" width="160" />
            <el-table-column prop="category_name" label="分类" width="140" />
            <el-table-column prop="title" label="标题" min-width="180" />
            <el-table-column prop="applicant_name" label="申请人" width="100" />
            <el-table-column prop="assignee_name" label="当前处理人" width="110" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">{{ statusLabel[String(row.status)] || row.status }}</template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="160" />
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openDetail(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small">{{ statusLabel[String(row.status)] || row.status }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
          </template>
        </TableOrCards>

        <el-collapse v-model="createOpen" class="manual-create">
          <el-collapse-item name="create" title="手工建单（工具领用等）">
            <p class="muted mb">过磅入厂请走员工 App，不要在这里手建。本表用于工具领用等协作单。</p>
            <el-form label-width="110px" size="small">
              <el-form-item label="工单类型">
                <el-select v-model="createForm.category_id" style="width:240px" @change="onCategoryChange">
                  <el-option v-for="c in categories" :key="String(c.id)" :label="String(c.name)" :value="Number(c.id)" />
                </el-select>
              </el-form-item>
              <el-form-item label="标题（可空）">
                <el-input v-model="createForm.title" placeholder="空则按表单内容自动生成" style="width:320px" />
              </el-form-item>
              <el-row :gutter="12">
                <el-col v-for="f in createSchema" :key="f.key" :span="f.type === 'textarea' ? 24 : 12" :xs="24">
                  <el-form-item :label="f.label + (f.required ? ' *' : '')">
                    <el-date-picker
                      v-if="f.type === 'date'"
                      v-model="createForm.payload[f.key]"
                      type="date"
                      value-format="YYYY-MM-DD"
                      style="width:100%"
                    />
                    <el-input-number
                      v-else-if="f.type === 'number'"
                      v-model="createForm.payload[f.key]"
                      :controls="false"
                      style="width:100%"
                    />
                    <el-select v-else-if="f.type === 'select'" v-model="createForm.payload[f.key]" style="width:100%">
                      <el-option v-for="o in f.options || []" :key="o" :label="o" :value="o" />
                    </el-select>
                    <el-input
                      v-else-if="f.type === 'textarea'"
                      v-model="createForm.payload[f.key]"
                      type="textarea"
                      :rows="2"
                    />
                    <el-input v-else v-model="createForm.payload[f.key]" :placeholder="f.unit || ''" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="下一手">
                <el-select v-model="createForm.next_assignee_user_id" style="width:220px" filterable :disabled="!pool.length">
                  <el-option
                    v-for="p in pool"
                    :key="String(p.user_id)"
                    :label="String(p.name || p.login_name)"
                    :value="Number(p.user_id)"
                  />
                </el-select>
                <el-button type="primary" style="margin-left:12px" @click="createTicket">创建</el-button>
                <p v-if="!pool.length" class="muted" style="margin:6px 0 0">请先到「业务类型与处理人」配置处理人池</p>
              </el-form-item>
            </el-form>
          </el-collapse-item>
        </el-collapse>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="detailVisible" title="工单详情" width="760px" destroy-on-close>
      <template v-if="detail">
        <p><b>{{ detail.doc_no }}</b> · {{ detail.category_name }} · {{ statusLabel[String(detail.status)] || detail.status }}</p>
        <p>{{ detail.title }}</p>
        <p class="muted">申请人 {{ detail.applicant_name }} → 当前 {{ detail.assignee_name || '-' }}</p>
        <el-divider>表单内容</el-divider>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item v-for="f in detailSchema" :key="f.key" :label="f.label">
            {{ detailPayload[f.key] ?? '-' }}{{ f.unit ? ` ${f.unit}` : '' }}
          </el-descriptions-item>
        </el-descriptions>
        <el-divider>流转日志</el-divider>
        <el-timeline>
          <el-timeline-item v-for="log in logs" :key="String(log.id)" :timestamp="String(log.created_at)">
            {{ log.action }} · {{ log.from_name || '-' }}
            <span v-if="log.to_name"> → {{ log.to_name }}</span>
            <span v-if="log.comment">（{{ log.comment }}）</span>
          </el-timeline-item>
        </el-timeline>
        <el-divider>处理</el-divider>
        <el-form label-width="90px" size="small">
          <el-form-item label="动作">
            <el-select v-model="actionForm.action" style="width:160px">
              <el-option label="通过/办结" value="approve" />
              <el-option v-if="detailIsTool" label="确认归还" value="return_confirm" />
              <el-option label="驳回" value="reject" />
              <el-option label="备注" value="comment" />
            </el-select>
          </el-form-item>
          <el-form-item label="下一手">
            <el-select v-model="actionForm.next_assignee_user_id" clearable filterable placeholder="空则结案" style="width:200px">
              <el-option
                v-for="p in pool"
                :key="String(p.user_id)"
                :label="String(p.name || p.login_name)"
                :value="Number(p.user_id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="意见"><el-input v-model="actionForm.comment" type="textarea" /></el-form-item>
        </el-form>
      </template>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button @click="doAssign">仅转交</el-button>
        <el-button type="primary" @click="doAction">提交</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="typeDialog"
      class="type-dialog"
      :title="isEditType ? '编辑工单类型' : '新建工单类型'"
      width="880px"
      top="8vh"
      destroy-on-close
    >
      <div class="type-dlg">
        <el-alert v-if="!isEditType" type="info" show-icon :closable="false" class="type-alert">
          过磅入厂、工具领用等已有类型请直接在列表里改，不必再建。新建类型保存后，还要去点「配置处理人」，否则无法指派。
        </el-alert>
        <el-alert v-else-if="isBuiltinType" type="warning" show-icon :closable="false" class="type-alert">
          这是系统内置类型（{{ typeForm.name || typeForm.code }}）。编码不可改；名称尽量保持原样，以免和员工 App 对不上。可指派的人请关掉本窗，到列表点「配置处理人」修改。
        </el-alert>

        <section class="type-section">
          <h4>基本信息</h4>
          <el-form label-width="108px" class="type-form">
            <el-form-item label="名称" required>
              <el-input
                :model-value="typeForm.name"
                maxlength="32"
                show-word-limit
                placeholder="员工看到的名字，如：设备报修"
                @update:model-value="onTypeNameInput"
              />
              <p class="field-hint">出现在工单列表、手工建单下拉。过磅入厂这类业务请用已有类型，不要新建同名。</p>
              <div v-if="!isEditType" class="code-examples">
                <span class="field-hint" style="margin:0 8px 0 0">可点示例带出名称和编码：</span>
                <el-button
                  v-for="ex in CODE_EXAMPLES"
                  :key="ex.code"
                  size="small"
                  text
                  type="primary"
                  @click="applyCodeExample(ex)"
                >{{ ex.name }}</el-button>
              </div>
            </el-form-item>
            <el-form-item label="编码" required>
              <el-input
                v-model="typeForm.code"
                :disabled="isEditType"
                maxlength="48"
                placeholder="小写英文，如 equipment_repair"
                @input="codeTouched = true"
              />
              <p class="field-hint">程序识别用，保存后不能改。仅小写字母、数字、下划线，须以字母开头。</p>
            </el-form-item>
            <el-form-item label="用途说明">
              <el-input
                v-model="typeForm.remark"
                type="textarea"
                :rows="2"
                maxlength="80"
                show-word-limit
                placeholder="一句话说明谁用、干什么。如：车间设备故障报修，派给维修班"
              />
              <p class="field-hint">显示在列表「做什么用」列，方便同事辨认，可空。</p>
            </el-form-item>
            <el-form-item label="启用">
              <div class="enable-row">
                <el-switch v-model="typeForm.enabled" />
                <span class="field-hint" style="margin:0">{{ typeForm.enabled ? '启用后可建单、可指派' : '停用后列表仍能看到，但不能再新建此类型工单' }}</span>
              </div>
            </el-form-item>
            <el-form-item label="关联页面">
              <el-input
                v-model="typeForm.biz_hint"
                placeholder="一般可空。内置类型才需要，如 /hr/tool-issues"
              />
              <p class="field-hint">管理端路径，给系统跳转用。自定义类型通常留空。</p>
            </el-form-item>
          </el-form>
        </section>

        <section class="type-section">
          <div class="schema-title">
            <h4>填报字段</h4>
            <span class="field-hint" style="margin:0">建单时要填的内容。「显示名称」给人看，「字段编码」给程序用。</span>
          </div>
          <div class="schema-table">
            <div class="schema-head">
              <span>#</span>
              <span>显示名称</span>
              <span>字段编码</span>
              <span>类型</span>
              <span>选项 / 单位</span>
              <span>必填</span>
              <span />
            </div>
            <div v-for="(f, i) in typeForm.form_schema" :key="i" class="schema-row">
              <span class="schema-idx">{{ i + 1 }}</span>
              <el-input
                :model-value="f.label"
                placeholder="如：车牌号"
                @update:model-value="(v: string) => onFieldLabelInput(f, i, v)"
              />
              <el-input v-model="f.key" placeholder="如：plate_no" />
              <el-select v-model="f.type" :placeholder="fieldTypeHint(f.type)">
                <el-option v-for="t in FIELD_TYPES" :key="t.value" :label="t.label" :value="t.value">
                  <span>{{ t.label }}</span>
                  <span class="opt-hint">{{ t.hint }}</span>
                </el-option>
              </el-select>
              <el-input
                v-if="f.type === 'select'"
                :model-value="(f.options || []).join('，')"
                placeholder="选项用逗号分隔，如：A，B，C"
                @update:model-value="(v: string) => (f.options = String(v).split(/[,，]/).map((x) => x.trim()).filter(Boolean))"
              />
              <el-input
                v-else-if="f.type === 'number'"
                v-model="f.unit"
                placeholder="单位，如 kg、元"
              />
              <span v-else class="schema-dash">—</span>
              <el-checkbox v-model="f.required">必填</el-checkbox>
              <el-button link type="danger" :disabled="typeForm.form_schema.length <= 1" @click="typeForm.form_schema.splice(i, 1)">
                删除
              </el-button>
            </div>
          </div>
          <div class="schema-actions">
            <el-button size="small" @click="addField()">添加字段</el-button>
            <el-button size="small" text type="primary" @click="addField({ key: 'qty', label: '数量', type: 'number', unit: 'kg' })">+ 数量</el-button>
            <el-button size="small" text type="primary" @click="addField({ key: 'grade', label: '等级', type: 'select', options: ['A', 'B', 'C'] })">+ 下拉</el-button>
            <el-button size="small" text type="primary" @click="addField({ key: 'remark', label: '备注', type: 'textarea' })">+ 备注</el-button>
          </div>
        </section>
      </div>
      <template #footer>
        <el-button @click="typeDialog = false">取消</el-button>
        <el-button type="primary" :loading="typeSaving" @click="saveType">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="handlerDialog" :title="`配置处理人 · ${handlerEdit.category_name || ''}`" width="640px">
      <el-alert type="info" show-icon :closable="false" class="mb">
        内置类型（过磅入厂等）也可以改处理人。名单按<strong>角色</strong>时，离职并停用账号的人会自动消失，给新人分配同一角色即可接单。也可以改成「用户」，只指定几个在职的人。
      </el-alert>
      <p class="muted mb">已办结的单不受影响。还挂在离职人员名下的待办，请到「工单跟踪」用管理员账号转交给在职同事。</p>
      <div v-for="(h, i) in handlerEdit.handlers" :key="i" class="handler-row">
        <el-select v-model="h.handler_type" style="width:100px">
          <el-option label="角色" value="role" />
          <el-option label="用户" value="user" />
        </el-select>
        <el-select v-if="h.handler_type === 'role'" v-model="h.handler_ref" filterable style="width:280px;margin-left:8px">
          <el-option v-for="r in roles" :key="String(r.id)" :label="`${r.name}(${r.code})`" :value="Number(r.id)" />
        </el-select>
        <el-select v-else v-model="h.handler_ref" filterable style="width:280px;margin-left:8px">
          <el-option
            v-for="u in handlerUserOptions(h.handler_ref)"
            :key="String(u.id)"
            :label="userOptionLabel(u)"
            :value="Number(u.id)"
          />
        </el-select>
        <el-button link type="danger" @click="handlerEdit.handlers.splice(i, 1)">删</el-button>
      </div>
      <el-button size="small" @click="addHandlerRow" style="margin-top:8px">添加</el-button>
      <p v-if="selectedHandlerLabels.length" class="muted" style="margin-top:12px">已选：{{ selectedHandlerLabels.join('、') }}</p>
      <p v-if="handlerPoolPreview.length" class="muted">
        当前在职可指派：{{ handlerPoolPreview.map((p) => String(p.name || p.login_name)).filter(Boolean).join('、') }}
      </p>
      <p v-else class="muted">当前在职可指派人为空。请添加仍在职的角色/用户，或给新人挂上仓管、质检等角色后再打开本窗。</p>
      <template #footer>
        <el-button @click="handlerDialog = false">取消</el-button>
        <el-button type="primary" @click="saveHandlers">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 16px; background: #fff; border-radius: 8px; border: 1px solid #d5dde3; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.mb { margin-bottom: 12px; }
.toolbar { display: flex; align-items: center; flex-wrap: wrap; gap: 4px; }
.muted { color: #6b7280; font-size: 13px; }
.handler-row { display: flex; align-items: center; margin-bottom: 8px; flex-wrap: wrap; gap: 4px; }
.howto { margin-bottom: 14px; }
.type-dlg { display: flex; flex-direction: column; gap: 16px; }
.type-alert { margin: 0; }
.type-section {
  border: 1px solid #e8eef2;
  border-radius: 8px;
  padding: 14px 16px 8px;
  background: #fafbfc;
}
.type-section h4 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: #1f2a33;
}
.type-form :deep(.el-form-item) { margin-bottom: 14px; }
.field-hint { margin: 6px 0 0; font-size: 12px; color: #6b7a85; line-height: 1.5; }
.code-examples { display: flex; align-items: center; flex-wrap: wrap; margin-top: 4px; }
.enable-row { display: flex; align-items: center; gap: 10px; min-height: 32px; }
.schema-title { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; margin-bottom: 10px; }
.schema-title h4 { margin: 0; }
.schema-table { display: flex; flex-direction: column; gap: 8px; }
.schema-head, .schema-row {
  display: grid;
  grid-template-columns: 28px 1.1fr 1fr 108px 1.3fr 56px 48px;
  gap: 8px;
  align-items: center;
}
.schema-head {
  font-size: 12px;
  color: #6b7a85;
  font-weight: 600;
  padding: 0 2px;
}
.schema-idx { font-size: 12px; color: #98a2a8; text-align: center; }
.schema-dash { color: #c5ced4; text-align: center; }
.schema-actions { display: flex; flex-wrap: wrap; gap: 4px; margin: 12px 0 4px; }
.opt-hint { float: right; color: #98a2a8; font-size: 12px; margin-left: 12px; }
@media (max-width: 760px) {
  .schema-head { display: none; }
  .schema-row {
    grid-template-columns: 1fr 1fr;
    background: #fff;
    border: 1px solid #e8eef2;
    border-radius: 8px;
    padding: 10px;
  }
  .schema-idx { display: none; }
}
.howto-list { margin: 6px 0 0; padding-left: 18px; color: #334155; font-size: 13px; line-height: 1.6; }
.howto-sub { margin: 10px 0 0; font-weight: 600; color: #1f2937; }
.role-tag { margin-right: 4px; }
.manual-create { margin-top: 16px; }
</style>
