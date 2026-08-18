<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  productApi,
  salesApi,
  useAuthStore,
  portalHomeUrl,
  statusLabel,
  statusType,
  money,
  dash,
  type PageData,
} from '@erp/shared'

type Row = Record<string, unknown>
type Product = { id: number; name: string; sale_price?: number; product_type?: string }

const ERR: Record<string, string> = {
  SELF_ORDER_MIN_QTY: '未达到最小下单数量',
  SELF_ORDER_MAX_QTY: '超过最大下单数量',
  SELF_ORDER_MAX_AMOUNT: '超过最大下单金额',
  PERM_DENIED: '无权操作',
  CUSTOMER_NOT_BOUND: '账号未绑定客户',
  LINES_REQUIRED: '请选择产品和数量',
  ONLY_DRAFT_EDITABLE: '仅草稿或已驳回可编辑',
  INVALID_STATUS: '当前状态不可操作',
  NOT_FOUND: '单据不存在',
}

function failMsg(msg?: string) {
  const m = String(msg || '')
  return ERR[m] || m || '操作失败'
}

const SECTIONS = [
  { key: 'inquiries', label: '询价' },
  { key: 'order', label: '下单' },
  { key: 'orders', label: '订单' },
  { key: 'progress', label: '进度' },
  { key: 'quotes', label: '报价' },
  { key: 'contracts', label: '合同' },
] as const

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const portalUrl = portalHomeUrl()

const active = computed(() => {
  const s = String(route.params.section || '')
  if (SECTIONS.some((x) => x.key === s)) return s
  return 'inquiries'
})

const customerName = computed(
  () => auth.user?.customer_name || auth.user?.name || auth.user?.login_name || '客户',
)

const loading = ref(false)
const products = ref<Product[]>([])
const inquiries = ref<Row[]>([])
const orders = ref<Row[]>([])
const deliveries = ref<Row[]>([])
const quotes = ref<Row[]>([])
const locks = ref<Row[]>([])
const contracts = ref<Row[]>([])
const selfRules = ref<Row[]>([])
const detail = ref<Row | null>(null)
const drawer = ref(false)

const inquiryForm = reactive({ product_id: null as number | null, qty: 100, remark: '' })
const orderForm = reactive({ product_id: null as number | null, qty: 50, price: 0 })
const calcForm = reactive({ product_id: 3, qty: 100, base_cost: 4, margin_rate: 0.2 })
const calcResult = ref<Row | null>(null)

const pendingInquiry = computed(
  () => inquiries.value.filter((r) => ['draft', 'pending', 'submitted'].includes(String(r.status))).length,
)
const pendingShip = computed(
  () => orders.value.filter((r) => ['submitted', 'approved', 'open'].includes(String(r.status))).length,
)

const list = computed(() => {
  switch (active.value) {
    case 'inquiries':
      return inquiries.value
    case 'orders':
      return orders.value
    case 'progress':
      return deliveries.value
    case 'quotes':
      return quotes.value
    case 'contracts':
      return contracts.value
    default:
      return []
  }
})

const hint = computed(() => {
  const map: Record<string, string> = {
    inquiries: '新建询价后提交，等待工厂审批。通过后可自助下单。',
    order: '选择产品与数量提交。规则校验失败会提示原因。',
    orders: '查看本客户订单，可一键复购。',
    progress: '发货单状态、物流与签收时间，只读。',
    quotes: '历史报价可钻取询价/订单；试算可回写本客户报价。',
    contracts: '生效中的锁价与合同只读，规则仍由工厂维护。',
  }
  return map[active.value] || ''
})

function go(key: string) {
  router.replace(`/shop/${key}`)
}

function pageList(res: { data?: PageData<Row> | Row | { list?: Row[]; rules?: Row[] } }) {
  const d = res.data as { list?: Row[]; rules?: Row[] } | undefined
  return d?.list || d?.rules || []
}

async function loadProducts() {
  const r = await productApi.list()
  const raw = ((r.data as { list?: Product[] })?.list) || []
  products.value = raw.filter((p) => p.product_type === 'finished' || Number(p.sale_price || 0) > 0)
  if (!products.value.length) products.value = raw
}

async function refresh() {
  loading.value = true
  try {
    const [iq, od, dl, qh, lk, ct, so] = await Promise.all([
      salesApi.inquiries(),
      salesApi.myOrders(),
      salesApi.deliveries(),
      salesApi.quoteHistories(),
      salesApi.priceLocks('status=active'),
      salesApi.contracts(),
      salesApi.selfOrders(),
    ])
    inquiries.value = pageList(iq)
    orders.value = pageList(od)
    deliveries.value = pageList(dl)
    quotes.value = pageList(qh)
    locks.value = pageList(lk)
    contracts.value = pageList(ct)
    selfRules.value = ((so.data as { rules?: Row[] })?.rules) || []
  } finally {
    loading.value = false
  }
}

async function openDetail(kind: string, id: number) {
  let res
  if (kind === 'inquiry') res = await salesApi.getInquiry(id)
  else if (kind === 'order') res = await salesApi.getOrder(id)
  else if (kind === 'delivery') res = await salesApi.getDelivery(id)
  else if (kind === 'contract') res = await salesApi.getContract(id)
  else return
  if (res.code !== 1) return ElMessage.error(failMsg(res.msg))
  detail.value = { ...(res.data as Row), _kind: kind }
  drawer.value = true
}

async function submitInquiry() {
  if (!inquiryForm.product_id) return ElMessage.warning('请选择产品')
  const r = await salesApi.createInquiry({
    product_id: inquiryForm.product_id,
    lines: [{ product_id: inquiryForm.product_id, qty: inquiryForm.qty }],
    remark: inquiryForm.remark,
    source: 'portal',
  })
  if (r.code !== 1) return ElMessage.error(failMsg(r.msg))
  const id = Number((r.data as Row)?.id || 0)
  if (id) {
    const s = await salesApi.submitInquiry(id)
    if (s.code !== 1) ElMessage.warning('已保存草稿，提交失败：' + failMsg(s.msg))
    else ElMessage.success('询价已提交')
  }
  inquiryForm.remark = ''
  await refresh()
}

async function withdrawInquiry(id: number) {
  const r = await salesApi.withdrawInquiry(id)
  if (r.code !== 1) return ElMessage.error(failMsg(r.msg))
  ElMessage.success('已撤回')
  await refresh()
}

async function submitSelfOrder() {
  if (!orderForm.product_id) return ElMessage.warning('请选择产品')
  const r = await salesApi.submitSelfOrder({
    product_id: orderForm.product_id,
    lines: [{ product_id: orderForm.product_id, qty: orderForm.qty, price: orderForm.price }],
  })
  if (r.code !== 1) return ElMessage.error(failMsg(r.msg))
  ElMessage.success('下单成功')
  router.replace('/shop/orders')
  await refresh()
}

async function rebuy(id: number) {
  const r = await salesApi.rebuyOrder(id)
  if (r.code !== 1) return ElMessage.error(failMsg(r.msg))
  ElMessage.success('已生成复购单')
  await refresh()
}

async function calcQuote() {
  const r = await salesApi.calcQuote({ ...calcForm })
  if (r.code !== 1) return ElMessage.error(failMsg(r.msg))
  calcResult.value = (r.data as Row) || null
}

async function applyQuote() {
  if (!calcResult.value) return ElMessage.warning('请先试算')
  const r = await salesApi.applyQuote({
    ...calcForm,
    quote_price: calcResult.value.quote_price,
  })
  if (r.code !== 1) return ElMessage.error(failMsg(r.msg))
  ElMessage.success('已写入历史报价')
  await refresh()
}

function logout() {
  auth.logout()
  router.replace('/shop/login')
}

watch(
  () => route.params.section,
  () => {
    if (!String(route.params.section || '')) router.replace('/shop/inquiries')
  },
  { immediate: true },
)

onMounted(async () => {
  if (!auth.user) await auth.fetchMe()
  await Promise.all([loadProducts(), refresh()])
})
</script>

<template>
  <div class="shop">
    <header class="top">
      <a :href="portalUrl" class="back">入口</a>
      <div class="who">
        <strong>{{ customerName }}</strong>
        <span>客户自助 · 仅本客户单据</span>
      </div>
      <el-button link type="danger" @click="logout">退出</el-button>
    </header>

    <section class="stats">
      <div class="kpi warn">
        <div class="l">待审询价</div>
        <div class="v">{{ pendingInquiry }}</div>
      </div>
      <div class="kpi">
        <div class="l">待发货订单</div>
        <div class="v">{{ pendingShip }}</div>
      </div>
      <div class="kpi ok">
        <div class="l">我的订单</div>
        <div class="v">{{ orders.length }}</div>
      </div>
    </section>

    <nav class="tabs">
      <button
        v-for="s in SECTIONS"
        :key="s.key"
        :class="{ on: active === s.key }"
        @click="go(s.key)"
      >{{ s.label }}</button>
    </nav>

    <p class="hint">{{ hint }}</p>

    <main class="body" v-loading="loading">
      <template v-if="active === 'inquiries'">
        <el-form class="form" @submit.prevent="submitInquiry">
          <el-select v-model="inquiryForm.product_id" placeholder="选择产品" filterable>
            <el-option v-for="p in products" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
          <el-input-number v-model="inquiryForm.qty" :min="1" :step="10" />
          <el-input v-model="inquiryForm.remark" placeholder="备注（可选）" />
          <el-button type="primary" native-type="submit">提交询价</el-button>
        </el-form>
        <el-empty v-if="!inquiries.length" description="暂无询价" />
        <ul class="cards">
          <li v-for="r in inquiries" :key="String(r.id)" @click="openDetail('inquiry', Number(r.id))">
            <div class="row">
              <b>{{ dash(r.doc_no) }}</b>
              <el-tag size="small" :type="statusType(r.status)">{{ statusLabel(r.status) }}</el-tag>
            </div>
            <div class="meta">{{ dash(r.created_at) }}</div>
            <el-button
              v-if="r.status === 'pending'"
              size="small"
              @click.stop="withdrawInquiry(Number(r.id))"
            >撤回</el-button>
          </li>
        </ul>
      </template>

      <template v-else-if="active === 'order'">
        <el-alert
          v-if="selfRules.length"
          :title="'规则：' + selfRules.map((x) => `${x.name} 最小${x.min_qty || 0} 最大${x.max_qty || '不限'} 限额${x.max_amount || '不限'}`).join('；')"
          type="info"
          :closable="false"
          class="mb"
        />
        <el-form class="form col" @submit.prevent="submitSelfOrder">
          <el-form-item label="产品">
            <el-select v-model="orderForm.product_id" placeholder="选择产品" filterable style="width:100%">
              <el-option v-for="p in products" :key="p.id" :label="`${p.name} ¥${money(p.sale_price)}`" :value="p.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="数量">
            <el-input-number v-model="orderForm.qty" :min="1" :step="10" />
          </el-form-item>
          <el-form-item label="意向单价（0 则取锁价/牌价）">
            <el-input-number v-model="orderForm.price" :min="0" :step="0.1" :precision="2" />
          </el-form-item>
          <el-button type="primary" native-type="submit">提交自助订单</el-button>
        </el-form>
      </template>

      <template v-else-if="active === 'orders'">
        <el-empty v-if="!orders.length" description="暂无订单" />
        <ul class="cards">
          <li v-for="r in orders" :key="String(r.id)" @click="openDetail('order', Number(r.id))">
            <div class="row">
              <b>{{ dash(r.doc_no) }}</b>
              <el-tag size="small" :type="statusType(r.status)">{{ statusLabel(r.status) }}</el-tag>
            </div>
            <div class="meta">¥{{ money(r.total_amount) }} · {{ dash(r.created_at) }}</div>
            <el-button size="small" type="primary" plain @click.stop="rebuy(Number(r.id))">一键复购</el-button>
          </li>
        </ul>
      </template>

      <template v-else-if="active === 'progress'">
        <el-empty v-if="!deliveries.length" description="暂无发货进度" />
        <ul class="cards">
          <li v-for="r in deliveries" :key="String(r.id)" @click="openDetail('delivery', Number(r.id))">
            <div class="row">
              <b>{{ dash(r.doc_no) }}</b>
              <el-tag size="small" :type="statusType(r.status)">{{ statusLabel(r.status) }}</el-tag>
            </div>
            <div class="meta">订单 {{ dash(r.order_no) }} · 物流 {{ dash(r.logistics_no) }}</div>
            <div class="meta">发货 {{ dash(r.shipped_at) }} · 签收 {{ dash(r.received_at) }}</div>
          </li>
        </ul>
      </template>

      <template v-else-if="active === 'quotes'">
        <el-form class="form" @submit.prevent="calcQuote">
          <el-select v-model="calcForm.product_id" placeholder="产品">
            <el-option v-for="p in products" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
          <el-input-number v-model="calcForm.qty" :min="1" :step="10" />
          <el-input-number v-model="calcForm.margin_rate" :min="0" :max="1" :step="0.05" :precision="2" />
          <el-button native-type="submit">试算</el-button>
          <el-button type="primary" :disabled="!calcResult" @click="applyQuote">回写报价</el-button>
        </el-form>
        <p v-if="calcResult" class="calc">试算单价 ¥{{ money(calcResult.quote_price) }} · 金额 ¥{{ money(calcResult.amount) }}</p>
        <el-empty v-if="!quotes.length" description="暂无历史报价" />
        <ul class="cards">
          <li
            v-for="r in quotes"
            :key="String(r.id)"
            @click="Number(r.inquiry_id) ? openDetail('inquiry', Number(r.inquiry_id)) : Number(r.order_id) ? openDetail('order', Number(r.order_id)) : undefined"
          >
            <div class="row">
              <b>{{ dash(r.product_name) }}</b>
              <span>¥{{ money(r.price) }}</span>
            </div>
            <div class="meta">{{ dash(r.quoted_at) }} · 询价 {{ dash(r.inquiry_id) }} · 订单 {{ dash(r.order_id) }}</div>
          </li>
        </ul>
      </template>

      <template v-else>
        <h3 class="sub">生效锁价</h3>
        <el-empty v-if="!locks.length" description="暂无锁价" />
        <ul class="cards">
          <li v-for="r in locks" :key="'l'+r.id">
            <div class="row">
              <b>{{ dash(r.product_name) }}</b>
              <el-tag size="small" :type="statusType(r.status)">{{ statusLabel(r.status) }}</el-tag>
            </div>
            <div class="meta">¥{{ money(r.lock_price) }} · {{ dash(r.effective_from) }} ~ {{ dash(r.effective_to) }}</div>
          </li>
        </ul>
        <h3 class="sub">合同</h3>
        <el-empty v-if="!contracts.length" description="暂无合同" />
        <ul class="cards">
          <li v-for="r in contracts" :key="'c'+r.id" @click="openDetail('contract', Number(r.id))">
            <div class="row">
              <b>{{ dash(r.doc_no) }}</b>
              <el-tag size="small" :type="statusType(r.status)">{{ statusLabel(r.status) }}</el-tag>
            </div>
            <div class="meta">{{ dash(r.title) }} · ¥{{ money(r.amount) }} · 订单 {{ dash(r.order_id) }}</div>
          </li>
        </ul>
      </template>
    </main>

    <el-drawer v-model="drawer" :title="String(detail?.doc_no || '详情')" size="92%">
      <template v-if="detail">
        <p><el-tag :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag></p>
        <p v-if="detail.total_amount != null">金额 ¥{{ money(detail.total_amount) }}</p>
        <p v-if="detail.logistics_no">物流 {{ dash(detail.logistics_no) }}</p>
        <p v-if="detail.shipped_at">发货 {{ dash(detail.shipped_at) }}</p>
        <p v-if="detail.received_at">签收 {{ dash(detail.received_at) }}</p>
        <p v-if="detail.approval_chain" class="meta">{{ detail.approval_chain }}</p>
        <h4>明细</h4>
        <el-table v-if="Array.isArray(detail.lines)" :data="detail.lines as Row[]" size="small" stripe>
          <el-table-column prop="product_name" label="产品" />
          <el-table-column prop="qty" label="数量" width="80" />
          <el-table-column label="单价" width="90">
            <template #default="{ row }">{{ money(row.price || row.quote_price) }}</template>
          </el-table-column>
        </el-table>
        <h4>审批时间线</h4>
        <el-empty v-if="!Array.isArray(detail.approvals) || !(detail.approvals as Row[]).length" description="暂无审批记录" />
        <el-timeline v-else>
          <el-timeline-item
            v-for="a in (detail.approvals as Row[])"
            :key="String(a.id)"
            :timestamp="String(a.acted_at || a.created_at || '')"
          >
            {{ dash(a.title) }} · {{ statusLabel(a.status) }}
            <div v-if="a.comment" class="meta">{{ a.comment }}</div>
          </el-timeline-item>
        </el-timeline>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.shop {
  min-height: 100vh;
  padding: 12px 12px 24px;
  background: #f4f7f8;
  color: #1a2b34;
  font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  padding-bottom: 28px;
}
.top {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.back { color: #0d7a6f; font-size: 13px; text-decoration: none; }
.who { flex: 1; }
.who strong { display: block; font-size: 16px; }
.who span { font-size: 12px; color: #5c6b75; }
.stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}
.kpi {
  background: #fff;
  border-radius: 10px;
  padding: 10px;
  box-shadow: 0 4px 12px rgba(15, 28, 34, 0.05);
}
.kpi .l { font-size: 12px; color: #5c6b75; }
.kpi .v { font-size: 20px; font-weight: 700; }
.kpi.warn .v { color: #d48b16; }
.kpi.ok .v { color: #0d7a6f; }
.tabs {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  padding-bottom: 6px;
}
.tabs button {
  border: 0;
  background: #e8eef1;
  color: #3a4a52;
  padding: 8px 12px;
  border-radius: 999px;
  white-space: nowrap;
  cursor: pointer;
  font-size: 13px;
}
.tabs button.on { background: #0d7a6f; color: #fff; }
.hint { font-size: 13px; color: #5c6b75; margin: 8px 0 12px; }
.form { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; align-items: center; }
.form.col { flex-direction: column; align-items: stretch; }
.cards { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
.cards li {
  background: #fff;
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 4px 12px rgba(15, 28, 34, 0.05);
}
.row { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.meta { font-size: 12px; color: #7a8a94; margin-top: 4px; }
.sub { margin: 16px 0 8px; font-size: 14px; }
.calc { font-size: 14px; color: #0d7a6f; font-weight: 600; }
.mb { margin-bottom: 12px; }
h4 { margin: 16px 0 8px; font-size: 14px; }
@media (min-width: 720px) {
  .shop { max-width: 720px; margin: 0 auto; }
}
</style>
