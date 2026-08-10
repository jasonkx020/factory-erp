<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { systemApi } from '@erp/shared'
import { ElMessage } from 'element-plus'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

const logCols: MobileCardColumn[] = [
  { prop: 'id', label: 'ID', primary: true },
  { prop: 'user_id', label: '用户' },
  { prop: 'module', label: '模块' },
  { prop: 'action', label: '动作' },
  { prop: 'trace_id', label: 'Trace' },
  { prop: 'ip', label: 'IP' },
  { prop: 'created_at', label: '时间' },
]

const logs = ref<Record<string, unknown>[]>([])
const detail = ref<Record<string, unknown> | null>(null)
const drawer = ref(false)

async function load() {
  const res = await systemApi.logs()
  if (res.code !== 1) return ElMessage.error(res.msg)
  logs.value = ((res.data as { list?: Record<string, unknown>[] })?.list) || []
}

function show(row: Record<string, unknown>) {
  detail.value = row
  drawer.value = true
}

onMounted(load)
</script>

<template>
  <div class="page">
    <h2>操作审计日志</h2>
    <p class="sub">全量非 GET 请求写入 sys_operation_log，含用户 / 路径 / 请求体 / IP / trace_id。</p>
    <el-button size="small" @click="load" style="margin-bottom:8px">刷新</el-button>
    <TableOrCards :data="logs" :columns="logCols">
      <el-table :data="logs" border size="small" height="420" @row-click="show">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="user_id" label="用户" width="80" />
        <el-table-column prop="module" label="模块" width="120" />
        <el-table-column prop="action" label="动作" width="140" />
        <el-table-column prop="trace_id" label="Trace" min-width="160" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="120" />
        <el-table-column prop="created_at" label="时间" width="170" />
      </el-table>
      <template #actions="{ row }">
        <el-button link type="primary" @click="show(row)">详情</el-button>
      </template>
    </TableOrCards>
    <el-drawer v-model="drawer" title="详情" size="40%">
      <pre v-if="detail" style="font-size:12px;white-space:pre-wrap">{{ JSON.stringify(detail, null, 2) }}</pre>
    </el-drawer>
  </div>
</template>

<style scoped>
.page { padding: 4px; }
.sub { color: #667; font-size: 13px; }
</style>
