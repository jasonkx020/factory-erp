<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, reportApi, productionApi, approvalApi, purchaseApi, financeApi, portalHomeUrl } from '@erp/shared'

const auth = useAuthStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const view = ref<'boss' | 'board' | 'live'>('boss')
const tasks = ref<Record<string, unknown>[]>([])
const approvals = ref(0)
const processes = ref<Record<string, unknown>[]>([])
const bossData = ref<Record<string, unknown>>({})
const farmerPendingAmt = ref(0)
const fundBalance = ref(0)

function money(v: unknown) {
  const n = Number(v)
  if (!Number.isFinite(n)) return '-'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 0 })
}

async function load() {
  if (!auth.isLoggedIn) await auth.login('admin', 'admin123', 'boss')
  const [t, a, p, b, fs, fa] = await Promise.all([
    productionApi.listTasks(),
    approvalApi.tasks(),
    productionApi.processes(),
    reportApi.boss(),
    purchaseApi.farmerSettlements().catch(() => ({ code: 0, data: { list: [] } })),
    financeApi.fundAccounts().catch(() => ({ code: 0, data: { list: [] } })),
  ])
  tasks.value = ((t.data as { list?: Record<string, unknown>[] })?.list) || []
  approvals.value = (((a.data as { list?: unknown[] })?.list) || []).length
  processes.value = ((p.data as { list?: Record<string, unknown>[] })?.list) || []
  bossData.value = (b.data as Record<string, unknown>) || {}
  const settles = ((fs.data as { list?: Record<string, unknown>[] })?.list) || []
  farmerPendingAmt.value = settles
    .filter((r) => {
      const st = String(r.status || '')
      return st !== 'settle_paid' && st !== 'paid' && st !== 'void'
    })
    .reduce((s, r) => s + (Number(r.amount) || 0), 0)
  const funds = ((fa.data as { list?: Record<string, unknown>[] })?.list) || []
  fundBalance.value = funds.reduce((s, r) => s + (Number(r.balance) || 0), 0)
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
      <div class="kpi warn"><div class="l">农户待付</div><div class="v">{{ money(farmerPendingAmt) }}</div></div>
      <div class="kpi ok"><div class="l">资金余额</div><div class="v">{{ money(fundBalance) }}</div></div>
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
.dash {
  min-height: 100vh;
  background:
    linear-gradient(rgba(31, 122, 77, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(31, 122, 77, 0.04) 1px, transparent 1px),
    #0c1a14;
  background-size: 28px 28px, 28px 28px, auto;
  color: #e8f5ee;
  padding: 12px 16px;
}
header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
h1 { margin: 0; font-size: 18px; color: #7dcea0; letter-spacing: 0.02em; }
.tabs button { margin-right: 6px; background: #14352a; border: 1px solid rgba(61, 155, 106, 0.2); color: #a8c4b4; padding: 6px 12px; border-radius: 4px; cursor: pointer; }
.tabs button.active { background: var(--accent, #1f7a4d); color: #fff; border-color: transparent; }
.grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.kpi, .box {
  background: #12291f;
  border-radius: 8px;
  padding: 14px;
  border: 1px solid rgba(61, 155, 106, 0.18);
  position: relative;
  overflow: hidden;
}
.kpi::before {
  content: '';
  position: absolute;
  left: 0; top: 0; bottom: 0;
  width: 3px;
  background: var(--accent-leaf, #3d9b6a);
}
.kpi .l { font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: #a8c4b4; }
.kpi .v { font-size: 28px; color: #7dcea0; font-weight: 700; font-variant-numeric: tabular-nums; font-family: var(--factory-mono, monospace); }
.kpi.warn::before { background: var(--factory-warn, #c47a12); }
.kpi.warn .v { color: #e8b86d; }
.kpi.ok::before { background: #3d9b6a; }
.wide { grid-column: span 4; }
.flow { display: flex; flex-wrap: wrap; gap: 0; }
.flow span {
  background: #14352a;
  padding: 8px 12px;
  border: 1px solid rgba(61, 155, 106, 0.2);
  border-right-width: 0;
  font-size: 13px;
}
.flow span:first-child { border-radius: 6px 0 0 6px; }
.flow span:last-child { border-right-width: 1px; border-radius: 0 6px 6px 0; }
.flow span.on {
  outline: none;
  background: rgba(31, 122, 77, 0.35);
  border-color: var(--accent-leaf, #3d9b6a);
  color: #fff;
  font-weight: 600;
}
</style>
