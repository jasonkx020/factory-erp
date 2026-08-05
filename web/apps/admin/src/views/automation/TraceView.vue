<script setup lang="ts">
import { ref } from 'vue'
import { inventoryApi, systemApi, purchaseApi } from '@erp/shared'
import { ElMessage } from 'element-plus'

type Row = Record<string, unknown>

const boxCode = ref('BX-RAW-DEMO')
const traceId = ref('')
const lotCode = ref('')
const boxTrace = ref<Row | null>(null)
const opTrace = ref<Row | null>(null)
const lotTimeline = ref<Row | null>(null)

async function loadBox() {
  const res = await inventoryApi.boxTrace(boxCode.value.trim())
  if (res.code !== 1) return ElMessage.error(res.msg)
  boxTrace.value = (res.data as Row) || null
}

async function loadOp() {
  if (!traceId.value.trim()) return ElMessage.warning('请输入 trace_id')
  const res = await systemApi.logTrace(traceId.value.trim())
  if (res.code !== 1) return ElMessage.error(res.msg)
  opTrace.value = (res.data as Row) || null
}

async function loadLot() {
  if (!lotCode.value.trim()) return ElMessage.warning('请输入溯源码/箱码')
  const res = await purchaseApi.traceLot(lotCode.value.trim())
  if (res.code !== 1) return ElMessage.error(res.msg)
  lotTimeline.value = (res.data as Row) || null
}
</script>

<template>
  <div class="page">
    <h2>全链追溯 / 证据倒查</h2>
    <el-tabs>
      <el-tab-pane label="采购溯源时间轴">
        <div class="row">
          <el-input v-model="lotCode" placeholder="T1-… / 箱码 / 过磅单号" style="max-width:360px" />
          <el-button type="primary" @click="loadLot">倒查</el-button>
        </div>
        <div v-if="lotTimeline" class="timeline">
          <el-alert
            v-if="(lotTimeline.lot as Row)?.signature_valid === false"
            type="warning"
            title="HMAC 验签未通过"
            show-icon
            :closable="false"
            style="margin-bottom:12px"
          />
          <div v-for="(ev, i) in ((lotTimeline.timeline as Row[]) || [])" :key="i" class="tl-item">
            <div class="tl-head">{{ ev.step }}</div>
            <div v-if="Array.isArray(ev.evidences) && (ev.evidences as Row[]).length" class="evs">
              <img
                v-for="e in (ev.evidences as Row[])"
                :key="String(e.id)"
                :src="String(e.file_url)"
                class="thumb"
                :alt="String(e.evidence_type)"
              />
            </div>
            <pre>{{ JSON.stringify(ev, null, 2) }}</pre>
          </div>
        </div>
      </el-tab-pane>
      <el-tab-pane label="按箱码">
        <div class="row">
          <el-input v-model="boxCode" placeholder="箱码" style="max-width:280px" />
          <el-button type="primary" @click="loadBox">查询</el-button>
        </div>
        <pre v-if="boxTrace" class="dump">{{ JSON.stringify(boxTrace, null, 2) }}</pre>
      </el-tab-pane>
      <el-tab-pane label="按 Trace ID">
        <div class="row">
          <el-input v-model="traceId" placeholder="X-Trace-Id / trace_id" style="max-width:360px" />
          <el-button type="primary" @click="loadOp">查询</el-button>
        </div>
        <pre v-if="opTrace" class="dump">{{ JSON.stringify(opTrace, null, 2) }}</pre>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.page { padding: 4px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; }
.dump { background: #f6f8fa; padding: 12px; border-radius: 8px; font-size: 12px; max-height: 70vh; overflow: auto; }
.timeline { max-height: 70vh; overflow: auto; }
.tl-item { border-left: 3px solid #0d7a6f; padding: 0 0 14px 12px; margin-bottom: 8px; }
.tl-head { font-weight: 600; margin-bottom: 4px; }
.tl-item pre { background: #f6f8fa; padding: 8px; border-radius: 6px; font-size: 11px; margin: 4px 0 0; }
.evs { display: flex; gap: 8px; flex-wrap: wrap; margin: 6px 0; }
.thumb { width: 72px; height: 72px; object-fit: cover; border-radius: 4px; background: #ddd; }
</style>
