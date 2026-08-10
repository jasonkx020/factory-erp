<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fieldLedgerApi } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const reportCols: MobileCardColumn[] = [
  { prop: 'process_name', label: '工序', primary: true },
  { prop: 'id', label: 'ID' },
  { prop: 'input_weight', label: '投料' },
  { prop: 'output_weight', label: '完工' },
  { prop: 'loss', label: '损耗' },
  { prop: 'bag_qty', label: '袋数' },
  { prop: 'scrap_type', label: '次品类型' },
  { prop: 'status', label: '状态' },
  { prop: 'reported_at', label: '时间' },
]
const list = ref<Row[]>([])
const scrapType = ref('')
const loading = ref(false)

async function refresh() {
  loading.value = true
  try {
    const q = scrapType.value ? `scrap_type=${encodeURIComponent(scrapType.value)}` : undefined
    const res = await fieldLedgerApi.processReports(q)
    if (res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>加工记录</h2>
    <div style="margin-bottom:12px;display:flex;gap:8px;align-items:center">
      <el-select v-model="scrapType" clearable placeholder="次品类型" style="width:200px" @change="refresh">
        <el-option label="切断次品" value="cut_defect" />
        <el-option label="去芯次品" value="core_defect" />
        <el-option label="切块次品" value="dice_defect" />
        <el-option label="筛选装袋次品" value="sieve_bag_defect" />
      </el-select>
      <el-button @click="refresh">刷新</el-button>
    </div>
    <TableOrCards :data="list" :loading="loading" :columns="reportCols">
      <el-table :data="list" size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="process_name" label="工序" width="120" />
        <el-table-column prop="input_weight" label="投料" width="80" />
        <el-table-column prop="output_weight" label="完工" width="80" />
        <el-table-column prop="loss" label="损耗" width="80" />
        <el-table-column prop="bag_qty" label="袋数" width="70" />
        <el-table-column prop="scrap_type" label="次品类型" width="120" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="reported_at" label="时间" min-width="160" />
      </el-table>
    </TableOrCards>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
</style>
