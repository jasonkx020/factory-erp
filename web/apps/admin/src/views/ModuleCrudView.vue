<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  useAuthStore,
  usePermStore,
  moduleList,
  moduleCreate,
  moduleUpdate,
  moduleDelete,
  moduleAction,
  STATUS_ACTIVE_OPTIONS,
  adminModuleForPath,
} from '@erp/shared'
import IamView from './iam/IamView.vue'
import SupplierView from './purchase/SupplierView.vue'
import FarmerInboundView from './purchase/FarmerInboundView.vue'
import OnboardView from './hr/OnboardView.vue'
import HrOpsView from './hr/HrOpsView.vue'
import EmployeeView from './hr/EmployeeView.vue'
import DeptView from './hr/DeptView.vue'
import SystemAdminView from './system/SystemAdminView.vue'
import PayrollView from './payroll/PayrollView.vue'
import {
  WarehouseSelect,
  WorkshopSelect,
  EmployeeSelect,
  UserSelect,
  ProductSelect,
  CustomerSelect,
  SupplierSelect,
  ProcessSelect,
  SalesOrderSelect,
  EnumSelect,
} from '../components/select'
import MobileDataCards from '../components/mobile/MobileDataCards.vue'
import { useIsMobile } from '../composables/useMediaQuery'
import type { MobileCardColumn } from '../components/mobile/MobileDataCards.vue'

type FieldKind = 'text' | 'status' | 'date' | 'month' | 'datetime' | 'ref'
type RefKind = 'warehouse' | 'workshop' | 'employee' | 'user' | 'product' | 'customer' | 'supplier' | 'process' | 'order'

function fieldKind(key: string): { kind: FieldKind; ref?: RefKind } {
  if (key === 'status') return { kind: 'status' }
  if (key === 'period' || key === 'period_ym' || key.endsWith('_ym')) return { kind: 'month' }
  if (key.endsWith('_at') || key.includes('datetime') || key.includes('remind_at')) return { kind: 'datetime' }
  if (key.includes('date') || key.endsWith('_day') || key === 'biz_date') return { kind: 'date' }
  const refMap: Record<string, RefKind> = {
    warehouse_id: 'warehouse',
    from_warehouse_id: 'warehouse',
    to_warehouse_id: 'warehouse',
    workshop_id: 'workshop',
    employee_id: 'employee',
    worker_id: 'employee',
    user_id: 'user',
    to_user_id: 'user',
    assignee_id: 'user',
    owner_id: 'user',
    target_user_id: 'user',
    product_id: 'product',
    customer_id: 'customer',
    supplier_id: 'supplier',
    process_id: 'process',
    order_id: 'order',
    sales_order_id: 'order',
  }
  if (refMap[key]) return { kind: 'ref', ref: refMap[key] }
  return { kind: 'text' }
}

const route = useRoute()
const auth = useAuthStore()
const perm = usePermStore()
const isMobile = useIsMobile()

const pathHit = computed(() => adminModuleForPath(route.path))
const domain = computed(() => {
  if (pathHit.value) return pathHit.value.domain
  return decodeURIComponent(String(route.params.domain || ''))
})
const moduleName = computed(() => {
  if (pathHit.value) return pathHit.value.module
  return decodeURIComponent(String(route.params.module || ''))
})
const meta = computed(() => perm.metaFor(domain.value, moduleName.value))

const loading = ref(false)
const list = ref<Record<string, unknown>[]>([])
const total = ref(0)
const columns = ref<string[]>([])
const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const form = reactive<Record<string, unknown>>({})
const formKeys = ref<string[]>(['name', 'code', 'remark', 'status'])

const costKeys = ['cost_price', 'cost', 'gross_margin', 'gross_profit']

function visibleColumns() {
  return columns.value.filter((k) => {
    if (costKeys.includes(k) && !auth.fieldVisible(k)) return false
    return true
  })
}

const cardColumns = computed<MobileCardColumn[]>(() => {
  const cols = visibleColumns()
  const primaryProp =
    cols.find((c) => c === 'name' || c === 'code' || c === 'doc_no' || c === 'title') || cols[0]
  return cols.map((col, i) => ({
    prop: col,
    label: col,
    primary: col === primaryProp,
    hideOnCard: cols.length > 8 && i >= 6 && !['status', 'name', 'code', 'doc_no', 'id'].includes(col) && col !== primaryProp,
  }))
})

async function load() {
  if (perm.isIamModule(moduleName.value) || perm.isSupplierModule(moduleName.value) || perm.isFarmerInboundModule(moduleName.value) || perm.isOnboardModule(moduleName.value) || perm.isEmployeeModule(moduleName.value) || perm.isDeptModule(moduleName.value) || perm.isHrOpsModule(moduleName.value) || perm.isSystemAdminModule(moduleName.value) || perm.isPayrollModule(moduleName.value)) return
  const m = meta.value
  if (!m?.list) {
    list.value = []
    total.value = 0
    columns.value = []
    if (m?.actionOnly || m?.create) {
      ElMessage.info('该模块为操作型接口，无列表资源；可通过「新建/执行」提交')
    } else {
      ElMessage.warning('未找到模块可列表 API 映射')
    }
    return
  }
  loading.value = true
  try {
    const r = await moduleList(m.list)
    if (r.code !== 1) {
      ElMessage.error(r.msg || '加载失败')
      list.value = []
      return
    }
    const data = r.data as { list?: Record<string, unknown>[]; total?: number } | Record<string, unknown>[] | undefined
    let rows: Record<string, unknown>[] = []
    if (Array.isArray(data)) rows = data
    else if (data && Array.isArray(data.list)) {
      rows = data.list
      total.value = data.total ?? rows.length
    } else if (data && typeof data === 'object') {
      // overview / dashboard style payloads
      rows = [data as Record<string, unknown>]
      total.value = 1
    }
    list.value = rows
    total.value = total.value || rows.length
    if (rows[0]) columns.value = Object.keys(rows[0]).slice(0, 10)
    else columns.value = ['id', 'status', 'doc_no']
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  formKeys.value.forEach((k) => { form[k] = '' })
  dialogVisible.value = true
}

function openEdit(row: Record<string, unknown>) {
  editingId.value = Number(row.id)
  formKeys.value.forEach((k) => { form[k] = row[k] ?? '' })
  // also copy known keys from row
  Object.keys(row).slice(0, 12).forEach((k) => {
    if (!(k in form)) form[k] = row[k]
    if (!formKeys.value.includes(k) && !['created_at', 'updated_at', 'payload'].includes(k)) {
      formKeys.value.push(k)
    }
  })
  dialogVisible.value = true
}

async function save() {
  const m = meta.value
  if (!m) return
  const body = { ...form }
  let r
  if (editingId.value && m.update) {
    r = await moduleUpdate(m.update, editingId.value, body)
  } else if (m.create) {
    r = await moduleCreate(m.create, body)
  } else {
    ElMessage.warning('该模块只读')
    return
  }
  if (r.code !== 1) {
    ElMessage.error(r.msg)
    return
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  await load()
}

async function onDelete(row: Record<string, unknown>) {
  const m = meta.value
  if (!m?.remove && !m?.detail) {
    ElMessage.warning('无删除接口')
    return
  }
  await ElMessageBox.confirm('确认删除？', '提示')
  const path = m.remove || m.detail || `${m.list}/{id}`
  const r = await moduleDelete(path, Number(row.id))
  if (r.code !== 1) {
    ElMessage.error(r.msg)
    return
  }
  ElMessage.success('已删除')
  await load()
}

async function onAction(row: Record<string, unknown>, action: string) {
  const m = meta.value
  if (!m) return
  const r = await moduleAction(m.list, Number(row.id), action, {})
  if (r.code !== 1) {
    ElMessage.error(r.msg)
    return
  }
  ElMessage.success(`已执行 ${action}`)
  await load()
}

watch([domain, moduleName], () => load(), { immediate: false })
onMounted(load)
watch(() => route.fullPath, load)
</script>

<template>
  <IamView v-if="perm.isIamModule(moduleName)" :module="moduleName" />
  <SupplierView v-else-if="perm.isSupplierModule(moduleName)" />
  <FarmerInboundView
    v-else-if="perm.isFarmerInboundModule(moduleName)"
    :section="moduleName === '农户档案' ? 'farmers' : moduleName === '农户结算' ? 'settlements' : 'weigh'"
  />
  <OnboardView v-else-if="perm.isOnboardModule(moduleName)" />
  <EmployeeView v-else-if="perm.isEmployeeModule(moduleName)" />
  <DeptView v-else-if="perm.isDeptModule(moduleName)" />
  <HrOpsView v-else-if="perm.isHrOpsModule(moduleName)" :module="moduleName" />
  <SystemAdminView
    v-else-if="perm.isSystemAdminModule(moduleName)"
    :module="moduleName"
    :list-path="meta?.list || ''"
    :meta="meta"
  />
  <PayrollView v-else-if="perm.isPayrollModule(moduleName)" :module="moduleName" />
  <div v-else class="panel">
    <h2 class="title">{{ moduleName }}</h2>
    <p class="desc">{{ domain }} → {{ moduleName }} · {{ meta?.list || '无路径' }}</p>
    <div class="toolbar">
      <el-button v-if="meta?.create && !meta.readOnly" type="primary" @click="openCreate">
        {{ meta?.actionOnly ? '执行' : '新建' }}
      </el-button>
      <el-button v-if="meta?.list" @click="load">刷新</el-button>
      <el-tag v-if="meta?.actionOnly" type="warning" size="small">操作型模块</el-tag>
      <span class="spacer" />
      <span class="muted">共 {{ total }} 条</span>
    </div>
    <el-table v-if="!isMobile" v-loading="loading" :data="list" stripe border style="width:100%">
      <el-table-column
        v-for="col in visibleColumns()"
        :key="col"
        :prop="col"
        :label="col"
        min-width="120"
        show-overflow-tooltip
      />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button v-if="meta?.update && !meta.readOnly" link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button
            v-for="act in (meta?.actions || [])"
            :key="act"
            link
            type="success"
            @click="onAction(row, act)"
          >{{ act }}</el-button>
          <el-button v-if="(meta?.remove || meta?.detail) && !meta.readOnly" link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <MobileDataCards
      v-else
      :data="list"
      :loading="loading"
      :columns="cardColumns"
    >
      <template #extra="{ row }">
        <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
      </template>
      <template #actions="{ row }">
        <el-button v-if="meta?.update && !meta.readOnly" link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button
          v-for="act in (meta?.actions || [])"
          :key="act"
          link
          type="success"
          @click="onAction(row, act)"
        >{{ act }}</el-button>
        <el-button v-if="(meta?.remove || meta?.detail) && !meta.readOnly" link type="danger" @click="onDelete(row)">删除</el-button>
      </template>
    </MobileDataCards>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑' : '新建'" :width="isMobile ? '95%' : '520px'">
      <el-form label-width="100px">
        <el-form-item v-for="k in formKeys.filter(x => x !== 'id')" :key="k" :label="k">
          <template v-if="fieldKind(k).kind === 'status'">
            <EnumSelect v-model="(form[k] as string)" :options="STATUS_ACTIVE_OPTIONS" style="width:100%" />
          </template>
          <el-date-picker
            v-else-if="fieldKind(k).kind === 'date'"
            v-model="form[k]"
            type="date"
            value-format="YYYY-MM-DD"
            style="width:100%"
          />
          <el-date-picker
            v-else-if="fieldKind(k).kind === 'month'"
            v-model="form[k]"
            type="month"
            value-format="YYYY-MM"
            style="width:100%"
          />
          <el-date-picker
            v-else-if="fieldKind(k).kind === 'datetime'"
            v-model="form[k]"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width:100%"
          />
          <WarehouseSelect v-else-if="fieldKind(k).ref === 'warehouse'" v-model="(form[k] as number)" style="width:100%" />
          <WorkshopSelect v-else-if="fieldKind(k).ref === 'workshop'" v-model="(form[k] as number)" style="width:100%" />
          <EmployeeSelect v-else-if="fieldKind(k).ref === 'employee'" v-model="(form[k] as number)" style="width:100%" />
          <UserSelect v-else-if="fieldKind(k).ref === 'user'" v-model="(form[k] as number)" style="width:100%" />
          <ProductSelect v-else-if="fieldKind(k).ref === 'product'" v-model="(form[k] as number)" style="width:100%" />
          <CustomerSelect v-else-if="fieldKind(k).ref === 'customer'" v-model="(form[k] as number)" style="width:100%" />
          <SupplierSelect v-else-if="fieldKind(k).ref === 'supplier'" v-model="(form[k] as number)" style="width:100%" />
          <ProcessSelect v-else-if="fieldKind(k).ref === 'process'" v-model="(form[k] as number)" style="width:100%" />
          <SalesOrderSelect v-else-if="fieldKind(k).ref === 'order'" v-model="(form[k] as number)" style="width:100%" />
          <el-input v-else v-model="form[k]" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.panel { background: #fff; border-radius: 8px; padding: 16px; border: 1px solid var(--border, #d5dde3); }
.title { margin: 0 0 4px; font-size: 18px; }
.desc { margin: 0 0 12px; color: #5c6b75; font-size: 13px; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.spacer { flex: 1; }
.muted { color: #5c6b75; font-size: 13px; }
</style>
