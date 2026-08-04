<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { purchaseApi } from '@erp/shared'

type Row = Record<string, unknown>

const farmers = ref<Row[]>([])
const tickets = ref<Row[]>([])
const settlements = ref<Row[]>([])
const loading = ref(false)
const farmerForm = reactive({ name: '', mobile: '', origin: '', remark: '' })
const weighForm = reactive({
  farmer_id: 1,
  channel: 'internal',
  gross_weight: 1000,
  deduct_rate: 0.05,
  variety: '鲜木薯',
  source_type: 'self',
  image_url: '',
})
const traceCode = ref('')
const traceResult = ref<Row | null>(null)

async function refresh() {
  loading.value = true
  try {
    const [f, t, s] = await Promise.all([
      purchaseApi.farmers(),
      purchaseApi.weighTickets(),
      purchaseApi.farmerSettlements(),
    ])
    farmers.value = ((f.data as { list?: Row[] })?.list) || []
    tickets.value = ((t.data as { list?: Row[] })?.list) || []
    settlements.value = ((s.data as { list?: Row[] })?.list) || []
    if (farmers.value.length && !weighForm.farmer_id) {
      weighForm.farmer_id = Number(farmers.value[0].id)
    }
  } finally {
    loading.value = false
  }
}

async function createFarmer() {
  if (!farmerForm.name) return ElMessage.warning('请填写农户姓名')
  const res = await purchaseApi.createFarmer({ ...farmerForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('农户已建档')
  farmerForm.name = ''
  farmerForm.mobile = ''
  farmerForm.origin = ''
  await refresh()
}

async function createWeigh() {
  const res = await purchaseApi.createWeighTicket({ ...weighForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`过磅单已创建，净重 ${(res.data as Row)?.net_weight}`)
  await refresh()
}

async function qcPass(id: number) {
  const res = await purchaseApi.qcWeighTicket(id, { qc_result: 'pass', auto_stock_in: true })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`质检合格已入保鲜库 箱码=${(res.data as Row)?.box_code || '-'}`)
  await refresh()
}

async function qcFail(id: number) {
  const res = await purchaseApi.qcWeighTicket(id, { qc_result: 'fail', auto_stock_in: false })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('不合格已留档，未入库')
  await refresh()
}

async function doTrace() {
  if (!traceCode.value) return
  const res = await purchaseApi.traceLot(traceCode.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  traceResult.value = (res.data as Row) || null
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>农户过磅入场</h2>
    <p class="hint">散户建档 → 双渠道过磅（外磅/厂内）→ 扣损净重 → 质检 → 合格入保鲜库并绑定追溯码；不合格留档不入库。</p>

    <el-row :gutter="16">
      <el-col :span="10">
        <el-card header="新建农户">
          <el-form label-width="80px">
            <el-form-item label="姓名"><el-input v-model="farmerForm.name" /></el-form-item>
            <el-form-item label="电话"><el-input v-model="farmerForm.mobile" /></el-form-item>
            <el-form-item label="产地"><el-input v-model="farmerForm.origin" /></el-form-item>
            <el-button type="primary" @click="createFarmer">保存</el-button>
          </el-form>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card header="过磅单">
          <el-form label-width="100px" inline>
            <el-form-item label="农户">
              <el-select v-model="weighForm.farmer_id" style="width:160px">
                <el-option v-for="f in farmers" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="渠道">
              <el-select v-model="weighForm.channel" style="width:120px">
                <el-option label="厂内叉车秤" value="internal" />
                <el-option label="外部过磅单" value="external" />
              </el-select>
            </el-form-item>
            <el-form-item label="来源">
              <el-select v-model="weighForm.source_type" style="width:120px">
                <el-option label="自产原料" value="self" />
                <el-option label="外购半成品" value="outsource" />
              </el-select>
            </el-form-item>
            <el-form-item label="毛重"><el-input-number v-model="weighForm.gross_weight" :min="0" /></el-form-item>
            <el-form-item label="扣损率"><el-input-number v-model="weighForm.deduct_rate" :min="0" :max="1" :step="0.01" /></el-form-item>
            <el-form-item label="影像URL"><el-input v-model="weighForm.image_url" placeholder="P2 OCR 前先存链接" style="width:200px" /></el-form-item>
            <el-button type="primary" @click="createWeigh">创建过磅单</el-button>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-card header="过磅单列表" style="margin-top:16px">
      <el-table :data="tickets" size="small">
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="farmer_name" label="农户" width="100" />
        <el-table-column prop="channel" label="渠道" width="90" />
        <el-table-column prop="source_type" label="来源" width="90" />
        <el-table-column prop="gross_weight" label="毛重" width="80" />
        <el-table-column prop="net_weight" label="净重" width="80" />
        <el-table-column prop="qc_result" label="质检" width="70" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="trace_code" label="追溯码" min-width="140" />
        <el-table-column prop="box_code" label="箱码" min-width="120" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="success" @click="qcPass(Number(row.id))">合格入库</el-button>
            <el-button v-if="row.status==='draft'" link type="danger" @click="qcFail(Number(row.id))">不合格</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="12">
        <el-card header="农户结算依据（扣损后净重）">
          <el-table :data="settlements" size="small">
            <el-table-column prop="doc_no" label="结算单" />
            <el-table-column prop="farmer_name" label="农户" width="100" />
            <el-table-column prop="net_weight" label="净重" width="90" />
            <el-table-column prop="amount" label="金额" width="90" />
            <el-table-column prop="status" label="状态" width="80" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card header="追溯码反查">
          <el-input v-model="traceCode" placeholder="追溯码或箱码" style="max-width:280px;margin-right:8px" />
          <el-button type="primary" @click="doTrace">查询</el-button>
          <pre v-if="traceResult" class="trace">{{ JSON.stringify(traceResult, null, 2) }}</pre>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
.trace { background: #f6f8fa; padding: 12px; border-radius: 8px; margin-top: 12px; max-height: 320px; overflow: auto; font-size: 12px; }
</style>
