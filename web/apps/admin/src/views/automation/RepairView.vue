<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { systemApi, REPAIR_ACTION_OPTIONS, REPAIR_TARGET_TYPE_OPTIONS } from '@erp/shared'
import { ElMessage } from 'element-plus'
import { EnumSelect } from '../../components/select'

const list = ref<Record<string, unknown>[]>([])
const form = reactive({
  target_type: 'prod_task',
  target_id: 0,
  reason: '',
  action: 'retry_flow',
})

async function load() {
  const res = await systemApi.listRepairs()
  if (res.code !== 1) return ElMessage.error(res.msg)
  list.value = ((res.data as { list?: Record<string, unknown>[] })?.list) || []
}

async function create() {
  if (!form.reason.trim()) return ElMessage.warning('必须填写 reason')
  const res = await systemApi.createRepair({
    target_type: form.target_type,
    target_id: form.target_id,
    reason: form.reason,
    action: form.action,
    status: 'draft',
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已创建修复单')
  form.reason = ''
  await load()
}

async function apply(id: number) {
  const res = await systemApi.applyRepair(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已执行修复')
  await load()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <h2>数据修复单</h2>
    <p class="sub">强制填写 reason；仅 sys_admin 可 apply；执行动作本身也会写入审计日志。</p>
    <el-card shadow="never" style="margin-bottom:12px">
      <el-form label-width="100px" style="max-width:560px">
        <el-form-item label="目标类型">
          <EnumSelect v-model="form.target_type" :options="REPAIR_TARGET_TYPE_OPTIONS" :clearable="false" style="width:100%" />
        </el-form-item>
        <el-form-item label="目标 ID"><el-input-number v-model="form.target_id" :min="0" /></el-form-item>
        <el-form-item label="原因 reason" required><el-input v-model="form.reason" type="textarea" /></el-form-item>
        <el-form-item label="动作">
          <EnumSelect v-model="form.action" :options="REPAIR_ACTION_OPTIONS" :clearable="false" style="width:100%" />
        </el-form-item>
        <el-form-item><el-button type="primary" @click="create">创建修复单</el-button></el-form-item>
      </el-form>
    </el-card>
    <el-table :data="list" border size="small">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="target_type" label="类型" />
      <el-table-column prop="target_id" label="目标" width="80" />
      <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button v-if="row.status !== 'applied'" link type="danger" @click="apply(Number(row.id))">执行</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.page { padding: 4px; }
.sub { color: #667; font-size: 13px; }
</style>
