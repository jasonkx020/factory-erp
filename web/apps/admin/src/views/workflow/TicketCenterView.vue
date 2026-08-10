<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'remark', label: '说明' },
]
type FieldDef = {
  key: string
  label: string
  type: string
  required?: boolean
  options?: string[]
  unit?: string
}

const tab = ref<'tickets' | 'categories'>('tickets')
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
  handlers: [] as { handler_type: string; handler_ref: number }[],
})
const handlerDialog = ref(false)

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

const statusLabel: Record<string, string> = {
  open: '待处理',
  in_progress: '处理中',
  done: '已办结',
  rejected: '已驳回',
  cancelled: '已取消',
}

function schemaOf(cat: Row | undefined): FieldDef[] {
  if (!cat) return []
  const arr = cat.form_schema
  if (Array.isArray(arr)) return arr as FieldDef[]
  return []
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
    return ElMessage.warning('请选择分类并指定下一手处理人')
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
  const res = await ticketApi.getHandlers(handlerEdit.category_id)
  const list = ((res.data as { handlers?: Row[] })?.handlers) || []
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

async function saveHandlers() {
  const handlers = handlerEdit.handlers.filter((h) => h.handler_ref > 0)
  const res = await ticketApi.putHandlers(handlerEdit.category_id, handlers)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('处理人池已保存')
  handlerDialog.value = false
  await loadMeta()
}

function openNewType() {
  Object.assign(typeForm, {
    id: 0,
    code: '',
    name: '',
    remark: '',
    biz_hint: '',
    enabled: true,
    form_schema: [{ key: 'biz_date', label: '日期', type: 'date', required: true }],
  })
  typeDialog.value = true
}

function openEditType(row: Row) {
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
    typeForm.form_schema.push({ key: 'remark', label: '备注', type: 'textarea', required: false })
  }
  typeDialog.value = true
}

function addField() {
  typeForm.form_schema.push({
    key: `field_${typeForm.form_schema.length + 1}`,
    label: '新字段',
    type: 'text',
    required: false,
    options: [],
  })
}

async function saveType() {
  if (!typeForm.code || !typeForm.name) return ElMessage.warning('请填写编码与名称')
  const body = {
    code: typeForm.code,
    name: typeForm.name,
    remark: typeForm.remark,
    biz_hint: typeForm.biz_hint,
    enabled: typeForm.enabled,
    form_schema: typeForm.form_schema.map((f) => ({
      ...f,
      options: f.type === 'select' ? (f.options || []).filter(Boolean) : undefined,
    })),
  }
  let res
  if (isEditType.value) {
    res = await ticketApi.updateCategory(typeForm.id, body)
  } else {
    res = await ticketApi.createCategory(body)
  }
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(isEditType.value ? '类型已更新' : '类型已创建')
  typeDialog.value = false
  await loadMeta()
}

const logs = computed(() => ((detail.value?.logs as Row[]) || []) as Row[])

watch(
  () => createForm.category_id,
  () => {
    /* handled by @change */
  },
)

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>工单中心</h2>
    <p class="desc">
      按类型配置动态字段与处理人池；创建/处理时可指定下一手。库存查询/盘点等仍走库存模块，不作为工单类型。
    </p>

    <el-tabs v-model="tab">
      <el-tab-pane label="工单跟踪" name="tickets">
        <el-card header="按类型创建工单" shadow="never" class="mb">
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
              <el-select v-model="createForm.next_assignee_user_id" style="width:220px" filterable>
                <el-option
                  v-for="p in pool"
                  :key="String(p.user_id)"
                  :label="String(p.name || p.login_name)"
                  :value="Number(p.user_id)"
                />
              </el-select>
              <el-button type="primary" style="margin-left:12px" @click="createTicket">创建</el-button>
            </el-form-item>
          </el-form>
        </el-card>

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

        <TableOrCards :data="tickets" :loading="loading" :columns="ticketCols" style="margin-top:12px">
          <el-table :data="tickets" border stripe size="small">
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
      </el-tab-pane>

      <el-tab-pane label="分类与字段" name="categories">
        <div style="margin-bottom:12px">
          <el-button type="primary" size="small" @click="openNewType">新建工单类型</el-button>
        </div>
        <TableOrCards :data="categories" :loading="loading" :columns="categoryCols">
          <el-table :data="categories" border stripe size="small">
            <el-table-column prop="code" label="编码" width="140" />
            <el-table-column prop="name" label="名称" width="160" />
            <el-table-column prop="remark" label="说明" min-width="160" />
            <el-table-column label="字段数" width="80">
              <template #default="{ row }">{{ schemaOf(row).length }}</template>
            </el-table-column>
            <el-table-column label="启用" width="70">
              <template #default="{ row }">{{ row.enabled ? '是' : '否' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="220">
              <template #default="{ row }">
                <el-button link type="primary" @click="openEditType(row)">编辑字段</el-button>
                <el-button link type="primary" @click="openHandlers(row)">处理人</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #extra="{ row }">
            <el-tag size="small" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openEditType(row)">编辑字段</el-button>
            <el-button link type="primary" @click="openHandlers(row)">处理人</el-button>
          </template>
        </TableOrCards>
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
              <el-option label="确认归还" value="return_confirm" />
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
        <el-button type="primary" @click="doAction">提交动作</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="typeDialog" :title="isEditType ? '编辑工单类型' : '新建工单类型'" width="720px" destroy-on-close>
      <el-form label-width="90px" size="small">
        <el-form-item label="编码" required>
          <el-input v-model="typeForm.code" :disabled="isEditType" placeholder="如 custom_repair" />
        </el-form-item>
        <el-form-item label="名称" required><el-input v-model="typeForm.name" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="typeForm.remark" /></el-form-item>
        <el-form-item label="业务提示"><el-input v-model="typeForm.biz_hint" placeholder="可选，如 /inventory/hub/inbound" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="typeForm.enabled" /></el-form-item>
      </el-form>
      <el-divider>表单字段</el-divider>
      <div v-for="(f, i) in typeForm.form_schema" :key="i" class="field-row">
        <el-input v-model="f.key" placeholder="key" style="width:120px" />
        <el-input v-model="f.label" placeholder="标签" style="width:120px;margin-left:6px" />
        <el-select v-model="f.type" style="width:110px;margin-left:6px">
          <el-option label="文本" value="text" />
          <el-option label="数字" value="number" />
          <el-option label="日期" value="date" />
          <el-option label="下拉" value="select" />
          <el-option label="多行" value="textarea" />
        </el-select>
        <el-input
          v-if="f.type === 'select'"
          :model-value="(f.options || []).join(',')"
          placeholder="选项,逗号分隔"
          style="width:160px;margin-left:6px"
          @update:model-value="(v: string) => (f.options = String(v).split(',').map((x) => x.trim()).filter(Boolean))"
        />
        <el-input v-model="f.unit" placeholder="单位" style="width:70px;margin-left:6px" />
        <el-checkbox v-model="f.required" style="margin-left:8px">必填</el-checkbox>
        <el-button link type="danger" @click="typeForm.form_schema.splice(i, 1)">删</el-button>
      </div>
      <el-button size="small" style="margin-top:8px" @click="addField">添加字段</el-button>
      <template #footer>
        <el-button @click="typeDialog = false">取消</el-button>
        <el-button type="primary" @click="saveType">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="handlerDialog" title="配置处理人池" width="560px">
      <div v-for="(h, i) in handlerEdit.handlers" :key="i" class="handler-row">
        <el-select v-model="h.handler_type" style="width:100px">
          <el-option label="角色" value="role" />
          <el-option label="用户" value="user" />
        </el-select>
        <el-select v-if="h.handler_type === 'role'" v-model="h.handler_ref" filterable style="width:220px;margin-left:8px">
          <el-option v-for="r in roles" :key="String(r.id)" :label="`${r.name}(${r.code})`" :value="Number(r.id)" />
        </el-select>
        <el-select v-else v-model="h.handler_ref" filterable style="width:220px;margin-left:8px">
          <el-option v-for="u in users" :key="String(u.id)" :label="String(u.login_name)" :value="Number(u.id)" />
        </el-select>
        <el-button link type="danger" @click="handlerEdit.handlers.splice(i, 1)">删</el-button>
      </div>
      <el-button size="small" @click="addHandlerRow" style="margin-top:8px">添加</el-button>
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
.handler-row, .field-row { display: flex; align-items: center; margin-bottom: 8px; flex-wrap: wrap; gap: 4px; }
</style>
