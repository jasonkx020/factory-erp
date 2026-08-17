<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { inventoryApi, systemApi } from '@erp/shared'
import { ElMessage } from 'element-plus'
import TraceLotPanel from '../../components/trace/TraceLotPanel.vue'
import { useCarrierCodeLabel } from '../../composables/useCarrierCodeLabel'

type Row = Record<string, unknown>

const { codeLabel, short, ensureLoaded } = useCarrierCodeLabel()

const boxCode = ref('')
const traceId = ref('')
const boxTrace = ref<Row | null>(null)
const opTrace = ref<Row | null>(null)
const boxLoading = ref(false)
const opLoading = ref(false)

async function loadBox() {
  const code = boxCode.value.trim()
  if (!code) return ElMessage.warning(`请输入${codeLabel.value}`)
  boxLoading.value = true
  try {
    const res = await inventoryApi.boxTrace(code)
    if (res.code !== 1) return ElMessage.error(res.msg)
    boxTrace.value = (res.data as Row) || null
  } finally {
    boxLoading.value = false
  }
}

async function loadOp() {
  const id = traceId.value.trim()
  if (!id) return ElMessage.warning('请输入 Trace ID')
  opLoading.value = true
  try {
    const res = await systemApi.logTrace(id)
    if (res.code !== 1) return ElMessage.error(res.msg)
    opTrace.value = (res.data as Row) || null
  } finally {
    opLoading.value = false
  }
}

const boxSummary = computed(() => {
  const m = boxTrace.value
  if (!m) return [] as { label: string; value: string }[]
  const kv = (label: string, v: unknown) =>
    v != null && String(v) !== '' ? { label, value: String(v) } : null
  const farmer = (m.farmer as Row) || {}
  return [
    kv(codeLabel.value, m.box_code),
    kv('溯源码', m.trace_code),
    kv('产地', m.origin),
    kv('收货日', m.receive_date),
    kv('来源类型', m.source_type),
    kv('农户', farmer.name),
    kv('农户电话', farmer.mobile),
    kv(`关联${short.value}数`, Array.isArray(m.related_boxes) ? (m.related_boxes as unknown[]).length : null),
  ].filter(Boolean) as { label: string; value: string }[]
})

const relatedBoxes = computed(() => {
  const list = boxTrace.value?.related_boxes
  return Array.isArray(list) ? list.map(String) : []
})

const flowEvents = computed(() => {
  const list = boxTrace.value?.flow_events
  return Array.isArray(list) ? (list as Row[]) : []
})

const boxOpLogs = computed(() => {
  const list = boxTrace.value?.operation_logs
  return Array.isArray(list) ? (list as Row[]) : []
})

const opList = computed(() => {
  const list = opTrace.value?.list
  return Array.isArray(list) ? (list as Row[]) : []
})

function triggerLabel(v: unknown) {
  const s = String(v || '')
  const map: Record<string, string> = {
    report_work: '报工过站',
    stock_in: '入库',
    stock_out: '出库',
    split: '分板',
    merge: '合箱',
  }
  return map[s] || s || '-'
}

onMounted(() => {
  void ensureLoaded()
})
</script>

<template>
  <div class="page">
    <h2>全链追溯 / 证据倒查</h2>
    <el-tabs>
      <el-tab-pane label="采购溯源时间轴">
        <TraceLotPanel />
      </el-tab-pane>

      <el-tab-pane :label="`按${codeLabel}`">
        <div class="row">
          <el-input v-model="boxCode" :placeholder="codeLabel" clearable style="max-width:280px" @keyup.enter="loadBox" />
          <el-button type="primary" :loading="boxLoading" @click="loadBox">查询</el-button>
        </div>
        <div v-if="boxTrace" v-loading="boxLoading">
          <h4 class="sec">{{ short }}信息</h4>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item v-for="f in boxSummary" :key="f.label" :label="f.label">
              {{ f.value }}
            </el-descriptions-item>
          </el-descriptions>

          <h4 class="sec">关联{{ codeLabel }}</h4>
          <div v-if="relatedBoxes.length" class="tags">
            <el-tag v-for="c in relatedBoxes" :key="c" size="small" class="tag">{{ c }}</el-tag>
          </div>
          <p v-else class="muted">无关联{{ short }}</p>

          <h4 class="sec">工艺流转事件</h4>
          <el-table v-if="flowEvents.length" :data="flowEvents" size="small" border>
            <el-table-column prop="created_at" label="时间" width="160" />
            <el-table-column label="触发" width="120">
              <template #default="{ row }">{{ triggerLabel(row.trigger) }}</template>
            </el-table-column>
            <el-table-column prop="source_type" label="来源类型" width="110" />
            <el-table-column prop="source_id" label="来源ID" width="90" />
            <el-table-column prop="from_step_id" label="从工序" width="90" />
            <el-table-column prop="to_step_id" label="到工序" width="90" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="error" label="错误" min-width="120" show-overflow-tooltip />
            <el-table-column prop="trace_id" label="Trace" min-width="140" show-overflow-tooltip />
          </el-table>
          <p v-else class="muted">暂无工艺流转事件</p>

          <h4 class="sec">操作日志</h4>
          <el-table v-if="boxOpLogs.length" :data="boxOpLogs" size="small" border>
            <el-table-column prop="created_at" label="时间" width="160" />
            <el-table-column prop="module" label="模块" width="120" />
            <el-table-column prop="action" label="动作" width="120" />
            <el-table-column prop="box_code" :label="codeLabel" min-width="140" />
            <el-table-column prop="trace_id" label="Trace" min-width="140" show-overflow-tooltip />
          </el-table>
          <p v-else class="muted">暂无操作日志</p>

          <el-collapse class="raw-collapse">
            <el-collapse-item title="原始数据（排查用）" name="raw">
              <pre class="raw">{{ JSON.stringify(boxTrace, null, 2) }}</pre>
            </el-collapse-item>
          </el-collapse>
        </div>
        <p v-else-if="!boxLoading" class="muted">输入{{ codeLabel }}后查询</p>
      </el-tab-pane>

      <el-tab-pane label="按 Trace ID">
        <div class="row">
          <el-input
            v-model="traceId"
            placeholder="X-Trace-Id / trace_id"
            clearable
            style="max-width:360px"
            @keyup.enter="loadOp"
          />
          <el-button type="primary" :loading="opLoading" @click="loadOp">查询</el-button>
        </div>
        <div v-if="opTrace" v-loading="opLoading">
          <el-descriptions :column="2" border size="small" class="mb">
            <el-descriptions-item label="Trace ID">{{ opTrace.trace_id }}</el-descriptions-item>
            <el-descriptions-item label="记录数">{{ opTrace.total ?? opList.length }}</el-descriptions-item>
          </el-descriptions>

          <el-timeline v-if="opList.length">
            <el-timeline-item
              v-for="row in opList"
              :key="String(row.id)"
              :timestamp="String(row.created_at || '')"
              placement="top"
            >
              <div class="tl-card">
                <div class="tl-title">{{ row.module || '-' }} · {{ row.action || '-' }}</div>
                <el-descriptions :column="2" size="small">
                  <el-descriptions-item label="用户">{{ row.user_id ?? '-' }}</el-descriptions-item>
                  <el-descriptions-item label="IP">{{ row.ip || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="对象">
                    {{ row.ref_type || '-' }} / {{ row.ref_id ?? '-' }}
                  </el-descriptions-item>
                </el-descriptions>
                <el-collapse v-if="row.detail_json != null" class="detail-collapse">
                  <el-collapse-item title="请求详情" name="d">
                    <pre class="raw">{{ typeof row.detail_json === 'string' ? row.detail_json : JSON.stringify(row.detail_json, null, 2) }}</pre>
                  </el-collapse-item>
                </el-collapse>
              </div>
            </el-timeline-item>
          </el-timeline>
          <p v-else class="muted">该 Trace 下暂无操作日志</p>
        </div>
        <p v-else-if="!opLoading" class="muted">输入 Trace ID 后查询</p>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.page { padding: 4px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; }
.sec { margin: 14px 0 8px; font-size: 14px; font-weight: 600; }
.mb { margin-bottom: 12px; }
.muted { color: #889; font-size: 13px; }
.tags { display: flex; flex-wrap: wrap; gap: 6px; }
.tag { margin: 0; }
.tl-card {
  background: #f8fafb;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
}
.tl-title { font-weight: 600; margin-bottom: 6px; }
.raw-collapse, .detail-collapse { margin-top: 10px; }
.raw {
  background: #f6f8fa;
  padding: 10px;
  border-radius: 6px;
  font-size: 11px;
  max-height: 280px;
  overflow: auto;
  margin: 0;
}
</style>
