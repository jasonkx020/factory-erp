<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { notifyApi, purchaseApi, parsePayload } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const loading = ref(false)
const tasks = ref<Row[]>([])

function payloadOf(row: Row) {
  return parsePayload(row.payload ?? row.payload_json)
}

const taskCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '过磅单号', primary: true },
  { prop: 'trace_code', label: '溯源码' },
  { prop: 'plate_no', label: '车牌' },
  { prop: 'weights', label: '入场/净重' },
  { prop: 'fees', label: '费用' },
  { prop: 'created_at', label: '推送时间' },
]

/** 卡片展示用：展开 payload 字段，不改业务数据源 */
const tasksForCards = computed(() =>
  tasks.value.map((row) => {
    const p = payloadOf(row)
    return {
      ...row,
      plate_no: p.plate_no || '-',
      weights: `${p.gross_weight ?? '-'} / ${p.net_weight ?? '-'}`,
      fees: `运${p.freight_fee ?? 0}/装${p.loading_fee ?? 0}/磅${p.weigh_fee ?? 0}`,
    }
  }),
)

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
  const p = payloadOf(row)
  const kind = String(p.receive_kind || row.receive_kind || '').toLowerCase()
  if (kind === 'stockin') {
    return ElMessage.warning('入库须在 App 仓管核对页分箱复磅后确认（本页仅支持入厂接收）')
  }
  const res = await purchaseApi.warehouseConfirmWeigh(bizId, { verified: true, match_confirmed: true })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const data = (res.data as Row) || {}
  ElMessage.success(`入厂接收完成 溯源=${data.trace_code || '-'}`)
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>仓管待办</h2>
    <p class="hint">采购出码后推送至此：先核对入厂接收；入厂后扫同一溯源码分箱入库（App）。</p>
    <el-button type="primary" @click="refresh">刷新待办</el-button>
    <TableOrCards :data="tasksForCards" :loading="loading" :columns="taskCols" style="margin-top:12px">
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
            <el-button link type="success" @click="confirmInbound(row)">确认接收/入库</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #actions="{ row }">
        <el-button link type="primary" @click="claim(Number(row.id))">认领</el-button>
        <el-button link type="success" @click="confirmInbound(row)">确认接收/入库</el-button>
      </template>
    </TableOrCards>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
</style>
