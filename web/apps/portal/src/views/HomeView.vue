<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { terminalUrl, productionApi } from '@erp/shared'

const linkCards = [
  {
    key: 'shop' as const,
    badge: '前台 · 客户',
    title: '客户自助',
    desc: '询价提交、自助下单、我的订单与发货进度。账号由后台开户，演示 cust01 / admin123。',
  },
  {
    key: 'admin' as const,
    badge: '后台 · 管理端',
    title: '管理后台',
    desc: '13 域菜单与权限中心：采购入库、生产派工、库存过账、销售出库与 IAM。',
  },
  {
    key: 'boss' as const,
    badge: '前台 · 看板',
    title: '老板驾驶舱',
    desc: '驾驶舱 · 生产看板 · 产线实况 · 经营报表只读。',
  },
]

function cardHref(key: (typeof linkCards)[number]['key']) {
  if (key === 'shop') return '/shop'
  return terminalUrl(key)
}

const flow = ref<string[]>([])
const flowHint = ref('未配置工艺')

onMounted(async () => {
  try {
    const res = await productionApi.plantLinePreview()
    const steps = (res.data as { steps?: Array<{ name: string }>; message?: string })?.steps
    if (Array.isArray(steps) && steps.length) {
      flow.value = steps.map((s) => s.name).filter(Boolean)
      flowHint.value = ''
    } else {
      flow.value = []
      flowHint.value = (res.data as { message?: string })?.message || '未配置工艺'
    }
  } catch {
    flow.value = []
    flowHint.value = '未配置工艺'
  }
})
</script>

<template>
  <div class="page">
    <header class="hero">
      <p class="eyebrow">木薯加工 · 工厂产线</p>
      <h1 class="brand">木薯加工厂 ERP</h1>
      <p class="lead">选择终端登录。员工现场作业仅使用 Flutter App；管理后台与老板驾驶舱为 Web。客户自助走 Portal <code>/shop</code>。</p>
      <div class="factory-conveyor portal-line" aria-label="产线节点">
        <template v-if="flow.length">
          <div
            v-for="(step, i) in flow"
            :key="step"
            class="node"
            :class="{ active: i === Math.min(2, flow.length - 1) || i === flow.length - 1 }"
          >
            <div class="seq">{{ String(i + 1).padStart(2, '0') }}</div>
            <div class="nm">{{ step }}</div>
          </div>
        </template>
        <div v-else class="node">
          <div class="seq">—</div>
          <div class="nm">{{ flowHint }}</div>
        </div>
      </div>
    </header>

    <section class="terminals">
      <h2 class="section-title">选择登录平台</h2>
      <div class="grid">
        <a
          v-for="c in linkCards"
          :key="c.key"
          class="card"
          :href="cardHref(c.key)"
        >
          <span class="badge">{{ c.badge }}</span>
          <h3>{{ c.title }}</h3>
          <p>{{ c.desc }}</p>
          <span class="enter">进入 →</span>
        </a>
        <div class="card card-static">
          <span class="badge">现场 · 员工 App</span>
          <h3>Flutter 员工端</h3>
          <p>车间 / 工人 / 仓管 / 销售。登录 <code>client_type=mobile</code>，按 IAM 显隐模块。请用 Android/iOS 安装包；说明见仓库 <code>mobile/README.md</code>。</p>
          <span class="enter muted">无 Web 入口</span>
        </div>
      </div>
    </section>

    <footer class="foot">
      管理端 <code>admin</code> / <code>admin123</code>
      · 客户自助 <code>cust01</code> / <code>admin123</code>
      · API <code>127.0.0.1:18080</code>
    </footer>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
  color: var(--text, #1a2e24);
  background:
    radial-gradient(ellipse 80% 50% at 20% -10%, rgba(31, 122, 77, 0.16), transparent),
    radial-gradient(ellipse 60% 40% at 90% 10%, rgba(166, 124, 61, 0.1), transparent),
    linear-gradient(180deg, #f2f8f4 0%, #e6efe9 100%);
  font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
}
.hero {
  padding: 48px 24px 28px;
  max-width: 960px;
  margin: 0 auto;
}
.eyebrow {
  margin: 0 0 8px;
  font-size: 13px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent, #1f7a4d);
  font-weight: 600;
}
.brand {
  margin: 0 0 12px;
  font-size: clamp(28px, 5vw, 40px);
  font-weight: 700;
  letter-spacing: -0.02em;
}
.lead {
  margin: 0 0 20px;
  max-width: 52ch;
  color: var(--muted, #5a6e64);
  line-height: 1.55;
  font-size: 15px;
}
.lead code {
  font-size: 13px;
  background: rgba(20, 53, 42, 0.06);
  padding: 1px 5px;
  border-radius: 3px;
}
.portal-line {
  margin-top: 4px;
}
.portal-line .node {
  background: #fff;
}
.terminals {
  max-width: 960px;
  margin: 0 auto;
  padding: 8px 24px 48px;
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--muted, #5a6e64);
  margin: 0 0 14px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 14px;
}
.card {
  display: block;
  text-decoration: none;
  color: inherit;
  background: #fff;
  border-radius: 14px;
  padding: 18px 18px 16px;
  border: 1px solid rgba(20, 53, 42, 0.08);
  box-shadow: 0 8px 24px rgba(20, 53, 42, 0.06);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.card:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px rgba(20, 53, 42, 0.1);
}
.card-static {
  cursor: default;
}
.card-static:hover {
  transform: none;
  box-shadow: 0 8px 24px rgba(20, 53, 42, 0.06);
}
.badge {
  display: inline-block;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--accent-soft, #e8f5ee);
  color: var(--accent-dark, #145c38);
  margin-bottom: 10px;
}
.card h3 {
  margin: 0 0 8px;
  font-size: 17px;
}
.card p {
  margin: 0 0 14px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--muted, #5a6e64);
  min-height: 3.9em;
}
.card code {
  font-size: 11px;
  background: rgba(20, 53, 42, 0.05);
  padding: 1px 4px;
  border-radius: 3px;
}
.enter {
  font-size: 13px;
  font-weight: 600;
  color: var(--accent, #1f7a4d);
}
.enter.muted {
  color: #7a8a94;
  font-weight: 500;
}
.foot {
  text-align: center;
  padding: 0 24px 32px;
  font-size: 12px;
  color: #7a8a94;
}
.foot code {
  background: rgba(20, 53, 42, 0.05);
  padding: 1px 5px;
  border-radius: 3px;
}
</style>
