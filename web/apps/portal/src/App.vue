<script setup lang="ts">
import { terminalUrl } from '@erp/shared'

const cards = [
  {
    key: 'admin' as const,
    badge: '后台 · 管理端',
    title: '管理后台',
    desc: '13 域菜单与权限中心：采购入库、生产派工、库存过账、销售出库与 IAM。',
  },
  {
    key: 'employee' as const,
    badge: '前台 · 员工端',
    title: '统一员工端',
    desc: '车间 / 工人 / 销售同一站点；登录后按 IAM 权限显示可用模块。另有 Android/iOS Flutter App。',
  },
  {
    key: 'boss' as const,
    badge: '前台 · 看板',
    title: '老板驾驶舱',
    desc: '驾驶舱 · 生产看板 · 产线实况 · 经营报表只读。',
  },
]

const flow = ['鲜木薯入库', '清洗', '去皮计件', '收货卡点', '切断去芯', '切块装袋', '成品出库']
</script>

<template>
  <div class="page">
    <header class="hero">
      <p class="eyebrow">木薯加工 · 计件产线</p>
      <h1 class="brand">木薯加工厂 ERP</h1>
      <p class="lead">选择终端登录。员工端为统一 Web（+ Flutter App）；管理与老板驾驶舱独立。客户自助仅保留 API，无前端。</p>
      <ol class="flow" aria-label="产线节点">
        <li v-for="(step, i) in flow" :key="step" :class="{ accent: i === 2 || i === 3 }">{{ step }}</li>
      </ol>
    </header>

    <section class="terminals">
      <h2 class="section-title">选择登录平台</h2>
      <div class="grid">
        <a
          v-for="c in cards"
          :key="c.key"
          class="card"
          :href="terminalUrl(c.key)"
        >
          <span class="badge">{{ c.badge }}</span>
          <h3>{{ c.title }}</h3>
          <p>{{ c.desc }}</p>
          <span class="enter">进入登录 →</span>
        </a>
      </div>
    </section>

    <footer class="foot">
      演示账号 <code>admin</code> / <code>admin123</code>
      · API <code>127.0.0.1:18080</code>
      · 开发请先启动目标终端再点选
    </footer>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
  color: #1a2b34;
  background:
    radial-gradient(ellipse 80% 50% at 20% -10%, rgba(13, 122, 111, 0.18), transparent),
    radial-gradient(ellipse 60% 40% at 90% 10%, rgba(47, 107, 69, 0.12), transparent),
    linear-gradient(180deg, #f4f7f8 0%, #e8eef1 100%);
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
  color: #0d7a6f;
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
  color: #44555e;
  line-height: 1.55;
  font-size: 15px;
}
.flow {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.flow li {
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(26, 43, 52, 0.06);
  color: #3a4a52;
}
.flow li.accent {
  background: #0d7a6f;
  color: #fff;
}
.terminals {
  max-width: 960px;
  margin: 0 auto;
  padding: 8px 24px 48px;
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #5c6b75;
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
  border: 1px solid rgba(26, 43, 52, 0.08);
  box-shadow: 0 8px 24px rgba(15, 28, 34, 0.06);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.card:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px rgba(15, 28, 34, 0.1);
}
.badge {
  display: inline-block;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: #e8f5f2;
  color: #0d7a6f;
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
  color: #5c6b75;
  min-height: 3.9em;
}
.enter {
  font-size: 13px;
  font-weight: 600;
  color: #0d7a6f;
}
.foot {
  text-align: center;
  padding: 0 24px 32px;
  font-size: 12px;
  color: #7a8a94;
}
.foot code {
  background: rgba(0, 0, 0, 0.05);
  padding: 1px 5px;
  border-radius: 3px;
}
</style>
