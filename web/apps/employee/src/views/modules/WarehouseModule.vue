<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  useAuthStore,
  useNotifyStore,
  notifyApi,
  purchaseApi,
  canAccessEmployeeModule,
  portalHomeUrl,
  parsePayload,
} from '@erp/shared'
import { ElMessage } from 'element-plus'

type Row = Record<string, unknown>

const auth = useAuthStore()
const notify = useNotifyStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const tasks = ref<Row[]>([])
const loading = ref(false)
const verifyCode = ref('')
const activeId = ref<number | null>(null)

const activeTask = computed(() => tasks.value.find((t) => Number(t.id) === activeId.value) || null)

function payloadOf(row: Row) {
  return parsePayload(row.payload ?? row.payload_json)
}

async function load() {
  if (!auth.isLoggedIn) {
    router.replace('/login')
    return
  }
  await auth.fetchMe()
  if (!canAccessEmployeeModule('warehouse', auth.permissions, auth.roles)) {
    ElMessage.warning('无仓管模块权限')
    router.replace('/')
    return
  }
  loading.value = true
  try {
    const res = await notifyApi.tasks('status=pending')
    if (res.code !== 1) return ElMessage.error(res.msg)
    const list = ((res.data as { list?: Row[] })?.list) || []
    tasks.value = list.filter(
      (t) => t.event_key === 'purchase.weigh_confirmed' || t.to_role === 'warehouse',
    )
    if (tasks.value.length && activeId.value == null) {
      activeId.value = Number(tasks.value[0].id)
      verifyCode.value = ''
    }
  } finally {
    loading.value = false
  }
}

async function claim(row: Row) {
  const res = await notifyApi.claimTask(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已认领')
  await load()
}

async function confirm(row: Row) {
  const expect = String(row.trace_code || payloadOf(row).trace_code || '').trim()
  const got = verifyCode.value.trim()
  if (!got) return ElMessage.warning('请输入或扫描溯源码核对')
  if (expect && got !== expect) return ElMessage.error('溯源码不一致，请核对推送内容')
  const id = Number(row.biz_id)
  const res = await purchaseApi.warehouseConfirmWeigh(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`入库完成 ${(res.data as Row)?.box_code || ''}，已推送财务`)
  verifyCode.value = ''
  activeId.value = null
  await load()
}

watch(
  () => notify.tick,
  () => {
    void load()
  },
)

onMounted(async () => {
  await notify.start()
  await load()
})
</script>

<template>
  <div class="pad" v-loading="loading">
    <header class="top">
      <a class="portal-link" :href="portalUrl">← 入口</a>
      <button type="button" class="back" @click="router.push('/')">模块</button>
      <h1>仓管入库</h1>
      <span class="meta">MQTT {{ notify.mqttStatus }} · 未读 {{ notify.unread }}</span>
    </header>
    <main>
      <p class="hint">核对推送的溯源码与过磅单号后确认入库；完成后推送财务结算。</p>
      <el-button @click="load">刷新待办</el-button>
      <el-table
        :data="tasks"
        size="small"
        style="margin-top:12px"
        highlight-current-row
        @current-change="(row: Row | undefined) => { if (row) { activeId = Number(row.id); verifyCode = '' } }"
      >
        <el-table-column prop="doc_no" label="单号" width="140" />
        <el-table-column prop="trace_code" label="溯源码" min-width="160" />
        <el-table-column label="详情" min-width="220">
          <template #default="{ row }">
            <span class="mono">
              {{ payloadOf(row).farmer_name || '' }}
              {{ payloadOf(row).plate_no ? `· 车牌${payloadOf(row).plate_no}` : '' }}
              {{ payloadOf(row).net_weight != null ? `· ${payloadOf(row).net_weight}kg` : '' }}
              {{ payloadOf(row).grade ? `· ${payloadOf(row).grade}` : '' }}
              {{ Number(payloadOf(row).freight_fee || 0) || Number(payloadOf(row).loading_fee || 0) || Number(payloadOf(row).weigh_fee || 0)
                ? `· 费运${payloadOf(row).freight_fee || 0}/装${payloadOf(row).loading_fee || 0}/磅${payloadOf(row).weigh_fee || 0}`
                : '' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button type="primary" link @click="claim(row)">认领</el-button>
            <el-button type="success" link @click="activeId = Number(row.id)">核对</el-button>
          </template>
        </el-table-column>
      </el-table>

      <section v-if="activeTask" class="verify">
        <h3>核对入库 · {{ activeTask.doc_no }}</h3>
        <p>推送溯源码：<code>{{ activeTask.trace_code }}</code></p>
        <p class="mono">
          车牌 {{ payloadOf(activeTask).plate_no || '-' }}
          · 收货地址 {{ payloadOf(activeTask).receive_address || '-' }}
          · 合格率 {{ payloadOf(activeTask).pass_rate ?? '-' }}%
          · 不合格重 {{ payloadOf(activeTask).reject_weight ?? 0 }}kg
        </p>
        <el-input v-model="verifyCode" placeholder="输入/扫描溯源码核对" clearable />
        <div style="margin-top:10px;display:flex;gap:8px">
          <el-button type="success" @click="confirm(activeTask)">确认入库</el-button>
          <el-button @click="activeId = null">取消</el-button>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.pad { max-width: 960px; margin: 0 auto; padding: 12px; }
.top { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
.top h1 { margin: 0; font-size: 18px; flex: 1; }
.back, .portal-link { border: 0; background: transparent; color: #0d7a6f; cursor: pointer; text-decoration: none; }
.meta { font-size: 12px; color: #889; }
.hint { color: #667; font-size: 13px; }
.mono { font-size: 12px; color: #445; }
.verify { margin-top: 16px; background: #f7faf9; border-radius: 8px; padding: 12px; }
.verify h3 { margin: 0 0 8px; font-size: 15px; color: #0d7a6f; }
</style>
