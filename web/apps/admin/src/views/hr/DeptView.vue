<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hrApi } from '@erp/shared'

type Row = Record<string, unknown>

const loading = ref(false)
const list = ref<Row[]>([])
const dlg = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({ code: '', name: '', status: 'active' })

const errLabel: Record<string, string> = {
  NAME_REQUIRED: '请填写部门名称',
  CODE_DUPLICATE: '部门编码已存在',
  DEPT_IN_USE: '仍有在职员工属于该部门，无法删除',
  NOT_FOUND: '部门不存在',
}

async function load() {
  loading.value = true
  try {
    const res = await hrApi.departments()
    if (res.code !== 1) {
      ElMessage.error(res.msg || '加载失败')
      list.value = []
      return
    }
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, { code: '', name: '', status: 'active' })
  dlg.value = true
}

function openEdit(row: Row) {
  editingId.value = Number(row.id)
  Object.assign(form, {
    code: String(row.code || ''),
    name: String(row.name || ''),
    status: String(row.status || 'active'),
  })
  dlg.value = true
}

async function save() {
  if (!form.name.trim()) return ElMessage.warning('请填写部门名称')
  let res
  if (editingId.value) {
    res = await hrApi.updateDepartment(editingId.value, {
      name: form.name.trim(),
      status: form.status,
    })
  } else {
    res = await hrApi.createDepartment({
      code: form.code.trim(),
      name: form.name.trim(),
      status: form.status,
    })
  }
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '保存失败')
  ElMessage.success(editingId.value ? '已保存' : '部门已创建')
  dlg.value = false
  await load()
}

async function deactivate(row: Row) {
  await ElMessageBox.confirm(`将部门「${row.name}」设为停用？`, '提示')
  const res = await hrApi.updateDepartment(Number(row.id), { status: 'inactive' })
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '失败')
  ElMessage.success('已停用')
  await load()
}

async function remove(row: Row) {
  await ElMessageBox.confirm(`删除部门「${row.name}」？有在职员工占用时将无法删除。`, '删除确认', {
    type: 'warning',
  })
  const res = await hrApi.removeDepartment(Number(row.id))
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '删除失败')
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="dept">
    <h2 class="title">部门管理</h2>
    <p class="desc">维护组织部门主数据；员工建档可不选部门。编码留空则系统自动生成。</p>

    <div class="row">
      <el-button type="primary" @click="openCreate">新建部门</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <el-table :data="list" border stripe>
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status === 'active' ? '启用' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button v-if="row.status === 'active'" link @click="deactivate(row)">停用</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dlg" :title="editingId ? '编辑部门' : '新建部门'" width="440px">
      <el-form label-width="80px">
        <el-form-item label="编码">
          <el-input
            v-model="form.code"
            :disabled="!!editingId"
            :placeholder="editingId ? '' : '可选，空则自动生成'"
          />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="64" placeholder="部门名称" />
        </el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">{{ editingId ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.dept { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
</style>
