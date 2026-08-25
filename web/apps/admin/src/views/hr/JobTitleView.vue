<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hrApi, EMP_TYPE_OPTIONS, STATUS_ACTIVE_OPTIONS, formOptionLabel } from '@erp/shared'
import { EnumSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const loading = ref(false)
const list = ref<Row[]>([])
const filterEmpType = ref('')
const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({
  code: '',
  name: '',
  emp_type: '',
  sort_no: 0,
  status: 'active',
})

const cardCols: MobileCardColumn[] = [
  { prop: 'name', label: '岗位名称', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'emp_type', label: '用工类型' },
  { prop: 'sort_no', label: '排序' },
  { prop: 'status', label: '状态' },
]

const filtered = computed(() => {
  let rows = list.value
  if (filterEmpType.value) {
    rows = rows.filter((r) => {
      const et = String(r.emp_type || '')
      return et === '' || et === filterEmpType.value
    })
  }
  return [...rows].sort((a, b) => Number(a.sort_no || 0) - Number(b.sort_no || 0))
})

const totalLabel = computed(() => {
  const n = filtered.value.length
  const all = list.value.length
  return filterEmpType.value ? `显示 ${n} / 共 ${all} 个岗位` : `共 ${all} 个岗位`
})

function empTypeLabel(v: unknown) {
  return formOptionLabel(EMP_TYPE_OPTIONS, String(v || '')) || (v ? String(v) : '通用')
}

async function refresh() {
  loading.value = true
  try {
    const res = await hrApi.jobTitles(undefined, 'all')
    if (res.code !== 1) {
      ElMessage.error(res.msg || '加载失败')
      return
    }
    const data = res.data as { list?: Row[] } | Row[] | undefined
    list.value = Array.isArray(data) ? data : data?.list || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.code = ''
  form.name = ''
  form.emp_type = ''
  form.sort_no = 0
  form.status = 'active'
  dialogVisible.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id)
  form.code = String(row.code || '')
  form.name = String(row.name || '')
  form.emp_type = String(row.emp_type || '')
  form.sort_no = Number(row.sort_no) || 0
  form.status = String(row.status || 'active')
  dialogVisible.value = true
}

async function save() {
  if (!form.name.trim()) {
    ElMessage.warning('请填写岗位名称')
    return
  }
  const body: Record<string, unknown> = {
    name: form.name.trim(),
    emp_type: form.emp_type,
    sort_no: form.sort_no,
    status: form.status,
  }
  if (form.code.trim()) body.code = form.code.trim()
  const res = editingId.value
    ? await hrApi.updateJobTitle(editingId.value, body)
    : await hrApi.createJobTitle(body)
  if (res.code !== 1) {
    ElMessage.error(res.msg || '保存失败')
    return
  }
  ElMessage.success('已保存')
  dialogVisible.value = false
  await refresh()
}

async function deactivate(row: Row) {
  const id = Number(row.id)
  if (!id) return
  await ElMessageBox.confirm(`停用岗位「${row.name}」？已引用员工的岗位名称仍保留。`, '确认', { type: 'warning' })
  const res = await hrApi.removeJobTitle(id)
  if (res.code !== 1) {
    ElMessage.error(res.msg || '操作失败')
    return
  }
  ElMessage.success('已停用')
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="panel">
    <h2 class="title">岗位管理</h2>
    <p class="desc">维护岗位主数据；员工档案与入职登记从此处下拉选择，可按用工类型筛选。</p>
    <div class="toolbar">
      <EnumSelect
        v-model="filterEmpType"
        :options="[{ value: '', label: '全部类型' }, ...EMP_TYPE_OPTIONS]"
        placeholder="筛选用工类型"
        style="width:160px"
      />
      <span class="total">{{ totalLabel }}</span>
      <el-button type="primary" @click="openCreate">新建岗位</el-button>
      <el-button :loading="loading" @click="refresh">刷新</el-button>
    </div>
    <TableOrCards :data="filtered" :columns="cardCols" :loading="loading" empty-text="暂无岗位">
      <el-table v-loading="loading" :data="filtered" border stripe empty-text="暂无岗位" size="small" max-height="560">
        <el-table-column prop="name" label="岗位名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="code" label="编码" width="140" show-overflow-tooltip />
        <el-table-column label="用工类型" width="110">
          <template #default="{ row }">{{ empTypeLabel(row.emp_type) }}</template>
        </el-table-column>
        <el-table-column prop="sort_no" label="排序" width="72" align="center" />
        <el-table-column label="状态" width="88" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="row.status === 'active'" link type="danger" @click="deactivate(row)">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </TableOrCards>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑岗位' : '新建岗位'" width="480px" destroy-on-close>
      <el-form label-width="96px">
        <el-form-item label="岗位名称" required>
          <el-input v-model="form.name" placeholder="如 去皮工" maxlength="40" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="form.code" placeholder="留空自动生成" maxlength="48" />
        </el-form-item>
        <el-form-item label="用工类型">
          <EnumSelect
            v-model="form.emp_type"
            :options="[{ value: '', label: '通用（全部类型可见）' }, ...EMP_TYPE_OPTIONS]"
            :clearable="false"
            style="width:100%"
          />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_no" :min="0" :max="9999" />
        </el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <EnumSelect v-model="form.status" :options="STATUS_ACTIVE_OPTIONS" :clearable="false" style="width:100%" />
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
.panel { padding: 8px 4px; }
.title { margin: 0 0 8px; font-size: 20px; }
.desc { margin: 0 0 16px; color: var(--el-text-color-secondary); font-size: 13px; }
.toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; align-items: center; }
.total { font-size: 13px; color: var(--el-text-color-secondary); margin-right: auto; }
</style>
