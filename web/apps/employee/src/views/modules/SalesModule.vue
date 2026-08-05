<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  useAuthStore,
  useNotifyStore,
  salesApi,
  crmApi,
  canAccessEmployeeModule,
  portalHomeUrl,
} from '@erp/shared'
import { showToast } from 'vant'

const auth = useAuthStore()
const notify = useNotifyStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const tab = ref('orders')
const orders = ref<Record<string, unknown>[]>([])
const inquiries = ref<Record<string, unknown>[]>([])
const customers = ref<Record<string, unknown>[]>([])
const settles = ref<Record<string, unknown>[]>([])
const orderForm = reactive({ customer: '', product: '', qty: 1 })
const inquiryForm = reactive({ customer: '', product: '', qty: 1, remark: '' })
const settleForm = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  product_name: '',
  plate_no: '',
  weight: 0,
  unit_price: 0,
  freight_fee: 0,
  loading_fee: 0,
  weigh_fee: 0,
})

async function boot() {
  if (!auth.isLoggedIn) {
    router.replace('/login')
    return
  }
  await auth.fetchMe()
  if (!canAccessEmployeeModule('sales', auth.permissions, auth.roles)) {
    showToast('无销售模块权限')
    router.replace('/')
    return
  }
  await notify.start()
  const [o, i, c, s] = await Promise.all([
    salesApi.orders(),
    salesApi.inquiries(),
    crmApi.customers(),
    salesApi.outboundSettles(),
  ])
  orders.value = ((o.data as { list?: Record<string, unknown>[] })?.list) || []
  inquiries.value = ((i.data as { list?: Record<string, unknown>[] })?.list) || []
  customers.value = ((c.data as { list?: Record<string, unknown>[] })?.list) || []
  settles.value = ((s.data as { list?: Record<string, unknown>[] })?.list) || []
}

async function createOrder() {
  const res = await salesApi.createOrder({ ...orderForm, lines: [{ product_id: 1, qty: orderForm.qty }] })
  if (res.code !== 1) return showToast(res.msg)
  showToast('订单已创建')
  await boot()
}

async function createInquiry() {
  const res = await salesApi.createInquiry({ ...inquiryForm })
  if (res.code !== 1) return showToast(res.msg)
  showToast('询价已提交')
  await boot()
}

async function createSettle() {
  const res = await salesApi.createOutboundSettle({ ...settleForm, qty: settleForm.weight, unit: 'kg' })
  if (res.code !== 1) return showToast(res.msg)
  showToast(`出厂结算已录 ¥${(res.data as Record<string, unknown>)?.amount ?? 0}`)
  await boot()
}

async function rebuy(row: Record<string, unknown>) {
  const res = await salesApi.createOrder({
    customer: row.customer || '复购客户',
    rebuy_from: row.id,
    lines: (row.lines as unknown[]) || [{ product_id: 1, qty: 1 }],
  })
  showToast(res.code === 1 ? '复购成功' : res.msg)
  await boot()
}

watch(tab, (t) => {
  if (t === 'remind') void notify.refresh()
  if (t === 'settle') void boot()
})

onMounted(boot)
</script>

<template>
  <div class="phone">
    <header>
      <a :href="portalUrl" style="color:#fff;font-size:12px;opacity:.9">← 入口</a>
      <button type="button" class="mod" @click="router.push('/')">模块</button>
      <h1>{{ { orders: '我的订单', inquiry: '询价', settle: '出厂结算', crm: '客户跟进', remind: '任务提醒' }[tab] }}</h1>
      <button class="out" @click="auth.logout(); router.replace('/login')">退出</button>
    </header>
    <main>
      <section v-if="tab==='orders'" class="card">
        <van-field v-model="orderForm.customer" label="客户" />
        <van-field v-model.number="orderForm.qty" type="number" label="数量" />
        <van-button type="primary" block @click="createOrder">新建订单</van-button>
        <van-cell-group inset title="订单列表" style="margin-top:12px">
          <van-cell v-for="o in orders" :key="String(o.id)" :title="`#${o.id} ${o.doc_no||''}`" :value="String(o.status||'')">
            <template #right-icon>
              <van-button size="mini" @click="rebuy(o)">复购</van-button>
            </template>
          </van-cell>
        </van-cell-group>
      </section>
      <section v-else-if="tab==='inquiry'" class="card">
        <van-field v-model="inquiryForm.customer" label="客户" />
        <van-field v-model="inquiryForm.product" label="产品" />
        <van-field v-model="inquiryForm.remark" label="备注" />
        <van-button type="primary" block @click="createInquiry">提交询价</van-button>
        <van-cell-group inset title="询价记录" style="margin-top:12px">
          <van-cell v-for="i in inquiries" :key="String(i.id)" :title="`#${i.id}`" :label="String(i.product||i.customer||'')" />
        </van-cell-group>
      </section>
      <section v-else-if="tab==='settle'" class="card">
        <p class="hint">补录出厂结算；金额=重量×单价+运/装/磅费</p>
        <van-field v-model="settleForm.product_name" label="产品" />
        <van-field v-model="settleForm.plate_no" label="车牌" />
        <van-field v-model.number="settleForm.weight" type="number" label="重量" />
        <van-field v-model.number="settleForm.unit_price" type="number" label="单价" />
        <van-field v-model.number="settleForm.freight_fee" type="number" label="运费" />
        <van-field v-model.number="settleForm.loading_fee" type="number" label="装卸" />
        <van-field v-model.number="settleForm.weigh_fee" type="number" label="过磅费" />
        <van-button type="primary" block @click="createSettle">保存补录</van-button>
        <van-cell-group inset title="出厂结算单" style="margin-top:12px">
          <van-cell
            v-for="s in settles"
            :key="String(s.id)"
            :title="String(s.doc_no)"
            :label="`${s.product_name||''} · ${s.plate_no||''}`"
            :value="`¥${s.amount??0}`"
          />
        </van-cell-group>
      </section>
      <section v-else-if="tab==='crm'" class="card">
        <van-cell-group inset title="客户">
          <van-cell v-for="c in customers" :key="String(c.id)" :title="String(c.name||c.code||c.id)" :label="String(c.status||'')" />
        </van-cell-group>
      </section>
      <section v-else class="card">
        <p class="hint">收件箱提醒 · MQTT {{ notify.mqttStatus }} · 未读 {{ notify.unread }}</p>
        <van-button size="small" @click="notify.refresh()">刷新</van-button>
        <van-empty v-if="!notify.inbox.length" description="暂无提醒" />
        <van-cell-group v-else inset style="margin-top:8px">
          <van-cell
            v-for="n in notify.inbox"
            :key="String(n.id)"
            :title="String(n.title || n.event_key)"
            :label="String(n.body || '')"
            :value="n.read_at ? '已读' : '未读'"
            @click="notify.markRead(Number(n.id))"
          />
        </van-cell-group>
      </section>
    </main>
    <nav class="tabbar">
      <button
        v-for="t in [['orders','订单'],['inquiry','询价'],['settle','出厂'],['crm','跟进'],['remind','提醒']]"
        :key="t[0]"
        :class="{ active: tab===t[0] }"
        @click="tab=t[0]"
      >{{ t[1] }}</button>
    </nav>
  </div>
</template>

<style scoped>
.phone { max-width: 420px; margin: 0 auto; min-height: 100vh; background: #f5f6f8; display: flex; flex-direction: column; }
header { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 12px 14px; background: #0d7a6f; color: #fff; }
header h1 { margin: 0; font-size: 16px; flex: 1; }
.mod, .out { background: transparent; border: 0; color: #fff; cursor: pointer; }
main { flex: 1; padding: 12px; }
.card { background: #fff; border-radius: 10px; padding: 12px; }
.hint { font-size: 12px; color: #667; margin: 0 0 8px; }
.tabbar { display: grid; grid-template-columns: repeat(5,1fr); background: #fff; border-top: 1px solid #e5e5e5; }
.tabbar button { border: 0; background: transparent; padding: 10px 0; font-size: 11px; color: #666; }
.tabbar button.active { color: #0d7a6f; font-weight: 600; }
</style>
