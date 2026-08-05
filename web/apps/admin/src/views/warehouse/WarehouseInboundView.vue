<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { notifyApi, purchaseApi, parsePayload } from '@erp/shared'

type Row = Record<string, unknown>

const loading = ref(false)
const tasks = ref<Row[]>([])

function payloadOf(row: Row) {
  return parsePayload(row.payload ?? row.payload_json)
}

async function refresh() {
  loading.value = true
  try {
    const res = await notifyApi.tasks('status=pending&page_num=1&page_size=50')
    if (res.code !== 1) return ElMessage.error(res.msg)
    const list = ((res.data as { list?: Row[] })?.list) || []
    tasks.value = list.filter((t) => t.event_key === 'purchase.weigh_confirmed' || t.to_role === 'warehouse')
  } finally {
    loading.value = false
  }
}

async function claim(id: number) {
  const res = await notifyApi.claimTask(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已认领')
  await refresh()
}

async function confirmInbound(row: Row) {
  const bizId = Number(row.biz_id)
  const res = await purchaseApi.warehouseConfirmWeigh(bizId)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`入库完成 箱码=${(res.data as Row)?.box_code || '-'}`)
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>仓管待入库</h2>
    <p class="hint">采购确认出码后推送溯源码与过磅单号至此；仓管核对车牌/溯源等后确认入库。</p>
    <el-button type="primary" @click="refresh">刷新待办</el-button>
    <el-table :data="tasks" size="small" style="margin-top:12px">
      <el-table-column prop="doc_no" label="过磅单号" width="160" />
      <el-table-column prop="trace_code" label="溯源码" min-width="180" />
      <el-table-column label="车牌" width="100">
        <template #default="{ row }">{{ payloadOf(row).plate_no || '-' }}</template>
      </el-table-column>
      <el-table-column label="入场/净重" width="120">
        <template #default="{ row }">
          {{ payloadOf(row).gross_weight ?? '-' }} / {{ payloadOf(row).net_weight ?? '-' }}
        </template>
      </el-table-column>
      <el-table-column label="费用" min-width="160">
        <template #default="{ row }">
          运{{ payloadOf(row).freight_fee ?? 0 }}
          /装{{ payloadOf(row).loading_fee ?? 0 }}
          /磅{{ payloadOf(row).weigh_fee ?? 0 }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="推送时间" width="160" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="claim(Number(row.id))">认领</el-button>
          <el-button link type="success" @click="confirmInbound(row)">确认入库</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
</style>
