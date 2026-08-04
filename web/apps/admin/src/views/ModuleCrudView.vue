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
} from '@erp/shared'
import IamView from './iam/IamView.vue'
import SupplierView from './purchase/SupplierView.vue'
import OnboardView from './hr/OnboardView.vue'
import HrOpsView from './hr/HrOpsView.vue'

const route = useRoute()
const auth = useAuthStore()
const perm = usePermStore()

const domain = computed(() => decodeURIComponent(String(route.params.domain || '')))
const moduleName = computed(() => decodeURIComponent(String(route.params.module || '')))
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

async function load() {
  if (perm.isIamModule(moduleName.value) || perm.isSupplierModule(moduleName.value) || perm.isOnboardModule(moduleName.value) || perm.isHrOpsModule(moduleName.value)) return
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
  <OnboardView v-else-if="perm.isOnboardModule(moduleName)" />
  <HrOpsView v-else-if="perm.isHrOpsModule(moduleName)" :module="moduleName" />
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
    <el-table v-loading="loading" :data="list" stripe border style="width:100%">
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

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑' : '新建'" width="520px">
      <el-form label-width="100px">
        <el-form-item v-for="k in formKeys.filter(x => x !== 'id')" :key="k" :label="k">
          <el-input v-model="form[k]" />
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
