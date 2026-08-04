<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { productionApi } from '@erp/shared'
import { ElMessage } from 'element-plus'

const routings = ref<Record<string, unknown>[]>([])
const current = ref<Record<string, unknown> | null>(null)
const rules = ref<Record<string, unknown> | null>(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const [r, fr] = await Promise.all([productionApi.listRoutings(), productionApi.flowRules()])
    routings.value = ((r.data as { list?: Record<string, unknown>[] })?.list) || []
    rules.value = (fr.data as Record<string, unknown>) || null
    if (routings.value.length && !current.value) {
      await openRouting(Number(routings.value[0].id))
    }
  } finally {
    loading.value = false
  }
}

async function openRouting(id: number) {
  const res = await productionApi.getRouting(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  current.value = (res.data as Record<string, unknown>) || null
}

onMounted(load)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>工艺路线编排</h2>
    <p class="sub">木薯产线 12 步：计件 / 卡点 / 自动入库 / 自动开下一工单由步骤标志驱动。</p>
    <div class="grid">
      <el-table :data="routings" border size="small" highlight-current-row @current-change="(r:any)=>r&&openRouting(Number(r.id))">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="code" label="编码" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="status" label="状态" width="100" />
      </el-table>
      <div>
        <h3>步骤明细 {{ current?.code || '' }}</h3>
        <el-table :data="(current?.lines as any[]) || (current?.steps as any[]) || []" border size="small">
          <el-table-column prop="seq_no" label="#" width="50" />
          <el-table-column prop="name" label="工序" />
          <el-table-column prop="process_id" label="工序ID" width="80" />
          <el-table-column prop="is_piecework" label="计件" width="70" />
          <el-table-column prop="is_inbound_checkpoint" label="卡点" width="70" />
          <el-table-column prop="auto_next" label="自动下步" width="90" />
          <el-table-column prop="stock_action" label="库存动作" />
        </el-table>
        <el-card v-if="rules" shadow="never" style="margin-top:12px">
          <template #header>流转规则</template>
          <pre style="margin:0;font-size:12px;white-space:pre-wrap">{{ JSON.stringify(rules, null, 2) }}</pre>
        </el-card>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 4px; }
.sub { color: #667; font-size: 13px; }
.grid { display: grid; grid-template-columns: 360px 1fr; gap: 16px; }
h2 { margin: 0 0 8px; }
h3 { margin: 0 0 8px; font-size: 14px; }
</style>
