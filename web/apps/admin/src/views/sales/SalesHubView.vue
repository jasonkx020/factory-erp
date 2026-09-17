<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'

type Row = Record<string, unknown>

const props = defineProps<{ section?: string }>()
const route = useRoute()
const router = useRouter()

const SECTIONS = [
  { key: 'customers', title: '客户档案', path: '/crm/customers' },
  { key: 'orders', title: '销售订单', path: '/sales/orders' },
  { key: 'deliveries', title: '销售出库', path: '/sales/deliveries' },
  { key: 'outbound-settles', title: '出厂结算', path: '/sales/outbound-settles' },
] as const

const section = computed(() => {
  const s = String(props.section || route.params.section || 'orders')
  return SECTIONS.some((x) => x.key === s) ? s : 'orders'
})

const loading = ref(false)
const list = ref<Row[]>([])

async function load() {
  loading.value = true
  try {
    const hit = SECTIONS.find((x) => x.key === section.value)
    if (!hit) return
    const res = await api.get<{ list?: Row[] }>(hit.path)
    if (res.code !== 1) {
      ElMessage.error(res.msg || '加载失败')
      list.value = []
      return
    }
    list.value = ((res.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

watch(section, load, { immediate: true })
onMounted(load)

function go(key: string) {
  router.push(`/sales/hub/${key}`)
}
</script>

<template>
  <div class="sales-hub" v-loading="loading">
    <h2>销售管理</h2>
    <p class="sub">扩展域能力（product.profile=extended）。成品出库扣账后可进入出厂结算与财务认款。</p>
    <nav class="tabs">
      <button
        v-for="s in SECTIONS"
        :key="s.key"
        type="button"
        class="tab"
        :class="{ active: section === s.key }"
        @click="go(s.key)"
      >
        {{ s.title }}
      </button>
    </nav>
    <TableOrCards
      :data="list"
      :columns="[
        { prop: 'doc_no', label: '单号', primary: true },
        { prop: 'name', label: '名称' },
        { prop: 'customer_name', label: '客户' },
        { prop: 'status', label: '状态' },
        { prop: 'biz_date', label: '日期' },
        { prop: 'amount', label: '金额' },
      ]"
    />
    <el-empty v-if="!list.length && !loading" description="暂无数据；请先维护客户并创建订单/出库单" />
  </div>
</template>

<style scoped>
.sales-hub { padding: 4px; }
h2 { margin: 0 0 6px; font-size: 18px; }
.sub { color: #64748b; font-size: 13px; margin: 0 0 12px; }
.tabs { display: flex; flex-wrap: wrap; gap: 0; margin-bottom: 12px; border-bottom: 1px solid #e5e7eb; }
.tab {
  border: 0; background: transparent; padding: 10px 14px; cursor: pointer;
  color: #64748b; font-size: 13px; border-bottom: 2px solid transparent;
}
.tab.active { color: #145c38; font-weight: 600; border-bottom-color: #1f7a4d; }
</style>
