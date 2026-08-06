<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { purchaseApi, productApi } from '@erp/shared'
import { ProductSelect } from '../../components/select'

type Row = Record<string, unknown>

const loading = ref(false)
const list = ref<Row[]>([])
const products = ref<Row[]>([])
const dlg = ref(false)
const editingId = ref(0)
const form = reactive({
  code: '',
  name: '',
  sort_no: 10,
  status: 'active',
  default_product_id: null as number | null,
  remark: '',
})

async function refresh() {
  loading.value = true
  try {
    const [v, p] = await Promise.all([purchaseApi.weighVarieties(), productApi.list()])
    list.value = ((v.data as { list?: Row[] })?.list) || []
    products.value = ((p.data as { list?: Row[] })?.list) || []
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

function productName(id: unknown) {
  const n = Number(id || 0)
  if (!n) return '-'
  const p = products.value.find((x) => Number(x.id) === n)
  return p ? String(p.name || p.code || n) : String(n)
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>过磅品种</h2>
      <p class="hint">配置手机端/管理端过磅收货可选品种（如鲜木薯、半成品、成品入库），可关联默认产品。</p>
    </div>
    <el-card>
      <div class="toolbar">
        <el-button type="primary" @click="openCreate">新建品种</el-button>
        <el-button @click="refresh">刷新</el-button>
      </div>
      <el-table :data="list" size="small" border>
        <el-table-column prop="sort_no" label="排序" width="70" />
        <el-table-column prop="code" label="编码" width="140" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column label="默认产品" min-width="140">
          <template #default="{ row }">{{ productName(row.default_product_id) }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link @click="toggleStatus(row)">{{ row.status === 'active' ? '停用' : '启用' }}</el-button>
            <el-button link type="danger" @click="removeRow(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dlg" :title="editingId ? '编辑过磅品种' : '新建过磅品种'" width="480px">
      <el-form label-width="100px">
        <el-form-item label="编码"><el-input v-model="form.code" placeholder="可空，自动生成" /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" placeholder="如：鲜木薯" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort_no" :min="0" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="默认产品">
          <ProductSelect v-model="form.default_product_id" clearable />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
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
.head { margin-bottom: 12px; }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.hint { margin: 0; color: #64748b; font-size: 13px; }
.toolbar { margin-bottom: 12px; display: flex; gap: 8px; }
</style>
