<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { salesApi, BASE_UNIT_OPTIONS } from '@erp/shared'
import { ProductSelect, EnumSelect, SalesOrderSelect } from '../../components/select'
import { loadProducts } from '../../components/select/entitySelects'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'
import { dash, money, statusLabel, statusType, type SalesRow } from './salesUi'

type Row = SalesRow

const settleCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'product_name', label: '产品' },
  { prop: 'order_no', label: '订单' },
  { prop: 'weight', label: '重量' },
  { prop: 'amount', label: '结算金额' },
  { prop: 'status', label: '状态' },
]

const router = useRouter()
const list = ref<Row[]>([])
const loading = ref(false)
const createDlg = ref(false)
const drawer = ref(false)
const detail = ref<Row | null>(null)
const productCache = ref<Row[]>([])
const form = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  product_id: null as number | null,
  product_name: '',
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
  order_id: null as number | null,
  delivery_id: null as number | null,
})

const stats = computed(() => {
  const draft = list.value.filter((r) => r.status === 'draft').length
  const closed = list.value.filter((r) => r.status === 'closed').length
  const amount = list.value.reduce((s, r) => s + Number(r.amount || 0), 0)
  return [
    { label: '单据', value: String(list.value.length) },
    { label: '草稿', value: String(draft), warn: true },
    { label: '已关单', value: String(closed), ok: true },
    { label: '结算额', value: money(amount), ok: true },
  ]
})

async function resolveProductName(id: number | null) {
  if (!id) {
    form.product_name = ''
    return
  }
  if (!productCache.value.length) productCache.value = await loadProducts()
  const p = productCache.value.find((x) => Number(x.id) === id)
  form.product_name = p ? String(p.name || '') : ''
}

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
  if (!form.product_id) return ElMessage.warning('请选择产品')
  await resolveProductName(form.product_id)
  const res = await salesApi.createOutboundSettle({
    ...form,
    product_id: form.product_id,
    product_name: form.product_name,
    qty: form.qty || form.weight,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已创建，结算金额 ${(res.data as Row)?.amount}`)
  createDlg.value = false
  await refresh()
}

async function closeRow(id: number) {
  const res = await salesApi.closeOutboundSettle(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已关单，反关单仅限误操作回退')
  await refresh()
}

async function reopenRow(id: number) {
  const res = await salesApi.reopenOutboundSettle(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已反关单，可继续修改')
  await refresh()
}

async function openDetail(id: number) {
  const res = await salesApi.getOutboundSettle(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
  drawer.value = true
}

onMounted(refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <header class="page-head">
      <div>
        <h2 class="title">出厂结算</h2>
        <p class="desc">结算金额 = 重量×单价 + 运费 + 装卸费 + 过磅费。可关联销售订单/发货单；关单后仅允许反关单回退草稿，不可删除。</p>
      </div>
      <span class="meta-pill">财务联动 · 收款核单 / 合同利润</span>
    </header>
    <div class="stats">
      <div v-for="s in stats" :key="s.label" class="stat" :class="{ ok: s.ok, warn: s.warn }">
        <div class="label">{{ s.label }}</div>
        <div class="value">{{ s.value }}</div>
      </div>
    </div>
    <div class="toolbar">
      <el-button type="primary" @click="createDlg = true">新建出厂单</el-button>
      <el-button @click="refresh">刷新</el-button>
      <el-button link type="primary" @click="router.push('/finance/hub/writeoffs')">收款核单</el-button>
      <el-button link type="primary" @click="router.push('/finance/hub/contract-profits')">合同利润</el-button>
      <el-button link @click="router.push('/sales/hub/deliveries')">发货审批</el-button>
    </div>
    <TableOrCards :data="list" :loading="loading" :columns="settleCols">
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" />
        <el-table-column prop="product_name" label="产品" width="120" />
        <el-table-column prop="order_no" label="订单" width="140" />
        <el-table-column prop="delivery_no" label="发货" width="140" />
        <el-table-column prop="plate_no" label="车牌" width="90" />
        <el-table-column prop="driver_name" label="司机" width="90" />
        <el-table-column prop="trace_code" label="溯源" min-width="120" />
        <el-table-column prop="weight" label="重量" width="80" />
        <el-table-column label="货款" width="80"><template #default="{ row }">{{ money(row.goods_amount) }}</template></el-table-column>
        <el-table-column label="结算金额" width="100"><template #default="{ row }">{{ money(row.amount) }}</template></el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button link @click="openDetail(Number(row.id))">详情</el-button>
            <el-button v-if="row.status==='draft'" link type="success" @click="closeRow(Number(row.id))">关单</el-button>
            <el-button v-if="row.status==='closed'" link type="warning" @click="reopenRow(Number(row.id))">反关单</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #actions="{ row }">
        <el-button link @click="openDetail(Number(row.id))">详情</el-button>
        <el-button v-if="row.status==='draft'" link type="success" @click="closeRow(Number(row.id))">关单</el-button>
        <el-button v-if="row.status==='closed'" link type="warning" @click="reopenRow(Number(row.id))">反关单</el-button>
      </template>
    </TableOrCards>

    <el-drawer v-model="drawer" title="出厂结算详情" size="480px">
      <template v-if="detail">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="单号">{{ dash(detail.doc_no) }}</el-descriptions-item>
          <el-descriptions-item label="状态"><el-tag size="small" :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="产品">{{ dash(detail.product_name) }}</el-descriptions-item>
          <el-descriptions-item label="重量">{{ dash(detail.weight) }} {{ dash(detail.unit) }}</el-descriptions-item>
          <el-descriptions-item label="货款">{{ money(detail.goods_amount) }}</el-descriptions-item>
          <el-descriptions-item label="结算金额">{{ money(detail.amount) }}</el-descriptions-item>
          <el-descriptions-item label="关单时间">{{ dash(detail.closed_at) }}</el-descriptions-item>
        </el-descriptions>
        <div class="mt">
          <el-button v-if="(detail.order as Row)?.id" link type="primary" @click="router.push('/sales/hub/orders')">关联订单 {{ (detail.order as Row).doc_no }}</el-button>
          <el-button v-if="(detail.delivery as Row)?.id" link type="primary" @click="router.push('/sales/hub/deliveries')">关联发货 {{ (detail.delivery as Row).doc_no }}</el-button>
        </div>
      </template>
    </el-drawer>

    <el-dialog v-model="createDlg" title="新建出厂单" width="640px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="日期"><el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="关联订单"><SalesOrderSelect v-model="form.order_id" /></el-form-item>
        <el-form-item label="产品"><ProductSelect v-model="form.product_id" @update:model-value="resolveProductName" /></el-form-item>
        <el-form-item label="车牌"><el-input v-model="form.plate_no" /></el-form-item>
        <el-form-item label="司机"><el-input v-model="form.driver_name" /></el-form-item>
        <el-form-item label="溯源批号"><el-input v-model="form.trace_code" /></el-form-item>
        <el-form-item label="生产日期"><el-date-picker v-model="form.produce_date" type="date" value-format="YYYY-MM-DD" clearable /></el-form-item>
        <el-form-item label="重量"><el-input-number v-model="form.weight" :min="0" /></el-form-item>
        <el-form-item label="单位"><EnumSelect v-model="form.unit" :options="BASE_UNIT_OPTIONS" :clearable="false" /></el-form-item>
        <el-form-item label="单价"><el-input-number v-model="form.unit_price" :min="0" :step="0.1" /></el-form-item>
        <el-form-item label="运费"><el-input-number v-model="form.freight_fee" :min="0" /></el-form-item>
        <el-form-item label="装卸"><el-input-number v-model="form.loading_fee" :min="0" /></el-form-item>
        <el-form-item label="过磅费"><el-input-number v-model="form.weigh_fee" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDlg = false">取消</el-button>
        <el-button type="primary" @click="create">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { background: #fff; padding: 16px 18px; border-radius: 10px; border: 1px solid #e2e8ee; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; color: #1f2a33; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; line-height: 1.5; max-width: 720px; }
.meta-pill { display: inline-block; padding: 4px 10px; border-radius: 999px; background: #eef6f1; color: #2f6b4f; font-size: 12px; }
.stats { display: grid; grid-template-columns: repeat(4, minmax(96px, 1fr)); gap: 10px; margin-bottom: 14px; }
.stat { background: #f6f8fa; border: 1px solid #e8eef2; border-radius: 8px; padding: 10px 12px; }
.stat.ok { background: #eef6f1; border-color: #d5eade; }
.stat.warn { background: #fff7f0; border-color: #f0e0d0; }
.stat .label { font-size: 12px; color: #6b7a85; }
.stat .value { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; }
.toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.mt { margin-top: 12px; }
</style>
