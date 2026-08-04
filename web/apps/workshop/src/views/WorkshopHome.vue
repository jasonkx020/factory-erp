<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, productionApi, inventoryApi, portalHomeUrl } from '@erp/shared'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const pane = ref('scan')
const tasks = ref<Record<string, unknown>[]>([])
const dispatches = ref<Record<string, unknown>[]>([])
const balances = ref<Record<string, unknown>[]>([])
const processes = ref<Record<string, unknown>[]>([])
const flowEvents = ref<Record<string, unknown>[]>([])
const scan = reactive({ badge_code: 'EMP0301', box_code: 'BX-RAW-DEMO', net_weight: 100 })
const preview = ref<Record<string, unknown> | null>(null)
const last = ref<Record<string, unknown> | null>(null)

async function load() {
  if (!auth.isLoggedIn) {
    await auth.login('admin', 'admin123', 'pad')
  } else {
    await auth.fetchMe()
  }
  const [t, d, b, p, f] = await Promise.all([
    productionApi.listTasks(),
    productionApi.listDispatches(),
    inventoryApi.balances(),
    productionApi.processes(),
    productionApi.flowEvents(),
  ])
  tasks.value = ((t.data as { list?: Record<string, unknown>[] })?.list) || []
  dispatches.value = ((d.data as { list?: Record<string, unknown>[] })?.list) || []
  balances.value = ((b.data as { list?: Record<string, unknown>[] })?.list) || []
  processes.value = ((p.data as { list?: Record<string, unknown>[] })?.list) || []
  flowEvents.value = ((f.data as { list?: Record<string, unknown>[] })?.list) || []
}

async function doResolve() {
  const res = await productionApi.scanResolve({ ...scan })
  if (res.code !== 1) return ElMessage.error(res.msg)
  preview.value = (res.data as Record<string, unknown>) || null
  ElMessage.success('已解析')
}

async function doScan() {
  const res = await productionApi.scan({ ...scan })
  if (res.code !== 1) return ElMessage.error(res.msg)
  last.value = (res.data as Record<string, unknown>) || null
  const next = (last.value?.next as Record<string, unknown>) || {}
  ElMessage.success(`报工成功 工钱¥${last.value?.wage_amount ?? 0}`)
  if (next.new_box_code) scan.box_code = String(next.new_box_code)
  await load()
}

async function retryFlow(id: number) {
  const res = await productionApi.retryFlow(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已重试')
  await load()
}

async function logout() {
  auth.logout()
  router.replace('/login')
}

onMounted(load)
</script>

<template>
  <div class="pad">
    <header class="top">
      <a class="portal-link" :href="portalUrl">← 入口</a>
      <h1>车间工作台</h1>
      <nav>
        <button v-for="p in [
          ['scan','扫码工作台'],['flow','流转监控'],['overview','任务'],['dispatch','派工'],
          ['progress','工序'],['stock','库存']
        ]" :key="p[0]" :class="{ active: pane === p[0] }" @click="pane = p[0]">{{ p[1] }}</button>
      </nav>
      <el-button link type="danger" @click="logout">退出</el-button>
    </header>
    <div class="flow">
      <span>原料入库</span><span>清洗</span><span class="on">去皮·计件</span>
      <span>收货卡点</span><span>切断</span><span>去芯</span><span>半成品入库</span>
      <span>切块</span><span>装袋</span><span>成品入库销售</span>
    </div>
    <div class="body">
      <section v-if="pane === 'scan'" class="panel scan-panel">
        <h3>双扫报工 · 工牌 + 箱码 + 净重</h3>
        <div class="scan-grid">
          <el-form label-width="90px" size="large">
            <el-form-item label="工牌码"><el-input v-model="scan.badge_code" placeholder="EMP0301" /></el-form-item>
            <el-form-item label="箱码"><el-input v-model="scan.box_code" placeholder="BX-RAW-DEMO" /></el-form-item>
            <el-form-item label="净重(kg)"><el-input-number v-model="scan.net_weight" :min="0.01" :step="1" /></el-form-item>
            <el-form-item>
              <el-button @click="doResolve">预解析</el-button>
              <el-button type="primary" @click="doScan">确认报工并流转</el-button>
              <el-button @click="load">刷新</el-button>
            </el-form-item>
          </el-form>
          <div class="side">
            <el-descriptions v-if="preview" title="预解析" :column="1" border size="small">
              <el-descriptions-item label="工人">{{ preview.worker_name }}</el-descriptions-item>
              <el-descriptions-item label="工序">{{ preview.step_name || preview.process_id }}</el-descriptions-item>
              <el-descriptions-item label="派工">{{ preview.dispatch_id || '-' }}</el-descriptions-item>
            </el-descriptions>
            <el-descriptions v-if="last" title="上次结果" :column="1" border size="small" style="margin-top:12px">
              <el-descriptions-item label="工钱">¥{{ last.wage_amount || 0 }}</el-descriptions-item>
              <el-descriptions-item label="下一步">{{ (last.next as any)?.next_step || ((last.next as any)?.finished ? '产线完成' : '-') }}</el-descriptions-item>
              <el-descriptions-item label="新箱码">{{ (last.next as any)?.new_box_code || '-' }}</el-descriptions-item>
            </el-descriptions>
          </div>
        </div>
      </section>
      <section v-else-if="pane === 'flow'" class="panel">
        <h3>流转事件监控 · 失败可重试</h3>
        <el-table :data="flowEvents" border size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="trigger" label="触发" width="120" />
          <el-table-column prop="from_step" label="从" width="80" />
          <el-table-column prop="to_step" label="到" width="80" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="trace_id" label="Trace" min-width="140" show-overflow-tooltip />
          <el-table-column prop="error" label="错误" min-width="120" show-overflow-tooltip />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status === 'failed'" link type="primary" @click="retryFlow(Number(row.id))">重试</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>
      <section v-else-if="pane === 'overview'" class="panel">
        <h3>今日任务 · 生产任务单</h3>
        <el-table :data="tasks" border size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="doc_no" label="单号" />
          <el-table-column prop="status" label="状态" />
          <el-table-column prop="created_at" label="创建时间" />
        </el-table>
      </section>
      <section v-else-if="pane === 'dispatch'" class="panel">
        <h3>生产派工</h3>
        <el-table :data="dispatches" border size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="doc_no" label="单号" />
          <el-table-column prop="task_id" label="任务" />
          <el-table-column prop="status" label="状态" />
        </el-table>
      </section>
      <section v-else-if="pane === 'progress'" class="panel">
        <h3>进度跟踪 · 工序</h3>
        <el-table :data="processes" border size="small">
          <el-table-column prop="code" label="编码" />
          <el-table-column prop="name" label="工序" />
          <el-table-column prop="is_piecework" label="计件" />
          <el-table-column prop="is_handover_point" label="收货卡点" />
        </el-table>
      </section>
      <section v-else class="panel">
        <h3>库存 · 授权仓余额</h3>
        <el-table :data="balances" border size="small">
          <el-table-column prop="warehouse_name" label="仓" />
          <el-table-column prop="product_code" label="料号" />
          <el-table-column prop="product_name" label="名称" />
          <el-table-column prop="qty" label="数量" />
        </el-table>
      </section>
    </div>
  </div>
</template>

<style scoped>
.pad { min-height: 100vh; background: #0f1c22; color: #e8eef1; }
.top { display: flex; align-items: center; gap: 12px; padding: 10px 16px; background: #0b1418; }
.portal-link { color: #9fd4cb; font-size: 13px; white-space: nowrap; }
.top h1 { font-size: 16px; margin: 0; color: #9fd4cb; }
.top nav { flex: 1; display: flex; gap: 6px; flex-wrap: wrap; }
.top button {
  background: #1a2b34; border: 0; color: #9fb0ba; padding: 6px 10px; border-radius: 4px; cursor: pointer;
}
.top button.active { background: #0d7a6f; color: #fff; }
.flow { display: flex; flex-wrap: wrap; gap: 6px; padding: 10px 16px; }
.flow span { background: #1a2b34; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
.flow span.on { background: #0d7a6f; }
.body { padding: 0 16px 16px; }
.panel { background: #16252d; border-radius: 8px; padding: 12px; }
.panel h3 { margin: 0 0 10px; color: #9fd4cb; font-size: 14px; }
.scan-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
@media (max-width: 900px) { .scan-grid { grid-template-columns: 1fr; } }
</style>
