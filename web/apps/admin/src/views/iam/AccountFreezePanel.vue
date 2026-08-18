<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { iamApi } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const loading = ref(false)
const users = ref<Row[]>([])
const sessions = ref<Row[]>([])

const userCols: MobileCardColumn[] = [
  { prop: 'login_name', label: '登录名', primary: true },
  { prop: 'name', label: '姓名' },
  { prop: 'status', label: '状态' },
]
const sessionCols: MobileCardColumn[] = [
  { prop: 'id', label: '会话ID', primary: true },
  { prop: 'user_id', label: '用户' },
  { prop: 'client_type', label: '端' },
  { prop: 'created_at', label: '创建时间' },
]

function listOf(env: { code?: number; data?: unknown } | undefined): Row[] {
  if (!env || env.code !== 1) return []
  const d = env.data
  if (Array.isArray(d)) return d as Row[]
  if (d && typeof d === 'object' && Array.isArray((d as { list?: unknown }).list)) {
    return (d as { list: Row[] }).list
  }
  return []
}

async function load() {
  loading.value = true
  try {
    const [u, s] = await Promise.all([iamApi.users(), iamApi.sessions()])
    users.value = listOf(u)
    sessions.value = listOf(s)
  } finally {
    loading.value = false
  }
}

async function freeze(id: number, frozen: boolean) {
  const res = frozen ? await iamApi.unfreeze(id) : await iamApi.freeze(id, { reason: '账户冻结' })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(frozen ? '已解冻' : '已冻结并踢下线')
  await load()
}

async function revokeSession(id: number) {
  const res = await iamApi.revokeSession(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('会话已撤销')
  await load()
}

onMounted(load)

defineExpose({ load })
</script>

<template>
  <div v-loading="loading" class="freeze-panel">
    <div class="row">
      <el-button @click="load">刷新</el-button>
    </div>
    <h3 class="sub">账户冻结</h3>
    <TableOrCards :data="users" :columns="userCols">
      <el-table :data="users" border stripe height="280">
        <el-table-column prop="login_name" label="登录名" />
        <el-table-column prop="name" label="姓名" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button
              link
              :type="row.status === 'frozen' ? 'success' : 'danger'"
              @click="freeze(Number(row.id), row.status === 'frozen')"
            >{{ row.status === 'frozen' ? '解冻' : '冻结' }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #actions="{ row }">
        <el-button
          link
          :type="row.status === 'frozen' ? 'success' : 'danger'"
          @click="freeze(Number(row.id), row.status === 'frozen')"
        >{{ row.status === 'frozen' ? '解冻' : '冻结' }}</el-button>
      </template>
    </TableOrCards>
    <h3 class="sub">在线会话</h3>
    <TableOrCards :data="sessions" :columns="sessionCols">
      <el-table :data="sessions" border size="small" height="220">
        <el-table-column prop="id" label="会话ID" width="90" />
        <el-table-column prop="user_id" label="用户" width="90" />
        <el-table-column prop="client_type" label="端" width="100" />
        <el-table-column prop="created_at" label="创建时间" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="danger" @click="revokeSession(Number(row.id))">踢下线</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #actions="{ row }">
        <el-button link type="danger" @click="revokeSession(Number(row.id))">踢下线</el-button>
      </template>
    </TableOrCards>
  </div>
</template>

<style scoped>
.row { display: flex; gap: 8px; margin-bottom: 8px; }
.sub { margin: 16px 0 8px; font-size: 15px; }
</style>
