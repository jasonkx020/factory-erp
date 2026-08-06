<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { purchaseApi } from '@erp/shared'

type Row = Record<string, unknown>

const loading = ref(false)
const list = ref<Row[]>([])
const form = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  lot_no: '01',
  qty: 20,
})
const filter = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  status: '',
  lot_no: '',
})

async function refresh() {
  loading.value = true
  try {
    const q = new URLSearchParams()
    q.set('biz_date', filter.biz_date)
    q.set('page_size', '200')
    if (filter.status) q.set('status', filter.status)
    if (filter.lot_no) q.set('lot_no', filter.lot_no)
    const res = await purchaseApi.traceBatchCodes(q.toString())
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function generate() {
  if (form.qty < 1) return ElMessage.warning('数量至少 1')
  const res = await purchaseApi.generateTraceBatchCodes({
    biz_date: form.biz_date,
    lot_no: form.lot_no || '01',
    qty: form.qty,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已生成 ${(res.data as Row)?.qty || form.qty} 条`)
  filter.biz_date = form.biz_date
  await refresh()
}

async function voidCode(row: Row) {
  const res = await purchaseApi.voidTraceBatchCode({ code: String(row.code || '') })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已作废')
  await refresh()
}

function copyCodes() {
  const text = list.value.map((x) => String(x.code || '')).filter(Boolean).join('\n')
  if (!text) return ElMessage.warning('无数据')
  void navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制全部批号'))
}

onMounted(refresh)
</script>

<template>
  <div v-loading="loading">
    <el-card header="按天批量生成溯源批号" style="margin-bottom: 16px">
      <el-form inline size="small">
        <el-form-item label="业务日">
          <el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="批次号">
          <el-input v-model="form.lot_no" style="width: 80px" maxlength="2" placeholder="01" />
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="form.qty" :min="1" :max="500" />
        </el-form-item>
        <el-button type="primary" @click="generate">生成</el-button>
      </el-form>
      <p class="hint">格式 TB+日期+流水+批次+校验号；落库后过磅占用，不可改码。</p>
    </el-card>

    <el-card>
      <template #header>
        <div class="hdr">
          <span>批号池</span>
          <div>
            <el-date-picker v-model="filter.biz_date" type="date" value-format="YYYY-MM-DD" size="small" @change="refresh" />
            <el-select v-model="filter.status" clearable placeholder="状态" size="small" style="width: 120px; margin-left: 8px" @change="refresh">
              <el-option value="available" label="可用" />
              <el-option value="used" label="已用" />
              <el-option value="void" label="作废" />
            </el-select>
            <el-button size="small" style="margin-left: 8px" @click="refresh">刷新</el-button>
            <el-button size="small" @click="copyCodes">复制列表</el-button>
          </div>
        </div>
      </template>
      <el-table :data="list" size="small" stripe>
        <el-table-column prop="code" label="溯源批号" min-width="200" />
        <el-table-column prop="seq_no" label="流水" width="80" />
        <el-table-column prop="lot_no" label="批次" width="70" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="weigh_ticket_id" label="过磅单" width="90" />
        <el-table-column prop="created_at" label="生成时间" width="160" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status === 'available'" link type="danger" @click="voidCode(row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.hint { color: #888; font-size: 12px; margin: 8px 0 0; }
</style>
