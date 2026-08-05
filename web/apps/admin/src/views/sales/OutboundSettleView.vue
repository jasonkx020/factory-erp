<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { salesApi } from '@erp/shared'

type Row = Record<string, unknown>
const list = ref<Row[]>([])
const loading = ref(false)
const form = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  product_name: '去芯切块',
  plate_no: '',
  driver_name: '',
  trace_code: '',
  produce_date: '',
  weight: 0,
  qty: 0,
  unit: 'kg',
  unit_price: 0,
  freight_fee: 0,
  loading_fee: 0,
  weigh_fee: 0,
})

async function refresh() {
  loading.value = true
  try {
    const res = await salesApi.outboundSettles()
    if (res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function create() {
  const res = await salesApi.createOutboundSettle({ ...form, qty: form.qty || form.weight })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已创建，结算金额 ${(res.data as Row)?.amount}`)
  await refresh()
}

async function closeRow(id: number) {
  const res = await salesApi.closeOutboundSettle(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已关单')
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>销售出厂结算</h2>
    <p class="hint">结算金额 = 重量×单价 + 运费 + 装卸费 + 过磅费</p>
    <el-card header="新建出厂单">
      <el-form inline size="small">
        <el-form-item label="日期"><el-input v-model="form.biz_date" /></el-form-item>
        <el-form-item label="产品"><el-input v-model="form.product_name" /></el-form-item>
        <el-form-item label="车牌"><el-input v-model="form.plate_no" /></el-form-item>
        <el-form-item label="司机"><el-input v-model="form.driver_name" /></el-form-item>
        <el-form-item label="溯源批号"><el-input v-model="form.trace_code" /></el-form-item>
        <el-form-item label="生产日期"><el-input v-model="form.produce_date" /></el-form-item>
        <el-form-item label="重量"><el-input-number v-model="form.weight" :min="0" /></el-form-item>
        <el-form-item label="单位">
          <el-select v-model="form.unit" style="width:90px">
            <el-option label="kg" value="kg" /><el-option label="袋" value="bag" />
          </el-select>
        </el-form-item>
        <el-form-item label="单价"><el-input-number v-model="form.unit_price" :min="0" :step="0.1" /></el-form-item>
        <el-form-item label="运费"><el-input-number v-model="form.freight_fee" :min="0" /></el-form-item>
        <el-form-item label="装卸"><el-input-number v-model="form.loading_fee" :min="0" /></el-form-item>
        <el-form-item label="过磅费"><el-input-number v-model="form.weigh_fee" :min="0" /></el-form-item>
        <el-button type="primary" @click="create">保存</el-button>
      </el-form>
    </el-card>
    <el-table :data="list" size="small" style="margin-top:12px">
      <el-table-column prop="doc_no" label="单号" width="150" />
      <el-table-column prop="product_name" label="产品" width="120" />
      <el-table-column prop="plate_no" label="车牌" width="90" />
      <el-table-column prop="driver_name" label="司机" width="90" />
      <el-table-column prop="trace_code" label="溯源" min-width="120" />
      <el-table-column prop="weight" label="重量" width="80" />
      <el-table-column prop="goods_amount" label="货款" width="80" />
      <el-table-column prop="amount" label="结算金额" width="100" />
      <el-table-column prop="status" label="状态" width="80" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button v-if="row.status==='draft'" link type="success" @click="closeRow(Number(row.id))">关单</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.hint { color: #667; font-size: 13px; }
</style>
