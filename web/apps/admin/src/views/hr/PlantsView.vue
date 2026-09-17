<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hrApi } from '@erp/shared'

type Row = Record<string, unknown>

const loading = ref(false)
const list = ref<Row[]>([])
const dialog = ref(false)
const form = reactive({
  id: 0,
  code: '',
  name: '',
  status: 'active',
  is_default: false,
  remark: '',
})

async function load() {
  loading.value = true
  try {
    const res = await hrApi.plants()
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { id: 0, code: '', name: '', status: 'active', is_default: false, remark: '' })
  dialog.value = true
}

function openEdit(row: Row) {
  Object.assign(form, {
    id: Number(row.id),
    code: String(row.code || ''),
    name: String(row.name || ''),
    status: String(row.status || 'active'),
    is_default: !!row.is_default,
    remark: String(row.remark || ''),
  })
  dialog.value = true
}

async function save() {
  if (!form.name.trim()) return ElMessage.warning('请填写厂区名称')
  const body = {
    code: form.code,
    name: form.name,
    status: form.status,
    is_default: form.is_default,
    remark: form.remark,
  }
  const res = form.id
    ? await hrApi.updatePlant(form.id, body)
    : await hrApi.createPlant(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已保存')
  dialog.value = false
  await load()
}

async function remove(row: Row) {
  try {
    await ElMessageBox.confirm(`删除厂区「${row.name}」？`, '确认', { type: 'warning' })
  } catch {
    return
  }
  const res = await hrApi.removePlant(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>厂区管理</h2>
      <el-button type="primary" @click="openCreate">新建厂区</el-button>
    </div>
    <p class="hint">同一组织下可配置多个厂区；仓库/工艺/班次可绑定厂区，用户通过厂区 scope 隔离现场数据。</p>
    <el-table :data="list" border stripe>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="code" label="编码" width="140" />
      <el-table-column label="默认" width="80">
        <template #default="{ row }">{{ row.is_default ? '是' : '' }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="90" />
      <el-table-column prop="remark" label="备注" min-width="160" />
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog" :title="form.id ? '编辑厂区' : '新建厂区'" width="480px">
      <el-form label-width="80px">
        <el-form-item label="编码"><el-input v-model="form.code" placeholder="可空自动生成" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="默认厂区"><el-switch v-model="form.is_default" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
h2 { margin: 0; font-size: 18px; }
.hint { color: #64748b; font-size: 13px; margin: 0 0 12px; }
</style>
