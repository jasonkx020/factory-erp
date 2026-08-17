<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { purchaseApi, productApi, STATUS_ACTIVE_OPTIONS, statusActiveLabel } from '@erp/shared'
import { EnumSelect, ProductSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import { downloadExcel } from '../../utils/exportExcel'

type Row = Record<string, unknown>

const varietyCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'sort_no', label: '排序' },
  { prop: 'code', label: '编码' },
  { prop: 'product_name', label: '默认产品' },
  { prop: 'status_label', label: '状态' },
  { prop: 'remark', label: '备注' },
]

const loading = ref(false)
const list = ref<Row[]>([])
const products = ref<Row[]>([])
const dlg = ref(false)
const editingId = ref(0)
const keyword = ref('')
const statusFilter = ref('')
const form = reactive({
  code: '',
  name: '',
  sort_no: 10,
  status: 'active',
  default_product_id: null as number | null,
  remark: '',
})

function productName(id: unknown) {
  const n = Number(id || 0)
  if (!n) return ''
  const p = products.value.find((x) => Number(x.id) === n)
  return p ? String(p.name || p.code || '') : ''
}

function statusTagType(st: unknown): 'success' | 'info' {
  return String(st) === 'active' ? 'success' : 'info'
}

function mapRow(r: Row): Row {
  return {
    ...r,
    status_label: statusActiveLabel(r.status),
    product_name: productName(r.default_product_id) || '未绑定',
  }
}

const filtered = computed(() => {
  let rows = list.value
  if (statusFilter.value) rows = rows.filter((r) => String(r.status) === statusFilter.value)
  if (keyword.value.trim()) {
    const k = keyword.value.trim().toLowerCase()
    rows = rows.filter(
      (r) =>
        String(r.name || '').toLowerCase().includes(k) ||
        String(r.code || '').toLowerCase().includes(k) ||
        String(r.product_name || '').toLowerCase().includes(k) ||
        String(r.remark || '').toLowerCase().includes(k),
    )
  }
  return rows
})

const summary = computed(() => {
  const all = list.value
  return {
    total: all.length,
    active: all.filter((r) => r.status === 'active').length,
    inactive: all.filter((r) => r.status === 'inactive').length,
  }
})

async function refresh() {
  loading.value = true
  try {
    const [v, p] = await Promise.all([purchaseApi.weighVarieties(), productApi.list()])
    products.value = ((p.data as { list?: Row[] })?.list) || []
    const rows = ((v.data as { list?: Row[] })?.list) || []
    list.value = rows.map(mapRow)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = 0
  form.code = ''
  form.name = ''
  form.sort_no = (list.value.length + 1) * 10
  form.status = 'active'
  form.default_product_id = null
  form.remark = ''
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id)
  form.code = String(row.code || '')
  form.name = String(row.name || '')
  form.sort_no = Number(row.sort_no || 0)
  form.status = String(row.status || 'active')
  form.default_product_id = row.default_product_id ? Number(row.default_product_id) : null
  form.remark = String(row.remark || '')
  dlg.value = true
}

async function save() {
  if (!form.name.trim()) return ElMessage.warning('请填写品种名称')
  const pid = Number(form.default_product_id || 0)
  const body: Record<string, unknown> = {
    code: form.code.trim(),
    name: form.name.trim(),
    sort_no: form.sort_no,
    status: form.status,
    remark: form.remark,
    default_product_id: pid > 0 ? pid : null,
  }
  const res = editingId.value
    ? await purchaseApi.updateWeighVariety(editingId.value, body)
    : await purchaseApi.createWeighVariety(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(editingId.value ? '已更新' : '已创建')
  dlg.value = false
  await refresh()
}

async function removeRow(row: Row) {
  await ElMessageBox.confirm(`停用并删除「${row.name}」？历史过磅单仍保留原品种名称。`, '确认', { type: 'warning' })
  const res = await purchaseApi.removeWeighVariety(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已删除')
  await refresh()
}

async function toggleStatus(row: Row) {
  const next = row.status === 'active' ? 'inactive' : 'active'
  const res = await purchaseApi.updateWeighVariety(Number(row.id), { status: next })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(next === 'active' ? '已启用' : '已停用')
  await refresh()
}

function exportExcel() {
  const rows = filtered.value
  if (!rows.length) return ElMessage.warning('当前筛选无品种可导出')
  const aoa: (string | number)[][] = [
    ['过磅品种'],
    ['排序', '编码', '名称', '默认产品', '状态', '备注'],
  ]
  for (const r of rows) {
    aoa.push([
      Number(r.sort_no) || 0,
      String(r.code || ''),
      String(r.name || ''),
      productName(r.default_product_id) || '',
      statusActiveLabel(r.status),
      String(r.remark || ''),
    ])
  }
  const today = new Date().toISOString().slice(0, 10)
  downloadExcel([{ name: '过磅品种', rows: aoa }], `过磅品种_${today}_${rows.length}`)
  ElMessage.success(`已导出 Excel（${rows.length} 条）`)
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <header class="page-head">
      <div>
        <h2 class="title">过磅品种</h2>
        <p class="desc">配置手机端/管理端过磅收货可选品种（如鲜木薯、半成品、成品入库），可关联默认产品。</p>
      </div>
      <div class="head-meta">
        <span class="meta-pill">筛选 {{ filtered.length }} / 共 {{ summary.total }}</span>
      </div>
    </header>

    <div class="stats">
      <div class="stat"><div class="label">全部</div><div class="value">{{ summary.total }}</div></div>
      <div class="stat ok"><div class="label">启用</div><div class="value">{{ summary.active }}</div></div>
      <div class="stat"><div class="label">停用</div><div class="value">{{ summary.inactive }}</div></div>
    </div>

    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建品种</el-button>
      <el-button type="success" plain :disabled="!filtered.length" @click="exportExcel">导出 Excel</el-button>
      <el-button @click="refresh">刷新</el-button>
      <el-input v-model="keyword" clearable placeholder="名称/编码/产品/备注" style="width:220px" />
      <EnumSelect v-model="statusFilter" :options="STATUS_ACTIVE_OPTIONS" clearable placeholder="状态" style="width:120px" />
    </div>

    <TableOrCards :data="filtered" :loading="loading" :columns="varietyCols" empty-text="暂无过磅品种，请点击「新建品种」">
      <el-table :data="filtered" border stripe class="variety-table" empty-text="暂无过磅品种">
        <el-table-column prop="sort_no" label="排序" width="80" align="center" />
        <el-table-column prop="code" label="编码" width="140">
          <template #default="{ row }">
            <span :class="{ muted: !row.code }">{{ row.code || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="140">
          <template #default="{ row }">
            <div class="name-cell">
              <span class="name">{{ row.name || '—' }}</span>
              <span v-if="row.id" class="id-hint">#{{ row.id }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="默认产品" min-width="160">
          <template #default="{ row }">
            <span :class="{ muted: !row.default_product_id }">{{ row.product_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label || statusActiveLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span :class="{ muted: !row.remark }">{{ row.remark || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link :type="row.status === 'active' ? 'warning' : 'success'" @click="toggleStatus(row)">
              {{ row.status === 'active' ? '停用' : '启用' }}
            </el-button>
            <el-button link type="danger" @click="removeRow(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #field-product_name="{ row }">
        <span :class="{ muted: !row.default_product_id }">{{ row.product_name }}</span>
      </template>
      <template #field-status_label="{ row }">
        <el-tag size="small" :type="statusTagType(row.status)">{{ row.status_label }}</el-tag>
      </template>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button link :type="row.status === 'active' ? 'warning' : 'success'" @click="toggleStatus(row)">
          {{ row.status === 'active' ? '停用' : '启用' }}
        </el-button>
        <el-button link type="danger" @click="removeRow(row)">删除</el-button>
      </template>
    </TableOrCards>

    <el-dialog v-model="dlg" :title="editingId ? '编辑过磅品种' : '新建过磅品种'" width="520px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="编码">
          <el-input v-model="form.code" placeholder="可空，保存时自动生成" maxlength="32" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如：鲜木薯" maxlength="64" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_no" :min="0" :step="10" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="状态">
          <EnumSelect v-model="form.status" :options="STATUS_ACTIVE_OPTIONS" :clearable="false" style="width:100%" />
        </el-form-item>
        <el-form-item label="默认产品">
          <ProductSelect v-model="form.default_product_id" clearable style="width:100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="可选说明" />
        </el-form-item>
        <p class="form-hint">过磅收货选此品种时，可自动带出默认产品，便于入库与溯源。</p>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 8px 4px 24px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; color: #1f2a33; }
.desc { margin: 0; color: #5c6b75; font-size: 13px; line-height: 1.5; max-width: 640px; }
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
.stats { display: grid; grid-template-columns: repeat(3, minmax(100px, 160px)); gap: 10px; margin-bottom: 14px; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat.ok { background: #eef6f1; border-color: #d5eade; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; color: #1f2a33; }
.toolbar { margin-bottom: 14px; display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.name-cell { display: flex; align-items: baseline; gap: 8px; }
.name { font-weight: 500; color: #1f2a33; }
.id-hint { font-size: 12px; color: #98a2a8; }
.muted { color: #98a2a8; }
.variety-table :deep(.el-table__header th) { background: #f6f8fa; color: #4a5a66; font-weight: 600; }
.form-hint { margin: 0 0 0 100px; font-size: 12px; color: #5c6b75; line-height: 1.5; }
@media (max-width: 720px) {
  .stats { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
</style>
