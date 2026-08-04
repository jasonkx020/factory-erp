<script setup lang="ts">
import { ref } from 'vue'
import { inventoryApi, systemApi } from '@erp/shared'
import { ElMessage } from 'element-plus'

const boxCode = ref('BX-RAW-DEMO')
const traceId = ref('')
const boxTrace = ref<Record<string, unknown> | null>(null)
const opTrace = ref<Record<string, unknown> | null>(null)

async function loadBox() {
  const res = await inventoryApi.boxTrace(boxCode.value.trim())
  if (res.code !== 1) return ElMessage.error(res.msg)
  boxTrace.value = (res.data as Record<string, unknown>) || null
}

async function loadOp() {
  if (!traceId.value.trim()) return ElMessage.warning('请输入 trace_id')
  const res = await systemApi.logTrace(traceId.value.trim())
  if (res.code !== 1) return ElMessage.error(res.msg)
  opTrace.value = (res.data as Record<string, unknown>) || null
}
</script>

<template>
  <div class="page">
    <h2>全链追溯</h2>
    <el-tabs>
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
</style>
