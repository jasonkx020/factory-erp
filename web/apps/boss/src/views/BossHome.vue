<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, reportApi, productionApi, approvalApi, portalHomeUrl } from '@erp/shared'

const auth = useAuthStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const view = ref<'boss' | 'board' | 'live'>('boss')
const tasks = ref<Record<string, unknown>[]>([])
const approvals = ref(0)
const processes = ref<Record<string, unknown>[]>([])
const bossData = ref<Record<string, unknown>>({})

async function load() {
  if (!auth.isLoggedIn) await auth.login('admin', 'admin123', 'boss')
  const [t, a, p, b] = await Promise.all([
    productionApi.listTasks(),
    approvalApi.tasks(),
    productionApi.processes(),
    reportApi.boss(),
  ])
  tasks.value = ((t.data as { list?: Record<string, unknown>[] })?.list) || []
  approvals.value = (((a.data as { list?: unknown[] })?.list) || []).length
  processes.value = ((p.data as { list?: Record<string, unknown>[] })?.list) || []
  bossData.value = (b.data as Record<string, unknown>) || {}
}

onMounted(load)
</script>

<template>
  <div class="dash">
    <header>
      <a :href="portalUrl" style="color:#7ad4c6;font-size:13px">← 入口</a>
      <h1>老板驾驶舱</h1>
      <div class="tabs">
        <button :class="{ active: view==='boss' }" @click="view='boss'">老板驾驶舱</button>
        <button :class="{ active: view==='board' }" @click="view='board'">生产看板</button>
        <button :class="{ active: view==='live' }" @click="view='live'">生产实况</button>
      </div>
      <el-button link type="danger" @click="auth.logout(); router.replace('/login')">退出</el-button>
    </header>
    <div v-if="view==='boss'" class="grid">
      <div class="kpi"><div class="l">生产任务</div><div class="v">{{ tasks.length }}</div></div>
      <div class="kpi"><div class="l">待办审批</div><div class="v">{{ approvals }}</div></div>
      <div class="kpi"><div class="l">工序数</div><div class="v">{{ processes.length }}</div></div>
      <div class="kpi"><div class="l">看板数据</div><div class="v">{{ Object.keys(bossData).length }}</div></div>
      <div class="box wide">
        <h3>任务一览</h3>
        <el-table :data="tasks.slice(0,8)" size="small" dark>
          <el-table-column prop="doc_no" label="单号" />
          <el-table-column prop="status" label="状态" />
        </el-table>
      </div>
    </div>
    <div v-else-if="view==='board'" class="box">
      <h3>生产看板</h3>
      <el-table :data="tasks" border>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="doc_no" label="任务" />
        <el-table-column prop="status" label="进度状态" />
        <el-table-column prop="created_at" label="下达时间" />
      </el-table>
    </div>
    <div v-else class="box">
      <h3>生产实况 · 工序节点</h3>
      <div class="flow">
        <span v-for="p in processes" :key="String(p.id)" :class="{ on: p.is_handover_point }">{{ p.name }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dash { min-height: 100vh; background: #0b1418; color: #e8eef1; padding: 12px 16px; }
header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
h1 { margin: 0; font-size: 18px; color: #7ad4c6; }
.tabs button { margin-right: 6px; background: #1a2b34; border: 0; color: #9fb0ba; padding: 6px 12px; border-radius: 4px; cursor: pointer; }
.tabs button.active { background: #0d7a6f; color: #fff; }
.grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.kpi, .box { background: #16252d; border-radius: 8px; padding: 14px; }
.kpi .l { font-size: 12px; color: #9fb0ba; }
.kpi .v { font-size: 28px; color: #7ad4c6; font-weight: 600; }
.wide { grid-column: span 4; }
.flow { display: flex; flex-wrap: wrap; gap: 8px; }
.flow span { background: #1a2b34; padding: 6px 10px; border-radius: 4px; }
.flow span.on { outline: 2px solid #0d7a6f; }
</style>
