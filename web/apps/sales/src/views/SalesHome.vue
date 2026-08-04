<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, salesApi, crmApi, portalHomeUrl } from '@erp/shared'
import { showToast } from 'vant'

const auth = useAuthStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const tab = ref('orders')
const orders = ref<Record<string, unknown>[]>([])
const inquiries = ref<Record<string, unknown>[]>([])
const customers = ref<Record<string, unknown>[]>([])
const orderForm = reactive({ customer: '', product: '', qty: 1 })
const inquiryForm = reactive({ customer: '', product: '', qty: 1, remark: '' })

async function boot() {
  if (!auth.isLoggedIn) await auth.login('admin', 'admin123', 'mp_sales')
  const [o, i, c] = await Promise.all([salesApi.orders(), salesApi.inquiries(), crmApi.customers()])
  orders.value = ((o.data as { list?: Record<string, unknown>[] })?.list) || []
  inquiries.value = ((i.data as { list?: Record<string, unknown>[] })?.list) || []
  customers.value = ((c.data as { list?: Record<string, unknown>[] })?.list) || []
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

async function rebuy(row: Record<string, unknown>) {
  const res = await salesApi.createOrder({
    customer: row.customer || '复购客户',
    rebuy_from: row.id,
    lines: (row.lines as unknown[]) || [{ product_id: 1, qty: 1 }],
  })
  showToast(res.code === 1 ? '复购成功' : res.msg)
  await boot()
}

onMounted(boot)
</script>

<template>
  <div class="phone">
    <header>
      <a :href="portalUrl" style="color:#fff;font-size:12px;opacity:.9">← 入口</a>
      <h1>{{ { orders: '我的订单', inquiry: '询价', crm: '客户跟进', remind: '任务提醒' }[tab] }}</h1>
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
      <section v-else-if="tab==='crm'" class="card">
        <van-cell-group inset title="客户（受保护/隐藏约束）">
          <van-cell v-for="c in customers" :key="String(c.id)" :title="String(c.name||c.code||c.id)" :label="String(c.status||'')" />
        </van-cell-group>
      </section>
      <section v-else class="card">
        <van-notice-bar text="跟进提醒：今日需回访重点客户；报价计算器请在后台销售模块使用。" />
        <van-cell title="报价计算器" is-link />
        <van-cell title="任务提醒" value="系统推送" />
      </section>
    </main>
    <nav class="tabbar">
      <button v-for="t in [['orders','订单'],['inquiry','询价'],['crm','跟进'],['remind','提醒']]" :key="t[0]"
        :class="{ active: tab===t[0] }" @click="tab=t[0]">{{ t[1] }}</button>
    </nav>
  </div>
</template>

<style scoped>
.phone { max-width: 420px; margin: 0 auto; min-height: 100vh; background: #f5f6f8; display: flex; flex-direction: column; }
header { display: flex; justify-content: space-between; align-items: center; padding: 12px 14px; background: #0d7a6f; color: #fff; }
header h1 { margin: 0; font-size: 16px; }
.out { background: transparent; border: 0; color: #fff; }
main { flex: 1; padding: 12px; }
.card { background: #fff; border-radius: 10px; padding: 12px; }
.tabbar { display: grid; grid-template-columns: repeat(4,1fr); background: #fff; border-top: 1px solid #e5e5e5; }
.tabbar button { border: 0; background: transparent; padding: 10px 0; font-size: 12px; color: #666; }
.tabbar button.active { color: #0d7a6f; font-weight: 600; }
</style>
