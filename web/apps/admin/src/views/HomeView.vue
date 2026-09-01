<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  useAuthStore,
  usePermStore,
  approvalApi,
  reportApi,
  adminSpecialPath,
} from '@erp/shared'

type Row = Record<string, unknown>
type Kpi = { key: string; title: string; value: unknown; unit?: string }

const auth = useAuthStore()
const perm = usePermStore()
const router = useRouter()

const loading = ref(false)
const asOf = ref('')
const bossKpis = ref<Kpi[]>([])
const productionKpis = ref<Kpi[]>([])
const dailyKpis = ref<Kpi[]>([])
const pendingApproval = ref(0)

const greeting = computed(() => {
  const name = auth.user?.name || auth.user?.login_name || '管理员'
  const h = new Date().getHours()
  const slot = h < 12 ? '上午好' : h < 18 ? '下午好' : '晚上好'
  return `${slot}，${name}`
})

function fmt(v: unknown) {
  if (v == null || v === '') return '—'
  if (typeof v === 'number') return Number.isInteger(v) ? String(v) : v.toFixed(2)
  return String(v)
}

function toKpis(list: Row[] | undefined): Kpi[] {
  return (list || []).map((k) => ({
    key: String(k.key || k.metric || k.title || ''),
    title: String(k.title || k.item || k.metric || k.key || ''),
    value: k.value ?? k.amount,
    unit: k.unit ? String(k.unit) : undefined,
  }))
}

function toKpisFromSummary(s: Row, labels: Record<string, string>): Kpi[] {
  return Object.entries(labels).map(([key, title]) => ({
    key,
    title,
    value: s[key],
    unit: key.includes('amount') || key.includes('balance') || key.includes('cash') ? '元' : undefined,
  }))
}

async function loadDashboard() {
  loading.value = true
  try {
    const today = new Date().toISOString().slice(0, 10)
    const [bossRes, prodRes, dailyRes, apprRes] = await Promise.allSettled([
      reportApi.boss(),
      reportApi.production(),
      reportApi.daily(today),
      approvalApi.tasks(),
    ])

    if (bossRes.status === 'fulfilled' && bossRes.value.code === 1) {
      const data = (bossRes.value.data as Row) || {}
      bossKpis.value = toKpis((data.kpis as Row[]) || (data.list as Row[]))
      asOf.value = String(data.as_of || '')
    } else {
      bossKpis.value = []
    }

    if (prodRes.status === 'fulfilled' && prodRes.value.code === 1) {
      const data = (prodRes.value.data as Row) || {}
      productionKpis.value = toKpis(data.list as Row[])
      if (!asOf.value) asOf.value = String(data.as_of || '')
    } else {
      productionKpis.value = []
    }

    if (dailyRes.status === 'fulfilled' && dailyRes.value.code === 1) {
      const data = (dailyRes.value.data as Row) || {}
      const s = (data.summary as Row) || ((data.list as Row[]) || [])[0] || {}
      dailyKpis.value = toKpisFromSummary(s, {
        sales_amount: '今日销售额',
        sales_orders: '今日销售单',
        report_works: '今日报工单',
        report_qty: '今日报工量',
        stock_in: '今日入库',
        stock_out: '今日出库',
      })
    } else {
      dailyKpis.value = []
    }

    if (apprRes.status === 'fulfilled' && apprRes.value.code === 1) {
      pendingApproval.value = ((apprRes.value.data as { list?: unknown[] })?.list || []).length
    } else {
      pendingApproval.value = 0
    }
  } finally {
    loading.value = false
  }
}

function open(domain: string, module: string) {
  const special = adminSpecialPath(domain, module)
  if (special) {
    router.push(special)
    return
  }
  router.push(`/m/${encodeURIComponent(domain)}/${encodeURIComponent(module)}`)
}

function goReport(section: string) {
  router.push(`/report/hub/${section}`)
}

onMounted(loadDashboard)
</script>

<template>
  <div v-loading="loading" class="dashboard">
    <header class="dash-head">
      <div>
        <h2 class="title">仪表盘总览</h2>
        <p class="greeting">{{ greeting }}</p>
        <p v-if="asOf" class="as-of">数据截止：{{ asOf }}</p>
      </div>
      <div class="head-actions">
        <el-button @click="goReport('boss')">老板驾驶舱</el-button>
        <el-button @click="goReport('production-board')">生产看板</el-button>
        <el-button :loading="loading" @click="loadDashboard">刷新</el-button>
      </div>
    </header>

    <section class="section">
      <h3 class="section-title">经营指标</h3>
      <el-row v-if="bossKpis.length" :gutter="12">
        <el-col v-for="k in bossKpis" :key="k.key" :xs="12" :sm="8" :md="6" :lg="3">
          <el-card shadow="never" class="kpi-card">
            <div class="kpi-label">{{ k.title }}</div>
            <div class="kpi-value">
              {{ fmt(k.value) }}<span v-if="k.unit" class="kpi-unit">{{ k.unit }}</span>
            </div>
          </el-card>
        </el-col>
      </el-row>
      <el-empty v-else description="暂无经营指标（请确认报表权限或演示数据）" :image-size="64" />
    </section>

    <section class="section">
      <h3 class="section-title">生产运行</h3>
      <el-row v-if="productionKpis.length" :gutter="12">
        <el-col v-for="k in productionKpis" :key="k.key" :xs="12" :sm="8" :md="6" :lg="4">
          <el-card shadow="never" class="kpi-card kpi-card-prod">
            <div class="kpi-label">{{ k.title }}</div>
            <div class="kpi-value">{{ fmt(k.value) }}</div>
          </el-card>
        </el-col>
      </el-row>
      <el-empty v-else description="暂无生产指标" :image-size="64" />
    </section>

    <section v-if="dailyKpis.length" class="section">
      <h3 class="section-title">今日业务</h3>
      <el-row :gutter="12">
        <el-col v-for="k in dailyKpis" :key="k.key" :xs="12" :sm="8" :md="6" :lg="4">
          <el-card shadow="never" class="kpi-card kpi-card-daily">
            <div class="kpi-label">{{ k.title }}</div>
            <div class="kpi-value">
              {{ fmt(k.value) }}<span v-if="k.unit" class="kpi-unit">{{ k.unit }}</span>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </section>

    <el-row :gutter="12" class="section">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>待办与快捷入口</template>
          <div class="quick-grid">
            <div class="quick-item click" @click="open('审批中心', '审批任务')">
              <span class="quick-label">审批待办</span>
              <span class="quick-value">{{ pendingApproval }}</span>
            </div>
            <div class="quick-item click" @click="goReport('daily')">
              <span class="quick-label">日统计报表</span>
              <span class="quick-value link">查看</span>
            </div>
            <div class="quick-item click" @click="goReport('live')">
              <span class="quick-label">生产实况</span>
              <span class="quick-value link">查看</span>
            </div>
            <div class="quick-item click" @click="open('系统管理', '业务闭环')">
              <span class="quick-label">业务闭环演示</span>
              <span class="quick-value link">进入</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>权限中心</template>
          <div class="quick-actions">
            <el-button type="primary" @click="open('人事管理', '角色管理')">角色管理</el-button>
            <el-button @click="open('系统管理', '自定义权限')">自定义权限</el-button>
            <el-button @click="open('系统管理', '自定义菜单')">自定义菜单</el-button>
            <el-button @click="open('系统管理', '登录控制')">登录控制</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <section class="section">
      <h3 class="section-title">功能模块</h3>
      <div class="domain-grid">
        <div
          v-for="d in perm.visibleMenus"
          :key="d.domain"
          class="domain-card click"
          @click="open(d.domain, d.modules[0])"
        >
          <div class="domain-name">{{ d.domain }}</div>
          <div class="domain-meta">{{ d.modules.length }} 个模块</div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 4px 2px 24px;
}
.dash-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.title {
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 600;
  color: #1a2b34;
}
.greeting {
  margin: 0;
  font-size: 14px;
  color: #5c6b75;
}
.as-of {
  margin: 4px 0 0;
  font-size: 12px;
  color: #8a9aa3;
}
.head-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.section {
  margin-bottom: 20px;
}
.section-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 600;
  color: #334;
}
.kpi-card {
  margin-bottom: 12px;
  border: 1px solid #e8eef2;
  background: linear-gradient(180deg, #fff 0%, #fafcfd 100%);
}
.kpi-card-prod {
  border-color: #d4e8f7;
}
.kpi-card-daily {
  border-color: #e8f5e9;
}
.kpi-label {
  color: #8a9aa3;
  font-size: 12px;
}
.kpi-value {
  margin-top: 8px;
  font-size: 24px;
  font-weight: 600;
  color: #0d7a6f;
  line-height: 1.2;
}
.kpi-unit {
  margin-left: 4px;
  font-size: 13px;
  font-weight: 400;
  color: #8a9aa3;
}
.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}
.quick-item {
  padding: 12px 14px;
  border: 1px solid #e8eef2;
  border-radius: 8px;
  background: #fafcfd;
}
.quick-item.click {
  cursor: pointer;
}
.quick-item.click:hover {
  border-color: #0d7a6f;
}
.quick-label {
  display: block;
  font-size: 12px;
  color: #8a9aa3;
}
.quick-value {
  display: block;
  margin-top: 4px;
  font-size: 20px;
  font-weight: 600;
  color: #0d7a6f;
}
.quick-value.link {
  font-size: 14px;
}
.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.domain-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 10px;
}
.domain-card {
  padding: 12px 14px;
  border: 1px solid #d5dde3;
  border-radius: 8px;
  background: #fff;
}
.domain-card.click {
  cursor: pointer;
}
.domain-card.click:hover {
  border-color: #0d7a6f;
}
.domain-name {
  font-weight: 600;
  font-size: 14px;
  color: #334;
}
.domain-meta {
  margin-top: 4px;
  font-size: 12px;
  color: #8a9aa3;
}
@media (max-width: 768px) {
  .kpi-value {
    font-size: 20px;
  }
  .quick-grid {
    grid-template-columns: 1fr;
  }
}
</style>
