<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  moduleList,
  moduleCreate,
  moduleUpdate,
  moduleDelete,
  moduleAction,
  moduleReplace,
  CURRENCY_OPTIONS,
  TIMEZONE_OPTIONS,
  DATE_FORMAT_OPTIONS,
  STATUS_ACTIVE_OPTIONS,
  APPROVAL_DOC_TYPE_OPTIONS,
  type FormOption,
} from '@erp/shared'
import {
  WarehouseSelect,
  WorkshopSelect,
  EmployeeSelect,
  UserSelect,
  ProductSelect,
  CustomerSelect,
  RoleSelect,
  EnumSelect,
} from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>
type RefKind = 'warehouse' | 'workshop' | 'employee' | 'user' | 'product' | 'customer' | 'role'
type FieldDef = {
  key: string
  label: string
  type?: 'text' | 'number' | 'switch' | 'textarea' | 'select' | 'date' | 'month' | 'datetime' | 'ref'
  options?: FormOption[]
  ref?: RefKind
}

const props = defineProps<{ module: string; listPath: string; meta?: {
  create?: string
  update?: string
  remove?: string
  detail?: string
  actions?: string[]
  readOnly?: boolean
} | null }>()

const LOGIC_OPTIONS: FormOption[] = [
  { value: 'AND', label: 'AND' },
  { value: 'OR', label: 'OR' },
]

const SETTINGS: Record<string, { title: string; fields: FieldDef[] }> = {
  基础设置: {
    title: '基础设置',
    fields: [
      { key: 'company_name', label: '公司名称' },
      { key: 'timezone', label: '时区', type: 'select', options: TIMEZONE_OPTIONS },
      { key: 'currency', label: '币种', type: 'select', options: CURRENCY_OPTIONS },
      { key: 'date_format', label: '日期格式', type: 'select', options: DATE_FORMAT_OPTIONS },
      { key: 'default_page_size', label: '默认分页', type: 'number' },
      { key: 'enable_mqtt_notify', label: 'MQTT 通知', type: 'switch' },
      {
        key: 'farmer_settle_point',
        label: '农户结算环节',
        type: 'select',
        options: [
          { value: 'gate', label: '入厂确认后（按票净重）' },
          { value: 'box_stockin', label: '分箱入库后（按箱合计）' },
        ],
      },
    ],
  },
  销售设置: {
    title: '销售设置',
    fields: [
      { key: 'default_tax_rate', label: '默认税率', type: 'number' },
      { key: 'allow_negative_stock', label: '允许负库存', type: 'switch' },
      { key: 'require_pre_ship', label: '必须预发货', type: 'switch' },
      { key: 'default_warehouse_id', label: '默认成品仓', type: 'ref', ref: 'warehouse' },
      { key: 'price_precision', label: '价格小数位', type: 'number' },
    ],
  },
  生产设置: {
    title: '生产设置',
    fields: [
      { key: 'auto_inbound_on_qc', label: '质检后自动入库', type: 'switch' },
      { key: 'require_box_code', label: '强制箱码', type: 'switch' },
      { key: 'default_workshop_id', label: '默认车间', type: 'ref', ref: 'workshop' },
      { key: 'piecework_confirm_required', label: '计件需确认', type: 'switch' },
    ],
  },
  表格自定义: {
    title: '表格显示',
    fields: [
      { key: 'dense', label: '紧凑行高', type: 'switch' },
      { key: 'stripe', label: '斑马纹', type: 'switch' },
      { key: 'show_id', label: '显示 ID 列', type: 'switch' },
    ],
  },
  单据审批: {
    title: '单据审批开关',
    fields: [
      { key: 'sales_order', label: '销售订单', type: 'switch' },
      { key: 'purchase_inbound', label: '采购入库', type: 'switch' },
      { key: 'stock_txn', label: '库存单据', type: 'switch' },
      { key: 'payroll_sheet', label: '工资单', type: 'switch' },
    ],
  },
  单据锁定: {
    title: '单据锁定规则',
    fields: [
      { key: 'lock_after_approve', label: '审批后锁定', type: 'switch' },
      { key: 'lock_after_days', label: 'N 天后锁定', type: 'number' },
      { key: 'allow_admin_unlock', label: '管理员可解锁', type: 'switch' },
    ],
  },
  单据通知: {
    title: '单据通知规则',
    fields: [
      { key: 'on_approve', label: '审批通过通知', type: 'switch' },
      { key: 'on_reject', label: '驳回通知', type: 'switch' },
      { key: 'on_assign', label: '指派通知', type: 'switch' },
    ],
  },
  单据编辑: {
    title: '单据编辑规则',
    fields: [
      { key: 'allow_edit_draft', label: '草稿可编辑', type: 'switch' },
      { key: 'allow_edit_after_approve', label: '审批后可编辑', type: 'switch' },
      { key: 'track_versions', label: '记录版本', type: 'switch' },
    ],
  },
  单据删除: {
    title: '单据删除规则',
    fields: [
      { key: 'allow_delete_draft', label: '草稿可删', type: 'switch' },
      { key: 'soft_delete_only', label: '仅软删除', type: 'switch' },
      { key: 'require_reason', label: '删除需原因', type: 'switch' },
    ],
  },
  多条件检索: {
    title: '多条件检索',
    fields: [
      { key: 'enable_advanced', label: '启用高级检索', type: 'switch' },
      { key: 'max_conditions', label: '最大条件数', type: 'number' },
      { key: 'default_operator', label: '默认逻辑', type: 'select', options: LOGIC_OPTIONS },
    ],
  },
  财审管控: {
    title: '财审管控',
    fields: [
      { key: 'require_finance_approve', label: '需财务审批', type: 'switch' },
      { key: 'amount_threshold', label: '金额阈值', type: 'number' },
    ],
  },
}

const CRUD_FIELDS: Record<string, FieldDef[]> = {
  自定义打印: [
    { key: 'code', label: '编码' }, { key: 'name', label: '名称' },
    { key: 'doc_type', label: '单据类型', type: 'select', options: APPROVAL_DOC_TYPE_OPTIONS },
    { key: 'content', label: '模板内容', type: 'textarea' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  公式设置: [
    { key: 'code', label: '编码' }, { key: 'name', label: '名称' }, { key: 'scope', label: '作用域' },
    { key: 'expression', label: '表达式', type: 'textarea' }, { key: 'remark', label: '备注' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  物流信息管理: [
    { key: 'code', label: '编码' }, { key: 'name', label: '承运商' }, { key: 'contact', label: '联系人' },
    { key: 'phone', label: '电话' }, { key: 'remark', label: '备注' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  审批流程设定: [
    { key: 'code', label: '编码' }, { key: 'name', label: '名称' },
    { key: 'doc_type', label: '单据类型', type: 'select', options: APPROVAL_DOC_TYPE_OPTIONS },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  人事调动: [
    { key: 'employee_id', label: '员工', type: 'ref', ref: 'employee' },
    { key: 'from_dept_id', label: '原部门ID', type: 'number' }, { key: 'to_dept_id', label: '新部门ID', type: 'number' },
    { key: 'from_workshop_id', label: '原车间', type: 'ref', ref: 'workshop' },
    { key: 'to_workshop_id', label: '新车间', type: 'ref', ref: 'workshop' },
    { key: 'reason', label: '原因', type: 'textarea' }, { key: 'effective_date', label: '生效日', type: 'date' },
  ],
  批量改价: [
    { key: 'target_type', label: '目标类型' }, { key: 'adjust_type', label: '调整方式' },
    { key: 'adjust_value', label: '调整值', type: 'number' }, { key: 'scope_json', label: '范围说明', type: 'textarea' },
  ],
  批量核算工资: [
    { key: 'period_ym', label: '期间', type: 'month' },
    { key: 'workshop_id', label: '车间', type: 'ref', ref: 'workshop' },
  ],
  事项提醒: [
    { key: 'title', label: '标题' }, { key: 'content', label: '内容', type: 'textarea' },
    { key: 'remind_at', label: '提醒时间', type: 'datetime' },
    { key: 'target_user_id', label: '用户', type: 'ref', ref: 'user' },
    { key: 'target_role', label: '角色' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  学堂管理: [
    { key: 'code', label: '编码' }, { key: 'title', label: '标题' }, { key: 'category', label: '分类' },
    { key: 'content', label: '内容', type: 'textarea' }, { key: 'duration_min', label: '时长(分)', type: 'number' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  知识库: [
    { key: 'code', label: '编码' }, { key: 'title', label: '标题' }, { key: 'category', label: '分类' },
    { key: 'content', label: '内容', type: 'textarea' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  图纸管理: [
    { key: 'code', label: '编码' }, { key: 'title', label: '标题' },
    { key: 'product_id', label: '产品', type: 'ref', ref: 'product' },
    { key: 'version_no', label: '版本' }, { key: 'file_url', label: '文件URL' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  文档管理: [
    { key: 'code', label: '编码' }, { key: 'title', label: '标题' }, { key: 'category', label: '分类' },
    { key: 'content', label: '内容', type: 'textarea' }, { key: 'file_url', label: '文件URL' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  公告设置: [
    { key: 'title', label: '标题' }, { key: 'content', label: '内容', type: 'textarea' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
  备忘录: [
    { key: 'title', label: '标题' }, { key: 'content', label: '内容', type: 'textarea' },
    { key: 'owner_id', label: '所有者', type: 'ref', ref: 'user' },
    { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
  ],
}

const isSetting = computed(() => !!SETTINGS[props.module])
const formFields = computed(() => CRUD_FIELDS[props.module] || [
  { key: 'name', label: '名称' }, { key: 'code', label: '编码' }, { key: 'remark', label: '备注' },
  { key: 'status', label: '状态', type: 'select', options: STATUS_ACTIVE_OPTIONS },
])

const loading = ref(false)
const list = ref<Row[]>([])
const total = ref(0)
const settingForm = reactive<Row>({})
const dlg = ref(false)
const editingId = ref<number | null>(null)
const form = reactive<Row>({})

async function load() {
  if (!props.listPath) {
    ElMessage.warning('未配置 API 路径')
    return
  }
  loading.value = true
  try {
    const r = await moduleList(props.listPath)
    if (r.code !== 1) {
      ElMessage.error(r.msg || '加载失败')
      return
    }
    const data = r.data as { list?: Row[]; total?: number }
    const rows = data?.list || []
    list.value = rows
    total.value = data?.total ?? rows.length
    if (isSetting.value) {
      const row = rows[0] || {}
      Object.keys(settingForm).forEach((k) => delete settingForm[k])
      Object.assign(settingForm, row)
      for (const f of SETTINGS[props.module].fields) {
        if (settingForm[f.key] === undefined) {
          settingForm[f.key] = f.type === 'switch' ? false : (f.type === 'number' || f.type === 'ref') ? 0 : ''
        }
      }
    }
  } finally {
    loading.value = false
  }
}

async function saveSetting() {
  const body = { ...settingForm }
  delete body.id
  delete body.setting_key
  const r = await moduleReplace(props.listPath, body)
  if (r.code !== 1) return ElMessage.error(r.msg || '保存失败')
  ElMessage.success('已保存')
  await load()
}

function openCreate() {
  editingId.value = null
  Object.keys(form).forEach((k) => delete form[k])
  for (const f of formFields.value) {
    form[f.key] = f.type === 'number' || f.type === 'ref' ? null : f.type === 'switch' ? false : ''
  }
  if (props.module === '批量核算工资') {
    const d = new Date()
    form.period_ym = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
  }
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id)
  Object.keys(form).forEach((k) => delete form[k])
  Object.assign(form, { ...row })
  dlg.value = true
}

async function saveRow() {
  const body = { ...form }
  delete body.id
  let r
  if (editingId.value) {
    const path = props.meta?.update || props.meta?.detail || `${props.listPath}/{id}`
    r = await moduleUpdate(path, editingId.value, body)
  } else {
    r = await moduleCreate(props.meta?.create || props.listPath, body)
  }
  if (r.code !== 1) return ElMessage.error(r.msg || '保存失败')
  ElMessage.success('已保存')
  dlg.value = false
  await load()
}

async function onDelete(row: Row) {
  await ElMessageBox.confirm('确认删除？', '提示')
  const path = props.meta?.remove || props.meta?.detail || `${props.listPath}/{id}`
  const r = await moduleDelete(path, Number(row.id))
  if (r.code !== 1) return ElMessage.error(r.msg || '删除失败')
  ElMessage.success('已删除')
  await load()
}

async function onAction(row: Row, action: string) {
  const path = props.meta?.detail || `${props.listPath}/{id}`
  const r = await moduleAction(path, Number(row.id), action)
  if (r.code !== 1) return ElMessage.error(r.msg || '执行失败')
  ElMessage.success(`已执行 ${action}`)
  await load()
}

const tableCols = computed(() => {
  const keys = new Set<string>()
  for (const f of formFields.value) keys.add(f.key)
  keys.add('id')
  keys.add('status')
  keys.add('doc_no')
  keys.add('result_msg')
  keys.add('published_at')
  keys.add('created_at')
  const fromData = list.value[0] ? Object.keys(list.value[0]) : []
  const ordered = [...keys].filter((k) => fromData.includes(k) || formFields.value.some((f) => f.key === k))
  return ordered.length ? ordered : fromData.filter((k) => !String(k).startsWith('_')).slice(0, 10)
})

const colLabel = (k: string) => formFields.value.find((f) => f.key === k)?.label || k

const cardColumns = computed<MobileCardColumn[]>(() => {
  const cols = tableCols.value
  const primaryProp =
    cols.find((c) => c === 'name' || c === 'code' || c === 'doc_no' || c === 'title') || cols[0]
  return cols.map((col, i) => ({
    prop: col,
    label: colLabel(col),
    primary: col === primaryProp,
    hideOnCard:
      cols.length > 8 &&
      i >= 6 &&
      !['status', 'name', 'code', 'doc_no', 'id'].includes(col) &&
      col !== primaryProp,
  }))
})

watch(() => props.module, () => load())
onMounted(load)
</script>

<template>
  <div v-loading="loading" class="panel">
    <h2 class="title">{{ module }}</h2>
    <p class="desc">系统管理 · 配置即时生效，可直接交付使用</p>

    <template v-if="isSetting">
      <el-form label-width="140px" style="max-width:560px">
        <el-form-item v-for="f in SETTINGS[module].fields" :key="f.key" :label="f.label">
          <el-switch v-if="f.type === 'switch'" v-model="settingForm[f.key]" />
          <el-input-number v-else-if="f.type === 'number'" v-model="settingForm[f.key]" :controls="true" style="width:100%" />
          <el-input v-else-if="f.type === 'textarea'" v-model="settingForm[f.key]" type="textarea" :rows="3" />
          <EnumSelect v-else-if="f.type === 'select'" v-model="settingForm[f.key] as string" :options="f.options || []" style="width:100%" />
          <WarehouseSelect v-else-if="f.type === 'ref' && f.ref === 'warehouse'" v-model="settingForm[f.key] as number" style="width:100%" />
          <WorkshopSelect v-else-if="f.type === 'ref' && f.ref === 'workshop'" v-model="settingForm[f.key] as number" style="width:100%" />
          <el-input v-else v-model="settingForm[f.key]" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveSetting">保存配置</el-button>
          <el-button @click="load">刷新</el-button>
        </el-form-item>
      </el-form>
    </template>

    <template v-else>
      <div class="toolbar">
        <el-button v-if="meta?.create && !meta?.readOnly" type="primary" @click="openCreate">新建</el-button>
        <el-button @click="load">刷新</el-button>
        <span class="spacer" />
        <span class="muted">共 {{ total }} 条</span>
      </div>
      <TableOrCards :data="list" :loading="loading" :columns="cardColumns">
        <el-table :data="list" border stripe style="width:100%">
          <el-table-column
            v-for="col in tableCols"
            :key="col"
            :prop="col"
            :label="colLabel(col)"
            min-width="110"
            show-overflow-tooltip
          />
          <el-table-column label="操作" width="240" fixed="right">
            <template #default="{ row }">
              <el-button v-if="meta?.update && !meta?.readOnly" link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button
                v-for="act in (meta?.actions || [])"
                :key="act"
                link
                type="success"
                @click="onAction(row, act)"
              >{{ act }}</el-button>
              <el-button v-if="meta?.remove && !meta?.readOnly" link type="danger" @click="onDelete(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="meta?.update && !meta?.readOnly" link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button
            v-for="act in (meta?.actions || [])"
            :key="act"
            link
            type="success"
            @click="onAction(row, act)"
          >{{ act }}</el-button>
          <el-button v-if="meta?.remove && !meta?.readOnly" link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </TableOrCards>
    </template>

    <el-dialog v-model="dlg" :title="editingId ? '编辑' : '新建'" width="560px" destroy-on-close>
      <el-form label-width="120px">
        <el-form-item v-for="f in formFields" :key="f.key" :label="f.label">
          <el-switch v-if="f.type === 'switch'" v-model="form[f.key]" />
          <el-input-number v-else-if="f.type === 'number'" v-model="form[f.key]" style="width:100%" />
          <el-input v-else-if="f.type === 'textarea'" v-model="form[f.key]" type="textarea" :rows="4" />
          <EnumSelect v-else-if="f.type === 'select'" v-model="form[f.key] as string" :options="f.options || []" style="width:100%" />
          <el-date-picker v-else-if="f.type === 'date'" v-model="form[f.key]" type="date" value-format="YYYY-MM-DD" style="width:100%" />
          <el-date-picker v-else-if="f.type === 'month'" v-model="form[f.key]" type="month" value-format="YYYY-MM" style="width:100%" />
          <el-date-picker v-else-if="f.type === 'datetime'" v-model="form[f.key]" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:100%" />
          <WarehouseSelect v-else-if="f.type === 'ref' && f.ref === 'warehouse'" v-model="form[f.key] as number" style="width:100%" />
          <WorkshopSelect v-else-if="f.type === 'ref' && f.ref === 'workshop'" v-model="form[f.key] as number" style="width:100%" />
          <EmployeeSelect v-else-if="f.type === 'ref' && f.ref === 'employee'" v-model="form[f.key] as number" style="width:100%" />
          <UserSelect v-else-if="f.type === 'ref' && f.ref === 'user'" v-model="form[f.key] as number" style="width:100%" />
          <ProductSelect v-else-if="f.type === 'ref' && f.ref === 'product'" v-model="form[f.key] as number" style="width:100%" />
          <CustomerSelect v-else-if="f.type === 'ref' && f.ref === 'customer'" v-model="form[f.key] as number" style="width:100%" />
          <RoleSelect v-else-if="f.type === 'ref' && f.ref === 'role'" v-model="form[f.key] as number" style="width:100%" />
          <el-input v-else v-model="form[f.key]" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="saveRow">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.panel { background: #fff; border-radius: 8px; padding: 16px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; font-size: 18px; }
.desc { margin: 0 0 16px; color: #5c6b75; font-size: 13px; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.spacer { flex: 1; }
.muted { color: #5c6b75; font-size: 13px; }
</style>
