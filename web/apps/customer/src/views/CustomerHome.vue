<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, salesApi, productApi, portalHomeUrl } from '@erp/shared'
import { showToast } from 'vant'

const auth = useAuthStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const tab = ref('order')
const products = ref<Record<string, unknown>[]>([])
const myOrders = ref<Record<string, unknown>[]>([])
const form = reactive({ product_id: 1, qty: 1, remark: '' })
const inquiry = reactive({ product: '', qty: 1, remark: '' })

async function boot() {
  if (!auth.isLoggedIn) await auth.login('admin', 'admin123', 'mp_sales')
  const [p, o] = await Promise.all([productApi.list(), salesApi.myOrders()])
  products.value = ((p.data as { list?: Record<string, unknown>[] })?.list) || []
  myOrders.value = ((o.data as { list?: Record<string, unknown>[] })?.list) || []
  if (!myOrders.value.length) {
    const all = await salesApi.orders()
    myOrders.value = ((all.data as { list?: Record<string, unknown>[] })?.list) || []
  }
  if (products.value[0]) form.product_id = Number(products.value[0].id)
}

async function placeOrder() {
  const res = await salesApi.createOrder({
    customer: auth.user?.login_name || 'customer',
    self_order: true,
    lines: [{ product_id: form.product_id, qty: form.qty }],
    remark: form.remark,
  })
  showToast(res.code === 1 ? '下单成功' : res.msg)
  await boot()
}

async function submitInquiry() {
  const res = await salesApi.createInquiry({ ...inquiry, customer: auth.user?.login_name })
  showToast(res.code === 1 ? '询价已提交' : res.msg)
}

async function rebuy(row: Record<string, unknown>) {
  const res = await salesApi.createOrder({
    customer: auth.user?.login_name,
    rebuy_from: row.id,
    lines: [{ product_id: form.product_id, qty: 1 }],
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
      <h1>{{ { order:'自助下单', inquiry:'询价管理', mine:'我的订单', rebuy:'订单复购', ship:'发货进度' }[tab] }}</h1>
      <button class="out" @click="auth.logout(); router.replace('/login')">退出</button>
    </header>
    <main>
      <section v-if="tab==='order'" class="card">
        <van-field label="商品">
          <template #input>
            <select v-model.number="form.product_id" style="width:100%;border:0;outline:0">
              <option v-for="p in products" :key="String(p.id)" :value="Number(p.id)">{{ p.code }} {{ p.name }}</option>
            </select>
          </template>
        </van-field>
        <van-field v-model.number="form.qty" type="number" label="数量" />
        <van-field v-model="form.remark" label="备注" />
        <van-button type="primary" block @click="placeOrder">提交订单</van-button>
      </section>
      <section v-else-if="tab==='inquiry'" class="card">
        <van-field v-model="inquiry.product" label="产品" />
        <van-field v-model.number="inquiry.qty" type="number" label="数量" />
        <van-button type="primary" block @click="submitInquiry">提交询价</van-button>
      </section>
      <section v-else-if="tab==='mine'" class="card">
        <van-cell v-for="o in myOrders" :key="String(o.id)" :title="`订单 #${o.id}`" :value="String(o.status||'')" :label="String(o.doc_no||'')" />
      </section>
      <section v-else-if="tab==='rebuy'" class="card">
        <van-cell v-for="o in myOrders" :key="'r'+o.id" :title="`历史 #${o.id}`" is-link @click="rebuy(o)" />
      </section>
      <section v-else class="card">
        <van-steps :active="1">
          <van-step>下单</van-step>
          <van-step>预发货</van-step>
          <van-step>发货审批</van-step>
          <van-step>物流在途</van-step>
        </van-steps>
        <van-cell title="系统物流查询" label="对接系统管理·物流信息" />
      </section>
    </main>
    <nav class="tabbar">
      <button v-for="t in [['order','下单'],['inquiry','询价'],['mine','订单'],['rebuy','复购'],['ship','发货']]" :key="t[0]"
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
.tabbar { display: grid; grid-template-columns: repeat(5,1fr); background: #fff; border-top: 1px solid #e5e5e5; }
.tabbar button { border: 0; background: transparent; padding: 10px 0; font-size: 12px; color: #666; }
.tabbar button.active { color: #0d7a6f; font-weight: 600; }
</style>
