<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { salesApi, productApi, crmApi } from '@erp/shared'

type Row = Record<string, unknown>

const props = defineProps<{ module?: string }>()
const route = useRoute()
const router = useRouter()

const MODULE_MAP: Record<string, string> = {
  销售订单: 'orders',
  修改订单: 'orders',
  订单复购: 'orders',
  我的订单: 'my-orders',
  询价管理: 'inquiries',
  询价审批: 'inquiries',
  预发货管理: 'pre-ships',
  发货审批: 'deliveries',
  销售锁价: 'price-locks',
  合同管理: 'contracts',
  历史报价查询: 'quotes',
  数据排行榜: 'rankings',
  销售BOM: 'boms',
  成本预算: 'budgets',
  报价计算器: 'calculator',
  单据打印: 'prints',
  自助下单: 'self-orders',
  出厂结算: 'outbound',
}

const active = computed(() => {
  const section = String(route.params.section || '')
  if (section) return section
  if (props.module && MODULE_MAP[props.module]) return MODULE_MAP[props.module]
  return 'orders'
})

const title = computed(() => {
  const entry = Object.entries(MODULE_MAP).find(([, v]) => v === active.value)
  return entry?.[0] || '销售管理'
})

const loading = ref(false)
const customers = ref<Row[]>([])
const products = ref<Row[]>([])
const list = ref<Row[]>([])
const detail = ref<Row | null>(null)

const orderForm = reactive({
  customer_id: 1,
  warehouse_id: 3,
  remark: '',
  product_id: 3,
  qty: 100,
  price: 0,
})
const inquiryForm = reactive({ customer_id: 1, product_id: 3, qty: 100, quote_price: 0, remark: '' })
const lockForm = reactive({ customer_id: 1, product_id: 3, lock_price: 6.8, effective_from: new Date().toISOString().slice(0, 10), effective_to: '2026-12-31' })
const contractForm = reactive({ customer_id: 1, title: '年度供货合同', amount: 100000, remark: '' })
const preShipForm = reactive({ order_id: 0 as number, plan_ship_date: new Date().toISOString().slice(0, 10) })
const deliveryForm = reactive({ order_id: 0 as number, logistics_no: '' })
const calcForm = reactive({ product_id: 3, qty: 100, base_cost: 4, margin_rate: 0.2 })
const calcResult = ref<Row | null>(null)
const bomForm = reactive({ product_id: 3, name: '袋装木薯丁BOM', material_product_id: 1, qty: 1.2 })
const budgetForm = reactive({ order_id: 0 as number, material_cost: 0, labor_cost: 0, other_cost: 0 })
const printForm = reactive({ doc_type: 'sales_order', doc_id: 0 as number })
const selfForm = reactive({ customer_id: 1, product_id: 3, qty: 50, price: 0 })

async function loadMeta() {
  const [c, p] = await Promise.all([crmApi.customers(), productApi.list()])
  customers.value = ((c.data as { list?: Row[] })?.list) || []
  products.value = ((p.data as { list?: Row[] })?.list) || []
  if (customers.value[0]) {
    orderForm.customer_id = Number(customers.value[0].id)
    inquiryForm.customer_id = Number(customers.value[0].id)
    lockForm.customer_id = Number(customers.value[0].id)
    contractForm.customer_id = Number(customers.value[0].id)
    selfForm.customer_id = Number(customers.value[0].id)
  }
}

async function refresh() {
  loading.value = true
  try {
    let res
    switch (active.value) {
      case 'orders':
        res = await salesApi.orders()
        break
      case 'my-orders':
        res = await salesApi.myOrders()
        break
      case 'inquiries':
        res = await salesApi.inquiries()
        break
      case 'pre-ships':
        res = await salesApi.preShipments()
        break
      case 'deliveries':
        res = await salesApi.deliveries()
        break
      case 'price-locks':
        res = await salesApi.priceLocks()
        break
      case 'contracts':
        res = await salesApi.contracts()
        break
      case 'quotes':
        res = await salesApi.quoteHistories()
        break
      case 'rankings':
        res = await salesApi.rankings()
        list.value = ((res.data as { list?: Row[] })?.list) || []
        return
      case 'boms':
        res = await salesApi.salesBoms()
        break
      case 'budgets':
        res = await salesApi.costBudgets()
        break
      case 'prints':
        res = await salesApi.prints()
        break
      case 'self-orders':
        res = await salesApi.selfOrders()
        list.value = ((res.data as { rules?: Row[] })?.rules) || []
        return
      case 'calculator':
        res = await salesApi.quoteCalculator()
        list.value = []
        detail.value = (res.data as Row) || null
        return
      case 'outbound':
        router.replace('/sales/outbound-settle')
        return
      default:
        res = await salesApi.orders()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function linePayload(productId: number, qty: number, price: number) {
  return [{ product_id: productId, qty, price }]
}

async function createOrder() {
  const res = await salesApi.createOrder({
    customer_id: orderForm.customer_id,
    warehouse_id: orderForm.warehouse_id,
    remark: orderForm.remark,
    lines: linePayload(orderForm.product_id, orderForm.qty, orderForm.price),
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`订单已创建 ${(res.data as Row)?.doc_no}`)
  preShipForm.order_id = Number((res.data as Row)?.id)
  deliveryForm.order_id = Number((res.data as Row)?.id)
  budgetForm.order_id = Number((res.data as Row)?.id)
  printForm.doc_id = Number((res.data as Row)?.id)
  await refresh()
}

async function submitOrder(id: number) {
  const res = await salesApi.submitOrder(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已提交')
  await refresh()
}

async function cancelOrder(id: number) {
  await ElMessageBox.confirm('确认取消订单并释放库存占用？')
  const res = await salesApi.cancelOrder(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已取消')
  await refresh()
}

async function rebuyOrder(id: number) {
  const res = await salesApi.rebuyOrder(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`复购成功 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function openOrder(id: number) {
  const res = await salesApi.getOrder(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

async function createInquiry() {
  const res = await salesApi.createInquiry({
    customer_id: inquiryForm.customer_id,
    remark: inquiryForm.remark,
    lines: [{ product_id: inquiryForm.product_id, qty: inquiryForm.qty, quote_price: inquiryForm.quote_price }],
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`询价单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function approveInquiry(id: number) {
  const res = await salesApi.approveInquiry(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('询价已审批')
  await refresh()
}

async function inquiryToOrder(id: number) {
  const res = await salesApi.inquiryToOrder(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已转订单 ${((res.data as Row)?.order as Row)?.doc_no}`)
  await refresh()
}

async function createLock() {
  const res = await salesApi.createPriceLock({ ...lockForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('锁价已生效')
  await refresh()
}

async function createContract() {
  const res = await salesApi.createContract({ ...contractForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`合同 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function createPreShip() {
  if (!preShipForm.order_id) return ElMessage.warning('请填写订单ID')
  const res = await salesApi.createPreShip({ ...preShipForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`预发货 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function reservePre(id: number) {
  const res = await salesApi.reservePreShip(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已占用库存')
  await refresh()
}

async function confirmPre(id: number) {
  const res = await salesApi.confirmPreShip(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已确认并生成发货单')
  await refresh()
}

async function createDelivery() {
  if (!deliveryForm.order_id) return ElMessage.warning('请填写订单ID')
  const res = await salesApi.createDelivery({ order_id: deliveryForm.order_id })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`发货单 ${(res.data as Row)?.doc_no}`)
  await refresh()
}

async function approveDelivery(id: number) {
  const res = await salesApi.approveDelivery(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('发货已审批')
  await refresh()
}

async function shipDelivery(id: number) {
  const res = await salesApi.shipDelivery(id, { logistics_no: deliveryForm.logistics_no })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已出库发货')
  await refresh()
}

async function doCalc() {
  const res = await salesApi.calcQuote({ ...calcForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  calcResult.value = (res.data as Row) || null
}

async function applyCalc() {
  if (!calcResult.value) return
  const res = await salesApi.applyQuote({
    customer_id: orderForm.customer_id,
    ...calcForm,
    quote_price: calcResult.value.quote_price,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已写入历史报价')
}

async function createBom() {
  const res = await salesApi.createSalesBom({ product_id: bomForm.product_id, name: bomForm.name })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const id = Number((res.data as Row)?.id)
  await salesApi.saveSalesBomLines(id, {
    lines: [{ material_product_id: bomForm.material_product_id, qty: bomForm.qty }],
  })
  ElMessage.success('销售BOM已保存')
  await refresh()
}

async function createBudget() {
  if (!budgetForm.order_id) return ElMessage.warning('请填写订单ID')
  const res = await salesApi.createCostBudget({ ...budgetForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`毛利率 ${(((res.data as Row)?.margin as number) || 0 * 100).toFixed?.(2) ?? (res.data as Row)?.margin}`)
  await refresh()
}

async function doPrint() {
  if (!printForm.doc_id) return ElMessage.warning('请填写单据ID')
  const res = await salesApi.createPrint({ ...printForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = ((res.data as Row)?.preview as Row) || (res.data as Row)
  ElMessage.success('打印预览已生成')
  await refresh()
}

async function pickOrder(row: Row) {
  preShipForm.order_id = Number(row.id)
  deliveryForm.order_id = Number(row.id)
  budgetForm.order_id = Number(row.id)
  printForm.doc_id = Number(row.id)
  ElMessage.success(`已选用订单 ${row.doc_no}`)
}

async function submitSelf() {
  const res = await salesApi.submitSelfOrder({
    customer_id: selfForm.customer_id,
    source: 'self',
    lines: linePayload(selfForm.product_id, selfForm.qty, selfForm.price),
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`自助订单 ${(res.data as Row)?.doc_no}`)
}

watch(active, () => {
  detail.value = null
  refresh()
})

onMounted(async () => {
  await loadMeta()
  await refresh()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <p class="hint">工厂销售交付：客户 → 询价/锁价 → 订单占用 → 预发货 → 发货出库。成品默认仓=成品冷库(3)。</p>
    </div>

    <!-- 销售订单 / 修改 / 复购 / 我的订单 -->
    <template v-if="active==='orders' || active==='my-orders'">
      <el-card v-if="active==='orders'" header="新建销售订单" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户">
            <el-select v-model="orderForm.customer_id" style="width:180px">
              <el-option v-for="c in customers" :key="String(c.id)" :label="String(c.name)" :value="Number(c.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="产品">
            <el-select v-model="orderForm.product_id" style="width:160px">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="数量"><el-input-number v-model="orderForm.qty" :min="1" /></el-form-item>
          <el-form-item label="单价(0=锁价/牌价)"><el-input-number v-model="orderForm.price" :min="0" :step="0.1" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="orderForm.remark" /></el-form-item>
          <el-button type="primary" @click="createOrder">下单</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small" @row-click="(r: Row) => openOrder(Number(r.id))">
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="customer_name" label="客户" width="140" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="source" label="来源" width="90" />
        <el-table-column prop="total_amount" label="金额" width="100" />
        <el-table-column prop="created_at" label="时间" min-width="150" />
        <el-table-column v-if="active==='orders'" label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click.stop="submitOrder(Number(row.id))">提交</el-button>
            <el-button link @click.stop="rebuyOrder(Number(row.id))">复购</el-button>
            <el-button link type="danger" @click.stop="cancelOrder(Number(row.id))">取消</el-button>
            <el-button link @click.stop="pickOrder(row)">选用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 询价 -->
    <template v-else-if="active==='inquiries'">
      <el-card header="新建询价" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户">
            <el-select v-model="inquiryForm.customer_id" style="width:180px">
              <el-option v-for="c in customers" :key="String(c.id)" :label="String(c.name)" :value="Number(c.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="产品">
            <el-select v-model="inquiryForm.product_id" style="width:160px">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="数量"><el-input-number v-model="inquiryForm.qty" :min="1" /></el-form-item>
          <el-form-item label="报价"><el-input-number v-model="inquiryForm.quote_price" :min="0" :step="0.1" /></el-form-item>
          <el-button type="primary" @click="createInquiry">保存</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="customer_name" label="客户" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="created_at" label="时间" width="160" />
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button link type="success" @click="approveInquiry(Number(row.id))">审批</el-button>
            <el-button link type="primary" @click="inquiryToOrder(Number(row.id))">转订单</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 预发货 -->
    <template v-else-if="active==='pre-ships'">
      <el-card header="新建预发货" class="mb">
        <el-form inline size="small">
          <el-form-item label="订单ID"><el-input-number v-model="preShipForm.order_id" :min="1" /></el-form-item>
          <el-form-item label="计划发货日"><el-input v-model="preShipForm.plan_ship_date" /></el-form-item>
          <el-button type="primary" @click="createPreShip">创建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="order_no" label="订单" width="160" />
        <el-table-column prop="plan_ship_date" label="计划日" width="120" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="reserved" label="已占用" width="90" />
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button link @click="reservePre(Number(row.id))">占用</el-button>
            <el-button link type="primary" @click="confirmPre(Number(row.id))">确认→发货单</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 发货审批 -->
    <template v-else-if="active==='deliveries'">
      <el-card header="新建发货单" class="mb">
        <el-form inline size="small">
          <el-form-item label="订单ID"><el-input-number v-model="deliveryForm.order_id" :min="1" /></el-form-item>
          <el-form-item label="物流单号"><el-input v-model="deliveryForm.logistics_no" /></el-form-item>
          <el-button type="primary" @click="createDelivery">创建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="order_no" label="订单" width="160" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="logistics_no" label="物流" />
        <el-table-column prop="shipped_at" label="发货时间" width="160" />
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button link type="success" @click="approveDelivery(Number(row.id))">审批</el-button>
            <el-button link type="primary" @click="shipDelivery(Number(row.id))">出库发货</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 锁价 -->
    <template v-else-if="active==='price-locks'">
      <el-card header="新建锁价" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户">
            <el-select v-model="lockForm.customer_id" style="width:180px">
              <el-option v-for="c in customers" :key="String(c.id)" :label="String(c.name)" :value="Number(c.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="产品">
            <el-select v-model="lockForm.product_id" style="width:160px">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="锁价"><el-input-number v-model="lockForm.lock_price" :min="0" :step="0.1" /></el-form-item>
          <el-form-item label="起"><el-input v-model="lockForm.effective_from" style="width:120px" /></el-form-item>
          <el-form-item label="止"><el-input v-model="lockForm.effective_to" style="width:120px" /></el-form-item>
          <el-button type="primary" @click="createLock">保存</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="customer_name" label="客户" />
        <el-table-column prop="product_name" label="产品" />
        <el-table-column prop="lock_price" label="锁价" width="100" />
        <el-table-column prop="effective_from" label="起" width="120" />
        <el-table-column prop="effective_to" label="止" width="120" />
        <el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <!-- 合同 -->
    <template v-else-if="active==='contracts'">
      <el-card header="新建合同" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户">
            <el-select v-model="contractForm.customer_id" style="width:180px">
              <el-option v-for="c in customers" :key="String(c.id)" :label="String(c.name)" :value="Number(c.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="标题"><el-input v-model="contractForm.title" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="contractForm.amount" :min="0" /></el-form-item>
          <el-button type="primary" @click="createContract">保存</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="合同号" width="160" />
        <el-table-column prop="customer_name" label="客户" />
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="amount" label="金额" width="120" />
        <el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <!-- 历史报价 / 排行榜 / BOM / 预算 / 打印 / 自助 / 计算器 -->
    <template v-else-if="active==='quotes'">
      <el-table :data="list" size="small">
        <el-table-column prop="customer_name" label="客户" />
        <el-table-column prop="product_name" label="产品" />
        <el-table-column prop="price" label="报价" width="100" />
        <el-table-column prop="quoted_at" label="时间" width="180" />
        <el-table-column prop="inquiry_id" label="询价ID" width="90" />
        <el-table-column prop="order_id" label="订单ID" width="90" />
      </el-table>
    </template>

    <template v-else-if="active==='rankings'">
      <el-table :data="list" size="small">
        <el-table-column prop="rank" label="排名" width="80" />
        <el-table-column prop="customer_name" label="客户" />
        <el-table-column prop="order_count" label="订单数" width="100" />
        <el-table-column prop="amount" label="销售额" width="120" />
      </el-table>
    </template>

    <template v-else-if="active==='boms'">
      <el-card header="新建销售BOM" class="mb">
        <el-form inline size="small">
          <el-form-item label="成品">
            <el-select v-model="bomForm.product_id" style="width:160px">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="名称"><el-input v-model="bomForm.name" /></el-form-item>
          <el-form-item label="原料ID"><el-input-number v-model="bomForm.material_product_id" :min="1" /></el-form-item>
          <el-form-item label="用量"><el-input-number v-model="bomForm.qty" :min="0" :step="0.1" /></el-form-item>
          <el-button type="primary" @click="createBom">保存</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="160" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="product_name" label="成品" />
        <el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <template v-else-if="active==='budgets'">
      <el-card header="成本预算测算" class="mb">
        <el-form inline size="small">
          <el-form-item label="订单ID"><el-input-number v-model="budgetForm.order_id" :min="1" /></el-form-item>
          <el-form-item label="材料"><el-input-number v-model="budgetForm.material_cost" :min="0" /></el-form-item>
          <el-form-item label="人工"><el-input-number v-model="budgetForm.labor_cost" :min="0" /></el-form-item>
          <el-form-item label="其他"><el-input-number v-model="budgetForm.other_cost" :min="0" /></el-form-item>
          <el-button type="primary" @click="createBudget">测算</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="order_id" label="订单" width="90" />
        <el-table-column prop="sale_amount" label="销售额" width="100" />
        <el-table-column prop="total_cost" label="成本" width="100" />
        <el-table-column prop="margin" label="毛利率" width="100" />
        <el-table-column prop="created_at" label="时间" />
      </el-table>
    </template>

    <template v-else-if="active==='calculator'">
      <el-card header="报价试算">
        <el-form inline size="small">
          <el-form-item label="产品">
            <el-select v-model="calcForm.product_id" style="width:160px">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="数量"><el-input-number v-model="calcForm.qty" :min="1" /></el-form-item>
          <el-form-item label="成本"><el-input-number v-model="calcForm.base_cost" :min="0" :step="0.1" /></el-form-item>
          <el-form-item label="毛利率"><el-input-number v-model="calcForm.margin_rate" :min="0" :max="1" :step="0.01" /></el-form-item>
          <el-button type="primary" @click="doCalc">试算</el-button>
          <el-button @click="applyCalc" :disabled="!calcResult">回写报价</el-button>
        </el-form>
        <el-descriptions v-if="calcResult" :column="3" border style="margin-top:12px">
          <el-descriptions-item label="单价">{{ calcResult.quote_price }}</el-descriptions-item>
          <el-descriptions-item label="金额">{{ calcResult.amount }}</el-descriptions-item>
          <el-descriptions-item label="成本">{{ calcResult.base_cost }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </template>

    <template v-else-if="active==='prints'">
      <el-card header="单据打印预览" class="mb">
        <el-form inline size="small">
          <el-form-item label="类型">
            <el-select v-model="printForm.doc_type" style="width:140px">
              <el-option label="销售订单" value="sales_order" />
              <el-option label="预发货" value="pre_shipment" />
              <el-option label="发货单" value="delivery" />
            </el-select>
          </el-form-item>
          <el-form-item label="单据ID"><el-input-number v-model="printForm.doc_id" :min="1" /></el-form-item>
          <el-button type="primary" @click="doPrint">生成预览</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_type" label="类型" width="120" />
        <el-table-column prop="doc_no" label="单号" />
        <el-table-column prop="printed_at" label="打印时间" width="180" />
      </el-table>
    </template>

    <template v-else-if="active==='self-orders'">
      <el-card header="客户自助下单（代录）" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户">
            <el-select v-model="selfForm.customer_id" style="width:180px">
              <el-option v-for="c in customers" :key="String(c.id)" :label="String(c.name)" :value="Number(c.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="产品">
            <el-select v-model="selfForm.product_id" style="width:160px">
              <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="数量"><el-input-number v-model="selfForm.qty" :min="1" /></el-form-item>
          <el-button type="primary" @click="submitSelf">提交自助单</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="name" label="规则" />
        <el-table-column prop="min_qty" label="最小量" width="100" />
        <el-table-column prop="enabled" label="启用" width="80" />
        <el-table-column prop="remark" label="说明" />
      </el-table>
    </template>

    <el-card v-if="detail" header="明细" style="margin-top:16px">
      <pre class="detail">{{ JSON.stringify(detail, null, 2) }}</pre>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.head h2 { margin: 0 0 4px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
.mb { margin-bottom: 12px; }
.detail { background: #f6f8fa; padding: 12px; border-radius: 8px; max-height: 360px; overflow: auto; font-size: 12px; }
</style>
