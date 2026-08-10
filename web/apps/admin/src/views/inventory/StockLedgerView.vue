<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fieldLedgerApi, inventoryApi } from '@erp/shared'
import { useRouter } from 'vue-router'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>
const bridges = ref<Row[]>([])
const balances = ref<Row[]>([])
const loading = ref(false)
const form = reactive({ code: '', name: '', location: '' })
const router = useRouter()

const bridgeCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'location', label: '位置' },
  { prop: 'status', label: '状态' },
]

const balanceCols: MobileCardColumn[] = [
  { prop: 'product_name', label: '名称', primary: true },
  { prop: 'warehouse_name', label: '仓' },
  { prop: 'product_code', label: '料号' },
  { prop: 'qty', label: '数量' },
]

async function refresh() {
  loading.value = true
  try {
    const [w, b] = await Promise.all([fieldLedgerApi.weighbridges(), inventoryApi.balances()])
    bridges.value = ((w.data as { list?: Row[] })?.list) || []
    balances.value = ((b.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!form.name) return ElMessage.warning('填写名称')
  const res = await fieldLedgerApi.createWeighbridge({ ...form })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已添加地磅')
  Object.assign(form, { code: '', name: '', location: '' })
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>库存台账 / 地磅</h2>
    <el-row :gutter="16">
      <el-col :span="10" :xs="24">
        <el-card header="地磅信息">
          <el-form label-width="70px" size="small">
            <el-form-item label="编码"><el-input v-model="form.code" placeholder="可空自动生成" /></el-form-item>
            <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
            <el-form-item label="位置"><el-input v-model="form.location" /></el-form-item>
            <el-button type="primary" @click="create">新增</el-button>
          </el-form>
          <TableOrCards :data="bridges" :loading="loading" :columns="bridgeCols" style="margin-top:12px">
            <el-table :data="bridges" size="small" style="margin-top:12px">
              <el-table-column prop="code" label="编码" width="100" />
              <el-table-column prop="name" label="名称" />
              <el-table-column prop="location" label="位置" />
              <el-table-column prop="status" label="状态" width="80" />
            </el-table>
            <template #extra="{ row }">
              <el-tag v-if="row.status != null" size="small">{{ row.status }}</el-tag>
            </template>
          </TableOrCards>
        </el-card>
      </el-col>
      <el-col :span="14" :xs="24">
        <el-card header="库存查询（三仓结存）">
          <div style="margin-bottom:8px">
            <el-button @click="router.push('/inventory/hub/inbound')">采购入库待办</el-button>
            <el-button @click="router.push('/inventory/hub/stocktakes')">仓库盘点</el-button>
            <el-button @click="router.push('/inventory/hub/stock-txns')">出入库流水</el-button>
            <el-button @click="router.push('/sales/hub/deliveries')">订单发货</el-button>
          </div>
          <TableOrCards :data="balances" :loading="loading" :columns="balanceCols">
            <el-table :data="balances" size="small" max-height="480">
              <el-table-column prop="warehouse_name" label="仓" width="120" />
              <el-table-column prop="product_code" label="料号" width="100" />
              <el-table-column prop="product_name" label="名称" />
              <el-table-column prop="qty" label="数量" width="100" />
            </el-table>
          </TableOrCards>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
</style>
